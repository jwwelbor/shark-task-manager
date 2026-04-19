---
feature_key: E07-F41-run-command-execution-logging
epic_key: E07
title: Run command execution logging
description: Structured per-stage slog events for `shark /run`, with lean-by-default agent subprocess logging and opt-in transcript capture.
---

# Run command execution logging

**Feature Key**: E07-F41-run-command-execution-logging

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem

E07-F40 wired slog output to `shark.log`, but the file stays empty during real runs because shark's code base emits almost no structured log events — 343 `fmt.Print*` call sites vs. 9 `slog` calls. When a `/run` invocation misbehaves (agent exits non-zero, workflow pauses unexpectedly, wrong status transition) the operator has no machine-parsable record of what happened. They can re-read the terminal scrollback, but the subprocess command line, exit codes, durations, and the actual `claude`/`codex` stderr are lost the moment the scrollback rotates.

### Solution

Instrument the `/run` execution stack with structured slog events at every meaningful step — command entry, per-stage dispatch, status transitions, errors, and final outcome. Always capture the agent subprocess command line, exit code, and duration. On failure, include the agent's stderr (truncated). Behind an opt-in config flag, mirror the full agent stdout/stderr to per-dispatch transcript files on disk and reference them from the slog event, so forensic depth is available without drowning the main log.

### Impact

- `shark.log` becomes a usable execution trace: every `/run` produces a grep-friendly record of what ran, what it returned, and how long it took.
- Failed runs surface `stderr` inline (up to 4 KB) so root cause is visible without re-running.
- Forensic mode (opt-in) preserves the full agent transcript on disk, keyed by run ID and stage.

---

## User Personas

### Persona 1: Shark Operator (primary)

**Profile**:
- **Role**: Developer using shark to drive feature work through agent-based workflows.
- **Experience Level**: Comfortable with CLI tooling and reading log files.

**Goals**:
1. Understand what happened during a `/run` invocation after the fact.
2. Diagnose agent failures without re-running the workflow.
3. Grep the log for all invocations of a given agent, provider, or status.

**Pain Points**:
- Terminal scrollback is ephemeral.
- No way to see what command line was actually executed against `claude` or `codex`.
- Failed agents report "agent exited with code N" with no visible stderr.

**Success Looks Like**:
After a `/run` fails, the operator opens `shark.log`, finds the failing stage's event, reads the command line and stderr, and fixes the problem — all without re-running.

---

## User Stories

### Must-Have Stories

**Story 1**: As a shark operator, I want every `/run` invocation to emit a `run.start` and `run.end` event so that I can see when runs began and ended and what the overall outcome was.

**Acceptance Criteria**:
- [ ] `run.start` is emitted at the top of `runRun` with `command`, `args`, `entity_key`, `entity_type`, `dry_run`, `worktree`, `worktree_path`, `run_id`.
- [ ] `run.end` is emitted on every return path (success and error) with `entity_key`, `outcome`, `final_status`, `stages_completed`, `duration_ms`, `error`.
- [ ] Both events appear in `shark.log` when `observability.enabled: true` and `log_file` is configured.

**Story 2**: As a shark operator, I want every workflow stage to emit structured events so that I can walk through the execution stack.

**Acceptance Criteria**:
- [ ] `run.stage.start` emitted at the top of each controller loop iteration with `entity_key`, `status`, `iteration`.
- [ ] `run.stage.dispatch` emitted immediately before spawning an agent with `entity_key`, `status`, `agent_type`, `provider`, `command`.
- [ ] `run.stage.complete` emitted after successful dispatch with `entity_key`, `status`, `agent_type`, `provider`, `exit_code`, `duration_ms`, `next_status`.
- [ ] `run.stage.transition` emitted on `handleAdvanceStatus` with `entity_key`, `from_status`, `to_status`.

**Story 3**: As a shark operator, I want failed agent dispatches to include stderr so that I can diagnose without re-running.

**Acceptance Criteria**:
- [ ] `run.stage.error` emitted at every `result.Error = ...` branch in the controller.
- [ ] When the failure originated from a non-zero agent exit, the event includes `stderr` (truncated to 4 KB) and `stdout_tail` (last 4 KB of stdout).
- [ ] Truncation is indicated with a `truncated: true` field on the event.
- [ ] Command line is always included so the failing invocation is reproducible.

**Story 4**: As a shark operator, when I enable transcript capture, I want the full agent stdout/stderr persisted to disk and referenced from the slog event so that I have a forensic record without bloating the main log.

**Acceptance Criteria**:
- [ ] New config key `observability.capture_agent_transcripts` (bool, default `false`).
- [ ] When enabled, every dispatch writes full stdout and stderr to `.shark/runs/{run_id}/{stage_n}-{status}-{provider}.log`.
- [ ] The `run.stage.complete` and `run.stage.error` events include `transcript_path` (relative to project root).
- [ ] Transcript directory is created with mode 0755; files with mode 0644.
- [ ] When disabled, no transcript files are written and no `transcript_path` field appears in events.

