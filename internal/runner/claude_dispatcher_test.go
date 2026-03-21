package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Test infrastructure: cmdFactory and lookPathFunc overrides
// =============================================================================

// capturedCmd records what was passed to cmdFactory without executing a real subprocess.
type capturedCmd struct {
	name string
	args []string
	cmd  *exec.Cmd // the Cmd created (used to inspect Dir, etc.)
}

// recordingFactory returns a cmdFactory that records the command and returns a
// no-op Cmd (runs /bin/true or equivalent) so tests can still call Start/Wait.
// captured is written on each factory call.
func recordingFactory(captured *capturedCmd) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		captured.name = name
		captured.args = append([]string{}, args...) // defensive copy
		// Return a real Cmd pointing at a no-op process so Start/Wait succeed.
		// We use "true" on Unix; on Windows this would need adjustment.
		cmd := exec.CommandContext(ctx, "true")
		captured.cmd = cmd
		return cmd
	}
}

// failedLookPath returns a lookPathFunc that always reports the binary as missing.
func failedLookPath(string) (string, error) {
	return "", fmt.Errorf("not found")
}

// successLookPath returns a lookPathFunc that always reports the binary as found.
func successLookPath(file string) (string, error) {
	return "/usr/bin/" + file, nil
}

// newTestDispatcher creates a ClaudeDispatcher with a recording cmdFactory and
// a successful lookPathFunc so tests don't need the real claude binary.
func newTestDispatcher(captured *capturedCmd) *ClaudeDispatcher {
	return &ClaudeDispatcher{
		cmdFactory:   recordingFactory(captured),
		lookPathFunc: successLookPath,
	}
}

// =============================================================================
// TC-007: ClaudeDispatcher.Name() returns "claude"
// =============================================================================

func TestClaudeDispatcher_Name(t *testing.T) {
	d := NewClaudeDispatcher()
	if d.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", d.Name(), "claude")
	}
}

func TestClaudeDispatcher_Name_NotEmpty(t *testing.T) {
	d := NewClaudeDispatcher()
	if d.Name() == "" {
		t.Error("Name() must not return empty string")
	}
	if d.Name() == "anthropic" {
		t.Error("Name() must return 'claude', not 'anthropic'")
	}
}

// =============================================================================
// TC-008: Basic claude command structure
// =============================================================================

func TestClaudeDispatcher_BasicCommand(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "implement the feature",
		EntityKey:   "E07-F01-001",
	})

	// Verify binary name
	if captured.name != "claude" {
		t.Errorf("binary name = %q, want %q", captured.name, "claude")
	}

	// Verify -p is args[0]
	if len(captured.args) < 2 || captured.args[0] != "-p" {
		t.Errorf("expected args[0] = '-p', got %v", captured.args)
	}

	// Verify instruction is a single argument (not split by whitespace)
	if captured.args[1] != "implement the feature" {
		t.Errorf("args[1] = %q, want %q", captured.args[1], "implement the feature")
	}

	// Verify --output-format json is present
	if !containsConsecutive(captured.args, "--output-format", "json") {
		t.Errorf("args must contain '--output-format json', got %v", captured.args)
	}
}

func TestClaudeDispatcher_BasicCommand_InstructionWithNewlines(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	instruction := "line one\nline two\nline three"
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: instruction,
		EntityKey:   "E07-F01-001",
	})

	// Instruction with newlines must still be a single arg
	if len(captured.args) >= 2 && captured.args[1] != instruction {
		t.Errorf("multi-line instruction split: got %q, want %q", captured.args[1], instruction)
	}
}

func TestClaudeDispatcher_BasicCommand_NoShellExpansion(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	instruction := `"quoted" $HOME ` + "`backtick`"
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: instruction,
		EntityKey:   "E07-F01-001",
	})

	// Must not shell-expand; instruction passed literally
	if len(captured.args) >= 2 && captured.args[1] != instruction {
		t.Errorf("shell metacharacters were expanded: got %q, want %q", captured.args[1], instruction)
	}
}

