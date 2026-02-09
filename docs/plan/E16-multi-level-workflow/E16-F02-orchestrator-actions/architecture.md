# Technical Architecture: E16-F02 Orchestrator Actions

**Feature**: E16-F02-orchestrator-actions
**Epic**: E16 Multi-Level Workflow System
**Author**: Architect Agent
**Date**: 2026-02-08
**Status**: Draft

---

## 1. Approach Overview

### Summary

This feature extends the existing task-level orchestrator action mechanism (E13) to epic and feature levels. The extension is straightforward because the existing infrastructure is already level-agnostic at the data level: `OrchestratorAction` is embedded in `StatusMetadata`, which is embedded in `WorkflowConfig`, and F01 already gives each entity level its own `WorkflowConfig`. What F02 adds is the **resolution and delivery** of those actions in transition responses, plus the **aggregated display** across all levels in `show-actions` and `validate-actions`.

### Reuse vs. New Code

**Full Reuse (zero changes needed)**:
- `config.OrchestratorAction` struct and its `Validate()`, `ValidateWithContext()` methods
- `config.PopulatedAction` struct
- `config.OrchestratorValidationError` type
- `config.ValidActionTypes` constants
- `config.ValidateAllOrchestratorActions()` function
- `config.MockActionService` (testing)

**Extension Required**:
- `OrchestratorAction.PopulateTemplate()` -- currently hardcoded to `{task_id}`; needs generic `{id}` support
- `config.DefaultActionService` -- currently wraps a single `WorkflowConfig`; needs multi-level awareness
- `validateTemplateSyntax()` -- known placeholders list needs expansion
- `workflow show-actions` command -- needs multi-level sections
- `workflow validate-actions` command -- needs multi-level sections

**New Code**:
- Orchestrator action resolution logic in `EpicService.TransitionStatus()` and `FeatureService.TransitionStatus()` (F01 creates these services; F02 adds action resolution to their transition responses)
- `TransitionResult.OrchestratorAction` field on the shared result type

### Design Principles

1. **Appropriate**: Reuse `OrchestratorAction` and `PopulatedAction` types verbatim. No new action type hierarchy.
2. **Proven**: Follow the exact pattern established by the task-level implementation in `TaskRepository.getOrchestratorAction()` and `DefaultActionService`.
3. **Simple**: Template resolution is a string replacement. Action lookup is a map access on `StatusMetadata`. No new abstractions needed.

---

## 2. Component Changes

### 2.a Config Layer -- OrchestratorAction Resolution

#### 2.a.1 PopulateTemplate Generalization

**File**: `internal/config/orchestrator_action.go`

**What changes**: The `PopulateTemplate` method currently only replaces `{task_id}`. It needs to also replace `{id}` as a generic placeholder, plus level-specific placeholders `{epic_id}` and `{feature_id}`.

**Why**: FR-003 requires `{id}` in `instruction_template` to be replaced with the entity key. Backward compatibility with `{task_id}` in existing task workflows must be preserved.

**How**:

```go
// PopulateTemplate replaces template variables with actual values.
// Supports {id} (generic), {task_id}, {epic_id}, {feature_id}.
// The entityID parameter is the entity key (e.g., "E16", "E16-F01", "T-E07-F01-001").
func (oa *OrchestratorAction) PopulateTemplate(entityID string) string {
    result := oa.InstructionTemplate
    result = strings.Replace(result, "{id}", entityID, -1)
    result = strings.Replace(result, "{task_id}", entityID, -1)
    result = strings.Replace(result, "{epic_id}", entityID, -1)
    result = strings.Replace(result, "{feature_id}", entityID, -1)
    return result
}
```

The existing `PopulateTemplate(taskID string)` signature is unchanged (same single-parameter pattern). The parameter name changes from `taskID` to `entityID` conceptually, but Go does not care about parameter names in function signatures, so this is not a breaking change. Every existing call site passes the entity key and gets the right behavior because `{task_id}` is still replaced.

**Risk mitigation**: All existing tests pass because `{task_id}` replacement still works identically. New tests verify `{id}`, `{epic_id}`, and `{feature_id}` placeholders.

#### 2.a.2 Template Syntax Validation Update

**File**: `internal/config/orchestrator_action.go`

**What changes**: `validateTemplateSyntax()` currently lists `{task_id}` as the only known placeholder. Expand the set.

**Why**: Without this, templates using `{id}`, `{epic_id}`, or `{feature_id}` produce spurious "Unknown placeholder" warnings.

**How**:

```go
func validateTemplateSyntax(template string) []string {
    warnings := []string{}

    // Check for at least one known placeholder
    knownPlaceholders := map[string]bool{
        "{id}":         true,
        "{task_id}":    true,
        "{epic_id}":    true,
        "{feature_id}": true,
    }

    hasKnownPlaceholder := false
    placeholders := extractPlaceholders(template)
    for _, p := range placeholders {
        if knownPlaceholders[p] {
            hasKnownPlaceholder = true
        }
    }

    if !hasKnownPlaceholder {
        warnings = append(warnings, "Template does not contain any known placeholder ({id}, {task_id}, {epic_id}, {feature_id})")
    }

    // Check for malformed placeholders (unclosed brace)
    if strings.Contains(template, "{") && !strings.Contains(template, "}") {
        warnings = append(warnings, "Malformed placeholder: unclosed brace {")
    }

    // Check for unknown placeholders
    for _, placeholder := range placeholders {
        if !knownPlaceholders[placeholder] {
            warnings = append(warnings, fmt.Sprintf("Unknown placeholder %s (supported: {id}, {task_id}, {epic_id}, {feature_id})", placeholder))
        }
    }

    // Check maximum length
    if len(template) > 2000 {
        warnings = append(warnings, "Template exceeds 2000 character limit")
    }

    return warnings
}
```

