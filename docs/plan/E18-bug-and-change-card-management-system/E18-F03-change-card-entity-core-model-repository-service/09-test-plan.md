# Test Plan: Change-Card Entity Core (Model, Repository, Service)

**Feature**: E18-F03 - Change-Card Entity Core (Model, Repository, Service)
**Epic**: E18 - Bug and Change-Card Management System
**Complexity**: STANDARD (S-M)
**Test Plan Type**: Focused Test Plan (AC Matrix + Component Test Strategy + API Contract Tests + Integration Scenarios)
**Date**: 2026-03-03

---

## Executive Summary

This test plan validates the Change-Card entity stack (model, repository, service) which provides the backend for the change-card lifecycle. The plan covers:

1. **Model structural validation** -- ChangeCard.Validate() enforces non-empty title and status without importing workflow packages
2. **Repository CRUD and queries** -- Real-database tests for Create, Get, Update, Delete, key auto-generation, dual-key lookup, linked entity filtering, and status counting
3. **Service business logic** -- Mocked-repository tests for create (with/without link), approve, advance, set-status, list with filters, delete, and all error paths
4. **Workflow integration** -- All status transitions validated via workflowSvc, never hardcoded

**Critical Success Criteria**:
- All 10 feature-level acceptance scenarios (from PRD) pass
- Model validation tests: 4 cases (valid, empty title, empty status, whitespace title)
- Repository tests: 11 test groups with real database and cleanup
- Service tests: 13 test groups with mocked repositories
- Zero workflow status values hardcoded in model or service
- Architecture compliance: model -> repository (pure CRUD) -> service (business logic)

**UAT Traceability**: This feature provides the backend for UAT scenarios J2-HP (Change-Card Happy Path), J2-ALT-A (Declined), J2-ALT-B (Promotion), J2-ERR-1 (Re-approve), and J2-ERR-2 (Decline from wrong status) from the E18 UAT Acceptance Plan.

---

## 1. Acceptance Criteria Test Matrix

### Story 1: Create Change-Card (Service Layer)

**User Goal**: As a service consumer, I want to call CreateChangeCard() with a title and optional link, so that a change-card is created with an auto-generated C### key, persisted in the database, and a markdown file is generated.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC1.1** | CreateChangeCard returns ChangeCard with C### key | Call CreateChangeCard with valid title | `CreateChangeCardInput{Title: "Add dark mode"}` | Returns `*models.ChangeCard` with Key matching `^C\d{3}$` (e.g., C001) | First change-card gets C001; key increments monotonically |
| **AC1.2** | Returned change-card has status `proposed` | Inspect status field of returned change-card | Same as AC1.1 | `Status == "proposed"` (read from workflowSvc.GetDefaultStatus()) | Default status must come from workflow, not hardcoded |
| **AC1.3** | Key is globally unique and auto-incremented | Create 3 change-cards sequentially | Three CreateChangeCard calls | Keys: C001, C002, C003 | After deleting C002, next key is C004 (never reused) |
| **AC1.4** | Slug auto-generated from title | Create with title "Add Dark Mode Toggle" | `CreateChangeCardInput{Title: "Add Dark Mode Toggle"}` | Slug: `add-dark-mode-toggle` | Special chars stripped: "Fix: login!!!" -> `fix-login` |
| **AC1.5** | Markdown file created at docs/changes/C###.md | Check filesystem after create | Same as AC1.1 | File exists at `docs/changes/C001.md` | Directory auto-created if missing |
| **AC1.6** | Markdown frontmatter contains key, title, status, linked entity fields | Read generated markdown file | Same as AC1.1 | Frontmatter has `change_card_key: C001`, `title: Add dark mode`, `status: proposed` | Linked entity fields omitted when no link |
| **AC1.7** | Atomic: file failure rolls back DB record | Mock file writer to return error | CreateChangeCard with simulated file failure | Error returned; no DB record exists | Verify via GetByKey returning NotFoundError |

### Story 2: Create Change-Card with Link Validation

