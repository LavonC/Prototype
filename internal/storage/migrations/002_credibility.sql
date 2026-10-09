-- UniCompute Database Schema: Credibility & Byzantine Verification
-- Migration 002: Tables for Sarmenta sabotage-tolerance model

CREATE TABLE IF NOT EXISTS verification_records (
    id              TEXT PRIMARY KEY,
    task_id         TEXT NOT NULL REFERENCES tasks(task_id),
    worker_id       TEXT NOT NULL REFERENCES workers(worker_id),
    expected_hash   TEXT NOT NULL DEFAULT '',
    actual_hash     TEXT NOT NULL DEFAULT '',
    passed          BOOLEAN NOT NULL DEFAULT FALSE,
    is_spot_check   BOOLEAN NOT NULL DEFAULT FALSE,
    old_credibility REAL NOT NULL DEFAULT 0.0,
    new_credibility REAL NOT NULL DEFAULT 0.0,
    verified_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS credibility_history (
    id          TEXT PRIMARY KEY,
    worker_id   TEXT NOT NULL REFERENCES workers(worker_id),
    score       REAL NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    task_id     TEXT,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS spot_check_tasks (
    id              TEXT PRIMARY KEY,
    runtime_image   TEXT NOT NULL,
    entrypoint      TEXT NOT NULL,
    capsule_sha256  TEXT NOT NULL,
    expected_hash   TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for verification queries.
CREATE INDEX IF NOT EXISTS idx_verification_worker ON verification_records(worker_id);
CREATE INDEX IF NOT EXISTS idx_verification_task ON verification_records(task_id);
CREATE INDEX IF NOT EXISTS idx_verification_passed ON verification_records(passed);
CREATE INDEX IF NOT EXISTS idx_credibility_worker ON credibility_history(worker_id);
CREATE INDEX IF NOT EXISTS idx_credibility_time ON credibility_history(recorded_at);
