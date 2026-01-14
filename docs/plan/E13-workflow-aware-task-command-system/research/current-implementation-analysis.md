# Current Implementation Analysis: E13 Workflow-Aware Task Command System

**Research Date**: 2026-01-11
**Epic**: E13 - Workflow-Aware Task Command System
**Purpose**: Document current implementation to guide refactoring for phase-aware commands

---

## Executive Summary

The shark task manager currently has **hardcoded workflow assumptions** scattered across 8 primary command files and the repository layer. The transition to workflow-aware commands requires:

1. **7 deprecated commands** to be replaced with 3 new phase-aware commands
2. **4 command consolidations** for better UX
3. **Workflow configuration integration** in 12+ command functions
4. **Repository layer** already supports workflow validation (good foundation)
5. **Estimated scope**: ~2,500 lines of code affected across 15 files

---

## Current Command Architecture

### 1. Core Task Commands (task.go)

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go` (2,230 lines)

#### Commands Affected by E13

| Command | Lines | Status | Replacement |
|---------|-------|--------|-------------|
| `task start` | 1184-1273 | DEPRECATE | `task claim` |
| `task complete` | 1275-1427 | DEPRECATE | `task finish` |
| `task approve` | 1429-1510 | DEPRECATE | `task finish` |
| `task reopen` | 1676-1763 | DEPRECATE | `task reject` |
| `task block` | 1523-1613 | KEEP | (special case) |
| `task unblock` | 1615-1673 | KEEP | (special case) |
| `task next` | 677-853 | ENHANCE | (workflow-aware query) |

#### Hardcoded Status Assumptions

**`runTaskStart` (lines 1184-1273)**:
```go
// Line 1249: Hardcoded transition to in_progress
err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusInProgress, &agent, nil, force)

// Workflow config is loaded but only for validation, not to determine target status
workflow, err := config.LoadWorkflowConfig(configPath)
```

**`runTaskComplete` (lines 1275-1427)**:
```go
// Line 1337: Hardcoded transition to ready_for_review
err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusReadyForReview, &agent, notes, force)
```

**`runTaskApprove` (lines 1429-1510)**:
```go
// Line 1491: Hardcoded transition to completed
err := repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, &agent, notes, force)
```

**`runTaskReopen` (lines 1676-1763)**:
```go
// Lines 1710-1737: Hardcoded list of "reopen target" statuses
reopenTargets := []string{"in_development", "in_progress", "ready_for_development", "ready_for_refinement", "in_refinement"}
```

**`runTaskNext` (lines 677-853)**:
```go
// Line 700: Hardcoded filter for "todo" status only
todoStatus := models.TaskStatusTodo
tasks, err := repo.FilterCombined(ctx, &todoStatus, epicKeyPtr, agentType, nil)
```

#### What Works Well

1. **Workflow config loading** is already implemented (lines 1206-1214, 1298-1305, etc.)
2. **Force flag** for admin overrides works consistently
3. **Work session tracking** exists (lines 1252-1261, 1346-1353)
4. **Cascade triggers** are centralized (function `triggerStatusCascade` at line 35)

---

### 2. next-status Command (task_next_status.go)

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task_next_status.go` (304 lines)

**Status**: DEPRECATE (overlaps with `complete`, `approve`)

#### Current Implementation

- **Purpose**: Progress task through workflow interactively
- **Workflow integration**: GOOD - reads workflow config (line 98)
- **Replacement**: `task finish` will subsume this functionality
- **Reusable logic**:
  - Interactive transition selection (lines 229-234)
  - Workflow-driven suggestions (lines 71-237)
  - Transition choice formatting (lines 50-58, 240-256)

```go
// Line 113: Uses workflow service correctly
workflowSvc := workflow.NewService(projectRoot)
transitions := workflowSvc.GetTransitionInfo(currentStatus)
```

**Migration Notes**:
- This command ALREADY does what `finish` should do
- Can reuse `TransitionChoice` struct and `printTransitions` formatting
- Interactive mode (`--preview` flag) is good UX to preserve

---

