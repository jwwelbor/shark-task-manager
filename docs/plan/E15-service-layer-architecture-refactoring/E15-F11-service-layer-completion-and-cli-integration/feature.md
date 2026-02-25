---
feature_key: E15-F11-service-layer-completion-and-cli-integration
epic_key: E15
title: Service Layer Completion and CLI Integration
description: Complete the service layer refactoring by migrating the remaining 13 fat-controller CLI command files to thin wrappers, adding missing service methods for history/notes/criteria/analytics/context concerns, and ensuring test coverage for the migrated commands.
status: in_refinement_ba
---

# Service Layer Completion and CLI Integration

**Feature Key**: E15-F11-service-layer-completion-and-cli-integration

---

## Epic

- **Epic PRD**: [Service Layer Architecture Refactoring](../../epic.md)

---

## Goal

### Problem

Epic E15 established a three-layer architecture (CLI Command → Service → Repository) and made significant progress. The core command files (`task.go`, `epic.go`, `feature.go`) are now thin wrappers, and the core services (`TaskService`, `EpicService`, `FeatureService`, `NoteService`, `ContextService`, `ResumeService`) exist with comprehensive methods.

However, 13 CLI command files still contain direct repository access and business logic (fat controller pattern). These commands cover task history, task notes, task criteria, task context management, task next-status transitions, analytics, view, notes search, validation, task linking, feature criteria, and task resume. Until these are migrated, the E15 goal of "zero fat controllers" is incomplete, and the architecture benefits (testability via mocks, reusability across CLI and HTTP API) are not fully realized.

### Current State

**Completed in E15 (prior features):**
- `task.go` - thin wrapper using `TaskService`
- `epic.go` - thin wrapper using `EpicService`
- `feature.go` - thin wrapper using `FeatureService`
- `TaskService` - 40+ methods covering lifecycle, CRUD, querying, dependencies, relationships, documents, work sessions
- `EpicService` - comprehensive methods covering CRUD, rollup, health, completion, cascade
- `FeatureService` - comprehensive methods covering CRUD, progress, health, work breakdown, completion
- `NoteService`, `ContextService`, `ResumeService` - global accessors registered in `services_global.go`
- All tests passing (30 packages, 0 failures)

**Remaining fat-controller CLI command files (13 files):**

| File | Lines | Direct Repo/DB Calls | Services Partially Covering This Concern |
|------|-------|---------------------|------------------------------------------|
| `task_next_status.go` | 361 | 7 | `TaskService.TransitionStatus`, `TaskService.GetNextStatus` |
| `task_note.go` | 442 | 12 | `NoteService.AddNote`, `NoteService.ListNotes` |
| `task_criteria.go` | 460 | 16 | None (criteria service needed) |
| `task_context.go` | 397 | 6 | `ContextService.GetContext`, `ContextService.SetContextField` |
| `task_resume.go` | 401 | 5 | `ResumeService` exists |
| `analytics.go` | 238 | 5 | `TaskService.GetWorkSessions` (partial) |
| `history.go` | 277 | 4 | None (history service method needed in TaskService or dedicated) |
| `notes_search.go` | 230 | 6 | `NoteService.SearchNotes` |
| `task_link.go` | 198 | 4 | `TaskService.LinkDocument`, `TaskService.UnlinkRelationships` |
| `feature_criteria.go` | 195 | 5 | None (criteria service needed) |
| `validate.go` | 98 | 4 | None (infrastructure command - may be acceptable) |
| `view.go` | 104 | 4 | `ContextService`, `EpicService`, `FeatureService` |
| `task_history.go` | 247 | 4 | None (history service method needed) |

**Infrastructure exceptions (acceptable direct DB access):**
- `cloud.go` - cloud connectivity test, not business logic
- `init.go` - project initialization, not business logic
- `migrate_backfill_slugs.go` - one-time migration utility

### Solution

Migrate the 13 remaining fat-controller files to thin wrappers by:
1. Adding missing service methods to existing services (`TaskService`, `NoteService`)
2. Adding a global service accessor for `GetIdeaService` if needed
3. Migrating each command file to call the appropriate service method
4. Updating CLI tests to use mocked services where they currently use real databases

### Impact

- **Architecture compliance**: Zero fat controllers remaining (excluding infrastructure commands)
- **Testability**: CLI commands testable via service mocks, not real database
- **Reusability**: All business logic exposed via service layer for future HTTP API use
- **Maintainability**: Single location for each piece of business logic

---

## User Stories (MoSCoW)

### Must-Have Stories

**Story 1**: As a developer working on the HTTP API, I want task history retrieval to be in a service method so that I can reuse it from HTTP handlers without duplicating the CLI command logic.

