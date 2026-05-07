---
feature_key: E19-F03-sprint-task-assignment-backlog
epic_key: E19
title: Sprint Entity Assignment & Backlog
spec_version: 1.0
last_updated: 2026-05-06
complexity: STANDARD
---

# Spec — E19-F03: Sprint Entity Assignment & Backlog

> **Scope guard.** This feature delivers the commands that connect work entities (tasks, bugs, change-cards, tech-debt) to sprints: `shark sprint add`, `shark sprint remove`, `shark sprint backlog`, and the carryover logic in `shark sprint close`. Out of scope here: analytics (velocity, burndown, summary — E19-F04), sprint planning view (E19-F05), capacity management (E19-F05).

---

## 1. Requirements

This feature satisfies REQ-F-004, REQ-F-005, and REQ-F-006 from [E19 requirements.md](../requirements.md). REQ-F-012 (bulk assignment) is addressed as a Should-Have extension of REQ-F-004. Non-functional requirements REQ-NF-001 through REQ-NF-006 apply; REQ-NF-001 and REQ-NF-002 are the binding performance targets.

### 1.1 Functional Requirements (incremental over E19-F01 and E19-F02)

| ID | Requirement | Priority |
|---|---|---|
| REQ-F-004 | Individual entity assignment (`sprint add` / `sprint remove`) for all four entity types | Must Have |
| REQ-F-005 | Sprint backlog view (`sprint backlog`) grouped by status, all entity types | Must Have |
| REQ-F-006 | Sprint close with carryover (`sprint close --carryover=next\|backlog`) | Must Have |
| REQ-F-012 | Bulk entity assignment (`sprint add --bulk`, `--bulk-bugs`, etc.) | Should Have |

### 1.2 Acceptance Criteria

Derived directly from epic PRD Section "Must Have Requirements — Entity-to-Sprint Assignment" and "Sprint Close with Carryover".

**REQ-F-004 — Individual Assignment**

- [ ] `shark sprint add S024 E07-F01-001` assigns a task; `sprint add S024 B001` assigns a bug; `sprint add S024 CC-001` assigns a change-card; `sprint add S024 TD-001` assigns tech-debt.
- [ ] `shark sprint remove S024 E07-F01-001` removes a task (and analogues for other types).
- [ ] An entity can belong to at most one sprint in `planning` or `active` status. Attempting to add an already-assigned entity returns an error naming the conflicting sprint key.
- [ ] Assigning an entity that would exceed agent capacity emits a warning but proceeds (advisory only — enforcement is E19-F05).
- [ ] Both `add` and `remove` support `--json` output.

**REQ-F-005 — Backlog View**

- [ ] `shark sprint backlog S024` displays all entity types grouped by status category.
- [ ] Each row shows entity type label (`[task]`, `[bug]`, `[change-card]`, `[tech-debt]`), key, title, status, agent type, and priority.
- [ ] Shows completion percentage (completed / total) in the header.
- [ ] `--type=task|bug|change_card|tech_debt` filters to one entity type.
- [ ] `--blocked` shows only blocked entities with blocking reason and days blocked.
- [ ] `--json` output includes `entity_type` field on every item.

**REQ-F-006 — Sprint Close with Carryover**

- [ ] `shark sprint close S024 --carryover=next` moves all incomplete assigned entities to the next sprint in `planning` status, then advances sprint to `completed`.
- [ ] `shark sprint close S024 --carryover=backlog` soft-deletes (sets `removed_at`) assignments for incomplete entities, then advances sprint to `completed`.
- [ ] If `--carryover=next` and no next sprint exists, a new sprint is auto-created in `planning` status with start date = (closed sprint end date + 1 day) and duration matching the closed sprint.
- [ ] Completed entities remain attached to the closed sprint (their `removed_at` stays NULL so velocity queries can count them).
- [ ] Sprint close generates a `sprint_completions` row with summary statistics (entity counts, size totals — see §3.2).
- [ ] Default carryover behavior is read from `.sharkconfig.json` key `sprint_defaults.carryover` (defaults to `"next"` if absent).
- [ ] The entire close+carryover operation executes in a single database transaction.

**REQ-F-012 — Bulk Assignment (Should Have)**

- [ ] `shark sprint add S024 --bulk E07-F34` assigns all tasks from feature E07-F34 that are in assignable statuses and not already in an active/planning sprint.
- [ ] `shark sprint add S024 --bulk-bugs` assigns all open bugs not already in an active/planning sprint.
- [ ] `shark sprint add S024 --bulk-tech-debt` assigns all open tech-debt items not already in an active/planning sprint.
- [ ] `shark sprint add S024 --bulk-changes` assigns all open change-cards not already in an active/planning sprint.
- [ ] Output displays count added per entity type and updated capacity utilization.
- [ ] Warns if bulk assignment exceeds any agent type's capacity.

### 1.3 Non-Functional Requirements

