# BA Review: E16-F02 Orchestrator Actions

**Reviewer**: Business Analyst
**Date**: 2026-02-08
**Status**: Review Complete -- Ready for Technical Architecture with Conditions
**Feature PRD**: `docs/plan/E16-multi-level-workflow/E16-F02-orchestrator-actions/feature.md`
**Epic PRD**: `docs/plan/E16-multi-level-workflow/epic.md`
**F01 Architecture**: `docs/plan/E16-multi-level-workflow/E16-F01-core-workflow-engine/architecture.md`

---

## 1. PRD Completeness Assessment

### 1.1 User Story Assessment

| Story | Format | Value Clear? | AC Complete? | Assessment |
|-------|--------|-------------|-------------|------------|
| Story 1: Transition responses include `orchestrator_action` | Adequate -- perspective is orchestrator | Yes -- config-driven dispatch | Partially -- see Finding 1.1 | Missing ACs for `update --status` command response format; only `next-status` is exemplified |
| Story 2: `show-actions` displays all three levels | Adequate | Yes -- full dispatch map | Adequate -- covers output sections, JSON, and human-readable | The `--level` filter flag is not mentioned (see Gap 2.4) |
| Story 3: `validate-actions` validates all three levels | Adequate -- perspective is developer | Yes -- config correctness | Partially -- see Finding 1.2 | Missing AC for per-level error reporting format |

**Finding 1.1: Story 1 acceptance criteria are incomplete for `update --status` responses.**

The acceptance criteria only list:
- `shark epic update --status` response includes `orchestrator_action` when defined
- `shark feature update --status` response includes `orchestrator_action` when defined

But there is no specification of the JSON response structure for `update --status`. The `next-status` response structure is shown in the epic PRD (FR-4 example), but the `update --status` response may have a different structure. Currently, `shark epic update --status` does not return JSON at all in the existing codebase -- it only prints a success message. F01 introduces `TransitionResult` as the JSON response type. F02 must specify whether `orchestrator_action` is added to `TransitionResult` or only to the `NextStatusResult`.

**Recommendation**: Add explicit AC specifying the JSON response structure for `update --status` with `orchestrator_action`. The simplest approach is to add an `OrchestratorAction` field to the `TransitionResult` struct defined in F01's `internal/services/transition_types.go`.

**Finding 1.2: Story 3 validation acceptance criteria lack per-level error detail.**

The current ACs say: "Reports per-level validation results." But they do not specify:
- What happens when one level is valid and another is invalid
- Whether validation continues after the first error or reports all levels
- The JSON structure for multi-level validation results

### 1.2 Functional Requirements Assessment

| Requirement | Priority | Complete? | Assessment |
|-------------|----------|-----------|------------|
| REQ-F-001: Epic transition includes `orchestrator_action` | Must-Have | Partially | Missing: which JSON response structure? `TransitionResult` vs `NextStatusResult`? Both? |
| REQ-F-002: Feature transition includes `orchestrator_action` | Must-Have | Partially | Same gap as REQ-F-001 |
| REQ-F-003: Template variable replacement with `{id}` | Must-Have | **Ambiguous** | CRITICAL: Contradicts existing `{task_id}` placeholder. See Section 2.1 |
| REQ-F-004: Action types same as task level | Must-Have | Complete | Leverages existing `ValidActionTypes` array |
| REQ-F-005: Extend `show-actions` for all levels | Must-Have | Adequate | Clear output format in epic PRD FR-5 |
| REQ-F-006: JSON output for `show-actions` | Must-Have | Partially | JSON structure not defined. See Gap 2.3 |
| REQ-F-007: Extend `validate-actions` for all levels | Must-Have | Adequate | Leverages existing `validateWorkflowActions()` |

### 1.3 Non-Functional Requirements Assessment

**REQ-NF-001: Action resolution adds < 5ms to transition commands.**

This NFR is measurable in principle, but the measurement approach is not defined.

- The action resolution itself is a single map lookup into `StatusMetadata` plus a `strings.Replace` on the template. This is sub-microsecond work.
- The real question is whether the multi-level config loading (from F01) adds latency. That is F01's responsibility, not F02's.
- **Assessment**: The NFR is valid but trivially met given the implementation approach. Consider removing or replacing with a more meaningful NFR, such as: "F02 must not introduce additional file I/O or database queries beyond what F01 already performs for transitions."

---

## 2. Gap Analysis

### 2.1 CRITICAL: Template Variable Naming Inconsistency (`{task_id}` vs `{id}`)

This is the single most important design decision for F02.

**Current state (E13 task-level implementation):**
- `PopulateTemplate(taskID string)` replaces `{task_id}` with the task key
- `validateTemplateSyntax()` warns if `{task_id}` is missing
- `validateTemplateSyntax()` warns about unknown placeholders (anything other than `{task_id}`)
- The known placeholder map is hardcoded: `knownPlaceholders := map[string]bool{"{task_id}": true}`
- All existing task workflow configs use `{task_id}` in `instruction_template`

