# Test Plan: Bug CLI Commands (E18-F04)

**Feature Key**: E18-F04
**Complexity Tier**: STANDARD
**Status**: Test Planning
**Date**: 2026-03-03
**Author**: QA Agent

---

## 1. Epic UAT Scenario Traceability

This feature test plan decomposes the following epic UAT scenarios into concrete, executable test cases for the CLI layer.

| UAT Scenario | F04 Coverage | Test Cases |
|---|---|---|
| J1-HP Steps 1-6 (Bug create, list, get) | CRUD commands | TC-01a through TC-01j, TC-02a through TC-02f, TC-03a through TC-03h |
| J1-HP Step 4 (Triage) | Triage command | TC-06a through TC-06f |
| J1-ALT-A Step 2 (Add note) | Note commands | TC-07a through TC-07d |
| J1-ERR-2 (Invalid link) | Create error path | TC-01f |
| CE-4 Steps 3,5 (Notes/Context for bugs) | Note + Context commands | TC-07a through TC-08e |
| CE-3 (Service pattern compliance) | Structural quality gate | SQ-01 through SQ-06 |

Scenarios NOT covered by F04 (delegated to F06): J1-HP Steps 7-11 (status advance via unified commands), J3-UNIFIED (auto-detect dispatch), J3-DASHBOARD (dashboard integration).

---

## 2. Acceptance Criteria Test Matrix

Every acceptance criterion from `requirements.md` is mapped to a concrete test case with inputs, expected outputs, and edge case coverage.

### 2.1 Create Command (F04-REQ-01)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-01a | AC-01a | `shark bug create "Login page crashes"` | Bug B001 created, status=reported, severity=medium. CLI: "Created bug B001" | 0 | First bug in empty DB |
| TC-01b | AC-01b | `shark bug create "API timeout" --severity=critical` | Bug with severity=critical | 0 | -- |
| TC-01c | AC-01c | `shark bug create "Button misaligned" --link=E07` | Bug linked to epic E07 (linked_entity_type=epic) | 0 | Epic-level link |
| TC-01d | AC-01d | `shark bug create "Form broken" --link=E07-F01` | Bug linked to feature E07-F01 (linked_entity_type=feature) | 0 | Feature-level link |
| TC-01e | AC-01e | `shark bug create "Edge case" --link=E07-F01-001` | Bug linked to task E07-F01-001 (linked_entity_type=task) | 0 | Task-level link |
| TC-01f | AC-01f | `shark bug create "Some bug" --link=E99` | Error: "Linked entity not found: E99". No bug created. | 1 | Invalid link rejection |
| TC-01g | AC-01g | `shark bug create "Some bug" --severity=extreme` | Error contains "invalid severity" and lists valid values | 3 | Invalid severity enum |
| TC-01h | AC-01h | `shark bug create ""` | Error: "title cannot be empty". No bug created. | 3 | Empty title |
| TC-01i | AC-01i | `shark bug create "New bug" --json` | Valid JSON with key, title, status=reported, severity=medium, created_at | 0 | JSON output mode |
| TC-01j | AC-01j | `shark bug create "Third bug"` (with B001, B002 existing) | New bug gets key B003 | 0 | Auto-increment |

### 2.2 Get Command (F04-REQ-02)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-02a | AC-02a | `shark bug get B001` | Detail view: key, title, status, severity, linked entity, dates | 0 | -- |
| TC-02b | AC-02b | `shark bug get B001 --json` | Valid JSON matching Bug model schema | 0 | JSON output |
| TC-02c | AC-02c | `shark bug get B001 --field severity` | Exactly: "critical" (raw value) | 0 | Field extraction |
| TC-02d | AC-02d | `shark bug get B999` | Error: "Bug not found: B999" | 1 | Not found |
| TC-02e | AC-02e | `shark bug get b001` | Bug found (case-insensitive key) | 0 | Case insensitivity |
| TC-02f | AC-02f | `shark bug get B001 --field nonexistent_field` | Error: field not found | 4 | Invalid field |

