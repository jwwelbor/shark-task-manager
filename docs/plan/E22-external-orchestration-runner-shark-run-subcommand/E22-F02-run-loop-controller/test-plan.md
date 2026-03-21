# Test Plan: E22-F02 Run Loop Controller

**Feature**: E22-F02 - Run Loop Controller
**Epic**: E22 - External Orchestration Runner
**Date**: 2026-03-21
**Status**: Draft

---

## 1. Scope and Approach

This test plan covers all acceptance criteria for the `RunController`, `shark run` Cobra command, `GetActionService()` global accessor, and the entity-type adapter implementations in `run.go`.

**Testing Strategy**:
- All `RunController` tests use **mocked dependencies only** — no real database, no real agent subprocesses (follows project testing golden rule: REQ-F02-NF-004).
- Mock infrastructure mirrors the pattern established in `internal/runner/dispatcher_test.go` and `internal/runner/claude_dispatcher_test.go` (function-field mocks, `helperCommand` subprocess pattern for real process tests).
- CLI command tests (`run.go`) use mocked services, never real DB.
- `GetActionService()` accessor is tested via compile-time interface satisfaction and a unit test that verifies `sync.Once` caching behavior with a stubbed config path.

**Primary test file**: `internal/runner/controller_test.go`
**Secondary test file**: `internal/cli/commands/run_test.go` (argument parsing, output formatting, error-to-exit-code mapping)
**Accessor test location**: `internal/cli/services_global_test.go` (or inline in `run_test.go` if the accessor test is minimal)

---

## 2. AC Test Matrix

### REQ-F02-001: RunController Struct with Constructor Injection

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-001-1 | `TestNewRunController_ValidDeps` | Provide all valid mocked deps | `RunControllerDeps{Transitioner, Placeholders, ActionSvc, WorkflowSvc, Dispatchers}` | Controller created without panic/error | — |
| TC-F02-001-2 | `TestNewRunController_NilTransitioner` | Nil `Transitioner` in deps | `RunControllerDeps{Transitioner: nil, ...}` | Panics or returns error | Required dep is nil |
| TC-F02-001-3 | `TestNewRunController_NilActionSvc` | Nil `ActionSvc` in deps | `RunControllerDeps{ActionSvc: nil, ...}` | Panics or returns error | Required dep is nil |
| TC-F02-001-4 | `TestNewRunController_NilWorkflowSvc` | Nil `WorkflowSvc` in deps | `RunControllerDeps{WorkflowSvc: nil, ...}` | Panics or returns error | Required dep is nil |
| TC-F02-001-5 | `TestNewRunController_InterfaceTypes` | Compile-time check | `var _ EntityTransitioner = (*MockTransitioner)(nil)` | Compiles without error | Interface contract validation |

**Notes**:
- TC-F02-001-2 through 4 may use `require.Panics` (testify) or a `defer/recover` pattern if the constructor panics. If it returns `error`, use standard error check.
- TC-F02-001-5 is a compile-time assertion using `var _ Interface = (*Mock)(nil)` pattern (mirrors `var _ AgentDispatcher = &ClaudeDispatcher{}` in `dispatcher_test.go`).

---

### REQ-F02-002: EntityTransitioner Interface

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-002-1 | `TestEntityTransitioner_InterfaceShape` | Compile-time check | `var _ EntityTransitioner = (*MockTransitioner)(nil)` | Compiles | Mock satisfies interface |
| TC-F02-002-2 | `TestEntityTransitioner_TransitionStatusSignature` | Call via interface | `transitioner.TransitionStatus(ctx, key, target, opts)` | Returns `(*services.TransitionResult, error)` | — |
| TC-F02-002-3 | `TestEntityTransitioner_GetNextStatusSignature` | Call via interface | `transitioner.GetNextStatus(ctx, key)` | Returns `(*services.NextStatusInfo, error)` | — |

**Notes**: These are primarily shape/contract tests. Concrete adapter behavior for `run.go` is covered in REQ-F02-011 tests.

---

