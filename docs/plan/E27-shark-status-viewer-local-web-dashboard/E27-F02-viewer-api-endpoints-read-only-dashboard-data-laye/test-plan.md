---
feature_key: E27-F02-viewer-api-endpoints-read-only-dashboard-data-laye
epic_key: E27
doc_type: test-plan
status: draft
author: qa
---

# E27-F02 — Test Plan: Viewer API Endpoints (Read-Only Dashboard Data Layer)

**Feature:** E27-F02 — Viewer API Endpoints - Read-Only Dashboard Data Layer
**Epic:** E27 — Shark Status Viewer — Local Web Dashboard
**Spec:** [spec.md](./spec.md)
**Dependency:** E27-F01 (`internal/dbinit` extraction) must be merged before integration tests run.

---

## 1. Testing Strategy Overview

Three test layers per `testing/architecture.md`, plus one full-stack smoke:

| Test File | Layer | Real DB? | Coverage Target |
|---|---|---|---|
| `internal/api/viewer/handler_test.go` | HTTP handler | No — mocks `ViewerServicer` | ≥ 85% |
| `internal/services/viewer_service_test.go` | Service | No — mocks all repos + workflow | ≥ 85% |
| `internal/repository/entityhistory/repository_test.go` | Repository | Yes — real test DB with cleanup | 100% branch |
| `cmd/server/viewer_integration_test.go` | Full-stack smoke | Yes — single integration test on temp DB | 7-endpoint happy path |

All handler tests use `net/http/httptest`. All service tests use function-field mock structs per `services/testing.md`. No test touches a real database except the repository layer and the integration smoke.

---

## 2. Mock Definitions

### 2.1 `MockViewerServicer` (handler tests)

Declare in `internal/api/viewer/handler_test.go` (or a shared `mock_test.go` in same package):

```go
type MockViewerServicer struct {
    SummaryFunc        func(ctx context.Context) (*SummaryResponse, error)
    HierarchyFunc      func(ctx context.Context) (*HierarchyResponse, error)
    HistoryFunc        func(ctx context.Context, key string) (*HistoryResponse, error)
    FileFunc           func(ctx context.Context, key string) (*FileResponse, error)
    FeatureTasksFunc   func(ctx context.Context, featureKey string, opts FeatureTaskOptions) (*FeatureTasksResponse, error)
    RecentActivityFunc func(ctx context.Context, opts RecentActivityOptions) (*RecentActivityResponse, error)
    WorkflowMetaFunc   func(ctx context.Context) (*WorkflowMetaResponse, error)
}

// Each method delegates to Func field if non-nil, else returns an error
// indicating "XFunc not set" (prevents accidental nil-pointer panics in tests).
```

### 2.2 Repository mocks (service tests)

Declare in `internal/services/viewer_service_test.go` (or a shared `mocks_test.go`):

- `MockEpicRepository` — `ListAllFunc`, `GetByKeyFunc`, `CountByStatusFunc`
- `MockFeatureRepository` — `ListByEpicIDFunc`, `GetByKeyFunc`, `CountByStatusFunc`
- `MockTaskRepository` — `ListByFeatureIDFunc`, `CountByFeatureIDFunc`, `CountBlockedByFeatureIDFunc`
- `MockBugRepository` — `CountByStatusFunc`, `CountBySeverityFunc`
- `MockChangeCardRepository` — `CountByStatusFunc`
- `MockEntityHistoryRepository` — `ListByEntityFunc`, `ListRecentAcrossEntitiesFunc`
- `MockWorkflowService` — `GetStatusMetaFunc`, `GetTransitionsFunc`, `AllLevelsFunc`

---

## 3. Handler Tests (`internal/api/viewer/handler_test.go`)

Each test uses `httptest.NewRecorder()` and `httptest.NewRequest(...)`. The handler is constructed with a `MockViewerServicer`. Assertions cover HTTP status code, `Content-Type: application/json`, and required top-level JSON keys.

### 3.1 `GET /api/v1/viewer/summary`

| TC-H-001 | Happy path — populated project |
|---|---|
| Mock | `SummaryFunc` returns a `SummaryResponse` with non-zero counts for all five entity types |
| Assert | HTTP 200; body contains `epics`, `features`, `tasks`, `bugs`, `change_cards`, `generated_at` |

