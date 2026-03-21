# E22 Architecture: External Orchestration Runner

**Epic**: E22 - External Orchestration Runner - shark run subcommand
**Date**: 2026-03-21
**Status**: Accepted

---

## 1. Component Overview

### What Changes

| Component | Change Type | Description |
|-----------|-------------|-------------|
| `internal/runner/` | **NEW package** | Run loop controller, agent dispatcher interface, Claude and Codex dispatcher implementations |
| `internal/cli/commands/run.go` | **NEW file** | Cobra command registration for `shark run <entity-key>` |
| `internal/cli/services_global.go` | **EXTEND** | Add `GetActionService()` global accessor (currently created inline) |

### What Stays Unchanged

| Component | Reason |
|-----------|--------|
| `internal/services/entity_service.go` | `TransitionStatus()` is called as-is by the run loop |
| `internal/config/orchestrator_action.go` | `OrchestratorAction` struct already has all required fields (`Provider`, `Model`, `AgentType`, `Skills`) |
| `internal/config/action_service.go` | `GetStatusActionPopulated()` already returns `PopulatedAction` with rendered instructions |
| `internal/workflow/service.go` | `GetValidTransitions()`, `IsTerminalStatus()` used as-is |
| `shark-templates/` | Agent instruction templates are consumed as-is |
| `.sharkconfig.json` schema | No changes to workflow configuration format |
| Database schema | No new tables or columns |

### Component Diagram

```
shark run <key>
  |
  v
internal/cli/commands/run.go           (Cobra command: parse args, call RunController)
  |
  v
internal/runner/controller.go          (RunController: orchestration loop)
  |-- reads entity status via EntityService/TaskService/etc. (existing)
  |-- reads orchestrator action via ActionService (existing)
  |-- dispatches to AgentDispatcher interface (new)
  |     |-- ClaudeDispatcher (new) --> os/exec: claude -p "..."
  |     |-- CodexDispatcher (new)  --> os/exec: codex exec "..."
  |-- advances status via TransitionStatus (existing)
  |-- logs stage results via RunLogger (new)
  |-- loops until terminal/pause/failure
  |
  v
internal/runner/dispatcher.go          (AgentDispatcher interface + DispatchResult)
internal/runner/claude_dispatcher.go   (Claude CLI implementation)
internal/runner/codex_dispatcher.go    (Codex CLI implementation)
internal/runner/logger.go              (Structured run logging)
```

---

## 2. Key Technical Decisions (ADRs)

### ADR-001: New `internal/runner/` Package for Run Loop

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The run loop is a new control flow pattern -- it reads orchestrator actions, dispatches external processes, waits for exit codes, and advances status in a loop. No existing package does this.

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
- Interface-based dispatch satisfies REQ-NF-005 (extensibility).
- Function-based selection (map of provider to dispatcher) is simpler than a plugin system.
- Each dispatcher encapsulates CLI-specific flag construction, which varies significantly between Claude (`-p`, `--disallowedTools`, `--max-turns`) and Codex (`exec`, `-m`, `-s`).
- Matches the project's preference for explicit interfaces over generic patterns (`.claude/rules/services/service-design.md` Section 2).

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
- The entity's database status is already the resumption point -- it reflects the last successfully completed stage.
- Run logs (stdout/file) provide the audit trail that persistent run sessions would offer.
- Aligns with constraint C-003 from the epic PRD: "No new database tables for run state."

**Consequences**:
- (+) Zero database schema changes; no migration concerns.
- (+) Re-running is idempotent -- always picks up from current persisted status.
- (-) Per-run metadata (stage durations, retry counts) is lost if the process is killed. Mitigated by log output.

### ADR-004: First Valid Transition as Default Forward Status

**Date**: 2026-03-21
**Status**: Accepted

**Context**: After an agent completes successfully, the run loop must decide which status to advance to. `GetValidTransitions()` returns a list of valid next statuses.

