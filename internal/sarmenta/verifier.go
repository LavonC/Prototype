package sarmenta

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/protocol"
	"github.com/unicompute/unicompute/internal/storage"
)

// Verifier inspects incoming task receipts for correctness.
type Verifier struct {
	db      storage.DB
	config  config.ByzantineConfig
	tracker *CredibilityTracker
	auditor *Auditor
}

// NewVerifier creates a new Verifier engine.
func NewVerifier(db storage.DB, cfg config.ByzantineConfig, tracker *CredibilityTracker, auditor *Auditor) *Verifier {
	return &Verifier{
		db:      db,
		config:  cfg,
		tracker: tracker,
		auditor: auditor,
	}
}

// VerifyReceipt checks a received task receipt against expected outcomes.
func (v *Verifier) VerifyReceipt(ctx context.Context, receipt *protocol.CompletionReceipt) error {
	task, err := v.db.GetTask(ctx, receipt.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task for verification: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task %s not found", receipt.TaskID)
	}

	actualHash := ""
	if receipt.ArtifactBundle != nil {
		actualHash = receipt.ArtifactBundle.SHA256
	}

	isSpotCheck := task.IsSpotCheck
	passed := true

	if isSpotCheck {
		passed = (actualHash == task.ExpectedHash)
		
		log.Info().
			Str("task_id", task.TaskID).
			Str("worker_id", receipt.WorkerID).
			Bool("passed", passed).
			Msg("Evaluated spot-check result")
			
	} else if task.ExpectedHash != "" {
		// This handles the case of redundant execution verification (Voting)
		passed = (actualHash == task.ExpectedHash)
		
		log.Info().
			Str("task_id", task.TaskID).
			Str("worker_id", receipt.WorkerID).
			Bool("passed", passed).
			Msg("Evaluated redundant execution result")
	} else {
		// Normal task, nothing to compare against right now. 
		// For redundant execution, the first arrival sets the expected hash.
		if v.config.ReplicationFactor > 1 {
			// Set this hash as the expected one for the replicas to verify against
			task.ExpectedHash = actualHash
			v.db.UpdateTaskStatus(ctx, task.TaskID, task.Status, map[string]interface{}{
				"expected_hash": actualHash,
			})
		}
		
		// If it's not a spot check and there is no prior hash, it passes by default 
		// (credibility is earned if it survives later spot checks).
	}

	// Update Credibility if this was an evaluated task
	if isSpotCheck || task.ExpectedHash != "" {
		oldCred, _ := v.db.GetCredibility(ctx, receipt.WorkerID)
		newCred, err := v.tracker.UpdateCredibility(ctx, receipt.WorkerID, passed)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update credibility")
		}

		// Log verification event
		record := &storage.VerificationRecord{
			ID:             uuid.New().String(),
			TaskID:         task.TaskID,
			WorkerID:       receipt.WorkerID,
			ExpectedHash:   task.ExpectedHash,
			ActualHash:     actualHash,
			Passed:         passed,
			IsSpotCheck:    isSpotCheck,
			OldCredibility: oldCred,
			NewCredibility: newCred,
			VerifiedAt:     time.Now(),
		}
		v.db.RecordVerification(ctx, record)

		if !passed {
			// Trigger sabotage audit
			log.Warn().Str("worker_id", receipt.WorkerID).Msg("Sabotage detected! Triggering audit.")
			v.auditor.AuditWorker(ctx, receipt.WorkerID)
			
			return fmt.Errorf("result verification failed (sabotage detected)")
		}
	}

	return nil
}
