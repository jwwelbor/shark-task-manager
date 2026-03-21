package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
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

	factory := d.cmdFactory
	if factory == nil {
		factory = exec.CommandContext
	}

	cmd := factory(ctx, "claude", args...)

	if input.WorkingDir != "" {
		cmd.Dir = input.WorkingDir
	}

	cmdStr := "claude " + strings.Join(args, " ")
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
