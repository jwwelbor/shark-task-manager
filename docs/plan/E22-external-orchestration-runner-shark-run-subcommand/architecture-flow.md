# E22: `shark run` — Architecture Flow Diagram

**Updated**: 2026-03-22 (simplified loop design)

## Overview

The `shark run <entity-key>` command drives an entity through its workflow with a simple loop:

1. **Get** entity state from DB (one call per iteration)
2. **Read** orchestrator action from workflow config (no DB)
3. **Execute** action (advance, spawn agent, pause, archive)
4. **Repeat** until terminal status or pause

The run controller reuses the **exact same** `GetNextStatus()` and `TransitionStatus()` methods that `shark status advance` calls. No separate transition logic exists in the controller.

---

## High-Level Architecture

```mermaid
graph TB
    subgraph "Entry Point"
        USER["User: shark run E07-F01-001<br/>--dry-run --verbose --worktree"]
        CMD["Cobra Command: run.go"]
    end

    subgraph "Command Handler — runRun()"
        PARSE["Step 1: Parse Entity Key<br/>ParseGetArgs() → entityType, key"]
        ADAPT["Step 2: Build Adapters<br/>buildTransitioner() + buildPlaceholderGenerator()"]
        SVCS["Step 3: Get Shared Services<br/>GetActionService() + GetWorkflowService()"]
        DISP_MAP["Step 4: Build Dispatcher Map<br/>provider → AgentDispatcher"]
        WORKTREE_SETUP["Step 5: Optional Worktree<br/>--worktree → isolated git copy"]
        CTRL_BUILD["Step 6: Construct RunController"]
        CTRL_RUN["Step 7: controller.Run(ctx, key, opts)"]
        OUTPUT["Step 8: Format Output<br/>JSON or human-readable"]
    end

    USER --> CMD
    CMD --> PARSE --> ADAPT --> SVCS --> DISP_MAP --> WORKTREE_SETUP --> CTRL_BUILD --> CTRL_RUN --> OUTPUT
```

---

## Core Orchestration Loop (Simplified)

This is the heart of the run command. Each iteration:
1. Reads entity state (DB call)
2. Reads action from config (no DB)
3. Executes the action
4. Loops with new status

```mermaid
flowchart TD
    START(["Run(ctx, key, opts)"])

    GET_STATUS["GetNextStatus(ctx, key)<br/>← same method as shark status advance<br/>Returns: currentStatus, availableTransitions, isTerminal"]

    IS_TERMINAL{"Is terminal?"}
    RETURN_TERMINAL["Return: outcome=already_terminal"]

    LOOP_START(["Loop iteration"])

    CTX_CHECK{"Context cancelled?"}
    CTX_FAIL["Return: outcome=failed"]

    GEN_VARS["GeneratePlaceholders(ctx, key)<br/>→ template variable map"]

    GET_ACTION["GetStatusActionPopulated(currentStatus, vars)<br/>← config read, no DB call<br/>Returns: PopulatedAction with rendered instruction"]

    NO_ACTION{"action == nil?"}
    RETURN_NO_ACTION["Return: outcome=no_action"]

    ACTION_SWITCH{"action.Action?"}

    PAUSE["pause / wait_for_triage<br/>Return: outcome=paused"]
    ARCHIVE["archive<br/>Return: outcome=completed"]

    ADVANCE["advance_status handler:<br/>TransitionStatus(key, firstTransition)<br/>← same as shark status advance"]

    SPAWN["spawn_agent handler:<br/>1. Dispatch agent<br/>2. Gate on exit code<br/>3. TransitionStatus(key, firstTransition)<br/>← same as shark status advance"]

    CHECK_TERMINAL{"New status terminal?"}
    RETURN_COMPLETED["Return: outcome=completed"]

    UPDATE["currentStatus = transitionResult.ToStatus"]

    START --> GET_STATUS
    GET_STATUS --> IS_TERMINAL
    IS_TERMINAL -->|Yes| RETURN_TERMINAL
    IS_TERMINAL -->|No| LOOP_START
    LOOP_START --> CTX_CHECK
    CTX_CHECK -->|Cancelled| CTX_FAIL
    CTX_CHECK -->|OK| GEN_VARS
    GEN_VARS --> GET_ACTION
    GET_ACTION --> NO_ACTION
    NO_ACTION -->|Yes| RETURN_NO_ACTION
    NO_ACTION -->|No| ACTION_SWITCH
    ACTION_SWITCH -->|pause/wait| PAUSE
    ACTION_SWITCH -->|archive| ARCHIVE
    ACTION_SWITCH -->|advance_status| ADVANCE
    ACTION_SWITCH -->|spawn_agent| SPAWN
    ADVANCE --> CHECK_TERMINAL
    SPAWN --> CHECK_TERMINAL
    CHECK_TERMINAL -->|Yes| RETURN_COMPLETED
    CHECK_TERMINAL -->|No| UPDATE
    UPDATE --> LOOP_START
```

