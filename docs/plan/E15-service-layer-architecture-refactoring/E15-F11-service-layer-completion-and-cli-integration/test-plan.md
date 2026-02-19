# Test Plan: E15-F11 Service Layer Completion and CLI Integration

**Feature Key**: E15-F11-service-layer-completion-and-cli-integration
**Date**: 2026-02-19
**QA Agent**: Claude (QA role)
**Status**: Ready for Development

---

## 1. Overview

This test plan covers quality assurance for the E15-F11 feature: eliminating the remaining 13 fat-controller CLI command files and adding the missing service methods they require.

### What Is Being Tested

1. **New `CriteriaService`** — new service with 5 methods, tested with mocked repositories
2. **New methods on `TaskService`** — `GetTaskHistory`, `ListHistory`, `GetAnalytics`, `CreateRelationships`, tested with mocked repositories
3. **New method on `NoteService`** — `GetTaskTimeline`, tested with mocked repositories
4. **New method on `ResumeService`** — `GetTaskResume`, tested with mocked repositories
5. **13 refactored CLI command files** — each must be a thin wrapper (parse → call service → format output), tested with mocked services
6. **Updated global service accessors** — `GetCriteriaService`, updated `GetTaskService`, `GetNoteService`, `GetResumeService` wiring

### Golden Rules (Non-Negotiable)

- **Service tests use MOCKED repositories.** No `test.GetTestDB()` in `internal/services/*_test.go` files.
- **CLI command tests use MOCKED services.** No `test.GetTestDB()` or `repository.New*` in `internal/cli/commands/*_test.go` files for the migrated commands.
- **Only `internal/repository/*_test.go` files use the real database.**
- `make test` must exit 0 with 0 failures before and after every migration.

---

## 2. Test Architecture

### 2.1 Layers Under Test

```
Test Pyramid:
┌─────────────────────────────────────────┐
│        CLI Command Tests                │ ← Mock services. Test: arg parsing,
│        (internal/cli/commands/)         │   output formatting, error propagation
├─────────────────────────────────────────┤
│        Service Tests                    │ ← Mock repositories. Test: business
│        (internal/services/)             │   rules, orchestration, error wrapping
├─────────────────────────────────────────┤
│        Repository Tests (EXISTING)      │ ← Real DB. Test: SQL correctness.
│        (internal/repository/)           │   NOT affected by E15-F11.
└─────────────────────────────────────────┘
```

### 2.2 Mock Patterns to Use

**Service tests** — define mocks in the same `package services` test file:
```go
type MockCriteriaRepository struct {
    CreateFunc         func(ctx context.Context, c *models.TaskCriteria) error
    GetByTaskIDFunc    func(ctx context.Context, taskID int64) ([]*models.TaskCriteria, error)
    UpdateStatusFunc   func(ctx context.Context, id int64, status models.CriteriaStatus, notes *string) error
    DeleteFunc         func(ctx context.Context, id int64) error
    GetSummaryFunc     func(ctx context.Context, taskID int64) (*repository.CriteriaSummary, error)
}
```

**CLI command tests** — mock service interfaces that commands depend on:
```go
type MockCriteriaService struct {
    ImportCriteriaFunc    func(ctx context.Context, taskKey string) (*services.CriteriaImportResult, error)
    ListCriteriaFunc      func(ctx context.Context, taskKey string) (*services.CriteriaListResult, error)
    CheckCriterionFunc    func(ctx context.Context, taskKey string, id int64, note string) (*models.TaskCriteria, error)
    FailCriterionFunc     func(ctx context.Context, taskKey string, id int64, note string) (*models.TaskCriteria, error)
    GetFeatureCriteriaFunc func(ctx context.Context, featureKey string, byTask bool) (*services.FeatureCriteriaResult, error)
}
```

---

## 3. Service Tests

### 3.1 CriteriaService Tests

**File**: `internal/services/criteria_service_test.go`
**Package**: `package services`
**Dependencies to mock**: `CriteriaRepository`, `CriteriaTaskRepository`, `CriteriaFeatureRepository`, `CriteriaTaskListRepository`

#### TC-CS-001: ImportCriteria — Task Not Found
```
Given: taskRepo.GetByKey returns NotFoundError
When:  ImportCriteria(ctx, "E07-F01-999")
Then:  Returns error containing "not found"
And:   criteriaRepo.Create is never called
```

#### TC-CS-002: ImportCriteria — Task Has No File Path
```
Given: taskRepo.GetByKey returns task with empty FilePath
When:  ImportCriteria(ctx, "E07-F01-001")
Then:  Returns error indicating no file path
And:   criteriaRepo.Create is never called
```

#### TC-CS-003: ImportCriteria — Happy Path
```
Given: taskRepo.GetByKey returns task with valid FilePath
  And: taskfile.ParseCriteriaFromFile returns 3 criteria items
  And: criteriaRepo.Create returns no error for each
When:  ImportCriteria(ctx, "E07-F01-001")
Then:  Returns CriteriaImportResult.Imported == 3
And:   criteriaRepo.Create called exactly 3 times
```

#### TC-CS-004: ImportCriteria — File Parse Failure
```
Given: taskRepo.GetByKey returns task with FilePath
  And: taskfile.ParseCriteriaFromFile returns error
When:  ImportCriteria(ctx, "E07-F01-001")
Then:  Returns wrapped error containing parse failure reason
```

