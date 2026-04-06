# Test Plan: Tech-Debt Entity Implementation (E25-F01)

**Feature**: E25-F01 Tech-Debt Entity Implementation
**Epic**: E25 Add Tech-Debt Entity Type
**Date**: 2026-04-05
**Status**: Active

---

## 1. Acceptance Criteria Test Matrix

Every requirement from spec.md maps to at least one test case. This table is the primary traceability artifact.

| Requirement | Test Case(s) | Category | Priority |
|-------------|-------------|----------|----------|
| FR-01: Create with auto-key TD-### | TC-REPO-001, TC-REPO-011, TC-SVC-001 | Repo + Service | Critical |
| FR-01: Create with default category/severity | TC-SVC-001, TC-CLI-001 | Service + CLI | Critical |
| FR-01: Read by key (all fields) | TC-REPO-003, TC-SVC-003, TC-CLI-002 | All | Critical |
| FR-01: Update specified fields only | TC-REPO-005, TC-SVC-005, TC-CLI-004 | All | High |
| FR-01: Delete removes from DB | TC-REPO-007, TC-SVC-006, TC-CLI-005 | All | Critical |
| FR-01: List with sort by key | TC-REPO-009, TC-SVC-008, TC-CLI-003 | All | High |
| FR-02: Triage advances identified->triaged | TC-SVC-011, TC-CLI-006 | Service + CLI | High |
| FR-02: Triage when past identified updates fields only | TC-SVC-012 | Service | High |
| FR-03: shark status advance TD-### | TC-SVC-009, TC-INT-003 | Service + Integration | Critical |
| FR-03: shark status set TD-### direct | TC-SVC-010, TC-INT-004 | Service + Integration | Critical |
| FR-03: Invalid transitions rejected with message | TC-SVC-010b, TC-INT-005 | Service + Integration | High |
| FR-03: shark status options TD-### | TC-SVC-013 | Service | Medium |
| FR-03: shark status history TD-### | TC-SVC-014, TC-INT-007 | Service + Integration | Medium |
| FR-04: Notes add with type validation | TC-INT-008 | Integration | Medium |
| FR-04: Notes list | TC-INT-009 | Integration | Medium |
| FR-04: Context set/get/clear | TC-INT-010 | Integration | Medium |
| FR-05: shark get TD-### auto-detection | TC-INT-001 | Integration | Critical |
| FR-05: shark delete TD-### auto-detection | TC-INT-002 | Integration | High |
| FR-05: shark status TD-### routing | TC-INT-003, TC-INT-004 | Integration | Critical |
| FR-05: shark update TD-### routing | TC-CLI-004, TC-INT-011 | CLI + Integration | High |
| FR-05: shark view/history/notes TD-### | TC-INT-007, TC-INT-009 | Integration | Medium |
| FR-05: Case-insensitive key td-001 | TC-REPO-004, TC-CLI-009 | Repo + CLI | High |
| FR-06: Search includes tech_debts by title | TC-INT-012 | Integration | High |
| FR-06: Search matches key and description | TC-INT-013, TC-INT-014 | Integration | Medium |
| FR-07: Analytics includes tech-debt counts | TC-INT-015 | Integration | Medium |
| FR-08: --json flag on all td subcommands | TC-CLI-002, TC-CLI-010 | CLI | High |
| FR-08: --field extracts single field | TC-CLI-010 | CLI | Medium |
| FR-09: Migration creates tech_debts table | TC-INT-016 | Integration | Critical |
| FR-09: Migration is idempotent | TC-INT-017 | Integration | Critical |
| FR-09: Schema version bumped 10->11 | TC-INT-016 | Integration | Critical |
| NFR-01: Indexed columns (status, category, severity) | TC-REPO-014 | Repo | Medium |
| NFR-02: No regressions in existing operations | TC-INT-018 | Integration | Critical |
| NFR-03: Migration does not ALTER existing tables | TC-INT-016, TC-INT-018 | Integration | Critical |

---

## 2. Repository Tests

**Location**: `internal/repository/tech_debt/repository_test.go`
**Pattern**: Real database with cleanup. Uses `test.GetTestDB()` and `dbconn.NewDB()`. Cleans up with key prefix `TD9` (safely outside user data range).

### Setup Pattern

```go
func techDebtTestSetup(t *testing.T) (*Repository, func()) {
    t.Helper()
    ctx := context.Background()
    database := test.GetTestDB()
    db := dbconn.NewDB(database)
    repo := NewTechDebtRepository(db)

    // Clean up existing test data before test
    _, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key LIKE 'TD9%'")

    cleanup := func() {
        _, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key LIKE 'TD9%'")
    }

    return repo, cleanup
}

func newTestTechDebt(key, title, status string, category models.TechDebtCategory, severity models.TechDebtSeverity) *models.TechDebt {
    return &models.TechDebt{
        BaseEntity: models.BaseEntity{Key: key, Title: title},
        Status:     models.TechDebtStatus(status),
        Category:   category,
        Severity:   severity,
    }
}
```

