package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/protocol"
)

// WebRTCManager abstracts WebRTC PeerConnection lifecycle and DataChannels.
type WebRTCManager struct {
	config      config.P2PConfig
	sigClient   *SignalingClient
	api         *webrtc.API
	
	mu          sync.Mutex
	connections map[string]*Connection
}

// Connection represents an active P2P link between a Requester and a Worker.
type Connection struct {
	SessionID   string
	PeerID      string
	PC          *webrtc.PeerConnection
	
	StreamCh    *DataChannel // Channel A: Real-Time Stream
	ArtifactCh  *DataChannel // Channel B: Structured Artifacts
	ControlCh   *DataChannel // Channel for ACKs/Receipts/Commands
	
	OnConnected func()
	OnClosed    func()
}

// NewWebRTCManager initializes a WebRTC manager with specific configurations.
func NewWebRTCManager(cfg config.P2PConfig, sigClient *SignalingClient) *WebRTCManager {
	settingEngine := webrtc.SettingEngine{}
	
	// Increase datachannel buffer size for artifact transfers
	settingEngine.SetSCTPMaxReceiveBufferSize(uint32(cfg.MaxDataChannelBufferSize))
	
	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))
	
	return &WebRTCManager{
		config:      cfg,
		sigClient:   sigClient,
		api:         api,
		connections: make(map[string]*Connection),
	}
}

// CreateConnection initializes a new PeerConnection.
func (m *WebRTCManager) CreateConnection(peerID string, isOfferer bool) (*Connection, error) {
	sessionID := uuid.New().String()
	
	wcfg := webrtc.Configuration{
		ICEServers: ParseICEServers(m.config),
	}

	pc, err := m.api.NewPeerConnection(wcfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	conn := &Connection{
		SessionID: sessionID,
		PeerID:    peerID,
		PC:        pc,
	}

	m.mu.Lock()
	m.connections[sessionID] = conn
	m.mu.Unlock()

	// ICE Candidate Handling (Trickle ICE)
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		
		payload := ExtractPionCandidate(c)
		if payload == nil {
			return
		}
		payload.SessionID = sessionID
		
		env := &protocol.Envelope{
			Type:     protocol.MsgICECandidate,
			TargetID: peerID,
			Payload:  payload,
		}
		m.sigClient.Send(env)
	})

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Info().Str("state", s.String()).Str("peer_id", peerID).Msg("PeerConnection state changed")
		if s == webrtc.PeerConnectionStateConnected && conn.OnConnected != nil {
			go conn.OnConnected()
		} else if s == webrtc.PeerConnectionStateClosed || s == webrtc.PeerConnectionStateFailed {
			if conn.OnClosed != nil {
				go conn.OnClosed()
			}
			m.RemoveConnection(sessionID)
		}
	})

	if isOfferer {
		// Initialize the standard Dual-Channels
		// Channel A: Ordered, unreliable (for logs - small retransmits ok, but prioritize flow)
		maxRetransmits := uint16(3)
		streamDC, err := pc.CreateDataChannel("stream", &webrtc.DataChannelInit{
			Ordered:        func(b bool) *bool { return &b }(true),
			MaxRetransmits: &maxRetransmits,
		})
		if err != nil {
			return nil, err
		}
		conn.StreamCh = NewDataChannel(streamDC)

		// Channel B: Ordered, reliable (for artifacts)
		artifactDC, err := pc.CreateDataChannel("artifact", &webrtc.DataChannelInit{
			Ordered: func(b bool) *bool { return &b }(true),
		})
		if err != nil {
			return nil, err
		}
		conn.ArtifactCh = NewDataChannel(artifactDC)

		// Create Offer
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			return nil, err
		}

		if err := pc.SetLocalDescription(offer); err != nil {
			return nil, err
		}

		env := &protocol.Envelope{
			Type:     protocol.MsgSDPOffer,
			TargetID: peerID,
			Payload: protocol.SDPPayload{
				SessionID: sessionID,
				SDP:       offer.SDP,
				Type:      offer.Type.String(),
			},
		}
		m.sigClient.Send(env)
	} else {
		// As an answerer, listen for DataChannels
		pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			log.Debug().Str("label", dc.Label()).Msg("New DataChannel received")
			wrapped := NewDataChannel(dc)
			if dc.Label() == "stream" {
				conn.StreamCh = wrapped
			} else if dc.Label() == "artifact" {
				conn.ArtifactCh = wrapped
			}
		})
	}

	return conn, nil
}

// HandleSDPOffer applies a remote offer and responds with an answer.
func (m *WebRTCManager) HandleSDPOffer(peerID string, payload protocol.SDPPayload) error {
	conn, err := m.CreateConnection(peerID, false)
	if err != nil {
		return err
	}
	conn.SessionID = payload.SessionID
	
	m.mu.Lock()
	m.connections[conn.SessionID] = conn
	m.mu.Unlock()

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  payload.SDP,
	}

	if err := conn.PC.SetRemoteDescription(offer); err != nil {
		return err
	}

	answer, err := conn.PC.CreateAnswer(nil)
	if err != nil {
		return err
	}

	if err := conn.PC.SetLocalDescription(answer); err != nil {
		return err
	}

	env := &protocol.Envelope{
		Type:     protocol.MsgSDPAnswer,
		TargetID: peerID,
		Payload: protocol.SDPPayload{
			SessionID: conn.SessionID,
			SDP:       answer.SDP,
			Type:      answer.Type.String(),
		},
	}
	m.sigClient.Send(env)

	return nil
}

// HandleSDPAnswer applies a remote answer to a pending connection.
func (m *WebRTCManager) HandleSDPAnswer(payload protocol.SDPPayload) error {
	m.mu.Lock()
	conn, ok := m.connections[payload.SessionID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("unknown session id: %s", payload.SessionID)
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  payload.SDP,
	}

	return conn.PC.SetRemoteDescription(answer)
}

// HandleICECandidate applies a remote ICE candidate.
func (m *WebRTCManager) HandleICECandidate(payload protocol.ICECandidatePayload) error {
	m.mu.Lock()
	conn, ok := m.connections[payload.SessionID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("unknown session id: %s", payload.SessionID)
	}

	candidate, err := ConvertToPionCandidate(payload)
	if err != nil {
		return err
	}

	return conn.PC.AddICECandidate(candidate)
}

func (m *WebRTCManager) RemoveConnection(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if conn, ok := m.connections[sessionID]; ok {
		if conn.PC != nil {
			conn.PC.Close()
		}
		delete(m.connections, sessionID)
	}
}

// WaitUntilConnected blocks until the PeerConnection enters the Connected state or context expires.
func (c *Connection) WaitUntilConnected(ctx context.Context) error {
	connectedCh := make(chan struct{})
	
	// Fast path check
	if c.PC.ConnectionState() == webrtc.PeerConnectionStateConnected {
		return nil
	}

	origOnConnected := c.OnConnected
	c.OnConnected = func() {
		close(connectedCh)
		if origOnConnected != nil {
			origOnConnected()
		}
	}

	select {
	case <-connectedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for WebRTC connection")
	}
}