---

## advance_status Handler

This is the simplest action: call `TransitionStatus()` (same as `shark status advance`) and loop. This is how `ready_for_` → `in_` transitions chain naturally.

```mermaid
flowchart TD
    ENTER(["handleAdvanceStatus()"])

    GET_NEXT["GetNextStatus(ctx, key)<br/>→ available transitions"]

    HAS_TRANS{"Has transitions?"}
    DONE_NO_TRANS["Return: done, outcome=completed"]

    TRANSITION["TransitionStatus(ctx, key, firstTransition)<br/>← same as shark status advance"]

    TRANS_ERR{"Error?"}
    TRANS_FAIL["Return: done, outcome=failed"]

    LOG_STAGE["Log stage: advance_status"]

    IS_TERM{"New status terminal?"}
    TERM_DONE["Return: done, outcome=completed"]
    CONTINUE["Return: nextStatus = result.ToStatus"]

    ENTER --> GET_NEXT
    GET_NEXT --> HAS_TRANS
    HAS_TRANS -->|No| DONE_NO_TRANS
    HAS_TRANS -->|Yes| TRANSITION
    TRANSITION --> TRANS_ERR
    TRANS_ERR -->|Yes| TRANS_FAIL
    TRANS_ERR -->|No| LOG_STAGE
    LOG_STAGE --> IS_TERM
    IS_TERM -->|Yes| TERM_DONE
    IS_TERM -->|No| CONTINUE
```

**Example chain** (advance_status auto-chains through the loop):
```
Iteration 1: status=draft           action=advance_status → TransitionStatus → ready_for_development
Iteration 2: status=ready_for_dev   action=advance_status → TransitionStatus → in_development
Iteration 3: status=in_development  action=spawn_agent    → dispatch developer agent...
```

---

## spawn_agent Handler

Dispatches an agent, waits for exit, gates advancement on exit code 0.

```mermaid
flowchart TD
    SPAWN_START(["handleSpawnAgent()"])

    SELECT_DISP["Select dispatcher by provider<br/>dispatchers[action.Provider]"]
    DISP_ERR{"Found?"}
    DISP_FAIL["Return: done, outcome=failed<br/>no dispatcher for provider"]

    BUILD_INPUT["Build DispatchInput:<br/>Instruction, WorkingDir, EntityKey,<br/>Status, AgentType, Model"]

    DRY_RUN{"opts.DryRun?"}
    DRY_LOG["Log planned stage<br/>Return: nextStatus (from config walk)"]

    DISPATCH["dispatcher.Dispatch(ctx, input)<br/>→ blocks until agent exits"]

    DISPATCH_ERR{"Go error?"}
    DISPATCH_FAIL["Return: done, outcome=failed"]

    EXIT_GATE{"ExitCode == 0?"}
    AGENT_FAIL["Return: done, outcome=failed<br/>agent exited with code N"]

    LOG_STAGE["Log stage: spawn_agent<br/>+ output summary"]

    GET_NEXT["GetNextStatus(ctx, key)<br/>→ available transitions"]
    HAS_TRANS{"Has transitions?"}
    NO_TRANS["Return: done, outcome=completed"]

    TRANSITION["TransitionStatus(ctx, key, firstTransition)<br/>← same as shark status advance"]
    TRANS_ERR{"Error?"}
    TRANS_FAIL["Return: done, outcome=failed"]

    IS_TERM{"New status terminal?"}
    TERM_DONE["Return: done, outcome=completed"]
    CONTINUE["Return: nextStatus = result.ToStatus"]

    SPAWN_START --> SELECT_DISP --> DISP_ERR
    DISP_ERR -->|No| DISP_FAIL
    DISP_ERR -->|Yes| BUILD_INPUT
    BUILD_INPUT --> DRY_RUN
    DRY_RUN -->|Yes| DRY_LOG
    DRY_RUN -->|No| DISPATCH
    DISPATCH --> DISPATCH_ERR
    DISPATCH_ERR -->|Yes| DISPATCH_FAIL
    DISPATCH_ERR -->|No| EXIT_GATE
    EXIT_GATE -->|Non-zero| AGENT_FAIL
    EXIT_GATE -->|0| LOG_STAGE
    LOG_STAGE --> GET_NEXT
    GET_NEXT --> HAS_TRANS
    HAS_TRANS -->|No| NO_TRANS
    HAS_TRANS -->|Yes| TRANSITION
    TRANSITION --> TRANS_ERR
    TRANS_ERR -->|Yes| TRANS_FAIL
    TRANS_ERR -->|No| IS_TERM
    IS_TERM -->|Yes| TERM_DONE
    IS_TERM -->|No| CONTINUE
```

