# Task Breakdown: Feature Get Workflow Enhancement

**Epic:** E07 - Configurable Workflow System
**Feature:** F16 - Workflow Config Integration
**Phase:** Enhancement

---

## Task Hierarchy

```
T-E07-F16-003: Create WorkflowService infrastructure
├─ T-E07-F16-003-001: Implement WorkflowService core
├─ T-E07-F16-003-002: Implement status formatter
└─ T-E07-F16-003-003: Write comprehensive unit tests

T-E07-F16-004: Enhance TaskRepository for workflow support
├─ T-E07-F16-004-001: Update GetStatusBreakdown signature
├─ T-E07-F16-004-002: Implement status ordering logic
└─ T-E07-F16-004-003: Update repository tests

T-E07-F16-005: Update feature get command
├─ T-E07-F16-005-001: Inject WorkflowService
├─ T-E07-F16-005-002: Update renderFeatureDetails
└─ T-E07-F16-005-003: Add phase grouping (optional)

T-E07-F16-006: Refactor task creation
└─ T-E07-F16-006-001: Replace getInitialTaskStatus with WorkflowService

T-E07-F16-007: Integration testing and documentation
├─ T-E07-F16-007-001: Write integration tests
├─ T-E07-F16-007-002: Update CLI_REFERENCE.md
└─ T-E07-F16-007-003: Add workflow config examples
```

---

## Task Details

### T-E07-F16-003-001: Implement WorkflowService core

**Priority:** 8
**Agent:** backend
**Dependencies:** None
**Estimated Effort:** 3-4 hours

**Description:**
Create the WorkflowService to centralize workflow config access across all CLI commands.

**Acceptance Criteria:**
- [ ] Create `internal/workflow/service.go`
- [ ] Implement `NewService(projectRoot string) *Service`
- [ ] Implement `GetWorkflow() *WorkflowConfig`
- [ ] Implement `GetInitialStatus() TaskStatus`
- [ ] Implement `GetAllStatuses() []string` with phase ordering
- [ ] Implement `GetStatusMetadata(status string) StatusMetadata`
- [ ] Implement `GetStatusesByPhase(phase string) []string`
- [ ] Service loads and caches workflow config on construction
- [ ] Graceful fallback to default workflow when config missing

**Files:**
- Create: `internal/workflow/service.go`
- Create: `internal/workflow/types.go`

**Test Plan:**
- Unit tests for each public method
- Test with valid workflow config
- Test with missing workflow config
- Test with invalid workflow config
- Test caching behavior

---

### T-E07-F16-003-002: Implement status formatter

**Priority:** 7
**Agent:** backend
**Dependencies:** T-E07-F16-003-001
**Estimated Effort:** 2-3 hours

**Description:**
Create status formatting utilities with ANSI color support and --no-color handling.

**Acceptance Criteria:**
- [ ] Create `internal/workflow/formatter.go`
- [ ] Implement `FormattedStatus` struct
- [ ] Implement `FormatStatusForDisplay(status, colorEnabled) FormattedStatus`
- [ ] Implement `colorizeStatus(status, colorName) string`
- [ ] Support standard terminal colors (red, green, yellow, blue, etc.)
- [ ] Support custom hex colors (optional)
- [ ] Respect --no-color flag
- [ ] Handle missing color metadata gracefully

**Files:**
- Create: `internal/workflow/formatter.go`

**Test Plan:**
- Test color codes applied correctly
- Test --no-color flag disables colors
- Test missing color metadata returns uncolored status
- Test all standard colors
- Visual test in terminal

---

### T-E07-F16-003-003: Write comprehensive unit tests

**Priority:** 8
**Agent:** backend
**Dependencies:** T-E07-F16-003-001, T-E07-F16-003-002
**Estimated Effort:** 2-3 hours

**Description:**
Write comprehensive unit tests for WorkflowService and formatter with >80% coverage.

**Acceptance Criteria:**
- [ ] Create `internal/workflow/service_test.go`
- [ ] Create `internal/workflow/formatter_test.go`
- [ ] Test all WorkflowService methods
- [ ] Test status ordering algorithm
- [ ] Test formatter with different configs
- [ ] Test error handling and fallbacks
- [ ] Code coverage > 80%
- [ ] All tests pass

**Files:**
- Create: `internal/workflow/service_test.go`
- Create: `internal/workflow/formatter_test.go`

**Test Plan:**
- Run `go test -v ./internal/workflow`
- Run `go test -cover ./internal/workflow`
- Verify coverage report shows >80%

---

### T-E07-F16-004-001: Update GetStatusBreakdown signature

