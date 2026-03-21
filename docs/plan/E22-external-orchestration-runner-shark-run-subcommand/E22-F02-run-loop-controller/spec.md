# E22-F02 Specification: Run Loop Controller

**Feature**: E22-F02 - Run Loop Controller
**Epic**: E22 - External Orchestration Runner
**Date**: 2026-03-21
**Status**: Draft

---

## 1. Purpose

This feature implements the `RunController` -- the core orchestration loop that reads entity state, reads orchestrator actions from workflow configuration, dispatches to agents via the `AgentDispatcher` interface (delivered in E22-F01), gates status advancement on exit code 0, and loops until terminal/pause/failure. It also delivers the `shark run <entity-key>` Cobra command and a `GetActionService()` global accessor.

For business context, see epic PRD Section 1 (Problem Statement) and Section 2 (Goals). For architectural decisions, see `docs/plan/E22-external-orchestration-runner-shark-run-subcommand/architecture.md` ADR-001 through ADR-006.

---

## 2. Functional Requirements

### REQ-F02-001: RunController Struct with Constructor Injection

**Traces to**: Epic ADR-006 (RunController Receives Services via Constructor Injection)

A `RunController` struct SHALL be defined in `internal/runner/controller.go` with all dependencies received via constructor injection.

**Acceptance Criteria**:
- [ ] `RunController` is defined in `internal/runner/controller.go`
- [ ] Constructor `NewRunController(opts RunControllerDeps)` accepts a deps struct containing: `EntityTransitioner`, `ActionService`, `WorkflowService`, dispatchers map, and options
- [ ] All dependencies are interfaces or well-known types, enabling mock injection for tests
- [ ] Constructor panics or returns error on nil required dependencies (transitioner, actionSvc, workflowSvc)

### REQ-F02-002: EntityTransitioner Interface

**Traces to**: Architecture doc Section 5.6

An `EntityTransitioner` interface SHALL be defined at point of use in `internal/runner/controller.go`, abstracting the per-entity-type status transition dispatch.

```go
type EntityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}
```

**Acceptance Criteria**:
- [ ] Interface defined in `internal/runner/controller.go`
- [ ] `TransitionStatus` matches the signature of `TaskService.TransitionStatus`, `FeatureService.TransitionStatus`, and `EpicService.TransitionStatus`
- [ ] `GetNextStatus` matches the signature of `TaskService.GetNextStatus`, `FeatureService.GetNextStatus`, and `EpicService.GetNextStatus`
- [ ] An adapter implementation is provided in `internal/cli/commands/run.go` that dispatches to the correct per-entity-type service based on entity type (task/feature/epic/bug/change_card)

### REQ-F02-003: PlaceholderGenerator Interface

**Traces to**: Architecture doc Section 4.3

A `PlaceholderGenerator` interface SHALL be defined to abstract template variable generation for different entity types.

```go
type PlaceholderGenerator interface {
    GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error)
}
```

**Acceptance Criteria**:
- [ ] Interface defined in `internal/runner/controller.go`
- [ ] An adapter implementation dispatches to the correct `config.TaskPlaceholders()`, `config.FeaturePlaceholders()`, `config.EpicPlaceholders()`, `config.BugPlaceholders()`, or `config.ChangeCardPlaceholders()` based on entity type
- [ ] The adapter retrieves the entity model via the appropriate service's `Get` method to pass to the placeholder function

### REQ-F02-004: Run Loop Core Logic

**Traces to**: Epic Approach Section (Architecture), Epic REQ-F-001 (Run Loop), Epic REQ-F-003 (Status Advancement Gate)

The `Run(ctx context.Context, key string, opts RunOptions) (*RunResult, error)` method SHALL implement the orchestration loop described in the epic approach.