### REQ-F02-003: PlaceholderGenerator Interface

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-003-1 | `TestPlaceholderGenerator_InterfaceShape` | Compile-time check | `var _ PlaceholderGenerator = (*MockPlaceholderGen)(nil)` | Compiles | Mock satisfies interface |
| TC-F02-003-2 | `TestPlaceholderGenerator_ReturnsMap` | Mock returns populated map | `GeneratePlaceholders(ctx, "E07-F01-001")` | Returns `map[string]string` with task variables | Non-nil map |
| TC-F02-003-3 | `TestPlaceholderGenerator_PropagatesError` | Mock returns error | `GeneratePlaceholders(ctx, "MISSING")` returns error | Error propagated to caller | nil map + error |

---

### REQ-F02-004: Run Loop Core Logic

#### Happy Path — Single Stage

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-1 | `TestRunController_HappyPath_SingleStage` | Entity at `in_development`, action=`spawn_agent`, dispatcher returns exit 0, next status=`ready_for_code_review` (terminal), transitioner succeeds | `Run(ctx, "E07-F01-001", RunOptions{})` | `RunResult{Outcome: "completed", StagesCompleted: 1, FinalStatus: "ready_for_code_review"}` | Exactly one `Dispatch` call, exactly one `TransitionStatus` call |

#### Happy Path — Multi-Stage

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-2 | `TestRunController_HappyPath_MultiStage` | 3 statuses configured (spawn_agent each), dispatcher always returns exit 0, 3rd status is terminal | `Run(ctx, "E07-F01-001", RunOptions{})` | `RunResult{Outcome: "completed", StagesCompleted: 3}` | Verify 3 `Dispatch` calls, 3 `TransitionStatus` calls in correct order |
| TC-F02-004-3 | `TestRunController_StageLog_Populated` | Single-stage run (spawn_agent) | `Run(ctx, key, RunOptions{})` | `RunResult.Stages[0]` contains: `Status`, `Action="spawn_agent"`, `AgentType`, `Provider`, `Duration >= 0`, `ExitCode=0` | All StageLog fields populated |

#### Already Terminal

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-4 | `TestRunController_AlreadyTerminal` | `GetNextStatus` returns `IsTerminal: true` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "already_terminal", StagesCompleted: 0}` | Zero dispatches, zero transitions |

#### Pause Action

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-5 | `TestRunController_PauseAction` | Action type `"pause"` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "paused", FinalStatus: currentStatus}` | No dispatch, no transition |
| TC-F02-004-6 | `TestRunController_WaitForTriageAction` | Action type `"wait_for_triage"` | `Run(ctx, key, RunOptions{})` | Same as pause: `RunResult{Outcome: "paused"}` | Same behavior as pause |
| TC-F02-004-7 | `TestRunController_CheckOrResumeAction` | Action type `"check_or_resume"` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "paused"}` | Same behavior as pause |

#### Advance Status Action

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-8 | `TestRunController_AdvanceStatusAction` | Action type `"advance_status"`, next status is terminal | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "completed"}` with `TransitionStatus` called, no `Dispatch` called | Verify zero dispatcher calls |

#### Archive Action

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-9 | `TestRunController_ArchiveAction` | Action type `"archive"` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "completed"}` with terminal-style stop | No dispatch |

#### No Action Configured

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-10 | `TestRunController_NoActionForStatus` | `GetStatusActionPopulated` returns `nil, nil` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "no_action", FinalStatus: currentStatus}` | Distinct from pause |

#### Agent Failure

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-11 | `TestRunController_AgentFailure_ExitCode1` | Dispatcher returns `exit 1` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "failed", Error: <non-empty>}`, `AgentFailedError` in error | No status advancement |
| TC-F02-004-12 | `TestRunController_AgentFailure_ExitCode2` | Dispatcher returns `exit 2` | Same | Same outcome, exit code preserved in error | — |
| TC-F02-004-13 | `TestRunController_AgentFailure_NoStatusAdvancement` | Dispatcher returns exit 1, `TransitionStatus` mock would track calls | `Run(ctx, key, RunOptions{})` | `TransitionStatus` called zero times | Critical gate invariant |

#### ToolNotFoundError

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-14 | `TestRunController_ToolNotFound` | Dispatcher returns `&ToolNotFoundError{Tool: "claude"}` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "failed"}`, error wraps `ToolNotFoundError` | `errors.As` extractable |