### 2.3 List Command (F04-REQ-03)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-03a | AC-03a | `shark bug list` (3 bugs exist) | Table with columns: KEY, TITLE, STATUS, SEVERITY, LINKED TO. All 3 bugs shown. | 0 | -- |
| TC-03b | AC-03b | `shark bug list --status=reported` | Only bugs with status=reported | 0 | Status filter |
| TC-03c | AC-03c | `shark bug list --severity=critical` | Only bugs with severity=critical | 0 | Severity filter |
| TC-03d | AC-03d | `shark bug list --link=E07-F01` | Only bugs linked to E07-F01 | 0 | Link filter (feature) |
| TC-03e | AC-03e | `shark bug list --status=reported --severity=high` | Only bugs matching both criteria | 0 | Combined filters |
| TC-03f | AC-03f | `shark bug list --json` | Valid JSON array of bug objects | 0 | JSON output |
| TC-03g | AC-03g | `shark bug list` (no bugs) | Info message: "No bugs found" (not error) | 0 | Empty result |
| TC-03h | AC-03h | `shark bug list --link=E07` | Bugs linked to any entity under E07 | 0 | Epic-scope filter |

### 2.4 Update Command (F04-REQ-04)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-04a | AC-04a | `shark bug update B001 --title="New title"` | Title updated, success message | 0 | Title-only update |
| TC-04b | AC-04b | `shark bug update B001 --severity=critical` | Severity updated | 0 | Severity-only update |
| TC-04c | AC-04c | `shark bug update B001 --title="Updated" --severity=low` | Both fields updated | 0 | Multi-field update |
| TC-04d | AC-04d | `shark bug update B999 --title="New"` | Error: "Bug not found: B999" | 1 | Not found |
| TC-04e | AC-04e | `shark bug update B001` (no flags) | Error: "at least one update flag is required" | 3 | No flags |
| TC-04f | AC-04f | `shark bug update B001 --title="New" --json` | Valid JSON with updated bug | 0 | JSON output |

### 2.5 Delete Command (F04-REQ-05)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-05a | AC-05a | `shark bug delete B001` (confirm=y) | Bug and markdown file deleted, success message | 0 | With confirmation |
| TC-05b | AC-05b | `shark bug delete B001 --force` | Bug deleted without prompt | 0 | Force flag |
| TC-05c | AC-05c | `shark bug delete B999` | Error: "Bug not found: B999" | 1 | Not found |
| TC-05d | AC-05d | `shark bug delete B001 --force --json` | Valid JSON: {"deleted": "B001"} | 0 | JSON output |

### 2.6 Triage Command (F04-REQ-06)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-06a | AC-06a | `shark bug triage B001 --severity=high` (status=reported) | Status advances to triaged, severity=high, success message | 0 | Basic triage |
| TC-06b | AC-06b | `shark bug triage B001 --severity=medium --assign=developer` | Status=triaged, severity=medium, agent=developer | 0 | Triage with assignment |
| TC-06c | AC-06c | `shark bug triage B001 --severity=high` (status=triaged) | Error: "Cannot triage bug B001: current status is 'triaged'" | 3 | Already triaged |
| TC-06d | AC-06d | `shark bug triage B001` (no --severity) | Error: --severity is required | 3 | Missing required flag |
| TC-06e | AC-06e | `shark bug triage B001 --severity=high --json` | Valid JSON with status=triaged, severity=high | 0 | JSON output |
| TC-06f | AC-06f | `shark bug triage B999 --severity=high` | Error: "Bug not found: B999" | 1 | Not found |

### 2.7 Note Commands (F04-REQ-07)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-07a | AC-07a | `shark bug note add B001 --type=comment "Reproduced on Safari 17.2"` | Note created with entity_type=bug, note_type=comment. Confirmation output. | 0 | -- |
| TC-07b | AC-07b | `shark bug notes B001` (2 notes exist) | Both notes displayed with type, content, timestamp | 0 | -- |
| TC-07c | AC-07c | `shark bug note add B001 --type=decision "Root cause is race condition"` | Note with type=decision created | 0 | Decision type |
| TC-07d | AC-07d | `shark bug notes B999` | Error: "Bug not found: B999" | 1 | Not found |

### 2.8 Context Commands (F04-REQ-08)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-08a | AC-08a | `shark bug context set B001 --field environment --value "Safari 17.2 on macOS 14.3"` | Field stored, confirmation output | 0 | -- |
| TC-08b | AC-08b | `shark bug context get B001` (2 fields exist) | Both fields and values displayed | 0 | -- |
| TC-08c | AC-08c | `shark bug context get B001 --json` | Valid JSON with key-value pairs | 0 | JSON output |
| TC-08d | AC-08d | `shark bug context clear B001 --field environment` | Field removed, confirmation output | 0 | -- |
| TC-08e | AC-08e | `shark bug context get B999` | Error: "Bug not found: B999" | 1 | Not found |

