# E16-F04: Notes and Context for Epic and Feature - Architecture Plan

**Feature**: E16-F04-notes-and-context-for-epic-and-feature
**Epic**: E16 - Multi-Level Workflow System
**Depends On**: Existing note/context system (task-only); E16-F01 (Core Workflow Engine - for workflow position in resume)

---

## Overview

This feature extends the existing task-level note, context, and resume system to support epic and feature entities. Today, agents can only record persistent context (decisions, blockers, questions, progress) at the task level. Agents working on epic research or feature refinement have no mechanism to persist context between sessions.

The solution generalizes the existing `task_notes` table into a polymorphic `entity_notes` table, adds `context_data` columns to epics and features, and extends the CLI commands to support all three entity levels.

---

## Architecture Approach

### Design Principle: Polymorphic Notes, Per-Entity Context

Two concerns need separate treatment:

1. **Notes** (append-only log of decisions, blockers, comments): Currently stored in `task_notes` with a `task_id` foreign key. Extend to a polymorphic pattern using `entity_type` + `entity_id` columns so a single table stores notes for all entity types.

2. **Context Data** (mutable key-value state): Currently stored as a JSON column `context_data` on the `tasks` table. Add the same column to `epics` and `features` tables.

### Key Design Decision: Single `entity_notes` Table vs Separate Tables

**Option A**: Create `epic_notes` and `feature_notes` tables (3 separate tables)
**Option B**: Create a single `entity_notes` table with `entity_type` discriminator

**Chosen: Option B (polymorphic `entity_notes`)** because:
- Note types, content structure, and query patterns are identical across entities
- Search across entity types is a stated future requirement
- Reduces schema duplication (one set of indexes, one repository)
- Matches the feature PRD's stated approach: `entity_type` + `entity_id` pattern
- Existing `task_notes` data migrates cleanly (all rows get `entity_type = 'task'`)

