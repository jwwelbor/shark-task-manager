# Architectural Review and Design: Feature Get Workflow Enhancement

**Date:** 2026-01-02
**Task:** Apply workflow config logic to feature get screen
**Reference Implementation:** T-E07-F16-002 (Task Creation Workflow Config)

---

## Executive Summary

This document provides:
1. **Code Review** of T-E07-F16-002 implementation
2. **Architecture Design** for applying workflow config to feature get screen
3. **API Contracts** for shared workflow services
4. **Implementation Recommendations** and technical considerations

### Key Findings

✅ **T-E07-F16-002 Implementation Quality:** Well-architected, follows good practices
⚠️ **Current Feature Get Screen:** Hardcodes task statuses in multiple places
🎯 **Recommendation:** Create shared `WorkflowService` to centralize workflow config access
📋 **Scope:** Update feature get, epic get, and task list commands to use workflow config

---

## Part 1: Code Review of T-E07-F16-002

### Implementation Location

**File:** `/internal/taskcreation/creator.go`
**Method:** `getInitialTaskStatus()` (lines 446-467)

### Architecture Analysis

#### ✅ **Strengths**

1. **Clean Separation of Concerns**
   - Workflow config loading isolated in dedicated helper method
   - Creator doesn't know about config file format details
   - Fallback behavior clearly defined

2. **Proper Error Handling**
   ```go
   workflow, err := config.LoadWorkflowConfig(configPath)
   if err != nil || workflow == nil {
       return models.TaskStatusTodo  // Safe fallback
   }
   ```
   - Graceful degradation when config missing
   - No panics or undefined behavior

3. **Leverages Existing Infrastructure**
   - Uses `config.LoadWorkflowConfig()` (with caching)
   - Uses `config.StartStatusKey` constant
   - Respects special_statuses._start_ convention

4. **Well-Documented**
   - Clear function comments explaining behavior
   - Documents fallback strategy
   - Explains special_statuses._start_ lookup

#### ⚠️ **Areas for Improvement**

1. **Code Duplication Risk**
   - This pattern will need to be repeated in other commands
   - No shared service for workflow config access
   - Each command builds its own configPath

2. **Hardcoded Config Path Construction**
   ```go
   configPath := filepath.Join(c.projectRoot, ".sharkconfig.json")
   ```
   - Magic string ".sharkconfig.json" should be constant
   - Pattern repeated across codebase

3. **Limited Workflow Metadata Usage**
   - Only uses `special_statuses._start_`
   - Doesn't leverage `status_metadata` for colors, descriptions, phases
   - Feature get screen could benefit from this metadata

### Code Quality: **8/10**

**Deductions:**
- -1 for code duplication risk
- -1 for hardcoded config path

**Overall Assessment:** Well-designed implementation that correctly applies workflow config to task creation. Ready for production use, but creates technical debt if pattern is repeated without abstraction.

---

## Part 2: Current Feature Get Screen Architecture

### Location Analysis

**File:** `/internal/cli/commands/feature.go`
**Functions:**
- `runFeatureGet()` (lines 445-587) - Main command handler
- `renderFeatureDetails()` (lines 633-734) - Rendering logic

### Current Status Display Issues

#### Issue 1: Hardcoded Status Breakdown (Lines 966-974)

**Location:** `/internal/repository/task_repository.go:952-974`

```go
func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
    // ...query...

    // ❌ HARDCODED: Initialize breakdown with all statuses set to 0
    breakdown := map[models.TaskStatus]int{
        models.TaskStatusTodo:           0,
        models.TaskStatusInProgress:     0,
        models.TaskStatusBlocked:        0,
        models.TaskStatusReadyForReview: 0,
        models.TaskStatusCompleted:      0,
        models.TaskStatusArchived:       0,
    }
    // ...
}
```

**Problem:** Returns only hardcoded statuses, ignoring workflow config.

#### Issue 2: No Status Ordering (Lines 682-694)

**Location:** `/internal/cli/commands/feature.go:682-694`

