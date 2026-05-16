---
feature_key: E19-F07-sprint-relative-ordering-for-backlog-items
epic_key: E19
title: Sprint-relative ordering for backlog items
spec_version: 1.0
last_updated: 2026-05-16
complexity: STANDARD
---

# Spec — E19-F07: Sprint-Relative Ordering for Backlog Items

> **Scope guard.** This feature adds a sprint-scoped ordering column on `sprint_assignments`, makes it the primary sort key in `GetNextTask`, exposes a `sprint reorder` command, adds an `--at=<position>` flag to `sprint add`, and surfaces ordering in `sprint backlog --view=ordered`. Out of scope: cross-sprint reordering, drag-and-drop UI, auto-ordering by dependencies or readiness scores, sparse/fractional ordering schemes.

---

## 1. Requirements

### 1.1 Trace to Epic PRD

This feature is a refinement of REQ-F-004 (Individual Entity Assignment) and REQ-F-005 (Sprint Backlog View) from [E19 requirements.md](../requirements.md). It also modifies the selection algorithm used by `shark sprint next` (delivered in E19-F06 / consumed by `GetNextTask`) without introducing a new top-level requirement in the epic — the change is internal to the sort precedence.

NFRs that apply unchanged: REQ-NF-001 (response time), REQ-NF-002 (query efficiency), REQ-NF-003 (data integrity), REQ-NF-004 (backward compatibility), REQ-NF-005 (JSON output consistency).

### 1.2 Functional Requirements (incremental over the epic)

| ID | Requirement | Priority |
|---|---|---|
| REQ-F-001 | Add nullable `sprint_order` column to `sprint_assignments` and use it as primary sort key in `GetNextTask` | Must Have |
| REQ-F-002 | `sprint add` auto-assigns `sprint_order = max(existing) + 1` and accepts `--at=<position>` to insert at a specific 1-based position | Must Have |
| REQ-F-003 | `sprint reorder <sprint> <entity> <position>` moves an item to position N and renumbers densely; `--top` and `--bottom` shortcuts supported | Must Have |
| REQ-F-004 | `sprint next --json` returns `sprint_order`, `sprint_key`, and `selection_reason` fields so AI agents can reason about ordering intent | Must Have |
| REQ-F-005 | `sprint backlog --view=ordered` shows a numbered pull queue; ordered view is the default for active sprints, grouped view is the default for planning/completed sprints | Must Have |
| REQ-F-006 | `sprint close --carryover=next` preserves relative order of carried-over assignments and renumbers from 1 in the receiving sprint; `--carryover=backlog` clears `sprint_order`; carryover JSON output includes `order_preserved` field | Must Have |
| REQ-F-007 | Migration backfills `sprint_order` for existing planning/active sprints in `assigned_at` order | Must Have |
| REQ-F-008 | Editing entity-level `priority` or `ExecutionOrder` does NOT change `sprint_order` (no cross-mutation); `sprint reorder` does NOT mutate entity-level fields | Must Have |
| REQ-F-009 | After `sprint start`, if any assignment in the sprint has `sprint_order = NULL`, surface a soft warning naming the count; do not block start | Should Have |

### 1.3 Acceptance Criteria

**REQ-F-001 — Sort precedence (binding for `GetNextTask`)**

- [ ] Sort precedence is, in order: (1) `sprint_order ASC NULLS LAST`, (2) `ExecutionOrder ASC NULLS LAST`, (3) `priority ASC` (lower number = higher priority — matches existing `GetNextTask` semantics), (4) `assigned_at ASC` (FIFO).
- [ ] When two candidates tie on every factor, the comparator is stable (insertion order preserved by `sort.SliceStable` — see §3.4).
- [ ] An item with `sprint_order = NULL` is selected only when no eligible candidate has `sprint_order` set.
- [ ] The agent-type filter (existing `--agent` flag) still applies before sorting.

**REQ-F-002 — `sprint add --at`**

- [ ] `shark sprint add S024 E07-F01-001` (no `--at`) assigns `sprint_order = max(sprint_order WHERE sprint_id = S024.id AND sprint_order IS NOT NULL) + 1`. If no ordered items exist yet, the new item gets `sprint_order = 1`.
- [ ] `shark sprint add S024 E07-F01-001 --at=1` places the new item at position 1 and shifts all existing items at positions 1..N to 2..N+1 within the same sprint.
- [ ] `--at=N` where `N == count(ordered_items) + 1` is equivalent to no flag (append).
- [ ] `--at=N` where `N > count(ordered_items) + 1` returns a validation error: `"position N is out of range (sprint has K ordered items)"` and the assignment is NOT created.
- [ ] `--at=0` and `--at=-N` return a validation error: `"position must be >= 1"`.
- [ ] Both human and `--json` output include the assigned `sprint_order` value.
- [ ] `--at` has no effect on bulk-add paths (`--bulk`, `--bulk-bugs`, `--bulk-tech-debt`, `--bulk-changes`); bulk inserts use the FIFO append behavior. Mutually-exclusive validation: passing both `--at` and any bulk flag returns an error.