**Acceptance Criteria**:
- [ ] Gets entity's current status via `transitioner.GetNextStatus(ctx, key)`
- [ ] Returns immediately with success if entity is already in a terminal status
- [ ] Loops: reads orchestrator action for current status via `actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)`
- [ ] Stops with a pause result when action is nil (no action configured for status)
- [ ] Handles action types: `spawn_agent`, `pause`, `advance_status`, `cascade`, `archive`, `wait_for_triage`, `check_or_resume`
- [ ] For `spawn_agent`: selects dispatcher from `dispatchers` map by `PopulatedAction.Provider` (defaults to `""` key when provider is empty), builds `DispatchInput`, calls `dispatcher.Dispatch(ctx, input)`, gates advancement on `DispatchResult.ExitCode == 0`
- [ ] For `advance_status`: calls `transitioner.TransitionStatus` to advance without agent dispatch
- [ ] For `pause`, `wait_for_triage`, `check_or_resume`: returns a pause `RunResult` with current status
- [ ] For `archive`: returns a terminal `RunResult`
- [ ] After successful agent dispatch (exit 0), determines next status from `transitioner.GetNextStatus(ctx, key)` using `AvailableTransitions[0].TargetStatus` (per ADR-004), then calls `transitioner.TransitionStatus(ctx, key, targetStatus, opts)`
- [ ] Updates `currentStatus` from `TransitionResult.ToStatus` after each successful transition
- [ ] Checks `workflowSvc.IsTerminalStatus(currentStatus)` after each transition; breaks loop on terminal
- [ ] On non-zero agent exit code: stops loop, returns failure `RunResult` with `AgentFailedError`

### REQ-F02-005: RunOptions Data Type

**Traces to**: Architecture doc Section 5.4

`RunOptions` SHALL control run loop behavior.

**Acceptance Criteria**:
- [ ] Contains `DryRun bool` -- when true, prints actions but does not dispatch agents or advance status
- [ ] Contains `Verbose bool` -- when true, prints detailed stage progress to stderr
- [ ] Contains `WorkingDir string` -- optional working directory override for agent processes

### REQ-F02-006: RunResult Data Type

**Traces to**: Architecture doc Section 5.5 (RunLogger)

`RunResult` SHALL capture the outcome of a run loop execution.

**Acceptance Criteria**:
- [ ] Contains `EntityKey string` -- the entity that was run
- [ ] Contains `FinalStatus string` -- the entity's status when the loop stopped
- [ ] Contains `StagesCompleted int` -- number of stages successfully completed
- [ ] Contains `Stages []StageLog` -- per-stage log entries
- [ ] Contains `Outcome string` -- one of: `"completed"`, `"paused"`, `"failed"`, `"already_terminal"`, `"no_action"`
- [ ] Contains `TotalDuration time.Duration` -- wall-clock time for the entire run
- [ ] Contains `Error string` -- error message if outcome is `"failed"` (empty otherwise)
- [ ] JSON-serializable with `json` struct tags

### REQ-F02-007: StageLog Data Type

**Traces to**: Architecture doc Section 5.5

`StageLog` SHALL capture per-stage execution details.

**Acceptance Criteria**:
- [ ] Contains `Status string` -- the workflow status this stage executed
- [ ] Contains `Action string` -- the action type (e.g., `"spawn_agent"`, `"advance_status"`)
- [ ] Contains `AgentType string` -- the agent type from the orchestrator action (e.g., `"developer"`)
- [ ] Contains `Provider string` -- the provider used (e.g., `"anthropic"`)
- [ ] Contains `Duration time.Duration` -- wall-clock time for this stage
- [ ] Contains `ExitCode int` -- agent exit code (0 for non-agent actions)
- [ ] JSON-serializable with `json` struct tags

### REQ-F02-008: Dispatcher Selection by Provider

**Traces to**: Epic ADR-002 (AgentDispatcher Interface), Architecture doc Section 5.4

The run controller SHALL select a dispatcher from its `dispatchers` map using the `PopulatedAction.Provider` field.