### Should-Have Stories

**Story 5**: As a shark operator, I want a `run_id` correlation field on every event in a given `/run` invocation so that I can filter the log to a single run.

**Acceptance Criteria**:
- [ ] A UUID or short random string is generated at `runRun` entry and included as `run_id` in every event emitted for that invocation.
- [ ] The same `run_id` is used to name the transcript subdirectory.

### Edge Case & Error Stories

**Error Story 1**: As a shark operator, if the transcript directory cannot be created (permissions, disk full), I want the run to continue normally and a warning logged.

**Acceptance Criteria**:
- [ ] Transcript write failures do not abort the `/run`.
- [ ] A single `run.transcript.warning` event is emitted with the error.
- [ ] Subsequent stages in the same run skip transcript writes for that run.

**Error Story 2**: As a shark operator, if `observability.enabled: false`, no events should be emitted regardless of other config.

**Acceptance Criteria**:
- [ ] With observability disabled, zero events reach the slog default logger (existing F40 behavior preserved).

---

## Requirements

### Functional Requirements

**Category: Event Emission**

1. **REQ-F-001**: Command-entry event
   - **Description**: Emit `run.start` at the top of `runRun` in `internal/cli/commands/run.go`.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Fields: `command`, `args`, `entity_key`, `entity_type`, `dry_run`, `worktree`, `worktree_path`, `run_id`.
     - [ ] Level: `INFO`.

2. **REQ-F-002**: Command-exit event
   - **Description**: Emit `run.end` on all return paths from `runRun`.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Fields: `entity_key`, `outcome` (one of `completed|paused|failed|cancelled`), `final_status`, `stages_completed`, `duration_ms`, `error`.
     - [ ] Level: `INFO` on success, `ERROR` on failure.

3. **REQ-F-003**: Per-stage lifecycle events
   - **Description**: Emit `run.stage.start`, `run.stage.dispatch`, `run.stage.complete`, `run.stage.transition` from the controller loop.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] See Story 2 field lists.
     - [ ] Level: `INFO`.
     - [ ] `command` field in `run.stage.dispatch` contains the exact shell-equivalent invocation used by `execAndCapture`.

4. **REQ-F-004**: Error events
   - **Description**: Emit `run.stage.error` at every controller branch that sets `result.Error`.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Level: `ERROR`.
     - [ ] Always includes `entity_key`, `status`, `phase`, `error`, `run_id`.
     - [ ] When error originates from `*AgentFailedError`, includes `exit_code`, `stderr` (truncated 4 KB), `stdout_tail` (last 4 KB), `command`, `truncated`.

**Category: Agent Subprocess Capture**

5. **REQ-F-010**: Lean dispatch logging
   - **Description**: By default, log only `command`, `exit_code`, `duration_ms` for successful dispatches; never inline `stdout`.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Successful dispatches produce a log event under ~1 KB.
     - [ ] `stdout` never appears in `shark.log` at any log level unless the run fails.

6. **REQ-F-011**: Error-path stderr capture
   - **Description**: When an agent dispatch fails, include truncated stderr and stdout tail in the error event.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Both fields capped at 4 KB (configurable via `observability.log_truncate_bytes`, default 4096).
     - [ ] Truncation adds a `truncated: true` field; non-truncated events omit it.

7. **REQ-F-012**: Opt-in transcript files
   - **Description**: When `observability.capture_agent_transcripts: true`, mirror full stdout and stderr to per-dispatch files on disk.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Path: `{project_root}/.shark/runs/{run_id}/{stage_n}-{status}-{provider}.log`.
     - [ ] File contents: `COMMAND: <cmd>\nEXIT: <code>\nDURATION: <ms>ms\n---STDOUT---\n<stdout>\n---STDERR---\n<stderr>`.
     - [ ] Event includes `transcript_path` as a project-relative path.
     - [ ] Failure to write transcript is non-fatal; logs `run.transcript.warning` once per run.

**Category: Configuration**

8. **REQ-F-020**: New config key
   - **Description**: Add `observability.capture_agent_transcripts` (bool, default false) and `observability.log_truncate_bytes` (int, default 4096) to `.sharkconfig.json` schema.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Defaults applied when keys are absent.
     - [ ] `shark admin config validate` accepts both keys.

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Logging overhead
   - **Target**: Per-stage event emission adds < 5ms overhead vs. baseline `/run` execution.
   - **Measurement**: Benchmark a dry-run `/run` with and without observability enabled; compare wall time.

2. **REQ-NF-002**: Transcript write non-blocking on success path
   - **Target**: Transcript writes happen synchronously but should not add > 20ms per dispatch on local disk.
   - **Measurement**: Benchmark a single dispatch with transcript enabled vs. disabled.

**Reliability**

