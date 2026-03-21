# E22-F02 Feature Research Report: Run Loop Controller

**Feature**: E22-F02 - Run Loop Controller
**Epic**: E22 - External Orchestration Runner
**Date**: 2026-03-21
**Researcher**: Researcher Agent

---

## Executive Summary

E22-F02 implements the `RunController` — the orchestration loop that reads entity state, reads orchestrator actions, dispatches to agents (via the `AgentDispatcher` interface from E22-F01), gates status advancement on exit code 0, and loops until terminal. The controller is a new `internal/runner/controller.go` file plus a new `internal/cli/commands/run.go` Cobra command. Almost all its dependencies already exist as tested, production code: entity status is read via `EntityService.GetNextStatus()`, orchestrator actions via `config.ActionService.GetStatusActionPopulated()`, status advancement via per-entity-type `TransitionStatus()` methods, and entity type detection via `ParseGetArgs()`. The only genuinely new code is the loop logic itself (~150 lines), the Cobra command registration (~60 lines), a `GetActionService()` global accessor extension (~15 lines), and thin interfaces for testability (`EntityTransitioner`, `EntityGetter`).

---

## 1. Existing Implementations with File Paths

### 1.1 AgentDispatcher Interface (E22-F01, DONE)

**This is the direct dependency of E22-F02.** E22-F01 has been implemented on the current branch.

| File | What Exists | Used By F02 |
|------|-------------|-------------|
| `internal/runner/dispatcher.go` | `AgentDispatcher` interface, `DispatchInput`, `DispatchResult`, `ToolNotFoundError`, `AgentFailedError`, `DefaultDisallowedTools` | Controller calls `dispatcher.Dispatch(ctx, input)` per stage |
| `internal/runner/claude_dispatcher.go` | `ClaudeDispatcher` implementing `AgentDispatcher` via `os/exec` with all required flags | Wired into controller's `dispatchers` map under `"anthropic"` and `""` keys |

Key details the controller needs to know:
- `DispatchInput.Instruction` — the rendered instruction string
- `DispatchInput.Model` — optional model override, comes from `PopulatedAction.Model`
- `DispatchResult.ExitCode` — 0 = advance, non-zero = stop
- `AgentFailedError` — returned when exit code != 0; has `ExitCode`, `Stdout`, `Stderr`, `Command`
- `ToolNotFoundError` — returned when CLI binary not found on PATH

### 1.2 Status Transition Engine

**Primary engine for advancing status. Used as-is, no changes needed.**

| File | What Exists | How F02 Uses It |
|------|-------------|-----------------|
| `internal/services/entity_service.go` | `EntityService.TransitionStatus(ctx, repo, entityType, key, targetStatus, opts, features, resolveActionFn)` | Called after each successful agent dispatch to advance to next status. Also: `EntityService.GetNextStatus()` to read available transitions. |
| `internal/services/transition_types.go` | `TransitionOptions{Force, Reason, DocumentPath, Agent}`, `TransitionResult{EntityType, EntityKey, ToStatus, OrchestratorAction, ...}`, `NextStatusInfo{CurrentStatus, AvailableTransitions, IsTerminal}` | Controller reads `TransitionResult.OrchestratorAction` for next dispatch. `NextStatusInfo.IsTerminal` controls loop exit. |
| `internal/services/task_service.go` | `TaskService.TransitionStatus(ctx, key, targetStatus, opts)` | Used when entity is a task |
| `internal/services/feature_service.go` | `FeatureService.TransitionStatus(ctx, key, targetStatus, opts)` | Used when entity is a feature |
| `internal/services/epic_service.go` | `EpicService.TransitionStatus(ctx, key, targetStatus, opts)` | Used when entity is an epic |
| `internal/services/bug_service.go` | `BugService.AdvanceBugStatus(ctx, key)` | Used when entity is a bug (uses native advance method) |
| `internal/services/change_card_service.go` | `ChangeCardService.AdvanceChangeCardStatus(ctx, key)` | Used when entity is a change card |

