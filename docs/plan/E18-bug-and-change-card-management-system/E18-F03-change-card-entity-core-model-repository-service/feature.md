---
feature_key: E18-F03-change-card-entity-core-model-repository-service
epic_key: E18
title: Change-Card Entity Core (Model, Repository, Service)
description: Implement the ChangeCard model, ChangeCardRepository, and ChangeCardService following E15 service layer patterns, including approval logic, link validation, and workflow integration.
---

# Change-Card Entity Core (Model, Repository, Service)

**Feature Key**: E18-F03
**Execution Order**: 3 (depends on F01; can be developed in parallel with F02)

---

## Epic

- **Epic PRD**: [Bug and Change-Card Management System](../epic.md)
- **Epic Requirements**: [Requirements](../requirements.md)
- **Epic User Journeys**: [User Journeys](../user-journeys.md) -- Journey 2 (Change-Card Lifecycle)
- **Epic Scope**: [Scope](../scope.md)

---

## Goal

### Problem

With the database schema and workflow engine in place (F01), there is no application-layer code for change-card entities. Without a ChangeCard model, repository, and service, the CLI commands (F05), unified integration (F06), and dashboard (F07) have no backend to call. The service layer must encapsulate all change-card business logic -- key generation, link validation, approval operations, and workflow transitions -- so that downstream features are thin wrappers.

### Solution

Implement the full Change-Card entity stack following E15 service layer patterns:
- **Model** (`internal/models/change_card.go`): ChangeCard struct with key validation, status enum, linked entity fields, and structural validation.
- **Repository** (`internal/repository/change_card_repository.go`): Pure CRUD operations plus link-based filtering queries. Repository interface defined for testability.
- **Service** (`internal/services/change_card_service.go`): All business logic including change-card creation with key generation, link validation, approval operation, workflow transitions, and markdown file generation.
- **Service DTOs** (`internal/services/change_dto.go`): CreateChangeCardInput, ChangeCardFilters structs.
- **Markdown template** (`templates/change_card.md.tmpl`): Change-card template with frontmatter and body sections.

### Impact

- The lightweight enhancement tracking path becomes functional at the service layer
- Approval operation is encapsulated as a single atomic service call, ready for CLI and API consumers
- Pattern parity with BugService (F02) ensures consistent architecture across both new entity types

---

## User Personas

Personas are defined at epic level in [personas.md](../personas.md). This feature is consumed by:
- **Developer** -- proposes change-cards, implements approved ones
- **Product Owner** -- reviews and approves/declines change-cards
- **QA Engineer** -- verifies completed change-cards (indirectly, via status queries)

No new personas are introduced by this feature.

---

## User Stories

### Must-Have Stories

**Story 1**: As a **service consumer** (CLI command or HTTP handler), I want to call `ChangeCardService.CreateChangeCard()` with a title and optional link, so that a change-card is created with an auto-generated C### key, persisted in the database, and a markdown file is generated at the correct path.

**Acceptance Criteria**:
- [ ] `CreateChangeCard(ctx, CreateChangeCardInput{Title: "Add dark mode"})` returns a `*models.ChangeCard` with a key in format `C###` (e.g., C001)
- [ ] The returned change-card has status `proposed` (the default initial status from the change workflow)
- [ ] The key is globally unique and auto-incremented from the current maximum key in the database
- [ ] A slug is auto-generated from the title (e.g., "add-dark-mode")
- [ ] A markdown file is created at `docs/changes/C###.md` using the change-card template
- [ ] The markdown file contains frontmatter with key, title, status, and linked entity fields
- [ ] If file creation fails after database insert, the database record is rolled back (atomic operation)

**Story 2**: As a **service consumer**, I want to call `ChangeCardService.CreateChangeCard()` with a `--link` parameter that references an epic or feature key, so that the change-card is linked to the correct parent entity with validated references.

**Acceptance Criteria**:
- [ ] `CreateChangeCardInput{Title: "...", LinkedEntityKey: "E07"}` creates a change-card linked to epic E07
- [ ] `CreateChangeCardInput{Title: "...", LinkedEntityKey: "E07-F01"}` creates a change-card linked to feature E07-F01
- [ ] If the linked entity does not exist, the service returns an error and does not create the change-card
- [ ] If no link is provided, `LinkedEntityType` and `LinkedEntityKey` are empty (both fields are optional)
- [ ] The service detects entity type (epic vs. feature) from the key format automatically

