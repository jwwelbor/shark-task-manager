# Test Plan: Service Layer Completion and CLI Integration

**Feature**: E15-F12 - Service Layer Completion and CLI Integration
**Epic**: E15 - Service Layer Architecture Refactoring
**Complexity**: STANDARD
**Test Plan Type**: Focused Test Plan (AC Matrix + API Contracts + Component Strategy + Integration Scenarios)
**Date**: 2026-02-17

---

## Executive Summary

This test plan validates the completion of service layer refactoring by ensuring:
1. **Repository layer becomes pure data access** - No business logic remains in repositories
2. **CLI commands become thin wrappers** - 82% line reduction (6,858 → ~1,200 lines) with zero behavioral changes
3. **Service layer fully integrated** - 100% of CLI commands use services, no direct repository calls
4. **Zero test regression** - All existing tests pass unchanged

**Critical Success Criteria**:
- ✅ All 6 repository business logic methods removed and moved to services
- ✅ CLI command files reduced: task.go ≤400 lines, feature.go ≤350 lines, epic.go ≤300 lines
- ✅ `make test` passes with 100% pass rate, zero test modifications
- ✅ Performance within ±10% of baseline

---

## 1. Acceptance Criteria Test Matrix

### Story 1: Business Logic Removed from Repositories

**User Goal**: As a Shark developer, I want all business logic removed from repositories so that repositories only perform data access queries.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC1.1** | FeatureRepository.CalculateProgress() removed (logic moved to FeatureService) | Verify method doesn't exist in FeatureRepository | Grep for `CalculateProgress` in `internal/repository/feature_repository.go` | No matches found | Method may be renamed or refactored; verify no progress calculation logic in repository |
| **AC1.1** | FeatureService.GetProgress() added | Call FeatureService.GetProgress(ctx, featureID) | featureID=1 with 2/4 tasks completed | Returns 50.0 (float64) | Empty feature (0 tasks), all tasks completed (100%), no completed tasks (0%) |
| **AC1.2** | TaskRepository.GetStatusBreakdown() removed (logic moved to TaskService) | Verify method doesn't exist in TaskRepository | Grep for `GetStatusBreakdown` in `internal/repository/task_repository.go` | No matches found | Method may be private helper; verify no status aggregation logic |
| **AC1.2** | TaskService.GetStatusSummary() added | Call TaskService.GetStatusSummary(ctx, featureID) | featureID=1 with tasks: 2 todo, 1 in_progress, 1 completed | Returns map: {"todo": 2, "in_progress": 1, "completed": 1} | Empty feature, all same status, many different statuses |
| **AC1.3** | EpicRepository.GetHealthStatus() removed (logic moved to EpicService) | Verify method doesn't exist in EpicRepository | Grep for `GetHealthStatus` in `internal/repository/epic_repository.go` | No matches found | May be multiple health methods; check all variants |
| **AC1.3** | EpicService.GetHealth() added | Call EpicService.GetHealth(ctx, epicID) | epicID=1 with blocked tasks | Returns "critical" string | No tasks, all healthy tasks, mixed health statuses |
| **AC1.4** | Repository methods only perform SELECT/INSERT/UPDATE/DELETE operations | Code review of all repository methods | Review FeatureRepository, TaskRepository, EpicRepository | All methods are: Create, Get, Update, Delete, List, or raw SQL queries | Verify no calculations, no conditionals beyond null checks |
| **AC1.5** | Repository methods return raw data models (no calculated fields) | Inspect returned models from repository methods | Call GetByID, List methods | Models contain only database columns, no derived fields | Check for Progress, Health, Status breakdowns in model structs |
| **AC1.6** | No workflow validation in repositories (moved to services) | Grep for workflow validation in repositories | Search for `workflow` or `ValidateTransition` in repository files | No matches in repository files (only in services) | Check for status constants, transition logic |

### Story 2: CLI Commands Become Thin Wrappers