### Test Cases

#### TC-REPO-001: Create — basic success
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtRepository_Create`
- **Steps**: Create a TechDebt with key TD901, title, status identified, category code-quality, severity medium.
- **Expected**: No error; ID is set to non-zero value after Create returns.

#### TC-REPO-002: Create — validation failure (empty title)
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtRepository_Create_ValidationFailure`
- **Steps**: Attempt to Create a TechDebt with empty title.
- **Expected**: Error returned; entity not created.

#### TC-REPO-003: Create — duplicate key rejected
- **FR**: FR-01 (Create), NFR-02 (constraints)
- **Test**: `TestTechDebtRepository_Create_DuplicateKey`
- **Steps**: Create TD901, then attempt to create another entity with key TD901.
- **Expected**: Second create returns an error containing "UNIQUE".

#### TC-REPO-004: GetByKey — success with all fields populated
- **FR**: FR-01 (Read)
- **Test**: `TestTechDebtRepository_GetByKey`
- **Steps**: Create TD901 with category=architecture, severity=high, effort_estimate="2 hours", description="Some description". Retrieve by key.
- **Expected**: Returned entity matches all inserted fields: key, title, status, category, severity, effort_estimate, description.

#### TC-REPO-005: GetByKey — case-insensitive lookup
- **FR**: FR-01 (Read), FR-05 (auto-detection)
- **Test**: `TestTechDebtRepository_GetByKey_CaseInsensitive`
- **Steps**: Create TD901 (uppercase). Call GetByKey with "td901" (lowercase).
- **Expected**: Entity returned correctly; no error.

#### TC-REPO-006: GetByKey — not found returns error
- **FR**: FR-01 (Read)
- **Test**: `TestTechDebtRepository_GetByKey_NotFound`
- **Steps**: Call GetByKey("TD999") when no such entity exists.
- **Expected**: Error returned; result is nil.

#### TC-REPO-007: Update — all mutable fields
- **FR**: FR-01 (Update)
- **Test**: `TestTechDebtRepository_Update`
- **Steps**: Create TD901 (title="Original", severity=medium, category=code-quality, effort_estimate=nil). Update title="Changed", severity=critical, category=architecture, effort_estimate="1 sprint". Retrieve and verify.
- **Expected**: All four fields reflect updated values; status and key unchanged.

#### TC-REPO-008: UpdateStatus — status field updated independently
- **FR**: FR-01 (Update), FR-03 (workflow)
- **Test**: `TestTechDebtRepository_UpdateStatus`
- **Steps**: Create TD901 (status=identified). Call UpdateStatus(id, "triaged", nil). Retrieve.
- **Expected**: Status is "triaged"; other fields unchanged.

#### TC-REPO-009: Delete — entity no longer retrievable
- **FR**: FR-01 (Delete)
- **Test**: `TestTechDebtRepository_Delete`
- **Steps**: Create TD901. Delete by key TD901. Attempt GetByKey("TD901").
- **Expected**: Delete returns no error; subsequent GetByKey returns error.

#### TC-REPO-010: List — returns all entities, ordered by key
- **FR**: FR-01 (List)
- **Test**: `TestTechDebtRepository_List`
- **Steps**: Create TD901, TD902, TD903 in non-sequential order. Call List().
- **Expected**: Returns 3 entities (at minimum) in ascending key order.

#### TC-REPO-011: ListWithFilters — filter by category
- **FR**: FR-01 (List)
- **Test**: `TestTechDebtRepository_ListWithFilters_Category`
- **Steps**: Create TD901 (category=architecture), TD902 (category=testing), TD903 (category=architecture). ListWithFilters(category="architecture").
- **Expected**: Returns exactly TD901 and TD903; TD902 excluded.

#### TC-REPO-012: ListWithFilters — filter by severity
- **FR**: FR-01 (List)
- **Test**: `TestTechDebtRepository_ListWithFilters_Severity`
- **Steps**: Create TD901 (severity=critical), TD902 (severity=low). ListWithFilters(severity="critical").
- **Expected**: Returns only TD901.

#### TC-REPO-013: ListWithFilters — filter by status
- **FR**: FR-01 (List)
- **Test**: `TestTechDebtRepository_ListWithFilters_Status`
- **Steps**: Create TD901 (status=identified), TD902 (status=triaged). ListWithFilters(status="identified").
- **Expected**: Returns only TD901.