**F02 PRD specifies:**
- `{id}` as the placeholder for epic and feature templates (REQ-F-003)
- Example: `"instruction_template": "Research market, competitors, and feasibility for epic {id}..."`

**Epic PRD specifies:**
- `{id}` in all example configs for epic and feature workflows

**The contradiction**: If epic/feature templates use `{id}` and task templates use `{task_id}`, the system has two different placeholder conventions. This creates confusion and breaks the template validation logic.

**Options:**

| Option | Description | Backward Compat? | Complexity |
|--------|-------------|-------------------|------------|
| A | Add `{id}` as new placeholder for epic/feature, keep `{task_id}` for tasks | Yes | Medium -- need level-aware `PopulateTemplate` |
| B | Use `{id}` universally, deprecate `{task_id}` but still support it | Yes (with deprecation warning) | Medium -- update `PopulateTemplate` to try both |
| C | Use `{entity_id}` universally as the generic placeholder | No (breaks existing task configs) | High -- migration required |
| D | Use `{id}` for new epic/feature, keep `{task_id}` for tasks, add `{epic_id}` and `{feature_id}` as aliases | Yes | High -- many placeholders to manage |

**Recommendation**: Option B. Use `{id}` as the canonical placeholder going forward. Modify `PopulateTemplate` to replace both `{id}` and `{task_id}` with the entity key. Update `validateTemplateSyntax` to recognize both as known placeholders. Add a deprecation warning when `{task_id}` is used. This is the cleanest path forward because:
1. `{id}` is entity-type-agnostic and works for all three levels
2. Existing task configs with `{task_id}` continue to work
3. New configs (epic, feature, and future task configs) use `{id}`
4. One method signature: `PopulateTemplate(entityKey string)` -- the parameter name changes from `taskID` to `entityKey` but the underlying logic is the same

This decision MUST be made before F02 implementation begins because it affects the `OrchestratorAction` struct API, the validation logic, and the config documentation.

### 2.2 Gap: `orchestrator_action` Field Placement in Response Structs

F01's architecture document defines two response types:

```go
type TransitionResult struct {
    EntityType   string `json:"entity_type"`
    EntityKey    string `json:"entity_key"`
    FromStatus   string `json:"from_status"`
    ToStatus     string `json:"to_status"`
    Transitioned bool   `json:"transitioned"`
    Message      string `json:"message,omitempty"`
}

type NextStatusInfo struct {
    EntityType           string                   `json:"entity_type"`
    EntityKey            string                   `json:"entity_key"`
    CurrentStatus        string                   `json:"current_status"`
    CurrentPhase         string                   `json:"current_phase,omitempty"`
    AvailableTransitions []workflow.TransitionInfo `json:"available_transitions"`
    IsTerminal           bool                     `json:"is_terminal"`
}
```

Neither struct currently includes an `OrchestratorAction` field. F02 must add it to one or both.

The existing task `next-status` command (in `task_next_status.go`) uses its own `NextStatusResult` struct, which also lacks an `orchestrator_action` field -- the task-level action is currently returned only through `task update --status` in the repository layer (`task_repository.go:1049`).

**Decision needed**: Should `orchestrator_action` be in:
- `TransitionResult` only (returned after a transition is performed)?
- `NextStatusInfo` only (returned when querying available transitions)?
- Both?

**Recommendation**: Add it to `TransitionResult` (the response after a transition is actually performed). This matches the pattern described in the epic PRD's FR-4 JSON examples, where the `orchestrator_action` appears alongside the `transition` result. The `NextStatusInfo` should NOT include actions for individual transitions -- that would require looking up the action for each possible next status, which is different from the current task-level behavior. The orchestrator only needs the action for the status that was actually transitioned to.

### 2.3 Gap: JSON Structure for Multi-Level `show-actions`

The PRD says `--json` output should be "grouped by entity level" (REQ-F-006) but does not define the JSON structure.

**Current `show-actions` JSON structure** (task-only):
```json
{
  "workflow_actions": [ ... ],
  "summary": { ... }
}
```

**Proposed multi-level JSON structure** (not in PRD, needs definition):
```json
{
  "epic_workflow_actions": [ ... ],
  "feature_workflow_actions": [ ... ],
  "task_workflow_actions": [ ... ],
  "summary": {
    "epic": { ... },
    "feature": { ... },
    "task": { ... }
  }
}
```

**Recommendation**: Define this structure explicitly in the PRD. The `WorkflowActionsDisplay` struct and `ActionsSummary` struct in `workflow_show_actions.go` must be extended or replaced with a multi-level variant.

### 2.4 Gap: Level Filtering for `show-actions` and `validate-actions`

The PRD does not mention a `--level` flag to filter output to a specific entity level.

**Current `show-actions` has**: `--status` and `--action-type` filters.
**Missing**: `--level=epic|feature|task` filter.

Without this, users must parse through all three levels even when they only care about one. For orchestrators consuming JSON, this adds unnecessary data.

