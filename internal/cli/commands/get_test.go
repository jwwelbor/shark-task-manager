package commands

import (
	"testing"
)

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