#### TC-CS-005: ListCriteria — Happy Path
```
Given: taskRepo.GetByKey returns valid task (ID=42)
  And: criteriaRepo.GetByTaskID(42) returns 4 criteria
  And: criteriaRepo.GetSummaryByTaskID(42) returns summary counts
When:  ListCriteria(ctx, "E07-F01-001")
Then:  Returns CriteriaListResult.Criteria with 4 items
And:   Returns CriteriaListResult.TaskKey == "E07-F01-001"
And:   Summary counts are populated
```

#### TC-CS-006: ListCriteria — Task Not Found
```
Given: taskRepo.GetByKey returns NotFoundError
When:  ListCriteria(ctx, "E07-F01-999")
Then:  Returns wrapped NotFoundError
```

#### TC-CS-007: CheckCriterion — Happy Path
```
Given: taskRepo.GetByKey returns task (ID=42)
  And: criteriaRepo.GetByID returns criterion with TaskID=42
  And: criteriaRepo.UpdateStatus returns no error
When:  CheckCriterion(ctx, "E07-F01-001", criterionID=7, note="verified")
Then:  Returns updated criterion
And:   criteriaRepo.UpdateStatus called with status="complete"
And:   Note "verified" passed to UpdateStatus
```

#### TC-CS-008: CheckCriterion — Criterion Belongs to Different Task
```
Given: taskRepo.GetByKey returns task (ID=42)
  And: criteriaRepo.GetByID returns criterion with TaskID=99 (mismatch)
When:  CheckCriterion(ctx, "E07-F01-001", criterionID=7, note="")
Then:  Returns error indicating criterion does not belong to task
And:   criteriaRepo.UpdateStatus is never called
```

#### TC-CS-009: FailCriterion — Note Required
```
Given: taskRepo.GetByKey returns valid task
  And: criteriaRepo.GetByID returns criterion with matching TaskID
When:  FailCriterion(ctx, "E07-F01-001", criterionID=7, note="")
Then:  Returns error indicating a failure note is required
And:   criteriaRepo.UpdateStatus is never called
```

#### TC-CS-010: FailCriterion — Happy Path
```
Given: taskRepo.GetByKey returns task (ID=42)
  And: criteriaRepo.GetByID returns criterion with TaskID=42
  And: criteriaRepo.UpdateStatus returns no error
When:  FailCriterion(ctx, "E07-F01-001", criterionID=7, note="Assertion failed")
Then:  Returns updated criterion
And:   criteriaRepo.UpdateStatus called with status="failed"
And:   Note "Assertion failed" passed to UpdateStatus
```

#### TC-CS-011: GetFeatureCriteria — Aggregates Correctly
```
Given: featureRepo.GetByKey returns feature (ID=10, Title="Auth")
  And: tasklistRepo.ListByFeature(10) returns 3 tasks
  And: criteriaRepo.GetByTaskID for each task returns:
       task-001: 3 criteria (2 complete, 1 pending)
       task-002: 2 criteria (1 complete, 1 failed)
       task-003: 0 criteria
When:  GetFeatureCriteria(ctx, "E07-F01", byTask=false)
Then:  Returns FeatureCriteriaResult.TotalCount == 5
And:   CompleteCount == 3, PendingCount == 1, FailedCount == 1
And:   CompletionPct == 60.0
And:   TaskSummaries is empty (byTask=false)
```

#### TC-CS-012: GetFeatureCriteria — With Task Breakdown
```
Given: Same setup as TC-CS-011
When:  GetFeatureCriteria(ctx, "E07-F01", byTask=true)
Then:  Returns FeatureCriteriaResult.TaskSummaries with 3 items
And:   Each item has correct per-task counts
And:   Task with 0 criteria still appears in TaskSummaries
```

#### TC-CS-013: GetFeatureCriteria — Feature Not Found
```
Given: featureRepo.GetByKey returns NotFoundError
When:  GetFeatureCriteria(ctx, "E07-F99", byTask=false)
Then:  Returns wrapped NotFoundError
```

---

### 3.2 TaskService New Method Tests

**File**: `internal/services/task_service_test.go` (additions)
**Existing mocks**: `MockTaskRepository` already defined — extend as needed.

#### TC-TS-001: GetTaskHistory — Happy Path
```
Given: taskRepo.GetByKey returns task (Key="E07-F01-001")
  And: historyRepo.GetHistoryByTaskKey returns 3 history records
When:  GetTaskHistory(ctx, "E07-F01-001")
Then:  Returns slice of 3 history records
And:   Records are in the order returned by historyRepo
```

#### TC-TS-002: GetTaskHistory — Task Not Found
```
Given: taskRepo.GetByKey returns NotFoundError
When:  GetTaskHistory(ctx, "E07-F01-999")
Then:  Returns wrapped NotFoundError
And:   historyRepo is never called
```

#### TC-TS-003: GetTaskHistory — historyRepo Not Set
```
Given: TaskService created without SetHistoryRepo being called
When:  GetTaskHistory(ctx, "E07-F01-001")
Then:  Returns empty slice (no error)
  — graceful degradation when historyRepo is nil
```

#### TC-TS-004: ListHistory — With Epic Filter
```
Given: epicRepo.GetByKey("E07") returns epic (ID=7)
  And: historyRepo.ListWithFilters(filters{EpicID=7}) returns 10 records
When:  ListHistory(ctx, HistoryFilters{EpicKey:"E07", Limit:10})
Then:  Returns 10 history records
And:   historyRepo called with EpicID=7 resolved from key
```

