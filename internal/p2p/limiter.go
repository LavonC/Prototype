package p2p

import (
	"context"

	"golang.org/x/time/rate"
)

// DataRateLimiter applies backpressure to fast-producing streams (like infinite loops dumping to stdout)
// to prevent WebRTC SCTP buffer overflows.
type DataRateLimiter struct {
	limiter *rate.Limiter
}

// NewDataRateLimiter creates a limiter with a specific byte-per-second rate.
func NewDataRateLimiter(bytesPerSecond int) *DataRateLimiter {
	// Token bucket where tokens represent bytes.
	// We allow bursts up to 2x the per-second rate.
	return &DataRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(bytesPerSecond), bytesPerSecond*2),
	}
}

// Wait blocks until enough tokens (bytes) are available to send the payload.
// If the stream produces data faster than this, Wait() causes backpressure all the 
// way up to the Sandbox IO pipes.
func (d *DataRateLimiter) Wait(ctx context.Context, payloadSize int) error {
	// If the payload is bigger than the burst size, it will never succeed.
	// We should technically chunk it, but the FileTransferManager/Stream handles chunking.
	// For this limiter, we just clamp the request size to the burst size to allow it through eventually.
	
	burstSize := d.limiter.Burst()
	if payloadSize > burstSize {
		payloadSize = burstSize
	}

	return d.limiter.WaitN(ctx, payloadSize)
}
