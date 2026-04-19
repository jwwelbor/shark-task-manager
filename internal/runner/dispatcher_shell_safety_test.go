package runner

import (
	"context"
	"strings"
	"testing"
)

// =============================================================================
// Shell-safety tests for BuildCommand
//
// Both ClaudeDispatcher and CodexDispatcher expose a BuildCommand method that
// returns a human-readable string representation of the subprocess invocation.
// The `command` field on the run.stage.dispatch slog event is populated from
// this string.
//
// The logged string must be the exact shell-equivalent invocation used by
// execAndCapture. Concretely: pasting the string into a POSIX shell must
// reproduce the same argv that Dispatch passes to exec.
//
// These tests tokenize the output of BuildCommand using a minimal shell-safe
// splitter (shellSplitForTest, defined in shell_quote_test.go) and verify
// the round-trip matches the argv built by buildClaudeArgs / buildCodexArgs.
// =============================================================================

// assertCommandRoundTrip verifies that splitting `got` via shell tokenization
// produces argv exactly equal to [bin, expectedArgs...]. Fails the test
// (with context) on any mismatch.
func assertCommandRoundTrip(t *testing.T, got, bin string, expectedArgs []string) {
	t.Helper()
	tokens, err := shellSplitForTest(got)
	if err != nil {
		t.Fatalf("command %q is not shell-tokenizable: %v", got, err)
	}
	if len(tokens) == 0 {
		t.Fatalf("command %q produced no tokens", got)
	}
	if tokens[0] != bin {
		t.Errorf("first token = %q, want %q (full=%q)", tokens[0], bin, got)
	}
	argv := tokens[1:]
	if len(argv) != len(expectedArgs) {
		t.Errorf("argv length = %d, want %d\n  got:  %q\n  want: %q\n  cmd:  %s",
			len(argv), len(expectedArgs), argv, expectedArgs, got)
		return
	}
	for i := range expectedArgs {
		if argv[i] != expectedArgs[i] {
			t.Errorf("argv[%d] = %q, want %q (cmd=%s)", i, argv[i], expectedArgs[i], got)
		}
	}
}

// -----------------------------------------------------------------------------
// ClaudeDispatcher.BuildCommand shell-safety
// -----------------------------------------------------------------------------

