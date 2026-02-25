---
feature_key: E15-F06-repository-layer-cleanup
epic_key: E15
title: Repository Layer Cleanup
description: Remove all business logic from repository layer (FeatureRepository, TaskRepository, EpicRepository), moving progress calculations, status derivations, and health checks into services. Repositories must only perform pure data access operations (SELECT/INSERT/UPDATE/DELETE).
---

# Repository Layer Cleanup

**Feature Key**: E15-F06-repository-layer-cleanup

---

## Epic

- **Epic PRD**: [Service Layer Architecture Refactoring](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Personas**: [User Personas](../../personas.md)

---

## Goal

### Problem

The repository layer currently violates the pure data access pattern by containing business logic. Three repositories perform calculations and derivations that belong in the service layer:

1. **FeatureRepository.CalculateProgress()** - Computes weighted and completion progress (business logic)
2. **TaskRepository.GetStatusBreakdown()** - Derives status summaries and categorizations (business logic)
3. **EpicRepository.GetHealthStatus()** - Calculates health indicators and impediments (business logic)

This creates two critical issues:
- **Architecture violation**: Repositories are "smart" when they should be "dumb" data access only
- **Logic duplication**: When services need progress/health data, they must either call repository methods (bypassing service layer) or duplicate the logic

### Solution

Remove all business logic methods from repositories, moving calculations into corresponding services (FeatureService, TaskService, EpicService). Repositories will only contain CRUD + List operations that return raw data models. Services will perform all calculations, derivations, and aggregations using data from repositories.

### Impact

- **Architecture clarity**: Clear layer boundary - repositories do data access, services do business logic
- **Service layer complete**: All business logic centralized in services (no exceptions)
- **Repository simplification**: ~30% reduction in repository method count (business logic methods removed)
- **Testability improved**: Business logic testable with mocked repositories (no database required)

---

## User Personas

### Primary Persona: Shark Core Developer

**Reference**: [Persona 1: Shark Core Developer](../../personas.md#persona-1-shark-core-developer-human) from epic personas

**Goals for This Feature**:
1. Find all business logic in services (not scattered across repositories and services)
2. Understand what repositories do by reading method signatures (no hidden logic)
3. Add new progress/health calculations in services only

**Pain Points This Feature Addresses**:
- Can't tell if repository methods contain business logic or just data access
- Must read repository implementation to understand if calculations are performed
- Business logic scattered across repositories and services (unclear ownership)

**Success Looks Like**:
Developer needs to change progress calculation logic. Opens `FeatureService.GetProgress()` (not repository), makes change in one place (service method), tests with mocked repository (no database needed).

---

## User Stories (MoSCoW)

### Must-Have Stories

**Story 1**: As a Shark developer, I want all business logic removed from FeatureRepository so that repositories only perform data access queries.

**Acceptance Criteria**:
- [ ] FeatureRepository.CalculateProgress() removed (logic moved to FeatureService)
- [ ] FeatureRepository.GetHealthStatus() removed (logic moved to FeatureService)
- [ ] FeatureRepository only contains: Create, Get, Update, Delete, List methods
- [ ] FeatureRepository methods return raw models (no calculated/derived fields)

**Story 2**: As a Shark developer, I want all business logic removed from TaskRepository so that repositories only perform data access queries.

**Acceptance Criteria**:
- [ ] TaskRepository.GetStatusBreakdown() removed (logic moved to TaskService)
- [ ] TaskRepository.GetDependencyTree() (if exists) simplified to data access only
- [ ] TaskRepository only contains: Create, Get, Update, Delete, List, UpdateStatus methods
- [ ] No workflow validation in repository (moved to TaskService)

**Story 3**: As a Shark developer, I want all business logic removed from EpicRepository so that repositories only perform data access queries.

**Acceptance Criteria**:
- [ ] EpicRepository.GetHealthStatus() removed (logic moved to EpicService)
- [ ] EpicRepository.GetImpediments() simplified to data query only (logic in EpicService)
- [ ] EpicRepository only contains: Create, Get, Update, Delete, List methods

### Should-Have Stories

**Story 4**: As a Shark maintainer, I want repository documentation updated so that contributors understand the pure data access pattern.

**Acceptance Criteria**:
- [ ] `.claude/rules/go/patterns.md` shows repository pattern (data access only)
- [ ] Repository godoc comments clarify "no business logic"
- [ ] Architecture docs show clear layer boundaries

---

## Requirements

### Functional Requirements

**Category: FeatureRepository Cleanup**

1. **REQ-F-001**: Remove CalculateProgress from FeatureRepository
   - **Description**: Remove FeatureRepository.CalculateProgress() method, move logic to FeatureService.GetProgress()
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Method removed from FeatureRepository
     - [ ] Logic migrated to FeatureService.GetProgress()
     - [ ] All callers updated to use service method
     - [ ] Tests updated to use service instead of repository

2. **REQ-F-002**: Remove GetHealthStatus from FeatureRepository
   - **Description**: Remove FeatureRepository.GetHealthStatus() method, move logic to FeatureService.GetHealth()
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Method removed from FeatureRepository
     - [ ] Logic migrated to FeatureService.GetHealth()
     - [ ] All callers updated to use service method

**Category: TaskRepository Cleanup**

3. **REQ-F-003**: Remove GetStatusBreakdown from TaskRepository
   - **Description**: Remove TaskRepository.GetStatusBreakdown() method, move logic to TaskService.GetStatusSummary()
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Method removed from TaskRepository
     - [ ] Logic migrated to TaskService.GetStatusSummary()
     - [ ] All callers updated to use service method

4. **REQ-F-004**: Simplify GetDependencyTree in TaskRepository
   - **Description**: If TaskRepository.GetDependencyTree() exists, simplify to pure data access (return raw task records, no logic)
   - **User Story**: Story 2
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Method returns raw task models only
     - [ ] No dependency resolution logic in repository
     - [ ] Logic moved to TaskService if needed

**Category: EpicRepository Cleanup**

5. **REQ-F-005**: Remove GetHealthStatus from EpicRepository
   - **Description**: Remove EpicRepository.GetHealthStatus() method, move logic to EpicService.GetHealth()
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Method removed from EpicRepository
     - [ ] Logic migrated to EpicService.GetHealth()
     - [ ] All callers updated to use service method

6. **REQ-F-006**: Simplify GetImpediments in EpicRepository
   - **Description**: Simplify EpicRepository.GetImpediments() to pure data query (no health/blocking analysis)
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Method returns raw blocked tasks only (SQL query: WHERE status = 'blocked')
     - [ ] No impediment analysis logic in repository
     - [ ] Logic moved to EpicService.GetImpediments()

### Non-Functional Requirements

**Architecture**

1. **REQ-NF-001**: Repository Purity
   - **Description**: All repository methods must perform only data access operations (no calculations, no derivations)
   - **Measurement**: Code review - verify no business logic in repository methods
   - **Target**: 100% pure data access (0 business logic methods remain)
   - **Justification**: Clean architecture requires clear layer boundaries

**Testability**

2. **REQ-NF-002**: Service Layer Testability
   - **Description**: All moved business logic must be testable with mocked repositories
   - **Measurement**: Service tests use mocks (not real database)
   - **Target**: 100% of service methods tested with mocks
   - **Justification**: Service layer tests should not require database

---

## Acceptance Criteria (Feature-Level)

### Scenario 1: Repository Layer Audit

**Given** a code reviewer audits the repository layer
**When** they examine FeatureRepository, TaskRepository, EpicRepository
**Then** no business logic methods exist (only CRUD + List operations)
**And** no progress calculations exist in repositories
**And** no status derivations exist in repositories
**And** no workflow validations exist in repositories

### Scenario 2: Service Layer Ownership

**Given** a developer needs to change progress calculation logic
**When** they search for progress calculation code
**Then** all logic exists in FeatureService.GetProgress() (not repository)
**And** repository only returns raw task data
**And** service performs all calculations

### Scenario 3: Test Coverage Verification

**Given** repository cleanup is complete
**When** service tests are examined
**Then** all service tests use mocked repositories
**And** no service tests require real database
**And** business logic is tested in isolation

---

## Out of Scope

### Explicitly Excluded

1. **Repository Method Renaming**
   - **Why**: Focus is removing business logic methods, not renaming remaining CRUD methods
   - **Future**: Method naming standardization can be addressed separately if needed
   - **Workaround**: N/A

2. **Database Schema Changes**
   - **Why**: Moving logic from repositories to services doesn't require schema changes
   - **Future**: N/A
   - **Workaround**: N/A

3. **HTTP API Repository Calls**
   - **Why**: HTTP API should use services (covered in separate features). This feature focuses on repository cleanup only.
   - **Future**: Ensure HTTP API uses services (not repositories directly)
   - **Workaround**: Audit HTTP API before removing repository methods

---

## Success Metrics

### Primary Metrics

1. **Repository Method Reduction**
   - **What**: Percentage of business logic methods removed from repositories
   - **Target**: 50% reduction (6 business logic methods → 0)
   - **Timeline**: Measured at feature completion
   - **Measurement**: Count methods before/after in FeatureRepository, TaskRepository, EpicRepository

2. **Business Logic Centralization**
   - **What**: Percentage of business logic in services vs. repositories
   - **Target**: 100% in services (0% in repositories)
   - **Timeline**: Measured at feature completion
   - **Measurement**: Code audit - verify no calculations/derivations in repositories

---

## Dependencies & Integrations

### Dependencies

- **FeatureService Implementation** (E15-F05): FeatureService must exist before moving repository logic
- **EpicService Implementation** (E15-F05): EpicService must exist before moving repository logic
- **TaskService Enhancement**: TaskService may need new methods for status breakdown logic

### Integration Requirements

- **Service Layer**: Services must have methods to accept raw data from repositories and perform calculations
- **CLI Commands**: CLI commands must call services (not repositories directly) after cleanup

---

## Compliance & Security Considerations

**Not Applicable**: This is an internal architecture refactoring. No user-facing changes, no data access changes, no security controls affected.

---

*Last Updated*: 2026-02-17
