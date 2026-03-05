# E18-F02: Bug Entity Core -- Test Plan

**Feature**: Bug Entity Core (Model, Repository, Service)
**Complexity**: STANDARD
**Date**: 2026-03-03
**Author**: QA Agent

---

## 1. Acceptance Criteria Test Matrix

Each acceptance criterion from the feature PRD is mapped to specific test cases with expected behavior.

### Story 1: Bug Creation (Programmatic)

| AC | Test Case | Input | Expected Outcome | Priority |
|----|-----------|-------|-------------------|----------|
| AC-1.1 | CreateBug returns persisted bug with auto-key | `CreateBugInput{Title: "Login fails"}` | Bug returned with key `B001`, status `reported`, severity `medium`, valid `CreatedAt` | P1 |
| AC-1.2 | Default severity is medium | `CreateBugInput{Title: "Test"}` (no severity) | `bug.Severity == "medium"` | P1 |
| AC-1.3 | Markdown file created | Create bug B001 | File exists at `docs/bugs/B001.md` with frontmatter (key, title, status, severity) and body sections | P1 |
| AC-1.4 | Atomic rollback on file failure | Simulate `fileops.WriteEntityFile` failure | No bug record in database after error | P1 |
| AC-1.5 | Sequential key generation | Create 3 bugs | Keys are B001, B002, B003 regardless of any prior deletions | P1 |

### Story 2: Bug Creation with Severity and Link

| AC | Test Case | Input | Expected Outcome | Priority |
|----|-----------|-------|-------------------|----------|
| AC-2.1 | Create with severity and valid link | `CreateBugInput{Title: "Crash", Severity: "critical", LinkedEntityKey: "E07-F01"}` (feature exists) | Bug created with severity `critical`, linked_entity_type `feature`, linked_entity_key `E07-F01` | P1 |
| AC-2.2 | Invalid link rejected | `LinkedEntityKey: "E07-F99"` (feature does not exist) | Error containing `"linked entity not found: E07-F99"`, no bug created | P1 |
| AC-2.3 | Entity type auto-detection | Links: `E07` (epic), `E07-F01` (feature), `E07-F01-001` (task) | Correct `LinkedEntityType` set for each | P1 |
| AC-2.4 | Invalid severity rejected | `Severity: "urgent"` | Error: `"invalid severity 'urgent': must be one of critical, high, medium, low"` | P1 |
| AC-2.5 | Linked entity visible in GetBug | Create linked bug, then GetBug | Returned bug contains `LinkedEntityKey` and `LinkedEntityType` | P2 |

### Story 3: Triage Operation

| AC | Test Case | Input | Expected Outcome | Priority |
|----|-----------|-------|-------------------|----------|
| AC-3.1 | Triage happy path | `TriageBugInput{Key: "B001", Severity: "high", AssignedTo: "developer"}` on `reported` bug | Severity updated to `high`, status advanced to `triaged`, `assigned_to` context field set | P1 |
| AC-3.2 | Triage on non-reported bug | Triage a bug with status `triaged` | Error: `"cannot triage bug B001: current status is 'triaged', must be 'reported'"` | P1 |
| AC-3.3 | Triage atomicity | Simulate status advance failure after severity update | All changes rolled back, severity unchanged | P1 |
| AC-3.4 | Triage records status history | Triage B001 | Status history entry exists for reported -> triaged | P2 |

### Story 4: Workflow Status Transitions

| AC | Test Case | Input | Expected Outcome | Priority |
|----|-----------|-------|-------------------|----------|
| AC-4.1 | Full workflow traversal | Advance through reported -> triaged -> in_fix -> in_verification -> resolved | Each transition succeeds | P1 |
| AC-4.2 | Set terminal status (wont_fix) | `SetBugStatus(ctx, "B001", "wont_fix")` from `triaged` | Status set to `wont_fix` | P1 |
| AC-4.3 | Set terminal status (duplicate) | `SetBugStatus(ctx, "B001", "duplicate")` from appropriate status | Status set to `duplicate` | P1 |
| AC-4.4 | Advance from terminal status | AdvanceBugStatus on `resolved` bug | Error: `"cannot advance bug B001: status 'resolved' is terminal"` | P1 |
| AC-4.5 | Invalid transition rejected | `SetBugStatus(ctx, "B001", "resolved")` from `reported` | Workflow error returned | P1 |

