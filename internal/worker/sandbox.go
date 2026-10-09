package worker

import (
	"context"
	"io"
)

// SandboxConfig defines the resource limits and environment for an isolated execution.
type SandboxConfig struct {
	CPUCores   float64
	MemoryMB   int
	NetworkOut bool // Allow outbound network? (usually false for strict tasks)
	Image      string
	Entrypoint string
	Args       []string
	Env        map[string]string
	Mounts     map[string]string // HostPath -> ContainerPath
}

// SandboxStatus represents the current state of a sandbox.
type SandboxStatus string

const (
	StatusCreated SandboxStatus = "CREATED"
	StatusRunning SandboxStatus = "RUNNING"
	StatusExited  SandboxStatus = "EXITED"
	StatusFailed  SandboxStatus = "FAILED"
)

// SandboxStats holds resource usage metrics.
type SandboxStats struct {
	CPUUsagePercent float64
	MemoryUsageMB   float64
}

// Sandbox represents an isolated execution environment (e.g., gVisor).
type Sandbox interface {
	// Create prepares the sandbox but does not start it.
	Create(ctx context.Context, config SandboxConfig) error
	
	// Start begins execution. stdout/stderr are piped to the provided writers.
	Start(ctx context.Context, stdout, stderr io.Writer) error
	
	// Wait blocks until the sandbox exits and returns the exit code.
	Wait(ctx context.Context) (int, error)
	
	// Stop forcefully terminates the sandbox.
	Stop(ctx context.Context) error
	
	// Stats returns current resource usage metrics.
	Stats(ctx context.Context) (*SandboxStats, error)
	
	// Cleanup removes the sandbox and frees resources.
	Cleanup(ctx context.Context) error
}