#### 2.a.3 ValidateWithContext Suggestion Update

**File**: `internal/config/orchestrator_action.go`

**What changes**: Line 89 in `ValidateWithContext` suggests `"Add instruction_template with {task_id} placeholder"`. Update the suggestion to mention `{id}` as the recommended placeholder.

**Why**: `{id}` is the new recommended generic placeholder. `{task_id}` still works but is level-specific.

**How**: Change the `SuggestedFix` string:

```go
SuggestedFix: "Add instruction_template with {id} placeholder (also supports {task_id}, {epic_id}, {feature_id})",
```

#### 2.a.4 Multi-Level ActionService

**File**: `internal/config/action_service.go`

**What changes**: `DefaultActionService` currently wraps a single `WorkflowConfig`. After F01, configs are loaded via `MultiLevelWorkflow`. The `ActionService` interface and `DefaultActionService` need level-aware variants.

**Why**: FR-005 and FR-007 require `show-actions` and `validate-actions` to operate across all three levels. The `ActionService` is the appropriate abstraction for this.

**How**: Add a new `MultiLevelActionService` that composes three `DefaultActionService` instances:

```go
// MultiLevelActionService provides orchestrator action access across all entity levels.
type MultiLevelActionService struct {
    Epic    ActionService
    Feature ActionService
    Task    ActionService
}

// NewMultiLevelActionService creates action services for all three levels.
func NewMultiLevelActionService(configPath string) (*MultiLevelActionService, error) {
    multi, err := LoadMultiLevelWorkflow(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load multi-level workflow: %w", err)
    }

    return &MultiLevelActionService{
        Epic:    newActionServiceFromWorkflow(multi.GetWorkflowForLevel("epic")),
        Feature: newActionServiceFromWorkflow(multi.GetWorkflowForLevel("feature")),
        Task:    newActionServiceFromWorkflow(multi.GetWorkflowForLevel("task")),
    }, nil
}
```

Add a constructor that accepts a pre-loaded `*WorkflowConfig` instead of loading from disk:

```go
// newActionServiceFromWorkflow creates an action service from a pre-loaded workflow config.
func newActionServiceFromWorkflow(workflow *WorkflowConfig) *DefaultActionService {
    return &DefaultActionService{
        workflow: workflow,
    }
}
```

The existing `NewActionService(configPath)` constructor remains unchanged for backward compatibility.

### 2.b Service Layer -- Action Resolution in Transition Responses

#### 2.b.1 TransitionResult Enhancement

**File**: `internal/services/transition_types.go` (created by F01)

**What changes**: Add `OrchestratorAction` field to `TransitionResult`.

**Why**: FR-001 and FR-002 require transition responses to include the orchestrator action for the new status.

**How**:

```go
type TransitionResult struct {
    EntityType         string                 `json:"entity_type"`
    EntityKey          string                 `json:"entity_key"`
    FromStatus         string                 `json:"from_status"`
    ToStatus           string                 `json:"to_status"`
    Transitioned       bool                   `json:"transitioned"`
    Message            string                 `json:"message,omitempty"`
    OrchestratorAction *config.PopulatedAction `json:"orchestrator_action,omitempty"`
}
```

The `omitempty` tag ensures the field is absent from JSON when no action is defined (FR: null/omitted when no action).

#### 2.b.2 NextStatusInfo Enhancement

**File**: `internal/services/transition_types.go`

**What changes**: Add `OrchestratorAction` to `NextStatusInfo` for preview responses.

**Why**: When `next-status --json` returns available transitions, the orchestrator needs to see the action that will be triggered.

**How**:

```go
type NextStatusInfo struct {
    EntityType           string                   `json:"entity_type"`
    EntityKey            string                   `json:"entity_key"`
    CurrentStatus        string                   `json:"current_status"`
    CurrentPhase         string                   `json:"current_phase,omitempty"`
    AvailableTransitions []TransitionInfoWithAction `json:"available_transitions"`
    IsTerminal           bool                     `json:"is_terminal"`
}

// TransitionInfoWithAction extends TransitionInfo with the action that would trigger.
type TransitionInfoWithAction struct {
    workflow.TransitionInfo
    OrchestratorAction *config.PopulatedAction `json:"orchestrator_action,omitempty"`
}
```

#### 2.b.3 EpicService.TransitionStatus -- Add Action Resolution

**File**: `internal/services/epic_service.go` (created by F01)

**What changes**: After F01 performs the status transition, F02 adds action resolution: look up `orchestrator_action` in `StatusMetadata` for the new status, populate the template with the epic key, and attach it to `TransitionResult`.

**Why**: FR-001 (epic transition responses include `orchestrator_action`).

**How**: Add a helper method and call it at the end of `TransitionStatus`:

```go
func (s *EpicService) TransitionStatus(ctx context.Context, epicKey string, targetStatus string, force bool) (*TransitionResult, error) {
    // ... F01 transition logic (get epic, validate, update) ...

    // F02: Resolve orchestrator action for the new status
    action := s.resolveAction(epicKey, targetStatus)

    return &TransitionResult{
        EntityType:         "epic",
        EntityKey:          epicKey,
        FromStatus:         currentStatus,
        ToStatus:           targetStatus,
        Transitioned:       true,
        OrchestratorAction: action,
    }, nil
}

// resolveAction looks up and populates the orchestrator action for a status.
// Returns nil if no action is defined (not an error).
func (s *EpicService) resolveAction(entityKey string, status string) *config.PopulatedAction {
    wf := s.workflowSvc.GetWorkflow()
    if wf == nil || wf.StatusMetadata == nil {
        return nil
    }

    meta, exists := wf.StatusMetadata[status]
    if !exists || meta.OrchestratorAction == nil {
        return nil
    }

    return &config.PopulatedAction{
        Action:      meta.OrchestratorAction.Action,
        AgentType:   meta.OrchestratorAction.AgentType,
        Skills:      meta.OrchestratorAction.Skills,
        Instruction: meta.OrchestratorAction.PopulateTemplate(entityKey),
    }
}
```

This follows the identical pattern as `TaskRepository.getOrchestratorAction()` (lines 1029-1059 of `task_repository.go`), but placed in the service layer where it belongs.

#### 2.b.4 FeatureService.TransitionStatus -- Add Action Resolution

**File**: `internal/services/feature_service.go` (created by F01)

**What changes**: Same as epic; add `resolveAction` and attach to `TransitionResult`.

**Why**: FR-002.

**How**: Identical pattern to `EpicService.resolveAction`, substituting the feature key.

#### 2.b.5 GetNextStatus -- Add Actions to Available Transitions

**File**: `internal/services/epic_service.go` and `internal/services/feature_service.go`

**What changes**: Enrich the `AvailableTransitions` list with the orchestrator action that would be triggered for each transition target.

**Why**: When the orchestrator calls `next-status --json`, it needs to see which actions will fire for each possible transition, enabling intelligent dispatch decisions.

**How**:

```go
func (s *EpicService) GetNextStatus(ctx context.Context, epicKey string) (*NextStatusInfo, error) {
    // ... F01 logic: get epic, get transitions, build base info ...

    // F02: Enrich transitions with orchestrator actions
    enrichedTransitions := make([]TransitionInfoWithAction, 0, len(transitions))
    for _, t := range transitions {
        enriched := TransitionInfoWithAction{
            TransitionInfo: t,
            OrchestratorAction: s.resolveAction(epicKey, t.TargetStatus),
        }
        enrichedTransitions = append(enrichedTransitions, enriched)
    }

    return &NextStatusInfo{
        EntityType:           "epic",
        EntityKey:            epicKey,
        CurrentStatus:        currentStatus,
        CurrentPhase:         currentMeta.Phase,
        AvailableTransitions: enrichedTransitions,
        IsTerminal:           s.workflowSvc.IsTerminalStatus(currentStatus),
    }, nil
}
```

### 2.c CLI Commands

#### 2.c.1 Epic Update and Next-Status -- Surface Action in Output

**Files**: `internal/cli/commands/epic_next_status.go` (F01), `internal/cli/commands/epic.go`

**What changes**: The CLI commands already return `TransitionResult` or `NextStatusInfo` from the service. Since F02 adds `OrchestratorAction` to those structs, the JSON output automatically includes the action via `cli.OutputJSON()`. For human-readable output, add a call to the existing `displayOrchestratorAction()` helper.

**Why**: FR-001 requires the action in transition responses.

**How** (in `runEpicNextStatus` after successful transition):

```go
// JSON output already includes OrchestratorAction via struct serialization
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(result)
}

// Human-readable output
cli.Success(fmt.Sprintf("Epic %s: %s -> %s", result.EntityKey, result.FromStatus, result.ToStatus))
displayOrchestratorAction(result.OrchestratorAction)
```

The `displayOrchestratorAction` function already exists in `internal/cli/commands/task.go` (line 2651). It should be extracted to a shared location (see 2.c.4).

#### 2.c.2 Feature Update and Next-Status -- Surface Action in Output

**Files**: `internal/cli/commands/feature_next_status.go` (F01), `internal/cli/commands/feature.go`

**What changes**: Same as epic; surface `OrchestratorAction` from `TransitionResult` in JSON and human-readable output.

**Why**: FR-002.

#### 2.c.3 Workflow Show-Actions -- Multi-Level Extension

**File**: `internal/cli/commands/workflow_show_actions.go`

**What changes**: Currently loads a single `WorkflowConfig` and displays actions from it. Extend to load all three levels and display each in its own section.

**Why**: FR-005 and FR-006.

**How**:

Add a `--level` flag to filter by entity level (default: all levels):

```go
func init() {
    workflowCmd.AddCommand(workflowShowActionsCmd)
    workflowShowActionsCmd.Flags().StringVar(&showActionsStatus, "status", "", "Filter by status")
    workflowShowActionsCmd.Flags().StringVar(&showActionsActionType, "action-type", "", "Filter by action type")
    workflowShowActionsCmd.Flags().StringVar(&showActionsLevel, "level", "", "Filter by level (epic, feature, task)")
}
```

The JSON output structure changes from a flat list to a grouped structure:

```go
type MultiLevelActionsDisplay struct {
    EpicActions    *WorkflowActionsDisplay `json:"epic_actions,omitempty"`
    FeatureActions *WorkflowActionsDisplay `json:"feature_actions,omitempty"`
    TaskActions    *WorkflowActionsDisplay `json:"task_actions,omitempty"`
    Summary        MultiLevelActionsSummary `json:"summary"`
}

type MultiLevelActionsSummary struct {
    EpicTotal          int `json:"epic_total"`
    EpicWithActions    int `json:"epic_with_actions"`
    FeatureTotal       int `json:"feature_total"`
    FeatureWithActions int `json:"feature_with_actions"`
    TaskTotal          int `json:"task_total"`
    TaskWithActions    int `json:"task_with_actions"`
}
```