**User Goal**: As a service consumer, I want to create a change-card linked to an epic or feature with validated references.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC2.1** | Link to epic sets LinkedEntityType="epic" | Create with epic link | `CreateChangeCardInput{Title: "...", LinkedEntityKey: "E07"}` | `LinkedEntityType: "epic"`, `LinkedEntityKey: "E07"` | Case insensitive: "e07" works |
| **AC2.2** | Link to feature sets LinkedEntityType="feature" | Create with feature link | `CreateChangeCardInput{Title: "...", LinkedEntityKey: "E07-F01"}` | `LinkedEntityType: "feature"`, `LinkedEntityKey: "E07-F01"` | Short format "F01" also detected |
| **AC2.3** | Non-existent link returns error, no record created | Create with bad link | `CreateChangeCardInput{Title: "...", LinkedEntityKey: "E99"}` | Error containing "epic E99 not found"; no DB record | No markdown file created either |
| **AC2.4** | Empty link creates unlinked change-card | Create without link | `CreateChangeCardInput{Title: "..."}` | `LinkedEntityType: nil`, `LinkedEntityKey: nil` | Both fields nil, not empty string |
| **AC2.5** | Entity type auto-detected from key format | Create with E## and E##-F## keys | Both epic and feature format keys | Correct entity type set automatically | Ambiguous format returns clear error |
| **AC2.6** | Task key rejected with clear error | Create with task link | `CreateChangeCardInput{Title: "...", LinkedEntityKey: "E07-F01-001"}` | Error: "task linking is not supported for change-cards" | No DB record, no file created |

### Story 3: ApproveChangeCard

**User Goal**: As a service consumer, I want to call ApproveChangeCard() to advance from proposed to approved.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC3.1** | Approve from proposed succeeds | Approve C001 in proposed status | `ApproveChangeCard(ctx, "C001")` | Status updated to `approved` | Workflow transition validated via workflowSvc |
| **AC3.2** | Approve from non-proposed returns error | Approve C001 in in_progress status | `ApproveChangeCard(ctx, "C001")` (status: in_progress) | Error: "cannot approve change-card C001: current status is 'in_progress', must be 'proposed'" | Status is not modified |
| **AC3.3** | Workflow transition validated via workflowSvc | Mock workflowSvc.ValidateTransition to reject | ApproveChangeCard with workflow rejection | Error from workflow service propagated | No hardcoded status checks |

### Story 4: AdvanceChangeCardStatus

**User Goal**: As a service consumer, I want to advance a change-card to its next valid workflow status.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC4.1** | Advance from approved to in_progress | Advance C001 in approved status | `AdvanceChangeCardStatus(ctx, "C001")` | Status updated to `in_progress` | Next status from workflowSvc.GetNextStatus() |
| **AC4.2** | Advance from terminal status returns error | Advance C001 in completed status | `AdvanceChangeCardStatus(ctx, "C001")` | Error: "change-card C001 is in terminal status 'completed'; no further transitions available" | Also test with `declined` |
| **AC4.3** | Full workflow traversal | Advance through all statuses | Approve then advance twice | `proposed -> approved -> in_progress -> completed` | Each transition recorded |

### Story 5: SetChangeCardStatus

**User Goal**: As a service consumer, I want to set a change-card status directly (e.g., to declined).

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC5.1** | Set to declined from proposed | Set C001 to declined | `SetChangeCardStatus(ctx, "C001", "declined")` | Status: `declined` | Declined is a terminal state |
| **AC5.2** | Invalid transition rejected | Set C001 from in_progress to declined | `SetChangeCardStatus(ctx, "C001", "declined")` (status: in_progress) | Error: workflow transition invalid | Transition validation via workflowSvc |
| **AC5.3** | Further advances after declined fail | Advance C001 in declined status | `AdvanceChangeCardStatus(ctx, "C001")` after decline | Terminal status error | Consistent with AC4.2 |

### Story 6: ListChangeCards with Filters

