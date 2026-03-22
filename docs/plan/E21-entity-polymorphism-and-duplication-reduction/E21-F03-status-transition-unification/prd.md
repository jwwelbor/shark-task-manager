# F03: Status Transition Unification

**Feature Key**: E21-F03
**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Complexity Tier**: STANDARD (score 12/27)
**Execution Order**: 3 (depends on F01 completion; parallel with F02)
**Effort Estimate**: L (5-8 tasks)
**Risk**: Medium (TaskService's atomic `StatusUpdateRaw` diverges significantly from Epic/Feature's simpler pattern)

---

## Epic

- **Epic PRD**: [Entity Polymorphism and Duplication Reduction](../epic.md)
- **Epic Architecture**: [Architecture Design](../architecture-design.md)

---

## Goal

### Problem

Five entity services each implement their own status transition logic. Epic and Feature have near-identical ~80-line `TransitionStatus` methods that perform the same 10 steps: get entity, extract current status, validate transition, normalize target, enforce force-reason, detect backward transition, require backward reason, update status, create rejection note, count children, and resolve orchestrator action. Bug and ChangeCard have simpler `SetXxxStatus`/`AdvanceXxxStatus` methods (~25 lines each) that share the validation-update core but skip backward detection, rejection notes, and child counting. Additionally, `resolveAction` (~40 lines) and `GetNextStatus` (~25 lines) are duplicated across entity services with structurally identical implementations that differ only in entity type strings and placeholder functions.

TaskService is the exception: its `TransitionStatus` delegates to an internal `executeStatusTransition` method that calls the repository's atomic `StatusUpdateRaw` (handling agent/notes/timestamps/auto-unblocking in a single DB operation), then separately builds the TransitionResult. This is a fundamentally different update mechanism from Epic/Feature (which call `repo.Update` on the full entity) and Bug/ChangeCard (which call `repo.UpdateStatus`).

A bug fix to transition validation logic (e.g., backward detection) must currently be applied in 3 places (Epic, Feature, Task). A bug fix to `resolveAction` must be applied in 5 places.

### Solution

Create a shared `EntityService` with a `TransitionStatus` method that implements the common transition algorithm once. Entity-specific services compose `EntityService` and delegate the shared steps to it, adding only entity-specific pre/post hooks (Task: auto-unblocking and feature progress recalculation; Epic: child feature count; Feature: child task count; Bug/ChangeCard: opt-out of backward detection and rejection notes). Unify `resolveAction` into `EntityService` using the Entity interface for base placeholder generation, with entity-specific placeholder extension points. Unify `GetNextStatus` into `EntityService` as well.

The key design decision: TaskService retains its `StatusUpdateRaw` call path inside its entity-specific hook, while the shared logic handles validation, backward detection, reason enforcement, rejection notes, and orchestrator action resolution. The `EntityRepository.UpdateStatus` adapter abstracts the status persistence mechanism per entity type for non-Task entities.

### Impact

- Eliminate ~400 lines of duplicated transition logic across 5 entity services
- Eliminate ~200 lines of duplicated `resolveAction` implementations
- Eliminate ~125 lines of duplicated `GetNextStatus` implementations
- Bug fixes to transition validation, backward detection, and orchestrator action resolution apply once instead of 3-5 times
- Adding a 6th entity type's status transition requires composing `EntityService` and adding only entity-specific hooks (~20 lines vs ~120 lines)

---

## Key Personas

From the [epic personas](../personas.md):

- **Codebase Maintainer**: Directly benefits from single-location transition logic. Bug fixes to backward detection, reason enforcement, and rejection notes apply once. The current risk of inconsistent behavior between Epic and Feature transition implementations (already observed: slightly different nil-check patterns) is eliminated.
- **New Entity Author**: A new entity type gets full transition support (validation, backward detection, rejection notes, orchestrator actions) by composing `EntityService` with 2-3 lines of delegation code, instead of copying ~120 lines.
- **Service Layer Consumer**: The `TransitionResult` type remains unchanged. CLI commands and HTTP handlers calling `TransitionStatus` see no behavioral change.

---

## User Stories

### Must-Have Stories

**Story 1**: As a codebase maintainer, I want a single `EntityService.TransitionStatus` method that implements the shared transition algorithm (validate, normalize, backward detect, reason enforce, update status, rejection note, resolve action), so that I fix transition bugs in one place instead of three.

