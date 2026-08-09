-- Minimal, task-001-scoped scratch DB fixture: just enough columns for the
-- timeout DB-status fallback's `SELECT status FROM tasks WHERE key = ?`
-- (uat-plan.md UAT-05). Not the full production schema (internal/db/db.go) --
-- T-E40-F02-002's own fixture reuses the real entity_history/work_sessions
-- shape for the rejection crosscheck, which this lookup does not need.
CREATE TABLE tasks (
    key TEXT PRIMARY KEY,
    status TEXT NOT NULL
);
INSERT INTO tasks (key, status) VALUES ('T-BENCH-DEMO-011', 'in_development');
