---
feature_key: E15-F10-cli-command-refactoring-and-enhancements
epic_key: E15
title: CLI Command Refactoring and Enhancements
description: Complete the thin-wrapper refactoring of all remaining CLI commands that still access the database directly, and create the missing IdeaService to cover the idea domain.
---

# CLI Command Refactoring and Enhancements

**Feature Key**: E15-F10-cli-command-refactoring-and-enhancements

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)
- **Feature Architecture**: [Architecture](./02-architecture.md)

---

## Goal

### Problem

Seven CLI command files in `internal/cli/commands/` still use the fat-controller pattern — they call `cli.GetDB()` and `repository.New*()` directly instead of delegating to the service layer. These files total approximately 2,892 lines of direct repository access code mixed with business logic in the command layer:

- `idea.go` (1002 lines) — No IdeaService exists; entire file is a fat controller
- `task_deps.go` (780 lines) — Dependency management bypasses service layer
- `related_docs.go` (462 lines) — Document linking bypasses service layer
- `search.go` (176 lines) — Search bypasses service layer
- `task_sessions.go` (153 lines) — Session tracking bypasses service layer
- `task_unlink.go` (178 lines) — File unlinking bypasses service layer
- `status.go` (141 lines) — Status dispatch uses GetDB directly

This violates the clean architecture established in E15-F07 through F09, creates inconsistency in the codebase, and prevents unit testing of business logic without a real database.

Additionally, the service layer work from E15-F09 (EpicService and FeatureService) is complete, but was originally scoped into E15-F10 task specifications — those tasks need to be updated to reflect the actual remaining work.

### Solution

1. Create `IdeaService` as the missing service for the idea domain, enabling `idea.go` to become a thin wrapper
2. Extend `TaskService`, `EpicService`, and `FeatureService` with missing methods needed by the remaining fat controllers
3. Refactor all seven fat-controller command files to thin wrappers following the established pattern: parse args → call service → format output
4. Add a `GetIdeaService()` accessor to `internal/cli/service_accessors.go`

### Impact

- All CLI command files conform to the thin-wrapper pattern
- Business logic is testable without a real database
- The architecture rule "CLI commands must not call repositories directly" holds across the entire codebase
- ~2,892 lines of mixed business/presentation code reduced to ~530 lines of pure presentation code

---

## User Personas

### Developer Maintainer

**Profile**:
- **Role/Title**: Go developer maintaining the shark-task-manager codebase
- **Experience Level**: Familiar with the project, understands the architecture goals
- **Key Characteristics**:
  - Writes unit tests for all new code
  - Expects consistent patterns across the codebase
  - Uses mock-based testing for service layer work

**Goals Related to This Feature**:
1. Add new features to idea management, task dependencies, and document linking without fighting mixed concerns
2. Write unit tests for idea and dependency business logic without a database

**Pain Points This Feature Addresses**:
- Cannot unit test idea.go or task_deps.go business logic without a real database
- Adding a feature to idea management requires understanding both CLI presentation and database access in the same file

**Success Looks Like**:
Developer can add a new field to idea creation by modifying IdeaService alone, and test it with a mock repository in under 30 minutes.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want all CLI command files to follow the thin-wrapper pattern so that business logic can be unit tested without a database.

**Acceptance Criteria**:
- [ ] `idea.go` contains no `repository.New*` or `cli.GetDB()` calls
- [ ] `task_deps.go` contains no `repository.New*` or `cli.GetDB()` calls
- [ ] `related_docs.go` contains no `repository.New*` or `cli.GetDB()` calls
- [ ] `search.go` contains no `repository.New*` or `cli.GetDB()` calls
- [ ] `task_sessions.go` contains no `repository.New*` or `cli.GetDB()` calls
- [ ] `task_unlink.go` contains no `repository.New*` or `cli.GetDB()` calls
- [ ] `status.go` contains no `repository.New*` or `cli.GetDB()` calls

**Story 2**: As a developer, I want an IdeaService in `internal/services/` that encapsulates all idea business logic so that idea.go becomes a thin wrapper.

**Acceptance Criteria**:
- [ ] `internal/services/idea_service.go` exists with CRUD methods and idea key generation
- [ ] `internal/services/idea_dto.go` exists with CreateIdeaInput, UpdateIdeaInput, IdeaFilters
- [ ] `GetIdeaService()` accessor exists in `internal/cli/service_accessors.go`
- [ ] `internal/services/idea_service_test.go` exists with mocked repository tests

**Story 3**: As a developer, I want TaskService extended with dependency management methods so that task_deps.go becomes a thin wrapper.

**Acceptance Criteria**:
- [ ] `TaskService.AddDependency(ctx, taskKey, depKey)` exists
- [ ] `TaskService.RemoveDependency(ctx, taskKey, depKey)` exists
- [ ] `TaskService.ListDependencies(ctx, taskKey)` exists
- [ ] New methods are tested with mocked repositories

---

### Should-Have Stories

**Story 4**: As a developer, I want EpicService and FeatureService extended with document linking methods so that related_docs.go becomes a thin wrapper.

**Acceptance Criteria**:
- [ ] `EpicService.LinkDocument(ctx, epicKey, docPath)` exists
- [ ] `EpicService.UnlinkDocument(ctx, epicKey, docPath)` exists
- [ ] `FeatureService.LinkDocument(ctx, featureKey, docPath)` exists
- [ ] `FeatureService.UnlinkDocument(ctx, featureKey, docPath)` exists

**Story 5**: As a developer, I want the existing EpicService and FeatureService integration with CLI commands verified so that any remaining partial violations are cleaned up.

