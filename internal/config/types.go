// Package config defines all configuration structures for UniCompute components.
package config

import "time"

// AppConfig is the root configuration for all UniCompute components.
type AppConfig struct {
	// Mode specifies which component mode to run: "requester", "worker", "coordinator".
	Mode string `mapstructure:"mode" json:"mode"`

	// Coordinator configuration for the cloud coordination plane.
	Coordinator CoordinatorConfig `mapstructure:"coordinator" json:"coordinator"`

	// Worker configuration for the volunteer worker daemon.
	Worker WorkerConfig `mapstructure:"worker" json:"worker"`

	// Requester configuration for the developer CLI/desktop client.
	Requester RequesterConfig `mapstructure:"requester" json:"requester"`

	// P2P networking configuration.
	P2P P2PConfig `mapstructure:"p2p" json:"p2p"`

	// Security configuration.
	Security SecurityConfig `mapstructure:"security" json:"security"`

	// Logging configuration.
	Logging LoggingConfig `mapstructure:"logging" json:"logging"`

	// Storage configuration.
	Storage StorageConfig `mapstructure:"storage" json:"storage"`
}

// CoordinatorConfig holds settings for the cloud coordination server.
type CoordinatorConfig struct {
	// Host address to bind the server.
	Host string `mapstructure:"host" json:"host"`

	// Port number for HTTP/WebSocket server.
	Port int `mapstructure:"port" json:"port"`

	// TLSCertFile is the path to the TLS certificate.
	TLSCertFile string `mapstructure:"tls_cert_file" json:"tls_cert_file"`

	// TLSKeyFile is the path to the TLS private key.
	TLSKeyFile string `mapstructure:"tls_key_file" json:"tls_key_file"`

	// HeartbeatInterval is the expected interval between worker heartbeats.
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval" json:"heartbeat_interval"`

	// HeartbeatTimeout is the duration after which a worker is considered unreachable.
	// This is typically 3x the heartbeat interval (6 seconds default).
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat_timeout" json:"heartbeat_timeout"`

	// MaxWorkersPerJob limits the number of workers assigned to a single job.
	MaxWorkersPerJob int `mapstructure:"max_workers_per_job" json:"max_workers_per_job"`

	// TaskReDispatchDelay is the delay before re-dispatching a failed task.
	TaskReDispatchDelay time.Duration `mapstructure:"task_redispatch_delay" json:"task_redispatch_delay"`

	// Byzantine verification settings.
	Byzantine ByzantineConfig `mapstructure:"byzantine" json:"byzantine"`
}

// ByzantineConfig holds settings for the Sarmenta sabotage-tolerance model.
type ByzantineConfig struct {
	// InitialCredibility is the starting trust score for new workers [0.0, 1.0].
	InitialCredibility float64 `mapstructure:"initial_credibility" json:"initial_credibility"`

	// HighTrustThreshold is the credibility score above which no redundancy is needed.
	HighTrustThreshold float64 `mapstructure:"high_trust_threshold" json:"high_trust_threshold"`

	// CredibilityGainRate controls how fast credibility increases on correct results.
	CredibilityGainRate float64 `mapstructure:"credibility_gain_rate" json:"credibility_gain_rate"`

	// CredibilityDecayRate is the hourly multiplicative decay for inactive workers.
	CredibilityDecayRate float64 `mapstructure:"credibility_decay_rate" json:"credibility_decay_rate"`

	// SpotCheckProbability is the chance (0.0–1.0) of injecting a spot-check task.
	SpotCheckProbability float64 `mapstructure:"spot_check_probability" json:"spot_check_probability"`

	// ReplicationFactor is the number of redundant task copies for untrusted workers.
	ReplicationFactor int `mapstructure:"replication_factor" json:"replication_factor"`

	// AuditWindowDuration is how far back to re-queue tasks if a worker fails a spot-check.
	AuditWindowDuration time.Duration `mapstructure:"audit_window_duration" json:"audit_window_duration"`
}