**Priority:** 8
**Agent:** backend
**Dependencies:** T-E07-F16-003-001
**Estimated Effort:** 2-3 hours

**Description:**
Update TaskRepository.GetStatusBreakdown to return ordered slice with metadata instead of unordered map.

**Acceptance Criteria:**
- [ ] Create `StatusCount` struct in `internal/repository/types.go`
- [ ] Update `GetStatusBreakdown` signature from `map[TaskStatus]int` to `[]StatusCount`
- [ ] Query actual task counts from database (unchanged)
- [ ] Get all statuses from workflow config
- [ ] Return ALL statuses with counts (including zero counts)
- [ ] Order results by workflow phase
- [ ] Include phase and description metadata
- [ ] Update all callers (feature get, epic get)

**Files:**
- Modify: `internal/repository/task_repository.go`
- Create: `internal/repository/types.go` (for StatusCount)

**Test Plan:**
- Unit test with workflow config
- Unit test with default workflow
- Verify zero counts included
- Verify ordering by phase
- Integration test with feature get

---

### T-E07-F16-004-002: Implement status ordering logic

**Priority:** 7
**Agent:** backend
**Dependencies:** T-E07-F16-004-001
**Estimated Effort:** 1-2 hours

**Description:**
Implement status ordering algorithm to sort statuses by workflow phase and alphabetically within phase.

**Acceptance Criteria:**
- [ ] Implement `getOrderedStatuses() []string` in TaskRepository
- [ ] Group statuses by phase (planning, development, review, qa, approval, done)
- [ ] Sort alphabetically within each phase
- [ ] Concatenate phases in order
- [ ] Handle statuses without phase metadata
- [ ] Return consistent ordering across calls

**Files:**
- Modify: `internal/repository/task_repository.go`

**Test Plan:**
- Unit test with various workflow configs
- Verify phase ordering (planning → development → review → qa → done)
- Verify alphabetical sorting within phases
- Test with missing phase metadata

---

### T-E07-F16-004-003: Update repository tests

**Priority:** 7
**Agent:** backend
**Dependencies:** T-E07-F16-004-001, T-E07-F16-004-002
**Estimated Effort:** 1-2 hours

**Description:**
Update existing repository tests to work with new StatusCount return type.

**Acceptance Criteria:**
- [ ] Update `task_repository_test.go` for GetStatusBreakdown changes
- [ ] Update test assertions from map to slice
- [ ] Add tests for zero-count statuses
- [ ] Add tests for status ordering
- [ ] Add tests for metadata inclusion
- [ ] All existing tests pass
- [ ] No regression in test coverage

**Files:**
- Modify: `internal/repository/task_repository_test.go`

**Test Plan:**
- Run `go test -v ./internal/repository`
- Verify all tests pass
- Check coverage hasn't decreased

---

### T-E07-F16-005-001: Inject WorkflowService

**Priority:** 8
**Agent:** backend
**Dependencies:** T-E07-F16-003-001
**Estimated Effort:** 1 hour

**Description:**
Update feature get command to create and inject WorkflowService.

**Acceptance Criteria:**
- [ ] Get project root in `runFeatureGet()`
- [ ] Create WorkflowService instance
- [ ] Pass workflow config to TaskRepository constructor
- [ ] Pass WorkflowService to renderFeatureDetails
- [ ] No breaking changes to existing behavior

**Files:**
- Modify: `internal/cli/commands/feature.go`

**Test Plan:**
- Manual test: `shark feature get E04-F06`
- Manual test with workflow config
- Manual test without workflow config
- Verify no regression

---

### T-E07-F16-005-002: Update renderFeatureDetails

**Priority:** 8
**Agent:** backend
**Dependencies:** T-E07-F16-004-001, T-E07-F16-005-001
**Estimated Effort:** 2-3 hours

**Description:**
Update feature details rendering to display workflow-aware status breakdown.

**Acceptance Criteria:**
- [ ] Update `renderFeatureDetails` signature to accept `[]StatusCount`
- [ ] Update signature to accept `*workflow.Service`
- [ ] Display status breakdown table with columns: Status, Count, Phase, Description
- [ ] Apply color coding if `--no-color` not set
- [ ] Truncate long descriptions (>50 chars)
- [ ] Display all statuses (including zero counts)
- [ ] Order matches workflow config
- [ ] Update JSON output to include metadata

**Files:**
- Modify: `internal/cli/commands/feature.go`

**Test Plan:**
- Manual test with color output
- Manual test with --no-color
- Manual test with --json
- Verify status ordering
- Verify descriptions shown
- Screenshot for visual review

---

### T-E07-F16-005-003: Add phase grouping (optional)