#### TC-REPO-014: ListWithFilters — combined multiple filters
- **FR**: FR-01 (List)
- **Test**: `TestTechDebtRepository_ListWithFilters_Combined`
- **Steps**: Create: TD901 (architecture, critical, identified), TD902 (architecture, low, identified), TD903 (testing, critical, identified). ListWithFilters(category="architecture", severity="critical").
- **Expected**: Returns only TD901.

#### TC-REPO-015: GenerateNextKey — first key is TD-001
- **FR**: FR-01 (Create), FR-09 (migration)
- **Test**: `TestTechDebtRepository_GenerateNextKey_FirstKey`
- **Steps**: Ensure no tech_debts exist. Call GenerateNextKey().
- **Expected**: Returns "TD-001".
- **Note**: Run this test in isolation with full table cleanup.

#### TC-REPO-016: GenerateNextKey — increments from highest existing key
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtRepository_GenerateNextKey_Increment`
- **Steps**: Create entities with keys TD901, TD903 (gap intentional). Call GenerateNextKey().
- **Expected**: Returns "TD-904" (next after the highest, TD903).

#### TC-REPO-017: CountByStatus — aggregation matches data
- **FR**: FR-07 (analytics)
- **Test**: `TestTechDebtRepository_CountByStatus`
- **Steps**: Create TD901 (identified), TD902 (identified), TD903 (triaged). Call CountByStatus().
- **Expected**: Returns map with "identified" -> 2, "triaged" -> 1 (at minimum for test data).

#### TC-REPO-018: CountByCategory — aggregation matches data
- **FR**: FR-07 (analytics)
- **Test**: `TestTechDebtRepository_CountByCategory`
- **Steps**: Create TD901 (architecture), TD902 (testing), TD903 (architecture). Call CountByCategory().
- **Expected**: Returns map with "architecture" -> 2, "testing" -> 1 (at minimum).

#### TC-REPO-019: Database constraint — invalid category rejected
- **FR**: FR-09 (migration), NFR-02 (constraints)
- **Test**: `TestTechDebtRepository_DatabaseConstraints_CategoryCheck`
- **Steps**: Attempt to INSERT directly with category="invalid-value" bypassing model validation.
- **Expected**: Database-level CHECK constraint rejects the insert; error returned.

#### TC-REPO-020: Database constraint — invalid severity rejected
- **FR**: FR-09 (migration), NFR-02 (constraints)
- **Test**: `TestTechDebtRepository_DatabaseConstraints_SeverityCheck`
- **Steps**: Attempt to INSERT directly with severity="super-critical" bypassing model validation.
- **Expected**: Database-level CHECK constraint rejects the insert; error returned.

#### TC-REPO-021: updated_at trigger — fires on update
- **FR**: FR-09 (migration)
- **Test**: `TestTechDebtRepository_UpdatedAtTrigger`
- **Steps**: Create TD901. Record created_at. Sleep 1 second. Update title. Retrieve and compare updated_at.
- **Expected**: updated_at > created_at after update.

---

## 3. Service Tests

**Location**: `internal/services/tech_debt_service_test.go`
**Pattern**: Mocked repositories using function-field structs. No real database. Follows pattern in `internal/services/bug_service_test.go`.

### Mock Definitions

Defined in `internal/services/tech_debt_service_test.go` (or a shared `mocks_test.go`):

```go
type mockTechDebtRepo struct {
    createFn           func(ctx context.Context, td *models.TechDebt) error
    getByKeyFn         func(ctx context.Context, key string) (*models.TechDebt, error)
    getByIDFn          func(ctx context.Context, id int64) (*models.TechDebt, error)
    updateFn           func(ctx context.Context, td *models.TechDebt) error
    deleteFn           func(ctx context.Context, id int64) error
    updateStatusFn     func(ctx context.Context, id int64, status models.TechDebtStatus, notes *string) error
    getNextKeyFn       func(ctx context.Context) (string, error)
    listFn             func(ctx context.Context, filters TechDebtFilters) ([]*models.TechDebt, error)
    countByStatusFn    func(ctx context.Context) (map[string]int, error)
    countByCategoryFn  func(ctx context.Context) (map[string]int, error)
}
```

### Test Cases

#### TC-SVC-001: Create — success with field defaults
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtService_Create_Success`
- **Mock setup**: getNextKeyFn returns "TD-001"; createFn captures args and verifies.
- **Input**: CreateTechDebtInput{Title: "Fix N+1", Category: "performance", Severity: "high"}
- **Expected**: Returned entity has Key="TD-001", Status="identified", Category="performance", Severity="high". createFn was called once with the entity.