### Story 5: List and Filter Bugs

| AC | Test Case | Input | Expected Outcome | Priority |
|----|-----------|-------|-------------------|----------|
| AC-5.1 | List all bugs | `BugFilters{}` | All bugs returned, sorted by creation date descending | P1 |
| AC-5.2 | Filter by status | `BugFilters{Status: "reported"}` | Only `reported` bugs returned | P1 |
| AC-5.3 | Filter by severity | `BugFilters{Severity: "critical"}` | Only critical bugs returned | P1 |
| AC-5.4 | Filter by linked entity | `BugFilters{LinkedEntityKey: "E07-F01"}` | Only bugs linked to E07-F01 | P2 |
| AC-5.5 | Combined filters (AND logic) | `BugFilters{Status: "reported", Severity: "critical"}` | Only reported AND critical bugs | P1 |
| AC-5.6 | Empty result returns empty slice | Filter with no matches | `len(result) == 0`, result is not nil | P2 |

### Story 6: CRUD Operations

| AC | Test Case | Input | Expected Outcome | Priority |
|----|-----------|-------|-------------------|----------|
| AC-6.1 | GetBug returns full record | GetBug("B001") on existing bug | All fields populated | P1 |
| AC-6.2 | GetBug not found | GetBug("B999") | `NotFoundError{Entity: "bug", Key: "B999"}` | P1 |
| AC-6.3 | UpdateBug partial update | `BugUpdates{Title: ptr("New title")}` | Only title changed, other fields unchanged | P1 |
| AC-6.4 | UpdateBug cannot change key | Attempt to change key | Key remains original value | P2 |
| AC-6.5 | DeleteBug existing | DeleteBug("B001") | Bug removed from database | P1 |
| AC-6.6 | DeleteBug not found | DeleteBug("B999") | `NotFoundError` returned | P2 |

---

## 2. Component Test Strategy

### 2.1 Bug Model Tests (`internal/models/bug_test.go`)

**Test approach**: Pure unit tests. No database, no mocks. Tests validate structural correctness of the Bug model and key format.

**Tests**:

| Test Name | What It Validates | Type |
|-----------|-------------------|------|
| TestValidateBugKey_ValidKeys | Accepts B001, B042, B999 | Table-driven |
| TestValidateBugKey_InvalidKeys | Rejects B0001, B01, b001, bug001, empty string, B000-style edge cases | Table-driven |
| TestValidateBugKey_ErrorMessages | Error messages include invalid key and expected format | Assertion |
| TestBug_Validate_ValidBug | Valid bug passes validation | Single case |
| TestBug_Validate_EmptyTitle | Returns error for empty/whitespace title | Table-driven |
| TestBug_Validate_InvalidKey | Returns error for malformed key | Table-driven |
| TestBug_Validate_EmptyStatus | Returns error for empty status | Single case |
| TestBug_Validate_InvalidSeverity | Returns error for severity not in {critical, high, medium, low} | Table-driven |
| TestBug_Validate_ValidSeverities | All 4 severity constants pass validation | Loop |
| TestBug_Validate_DoesNotCheckWorkflowStatus | Validate() accepts any non-empty status string (workflow checking is service-layer concern) | Assertion |

**Coverage target**: 100% of validation paths.

### 2.2 Bug Repository Tests (`internal/repository/bug_repository_test.go`)

**Test approach**: Real database with cleanup. Use B900-B999 key range to avoid collision with production data. Clean before each test.

**Setup pattern**:
```
1. Get test DB via test.GetTestDB()
2. DELETE FROM bugs WHERE key LIKE 'B9%' (cleanup prefix)
3. Seed test data as needed
4. Run test
5. Cleanup via defer
```

