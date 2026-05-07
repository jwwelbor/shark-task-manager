---
feature_key: E19-F05-sprint-planning-view-capacity-management
epic_key: E19
title: Sprint Planning View & Capacity Management
spec_version: 1.0
last_updated: 2026-05-06
complexity: STANDARD
---

# Spec — E19-F05: Sprint Planning View & Capacity Management

> **Scope guard.** This feature delivers the *planning surface and capacity model* on top of the sprint foundation established by E19-F01 through E19-F04. It adds five capabilities: `shark sprint plan` (composite planning view), `shark sprint add --bulk` (feature-level bulk assignment), `shark sprint readiness` (scored 0-100), `shark sprint capacity set/show` (per-sprint capacity CRUD), and `sprint_defaults` config support. All computation is done in SprintService with no new tables. Covers REQ-F-011 through REQ-F-015 (Should Have) and the Could Have REQ-F-016 if time permits.

---

## 1. Requirements (Incremental)

This feature satisfies five Should Have requirements from [E19 requirements.md](../requirements.md). Must Have and analytics requirements are covered by E19-F01 through E19-F04. Only incremental scope is documented here.

| Req | Title | What this feature delivers |
|---|---|---|
| **REQ-F-011** | Sprint Planning Command | `shark sprint plan S024`: unassigned backlog sorted by priority+dependency order, capacity utilization per agent type, readiness score |
| **REQ-F-012** | Bulk Entity Assignment | `shark sprint add S024 --bulk E07-F34` and `--bulk-bugs`, `--bulk-tech-debt`, `--bulk-changes` variants |
| **REQ-F-013** | Sprint Readiness Score | `shark sprint readiness S024`: 0-100 score with 6-factor breakdown |
| **REQ-F-014** | Agent Capacity Configuration | `shark sprint capacity set/show`: per-sprint agent capacity CRUD with computed allocation |
| **REQ-F-015** | Default Capacity Configuration | `sprint_defaults.capacity` section in `.sharkconfig.json` with per-sprint inheritance |

REQ-F-016 (auto-creation on close) and REQ-F-017 (dashboard integration) are Could Have; they are called out in §2 as optional extensions after the Should Have scope is complete.

### 1.1 Functional Requirements

#### REQ-F-011: Sprint Planning Command

- **Description**: A single composite view that surfaces the unassigned backlog, capacity utilization, and readiness score for a sprint so PMs can scope a sprint without context-switching.
- **Acceptance Criteria**:
  - [ ] `shark sprint plan S024` displays three sections: (a) unassigned backlog sorted by priority descending then execution_order ascending, limited to tasks in statuses `ready_for_development` or earlier that are not already assigned to any active/planning sprint; (b) current sprint capacity utilization per agent type (capacity_points vs allocated_points = Σ size); (c) sprint readiness score (0-100)
  - [ ] Backlog section shows: entity key, entity type, title, priority, size (or "unsized"), agent_type
  - [ ] Backlog excludes entities already assigned to S024 or any sprint with status `planning` or `active` (uses `sprint_assignments WHERE removed_at IS NULL`)
  - [ ] `--json` output returns structured object with keys `backlog`, `capacity`, `readiness`
  - [ ] Command completes in <500ms for projects with ≤500 backlog entities

#### REQ-F-012: Bulk Entity Assignment