**Story 3**: As a **service consumer**, I want to call `ChangeCardService.ApproveChangeCard()` with a change-card key, so that the change-card advances from `proposed` to `approved` status in a single operation.

**Acceptance Criteria**:
- [ ] `ApproveChangeCard(ctx, "C001")` advances the change-card from `proposed` to `approved`
- [ ] If the change-card is not in `proposed` status, the service returns an error with the current status
- [ ] The workflow transition is validated via `workflowSvc` (not hardcoded status checks)
- [ ] Status history is recorded for audit trail

**Story 4**: As a **service consumer**, I want to call `ChangeCardService.AdvanceChangeCardStatus()` with a change-card key, so that the change-card moves to its next valid workflow status.

**Acceptance Criteria**:
- [ ] `AdvanceChangeCardStatus(ctx, "C001")` advances from the current status to the next status in the change workflow
- [ ] The next status is determined by the workflow service, not hardcoded
- [ ] If the change-card is in a terminal status (`completed`, `declined`), the service returns an error
- [ ] The full workflow path is: `proposed -> approved -> in_progress -> completed`

**Story 5**: As a **service consumer**, I want to call `ChangeCardService.SetChangeCardStatus()` with a change-card key and a target status, so that the change-card status can be set directly (e.g., to `declined`).

**Acceptance Criteria**:
- [ ] `SetChangeCardStatus(ctx, "C001", "declined")` sets status to `declined`
- [ ] The transition is validated via `workflowSvc` (only valid transitions are allowed)
- [ ] `declined` is valid from `proposed` status
- [ ] Status history is recorded for audit trail

**Story 6**: As a **service consumer**, I want to call `ChangeCardService.ListChangeCards()` with optional filters, so that I can retrieve change-cards filtered by status and/or linked entity.

**Acceptance Criteria**:
- [ ] `ListChangeCards(ctx, ChangeCardFilters{})` returns all change-cards
- [ ] `ListChangeCards(ctx, ChangeCardFilters{Status: "proposed"})` returns only change-cards in `proposed` status
- [ ] `ListChangeCards(ctx, ChangeCardFilters{LinkedEntityKey: "E07"})` returns only change-cards linked to epic E07
- [ ] `ListChangeCards(ctx, ChangeCardFilters{ShowAll: false})` excludes completed and declined change-cards (default behavior)
- [ ] `ListChangeCards(ctx, ChangeCardFilters{ShowAll: true})` returns all change-cards including terminal statuses
- [ ] Results are sorted by creation date (newest first) by default

**Story 7**: As a **service consumer**, I want to call `ChangeCardService.GetChangeCard()`, `UpdateChangeCard()`, and `DeleteChangeCard()`, so that standard CRUD operations are available for change-cards.

**Acceptance Criteria**:
- [ ] `GetChangeCard(ctx, "C001")` returns the change-card or a NotFoundError
- [ ] `GetChangeCard` accepts both numeric (`C001`) and slugged (`C001-add-dark-mode`) key formats
- [ ] `UpdateChangeCard(ctx, "C001", ChangeCardUpdates{Title: ptr("New Title")})` updates only the specified fields
- [ ] `DeleteChangeCard(ctx, "C001")` removes the change-card from the database and deletes the markdown file
- [ ] Delete is idempotent -- deleting a non-existent change-card returns NotFoundError, not a panic

---

### Should-Have Stories

**Story 8**: As a **service consumer**, I want the ChangeCard entity to support notes via the existing EntityNoteRepository with `entity_type = "change"`, so that approval rationale and progress updates can be attached.

**Acceptance Criteria**:
- [ ] The entity type constant `EntityTypeChange = "change"` is defined in the models package
- [ ] The NoteService and ContextService accept `"change"` as a valid entity type
- [ ] Notes created with entity_type "change" are retrievable by change-card ID

---

### Edge Case & Error Stories

**Error Story 1**: As a **service consumer**, when I attempt to create a change-card with a link to a non-existent entity (e.g., `--link=E99`), I want to receive a clear validation error so that I know the link target does not exist.

**Acceptance Criteria**:
- [ ] Error message includes the invalid key and entity type (e.g., "epic E99 not found")
- [ ] No database record is created
- [ ] No markdown file is created