| TC-H-002 | Happy path — empty project |
|---|---|
| Mock | `SummaryFunc` returns all-zero `SummaryResponse` (valid, not an error) |
| Assert | HTTP 200; `tasks.total = 0`, `epics.total = 0`, `status_counts` arrays are empty |

| TC-H-003 | Service error → 500 |
|---|---|
| Mock | `SummaryFunc` returns `nil, errors.New("db failure")` |
| Assert | HTTP 500; JSON error envelope present |

### 3.2 `GET /api/v1/viewer/hierarchy`

| TC-H-010 | Happy path — 1 epic, 2 features |
|---|---|
| Mock | `HierarchyFunc` returns hierarchy with one epic, two features |
| Assert | HTTP 200; `epics[0].features` has length 2; each feature has `task_count`, `blocked_count`, `status_color` |

| TC-H-011 | Empty project |
|---|---|
| Mock | `HierarchyFunc` returns `HierarchyResponse{Epics: []}` |
| Assert | HTTP 200; body is `{"epics":[]}` (not 404, not 500) |

### 3.3 `GET /api/v1/viewer/history/{key}` — 7 cases

| TC-H-020 | Happy path — epic key `E01` |
|---|---|
| Path param | `E01` |
| Mock | `HistoryFunc` returns 2 history entries, newest first |
| Assert | HTTP 200; `key = "E01"`; `entries` has 2 items; `entries[0].changed_at > entries[1].changed_at` |

| TC-H-021 | Happy path — feature key `E01-F01` |
|---|---|
| Path param | `E01-F01` |
| Mock | Returns valid `HistoryResponse` |
| Assert | HTTP 200 |

| TC-H-022 | Happy path — task key `E01-F01-001` |
|---|---|
| Path param | `E01-F01-001` |
| Mock | Returns valid `HistoryResponse` |
| Assert | HTTP 200 |

| TC-H-023 | Happy path — bug key `B001` |
|---|---|
| Path param | `B001` |
| Mock | Returns valid `HistoryResponse` |
| Assert | HTTP 200 |

| TC-H-024 | Happy path — change-card key `CC-001` |
|---|---|
| Path param | `CC-001` |
| Mock | Returns valid `HistoryResponse` |
| Assert | HTTP 200 |

| TC-H-025 | Invalid key format → 400 |
|---|---|
| Path param | `NOT-A-KEY` |
| Mock | `HistoryFunc` must NOT be called |
| Assert | HTTP 400; `{"error": "Bad Request", "message": "invalid entity key: NOT-A-KEY"}` |

| TC-H-026 | Valid format, not found → 404 |
|---|---|
| Path param | `E99` |
| Mock | `HistoryFunc` returns `nil, repository.NotFoundError{...}` |
| Assert | HTTP 404; error envelope present |

| TC-H-027 | Lowercase key normalized → valid lookup |
|---|---|
| Path param | `e01-f01-001` |
| Mock | `HistoryFunc` receives uppercase `E01-F01-001`; returns valid response |
| Assert | HTTP 200; key normalized before service call |

### 3.4 `GET /api/v1/viewer/file/{key}` — 7 cases

| TC-H-030 | Happy path — file exists |
|---|---|
| Path param | `E01` |
| Mock | `FileFunc` returns `{Key:"E01", Exists:true, Path:"docs/plan/E01/epic.md", Content:"# E01..."}` |
| Assert | HTTP 200; `exists = true`; `content` present; `path` is relative |

| TC-H-031 | File missing on disk — exists:false, HTTP 200 |
|---|---|
| Path param | `E01` |
| Mock | `FileFunc` returns `{Key:"E01", Exists:false}` (no error) |
| Assert | HTTP 200; `exists = false`; no `content` field |

| TC-H-032 | Security error (path traversal) → 403 |
|---|---|
| Path param | `E01` |
| Mock | `FileFunc` returns `nil, &SecurityError{Reason:"file path escapes project root"}` |
| Assert | HTTP 403; `{"error":"Forbidden","message":"file path escapes project root"}` |

| TC-H-033 | File too large → 413 |
|---|---|
| Path param | `E01` |
| Mock | `FileFunc` returns `nil, &FileTooLargeError{Size: 3*1024*1024, Limit: 2*1024*1024}` |
| Assert | HTTP 413; `{"error":"Payload Too Large","message":"file exceeds 2 MiB limit"}` |