**User Goal**: As a service consumer, I want to retrieve change-cards filtered by status and/or linked entity.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC6.1** | Empty filters return all (non-terminal) | List with default filters | `ListChangeCards(ctx, ChangeCardFilters{})` | All non-terminal change-cards | Completed and declined excluded by default |
| **AC6.2** | Status filter works | List with status=proposed | `ChangeCardFilters{Status: "proposed"}` | Only proposed change-cards | Empty result if none match |
| **AC6.3** | Linked entity filter works | List linked to E07 | `ChangeCardFilters{LinkedEntityKey: "E07"}` | Only change-cards linked to E07 | Does not return unlinked cards |
| **AC6.4** | ShowAll=false excludes terminal | List with ShowAll=false | `ChangeCardFilters{ShowAll: false}` | Excludes completed and declined | Default behavior |
| **AC6.5** | ShowAll=true includes all | List with ShowAll=true | `ChangeCardFilters{ShowAll: true}` | Includes completed and declined | All statuses returned |
| **AC6.6** | Results sorted by creation date desc | List all | Default listing | Newest first | Verify order with multiple cards |

### Story 7: CRUD Operations (Get, Update, Delete)

**User Goal**: As a service consumer, I want standard CRUD operations for change-cards.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC7.1** | GetChangeCard returns by key | Get existing card | `GetChangeCard(ctx, "C001")` | Returns ChangeCard with correct fields | NotFoundError for non-existent key |
| **AC7.2** | GetChangeCard accepts slugged key | Get with slug | `GetChangeCard(ctx, "C001-add-dark-mode")` | Returns same card as numeric key | Mismatched slug returns not-found |
| **AC7.3** | UpdateChangeCard updates specified fields | Update title only | `UpdateChangeCard(ctx, "C001", ChangeCardUpdates{Title: ptr("New Title")})` | Title changed; other fields unchanged | Nil fields are not updated |
| **AC7.4** | DeleteChangeCard removes record and file | Delete existing card | `DeleteChangeCard(ctx, "C001")` | DB record gone; markdown file deleted | File missing is logged but not error |
| **AC7.5** | DeleteChangeCard returns NotFoundError for missing | Delete non-existent | `DeleteChangeCard(ctx, "C999")` | NotFoundError returned | No panic |

### Story 8: Entity Note Support

**User Goal**: As a service consumer, I want change-cards to support notes via the existing entity notes system.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC8.1** | EntityTypeChange constant defined | Check models package | `models.EntityTypeChange` | Value: `"change"` | Present in ValidEntityTypes map |
| **AC8.2** | NoteService accepts "change" entity type | Create note for change-card | NoteService.CreateNote with entity_type="change" | Note created successfully | Entity existence validated before note creation |
| **AC8.3** | ContextService accepts "change" entity type | Set context field | ContextService.SetField with entity_type="change" | Context updated | GetField also works |

---

## 2. Component Test Strategy

### 2.1 Model Layer: ChangeCard (`internal/models/change_card.go`)

**Test file**: `internal/models/change_card_test.go`
**Test type**: Unit tests (no database, no dependencies)

| Test Name | Input | Expected | Validates |
|-----------|-------|----------|-----------|
| TestChangeCard_Validate_Valid | Title="Test", Status="proposed" | nil error | Happy path structural validation |
| TestChangeCard_Validate_EmptyTitle | Title="", Status="proposed" | Error: "title cannot be empty" | Empty string rejection |
| TestChangeCard_Validate_WhitespaceTitle | Title="   ", Status="proposed" | Error: "title cannot be empty" | Whitespace-only rejection (TrimSpace) |
| TestChangeCard_Validate_EmptyStatus | Title="Test", Status="" | Error: "status cannot be empty" | Status presence check |

**Architecture Compliance Check**: Verify `change_card.go` does NOT import `internal/workflow`, `internal/config`, or any package that would couple the model to workflow definitions.

### 2.2 Repository Layer: ChangeCardRepository (`internal/repository/change_card_repository.go`)

**Test file**: `internal/repository/change_card_repository_test.go`
**Test type**: Integration tests (real SQLite database with cleanup)
**Setup pattern**: Uses `test.GetTestDB()`, cleans `change_cards` table before each test with `DELETE FROM change_cards WHERE key LIKE 'TEST-C%'` or `DELETE FROM change_cards`.