#### TC-SVC-002: Create — default category applied when not specified
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtService_Create_DefaultCategory`
- **Mock setup**: getNextKeyFn returns "TD-001"; createFn succeeds.
- **Input**: CreateTechDebtInput{Title: "Some debt"} (no Category or Severity)
- **Expected**: Entity created with Category="code-quality", Severity="medium".

#### TC-SVC-003: Create — empty title returns validation error
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtService_Create_EmptyTitle`
- **Mock setup**: getNextKeyFn and createFn not called.
- **Input**: CreateTechDebtInput{Title: "   "} (whitespace only)
- **Expected**: Error returned; createFn NOT called; error message indicates title validation failure.

#### TC-SVC-004: Create — invalid category returns validation error
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtService_Create_InvalidCategory`
- **Mock setup**: getNextKeyFn and createFn not called.
- **Input**: CreateTechDebtInput{Title: "Test", Category: "frontend"}
- **Expected**: Error returned; error message indicates invalid category.

#### TC-SVC-005: Get — success
- **FR**: FR-01 (Read)
- **Test**: `TestTechDebtService_Get_Success`
- **Mock setup**: getByKeyFn returns a valid TechDebt with key "TD-001".
- **Input**: key="TD-001"
- **Expected**: Returned entity matches mock. getByKeyFn called with "TD-001".

#### TC-SVC-006: Get — not found propagated
- **FR**: FR-01 (Read)
- **Test**: `TestTechDebtService_Get_NotFound`
- **Mock setup**: getByKeyFn returns an error (not found).
- **Input**: key="TD-999"
- **Expected**: Error propagated; result nil.

#### TC-SVC-007: Update — partial update applies only specified fields
- **FR**: FR-01 (Update)
- **Test**: `TestTechDebtService_Update_PartialUpdate`
- **Mock setup**: getByKeyFn returns entity with severity=medium, category=testing. updateFn captures args.
- **Input**: TechDebtUpdates{Severity: ptr("critical")} (only severity specified)
- **Expected**: updateFn called with entity where Severity="critical" and Category="testing" (unchanged).

#### TC-SVC-008: Update — title update only
- **FR**: FR-01 (Update)
- **Test**: `TestTechDebtService_Update_TitleOnly`
- **Mock setup**: getByKeyFn returns entity. updateFn captures args.
- **Input**: TechDebtUpdates{Title: ptr("New Title")}
- **Expected**: updateFn entity has new title; all other fields unchanged.

#### TC-SVC-009: Delete — success
- **FR**: FR-01 (Delete)
- **Test**: `TestTechDebtService_Delete_Success`
- **Mock setup**: getByKeyFn returns entity with ID=42. deleteFn succeeds.
- **Input**: key="TD-001"
- **Expected**: deleteFn called with id=42; no error.

#### TC-SVC-010: List — filters passed to repository
- **FR**: FR-01 (List)
- **Test**: `TestTechDebtService_List_WithFilters`
- **Mock setup**: listFn captures filters arg and returns empty slice.
- **Input**: TechDebtFilters{Category: "architecture", Status: "identified"}
- **Expected**: listFn called with matching filters.

#### TC-SVC-011: AdvanceStatus — valid transition (identified -> triaged)
- **FR**: FR-03 (workflow)
- **Test**: `TestTechDebtService_AdvanceStatus_Success`
- **Mock setup**: getByKeyFn returns entity (status=identified). Workflow service validates transition. updateStatusFn captures new status.
- **Input**: key="TD-001"
- **Expected**: updateStatusFn called with "triaged"; returned entity has status="triaged".

#### TC-SVC-012: AdvanceStatus — terminal state returns error
- **FR**: FR-03 (workflow)
- **Test**: `TestTechDebtService_AdvanceStatus_TerminalState`
- **Mock setup**: getByKeyFn returns entity (status=resolved). Workflow service returns error (no next status).
- **Input**: key="TD-001"
- **Expected**: Error returned; updateStatusFn NOT called.

#### TC-SVC-013: SetStatus — valid transition succeeds
- **FR**: FR-03 (workflow)
- **Test**: `TestTechDebtService_SetStatus_ValidTransition`
- **Mock setup**: getByKeyFn returns entity (status=triaged). Workflow service validates "triaged"->"wont_fix". updateStatusFn captures args.
- **Input**: key="TD-001", status="wont_fix"
- **Expected**: updateStatusFn called with "wont_fix"; no error.

#### TC-SVC-014: SetStatus — invalid transition returns workflow error
- **FR**: FR-03 (workflow)
- **Test**: `TestTechDebtService_SetStatus_InvalidTransition`
- **Mock setup**: getByKeyFn returns entity (status=identified). Workflow service rejects "identified"->"resolved".
- **Input**: key="TD-001", status="resolved"
- **Expected**: Error returned containing workflow violation message; updateStatusFn NOT called.

#### TC-SVC-015: SetStatus — force flag bypasses transition validation
- **FR**: FR-03 (workflow)
- **Test**: `TestTechDebtService_SetStatus_Force`
- **Mock setup**: getByKeyFn returns entity. updateStatusFn captures args.
- **Input**: key="TD-001", status="resolved", force=true
- **Expected**: updateStatusFn called with "resolved" regardless of workflow validation result.

#### TC-SVC-016: Triage — from identified: advances to triaged and updates fields
- **FR**: FR-02 (triage)
- **Test**: `TestTechDebtService_Triage_FromIdentified`
- **Mock setup**: getByKeyFn returns entity (status=identified). updateStatusFn and updateFn capture args.
- **Input**: key="TD-001", TriageTechDebtInput{Severity: "high", Category: "architecture", EffortEstimate: "2 hours"}
- **Expected**: updateStatusFn called with "triaged"; updateFn called with updated fields (severity=high, category=architecture, effort_estimate=2 hours).

#### TC-SVC-017: Triage — from triaged: updates fields without status change
- **FR**: FR-02 (triage)
- **Test**: `TestTechDebtService_Triage_AlreadyTriaged`
- **Mock setup**: getByKeyFn returns entity (status=triaged). updateFn captures args.
- **Input**: key="TD-001", TriageTechDebtInput{Severity: "critical"}
- **Expected**: updateFn called with severity=critical; updateStatusFn NOT called (status already past identified).

#### TC-SVC-018: GetStatusOptions — returns valid next statuses
- **FR**: FR-03 (workflow)
- **Test**: `TestTechDebtService_GetStatusOptions`
- **Mock setup**: getByKeyFn returns entity (status=triaged). Workflow service returns ["in_progress", "wont_fix", "cancelled"].
- **Input**: key="TD-001"
- **Expected**: Returned options slice contains exactly "in_progress", "wont_fix", "cancelled".

#### TC-SVC-019: GetStatusHistory — returns history entries
- **FR**: FR-03 (workflow)
- **Test**: `TestTechDebtService_GetStatusHistory`
- **Mock setup**: History repo returns two entries (identified->triaged, triaged->in_progress).
- **Input**: key="TD-001"
- **Expected**: Two entries returned in chronological order.

#### TC-SVC-020: Create — key generation failure propagated
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtService_Create_KeyGenerationFailure`
- **Mock setup**: getNextKeyFn returns error.
- **Input**: CreateTechDebtInput{Title: "Test"}
- **Expected**: Error propagated; createFn NOT called.

