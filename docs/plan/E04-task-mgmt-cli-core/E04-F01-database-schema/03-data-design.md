# Data Design: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: db-admin

## Purpose

This document provides the complete database schema specification for the shark task management system. It defines all tables, columns, constraints, indexes, and data relationships in detail. This schema serves as the single source of truth for all project state (epics, features, tasks, history).

---

## Database Technology

**Engine**: SQLite 3.35+

**Rationale**:
- Embedded database (no separate server process)
- Zero configuration (single file: `project.db`)
- ACID-compliant transactions
- Cross-platform portable
- Sufficient performance for 10,000+ tasks
- Built-in JSON support (JSON1 extension)
- Write-Ahead Logging (WAL) for concurrency

**File Location**: `project.db` in project root

**Required SQLite Configuration**:
```sql
-- Enable foreign key constraints (not default)
PRAGMA foreign_keys = ON;

-- Use WAL mode for better concurrency
PRAGMA journal_mode = WAL;

-- Verify database integrity on startup
PRAGMA integrity_check;

-- Query timeout for writes (5 seconds)
PRAGMA busy_timeout = 5000;
```

---

## Entity Relationship Diagram

```
┌──────────────────┐
│      epics       │
│                  │
│ PK: id           │
│ UK: key          │  ONE
│     title        │    │
│     description  │    │
│     status       │    │
│     priority     │    │
│     business_val │    │
│     created_at   │    │
│     updated_at   │    │
└──────────────────┘    │
                        │
                        │ 1:N (CASCADE DELETE)
                        │
                        ▼
                  ┌──────────────────┐
                  │    features      │
                  │                  │
                  │ PK: id           │
                  │ FK: epic_id      │ ONE
                  │ UK: key          │   │
                  │     title        │   │
                  │     description  │   │
                  │     status       │   │
                  │     progress_pct │   │
                  │     created_at   │   │
                  │     updated_at   │   │
                  └──────────────────┘   │
                                         │
                                         │ 1:N (CASCADE DELETE)
                                         │
                                         ▼
                                   ┌──────────────────┐
                                   │      tasks       │
                                   │                  │
                                   │ PK: id           │
                                   │ FK: feature_id   │ ONE
                                   │ UK: key          │   │
                                   │     title        │   │
                                   │     description  │   │
                                   │     status       │   │
                                   │     agent_type   │   │
                                   │     priority     │   │
                                   │     depends_on   │   │
                                   │     assigned_agt │   │
                                   │     file_path    │   │
                                   │     blocked_rsn  │   │
                                   │     created_at   │   │
                                   │     started_at   │   │
                                   │     completed_at │   │
                                   │     blocked_at   │   │
                                   │     updated_at   │   │
                                   └──────────────────┘   │
                                                          │
                                                          │ 1:N (CASCADE DELETE)
                                                          │
                                                          ▼
                                                    ┌──────────────────┐
                                                    │  task_history    │
                                                    │                  │
                                                    │ PK: id           │
                                                    │ FK: task_id      │
                                                    │     old_status   │
                                                    │     new_status   │
                                                    │     agent        │
                                                    │     notes        │
                                                    │     timestamp    │
                                                    └──────────────────┘
```

**Legend**:
- PK = Primary Key
- FK = Foreign Key
- UK = Unique Key
- CASCADE DELETE = Parent deletion deletes children

---

## Table Schemas

### Table: epics

**Purpose**: Top-level project organization units (e.g., major features, initiatives)

**DDL**:
```sql
CREATE TABLE epics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived')),
    priority TEXT NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    business_value TEXT CHECK (business_value IN ('high', 'medium', 'low')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Unique constraint on key (already implicit from UNIQUE above)
CREATE UNIQUE INDEX idx_epics_key ON epics(key);

-- Trigger to auto-update updated_at
CREATE TRIGGER epics_updated_at
AFTER UPDATE ON epics
FOR EACH ROW
BEGIN
    UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Column Specifications**:

| Column | Type | Nullable | Default | Constraints | Description |
|--------|------|----------|---------|-------------|-------------|
| id | INTEGER | No | AUTO | PRIMARY KEY | Auto-incrementing surrogate key |
| key | TEXT | No | - | UNIQUE, CHECK format | Epic identifier (e.g., `E04`) |
| title | TEXT | No | - | - | Epic title (max ~1000 chars) |
| description | TEXT | Yes | NULL | - | Epic description (markdown, unlimited length) |
| status | TEXT | No | - | CHECK enum | One of: `draft`, `active`, `completed`, `archived` |
| priority | TEXT | No | - | CHECK enum | One of: `high`, `medium`, `low` |
| business_value | TEXT | Yes | NULL | CHECK enum | One of: `high`, `medium`, `low`, or NULL |
| created_at | TIMESTAMP | No | CURRENT_TIMESTAMP | - | UTC timestamp of creation |
| updated_at | TIMESTAMP | No | CURRENT_TIMESTAMP | Auto-update | UTC timestamp of last modification |

**Key Format Constraint** (enforced at application level):
- Pattern: `^E\d{2}$`
- Examples: `E01`, `E04`, `E99`

**Status Enum Values**:
- `draft`: Epic is being planned
- `active`: Epic is in progress
- `completed`: Epic is finished
- `archived`: Epic is completed and no longer active

**Priority Enum Values**:
- `high`: Urgent, high business impact
- `medium`: Standard priority
- `low`: Nice to have, low urgency

**Indexes**:
- `idx_epics_key` (UNIQUE): Fast lookup by epic key

**Sample Data**:
```sql
INSERT INTO epics (key, title, description, status, priority, business_value)
VALUES (
    'E04',
    'Task Management CLI - Core Functionality',
    'Foundational infrastructure for SQLite-backed task management...',
    'active',
    'high',
    'high'
);
```

---

### Table: features

**Purpose**: Mid-level units within epics (e.g., specific features, components)

**DDL**:
```sql
CREATE TABLE features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('draft', 'active', 'completed', 'archived')),
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (progress_pct >= 0.0 AND progress_pct <= 100.0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
);