| Test Group | Tests | Validates |
|-----------|-------|-----------|
| **Create + GetByKey** | Create card, retrieve by key, verify all fields match | Round-trip CRUD correctness |
| **Key Auto-Generation** | Create 3 cards sequentially; verify keys C001, C002, C003 | Monotonic increment, zero-padding |
| **Key Never Reused** | Create C001, delete C001, create again; verify new key is C002 | Keys never recycled |
| **Dual Key Lookup (numeric)** | Create card, GetByKey("C001") | Exact key match works |
| **Dual Key Lookup (slugged)** | Create card with slug, GetByKey("C001-add-dark-mode") | Key+slug composite lookup |
| **Dual Key Lookup (case insensitive)** | GetByKey("c001") | Case normalization |
| **List with Status Filter** | Create cards in different statuses, filter by "proposed" | Status filtering correctness |
| **List Excluding Terminal** | Create proposed + completed + declined, list with IncludeTerminal=false | Terminal statuses excluded |
| **ListByLinkedEntity** | Create cards linked to different entities, query by entity | Linked entity filtering |
| **Update** | Modify title, verify updated_at changes | Update correctness, trigger fires |
| **Delete** | Delete card, verify GetByKey returns not-found | Delete correctness, RowsAffected check |
| **UpdateStatus** | Change status, verify via GetByKey | Status column update |
| **CountByStatus** | Create cards in various statuses, verify counts | GROUP BY correctness |
| **GetNextKey** | Empty table returns C001; after 3 inserts returns C004 | MAX query correctness |
| **Not Found** | GetByKey("C999") | Returns not-found error (not panic) |
| **Unique Constraint** | Insert two cards with same key | Returns UNIQUE constraint error |

**Cleanup pattern**: All tests use `defer database.ExecContext(ctx, "DELETE FROM change_cards WHERE ...")` and clean before each test.

### 2.3 Service Layer: ChangeCardService (`internal/services/change_card_service.go`)

**Test file**: `internal/services/change_card_service_test.go`
**Test type**: Unit tests with mocked repositories (no database)
**Mock pattern**: Function-field mocks per project convention

**Mock Interfaces Required**:

```
MockChangeCardRepository:
  - CreateFunc, GetByKeyFunc, GetByIDFunc, UpdateFunc, DeleteFunc
  - UpdateStatusFunc, ListFunc, ListByLinkedEntityFunc
  - CountByStatusFunc, GetNextKeyFunc

MockWorkflowService:
  - ValidateTransitionFunc, GetNextStatusFunc, GetDefaultStatusFunc
  - IsTerminalStatusFunc

MockEpicRepository:
  - GetByKeyFunc (for link validation)

MockFeatureRepository:
  - GetByKeyFunc (for link validation)
```

| Test Group | Mock Setup | Verifies |
|-----------|------------|----------|
| **CreateChangeCard (happy path)** | GetNextKey returns "C001", Create succeeds, workflow returns default status "proposed" | Key assigned, status is default, Validate() called, slug generated |
| **CreateChangeCard (with epic link)** | EpicRepo.GetByKey returns epic | LinkedEntityType="epic", LinkedEntityKey set correctly |
| **CreateChangeCard (with feature link)** | FeatureRepo.GetByKey returns feature | LinkedEntityType="feature", LinkedEntityKey set |
| **CreateChangeCard (invalid link)** | EpicRepo.GetByKey returns not-found | Error returned, Create NOT called (verify via call count) |
| **CreateChangeCard (task link rejected)** | No mock needed (detected by key format) | Error: "task linking is not supported for change-cards" |
| **CreateChangeCard (empty title)** | N/A | Error from Validate(): "title cannot be empty" |
| **ApproveChangeCard (happy path)** | GetByKey returns card with status=proposed, workflow validates transition | UpdateStatus called with "approved" |
| **ApproveChangeCard (wrong status)** | GetByKey returns card with status=in_progress | Error includes current status; UpdateStatus NOT called |
| **AdvanceChangeCardStatus (happy)** | GetByKey returns card, GetNextStatus returns next status | UpdateStatus called with next status |
| **AdvanceChangeCardStatus (terminal)** | GetNextStatus returns error (terminal) | Terminal status error returned |
| **SetChangeCardStatus (valid)** | ValidateTransition returns nil | UpdateStatus called with target status |
| **SetChangeCardStatus (invalid)** | ValidateTransition returns error | Transition error returned; UpdateStatus NOT called |
| **ListChangeCards (no filters)** | List returns all cards | ShowAll=false sets IncludeTerminal=false in repo filter |
| **ListChangeCards (status filter)** | List returns filtered cards | Status filter passed to repo |
| **ListChangeCards (linked entity filter)** | List returns filtered cards | LinkedEntityKey filter passed to repo |
| **ListChangeCards (ShowAll)** | List returns all including terminal | IncludeTerminal=true in repo filter |
| **GetChangeCard (found)** | GetByKey returns card | Card returned with no error |
| **GetChangeCard (not found)** | GetByKey returns NotFoundError | Error propagated with wrapping |
| **UpdateChangeCard** | GetByKey returns card, Update succeeds | Only non-nil fields applied, Validate() called |
| **DeleteChangeCard (found)** | GetByKey returns card, Delete succeeds | Both GetByKey and Delete called |
| **DeleteChangeCard (not found)** | GetByKey returns NotFoundError | Error returned, Delete NOT called |
| **CountByStatus** | CountByStatus returns map | Delegates to repo, no transformation |