### 3. Dependency Commands (task_deps.go)

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task_deps.go` (779 lines)

**Status**: CONSOLIDATE

| Command | Lines | Status | New Signature |
|---------|-------|--------|---------------|
| `task deps` | 101-276 | ENHANCE | Add `--type` flag |
| `task blocked-by` | 278-350 | DEPRECATE | `task deps --type=blocked-by` |
| `task blocks` | 352-450 | DEPRECATE | `task deps --type=blocks` |

#### Consolidation Plan

1. **Keep**: `runTaskDeps` (already supports `--type` flag at line 104)
2. **Deprecate**: `runTaskBlockedBy` and `runTaskBlocks`
3. **Add warnings**: "Use `shark task deps --type=blocked-by` instead"

**Current `--type` flag** (line 81):
```go
taskDepsCmd.Flags().String("type", "", "Filter by relationship types (comma-separated)")
```

Already supports filtering! Just need deprecation warnings in the old commands.

---

## Workflow System Architecture

### 1. Workflow Configuration

**Schema**: `/home/jwwelbor/projects/shark-task-manager/internal/config/workflow_schema.go`

```go
type WorkflowConfig struct {
    Version         string                     `json:"status_flow_version"`
    StatusFlow      map[string][]string        `json:"status_flow"`
    StatusMetadata  map[string]StatusMetadata  `json:"status_metadata"`
    SpecialStatuses map[string][]string        `json:"special_statuses"`
}

type StatusMetadata struct {
    Color       string   `json:"color,omitempty"`
    Description string   `json:"description,omitempty"`
    Phase       string   `json:"phase,omitempty"`
    AgentTypes  []string `json:"agent_types,omitempty"`
}
```

**Key Methods**:
- `GetStatusesByPhase(phase string) []string`
- `GetStatusesByAgentType(agentType string) []string`
- `GetStatusMetadata(status string) (StatusMetadata, bool)`

### 2. Workflow Service

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/workflow/service.go` (363 lines)

**Excellent foundation** for phase-aware commands:

```go
// Line 193: Get valid transitions from current status
func (s *Service) GetValidTransitions(currentStatus string) []string

// Line 203: Get detailed transition info with metadata
func (s *Service) GetTransitionInfo(currentStatus string) []TransitionInfo

// Line 228: Validate transition
func (s *Service) IsValidTransition(currentStatus, targetStatus string) bool

// Line 182: Get statuses by phase
func (s *Service) GetStatusesByPhase(phase string) []string

// Line 187: Get statuses by agent type
func (s *Service) GetStatusesByAgentType(agentType string) []string
```

**Critical for E13**:
- Already supports phase queries
- Already supports agent type filtering
- Already validates transitions
- Just need CLI commands to use these methods!

### 3. Repository Layer

**File**: `/home/jwwelbor/projects/shark-task-manager/internal/repository/task_repository.go`

**Status Transition Functions**:

```go
// Line 796: Standard status update with workflow validation
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error

// Line 801: Force update bypassing validation
func (r *TaskRepository) UpdateStatusForced(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string, force bool) error

// Line 792: Validate transition against workflow
return config.ValidateTransition(r.workflow, string(from), string(to)) == nil
```

**Good news**: Repository already supports workflow validation!

**Challenge**: Commands currently call `UpdateStatusForced` with hardcoded target statuses. Need to:
1. Query workflow for valid next status
2. Use pattern matching (`ready_for_*` → `in_*`)
3. Pass determined status to repository

---

## Hardcoded Workflow Assumptions

### Pattern 1: Hardcoded Target Statuses

**Where**: `task.go` - `runTaskStart`, `runTaskComplete`, `runTaskApprove`

**Current**:
```go
// start → in_progress (hardcoded)
repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusInProgress, ...)

// complete → ready_for_review (hardcoded)
repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusReadyForReview, ...)

// approve → completed (hardcoded)
repo.UpdateStatusForced(ctx, task.ID, models.TaskStatusCompleted, ...)
```

**Should be**:
```go
// claim: ready_for_X → in_X (workflow-driven)
targetStatus := determineInStatus(task.Status, workflow)
repo.UpdateStatusForced(ctx, task.ID, targetStatus, ...)

// finish: in_X → ready_for_Y (workflow-driven)
targetStatus := determineNextReadyStatus(task.Status, workflow)
repo.UpdateStatusForced(ctx, task.ID, targetStatus, ...)
```

