---
feature_key: E15-F12-service-layer-completion-and-cli-integration
epic_key: E15
title: Service Layer Completion and CLI Integration
description: Final phase of service layer refactoring - remove business logic from repositories, complete CLI command refactoring to thin wrappers, and verify service layer integration is complete across all commands.
---

# Service Layer Completion and CLI Integration

**Feature Key**: E15-F12-service-layer-completion-and-cli-integration

---

## Epic

- **Epic PRD**: [Service Layer Architecture Refactoring](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Personas**: [User Personas](../../personas.md)

---

## Goal

### Problem

While TaskService, EpicService, and FeatureService have been created in prior features (F01-F11), the CLI commands remain fat controllers (2,664 lines for task.go, 2,254 for feature.go, 1,940 for epic.go) with embedded business logic. Additionally, repository layer still contains business logic (progress calculations, status derivations, health checks) that should reside in services. This incomplete migration creates three critical issues:

1. **CLI commands still too complex**: Commands contain 40-45% of business logic mixed with argument parsing and output formatting, making them difficult to test and maintain
2. **Repository layer violations**: Repositories perform business logic (FeatureRepository.CalculateProgress, TaskRepository.GetStatusBreakdown, EpicRepository.GetHealthStatus) violating the pure data access pattern
3. **Service layer underutilized**: Existing services (TaskService, FeatureService, EpicService) exist but CLI commands bypass them, calling repositories directly, defeating the purpose of the service layer

### Solution

Complete the service layer migration by: (1) removing all business logic from repositories, moving progress/health calculations into services, (2) refactoring all CLI commands to thin wrappers (200-400 lines) that only parse arguments, call service methods, and format output, and (3) adding global service accessors (GetTaskService(), GetFeatureService(), GetEpicService()) to enable clean dependency injection in CLI commands. This finalizes the clean three-layer architecture: CLI → Service → Repository → Database.

### Impact

- **CLI commands reduced**: From 6,858 lines (task + feature + epic) to ~1,200 lines total (82% reduction)
- **Repository layer simplified**: Business logic removed from 3 repositories (FeatureRepository, TaskRepository, EpicRepository), making them pure data access
- **Service layer complete**: All business logic centralized in services, reusable across CLI and HTTP API
- **Testability improved**: Services unit-testable with mocked repositories (no database required)
- **Architecture clarity**: Clear layer boundaries enforced through code structure (no violations possible)

---

## User Personas

### Primary Persona: Shark Core Developer

**Reference**: [Persona 1: Shark Core Developer](../../personas.md#persona-1-shark-core-developer-human) from epic personas

**Goals for This Feature**:
1. Add new CLI commands by calling existing service methods (not duplicating business logic)
2. Find business logic in services (not scattered across commands and repositories)
3. Write unit tests for business logic without mocking CLI framework or database
4. Understand codebase architecture through clear layer boundaries

**Pain Points This Feature Addresses**:
- Cannot find where business logic lives (commands? services? repositories? all three?)
- Must read 2,000+ line command files to understand simple operations
- Tests require complex mocking (CLI framework + repositories + workflow services)
- New features copy-paste patterns from existing commands, propagating duplication

**Success Looks Like**:
Developer adds "task archive" command in 2 hours by creating 150-line CLI wrapper calling `TaskService.Archive()`. Business logic is in the service (tested independently), repository performs data access only. Developer reads service interface to understand available operations instead of parsing command files.

### Secondary Persona: AI Code Agent

**Reference**: [Persona 2: AI Code Agent](../../personas.md#persona-2-ai-code-agent-cursorclaude) from epic personas

**Goals for This Feature**:
1. Suggest service method calls for new CLI commands (not inline business logic)
2. Identify where business logic lives (services only, clear boundary)
3. Generate service layer unit tests (not CLI integration tests)

**Success Looks Like**:
Agent asked "How do I prevent deleting tasks in 'completed' status?" responds with 10-line change to `TaskService.Delete()` method (not 100-line CLI command modification). Generated tests mock repositories, not CLI framework.

---

## User Stories (MoSCoW)

### Must-Have Stories

**Story 1**: As a Shark developer, I want all business logic removed from repositories so that repositories only perform data access queries.

**Acceptance Criteria**:
- [ ] FeatureRepository.CalculateProgress() removed (logic moved to FeatureService)
- [ ] TaskRepository.GetStatusBreakdown() removed (logic moved to TaskService)
- [ ] EpicRepository.GetHealthStatus() removed (logic moved to EpicService)
- [ ] Repository methods only perform SELECT/INSERT/UPDATE/DELETE operations
- [ ] Repository methods return raw data models (no calculated fields)
- [ ] No workflow validation in repositories (moved to services)

**Story 2**: As a Shark developer, I want CLI commands to be thin wrappers (200-400 lines) so that I can understand command flow without reading thousands of lines.

**Acceptance Criteria**:
- [ ] task.go reduced from 2,664 lines to ≤400 lines (85% reduction)
- [ ] feature.go reduced from 2,254 lines to ≤350 lines (84% reduction)
- [ ] epic.go reduced from 1,940 lines to ≤300 lines (85% reduction)
- [ ] Each command has structure: parse args → call service → format output
- [ ] No business logic in commands (only parsing, service calls, formatting)
- [ ] All commands use service layer methods (TaskService, FeatureService, EpicService)

**Story 3**: As a Shark developer, I want global service accessors (cli.GetTaskService()) so that commands can easily access services without complex dependency wiring.

**Acceptance Criteria**:
- [ ] Global accessor functions exist: GetTaskService(), GetFeatureService(), GetEpicService()
- [ ] Accessors defined in internal/cli/services_global.go
- [ ] Accessors use lazy initialization (create service on first call)
- [ ] Accessors reuse shared dependencies (DB connection, workflow service)
- [ ] Commands call accessors instead of constructing services manually

### Should-Have Stories

**Story 4**: As a Shark developer, I want all existing CLI tests to pass without modification so that refactoring doesn't break functionality.

**Acceptance Criteria**:
- [ ] `make test` passes with zero test changes required
- [ ] CLI output format unchanged (same JSON structure, same table columns)
- [ ] CLI command syntax unchanged (same flags, same arguments)
- [ ] Performance within ±10% of baseline (refactoring doesn't slow commands)

**Story 5**: As a Shark maintainer, I want architecture documentation updated so that contributors understand the new layer boundaries.

**Acceptance Criteria**:
- [ ] `.claude/rules/architecture.md` updated with service layer patterns
- [ ] `.claude/rules/services/service-design.md` reflects completed migration
- [ ] `.claude/rules/cli/commands.md` shows thin wrapper pattern
- [ ] Migration guide exists in `docs/guides/service-layer-migration.md`

### Could-Have Stories

**Story 6**: As an HTTP API consumer, I want API documentation showing which service methods each endpoint uses so that I understand API/CLI feature parity.

**Acceptance Criteria**:
- [ ] API docs list service method for each endpoint (e.g., POST /tasks/{key}/start → TaskService.StartTask())
- [ ] API docs reference CLI command equivalents
- [ ] Service interface documentation is publicly accessible

---

## Requirements

### Functional Requirements

**Category: Repository Layer Cleanup**

1. **REQ-F-001**: Remove Business Logic from FeatureRepository
   - **Description**: Remove all business logic methods from FeatureRepository, moving calculations into FeatureService
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] FeatureRepository.CalculateProgress() removed → moved to FeatureService.GetProgress()
     - [ ] FeatureRepository.GetHealthStatus() removed → moved to FeatureService.GetHealth()
     - [ ] FeatureRepository only contains: Create, Get, Update, Delete, List methods
     - [ ] FeatureRepository methods return raw models (no calculated/derived fields)

2. **REQ-F-002**: Remove Business Logic from TaskRepository
   - **Description**: Remove all business logic methods from TaskRepository, moving query logic into TaskService
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] TaskRepository.GetStatusBreakdown() removed → moved to TaskService.GetStatusSummary()
     - [ ] TaskRepository.GetDependencyTree() (if exists) simplified to data access only
     - [ ] TaskRepository only contains: Create, Get, Update, Delete, List, UpdateStatus methods
     - [ ] No workflow validation in repository (moved to TaskService)

3. **REQ-F-003**: Remove Business Logic from EpicRepository
   - **Description**: Remove all business logic methods from EpicRepository, moving aggregations into EpicService
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] EpicRepository.GetHealthStatus() removed → moved to EpicService.GetHealth()
     - [ ] EpicRepository.GetImpediments() simplified to data query only (logic in EpicService)
     - [ ] EpicRepository only contains: Create, Get, Update, Delete, List methods

**Category: CLI Command Refactoring**

4. **REQ-F-004**: Refactor task.go to Thin Wrapper
   - **Description**: Refactor task.go (2,664 lines) to thin wrapper calling TaskService methods
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] task.go reduced to ≤400 lines (excluding tests)
     - [ ] All task commands call TaskService methods (no direct repository calls)
     - [ ] Command structure: parse args (20-50 lines) → call service (1-5 lines) → format output (20-50 lines)
     - [ ] No business logic in command files (validation, orchestration, calculations in TaskService)