---

## 3. API Contract Tests

The ChangeCardService is the API contract for downstream consumers (CLI commands in F05, unified integration in F06). These tests validate the contract surface.

### 3.1 Service Constructor Contract

| Contract | Test | Expected |
|----------|------|----------|
| NewChangeCardService requires non-nil db | Pass nil db | Panic with message "requires a non-nil *repository.DB" |
| NewChangeCardService requires non-nil repo | Pass nil repo | Panic with message "requires a non-nil ChangeCardRepository" |
| NewChangeCardService requires non-nil workflowSvc | Pass nil workflowSvc | Panic with message "requires a non-nil workflow.Service" |
| NewChangeCardService accepts nil epicRepo | Pass nil epicRepo | No panic; link validation degrades gracefully |
| NewChangeCardService accepts nil featureRepo | Pass nil featureRepo | No panic; link validation degrades gracefully |
| Workflow scoped to LevelChange | Inspect workflowSvc after construction | ForLevel("change") called |

### 3.2 DTO Contracts

**CreateChangeCardInput**:

| Field | Type | Required | Contract |
|-------|------|----------|----------|
| Title | string | Yes | Non-empty after trim; error if empty |
| LinkedEntityKey | string | No | If set, must match E## or E##-F## format; entity must exist |

**ChangeCardFilters**:

| Field | Type | Default | Contract |
|-------|------|---------|----------|
| Status | string | "" | If set, filters to matching status only |
| LinkedEntityKey | string | "" | If set, filters to matching link only |
| ShowAll | bool | false | If false, excludes completed + declined |

**ChangeCardUpdates**:

| Field | Type | Contract |
|-------|------|----------|
| Title | *string | If non-nil, updates title (must be non-empty after trim) |
| LinkedEntityKey | *string | If non-nil, validates and updates link; set to "" to unlink |

### 3.3 Return Value Contracts

| Method | Returns | Error Types |
|--------|---------|------------|
| CreateChangeCard | *models.ChangeCard (fully populated, ID set) | Validation error, link not-found error, DB error |
| GetChangeCard | *models.ChangeCard | NotFoundError |
| ListChangeCards | []*models.ChangeCard (never nil, empty slice if none) | DB error |
| UpdateChangeCard | *models.ChangeCard (updated) | NotFoundError, validation error |
| DeleteChangeCard | error (nil on success) | NotFoundError |
| ApproveChangeCard | *models.ChangeCard (status=approved) | NotFoundError, workflow error (wrong status) |
| AdvanceChangeCardStatus | *models.ChangeCard (new status) | NotFoundError, workflow error (terminal) |
| SetChangeCardStatus | *models.ChangeCard (new status) | NotFoundError, workflow error (invalid transition) |
| CountByStatus | map[string]int | DB error |

### 3.4 Error Message Contracts

These error messages are part of the API contract. Downstream consumers (CLI commands) parse or display them.

| Scenario | Error Message Pattern |
|----------|-----------------------|
| Empty title | "change-card title cannot be empty" |
| Non-existent epic link | "epic E99 not found: ..." |
| Non-existent feature link | "feature E07-F99 not found: ..." |
| Task link attempt | "task linking is not supported for change-cards; link to epic or feature instead" |
| Unrecognized key format | "unrecognized entity key format: ... (expected E## for epic or E##-F## for feature)" |
| Approve wrong status | "cannot approve change-card C001: current status is 'in_progress', must be 'proposed'" |
| Advance terminal status | "change-card C001 is in terminal status 'completed'; no further transitions available" |
| Not found | Standard NotFoundError from repository |