**Acceptance Criteria**:
- Given `history.go` and `task_history.go` contain direct repository access
- When these commands are refactored
- Then they call a `TaskService.GetHistory()` or equivalent service method
- And the service method is usable from any entry point (CLI, HTTP)

**Story 2**: As a developer writing CLI tests, I want `task_note.go`, `task_criteria.go`, and `task_context.go` commands to call service methods so that I can test them by mocking the service instead of using a real database.

**Acceptance Criteria**:
- Given the commands currently create repositories directly from `cli.GetDB()`
- When each command is refactored to call the appropriate service
- Then no `repository.New*` calls remain in the command handler functions
- And tests for these commands can use mock services

**Story 3**: As a developer maintaining the codebase, I want `task_next_status.go` to call `TaskService.TransitionStatus()` so that the transition logic is not duplicated between CLI and future API consumers.

**Acceptance Criteria**:
- Given `task_next_status.go` contains `performTransition()` with direct repository access and cascade triggering
- When it is refactored
- Then `performTransition()` is removed from the CLI layer
- And the command calls `TaskService.TransitionStatus()` which handles validation, transition, and cascade
- And the `--preview`, `--force`, `--status`, `--reason` flags continue to work correctly

**Story 4**: As a developer, I want `analytics.go` to call a service method for work session analytics so that the analytics business logic is testable and reusable.

**Acceptance Criteria**:
- Given `analytics.go` creates `WorkSessionRepository` directly
- When it is refactored
- Then it calls `TaskService.GetWorkSessions()` or a new `AnalyticsService` method
- And analytics filtering (by epic, feature, agent) remains functional

**Story 5**: As a QA engineer, I want all 13 migrated command files to have their CLI tests updated so that they use mock services and do not depend on a real database.

**Acceptance Criteria**:
- Given 17 CLI test files currently use real test databases
- When migration is complete for these commands
- Then integration tests for migrated commands use mocked services
- And `make test` continues to pass with 0 failures

### Should-Have Stories

**Story 6**: As a developer, I want `notes_search.go` to call `NoteService.SearchNotes()` so that note searching is in the service layer.

**Acceptance Criteria**:
- Given `notes_search.go` creates repositories directly
- When refactored
- Then it calls `NoteService.SearchNotes()` via `cli.GetNoteService()`
- And search filters (entity type, epic/feature scope, note type) remain functional

**Story 7**: As a developer, I want `task_context.go` to call `ContextService` methods so that context management is in the service layer.

**Acceptance Criteria**:
- Given `task_context.go` creates `TaskRepository` directly for context operations
- When refactored
- Then it calls `ContextService.GetContext()`, `ContextService.SetContextField()`, `ContextService.ClearContext()`
- And all context subcommands (`get`, `set`, `clear`) continue to work

**Story 8**: As a developer, I want `task_resume.go` and `feature_resume.go` (if any) to call `ResumeService` so that resume logic is in the service layer.

**Acceptance Criteria**:
- Given resume commands create repositories directly
- When refactored
- Then they call `ResumeService.GetTaskResume()` or `ResumeService.GetFeatureResume()`
- And resume output (context, recent history, notes) remains complete

### Could-Have Stories

**Story 9**: As a developer, I want `validate.go` to call a `ValidationService` or use existing services for entity resolution so that repository access is abstracted.

**Note**: `validate.go` uses the `validation` package which wraps repositories via an adapter pattern. This is lower priority as the business logic is already encapsulated in the `validation` package.

**Story 10**: As a developer, I want `task_criteria.go` and `feature_criteria.go` to call a `CriteriaService` so that acceptance criteria management is in the service layer.

**Acceptance Criteria**:
- Given both files create `TaskCriteriaRepository` directly with significant logic
- When a `CriteriaService` is created
- Then both commands call the service
- And CRUD operations for criteria (add, list, update, complete, delete) remain functional

---

## Scope

### In Scope

**Service method additions (new methods on existing services):**
1. `TaskService.GetTaskHistory()` - retrieve task change history for `history.go` / `task_history.go`
2. `TaskService.GetAnalytics()` - analytics aggregation for `analytics.go` (or delegate to `TaskService.GetWorkSessions()`)
3. `NoteService` may need a `GetNoteTimeline()` for task timeline display

**CLI command migrations (thin wrapper refactoring):**
1. `task_next_status.go` - call `TaskService.TransitionStatus()`, `TaskService.GetNextStatus()`
2. `task_note.go` - call `NoteService.AddNote()`, `NoteService.ListNotes()`, and new timeline method
3. `task_context.go` - call `ContextService.*` methods via `cli.GetContextService()`
4. `task_resume.go` - call `ResumeService.*` methods via `cli.GetResumeService()`
5. `history.go` - call new `TaskService.GetTaskHistory()`
6. `task_history.go` - call new `TaskService.GetTaskHistory()` (may overlap with `history.go`)
7. `notes_search.go` - call `NoteService.SearchNotes()` via `cli.GetNoteService()`
8. `task_link.go` - call `TaskService.LinkDocument()`, `TaskService.UnlinkRelationships()`
9. `analytics.go` - call service method for work session analytics
10. `view.go` - call `ContextService`, `EpicService`, `FeatureService` as appropriate
11. `validate.go` - assess whether `validation` package adapter pattern is acceptable or needs service wrapping
12. `task_criteria.go` - create `CriteriaService` or extend existing service
13. `feature_criteria.go` - use same `CriteriaService`