**Priority:** 5
**Agent:** backend
**Dependencies:** T-E07-F16-005-002
**Estimated Effort:** 1-2 hours

**Description:**
Add optional phase grouping to display tasks grouped by workflow phase.

**Acceptance Criteria:**
- [ ] Create `renderTasksByPhase()` function
- [ ] Group tasks by phase using workflow metadata
- [ ] Display phase headers (Planning, Development, Review, etc.)
- [ ] Render task table for each phase
- [ ] Only show phases with tasks
- [ ] Maintain backward compatibility (make optional)

**Files:**
- Modify: `internal/cli/commands/feature.go`

**Test Plan:**
- Manual test: verify phase grouping
- Verify all phases shown
- Verify empty phases hidden
- Visual review

---

### T-E07-F16-006-001: Replace getInitialTaskStatus with WorkflowService

**Priority:** 7
**Agent:** backend
**Dependencies:** T-E07-F16-003-001
**Estimated Effort:** 1-2 hours

**Description:**
Refactor task creation to use shared WorkflowService instead of duplicated logic.

**Acceptance Criteria:**
- [ ] Add `workflowService *workflow.Service` field to Creator struct
- [ ] Add WorkflowService parameter to NewCreator constructor
- [ ] Replace `getInitialTaskStatus()` call with `workflowService.GetInitialStatus()`
- [ ] Remove `getInitialTaskStatus()` method
- [ ] Update all Creator construction sites
- [ ] No change in behavior (tests pass)
- [ ] Code duplication eliminated

**Files:**
- Modify: `internal/taskcreation/creator.go`
- Modify: `internal/cli/commands/task.go` (Creator construction)

**Test Plan:**
- Unit tests for Creator (unchanged behavior)
- Integration test: task creation uses workflow config
- Verify initial status matches workflow config
- All existing tests pass

---

### T-E07-F16-007-001: Write integration tests

**Priority:** 8
**Agent:** backend
**Dependencies:** All implementation tasks
**Estimated Effort:** 2-3 hours

**Description:**
Write comprehensive integration tests for feature get with workflow config.

**Acceptance Criteria:**
- [ ] Create `internal/cli/commands/feature_workflow_test.go`
- [ ] Test feature get with custom workflow config
- [ ] Test feature get with default workflow
- [ ] Test feature get with missing config
- [ ] Test status breakdown ordering
- [ ] Test color output (if possible)
- [ ] Test JSON output includes metadata
- [ ] All tests pass

**Files:**
- Create: `internal/cli/commands/feature_workflow_test.go`

**Test Plan:**
- Run `go test -v ./internal/cli/commands`
- Verify all integration tests pass
- Check coverage for feature.go

---

### T-E07-F16-007-002: Update CLI_REFERENCE.md

**Priority:** 6
**Agent:** general
**Dependencies:** All implementation tasks
**Estimated Effort:** 1 hour

**Description:**
Update CLI reference documentation to reflect new workflow-aware feature get behavior.

**Acceptance Criteria:**
- [ ] Update `docs/CLI_REFERENCE.md`
- [ ] Document new status breakdown format
- [ ] Add examples with workflow config
- [ ] Document --no-color flag interaction
- [ ] Add screenshots (optional)
- [ ] Update JSON output examples

**Files:**
- Modify: `docs/CLI_REFERENCE.md`

**Test Plan:**
- Review documentation for accuracy
- Verify examples work as documented

---

### T-E07-F16-007-003: Add workflow config examples

**Priority:** 5
**Agent:** general
**Dependencies:** All implementation tasks
**Estimated Effort:** 1 hour

**Description:**
Create example workflow configurations for common use cases.

**Acceptance Criteria:**
- [ ] Create `docs/examples/workflows/` directory
- [ ] Add simple workflow example (3 statuses)
- [ ] Add complex workflow example (10+ statuses)
- [ ] Add agile workflow example (sprint-based)
- [ ] Add waterfall workflow example
- [ ] Document each example's use case

**Files:**
- Create: `docs/examples/workflows/simple.json`
- Create: `docs/examples/workflows/complex.json`
- Create: `docs/examples/workflows/agile.json`
- Create: `docs/examples/workflows/waterfall.json`
- Create: `docs/examples/workflows/README.md`

**Test Plan:**
- Validate JSON syntax
- Test each example with shark
- Verify visual output

---

## Task Creation Commands

