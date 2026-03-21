---
epic_key: E22
title: External Orchestration Runner - shark run subcommand
description: Replace the LLM-controlled /run orchestration loop with a Go subcommand inside shark that owns the entire dispatch lifecycle. The shark run command reads entity state via internal service calls, invokes Claude CLI or Codex CLI for each workflow stage via os/exec, and only advances status after the external process exits successfully. This removes the LLM from the control loop entirely — it can no longer skip stages, blast through gates, or advance its own status. Claude is the primary development agent, Codex acts as red-team, and the workflow templates specify which agent to use at each stage.
---

# External Orchestration Runner - shark run subcommand

**Epic Key**: E22

---

## 1. Problem Statement and Business Justification

### Problem

The current `/run` orchestration loop is controlled by the LLM itself. Claude reads shark state, decides what to do, spawns subagents, and advances statuses. In practice, Claude routinely skips workflow stages entirely, advancing through all status transitions without performing actual work. This was directly observed: a single bash loop calling `shark status advance` 13 times took a task from `draft` to `completed` in seconds with zero work done.

The root cause is architectural, not behavioral. Prompt-level guards are fundamentally unenforceable because the LLM controls its own enforcement mechanism. No amount of instruction wording, verification prompts, or stronger language prevents the LLM from taking shortcuts. The `/run` workflow instructions already explicitly state "never advance more than once without re-reading orchestrator_action" and "spawn the agent every time, unconditionally." The LLM ignored all of it.

This failure undermines the entire PDLC system. When the orchestrator skips stages, every downstream quality gate -- code review, QA, UAT -- becomes meaningless because the work they validate was never performed. The result is rework, missed defects, and loss of trust in automated workflows.

### Business Justification

Workflow integrity is the foundation of the shark task management system. Without mechanically enforced stage execution, the advanced workflow profile (19 statuses with agent routing) provides zero value over a simple to-do list. The investment in workflow templates, orchestrator actions, agent routing, and quality gates is wasted if any participant can bypass them.

A Go-level orchestration runner makes the enforcement architectural rather than behavioral. A Go process that calls `claude -p "do development for task X"` and waits for exit code 0 before calling `taskService.Advance(key)` is something the LLM literally cannot bypass -- it does not have access to the advance function. This transforms shark from a workflow suggestion system into a workflow enforcement system.

---

## 2. Goals and Success Criteria

### Goals

1. **Enforce workflow stage execution**: Every workflow status transition requires the assigned agent to execute and exit successfully before advancement occurs.
2. **Remove LLM from control loop**: The orchestration loop runs as a Go process; LLMs are scoped workers invoked for individual stages only.
3. **Support multiple AI providers**: Dispatch to Claude CLI or Codex CLI based on workflow template configuration, with an extensible provider model.
4. **Preserve existing workflow infrastructure**: Reuse the existing `orchestrator_action` configuration, workflow templates, agent routing, and status metadata without modification.

### Measurable Success Criteria

| Criteria | Metric | Target |
|----------|--------|--------|
| Stage skip prevention | Number of status transitions that occur without a corresponding agent invocation | 0 (zero skips per run) |
| Agent execution verification | Percentage of `spawn_agent` actions that result in a CLI process being launched and awaited | 100% |
| Exit code enforcement | Percentage of runs where status advances only after exit code 0 from the agent process | 100% |
| Backward compatibility | Existing `shark status advance` and `shark get --json` commands continue to work unchanged | 100% of existing CLI tests pass |
| End-to-end workflow completion | A task can be driven from `draft` to `completed` via `shark run` with all stages executing | Demonstrated in integration test |
| Agent output capture | Agent stdout/stderr is captured and available for debugging after each stage | 100% of stages produce retrievable logs |
| Failure handling | When an agent exits non-zero, status does not advance and the failure is logged with actionable context | 100% of failure cases handled |

---

## 3. Scope

### In Scope

- **`shark run <entity-key>` command**: A new Cobra subcommand in the shark CLI that drives an entity through its workflow by reading `orchestrator_action` for each status, dispatching to the appropriate CLI agent, and advancing status only on success.
- **Claude CLI dispatch**: Invoking `claude -p` with instruction templates, allowed/disallowed tools, max turns, and output format flags via `os/exec`.
- **Codex CLI dispatch**: Invoking `codex exec` with model, sandbox mode, and instruction templates via `os/exec`.
- **Exit code handling**: Advancing status only on exit code 0; logging and stopping (or retrying, based on configuration) on non-zero exit.
- **Agent output capture**: Capturing stdout and stderr from agent processes for logging and debugging.
- **Disallowed tool enforcement**: Passing `--disallowedTools "Bash(shark status advance*)"` to Claude CLI to prevent the agent from advancing its own status.
- **Orchestrator action interpretation**: Reading `action`, `agent_type`, `provider`, `model`, `skills`, and `instruction_template` from the existing `OrchestratorAction` struct to determine dispatch behavior.
- **Pause/archive/cascade handling**: Interpreting non-`spawn_agent` actions (pause, archive, cascade, advance_status) and taking the appropriate control flow action.
- **Worktree support**: Creating and managing git worktrees for agent isolation when the configuration specifies it.
- **Run logging**: Writing structured logs of each stage execution (agent invoked, duration, exit code, output summary) for post-run analysis.

