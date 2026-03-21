package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Test infrastructure: helpers specific to CodexDispatcher
// =============================================================================

// newTestCodexDispatcher creates a CodexDispatcher with a recording cmdFactory and
// a successful lookPathFunc so tests don't need the real codex binary.
func newTestCodexDispatcher(captured *capturedCmd) *CodexDispatcher {
	return &CodexDispatcher{
		cmdFactory:   recordingFactory(captured),
		lookPathFunc: successLookPath,
	}
}

// =============================================================================
// Name() tests
// =============================================================================

func TestCodexDispatcher_Name(t *testing.T) {
	d := NewCodexDispatcher()
	if d.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", d.Name(), "codex")
	}
}

func TestCodexDispatcher_Name_NotEmpty(t *testing.T) {
	d := NewCodexDispatcher()
	if d.Name() == "" {
		t.Error("Name() must not return empty string")
	}
	if d.Name() == "claude" {
		t.Error("Name() must return 'codex', not 'claude'")
	}
}

func TestCodexDispatcher_Name_NotClaude(t *testing.T) {
	d := NewCodexDispatcher()
	if d.Name() == "claude" || d.Name() == "anthropic" {
		t.Errorf("Name() = %q; CodexDispatcher must not identify as claude/anthropic", d.Name())
	}
}

// =============================================================================
// Basic command structure tests
// =============================================================================

func TestCodexDispatcher_BasicCommand_BinaryName(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "implement the feature",
		EntityKey:   "E07-F01-001",
	})

	if captured.name != "codex" {
		t.Errorf("binary name = %q, want %q", captured.name, "codex")
	}
}

func TestCodexDispatcher_BasicCommand_ExecSubcommand(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "implement the feature",
	})

	if len(captured.args) == 0 || captured.args[0] != "exec" {
		t.Errorf("expected args[0] = 'exec', got %v", captured.args)
	}
}

func TestCodexDispatcher_BasicCommand_FullAutoFlag(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
	})

	if !containsArg(captured.args, "--full-auto") {
		t.Errorf("expected '--full-auto' in args, got %v", captured.args)
	}
}

func TestCodexDispatcher_BasicCommand_SkipGitRepoCheckFlag(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
	})

	if !containsArg(captured.args, "--skip-git-repo-check") {
		t.Errorf("expected '--skip-git-repo-check' in args, got %v", captured.args)
	}
}

func TestCodexDispatcher_BasicCommand_InstructionIsLastArg(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	instruction := "implement the feature"
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: instruction,
	})

	if len(captured.args) == 0 {
		t.Fatal("no args captured")
	}
	last := captured.args[len(captured.args)-1]
	if last != instruction {
		t.Errorf("last arg = %q, want instruction %q", last, instruction)
	}
}

func TestCodexDispatcher_BasicCommand_InstructionAsSingleArg(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	instruction := "implement the feature with spaces"
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: instruction,
	})

	// Instruction must be passed as a single argument (not split by whitespace)
	for _, arg := range captured.args {
		if arg == "implement" {
			t.Errorf("instruction was split on whitespace; found 'implement' as separate arg in %v", captured.args)
			break
		}
	}

	last := captured.args[len(captured.args)-1]
	if last != instruction {
		t.Errorf("instruction not passed as single arg: last=%q, want %q", last, instruction)
	}
}

func TestCodexDispatcher_BasicCommand_InstructionWithNewlines(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	instruction := "line one\nline two\nline three"
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: instruction,
	})

	last := captured.args[len(captured.args)-1]
	if last != instruction {
		t.Errorf("multi-line instruction split: got %q, want %q", last, instruction)
	}
}

func TestCodexDispatcher_BasicCommand_NoShellExpansion(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	instruction := `"quoted" $HOME ` + "`backtick`"
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: instruction,
	})

	last := captured.args[len(captured.args)-1]
	if last != instruction {
		t.Errorf("shell metacharacters were expanded: got %q, want %q", last, instruction)
	}
}

// =============================================================================
// Model flag tests
// =============================================================================

func TestCodexDispatcher_ModelOverride(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		Model:       "o4-mini",
	})

	if !containsConsecutive(captured.args, "-m", "o4-mini") {
		t.Errorf("expected '-m o4-mini' in args, got %v", captured.args)
	}
}

