-- UniCompute Database Schema: Initial Setup
-- Migration 001: Core tables for workers, tasks, jobs

CREATE TABLE IF NOT EXISTS workers (
    worker_id         TEXT PRIMARY KEY,
    online            BOOLEAN NOT NULL DEFAULT FALSE,
    cpu_cores         INTEGER NOT NULL DEFAULT 0,
    cpu_model         TEXT NOT NULL DEFAULT '',
    total_memory_mb   INTEGER NOT NULL DEFAULT 0,
    total_disk_mb     INTEGER NOT NULL DEFAULT 0,
    gpu_count         INTEGER NOT NULL DEFAULT 0,
    os                TEXT NOT NULL DEFAULT '',
    arch              TEXT NOT NULL DEFAULT '',
    supported_runtimes TEXT NOT NULL DEFAULT '',
    credibility       REAL NOT NULL DEFAULT 0.5,
    version           TEXT NOT NULL DEFAULT '',
    last_heartbeat    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    registered_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    total_tasks_run   INTEGER NOT NULL DEFAULT 0,
    total_cpu_seconds REAL NOT NULL DEFAULT 0.0
);

CREATE TABLE IF NOT EXISTS jobs (
    job_id          TEXT PRIMARY KEY,
    requester_id    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    total_tasks     INTEGER NOT NULL DEFAULT 0,
    completed_tasks INTEGER NOT NULL DEFAULT 0,
    failed_tasks    INTEGER NOT NULL DEFAULT 0,
    entrypoint      TEXT NOT NULL DEFAULT '',
    reduce_command  TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at    TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tasks (
    task_id          TEXT PRIMARY KEY,
    job_id           TEXT NOT NULL REFERENCES jobs(job_id),
    requester_id     TEXT NOT NULL,
    worker_id        TEXT,
    status           TEXT NOT NULL DEFAULT 'PENDING',
    entrypoint       TEXT NOT NULL,
    runtime_image    TEXT NOT NULL DEFAULT '',
    exit_code        INTEGER NOT NULL DEFAULT -1,
    error_message    TEXT NOT NULL DEFAULT '',
    priority         INTEGER NOT NULL DEFAULT 0,
    attempt          INTEGER NOT NULL DEFAULT 0,
    max_attempts     INTEGER NOT NULL DEFAULT 3,
    capsule_sha256   TEXT NOT NULL DEFAULT '',
    result_sha256    TEXT NOT NULL DEFAULT '',
    artifact_path    TEXT NOT NULL DEFAULT '',
    artifact_size    INTEGER NOT NULL DEFAULT 0,
    duration_ms      INTEGER NOT NULL DEFAULT 0,
    cpu_user_seconds REAL NOT NULL DEFAULT 0.0,
    peak_memory_mb   REAL NOT NULL DEFAULT 0.0,
    is_spot_check    BOOLEAN NOT NULL DEFAULT FALSE,
    expected_hash    TEXT,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at       TIMESTAMP,
    completed_at     TIMESTAMP
);

CREATE TABLE IF NOT EXISTS verification_records (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    worker_id TEXT NOT NULL REFERENCES workers(worker_id) ON DELETE CASCADE,
    expected_hash TEXT,
    actual_hash TEXT,
    passed BOOLEAN NOT NULL,
    is_spot_check BOOLEAN NOT NULL,
    old_credibility REAL NOT NULL,
    new_credibility REAL NOT NULL,
    verified_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries.
CREATE INDEX IF NOT EXISTS idx_tasks_job_id ON tasks(job_id);
CREATE INDEX IF NOT EXISTS idx_tasks_worker_id ON tasks(worker_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority DESC);
CREATE INDEX IF NOT EXISTS idx_workers_online ON workers(online);
CREATE INDEX IF NOT EXISTS idx_workers_credibility ON workers(credibility);
