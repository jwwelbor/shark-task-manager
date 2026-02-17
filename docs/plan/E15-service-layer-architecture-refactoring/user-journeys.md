# User Journeys

**Epic**: [Service Layer Architecture Refactoring](./epic.md)

---

## Overview

This document maps the developer experience before and after the service layer refactoring. Each journey shows the current pain points (fat controllers, duplication, poor testability) and how the service layer architecture resolves them.

---

## Journey 1: Adding a New CLI Command

### Actors
- Shark Core Developer (Persona 1)
- AI Code Agent (Persona 2)

### Current State (Fat Controller Pattern)

**Scenario**: Developer wants to add `shark task archive` command to move completed tasks to archive directory.

**Steps**:
1. Developer opens `internal/cli/commands/task.go` (2,671 lines)
2. Scrolls through mixed concerns (argument parsing, business logic, repo calls, formatting)
3. Finds `taskCompleteCmd` as reference example
4. Copies 200+ lines of boilerplate (DB initialization, repo creation, error handling, key lookup)
5. Adds business logic inline: check status is "completed", get file path, move file, update database
6. Realizes progress calculation is needed - searches for pattern in file
7. Finds progress logic at line 1,843 in separate function
8. Copies progress calculation pattern (another 50 lines)
9. Adds status validation by copying pattern from line 342
10. Writes integration test that requires full database setup (100+ lines)
11. PR submitted with 350 new lines, 200 of which are duplicated patterns

**Pain Points**:
- 2+ hours spent navigating large file to find patterns
- Duplicated 60% of code from existing commands
- Test requires database, cannot isolate business logic
- Business logic now lives in 3 places (`task.go`, `task_deps.go`, `task_archive.go`)

**Time Taken**: 8 hours (4 hours coding, 2 hours testing, 2 hours fixing review feedback)

### Target State (Service Layer Pattern)

**Scenario**: Same requirement - add `shark task archive` command.

**Steps**:
1. Developer reads `internal/services/task_service.go` interface (200 lines, well-documented)
2. Adds `Archive(ctx context.Context, taskKey string) error` method signature
3. Implements business logic in service method (30 lines):
   - Call `repo.GetByKey()` for task lookup
   - Validate status is "completed" via `workflowSvc.ValidateTransition()`
   - Call `archiveService.MoveToArchive()` for file operation
   - Call `repo.UpdateArchived()` to mark in database
4. Creates CLI command wrapper in `internal/cli/commands/task_archive.go` (80 lines):
   - Parse arguments
   - Call `taskService.Archive()`
   - Format output (JSON or success message)
5. Writes service layer unit test (40 lines) with mocked repository
6. PR submitted with 150 new lines, zero duplication

**Improvements**:
- 30 minutes to find interface and understand pattern
- Zero duplication (reuses existing service methods)
- Test is unit test with mocks, runs in 10ms instead of 500ms
- Business logic consolidated in service layer
- HTTP API automatically gets archive capability

**Time Taken**: 2 hours (1 hour coding, 30 minutes testing, 30 minutes documentation)

**Efficiency Gain**: 75% reduction in development time

---

## Journey 2: Understanding Business Logic

### Actors
- AI Code Agent (Persona 2)
- Shark Maintainer (Persona 4)

### Current State (Logic Scattered Across Files)

**Scenario**: AI agent asked "How does task dependency checking work?"

**Steps**:
1. AI loads `internal/cli/commands/task.go` into context (2,671 lines - context overflow warning)
2. Searches for "dependency" - finds 8 references across 500 lines
3. Some dependency logic in `task.go` lines 890-950
4. Some dependency logic in `task_deps.go` lines 120-340
5. Some dependency logic in `internal/repository/task_repository.go` lines 456-489
6. Cannot load all files simultaneously due to context limits
7. Makes partial answer based on incomplete information
8. Suggests change that misses 2 of the 3 dependency check locations