**Recommendation**: Add `--level` flag (optional). When omitted, show all levels (backward compatible with current task-only behavior). When specified, show only the requested level.

### 2.5 Gap: Backward Compatibility for `show-actions` JSON Output

The current `show-actions` command returns a flat `workflow_actions` array with task-level actions. Changing this to a multi-level structure is a **breaking change** for any orchestrator that parses the JSON output.

**Options:**
| Option | Description | Breaking? |
|--------|-------------|-----------|
| A | Replace flat structure with multi-level (as described in 2.3) | Yes -- existing JSON consumers break |
| B | Add multi-level fields alongside existing flat structure | No -- existing `workflow_actions` field preserved |
| C | Use `--level=task` as default behavior (flat), multi-level only with `--level=all` | No |

**Recommendation**: Option B. Keep `workflow_actions` as the task-level actions (backward compatible). Add new top-level fields `epic_workflow_actions` and `feature_workflow_actions`. Update `summary` to include per-level breakdowns. This ensures existing orchestrator scripts continue to work while new multi-level data is available.

### 2.6 Gap: `validate-actions` Multi-Level JSON Structure

Same as 2.3 but for validation. The current `ValidationReport` struct is task-only. The multi-level version needs per-level results.

**Proposed structure:**
```json
{
  "valid": true,
  "levels": {
    "epic": { "valid": true, "total_statuses": 12, ... },
    "feature": { "valid": true, "total_statuses": 13, ... },
    "task": { "valid": false, "total_statuses": 19, ... }
  },
  "overall_error_count": 1,
  "overall_warning_count": 0
}
```

### 2.7 Gap: `instruction` vs `instruction_template` in Response

The existing task-level response (shown in the PRD context) returns a resolved `instruction` field (template already populated):
```json
{
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "instruction": "Launch a developer agent to implement task T-E01-F03-002..."
  }
}
```

Note: the response contains `instruction` (resolved), NOT `instruction_template` (raw). The PRD does not clarify whether the response should contain the raw template, the resolved instruction, or both.

**Current behavior**: In `task_repository.go:1049`, `PopulateTemplate(task.Key)` is called and the resolved instruction is returned. The raw template is NOT included in the response.

**Recommendation**: Follow the existing pattern. The transition response should include a resolved `instruction` field with the entity key substituted into the template. The raw `instruction_template` should NOT be in the response (it is a config detail, not an API response field). This means the response `OrchestratorAction` may need a slightly different struct from the config `OrchestratorAction` -- or the `PopulateTemplate` method returns a new struct with `instruction` instead of `instruction_template`.

---

## 3. Edge Cases

### EC-1: Status with `orchestrator_action` defined but no `instruction_template` (CRITICAL)

**Scenario**: A user configures an orchestrator action in `epic_workflow.status_metadata` but omits or leaves empty the `instruction_template` field.

**Expected Behavior**: Validation should catch this at config load time via `OrchestratorAction.Validate()`. The existing validator already rejects empty `instruction_template`. However, when the transition occurs, if validation was bypassed (e.g., `--force`), `PopulateTemplate` would return an empty string.

**Recommendation**: No code change needed -- existing validation covers this. Document that `--force` bypasses workflow validation but NOT action validation.

### EC-2: Status with `orchestrator_action` defined but entity has no key yet (CRITICAL)

**Scenario**: During template population, the entity key is empty or malformed. For example, an epic that was just created and not yet assigned a key.

**Expected Behavior**: `PopulateTemplate("")` would produce an instruction with a blank entity reference: "Research market... for epic ..." -- which is misleading but not an error.

**Recommendation**: Add a guard in the transition service: if the entity key is empty, do not resolve the template and return `orchestrator_action: null` with a warning. This prevents orchestrators from receiving instructions with blank entity references.

### EC-3: Epic/feature workflow defines `orchestrator_action` but F01 is not yet deployed (Nice-to-have)

**Scenario**: A user adds `epic_workflow` with `orchestrator_action` entries to their `.sharkconfig.json`, but they are running a shark version that only has task-level workflow support.

**Expected Behavior**: The old parser ignores unknown top-level keys (`epic_workflow`, `feature_workflow`), so these actions are simply invisible. No error, no action.

**Recommendation**: No action needed. This is graceful degradation by design.

### EC-4: Multiple valid next statuses, each with different `orchestrator_action` (CRITICAL)

**Scenario**: An epic at `draft` has two valid transitions: `ready_for_research` (spawn_agent: researcher) and `active` (no action). The orchestrator calls `shark epic next-status E16 --json`.

**Expected Behavior**: The `next-status` command auto-selects the first transition (`ready_for_research`), performs it, and returns the `orchestrator_action` for `ready_for_research`. The orchestrator never sees the action for `active` because that transition was not taken.

**Recommendation**: This is correct behavior. However, the `--preview` mode should show the `orchestrator_action` for each available transition, so the orchestrator can make an informed choice. The PRD does not specify whether `--preview` includes actions per transition.