**Decision**: Use `transitions[0]` (the first valid transition) as the default forward target, matching the existing behavior in `runStatusAdvance()`.

**Rationale**:
- The existing `status advance` command uses this exact convention (`status_group.go:481`).
- The workflow configuration orders transitions intentionally -- the first entry is the "happy path" forward transition.
- Backward transitions (like `changes_requested`) are listed but never selected automatically; they require explicit agent or human action.
- Consistency with existing behavior eliminates surprise.

**Consequences**:
- (+) Consistent with how `shark status advance` already works.
- (+) No new configuration needed for forward transitions.
- (-) If a status has multiple forward paths (rare), the first is always chosen. A future enhancement could allow the orchestrator action to specify the target status.

### ADR-005: Disallowed Tools for Agent Isolation

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The core security property of E22 is that agents cannot advance their own status. Claude CLI supports `--disallowedTools` to block specific tool invocations.

**Decision**: Pass `--disallowedTools "Bash(shark status advance*)" "Bash(shark task next-status*)" "Bash(shark status set*)" "Bash(shark task set-status*)"` to Claude CLI. For Codex, rely on its sandbox mode which restricts filesystem/network access.

**Rationale**:
- This blocks the four CLI commands an agent could use to self-advance: `shark status advance`, `shark task next-status`, `shark status set`, and `shark task set-status`.
- The `*` wildcard ensures variants with arguments are also blocked.
- Codex's `--full-auto` sandbox mode already restricts tool access; no additional flags needed.
- This is the architectural enforcement the entire epic is built around.

**Consequences**:
- (+) Agents physically cannot advance status -- the enforcement is not prompt-level.
- (+) Agents can still perform backward transitions (e.g., `shark status set ... changes_requested`) if needed for rejection workflows.
- (-) Depends on Claude CLI `--disallowedTools` flag stability. Mitigated by isolating flag construction to the dispatcher implementation (~5 lines to change if the API changes).

### ADR-006: RunController Receives Services via Constructor Injection

**Date**: 2026-03-21
**Status**: Accepted

**Context**: The run controller needs access to entity services (for status transition), the action service (for orchestrator actions), the workflow service (for valid transitions), and agent dispatchers.

**Decision**: Use constructor injection matching the existing service pattern in `internal/services/`.

**Rationale**:
- Follows the established dependency injection pattern (`.claude/rules/architecture.md` Section 2).
- Constructor receives interfaces, enabling mock injection for tests.
- The CLI command wires dependencies via global accessors, matching existing commands.

**Consequences**:
- (+) Testable with mocked services and dispatchers.
- (+) Consistent with the rest of the codebase.

---

## 3. Data Model Changes

**None.** No new database tables, columns, views, indexes, or migrations are required.

Run-level state (current stage index, per-stage timing, total duration) is held in memory within the `RunController` struct and written to the run log on completion.

Entity status (the only persistent state the run loop modifies) is managed by the existing `EntityService.TransitionStatus()` method and its underlying repository calls.

---

## 4. Integration Approach

### 4.1 Services Consumed (All Existing)

| Service | Method | How RunController Uses It |
|---------|--------|--------------------------|
| `EntityService` | `TransitionStatus(ctx, repo, entityType, key, targetStatus, opts, features, resolveActionFn)` | Called after each successful agent dispatch to advance the entity to the next status |
| `ActionService` | `GetStatusActionPopulated(ctx, status, vars)` | Called at each loop iteration to get the orchestrator action and rendered instruction for the current status |
| `workflow.Service` | `GetValidTransitions(currentStatus)` | Called to determine the default forward target status (first valid transition) |
| `workflow.Service` | `IsTerminalStatus(status)` | Called to detect when the entity has reached a terminal status and the loop should end |
| `TaskService` / `FeatureService` / `EpicService` | `TransitionStatus(ctx, key, targetStatus, opts)` | Entity-specific transition methods that delegate to `EntityService`; the run command dispatches to the correct one based on entity type |