**REQ-F-003 — `sprint reorder`**

- [ ] `shark sprint reorder S024 E07-F01-001 3` moves the entity to position 3 and densely renumbers the remaining ordered items so positions are gapless (1..N).
- [ ] `shark sprint reorder S024 E07-F01-001 --top` is equivalent to `--at=1` semantics; positions all other ordered items at 2..N+1 if needed (densely renumbered).
- [ ] `shark sprint reorder S024 E07-F01-001 --bottom` assigns `sprint_order = max(existing) + 1` after dense renumbering of others (no gaps left at the previous position).
- [ ] `<position>`, `--top`, `--bottom` are mutually exclusive; passing more than one returns a CLI parse error.
- [ ] If the entity is not assigned to the sprint: returns error `"entity %q is not assigned to sprint %s"` (exit code 3 — invalid state).
- [ ] If the entity has `sprint_order = NULL`: the reorder still works — it inserts at the requested position and renumbers (the entity transitions from unordered to ordered).
- [ ] If `<position>` > `count(ordered_items_after_target_excluded) + 1`: returns validation error.
- [ ] After a successful reorder, the abbreviated pull queue (top 8 items by `sprint_order`) is printed to stdout in human mode; `--json` output returns the updated assignment with the new `sprint_order` and a top-N summary.
- [ ] No entity-level fields (`priority`, `ExecutionOrder`) are mutated.

**REQ-F-004 — `sprint next --json` shape**