**User Goal**: As a Shark developer, I want CLI commands to be thin wrappers (200-400 lines) so that I can understand command flow without reading thousands of lines.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC2.1** | task.go reduced from 2,664 lines to ≤400 lines (85% reduction) | Line count measurement | `wc -l internal/cli/commands/task.go` | ≤400 lines (excluding tests) | Count includes blank lines/comments; verify actual code lines |
| **AC2.2** | feature.go reduced from 2,254 lines to ≤350 lines (84% reduction) | Line count measurement | `wc -l internal/cli/commands/feature.go` | ≤350 lines (excluding tests) | Same as above |
| **AC2.3** | epic.go reduced from 1,940 lines to ≤300 lines (85% reduction) | Line count measurement | `wc -l internal/cli/commands/epic.go` | ≤300 lines (excluding tests) | Same as above |
| **AC2.4** | Each command has structure: parse args → call service → format output | Code review of task.go commands | Review runTaskStart, runTaskComplete, runTaskGet | Each handler: 1) Parse args (10-50 lines), 2) Call service (1-5 lines), 3) Format output (10-50 lines) | Verify no inline business logic, no repository calls |
| **AC2.5** | No business logic in commands (only parsing, service calls, formatting) | Grep for business logic patterns | Search for validation loops, calculations, workflow checks in command files | No matches (only in services) | Check for conditionals beyond nil checks, loops beyond formatting |
| **AC2.6** | All commands use service layer methods (TaskService, FeatureService, EpicService) | Grep for service calls | Search for `cli.GetTaskService()`, `cli.GetFeatureService()`, `cli.GetEpicService()` in commands | All commands call services | No `repository.New*Repository()` calls in commands |

### Story 3: Global Service Accessors

**User Goal**: As a Shark developer, I want global service accessors (cli.GetTaskService()) so that commands can easily access services without complex dependency wiring.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC3.1** | Global accessor functions exist: GetTaskService(), GetFeatureService(), GetEpicService() | File existence and function signatures | Check `internal/cli/services_global.go` | Functions exist with correct signatures | Functions may be unexported; verify public API |
| **AC3.2** | Accessors defined in internal/cli/services_global.go | File location verification | `ls internal/cli/services_global.go` | File exists | File may be named differently; check for pattern |
| **AC3.3** | Accessors use lazy initialization (create service on first call) | Unit test of accessor pattern | Call GetTaskService() twice | Service created once, reused on second call | Verify no global state pollution between calls |
| **AC3.4** | Accessors reuse shared dependencies (DB connection, workflow service) | Verify dependency wiring | Inspect GetTaskService() implementation | Calls cli.GetDB(), cli.GetWorkflowService() (shared singletons) | Check for duplicate DB connections, workflow service instances |
| **AC3.5** | Commands call accessors instead of constructing services manually | Grep for service construction | Search for `services.NewTaskService()` in command files | No direct service construction in commands (only in accessors) | May have legacy code; verify all paths use accessors |

### Story 4: All Existing CLI Tests Pass