**Critical observation**: `EntityService.TransitionStatus()` at line 168-252 of `internal/services/entity_service.go` already:
1. Gets entity by key
2. Validates the transition
3. Updates status in database
4. Records entity history
5. Creates rejection notes
6. Resolves orchestrator action for the NEW status
7. Returns `TransitionResult.OrchestratorAction` — this is what the loop reads for the NEXT iteration

This means the controller does NOT need to call `ActionService.GetStatusActionPopulated()` after advancing — the `TransitionResult` already contains the populated action for the new status. **This simplifies the loop significantly.**

### 1.3 Workflow Service (for Terminal Detection and Next Status)

| File | What Exists | How F02 Uses It |
|------|-------------|-----------------|
| `internal/workflow/service.go` | `IsTerminalStatus(status string) bool` (line 135), `GetValidTransitions(currentStatus string) []string` (line 249), `GetTransitionInfo(currentStatus string) []TransitionInfo` (line 260) | `IsTerminalStatus()` controls loop exit. `GetValidTransitions()` provides the default forward target (first entry = happy path). |

**Already accessed via**: `entitySvc.GetWorkflowService()` — no additional dependency needed.

### 1.4 Orchestrator Action Service (for Initial Read and Entity Getter)

| File | What Exists | How F02 Uses It |
|------|-------------|-----------------|
| `internal/config/action_service.go` | `ActionService` interface, `DefaultActionService`, `GetStatusActionPopulated(ctx, status, vars) (*PopulatedAction, error)`, `GetStatusAction(ctx, status) (*OrchestratorAction, error)` | Called at start of each loop iteration for statuses that DON'T come from a `TransitionResult` (e.g., the very first iteration, `advance_status` action type, `cascade` action type) |
| `internal/config/orchestrator_action.go` | `OrchestratorAction` struct with `Action`, `AgentType`, `Provider`, `Model`, `Skills`, `InstructionTemplate`. Action constants: `ActionSpawnAgent`, `ActionPause`, `ActionAdvanceStatus`, `ActionCascade`, `ActionArchive`, `ActionWaitForTriage`, `ActionCheckOrResume` | Controller switches on `PopulatedAction.Action` to route to correct handler |
| `internal/config/manager.go:151` | `Manager.GetActionService() (ActionService, error)` — lazy factory on `*Manager` | No global CLI accessor exists yet for `ActionService`; currently created inline per-command. **F02 needs to add `GetActionService()` to `internal/cli/services_global.go`** |

**Important**: `PopulatedAction.Provider` (empty string or `"anthropic"` = Claude, `"openai"` = Codex) drives dispatcher selection. `PopulatedAction.Model` is the optional model override passed to `DispatchInput.Model`.

### 1.5 Template Variable Generation

| File | What Exists | How F02 Uses It |
|------|-------------|-----------------|
| `internal/config/template_helpers.go` | `TaskPlaceholders(task *models.Task)`, `FeaturePlaceholders(feature *models.Feature)`, `EpicPlaceholders(epic *models.Epic)`, `BugPlaceholders(bug *models.Bug)`, `ChangeCardPlaceholders(card *models.ChangeCard)`, `EntityPlaceholders(entity models.Entity)` — all returning `map[string]string` | Called to generate `vars` for `ActionService.GetStatusActionPopulated()`. `EntityPlaceholders()` is the polymorphic version that works for any entity type. |

**Key insight**: `EntityPlaceholders(entity models.Entity)` at line 93 of `template_helpers.go` works polymorphically for any entity type. The controller only needs to hold a `models.Entity` and can call this single function — no per-type switch needed for placeholder generation.

### 1.6 Entity Type Detection and Parsing

| File | What Exists | How F02 Uses It |
|------|-------------|-----------------|
| `internal/cli/commands/status_group.go` | `ParseGetArgs(args []string)` — detects entity type from key format (`E##` = epic, `E##-F##` = feature, `E##-F##-###` = task, `B###` = bug, `CC-###` = change card) | Run command uses this to parse the entity key argument |
| `internal/cli/commands/status_group.go:385-459` | `runStatusAdvance()` — the closest existing analog to the run loop. Uses `dispatchAdvance()` for bug/change_card, falls through to `GetNextStatus()` + first valid transition for epic/feature/task | Provides the per-entity-type dispatch pattern the controller should mirror |