**Acceptance Criteria**:
- [ ] `EntityService.TransitionStatus(ctx, repo EntityRepository, entityType string, key string, targetStatus string, opts TransitionOptions, features TransitionFeatures) (*TransitionResult, error)` is implemented
- [ ] The method handles all shared steps: get entity, extract status, validate transition (unless forced), normalize status (unless forced), enforce force-reason, detect backward (opt-in), require backward reason (opt-in), update status via `repo.UpdateStatus`, create rejection note (opt-in), resolve orchestrator action (opt-in)
- [ ] The `TransitionFeatures` struct controls which optional steps are active per entity type
- [ ] All 3 core entity services (EpicService, FeatureService, TaskService) delegate shared transition logic to `EntityService.TransitionStatus`
- [ ] All existing `TransitionStatus` behavior is preserved (verified by passing all existing tests)

**Story 2**: As a codebase maintainer, I want EpicService and FeatureService `TransitionStatus` methods to delegate to `EntityService.TransitionStatus` and add only their entity-specific post-hooks (child counting), so that their implementations are reduced from ~80 lines to ~15 lines each.

**Acceptance Criteria**:
- [ ] EpicService.TransitionStatus calls `EntityService.TransitionStatus` with `DefaultTransitionFeatures()` where `CountChildren = true`
- [ ] EpicService adds child feature count to the TransitionResult in a post-hook
- [ ] FeatureService.TransitionStatus calls `EntityService.TransitionStatus` with `DefaultTransitionFeatures()` where `CountChildren = true`
- [ ] FeatureService adds child task count to the TransitionResult in a post-hook
- [ ] Both services remove their inline transition validation, backward detection, rejection note, and `resolveAction` logic
- [ ] `workflowSvc` and `noteRepo` fields move from EpicService/FeatureService to EntityService (they are no longer direct dependencies of these services for transition purposes)

**Story 3**: As a codebase maintainer, I want TaskService to delegate shared transition logic (validation, backward detection, reason enforcement, rejection notes, orchestrator action) to `EntityService` while retaining its entity-specific `StatusUpdateRaw` call path for atomic task status updates, so that Task transition behavior is preserved exactly.

**Acceptance Criteria**:
- [ ] TaskService.TransitionStatus still calls `executeStatusTransition` for the atomic DB write (agent, notes, timestamps, auto-unblocking via `StatusUpdateRaw`)
- [ ] Shared validation logic (validate transition, normalize status) is delegated to EntityService or reused from the shared implementation
- [ ] Backward detection and reason enforcement use the shared implementation from EntityService
- [ ] Rejection note creation uses the shared implementation from EntityService
- [ ] Orchestrator action resolution uses the shared `ResolveAction` from EntityService
- [ ] Auto-unblocking behavior and feature progress recalculation remain task-specific
- [ ] `StatusUpdateRaw` is not called through the `EntityRepository.UpdateStatus` adapter (Task retains its direct typed repository call for this operation)

**Story 4**: As a codebase maintainer, I want `resolveAction` unified into EntityService using the Entity interface for base placeholder generation, so that the identical ~40-line method is not duplicated across 5 entity services.

**Acceptance Criteria**:
- [ ] `EntityService.ResolveAction(entity models.Entity, status string, extraPlaceholders map[string]string) *config.PopulatedAction` is implemented using `EntityPlaceholders(entity)` for base placeholders
- [ ] Entity-specific services can extend placeholders before calling the shared method (Task adds related docs; Epic adds feature list; etc.)
- [ ] All 5 entity services delegate `resolveAction` calls to the shared implementation
- [ ] Existing orchestrator action resolution produces identical `PopulatedAction` results for all entity types

**Story 5**: As a codebase maintainer, I want `GetNextStatus` unified into EntityService, so that the identical ~25-line method is not duplicated across 3 entity services (Epic, Feature, Task).

**Acceptance Criteria**:
- [ ] `EntityService.GetNextStatus(ctx, repo EntityRepository, entityType string, key string, resolveActionFn func(models.Entity, string) *config.PopulatedAction) (*NextStatusInfo, error)` is implemented
- [ ] Entity-specific services delegate `GetNextStatus` calls to the shared implementation
- [ ] The `resolveActionFn` callback allows entity-specific placeholder enrichment per transition target
- [ ] Existing `GetNextStatus` behavior is preserved for all 3 entity types