**Trade-off**: No foreign key enforcement at the database level (SQLite doesn't support polymorphic FKs). Referential integrity is enforced at the repository layer via entity existence checks before note creation.

---

## Layer-by-Layer Design

### 1. Database Schema (`internal/db/db.go`)

#### New Table: `entity_notes`

Replaces `task_notes`. Existing data is migrated.

```sql
CREATE TABLE IF NOT EXISTS entity_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('epic', 'feature', 'task')),
    entity_id INTEGER NOT NULL,
    note_type TEXT CHECK (note_type IN (
        'comment',
        'decision',
        'blocker',
        'solution',
        'reference',
        'implementation',
        'testing',
        'future',
        'question',
        'rejection'
    )) NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT,  -- JSON for rejection metadata, etc.

    -- No foreign key: polymorphic reference enforced at repository layer
);

-- Indexes for entity_notes
CREATE INDEX IF NOT EXISTS idx_entity_notes_entity ON entity_notes(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_notes_type ON entity_notes(note_type);
CREATE INDEX IF NOT EXISTS idx_entity_notes_created_at ON entity_notes(created_at);
CREATE INDEX IF NOT EXISTS idx_entity_notes_entity_type ON entity_notes(entity_type, entity_id, note_type);
CREATE INDEX IF NOT EXISTS idx_entity_notes_type_entity ON entity_notes(note_type, entity_type, entity_id);
```

#### Schema Changes: Epics and Features Tables

Add `context_data` column to both tables:

```sql
ALTER TABLE epics ADD COLUMN context_data TEXT;
ALTER TABLE features ADD COLUMN context_data TEXT;
```

#### Migration: `task_notes` to `entity_notes`

```sql
-- Step 1: Create entity_notes table
-- Step 2: Copy existing task notes
INSERT INTO entity_notes (id, entity_type, entity_id, note_type, content, created_by, created_at, metadata)
SELECT id, 'task', task_id, note_type, content, created_by, created_at, metadata
FROM task_notes;

-- Step 3: Drop old table (or keep as backup and rename)
-- Recommend: rename task_notes to task_notes_backup, drop after verification
```

**Migration safety**: The migration runs in `runMigrations()` using the existing pattern (check if table/column exists before altering). The old `task_notes` table is kept as `task_notes_backup` for one release cycle.

---

### 2. Models (`internal/models/`)

#### Rename/Generalize: `TaskNote` -> `EntityNote`

**New file**: `internal/models/entity_note.go`

```go
package models

// EntityType represents the type of entity a note is attached to
type EntityType string

const (
    EntityTypeEpic    EntityType = "epic"
    EntityTypeFeature EntityType = "feature"
    EntityTypeTask    EntityType = "task"
)

// EntityNote represents a typed note attached to any entity
type EntityNote struct {
    ID         int64      `json:"id" db:"id"`
    EntityType EntityType `json:"entity_type" db:"entity_type"`
    EntityID   int64      `json:"entity_id" db:"entity_id"`
    NoteType   NoteType   `json:"note_type" db:"note_type"`
    Content    string     `json:"content" db:"content"`
    CreatedBy  *string    `json:"created_by,omitempty" db:"created_by"`
    Metadata   *string    `json:"metadata,omitempty" db:"metadata"`
    CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

func (n *EntityNote) Validate() error {
    if n.EntityType == "" {
        return errors.New("entity_type is required")
    }
    if n.EntityID <= 0 {
        return errors.New("entity_id must be positive")
    }
    if !isValidNoteType(n.NoteType) {
        return fmt.Errorf("invalid note type: %s", n.NoteType)
    }
    if strings.TrimSpace(n.Content) == "" {
        return errors.New("content cannot be empty")
    }
    return nil
}
```

**Clean replacement**: Delete `internal/models/task_note.go`. All ~17 files referencing `TaskNote` are updated to use `EntityNote` directly. The `NoteType` constants (`NoteTypeComment`, `NoteTypeDecision`, etc.) move into `entity_note.go`. This is a straightforward rename across `internal/` -- no external consumers exist.

#### ContextData Model: No Changes

The existing `ContextData` struct (`internal/models/context_data.go`) works for all entity types. The same fields (progress, decisions, blockers, questions, acceptance criteria) are meaningful for epics and features too.

---

### 3. Repository Layer (`internal/repository/`)

#### New File: `internal/repository/entity_note_repository.go`

Generalizes `TaskNoteRepository` to work with any entity type.

```go
type EntityNoteRepository struct {
    db *db.DB
}

func NewEntityNoteRepository(db *db.DB) *EntityNoteRepository

// Core CRUD
func (r *EntityNoteRepository) Create(ctx context.Context, note *models.EntityNote) error
func (r *EntityNoteRepository) GetByID(ctx context.Context, id int64) (*models.EntityNote, error)
func (r *EntityNoteRepository) GetByEntity(ctx context.Context, entityType models.EntityType, entityID int64) ([]*models.EntityNote, error)
func (r *EntityNoteRepository) GetByEntityAndType(ctx context.Context, entityType models.EntityType, entityID int64, noteTypes []models.NoteType) ([]*models.EntityNote, error)
func (r *EntityNoteRepository) Delete(ctx context.Context, id int64) error

// Search (extends existing search to include entity type filter)
func (r *EntityNoteRepository) Search(ctx context.Context, query string, noteTypes []models.NoteType, entityType *models.EntityType, epicKey, featureKey string) ([]*NoteSearchResult, error)

// Rejection notes (used by task workflow)
func (r *EntityNoteRepository) CreateRejectionNote(ctx context.Context, entityType models.EntityType, entityID int64, historyID int64, fromStatus, toStatus, reason, rejectedBy, documentPath string) error
func (r *EntityNoteRepository) GetRejectionHistory(ctx context.Context, entityType models.EntityType, entityID int64) ([]*RejectionHistoryEntry, error)
```

**Entity existence validation**: Before creating a note, the repository validates that the referenced entity exists:

```go
func (r *EntityNoteRepository) validateEntity(ctx context.Context, entityType models.EntityType, entityID int64) error {
    var count int
    var query string
    switch entityType {
    case models.EntityTypeEpic:
        query = "SELECT COUNT(*) FROM epics WHERE id = ?"
    case models.EntityTypeFeature:
        query = "SELECT COUNT(*) FROM features WHERE id = ?"
    case models.EntityTypeTask:
        query = "SELECT COUNT(*) FROM tasks WHERE id = ?"
    default:
        return fmt.Errorf("unknown entity type: %s", entityType)
    }
    if err := r.db.QueryRowContext(ctx, query, entityID).Scan(&count); err != nil {
        return fmt.Errorf("failed to validate entity: %w", err)
    }
    if count == 0 {
        return fmt.Errorf("%s with id %d not found", entityType, entityID)
    }
    return nil
}
```

#### `TaskNoteRepository`: Delete and Replace

Delete `internal/repository/task_note_repository.go`. All ~11 files referencing `TaskNoteRepository` are updated to use `EntityNoteRepository` directly. Callers that previously passed `taskID` now pass `(models.EntityTypeTask, taskID)`. This is a clean cut -- no wrappers, no aliases, no delegation.

**Files to update** (all in `internal/`):
- `cli/commands/task_note.go` - use `EntityNoteRepository`
- `cli/commands/task_resume.go` - use `EntityNoteRepository`
- `cli/commands/task.go` - use `EntityNoteRepository`
- `cli/commands/notes_search.go` - use `EntityNoteRepository`
- `repository/task_repository.go` - remove `TaskNoteRepository` references
- All associated test files

#### Context Data: Epic and Feature Repositories

Add methods to existing `EpicRepository` and `FeatureRepository`:

```go
// In EpicRepository
func (r *EpicRepository) GetContextData(ctx context.Context, epicID int64) (*models.ContextData, error)
func (r *EpicRepository) UpdateContextData(ctx context.Context, epicID int64, data *models.ContextData) error

// In FeatureRepository
func (r *FeatureRepository) GetContextData(ctx context.Context, featureID int64) (*models.ContextData, error)
func (r *FeatureRepository) UpdateContextData(ctx context.Context, featureID int64, data *models.ContextData) error
```

These follow the exact same pattern as the existing task `context_data` handling: read JSON from column, deserialize, and merge updates.

---

### 4. Service Layer (`internal/services/`)

#### New File: `internal/services/note_service.go`

Encapsulates note operations across all entity types with entity key resolution.

```go
type NoteService struct {
    entityNoteRepo *repository.EntityNoteRepository
    epicRepo       *repository.EpicRepository
    featureRepo    *repository.FeatureRepository
    taskRepo       *repository.TaskRepository
}

func NewNoteService(
    entityNoteRepo *repository.EntityNoteRepository,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
    taskRepo *repository.TaskRepository,
) *NoteService

// AddNote resolves entity key to type+ID and creates the note
func (s *NoteService) AddNote(ctx context.Context, entityType models.EntityType, entityKey string, noteType models.NoteType, content string, createdBy *string) (*models.EntityNote, error)

// ListNotes retrieves all notes for an entity
func (s *NoteService) ListNotes(ctx context.Context, entityType models.EntityType, entityKey string, noteTypes []models.NoteType) ([]*models.EntityNote, error)

// resolveEntityID converts a user-provided key to an entity ID
func (s *NoteService) resolveEntityID(ctx context.Context, entityType models.EntityType, key string) (int64, error)
```

#### New File: `internal/services/context_service.go`

Encapsulates context data operations across all entity types.

```go
type ContextService struct {
    epicRepo    *repository.EpicRepository
    featureRepo *repository.FeatureRepository
    taskRepo    *repository.TaskRepository
}

func NewContextService(
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
    taskRepo *repository.TaskRepository,
) *ContextService

// GetContext retrieves context data for any entity
func (s *ContextService) GetContext(ctx context.Context, entityType models.EntityType, entityKey string) (*models.ContextData, error)

// SetContextField sets a single field on an entity's context data
func (s *ContextService) SetContextField(ctx context.Context, entityType models.EntityType, entityKey string, field string, value string) error

// ClearContext clears all context data for an entity
func (s *ContextService) ClearContext(ctx context.Context, entityType models.EntityType, entityKey string) error
```

#### New File: `internal/services/resume_service.go`

Encapsulates the resume command logic for all entity types.

```go
type ResumeService struct {
    noteSvc    *NoteService
    contextSvc *ContextService
    epicRepo   *repository.EpicRepository
    featureRepo *repository.FeatureRepository
    taskRepo   *repository.TaskRepository
    workflowSvc *workflow.Service
}

// EpicResumeContext contains all data needed to resume work on an epic
type EpicResumeContext struct {
    Epic        *models.Epic        `json:"epic"`
    ContextData *models.ContextData `json:"context_data,omitempty"`
    Notes       []*models.EntityNote `json:"notes,omitempty"`
    Features    []*models.Feature   `json:"features,omitempty"`
    // Workflow position (from E16-F01 if available)
    WorkflowPhase *string          `json:"workflow_phase,omitempty"`
}

// FeatureResumeContext contains all data needed to resume work on a feature
type FeatureResumeContext struct {
    Feature     *models.Feature     `json:"feature"`
    ContextData *models.ContextData `json:"context_data,omitempty"`
    Notes       []*models.EntityNote `json:"notes,omitempty"`
    Tasks       []*models.Task      `json:"tasks,omitempty"`
    Epic        *models.Epic        `json:"epic,omitempty"` // Parent context
    // Workflow position (from E16-F01 if available)
    WorkflowPhase *string          `json:"workflow_phase,omitempty"`
}

func (s *ResumeService) GetEpicResumeContext(ctx context.Context, epicKey string) (*EpicResumeContext, error)
func (s *ResumeService) GetFeatureResumeContext(ctx context.Context, featureKey string) (*FeatureResumeContext, error)
```

---

### 5. CLI Command Layer (`internal/cli/commands/`)

All new commands follow the **thin wrapper** pattern: parse args, call service, format output.

#### Epic Note Commands

**New file**: `internal/cli/commands/epic_note.go`

```
shark epic note add <key> --type <type> "<content>" [--created-by <agent>] [--json]
shark epic note list <key> [--type <types>] [--json]
```

Registered as subcommands of `epicCmd`.

#### Feature Note Commands

**New file**: `internal/cli/commands/feature_note.go`

```
shark feature note add <key> --type <type> "<content>" [--created-by <agent>] [--json]
shark feature note list <key> [--type <types>] [--json]
```

Registered as subcommands of `featureCmd`.

#### Epic Context Commands

**New file**: `internal/cli/commands/epic_context.go`

```
shark epic context set <key> --field <field> --value "<value>" [--json]
shark epic context get <key> [--json]
shark epic context clear <key> [--json]
```

#### Feature Context Commands

**New file**: `internal/cli/commands/feature_context.go`

```
shark feature context set <key> --field <field> --value "<value>" [--json]
shark feature context get <key> [--json]
shark feature context clear <key> [--json]
```

#### Epic Resume Command

**New file**: `internal/cli/commands/epic_resume.go`

```
shark epic resume <key> [--json]
```

Displays: epic details, context data (progress, decisions, blockers, questions), notes, features list, workflow position.

#### Feature Resume Command

**New file**: `internal/cli/commands/feature_resume.go`

```
shark feature resume <key> [--json]
```

Displays: feature details, context data, notes, task list, parent epic context, workflow position.

#### Modified: Epic Get and Feature Get

**Modified files**: `epic.go`, `feature.go`

Add notes and context data to the output of `shark get` commands:
- JSON output: add `notes` and `context_data` fields
- Human-readable output: add "Notes" and "Context" sections at the bottom

---

### 6. JSON Output Structures

#### Note Commands

```json
// shark epic note add E16 --type decision "Chose polymorphic notes" --json
{
    "id": 42,
    "entity_type": "epic",
    "entity_id": 16,
    "note_type": "decision",
    "content": "Chose polymorphic notes table over separate tables",
    "created_by": "architect-agent",
    "created_at": "2026-02-09T10:30:00Z"
}
```

```json
// shark epic note list E16 --json
[
    {
        "id": 42,
        "entity_type": "epic",
        "entity_id": 16,
        "note_type": "decision",
        "content": "Chose polymorphic notes table over separate tables",
        "created_by": "architect-agent",
        "created_at": "2026-02-09T10:30:00Z"
    }
]
```

#### Context Commands

```json
// shark feature context get E16-F01 --json
{
    "progress": {
        "completed_steps": ["BA refinement", "Tech review"],
        "current_step": "Task generation",
        "remaining_steps": ["Build", "QA"]
    },
    "implementation_decisions": {
        "database_approach": "polymorphic entity_notes table",
        "migration_strategy": "rename old table as backup"
    },
    "open_questions": [
        "Should we add entity-specific note types later?"
    ],
    "blockers": [],
    "related_tasks": []
}
```

#### Resume Commands

```json
// shark epic resume E16 --json
{
    "epic": { "key": "E16", "title": "Multi-Level Workflow System", "status": "active" },
    "context_data": { ... },
    "notes": [ ... ],
    "features": [ ... ],
    "workflow_phase": "execution"
}
```

```json
// shark feature resume E16-F04 --json
{
    "feature": { "key": "E16-F04", "title": "Notes and Context for Epic and Feature", "status": "in_refinement_tech" },
    "context_data": { ... },
    "notes": [ ... ],
    "tasks": [ ... ],
    "epic": { "key": "E16", "title": "Multi-Level Workflow System" },
    "workflow_phase": "refinement"
}
```

#### Get Commands (Enhanced)

```json
// shark get E16 --json (new fields: notes, context_data)
{
    "key": "E16",
    "title": "Multi-Level Workflow System",
    "status": "active",
    "notes": [ ... ],
    "context_data": { ... },
    "features": [ ... ],
    "progress_pct": 45.0
}
```

---

## Backward Compatibility

### Database Migration

- The `entity_notes` table is created alongside the existing `task_notes` table
- Existing `task_notes` data is migrated to `entity_notes` with `entity_type = 'task'`
- The old `task_notes` table is renamed to `task_notes_backup` (not dropped)
- Migration is idempotent (safe to run multiple times)

### Code References: Clean Replacement

- `TaskNote` model is deleted; all ~17 files updated to use `EntityNote` directly
- `TaskNoteRepository` is deleted; all ~11 files updated to use `EntityNoteRepository` with `entity_type = 'task'`
- `NoteType` constants move from `task_note.go` to `entity_note.go` unchanged
- All existing tests are updated in the same pass (repository tests, CLI tests, search tests)

### Existing CLI Commands

- `shark task note add`, `shark task note list`, `shark task context` commands continue to work with updated internals
- `shark task resume` continues to work; it uses `EntityNoteRepository` with `entity_type = 'task'`

### Context Data Model

- `models.ContextData` is reused unchanged for epics and features
- All fields are optional (`omitempty`), so entities with no context return `null`/empty

---

## Performance Considerations

### Entity Notes Table Size

The polymorphic `entity_notes` table will hold notes for all entity types. Performance is maintained via:
- Composite index on `(entity_type, entity_id)` for filtered queries
- Composite index on `(entity_type, entity_id, note_type)` for type-filtered queries
- Existing queries that previously hit `task_notes` now hit `entity_notes` with an additional `entity_type = 'task'` predicate, which the composite index handles efficiently

### Context Data

- Context data is stored as JSON in a single column (same pattern as tasks)
- No additional queries needed; context is loaded with the entity's primary query
- Merge operations happen in-memory, not via SQL

### Resume Command

The resume command aggregates multiple data sources (entity + notes + context + children + workflow). This is the most expensive operation but is only called on-demand. The query count per resume call:
- Epic resume: 1 (epic) + 1 (notes) + 1 (features) = 3 queries
- Feature resume: 1 (feature) + 1 (notes) + 1 (tasks) + 1 (parent epic) = 4 queries

---

## File Changes Summary

### New Files

| File | Purpose |
|------|---------|
| `internal/models/entity_note.go` | EntityNote model with EntityType enum |
| `internal/repository/entity_note_repository.go` | Polymorphic note CRUD |
| `internal/repository/entity_note_repository_test.go` | Repository tests (real DB) |
| `internal/services/note_service.go` | Note business logic, entity key resolution |
| `internal/services/note_service_test.go` | Service tests (mocked repos) |
| `internal/services/context_service.go` | Context data operations |
| `internal/services/context_service_test.go` | Service tests (mocked repos) |
| `internal/services/resume_service.go` | Resume context assembly |
| `internal/services/resume_service_test.go` | Service tests (mocked repos) |
| `internal/cli/commands/epic_note.go` | Epic note CLI commands |
| `internal/cli/commands/feature_note.go` | Feature note CLI commands |
| `internal/cli/commands/epic_context.go` | Epic context CLI commands |
| `internal/cli/commands/feature_context.go` | Feature context CLI commands |
| `internal/cli/commands/epic_resume.go` | Epic resume CLI command |
| `internal/cli/commands/feature_resume.go` | Feature resume CLI command |

### Modified Files

| File | Changes |
|------|---------|
| `internal/db/db.go` | Add `entity_notes` table, migration from `task_notes`, add `context_data` to epics/features |
| `internal/repository/epic_repository.go` | Add `GetContextData`, `UpdateContextData` methods |
| `internal/repository/feature_repository.go` | Add `GetContextData`, `UpdateContextData` methods |
| `internal/cli/commands/epic.go` | Add notes/context to `shark epic get` output |
| `internal/cli/commands/feature.go` | Add notes/context to `shark feature get` output |
| `internal/cli/commands/task_note.go` | Replace `TaskNoteRepository` with `EntityNoteRepository` |
| `internal/cli/commands/task_resume.go` | Replace `TaskNoteRepository` with `EntityNoteRepository` |
| `internal/cli/commands/task.go` | Replace `TaskNoteRepository` references |
| `internal/cli/commands/notes_search.go` | Replace `TaskNoteRepository` with `EntityNoteRepository` |
| `internal/repository/task_repository.go` | Remove `TaskNoteRepository` references |
| All associated test files (~6 files) | Update to use `EntityNote` and `EntityNoteRepository` |

### Deleted Files

| File | Reason |
|------|--------|
| `internal/models/task_note.go` | Replaced by `entity_note.go` |
| `internal/repository/task_note_repository.go` | Replaced by `entity_note_repository.go` |

### Files NOT Modified

| File | Reason |
|------|--------|
| `internal/models/context_data.go` | Reused unchanged for all entity types |
| `internal/cli/commands/task_context.go` | Existing task context commands remain unchanged (already uses task repo) |

---

## Task Breakdown

### Task 1: Database Schema & Migration (M)
- Create `entity_notes` table in `createSchema()`
- Add migration in `runMigrations()` to:
  - Create `entity_notes` if not exists
  - Copy data from `task_notes` to `entity_notes` with `entity_type = 'task'`
  - Rename `task_notes` to `task_notes_backup`
- Add `context_data TEXT` column to `epics` and `features` tables
- Verify migration is idempotent
- Tests: verify schema creation and data migration

### Task 2: EntityNote Model & Repository (M)
- Create `internal/models/entity_note.go` with `EntityNote` struct, `EntityType` enum, and `NoteType` constants
- Delete `internal/models/task_note.go`
- Create `internal/repository/entity_note_repository.go` with CRUD methods
- Delete `internal/repository/task_note_repository.go`
- Implement entity existence validation
- Update all ~17 files referencing `TaskNote` / `TaskNoteRepository` to use new types
- Repository tests with real database

### Task 3: Context Data for Epic/Feature Repositories (S)
- Add `GetContextData()` and `UpdateContextData()` to `EpicRepository`
- Add `GetContextData()` and `UpdateContextData()` to `FeatureRepository`
- Follow same JSON serialization pattern as task `context_data`
- Repository tests with real database

### Task 4: NoteService & ContextService (M)
- Create `internal/services/note_service.go` with entity key resolution
- Create `internal/services/context_service.go` with field set/get/clear
- Service tests with mocked repositories

### Task 5: Epic Note & Context CLI Commands (S)
- Create `epic_note.go`: `shark epic note add`, `shark epic note list`
- Create `epic_context.go`: `shark epic context set/get/clear`
- CLI tests with mocked services

### Task 6: Feature Note & Context CLI Commands (S)
- Create `feature_note.go`: `shark feature note add`, `shark feature note list`
- Create `feature_context.go`: `shark feature context set/get/clear`
- CLI tests with mocked services

### Task 7: ResumeService & Resume CLI Commands (M)
- Create `internal/services/resume_service.go`
- Create `epic_resume.go`: `shark epic resume <key>`
- Create `feature_resume.go`: `shark feature resume <key>`
- Aggregate entity + notes + context + children + workflow position
- Service tests and CLI tests

### Task 8: Enhance Get Commands (S)
- Modify `shark epic get` to include notes and context in output
- Modify `shark feature get` to include notes and context in output
- Update JSON output structures
- Update human-readable formatters

### Task 9: Integration Testing & Edge Cases (S)
- Test full lifecycle: add note -> list notes -> set context -> resume
- Test across entity types
- Test backward compatibility (existing task note/context commands)
- Test with empty notes/context
- Test JSON output for all commands

---

## Dependency Graph

```
Task 1: Database Schema & Migration
    │
    ├──► Task 2: EntityNote Model & Repository
    │       │
    │       ├──► Task 4: NoteService & ContextService
    │       │       │
    │       │       ├──► Task 5: Epic Note & Context CLI
    │       │       ├──► Task 6: Feature Note & Context CLI
    │       │       └──► Task 7: ResumeService & Resume CLI
    │       │
    │       └──► Task 8: Enhance Get Commands
    │
    └──► Task 3: Context Data for Epic/Feature Repos
            │
            └──► Task 4 (also depends on Task 3)

All Tasks ──► Task 9: Integration Testing
```

---

## Design Decisions (Resolved)

1. **Note Timeline for Epics/Features**: The existing `shark task note timeline` command interleaves status changes with notes. Should we implement this for epics and features? **Decision**: Yes, implement eventually but low priority. Deferred until status history tracking exists at epic/feature level (E16-F01 may add this). Tracked as T-E16-F04-001 (priority 8).

2. **Work Sessions for Epics/Features**: The existing `work_sessions` table is task-specific (`task_id` FK). Should we extend work sessions to epics/features? **Decision**: Do NOT extend. The work_sessions feature has never been used. Instead, investigate the feature for possible removal. Tracked as T-E16-F04-002 (priority 9).

3. **Note Search Across Entity Types**: The feature PRD lists "Note search/filtering across entities" as out of scope. However, the polymorphic `entity_notes` table makes this trivial to add later. **Decision**: Address this -- add `--entity-type` filter to the `notes search` command. With notes spanning all entity types, users need the ability to scope searches. Tracked as T-E16-F04-003 (priority 5).

4. **Rejection Notes for Epics/Features**: Rejection notes have special metadata (history_id, from_status, to_status). Should epics/features support rejection notes? **Decision**: Agreed with recommendation. Support the note type in the schema (it's just a `note_type` value), but don't implement rejection-specific CLI commands until E16-F05 (Backward Transition and Escalation) needs them. No separate task needed.

---

*Last Updated*: 2026-02-09