-- Unique index on key
CREATE UNIQUE INDEX idx_features_key ON features(key);

-- Index on epic_id for fast epic → features queries
CREATE INDEX idx_features_epic_id ON features(epic_id);

-- Trigger to auto-update updated_at
CREATE TRIGGER features_updated_at
AFTER UPDATE ON features
FOR EACH ROW
BEGIN
    UPDATE features SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Column Specifications**:

| Column | Type | Nullable | Default | Constraints | Description |
|--------|------|----------|---------|-------------|-------------|
| id | INTEGER | No | AUTO | PRIMARY KEY | Auto-incrementing surrogate key |
| epic_id | INTEGER | No | - | FOREIGN KEY → epics(id) CASCADE | Parent epic |
| key | TEXT | No | - | UNIQUE, CHECK format | Feature identifier (e.g., `E04-F01`) |
| title | TEXT | No | - | - | Feature title |
| description | TEXT | Yes | NULL | - | Feature description (markdown) |
| status | TEXT | No | - | CHECK enum | One of: `draft`, `active`, `completed`, `archived` |
| progress_pct | REAL | No | 0.0 | CHECK 0-100 | Percentage of completed tasks (cached) |
| created_at | TIMESTAMP | No | CURRENT_TIMESTAMP | - | UTC timestamp of creation |
| updated_at | TIMESTAMP | No | CURRENT_TIMESTAMP | Auto-update | UTC timestamp of last modification |

**Key Format Constraint** (enforced at application level):
- Pattern: `^E\d{2}-F\d{2}$`
- Examples: `E04-F01`, `E04-F12`, `E99-F99`

**Foreign Key Behavior**:
- `ON DELETE CASCADE`: Deleting an epic deletes all its features
- `ON UPDATE RESTRICT`: Cannot update epic.id (should never happen with AUTOINCREMENT)

**Progress Calculation**:
- `progress_pct` is a **cached field** updated via application logic
- Formula: `(completed_tasks_count / total_tasks_count) × 100`
- 0.0 if feature has no tasks
- Recalculated when:
  - Task status changes to/from `completed`
  - Task is created or deleted

**Indexes**:
- `idx_features_key` (UNIQUE): Fast lookup by feature key
- `idx_features_epic_id`: Fast queries for features in an epic

**Sample Data**:
```sql
INSERT INTO features (epic_id, key, title, description, status, progress_pct)
VALUES (
    1,  -- epic_id (E04)
    'E04-F01',
    'Database Schema & Core Data Model',
    'SQLite database structure with epics, features, tasks...',
    'active',
    0.0
);
```

---

### Table: tasks

**Purpose**: Atomic work units within features (e.g., implementation tasks, bugs, improvements)