- **Performance** (REQ-NF-001): `sprint add`, `sprint remove`, `sprint backlog` complete in <500ms. The carryover transaction in `sprint close` completes in <2s for sprints with up to 200 assigned entities.
- **Data Integrity** (REQ-NF-003): All assignment writes go through the partial unique index `idx_sprint_assignments_active_one` (from E19-F01). No duplicate active assignments possible at the DB level.
- **Backward Compatibility** (REQ-NF-004): No schema changes beyond adding `sprint_completions` (see §3.2). No existing commands affected.
- **JSON Consistency** (REQ-NF-005): All new commands support `--json` and `--field` flags.

### 1.4 Out of Scope for This Feature

- Velocity calculation, burndown chart, sprint summary analytics — E19-F04.
- Sprint planning view (`shark sprint plan`) and capacity configuration commands — E19-F05.
- Sprint history per entity (`shark sprint history <entity-key>`) — E19-F05 or later.
- `shark status` dashboard integration showing active sprint — Could Have (E19 scope.md).
- Enforcing sprint capacity as a hard block (advisory only here) — E19-F05.

---

## 2. Foundational State (Delivered by E19-F01 and E19-F02)

The following are already present and must not be recreated:

- **Schema**: `sprint_assignments` (polymorphic with partial unique index), `sprint_capacity`, `sprints` tables — `internal/db/db.go` `migrateSprintTables`, schema version 18.
- **Models**: `Sprint`, `SprintAssignment`, `SprintCapacity` — `internal/models/sprint.go`.
- **Validation**: `ValidateSprintAssignmentEntityType` — `internal/models/validation.go`.
- **Repository**: `SprintRepository` (CRUD + `GetNextKey`) — `internal/repository/sprint/repository.go`.
- **Service**: `SprintService` with `CreateSprint`, `GetSprint`, `ListSprints`, `UpdateSprint`, `DeleteSprint`, `StartSprint`, `CloseSprint` (stub — no carryover), `ArchiveSprint` — `internal/services/sprint_service.go`.
- **CLI commands**: `sprint create`, `get`, `list`, `update`, `delete`, `start`, `close` (stub), `archive` — `internal/cli/commands/sprint.go`.
- **Service accessor**: `cli.GetSprintService()` — `internal/cli/services_global.go`.
- **Key parsing**: `EntityTypeSprint`, `IsSprintKey()` — `internal/keys/service.go`, `internal/keys/validation.go`.

---

## 3. Data Model Changes

### 3.1 No Changes to Existing Tables

`sprint_assignments` already has the required columns (`sprint_id`, `entity_type`, `entity_id`, `assigned_at`, `removed_at`) and the partial unique index. This feature reads and writes through the existing schema only.

### 3.2 New Table: `sprint_completions`

Required by REQ-F-006 (completion record with summary statistics). Add to `migrateSprintTables` in `internal/db/db.go` as a deferred addition using `IF NOT EXISTS` — or, if cleaner, as a separate `migrateSprintCompletionsTable` function called from `runMigrations()` after `migrateSprintTables`.

```sql
CREATE TABLE IF NOT EXISTS sprint_completions (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    sprint_id             INTEGER NOT NULL UNIQUE,           -- one record per sprint
    completed_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    planned_entity_count  INTEGER NOT NULL DEFAULT 0,
    completed_entity_count INTEGER NOT NULL DEFAULT 0,
    carried_over_count    INTEGER NOT NULL DEFAULT 0,        -- moved to next sprint
    dropped_count         INTEGER NOT NULL DEFAULT 0,        -- returned to backlog
    planned_size_sum      REAL,                              -- Σ(size) of all assigned entities (NULL if unsized)
    completed_size_sum    REAL,                              -- Σ(size) of completed entities at close
    carryover_mode        TEXT NOT NULL,                     -- 'next' or 'backlog'
    next_sprint_id        INTEGER,                           -- populated when carryover_mode = 'next'
    FOREIGN KEY (sprint_id) REFERENCES sprints(id) ON DELETE CASCADE,
    FOREIGN KEY (next_sprint_id) REFERENCES sprints(id)
);

CREATE INDEX IF NOT EXISTS idx_sprint_completions_sprint ON sprint_completions(sprint_id);
```

**Schema version**: Bump `CurrentSchemaVersion` from 18 → 19.

**SprintCompletion Go model** (add to `internal/models/sprint.go`):

```go
type SprintCompletion struct {
    ID                   int64
    SprintID             int64
    CompletedAt          time.Time
    PlannedEntityCount   int
    CompletedEntityCount int
    CarriedOverCount     int
    DroppedCount         int
    PlannedSizeSum       *float64   // nil if all entities are unsized
    CompletedSizeSum     *float64
    CarryoverMode        string     // "next" | "backlog"
    NextSprintID         *int64
}
```

---

## 4. Component Changes

### 4.1 Repository Layer — `internal/repository/sprint/repository.go`

Add the following methods to `SprintRepository`. All follow the parameterized-query pattern established in the existing file (see `GetByKey`, `List`).

#### 4.1.1 Assignment Methods