**Error Story 2**: As a **service consumer**, when I attempt to approve a change-card that is not in `proposed` status, I want to receive a clear error so that I understand the workflow constraint.

**Acceptance Criteria**:
- [ ] Error message includes the current status (e.g., "cannot approve change-card C001: current status is 'in_progress', must be 'proposed'")
- [ ] The change-card status is not modified

**Error Story 3**: As a **service consumer**, when I attempt to advance a change-card that is in a terminal status (`completed` or `declined`), I want to receive a clear error so that I know no further transitions are available.

**Acceptance Criteria**:
- [ ] Error message indicates the status is terminal (e.g., "change-card C001 is in terminal status 'completed'; no further transitions available")

**Error Story 4**: As a **service consumer**, when concurrent create operations race to generate the next C### key, I want key generation to be safe from duplicates.

**Acceptance Criteria**:
- [ ] Key generation uses a database-level mechanism (MAX query + UNIQUE constraint) that prevents duplicate keys
- [ ] If a UNIQUE constraint violation occurs, the service retries with the next available key or returns a clear error

---

## Functional Requirements

### Category: ChangeCard Model

**F03-REQ-001**: ChangeCard Struct Definition
- **Description**: The ChangeCard model struct must contain fields: ID (int64), Key (string), Title (string), Status (ChangeCardStatus), Slug (string), LinkedEntityType (string, nullable), LinkedEntityKey (string, nullable), FilePath (string), CreatedAt (time.Time), UpdatedAt (time.Time).
- **Traces to**: Epic REQ-F-007
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Struct defined in `internal/models/change_card.go`
  - [ ] JSON tags match existing entity conventions (snake_case)
  - [ ] ChangeCardStatus type alias defined with constants for workflow statuses

**F03-REQ-002**: ChangeCard Structural Validation
- **Description**: The `Validate()` method must check: title is non-empty (after trimming), status is non-empty. Validate() must NOT check status validity against the workflow -- that is the service layer's responsibility.
- **Traces to**: Epic REQ-NF-005 (consistency with existing model validation patterns)
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `Validate()` returns error for empty title
  - [ ] `Validate()` returns error for empty status
  - [ ] `Validate()` does NOT import workflow or config packages
  - [ ] `Validate()` does NOT hardcode status values

**F03-REQ-003**: ChangeCard Key Format
- **Description**: Change-card keys must follow the format `C###` where `###` is a zero-padded three-digit number (e.g., C001, C042). Keys are globally unique and never reused.
- **Traces to**: Epic REQ-F-007, REQ-NF-004
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Key regex pattern: `^C\d{3}$`
  - [ ] Keys auto-increment from the current maximum
  - [ ] Deleted keys are never reused

### Category: ChangeCard Repository

**F03-REQ-004**: Repository Interface
- **Description**: The ChangeCardRepository interface must define: Create, GetByKey, GetByID, List, Update, Delete, UpdateStatus, CountByStatus, ListByLinkedEntity.
- **Traces to**: Epic REQ-F-007, REQ-F-011
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Interface defined in `internal/services/change_card_service.go` (consumer-side per project conventions)
  - [ ] All methods accept `context.Context` as first parameter
  - [ ] Concrete implementation in `internal/repository/change_card_repository.go`

**F03-REQ-005**: Key Auto-Generation in Repository
- **Description**: The Create method must auto-generate the next available C### key by querying `SELECT MAX(CAST(SUBSTR(key, 2) AS INTEGER)) FROM change_cards` and incrementing.
- **Traces to**: Epic REQ-F-007, REQ-NF-004
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] First change-card created gets key C001
  - [ ] Subsequent change-cards get monotonically increasing keys
  - [ ] If all change-cards are deleted, the next key is still higher than any previously used key (AUTOINCREMENT behavior)

**F03-REQ-006**: Dual Key Lookup
- **Description**: `GetByKey` must support both numeric (`C001`) and slugged (`C001-add-dark-mode`) key formats, following the same dual-key lookup strategy used for tasks.
- **Traces to**: Epic REQ-NF-005 (CLI pattern consistency)
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `GetByKey(ctx, "C001")` returns the change-card
  - [ ] `GetByKey(ctx, "C001-add-dark-mode")` returns the same change-card (if slug matches)
  - [ ] `GetByKey(ctx, "C999")` returns NotFoundError
  - [ ] Key lookup is case-insensitive