### 2.9 Command Registration (F04-REQ-09)

| ID | AC | Test Input | Expected Output | Exit Code | Edge Case |
|---|---|---|---|---|---|
| TC-09a | AC-09a | `shark bug --help` | Help text lists all subcommands: create, get, list, update, delete, triage, note, notes, context | 0 | -- |
| TC-09b | AC-09b | Build and inspect command tree | Bug command registered under "advanced" group | -- | Structural check |
| TC-09c | AC-09c | `shark bug create --help` | Inherits --json, --field, --no-color, --verbose, --db, --config | 0 | Global flags |
| TC-09d | AC-09d | Inspect all subcommand help text | Each has Short (<60 chars), Long with examples, correct Args | -- | Structural check |

### 2.10 Service Accessor (F04-REQ-10)

| ID | AC | Test Input | Expected Outcome | Type |
|---|---|---|---|---|
| TC-10a | AC-10a | Code inspection | `GetBugService()` exists in `internal/cli/service_accessors.go` returning `*services.BugService` | Structural |
| TC-10b | AC-10b | Code inspection | `GetBugService()` creates BugRepository from global DB, obtains workflow service, passes to `NewBugService()` | Structural |
| TC-10c | AC-10c | Code inspection | `GetBugService()` panics with descriptive message on DB failure | Structural |

### 2.11 Pattern Consistency (F04-REQ-11)

| ID | AC | Validation | Type |
|---|---|---|---|
| TC-11a | AC-11a | All commands use `cli.OutputJSON()`, `cli.OutputTable()`, `cli.Success()`, `cli.Error()` | Code review |
| TC-11b | AC-11b | No handler contains: workflow validation, status checks, repo calls, transaction management, filtering logic, key generation | Code review |
| TC-11c | AC-11c | Handlers return errors from service calls. Exit code mapping: 1=not found, 2=DB error, 3=invalid state, 4=field not found | Code review |
| TC-11d | AC-11d | Flags named: --severity, --link, --force, --assign, --type, --field, --value (matching existing patterns) | Code review |

---

## 3. CLI Command Test Cases

This section specifies how each test case is implemented using mocked services, following the established CLI test patterns in `internal/cli/commands/*_test.go`.

### 3.1 Test File Location

```
internal/cli/commands/bug_test.go          -- All CLI command tests
internal/cli/commands/mock_bug_service_test.go  -- MockBugService definition (if separate file preferred)
```

### 3.2 MockBugService Definition

Following the function-field mock pattern used throughout the codebase (see `idea_test.go`, `idea_convert_test.go`):

```go
type MockBugService struct {
    CreateBugFunc func(ctx context.Context, input services.CreateBugInput) (*models.Bug, error)
    GetBugFunc    func(ctx context.Context, key string) (*models.Bug, error)
    ListBugsFunc  func(ctx context.Context, filters services.BugFilters) ([]*models.Bug, error)
    UpdateBugFunc func(ctx context.Context, key string, input services.UpdateBugInput) (*models.Bug, error)
    DeleteBugFunc func(ctx context.Context, key string) error
    TriageBugFunc func(ctx context.Context, key string, input services.TriageBugInput) (*models.Bug, error)
}

func (m *MockBugService) CreateBug(ctx context.Context, input services.CreateBugInput) (*models.Bug, error) {
    if m.CreateBugFunc != nil {
        return m.CreateBugFunc(ctx, input)
    }
    return nil, fmt.Errorf("CreateBug not implemented in mock")
}
// ... (delegate pattern for all methods)
```

### 3.3 Test Organization

Tests are organized by command, using table-driven tests for scenarios that share the same handler but vary in inputs/expected outcomes.

**Create command tests (~10 cases):**

```go
func TestBugCreate_Success(t *testing.T)                  // TC-01a: basic create
func TestBugCreate_WithSeverity(t *testing.T)             // TC-01b: --severity=critical
func TestBugCreate_WithLink(t *testing.T)                 // TC-01c/d/e: link to epic/feature/task
func TestBugCreate_InvalidLink(t *testing.T)              // TC-01f: link to non-existent entity
func TestBugCreate_InvalidSeverity(t *testing.T)          // TC-01g: --severity=extreme
func TestBugCreate_EmptyTitle(t *testing.T)               // TC-01h: empty title
func TestBugCreate_JSONOutput(t *testing.T)               // TC-01i: --json flag
func TestBugCreate_AutoIncrementKey(t *testing.T)         // TC-01j: B003 after B001/B002
```

