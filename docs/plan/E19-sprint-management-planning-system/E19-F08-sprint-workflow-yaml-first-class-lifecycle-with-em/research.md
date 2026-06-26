# Research Report: E19-F08 — Sprint Workflow YAML

**Date**: 2026-06-26  
**Status**: Complete

---

## 1. Existing Workflow YAML Schema

Template from `internal/sharkdata/default_data/workflow/task.yaml`:

```yaml
version: "1.0"
start: <entry-status>
steps:
  <status-name>:
    phase: planning|execution|review|qa|done|blocked|paused
    color: <display-color>
    display_token: <3-letter>
    description: <human-readable>
    progress_weight: 0..1.0
    responsibility: human|agent|none
    action: pause|advance_status|spawn_agent|archive|cascade
    agent: <agent-name>           # for spawn_agent
    provider: anthropic
    model: sonnet
    skills: [<skill-name>]
    prompt: <path-to-prompt>
    outcomes:                     # for spawn_agent
      pass: <next-status>
      fail: <fallback-status>
      blocked: on_hold
    parking: true                 # optional, marks parking statuses
    is_planning: true             # optional
    exclude_from_progress: true   # optional
    terminal: true                # optional
```

Sprint YAML will use route-based `steps:` schema (E35 format).

---

## 2. Hardcoded Strings in sprint_service.go

**File**: `internal/services/sprint_service.go`

### Deletion guard (line 412)
```go
if string(sprint.Status) != "planning"
```
**Replace with**: `s.workflowSvc.GetInitialStatusString()`

### assignableSprintPhases() (line 861)
```go
return []string{"planning", "execution"}
```
**Replace with**: Remove function; derive dynamically in `assignableSprintStatuses()`

### assignableSprintStatuses() (lines 869–882)
Loop using hardcoded phase names from `assignableSprintPhases()`  
**Replace with**: `s.workflowSvc.GetStatusesByPhase("planning")` + `s.workflowSvc.GetStatusesByPhase("execution")`

### GetSprintBacklog view mode (line 1093)
```go
if string(sprintEntity.Status) == "active"
```
**Replace with**: Check `s.workflowSvc.GetStatusesByPhase("execution")` contains status

### StartSprint single-active constraint (lines 455–467)
```go
s.ListSprints(ctx, &SprintListFilters{Status: "active"})
// ... active sprint uniqueness check ...
```
**Action**: **DELETE entirely** — per design decision, multiple active sprints are valid

### StartSprint status update (line 470)
```go
s.repo.UpdateStatus(ctx, sprint.ID, models.SprintStatus("active"))
```
**Note**: This is the valid status target for `StartSprint` — leave but could use `ValidateTransition` result

### ReorderAssignment check (line 1464)
```go
if status != "planning" && status != "active"
```
**Replace with**: `!s.workflowSvc.IsStatusInPhase(status, "planning") && !s.workflowSvc.IsStatusInPhase(status, "execution")` or use `assignableSprintStatuses()`

### CloseSprintWithCarryover planning filter (line 2055)
```go
planningFilter := &sprint.SprintListFilters{Status: closeSprintStatusPtr("planning")}
```
**Replace with**: `GetStatusesByPhase("planning")[0]` or `GetInitialStatusString()`

### CloseSprintWithCarryover target status (line 2081)
```go
Status: models.SprintStatus("planning")
```
**Replace with**: `models.SprintStatus(s.workflowSvc.GetInitialStatusString())`

### GetNextTask active filter (line 1294)
```go
filters := &SprintListFilters{Status: "active"}
```
**Replace with**: First status from `GetStatusesByPhase("execution")`

---

## 3. workflow.Service API — Available Methods

From `internal/workflow/service.go` (accessed via `s.workflowSvc = workflowSvc.ForLevel(workflow.LevelSprint)`):

```go
func (s *Service) GetStatusesByPhase(phase string) []string
func (s *Service) GetInitialStatusString() string
func (s *Service) GetTerminalStatuses() []string
func (s *Service) ValidateTransition(fromStatus, toStatus string) error
func (s *Service) IsValidTransition(currentStatus, targetStatus string) bool
func (s *Service) GetValidTransitions(currentStatus string) []string
func (s *Service) GetStatusMetadata(status string) StatusInfo
func (s *Service) IsTerminalStatus(status string) bool
```