// =============================================================================
// TC-009: Default disallowed tools — all 6 patterns present
// =============================================================================

func TestClaudeDispatcher_DisallowedTools_AllDefaults(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
	})

	expectedTools := []string{
		"Bash(shark status advance*)",
		"Bash(shark task next-status*)",
		"Bash(shark status set*)",
		"Bash(shark task set-status*)",
		"Bash(shark feature next-status*)",
		"Bash(shark epic next-status*)",
	}

	for _, expected := range expectedTools {
		if !containsFlag(captured.args, "--disallowedTools", expected) {
			t.Errorf("missing --disallowedTools %q in args %v", expected, captured.args)
		}
	}
}

func TestClaudeDispatcher_DisallowedTools_NilInput(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	// nil DisallowedTools — only defaults should appear
	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:     "test",
		DisallowedTools: nil,
	})

	count := countFlag(captured.args, "--disallowedTools")
	if count != 6 {
		t.Errorf("expected 6 --disallowedTools entries, got %d (args: %v)", count, captured.args)
	}
}

// =============================================================================
// TC-010: User-specified DisallowedTools are additive (not replacing defaults)
// =============================================================================

func TestClaudeDispatcher_DisallowedTools_Additive(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:     "test",
		DisallowedTools: []string{"Bash(rm -rf*)"},
	})

	// All 6 defaults must be present
	for _, expected := range DefaultDisallowedTools {
		if !containsFlag(captured.args, "--disallowedTools", expected) {
			t.Errorf("default tool missing: --disallowedTools %q", expected)
		}
	}

	// User tool must also be present
	if !containsFlag(captured.args, "--disallowedTools", "Bash(rm -rf*)") {
		t.Errorf("user-specified tool missing from args: %v", captured.args)
	}

	// Total count must be 7 (6 defaults + 1 user)
	count := countFlag(captured.args, "--disallowedTools")
	if count != 7 {
		t.Errorf("expected 7 --disallowedTools entries (6 defaults + 1 user), got %d", count)
	}
}

func TestClaudeDispatcher_DisallowedTools_EmptySlice(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:     "test",
		DisallowedTools: []string{},
	})

	// Empty slice — only 6 defaults
	count := countFlag(captured.args, "--disallowedTools")
	if count != 6 {
		t.Errorf("expected exactly 6 --disallowedTools entries, got %d", count)
	}
}

// =============================================================================
// TC-011: MaxTurns > 0 adds --max-turns flag
// =============================================================================

func TestClaudeDispatcher_MaxTurns_Added(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		MaxTurns:    15,
	})

	if !containsConsecutive(captured.args, "--max-turns", "15") {
		t.Errorf("expected '--max-turns 15' in args, got %v", captured.args)
	}
}

func TestClaudeDispatcher_MaxTurns_Zero_NotAdded(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		MaxTurns:    0,
	})

	if containsArg(captured.args, "--max-turns") {
		t.Errorf("--max-turns must NOT appear when MaxTurns == 0, got %v", captured.args)
	}
}

func TestClaudeDispatcher_MaxTurns_Values(t *testing.T) {
	tests := []struct {
		maxTurns int
		want     string
	}{
		{1, "1"},
		{100, "100"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("MaxTurns=%d", tt.maxTurns), func(t *testing.T) {
			var captured capturedCmd
			d := newTestDispatcher(&captured)

			_, _ = d.Dispatch(context.Background(), DispatchInput{
				Instruction: "test",
				MaxTurns:    tt.maxTurns,
			})

			if !containsConsecutive(captured.args, "--max-turns", tt.want) {
				t.Errorf("expected '--max-turns %s' in args, got %v", tt.want, captured.args)
			}
		})
	}
}

// =============================================================================
// TC-012: AllowedTools non-empty adds --allowedTools entries
// =============================================================================

func TestClaudeDispatcher_AllowedTools_NonEmpty(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:  "test",
		AllowedTools: []string{"Read", "Write", "Bash"},
	})

	for _, tool := range []string{"Read", "Write", "Bash"} {
		if !containsFlag(captured.args, "--allowedTools", tool) {
			t.Errorf("missing --allowedTools %q in args %v", tool, captured.args)
		}
	}
}