| TC-H-034 | Invalid key format → 400 |
|---|---|
| Path param | `bad-key!!` |
| Mock | `FileFunc` must NOT be called |
| Assert | HTTP 400 |

| TC-H-035 | Valid format, not found → 404 |
|---|---|
| Path param | `E99` |
| Mock | `FileFunc` returns `nil, &NotFoundError{Entity:"epic", Key:"E99"}` |
| Assert | HTTP 404 |

| TC-H-036 | T-prefixed task key accepted |
|---|---|
| Path param | `T-E01-F01-001` |
| Mock | Returns valid `FileResponse` |
| Assert | HTTP 200 |

### 3.5 `GET /api/v1/viewer/features/{key}/tasks` — 5 cases

| TC-H-040 | Happy path |
|---|---|
| Path param | `E01-F01` |
| Mock | `FeatureTasksFunc` returns 5 tasks, `total=5` |
| Assert | HTTP 200; `feature_key`, `total`, `tasks` keys present; each task has `status_color` |

| TC-H-041 | limit clamp — `limit=9999` clamped to 500 |
|---|---|
| Query | `limit=9999` |
| Mock | `FeatureTasksFunc` receives `opts.Limit = 500` (clamped) — verify via assertion inside mock |
| Assert | HTTP 200; no 400 error |

| TC-H-042 | Invalid feature key → 400 |
|---|---|
| Path param | `NOTAFEATURE` |
| Mock | Must NOT be called |
| Assert | HTTP 400 |

| TC-H-043 | Valid feature key, not found → 404 |
|---|---|
| Path param | `E99-F99` |
| Mock | `FeatureTasksFunc` returns not-found error |
| Assert | HTTP 404 |

| TC-H-044 | `blocked=invalid` → 400 |
|---|---|
| Query | `blocked=maybe` |
| Mock | Must NOT be called |
| Assert | HTTP 400 |

### 3.6 `GET /api/v1/viewer/recent-activity` — 6 cases

| TC-H-050 | Happy path — no filters |
|---|---|
| Mock | `RecentActivityFunc` returns 10 entries |
| Assert | HTTP 200; `limit`, `entries` keys present; entries have `entity_type`, `key`, `title`, `from_status`, `to_status`, `changed_at` |

| TC-H-051 | `entity_type=task` filter |
|---|---|
| Query | `entity_type=task` |
| Mock | `RecentActivityFunc` receives `opts.EntityType = "task"` |
| Assert | HTTP 200 |

| TC-H-052 | `since` filter — valid RFC3339 |
|---|---|
| Query | `since=2026-01-01T00:00:00Z` |
| Mock | `RecentActivityFunc` receives non-nil `opts.Since` |
| Assert | HTTP 200 |

| TC-H-053 | Invalid `entity_type` → 400 |
|---|---|
| Query | `entity_type=blorp` |
| Mock | Must NOT be called |
| Assert | HTTP 400; error envelope |

| TC-H-054 | Malformed `since` → 400 |
|---|---|
| Query | `since=not-a-date` |
| Mock | Must NOT be called |
| Assert | HTTP 400 |

| TC-H-055 | limit clamp — `limit=9999` silently clamped to 200 |
|---|---|
| Query | `limit=9999` |
| Mock | `RecentActivityFunc` receives `opts.Limit = 200` |
| Assert | HTTP 200 |

### 3.7 `GET /api/v1/viewer/workflow-meta` — 2 cases

| TC-H-060 | Happy path — short workflow |
|---|---|
| Mock | `WorkflowMetaFunc` returns meta shaped as in spec §3.7, with 5 levels |
| Assert | HTTP 200; `levels` map contains `epic`, `feature`, `task`, `bug`, `change_card` keys; each level has `statuses` and `transitions` |

| TC-H-061 | Missing level emitted as empty object |
|---|---|
| Mock | `WorkflowMetaFunc` returns meta where `bug` level has `statuses:[]` and `transitions:[]` |
| Assert | HTTP 200; `levels.bug` key is present, not omitted |

### 3.8 CORS middleware (`cors.go`) — 4 cases

These tests call the `withLocalCORS` middleware directly (not through the full handler stack).

| TC-H-070 | `localhost` origin → CORS headers echoed |
|---|---|
| Request | `Origin: http://localhost:5173` |
| Assert | Response has `Access-Control-Allow-Origin: http://localhost:5173`; `Vary: Origin` present |