// WorkerConfig holds settings for the volunteer worker daemon.
type WorkerConfig struct {
	// CoordinatorURL is the WebSocket URL of the coordinator server.
	CoordinatorURL string `mapstructure:"coordinator_url" json:"coordinator_url"`

	// WorkerID is the unique identifier for this worker (auto-generated if empty).
	WorkerID string `mapstructure:"worker_id" json:"worker_id"`

	// AuthToken is the authentication token for coordinator registration.
	AuthToken string `mapstructure:"auth_token" json:"auth_token"`

	// MaxCPUs is the maximum number of CPU cores to allocate to tasks.
	MaxCPUs float64 `mapstructure:"max_cpus" json:"max_cpus"`

	// MaxMemoryMB is the maximum RAM in megabytes to allocate to tasks.
	MaxMemoryMB int `mapstructure:"max_memory_mb" json:"max_memory_mb"`

	// MaxDiskMB is the maximum disk space in megabytes for task workspace.
	MaxDiskMB int `mapstructure:"max_disk_mb" json:"max_disk_mb"`

	// MaxConcurrentTasks limits how many tasks run simultaneously on this worker.
	MaxConcurrentTasks int `mapstructure:"max_concurrent_tasks" json:"max_concurrent_tasks"`

	// RuntimeCacheDir is the local directory for cached OCI runtime images.
	RuntimeCacheDir string `mapstructure:"runtime_cache_dir" json:"runtime_cache_dir"`

	// WorkspaceDir is the root directory for task workspaces.
	WorkspaceDir string `mapstructure:"workspace_dir" json:"workspace_dir"`

	// PauseOnUserActivity enables preemption when user mouse/keyboard activity is detected.
	PauseOnUserActivity bool `mapstructure:"pause_on_user_activity" json:"pause_on_user_activity"`

	// PauseOnBattery pauses tasks when the machine is on battery power.
	PauseOnBattery bool `mapstructure:"pause_on_battery" json:"pause_on_battery"`

	// IdleThreshold is the duration of inactivity before resuming paused tasks.
	IdleThreshold time.Duration `mapstructure:"idle_threshold" json:"idle_threshold"`

	// Sandbox configuration for gVisor isolation.
	Sandbox SandboxConfig `mapstructure:"sandbox" json:"sandbox"`
}

// SandboxConfig configures the gVisor (runsc) sandbox for task execution.
type SandboxConfig struct {
	// Runtime is the OCI runtime binary name (default: "runsc").
	Runtime string `mapstructure:"runtime" json:"runtime"`

	// CPUs is the hard CPU core limit for the sandbox.
	CPUs float64 `mapstructure:"cpus" json:"cpus"`

	// MemoryMB is the hard memory limit in megabytes.
	MemoryMB int `mapstructure:"memory_mb" json:"memory_mb"`

	// PidsLimit is the maximum number of processes inside the sandbox.
	PidsLimit int `mapstructure:"pids_limit" json:"pids_limit"`

	// UID is the non-root user ID for the sandbox.
	UID int `mapstructure:"uid" json:"uid"`

	// GID is the non-root group ID for the sandbox.
	GID int `mapstructure:"gid" json:"gid"`

	// ReadOnlyRootfs makes the root filesystem read-only.
	ReadOnlyRootfs bool `mapstructure:"readonly_rootfs" json:"readonly_rootfs"`

	// TmpfsMounts are the tmpfs mount points inside the sandbox.
	TmpfsMounts []string `mapstructure:"tmpfs_mounts" json:"tmpfs_mounts"`

	// BlockedSubnets are the RFC 1918 private subnets blocked by iptables.
	BlockedSubnets []string `mapstructure:"blocked_subnets" json:"blocked_subnets"`

	// MaxExecutionTime is the maximum wall-clock duration for a single task.
	MaxExecutionTime time.Duration `mapstructure:"max_execution_time" json:"max_execution_time"`

	// NetworkEnabled controls whether the sandbox has outbound internet access.
	NetworkEnabled bool `mapstructure:"network_enabled" json:"network_enabled"`
}