3. **REQ-NF-010**: No data loss on crash
   - **Description**: Events are emitted to the log file synchronously so a crashed run leaves a complete trace up to the point of failure.
   - **Implementation**: Rely on existing slog file handler opened with `O_APPEND|O_CREATE|O_WRONLY`; no buffering layer.

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Successful run produces walkable trace**
- **Given** observability is enabled with `log_file: "shark.log"` and `capture_agent_transcripts: false`
- **When** `/run E01-F02-003` completes a 4-stage workflow successfully
- **Then** `shark.log` contains exactly one `run.start`, one `run.end`, four `run.stage.start`, four `run.stage.dispatch`, four `run.stage.complete`, and three `run.stage.transition` events, all sharing the same `run_id`
- **And** no `stdout` field appears in any event

**Scenario 2: Failing agent dispatch surfaces stderr**
- **Given** observability is enabled
- **When** an agent exits with code 1 and prints "invalid prompt" to stderr
- **Then** `shark.log` contains a `run.stage.error` event with `phase: "dispatch"`, `exit_code: 1`, `stderr: "invalid prompt"`, and the full agent command line
- **And** no transcript file is written (capture disabled)

**Scenario 3: Transcript capture produces forensic files**
- **Given** `capture_agent_transcripts: true`
- **When** `/run E01-F02-003` dispatches an agent that prints 50 KB of JSON to stdout
- **Then** `.shark/runs/{run_id}/1-ready_for_research-claude.log` is created containing the full stdout and stderr
- **And** the `run.stage.complete` event includes `transcript_path: ".shark/runs/{run_id}/1-ready_for_research-claude.log"`
- **And** `shark.log` does not contain the 50 KB stdout inline

**Scenario 4: Transcript write failure is non-fatal**
- **Given** `capture_agent_transcripts: true` and `.shark/runs/` is read-only
- **When** `/run E01-F02-003` runs
- **Then** the run completes normally
- **And** a single `run.transcript.warning` event is emitted
- **And** events for subsequent stages omit `transcript_path`

---

## Out of Scope

### Explicitly Excluded

1. **Teeing shark's own terminal output (stream #1) to `shark.log`**
   - **Why**: The visible stdout (`pterm.Success`, `fmt.Printf`) is already reconstructable from structured events. Teeing would double-log without adding information.
   - **Future**: If a future use case needs a verbatim terminal transcript, it can be added as a separate feature.

2. **Log rotation for `shark.log` or `.shark/runs/`**
   - **Why**: Out of scope for this feature. Users can rotate manually or via external tools (logrotate).
   - **Future**: Consider a `shark admin log prune` command in a later feature.

3. **Emitting equivalent events for non-`/run` commands**
   - **Why**: This feature is scoped to `/run` specifically. Instrumenting the other ~60 commands is a separate, larger effort.
   - **Future**: A general "CLI command observability" feature could follow.

### Alternative Approaches Rejected

**Alternative 1: Inline full agent stdout/stderr in `shark.log` on every dispatch**
- **Description**: Every `run.stage.complete` event carries the full subprocess output.
- **Why Rejected**: Claude `--output-format json` emits tens of KB per call; a typical run produces 1+ MB per `shark.log` entry. Makes the log unusable for grep and drowns the structured events.

**Alternative 2: Stream agent output to stdout via `cmd.Stdout = os.Stdout` as it runs**
- **Description**: Let the operator see live agent output during `/run`.
- **Why Rejected**: Breaks the current UX where shark owns the terminal and prints a clean summary. Also not what the operator asked for ("I don't need to see the output for most things").

---

## Success Metrics

### Primary Metrics

1. **Log completeness**
   - **What**: Every successful `/run` produces a `shark.log` trace from which a human can reconstruct the stage sequence without consulting the terminal.
   - **Target**: 100% of runs produce start + end events; 100% of dispatches produce dispatch + complete events.
   - **Measurement**: Post-feature test: parse `shark.log` after 10 runs, verify all events present.

2. **Error diagnosability**
   - **What**: Failed runs surface the failing command line and stderr in the log.
   - **Target**: For any agent-exit-code failure, operator can identify the failing command and read the stderr from `shark.log` alone.
   - **Measurement**: Simulate 3 failure modes; verify log contains sufficient context in each case.

---

## Dependencies & Integrations

### Dependencies

- **E07-F40**: `observability.InitLoggerWithRoot` — file destination and log handler already wired. F41 adds the call sites that populate the file.
- **`internal/runner/controller.go`**: Owns the workflow loop; all stage events fire from here.
- **`internal/runner/dispatcher.go:execAndCapture`**: Already captures subprocess command, stdout, stderr, exit code, duration — F41 consumes this data.
- **`internal/cli/commands/run.go`**: Owns the `runRun` entry/exit points.

### Integration Requirements

- **`.sharkconfig.json` schema**: Adds two new keys under `observability`.
- **No DB changes**.
- **No API changes**.

---

## Compliance & Security Considerations

- **Data Protection**: Transcript files may contain sensitive project content (source code, file paths). They are written under `.shark/runs/` which should be in `.gitignore` (verify and amend).
- **Audit**: The structured log becomes an audit trail of agent invocations — useful for security review but also means log rotation/retention needs to be documented for the operator.

---

*Last Updated*: 2026-04-18
