// Package api defines shared API types, error codes, and request validation
// for UniCompute's public-facing interfaces.
package api

import "time"

// TaskStatus represents user-visible task states.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusQueued    TaskStatus = "queued"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
)

// JobStatus represents the aggregate status of a multi-task job.
type JobStatus string

const (
	JobPending    JobStatus = "pending"
	JobRunning    JobStatus = "running"
	JobCompleted  JobStatus = "completed"
	JobPartial    JobStatus = "partial"    // Some tasks completed, some failed.
	JobFailed     JobStatus = "failed"
	JobCancelled  JobStatus = "cancelled"
)

// TaskSpec defines a task submission request.
type TaskSpec struct {
	// Entrypoint is the command to execute (e.g., "python3 main.py").
	Entrypoint string `json:"entrypoint"`

	// WorkDir is the working directory for execution (relative to /workspace/input).
	WorkDir string `json:"work_dir,omitempty"`

	// Environment variables to set inside the sandbox.
	Environment map[string]string `json:"environment,omitempty"`

	// RequiredCPUs is the minimum CPU cores needed.
	RequiredCPUs float64 `json:"required_cpus,omitempty"`

	// RequiredMemoryMB is the minimum memory in megabytes needed.
	RequiredMemoryMB int `json:"required_memory_mb,omitempty"`

	// RequiredGPU indicates whether a GPU is needed.
	RequiredGPU bool `json:"required_gpu,omitempty"`

	// Timeout is the maximum execution time.
	Timeout time.Duration `json:"timeout,omitempty"`

	// Priority determines scheduling order (higher = more urgent).
	Priority int `json:"priority,omitempty"`
}

// MapSpec defines a Bag-of-Tasks map submission.
type MapSpec struct {
	// Entrypoint is the command template with {{item}} placeholder.
	Entrypoint string `json:"entrypoint"`

	// Items are the values to substitute into the entrypoint template.
	Items []string `json:"items"`

	// Concurrency is the maximum number of parallel tasks.
	Concurrency int `json:"concurrency,omitempty"`

	// OutputDir is the local directory for aggregated results.
	OutputDir string `json:"output_dir,omitempty"`

	// ReduceCommand is an optional local script to run after all tasks complete.
	ReduceCommand string `json:"reduce_command,omitempty"`

	// TaskSpec provides shared configuration for all map tasks.
	TaskSpec TaskSpec `json:"task_spec"`
}

// TaskResult represents the result of a completed task.
type TaskResult struct {
	TaskID            string        `json:"task_id"`
	JobID             string        `json:"job_id"`
	Status            TaskStatus    `json:"status"`
	ExitCode          int           `json:"exit_code"`
	Duration          time.Duration `json:"duration"`
	WorkerID          string        `json:"worker_id"`
	ArtifactPath      string        `json:"artifact_path,omitempty"`
	ArtifactSHA256    string        `json:"artifact_sha256,omitempty"`
	ArtifactSizeBytes int64         `json:"artifact_size_bytes,omitempty"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	StartedAt         time.Time     `json:"started_at"`
	CompletedAt       time.Time     `json:"completed_at"`
}

// JobResult represents the aggregate result of a multi-task job.
type JobResult struct {
	JobID          string       `json:"job_id"`
	Status         JobStatus    `json:"status"`
	TotalTasks     int          `json:"total_tasks"`
	CompletedTasks int          `json:"completed_tasks"`
	FailedTasks    int          `json:"failed_tasks"`
	TaskResults    []TaskResult `json:"task_results"`
	Duration       time.Duration `json:"duration"`
	SubmittedAt    time.Time     `json:"submitted_at"`
	CompletedAt    time.Time     `json:"completed_at,omitempty"`
}

// WorkerInfo provides public worker information.
type WorkerInfo struct {
	WorkerID    string  `json:"worker_id"`
	Online      bool    `json:"online"`
	CPUCores    int     `json:"cpu_cores"`
	MemoryMB    int     `json:"memory_mb"`
	GPUs        int     `json:"gpus"`
	ActiveTasks int     `json:"active_tasks"`
	Credibility float64 `json:"credibility"`
	Uptime      string  `json:"uptime"`
}

// NetworkStats provides global network statistics.
type NetworkStats struct {
	TotalWorkers    int     `json:"total_workers"`
	OnlineWorkers   int     `json:"online_workers"`
	PendingTasks    int     `json:"pending_tasks"`
	RunningTasks    int     `json:"running_tasks"`
	CompletedTasks  int64   `json:"completed_tasks"`
	TotalCPUCores   int     `json:"total_cpu_cores"`
	TotalMemoryGB   float64 `json:"total_memory_gb"`
	AvgCredibility  float64 `json:"avg_credibility"`
}