#### TC-SVC-021: Create — repository create failure propagated
- **FR**: FR-01 (Create)
- **Test**: `TestTechDebtService_Create_RepoFailure`
- **Mock setup**: getNextKeyFn returns "TD-001"; createFn returns a DB error.
- **Input**: CreateTechDebtInput{Title: "Test"}
- **Expected**: DB error propagated with context wrapping.

---

## 4. CLI Tests

**Location**: `internal/cli/commands/tech_debt_test.go`
**Pattern**: Mocked services. No real database, no real service. Uses `tdSvcOverride` injection pattern from the command file.

### Mock Service Interface

```go
type mockTechDebtServicer struct {
    createFn         func(ctx context.Context, input services.CreateTechDebtInput) (*models.TechDebt, error)
    getFn            func(ctx context.Context, key string) (*models.TechDebt, error)
    listFn           func(ctx context.Context, filters services.TechDebtFilters) ([]*models.TechDebt, error)
    updateFn         func(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error)
    deleteFn         func(ctx context.Context, key string) error
    triageFn         func(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error)
    advanceStatusFn  func(ctx context.Context, key string) (*models.TechDebt, error)
    setStatusFn      func(ctx context.Context, key string, status string, reason string, force bool) (*models.TechDebt, error)
}
```

### Test Cases

#### TC-CLI-001: td create — positional title and flags parsed correctly
- **FR**: FR-01 (Create), FR-08 (JSON)
- **Test**: `TestTdCreate_ArgsAndFlags`
- **Setup**: Inject mock service; createFn captures input.
- **Command args**: `td create "Refactor auth module" --category=architecture --severity=high --effort-estimate="3 days"`
- **Expected**: createFn called with Title="Refactor auth module", Category="architecture", Severity="high", EffortEstimate="3 days".

#### TC-CLI-002: td create — defaults when flags omitted
- **FR**: FR-01 (Create)
- **Test**: `TestTdCreate_Defaults`
- **Setup**: Inject mock service returning entity with defaults.
- **Command args**: `td create "Some debt"`
- **Expected**: createFn called with Category="" or service default; no error from missing flags.