**Acceptance Criteria**:
- [ ] `epic_helpers.go` has no functions accepting `*repository.EpicRepository` as a parameter
- [ ] `feature_helpers.go` has no functions accepting `*repository.FeatureRepository` as a parameter
- [ ] Service tests confirm all critical code paths are covered with mocked repositories

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when an idea key already exists for a given date and sequence, I want the IdeaService to return a clear error so that the CLI can display an appropriate message.

**Acceptance Criteria**:
- [ ] IdeaService returns a typed error for duplicate key conflicts
- [ ] idea.go command displays "Idea with this key already exists" and exits with code 1

---

## Requirements

### Functional Requirements

**Category: IdeaService Implementation**

1. **REQ-F-001**: IdeaService CRUD Operations
   - **Description**: IdeaService must implement Create, Get, List, Update, Delete, and MarkAsConverted operations, encapsulating all business logic currently in idea.go
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `CreateIdea(ctx, CreateIdeaInput) (*models.Idea, error)` — includes key generation
     - [ ] `GetIdea(ctx, key) (*models.Idea, error)`
     - [ ] `ListIdeas(ctx, IdeaFilters) ([]*models.Idea, error)`
     - [ ] `UpdateIdea(ctx, key, UpdateIdeaInput) (*models.Idea, error)`
     - [ ] `DeleteIdea(ctx, key) error`
     - [ ] `ConvertIdea(ctx, key, convertedToType, convertedToKey) error`

2. **REQ-F-002**: Idea Key Generation in Service
   - **Description**: The `I-YYYY-MM-DD-xx` key format generation must live in IdeaService, not in idea.go command
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Key generation uses current date at service invocation time
     - [ ] Sequence number increments correctly when multiple ideas created on same date
     - [ ] Key generation logic tested with mocked date and repository

**Category: TaskService Extensions**

3. **REQ-F-003**: Dependency Management in TaskService
   - **Description**: TaskService must provide methods for adding, removing, listing, and validating task dependencies
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `AddDependency` validates both task and dependency exist before adding
     - [ ] `RemoveDependency` returns error if dependency does not exist
     - [ ] `ListDependencies` returns all tasks that the given task depends on
     - [ ] Circular dependency detection implemented in service layer (not CLI)

**Category: CLI Command Refactoring**

4. **REQ-F-004**: All Fat Controllers Eliminated
   - **Description**: All seven identified fat-controller files must be refactored to thin wrappers
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] No `repository.New*` calls in any file in `internal/cli/commands/` (except test files)
     - [ ] No `cli.GetDB()` calls in command handler functions
     - [ ] `make lint` passes with no architectural violations

### Non-Functional Requirements

**Architecture Compliance**

1. **REQ-NF-001**: Thin Wrapper Standard
   - **Description**: All CLI command handler functions must conform to the three-step pattern: parse args, call service, format output
   - **Target**: Each command handler file averages fewer than 50 lines per command handler function
   - **Measurement**: Manual review and line count audit

**Testing**

2. **REQ-NF-002**: Service Test Coverage
   - **Description**: All new service code (IdeaService, TaskService extensions) must have unit tests using mocked repositories
   - **Target**: Minimum 80% line coverage for new service code
   - **Measurement**: `make test-coverage` report

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: No Direct Repository Access in CLI Commands**
- **Given** the E15-F10 implementation is complete
- **When** `grep -rn "repository\.New\|cli\.GetDB" internal/cli/commands/` is run
- **Then** only test files and mock files contain matches
- **And** all production command handler files are free of direct repository access

**Scenario 2: IdeaService Enables Thin Wrapper**
- **Given** `IdeaService` exists in `internal/services/idea_service.go`
- **When** `idea.go` is inspected
- **Then** all run* functions follow the parse→call→format pattern
- **And** idea.go contains no repository imports

**Scenario 3: All Tests Pass**
- **Given** all refactoring is complete
- **When** `make fmt && make lint && make test` is run
- **Then** all three commands exit with code 0
- **And** no test failures are reported

---

## Out of Scope

### Explicitly Excluded

1. **epic_helpers.go and feature_helpers.go further refactoring beyond current state**
   - **Why**: These files were the focus of E15-F07 and are substantially compliant
   - **Future**: May be addressed in a future cleanup epic if remaining helpers need service extraction
   - **Workaround**: Current state is functionally correct; this is a code quality concern only

2. **HTTP API handler refactoring**
   - **Why**: HTTP handlers are a separate concern from CLI commands; cmd/server/services.go already uses proper service wiring
   - **Future**: HTTP handler layer already follows the correct pattern

3. **New CLI features or commands**
   - **Why**: E15-F10 is strictly a refactoring effort; new functionality would be a separate feature

---

## Success Metrics

### Primary Metrics

1. **Fat Controller Count**
   - **What**: Number of CLI command files with direct repository access
   - **Target**: 0 (from current 7)
   - **Timeline**: End of E15-F10 implementation
   - **Measurement**: `grep -rn "repository\.New\|cli\.GetDB" internal/cli/commands/ | grep -v "_test.go"` returns empty

2. **New Service Test Coverage**
   - **What**: Unit test coverage of IdeaService and TaskService extensions
   - **Target**: >= 80% line coverage
   - **Timeline**: End of E15-F10 implementation
   - **Measurement**: `make test-coverage` HTML report

---

## Dependencies & Integrations

### Dependencies

- **E15-F09 (Additional Entity Services)**: EpicService and FeatureService must be fully implemented (complete as of 2026-02-18)
- **E15-F07 (CLI Commands as Thin Wrappers)**: Pattern established for task.go must be followed

### Integration Requirements

- **`internal/repository/idea_repository.go`**: IdeaService depends on the existing IdeaRepository implementation
- **`internal/models/idea.go`**: IdeaService uses existing Idea model

---

*Last Updated*: 2026-02-18