---

## 4. Integration Scenarios

### 4.1 Cross-Feature Integration: F01 (Database Schema + Workflow Engine)

**What**: F03 consumes F01's `change_cards` table and `"change"` workflow level.

| Step | Action | Expected | Validates |
|------|--------|----------|-----------|
| 1 | F01 creates `change_cards` table with schema | Table exists with all expected columns and indexes | Data design matches schema |
| 2 | F03 repository inserts a change-card | INSERT succeeds; all columns populated correctly | Column types and constraints compatible |
| 3 | F01 registers `"change"` workflow level | `workflowSvc.ForLevel("change")` returns scoped service | Workflow integration works |
| 4 | F03 service uses scoped workflow for transitions | `ValidateTransition("proposed", "approved")` succeeds | Workflow definitions match expected flow |
| 5 | `updated_at` trigger fires on UPDATE | Update a change-card; verify updated_at changes | F01 trigger works for change_cards |
| 6 | UNIQUE constraint on key column | Attempt duplicate key insert | Constraint error returned |

### 4.2 Cross-Feature Integration: Existing Services (Notes + Context)

**What**: Change-cards integrate with existing NoteService and ContextService.

| Step | Action | Expected | Validates |
|------|--------|----------|-----------|
| 1 | Add EntityTypeChange="change" to models | Constant available in ValidEntityTypes map | Entity type registration |
| 2 | NoteService validates change-card existence | Create note with entity_type="change", entity_key="C001" | NoteService dispatches to ChangeCardRepository for validation |
| 3 | ContextService sets/gets context for change-card | SetField and GetField with entity_type="change" | ContextService switch-case handles "change" |
| 4 | services_global.go wires ChangeCardRepository into NoteService and ContextService | GetNoteService() and GetContextService() include changeCardRepo | Dependency injection complete |

### 4.3 Cross-Feature Integration: F02 (Bug Entity -- Parallel Development)

**What**: F02 and F03 both modify shared files (entity_note.go, services_global.go, workflow levels).

| Risk | Mitigation | Test |
|------|------------|------|
| Both add entity type constants to entity_note.go | Coordinate: F02 adds "bug", F03 adds "change" | Verify both constants exist after merge |
| Both add workflow level constants | Coordinate: F02 adds LevelBug, F03 adds LevelChange | Verify both levels registered |
| Both modify NoteService/ContextService constructors | Constructors accept both bugRepo and changeCardRepo | Verify all entity types handled in switch-cases |
| Both add service accessor to services_global.go | Non-conflicting additions | GetBugService() and GetChangeCardService() both work |

### 4.4 Pattern Parity with Bug Entity (F02)

**What**: F03 must follow identical architectural patterns as F02 for consistency.

| Pattern | F02 (Bug) | F03 (Change-Card) | Check |
|---------|-----------|-------------------|-------|
| Model file location | `internal/models/bug.go` | `internal/models/change_card.go` | Consistent naming |
| Validate() checks | Empty title, empty status, no workflow imports | Same checks | Same structural validation scope |
| Repository interface location | `internal/services/bug_service.go` | `internal/services/change_card_service.go` | Consumer-side definition |
| Key format | B### | C### | Same zero-padded pattern |
| Dual-key lookup | Numeric + slugged | Numeric + slugged | Same strategy |
| Service constructor | NewBugService(db, repo, workflowSvc, epicRepo, featureRepo, projectRoot) | NewChangeCardService(db, repo, workflowSvc, epicRepo, featureRepo, projectRoot) | Same dependency shape |
| Workflow scoping | ForLevel("bug") | ForLevel("change") | Different level, same mechanism |
| File path | docs/bugs/B###.md | docs/changes/C###.md | Flat directory, standalone entity |
| Atomic create | DB insert + file write in transaction | Same pattern | Same fileops usage |

---

## 5. Non-Functional Test Approach

### 5.1 Performance

| Metric | Target | Measurement | Test |
|--------|--------|-------------|------|
| CreateChangeCard latency | < 500ms local | `time` around service call | Create 5 change-cards, all under 500ms |
| ListChangeCards (1000 records) | < 1s | `time` around service call | Seed 1000 records, measure list time |