#### TC-CLI-003: td get — human output format
- **FR**: FR-01 (Read), FR-08 (JSON)
- **Test**: `TestTdGet_Output`
- **Setup**: Mock getFn returns entity with all fields populated.
- **Command args**: `td get TD-001`
- **Expected**: Output contains key, title, status, category, severity.

#### TC-CLI-004: td get — JSON output format
- **FR**: FR-08 (JSON)
- **Test**: `TestTdGet_JSONOutput`
- **Setup**: Mock getFn returns entity. GlobalConfig.JSON = true.
- **Command args**: `td get TD-001 --json`
- **Expected**: stdout is valid JSON containing fields: key, title, status, category, severity, effort_estimate, file_path.

#### TC-CLI-005: td get — --field extracts single value
- **FR**: FR-08 (JSON)
- **Test**: `TestTdGet_FieldExtraction`
- **Setup**: Mock getFn returns entity with status="in_progress".
- **Command args**: `td get TD-001 --field status`
- **Expected**: Output is exactly "in_progress" (plain string, no JSON wrapper).

#### TC-CLI-006: td list — filter flags parsed and passed to service
- **FR**: FR-01 (List)
- **Test**: `TestTdList_FilterFlags`
- **Setup**: Mock listFn captures filters arg.
- **Command args**: `td list --category=testing --severity=medium --status=identified`
- **Expected**: listFn called with TechDebtFilters{Category: "testing", Severity: "medium", Status: "identified"}.

#### TC-CLI-007: td list — JSON output
- **FR**: FR-08 (JSON)
- **Test**: `TestTdList_JSONOutput`
- **Setup**: Mock listFn returns two entities. GlobalConfig.JSON = true.
- **Command args**: `td list --json`
- **Expected**: stdout is a valid JSON array with 2 elements.

#### TC-CLI-008: td update — only specified flags included in update DTO
- **FR**: FR-01 (Update)
- **Test**: `TestTdUpdate_PartialFlags`
- **Setup**: Mock updateFn captures updates arg.
- **Command args**: `td update TD-001 --severity=critical`
- **Expected**: updateFn called with TechDebtUpdates where only Severity is set (non-nil), Title and Category are nil.

#### TC-CLI-009: td delete — force flag skips confirmation
- **FR**: FR-01 (Delete)
- **Test**: `TestTdDelete_ForceFlag`
- **Setup**: Mock deleteFn succeeds.
- **Command args**: `td delete TD-001 --force`
- **Expected**: deleteFn called with "TD-001"; no stdin prompt required.

#### TC-CLI-010: td triage — flag parsing
- **FR**: FR-02 (triage)
- **Test**: `TestTdTriage_ArgsAndFlags`
- **Setup**: Mock triageFn captures input.
- **Command args**: `td triage TD-001 --severity=high --category=testing --effort-estimate="S"`
- **Expected**: triageFn called with TriageTechDebtInput{Severity: "high", Category: "testing", EffortEstimate: "S"}.

#### TC-CLI-011: DetectEntityType — TD-001 detected as tech_debt
- **FR**: FR-05 (auto-detection)
- **Test**: `TestDetectEntityType_TDKey`
- **Setup**: Pure function test; no mock needed.
- **Input**: key="TD-001"
- **Expected**: DetectEntityType("TD-001") returns "tech_debt".

#### TC-CLI-012: DetectEntityType — td-001 lowercase detected as tech_debt
- **FR**: FR-05 (auto-detection)
- **Test**: `TestDetectEntityType_TDKey_CaseInsensitive`
- **Input**: key="td-001"
- **Expected**: DetectEntityType("td-001") returns "tech_debt".

#### TC-CLI-013: DetectEntityType — TD key takes precedence before task key check
- **FR**: FR-05 (auto-detection), architecture correctness
- **Test**: `TestDetectEntityType_TDKeyBeforeTaskKey`
- **Description**: Ensures TD-001 is not mis-detected as a slug-based task key (since TD starts with T).
- **Input**: key="TD-042"
- **Expected**: DetectEntityType("TD-042") returns "tech_debt", NOT "task".

#### TC-CLI-014: ParseGetArgs — TD-001 parsed with scopeTechDebt
- **FR**: FR-05 (auto-detection)
- **Test**: `TestParseGetArgs_TDKey`
- **Input**: args=["TD-001"]
- **Expected**: ParseGetArgs returns scope type scopeTechDebt, key "TD-001".

#### TC-CLI-015: td create — error from service propagates
- **FR**: FR-01 (Create)
- **Test**: `TestTdCreate_ServiceError`
- **Setup**: Mock createFn returns error.
- **Command args**: `td create "Test"`
- **Expected**: Command returns error; non-zero exit.