// RequesterConfig holds settings for the developer requester client.
type RequesterConfig struct {
	// CoordinatorURL is the WebSocket URL of the coordinator server.
	CoordinatorURL string `mapstructure:"coordinator_url" json:"coordinator_url"`

	// AuthToken is the authentication token for the coordinator.
	AuthToken string `mapstructure:"auth_token" json:"auth_token"`

	// OutputDir is the default local directory for task results.
	OutputDir string `mapstructure:"output_dir" json:"output_dir"`

	// MaxCapsuleSizeMB is the maximum allowed Execution Capsule size.
	MaxCapsuleSizeMB int `mapstructure:"max_capsule_size_mb" json:"max_capsule_size_mb"`

	// DefaultConcurrency is the default number of parallel tasks for `map` commands.
	DefaultConcurrency int `mapstructure:"default_concurrency" json:"default_concurrency"`

	// StreamBufferSize is the size of the local stream receive buffer.
	StreamBufferSize int `mapstructure:"stream_buffer_size" json:"stream_buffer_size"`

	// AutoRetry enables automatic retry of failed tasks.
	AutoRetry bool `mapstructure:"auto_retry" json:"auto_retry"`

	// MaxRetries is the maximum number of retries per task.
	MaxRetries int `mapstructure:"max_retries" json:"max_retries"`
}

// P2PConfig holds settings for peer-to-peer networking.
type P2PConfig struct {
	// STUNServers is a list of STUN server URLs for ICE candidate gathering.
	STUNServers []string `mapstructure:"stun_servers" json:"stun_servers"`

	// TURNServers is a list of TURN server configurations for relay fallback.
	TURNServers []TURNServer `mapstructure:"turn_servers" json:"turn_servers"`

	// MaxDataChannelBufferSize is the maximum buffer size per DataChannel in bytes.
	MaxDataChannelBufferSize int `mapstructure:"max_datachannel_buffer_size" json:"max_datachannel_buffer_size"`

	// FileTransferChunkSize is the chunk size for file transfers over DataChannels.
	FileTransferChunkSize int `mapstructure:"file_transfer_chunk_size" json:"file_transfer_chunk_size"`

	// RingBufferSize is the size of the stream resilience ring buffer in bytes.
	RingBufferSize int `mapstructure:"ring_buffer_size" json:"ring_buffer_size"`

	// ICEConnectionTimeout is the maximum time to establish an ICE connection.
	ICEConnectionTimeout time.Duration `mapstructure:"ice_connection_timeout" json:"ice_connection_timeout"`
}

// TURNServer holds credentials for a TURN relay server.
type TURNServer struct {
	// URL of the TURN server.
	URL string `mapstructure:"url" json:"url"`

	// Username for TURN authentication.
	Username string `mapstructure:"username" json:"username"`

	// Credential (password) for TURN authentication.
	Credential string `mapstructure:"credential" json:"credential"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	// JWTSecret is the HMAC secret for JWT token signing.
	JWTSecret string `mapstructure:"jwt_secret" json:"jwt_secret"`

	// JWTExpiration is the duration for which JWT tokens are valid.
	JWTExpiration time.Duration `mapstructure:"jwt_expiration" json:"jwt_expiration"`

	// APIKeyHashAlgorithm is the algorithm for hashing API keys (default: "sha256").
	APIKeyHashAlgorithm string `mapstructure:"api_key_hash_algorithm" json:"api_key_hash_algorithm"`

	// EnableTLS forces TLS on all connections.
	EnableTLS bool `mapstructure:"enable_tls" json:"enable_tls"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	// Level is the minimum log level: "debug", "info", "warn", "error".
	Level string `mapstructure:"level" json:"level"`

	// Format is the log output format: "json", "console".
	Format string `mapstructure:"format" json:"format"`

	// OutputPath is the log file path (empty for stdout).
	OutputPath string `mapstructure:"output_path" json:"output_path"`

	// EnableCaller adds caller file and line to log entries.
	EnableCaller bool `mapstructure:"enable_caller" json:"enable_caller"`
}

// StorageConfig holds database storage settings.
type StorageConfig struct {
	// Driver is the database driver: "sqlite", "postgres".
	Driver string `mapstructure:"driver" json:"driver"`

	// DSN is the database connection string.
	DSN string `mapstructure:"dsn" json:"dsn"`

	// MaxOpenConns is the maximum number of open database connections.
	MaxOpenConns int `mapstructure:"max_open_conns" json:"max_open_conns"`

	// MaxIdleConns is the maximum number of idle database connections.
	MaxIdleConns int `mapstructure:"max_idle_conns" json:"max_idle_conns"`

	// MigrationsDir is the directory containing SQL migration files.
	MigrationsDir string `mapstructure:"migrations_dir" json:"migrations_dir"`
}