#### Transition Error

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-004-15 | `TestRunController_TransitionError` | Dispatcher returns exit 0, `TransitionStatus` returns error | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "failed", Error: <non-empty>}` | Loop stopped on transition failure |

---

### REQ-F02-005: RunOptions Data Type

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-005-1 | `TestRunOptions_Fields` | Struct literal | `RunOptions{DryRun: true, Verbose: false, WorkingDir: "/tmp"}` | All fields readable, correct types | — |
| TC-F02-005-2 | `TestRunOptions_ZeroValues` | Zero-value struct | `RunOptions{}` | `DryRun=false`, `Verbose=false`, `WorkingDir=""` | Zero-value is valid default |

---

### REQ-F02-006: RunResult Data Type

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-006-1 | `TestRunResult_Fields` | Struct literal | All fields populated | All readable with correct types | — |
| TC-F02-006-2 | `TestRunResult_JSONSerializable` | `json.Marshal(result)` | Fully populated `RunResult` | Marshals and unmarshals without error, fields preserved | — |
| TC-F02-006-3 | `TestRunResult_OutcomeValues` | Table-driven | `"completed"`, `"paused"`, `"failed"`, `"already_terminal"`, `"no_action"` | All are valid strings (documentation check — spec lists exactly these 5) | No other outcome strings expected |
| TC-F02-006-4 | `TestRunResult_TotalDuration` | After multi-stage run | `RunResult.TotalDuration` | `>= 0`, reflects sum of stage durations | Non-negative |

---

### REQ-F02-007: StageLog Data Type

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-007-1 | `TestStageLog_Fields` | Struct literal | All fields populated | All readable with correct types | — |
| TC-F02-007-2 | `TestStageLog_JSONSerializable` | `json.Marshal(stageLog)` | Populated `StageLog` | Marshals and unmarshals without error | — |
| TC-F02-007-3 | `TestStageLog_ExitCode_NonAgentActions` | `advance_status` action stage | `StageLog.ExitCode` | `0` for non-agent actions | Spec states `0 for non-agent actions` |
| TC-F02-007-4 | `TestStageLog_Duration_NonNegative` | Any completed stage | `StageLog.Duration` | `>= 0` | Zero duration is valid |

---

### REQ-F02-008: Dispatcher Selection by Provider

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-008-1 | `TestRunController_DispatcherSelection_DefaultProvider` | `dispatchers = {"": mockDispatcher}`, action provider `""` | `Run(ctx, key, RunOptions{})` | `mockDispatcher.Dispatch` called | Empty string key lookup |
| TC-F02-008-2 | `TestRunController_DispatcherSelection_AnthropicProvider` | `dispatchers = {"anthropic": mockDispatcher}`, action provider `"anthropic"` | `Run(ctx, key, RunOptions{})` | `mockDispatcher.Dispatch` called | Named provider key |
| TC-F02-008-3 | `TestRunController_DispatcherSelection_UnknownProvider` | `dispatchers = {"": mockDispatcher}`, action provider `"openai"` | `Run(ctx, key, RunOptions{})` | `RunResult{Outcome: "failed"}`, error message contains `"openai"` and lists available providers | Error message descriptive |
| TC-F02-008-4 | `TestRunController_DispatcherSelection_EmptyDispatchers` | `dispatchers = {}`, any provider | `Run(ctx, key, RunOptions{})` | Error returned with descriptive message | Empty map edge case |

---

### REQ-F02-009: DispatchInput Construction from PopulatedAction

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-009-1 | `TestRunController_DispatchInputConstruction` | Capture `DispatchInput` in mock dispatcher; entity key `"E07-F01-001"`, status `"in_development"`, action with instruction, agent_type, model, provider | `Run(ctx, "E07-F01-001", RunOptions{WorkingDir: "/tmp"})` | `input.Instruction` = action instruction; `input.WorkingDir` = `"/tmp"`; `input.EntityKey` = `"E07-F01-001"`; `input.EntityType` = `"task"`; `input.Status` = `"in_development"`; `input.AgentType` = from action; `input.Model` = from action | All 7 fields verified |
| TC-F02-009-2 | `TestRunController_DispatchInput_EmptyWorkingDir` | `RunOptions{WorkingDir: ""}` | `Run(ctx, key, RunOptions{})` | `input.WorkingDir` = `""` | Zero-value passed through |

---

### REQ-F02-010: `shark run` Cobra Command

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-010-1 | `TestRunCommand_Registered` | `cli.RootCmd.Commands()` inspection | n/a | `"run"` is in root command list | Registered via `init()` |
| TC-F02-010-2 | `TestRunCommand_Args_TaskKey` | `ParseGetArgs(["E07-F01-001"])` | `E07-F01-001` | Detected as `"task"` entity type | — |
| TC-F02-010-3 | `TestRunCommand_Args_FeatureKey` | `ParseGetArgs(["E07-F01"])` | `E07-F01` | Detected as `"feature"` entity type | — |
| TC-F02-010-4 | `TestRunCommand_Args_EpicKey` | `ParseGetArgs(["E07"])` | `E07` | Detected as `"epic"` entity type | — |
| TC-F02-010-5 | `TestRunCommand_Flags_DryRun` | Parse `--dry-run` flag | `["E07-F01-001", "--dry-run"]` | `RunOptions.DryRun = true` | — |
| TC-F02-010-6 | `TestRunCommand_Flags_Verbose` | Parse `--verbose` flag | `["E07-F01-001", "--verbose"]` | `RunOptions.Verbose = true` | — |
| TC-F02-010-7 | `TestRunCommand_Flags_Workdir` | Parse `--workdir` flag | `["E07-F01-001", "--workdir", "/tmp/work"]` | `RunOptions.WorkingDir = "/tmp/work"` | — |
| TC-F02-010-8 | `TestRunCommand_JSONOutput` | Mock controller returns `RunResult`; `--json` flag set | `Run(ctx, key, RunOptions{})` | JSON-marshaled `RunResult` written to stdout | Machine-readable output |
| TC-F02-010-9 | `TestRunCommand_HumanOutput_Summary` | Mock controller returns success result | No `--json` flag | Summary line printed (final status, stages completed, total duration) | Human-readable output |
| TC-F02-010-10 | `TestRunCommand_ExitCode_AgentFailed` | Controller returns `AgentFailedError` | `Run(cmd, ["E07-F01-001"])` | Exit code 2 translation | `AgentFailedError` → exit 2 |
| TC-F02-010-11 | `TestRunCommand_ExitCode_ToolNotFound` | Controller returns `ToolNotFoundError` | `Run(cmd, ["E07-F01-001"])` | Exit code 2 translation | `ToolNotFoundError` → exit 2 |
| TC-F02-010-12 | `TestRunCommand_ExitCode_EntityNotFound` | Controller returns not-found error | `Run(cmd, ["E07-F01-999"])` | Exit code 1 translation | Not-found → exit 1 |
| TC-F02-010-13 | `TestRunCommand_MissingArg` | No args | `cobra.Command.Execute()` with no args | Error: usage message or `cobra.ExactArgs(1)` failure | Required positional arg |

---

### REQ-F02-011: Controller Wiring in run.go

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-011-1 | `TestEntityTransitionerAdapter_Task` | Adapter dispatches to `TaskService.TransitionStatus` | Task key, mock `TaskService` | `TaskService.TransitionStatus` called with correct key and target | Correct service routing |
| TC-F02-011-2 | `TestEntityTransitionerAdapter_Feature` | Adapter dispatches to `FeatureService.TransitionStatus` | Feature key | `FeatureService.TransitionStatus` called | — |
| TC-F02-011-3 | `TestEntityTransitionerAdapter_Epic` | Adapter dispatches to `EpicService.TransitionStatus` | Epic key | `EpicService.TransitionStatus` called | — |
| TC-F02-011-4 | `TestEntityTransitionerAdapter_Bug_AdvanceWrapped` | Bug adapter wraps `AdvanceBugStatus`, ignores `targetStatus` | Bug key `"B001"` | `AdvanceBugStatus` called; result wrapped into `TransitionResult` | Bug ignores targetStatus param |
| TC-F02-011-5 | `TestEntityTransitionerAdapter_ChangeCard_AdvanceWrapped` | ChangeCard adapter wraps `AdvanceChangeCardStatus` | `"CC-001"` | `AdvanceChangeCardStatus` called; wrapped into `TransitionResult` | — |
| TC-F02-011-6 | `TestPlaceholderGeneratorAdapter_Task` | Adapter retrieves task via `TaskService.GetTask` then calls `config.TaskPlaceholders` | Task key | Returns map with task-specific keys (`task_key`, `task_title`, etc.) | Correct entity retrieval |
| TC-F02-011-7 | `TestPlaceholderGeneratorAdapter_EntityNotFound` | `TaskService.GetTask` returns not-found error | Non-existent task key | Error propagated | Error path |
| TC-F02-011-8 | `TestDispatchersMap_DefaultProviderEntry` | Inspect wired `dispatchers` map in `buildRunController` | n/a | Map has `""` key mapped to `*ClaudeDispatcher` | Wire-up validation |
| TC-F02-011-9 | `TestDispatchersMap_AnthropicProviderEntry` | Inspect wired `dispatchers` map | n/a | Map has `"anthropic"` key mapped to `*ClaudeDispatcher` | Wire-up validation |

---

### REQ-F02-012: GetActionService() Global Accessor

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-012-1 | `TestGetActionService_InterfaceSatisfied` | Compile-time check | `var _ config.ActionService = ...` | Returned type satisfies `config.ActionService` | Interface shape |
| TC-F02-012-2 | `TestGetActionService_SyncOnce_SameInstance` | Call `GetActionService` twice with same ctx | Two calls | Second call returns same instance (pointer equality or equivalent) | `sync.Once` caching |
| TC-F02-012-3 | `TestGetActionService_ErrorOnMissingConfig` | `FindProjectRoot()` returns path with no `.sharkconfig.json` | Call `GetActionService(ctx)` | Returns `(nil, error)` | Config-not-found path |
| TC-F02-012-4 | `TestGetActionService_MatchesGetNoteService_Pattern` | Code review check (not automated) | Review `services_global.go` | Same `sync.Once` + error return pattern as `GetNoteService` | Pattern consistency |

---

### REQ-F02-013: Context Cancellation Support

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-013-1 | `TestRunController_ContextCancellation_BeforeLoop` | Cancel ctx before calling `Run` | Cancelled ctx + `Run(ctx, key, RunOptions{})` | Returns quickly with `Outcome: "failed"` or context error | Pre-cancelled context |
| TC-F02-013-2 | `TestRunController_ContextCancellation_BetweenStages` | In multi-stage mock, cancel ctx after first dispatch completes | Cancel via goroutine after first `Dispatch` call | Partial `RunResult`: `StagesCompleted: 1`, `Outcome: "failed"` | Loop respects cancellation at iteration boundary |
| TC-F02-013-3 | `TestRunController_ContextCancellation_NoStatusAfterCancel` | Track `TransitionStatus` calls with cancellation mid-loop | Cancel after dispatch 1 succeeds | `TransitionStatus` called for stage 1 only; no advancement for cancelled stages | Core invariant: no advancement after cancel |
| TC-F02-013-4 | `TestRunController_ContextCancellation_AgentKilled` | Real subprocess (using `helperCommand` pattern from `claude_dispatcher_test.go`) sleeping 30s; cancel context | `Run(ctx, key, RunOptions{})` | Dispatch returns error promptly (subprocess killed by `exec.CommandContext`) | Integration-level validation of kill behavior |

---

### REQ-F02-014: Dry Run Mode

| TC ID | Test Name | Setup | Input | Expected Outcome | Edge Cases |
|-------|-----------|-------|-------|------------------|------------|
| TC-F02-014-1 | `TestRunController_DryRun_NoDispatch` | Track `Dispatch` calls; single `spawn_agent` stage | `Run(ctx, key, RunOptions{DryRun: true})` | `Dispatch` called zero times | Core dry-run invariant |
| TC-F02-014-2 | `TestRunController_DryRun_NoTransition` | Track `TransitionStatus` calls | `Run(ctx, key, RunOptions{DryRun: true})` | `TransitionStatus` called zero times | No real state change |
| TC-F02-014-3 | `TestRunController_DryRun_SimulatesAdvancement` | 3-stage workflow configured | `Run(ctx, key, RunOptions{DryRun: true})` | `RunResult.StagesCompleted == 3`, `Outcome: "completed"` | Simulated advancement via `GetNextStatus` |
| TC-F02-014-4 | `TestRunController_DryRun_StagesListed` | Single stage | `Run(ctx, key, RunOptions{DryRun: true})` | `RunResult.Stages` has one entry describing the planned action | Dry-run stages logged |
| TC-F02-014-5 | `TestRunController_DryRun_PrintsActionSummary` | Capture stderr output | `Run(ctx, key, RunOptions{DryRun: true})` | Stderr contains: status, action type, agent type, provider, instruction prefix (max 200 chars) | Output format verification |

---

## 3. Integration Scenarios

### INT-1: Full Loop with Multiple Mock Stages

**Scope**: Validates `controller.go` loop logic, action reading, dispatcher selection, and terminal detection working together without real dependencies.

**Setup**:
```
Status sequence: in_development -> ready_for_code_review -> completed (terminal)
Actions:
  in_development:       spawn_agent (provider="", agent_type="developer")
  ready_for_code_review: spawn_agent (provider="anthropic", agent_type="tech_lead")