```go
// AddAssignment creates a sprint_assignments row for the given entity.
// Returns a duplicate-assignment error if the entity already has an active assignment
// (relies on the partial unique index to surface the SQLite UNIQUE constraint error).
func (r *SprintRepository) AddAssignment(ctx context.Context, assignment *models.SprintAssignment) error

// RemoveAssignment soft-deletes an active assignment by setting removed_at = NOW().
// Returns an error if no active assignment exists for (sprint_id, entity_type, entity_id).
func (r *SprintRepository) RemoveAssignment(ctx context.Context, sprintID int64, entityType string, entityID int64) error

// GetActiveAssignment returns the active assignment for an entity, or nil if none.
func (r *SprintRepository) GetActiveAssignment(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error)

// ListAssignments returns all active assignments for a sprint, optionally filtered by entity_type.
func (r *SprintRepository) ListAssignments(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error)

// ListAssignmentsForCarryover returns all active assignments for a sprint where the entity
// is NOT in a "completed" status (cross-joins tasks/bugs/change_cards/tech_debts via UNION).
// Used exclusively by the carryover transaction in SprintService.CloseSprintWithCarryover.
func (r *SprintRepository) ListAssignmentsForCarryover(ctx context.Context, sprintID int64) ([]*models.SprintAssignment, error)

// ReassignToSprint updates the sprint_id on a set of active assignments (used by carryover).
// Executes within the provided *sql.Tx.
func (r *SprintRepository) ReassignToSprintTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error

// DropAssignmentsTx soft-deletes a set of assignments (sets removed_at = NOW()) within a tx.
func (r *SprintRepository) DropAssignmentsTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64) error
```

#### 4.1.2 Completion Record Methods

```go
// CreateCompletion inserts a sprint_completions row within the provided transaction.
func (r *SprintRepository) CreateCompletionTx(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error
```

#### 4.1.3 Backlog Query Method

The backlog view requires fetching entity details (title, status, agent_type, priority) across four different tables. Because the repository layer must not join across entity tables for business logic, the backlog query uses a UNION across the four entity tables joined to `sprint_assignments`:

```go
// ListBacklog returns BacklogItem rows for all active assignments in a sprint.
// entityType is optional; if non-nil, limits to that type.
// blockedOnly limits to entities in a blocked-equivalent status.
func (r *SprintRepository) ListBacklog(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool) ([]*BacklogItem, error)

// BacklogItem is a read-only projection (not a stored model) returned by ListBacklog.
type BacklogItem struct {
    AssignmentID int64
    SprintID     int64
    EntityType   string
    EntityID     int64
    EntityKey    string
    Title        string
    Status       string
    AgentType    string
    Priority     int
    Size         *int    // nullable — from entity's size column (E07-F42)
    AssignedAt   time.Time
}
```

The SQL for `ListBacklog` is a UNION of four sub-selects:

```sql
SELECT sa.id, sa.sprint_id, 'task' AS entity_type, sa.entity_id,
       t.key, t.title, t.status, t.agent_type, t.priority, t.size, sa.assigned_at
FROM sprint_assignments sa
JOIN tasks t ON t.id = sa.entity_id
WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'task'

UNION ALL

SELECT sa.id, sa.sprint_id, 'bug' AS entity_type, sa.entity_id,
       b.key, b.title, b.status, b.agent_type, b.priority, b.size, sa.assigned_at
FROM sprint_assignments sa
JOIN bugs b ON b.id = sa.entity_id
WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'bug'

UNION ALL

SELECT sa.id, sa.sprint_id, 'change_card' AS entity_type, sa.entity_id,
       cc.key, cc.title, cc.status, cc.agent_type, cc.priority, cc.size, sa.assigned_at
FROM sprint_assignments sa
JOIN change_cards cc ON cc.id = sa.entity_id
WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'change_card'

UNION ALL

SELECT sa.id, sa.sprint_id, 'tech_debt' AS entity_type, sa.entity_id,
       td.key, td.title, td.status, td.agent_type, td.priority, td.size, sa.assigned_at
FROM sprint_assignments sa
JOIN tech_debts td ON td.id = sa.entity_id
WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'tech_debt'

ORDER BY entity_type, priority DESC
```

When `entityType` is non-nil, only the matching sub-select is included. When `blockedOnly` is true, a `AND <table>.status IN (blocked-status-values)` clause is appended; the service layer passes the blocked statuses from `workflow.Service` to avoid hardcoding them in the repository.

**Rationale for UNION in repository**: The polymorphic `sprint_assignments` table references four parent tables. A single query with dynamic table names would require string interpolation (SQL injection risk). The UNION approach uses static SQL with parameterized sprint_id bindings only, consistent with the parameterized-query mandate in `.claude/rules/go/input-sanitization.md`.

#### 4.1.4 Entity ID Resolution Helper

The service needs to resolve a human-facing key (e.g., `E07-F01-001`, `B001`, `CC-001`, `TD-001`) to `(entity_type, entity_id)` before calling repository methods. The repository exposes one method per entity type:

```go
// GetEntityIDByKey resolves a key to (entity_type, entity_id) by querying the correct table.
// entity_type is determined by the caller (via keys.KeyService.Parse).
func (r *SprintRepository) GetTaskIDByKey(ctx context.Context, key string) (int64, error)
func (r *SprintRepository) GetBugIDByKey(ctx context.Context, key string) (int64, error)
func (r *SprintRepository) GetChangeCardIDByKey(ctx context.Context, key string) (int64, error)
func (r *SprintRepository) GetTechDebtIDByKey(ctx context.Context, key string) (int64, error)
```

