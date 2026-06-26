# Test Plan: E19-F08 — Sprint Workflow YAML First-Class Lifecycle

**Feature:** Sprint Workflow YAML — first-class lifecycle with embedded agent routing  
**Complexity:** STANDARD  
**Date:** 2026-06-26

---

## 1. AC Test Matrix

### AC-1: `shark status advance <sprint-key>` from `planning` triggers sprint-planning skill

| Test | Input | Expected | Edge Case |
|------|-------|----------|-----------|
| AC-1a (unit) | Sprint in `planning` status; mock workflow returns `pass→active` route with `action: spawn_agent`, `skills: [sprint-planning]` | `GetValidTransitions` returns `active`; step metadata has `agent: sprint-planning`, `skills: [sprint-planning]` | Workflow YAML missing sprint-planning skill → graceful error, no panic |
| AC-1b (YAML load) | Load embedded `sprint.yaml`; call `GetStatusMetadata("planning")` | `action == "spawn_agent"`, `skills` contains `"sprint-planning"`, `outcomes["pass"] == "active"` | — |

### AC-2: `shark status advance <sprint-key>` from `active` triggers sprint-execution skill

| Test | Input | Expected | Edge Case |
|------|-------|----------|-----------|
| AC-2a (unit) | Sprint in `active` status; mock workflow | Step metadata has `agent: sprint-execution`, `skills: [sprint-execution]`, `outcomes["pass"] == "closing"` | — |

### AC-3: `shark status advance <sprint-key>` from `closing` triggers sprint-analytics skill

| Test | Input | Expected | Edge Case |
|------|-------|----------|-----------|
| AC-3a (unit) | Sprint in `closing` status; mock workflow | Step metadata has `agent: sprint-analytics`, `skills: [sprint-analytics]`, `outcomes["pass"] == "archived"` | — |

### AC-4: Embedded workflow test passes for sprint level

| Test | Input | Expected | Edge Case |
|------|-------|----------|-----------|
| AC-4a | Load embedded default data; check sprint slot | `sprint` key present; status set equals `{planning, active, closing, archived, on_hold}` | Loader returns empty sprint slot → test fails (must not skip) |
| AC-4b | Check start status | `GetInitialStatusString()` returns `"planning"` | — |
| AC-4c | Check terminal | `GetTerminalStatuses()` returns `["archived"]` | — |

### AC-5: `make test` passes with no regressions

| Test | Input | Expected |
|------|-------|----------|
| AC-5a | `make test` on full suite | 0 failures; no compilation errors |

---

## 2. Service Tests — Caller-Path Contracts

Each test lives in `internal/services/sprint_service_test.go` and uses the existing `MockWorkflowService` + mock sprint repo pattern. **No real DB.**

### T-01: `DeleteSprint()` — initial-status guard

- **Production path:** `DeleteSprint()` → `s.workflowSvc.GetInitialStatusString()`
- **Mock seam:** `MockWorkflowService.GetInitialStatusString` returns `"planning"`
- **Forbidden mocks:** real DB, real workflow.Service
- **Test:** Sprint.Status = `"planning"` → error "cannot delete sprint that has not started"; Sprint.Status = `"active"` → delete proceeds
- **Counter-factual:** If hardcoded `"planning"` remains, a custom workflow with `start: queued` allows deleting active sprints silently

### T-02: `assignableSprintStatuses()` — phase-driven

- **Production path:** `assignableSprintStatuses()` → `GetStatusesByPhase("planning")` + `GetStatusesByPhase("execution")`
- **Mock seam:** `MockWorkflowService.GetStatusesByPhase("planning")` returns `["planning"]`; `("execution")` returns `["active"]`
- **Test:** Result equals `["planning", "active"]` with no duplicates; order stable
- **Counter-factual:** If old `assignableSprintPhases()` remains, `assignableSprintStatuses()` returns hardcoded strings regardless of workflow config

### T-03: `GetSprintBacklog()` — execution-phase check