**Tests**:

| Test Name | What It Validates | Type |
|-----------|-------------------|------|
| TestBugRepository_Create_And_GetByKey | Create + read round-trip, all fields persisted correctly | Single case |
| TestBugRepository_GetByKey_CaseInsensitive | `GetByKey("b001")` finds `B001` | Single case |
| TestBugRepository_GetByKey_NotFound | Returns NotFoundError for non-existent key | Single case |
| TestBugRepository_GetNextKey_EmptyTable | Returns "B001" when no bugs exist | Single case |
| TestBugRepository_GetNextKey_Increment | Returns B003 when B001 and B002 exist | Single case |
| TestBugRepository_GetNextKey_AfterDeletion | Returns B003 (not B002) after B002 is deleted | Single case |
| TestBugRepository_Update | Updates title, severity, description; key and status unchanged | Single case |
| TestBugRepository_UpdateStatus | Updates status and updated_at only | Single case |
| TestBugRepository_Delete_Existing | Deletes bug, subsequent GetByKey returns NotFoundError | Single case |
| TestBugRepository_Delete_NotFound | Returns NotFoundError for non-existent key | Single case |
| TestBugRepository_List_NoFilters | Returns all bugs, ordered by created_at DESC | Single case |
| TestBugRepository_List_StatusFilter | Filters to matching status only | Table-driven |
| TestBugRepository_List_SeverityFilter | Filters to matching severity only | Table-driven |
| TestBugRepository_List_LinkedEntityFilter | Filters to bugs linked to specific entity | Single case |
| TestBugRepository_List_CombinedFilters | AND logic for status + severity | Single case |
| TestBugRepository_List_EmptyResult | Returns empty slice (not nil) | Single case |
| TestBugRepository_CountByStatus | Returns correct status -> count map | Single case |
| TestBugRepository_CountBySeverity | Returns correct severity -> count map | Single case |
| TestBugRepository_Create_UniqueKeyConstraint | Duplicate key insert fails | Single case |

**Coverage target**: All CRUD operations, all filter combinations, key generation edge cases.

**Prerequisite**: The `bugs` table must exist (F01 schema migration). If F01 is not yet implemented when tests are written, tests should be skipped with `t.Skip("requires F01 bugs table")`.

### 2.3 Bug Service Tests (`internal/services/bug_service_test.go`)

**Test approach**: Mocked repositories. No real database. All repository interfaces are implemented as function-field mocks. Table-driven tests used where multiple scenarios share setup.

**Mock definitions** (in `internal/services/bug_service_test.go` or shared `mocks_test.go`):

- `MockBugRepository` -- function fields for all BugRepository interface methods
- `MockWorkflowService` -- function fields for `GetDefaultStatus`, `GetNextStatus`, `ValidateTransition`, `IsTerminalStatus`
- `MockLinkValidatorEpicRepo` -- function field for `GetByKey`
- `MockLinkValidatorFeatureRepo` -- function field for `GetByKey`
- `MockLinkValidatorTaskRepo` -- function field for `GetByKey`

**Tests**:

| Test Name | What It Validates | Type |
|-----------|-------------------|------|
| TestBugService_CreateBug_HappyPath | Auto-key, default severity, status from workflow, repo.Create called | Single case |
| TestBugService_CreateBug_WithSeverityAndLink | Severity set, link validated, entity type detected | Single case |
| TestBugService_CreateBug_InvalidLink | Returns error when linked entity not found | Table-driven (epic/feature/task) |
| TestBugService_CreateBug_UnrecognizedKeyFormat | Returns error for unrecognized linked entity key format | Single case |
| TestBugService_CreateBug_InvalidSeverity | Returns validation error before any DB call | Single case |
| TestBugService_CreateBug_EmptyTitle | Returns error, no repo.Create called | Single case |
| TestBugService_CreateBug_FileFailureRollback | DB changes rolled back when file creation fails | Single case |
| TestBugService_GetBug_HappyPath | Returns bug from repository | Single case |
| TestBugService_GetBug_NotFound | Propagates NotFoundError | Single case |
| TestBugService_UpdateBug_PartialUpdate | Only non-nil fields applied, key immutable | Single case |
| TestBugService_UpdateBug_ValidationFailure | Returns error if updated model fails Validate() | Single case |
| TestBugService_DeleteBug_HappyPath | Calls repo.Delete | Single case |
| TestBugService_DeleteBug_NotFound | Propagates NotFoundError | Single case |
| TestBugService_ListBugs_WithFilters | Filters converted to repo filters, results returned | Table-driven |
| TestBugService_ListBugs_EmptyResult | Returns empty slice, not nil | Single case |
| TestBugService_AdvanceBugStatus_HappyPath | Gets next status from workflow, calls UpdateStatus | Single case |
| TestBugService_AdvanceBugStatus_TerminalStatus | Returns terminal status error | Table-driven (resolved, wont_fix, duplicate) |
| TestBugService_AdvanceBugStatus_WorkflowTraversal | Full path: reported -> triaged -> in_fix -> in_verification -> resolved | Sequential |
| TestBugService_SetBugStatus_ValidTransition | Validates transition, updates status | Single case |
| TestBugService_SetBugStatus_InvalidTransition | Returns workflow error | Single case |
| TestBugService_TriageBug_HappyPath | Severity updated, assigned_to set in context, status advanced to triaged | Single case |
| TestBugService_TriageBug_WrongStatus | Returns error when status is not `reported` | Table-driven (triaged, in_fix, resolved) |
| TestBugService_TriageBug_Rollback | All changes rolled back if status advance fails | Single case |
| TestBugService_TriageBug_InvalidSeverity | Returns validation error | Single case |

**Coverage target**: 80%+ overall. 100% for error paths and edge cases.

---

## 3. Integration Scenarios

### 3.1 Dependency: F01 (Database Schema and Workflow Engine Extension)

F02 requires F01 to deliver the following. These are integration validation points, not F02 tests -- but F02 tests will fail if these are missing.

| F01 Deliverable | How F02 Depends on It | Verification |
|-----------------|----------------------|--------------|
| `bugs` table in SQLite schema | BugRepository issues queries against it | Repository tests fail with "no such table: bugs" if missing |
| `workflow.LevelBug` constant | BugService constructor calls `ForLevel(LevelBug)` | Compile error if constant missing |
| Bug workflow in `.sharkconfig.json` | `GetDefaultStatus()`, `GetNextStatus()`, `ValidateTransition()` return correct values for bug statuses | Service tests mock workflow, but integration tests need real config |
| `entity_notes` table supports `entity_type = 'bug'` | Notes and context operations on bugs | NoteService and ContextService tests with bug entity type |

**Action**: Before starting F02 development, run `shark init update --workflow=advanced` and verify the `.sharkconfig.json` contains a `bug_workflow` section. If it does not, F01 is not complete.

### 3.2 Consumer: F04 (Bug CLI Commands)

F04 will call BugService methods as thin wrappers. The following table documents the service API surface that F04 depends on.

| F04 CLI Command | BugService Method Called | Input DTO | Output |
|-----------------|------------------------|-----------|--------|
| `shark bug create "Title" --severity=high --link=E07-F01` | `CreateBug(ctx, CreateBugInput)` | `CreateBugInput{Title, Severity, LinkedEntityKey}` | `*models.Bug` |
| `shark bug get B001` | `GetBug(ctx, "B001")` | key string | `*models.Bug` |
| `shark bug list --status=reported --severity=critical` | `ListBugs(ctx, BugFilters)` | `BugFilters{Status, Severity}` | `[]*models.Bug` |
| `shark bug update B001 --title="New"` | `UpdateBug(ctx, "B001", BugUpdates)` | `BugUpdates{Title *string}` | `*models.Bug` |
| `shark bug delete B001` | `DeleteBug(ctx, "B001")` | key string | `error` |
| `shark bug triage B001 --severity=high --assign=dev` | `TriageBug(ctx, TriageBugInput)` | `TriageBugInput{Key, Severity, AssignedTo}` | `*models.Bug` |
| `shark status advance B001` | `AdvanceBugStatus(ctx, "B001")` | key string | `*models.Bug` |
| `shark status set B001 wont_fix` | `SetBugStatus(ctx, "B001", "wont_fix")` | key, status strings | `*models.Bug` |