**Pain Points**:
- Logic scattered across 3 files and 1,500 total lines
- AI cannot load full context
- Incomplete understanding leads to incorrect suggestions
- Developer must manually verify AI suggestion across all 3 files

**Time to Answer**: 45 minutes (AI) + 30 minutes (developer verification) = 75 minutes

### Target State (Logic Centralized in Service)

**Scenario**: Same question - "How does task dependency checking work?"

**Steps**:
1. AI loads `internal/services/task_service.go` (200 lines total)
2. Finds `ValidateDependencies()` method (lines 145-180)
3. Reads complete logic in one place:
   - Check if dependencies exist
   - Check if dependencies are completed
   - Return validation error if blocked
4. AI provides accurate answer with code reference
5. Suggests correct change location: `TaskService.ValidateDependencies()`

**Improvements**:
- Logic in single 35-line method
- AI loads full context without overflow
- Complete understanding on first read
- Suggestions are accurate and complete

**Time to Answer**: 5 minutes (AI)

**Efficiency Gain**: 93% reduction in research time

---

## Journey 3: Writing Tests for Business Logic

### Actors
- Shark Core Developer (Persona 1)
- Test Engineer (Secondary Persona)

### Current State (Testing Requires Full Stack)

**Scenario**: Write test for "cannot complete a task that is already completed"

**Steps**:
1. Create test database setup (20 lines)
2. Initialize database schema (5 lines)
3. Create test epic and feature (30 lines)
4. Create task in "completed" status (15 lines)
5. Call CLI command function `runTaskCompleteCommand()` with mocked Cobra context (25 lines)
6. Assert error contains expected message (5 lines)
7. Cleanup database (10 lines)
8. Test runs in 450ms due to database I/O

**Pain Points**:
- 110 lines of test code for 1 business rule
- Must initialize full database schema
- Test is slow (450ms)
- Cannot test business logic in isolation
- Brittle - breaks if database schema changes

**Test Complexity**: High (integration test)
**Execution Time**: 450ms per test

### Target State (Unit Test with Mocks)

**Scenario**: Same test - "cannot complete a task that is already completed"

**Steps**:
1. Create mock repository returning task with "completed" status (8 lines)
2. Call `taskService.Complete(ctx, taskKey)` (1 line)
3. Assert error is `ErrAlreadyCompleted` (2 lines)
4. Test runs in 0.5ms (in-memory)

**Improvements**:
- 11 lines of test code (90% reduction)
- No database required
- Test is fast (0.5ms - 900x faster)
- Tests business logic in isolation
- Resilient to database schema changes

**Test Complexity**: Low (unit test)
**Execution Time**: 0.5ms per test

**Efficiency Gain**: 90% less test code, 900x faster execution

---

## Journey 4: HTTP API Feature Parity

### Actors
- HTTP API Consumer (Persona 3)
- DevOps Engineer (Secondary Persona)

### Current State (API Lacks CLI Features)

**Scenario**: Build CI/CD integration that gets next task for backend agent

**Steps**:
1. Check HTTP API documentation - `/api/tasks/next` endpoint exists
2. Call `GET /api/tasks/next?agent=backend`
3. Discover endpoint doesn't support agent-type filtering (logic only in CLI)
4. Attempt workaround: `GET /api/tasks?status=ready_for_development` and filter client-side
5. Discover client-side filter is incomplete (missing priority sorting, dependency checking)
6. Give up on API, shell out to CLI: `shark task next --agent=backend --json`
7. Parse CLI JSON output in CI/CD script
8. Now dependent on CLI binary being installed in CI environment

**Pain Points**:
- API lacks features available in CLI
- Client-side workarounds are incomplete and buggy
- Forced to use CLI in headless environments
- CI/CD pipeline depends on CLI binary availability
- No API documentation for "feature gaps"

**Integration Effort**: 6 hours (4 hours building workarounds, 2 hours debugging edge cases)

### Target State (API Has Full CLI Parity)

