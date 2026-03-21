# E22-F01 Specification: Agent Dispatcher Interface and Claude Implementation

**Feature**: E22-F01 - Agent Dispatcher Interface and Claude Implementation
**Epic**: E22 - External Orchestration Runner
**Date**: 2026-03-21
**Status**: Draft

---

## 1. Purpose

This feature defines the `AgentDispatcher` Go interface used by the E22 run loop to invoke external AI agents, and provides the first concrete implementation: the Claude CLI dispatcher. It is the foundation for all other E22 features -- the run loop controller (E22-F02+) depends on this interface to dispatch work to agents.

For business context and motivation, see epic PRD Section 1 (Problem Statement) and Section 2 (Goals).

---

## 2. Functional Requirements

These requirements trace to epic-level requirements as noted.

### REQ-F01-001: AgentDispatcher Interface Definition

**Traces to**: Epic REQ-NF-005 (Dispatcher Interface Extensibility)

The `internal/runner/` package SHALL define an `AgentDispatcher` interface with exactly two methods:

```go
type AgentDispatcher interface {
    Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error)
    Name() string
}
```

**Acceptance Criteria**:
- [ ] Interface is defined in `internal/runner/dispatcher.go`
- [ ] `Dispatch` accepts `context.Context` and `DispatchInput`, returns `*DispatchResult` and `error`
- [ ] `Name` returns a human-readable identifier (e.g., `"claude"`)
- [ ] Interface is satisfied by at least one concrete implementation (Claude)

### REQ-F01-002: DispatchInput Data Type

**Traces to**: Epic REQ-F-005 (Claude CLI Dispatch)

The `DispatchInput` struct SHALL contain all information needed to invoke an agent for a single workflow stage.

**Acceptance Criteria**:
- [ ] Contains `Instruction string` -- the rendered instruction from the template engine
- [ ] Contains `WorkingDir string` -- the working directory for the agent process
- [ ] Contains `EntityKey string` -- the entity key for context/logging
- [ ] Contains `EntityType string` -- the entity type (task, feature, epic)
- [ ] Contains `Status string` -- the current workflow status being executed
- [ ] Contains `AgentType string` -- the agent type from orchestrator action (developer, qa, etc.)
- [ ] Contains `Model string` -- optional model override
- [ ] Contains `MaxTurns int` -- optional max turns limit (0 means no limit)
- [ ] Contains `AllowedTools []string` -- optional tool allowlist
- [ ] Contains `DisallowedTools []string` -- additional tools to disallow beyond defaults

### REQ-F01-003: DispatchResult Data Type

**Traces to**: Epic REQ-F-003 (Status Advancement Gate), REQ-F-011 (Structured Stage Logging)

The `DispatchResult` struct SHALL capture the outcome of an agent dispatch.

**Acceptance Criteria**:
- [ ] Contains `ExitCode int` -- process exit code (0 = success)
- [ ] Contains `Stdout string` -- captured stdout from the agent process
- [ ] Contains `Stderr string` -- captured stderr from the agent process
- [ ] Contains `Duration time.Duration` -- wall clock execution time
- [ ] Contains `Command string` -- the full command string that was executed (for logging/debugging)

### REQ-F01-004: Claude CLI Dispatcher Implementation

**Traces to**: Epic REQ-F-005 (Claude CLI Dispatch), REQ-NF-003 (Agent Isolation)

A `ClaudeDispatcher` struct SHALL implement `AgentDispatcher` by invoking the `claude` CLI tool via `os/exec`.

**Acceptance Criteria**:
- [ ] Invokes `claude -p "<instruction>"` with the rendered instruction from `DispatchInput.Instruction`
- [ ] Passes `--output-format json` for structured output
- [ ] Passes default disallowed tools to prevent self-advancement (see REQ-F01-005)
- [ ] Passes `--max-turns N` when `DispatchInput.MaxTurns > 0`
- [ ] Passes `--allowedTools` entries when `DispatchInput.AllowedTools` is non-empty
- [ ] Passes additional `--disallowedTools` entries from `DispatchInput.DisallowedTools`
- [ ] Uses `exec.CommandContext(ctx, "claude", args...)` for context cancellation support
- [ ] Captures stdout and stderr via pipes
- [ ] Returns `DispatchResult` with exit code, stdout, stderr, duration, and command string
- [ ] Sets `cmd.Dir` to `DispatchInput.WorkingDir` when non-empty