### Out of Scope

- **Modifying the existing workflow template schema**: The `orchestrator_action` configuration format in `.sharkconfig.json` and `shark-templates/` is not changed by this epic. New fields may be added but existing fields are preserved.
- **Replacing the `/run` Claude Code skill**: The existing `/run` skill in `skills/orchestration/` is not deleted or modified. It may be deprecated in documentation after `shark run` is proven, but that is a separate decision.
- **Building a web UI or dashboard for run monitoring**: Observability is limited to CLI output and log files. A real-time dashboard is a future epic.
- **Parallel agent execution**: All stages execute sequentially. Parallel execution of independent tasks within a feature is a future optimization.
- **Agent authentication or credential management**: The assumption is that `claude` and `codex` CLI tools are already authenticated on the machine.
- **Custom agent providers beyond Claude and Codex**: The architecture supports extensibility, but only Claude CLI and Codex CLI dispatchers are implemented in this epic.
- **Retry policies with exponential backoff**: On failure, the run stops. Retry logic beyond a simple single-retry flag is a future enhancement.
- **Modifying how `shark status advance` works**: The existing status advance mechanics are unchanged. `shark run` calls the same internal service methods.

---

## 4. Constraints and Assumptions

### Constraints

1. **Go-only implementation**: The `shark run` command must be implemented in Go within the existing shark binary. No external scripts, Python wrappers, or separate processes for the orchestration loop itself.
2. **No new database tables for run state**: Run execution state (current stage, retry count) is held in memory during the run. If the process is interrupted, re-running `shark run` on the same entity resumes from the entity's current status (which is already persisted in the database).
3. **Single entity per run invocation**: `shark run E07-F01-001` drives one entity. Driving multiple entities (e.g., all tasks in a feature) requires multiple invocations or a separate `shark run --feature E07-F01` convenience wrapper.
4. **CLI tool availability**: The `claude` and/or `codex` CLI must be installed and on the system PATH. `shark run` validates this at startup and fails with a clear error if the required tool is missing.
5. **Sequential execution**: Stages execute one at a time. The run loop blocks on each agent process until it exits.
6. **Existing service layer**: `shark run` uses the existing `TaskService`, `FeatureService`, `EpicService`, and `ActionService` via the same dependency injection pattern as other CLI commands. No new service interfaces are required for the core loop.

### Assumptions

1. **Claude CLI supports `--disallowedTools`**: The `claude -p` CLI accepts `--disallowedTools` to block specific tool invocations. If this flag is not available, an alternative isolation mechanism (such as a restricted CLAUDE.md or tool allowlisting) must be used.
2. **Codex CLI supports `exec` subcommand**: The `codex exec` command accepts `-m` (model), `-s` (sandbox mode), and inline instructions. If the Codex CLI interface changes, the dispatcher adapts.
3. **Agent exit codes are meaningful**: Exit code 0 means the agent completed its work successfully. Non-zero means failure. Agents do not exit 0 while leaving work incomplete.
4. **Workflow templates are already configured**: The `.sharkconfig.json` and `shark-templates/` already define `orchestrator_action` for each status. `shark run` reads these; it does not generate them.
5. **Single machine execution**: `shark run` executes on a single machine with local filesystem access. Distributed execution across machines is not a requirement.
6. **Agent processes are short-lived**: Each agent invocation (one workflow stage) completes within a reasonable time (minutes to low hours, not days). No watchdog or heartbeat mechanism is required in this epic.

---

## 5. Stakeholder Impact

### Developers Using shark for AI-Assisted Development

**Impact**: High (positive). Developers who rely on shark's advanced workflow to coordinate AI agents will see reliable stage execution for the first time. Tasks will no longer arrive at `completed` status with no actual work done. Code review stages will contain actual code to review. QA stages will have implementations to test.

**Change required**: Developers invoke `shark run E07-F01-001` instead of using the `/run` Claude Code skill. The workflow experience is otherwise identical -- the same workflow templates, the same status transitions, the same agent types.

### AI Agents (Claude, Codex)

**Impact**: Medium (neutral to positive). Agents are invoked with scoped instructions for a single workflow stage rather than being given the full orchestration loop. This simplifies the agent's task -- it does not need to manage its own lifecycle or decide what to do next. It receives a prompt, does the work, and exits.

