# Test Plan: E17-F07 Status Subcommand Group

**Feature**: E17-F07 Status Subcommand Group
**Author**: QA Agent
**Date**: 2026-02-25
**Status**: Ready for Review

---

## Table of Contents

1. [Scope and Objectives](#1-scope-and-objectives)
2. [Test Architecture and Approach](#2-test-architecture-and-approach)
3. [Unit Tests: Handler Functions with Mocked Services](#3-unit-tests-handler-functions-with-mocked-services)
4. [Integration Tests: Entity Auto-Detection and Workflow](#4-integration-tests-entity-auto-detection-and-workflow)
5. [Backward Compatibility Tests](#5-backward-compatibility-tests)
6. [Edge Case Tests](#6-edge-case-tests)
7. [UAT Scenarios from Epic Plan](#7-uat-scenarios-from-epic-plan)
8. [Exit Criteria and Quality Gates](#8-exit-criteria-and-quality-gates)
9. [Test Execution Order](#9-test-execution-order)

---

## 1. Scope and Objectives

### What Is Being Tested

Feature E17-F07 introduces `status_group.go` into `internal/cli/commands/`, adding four subcommands to the existing `statusCmd`:

- `shark status set <key> <status>` -- idempotent status setter
- `shark status advance <key>` -- advance to next workflow status
- `shark status options <key>` -- read-only: show current status and valid transitions
- `shark status history <key>` -- show status change log (tasks only)

### Out of Scope

- Service layer internals (tested in service unit tests, not CLI tests)
- Database persistence correctness (tested in repository tests)
- `shark status <id>` dashboard behavior (existing `statusCmd.RunE` -- zero changes to this path)
- Phase 2 batch mode (`shark status set --feature ...`) -- not part of this feature
- `history.go` hidden alias -- tracked as a separate follow-on task

### Test Objectives

1. Each handler (`runStatusSet`, `runStatusAdvance`, `runStatusOptions`, `runStatusHistory`) behaves correctly for all input scenarios
2. Entity type is auto-detected correctly from key format for all four handlers
3. Cobra command routing coexists with the existing `statusCmd.RunE` -- `shark status E07` still routes to the dashboard
4. Idempotency contract: `status set` with the current status returns exit 0 with `"changed": false`
5. All existing commands that were working before this feature continue to work identically
6. Edge cases produce correct exit codes, error messages, and JSON structures

---

## 2. Test Architecture and Approach

### Test File Location

```
internal/cli/commands/status_group_test.go
```

### Testing Strategy

CLI command tests MUST use mocked services. No real database. No real workflow service. All tests inject mocks via the function-field mock pattern established in the codebase.

**The Three-Step Test Pattern:**
```
1. Arrange: Create mock services with controlled return values
2. Act: Invoke the handler function directly (or via cobra Execute)
3. Assert: Verify output, exit codes, mock call parameters
```

### Mock Structure

The test file defines mock implementations of the service interfaces:

```go
// MockTaskServiceForStatus covers TransitionStatus, GetNextStatus, GetTaskHistory
type MockTaskServiceForStatus struct {
    TransitionStatusFunc func(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatusFunc    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
    GetTaskHistoryFunc   func(ctx context.Context, key string) ([]*models.TaskHistory, error)
}

// MockFeatureServiceForStatus covers TransitionStatus, GetNextStatus
type MockFeatureServiceForStatus struct {
    TransitionStatusFunc func(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatusFunc    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

// MockEpicServiceForStatus covers TransitionStatus, GetNextStatus
type MockEpicServiceForStatus struct {
    TransitionStatusFunc func(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatusFunc    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}
```

### Test Data Fixtures

Standard task keys used throughout:
- Task key: `E07-F01-001` (short format)
- Task key with prefix: `T-E07-F01-001` (traditional format)
- Feature key: `E07-F01`
- Epic key: `E07`
- Non-existent key: `E99-F99-999`

Standard status values:
- Source status: `todo`
- Target status: `in_development`
- Terminal status: `completed` (for terminal-state tests)

---

## 3. Unit Tests: Handler Functions with Mocked Services

### 3.1 `runStatusSet` Handler Tests

**Test file section**: `TestRunStatusSet`

---

#### TC-SET-01: Happy path -- task transition succeeds

**Setup:**
- `dispatchTransition` returns `&TransitionResult{Transitioned: true, EntityType: "task", EntityKey: "E07-F01-001", FromStatus: "todo", ToStatus: "in_development", IsBackward: false, IsForced: false}`

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01-001", "in_development"]`, no flags
2. Capture stdout output

**Expected:**
- No error returned
- Human-readable success message contains `E07-F01-001`, `todo`, `in_development`
- Exit code: 0 (handler returns nil)

**Assertions:**
```go
assert.NoError(t, err)
assert.Contains(t, output, "E07-F01-001")
assert.Contains(t, output, "todo")
assert.Contains(t, output, "in_development")
```

---

#### TC-SET-02: Happy path -- task transition succeeds with `--json`

**Setup:**
- Same mock as TC-SET-01

**Test steps:**
1. Set `cli.GlobalConfig.JSON = true`
2. Invoke `runStatusSet` with args `["E07-F01-001", "in_development"]`

**Expected:**
- Output is valid JSON
- JSON contains `"changed": true`
- JSON contains `"entity_type": "task"`
- JSON contains `"from_status": "todo"`
- JSON contains `"to_status": "in_development"`

**Assertions:**
```go
assert.NoError(t, err)
var result map[string]interface{}
assert.NoError(t, json.Unmarshal([]byte(output), &result))
assert.Equal(t, true, result["changed"])
assert.Equal(t, "task", result["entity_type"])
assert.Equal(t, "todo", result["from_status"])
assert.Equal(t, "in_development", result["to_status"])
```

---

#### TC-SET-03: Idempotent -- entity already at target status (human output)

**Setup:**
- `dispatchTransition` returns `&TransitionResult{Transitioned: false, EntityType: "task", EntityKey: "E07-F01-001", ToStatus: "in_development"}`

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01-001", "in_development"]`

**Expected:**
- No error returned (exit 0)
- Human output contains "already in status" or equivalent no-op message
- Does NOT print a "success" transition message

**Assertions:**
```go
assert.NoError(t, err)
assert.Contains(t, output, "already")
assert.Contains(t, output, "in_development")
```

---

#### TC-SET-04: Idempotent -- entity already at target status (`--json`)

**Setup:**
- Same mock as TC-SET-03

**Test steps:**
1. Set `cli.GlobalConfig.JSON = true`
2. Invoke `runStatusSet` with args `["E07-F01-001", "in_development"]`

**Expected:**
- No error returned (exit 0)
- JSON output contains `"changed": false`
- JSON output contains `"status": "in_development"`
- JSON output does NOT contain `"from_status"` or `"to_status"` (those are for actual transitions)

**Assertions:**
```go
assert.NoError(t, err)
var result map[string]interface{}
assert.NoError(t, json.Unmarshal([]byte(output), &result))
assert.Equal(t, false, result["changed"])
assert.Equal(t, "in_development", result["status"])
```

---

#### TC-SET-05: Entity not found -- exits with code 1

**Setup:**
- `dispatchTransition` returns error containing "not found"

**Test steps:**
1. Invoke `runStatusSet` with args `["E99-F99-999", "in_development"]`

**Expected:**
- `os.Exit(1)` is called (or error returned with exit-1 sentinel)
- Error message on stderr contains "not found" and the key `E99-F99-999`

**Assertions:**
```go
// Use os.Exit interceptor or verify os.Exit(1) call
assert.Contains(t, stderrOutput, "E99-F99-999")
assert.Contains(t, stderrOutput, "not found")
```

---

#### TC-SET-06: Invalid transition -- exits with code 3

**Setup:**
- `dispatchTransition` returns error containing "invalid transition"

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01-001", "completed"]`

**Expected:**
- `os.Exit(3)` is called
- Error message contains "Invalid transition"
- Error message contains hint to use `shark status options`

**Assertions:**
```go
assert.Contains(t, stderrOutput, "Invalid transition")
assert.Contains(t, stderrOutput, "shark status options")
assert.Contains(t, stderrOutput, "E07-F01-001")
```

---

#### TC-SET-07: Backward transition without `--reason` -- exits with code 3

**Setup:**
- `dispatchTransition` returns `services.ErrReasonRequired`

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01-001", "todo"]` (backward), no `--reason` flag

**Expected:**
- `os.Exit(3)` is called
- Error message tells user to provide `--reason`

**Assertions:**
```go
assert.Contains(t, stderrOutput, "--reason")
```

---

#### TC-SET-08: `--force` flag bypasses validation with `--reason`

**Setup:**
- `dispatchTransition` returns `&TransitionResult{Transitioned: true, IsForced: true, ...}`
- Verify `opts.Force = true` was passed to the mock

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01-001", "todo"]`, flags `--force --reason="Admin override"`

**Expected:**
- No error returned
- Human output contains "Workflow validation was bypassed" warning
- JSON output contains `"is_forced": true`

**Assertions:**
```go
assert.NoError(t, err)
assert.True(t, capturedOpts.Force)
assert.Equal(t, "Admin override", capturedOpts.Reason)
// Human mode:
assert.Contains(t, output, "bypassed")
// JSON mode:
assert.Equal(t, true, result["is_forced"])
```

---

#### TC-SET-09: `--notes` and `--agent` flags are passed through to service

**Setup:**
- Capture the `TransitionOptions` passed to `dispatchTransition`

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01-001", "in_development"]`, flags `--notes="Starting" --agent="agent-1"`

**Expected:**
- `capturedOpts.Notes` equals `"Starting"`
- `capturedOpts.Agent` equals `"agent-1"`

**Assertions:**
```go
assert.Equal(t, "Starting", capturedOpts.Notes)
assert.Equal(t, "agent-1", capturedOpts.Agent)
```

---

#### TC-SET-10: Feature entity -- routes to FeatureService

**Setup:**
- `dispatchTransition` for entity type "feature" returns success result

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01", "active"]`

**Expected:**
- `FeatureService.TransitionStatus()` was called (not TaskService)
- JSON result contains `"entity_type": "feature"`

---

#### TC-SET-11: Epic entity -- routes to EpicService

**Setup:**
- `dispatchTransition` for entity type "epic" returns success result

**Test steps:**
1. Invoke `runStatusSet` with args `["E07", "active"]`

**Expected:**
- `EpicService.TransitionStatus()` was called
- JSON result contains `"entity_type": "epic"`

---

#### TC-SET-12: Backward transition with reason recorded in `is_backward` JSON field

**Setup:**
- `dispatchTransition` returns `&TransitionResult{Transitioned: true, IsBackward: true, Reason: "Redo needed", ...}`

**Test steps:**
1. Set `cli.GlobalConfig.JSON = true`
2. Invoke `runStatusSet` with args `["E07-F01-001", "todo"]`, flag `--reason="Redo needed"`

**Expected:**
- JSON contains `"is_backward": true`
- JSON contains `"reason": "Redo needed"`

---

#### TC-SET-13: `child_count` warning appears in human output when child entities remain

**Setup:**
- `dispatchTransition` returns `&TransitionResult{Transitioned: true, ChildCount: 3, ...}`

**Test steps:**
1. Invoke `runStatusSet` with args `["E07-F01", "completed"]`

**Expected:**
- Human output contains warning about "3 child entities"

**Assertions:**
```go
assert.Contains(t, output, "3 child entities")
```

---

### 3.2 `runStatusAdvance` Handler Tests

**Test file section**: `TestRunStatusAdvance`

---

#### TC-ADV-01: Auto-advance -- single valid transition

**Setup:**
- `dispatchNextStatus` returns `&NextStatusInfo{CurrentStatus: "todo", AvailableTransitions: [{TargetStatus: "in_development"}], IsTerminal: false}`
- `dispatchTransition` returns successful `TransitionResult`

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`

**Expected:**
- No error returned
- Transition is performed to `in_development`
- Success message contains `E07-F01-001` and `in_development`

---

#### TC-ADV-02: Auto-advance -- multiple transitions, uses first (primary)

**Setup:**
- `dispatchNextStatus` returns two transitions: `[{TargetStatus: "in_development"}, {TargetStatus: "blocked"}]`
- `dispatchTransition` returns success for `in_development`

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`

**Expected:**
- Transition is performed to `in_development` (first in list)
- `blocked` is NOT chosen

---

#### TC-ADV-03: `--to` flag selects specific next status from valid options

**Setup:**
- `dispatchNextStatus` returns two valid transitions: `in_development`, `blocked`
- `dispatchTransition` returns success for `blocked`

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`, flag `--to=blocked`

**Expected:**
- Transition is performed to `blocked` (as specified by `--to`)

---

#### TC-ADV-04: `--to` flag with invalid target -- exits with code 3

**Setup:**
- `dispatchNextStatus` returns transitions: `[{TargetStatus: "in_development"}]`

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`, flag `--to=nonexistent_status`

**Expected:**
- `os.Exit(3)` is called
- Error message contains `'nonexistent_status' is not a valid transition`
- Error message lists valid transitions
- Error message suggests `--force` to bypass

**Assertions:**
```go
assert.Contains(t, stderrOutput, "nonexistent_status")
assert.Contains(t, stderrOutput, "in_development")
assert.Contains(t, stderrOutput, "--force")
```

---

#### TC-ADV-05: Terminal status -- returns exit 0 with warning, no transition

**Setup:**
- `dispatchNextStatus` returns `&NextStatusInfo{CurrentStatus: "completed", IsTerminal: true, AvailableTransitions: []}`

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`

**Expected:**
- No error returned (exit 0)
- No transition is attempted (`dispatchTransition` is NOT called)
- Warning message contains "terminal status" and "no transitions"

**Assertions:**
```go
assert.NoError(t, err)
assert.False(t, transitionCalled) // verify mock was not called
assert.Contains(t, output, "terminal")
```

---

#### TC-ADV-06: No available transitions -- returns exit 0 with warning

**Setup:**
- `dispatchNextStatus` returns `&NextStatusInfo{CurrentStatus: "on_hold", IsTerminal: false, AvailableTransitions: []}`

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`

**Expected:**
- No error returned (exit 0)
- No transition is attempted
- Warning message contains "No valid transitions"

---

#### TC-ADV-07: JSON output for successful advance

**Setup:**
- Same as TC-ADV-01 but with `cli.GlobalConfig.JSON = true`

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`

**Expected:**
- Output is valid JSON
- JSON reflects the transition result from `performEntityTransition`

---

#### TC-ADV-08: `--to` flag with `--force` bypasses validation

**Setup:**
- `dispatchNextStatus` returns empty `AvailableTransitions`
- `dispatchTransition` returns success (forced)

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01-001"]`, flags `--to=completed --force --reason="Admin"`

**Expected:**
- Transition is performed (force bypasses validation of `--to` against available transitions)
- `opts.Force = true` was passed to service

---

#### TC-ADV-09: Feature entity -- routes to FeatureService.GetNextStatus

**Setup:**
- `dispatchNextStatus` for "feature" returns transitions
- `dispatchTransition` for "feature" returns success

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E07-F01"]`

**Expected:**
- `FeatureService.GetNextStatus()` was called
- `FeatureService.TransitionStatus()` was called

---

#### TC-ADV-10: Entity not found in advance -- exits with code 1

**Setup:**
- `dispatchNextStatus` returns error containing "not found"

**Test steps:**
1. Invoke `runStatusAdvance` with args `["E99-F99-999"]`

**Expected:**
- `os.Exit(1)` is called
- Error message contains the key `E99-F99-999`

---

### 3.3 `runStatusOptions` Handler Tests

**Test file section**: `TestRunStatusOptions`

---

#### TC-OPT-01: Task with multiple transitions -- human output

**Setup:**
- `dispatchNextStatus` returns `&NextStatusInfo{CurrentStatus: "in_development", CurrentPhase: "development", IsTerminal: false, AvailableTransitions: [{TargetStatus: "ready_for_code_review"}, {TargetStatus: "blocked"}]}`

**Test steps:**
1. Invoke `runStatusOptions` with args `["E07-F01-001"]`

**Expected:**
- No error returned
- Output contains `in_development`
- Output contains `development` (phase)
- Output contains `ready_for_code_review`
- Output contains `blocked`
- Output contains usage hint `shark status advance E07-F01-001`
- Output contains usage hint `shark status set E07-F01-001`

**Assertions:**
```go
assert.NoError(t, err)
assert.Contains(t, output, "in_development")
assert.Contains(t, output, "development")
assert.Contains(t, output, "ready_for_code_review")
assert.Contains(t, output, "blocked")
assert.Contains(t, output, "shark status advance")
assert.Contains(t, output, "shark status set")
```

---

#### TC-OPT-02: Task with multiple transitions -- JSON output

**Setup:**
- Same as TC-OPT-01

**Test steps:**
1. Set `cli.GlobalConfig.JSON = true`
2. Invoke `runStatusOptions` with args `["E07-F01-001"]`

**Expected:**
- Valid JSON output
- JSON contains `"entity_type": "task"`
- JSON contains `"entity_key": "E07-F01-001"`
- JSON contains `"current_status": "in_development"`
- JSON contains `"current_phase": "development"`
- JSON contains `"is_terminal": false`
- JSON contains `"available_transitions"` array with at least 2 items

**Assertions:**
```go
var result map[string]interface{}
assert.NoError(t, json.Unmarshal([]byte(output), &result))
assert.Equal(t, "task", result["entity_type"])
assert.Equal(t, "E07-F01-001", result["entity_key"])
assert.Equal(t, "in_development", result["current_status"])
assert.Equal(t, false, result["is_terminal"])
transitions := result["available_transitions"].([]interface{})
assert.GreaterOrEqual(t, len(transitions), 2)
```

---

#### TC-OPT-03: Terminal status -- shows no transitions message

**Setup:**
- `dispatchNextStatus` returns `&NextStatusInfo{CurrentStatus: "completed", IsTerminal: true, AvailableTransitions: []}`

**Test steps:**
1. Invoke `runStatusOptions` with args `["E07-F01-001"]`

**Expected:**
- No error returned (exit 0)
- Output indicates "terminal status" with no further transitions

**Assertions:**
```go
assert.NoError(t, err)
assert.Contains(t, output, "terminal")
```

---

#### TC-OPT-04: Terminal status -- JSON output

**Setup:**
- Same as TC-OPT-03, with `cli.GlobalConfig.JSON = true`

**Expected:**
- JSON contains `"is_terminal": true`
- JSON contains `"available_transitions": []` (empty array)

---

#### TC-OPT-05: No transitions available (non-terminal) -- warning

**Setup:**
- `dispatchNextStatus` returns `&NextStatusInfo{CurrentStatus: "on_hold", IsTerminal: false, AvailableTransitions: []}`

**Test steps:**
1. Invoke `runStatusOptions` with args `["E07-F01-001"]`

**Expected:**
- No error returned
- Warning output contains "No valid transitions"

---

#### TC-OPT-06: Feature entity -- routes to FeatureService

**Setup:**
- `dispatchNextStatus` for "feature" returns valid info

**Test steps:**
1. Invoke `runStatusOptions` with args `["E07-F01"]`

**Expected:**
- `FeatureService.GetNextStatus()` was called
- JSON result contains `"entity_type": "feature"`

---

#### TC-OPT-07: Epic entity -- routes to EpicService

**Test steps:**
1. Invoke `runStatusOptions` with args `["E07"]`

**Expected:**
- `EpicService.GetNextStatus()` was called
- JSON result contains `"entity_type": "epic"`

---

#### TC-OPT-08: Read-only -- no service write calls made

**Setup:**
- Mock `dispatchNextStatus` to return valid info
- Verify `dispatchTransition` is NEVER called

**Test steps:**
1. Invoke `runStatusOptions` with args `["E07-F01-001"]`

**Assertions:**
```go
assert.False(t, transitionCalled, "status options must be read-only")
```

---

#### TC-OPT-09: Entity not found -- exits with code 1

**Setup:**
- `dispatchNextStatus` returns error containing "not found"

**Expected:**
- `os.Exit(1)` is called
- Error message contains the key

---

### 3.4 `runStatusHistory` Handler Tests

**Test file section**: `TestRunStatusHistory`

---

#### TC-HIST-01: Task history -- human table output

**Setup:**
- `GetTaskHistory` returns:
  ```go
  []*models.TaskHistory{
      {OldStatus: nil, NewStatus: "todo", Timestamp: time.Now().Add(-2*time.Hour), Agent: nil, Notes: nil},
      {OldStatus: strPtr("todo"), NewStatus: "in_development", Timestamp: time.Now().Add(-1*time.Hour), Agent: strPtr("agent-1"), Notes: strPtr("Starting")},
  }
  ```

**Test steps:**
1. Invoke `runStatusHistory` with args `["E07-F01-001"]`

**Expected:**
- No error returned
- Output is a table with headers: Timestamp, Old Status, New Status, Agent, Notes
- Row 1: `(initial)` in Old Status column, `todo` in New Status
- Row 2: `todo` in Old Status, `in_development` in New Status, `agent-1` in Agent, `Starting` in Notes
- Footer indicates "Showing 2 history records"

**Assertions:**
```go
assert.NoError(t, err)
assert.Contains(t, output, "Timestamp")
assert.Contains(t, output, "Old Status")
assert.Contains(t, output, "New Status")
assert.Contains(t, output, "(initial)")
assert.Contains(t, output, "in_development")
assert.Contains(t, output, "agent-1")
assert.Contains(t, output, "Starting")
assert.Contains(t, output, "2 history records")
```

---

#### TC-HIST-02: Task history -- JSON output

**Setup:**
- Same as TC-HIST-01, with `cli.GlobalConfig.JSON = true`

**Expected:**
- Output is a JSON array
- Array has 2 elements
- Each element has `old_status`, `new_status`, `timestamp`, `agent`, `notes` fields

**Assertions:**
```go
var histories []map[string]interface{}
assert.NoError(t, json.Unmarshal([]byte(output), &histories))
assert.Len(t, histories, 2)
assert.Equal(t, "in_development", histories[1]["new_status"])
```

---

#### TC-HIST-03: Empty history -- info message, no table, exit 0

**Setup:**
- `GetTaskHistory` returns `[]*models.TaskHistory{}`

**Test steps:**
1. Invoke `runStatusHistory` with args `["E07-F01-001"]`

**Expected:**
- No error returned (exit 0)
- Info message contains "No history records"

**Assertions:**
```go
assert.NoError(t, err)
assert.Contains(t, output, "No history records")
```

---

#### TC-HIST-04: Non-task entity (feature key) -- clear error, no service call

**Test steps:**
1. Invoke `runStatusHistory` with args `["E07-F01"]`

**Expected:**
- Error returned
- Error message contains "only supported for tasks"
- Error message identifies the input as a feature key
- `GetTaskHistory` is NEVER called

**Assertions:**
```go
assert.Error(t, err)
assert.Contains(t, err.Error(), "only supported for tasks")
assert.Contains(t, err.Error(), "feature")
assert.False(t, historyServiceCalled)
```

---

#### TC-HIST-05: Non-task entity (epic key) -- clear error

**Test steps:**
1. Invoke `runStatusHistory` with args `["E07"]`

**Expected:**
- Error returned
- Error message contains "only supported for tasks"
- Error message identifies the input as an epic key

---

#### TC-HIST-06: Task not found -- exits with code 1

**Setup:**
- `GetTaskHistory` returns error containing "not found"

**Test steps:**
1. Invoke `runStatusHistory` with args `["E99-F99-999"]`

**Expected:**
- `os.Exit(1)` is called
- Error message contains `E99-F99-999`

---

#### TC-HIST-07: Traditional task key format (`T-E07-F01-001`) is accepted

**Test steps:**
1. Invoke `runStatusHistory` with args `["T-E07-F01-001"]`

**Expected:**
- Entity type is detected as "task" (T- prefix stripped correctly)
- `GetTaskHistory` is called with normalized key `E07-F01-001`

---

---

## 4. Integration Tests: Entity Auto-Detection and Workflow

These tests use the real `ParseGetArgs` function and test the full dispatch table, not just individual handlers. They may use a lightweight test setup (not necessarily a real database).

**Test file section**: `TestStatusGroupIntegration`

### 4.1 Entity Auto-Detection via `ParseGetArgs`

---

#### TC-INT-DET-01: Epic key detection

| Input | Expected EntityType | Expected Key |
|-------|--------------------|----|
| `E07` | `epic` | `E07` |
| `e07` | `epic` | `E07` (normalized uppercase) |
| `E07-epic-name` (slugged) | `epic` | `E07` |

---

#### TC-INT-DET-02: Feature key detection

| Input | Expected EntityType | Expected Key |
|-------|--------------------|----|
| `E07-F01` | `feature` | `E07-F01` |
| `e07-f01` | `feature` | `E07-F01` |
| `F01` | `feature` | `F01` |
| `E07-F01-feature-name` (slugged) | `feature` | `E07-F01` |

---

#### TC-INT-DET-03: Task key detection

| Input | Expected EntityType | Expected Key |
|-------|--------------------|----|
| `E07-F01-001` | `task` | `E07-F01-001` |
| `e07-f01-001` | `task` | `E07-F01-001` (normalized) |
| `T-E07-F01-001` | `task` | `E07-F01-001` (T- stripped) |
| `t-e07-f01-001` | `task` | `E07-F01-001` (normalized, T- stripped) |
| `E07-F01-001-task-name` (slugged) | `task` | `E07-F01-001` |

---

#### TC-INT-DET-04: Invalid key format returns error

| Input | Expected |
|-------|----------|
| `""` (empty) | Error: invalid key format |
| `not-a-key` | Error: invalid key format |
| `123` | Error: invalid key format |

---

### 4.2 Cobra Namespace Coexistence

These tests verify the critical design requirement: the existing `statusCmd.RunE` (dashboard) is NOT disrupted by the new subcommands.

---

#### TC-INT-COBRA-01: `shark status E07` still routes to dashboard (not `set`)

**Test:** Execute `statusCmd` directly with args `["E07"]`. Verify `runStatus` (existing handler) is called, NOT `runStatusSet`.

**Key verification:** `E07` does not match any registered subcommand name (`set`, `advance`, `options`, `history`), so Cobra falls through to `statusCmd.RunE`.

---

#### TC-INT-COBRA-02: `shark status E07-F01` routes to dashboard

**Test:** Execute `statusCmd` with args `["E07-F01"]`. Verify `runStatus` is called.

---

#### TC-INT-COBRA-03: `shark status set E07-F01-001 in_development` routes to `runStatusSet`

**Test:** Execute `statusCmd` with args `["set", "E07-F01-001", "in_development"]`. Verify `runStatusSet` is called with correct args.

---

#### TC-INT-COBRA-04: `shark status advance E07-F01-001` routes to `runStatusAdvance`

**Test:** Execute `statusCmd` with args `["advance", "E07-F01-001"]`. Verify `runStatusAdvance` is called.

---

#### TC-INT-COBRA-05: `shark status options E07-F01-001` routes to `runStatusOptions`

**Test:** Execute `statusCmd` with args `["options", "E07-F01-001"]`. Verify `runStatusOptions` is called.

---

#### TC-INT-COBRA-06: `shark status history E07-F01-001` routes to `runStatusHistory`

**Test:** Execute `statusCmd` with args `["history", "E07-F01-001"]`. Verify `runStatusHistory` is called.

---

### 4.3 Workflow Validation via Service

These tests verify that the command layer correctly passes workflow validation responsibilities to the service layer (no business logic in the command).

---

#### TC-INT-WF-01: Invalid transition is rejected by service -- command propagates error with correct exit code

**Test:** Mock `TransitionStatus` to return an error containing "invalid transition". Verify `runStatusSet` exits with code 3 (not code 2 or 0).

---

#### TC-INT-WF-02: Force flag bypasses workflow -- `opts.Force = true` is passed to service

**Test:** Invoke `runStatusSet` with `--force`. Capture `TransitionOptions` passed to service mock. Assert `opts.Force = true`.

---

#### TC-INT-WF-03: Workflow service determines next status for `advance` -- command does not hardcode statuses

**Test:** Mock `GetNextStatus` to return a custom transition `[{TargetStatus: "custom_status"}]`. Verify `runStatusAdvance` transitions to `custom_status` (not a hardcoded status). This proves the command delegates, not assumes.

---

### 4.4 Idempotency Contract

---

#### TC-INT-IDEM-01: Setting same status twice is idempotent -- second call returns exit 0

**Sequence:**
1. Call `runStatusSet("E07-F01-001", "in_development")` -- mock returns `Transitioned: true`
2. Call `runStatusSet("E07-F01-001", "in_development")` -- mock returns `Transitioned: false`

**Expected:** Both calls return nil (exit 0). Second call produces a no-op message.

---

#### TC-INT-IDEM-02: Idempotent set does NOT call transition service for history recording

**Test:** When `TransitionResult.Transitioned = false`, verify that no subsequent history-related service calls are made.

**Rationale:** The feature spec states "No history record created for no-op transitions". This is enforced at the service layer, but the command must not try to workaround it.

---

---

## 5. Backward Compatibility Tests

**Test file section**: `TestBackwardCompatibility`

These are regression tests that verify pre-existing commands are completely unaffected by the addition of `status_group.go`.

---

#### TC-BC-01: `make test` passes with zero failures

**Test:** Run `make test` after implementing `status_group.go`. All existing tests pass.

**Pass Gate:** `make test` returns exit code 0.

---

#### TC-BC-02: `shark task start` still works -- output unchanged

**Test:** Execute `shark task start <task-key>` and compare output against pre-E17-F07 snapshot.

**Expected:** Byte-identical output to snapshot.

---

#### TC-BC-03: `shark task complete` still works -- output unchanged

**Test:** Execute `shark task complete <task-key> --notes="Done"` and compare against snapshot.

---

#### TC-BC-04: `shark task approve` still works -- output unchanged

**Test:** Execute `shark task approve <task-key>` and compare against snapshot.

---

#### TC-BC-05: `shark epic set-status` still works -- output unchanged

**Test:** Execute `shark epic set-status <epic-key> active` and compare against snapshot.

---

#### TC-BC-06: `shark feature set-status` still works -- output unchanged

**Test:** Execute `shark feature set-status <feature-key> active` and compare against snapshot.

---

#### TC-BC-07: `shark epic next-status` still works -- output unchanged

**Test:** Execute `shark epic next-status <epic-key>` and compare against snapshot.

---

#### TC-BC-08: `shark feature next-status` still works -- output unchanged

**Test:** Execute `shark feature next-status <feature-key>` and compare against snapshot.

---

#### TC-BC-09: `shark status <key>` (dashboard) still works -- output unchanged

**Test:** Execute `shark status E07-F01` and verify the output is the feature dashboard. Compare against snapshot.

**Critical:** This is the primary namespace collision risk. If this test fails, the Cobra coexistence mechanism is broken.

---

#### TC-BC-10: `shark history` still works (pre-hidden-alias state)

**Test:** Execute `shark history` and verify it produces the project history output.

**Note:** The `history.go` hidden alias is a FOLLOW-ON task. Before that change, `shark history` must remain fully functional and unchanged.

---

#### TC-BC-11: `shark task block` still works -- output unchanged

**Test:** Execute `shark task block <task-key> --reason="Dependency"` and compare against snapshot.

---

#### TC-BC-12: Exit codes are unchanged for existing commands

**Test Matrix:**

| Command | Scenario | Expected Exit Code |
|---------|----------|--------------------|
| `shark task start E99-F99-999` | Not found | 1 |
| `shark task start E07-F01-001` | Success | 0 |
| `shark epic set-status E99 active` | Not found | 1 |
| `shark feature next-status E07-F01` | Success | 0 |

---

---

## 6. Edge Case Tests

**Test file section**: `TestStatusGroupEdgeCases`

### 6.1 Invalid Key Formats

---

#### TC-EDGE-KEY-01: Empty key string

**Test:** Invoke `runStatusSet` with args `["", "in_development"]`.

**Expected:** Error returned containing "invalid key format". No service calls made.

---

#### TC-EDGE-KEY-02: Malformed key (no dash pattern)

**Test:** Invoke `runStatusSet` with args `["notakey", "in_development"]`.

**Expected:** Error returned. No service calls made.

---

#### TC-EDGE-KEY-03: Case insensitivity -- lowercase key is normalized

**Test:** Invoke `runStatusSet` with args `["e07-f01-001", "in_development"]`.

**Expected:**
- Entity type detected as "task"
- Service is called with normalized key `E07-F01-001` (uppercase)
- No error returned (assuming mock returns success)

---

#### TC-EDGE-KEY-04: Slug suffix in task key -- slug stripped correctly

**Test:** Invoke `runStatusOptions` with args `["E07-F01-001-implement-feature"]`.

**Expected:**
- Entity type detected as "task"
- Service called with key `E07-F01-001` (slug removed)

---

#### TC-EDGE-KEY-05: `T-` prefix in task key -- prefix stripped correctly

**Test:** Invoke `runStatusSet` with args `["T-E07-F01-001", "in_development"]`.

**Expected:**
- Entity type detected as "task"
- Service called with key `E07-F01-001`

---

### 6.2 Missing and Non-Existent Entities

---

#### TC-EDGE-ENT-01: Task not found in set -- exit code 1, not exit code 3

**Test:** `dispatchTransition` returns error containing "not found".

**Expected:**
- `os.Exit(1)` is called (NOT 3)
- Error message is about entity not found, not invalid transition

---

#### TC-EDGE-ENT-02: Task not found in advance -- exit code 1

**Test:** `dispatchNextStatus` returns error containing "not found".

**Expected:** `os.Exit(1)`.

---

#### TC-EDGE-ENT-03: Task not found in options -- exit code 1

**Test:** `dispatchNextStatus` returns error containing "not found".

**Expected:** `os.Exit(1)`.

---

#### TC-EDGE-ENT-04: Task not found in history -- exit code 1

**Test:** `GetTaskHistory` returns error containing "not found".

**Expected:** `os.Exit(1)`.

---

### 6.3 Force Mode Edge Cases

---

#### TC-EDGE-FORCE-01: `--force` without `--reason` fails with helpful message

**Test:** Invoke `runStatusSet` with `--force` but no `--reason`, and mock returns `services.ErrForceReasonRequired`.

**Expected:**
- `os.Exit(3)` is called
- Error message tells user to provide `--reason` when using `--force`

---

#### TC-EDGE-FORCE-02: `--to` with `--force` in advance -- skips validation of `--to` against available transitions

**Test:** Mock `dispatchNextStatus` returns `AvailableTransitions: [{TargetStatus: "in_development"}]`. Invoke `runStatusAdvance` with `--to=completed --force --reason="Admin"`.

**Expected:**
- Transition to `completed` is attempted (force allows it)
- `opts.Force = true` is passed to service

---

### 6.4 Concurrent Access

---

#### TC-EDGE-CONC-01: Two goroutines calling `runStatusSet` on different entities concurrently

**Test:** Launch two goroutines simultaneously, each calling `runStatusSet` on different entity keys with independent mocks.

**Expected:**
- Both complete without panics or data races
- Output from each does not interleave incorrectly

**Note:** This is a smoke test for goroutine safety. Run with `go test -race`.

---

### 6.5 Empty Arguments

---

#### TC-EDGE-ARG-01: `runStatusSet` with missing status argument

**Test:** Invoke `runStatusSet` with args `["E07-F01-001"]` only (missing the second arg for target status). Cobra should reject this via `cobra.ExactArgs(2)`.

**Expected:** Cobra returns "accepts 2 arg(s), received 1" error.

---

#### TC-EDGE-ARG-02: `runStatusAdvance` with no arguments

**Test:** Invoke `runStatusAdvance` with no args. Cobra should reject via `cobra.ExactArgs(1)`.

**Expected:** Cobra returns argument count error.

---

#### TC-EDGE-ARG-03: `runStatusOptions` with extra argument

**Test:** Invoke `runStatusOptions` with args `["E07-F01-001", "extra"]`.

**Expected:** Cobra returns "accepts 1 arg(s), received 2" error.

---

#### TC-EDGE-ARG-04: `runStatusHistory` with extra argument

**Test:** Invoke `runStatusHistory` with args `["E07-F01-001", "extra"]`.

**Expected:** Cobra returns argument count error.

---

### 6.6 Status Value Edge Cases

---

#### TC-EDGE-STATUS-01: Status value is case-normalized (uppercase input)

**Test:** Invoke `runStatusSet` with args `["E07-F01-001", "IN_DEVELOPMENT"]`.

**Expected:** Service is called with normalized `"in_development"` (lowercase).

---

#### TC-EDGE-STATUS-02: Status value with leading/trailing whitespace is trimmed

**Test:** Invoke `runStatusSet` with args `["E07-F01-001", "  in_development  "]`.

**Expected:** Service is called with `"in_development"` (trimmed).

---

#### TC-EDGE-STATUS-03: Empty status string is rejected before service call

**Test:** Invoke `runStatusSet` with args `["E07-F01-001", ""]`.

**Expected:** Error returned before any service call. "status cannot be empty" or similar.

---

---

## 7. UAT Scenarios from Epic Plan

The following scenarios from the epic UAT plan (`uat-plan.md`) are mapped to this feature. These are end-to-end acceptance scenarios that use the full binary (not isolated handler tests).

### 7.1 Journey 1 Scenarios (DevAgent Daily Workflow)

---

#### UAT-J1-S03: Start Task via Status Advance

**Source:** `uat-plan.md` J1-S03

**Preconditions:**
- Task exists in `ready_for_development` status
- `SHARK_OUTPUT=json` set in environment

**Steps:**
1. Run: `shark status advance <task-key>`
2. Parse JSON response

**Expected:**
- Exit code 0
- Task status changes to `in_development`
- Response JSON includes the updated entity with new status
- Task history records the transition

**Pass Criteria:** Status is `in_development` and history entry created.

---

#### UAT-J1-S05: View Valid Transitions with `status options`

**Source:** `uat-plan.md` J1-S05

**Preconditions:**
- Task is in `in_development` status
- Advanced workflow profile configured

**Steps:**
1. Run: `shark status options <task-key> --json`

**Expected:**
- Exit code 0
- JSON output includes `current_status: "in_development"`
- JSON output includes `valid_transitions` array with at least one transition
- JSON output includes `phase` field
- JSON output includes `agent_type` field (or equivalent)

**Pass Criteria:** JSON structure matches specification and transitions are correct per workflow config.

---

#### UAT-J1-S06: Complete Task via Status Advance with Notes

**Source:** `uat-plan.md` J1-S06

**Preconditions:**
- Task is in `in_development` status

**Steps:**
1. Run: `shark status advance <task-key> --notes "Implementation complete"`

**Expected:**
- Exit code 0
- Task advances to `ready_for_code_review` (or appropriate next status)
- Notes are stored: visible in `shark status history <task-key>`

**Pass Criteria:** Status advanced and notes appear in history.

---

#### UAT-J1-S10: Idempotent Status Set

**Source:** `uat-plan.md` J1-S10

**Preconditions:**
- Task is in `in_development` status

**Steps:**
1. Run: `shark status set <task-key> in_development --json`

**Expected:**
- Exit code 0 (not an error -- this is the idempotent guarantee)
- JSON response includes `"changed": false`
- No new history record created (verify with `shark status history <task-key>`)

**Pass Criteria:** Returns success with `changed: false` and no new history entry.

---

### 7.2 Journey 4 Scenarios (Status Check and Decision Making)

---

#### UAT-J4-S04: Check Available Feature Transitions via `status options`

**Source:** `uat-plan.md` J4-S04

**Preconditions:**
- Feature `E18-F05` exists with known status (e.g., `active`)

**Steps:**
1. Run: `shark status options E18-F05 --json`

**Expected:**
- Exit code 0
- JSON shows `current_status` and `valid_transitions` array
- Transitions are correct per workflow configuration for features

**Pass Criteria:** Transitions match workflow config for feature-level statuses.

---

### 7.3 Namespace Collision Risk Scenarios

---

#### UAT-NS-01: `shark status E07` still shows progress dashboard

**Source:** `uat-plan.md` NS-01

**Steps:**
1. Run: `shark status E07`

**Expected:**
- Dashboard output appears (feature rollup, task counts, impediments)
- NOT routed to `runStatusSet` or any other new handler

**Pass Criteria:** Dashboard output is byte-identical to pre-E17-F07 output.

---

#### UAT-NS-02: `shark status set E07 active` changes epic status

**Source:** `uat-plan.md` NS-02

**Steps:**
1. Run: `shark status set E07 active --json`

**Expected:**
- Exit code 0
- JSON contains `"entity_type": "epic"` and `"to_status": "active"`
- Epic status changed in database

---

#### UAT-NS-03: `shark status advance E07-F01-001` advances task

**Source:** `uat-plan.md` NS-03

**Steps:**
1. Run: `shark status advance E07-F01-001 --json`

**Expected:**
- Exit code 0
- Task moves to next valid status
- JSON contains updated status

---

#### UAT-NS-04: `shark status options E07-F01-001` shows valid transitions

**Source:** `uat-plan.md` NS-04

**Steps:**
1. Run: `shark status options E07-F01-001 --json`

**Expected:**
- Exit code 0
- JSON contains `current_status` and `available_transitions`

---

#### UAT-NS-05: `shark status history E07-F01-001` shows history

**Source:** `uat-plan.md` NS-05

**Steps:**
1. Run: `shark status history E07-F01-001 --json`

**Expected:**
- Exit code 0
- JSON is an array of history entries
- Each entry has timestamp, old_status, new_status fields

---

### 7.4 Backward Compatibility Scenarios

---

#### UAT-BC-05: `shark task start/complete/approve` still work

**Source:** `uat-plan.md` BC-05

**Steps:**
1. Run `shark task start <task-key>` -- verify works and output matches pre-E17-F07 snapshot
2. Run `shark task complete <task-key>` -- verify works
3. Run `shark task approve <task-key>` -- verify works

**Pass Criteria:** All three commands produce exit 0 and identical output to pre-implementation snapshots.

---

#### UAT-BC-06: `shark epic/feature set-status` still work

**Source:** `uat-plan.md` BC-06

**Steps:**
1. Run `shark epic set-status <epic-key> active`
2. Run `shark feature set-status <feature-key> active`

**Pass Criteria:** Both commands work with exit 0 and unchanged output.

---

#### UAT-BC-07: `shark epic/feature next-status` still work

**Source:** `uat-plan.md` BC-07

**Steps:**
1. Run `shark epic next-status <epic-key>`
2. Run `shark feature next-status <feature-key>`

**Pass Criteria:** Both commands work with exit 0 and unchanged output.

---

### 7.5 Service Layer Integration Scenarios

---

#### UAT-CE-E15-01: `status set` uses service layer (no direct repo calls)

**Source:** `uat-plan.md` CE-E15-01

**Validation Method:** Code review of `status_group.go`.

**Pass Criteria:** The file contains zero calls to `repository.New*Repository()`. All data operations go through `cli.GetTaskService()`, `cli.GetFeatureService()`, or `cli.GetEpicService()`.

---

#### UAT-CE-E15-02: `status advance` uses workflow service for transitions

**Source:** `uat-plan.md` CE-E15-02

**Validation Method:** Code review -- `runStatusAdvance` calls `dispatchNextStatus` which calls service's `GetNextStatus()`, which internally uses `workflow.Service`.

**Pass Criteria:** No direct workflow config parsing in `status_group.go`.

---

---

## 8. Exit Criteria and Quality Gates

### Feature-Level Gate (MUST pass before marking E17-F07 complete)

| Criterion | Test Coverage | Pass Condition |
|-----------|--------------|----------------|
| `make test` green | TC-BC-01 | Exit code 0 |
| All handler unit tests pass | TC-SET-01 through TC-HIST-07 | All pass |
| Entity auto-detection correct | TC-INT-DET-01 through TC-INT-DET-04 | All pass |
| Cobra coexistence verified | TC-INT-COBRA-01 through TC-INT-COBRA-06 | All 6 pass |
| Idempotency contract verified | TC-SET-03, TC-SET-04, TC-INT-IDEM-01, TC-INT-IDEM-02 | All pass |
| Backward compat for key commands | TC-BC-02 through TC-BC-12 | All pass |
| Edge cases (invalid keys) | TC-EDGE-KEY-01 through TC-EDGE-KEY-05 | All pass |
| Edge cases (missing entities) | TC-EDGE-ENT-01 through TC-EDGE-ENT-04 | All pass |
| UAT journey scenarios | UAT-J1-S03, UAT-J1-S05, UAT-J1-S06, UAT-J1-S10 | All pass |
| Namespace collision tests | UAT-NS-01 through UAT-NS-05 | All pass |
| Service layer compliance | UAT-CE-E15-01, UAT-CE-E15-02 | Pass (code review) |

### Blocking Defect Categories

The following findings BLOCK release of this feature:

- Any existing command (BC-02 through BC-12) produces different output than before implementation
- `shark status E07` (dashboard) is disrupted in any way
- Any handler exits with wrong code (e.g., not-found exits 3 instead of 1)
- `shark status set` with current status returns non-zero exit code (idempotency broken)
- `status history` with a feature key does not produce a clear error message
- Any handler calls repository methods directly (bypasses service layer)
- Race condition detected by `go test -race`

### Non-Blocking Findings

The following are noted as issues to address post-release or in follow-on tasks:

- Minor wording differences in human-readable messages (as long as JSON is correct)
- Performance beyond 200ms for read-only commands under normal conditions
- The `shark history` hidden alias (tracked separately as a follow-on task)

---

## 9. Test Execution Order

Execute tests in this order to maximize signal clarity:

1. **TC-BC-01**: Run `make test` first. If the existing suite is broken, stop and fix before continuing.
2. **TC-INT-COBRA-01 through TC-INT-COBRA-06**: Verify Cobra routing is correct. These are the highest-risk area.
3. **TC-INT-DET-01 through TC-INT-DET-04**: Verify entity auto-detection. All handlers depend on this.
4. **TC-SET-01 through TC-SET-13**: Full coverage of `runStatusSet`.
5. **TC-ADV-01 through TC-ADV-10**: Full coverage of `runStatusAdvance`.
6. **TC-OPT-01 through TC-OPT-09**: Full coverage of `runStatusOptions`.
7. **TC-HIST-01 through TC-HIST-07**: Full coverage of `runStatusHistory`.
8. **TC-INT-WF-01 through TC-INT-WF-03**: Workflow delegation tests.
9. **TC-INT-IDEM-01, TC-INT-IDEM-02**: Idempotency contract.
10. **TC-BC-02 through TC-BC-12**: Full backward compatibility regression.
11. **TC-EDGE-***: All edge case tests.
12. **UAT-***: UAT scenarios using the full binary.
13. **go test -race**: Race condition detection (TC-EDGE-CONC-01).

---

## Appendix A: Test Scenario Summary

| Category | Test Count | Priority |
|----------|-----------|----------|
| Unit: `runStatusSet` | 13 (TC-SET-01 through TC-SET-13) | CRITICAL |
| Unit: `runStatusAdvance` | 10 (TC-ADV-01 through TC-ADV-10) | CRITICAL |
| Unit: `runStatusOptions` | 9 (TC-OPT-01 through TC-OPT-09) | CRITICAL |
| Unit: `runStatusHistory` | 7 (TC-HIST-01 through TC-HIST-07) | CRITICAL |
| Integration: Auto-Detection | 4 (TC-INT-DET-*) | HIGH |
| Integration: Cobra Coexistence | 6 (TC-INT-COBRA-*) | CRITICAL |
| Integration: Workflow Delegation | 3 (TC-INT-WF-*) | HIGH |
| Integration: Idempotency | 2 (TC-INT-IDEM-*) | HIGH |
| Backward Compatibility | 12 (TC-BC-*) | BLOCKER |
| Edge Cases: Key Format | 5 (TC-EDGE-KEY-*) | HIGH |
| Edge Cases: Missing Entities | 4 (TC-EDGE-ENT-*) | CRITICAL |
| Edge Cases: Force Mode | 2 (TC-EDGE-FORCE-*) | HIGH |
| Edge Cases: Concurrent Access | 1 (TC-EDGE-CONC-*) | MEDIUM |
| Edge Cases: Arguments | 4 (TC-EDGE-ARG-*) | HIGH |
| Edge Cases: Status Values | 3 (TC-EDGE-STATUS-*) | MEDIUM |
| UAT Scenarios | 12 (UAT-*) | HIGH |
| **TOTAL** | **97 test points** | |

---

*Last Updated: 2026-02-25*
*Feature: E17-F07 Status Subcommand Group*
*Advance feature with: `./bin/shark feature next-status E17-F07`*
