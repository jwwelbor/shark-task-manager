---
feature_key: E19-F04-sprint-analytics-velocity-burndown-summary
epic_key: E19
title: Sprint Analytics — Velocity, Burndown & Summary
spec_version: 1.0
last_updated: 2026-05-05
complexity: STANDARD
---

# Spec — E19-F04: Sprint Analytics — Velocity, Burndown & Summary

> **Scope guard.** This feature delivers three read-only analytics commands: `shark sprint velocity`, `shark sprint burndown`, and `shark sprint summary`. It introduces one new analytics service (`SprintAnalyticsService`), one new repository query file (`internal/repository/sprint/analytics.go`), and one new CLI command file extension. **Out of scope here:** sprint lifecycle transitions, entity assignment, capacity management, planning view (E19-F01 through F03, F05).

---

## 1. Requirements (Mapped)

This feature satisfies requirements REQ-F-007, REQ-F-008, and REQ-F-009 from [E19 requirements.md](../requirements.md). REQ-NF-001 (analytics < 2 s for ≤ 50 sprints / 1000 tasks) also applies.

| Req | Title | What this feature delivers |
|---|---|---|
| **REQ-F-007** | Velocity Calculation | `shark sprint velocity [--sprints=N] [--json]`: Σ size of completed entities per sprint, trailing average, unsized_completed count |
| **REQ-F-008** | Sprint Burndown | `shark sprint burndown [KEY] [--json]`: ideal vs. actual remaining Σ size by day, reconstructed from `task_history`, text-art chart |
| **REQ-F-009** | Sprint Summary Report | `shark sprint summary KEY [--detailed] [--json]`: planned/completed Σ size, task counts, velocity comparison, optionally cycle-time-by-phase from `work_sessions` |

All analytics depend on:
- **E19-F01** (schema: `sprints`, `sprint_assignments` tables exist)
- **E19-F02** (SprintRepository, SprintService, sprint CLI command root)
- **E19-F03** (entity assignment populates `sprint_assignments`)
- **E07-F42** (`size` column on `tasks`, `bugs`, `change_cards`, `tech_debts`) — consumed at query time, not cached

---

## 2. Functional Requirements

### REQ-F-007: Velocity Calculation

**AC-V-1**: `shark sprint velocity` shows completed Σ size per sprint for the last 5 completed sprints (default), ordered oldest→newest.

**AC-V-2**: `--sprints=N` overrides the number of historical sprints. `N` must be ≥ 1 and ≤ 100; values outside this range return a validation error.

**AC-V-3**: Entities with `size IS NULL` contribute 0 to Σ size and are reported separately as `unsized_completed: N` per sprint. Both the human-readable and JSON outputs include this count, making the omission visible.

**AC-V-4**: The trailing average is the mean of non-null Σ size values across the returned sprints (i.e., sprints with zero sized completions contribute 0, but the count is divided by the number of sprints, not just sprints with non-zero velocity).

**AC-V-5**: When fewer than 3 completed sprints exist, the command displays an "insufficient data" message and exits 0 (not an error; informational).

**AC-V-6**: `--json` output includes an array of per-sprint objects:
```json
{
  "sprints": [
    { "key": "S001", "name": "Sprint 1", "completed_size": 18, "unsized_completed": 2 },
    { "key": "S002", "name": "Sprint 2", "completed_size": 21, "unsized_completed": 0 }
  ],
  "trailing_average": 19.5,
  "sprint_count": 2
}
```

### REQ-F-008: Sprint Burndown

**AC-B-1**: `shark sprint burndown` (no key) shows the burndown for the current active sprint. `shark sprint burndown S024` shows the burndown for the specified sprint.

**AC-B-2**: The command accepts sprints in any of the following statuses: `active`, `closing`, `completed`, `archived`. A sprint in `planning` status has no burndown data and returns an informational message.

**AC-B-3**: The ideal burndown is a linear interpolation from the sprint's initial Σ size (calculated at `start_date`) to 0 at `end_date`. If entities were added or removed mid-sprint, the ideal line recalculates from the new Σ size at the point of change (piecewise linear reset).