#### TC-TS-005: ListHistory — Epic Key Not Found
```
Given: epicRepo.GetByKey("E99") returns NotFoundError
When:  ListHistory(ctx, HistoryFilters{EpicKey:"E99"})
Then:  Returns wrapped NotFoundError
And:   historyRepo.ListWithFilters never called
```

#### TC-TS-006: ListHistory — No Filters (Empty HistoryFilters)
```
Given: historyRepo.ListWithFilters(filters{}) returns 50 records
When:  ListHistory(ctx, HistoryFilters{})
Then:  Returns 50 records (no filter applied)
```

#### TC-TS-007: GetAnalytics — Requires At Least One Analysis Type
```
Given: AnalyticsInput{EpicKey:"E07", SessionDuration:false, PauseFrequency:false}
When:  GetAnalytics(ctx, input)
Then:  Returns error indicating at least one analysis type must be selected
```

#### TC-TS-008: GetAnalytics — Requires Scope Key
```
Given: AnalyticsInput{EpicKey:"", FeatureKey:"", SessionDuration:true}
When:  GetAnalytics(ctx, input)
Then:  Returns error indicating epic or feature key is required
```

#### TC-TS-009: GetAnalytics — By Epic, Session Duration
```
Given: epicRepo.GetByKey("E07") returns epic (ID=7)
  And: workSessionRepo.GetSessionAnalyticsByEpic(7, agentType) returns analytics
When:  GetAnalytics(ctx, AnalyticsInput{EpicKey:"E07", SessionDuration:true})
Then:  Returns AnalyticsResult.Scope == "epic"
And:   AnalyticsResult.ScopeKey == "E07"
And:   SessionAnalysis is populated
And:   PauseAnalysis is nil (not requested)
```

#### TC-TS-010: GetAnalytics — By Feature, Pause Frequency
```
Given: featureRepo.GetByKey("E07-F01") returns feature (ID=1)
  And: workSessionRepo.GetSessionAnalyticsByFeature(1, "") returns analytics
When:  GetAnalytics(ctx, AnalyticsInput{FeatureKey:"E07-F01", PauseFrequency:true})
Then:  Returns AnalyticsResult.Scope == "feature"
And:   PauseAnalysis is populated
And:   SessionAnalysis is nil (not requested)
```

#### TC-TS-011: CreateRelationships — Multiple Types
```
Given: taskRepo.GetByKey("E07-F01-001") returns source task
  And: taskRepo.GetByKey("E07-F01-002") returns target task 1
  And: taskRepo.GetByKey("E07-F01-003") returns target task 2
  And: relationship creation succeeds for all
When:  CreateRelationships(ctx, "E07-F01-001", map{
         "depends_on": ["E07-F01-002"],
         "blocks":     ["E07-F01-003"],
       })
Then:  Returns 2 RelationshipWithTask results
And:   Repository called twice for relationship creation
```

#### TC-TS-012: CreateRelationships — Unknown Target Task
```
Given: taskRepo.GetByKey("E07-F01-001") returns source task
  And: taskRepo.GetByKey("E07-F01-999") returns NotFoundError
When:  CreateRelationships(ctx, "E07-F01-001", map{"depends_on": ["E07-F01-999"]})
Then:  Returns error indicating target task not found
```

---

### 3.3 NoteService New Method Tests

**File**: `internal/services/note_service_test.go` (additions)
**New mock to add**: `mockNoteTimelineHistoryRepo`

#### TC-NS-001: GetTaskTimeline — Happy Path, Sorted Chronologically
```
Given: noteRepo.GetByEntity returns 2 notes (at T+10m, T+30m)
  And: historyRepo.GetHistoryByTaskKey returns 2 history records (at T+5m, T+20m)
  And: historyRepo.GetRejectionHistoryForTask returns 1 rejection (at T+15m)
When:  GetTaskTimeline(ctx, "E07-F01-001")
Then:  Returns NoteTimeline.Events with 5 items (plus creation event = 6)
And:   Events are sorted ascending by timestamp
And:   Order: creation, history@T+5m, note@T+10m, rejection@T+15m, history@T+20m, note@T+30m
```

#### TC-NS-002: GetTaskTimeline — Task Not Found
```
Given: taskRepo.GetByKey returns NotFoundError
When:  GetTaskTimeline(ctx, "E07-F01-999")
Then:  Returns wrapped NotFoundError
```

#### TC-NS-003: GetTaskTimeline — historyRepo Not Set (Graceful Degradation)
```
Given: NoteService created without SetHistoryRepo
  And: noteRepo.GetByEntity returns 3 notes
When:  GetTaskTimeline(ctx, "E07-F01-001")
Then:  Returns NoteTimeline with only note events (plus creation event)
And:   No error returned
```

#### TC-NS-004: GetTaskTimeline — No Notes, No History
```
Given: taskRepo.GetByKey returns valid task
  And: noteRepo.GetByEntity returns empty slice
  And: historyRepo.GetHistoryByTaskKey returns empty slice
  And: historyRepo.GetRejectionHistoryForTask returns empty slice
When:  GetTaskTimeline(ctx, "E07-F01-001")
Then:  Returns NoteTimeline.Events with 1 item (creation event only)
And:   TaskKey == "E07-F01-001"
```