5. **REQ-F-005**: Refactor feature.go to Thin Wrapper
   - **Description**: Refactor feature.go (2,254 lines) to thin wrapper calling FeatureService methods
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] feature.go reduced to ≤350 lines (excluding tests)
     - [ ] All feature commands call FeatureService methods (no direct repository calls)
     - [ ] Command structure: parse args → call service → format output

6. **REQ-F-006**: Refactor epic.go to Thin Wrapper
   - **Description**: Refactor epic.go (1,940 lines) to thin wrapper calling EpicService methods
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] epic.go reduced to ≤300 lines (excluding tests)
     - [ ] All epic commands call EpicService methods (no direct repository calls)
     - [ ] Command structure: parse args → call service → format output

**Category: Service Integration**

7. **REQ-F-007**: Global Service Accessors
   - **Description**: Create global accessor functions for services in CLI package
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] File internal/cli/services_global.go created
     - [ ] Functions exist: GetTaskService(), GetFeatureService(), GetEpicService()
     - [ ] Accessors use lazy initialization (create on first call, cache instance)
     - [ ] Accessors reuse shared dependencies (DB connection via GetDB(), workflow service)
     - [ ] All commands use accessors instead of constructing services manually

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Performance Neutrality
   - **Description**: Refactoring must not degrade CLI command performance by more than 10%
   - **Measurement**: Benchmark CLI commands before/after refactoring
   - **Target**: `shark task next --agent=backend` executes in ≤55ms (baseline: 50ms, +10%)
   - **Justification**: Service layer adds abstraction but should not add significant overhead