```bash
# Create main tasks
shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  "Create WorkflowService infrastructure"

shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  "Enhance TaskRepository for workflow support"

shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  "Update feature get command"

shark task create --epic=E07 --feature=F16 --priority=7 --agent=backend \
  "Refactor task creation to use WorkflowService"

shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  "Integration testing and documentation"

# Create subtasks (after parent tasks exist)
# T-E07-F16-003: WorkflowService
shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  --depends-on=T-E07-F16-003 \
  "Implement WorkflowService core"

shark task create --epic=E07 --feature=F16 --priority=7 --agent=backend \
  --depends-on=T-E07-F16-003-001 \
  "Implement status formatter"

shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  --depends-on=T-E07-F16-003-001,T-E07-F16-003-002 \
  "Write comprehensive unit tests"

# T-E07-F16-004: TaskRepository
shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  --depends-on=T-E07-F16-003-001 \
  "Update GetStatusBreakdown signature"

shark task create --epic=E07 --feature=F16 --priority=7 --agent=backend \
  --depends-on=T-E07-F16-004-001 \
  "Implement status ordering logic"

shark task create --epic=E07 --feature=F16 --priority=7 --agent=backend \
  --depends-on=T-E07-F16-004-001,T-E07-F16-004-002 \
  "Update repository tests"

# T-E07-F16-005: Feature get command
shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  --depends-on=T-E07-F16-003-001 \
  "Inject WorkflowService into feature get"

shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  --depends-on=T-E07-F16-004-001,T-E07-F16-005-001 \
  "Update renderFeatureDetails"

shark task create --epic=E07 --feature=F16 --priority=5 --agent=backend \
  --depends-on=T-E07-F16-005-002 \
  "Add phase grouping (optional)"

# T-E07-F16-006: Task creation refactor
shark task create --epic=E07 --feature=F16 --priority=7 --agent=backend \
  --depends-on=T-E07-F16-003-001 \
  "Replace getInitialTaskStatus with WorkflowService"

# T-E07-F16-007: Testing and docs
shark task create --epic=E07 --feature=F16 --priority=8 --agent=backend \
  --depends-on=T-E07-F16-003-003,T-E07-F16-004-003,T-E07-F16-005-002 \
  "Write integration tests"

shark task create --epic=E07 --feature=F16 --priority=6 --agent=general \
  "Update CLI_REFERENCE.md"

shark task create --epic=E07 --feature=F16 --priority=5 --agent=general \
  "Add workflow config examples"
```

---

## Dependency Graph

```
T-E07-F16-003-001 (WorkflowService core)
    ├─→ T-E07-F16-003-002 (Formatter)
    │       └─→ T-E07-F16-003-003 (Unit tests)
    ├─→ T-E07-F16-004-001 (GetStatusBreakdown)
    │       ├─→ T-E07-F16-004-002 (Ordering logic)
    │       │       └─→ T-E07-F16-004-003 (Repository tests)
    │       └─→ T-E07-F16-005-002 (renderFeatureDetails)
    │               └─→ T-E07-F16-005-003 (Phase grouping)
    ├─→ T-E07-F16-005-001 (Inject WorkflowService)
    │       └─→ T-E07-F16-005-002 (renderFeatureDetails)
    └─→ T-E07-F16-006-001 (Refactor task creation)

T-E07-F16-003-003 + T-E07-F16-004-003 + T-E07-F16-005-002
    └─→ T-E07-F16-007-001 (Integration tests)

All implementation tasks
    ├─→ T-E07-F16-007-002 (Update docs)
    └─→ T-E07-F16-007-003 (Workflow examples)
```

---

## Execution Order (Recommended)

1. **Phase 1: Core Infrastructure**
   - T-E07-F16-003-001 (WorkflowService core)
   - T-E07-F16-003-002 (Formatter)
   - T-E07-F16-003-003 (Unit tests)

2. **Phase 2: Repository Enhancement**
   - T-E07-F16-004-001 (GetStatusBreakdown)
   - T-E07-F16-004-002 (Ordering logic)
   - T-E07-F16-004-003 (Repository tests)

3. **Phase 3: Feature Get Command**
   - T-E07-F16-005-001 (Inject WorkflowService)
   - T-E07-F16-005-002 (renderFeatureDetails)
   - T-E07-F16-005-003 (Phase grouping) [Optional]

4. **Phase 4: Refactor Task Creation**
   - T-E07-F16-006-001 (Replace getInitialTaskStatus)

5. **Phase 5: Testing & Documentation**
   - T-E07-F16-007-001 (Integration tests)
   - T-E07-F16-007-002 (Update CLI docs)
   - T-E07-F16-007-003 (Workflow examples)

---

**Total Tasks:** 13 main tasks
**Total Estimated Effort:** 16-23 hours
**Critical Path:** 1 → 2 → 3 → 5 (Core → Repository → Feature Get → Testing)
