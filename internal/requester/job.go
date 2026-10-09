package requester

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/protocol"
)

// JobStatus represents the state of an entire distributed job.
type JobStatus string

const (
	JobPending   JobStatus = "PENDING"
	JobRunning   JobStatus = "RUNNING"
	JobCompleted JobStatus = "COMPLETED"
	JobFailed    JobStatus = "FAILED"
)

// SubTask tracks the state of a single parallel execution shard.
type SubTask struct {
	TaskID    string
	Status    protocol.TaskStatus
	WorkerID  string
	ExitCode  int
	ErrorMsg  string
	Artifact  string // Path to the downloaded result artifact
}

// JobManager tracks the submission, distribution, and completion of parallel sub-tasks.
type JobManager struct {
	JobID    string
	Status   JobStatus
	
	mu       sync.Mutex
	subTasks map[string]*SubTask
	
	onComplete func(jobID string)
}

// NewJobManager initializes a new job consisting of multiple parallel tasks.
func NewJobManager() *JobManager {
	return &JobManager{
		JobID:    "job-" + uuid.New().String()[:8],
		Status:   JobPending,
		subTasks: make(map[string]*SubTask),
	}
}

// AddSubTask registers a task ID belonging to this job.
func (j *JobManager) AddSubTask(taskID string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.subTasks[taskID] = &SubTask{
		TaskID: taskID,
		Status: "PENDING",
	}
}

// UpdateSubTask updates the status of a specific shard.
func (j *JobManager) UpdateSubTask(taskID string, status protocol.TaskStatus, workerID string, exitCode int, artifact string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	
	task, ok := j.subTasks[taskID]
	if !ok {
		log.Warn().Str("task_id", taskID).Msg("Received update for unknown sub-task")
		return
	}
	
	task.Status = status
	if workerID != "" {
		task.WorkerID = workerID
	}
	if status == "COMPLETED" || status == "FAILED" {
		task.ExitCode = exitCode
		task.Artifact = artifact
	}
	
	j.checkJobCompletion()
}

// checkJobCompletion verifies if all sub-tasks have finished.
// Must be called with lock held.
func (j *JobManager) checkJobCompletion() {
	allDone := true
	anyFailed := false
	
	for _, t := range j.subTasks {
		if t.Status != "COMPLETED" && t.Status != "FAILED" {
			allDone = false
			break
		}
		if t.Status == "FAILED" || t.ExitCode != 0 {
			anyFailed = true
		}
	}
	
	if allDone {
		if anyFailed {
			j.Status = JobFailed
		} else {
			j.Status = JobCompleted
		}
		
		log.Info().Str("job_id", j.JobID).Str("status", string(j.Status)).Msg("Job finished")
		
		if j.onComplete != nil {
			go j.onComplete(j.JobID)
		}
	} else if j.Status == JobPending {
		j.Status = JobRunning
	}
}

// Wait blocks until the entire job has finished (or context expires).
func (j *JobManager) Wait(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			j.mu.Lock()
			status := j.Status
			j.mu.Unlock()
			
			if status == JobCompleted || status == JobFailed {
				if status == JobFailed {
					return fmt.Errorf("job %s failed during execution", j.JobID)
				}
				return nil
			}
		}
	}
}