---

### Should-Have Stories

**Story 6**: As a codebase maintainer, I want Bug and ChangeCard services to optionally delegate their `SetXxxStatus`/`AdvanceXxxStatus` methods to `EntityService.TransitionStatus` using `SimpleTransitionFeatures()`, so that they benefit from shared validation logic without backward detection or rejection notes.

**Acceptance Criteria**:
- [ ] BugService can call `EntityService.TransitionStatus` with `SimpleTransitionFeatures()` for SetBugStatus
- [ ] ChangeCardService can call `EntityService.TransitionStatus` with `SimpleTransitionFeatures()` for SetChangeCardStatus
- [ ] Bug and ChangeCard retain their simpler return types (`*models.Bug`, `*models.ChangeCard`) by reloading the typed entity after the shared transition
- [ ] Existing Bug and ChangeCard status transition behavior is preserved

**Story 7**: As a test author, I want `EntityService.TransitionStatus` tested with parameterized entity types covering all TransitionFeatures combinations, so that shared logic is verified once with comprehensive coverage.

**Acceptance Criteria**:
- [ ] `entity_service_test.go` includes parameterized tests for TransitionStatus with mock EntityRepository
- [ ] Test cases cover: happy path, forced transition, backward transition with reason, backward transition without reason (error), unregistered entity (error), repository error propagation
- [ ] Test cases cover TransitionFeatures permutations: `DefaultTransitionFeatures()`, `SimpleTransitionFeatures()`, custom combinations
- [ ] Coverage for EntityService.TransitionStatus is at least 85%

---

### Edge Case & Error Stories

**Error Story 1**: As a codebase maintainer, when `EntityRepository.UpdateStatus` fails during a transition, I want the error to propagate with full context (entity type, key, from-status, to-status), so that I can diagnose the failure.

**Acceptance Criteria**:
- [ ] Error message includes entity type, entity key, and target status: `"failed to update <type> status: <underlying error>"`
- [ ] The error is returned before any rejection note creation or action resolution (partial state is avoided)

**Error Story 2**: As a codebase maintainer, when TaskService's `StatusUpdateRaw` fails but shared validation has already passed, I want the error to clearly indicate the failure was in the task-specific update path, not in shared validation.

**Acceptance Criteria**:
- [ ] TaskService error messages distinguish between shared validation failures and task-specific update failures
- [ ] Error wrapping preserves the full chain: `"failed to transition task E07-F01-001 to in_progress: failed to update task E07-F01-001 status: <db error>"`

**Error Story 3**: As a codebase maintainer, when `resolveAction` encounters a nil workflow or missing status metadata, I want the shared implementation to return nil gracefully (not panic), so that transitions succeed even without orchestrator configuration.

**Acceptance Criteria**:
- [ ] `EntityService.ResolveAction` returns nil when `workflowSvc.GetWorkflow()` is nil
- [ ] Returns nil when status metadata does not exist for the target status
- [ ] Returns nil when `OrchestratorAction` is nil in the status metadata
- [ ] No panic under any combination of nil inputs

---

## Requirements

### Functional Requirements

**Category: Shared EntityService TransitionStatus**

1. **REQ-F03-001**: EntityService TransitionStatus Method
   - **Description**: Implement `EntityService.TransitionStatus` that performs the shared transition algorithm, controlled by `TransitionFeatures` for opt-in/opt-out of backward detection, rejection notes, child counting, and orchestrator action resolution.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-008
   - **Acceptance Criteria**:
     - [ ] Method signature: `TransitionStatus(ctx, repo EntityRepository, entityType string, key string, targetStatus string, opts TransitionOptions, features TransitionFeatures) (*TransitionResult, error)`
     - [ ] Steps: (1) get entity via repo, (2) extract current status via `entity.GetStatus()`, (3) validate transition unless forced, (4) normalize status unless forced, (5) enforce force-reason, (6) detect backward if `features.DetectBackward`, (7) require backward reason if `features.DetectBackward` and not forced, (8) update status via `repo.UpdateStatus`, (9) create rejection note if `features.CreateRejectionNotes`, (10) resolve orchestrator action if `features.ResolveOrchestratorAction`
     - [ ] Returns `TransitionResult` with EntityType, EntityKey, FromStatus, ToStatus, Transitioned, OrchestratorAction, IsBackward, IsForced, Reason, ChildCount (set by caller)