func TestCodexDispatcher_ModelOverride_Empty_NotAdded(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		Model:       "",
	})

	if containsArg(captured.args, "-m") {
		t.Errorf("-m must NOT appear when Model is empty, got %v", captured.args)
	}
}

func TestCodexDispatcher_ModelOverride_DifferentModels(t *testing.T) {
	tests := []string{"o3", "o4-mini", "gpt-4o"}

	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			var captured capturedCmd
			d := newTestCodexDispatcher(&captured)

			_, _ = d.Dispatch(context.Background(), DispatchInput{
				Instruction: "test",
				Model:       model,
			})

			if !containsConsecutive(captured.args, "-m", model) {
				t.Errorf("expected '-m %s' in args, got %v", model, captured.args)
			}
		})
	}
}

func TestCodexDispatcher_ModelFlag_AppearsBeforeInstruction(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test instruction",
		Model:       "o4-mini",
	})

	// Find positions of -m and instruction
	mIdx := -1
	instrIdx := -1
	for i, arg := range captured.args {
		if arg == "-m" {
			mIdx = i
		}
		if arg == "test instruction" {
			instrIdx = i
		}
	}

	if mIdx == -1 {
		t.Fatal("'-m' flag not found in args")
	}
	if instrIdx == -1 {
		t.Fatal("instruction not found in args")
	}
	if mIdx > instrIdx {
		t.Errorf("-m flag (pos %d) must appear before instruction (pos %d)", mIdx, instrIdx)
	}
}

// =============================================================================
// WorkingDir tests
// =============================================================================

func TestCodexDispatcher_WorkingDir_Set(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		WorkingDir:  "/tmp/worktree-E07-F01-001",
	})

	if captured.cmd == nil {
		t.Fatal("cmdFactory was not called")
	}
	if captured.cmd.Dir != "/tmp/worktree-E07-F01-001" {
		t.Errorf("cmd.Dir = %q, want %q", captured.cmd.Dir, "/tmp/worktree-E07-F01-001")
	}
}

func TestCodexDispatcher_WorkingDir_Empty_NotSet(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		WorkingDir:  "",
	})

	if captured.cmd == nil {
		t.Fatal("cmdFactory was not called")
	}
	if captured.cmd.Dir != "" {
		t.Errorf("cmd.Dir should not be set when WorkingDir is empty, got %q", captured.cmd.Dir)
	}
}

func TestCodexDispatcher_WorkingDir_WithSpaces(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		WorkingDir:  "/home/user/my project",
	})

	if captured.cmd == nil {
		t.Fatal("cmdFactory was not called")
	}
	if captured.cmd.Dir != "/home/user/my project" {
		t.Errorf("cmd.Dir = %q, want %q", captured.cmd.Dir, "/home/user/my project")
	}
}

// =============================================================================
// ToolNotFoundError tests
// =============================================================================

func TestCodexDispatcher_ToolNotFound(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   nil, // should never be called
		lookPathFunc: failedLookPath,
	}

	result, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var toolErr *ToolNotFoundError
	if !errors.As(err, &toolErr) {
		t.Fatalf("expected *ToolNotFoundError, got %T: %v", err, err)
	}
	if toolErr.Tool != "codex" {
		t.Errorf("ToolNotFoundError.Tool = %q, want %q", toolErr.Tool, "codex")
	}
	if !strings.Contains(err.Error(), "codex") {
		t.Errorf("error message must contain 'codex': %q", err.Error())
	}
}

func TestCodexDispatcher_ToolNotFound_ConstructionSucceeds(t *testing.T) {
	// Construction must succeed even when codex is not on PATH.
	// Validation happens at dispatch time (lazy), not at construction time.
	d := NewCodexDispatcher()
	if d == nil {
		t.Error("NewCodexDispatcher() must not return nil")
	}
}

func TestCodexDispatcher_ToolNotFound_ErrorMessage(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   nil,
		lookPathFunc: failedLookPath,
	}

	_, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "codex") {
		t.Errorf("error message %q does not contain 'codex'", msg)
	}
	if !strings.Contains(msg, "PATH") {
		t.Errorf("error message %q does not mention PATH", msg)
	}
}

// =============================================================================
// Context cancellation tests
// =============================================================================