### 1.7 Entity Registry (Polymorphic Entity Access)

| File | What Exists | How F02 Uses It |
|------|-------------|-----------------|
| `internal/services/entity_registry.go` | `EntityRegistry` with `Register(entityType, repo)` and `GetByKey(ctx, key, entityType)` for polymorphic entity lookup | Used by `GetEntityService()` — the controller's `EntityGetter` dependency. Controller gets an entity by key without a per-type switch. |
| `internal/cli/services_global.go:36-59` | `GetEntityRegistry()` — global accessor, wires all 5 entity type repositories | Available for the run controller to use for entity retrieval |

### 1.8 Global Service Accessors

| File | What Exists | Status |
|------|-------------|--------|
| `internal/cli/services_global.go:36` | `GetEntityRegistry()` | Exists, usable |
| `internal/cli/services_global.go:62` | `GetEntityService()` | Exists, usable |
| `internal/cli/services_global.go:131-160` | `buildTaskServiceDeps()`, task service wiring | Exists |
| `internal/cli/service_accessors.go` | `GetTaskService()`, `GetEpicService()`, `GetFeatureService()`, `GetBugService()` etc. | Exist |
| `internal/cli/workflow_global.go:29` | `GetWorkflowService()` | Exists, usable |
| **`GetActionService()`** | **Does NOT exist as global accessor** | **Must add to `services_global.go`** |

---

## 2. Integration Points

### 2.1 Status Transition Dispatch Pattern

The controller needs to call the correct `TransitionStatus` for each entity type. The existing `dispatchTransition()` helper in `internal/cli/commands/status_group.go` already implements this pattern (lines ~250-300). The controller should define a similar function or thin interface.

The architecture doc (Section 5.4) specifies `EntityTransitioner` as the interface:
```go
type EntityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}
```

This is implemented as a dispatch function — not a real service — since the per-entity services all have this same signature shape.

### 2.2 Entity Get for Placeholder Generation

The controller needs to get the entity model to generate template placeholders. `GetEntityRegistry()` provides `GetByKey(ctx, key, entityType)` returning `models.Entity`. Combined with `config.EntityPlaceholders(entity)`, this covers all entity types without per-type switches.

`EntityGetter` interface (from architecture doc Section 5.4):
```go
type EntityGetter interface {
    GetEntity(ctx context.Context, key string) (models.Entity, error)
}
```

The `EntityRegistry` satisfies this if wrapped in a closure or thin adapter.

### 2.3 Action Service for Initial Loop Read

On the first loop iteration, the entity status has not just been transitioned — it came from the database. The controller uses `ActionService.GetStatusActionPopulated(ctx, currentStatus, vars)` to get the action for that status.

On subsequent iterations: `TransitionResult.OrchestratorAction` already contains the `PopulatedAction` for the new status (from `EntityService.TransitionStatus()` step 9). The controller can short-circuit the `ActionService` call by reading directly from `TransitionResult.OrchestratorAction`.

**This is a minor optimization** — calling `ActionService` every iteration is also correct and simpler. For the initial implementation, calling `ActionService` every iteration avoids this optimization complexity.

### 2.4 Cobra Command Wiring

The `shark run` command follows the standard thin-wrapper pattern:

```go
// internal/cli/commands/run.go
var runCmd = &cobra.Command{
    Use:   "run <entity-key>",
    Short: "Drive an entity through its workflow stages",
    RunE:  runRun,
}

func init() {
    cli.RootCmd.AddCommand(runCmd)
    // flags: --no-worktree, --verbose, --dry-run
}

func runRun(cmd *cobra.Command, args []string) error {
    entityType, key, err := ParseGetArgs(args)
    // build controller
    result, err := controller.Run(cmd.Context(), key)
    // format output
}
```

The controller wiring uses `GetEntityService()`, `GetEntityRegistry()`, `GetWorkflowService()`, and the new `GetActionService()`.

---

## 3. Inter-Feature Dependency Map

