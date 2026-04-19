package runner

import (
	"context"
	"os/exec"
)

// CodexDispatcher implements AgentDispatcher by invoking the codex CLI tool
// via os/exec. It validates tool availability at dispatch time (lazy), not at
// construction time.
//
// Command format: codex exec -m {model} --full-auto --skip-git-repo-check "{instruction}"
//
// Testing: The cmdFactory and lookPathFunc fields can be replaced in tests to
// capture command arguments without executing a real subprocess.
type CodexDispatcher struct {
	// cmdFactory creates an *exec.Cmd. Defaults to exec.CommandContext.
	// Tests replace this to record command arguments without execution.
	cmdFactory func(ctx context.Context, name string, args ...string) *exec.Cmd

	// lookPathFunc checks for binary availability. Defaults to exec.LookPath.
	// Tests replace this to simulate missing binaries.
	lookPathFunc func(file string) (string, error)
}

// NewCodexDispatcher creates a CodexDispatcher with default os/exec implementations.
func NewCodexDispatcher() *CodexDispatcher {
	return &CodexDispatcher{
		cmdFactory:   exec.CommandContext,
		lookPathFunc: exec.LookPath,
	}
}

// Name returns the human-readable identifier for this dispatcher.
func (d *CodexDispatcher) Name() string {
	return "codex"
}

// BuildCommand returns the exact shell-equivalent codex CLI command string
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
func (d *CodexDispatcher) BuildCommand(input DispatchInput) (string, error) {
	return joinCommand("codex", buildCodexArgs(input))
}

// Dispatch validates the codex binary is on PATH, builds the command with all
// required flags, executes it, and returns the result.
//
// Context cancellation kills the subprocess via exec.CommandContext semantics.
func (d *CodexDispatcher) Dispatch(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
	lookPath := d.lookPathFunc
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	// Validate codex binary availability at dispatch time (lazy validation)
	if _, err := lookPath("codex"); err != nil {
		return nil, &ToolNotFoundError{Tool: "codex"}
	}

	args := buildCodexArgs(input)

	// joinCommand shell-quotes each arg so the logged command string is a
	// faithful shell-equivalent of the argv passed to exec.Command. Sharing
	// the helper with BuildCommand guarantees the two construction sites
	// cannot drift. Built BEFORE spawning the subprocess so NUL-byte input
	// is surfaced as a plain error rather than an opaque exec failure.
	cmdStr, err := joinCommand("codex", args)
	if err != nil {
		return nil, err
	}

	factory := d.cmdFactory
	if factory == nil {
		factory = exec.CommandContext
	}

	cmd := factory(ctx, "codex", args...)

	if input.WorkingDir != "" {
		cmd.Dir = input.WorkingDir
	}

	return execAndCapture(cmd, cmdStr)
}

// buildCodexArgs constructs the argument slice for the codex CLI invocation.
//
// Command structure: codex exec [-m model] --full-auto --skip-git-repo-check "instruction"
//
// Disallowed tools are not passed to codex (codex does not support the same
// flag conventions as claude). The instruction is always the last argument,
// passed as a single quoted string to avoid shell splitting.
func buildCodexArgs(input DispatchInput) []string {
	args := []string{"exec"}

	// Model override
	if input.Model != "" {
		args = append(args, "-m", input.Model)
	}

	// Codex-specific flags for autonomous operation
	args = append(args, "--full-auto", "--skip-git-repo-check")

	// Instruction is always last, as a single argument
	args = append(args, input.Instruction)

	return args
}
