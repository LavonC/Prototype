package coordinator

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/protocol"
)

// SignalingHub handles P2P signaling routing between Requesters and Workers.
type SignalingHub struct {
	sessions *SessionManager
}

// NewSignalingHub creates a new SignalingHub.
func NewSignalingHub(sm *SessionManager) *SignalingHub {
	return &SignalingHub{
		sessions: sm,
	}
}

// HandleMessage processes incoming control envelopes that relate to signaling.
func (sh *SignalingHub) HandleMessage(sess *Session, env *protocol.Envelope) {
	// For P2P signaling (SDP and ICE), the coordinator acts purely as a relay
	// between the sender and the target.
	
	if env.TargetID == "" {
		log.Warn().Str("msg_type", string(env.Type)).Str("sender_id", env.SenderID).Msg("Signaling message missing TargetID, dropping")
		return
	}
	
	// Relay the envelope to the target
	success := sh.sessions.SendTo(env.TargetID, env)
	if !success {
		log.Warn().
			Str("msg_type", string(env.Type)).
			Str("target_id", env.TargetID).
			Msg("Failed to relay signaling message, target not found or queue full")
			
		// Optionally, send an error back to the sender
		errMsg := protocol.ErrorPayload{
			Code:    1400, // ErrSignalingFailed
			Message: "Target peer is unreachable",
		}
		
		errEnv := &protocol.Envelope{
			Type:      protocol.MsgError,
			ID:        env.ID + "_err",
			SenderID:  "coordinator",
			TargetID:  env.SenderID,
			Timestamp: time.Now(),
			Payload:   errMsg,
		}
		sh.sessions.SendTo(env.SenderID, errEnv)
	} else {
		log.Debug().
			Str("msg_type", string(env.Type)).
			Str("from", env.SenderID).
			Str("to", env.TargetID).
			Msg("Relayed signaling message")
	}
}