2. **REQ-F03-002**: TransitionFeatures Configuration
   - **Description**: Define `TransitionFeatures` struct with boolean fields controlling which optional steps are active, with preset functions for common configurations.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-008
   - **Acceptance Criteria**:
     - [ ] `TransitionFeatures` has fields: DetectBackward, CreateRejectionNotes, CountChildren, ResolveOrchestratorAction
     - [ ] `DefaultTransitionFeatures()` returns all-true except CountChildren (used by Epic, Feature, Task)
     - [ ] `SimpleTransitionFeatures()` returns only ResolveOrchestratorAction=true (used by Bug, ChangeCard)

**Category: Epic/Feature Delegation**

3. **REQ-F03-003**: EpicService TransitionStatus Delegation
   - **Description**: Refactor EpicService.TransitionStatus to delegate shared logic to EntityService.TransitionStatus and retain only child feature counting as a post-hook.
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-009
   - **Acceptance Criteria**:
     - [ ] EpicService.TransitionStatus body reduced to ~15 lines (delegation + child count post-hook)
     - [ ] Inline transition validation, backward detection, rejection note creation, and `resolveAction` removed from EpicService
     - [ ] EpicService constructor adds `entitySvc *EntityService` and `entityRepo EntityRepository` as dependencies
     - [ ] `workflowSvc` field optionally retained for non-transition uses (ValidateStatus, GetNextStatus) or also delegated

4. **REQ-F03-004**: FeatureService TransitionStatus Delegation
   - **Description**: Refactor FeatureService.TransitionStatus to delegate shared logic to EntityService.TransitionStatus and retain only child task counting as a post-hook.
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-009
   - **Acceptance Criteria**:
     - [ ] FeatureService.TransitionStatus body reduced to ~15 lines (delegation + child count post-hook)
     - [ ] Inline transition validation, backward detection, rejection note creation, and `resolveAction` removed from FeatureService
     - [ ] FeatureService constructor adds `entitySvc *EntityService` and `entityRepo EntityRepository` as dependencies

**Category: TaskService Delegation (Hybrid Pattern)**

5. **REQ-F03-005**: TaskService Transition Delegation with StatusUpdateRaw Preservation
   - **Description**: Refactor TaskService.TransitionStatus to use shared validation and action resolution from EntityService while retaining `executeStatusTransition` for the atomic `StatusUpdateRaw` call. This is a hybrid delegation where shared logic is reused but the status persistence mechanism remains task-specific.
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-008, REQ-F-009
   - **Acceptance Criteria**:
     - [ ] Shared steps reused from EntityService: validation, normalization, backward detection, reason enforcement
     - [ ] Task-specific steps preserved: `StatusUpdateRaw` (agent, notes, timestamps, auto-unblocking), feature progress recalculation
     - [ ] Rejection note creation uses the shared EntityService implementation
     - [ ] Orchestrator action resolution uses the shared `ResolveAction` from EntityService
     - [ ] TaskService does NOT route status updates through `EntityRepository.UpdateStatus` (retains typed repo call)

**Category: resolveAction Unification**

6. **REQ-F03-006**: Shared resolveAction Implementation
   - **Description**: Implement `EntityService.ResolveAction` using `EntityPlaceholders(entity)` for base template placeholders, with entity-specific extension via extra placeholders map.
   - **User Story**: Story 4
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-011 (partial -- base placeholders)
   - **Acceptance Criteria**:
     - [ ] `EntityService.ResolveAction(entity models.Entity, status string, extraPlaceholders map[string]string) *config.PopulatedAction`
     - [ ] Uses `EntityPlaceholders(entity)` for base placeholder generation (entity_type, entity_key, entity_title, entity_slug, status)
     - [ ] Merges `extraPlaceholders` over base placeholders (entity-specific fields override base)
     - [ ] Entity services pass entity-specific placeholders (Task: related docs, enrichment; Epic: feature list; Feature: task list)
     - [ ] Returns nil gracefully for nil workflow, missing metadata, or nil OrchestratorAction

**Category: GetNextStatus Unification**

