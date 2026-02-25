# UAT Acceptance Plan - Epic E15: Service Layer Architecture Refactoring

**Epic:** E15 - Service Layer Architecture Refactoring
**Status:** Strategic UAT Planning (Epic-Level)
**Generated:** 2026-02-16
**Type:** Architecture Refactoring (Non-Functional Epic)

---

## Executive Summary

This UAT plan validates that Epic E15 successfully transforms Shark's architecture from a fat-controller pattern (20,952 LOC in commands with 40-45% business logic) to a clean three-layer architecture with a dedicated service layer. The epic delivers measurable improvements: 65% code reduction in command layer, 60-75% development efficiency gains, and 100% HTTP API feature parity with CLI.

**Strategic Validation Goals:**
1. **Architecture Quality**: Service layer contains 90%+ of business logic (not commands or repositories)
2. **Developer Efficiency**: Development time reduced 60-75% for common workflows (measured via user journey scenarios)
3. **Zero Regression**: All existing functionality works identically after refactoring
4. **API Enablement**: HTTP API achieves 100% feature parity with CLI using shared service layer

---

## Epic Context

**Epic Goal:** Extract business logic from CLI commands into a proper service/application layer, enabling code reuse across CLI and HTTP API while improving testability and eliminating duplication.

**Business Value:** High - This refactoring is a prerequisite for HTTP API feature parity with CLI, which blocks integration with AI orchestrators, CI/CD pipelines, and web interfaces. Eliminating architectural debt now reduces future development costs by 40-60%.

**Scope:**
- **In Scope**: Service layer creation (Task/Feature/Epic services), CLI command slimming (47 files to <500 LOC), repository cleanup (pure data access), HTTP API integration
- **Out of Scope**: New CLI commands, database schema changes, new test frameworks, new external dependencies

**Related Epics:**
- **E12 (Repository DB Interface)**: Sequenced AFTER E15 (E15 removes 156 LOC from repositories first)
- **E16 (Multi-Level Workflow)**: Sequenced AFTER E15-F04/F05 (workflow integration into service layer)

---

## 1. Acceptance Scenarios Derived from User Journeys

These scenarios validate the 5 user journeys from `user-journeys.md`, proving efficiency gains match targets.

### Journey 1: Adding a New CLI Command (Developer Efficiency)

**Baseline:** 8 hours (4h coding, 2h testing, 2h review)
**Target:** 2 hours (1h coding, 0.5h testing, 0.5h review)
**Efficiency Gain:** 75% reduction

**Scenario 1.1: Add "Task Archive" Command via Service Layer**

**Persona:** Shark Core Developer (Human)

**Preconditions:**
- E15 Phase 1-4 complete (TaskService exists, CLI commands slim)
- Developer has understanding of TaskService interface

**Steps:**
1. Developer reads `internal/services/task_service.go` interface documentation (200 lines)
2. Adds `Archive(ctx context.Context, taskKey string) error` method to TaskService (30 LOC implementation)
3. Implements business logic in service:
   - Validate task status is "completed"
   - Call `archiveService.MoveToArchive()` for file operation
   - Update database with `repo.UpdateArchived()`
4. Creates CLI command wrapper in `internal/cli/commands/task_archive.go` (80 LOC):
   - Parse arguments
   - Call `taskService.Archive()`
   - Format output (JSON or success message)
5. Writes service unit test with mocked repository (40 LOC)
6. Submits PR with 150 new lines, zero duplication

**Expected Outcomes:**
- ✅ Development time: 2 hours (measured from start to PR submission)
- ✅ Lines of code: 150 total (30 service + 80 command + 40 test)
- ✅ Duplication: 0% (no copied boilerplate)
- ✅ Test execution: Service test runs in <10ms (vs 500ms integration test)
- ✅ HTTP API: Archive capability automatically available (service is reusable)

**Success Metrics:**
- [ ] Development time ≤2 hours (75% improvement validated)
- [ ] LOC ≤200 (vs 350 baseline)
- [ ] Zero duplication detected (grep for copied patterns)
- [ ] Service test passes in <10ms
- [ ] HTTP API endpoint works with same service method

**Validation Method:**
- Track time from task start to PR submission (stopwatch)
- Count LOC added in PR (git diff --stat)
- Run duplication analysis (grep for known patterns)
- Benchmark service test execution time
- Verify HTTP API endpoint exists and works

---

### Journey 2: Understanding Business Logic (AI Agent Efficiency)

**Baseline:** 75 minutes (AI research + developer verification)
**Target:** 5 minutes (AI reads centralized service)
**Efficiency Gain:** 93% reduction

**Scenario 2.1: AI Agent Answers "How Does Task Dependency Checking Work?"**

**Persona:** AI Code Agent (Cursor/Claude)

**Preconditions:**
- E15 Phase 1 complete (TaskService exists)
- Dependency checking logic extracted to TaskService

**Steps:**
1. AI loads `internal/services/task_service.go` (200 LOC total file)
2. AI finds `ValidateDependencies()` method (lines 145-180, 35 LOC)
3. AI reads complete logic in one place:
   - Check if dependencies exist
   - Check if dependencies are completed
   - Return validation error if blocked
4. AI provides accurate answer with code reference
5. AI suggests correct change location: `TaskService.ValidateDependencies()`

**Expected Outcomes:**
- ✅ Time to answer: <5 minutes (AI can load full context)
- ✅ Context loading: Success (200 LOC fits in AI context window)
- ✅ Answer accuracy: 100% (complete logic in one location)
- ✅ Suggested change location: Correct (points to service, not command)