---

## Claude Dispatcher Internals

```mermaid
flowchart TD
    CLAUDE_START(["ClaudeDispatcher.Dispatch()"])

    LOOK_PATH["exec.LookPath('claude')"]
    FOUND{"Found?"}
    NOT_FOUND["Return ToolNotFoundError"]

    BUILD_ARGS["Build args:<br/>-p instruction<br/>--output-format json<br/>--model (if set)<br/>--max-turns (if > 0)"]

    DISALLOW["Append --disallowedTools:<br/>Bash(shark status advance*)<br/>Bash(shark task next-status*)<br/>Bash(shark status set*)<br/>Bash(shark task set-status*)<br/>Bash(shark feature next-status*)<br/>Bash(shark epic next-status*)"]

    EXEC["exec.CommandContext(ctx, 'claude', args...)<br/>cmd.Dir = input.WorkingDir"]

    CAPTURE["execAndCapture():<br/>Pipe stdout/stderr<br/>Read concurrently<br/>Wait for exit"]

    RESULT["Return DispatchResult:<br/>ExitCode, Stdout, Stderr, Duration"]

    CLAUDE_START --> LOOK_PATH --> FOUND
    FOUND -->|No| NOT_FOUND
    FOUND -->|Yes| BUILD_ARGS --> DISALLOW --> EXEC --> CAPTURE --> RESULT
```

---

## Git Worktree Isolation

```mermaid
flowchart LR
    FLAG{"--worktree flag?"}
    SKIP["Use --workdir or cwd"]

    CREATE["NewGitWorktreeCreator()"]
    PATHS["WorktreePaths(baseDir, key, now)<br/>→ .shark-run-worktrees/KEY-TIMESTAMP"]
    GIT_ADD["git worktree add path -b branch"]
    DEFER_RM["defer: git worktree remove --force"]

    FLAG -->|No| SKIP
    FLAG -->|Yes| CREATE --> PATHS --> GIT_ADD --> DEFER_RM
```

---

## Entity Type Adapter Resolution

```mermaid
graph LR
    subgraph "buildTransitioner(entityType)"
        ET{entityType?}
        TASK["taskTransitionerAdapter<br/>→ TaskService.TransitionStatus()<br/>→ TaskService.GetNextStatus()"]
        FEAT["featureTransitionerAdapter<br/>→ FeatureService.TransitionStatus()<br/>→ FeatureService.GetNextStatus()"]
        EPIC["epicTransitionerAdapter<br/>→ EpicService.TransitionStatus()<br/>→ EpicService.GetNextStatus()"]
        BUG["bugTransitionerAdapter<br/>→ BugService.SetBugStatus()<br/>→ BugService.GetValidTransitions()"]
        CC["changeCardTransitionerAdapter<br/>→ ChangeCardService.SetChangeCardStatus()<br/>→ ChangeCardService.GetValidTransitions()"]
    end

    ET -->|task| TASK
    ET -->|feature| FEAT
    ET -->|epic| EPIC
    ET -->|bug| BUG
    ET -->|change_card| CC
```

---

## Dependency Injection & Interface Map

```mermaid
graph TB
    subgraph "Interfaces (defined at point of use)"
        I_TRANS["EntityTransitioner<br/>TransitionStatus() + GetNextStatus()"]
        I_PLACE["PlaceholderGenerator<br/>GeneratePlaceholders()"]
        I_DISP["AgentDispatcher<br/>Dispatch() + Name()"]
        I_WORK["WorktreeCreator<br/>CreateWorktree() + RemoveWorktree()"]
        I_ACTION["config.ActionService<br/>GetStatusActionPopulated()"]
    end

    subgraph "Implementations"
        TASK_A["taskTransitionerAdapter"]
        FEAT_A["featureTransitionerAdapter"]
        EPIC_A["epicTransitionerAdapter"]
        BUG_A["bugTransitionerAdapter"]
        CC_A["changeCardTransitionerAdapter"]
        CLAUDE_D["ClaudeDispatcher"]
        CODEX_D["CodexDispatcher"]
        GIT_W["GitWorktreeCreator"]
    end

    subgraph "RunController"
        RC_TRANS["transitioner"]
        RC_PLACE["placeholders"]
        RC_ACTION["actionSvc"]
        RC_WF["workflowSvc"]
        RC_DISP["dispatchers map"]
    end

    I_TRANS -.->|implements| TASK_A & FEAT_A & EPIC_A & BUG_A & CC_A
    I_DISP -.->|implements| CLAUDE_D & CODEX_D
    I_WORK -.->|implements| GIT_W

    RC_TRANS --- I_TRANS
    RC_PLACE --- I_PLACE
    RC_DISP --- I_DISP
    RC_WF --- I_ACTION
    RC_ACTION --- I_ACTION
```