func TestCodexDispatcher_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "sleep"),
		lookPathFunc: successLookPath,
	}

	// Cancel context immediately after launching dispatch
	done := make(chan struct{})
	var dispatchErr error
	go func() {
		defer close(done)
		_, dispatchErr = d.Dispatch(ctx, DispatchInput{Instruction: "test"})
	}()

	// Cancel shortly after starting
	cancel()

	select {
	case <-done:
		// Good — dispatch returned
	case <-time.After(5 * time.Second):
		t.Fatal("Dispatch did not return after context cancellation (timeout)")
	}

	if dispatchErr == nil {
		t.Error("Dispatch must return non-nil error when context is cancelled")
	}
}

func TestCodexDispatcher_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before Dispatch

	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "sleep"),
		lookPathFunc: successLookPath,
	}

	_, err := d.Dispatch(ctx, DispatchInput{Instruction: "test"})
	if err == nil {
		t.Error("Dispatch must return error when context is already cancelled")
	}
}

// =============================================================================
// Output capture tests
// =============================================================================

func TestCodexDispatcher_OutputCapture_Stdout(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "stdout", "hello stdout"),
		lookPathFunc: successLookPath,
	}

	result, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !strings.Contains(result.Stdout, "hello stdout") {
		t.Errorf("Stdout = %q, want it to contain %q", result.Stdout, "hello stdout")
	}
	if strings.Contains(result.Stderr, "hello stdout") {
		t.Errorf("stdout content appeared in Stderr: %q", result.Stderr)
	}
}

func TestCodexDispatcher_OutputCapture_Stderr(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "stderr", "hello stderr"),
		lookPathFunc: successLookPath,
	}

	result, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !strings.Contains(result.Stderr, "hello stderr") {
		t.Errorf("Stderr = %q, want it to contain %q", result.Stderr, "hello stderr")
	}
	if strings.Contains(result.Stdout, "hello stderr") {
		t.Errorf("stderr content appeared in Stdout: %q", result.Stdout)
	}
}

func TestCodexDispatcher_OutputCapture_BothStreams(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "both", "hello stdout", "hello stderr"),
		lookPathFunc: successLookPath,
	}

	result, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !strings.Contains(result.Stdout, "hello stdout") {
		t.Errorf("Stdout = %q, want it to contain 'hello stdout'", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "hello stderr") {
		t.Errorf("Stderr = %q, want it to contain 'hello stderr'", result.Stderr)
	}
	// Streams must NOT be combined
	if strings.Contains(result.Stdout, "hello stderr") {
		t.Error("stderr content must not appear in Stdout")
	}
	if strings.Contains(result.Stderr, "hello stdout") {
		t.Error("stdout content must not appear in Stderr")
	}
}

// =============================================================================
// Exit code capture tests
// =============================================================================

func TestCodexDispatcher_ExitCodeCapture_NonZero(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "exit", "2"),
		lookPathFunc: successLookPath,
	}

	result, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})

	// Either result or error must expose exit code 2
	if result != nil && result.ExitCode == 2 {
		// OK — exit code accessible via result
		return
	}

	if err != nil {
		var agentErr *AgentFailedError
		if errors.As(err, &agentErr) && agentErr.ExitCode == 2 {
			// OK — exit code accessible via error
			return
		}
	}

	t.Errorf("exit code 2 not accessible: result=%v, err=%v", result, err)
}

func TestCodexDispatcher_ExitCodeCapture_Zero(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "exit", "0"),
		lookPathFunc: successLookPath,
	}

	result, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})

	if err != nil {
		t.Errorf("exit code 0 must not produce error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for exit code 0")
	}
	if result.ExitCode != 0 {
		t.Errorf("result.ExitCode = %d, want 0", result.ExitCode)
	}
}

// =============================================================================
// Duration tests
// =============================================================================