Implementation approach:

```go
func runWorkflowShowActions(cmd *cobra.Command, args []string) error {
    configPath, err := cli.GetConfigPath()
    if err != nil {
        return fmt.Errorf("failed to get config path: %w", err)
    }

    // Load multi-level workflow (F01 provides this function)
    multi := config.LoadMultiLevelWorkflowOrDefault(configPath)

    // Build display for each level
    display := &MultiLevelActionsDisplay{}

    if showActionsLevel == "" || showActionsLevel == "epic" {
        epicWf := multi.GetWorkflowForLevel("epic")
        display.EpicActions = buildActionsDisplay(epicWf, showActionsStatus, showActionsActionType)
    }
    if showActionsLevel == "" || showActionsLevel == "feature" {
        featureWf := multi.GetWorkflowForLevel("feature")
        display.FeatureActions = buildActionsDisplay(featureWf, showActionsStatus, showActionsActionType)
    }
    if showActionsLevel == "" || showActionsLevel == "task" {
        taskWf := multi.GetWorkflowForLevel("task")
        display.TaskActions = buildActionsDisplay(taskWf, showActionsStatus, showActionsActionType)
    }

    // Build summary
    display.Summary = buildMultiLevelSummary(display)

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(display)
    }

    displayMultiLevelActionsHumanReadable(display)
    return nil
}
```

The existing `buildActionsDisplay()` function is already level-agnostic (it takes a `*WorkflowConfig`), so it can be called three times without modification.

Human-readable output format:

```
Workflow Orchestrator Actions
================================================================

--- Epic Workflow Actions ---

Planning Phase:
  STATUS                   ACTION          AGENT TYPE       SKILLS
  ready_for_research       spawn_agent     researcher       discovery, research

Development Phase:
  ...

--- Feature Workflow Actions ---

Planning Phase:
  ...

--- Task Workflow Actions ---

Development Phase:
  ...

Summary:
  Epic:    2 of 8 statuses have actions
  Feature: 3 of 12 statuses have actions
  Task:    5 of 19 statuses have actions
```

#### 2.c.4 Extract Shared displayOrchestratorAction

**Current location**: `internal/cli/commands/task.go` (line 2651)

**What changes**: Move `displayOrchestratorAction()` to a shared file so it can be called from epic, feature, and task command files.

**New file**: `internal/cli/commands/orchestrator_display.go`

**Why**: Avoid duplication; epic and feature next-status commands need the same display logic.

**How**: Extract without changing the function signature:

```go
// internal/cli/commands/orchestrator_display.go
package commands

import (
    "fmt"
    "strings"
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

// displayOrchestratorAction displays the orchestrator action summary in human-readable format
func displayOrchestratorAction(action *config.PopulatedAction) {
    // ... existing implementation from task.go ...
}
```

#### 2.c.5 Workflow Validate-Actions -- Multi-Level Extension

**File**: `internal/cli/commands/workflow_validate_actions.go`

**What changes**: Currently validates a single workflow config. Extend to validate all three levels independently and report per-level results.

**Why**: FR-007.

**How**: Same approach as show-actions -- load `MultiLevelWorkflow`, call `validateWorkflowActions()` for each level (function is already level-agnostic), aggregate results:

```go
type MultiLevelValidationReport struct {
    Valid         bool              `json:"valid"`
    StrictMode    bool              `json:"strict_mode"`
    EpicReport    *ValidationReport `json:"epic_report,omitempty"`
    FeatureReport *ValidationReport `json:"feature_report,omitempty"`
    TaskReport    *ValidationReport `json:"task_report,omitempty"`
}
```

Add `--level` flag for filtering (same as show-actions).

Human-readable output:

```
Validating workflow orchestrator actions...

--- Epic Workflow ---
[results for each epic status]

--- Feature Workflow ---
[results for each feature status]

--- Task Workflow ---
[results for each task status]

Overall Summary:
  Epic:    Valid (2 actions validated)
  Feature: Valid (3 actions validated)
  Task:    2 errors, 1 warning
```

### 2.d Models -- No Changes Required

The `OrchestratorAction` and `PopulatedAction` structs already live in `internal/config/` and are entity-agnostic. No model changes are needed.

The `TransitionResult` and `NextStatusInfo` types live in `internal/services/transition_types.go` (created by F01). The only model-level change is adding the `OrchestratorAction` field to these types (covered in 2.b.1 and 2.b.2).

---

## 3. Template Variable Strategy

### Placeholder Hierarchy

| Placeholder | Scope | Replaced With | Example |
|---|---|---|---|
| `{id}` | Generic (all levels) | Entity key | `E16`, `E16-F01`, `T-E07-F01-001` |
| `{task_id}` | Task-specific | Task key | `T-E07-F01-001` |
| `{epic_id}` | Epic-specific | Epic key | `E16` |
| `{feature_id}` | Feature-specific | Feature key | `E16-F01` |

### Resolution Behavior

All four placeholders are replaced with the same value (the entity key passed to `PopulateTemplate`). This is intentional: each template is defined within a level-specific workflow section, so the calling code always passes the correct entity key. The multiple placeholder names exist for readability -- a template author writing in `epic_workflow` can use `{epic_id}` for clarity, while `{id}` works everywhere.

### Recommended Convention

- Use `{id}` in new templates (generic, works at any level)
- `{task_id}` continues to work in existing task templates (backward compatible)
- `{epic_id}` and `{feature_id}` are available for readability but are not required

