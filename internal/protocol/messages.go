// Package protocol defines control messages exchanged between UniCompute components
// over WebSocket signaling and WebRTC DataChannels.
package protocol

import "time"

// MessageType identifies the type of control message.
type MessageType string

const (
	// Worker registration & lifecycle.
	MsgWorkerRegister    MessageType = "WORKER_REGISTER"
	MsgWorkerRegistered  MessageType = "WORKER_REGISTERED"
	MsgWorkerHeartbeat   MessageType = "WORKER_HEARTBEAT"
	MsgWorkerUnreachable MessageType = "WORKER_UNREACHABLE"
	MsgWorkerDeregister  MessageType = "WORKER_DEREGISTER"

	// Task lifecycle.
	MsgTaskSubmit   MessageType = "TASK_SUBMIT"
	MsgTaskAssign   MessageType = "TASK_ASSIGN"
	MsgTaskAccepted MessageType = "TASK_ACCEPTED"
	MsgTaskProgress MessageType = "TASK_PROGRESS"
	MsgTaskComplete MessageType = "TASK_COMPLETE"
	MsgTaskFailed   MessageType = "TASK_FAILED"
	MsgTaskCancel   MessageType = "TASK_CANCEL"

	// WebRTC signaling.
	MsgSDPOffer     MessageType = "SDP_OFFER"
	MsgSDPAnswer    MessageType = "SDP_ANSWER"
	MsgICECandidate MessageType = "ICE_CANDIDATE"

	// Stream control.
	MsgResumeStream MessageType = "RESUME_STREAM"
	MsgPauseStream  MessageType = "PAUSE_STREAM"

	// Byzantine verification.
	MsgSpotCheckTask   MessageType = "SPOT_CHECK_TASK"
	MsgSpotCheckResult MessageType = "SPOT_CHECK_RESULT"

	// System.
	MsgError MessageType = "ERROR"
	MsgPing  MessageType = "PING"
	MsgPong  MessageType = "PONG"
)