### Pattern 2: Hardcoded Status Filters

**Where**: `task.go` - `runTaskNext` (line 700)

**Current**:
```go
// Only queries tasks in "todo" status
todoStatus := models.TaskStatusTodo
tasks, err := repo.FilterCombined(ctx, &todoStatus, epicKeyPtr, agentType, nil)
```

**Should be**:
```go
// Query tasks in any "ready_for_X" status matching agent type
readyStatuses := workflow.GetStatusesByAgentType(agentType)
// Filter for "ready_for_" pattern
tasks := filterTasksByReadyStatuses(ctx, repo, readyStatuses, agentType)
```

### Pattern 3: Hardcoded Reopen Targets

**Where**: `task.go` - `runTaskReopen` (lines 1717-1718)

**Current**:
```go
reopenTargets := []string{"in_development", "in_progress", "ready_for_development", "ready_for_refinement", "in_refinement"}
```

**Should be**:
```go
// Use workflow to determine valid backward transitions
backwardStatuses := workflow.GetValidTransitions(task.Status)
// Or use phase-based logic: go to previous phase
```

---

## Code Reuse Opportunities

### 1. Workflow Config Loading Pattern

**Appears in**: `runTaskStart`, `runTaskComplete`, `runTaskApprove`, `runTaskReopen`, `runTaskSetStatus`

**Duplicated code** (lines 1206-1214, 1298-1305, 1451-1459, etc.):
```go
configPath := cli.GlobalConfig.ConfigFile
if configPath == "" {
    configPath = ".sharkconfig.json"
}
workflow, err := config.LoadWorkflowConfig(configPath)
if err != nil {
    return fmt.Errorf("failed to load workflow config: %w", err)
}
```

**Refactor to**:
```go
// Helper function in task.go
func loadWorkflowForCommand() (*config.WorkflowConfig, error) {
    configPath := cli.GlobalConfig.ConfigFile
    if configPath == "" {
        configPath = ".sharkconfig.json"
    }
    workflow, err := config.LoadWorkflowConfig(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load workflow config: %w", err)
    }
    return workflow, nil
}
```

**Impact**: Eliminate ~50 lines of duplicate code across 5 commands.

### 2. Work Session Management

**Appears in**: `runTaskStart` (create), `runTaskComplete` (end), `runTaskBlock` (end)

**Pattern** (lines 1252-1261, 1346-1353, 1595-1601):
```go
sessionRepo := repository.NewWorkSessionRepository(dbWrapper)
session := &models.WorkSession{
    TaskID:    task.ID,
    AgentID:   &agent,
    StartedAt: time.Now(),
}
err := sessionRepo.Create(ctx, session)
```

**Can abstract to**:
```go
func createWorkSession(ctx context.Context, db *repository.DB, taskID int64, agent string) error
func endWorkSession(ctx context.Context, db *repository.DB, taskID int64, outcome models.SessionOutcome, notes *string) error
```

### 3. Interactive Transition Selection

**Already exists in**: `task_next_status.go` (lines 229-274)

**Reuse for**: `task finish` interactive mode

```go
// From task_next_status.go - can move to shared utility
func promptForSelection(max int) (int, error) { ... }
func printTransitions(transitions []TransitionChoice) { ... }
```

---

## Dependency Analysis

### Commands That Import Each Other

**None** - good isolation!

### Shared Dependencies

All task commands depend on:
1. `internal/repository` - database layer
2. `internal/config` - workflow config
3. `internal/models` - data types
4. `internal/cli` - global config and output helpers
5. `internal/workflow` - workflow service (some commands)

**Observation**: `workflow.Service` is underutilized. Only `task_next_status.go` uses it properly.

---

## Database Query Analysis

### 1. Task Filtering

**Current**: `repo.FilterCombined(ctx, status, epicKey, agentType, maxPriority)`

**File**: `internal/repository/task_repository.go`

**Supports**:
- Status filtering (single status)
- Epic filtering
- Agent type filtering
- Priority filtering

**Missing**:
- Phase filtering
- Multiple status filtering (e.g., all `ready_for_*`)