**Note**: Performance tests are run as part of repository integration tests, not as separate performance suite. They validate UAT Metric 4 (creation speed) from the E18 UAT Acceptance Plan.

### 5.2 Architecture Compliance (Code Review Checklist)

| Check | What to Verify | Pass Criteria |
|-------|---------------|---------------|
| No business logic in repository | Review change_card_repository.go | Only SELECT/INSERT/UPDATE/DELETE queries |
| No workflow imports in model | Review change_card.go imports | No `internal/workflow` or `internal/config` imports |
| No hardcoded statuses in service | Grep for "proposed", "approved", "declined" as string literals in service | All status values come from workflowSvc or method parameters |
| Service uses workflowSvc for all transitions | Review all status transition methods | Every transition calls ValidateTransition or GetNextStatus |
| Constructor validates required deps | Review NewChangeCardService | Panics on nil db, nil repo, nil workflowSvc |
| Context as first parameter | Review all service methods | All accept context.Context as first param |
| Error wrapping with business context | Review all error returns | All errors wrapped with fmt.Errorf and entity key |

### 5.3 Test Coverage

| Layer | Target | Measurement |
|-------|--------|-------------|
| Model (change_card.go) | 100% | `go test -cover ./internal/models/` |
| Repository (change_card_repository.go) | 80%+ | `go test -cover ./internal/repository/` |
| Service (change_card_service.go) | 80%+ | `go test -cover ./internal/services/` |
| Error paths | 100% | All error scenarios in AC matrix have test cases |

---

## UAT Scenario Traceability

This section maps feature-level tests back to E18 UAT acceptance scenarios.

| UAT Scenario | Feature Tests That Validate | Coverage |
|-------------|---------------------------|----------|
| **J2-HP** (Change-Card Happy Path: propose through completion) | AC1.1-AC1.6 (create), AC3.1 (approve), AC4.1+AC4.3 (advance through workflow) | Full lifecycle covered at service layer |
| **J2-ALT-A** (Change-Card Declined) | AC5.1 (set to declined), AC5.3 (terminal after decline) | Decline path covered |
| **J2-ALT-B** (Promotion to Feature -- manual) | AC8.2 (notes for rationale) | Notes support enables rationale; manual workflow validated at CLI level (F05) |
| **J2-ERR-1** (Approve already-approved) | AC3.2 (approve from wrong status) | Wrong-status approval error covered |
| **J2-ERR-2** (Decline from non-proposed) | AC5.2 (invalid transition rejected) | Workflow enforcement covered |
| **CE-2** (Workflow Engine Extension) | AC1.2 (default status from workflow), AC3.3 (workflow validation), AC4.1 (next status from workflow) | All transitions via workflowSvc |
| **CE-3** (Service Pattern Compliance) | Section 5.2 Architecture Compliance checklist | Constructor injection, interface-based repos, no business logic in CLI |

---

## Test Execution Order

Tests should be executed in dependency order:

1. **Model tests** (no dependencies) -- validates structural validation works
2. **Repository tests** (depends on F01 schema) -- validates CRUD against real database
3. **Service tests** (depends on model + repo interfaces) -- validates business logic with mocks
4. **Integration checks** (depends on all layers) -- validates cross-feature wiring
5. **Architecture compliance** (code review) -- validates structural quality

---

## Exit Gate Checklist

| Gate | Criteria | Status |
|------|----------|--------|
| Every acceptance criterion has a test case | All 7 stories (AC1-AC8) have test cases with inputs and expected outputs | PASS |
| API contracts tested | Constructor, DTO, return value, and error message contracts defined in Section 3 | PASS |
| Integration points identified | 4 integration scenarios: F01, Notes/Context, F02 coordination, pattern parity | PASS |
| Test cases are actionable for TDD | Each test specifies exact input, mock setup, and expected output | PASS |
| UAT traceability | All relevant J2 and CE scenarios mapped to feature tests | PASS |
| Architecture compliance criteria defined | Code review checklist in Section 5.2 | PASS |

---

*This test plan is ready for task generation. Tests should be implemented TDD-style: write test first, then implement the code to make it pass.*