### REQ-F01-005: Default Disallowed Tools for Agent Isolation

**Traces to**: Epic REQ-NF-003 (Agent Isolation from Status Advancement)

The Claude dispatcher SHALL always include the following disallowed tools, regardless of `DispatchInput.DisallowedTools`:

```
Bash(shark status advance*)
Bash(shark task next-status*)
Bash(shark status set*)
Bash(shark task set-status*)
Bash(shark feature next-status*)
Bash(shark epic next-status*)
```

**Acceptance Criteria**:
- [ ] All six patterns are passed as `--disallowedTools` flags
- [ ] Patterns are additive with any user-specified `DisallowedTools` from the input
- [ ] Default patterns are defined as a package-level constant slice (not hardcoded inline)

### REQ-F01-006: CLI Tool Availability Check

**Traces to**: Epic REQ-F-007 (CLI Tool Validation)

The Claude dispatcher SHALL validate that the `claude` binary is available on the system PATH before attempting dispatch.

**Acceptance Criteria**:
- [ ] Uses `exec.LookPath("claude")` to check availability
- [ ] Returns a `*ToolNotFoundError` with a clear message when the tool is missing
- [ ] Validation occurs at dispatch time (lazy), not at construction time

### REQ-F01-007: ToolNotFoundError Custom Error Type

**Traces to**: Epic Section 7 (Error Handling Strategy)

A `ToolNotFoundError` struct SHALL be defined for missing CLI tool errors.

**Acceptance Criteria**:
- [ ] Contains `Tool string` (the tool name, e.g., "claude")
- [ ] Implements `error` interface with message format: `"<tool> CLI not found on PATH. Install <tool> to continue."`
- [ ] Can be matched with `errors.As()` by callers

### REQ-F01-008: AgentFailedError Custom Error Type

**Traces to**: Epic Section 7 (Error Handling Strategy)

An `AgentFailedError` struct SHALL be defined for non-zero agent exit codes.

**Acceptance Criteria**:
- [ ] Contains `ExitCode int`, `Stdout string`, `Stderr string`, `Command string`
- [ ] Implements `error` interface with message format including exit code and stderr summary
- [ ] Can be matched with `errors.As()` by callers

---

## 3. Non-Functional Requirements

### REQ-F01-NF-001: Dispatch Overhead

**Traces to**: Epic REQ-NF-001 (Run Loop Overhead)

The dispatcher overhead (time between receiving `DispatchInput` and launching the subprocess) SHALL be under 50ms. This excludes the agent process execution time.

### REQ-F01-NF-002: Output Capture Memory Bounds

Agent stdout and stderr SHALL be captured in memory. For this initial implementation, no size limit is enforced (the run logger in E22-F03+ will handle truncation). The dispatcher captures complete output.

### REQ-F01-NF-003: Context Cancellation

When the `context.Context` passed to `Dispatch` is cancelled (e.g., SIGINT), the agent subprocess SHALL be killed via the `exec.CommandContext` mechanism. The `Dispatch` method SHALL return `context.Canceled` or `context.DeadlineExceeded` as appropriate.

---

## 4. Out of Scope for This Feature

- **Codex CLI dispatcher** -- separate feature (E22-F02 or later). This feature only delivers the interface and the Claude implementation.
- **Run loop controller** -- separate feature. This feature only delivers the dispatcher interface and types that the controller will consume.
- **Run logging** -- the `RunLogger` is a separate feature. `DispatchResult` contains the raw data the logger needs.
- **Worktree management** -- separate feature. `DispatchInput.WorkingDir` is passed through but worktree creation/cleanup is not part of this feature.
- **Retry logic** -- the dispatcher does not retry. The run controller decides retry policy.
- **`PopulatedAction` extension with Provider/Model fields** -- while the epic architecture assumes `Provider` and `Model` exist on `PopulatedAction`, these fields do not currently exist in the codebase. Adding them to `OrchestratorAction` and `PopulatedAction` is a prerequisite task within this feature (see Architecture Section 5.2).

