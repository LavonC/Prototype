package protocol

import "time"

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusPending     TaskStatus = "PENDING"
	TaskStatusQueued      TaskStatus = "QUEUED"
	TaskStatusAssigned    TaskStatus = "ASSIGNED"
	TaskStatusDownloading TaskStatus = "DOWNLOADING"
	TaskStatusInstalling  TaskStatus = "INSTALLING"
	TaskStatusRunning     TaskStatus = "RUNNING"
	TaskStatusUploading   TaskStatus = "UPLOADING"
	TaskStatusCompleted   TaskStatus = "COMPLETED"
	TaskStatusFailed      TaskStatus = "FAILED"
	TaskStatusCancelled   TaskStatus = "CANCELLED"
	TaskStatusTimeout     TaskStatus = "TIMEOUT"
	TaskStatusReDispatched TaskStatus = "RE_DISPATCHED"
)

// TaskTicket is the internal representation of an assigned task, issued by the coordinator.
type TaskTicket struct {
	// TicketID is the unique identifier for this assignment.
	TicketID string `json:"ticket_id"`

	// TaskID references the original submitted task.
	TaskID string `json:"task_id"`

	// JobID groups this task with other tasks in the same batch.
	JobID string `json:"job_id"`

	// RequesterID identifies who submitted the task.
	RequesterID string `json:"requester_id"`

	// WorkerID identifies the assigned worker.
	WorkerID string `json:"worker_id"`

	// Entrypoint is the command to execute inside the sandbox.
	Entrypoint string `json:"entrypoint"`

	// RuntimeImage is the pre-cached container image to use.
	RuntimeImage string `json:"runtime_image"`

	// Environment variables to inject into the sandbox.
	Environment map[string]string `json:"environment,omitempty"`

	// CapsuleManifest describes the payload to download.
	CapsuleManifest CapsuleManifest `json:"capsule_manifest"`

	// ResourceLimits specifies sandbox resource constraints.
	ResourceLimits ResourceLimits `json:"resource_limits"`

	// SignalingToken is a short-lived token for P2P signaling.
	SignalingToken string `json:"signaling_token"`

	// Timeout is the maximum execution time.
	Timeout time.Duration `json:"timeout"`

	// Priority determines scheduling order (higher = more urgent).
	Priority int `json:"priority"`

	// Attempt tracks the retry count.
	Attempt int `json:"attempt"`

	// MaxAttempts limits total retry attempts.
	MaxAttempts int `json:"max_attempts"`

	// IsSpotCheck indicates this is a Byzantine verification task.
	IsSpotCheck bool `json:"is_spot_check"`

	// ExpectedHash is the known-correct result hash (only set for spot-checks).
	ExpectedHash string `json:"expected_hash,omitempty"`

	// IssuedAt is when the ticket was created.
	IssuedAt time.Time `json:"issued_at"`

	// ExpiresAt is when the ticket becomes invalid.
	ExpiresAt time.Time `json:"expires_at"`
}

// ResourceLimits specifies the hard resource caps for a sandbox.
type ResourceLimits struct {
	CPUs      float64 `json:"cpus"`
	MemoryMB  int     `json:"memory_mb"`
	DiskMB    int     `json:"disk_mb"`
	PidsLimit int     `json:"pids_limit"`
}