**Success Metrics:**
- [ ] AI loads full service file without context overflow warning
- [ ] AI identifies correct method in <2 minutes
- [ ] AI answer is complete (no missing logic from other files)
- [ ] AI suggests service layer change (not command layer)
- [ ] Developer verification: AI answer is 100% accurate

**Validation Method:**
- Run AI query: "How does task dependency checking work?"
- Measure time from query to complete answer
- Verify AI loads only 1 file (not 3 files in baseline)
- Validate answer completeness against actual implementation
- Developer peer review confirms accuracy

---

### Journey 3: Writing Tests for Business Logic (Test Efficiency)

**Baseline:** 110 lines test code, 450ms execution
**Target:** 11 lines test code, 0.5ms execution
**Efficiency Gain:** 90% less code, 900x faster

**Scenario 3.1: Test "Cannot Complete Already-Completed Task"**

**Persona:** Shark Core Developer / Test Engineer

**Preconditions:**
- E15 Phase 1 complete (TaskService exists with unit tests)
- Mock repository patterns established

**Steps:**
1. Create mock repository returning task with "completed" status (8 LOC):
   ```go
   mockRepo := &MockTaskRepository{
       GetByKeyFunc: func(ctx, key) (*models.Task, error) {
           return &models.Task{Status: "completed"}, nil
       },
   }
   ```
2. Call `taskService.Complete(ctx, taskKey)` (1 LOC)
3. Assert error is `ErrAlreadyCompleted` (2 LOC)
4. Run test (executes in <1ms in-memory)

**Expected Outcomes:**
- ✅ Test code: 11 lines total
- ✅ Test execution: <1ms (no database I/O)
- ✅ Test isolation: Business logic tested without CLI/database
- ✅ Test maintainability: No brittleness from database schema changes

**Success Metrics:**
- [ ] Test code ≤15 LOC (vs 110 baseline)
- [ ] Test execution <5ms (vs 450ms baseline)
- [ ] No database initialization in test
- [ ] Test passes with mocked repository only
- [ ] Test is resilient to database schema changes

**Validation Method:**
- Count lines in test function
- Benchmark test execution time (go test -bench)
- Verify no `test.GetTestDB()` call in test
- Run test suite after database schema change (test still passes)

---

### Journey 4: HTTP API Feature Parity (Integration Efficiency)

**Baseline:** 6 hours (4h workarounds, 2h debugging)
**Target:** 1 hour (0.5h docs, 0.5h implementation)
**Efficiency Gain:** 83% reduction

**Scenario 4.1: Build CI/CD Integration for "Get Next Task for Backend Agent"**

**Persona:** HTTP API Consumer / DevOps Engineer

**Preconditions:**
- E15 Phase 5 complete (HTTP API wired to service layer)
- TaskService.GetNextTask() method exists

**Steps:**
1. Check HTTP API documentation - `/api/tasks/next?agent=backend` endpoint documented
2. Call `GET /api/tasks/next?agent=backend`
3. Receive JSON response with same data as `shark task next --agent=backend --json`
4. Verify API uses same `TaskService.GetNextTask()` method as CLI
5. All business logic (filtering, priority sorting, dependency checking) works identically
6. Integration complete

**Expected Outcomes:**
- ✅ API endpoint exists and documented
- ✅ API response matches CLI JSON output exactly
- ✅ Business logic identical (same service method)
- ✅ No workarounds needed (no client-side filtering)
- ✅ Integration time: 1 hour (vs 6 hours baseline)

**Success Metrics:**
- [ ] API endpoint `/api/tasks/next?agent=backend` exists
- [ ] API response schema matches CLI `--json` output
- [ ] API uses TaskService (not duplicate logic)
- [ ] Integration time ≤1 hour (measured)
- [ ] Zero client-side workarounds needed

**Validation Method:**
- Curl API endpoint: `curl http://localhost:8080/api/tasks/next?agent=backend`
- Compare JSON response to CLI: `shark task next --agent=backend --json`
- Check API handler code uses `TaskService.GetNextTask()`
- Time integration from docs to working code
- Verify no client-side filtering logic needed

---

### Journey 5: Maintaining Architecture Over Time (Review Efficiency)

**Baseline:** 3 review cycles over 5 days
**Target:** 1 review cycle (same day)
**Efficiency Gain:** 80% reduction in review time

**Scenario 5.1: Review PR Adding Epic Progress Calculation**

**Persona:** Shark Maintainer

**Preconditions:**
- E15 complete (EpicService exists with clear interface)
- Developer familiar with service layer pattern

**Steps:**
1. Developer reads existing `EpicService` interface (already exists post-E15)
2. Adds `CalculateProgress(ctx, epicKey) (float64, error)` method to EpicService (35 LOC)
3. Implements business logic in service method
4. Updates CLI command to call `epicService.CalculateProgress()` (5 LOC)
5. Maintainer reviews PR:
   - Architecture is correct by construction (service layer pattern followed)
   - Review focuses on business logic correctness, not architecture
6. PR approved in 1 review cycle (same day)

**Expected Outcomes:**
- ✅ Developer knows where code belongs (service layer is obvious)
- ✅ No architecture feedback needed (pattern is self-evident)
- ✅ Review focuses on feature quality (not structure)
- ✅ PR merged in 1 cycle (same day vs 5 days)
- ✅ Contributor frustration: None (clear pattern to follow)

**Success Metrics:**
- [ ] Developer places logic in service layer (not command)
- [ ] Maintainer gives zero architecture feedback
- [ ] PR review focuses on business logic (not structure)
- [ ] PR merged in ≤1 review cycle
- [ ] Time to merge ≤1 day (vs 5 days baseline)