- **Description**: Assign all eligible entities from a feature, or all entities of a given type, to a sprint in one command.
- **Acceptance Criteria**:
  - [ ] `shark sprint add S024 --bulk E07-F34` assigns all tasks from feature E07-F34 that (a) are in statuses eligible for assignment and (b) are not already in any active/planning sprint; skips ineligible tasks silently with a count in output
  - [ ] `shark sprint add S024 --bulk-bugs` assigns all bugs (B###) in open statuses not already in an active/planning sprint
  - [ ] `shark sprint add S024 --bulk-tech-debt` assigns all tech-debt items (TD-###) not already assigned
  - [ ] `shark sprint add S024 --bulk-changes` assigns all change-cards (CC-###) not already assigned
  - [ ] Output shows: count added per entity type, total added, updated capacity utilization summary
  - [ ] Warns (does not fail) if bulk assignment would cause any agent type to exceed its capacity
  - [ ] Bulk assignment is transactional: all-or-nothing per entity type group; partial failure rolls back the group
  - [ ] `--json` output includes `added`, `skipped`, `warnings` fields
  - [ ] Assignable task statuses: any status that is not `completed`, `archived`, or `cancelled` (same logic as individual add in E19-F03)

#### REQ-F-013: Sprint Readiness Score

- **Description**: A 0-100 score composed of six weighted factors that indicate how well-prepared a sprint is for execution.
- **Acceptance Criteria**:
  - [ ] `shark sprint readiness S024` displays the overall score plus per-factor breakdown
  - [ ] Factor 1 — **Capacity utilization** (0-25 pts): 25 pts if utilization 50-100% of capacity; scaled penalty for <50% or >100% (overcommit penalises harder)
  - [ ] Factor 2 — **Dependency satisfaction** (0-20 pts): 20 pts if all dependencies of assigned entities are either also assigned or already completed; 1 pt deducted per unsatisfied external dependency, min 0
  - [ ] Factor 3 — **Task count** (0-15 pts): 0 pts for 0 entities, 15 pts for ≥3 entities, scaled linearly between 1 and 3
  - [ ] Factor 4 — **Agent balance** (0-15 pts): 15 pts if ≥2 distinct agent types present; 0 pts if only one agent type (or all entities have null agent)
  - [ ] Factor 5 — **Sizing coverage** (0-15 pts): 15 pts if all assigned entities have non-null size; 1 pt deducted per unsized entity, min 0
  - [ ] Factor 6 — **Oversized-entity flag** (0-10 pts): 10 pts if no assigned entity has `size >= 8` (L/XL/XXL per `.claude/rules/development-workflows.md` breakdown guidance); 0 pts if any such entity exists
  - [ ] Each factor shows its label, individual score, max score, and a one-line explanation in human-readable output
  - [ ] `--json` output includes: `overall_score`, `factors` array (each with `name`, `score`, `max_score`, `detail`), `unsized_entities` list (key, title), `oversized_entities` list (key, title, size)
  - [ ] Score computed entirely in SprintService; no new DB queries beyond what `plan` already fetches
  - [ ] Sprint with 0 entities returns score 0 with factor-level explanation

#### REQ-F-014: Agent Capacity Configuration

- **Description**: Set and view capacity targets per agent type per sprint; computed allocation shown alongside.
- **Acceptance Criteria**:
  - [ ] `shark sprint capacity set S024 --agent=backend --points=21` creates or updates the `sprint_capacity` row for (S024, backend); validates that `points > 0`
  - [ ] `shark sprint capacity show S024` displays a table: agent_type | capacity_points | allocated_points | remaining | unsized_assigned
  - [ ] `allocated_points` is computed at query time as `Σ size` over assigned entities where `entity_type = 'task'` and `agent_type = <agent>` (bugs/changes/tech-debt do not carry agent_type — they are aggregated into an "unattributed" row if capacity records exist)
  - [ ] `unsized_assigned` is the count of assigned entities with `size IS NULL` for that agent type
  - [ ] `remaining = capacity_points - allocated_points` (can be negative, indicating overcommit)
  - [ ] If no capacity rows exist for a sprint, `show` returns an empty table (not an error)
  - [ ] `--json` output for both `set` and `show`
  - [ ] `shark sprint capacity set` inherits defaults from `sprint_defaults.capacity` in `.sharkconfig.json` when the sprint has no existing capacity rows (first-time setup convenience only — subsequent explicit sets always override)

#### REQ-F-015: Default Capacity Configuration

- **Description**: A `sprint_defaults` section in `.sharkconfig.json` establishes team-level default capacity so it does not need to be configured per sprint.
- **Acceptance Criteria**:
  - [ ] `.sharkconfig.json` accepts a `sprint_defaults` key with the schema described in §3.2
  - [ ] When `shark sprint create` is called and `sprint_defaults.capacity` is non-empty, capacity rows are inserted into `sprint_capacity` at creation time using the default values
  - [ ] `shark sprint capacity set --default --agent=backend --points=21` updates the `sprint_defaults.capacity.backend` value in `.sharkconfig.json`; new sprints created afterward inherit the new value
  - [ ] Per-sprint capacity set via `shark sprint capacity set S024 ...` always overrides defaults and does not modify `.sharkconfig.json`
  - [ ] `sprint_defaults.carryover_behavior` (values: `next`, `backlog`) is read by `shark sprint close` as the default `--carryover` value when the flag is omitted (this integrates with E19-F03 close logic)
  - [ ] `sprint_defaults.auto_create` (bool, default false) is parsed and stored; the auto-creation behaviour it enables is REQ-F-016 (Could Have) and is gated in code by this flag

### 1.2 Non-Functional Requirements

These requirements are additive to those in the epic PRD (see [requirements.md](../requirements.md) §Non-Functional Requirements).

| ID | Requirement | Target |
|---|---|---|
| REQ-NF-001 (from epic) | Sprint command response time | `shark sprint plan` < 500ms; `shark sprint readiness` < 500ms for ≤500 entities |
| REQ-NF-004 (from epic) | Backward compatibility | Zero breaking changes to E19-F01/F02/F03 service interfaces |
| REQ-NF-005 (from epic) | JSON output | All new commands support `--json` and `--field` flags |

**Bulk assignment transaction safety**: The `--bulk` variants must execute all inserts in a single database transaction. If any insert fails (e.g., race condition causing duplicate active assignment), the entire batch is rolled back and a descriptive error is returned.

**Readiness score determinism**: Given identical database state, `shark sprint readiness` must always return the same score. No random or time-dependent factors.

### 1.3 Acceptance Criteria (Feature-Level)

**Scenario 1: Sprint planning view shows composite data**
- Given S024 exists in `planning` status with some capacity set and some tasks assigned
- When `shark sprint plan S024` is run
- Then output contains exactly three sections: backlog (unassigned only), capacity (per agent type), readiness (score + label)
- And backlog excludes tasks already assigned to S024

**Scenario 2: Bulk assignment assigns and warns on capacity**
- Given S024 exists with backend capacity 21 points; feature E07-F34 has 5 ready tasks (4 backend, 1 frontend, total size 25)
- When `shark sprint add S024 --bulk E07-F34` is run
- Then all 5 tasks are assigned, output shows "Added 5 tasks"
- And a warning is emitted: "Backend capacity exceeded: allocated 24/21 points"

**Scenario 3: Readiness score penalises unsized entities**
- Given S024 has 4 assigned tasks, 2 of which have `size IS NULL`
- When `shark sprint readiness S024` is run
- Then Factor 5 (Sizing coverage) shows a score < 15
- And `--json` output lists the 2 unsized entities by key and title

**Scenario 4: Default capacity applied on sprint create**
- Given `.sharkconfig.json` has `sprint_defaults.capacity: {backend: 21, frontend: 13}`
- When `shark sprint create "Sprint 10" --start=2026-06-01 --end=2026-06-14` is run
- Then two rows are inserted into `sprint_capacity` for (S010, backend, 21) and (S010, frontend, 13)

**Scenario 5: Per-sprint capacity set overrides default**
- Given S024 has a capacity row from defaults (backend: 21)
- When `shark sprint capacity set S024 --agent=backend --points=34` is run
- Then the sprint_capacity row for (S024, backend) is updated to 34
- And `.sharkconfig.json` is unchanged

### 1.4 Out of Scope

| Item | Reason | Future |
|---|---|---|
| REQ-F-016 (auto-create on close) | Could Have; config parsed but behavior not invoked unless explicitly wired | E19 follow-on if time permits |
| REQ-F-017 (dashboard integration) | Could Have; `shark status` integration deferred | Same |
| REQ-F-018 (sprint history per entity) | Could Have | Same |
| Real-time capacity reservation | Advisory-only model is sufficient for CLI use case | Possible future feature |
| Bugs/changes/tech-debt agent attribution in capacity | These entity types have no `agent_type` field; tracked as unattributed | Future modelling work |
| Graphical burndown or planning UI | CLI-only per epic scope | Out of epic scope |

---

## 2. Architecture

### 2.1 Existing Foundation (Do Not Modify)

The following are already delivered by E19-F01 through E19-F04 and are consumed as-is:

- **Tables**: `sprints`, `sprint_assignments`, `sprint_capacity` — `internal/db/db.go` (schema version 18)
- **Models**: `models.Sprint`, `models.SprintAssignment`, `models.SprintCapacity` — `internal/models/sprint.go`
- **Repository**: `*sprint.SprintRepository` — `internal/repository/sprint/repository.go` (CRUD, List, GetNextKey)
- **Service**: `*services.SprintService` with `CreateSprint`, `GetSprint`, `ListSprints`, `UpdateSprint`, `DeleteSprint`, `StartSprint`, `CloseSprint`, `ArchiveSprint` — `internal/services/sprint_service.go`
- **CLI commands**: `sprintCmd`, `sprintCreateCmd`, `sprintGetCmd`, `sprintListCmd`, `sprintUpdateCmd`, `sprintDeleteCmd`, `sprintStartCmd`, `sprintCloseCmd`, `sprintArchiveCmd` — `internal/cli/commands/sprint.go`
- **Service accessor**: `cli.GetSprintService()` — `internal/cli/services_global.go`
- **Assignment/backlog commands** from E19-F03: `sprintAddCmd`, `sprintRemoveCmd`, `sprintBacklogCmd`

### 2.2 Component Changes

#### Files Modified

| File | What Changes |
|---|---|
| `internal/services/sprint_service.go` | Add: `PlanSprint`, `BulkAddToSprint`, `GetSprintReadiness`, `SetSprintCapacity`, `GetSprintCapacity` methods; add new repository interfaces `SprintAssignmentQueryRepository` and `SprintCapacityRepository` to the service struct |
| `internal/cli/commands/sprint.go` | Add: `sprintPlanCmd`, `sprintReadinessCmd`, `sprintCapacityCmd` (parent), `sprintCapacitySetCmd`, `sprintCapacityShowCmd`; extend `sprintAddCmd` with `--bulk`, `--bulk-bugs`, `--bulk-tech-debt`, `--bulk-changes` flags |
| `internal/cli/services_global.go` | Update `GetSprintService()` to inject the two new repository dependencies |
| `internal/config/config.go` | Add `SprintDefaults *SprintDefaultsConfig` field to `Config` struct |
| `internal/config/manager.go` | Add `SetSprintCapacityDefault(agentType string, points float64) error` method |
| `internal/repository/sprint/repository.go` | Add methods: `GetCapacity`, `SetCapacity`, `ListCapacity`, `GetAssignmentsWithSize`, `ListUnassignedBacklog`, `BulkAssign` |

#### Files Created

None. All new code extends existing files.

### 2.3 Data Model Changes

No new tables. E19-F01 already delivered `sprint_capacity` and `sprint_assignments`. This feature adds only:

1. **Config struct extension** — `SprintDefaultsConfig` parsed from `sprint_defaults` in `.sharkconfig.json`
2. **Computed capacity fields** — `allocated_points` computed at query time (not stored persistently)

#### 2.3.1 Config Schema Extension

`internal/config/config.go`:

```go
// SprintDefaultsConfig holds team-level defaults for sprint creation.
type SprintDefaultsConfig struct {
    // Capacity is a map of agent_type -> default capacity_points.
    // Applied to new sprints at creation time when sprint_capacity rows are absent.
    Capacity map[string]float64 `json:"capacity,omitempty"`

    // CarryoverBehavior is the default --carryover flag value for shark sprint close.
    // Valid values: "next" (move to next planning sprint) or "backlog" (unassign).
    // Default: "backlog" when absent.
    CarryoverBehavior string `json:"carryover_behavior,omitempty"`

    // AutoCreate, when true, causes shark sprint close to create a new sprint
    // automatically if no planning sprint exists. REQ-F-016 (Could Have).
    AutoCreate bool `json:"auto_create,omitempty"`
}
```

`Config` struct gains:
```go
SprintDefaults *SprintDefaultsConfig `json:"sprint_defaults,omitempty"`
```

#### 2.3.2 New Repository Query Signatures

Added to `internal/repository/sprint/repository.go`:

```go
// GetCapacity returns all capacity rows for a sprint, ordered by agent_type.
func (r *SprintRepository) GetCapacity(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error)

// SetCapacity upserts a capacity row for (sprint_id, agent_type).
func (r *SprintRepository) SetCapacity(ctx context.Context, c *models.SprintCapacity) error

// BulkAssign inserts multiple sprint_assignments in a single transaction.
// Skips entities already actively assigned (removed_at IS NULL) without error.
// Returns count of rows inserted.
func (r *SprintRepository) BulkAssign(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error)

// ListUnassignedBacklog returns entities eligible for assignment, not already in any
// active/planning sprint.  entityType is one of "task", "bug", "change_card", "tech_debt".
// For tasks: filters to statuses in assignableTaskStatuses.
// Returns rows with: entity_type, entity_id, key, title, priority, size, agent_type.
func (r *SprintRepository) ListUnassignedBacklog(ctx context.Context, entityTypes []string) ([]BacklogItem, error)

// GetAssignmentsWithSize returns all active assignments for a sprint with size data
// joined from the appropriate entity table.
func (r *SprintRepository) GetAssignmentsWithSize(ctx context.Context, sprintID int64) ([]AssignmentWithSize, error)
```

Supporting DTOs (in `internal/repository/sprint/repository.go`):

```go
// BacklogItem represents a single item in the unassigned backlog.
type BacklogItem struct {
    EntityType string
    EntityID   int64
    Key        string
    Title      string
    Priority   int
    Size       *int    // nil if unsized
    AgentType  *string // nil for bugs/changes/tech-debt
}

// AssignmentWithSize joins sprint_assignments with entity size data.
type AssignmentWithSize struct {
    EntityType string
    EntityID   int64
    Key        string
    AgentType  *string
    Size       *int
}
```

### 2.4 Service Layer Design

#### New Methods on SprintService

All methods follow the pattern established in `internal/services/sprint_service.go`. Each:
- Takes `context.Context` as first parameter
- Returns domain models (not DTOs)
- Wraps errors with business context using `fmt.Errorf("...: %w", err)`
- Does not format output

```go
// PlanSprintInput are the parameters for the planning view.
type PlanSprintInput struct {
    SprintKey string
}

// SprintPlanView is the output of PlanSprint.
type SprintPlanView struct {
    Sprint    *models.Sprint
    Backlog   []sprint.BacklogItem
    Capacity  []CapacityRow       // per agent type: capacity, allocated, remaining, unsized
    Readiness *SprintReadiness
}

// PlanSprint returns the composite planning view for a sprint.
// Fetches backlog, capacity, and readiness in parallel-safe sequential queries.
func (s *SprintService) PlanSprint(ctx context.Context, key string) (*SprintPlanView, error)
```

```go
// BulkAddInput parameters for bulk assignment.
type BulkAddInput struct {
    SprintKey   string
    FeatureKey  string // non-empty means bulk by feature
    EntityType  string // "bug", "change_card", "tech_debt" for --bulk-X variants; empty for feature
}

// BulkAddResult is the output of BulkAddToSprint.
type BulkAddResult struct {
    Added    int
    Skipped  int
    Warnings []string // e.g. "Backend capacity exceeded: 24/21 points"
    // per-entity-type breakdown if multiple types were processed
    ByType   map[string]int
}

// BulkAddToSprint assigns all eligible entities of the given type/feature to a sprint.
// Uses a database transaction for atomicity. Returns result with added/skipped counts.
func (s *SprintService) BulkAddToSprint(ctx context.Context, input BulkAddInput) (*BulkAddResult, error)
```

```go
// SprintReadiness is the output of GetSprintReadiness.
type SprintReadiness struct {
    OverallScore    int
    Factors         []ReadinessFactor
    UnsizedEntities []sprint.BacklogItem  // entities with size IS NULL
    OversizedEntities []sprint.BacklogItem // entities with size >= 8
}

// ReadinessFactor is a single readiness scoring factor.
type ReadinessFactor struct {
    Name        string // e.g. "Capacity utilization"
    Score       int
    MaxScore    int
    Detail      string // one-line explanation
}

// GetSprintReadiness computes the 0-100 readiness score for a sprint.
// Computation is entirely in-memory after fetching assignments and capacity;
// no additional DB queries per factor.
func (s *SprintService) GetSprintReadiness(ctx context.Context, key string) (*SprintReadiness, error)
```

```go
// CapacityRow is one row in the capacity display.
type CapacityRow struct {
    AgentType       string
    CapacityPoints  float64
    AllocatedPoints float64 // Σ size of assigned entities for this agent type
    Remaining       float64
    UnsizedAssigned int
}

// SetSprintCapacityInput parameters for setting capacity.
type SetSprintCapacityInput struct {
    SprintKey string
    AgentType string
    Points    float64
}

// SetSprintCapacity creates or updates a capacity row for a (sprint, agent_type) pair.
func (s *SprintService) SetSprintCapacity(ctx context.Context, input SetSprintCapacityInput) (*models.SprintCapacity, error)

// GetSprintCapacity returns capacity vs. allocation for all agent types in a sprint.
// Returns empty slice (not error) if no capacity rows exist.
func (s *SprintService) GetSprintCapacity(ctx context.Context, key string) ([]CapacityRow, error)
```

#### Readiness Score Algorithm

Implemented in `GetSprintReadiness`. All state fetched once via `GetAssignmentsWithSize` and `GetCapacity`:

```
assignments  = GetAssignmentsWithSize(ctx, sprint.ID)
capacityRows = GetCapacity(ctx, sprint.ID)

totalEntities = len(assignments)
totalCapacity = Σ capacity_rows.capacity_points
totalAllocated = Σ(size) for sized assignments

Factor 1 — Capacity utilization (0-25):
  if totalCapacity == 0: score = 0
  utilization = totalAllocated / totalCapacity
  if 0.5 <= utilization <= 1.0: score = 25
  elif utilization > 1.0: score = max(0, 25 - int((utilization-1.0)*50))  // -2 pts per 4% over
  else: score = int(utilization / 0.5 * 25)

Factor 2 — Dependency satisfaction (0-20):
  unsatisfied = count of external task dependencies not assigned to sprint AND not completed
  score = max(0, 20 - unsatisfied)

Factor 3 — Task count (0-15):
  if totalEntities == 0: score = 0
  elif totalEntities >= 3: score = 15
  else: score = int(totalEntities / 3.0 * 15)

Factor 4 — Agent balance (0-15):
  distinctAgents = count of distinct non-nil agent_type values in assignments
  score = 15 if distinctAgents >= 2 else 0

Factor 5 — Sizing coverage (0-15):
  unsized = count of assignments with size IS NULL
  score = max(0, 15 - unsized)

Factor 6 — Oversized-entity flag (0-10):
  oversized = count of assignments with size >= 8
  score = 10 if oversized == 0 else 0

overallScore = Σ factor scores (max 100)
```

This algorithm is stateless and deterministic — same inputs always produce same output.

#### Service Struct Extension

The existing `SprintService` struct gains two new repository dependencies:

```go
type SprintService struct {
    repo            SprintRepository               // existing
    workflowSvc     *workflow.Service              // existing
    assignmentRepo  SprintAssignmentQueryRepository // NEW
    capacityRepo    SprintCapacityRepository        // NEW
    cfg             *config.Config                  // NEW — for sprint_defaults
}
```

Two new repository interfaces (defined in `sprint_service.go` near `SprintRepository`):

```go
// SprintAssignmentQueryRepository handles assignment queries needed for planning.
type SprintAssignmentQueryRepository interface {
    BulkAssign(ctx context.Context, sprintID int64, assignments []models.SprintAssignment) (int, error)
    ListUnassignedBacklog(ctx context.Context, entityTypes []string) ([]sprint.BacklogItem, error)
    GetAssignmentsWithSize(ctx context.Context, sprintID int64) ([]sprint.AssignmentWithSize, error)
}

// SprintCapacityRepository handles capacity CRUD.
type SprintCapacityRepository interface {
    GetCapacity(ctx context.Context, sprintID int64) ([]*models.SprintCapacity, error)
    SetCapacity(ctx context.Context, c *models.SprintCapacity) error
}
```

Both interfaces are implemented by the existing `*sprint.SprintRepository` (extended in §2.3.2). This avoids introducing a second repository type and keeps wiring simple.

Constructor update:

```go
func NewSprintService(
    repo SprintRepository,
    workflowSvc *workflow.Service,
    assignmentRepo SprintAssignmentQueryRepository,
    capacityRepo SprintCapacityRepository,
    cfg *config.Config,
) *SprintService {
    if repo == nil { panic("SprintRepository cannot be nil") }
    if workflowSvc == nil { panic("workflow.Service cannot be nil") }
    // assignmentRepo, capacityRepo, cfg are nil-safe (graceful degradation)
    return &SprintService{
        repo:           repo,
        workflowSvc:    workflowSvc.ForLevel("sprint"),
        assignmentRepo: assignmentRepo,
        capacityRepo:   capacityRepo,
        cfg:            cfg,
    }
}
```

Existing callers of `NewSprintService` in `internal/cli/services_global.go` must be updated to pass the new parameters.

### 2.5 CLI Commands

All new commands are added to `internal/cli/commands/sprint.go` following the thin-wrapper pattern. Each does: parse → call service → format.

#### New Commands

**`sprintPlanCmd`** (`shark sprint plan <KEY>`):
```
cobra.Command{Use: "plan <sprint-key>", Short: "Show planning view", Args: ExactArgs(1)}
runSprintPlan: parse key → GetSprintService().PlanSprint(ctx, key) → format output
Output: three sections (Backlog / Capacity / Readiness); --json returns SprintPlanView as JSON
```

**`sprintReadinessCmd`** (`shark sprint readiness <KEY>`):
```
cobra.Command{Use: "readiness <sprint-key>", Short: "Show sprint readiness score", Args: ExactArgs(1)}
runSprintReadiness: parse key → GetSprintService().GetSprintReadiness(ctx, key) → format table
Output: score summary + factor table; --json returns SprintReadiness as JSON
```

**`sprintCapacityCmd`** (`shark sprint capacity`): parent command with two subcommands:

**`sprintCapacitySetCmd`** (`shark sprint capacity set <KEY>`):
```
Flags: --agent=STRING (required), --points=FLOAT (required), --default (bool)
runSprintCapacitySet:
  if --default: call config.SetSprintCapacityDefault(agent, points)
  else: call GetSprintService().SetSprintCapacity(ctx, input) → print updated row
```

**`sprintCapacityShowCmd`** (`shark sprint capacity show <KEY>`):
```
cobra.Command{Use: "show <sprint-key>", Short: "Show capacity vs. allocation", Args: ExactArgs(1)}
runSprintCapacityShow: parse key → GetSprintService().GetSprintCapacity(ctx, key) → table
Output: table [AgentType | Capacity | Allocated | Remaining | Unsized]; --json returns []CapacityRow
```

#### Extended Flags on sprintAddCmd

The existing `sprintAddCmd` (from E19-F03) gains four boolean flags:

```go
sprintAddCmd.Flags().String("bulk", "", "Feature key: assign all eligible tasks from this feature")
sprintAddCmd.Flags().Bool("bulk-bugs", false, "Assign all open bugs not already in a sprint")
sprintAddCmd.Flags().Bool("bulk-tech-debt", false, "Assign all open tech-debt items not already in a sprint")
sprintAddCmd.Flags().Bool("bulk-changes", false, "Assign all open change-cards not already in a sprint")
```

`runSprintAdd` updated to detect which flag is set and route to `GetSprintService().BulkAddToSprint(ctx, input)` when any bulk flag is present. Individual add path unchanged.

#### Command Registration

New commands registered in `init()` at the bottom of `sprint.go`:
```go
func init() {
    // ...existing registrations...
    sprintCmd.AddCommand(sprintPlanCmd)
    sprintCmd.AddCommand(sprintReadinessCmd)
    sprintCapacityCmd.AddCommand(sprintCapacitySetCmd)
    sprintCapacityCmd.AddCommand(sprintCapacityShowCmd)
    sprintCmd.AddCommand(sprintCapacityCmd)
}
```

### 2.6 Repository Queries

The `ListUnassignedBacklog` query (simplified):

```sql
-- Tasks eligible for assignment
SELECT 'task' AS entity_type, t.id, t.key, t.title, t.priority, t.size, t.agent_type
FROM tasks t
WHERE t.status NOT IN ('completed', 'archived', 'cancelled')
  AND NOT EXISTS (
    SELECT 1 FROM sprint_assignments sa
    JOIN sprints s ON s.id = sa.sprint_id
    WHERE sa.entity_type = 'task'
      AND sa.entity_id = t.id
      AND sa.removed_at IS NULL
      AND s.status IN ('planning', 'active')
  )
-- UNION ALL similar selects for bugs, change_cards, tech_debts when requested
ORDER BY priority DESC, t.execution_order ASC NULLS LAST
```

The `GetAssignmentsWithSize` query uses a `UNION ALL` to join each entity type's table:

```sql
SELECT sa.entity_type, sa.entity_id, t.key, t.size, t.agent_type
FROM sprint_assignments sa
JOIN tasks t ON sa.entity_id = t.id
WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'task'
UNION ALL
SELECT sa.entity_type, sa.entity_id, b.key, b.size, NULL as agent_type
FROM sprint_assignments sa
JOIN bugs b ON sa.entity_id = b.id
WHERE sa.sprint_id = ? AND sa.removed_at IS NULL AND sa.entity_type = 'bug'
-- ... similar for change_cards, tech_debts
```

`SetCapacity` uses `INSERT OR REPLACE` (SQLite upsert via UNIQUE constraint on `(sprint_id, agent_type)`):

```sql
INSERT INTO sprint_capacity (sprint_id, agent_type, capacity_points)
VALUES (?, ?, ?)
ON CONFLICT(sprint_id, agent_type)
DO UPDATE SET capacity_points = excluded.capacity_points, updated_at = CURRENT_TIMESTAMP
```

### 2.7 Config Integration

`internal/config/manager.go` gains:

```go
// SetSprintCapacityDefault updates sprint_defaults.capacity.<agentType> in .sharkconfig.json.
// Creates the sprint_defaults section if absent.
func (m *Manager) SetSprintCapacityDefault(agentType string, points float64) error
```

This follows the pattern of existing config-mutation methods in `manager.go` (e.g., `SetCloudConfig`): read current config, mutate, write back as JSON.

### 2.8 Default Capacity on Sprint Create

`CreateSprint` in `sprint_service.go` is extended:

```go
func (s *SprintService) CreateSprint(ctx context.Context, input CreateSprintInput) (*models.Sprint, error) {
    // ... existing validation and creation ...

    // Apply sprint_defaults.capacity if configured and capacityRepo is available
    if s.cfg != nil && s.cfg.SprintDefaults != nil && len(s.cfg.SprintDefaults.Capacity) > 0 && s.capacityRepo != nil {
        for agentType, points := range s.cfg.SprintDefaults.Capacity {
            _ = s.capacityRepo.SetCapacity(ctx, &models.SprintCapacity{
                SprintID:       newSprint.ID,
                AgentType:      agentType,
                CapacityPoints: points,
            })
            // Non-fatal: log but don't fail sprint creation if capacity insert fails
        }
    }

    return newSprint, nil
}
```

### 2.9 Service Accessor Update

`internal/cli/services_global.go` — `GetSprintService()` updated:

```go
func GetSprintService() *services.SprintService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    cfg, _ := GetConfig()  // nil-safe; sprint_defaults read here
    sprintRepo := sprint.NewSprintRepository(db)
    workflowSvc := GetWorkflowService()
    return services.NewSprintService(sprintRepo, workflowSvc, sprintRepo, sprintRepo, cfg)
}
```

`sprintRepo` satisfies all three repository interfaces (`SprintRepository`, `SprintAssignmentQueryRepository`, `SprintCapacityRepository`) because the concrete `*sprint.SprintRepository` implements all methods. This avoids extra allocations and keeps wiring minimal.

### 2.10 Key Technical Decisions

| Decision | Rationale |
|---|---|
| Single `*sprint.SprintRepository` implements all three interfaces | Avoids a separate `CapacityRepository` type; `UNION ALL` queries for multi-entity backlog are contained in one struct |
| Readiness score computed in-memory after two queries | Keeps score computation testable without DB; avoids N+1 queries per factor |
| Bulk assignment is transactional at the group level | Partial inserts would corrupt capacity calculations; atomicity matches user expectation |
| `INSERT OR REPLACE` for capacity | SQLite-native upsert on `UNIQUE(sprint_id, agent_type)` — simpler than separate insert/update branches |
| Config mutation via `manager.SetSprintCapacityDefault` | Consistent with existing cloud config mutation pattern; avoids a new persistence layer |
| `--bulk` takes a feature key as string value (not bool) | Allows `--bulk E07-F34` to be unambiguous and consistent with how entity keys are passed elsewhere |
| Dependency satisfaction (Factor 2) skips non-task entities | Bugs/changes/tech-debt do not have `depends_on` in the schema; factor only evaluates task dependencies |

---

## 3. Test Plan

Follows patterns in `.claude/rules/testing/architecture.md`.

### 3.1 Service Tests (`internal/services/sprint_service_test.go` extended)

Tests use mock repositories (function-field mocks, per existing service test patterns). No real database.

| Test | Asserts |
|---|---|
| `TestSprintService_PlanSprint_HappyPath` | Returns backlog, capacity, readiness for a sprint with data |
| `TestSprintService_PlanSprint_EmptySprint` | Returns empty backlog and zero readiness score |
| `TestSprintService_BulkAddToSprint_ByFeature` | Assigns expected entities; returns correct added/skipped counts |
| `TestSprintService_BulkAddToSprint_ExceedsCapacity_Warning` | Returns BulkAddResult.Warnings non-empty when over capacity |
| `TestSprintService_BulkAddToSprint_TransactionRollback` | On repo error, no entities are added |
| `TestSprintService_GetSprintReadiness_AllFactors` | Each factor computed correctly for known input |
| `TestSprintService_GetSprintReadiness_ZeroEntities` | Returns score 0 |
| `TestSprintService_GetSprintReadiness_OversizedPenalty` | Factor 6 = 0 when any size >= 8 |
| `TestSprintService_GetSprintReadiness_CapacityOvercommit` | Factor 1 < 25 when utilization > 100% |
| `TestSprintService_SetSprintCapacity_CreateNew` | Calls SetCapacity with correct arguments |
| `TestSprintService_SetSprintCapacity_ZeroPoints` | Returns validation error |
| `TestSprintService_GetSprintCapacity_ComputesAllocated` | Computes allocated_points from AssignmentWithSize data |
| `TestSprintService_CreateSprint_AppliesDefaults` | When cfg.SprintDefaults.Capacity is set, capacity rows are created |

### 3.2 Repository Tests (`internal/repository/sprint/repository_test.go` extended)

Use real test database (`test.GetTestDB()`). Clean up with `DELETE FROM sprint_capacity WHERE sprint_id = ?` etc.

| Test | Asserts |
|---|---|
| `TestSprintRepository_SetCapacity_Insert` | New row inserted; GetCapacity returns it |
| `TestSprintRepository_SetCapacity_Upsert` | Existing row updated; count stays 1 |
| `TestSprintRepository_GetCapacity_Empty` | Returns empty slice (not error) |
| `TestSprintRepository_ListUnassignedBacklog_ExcludesAssigned` | Tasks already in active sprint not returned |
| `TestSprintRepository_ListUnassignedBacklog_ExcludesCompleted` | Tasks in completed status not returned |
| `TestSprintRepository_BulkAssign_InsertsAll` | All assignments inserted; GetAssignmentsWithSize returns them |
| `TestSprintRepository_BulkAssign_SkipsDuplicate` | Re-assigning an already-assigned entity returns count 0, no error |
| `TestSprintRepository_GetAssignmentsWithSize_MultiType` | Returns tasks and bugs with correct size values |

### 3.3 CLI Tests (`internal/cli/commands/sprint_test.go` extended)

Use mock service (function-field mock implementing the `sprintServicer` interface extended with new methods). No real database.

| Test | Asserts |
|---|---|
| `TestSprintPlanCommand_JSON` | `--json` output contains backlog, capacity, readiness keys |
| `TestSprintReadinessCommand_Output` | Prints score and factor table |
| `TestSprintCapacitySet_RequiresAgentAndPoints` | Missing --agent or --points returns error |
| `TestSprintCapacitySet_Default` | `--default` calls config mutation, not service |
| `TestSprintCapacityShow_EmptySprint` | Prints empty table, no error |
| `TestSprintAdd_BulkByFeature` | Calls BulkAddToSprint with FeatureKey populated |
| `TestSprintAdd_BulkBugs` | Calls BulkAddToSprint with EntityType="bug" |

---

## 4. File Paths Summary

### Modified Files

| File | Change |
|---|---|
| `internal/services/sprint_service.go` | New methods: PlanSprint, BulkAddToSprint, GetSprintReadiness, SetSprintCapacity, GetSprintCapacity; new interfaces; updated constructor; CreateSprint default-capacity logic |
| `internal/cli/commands/sprint.go` | New commands: plan, readiness, capacity set, capacity show; extended add flags |
| `internal/cli/services_global.go` | Updated GetSprintService() constructor call |
| `internal/config/config.go` | Added SprintDefaultsConfig type and SprintDefaults field on Config |
| `internal/config/manager.go` | Added SetSprintCapacityDefault method |
| `internal/repository/sprint/repository.go` | Added GetCapacity, SetCapacity, BulkAssign, ListUnassignedBacklog, GetAssignmentsWithSize methods; BacklogItem and AssignmentWithSize DTOs |

### Test Files Modified

| File | Change |
|---|---|
| `internal/services/sprint_service_test.go` | New test cases per §3.1 |
| `internal/repository/sprint/repository_test.go` | New test cases per §3.2 |
| `internal/cli/commands/sprint_test.go` | New test cases per §3.3 |

### No New Files Required

All new code fits within existing files.

---

## 5. Acceptance Gate

- [ ] `shark sprint plan S024` outputs all three sections (backlog, capacity, readiness) and `--json` is well-formed
- [ ] `shark sprint add S024 --bulk E07-F34` assigns expected tasks transactionally; over-capacity emits warning not error
- [ ] `shark sprint readiness S024` shows score 0-100 with all 6 factors and their individual scores
- [ ] `shark sprint capacity set S024 --agent=backend --points=21` upserts capacity row
- [ ] `shark sprint capacity show S024` shows computed allocation alongside capacity
- [ ] `shark sprint capacity set --default --agent=backend --points=21` updates `.sharkconfig.json`
- [ ] New sprint inherits capacity from `sprint_defaults.capacity` when config is set
- [ ] `make fmt && make lint && make test` all pass
- [ ] No existing sprint, task, feature, or epic tests regress

---

*End of spec.*