**F03-REQ-007**: Linked Entity Filtering
- **Description**: `ListByLinkedEntity` must return change-cards filtered by linked entity type and key.
- **Traces to**: Epic REQ-F-018 (list filtering by linked entity)
- **Priority**: Should-Have
- **Acceptance Criteria**:
  - [ ] `ListByLinkedEntity(ctx, "epic", "E07")` returns change-cards linked to epic E07
  - [ ] `ListByLinkedEntity(ctx, "feature", "E07-F01")` returns change-cards linked to feature E07-F01

### Category: ChangeCard Service

**F03-REQ-008**: CreateChangeCard with Atomic File+DB
- **Description**: The `CreateChangeCard` method must atomically create both the database record and the markdown file. If either operation fails, neither persists.
- **Traces to**: Epic REQ-NF-003
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Uses the `fileops.NewEntityFileWriter()` pattern for atomic file creation
  - [ ] If file creation fails, no database record exists
  - [ ] If database insert fails, no orphaned file exists
  - [ ] File is created at `docs/changes/C###.md`

**F03-REQ-009**: Link Validation
- **Description**: When `LinkedEntityKey` is provided, the service must validate the entity exists by calling the appropriate repository (EpicRepository for `E##` keys, FeatureRepository for `E##-F##` keys). Task linking is not supported for change-cards (per epic REQ-F-008).
- **Traces to**: Epic REQ-F-008
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] `LinkedEntityKey: "E07"` triggers EpicRepository.GetByKey validation
  - [ ] `LinkedEntityKey: "E07-F01"` triggers FeatureRepository.GetByKey validation
  - [ ] `LinkedEntityKey: "E07-F01-001"` returns an error (task linking not supported for change-cards)
  - [ ] Invalid entity format returns a clear error message

**F03-REQ-010**: ApproveChangeCard
- **Description**: The `ApproveChangeCard` method must be a convenience method that validates the change-card is in `proposed` status and advances it to `approved`.
- **Traces to**: Epic REQ-F-010
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Only succeeds when current status is `proposed`
  - [ ] Uses workflow service for transition validation (not hardcoded)
  - [ ] Records status change in history

**F03-REQ-011**: Workflow Integration
- **Description**: All status transitions must be validated via `workflowSvc.ForLevel(workflow.LevelChange)`. The service must not hardcode status values or transition rules.
- **Traces to**: Epic REQ-F-009, REQ-NF-006
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Service constructor calls `workflowSvc.ForLevel(workflow.LevelChange)` to scope the workflow
  - [ ] `AdvanceChangeCardStatus` delegates to workflow service for next-status determination
  - [ ] `SetChangeCardStatus` delegates to workflow service for transition validation
  - [ ] Default initial status is read from workflow service, not hardcoded

**F03-REQ-012**: Service Accessor
- **Description**: A `GetChangeCardService()` function must be added to `internal/cli/services_global.go` that constructs a ChangeCardService with all required dependencies.
- **Traces to**: Architecture pattern from E15
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Function follows the lazy initialization pattern matching `GetTaskService()`
  - [ ] Dependencies: ChangeCardRepository, workflow.Service, EpicRepository (for link validation), FeatureRepository (for link validation)
  - [ ] Creates a new instance per call (stateless)

### Category: Markdown Template

**F03-REQ-013**: Change-Card Markdown Template
- **Description**: A template for change-card markdown files with frontmatter and body sections.
- **Traces to**: Epic REQ-F-007 (file generation)
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Frontmatter includes: change_card_key, title, status, linked_entity_type, linked_entity_key
  - [ ] Body sections: Description, Justification, Linked Entity
  - [ ] Template follows existing template patterns in `templates/` package

### Category: Tests

**F03-REQ-014**: Model Validation Tests
- **Description**: Unit tests for ChangeCard.Validate() covering valid, empty title, and empty status cases.
- **Traces to**: Testing architecture
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Tests in `internal/models/change_card_test.go`
  - [ ] Table-driven tests covering valid input, empty title, empty status