Each method queries its respective table (`tasks`, `bugs`, `change_cards`, `tech_debts`) using `UPPER(key) = UPPER(?)`. Returns a not-found error if the key does not exist.

**Alternative considered**: A single `GetEntityIDByKey(entityType, key)` with a switch statement. Rejected because it would require string-interpolated table names; separate typed methods keep every query static.

---

### 4.2 Service Layer — `internal/services/sprint_service.go`

Add the following methods to the existing `SprintService`. The service receives additional repository methods through the existing `SprintRepository` interface, which is extended to include the new methods from §4.1.

**Extended SprintRepository interface** (in `sprint_service.go`):

```go
type SprintRepository interface {
    // ... existing methods from E19-F02 ...

    // New for F03:
    AddAssignment(ctx context.Context, assignment *models.SprintAssignment) error
    RemoveAssignment(ctx context.Context, sprintID int64, entityType string, entityID int64) error
    GetActiveAssignment(ctx context.Context, entityType string, entityID int64) (*models.SprintAssignment, error)
    ListAssignments(ctx context.Context, sprintID int64, entityType *string) ([]*models.SprintAssignment, error)
    ListAssignmentsForCarryover(ctx context.Context, sprintID int64) ([]*models.SprintAssignment, error)
    ReassignToSprintTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64, newSprintID int64) error
    DropAssignmentsTx(ctx context.Context, tx *sql.Tx, assignmentIDs []int64) error
    CreateCompletionTx(ctx context.Context, tx *sql.Tx, completion *models.SprintCompletion) error
    ListBacklog(ctx context.Context, sprintID int64, entityType *string, blockedOnly bool) ([]*sprint.BacklogItem, error)
    GetTaskIDByKey(ctx context.Context, key string) (int64, error)
    GetBugIDByKey(ctx context.Context, key string) (int64, error)
    GetChangeCardIDByKey(ctx context.Context, key string) (int64, error)
    GetTechDebtIDByKey(ctx context.Context, key string) (int64, error)
}
```

Note: The `SprintService` also needs access to `*dbconn.DB` directly for the `CloseSprintWithCarryover` transaction. Add `db *dbconn.DB` to the struct and `NewSprintService` constructor (alongside the existing `repo` and `workflowSvc`). Update `GetSprintService()` in `services_global.go` to pass the db.

#### 4.2.1 AddEntityToSprint

```go
// AddEntityInput contains the parameters for assigning an entity to a sprint.
type AddEntityInput struct {
    SprintKey  string // e.g., "S024"
    EntityKey  string // e.g., "E07-F01-001", "B001", "CC-001", "TD-001"
}

// AddEntityToSprint assigns any supported entity to a sprint.
// Steps:
//   1. Resolve sprint by key; validate it is in planning or active status.
//   2. Parse entity key via keys.KeyService.Parse() to determine entity_type.
//   3. Resolve entity_id by querying the entity's table.
//   4. Check for existing active assignment (returns ConflictError if found).
//   5. Call repo.AddAssignment().
//   6. Optionally compute capacity warning if sprint_capacity records exist.
// Returns the created SprintAssignment.
func (s *SprintService) AddEntityToSprint(ctx context.Context, input AddEntityInput) (*models.SprintAssignment, *CapacityWarning, error)

// CapacityWarning is non-nil when assigning would push an agent type over capacity.
type CapacityWarning struct {
    AgentType  string
    Capacity   float64
    Allocated  float64  // after this assignment
}
```

#### 4.2.2 RemoveEntityFromSprint

```go
// RemoveEntityFromSprint soft-deletes the active assignment for an entity from a sprint.
// Validates the sprint key and entity key; returns not-found if no active assignment exists.
func (s *SprintService) RemoveEntityFromSprint(ctx context.Context, sprintKey, entityKey string) error
```

#### 4.2.3 GetSprintBacklog

```go
// BacklogOptions carries filter parameters for the backlog view.
type BacklogOptions struct {
    EntityType  string // "" = all types; "task", "bug", "change_card", "tech_debt" = filtered
    BlockedOnly bool
}

// GetSprintBacklog returns all entities assigned to a sprint, grouped by status.
// Delegates to repo.ListBacklog(); groups results into BacklogGroup slices.
func (s *SprintService) GetSprintBacklog(ctx context.Context, sprintKey string, opts BacklogOptions) (*SprintBacklog, error)

// SprintBacklog is the return value of GetSprintBacklog.
type SprintBacklog struct {
    SprintKey          string
    SprintName         string
    TotalCount         int
    CompletedCount     int
    CompletionPercent  float64
    Groups             []*BacklogGroup  // ordered by status phase
}

// BacklogGroup is a set of entities sharing the same status category.
type BacklogGroup struct {
    StatusCategory string           // e.g., "todo/planning", "in-progress", "review", "completed", "blocked"
    Items          []*BacklogItemView
}

// BacklogItemView is the CLI-friendly projection of a BacklogItem.
type BacklogItemView struct {
    EntityType string
    Key        string
    Title      string
    Status     string
    AgentType  string
    Priority   int
    Size       *int
    // For --blocked view:
    DaysBlocked int
}
```