**Acceptance Criteria**:
- [ ] When `Provider` is `""` (empty), selects the dispatcher mapped to `""` key (Claude by default)
- [ ] When `Provider` is `"anthropic"`, selects the dispatcher mapped to `"anthropic"` key
- [ ] When `Provider` value has no matching dispatcher, returns an error including the provider name and available dispatcher keys
- [ ] The dispatchers map is populated at wiring time in `run.go`, not hardcoded in the controller

### REQ-F02-009: DispatchInput Construction from PopulatedAction

**Traces to**: E22-F01 REQ-F01-002 (DispatchInput Data Type)

The run controller SHALL construct a `DispatchInput` from the `PopulatedAction` and entity context.

**Acceptance Criteria**:
- [ ] `DispatchInput.Instruction` set from `PopulatedAction.Instruction`
- [ ] `DispatchInput.WorkingDir` set from `RunOptions.WorkingDir` (or empty if not provided)
- [ ] `DispatchInput.EntityKey` set from the entity key argument
- [ ] `DispatchInput.EntityType` set from the detected entity type
- [ ] `DispatchInput.Status` set from the current workflow status
- [ ] `DispatchInput.AgentType` set from `PopulatedAction.AgentType`
- [ ] `DispatchInput.Model` set from `PopulatedAction.Model`

### REQ-F02-010: `shark run` Cobra Command

**Traces to**: Architecture doc Section 4.5

A new `shark run <entity-key>` CLI command SHALL be registered as a top-level Cobra command.

**Acceptance Criteria**:
- [ ] Defined in `internal/cli/commands/run.go`
- [ ] Registered via `init()` as `cli.RootCmd.AddCommand(runCmd)`
- [ ] Uses `ParseGetArgs(args)` to detect entity type from key format
- [ ] Accepts flags: `--dry-run` (bool), `--verbose` (bool), `--workdir` (string)
- [ ] Follows thin wrapper pattern: parse args, wire controller, call `controller.Run()`, format output
- [ ] On `--json` flag: outputs `RunResult` as JSON
- [ ] On human output: prints stage-by-stage progress with final summary
- [ ] Handles errors by translating `AgentFailedError` to exit code 2, `ToolNotFoundError` to exit code 2, entity-not-found to exit code 1

### REQ-F02-011: Controller Wiring in run.go

**Traces to**: Architecture doc Section 4.5 (buildRunController)

The `run.go` command SHALL wire the `RunController` with all dependencies using existing global service accessors.

**Acceptance Criteria**:
- [ ] Builds `EntityTransitioner` adapter that dispatches to `GetTaskService().TransitionStatus`, `GetFeatureService().TransitionStatus`, or `GetEpicService().TransitionStatus` based on entity type; for bug/change_card, wraps `GetBugService().AdvanceBugStatus` / `GetChangeCardService().AdvanceChangeCardStatus` to return `TransitionResult`
- [ ] Builds `PlaceholderGenerator` adapter that dispatches to the correct `config.*Placeholders()` function based on entity type
- [ ] Gets `ActionService` via `GetActionService(ctx)` (new global accessor, REQ-F02-012)
- [ ] Gets workflow service via `GetWorkflowService()` scoped to the correct entity level via `.ForLevel()`
- [ ] Populates `dispatchers` map: `"" -> ClaudeDispatcher`, `"anthropic" -> ClaudeDispatcher` using `runner.NewClaudeDispatcher(nil)` (from E22-F01)

### REQ-F02-012: GetActionService() Global Accessor

**Traces to**: Research report Section 1.8

A `GetActionService(ctx context.Context) (config.ActionService, error)` function SHALL be added to `internal/cli/services_global.go`.

**Acceptance Criteria**:
- [ ] Uses `sync.Once` pattern matching existing global accessors in the file
- [ ] Creates `config.NewActionService(configPath)` where `configPath` is resolved via `FindProjectRoot()` + `".sharkconfig.json"`
- [ ] Returns `(config.ActionService, error)` matching the `GetNoteService` accessor signature pattern
- [ ] Caches the service instance for reuse within the same CLI invocation