**F03-REQ-015**: Repository Tests (Real DB)
- **Description**: Repository tests using real database with cleanup, covering CRUD, key auto-generation, dual-key lookup, and linked entity filtering.
- **Traces to**: Testing architecture
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Tests in `internal/repository/change_card_repository_test.go`
  - [ ] Uses `test.GetTestDB()` with cleanup before each test
  - [ ] Tests: Create, GetByKey, GetByID, List, Update, Delete, UpdateStatus, CountByStatus, ListByLinkedEntity
  - [ ] Tests key auto-generation (first key is C001, next is C002)
  - [ ] Tests dual-key lookup (numeric and slugged)

**F03-REQ-016**: Service Tests (Mocked Repos)
- **Description**: Service tests using mocked repositories, covering create, approve, advance status, set status, list with filters, link validation, and error cases.
- **Traces to**: Testing architecture
- **Priority**: Must-Have
- **Acceptance Criteria**:
  - [ ] Tests in `internal/services/change_card_service_test.go`
  - [ ] Mock ChangeCardRepository using function-field pattern
  - [ ] Mock workflow service
  - [ ] Tests: CreateChangeCard (happy path), CreateChangeCard (invalid link), ApproveChangeCard (happy path), ApproveChangeCard (wrong status), AdvanceChangeCardStatus, SetChangeCardStatus (valid and invalid transitions), ListChangeCards (with filters), GetChangeCard, DeleteChangeCard

---

## Non-Functional Requirements

**F03-NFR-001**: Creation Performance
- **Description**: `CreateChangeCard` must complete in under 500ms on local SQLite, including file generation.
- **Measurement**: Wall-clock time from service call to return
- **Target**: < 500ms local, < 2s Turso cloud
- **Traces to**: Epic REQ-NF-001

**F03-NFR-002**: List Performance
- **Description**: `ListChangeCards` must return results in under 1 second with up to 1000 change-cards.
- **Measurement**: Wall-clock time from service call to return
- **Target**: < 1s for 1000 entities
- **Traces to**: Epic REQ-NF-002

**F03-NFR-003**: Architecture Compliance
- **Description**: All code must follow the project's layered architecture: model (structural validation only) -> repository (pure CRUD) -> service (business logic). No business logic in repository. No workflow hardcoding in model.
- **Measurement**: Code review against `.claude/rules/architecture.md` and `.claude/rules/services/service-design.md`
- **Target**: Zero violations
- **Traces to**: Epic REQ-NF-005

**F03-NFR-004**: Test Coverage
- **Description**: Service layer must have 80%+ code coverage. All error paths must be tested.
- **Measurement**: `go test -cover`
- **Target**: >= 80% service coverage, 100% error path coverage
- **Traces to**: Testing architecture

---

## Feature-Level Acceptance Criteria

**Scenario 1: Create Change-Card (Happy Path)**
- **Given** the change_cards table exists and the change workflow is configured
- **When** `CreateChangeCard(ctx, CreateChangeCardInput{Title: "Add dark mode toggle"})` is called
- **Then** a change-card with key `C001` (or next available) is returned with status `proposed`
- **And** a markdown file exists at `docs/changes/C001.md` with correct frontmatter
- **And** the change-card is retrievable via `GetChangeCard(ctx, "C001")`

**Scenario 2: Create Change-Card with Link**
- **Given** epic E07 exists in the database
- **When** `CreateChangeCard(ctx, CreateChangeCardInput{Title: "Improve auth UX", LinkedEntityKey: "E07"})` is called
- **Then** a change-card is returned with `LinkedEntityType: "epic"` and `LinkedEntityKey: "E07"`
- **And** the markdown file frontmatter includes `linked_entity_type: epic` and `linked_entity_key: E07`

**Scenario 3: Create Change-Card with Invalid Link**
- **Given** epic E99 does not exist in the database
- **When** `CreateChangeCard(ctx, CreateChangeCardInput{Title: "...", LinkedEntityKey: "E99"})` is called
- **Then** an error is returned containing "epic E99 not found"
- **And** no database record is created
- **And** no markdown file is created

**Scenario 4: Approve Change-Card**
- **Given** change-card C001 exists with status `proposed`
- **When** `ApproveChangeCard(ctx, "C001")` is called
- **Then** the change-card status is updated to `approved`
- **And** status history records the transition from `proposed` to `approved`

**Scenario 5: Approve Change-Card in Wrong Status**
- **Given** change-card C001 exists with status `in_progress`
- **When** `ApproveChangeCard(ctx, "C001")` is called
- **Then** an error is returned indicating the current status is `in_progress` and approval requires `proposed`
- **And** the change-card status is not modified

