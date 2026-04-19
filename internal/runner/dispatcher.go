// Package runner provides the AgentDispatcher interface and related types for
// invoking external AI agents (Claude, Codex, etc.) as part of the E22 run loop.
package runner

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// DefaultDisallowedTools is the set of shark status-advancement commands that are
// always blocked from running inside a dispatched agent session, regardless of any
// user-specified DisallowedTools. This prevents agent self-advancement, which must
// be done exclusively by the run controller.
//
// Defined as a package-level variable (not a constant, since Go doesn't support
// constant slices) for easy inspection and extension in tests.
var DefaultDisallowedTools = []string{
	"Bash(shark status advance*)",
	"Bash(shark task next-status*)",
	"Bash(shark status set*)",
	"Bash(shark task set-status*)",
	"Bash(shark feature next-status*)",
	"Bash(shark epic next-status*)",
}

// AgentDispatcher is the interface that all agent dispatch implementations must satisfy.
// It allows the run controller to invoke external AI agents without coupling to any
// specific agent CLI tool.
//
// Implementations:
//   - ClaudeDispatcher — invokes the claude CLI
//   - CodexDispatcher  — future implementation
type AgentDispatcher interface {
	// Dispatch invokes the agent with the given input and returns the result.
	// If the context is cancelled, the subprocess is killed and an error is returned.
	Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error)

	// Name returns a human-readable identifier for this dispatcher (e.g. "claude").
	Name() string

	// BuildCommand returns the exact CLI command string that Dispatch would
	// execute for the given input, without actually running the subprocess.
	// This is used by the run controller to populate the command attribute
	// on the run.stage.dispatch slog event BEFORE the subprocess is spawned.
	// The returned string must match the Command field that a successful
	// Dispatch would set on DispatchResult.
	//
	// Returns an error if the input cannot be represented as a POSIX
	// shell-equivalent command string (e.g. an argument contains a NUL
	// byte). In that case the controller must emit run.stage.error with
	// phase="shell_quote" and skip Dispatch — os/exec would reject the
	// argv with EINVAL anyway, so the dispatch cannot proceed.
	BuildCommand(input DispatchInput) (string, error)
}

// DispatchInput contains all information needed to invoke an agent for a single
// workflow stage. It is constructed by the run controller from the entity data
// and the populated orchestrator action.
type DispatchInput struct {
	// Instruction is the rendered instruction string from the template engine.
	Instruction string

	// WorkingDir is the working directory for the agent process. When empty, the
	// agent inherits the run controller's working directory.
	WorkingDir string

	// EntityKey is the entity key (e.g. "E07-F01-001") for context and logging.
	EntityKey string

	// EntityType is the entity type (e.g. "task", "feature", "epic").
	EntityType string

	// Status is the current workflow status being executed (e.g. "in_development").
	Status string

	// AgentType is the agent type from the orchestrator action (e.g. "developer", "qa").
	AgentType string

	// Model is an optional model override (e.g. "claude-opus-4-5"). When empty,
	// the agent uses its default model.
	Model string

	// MaxTurns is an optional limit on the number of turns the agent may take.
	// When 0, no limit is imposed.
	MaxTurns int

	// AllowedTools is an optional list of tools the agent is allowed to use.
	// When empty, no tool allowlist is applied.
	AllowedTools []string

	// DisallowedTools is an optional list of additional tools to disallow beyond
	// DefaultDisallowedTools. These are additive: DefaultDisallowedTools are
	// always applied regardless of this field.
	DisallowedTools []string
}

// DispatchResult captures the outcome of an agent dispatch. It is returned by
// AgentDispatcher.Dispatch() on success or partial success (non-zero exit code).
type DispatchResult struct {
	// ExitCode is the process exit code. 0 indicates success.
	ExitCode int

	// Stdout is the complete captured standard output from the agent process.
	Stdout string

	// Stderr is the complete captured standard error from the agent process.
	Stderr string

	// Duration is the wall-clock time elapsed from subprocess start to exit.
	Duration time.Duration

	// Command is the full command string that was executed (for logging and debugging).
	// It includes the binary name and all flags, but should not include auth tokens.
	Command string
}

// ToolNotFoundError is returned by a dispatcher when the required CLI tool is not
// found on the system PATH. Callers can match it with errors.As.
type ToolNotFoundError struct {
	// Tool is the name of the missing CLI tool (e.g. "claude").
	Tool string
}

// Error implements the error interface.
// Message format: "<tool> CLI not found on PATH. Install <tool> to continue."
func (e *ToolNotFoundError) Error() string {
	return fmt.Sprintf("%s CLI not found on PATH. Install %s to continue.", e.Tool, e.Tool)
}

// AgentFailedError is returned when a dispatched agent process exits with a
// non-zero exit code. It carries the exit code, output streams, and the command
// that was run for diagnostic purposes. Callers can match it with errors.As.
type AgentFailedError struct {
	// ExitCode is the non-zero process exit code.
	ExitCode int

	// Stdout is the captured standard output from the agent process.
	Stdout string

	// Stderr is the captured standard error from the agent process.
	Stderr string

	// Command is the command string that was executed.
	Command string
}

// Error implements the error interface.
// The message includes the exit code and a stderr summary to aid diagnostics.
func (e *AgentFailedError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("agent exited with code %d: %s", e.ExitCode, e.Stderr)
	}
	return fmt.Sprintf("agent exited with code %d", e.ExitCode)
}

// execAndCapture runs a prepared *exec.Cmd, captures stdout/stderr via pipes
// concurrently (to prevent deadlocks), and returns a DispatchResult. On non-zero
// exit, it returns both the result and an *AgentFailedError. On non-exit errors
// (e.g., context cancellation), it returns only an error.
//
// This is the shared subprocess execution logic used by all dispatchers.
func execAndCapture(cmd *exec.Cmd, cmdStr string) (*DispatchResult, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Read stdout and stderr concurrently to prevent pipe deadlocks.
	stdoutCh := make(chan []byte, 1)
	stderrCh := make(chan []byte, 1)

	go func() {
		data, _ := io.ReadAll(stdoutPipe)
		stdoutCh <- data
	}()
	go func() {
		data, _ := io.ReadAll(stderrPipe)
		stderrCh <- data
	}()

	stdoutData := <-stdoutCh
	stderrData := <-stderrCh

	waitErr := cmd.Wait()
	duration := time.Since(start)

	stdout := string(stdoutData)
	stderr := string(stderrData)

	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("process error: %w", waitErr)
		}
	}

	result := &DispatchResult{
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Duration: duration,
		Command:  cmdStr,
	}

	if exitCode != 0 {
		return result, &AgentFailedError{
			ExitCode: exitCode,
			Stdout:   stdout,
			Stderr:   stderr,
			Command:  cmdStr,
		}
	}

	return result, nil
}