| TC-H-071 | `127.0.0.1` origin → CORS headers echoed |
|---|---|
| Request | `Origin: http://127.0.0.1:3000` |
| Assert | Response has `Access-Control-Allow-Origin: http://127.0.0.1:3000` |

| TC-H-072 | Non-local origin → no CORS header |
|---|---|
| Request | `Origin: http://evil.example.com` |
| Assert | Response does NOT have `Access-Control-Allow-Origin` header |

| TC-H-073 | OPTIONS preflight → 204, no body |
|---|---|
| Request | `OPTIONS /api/v1/viewer/summary`, `Origin: http://localhost:5173` |
| Assert | HTTP 204; `Access-Control-Allow-Methods` present; downstream handler NOT called |

| TC-H-074 | No `Origin` header → no CORS header, request proceeds normally |
|---|---|
| Request | No `Origin` header |
| Assert | No CORS headers; HTTP 200 from the wrapped handler |

---

## 4. Service Tests (`internal/services/viewer_service_test.go`)

All tests construct `ViewerService` via `NewViewerService(...)` injecting function-field mocks.

### 4.1 `Summary` method

| TC-S-001 | Happy path — mixed entity counts |
|---|---|
| Setup | EpicRepo returns 5 epics in 2 statuses; FeatureRepo returns 20 features; TaskRepo returns 150 tasks, 3 blocked; BugRepo returns 12 bugs with severity breakdown; ChangeCardRepo returns 4 change-cards |
| Assert | `SummaryResponse` reflects all counts; each `status_counts` entry has non-empty `color` and `phase` sourced from workflow mock (not hardcoded) |

| TC-S-002 | Empty project — all zeros, no error |
|---|---|
| Setup | All count methods return empty maps / 0 |
| Assert | Response has all `total = 0` and empty `status_counts` slices; no error |

| TC-S-003 | Unknown status in DB — fallback `color = "gray"`, `phase = "unknown"` |
|---|---|
| Setup | EpicRepo returns status `"legacy_status"` which is absent from workflow mock |
| Assert | Entry for `"legacy_status"` has `color = "gray"`, `phase = "unknown"` |

| TC-S-004 | Repository error propagates |
|---|---|
| Setup | `EpicRepo.CountByStatus` returns an error |
| Assert | `Summary` returns a wrapped error; no partial response |

### 4.2 `Hierarchy` method

| TC-S-010 | Happy path — epics in execution order |
|---|---|
| Setup | EpicRepo returns epics in scrambled order; features returned in scrambled execution_order |
| Assert | Epics sorted by `execution_order ASC, created_at ASC`; features within each epic also sorted; `task_count` and `blocked_count` correct |

| TC-S-011 | Epic with no features |
|---|---|
| Setup | Epic has no features |
| Assert | Epic included with `features: []`; not an error |

| TC-S-012 | Feature with zero tasks |
|---|---|
| Setup | Feature exists; task count = 0; blocked count = 0 |
| Assert | Feature has `task_count = 0`, `blocked_count = 0` |

| TC-S-013 | `status_color` resolved for each entity |
|---|---|
| Setup | Workflow mock returns `"yellow"` for status `"active"` |
| Assert | Epics and features with status `"active"` have `status_color = "yellow"` |

### 4.3 `History` method

| TC-S-020 | Happy path — epic key |
|---|---|
| Setup | `EpicRepo.GetByKey` returns epic; `HistoryRepo.ListByEntity` returns 3 entries |
| Assert | Entries returned newest-first; each entry has `id`, `from_status`, `to_status`, `changed_at` |

| TC-S-021 | Happy path — task key (detects type automatically) |
|---|---|
| Setup | `TaskRepo.GetByKey` called (not EpicRepo); 2 entries returned |
| Assert | HTTP 200 shape; correct entity type used for lookup |

| TC-S-022 | Not found — entity lookup returns NotFoundError |
|---|---|
| Setup | `EpicRepo.GetByKey` returns `NotFoundError` |
| Assert | Service returns `NotFoundError` (not wrapped into 500) |

| TC-S-023 | Empty history — entity exists, no history rows |
|---|---|
| Setup | `HistoryRepo.ListByEntity` returns empty slice |
| Assert | `entries = []`; no error |