#### TC-NS-005: GetTaskTimeline — Events From Correct Task Only
```
Given: taskRepo.GetByKey returns task (ID=42)
When:  GetTaskTimeline(ctx, "E07-F01-001")
Then:  historyRepo.GetHistoryByTaskKey called with "E07-F01-001"
And:   historyRepo.GetRejectionHistoryForTask called with taskID=42
And:   noteRepo.GetByEntity called with (EntityTypeTask, 42)
```

---

### 3.4 ResumeService New Method Tests

**File**: `internal/services/resume_service_test.go` (create if missing)
**New mocks**: `MockResumeTaskRepository` (extended), `MockResumeWorkSessionRepo`

#### TC-RS-001: GetTaskResume — Happy Path with Work Sessions
```
Given: taskRepo.GetByKey("E07-F01-001") returns task (ID=42)
  And: taskRepo.GetContextData(42) returns context JSON string
  And: noteRepo.GetByEntity returns 3 notes
  And: workSessionRepo.GetByTaskID(42) returns 2 sessions
  And: workSessionRepo.GetSessionStatsByTaskID(42) returns stats
  And: workSessionRepo.GetActiveSessionByTaskID(42) returns nil (no active)
When:  GetTaskResume(ctx, "E07-F01-001")
Then:  Returns TaskResumeContext.Task != nil
And:   ContextData is populated
And:   Notes has 3 items
And:   WorkSessions has 2 items
And:   ActiveSession is nil
```

#### TC-RS-002: GetTaskResume — Task Not Found
```
Given: taskRepo.GetByKey returns NotFoundError
When:  GetTaskResume(ctx, "E07-F01-999")
Then:  Returns wrapped NotFoundError
```

#### TC-RS-003: GetTaskResume — workSessionRepo Not Set (Graceful Degradation)
```
Given: taskRepo.GetByKey returns valid task
  And: ResumeService created without SetWorkSessionRepo
When:  GetTaskResume(ctx, "E07-F01-001")
Then:  Returns TaskResumeContext with no WorkSessions (nil or empty)
And:   No error returned
And:   Task and Notes are still populated
```

#### TC-RS-004: GetTaskResume — Active Session Present
```
Given: taskRepo.GetByKey returns task (ID=42)
  And: workSessionRepo.GetActiveSessionByTaskID(42) returns a session
When:  GetTaskResume(ctx, "E07-F01-001")
Then:  Returns TaskResumeContext.ActiveSession != nil
```

#### TC-RS-005: GetTaskResume — No Context Data
```
Given: taskRepo.GetByKey returns task
  And: taskRepo.GetContextData returns nil (no context set)
When:  GetTaskResume(ctx, "E07-F01-001")
Then:  Returns TaskResumeContext.ContextData == nil
And:   No error
```

---

## 4. CLI Command Tests

Each migrated command file must have a corresponding test that:
1. Does NOT call `test.GetTestDB()`, `db.InitDB()`, or `repository.New*`
2. Mocks the service the command calls
3. Tests at minimum: happy path, not-found error, and invalid-input error

### 4.1 task_context.go Tests

**File**: `internal/cli/commands/task_context_test.go` (update existing or create)
**Service to mock**: `MockContextService`

#### TC-CLI-CTX-001: task context get — Happy Path
```
Given: contextSvc.GetContext returns context data for key "E07-F01-001"
When:  runTaskContextGet(cmd, ["E07-F01-001"])
Then:  No error returned
And:   Context data printed (human or JSON)
And:   contextSvc.GetContext called with (EntityTypeTask, "E07-F01-001")
```

#### TC-CLI-CTX-002: task context set — Happy Path
```
Given: contextSvc.SetContextField returns no error
When:  runTaskContextSet(cmd, ["E07-F01-001"]) with --field=current_step --value="Testing"
Then:  No error returned
And:   contextSvc.SetContextField called with (EntityTypeTask, "E07-F01-001", "current_step", "Testing")
```

#### TC-CLI-CTX-003: task context clear — Task Not Found
```
Given: contextSvc.ClearContext returns NotFoundError
When:  runTaskContextClear(cmd, ["E07-F01-999"])
Then:  Error propagated from service
```

#### TC-CLI-CTX-004: task context set — Missing Required Flags
```
Given: No --field flag provided
When:  runTaskContextSet(cmd, ["E07-F01-001"])
Then:  Error returned indicating --field is required
And:   contextSvc.SetContextField never called
```

---

### 4.2 notes_search.go Tests

**File**: `internal/cli/commands/notes_search_filter_test.go` (update) and/or new test
**Service to mock**: `MockNoteService`

#### TC-CLI-NS-001: notes search — Basic Query
```
Given: noteSvc.SearchNotes returns 3 matching notes
When:  runNotesSearch(cmd, []) with --query="authentication"
Then:  No error returned
And:   noteSvc.SearchNotes called with query="authentication"
And:   3 results formatted for output
```

#### TC-CLI-NS-002: notes search — Entity Type Filter
```
Given: noteSvc.SearchNotes returns results
When:  runNotesSearch(cmd, []) with --query="test" --entity-type="task"
Then:  noteSvc.SearchNotes called with entityType=EntityTypeTask
```