**Contract validation**: F04 should NOT need to change any F02 interfaces. If F04 discovers missing methods or incorrect signatures, that is a defect against F02.

### 3.3 Consumer: F06 (Unified CLI Integration)

F06 adds B### key auto-detection to unified commands (`shark get`, `shark status`, `shark delete`). F02 delivers:

- `GetBugService()` global accessor in `services_global.go`
- All service methods accept business-level keys (not database IDs)
- Error types are compatible with existing error handling in unified commands (`NotFoundError`, workflow errors)

### 3.4 Integration: Context and Note Services

F02 modifies `ContextService` and `NoteService` to accept `EntityTypeBug`. Integration tests should verify:

| Test | Action | Expected |
|------|--------|----------|
| Bug context set | `ContextService.SetField(ctx, "bug", "B001", "environment", "Safari 17.2")` | Context field stored in bug's `context_data` JSON |
| Bug context get | `ContextService.GetField(ctx, "bug", "B001", "environment")` | Returns `"Safari 17.2"` |
| Bug note add | `NoteService.AddNote(ctx, "bug", "B001", "comment", "Investigating root cause")` | Note persisted for bug entity |
| Invalid bug entity | `NoteService.AddNote(ctx, "bug", "B999", ...)` | Error: bug not found |

---

## 4. Non-Functional Test Criteria

### 4.1 Performance

| Metric | Target | How Measured |
|--------|--------|--------------|
| Bug creation speed (local SQLite) | < 500ms | Timed integration test: `time` wrapper around `CreateBug` call including key gen, link validation, file creation, and DB insert |
| Bug list performance (1000 bugs) | < 1s | Repository test: seed 1000 bugs, measure `List(ctx, BugListFilters{})` duration |

### 4.2 Data Integrity

| Scenario | Verification |
|----------|-------------|
| Atomic create (file failure rollback) | After simulated file failure, `SELECT COUNT(*) FROM bugs WHERE key = ?` returns 0 |
| Atomic triage (status failure rollback) | After simulated workflow error, severity and context_data remain at pre-triage values |
| Key uniqueness | `CREATE` with duplicate key returns UNIQUE constraint error |
| No key reuse after deletion | Delete B002, create new bug -- key is B003 not B002 |

### 4.3 Pattern Consistency (Code Review Checklist)

These are verified during code review, not via automated tests:

- [ ] `BugService` constructor uses dependency injection (no global state)
- [ ] `BugRepository` is an interface defined in the services package
- [ ] All service methods accept `context.Context` as first parameter
- [ ] Service methods return domain models (`*models.Bug`), not formatted output
- [ ] Error wrapping uses `fmt.Errorf("context: %w", err)` pattern
- [ ] No business logic in repository (CRUD only)
- [ ] Model `Validate()` does NOT check workflow status validity
- [ ] `GetBugService()` accessor follows `GetTaskService()` pattern

---

## 5. Test Execution Order

Tests should be executed in this order to catch blocking failures early:

1. **Model tests** (`go test ./internal/models/ -run TestBug`) -- no dependencies, fastest feedback
2. **Repository tests** (`go test ./internal/repository/ -run TestBugRepository`) -- requires F01 schema
3. **Service tests** (`go test ./internal/services/ -run TestBugService`) -- mocked, no F01 dependency
4. **Full suite** (`make test`) -- validates no regressions across the codebase
5. **Code review** -- pattern consistency checklist (Section 4.3)

**Quality gate**: All 3 test categories must pass and `make fmt && make lint && make test` must succeed before declaring F02 complete.

---

*Last Updated: 2026-03-03*