### Backward Compatibility

Existing task-level configs using `{task_id}` work without modification because `PopulateTemplate` still replaces `{task_id}`. The template syntax validation warns only if NO known placeholder is present (changed from warning if `{task_id}` is absent).

### Config Example

```json
{
  "epic_workflow": {
    "status_metadata": {
      "ready_for_research": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "researcher",
          "skills": ["discovery", "research"],
          "instruction_template": "Research market and feasibility for epic {id}. Report findings."
        }
      }
    }
  },
  "feature_workflow": {
    "status_metadata": {
      "ready_for_refinement_tech": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "architect",
          "skills": ["architecture", "tech-spec"],
          "instruction_template": "Review technical feasibility and create spec for feature {id}."
        }
      }
    }
  }
}
```

---

## 4. Integration Points

### F02 Dependency on F01

F02 depends on F01 for:

1. **`config.MultiLevelWorkflow` and `LoadMultiLevelWorkflow()`** -- F02 needs these to load epic/feature workflow configs. F02 does NOT modify the parsing; it only reads the result.

2. **`workflow.Service.ForLevel()`** -- F02 uses this indirectly through `EpicService` and `FeatureService`, which F01 creates with level-specific workflow services.

3. **`services.EpicService` and `services.FeatureService`** -- F01 creates these with `TransitionStatus()` and `GetNextStatus()` methods. F02 adds orchestrator action resolution to these methods.

4. **`services.TransitionResult` and `services.NextStatusInfo`** -- F01 defines these types. F02 adds the `OrchestratorAction` field.

5. **`shark epic next-status` and `shark feature next-status` commands** -- F01 creates these. F02 modifies them to display orchestrator actions.

### Existing Task-Level Actions -- No Changes

The existing task-level action mechanism in `TaskRepository.getOrchestratorAction()` and `TaskRepository.UpdateStatusWithAction()` continues to work unchanged. F02 does not modify the task update or task next-status commands. The only change visible to the task level is the expanded placeholder set in `PopulateTemplate()`, which is backward compatible.

### ActionService Integration

The existing `config.ActionService` interface and `DefaultActionService` continue to work for task-level operations. The new `MultiLevelActionService` is used only by the `show-actions` and `validate-actions` commands, which need cross-level access.

---

## 5. Data Flow Diagrams

### 5.1 Epic Transition with Action Resolution

```
User: shark epic next-status E16 --json
       |
       v
CLI: runEpicNextStatus(cmd, args)
       |
       +--- Parse "E16" from args[0]
       +--- epicSvc := cli.GetEpicService()
       |
       +--- result, err := epicSvc.TransitionStatus(ctx, "E16", targetStatus, false)
       |       |
       |       +--- repo.GetByKey(ctx, "E16") --> epic at status "draft"
       |       +--- workflowSvc.ValidateTransition("draft", "ready_for_research") --> valid
       |       +--- repo.Update(ctx, epic{Status: "ready_for_research"})
       |       |
       |       +--- [F02] resolveAction("E16", "ready_for_research")
       |       |       +--- wf.StatusMetadata["ready_for_research"].OrchestratorAction
       |       |       +--- PopulateTemplate("E16") --> "Research... for epic E16..."
       |       |       +--- return PopulatedAction{Action: "spawn_agent", ...}
       |       |
       |       +--- return TransitionResult{
       |               EntityType: "epic",
       |               EntityKey: "E16",
       |               FromStatus: "draft",
       |               ToStatus: "ready_for_research",
       |               Transitioned: true,
       |               OrchestratorAction: {action: "spawn_agent", agent_type: "researcher", ...}
       |           }
       |
       +--- cli.OutputJSON(result)
```

Output:
```json
{
  "entity_type": "epic",
  "entity_key": "E16",
  "from_status": "draft",
  "to_status": "ready_for_research",
  "transitioned": true,
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "researcher",
    "skills": ["discovery", "research"],
    "instruction": "Research market and feasibility for epic E16. Report findings."
  }
}
```

### 5.2 Feature Transition -- No Action Defined

```
User: shark feature next-status E16-F01 --json
       |
       v
CLI: runFeatureNextStatus(cmd, args)
       |
       +--- featureSvc.TransitionStatus(ctx, "E16-F01", "active", false)
       |       |
       |       +--- [F02] resolveAction("E16-F01", "active")
       |       |       +--- StatusMetadata["active"].OrchestratorAction == nil
       |       |       +--- return nil
       |       |
       |       +--- return TransitionResult{
       |               ...,
       |               OrchestratorAction: nil  // omitted from JSON due to omitempty
       |           }
```

Output (no `orchestrator_action` field):
```json
{
  "entity_type": "feature",
  "entity_key": "E16-F01",
  "from_status": "draft",
  "to_status": "active",
  "transitioned": true
}
```

### 5.3 Show-Actions Multi-Level Flow

```
User: shark workflow show-actions --json
       |
       v
CLI: runWorkflowShowActions(cmd, args)
       |
       +--- configPath := cli.GetConfigPath()
       +--- multi := config.LoadMultiLevelWorkflowOrDefault(configPath)
       |
       +--- epicWf := multi.GetWorkflowForLevel("epic")
       +--- epicDisplay := buildActionsDisplay(epicWf, "", "")
       |       +--- iterate epicWf.StatusMetadata
       |       +--- collect statuses with OrchestratorAction != nil
       |
       +--- featureWf := multi.GetWorkflowForLevel("feature")
       +--- featureDisplay := buildActionsDisplay(featureWf, "", "")
       |
       +--- taskWf := multi.GetWorkflowForLevel("task")
       +--- taskDisplay := buildActionsDisplay(taskWf, "", "")
       |
       +--- assemble MultiLevelActionsDisplay
       +--- cli.OutputJSON(display)
```