### 4.4 `File` method

| TC-S-030 | Happy path — file exists within project root |
|---|---|
| Setup | `EpicRepo.GetByKey` returns epic with `file_path = "docs/plan/E01/epic.md"`; file exists on disk (use `t.TempDir()`) |
| Assert | Response has `exists = true`, `content` non-empty, `path` relative to project root |

| TC-S-031 | Path traversal in DB → `SecurityError` |
|---|---|
| Setup | `file_path = "../../etc/passwd"` in DB; temp project root |
| Assert | Returns `*SecurityError`; no file read attempted |

| TC-S-032 | Symlink escaping project root → `SecurityError` |
|---|---|
| Setup | `file_path` points to a symlink inside the project root that resolves outside; use `os.Symlink` in temp dir |
| Assert | Returns `*SecurityError` after `EvalSymlinks` reveals escape |

| TC-S-033 | File missing on disk → `exists = false`, no error |
|---|---|
| Setup | `file_path` is within project root but the file does not exist |
| Assert | Returns `FileResponse{Exists: false}`; no error |

| TC-S-034 | File > 2 MiB → `FileTooLargeError` |
|---|---|
| Setup | Create a temp file of exactly 2 MiB + 1 byte inside the project root |
| Assert | Returns `*FileTooLargeError`; file content not returned |

| TC-S-035 | File exactly 2 MiB → success |
|---|---|
| Setup | Temp file of exactly 2 MiB |
| Assert | Returns `FileResponse{Exists: true}`; content length = 2 MiB |

| TC-S-036 | Entity not found → NotFoundError |
|---|---|
| Setup | `EpicRepo.GetByKey` returns `NotFoundError` |
| Assert | `NotFoundError` returned from `File` method |

### 4.5 `FeatureTasks` method

| TC-S-040 | Happy path — 10 tasks ordered correctly |
|---|---|
| Setup | TaskRepo returns 10 tasks in random order |
| Assert | Tasks ordered `execution_order ASC, priority DESC, created_at ASC`; each task has `status_color` |

| TC-S-041 | Filter by status |
|---|---|
| Setup | 10 tasks: 4 `todo`, 6 `in_progress`; `opts.Status = "todo"` |
| Assert | Response `tasks` has 4 items; `total` is unfiltered count (10) |

| TC-S-042 | Filter by `blocked = true` |
|---|---|
| Setup | 10 tasks, 3 with `blocked = true`; `opts.Blocked = &trueVal` |
| Assert | 3 tasks in response; `total = 10` |

| TC-S-043 | `limit` clamping — negative limit → 0 default |
|---|---|
| Setup | `opts.Limit = -5` |
| Assert | Service uses default (200), no error returned |

| TC-S-044 | `limit` clamping — exceeds max → 500 |
|---|---|
| Setup | `opts.Limit = 9999` |
| Assert | Service uses 500 |

| TC-S-045 | `offset` clamping — negative offset → 0 |
|---|---|
| Setup | `opts.Offset = -10` |
| Assert | Service uses 0 |

| TC-S-046 | Feature not found → NotFoundError |
|---|---|
| Setup | `FeatureRepo.GetByKey` returns `NotFoundError` |
| Assert | Service returns `NotFoundError` |

### 4.6 `RecentActivity` method

| TC-S-050 | Happy path — mixed entity types |
|---|---|
| Setup | `HistoryRepo.ListRecentAcrossEntities` returns 10 entries across 3 entity types |
| Assert | Entries include `entity_type`, `key`, `title`, `from_status`, `to_status`, `changed_at` |

| TC-S-051 | Deleted entity omitted (title = nil from JOIN) |
|---|---|
| Setup | HistoryRepo returns an entry where the entity no longer exists (title is empty/nil in the join result) |
| Assert | That entry is excluded from the response |

| TC-S-052 | `entity_type` filter passed through to repository |
|---|---|
| Setup | `opts.EntityType = "task"` |
| Assert | `ListRecentAcrossEntities` called with `EntityType = "task"` |

| TC-S-053 | `since` filter passed through to repository |
|---|---|
| Setup | `opts.Since = &someTime` |
| Assert | `ListRecentAcrossEntities` called with non-nil `Since` matching `someTime` |

