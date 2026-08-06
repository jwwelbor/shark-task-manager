-- T-E40-F02-002's rejection-crosscheck DB fixture (AC-19, ADR-F02-09). Uses
-- the real entity_history/work_sessions column shape (internal/db/db.go
-- :219-230, :1356-1370) -- not the full production schema, just the columns
-- collect-run.sh's crosscheck query touches: tasks(id, key) for the
-- entity_key -> entity_id resolution ADR-F02-09 requires, entity_history
-- (entity_type, entity_id, to_status, changed_at) for the backward-
-- transition count, and work_sessions (entity_type, entity_key, outcome)
-- for the supplementary work_session_outcomes count.
--
-- entity_history rows below are ordered by changed_at and walk:
--   in_development (new) -> in_qa (new) -> in_development (REPEAT of row 1,
--   backward #1) -> in_qa (REPEAT of row 2, backward #2)
-- giving entity_history_backward_transitions=2, deliberately disagreeing
-- with the RunResult-inferred rework_loops=1 that
-- run/stdout.json's stages[] implies (one genuine re-entry: stage 3's
-- "in_development" repeats stage 1's, attributed to the "in_qa" gate at
-- stage 2; stage 4 moves to a never-before-seen "ready_for_approval").
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE
);
CREATE TABLE entity_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL,
    changed_at DATETIME NOT NULL
);
CREATE TABLE work_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_key TEXT,
    outcome TEXT,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP
);

INSERT INTO tasks (id, key) VALUES (42, 'T-BENCH-DEMO-020');

INSERT INTO entity_history (entity_type, entity_id, from_status, to_status, changed_at) VALUES
    ('task', 42, NULL, 'in_development', '2026-08-06T11:00:00Z'),
    ('task', 42, 'in_development', 'in_qa', '2026-08-06T11:01:30Z'),
    ('task', 42, 'in_qa', 'in_development', '2026-08-06T11:03:00Z'),
    ('task', 42, 'in_development', 'in_qa', '2026-08-06T11:04:30Z');

INSERT INTO work_sessions (entity_type, entity_key, outcome, started_at, ended_at) VALUES
    ('task', 'T-BENCH-DEMO-020', 'blocked', '2026-08-06T11:01:00Z', '2026-08-06T11:03:00Z'),
    ('task', 'T-BENCH-DEMO-020', 'completed', '2026-08-06T11:04:30Z', '2026-08-06T11:05:00Z');