- [ ] `sprint next --json` response includes:
  - `key` (entity key — already present)
  - `entity_type` (already present)
  - `title`, `status`, `agent_type`, `priority`, `size`, `assigned_at`, `execution_order` (already present)
  - **NEW** `sprint_order` (integer, nullable — null when the selected item is unordered)
  - **NEW** `sprint_key` (string — the active sprint's key)
  - **NEW** `selection_reason` (enum string — exactly one of `"sprint_order"`, `"execution_order"`, `"priority"`, `"assigned_at"` — the first sort factor that broke the tie against the next-best candidate; defaults to `"assigned_at"` when there is only one candidate)
- [ ] Human-mode output (no `--json`) adds two new lines below the existing `Next Task:` block:
  - `  Sprint position: N of K  |  Selected by: <reason>` where K is the count of eligible candidates after the agent filter (not the total sprint backlog)
  - `  Sprint: <sprint_key>` (when the sprint has a `name`, append ` (<name>)`)
- [ ] When `--field sprint_order` is used and the selected item is unordered, the CLI returns the JSON literal `null` (not an error, exit 0).

**REQ-F-005 — `sprint backlog --view`**

- [ ] `shark sprint backlog <sprint>` defaults to `--view=ordered` when the sprint is in `active` status; defaults to `--view=grouped` for any other sprint status (`planning`, `completed`, `archived`).
- [ ] `--view=ordered` output (human mode) renders a single table sorted by `sprint_order ASC NULLS LAST` with columns: `#`, `KEY`, `TYPE`, `STATUS`, `AGENT`, `TITLE`. Position column displays the integer `sprint_order` for ordered items and the literal `~` (tilde) for unordered items. Items in terminal status (per workflow) are omitted by default; `--include-completed` adds them back.
- [ ] `--view=ordered` blocked items are prefixed with `!` in the position column (e.g., `!2`) when their workflow status falls in the `blocked` phase. The existing `--blocked` filter still works in either view.
- [ ] `--view=grouped` retains the current behavior unchanged (groups by status category).
- [ ] `--json` output of `--view=ordered` returns a flat array of items in sort order, each item including a `position` field (1-based dense rank computed at view time) AND `sprint_order` field (raw stored value, nullable). `position` and `sprint_order` are always equal for items with non-null `sprint_order` in a fully-ordered sprint; they diverge only when null values are present (unordered items get `position` but `sprint_order = null`).
- [ ] All `--json` output of `sprint backlog` (regardless of `--view`) MUST include `sprint_order` on every item — see Decision §3.5.

**REQ-F-006 — Carryover behavior**

- [ ] `sprint close --carryover=next`: the receiving sprint's existing ordered items keep their positions; carried-over items are appended after them in the order `(sprint_order ASC NULLS LAST, assigned_at ASC)` of the closing sprint, starting at `max(receiving.sprint_order) + 1` and densely numbered.
- [ ] `sprint close --carryover=backlog`: the soft-deleted assignments have `sprint_order` set to NULL in the same `UPDATE` that sets `removed_at` (see §3.3 transaction).
- [ ] `SprintCloseResult` JSON gains a `carryover_preserved` boolean field (true for `next`, false for `backlog`). The CLI also prints `Order preserved: yes|no` in human mode.
- [ ] If the receiving sprint had `NULL` for any of its existing items, those remain `NULL` (not renumbered). Only the carried-over items get fresh sprint_order values.

**REQ-F-007 — Migration**

- [ ] `migrateSprintAssignmentsAddSprintOrder` adds the column via `ALTER TABLE sprint_assignments ADD COLUMN sprint_order INTEGER` (idempotent — checks `pragma_table_info` first, mirroring `migrateSlugColumns`).
- [ ] After adding the column, the migration backfills: for each sprint in status `planning` or `active`, assigns `sprint_order = ROW_NUMBER() OVER (PARTITION BY sprint_id ORDER BY assigned_at, id)` for active assignments only (`removed_at IS NULL`). Sprints in `completed`/`archived` status are NOT backfilled — their items stay NULL.
- [ ] Migration creates a partial unique index `idx_sprint_assignments_order_unique ON sprint_assignments(sprint_id, sprint_order) WHERE sprint_order IS NOT NULL AND removed_at IS NULL` to prevent duplicate positions per active sprint.
- [ ] `CurrentSchemaVersion` is bumped from 19 → 20 in the same commit.
- [ ] Migration is rerunnable: a second run is a no-op (column already exists; backfill skipped because the first run set non-NULL values).

**REQ-F-008 — Decoupling from entity-level fields**

- [ ] Manually changing `tasks.priority` or `tasks.execution_order` (via `shark task update` or direct DB writes) leaves all `sprint_assignments.sprint_order` values unchanged.
- [ ] `shark sprint reorder` does NOT issue any UPDATE on entity tables (`tasks`, `bugs`, `change_cards`, `tech_debts`).
- [ ] Adding the same entity to a different sprint later (after removal/carryover-to-backlog) does NOT recall the old `sprint_order` — each sprint's ordering is independent.

**REQ-F-009 — Soft warning on `sprint start` (Should Have)**

- [ ] When `shark sprint start <key>` succeeds, if `count(active assignments WHERE sprint_order IS NULL AND sprint_id = key.id) > 0`, a single warning line is emitted: `"Warning: N items have no sprint order. Use \`shark sprint reorder\` to set pull priority."`
- [ ] The warning does NOT prevent the sprint from starting (status transition succeeds).
- [ ] The warning is omitted in `--json` mode (the JSON contract is unchanged).

### 1.4 Non-Functional Requirements

- **Performance** (REQ-NF-001): `sprint add --at=N` and `sprint reorder` complete in <500ms for sprints with up to 50 ordered items. The renumber UPDATE is a single statement (CASE WHEN expression) inside a transaction.
- **Query efficiency** (REQ-NF-002): The new `idx_sprint_assignments_order_unique` partial index covers the `ORDER BY sprint_order` clause in `GetNextTask`'s candidate sort. The existing `idx_sprint_assignments_sprint` continues to cover the `WHERE sprint_id = ?` lookup.
- **Backward compatibility** (REQ-NF-004): Existing `sprint add` invocations without `--at` continue to work and assign `sprint_order` automatically. Existing `sprint backlog` invocations on non-active sprints continue to use the grouped view. Existing `sprint next --json` consumers tolerate the additional fields (additive change only — no field removed or renamed).
- **JSON consistency** (REQ-NF-005): All new fields on `sprint next`, `sprint backlog --view=ordered`, `sprint reorder`, and `SprintCloseResult` use snake_case (`sprint_order`, `sprint_key`, `selection_reason`, `carryover_preserved`).

### 1.5 Out of Scope for This Feature

- Cross-sprint reordering (move item from sprint A to sprint B with position preserved).
- Drag-and-drop UI or web-based reordering.
- Auto-ordering based on dependency graph or readiness scores.
- Sparse / fractional indexing (e.g., 1.5 between 1 and 2). The spec is explicit about dense renumbering — see §3.4.
- Reordering within a sprint by `--include-completed` items: completed items are omitted from the ordered view by default and may not be reordered (their position is frozen at completion time).
- A `sprint reorder --swap A B` operation. Position-based moves cover the use case; pairwise swap adds complexity without value (per CX feedback §3.5).
- A `blocked_by` array in `sprint next --json` output. The CX feedback §2.4 leaves room for this in a future feature; this spec does NOT implement it but does NOT break the additive contract.
- Mutating `sprint_order` from any code path other than `sprint add`, `sprint reorder`, `sprint close --carryover=next`, and `sprint close --carryover=backlog`. In particular, `BulkAddToSprint` MUST NOT bypass the auto-numbering (see §3.6).

---

## 2. Foundational State (Delivered by E19-F01, F02, F03, F05, F06)

The following are present and must NOT be recreated:

- **Schema**: `sprint_assignments` (polymorphic with partial unique index `idx_sprint_assignments_active_one`), `sprint_capacity`, `sprints`, `sprint_completions` tables — `internal/db/db.go` `migrateSprintTables` (schema version 18) and `migrateSprintCompletionsTable` (schema version 19).
- **Models**: `Sprint`, `SprintAssignment`, `SprintCapacity`, `SprintCompletion` — `internal/models/sprint.go`. `SprintAssignment.Validate()` already handles structural checks; this feature adds an optional `SprintOrder *int` field — see §3.1.
- **Repository**: `*sprint.SprintRepository` with `AddAssignment`, `RemoveAssignment`, `GetActiveAssignment`, `ListAssignments`, `ListBacklog`, `ListAssignmentsForCarryover`, `ReassignToSprintTx`, `DropAssignmentsTx`, `BulkAssign`, `ListUnassignedBacklog`, `GetAssignmentsWithSize` — `internal/repository/sprint/repository.go`.
- **Service**: `SprintService.AddEntityToSprint`, `RemoveEntityFromSprint`, `GetSprintBacklog`, `BulkAddToSprint`, `GetNextTask`, `CloseSprintWithCarryover` — `internal/services/sprint_service.go`.
- **CLI**: `sprintAddCmd`, `sprintRemoveCmd`, `sprintBacklogCmd`, `sprintCloseCmd`, `sprintNextCmd`, `sprintStartCmd` — `internal/cli/commands/sprint.go`. The `sprintAssignmentServicer` interface (lines 38–45) is the binding for new commands.

---

## 3. Architecture

### 3.1 Data model changes

**Single ALTER TABLE on `sprint_assignments`**:

```sql
ALTER TABLE sprint_assignments ADD COLUMN sprint_order INTEGER;
```

- **Nullable** by design: tracks "intentionally unordered" vs "ordered". `0` is reserved (positions are 1-based) and never written.
- **Stored as INTEGER**, dense (gapless within an active sprint after any `reorder`).
- **NOT a foreign key**, no CHECK constraint at the DB level (consistent with the post-B018 convention referenced in `internal/db/db.go:3361`).

**New partial unique index**:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_sprint_assignments_order_unique
    ON sprint_assignments(sprint_id, sprint_order)
    WHERE sprint_order IS NOT NULL AND removed_at IS NULL;
```

- Mirrors the `idx_sprint_assignments_active_one` partial-index pattern.
- Catches programming errors (two active rows in the same sprint sharing a position) at the DB layer.

**Migration function**: `migrateSprintAssignmentsAddSprintOrder` in `internal/db/db.go`. Wired into `runMigrations()` after `migrateSprintCompletionsTable`. Bumps `CurrentSchemaVersion` from `19` → `20`. See `.claude/rules/database-critical.md` — bumping the version constant is the only required step to make the migration run on existing databases.

**Backfill SQL** (executed once during the migration if `sprint_assignments.sprint_order IS NULL` for every row in any planning/active sprint):

```sql
WITH numbered AS (
  SELECT
    sa.id,
    ROW_NUMBER() OVER (PARTITION BY sa.sprint_id ORDER BY sa.assigned_at, sa.id) AS rn
  FROM sprint_assignments sa
  JOIN sprints s ON s.id = sa.sprint_id
  WHERE sa.removed_at IS NULL
    AND s.status IN ('planning', 'active')
    AND sa.sprint_order IS NULL
)
UPDATE sprint_assignments
SET sprint_order = (SELECT rn FROM numbered WHERE numbered.id = sprint_assignments.id)
WHERE id IN (SELECT id FROM numbered);
```

SQLite supports `ROW_NUMBER()` window functions since 3.25 (2018) and Turso supports them. The Go side uses `db.ExecContext` with this static SQL — no string interpolation.

**Model change** (`internal/models/sprint.go`):

```go
type SprintAssignment struct {
    ID          int64      `json:"id" db:"id"`
    SprintID    int64      `json:"sprint_id" db:"sprint_id"`
    EntityType  string     `json:"entity_type" db:"entity_type"`
    EntityID    int64      `json:"entity_id" db:"entity_id"`
    AssignedAt  time.Time  `json:"assigned_at" db:"assigned_at"`
    RemovedAt   *time.Time `json:"removed_at,omitempty" db:"removed_at"`
    SprintOrder *int       `json:"sprint_order,omitempty" db:"sprint_order"` // NEW — nullable
}
```

`Validate()` does NOT need to validate `SprintOrder` (>=1 is a service-level check; the partial unique index catches duplicates).

### 3.2 API / interface contracts

**`SprintAssignment` JSON shape** (additive — `sprint_order` field appears in every place `SprintAssignment` is serialized):

```json
{
  "id": 42,
  "sprint_id": 5,
  "entity_type": "task",
  "entity_id": 101,
  "assigned_at": "2026-05-15T10:00:00Z",
  "sprint_order": 3
}
```

**`AddEntityInput` DTO** in `internal/services/sprint_service.go` gains an optional `Position *int` field (nil = append):

```go
type AddEntityInput struct {
    SprintKey     string
    EntityKey     string
    AgentType     string
    EstimatedSize int
    Position      *int   // NEW — nil means "append at end"; 1-based when set
}
```

**New service method** `SprintService.ReorderAssignment`:

```go
type ReorderTarget struct {
    Position *int  // nil if Top or Bottom is set
    Top      bool
    Bottom   bool
}

func (s *SprintService) ReorderAssignment(
    ctx context.Context,
    sprintKey, entityKey string,
    target ReorderTarget,
) (*models.SprintAssignment, []*models.SprintAssignment, error)
```

Returns: the moved assignment with its new `sprint_order`, and the abbreviated top-N list (N=8 by default — matches the CX feedback §3.5 sample) for display. Errors: not-found, invalid position, mutually-exclusive target violation.

**`SprintCloseResult`** gains one new field (additive):

```go
type SprintCloseResult struct {
    Sprint              *models.Sprint
    CompletedCount      int
    CarriedOverCount    int
    DroppedCount        int
    NextSprintKey       string
    CarryoverPreserved  bool   // NEW — true for "next", false for "backlog"
}
```

**`BacklogItemView`** gains two new fields (additive):

```go
type BacklogItemView struct {
    EntityType     string    `json:"entity_type"`
    Key            string    `json:"key"`
    Title          string    `json:"title"`
    Status         string    `json:"status"`
    AgentType      string    `json:"agent_type,omitempty"`
    Priority       int       `json:"priority,omitempty"`
    ExecutionOrder *int      `json:"execution_order,omitempty"`
    Size           *int      `json:"size,omitempty"`
    AssignedAt     time.Time `json:"assigned_at,omitempty"`
    DaysBlocked    int       `json:"days_blocked,omitempty"`
    SprintOrder    *int      `json:"sprint_order,omitempty"` // NEW
    SprintKey      string    `json:"sprint_key,omitempty"`   // NEW (set by GetNextTask only; backlog views don't need it)
    SelectionReason string   `json:"selection_reason,omitempty"` // NEW (set by GetNextTask only)
}
```

`SprintBacklog` struct gains a `View string` field (`"ordered"` or `"grouped"`) so JSON consumers can confirm which view was rendered. The flat-array `Items []*OrderedBacklogItem` field appears on `--view=ordered` only; `Groups` continues to appear for `--view=grouped`. Both views always include `sprint_order` on every item per §3.5.

### 3.3 Component changes

| File | Change |
|---|---|
| `internal/db/db.go` | Add `migrateSprintAssignmentsAddSprintOrder` function near line 3593 (after `migrateSprintCompletionsTable`). Wire into `runMigrations()`. Bump `CurrentSchemaVersion` constant from 19 to 20. Update the version comment block (line ~440). |
| `internal/models/sprint.go` | Add `SprintOrder *int` field to `SprintAssignment` struct. No change to `Validate()`. |
| `internal/repository/sprint/repository.go` | (1) Update `sprintSelectColumns`-equivalent for assignments (lines ~331–349) to include `sprint_order`. (2) Update `AddAssignment`, `GetActiveAssignment`, `ListAssignments`, `ListAssignmentsForCarryover` SCAN/INSERT to include the new column. (3) Add new methods (see signatures below). (4) Update `ListBacklog` UNION-ALL to project `sa.sprint_order` (a simple column added to each sub-select; no per-entity-table change). (5) Update `BulkAssign` to use a per-row position computed via `(SELECT COALESCE(MAX(sprint_order), 0) + ROW_NUMBER() OVER (ORDER BY ...) FROM sprint_assignments WHERE sprint_id = ? AND removed_at IS NULL)` — see §3.6. |
| `internal/services/sprint_service.go` | (1) Update `AddEntityToSprint` to accept `Position` and call new repo method. (2) Add `ReorderAssignment` method. (3) Update `GetNextTask` sort comparator (see §3.4) and populate `SprintOrder`, `SprintKey`, `SelectionReason` on the returned `BacklogItemView`. (4) Update `CloseSprintWithCarryover` carryover branches to renumber/clear `sprint_order` (see §3.7) and set `CarryoverPreserved` on result. (5) Update `GetSprintBacklog` to support `--view=ordered` and project `SprintOrder` onto every item. (6) Update `StartSprint` (or wherever the start transition lives) to count NULL `sprint_order` rows and return a soft warning string in the result. |
| `internal/cli/commands/sprint.go` | (1) Add `--at=<int>` flag to `sprintAddCmd` and pipe into `AddEntityInput.Position`; reject combinations with bulk flags. (2) Add new `sprintReorderCmd` (`shark sprint reorder <sprint> <entity> [<position>] [--top] [--bottom]`). (3) Add `--view=ordered|grouped` and `--include-completed` flags to `sprintBacklogCmd`; default `--view=ordered` when sprint is active (decided in `runSprintBacklog` after fetching sprint). (4) Update `runSprintNext` human output to show position and selection reason (see §3.4). (5) Update `runSprintStart` to print the soft warning when present. (6) Update `runSprintClose` to print `Order preserved: yes|no`. |
| `internal/cli/commands/sprint_test.go` | Add unit tests for new flags / commands using the existing `sprintAssignmentServicer` mock pattern. |
| `internal/services/sprint_service_test.go` | Add table-driven tests for `ReorderAssignment`, the new sort precedence in `GetNextTask`, and the carryover renumbering paths. |
| `internal/repository/sprint/repository_test.go` | Add tests for the migration backfill, `AddAssignment` with explicit `sprint_order`, the new partial unique index, and the renumber-on-reorder UPDATE. |
| `internal/db/sprint_tables_migration_test.go` | Add a test for `migrateSprintAssignmentsAddSprintOrder` covering: column added, index created, idempotency, backfill correctness for planning/active sprints, no-op for completed/archived sprints. |
| `internal/cli/commands/sprint.go` (init function) | Register `sprintReorderCmd` under `sprintCmd`. |

**New repository methods** (add to the `SprintRepository` interface in `sprint_service.go` and to `*sprint.SprintRepository`):

```go
// SetSprintOrderTx assigns sprint_order = newPosition for the given assignment ID.
// Used as the final step of insert-at and reorder; no renumbering of siblings.
SetSprintOrderTx(ctx context.Context, tx *sql.Tx, assignmentID int64, newPosition *int) error

// RenumberAssignmentsTx accepts a sprintID and a slice of (assignment_id, new_position)
// pairs and applies them all in a single CASE WHEN UPDATE. Used by ReorderAssignment
// and by the carryover-to-next renumber path. positions of NULL entries set sprint_order
// to NULL (used by carryover-to-backlog). The slice may be empty (no-op).
RenumberAssignmentsTx(ctx context.Context, tx *sql.Tx, sprintID int64, ops []RenumberOp) error

// MaxSprintOrder returns max(sprint_order) for active assignments in the sprint, or
// 0 if no ordered items exist. Used by AddEntityToSprint when --at is omitted.
MaxSprintOrder(ctx context.Context, sprintID int64) (int, error)

// ListOrderedAssignments returns active assignments for a sprint sorted by
// sprint_order ASC NULLS LAST, assigned_at ASC. Used by ReorderAssignment to
// compute the renumber plan without re-scanning the backlog UNION.
ListOrderedAssignments(ctx context.Context, sprintID int64) ([]*models.SprintAssignment, error)
```

```go
type RenumberOp struct {
    AssignmentID int64
    NewPosition  *int   // nil → set sprint_order = NULL
}
```

These methods are pure data access; all business logic (which positions to assign, which IDs to renumber) lives in `SprintService`.

### 3.4 Sort algorithm — `GetNextTask`

Replace the comparator in `GetNextTask` (currently at `internal/services/sprint_service.go:1007–1030`) with a stable, four-tier comparator. Use `sort.SliceStable` (not `sort.Slice`) to preserve insertion order on full ties.

Pseudocode:

```go
sort.SliceStable(candidates, func(i, j int) bool {
    a, b := candidates[i], candidates[j]

    // Tier 1: sprint_order (NULLS LAST)
    if a.SprintOrder != nil && b.SprintOrder == nil { return true }
    if a.SprintOrder == nil && b.SprintOrder != nil { return false }
    if a.SprintOrder != nil && b.SprintOrder != nil {
        if *a.SprintOrder != *b.SprintOrder { return *a.SprintOrder < *b.SprintOrder }
    }

    // Tier 2: ExecutionOrder (NULLS LAST) — unchanged from current logic
    if a.ExecutionOrder != nil && b.ExecutionOrder == nil { return true }
    if a.ExecutionOrder == nil && b.ExecutionOrder != nil { return false }
    if a.ExecutionOrder != nil && b.ExecutionOrder != nil {
        if *a.ExecutionOrder != *b.ExecutionOrder { return *a.ExecutionOrder < *b.ExecutionOrder }
    }

    // Tier 3: Priority (lower = higher priority — preserves existing semantics at line 1024–1026)
    if a.Priority != b.Priority { return a.Priority < b.Priority }

    // Tier 4: AssignedAt (oldest first)
    return a.AssignedAt.Before(b.AssignedAt)
})
```

**Computing `selection_reason`**: After sorting, compare `candidates[0]` to `candidates[1]` (if it exists). The first tier on which they differ is the `selection_reason`. If only one candidate exists, default to `"assigned_at"`. Implementation lives in a small private helper `func selectionReason(top, runnerUp *BacklogItemView) string`.

**Why `sort.SliceStable`**: the current code uses `sort.Slice`, which is unstable. With nullable sort keys, instability can cause flaky AI-agent behavior (running `sprint next` twice in a row could return different items if ties exist). Stable sort costs marginally more but is the right behavior for a "the agent expects determinism" interface.

### 3.5 CLI display changes

**Ordered backlog view (human mode)**:

```
Pull Queue: S005 (Sprint 24)  —  3/8 complete (37%)

  #   KEY             TYPE     STATUS          AGENT      TITLE
  --  --------------- -------  --------------  ---------  ------------------------
   1  E19-F07-001     task     in_progress     backend    Implement sprint_order col...
   2  B042            bug      todo            frontend   Fix login redirect loop
  !3  E27-F14-003     task     blocked         backend    Fix Plan tab badges
   4  E19-F07-002     task     todo            backend    Add reorder command
   ~  E19-F07-003     task     todo            backend    Write migration            (unordered)
```

Format rules:
- Header: `Pull Queue: <KEY> (<NAME>)  —  <C>/<T> complete (<P>%)` — uses the existing `SprintBacklog.CompletionPercent` value.
- `#` column: integer for ordered items; `!N` if the item is in the `blocked` workflow phase; `~` for unordered items (left-aligned).
- Items are sorted by `sprint_order ASC NULLS LAST`, then `assigned_at ASC` (matches the comparator above for the visible columns).
- Truncate `TITLE` at 30 characters with `...` suffix.
- Completed items omitted unless `--include-completed`.

**`sprint next` human mode** adds two lines after the existing block:

```
Next Task: E19-F07-001 — Implement sprint_order column
  Type:    task
  Title:   Implement sprint_order column
  Agent:   backend
  Status:  todo
  Priority: 3
  Size:     2
  Sprint position: 2 of 8  |  Selected by: sprint_order
  Sprint:  S005 (Sprint 24)
```

The `position N of K` count is the candidates-after-agent-filter count, not the full sprint backlog. This matches what the agent should reason about.

**`sprint reorder` human output** (top-8 abbreviated queue):

```
Moved B042 to position 1 in S005.

  #   KEY             TITLE
  1   B042            Fix login redirect loop
  2   E19-F07-001     Implement sprint_order column
  3   E19-F07-002     Add reorder command
  ~   E19-F07-003     Write migration
```

**`sprint close --carryover=next` human mode** adds one line:

```
Closed sprint S005 (Sprint 24)
  Completed: 5  Carried over: 3  Dropped: 0
  Incomplete entities moved to: S006
  Order preserved: yes
```

**`sprint start` warning** (after the existing success line):

```
Started sprint S005 (Sprint 24)
Warning: 3 items have no sprint order. Use `shark sprint reorder` to set pull priority.
```

### 3.6 Bulk-add interaction

`BulkAddToSprint` currently calls `BulkAssign` which uses `INSERT OR IGNORE` to skip duplicates. The straightforward extension is:

1. In `SprintService.BulkAddToSprint` (around `sprint_service.go:1192`), before building `toAssign`, fetch `MaxSprintOrder(ctx, sprintEntity.ID)`.
2. Assign `sprint_order = max + i` (1-based offset within the bulk batch) per row in the `toAssign` slice.
3. Pass the per-row `sprint_order` through to `BulkAssign`. Update `BulkAssign`'s INSERT to include the new column.

Rationale: bulk-add happens during planning, when the user has explicitly chosen a feature or entity-type batch. Auto-numbering them densely lets the user immediately see the planned pull order in `sprint backlog --view=ordered`. Skipping bulk-add ordering would create a sprint that is "half-ordered" the moment it is populated, which conflicts with the CX feedback's "ordered view as default" intent (§3.1).

The `BulkAssign` repo method's `INSERT OR IGNORE` semantic must be preserved: rows that violate the partial unique index for `(entity_type, entity_id) WHERE removed_at IS NULL` are silently skipped. The `sprint_order` for skipped rows is wasted (gaps appear), and the service MUST follow the bulk insert with a single dense-renumber UPDATE on the inserted IDs — this is the only path where bulk insert + renumber happens in the same call. Tracked as ImplementationNote-1.

### 3.7 Carryover renumbering

In `CloseSprintWithCarryover`, the existing two branches change as follows:

**`CarryoverNext` branch** (after `ReassignToSprintTx` succeeds):
1. Query `MaxSprintOrder(ctx, nextSprint.ID)` to get the existing high-water mark (call it `M`).
2. Order the carried-over `incompleteAssignments` by `(sprint_order ASC NULLS LAST, assigned_at ASC, id ASC)`.
3. Build a `[]RenumberOp` assigning each carried-over item `sprint_order = M + 1, M + 2, ...`.
4. Call `RenumberAssignmentsTx(ctx, tx, nextSprint.ID, ops)` inside the same transaction.
5. Set `result.CarryoverPreserved = true`.

**`CarryoverBacklog` branch**: `DropAssignmentsTx` already sets `removed_at`. Update the SQL in `DropAssignmentsTx` to also set `sprint_order = NULL` in the same UPDATE statement. This is the cleanest path — the carryover-to-backlog operation atomically clears both fields. Set `result.CarryoverPreserved = false`.

Why renumber from `M + 1` instead of interleaving by priority: the CX feedback §5.5 edge case is binding — interleaving would mutate the receiving sprint's deliberate ordering. PMs can run `sprint reorder` afterwards if they want to slot carried-over items between existing items.

### 3.8 Entity-level decoupling

`shark task update` and other entity-level update paths must NOT touch `sprint_assignments.sprint_order`. The current code does not — there is no cross-write today — so this is a regression-test obligation rather than a code change. Add a regression test in `sprint_service_test.go` (or a new `sprint_decoupling_test.go`) that:

1. Creates a task assigned to a sprint with `sprint_order = 5`.
2. Calls `task update --priority=10 --execution-order=99`.
3. Re-reads the assignment and asserts `sprint_order` is still `5`.

Symmetric test: `sprint reorder` does not change `tasks.priority` or `tasks.execution_order`.

### 3.9 Key technical decisions

| # | Decision | Rationale |
|---|---|---|
| TD-1 | **Dense renumbering** on every explicit reorder; gaps allowed only as a transient state during `sprint add --at` operations within a single transaction. | CX feedback §5.1 is binding. Sprints have 5–15 items; managing sparse positions adds cognitive overhead with no performance gain at this scale. |
| TD-2 | **`sprint_order` is nullable** rather than defaulting to a sentinel value (e.g., `0` or `MAX_INT`). | Lets the system distinguish "unordered" from "intentionally last" and supports the `~` display in CX §3.1. Also matches the non-binding default in the feature.md ("Default it to insertion order so existing workflows keep working"). |
| TD-3 | **`selection_reason` computed at query time**, not stored. | The reason is a function of the candidate set. Storing it would require recomputing on every add/remove/reorder. Computing it once per `sprint next` call is O(1) extra work. |
| TD-4 | **Carryover-to-next appends after existing items** rather than interleaving. | CX feedback §5.5 edge case is binding. Interleaving would mutate the receiving sprint's deliberate ordering. |
| TD-5 | **No auto-ordering** by priority, ExecutionOrder, or dependencies. | CX feedback §5.6 is binding. Auto-ordering would re-introduce exactly the priority-vs-sprint-intent collision this feature exists to solve. |
| TD-6 | **Default view of `sprint backlog`** depends on sprint status (active → ordered, otherwise → grouped). | CX feedback §5.2 is binding. The most useful view for the sprint's lifecycle phase appears without a flag. Documented in `--view` flag help text. |
| TD-7 | **Add a partial unique index** on `(sprint_id, sprint_order) WHERE sprint_order IS NOT NULL AND removed_at IS NULL`. | Enforces the "no two items at the same position in an active sprint" invariant at the DB layer. Catches programming errors that would otherwise corrupt the pull queue. Mirrors `idx_sprint_assignments_active_one`. |
| TD-8 | **Use `sort.SliceStable`** (not `sort.Slice`) in `GetNextTask`. | AI agents call `sprint next` repeatedly and expect deterministic output on ties. Unstable sort is a latent flaky-test source. |
| TD-9 | **Service-owned transaction** for both `sprint add --at` (insert + shift) and `sprint reorder` (compute plan + renumber). | Aligns with `.claude/rules/services/service-design.md` §7. The repo methods accept `*sql.Tx`; the service decides the boundary. |
| TD-10 | **No CHECK constraint on `sprint_order >= 1`** at the DB layer. | Consistent with the post-B018 convention (see `internal/db/db.go:3361`). App-layer validation in `SprintService.AddEntityToSprint` and `ReorderAssignment` rejects `<= 0`. |
| TD-11 | **`BulkAssign` writes `sprint_order` with the bulk operation** (followed by a renumber on skipped rows). | Avoids leaving a sprint half-ordered immediately after planning. See §3.6 ImplementationNote-1. |
| TD-12 | **`SprintBacklog.View` field returned in JSON** so consumers can confirm which view shape they got. | NFR REQ-NF-005 (JSON consistency). Without it, an automated consumer would have to infer the shape from the presence/absence of `Items` vs `Groups`. |

---

## 4. Open Questions and Risks

1. **Concurrent `sprint reorder` calls** — two human users (or two AI agents) calling `reorder` on the same sprint at the same time could race on the renumber UPDATE. SQLite WAL serializes writes, so the second call sees the first's result; the second renumber is applied to the post-first state. This is acceptable behavior — no data corruption, slightly surprising last-writer-wins semantics. Documented in the Long Description of `sprintReorderCmd`.
2. **Migration on very large historical databases** — the backfill `ROW_NUMBER()` window function over `sprint_assignments` is O(n log n) per partition. For databases with thousands of historical assignments this could add measurable startup latency. Acceptable risk: backfill only touches planning/active sprints, which are typically <100 items total. Migration runs once per shark version bump.
3. **Turso compatibility** — `ROW_NUMBER() OVER (PARTITION BY ...)` and partial unique indexes are both supported by Turso's libsql. No special handling needed.

---

## 5. Exit Gate Verification

- [x] Every requirement is testable (REQ-F-001 through REQ-F-009 and the four NFRs each have explicit acceptance criteria).
- [x] Every architecture decision references existing patterns (`migrateSlugColumns` for migration shape, `idx_sprint_assignments_active_one` for partial unique index, `sort.SliceStable` over `sort.Slice`, post-B018 convention for no DB CHECK, `.claude/rules/services/service-design.md` §7 for transaction ownership).
- [x] File paths listed for all changes (see §3.3 component changes table).
- [x] No TBDs in critical sections (Open Questions in §4 are explicitly noted as risks, not blockers).
- [x] Both AI-agent (REQ-F-004 sprint next JSON, REQ-F-006 carryover JSON) and human (REQ-F-005 ordered view, REQ-F-009 start warning) UX requirements are captured.

---

*Last Updated*: 2026-05-16
