// Package protocol defines wire protocol formats, control messages, task tickets,
// and completion receipts for all UniCompute communication channels.
package protocol

import (
	"encoding/binary"
	"fmt"
	"time"
)

// StreamID identifies the type of stream in Channel A frames.
type StreamID byte

const (
	// StreamStdout identifies stdout stream data.
	StreamStdout StreamID = 0x01

	// StreamStderr identifies stderr stream data.
	StreamStderr StreamID = 0x02

	// StreamControl identifies control/metadata frames.
	StreamControl StreamID = 0x03
)

// FrameHeaderSize is the size of the Channel A stream frame header in bytes.
// Layout: [StreamID(1)] [Timestamp(8)] [PayloadLength(4)] = 13 bytes
const FrameHeaderSize = 13

// StreamFrame represents a single framed chunk of Channel A streaming data.
type StreamFrame struct {
	// StreamID identifies the stream type (stdout=0x01, stderr=0x02, control=0x03).
	StreamID StreamID

	// Timestamp is the Unix timestamp in milliseconds when the frame was created.
	Timestamp uint64

	// SequenceID is a monotonically increasing sequence number for ordering and resumption.
	SequenceID uint64

	// Payload is the raw UTF-8 text or ANSI byte content.
	Payload []byte
}

// MarshalBinary encodes a StreamFrame into the wire format.
//
// Wire layout (13-byte header + payload):
//
//	+---------------+-------------------------+----------------------+--------------------------+
//	| Byte 0        | Bytes 1-8               | Bytes 9-12           | Bytes 13+                |
//	| Stream ID     | Unix Timestamp (ms)     | Payload Length (N)   | Raw Text / ANSI Bytes    |
//	+---------------+-------------------------+----------------------+--------------------------+
func (f *StreamFrame) MarshalBinary() ([]byte, error) {
	payloadLen := len(f.Payload)
	if payloadLen > MaxPayloadSize {
		return nil, fmt.Errorf("payload size %d exceeds maximum %d", payloadLen, MaxPayloadSize)
	}

	buf := make([]byte, FrameHeaderSize+payloadLen)

	// Byte 0: Stream ID
	buf[0] = byte(f.StreamID)

	// Bytes 1-8: Timestamp (64-bit big-endian)
	binary.BigEndian.PutUint64(buf[1:9], f.Timestamp)

	// Bytes 9-12: Payload length (32-bit big-endian)
	binary.BigEndian.PutUint32(buf[9:13], uint32(payloadLen))

	// Bytes 13+: Payload
	copy(buf[13:], f.Payload)

	return buf, nil
}

// UnmarshalBinary decodes a StreamFrame from the wire format.
func (f *StreamFrame) UnmarshalBinary(data []byte) error {
	if len(data) < FrameHeaderSize {
		return fmt.Errorf("frame too short: %d bytes, need at least %d", len(data), FrameHeaderSize)
	}

	f.StreamID = StreamID(data[0])
	f.Timestamp = binary.BigEndian.Uint64(data[1:9])

	payloadLen := binary.BigEndian.Uint32(data[9:13])
	if int(payloadLen) > len(data)-FrameHeaderSize {
		return fmt.Errorf("payload length %d exceeds available data %d", payloadLen, len(data)-FrameHeaderSize)
	}

	f.Payload = make([]byte, payloadLen)
	copy(f.Payload, data[FrameHeaderSize:FrameHeaderSize+int(payloadLen)])

	return nil
}

// MaxPayloadSize is the maximum size of a single frame payload (1 MB).
const MaxPayloadSize = 1024 * 1024

// NewStreamFrame creates a new StreamFrame with the current timestamp.
func NewStreamFrame(streamID StreamID, seqID uint64, payload []byte) *StreamFrame {
	return &StreamFrame{
		StreamID:   streamID,
		Timestamp:  uint64(time.Now().UnixMilli()),
		SequenceID: seqID,
		Payload:    payload,
	}
}
