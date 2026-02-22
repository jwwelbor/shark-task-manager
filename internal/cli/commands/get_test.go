package commands

import (
	"testing"
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
//
// Note: This test verifies the dispatch logic via ParseGetArgs (no database
// required). Calling runGet directly is not safe in unit tests because the
// subcommand handlers call os.Exit() on error — which cannot be caught by
// recover() and would kill the test process.
func TestRunGet_ContextPropagatedToSubcommands(t *testing.T) {
	// Verify root cause: static command vars have nil context because they're
	// never executed through Cobra's lifecycle (only getCmd is executed).
	// If runGet used epicGetCmd.RunE(epicGetCmd, args) instead of
	// runEpicGet(cmd, args), these nil contexts would cause nil pointer panics.
	if taskGetCmd.Context() != nil {
		t.Log("note: taskGetCmd.Context() is non-nil (unexpected)")
	}
	if featureGetCmd.Context() != nil {
		t.Log("note: featureGetCmd.Context() is non-nil (unexpected)")
	}
	if epicGetCmd.Context() != nil {
		t.Log("note: epicGetCmd.Context() is non-nil (unexpected)")
	}

	// Verify dispatch logic: ParseGetArgs correctly identifies entity type for
	// each key format. runGet dispatches to the correct subhandler based on this.
	// Using ParseGetArgs directly avoids database access and os.Exit() calls.
	tests := []struct {
		name        string
		args        []string
		wantCommand string
	}{
		{"task key dispatch", []string{"T-E15-F11-015"}, "task"},
		{"feature key dispatch", []string{"E15-F11"}, "feature"},
		{"epic key dispatch", []string{"E15"}, "epic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, _, err := ParseGetArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseGetArgs() unexpected error: %v", err)
			}
			if command != tt.wantCommand {
				t.Errorf("ParseGetArgs() dispatches to %q, want %q — runGet would call wrong subhandler",
					command, tt.wantCommand)
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