```
E22-F01 (AgentDispatcher Interface + ClaudeDispatcher) ← COMPLETED
    |
    v
E22-F02 (Run Loop Controller) ← THIS FEATURE
    |
    +-- depends on: internal/runner/dispatcher.go (AgentDispatcher, DispatchInput, DispatchResult)
    +-- depends on: internal/runner/claude_dispatcher.go (ClaudeDispatcher)
    |
    +-- uses existing: internal/services/entity_service.go (TransitionStatus, GetNextStatus)
    +-- uses existing: internal/config/action_service.go (GetStatusActionPopulated)
    +-- uses existing: internal/config/template_helpers.go (EntityPlaceholders)
    +-- uses existing: internal/workflow/service.go (IsTerminalStatus, GetValidTransitions)
    +-- uses existing: internal/cli/commands/status_group.go pattern (entity-type dispatch)
    +-- uses existing: internal/cli/services_global.go (GetEntityService, GetEntityRegistry)
    |
    +-- adds: internal/runner/controller.go (new)
    +-- adds: internal/cli/commands/run.go (new)
    +-- adds: GetActionService() to internal/cli/services_global.go (minor extension)
    |
    v
E22-F03 (Codex Dispatcher) — optional, adds CodexDispatcher to dispatchers map
E22-F04 (Run Logging) — optional, adds RunLogger to controller
E22-F05 (Worktree Support) — optional, adds worktree lifecycle to controller
```

**Critical dependency**: `internal/runner/dispatcher.go` and `internal/runner/claude_dispatcher.go` from E22-F01 must exist before E22-F02 is specifiable. They are present on the current branch.

---

## 4. Extension-vs-New Analysis

### 4.1 Components That EXTEND Existing Code

| Component | What Exists | Extension Needed | Effort |
|-----------|-------------|-----------------|--------|
| `internal/cli/services_global.go` | All other global accessors | Add `GetActionService()` (~15 lines): `configPath, _ := cli.FindProjectRoot()` + `config.NewActionService(configPath)` | Trivial |
| Entity type dispatch | Pattern in `status_group.go:dispatchAdvance()` and `dispatchTransition()` | Define `EntityTransitioner` interface in `internal/runner/` and wire per-entity dispatches in `run.go` | Small |

### 4.2 Components That Are NEW

| Component | Why New | File | Estimated Size |
|-----------|---------|------|----------------|
| `RunController` struct | The orchestration loop — read action → dispatch agent → gate on exit code → advance → loop — is a new control flow pattern | `internal/runner/controller.go` | ~150 lines |
| `RunController` tests | Test loop with mocked `AgentDispatcher`, `EntityTransitioner`, `ActionService` | `internal/runner/controller_test.go` | ~200 lines |
| `shark run` Cobra command | New CLI entry point | `internal/cli/commands/run.go` | ~60 lines |
| `RunResult` / `RunOptions` types | Output type for the run loop (stages, duration, final status) | `internal/runner/controller.go` | Included in ~150 lines above |
| `EntityTransitioner` interface | Thin interface enabling testability | `internal/runner/controller.go` | ~10 lines |
| `EntityGetter` interface | Thin interface for entity retrieval (for placeholder generation) | `internal/runner/controller.go` | ~10 lines |

### 4.3 Minor Config Extension (If Needed)

The architecture specifies `OrchestratorAction.MaxTurns`, `OrchestratorAction.AllowedTools` fields for passing through to `DispatchInput`. These fields do NOT currently exist in `internal/config/orchestrator_action.go`. The controller can either:
- (A) Add these fields to `OrchestratorAction` and propagate through `PopulatedAction` — matches E22-F01's `Provider`/`Model` extension pattern
- (B) Defer to workflow template instructions (agents set `--max-turns` themselves via the instruction text)

Option B is the minimal approach for F02. Option A is the clean approach but slightly expands scope. **Recommend Option B for F02 to avoid another config struct change**, and document as a follow-up.

---

## 5. Recommended Implementation Approach

### 5.1 Architecture of `controller.go`