---

## 5. Architecture

### 5.1 Component Changes

#### New Files

| File | Description | Estimated Lines |
|------|-------------|-----------------|
| `internal/runner/dispatcher.go` | `AgentDispatcher` interface, `DispatchInput`, `DispatchResult`, error types | ~75 |
| `internal/runner/claude_dispatcher.go` | `ClaudeDispatcher` struct implementing `AgentDispatcher` | ~100 |
| `internal/runner/claude_dispatcher_test.go` | Unit tests for Claude dispatcher command construction | ~150 |
| `internal/runner/dispatcher_test.go` | Tests for error types and DispatchInput/DispatchResult | ~40 |

#### Modified Files

| File | Change | Reason |
|------|--------|--------|
| `internal/config/orchestrator_action.go` | Add `Provider` and `Model` fields to `OrchestratorAction` struct | Run loop needs to select dispatcher by provider and pass model to agent |
| `internal/config/action_service.go` | Add `Provider` and `Model` fields to `PopulatedAction` struct | These fields must propagate through the action service to the run controller |

### 5.2 Data Model Changes (Configuration Schema)

**No database changes.** The only data model change is to the `OrchestratorAction` JSON struct in `.sharkconfig.json`.

#### OrchestratorAction Struct Extension

Current struct (`internal/config/orchestrator_action.go:14-28`):

```go
type OrchestratorAction struct {
    Action              string   `json:"action" yaml:"action"`
    AgentType           string   `json:"agent_type,omitempty" yaml:"agent_type,omitempty"`
    Skills              []string `json:"skills,omitempty" yaml:"skills,omitempty"`
    InstructionTemplate string   `json:"instruction_template" yaml:"instruction_template"`
}
```

New fields to add:

```go
type OrchestratorAction struct {
    Action              string   `json:"action" yaml:"action"`
    AgentType           string   `json:"agent_type,omitempty" yaml:"agent_type,omitempty"`
    Provider            string   `json:"provider,omitempty" yaml:"provider,omitempty"`       // NEW: "anthropic" (default), "openai"
    Model               string   `json:"model,omitempty" yaml:"model,omitempty"`             // NEW: model override (e.g., "o3")
    Skills              []string `json:"skills,omitempty" yaml:"skills,omitempty"`
    InstructionTemplate string   `json:"instruction_template" yaml:"instruction_template"`
}
```

**Rationale**: The epic architecture (Section 5.2) specifies that the run controller selects a dispatcher based on the `Provider` field. Without this field on the config struct, the controller has no way to know which agent CLI to invoke. `Model` allows per-status model selection (e.g., using a cheaper model for simple statuses).

**Backward compatibility**: Both fields are `omitempty`. Existing configs without these fields continue to work. When `Provider` is empty, the run controller defaults to `"anthropic"` (Claude).

#### PopulatedAction Struct Extension

Current struct (`internal/config/action_service.go:33-38`):

```go
type PopulatedAction struct {
    Action      string   `json:"action"`
    AgentType   string   `json:"agent_type,omitempty"`
    Skills      []string `json:"skills,omitempty"`
    Instruction string   `json:"instruction"`
}
```

Add:

```go
type PopulatedAction struct {
    Action      string   `json:"action"`
    AgentType   string   `json:"agent_type,omitempty"`
    Provider    string   `json:"provider,omitempty"`    // NEW
    Model       string   `json:"model,omitempty"`       // NEW
    Skills      []string `json:"skills,omitempty"`
    Instruction string   `json:"instruction"`
}
```

The `GetStatusActionPopulated()` method in `DefaultActionService` must copy `Provider` and `Model` from the source `OrchestratorAction` to the `PopulatedAction`. This is a one-line addition in the existing population logic.

