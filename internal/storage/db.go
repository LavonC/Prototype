// Package storage provides the database abstraction layer for UniCompute.
// Workers use SQLite for local storage, the coordinator uses PostgreSQL.
package storage

import (
	"context"
	"time"
)

// DB defines the unified database interface used by all UniCompute components.
type DB interface {
	// Close closes the database connection.
	Close() error

	// Migrate runs pending database migrations.
	Migrate(ctx context.Context) error

	// --- Worker Registry ---

	// UpsertWorker creates or updates a worker record.
	UpsertWorker(ctx context.Context, worker *WorkerRecord) error

	// GetWorker retrieves a worker by ID.
	GetWorker(ctx context.Context, workerID string) (*WorkerRecord, error)

	// ListWorkers lists workers matching the given filter.
	ListWorkers(ctx context.Context, filter WorkerFilter) ([]*WorkerRecord, error)

	// UpdateWorkerStatus updates a worker's online status and last heartbeat.
	UpdateWorkerStatus(ctx context.Context, workerID string, online bool, lastHeartbeat time.Time) error

	// DeleteWorker removes a worker record.
	DeleteWorker(ctx context.Context, workerID string) error

	// --- Task Management ---

	// CreateTask stores a new task record.
	CreateTask(ctx context.Context, task *TaskRecord) error

	// GetTask retrieves a task by ID.
	GetTask(ctx context.Context, taskID string) (*TaskRecord, error)

	// UpdateTaskStatus updates a task's status and optional metadata.
	UpdateTaskStatus(ctx context.Context, taskID string, status string, metadata map[string]interface{}) error

	// ListTasks lists tasks matching the given filter.
	ListTasks(ctx context.Context, filter TaskFilter) ([]*TaskRecord, error)

	// GetPendingTasks returns tasks waiting for assignment, ordered by priority.
	GetPendingTasks(ctx context.Context, limit int) ([]*TaskRecord, error)

	// --- Job Management ---

	// CreateJob stores a new job record.
	CreateJob(ctx context.Context, job *JobRecord) error

	// GetJob retrieves a job by ID.
	GetJob(ctx context.Context, jobID string) (*JobRecord, error)

	// UpdateJobStatus updates a job's status.
	UpdateJobStatus(ctx context.Context, jobID string, status string) error

	// --- Credibility ---

	// GetCredibility retrieves a worker's credibility score.
	GetCredibility(ctx context.Context, workerID string) (float64, error)

	// UpdateCredibility sets a worker's credibility score.
	UpdateCredibility(ctx context.Context, workerID string, score float64) error

	// RecordVerification logs a verification event (correct/incorrect result).
	RecordVerification(ctx context.Context, record *VerificationRecord) error

	// GetRecentTasksByWorker retrieves tasks completed by a worker within a time window.
	GetRecentTasksByWorker(ctx context.Context, workerID string, since time.Time) ([]*TaskRecord, error)
}

// WorkerRecord represents a worker in the database.
type WorkerRecord struct {
	WorkerID          string    `json:"worker_id" db:"worker_id"`
	Online            bool      `json:"online" db:"online"`
	CPUCores          int       `json:"cpu_cores" db:"cpu_cores"`
	CPUModel          string    `json:"cpu_model" db:"cpu_model"`
	TotalMemoryMB     int       `json:"total_memory_mb" db:"total_memory_mb"`
	TotalDiskMB       int       `json:"total_disk_mb" db:"total_disk_mb"`
	GPUCount          int       `json:"gpu_count" db:"gpu_count"`
	OS                string    `json:"os" db:"os"`
	Arch              string    `json:"arch" db:"arch"`
	SupportedRuntimes string    `json:"supported_runtimes" db:"supported_runtimes"` // Comma-separated
	Credibility       float64   `json:"credibility" db:"credibility"`
	Version           string    `json:"version" db:"version"`
	LastHeartbeat     time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	RegisteredAt      time.Time `json:"registered_at" db:"registered_at"`
	TotalTasksRun     int64     `json:"total_tasks_run" db:"total_tasks_run"`
	TotalCPUSeconds   float64   `json:"total_cpu_seconds" db:"total_cpu_seconds"`
}

