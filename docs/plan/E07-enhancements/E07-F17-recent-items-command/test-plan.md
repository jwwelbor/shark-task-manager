---
feature_key: E07-F17-recent-items-command
epic_key: E07
title: Test Plan — Recent Items Command
status: in_test_planning
last_updated: 2026-04-27
---

# Test Plan: Recent Items Command (E07-F17)

> Traces to: `spec.md` acceptance criteria (AC-1 through AC-7) and functional requirements
> (REQ-F-001 through REQ-F-011, REQ-NF-001 through REQ-NF-004).
> E07 is a placeholder epic with no UAT plan; all UAT scenarios are defined at this feature level.

---

## 1. AC Test Matrix

Each acceptance criterion from `spec.md` is broken out into discrete test cases with
input/setup, expected outcome, and edge cases. Every test case maps to a named test function
that will be written during implementation.

---

### AC-1: Happy path — default limit with no config override

**REQ traces:** REQ-F-001, REQ-F-002, REQ-F-006, REQ-F-007, REQ-NF-001

**Test case: TC-AC1-01 — Default limit returns 5 rows from single-type DB**

- Setup: 12 tasks in DB, no `recent.default_limit` in `.sharkconfig.json`
- Input: `shark recent` (no args, no flags)
- Expected: exit 0, table with exactly 5 rows, all `Type=task`, rows ordered `Created DESC`
- Test layer: CLI integration / service (mocked repos for speed)

**Test case: TC-AC1-02 — Ordering is strictly descending by `created_at`**

- Setup: seed 5 tasks with distinct `created_at` values spread 1 second apart
- Input: `shark recent 5`
- Expected: `created_at[0] >= created_at[1] >= ... >= created_at[4]`
- Test layer: repository (real DB, `TestTaskRepository_GetRecent_OrdersByCreatedAtDesc`)

**Edge cases for AC-1:**

| Edge case | Input | Expected |
|---|---|---|
| Exactly 5 rows in DB | `shark recent` | All 5 rows returned |
| Fewer than 5 rows in DB | `shark recent` | All available rows returned (len < 5), exit 0 |
| 1 row in DB | `shark recent` | 1 row returned |

---

### AC-2: Positional limit + type filter combination

**REQ traces:** REQ-F-003, REQ-F-005, REQ-F-006

**Test case: TC-AC2-01 — Type filter excludes unwanted types**

- Setup: 3 epics, 5 features, 20 tasks in DB
- Input: `shark recent 4 --features --epics`
- Expected: exit 0, exactly 4 rows, no `Type=task` rows, `created_at` descending
- Test layer: service (mocked repos: task repo returns `[]`, feature+epic repos return data)

**Test case: TC-AC2-02 — Single type filter — tasks only**

- Setup: 3 epics, 5 features, 8 tasks
- Input: `shark recent --tasks`
- Expected: up to 5 rows, all `Type=task`, feature/epic repos never called
- Test layer: service (assert feature/epic mock `GetRecentFunc` is NOT invoked)

**Test case: TC-AC2-03 — Combined flags — tasks + epics, no features**

- Setup: 2 epics, 3 features, 4 tasks
- Input: `shark recent --tasks --epics`
- Expected: up to 5 rows, no `Type=feature` rows
- Test layer: service (assert feature mock `GetRecentFunc` is NOT invoked)

**Edge cases for AC-2:**

| Edge case | Input | Expected |
|---|---|---|
| Type filter with no matching entities | `--features` on a DB with 0 features | exit 0, empty state message |
| All three flags together | `--tasks --features --epics` | Equivalent to no flag (all types) |
| Limit = 1, mixed types | `shark recent 1 --tasks --features` | Exactly 1 row, the most-recent of all tasks+features |

---

### AC-3: JSON output, all types

**REQ traces:** REQ-F-008

**Test case: TC-AC3-01 — JSON array shape and field names**

