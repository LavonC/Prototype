package sarmenta

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/unicompute/unicompute/internal/protocol"
	"github.com/unicompute/unicompute/internal/storage"
)

// SpotCheckGenerator creates deterministic dummy tasks to test worker honesty.
type SpotCheckGenerator struct {
	// A real implementation would have a registry of known deterministic payloads
}

// NewSpotCheckGenerator creates a new generator.
func NewSpotCheckGenerator() *SpotCheckGenerator {
	return &SpotCheckGenerator{}
}

// Generate creates a TaskRecord that acts as a spot check.
func (g *SpotCheckGenerator) Generate() *storage.TaskRecord {
	taskID := "spot-" + uuid.New().String()[:8]
	
	// In a real system, the Entrypoint and Artifact hash would map to a known, pre-calculated result.
	// For this spec, we simulate a simple prime factorization task.
	
	// Create a deterministic hash for the expected artifact output
	hasher := sha256.New()
	hasher.Write([]byte("expected_spot_check_result_42"))
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	return &storage.TaskRecord{
		TaskID:        taskID,
		JobID:         "sys-spotcheck", // Internal system job
		RequesterID:   "system",
		Status:        "PENDING",
		Entrypoint:    "python3 prime_fact.py 10000",
		RuntimeImage:  "unicompute/runtime-python:3.11",
		Priority:      100, // High priority to get executed quickly
		CreatedAt:     time.Now(),
		MaxAttempts:   1, // Spot checks don't retry if they fail to run
		IsSpotCheck:   true,
		ExpectedHash:  expectedHash,
	}
}

// IsSpotCheck checks if a TaskTicket is a spot check.
func IsSpotCheck(ticket *protocol.TaskTicket) bool {
	return ticket.IsSpotCheck
}
