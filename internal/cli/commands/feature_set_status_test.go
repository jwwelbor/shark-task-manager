package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestFeatureSetStatusCmd_Registration(t *testing.T) {
	found := false
	for _, cmd := range featureCmd.Commands() {
		if cmd.Name() == "set-status" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected feature set-status command to be registered as subcommand of featureCmd")
	}
}

func TestFeatureSetStatusCmd_Args(t *testing.T) {
	var cmd *cobra.Command
	for _, c := range featureCmd.Commands() {
		if c.Name() == "set-status" {
			cmd = c
			break
		}
	}
	if cmd == nil {
		t.Fatal("set-status command not found")
	}

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
	var cmd *cobra.Command
	for _, c := range featureCmd.Commands() {
		if c.Name() == "set-status" {
			cmd = c
			break
		}
	}
	if cmd == nil {
		t.Fatal("set-status command not found")
	}

	flags := []string{"reason", "force", "agent"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag %q to be registered", flag)
		}
	}
}
