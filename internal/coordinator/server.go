package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/protocol"
	"github.com/unicompute/unicompute/internal/storage"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// In production, check origin properly
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server is the main Cloud Coordinator HTTP/WS server.
type Server struct {
	config     config.CoordinatorConfig
	db         storage.DB
	sessionMgr *SessionManager
	queue      *TaskQueue
	matchmaker *Matchmaker
	heartbeat  *HeartbeatMonitor
	signaling  *SignalingHub
	middleware *Middleware
}

// NewServer initializes the coordinator server.
func NewServer(cfg config.CoordinatorConfig, db storage.DB, mw *Middleware) *Server {
	sm := NewSessionManager()
	q := NewTaskQueue()
	mm := NewMatchmaker(db, q)
	hb := NewHeartbeatMonitor(db, cfg, q, sm)
	sig := NewSignalingHub(sm)

	s := &Server{
		config:     cfg,
		db:         db,
		sessionMgr: sm,
		queue:      q,
		matchmaker: mm,
		heartbeat:  hb,
		signaling:  sig,
		middleware: mw,
	}

	// Register message handler
	sm.OnMessage = s.handleEnvelope

	return s
}

// Start launches the HTTP server and background loops.
func (s *Server) Start(ctx context.Context) error {
	// Load pending tasks into queue
	if err := s.queue.LoadPendingTasks(ctx, s.db); err != nil {
		log.Warn().Err(err).Msg("Failed to load pending tasks, starting with empty queue")
	}

	// Start background workers
	go s.matchmaker.MatchLoop(ctx, s.notifyTaskAssign)
	go s.heartbeat.Start(ctx)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.middleware.RequireAuth(s.handleWebSocket))
	
	// API routes for Requester CLI (Task submission, etc)
	mux.HandleFunc("/api/v1/tasks", s.middleware.RequireAuth(s.handleTaskSubmit))
	
	// Auth routes
	mux.HandleFunc("/api/v1/auth", s.handleAuth)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Info().Str("addr", addr).Msg("Starting Coordinator Server")

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Graceful shutdown handling
	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutting down Coordinator Server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	return server.ListenAndServe()
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	// Simplified auth: exchange static token for JWT
	// In production, this would check against a DB of users/workers
	
	role := r.URL.Query().Get("role")
	if role != "worker" && role != "requester" {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}
	
	id := r.URL.Query().Get("id")
	if id == "" {
		id = uuid.New().String()
	}

	token, err := s.middleware.GenerateToken(id, role)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
		"id":    id,
	})
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}

	ctx := r.Context()
	userID := ctx.Value(ContextKeyUserID).(string)
	role := ctx.Value(ContextKeyRole).(string)

	sessCtx, cancel := context.WithCancel(context.Background())

	sess := &Session{
		ID:        userID,
		Role:      role,
		Conn:      conn,
		Manager:   s.sessionMgr,
		SendQueue: make(chan []byte, 256),
		CloseCh:   make(chan struct{}),
		Cancel:    cancel,
	}

	s.sessionMgr.Register(sess)

	go sess.StartWritePump()
	go sess.StartReadPump()
	
	// Wait for session to end
	<-sessCtx.Done()
}

func (s *Server) handleEnvelope(sess *Session, env *protocol.Envelope) {
	ctx := context.Background()

	switch env.Type {
	case protocol.MsgWorkerRegister:
		s.handleWorkerRegister(ctx, sess, env)
	case protocol.MsgWorkerHeartbeat:
		s.handleWorkerHeartbeat(ctx, sess, env)
	case protocol.MsgSDPOffer, protocol.MsgSDPAnswer, protocol.MsgICECandidate:
		s.signaling.HandleMessage(sess, env)
	case protocol.MsgTaskComplete, protocol.MsgTaskFailed:
		s.handleTaskStatusUpdate(ctx, sess, env)
	default:
		log.Warn().Str("type", string(env.Type)).Msg("Unhandled message type")
	}
}