7. **REQ-F03-007**: Shared GetNextStatus Implementation
   - **Description**: Implement `EntityService.GetNextStatus` that retrieves available transitions for an entity's current status with orchestrator actions per transition.
   - **User Story**: Story 5
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-008 (implied by shared status operations)
   - **Acceptance Criteria**:
     - [ ] `EntityService.GetNextStatus(ctx, repo EntityRepository, entityType string, key string, resolveActionFn func(models.Entity, string) *config.PopulatedAction) (*NextStatusInfo, error)` is implemented
     - [ ] Gets entity via repo, extracts current status, gets transitions from workflow service, wraps each with action
     - [ ] Entity-specific services pass a `resolveActionFn` that includes their placeholder enrichment
     - [ ] EpicService, FeatureService, TaskService delegate GetNextStatus to the shared implementation

**Category: Bug/ChangeCard Delegation (Optional)**

8. **REQ-F03-008**: Bug/ChangeCard Optional TransitionStatus Delegation
   - **Description**: Optionally refactor Bug and ChangeCard status methods to use `EntityService.TransitionStatus` with `SimpleTransitionFeatures()`.
   - **User Story**: Story 6
   - **Priority**: Should-Have
   - **Traces to**: Epic REQ-F-008
   - **Acceptance Criteria**:
     - [ ] BugService.SetBugStatus can delegate to EntityService.TransitionStatus with SimpleTransitionFeatures
     - [ ] ChangeCardService.SetChangeCardStatus can delegate to EntityService.TransitionStatus with SimpleTransitionFeatures
     - [ ] Both services reload typed entity after shared transition to return `*models.Bug`/`*models.ChangeCard`
     - [ ] Existing behavior preserved (no backward detection, no rejection notes)

**Category: Constructor and Wiring Updates**

9. **REQ-F03-009**: EntityService Constructor and CLI Wiring
   - **Description**: Create EntityService constructor with workflow service and note repository dependencies. Update CLI wiring in `services_global.go` to create EntityService and pass it to entity-specific services.
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Traces to**: Epic REQ-F-008
   - **Acceptance Criteria**:
     - [ ] `NewEntityService(workflowSvc *workflow.Service, noteRepo RejectionNoteCreator) *EntityService`
     - [ ] `services_global.go` creates EntityService with shared workflow and note dependencies
     - [ ] Entity-specific service constructors updated to accept `entitySvc *EntityService` and `entityRepo EntityRepository`
     - [ ] Removed: direct `workflowSvc` and `noteRepo` fields from entity services where they are only used for transition logic

---

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF03-001**: Zero Behavioral Changes
   - **Description**: All existing CLI commands that exercise status transitions must produce identical output after the refactoring. All existing tests must pass without modification (except test infrastructure updates for new constructor signatures).
   - **Measurement**: All existing tests pass; manual verification of `shark status advance` and `shark status set` for all 5 entity types
   - **Target**: 100% test pass rate with zero behavioral deltas
   - **Traces to**: Epic REQ-NF-001

**Performance**

2. **REQ-NF03-002**: No Measurable Transition Latency Increase
   - **Description**: The additional interface dispatch from EntityService delegation must not introduce measurable latency to status transitions.
   - **Measurement**: Benchmark `TransitionStatus` before and after for Task, Epic, Feature
   - **Target**: Less than 1ms additional latency (dominated by DB I/O of 1-10ms)
   - **Traces to**: Epic REQ-NF-003
   - **Justification**: The refactoring adds 1-2 interface method calls (Entity accessor, EntityRepository.UpdateStatus). At ~2ns per dispatch, the overhead is 6 orders of magnitude below DB latency.

**Test Coverage**

3. **REQ-NF03-003**: EntityService Test Coverage
   - **Description**: The shared EntityService must have comprehensive test coverage for TransitionStatus, ResolveAction, and GetNextStatus.
   - **Measurement**: `go test -cover ./internal/services/`
   - **Target**: 85%+ coverage for EntityService methods
   - **Traces to**: Epic REQ-NF-006

**Maintainability**

