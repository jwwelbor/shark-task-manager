# Requirements

**Epic**: [Service Layer Architecture Refactoring](./epic.md)

---

## Functional Requirements

### FR1: Service Layer Architecture

**Requirement**: Create a dedicated service layer in `internal/services/` that contains all business logic currently scattered across CLI commands and repositories.

**Acceptance Criteria**:
- Directory `internal/services/` contains domain services: `TaskService`, `FeatureService`, `EpicService`
- Each service has a clearly defined interface documented with godoc comments
- Services receive repositories via constructor dependency injection
- Services own all business logic: validation, orchestration, transactions, status transitions
- No business logic remains in CLI command files (only parsing, service calls, formatting)
- No business logic remains in repository files (only data access queries)

**Priority**: P0 (Foundation for all other requirements)

---

### FR2: TaskService Implementation

**Requirement**: Extract all task-related business logic from CLI commands into `TaskService`.

**Acceptance Criteria**:
- `TaskService` interface defines all task operations:
  - CRUD: `Create()`, `Get()`, `Update()`, `Delete()`
  - Lifecycle: `Start()`, `Complete()`, `Approve()`, `Reopen()`, `Block()`, `Unblock()`
  - Querying: `GetNext()`, `List()`, `GetByStatus()`, `GetByAgent()`
  - Validation: `ValidateDependencies()`, `ValidateTransition()`
  - Calculations: `GetProgress()`, `GetBlockingIssues()`
- Service methods contain complete business logic (no logic in commands)
- Service methods are unit-testable with mocked repositories
- All existing CLI commands use `TaskService` methods
- HTTP API can call same `TaskService` methods

**Priority**: P0 (Largest surface area - 2,671 lines in task.go)

**Dependencies**: FR1 (Service Layer Architecture)

---

### FR3: FeatureService Implementation

**Requirement**: Extract all feature-related business logic from CLI commands into `FeatureService`.

**Acceptance Criteria**:
- `FeatureService` interface defines all feature operations:
  - CRUD: `Create()`, `Get()`, `Update()`, `Delete()`
  - Progress: `CalculateProgress()`, `GetHealthStatus()`, `GetActionItems()`
  - Rollups: `GetTaskBreakdown()`, `GetWorkRemaining()`
  - Lifecycle: `TransitionStatus()`, `GetNextStatus()`
- Service methods integrate with existing `status.CalculationService` and `workflow.Service`
- All existing CLI commands use `FeatureService` methods
- HTTP API can call same `FeatureService` methods

**Priority**: P0 (Second largest surface area - 2,090 lines in feature.go)

**Dependencies**: FR1 (Service Layer Architecture)

---

### FR4: EpicService Implementation

**Requirement**: Extract all epic-related business logic from CLI commands into `EpicService`.

**Acceptance Criteria**:
- `EpicService` interface defines all epic operations:
  - CRUD: `Create()`, `Get()`, `Update()`, `Delete()`
  - Rollups: `GetFeatureRollup()`, `GetTaskRollup()`, `GetImpediments()`
  - Progress: `CalculateProgress()` (aggregated from features)
  - Lifecycle: `TransitionStatus()`, `GetNextStatus()`
- Service methods aggregate data from `FeatureService` and `TaskService`
- All existing CLI commands use `EpicService` methods
- HTTP API can call same `EpicService` methods

**Priority**: P0 (Third largest surface area - 1,739 lines in epic.go)

**Dependencies**: FR1, FR3 (Service Layer Architecture, FeatureService)

---

### FR5: QueryService for Cross-Entity Operations

**Requirement**: Create `QueryService` to centralize complex filtering and querying logic used across multiple entities.

**Acceptance Criteria**:
- `QueryService` provides methods for:
  - Multi-entity search: `Search(query string, filters...)`
  - Smart dispatching: `GetByKey(key string)` (auto-detects entity type)
  - Filtering: `ApplyFilters(entities, filters)` (reusable filter logic)
  - Sorting: `SortByPriority()`, `SortByStatus()`, `SortByDate()`
- Eliminates duplicated filter/query code across task/feature/epic commands
- Used by smart dispatcher commands (`get`, `list`, `status`)