---

## 6. Implementation Waves

### Wave 1: Template Generalization (Config Layer, No CLI Changes)

**Tasks** (in order):

1. **Expand `PopulateTemplate` to support `{id}`, `{epic_id}`, `{feature_id}`**
   - File: `internal/config/orchestrator_action.go`
   - Complexity: XS
   - Tests: Verify `{id}`, `{epic_id}`, `{feature_id}` are replaced; verify `{task_id}` still works
   - Independent: Yes (no dependencies)

2. **Update `validateTemplateSyntax` known placeholder set**
   - File: `internal/config/orchestrator_action.go`
   - Complexity: XS
   - Tests: Verify no warning for `{id}`, `{epic_id}`, `{feature_id}`; verify unknown placeholders still warn
   - Independent: Yes

3. **Update `ValidateWithContext` suggested fix message**
   - File: `internal/config/orchestrator_action.go`
   - Complexity: XS
   - Tests: Verify updated suggestion text
   - Independent: Yes

Tasks 1-3 can be implemented in parallel as a single commit.

### Wave 2: Service Layer Action Resolution

**Depends on**: Wave 1 (for `PopulateTemplate` changes), F01 Wave 3 (for `EpicService`, `FeatureService`, `TransitionResult`)

4. **Add `OrchestratorAction` field to `TransitionResult` and `NextStatusInfo`**
   - File: `internal/services/transition_types.go`
   - Complexity: XS
   - Tests: Verify JSON serialization with and without action (omitempty behavior)

5. **Add `resolveAction` to `EpicService` and integrate into `TransitionStatus` and `GetNextStatus`**
   - File: `internal/services/epic_service.go`
   - Complexity: S
   - Tests: Valid action resolved, no action returns nil, missing status returns nil, template populated with epic key
   - Depends on: Task 4

6. **Add `resolveAction` to `FeatureService` and integrate**
   - File: `internal/services/feature_service.go`
   - Complexity: S
   - Tests: Same pattern as epic
   - Depends on: Task 4
   - Can parallelize with: Task 5

### Wave 3: CLI Display Extraction

**Depends on**: Wave 2

7. **Extract `displayOrchestratorAction` to shared file**
   - New file: `internal/cli/commands/orchestrator_display.go`
   - Remove from: `internal/cli/commands/task.go`
   - Complexity: XS
   - Tests: Existing tests continue to pass (behavioral, not structural)
   - Independent of Waves 1-2

8. **Surface `OrchestratorAction` in epic next-status human-readable output**
   - File: `internal/cli/commands/epic_next_status.go`
   - Complexity: XS (JSON already works from struct; add human-readable call)
   - Tests: Verify human-readable output shows action details

9. **Surface `OrchestratorAction` in feature next-status human-readable output**
   - File: `internal/cli/commands/feature_next_status.go`
   - Complexity: XS
   - Tests: Same pattern
   - Can parallelize with: Task 8

### Wave 4: Multi-Level Show-Actions and Validate-Actions

**Depends on**: F01 (for `LoadMultiLevelWorkflowOrDefault`), Wave 1

10. **Extend `show-actions` for multi-level display**
    - File: `internal/cli/commands/workflow_show_actions.go`
    - Complexity: M
    - New types: `MultiLevelActionsDisplay`, `MultiLevelActionsSummary`
    - Add `--level` flag
    - Tests: All levels shown, single level filter, JSON structure, backward compatibility (no flag = all)

11. **Extend `validate-actions` for multi-level validation**
    - File: `internal/cli/commands/workflow_validate_actions.go`
    - Complexity: M
    - New type: `MultiLevelValidationReport`
    - Add `--level` flag
    - Tests: All levels validated, per-level errors, strict mode, JSON structure
    - Can parallelize with: Task 10

12. **Add `MultiLevelActionService` to config package (optional -- only if show-actions/validate-actions need it)**
    - File: `internal/config/action_service.go`
    - Complexity: S
    - Tests: Multi-level construction, per-level queries
    - Note: If Tasks 10 and 11 directly use `LoadMultiLevelWorkflowOrDefault` + `buildActionsDisplay` (which is simpler), this task may be unnecessary. Decide during implementation.

---

## 7. Testing Strategy

### 7.1 Unit Tests -- Config Layer (No DB, No Mocks)

**File**: `internal/config/orchestrator_action_test.go` (existing, extend)

| Test | Description |
|------|-------------|
| `TestPopulateTemplate_GenericId` | `{id}` replaced with entity key |
| `TestPopulateTemplate_EpicId` | `{epic_id}` replaced |
| `TestPopulateTemplate_FeatureId` | `{feature_id}` replaced |
| `TestPopulateTemplate_TaskId_BackwardCompat` | `{task_id}` still works unchanged |
| `TestPopulateTemplate_MixedPlaceholders` | Template with `{id}` and `{task_id}` both replaced |
| `TestPopulateTemplate_NoPlaceholders_Unchanged` | Template without any placeholder unchanged |

**File**: `internal/config/orchestrator_action_validation_test.go` (existing, extend)

| Test | Description |
|------|-------------|
| `TestValidateTemplate_GenericId_NoWarning` | Template with `{id}` produces no warnings |
| `TestValidateTemplate_EpicId_NoWarning` | Template with `{epic_id}` produces no warnings |
| `TestValidateTemplate_NoKnownPlaceholder_Warning` | Template without any known placeholder warns |
| `TestValidateTemplate_UnknownPlaceholder_Warning` | `{unknown}` still produces warning |

