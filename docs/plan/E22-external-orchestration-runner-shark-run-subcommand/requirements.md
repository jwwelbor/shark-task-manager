# Requirements

**Epic**: [External Orchestration Runner - shark run subcommand](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for E22. Each requirement maps to the acceptance criteria defined in the [epic PRD](./epic.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for launch; epic fails without these
- **Should Have**: Important but workarounds exist; target for initial release
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

### Must Have Requirements

#### Core Run Loop

**REQ-F-001**: Run Command Entry Point
- **Description**: `shark run <entity-key>` accepts a task, feature, or epic key and initiates the orchestration loop for that entity.
- **Acceptance Criteria**:
  - [ ] Command accepts task keys (E07-F01-001), feature keys (E07-F01), and epic keys (E07)
  - [ ] Command validates the entity exists before starting the loop
  - [ ] Command reads the entity's current status from the database
  - [ ] Invalid keys produce a clear error message and exit code 1

**REQ-F-002**: Orchestrator Action Reading
- **Description**: For each status in the workflow, `shark run` reads the `orchestrator_action` from the workflow configuration to determine what action to take.
- **Acceptance Criteria**:
  - [ ] Reads `action`, `agent_type`, `provider`, `model`, `skills`, and `instruction_template` from the entity's current status configuration
  - [ ] Renders the `instruction_template` with entity context (key, title, status, description)
  - [ ] Handles missing orchestrator_action gracefully (logs warning, pauses)

**REQ-F-003**: Status Advancement Gate
- **Description**: Status advances ONLY after the dispatched agent process exits with code 0. Non-zero exit codes prevent advancement.
- **Acceptance Criteria**:
  - [ ] Exit code 0 triggers status advancement via the existing service layer
  - [ ] Non-zero exit codes do NOT advance status
  - [ ] Non-zero exit codes log the failure with agent stdout/stderr
  - [ ] The run loop stops on non-zero exit (no automatic retry by default)

**REQ-F-004**: Loop Termination
- **Description**: The run loop terminates when the entity reaches a terminal status, a pause action is encountered, or an agent fails.
- **Acceptance Criteria**:
  - [ ] Terminates on terminal statuses: `completed`, `cancelled`
  - [ ] Terminates on `pause` action with a clear "paused" message
  - [ ] Terminates on agent failure with error details
  - [ ] Terminates on `archive` action
  - [ ] Outputs a summary of stages completed when the loop ends

#### Agent Dispatch

**REQ-F-005**: Claude CLI Dispatch
- **Description**: When the orchestrator action specifies Claude as the provider (or defaults to Claude), invoke `claude -p` with the rendered instruction template.
- **Acceptance Criteria**:
  - [ ] Invokes `claude -p "<instruction>"` via os/exec
  - [ ] Passes `--disallowedTools "Bash(shark status advance*)"` to prevent self-advancement
  - [ ] Passes `--allowedTools` if specified in the orchestrator action
  - [ ] Passes `--max-turns` if specified in the orchestrator action
  - [ ] Captures stdout and stderr from the process
  - [ ] Returns the process exit code to the run loop

**REQ-F-006**: Codex CLI Dispatch
- **Description**: When the orchestrator action specifies OpenAI/Codex as the provider, invoke `codex exec` with the rendered instruction template.
- **Acceptance Criteria**:
  - [ ] Invokes `codex exec -m <model> -s <sandbox> "<instruction>"` via os/exec
  - [ ] Passes `--skip-git-repo-check` flag
  - [ ] Passes custom configuration flags (e.g., `-c model_reasoning_effort=high`)
  - [ ] Captures stdout and stderr from the process
  - [ ] Returns the process exit code to the run loop

**REQ-F-007**: CLI Tool Validation
- **Description**: Before dispatching to an agent CLI, `shark run` verifies the required CLI tool is available on the system PATH.
- **Acceptance Criteria**:
  - [ ] Checks for `claude` binary when Claude dispatch is needed
  - [ ] Checks for `codex` binary when Codex dispatch is needed
  - [ ] Fails with a clear error message if the tool is missing: "[tool] CLI not found on PATH"
  - [ ] Validation occurs before dispatch, not at run startup (lazy validation)

#### Non-spawn_agent Actions

**REQ-F-008**: Pause Action Handling
- **Description**: When the orchestrator action is `pause`, the run loop stops and outputs a message indicating the entity is paused.
- **Acceptance Criteria**:
  - [ ] Does not advance status
  - [ ] Outputs: "Run paused at status [status]. Manual intervention required."
  - [ ] Exits with code 0 (pause is not a failure)

**REQ-F-009**: Advance Status Action Handling
- **Description**: When the orchestrator action is `advance_status`, the run loop advances status without dispatching an agent.
- **Acceptance Criteria**:
  - [ ] Advances to next status immediately (no agent dispatch)
  - [ ] Logs the automatic advancement
  - [ ] Continues the loop after advancement

**REQ-F-010**: Cascade Action Handling
- **Description**: When the orchestrator action is `cascade`, the run loop triggers a status cascade to child entities.
- **Acceptance Criteria**:
  - [ ] Calls the existing cascade service method
  - [ ] Logs which child entities were affected
  - [ ] Continues the loop after cascade

---

### Should Have Requirements

#### Run Logging

**REQ-F-011**: Structured Stage Logging
- **Description**: Each stage execution is logged with structured data: stage name, agent type, duration, exit code, and output summary.
- **Acceptance Criteria**:
  - [ ] Logs are written to stdout in a human-readable format
  - [ ] Each log entry includes: timestamp, status, agent_type, duration, exit_code
  - [ ] Agent output (first and last 20 lines of stdout) is included in the log
  - [ ] Full agent output is available via `--verbose` flag

**REQ-F-012**: Run Summary
- **Description**: When the run loop completes (success, pause, or failure), output a summary of all stages executed.
- **Acceptance Criteria**:
  - [ ] Lists each status transition with duration
  - [ ] Shows total run duration
  - [ ] Indicates final status and reason for termination
  - [ ] Includes agent types used at each stage

#### Worktree Support

**REQ-F-013**: Git Worktree Isolation
- **Description**: When dispatching an agent, `shark run` can optionally create a git worktree for the agent to work in, preventing conflicts with the main working tree.
- **Acceptance Criteria**:
  - [ ] Creates a worktree in `.claude/worktrees/<run-id>/` before agent dispatch
  - [ ] Passes the worktree path as the working directory for the agent process
  - [ ] Cleans up the worktree after the agent exits (configurable)
  - [ ] Can be disabled via `--no-worktree` flag

---

### Could Have Requirements

**REQ-F-014**: Single Retry on Failure
- **Description**: A `--retry` flag allows `shark run` to retry a failed stage once before stopping.
- **Acceptance Criteria**:
  - [ ] `--retry` flag causes one retry attempt on agent failure
  - [ ] Retry is logged distinctly from the initial attempt
  - [ ] Second failure stops the run

**REQ-F-015**: Dry Run Mode
- **Description**: `shark run --dry-run` shows what actions would be taken without executing agents or advancing status.
- **Acceptance Criteria**:
  - [ ] Lists each status and the action that would be taken
  - [ ] Shows the agent command that would be executed
  - [ ] Does not modify any state

**REQ-F-016**: Feature-Level Run
- **Description**: `shark run E07-F01` drives all tasks in a feature through their workflows sequentially.
- **Acceptance Criteria**:
  - [ ] Discovers all tasks in the feature ordered by execution_order
  - [ ] Runs each task through its workflow sequentially
  - [ ] Stops on first task failure

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: Run Loop Overhead
- **Description**: The `shark run` loop itself adds minimal overhead between stages.
- **Measurement**: Time between agent process exit and next agent process launch.
- **Target**: Less than 500ms between stages (excluding status advance database write).
- **Justification**: The bottleneck is agent execution time (minutes), not loop overhead.

### Reliability

**REQ-NF-002**: Crash Recovery via Status Persistence
- **Description**: If `shark run` is interrupted, re-running the command resumes from the entity's current database status.
- **Measurement**: After killing `shark run` mid-stage and re-running, the workflow continues from the correct status.
- **Target**: 100% correct resumption.
- **Justification**: Status is the only persistent state. In-memory run metadata is disposable.

### Security

**REQ-NF-003**: Agent Isolation from Status Advancement
- **Description**: Dispatched agents must not be able to advance the entity's status forward.
- **Measurement**: Claude CLI receives `--disallowedTools` blocking `shark status advance` and `shark task next-status`. Codex runs in read-only or restricted sandbox.
- **Target**: Zero forward status advancements by agents during `shark run`.
- **Justification**: This is the core security property of the entire epic.

**REQ-NF-004**: No Credential Exposure in Logs
- **Description**: Agent output logs must not contain authentication tokens, API keys, or other secrets.
- **Measurement**: Log output is sanitized or truncated to exclude environment variables and auth headers.
- **Target**: Zero secret exposure in log output.

### Maintainability

**REQ-NF-005**: Dispatcher Interface Extensibility
- **Description**: The agent dispatch mechanism uses a Go interface so new providers can be added without modifying the core run loop.
- **Measurement**: Adding a new provider requires implementing one interface (no changes to run loop code).
- **Target**: Interface-based dispatch with at least two implementations (Claude, Codex).
- **Justification**: Future providers should not require refactoring the core loop.

---

*See also*: [Scope](./scope.md)