**Add AC**: "When `--preview` is used, each available transition in the JSON output includes its `orchestrator_action` (or null if none defined)."

### EC-5: `show-actions` when no epic/feature workflow is configured (Nice-to-have)

**Scenario**: User runs `shark workflow show-actions` with no `epic_workflow` or `feature_workflow` in `.sharkconfig.json`.

**Expected Behavior**: Two options:
1. Show actions from default workflows (which have no `orchestrator_action` entries) -- sections would be empty
2. Show only task actions (current behavior) and note that epic/feature use defaults

**Recommendation**: Option 2. When a workflow uses defaults and the defaults have no actions, display: "Epic Workflow Actions: (using defaults, no actions configured)" rather than a confusing empty section.

### EC-6: `orchestrator_action` defined on a terminal status like `completed` or `cancelled` (Nice-to-have)

**Scenario**: Config defines an `archive` action on the `completed` status. An epic transitions to `completed`.

**Expected Behavior**: The action is returned in the transition response. This is valid -- an `archive` action on `completed` makes semantic sense (clean up resources after completion).

**Recommendation**: This is valid and should work. No special handling needed. The validation should NOT reject actions on terminal statuses.

### EC-7: Template contains both `{id}` and `{task_id}` placeholders (Nice-to-have)

**Scenario**: An instruction template contains both: "Implement {task_id} for entity {id}".

**Expected Behavior**: Depends on the resolution in Gap 2.1. If Option B is chosen, both `{id}` and `{task_id}` would be replaced with the entity key, resulting in: "Implement E16-F01 for entity E16-F01".

**Recommendation**: This should be documented as expected behavior. Validation should warn about duplicate entity references but not reject the template.

### EC-8: `validate-actions` in strict mode with mixed valid/invalid levels (Nice-to-have)

**Scenario**: Epic actions are all valid, feature actions have a missing `agent_type`, task actions are all valid. User runs `shark workflow validate-actions --strict`.

**Expected Behavior**: Overall result is INVALID because one level failed. The report should show per-level results so the user knows which level to fix.

**Recommendation**: The multi-level validation report must clearly identify which level contains the error. See Gap 2.6 for the proposed JSON structure.

### EC-9: Concurrent transition attempts with conflicting actions (Nice-to-have)

**Scenario**: Two orchestrator agents independently call `shark feature next-status E16-F01`. Both see the feature at `in_refinement_ba`. One transitions to `ready_for_refinement_tech` (spawn architect), the other fails because the status has already changed.

**Expected Behavior**: The first call succeeds and returns the `orchestrator_action`. The second call fails with an error indicating the feature is no longer at the expected status.

**Recommendation**: This is handled at the F01 level (transition validation). F02 does not need additional concurrency handling. The orchestrator should check the transition result's `transitioned` field and handle failure gracefully.

---

## 4. Dependency Analysis

### 4.1 What from E16-F01 Must Be Complete

| F01 Component | Required By F02 | Status in F01 Architecture |
|---------------|-----------------|---------------------------|
| `MultiLevelWorkflow` container | `show-actions` and `validate-actions` must load all three workflow configs | Defined in architecture as `internal/config/workflow_multilevel.go` |
| `workflow.Service.ForLevel()` | Service layer needs level-specific workflow to look up `StatusMetadata` and extract `OrchestratorAction` | Defined in architecture |
| `EpicService.TransitionStatus()` | Must return `TransitionResult` that F02 extends with action | Defined in architecture |
| `FeatureService.TransitionStatus()` | Same as above | Defined in architecture |
| `config.LoadMultiLevelWorkflow()` | `show-actions` needs to load all three configs in a single call | Defined in architecture |
| `TransitionResult` and `NextStatusInfo` structs | F02 adds `OrchestratorAction` field to these | Defined in `internal/services/transition_types.go` |
| `epic next-status` CLI command | F02 adds action resolution to the output | Defined in architecture |
| `feature next-status` CLI command | Same as above | Defined in architecture |
| `epic update --status` refactored to use service | F02 adds action to service response | Defined in architecture |
| `feature update --status` refactored to use service | Same as above | Defined in architecture |

**Critical F01 Prerequisite**: F01's `TransitionResult` struct must be designed to be extensible. F02 will add an `OrchestratorAction *config.OrchestratorAction` field. If F01 hardcodes the struct without considering this, F02 will need to modify it, creating a merge conflict.

**Recommendation**: F01 should add a placeholder field: `OrchestratorAction *config.OrchestratorAction \`json:"orchestrator_action,omitempty"\`` to `TransitionResult` and `NextStatusInfo`, set to `nil`. F02 populates it. This is consistent with the F01 BA review's recommendation (Finding 2.2: "F01 should define the JSON response structure with `orchestrator_action: null` always").

### 4.2 What from E13 Can Be Reused vs. Must Be Extended