**Scenario**: Same requirement - CI/CD integration for next task

**Steps**:
1. Check HTTP API documentation - `/api/tasks/next?agent=backend` documented
2. Call `GET /api/tasks/next?agent=backend`
3. Receives JSON response with same data as CLI
4. API uses same `TaskService.GetNextTask()` method as CLI
5. All business logic (filtering, priority sorting, dependency checking) works identically
6. Integration complete

**Improvements**:
- API has 100% feature parity with CLI
- No workarounds needed
- CI/CD pipeline is clean API calls
- No dependency on CLI binary
- Behavior guaranteed consistent (same code path)

**Integration Effort**: 1 hour (30 minutes reading docs, 30 minutes implementation)

**Efficiency Gain**: 83% reduction in integration effort

---

## Journey 5: Maintaining Architecture Over Time

### Actors
- Shark Maintainer (Persona 4)
- Shark Core Developer (Persona 1)

### Current State (Architecture Drift)

**Scenario**: Review PR adding feature that calculates epic progress

**Steps**:
1. Maintainer reviews PR - sees business logic added to `epic.go` command (150 lines)
2. Leaves comment: "Business logic should not be in command layer, please extract to service"
3. Developer asks: "Where should it go? We don't have EpicService yet"
4. Maintainer: "Create EpicService or add to a helper package for now"
5. Developer creates `internal/helpers/epic_progress.go` (not a service, just moves code)
6. Maintainer: "This is still not service layer architecture, see architecture.md"
7. Developer now frustrated, creates minimal `EpicService` with only this one method
8. PR goes through 3 review cycles over 5 days
9. Next contributor sees `helpers/epic_progress.go` and adds more business logic there
10. Architecture continues to drift despite best efforts

**Pain Points**:
- No enforced architecture pattern
- Contributors unclear on "where things go"
- Maintainer burden: same feedback on every PR
- Helper packages become dumping ground for business logic
- Architecture debt grows despite code review

**Review Cycles**: 3 cycles over 5 days
**Contributor Frustration**: High

### Target State (Self-Documenting Architecture)

**Scenario**: Same PR - add epic progress calculation

**Steps**:
1. Developer reads existing `EpicService` interface (already exists post-E15)
2. Adds `CalculateProgress(ctx, epicKey) (float64, error)` method to service
3. Implements business logic in service method (35 lines)
4. Updates CLI command to call `epicService.CalculateProgress()` (5 lines)
5. Maintainer reviews PR - architecture is correct by construction
6. Reviewer focuses on business logic correctness, not architecture
7. PR approved in 1 cycle (same day)

**Improvements**:
- Architecture is self-evident (service layer exists with clear interfaces)
- Contributor knows immediately where code belongs
- No architecture feedback needed in review
- Focus shifts to feature quality, not structure
- Pattern is established for future contributors

**Review Cycles**: 1 cycle (same day)
**Contributor Frustration**: None (clear pattern to follow)

**Efficiency Gain**: 80% reduction in review time, 100% elimination of architecture drift

---

## Journey Summary

| Journey | Metric | Before (Current) | After (Service Layer) | Improvement |
|---------|--------|------------------|----------------------|-------------|
| Adding New Command | Development Time | 8 hours | 2 hours | **75% faster** |
| Understanding Logic | Research Time | 75 minutes | 5 minutes | **93% faster** |
| Writing Tests | Lines of Test Code | 110 lines | 11 lines | **90% less code** |
| Writing Tests | Test Execution | 450ms | 0.5ms | **900x faster** |
| API Integration | Integration Effort | 6 hours | 1 hour | **83% faster** |
| PR Review | Review Cycles | 3 cycles (5 days) | 1 cycle (same day) | **80% faster** |

**Overall Development Efficiency Gain: 60-75% reduction in time spent on architecture-related work**

---

*See also*: [Requirements](./requirements.md), [Success Metrics](./success-metrics.md)
