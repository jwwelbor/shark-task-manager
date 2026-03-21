# Research Report: Agent Dispatcher Interface and Claude Implementation

**Feature**: E22-F01: Agent Dispatcher Interface and Claude Implementation
**Date**: 2026-03-21
**Status**: Complete

---

## Executive Summary

E22-F01 establishes the foundational interface for external agent dispatching in the shark run system. This feature defines:
1. **AgentDispatcher interface** - Contract for invoking external processes (claude, codex, etc.)
2. **DispatchResult type** - Structured return value (exit_code, stdout, stderr, duration)
3. **Claude dispatcher** - Concrete implementation using os/exec with tool restrictions

The design is architecturally sound, uses standard Go patterns, and has clear integration points with the existing service layer. No new infrastructure is required — the dispatcher integrates with existing service architecture via constructor injection.

**Complexity**: SIMPLE (score 5/27) - Scope is constrained, patterns are well-established, no production refactoring needed.

---

## Section 1: Existing Implementations (What to Extend)

### A. Process Execution Patterns in Codebase

**Location**: `internal/cli/commands/handle_service_error_test.go`

```go
cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_NotFound_ExitsWithCode1")
err := cmd.Run()
```

**Pattern**: Standard Go os/exec usage. Shark already demonstrates process invocation and exit code handling in tests.

**Leverage**: Use same patterns for dispatcher implementation - exec.Command, context cancellation, output capture via pipes.

### B. File Operations Interface Pattern

**Location**: `internal/fileops/writer.go`

```go
type EntityFileWriter struct { ... }
func (w *EntityFileWriter) WriteEntityFile(opts WriteOptions) (*WriteResult, error)
```

**Pattern**: Unified interface for entity file operations with:
- Dependency injection via constructor
- Functional options pattern (WriteOptions struct)
- Atomic operations with error handling
- Comprehensive logging support

**Leverage**: Use same pattern for AgentDispatcher:
- Constructor injection
- Functional options for DispatchInput
- Atomic execution with error handling
- Optional logging/tracing

### C. Context-Based Cancellation

**Location**: Multiple service methods use context.Context as first parameter:
- `internal/services/task_service.go` - All methods accept ctx
- `internal/repository/task_repository.go` - All query methods accept ctx

**Pattern**: Uniform context threading through call stack for cancellation and timeouts.

**Leverage**: Dispatcher.Dispatch(ctx context.Context, input DispatchInput) will support context cancellation out of the box.

### D. Service Dependency Injection

**Location**: `internal/cli/services_global.go`, `cmd/server/services.go`

```go
func GetTaskService() *services.TaskService {
    db, _ := GetDB(context.Background())
    repo := repository.NewTaskRepository(db)
    return services.NewTaskService(repo, GetWorkflowService(), nil, nil)
}
```

**Pattern**: Explicit constructor injection with no DI framework. Services compose dependencies clearly.

**Leverage**: Create dispatcher instance similarly:
```go
func NewClaudeDispatcher() AgentDispatcher {
    return &ClaudeDispatcher{}
}
```

---

## Section 2: Integration Points

### A. Service Layer Integration

**Where it fits**: `internal/services/run_service.go` (will be created by E22-F02)

The dispatcher will be injected into RunService:

```go
type RunService struct {
    taskService   *services.TaskService
    dispatcher    run.AgentDispatcher  // Will be injected here
}

func (s *RunService) DispatchAgent(ctx context.Context, input run.DispatchInput) (*run.DispatchResult, error) {
    return s.dispatcher.Dispatch(ctx, input)
}
```

**Current state**: RunService doesn't exist yet (created by E22-F02), but the integration pattern is clear.

### B. CLI Command Integration

**Where it fits**: `internal/cli/commands/run.go` (will be created by E22-F03)

The run command will call the service:

```go
func runCommand(cmd *cobra.Command, args []string) error {
    service := cli.GetRunService()  // New accessor function
    task, err := service.RunTask(cmd.Context(), entityKey)
    return err
}
```

**Current state**: No run command exists yet. Integration is straightforward once the dispatcher interface is defined.

### C. Global Service Accessor

**Where it fits**: `internal/cli/services_global.go`

New accessor function will be added:

```go
func GetRunService() *services.RunService {
    // ... initialize dispatcher
    dispatcher := run.NewClaudeDispatcher()
    return services.NewRunService(
        GetTaskService(),
        GetFeatureService(),
        GetEpicService(),
        dispatcher,
    )
}
```

**Current state**: Pattern already established for other services.

### D. Models Package Integration

