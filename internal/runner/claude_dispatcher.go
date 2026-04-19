package runner

import (
	"context"
	"fmt"
	"os/exec"
)

// ClaudeDispatcher implements AgentDispatcher by invoking the claude CLI tool
// via os/exec. It validates tool availability at dispatch time (lazy), not at
// construction time.
//
// Testing: The cmdFactory and lookPathFunc fields can be replaced in tests to
// capture command arguments without executing a real subprocess.
type ClaudeDispatcher struct {
	// cmdFactory creates an *exec.Cmd. Defaults to exec.CommandContext.
	// Tests replace this to record command arguments without execution.
	cmdFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

	// lookPathFunc checks for binary availability. Defaults to exec.LookPath.
	// Tests replace this to simulate missing binaries.
	lookPathFunc func(file string) (string, error)
}

// NewClaudeDispatcher creates a ClaudeDispatcher with default os/exec implementations.
func NewClaudeDispatcher() *ClaudeDispatcher {
	return &ClaudeDispatcher{
		cmdFactory:   exec.CommandContext,
		lookPathFunc: exec.LookPath,
	}
}

// Name returns the human-readable identifier for this dispatcher.
func (d *ClaudeDispatcher) Name() string {
	return "claude"
}

// BuildCommand returns the exact shell-equivalent claude CLI command string
// that Dispatch would execute for the given input. This is used to populate
// the command attribute on the run.stage.dispatch slog event before the
// subprocess is spawned. The returned string matches the Command field set
// by a successful Dispatch (they share joinCommand so they cannot drift).
//
// Each argument is POSIX-safely shell-quoted so pasting the logged command
// into a shell reproduces exactly the argv that exec.Command passes to the
// OS, including arguments containing spaces, quotes, or shell metacharacters.
//
// Returns errShellQuoteNUL if any argument contains a NUL byte — such input
// cannot be executed by os/exec and has no POSIX shell representation.
func (d *ClaudeDispatcher) BuildCommand(input DispatchInput) (string, error) {
	return joinCommand("claude", buildClaudeArgs(input))
}

// Dispatch validates the claude binary is on PATH, builds the command with all
// required flags, executes it, and returns the result.
//
// Context cancellation kills the subprocess via exec.CommandContext semantics.
func (d *ClaudeDispatcher) Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
	lookPath := d.lookPathFunc
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	// Validate claude binary availability at dispatch time (REQ-F01-006)
	if _, err := lookPath("claude"); err != nil {
		return nil, &ToolNotFoundError{Tool: "claude"}
	}

	args := buildClaudeArgs(input)

	// joinCommand shell-quotes each arg so the logged command string is a
	// faithful shell-equivalent of the argv passed to exec.Command. Sharing
	// the helper with BuildCommand guarantees the two construction sites
	// cannot drift. Built BEFORE spawning the subprocess so NUL-byte input
	// is surfaced as a plain error rather than an opaque exec failure.
	cmdStr, err := joinCommand("claude", args)
	if err != nil {
		return nil, err
	}

	factory := d.cmdFactory
	if factory == nil {
		factory = exec.CommandContext
	}

	cmd := factory(ctx, "claude", args...)

	if input.WorkingDir != "" {
		cmd.Dir = input.WorkingDir
	}

	return execAndCapture(cmd, cmdStr)
}

// buildClaudeArgs constructs the argument slice for the claude CLI invocation.
func buildClaudeArgs(input DispatchInput) []string {
	args := []string{
		"-p", input.Instruction,
		"--output-format", "json",
	}

	// Model override (REQ-F01-004)
	if input.Model != "" {
		args = append(args, "--model", input.Model)
	}

	// Max turns (REQ-F01-004)
	if input.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", input.MaxTurns))
	}

	// Allowed tools (REQ-F01-004)
	for _, tool := range input.AllowedTools {
		args = append(args, "--allowedTools", tool)
	}

	// Default disallowed tools — always appended first (REQ-F01-005)
	for _, tool := range DefaultDisallowedTools {
		args = append(args, "--disallowedTools", tool)
	}

	// Additional user-specified disallowed tools — additive (REQ-F01-005)
	for _, tool := range input.DisallowedTools {
		args = append(args, "--disallowedTools", tool)
	}

	return args
}