| E13 Component | Reusable? | Changes Needed for F02 |
|---------------|-----------|------------------------|
| `OrchestratorAction` struct (`orchestrator_action.go`) | **Fully reusable** | No structural changes. Same struct for all levels. |
| `OrchestratorAction.Validate()` | **Fully reusable** | No changes. Validation rules are level-agnostic. |
| `OrchestratorAction.ValidateWithContext()` | **Fully reusable** | No changes. Status context is already a parameter. |
| `OrchestratorAction.PopulateTemplate()` | **Must be extended** | Currently only replaces `{task_id}`. Must also replace `{id}`. See Gap 2.1. |
| `ValidateAllOrchestratorActions()` | **Fully reusable** | Takes `map[string]StatusMetadata` -- works for any level's metadata. |
| `validateTemplateSyntax()` | **Must be extended** | Known placeholders must include `{id}` in addition to `{task_id}`. |
| `extractPlaceholders()` | **Fully reusable** | Generic regex-based extraction. |
| `ValidActionTypes` array | **Fully reusable** | Same action types for all levels. |
| `DefaultActionService` (`action_service.go`) | **Must be extended** | Currently wraps a single `WorkflowConfig`. Must become level-aware or accept a level parameter. |
| `MockActionService` | **Must be extended** | Needs level awareness for testing. |
| `workflow_show_actions.go` | **Must be extended** | Currently loads only task workflow. Must load all three and display sections. See Gap 2.3-2.5. |
| `workflow_validate_actions.go` | **Must be extended** | Currently validates only task workflow. Must validate all three. See Gap 2.6. |
| `buildActionsDisplay()` | **Partially reusable** | Can be called per-level with different `WorkflowConfig` inputs. |
| `validateWorkflowActions()` | **Partially reusable** | Can be called per-level. Need wrapper to aggregate results. |
| `WorkflowActionsDisplay` struct | **Must be extended** | Currently flat. Need multi-level variant. |
| `ValidationReport` struct | **Must be extended** | Currently flat. Need multi-level variant. |
| Task repository `PopulateTemplate` call (`task_repository.go:1049`) | **Not reusable directly** | Epic/feature transitions go through services, not repositories. The pattern of calling `PopulateTemplate` at the service layer is different from the task implementation which calls it in the repository. |

### 4.3 Hidden Dependencies

**Hidden Dependency 1: `action_service.go` level awareness.**

The `DefaultActionService` in `internal/config/action_service.go` provides `GetStatusAction(ctx, status)` and `GetAllActions(ctx)`. These operate on a single `WorkflowConfig`. F02 either needs:
- Three `ActionService` instances (one per level), or
- A level-aware `ActionService` that wraps `MultiLevelWorkflow`

This dependency is not mentioned in the F02 PRD.

**Hidden Dependency 2: `task_repository.go` orchestrator action pattern.**

The existing task-level transition response includes `orchestrator_action` because `task_repository.go:1049` calls `PopulateTemplate` directly within the `UpdateStatusForced` flow. The F01 architecture moves transitions to the service layer for epic/feature. F02 must implement the action resolution at the service layer, NOT in the repository -- this is a different pattern from the existing task implementation.

**Hidden Dependency 3: `init update --workflow=advanced` must include actions.**

If the advanced workflow profile is updated to include `epic_workflow` and `feature_workflow` (as F01 requires), those workflow definitions should include `orchestrator_action` entries. Otherwise, the actions are only available to users who manually edit `.sharkconfig.json`. The `init update` command is in `internal/init/`.

---

## 5. Risk Assessment

### 5.1 Technical Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Template placeholder inconsistency breaks orchestrator parsing.** If `{task_id}` and `{id}` coexist without clear rules, orchestrators may use the wrong placeholder and get blank entity references. | High | High | Resolve Gap 2.1 before implementation begins. Choose Option B (support both, deprecate `{task_id}`). |
| **`show-actions` JSON breaking change for existing consumers.** Changing from flat to multi-level JSON output breaks existing automation. | High | Medium | Use Option B from Gap 2.5: add multi-level fields alongside existing flat structure. |
| **Service layer action resolution introduces latency.** Action lookup requires reading `StatusMetadata` from the workflow config for the target status. | Low | Low | NFR-001 (< 5ms) is trivially met -- this is a single map lookup. No risk. |
| **`PopulateTemplate` method signature change.** Renaming the parameter from `taskID` to `entityKey` is cosmetic, but adding `{id}` replacement changes behavior for templates that happen to contain literal `{id}` text. | Medium | Low | Add `{id}` replacement BEFORE `{task_id}` in the method. Document that `{id}` is now a reserved placeholder. |
| **Action resolution in wrong layer.** Existing task-level resolution happens in the repository (`task_repository.go`). New epic/feature resolution should happen in the service layer. Inconsistency creates confusion. | Medium | Certain | Accept the inconsistency for now. Document that task-level action resolution will be migrated to the service layer in a future refactoring (aligns with E15 goals). |