func TestClaudeDispatcher_BuildCommand_PlainInstructionUnquoted(t *testing.T) {
	d := NewClaudeDispatcher()
	in := DispatchInput{Instruction: "plain", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// A safe single-word instruction should NOT be wrapped in quotes; keeps
	// the logged string readable in the common case.
	if strings.Contains(got, "'plain'") {
		t.Errorf("safe instruction should not be quoted: %q", got)
	}
	assertCommandRoundTrip(t, got, "claude", buildClaudeArgs(in))
}

func TestClaudeDispatcher_BuildCommand_InstructionWithSpaces(t *testing.T) {
	d := NewClaudeDispatcher()
	in := DispatchInput{Instruction: "do work now", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// Must wrap in quotes so shell sees a single argument.
	if !strings.Contains(got, "'do work now'") {
		t.Errorf("instruction with spaces must be single-quoted, got: %q", got)
	}
	assertCommandRoundTrip(t, got, "claude", buildClaudeArgs(in))
}

func TestClaudeDispatcher_BuildCommand_InstructionWithSingleQuote(t *testing.T) {
	d := NewClaudeDispatcher()
	in := DispatchInput{Instruction: "it's fine", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// POSIX-safe escaping: ' inside single quotes is written as '\''.
	if !strings.Contains(got, `'it'\''s fine'`) {
		t.Errorf("instruction with single quote must be POSIX-escaped, got: %q", got)
	}
	assertCommandRoundTrip(t, got, "claude", buildClaudeArgs(in))
}

func TestClaudeDispatcher_BuildCommand_InstructionWithMetacharacters(t *testing.T) {
	d := NewClaudeDispatcher()
	// Contains a selection of dangerous shell metacharacters; if any of these
	// are not escaped the logged command string could execute arbitrary code
	// when pasted into a shell.
	instruction := `rm -rf /; echo "pwned" && cat $HOME | grep secret`
	in := DispatchInput{Instruction: instruction, EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// The instruction must appear exactly once in quoted form.
	assertCommandRoundTrip(t, got, "claude", buildClaudeArgs(in))
}

func TestClaudeDispatcher_BuildCommand_EmptyInstruction(t *testing.T) {
	d := NewClaudeDispatcher()
	in := DispatchInput{Instruction: "", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	// Empty instruction must still appear in the logged string as '' so
	// splitting recovers an empty-string argv element.
	if !strings.Contains(got, "''") {
		t.Errorf("empty instruction must be rendered as '', got: %q", got)
	}
	assertCommandRoundTrip(t, got, "claude", buildClaudeArgs(in))
}

func TestClaudeDispatcher_BuildCommand_AllowedToolsWithCommas(t *testing.T) {
	d := NewClaudeDispatcher()
	in := DispatchInput{
		Instruction:  "do stuff",
		EntityKey:    "E07-F41-003",
		AllowedTools: []string{"Read,Write,Bash"},
		Model:        "sonnet",
		MaxTurns:     5,
	}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	assertCommandRoundTrip(t, got, "claude", buildClaudeArgs(in))
}

// TestClaudeDispatcher_BuildCommand_MatchesDispatchCommandString asserts that
// BuildCommand and Dispatch produce the same command string, so the field
// logged on stage.dispatch matches the field set on the DispatchResult.
// Without this guarantee the two construction sites can drift.
func TestClaudeDispatcher_BuildCommand_MatchesDispatchCommandString(t *testing.T) {
	var captured capturedCmd
	d := newTestDispatcher(&captured)
	in := DispatchInput{
		Instruction:  "it's a \"test\" with $vars",
		EntityKey:    "E07-F41-003",
		AllowedTools: []string{"Read", "Write"},
	}

	preLogged, buildErr := d.BuildCommand(in)
	if buildErr != nil {
		t.Fatalf("BuildCommand() error = %v", buildErr)
	}
	result, err := d.Dispatch(context.Background(), in)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result == nil {
		t.Fatal("Dispatch() returned nil result")
	}
	if result.Command != preLogged {
		t.Errorf("BuildCommand/Dispatch drift:\n  BuildCommand: %q\n  Dispatch:     %q",
			preLogged, result.Command)
	}
}

// -----------------------------------------------------------------------------
// CodexDispatcher.BuildCommand shell-safety
// -----------------------------------------------------------------------------

func TestCodexDispatcher_BuildCommand_PlainInstructionUnquoted(t *testing.T) {
	d := NewCodexDispatcher()
	in := DispatchInput{Instruction: "plain", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if strings.Contains(got, "'plain'") {
		t.Errorf("safe instruction should not be quoted: %q", got)
	}
	assertCommandRoundTrip(t, got, "codex", buildCodexArgs(in))
}

func TestCodexDispatcher_BuildCommand_InstructionWithSpaces(t *testing.T) {
	d := NewCodexDispatcher()
	in := DispatchInput{Instruction: "do work now", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if !strings.Contains(got, "'do work now'") {
		t.Errorf("instruction with spaces must be single-quoted, got: %q", got)
	}
	assertCommandRoundTrip(t, got, "codex", buildCodexArgs(in))
}

func TestCodexDispatcher_BuildCommand_InstructionWithSingleQuote(t *testing.T) {
	d := NewCodexDispatcher()
	in := DispatchInput{Instruction: "it's fine", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if !strings.Contains(got, `'it'\''s fine'`) {
		t.Errorf("instruction with single quote must be POSIX-escaped, got: %q", got)
	}
	assertCommandRoundTrip(t, got, "codex", buildCodexArgs(in))
}

func TestCodexDispatcher_BuildCommand_InstructionWithMetacharacters(t *testing.T) {
	d := NewCodexDispatcher()
	instruction := `rm -rf /; echo "pwned" && cat $HOME | grep secret`
	in := DispatchInput{Instruction: instruction, EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	assertCommandRoundTrip(t, got, "codex", buildCodexArgs(in))
}

func TestCodexDispatcher_BuildCommand_EmptyInstruction(t *testing.T) {
	d := NewCodexDispatcher()
	in := DispatchInput{Instruction: "", EntityKey: "E07-F41-003"}

	got, err := d.BuildCommand(in)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	if !strings.Contains(got, "''") {
		t.Errorf("empty instruction must be rendered as '', got: %q", got)
	}
	assertCommandRoundTrip(t, got, "codex", buildCodexArgs(in))
}

// TestCodexDispatcher_BuildCommand_MatchesDispatchCommandString asserts that
// BuildCommand and Dispatch produce the same command string.
func TestCodexDispatcher_BuildCommand_MatchesDispatchCommandString(t *testing.T) {
	var captured capturedCmd
	d := &CodexDispatcher{
		cmdFactory:   recordingFactory(&captured),
		lookPathFunc: successLookPath,
	}
	in := DispatchInput{
		Instruction: "it's a \"test\" with $vars",
		EntityKey:   "E07-F41-003",
		Model:       "gpt-5",
	}

	preLogged, buildErr := d.BuildCommand(in)
	if buildErr != nil {
		t.Fatalf("BuildCommand() error = %v", buildErr)
	}
	result, err := d.Dispatch(context.Background(), in)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result == nil {
		t.Fatal("Dispatch() returned nil result")
	}
	if result.Command != preLogged {
		t.Errorf("BuildCommand/Dispatch drift:\n  BuildCommand: %q\n  Dispatch:     %q",
			preLogged, result.Command)
	}
}
