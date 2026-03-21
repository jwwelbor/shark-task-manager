# E22 UAT Plan: External Orchestration Runner

**Epic**: E22 - External Orchestration Runner - shark run subcommand
**Date**: 2026-03-21
**Status**: Accepted

---

## 1. UAT Scenarios from Epic Success Criteria

Each UAT scenario maps directly to the success criteria defined in the epic PRD (Section 2) and the high-level acceptance criteria (Section 6).

---

### UAT-1: Full Workflow Execution (End-to-End)

**Maps to**: Success Criteria "End-to-end workflow completion" and Epic UAT-1

**Preconditions**:
- A task exists in `draft` status with the advanced workflow profile configured
- All workflow statuses have `orchestrator_action` defined in `.sharkconfig.json` / `.sharkworkflow.json`
- Required CLI tools (claude/codex) are available on PATH

**Scenario**: Run a task through its complete workflow via `shark run`.

**What to Verify**:
1. The task progresses through each workflow stage sequentially (draft -> ready_for_refinement_ba -> in_refinement_ba -> ... -> completed)
2. An agent process is launched for every `spawn_agent` action
3. Status advances only after each agent exits with code 0
4. The task reaches `completed` status
5. All intermediate statuses are recorded in entity history (`shark status history <key>`)
6. No status transitions are skipped

**Traceability**: REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-004

---

### UAT-2: Stage Skip Prevention

**Maps to**: Success Criteria "Stage skip prevention" (0 skips per run) and "Exit code enforcement" (100%) and Epic UAT-2

**Preconditions**:
- A task in `ready_for_development` status
- `shark run` dispatches a Claude agent for the development stage

**Scenario**: Verify that the dispatched agent cannot advance status forward.

**What to Verify**:
1. The Claude CLI is invoked with `--disallowedTools` that blocks `shark status advance`, `shark task next-status`, `shark status set`, and `shark task set-status`
2. If the agent attempts to call a blocked command, the call fails within the agent session
3. Only the `shark run` process advances status after the agent exits
4. The task's status history shows exactly one transition per stage (no double-advances)

**Traceability**: REQ-F-005, REQ-NF-003

---

### UAT-3: Agent Failure Handling

**Maps to**: Success Criteria "Failure handling" (100% of failure cases handled) and Epic UAT-3

**Preconditions**:
- A task in `in_development` status
- `shark run` dispatches an agent that fails (exits non-zero)

**Scenario**: Verify that agent failure prevents status advancement.

**What to Verify**:
1. The task status does NOT advance when the agent exits non-zero
2. The run logs capture the agent's stdout and stderr
3. The run stops with a clear error message indicating which stage failed
4. The error message includes: status name, agent type, exit code, and output summary
5. The task remains in `in_development` status (verified via `shark get <key> --field status`)
6. Re-running `shark run` resumes from the same status

**Traceability**: REQ-F-003, REQ-F-004, REQ-NF-002

---

### UAT-4: Codex Red-Team Dispatch

**Maps to**: Success Criteria "Agent execution verification" (100%) and Epic UAT-4

**Preconditions**:
- A task in `ready_for_qa` status
- The workflow template specifies `provider: "openai"` and `agent_type: "qa"` in the orchestrator action
- The `codex` CLI is installed on PATH

**Scenario**: Verify that the correct agent provider is dispatched based on orchestrator action configuration.

**What to Verify**:
1. The Codex CLI is invoked (not Claude CLI) when `provider` is "openai"
2. The correct model is passed via `-m` flag
3. The sandbox mode is set via `--full-auto`
4. The rendered instruction template is passed to Codex
5. Status advances only after Codex exits with code 0

**Traceability**: REQ-F-006, REQ-F-002

---

### UAT-5: Pause Action Handling

**Maps to**: Epic UAT-5

**Preconditions**:
- A task in a status where the workflow template specifies `action: "pause"`
- `shark run` is executing

**Scenario**: Verify that the run loop handles pause actions correctly.

**What to Verify**:
1. The run stops gracefully (not as an error)
2. A message is output: "Run paused at status [status]. Manual intervention required."
3. The task remains in the current status (no advancement)
4. The run exits with code 0 (pause is not a failure)
5. After manual intervention (e.g., human advances status), re-running `shark run` continues from the new status