**Enhancement needed**:
```go
// Add to TaskRepository
func (r *TaskRepository) FilterByPhase(ctx context.Context, phase string) ([]*models.Task, error) {
    statuses := r.workflow.GetStatusesByPhase(phase)
    // Query tasks WHERE status IN (statuses)
}

func (r *TaskRepository) FilterByStatusPattern(ctx context.Context, pattern string) ([]*models.Task, error) {
    // Query tasks WHERE status LIKE pattern
}
```

### 2. Status Transitions

**Current**: `repo.UpdateStatusForced(ctx, taskID, newStatus, agent, notes, force)`

**Works well** - no changes needed in repository layer!

**Workflow validation** already happens in repository (line 792):
```go
return config.ValidateTransition(r.workflow, string(from), string(to)) == nil
```

---

## Validation Logic

### Current Approach

**Location**: Repository layer (`task_repository.go`)

**Flow**:
1. Command passes target status to `UpdateStatusForced`
2. Repository validates transition against workflow config
3. Returns error if invalid (unless `force=true`)

**Problem**: Commands hardcode the target status instead of consulting workflow.

### New Approach for Phase-Aware Commands

**Flow**:
1. Command reads current task status
2. Command consults workflow service for valid next statuses
3. Command selects appropriate target (pattern matching or interactive)
4. Command passes target to repository
5. Repository validates (redundant but safe)

**Example**:
```go
// In runTaskClaim:
workflow := loadWorkflowForCommand()
workflowSvc := workflow.NewService(projectRoot)

// Determine target status based on phase pattern
currentStatus := string(task.Status)
if !strings.HasPrefix(currentStatus, "ready_for_") {
    return fmt.Errorf("can only claim tasks in ready_for_X status, current: %s", currentStatus)
}

// Transform ready_for_X → in_X
targetPhase := strings.TrimPrefix(currentStatus, "ready_for_")
targetStatus := "in_" + targetPhase

// Validate it's a valid transition
if !workflowSvc.IsValidTransition(currentStatus, targetStatus) {
    return fmt.Errorf("workflow does not allow transition from %s to %s", currentStatus, targetStatus)
}

// Execute transition
repo.UpdateStatus(ctx, task.ID, models.TaskStatus(targetStatus), &agent, nil)
```

---

## Test Coverage Analysis

### Test Files Affected

**Pattern**: `internal/cli/commands/*_test.go`

| Test File | Purpose | Changes Needed |
|-----------|---------|----------------|
| `task_workflow_test.go` | Workflow validation | Update for new commands |
| `task_update_status_test.go` | Status transitions | Add phase-aware tests |
| `task_next_test.go` | Next task selection | Add phase filtering tests |
| `task_work_session_test.go` | Work sessions | Update for claim/finish |
| `task_deps_tree_test.go` | Dependency tree | Minimal (consolidation) |

### Current Test Approach

**Repository tests**: Use real database (good)
**CLI tests**: SHOULD use mocks (per CLAUDE.md), but some use real DB

**Example from testing architecture**:
```go
// Repository test (✅ correct)
func TestTaskRepository_UpdateStatus(t *testing.T) {
    database := test.GetTestDB()
    // Clean up before test
    database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")
    // Test status transition
}

// CLI test (❌ wrong - should use mock)
func TestTaskStartCommand(t *testing.T) {
    // Should create MockTaskRepository, not use real DB
}
```

**Migration requirement**: Add mocks for new commands (`MockWorkflowService`, update `MockTaskRepository`).

---

## Migration Complexity Estimate

### Phase 1: Add New Commands (Non-Breaking)

**Effort**: 3-5 days

**Tasks**:
1. Implement `shark task claim` (reuse from `start` + workflow logic)
2. Implement `shark task finish` (reuse from `next-status` + session end)
3. Implement `shark task reject` (new logic + workflow backward transition)
4. Add deprecation warnings to old commands
5. Write tests for new commands

**Files to create**:
- `internal/cli/commands/task_claim.go` (~150 lines)
- `internal/cli/commands/task_finish.go` (~200 lines, reuse from task_next_status)
- `internal/cli/commands/task_reject.go` (~180 lines)

