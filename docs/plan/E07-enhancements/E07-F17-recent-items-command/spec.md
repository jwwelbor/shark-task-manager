---
feature_key: E07-F17-recent-items-command
epic_key: E07
title: Recent Items Command
status: in_specification
spec_type: combined-requirements-architecture
last_updated: 2026-04-26
---

# Spec: Recent Items Command (E07-F17)

> Combined requirements + architecture document. See `docs/plan/E07-enhancements/epic.md` for parent epic context.

---

## 1. Overview (Incremental Scope)

E07 ("Enhancements") is a placeholder epic that holds quality-of-life improvements to the Shark CLI. This feature adds **one new top-level command, `shark recent`**, that lists the most recently created entities across tasks, features, and epics.

### Problem (incremental over epic)

Today an operator who wants to find "what was just created" must run multiple list commands and visually scan timestamps. There is no single view of recent activity, and `created_at` is not exposed as a sort key on any existing list command.

### Solution (this feature only)

Add `shark recent [N]` that returns the newest items across all three primary entity types, sorted by `created_at DESC`, with optional type filters and a configurable default limit.

### What this feature is **not**

See [Section 7. Out of Scope](#7-out-of-scope).

---

## 2. Functional Requirements

All requirements are testable. IDs are unique within this feature.

### Core command

**REQ-F-001 — `shark recent` top-level command exists**
- A new top-level Cobra command `recent` is registered on `cli.RootCmd` with `GroupID: "inspect"`.
- `shark recent --help` returns exit code 0 and shows usage.
- Acceptance: `./bin/shark recent --help` lists `recent` in the help text and exits 0.

**REQ-F-002 — Default limit comes from config**
- When no `[N]` positional argument and no `--limit` flag are given, the limit is read from `.sharkconfig.json` field `recent.default_limit`.
- If `recent.default_limit` is unset, missing, or `<= 0`, the limit defaults to **5**.
- Acceptance: With `recent.default_limit: 7` in config, `shark recent` returns at most 7 items. With no config field, it returns at most 5.

**REQ-F-003 — Positional limit argument**
- `shark recent N` (where `N` is a positive integer, e.g. `shark recent 10`) sets the limit to `N`.
- `N` must be a positive integer; invalid values exit with code 3 (invalid state) and an error message naming the offending argument.
- Acceptance: `shark recent 10` returns at most 10 items. `shark recent abc` exits 3. `shark recent 0` exits 3. `shark recent -1` exits 3.

**REQ-F-004 — `--limit` flag**
- `--limit=N` is equivalent to passing `N` positionally.
- If both a positional `N` and `--limit=N` are given, **the flag wins** (matches Cobra precedence elsewhere in the codebase) and a warning is printed in non-JSON mode.
- Acceptance: `shark recent 5 --limit=20` returns at most 20 items. JSON mode prints no warning.

**REQ-F-005 — Type filter flags**
- `--tasks`: include only tasks.
- `--features`: include only features.
- `--epics`: include only epics.
- Flags are independent and may be combined (e.g. `--tasks --features`). With no type flag, all three types are included.
- Acceptance: `shark recent --tasks` returns only rows where `Type == "task"`. `shark recent --tasks --epics` returns no features.

**REQ-F-006 — Cross-type ordering**
- Results are merged across the included types and sorted by `created_at DESC`. Ties are broken by entity type in the order `epic, feature, task` (deterministic for tests), then by `key` ascending.
- Acceptance: A task created at `T+1s` appears before a feature created at `T+0s`.

**REQ-F-007 — Output columns (table)**
- Default human output is a table with columns, in order: **Type, Key, Title, Created, Status**.
- `Created` is rendered as RFC3339 in UTC (e.g. `2026-04-26T15:04:05Z`).
- Status uses the entity's typed status string (`models.TaskStatus`, `models.FeatureStatus`, `models.EpicStatus`).
- Acceptance: `shark recent --no-color` output, when split into rows, has 5 columns matching the header.

**REQ-F-008 — JSON output**
- `--json` (and the global `--field <name>` flag, by inheritance from `cli.GlobalConfig`) emits a JSON array of objects with fields `type`, `key`, `title`, `created_at` (RFC3339), `status`.
- Acceptance: `shark recent --json | jq '.[0].type'` returns `"epic"`, `"feature"`, or `"task"`.

**REQ-F-009 — Empty result handling**
- When no entities exist (or none match the type filters), the command exits 0 and prints `No recent items found.` (table mode) or `[]` (JSON mode).
- Acceptance: An empty database returns exit 0 with the empty-state message.

### Configuration

**REQ-F-010 — `recent` config section**
- `.sharkconfig.json` accepts an optional new section:
  ```json
  "recent": {
    "default_limit": 5
  }
  ```
- Absent section, absent field, or non-positive value all resolve to the built-in default of 5 (REQ-F-002).
- Acceptance: All four config states (section absent / field absent / field = 0 / field = 7) yield the documented limit.

**REQ-F-011 — Backward compatibility**
- Existing `.sharkconfig.json` files without a `recent` section continue to load and validate.
- Acceptance: Loading the current production `.sharkconfig.json` (which has no `recent` section) returns no validation error.

---

## 3. Non-Functional Requirements

**REQ-NF-001 — Performance: bounded query**
- Each entity-type query MUST execute as `ORDER BY created_at DESC LIMIT ?` against the relevant table, taking advantage of any existing index on `created_at` (or a full-table scan acceptable since the row count is small in practice). Total wall-clock time for `shark recent 100` MUST complete in under **150 ms** on a local SQLite database with up to 10 000 rows per table on the developer's reference machine.
- Measurement: a benchmark added to `internal/repository/task/repository_test.go` (or equivalent) exercising `GetRecent(ctx, 100)` against a seeded test DB.

**REQ-NF-002 — Backend agnosticism**
- The query MUST work identically against local SQLite and Turso libSQL (no SQLite-only syntax such as `RANDOM()` tricks). Standard `ORDER BY ... LIMIT` is portable to both.
- Verification: the new repository methods are exercised by the existing repository test suite which runs against the test DB injected by `internal/test/testdb.go`.

**REQ-NF-003 — No new dependencies**
- No new third-party Go modules. Use existing stdlib + project dependencies (`cobra`, `database/sql`, `pterm`).

**REQ-NF-004 — Input sanitization**
- The integer limit is parsed with `strconv.Atoi` and bounded `1 <= limit <= 10_000`. Values outside this range exit code 3.
- Type filter flags are booleans only — no string-based status injection point exists.
- Justification: follows `.claude/rules/go/input-sanitization.md` "Layer 1: Model Layer — Structural Validation".

---

## 4. Acceptance Criteria (feature-level, Given/When/Then)

**AC-1 (happy path, default limit)**
- **Given** a database with 12 tasks created over the last hour and `recent.default_limit` unset
- **When** I run `shark recent`
- **Then** exit code is 0
- **And** the table shows exactly 5 rows
- **And** all 5 rows have `Type=task`
- **And** rows are ordered by `Created` descending

**AC-2 (positional limit + type filter)**
- **Given** a database with 3 epics, 5 features, 20 tasks
- **When** I run `shark recent 4 --features --epics`
- **Then** exit code is 0
- **And** the table shows exactly 4 rows
- **And** no row has `Type=task`
- **And** the first row's `Created` is `>=` the last row's `Created`

**AC-3 (JSON output, all types)**
- **Given** a database with at least one row in each of epics, features, tasks
- **When** I run `shark recent --json`
- **Then** stdout parses as a JSON array of length `min(5, total_rows)`
- **And** each element has keys `{type, key, title, created_at, status}` and no others
- **And** `created_at` parses as RFC3339

**AC-4 (config-driven default)**
- **Given** `.sharkconfig.json` contains `"recent": {"default_limit": 3}`
- **When** I run `shark recent`
- **Then** the table has exactly 3 rows

**AC-5 (empty state)**
- **Given** an empty database (no tasks, features, or epics)
- **When** I run `shark recent`
- **Then** exit code is 0
- **And** stdout contains `No recent items found.`

**AC-6 (invalid argument)**
- **Given** any database state
- **When** I run `shark recent abc`
- **Then** exit code is 3
- **And** stderr names the invalid argument

**AC-7 (flag overrides positional)**
- **Given** any database with 30+ items
- **When** I run `shark recent 5 --limit=20`
- **Then** the result has at most 20 rows

---

## 5. Architecture (this feature)

The implementation follows the standard three-layer pattern documented in `.claude/rules/architecture.md` (CLI → Service → Repository). No deviations.

### 5.1 Component Map

| Layer | New / Modified | Path | Role |
|---|---|---|---|
| CLI command | **NEW** | `internal/cli/commands/recent.go` | Thin Cobra wrapper: parse args/flags, call service, format output |
| CLI command tests | **NEW** | `internal/cli/commands/recent_test.go` | Mocked-service tests of arg parsing, flag handling, output |
| Service | **NEW** | `internal/services/recent_service.go` | Orchestration: load default limit from config, fan out to 3 repos, merge + sort, apply final limit |
| Service DTO | **NEW** | `internal/services/recent_dto.go` | `RecentFilters` input + `RecentItem` output struct |
| Service tests | **NEW** | `internal/services/recent_service_test.go` | Mocks of the three repo methods; covers all REQ-F-* and edge cases |
| Service accessor | **MODIFIED** | `internal/cli/service_accessors.go` | Add `GetRecentService()` (lazy init pattern matching `GetEpicService`) |
| Task repository | **MODIFIED** | `internal/repository/task/repository.go` | Add `GetRecent(ctx context.Context, limit int) ([]*models.Task, error)` |
| Feature repository | **MODIFIED** | `internal/repository/feature/repository.go` | Add `GetRecent(ctx context.Context, limit int) ([]*models.Feature, error)` |
| Epic repository | **MODIFIED** | `internal/repository/epic/repository.go` | Add `GetRecent(ctx context.Context, limit int) ([]*models.Epic, error)` |
| Repository tests | **MODIFIED** | `internal/repository/{task,feature,epic}/repository_test.go` | Add `TestXxx_GetRecent_*` cases with real test DB + cleanup |
| Config | **MODIFIED** | `internal/config/config.go` | Add `Recent *RecentConfig` field + `GetRecentDefaultLimit() int` method |
| Config tests | **MODIFIED** | `internal/config/config_test.go` | Cover all four resolution states for `default_limit` |
| Workflow registration | **MODIFIED** | `internal/cli/root.go` (verify) | Confirm `inspect` group is registered (it already is — see `docs/cli-reference/README.md`) |

**No changes to**: database schema (no migration), workflow JSON, models package, status package, sync, discovery, templates, or HTTP server. This feature is purely additive and does not touch any existing query or state-mutating code path.

### 5.2 Data Model Changes

**Database schema**: **None**. The required `created_at` column already exists on all three tables (`epics`, `features`, `tasks`) via `models.BaseEntity` and is already populated by every existing `Create` path. No migration. `CurrentSchemaVersion` is **not** bumped.

**Configuration schema (additive)**:
```go
// internal/config/config.go (new type)
type RecentConfig struct {
    DefaultLimit int `json:"default_limit,omitempty"`
}
```
Added as `Recent *RecentConfig \`json:"recent,omitempty"\`` on `Config`. The `omitempty` + pointer pattern matches existing optional sub-configs (`Web`, `Maintainer`, `Observability`).

**Index considerations**:
- A `LIMIT N ORDER BY created_at DESC` query on tables with ≤ 10 000 rows requires no new index — full scan + sort completes in single-digit milliseconds.
- If profiling shows a hot path on a larger dataset later, an index on `created_at DESC` can be added in a separate task. Out of scope here.

### 5.3 Interface Contracts

#### 5.3.1 Repository methods (all three follow the same pattern)

```go
// internal/repository/task/repository.go
//
// GetRecent returns the most recently created tasks, ordered by created_at DESC.
// limit must be positive; the caller (service) is responsible for bounds-checking.
// Returns an empty slice (not nil) if no rows exist.
func (r *TaskRepository) GetRecent(ctx context.Context, limit int) ([]*models.Task, error)
```

Implementation pattern (matches existing `List`):
```sql
SELECT <existing-task-column-list>
FROM tasks
ORDER BY created_at DESC
LIMIT ?
```
Re-use the existing `r.queryTasks(ctx, query, limit)` helper for row scanning. No new column list — copy the one used by `List(ctx)` at `internal/repository/task/repository.go:714-718`.

Identical signatures (with their respective model types) for `FeatureRepository.GetRecent` and `EpicRepository.GetRecent`.

#### 5.3.2 Service interfaces and DTO

```go
// internal/services/recent_dto.go
package services

import "time"

type RecentFilters struct {
    Limit          int  // resolved positive value (service callers pass the final number)
    IncludeTasks   bool
    IncludeFeatures bool
    IncludeEpics   bool
}

type RecentItem struct {
    Type      string    `json:"type"`       // "epic" | "feature" | "task"
    Key       string    `json:"key"`
    Title     string    `json:"title"`
    CreatedAt time.Time `json:"created_at"`
    Status    string    `json:"status"`
}
```

```go
// internal/services/recent_service.go
package services

// Repository interfaces — defined here (consumer side) per
// .claude/rules/go/patterns.md "Interface Design".
type recentTaskRepo interface {
    GetRecent(ctx context.Context, limit int) ([]*models.Task, error)
}
type recentFeatureRepo interface {
    GetRecent(ctx context.Context, limit int) ([]*models.Feature, error)
}
type recentEpicRepo interface {
    GetRecent(ctx context.Context, limit int) ([]*models.Epic, error)
}

type RecentService struct {
    taskRepo    recentTaskRepo
    featureRepo recentFeatureRepo
    epicRepo    recentEpicRepo
}

func NewRecentService(t recentTaskRepo, f recentFeatureRepo, e recentEpicRepo) *RecentService { ... }

// ListRecent fans out to enabled repos, merges, sorts by CreatedAt DESC,
// and returns the top `filters.Limit` items.
//
// If no Include* flag is true, all three are treated as included (so a default
// invocation from the command works without explicit flags).
//
// Returns an empty slice (not nil) if no items found. Errors from any single
// repository call abort the whole operation and are wrapped with "failed to
// list recent <type>: %w".
func (s *RecentService) ListRecent(ctx context.Context, filters RecentFilters) ([]RecentItem, error)
```

#### 5.3.3 Config accessor

```go
// internal/config/config.go (new method on *Config)
//
// GetRecentDefaultLimit returns the configured default limit for `shark recent`,
// or 5 if the field is missing, the section is absent, or the value is <= 0.
func (c *Config) GetRecentDefaultLimit() int {
    const builtinDefault = 5
    if c == nil || c.Recent == nil || c.Recent.DefaultLimit <= 0 {
        return builtinDefault
    }
    return c.Recent.DefaultLimit
}
```

#### 5.3.4 CLI accessor

```go
// internal/cli/service_accessors.go (new function, follows GetEpicService pattern)
func GetRecentService() *services.RecentService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    workflowSvc := GetWorkflowService()
    taskRepo := repository.NewTaskRepositoryWithWorkflow(db, workflowSvc.GetWorkflow())
    featureRepo := repository.NewFeatureRepository(db)
    epicRepo := repository.NewEpicRepository(db)
    return services.NewRecentService(taskRepo, featureRepo, epicRepo)
}
```

#### 5.3.5 Cobra command (skeleton)

```go
// internal/cli/commands/recent.go
var recentCmd = &cobra.Command{
    Use:     "recent [N]",
    Short:   "List most recently created entities across tasks, features, and epics",
    GroupID: "inspect",
    Args:    cobra.MaximumNArgs(1),
    RunE:    runRecent,
}

func init() {
    cli.RootCmd.AddCommand(recentCmd)
    recentCmd.Flags().Int("limit", 0, "Limit results (overrides positional and config default)")
    recentCmd.Flags().Bool("tasks", false, "Show only tasks")
    recentCmd.Flags().Bool("features", false, "Show only features")
    recentCmd.Flags().Bool("epics", false, "Show only epics")
}

func runRecent(cmd *cobra.Command, args []string) error {
    filters, err := parseRecentFilters(cmd, args) // helper
    if err != nil {
        return err // exit code 3 via existing error mapping
    }
    svc := cli.GetRecentService()
    items, err := svc.ListRecent(cmd.Context(), filters)
    if err != nil {
        return err
    }
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(items)
    }
    return renderRecentTable(items) // helper
}
```

The helper `parseRecentFilters` resolves the limit precedence (`--limit` > positional > config > 5) and applies the bounds check (`1 <= limit <= 10000`). The helper `renderRecentTable` emits either the empty-state line or `cli.OutputTable(headers, rows)`.

### 5.4 Key Technical Decisions

| Decision | Rationale | Alternative Rejected |
|---|---|---|
| Per-type repository methods (`GetRecent`) instead of one polymorphic query | Each entity has its own table and column set; following existing repo patterns (`List`, `ListByFeature`) keeps repos pure data access. Aligns with `.claude/rules/architecture.md` "Repository Pattern". | A `UNION ALL` across the three tables would couple the SQL to schema details and complicate slug/relationship handling. |
| Service merges in-memory after three small queries | Each query is bounded by `LIMIT N`, so the largest possible in-memory set is `3 * limit`. Sorting `3 * 100 = 300` items is trivial. Keeps SQL portable across SQLite and Turso. | A single SQL `UNION ALL ... ORDER BY ... LIMIT` would be marginally faster but tightly couples three table layouts. |
| New `Recent` sub-config object (vs. flat `recent_default_limit` field) | Matches the existing pattern for `Web`, `Maintainer`, `Observability` — leaves room for future fields (e.g. `recent.show_completed`). | A flat top-level field would not scale and breaks the established sub-config convention. |
| Pointer (`*RecentConfig`) on `Config` | Matches existing optional-section pattern; `nil` cleanly means "not configured" without ambiguous zero values. | A non-pointer struct with a sentinel zero value would force every reader to know that `0` means "use default". |
| No new database index | Expected dataset sizes (≤ 10 000 rows / table) make a sort-then-limit fast enough; adding indexes is a separate maintenance concern. Confirmed against REQ-NF-001. | Adding `CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC)` now would require a migration + version bump for negligible benefit. |
| No new database migration / no `CurrentSchemaVersion` bump | All required columns (`created_at`) already exist on every entity table via `models.BaseEntity`. See `.claude/rules/database-critical.md` "When You Add a Migration". | A schema change is unnecessary and would invalidate the "no DDL on every command" optimization. |
| Service interfaces defined consumer-side (in `recent_service.go`, not the repo packages) | Follows `.claude/rules/go/patterns.md` "Interface Design" — accept interfaces, return structs, define interfaces at the point of use. Keeps repos free of speculative abstractions. | Defining a shared `RecentRepo` interface in a new package adds indirection with no caller benefit. |
| Empty results return `[]RecentItem{}`, not `nil` | Predictable for both JSON output (`[]` vs `null`) and table rendering. | Returning `nil` requires the command layer to guard against nil-vs-empty differences in two output paths. |
| Flag `--limit` wins over positional argument | Matches Cobra precedence used elsewhere in the codebase (e.g. `task create --order=N` overrides positional defaults). REQ-F-004 documents this explicitly. | Positional-wins would surprise users who add `--limit` later in their invocation. |

### 5.5 Integration Points (precise)

- **`cli.RootCmd`** at `internal/cli/root.go` — `recent.go`'s `init()` calls `cli.RootCmd.AddCommand(recentCmd)`. The `inspect` GroupID is already registered (see `internal/cli/commands/list.go:9` for prior art).
- **`cli.GlobalConfig`** — read `cli.GlobalConfig.JSON` for output mode selection. No changes to GlobalConfig itself.
- **`cli.GetDB`, `cli.GetWorkflowService`** at `internal/cli/db_global.go` and `internal/cli/workflow_global.go` — used inside the new `GetRecentService()` accessor exactly as in `GetEpicService()` (`internal/cli/service_accessors.go:138-167`).
- **`cli.GetConfig`** at `internal/cli/root.go:360` — called by `parseRecentFilters` to obtain the project config and resolve `recent.default_limit`. If `GetConfig()` returns an error, the helper falls back to the built-in default of 5 (consistent with how `GetConfigPath` failures are tolerated elsewhere).
- **`config.NewManager(...).Load()`** at `internal/config/manager.go:28` — must continue to deserialize the new `recent` field via the existing `json.Unmarshal` into `Config`. No code change required there; the new field tag is sufficient.
- **`models.BaseEntity.CreatedAt`** at `internal/models/entity.go:56` — the source of truth for `created_at` on all three entity types. The `GetCreatedAt()` accessor at `internal/models/entity.go:63` is used by the service to read into `RecentItem.CreatedAt`.
- **Existing repository row scanners** — re-use `queryTasks` (`internal/repository/task/repository.go` near `List`), and the equivalent scan helpers in feature/epic repositories. No new scan logic is introduced.
- **`cli.OutputJSON` / `cli.OutputTable` / `cli.Success`** at `internal/cli/output.go` — used as in every other command.

### 5.6 Error Handling

Per `.claude/rules/go/error-handling.md`:

| Error source | Wrap/return | Exit code (CLI) |
|---|---|---|
| Invalid limit argument (not int, ≤ 0, > 10 000) | `fmt.Errorf("invalid limit %q: must be a positive integer ≤ 10000", arg)` | 3 |
| Repository query failure | `fmt.Errorf("failed to list recent <type>: %w", err)` | 2 |
| `cli.GetConfig()` failure | Logged, fall back to default 5 | 0 (non-fatal) |
| Empty result set | nil error, command renders empty-state line | 0 |

No new sentinel errors or custom error types are introduced.

---

## 6. Testing Strategy

### 6.1 Repository tests (real DB, follows `.claude/rules/testing/repository-tests.md`)

`internal/repository/task/repository_test.go`:
- `TestTaskRepository_GetRecent_OrdersByCreatedAtDesc` — seed 5 tasks with staggered `created_at`, assert order
- `TestTaskRepository_GetRecent_LimitRespected` — seed 10, request 3, assert len = 3
- `TestTaskRepository_GetRecent_EmptyTable` — empty table, assert empty (non-nil) slice
- `TestTaskRepository_GetRecent_LimitExceedsRowCount` — seed 2, request 100, assert len = 2

Identical test sets for `feature/repository_test.go` and `epic/repository_test.go`.

### 6.2 Service tests (mocked repos, follows `.claude/rules/services/testing.md`)

`internal/services/recent_service_test.go` (table-driven, mocks per the function-field pattern):
- `default_filters_includes_all_types` — all three Include* false → all three repos called
- `single_type_filter_skips_other_repos` — `IncludeTasks=true` only → feature/epic repos NOT called
- `merges_and_sorts_across_types` — mock returns out-of-order timestamps → assert merged result is sorted desc
- `applies_final_limit_after_merge` — three repos return 5 each, limit=4 → assert len = 4 of the 4 most-recent across all three
- `repository_error_is_wrapped` — task repo returns error → service returns wrapped error with `failed to list recent task` substring
- `empty_repos_return_empty_slice` — all three return `[]` → service returns `[]RecentItem{}` (not nil)
- `tie_break_by_type_then_key` — two items with identical `created_at` → deterministic order per REQ-F-006

### 6.3 CLI tests (mocked service, follows `.claude/rules/testing/cli-tests.md`)

`internal/cli/commands/recent_test.go`:
- `parses_positional_limit` — `args=["10"]` → `filters.Limit == 10`
- `parses_limit_flag_overrides_positional` — `args=["5"]`, `--limit=20` → `filters.Limit == 20`
- `falls_back_to_config_default` — config returns `7`, no args/flags → `filters.Limit == 7`
- `falls_back_to_built_in_default` — config returns nil, no args/flags → `filters.Limit == 5`
- `invalid_limit_returns_exit_3` — `args=["abc"]` → returns error mapping to exit 3
- `type_flags_set_correctly` — `--tasks --epics` → `IncludeTasks && IncludeEpics && !IncludeFeatures`
- `json_output_emits_array` — JSON mode, mock returns 2 items → stdout parses as 2-element JSON array
- `empty_state_message_in_table_mode` — mock returns `[]` → stdout contains `No recent items found.`

### 6.4 Config tests

`internal/config/config_test.go`:
- `GetRecentDefaultLimit_returns_5_when_nil_config`
- `GetRecentDefaultLimit_returns_5_when_section_absent`
- `GetRecentDefaultLimit_returns_5_when_field_zero`
- `GetRecentDefaultLimit_returns_5_when_field_negative`
- `GetRecentDefaultLimit_returns_configured_value_when_positive`

### 6.5 Quality gate

Per `.claude/rules/development-workflows.md`, before declaring complete: `make fmt && make lint && make test` must all pass.

---

## 7. Out of Scope

**Explicitly excluded from this feature:**

1. **Date-range filters (`--since`, `--until`, `--days=N`)** — Future enhancement; opens questions about timezone handling and parser semantics. Workaround: pipe `shark recent --json` through `jq`.
2. **Other entity types (bugs, change-cards, ideas)** — The original task description names tasks/features/epics only. Adding bugs and change-cards is mechanically straightforward but is left as a follow-up to keep this feature small.
3. **Sort by `updated_at`** — `created_at` only, per task description. A `--sort` flag would broaden the scope.
4. **Pagination / cursor support** — `--limit` only. Not required for the documented use case (recent N items).
5. **Per-type sub-limits** (e.g. "5 of each type") — Single global limit only.
6. **Database index on `created_at`** — Not needed at current data sizes (REQ-NF-001 + Section 5.4). If profiling later shows a need, add in a separate task with a proper migration.
7. **Schema migration / version bump** — All required columns already exist; no DDL.
8. **HTTP API endpoint** — `cmd/server` is not modified. Can be added later by reusing `RecentService` per the HTTP integration pattern at `.claude/rules/services/http-integration.md`.
9. **Status filter** — Bool type filters only. A `--status` flag would conflict with the multi-entity nature of the result set (each entity has a different status enum).

**Alternative approaches rejected:**

- **Polymorphic `entity_recent` view in SQL** — Would couple three table schemas and require a schema migration. The in-memory merge in the service is simpler, portable, and bounded.
- **Reusing `internal/services/search_service.go`** — Search semantics differ (text query, ranking) and `created_at DESC` is not a primary search concern. Cleanest separation is a dedicated `RecentService`.

---

## Exit Gate Self-Check

- [x] Every requirement is testable (each REQ-F-* and REQ-NF-* maps to a concrete acceptance criterion or test in §6)
- [x] Every architecture decision references existing patterns or explains deviation (§5.4 table)
- [x] File paths listed for all changes (§5.1 component map)
- [x] No TBDs in critical sections
- [x] Requirements are incremental over the parent epic (E07 is a placeholder; this feature adds one concrete capability)
- [x] Architecture aligns with `.claude/rules/architecture.md`, `.claude/rules/services/service-design.md`, and `.claude/rules/database/schema.md` (no migration needed)
