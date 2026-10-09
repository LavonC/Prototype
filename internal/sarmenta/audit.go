package sarmenta

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/storage"
)

// Auditor handles backtracking and re-evaluating tasks completed by a malicious worker.
type Auditor struct {
	db     storage.DB
	config config.ByzantineConfig
}

// NewAuditor creates a new auditing engine.
func NewAuditor(db storage.DB, cfg config.ByzantineConfig) *Auditor {
	return &Auditor{
		db:     db,
		config: cfg,
	}
}

// AuditWorker finds all recently completed tasks by a malicious worker and invalidates them.
func (a *Auditor) AuditWorker(ctx context.Context, workerID string) {
	since := time.Now().Add(-a.config.AuditWindowDuration)
	
	log.Info().
		Str("worker_id", workerID).
		Time("since", since).
		Msg("Starting audit on malicious worker")

	tasks, err := a.db.GetRecentTasksByWorker(ctx, workerID, since)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve recent tasks for audit")
		return
	}

	count := 0
	for _, task := range tasks {
		if task.IsSpotCheck {
			continue // Don't re-queue dummy spot-checks
		}

		log.Info().Str("task_id", task.TaskID).Msg("Invalidating task and re-queueing due to audit")
		
		// Reset task state so it goes back into the queue
		err := a.db.UpdateTaskStatus(ctx, task.TaskID, "PENDING", map[string]interface{}{
			"worker_id":      "",
			"result_sha256":  "",
			"artifact_path":  "",
			"artifact_size":  0,
			"attempt":        0, // Reset attempts so it gets a fresh start
			"expected_hash":  "", // Invalidate expected hash since it was provided by malicious worker
		})
		
		if err != nil {
			log.Error().Err(err).Str("task_id", task.TaskID).Msg("Failed to re-queue audited task")
		} else {
			count++
			// In a real system, we'd also notify the queue manager to actually enqueue it in memory again,
			// or the queue manager would poll for these status changes.
		}
	}
	
	log.Info().Int("requeued_tasks", count).Msg("Audit complete")
}