**AC-B-4**: The actual remaining Σ size per calendar day is reconstructed from `sprint_assignments` (which entities were assigned) cross-referenced with `task_history` (when each entity's status transitioned to a terminal state within the sprint window). An entity is "remaining" on day D if it has not reached a terminal completion status as of end-of-day D. Unsized entities contribute 0 to the remaining Σ size.

**AC-B-5**: `unsized_remaining: N` is included in each daily data point in `--json` output and in the chart legend for human-readable output.

**AC-B-6**: Human-readable output uses a text table (not Unicode block characters for maximum terminal compatibility):
```
Sprint S024 Burndown — Sprint 24 (2026-03-18 to 2026-04-01)
Day        Ideal    Actual    Unsized Remaining
─────────────────────────────────────────────────
2026-03-18  42       42        3
2026-03-19  39       40        3
2026-03-20  36       38        2
...
```

**AC-B-7**: `--json` output includes a `data_points` array:
```json
{
  "sprint_key": "S024",
  "sprint_name": "Sprint 24",
  "total_size": 42,
  "unsized_total": 3,
  "data_points": [
    { "date": "2026-03-18", "ideal_remaining": 42.0, "actual_remaining": 42.0, "unsized_remaining": 3 },
    { "date": "2026-03-19", "ideal_remaining": 39.0, "actual_remaining": 40.0, "unsized_remaining": 3 }
  ]
}
```

**AC-B-8**: For days in the future (sprint in `active` or `closing` status), ideal remaining is projected; actual remaining column shows `—` in human-readable output and is omitted from JSON data points.

### REQ-F-009: Sprint Summary Report

**AC-S-1**: `shark sprint summary S024` is available for sprints in `completed` or `archived` status. Calling it on a `planning` or `active` sprint returns an informational message (not an error).

**AC-S-2**: Base summary output includes:
- Planned Σ size (sum of sizes assigned at sprint start or first assigned_at date)
- Completed Σ size (sum of sizes of entities that reached terminal status within the sprint)
- Completion percentage by size: `completed_size / planned_size * 100`
- Planned entity count (total entities ever assigned, excluding mid-sprint removals that were then readded)
- Completed entity count (entities that reached terminal status within the sprint window)
- Velocity this sprint (= completed Σ size)
- Trailing average velocity (same calculation as velocity command, using the N-1 sprints before this one)
- Velocity delta vs. trailing average (absolute and percentage)
- Unsized planned count (entities assigned but with `size IS NULL`)
- Unsized completed count (completed entities with `size IS NULL`)

**AC-S-3**: `--detailed` adds (with graceful degradation when E13 `work_sessions` data is absent):
- Entities added mid-sprint: count and Σ size
- Entities removed mid-sprint: count and Σ size
- Average cycle time by workflow phase (computed from `task_history` timestamps between status transitions). If `work_sessions` table is empty for this sprint's tasks, this section shows "No session data available" rather than failing.
- Average size of completed entities
- Size-band distribution: count of XS(1)/S(2)/M(3)/L(5)/XL(8)/XXL(13) completed
- Carryover entity list: entities not completed, with their keys, types, and sizes

**AC-S-4**: `--json` output includes all base summary fields. With `--detailed`, adds the detailed fields. Missing `work_sessions` data is represented as `null` in JSON (not omitted), allowing callers to distinguish "no data" from "field not computed".

### Non-Functional Requirements

**REQ-NF-001 (performance)**: Analytics commands complete in < 2 seconds for up to 50 sprints and 1000 tasks. Query plans for the core analytics queries must use indexed lookups (verified with SQLite `EXPLAIN QUERY PLAN`). The existing indexes on `sprint_assignments(sprint_id)`, `sprint_assignments(entity_type, entity_id)`, and `task_history` satisfy this.

**REQ-NF-005 (JSON consistency)**: All three commands support `--json` and `--field` flags consistent with existing entity commands (see `internal/cli/services_global.go` accessor pattern).

---

## 3. Out of Scope

- Writing or updating sprint data (read-only feature)
- Predictive sprint scope suggestions based on velocity (deferred)
- Chart rendering beyond text table (no terminal graphics libraries)
- `--detailed` cycle-time when `work_sessions` is unavailable causes a visible note, not a hard error
- Analytics for sprints in `planning` status (no meaningful data exists yet)
- Capacity analysis (REQ-F-014) — belongs to E19-F05

---

## 4. Architecture

### 4.1 Component Overview

This feature adds:

1. **`internal/repository/sprint/analytics.go`** (new file, ~200 lines)
   - Read-only aggregate queries on `sprint_assignments` joined to entity tables
   - No business logic; returns raw query results as typed structs

2. **`internal/services/sprint_analytics_service.go`** (new file, ~300 lines)
   - `SprintAnalyticsService`: orchestrates analytics calculations
   - Depends on `SprintAnalyticsRepository` interface (defined here) and the existing `SprintRepository` interface from `internal/services/sprint_service.go`
   - Optional `WorkSessionRepository` interface for cycle-time in `--detailed` (degrades gracefully when nil)

3. **`internal/services/sprint_analytics_dto.go`** (new file, ~80 lines)
   - Output DTOs: `VelocityResult`, `BurndownResult`, `BurndownDataPoint`, `SprintSummaryResult`, `SprintSummaryDetailed`

4. **CLI additions to `internal/cli/commands/sprint.go`** (existing file, +~150 lines)
   - Three new command handlers: `runSprintVelocity`, `runSprintBurndown`, `runSprintSummary`
   - Commands registered as subcommands of existing `sprintCmd`

5. **`internal/cli/services_global.go`** (existing file, +~15 lines)
   - `GetSprintAnalyticsService()` global accessor

### 4.2 Data Model — No Schema Changes

E19-F01 delivered all required tables. This feature performs read-only queries. No `CurrentSchemaVersion` bump is needed.

The queries join `sprints` ↔ `sprint_assignments` ↔ entity tables (`tasks`, `bugs`, `change_cards`, `tech_debts`) and `task_history`. The `size` column on each entity table comes from E07-F42 and is treated as nullable (`*int`).

### 4.3 Repository Layer — `internal/repository/sprint/analytics.go`

Package: `sprint` (same package as `repository.go`).

**New struct and constructor:**
```go
type SprintAnalyticsRepository struct {
    db *dbconn.DB
}

func NewSprintAnalyticsRepository(db *dbconn.DB) *SprintAnalyticsRepository {
    return &SprintAnalyticsRepository{db: db}
}
```

**Key query methods:**

```go
// VelocityRow is one sprint's velocity data, returned by GetVelocityData.
type VelocityRow struct {
    SprintKey        string
    SprintName       string
    CompletedSize    int
    UnsizedCompleted int
}

// GetVelocityData returns the last N completed sprints with their completed Σ size.
// Entities with size IS NULL contribute 0 to CompletedSize and are counted in UnsizedCompleted.
// Results are ordered oldest to newest.
func (r *SprintAnalyticsRepository) GetVelocityData(ctx context.Context, limit int) ([]VelocityRow, error)

// AssignedEntity represents one row from the polymorphic sprint_assignments join.
type AssignedEntity struct {
    EntityType string
    EntityID   int64
    AssignedAt time.Time
    RemovedAt  *time.Time
    Size       *int    // from entity table; nil when size IS NULL
}

// GetSprintAssignedEntities returns all entities ever assigned to the sprint (including removed ones).
// Used for burndown reconstruction and summary.
func (r *SprintAnalyticsRepository) GetSprintAssignedEntities(ctx context.Context, sprintID int64) ([]AssignedEntity, error)

// TaskCompletionEvent is a status transition that constitutes "entity completed" for burndown.
type TaskCompletionEvent struct {
    EntityID   int64
    EntityType string
    NewStatus  string
    Timestamp  time.Time
}

// GetCompletionEvents returns status transitions to terminal states for all entities
// assigned to the sprint, filtered to within the sprint's date window.
// Terminal states are determined by the caller (service layer owns workflow knowledge).
// The query pulls from task_history; bugs/change_cards/tech_debts with their own
// status-history tables are treated similarly via UNION or separate queries.
func (r *SprintAnalyticsRepository) GetCompletionEvents(ctx context.Context, sprintID int64, startDate, endDate time.Time) ([]TaskCompletionEvent, error)

// PhaseTimeRow is one phase's average duration for work_sessions in a sprint.
type PhaseTimeRow struct {
    Phase        string        // old_status of the task_history transition
    AverageDays  float64
}

// GetCycleTimeByPhase returns average time spent in each workflow phase for tasks
// assigned to the sprint. Returns nil slice (not error) when work_sessions is empty.
func (r *SprintAnalyticsRepository) GetCycleTimeByPhase(ctx context.Context, sprintID int64) ([]PhaseTimeRow, error)
```

**Query design notes:**

- `GetVelocityData` uses a single SQL query with a `LEFT JOIN` to each entity table (task, bug, change_card, tech_debt) via `CASE entity_type` or four `UNION ALL` subqueries, selecting `COALESCE(t.size, b.size, cc.size, td.size)`. The cleanest approach for SQLite is four `UNION ALL` subqueries per entity type inside a CTE, then aggregated by sprint.

- `GetCompletionEvents` joins `sprint_assignments sa` with `task_history th ON th.task_id = sa.entity_id AND sa.entity_type = 'task'` for task completions. For bugs/change_cards/tech_debts, those entity types do not have a `task_history` equivalent — their status changes are tracked in the main entity table's `updated_at` and status. A pragmatic simplification for v1: completion events for non-task entity types use `sprint_assignments.removed_at IS NULL AND entity.status IN (terminal_statuses)` as a point-in-time check, not a time-series reconstruction. This is noted in the output legend as "task burndown reconstructed from history; other entity types use current status."

- `GetCycleTimeByPhase` joins `task_history` with `work_sessions` on `task_id`, grouping by `old_status` to compute average elapsed time. Returns an empty slice (not an error) when the join produces no rows.

All queries use parameterized `?` placeholders (no string interpolation). Results wrapped with `fmt.Errorf("failed to get velocity data: %w", err)`.

### 4.4 Service Layer — `internal/services/sprint_analytics_service.go`

**Repository interfaces (defined at point of use):**

```go
// SprintAnalyticsRepository is the interface the analytics service requires
// from the repository layer. The concrete implementation is
// *sprint.SprintAnalyticsRepository.
type SprintAnalyticsRepository interface {
    GetVelocityData(ctx context.Context, limit int) ([]sprint.VelocityRow, error)
    GetSprintAssignedEntities(ctx context.Context, sprintID int64) ([]sprint.AssignedEntity, error)
    GetCompletionEvents(ctx context.Context, sprintID int64, start, end time.Time) ([]sprint.TaskCompletionEvent, error)
    GetCycleTimeByPhase(ctx context.Context, sprintID int64) ([]sprint.PhaseTimeRow, error)
}

// SprintAnalyticsService orchestrates sprint analytics queries.
type SprintAnalyticsService struct {
    analyticsRepo SprintAnalyticsRepository  // required
    sprintRepo    SprintRepository            // required — reuses interface from sprint_service.go
    // workSessionRepo is optional; when nil, cycle-time fields in --detailed degrade gracefully.
}

func NewSprintAnalyticsService(
    analyticsRepo SprintAnalyticsRepository,
    sprintRepo SprintRepository,
) *SprintAnalyticsService
```

**Key methods:**

```go
// GetVelocity returns velocity data for the last N completed sprints.
// N defaults to 5 when ≤ 0.
func (s *SprintAnalyticsService) GetVelocity(ctx context.Context, n int) (*VelocityResult, error)

// GetBurndown returns burndown data for the sprint identified by key.
// If key is empty, uses the current active sprint.
func (s *SprintAnalyticsService) GetBurndown(ctx context.Context, sprintKey string) (*BurndownResult, error)

// GetSummary returns a sprint summary report. When detailed is true,
// additional fields are populated (cycle time, size bands, carryover list).
// Cycle-time fields degrade to nil when work_sessions data is unavailable.
func (s *SprintAnalyticsService) GetSummary(ctx context.Context, sprintKey string, detailed bool) (*SprintSummaryResult, error)
```

**Velocity calculation logic (in `GetVelocity`):**

1. Call `analyticsRepo.GetVelocityData(ctx, n)`.
2. Sum `VelocityRow.CompletedSize` values to compute the trailing average (mean over all rows, including rows with zero).
3. Return `VelocityResult` with per-sprint data and trailing average.
4. If len(rows) < 3, set `VelocityResult.InsufficientData = true` (caller renders message).

**Burndown calculation logic (in `GetBurndown`):**

1. If `sprintKey` is empty, call `sprintRepo.List(ctx, &sprint.SprintListFilters{Status: ptr("active")})` and take the first result.
2. Call `analyticsRepo.GetSprintAssignedEntities(ctx, sprint.ID)` to get all assignment records with their sizes.
3. Compute the initial sprint size as Σ of sizes of all entities assigned from `start_date` (i.e., `assigned_at <= start_date` and `removed_at IS NULL OR removed_at > start_date`).
4. For each calendar day D from `start_date` to today (or `end_date`, whichever is earlier):
   - Compute active entities on day D: assigned before or on D, not removed before D.
   - Compute completed entities on day D: those with a `TaskCompletionEvent.Timestamp <= end-of-day D`.
   - `actual_remaining = Σ(size of active - completed entities)`.
   - `ideal_remaining` = linear interpolation (sprint total size × days_remaining / sprint_duration_days).
5. Piecewise recalculation: if a net size change occurred on day D (entity added or removed), recompute ideal from the new total for the remaining days.
6. Future days (D > today) have `ActualRemaining = nil` in the DTO.

**Summary calculation logic (in `GetSummary`):**

1. Load sprint via `sprintRepo.GetByKey(ctx, sprintKey)`.
2. Validate status is `completed` or `archived`; return informational error otherwise.
3. Load all assigned entities via `GetSprintAssignedEntities`.
4. Compute planned Σ size (entities assigned at or before `start_date`), completed Σ size (entities with `CompletionEvent` within sprint window).
5. Compute trailing average by calling `GetVelocity(ctx, 6)` and taking the first 5 rows (excluding this sprint).
6. If `detailed`:
   - Load completion events for phase duration via `GetCycleTimeByPhase`.
   - Compute size-band distribution from completed entities.
   - Build carryover list from entities still assigned (`removed_at IS NULL`) and not completed.

### 4.5 DTOs — `internal/services/sprint_analytics_dto.go`

```go
type VelocityResult struct {
    Sprints         []VelocitySprint
    TrailingAverage float64
    SprintCount     int
    InsufficientData bool // true when SprintCount < 3
}

type VelocitySprint struct {
    Key              string
    Name             string
    CompletedSize    int
    UnsizedCompleted int
}

type BurndownResult struct {
    SprintKey   string
    SprintName  string
    TotalSize   int
    UnsizedTotal int
    DataPoints  []BurndownDataPoint
}

type BurndownDataPoint struct {
    Date             time.Time
    IdealRemaining   float64
    ActualRemaining  *float64 // nil for future dates
    UnsizedRemaining int
}

type SprintSummaryResult struct {
    SprintKey            string
    SprintName           string
    PlannedSize          int
    CompletedSize        int
    CompletionPctBySize  float64
    PlannedCount         int
    CompletedCount       int
    VelocityThisSprint   int
    TrailingAvgVelocity  float64
    VelocityDelta        float64 // absolute
    VelocityDeltaPct     float64 // percentage
    UnsizedPlanned       int
    UnsizedCompleted     int
    // Detailed fields (nil/empty when detailed=false):
    AddedMidSprintCount  *int
    AddedMidSprintSize   *int
    RemovedMidSprintCount *int
    RemovedMidSprintSize  *int
    CycleTimeByPhase     []PhaseTime // nil when work_sessions unavailable
    AvgCompletedSize     *float64
    SizeBandDistribution []SizeBand
    CarryoverEntities    []CarryoverEntity
}

type PhaseTime struct {
    Phase       string
    AverageDays float64
}

type SizeBand struct {
    Label string // XS, S, M, L, XL, XXL
    Count int
}

type CarryoverEntity struct {
    Key        string
    EntityType string
    Size       *int
}
```

### 4.6 CLI Commands — `internal/cli/commands/sprint.go`

Three new commands added to the existing `sprintCmd` parent (follows the same pattern as the existing `sprintCreateCmd`, `sprintGetCmd` in this file):

**`sprintVelocityCmd`:**
```
Use:   "velocity [--sprints=N] [--json]"
Short: "Show historical sprint velocity"
RunE:  runSprintVelocity
```

Handler: parse `--sprints` flag (default 5, validate 1-100), call `cli.GetSprintAnalyticsService().GetVelocity(ctx, n)`, format as table or JSON.

**`sprintBurndownCmd`:**
```
Use:   "burndown [SPRINT_KEY] [--json]"
Short: "Show sprint burndown chart"
Args:  cobra.MaximumNArgs(1)
RunE:  runSprintBurndown
```

Handler: parse optional key arg (empty = active sprint), call `GetBurndown`, format as table or JSON.

**`sprintSummaryCmd`:**
```
Use:   "summary SPRINT_KEY [--detailed] [--json]"
Short: "Sprint summary report for retrospectives"
Args:  cobra.ExactArgs(1)
RunE:  runSprintSummary
```

Handler: parse key and `--detailed` flag, call `GetSummary`, format as table or JSON.

All three commands follow the three-step pattern: parse → call service → format output. No business logic in command handlers. Follows `internal/cli/commands/bug.go` as the reference for command registration style.

### 4.7 Service Accessor — `internal/cli/services_global.go`

```go
// GetSprintAnalyticsService returns a SprintAnalyticsService instance.
// Creates a new instance per call with the global DB.
func GetSprintAnalyticsService() *services.SprintAnalyticsService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    analyticsRepo := sprint.NewSprintAnalyticsRepository(db)
    sprintRepo := repository.NewSprintRepository(db)
    return services.NewSprintAnalyticsService(analyticsRepo, sprintRepo)
}
```

Pattern matches `GetSprintService()` already present in this file (E19-F02).

---

## 5. Key Technical Decisions

**Decision 1: Burndown from task_history, not a separate time-series table.**
Rationale: `task_history` already exists with `old_status`, `new_status`, `timestamp` columns (confirmed in `internal/repository/task/history.go:46`). Reconstructing from this table avoids schema changes. Limitation: non-task entities (bugs, change_cards, tech_debts) do not have status-history tables, so their burndown contribution uses point-in-time status. This is documented in the output legend.

**Decision 2: No separate analytics table or caching.**
Rationale: Sprint analytics are read rarely (retrospectives, daily standups). The <2 s performance target is achievable with the existing indexes for up to 50 sprints/1000 tasks. Caching introduces staleness complexity disproportionate to the benefit for a CLI tool.

**Decision 3: `SprintAnalyticsService` is separate from `SprintService`.**
Rationale: Follows the `DashboardAnalyticsService` precedent in `internal/services/dashboard_analytics_service.go` (E18-F01 pattern). Analytics logic is read-only and has different repository dependencies; keeping it separate maintains single responsibility and makes each service independently testable.

**Decision 4: Cycle-time uses `task_history.timestamp` deltas, not `work_sessions`.**
Rationale: The feature description mentions cycle-time-by-phase "from work_sessions with graceful degradation when E13 data is absent." After inspection, `work_sessions` tracks session start/end but not the workflow phase directly. Phase-level durations are more accurately computed from `task_history` transitions (`old_status` → `new_status` with timestamps). `work_sessions` is referenced for compatibility but the primary source is `task_history`. The `--detailed` output labels this as "phase duration from status history" to avoid misleading the user.

**Decision 5: Unsized entities contribute 0 to all sums but are always counted separately.**
Rationale: Per REQ-F-007 through REQ-F-009 and the cross-cutting dependency note in `requirements.md` (see "Cross-Cutting Dependency: Entity Size" section). The service layer enforces this; no repository query treats NULL size as an implicit value.

---

## 6. Integration with Existing Code

| File | Change type | What changes |
|---|---|---|
| `internal/repository/sprint/analytics.go` | **New file** | `SprintAnalyticsRepository` struct + 4 query methods |
| `internal/services/sprint_analytics_service.go` | **New file** | `SprintAnalyticsService` struct + 3 business logic methods |
| `internal/services/sprint_analytics_dto.go` | **New file** | 7 output DTO structs |
| `internal/cli/commands/sprint.go` | **Modify** | Add 3 command vars + 3 RunE handlers + flag registration in `init()` |
| `internal/cli/services_global.go` | **Modify** | Add `GetSprintAnalyticsService()` (~15 lines) |

No changes to:
- Database schema / `internal/db/db.go` (no migration needed)
- `internal/models/` (no new model types)
- `internal/keys/` (S### parsing already complete from E19-F01)
- `internal/services/sprint_service.go` (reuses its `SprintRepository` interface)

---

## 7. Test Plan

All tests follow patterns in `.claude/rules/testing/architecture.md`.

### 7.1 Repository Tests — `internal/repository/sprint/analytics_test.go` (new)

Use `test.GetTestDB()` (real database). Clean up `sprints`, `sprint_assignments` rows with TEST prefix before each test.

| Test | Asserts |
|---|---|
| `TestGetVelocityData_CompletedSprints` | Returns correct Σ size for each sprint; unsized entities counted separately |
| `TestGetVelocityData_Limit` | Returns at most `limit` sprints in oldest-first order |
| `TestGetVelocityData_Empty` | Returns empty slice (not error) when no completed sprints |
| `TestGetSprintAssignedEntities` | Returns all assignments (active and soft-deleted) for a sprint |
| `TestGetCompletionEvents` | Returns task_history events within sprint window; filters out events outside window |
| `TestGetCycleTimeByPhase` | Returns non-empty slice when task_history has transitions; empty slice when none |

### 7.2 Service Tests — `internal/services/sprint_analytics_service_test.go` (new)

Use mock repositories (function-field pattern, consistent with `internal/services/mocks_test.go`).

| Test | Asserts |
|---|---|
| `TestGetVelocity_Happy` | Correct trailing average, per-sprint data |
| `TestGetVelocity_InsufficientData` | `InsufficientData = true` when < 3 sprints |
| `TestGetVelocity_LimitValidation` | Returns error for N < 1 or N > 100 |
| `TestGetBurndown_ActiveSprint` | Uses active sprint when no key provided; future days have nil ActualRemaining |
| `TestGetBurndown_PlanningStatus` | Returns informational error for planning sprint |
| `TestGetBurndown_IdealLineReset` | Piecewise ideal line recalculates after mid-sprint entity add |
| `TestGetSummary_CompletedSprint` | All base fields populated correctly |
| `TestGetSummary_ActiveSprint` | Returns informational error |
| `TestGetSummary_Detailed_WithSessionData` | CycleTimeByPhase populated |
| `TestGetSummary_Detailed_NoSessionData` | CycleTimeByPhase is nil; no error |
| `TestGetSummary_UnsizedCounting` | UnsizedPlanned and UnsizedCompleted counted correctly |

### 7.3 CLI Command Tests — extension of `internal/cli/commands/sprint_test.go`

Mock `SprintAnalyticsService`. Test argument parsing and output shape only.

| Test | Asserts |
|---|---|
| `TestSprintVelocityCmd_DefaultLimit` | Calls service with n=5 by default |
| `TestSprintVelocityCmd_CustomLimit` | Parses `--sprints=10` correctly |
| `TestSprintVelocityCmd_InvalidLimit` | Returns error for `--sprints=0` |
| `TestSprintVelocityCmd_JSON` | JSON output matches VelocityResult schema |
| `TestSprintBurndownCmd_NoKey` | Passes empty string to service (active sprint) |
| `TestSprintBurndownCmd_WithKey` | Passes key to service |
| `TestSprintSummaryCmd_JSON` | JSON output matches SprintSummaryResult schema |
| `TestSprintSummaryCmd_Detailed` | Passes `detailed=true` to service |

---

## 8. Acceptance Criteria

### From REQ-F-007 (Velocity)

- [ ] **AC-F007-1** — `shark sprint velocity` outputs last 5 completed sprints with Σ size and unsized_completed per sprint.
- [ ] **AC-F007-2** — `--sprints=N` flag overrides the count (validated 1–100).
- [ ] **AC-F007-3** — Trailing average is the mean of completed Σ sizes.
- [ ] **AC-F007-4** — "insufficient data" message shown when < 3 completed sprints.
- [ ] **AC-F007-5** — `--json` output matches the schema in AC-V-6.

### From REQ-F-008 (Burndown)

- [ ] **AC-F008-1** — `shark sprint burndown` uses active sprint when no key given.
- [ ] **AC-F008-2** — Ideal line is linear; resets piecewise on mid-sprint entity changes.
- [ ] **AC-F008-3** — Actual remaining reconstructed from `task_history` (tasks) and current status (other types).
- [ ] **AC-F008-4** — Future days show `—` in human output, omitted from JSON data_points.
- [ ] **AC-F008-5** — `unsized_remaining` included in each data point.
- [ ] **AC-F008-6** — `--json` output matches schema in AC-B-7.

### From REQ-F-009 (Summary)

- [ ] **AC-F009-1** — `shark sprint summary S024` returns all base summary fields for a completed sprint.
- [ ] **AC-F009-2** — `--detailed` adds cycle-time-by-phase, size-band distribution, carryover list.
- [ ] **AC-F009-3** — Cycle-time section shows "No session data available" gracefully when absent.
- [ ] **AC-F009-4** — Planning or active sprint returns informational message, exit 0.
- [ ] **AC-F009-5** — `--json` includes all populated fields; nil for unavailable detailed fields.

### Cross-cutting

- [ ] **AC-X-1** — `make fmt && make lint && make test` all pass.
- [ ] **AC-X-2** — Analytics commands complete in < 2 s on test dataset (50 sprints, 1000 tasks).
- [ ] **AC-X-3** — No regression in existing sprint or task tests.
- [ ] **AC-X-4** — All service tests use mocked repositories (no real DB in service tests).

---

## 9. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Burndown reconstruction inaccurate for non-task entities (no history table) | Medium | Document limitation in output legend; use point-in-time status. Deferred to future feature if history tables are added for bugs/change_cards. |
| `GetVelocityData` UNION ALL query becomes slow for large entity counts | Low | Existing indexes on `sprint_assignments(sprint_id)` and entity `size` columns (from E07-F42 migration) are sufficient for 1000 tasks. Add `EXPLAIN QUERY PLAN` assertion in performance test. |
| `--detailed` fails hard when work_sessions is empty | Medium | Service returns nil CycleTimeByPhase (not error) when query returns 0 rows. CLI renders "No session data available" string instead of nil display. Covered by `TestGetSummary_Detailed_NoSessionData`. |
| Velocity trailing average misleading when sprints have 0 completed items | Low | Document in output (trailing average includes zero-velocity sprints in denominator). User can use `--sprints=N` to exclude early data collection sprints. |

---

*End of spec.*