### 5.2 Integration Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **F01 `TransitionResult` struct does not include `OrchestratorAction` placeholder.** If F01 is implemented without the `omitempty` action field, F02 must modify it. | Medium | Medium | Coordinate with F01 implementation. The F01 BA review already recommends adding the field as `null`. |
| **`show-actions` and `validate-actions` command registration conflicts.** Both commands are registered under `workflowCmd`. Adding level-awareness must not break the existing registration pattern. | Low | Low | The existing registration pattern (`workflowCmd.AddCommand(...)`) handles this cleanly. |
| **Test isolation for multi-level action service.** Tests need to mock workflow configs at different levels independently. | Medium | Medium | Create `MockMultiLevelActionService` or parameterize existing mocks by level. |

### 5.3 Backward Compatibility Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Existing `.sharkconfig.json` with `{task_id}` templates continue to work.** | Blocking if broken | Certain | Ensure `PopulateTemplate` still replaces `{task_id}`. Test with real-world configs. |
| **Existing `show-actions --json` output structure preserved.** | Blocking if broken | Certain | Use Gap 2.5 Option B: additive changes only to JSON output. |
| **Existing `validate-actions --json` output structure preserved.** | Blocking if broken | Certain | Same additive approach: new fields alongside existing ones. |
| **No `epic_workflow`/`feature_workflow` in config produces sensible `show-actions` output.** | Medium | Certain | Show "using defaults, no actions configured" for missing levels. |

---

## 6. Refined Acceptance Criteria

### Existing AC Clarifications

**AC-CLARIFY-1: Scenario 1 response structure.**

Current:
> "Then JSON includes `orchestrator_action` with `action: "spawn_agent"`, `agent_type: "researcher"`, and resolved `instruction`"

Clarify to specify exact response structure:
> "Then JSON response includes a `TransitionResult` with `orchestrator_action` containing: `action: "spawn_agent"`, `agent_type: "researcher"`, `skills: ["discovery", "research"]`, and `instruction` (resolved from `instruction_template` with `E16` substituted for `{id}`). The `instruction_template` field is NOT included in the response."

**AC-CLARIFY-2: Scenario 4 null action.**

Current:
> "Then `orchestrator_action` is `null` in JSON response"

Clarify:
> "Then `orchestrator_action` is `null` (not omitted, explicitly `null`) in JSON response. This applies to both `update --status` and `next-status` commands."

### New Acceptance Criteria

**AC-ADD-1: Template variable `{id}` replacement for epics.**

Given an epic at `draft` with `ready_for_research` having `instruction_template: "Research epic {id} feasibility"`
When `shark epic next-status E16 --json` is executed
Then the `orchestrator_action.instruction` field contains "Research epic E16 feasibility"
And the `instruction_template` field is NOT present in the JSON response

**AC-ADD-2: Template variable `{id}` replacement for features.**

Given a feature at `in_refinement_ba` with `ready_for_refinement_tech` having `instruction_template: "Design architecture for {id}"`
When `shark feature next-status E16-F01 --json` is executed
Then the `orchestrator_action.instruction` field contains "Design architecture for E16-F01"

**AC-ADD-3: Backward compatibility for `{task_id}` placeholder.**

Given a task workflow `instruction_template` using `{task_id}` (existing format)
When `shark task next-status E16-F01-001 --json` is executed
Then `{task_id}` is replaced with the task key (existing behavior unchanged)

**AC-ADD-4: `show-actions` multi-level human-readable output.**

Given configured epic, feature, and task workflows with orchestrator actions
When `shark workflow show-actions` is executed
Then output displays three clearly labeled sections: "Epic Workflow Actions:", "Feature Workflow Actions:", "Task Workflow Actions:"
And each section lists status -> action type (agent type) entries
And sections with no actions display "(no actions configured)" or "(using defaults)"

**AC-ADD-5: `show-actions` multi-level JSON output backward compatible.**

Given configured epic, feature, and task workflows
When `shark workflow show-actions --json` is executed
Then JSON includes existing `workflow_actions` field with task-level actions (backward compatible)
And JSON includes new `epic_workflow_actions` field with epic-level actions
And JSON includes new `feature_workflow_actions` field with feature-level actions
And `summary` includes per-level breakdowns

**AC-ADD-6: `show-actions` with `--level` filter.**

Given configured workflows at all levels
When `shark workflow show-actions --level=epic` is executed
Then only epic workflow actions are displayed
And `--level=feature` shows only feature actions
And `--level=task` shows only task actions (equivalent to current behavior)

**AC-ADD-7: `validate-actions` multi-level output.**

Given epic actions are valid but feature actions have a missing `agent_type` on a `spawn_agent` action
When `shark workflow validate-actions` is executed
Then output shows "Epic workflow actions: VALID" and "Feature workflow actions: INVALID (1 error)"
And the overall result is INVALID
And each level's validation result is independently reported

**AC-ADD-8: `validate-actions` multi-level JSON output.**

