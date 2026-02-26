# E17-F06 Test Plan: Progress Command

**Feature**: E17-F06 Progress Command
**Author**: QA Agent
**Date**: 2026-02-25
**Status**: Ready for Development

---

## Table of Contents

1. [Test Scope and Objectives](#1-test-scope-and-objectives)
2. [Test Data Requirements](#2-test-data-requirements)
3. [Unit Tests: Argument Parsing](#3-unit-tests-argument-parsing)
4. [Unit Tests: Output Formatting](#4-unit-tests-output-formatting)
5. [Integration Tests: StatusService.GetDashboard()](#5-integration-tests-statusservicegetdashboard)
6. [Backward Compatibility: shark status alias](#6-backward-compatibility-shark-status-alias)
7. [Edge Cases](#7-edge-cases)
8. [UAT Scenarios from Epic Plan](#8-uat-scenarios-from-epic-plan)
9. [Non-Functional Requirements](#9-non-functional-requirements)
10. [Quality Gates](#10-quality-gates)
11. [Test Execution Checklist](#11-test-execution-checklist)

---

## 1. Test Scope and Objectives

### What is Being Tested

E17-F06 introduces `shark progress [EPIC] [FEATURE]` as a dedicated command for viewing entity progress rollups, health indicators, task breakdowns, and action items. The implementation is a near-copy of `status.go` that delegates entirely to `StatusService.GetDashboard()`.

### Primary Risk: Regression

Since `status.go` is NOT modified in Phase 1, the primary regression risk is low. The new `progress.go` file introduces a new command that wraps the same service call. The test focus is:

1. The new command's argument parsing is correct and matches `status.go` behavior
2. JSON output is structurally valid and complete
3. `shark status` continues to produce identical output as before
4. Edge cases (empty project, all completed, mixed statuses) are handled without panics or incorrect output

### Out of Scope for This Test Plan

- `--field` flag behavior (deferred to E17-F02; the progress command does not add special handling)
- Phase 2 `statusCmd.Deprecated` activation (deferred until E17-F07 is complete)
- `StatusService.GetDashboard()` internal correctness (already tested in `internal/status/status_test.go`)
- Batch operations (E17-F07 scope)

### Test File Location

```
internal/cli/commands/progress_test.go
```

---

## 2. Test Data Requirements

### Test Data for Integration Tests

A test project state with the following characteristics allows all test scenarios to be validated:

| Data Set | Description | Used In |
|----------|-------------|---------|
| **empty-project** | Project initialized, 0 epics, 0 features, 0 tasks | EC-01, EC-02 |
| **single-feature-all-todo** | 1 epic, 1 feature, 5 tasks all in `todo` | EC-03 |
| **single-feature-all-completed** | 1 epic, 1 feature, 5 tasks all in `completed` | EC-04 |
| **mixed-statuses** | 1 epic, 1 feature, tasks in todo/in_progress/blocked/completed | EC-05, IT-01 through IT-05 |
| **multi-epic** | 3 epics, 5 features, 20 tasks in various statuses | IT-06, IT-07, UAT scenarios |
| **blocked-tasks** | At least 2 blocked tasks with blocked_reason set | EC-06, IT-05 |
| **recent-completions** | Tasks completed within last 7 days | IT-08 |

### Test Data Setup Notes

- Unit tests use no database (mock or in-memory structs only)
- Integration smoke tests require a built binary (`make build`) and a real project directory
- The existing `shark-tasks.db` in this repository provides suitable real-world data for integration smoke tests

---

## 3. Unit Tests: Argument Parsing

### Overview

`parseProgressRequest(cmd, args)` is pure argument parsing with no database access. It must be tested to 100% coverage. These tests go in `internal/cli/commands/progress_test.go` using only `*cobra.Command` and string slices.

**Testing approach**: Direct function call with a manually constructed `*cobra.Command` that has the required flags registered. No database. No service calls.

---

### TC-PARSE-01: No arguments, no flags

```
Test Name:   TestParseProgressRequest_NoArgs
Preconditions: None
Input:       args=[], no flags set
Expected:    req.EpicKey == ""
             req.RecentWindow == ""
             req.IncludeArchived == false
             err == nil
Priority:    BLOCKER
```

**Rationale**: The most common invocation. Produces a full project dashboard.

---

### TC-PARSE-02: Epic key as positional argument

```
Test Name:   TestParseProgressRequest_EpicPositional
Preconditions: None
Input:       args=["E05"], no flags set
Expected:    req.EpicKey == "E05"
             err == nil
Priority:    BLOCKER
```

---

### TC-PARSE-03: Epic key via flag

```
Test Name:   TestParseProgressRequest_EpicFlag
Preconditions: None
Input:       args=[], --epic=E05
Expected:    req.EpicKey == "E05"
             err == nil
Priority:    HIGH
```

---

### TC-PARSE-04: Positional argument overrides flag

```
Test Name:   TestParseProgressRequest_PositionalOverridesFlag
Preconditions: None
Input:       args=["E05"], --epic=E07
Expected:    req.EpicKey == "E05"  (positional takes precedence)
             err == nil
Priority:    HIGH
```

**Rationale**: Matches `status.go` behavior. Positional takes precedence over flag.

---

### TC-PARSE-05: Combined feature format (E05-F02)

```
Test Name:   TestParseProgressRequest_CombinedFormat
Preconditions: None
Input:       args=["E05-F02"], no flags
Expected:    req.EpicKey == "E05"  (ParseListArgs extracts epic from combined key)
             err == nil
Priority:    HIGH
```

**Rationale**: Users and agents often pass `E05-F02` as a single argument; `ParseListArgs` handles the split.

---

### TC-PARSE-06: Separate epic and feature positional arguments

```
Test Name:   TestParseProgressRequest_EpicAndFeaturePositional
Preconditions: None
Input:       args=["E05", "F02"], no flags
Expected:    req.EpicKey == "E05"
             err == nil
Priority:    HIGH
```

---

### TC-PARSE-07: Recent window flag

```
Test Name:   TestParseProgressRequest_RecentWindow
Preconditions: None
Input:       args=[], --recent=7d
Expected:    req.RecentWindow == "7d"
             err == nil
Priority:    MEDIUM
```

---

### TC-PARSE-08: Include-archived flag

```
Test Name:   TestParseProgressRequest_IncludeArchived
Preconditions: None
Input:       args=[], --include-archived
Expected:    req.IncludeArchived == true
             err == nil
Priority:    MEDIUM
```

---

### TC-PARSE-09: Too many arguments returns error

```
Test Name:   TestParseProgressRequest_TooManyArgs
Preconditions: None
Input:       args=["E05", "F02", "extra-arg"], no flags
Expected:    err != nil (error from ParseListArgs)
Priority:    HIGH
```

**Rationale**: Ensures users get a clear error instead of silent mismatch.

---

### TC-PARSE-10: Lowercase epic key

```
Test Name:   TestParseProgressRequest_LowercaseEpic
Preconditions: None
Input:       args=["e05"], no flags
Expected:    req.EpicKey == "E05"  (normalized to uppercase by ParseListArgs)
             err == nil
Priority:    MEDIUM
```

**Note**: Verify behavior matches `status.go`'s `parseStatusRequest`. If `ParseListArgs` normalizes case, this test documents that. If it does not normalize, the expected output is `"e05"` and ServiceRequest.Validate() will reject it later.

---

### TC-PARSE-11: All flags combined

```
Test Name:   TestParseProgressRequest_AllFlags
Preconditions: None
Input:       args=["E07"], --recent=30d, --include-archived
Expected:    req.EpicKey == "E07"
             req.RecentWindow == "30d"
             req.IncludeArchived == true
             err == nil
Priority:    MEDIUM
```

---

### TC-PARSE-12: Empty epic flag (flag present but empty)

```
Test Name:   TestParseProgressRequest_EmptyEpicFlag
Preconditions: None
Input:       args=[], --epic=""
Expected:    req.EpicKey == ""
             err == nil
Priority:    LOW
```

---

## 4. Unit Tests: Output Formatting

### Overview

`outputProgressJSON(dashboard)` writes to stdout. These tests capture stdout and verify the JSON structure. No database access needed -- construct a `*status.StatusDashboard` directly.

**Testing approach**: Use `os.Pipe()` to capture stdout during the function call. Unmarshal the captured output and assert structure. This matches the approach shown in the technical design (Section 7).

---

### TC-JSON-01: Minimal dashboard marshals to valid JSON

```
Test Name:   TestOutputProgressJSON_MinimalDashboard
Input:
  dashboard = &status.StatusDashboard{
      Summary: &status.ProjectSummary{
          OverallProgress: 0.0,
          Epics:    &status.CountBreakdown{Total: 0, Active: 0},
          Features: &status.CountBreakdown{Total: 0, Active: 0},
          Tasks:    &status.StatusBreakdown{Total: 0},
          BlockedCount: 0,
      },
      Epics:        []*status.EpicSummary{},
      ActiveTasks:  map[string][]*status.TaskInfo{},
      BlockedTasks: []*status.BlockedTaskInfo{},
  }
Expected:
  - err == nil
  - Output is valid JSON (parseable without error)
  - JSON contains "summary" key
  - JSON contains "epics" key (empty array)
  - JSON contains "active_tasks" key
  - JSON contains "blocked_tasks" key
Priority: BLOCKER
```

---

### TC-JSON-02: Progress value appears in summary

```
Test Name:   TestOutputProgressJSON_ProgressValue
Input:
  dashboard with Summary.OverallProgress = 75.5
Expected:
  - Output contains `"overall_progress": 75.5`
Priority: HIGH
```

---

### TC-JSON-03: Epic health indicators in output

```
Test Name:   TestOutputProgressJSON_EpicHealthIndicators
Input:
  dashboard with Epics = []*EpicSummary{
      {Key: "E01", Title: "Test Epic", Health: "healthy", ProgressPercent: 100.0},
      {Key: "E02", Title: "Blocked Epic", Health: "critical", ProgressPercent: 20.0},
  }
Expected:
  - JSON output contains both epic keys
  - E01 health field is "healthy"
  - E02 health field is "critical"
Priority: HIGH
```

---

### TC-JSON-04: Blocked tasks appear in output

```
Test Name:   TestOutputProgressJSON_BlockedTasks
Input:
  dashboard with BlockedTasks = []*BlockedTaskInfo{
      {Key: "E01-F01-001", Title: "Blocked Task", Feature: "E01-F01", Epic: "E01"},
  }
Expected:
  - JSON output contains "blocked_tasks" array with 1 entry
  - Entry contains "key": "E01-F01-001"
Priority: HIGH
```

---

### TC-JSON-05: Recent completions omitted when nil

```
Test Name:   TestOutputProgressJSON_NoRecentCompletions
Input:
  dashboard with RecentCompletions = nil
Expected:
  - "recent_completions" key absent from JSON (omitempty)
Priority: MEDIUM
```

---

### TC-JSON-06: Recent completions present when populated

```
Test Name:   TestOutputProgressJSON_WithRecentCompletions
Input:
  dashboard with RecentCompletions = []*CompletionInfo{
      {Key: "E01-F01-005", Title: "Done Task", CompletedAt: time.Now()},
  }
Expected:
  - "recent_completions" array present in JSON with 1 entry
Priority: MEDIUM
```

---

### TC-JSON-07: Filter metadata in output when present

```
Test Name:   TestOutputProgressJSON_WithFilter
Input:
  dashboard with Filter = &DashboardFilter{EpicKey: strptr("E01"), IncludeArchived: false}
Expected:
  - JSON contains "filter" key
  - "epic_key" is "E01"
Priority: LOW
```

---

## 5. Integration Tests: StatusService.GetDashboard()

### Overview

These are manual smoke tests run after building the binary. They verify the complete request-to-output path using the real database. They cannot be fully automated as unit tests because `GetStatusService()` uses the global database connection, matching the same limitation as `status_test.go`.

**Execution method**: Run the built binary against the project's own `shark-tasks.db`.

```bash
# Build first
make build

# Then run each scenario
./bin/shark progress [args]
```

---

### TC-INT-01: Full project dashboard

```
Test Name:   Integration_FullProjectDashboard
Command:     ./bin/shark progress
Expected:
  - Exit code 0
  - Terminal output contains project summary section
  - Terminal output contains epic list with progress indicators
  - No panic, no empty output
Priority:    BLOCKER
```

---

### TC-INT-02: Epic-filtered dashboard

```
Test Name:   Integration_EpicFiltered
Command:     ./bin/shark progress E17
Expected:
  - Exit code 0
  - Output shows only E17 data
  - Other epics not shown
  - No error about invalid epic key
Priority:    BLOCKER
```

---

### TC-INT-03: Epic-filtered dashboard via combined feature format

```
Test Name:   Integration_CombinedFeatureFormat
Command:     ./bin/shark progress E17-F06
Expected:
  - Exit code 0
  - Output is filtered to E17 (same as TC-INT-02 since feature-level filter not supported)
  - No error or panic
Priority:    HIGH
```

---

### TC-INT-04: Feature-filtered via two positional args

```
Test Name:   Integration_EpicAndFeaturePositional
Command:     ./bin/shark progress E17 F06
Expected:
  - Exit code 0
  - Output is filtered to E17
  - No error or panic
Priority:    HIGH
```

---

### TC-INT-05: JSON output mode

```
Test Name:   Integration_JSONOutput
Command:     ./bin/shark progress --json
Expected:
  - Exit code 0
  - Output is valid JSON (parseable with `jq .`)
  - Contains "summary" key with numeric progress value
  - Contains "epics" array
  - Contains "blocked_tasks" array
Priority:    BLOCKER
Verification:
  ./bin/shark progress --json | jq '.summary.overall_progress'
  # Must return a numeric value, not null
```

---

### TC-INT-06: JSON output with epic filter

```
Test Name:   Integration_JSONOutputEpicFiltered
Command:     ./bin/shark progress E17 --json
Expected:
  - Exit code 0
  - Valid JSON
  - "filter" object present: {"epic_key": "E17", ...}
Priority:    HIGH
Verification:
  ./bin/shark progress E17 --json | jq '.filter.epic_key'
  # Must return "E17"
```

---

### TC-INT-07: Recent completions window

```
Test Name:   Integration_RecentWindow
Command:     ./bin/shark progress --recent=7d
Expected:
  - Exit code 0
  - No error about invalid timeframe
  - If any tasks completed in last 7 days: "recent_completions" section appears in output
Priority:    MEDIUM
```

---

### TC-INT-08: Invalid epic key returns error

```
Test Name:   Integration_InvalidEpicKey
Command:     ./bin/shark progress ZZ99
Expected:
  - Exit code non-zero
  - Error message contains "invalid epic key" or similar
  - No panic
Priority:    HIGH
```

**Note**: The validation happens in `StatusRequest.Validate()` which checks the regex `^E\d+$`. `ZZ99` does not match, so an error is returned.

---

### TC-INT-09: Invalid timeframe returns error

```
Test Name:   Integration_InvalidTimeframe
Command:     ./bin/shark progress --recent=99x
Expected:
  - Exit code non-zero
  - Error message contains "invalid timeframe"
  - Valid timeframes listed in error output
Priority:    MEDIUM
```

---

### TC-INT-10: Include-archived flag

```
Test Name:   Integration_IncludeArchived
Command:     ./bin/shark progress --include-archived
Expected:
  - Exit code 0
  - No error
  - Output may include more epics/features than without the flag
Priority:    LOW
```

---

## 6. Backward Compatibility: shark status alias

### Criticality

Backward compatibility is a NON-NEGOTIABLE requirement (NFR-1). The `shark status` command must continue to work identically. These tests are BLOCKER-priority.

### Phase 1 Scope

In Phase 1 (this feature), `status.go` is NOT modified. The `statusCmd` is unchanged. These tests verify that F06's addition of `progressCmd` does not interfere with `statusCmd`.

---

### TC-BC-01: shark status (no args) still works

```
Test Name:   BC_StatusNoArgs
Command:     ./bin/shark status
Expected:
  - Exit code 0
  - Output is identical to pre-F06 output (same terminal dashboard)
  - No "deprecated" warning (Phase 1 only)
Priority:    BLOCKER
```

---

### TC-BC-02: shark status with epic filter still works

```
Test Name:   BC_StatusEpicFilter
Command:     ./bin/shark status E17
Expected:
  - Exit code 0
  - Output is identical to pre-F06 output
  - No interference from progressCmd registration
Priority:    BLOCKER
```

---

### TC-BC-03: shark status --json still works

```
Test Name:   BC_StatusJSON
Command:     ./bin/shark status --json
Expected:
  - Exit code 0
  - JSON output identical to pre-F06 output
  - No structure changes
Priority:    BLOCKER
Verification:
  # Capture before F06:
  ./bin/shark status --json > before.json
  # After F06 implementation:
  ./bin/shark status --json > after.json
  diff before.json after.json
  # Diff must be empty
```

---

### TC-BC-04: shark status output equals shark progress output

```
Test Name:   BC_ProgressMatchesStatus
Command A:   ./bin/shark status --json > status-output.json
Command B:   ./bin/shark progress --json > progress-output.json
Expected:
  - Both commands exit with code 0
  - Both JSON outputs are structurally identical
  - All numeric values are equal (same underlying data)
Priority:    BLOCKER
Verification:
  diff <(./bin/shark status --json) <(./bin/shark progress --json)
  # Diff must be empty
```

**Rationale**: Both commands call `cli.GetStatusService().GetDashboard()` with the same request (no filters). They must produce byte-identical output.

---

### TC-BC-05: shark status --help does NOT mention "deprecated"

```
Test Name:   BC_StatusHelpNoDeprecated
Command:     ./bin/shark status --help
Expected:
  - Exit code 0
  - Output does NOT contain "deprecated" or "DEPRECATED"
  - Output does NOT contain "Use 'shark progress'"
  - Phase 2 deprecation TODO comment is NOT uncommented
Priority:    BLOCKER
```

**Rationale**: The deprecation line is a commented TODO in Phase 1. It must not be accidentally activated.

---

### TC-BC-06: make test passes without modification

```
Test Name:   BC_ExistingTestSuite
Command:     make test
Expected:
  - Exit code 0
  - All existing tests pass
  - No new test failures introduced by progress.go
  - The existing status_test.go skipped test remains skipped (not newly broken)
Priority:    BLOCKER
```

---

## 7. Edge Cases

### Overview

These tests validate behavior when the project is in unusual states. They are executed as integration smoke tests using the binary.

---

### TC-EC-01: Empty project (no epics)

```
Test Name:   EC_EmptyProject_NoEpics
Setup:       Initialize fresh shark project with no entities created
Command:     ./bin/shark progress
Expected:
  - Exit code 0
  - No panic
  - Output indicates no active work (e.g., empty dashboard or "No epics found")
  - Does NOT produce: nil pointer panic, empty JSON object {}, or partial output
Priority:    HIGH
JSON verification:
  ./bin/shark progress --json | jq '.summary.tasks.total'
  # Must return 0, not null
```

---

### TC-EC-02: Empty project JSON output structure

```
Test Name:   EC_EmptyProject_JSONStructure
Setup:       Fresh project with no entities
Command:     ./bin/shark progress --json
Expected:
  - Exit code 0
  - Valid JSON with all required keys present:
    "summary", "epics", "active_tasks", "blocked_tasks"
  - summary.overall_progress = 0
  - epics = [] (empty array, not null)
  - active_tasks = {} (empty object, not null)
  - blocked_tasks = [] (empty array, not null)
Priority:    HIGH
```

---

### TC-EC-03: Feature with all tasks in todo

```
Test Name:   EC_AllTodo
Setup:       1 epic, 1 feature, 5 tasks all in "todo" status
Command:     ./bin/shark progress --json
Expected:
  - Exit code 0
  - summary.overall_progress = 0.0 (or near 0 depending on weight config)
  - No blocked tasks
  - No active tasks (todo tasks are not "active" until started)
  - Epic health is "healthy" or "warning" (not "critical" without blockers)
Priority:    HIGH
```

---

### TC-EC-04: Feature with all tasks completed

```
Test Name:   EC_AllCompleted
Setup:       1 epic, 1 feature, 5 tasks all in "completed" status
Command:     ./bin/shark progress --json
Expected:
  - Exit code 0
  - summary.overall_progress = 100.0 (or high value per weight config)
  - active_tasks is empty or absent
  - blocked_tasks is empty
  - Epic health is "healthy"
  - Epic progress_percent = 100.0
Priority:    HIGH
```

---

### TC-EC-05: Mixed statuses in one feature

```
Test Name:   EC_MixedStatuses
Setup:       1 epic, 1 feature with tasks:
             - 2 tasks in "todo"
             - 2 tasks in "in_progress"
             - 1 task in "blocked" (with reason "waiting on external API")
             - 2 tasks in "completed"
Command:     ./bin/shark progress --json
Expected:
  - Exit code 0
  - summary.blocked_count >= 1
  - blocked_tasks array has 1 entry with the blocked reason
  - active_tasks has entries for in_progress tasks
  - Epic health reflects blocked state ("warning" or "critical")
  - overall_progress is between 0 and 100 (non-zero due to completed tasks)
Priority:    HIGH
```

---

### TC-EC-06: Multiple epics with different health states

```
Test Name:   EC_MultipleEpics_MixedHealth
Setup:       Project with 3 epics:
             - Epic A: all tasks completed (healthy, 100%)
             - Epic B: tasks in progress, no blockers (healthy, ~50%)
             - Epic C: multiple blocked tasks (critical)
Command:     ./bin/shark progress --json
Expected:
  - Exit code 0
  - epics array has 3 entries
  - Epic A health is "healthy", progress_percent is 100
  - Epic C health is "critical"
  - summary.blocked_count reflects total blocked tasks across all epics
Priority:    MEDIUM
```

---

### TC-EC-07: Invalid epic key in filter

```
Test Name:   EC_InvalidEpicKeyFilter
Command:     ./bin/shark progress INVALID_KEY
Expected:
  - Exit code non-zero
  - Error message mentions "invalid epic key format"
  - No panic, clean error propagation
Priority:    HIGH
```

---

### TC-EC-08: Context timeout (5-second guard)

```
Test Name:   EC_ContextNilGuard
Scenario:    Verify the nil context guard in runProgress executes without panic
             when cmd.Context() returns nil (test invocation scenario)
Method:      This is validated by the progress_test.go tests that construct a
             cobra.Command without a context. The nil guard must not panic.
Expected:
  - parseProgressRequest calls complete without panic
  - outputProgressJSON/outputProgressTerminal calls complete without panic
Priority:    MEDIUM
```

---

## 8. UAT Scenarios from Epic Plan

The following scenarios from the epic UAT plan (`uat-plan.md`) apply directly to E17-F06. They are acceptance gates for Phase 2 completion.

---

### UAT-J2-S05: Feature Progress Check (Journey 2, Scenario 5)

**Source**: uat-plan.md, Section 2, Journey 2

```
Test Name:   UAT_J2S05_FeatureProgressCheck
Command:     ./bin/shark progress E18-F05 --json
(Use actual feature key from test project)
Preconditions:
  - Feature exists with tasks in various statuses
Expected:
  - Exit code 0
  - JSON includes: progress_pct (weighted), completion_pct, total_tasks
  - JSON includes task_breakdown (count by status)
  - JSON includes health indicator ("healthy", "warning", or "critical")
  - Progress values are numerically correct (verify manually against known task distribution)
Pass/Fail:   Progress data is present and numerically correct.
Priority:    HIGH (Phase 2 gate)
```

**Note**: The UAT plan references `progress_pct` as a top-level field. Verify this maps to `summary.overall_progress` or an equivalent epic-level field in `StatusDashboard`. If the field path differs, document the actual path.

---

### UAT-J4-S01: Feature Progress with Task Rollup (Journey 4, Scenario 1)

**Source**: uat-plan.md, Section 2, Journey 4

```
Test Name:   UAT_J4S01_FeatureProgressWithTaskRollup
Command:     ./bin/shark progress E18-F05 --json
Preconditions:
  - Feature with tasks in various statuses
Expected:
  - Exit code 0
  - JSON contains all of: progress_pct/overall_progress, completion data,
    total_tasks, task_breakdown by status, health indicator
Pass/Fail:   All required fields present and numerically correct.
Priority:    HIGH (Phase 2 gate)
```

---

### UAT-J4-S02: Epic Progress with Feature Rollup (Journey 4, Scenario 2)

**Source**: uat-plan.md, Section 2, Journey 4

```
Test Name:   UAT_J4S02_EpicProgressWithFeatureRollup
Command:     ./bin/shark progress E18 --json
(Use actual epic key from test project)
Preconditions:
  - Epic with multiple features at different progress levels
Expected:
  - Exit code 0
  - JSON contains "epics" array with feature rollup data
  - JSON contains "blocked_tasks" (impediments list)
  - Rollup data matches manual calculation from known task states
Pass/Fail:   Rollup data matches expected values.
Priority:    HIGH (Phase 2 gate)
```

---

### UAT-J2-S06: Progress Field Extraction (Journey 2, Scenario 6)

**Source**: uat-plan.md, Section 2, Journey 2

```
Test Name:   UAT_J2S06_ProgressFieldExtraction
Command:     ./bin/shark progress E18-F05 --field progress_pct
Preconditions:
  - E17-F02 (--field flag) is implemented and available
  - Feature with known progress value
Expected:
  - Exit code 0
  - Output is a raw number (e.g., 78.5)
  - No JSON wrapping
Pass/Fail:   Single numeric value on output.
Priority:    HIGH (E17-F02 dependency -- test after F02 is implemented)
Note:        This test is BLOCKED on E17-F02 implementation. Mark as
             deferred until F02 is available.
```

---

### UAT-BC-05: shark status E18-F05 still works (Backward Compatibility)

**Source**: uat-plan.md, Section 5, Priority 1

```
Test Name:   UAT_BC05_StatusStillWorks
Command A:   ./bin/shark status E17 --json
Command B:   ./bin/shark progress E17 --json
Preconditions:
  - F06 implemented and binary built
Expected:
  - Both commands exit with code 0
  - Both produce identical JSON output
Pass/Fail:   diff <(./bin/shark status E17 --json) <(./bin/shark progress E17 --json) returns empty.
Priority:    BLOCKER
```

---

### UAT-SVC-04: StatusService.GetDashboard() callable from progress

**Source**: uat-plan.md, Section 5, Priority 6

```
Test Name:   UAT_SVC04_GetDashboardCallable
Validation:  Code review of progress.go
Expected:
  - runProgress calls cli.GetStatusService().GetDashboard(ctx, req)
  - No direct repository calls in progress.go
  - No business logic in progress.go (thin wrapper only)
Pass/Fail:   Code review confirms no repo calls; functional test confirms output.
Priority:    HIGH
```

---

## 9. Non-Functional Requirements

### NFR-1: Backward Compatibility

All TC-BC-* tests above. The `make test` suite must continue to pass with zero failures.

### NFR-3: Service Layer Integration

**Test**: Code review of `progress.go`

```
Acceptance Criteria (all must pass):
- [ ] progress.go contains zero direct repository calls
- [ ] progress.go calls only: cli.GetStatusService(), cli.GetDisplayService(), enrichEpicSummaries()
- [ ] enrichEpicSummaries is NOT duplicated in progress.go (called from status.go, same package)
- [ ] No "fat controller" anti-patterns (no data filtering, no status derivation)
```

### NFR-4: Testing Coverage

```
Coverage Targets:
- parseProgressRequest: 100% (all branches covered by TC-PARSE-*)
- outputProgressJSON: 80%+ (major paths covered by TC-JSON-*)
- outputProgressTerminal: 50%+ (partial coverage; full coverage blocked by mock injection limitation)
- runProgress: Deferred (same limitation as status_test.go)
- Overall progress.go: Target 70%+ coverage
```

**Measurement**:
```bash
go test ./internal/cli/commands/ -run TestParseProgressRequest -v -coverprofile=cover.out
go tool cover -func=cover.out | grep progress
```

### NFR-2: Performance

```
Single command latency:
- ./bin/shark progress: < 200ms average over 10 runs
- ./bin/shark progress --json: < 200ms average
- --field overhead (when E17-F02 available): < 10ms additional

Measurement:
  for i in $(seq 1 10); do
    /usr/bin/time -f "%e" ./bin/shark progress --json 2>&1 | tail -1
  done
```

### NFR-5: Non-Interactive Operation

```
Test:        echo "" | ./bin/shark progress
             echo "" | ./bin/shark progress --json
Expected:    Both succeed with exit code 0. No stdin prompt.
```

---

## 10. Quality Gates

The following gates must ALL pass before E17-F06 is considered complete.

### Gate 1: Unit Tests (BLOCKER)

```
make test
```

Exit code must be 0. All existing tests continue to pass. New tests in `progress_test.go` pass.

### Gate 2: Argument Parsing Coverage (BLOCKER)

`parseProgressRequest` must be covered by at minimum TC-PARSE-01 through TC-PARSE-05 and TC-PARSE-09. Full coverage (TC-PARSE-01 through TC-PARSE-12) is the target.

### Gate 3: JSON Output Correctness (BLOCKER)

TC-JSON-01 and TC-JSON-02 must pass. `outputProgressJSON` produces valid, parseable JSON with the `summary.overall_progress` field present.

### Gate 4: Backward Compatibility (BLOCKER)

TC-BC-01 through TC-BC-06 must all pass. `make test` green. `shark status` output is byte-identical before and after F06.

### Gate 5: Integration Smoke Tests (HIGH)

TC-INT-01 through TC-INT-05 must pass manually before PR submission.

### Gate 6: Code Review (HIGH)

```
Quality Checklist (from technical-design.md Section 10):
- [ ] Thin wrapper: parse -> call service -> format output. No business logic in command.
- [ ] Calls service via cli.GetStatusService() global accessor (no direct repo access)
- [ ] enrichEpicSummaries called from status.go (same package), NOT duplicated
- [ ] --json routes to outputProgressJSON, terminal routes to outputProgressTerminal
- [ ] --epic, --recent, --include-archived flags registered with identical behavior to status
- [ ] Positional args handled by ParseListArgs
- [ ] Error from GetDashboard wrapped with "failed to get progress: %w"
- [ ] Context nil guard matches status.go pattern
- [ ] statusCmd.Deprecated is commented TODO in status.go (Phase 1 only, NOT uncommented)
- [ ] make fmt && make lint && make test all pass green
```

---

## 11. Test Execution Checklist

### Developer Checklist (before PR)

```
[ ] TC-PARSE-01 through TC-PARSE-12 implemented in progress_test.go and passing
[ ] TC-JSON-01 through TC-JSON-07 implemented in progress_test.go and passing
[ ] make test exits 0 (TC-BC-06)
[ ] ./bin/shark progress produces non-empty output (TC-INT-01)
[ ] ./bin/shark progress --json | jq . produces valid JSON (TC-INT-05)
[ ] diff <(./bin/shark status --json) <(./bin/shark progress --json) is empty (TC-BC-04)
[ ] ./bin/shark status --help does not mention "deprecated" (TC-BC-05)
[ ] Code review checklist completed (Gate 6)
```

### QA Checklist (after PR, before feature approval)

```
[ ] make test passes (Gate 1)
[ ] TC-INT-01 through TC-INT-10 executed manually, results documented
[ ] TC-BC-01 through TC-BC-05 executed manually, results documented
[ ] TC-EC-01 through TC-EC-07 executed manually against empty/edge-case projects
[ ] UAT-J2-S05 executed (feature progress check)
[ ] UAT-J4-S02 executed (epic progress with feature rollup)
[ ] UAT-BC-05 executed (shark status == shark progress output)
[ ] Performance: ./bin/shark progress --json completes in < 200ms
[ ] No direct repository calls in progress.go (code review)
[ ] enrichEpicSummaries not duplicated (code review)
[ ] Phase 2 TODO comment is present but NOT uncommented in status.go
```

### Deferred Tests (require E17-F02)

```
[ ] UAT-J2-S06: shark progress E18-F05 --field progress_pct
[ ] UAT-J4-S03: shark progress E18-F05 --field progress_pct (same)
[ ] TC-FIELD-01 through TC-FIELD-N (to be defined in E17-F02 test plan)
```

---

## Appendix A: Test Implementation Skeleton

The following is a sketch of the test file structure to guide the developer. This is NOT production code -- it is a starting template.

```go
// File: internal/cli/commands/progress_test.go
package commands

import (
    "bytes"
    "encoding/json"
    "io"
    "os"
    "testing"

    "github.com/jwwelbor/shark-task-manager/internal/status"
    "github.com/spf13/cobra"
)

// newTestProgressCmd creates a minimal cobra.Command with progress flags registered.
// Used by parseProgressRequest tests to avoid requiring the full CLI initialization.
func newTestProgressCmd() *cobra.Command {
    cmd := &cobra.Command{}
    cmd.Flags().String("epic", "", "")
    cmd.Flags().String("recent", "", "")
    cmd.Flags().Bool("include-archived", false, "")
    return cmd
}

// captureStdout captures standard output during fn() and returns it as a string.
func captureStdout(fn func()) string {
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    fn()

    _ = w.Close()
    os.Stdout = old

    var buf bytes.Buffer
    _, _ = io.Copy(&buf, r)
    return buf.String()
}

// TC-PARSE-01
func TestParseProgressRequest_NoArgs(t *testing.T) {
    cmd := newTestProgressCmd()
    req, err := parseProgressRequest(cmd, []string{})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if req.EpicKey != "" {
        t.Errorf("expected empty EpicKey, got %q", req.EpicKey)
    }
    if req.RecentWindow != "" {
        t.Errorf("expected empty RecentWindow, got %q", req.RecentWindow)
    }
    if req.IncludeArchived {
        t.Error("expected IncludeArchived=false")
    }
}

// TC-PARSE-02
func TestParseProgressRequest_EpicPositional(t *testing.T) {
    cmd := newTestProgressCmd()
    req, err := parseProgressRequest(cmd, []string{"E05"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if req.EpicKey != "E05" {
        t.Errorf("expected EpicKey=E05, got %q", req.EpicKey)
    }
}

// TC-PARSE-09
func TestParseProgressRequest_TooManyArgs(t *testing.T) {
    cmd := newTestProgressCmd()
    _, err := parseProgressRequest(cmd, []string{"E05", "F02", "extra"})
    if err == nil {
        t.Error("expected error for too many args, got nil")
    }
}

// TC-JSON-01
func TestOutputProgressJSON_MinimalDashboard(t *testing.T) {
    dashboard := &status.StatusDashboard{
        Summary: &status.ProjectSummary{
            OverallProgress: 0.0,
            Epics:           &status.CountBreakdown{},
            Features:        &status.CountBreakdown{},
            Tasks:           &status.StatusBreakdown{},
        },
        Epics:        []*status.EpicSummary{},
        ActiveTasks:  map[string][]*status.TaskInfo{},
        BlockedTasks: []*status.BlockedTaskInfo{},
    }

    output := captureStdout(func() {
        if err := outputProgressJSON(dashboard); err != nil {
            t.Errorf("outputProgressJSON returned error: %v", err)
        }
    })

    var parsed map[string]interface{}
    if err := json.Unmarshal([]byte(output), &parsed); err != nil {
        t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, output)
    }

    for _, key := range []string{"summary", "epics", "active_tasks", "blocked_tasks"} {
        if _, ok := parsed[key]; !ok {
            t.Errorf("missing key %q in JSON output", key)
        }
    }
}

// Additional test cases follow the same pattern for TC-PARSE-03 through TC-PARSE-12
// and TC-JSON-02 through TC-JSON-07.
```

---

## Appendix B: Scenario-to-Acceptance Criteria Traceability

| Acceptance Criterion (feature.md) | Test Cases |
|----------------------------------|------------|
| `shark progress E18-F05` shows feature progress | TC-INT-02, UAT-J4-S01 |
| `shark progress E18` shows epic progress | TC-INT-02, UAT-J4-S02 |
| `shark progress E18-F05-001` shows task progress context | TC-INT-03 (note: design scopes to dashboard, not per-task) |
| `--field` flag works: `--field progress_pct` returns `78.5` | UAT-J2-S06 (blocked on E17-F02) |
| `--json` returns structured progress data | TC-INT-05, TC-JSON-01 through TC-JSON-07 |
| Existing `shark status <id>` is a hidden alias | TC-BC-01 through TC-BC-05, UAT-BC-05 |
| Health indicators: healthy, warning, critical | TC-JSON-03, TC-EC-05, TC-EC-06, UAT-J2-S05 |
| Action items: tasks requiring attention | TC-JSON-04, TC-EC-05, UAT-J4-S01 |
| All existing tests pass | TC-BC-06, Gate 1 |

---

*Last Updated*: 2026-02-25
*Feature*: E17-F06 Progress Command
*Epic UAT Plan*: docs/plan/E17-cli-simplification-for-ai-agents/uat-plan.md