---

## Data Types

```mermaid
classDiagram
    class RunOptions {
        +DryRun bool
        +Verbose bool
        +WorkingDir string
    }

    class RunResult {
        +EntityKey string
        +FinalStatus string
        +StagesCompleted int
        +Stages []StageLog
        +Outcome string
        +TotalDuration Duration
        +Error string
    }

    class StageLog {
        +Status string
        +Action string
        +AgentType string
        +Provider string
        +Duration Duration
        +ExitCode int
        +OutputSummary string
    }

    class DispatchInput {
        +Instruction string
        +WorkingDir string
        +EntityKey string
        +Status string
        +AgentType string
        +Model string
    }

    class DispatchResult {
        +ExitCode int
        +Stdout string
        +Stderr string
        +Duration Duration
        +Command string
    }

    class RunControllerDeps {
        +Transitioner EntityTransitioner
        +Placeholders PlaceholderGenerator
        +ActionSvc ActionService
        +WorkflowSvc *workflow.Service
        +Dispatchers map~string AgentDispatcher~
    }

    RunResult *-- StageLog : contains
    RunControllerDeps ..> RunOptions : used by Run()
    RunControllerDeps ..> RunResult : produces
```

---

## Example: Full Task Run

```
$ shark run E07-F01-001

Iteration 1:
  GetNextStatus("E07-F01-001") → status=draft, transitions=[ready_for_development]
  GetStatusActionPopulated("draft", vars) → action=advance_status
  TransitionStatus("E07-F01-001", "ready_for_development") → ok
  → new status: ready_for_development

Iteration 2:
  GetNextStatus("E07-F01-001") → status=ready_for_development, transitions=[in_development]
  GetStatusActionPopulated("ready_for_development", vars) → action=advance_status
  TransitionStatus("E07-F01-001", "in_development") → ok
  → new status: in_development

Iteration 3:
  GetNextStatus("E07-F01-001") → status=in_development, transitions=[ready_for_code_review]
  GetStatusActionPopulated("in_development", vars) → action=spawn_agent, agent=developer
  Dispatch(claude -p "Implement task E07-F01-001...") → exit code 0
  TransitionStatus("E07-F01-001", "ready_for_code_review") → ok
  → new status: ready_for_code_review

Iteration 4:
  GetNextStatus("E07-F01-001") → status=ready_for_code_review, transitions=[in_code_review]
  GetStatusActionPopulated("ready_for_code_review", vars) → action=advance_status
  TransitionStatus("E07-F01-001", "in_code_review") → ok
  → new status: in_code_review

...continues until terminal status (completed/cancelled)...
```

---

## Safety: DefaultDisallowedTools

Agents are **always** blocked from self-advancing status:

```
Bash(shark status advance*)
Bash(shark task next-status*)
Bash(shark status set*)
Bash(shark task set-status*)
Bash(shark feature next-status*)
Bash(shark epic next-status*)
```

Only the RunController advances status — after gating on agent exit code 0.

---

## File Index

| File | Purpose |
|------|---------|
| `internal/cli/commands/run.go` | CLI command, flag registration, entity-type adapters |
| `internal/runner/controller.go` | RunController: simplified orchestration loop |
| `internal/runner/dispatcher.go` | AgentDispatcher interface, DispatchInput/Result, error types, execAndCapture() |
| `internal/runner/claude_dispatcher.go` | ClaudeDispatcher: validates binary, builds args, executes `claude` CLI |
| `internal/runner/codex_dispatcher.go` | CodexDispatcher: same pattern for OpenAI Codex CLI |
| `internal/runner/worktree.go` | WorktreeCreator interface, GitWorktreeCreator, path sanitization |
| `internal/runner/output_format.go` | TruncateOutput() helper for display |
| `internal/runner/integration_test/env.go` | Isolated test environment with mock dispatchers (CC-020) |
| `internal/runner/integration_test/run_loop_test.go` | End-to-end run loop integration tests |
| `internal/cli/services_global.go` | GetActionService(), GetWorkflowService() global accessors |
