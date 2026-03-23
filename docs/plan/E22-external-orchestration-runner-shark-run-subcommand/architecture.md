# E22 Architecture: External Orchestration Runner

**Epic**: E22 - External Orchestration Runner - shark run subcommand
**Date**: 2026-03-21
**Updated**: 2026-03-22 (simplified loop design)
**Status**: Accepted

---

## 1. Core Design Principle

The run loop is deliberately simple: **read entity state, read action from config, execute, repeat.**

```
loop:
  1. Get entity from DB → current status                (one DB call)
  2. Get orchestrator_action for status from config      (config read, no DB)
  3. Build template placeholders from entity data        (in-memory)
  4. Execute action:
     - advance_status → TransitionStatus()              (same as `shark status advance`)
     - spawn_agent   → advance to in_ → dispatch agent → on success, TransitionStatus()
     - cascade       → list children → Run() each       (parallel with --parallel=N + worktrees)
     - pause/wait    → STOP
     - archive       → STOP
  5. If terminal → STOP, else → loop with new status
```

**Critical invariant**: `TransitionStatus()` is the **exact same method** called by `shark status advance`. The run controller does not have its own transition logic — it reuses the existing service layer.

**Auto-chaining `advance_status` actions**: Statuses like `ready_for_development` have `advance_status` as their orchestrator action. The controller calls `TransitionStatus()` and loops, naturally chaining through:
```
draft [advance_status] → ready_for_development [advance_status] → in_development [spawn_agent] → ...
```

No special handling is needed for `ready_for_` → `in_` transitions — they are just `advance_status` actions that resolve through the normal loop.

---

## 2. Component Overview

### What Changes

| Component | Change Type | Description |
|-----------|-------------|-------------|
| `internal/runner/` | **NEW package** | Run loop controller, agent dispatcher interface, Claude and Codex dispatcher implementations |
| `internal/cli/commands/run.go` | **NEW file** | Cobra command registration for `shark run <entity-key>`, entity-type adapters |
| `internal/cli/services_global.go` | **EXTEND** | Add `GetActionService()` global accessor |

### What Stays Unchanged

| Component | Reason |
|-----------|--------|
| `internal/services/entity_service.go` | `TransitionStatus()` is called as-is — it's the same method `shark status advance` uses |
| `internal/config/orchestrator_action.go` | `OrchestratorAction` struct already has all required fields (`Provider`, `Model`, `AgentType`, `Skills`) |
| `internal/config/action_service.go` | `GetStatusActionPopulated()` resolves action + renders templates from config |
| `internal/workflow/service.go` | `GetValidTransitions()`, `IsTerminalStatus()` used as-is |
| `shark-templates/` | Agent instruction templates consumed as-is |
| `.sharkconfig.json` schema | No changes to workflow configuration format |
| Database schema | No new tables or columns |

### Component Diagram

```
shark run <key>
  |
  v
internal/cli/commands/run.go           (Cobra command: parse args, build adapters, call RunController)
  |
  v
internal/runner/controller.go          (RunController: orchestration loop)
  |-- reads entity status via GetNextStatus()          (same as `shark status advance`)
  |-- reads orchestrator action via ActionService      (config read, no DB)
  |-- generates template placeholders from entity      (in-memory)
  |-- dispatches to AgentDispatcher interface           (new)
  |     |-- ClaudeDispatcher --> os/exec: claude -p "..."
  |     |-- CodexDispatcher  --> os/exec: codex exec "..."
  |-- advances status via TransitionStatus()           (same as `shark status advance`)
  |-- loops until terminal/pause/failure
  |
  v
internal/runner/dispatcher.go          (AgentDispatcher interface + DispatchResult)
internal/runner/claude_dispatcher.go   (Claude CLI implementation)
internal/runner/codex_dispatcher.go    (Codex CLI implementation)
internal/runner/worktree.go            (Git worktree isolation)
```

---

## 3. Key Technical Decisions (ADRs)

### ADR-001: New `internal/runner/` Package for Run Loop

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The run loop is a new control flow pattern — it reads orchestrator actions, dispatches external processes, waits for exit codes, and advances status in a loop. No existing package does this.

**Decision**: Create a new `internal/runner/` package rather than adding to `internal/services/` or `internal/cli/commands/`.

**Rationale**:
- The run controller is neither a service (it orchestrates services) nor a command (it contains significant logic beyond parse/call/format).
- Separation allows the runner to be tested independently with mocked services and dispatchers.
- Follows the existing pattern where packages like `internal/taskcreation/`, `internal/templates/`, and `internal/view/` contain domain-specific logic that doesn't fit the service or command layer.

