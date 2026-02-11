package commands

import (
	"testing"
)

func TestEpicSetStatusCmd_Registration(t *testing.T) {
	// Verify the command is properly registered as subcommand of epicCmd
	found := false
	for _, cmd := range epicCmd.Commands() {
		if cmd.Use == "set-status <epic-key> <status>" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected epic set-status command to be registered as subcommand of epicCmd")
	}
}

func TestEpicSetStatusCmd_Args(t *testing.T) {
	cmd := epicSetStatusCmd

	// Verify exact 2 args required
	if err := cmd.Args(cmd, []string{"E16"}); err == nil {
		t.Error("expected error with 1 arg, got nil")
	}

	if err := cmd.Args(cmd, []string{"E16", "active"}); err != nil {
		t.Errorf("expected no error with 2 args, got: %v", err)
	}

	if err := cmd.Args(cmd, []string{"E16", "active", "extra"}); err == nil {
		t.Error("expected error with 3 args, got nil")
	}
}

func TestEpicSetStatusCmd_Flags(t *testing.T) {
	cmd := epicSetStatusCmd

	// Verify flags are registered
	flags := []string{"reason", "force", "agent"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag %q to be registered", flag)
		}
	}
}