### 7.2 Unit Tests -- Service Layer (Mocked Repository)

**File**: `internal/services/epic_service_test.go` (F01 creates; F02 extends)

| Test | Description |
|------|-------------|
| `TestTransitionStatus_WithAction` | Transition to status with action returns populated action |
| `TestTransitionStatus_WithoutAction` | Transition to status without action returns nil |
| `TestTransitionStatus_ActionTemplateResolved` | `{id}` in template replaced with epic key |
| `TestGetNextStatus_TransitionsIncludeActions` | Available transitions include orchestrator actions |
| `TestGetNextStatus_MixedActionsAndNoActions` | Some transitions have actions, some do not |

**File**: `internal/services/feature_service_test.go` (same pattern)

| Test | Description |
|------|-------------|
| `TestFeatureTransition_WithAction` | Same coverage as epic |
| `TestFeatureTransition_FeatureKeyInTemplate` | `{id}` replaced with feature key (e.g., `E16-F01`) |

### 7.3 Unit Tests -- CLI Commands (Mocked Service)

**File**: `internal/cli/commands/workflow_show_actions_test.go` (new or extend existing)

| Test | Description |
|------|-------------|
| `TestShowActions_MultiLevel_JSON` | JSON output contains `epic_actions`, `feature_actions`, `task_actions` |
| `TestShowActions_LevelFilter_Epic` | `--level=epic` returns only epic actions |
| `TestShowActions_LevelFilter_Task` | `--level=task` returns same as current behavior |
| `TestShowActions_NoLevelFilter` | Default returns all three levels |
| `TestShowActions_BackwardCompat_NoMultiLevel` | Config with no `epic_workflow` uses defaults |

**File**: `internal/cli/commands/workflow_validate_actions_test.go` (new or extend existing)

| Test | Description |
|------|-------------|
| `TestValidateActions_MultiLevel_AllValid` | All three levels valid |
| `TestValidateActions_MultiLevel_EpicInvalid` | Epic has error, feature/task valid |
| `TestValidateActions_LevelFilter` | `--level=feature` validates only feature |
| `TestValidateActions_StrictMode_MultiLevel` | Strict mode applied per level |

### 7.4 Performance Benchmark

**File**: `internal/config/orchestrator_action_bench_test.go` (new)

```go
func BenchmarkPopulateTemplate(b *testing.B) {
    oa := &OrchestratorAction{
        InstructionTemplate: "Research epic {id} for market viability...",
    }
    for i := 0; i < b.N; i++ {
        _ = oa.PopulateTemplate("E16")
    }
}
```

Target: <5ms per resolution (NFR-001). String replacement on a <2000 char template should be sub-microsecond.

---

## 8. Risk Mitigation

### 8.1 Backward Compatibility

**Risk**: Existing configs with `{task_id}` templates break.

**Mitigation**: `PopulateTemplate` continues to replace `{task_id}`. No existing template is invalidated. The `validateTemplateSyntax` function is updated to accept `{task_id}` as a known placeholder alongside the new ones.

**Risk**: `show-actions` JSON output format changes break orchestrator consumers.

**Mitigation**: The current `show-actions` JSON structure (`WorkflowActionsDisplay` with `workflow_actions` array) is preserved within each level. The new multi-level wrapper adds `epic_actions`, `feature_actions`, `task_actions` as top-level keys. If consumers parse `workflow_actions` directly from the root, they will need to update. To mitigate this, when `--level=task` is specified, the output could optionally return the flat format. However, the show-actions command is not part of the orchestrator's critical path (transition responses are), so this is low risk. Document the change in release notes.

### 8.2 Performance (NFR-001: <5ms)

**Risk**: Action resolution adds latency to transition commands.

**Mitigation**: Action resolution consists of:
1. One map lookup in `StatusMetadata` (O(1))
2. One `strings.Replace` call per placeholder (O(n) on template length, max 2000 chars)

Total cost: sub-microsecond. Well within the 5ms budget. No caching needed beyond what F01 already provides for `WorkflowConfig`.

Benchmark test included in testing strategy to verify.

### 8.3 Configuration Validation Edge Cases

**Risk**: Config has `orchestrator_action` in `epic_workflow` but no `epic_workflow.status_flow`, causing the action to be unreachable.

**Mitigation**: `validate-actions` checks that the status with an action actually exists in `status_flow`. The existing `ValidateAllOrchestratorActions` function iterates `StatusMetadata`, and `ValidateWorkflow` checks that all metadata statuses are in `status_flow`. These validations compose correctly at each level.

**Risk**: Empty `instruction_template` for epic/feature actions.

**Mitigation**: The existing `OrchestratorAction.Validate()` method already rejects empty templates. This validation is level-agnostic and applies to all levels.

### 8.4 F01 Dependency Timing

**Risk**: F01 is not complete when F02 starts.

**Mitigation**: Wave 1 (template generalization) has no dependency on F01 and can start immediately. Waves 2-4 depend on F01 constructs (`MultiLevelWorkflow`, `EpicService`, etc.). If F01 is delayed, Wave 1 can be merged independently. The architecture document clearly marks dependencies in each wave.

---

## 9. File Change Inventory

### Modified Files