### REQ-F02-013: Context Cancellation Support

**Traces to**: Epic Section 7 (Error Handling Strategy), E22-F01 REQ-F01-NF-003

The run loop SHALL respect context cancellation (e.g., SIGINT) at every iteration boundary.

**Acceptance Criteria**:
- [ ] Checks `ctx.Done()` at the top of each loop iteration
- [ ] When context is cancelled during an agent dispatch, the agent subprocess is killed (handled by `exec.CommandContext` in dispatchers)
- [ ] Returns a partial `RunResult` with `Outcome: "failed"` and the stages completed so far
- [ ] Does not advance status after a cancelled dispatch

### REQ-F02-014: Dry Run Mode

**Traces to**: Research report Section 2.4 (flags)

When `--dry-run` is set, the run loop SHALL print what it would do without actually dispatching agents or advancing status.

**Acceptance Criteria**:
- [ ] For each loop iteration, prints the status, action type, agent type, provider, and first 200 characters of instruction
- [ ] Does not call `dispatcher.Dispatch()`
- [ ] Does not call `transitioner.TransitionStatus()`
- [ ] Simulates advancement by reading `GetNextStatus()` and using `AvailableTransitions[0]` as the next status
- [ ] Returns a `RunResult` with `Outcome: "completed"` and all stages listed as dry-run

---

## 3. Non-Functional Requirements

### REQ-F02-NF-001: Loop Iteration Overhead

**Traces to**: Epic REQ-NF-001 (Run Loop Overhead)

Each loop iteration (excluding agent execution time and status transition DB calls) SHALL complete in under 100ms. This covers: reading the orchestrator action, selecting the dispatcher, constructing `DispatchInput`, and checking terminal status.

### REQ-F02-NF-002: No Database Schema Changes

**Traces to**: Epic ADR-003 (No New Database Tables)

This feature SHALL NOT introduce any new database tables, columns, views, indexes, or migrations. All run-level state is held in memory.

### REQ-F02-NF-003: Idempotent Re-run

The run loop SHALL be idempotent: if `shark run E07-F01-001` is interrupted and re-run, it resumes from the entity's current persisted database status. No cleanup is required between runs.

### REQ-F02-NF-004: Testability

The `RunController` SHALL be testable with mocked dependencies (no real database, no real agent subprocesses). All external dependencies are injected via interfaces.

---

## 4. Out of Scope

- **Codex CLI dispatcher** -- separate feature (E22-F03). The `dispatchers` map in `run.go` only includes `ClaudeDispatcher` entries for now.
- **Structured run logging to file** -- separate feature (E22-F04). `RunResult`/`StageLog` types provide the data; file-based logging is future work.
- **Git worktree management** -- separate feature (E22-F05). `RunOptions.WorkingDir` is passed through to `DispatchInput.WorkingDir` but worktree creation/cleanup is not implemented.
- **Retry logic** -- the controller does not retry failed agent dispatches. It stops on first non-zero exit code.
- **MaxTurns/AllowedTools on OrchestratorAction** -- deferred per research report Section 4.3 recommendation (option B). Templates can encode these in instruction text.
- **The `/run` Claude Code skill** -- not modified or deprecated in this feature.
- **Bug and change_card `GetNextStatus` method** -- these entity types use native advance methods (`AdvanceBugStatus`, `AdvanceChangeCardStatus`) which auto-select the next status. The `EntityTransitioner` adapter wraps these differently than task/feature/epic.

---

## 5. Architecture

### 5.1 Component Changes

#### New Files

| File | Description | Estimated Lines |
|------|-------------|-----------------|
| `internal/runner/controller.go` | `RunController`, `EntityTransitioner`, `PlaceholderGenerator`, `RunControllerDeps`, `RunOptions`, `RunResult`, `StageLog` | ~200 |
| `internal/runner/controller_test.go` | Unit tests with mocked dispatchers, transitioner, actionSvc | ~300 |
| `internal/cli/commands/run.go` | `shark run` Cobra command, entity-type adapter implementations | ~150 |