| TC-S-054 | limit clamp — exceeds max → 200 |
|---|---|
| Setup | `opts.Limit = 9999` |
| Assert | `ListRecentAcrossEntities` called with `Limit = 200` |

| TC-S-055 | limit default — 0 → 50 |
|---|---|
| Setup | `opts.Limit = 0` |
| Assert | `ListRecentAcrossEntities` called with `Limit = 50` |

### 4.7 `WorkflowMeta` method

| TC-S-060 | Happy path — 5 levels returned |
|---|---|
| Setup | Workflow mock returns statuses + transitions for epic, feature, task, bug, change_card levels |
| Assert | Response has `levels` with all 5 keys; each level has `statuses` and `transitions` |

| TC-S-061 | Direction computed correctly |
|---|---|
| Setup | Statuses in order `["draft", "active", "completed"]`; transitions include `draft→active`, `active→draft`, `active→completed` |
| Assert | `draft→active` direction = `"forward"`; `active→draft` direction = `"backward"`; `active→completed` direction = `"forward"` |

| TC-S-062 | Missing level → empty object, not omitted |
|---|---|
| Setup | Workflow mock returns no transitions for `change_card` |
| Assert | `levels.change_card` key is present in response with `statuses: []` and `transitions: []` |

| TC-S-063 | No hardcoded values — all from workflow mock |
|---|---|
| Setup | Workflow mock returns unusual color `"purple"` for a status |
| Assert | Response for that status has `color = "purple"` (not a hardcoded value) |

---

## 5. Repository Tests (`internal/repository/entityhistory/repository_test.go`)

These tests use `test.GetTestDB()` and follow the clean-before pattern from `testing/repository-tests.md`. All test data uses the `TEST-` key prefix.

### 5.1 `ListRecentAcrossEntities`

| TC-R-001 | Returns top N across all entity types, newest first |
|---|---|
| Setup | Seed 1 entity per type (5 total), 2 history rows each (10 rows). History rows have distinct `changed_at` timestamps spanning 10 minutes |
| Assert | With `Limit = 5`, exactly 5 rows returned; rows ordered DESC by `changed_at`; top row has the most recent timestamp |

| TC-R-002 | Filter by `EntityType = "task"` |
|---|---|
| Setup | Same 10 rows across 5 entity types |
| Assert | Only rows whose `entity_type = "task"` returned; other types absent |

| TC-R-003 | Filter by `Since` — rows before cutoff excluded |
|---|---|
| Setup | 10 rows, half before and half after a cutoff time |
| Assert | Only rows with `changed_at > cutoff` returned |

| TC-R-004 | Deleted entity row omitted (JOIN filter) |
|---|---|
| Setup | Seed 1 task with 2 history rows; then delete the task from DB; leave orphaned history rows |
| Assert | `ListRecentAcrossEntities` returns 0 rows for that task (JOIN omits orphans) |

| TC-R-005 | Mixed entity types carry correct `entity_type` label |
|---|---|
| Setup | 5 entities (one per type), 1 history row each |
| Assert | Each returned row has the correct `entity_type` string matching its source table |

| TC-R-006 | `Limit = 0` or negative is rejected or defaults |
|---|---|
| Setup | Caller passes `Limit = 0` (caller is required to clamp, but test verifies safe behavior) |
| Assert | No panic; returns at most some reasonable upper bound or empty slice |

| TC-R-007 | Combined filter: entity_type + since + limit |
|---|---|
| Setup | 30 rows across types and time range |
| Assert | Only rows matching all three constraints returned |

---

## 6. Integration Smoke Test (`cmd/server/viewer_integration_test.go`)

**Function: `TestViewerIntegration_HappyPath`**

**Setup:**
1. `test.GetTestDB()` — real SQLite in test mode
2. Seed: 1 epic, 2 features, 5 tasks per feature (10 tasks total), 3 history rows per task (30 history rows), 1 bug with severity `"high"`, 1 change-card
3. Create a temp project root directory with the seeded SQLite file
4. `WireServices(db, tempProjectRoot)` — full `ServiceContainer` including `ViewerService`
5. Register all routes including `viewer.NewHandler(svcs.ViewerService).RegisterRoutes(mux)`
6. `httptest.NewServer(mux)` — real HTTP stack

**Test steps (one per endpoint):**

