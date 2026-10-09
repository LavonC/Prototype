package coordinator

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/protocol"
	"github.com/unicompute/unicompute/internal/storage"
)

// HeartbeatMonitor tracks worker check-ins and detects disconnected or stalled workers.
type HeartbeatMonitor struct {
	db      storage.DB
	config  config.CoordinatorConfig
	queue   *TaskQueue
	session *SessionManager
}

// NewHeartbeatMonitor creates a new HeartbeatMonitor.
func NewHeartbeatMonitor(db storage.DB, cfg config.CoordinatorConfig, queue *TaskQueue, session *SessionManager) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		db:      db,
		config:  cfg,
		queue:   queue,
		session: session,
	}
}

// Start polling for stale heartbeats.
func (hm *HeartbeatMonitor) Start(ctx context.Context) {
	// Watchdog runs every few seconds
	ticker := time.NewTicker(hm.config.HeartbeatTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.checkStaleWorkers(ctx)
		}
	}
}

// HandleHeartbeat updates the worker's last seen time in the database.
func (hm *HeartbeatMonitor) HandleHeartbeat(ctx context.Context, hb *protocol.WorkerHeartbeatPayload) error {
	err := hm.db.UpdateWorkerStatus(ctx, hb.WorkerID, true, time.Now())
	if err != nil {
		log.Error().Err(err).Str("worker_id", hb.WorkerID).Msg("Failed to update heartbeat in DB")
		return err
	}
	
	// Could also update metrics/load in memory for matchmaking here
	return nil
}

// checkStaleWorkers queries the DB for workers that haven't sent a heartbeat within the timeout.
func (hm *HeartbeatMonitor) checkStaleWorkers(ctx context.Context) {
	threshold := time.Now().Add(-hm.config.HeartbeatTimeout)
	
	// Query workers marked as online but with last_heartbeat < threshold
	workers, err := hm.db.ListWorkers(ctx, storage.WorkerFilter{OnlineOnly: true})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list workers for heartbeat check")
		return
	}

	for _, w := range workers {
		if w.LastHeartbeat.Before(threshold) {
			log.Warn().Str("worker_id", w.WorkerID).Time("last_heartbeat", w.LastHeartbeat).Msg("Worker timed out. Marking UNREACHABLE")
			
			// Mark offline
			hm.db.UpdateWorkerStatus(ctx, w.WorkerID, false, w.LastHeartbeat)

			// Re-dispatch any tasks currently assigned to this worker
			hm.reDispatchTasks(ctx, w.WorkerID)
			
			// Notify other components if needed (e.g. Requester tracking the task)
			// For this prototype, Requester will poll or be notified when task status changes
		}
	}
}

func (hm *HeartbeatMonitor) reDispatchTasks(ctx context.Context, workerID string) {
	tasks, err := hm.db.ListTasks(ctx, storage.TaskFilter{
		WorkerID: workerID,
		Status:   "RUNNING", // Or ASSIGNED, DOWNLOADING, etc.
	})
	if err != nil {
		log.Error().Err(err).Str("worker_id", workerID).Msg("Failed to list running tasks for stalled worker")
		return
	}

	for _, t := range tasks {
		if t.Attempt < t.MaxAttempts {
			log.Info().Str("task_id", t.TaskID).Str("worker_id", workerID).Msg("Re-dispatching task from failed worker")
			
			// Re-queue
			hm.queue.Requeue(t)
			
			// Update DB
			hm.db.UpdateTaskStatus(ctx, t.TaskID, "RE_DISPATCHED", nil)
		} else {
			log.Error().Str("task_id", t.TaskID).Msg("Task failed max attempts after worker failure")
			hm.db.UpdateTaskStatus(ctx, t.TaskID, "FAILED", map[string]interface{}{
				"error_message": "Worker unreachable, max retries exceeded",
			})
		}
	}
}