#### TC-CLI-NS-003: notes search — Epic Scope Filter
```
Given: noteSvc.SearchNotes returns results
When:  runNotesSearch(cmd, []) with --query="test" --epic="E07"
Then:  noteSvc.SearchNotes called with epicKey="E07"
```

#### TC-CLI-NS-004: notes search — Empty Query Returns Error
```
Given: No --query flag provided
When:  runNotesSearch(cmd, [])
Then:  Error returned indicating query is required
And:   noteSvc.SearchNotes never called
```

---

### 4.3 task_link.go Tests

**File**: `internal/cli/commands/task_relationship_commands_test.go` (update)
**Service to mock**: `MockTaskService`

#### TC-CLI-TL-001: task link — Creates Relationship
```
Given: taskSvc.CreateRelationships returns 1 relationship
When:  runTaskLink(cmd, ["E07-F01-001"]) with --depends-on="E07-F01-002"
Then:  No error
And:   taskSvc.CreateRelationships called with source="E07-F01-001", type="depends_on", target="E07-F01-002"
```

#### TC-CLI-TL-002: task link — Target Not Found
```
Given: taskSvc.CreateRelationships returns NotFoundError for target
When:  runTaskLink(cmd, ["E07-F01-001"]) with --depends-on="E07-F01-999"
Then:  Error propagated from service
```

---

### 4.4 task_history.go Tests

**File**: New or updated `internal/cli/commands/task_history_test.go`
**Service to mock**: `MockTaskService`

#### TC-CLI-TH-001: task history — Displays Records
```
Given: taskSvc.GetTaskHistory returns 5 history records
When:  runTaskHistory(cmd, ["E07-F01-001"])
Then:  No error
And:   taskSvc.GetTaskHistory called with "E07-F01-001"
And:   5 records formatted for output
```

#### TC-CLI-TH-002: task history — JSON Output
```
Given: taskSvc.GetTaskHistory returns 3 history records
  And: GlobalConfig.JSON == true
When:  runTaskHistory(cmd, ["E07-F01-001"])
Then:  Output is valid JSON array
And:   Contains 3 entries
```

#### TC-CLI-TH-003: task history — Task Not Found
```
Given: taskSvc.GetTaskHistory returns NotFoundError
When:  runTaskHistory(cmd, ["E07-F01-999"])
Then:  Error propagated from service
```

#### TC-CLI-TH-004: task history — Missing Task Key Argument
```
Given: No task key argument provided
When:  runTaskHistory(cmd, [])
Then:  Command argument validation error returned
```

---

### 4.5 history.go Tests

**File**: New `internal/cli/commands/history_command_test.go`
**Service to mock**: `MockTaskService`

#### TC-CLI-H-001: history — Project Wide, No Filters
```
Given: taskSvc.ListHistory(HistoryFilters{}) returns 20 records
When:  runHistory(cmd, [])
Then:  No error
And:   taskSvc.ListHistory called with empty filters
And:   20 records formatted for output
```

#### TC-CLI-H-002: history — Epic Filter via Positional Arg
```
Given: taskSvc.ListHistory returns records
When:  runHistory(cmd, ["E07"])
Then:  taskSvc.ListHistory called with HistoryFilters{EpicKey:"E07"}
```

#### TC-CLI-H-003: history — Agent Filter via Flag
```
Given: taskSvc.ListHistory returns records
When:  runHistory(cmd, []) with --agent="backend"
Then:  taskSvc.ListHistory called with HistoryFilters{AgentID:"backend"}
```

#### TC-CLI-H-004: history — CSV Output
```
Given: taskSvc.ListHistory returns 3 records
  And: --format=csv flag set
When:  runHistory(cmd, [])
Then:  Output is CSV formatted
And:   3 data rows present
```

---

### 4.6 task_resume.go Tests

**File**: New `internal/cli/commands/task_resume_command_test.go`
**Service to mock**: `MockResumeService`

#### TC-CLI-TR-001: task resume — Displays Full Context
```
Given: resumeSvc.GetTaskResume returns TaskResumeContext with task, notes, sessions
When:  runTaskResume(cmd, ["E07-F01-001"])
Then:  No error
And:   resumeSvc.GetTaskResume called with "E07-F01-001"
And:   Output includes task key and title
```

#### TC-CLI-TR-002: task resume — JSON Output
```
Given: resumeSvc.GetTaskResume returns TaskResumeContext
  And: GlobalConfig.JSON == true
When:  runTaskResume(cmd, ["E07-F01-001"])
Then:  Output is valid JSON object containing task key
```

#### TC-CLI-TR-003: task resume — Task Not Found
```
Given: resumeSvc.GetTaskResume returns NotFoundError
When:  runTaskResume(cmd, ["E07-F01-999"])
Then:  Error propagated from service
```

---

### 4.7 analytics.go Tests

**File**: New `internal/cli/commands/analytics_command_test.go`
**Service to mock**: `MockTaskService`

#### TC-CLI-AN-001: analytics — By Epic, Session Duration
```
Given: taskSvc.GetAnalytics returns AnalyticsResult for epic scope
When:  runAnalytics(cmd, []) with --epic="E07" --session-duration
Then:  No error
And:   taskSvc.GetAnalytics called with AnalyticsInput{EpicKey:"E07", SessionDuration:true}
And:   Output includes session duration metrics
```