#### Modified Files

| File | Change | Reason |
|------|--------|--------|
| `internal/cli/services_global.go` | Add `GetActionService()` (~20 lines) | Run command needs ActionService; no global accessor exists |

### 5.2 Data Model Changes

**None.** No database tables, columns, views, indexes, or migrations. Per ADR-003.

### 5.3 Interface Contracts

#### EntityTransitioner (defined in `internal/runner/controller.go`)

```go
type EntityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string,
        opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}
```

Satisfied by per-entity-type adapter in `run.go`. For tasks/features/epics, delegates directly. For bugs/change_cards, wraps their native advance methods to return `TransitionResult`.

#### PlaceholderGenerator (defined in `internal/runner/controller.go`)

```go
type PlaceholderGenerator interface {
    GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error)
}
```

Satisfied by per-entity-type adapter in `run.go`. Retrieves entity via the appropriate service, then calls the correct `config.*Placeholders()` function.

#### RunControllerDeps (defined in `internal/runner/controller.go`)

```go
type RunControllerDeps struct {
    Transitioner  EntityTransitioner
    Placeholders  PlaceholderGenerator
    ActionSvc     config.ActionService
    WorkflowSvc   *workflow.Service
    Dispatchers   map[string]AgentDispatcher
}
```

#### Consumed Interfaces (all existing, unchanged)

| Interface/Type | Package | Used For |
|----------------|---------|----------|
| `config.ActionService` | `internal/config` | `GetStatusActionPopulated(ctx, status, vars)` to read orchestrator action |
| `config.PopulatedAction` | `internal/config` | Action type, provider, model, instruction from template |
| `*workflow.Service` | `internal/workflow` | `IsTerminalStatus(status)` for loop exit detection |
| `AgentDispatcher` | `internal/runner` (E22-F01) | `Dispatch(ctx, input)` to invoke agent |
| `DispatchInput` / `DispatchResult` | `internal/runner` (E22-F01) | Agent dispatch data types |
| `services.TransitionOptions` | `internal/services` | Options for status transition |
| `services.TransitionResult` | `internal/services` | Result of status transition including `OrchestratorAction` |
| `services.NextStatusInfo` | `internal/services` | Current status, available transitions, terminal flag |

### 5.4 Key Technical Decisions

#### Decision 1: PlaceholderGenerator Interface Instead of `EntityPlaceholders()` Polymorphic Function

The research report referenced `config.EntityPlaceholders(entity models.Entity)` but this function does not exist. Each entity type has its own placeholder function (`config.TaskPlaceholders`, `config.FeaturePlaceholders`, etc.). The controller uses a `PlaceholderGenerator` interface instead, with a per-entity-type adapter built in `run.go`.

**Rationale**: Avoids modifying the `config` package to add a polymorphic function. The adapter pattern keeps the controller clean and the `config` package unchanged. Follows the project's "define interfaces at point of use" convention (`.claude/rules/go/patterns.md`).

#### Decision 2: Per-Entity-Type Adapter in run.go (Not a Generic EntityService)

The research report referenced `GetEntityService()` and `GetEntityRegistry()` but neither exists. Rather than creating new polymorphic services, the `run.go` command builds thin adapter structs that dispatch to existing per-type services (`GetTaskService()`, `GetFeatureService()`, `GetEpicService()`, `GetBugService()`, `GetChangeCardService()`).

**Rationale**: Minimal change. Reuses existing global accessors exactly as they are. Adding `EntityService` or `EntityRegistry` would be a separate cross-cutting refactoring concern outside E22 scope.

#### Decision 3: Bug/ChangeCard Advance Wrapping