#### 4.2.4 CloseSprintWithCarryover (replaces the F02 stub)

This is the highest-complexity method in the epic (as noted in the feature description). It replaces the stub `CloseSprint` from E19-F02 (which simply updated the status to `ready_for_review`).

```go
// CarryoverMode controls what happens to incomplete assignments on close.
type CarryoverMode string

const (
    CarryoverNext    CarryoverMode = "next"
    CarryoverBacklog CarryoverMode = "backlog"
)

// CloseSprintWithCarryover atomically:
//   1. Validates sprint is in active status.
//   2. Fetches all active assignments.
//   3. Identifies "incomplete" assignments (entity status not in completed-equivalent statuses).
//   4. Based on carryoverMode:
//      a. "next": finds or creates the next planning sprint; calls ReassignToSprintTx.
//      b. "backlog": calls DropAssignmentsTx on incomplete entities.
//   5. Advances sprint status to "completed" (via UpdateStatus inside the tx).
//   6. Inserts a sprint_completions row (via CreateCompletionTx).
//   7. Commits. Any failure rolls back all changes atomically.
//
// The default carryover mode is read from config if carryoverMode == "".
func (s *SprintService) CloseSprintWithCarryover(ctx context.Context, sprintKey string, carryoverMode CarryoverMode) (*SprintCloseResult, error)

// SprintCloseResult summarizes what happened during close.
type SprintCloseResult struct {
    Sprint           *models.Sprint
    CompletedCount   int
    CarriedOverCount int
    DroppedCount     int
    NextSprintKey    string  // set when carryover=next; empty otherwise
}
```

**Transaction pattern**: Follows `.claude/rules/go/patterns.md` transaction pattern — `tx, err := s.db.BeginTx(ctx, nil)` with `defer tx.Rollback()` and explicit `tx.Commit()` at the end.

**Completed status detection**: The service calls `s.workflowSvc.GetAllStatuses()` (or an equivalent) to identify which statuses represent "completed" work. Never hardcode status strings in the service. If the workflow service does not expose a "get terminal statuses" method, use a configurable string slice injected via the service constructor (with a reasonable default of `["completed"]`).

#### 4.2.5 BulkAddToSprint (Should Have)

```go
// BulkAddInput specifies what to bulk-assign.
type BulkAddInput struct {
    SprintKey   string
    FeatureKey  string  // if set: bulk-assign tasks from this feature
    EntityTypes []string // "bug", "change_card", "tech_debt" — for type-specific bulk
}

// BulkAddToSprint assigns multiple eligible entities to a sprint in one operation.
// "Eligible" means: entity is in an assignable status AND not already in an active/planning sprint.
// Returns counts per entity type and any capacity warnings.
func (s *SprintService) BulkAddToSprint(ctx context.Context, input BulkAddInput) (*BulkAddResult, error)

type BulkAddResult struct {
    AddedByType      map[string]int  // entity_type -> count added
    SkippedByType    map[string]int  // entity_type -> count skipped (already assigned or ineligible)
    CapacityWarnings []*CapacityWarning
}
```

---

### 4.3 CLI Commands — `internal/cli/commands/sprint.go`

#### 4.3.1 New Commands to Register

Add four commands to the existing `init()` block in `sprint.go`:

```go
sprintCmd.AddCommand(sprintAddCmd)
sprintCmd.AddCommand(sprintRemoveCmd)
sprintCmd.AddCommand(sprintBacklogCmd)
// sprintCloseCmd is already registered; its RunE handler is replaced (see §4.3.5)
```

#### 4.3.2 `shark sprint add`

```
Use:   "add <sprint-key> <entity-key> [--bulk <feature-key>] [--bulk-bugs] [--bulk-tech-debt] [--bulk-changes]"
Short: "Add an entity to a sprint"
Args:  cobra.RangeArgs(1, 2)  -- sprint-key always required; entity-key required if no --bulk flag
```

Handler `runSprintAdd`:
1. Parse sprint key from `args[0]`.
2. If `--bulk <feature-key>` or `--bulk-*` flag: call `svc.BulkAddToSprint()`, format result table.
3. Otherwise: parse entity key from `args[1]`, call `svc.AddEntityToSprint()`.
4. If `CapacityWarning` non-nil: emit `cli.Warning(...)` before success message.
5. `--json` outputs the created `SprintAssignment`.

#### 4.3.3 `shark sprint remove`

```
Use:   "remove <sprint-key> <entity-key>"
Short: "Remove an entity from a sprint"
Args:  cobra.ExactArgs(2)
```

Handler `runSprintRemove`: parse keys, call `svc.RemoveEntityFromSprint()`, format output.

#### 4.3.4 `shark sprint backlog`