- Setup: at least 1 epic, 1 feature, 1 task in DB
- Input: `shark recent --json`
- Expected: stdout is valid JSON array, each element has exactly the keys
  `{type, key, title, created_at, status}`, `created_at` is RFC3339, `type` is
  one of `"epic"`, `"feature"`, `"task"`
- Test layer: CLI (mock service returns 3 `RecentItem` values, assert JSON)

**Test case: TC-AC3-02 — JSON mode with no items — emits empty array**

- Setup: empty DB
- Input: `shark recent --json`
- Expected: stdout is `[]`, exit 0
- Test layer: CLI (mock service returns `[]RecentItem{}`)

**Test case: TC-AC3-03 — JSON mode with `--field type` extraction**

- Setup: non-empty DB
- Input: `shark recent --json --field type` (global `--field` flag)
- Expected: stdout is a single string (`"epic"`, `"feature"`, or `"task"`) matching
  `.[0].type` of the full JSON
- Test layer: CLI (existing `--field` flag behaviour; no new code needed)

**Edge cases for AC-3:**

| Edge case | Input | Expected |
|---|---|---|
| `created_at` zero value | Mock returns item with zero `time.Time` | Serializes to RFC3339 zero `"0001-01-01T00:00:00Z"` |
| Status with spaces/special chars | Any status with unusual characters | Still valid JSON string |

---

### AC-4: Config-driven default limit

**REQ traces:** REQ-F-002, REQ-F-010, REQ-F-011

**Test case: TC-AC4-01 — Config `default_limit: 3` is respected**

- Setup: 10 tasks, `.sharkconfig.json` with `"recent": {"default_limit": 3}`
- Input: `shark recent`
- Expected: exactly 3 rows returned
- Test layer: CLI (mock config returns 3, assert service called with `Limit=3`)

**Test case: TC-AC4-02 — All four config resolution states**

Run as table-driven unit test in `config_test.go`:

| Config state | `default_limit` field | Expected `GetRecentDefaultLimit()` |
|---|---|---|
| `recent` section absent | (n/a) | 5 |
| `recent` section present, field absent | (n/a) | 5 |
| `recent.default_limit` = 0 | 0 | 5 |
| `recent.default_limit` = -1 | -1 | 5 |
| `recent.default_limit` = 7 | 7 | 7 |

- Test layer: pure unit test (`config_test.go`, no DB)

**Test case: TC-AC4-03 — Existing config without `recent` section loads without error**

- Setup: load the current production `.sharkconfig.json` (which has no `recent` key)
- Expected: no validation error, `GetRecentDefaultLimit()` returns 5
- Test layer: config integration test using `NewManager(configPath).Load()`

**Edge cases for AC-4:**

| Edge case | Input | Expected |
|---|---|---|
| `default_limit` = 10000 (max boundary) | Config file | Used as-is (10000) |
| `default_limit` = 10001 (above max) | Config file | Falls back to 5 (non-positive guard in `GetRecentDefaultLimit`) |

---

### AC-5: Empty state

**REQ traces:** REQ-F-009

**Test case: TC-AC5-01 — Empty database, table mode**

- Setup: empty DB (no epics, features, or tasks)
- Input: `shark recent`
- Expected: exit 0, stdout contains `No recent items found.`, no table rendered
- Test layer: CLI (mock service returns `[]`)

**Test case: TC-AC5-02 — Empty database, JSON mode**

- Setup: empty DB
- Input: `shark recent --json`
- Expected: exit 0, stdout is `[]`
- Test layer: CLI (mock service returns `[]`)

**Test case: TC-AC5-03 — Type filter yields no results (non-empty DB)**

- Setup: DB has only tasks, no features
- Input: `shark recent --features`
- Expected: exit 0, empty state message
- Test layer: service (feature mock returns `[]`, task mock not called because `IncludeTasks=false`)

---

### AC-6: Invalid argument — exit code 3

**REQ traces:** REQ-F-003, REQ-NF-004

**Test case: TC-AC6-01 — Non-integer positional argument**

- Input: `shark recent abc`
- Expected: exit 3, stderr names the offending argument (`"abc"`)
- Test layer: CLI unit test (no service call should occur)