// Envelope wraps every control message with routing metadata.
type Envelope struct {
	// Type identifies the message type.
	Type MessageType `json:"type"`

	// ID is a unique message identifier for correlation.
	ID string `json:"id"`

	// SenderID is the ID of the sending node (worker_id or requester_id).
	SenderID string `json:"sender_id"`

	// TargetID is the intended recipient (empty for broadcast/coordinator).
	TargetID string `json:"target_id,omitempty"`

	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`

	// Payload is the type-specific message body.
	Payload interface{} `json:"payload"`
}

// WorkerRegisterPayload is sent by a worker to register with the coordinator.
type WorkerRegisterPayload struct {
	WorkerID          string          `json:"worker_id"`
	HardwareProfile   HardwareProfile `json:"hardware_profile"`
	SupportedRuntimes []string        `json:"supported_runtimes"`
	Version           string          `json:"version"`
	AuthToken         string          `json:"auth_token"`
}

// HardwareProfile describes a worker's hardware capabilities.
type HardwareProfile struct {
	CPUCores       int      `json:"cpu_cores"`
	CPUModel       string   `json:"cpu_model"`
	CPUFreqMHz     float64  `json:"cpu_freq_mhz"`
	TotalMemoryMB  int      `json:"total_memory_mb"`
	AvailMemoryMB  int      `json:"avail_memory_mb"`
	TotalDiskMB    int      `json:"total_disk_mb"`
	AvailDiskMB    int      `json:"avail_disk_mb"`
	GPUs           []GPUInfo `json:"gpus,omitempty"`
	OS             string   `json:"os"`
	Arch           string   `json:"arch"`
	NetworkBandwidth float64 `json:"network_bandwidth_mbps"`
}

// GPUInfo describes a GPU device.
type GPUInfo struct {
	Name     string `json:"name"`
	MemoryMB int    `json:"memory_mb"`
	Driver   string `json:"driver"`
	CUDA     string `json:"cuda_version,omitempty"`
}

// WorkerRegisteredPayload is sent by the coordinator to confirm worker registration.
type WorkerRegisteredPayload struct {
	WorkerID    string `json:"worker_id"`
	Accepted    bool   `json:"accepted"`
	Message     string `json:"message,omitempty"`
	Credibility float64 `json:"credibility"`
}

// WorkerHeartbeatPayload is sent periodically by workers to signal availability.
type WorkerHeartbeatPayload struct {
	WorkerID      string      `json:"worker_id"`
	LoadMetrics   LoadMetrics `json:"load_metrics"`
	ActiveTasks   int         `json:"active_tasks"`
	AvailSlots    int         `json:"avail_slots"`
	UptimeSeconds float64    `json:"uptime_seconds"`
}

// LoadMetrics reports current resource utilization on a worker.
type LoadMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	NetworkRxBps  float64 `json:"network_rx_bps"`
	NetworkTxBps  float64 `json:"network_tx_bps"`
}

// TaskSubmitPayload is sent by a requester to submit a new task.
type TaskSubmitPayload struct {
	JobID           string   `json:"job_id"`
	TaskID          string   `json:"task_id"`
	Entrypoint      string   `json:"entrypoint"`
	RuntimeImage    string   `json:"runtime_image"`
	RequiredCPUs    float64  `json:"required_cpus"`
	RequiredMemMB   int      `json:"required_mem_mb"`
	RequiredGPU     bool     `json:"required_gpu"`
	CapsuleManifest CapsuleManifest `json:"capsule_manifest"`
	Environment     map[string]string `json:"environment,omitempty"`
	Timeout         time.Duration     `json:"timeout"`
	Priority        int               `json:"priority"`
}

// CapsuleManifest describes the contents of an Execution Capsule.
type CapsuleManifest struct {
	CapsuleID        string       `json:"capsule_id"`
	DetectedLanguage string       `json:"detected_language"`
	RuntimeImage     string       `json:"runtime_image"`
	Entrypoint       string       `json:"entrypoint"`
	TotalSizeBytes   int64        `json:"total_size_bytes"`
	FileCount        int          `json:"file_count"`
	Files            []FileEntry  `json:"files"`
	Dependencies     *DependencyInfo `json:"dependencies,omitempty"`
	SHA256           string       `json:"sha256"`
}

// FileEntry describes a single file in the Execution Capsule.
type FileEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	IsDir  bool   `json:"is_dir"`
}

// DependencyInfo describes how to install dependencies for the capsule.
type DependencyInfo struct {
	Type    string `json:"type"`    // "pip", "npm", "cargo", "go-mod", "make"
	File    string `json:"file"`    // "requirements.txt", "package.json", etc.
	Command string `json:"command"` // Install command to run
}

// TaskAssignPayload is sent by the coordinator to assign a task to a worker.
type TaskAssignPayload struct {
	TaskID         string `json:"task_id"`
	JobID          string `json:"job_id"`
	RequesterID    string `json:"requester_id"`
	SignalingToken string `json:"signaling_token"`
	Entrypoint     string `json:"entrypoint"`
	RuntimeImage   string `json:"runtime_image"`
	Environment    map[string]string `json:"environment,omitempty"`
	Timeout        time.Duration     `json:"timeout"`
	IsSpotCheck    bool              `json:"is_spot_check"`
}

// TaskProgressPayload reports progress updates for a running task.
type TaskProgressPayload struct {
	TaskID     string  `json:"task_id"`
	Progress   float64 `json:"progress"` // 0.0 to 1.0
	Phase      string  `json:"phase"`    // "downloading", "installing", "running", "uploading"
	Message    string  `json:"message,omitempty"`
}

// SDPPayload carries WebRTC Session Description Protocol data.
type SDPPayload struct {
	SessionID string `json:"session_id"`
	SDP       string `json:"sdp"`
	Type      string `json:"type"` // "offer" or "answer"
}

// ICECandidatePayload carries ICE candidate information for NAT traversal.
type ICECandidatePayload struct {
	SessionID       string `json:"session_id"`
	Candidate       string `json:"candidate"`
	SDPMid          string `json:"sdp_mid"`
	SDPMLineIndex   uint16 `json:"sdp_mline_index"`
}

// ResumeStreamPayload is sent by a requester to resume stream from a sequence point.
type ResumeStreamPayload struct {
	TaskID           string `json:"task_id"`
	LastReceivedSeqID uint64 `json:"last_received_seq_id"`
}

// ErrorPayload carries error information.
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