func TestClaudeDispatcher_AllowedTools_Empty_NotAdded(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:  "test",
		AllowedTools: []string{},
	})

	if containsArg(captured.args, "--allowedTools") {
		t.Errorf("--allowedTools must NOT appear when AllowedTools is empty, got %v", captured.args)
	}
}

func TestClaudeDispatcher_AllowedTools_Nil_NotAdded(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:  "test",
		AllowedTools: nil,
	})

	if containsArg(captured.args, "--allowedTools") {
		t.Errorf("--allowedTools must NOT appear when AllowedTools is nil, got %v", captured.args)
	}
}

func TestClaudeDispatcher_AllowedTools_Single(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction:  "test",
		AllowedTools: []string{"Read"},
	})

	count := countFlag(captured.args, "--allowedTools")
	if count != 1 {
		t.Errorf("expected 1 --allowedTools entry, got %d", count)
	}
	if !containsFlag(captured.args, "--allowedTools", "Read") {
		t.Errorf("--allowedTools Read missing from args %v", captured.args)
	}
}

// =============================================================================
// TC-013: Model override adds --model flag
// =============================================================================

func TestClaudeDispatcher_ModelOverride(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		Model:       "claude-opus-4-5",
	})

	if !containsConsecutive(captured.args, "--model", "claude-opus-4-5") {
		t.Errorf("expected '--model claude-opus-4-5' in args, got %v", captured.args)
	}
}

func TestClaudeDispatcher_ModelOverride_Empty_NotAdded(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	_, _ = d.Dispatch(context.Background(), DispatchInput{
		Instruction: "test",
		Model:       "",
	})

	if containsArg(captured.args, "--model") {
		t.Errorf("--model must NOT appear when Model is empty, got %v", captured.args)
	}
}

func TestClaudeDispatcher_ModelOverride_DifferentModels(t *testing.T) {
	tests := []string{"claude-haiku-3", "claude-sonnet-4-5", "o3"}

	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			var captured capturedCmd
			d := newTestDispatcher(&captured)

			_, _ = d.Dispatch(context.Background(), DispatchInput{
				Instruction: "test",
				Model:       model,
			})

			if !containsConsecutive(captured.args, "--model", model) {
				t.Errorf("expected '--model %s' in args, got %v", model, captured.args)
			}
		})
	}
}

// =============================================================================
// TC-014: WorkingDir sets cmd.Dir
// =============================================================================