**Traceability**: REQ-F-008

---

### UAT-6: Resume After Interruption

**Maps to**: Success Criteria "Crash recovery" and Epic UAT-6

**Preconditions**:
- A `shark run` process was interrupted (SIGINT, SIGTERM, or machine restart) while a task was in `in_code_review` status

**Scenario**: Verify that re-running `shark run` resumes from the persisted status.

**What to Verify**:
1. Re-running `shark run <key>` reads the task's current database status
2. The run resumes from `in_code_review` (not from the beginning)
3. The agent is dispatched for the `in_code_review` stage
4. The workflow continues from that point to completion
5. No previously completed stages are re-executed

**Traceability**: REQ-NF-002

---

### UAT-7: Missing CLI Tool Detection

**Maps to**: Epic UAT-7

**Preconditions**:
- The workflow template requires dispatching to `codex` but `codex` is not installed
- `shark run` is started

**Scenario**: Verify that missing CLI tools are detected before dispatch.

**What to Verify**:
1. When `shark run` reaches a stage requiring a missing CLI tool, it fails immediately
2. The error message clearly identifies which tool is missing: "[tool] CLI not found on PATH. Install [tool] to continue."
3. The task status does not advance
4. The detection happens at dispatch time (lazy validation), not at run startup

**Traceability**: REQ-F-007

---

### UAT-8: Run Logging

**Maps to**: Success Criteria "Agent output capture" (100%) and Epic UAT-8

**Preconditions**:
- A task is driven through multiple workflow stages by `shark run`

**Scenario**: Verify that structured logs are produced for each stage.

**What to Verify**:
1. Each stage log entry includes: timestamp, status, agent_type, provider, duration, exit_code
2. Agent output (first and last 20 lines of stdout) is included in the default log
3. Full agent output is available when `--verbose` is used
4. A run summary is output at the end showing: all stages, total duration, final status, termination reason
5. The log output is parseable (for `--json` mode)

**Traceability**: REQ-F-011, REQ-F-012

---

### UAT-9: Advance Status Action (Auto-Advance)

**Preconditions**:
- A status in the workflow where `orchestrator_action.action` is `"advance_status"`

**Scenario**: Verify that `advance_status` actions skip agent dispatch and advance immediately.

**What to Verify**:
1. No agent process is launched for this status
2. Status advances to the next status automatically
3. The advancement is logged
4. The loop continues to the next status after advancement

**Traceability**: REQ-F-009

---

### UAT-10: Cascade Action

**Preconditions**:
- A status where `orchestrator_action.action` is `"cascade"`
- The entity has child entities (e.g., epic with features, feature with tasks)

**Scenario**: Verify that cascade actions trigger child entity status updates.

**What to Verify**:
1. The cascade service method is called
2. Affected child entities are logged
3. The loop continues after the cascade

**Traceability**: REQ-F-010

---

## 2. Cross-Feature Integration Scenarios

### INT-1: Run with Existing Workflow Templates

**Scenario**: `shark run` works with the existing `shark-templates/task/*.tmpl` template files without modification.

**What to Verify**:
- Template variables (`{task_key}`, `{task_title}`, `{task_status}`, etc.) are populated correctly
- Template partials (`_tdd_process`, etc.) are rendered
- The instruction passed to the agent CLI matches what `shark get <key> --json` would show in `orchestrator_action.instruction`

### INT-2: Run with Entity Polymorphism

**Scenario**: `shark run` works for tasks, features, and epics with their respective workflow configurations.

**What to Verify**:
- `shark run E07-F01-001` uses the task workflow
- `shark run E07-F01` uses the feature workflow
- `shark run E07` uses the epic workflow
- Each uses the correct level-scoped workflow service (`ForLevel("task")`, `ForLevel("feature")`, `ForLevel("epic")`)

### INT-3: Run with Status History

**Scenario**: All status transitions performed by `shark run` are recorded in entity history.

**What to Verify**:
- `shark status history <key>` shows all transitions after a run
- Each transition has the correct from/to statuses
- The `changed_by` field indicates the run process (or agent type)

### INT-4: Backward Compatibility

**Scenario**: Existing CLI commands continue to work unchanged after E22 is merged.