| Step | Request | Assert |
|---|---|---|
| 1 | `GET /api/v1/viewer/summary` | HTTP 200; `tasks.total = 10`; `epics.total = 1`; `generated_at` present |
| 2 | `GET /api/v1/viewer/hierarchy` | HTTP 200; `epics` has length 1; `epics[0].features` has length 2 |
| 3 | `GET /api/v1/viewer/history/E01` | HTTP 200; `entries` present (may be empty for the test epic) |
| 4 | `GET /api/v1/viewer/file/E01` | HTTP 200; `exists` key present (file may or may not exist in temp dir) |
| 5 | `GET /api/v1/viewer/features/E01-F01/tasks` | HTTP 200; `feature_key = "E01-F01"`; `tasks` key present |
| 6 | `GET /api/v1/viewer/recent-activity` | HTTP 200; `entries` key present |
| 7 | `GET /api/v1/viewer/workflow-meta` | HTTP 200; `levels` map present with at least `task` and `epic` keys |

**Teardown:** close server; close DB; `os.RemoveAll(tempProjectRoot)`.

**Note:** This test guards against wiring regressions only. Shape assertions are minimal — unit tests cover detailed behavior.

---

## 7. Acceptance Criteria Matrix

Maps every spec §2.1, §2.2, and §2.3 requirement to the test cases that verify it.

### 7.1 Functional requirements

| Requirement | Description | Test Cases |
|---|---|---|
| REQ-F-001 | Entity-type counts with status breakdowns + color/phase | TC-S-001, TC-S-002, TC-S-003, TC-H-001, TC-H-002, TC-H-003, TC-R-001 (integration: step 1) |
| REQ-F-001 `generated_at` | `generated_at` is RFC3339Nano UTC in summary | TC-H-001 (verify field present + format) |
| REQ-F-001 `blocked_count` | Task `blocked_count` reflects DB `blocked = 1` | TC-S-001 |
| REQ-F-001 `severity_counts` | Bug severity map; absent severities omitted | TC-S-001 |
| REQ-F-001 legacy status fallback | `color = "gray"`, `phase = "unknown"` for unknown status | TC-S-003 |
| REQ-F-002 | Hierarchy — full epic→feature tree, execution order | TC-S-010, TC-S-011, TC-S-012, TC-S-013, TC-H-010, TC-H-011 |
| REQ-F-002 task_count / blocked_count | Per-feature counts correct | TC-S-010, TC-S-012 |
| REQ-F-003 | Unified history for any entity type | TC-S-020–023, TC-H-020–027 |
| REQ-F-003 key normalization | Case-insensitive key lookup | TC-H-027, TC-S-020 |
| REQ-F-003 newest-first | Entries DESC by changed_at | TC-S-020, TC-H-020 |
| REQ-F-003 invalid key → 400 | Bad format rejected | TC-H-025 |
| REQ-F-003 not found → 404 | Valid format, no entity | TC-H-026, TC-S-022 |
| REQ-F-004 | File endpoint — DB-path-first, containment | TC-S-030–036, TC-H-030–036 |
| REQ-F-004 path traversal → 403 | `../` in DB path rejected | TC-S-031, TC-H-032 |
| REQ-F-004 missing file → 200 `exists:false` | Not treated as 404 | TC-S-033, TC-H-031 |
| REQ-F-004 file > 2 MiB → 413 | LimitReader enforced | TC-S-034, TC-H-033 |
| REQ-F-004 symlink escape → 403 | EvalSymlinks used | TC-S-032 |
| REQ-F-005 | Feature tasks — filterable, paginated | TC-S-040–046, TC-H-040–044 |
| REQ-F-005 ordering | `execution_order ASC, priority DESC, created_at ASC` | TC-S-040 |
| REQ-F-005 limit clamp | > 500 silently clamped | TC-S-044, TC-H-041 |
| REQ-F-005 total = unfiltered count | Pagination total correct | TC-S-041, TC-S-042 |
| REQ-F-006 | Recent activity — cross-entity, filterable | TC-S-050–055, TC-H-050–055 |
| REQ-F-006 deleted entity omitted | Orphaned history rows excluded | TC-S-051, TC-R-004 |
| REQ-F-006 invalid entity_type → 400 | Enum check | TC-H-053 |
| REQ-F-006 malformed since → 400 | RFC3339 parse failure | TC-H-054 |
| REQ-F-006 limit clamp | > 200 silently clamped | TC-S-054, TC-H-055 |
| REQ-F-007 | Workflow meta — levels, statuses, transitions | TC-S-060–063, TC-H-060–061 |
| REQ-F-007 direction computed | forward/backward/lateral from ordinal | TC-S-061 |
| REQ-F-007 missing level → empty object | Stable shape for UI | TC-S-062, TC-H-061 |
| REQ-F-007 no hardcoded values | All from workflow.Service | TC-S-063 |

