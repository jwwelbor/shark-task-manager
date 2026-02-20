package commands

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRunGet_ContextPropagatedToSubcommands verifies that runGet passes the live
// executing command's context to subcommand handlers — not the static package-level
// command vars (taskGetCmd, epicGetCmd, featureGetCmd) which have nil context.
//
// Root cause: runGet dispatched with static *cobra.Command vars that are never
// executed through Cobra's lifecycle, so Context() returns nil. Nil context
// panics in database/sql: "invalid memory address or nil pointer dereference".
//
// Fix: pass cmd (the live executing command with real context) to subhandlers.
func TestRunGet_ContextPropagatedToSubcommands(t *testing.T) {
	// Verify root cause: static command vars have nil context because they're
	// never executed through Cobra's lifecycle (only getCmd is executed).
	if taskGetCmd.Context() != nil {
		t.Log("note: taskGetCmd.Context() is non-nil (unexpected)")
	}
	if featureGetCmd.Context() != nil {
		t.Log("note: featureGetCmd.Context() is non-nil (unexpected)")
	}
	if epicGetCmd.Context() != nil {
		t.Log("note: epicGetCmd.Context() is non-nil (unexpected)")
	}

	tests := []struct {
		name string
		args []string
	}{
		{"task key dispatch", []string{"T-E15-F11-015"}},
		{"feature key dispatch", []string{"E15-F11"}},
		{"epic key dispatch", []string{"E15"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create cmd with proper context, as Cobra sets during real execution.
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Capture any panic. Before fix: nil pointer dereference (nil context
			// from static command var). After fix: error return, no panic.
			var panicMsg string
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicMsg = fmt.Sprintf("%v", r)
					}
				}()
				_ = runGet(cmd, tt.args)
			}()

			if strings.Contains(panicMsg, "invalid memory address") ||
				strings.Contains(panicMsg, "nil pointer dereference") {
				t.Errorf("runGet panicked with nil pointer — context not propagated to subcommand.\n"+
					"Bug: runGet passes static command var with nil Context().\n"+
					"Fix: pass cmd (the live executing command) instead.\n"+
					"Panic: %s", panicMsg)
			}
		})
	}
}

// TestParseGetArgs tests the parsing of positional arguments for the get command
func TestParseGetArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string // "epic", "feature", or "task"
		wantKey     string // The key to pass to the get command
		wantErr     bool
	}{
		{
			name:        "epic key - get epic",
			args:        []string{"E10"},
			wantCommand: "epic",
			wantKey:     "E10",
			wantErr:     false,
		},
		{
			name:        "epic and feature separate - get feature",
			args:        []string{"E10", "F01"},
			wantCommand: "feature",
			wantKey:     "E10-F01",
			wantErr:     false,
		},
		{
			name:        "feature key combined - get feature",
			args:        []string{"E10-F01"},
			wantCommand: "feature",
			wantKey:     "E10-F01",
			wantErr:     false,
		},
		{
			name:        "epic + feature + task number - get task",
			args:        []string{"E10", "F01", "001"},
			wantCommand: "task",
			wantKey:     "T-E10-F01-001",
			wantErr:     false,
		},
		{
			name:        "epic + feature + task number short - get task",
			args:        []string{"E10", "F01", "1"},
			wantCommand: "task",
			wantKey:     "T-E10-F01-001",
			wantErr:     false,
		},
		{
			name:        "full task key - get task",
			args:        []string{"T-E10-F01-001"},
			wantCommand: "task",
			wantKey:     "T-E10-F01-001",
			wantErr:     false,
		},
		{
			name:        "full task key uppercase - get task",
			args:        []string{"T-E05-F02-012"},
			wantCommand: "task",
			wantKey:     "T-E05-F02-012",
			wantErr:     false,
		},
		{
			name:        "epic + feature suffix + task short - get task",
			args:        []string{"E10", "F01", "5"},
			wantCommand: "task",
			wantKey:     "T-E10-F01-005",
			wantErr:     false,
		},
		{
			name:        "no args - error",
			args:        []string{},
			wantCommand: "",
			wantKey:     "",
			wantErr:     true,
		},
		{
			name:        "invalid epic format",
			args:        []string{"E1"},
			wantCommand: "",
			wantKey:     "",
			wantErr:     true,
		},
		{
			name:        "invalid feature format",
			args:        []string{"E10-F1"},
			wantCommand: "",
			wantKey:     "",
			wantErr:     true,
		},
		{
			name:        "too many args",
			args:        []string{"E10", "F01", "001", "extra"},
			wantCommand: "",
			wantKey:     "",
			wantErr:     true,
		},
		{
			name:        "invalid task number - not numeric",
			args:        []string{"E10", "F01", "abc"},
			wantCommand: "",
			wantKey:     "",
			wantErr:     true,
		},
		{
			name:        "invalid task number - too large",
			args:        []string{"E10", "F01", "1000"},
			wantCommand: "",
			wantKey:     "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, key, err := ParseGetArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGetArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if command != tt.wantCommand {
				t.Errorf("ParseGetArgs() command = %v, want %v", command, tt.wantCommand)
			}

			if key != tt.wantKey {
				t.Errorf("ParseGetArgs() key = %v, want %v", key, tt.wantKey)
			}
		})
	}
}