**DDL**:
```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feature_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived')),
    agent_type TEXT CHECK (agent_type IN ('frontend', 'backend', 'api', 'testing', 'devops', 'general')),
    priority INTEGER NOT NULL DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    depends_on TEXT,
    assigned_agent TEXT,
    file_path TEXT,
    blocked_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    blocked_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
);

-- Unique index on key
CREATE UNIQUE INDEX idx_tasks_key ON tasks(key);

-- Index on feature_id for fast feature → tasks queries
CREATE INDEX idx_tasks_feature_id ON tasks(feature_id);

-- Index on status for filtering
CREATE INDEX idx_tasks_status ON tasks(status);

-- Index on agent_type for filtering
CREATE INDEX idx_tasks_agent_type ON tasks(agent_type);

-- Composite index for common query: status + priority
CREATE INDEX idx_tasks_status_priority ON tasks(status, priority);

-- Trigger to auto-update updated_at
CREATE TRIGGER tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Column Specifications**:

| Column | Type | Nullable | Default | Constraints | Description |
|--------|------|----------|---------|-------------|-------------|
| id | INTEGER | No | AUTO | PRIMARY KEY | Auto-incrementing surrogate key |
| feature_id | INTEGER | No | - | FOREIGN KEY → features(id) CASCADE | Parent feature |
| key | TEXT | No | - | UNIQUE, CHECK format | Task identifier (e.g., `T-E04-F01-001`) |
| title | TEXT | No | - | - | Task title |
| description | TEXT | Yes | NULL | - | Task description (markdown) |
| status | TEXT | No | - | CHECK enum | Task status (see below) |
| agent_type | TEXT | Yes | NULL | CHECK enum | Agent specialization (see below) |
| priority | INTEGER | No | 5 | CHECK 1-10 | 1 = highest, 10 = lowest |
| depends_on | TEXT | Yes | NULL | Valid JSON array | JSON array of task keys |
| assigned_agent | TEXT | Yes | NULL | - | Free text agent identifier |
| file_path | TEXT | Yes | NULL | - | Absolute path to task markdown file |
| blocked_reason | TEXT | Yes | NULL | - | Only populated when status=blocked |
| created_at | TIMESTAMP | No | CURRENT_TIMESTAMP | - | UTC timestamp of creation |
| started_at | TIMESTAMP | Yes | NULL | - | When task started (status → in_progress) |
| completed_at | TIMESTAMP | Yes | NULL | - | When task completed |
| blocked_at | TIMESTAMP | Yes | NULL | - | When task blocked |
| updated_at | TIMESTAMP | No | CURRENT_TIMESTAMP | Auto-update | UTC timestamp of last modification |

**Key Format Constraint** (enforced at application level):
- Pattern: `^T-E\d{2}-F\d{2}-\d{3}$`
- Examples: `T-E04-F01-001`, `T-E04-F01-042`, `T-E99-F99-999`

**Status Enum Values**:
- `todo`: Task is planned but not started
- `in_progress`: Task is actively being worked on
- `blocked`: Task is blocked by external dependency or issue
- `ready_for_review`: Task implementation is complete, awaiting review
- `completed`: Task is finished and reviewed
- `archived`: Task is completed and no longer relevant

**Agent Type Enum Values**:
- `frontend`: UI/UX implementation tasks
- `backend`: Server-side logic, APIs
- `api`: API design and implementation
- `testing`: Test creation and execution
- `devops`: Infrastructure, deployment, CI/CD
- `general`: Tasks not specific to an agent type
- `NULL`: No agent type assigned

**Priority Values**:
- 1-3: High priority (urgent, blocking)
- 4-6: Medium priority (standard work)
- 7-10: Low priority (nice to have, backlog)

**Dependency Field (`depends_on`)**:
- Stored as JSON string: `'["T-E01-F01-001", "T-E01-F02-003"]'`
- Empty dependencies: `'[]'` or `NULL`
- Validation: Must be valid JSON array of strings
- **Not enforced as foreign keys** (allows forward references, cross-epic dependencies)
- Validation of referenced task existence is deferred to application logic (E04-F03)

**Timestamp Management**:
- `started_at`: Set when status changes to `in_progress` (first time)
- `completed_at`: Set when status changes to `completed`
- `blocked_at`: Set when status changes to `blocked`
- These timestamps are **not cleared** on status reversions (preserve history)

**Foreign Key Behavior**:
- `ON DELETE CASCADE`: Deleting a feature deletes all its tasks

**Indexes**:
- `idx_tasks_key` (UNIQUE): Fast lookup by task key
- `idx_tasks_feature_id`: Fast queries for tasks in a feature
- `idx_tasks_status`: Fast filtering by status
- `idx_tasks_agent_type`: Fast filtering by agent type
- `idx_tasks_status_priority`: Optimized for query: "get high-priority todo tasks"

**Sample Data**:
```sql
INSERT INTO tasks (
    feature_id, key, title, description, status, agent_type, priority,
    depends_on, assigned_agent, file_path
)
VALUES (
    1,  -- feature_id (E04-F01)
    'T-E04-F01-001',
    'Create SQLAlchemy ORM models',
    'Define Epic, Feature, Task, TaskHistory models with relationships...',
    'todo',
    'backend',
    3,
    '[]',  -- No dependencies
    NULL,
    '/home/user/project/docs/tasks/todo/T-E04-F01-001.md'
);
```

---

### Table: task_history

**Purpose**: Audit trail of task status changes

**DDL**:
```sql
CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    old_status TEXT,
    new_status TEXT NOT NULL,
    agent TEXT,
    notes TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Index on task_id for fast history lookup by task
CREATE INDEX idx_task_history_task_id ON task_history(task_id);

-- Index on timestamp for recent history queries
CREATE INDEX idx_task_history_timestamp ON task_history(timestamp DESC);
```

**Column Specifications**:

| Column | Type | Nullable | Default | Constraints | Description |
|--------|------|----------|---------|-------------|-------------|
| id | INTEGER | No | AUTO | PRIMARY KEY | Auto-incrementing surrogate key |
| task_id | INTEGER | No | - | FOREIGN KEY → tasks(id) CASCADE | Task that changed |
| old_status | TEXT | Yes | NULL | - | Previous status (NULL for task creation) |
| new_status | TEXT | No | - | - | New status value |
| agent | TEXT | Yes | NULL | - | Who made the change (user, agent name, etc.) |
| notes | TEXT | Yes | NULL | - | Optional change notes |
| timestamp | TIMESTAMP | No | CURRENT_TIMESTAMP | - | UTC timestamp of status change |

**Usage Patterns**:
- Record created automatically when task status changes
- `old_status = NULL` for initial task creation
- `agent` stores identifier of who made change (CLI user, agent name, etc.)
- `notes` can store reason for change, commit message, etc.

**Foreign Key Behavior**:
- `ON DELETE CASCADE`: Deleting a task deletes all its history

**Indexes**:
- `idx_task_history_task_id`: Fast queries for task history
- `idx_task_history_timestamp`: Fast queries for recent changes across all tasks

**Sample Data**:
```sql
-- Task creation
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (1, NULL, 'todo', 'system', 'Task created via shark task create');