### 4.2 Entity Type Detection and Dispatch

The run command reuses the existing `ParseGetArgs()` function from `internal/cli/commands/helpers.go` to detect entity type from key format. Based on the entity type, it dispatches to the appropriate service:

```
Entity Key      -> Entity Type   -> Service
E07             -> epic          -> EpicService.TransitionStatus()
E07-F01         -> feature       -> FeatureService.TransitionStatus()
E07-F01-001     -> task          -> TaskService.TransitionStatus()
B001            -> bug           -> BugService.Advance()
CC-001          -> change_card   -> ChangeCardService.Advance()
```

This follows the same dispatch pattern used by `runStatusAdvance()` in `status_group.go`.

### 4.3 Template Variable Generation

The run controller generates template placeholder variables using the existing helper functions:

- `config.TaskPlaceholders(task)` for tasks
- `config.FeaturePlaceholders(feature)` for features
- `config.EpicPlaceholders(epic)` for epics

These are passed to `ActionService.GetStatusActionPopulated()` which handles template rendering.

### 4.4 Worktree Integration (Should Have)

For agent isolation, the run controller optionally creates a git worktree before dispatching each agent:

1. Call `git worktree add .claude/worktrees/<entity-key>-<status> <current-branch>` via `os/exec`
2. Set the worktree path as the working directory (`cmd.Dir`) for the agent process
3. After agent exits, optionally remove the worktree via `git worktree remove`

This is implemented as an optional pre/post hook on agent dispatch, controlled by a `--no-worktree` flag.

### 4.5 CLI Command Wiring

The `shark run` command follows the thin wrapper pattern:

```go
// internal/cli/commands/run.go
func runRun(cmd *cobra.Command, args []string) error {
    // Step 1: Parse arguments
    entityType, key, err := ParseGetArgs(args)

    // Step 2: Wire controller with services
    controller := buildRunController(entityType)

    // Step 3: Execute run loop
    result, err := controller.Run(cmd.Context(), key, runOptions)

    // Step 4: Format output
    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }
    // Human-readable summary output
}
```

The `buildRunController` function uses global service accessors (`cli.GetEntityService()`, etc.) to wire dependencies, matching the established pattern.

---

## 5. Detailed Design: `internal/runner/` Package

### 5.1 `dispatcher.go` -- Interface and Types

```go
// AgentDispatcher dispatches work to an external AI agent CLI tool.
type AgentDispatcher interface {
    // Dispatch invokes the agent CLI with the given input and blocks until completion.
    // Returns the result including exit code, stdout, stderr, and duration.
    Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error)

    // Name returns the dispatcher name for logging (e.g., "claude", "codex").
    Name() string
}

// DispatchInput contains all information needed to invoke an agent.
type DispatchInput struct {
    Instruction    string            // Rendered instruction from template
    WorkingDir     string            // Working directory for the agent process
    EntityKey      string            // Entity key for context
    EntityType     string            // Entity type for context
    Status         string            // Current workflow status
    AgentType      string            // Agent type from orchestrator action
    Model          string            // Model override (optional)
    ExtraFlags     map[string]string // Additional CLI flags
}

// DispatchResult contains the outcome of an agent dispatch.
type DispatchResult struct {
    ExitCode  int           // Process exit code (0 = success)
    Stdout    string        // Captured stdout
    Stderr    string        // Captured stderr
    Duration  time.Duration // Wall clock duration
    Command   string        // The full command that was executed (for logging)
}
```

### 5.2 `claude_dispatcher.go` -- Claude CLI Implementation

Constructs and executes:
```
claude -p "<instruction>" \
  --disallowedTools "Bash(shark status advance*)" \
  --disallowedTools "Bash(shark task next-status*)" \
  --disallowedTools "Bash(shark status set*)" \
  --disallowedTools "Bash(shark task set-status*)" \
  --output-format json \
  [--max-turns N] \
  [--allowedTools ...]
```