```go
// Task status breakdown
if len(statusBreakdown) > 0 {
    pterm.DefaultSection.Println("Task Status Breakdown")
    fmt.Println()
    breakdownData := pterm.TableData{}
    // ❌ RANDOM ORDER: map iteration is non-deterministic
    for status, count := range statusBreakdown {
        breakdownData = append(breakdownData, []string{
            string(status),
            fmt.Sprintf("%d", count),
        })
    }
    _ = pterm.DefaultTable.WithData(breakdownData).Render()
}
```

**Problem:** Status breakdown displays in random order (Go map iteration is non-deterministic).

#### Issue 3: Missing Status Metadata

**Current Output:**
```
Task Status Breakdown
todo                5
in_progress        3
completed          2
```

**What Users Want:**
```
Task Status Breakdown
Status                Count  Phase        Description
draft                 5      planning     Task created but not yet refined
ready_for_development 3      development  Spec complete, ready for implementation
completed             2      done         Task finished and approved
```

#### Issue 4: Tasks Table Uses Hardcoded Statuses (Lines 706-729)

```go
tableData := pterm.TableData{
    {"Key", "Title", "Status", "Priority", "Agent"},
}

for _, task := range tasks {
    // ...
    tableData = append(tableData, []string{
        task.Key,
        title,
        string(task.Status),  // ❌ Just dumps status string, no metadata
        fmt.Sprintf("%d", task.Priority),
        agent,
    })
}
```

**Problem:** Status displayed as raw string, no colors, no phase grouping.

---

## Part 3: Architecture Design

### Design Goals

1. **Centralize Workflow Config Access** - One service for all commands
2. **Respect Workflow Ordering** - Display statuses in workflow-defined order
3. **Leverage Status Metadata** - Use colors, descriptions, phases from config
4. **Maintain Backward Compatibility** - Graceful fallback for projects without workflow config
5. **Minimize Code Duplication** - Reusable across feature get, epic get, task list

### Proposed Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Commands Layer                       │
│  (feature get, epic get, task list, task create)           │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────────────────┐
│              WorkflowService (NEW)                          │
│  - LoadWorkflow(projectRoot) → *WorkflowConfig              │
│  - GetInitialStatus() → TaskStatus                          │
│  - GetAllStatuses() → []string (ordered)                    │
│  - GetStatusMetadata(status) → StatusMetadata               │
│  - FormatStatusForDisplay(status) → FormattedStatus         │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ↓
┌─────────────────────────────────────────────────────────────┐
│           config.LoadWorkflowConfig()                       │
│  (existing, with caching)                                   │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

#### 1. WorkflowService (NEW)

**Package:** `internal/workflow/service.go`

**Responsibilities:**
- Load workflow config from project root
- Provide ordered list of statuses (respecting workflow phases)
- Format statuses for display (with metadata)
- Return initial status for task creation
- Cache workflow config per project root

**Why a new package?**
- `internal/config` handles config file I/O and parsing
- `internal/workflow` handles workflow-specific business logic
- Follows Single Responsibility Principle
- Makes testing easier (mock WorkflowService, not config.LoadWorkflowConfig)

#### 2. TaskRepository Enhancement

**Changes:**
- `GetStatusBreakdown()` returns ALL statuses (including zero counts)
- Use workflow config to determine which statuses to include
- Return breakdown in workflow-defined order

#### 3. Feature Get Command Enhancement

**Changes:**
- Inject WorkflowService
- Pass status metadata to rendering layer
- Sort status breakdown by workflow order
- Color-code statuses in terminal output

---

## Part 4: API Contracts

### WorkflowService Interface