**Files to modify**:
- `internal/cli/commands/task.go` (add command registrations)
- `internal/cli/commands/task_start.go` (extract to separate file, add warning)
- `internal/cli/commands/task_complete.go` (extract to separate file, add warning)
- `internal/cli/commands/task_approve.go` (extract to separate file, add warning)
- `internal/cli/commands/task_reopen.go` (extract to separate file, add warning)

### Phase 2: Enhance Workflow-Aware Next

**Effort**: 2-3 days

**Tasks**:
1. Update `runTaskNext` to filter by phase
2. Add `--phase` flag
3. Pattern matching for `ready_for_*` statuses
4. Integration with `workflow.Service.GetStatusesByPhase`

**Files to modify**:
- `internal/cli/commands/task.go` (`runTaskNext` function)
- `internal/repository/task_repository.go` (add `FilterByPhase` method)

### Phase 3: Command Consolidation

**Effort**: 1-2 days

**Tasks**:
1. Add deprecation warnings to `blocked-by` and `blocks`
2. Verify `deps --type` flag works
3. Update help text
4. Update documentation

**Files to modify**:
- `internal/cli/commands/task_deps.go` (add warnings)

### Phase 4: Documentation & Migration Guide

**Effort**: 1-2 days

**Tasks**:
1. Write `OLD_TO_NEW_COMMANDS.md` migration guide
2. Update `CLI_REFERENCE.md`
3. Update `CLAUDE.md` with new examples
4. Update `README.md`

---

## Command Mapping Table

### Commands to Replace

| Old Command | New Command | Status | Complexity | Reusable Code |
|-------------|-------------|--------|------------|---------------|
| `task start` | `task claim` | DEPRECATE | Medium | Workflow loading, session start |
| `task complete` | `task finish` | DEPRECATE | Medium | Session end, metadata |
| `task approve` | `task finish` | DEPRECATE | Low | Same as complete |
| `task next-status` | `task finish` | DEPRECATE | Low | 90% reusable |
| `task reopen` | `task reject` | DEPRECATE | Medium | Workflow validation |
| `task blocks` | `task deps --type=blocks` | DEPRECATE | Low | Already implemented |
| `task blocked-by` | `task deps --type=blocked-by` | DEPRECATE | Low | Already implemented |

### Commands to Keep

| Command | Changes | Complexity |
|---------|---------|------------|
| `task block` | None (special case) | None |
| `task unblock` | None (special case) | None |
| `task next` | Enhance with phase filtering | Medium |
| `task list` | Add `--phase` flag | Low |
| `task deps` | Already supports `--type` | None |
| `task get` | None | None |
| `task create` | None | None |
| `task delete` | None | None |
| `task update` | None | None |

---

## Risk Assessment

### High Risk

**1. Breaking existing automation**:
- Many scripts may use `shark task start`
- Mitigation: Deprecation warnings, not immediate removal

**2. Workflow config edge cases**:
- What if workflow has no `ready_for_*` or `in_*` statuses?
- Mitigation: Fallback to default workflow, clear error messages

### Medium Risk

**3. Test coverage gaps**:
- Need comprehensive tests for all custom workflows
- Mitigation: Create test suite with 5+ workflow configurations

**4. Repository layer changes**:
- May need new query methods
- Mitigation: Add methods, don't modify existing signatures

### Low Risk

**5. Documentation maintenance**:
- Multiple docs need updates
- Mitigation: Checklist of all docs to update

---

## Recommended Implementation Order

### Week 1: Foundation
1. Extract shared helpers (workflow loading, session management)
2. Add repository methods for phase filtering
3. Write comprehensive workflow tests

### Week 2: New Commands
1. Implement `task claim` with tests
2. Implement `task finish` with tests
3. Implement `task reject` with tests

### Week 3: Enhancement & Consolidation
1. Enhance `task next` for phase filtering
2. Add deprecation warnings to old commands
3. Test all workflows (default, custom, edge cases)

### Week 4: Polish & Documentation
1. Update all documentation
2. Write migration guide
3. Create example workflow configs
4. Integration testing

---

## Key Files Reference