Dispatchers:
  "":          mockDispatcher → exit 0
  "anthropic": mockDispatcher → exit 0
WorkflowSvc: IsTerminalStatus("completed") = true
```

**What to verify**:
1. Two `Dispatch` calls, in correct order with correct dispatcher per provider
2. Two `TransitionStatus` calls with correct target statuses
3. `RunResult.StagesCompleted == 2`
4. `RunResult.FinalStatus == "completed"`
5. `RunResult.Outcome == "completed"`
6. `RunResult.Stages[0].Provider == ""`, `RunResult.Stages[1].Provider == "anthropic"`
7. `RunResult.TotalDuration >= RunResult.Stages[0].Duration + RunResult.Stages[1].Duration` (approximately)

**Test function**: `TestRunController_Integration_MultiStageMultiProvider`

---

### INT-2: Loop Stops at First Failure Preserving Prior Stage Logs

**Scope**: Validates that stage logs from successful stages are preserved even when a later stage fails.

**Setup**: 3-stage workflow. Stage 1 and 2 succeed (exit 0). Stage 3 dispatcher returns exit 1.

**What to verify**:
1. `RunResult.StagesCompleted == 2`
2. `RunResult.Stages` has 3 entries (2 successful + 1 failed)
3. `RunResult.Stages[2].ExitCode == 1`
4. `RunResult.Outcome == "failed"`
5. No `TransitionStatus` call for stage 3 (gate enforced)

**Test function**: `TestRunController_Integration_FailurePreservesLog`

---

### INT-3: Advance Status Mid-Loop (Mixed Action Types)

**Scope**: Validates loop handles a mix of `spawn_agent` and `advance_status` actions in the same run.

**Setup**: 3 statuses: `draft` (action=`advance_status`), `in_development` (action=`spawn_agent`, exit 0), `completed` (terminal).

**What to verify**:
1. Zero `Dispatch` calls for `draft` stage
2. One `Dispatch` call for `in_development` stage
3. Two `TransitionStatus` calls total (one for `advance_status`, one after dispatch)
4. `RunResult.StagesCompleted == 2`
5. `RunResult.Stages[0].Action == "advance_status"`, `ExitCode == 0`

**Test function**: `TestRunController_Integration_MixedActionTypes`

---

### INT-4: Dry Run Simulates Full Workflow Without Side Effects

**Scope**: Validates dry-run mode simulates all stages correctly and produces no real changes.

**Setup**: 3-stage workflow with `spawn_agent` at each. `DryRun: true`.

**What to verify**:
1. Zero `Dispatch` calls
2. Zero `TransitionStatus` calls
3. `GetNextStatus` called once per stage (for simulation)
4. `RunResult.Outcome == "completed"` with `StagesCompleted == 3`
5. All 3 stages appear in `RunResult.Stages`

**Test function**: `TestRunController_Integration_DryRunFullWorkflow`

---

### INT-5: `shark run` Command End-to-End with Mock Controller

**Scope**: Validates CLI command parsing, controller invocation, and output formatting work together.

**Setup**: Mock controller that returns a predefined `RunResult`. Test `run.go` command execution.

**What to verify** (human output):
1. Stage-by-stage progress printed to stderr/stdout
2. Final summary line printed
3. Exit code 0 on success

**What to verify** (`--json` output):
1. `RunResult` marshaled to JSON
2. All fields present in JSON output
3. Exit code 0

**What to verify** (failure):
1. `AgentFailedError` → exit code 2
2. Error message printed to stderr

**Test function**: `TestRunCommand_Integration_HumanOutput`, `TestRunCommand_Integration_JSONOutput`, `TestRunCommand_Integration_AgentFailedExitCode`

---

## 4. Test Infrastructure

### What Exists (from E22-F01)

| Infrastructure | File | Available For |
|----------------|------|---------------|
| `AgentDispatcher` interface | `internal/runner/dispatcher.go` | Mock satisfaction in controller tests |
| `DispatchInput` / `DispatchResult` types | `internal/runner/dispatcher.go` | Direct use in controller tests |
| `AgentFailedError` / `ToolNotFoundError` | `internal/runner/dispatcher.go` | Error type assertions in controller tests |
| `DefaultDisallowedTools` | `internal/runner/dispatcher.go` | Referenced in wiring tests |
| `ClaudeDispatcher` (with injectable `cmdFactory` + `lookPathFunc`) | `internal/runner/claude_dispatcher.go` | Constructor validation tests |
| `helperCommand()` pattern | `internal/runner/claude_dispatcher_test.go` | Context-cancellation subprocess tests in controller_test.go |
| `recordingFactory()` / `successLookPath()` / `failedLookPath()` | `internal/runner/claude_dispatcher_test.go` | Not directly usable in controller tests (different package-level file) — must be re-defined or extracted |
| `containsString()` / `containsArg()` / `containsFlag()` / `containsConsecutive()` / `countFlag()` | `internal/runner/claude_dispatcher_test.go` | Re-usable helpers if `controller_test.go` is in `package runner` |

**Note**: Since `controller_test.go` will be in `package runner`, all helpers from `claude_dispatcher_test.go` are directly available in the same test binary. No re-definition needed.

### What Needs Creation

#### Mock Types (define in `controller_test.go`)

```go
// MockTransitioner implements EntityTransitioner
type MockTransitioner struct {
    TransitionStatusFunc func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatusFunc    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
    transitionCalls      []transitionCall // for call tracking
    nextStatusCalls      []string
}