**Validation Method:**
- Review PR comments for architecture feedback (count = 0)
- Measure PR review cycles (count ≤1)
- Track time from PR open to merge (≤24 hours)
- Survey developer: "Was the architecture clear?" (yes)
- Survey maintainer: "Did you need to explain architecture?" (no)

---

## 2. Success Metrics Validation Plan

These metrics from `success-metrics.md` are validated with specific measurement methods and pass/fail criteria.

### Architecture Quality Metrics

#### AQ1: Code Reduction in Command Layer

**Baseline:** 43,590 LOC (across 47 files), average 927 LOC, largest files 2,671 LOC
**Target:** <15,000 LOC total, <320 LOC average, <500 LOC max per file
**Success Threshold:** >50% reduction acceptable, >65% reduction excellent

**Measurement Method:**
```bash
# Count lines before E15
find internal/cli/commands -name "*.go" ! -name "*_test.go" -exec wc -l {} + | tail -1
# Baseline: 43,590

# Count lines after E15
find internal/cli/commands -name "*.go" ! -name "*_test.go" -exec wc -l {} + | tail -1
# Target: <15,000

# Calculate reduction percentage
echo "scale=2; (43590 - $(find internal/cli/commands -name \"*.go\" ! -name \"*_test.go\" -exec wc -l {} + | tail -1 | awk '{print $1}')) / 43590 * 100" | bc
# Target: >65%
```

**Pass Criteria:**
- [ ] Total LOC ≤15,000 (65% reduction achieved)
- [ ] Average file size ≤320 LOC
- [ ] Max file size ≤500 LOC (no fat controllers remain)
- [ ] Top 3 files (task.go, feature.go, epic.go) ≤500 LOC each

---

#### AQ2: Service Layer Test Coverage