```
Use:   "backlog <sprint-key> [--type=task|bug|change_card|tech_debt] [--blocked]"
Short: "View all entities assigned to a sprint"
Args:  cobra.ExactArgs(1)
```

Handler `runSprintBacklog`:
1. Parse sprint key and flags.
2. Call `svc.GetSprintBacklog()`.
3. Human-readable output: header with sprint name + completion %, then one section per `BacklogGroup` with a table per group. Entity type label column uses `[task]`, `[bug]`, `[change-card]`, `[tech-debt]`.
4. `--blocked` output: table with columns Key, Type, Title, Status, Days Blocked.
5. `--json` outputs the full `SprintBacklog` struct.

#### 4.3.5 `shark sprint close` — Replace Stub

The existing `runSprintClose` in F02 is a stub that calls `svc.CloseSprint()` (status → `ready_for_review` only). Replace with:

```go
func runSprintClose(cmd *cobra.Command, args []string) error {
    sprintKey := args[0]
    carryoverFlag, _ := cmd.Flags().GetString("carryover")

    svc := getSprintService()
    result, err := svc.CloseSprintWithCarryover(cmd.Context(), sprintKey, services.CarryoverMode(carryoverFlag))
    if err != nil {
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    cli.Success(fmt.Sprintf("Closed sprint %s", result.Sprint.Key))
    cli.Info(fmt.Sprintf("  Completed: %d  Carried over: %d  Dropped: %d",
        result.CompletedCount, result.CarriedOverCount, result.DroppedCount))
    if result.NextSprintKey != "" {
        cli.Info(fmt.Sprintf("  Incomplete entities moved to: %s", result.NextSprintKey))
    }
    return nil
}
```

Add flag to `sprintCloseCmd`:
```go
sprintCloseCmd.Flags().String("carryover", "", "Carryover mode: next or backlog (default from config)")
```

The `sprintServicer` interface in `sprint.go` must be extended with the new methods:

```go
type sprintServicer interface {
    // ... existing from F02 ...
    AddEntityToSprint(ctx context.Context, input services.AddEntityInput) (*models.SprintAssignment, *services.CapacityWarning, error)
    RemoveEntityFromSprint(ctx context.Context, sprintKey, entityKey string) error
    GetSprintBacklog(ctx context.Context, sprintKey string, opts services.BacklogOptions) (*services.SprintBacklog, error)
    CloseSprintWithCarryover(ctx context.Context, sprintKey string, carryoverMode services.CarryoverMode) (*services.SprintCloseResult, error)
    BulkAddToSprint(ctx context.Context, input services.BulkAddInput) (*services.BulkAddResult, error)
}
```

---

### 4.4 Service Accessor — `internal/cli/services_global.go`

Update `GetSprintService()` to pass the database connection for transaction support:

```go
func GetSprintService() *services.SprintService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    sprintRepo := repository.NewSprintRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewSprintService(sprintRepo, workflowSvc, db)  // db added
}
```

Update `NewSprintService` signature in `internal/services/sprint_service.go`:

```go
func NewSprintService(repo SprintRepository, workflowSvc *workflow.Service, db *dbconn.DB) *SprintService {
    // ... existing nil guards ...
    return &SprintService{repo: repo, workflowSvc: workflowSvc.ForLevel("sprint"), db: db}
}
```

---

### 4.5 Config Support — `internal/config/config.go`

Add `sprint_defaults.carryover` to the config struct. See existing pattern for `sprint_defaults` if already defined by E19-F02; if not:

```go
// In SprintDefaults struct (create or extend):
type SprintDefaults struct {
    Carryover string `mapstructure:"carryover" json:"carryover"` // "next" | "backlog"
    // future: AutoCreate, Capacity
}

// Config struct addition:
SprintDefaults SprintDefaults `mapstructure:"sprint_defaults" json:"sprint_defaults"`
```

Default: if `SprintDefaults.Carryover == ""`, the service defaults to `CarryoverNext`.

---

## 5. Key Technical Decisions

| Decision | Choice | Rationale |
|---|---|---|
| UNION query for backlog | Static UNION ALL across four tables | Parameterized queries required; string-interpolated table names would be a SQL injection risk. The UNION approach handles all four entity types without dynamic SQL. |
| Entity ID resolution via separate typed methods | `GetTaskIDByKey`, `GetBugIDByKey`, etc. | Same rationale as UNION — avoids dynamic table names. |
| Transaction ownership in service | `CloseSprintWithCarryover` opens `*sql.Tx`, passes to `Tx` variants | Follows `.claude/rules/go/patterns.md` transaction pattern; service owns the boundary, repository exposes `*Tx` methods. |
| Capacity warning is advisory, not blocking | Return `*CapacityWarning` alongside assignment, never reject | REQ-F-004 explicitly says advisory. Hard enforcement deferred to E19-F05. |
| Replace F02 `CloseSprint` stub | Replace `runSprintClose` handler and `CloseSprint` → `CloseSprintWithCarryover` | The F02 spec explicitly deferred carryover to F03. The stub status transition is absorbed into `CloseSprintWithCarryover`. |
| Completed-status detection via workflow.Service | Ask workflow service, never hardcode | Follows CLAUDE.md `.claude/rules/go/patterns.md` — "never hardcode status lists". |
| `sprint_completions` table | New table, new migration, version bump 18→19 | Completion record required for both REQ-F-006 and analytics (E19-F04 reads from it). No existing table is suitable. |