4. **REQ-NF03-004**: Code Reduction
   - **Description**: The refactoring must achieve a net line reduction in the affected service files.
   - **Measurement**: `wc -l` on `epic_service.go`, `feature_service.go`, `task_service.go`, `bug_service.go`, `change_card_service.go`, and new `entity_service.go`
   - **Measurement formula**: (lines removed from 5 entity services) - (lines added in entity_service.go)
   - **Target**: Net reduction of at least 300 lines (conservative; expected ~500 based on duplication analysis of ~725 lines total across TransitionStatus, resolveAction, GetNextStatus)
   - **Traces to**: Epic REQ-NF-007

**Build Quality Gate**

5. **REQ-NF03-005**: Quality Gate
   - **Description**: `make fmt && make lint && make test` must pass after every task in this feature.
   - **Measurement**: CI/CD pipeline or manual execution
   - **Target**: Zero lint warnings, zero test failures
   - **Traces to**: Epic REQ-NF-009

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Epic TransitionStatus Delegates to EntityService**
- **Given** EpicService is constructed with an EntityService and EntityRepository adapter
- **When** `EpicService.TransitionStatus(ctx, "E21", "in_progress", opts)` is called
- **Then** the shared `EntityService.TransitionStatus` handles validation, backward detection, status update, and rejection note creation
- **And** EpicService adds child feature count to the TransitionResult in its post-hook
- **And** the result is identical to the pre-refactoring behavior

**Scenario 2: Feature TransitionStatus Delegates to EntityService**
- **Given** FeatureService is constructed with an EntityService and EntityRepository adapter
- **When** `FeatureService.TransitionStatus(ctx, "E21-F03", "in_development", opts)` is called
- **Then** the shared `EntityService.TransitionStatus` handles validation, backward detection, status update, and rejection note creation
- **And** FeatureService adds child task count to the TransitionResult in its post-hook

**Scenario 3: Task TransitionStatus Hybrid Delegation**
- **Given** TaskService is constructed with both its typed repository and EntityService
- **When** `TaskService.TransitionStatus(ctx, "E21-F03-001", "in_progress", opts)` is called
- **Then** shared validation and backward detection are reused from EntityService
- **And** the atomic `StatusUpdateRaw` is called via the typed task repository (not through EntityRepository adapter)
- **And** auto-unblocked keys are included in the result message
- **And** feature progress is recalculated after the transition

**Scenario 4: Backward Transition with Reason**
- **Given** an Epic in status "in_development"
- **When** `TransitionStatus(ctx, "E21", "draft", TransitionOptions{Reason: "Requirements changed"})` is called
- **Then** backward detection identifies this as a backward transition
- **And** the transition succeeds because a reason is provided
- **And** a rejection note is created with the reason

**Scenario 5: Backward Transition without Reason (Error)**
- **Given** a Feature in status "in_development"
- **When** `TransitionStatus(ctx, "E21-F03", "draft", TransitionOptions{})` is called (no reason)
- **Then** a `BackwardReasonError` is returned with from-status and to-status
- **And** the entity status is not changed

**Scenario 6: Forced Transition**
- **Given** any entity in any status
- **When** `TransitionStatus(ctx, key, targetStatus, TransitionOptions{Force: true, Reason: "Emergency fix"})` is called
- **Then** transition validation is skipped
- **And** the transition succeeds with `IsForced=true` in the result

**Scenario 7: Forced Transition without Reason (Error)**
- **Given** any entity in any status
- **When** `TransitionStatus(ctx, key, targetStatus, TransitionOptions{Force: true})` is called (no reason)
- **Then** `ErrForceReasonRequired` is returned

**Scenario 8: resolveAction Unification**
- **Given** EntityService is constructed with a workflow service that has status metadata with orchestrator actions
- **When** `EntityService.ResolveAction(entity, "in_development", extraPlaceholders)` is called
- **Then** base placeholders are generated from `EntityPlaceholders(entity)` via the Entity interface
- **And** extra placeholders are merged over the base
- **And** the orchestrator action instruction template is populated with all placeholders
- **And** the resulting `PopulatedAction` is identical to pre-refactoring behavior

**Scenario 9: GetNextStatus Unification**
- **Given** an Epic in status "draft"
- **When** `EpicService.GetNextStatus(ctx, "E21")` is called
- **Then** the shared `EntityService.GetNextStatus` retrieves available transitions from the workflow service
- **And** each transition includes an orchestrator action resolved via the entity-specific placeholder function