### 5.3 Interface Contracts

#### AgentDispatcher Interface

Defined in `internal/runner/dispatcher.go`. See REQ-F01-001 through REQ-F01-003 for exact type definitions.

The interface is consumed by:
- `RunController` (future E22 feature) -- calls `Dispatch()` per workflow stage
- Tests -- mock implementations for controller testing

The interface is implemented by:
- `ClaudeDispatcher` (this feature)
- `CodexDispatcher` (future E22 feature)

#### Dispatcher Selection

The run controller (future feature) will select a dispatcher using a `map[string]AgentDispatcher`:

```go
dispatchers := map[string]AgentDispatcher{
    "anthropic": claudeDispatcher,
    "":          claudeDispatcher, // Default when provider is empty
    "openai":    codexDispatcher,
}
```

This feature only provides the `ClaudeDispatcher` entry. The map construction happens in the run command wiring (future feature).

### 5.4 Key Technical Decisions

#### Decision 1: Command Execution via Function Field for Testability

The `ClaudeDispatcher` wraps `exec.CommandContext` in a function field on the struct, allowing tests to substitute a recorder without running real subprocesses.

```go
type ClaudeDispatcher struct {
    // cmdFactory creates an exec.Cmd. Defaults to exec.CommandContext.
    // Tests replace this to capture command arguments without execution.
    cmdFactory func(ctx context.Context, name string, args ...string) *exec.Cmd
}
```

**Rationale**: Follows the pattern from the epic architecture doc Section 8 (Testing Strategy). Testing the dispatcher means verifying correct flag construction, not running the `claude` binary. The function field is the simplest mechanism without introducing a process runner interface.

**Alternative considered**: Using an interface for the process runner. Rejected because it adds an abstraction layer for one call site, violating the project's "no unnecessary complexity" principle.

#### Decision 2: Output Capture via cmd.Output() / CombinedOutput()

The dispatcher captures stdout and stderr using `cmd.StdoutPipe()` and `cmd.StderrPipe()` with goroutines to avoid pipe deadlocks, following Go standard library best practices.

**Rationale**: `cmd.CombinedOutput()` merges streams (loses separation). `cmd.Output()` discards stderr. Separate pipes with goroutines preserve both streams independently.

#### Decision 3: Default Disallowed Tools as Package Constant

```go
var DefaultDisallowedTools = []string{
    "Bash(shark status advance*)",
    "Bash(shark task next-status*)",
    "Bash(shark status set*)",
    "Bash(shark task set-status*)",
    "Bash(shark feature next-status*)",
    "Bash(shark epic next-status*)",
}
```