-- Status change
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (1, 'todo', 'in_progress', 'claude-backend-specialist', 'Starting implementation');
```

---

## Data Relationships

### Epic → Features (1:N)

- One epic can have many features
- Each feature belongs to exactly one epic
- Deleting an epic **cascades delete** to all its features
- Querying features by epic uses index `idx_features_epic_id`

**Example Query**:
```sql
-- Get all features for epic E04
SELECT f.*
FROM features f
JOIN epics e ON f.epic_id = e.id
WHERE e.key = 'E04'
ORDER BY f.key ASC;
```

### Feature → Tasks (1:N)

- One feature can have many tasks
- Each task belongs to exactly one feature
- Deleting a feature **cascades delete** to all its tasks
- Querying tasks by feature uses index `idx_tasks_feature_id`

**Example Query**:
```sql
-- Get all tasks for feature E04-F01
SELECT t.*
FROM tasks t
JOIN features f ON t.feature_id = f.id
WHERE f.key = 'E04-F01'
ORDER BY t.key ASC;
```

### Task → Task History (1:N)

- One task can have many history entries
- Each history entry belongs to exactly one task
- Deleting a task **cascades delete** to all its history
- Querying history by task uses index `idx_task_history_task_id`

**Example Query**:
```sql
-- Get history for task T-E04-F01-001
SELECT h.*
FROM task_history h
JOIN tasks t ON h.task_id = t.id
WHERE t.key = 'T-E04-F01-001'
ORDER BY h.timestamp DESC;
```

### Epic → Tasks (1:N indirect)

- One epic has many tasks (via features)
- No direct foreign key (enforced via features table)
- Query requires JOIN through features

**Example Query**:
```sql
-- Get all tasks for epic E04
SELECT t.*
FROM tasks t
JOIN features f ON t.feature_id = f.id
JOIN epics e ON f.epic_id = e.id
WHERE e.key = 'E04'
ORDER BY t.key ASC;
```

### Task → Task Dependencies (N:N logical)

- Tasks can depend on other tasks (stored in `depends_on` JSON field)
- **Not enforced as database foreign keys** (intentional)
- Allows forward references (depend on task not yet created)
- Allows cross-epic dependencies
- Validation happens at application level

**Example**:
```sql
-- Task with dependencies
INSERT INTO tasks (feature_id, key, title, status, depends_on)
VALUES (
    1,
    'T-E04-F01-005',
    'Create integration tests',
    'todo',
    '["T-E04-F01-001", "T-E04-F01-002", "T-E04-F01-003"]'
);

-- Query tasks with specific dependency (using SQLite JSON1 extension)
SELECT *
FROM tasks
WHERE json_extract(depends_on, '$') LIKE '%T-E04-F01-001%';
```

---

## Constraints

### Primary Keys

All tables use auto-incrementing integer primary keys:
- `epics.id`
- `features.id`
- `tasks.id`
- `task_history.id`

**Rationale**: Stable, immutable, efficient for JOINs and foreign keys.

### Unique Constraints

Business keys are enforced as unique:
- `epics.key` (UNIQUE) - e.g., `E04`
- `features.key` (UNIQUE) - e.g., `E04-F01`
- `tasks.key` (UNIQUE) - e.g., `T-E04-F01-001`

**Rationale**: Business keys are the primary identifiers used in code and documentation.

### Foreign Keys

| Child Table | Column | Parent Table | Parent Column | On Delete | On Update |
|-------------|--------|--------------|---------------|-----------|-----------|
| features | epic_id | epics | id | CASCADE | RESTRICT |
| tasks | feature_id | features | id | CASCADE | RESTRICT |
| task_history | task_id | tasks | id | CASCADE | RESTRICT |

**CASCADE DELETE Rationale**:
- Deleting an epic should remove all related data (features, tasks, history)
- Maintains referential integrity automatically
- Prevents orphaned records

**RESTRICT UPDATE Rationale**:
- Primary keys never change (AUTOINCREMENT)
- Prevents accidental ID modification

### Check Constraints

**Enums** (enforced at database level):
```sql
-- epics.status
CHECK (status IN ('draft', 'active', 'completed', 'archived'))

-- epics.priority
CHECK (priority IN ('high', 'medium', 'low'))

-- epics.business_value
CHECK (business_value IN ('high', 'medium', 'low'))

-- features.status
CHECK (status IN ('draft', 'active', 'completed', 'archived'))

-- features.progress_pct
CHECK (progress_pct >= 0.0 AND progress_pct <= 100.0)

