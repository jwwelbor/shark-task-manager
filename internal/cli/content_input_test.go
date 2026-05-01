package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestResolveContentInput_FlagOnly verifies that a --content flag value is
// returned when stdin is a TTY (the default in tests). Stdin piping is awkward
// to simulate inside the test process, so the flag path is the practical check
// for `ResolveContentInput`.
func TestResolveContentInput_FlagOnly(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("content", "", "")
	if err := cmd.Flags().Set("content", "hello body"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	got, err := ResolveContentInput(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello body" {
		t.Errorf("expected %q, got %q", "hello body", got)
	}
}

// TestResolveContentInput_NoFlag returns "" when the --content flag is empty
// and stdin is a TTY.
func TestResolveContentInput_NoFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("content", "", "")

	got, err := ResolveContentInput(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestResolveContentInput_FlagNotRegistered tolerates commands that haven't
// declared a --content flag — the function falls through to "".
func TestResolveContentInput_FlagNotRegistered(t *testing.T) {
	cmd := &cobra.Command{}
	got, err := ResolveContentInput(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