**Rationale**: Defined as a package-level variable (not a constant, since Go doesn't support constant slices) for easy extension and testing. Includes feature/epic advancement commands in addition to the task-level ones specified in epic ADR-005, because the run loop operates on all entity types.

### 5.5 Integration with Existing Code

#### Dependency: `exec.CommandContext`

Standard library `os/exec`. No new dependencies.

#### Dependency: `exec.LookPath`

Standard library `os/exec`. Used for tool availability check (REQ-F01-006).

#### Config Integration (`internal/config/orchestrator_action.go`)

The `Provider` and `Model` fields are added to the struct at lines 14-28. The `Validate()` method (line 53) does NOT need to validate `Provider` -- any string is valid, and the run controller is responsible for checking if a dispatcher exists for the provider. The `PopulateTemplate()` method is unchanged -- `Provider` and `Model` are not template variables.

#### Config Integration (`internal/config/action_service.go`)

The `GetStatusActionPopulated()` method (around line 70+) constructs a `PopulatedAction` from the `OrchestratorAction`. Two new field copies are added:

```go
populated := &PopulatedAction{
    Action:    action.Action,
    AgentType: action.AgentType,
    Provider:  action.Provider,   // NEW
    Model:     action.Model,      // NEW
    Skills:    action.Skills,
    Instruction: renderedInstruction,
}
```

#### No Service Layer Changes

This feature does not modify any service (`TaskService`, `FeatureService`, `EpicService`). It does not modify `services_global.go`. The `GetActionService()` accessor (mentioned in the research report as a minor extension) is NOT needed for this feature -- it will be added when the run command is wired (future feature).

### 5.6 File Organization

```
internal/runner/                          # NEW package
    dispatcher.go                         # Interface + types + error types
    claude_dispatcher.go                  # ClaudeDispatcher implementation
    claude_dispatcher_test.go             # Command construction tests
    dispatcher_test.go                    # Type tests, error type tests

internal/config/
    orchestrator_action.go                # MODIFIED: +Provider, +Model fields
    action_service.go                     # MODIFIED: +Provider, +Model in PopulatedAction
```

---

## 6. Testing Strategy

### Unit Tests: `dispatcher_test.go`

| Test | Description |
|------|-------------|
| `TestDispatchInput_Fields` | Verify all fields are settable and accessible |
| `TestDispatchResult_Fields` | Verify all fields including Duration |
| `TestToolNotFoundError_Message` | Error message format includes tool name |
| `TestToolNotFoundError_ErrorsAs` | Can be matched with `errors.As` |
| `TestAgentFailedError_Message` | Error message includes exit code and stderr |
| `TestAgentFailedError_ErrorsAs` | Can be matched with `errors.As` |

### Unit Tests: `claude_dispatcher_test.go`

| Test | Description |
|------|-------------|
| `TestClaudeDispatcher_Name` | Returns `"claude"` |
| `TestClaudeDispatcher_BasicCommand` | Verifies `claude -p "<instruction>" --output-format json` args |
| `TestClaudeDispatcher_DisallowedTools` | All 6 default disallowed tool patterns are included as flags |
| `TestClaudeDispatcher_AdditionalDisallowed` | User-specified disallowed tools are appended |
| `TestClaudeDispatcher_MaxTurns` | `--max-turns N` included when `MaxTurns > 0` |
| `TestClaudeDispatcher_AllowedTools` | `--allowedTools` included when non-empty |
| `TestClaudeDispatcher_ModelOverride` | `--model` flag included when `Model` is non-empty |
| `TestClaudeDispatcher_WorkingDir` | `cmd.Dir` set to `WorkingDir` when non-empty |
| `TestClaudeDispatcher_ContextCancellation` | Context cancel propagates to process kill |
| `TestClaudeDispatcher_ToolNotFound` | Returns `ToolNotFoundError` when `claude` not on PATH |
| `TestClaudeDispatcher_ExitCodeCapture` | Non-zero exit captured in `DispatchResult.ExitCode` |
| `TestClaudeDispatcher_OutputCapture` | Stdout and stderr captured separately |

**Mock Strategy**: Tests use a `cmdFactory` function field override to record command arguments without executing a real subprocess. For tests that need to verify exit codes and output capture, the factory returns a command that runs a simple Go test helper process (the `TestHelperProcess` pattern from the Go standard library).

### Integration Points

No integration tests in this feature. Integration testing of the full dispatch-advance-loop cycle is covered by the run controller feature (E22-F02+).

---

## 7. Acceptance Criteria Summary

1. `internal/runner/dispatcher.go` exists with `AgentDispatcher` interface, `DispatchInput`, `DispatchResult`, `ToolNotFoundError`, and `AgentFailedError` types.
2. `internal/runner/claude_dispatcher.go` exists with `ClaudeDispatcher` implementing `AgentDispatcher`.
3. `ClaudeDispatcher.Dispatch()` constructs the correct `claude -p` command with all required flags including `--disallowedTools`.
4. `OrchestratorAction` struct has `Provider` and `Model` fields (backward-compatible, omitempty).
5. `PopulatedAction` struct has `Provider` and `Model` fields, populated by `GetStatusActionPopulated()`.
6. All tests pass (`make fmt && make lint && make test`).
7. No database schema changes.

---

*Specification complete: 2026-03-21*
