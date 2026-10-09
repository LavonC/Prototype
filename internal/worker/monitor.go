package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// ResourceMonitor periodically polls Sandbox stats and enforces quotas.
type ResourceMonitor struct {
	sandbox  Sandbox
	interval time.Duration
}

// NewResourceMonitor creates a monitor for a specific sandbox.
func NewResourceMonitor(s Sandbox) *ResourceMonitor {
	return &ResourceMonitor{
		sandbox:  s,
		interval: 5 * time.Second,
	}
}

// Start begins tracking resource usage in the background.
// Canceling the context stops the monitor.
func (rm *ResourceMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := rm.sandbox.Stats(ctx)
			if err != nil {
				// Sandbox might be exited or failing
				log.Debug().Err(err).Msg("Failed to get sandbox stats")
				continue
			}

			// In a real implementation, if stats exceed the worker's hardware limits
			// or the task's assigned quota, we would forcefully stop the sandbox here.
			// e.g., if stats.MemoryUsageMB > MaxTaskMemory { rm.sandbox.Stop(ctx) }

			_ = stats // Log or emit metric event
		}
	}
}
