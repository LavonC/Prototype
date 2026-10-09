// Package config provides configuration loading, defaults, and validation for UniCompute.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultConfig returns the default configuration with sane defaults.
func DefaultConfig() *AppConfig {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".unicompute")

	return &AppConfig{
		Mode: "requester",

		Coordinator: CoordinatorConfig{
			Host:                "0.0.0.0",
			Port:                8600,
			HeartbeatInterval:   2 * time.Second,
			HeartbeatTimeout:    6 * time.Second,
			MaxWorkersPerJob:    64,
			TaskReDispatchDelay: 1 * time.Second,
			Byzantine: ByzantineConfig{
				InitialCredibility:   0.5,
				HighTrustThreshold:   0.85,
				CredibilityGainRate:  0.1,
				CredibilityDecayRate: 0.999,
				SpotCheckProbability: 0.05,
				ReplicationFactor:    2,
				AuditWindowDuration:  1 * time.Hour,
			},
		},

		Worker: WorkerConfig{
			CoordinatorURL:     "wss://coordinator.unicompute.io",
			MaxCPUs:            2.0,
			MaxMemoryMB:        4096,
			MaxDiskMB:          10240,
			MaxConcurrentTasks: 1,
			RuntimeCacheDir:    filepath.Join(dataDir, "runtimes"),
			WorkspaceDir:       filepath.Join(dataDir, "workspaces"),
			PauseOnUserActivity: true,
			PauseOnBattery:      true,
			IdleThreshold:       30 * time.Second,
			Sandbox: SandboxConfig{
				Runtime:          "runsc",
				CPUs:             2.0,
				MemoryMB:         4096,
				PidsLimit:        256,
				UID:              10001,
				GID:              10001,
				ReadOnlyRootfs:   true,
				TmpfsMounts:      []string{"/workspace/output", "/tmp"},
				BlockedSubnets:   []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"},
				MaxExecutionTime: 1 * time.Hour,
				NetworkEnabled:   true,
			},
		},

		Requester: RequesterConfig{
			CoordinatorURL:     "wss://coordinator.unicompute.io",
			OutputDir:          "./results",
			MaxCapsuleSizeMB:   50,
			DefaultConcurrency: 4,
			StreamBufferSize:   65536,
			AutoRetry:          true,
			MaxRetries:         3,
		},

		P2P: P2PConfig{
			STUNServers: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
				"stun:stun2.l.google.com:19302",
			},
			TURNServers:              []TURNServer{},
			MaxDataChannelBufferSize: 16 * 1024 * 1024, // 16 MB
			FileTransferChunkSize:    64 * 1024,         // 64 KB
			RingBufferSize:           10 * 1024 * 1024,  // 10 MB
			ICEConnectionTimeout:     30 * time.Second,
		},

		Security: SecurityConfig{
			JWTExpiration:       24 * time.Hour,
			APIKeyHashAlgorithm: "sha256",
			EnableTLS:           false,
		},

		Logging: LoggingConfig{
			Level:        "info",
			Format:       "console",
			OutputPath:   "",
			EnableCaller: false,
		},

		Storage: StorageConfig{
			Driver:        "sqlite",
			DSN:           filepath.Join(dataDir, "unicompute.db"),
			MaxOpenConns:  10,
			MaxIdleConns:  5,
			MigrationsDir: "internal/storage/migrations",
		},
	}
}

// Validate checks the configuration for errors and returns a list of issues.
func (c *AppConfig) Validate() error {
	var errs []string

	switch c.Mode {
	case "requester", "worker", "coordinator":
		// valid
	default:
		errs = append(errs, fmt.Sprintf("invalid mode %q: must be requester, worker, or coordinator", c.Mode))
	}

	if c.Mode == "coordinator" {
		if c.Coordinator.Port < 1 || c.Coordinator.Port > 65535 {
			errs = append(errs, fmt.Sprintf("invalid coordinator port: %d", c.Coordinator.Port))
		}
		if c.Coordinator.HeartbeatInterval <= 0 {
			errs = append(errs, "heartbeat_interval must be positive")
		}
		if c.Coordinator.HeartbeatTimeout <= c.Coordinator.HeartbeatInterval {
			errs = append(errs, "heartbeat_timeout must be greater than heartbeat_interval")
		}
		byz := c.Coordinator.Byzantine
		if byz.InitialCredibility < 0 || byz.InitialCredibility > 1 {
			errs = append(errs, "initial_credibility must be between 0.0 and 1.0")
		}
		if byz.HighTrustThreshold < 0 || byz.HighTrustThreshold > 1 {
			errs = append(errs, "high_trust_threshold must be between 0.0 and 1.0")
		}
		if byz.ReplicationFactor < 1 {
			errs = append(errs, "replication_factor must be at least 1")
		}
	}

	if c.Mode == "worker" {
		if c.Worker.CoordinatorURL == "" {
			errs = append(errs, "worker coordinator_url is required")
		}
		if c.Worker.MaxCPUs <= 0 {
			errs = append(errs, "max_cpus must be positive")
		}
		if c.Worker.MaxMemoryMB <= 0 {
			errs = append(errs, "max_memory_mb must be positive")
		}
		if c.Worker.Sandbox.PidsLimit < 1 {
			errs = append(errs, "sandbox pids_limit must be at least 1")
		}
	}

	if c.Mode == "requester" {
		if c.Requester.CoordinatorURL == "" {
			errs = append(errs, "requester coordinator_url is required")
		}
		if c.Requester.MaxCapsuleSizeMB <= 0 {
			errs = append(errs, "max_capsule_size_mb must be positive")
		}
	}

	if len(c.P2P.STUNServers) == 0 {
		errs = append(errs, "at least one STUN server is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", joinErrors(errs))
	}

	return nil
}

func joinErrors(errs []string) string {
	result := ""
	for i, e := range errs {
		if i > 0 {
			result += "\n  - "
		}
		result += e
	}
	return result
}