```go
// internal/runner/controller.go

// EntityTransitioner advances entity status. Defined at point of use.
// Satisfied by per-entity-type closures wrapping TaskService/EpicService/etc.
type EntityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
    GetEntity(ctx context.Context, key string) (models.Entity, error)
}

// RunController orchestrates the run loop.
type RunController struct {
    transitioner EntityTransitioner   // status transitions + entity get
    actionSvc    config.ActionService // orchestrator action reading
    dispatchers  map[string]AgentDispatcher // provider -> dispatcher
    workdir      string               // optional working directory override
    verbose      bool
}

// Run drives an entity through its workflow stages.
// Returns RunResult with stage summary, or error on failure.
func (c *RunController) Run(ctx context.Context, entityType, key string) (*RunResult, error) {
    // 1. Get entity (validate it exists)
    entity, err := c.transitioner.GetEntity(ctx, key)

    // 2. Get current status info (is it already terminal?)
    info, err := c.transitioner.GetNextStatus(ctx, key)
    if info.IsTerminal { return successResult("already terminal"), nil }

    currentStatus := info.CurrentStatus
    var stages []StageLog

    // 3. LOOP
    for {
        // 3a. Read orchestrator action for current status
        vars := config.EntityPlaceholders(entity)
        action, err := c.actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)

        // 3b. Handle nil action (no action configured)
        if action == nil {
            return pauseResult("no action configured for status", currentStatus, stages), nil
        }

        // 3c. Switch on action type
        switch action.Action {
        case config.ActionPause, config.ActionWaitForTriage, config.ActionCheckOrResume:
            return pauseResult(fmt.Sprintf("paused at status %s", currentStatus), currentStatus, stages), nil

        case config.ActionArchive:
            return terminateResult("archived", currentStatus, stages), nil

        case config.ActionAdvanceStatus:
            // Auto-advance without agent dispatch
            result, err := c.transitioner.TransitionStatus(ctx, key, "", services.TransitionOptions{})
            // update currentStatus from result, continue loop

        case config.ActionSpawnAgent:
            // Select dispatcher based on provider
            dispatcher := c.selectDispatcher(action.Provider)

            // Build DispatchInput
            input := DispatchInput{
                Instruction: action.Instruction,
                WorkingDir:  c.workdir,
                EntityKey:   key,
                EntityType:  string(entity.GetEntityType()),
                Status:      currentStatus,
                AgentType:   action.AgentType,
                Model:       action.Model,
            }

            // Dispatch agent
            dispatchResult, err := dispatcher.Dispatch(ctx, input)
            stage := StageLog{Status: currentStatus, AgentType: action.AgentType, ...}
            stages = append(stages, stage)

            // Gate: only advance on exit 0
            if dispatchResult.ExitCode != 0 {
                return failureResult(dispatchResult, stages), &AgentFailedError{...}
            }

            // Advance status
            transResult, err := c.transitioner.TransitionStatus(ctx, key, "", services.TransitionOptions{})
            currentStatus = transResult.ToStatus

            // Check terminal
            workflowSvc := ... // available via entitySvc.GetWorkflowService()
            if workflowSvc.IsTerminalStatus(currentStatus) {
                return successResult("completed", currentStatus, stages), nil
            }
        }
    }
}
```

**Note on empty targetStatus**: The architecture doc (ADR-004) specifies using `transitions[0]` as the default forward target, matching `runStatusAdvance()`. The controller can either pass the target status explicitly (getting it from `GetValidTransitions()`) or pass empty string and let `EntityService.ValidateAndNormalize()` auto-select — **check if the service supports empty-string auto-select**. Looking at `entity_service.go:257`, `ValidateAndNormalize()` does not auto-select; it validates the provided target. The controller must call `GetValidTransitions()` and pick `[0]`. This is accessed via `transitioner.GetNextStatus()` which returns `AvailableTransitions[0].TargetStatus`.

### 5.2 Entity Transitioner Implementation

The `EntityTransitioner` interface is satisfied by an adapter wrapping per-entity services. In `run.go`:

```go
func buildEntityTransitioner(entityType string, key string) runner.EntityTransitioner {
    // Returns a struct that dispatches to the correct service based on entityType
    // Captures entityType at wiring time (set once per run command invocation)
}
```