// MockPlaceholderGen implements PlaceholderGenerator
type MockPlaceholderGen struct {
    GenerateFunc func(ctx context.Context, key string) (map[string]string, error)
}

// MockActionService implements config.ActionService
type MockActionService struct {
    GetStatusActionPopulatedFunc func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error)
}

// MockDispatcher implements AgentDispatcher
type MockDispatcher struct {
    DispatchFunc   func(ctx context.Context, input DispatchInput) (*DispatchResult, error)
    NameFunc       func() string
    dispatchCalls  []DispatchInput // for call tracking
}

// MockWorkflowService wraps *workflow.Service for terminal check
type MockWorkflowService struct {
    IsTerminalStatusFunc func(status string) bool
}
```

#### Helper: `buildTestController`

```go
// buildTestController creates a RunController with all mocked dependencies.
// Callers override specific func fields as needed.
func buildTestController(t *testing.T, opts controllerTestOpts) *RunController {
    t.Helper()
    // Returns configured controller with sensible defaults
    // (e.g., defaultMockDispatcher returns exit 0 always)
}
```

#### Standard Test Action Builders

```go
// spawnAgentAction returns a *config.PopulatedAction for spawn_agent
func spawnAgentAction(provider, agentType, instruction string) *config.PopulatedAction

// pauseAction returns a *config.PopulatedAction for pause
func pauseAction() *config.PopulatedAction