- **Production path:** `GetSprintBacklog()` → `GetStatusesByPhase("execution")` contains check
- **Mock seam:** `GetStatusesByPhase("execution")` returns `["active"]`
- **Test:** Sprint.Status = `"active"` → backlog returned; Sprint.Status = `"closing"` → error or filtered-mode only
- **Counter-factual:** If `== "active"` literal remains, a renamed execution status (e.g. `"in-flight"`) never returns a backlog

### T-04: `StartSprint()` — single-active constraint deleted

- **Production path:** `StartSprint()` → `ValidateTransition("planning","active")` only; no `ListSprints{Status:"active"}` call
- **Mock seam:** `MockSprintRepo.ListSprints` is NOT called (assert call count = 0); `ValidateTransition` returns nil
- **Tests:**
  - Two sprints both set to active → **both succeed** (no error from single-active check)
  - `ValidateTransition` failure → `StartSprint` still returns error (transition validation kept)
- **Counter-factual:** If block remains, starting a second sprint returns "already have an active sprint" error

### T-05: `ReorderAssignment()` — uses `assignableSprintStatuses()`

- **Production path:** `ReorderAssignment()` → `!s.sprintAcceptsAssignments(status)` → `assignableSprintStatuses()`
- **Mock seam:** `GetStatusesByPhase` returns appropriate slices
- **Tests:** Sprint in `"planning"` → reorder allowed; Sprint in `"archived"` → reorder rejected with meaningful error
- **Counter-factual:** Hardcoded `!= "planning" && != "active"` misses custom status names

### T-06: `CloseSprintWithCarryover()` — planning filter and target

- **Production path:** planning filter → `GetStatusesByPhase("planning")[0]`; target status → `GetInitialStatusString()`
- **Mock seam:** Both return `"planning"`
- **Tests:** Carryover tasks assigned to planning-phase sprint; new sprint created with initial status; empty `GetStatusesByPhase("planning")` → returns descriptive error (no panic on `[0]`)
- **Counter-factual:** Hardcoded `"planning"` continues working for default workflow but silently breaks custom workflows

### T-07: `GetNextTask()` — multi-active sprint selection

- **Production path:** `GetNextTask()` → `GetStatusesByPhase("execution")[0]` filter → `ListSprints` → pick first result
- **Mock seam:** `GetStatusesByPhase("execution")` returns `["active"]`; `MockSprintRepo.ListSprints` returns 2 sprints with Status=`"active"` ordered by ID
- **Tests:**
  - Single active sprint → returns task from that sprint (backward-compat)
  - Two active sprints → returns task from the **first** sprint by ordering (deterministic first-match)
  - Empty execution phase slice → returns descriptive error, no panic
- **Counter-factual:** If original `Status:"active"` literal remains with a custom execution-phase name, `GetNextTask()` returns no sprints

---

## 3. Integration Scenarios

### I-01: Full sprint state machine (YAML + service)

Load real `sprint.yaml` from embedded data → create `workflow.Service` for sprint level → verify:
- `GetInitialStatusString()` = `"planning"`
- `IsValidTransition("planning", "active")` = true
- `IsValidTransition("planning", "archived")` = false
- `IsValidTransition("active", "closing")` = true
- `IsValidTransition("closing", "archived")` = true
- `IsTerminalStatus("archived")` = true
- `GetStatusesByPhase("execution")` contains `"active"`
- `GetStatusesByPhase("planning")` contains `"planning"`

**Location:** Add as `TestSprintWorkflowStateMachine` in `internal/config/cc040_embedded_workflow_test.go`

### I-02: Parking and terminal states

- `on_hold` has `parking: true` → `GetStatusMetadata("on_hold").Parking` = true
- `archived` has `terminal: true` → `IsTerminalStatus("archived")` = true
- Every workable step (`planning`, `active`, `closing`) defines `blocked` → `on_hold` outcome

### I-03: `assignableSprintStatuses()` dedup