Key implementation details:
- Uses `exec.CommandContext(ctx, "claude", args...)` for cancellation support
- Captures stdout and stderr via `cmd.StdoutPipe()` / `cmd.StderrPipe()`
- Validates `claude` binary exists via `exec.LookPath("claude")` before dispatch
- Returns `DispatchResult` with exit code from `cmd.Wait()`

### 5.3 `codex_dispatcher.go` -- Codex CLI Implementation

Constructs and executes:
```
codex exec \
  -m <model> \
  --full-auto \
  -c "instruction: <instruction>" \
  [--skip-git-repo-check]
```

Same `exec.CommandContext` pattern as Claude dispatcher.

### 5.4 `controller.go` -- RunController

The core orchestration loop:

```go
type RunController struct {
    entitySvc    EntityTransitioner    // Advance status (interface wrapping per-type services)
    actionSvc    config.ActionService  // Read orchestrator actions
    workflowSvc  *workflow.Service     // Get valid transitions, check terminal
    dispatchers  map[string]AgentDispatcher  // Provider -> dispatcher
    logger       *RunLogger            // Structured logging
    entityGetter EntityGetter          // Get entity by key for placeholder generation
}

// Run executes the orchestration loop for an entity.
func (c *RunController) Run(ctx context.Context, key string, opts RunOptions) (*RunResult, error) {
    // 1. Get entity and validate it exists
    // 2. Check current status
    // 3. LOOP:
    //    a. If terminal status -> break (success)
    //    b. Get orchestrator action for current status
    //    c. Switch on action type:
    //       - spawn_agent: dispatch to agent, wait for exit
    //       - pause: break (paused)
    //       - advance_status: advance without agent
    //       - cascade: trigger cascade
    //       - archive: break (archived)
    //    d. If agent exit code != 0 -> break (failure)
    //    e. Advance to next status
    //    f. Log stage result
    // 4. Output run summary
}
```

### 5.5 `logger.go` -- Run Logging

```go
type RunLogger struct {
    stages  []StageLog
    verbose bool
}

type StageLog struct {
    Status    string        `json:"status"`
    Action    string        `json:"action"`
    AgentType string        `json:"agent_type,omitempty"`
    Provider  string        `json:"provider,omitempty"`
    Duration  time.Duration `json:"duration"`
    ExitCode  int           `json:"exit_code"`
    Output    string        `json:"output_summary,omitempty"` // First/last 20 lines
}
```

### 5.6 Interface Abstractions for Testability

The `RunController` depends on thin interfaces rather than concrete service types:

```go
// EntityTransitioner advances entity status.
// Satisfied by the per-entity-type dispatch function.
type EntityTransitioner interface {
    TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)
}

// EntityGetter retrieves entity information for placeholder generation.
type EntityGetter interface {
    GetEntity(ctx context.Context, key string) (models.Entity, error)
}
```

This allows tests to inject mock transitioners and entity getters without needing real database connections.

---

## 6. File Organization

```
internal/
  runner/                        # NEW package
    controller.go                # RunController: orchestration loop (~150 lines)
    controller_test.go           # Tests with mocked dispatchers and services
    dispatcher.go                # AgentDispatcher interface + types (~50 lines)
    claude_dispatcher.go         # Claude CLI implementation (~80 lines)
    claude_dispatcher_test.go    # Claude dispatcher unit tests
    codex_dispatcher.go          # Codex CLI implementation (~60 lines)
    codex_dispatcher_test.go     # Codex dispatcher unit tests
    logger.go                    # Structured run logging (~80 lines)
    logger_test.go               # Logger tests
    worktree.go                  # Git worktree management (~80 lines, Should Have)
    worktree_test.go             # Worktree tests
  cli/commands/
    run.go                       # NEW: shark run command registration (~60 lines)
  cli/
    services_global.go           # EXTEND: add GetActionService() accessor
```