**What to Verify**:
- `shark status advance <key>` still works as before
- `shark get <key> --json` still returns orchestrator_action in the response
- `shark status set <key> <status>` still works for manual status changes
- All existing CLI tests pass without modification

---

## 3. Performance Considerations

### PERF-1: Loop Overhead

**What to Verify**: The time between an agent process exiting and the next agent process launching is less than 500ms.

**How to Measure**: Compare timestamps between consecutive stage log entries, subtracting agent execution duration.

**Target**: < 500ms per inter-stage gap (REQ-NF-001).

### PERF-2: Memory Usage

**What to Verify**: `shark run` does not accumulate significant memory during long runs (many stages).

**How to Measure**: Monitor RSS during a full workflow run. Agent stdout/stderr buffers should be bounded (captured but not all held simultaneously in memory for the entire run).

**Target**: Memory usage remains stable regardless of number of stages executed.

---

## 4. Security Considerations

### SEC-1: Forward Advancement Blocking

**What to Verify**: No agent dispatched by `shark run` can advance the entity's status forward. This is the core security invariant of the entire epic.

**Test Approach**:
- Inspect the Claude CLI command args to confirm `--disallowedTools` includes all status-advancing commands
- Inspect the Codex dispatch to confirm sandbox mode is enabled
- Attempt to call `shark status advance` from within an agent session and verify it is blocked

**Target**: Zero forward status advancements by agents (REQ-NF-003).

### SEC-2: No Secret Exposure in Logs

**What to Verify**: Run logs do not contain authentication tokens, API keys, or other sensitive environment variables.

**Test Approach**:
- Review log output for patterns matching tokens (Bearer, eyJ..., etc.)
- Verify that the full agent command logged does not include auth tokens

**Target**: Zero secret exposure (REQ-NF-004).

### SEC-3: Backward Transition Availability

**What to Verify**: Agents CAN still perform backward transitions (e.g., setting status to `changes_requested` with a reason) for legitimate rejection workflows.

**Test Approach**:
- Verify `--disallowedTools` does NOT block `shark status set <key> changes_requested --reason "..."` (backward transitions are permitted)
- Confirm the orchestrator templates instruct agents on how to reject/send backward when appropriate

---

## 5. Requirement Traceability Matrix

| Requirement | UAT Scenario | Coverage |
|-------------|-------------|----------|
| REQ-F-001 (Run Command Entry Point) | UAT-1, INT-2 | Full |
| REQ-F-002 (Orchestrator Action Reading) | UAT-1, UAT-4, INT-1 | Full |
| REQ-F-003 (Status Advancement Gate) | UAT-1, UAT-2, UAT-3 | Full |
| REQ-F-004 (Loop Termination) | UAT-1, UAT-3, UAT-5 | Full |
| REQ-F-005 (Claude CLI Dispatch) | UAT-2, UAT-1 | Full |
| REQ-F-006 (Codex CLI Dispatch) | UAT-4 | Full |
| REQ-F-007 (CLI Tool Validation) | UAT-7 | Full |
| REQ-F-008 (Pause Action) | UAT-5 | Full |
| REQ-F-009 (Advance Status Action) | UAT-9 | Full |
| REQ-F-010 (Cascade Action) | UAT-10 | Full |
| REQ-F-011 (Stage Logging) | UAT-8 | Full |
| REQ-F-012 (Run Summary) | UAT-8 | Full |
| REQ-F-013 (Worktree Isolation) | Not UAT-tested (Should Have) | Deferred |
| REQ-NF-001 (Loop Overhead) | PERF-1 | Full |
| REQ-NF-002 (Crash Recovery) | UAT-6 | Full |
| REQ-NF-003 (Agent Isolation) | UAT-2, SEC-1 | Full |
| REQ-NF-004 (No Secret Exposure) | SEC-2 | Full |
| REQ-NF-005 (Dispatcher Extensibility) | INT-2 (implicit via two implementations) | Partial |

**No orphaned requirements.** Every Must Have and Should Have functional requirement maps to at least one UAT scenario. REQ-F-013 (worktree) is a Should Have and is deferred for separate verification once implemented.

---

*UAT Plan approved: 2026-03-21*