`BugService.AdvanceBugStatus(ctx, key)` and `ChangeCardService.AdvanceChangeCardStatus(ctx, key)` auto-select the next status and return `(*models.Bug, error)` / `(*models.ChangeCard, error)` respectively -- not `(*TransitionResult, error)`. The `EntityTransitioner` adapter wraps these by:
1. Calling the native advance method
2. Constructing a `TransitionResult` from the returned entity's new status
3. Ignoring the `targetStatus` parameter (bugs/change_cards auto-advance)

For `GetNextStatus`: bugs/change_cards use their service's existing status checking to determine if terminal. The adapter constructs a `NextStatusInfo` from the bug/change_card's current status.

**Rationale**: Avoids modifying bug/change_card services to add `TransitionStatus` and `GetNextStatus` methods matching the epic/feature/task signature. This is a wrapper, not a refactoring.

#### Decision 4: First Valid Transition as Forward Target

**Traces to**: Architecture ADR-004

After a successful agent dispatch, the controller calls `GetNextStatus()`, reads `AvailableTransitions[0].TargetStatus`, and passes it to `TransitionStatus()` as the explicit target. This matches the behavior of `runStatusAdvance()` in `status_group.go`.

**Rationale**: `TransitionStatus()` does not support empty-string auto-select. The first transition is the configured "happy path" forward step.

### 5.5 Integration with Existing Code

#### Services Consumed (all existing, no changes)

| Service | Global Accessor | Method(s) Used |
|---------|-----------------|----------------|
| `TaskService` | `cli.GetTaskService()` | `TransitionStatus()`, `GetNextStatus()`, `GetTask()` |
| `FeatureService` | `cli.GetFeatureService()` | `TransitionStatus()`, `GetNextStatus()`, `GetFeature()` |
| `EpicService` | `cli.GetEpicService()` | `TransitionStatus()`, `GetNextStatus()`, `GetEpic()` |
| `BugService` | `cli.GetBugService()` | `AdvanceBugStatus()`, `GetBug()` |
| `ChangeCardService` | `cli.GetChangeCardService()` | `AdvanceChangeCardStatus()`, `GetChangeCard()` |
| `ActionService` | `cli.GetActionService()` (NEW) | `GetStatusActionPopulated()` |
| `workflow.Service` | `cli.GetWorkflowService()` | `IsTerminalStatus()` |

#### Entity Type Detection

Reuses `ParseGetArgs(args)` from `internal/cli/commands/helpers.go` which returns `(entityType string, key string, error)`. The command string maps to entity types as established by existing commands.

#### Orchestrator Action Constants

Uses `config.ActionSpawnAgent`, `config.ActionPause`, `config.ActionAdvanceStatus`, `config.ActionCascade`, `config.ActionArchive`, `config.ActionWaitForTriage`, `config.ActionCheckOrResume` from `internal/config/orchestrator_action.go`.

#### Dispatcher Types (from E22-F01)

Uses `runner.AgentDispatcher`, `runner.DispatchInput`, `runner.DispatchResult`, `runner.AgentFailedError`, `runner.ToolNotFoundError`, `runner.NewClaudeDispatcher()` from `internal/runner/dispatcher.go` and `internal/runner/claude_dispatcher.go`.

### 5.6 File Organization

```
internal/runner/
    dispatcher.go              # EXISTS (E22-F01): AgentDispatcher interface + types
    claude_dispatcher.go       # EXISTS (E22-F01): ClaudeDispatcher
    claude_dispatcher_test.go  # EXISTS (E22-F01)
    dispatcher_test.go         # EXISTS (E22-F01)
    controller.go              # NEW: RunController, interfaces, types
    controller_test.go         # NEW: Tests with mocked dependencies

internal/cli/commands/
    run.go                     # NEW: shark run command + entity-type adapters

internal/cli/
    services_global.go         # EXTEND: +GetActionService()
```

---

## 6. Testing Strategy

### Unit Tests: `controller_test.go` (mocked dependencies, no real DB)