Given the same scenario as AC-ADD-7
When `shark workflow validate-actions --json` is executed
Then JSON includes per-level validation results with independent `valid` boolean and error details
And an overall `valid` field that is `false` if any level fails

**AC-ADD-9: `update --status` response includes `orchestrator_action`.**

Given an epic at `draft` and `ready_for_research` has an orchestrator action defined
When `shark epic update E16 --status ready_for_research --json` is executed
Then JSON response includes `orchestrator_action` with resolved `instruction`
And the same behavior applies to `shark feature update`

**AC-ADD-10: `--preview` mode shows actions for each transition.**

Given an epic at `draft` with multiple valid transitions, some having `orchestrator_action` defined
When `shark epic next-status E16 --preview --json` is executed
Then each entry in `available_transitions` includes an `orchestrator_action` field (null if not defined)

**AC-ADD-11: Default workflows produce empty action sections.**

Given no `epic_workflow` or `feature_workflow` in `.sharkconfig.json`
When `shark workflow show-actions` is executed
Then the task workflow actions are displayed as before
And epic and feature sections indicate "using defaults, no actions configured"

### Validation Gates

**Gate 1**: Template placeholder decision (Gap 2.1) must be resolved before implementation.

**Gate 2**: F01 `TransitionResult` struct must include `OrchestratorAction` field (even if nil) before F02 can integrate.

**Gate 3**: All existing `show-actions` and `validate-actions` tests must continue passing after F02 changes (backward compatibility gate).

---

## 7. Implementation Sizing

### Scope Breakdown

| Component | Estimated Effort | Complexity |
|-----------|-----------------|------------|
| Extend `PopulateTemplate` to support `{id}` alongside `{task_id}` | XS | Low -- single method change plus tests |
| Update `validateTemplateSyntax` known placeholders | XS | Low -- add `{id}` to the map |
| Add `OrchestratorAction` field to `TransitionResult` (F01 struct) | XS | Low -- one field addition |
| Resolve and populate `orchestrator_action` in `EpicService.TransitionStatus` | S | Medium -- service layer integration |
| Resolve and populate `orchestrator_action` in `FeatureService.TransitionStatus` | S | Medium -- same pattern as epic |
| Resolve `orchestrator_action` in `next-status` command output for epic/feature | S | Medium -- output formatting |
| Extend `show-actions` for multi-level (human-readable + JSON) | M | Medium -- refactor existing display logic, add backward-compatible JSON |
| Extend `validate-actions` for multi-level (human-readable + JSON) | S | Medium -- call existing validator per level, aggregate results |
| Add `--level` flag to `show-actions` and `validate-actions` | XS | Low -- flag parsing and conditional logic |
| Extend `DefaultActionService` for level awareness (or create per-level instances) | S | Medium -- depends on F01's `ForLevel` pattern |
| Update advanced workflow profile with epic/feature actions | S | Low -- config-only, no code logic |
| Tests: service-layer action resolution (mocked repos) | M | Medium -- multiple test cases per entity level |
| Tests: `show-actions` multi-level output (mocked config) | S | Low -- extend existing test patterns |
| Tests: `validate-actions` multi-level output (mocked config) | S | Low -- extend existing test patterns |
| Tests: `PopulateTemplate` with `{id}` and backward compat | XS | Low -- extend existing test cases |

### Overall Estimate: **M** (Medium)

**Justification**: The majority of the infrastructure already exists from E13 (task-level orchestrator actions). The core work is:
1. Template placeholder extension (XS)
2. Service-layer action resolution for two new entity types (S + S)
3. Two CLI commands extended for multi-level display (M + S)

No new architectural patterns are introduced -- F02 extends existing patterns to new entity levels. The complexity is primarily in maintaining backward compatibility for JSON output structures and the template placeholder naming.

**Estimated implementation tasks**: 5-7 tasks, following the pattern from E13.

---

## 8. Recommendation

### Verdict: Ready for Technical Architecture -- with the following conditions:

**Condition 1 (BLOCKING): Resolve template placeholder naming.**

The `{task_id}` vs `{id}` inconsistency must be decided before technical architecture begins. The BA recommends Option B from Gap 2.1: use `{id}` as canonical, support `{task_id}` with deprecation warning. The decision must be documented in the feature PRD and the technical architecture must implement the chosen approach.

**Condition 2 (BLOCKING): Confirm F01 `TransitionResult` includes `OrchestratorAction` field.**

The F01 architecture document (`architecture.md`) defines `TransitionResult` without an `OrchestratorAction` field. F02 requires this field. Either:
- F01 adds it as `OrchestratorAction *config.OrchestratorAction \`json:"orchestrator_action,omitempty"\`` (set to nil), or
- F02 modifies the struct (creates merge dependency on F01 completion)

The F01 BA review already recommends the former approach (Finding 2.2). Confirm this is accepted.

**Condition 3 (RECOMMENDED): Define JSON output structures.**