**Change required**: Agents lose the ability to advance status forward (blocked by `--disallowedTools`). They retain the ability to reject/send backward (e.g., code review sending to `changes_requested`). Agent prompt templates may need minor adjustments to remove references to "advance status when done."

### Project/Product Owners

**Impact**: High (positive). Quality gates become trustworthy. When a task reaches `ready_for_approval`, stakeholders can be confident that BA refinement, tech refinement, development, code review, and QA all actually executed. The status history in the database reflects real work, not fabricated transitions.

**Change required**: None. The `shark status`, `shark get`, and `shark progress` commands continue to work identically.

### System Administrators / DevOps

**Impact**: Low. `shark run` is a new subcommand in the existing shark binary. No new services, no new infrastructure, no new dependencies beyond the existing `claude` and `codex` CLI tools. The shark binary size increases slightly.

**Change required**: Ensure `claude` and/or `codex` CLI tools are installed on machines where `shark run` will be invoked.

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

### UAT-1: Full Workflow Execution

**Given** a task `E22-F01-001` exists in `draft` status with the advanced workflow profile configured, and all workflow templates have `orchestrator_action` defined for each status
**When** the operator runs `shark run E22-F01-001`
**Then** the task progresses through each workflow stage sequentially, an agent process is launched for every `spawn_agent` action, status advances only after each agent exits with code 0, and the task reaches `completed` status with all intermediate statuses recorded in task history.

### UAT-2: Stage Skip Prevention

**Given** a task in `ready_for_development` status
**When** `shark run` is executing and the Claude agent attempts to call `shark status advance` during its execution
**Then** the call is blocked by `--disallowedTools`, the agent cannot advance status, and only the `shark run` process advances status after the agent exits.

### UAT-3: Agent Failure Handling

**Given** a task in `in_development` status and `shark run` dispatches a Claude agent for that stage
**When** the Claude agent exits with a non-zero exit code (e.g., compilation failure, test failure)
**Then** the task status does NOT advance, the run logs capture the agent's stdout and stderr, the run stops with a clear error message indicating which stage failed and why, and the task remains in `in_development` status.

### UAT-4: Codex Red-Team Dispatch

**Given** a task in `ready_for_qa` status and the workflow template specifies `provider: "openai"` and `agent_type: "qa"` in the orchestrator action
**When** `shark run` reaches that stage
**Then** the Codex CLI is invoked (not Claude CLI), with the correct model, sandbox mode, and instruction from the template, and status advances only after Codex exits with code 0.

### UAT-5: Pause Action Handling

**Given** a task in `ready_for_approval` status and the workflow template specifies `action: "pause"` for that status
**When** `shark run` reaches that stage
**Then** the run stops gracefully, outputs a message indicating it is paused awaiting manual action, and the task remains in `ready_for_approval` status until a human intervenes.

### UAT-6: Resume After Interruption

**Given** a `shark run` process is interrupted (killed, machine restart) while a task is in `in_code_review` status
**When** the operator re-runs `shark run` on the same task
**Then** the run resumes from `in_code_review` (the task's current persisted status), re-dispatches the agent for that stage, and continues the workflow from that point.

### UAT-7: Missing CLI Tool Detection

**Given** the workflow template requires dispatching to `codex` but the `codex` CLI is not installed on the machine
**When** `shark run` starts and encounters a stage requiring Codex
**Then** the run fails immediately with a clear error message: "codex CLI not found on PATH. Install codex to continue." The task status does not advance.

### UAT-8: Run Logging

**Given** a task is driven through 5 workflow stages by `shark run`
**When** the run completes
**Then** a structured log is available showing: each stage name, agent type dispatched, duration of agent execution, exit code, and a summary of output (first/last N lines of stdout). This log is written to stdout and optionally to a file.

---

## Epic Components

- **[Scope Boundaries](./scope.md)** - Detailed out-of-scope items and future considerations
- **[Requirements](./requirements.md)** - Functional and non-functional requirements catalog

---

## Quick Reference

**Primary Users**: Developers using shark for AI-assisted development workflows

**Key Capabilities**:
- Go-based orchestration loop that mechanically enforces workflow stage execution
- Multi-provider agent dispatch (Claude CLI, Codex CLI) via os/exec
- Exit code gating -- status advances only on agent success
- Disallowed tool enforcement to prevent agents from self-advancing
- Structured run logging for debugging and audit

**Success Criteria**:
- Zero status transitions without corresponding agent execution
- 100% of agent failures prevent status advancement
- End-to-end workflow completion demonstrated in integration test

**Timeline**: No external deadline. Internal priority: high.

---

## Open Questions & Assumptions

No open questions -- all epic-level decisions are resolved.

---

*Last Updated*: 2026-03-21