**Test updates:**
- Update CLI tests for migrated commands to use mock services instead of real databases
- Ensure `make test` passes with 0 failures after each migration

### Out of Scope

- Infrastructure commands (`cloud.go`, `init.go`, `migrate_backfill_slugs.go`) - acceptable to keep direct DB access
- `epic_helpers.go` (1149 lines) and `feature_helpers.go` (1270 lines) - these are helper functions, not command handlers; assess separately
- HTTP API handlers - not in this epic
- New CLI commands - this feature is refactoring only

---

## Requirements

### REQ-F-001: Remaining Fat Controllers Eliminated
All 13 identified CLI command files must have zero `repository.New*()` calls in their command handler functions (`runXxx` functions). Helper/private functions within the same file may retain repository calls only if they exist solely to support a service-level concern that has no service backing yet (documented exceptions).

### REQ-F-002: Service Methods for Uncovered Concerns
All business logic currently in fat controllers must move to an appropriate service:
- History retrieval → `TaskService` or dedicated `HistoryService`
- Analytics computation → `TaskService.GetAnalytics()` or similar
- Note timeline → `NoteService`
- Criteria CRUD → new `CriteriaService` or extended existing service

### REQ-F-003: Zero Test Regression
`make test` must pass with 0 failures before and after each migration. Tests must be updated alongside command migrations, not after.

### REQ-F-004: CLI Test Mock Compliance
For each migrated command file, the corresponding `*_test.go` file must not call `test.GetTestDB()` for testing the command logic. Integration tests that require real database access must be in `internal/repository/` tests only.

### REQ-F-005: Functional Equivalence
All CLI command behaviors must be preserved exactly. No flags, options, or output formats may change as a result of refactoring. The refactoring is purely architectural.

---

## Edge Cases and Error Scenarios

### EC-001: Commands with Interactive Mode
`task_next_status.go` includes interactive status selection (prompts user when multiple transitions available). The service method must support preview mode and non-interactive flag-driven transitions. The interactive prompt logic remains in the CLI layer.

### EC-002: Status Cascade Side Effects
`task_next_status.go` calls `triggerStatusCascade()` after transitions. This side effect must be preserved in `TaskService.TransitionStatus()`. Verify cascade still fires when migrating this command.

### EC-003: Criteria Operations with Multiple Subcommands
`task_criteria.go` and `feature_criteria.go` implement multiple subcommands (add, list, update, complete, delete). Any new `CriteriaService` must expose all these operations, not just the most common ones.

### EC-004: Note Timeline vs Note List
`task_note.go` implements both a notes list (`task notes`) and a timeline view (`task timeline`) which formats notes with history interleaved. These have different data requirements. The timeline service method needs to aggregate both notes and history records.

### EC-005: Analytics Scope Resolution
`analytics.go` resolves feature/epic scope by looking up entity IDs. This lookup logic (validate entity exists, get ID) belongs in the service layer, not the command.

### EC-006: View Command Multi-Entity Support
`view.go` opens files for epics, features, and tasks. The path resolution logic (finding the file path from the entity key) should use existing service methods (`EpicService.ResolveEpicPath`, `FeatureService.ResolveFeaturePath`).

### EC-007: Validation Command Infrastructure Pattern
`validate.go` uses a `validation.RepositoryAdapter` pattern already encapsulating repository logic. This is an acceptable boundary case - the command calls `validation.NewValidator()` which is not a raw repository. Document as an accepted infrastructure pattern rather than requiring a new service wrapper.

---

## Success Metrics

- **Fat controllers remaining**: 0 (excluding 3 infrastructure commands)
- **Test pass rate**: 100% (0 failures on `make test`)
- **CLI tests using real DB**: Reduction from 17 files to ≤3 files (integration tests for complex scenarios only)
- **Service method coverage**: All business logic accessible via service interfaces
- **Build**: `make build` succeeds after each migration

---

## Dependencies

- **Requires**: `TaskService.TransitionStatus()` - already implemented
- **Requires**: `TaskService.GetNextStatus()` - already implemented
- **Requires**: `NoteService.AddNote()`, `NoteService.ListNotes()`, `NoteService.SearchNotes()` - already implemented
- **Requires**: `ContextService.GetContext()`, `ContextService.SetContextField()`, `ContextService.ClearContext()` - already implemented
- **Requires**: `ResumeService.GetFeatureResume()` - already implemented
- **New**: `TaskService.GetTaskHistory()` - to be added
- **New**: Analytics service method - to be added
- **New**: Note timeline service method - to be added
- **New**: `CriteriaService` (or extension of existing service) - to be created