func (s *Server) handleWorkerRegister(ctx context.Context, sess *Session, env *protocol.Envelope) {
	// Parse payload
	payloadBytes, _ := json.Marshal(env.Payload)
	var reg protocol.WorkerRegisterPayload
	if err := json.Unmarshal(payloadBytes, &reg); err != nil {
		log.Error().Err(err).Msg("Invalid WorkerRegister payload")
		return
	}

	// Update DB
	runtimesStr := ""
	for i, r := range reg.SupportedRuntimes {
		if i > 0 {
			runtimesStr += ","
		}
		runtimesStr += r
	}

	worker := &storage.WorkerRecord{
		WorkerID:          sess.ID,
		Online:            true,
		CPUCores:          reg.HardwareProfile.CPUCores,
		CPUModel:          reg.HardwareProfile.CPUModel,
		TotalMemoryMB:     reg.HardwareProfile.TotalMemoryMB,
		TotalDiskMB:       reg.HardwareProfile.TotalDiskMB,
		GPUCount:          len(reg.HardwareProfile.GPUs),
		OS:                reg.HardwareProfile.OS,
		Arch:              reg.HardwareProfile.Arch,
		SupportedRuntimes: runtimesStr,
		Credibility:       s.config.Byzantine.InitialCredibility, // Defaults from config
		Version:           reg.Version,
		LastHeartbeat:     time.Now(),
		RegisteredAt:      time.Now(),
	}

	if err := s.db.UpsertWorker(ctx, worker); err != nil {
		log.Error().Err(err).Msg("Failed to upsert worker")
		return
	}

	// Reply
	reply := &protocol.Envelope{
		Type:      protocol.MsgWorkerRegistered,
		ID:        uuid.New().String(),
		SenderID:  "coordinator",
		TargetID:  sess.ID,
		Timestamp: time.Now(),
		Payload: protocol.WorkerRegisteredPayload{
			WorkerID:    sess.ID,
			Accepted:    true,
			Credibility: worker.Credibility,
		},
	}
	s.sessionMgr.SendTo(sess.ID, reply)
}

func (s *Server) handleWorkerHeartbeat(ctx context.Context, sess *Session, env *protocol.Envelope) {
	payloadBytes, _ := json.Marshal(env.Payload)
	var hb protocol.WorkerHeartbeatPayload
	if err := json.Unmarshal(payloadBytes, &hb); err != nil {
		return
	}
	s.heartbeat.HandleHeartbeat(ctx, &hb)
}

func (s *Server) notifyTaskAssign(task *storage.TaskRecord) {
	ticket := protocol.TaskTicket{
		TicketID:       uuid.New().String(),
		TaskID:         task.TaskID,
		JobID:          task.JobID,
		RequesterID:    task.RequesterID,
		WorkerID:       task.WorkerID,
		Entrypoint:     task.Entrypoint,
		RuntimeImage:   task.RuntimeImage,
		Priority:       task.Priority,
		Attempt:        task.Attempt,
		MaxAttempts:    task.MaxAttempts,
		IsSpotCheck:    task.IsSpotCheck,
		IssuedAt:       time.Now(),
		// We'd add CapsuleManifest from DB/storage here
	}

	env := &protocol.Envelope{
		Type:      protocol.MsgTaskAssign,
		ID:        uuid.New().String(),
		SenderID:  "coordinator",
		TargetID:  task.WorkerID,
		Timestamp: time.Now(),
		Payload:   ticket,
	}

	s.sessionMgr.SendTo(task.WorkerID, env)
}

func (s *Server) handleTaskSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload protocol.TaskSubmitPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Validate (using api types validation theoretically)
	
	taskID := payload.TaskID
	if taskID == "" {
		taskID = "task-" + uuid.New().String()[:8]
	}

	task := &storage.TaskRecord{
		TaskID:        taskID,
		JobID:         payload.JobID,
		RequesterID:   r.Context().Value(ContextKeyUserID).(string),
		Status:        "PENDING",
		Entrypoint:    payload.Entrypoint,
		RuntimeImage:  payload.RuntimeImage,
		Priority:      payload.Priority,
		CapsuleSHA256: payload.CapsuleManifest.SHA256,
		CreatedAt:     time.Now(),
		MaxAttempts:   3,
	}

	if err := s.db.CreateTask(context.Background(), task); err != nil {
		http.Error(w, "Failed to store task", http.StatusInternalServerError)
		return
	}

	s.queue.Enqueue(task)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"task_id": task.TaskID})
}

func (s *Server) handleTaskStatusUpdate(ctx context.Context, sess *Session, env *protocol.Envelope) {
	// Parse receipt and update DB
	// Handle Byzantine spot-checks logic here
}