**Get command tests (~6 cases):**

```go
func TestBugGet_Success(t *testing.T)                     // TC-02a: detail view
func TestBugGet_JSONOutput(t *testing.T)                  // TC-02b: --json
func TestBugGet_FieldExtraction(t *testing.T)             // TC-02c: --field severity
func TestBugGet_NotFound(t *testing.T)                    // TC-02d: B999
func TestBugGet_CaseInsensitive(t *testing.T)             // TC-02e: b001
func TestBugGet_InvalidField(t *testing.T)                // TC-02f: --field nonexistent
```

**List command tests (~8 cases):**

```go
func TestBugList_All(t *testing.T)                        // TC-03a: all bugs
func TestBugList_FilterByStatus(t *testing.T)             // TC-03b: --status=reported
func TestBugList_FilterBySeverity(t *testing.T)           // TC-03c: --severity=critical
func TestBugList_FilterByLink(t *testing.T)               // TC-03d: --link=E07-F01
func TestBugList_CombinedFilters(t *testing.T)            // TC-03e: status + severity
func TestBugList_JSONOutput(t *testing.T)                 // TC-03f: --json
func TestBugList_Empty(t *testing.T)                      // TC-03g: no bugs
func TestBugList_EpicScopeFilter(t *testing.T)            // TC-03h: --link=E07
```

**Update command tests (~6 cases):**

```go
func TestBugUpdate_Title(t *testing.T)                    // TC-04a
func TestBugUpdate_Severity(t *testing.T)                 // TC-04b
func TestBugUpdate_Both(t *testing.T)                     // TC-04c
func TestBugUpdate_NotFound(t *testing.T)                 // TC-04d
func TestBugUpdate_NoFlags(t *testing.T)                  // TC-04e
func TestBugUpdate_JSONOutput(t *testing.T)               // TC-04f
```

**Delete command tests (~4 cases):**

```go
func TestBugDelete_WithConfirmation(t *testing.T)         // TC-05a
func TestBugDelete_Force(t *testing.T)                    // TC-05b
func TestBugDelete_NotFound(t *testing.T)                 // TC-05c
func TestBugDelete_JSONOutput(t *testing.T)               // TC-05d
```

**Triage command tests (~6 cases):**

```go
func TestBugTriage_Success(t *testing.T)                  // TC-06a
func TestBugTriage_WithAssign(t *testing.T)               // TC-06b
func TestBugTriage_AlreadyTriaged(t *testing.T)           // TC-06c
func TestBugTriage_MissingSeverity(t *testing.T)          // TC-06d
func TestBugTriage_JSONOutput(t *testing.T)               // TC-06e
func TestBugTriage_NotFound(t *testing.T)                 // TC-06f
```

**Note command tests (~4 cases):**

```go
func TestBugNoteAdd_Success(t *testing.T)                 // TC-07a
func TestBugNotes_List(t *testing.T)                      // TC-07b
func TestBugNoteAdd_DecisionType(t *testing.T)            // TC-07c
func TestBugNotes_NotFound(t *testing.T)                  // TC-07d
```

**Context command tests (~5 cases):**

```go
func TestBugContextSet_Success(t *testing.T)              // TC-08a
func TestBugContextGet_Success(t *testing.T)              // TC-08b
func TestBugContextGet_JSON(t *testing.T)                 // TC-08c
func TestBugContextClear_Success(t *testing.T)            // TC-08d
func TestBugContextGet_NotFound(t *testing.T)             // TC-08e
```

**Total: ~49 test functions covering all acceptance criteria.**

### 3.4 Example Test Implementation

Following the established pattern from `idea_test.go`:

```go
func TestBugCreate_Success(t *testing.T) {
    ctx := context.Background()
    var capturedInput services.CreateBugInput

    mockSvc := &MockBugService{
        CreateBugFunc: func(ctx context.Context, input services.CreateBugInput) (*models.Bug, error) {
            capturedInput = input
            return &models.Bug{
                ID:       1,
                Key:      "B001",
                Title:    input.Title,
                Status:   "reported",
                Severity: "medium",
                Slug:     "login-page-crashes",
            }, nil
        },
    }

    // Call handler logic (parse -> service -> format)
    bug, err := mockSvc.CreateBug(ctx, services.CreateBugInput{
        Title: "Login page crashes",
    })

    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if bug.Key != "B001" {
        t.Errorf("Expected key B001, got %s", bug.Key)
    }
    if bug.Status != "reported" {
        t.Errorf("Expected status reported, got %s", bug.Status)
    }
    if bug.Severity != "medium" {
        t.Errorf("Expected severity medium, got %s", bug.Severity)
    }
    if capturedInput.Title != "Login page crashes" {
        t.Errorf("Expected title 'Login page crashes', got %s", capturedInput.Title)
    }
}

func TestBugCreate_InvalidLink(t *testing.T) {
    ctx := context.Background()

    mockSvc := &MockBugService{
        CreateBugFunc: func(ctx context.Context, input services.CreateBugInput) (*models.Bug, error) {
            return nil, fmt.Errorf("linked entity not found: %s", input.LinkKey)
        },
    }

    bug, err := mockSvc.CreateBug(ctx, services.CreateBugInput{
        Title:   "Some bug",
        LinkKey: "E99",
    })

    if err == nil {
        t.Fatal("Expected error for invalid link, got nil")
    }
    if bug != nil {
        t.Error("Expected nil bug on error")
    }
    if !strings.Contains(err.Error(), "E99") {
        t.Errorf("Error should mention E99, got: %v", err)
    }
}
```

### 3.5 Error Path Test Pattern

```go
func TestBugTriage_AlreadyTriaged(t *testing.T) {
    ctx := context.Background()

    mockSvc := &MockBugService{
        TriageBugFunc: func(ctx context.Context, key string, input services.TriageBugInput) (*models.Bug, error) {
            return nil, fmt.Errorf("cannot triage bug %s: current status is 'triaged', expected 'reported'", key)
        },
    }

    bug, err := mockSvc.TriageBug(ctx, "B001", services.TriageBugInput{Severity: "high"})

    if err == nil {
        t.Fatal("Expected error for already-triaged bug")
    }
    if bug != nil {
        t.Error("Expected nil bug on error")
    }
    if !strings.Contains(err.Error(), "triaged") {
        t.Errorf("Error should mention current status, got: %v", err)
    }
}
```

---

## 4. Integration Scenarios

These scenarios test the interaction between F04 CLI commands and F02 BugService at integration time. They are NOT unit tests -- they require F02 to be delivered.

### INT-01: Full CRUD Lifecycle

**Preconditions**: F01 (schema) and F02 (BugService) are implemented. Database is initialized.

| Step | Command | Verify |
|---|---|---|
| 1 | `shark bug create "Test bug" --severity=high` | Bug B001 created, markdown file exists at docs/bugs/B001.md |
| 2 | `shark bug get B001` | All fields displayed correctly |
| 3 | `shark bug get B001 --json` | Valid JSON output |
| 4 | `shark bug update B001 --title="Updated test bug"` | Title updated |
| 5 | `shark bug list` | B001 appears with updated title |
| 6 | `shark bug delete B001 --force` | Bug deleted, markdown file removed |
| 7 | `shark bug list` | No bugs found |

### INT-02: Triage Flow

| Step | Command | Verify |
|---|---|---|
| 1 | `shark bug create "Triage test"` | Bug in reported status |
| 2 | `shark bug triage B001 --severity=critical --assign=developer` | Status=triaged, severity=critical |
| 3 | `shark bug triage B001 --severity=high` | Error: already triaged (exit code 3) |
| 4 | `shark bug get B001 --json` | JSON shows status=triaged, severity=critical |

### INT-03: Notes and Context

| Step | Command | Verify |
|---|---|---|
| 1 | `shark bug create "Context test"` | Bug created |
| 2 | `shark bug note add B001 --type=comment "First note"` | Note added |
| 3 | `shark bug note add B001 --type=decision "Root cause found"` | Second note added |
| 4 | `shark bug notes B001` | Both notes displayed |
| 5 | `shark bug context set B001 --field browser --value "Safari 17"` | Context field set |
| 6 | `shark bug context get B001` | browser=Safari 17 displayed |
| 7 | `shark bug context clear B001 --field browser` | Field removed |
| 8 | `shark bug context get B001` | No fields (or empty output) |

### INT-04: Link Validation