**Testability**

2. **REQ-NF-002**: Zero Test Regression
   - **Description**: All existing tests must pass without modification
   - **Measurement**: `make test` exit code
   - **Target**: 100% pass rate (0 failures, 0 modifications to existing test expectations)
   - **Justification**: Refactoring is behavior-preserving transformation

**Maintainability**

3. **REQ-NF-003**: Code Duplication Elimination
   - **Description**: Remove duplicated business logic patterns across command files
   - **Measurement**: Code duplication analysis (e.g., jscpd)
   - **Target**: <5% duplication in CLI command files (current: ~30%)
   - **Justification**: Duplication makes maintenance expensive and error-prone

---

## Acceptance Criteria (Feature-Level)

### Scenario 1: Developer Adds New CLI Command

**Given** a developer needs to add a new "task archive" command
**When** they create the command file and implement the handler
**Then** the command file is <200 lines (parse args, call TaskService.Archive(), format output)
**And** no business logic exists in the command (all logic in TaskService.Archive() method)
**And** tests mock TaskService (not repositories or database)

### Scenario 2: Repository Layer Audit

**Given** a code reviewer audits the repository layer
**When** they examine FeatureRepository, TaskRepository, EpicRepository
**Then** no business logic methods exist (only CRUD + List operations)
**And** no progress calculations exist in repositories
**And** no status derivations exist in repositories
**And** no workflow validations exist in repositories

### Scenario 3: CLI Command Size Verification

**Given** the refactoring is complete
**When** line counts are measured for task.go, feature.go, epic.go
**Then** task.go is ≤400 lines (85% reduction from 2,664)
**And** feature.go is ≤350 lines (84% reduction from 2,254)
**And** epic.go is ≤300 lines (85% reduction from 1,940)
**And** total reduction is ~5,600 lines removed (82% reduction)

### Scenario 4: Service Layer Integration

**Given** all CLI commands are refactored
**When** a developer inspects command implementations
**Then** every command calls a service method (GetTaskService(), GetFeatureService(), GetEpicService())
**And** no command directly creates repository instances
**And** no command contains validation logic (all in services)
**And** no command contains progress calculations (all in services)

### Scenario 5: Test Suite Regression Check

**Given** refactoring is complete
**When** `make test` is executed
**Then** all tests pass (100% pass rate)
**And** no test expectations were modified (same assertions)
**And** test execution time is unchanged ±10%
**And** no new test failures introduced

---

## Out of Scope

### Explicitly Excluded