**Test case: TC-AC6-02 — Zero as positional argument**

- Input: `shark recent 0`
- Expected: exit 3, stderr includes argument context
- Test layer: CLI unit test

**Test case: TC-AC6-03 — Negative integer as positional argument**

- Input: `shark recent -1`
- Expected: exit 3, stderr includes argument context
- Test layer: CLI unit test

**Test case: TC-AC6-04 — Limit above maximum (10001)**

- Input: `shark recent 10001`
- Expected: exit 3
- Test layer: CLI unit test (`parseRecentFilters` bounds check)

**Test case: TC-AC6-05 — `--limit` flag with zero**

- Input: `shark recent --limit=0`
- Expected: exit 3
- Test layer: CLI unit test

**Edge cases for AC-6:**

| Edge case | Input | Expected |
|---|---|---|
| Float string | `shark recent 3.5` | exit 3 (not a valid integer) |
| Whitespace-only | `shark recent "  "` | exit 3 (empty after trim) |
| Very large number (overflow) | `shark recent 99999999999` | exit 3 |
| Boundary: exactly 10000 | `shark recent 10000` | exit 0, up to 10000 rows |
| Boundary: exactly 1 | `shark recent 1` | exit 0, exactly 1 row |

---

### AC-7: Flag overrides positional argument

**REQ traces:** REQ-F-004

**Test case: TC-AC7-01 — `--limit` flag wins over positional**

- Setup: DB with 30+ items
- Input: `shark recent 5 --limit=20`
- Expected: at most 20 rows returned
- Test layer: CLI unit test (assert `filters.Limit == 20`)

**Test case: TC-AC7-02 — Warning printed in table mode when both given**

- Input: `shark recent 5 --limit=20` (table mode, no `--json`)
- Expected: a warning message is printed to stderr/output before the table
- Test layer: CLI output capture test

**Test case: TC-AC7-03 — No warning in JSON mode**

- Input: `shark recent 5 --limit=20 --json`
- Expected: stdout is valid JSON array only (no warning text mixed in)
- Test layer: CLI output capture test (assert stdout parses as JSON cleanly)

---

## 2. Integration Scenarios

These scenarios cover cross-component interactions that span multiple layers (CLI → Service → Repository → DB). They complement the unit-level AC tests.

---

### IS-1: Full CLI-to-DB round-trip (repository integration)

**Components involved:** `GetRecent` on `TaskRepository`, `FeatureRepository`, `EpicRepository` → real SQLite DB via `test.GetTestDB()`

**What to verify at the boundary:**
- The SQL query `ORDER BY created_at DESC LIMIT ?` executes correctly against real schema
- Returns a non-nil empty slice (not `nil`) when no rows exist
- `created_at` field in returned models matches the values inserted at seed time

**Test cases (one per entity type, in their respective `repository_test.go` files):**

| Test function | File | Scenario |
|---|---|---|
| `TestTaskRepository_GetRecent_OrdersByCreatedAtDesc` | `internal/repository/task/repository_test.go` | Seed 5 tasks with distinct timestamps; assert order |
| `TestTaskRepository_GetRecent_LimitRespected` | same | Seed 10, request 3; assert `len == 3` |
| `TestTaskRepository_GetRecent_EmptyTable` | same | Clean slate; assert `len == 0`, not `nil` |
| `TestTaskRepository_GetRecent_LimitExceedsRowCount` | same | Seed 2, request 100; assert `len == 2` |
| `TestFeatureRepository_GetRecent_OrdersByCreatedAtDesc` | `internal/repository/feature/repository_test.go` | Mirrors task pattern |
| `TestFeatureRepository_GetRecent_LimitRespected` | same | Mirrors task pattern |
| `TestFeatureRepository_GetRecent_EmptyTable` | same | Mirrors task pattern |
| `TestEpicRepository_GetRecent_OrdersByCreatedAtDesc` | `internal/repository/epic/repository_test.go` | Mirrors task pattern |
| `TestEpicRepository_GetRecent_LimitRespected` | same | Mirrors task pattern |
| `TestEpicRepository_GetRecent_EmptyTable` | same | Mirrors task pattern |