-- tasks.status
CHECK (status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived'))

-- tasks.agent_type
CHECK (agent_type IN ('frontend', 'backend', 'api', 'testing', 'devops', 'general'))

-- tasks.priority
CHECK (priority >= 1 AND priority <= 10)
```

**Range Constraints**:
- `features.progress_pct`: 0.0 to 100.0
- `tasks.priority`: 1 to 10

### Not Null Constraints

**Required Fields** (NOT NULL):
- All `id` fields (primary keys)
- All `key` fields (business keys)
- All `title` fields
- All `status` fields
- All `created_at`, `updated_at` fields
- `epics.priority`
- `features.epic_id`, `features.progress_pct`
- `tasks.feature_id`, `tasks.priority`
- `task_history.task_id`, `task_history.new_status`, `task_history.timestamp`

**Optional Fields** (nullable):
- All `description` fields
- `epics.business_value`
- `tasks.agent_type`, `tasks.depends_on`, `tasks.assigned_agent`, `tasks.file_path`, `tasks.blocked_reason`
- `tasks.started_at`, `tasks.completed_at`, `tasks.blocked_at`
- `task_history.old_status`, `task_history.agent`, `task_history.notes`

---

## Indexes

### Unique Indexes (Enforcing Uniqueness)

| Index Name | Table | Column(s) | Purpose |
|------------|-------|-----------|---------|
| idx_epics_key | epics | key | Enforce unique epic keys |
| idx_features_key | features | key | Enforce unique feature keys |
| idx_tasks_key | tasks | key | Enforce unique task keys |

### Non-Unique Indexes (Query Performance)

| Index Name | Table | Column(s) | Purpose |
|------------|-------|-----------|---------|
| idx_features_epic_id | features | epic_id | Fast epic → features queries |
| idx_tasks_feature_id | tasks | feature_id | Fast feature → tasks queries |
| idx_tasks_status | tasks | status | Filter tasks by status |
| idx_tasks_agent_type | tasks | agent_type | Filter tasks by agent type |
| idx_tasks_status_priority | tasks | status, priority | Combined filter (status + priority) |
| idx_task_history_task_id | task_history | task_id | Fast task → history queries |
| idx_task_history_timestamp | task_history | timestamp DESC | Recent history queries |

**Index Strategy**:
- Index all foreign keys for JOIN performance
- Index frequently filtered columns (status, agent_type)
- Composite index for common combined queries
- Descending index on timestamp for "recent activity" queries

---

## Triggers

### Auto-Update Timestamps

SQLite requires triggers for automatic `updated_at` modification:

**epics_updated_at**:
```sql
CREATE TRIGGER epics_updated_at
AFTER UPDATE ON epics
FOR EACH ROW
BEGIN
    UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**features_updated_at**:
```sql
CREATE TRIGGER features_updated_at
AFTER UPDATE ON features
FOR EACH ROW
BEGIN
    UPDATE features SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**tasks_updated_at**:
```sql
CREATE TRIGGER tasks_updated_at
AFTER UPDATE ON tasks
FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Note**: `task_history` doesn't need an update trigger (records are immutable).

---

## Data Integrity Rules

### Referential Integrity

1. **Epic Deletion**: Deleting epic cascades to features → tasks → task_history
2. **Feature Deletion**: Deleting feature cascades to tasks → task_history
3. **Task Deletion**: Deleting task cascades to task_history
4. **Foreign Key Validation**: Cannot create feature without valid epic_id
5. **Foreign Key Validation**: Cannot create task without valid feature_id

### Business Logic Integrity (Application-Enforced)

These rules are enforced by application code, not database constraints:

1. **Key Format Validation**:
   - Epic key: `^E\d{2}$`
   - Feature key: `^E##-F\d{2}$` (epic portion must match parent epic)
   - Task key: `^T-E##-F##-\d{3}$` (epic+feature portions must match parent feature)

2. **Progress Calculation**:
   - `features.progress_pct` must equal `(completed_tasks / total_tasks) × 100`
   - Recalculated when task status changes

3. **Status Transition Validation**:
   - Valid transitions (e.g., `todo` → `in_progress`, not `todo` → `completed`)
   - Defined in E04-F03 (Task Lifecycle Operations)

4. **Dependency Validation**:
   - Tasks in `depends_on` JSON array should exist
   - Circular dependencies should be prevented
   - Defined in E05-F02 (Dependency Management)

5. **Blocked Reason Enforcement**:
   - `tasks.blocked_reason` should only be set when `status = 'blocked'`
   - Application should clear `blocked_reason` when status changes away from `blocked`

6. **Timestamp Management**:
   - `started_at` set when status → `in_progress` (first time)
   - `completed_at` set when status → `completed`
   - `blocked_at` set when status → `blocked`
   - Timestamps not cleared on status reversions

---

## Sample Queries

### Create Operations

**Create Epic**:
```sql
INSERT INTO epics (key, title, description, status, priority, business_value)
VALUES ('E04', 'Task Management CLI', '...', 'active', 'high', 'high');
```

**Create Feature**:
```sql
INSERT INTO features (epic_id, key, title, description, status, progress_pct)
VALUES (1, 'E04-F01', 'Database Schema', '...', 'active', 0.0);
```

**Create Task**:
```sql
INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
VALUES (1, 'T-E04-F01-001', 'Create models', 'todo', 'backend', 3, '[]');
```

**Create History Entry**:
```sql
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (1, 'todo', 'in_progress', 'claude', 'Starting work');
```

### Read Operations

**Get Epic by Key**:
```sql
SELECT * FROM epics WHERE key = 'E04';
```

**Get Features for Epic**:
```sql
SELECT f.*
FROM features f
JOIN epics e ON f.epic_id = e.id
WHERE e.key = 'E04'
ORDER BY f.key ASC;
```

**Get Tasks for Feature**:
```sql
SELECT * FROM tasks WHERE feature_id = 1 ORDER BY key ASC;
```

**Get Tasks by Status**:
```sql
SELECT * FROM tasks WHERE status = 'todo' ORDER BY priority ASC, created_at DESC;
```

**Get High-Priority Backend Tasks**:
```sql
SELECT *
FROM tasks
WHERE status = 'todo' AND agent_type = 'backend' AND priority <= 3
ORDER BY priority ASC, created_at DESC;
```

**Get Task History**:
```sql
SELECT * FROM task_history WHERE task_id = 1 ORDER BY timestamp DESC;
```

**Get Recent Activity (Last 50 Changes)**:
```sql
SELECT h.*, t.key AS task_key, t.title
FROM task_history h
JOIN tasks t ON h.task_id = t.id
ORDER BY h.timestamp DESC
LIMIT 50;
```

### Update Operations

**Update Epic Status**:
```sql
UPDATE epics SET status = 'completed' WHERE id = 1;
-- Trigger automatically updates updated_at
```

**Update Feature Progress**:
```sql
UPDATE features SET progress_pct = 42.5 WHERE id = 1;
```

**Update Task Status**:
```sql
-- Update task
UPDATE tasks SET status = 'in_progress', started_at = CURRENT_TIMESTAMP WHERE id = 1;

-- Record history (should be atomic transaction)
INSERT INTO task_history (task_id, old_status, new_status, agent)
VALUES (1, 'todo', 'in_progress', 'claude');
```

**Block Task**:
```sql
UPDATE tasks
SET status = 'blocked', blocked_reason = 'Waiting for API design', blocked_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

### Delete Operations

**Delete Epic (Cascades)**:
```sql
DELETE FROM epics WHERE id = 1;
-- Automatically deletes: features, tasks, task_history
```

**Delete Feature (Cascades)**:
```sql
DELETE FROM features WHERE id = 1;
-- Automatically deletes: tasks, task_history
```

**Delete Task (Cascades)**:
```sql
DELETE FROM tasks WHERE id = 1;
-- Automatically deletes: task_history
```

### Aggregation Queries

**Calculate Feature Progress**:
```sql
SELECT
    f.id,
    f.key,
    COUNT(t.id) AS total_tasks,
    SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) AS completed_tasks,
    CASE
        WHEN COUNT(t.id) = 0 THEN 0.0
        ELSE (SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END) * 100.0 / COUNT(t.id))
    END AS progress_pct
