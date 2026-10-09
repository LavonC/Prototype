package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/p2p"
	"github.com/unicompute/unicompute/internal/protocol"
)

// Controller manages the Worker daemon lifecycle.
type Controller struct {
	config    config.WorkerConfig
	p2pCfg    config.P2PConfig
	sigClient *p2p.SignalingClient
	webrtcMgr *p2p.WebRTCManager
	nodeID    string
}

// NewController creates a new Worker daemon.
func NewController(cfg config.WorkerConfig, p2pCfg config.P2PConfig) *Controller {
	id := "w-" + uuid.New().String()[:8]
	
	// Assuming coordinator URL provides /ws endpoint for signaling
	wsURL := cfg.CoordinatorURL
	
	sigClient := p2p.NewSignalingClient(wsURL, cfg.AuthToken, "worker", id)
	webrtcMgr := p2p.NewWebRTCManager(p2pCfg, sigClient)
	
	c := &Controller{
		config:    cfg,
		p2pCfg:    p2pCfg,
		sigClient: sigClient,
		webrtcMgr: webrtcMgr,
		nodeID:    id,
	}

	sigClient.SetMessageHandler(c.handleMessage)
	
	return c
}

// Start begins the worker daemon.
func (c *Controller) Start(ctx context.Context) error {
	log.Info().Str("node_id", c.nodeID).Msg("Starting Worker Daemon")

	if err := c.sigClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to coordinator: %w", err)
	}

	// Register with Coordinator
	c.register()

	// Start Heartbeat
	go c.heartbeatLoop(ctx)

	// Block until context cancelled
	<-ctx.Done()
	return nil
}

func (c *Controller) register() {
	payload := protocol.WorkerRegisterPayload{
		Version: "1.0.0",
		HardwareProfile: protocol.HardwareProfile{
			CPUCores:      int(c.config.MaxCPUs),
			CPUModel:      "Generic x86_64", // Ideally derived via OS queries
			TotalMemoryMB: c.config.MaxMemoryMB,
			TotalDiskMB:   50000,
			OS:            "linux",
			Arch:          "amd64",
		},
		SupportedRuntimes: []string{
			"unicompute/runtime-universal:latest",
			"unicompute/runtime-python:3.11",
			"unicompute/runtime-node:20",
		},
	}

	env := &protocol.Envelope{
		Type:      protocol.MsgWorkerRegister,
		TargetID:  "coordinator",
		Payload:   payload,
	}
	
	c.sigClient.Send(env)
}

func (c *Controller) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := protocol.WorkerHeartbeatPayload{
				WorkerID:      c.nodeID,
				ActiveTasks:   0, // Track actual later
				AvailSlots:    c.config.MaxConcurrentTasks,
				UptimeSeconds: 0,
			}
			env := &protocol.Envelope{
				Type:     protocol.MsgWorkerHeartbeat,
				TargetID: "coordinator",
				Payload:  payload,
			}
			c.sigClient.Send(env)
		}
	}
}

func (c *Controller) handleMessage(env *protocol.Envelope) {
	payloadBytes, _ := json.Marshal(env.Payload)

	switch env.Type {
	case protocol.MsgWorkerRegistered:
		log.Info().Msg("Successfully registered with Cloud Coordinator")
	case protocol.MsgTaskAssign:
		var ticket protocol.TaskTicket
		json.Unmarshal(payloadBytes, &ticket)
		go c.handleTaskAssignment(ticket)
	case protocol.MsgSDPOffer:
		var sdp protocol.SDPPayload
		json.Unmarshal(payloadBytes, &sdp)
		if err := c.webrtcMgr.HandleSDPOffer(env.SenderID, sdp); err != nil {
			log.Error().Err(err).Msg("Failed to handle SDP offer")
		}
	case protocol.MsgSDPAnswer:
		var sdp protocol.SDPPayload
		json.Unmarshal(payloadBytes, &sdp)
		c.webrtcMgr.HandleSDPAnswer(sdp)
	case protocol.MsgICECandidate:
		var ice protocol.ICECandidatePayload
		json.Unmarshal(payloadBytes, &ice)
		c.webrtcMgr.HandleICECandidate(ice)
	default:
		log.Debug().Str("type", string(env.Type)).Msg("Unhandled message")
	}
}

func (c *Controller) handleTaskAssignment(ticket protocol.TaskTicket) {
	log.Info().Str("task_id", ticket.TaskID).Str("requester_id", ticket.RequesterID).Msg("Received task assignment")
	
	// Create WebRTC connection back to Requester (Offerer)
	// Alternatively, the coordinator tells the requester to initiate the offer.
	// For this design, let's assume Worker initiates the offer to open DataChannels
	conn, err := c.webrtcMgr.CreateConnection(ticket.RequesterID, true)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create WebRTC connection to requester")
		c.reportTaskStatus(ticket.TaskID, protocol.MsgTaskFailed, "Failed to connect P2P")
		return
	}

	conn.OnConnected = func() {
		log.Info().Str("task_id", ticket.TaskID).Msg("P2P connected. Awaiting capsule transfer...")
		
		// Setup File Transfer receiver for the Capsule
		ftm := p2p.NewFileTransferManager(conn.ArtifactCh)
		
		// We need to wait for the file to finish receiving (this would be asynchronous callback in production)
		// Simulating for now...
		time.Sleep(2 * time.Second)
		
		// Execute
		c.executeTask(ticket, conn, ftm)
	}
}

func (c *Controller) executeTask(ticket protocol.TaskTicket, conn *p2p.Connection, ftm *p2p.FileTransferManager) {
	// Setup executor
	exec, err := NewExecutor(ticket, os.TempDir())
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize executor")
		return
	}
	defer exec.Cleanup()

	// In real logic, we use the downloaded capsule path from FTM
	capsulePath := filepath.Join(os.TempDir(), "simulated_capsule.tar.gz")
	
	if err := exec.Prepare(capsulePath); err != nil {
		log.Error().Err(err).Msg("Failed to prepare capsule")
	}

	// Create a pipe or buffer to stream stdout/stderr into WebRTC
	// For now, we simulate by ignoring streams and returning exit code 0
	
	// exitCode, err := exec.Run(context.Background(), os.Stdout, os.Stderr)
	exitCode := 0
	
	if exitCode == 0 {
		c.reportTaskStatus(ticket.TaskID, protocol.MsgTaskComplete, "")
	} else {
		c.reportTaskStatus(ticket.TaskID, protocol.MsgTaskFailed, fmt.Sprintf("Exit code %d", exitCode))
	}
}

func (c *Controller) reportTaskStatus(taskID string, msgType protocol.MessageType, errMsg string) {
	var payload interface{}
	if msgType == protocol.MsgTaskComplete {
		payload = protocol.CompletionReceipt{
			TaskID: taskID,
			Status: "COMPLETED",
		}
	} else {
		payload = protocol.ErrorPayload{
			Code:    500,
			Message: errMsg,
		}
	}

	env := &protocol.Envelope{
		Type:     msgType,
		TargetID: "coordinator",
		Payload:  payload,
	}
	c.sigClient.Send(env)
}
