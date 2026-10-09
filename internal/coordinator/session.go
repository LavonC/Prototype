package coordinator

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/protocol"
)

// Session represents an active WebSocket connection to a Worker or Requester.
type Session struct {
	ID        string
	Role      string // "worker" or "requester"
	Conn      *websocket.Conn
	Manager   *SessionManager
	SendQueue chan []byte
	CloseCh   chan struct{}
	Cancel    context.CancelFunc
}

// SessionManager tracks all active WebSocket sessions and handles routing.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	// Callbacks for business logic
	OnMessage func(sess *Session, env *protocol.Envelope)
	OnClose   func(sess *Session)
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// Register adds a session to the manager.
func (sm *SessionManager) Register(sess *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[sess.ID] = sess
	log.Debug().Str("session_id", sess.ID).Str("role", sess.Role).Msg("Session registered")
}

// Unregister removes a session from the manager.
func (sm *SessionManager) Unregister(sess *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _, ok := sm.sessions[sess.ID]; ok {
		delete(sm.sessions, sess.ID)
		close(sess.CloseCh)
		sess.Conn.Close()
		log.Debug().Str("session_id", sess.ID).Msg("Session unregistered")
		if sm.OnClose != nil {
			go sm.OnClose(sess)
		}
	}
}

// Get retrieves a session by ID.
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sess, ok := sm.sessions[id]
	return sess, ok
}

// SendTo routes an envelope to a specific session.
func (sm *SessionManager) SendTo(id string, env *protocol.Envelope) bool {
	sess, ok := sm.Get(id)
	if !ok {
		return false
	}
	data, err := json.Marshal(env)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal envelope for routing")
		return false
	}
	select {
	case sess.SendQueue <- data:
		return true
	default:
		// Queue full
		log.Warn().Str("session_id", id).Msg("Session send queue full, dropping message")
		return false
	}
}

// StartReadPump reads messages from the WebSocket and passes them to the OnMessage callback.
func (sess *Session) StartReadPump() {
	defer sess.Manager.Unregister(sess)

	// Configure connection deadlines
	sess.Conn.SetReadLimit(1024 * 1024) // 1MB limit for control messages
	sess.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	sess.Conn.SetPongHandler(func(string) error {
		sess.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := sess.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("session_id", sess.ID).Msg("Unexpected websocket closure")
			}
			break
		}

		var env protocol.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			log.Warn().Err(err).Str("session_id", sess.ID).Msg("Invalid message envelope received")
			continue
		}

		if sess.Manager.OnMessage != nil {
			sess.Manager.OnMessage(sess, &env)
		}
	}
}

// StartWritePump sends messages from the SendQueue to the WebSocket.
func (sess *Session) StartWritePump() {
	ticker := time.NewTicker(20 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		sess.Conn.Close()
	}()

	for {
		select {
		case data, ok := <-sess.SendQueue:
			if !ok {
				// Channel closed
				sess.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			sess.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := sess.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Error().Err(err).Str("session_id", sess.ID).Msg("Failed to write websocket message")
				return
			}
		case <-ticker.C:
			sess.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := sess.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-sess.CloseCh:
			return
		}
	}
}