**Seeding note:** Repository tests in `internal/repository/task/` create their own epics and features
directly (as seen in `repository_test.go` lines 30-60), then clean up with `defer` blocks. The new
`GetRecent` tests follow the same self-contained seeding pattern — no shared `test.SeedTestData()` needed.

---

### IS-2: Service merge-and-sort logic (mocked repos)

**Components involved:** `RecentService.ListRecent` orchestrates three mock repos, merges, sorts, applies final limit.

**What to verify at the boundary:**
- When all three Include* flags are false, all three repos are called (default-all behavior)
- When only one Include* is true, the other two repos are NOT called
- Merged results are sorted `created_at DESC` regardless of which repo returned which item
- Tie-breaking: `epic` before `feature` before `task`, then `key ASC` (REQ-F-006)
- Error from any single repo aborts the whole operation; error message contains `"failed to list recent <type>"`

**Test cases (in `internal/services/recent_service_test.go`):**

| Test function | Scenario |
|---|---|
| `TestRecentService_DefaultFilters_IncludesAllTypes` | All three Include* false → all repos called |
| `TestRecentService_SingleTypeFilter_SkipsOtherRepos` | `IncludeTasks=true` only → feature+epic mocks NOT called |
| `TestRecentService_MergesAndSortsAcrossTypes` | Mock returns out-of-order timestamps → assert merged order is descending |
| `TestRecentService_AppliesFinalLimitAfterMerge` | 3 repos × 5 items each, limit=4 → returns 4 most-recent across all |
| `TestRecentService_RepositoryErrorIsWrapped` | Task repo returns error → error contains `"failed to list recent task"` |
| `TestRecentService_EmptyReposReturnEmptySlice` | All repos return `[]` → service returns `[]RecentItem{}`, not nil |
| `TestRecentService_TieBreakByTypeThenKey` | Items with identical `created_at` → deterministic: epic < feature < task, then key ASC |

---

### IS-3: CLI argument parsing to service call (mocked service)

**Components involved:** `parseRecentFilters` helper → `RecentFilters` DTO → mock `RecentService`

**What to verify at the boundary:**
- Cobra arg parsing correctly populates `RecentFilters`
- Config default limit is read from `cli.GetConfig()` when no arg/flag given
- Invalid args are caught in `parseRecentFilters` before service is called (service mock not invoked)
- Output formatters (`renderRecentTable`, `cli.OutputJSON`) consume `[]RecentItem` correctly

**Test cases (in `internal/cli/commands/recent_test.go`):**

| Test function | Scenario |
|---|---|
| `TestRunRecent_ParsesPositionalLimit` | `args=["10"]` → `filters.Limit == 10` |
| `TestRunRecent_LimitFlagOverridesPositional` | `args=["5"]`, `--limit=20` → `filters.Limit == 20` |
| `TestRunRecent_FallsBackToConfigDefault` | Config returns 7, no args/flags → `filters.Limit == 7` |
| `TestRunRecent_FallsBackToBuiltInDefault` | Config returns nil, no args/flags → `filters.Limit == 5` |
| `TestRunRecent_InvalidLimitReturnsExit3` | `args=["abc"]` → returns error, service NOT called |
| `TestRunRecent_TypeFlagsSetCorrectly` | `--tasks --epics` → `IncludeTasks && IncludeEpics && !IncludeFeatures` |
| `TestRunRecent_JSONOutputEmitsArray` | JSON mode, mock returns 2 items → stdout parses as 2-element array |
| `TestRunRecent_EmptyStateMessageInTableMode` | Mock returns `[]` → stdout contains `"No recent items found."` |

---

### IS-4: Config loading integration

**Components involved:** `NewManager(path).Load()` → `Config.Recent` pointer → `GetRecentDefaultLimit()`

**What to verify at the boundary:**
- JSON unmarshalling correctly populates `*RecentConfig` when section is present
- `nil` pointer when section is absent (backward compatibility — REQ-F-011)
- `GetRecentDefaultLimit()` safely handles nil receiver, nil `Recent`, zero, and negative values