| Test | What It Validates |
|------|-------------------|
| `TestRunController_HappyPath_SingleStage` | Entity at status X, spawn_agent action, dispatcher returns exit 0, advances to next status, terminal -> returns success RunResult |
| `TestRunController_HappyPath_MultiStage` | Entity progresses through 3 stages (spawn_agent at each), ends at terminal status |
| `TestRunController_AlreadyTerminal` | Entity already in terminal status -> returns immediately with `Outcome: "already_terminal"`, zero stages |
| `TestRunController_AgentFailure` | Dispatcher returns exit code 1 -> loop stops, returns `Outcome: "failed"` with `AgentFailedError`, no status advancement |
| `TestRunController_PauseAction` | Action type is `pause` -> loop stops, returns `Outcome: "paused"` with current status |
| `TestRunController_WaitForTriageAction` | Action type is `wait_for_triage` -> same behavior as pause |
| `TestRunController_AdvanceStatusAction` | Action type is `advance_status` -> advances without agent dispatch, continues loop |
| `TestRunController_ArchiveAction` | Action type is `archive` -> loop stops, returns terminal result |
| `TestRunController_NoActionForStatus` | `GetStatusActionPopulated` returns nil -> loop stops with `Outcome: "no_action"` |
| `TestRunController_ToolNotFound` | Dispatcher returns `ToolNotFoundError` -> loop stops with failure |
| `TestRunController_ContextCancellation` | Context cancelled mid-loop -> returns partial RunResult with completed stages |
| `TestRunController_DispatcherSelection_DefaultProvider` | Empty provider selects `""` dispatcher entry |
| `TestRunController_DispatcherSelection_ExplicitProvider` | `"anthropic"` provider selects `"anthropic"` dispatcher entry |
| `TestRunController_DispatcherSelection_UnknownProvider` | Unknown provider returns descriptive error |
| `TestRunController_DispatchInputConstruction` | Verifies `DispatchInput` fields are populated correctly from `PopulatedAction` and entity context |
| `TestRunController_DryRun` | With `DryRun: true`, no dispatch or transition calls made, returns complete run plan |
| `TestRunController_TransitionError` | `TransitionStatus` returns error -> loop stops with failure |

### Mock Definitions (in `controller_test.go`)

```go
type MockTransitioner struct {
    TransitionStatusFunc func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatusFunc    func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

type MockPlaceholderGen struct {
    GenerateFunc func(ctx context.Context, key string) (map[string]string, error)
}

type MockActionService struct {
    GetStatusActionPopulatedFunc func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error)
}

type MockDispatcher struct {
    DispatchFunc func(ctx context.Context, input runner.DispatchInput) (*runner.DispatchResult, error)
    NameFunc     func() string
}
```

### Integration Testing

A full-loop integration test drives a mock entity through 3 statuses using mocked dispatchers (always exit 0) and mocked transitioner (returns pre-configured NextStatusInfo at each step). This validates the loop logic, action reading, dispatcher selection, and terminal detection work together without real database or real agent processes.

---

## 7. Acceptance Criteria Summary

1. `internal/runner/controller.go` exists with `RunController`, `EntityTransitioner`, `PlaceholderGenerator`, `RunControllerDeps`, `RunOptions`, `RunResult`, and `StageLog` types.
2. `RunController.Run()` implements the orchestration loop: read action -> dispatch agent -> gate on exit code -> advance -> loop until terminal/pause/failure.
3. `internal/cli/commands/run.go` exists with `shark run <entity-key>` Cobra command that follows the thin wrapper pattern.
4. `run.go` builds entity-type adapters for `EntityTransitioner` and `PlaceholderGenerator` dispatching to existing per-type services.
5. `internal/cli/services_global.go` has `GetActionService(ctx)` returning `(config.ActionService, error)` using `sync.Once` pattern.
6. Controller respects context cancellation and stops cleanly.
7. Dry-run mode prints planned actions without dispatching or transitioning.
8. All tests pass (`make fmt && make lint && make test`).
9. No database schema changes.

---

*Specification complete: 2026-03-21*