```go
package workflow

import (
    "context"
    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// Service provides workflow configuration and status management.
// This service centralizes workflow config access across all CLI commands.
type Service struct {
    projectRoot string
    workflow    *config.WorkflowConfig
}

// NewService creates a new workflow service for the given project root.
// It loads workflow config from .sharkconfig.json (or uses default if not found).
//
// Example:
//   service := workflow.NewService("/path/to/project")
//   initialStatus := service.GetInitialStatus()
func NewService(projectRoot string) *Service

// GetWorkflow returns the loaded workflow configuration.
// Returns default workflow if config not found or invalid.
func (s *Service) GetWorkflow() *config.WorkflowConfig

// GetInitialStatus returns the first entry status from workflow config.
// Falls back to "todo" if workflow config not found.
//
// Example:
//   status := service.GetInitialStatus()  // Returns "draft" if workflow defines it
func (s *Service) GetInitialStatus() models.TaskStatus

// GetAllStatuses returns all statuses defined in workflow config, in workflow order.
// Order is determined by workflow phases and status_flow topology.
//
// Example:
//   statuses := service.GetAllStatuses()
//   // Returns: ["draft", "ready_for_refinement", "in_refinement", "ready_for_development", ...]
func (s *Service) GetAllStatuses() []string

// GetStatusMetadata returns metadata for a given status.
// Returns empty metadata if status not found in config.
//
// Example:
//   meta := service.GetStatusMetadata("ready_for_development")
//   fmt.Println(meta.Description)  // "Spec complete, ready for implementation"
func (s *Service) GetStatusMetadata(status string) config.StatusMetadata

// GetStatusesByPhase returns all statuses in the given phase.
// Phases: "planning", "development", "review", "qa", "done"
//
// Example:
//   devStatuses := service.GetStatusesByPhase("development")
//   // Returns: ["in_development", "ready_for_code_review"]
func (s *Service) GetStatusesByPhase(phase string) []string

// FormatStatusForDisplay returns a formatted status string with color and metadata.
// Returns FormattedStatus with color codes, description, and raw status.
//
// Example:
//   formatted := service.FormatStatusForDisplay("in_progress")
//   fmt.Println(formatted.Colored)  // "\033[33min_progress\033[0m" (yellow)
func (s *Service) FormatStatusForDisplay(status string) FormattedStatus
```

### FormattedStatus Type

```go
// FormattedStatus represents a status formatted for display
type FormattedStatus struct {
    // Raw status string (e.g., "in_progress")
    Status string

    // Colored status with ANSI codes (e.g., "\033[33min_progress\033[0m")
    Colored string

    // Human-readable description (e.g., "Code implementation in progress")
    Description string

    // Phase (e.g., "development")
    Phase string

    // Color name (e.g., "yellow")
    ColorName string
}
```

### Enhanced TaskRepository Methods

```go
// GetStatusBreakdown returns a count of tasks by status for a feature.
// Now includes ALL workflow-defined statuses (with zero counts).
// Results are ordered by workflow phase and status_flow topology.
//
// Before:
//   {
//     "todo": 5,
//     "in_progress": 3,
//     "completed": 2
//   }
//
// After:
//   []StatusCount{
//     {Status: "draft", Count: 5, Phase: "planning"},
//     {Status: "ready_for_refinement", Count: 0, Phase: "planning"},
//     {Status: "in_refinement", Count: 0, Phase: "planning"},
//     {Status: "ready_for_development", Count: 3, Phase: "development"},
//     {Status: "completed", Count: 2, Phase: "done"},
//   }
func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) ([]StatusCount, error)

// StatusCount represents a status and its count
type StatusCount struct {
    Status      models.TaskStatus
    Count       int
    Phase       string  // From workflow metadata
    Description string  // From workflow metadata
}
```

---

## Part 5: Implementation Plan

### Phase 1: Create WorkflowService

**Files to Create:**
- `internal/workflow/service.go` - Service implementation
- `internal/workflow/service_test.go` - Unit tests
- `internal/workflow/formatter.go` - Status formatting utilities
- `internal/workflow/formatter_test.go` - Formatter tests

**Implementation Steps:**

1. Create `WorkflowService` struct
2. Implement `NewService()` constructor
3. Implement `GetInitialStatus()` (migrate from creator.go)
4. Implement `GetAllStatuses()` with workflow ordering
5. Implement `GetStatusMetadata()` wrapper
6. Implement `FormatStatusForDisplay()` with ANSI colors
7. Write comprehensive unit tests

**Acceptance Criteria:**
- [ ] All methods have unit tests
- [ ] Service correctly caches workflow config
- [ ] Graceful fallback when config missing
- [ ] Status ordering respects workflow phases
- [ ] Color formatting works with --no-color flag

### Phase 2: Update TaskRepository

**Files to Modify:**
- `internal/repository/task_repository.go`

**Changes:**