**Consequences**:
- (+) Clean separation of concerns; runner logic is testable in isolation.
- (+) Commands remain thin wrappers per CLAUDE.md conventions.
- (-) One more package to navigate, but the naming is self-documenting.

### ADR-002: AgentDispatcher as Go Interface with Function-Based Dispatch

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The run loop needs to invoke different CLI tools (Claude, Codex) depending on the `provider` field in the orchestrator action. Future providers may be added.

**Decision**: Define an `AgentDispatcher` interface with a single `Dispatch(ctx, DispatchInput) (*DispatchResult, error)` method. Select the dispatcher per-stage based on the `PopulatedAction.Provider` field.

**Rationale**:
- Interface-based dispatch satisfies extensibility requirements.
- Function-based selection (map of provider to dispatcher) is simpler than a plugin system.
- Each dispatcher encapsulates CLI-specific flag construction, which varies significantly between Claude (`-p`, `--disallowedTools`, `--max-turns`) and Codex (`exec`, `-m`, `-s`).
- Matches the project's preference for explicit interfaces over generic patterns.

**Consequences**:
- (+) Adding a new provider requires only implementing one interface and adding a map entry.
- (+) Each dispatcher is independently testable.
- (-) Two concrete implementations to maintain, but they are small (~60-80 lines each).

### ADR-003: No New Database Tables

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The run loop needs to track which stage it is on, retry counts, and stage durations. This state could be persisted in the database or held in memory.

**Decision**: All run-level state is held in memory. The entity's persisted database status is the only durable state. If `shark run` is interrupted, re-running it resumes from the entity's current status.

**Rationale**:
- Simplifies implementation (no migrations, no new repository methods).
- The entity's database status is already the resumption point — it reflects the last successfully completed stage.
- Run logs (stdout/file) provide the audit trail that persistent run sessions would offer.

**Consequences**:
- (+) Zero database schema changes; no migration concerns.
- (+) Re-running is idempotent — always picks up from current persisted status.
- (-) Per-run metadata (stage durations, retry counts) is lost if the process is killed. Mitigated by log output.

### ADR-004: Reuse `shark status advance` Transition Logic

**Date**: 2026-03-22
**Status**: Accepted (updated from ADR-004 v1)

**Context**: After an agent completes successfully (or for `advance_status` actions), the run loop must advance the entity to its next status.

**Decision**: The run controller calls the **exact same** `TransitionStatus()` / `GetNextStatus()` service methods that `shark status advance` uses. No separate transition logic exists in the controller.

**Rationale**:
- `shark status advance` already picks the first valid transition, validates the transition, records history, resolves the orchestrator action for the new status, and returns a `TransitionResult` with the populated action.
- Duplicating this logic in the controller would create divergence and bugs.
- The `TransitionResult.OrchestratorAction` field already carries the action for the new status, so the controller can use it for the next iteration without an extra config lookup.

**Consequences**:
- (+) Consistent behavior between `shark status advance` and `shark run` — they use identical code paths.
- (+) The controller is simpler: it delegates all transition logic to services.
- (+) Dry-run can walk the config to preview the chain without touching DB.
- (+) `advance_status` actions (e.g., `ready_for_development` → `in_development`) chain naturally through the loop.

### ADR-005: Cascade Handled Internally with Optional Parallel Dispatch

**Date**: 2026-03-23
**Status**: Accepted

**Context**: When an entity (epic or feature) reaches `active` status, its orchestrator action is `cascade` — meaning "drive all child entities forward." This could be delegated to an agent or handled by the controller.

**Decision**: The controller handles `cascade` internally by listing child entities and calling `Run()` recursively for each non-terminal child. With `--parallel=N`, children are dispatched concurrently using Go goroutines bounded by a semaphore, each in its own git worktree.