**Priority**: P1 (Reduces duplication, not blocking)

**Dependencies**: FR2, FR3, FR4 (Task/Feature/Epic services)

---

### FR6: CLI Commands as Thin Wrappers

**Requirement**: Refactor all CLI command files to be thin wrappers (200-400 lines) that only parse arguments, call services, and format output.

**Acceptance Criteria**:
- No CLI command file exceeds 500 lines (excluding tests)
- Each command file has structure:
  1. Argument/flag parsing (20-50 lines)
  2. Service method call (1-5 lines)
  3. Output formatting (20-50 lines)
  4. Error handling (10-30 lines)
- No business logic in command files (only parsing and formatting)
- All commands call service layer methods
- Commands use dependency injection via `cli.GetTaskService()`, `cli.GetFeatureService()`, etc.

**Priority**: P0 (Core goal of epic)

**Dependencies**: FR2, FR3, FR4 (Services must exist before commands can use them)

**Metrics**:
- Current: 43,590 lines across 47 files (avg 927 lines/file)
- Target: ~12,000 lines across 47 files (avg 255 lines/file)
- Reduction: 72% less code in command layer

---

### FR7: Repository Layer Cleanup

**Requirement**: Remove all business logic from repository layer, making repositories pure data access.

**Acceptance Criteria**:
- Repository methods only perform database operations (SELECT, INSERT, UPDATE, DELETE)
- No progress calculation in repositories (moved to services)
- No status derivation in repositories (moved to services)
- No workflow validation in repositories (moved to services)
- Repository methods return raw data models
- Repository methods accept `context.Context` and return `(result, error)`

**Priority**: P0 (Clarifies layer boundaries)