#### TC-CLI-016: td get — not found error formats helpful message
- **FR**: FR-01 (Read)
- **Test**: `TestTdGet_NotFoundError`
- **Setup**: Mock getFn returns NotFoundError.
- **Command args**: `td get TD-999`
- **Expected**: Output contains helpful "not found" message referencing TD-999.

---

## 5. Integration Scenarios

These scenarios test cross-cutting concerns and end-to-end paths. They may use real binaries or real databases depending on test infrastructure. For the developer test suite, these map to tests in `internal/repository/` (for DB integration) or table-driven acceptance tests.

### Core Command Auto-Detection

#### TC-INT-001: shark get TD-### routes to tech-debt get
- **FR**: FR-05 (auto-detection)
- **Scenario**: TD-001 exists. Run `shark get TD-001`.
- **Expected**: Output shows tech-debt details (key, title, status, category) — same as `shark td get TD-001`. No "unsupported entity type" error.

#### TC-INT-002: shark delete TD-### routes to tech-debt delete
- **FR**: FR-05 (auto-detection)
- **Scenario**: TD-001 exists. Run `shark delete TD-001 --force`.
- **Expected**: Entity deleted from database. `shark td get TD-001` returns not-found error.

### Workflow Integration

#### TC-INT-003: shark status advance TD-### full lifecycle
- **FR**: FR-03 (workflow)
- **Scenario**: Create TD-001 (identified). Advance 3 times.
- **Expected**: Status sequence: identified -> triaged -> in_progress -> resolved. Each advance returns no error and correct new status.

#### TC-INT-004: shark status set TD-### direct status change
- **FR**: FR-03 (workflow)
- **Scenario**: TD-001 in triaged. Run `shark status set TD-001 wont_fix`.
- **Expected**: Status updated to wont_fix. `shark td get TD-001 --field status` returns "wont_fix".

#### TC-INT-005: shark status advance on terminal state returns error
- **FR**: FR-03 (workflow)
- **Scenario**: TD-001 in resolved. Run `shark status advance TD-001`.
- **Expected**: Error returned indicating terminal state. Status remains resolved.

#### TC-INT-006: shark status options shows valid transitions
- **FR**: FR-03 (workflow)
- **Scenario**: TD-001 in triaged. Run `shark status options TD-001`.
- **Expected**: Output lists "in_progress", "wont_fix", "cancelled" as valid next statuses.

#### TC-INT-007: shark history TD-### shows status change log
- **FR**: FR-03 (workflow)
- **Scenario**: Create TD-001, advance to triaged, advance to in_progress. Run `shark history TD-001`.
- **Expected**: History shows two transitions: identified->triaged and triaged->in_progress with timestamps.

### Notes and Context

#### TC-INT-008: shark td note add persists note
- **FR**: FR-04 (notes)
- **Scenario**: TD-001 exists. Run `shark td note add TD-001 --content="Found in review" --type=comment`.
- **Expected**: Note persisted. `shark td notes TD-001` shows the note with content and type.

#### TC-INT-009: All valid note types accepted
- **FR**: FR-04 (notes)
- **Scenario**: Add notes with types: comment, decision, blocker, solution, reference, implementation, testing, future, question, rejection, requirement.
- **Expected**: All 11 types accepted without validation error.

#### TC-INT-010: shark td context set/get/clear
- **FR**: FR-04 (context)
- **Scenario**: Set `affected_area` = "internal/repository/". Get context. Clear context field. Get context again.
- **Expected**: After set: context includes affected_area. After clear: context does not include affected_area.

### Core Command Routing (Additional)

#### TC-INT-011: shark update TD-### routes to tech-debt update
- **FR**: FR-05 (auto-detection)
- **Scenario**: TD-001 exists (severity=low). Run `shark update TD-001 --severity=critical`.
- **Expected**: Severity updated to critical. `shark td get TD-001 --field severity` returns "critical".

### Search Integration

#### TC-INT-012: shark search finds tech-debt by title
- **FR**: FR-06 (search)
- **Scenario**: TD-001 exists with title "Refactor database connection pooling". Run `shark search "database"`.
- **Expected**: TD-001 appears in search results with entity_type "tech_debt".

#### TC-INT-013: shark search finds tech-debt by key
- **FR**: FR-06 (search)
- **Scenario**: TD-001 exists. Run `shark search "TD-001"`.
- **Expected**: TD-001 appears in results.

#### TC-INT-014: shark search finds tech-debt by description
- **FR**: FR-06 (search)
- **Scenario**: TD-002 created with description "authentication module has no unit tests". Run `shark search "authentication"`.
- **Expected**: TD-002 appears in results.