func TestClaudeDispatcher_WorkingDir_Set(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

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

func TestClaudeDispatcher_WorkingDir_Empty_NotSet(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

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

func TestClaudeDispatcher_WorkingDir_WithSpaces(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

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
// TC-015: Context cancellation kills subprocess
// =============================================================================

// TestHelperProcess is invoked by tests via exec.Command(os.Args[0], "-test.run=TestHelperProcess").
// It checks GO_TEST_HELPER_PROCESS=1 before acting.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
		return
	}

	// Find the helper action from args after "--"
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	if len(args) == 0 {
		os.Exit(0)
	}

	switch args[0] {
	case "sleep":
		// Sleep for a long time (to be killed by context cancellation)
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "exit":
		code := 0
		if len(args) > 1 {
			_, _ = fmt.Sscanf(args[1], "%d", &code)
		}
		os.Exit(code)
	case "stdout":
		if len(args) > 1 {
			fmt.Print(args[1])
		}
		os.Exit(0)
	case "stderr":
		if len(args) > 1 {
			fmt.Fprint(os.Stderr, args[1])
		}
		os.Exit(0)
	case "both":
		if len(args) > 1 {
			fmt.Print(args[1])
		}
		if len(args) > 2 {
			fmt.Fprint(os.Stderr, args[2])
		}
		os.Exit(0)
	default:
		os.Exit(0)
	}
}

// helperCommand returns an exec.Cmd pointing at the test binary's TestHelperProcess.
func helperCommand(t *testing.T, action string, args ...string) func(ctx context.Context, name string, a ...string) *exec.Cmd {
	t.Helper()
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--"}
		cs = append(cs, action)
		cs = append(cs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		cmd.Env = []string{"GO_TEST_HELPER_PROCESS=1"}
		return cmd
	}
}

func TestClaudeDispatcher_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	d := &ClaudeDispatcher{
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

func TestClaudeDispatcher_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before Dispatch

	d := &ClaudeDispatcher{
		cmdFactory:   helperCommand(t, "sleep"),
		lookPathFunc: successLookPath,
	}

	_, err := d.Dispatch(ctx, DispatchInput{Instruction: "test"})
	if err == nil {
		t.Error("Dispatch must return error when context is already cancelled")
	}
}

// =============================================================================
// TC-016: ToolNotFoundError returned when claude not on PATH
// =============================================================================

func TestClaudeDispatcher_ToolNotFound(t *testing.T) {
	d := &ClaudeDispatcher{
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
	if toolErr.Tool != "claude" {
		t.Errorf("ToolNotFoundError.Tool = %q, want %q", toolErr.Tool, "claude")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("error message must contain 'claude': %q", err.Error())
	}
}

func TestClaudeDispatcher_ToolNotFound_ConstructionSucceeds(t *testing.T) {
	// Construction must succeed even when claude is not on PATH.
	// Validation happens at dispatch time (lazy), not at construction time.
	d := NewClaudeDispatcher()
	if d == nil {
		t.Error("NewClaudeDispatcher() must not return nil")
	}
}

// =============================================================================
// TC-017: DispatchResult captures stdout and stderr separately
// =============================================================================

func TestClaudeDispatcher_OutputCapture_Stdout(t *testing.T) {
	d := &ClaudeDispatcher{
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

func TestClaudeDispatcher_OutputCapture_Stderr(t *testing.T) {
	d := &ClaudeDispatcher{
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

func TestClaudeDispatcher_OutputCapture_BothStreams(t *testing.T) {
	d := &ClaudeDispatcher{
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
// TC-018: DispatchResult captures non-zero exit code
// =============================================================================

func TestClaudeDispatcher_ExitCodeCapture_NonZero(t *testing.T) {
	d := &ClaudeDispatcher{
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

func TestClaudeDispatcher_ExitCodeCapture_Zero(t *testing.T) {
	d := &ClaudeDispatcher{
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
// TC-019: DispatchResult.Duration captures wall-clock execution time
// =============================================================================

func TestClaudeDispatcher_Duration(t *testing.T) {
	d := &ClaudeDispatcher{
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
// TC-020: DispatchResult.Command contains the executed command string
// =============================================================================

func TestClaudeDispatcher_CommandString(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)

	result, _ := d.Dispatch(context.Background(), DispatchInput{Instruction: "test instruction"})

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Command == "" {
		t.Error("DispatchResult.Command must not be empty")
	}
	if !strings.Contains(result.Command, "claude") {
		t.Errorf("Command must contain 'claude': %q", result.Command)
	}
	if !strings.Contains(result.Command, "-p") {
		t.Errorf("Command must contain '-p': %q", result.Command)
	}
	if !strings.Contains(result.Command, "--output-format") {
		t.Errorf("Command must contain '--output-format': %q", result.Command)
	}
}

// =============================================================================
// Helper utilities for argument inspection
// =============================================================================

// containsArg reports whether flag appears in args.
func containsArg(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// containsConsecutive reports whether flag immediately followed by value appears in args.
func containsConsecutive(args []string, flag, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

// containsFlag reports whether "--flag value" pair appears anywhere in args.
func containsFlag(args []string, flag, value string) bool {
	return containsConsecutive(args, flag, value)
}

// countFlag counts how many times flag appears in args.
func countFlag(args []string, flag string) int {
	count := 0
	for _, a := range args {
		if a == flag {
			count++
		}
	}
	return count
}
