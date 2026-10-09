package p2p

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/protocol"
)

// ResilienceManager handles WebRTC reconnections and stream resumption.
type ResilienceManager struct {
	multiplexer *Multiplexer
	sigClient   *SignalingClient
}

// NewResilienceManager creates a new stream resilience manager.
func NewResilienceManager(mux *Multiplexer, sig *SignalingClient) *ResilienceManager {
	return &ResilienceManager{
		multiplexer: mux,
		sigClient:   sig,
	}
}

// HandleDisconnect is triggered when a WebRTC connection drops unexpectedly.
func (r *ResilienceManager) HandleDisconnect(ctx context.Context, peerID string, taskID string, lastKnownSeq uint64) {
	log.Warn().Str("peer_id", peerID).Str("task_id", taskID).Msg("P2P connection dropped. Attempting recovery...")
	
	// Send a RESUME_STREAM control message over the signaling channel 
	// (since WebRTC is dead) so the worker knows to keep the sandbox alive and buffer logs.
	
	payload := protocol.ResumeStreamPayload{
		TaskID:           taskID,
		LastReceivedSeqID: lastKnownSeq,
	}

	env := &protocol.Envelope{
		Type:     protocol.MsgResumeStream,
		TargetID: peerID, // Route via Coordinator to the specific worker
		Payload:  payload,
	}

	// This sends via WebSocket
	r.sigClient.Send(env)
	
	// In a complete implementation, this would also trigger a new SDP Offer 
	// to re-establish the ICE connection.
}

// HandleResumeStream is called on the Worker when the Requester reconnects.
func (r *ResilienceManager) HandleResumeStream(taskID string, lastReceivedSeqID uint64, ringBuf *FrameRingBuffer, dataChan *DataChannel) {
	log.Info().Str("task_id", taskID).Uint64("resume_seq", lastReceivedSeqID).Msg("Re-streaming buffered logs")
	
	// Fetch missed frames from the ring buffer
	missedFrames := ringBuf.ReadFrom(lastReceivedSeqID + 1)
	
	for _, framePayload := range missedFrames {
		// Multiplex byte 0x01 = stdout
		prefixed := append([]byte{0x01}, framePayload...)
		if err := dataChan.SendBytes(prefixed); err != nil {
			log.Error().Err(err).Msg("Failed to send resumed frame")
			return
		}
		
		// Small delay to prevent swamping the newly established channel
		time.Sleep(5 * time.Millisecond)
	}
	
	log.Info().Str("task_id", taskID).Int("frames_recovered", len(missedFrames)).Msg("Stream recovery complete")
}
