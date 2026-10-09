package sarmenta

import (
	"context"

	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/storage"
)

// CredibilityTracker implements the Sarmenta credibility update formula.
type CredibilityTracker struct {
	db     storage.DB
	config config.ByzantineConfig
}

// NewCredibilityTracker creates a new tracker.
func NewCredibilityTracker(db storage.DB, cfg config.ByzantineConfig) *CredibilityTracker {
	return &CredibilityTracker{
		db:     db,
		config: cfg,
	}
}

// UpdateCredibility updates a worker's credibility after a spot check or verification.
func (c *CredibilityTracker) UpdateCredibility(ctx context.Context, workerID string, passed bool) (float64, error) {
	oldCred, err := c.db.GetCredibility(ctx, workerID)
	if err != nil {
		return 0, err
	}

	var newCred float64
	
	// Sarmenta formula abstraction:
	// If passed, increase credibility asymptotically towards 1.0 based on gain rate.
	// If failed, drop credibility exponentially (or immediately to 0 for strict sabotage models).
	
	if passed {
		// e.g., new = old + (1 - old) * gainRate
		newCred = oldCred + (1.0-oldCred)*c.config.CredibilityGainRate
		if newCred > 1.0 {
			newCred = 1.0
		}
	} else {
		// Sabotage detected: strict penalty. Drop to 0 or a very low value.
		newCred = oldCred * 0.1 // 90% penalty
		if newCred < 0.01 {
			newCred = 0.0 // Blacklisted
		}
	}

	if err := c.db.UpdateCredibility(ctx, workerID, newCred); err != nil {
		return 0, err
	}

	return newCred, nil
}

// DecayCredibility applies hourly time-based decay to inactive workers.
func (c *CredibilityTracker) DecayCredibility(ctx context.Context) error {
	// In a real system, this would iterate over workers whose LastHeartbeat > 1 hour
	// and multiply their credibility by (1 - CredibilityDecayRate).
	// Implementation omitted for brevity in spec.
	return nil
}