---

## Implementation Phases

### Phase 1: Service Method Additions (prerequisite)
Add missing service methods to enable migration without breaking functionality:
- `TaskService.GetTaskHistory(ctx, key string) ([]*models.TaskHistory, error)`
- `TaskService.GetAnalyticsByFeature(ctx, featureKey, agentType string) (*AnalyticsResult, error)`
- `TaskService.GetAnalyticsByEpic(ctx, epicKey, agentType string) (*AnalyticsResult, error)`
- `NoteService.GetNoteTimeline(ctx, taskKey string) (*NoteTimeline, error)` (aggregates notes + history)
- `CriteriaService` with Add, List, Update, Complete, Delete methods (or extend `TaskService`)

### Phase 2: High-Impact Migrations
Migrate commands with the most lines and complexity first:
- `task_criteria.go` (460 lines, highest complexity)
- `task_note.go` (442 lines)
- `task_context.go` (397 lines)
- `task_resume.go` (401 lines)

### Phase 3: Lifecycle and Search Migrations
- `task_next_status.go` (361 lines, critical workflow path)
- `notes_search.go` (230 lines)
- `task_history.go` (247 lines)
- `history.go` (277 lines)

### Phase 4: Utility Command Migrations
- `analytics.go` (238 lines)
- `task_link.go` (198 lines)
- `feature_criteria.go` (195 lines)
- `view.go` (104 lines)
- `validate.go` (98 lines) - assess and document pattern decision

### Phase 5: Test Compliance
- Update CLI tests for all migrated commands to use mock services
- Remove `test.GetTestDB()` calls from command tests
- Verify `make test` passes with 0 failures

---

## Acceptance Criteria

### AC-001: Zero Fat Controllers
**Given** the list of 13 fat-controller CLI command files identified in this PRD
**When** this feature is complete
**Then** `grep -rn "repository.New\|cli.GetDB\|repoDb" internal/cli/commands/ --include="*.go" | grep -v "_test.go" | grep -v "mock_"` returns only results from `cloud.go`, `init.go`, and `migrate_backfill_slugs.go` (infrastructure commands)

### AC-002: All Tests Pass
**Given** the current test baseline (30 packages passing, 0 failing)
**When** all migrations are complete
**Then** `make test` exits with code 0 and reports 0 failures
**And** `make lint` exits with code 0 and reports no new violations
**And** `make fmt` requires 0 changes

### AC-003: Service Methods Available for HTTP API
**Given** the new service methods added in Phase 1
**When** an HTTP handler needs task history, analytics, note timeline, or criteria
**Then** it can call the service methods directly without accessing repositories
**And** the service methods accept `context.Context` as first parameter and return domain models

### AC-004: Task Next-Status Transition Preserved
**Given** the `task next-status` command with `--preview`, `--force`, `--status`, and `--reason` flags
**When** refactored to call `TaskService.TransitionStatus()`
**Then** interactive selection still works when no `--status` flag is provided
**And** `--preview` shows available transitions without making changes
**And** `--force` bypasses workflow validation
**And** auto-unblocked tasks are still displayed after a transition
**And** status cascade is still triggered after a transition

### AC-005: CLI Test Database Independence
**Given** CLI tests for migrated commands currently use `test.GetTestDB()`
**When** migration is complete
**Then** command-level tests for migrated files do not use `test.GetTestDB()`
**And** mocked services are used to test command parsing and output formatting

### AC-006: Functional Equivalence
**Given** all existing CLI commands and their output formats
**When** any command from the 13 migrated files is run
**Then** its output (JSON and human-readable) is identical to pre-migration behavior
**And** all flags continue to work as documented

---

## Notes for Technical Design

- `task_next_status.go` contains the `triggerStatusCascade` helper that passes `repoDb` directly. This needs to move to `TaskService` or be called via the service after a transition.
- `task_criteria.go` handles criteria with complex state (pending, completed). A `CriteriaService` may need to support optimistic locking or ordering.
- `task_note.go` timeline view interleaves notes with task history - the service method needs to aggregate both note records and `task_history` records in chronological order.
- `feature_helpers.go` (1270 lines) and `epic_helpers.go` (1149 lines) were noted as containing significant logic. These are helper files (not command handlers) and should be assessed in a separate task to determine if they are thin enough or need service extraction.
- `view.go` opens entity files in an editor - the path resolution is the business logic concern. The `os.Open` or editor invocation stays in the CLI layer.

---

*Last Updated*: 2026-02-19