Alternatively, define `EntityTransitioner` to accept the entity type as a parameter. The simpler approach is a constructor-time capture.

### 5.3 IsTerminalStatus Access

The controller needs `workflow.Service.IsTerminalStatus()`. This is accessed via:
- `GetEntityService().GetWorkflowService()` — already accessible in CLI context
- Or pass `*workflow.Service` directly to `RunController` constructor

The architecture doc's `RunController` struct includes `workflowSvc *workflow.Service`. Inject it directly. This is cleaner than going through `EntityService`.

### 5.4 GetActionService() Addition

Add to `internal/cli/services_global.go`:

```go
var (
    globalActionService config.ActionService
    actionServiceOnce   sync.Once
    actionServiceErr    error
)

func GetActionService(ctx context.Context) (config.ActionService, error) {
    actionServiceOnce.Do(func() {
        projectRoot, _ := FindProjectRoot()
        if projectRoot == "" {
            projectRoot = "."
        }
        configPath := filepath.Join(projectRoot, ".sharkconfig.json")
        svc, err := config.NewActionService(configPath)
        if err != nil {
            actionServiceErr = fmt.Errorf("failed to create action service: %w", err)
            return
        }
        globalActionService = svc
    })
    if actionServiceErr != nil {
        return nil, actionServiceErr
    }
    return globalActionService, nil
}
```

### 5.5 File Organization

```
internal/runner/
    dispatcher.go              # EXISTS (E22-F01): AgentDispatcher interface + types
    claude_dispatcher.go       # EXISTS (E22-F01): ClaudeDispatcher
    claude_dispatcher_test.go  # EXISTS (E22-F01)
    dispatcher_test.go         # EXISTS (E22-F01)
    controller.go              # NEW (E22-F02): RunController, EntityTransitioner, EntityGetter, RunResult, StageLog
    controller_test.go         # NEW (E22-F02): Tests with mocked dispatchers and transitioner

internal/cli/commands/
    run.go                     # NEW (E22-F02): shark run Cobra command

internal/cli/
    services_global.go         # EXTEND (E22-F02): Add GetActionService()
```

### 5.6 Testing Approach

The controller is tested entirely with mocked dependencies (no real database):

**Mock `EntityTransitioner`**: Returns pre-configured `NextStatusInfo` and `TransitionResult`. Test scenarios:
- Happy path: spawn_agent at each status, exit 0, loop terminates on terminal status
- Agent failure: exit non-zero at stage 2, loop stops, returns `AgentFailedError`
- Pause action: loop stops at `pause` action
- Advance_status action: auto-advances without dispatching
- Terminal on start: entity already in terminal status, no dispatch
- Tool not found: dispatcher returns `ToolNotFoundError`

**Mock `AgentDispatcher`**: Returns `DispatchResult{ExitCode: 0}` for success, non-zero for failure.

**Mock `ActionService`**: Returns pre-configured `PopulatedAction` for each status.

The `ClaudeDispatcher` integration (subprocess execution) is tested in `claude_dispatcher_test.go` (E22-F01). The controller tests do not need to invoke real subprocesses.

---

## 6. Constraints and Risks

| Constraint | Impact |
|------------|--------|
| `TransitionStatus()` requires a valid target status | Controller must always call `GetNextStatus()` to get `transitions[0]` before calling `TransitionStatus()`. Empty-string auto-advance is not supported by the current service. |
| `ActionService` has no global CLI accessor | Must add `GetActionService()` to `services_global.go`. Low-risk, follows established pattern exactly. |
| `OrchestratorAction` has no `MaxTurns`/`AllowedTools` fields | For F02, these are not passed to `DispatchInput` (both are zero/nil by default). Can be added in a follow-up feature. Templates can encode `--max-turns` in instruction text as a workaround. |
| Bug/ChangeCard use `AdvanceBugStatus()`/`AdvanceChangeCardStatus()` not generic `TransitionStatus()` | The `EntityTransitioner` interface must handle this divergence. Either: (A) use separate code paths for bug/change_card in the controller; or (B) wrap their advance methods behind the interface. Option B is cleaner — the wrapper returns a `TransitionResult` from the native advance call. |
| `IsTerminalStatus()` requires the workflow service scoped to entity level | Task-level `IsTerminalStatus()` uses a different scope than epic-level. The controller must scope the workflow service correctly via `.ForLevel()`. |