### 7.2 Non-functional requirements

| Requirement | Description | Test Cases |
|---|---|---|
| REQ-NF-001 | Summary p95 < 150 ms (10k rows) | TC-S-001 fixture — assert duration < 150 ms |
| REQ-NF-002 | Hierarchy p95 < 300 ms | TC-S-010 fixture — assert duration < 300 ms |
| REQ-NF-003 | File endpoint memory bound via LimitReader | TC-S-034, TC-H-033 |
| REQ-NF-010 | Localhost-only CORS | TC-H-070–074 |
| REQ-NF-011 | File path containment | TC-S-031, TC-S-032, TC-H-032 |
| REQ-NF-012 | Read-only invariant | Static: `grep -n "Insert\|Update\|Delete\|Exec" internal/services/viewer_service.go` must return no matches; code review gate |
| REQ-NF-013 | Key-format validation before DB lookup | TC-H-025, TC-H-034, TC-H-042 (service mock must NOT be called when key is invalid) |
| REQ-NF-020 | otelhttp tracing inherited | Integration smoke (no explicit assertion; route registration test verifies no panic) |
| REQ-NF-021 | Structured logging — no SQL leakage | Service error test TC-S-004: verify error message does NOT contain "SQL" or "sqlite" in user-facing response |

### 7.3 Feature-level acceptance scenarios (spec §2.3)

| Scenario | Description | Test Cases |
|---|---|---|
| Scenario 1 | Summary on empty project → 200, all zeros | TC-S-002, TC-H-002 |
| Scenario 2 | Hierarchy — 1 epic, 2 features, counts correct | TC-S-010, TC-H-010; integration step 2 |
| Scenario 3 | File endpoint — path traversal rejected | TC-S-031, TC-H-032 |
| Scenario 4 | File endpoint — missing file → 200 exists:false | TC-S-033, TC-H-031 |
| Scenario 5 | Recent activity — deleted entity omitted | TC-S-051, TC-R-004 |
| Scenario 6 | Workflow meta — no hardcoded status names | TC-S-063, TC-H-060 |

---

## 8. Quality Gates

The following gates must ALL pass before E27-F02 is approved:

- [ ] `make fmt && make lint` — zero warnings and no formatting diffs
- [ ] `make test` — all tests pass, zero failures
- [ ] Handler package coverage ≥ 85% (`go test -cover ./internal/api/viewer/...`)
- [ ] Service file coverage ≥ 85% (`go test -cover ./internal/services/...` — viewer_service.go specifically)
- [ ] Repository new method: 100% branch coverage on `ListRecentAcrossEntities`
- [ ] No `ViewerService` method calls any write method (static inspection or `go vet` custom check)
- [ ] All 7 integration smoke steps pass with no panics
- [ ] `grep -n "Access-Control-Allow-Origin" internal/api/viewer/handler.go` returns no matches (header set only in cors.go)
- [ ] `grep -rn 'INSERT\|UPDATE\|DELETE' internal/services/viewer_service.go` returns no matches (read-only invariant)
- [ ] `grep -rn '"todo"\|"in_progress"\|"completed"\|"active"' internal/api/viewer/ internal/services/viewer_service.go` returns no matches (no hardcoded status strings)

---

## 9. Test Execution Order

Recommended order for a developer implementing this feature:

1. **Phase A** — Run CORS unit tests (`TC-H-070–074`) after `cors.go` is created. These run immediately with no dependencies.
2. **Phase C** — Run service tests (`TC-S-*`) after `viewer_service.go` is created.
3. **Phase D** — Run handler tests (`TC-H-*`) after `handler.go` is created.
4. **Phase B** — Run repository tests (`TC-R-*`) after `ListRecentAcrossEntities` is added.
5. **Phase E/F** — Run integration smoke after server wiring and route registration is complete.
6. **Full suite** — `make test` before marking feature complete.

---

*Last Updated*: 2026-04-11