1. **HTTP API Implementation**
   - **Why**: HTTP API service integration is covered in separate features (F08-F11). This feature focuses on CLI refactoring completion only.
   - **Future**: HTTP API wiring continues in parallel with CLI refactoring but is not blocked by this feature.
   - **Workaround**: HTTP API can begin using services as they are created, independent of CLI refactoring status.

2. **New Service Methods**
   - **Why**: Services (TaskService, FeatureService, EpicService) already exist with necessary methods. This feature moves existing logic, not creates new capabilities.
   - **Future**: New service methods will be added in future features as needed.
   - **Workaround**: If CLI refactoring discovers missing service methods, create them as part of refactoring.

3. **Service Layer Testing Improvements**
   - **Why**: Service layer tests exist from prior features (F01-F11). This feature focuses on CLI integration, not service test coverage improvements.
   - **Future**: Service test coverage improvements tracked separately.
   - **Workaround**: Add service tests if gaps discovered during CLI refactoring.

4. **Database Schema Changes**
   - **Why**: Repository cleanup involves removing methods, not changing database schema. No migrations required.
   - **Future**: N/A
   - **Workaround**: N/A

---

## Success Metrics

### Primary Metrics

1. **CLI Command Size Reduction**
   - **What**: Line count reduction in task.go, feature.go, epic.go
   - **Target**: 82% reduction (6,858 lines → 1,200 lines)
   - **Timeline**: Measured at feature completion
   - **Measurement**: `wc -l internal/cli/commands/{task,feature,epic}.go`

2. **Business Logic Centralization**
   - **What**: Percentage of business logic in service layer vs. command layer
   - **Target**: 100% of business logic in services (0% in commands or repositories)
   - **Timeline**: Measured at feature completion
   - **Measurement**: Code audit - verify no validation/calculation/orchestration in commands or repositories

3. **Zero Test Regression**
   - **What**: Test pass rate before vs. after refactoring
   - **Target**: 100% pass rate maintained (0 new failures)
   - **Timeline**: Verified on each commit
   - **Measurement**: `make test` exit code

### Secondary Metrics

- **Repository Method Count**: 50% reduction (business logic methods removed)
- **Code Duplication**: <5% in CLI commands (from ~30%)
- **PR Review Cycles**: 50% reduction (architecture is clear, less review overhead)

---

## Dependencies & Integrations

### Dependencies

- **TaskService Implementation** (E15-F02, F03, F04): TaskService must have all methods needed by CLI commands
- **FeatureService Implementation** (E15-F05): FeatureService must handle progress/health calculations
- **EpicService Implementation** (E15-F05): EpicService must handle feature rollups and impediments
- **Existing Tests**: All existing CLI tests must be compatible with service layer refactoring

### Integration Requirements

- **Global Service Accessors**: Must integrate with existing `cli.GetDB()` pattern for database access
- **Workflow Service**: Service accessors must reuse existing `cli.GetWorkflowService()` singleton
- **Error Handling**: Service errors must be translated to CLI exit codes (1=not found, 2=DB error, 3=invalid state)

---

## Compliance & Security Considerations

**Not Applicable**: This is an internal architecture refactoring. No user-facing changes, no data access changes, no security controls affected. All existing security patterns (input validation, error handling) preserved in service layer.

---

## Open Questions & Assumptions

**Assumption 1: Service Methods Complete**
- **Context**: CLI refactoring assumes all necessary service methods exist from F01-F11
- **Impact**: If methods are missing, CLI refactoring will discover gaps
- **Recommendation**: Audit service interfaces before starting CLI refactoring; add missing methods as discovered

**Assumption 2: Test Compatibility**
- **Context**: Assumes existing CLI tests can pass with service layer refactoring
- **Impact**: If tests are tightly coupled to implementation, may need test updates
- **Recommendation**: Review existing test structure; prioritize tests that verify behavior (not implementation)

**Assumption 3: Performance Acceptable**
- **Context**: Service layer adds abstraction overhead (function calls, interface dispatching)
- **Impact**: Assumes <10% overhead is acceptable for maintainability gains
- **Recommendation**: Benchmark critical paths (task next, task start) before/after refactoring

**Assumption 4: Repository Cleanup Doesn't Break API**
- **Context**: HTTP API may call repository methods directly (bypassing services)
- **Impact**: Removing repository methods could break API endpoints
- **Recommendation**: Audit HTTP API handlers before removing repository methods; ensure API uses services

**No additional open questions** - Feature scope is well-defined based on existing service implementations and clear architectural target.

---

*Last Updated*: 2026-02-17