// TaskRecord represents a task in the database.
type TaskRecord struct {
	TaskID          string    `json:"task_id" db:"task_id"`
	JobID           string    `json:"job_id" db:"job_id"`
	RequesterID     string    `json:"requester_id" db:"requester_id"`
	WorkerID        string    `json:"worker_id" db:"worker_id"`
	Status          string    `json:"status" db:"status"`
	Entrypoint      string    `json:"entrypoint" db:"entrypoint"`
	RuntimeImage    string    `json:"runtime_image" db:"runtime_image"`
	ExitCode        int       `json:"exit_code" db:"exit_code"`
	ErrorMessage    string    `json:"error_message" db:"error_message"`
	Priority        int       `json:"priority" db:"priority"`
	Attempt         int       `json:"attempt" db:"attempt"`
	MaxAttempts     int       `json:"max_attempts" db:"max_attempts"`
	CapsuleSHA256   string    `json:"capsule_sha256" db:"capsule_sha256"`
	ResultSHA256    string    `json:"result_sha256" db:"result_sha256"`
	ArtifactPath    string    `json:"artifact_path" db:"artifact_path"`
	ArtifactSize    int64     `json:"artifact_size" db:"artifact_size"`
	DurationMs      int64     `json:"duration_ms" db:"duration_ms"`
	CPUUserSeconds  float64   `json:"cpu_user_seconds" db:"cpu_user_seconds"`
	PeakMemoryMB    float64   `json:"peak_memory_mb" db:"peak_memory_mb"`
	IsSpotCheck     bool      `json:"is_spot_check" db:"is_spot_check"`
	ExpectedHash    string    `json:"expected_hash" db:"expected_hash"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	StartedAt       time.Time `json:"started_at" db:"started_at"`
	CompletedAt     time.Time `json:"completed_at" db:"completed_at"`
}

// JobRecord represents a batch job in the database.
type JobRecord struct {
	JobID          string    `json:"job_id" db:"job_id"`
	RequesterID    string    `json:"requester_id" db:"requester_id"`
	Status         string    `json:"status" db:"status"`
	TotalTasks     int       `json:"total_tasks" db:"total_tasks"`
	CompletedTasks int       `json:"completed_tasks" db:"completed_tasks"`
	FailedTasks    int       `json:"failed_tasks" db:"failed_tasks"`
	Entrypoint     string    `json:"entrypoint" db:"entrypoint"`
	ReduceCommand  string    `json:"reduce_command" db:"reduce_command"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	CompletedAt    time.Time `json:"completed_at" db:"completed_at"`
}

// VerificationRecord logs a Byzantine verification event.
type VerificationRecord struct {
	ID             string    `json:"id" db:"id"`
	TaskID         string    `json:"task_id" db:"task_id"`
	WorkerID       string    `json:"worker_id" db:"worker_id"`
	ExpectedHash   string    `json:"expected_hash" db:"expected_hash"`
	ActualHash     string    `json:"actual_hash" db:"actual_hash"`
	Passed         bool      `json:"passed" db:"passed"`
	IsSpotCheck    bool      `json:"is_spot_check" db:"is_spot_check"`
	OldCredibility float64   `json:"old_credibility" db:"old_credibility"`
	NewCredibility float64   `json:"new_credibility" db:"new_credibility"`
	VerifiedAt     time.Time `json:"verified_at" db:"verified_at"`
}

// WorkerFilter defines criteria for listing workers.
type WorkerFilter struct {
	OnlineOnly    bool
	MinCPUCores   int
	MinMemoryMB   int
	RequiresGPU   bool
	RuntimeImage  string
	MinCredibility float64
	Limit         int
	Offset        int
}

// TaskFilter defines criteria for listing tasks.
type TaskFilter struct {
	JobID       string
	WorkerID    string
	RequesterID string
	Status      string
	Limit       int
	Offset      int
}