---

## 6. Integration with Existing Code

| File | Change |
|---|---|
| `internal/db/db.go` | Add `migrateSprintCompletionsTable()`, call from `runMigrations()`, bump `CurrentSchemaVersion` 18→19. |
| `internal/models/sprint.go` | Add `SprintCompletion` struct. |
| `internal/repository/sprint/repository.go` | Add assignment, backlog, completion, and entity-ID-lookup methods (§4.1). |
| `internal/services/sprint_service.go` | Extend `SprintRepository` interface; add `db` field and updated constructor; add `AddEntityToSprint`, `RemoveEntityFromSprint`, `GetSprintBacklog`, `CloseSprintWithCarryover`, `BulkAddToSprint` methods; define DTOs (`AddEntityInput`, `BacklogOptions`, `SprintBacklog`, etc.). |
| `internal/cli/commands/sprint.go` | Add `sprintAddCmd`, `sprintRemoveCmd`, `sprintBacklogCmd`; replace `runSprintClose`; extend `sprintServicer` interface; add `--carryover` flag to `sprintCloseCmd`. |
| `internal/cli/services_global.go` | Update `GetSprintService()` to pass `db` to constructor. |
| `internal/config/config.go` | Add `SprintDefaults.Carryover` field. |
| `internal/config/config_test.go` | Test new config field round-trips correctly. |

---

## 7. Test Plan

### 7.1 Repository Tests — `internal/repository/sprint/repository_test.go`

Use real test database (`test.GetTestDB()`). Follow pattern from existing `repository_test.go`.

| Test | Asserts |
|---|---|
| `TestAddAssignment_Success` | Inserts row; `GetActiveAssignment` returns it. |
| `TestAddAssignment_DuplicateActiveReturnsError` | Inserting a second active assignment for the same entity fails with a unique-constraint-derived error. |
| `TestAddAssignment_AllowsAfterRemoval` | After `RemoveAssignment`, a second `AddAssignment` succeeds (removed_at was set). |
| `TestRemoveAssignment_SetsRemovedAt` | `removed_at` is populated; `GetActiveAssignment` returns nil afterward. |
| `TestRemoveAssignment_NotFoundReturnsError` | Removing a non-existent assignment returns an error. |
| `TestListAssignments_FilterByEntityType` | With `entityType="task"`, only task rows returned. |
| `TestListAssignments_IgnoresSoftDeleted` | Rows with `removed_at` set are excluded. |
| `TestListBacklog_GroupsCorrectly` | Backlog items from tasks/bugs returned with correct entity_type labels. |
| `TestListBacklog_BlockedOnlyFilter` | Non-blocked entities excluded when `blockedOnly=true`. |
| `TestListAssignmentsForCarryover_ExcludesCompleted` | Entities in completed status are not returned. |
| `TestReassignToSprintTx_UpdatesSprintID` | sprint_id is updated for all listed assignment IDs. |
| `TestDropAssignmentsTx_SetsRemovedAt` | All listed assignment IDs have `removed_at` populated. |
| `TestCreateCompletionTx_InsertsRow` | `sprint_completions` row is queryable after commit. |
| `TestGetTaskIDByKey_Found` | Returns correct ID. |
| `TestGetTaskIDByKey_NotFound` | Returns error. |
| `TestMigrateSprintCompletionsTable_Idempotent` | Running twice does not error. |
| `TestSchemaVersionBumpedTo19` | `getSchemaVersion(db) == 19` after migration. |

### 7.2 Service Tests — `internal/services/sprint_service_test.go`

Use mock repository (function-field pattern per `.claude/rules/services/testing.md`).

| Test | Asserts |
|---|---|
| `TestAddEntityToSprint_Task_Success` | Parses task key, resolves task ID, calls `AddAssignment`, returns assignment. |
| `TestAddEntityToSprint_Bug_Success` | Same, bug variant. |
| `TestAddEntityToSprint_CC_Success` | Same, change_card variant. |
| `TestAddEntityToSprint_TD_Success` | Same, tech_debt variant. |
| `TestAddEntityToSprint_AlreadyAssigned_ReturnsConflict` | `GetActiveAssignment` returns non-nil; error returned with conflicting sprint key. |
| `TestAddEntityToSprint_SprintNotPlanningOrActive_Rejects` | Sprint in `completed` status; operation rejected. |
| `TestAddEntityToSprint_CapacityWarning` | Allocation exceeds capacity; warning returned, assignment created. |
| `TestRemoveEntityFromSprint_Success` | Calls `RemoveAssignment` with correct args. |
| `TestRemoveEntityFromSprint_NoActiveAssignment_ReturnsError` | `GetActiveAssignment` returns nil; error returned. |
| `TestGetSprintBacklog_GroupsByStatus` | Items grouped into correct status categories. |
| `TestGetSprintBacklog_CompletionPercent` | Math is correct (3/5 → 60%). |
| `TestGetSprintBacklog_TypeFilter` | Only matching entity type returned. |
| `TestCloseSprintWithCarryover_Next_CreatesNextSprint` | When no next sprint exists and `carryoverMode=next`, `CreateSprint` is called, incomplete assignments are reassigned. |
| `TestCloseSprintWithCarryover_Next_UsesExistingPlanningSpint` | When a planning sprint exists, uses it; does not create another. |
| `TestCloseSprintWithCarryover_Backlog_DropsIncomplete` | `DropAssignmentsTx` called for incomplete; completed assignments untouched. |
| `TestCloseSprintWithCarryover_Rollback_OnError` | If any step fails, transaction is rolled back (mock tx returns error). |
| `TestCloseSprintWithCarryover_CompletionRecordCreated` | `CreateCompletionTx` called with correct counts. |
| `TestCloseSprintWithCarryover_DefaultModeFromConfig` | Empty `carryoverMode` reads from `SprintDefaults.Carryover`. |
| `TestBulkAddToSprint_FeatureKey_AddsEligible` | Only tasks not already assigned are added. |
| `TestBulkAddToSprint_BugsBulk` | Correct entity_type used. |