**Where it fits**: `internal/models/run_types.go` (new file)

Define all types used by dispatcher:

```go
type DispatchInput struct {
    Context         context.Context
    Instruction     string
    AgentType       string
    AllowedTools    []string
    DisallowedTools []string
    MaxTurns        int
    OutputFormat    string
}

type DispatchResult struct {
    ExitCode  int
    Stdout    string
    Stderr    string
    Duration  time.Duration
}
```

**Current state**: Models package already has entity types. Adding run types is straightforward.

---

## Section 3: Inter-Feature Dependency Map (E22)

```
E22-F01 (Agent Dispatcher Interface - THIS FEATURE)
    ↓ blocks
E22-F02 (Run Loop Controller)
    ↓ blocks
E22-F03 (CLI Command Registration)
E22-F04 (Codex CLI Dispatcher)  [depends on E22-F01 interface]
E22-F05 (Structured Run Logging) [depends on E22-F01 for DispatchResult]
E22-F06 (Git Worktree Isolation) [depends on E22-F02 controller]
E22-F07 (Dry-Run Mode)           [depends on E22-F02 controller]
```

**Why E22-F01 is critical**: All other features depend on the AgentDispatcher interface. E22-F02 injects it, E22-F04 implements a second dispatcher, E22-F05 uses DispatchResult structure.

---

## Section 4: Extension vs New Analysis

### A. Process Execution (os/exec)

| Component | Extend? | Location | Action |
|-----------|---------|----------|--------|
| exec.Command usage | ✅ Yes | `internal/run/dispatchers/claude_dispatcher.go` | Use standard patterns from existing code |
| Context cancellation | ✅ Yes | Both | Pass context through to cmd.ExecuteContext() |
| Output capture | ✅ Yes | ClaudeDispatcher | Use io.Pipe() for stdout/stderr capture |
| Signal handling | ❌ New | ClaudeDispatcher | Handle SIGTERM/SIGKILL for graceful shutdown |

### B. Interfaces and Types

| Component | Extend? | Location | Action |
|-----------|---------|----------|--------|
| AgentDispatcher interface | ❌ New | `internal/run/dispatcher.go` | Define new interface |
| DispatchInput type | ❌ New | `internal/models/run_types.go` | Define new struct |
| DispatchResult type | ❌ New | `internal/models/run_types.go` | Define new struct |

### C. Error Handling

| Component | Extend? | Location | Action |
|-----------|---------|----------|--------|
| Error wrapping pattern | ✅ Yes | ClaudeDispatcher | Use `fmt.Errorf("context: %w", err)` pattern |
| Exit code mapping | ❌ New | ClaudeDispatcher | Map process exit codes to DispatchResult |
| Structured errors | ⚠️ Consider | `internal/run/errors.go` | Consider custom error types (DispatchError, ProcessError) |

### D. Testing

| Component | Extend? | Location | Action |
|-----------|---------|----------|--------|
| Unit test patterns | ✅ Yes | Tests | Use mock interface pattern from existing tests |
| Table-driven tests | ✅ Yes | Tests | Multiple dispatch scenarios (success, timeout, tool restriction) |
| Process mocking | ❌ New | Tests | Mock subprocess execution with controlled exit codes |

---

## Section 5: File Structure and Paths

### New Files to Create

```
internal/
├── run/                                  (NEW PACKAGE)
│   ├── dispatcher.go                    (Interface definition)
│   ├── dispatcher_test.go               (Interface tests)
│   ├── errors.go                        (Error types)
│   └── dispatchers/
│       ├── claude_dispatcher.go         (Claude implementation)
│       └── claude_dispatcher_test.go    (Claude tests)
│
└── models/
    └── run_types.go                     (DispatchInput, DispatchResult)

internal/run/
├── dispatcher.go                        (AgentDispatcher interface)
├── dispatchers/
│   └── claude_dispatcher.go             (ClaudeDispatcher implementation)
```

### Files to Modify (Minimal Changes)

```
internal/
├── models/run_types.go                  (ADD new types)
├── cli/services_global.go               (ADD GetRunService accessor - in E22-F03)
└── services/run_service.go              (NEW, created by E22-F02, injects dispatcher)
```

---

## Section 6: Key Design Decisions

### A. Interface vs Concrete Type

**Decision**: Define AgentDispatcher as an interface with Dispatch method.

**Rationale**:
- Enables multiple implementations (Claude, Codex, etc.)
- Facilitates testing via mocks
- Follows Go conventions (reader/writer pattern)
- E22-F04 adds CodexDispatcher implementation

