package commands

import (
	"testing"
)

func TestFeatureSetStatusCmd_Registration(t *testing.T) {
	// Verify the command is properly registered as subcommand of featureCmd
	found := false
	for _, cmd := range featureCmd.Commands() {
		if cmd.Use == "set-status <feature-key> <status>" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected feature set-status command to be registered as subcommand of featureCmd")
	}
}

func TestFeatureSetStatusCmd_Args(t *testing.T) {
	cmd := featureSetStatusCmd

	// Verify exact 2 args required
	if err := cmd.Args(cmd, []string{"E16-F01"}); err == nil {
		t.Error("expected error with 1 arg, got nil")
	}

	if err := cmd.Args(cmd, []string{"E16-F01", "active"}); err != nil {
		t.Errorf("expected no error with 2 args, got: %v", err)
	}

	if err := cmd.Args(cmd, []string{"E16-F01", "active", "extra"}); err == nil {
		t.Error("expected error with 3 args, got nil")
	}
}

func TestFeatureSetStatusCmd_Flags(t *testing.T) {
	cmd := featureSetStatusCmd

	// Verify flags are registered
	flags := []string{"reason", "force", "agent"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag %q to be registered", flag)
		}
	}
}