| File | Change Description | Complexity |
|------|-------------------|------------|
| `internal/config/orchestrator_action.go` | Expand `PopulateTemplate` for `{id}`, `{epic_id}`, `{feature_id}`; update `validateTemplateSyntax` known set; update `ValidateWithContext` suggestion | XS |
| `internal/config/orchestrator_action_test.go` | Add tests for new placeholders | XS |
| `internal/config/orchestrator_action_validation_test.go` | Add tests for expanded validation | XS |
| `internal/services/transition_types.go` | Add `OrchestratorAction` to `TransitionResult` and `NextStatusInfo`; add `TransitionInfoWithAction` type | XS |
| `internal/services/epic_service.go` | Add `resolveAction()` method; integrate into `TransitionStatus()` and `GetNextStatus()` | S |
| `internal/services/feature_service.go` | Same as epic | S |
| `internal/services/epic_service_test.go` | Add action resolution tests | S |
| `internal/services/feature_service_test.go` | Add action resolution tests | S |
| `internal/cli/commands/task.go` | Remove `displayOrchestratorAction` (moved to shared file) | XS |
| `internal/cli/commands/epic_next_status.go` | Add human-readable action display | XS |
| `internal/cli/commands/feature_next_status.go` | Add human-readable action display | XS |
| `internal/cli/commands/workflow_show_actions.go` | Multi-level support, `--level` flag, new output types | M |
| `internal/cli/commands/workflow_validate_actions.go` | Multi-level support, `--level` flag, new report type | M |

### New Files

| File | Purpose | Complexity |
|------|---------|------------|
| `internal/cli/commands/orchestrator_display.go` | Extracted `displayOrchestratorAction` shared helper | XS |
| `internal/cli/commands/workflow_show_actions_test.go` | Tests for multi-level show-actions | S |
| `internal/cli/commands/workflow_validate_actions_test.go` | Tests for multi-level validate-actions | S |
| `internal/config/orchestrator_action_bench_test.go` | Performance benchmark for template resolution | XS |

### Files NOT Changed

| File | Reason |
|------|--------|
| `internal/config/config.go` | No changes needed; `PopulatedAction` is already generic |
| `internal/config/workflow_schema.go` | `StatusMetadata.OrchestratorAction` field already exists and is level-agnostic |
| `internal/config/action_service.go` | Preserved for backward compatibility; multi-level access done via direct config loading in commands |
| `internal/config/mock_action_service.go` | No interface changes to `ActionService` |
| `internal/repository/task_repository.go` | Task action resolution unchanged |
| `internal/workflow/service.go` | F01 adds `ForLevel()`; F02 does not modify workflow service |
| `internal/workflow/types.go` | No changes needed |
| `internal/models/*` | No model changes required |

---

## Appendix A: JSON Response Schema

### Epic/Feature Transition Response (with action)

```json
{
  "entity_type": "epic",
  "entity_key": "E16",
  "from_status": "draft",
  "to_status": "ready_for_research",
  "transitioned": true,
  "message": "Epic E16: draft -> ready_for_research",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "researcher",
    "skills": ["discovery", "research"],
    "instruction": "Research market, competitors, and feasibility for epic E16. Document findings."
  }
}
```

### Epic/Feature Transition Response (without action)

```json
{
  "entity_type": "feature",
  "entity_key": "E16-F01",
  "from_status": "draft",
  "to_status": "active",
  "transitioned": true,
  "message": "Feature E16-F01: draft -> active"
}
```

### Show-Actions Multi-Level Response

```json
{
  "epic_actions": {
    "workflow_actions": [
      {
        "status": "ready_for_research",
        "phase": "research",
        "color": "blue",
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "researcher",
          "skills": ["discovery", "research"],
          "instruction_template": "Research epic {id}..."
        }
      }
    ],
    "summary": {
      "total_statuses": 8,
      "statuses_with_actions": 2,
      "action_counts": {"spawn_agent": 2}
    }
  },
  "feature_actions": { "..." : "..." },
  "task_actions": { "..." : "..." },
  "summary": {
    "epic_total": 8,
    "epic_with_actions": 2,
    "feature_total": 12,
    "feature_with_actions": 3,
    "task_total": 19,
    "task_with_actions": 5
  }
}
```

### Validate-Actions Multi-Level Response

```json
{
  "valid": true,
  "strict_mode": false,
  "epic_report": {
    "valid": true,
    "total_statuses": 8,
    "valid_count": 2,
    "warning_count": 0,
    "error_count": 0,
    "results": [...]
  },
  "feature_report": { "..." : "..." },
  "task_report": { "..." : "..." }
}
```

---

## Appendix B: Dependency Graph

```
Wave 1: Template Generalization (no F01 dependency)
  T-001: Expand PopulateTemplate
  T-002: Update validateTemplateSyntax     (parallel with T-001)
  T-003: Update ValidateWithContext msg    (parallel with T-001)

Wave 2: Service Layer (depends on F01 Wave 3)
  T-004: Add OrchestratorAction to TransitionResult  (depends on F01: transition_types.go)
  T-005: EpicService.resolveAction                   (depends on T-004, F01: EpicService)
  T-006: FeatureService.resolveAction                (depends on T-004, F01: FeatureService)

Wave 3: CLI Display (depends on Wave 2)
  T-007: Extract displayOrchestratorAction to shared file
  T-008: Epic next-status human-readable action display  (depends on T-005, T-007)
  T-009: Feature next-status human-readable display      (depends on T-006, T-007)

Wave 4: Multi-Level Commands (depends on F01 Wave 1)
  T-010: show-actions multi-level extension          (depends on F01: LoadMultiLevelWorkflow)
  T-011: validate-actions multi-level extension      (depends on F01: LoadMultiLevelWorkflow)
```

---

*Last Updated*: 2026-02-08