// advanceStatusAction returns a *config.PopulatedAction for advance_status
func advanceStatusAction() *config.PopulatedAction
```

#### Standard NextStatusInfo Builders

```go
// nextStatusInfo returns a *services.NextStatusInfo for a non-terminal status with one forward transition
func nextStatusInfo(currentStatus, nextStatus string) *services.NextStatusInfo

// terminalNextStatusInfo returns a *services.NextStatusInfo for a terminal status
func terminalNextStatusInfo(currentStatus string) *services.NextStatusInfo
```

### Test File Layout

```
internal/runner/
    controller.go              # Implementation
    controller_test.go         # package runner; uses all helpers above
                               # Covers REQ-F02-001 through REQ-F02-014
                               # Covers INT-1 through INT-4

internal/cli/commands/
    run.go                     # Implementation
    run_test.go                # package commands_test (or commands)
                               # Covers REQ-F02-010 partially (arg parsing, output)
                               # Covers INT-5

internal/cli/
    services_global_test.go    # GetActionService accessor tests (TC-F02-012-*)
    (or add to run_test.go)
```

### Running Tests

```bash
# Controller unit tests only (fast, no DB, no real processes)
go test -v ./internal/runner/... -run TestRunController

# Integration scenarios
go test -v ./internal/runner/... -run TestRunController_Integration