**Test cases (in `internal/config/config_test.go`):**

| Test function | Scenario |
|---|---|
| `TestGetRecentDefaultLimit_NilConfig` | Nil `*Config` receiver → returns 5 |
| `TestGetRecentDefaultLimit_SectionAbsent` | Config without `recent` key → `Recent == nil`, returns 5 |
| `TestGetRecentDefaultLimit_FieldZero` | `{"recent": {"default_limit": 0}}` → returns 5 |
| `TestGetRecentDefaultLimit_FieldNegative` | `{"recent": {"default_limit": -1}}` → returns 5 |
| `TestGetRecentDefaultLimit_FieldPositive` | `{"recent": {"default_limit": 7}}` → returns 7 |
| `TestRecentConfig_BackwardCompat_ExistingConfigLoadsOK` | Load real-looking config JSON without `recent` section → no error |

---

### IS-5: Performance benchmark (repository)

**Components involved:** `TaskRepository.GetRecent` against a seeded real DB

**What to verify:** `GetRecent(ctx, 100)` on a 10 000-row table completes in < 150ms (REQ-NF-001)

**Test function:** `BenchmarkTaskRepository_GetRecent` in `internal/repository/task/repository_test.go`

```
// Seed 10000 tasks, then:
b.ResetTimer()
for i := 0; i < b.N; i++ {
    _, _ = repo.GetRecent(ctx, 100)
}
```

Run with: `go test -bench=BenchmarkTaskRepository_GetRecent -benchtime=5s ./internal/repository/task/`

---

## 3. Test Infrastructure

### 3.1 Existing infrastructure to reuse

| Infrastructure | Location | Used by |
|---|---|---|
| `test.GetTestDB()` | `internal/test/testdb.go` | All new repository tests |
| `dbconn.NewDB(database)` | `internal/repository/dbconn/db.go` | Repository test setup (see `task/repository_test.go:22`) |
| `epic.NewEpicRepository`, `feature.NewFeatureRepository` | respective packages | Seeding parent entities in task repo tests |
| `models.Epic`, `models.Feature`, `models.Task` structs | `internal/models/` | Seeding test data |
| `testify/assert`, `testify/require` | project dependency | All new tests |
| Mock function-field pattern | `internal/services/search_service_test.go` lines 14-24 | Service mock pattern to replicate |
| Config file loading via `NewManager(path).Load()` | `internal/config/config_observability_test.go` lines 12-41 | Config integration tests |
| `os.WriteFile` + `t.TempDir()` for config fixtures | same file | Config unit tests |

### 3.2 New test helpers needed

**In `internal/repository/task/repository_test.go` (or a new `recent_test.go` file):**

```go
// seedTasksWithTimestamps creates n tasks with created_at staggered by 1 second each.
// Returns task IDs for cleanup via defer.
func seedTasksWithTimestamps(t *testing.T, repo *TaskRepository, db *sql.DB,
    epicID, featureID int64, n int) []int64
```

**In `internal/repository/feature/repository_test.go`:**

```go
// seedFeaturesWithTimestamps — mirrors task helper, creates n features under epicID.
func seedFeaturesWithTimestamps(t *testing.T, repo *FeatureRepository, db *sql.DB,
    epicID int64, n int) []int64
```

**In `internal/repository/epic/repository_test.go`:**

```go
// seedEpicsWithTimestamps — creates n epics with staggered created_at.
func seedEpicsWithTimestamps(t *testing.T, repo *EpicRepository, db *sql.DB, n int) []int64
```

**In `internal/services/recent_service_test.go`:**

Three mock types following the function-field pattern (see `search_service_test.go` for the exact structure):

```go
type mockRecentTaskRepo struct {
    GetRecentFunc func(ctx context.Context, limit int) ([]*models.Task, error)
}

type mockRecentFeatureRepo struct {
    GetRecentFunc func(ctx context.Context, limit int) ([]*models.Feature, error)
}

type mockRecentEpicRepo struct {
    GetRecentFunc func(ctx context.Context, limit int) ([]*models.Epic, error)
}
```