**Dependencies**: FR2, FR3, FR4 (Services must handle logic before it's removed from repos)

**Specific Cleanups**:
- Remove `FeatureRepository.CalculateProgress()` → moved to `FeatureService`
- Remove `TaskRepository.GetStatusBreakdown()` → moved to `TaskService`
- Remove `EpicRepository.GetHealthStatus()` → moved to `EpicService`

---

### FR8: HTTP API Service Integration

**Requirement**: Wire HTTP API server to use service layer methods, achieving 100% feature parity with CLI.

**Acceptance Criteria**:
- HTTP API server (`cmd/server/`) uses same service methods as CLI
- All CLI operations have equivalent API endpoints
- API and CLI behavior is identical (same validation, same errors, same calculations)
- API tests verify feature parity with CLI
- API documentation lists all endpoints with CLI command equivalents

**Priority**: P1 (Enabler for API consumers, not blocking CLI refactoring)

**Dependencies**: FR2, FR3, FR4, FR6 (Services and CLI refactoring complete)

**API Endpoints to Implement**:
- `/api/tasks/next?agent={type}` (equivalent to `shark task next`)
- `/api/tasks/{key}/start` (equivalent to `shark task start`)
- `/api/features/{key}/progress` (equivalent to `shark feature get --json | jq .progress`)
- `/api/epics/{key}/rollup` (equivalent to `shark epic get --json | jq .feature_rollup`)

---

### FR9: Service Interface Documentation

**Requirement**: Document all service interfaces with comprehensive godoc comments and usage examples.

**Acceptance Criteria**:
- Each service method has godoc comment describing:
  - Purpose and behavior
  - Parameters and return values
  - Error conditions
  - Example usage
- Service package has overview documentation explaining architecture
- README.md updated with service layer architecture diagram
- `.claude/rules/architecture.md` updated to reflect new architecture

**Priority**: P1 (Important for maintainability, not blocking implementation)

**Dependencies**: FR2, FR3, FR4 (Services implemented)

---

## Non-Functional Requirements

### NFR1: Zero Behavior Regression

**Requirement**: All existing functionality must work identically before and after refactoring.

**Acceptance Criteria**:
- All existing tests pass without modification (except test structure improvements)
- Manual regression testing shows no behavior changes
- Integration tests verify CLI commands produce identical output
- API tests verify endpoints produce identical responses

**Validation**:
- Run full test suite: `make test`
- Compare CLI output before/after for sample operations
- No user-reported regressions in production for 30 days post-deployment

**Priority**: P0 (Non-negotiable - refactoring must not break functionality)

---

### NFR2: Test Coverage Improvement

**Requirement**: Service layer must have >80% test coverage with unit tests.

**Acceptance Criteria**:
- `TaskService` has >80% test coverage
- `FeatureService` has >80% test coverage
- `EpicService` has >80% test coverage
- Tests are unit tests with mocked repositories (not integration tests)
- Test execution time is <100ms for full service layer suite
- Coverage report shows untested edge cases

**Validation**:
- `make test-coverage` shows >80% for `internal/services/`
- Test suite runs in <100ms on CI server

**Priority**: P0 (Tests are documentation and prevent regressions)

---

### NFR3: Performance Neutrality

**Requirement**: Refactoring must not degrade performance (acceptable: slight improvement, not acceptable: >10% slower).

**Acceptance Criteria**:
- CLI command execution time unchanged ±10%
- API endpoint response time unchanged ±10%
- Memory usage unchanged ±10%
- Database query count unchanged

**Validation**:
- Benchmark CLI commands before/after: `shark task next --agent=backend` (baseline: ~50ms)
- Benchmark API endpoints before/after: `GET /api/tasks/next?agent=backend` (baseline: ~45ms)
- Memory profiling shows no leaks or significant allocations

**Priority**: P1 (Important but not blocking - minor performance trade-offs acceptable for maintainability)

---

### NFR4: Incremental Migration

**Requirement**: Refactoring must be done incrementally without long-lived feature branches.

**Acceptance Criteria**:
- Each service (Task/Feature/Epic) can be extracted independently
- Intermediate states are valid (some commands use services, others don't)
- Each PR is <500 lines of changes (excluding tests)
- Main branch is always deployable during refactoring

**Validation**:
- PR size stays <500 lines
- CI passes on main branch after each merge
- No merge conflicts from long-lived branches

**Priority**: P0 (Risk management - large refactorings fail without incremental approach)

---

### NFR5: Backward Compatibility

**Requirement**: External consumers (HTTP API clients, CLI scripts) see no breaking changes.

**Acceptance Criteria**:
- CLI command syntax unchanged (same flags, same arguments)
- CLI output format unchanged (same JSON structure, same table columns)
- API endpoint URLs unchanged
- API response schemas unchanged

**Validation**:
- Run existing CLI scripts against new version
- Run existing API integration tests against new version
- Compare JSON output schemas before/after

**Priority**: P0 (Breaking changes require major version bump and migration plan)

---

## Constraints

### C1: Codebase Constraints
- Must maintain Go 1.23.4+ compatibility
- Must not add new external dependencies (use stdlib and existing deps only)
- Must follow existing project structure (`internal/` for private packages)

### C2: Testing Constraints
- Must use existing test utilities in `internal/test/`
- Cannot use real database in service layer tests (mock repositories only)
- Must maintain or improve test execution speed

### C3: Documentation Constraints
- Must update `.claude/rules/architecture.md` with new patterns
- Must update `CLAUDE.md` with service layer guidance
- Must add migration guide for contributors

### C4: Timeline Constraints
- Must complete refactoring within 7 weeks
- Each phase must be independently shippable
- Cannot block new feature development for >2 weeks

---

## Dependencies

### Internal Dependencies
- Existing services (`workflow.Service`, `status.CalculationService`) must be integrated
- Existing repositories continue to work during migration
- Existing test utilities in `internal/test/` must support service testing

### External Dependencies
- No new external libraries required
- No changes to `.sharkconfig.json` schema
- No database schema migrations required

---

## Acceptance Criteria Summary

This epic is considered complete when:
1. ✅ All 47 CLI command files are <500 lines (FR6)
2. ✅ `TaskService`, `FeatureService`, `EpicService` exist and are used by all commands (FR2-4)
3. ✅ Repository layer contains only data access (FR7)
4. ✅ HTTP API uses service layer (FR8)
5. ✅ All existing tests pass (NFR1)
6. ✅ Service layer has >80% test coverage (NFR2)
7. ✅ Performance is within ±10% of baseline (NFR3)
8. ✅ Documentation updated (FR9)

---

*See also*: [Success Metrics](./success-metrics.md), [User Journeys](./user-journeys.md)
