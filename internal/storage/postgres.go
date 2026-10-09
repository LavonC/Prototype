package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// PostgresDB implements the DB interface using PostgreSQL for the cloud coordinator.
type PostgresDB struct {
	db            *sql.DB
	migrationsDir string
}

// NewPostgresDB creates a new PostgreSQL database connection.
func NewPostgresDB(dsn string, migrationsDir string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	return &PostgresDB{
		db:            db,
		migrationsDir: migrationsDir,
	}, nil
}

// Close closes the database connection.
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// Migrate runs all SQL migration files in order (Postgres dialect).
func (p *PostgresDB) Migrate(ctx context.Context) error {
	// Create migrations tracking table
	_, err := p.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	entries, err := os.ReadDir(p.migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := entry.Name()
		var count int
		err := p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(filepath.Join(p.migrationsDir, version))
		if err != nil {
			return err
		}

		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
		log.Info().Str("migration", version).Msg("Applied PostgreSQL migration")
	}

	return nil
}

// UpsertWorker creates or updates a worker record.
func (p *PostgresDB) UpsertWorker(ctx context.Context, worker *WorkerRecord) error {
	query := `
		INSERT INTO workers (worker_id, online, cpu_cores, cpu_model, total_memory_mb, total_disk_mb,
			gpu_count, os, arch, supported_runtimes, credibility, version, last_heartbeat, registered_at,
			total_tasks_run, total_cpu_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT(worker_id) DO UPDATE SET
			online = EXCLUDED.online,
			cpu_cores = EXCLUDED.cpu_cores,
			cpu_model = EXCLUDED.cpu_model,
			total_memory_mb = EXCLUDED.total_memory_mb,
			total_disk_mb = EXCLUDED.total_disk_mb,
			gpu_count = EXCLUDED.gpu_count,
			os = EXCLUDED.os,
			arch = EXCLUDED.arch,
			supported_runtimes = EXCLUDED.supported_runtimes,
			version = EXCLUDED.version,
			last_heartbeat = EXCLUDED.last_heartbeat
	`
	_, err := p.db.ExecContext(ctx, query,
		worker.WorkerID, worker.Online, worker.CPUCores, worker.CPUModel,
		worker.TotalMemoryMB, worker.TotalDiskMB, worker.GPUCount,
		worker.OS, worker.Arch, worker.SupportedRuntimes, worker.Credibility,
		worker.Version, worker.LastHeartbeat, worker.RegisteredAt,
		worker.TotalTasksRun, worker.TotalCPUSeconds)
	return err
}

func (p *PostgresDB) GetWorker(ctx context.Context, workerID string) (*WorkerRecord, error) {
	// Implemented similarly to SQLite but using $1 placeholders
	return nil, nil // Placeholder for brevity
}

func (p *PostgresDB) ListWorkers(ctx context.Context, filter WorkerFilter) ([]*WorkerRecord, error) {
	return nil, nil // Placeholder for brevity
}

func (p *PostgresDB) UpdateWorkerStatus(ctx context.Context, workerID string, online bool, lastHeartbeat time.Time) error {
	_, err := p.db.ExecContext(ctx, "UPDATE workers SET online = $1, last_heartbeat = $2 WHERE worker_id = $3", online, lastHeartbeat, workerID)
	return err
}

func (p *PostgresDB) DeleteWorker(ctx context.Context, workerID string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM workers WHERE worker_id = $1", workerID)
	return err
}

func (p *PostgresDB) CreateTask(ctx context.Context, task *TaskRecord) error {
	// Uses $1, $2, etc placeholders
	return nil // Placeholder for brevity
}

func (p *PostgresDB) GetTask(ctx context.Context, taskID string) (*TaskRecord, error) {
	return nil, nil // Placeholder
}

func (p *PostgresDB) UpdateTaskStatus(ctx context.Context, taskID string, status string, metadata map[string]interface{}) error {
	return nil // Placeholder
}

func (p *PostgresDB) ListTasks(ctx context.Context, filter TaskFilter) ([]*TaskRecord, error) {
	return nil, nil // Placeholder
}

func (p *PostgresDB) GetPendingTasks(ctx context.Context, limit int) ([]*TaskRecord, error) {
	return nil, nil // Placeholder
}

func (p *PostgresDB) CreateJob(ctx context.Context, job *JobRecord) error {
	return nil // Placeholder
}

func (p *PostgresDB) GetJob(ctx context.Context, jobID string) (*JobRecord, error) {
	return nil, nil // Placeholder
}

func (p *PostgresDB) UpdateJobStatus(ctx context.Context, jobID string, status string) error {
	return nil // Placeholder
}

func (p *PostgresDB) GetCredibility(ctx context.Context, workerID string) (float64, error) {
	return 0, nil // Placeholder
}

func (p *PostgresDB) UpdateCredibility(ctx context.Context, workerID string, score float64) error {
	return nil // Placeholder
}

func (p *PostgresDB) RecordVerification(ctx context.Context, record *VerificationRecord) error {
	return nil // Placeholder
}

func (p *PostgresDB) GetRecentTasksByWorker(ctx context.Context, workerID string, since time.Time) ([]*TaskRecord, error) {
	return nil, nil // Placeholder
}
