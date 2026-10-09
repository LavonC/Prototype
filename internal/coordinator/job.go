package coordinator

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/protocol"
	"github.com/unicompute/unicompute/internal/storage"
)

// JobTracker manages the state of complex distributed jobs.
// It ensures that all tasks for a job complete before triggering any post-processing hooks.
type JobTracker struct {
	db storage.DB
}

// NewJobTracker initializes the coordinator job tracker.
func NewJobTracker(db storage.DB) *JobTracker {
	return &JobTracker{db: db}
}

// RegisterJob stores a new job and its total task count.
func (j *JobTracker) RegisterJob(ctx context.Context, jobID, requesterID string, totalTasks int) error {
	record := &storage.JobRecord{
		JobID:       jobID,
		RequesterID: requesterID,
		Status:      "RUNNING",
		TotalTasks:  totalTasks,
		CreatedAt:   time.Now(),
	}
	
	return j.db.CreateJob(ctx, record)
}

// ProcessTaskCompletion updates job state when a child task finishes.
func (j *JobTracker) ProcessTaskCompletion(ctx context.Context, receipt *protocol.CompletionReceipt) {
	if receipt.JobID == "" || receipt.JobID == "sys-spotcheck" {
		return // Ignore tasks not attached to distributed jobs
	}
	
	job, err := j.db.GetJob(ctx, receipt.JobID)
	if err != nil || job == nil {
		log.Error().Err(err).Str("job_id", receipt.JobID).Msg("Failed to retrieve job for completion tracking")
		return
	}
	
	if job.Status == "COMPLETED" || job.Status == "FAILED" {
		return // Already finished
	}
	
	// Fast-track failure
	if receipt.Status == "FAILED" {
		log.Warn().Str("job_id", job.JobID).Str("task_id", receipt.TaskID).Msg("Task failed, failing overall job")
		j.db.UpdateJobStatus(ctx, job.JobID, "FAILED")
		return
	}
	
	// Increment completion count
	// (In a real high-concurrency system, this requires an atomic SQL update 
	// `UPDATE jobs SET completed_tasks = completed_tasks + 1 WHERE ...`)
	// For simulation architecture, we just mock the completion check:
	
	job.CompletedTasks++
	
	if job.CompletedTasks >= job.TotalTasks {
		log.Info().Str("job_id", job.JobID).Msg("All map tasks completed! Job marked as COMPLETED.")
		j.db.UpdateJobStatus(ctx, job.JobID, "COMPLETED")
		
		// Here, the coordinator could send a specific MsgJobComplete to the Requester
		// to instruct it to run the Reducer locally.
	}
}