Mock `GetStatusesByPhase` to return overlapping lists → result has no duplicates (existing dedup behavior preserved post-refactor).

---

## 4. Embedded Workflow Test Updates

**File:** `internal/config/cc040_embedded_workflow_test.go`

### Changes required

1. **Remove the sprint skip comment** at lines 176-178 / 388-389 ("sprint.yaml intentionally omitted").

2. **Add `TestDefaultWorkflowDataLoader_SprintWorkflowLoads`:**
```go
func TestDefaultWorkflowDataLoader_SprintWorkflowLoads(t *testing.T) {
    loader := NewDefaultWorkflowDataLoader()
    data, err := loader.Load()
    require.NoError(t, err)

    sprintWf, ok := data["sprint"]
    require.True(t, ok, "sprint workflow must be present in default data")

    statuses := sprintWf.GetAllStatuses()
    expected := []string{"planning", "active", "closing", "archived", "on_hold"}
    assert.ElementsMatch(t, expected, statuses)

    assert.Equal(t, "planning", sprintWf.StartStatus)

    terminals := sprintWf.GetTerminalStatuses()
    assert.Equal(t, []string{"archived"}, terminals)
}
```

3. **Extend `TestDefaultWorkflowDataLoader_Pass3_AllEmbeddedEntitiesLoaded`** to include `"sprint"` in the expected entity set (remove from skip/omit list).

---

## 5. Test Infrastructure

### Existing — reuse as-is

| File | What to reuse |
|------|---------------|
| `internal/services/sprint_service_test.go` | `MockWorkflowService` function-field pattern; mock sprint repo |
| `internal/config/cc040_embedded_workflow_test.go` | Loader harness; existing `TestDefaultWorkflowDataLoader_*` helpers |

### New test functions needed

| Function | File | Priority |
|----------|------|----------|
| `TestDeleteSprint_InitialStatusFromWorkflow` | `sprint_service_test.go` | High |
| `TestAssignableSprintStatuses_Phasedriven` | `sprint_service_test.go` | High |
| `TestGetSprintBacklog_ExecutionPhase` | `sprint_service_test.go` | High |
| `TestStartSprint_MultipleActiveAllowed` | `sprint_service_test.go` | High |
| `TestReorderAssignment_UsesAssignableStatuses` | `sprint_service_test.go` | Medium |
| `TestCloseSprintWithCarryover_WorkflowStatusLookup` | `sprint_service_test.go` | High |
| `TestCloseSprintWithCarryover_EmptyPlanningPhase_Error` | `sprint_service_test.go` | Medium |
| `TestGetNextTask_SingleActiveSprint` | `sprint_service_test.go` | High |
| `TestGetNextTask_MultipleActiveSprints_DeterministicFirst` | `sprint_service_test.go` | High |
| `TestGetNextTask_EmptyExecutionPhase_Error` | `sprint_service_test.go` | Medium |
| `TestSprintWorkflowStateMachine` | `cc040_embedded_workflow_test.go` | High |
| `TestDefaultWorkflowDataLoader_SprintWorkflowLoads` | `cc040_embedded_workflow_test.go` | High |

### Deleted tests

- Any test asserting `StartSprint` returns "already have an active sprint" / single-active error → delete or invert to assert **no error** when two sprints active

---

## 6. Validation Checklist

- [ ] `internal/sharkdata/default_data/workflow/sprint.yaml` created with 5 steps
- [ ] 3 prompt stubs created under `internal/sharkdata/default_data/prompts/sprint/`
- [ ] `assignableSprintPhases()` function is gone (no callers, no definition)
- [ ] `StartSprint()` has no `ListSprints{Status:"active"}` call or single-active error
- [ ] All 8 hardcoded string replacements applied (see spec mapping table)
- [ ] Empty-slice guards on `GetStatusesByPhase(...)[0]` calls
- [ ] `cc040_embedded_workflow_test.go` no longer skips sprint
- [ ] `make fmt && make lint && make test` — green