**User Goal**: As a Shark developer, I want all existing CLI tests to pass without modification so that refactoring doesn't break functionality.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC4.1** | `make test` passes with zero test changes required | Full test suite execution | Run `make test` | Exit code 0, 100% pass rate | Flaky tests may fail intermittently; rerun to confirm |
| **AC4.2** | CLI output format unchanged (same JSON structure, same table columns) | Output comparison tests | Run `shark task get E07-F01-001 --json` before/after | Identical JSON structure (same fields, same types) | Field order may change; verify semantic equivalence |
| **AC4.3** | CLI command syntax unchanged (same flags, same arguments) | Command help comparison | Run `shark task start --help` before/after | Identical help text, same flags, same usage | Help text formatting may differ; verify functionality unchanged |
| **AC4.4** | Performance within ±10% of baseline (refactoring doesn't slow commands) | Benchmark comparison | Benchmark `shark task next --agent=backend` | Baseline: 50ms, Acceptable: 45-55ms | System load may affect timing; run multiple iterations |

### Story 5: Architecture Documentation Updated

**User Goal**: As a Shark maintainer, I want architecture documentation updated so that contributors understand the new layer boundaries.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|----------------|------------|
| **AC5.1** | `.claude/rules/architecture.md` updated with service layer patterns | Documentation review | Read "Current State (Legacy)" section | Section renamed to "Current Architecture" with no legacy warnings | May keep legacy section for history; verify prominent current state |
| **AC5.2** | `.claude/rules/services/service-design.md` reflects completed migration | Documentation review | Read service design doc | No "partial services" or "being refactored" language | May have forward-looking content; verify current state accurate |
| **AC5.3** | `.claude/rules/cli/commands.md` shows thin wrapper pattern | Documentation review | Read CLI command patterns section | Examples show thin wrapper (parse → call → format), no fat controller examples except as anti-patterns | Anti-patterns may be useful; verify clearly labeled |
| **AC5.4** | Migration guide exists in `docs/guides/service-layer-migration.md` | File existence and content | Check file exists and contains E15-F12 completion note | File exists, references this feature, marks migration complete | Guide may be updated incrementally; verify accurate status |

---

## 2. API Contract Test Cases

### Repository Layer Contracts (Removal)

**Contract**: Repository methods must ONLY perform data access. No business logic.

| Contract ID | Method Removed | Replacement | Test | Validation |
|-------------|----------------|-------------|------|------------|
| **REPO-C1** | FeatureRepository.CalculateProgress(ctx, featureID) | FeatureService.GetProgress(ctx, featureID) | Attempt to call removed method | Compile error: "method not found" |
| **REPO-C2** | FeatureRepository.GetHealthStatus(ctx, featureID) | FeatureService.GetHealth(ctx, featureID) | Attempt to call removed method | Compile error: "method not found" |
| **REPO-C3** | TaskRepository.GetStatusBreakdown(ctx, featureID) | TaskService.GetStatusSummary(ctx, featureID) | Attempt to call removed method | Compile error: "method not found" |
| **REPO-C4** | EpicRepository.GetHealthStatus(ctx, epicID) | EpicService.GetHealth(ctx, epicID) | Attempt to call removed method | Compile error: "method not found" |
| **REPO-C5** | EpicRepository.CalculateProgress(ctx, epicID) | EpicService.GetProgress(ctx, epicID) | Attempt to call removed method | Compile error: "method not found" |
| **REPO-C6** | EpicRepository.CalculateProgressByKey(ctx, key) | EpicService.GetProgressByKey(ctx, key) | Attempt to call removed method | Compile error: "method not found" |

**Verification Strategy**: Code won't compile if removed methods are still called. Use `go build` as validation.

### Service Layer Contracts (Addition)

**Contract**: Service methods must implement business logic previously in repositories.

| Contract ID | New Service Method | Contract | Test Input | Expected Output |
|-------------|-------------------|----------|------------|----------------|
| **SVC-C1** | FeatureService.GetProgress(ctx, featureID) | Returns progress percentage (0.0-100.0) based on completed tasks | featureID=1 with 3/5 tasks completed | 60.0 |
| **SVC-C2** | FeatureService.GetHealth(ctx, featureID) | Returns health status string: "healthy", "warning", "critical" | featureID=1 with 2 blocked tasks | "critical" |
| **SVC-C3** | TaskService.GetStatusSummary(ctx, featureID) | Returns map of status → count | featureID=1 with 2 todo, 1 in_progress, 1 completed | {"todo": 2, "in_progress": 1, "completed": 1} |
| **SVC-C4** | EpicService.GetHealth(ctx, epicID) | Returns health status string based on feature/task impediments | epicID=1 with all features healthy | "healthy" |
| **SVC-C5** | EpicService.GetProgress(ctx, epicID) | Returns epic progress (0.0-100.0) as average of feature progress | epicID=1 with features at 50%, 75%, 100% | 75.0 |
| **SVC-C6** | EpicService.GetProgressByKey(ctx, key) | Returns epic progress by key (not ID) | key="E07" with features at 50%, 50% | 50.0 |

**Verification Strategy**: Unit tests for each service method with mocked repositories. No database required.

### CLI Command Contracts (Refactoring)

**Contract**: CLI commands must follow three-step pattern: parse → call service → format output.

| Contract ID | Command | Input | Service Method Called | Output Format | Behavioral Change |
|-------------|---------|-------|----------------------|---------------|-------------------|
| **CLI-C1** | `shark task start E07-F01-001 --agent=backend` | Valid task key, agent ID | TaskService.StartTask(ctx, "E07-F01-001", "backend") | JSON or success message | NONE (output identical) |
| **CLI-C2** | `shark feature get E07-F01` | Valid feature key | FeatureService.GetFeature(ctx, "E07-F01") | JSON or table | NONE |
| **CLI-C3** | `shark epic get E07` | Valid epic key | EpicService.GetEpic(ctx, "E07") | JSON or table with feature rollups | NONE |
| **CLI-C4** | `shark task list --status=todo` | Filter by status | TaskService.ListTasks(ctx, filters) | JSON array or table | NONE |
| **CLI-C5** | `shark task next --agent=backend` | Agent type filter | TaskService.GetNextTask(ctx, filters) | JSON or task details | NONE |

**Verification Strategy**: Run existing CLI integration tests (if they exist) or manual smoke tests comparing before/after output.

---

## 3. Component Test Strategy

### Component 1: Global Service Accessors (`internal/cli/services_global.go`)

**Purpose**: Provide easy service access for CLI commands with lazy initialization and shared dependencies.

**Test Strategy**:
- **Unit tests**: Verify accessor functions return correct service types
- **Integration tests**: Verify accessors reuse DB/workflow singletons
- **Negative tests**: Verify panic on DB failure (fail-fast for CLI)

**Key Test Cases**:

| Test Case | Approach | Assertion |
|-----------|----------|-----------|
| GetTaskService returns TaskService instance | Call GetTaskService(), check type | Type is *services.TaskService |
| GetFeatureService returns FeatureService instance | Call GetFeatureService(), check type | Type is *services.FeatureService |
| GetEpicService returns EpicService instance | Call GetEpicService(), check type | Type is *services.EpicService |
| Accessors reuse DB connection | Call GetTaskService() twice, inspect DB instances | Same DB instance used |
| Accessors reuse workflow service | Call GetTaskService() twice, inspect workflow service | Same workflow instance used |
| Accessor panics on DB failure | Mock DB error, call GetTaskService() | Panic with "failed to get database" message |

**Coverage Target**: 100% (critical infrastructure code)

---

### Component 2: Repository Layer (Business Logic Removal)

**Purpose**: Ensure repositories are pure data access (no business logic).

**Test Strategy**:
- **Existing repository tests**: Update to remove business logic method tests
- **New repository tests**: Add tests for any new CRUD methods introduced
- **Code review**: Manual inspection of repository files

**Key Test Cases**:

| Test Case | Approach | Assertion |
|-----------|----------|-----------|
| FeatureRepository contains only CRUD methods | List all public methods | Only: Create, Get, Update, Delete, List, GetByKey |
| TaskRepository contains only CRUD methods | List all public methods | Only: Create, Get, Update, Delete, List, GetByKey, UpdateStatus |
| EpicRepository contains only CRUD methods | List all public methods | Only: Create, Get, Update, Delete, List, GetByKey |
| Repository methods return raw models | Inspect returned model fields | No Progress, Health, or calculated fields |
| Repository tests use real database | Check test setup | Tests call test.GetTestDB() |

**Coverage Target**: No change (maintain existing coverage, ~80-90%)

---

### Component 3: Service Layer (Business Logic Addition)

**Purpose**: Centralize all business logic in services.

**Test Strategy**:
- **Unit tests with mocked repositories**: Test business logic without database
- **Table-driven tests**: Cover edge cases (empty data, 100% completion, etc.)
- **Error path testing**: Verify error handling and propagation

**Key Test Cases**:

| Test Case | Approach | Assertion |
|-----------|----------|-----------|
| FeatureService.GetProgress calculates correctly | Mock repo with 3/5 completed tasks | Returns 60.0 |
| FeatureService.GetProgress handles empty feature | Mock repo with 0 tasks | Returns 0.0 (not error) |
| FeatureService.GetHealth identifies blocked tasks | Mock repo with blocked tasks | Returns "critical" |
| TaskService.GetStatusSummary aggregates correctly | Mock repo with mixed statuses | Map has correct counts |
| EpicService.GetProgress averages feature progress | Mock repo with features at 50%, 100% | Returns 75.0 |
| Service methods wrap repository errors | Mock repo error | Error wrapped with business context |

**Coverage Target**: 80%+ for new methods

---

### Component 4: CLI Commands (Thin Wrapper Refactoring)

**Purpose**: Ensure commands only parse, call service, format output.

**Test Strategy**:
- **Existing CLI tests must pass unchanged**: Regression prevention
- **Code review**: Verify three-step pattern in all commands
- **Line count verification**: Automated check for size reduction

**Key Test Cases**:

| Test Case | Approach | Assertion |
|-----------|----------|-----------|
| runTaskStart calls TaskService.StartTask | Review code | No direct repository calls, only svc.StartTask() |
| runTaskStart has no business logic | Review code | No validation loops, no calculations, no workflow checks |
| runTaskStart parses arguments | Review code | Args parsed before service call |
| runTaskStart formats output | Review code | JSON or table formatting after service call |
| All task commands follow pattern | Review all handlers | Consistent three-step structure |

**Coverage Target**: Maintain existing CLI test coverage (no new tests required)

---

## 4. Integration Scenarios

### Scenario 1: End-to-End Task Lifecycle (CLI → Service → Repository → Database)

**User Journey**: Developer starts a task, completes it, gets approval.

**Integration Points**:
- CLI command parsing
- Service layer orchestration (workflow validation, dependency checks)
- Repository data access
- Database persistence

**Test Flow**:

| Step | Action | Component | Verification |
|------|--------|-----------|-------------|
| 1 | `shark task start E07-F01-001 --agent=backend` | CLI command (task.go) | Command calls cli.GetTaskService() |
| 2 | Service validates task exists | TaskService.StartTask() | Calls taskRepo.GetByKey() |
| 3 | Service validates transition | TaskService.StartTask() | Calls workflowSvc.ValidateTransition() |
| 4 | Service updates status | TaskService.StartTask() | Calls taskRepo.UpdateStatus() |
| 5 | Repository executes SQL | TaskRepository.UpdateStatus() | Database updated |
| 6 | CLI formats output | task.go runTaskStart | Success message or JSON output |

**Expected Result**: Task status changed to "in_progress", agent recorded, no errors.

**Edge Cases**:
- Task not found → 404 error
- Invalid transition (already completed) → 422 error
- Database error → 500 error

---

### Scenario 2: Feature Progress Calculation (Service → Repository → Database)

**User Journey**: Developer gets feature details, sees progress percentage.

**Integration Points**:
- CLI command
- FeatureService.GetProgress() (NEW method)
- TaskRepository.List() (existing method)
- Database query

**Test Flow**:

| Step | Action | Component | Verification |
|------|--------|-----------|-------------|
| 1 | `shark feature get E07-F01` | CLI command (feature.go) | Command calls cli.GetFeatureService() |
| 2 | Service gets feature | FeatureService.GetFeature() | Calls featureRepo.GetByKey() |
| 3 | Service calculates progress | FeatureService.GetProgress() | Calls taskRepo.List(featureID) |
| 4 | Service counts completed tasks | FeatureService.GetProgress() | Business logic: count where status="completed" |
| 5 | Service returns progress | FeatureService.GetProgress() | Returns float64 (0.0-100.0) |
| 6 | CLI formats output | feature.go runFeatureGet | JSON or table with progress field |

**Expected Result**: Feature displayed with progress percentage (e.g., 60% if 3/5 tasks completed).

**Edge Cases**:
- Feature with 0 tasks → 0% progress
- Feature with all tasks completed → 100% progress
- Feature not found → 404 error

---

### Scenario 3: Epic Health Aggregation (Multi-Layer)

**User Journey**: Developer gets epic details, sees health status based on blocked tasks.

**Integration Points**:
- CLI command
- EpicService.GetHealth() (NEW method)
- FeatureRepository.List() (existing)
- TaskRepository.List() (existing)
- Database queries

**Test Flow**:

| Step | Action | Component | Verification |
|------|--------|-----------|-------------|
| 1 | `shark epic get E07` | CLI command (epic.go) | Command calls cli.GetEpicService() |
| 2 | Service gets epic | EpicService.GetEpic() | Calls epicRepo.GetByKey() |
| 3 | Service gets features | EpicService.GetHealth() | Calls featureRepo.List(epicID) |
| 4 | Service checks each feature's tasks | EpicService.GetHealth() | Calls taskRepo.List(featureID) for each feature |
| 5 | Service identifies blockers | EpicService.GetHealth() | Business logic: check for status="blocked" |
| 6 | Service derives health | EpicService.GetHealth() | Returns "critical" if blockers, "healthy" otherwise |
| 7 | CLI formats output | epic.go runEpicGet | JSON or table with health field |

**Expected Result**: Epic displayed with health status ("healthy", "warning", "critical").

**Edge Cases**:
- Epic with no features → "healthy"
- Epic with all blocked tasks → "critical"
- Epic with old approval tasks → "warning"

---

### Scenario 4: Service Accessor Reuse (Performance)

**User Journey**: Developer runs multiple commands in quick succession.

**Integration Points**:
- Global service accessors
- DB singleton reuse
- Workflow service singleton reuse

**Test Flow**:

| Step | Action | Component | Verification |
|------|--------|-----------|-------------|
| 1 | `shark task get E07-F01-001` | CLI command | Calls cli.GetTaskService() → creates TaskService instance A |
| 2 | Service accesses DB | TaskService.GetTask() | Uses DB connection from cli.GetDB() (singleton) |
| 3 | `shark task start E07-F01-002` | CLI command | Calls cli.GetTaskService() → creates TaskService instance B |
| 4 | Service accesses DB | TaskService.StartTask() | Uses SAME DB connection (reused singleton) |
| 5 | Verify no duplicate DB connections | Monitor DB pool | Only one DB connection created |
| 6 | Verify workflow service reused | Monitor workflow service creation | Only one workflow service instance created |

**Expected Result**: DB and workflow service are singletons (created once, reused across commands).

**Performance Expectation**: No overhead from service layer (commands complete in ≤55ms vs. baseline 50ms).

---

## 5. Non-Functional Test Requirements

### Performance Testing

**Objective**: Ensure refactoring doesn't degrade performance by more than 10%.

**Baseline Measurements** (before refactoring):
- `shark task next --agent=backend`: 50ms average
- `shark task list`: 120ms average (100 tasks)
- `shark feature get E07-F01`: 80ms average

**Performance Tests**:

| Test | Command | Baseline | Target (±10%) | Measurement Method |
|------|---------|----------|---------------|-------------------|
| PT-1 | `shark task next --agent=backend` | 50ms | 45-55ms | Benchmark tool, 100 iterations |
| PT-2 | `shark task list` | 120ms | 108-132ms | Benchmark tool, 100 iterations |
| PT-3 | `shark feature get E07-F01` | 80ms | 72-88ms | Benchmark tool, 100 iterations |
| PT-4 | `shark task start E07-F01-001` | 60ms | 54-66ms | Benchmark tool, 100 iterations |
| PT-5 | `shark epic get E07` | 150ms | 135-165ms | Benchmark tool, 100 iterations |

**Validation**: Run benchmarks before/after refactoring, compare results. If any test exceeds ±10%, investigate and optimize.

---

### Regression Testing

**Objective**: Ensure all existing tests pass unchanged (zero behavioral changes).

**Test Suite Coverage**:

| Test Suite | Location | Test Count | Coverage | Validation |
|------------|----------|------------|----------|------------|
| Repository Tests | `internal/repository/*_test.go` | ~50 | 80-90% | ✅ USE REAL DATABASE |
| Service Tests | `internal/services/*_test.go` | ~40 | 70-80% | ❌ MOCK REPOSITORIES |
| CLI Tests | `internal/cli/commands/*_test.go` | ~30 | 60-70% | ❌ MOCK SERVICES |
| Integration Tests | `internal/*_integration_test.go` | ~20 | N/A | ✅ USE REAL DATABASE |

**Regression Validation**:
1. Run `make test` before refactoring → Record baseline (all tests pass)
2. Run `make test` after each phase → Compare to baseline (must match)
3. Run `make test` after completion → Final validation (100% pass rate)

**Exit Criteria**: Zero test failures, zero test modifications, identical pass rate.

---

### Code Quality Testing

**Objective**: Reduce code duplication and improve maintainability.

**Code Quality Metrics**:

| Metric | Tool | Before | After Target | Measurement |
|--------|------|--------|--------------|-------------|
| Code Duplication | jscpd | ~30% | <5% | Run jscpd on CLI command files |
| Cyclomatic Complexity | gocyclo | High (avg 15) | Low (avg 5) | Run gocyclo on command handlers |
| Line Count | wc -l | 6,858 lines | ~1,200 lines | Measure task.go + feature.go + epic.go |
| Business Logic in Commands | Manual | ~40% | 0% | Code review audit |
| Direct Repository Calls | grep | ~50 calls | 0 calls | Grep for `repository.New*Repository()` in commands |

**Validation**: Run code quality tools before/after refactoring, verify targets met.

---

## 6. Test Execution Strategy

### Phase 1: Repository Cleanup Validation

**Timing**: After repository business logic removal

**Tests to Run**:
1. Repository unit tests (verify CRUD methods still work)
2. Service unit tests (verify new GetProgress/GetHealth methods work)
3. Verify removed methods cause compile errors (negative test)

**Exit Criteria**:
- All repository tests pass
- All service tests pass
- Code compiles without errors (no calls to removed methods)

---

### Phase 2: Service Accessor Validation

**Timing**: After global accessor creation

**Tests to Run**:
1. Unit tests for GetTaskService(), GetFeatureService(), GetEpicService()
2. Verify accessors reuse DB/workflow singletons
3. Verify panic on DB failure

**Exit Criteria**:
- Accessor unit tests pass
- DB singleton behavior confirmed
- Fail-fast behavior confirmed

---

### Phase 3: CLI Command Refactoring Validation

**Timing**: After each command file refactoring (task.go, feature.go, epic.go)

**Tests to Run**:
1. All existing CLI tests for that command file
2. Manual smoke tests (compare before/after output)
3. Line count verification (measure reduction)
4. Code review (verify three-step pattern)

**Exit Criteria** (per file):
- All CLI tests pass
- Output identical to baseline
- Line count target met (≤400/350/300 lines)
- No business logic in command file

---

### Phase 4: End-to-End Integration Validation

**Timing**: After all refactoring complete

**Tests to Run**:
1. Full test suite (`make test`)
2. Performance benchmarks (compare to baseline)
3. Integration scenarios (E2E flows)
4. Code quality metrics

**Exit Criteria**:
- 100% test pass rate
- Performance within ±10%
- All integration scenarios pass
- Code quality targets met

---

## 7. Risk-Based Test Prioritization

### High-Risk Areas (Test First)

| Risk Area | Risk | Mitigation Test Strategy |
|-----------|------|-------------------------|
| **Repository method removal breaks CLI** | High | Test repository cleanup in isolation, verify compile errors prevent usage |
| **Service method logic incorrect** | Medium | Unit test new service methods thoroughly with mocked repos (edge cases, error paths) |
| **CLI tests tightly coupled to implementation** | Medium | Run existing tests early, identify coupling, refactor tests to verify behavior |
| **Performance degradation** | Medium | Benchmark before refactoring, monitor after each phase |

### Medium-Risk Areas (Test During Refactoring)

| Risk Area | Risk | Mitigation Test Strategy |
|-----------|------|-------------------------|
| **Service accessors create duplicate DB connections** | Medium | Integration test to verify singleton reuse |
| **CLI output format changes** | Low | Manual smoke tests, compare JSON structure |
| **Missing service methods** | Low | Service method inventory audit before refactoring |

### Low-Risk Areas (Test After Refactoring)

| Risk Area | Risk | Mitigation Test Strategy |
|-----------|------|-------------------------|
| **Documentation outdated** | Low | Manual review of docs, verify accuracy |
| **Code duplication not reduced** | Low | Run jscpd tool, verify <5% target |

---

## 8. Test Environment Setup

### Prerequisites

1. **Database**: Test database with sample data (epics, features, tasks)
2. **Go toolchain**: Go 1.23.4+
3. **Testing tools**: `make test`, `gocyclo`, `jscpd`, benchmark tools
4. **Sample data**: E07 epic with F01 feature and 5 tasks (2 completed, 1 in_progress, 2 todo)

### Test Data Setup

```bash
# Initialize test database
make test-db

# Create sample epic/feature/tasks
shark epic create "Test Epic E07"
shark feature create E07 "Test Feature F01"
shark task create E07 F01 "Task 1" --status=completed
shark task create E07 F01 "Task 2" --status=completed
shark task create E07 F01 "Task 3" --status=in_progress
shark task create E07 F01 "Task 4" --status=todo
shark task create E07 F01 "Task 5" --status=todo
```

### Baseline Measurement

```bash
# Measure baseline performance
time shark task next --agent=backend
time shark task list
time shark feature get E07-F01

# Measure baseline line counts
wc -l internal/cli/commands/task.go
wc -l internal/cli/commands/feature.go
wc -l internal/cli/commands/epic.go

# Run baseline tests
make test | tee baseline-test-results.txt
```

---

## 9. Acceptance Criteria Traceability

**Feature-Level Acceptance Scenarios** (from PRD):

| Scenario ID | Scenario | Test Coverage | Verification Method |
|-------------|----------|---------------|---------------------|
| **Scenario 1** | Developer adds new CLI command (<200 lines, no business logic) | Story 2 AC2.4-2.6 | Code review, line count |
| **Scenario 2** | Repository layer audit (no business logic) | Story 1 AC1.1-1.6 | Code review, method existence checks |
| **Scenario 3** | CLI command size verification (82% reduction) | Story 2 AC2.1-2.3 | Line count measurement |
| **Scenario 4** | Service layer integration (100% usage) | Story 2 AC2.6, Story 3 AC3.5 | Grep for service calls, no repository calls |
| **Scenario 5** | Test suite regression check (100% pass) | Story 4 AC4.1-4.4 | `make test`, output comparison, performance benchmarks |

---

## 10. Test Deliverables

### Test Artifacts

1. **Test Results Report**: `test-results-E15-F12.md`
   - Test execution summary
   - Pass/fail counts
   - Performance benchmark results
   - Code quality metrics

2. **Line Count Metrics**: `line-count-reduction.csv`
   - Before/after line counts for task.go, feature.go, epic.go
   - Reduction percentages

3. **Code Quality Report**: `code-quality-E15-F12.md`
   - Duplication analysis (jscpd output)
   - Cyclomatic complexity (gocyclo output)
   - Repository method audit (list of removed methods)

4. **Performance Benchmarks**: `benchmarks-E15-F12.csv`
   - Baseline measurements
   - Post-refactoring measurements
   - Delta analysis (±%)

---

## 11. Exit Criteria

### Must Pass (Blocking)

- ✅ All 6 repository business logic methods removed (compile errors confirm)
- ✅ All 6 new service methods added and tested (unit tests pass)
- ✅ CLI command files reduced: task.go ≤400, feature.go ≤350, epic.go ≤300
- ✅ `make test` passes with 100% pass rate, zero test modifications
- ✅ Performance within ±10% of baseline (all benchmarks pass)
- ✅ Zero direct repository calls in CLI commands (grep confirms)
- ✅ All commands use global service accessors (grep confirms)

### Should Pass (Non-Blocking)

- Code duplication <5% (jscpd)
- Documentation updated (manual review)
- Code quality metrics improved (cyclomatic complexity reduced)

---

## 12. Test Schedule

| Phase | Duration | Tests | Deliverables |
|-------|----------|-------|--------------|
| **Phase 1: Repository Cleanup** | Week 1 | Repository tests, service tests, compile checks | Test results, removed methods list |
| **Phase 2: Service Accessors** | Week 1 | Accessor unit tests, singleton verification | Test results, accessor coverage |
| **Phase 3: CLI Refactoring** | Weeks 2-3 | CLI tests, smoke tests, line count, code review | Test results per file, line count metrics |
| **Phase 4: Integration & Completion** | Week 3 | Full test suite, benchmarks, E2E scenarios, code quality | Final test report, performance report, quality metrics |

---

**Test Plan Status**: ✅ READY FOR REVIEW
**Next Step**: Advance feature to `in_task_generation` status via `shark feature next-status E15-F12`