1. Add `workflow` field to `TaskRepository`
2. Update `GetStatusBreakdown()` signature:
   ```go
   // Before
   func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)

   // After
   func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) ([]StatusCount, error)
   ```
3. Return ALL workflow statuses (including zero counts)
4. Order results by workflow phase

**Acceptance Criteria:**
- [ ] Returns all workflow-defined statuses
- [ ] Includes zero counts for statuses with no tasks
- [ ] Results ordered by workflow phase
- [ ] Backward compatible (existing tests pass)

### Phase 3: Update Feature Get Command

**Files to Modify:**
- `internal/cli/commands/feature.go`

**Changes:**

1. Inject `WorkflowService` in `runFeatureGet()`
2. Update `renderFeatureDetails()` to accept workflow metadata
3. Sort status breakdown by workflow order
4. Add color-coded status display
5. Add phase grouping (optional enhancement)

**Before:**
```go
func renderFeatureDetails(feature *models.Feature, tasks []*models.Task,
    statusBreakdown map[models.TaskStatus]int, ...) {
    // ...
}
```

**After:**
```go
func renderFeatureDetails(feature *models.Feature, tasks []*models.Task,
    statusBreakdown []StatusCount, workflowService *workflow.Service, ...) {
    // ...
}
```

**Acceptance Criteria:**
- [ ] Status breakdown displays in workflow order
- [ ] Status descriptions shown (from metadata)
- [ ] Status colors applied (unless --no-color)
- [ ] Phase grouping works (optional)
- [ ] JSON output includes metadata

### Phase 4: Update Task Creation

**Files to Modify:**
- `internal/taskcreation/creator.go`

**Changes:**

1. Replace `getInitialTaskStatus()` with `WorkflowService.GetInitialStatus()`
2. Inject `WorkflowService` in `Creator` constructor
3. Remove hardcoded config path construction

**Before:**
```go
func (c *Creator) getInitialTaskStatus() models.TaskStatus {
    configPath := filepath.Join(c.projectRoot, ".sharkconfig.json")
    workflow, err := config.LoadWorkflowConfig(configPath)
    // ...
}
```

**After:**
```go
func (c *Creator) CreateTask(ctx context.Context, input CreateTaskInput) (*CreateTaskResult, error) {
    initialStatus := c.workflowService.GetInitialStatus()
    // ...
}
```

**Acceptance Criteria:**
- [ ] Code duplication removed
- [ ] Behavior unchanged (tests pass)
- [ ] Cleaner, more maintainable code

### Phase 5: Extend to Other Commands

**Files to Modify:**
- `internal/cli/commands/epic.go` - Epic get command
- `internal/cli/commands/task.go` - Task list command

**Changes:**

1. Apply same pattern to epic get command
2. Apply same pattern to task list command
3. Ensure consistent status display across all commands

**Acceptance Criteria:**
- [ ] All commands use WorkflowService
- [ ] Consistent status display UX
- [ ] All tests pass

---

## Part 6: Technical Considerations

### 1. Workflow Ordering Algorithm

**Challenge:** How to order statuses when workflow is a DAG (directed acyclic graph)?

**Solution:** Use topological sort with phase grouping:

```go
// Pseudocode for status ordering
func (s *Service) GetAllStatuses() []string {
    // 1. Group statuses by phase
    phases := []string{"planning", "development", "review", "qa", "approval", "done"}
    statusesByPhase := make(map[string][]string)

    for status, meta := range s.workflow.StatusMetadata {
        phase := meta.Phase
        if phase == "" {
            phase = "other"
        }
        statusesByPhase[phase] = append(statusesByPhase[phase], status)
    }

    // 2. Within each phase, sort alphabetically (or by custom order)
    for _, statuses := range statusesByPhase {
        sort.Strings(statuses)
    }

    // 3. Concatenate phases in order
    var result []string
    for _, phase := range phases {
        result = append(result, statusesByPhase[phase]...)
    }

    return result
}
```

### 2. Performance Considerations

**Current:** `GetStatusBreakdown()` makes 1 SQL query
**After:** `GetStatusBreakdown()` makes 1 SQL query + workflow config lookup

**Optimization:** Workflow config is cached, so performance impact is negligible.

