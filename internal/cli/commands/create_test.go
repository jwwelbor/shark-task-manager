package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// TestCreateDispatcherRegistered verifies the create dispatcher command is registered
func TestCreateDispatcherRegistered(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create command not found: %v", err)
	}
	if cmd == nil || cmd.Name() != "create" {
		t.Fatal("expected create command")
	}
}

// TestCreateDispatcherHasCorrectGroupID verifies create is in the essentials group
func TestCreateDispatcherHasCorrectGroupID(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("create command is nil")
	}

	if cmd.GroupID != "essentials" {
		t.Errorf("create command has GroupID %q, expected 'essentials'", cmd.GroupID)
	}
}

// TestCreateDispatcherSubcommands verifies all expected subcommands exist
func TestCreateDispatcherSubcommands(t *testing.T) {
	createCmd, _, err := cli.RootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create command not found: %v", err)
	}
	if createCmd == nil {
		t.Fatal("create command not found")
	}

	subcommandNames := []string{"epic", "feature", "task"}
	for _, name := range subcommandNames {
		t.Run(name, func(t *testing.T) {
			found := false
			for _, sub := range createCmd.Commands() {
				if sub.Name() == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("create subcommand %q not found", name)
			}
		})
	}
}

// TestCreateEpicHasExpectedFlags verifies create epic subcommand has correct flags
func TestCreateEpicHasExpectedFlags(t *testing.T) {
	createCmd, _, err := cli.RootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create command not found: %v", err)
	}

	var epicCmd *cobra.Command
	for _, sub := range createCmd.Commands() {
		if sub.Name() == "epic" {
			epicCmd = sub
			break
		}
	}
	if epicCmd == nil {
		t.Fatal("create epic subcommand not found")
	}

	expectedFlags := []string{"description", "key", "file", "force", "priority", "business-value", "status"}
	for _, flag := range expectedFlags {
		if epicCmd.Flags().Lookup(flag) == nil {
			t.Errorf("create epic missing expected flag: %s", flag)
		}
	}

	// Verify hidden aliases exist
	hiddenAliases := []string{"filename", "path"}
	for _, flag := range hiddenAliases {
		f := epicCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("create epic missing hidden alias flag: %s", flag)
		} else if !f.Hidden {
			t.Errorf("create epic flag %s should be hidden", flag)
		}
	}
}

// TestCreateFeatureHasExpectedFlags verifies create feature subcommand has correct flags
func TestCreateFeatureHasExpectedFlags(t *testing.T) {
	createCmd, _, err := cli.RootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create command not found: %v", err)
	}

	var featureCmd *cobra.Command
	for _, sub := range createCmd.Commands() {
		if sub.Name() == "feature" {
			featureCmd = sub
			break
		}
	}
	if featureCmd == nil {
		t.Fatal("create feature subcommand not found")
	}

	expectedFlags := []string{"epic", "description", "execution-order", "key", "force", "status", "file"}
	for _, flag := range expectedFlags {
		if featureCmd.Flags().Lookup(flag) == nil {
			t.Errorf("create feature missing expected flag: %s", flag)
		}
	}

	// Verify hidden aliases exist
	hiddenAliases := []string{"filename", "path"}
	for _, flag := range hiddenAliases {
		f := featureCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("create feature missing hidden alias flag: %s", flag)
		} else if !f.Hidden {
			t.Errorf("create feature flag %s should be hidden", flag)
		}
	}
}

// TestCreateTaskHasExpectedFlags verifies create task subcommand has correct flags
func TestCreateTaskHasExpectedFlags(t *testing.T) {
	createCmd, _, err := cli.RootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create command not found: %v", err)
	}

	var taskCmd *cobra.Command
	for _, sub := range createCmd.Commands() {
		if sub.Name() == "task" {
			taskCmd = sub
			break
		}
	}
	if taskCmd == nil {
		t.Fatal("create task subcommand not found")
	}

	expectedFlags := []string{
		"epic", "feature", "agent", "description", "priority",
		"depends-on", "execution-order", "order", "key", "force", "create", "file",
	}
	for _, flag := range expectedFlags {
		if taskCmd.Flags().Lookup(flag) == nil {
			t.Errorf("create task missing expected flag: %s", flag)
		}
	}

	// Verify hidden aliases exist
	hiddenAliases := []string{"filename", "path"}
	for _, flag := range hiddenAliases {
		f := taskCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("create task missing hidden alias flag: %s", flag)
		} else if !f.Hidden {
			t.Errorf("create task flag %s should be hidden", flag)
		}
	}
}

// TestCreateSubcommandsHaveCorrectArgsValidation verifies argument requirements
func TestCreateSubcommandsHaveCorrectArgsValidation(t *testing.T) {
	createCmd, _, err := cli.RootCmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create command not found: %v", err)
	}

	tests := []struct {
		name      string
		subcmd    string
		wantArgs  bool // true if Args validator should exist
		wantRange bool // true if should accept range of args
	}{
		{"epic requires args", "epic", true, false},
		{"feature accepts range", "feature", true, true},
		{"task accepts range", "task", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var subcmd *cobra.Command
			for _, sub := range createCmd.Commands() {
				if sub.Name() == tt.subcmd {
					subcmd = sub
					break
				}
			}
			if subcmd == nil {
				t.Fatalf("create %s subcommand not found", tt.subcmd)
			}

			if tt.wantArgs && subcmd.Args == nil {
				t.Errorf("create %s should have Args validator", tt.subcmd)
			}
		})
	}
}
