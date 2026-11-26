package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// B088 F-3e: the boundaries of what counts as a write attempt. Every case that
// is not provably read-only must be a write, so a malformed command line can
// never hide an attempt from the ledger.
func TestClassifyTrackerAttemptBoundaries(t *testing.T) {
	noop := func(*cobra.Command, []string) {}
	task := &cobra.Command{Use: "task"}
	for _, leaf := range []string{"get", "list", "create", "history"} {
		task.AddCommand(&cobra.Command{Use: leaf, Run: noop})
	}
	create := &cobra.Command{Use: "create"}
	create.AddCommand(&cobra.Command{Use: "note", Run: noop})
	RootCmd.AddCommand(task, create)
	t.Cleanup(func() { RootCmd.RemoveCommand(task, create) })

	for _, tc := range []struct {
		name    string
		args    []string
		command string
		write   bool
	}{
		{"read-only leaf", []string{"task", "get", "E01"}, "task get", false},
		{"read-only leaf with flags", []string{"task", "list", "--json"}, "task list", false},
		{"mutating leaf", []string{"task", "create", "x"}, "task create", true},
		{"nested mutating leaf", []string{"create", "note", "B001"}, "create note", true},
		{"read-only name under another group", []string{"create", "history"}, "create", true},
		{"unknown subcommand under a group", []string{"task", "frobnicate"}, "task", true},
		{"unknown top-level command", []string{"no-such-command"}, "unresolved", true},
		{"bare root", nil, "shark", true},
		{"root help", []string{"--help"}, "shark", false},
		{"root version", []string{"--version"}, "shark", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command, write := classifyTrackerAttempt(tc.args)
			require.Equal(t, tc.write, write)
			if tc.command != "" && tc.command != "unresolved" {
				require.Equal(t, tc.command, command)
			}
		})
	}
}