Each mock delegates to its `GetRecentFunc` if set, otherwise returns a descriptive "not implemented" error.

**In `internal/cli/commands/recent_test.go`:**

A mock `RecentService` interface and implementation matching the `ListRecent` signature:

```go
type mockRecentService struct {
    ListRecentFunc func(ctx context.Context, filters services.RecentFilters) ([]services.RecentItem, error)
}
```

The CLI tests also need to inject the mock service without hitting the real DB. Follow the same pattern used in `internal/cli/commands/status_group_test.go` (which uses `cli.GlobalConfig` manipulation + test-local helper setup).

### 3.3 Key file locations for new tests

| New test file | Layer | Pattern |
|---|---|---|
| `internal/repository/task/repository_test.go` (add to) | Repository | Real DB + cleanup (see existing `TestTaskRepository_Create_GeneratesAndStoresSlug`) |
| `internal/repository/feature/repository_test.go` (add to) | Repository | Same as task |
| `internal/repository/epic/repository_test.go` (add to) | Repository | Same as task |
| `internal/services/recent_service_test.go` (new file) | Service | Mocked repos, function-field pattern |
| `internal/config/config_test.go` (add to) | Config | `os.WriteFile` + `NewManager().Load()` |
| `internal/cli/commands/recent_test.go` (new file) | CLI | Mocked service, no real DB |

### 3.4 Test naming conventions

Follow the existing project convention observed in the codebase:

- Repository: `TestXxxRepository_MethodName_Scenario`
  - Example: `TestTaskRepository_GetRecent_OrdersByCreatedAtDesc`
- Service: `TestXxxService_MethodName_Scenario`
  - Example: `TestRecentService_MergesAndSortsAcrossTypes`
- CLI: `TestRunXxx_Scenario` or `TestXxxCommand_Scenario`
  - Example: `TestRunRecent_ParsesPositionalLimit`
- Config: `TestGetXxx_Scenario`
  - Example: `TestGetRecentDefaultLimit_SectionAbsent`

### 3.5 Test isolation requirements

- All repository tests: clean up test data **before** test starts (DELETE WHERE key/id), plus `defer` cleanup after
- Use unique key prefixes to avoid collision: `TEST-RECENT-E90`, `TEST-RECENT-F01`, `TEST-RECENT-T001`
- Service tests: never touch the database — all repos are mocked
- CLI tests: never call `test.GetTestDB()` — mock the service, not the repos
- Config tests: use `t.TempDir()` + `os.WriteFile` for config fixtures, never modify the real `.sharkconfig.json`

---

## 4. Quality Gates

Before implementation is complete, all of the following must pass:

- [ ] `make fmt` — no formatting changes
- [ ] `make lint` — zero lint errors
- [ ] `make test` — full test suite passes (including all new tests above)
- [ ] Every AC (AC-1 through AC-7) has at least one passing automated test
- [ ] Every REQ-NF (REQ-NF-001 through REQ-NF-004) has a corresponding test or verification note
- [ ] `go test -bench=BenchmarkTaskRepository_GetRecent ./internal/repository/task/` — completes in < 150ms per op

---

## 5. Exit Gate Self-Check

- [x] Every AC in spec.md (AC-1 through AC-7) has at least one test case in this plan
- [x] Edge cases identified for each AC (boundary values, empty state, type combination variations)
- [x] Integration scenarios cover all cross-component boundaries (repo→DB, service→repo, CLI→service, config→CLI)
- [x] Test infrastructure references existing files by path; no orphaned test patterns
- [x] Mocking strategy follows project conventions (`search_service_test.go`, `feature/repository_test.go`)
- [x] Repository tests use real DB with cleanup; service+CLI tests use mocks only (per `.claude/rules/testing/architecture.md`)
- [x] No orphaned tests — every test traces to a named REQ-F-* or REQ-NF-* or AC-* requirement