#### TC-CLI-AN-002: analytics — Missing Scope Flag
```
Given: No --epic or --feature flag provided
When:  runAnalytics(cmd, []) with --session-duration
Then:  Error returned indicating epic or feature scope is required
And:   taskSvc.GetAnalytics never called
```

#### TC-CLI-AN-003: analytics — Missing Analysis Type Flag
```
Given: --epic="E07" provided
  And: No --session-duration or --pause-frequency flag
When:  runAnalytics(cmd, [])
Then:  Error returned indicating at least one analysis type is required
And:   taskSvc.GetAnalytics never called
```

#### TC-CLI-AN-004: analytics — Feature Not Found
```
Given: taskSvc.GetAnalytics returns NotFoundError
When:  runAnalytics(cmd, []) with --feature="E07-F99" --session-duration
Then:  Error propagated from service
```

---

### 4.8 task_note.go Tests

**File**: `internal/cli/commands/task_note_timeline_test.go` (update existing)
**Service to mock**: `MockNoteService`

#### TC-CLI-TN-001: task notes — Lists Notes
```
Given: noteSvc.ListNotes returns 4 notes for task
When:  runTaskNotes(cmd, ["E07-F01-001"])
Then:  No error
And:   noteSvc.ListNotes called with (EntityTypeTask, "E07-F01-001", noteTypes)
And:   4 notes displayed
```

#### TC-CLI-TN-002: task notes add — Adds Note
```
Given: noteSvc.AddNote returns new note
When:  runTaskNoteAdd(cmd, ["E07-F01-001"]) with --type=testing --content="Tests pass"
Then:  No error
And:   noteSvc.AddNote called with (EntityTypeTask, "E07-F01-001", "testing", "Tests pass")
```

#### TC-CLI-TN-003: task timeline — Displays Chronological Timeline
```
Given: noteSvc.GetTaskTimeline returns NoteTimeline with 6 events
When:  runTaskTimeline(cmd, ["E07-F01-001"])
Then:  No error
And:   noteSvc.GetTaskTimeline called with "E07-F01-001"
And:   6 events displayed in chronological order
```

#### TC-CLI-TN-004: task notes add — Content Required
```
Given: No --content flag provided
When:  runTaskNoteAdd(cmd, ["E07-F01-001"])
Then:  Error returned indicating content is required
And:   noteSvc.AddNote never called
```

---

### 4.9 task_criteria.go Tests

**File**: `internal/cli/commands/task_criteria_test.go` (replace current real-DB test)
**Service to mock**: `MockCriteriaService`

#### TC-CLI-TC-001: task criteria import — Happy Path
```
Given: criteriaSvc.ImportCriteria returns CriteriaImportResult{Imported:5}
When:  runTaskCriteriaImport(cmd, ["E07-F01-001"])
Then:  No error
And:   criteriaSvc.ImportCriteria called with "E07-F01-001"
And:   Success message includes "5" criteria imported
```

#### TC-CLI-TC-002: task criteria list — Shows Summary
```
Given: criteriaSvc.ListCriteria returns CriteriaListResult with 3 criteria and summary
When:  runTaskCriteriaList(cmd, ["E07-F01-001"])
Then:  No error
And:   criteriaSvc.ListCriteria called with "E07-F01-001"
And:   Output includes each criterion and completion summary
```

#### TC-CLI-TC-003: task criteria check — Marks Complete
```
Given: criteriaSvc.CheckCriterion returns updated criterion
When:  runTaskCriteriaCheck(cmd, ["E07-F01-001", "7"]) with --note="Verified"
Then:  No error
And:   criteriaSvc.CheckCriterion called with ("E07-F01-001", 7, "Verified")
```

#### TC-CLI-TC-004: task criteria fail — Marks Failed
```
Given: criteriaSvc.FailCriterion returns updated criterion
When:  runTaskCriteriaFail(cmd, ["E07-F01-001", "7"]) with --note="Broken"
Then:  No error
And:   criteriaSvc.FailCriterion called with ("E07-F01-001", 7, "Broken")
```

#### TC-CLI-TC-005: task criteria import — Task Not Found
```
Given: criteriaSvc.ImportCriteria returns NotFoundError
When:  runTaskCriteriaImport(cmd, ["E07-F01-999"])
Then:  Error propagated from service
```

#### TC-CLI-TC-006: task criteria check — Invalid Criterion ID Format
```
Given: Criterion ID argument is not an integer
When:  runTaskCriteriaCheck(cmd, ["E07-F01-001", "abc"])
Then:  Error returned for invalid ID format
And:   criteriaSvc.CheckCriterion never called
```

---

### 4.10 feature_criteria.go Tests

**File**: New `internal/cli/commands/feature_criteria_test.go`
**Service to mock**: `MockCriteriaService`

#### TC-CLI-FC-001: feature criteria — Shows Aggregated Summary
```
Given: criteriaSvc.GetFeatureCriteria returns FeatureCriteriaResult
When:  runFeatureCriteria(cmd, ["E07-F01"])
Then:  No error
And:   criteriaSvc.GetFeatureCriteria called with ("E07-F01", byTask=false)
And:   Aggregated counts displayed
```

#### TC-CLI-FC-002: feature criteria --by-task — Shows Per-Task Breakdown
```
Given: criteriaSvc.GetFeatureCriteria returns result with TaskSummaries populated
When:  runFeatureCriteria(cmd, ["E07-F01"]) with --by-task flag
Then:  criteriaSvc.GetFeatureCriteria called with ("E07-F01", byTask=true)
And:   Per-task breakdown displayed
```