---

## 7. Summary: What Can Be Extended vs. What Is New

**Extend (zero new code, used as-is)**:
- `internal/runner/dispatcher.go` — E22-F01 AgentDispatcher interface and types
- `internal/runner/claude_dispatcher.go` — E22-F01 ClaudeDispatcher
- `internal/services/entity_service.go` — `TransitionStatus()` and `GetNextStatus()` called directly
- `internal/config/action_service.go` — `GetStatusActionPopulated()` called directly
- `internal/config/template_helpers.go` — `EntityPlaceholders()` called directly
- `internal/workflow/service.go` — `IsTerminalStatus()`, `GetValidTransitions()` called directly
- `internal/cli/commands/status_group.go` — `ParseGetArgs()` reused in `run.go`
- `internal/cli/services_global.go` — Existing accessors reused

**Extend (minor addition to existing file)**:
- `internal/cli/services_global.go` — Add `GetActionService()` (~15 lines)

**New (net new files)**:
- `internal/runner/controller.go` — RunController, interfaces, RunResult, StageLog (~150 lines)
- `internal/runner/controller_test.go` — Controller unit tests (~200 lines)
- `internal/cli/commands/run.go` — Cobra command (~60 lines)

**Total new code: ~425 lines.** This is a lean, well-constrained feature with clear dependencies on fully-implemented predecessor code.

---

## References

| File | Relevance |
|------|-----------|
| `/home/jwwel/projects/shark-task-manager/.claude/worktrees/effervescent-mapping-wreath/internal/runner/dispatcher.go` | AgentDispatcher interface — direct F02 dependency |
| `/home/jwwel/projects/shark-task-manager/.claude/worktrees/effervescent-mapping-wreath/internal/runner/claude_dispatcher.go` | ClaudeDispatcher — direct F02 dependency |
| `/home/jwwel/projects/shark-task-manager/internal/services/entity_service.go` | TransitionStatus(), GetNextStatus(), ResolveActionForStatus() |
| `/home/jwwel/projects/shark-task-manager/internal/services/transition_types.go` | TransitionOptions, TransitionResult, NextStatusInfo |
| `/home/jwwel/projects/shark-task-manager/internal/config/action_service.go` | ActionService interface, PopulatedAction, GetStatusActionPopulated() |
| `/home/jwwel/projects/shark-task-manager/internal/config/orchestrator_action.go` | OrchestratorAction, action constants (ActionSpawnAgent etc.) |
| `/home/jwwel/projects/shark-task-manager/internal/config/template_helpers.go` | EntityPlaceholders(), TaskPlaceholders(), FeaturePlaceholders() |
| `/home/jwwel/projects/shark-task-manager/internal/workflow/service.go` | IsTerminalStatus(), GetValidTransitions(), GetTransitionInfo() |
| `/home/jwwel/projects/shark-task-manager/internal/cli/services_global.go` | GetEntityService(), GetEntityRegistry(), GetWorkflowService() — existing accessors |
| `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status_group.go` | runStatusAdvance(), ParseGetArgs(), dispatchAdvance() — reference patterns |
| `/home/jwwel/projects/shark-task-manager/internal/cli/service_accessors.go` | GetTaskService(), GetEpicService(), GetFeatureService(), GetBugService() |
| `/home/jwwel/projects/shark-task-manager/docs/plan/E22-external-orchestration-runner-shark-run-subcommand/architecture.md` | Accepted ADRs and controller design |
| `/home/jwwel/projects/shark-task-manager/docs/plan/E22-external-orchestration-runner-shark-run-subcommand/requirements.md` | Must Have / Should Have requirements |
| `/home/jwwel/projects/shark-task-manager/docs/plan/E22-external-orchestration-runner-shark-run-subcommand/E22-F01-agent-dispatcher-interface-and-claude-implementati/spec.md` | E22-F01 specification — completed predecessor feature |

---

*Research complete: 2026-03-21*