**Scenario 6: Advance Through Full Workflow**
- **Given** change-card C001 exists with status `proposed`
- **When** `ApproveChangeCard` is called, then `AdvanceChangeCardStatus` twice
- **Then** the status transitions are: `proposed -> approved -> in_progress -> completed`
- **And** each transition is recorded in status history

**Scenario 7: Decline Change-Card**
- **Given** change-card C001 exists with status `proposed`
- **When** `SetChangeCardStatus(ctx, "C001", "declined")` is called
- **Then** the change-card status is updated to `declined`
- **And** further advance operations return a terminal-status error

**Scenario 8: List with Filters**
- **Given** change-cards exist: C001 (proposed, linked to E07), C002 (approved, no link), C003 (completed, linked to E07)
- **When** `ListChangeCards(ctx, ChangeCardFilters{LinkedEntityKey: "E07", ShowAll: false})` is called
- **Then** only C001 is returned (C003 is excluded because it is in terminal status and ShowAll is false)

**Scenario 9: Dual Key Lookup**
- **Given** change-card C001 exists with slug `add-dark-mode-toggle`
- **When** `GetChangeCard(ctx, "C001-add-dark-mode-toggle")` is called
- **Then** the change-card is returned successfully

**Scenario 10: Atomic File+DB Failure**
- **Given** the file system is read-only for the target path
- **When** `CreateChangeCard` is called
- **Then** no database record is created (rollback)
- **And** the error message indicates the file creation failure

---

## Out of Scope

### Explicitly Excluded

1. **CLI Commands** -- All `shark change` CLI commands are in F05. This feature provides only the service layer that F05 will call.
2. **Unified CLI Integration** -- Key auto-detection for C### in `shark get`, `shark status`, etc. is in F06.
3. **Dashboard and Analytics** -- Change-card counts and metrics in `shark status` and `shark analytics` are in F07.
4. **Change-Card Promotion to Feature** -- Epic REQ-F-020 (Could Have) is deferred to a follow-on epic.
5. **Task Linking** -- Change-cards link to epics and features only (per epic REQ-F-008). Task-level linking is not supported for change-cards.
6. **Priority or Effort Fields** -- Change-cards intentionally have no priority or effort fields. They are lightweight items. If more granularity is needed, the change-card should be promoted to a feature.

---

## Dependencies & Integrations

### Dependencies

- **E18-F01 (Database Schema and Workflow Engine Extension)**: Requires the `change_cards` table to exist and the `change` workflow level to be registered in the workflow engine. This feature cannot function without F01.
- **E15 (Service Layer Architecture)**: This feature follows established service patterns. The `workflow.Service`, `fileops` package, and repository interface patterns must be in place.
- **Existing Repositories**: Link validation requires `EpicRepository.GetByKey()` and `FeatureRepository.GetByKey()` to be available.

### Integration Points

- **EntityNoteRepository**: The existing notes system must accept `entity_type = "change"` for change-card notes. No code changes are expected in EntityNoteRepository itself -- it already supports arbitrary entity types.
- **ContextService**: Must accept "change" as a valid entity type for context field operations.
- **ResumeService**: Should be extensible to support "change" entity type (future, not required for this feature).

---

## Open Questions & Assumptions

No open questions -- all requirements are clear.

**Resolved Assumptions**:
1. Change-card keys use three-digit zero-padded format (`C001` through `C999`). If more than 999 change-cards are needed, key format expansion is a future concern.
2. The change workflow statuses (`proposed`, `approved`, `in_progress`, `completed`, `declined`) are defined in F01's workflow engine extension. This feature consumes them, not defines them.
3. The ChangeCardService constructor signature follows the same pattern as TaskService: `NewChangeCardService(repo, workflowSvc, epicRepo, featureRepo)`.
4. File path for change-card markdown is `docs/changes/C###.md` (flat directory, not nested under epics/features, since change-cards are standalone entities).

---

## Sprint Sizing

**Estimate**: 1 sprint (S-M complexity)

- Simpler than F02 (no severity field, shorter workflow, fewer filter dimensions)
- Pattern is identical to F02; can be developed in parallel
- Model: Small
- Repository: Small-Medium
- Service: Medium (approval logic, link validation)
- Tests: Small-Medium

---

*Last Updated*: 2026-03-03
