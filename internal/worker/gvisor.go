package worker

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// GVisorSandbox implements the Sandbox interface using Docker configured with runsc.
type GVisorSandbox struct {
	id     string
	config SandboxConfig
}

// NewGVisorSandbox creates a new gVisor sandbox instance.
func NewGVisorSandbox() *GVisorSandbox {
	return &GVisorSandbox{
		id: "uc-" + uuid.New().String()[:12],
	}
}

func (s *GVisorSandbox) Create(ctx context.Context, config SandboxConfig) error {
	s.config = config
	
	args := []string{
		"create",
		"--name", s.id,
		// Try to use runsc runtime (gVisor) if available. If not, fallback to runc based on system setup.
		// In a strict production worker, this would hard-require --runtime=runsc
		"--runtime=runsc",
		"--cpus", fmt.Sprintf("%.2f", config.CPUCores),
		"--memory", fmt.Sprintf("%dm", config.MemoryMB),
		"--rm", // Auto-remove on exit
	}

	if !config.NetworkOut {
		args = append(args, "--network", "none")
	}

	for k, v := range config.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for host, container := range config.Mounts {
		args = append(args, "-v", fmt.Sprintf("%s:%s", host, container))
	}

	// Working dir
	args = append(args, "-w", "/workspace")

	args = append(args, config.Image)
	
	if config.Entrypoint != "" {
		// Split entrypoint by space (simplified, real implementation needs proper shell parsing)
		epArgs := strings.Fields(config.Entrypoint)
		args = append(args, epArgs...)
	}

	args = append(args, config.Args...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	
	// Just create, don't start
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If runsc is not installed (e.g. testing on Windows/Mac without it), fallback for development
		if strings.Contains(string(out), "Unknown runtime specified runsc") {
			log.Warn().Msg("gVisor (runsc) not found, falling back to standard docker runtime for development")
			
			// Remove --runtime=runsc and try again
			var fallbackArgs []string
			for _, arg := range args {
				if arg != "--runtime=runsc" {
					fallbackArgs = append(fallbackArgs, arg)
				}
			}
			cmd = exec.CommandContext(ctx, "docker", fallbackArgs...)
			out, err = cmd.CombinedOutput()
		}
		
		if err != nil {
			return fmt.Errorf("docker create failed: %v, output: %s", err, string(out))
		}
	}

	return nil
}

func (s *GVisorSandbox) Start(ctx context.Context, stdout, stderr io.Writer) error {
	// docker start -a s.id
	cmd := exec.CommandContext(ctx, "docker", "start", "-a", s.id)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	
	// Start async, Wait will be called separately
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start sandbox: %w", err)
	}
	
	go func() {
		// Wait for the start command to finish attaching
		cmd.Wait()
	}()

	return nil
}

func (s *GVisorSandbox) Wait(ctx context.Context) (int, error) {
	// Wait for container to stop using docker wait
	cmd := exec.CommandContext(ctx, "docker", "wait", s.id)
	out, err := cmd.Output()
	if err != nil {
		return -1, fmt.Errorf("failed to wait for sandbox: %w", err)
	}

	exitCodeStr := strings.TrimSpace(string(out))
	exitCode, err := strconv.Atoi(exitCodeStr)
	if err != nil {
		return -1, fmt.Errorf("invalid exit code '%s': %w", exitCodeStr, err)
	}

	return exitCode, nil
}

func (s *GVisorSandbox) Stop(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", "2", s.id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}
	return nil
}

func (s *GVisorSandbox) Stats(ctx context.Context) (*SandboxStats, error) {
	// Parse docker stats
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{.CPUPerc}} {{.MemUsage}}", s.id)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parsing skipped for brevity (e.g. "0.00% 12MiB / 1GiB")
	// Returns a dummy struct for this spec implementation
	_ = string(out)

	return &SandboxStats{
		CPUUsagePercent: 0.0,
		MemoryUsageMB:   0.0,
	}, nil
}

func (s *GVisorSandbox) Cleanup(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", s.id)
	return cmd.Run()
}