### Commands Layer
- **Main task commands**: `internal/cli/commands/task.go` (2,230 lines)
- **Next status**: `internal/cli/commands/task_next_status.go` (304 lines)
- **Dependencies**: `internal/cli/commands/task_deps.go` (779 lines)
- **Other helpers**: `task_context.go`, `task_resume.go`, `task_history.go`, etc.

### Workflow System
- **Workflow service**: `internal/workflow/service.go` (363 lines)
- **Workflow types**: `internal/workflow/types.go`
- **Workflow config schema**: `internal/config/workflow_schema.go` (149 lines)
- **Workflow parser**: `internal/config/workflow_parser.go`
- **Workflow validator**: `internal/config/workflow_validator.go`

### Repository Layer
- **Task repository**: `internal/repository/task_repository.go`
- **Work sessions**: `internal/repository/work_session_repository.go`
- **Task relationships**: `internal/repository/task_relationship_repository.go`

### Tests
- **Workflow tests**: `internal/cli/commands/task_workflow_test.go`
- **Status update tests**: `internal/cli/commands/task_update_status_test.go`
- **Next task tests**: `internal/cli/commands/task_next_test.go`
- **Repository tests**: `internal/repository/task_workflow_validation_test.go`

---

## Helper Functions to Extract

### 1. Workflow Loading
```go
// Extract from task.go
func loadWorkflowForCommand() (*config.WorkflowConfig, error)
```

### 2. Session Management
```go
func startWorkSession(ctx, db, taskID, agent) error
func endWorkSession(ctx, db, taskID, outcome, notes) error
func pauseWorkSession(ctx, db, taskID) error
```

### 3. Phase Pattern Matching
```go
func determineInStatus(currentStatus string, workflow *WorkflowConfig) (string, error)
func determineNextReadyStatus(currentStatus string, workflow *WorkflowConfig) (string, error)
func isPhasePair(status1, status2 string) bool // ready_for_X and in_X
```

### 4. Transition Selection
```go
func selectBestForwardTransition(current string, workflow) (string, error)
func selectBestBackwardTransition(current string, workflow) (string, error)
func isForwardTransition(from, to string, workflow) bool
```

---

## Database Schema Considerations

**Good news**: No database migrations needed!

**Current schema** supports everything E13 requires:
- `tasks.status` is VARCHAR, not enum (flexible)
- `work_sessions` table already exists
- `task_history` tracks all transitions

**Only enhancement needed**: Add indexes for phase-based queries (optional optimization).

```sql
-- Optional: Add index for LIKE queries (if phase filtering becomes slow)
CREATE INDEX IF NOT EXISTS idx_tasks_status_pattern ON tasks(status);
```

---

## Conclusion

### Strengths of Current Implementation
1. ✅ Workflow system already exists and is robust
2. ✅ Repository layer already validates transitions
3. ✅ Work session tracking is in place
4. ✅ Commands are well-isolated (no cross-dependencies)
5. ✅ `task_next_status.go` is 90% of what `finish` should be

### Weaknesses to Address
1. ❌ Hardcoded status targets in 5 commands
2. ❌ Workflow service underutilized (only 1 command uses it)
3. ❌ Duplicate workflow loading code (50+ lines repeated)
4. ❌ No phase-based filtering in `task next`
5. ❌ Confusing overlap between `complete`, `approve`, `next-status`

### Implementation Complexity
- **Total files to modify**: ~15 files
- **Total files to create**: ~3 new command files
- **Lines of code affected**: ~2,500 lines
- **Reusable code**: ~600 lines can be extracted to helpers
- **Test files to update**: ~5 test files

### Recommended Approach
**Incremental refactoring** over 3-4 weeks:
1. Add new commands alongside old ones (non-breaking)
2. Add deprecation warnings
3. Update documentation
4. Remove old commands after 2-3 releases

### Success Criteria
- ✅ All commands work with any custom workflow
- ✅ No hardcoded status assumptions
- ✅ Clear phase-based semantics
- ✅ Zero workflow-related bugs in AI orchestrator
- ✅ 90% of users migrate within 2 releases

---

**Research Complete**: 2026-01-11
**Next Step**: Create implementation tasks in E13 feature breakdown
