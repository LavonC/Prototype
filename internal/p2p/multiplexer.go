package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

// Multiplexer routes incoming DataChannel messages to the appropriate handler
// based on the message prefix or parsed type.
type Multiplexer struct {
	streamHandler   func(taskID string, isError bool, data []byte)
	artifactHandler func(data []byte)
	controlHandler  func(msg ControlMessage)
}

// ControlMessage represents out-of-band P2P control instructions.
type ControlMessage struct {
	Type    string `json:"type"`     // "RESUME_STREAM", "PAUSE_STREAM", "ACK"
	SeqID   uint64 `json:"seq_id"`
	TaskID  string `json:"task_id"`
}

// NewMultiplexer creates a new data channel multiplexer.
func NewMultiplexer() *Multiplexer {
	return &Multiplexer{}
}

// SetStreamHandler registers the callback for live log streams (Channel A).
func (m *Multiplexer) SetStreamHandler(handler func(taskID string, isError bool, data []byte)) {
	m.streamHandler = handler
}

// SetArtifactHandler registers the callback for large binary chunks (Channel B).
func (m *Multiplexer) SetArtifactHandler(handler func(data []byte)) {
	m.artifactHandler = handler
}

// SetControlHandler registers the callback for P2P control messages.
func (m *Multiplexer) SetControlHandler(handler func(msg ControlMessage)) {
	m.controlHandler = handler
}

// Route processes a raw incoming byte array and dispatches it.
func (m *Multiplexer) Route(channelID string, data []byte) error {
	// Simple routing logic:
	// If channel is artifact channel, send to artifact handler.
	// If channel is stream channel, inspect the first byte for multiplexing.
	// 0x01 = STDOUT stream chunk
	// 0x02 = STDERR stream chunk
	// 0x03 = JSON Control Message

	if len(data) == 0 {
		return nil
	}

	// Assuming the caller specifies which logical channel this arrived on.
	// For architecture simplicity, we route everything here via prefixes if it's the stream channel.
	
	if channelID == "artifact" {
		if m.artifactHandler != nil {
			m.artifactHandler(data)
		}
		return nil
	}

	if channelID == "stream" {
		msgType := data[0]
		payload := data[1:]

		switch msgType {
		case 0x01: // STDOUT
			if m.streamHandler != nil {
				// We don't have taskID embedded in every chunk to save bandwidth.
				// In a real system, the connection is 1:1 with the task, or the taskID is in a header frame.
				// We pass a placeholder here.
				m.streamHandler("task-unknown", false, payload)
			}
		case 0x02: // STDERR
			if m.streamHandler != nil {
				m.streamHandler("task-unknown", true, payload)
			}
		case 0x03: // CONTROL
			if m.controlHandler != nil {
				var ctrl ControlMessage
				if err := json.Unmarshal(payload, &ctrl); err != nil {
					log.Warn().Err(err).Msg("Failed to decode P2P control message")
					return err
				}
				m.controlHandler(ctrl)
			}
		default:
			return fmt.Errorf("unknown stream multiplexer prefix: %d", msgType)
		}
	}

	return nil
}