The technical architecture should include exact JSON response structures for:
- Multi-level `show-actions` output (Gap 2.3)
- Multi-level `validate-actions` output (Gap 2.6)
- `TransitionResult` with populated `orchestrator_action` (Gap 2.2)
- `--preview` mode with per-transition actions (EC-4)

**Condition 4 (RECOMMENDED): Define backward compatibility strategy for `show-actions` JSON.**

Choose between additive (Option B, recommended) or breaking change (Option A) for the `show-actions --json` output structure. Document the decision.

### Pre-requisites That Must Be Resolved First

1. **F01 Core Workflow Engine must be complete** (at minimum: config parsing, `ForLevel()`, service layer, and `TransitionResult` struct).
2. **Template placeholder decision** agreed upon by architect and BA.
3. **JSON backward compatibility decision** for `show-actions` agreed upon.

---

## Appendix A: File Impact Map

Files that will require modification or creation for F02:

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/config/orchestrator_action.go` | Modify | Extend `PopulateTemplate` to replace `{id}` in addition to `{task_id}`. Update parameter name from `taskID` to `entityKey`. |
| `internal/config/orchestrator_action.go` | Modify | Update `validateTemplateSyntax` to include `{id}` in known placeholders. |
| `internal/services/transition_types.go` | Modify | Add `OrchestratorAction *config.OrchestratorAction` to `TransitionResult`. |
| `internal/services/epic_service.go` | Modify | Populate `OrchestratorAction` in `TransitionStatus` response by looking up target status metadata. |
| `internal/services/feature_service.go` | Modify | Same as above for features. |
| `internal/cli/commands/workflow_show_actions.go` | Modify | Load all three workflow configs. Add multi-level display. Add `--level` flag. Maintain backward-compatible JSON. |
| `internal/cli/commands/workflow_validate_actions.go` | Modify | Validate all three levels. Add per-level reporting. Add `--level` flag. |
| `internal/cli/commands/epic_next_status.go` | Modify | Include `orchestrator_action` in output when present. |
| `internal/cli/commands/feature_next_status.go` | Modify | Same as above. |
| `internal/config/action_service.go` | Potentially modify | May need level awareness or separate instances per level. |

### New Test Files

| File | Tests |
|------|-------|
| `internal/config/orchestrator_action_test.go` | Add: `PopulateTemplate` with `{id}`, backward compat with `{task_id}`, both placeholders |
| `internal/config/orchestrator_action_validation_test.go` | Add: validation with `{id}` placeholder |
| `internal/services/epic_service_test.go` | Add: transition with action populated, transition without action (null) |
| `internal/services/feature_service_test.go` | Add: same as epic |
| `internal/cli/commands/workflow_show_actions_test.go` | Add: multi-level display, `--level` flag, backward-compatible JSON |
| `internal/cli/commands/workflow_validate_actions_test.go` | Add: multi-level validation, per-level reporting |

## Appendix B: Template Placeholder Decision Matrix

| Placeholder | Currently Supported | Proposed | Entity Level | Notes |
|-------------|--------------------|----------|-------------|-------|
| `{task_id}` | Yes (E13) | Deprecated but supported | Task | Existing configs use this |
| `{id}` | No | **New canonical** | All | Generic, entity-type-agnostic |
| `{epic_id}` | No | Not proposed | -- | Rejected: too level-specific |
| `{feature_id}` | No | Not proposed | -- | Rejected: too level-specific |
| `{entity_type}` | No | Future consideration | All | Could be useful for generic instructions |
| `{entity_key}` | No | Not proposed | -- | Rejected: `{id}` is shorter and sufficient |

## Appendix C: Existing Task-Level Action Flow (for reference)

```
task_next_status.go
  |
  +-> taskRepo.GetByKey(ctx, taskKey) -> task
  +-> workflowSvc.GetTransitionInfo(currentStatus) -> transitions
  +-> [select target status]
  +-> taskRepo.UpdateStatusForcedWithUnblock(...)
  |     |
  |     +-> task_repository.go:1049
  |           metadata.OrchestratorAction.PopulateTemplate(task.Key)
  |           -> adds resolved action to response
  |
  +-> cli.OutputJSON(result)  [result includes orchestrator_action]
```

**Note**: The action resolution currently happens IN THE REPOSITORY for tasks. For epic/feature (per F01 architecture), transitions happen through the SERVICE LAYER. F02 must implement action resolution at the service layer, following the cleaner architecture pattern. This creates a deliberate inconsistency with the task implementation that should be documented.

## Appendix D: Glossary

- **Orchestrator action**: A config-driven instruction telling an orchestrator what agent to spawn or what action to take when an entity enters a given status.
- **Template variable**: A placeholder in `instruction_template` (like `{id}` or `{task_id}`) that is replaced with the actual entity key at runtime.
- **Action resolution**: The process of looking up the `orchestrator_action` for a target status and populating its template with the entity key.
- **Level-aware**: A component that can operate on epic, feature, or task workflows independently based on a `level` parameter.
