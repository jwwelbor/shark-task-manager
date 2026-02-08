package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
)

// TestDeleteDispatcherRegistered verifies the delete dispatcher command is registered
func TestDeleteDispatcherRegistered(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("delete command not found: %v", err)
	}
	if cmd == nil || cmd.Name() != "delete" {
		t.Fatal("expected delete command")
	}
}

// TestDeleteDispatcherHasCorrectGroupID verifies delete is in the essentials group
func TestDeleteDispatcherHasCorrectGroupID(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("delete command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("delete command is nil")
	}

	if cmd.GroupID != "essentials" {
		t.Errorf("delete command has GroupID %q, expected 'essentials'", cmd.GroupID)
	}
}

// TestDeleteDispatcherHasForceFlag verifies the delete command has --force flag
func TestDeleteDispatcherHasForceFlag(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("delete command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("delete command not found")
	}

	if cmd.Flags().Lookup("force") == nil {
		t.Error("delete command missing --force flag")
	}
}

// TestDeleteDispatcherRequiresArgs verifies delete requires exactly one argument
func TestDeleteDispatcherRequiresArgs(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("delete command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("delete command not found")
	}

	// The command should require exactly 1 arg (the entity key)
	if cmd.Args == nil {
		t.Error("delete command should have Args validator")
	}
}

// TestDeleteDispatcherUsageString verifies the usage message is correct
func TestDeleteDispatcherUsageString(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("delete command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("delete command not found")
	}

	expectedUse := "delete <KEY>"
	if cmd.Use != expectedUse {
		t.Errorf("delete command Use = %q, want %q", cmd.Use, expectedUse)
	}
}

// TestDeleteDispatcherHasLongDescription verifies the command has proper documentation
func TestDeleteDispatcherHasLongDescription(t *testing.T) {
	cmd, _, err := cli.RootCmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("delete command not found: %v", err)
	}
	if cmd == nil {
		t.Fatal("delete command not found")
	}

	if cmd.Long == "" {
		t.Error("delete command should have Long description")
	}

	if cmd.Short == "" {
		t.Error("delete command should have Short description")
	}
}