**Scenario 10: Quality Gate**
- **Given** all F03 refactoring is complete
- **When** `make fmt && make lint && make test` is run
- **Then** there are zero formatting changes, zero lint warnings, and zero test failures

---

## Out of Scope

### Explicitly Excluded

1. **Cross-Cutting Service Refactoring (NoteService, ContextService, ResumeService)**
   - **Why**: Cross-cutting services are refactored in F02 using the EntityRegistry pattern, which is a separate concern from status transition unification.
   - **Future**: F02 (already specified).

2. **Document Operations Unification (LinkDocument, UnlinkDocument)**
   - **Why**: Document operations are a separate duplication target with lower coupling to status transitions.
   - **Future**: Addressed in F04.

3. **CLI Command Refactoring**
   - **Why**: CLI commands calling `TransitionStatus` are thin wrappers that do not change. The service-layer method signatures remain compatible. CLI unification is F06.
   - **Workaround**: CLI commands continue to call entity-specific services, which delegate internally.

4. **Database Schema Changes**
   - **Why**: F03 is a pure service-layer refactoring. `StatusUpdateRaw` and all repository methods are unchanged.

5. **TaskService `StatusUpdateRaw` Refactoring**
   - **Why**: `StatusUpdateRaw` is a task-specific atomic operation (agent, notes, timestamps, auto-unblocking, session tracking) that has no equivalent in other entity types. Unifying it would be over-abstraction.
   - **Workaround**: TaskService retains its typed repository call for status updates.

6. **Enrichment Data Unification**
   - **Why**: Each entity type's enrichment data is structurally different (task enrichment includes dependencies and sessions; epic enrichment includes feature lists). Unifying enrichment is a separate effort that depends on F05 (Template Placeholder Unification).
   - **Workaround**: Entity-specific services pass enrichment data via `extraPlaceholders` to the shared `ResolveAction`.

---

## Success Metrics

### Primary Metrics

1. **Transition Logic Single-Location**
   - **What**: Count of distinct `TransitionStatus` implementations containing validation, backward detection, and rejection note logic
   - **Target**: 1 (EntityService) instead of current 3 (Epic, Feature, Task)
   - **Timeline**: After all F03 tasks complete
   - **Measurement**: Grep for backward detection logic (`IsBackwardTransition`) in entity services -- should appear only in `entity_service.go`

2. **resolveAction Deduplication**
   - **What**: Count of `resolveAction` method implementations across entity services
   - **Target**: 1 shared implementation (EntityService.ResolveAction) with 5 delegation call sites
   - **Timeline**: After all F03 tasks complete
   - **Measurement**: `grep -c "func.*resolveAction" internal/services/*_service.go`

3. **Net Line Reduction**
   - **What**: Total lines removed from entity services minus lines added in entity_service.go
   - **Target**: Net reduction of at least 300 lines
   - **Timeline**: After all F03 tasks complete
   - **Measurement**: `wc -l` before and after on affected files

### Secondary Metrics

- **GetNextStatus Deduplication**: 1 shared implementation instead of 3
- **New Entity Transition Effort**: ~20 lines of delegation code instead of ~120 lines of copied logic
- **Test Pass Rate**: 100% of existing tests pass after refactoring

---

## Dependencies & Integrations

### Dependencies

- **F01 (Entity Interface Foundation)**: F03 requires the `Entity` interface (for `GetStatus`, `GetID`, `GetKey`, etc.), the `EntityRepository` interface (for `UpdateStatus`, `GetByKey`), and the `EntityRepository` adapters for all 5 entity types. F01 must be complete before F03 implementation begins.
- **Existing typed repositories**: F03 wraps existing repositories via F01 adapters for the shared path. TaskService's `StatusUpdateRaw` bypasses the adapter and continues to use the typed repository directly.
- **`workflow.Service`**: The shared `EntityService` depends on `workflow.Service` for transition validation, normalization, backward detection, and status metadata. No changes to `workflow.Service` are required.

### Integration Points

- **`services_global.go`**: Updated to create `EntityService` and pass it to entity-specific service constructors.
- **Entity-specific service constructors**: Updated signatures to accept `entitySvc *EntityService` and `entityRepo EntityRepository`.
- **Existing tests**: Test files for entity services must be updated to pass `EntityService` mock/instance to constructors. Test assertions on behavior remain unchanged.
- **F02 (Cross-Cutting Service Unification)**: F02 and F03 can proceed in parallel after F01. They modify different files (F02: NoteService, ContextService, ResumeService; F03: EntityService, EpicService, FeatureService, TaskService). The EntityService created in F03 is distinct from the EntityRegistry created in F02, though both use the Entity interface from F01.

