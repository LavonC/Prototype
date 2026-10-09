package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// SQLiteDB implements the DB interface using SQLite for local worker storage.
type SQLiteDB struct {
	db            *sql.DB
	migrationsDir string
}

// NewSQLiteDB creates a new SQLite database connection.
func NewSQLiteDB(dsn string, migrationsDir string) (*SQLiteDB, error) {
	// Ensure parent directory exists.
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return &SQLiteDB{
		db:            db,
		migrationsDir: migrationsDir,
	}, nil
}

// Close closes the database connection.
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// Migrate runs all SQL migration files in order.
func (s *SQLiteDB) Migrate(ctx context.Context) error {
	// Create migrations tracking table.
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read migration files.
	entries, err := os.ReadDir(s.migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", s.migrationsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := entry.Name()

		// Check if already applied.
		var count int
		err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}

		// Read and execute migration.
		content, err := os.ReadFile(filepath.Join(s.migrationsDir, version))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", version, err)
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}

		log.Info().Str("migration", version).Msg("Applied database migration")
	}

	return nil
}

// UpsertWorker creates or updates a worker record.
func (s *SQLiteDB) UpsertWorker(ctx context.Context, worker *WorkerRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workers (worker_id, online, cpu_cores, cpu_model, total_memory_mb, total_disk_mb,
			gpu_count, os, arch, supported_runtimes, credibility, version, last_heartbeat, registered_at,
			total_tasks_run, total_cpu_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(worker_id) DO UPDATE SET
			online = excluded.online,
			cpu_cores = excluded.cpu_cores,
			cpu_model = excluded.cpu_model,
			total_memory_mb = excluded.total_memory_mb,
			total_disk_mb = excluded.total_disk_mb,
			gpu_count = excluded.gpu_count,
			os = excluded.os,
			arch = excluded.arch,
			supported_runtimes = excluded.supported_runtimes,
			version = excluded.version,
			last_heartbeat = excluded.last_heartbeat
	`, worker.WorkerID, worker.Online, worker.CPUCores, worker.CPUModel,
		worker.TotalMemoryMB, worker.TotalDiskMB, worker.GPUCount,
		worker.OS, worker.Arch, worker.SupportedRuntimes, worker.Credibility,
		worker.Version, worker.LastHeartbeat, worker.RegisteredAt,
		worker.TotalTasksRun, worker.TotalCPUSeconds)
	return err
}

// GetWorker retrieves a worker by ID.
func (s *SQLiteDB) GetWorker(ctx context.Context, workerID string) (*WorkerRecord, error) {
	w := &WorkerRecord{}
	err := s.db.QueryRowContext(ctx, `
		SELECT worker_id, online, cpu_cores, cpu_model, total_memory_mb, total_disk_mb,
			gpu_count, os, arch, supported_runtimes, credibility, version,
			last_heartbeat, registered_at, total_tasks_run, total_cpu_seconds
		FROM workers WHERE worker_id = ?
	`, workerID).Scan(
		&w.WorkerID, &w.Online, &w.CPUCores, &w.CPUModel,
		&w.TotalMemoryMB, &w.TotalDiskMB, &w.GPUCount,
		&w.OS, &w.Arch, &w.SupportedRuntimes, &w.Credibility,
		&w.Version, &w.LastHeartbeat, &w.RegisteredAt,
		&w.TotalTasksRun, &w.TotalCPUSeconds)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// ListWorkers lists workers matching the given filter.
func (s *SQLiteDB) ListWorkers(ctx context.Context, filter WorkerFilter) ([]*WorkerRecord, error) {
	query := "SELECT worker_id, online, cpu_cores, cpu_model, total_memory_mb, total_disk_mb, gpu_count, os, arch, supported_runtimes, credibility, version, last_heartbeat, registered_at, total_tasks_run, total_cpu_seconds FROM workers WHERE 1=1"
	var args []interface{}

	if filter.OnlineOnly {
		query += " AND online = ?"
		args = append(args, true)
	}
	if filter.MinCPUCores > 0 {
		query += " AND cpu_cores >= ?"
		args = append(args, filter.MinCPUCores)
	}
	if filter.MinMemoryMB > 0 {
		query += " AND total_memory_mb >= ?"
		args = append(args, filter.MinMemoryMB)
	}
	if filter.RequiresGPU {
		query += " AND gpu_count > 0"
	}
	if filter.RuntimeImage != "" {
		query += " AND supported_runtimes LIKE ?"
		args = append(args, "%"+filter.RuntimeImage+"%")
	}
	if filter.MinCredibility > 0 {
		query += " AND credibility >= ?"
		args = append(args, filter.MinCredibility)
	}

	query += " ORDER BY credibility DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*WorkerRecord
	for rows.Next() {
		w := &WorkerRecord{}
		if err := rows.Scan(&w.WorkerID, &w.Online, &w.CPUCores, &w.CPUModel,
			&w.TotalMemoryMB, &w.TotalDiskMB, &w.GPUCount,
			&w.OS, &w.Arch, &w.SupportedRuntimes, &w.Credibility,
			&w.Version, &w.LastHeartbeat, &w.RegisteredAt,
			&w.TotalTasksRun, &w.TotalCPUSeconds); err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	return workers, rows.Err()
}

// UpdateWorkerStatus updates a worker's online status and last heartbeat.
func (s *SQLiteDB) UpdateWorkerStatus(ctx context.Context, workerID string, online bool, lastHeartbeat time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE workers SET online = ?, last_heartbeat = ? WHERE worker_id = ?",
		online, lastHeartbeat, workerID)
	return err
}

// DeleteWorker removes a worker record.
func (s *SQLiteDB) DeleteWorker(ctx context.Context, workerID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM workers WHERE worker_id = ?", workerID)
	return err
}

// CreateTask stores a new task record.
func (s *SQLiteDB) CreateTask(ctx context.Context, task *TaskRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (task_id, job_id, requester_id, worker_id, status, entrypoint,
			runtime_image, exit_code, error_message, priority, attempt, max_attempts,
			capsule_sha256, result_sha256, artifact_path, artifact_size, duration_ms,
			cpu_user_seconds, peak_memory_mb, is_spot_check, expected_hash, created_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.TaskID, task.JobID, task.RequesterID, task.WorkerID, task.Status,
		task.Entrypoint, task.RuntimeImage, task.ExitCode, task.ErrorMessage,
		task.Priority, task.Attempt, task.MaxAttempts, task.CapsuleSHA256,
		task.ResultSHA256, task.ArtifactPath, task.ArtifactSize, task.DurationMs,
		task.CPUUserSeconds, task.PeakMemoryMB, task.IsSpotCheck, task.ExpectedHash,
		task.CreatedAt, task.StartedAt, task.CompletedAt)
	return err
}

// GetTask retrieves a task by ID.
func (s *SQLiteDB) GetTask(ctx context.Context, taskID string) (*TaskRecord, error) {
	t := &TaskRecord{}
	err := s.db.QueryRowContext(ctx, `
		SELECT task_id, job_id, requester_id, worker_id, status, entrypoint, runtime_image,
			exit_code, error_message, priority, attempt, max_attempts, capsule_sha256,
			result_sha256, artifact_path, artifact_size, duration_ms, cpu_user_seconds,
			peak_memory_mb, is_spot_check, expected_hash, created_at, started_at, completed_at
		FROM tasks WHERE task_id = ?
	`, taskID).Scan(
		&t.TaskID, &t.JobID, &t.RequesterID, &t.WorkerID, &t.Status,
		&t.Entrypoint, &t.RuntimeImage, &t.ExitCode, &t.ErrorMessage,
		&t.Priority, &t.Attempt, &t.MaxAttempts, &t.CapsuleSHA256,
		&t.ResultSHA256, &t.ArtifactPath, &t.ArtifactSize, &t.DurationMs,
		&t.CPUUserSeconds, &t.PeakMemoryMB, &t.IsSpotCheck, &t.ExpectedHash,
		&t.CreatedAt, &t.StartedAt, &t.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// UpdateTaskStatus updates a task's status and optional metadata.
func (s *SQLiteDB) UpdateTaskStatus(ctx context.Context, taskID string, status string, metadata map[string]interface{}) error {
	setClauses := []string{"status = ?"}
	args := []interface{}{status}

	for key, val := range metadata {
		setClauses = append(setClauses, key+" = ?")
		args = append(args, val)
	}

	args = append(args, taskID)
	query := fmt.Sprintf("UPDATE tasks SET %s WHERE task_id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ListTasks lists tasks matching the given filter.
func (s *SQLiteDB) ListTasks(ctx context.Context, filter TaskFilter) ([]*TaskRecord, error) {
	query := `SELECT task_id, job_id, requester_id, worker_id, status, entrypoint, runtime_image,
		exit_code, error_message, priority, attempt, max_attempts, capsule_sha256,
		result_sha256, artifact_path, artifact_size, duration_ms, cpu_user_seconds,
		peak_memory_mb, is_spot_check, expected_hash, created_at, started_at, completed_at
		FROM tasks WHERE 1=1`
	var args []interface{}

	if filter.JobID != "" {
		query += " AND job_id = ?"
		args = append(args, filter.JobID)
	}
	if filter.WorkerID != "" {
		query += " AND worker_id = ?"
		args = append(args, filter.WorkerID)
	}
	if filter.RequesterID != "" {
		query += " AND requester_id = ?"
		args = append(args, filter.RequesterID)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskRecord
	for rows.Next() {
		t := &TaskRecord{}
		if err := rows.Scan(&t.TaskID, &t.JobID, &t.RequesterID, &t.WorkerID,
			&t.Status, &t.Entrypoint, &t.RuntimeImage, &t.ExitCode, &t.ErrorMessage,
			&t.Priority, &t.Attempt, &t.MaxAttempts, &t.CapsuleSHA256,
			&t.ResultSHA256, &t.ArtifactPath, &t.ArtifactSize, &t.DurationMs,
			&t.CPUUserSeconds, &t.PeakMemoryMB, &t.IsSpotCheck, &t.ExpectedHash,
			&t.CreatedAt, &t.StartedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetPendingTasks returns tasks waiting for assignment, ordered by priority.
func (s *SQLiteDB) GetPendingTasks(ctx context.Context, limit int) ([]*TaskRecord, error) {
	return s.ListTasks(ctx, TaskFilter{Status: "PENDING", Limit: limit})
}

// CreateJob stores a new job record.
func (s *SQLiteDB) CreateJob(ctx context.Context, job *JobRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO jobs (job_id, requester_id, status, total_tasks, completed_tasks,
			failed_tasks, entrypoint, reduce_command, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.JobID, job.RequesterID, job.Status, job.TotalTasks,
		job.CompletedTasks, job.FailedTasks, job.Entrypoint,
		job.ReduceCommand, job.CreatedAt, job.CompletedAt)
	return err
}

// GetJob retrieves a job by ID.
func (s *SQLiteDB) GetJob(ctx context.Context, jobID string) (*JobRecord, error) {
	j := &JobRecord{}
	err := s.db.QueryRowContext(ctx, `
		SELECT job_id, requester_id, status, total_tasks, completed_tasks,
			failed_tasks, entrypoint, reduce_command, created_at, completed_at
		FROM jobs WHERE job_id = ?
	`, jobID).Scan(&j.JobID, &j.RequesterID, &j.Status, &j.TotalTasks,
		&j.CompletedTasks, &j.FailedTasks, &j.Entrypoint,
		&j.ReduceCommand, &j.CreatedAt, &j.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return j, err
}

// UpdateJobStatus updates a job's status.
func (s *SQLiteDB) UpdateJobStatus(ctx context.Context, jobID string, status string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE jobs SET status = ? WHERE job_id = ?", status, jobID)
	return err
}

// GetCredibility retrieves a worker's credibility score.
func (s *SQLiteDB) GetCredibility(ctx context.Context, workerID string) (float64, error) {
	var score float64
	err := s.db.QueryRowContext(ctx, "SELECT credibility FROM workers WHERE worker_id = ?", workerID).Scan(&score)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return score, err
}

// UpdateCredibility sets a worker's credibility score.
func (s *SQLiteDB) UpdateCredibility(ctx context.Context, workerID string, score float64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE workers SET credibility = ? WHERE worker_id = ?", score, workerID)
	return err
}

// RecordVerification logs a verification event.
func (s *SQLiteDB) RecordVerification(ctx context.Context, record *VerificationRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO verification_records (id, task_id, worker_id, expected_hash, actual_hash,
			passed, is_spot_check, old_credibility, new_credibility, verified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.TaskID, record.WorkerID, record.ExpectedHash,
		record.ActualHash, record.Passed, record.IsSpotCheck,
		record.OldCredibility, record.NewCredibility, record.VerifiedAt)
	return err
}

// GetRecentTasksByWorker retrieves tasks completed by a worker within a time window.
func (s *SQLiteDB) GetRecentTasksByWorker(ctx context.Context, workerID string, since time.Time) ([]*TaskRecord, error) {
	return s.ListTasks(ctx, TaskFilter{WorkerID: workerID, Status: "COMPLETED"})
}