`LevelSprint` constant already defined. SprintService already initializes `s.workflowSvc = workflowSvc.ForLevel(workflow.LevelSprint)` (line 159).

---

## 4. Sprint Skills Status

All three skills exist and are ready to reference:

| Skill | Path | Status |
|-------|------|--------|
| sprint-planning | `internal/sharkdata/default_data/skills/sprint-planning/SKILL.md` | ✅ Exists |
| sprint-execution | `internal/sharkdata/default_data/skills/sprint-execution/SKILL.md` | ✅ Exists |
| sprint-analytics | `internal/sharkdata/default_data/skills/sprint-analytics/SKILL.md` | ✅ Exists |

---

## 5. Prompt Template Pattern

Existing pattern from `internal/sharkdata/default_data/prompts/task/`:
- `draft.md`, `development.md`, `completed.md`, `blocked.md`, `on_hold.md`, `code_review.md`, `qa.md`, `approval.md`, `cancelled.md`

Sprint stubs to create:
- `internal/sharkdata/default_data/prompts/sprint/planning.md`
- `internal/sharkdata/default_data/prompts/sprint/active.md`
- `internal/sharkdata/default_data/prompts/sprint/closing.md`

---

## 6. Embedded Workflow Test

**File**: `internal/config/cc040_embedded_workflow_test.go`

Existing tests:
- `TestDefaultWorkflowDataLoader_Pass3_AllSlotsPopulated` — verifies all entity types present
- `TestDefaultWorkflowDataLoader_Pass3_TaskMatchesEmbeddedStatuses` — compares loader result with embedded YAML
- `TestDefaultWorkflowDataLoader_Pass3_AllEmbeddedEntitiesLoaded` — ensures all entity workflows load

**Key comment (lines 388–389)**: "tech-debt.yaml and sprint.yaml are intentionally omitted: they are supplementary — most projects do not define them"

Sprint.yaml is optional/supplementary. The test should verify sprint.yaml loads when present without breaking when absent.

---

## 7. Files to Create/Modify

### New Files
1. `internal/sharkdata/default_data/workflow/sprint.yaml` — sprint lifecycle workflow
2. `internal/sharkdata/default_data/prompts/sprint/planning.md` — planning step prompt stub
3. `internal/sharkdata/default_data/prompts/sprint/active.md` — active step prompt stub
4. `internal/sharkdata/default_data/prompts/sprint/closing.md` — closing step prompt stub

### Modified Files
5. `internal/services/sprint_service.go` — remove single-active constraint, replace hardcoded strings
6. `internal/services/sprint_service_test.go` — update mocks/assertions for phase-aware changes
7. `internal/config/cc040_embedded_workflow_test.go` — add sprint.yaml load verification

---

## 8. Replacement Map Summary

| Sprint Service Location | Line | Hardcoded | Replacement |
|------------------------|------|-----------|-------------|
| `DeleteSprint()` | 412 | `"planning"` | `s.workflowSvc.GetInitialStatusString()` |
| `assignableSprintPhases()` | 861 | `[]string{"planning","execution"}` | **Remove function** |
| `assignableSprintStatuses()` | 869–882 | phase loop | `GetStatusesByPhase("planning") + GetStatusesByPhase("execution")` |
| `GetSprintBacklog()` | 1093 | `== "active"` | `GetStatusesByPhase("execution")` contains check |
| `StartSprint()` | 455–467 | single-active check | **DELETE** |
| `ReorderAssignment()` | 1464 | `"planning" && "active"` | use `assignableSprintStatuses()` |
| `CloseSprintWithCarryover()` | 2055 | planning filter | `GetStatusesByPhase("planning")[0]` |
| `CloseSprintWithCarryover()` | 2081 | `"planning"` target | `GetInitialStatusString()` |
| `GetNextTask()` | 1294 | `"active"` filter | `GetStatusesByPhase("execution")[0]` |

---

## 9. Recommended Task Breakdown

1. **Add sprint.yaml** (new file, follows existing YAML schema)
2. **Add sprint prompt stubs** (3 new files, minimal content)
3. **Remove single-active constraint** in `StartSprint()` (surgical delete)
4. **Replace hardcoded phase strings** with workflow.Service calls across sprint_service.go
5. **Update tests** — sprint_service_test.go + cc040 embedded workflow test