# CLI command tests
go test -v ./internal/cli/commands/... -run TestRunCommand

# Full suite (mandatory before completion)
make fmt && make lint && make test
```

---

## 5. Non-Functional Requirements Validation

### REQ-F02-NF-001: Loop Iteration Overhead < 100ms

**Approach**: Benchmark test in `controller_test.go`.

```go
func BenchmarkRunController_LoopIteration(b *testing.B) {
    // Mocked deps that return instantly
    // Measure time per iteration excluding agent execution
    // Target: < 100ms per iteration
}
```

**Acceptance**: Benchmark result `ns/op` < 100,000,000 (100ms) on development hardware.

---

### REQ-F02-NF-002: No Database Schema Changes

**Approach**: Static analysis / code review check.

**Test**: `TestNoNewMigrations` (can be a simple grep/scan test):
- Verify `internal/db/migrate.go` has no new migration functions added by this feature.
- Verify `CurrentSchemaVersion` unchanged.
- No new SQL table/column/index definitions in `controller.go` or `run.go`.

---

### REQ-F02-NF-003: Idempotent Re-run

**Test**: `TestRunController_Idempotent_ResumeFromCurrentStatus`

**Setup**: Entity is already at status `in_code_review` (partially through workflow). `GetNextStatus` returns `in_code_review` as non-terminal with one transition forward.

**What to verify**:
1. `Run(ctx, key, RunOptions{})` dispatches agent for `in_code_review` (not for prior stages)
2. `StagesCompleted` reflects only stages executed in this run
3. No "restart from beginning" logic exists

---

### REQ-F02-NF-004: Testability (All Dependencies Are Interfaces)

**Approach**: Compile-time satisfaction checks in `controller_test.go`.

```go
var _ EntityTransitioner = (*MockTransitioner)(nil)
var _ PlaceholderGenerator = (*MockPlaceholderGen)(nil)
var _ AgentDispatcher = (*MockDispatcher)(nil)
// config.ActionService satisfaction already tested via config package tests
```

**Acceptance**: All compile-time checks pass; no real DB or agent subprocesses needed for any `RunController` unit test.

---

## 6. Edge Cases and Boundary Conditions

| ID | Edge Case | Test Coverage |
|----|-----------|---------------|
| EC-1 | Empty `dispatchers` map | TC-F02-008-4 |
| EC-2 | `AvailableTransitions` is empty slice in `NextStatusInfo` | Add `TestRunController_NoAvailableTransitions`: controller handles gracefully (error or pause) |
| EC-3 | `GetNextStatus` returns `IsTerminal=true` on initial call | TC-F02-004-4 |
| EC-4 | `WorkingDir` with spaces or special characters | TC-F02-009-2 extended; mirrors `TestClaudeDispatcher_WorkingDir_WithSpaces` |
| EC-5 | Instruction truncated to 200 chars in dry-run print | TC-F02-014-5: instruction exactly 200 chars, 201 chars, 0 chars |
| EC-6 | `StagesCompleted` is 0 when already terminal | TC-F02-004-4 |
| EC-7 | `TotalDuration` on zero-stage run | Verify `>= 0` for `already_terminal` outcome |
| EC-8 | `PopulatedAction.Provider` with whitespace | `TestRunController_DispatcherSelection_ProviderWithSpaces`: `"  "` → treated as unknown provider |
| EC-9 | Loop iteration check `ctx.Done()` before first action fetch | TC-F02-013-1 (pre-cancelled context) |
| EC-10 | Bug/ChangeCard `targetStatus` parameter ignored | TC-F02-011-4, TC-F02-011-5 |

---

## 7. Exit Gate Verification

This test plan satisfies the exit gate criteria:

- **Every AC in spec.md has at least one test case**: Verified in Section 2. All 14 REQs (F02-001 through F02-014) are covered.
- **Edge cases identified**: Section 6 documents 10 edge cases with test assignments.
- **Integration scenarios cover boundaries**: Section 3 covers multi-provider dispatch, failure mid-run, mixed action types, dry-run without side effects, and CLI end-to-end.
- **Test patterns reference existing infrastructure**: All mock patterns reference `dispatcher_test.go` (function-field mocks) and `claude_dispatcher_test.go` (`helperCommand`, `recordingFactory`, arg-inspection helpers). All tests are in `package runner` to leverage shared test helpers.

---

*Test plan complete: 2026-03-21*
