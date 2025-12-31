package commands

import (
	"testing"
)

// TestParseListArgs tests the parsing of positional arguments for the list command
func TestParseListArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string // "epic", "feature", or "task"
		wantEpic    *string
		wantFeature *string
		wantErr     bool
	}{
		{
			name:        "no args - list epics",
			args:        []string{},
			wantCommand: "epic",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     false,
		},
		{
			name:        "epic key - list features in epic",
			args:        []string{"E10"},
			wantCommand: "feature",
			wantEpic:    listStringPtr("E10"),
			wantFeature: nil,
			wantErr:     false,
		},
		{
			name:        "feature key combined - list tasks in feature",
			args:        []string{"E10-F01"},
			wantCommand: "task",
			wantEpic:    listStringPtr("E10"),
			wantFeature: listStringPtr("F01"),
			wantErr:     false,
		},
		{
			name:        "epic and feature separate - list tasks",
			args:        []string{"E10", "F01"},
			wantCommand: "task",
			wantEpic:    listStringPtr("E10"),
			wantFeature: listStringPtr("F01"),
			wantErr:     false,
		},
		{
			name:        "epic and feature full key - list tasks",
			args:        []string{"E10", "E10-F01"},
			wantCommand: "task",
			wantEpic:    listStringPtr("E10"),
			wantFeature: listStringPtr("F01"),
			wantErr:     false,
		},
		{
			name:        "invalid epic format",
			args:        []string{"E1"},
			wantCommand: "",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
		},
		{
			name:        "invalid feature format",
			args:        []string{"E10-F1"},
			wantCommand: "",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
		},
		{
			name:        "too many args",
			args:        []string{"E10", "F01", "extra"},
			wantCommand: "",
			wantEpic:    nil,
			wantFeature: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, epic, feature, err := ParseListArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseListArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if command != tt.wantCommand {
				t.Errorf("ParseListArgs() command = %v, want %v", command, tt.wantCommand)
			}

			if !listStringPtrEqual(epic, tt.wantEpic) {
				t.Errorf("ParseListArgs() epic = %v, want %v", listStringPtrValue(epic), listStringPtrValue(tt.wantEpic))
			}

			if !listStringPtrEqual(feature, tt.wantFeature) {
				t.Errorf("ParseListArgs() feature = %v, want %v", listStringPtrValue(feature), listStringPtrValue(tt.wantFeature))
			}
		})
	}
}

// Helper functions for testing
func listStringPtr(s string) *string {
	return &s
}

func listStringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func listStringPtrValue(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