**Benchmark Target:** < 1ms overhead for workflow config lookup (cached)

### 3. Backward Compatibility

**Projects without workflow config:**
- Use default workflow (defined in `config/workflow_default.go`)
- No breaking changes to existing behavior
- Gradual migration path

**Projects with workflow config:**
- Automatically pick up new workflow-aware display
- No code changes required

### 4. Testing Strategy

**Unit Tests:**
- Test WorkflowService in isolation (mock config.LoadWorkflowConfig)
- Test status ordering algorithm
- Test formatter with different color schemes

**Integration Tests:**
- Test feature get command with real workflow config
- Test backward compatibility with projects lacking workflow config
- Test --no-color flag interaction

**Repository Tests:**
- Test GetStatusBreakdown returns all statuses
- Test status ordering matches workflow config

### 5. Configuration Path Management

**Current Problem:** Config path hardcoded in multiple places
```go
configPath := filepath.Join(c.projectRoot, ".sharkconfig.json")
```

**Solution:** Add constant to config package:
```go
// internal/config/config.go
const DefaultConfigFilename = ".sharkconfig.json"

// Helper function
func GetConfigPath(projectRoot string) string {
    return filepath.Join(projectRoot, DefaultConfigFilename)
}
```

### 6. Color Handling

**Respect CLI flags:**
- `--no-color`: Disable all ANSI color codes
- `--json`: Omit colors from JSON output
- Default: Use colors from workflow metadata

**Implementation:**
```go
func (s *Service) FormatStatusForDisplay(status string, colorEnabled bool) FormattedStatus {
    meta := s.GetStatusMetadata(status)

    formatted := FormattedStatus{
        Status:      status,
        Description: meta.Description,
        Phase:       meta.Phase,
        ColorName:   meta.Color,
    }

    if colorEnabled {
        formatted.Colored = colorize(status, meta.Color)
    } else {
        formatted.Colored = status
    }

    return formatted
}
```

---

## Part 7: Migration Path

### Step 1: Feature Flag (Optional)

Add feature flag to control new behavior:
```json
{
  "features": {
    "workflow_aware_display": true
  }
}
```

### Step 2: Parallel Implementation

Implement new WorkflowService without breaking existing code:
- Keep old `GetStatusBreakdown()` signature
- Add new `GetStatusBreakdownWithMetadata()` method
- Migrate commands one at a time

### Step 3: Deprecation

Mark old methods as deprecated:
```go
// Deprecated: Use GetStatusBreakdownWithMetadata instead
func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
```

### Step 4: Cleanup

Remove deprecated methods in next major version.

---

## Part 8: Risk Assessment

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking change to existing commands | Low | High | Maintain backward compatibility, add integration tests |
| Performance regression | Low | Medium | Benchmark workflow config lookup, use caching |
| Workflow config parsing errors | Medium | Medium | Graceful fallback to default workflow |
| Color display issues in different terminals | Medium | Low | Test on multiple terminals, respect --no-color |

### Recommended Risk Mitigations

1. **Comprehensive Testing**
   - Add integration tests for all affected commands
   - Test with and without workflow config
   - Test --no-color flag

2. **Gradual Rollout**
   - Implement WorkflowService first
   - Migrate commands one at a time
   - Keep old code paths working

3. **Documentation**
   - Update CLI_REFERENCE.md
   - Add workflow config examples
   - Document migration path for users

---

## Part 9: Success Criteria

### Functional Requirements

- [ ] Feature get screen displays statuses in workflow order
- [ ] Status breakdown includes ALL workflow-defined statuses
- [ ] Status colors applied (unless --no-color)
- [ ] Status descriptions shown (from metadata)
- [ ] Backward compatible with projects lacking workflow config

### Non-Functional Requirements

- [ ] No performance regression (< 1ms overhead)
- [ ] Code duplication removed (DRY principle)
- [ ] Comprehensive test coverage (> 80%)
- [ ] Documentation updated

### User Experience

- [ ] Status display is intuitive and informative
- [ ] Colors improve readability
- [ ] Phase grouping helps understand workflow state
- [ ] JSON output includes metadata for programmatic access

---