#### TC-INT-015: shark analytics includes tech-debt summary
- **FR**: FR-07 (analytics)
- **Scenario**: Create 2 tech-debt items (1 identified, 1 triaged). Run `shark analytics`.
- **Expected**: Output includes tech-debt section with total count and breakdown by status.

### Database Migration

#### TC-INT-016: Migration creates tech_debts table with correct schema
- **FR**: FR-09 (migration)
- **Steps**:
  1. Inspect DB schema after InitDB() runs (schema version 11).
  2. Verify `tech_debts` table exists with all columns: id, key, title, slug, description, status, category, severity, effort_estimate, context_data, file_path, created_at, updated_at.
  3. Verify all 5 indexes exist: idx_tech_debts_key, idx_tech_debts_status, idx_tech_debts_severity, idx_tech_debts_category, idx_tech_debts_slug.
  4. Verify trigger tech_debts_updated_at exists.
  5. Verify CurrentSchemaVersion == 11 in db.go constant.
- **Expected**: All schema elements present; no errors.

#### TC-INT-017: Migration is idempotent (CREATE TABLE IF NOT EXISTS)
- **FR**: FR-09 (migration), NFR-03 (migration safety)
- **Steps**: Run InitDB() twice (simulate double migration).
- **Expected**: Second run completes without error. No duplicate tables or indexes.

#### TC-INT-018: Existing data untouched after migration
- **FR**: NFR-02 (backward compatibility), NFR-03 (migration safety)
- **Steps**:
  1. Seed epics, features, tasks, bugs, change-cards.
  2. Record counts for each entity type.
  3. Run migration (simulate by bumping schema and calling runMigrations).
  4. Re-query all entity types.
- **Expected**: All counts identical before and after migration. No data corruption in any existing table.

### Key Format Edge Cases

#### TC-INT-019: Case-insensitive key normalization end-to-end
- **FR**: FR-05 (auto-detection), spec section 2.4
- **Scenario**: Create TD-001. Run `shark get td-001` (lowercase).
- **Expected**: Entity retrieved correctly; same output as `shark get TD-001`.

#### TC-INT-020: Key format TD-### does not collide with task key detection
- **FR**: FR-05 (auto-detection), architecture spec ADR-2
- **Description**: Verifies TD-prefix check runs before task slug detection in DetectEntityType.
- **Scenario**: Create TD-001 and a task E01-F01-001 with slug "td-style-naming". Run `shark get TD-001`.
- **Expected**: Correctly routes to tech-debt, not task.

---

## 6. Quality Gates

All of the following must pass before the feature is approved:

| Gate | Requirement | Command |
|------|-------------|---------|
| Code formatting | All new Go code formatted | `make fmt` (no diff) |
| Static analysis | No linting warnings | `make lint` |
| Full test suite | All tests pass including new tests | `make test` |
| Regression | No existing tests broken | `make test` (all green) |
| Schema version | CurrentSchemaVersion == 11 in db.go | Code inspection |
| Migration warning | dev notes to run with skip_migrations: false | Code review |

**Minimum coverage targets (new code only)**:
- Repository: 80%+ (measured via `go test -cover ./internal/repository/tech_debt/...`)
- Service: 80%+ (measured via `go test -cover ./internal/services/...`)
- Model validation: 100% for Validate() and ValidateTechDebtKey()

---

## 7. Test Execution Order

Execute in this order to validate each implementation phase independently:

### Phase 1: Foundation (no DB)
1. Model validation unit tests (Validate(), ValidateTechDebtKey(), category/severity constants)
2. Key detection unit tests (IsTechDebtKey, DetectEntityType)
3. Workflow defaults unit tests (DefaultTechDebtWorkflow shape)

### Phase 2: Data Layer
4. TC-INT-016: Migration schema validation
5. TC-INT-017: Migration idempotency
6. TC-REPO-001 through TC-REPO-021 (all repository tests)

### Phase 3: Service Layer
7. TC-SVC-001 through TC-SVC-021 (all service tests)

### Phase 4: CLI Layer
8. TC-CLI-001 through TC-CLI-016 (all CLI tests)

### Phase 5: Integration
9. TC-INT-001 through TC-INT-020 (all integration scenarios)

### Phase 6: Regression
10. Full `make test` — verifies no existing tests broken by the implementation.

---

## 8. Defect Severity Definitions

| Severity | Condition | Release Gate |
|----------|-----------|--------------|
| Critical | Any TC-REPO, TC-SVC, or TC-INT test failing; migration fails; key detection wrong | Blocks release |
| High | CLI output format wrong; filter not applied; triage behavior incorrect | Blocks release |
| Medium | --field extraction edge case; analytics count off; history order | Fix before release |
| Low | Search result ordering; cosmetic output formatting | Document and defer |

---

*Test Plan by: QA agent | Date: 2026-04-05*