FROM features f
LEFT JOIN tasks t ON f.id = t.feature_id
WHERE f.id = 1
GROUP BY f.id;
```

**Calculate Epic Progress**:
```sql
SELECT
    e.id,
    e.key,
    AVG(f.progress_pct) AS epic_progress_pct
FROM epics e
LEFT JOIN features f ON e.id = f.epic_id
WHERE e.id = 1
GROUP BY e.id;
```

**Count Tasks by Status**:
```sql
SELECT status, COUNT(*) AS task_count
FROM tasks
GROUP BY status
ORDER BY task_count DESC;
```

---

## Migration Strategy

### Initial Schema Migration

**Alembic Migration**: `001_initial_schema.py`

```python
"""Initial schema with epics, features, tasks, task_history

Revision ID: 001
Revises: None
Create Date: 2025-12-14
"""

def upgrade():
    # Create epics table
    op.create_table(
        'epics',
        sa.Column('id', sa.Integer(), primary_key=True, autoincrement=True),
        sa.Column('key', sa.String(10), nullable=False, unique=True),
        sa.Column('title', sa.Text(), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('status', sa.String(20), nullable=False),
        sa.Column('priority', sa.String(10), nullable=False),
        sa.Column('business_value', sa.String(10), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.CheckConstraint("status IN ('draft', 'active', 'completed', 'archived')"),
        sa.CheckConstraint("priority IN ('high', 'medium', 'low')"),
        sa.CheckConstraint("business_value IN ('high', 'medium', 'low')")
    )

    # Create features table
    op.create_table(
        'features',
        sa.Column('id', sa.Integer(), primary_key=True, autoincrement=True),
        sa.Column('epic_id', sa.Integer(), nullable=False),
        sa.Column('key', sa.String(20), nullable=False, unique=True),
        sa.Column('title', sa.Text(), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('status', sa.String(20), nullable=False),
        sa.Column('progress_pct', sa.Float(), nullable=False, server_default='0.0'),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('updated_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.ForeignKeyConstraint(['epic_id'], ['epics.id'], ondelete='CASCADE'),
        sa.CheckConstraint("status IN ('draft', 'active', 'completed', 'archived')"),
        sa.CheckConstraint("progress_pct >= 0.0 AND progress_pct <= 100.0")
    )

    # Create tasks table
    op.create_table(
        'tasks',
        sa.Column('id', sa.Integer(), primary_key=True, autoincrement=True),
        sa.Column('feature_id', sa.Integer(), nullable=False),
        sa.Column('key', sa.String(30), nullable=False, unique=True),
        sa.Column('title', sa.Text(), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        sa.Column('status', sa.String(20), nullable=False),
        sa.Column('agent_type', sa.String(20), nullable=True),
        sa.Column('priority', sa.Integer(), nullable=False, server_default='5'),
        sa.Column('depends_on', sa.Text(), nullable=True),
        sa.Column('assigned_agent', sa.String(100), nullable=True),
        sa.Column('file_path', sa.Text(), nullable=True),
        sa.Column('blocked_reason', sa.Text(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.Column('started_at', sa.DateTime(), nullable=True),
        sa.Column('completed_at', sa.DateTime(), nullable=True),
        sa.Column('blocked_at', sa.DateTime(), nullable=True),
        sa.Column('updated_at', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.ForeignKeyConstraint(['feature_id'], ['features.id'], ondelete='CASCADE'),
        sa.CheckConstraint("status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived')"),
        sa.CheckConstraint("agent_type IN ('frontend', 'backend', 'api', 'testing', 'devops', 'general')"),
        sa.CheckConstraint("priority >= 1 AND priority <= 10")
    )

    # Create task_history table
    op.create_table(
        'task_history',
        sa.Column('id', sa.Integer(), primary_key=True, autoincrement=True),
        sa.Column('task_id', sa.Integer(), nullable=False),
        sa.Column('old_status', sa.String(20), nullable=True),
        sa.Column('new_status', sa.String(20), nullable=False),
        sa.Column('agent', sa.String(100), nullable=True),
        sa.Column('notes', sa.Text(), nullable=True),
        sa.Column('timestamp', sa.DateTime(), nullable=False, server_default=sa.func.now()),
        sa.ForeignKeyConstraint(['task_id'], ['tasks.id'], ondelete='CASCADE')
    )

    # Create indexes
    op.create_index('idx_epics_key', 'epics', ['key'], unique=True)
    op.create_index('idx_features_key', 'features', ['key'], unique=True)
    op.create_index('idx_features_epic_id', 'features', ['epic_id'])
    op.create_index('idx_tasks_key', 'tasks', ['key'], unique=True)
    op.create_index('idx_tasks_feature_id', 'tasks', ['feature_id'])
    op.create_index('idx_tasks_status', 'tasks', ['status'])
    op.create_index('idx_tasks_agent_type', 'tasks', ['agent_type'])
    op.create_index('idx_tasks_status_priority', 'tasks', ['status', 'priority'])
    op.create_index('idx_task_history_task_id', 'task_history', ['task_id'])
    op.create_index('idx_task_history_timestamp', 'task_history', ['timestamp'])

    # Create triggers for updated_at auto-update
    op.execute("""
        CREATE TRIGGER epics_updated_at
        AFTER UPDATE ON epics
        FOR EACH ROW
        BEGIN
            UPDATE epics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;
    """)

    op.execute("""
        CREATE TRIGGER features_updated_at
        AFTER UPDATE ON features
        FOR EACH ROW
        BEGIN
            UPDATE features SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;
    """)

    op.execute("""
        CREATE TRIGGER tasks_updated_at
        AFTER UPDATE ON tasks
        FOR EACH ROW
        BEGIN
            UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
        END;
    """)


def downgrade():
    # Drop tables in reverse order (respecting foreign keys)
    op.drop_table('task_history')
    op.drop_table('tasks')
    op.drop_table('features')
    op.drop_table('epics')
```

### Future Schema Changes

**Guidelines**:
1. **Always create new migration** - Never modify existing migrations
2. **Test up and down** - Verify both upgrade() and downgrade() work
3. **Data migration** - Include data transformation code if needed
4. **Document breaking changes** - Add comments explaining impact

**Example Future Migration** (hypothetical):
```python
"""Add task estimation fields

Revision ID: 002
Revises: 001
Create Date: 2025-12-20
"""

def upgrade():
    # Add new columns
    op.add_column('tasks', sa.Column('estimated_hours', sa.Integer(), nullable=True))
    op.add_column('tasks', sa.Column('actual_hours', sa.Integer(), nullable=True))

def downgrade():
    # Remove columns
    op.drop_column('tasks', 'actual_hours')
    op.drop_column('tasks', 'estimated_hours')
```

---

## Performance Considerations

### Query Performance Targets

From PRD Non-Functional Requirements:

| Query Type | Target Latency | Dataset Size |
|------------|---------------|--------------|
| get_by_id() | <10ms | N/A |
| get_by_key() | <10ms | N/A (indexed) |
| list_all() | <100ms | 10,000 tasks |
| filter_by_status() | <100ms | 10,000 tasks |
| filter_combined() | <100ms | 10,000 tasks |
| calculate_progress() | <200ms | 50 features |
| INSERT task | <50ms | N/A |
| UPDATE task | <50ms | N/A |
| DELETE epic (cascade) | <500ms | 100 features, 1000 tasks |

### Index Usage

**Queries using indexes**:
- `WHERE epic.key = ?` → `idx_epics_key` (UNIQUE)
- `WHERE feature.key = ?` → `idx_features_key` (UNIQUE)
- `WHERE task.key = ?` → `idx_tasks_key` (UNIQUE)
- `WHERE feature.epic_id = ?` → `idx_features_epic_id`
- `WHERE task.feature_id = ?` → `idx_tasks_feature_id`
- `WHERE task.status = ?` → `idx_tasks_status`
- `WHERE task.agent_type = ?` → `idx_tasks_agent_type`
- `WHERE task.status = ? AND task.priority <= ?` → `idx_tasks_status_priority` (composite)
- `WHERE task_history.task_id = ?` → `idx_task_history_task_id`
- `ORDER BY task_history.timestamp DESC` → `idx_task_history_timestamp`

### Write Performance

**Batch inserts** (from PRD: 100 tasks in <2s):
```python
# Use transaction for batch inserts
with session.begin():
    for task_data in task_list:
        session.add(Task(**task_data))
    # Single commit at end (~20ms per task avg)
```

**Cascade delete performance**:
- Deleting epic with 50 features and 1000 tasks: ~500ms (acceptable)
- SQLite handles cascade efficiently with indexes

---

## Data Validation

### Application-Level Validation

**Before INSERT/UPDATE**:
1. Validate key format (regex patterns)
2. Validate enum values (status, priority, agent_type)
3. Validate priority range (1-10)
4. Validate progress_pct range (0.0-100.0)
5. Validate depends_on is valid JSON array
6. Validate foreign key references exist (epic_id, feature_id)

**Example Validation Function**:
```python
def validate_task_data(data: dict) -> None:
    # Key format
    if not re.match(r'^T-E\d{2}-F\d{2}-\d{3}$', data['key']):
        raise ValidationError("Invalid task key format")

    # Status enum
    if data['status'] not in TaskStatus.__members__.values():
        raise ValidationError(f"Invalid status: {data['status']}")

    # Priority range
    if not (1 <= data['priority'] <= 10):
        raise ValidationError(f"Priority must be 1-10, got: {data['priority']}")

    # JSON validation
    if data.get('depends_on'):
        try:
            deps = json.loads(data['depends_on'])
            if not isinstance(deps, list):
                raise ValidationError("depends_on must be JSON array")
        except json.JSONDecodeError:
            raise ValidationError("depends_on must be valid JSON")
```

---

## Backup and Recovery

### Database Backup

**Strategy**: Copy database file
```bash
# Simple file copy
cp project.db project_backup_$(date +%Y%m%d_%H%M%S).db

# SQLite backup command (online backup, doesn't lock)
sqlite3 project.db ".backup project_backup.db"
```

**Frequency**: Before schema migrations, daily automated backups

### Recovery

**Corruption Detection**:
```sql
PRAGMA integrity_check;
-- Should return: ok
```

**Restore from Backup**:
```bash
# Replace current database
cp project_backup.db project.db
```

**WAL Recovery**:
- WAL mode automatically recovers from crashes
- `.db-wal` and `.db-shm` files contain pending transactions
- On next database open, SQLite applies WAL automatically

---

## Security Considerations

### SQL Injection Prevention

- **Parameterized queries only** (enforced by SQLAlchemy ORM)
- Never concatenate user input into SQL strings
- All application queries use ORM or bound parameters

### File Permissions

**Unix systems**:
```bash
# Set restrictive permissions (owner read/write only)
chmod 600 project.db
chmod 600 project.db-wal
chmod 600 project.db-shm
```

**Application should enforce**:
```python
import os
import stat

# After creating database
os.chmod(db_path, stat.S_IRUSR | stat.S_IWUSR)  # 600
```

### Sensitive Data

**No sensitive data in this schema**:
- No passwords, tokens, or secrets
- All data is project metadata

**Logging**:
- Disable SQL query logging in production
- Don't log query parameters (could contain user input)

---

## Schema Versioning

### Alembic Version Table

Alembic automatically creates:
```sql
CREATE TABLE alembic_version (
    version_num VARCHAR(32) NOT NULL PRIMARY KEY
);
```

**Current Version**: `001` (initial schema)

### Version Check on Startup

```python
def check_schema_version(session):
    result = session.execute("SELECT version_num FROM alembic_version")
    current_version = result.scalar()

    if current_version != EXPECTED_VERSION:
        raise SchemaVersionMismatch(current_version, EXPECTED_VERSION)
```

---

## Summary

This data design defines:

1. **4 Tables**: epics, features, tasks, task_history
2. **3 Foreign Key Relationships**: epic→feature→task→history (all CASCADE DELETE)
3. **10 Indexes**: 3 unique (keys), 7 non-unique (query performance)
4. **3 Triggers**: Auto-update updated_at timestamps
5. **12 Check Constraints**: Enum validation, range validation
6. **Referential Integrity**: Enforced via foreign keys and cascading deletes
7. **Progress Calculation**: Cached in features.progress_pct, recalculated on task status changes
8. **Audit Trail**: task_history records all status transitions
9. **Performance**: Optimized for <100ms queries on 10K task datasets
10. **Migration**: Alembic-based versioned schema with up/down support

All schema definitions align with interface contracts (01-interface-contracts.md) and PRD requirements.

---

**Data Design Complete**: 2025-12-14
**Next Document**: 04-backend-design.md (backend architect creates ORM implementation)