#### TC-CLI-FC-003: feature criteria — Feature Not Found
```
Given: criteriaSvc.GetFeatureCriteria returns NotFoundError
When:  runFeatureCriteria(cmd, ["E07-F99"])
Then:  Error propagated from service
```

---

### 4.11 task_next_status.go Tests

**File**: `internal/cli/commands/task_next_status_test.go` (new or update)
**Service to mock**: `MockTaskService`

#### TC-CLI-TNS-001: task next-status --preview — Shows Transitions Without Applying
```
Given: taskSvc.GetNextStatus returns list of available transitions
When:  runTaskNextStatus(cmd, ["E07-F01-001"]) with --preview flag
Then:  No error
And:   taskSvc.GetNextStatus called with "E07-F01-001"
And:   taskSvc.TransitionStatus never called
And:   Available transitions displayed
```

#### TC-CLI-TNS-002: task next-status --status — Direct Transition
```
Given: taskSvc.TransitionStatus returns updated task
When:  runTaskNextStatus(cmd, ["E07-F01-001"]) with --status="ready_for_code_review"
Then:  No error
And:   taskSvc.TransitionStatus called with ("E07-F01-001", "ready_for_code_review", opts)
And:   Success message displays new status
```

#### TC-CLI-TNS-003: task next-status --force — Bypasses Validation
```
Given: taskSvc.TransitionStatus with force=true returns updated task
When:  runTaskNextStatus(cmd, ["E07-F01-001"]) with --status="completed" --force
Then:  No error
And:   taskSvc.TransitionStatus called with force=true in opts
```

#### TC-CLI-TNS-004: task next-status --reason — Passes Rejection Reason
```
Given: taskSvc.TransitionStatus returns updated task
When:  runTaskNextStatus(cmd, ["E07-F01-001"]) with --status="changes_requested" --reason="Fix tests"
Then:  taskSvc.TransitionStatus called with reason="Fix tests" in opts
```

#### TC-CLI-TNS-005: task next-status — Task Not Found
```
Given: taskSvc.TransitionStatus returns NotFoundError
When:  runTaskNextStatus(cmd, ["E07-F01-999"]) with --status="in_development"
Then:  Error propagated from service
```

---

### 4.12 view.go Tests

**File**: New `internal/cli/commands/view_command_test.go`
**Services to mock**: `MockTaskService`, `MockEpicService`, `MockFeatureService`

#### TC-CLI-VW-001: view task — Resolves File Path from TaskService
```
Given: taskSvc.GetTask returns task with FilePath="/path/to/task.md"
When:  runView(cmd, ["E07-F01-001"])
Then:  taskSvc.GetTask called with "E07-F01-001"
And:   No repository.New* calls made in command handler
  Note: actual file open/editor invocation is an I/O side effect;
        test verifies path resolution logic only.
```

#### TC-CLI-VW-002: view epic — Resolves File Path from EpicService
```
Given: epicSvc.GetEpic returns epic with FilePath="/path/to/epic.md"
When:  runView(cmd, ["E07"])
Then:  epicSvc.GetEpic called with "E07"
```

#### TC-CLI-VW-003: view — Task Not Found
```
Given: taskSvc.GetTask returns NotFoundError
When:  runView(cmd, ["E07-F01-999"])
Then:  Error propagated from service
```

---

## 5. Architecture Compliance Tests

These are automated shell checks run as part of the acceptance criteria gate. They are not Go tests, but must pass before QA sign-off.

### AC-001: Zero Fat Controllers
```bash
grep -rn "repository\.New\|cli\.GetDB\|repoDb" internal/cli/commands/ \
  --include="*.go" \
  | grep -v "_test\.go" \
  | grep -v "mock_" \
  | grep -v "cloud\.go" \
  | grep -v "init\.go" \
  | grep -v "migrate_backfill_slugs\.go" \
  | grep -v "validate\.go"
# Expected: empty output (0 lines)
```

### AC-002: No Real DB in Migrated CLI Tests
```bash
grep -rn "test\.GetTestDB\|db\.InitDB\|repository\.New" \
  internal/cli/commands/task_criteria_test.go \
  internal/cli/commands/task_history_test.go \
  internal/cli/commands/task_note_timeline_test.go \
  internal/cli/commands/notes_search_filter_test.go \
  internal/cli/commands/notes_search_integration_test.go 2>/dev/null
# Expected: empty output OR results only from test helper setup, not command-level tests
```

### AC-003: Full Test Suite Passes
```bash
make fmt && make lint && make test
# Expected: exit 0, "0 failures"
```

### AC-004: Build Succeeds
```bash
make build
# Expected: exit 0
```

---

## 6. Edge Case Coverage Requirements

The following edge cases from the PRD (EC-001 through EC-007) must have corresponding test coverage:

| Edge Case | ID | Required Test Coverage |
|-----------|-----|------------------------|
| Interactive mode in task_next_status | EC-001 | TC-CLI-TNS-001 (preview), TC-CLI-TNS-002 (direct). Interactive stdin logic stays in CLI; test non-interactive paths via mock. |
| Status cascade side effects | EC-002 | TC-CLI-TNS-002 verifies `TransitionStatus` is called (service owns cascade). Cascade itself tested in existing TaskService tests. |
| Criteria multiple subcommands | EC-003 | TC-CLI-TC-001 through TC-CLI-TC-006 cover all subcommands. |
| Note timeline vs note list | EC-004 | TC-CLI-TN-001 (list), TC-CLI-TN-003 (timeline) are distinct test cases. |
| Analytics scope resolution | EC-005 | TC-TS-009 (epic), TC-TS-010 (feature), TC-CLI-AN-001 through TC-CLI-AN-004. |
| View command path resolution | EC-006 | TC-CLI-VW-001, TC-CLI-VW-002. |
| validate.go infrastructure exception | EC-007 | Document as accepted pattern. No new test required. Verify AC-001 grep excludes validate.go. |

---

## 7. Coverage Goals

| Area | Target | Measurement |
|------|--------|-------------|
| `CriteriaService` business logic | ≥ 85% | `go test ./internal/services/ -cover` |
| New `TaskService` methods | ≥ 80% | Same |
| `NoteService.GetTaskTimeline` | ≥ 85% | Same |
| `ResumeService.GetTaskResume` | ≥ 80% | Same |
| Error paths in all new service methods | 100% | All error branches in §3 have test cases |
| Nil/optional dependency paths | 100% | TC-TS-003, TC-NS-003, TC-RS-003 |
| Migrated CLI command handlers | ≥ 70% | `go test ./internal/cli/commands/ -cover` |

---

## 8. Testing Anti-Patterns — What MUST NOT Appear

The following patterns will cause test plan failure if found in new or updated test code:

```
ANTI-PATTERN 1: Real DB in service tests
internal/services/*_test.go:
   db := test.GetTestDB()        ← FORBIDDEN
   db.InitDB(":memory:")         ← FORBIDDEN

ANTI-PATTERN 2: Real DB in CLI command tests (migrated files)
internal/cli/commands/task_criteria_test.go etc.:
   db := setupTestDB(t)          ← FORBIDDEN (existing pattern to be removed)
   repository.NewTaskRepository  ← FORBIDDEN in command handler tests

ANTI-PATTERN 3: Repository construction in service tests
   repo := repository.NewTaskCriteriaRepository(db) ← FORBIDDEN
   (Use MockCriteriaRepository instead)

ANTI-PATTERN 4: Business logic assertions in CLI tests
   CLI tests must verify that the service was called with correct args,
   NOT verify the business outcome (that belongs in service tests).
```

---

## 9. Test File Checklist

### New files to create:
- [ ] `internal/services/criteria_service_test.go`
- [ ] `internal/services/resume_service_test.go`
- [ ] `internal/cli/commands/analytics_command_test.go`
- [ ] `internal/cli/commands/history_command_test.go`
- [ ] `internal/cli/commands/task_resume_command_test.go`
- [ ] `internal/cli/commands/feature_criteria_test.go`
- [ ] `internal/cli/commands/view_command_test.go`
- [ ] `internal/cli/commands/task_next_status_test.go`

### Existing files to update (remove real-DB patterns, add mock-service tests):
- [ ] `internal/cli/commands/task_criteria_test.go` — replace in-memory DB setup with MockCriteriaService
- [ ] `internal/cli/commands/task_history_test.go` — update to use MockTaskService
- [ ] `internal/cli/commands/task_note_timeline_test.go` — update to use MockNoteService
- [ ] `internal/cli/commands/notes_search_filter_test.go` — update to use MockNoteService
- [ ] `internal/cli/commands/notes_search_integration_test.go` — assess: convert to service mock or move to repository tests

### Additions to existing service test files:
- [ ] `internal/services/task_service_test.go` — add TC-TS-001 through TC-TS-012
- [ ] `internal/services/note_service_test.go` — add TC-NS-001 through TC-NS-005

---

## 10. QA Sign-Off Checklist

Before the feature can advance to `ready_for_approval`, ALL of the following must be true:

- [ ] `make test` exits 0, 0 failures, no test panics
- [ ] `make lint` exits 0, no new violations
- [ ] `make fmt` requires 0 changes
- [ ] `make build` exits 0
- [ ] AC-001 grep check returns empty output
- [ ] All service test files in scope use only mocked repositories
- [ ] All migrated CLI command test files use mocked services (not real DB)
- [ ] All test cases in §3 (Service Tests) are implemented and passing
- [ ] All test cases in §4 (CLI Command Tests) are implemented and passing
- [ ] Edge cases EC-001 through EC-007 are addressed per §6
- [ ] Coverage goals in §7 are met (run `make test-coverage`, verify report)
- [ ] No test cases are stubbed with `t.Skip()` unless explicitly justified

---

## 11. Defect Severity Classifications

| Severity | Description | Examples for This Feature |
|----------|-------------|---------------------------|
| Critical | Blocks testing or causes data loss | Service test uses real DB; command crashes at startup |
| High | Key functionality broken with no workaround | CriteriaService.CheckCriterion doesn't validate task ownership; cascade not triggered after TransitionStatus |
| Medium | Partial functionality broken, workaround exists | Analytics command returns wrong scope label; JSON output missing field |
| Low | Cosmetic or non-blocking | Error message wording inconsistent; minor formatting differences |

Any Critical or High severity defect blocks advancement to the next workflow status.

---

*Test Plan created*: 2026-02-19
*QA Agent*: Claude (QA role, E15-F11)
*Version*: 1.0
