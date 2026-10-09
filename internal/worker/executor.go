package worker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/capsule"
	"github.com/unicompute/unicompute/internal/protocol"
)

// Executor manages the lifecycle of a task assignment on the worker.
type Executor struct {
	ticket    protocol.TaskTicket
	workspace string
	sandbox   Sandbox
}

// NewExecutor creates a new task executor.
func NewExecutor(ticket protocol.TaskTicket, baseWorkspaceDir string) (*Executor, error) {
	wsDir := filepath.Join(baseWorkspaceDir, ticket.TaskID)
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create task workspace: %w", err)
	}

	return &Executor{
		ticket:    ticket,
		workspace: wsDir,
		sandbox:   NewGVisorSandbox(),
	}, nil
}

// Prepare unpacks the Capsule into the task's workspace.
func (e *Executor) Prepare(capsuleArchive string) error {
	log.Info().Str("task_id", e.ticket.TaskID).Msg("Unpacking capsule payload")
	
	destDir := filepath.Join(e.workspace, "code")
	if err := capsule.Extract(capsuleArchive, destDir); err != nil {
		return fmt.Errorf("failed to unpack capsule: %w", err)
	}

	return nil
}

// Run configures the sandbox and starts execution, hooking up the stream writers.
func (e *Executor) Run(ctx context.Context, streamOut, streamErr io.Writer) (int, error) {
	// Configure Sandbox limits based on ticket/hardware (hardcoded defaults here for demo)
	cfg := SandboxConfig{
		CPUCores:   1.0,
		MemoryMB:   512,
		NetworkOut: false, // Strict isolation
		Image:      e.ticket.RuntimeImage,
		Entrypoint: e.ticket.Entrypoint,
		Mounts: map[string]string{
			filepath.Join(e.workspace, "code"): "/workspace",
		},
	}

	log.Info().Str("task_id", e.ticket.TaskID).Msg("Creating sandbox")
	if err := e.sandbox.Create(ctx, cfg); err != nil {
		return -1, fmt.Errorf("sandbox creation failed: %w", err)
	}
	
	defer e.sandbox.Cleanup(context.Background())

	log.Info().Str("task_id", e.ticket.TaskID).Msg("Starting sandbox execution")
	if err := e.sandbox.Start(ctx, streamOut, streamErr); err != nil {
		return -1, fmt.Errorf("sandbox start failed: %w", err)
	}

	// Start resource monitor
	monitor := NewResourceMonitor(e.sandbox)
	monCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go monitor.Start(monCtx)

	// Wait for exit
	exitCode, err := e.sandbox.Wait(ctx)
	if err != nil {
		return -1, fmt.Errorf("sandbox wait failed: %w", err)
	}

	log.Info().Str("task_id", e.ticket.TaskID).Int("exit_code", exitCode).Msg("Sandbox execution finished")
	return exitCode, nil
}

// Cleanup removes the workspace directory.
func (e *Executor) Cleanup() error {
	return os.RemoveAll(e.workspace)
}