**Rationale**:
- The controller already has `Run()` — recursion is natural and keeps cascade logic tight.
- Dispatching a "tech-director" agent to do `shark run` recursively adds an unnecessary layer of indirection.
- Go's goroutines + channels are purpose-built for bounded concurrency.
- Git worktrees provide filesystem isolation for parallel agents (two agents can't safely edit the same files).
- SQLite WAL mode supports concurrent reads + serial writes; status transitions are atomic.

**Consequences**:
- (+) No agent dispatch cost for cascade — just recursive controller calls.
- (+) Parallel mode enables significant speedup for independent children (e.g., features in an epic).
- (+) Each parallel agent gets full isolation via worktree — no file conflicts.
- (+) `--parallel=1` (default) is sequential — safe fallback, no worktrees needed.
- (-) Parallel mode requires worktree support (already implemented).
- (-) Tasks with `depends_on` within a feature should respect ordering — parallel mode assumes children are independent (true for features, mostly true for tasks).

### ADR-006: Disallowed Tools for Agent Isolation

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The core security property of E22 is that agents cannot advance their own status.

**Decision**: Pass `--disallowedTools "Bash(shark status advance*)" "Bash(shark task next-status*)" "Bash(shark status set*)" "Bash(shark task set-status*)"` to Claude CLI. For Codex, rely on its sandbox mode which restricts filesystem/network access.

**Rationale**:
- This blocks the four CLI commands an agent could use to self-advance.
- The `*` wildcard ensures variants with arguments are also blocked.
- Codex's `--full-auto` sandbox mode already restricts tool access; no additional flags needed.
- This is the architectural enforcement the entire epic is built around.

**Consequences**:
- (+) Agents physically cannot advance status — the enforcement is not prompt-level.
- (+) Agents can still perform backward transitions (e.g., `shark status set ... changes_requested`) if needed for rejection workflows.
- (-) Depends on Claude CLI `--disallowedTools` flag stability. Mitigated by isolating flag construction to the dispatcher implementation.

### ADR-007: RunController Receives Services via Constructor Injection

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The run controller needs access to entity services (for status transition), the action service (for orchestrator actions), the workflow service (for valid transitions), and agent dispatchers.

**Decision**: Use constructor injection matching the existing service pattern in `internal/services/`.

**Rationale**:
- Follows the established dependency injection pattern.
- Constructor receives interfaces, enabling mock injection for tests.
- The CLI command wires dependencies via global accessors, matching existing commands.

**Consequences**:
- (+) Testable with mocked services and dispatchers.
- (+) Consistent with the rest of the codebase.

---

## 4. Data Model Changes

**None.** No new database tables, columns, views, indexes, or migrations are required.

Run-level state (current stage index, per-stage timing, total duration) is held in memory within the `RunController` struct and written to the run log on completion.

Entity status (the only persistent state the run loop modifies) is managed by the existing `EntityService.TransitionStatus()` method and its underlying repository calls.

---

## 5. Integration Approach

### 5.1 Services Consumed (All Existing)

| Service | Method | How RunController Uses It |
|---------|--------|--------------------------|
| `TaskService` / `FeatureService` / `EpicService` | `GetNextStatus(ctx, key)` | Called once per iteration to read current status + available transitions. Same method `shark status advance` uses. |
| `TaskService` / `FeatureService` / `EpicService` | `TransitionStatus(ctx, key, targetStatus, opts)` | Called to advance status after successful agent dispatch or for `advance_status` actions. Same method `shark status advance` uses. Returns `TransitionResult` with `OrchestratorAction` for new status. |
| `ActionService` | `GetStatusActionPopulated(ctx, status, vars)` | Called to get the orchestrator action and rendered instruction for the current status. Config read only — no DB call. |
| `workflow.Service` | `IsTerminalStatus(status)` | Called to detect when the entity has reached a terminal status and the loop should end. |

### 5.2 Entity Type Detection and Dispatch

The run command reuses the existing `ParseGetArgs()` function from `internal/cli/commands/helpers.go` to detect entity type from key format. Based on the entity type, it dispatches to the appropriate service:

```
Entity Key      -> Entity Type   -> Service
E07             -> epic          -> EpicService.TransitionStatus()
E07-F01         -> feature       -> FeatureService.TransitionStatus()
E07-F01-001     -> task          -> TaskService.TransitionStatus()
B001            -> bug           -> BugService.SetBugStatus()
CC-001          -> change_card   -> ChangeCardService.SetChangeCardStatus()
```

This follows the same dispatch pattern used by `runStatusAdvance()` in `status_group.go`.

### 5.3 Template Variable Generation

The run controller generates template placeholder variables using the existing helper functions:

- `config.TaskPlaceholders(task)` for tasks
- `config.FeaturePlaceholders(feature)` for features
- `config.EpicPlaceholders(epic)` for epics
- `config.BugPlaceholders(bug)` for bugs
- `config.ChangeCardPlaceholders(card)` for change cards

These are passed to `ActionService.GetStatusActionPopulated()` which handles template rendering.

### 5.4 Worktree Integration

For agent isolation, the run controller optionally creates a git worktree before dispatching:

1. Call `git worktree add <path> -b <branch>` via `os/exec`
2. Set the worktree path as the working directory (`cmd.Dir`) for the agent process
3. After agent exits, remove the worktree via `git worktree remove`

Controlled by the `--worktree` flag.

### 5.5 CLI Command Wiring

The `shark run` command follows the thin wrapper pattern:

```go
func runRun(cmd *cobra.Command, args []string) error {
    // Step 1: Parse entity key → detect type
    entityType, key, err := ParseGetArgs(args)

    // Step 2: Build entity-type adapters (transitioner + placeholder generator)
    transitioner := buildTransitioner(entityType)
    placeholders := buildPlaceholderGenerator(entityType)

    // Step 3: Get shared services + build dispatcher map
    actionSvc := cli.GetActionService()
    workflowSvc := cli.GetWorkflowService()
    dispatchers := map[string]AgentDispatcher{...}

    // Step 4: Construct and run controller
    controller := runner.NewRunController(deps)
    result := controller.Run(ctx, key, opts)

    // Step 5: Format output (JSON or human-readable)
}
```

---

## 6. Detailed Design: `internal/runner/` Package

### 6.1 `dispatcher.go` — Interface and Types

```go
// AgentDispatcher dispatches work to an external AI agent CLI tool.
type AgentDispatcher interface {
    Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error)
    Name() string
}

type RunOptions struct {
    DryRun     bool   // Preview actions without dispatching or advancing
    Verbose    bool   // Detailed stage progress to stderr
    WorkingDir string // Working directory override for agent processes
    Parallel   int    // Max concurrent children for cascade (0 or 1 = sequential)
}


type DispatchInput struct {
    Instruction    string   // Rendered instruction from template
    WorkingDir     string   // Working directory for the agent process
    EntityKey      string   // Entity key for context
    Status         string   // Current workflow status
    AgentType      string   // Agent type from orchestrator action
    Model          string   // Model override (optional)
}

type DispatchResult struct {
    ExitCode  int           // Process exit code (0 = success)
    Stdout    string        // Captured stdout
    Stderr    string        // Captured stderr
    Duration  time.Duration // Wall clock duration
    Command   string        // The full command that was executed (for logging)
}
```

### 6.2 `claude_dispatcher.go` — Claude CLI Implementation

Constructs and executes:
```
claude -p "<instruction>" \
  --disallowedTools "Bash(shark status advance*)" \
  --disallowedTools "Bash(shark task next-status*)" \
  --disallowedTools "Bash(shark status set*)" \
  --disallowedTools "Bash(shark task set-status*)" \
  --output-format json \
  [--model MODEL] \
  [--max-turns N]
```

### 6.3 `codex_dispatcher.go` — Codex CLI Implementation

Constructs and executes:
```
codex exec \
  -m <model> \
  --full-auto \
  -c "instruction: <instruction>"
```

### 6.4 `controller.go` — RunController (Simplified Loop)

```go
type RunController struct {
    transitioner    EntityTransitioner       // Same interface as shark status advance
    placeholders    PlaceholderGenerator     // Template variable generation
    actionSvc       config.ActionService     // Read orchestrator actions from config
    workflowSvc     *workflow.Service        // Terminal status detection
    dispatchers     map[string]AgentDispatcher
    childLister     ChildLister              // List child entities for cascade
    worktreeCreator WorktreeCreator          // Create/remove worktrees for parallel
}

func (c *RunController) Run(ctx context.Context, key string, opts RunOptions) (*RunResult, error) {
    // 1. GetNextStatus(key) → current status + is terminal?
    // 2. If terminal → return "already_terminal"
    // 3. LOOP:
    //    a. GeneratePlaceholders(key) → template vars
    //    b. GetStatusActionPopulated(currentStatus, vars) → orchestrator action
    //    c. Switch on action.Action:
    //       - advance_status:
    //           TransitionStatus(key, firstAvailableTransition)  ← same as shark status advance
    //           Update currentStatus from TransitionResult
    //           Loop
    //       - spawn_agent:
    //           Select dispatcher by provider
    //           Dispatch(instruction) → wait for exit
    //           If exit != 0 → return "failed"
    //           TransitionStatus(key, firstAvailableTransition)  ← same as shark status advance
    //           Update currentStatus from TransitionResult
    //           Loop
    //       - pause / wait_for_triage / check_or_resume:
    //           return "paused"
    //       - archive:
    //           return "completed"
    //    d. If new status is terminal → return "completed"
}
```

**Key simplifications**:
- The controller never computes transitions itself — it always delegates to `TransitionStatus()`.
- `spawn_agent` at `ready_for_*`: advance to `in_*` first, then dispatch. One agent launch per phase.
- `spawn_agent` at `in_*` (resume): dispatch directly, then advance on success.
- `cascade`: controller lists children and calls `Run()` recursively. With `--parallel=N`, uses goroutines + semaphore + worktrees for concurrent dispatch.

### 6.5 Interface Abstractions for Testability

```go
// EntityTransitioner advances entity status.
// Satisfied by the per-entity-type dispatch function.
type EntityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
    GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

// PlaceholderGenerator — generates template variables for instruction rendering.
type PlaceholderGenerator interface {
    GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error)
}

// ChildLister — lists child entities for cascade actions.
// Epic returns features, Feature returns tasks.
type ChildLister interface {
    ListChildren(ctx context.Context, parentKey string) ([]ChildEntity, error)
}

type ChildEntity struct {
    Key            string
    Status         string
    ExecutionOrder int
    IsTerminal     bool
}
```

---

## 7. File Organization

```
internal/
  runner/                        # NEW package
    controller.go                # RunController: orchestration loop
    controller_test.go           # Tests with mocked dispatchers and services
    dispatcher.go                # AgentDispatcher interface + types + execAndCapture()
    claude_dispatcher.go         # Claude CLI implementation
    claude_dispatcher_test.go    # Claude dispatcher unit tests
    codex_dispatcher.go          # Codex CLI implementation
    codex_dispatcher_test.go     # Codex dispatcher unit tests
    worktree.go                  # WorktreeCreator interface, GitWorktreeCreator
    worktree_test.go             # Worktree tests
    output_format.go             # TruncateOutput() helper
  runner/integration_test/       # Integration test environment (CC-020)
    env.go                       # Isolated test env with mock dispatchers
    run_loop_test.go             # End-to-end run loop tests
  cli/commands/
    run.go                       # shark run command, entity-type adapters
  cli/
    services_global.go           # GetActionService() accessor
```

---

## 8. Error Handling Strategy

### Run Loop Errors

| Error Condition | Behavior | Exit Code |
|-----------------|----------|-----------|
| Entity not found | Stop with "entity not found" error | 1 |
| Terminal status at start | Stop with "already_terminal" outcome | 0 |
| Agent exits non-zero | Stop, log failure, do not advance status | 2 |
| Agent CLI tool not found | Stop with "[tool] CLI not found on PATH" error | 2 |
| No orchestrator action for status | Stop with "no_action" outcome | 2 |
| Pause action encountered | Stop gracefully with "paused" outcome | 0 |
| Status transition fails | Stop with transition error details | 3 |
| Context cancelled (SIGINT) | Stop gracefully, log partial results | 130 |

### Error Types

The runner package uses existing error types where applicable:
- `*services.BackwardReasonError` from transition failures
- `*config.StatusNotFoundError` from missing actions
- New `*runner.AgentFailedError` wrapping non-zero exit codes
- New `*runner.ToolNotFoundError` wrapping `exec.LookPath` failures

---

## 9. Testing Strategy

### Unit Tests (Mocked Dependencies)

| Test File | What Is Tested | Mock Strategy |
|-----------|----------------|---------------|
| `controller_test.go` | Run loop logic: stage iteration, terminal detection, pause handling, failure stopping | Mock `EntityTransitioner`, `ActionService`, `workflow.Service`, `AgentDispatcher` |
| `claude_dispatcher_test.go` | Command construction: flag assembly, disallowed tools | Do not execute real process; verify command args |
| `codex_dispatcher_test.go` | Command construction: flag assembly, model passthrough | Do not execute real process; verify command args |

### Integration Tests (CC-020)

`internal/runner/integration_test/` provides a self-contained test environment with:
- Isolated SQLite database in `t.TempDir()`
- Its own `.sharkconfig.json` and `.sharkworkflow.json`
- `MockDispatcher` that returns canned responses instantly
- Tests that drive a task from `todo` → `completed` through the full run loop

Six integration tests cover: successful run, dry-run, agent failure, terminal entity, dispatch input validation, and dispatcher errors.

---

## 10. Security Considerations

### Agent Isolation

The primary security property is that agents cannot advance their own status forward:

1. **Claude CLI**: `--disallowedTools` blocks `Bash(shark status advance*)`, `Bash(shark task next-status*)`, `Bash(shark status set*)`, `Bash(shark task set-status*)`.
2. **Codex CLI**: Runs in sandbox mode (`--full-auto`) which restricts tool access.
3. **Backward transitions**: Agents retain ability to send backward (e.g., `shark status set E07-F01-001 changes_requested --reason "..."`) for rejection workflows. This is intentional — only forward advancement is blocked.

### Log Sanitization

Agent stdout/stderr is captured for logging. The logger truncates output to first/last 20 lines in non-verbose mode. Environment variables and auth tokens are not passed to agent subprocesses beyond what the system shell provides.

---

*Architecture approved: 2026-03-21. Updated: 2026-03-22 (simplified loop design per ADR-004 v2).*