**Baseline:** EpicService 85.7%, FeatureService 82.3%, TaskService 0% (doesn't exist)
**Target:** All services >80% coverage
**Success Threshold:** >70% acceptable, >80% excellent

**Measurement Method:**
```bash
# Generate coverage report for services
go test -coverprofile=coverage.out ./internal/services/...
go tool cover -func=coverage.out | grep total

# Individual service coverage
go test -coverprofile=coverage.out ./internal/services/task_service.go
go tool cover -func=coverage.out | grep task_service.go

# HTML coverage report
make test-coverage
# Open coverage.html, navigate to internal/services/
```

**Pass Criteria:**
- [ ] TaskService coverage >80%
- [ ] FeatureService coverage ≥82.3% (maintained)
- [ ] EpicService coverage ≥85.7% (maintained)
- [ ] Overall internal/services/ coverage >80%
- [ ] No critical paths uncovered (<50% coverage in key methods)

---

#### AQ3: Duplication Elimination

**Baseline:** 351 repository instantiations, 32 key lookup duplications, 96 progress calculations
**Target:** <10 repository instantiations, 1 key lookup implementation, 3 progress implementations (Task/Feature/Epic)
**Success Threshold:** >60% reduction acceptable, >80% reduction excellent

**Measurement Method:**
```bash
# Count repository instantiations (should be ~0 in commands after E15)
grep -r "repository\.New" internal/cli/commands/*.go | wc -l
# Baseline: 351, Target: <10

# Count key lookup pattern (should be centralized in services)
grep -r "GetByKey.*context\.Context" internal/cli/commands/*.go | wc -l
# Baseline: 32, Target: <5

# Count progress calculation occurrences
grep -r "CalculateProgress" internal | wc -l
# Baseline: 96, Target: <30 (services + tests)

# Calculate duplication reduction
echo "scale=2; (351 - $(grep -r \"repository\.New\" internal/cli/commands/*.go | wc -l)) / 351 * 100" | bc
# Target: >80%
```

**Pass Criteria:**
- [ ] Repository instantiations in commands ≤10 (97% reduction)
- [ ] Key lookup pattern in commands ≤5 (84% reduction)
- [ ] Progress calculation centralized in 3 services (not scattered)
- [ ] Error handling pattern consolidated (HandleServiceError utility)
- [ ] Overall duplication reduction >80%

---

#### AQ4: Business Logic in Service Layer

**Baseline:** Commands 40-45%, Services 5%, Repositories 10% business logic
**Target:** Commands 0%, Services 90%+, Repositories 0%
**Success Threshold:** >95% acceptable, 100% excellent

**Measurement Method:**
Manual code review checklist per PR:
- [ ] No `if` statements for business rules in commands (only formatting conditionals)
- [ ] No calculations in commands (delegates to services)
- [ ] No database transactions in commands (orchestrated by services)
- [ ] No progress/health calculations in repositories
- [ ] All validation logic in services via `workflow.Service`

**Pass Criteria:**
- [ ] 100% of business logic in service layer (commands are thin wrappers)
- [ ] Commands only contain: parsing, service calls, output formatting
- [ ] Repositories only contain: SQL queries, data access
- [ ] No business rules in commands (verified via code review)
- [ ] No business logic in repositories (verified via code review)

---

### Developer Experience Metrics

#### DX1: Time to Add New CLI Command

**Baseline:** 8 hours average
**Target:** 2 hours average
**Success Threshold:** >50% reduction acceptable, >75% excellent

**Measurement Method:**
Track 3-5 new commands added post-E15:
- Measure PR creation time to merge time (GitHub metadata)
- Count LOC added (git diff --stat)
- Developer survey: "How long did this take vs previous commands?"

**Pass Criteria:**
- [ ] Average development time ≤3 hours (>50% reduction)
- [ ] At least 3 of 5 commands achieve ≤2 hours (75% target)
- [ ] Average LOC per command ≤200 (vs 350 baseline)
- [ ] Zero duplication detected in new commands
- [ ] Developer survey: >80% report time savings

---

#### DX2: PR Review Cycle Time

**Baseline:** 2-3 cycles, 3-5 days to merge, 60% get architecture feedback
**Target:** 1 cycle, 1 day to merge, <10% get architecture feedback
**Success Threshold:** >50% reduction acceptable, >70% excellent

**Measurement Method:**
- GitHub PR metrics: time from open to first review, time to merge
- Review comment analysis: count architecture-related comments
- Maintainer survey: "How often do you give architecture feedback?"

**Pass Criteria:**
- [ ] Average review cycles ≤2 (vs 3 baseline)
- [ ] Average time to merge ≤2 days (vs 5 baseline)
- [ ] Architecture feedback <20% of PRs (vs 60% baseline)
- [ ] Maintainer reports reduced review burden (survey)
- [ ] 70% reduction in review time achieved (≤1.5 cycles, ≤1.5 days)

---

#### DX3: Test Execution Speed

**Baseline:** CLI integration tests 450ms/test, total suite 45s
**Target:** Service unit tests <1ms/test, total suite <100ms, full suite <60s
**Success Threshold:** Service <500ms acceptable, <100ms excellent

**Measurement Method:**
```bash
# Time service layer tests
time go test -v ./internal/services/...
# Target: <100ms total

# Time full test suite
make test | grep "ok\|FAIL" | awk '{sum+=$3} END {print sum "s"}'
# Target: <60s (maintains or improves)

# Benchmark individual test
go test -bench=BenchmarkTaskService_StartTask ./internal/services/
# Target: <1000 ns/op (1μs)
```

**Pass Criteria:**
- [ ] Service layer test suite <100ms total execution
- [ ] Individual service tests <5ms each
- [ ] Full test suite ≤60s (maintains current speed)
- [ ] No database initialization in service tests
- [ ] Service tests 100x faster than integration tests (0.5ms vs 50ms)

---

### System Health Metrics

#### SH1: HTTP API Feature Parity

**Baseline:** 15/47 endpoints (32% coverage)
**Target:** 47/47 endpoints (100% coverage)
**Success Threshold:** >80% acceptable, 100% excellent

**Measurement Method:**
```bash
# Count CLI commands
ls internal/cli/commands/*.go | grep -v test | wc -l
# Baseline: 47

# Count API endpoints
grep -r "router\." cmd/server/*.go | grep -E "(GET|POST|PUT|DELETE)" | wc -l
# Baseline: 15, Target: 47

# API parity percentage
echo "scale=2; $(grep -r \"router\.\" cmd/server/*.go | grep -E \"(GET|POST|PUT|DELETE)\" | wc -l) / 47 * 100" | bc
# Target: 100%
```

**Pass Criteria:**
- [ ] All 47 CLI commands have API equivalents
- [ ] API and CLI use same service methods (no duplication)
- [ ] API responses match CLI `--json` output schemas
- [ ] Feature parity documented (API docs list CLI equivalents)
- [ ] Zero feature gaps between API and CLI

---

#### SH2: Regression Rate

**Baseline:** Expected 60-150 bugs for 30K LOC change (industry average)
**Target:** <15 total regressions, 0 critical
**Success Threshold:** <30 acceptable, <15 excellent

**Measurement Method:**
- Track GitHub issues tagged "regression" and "E15"
- Monitor user reports of broken functionality
- Track test failures in main branch during refactoring

**Pass Criteria:**
- [ ] Total regressions ≤15 (>75% better than industry average)
- [ ] Critical regressions (data loss, crashes) = 0
- [ ] Regressions fixed within 24 hours of discovery
- [ ] No regressions reported in production after 30 days
- [ ] All existing tests pass (NFR1 zero regression requirement)

---

#### SH3: Performance Stability

**Baseline:** `shark task next` 50ms, `shark feature get` 35ms, API `/tasks/next` 45ms
**Target:** All operations ±10% of baseline
**Success Threshold:** ±20% acceptable, ±10% excellent

**Measurement Method:**
```bash
# Benchmark before E15 (baseline)
for i in {1..100}; do time shark task next --agent=backend >/dev/null 2>&1; done | awk '{sum+=$1; count++} END {print sum/count}'
# Baseline: 50ms

# Benchmark after E15
for i in {1..100}; do time shark task next --agent=backend >/dev/null 2>&1; done | awk '{sum+=$1; count++} END {print sum/count}'
# Target: 45-55ms (±10%)

# Go benchmark
go test -bench=. ./internal/services/task_service_test.go
# Measure service layer overhead
```

**Pass Criteria:**
- [ ] `shark task next` execution time: 45-55ms (±10% of 50ms baseline)
- [ ] `shark feature get` execution time: 31.5-38.5ms (±10% of 35ms)
- [ ] API `/tasks/next` execution time: 40.5-49.5ms (±10% of 45ms)
- [ ] No operation slower by >20%
- [ ] Service layer adds <1ms overhead (negligible)

---

## 3. Cross-Epic Integration Test Scenarios

These scenarios validate integration with E12 (Repository Layer) and E16 (Multi-Level Workflow).

### Integration Scenario 1: E12 Repository DB Interface Migration

**Epics:** E15 (Service Layer) → E12 (Repository DB Interface)
**Sequencing:** E15 completes first (removes business logic), then E12 (changes DB interface)

**Scenario:**
1. E15 completes: Business logic moved from repositories to services (106 LOC removed)
2. Verify repositories contain only data access (CRUD, queries)
3. E12 starts: Change repository constructors from `*sql.DB` to `db.Database` interface
4. Verify services continue working (they call repositories via interface)
5. Verify zero breaking changes in service layer (orthogonal concerns)

**Expected Outcomes:**
- ✅ E15 cleanup makes E12 migration easier (fewer repository methods to change)
- ✅ Service layer unaffected by E12 (services call repos via interface)
- ✅ No merge conflicts (E15 doesn't modify repo constructors)

**Success Metrics:**
- [ ] E15 removes 106 LOC from repositories before E12 starts
- [ ] E12 migration touches only repository constructors (not service layer)
- [ ] All service tests pass after E12 migration (no breaking changes)
- [ ] Zero regression from E15→E12 handoff

---

### Integration Scenario 2: E16 Multi-Level Workflow System

**Epics:** E15 (Service Layer) → E16 (Multi-Level Workflow)
**Sequencing:** E15-F04/F05 complete (Epic/FeatureService expansion), then E16 starts

**Scenario:**
1. E15 completes: EpicService and FeatureService exist with workflow integration
2. Verify services use `workflow.Service` for status validation (pattern established)
3. E16 starts: Add epic/feature workflow to `.sharkconfig.json`
4. E16 integrates workflow logic into existing EpicService/FeatureService methods
5. Verify no command-layer changes needed (workflow logic in services)

**Expected Outcomes:**
- ✅ E15 creates service layer where E16 workflow logic belongs
- ✅ E16 leverages existing service methods (no duplication)
- ✅ Workflow integration is service-layer only (not CLI commands)

**Success Metrics:**
- [ ] E15 creates EpicService.TransitionStatus() and FeatureService.TransitionStatus()
- [ ] E16 extends these methods with epic/feature workflow (not commands)
- [ ] Zero CLI command changes for E16 (workflow is in services)
- [ ] E16 reuses E15 service layer pattern (no new architecture)

---

### Integration Scenario 3: HTTP API Parity (Potential E17)

**Epics:** E15 (Service Layer) → E17 (HTTP API Parity)
**Dependency:** E15 enables E17 (service layer provides reusable business logic)

**Scenario:**
1. E15 completes: TaskService, FeatureService, EpicService exist with all business logic
2. E17 starts: Create HTTP endpoints that call service methods
3. Verify API uses same service methods as CLI (zero duplication)
4. Verify API responses match CLI `--json` output (same data models)
5. Verify 100% feature parity (all CLI operations available via API)

**Expected Outcomes:**
- ✅ E15 service layer eliminates need for duplicate API logic
- ✅ E17 is HTTP plumbing only (no business logic implementation)
- ✅ API and CLI behavior guaranteed identical (same code path)

**Success Metrics:**
- [ ] E17 creates API endpoints without duplicating business logic
- [ ] API handlers call TaskService/FeatureService/EpicService (not repositories)
- [ ] API feature parity 100% (47/47 commands have endpoints)
- [ ] API and CLI responses match exactly (same service methods)

---

## 4. Risk-Based Test Priorities

These priorities are based on research findings and technical feasibility review.

### Critical Priority (P0) - Must Pass First

**TaskService Extraction (Highest Risk Area)**
- **Risk:** 2,664 LOC command → 400 LOC command + 1,865 LOC service (85% reduction)
- **Complexity:** High (30+ methods, dependencies, workflow integration)
- **Impact:** 32.6% of all command code in task.go

**Test Scenarios:**
1. **TaskService CRUD Operations**
   - Create, GetTask, UpdateTask, DeleteTask work correctly
   - Validation logic moved from command to service
   - Repository called with correct parameters
2. **TaskService Lifecycle Operations**
   - Start, Complete, Approve, Reopen, Block, Unblock
   - Workflow integration (status transitions)
   - History tracking (task_history table)
3. **TaskService Querying**
   - GetNextTask with agent filtering
   - List with status/epic/feature filters
   - Dependency checking (ValidateDependencies)

**Success Gate:** All task commands work identically before/after extraction. Zero regressions in task functionality.

---

### High Priority (P1) - Core Architecture Goals

**Repository Cleanup (Remove Business Logic)**
- **Risk:** 106 LOC progress calculation + 200 LOC status derivation to remove
- **Complexity:** Medium (logic exists, needs to move)
- **Impact:** Clarifies layer boundaries (repos = pure data access)

**Test Scenarios:**
1. **Progress Calculation Moved to Services**
   - FeatureService.GetProgress() replaces FeatureRepository.CalculateProgress()
   - EpicService.GetProgress() replaces EpicRepository.CalculateProgress()
   - Calculation results identical to original
2. **Status Derivation in Services**
   - Services wrap status.CalculationService (not commands or repos)
   - Repositories return raw counts (not calculated percentages)
3. **Repository Purity Verification**
   - No `if` statements for business rules in repos
   - Only CRUD and query methods remain
   - No workflow or validation logic in repos

**Success Gate:** Repositories contain only data access. All business logic in services. Zero regressions in progress/status calculations.

---

**CLI Command Slimming (47 Files to <500 LOC)**
- **Risk:** High volume (47 files to refactor)
- **Complexity:** Medium (straightforward pattern: parse → call service → format)
- **Impact:** 65% code reduction target (20,952 → ~7,000 LOC)

**Test Scenarios:**
1. **Command Size Verification**
   - All 47 command files <500 LOC (excluding tests)
   - Average file size ≤320 LOC
   - Largest files (task.go, feature.go, epic.go) <500 LOC
2. **Command Responsibility Check**
   - Commands only parse arguments (20-50 LOC)
   - Commands only call services (1-5 LOC per operation)
   - Commands only format output (20-50 LOC)
   - No business logic in commands (manual code review)
3. **CLI Interface Compatibility**
   - Same flags and arguments (no breaking changes)
   - Same JSON output schemas (--json flag)
   - Same exit codes (0, 1, 2, 3)
   - Same table formatting

**Success Gate:** No command exceeds 500 LOC. All commands are thin wrappers. Zero breaking changes to CLI interface.

---

### Medium Priority (P2) - Post-MVP

**QueryService for Cross-Entity Operations**
- **Risk:** Low (consolidates existing dispatch logic)
- **Complexity:** Low-Medium (key parsing already exists)
- **Impact:** Reduces duplication in smart dispatchers (get, list, status)

**Test Scenarios:**
1. **Smart Key Detection**
   - Detect epic keys: E07, e07, E07-epic-name
   - Detect feature keys: E07-F01, F01, E07-F01-feature-name
   - Detect task keys: E07-F01-001, T-E07-F01-001, E07-F01-001-task-name
2. **QueryService.GetByKey()**
   - Auto-dispatches to TaskService.GetTask() for task keys
   - Auto-dispatches to FeatureService.GetFeature() for feature keys
   - Auto-dispatches to EpicService.GetEpic() for epic keys
3. **QueryService.List() Smart Dispatch**
   - List epics (no args)
   - List features (epic arg only)
   - List tasks (epic + feature args)

**Success Gate:** Smart dispatchers consolidated in QueryService. Commands use QueryService (not duplicate dispatch logic).

---

**HTTP API Integration (Feature Parity)**
- **Risk:** Low (service layer enables this)
- **Complexity:** Low-Medium (HTTP plumbing, JSON serialization)
- **Impact:** Unblocks AI orchestrators, CI/CD, web dashboards

**Test Scenarios:**
1. **API Endpoint Coverage**
   - All 47 CLI commands have API equivalents
   - API documentation lists CLI command mappings
2. **API-Service Integration**
   - API handlers call TaskService/FeatureService/EpicService
   - No duplicate business logic in API layer
3. **Response Parity**
   - API JSON matches CLI `--json` output exactly
   - Same validation errors (consistent behavior)
4. **API Functionality**
   - GET /api/tasks/next?agent=backend works
   - POST /api/tasks/{key}/start works
   - GET /api/features/{key}/progress works
   - All CRUD operations via API

**Success Gate:** 100% API feature parity. API and CLI use same service methods. Zero behavior differences.

---

## 5. Persona-Based Acceptance Matrix

This matrix validates that each persona's needs are met across user journeys.

| Persona | Journey | Acceptance Scenario | Success Criteria |
|---------|---------|---------------------|------------------|
| **Shark Core Developer (Human)** | Adding New Command | Scenario 1.1: Task Archive Command | Dev time ≤2h, LOC ≤200, Zero duplication |
| **Shark Core Developer (Human)** | Writing Tests | Scenario 3.1: Test Business Logic | Test code ≤15 LOC, Execution <5ms |
| **AI Code Agent (Cursor/Claude)** | Understanding Logic | Scenario 2.1: Dependency Checking | Answer time <5min, Context fits, Accuracy 100% |
| **AI Code Agent (Cursor/Claude)** | Adding New Command | Scenario 1.1: Task Archive Command | Correct file identified, Correct pattern suggested |
| **HTTP API Consumer** | API Integration | Scenario 4.1: CI/CD Integration | Integration time ≤1h, Zero workarounds |
| **HTTP API Consumer** | API Feature Parity | HTTP API Integration (P2) | 100% feature parity, Identical behavior |
| **Shark Maintainer** | PR Review | Scenario 5.1: Epic Progress PR | Review cycles ≤1, Time to merge ≤1 day |
| **Shark Maintainer** | Architecture Enforcement | Architecture Quality (AQ4) | 100% logic in services, Zero arch feedback |
| **Test Engineer** | Writing Tests | Scenario 3.1: Unit Tests | >80% service coverage, Fast execution |
| **Test Engineer** | Test Maintenance | Test Coverage (AQ2) | Tests resilient to schema changes |
| **DevOps Engineer** | CI/CD Integration | Scenario 4.1: API Integration | No CLI dependency, Pure API calls |

**Validation Method:**
- For each persona, run their acceptance scenario
- Measure success criteria (time, LOC, accuracy, etc.)
- Survey persona (if human): "Did this meet your needs?"
- Check pass/fail criteria for each metric

**Pass Criteria:**
- [ ] All 4 personas (Shark Dev, AI Agent, API Consumer, Maintainer) scenarios pass
- [ ] At least 3 of 4 secondary personas scenarios pass
- [ ] Developer survey: >80% report improved experience
- [ ] Maintainer survey: >80% report reduced review burden

---

## 6. Epic Exit Gate Criteria

These are the mandatory success conditions from `requirements.md` and `success-metrics.md`. The epic is NOT complete until all criteria pass.

### Functional Requirements Validation

| Requirement | Description | Validation Method | Status |
|-------------|-------------|-------------------|--------|
| **FR1: Service Layer Architecture** | Task/Feature/Epic services exist with clear interfaces | Check files exist: task_service.go, feature_service.go, epic_service.go; Interface documented | [ ] |
| **FR2: TaskService Implementation** | All task logic in TaskService (2,664 LOC → 400 + service) | Run metrics: command LOC <500, service exists; All task commands use TaskService | [ ] |
| **FR3-FR4: Feature/Epic Service Expansion** | Feature/Epic services expanded from transitions to full CRUD | Check methods: Create, Get, Update, Delete, List, Progress, Health in services | [ ] |
| **FR5: QueryService** | Cross-entity smart dispatch (optional P1) | Check QueryService.GetByKey(), List() exist; Commands use QueryService | [ ] |
| **FR6: CLI Commands as Thin Wrappers** | All 47 commands <500 LOC | Run: `find internal/cli/commands -name "*.go" ! -name "*_test.go" -exec wc -l {} + | awk '{if ($1 > 500) print $0}'` Result: 0 files | [ ] |
| **FR7: Repository Cleanup** | Repositories = pure data access (no business logic) | Manual review: No CalculateProgress, no validation, no status derivation in repos | [ ] |
| **FR8: HTTP API Integration** | API uses services, 100% parity with CLI (optional P1) | Check: All 47 CLI commands have API endpoints; API handlers call services | [ ] |
| **FR9: Documentation** | Service interfaces documented with godoc | Check: All public methods have godoc; .claude/rules/architecture.md updated | [ ] |

---

### Non-Functional Requirements Validation

| Requirement | Description | Validation Method | Status |
|-------------|-------------|-------------------|--------|
| **NFR1: Zero Behavior Regression** | All existing functionality works identically | Run: `make test` exits 0; Manual regression checklist; Production monitoring (30 days) | [ ] |
| **NFR2: Test Coverage >80%** | Service layer >80% coverage | Run: `go test -coverprofile=coverage.out ./internal/services/... && go tool cover -func=coverage.out | grep total` Result: >80% | [ ] |
| **NFR3: Performance ±10%** | No degradation from baseline | Benchmark: `shark task next` 45-55ms, `shark feature get` 31.5-38.5ms; All operations ±10% | [ ] |
| **NFR4: Incremental Migration** | PRs <500 LOC, main always deployable | Check: All E15 PRs <500 LOC; CI passes on main after each PR; No long-lived branches | [ ] |
| **NFR5: Backward Compatibility** | No breaking changes to CLI/API | Check: Same flags, same output schemas, same exit codes; Run existing CLI scripts | [ ] |

---

### Success Metrics Dashboard

At epic completion, this dashboard must show "excellent" threshold for at least 8/10 metrics:

| Category | Metric | Target | Status | Result |
|----------|--------|--------|--------|--------|
| **Architecture** | Code reduction in commands | 65% | [ ] | TBD |
| **Architecture** | Service layer test coverage | >80% | [ ] | TBD |
| **Architecture** | Duplication elimination | 80% | [ ] | TBD |
| **Architecture** | Business logic in services | 100% | [ ] | TBD |
| **Developer** | Time to add new command | 75% faster | [ ] | TBD |
| **Developer** | PR review cycle time | 70% faster | [ ] | TBD |
| **Developer** | Test execution speed | <100ms | [ ] | TBD |
| **System** | API feature parity | 100% | [ ] | TBD |
| **System** | Regression rate | <15 bugs | [ ] | TBD |
| **System** | Performance stability | ±10% | [ ] | TBD |

**Overall Success Criteria:** ≥8/10 metrics meet "excellent" threshold, 10/10 meet "acceptable" threshold

---

## 7. Test Execution Plan

### Phase 1: Service Layer Foundation (Weeks 2-4)

**Focus:** TaskService extraction, unit tests, zero regression

**Test Priority:**
1. TaskService CRUD operations (P0)
2. TaskService lifecycle operations (P0)
3. TaskService querying (P0)
4. Service layer test coverage >80% (P0)

**Execution:**
- Run service unit tests daily (CI)
- Run integration tests after each PR (CI)
- Manual regression testing weekly (Scenario 8)

---

### Phase 2: Epic/Feature Expansion (Week 5)

**Focus:** Expand Epic/FeatureService, repository cleanup

**Test Priority:**
1. Epic/FeatureService CRUD (P1)
2. Repository cleanup validation (P1)
3. Progress calculation parity (P1)

**Execution:**
- Run service tests daily
- Verify repository purity (manual review)
- Benchmark performance (NFR3)

---

### Phase 3: CLI Slimming (Week 7)

**Focus:** Refactor commands to thin wrappers, verify compatibility

**Test Priority:**
1. Command size validation (P1)
2. CLI interface compatibility (P0)
3. User journey scenarios (P0)

**Execution:**
- Run full test suite after each command refactor
- Manual CLI testing (all flags, all commands)
- Developer time tracking (DX1 metric)

---

### Phase 4: API Integration (Optional Week 8)

**Focus:** Wire HTTP API to services, verify feature parity

**Test Priority:**
1. API endpoint coverage (P2)
2. API-service integration (P2)
3. Response parity (P2)

**Execution:**
- API integration tests (automated)
- Curl test scripts (manual verification)
- Feature parity audit (checklist)

---

## 8. Measurement and Reporting

### During Refactoring (Weekly)

**Automated Metrics:**
```bash
# Run weekly measurements script
./scripts/measure-e15-progress.sh
```
- CLI command file sizes (trend)
- Test coverage (CI report)
- Repository instantiations (grep count)
- Performance benchmarks (go test -bench)

**Manual Metrics:**
- Regression issues (GitHub tag "E15")
- PR review cycles (GitHub API)
- Developer feedback (weekly standup)

---

### At Epic Completion

**Full Analysis:**
- Code analysis: Duplication, LOC reduction
- Developer survey: Time to add command, test writing
- API feature parity audit: 47/47 endpoints
- Documentation review: Completeness, accuracy

**Final Report:**
- Success metrics dashboard (all 10 metrics)
- User journey validation (all 5 scenarios)
- Persona acceptance matrix (all 11 entries)
- Exit gate criteria (all FR/NFR pass)

---

### Post-Epic (Monthly for 3 Months)

**Long-Term Monitoring:**
- Track new command development time (PR metadata)
- Track PR review cycles (GitHub API)
- Track production regressions (issue tracker)
- Developer onboarding time (new contributor surveys)

**Reports:**
- Month 1: Early adoption metrics
- Month 2: Sustained efficiency gains
- Month 3: Architectural stability validation

---

## 9. UAT Sign-Off Checklist

This checklist must be completed before Epic E15 is marked as "completed."

### Functional Validation

- [ ] All 5 user journey scenarios executed and pass
- [ ] All 10 success metrics measured and meet threshold
- [ ] All FR1-FR9 functional requirements validated
- [ ] All NFR1-NFR5 non-functional requirements validated
- [ ] Cross-epic integration scenarios (E12, E16) validated
- [ ] Persona acceptance matrix: all personas satisfied

### Metrics Validation

- [ ] Code reduction: ≥65% (excellent) or ≥50% (acceptable)
- [ ] Test coverage: >80% (excellent) or >70% (acceptable)
- [ ] Duplication elimination: ≥80% (excellent) or ≥60% (acceptable)
- [ ] Business logic in services: 100% (excellent) or >95% (acceptable)
- [ ] Development time: ≤2h (excellent) or ≤4h (acceptable)
- [ ] Review cycles: ≤1 (excellent) or ≤2 (acceptable)
- [ ] Test execution: <100ms (excellent) or <500ms (acceptable)
- [ ] API feature parity: 100% (excellent) or >80% (acceptable)
- [ ] Regression rate: <15 (excellent) or <30 (acceptable)
- [ ] Performance: ±10% (excellent) or ±20% (acceptable)

### Documentation Validation

- [ ] Service interfaces documented (godoc)
- [ ] Architecture documentation updated (.claude/rules/architecture.md)
- [ ] Migration guide created (CLAUDE.md)
- [ ] API documentation updated (CLI_REFERENCE.md)
- [ ] Success metrics report published

### Regression Validation

- [ ] All existing tests pass (make test exits 0)
- [ ] Manual regression checklist completed (all features work)
- [ ] No critical regressions (data loss, crashes)
- [ ] Total regressions ≤15 (excellent threshold)
- [ ] Production stable for 30 days (zero reported issues)

### Sign-Off Authority

| Role | Name | Date | Signature |
|------|------|------|-----------|
| **QA Lead** | | | |
| **Tech Lead** | | | |
| **Product Owner** | | | |
| **Epic Sponsor** | | | |

---

## 10. Appendix: Test Data and Fixtures

### Test Task Creation (for Scenario Execution)

```bash
# Create test epic
shark epic create "UAT Test Epic"

# Create test feature
shark feature create E99 "UAT Test Feature"

# Create test task (for service layer testing)
shark task create E99 F01 "UAT Test Task" --agent=backend --priority=5

# Set task to completed status (for archive scenario)
shark task start E99-F01-001
shark task complete E99-F01-001
shark task approve E99-F01-001
```

### Baseline Measurements (Capture Before E15)

```bash
# Capture baselines for comparison
echo "=== E15 Baseline Measurements ==="
echo "Command LOC:" $(find internal/cli/commands -name "*.go" ! -name "*_test.go" -exec wc -l {} + | tail -1)
echo "Repository instantiations:" $(grep -r "repository\.New" internal/cli/commands/*.go | wc -l)
echo "Task.go size:" $(wc -l internal/cli/commands/task.go)
echo "Feature.go size:" $(wc -l internal/cli/commands/feature.go)
echo "Epic.go size:" $(wc -l internal/cli/commands/epic.go)

# Performance baselines
for i in {1..10}; do /usr/bin/time -f "%e" shark task next --agent=backend 2>&1 >/dev/null; done | awk '{sum+=$1; count++} END {print "Avg task next: " sum/count "s"}'
```

### Test Result Recording Template

```markdown
# UAT Execution Results - Epic E15

**Date:** YYYY-MM-DD
**Tester:** [Name]
**Environment:** [Local/CI/Staging]

## User Journey Scenarios

### Journey 1: Adding New Command
- [ ] Scenario 1.1: Task Archive Command
  - Development time: ___h (target: ≤2h)
  - Lines of code: ___ (target: ≤200)
  - Duplication: ___% (target: 0%)
  - Result: PASS / FAIL
  - Notes: ___

### Journey 2: Understanding Logic
- [ ] Scenario 2.1: AI Dependency Check
  - Time to answer: ___min (target: <5min)
  - Context loading: SUCCESS / FAIL
  - Answer accuracy: ___% (target: 100%)
  - Result: PASS / FAIL
  - Notes: ___

[... continue for all scenarios ...]

## Success Metrics

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| AQ1: Code Reduction | 65% | ___% | PASS / FAIL |
| AQ2: Test Coverage | >80% | ___% | PASS / FAIL |
[... all 10 metrics ...]

## Sign-Off

- [ ] All exit gate criteria met
- [ ] Zero regressions confirmed
- [ ] Documentation complete
- [ ] Epic approved for completion

**Approved by:** _______________
**Date:** _______________
```

---

**UAT Plan Complete**
**Status:** Ready for Feature-Level Test Strategy Decomposition
**Next Phase:** QA to create feature-level UAT plans for E15-F01 through E15-F07

