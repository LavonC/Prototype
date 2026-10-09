package coordinator

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/storage"
)

// TaskQueue manages pending tasks in memory, ordering by priority.
type TaskQueue struct {
	mu    sync.Mutex
	tasks []*storage.TaskRecord
}

// NewTaskQueue initializes a new TaskQueue.
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks: make([]*storage.TaskRecord, 0),
	}
}

// Enqueue adds a task to the queue and sorts by priority (highest first), then by CreatedAt.
func (q *TaskQueue) Enqueue(task *storage.TaskRecord) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks = append(q.tasks, task)
	// Sort could be optimized with a heap, but for simplicity we sort on insert or just search linearly.
	// We'll leave it as an append here and let the matchmaker scan to find the best match.
	log.Debug().Str("task_id", task.TaskID).Msg("Task enqueued")
}

// Dequeue removes and returns the first task that satisfies the given predicate.
func (q *TaskQueue) Dequeue(predicate func(*storage.TaskRecord) bool) *storage.TaskRecord {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, task := range q.tasks {
		if predicate(task) {
			// Remove from slice
			q.tasks = append(q.tasks[:i], q.tasks[i+1:]...)
			return task
		}
	}
	return nil
}

// LoadPendingTasks populates the queue from the database on startup.
func (q *TaskQueue) LoadPendingTasks(ctx context.Context, db storage.DB) error {
	pending, err := db.GetPendingTasks(ctx, 1000)
	if err != nil {
		return err
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = pending
	log.Info().Int("count", len(pending)).Msg("Loaded pending tasks into queue")
	return nil
}

// Requeue adds a task back to the queue (e.g., after worker failure).
func (q *TaskQueue) Requeue(task *storage.TaskRecord) {
	task.Status = "PENDING"
	task.Attempt++
	q.Enqueue(task)
}

// Len returns the current queue size.
func (q *TaskQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.tasks)
}

// Matchmaker coordinates matching queued tasks with available workers.
type Matchmaker struct {
	db    storage.DB
	queue *TaskQueue
}

// NewMatchmaker creates a new Matchmaker.
func NewMatchmaker(db storage.DB, queue *TaskQueue) *Matchmaker {
	return &Matchmaker{
		db:    db,
		queue: queue,
	}
}

// MatchAndAssign attempts to find a suitable pending task for the given worker.
// Returns the task if a match is found, or nil if no suitable task exists.
func (m *Matchmaker) MatchAndAssign(worker *storage.WorkerRecord) *storage.TaskRecord {
	if !worker.Online {
		return nil
	}

	// Define match predicate based on worker capabilities
	predicate := func(t *storage.TaskRecord) bool {
		// Example matching logic based on requirements (assumed decoded from entrypoint/metadata if needed)
		// For now, we just check runtime compatibility
		// SupportedRuntimes is comma-separated
		supportsRuntime := false
		if strings.Contains(worker.SupportedRuntimes, t.RuntimeImage) {
			supportsRuntime = true
		}

		// Check basic resource constraints (if we had them expanded in TaskRecord)
		// Assuming we can parse them from metadata or they are implicit
		
		return supportsRuntime
	}

	task := m.queue.Dequeue(predicate)
	if task != nil {
		// Update task state
		task.WorkerID = worker.WorkerID
		task.Status = "ASSIGNED"
		// In a real system, we'd also update the DB here
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err := m.db.UpdateTaskStatus(ctx, task.TaskID, task.Status, map[string]interface{}{
			"worker_id": worker.WorkerID,
		})
		if err != nil {
			log.Error().Err(err).Str("task_id", task.TaskID).Msg("Failed to update task status in DB during matchmaking")
			// Re-queue
			m.queue.Requeue(task)
			return nil
		}
		
		log.Info().Str("task_id", task.TaskID).Str("worker_id", worker.WorkerID).Msg("Task matched and assigned")
		return task
	}

	return nil
}

// MatchLoop continuously polls the queue and database to match unassigned tasks to idle workers.
func (m *Matchmaker) MatchLoop(ctx context.Context, notifyAssign func(*storage.TaskRecord)) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if m.queue.Len() == 0 {
				continue
			}

			// Get all online workers with capacity
			// Simplified: just list online workers
			workers, err := m.db.ListWorkers(ctx, storage.WorkerFilter{OnlineOnly: true})
			if err != nil {
				log.Error().Err(err).Msg("Failed to list online workers for matchmaking")
				continue
			}

			// For each worker, try to assign a task
			for _, worker := range workers {
				if m.queue.Len() == 0 {
					break // Queue empty
				}
				
				// Optional: Check worker active tasks vs max concurrent tasks
				// Skipping detailed load check for this prototype
				
				task := m.MatchAndAssign(worker)
				if task != nil {
					// Notify the signaling system to send TASK_ASSIGN to the worker
					if notifyAssign != nil {
						notifyAssign(task)
					}
				}
			}
		}
	}
}