---

## Key Design Decision: TaskService Hybrid Delegation

The most significant design decision in F03 is how TaskService participates in the shared transition logic.

**Challenge**: TaskService's `executeStatusTransition` calls `StatusUpdateRaw`, which performs an atomic database operation including agent tracking, notes recording, timestamp management (started_at, completed_at, blocked_at), auto-unblocking of dependent tasks, and session tracking. This is fundamentally different from Epic/Feature's `repo.Update(entity)` and Bug/ChangeCard's `repo.UpdateStatus(id, status)`.

**Decision**: Hybrid delegation. TaskService reuses the shared validation/backward-detection/reason-enforcement logic from EntityService (either by calling EntityService methods directly or by extracting shared validation into helper methods on EntityService). TaskService retains its typed `executeStatusTransition` call for the actual status persistence. Post-transition, TaskService uses the shared `ResolveAction` for orchestrator actions and the shared rejection note creation.

**Alternative Rejected**: Full delegation through `EntityRepository.UpdateStatus`. This would require the TaskRepositoryAdapter's `UpdateStatus` to replicate `StatusUpdateRaw` behavior, which would be a leaky abstraction that defeats the purpose of the adapter pattern. The adapter is designed for simple status field updates, not complex atomic operations with side effects.

**Consequence**: TaskService's `TransitionStatus` method will be longer (~30-40 lines) than Epic/Feature's (~15 lines), but it will still eliminate the duplicated validation, backward detection, and action resolution logic (~50 lines saved from the current ~60-line method).

---

## Estimated Tasks

1. Create `EntityService` struct with `TransitionStatus`, `TransitionFeatures`, constructor, and sentinel errors
2. Implement `EntityService.ResolveAction` using `EntityPlaceholders` with extra-placeholder merge
3. Implement `EntityService.GetNextStatus` with action resolution callback
4. Refactor EpicService to delegate TransitionStatus, GetNextStatus, and resolveAction to EntityService
5. Refactor FeatureService to delegate TransitionStatus, GetNextStatus, and resolveAction to EntityService
6. Refactor TaskService to use hybrid delegation (shared validation + task-specific StatusUpdateRaw)
7. Update `services_global.go` to create and wire EntityService into entity-specific services
8. (Should-Have) Refactor Bug/ChangeCard to optionally delegate via SimpleTransitionFeatures
9. Write EntityService unit tests (parameterized by entity type and TransitionFeatures)
10. End-to-end validation: run full test suite and verify all status transition CLI operations

---

## Requirement Traceability

| Feature Requirement | Epic Requirement | Coverage |
|---|---|---|
| REQ-F03-001 (EntityService.TransitionStatus) | REQ-F-008 | Full |
| REQ-F03-002 (TransitionFeatures) | REQ-F-008 | Full |
| REQ-F03-003 (EpicService delegation) | REQ-F-009 | Full |
| REQ-F03-004 (FeatureService delegation) | REQ-F-009 | Full |
| REQ-F03-005 (TaskService hybrid delegation) | REQ-F-008, REQ-F-009 | Full |
| REQ-F03-006 (resolveAction unification) | REQ-F-011 | Partial (base placeholders; full placeholder unification is F05) |
| REQ-F03-007 (GetNextStatus unification) | REQ-F-008 | Full (implied) |
| REQ-F03-008 (Bug/ChangeCard optional) | REQ-F-008 | Partial (Should-Have; lower priority) |
| REQ-F03-009 (Constructor/wiring) | REQ-F-008 | Full |
| REQ-NF03-001 (zero behavioral changes) | REQ-NF-001 | Full |
| REQ-NF03-002 (performance) | REQ-NF-003 | Full |
| REQ-NF03-003 (test coverage) | REQ-NF-006 | Full |
| REQ-NF03-004 (code reduction) | REQ-NF-007 | Partial (~300-500 lines of epic's 800+ target) |
| REQ-NF03-005 (quality gate) | REQ-NF-009 | Full |

---

*Last Updated*: 2026-03-20
