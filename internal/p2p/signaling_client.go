package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/protocol"
)

// SignalingClient manages the WebSocket connection to the Cloud Coordinator for WebRTC signaling.
type SignalingClient struct {
	conn       *websocket.Conn
	url        string
	token      string
	role       string
	nodeID     string
	sendQueue  chan *protocol.Envelope
	msgHandler func(*protocol.Envelope)
}

// NewSignalingClient initializes a client for Coordinator WebSocket signaling.
func NewSignalingClient(url, token, role, nodeID string) *SignalingClient {
	if nodeID == "" {
		nodeID = uuid.New().String()
	}
	return &SignalingClient{
		url:       url,
		token:     token,
		role:      role,
		nodeID:    nodeID,
		sendQueue: make(chan *protocol.Envelope, 100),
	}
}

// Connect establishes the WebSocket connection and authenticates.
func (c *SignalingClient) Connect(ctx context.Context) error {
	// 1. Exchange token for JWT if needed (simplified here to assume token is JWT)
	// Alternatively, call /api/v1/auth if using static tokens
	
	dialer := websocket.DefaultDialer
	headers := map[string][]string{
		"Authorization": {"Bearer " + c.token},
	}
	
	conn, _, err := dialer.DialContext(ctx, c.url+"/ws", headers)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	c.conn = conn
	log.Info().Str("role", c.role).Str("node_id", c.nodeID).Msg("Connected to Signaling Hub")

	go c.writePump(ctx)
	go c.readPump(ctx)

	return nil
}

// SetMessageHandler sets the callback for incoming signaling messages.
func (c *SignalingClient) SetMessageHandler(handler func(*protocol.Envelope)) {
	c.msgHandler = handler
}

// Send routes a control envelope through the Coordinator to the target peer.
func (c *SignalingClient) Send(env *protocol.Envelope) {
	// Ensure metadata is filled
	if env.SenderID == "" {
		env.SenderID = c.nodeID
	}
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	if env.Timestamp.IsZero() {
		env.Timestamp = time.Now()
	}
	
	c.sendQueue <- env
}

func (c *SignalingClient) writePump(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	defer c.conn.Close()

	for {
		select {
		case <-ctx.Done():
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case env, ok := <-c.sendQueue:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(env); err != nil {
				log.Error().Err(err).Msg("Failed to write JSON envelope")
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *SignalingClient) readPump(ctx context.Context) {
	defer c.conn.Close()

	c.conn.SetReadLimit(1024 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Msg("Unexpected websocket close")
			}
			break
		}

		var env protocol.Envelope
		if err := json.Unmarshal(message, &env); err != nil {
			log.Warn().Err(err).Msg("Invalid JSON envelope received")
			continue
		}

		if c.msgHandler != nil {
			c.msgHandler(&env)
		}
	}
}