## Part 10: Conclusion

### Summary

The T-E07-F16-002 implementation demonstrates a solid pattern for workflow config integration. However, without abstraction, this pattern creates technical debt through code duplication.

**Recommended Approach:**
1. Create `WorkflowService` to centralize workflow config access
2. Update `TaskRepository.GetStatusBreakdown()` to return ordered status counts
3. Enhance feature get command to display workflow-aware status breakdown
4. Extend pattern to epic get and task list commands
5. Refactor task creation to use shared WorkflowService

### Estimated Effort

- **Phase 1 (WorkflowService):** 4-6 hours
- **Phase 2 (TaskRepository):** 2-3 hours
- **Phase 3 (Feature Get):** 3-4 hours
- **Phase 4 (Task Creation):** 1-2 hours
- **Phase 5 (Other Commands):** 2-3 hours
- **Testing & Documentation:** 4-5 hours

**Total:** 16-23 hours

### Next Steps

1. Review this design document with team
2. Create task in shark for WorkflowService implementation
3. Implement Phase 1 (WorkflowService)
4. Write integration tests
5. Roll out to feature get command
6. Extend to other commands

---

## Appendix A: Example Workflow Config

```json
{
  "status_flow_version": "1.0",
  "status_flow": {
    "draft": ["ready_for_refinement", "cancelled"],
    "ready_for_refinement": ["in_refinement", "cancelled"],
    "in_refinement": ["ready_for_development", "draft"],
    "ready_for_development": ["in_development", "cancelled"],
    "in_development": ["ready_for_code_review", "blocked"],
    "ready_for_code_review": ["in_code_review"],
    "in_code_review": ["ready_for_qa", "in_development"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed", "in_qa"],
    "completed": [],
    "cancelled": [],
    "blocked": ["ready_for_development"]
  },
  "status_metadata": {
    "draft": {
      "color": "gray",
      "description": "Task created but not yet refined",
      "phase": "planning"
    },
    "ready_for_development": {
      "color": "yellow",
      "description": "Spec complete, ready for implementation",
      "phase": "development",
      "agent_types": ["developer", "ai-coder"]
    },
    "in_development": {
      "color": "yellow",
      "description": "Code implementation in progress",
      "phase": "development"
    },
    "completed": {
      "color": "green",
      "description": "Task finished and approved",
      "phase": "done"
    }
  },
  "special_statuses": {
    "_start_": ["draft"],
    "_complete_": ["completed", "cancelled"]
  }
}
```

---

## Appendix B: Example Feature Get Output (After Enhancement)

### Before (Current)

```
Feature: E04-F06

Title:       Advanced Task Metadata Filtering
Epic ID:     4
Status:      active (calculated)
Progress:    60.0%

Task Status Breakdown
ready_for_review       3
in_progress           2
todo                  5
```

### After (Enhanced)

```
Feature: E04-F06

Title:       Advanced Task Metadata Filtering
Epic ID:     4
Status:      active (calculated)
Progress:    60.0%

Task Status Breakdown
Status                      Count  Phase        Description
draft                       5      planning     Task created but not yet refined
ready_for_refinement        0      planning     Awaiting specification and analysis
ready_for_development       2      development  Spec complete, ready for implementation
in_development              0      development  Code implementation in progress
ready_for_code_review       0      review       Code complete, awaiting review
ready_for_qa                3      qa           Ready for quality assurance testing
completed                   0      done         Task finished and approved

Tasks by Phase

Planning (5 tasks):
  T-E04-F06-001  Setup metadata schema           draft            5  backend
  T-E04-F06-002  Design filter API               draft            5  architect

Development (2 tasks):
  T-E04-F06-003  Implement filter engine         ready_for_dev    8  backend
  T-E04-F06-004  Add SQL query builder           ready_for_dev    7  backend

QA (3 tasks):
  T-E04-F06-005  Test filter combinations        ready_for_qa     5  qa
  T-E04-F06-006  Load testing                    ready_for_qa     4  qa
  T-E04-F06-007  Integration tests               ready_for_qa     6  qa
```

---

**Document Version:** 1.0
**Author:** Architect Agent (Claude)
**Review Status:** Ready for Review