func TestCodexDispatcher_Duration(t *testing.T) {
	d := &CodexDispatcher{
		cmdFactory:   helperCommand(t, "exit", "0"),
		lookPathFunc: successLookPath,
	}

	result, err := d.Dispatch(context.Background(), DispatchInput{Instruction: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Duration must be non-negative
	if result.Duration < 0 {
		t.Errorf("Duration = %v, must be >= 0", result.Duration)
	}

	// Duration must be reasonable (< 10 seconds for a no-op process)
	if result.Duration > 10*time.Second {
		t.Errorf("Duration = %v, unexpectedly large for no-op process", result.Duration)
	}
}

// =============================================================================
// Command string tests
// =============================================================================

func TestCodexDispatcher_CommandString(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	result, _ := d.Dispatch(context.Background(), DispatchInput{Instruction: "test instruction"})

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Command == "" {
		t.Error("DispatchResult.Command must not be empty")
	}
	if !strings.Contains(result.Command, "codex") {
		t.Errorf("Command must contain 'codex': %q", result.Command)
	}
	if !strings.Contains(result.Command, "exec") {
		t.Errorf("Command must contain 'exec': %q", result.Command)
	}
	if !strings.Contains(result.Command, "--full-auto") {
		t.Errorf("Command must contain '--full-auto': %q", result.Command)
	}
}

func TestCodexDispatcher_CommandString_ContainsInstruction(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	result, _ := d.Dispatch(context.Background(), DispatchInput{Instruction: "my test instruction"})

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.Contains(result.Command, "my test instruction") {
		t.Errorf("Command must contain the instruction: %q", result.Command)
	}
}

// =============================================================================
// No disallowed tools tests (codex does not support claude's --disallowedTools flag)
// =============================================================================

func TestCodexDispatcher_NoDisallowedToolsFlag(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:     "test",
		DisallowedTools: []string{"Bash(rm -rf*)"},
	})

	// Codex does not support --disallowedTools; it must not appear in args
	if containsArg(captured.args, "--disallowedTools") {
		t.Errorf("--disallowedTools must NOT appear in codex args, got %v", captured.args)
	}
}

func TestCodexDispatcher_NoAllowedToolsFlag(t *testing.T) {
	var captured capturedCmd
	d := newTestCodexDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:  "test",
		AllowedTools: []string{"Read", "Write"},
	})

	// Codex does not support --allowedTools in the same format; it must not appear
	if containsArg(captured.args, "--allowedTools") {
		t.Errorf("--allowedTools must NOT appear in codex args, got %v", captured.args)
	}
}

// =============================================================================
// Interface compliance test
// =============================================================================

func TestCodexDispatcher_ImplementsAgentDispatcher(t *testing.T) {
	// Compile-time check: CodexDispatcher must implement AgentDispatcher.
	// If this doesn't compile, the interface is not satisfied.
	var _ AgentDispatcher = (*CodexDispatcher)(nil)
}

// =============================================================================
// Table-driven scenarios
// =============================================================================

func TestCodexDispatcher_ModelScenarios(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		expectMFlag  bool
		expectedArgs []string
	}{
		{
			name:        "no model",
			model:       "",
			expectMFlag: false,
		},
		{
			name:         "o3 model",
			model:        "o3",
			expectMFlag:  true,
			expectedArgs: []string{"-m", "o3"},
		},
		{
			name:         "o4-mini model",
			model:        "o4-mini",
			expectMFlag:  true,
			expectedArgs: []string{"-m", "o4-mini"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured capturedCmd
			d := newTestCodexDispatcher(&captured)

			_, _ = d.Dispatch(context.Background(), DispatchInput{
				Instruction: "test",
				Model:       tt.model,
			})

			if tt.expectMFlag {
				if !containsConsecutive(captured.args, "-m", tt.model) {
					t.Errorf("expected '-m %s' in args, got %v", tt.model, captured.args)
				}
			} else {
				if containsArg(captured.args, "-m") {
					t.Errorf("-m must NOT appear when model is empty, got %v", captured.args)
				}
			}
		})
	}
}

func TestCodexDispatcher_RequiredFlagsAlwaysPresent(t *testing.T) {
	// These flags must always be present regardless of input
	requiredFlags := []string{"exec", "--full-auto", "--skip-git-repo-check"}

	instructions := []string{
		"simple instruction",
		"",
		"instruction with\nnewlines",
		`instruction with "quotes"`,
	}

	for _, instruction := range instructions {
		t.Run(fmt.Sprintf("instruction=%q", instruction), func(t *testing.T) {
			var captured capturedCmd
			d := newTestCodexDispatcher(&captured)

			_, _ = d.Dispatch(context.Background(), DispatchInput{
				Instruction: instruction,
			})

			for _, flag := range requiredFlags {
				if !containsArg(captured.args, flag) {
					t.Errorf("required arg %q missing from args %v", flag, captured.args)
				}
			}
		})
	}
}