**Estimated new code**: ~560 lines (Must Have) + ~80 lines (Should Have: worktree) = ~640 lines total.

---

## 7. Error Handling Strategy

### Run Loop Errors

| Error Condition | Behavior | Exit Code |
|-----------------|----------|-----------|
| Entity not found | Stop with "entity not found" error | 1 |
| Terminal status at start | Stop with "entity is already in terminal status" message | 0 |
| Agent exits non-zero | Stop, log failure, do not advance status | 2 |
| Agent CLI tool not found | Stop with "[tool] CLI not found on PATH" error | 2 |
| No orchestrator action for status | Stop with warning, suggest checking workflow config | 2 |
| Pause action encountered | Stop gracefully with "paused" message | 0 |
| Status transition fails | Stop with transition error details | 3 |
| Context cancelled (SIGINT) | Stop gracefully, log partial results | 130 |

### Error Types

The runner package uses existing error types where applicable:
- `*services.BackwardReasonError` from transition failures
- `*config.StatusNotFoundError` from missing actions
- New `*runner.AgentFailedError` wrapping non-zero exit codes
- New `*runner.ToolNotFoundError` wrapping `exec.LookPath` failures

---

## 8. Testing Strategy

### Unit Tests (Mocked Dependencies)

| Test File | What Is Tested | Mock Strategy |
|-----------|----------------|---------------|
| `controller_test.go` | Run loop logic: stage iteration, terminal detection, pause handling, failure stopping | Mock `EntityTransitioner`, `ActionService`, `workflow.Service`, `AgentDispatcher` |
| `claude_dispatcher_test.go` | Command construction: flag assembly, disallowed tools | Do not execute real process; verify command args |
| `codex_dispatcher_test.go` | Command construction: flag assembly, model passthrough | Do not execute real process; verify command args |
| `logger_test.go` | Log formatting, summary generation | Pure logic, no mocks needed |

### Integration Tests

A full integration test drives a task from `draft` to `completed` using a mock dispatcher that always exits 0. This validates the loop, status transitions, and action reading work together end-to-end without needing real Claude/Codex CLI tools.

### Testing the Dispatcher Interface

Dispatchers are tested by verifying the `exec.Cmd` construction (arguments, environment, working directory) without actually running the subprocess. The `exec.CommandContext` call is wrapped in a function field on the dispatcher struct, allowing tests to substitute a recorder.

---

## 9. Migration Strategy

**No migration required.** This epic adds a new CLI command (`shark run`) and a new internal package (`internal/runner/`). It does not modify existing commands, services, repositories, or database schema.

Existing users continue using `shark status advance` and the `/run` skill unchanged. `shark run` is a new entry point that coexists with existing workflows.

### Rollout Plan

1. Build and merge `shark run` behind a feature flag (or simply as a new subcommand that users opt into).
2. Validate with integration tests that the run loop drives a task through all statuses.
3. Update documentation to recommend `shark run` for automated workflow execution.
4. The `/run` skill is not modified or deprecated in this epic (per scope exclusion).

---

## 10. Security Considerations

### Agent Isolation

The primary security property is that agents cannot advance their own status forward:

1. **Claude CLI**: `--disallowedTools` blocks `Bash(shark status advance*)`, `Bash(shark task next-status*)`, `Bash(shark status set*)`, `Bash(shark task set-status*)`.
2. **Codex CLI**: Runs in sandbox mode (`--full-auto`) which restricts tool access.
3. **Backward transitions**: Agents retain ability to send backward (e.g., `shark status set E07-F01-001 changes_requested --reason "..."`) for rejection workflows. This is intentional -- only forward advancement is blocked.

### Log Sanitization

Agent stdout/stderr is captured for logging. The logger truncates output to first/last 20 lines in non-verbose mode. Environment variables and auth tokens are not passed to agent subprocesses beyond what the system shell provides.

---

*Architecture approved: 2026-03-21*