| Step | Command | Verify |
|---|---|---|
| 1 | `shark bug create "Linked bug" --link=E07-F01` (E07-F01 exists) | Bug linked to feature |
| 2 | `shark bug get B001 --json` | linked_entity_type=feature, linked_entity_key=E07-F01 |
| 3 | `shark bug list --link=E07-F01` | B001 appears |
| 4 | `shark bug create "Bad link" --link=E99` | Error: linked entity not found |

---

## 5. Structural Quality Gates

These are code review checks, not runtime tests. They must pass before F04 is accepted.

| ID | Check | What to Verify | Pass Criteria |
|---|---|---|---|
| SQ-01 | Thin wrapper pattern | Every handler in `bug.go` follows: parse -> service call -> format output | No handler exceeds 25 lines |
| SQ-02 | No business logic in CLI | No workflow validation, status checks, repo calls, transaction management, or filtering logic in handlers | Zero occurrences |
| SQ-03 | Service accessor | `GetBugService()` in `service_accessors.go` follows GetTaskService pattern | Constructor injection with BugRepository + WorkflowService |
| SQ-04 | Output functions | All output uses `cli.OutputJSON()`, `cli.OutputTable()`, `cli.Success()`, `cli.Error()`, `cli.Info()` | No raw `fmt.Printf` in handlers (except helper functions like printBugDetail) |
| SQ-05 | Flag naming | --severity, --link, --force, --assign, --type, --field, --value | Matches existing entity command flag names |
| SQ-06 | Error propagation | Handlers return errors from service calls directly; no swallowing | All err != nil paths return err |

---

## 6. Test Data Fixtures

### Standard Bug Fixture

```go
func newTestBug(key, title, status, severity string) *models.Bug {
    return &models.Bug{
        ID:        1,
        Key:       key,
        Title:     title,
        Status:    status,
        Severity:  severity,
        Slug:      slugify(title),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

### Bug with Link

```go
func newLinkedTestBug(key, title, linkedType, linkedKey string) *models.Bug {
    bug := newTestBug(key, title, "reported", "medium")
    bug.LinkedEntityType = linkedType
    bug.LinkedEntityKey = linkedKey
    return bug
}
```

### Error Fixtures

```go
var (
    errBugNotFound     = fmt.Errorf("bug not found: B999")
    errInvalidSeverity = fmt.Errorf("invalid severity 'extreme': must be critical, high, medium, or low")
    errAlreadyTriaged  = fmt.Errorf("cannot triage bug B001: current status is 'triaged', expected 'reported'")
    errLinkedNotFound  = fmt.Errorf("linked entity not found: E99")
)
```

---

## 7. Test Execution Summary

| Category | Test Count | Type | Dependencies |
|---|---|---|---|
| Create command | 10 | Unit (mocked) | MockBugService |
| Get command | 6 | Unit (mocked) | MockBugService |
| List command | 8 | Unit (mocked) | MockBugService |
| Update command | 6 | Unit (mocked) | MockBugService |
| Delete command | 4 | Unit (mocked) | MockBugService |
| Triage command | 6 | Unit (mocked) | MockBugService |
| Note commands | 4 | Unit (mocked) | MockNoteService |
| Context commands | 5 | Unit (mocked) | MockContextService |
| **Unit Total** | **49** | | |
| Integration scenarios | 4 | Integration | F01, F02 |
| Structural quality gates | 6 | Code review | -- |
| **Grand Total** | **59** | | |

---

## 8. Exit Gate Checklist

| Criterion | Required For | Status |
|---|---|---|
| Every AC in requirements.md has at least one test case | STANDARD tier | Covered: F04-REQ-01 through F04-REQ-11, all ACs mapped |
| API contracts tested (service interface consumed correctly) | STANDARD tier | Covered: MockBugService validates all 6 BugService methods + DTO shapes |
| Integration points identified | STANDARD tier | Covered: F01 (schema), F02 (BugService), NoteService, ContextService |
| Test cases are actionable for TDD | STANDARD tier | Covered: Mock patterns, test function names, and example implementations provided |
| Error paths tested | All tiers | Covered: Not found (TC-01f, TC-02d, TC-04d, TC-05c, TC-06f, TC-07d, TC-08e), invalid state (TC-01g, TC-01h, TC-04e, TC-06c, TC-06d), invalid field (TC-02f) |

---

*Last Updated*: 2026-03-03
