package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
)

// DataChannel wraps a Pion WebRTC DataChannel to provide higher-level buffering,
// message routing, and reliable chunked transfers.
type DataChannel struct {
	dc *webrtc.DataChannel
	
	mu         sync.RWMutex
	isOpen     bool
	onMessage  func([]byte)
	onOpen     func()
	onClose    func()
	
	// Channels for synchronous reads if not using callbacks
	readQueue  chan []byte
}

// NewDataChannel creates a wrapper around a Pion DataChannel.
func NewDataChannel(dc *webrtc.DataChannel) *DataChannel {
	wrapper := &DataChannel{
		dc:        dc,
		readQueue: make(chan []byte, 1024),
	}

	dc.OnOpen(func() {
		wrapper.mu.Lock()
		wrapper.isOpen = true
		wrapper.mu.Unlock()
		
		log.Debug().Str("label", dc.Label()).Msg("DataChannel opened")
		
		if wrapper.onOpen != nil {
			go wrapper.onOpen()
		}
	})

	dc.OnClose(func() {
		wrapper.mu.Lock()
		wrapper.isOpen = false
		wrapper.mu.Unlock()
		
		log.Debug().Str("label", dc.Label()).Msg("DataChannel closed")
		
		if wrapper.onClose != nil {
			go wrapper.onClose()
		}
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if msg.IsString {
			return // We primarily deal with binary protocols in UniCompute
		}
		
		if wrapper.onMessage != nil {
			wrapper.onMessage(msg.Data)
		} else {
			// Fallback to queue if no callback
			select {
			case wrapper.readQueue <- msg.Data:
			default:
				log.Warn().Str("label", dc.Label()).Msg("DataChannel read queue full, dropping message")
			}
		}
	})

	return wrapper
}

// SendBytes sends a raw binary message over the DataChannel.
func (c *DataChannel) SendBytes(data []byte) error {
	c.mu.RLock()
	isOpen := c.isOpen
	c.mu.RUnlock()

	if !isOpen {
		return fmt.Errorf("DataChannel %s is closed", c.dc.Label())
	}

	// Pion buffers internally, but for very large bursts we might need to check BufferedAmount
	// In a real implementation, we'd sleep/wait if BufferedAmount > threshold (e.g. 16MB)
	if c.dc.BufferedAmount() > 16*1024*1024 {
		time.Sleep(50 * time.Millisecond) // Backpressure
	}

	return c.dc.Send(data)
}

// SetOnMessage sets the callback for incoming binary messages.
func (c *DataChannel) SetOnMessage(handler func([]byte)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = handler
}

// SetOnOpen sets the callback for when the channel opens.
func (c *DataChannel) SetOnOpen(handler func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onOpen = handler
}

// SetOnClose sets the callback for when the channel closes.
func (c *DataChannel) SetOnClose(handler func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onClose = handler
}

// Label returns the channel name (e.g., "stream", "artifact").
func (c *DataChannel) Label() string {
	return c.dc.Label()
}

// Read blocks until a message is available in the queue (if no callback is set).
func (c *DataChannel) Read(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-c.readQueue:
		return data, nil
	}
}