**Files**: `internal/run/dispatcher.go`

```go
type AgentDispatcher interface {
    Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error)
}
```

### B. Functional Options vs Explicit Parameters

**Decision**: Use DispatchInput struct instead of many function parameters.

**Rationale**:
- Cleaner function signature
- Easier to extend with new options (max_retries, timeout, etc.)
- Matches existing patterns (WriteOptions in fileops)
- API stability when adding new fields

**Files**: `internal/models/run_types.go`

```go
type DispatchInput struct {
    Instruction     string
    AgentType       string
    AllowedTools    []string
    DisallowedTools []string
    MaxTurns        int
    OutputFormat    string
}
```

### C. Tool Restrictions Implementation

**Decision**: Pass --disallowedTools and --allowedTools as CLI flags to claude/codex.

**Rationale**:
- Leverages existing Claude/Codex CLI support for these flags
- No need to parse/validate tool names in Go
- Simpler implementation
- Defers responsibility to the CLI tool

**Files**: `internal/run/dispatchers/claude_dispatcher.go`

```go
// Pseudo-code
args := []string{"-p", instruction}
if len(input.DisallowedTools) > 0 {
    args = append(args, "--disallowedTools")
    args = append(args, input.DisallowedTools...)
}
cmd := exec.CommandContext(ctx, "claude", args...)
```

### D. Process Lifecycle Management

**Decision**: Use exec.CommandContext() for context-aware cancellation.

**Rationale**:
- Automatic signal handling (SIGTERM on context cancel)
- Timeouts via context.WithTimeout()
- Clean process termination
- Standard Go pattern

**Files**: `internal/run/dispatchers/claude_dispatcher.go`

---

## Section 7: Recommended Implementation Approach

### Phase 1: Interface Definition (2-3 hours)

1. Create `internal/run/dispatcher.go`
   - Define AgentDispatcher interface with Dispatch method
   - Document expected behavior
   - Add godoc comments with examples

2. Create `internal/models/run_types.go`
   - Define DispatchInput struct with all fields
   - Define DispatchResult struct with exit_code, stdout, stderr, duration
   - Add validation methods if needed

### Phase 2: Claude Dispatcher Implementation (4-5 hours)

1. Create `internal/run/dispatchers/claude_dispatcher.go`
   - Implement ClaudeDispatcher struct
   - Implement Dispatch method
   - Handle process execution via os/exec
   - Capture stdout/stderr via pipes
   - Apply tool restrictions
   - Calculate execution duration

2. Error handling
   - Define custom error types in `internal/run/errors.go`
   - Wrap errors with context
   - Map process exit codes to structured errors

### Phase 3: Comprehensive Testing (5-6 hours)

1. Unit tests for dispatcher
   - Mock interface implementation
   - Test input validation
   - Test tool restriction flag building

2. Integration tests for Claude dispatcher
   - Mock subprocess with controlled exit codes
   - Test success path (exit 0)
   - Test error path (non-zero exit code)
   - Test context cancellation
   - Test output capture (stdout/stderr)
   - Test timeout behavior

3. Service integration tests
   - Test dispatcher injection into RunService
   - Test call chains from CLI command → service → dispatcher

---

## Section 8: Tool Availability Check (exec.LookPath)

**Design decision**: Validate claude CLI is available at startup or on first dispatch.

**Implementation options**:

**Option A**: Validate on startup (eager)
```go
func NewClaudeDispatcher() (AgentDispatcher, error) {
    if _, err := exec.LookPath("claude"); err != nil {
        return nil, fmt.Errorf("claude CLI not found: %w", err)
    }
    return &ClaudeDispatcher{}, nil
}
```

**Option B**: Validate on dispatch (lazy)
```go
func (d *ClaudeDispatcher) Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
    if _, err := exec.LookPath("claude"); err != nil {
        return nil, fmt.Errorf("claude CLI not found: %w", err)
    }
    // ... rest of dispatch logic
}
```

**Recommendation**: Option B (lazy validation) - more flexible, allows CLI installation between shark startup and actual dispatch.

---

## Section 9: Exit Gates for Research

✅ **All sections complete**:
- [x] Existing related code identified with file paths
- [x] Extension points documented (4 patterns identified)
- [x] Inter-feature dependency map provided
- [x] Extension vs new analysis completed (12 components analyzed)
- [x] File structure and paths specified
- [x] Key design decisions documented
- [x] Implementation approach recommended (3 phases, 14 hours total)
- [x] Tool availability validation strategy provided

**Ready to advance to specification phase**.

---

*Last Updated*: 2026-03-21