### 7.3 CLI Tests — `internal/cli/commands/sprint_test.go`

Use mock `sprintServicer`. Follow pattern from existing `sprint_test.go`.

| Test | Asserts |
|---|---|
| `TestSprintAdd_SingleEntity_CallsService` | Parses sprint key + entity key; calls `AddEntityToSprint`. |
| `TestSprintAdd_CapacityWarning_PrintsWarning` | Warning emitted when `CapacityWarning` non-nil. |
| `TestSprintAdd_JSON_OutputsAssignment` | JSON flag produces `SprintAssignment` JSON. |
| `TestSprintAdd_BulkFlag_CallsBulkService` | `--bulk E07-F01` calls `BulkAddToSprint`. |
| `TestSprintRemove_CallsService` | Correct args passed. |
| `TestSprintBacklog_FormatsGroups` | Human-readable output contains status group headers. |
| `TestSprintBacklog_BlockedFilter` | `--blocked` flag passes `BlockedOnly=true`. |
| `TestSprintBacklog_JSON` | JSON flag outputs `SprintBacklog`. |
| `TestSprintClose_CarryoverNext` | `--carryover=next` passes correct enum; output contains next sprint key. |
| `TestSprintClose_CarryoverBacklog` | `--carryover=backlog`; no next sprint key in output. |
| `TestSprintClose_DefaultCarryover_NoFlag` | No `--carryover` flag passes empty string to service. |

---

## 8. Acceptance Criteria Summary

Mapped 1:1 to the epic PRD acceptance criteria for REQ-F-004, REQ-F-005, REQ-F-006:

- [ ] **AC-1** — All four entity types assignable/removable via `shark sprint add/remove`.
- [ ] **AC-2** — Double-assignment to an active sprint blocked by partial unique index and returns descriptive error with conflicting sprint key.
- [ ] **AC-3** — `shark sprint backlog` groups all entity types by status; shows entity type label per row.
- [ ] **AC-4** — Completion percentage displayed in backlog header.
- [ ] **AC-5** — `--type=` filter works for backlog.
- [ ] **AC-6** — `--blocked` filter works for backlog with days-blocked shown.
- [ ] **AC-7** — `--json` produces correct output with `entity_type` field for backlog items.
- [ ] **AC-8** — `shark sprint close S024 --carryover=next` atomically moves incomplete entities and creates next sprint if needed.
- [ ] **AC-9** — `shark sprint close S024 --carryover=backlog` atomically soft-deletes incomplete assignments.
- [ ] **AC-10** — Completed entities remain attached to closed sprint after close.
- [ ] **AC-11** — `sprint_completions` row created on every sprint close.
- [ ] **AC-12** — Default carryover mode read from `.sharkconfig.json`.
- [ ] **AC-13** — `make fmt && make lint && make test` all pass.
- [ ] **AC-14** — `CurrentSchemaVersion` is 19; history comment updated.

---

## 9. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| UNION query becomes unwieldy when a 5th entity type is added | Low | Each sub-select is short and templated; adding a new entity type is one new SELECT block. Document the extension point in a comment above the SQL. |
| `CloseSprintWithCarryover` transaction holds lock too long | Low | SQLite WAL mode allows concurrent reads; the lock is write-only. 200 entities is well within sub-2s target. |
| `CloseSprint` (F02 stub) used in CLI tests that assume `ready_for_review` status | Medium | Replace `CloseSprint` stub entirely and update any F02 tests that relied on its behavior. The F02 service test for `CloseSprint` must be updated to use `CloseSprintWithCarryover`. |
| Forgetting to bump `CurrentSchemaVersion` | High (common mistake per `database-critical.md`) | `TestSchemaVersionBumpedTo19` is a hard fail gate. |
| `GetSprintService()` not passing `db` after constructor update | Medium | Update `services_global.go` atomically with the constructor signature change; compile error catches it. |

---

*End of spec.*
