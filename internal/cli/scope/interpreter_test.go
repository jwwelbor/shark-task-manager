package scope

import (
	"testing"
)

func TestInterpreter_ParseScope(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantType  ScopeType
		wantKey   string
		wantError bool
	}{
		// Epic scope tests
		{
			name:     "epic scope - uppercase",
			args:     []string{"E01"},
			wantType: ScopeEpic,
			wantKey:  "E01",
		},
		{
			name:     "epic scope - lowercase",
			args:     []string{"e01"},
			wantType: ScopeEpic,
			wantKey:  "E01",
		},
		{
			name:     "epic scope - mixed case",
			args:     []string{"e10"},
			wantType: ScopeEpic,
			wantKey:  "E10",
		},

		// Feature scope tests - combined format
		{
			name:     "feature scope - combined uppercase",
			args:     []string{"E01-F01"},
			wantType: ScopeFeature,
			wantKey:  "E01-F01",
		},
		{
			name:     "feature scope - combined lowercase",
			args:     []string{"e01-f01"},
			wantType: ScopeFeature,
			wantKey:  "E01-F01",
		},
		{
			name:     "feature scope - combined mixed case",
			args:     []string{"E01-f01"},
			wantType: ScopeFeature,
			wantKey:  "E01-F01",
		},

		// Feature scope tests - separate args
		{
			name:     "feature scope - separate args uppercase",
			args:     []string{"E01", "F01"},
			wantType: ScopeFeature,
			wantKey:  "E01-F01",
		},
		{
			name:     "feature scope - separate args lowercase",
			args:     []string{"e01", "f01"},
			wantType: ScopeFeature,
			wantKey:  "E01-F01",
		},
		{
			name:     "feature scope - separate args mixed",
			args:     []string{"E01", "f01"},
			wantType: ScopeFeature,
			wantKey:  "E01-F01",
		},
		{
			name:     "feature scope - epic and full feature key",
			args:     []string{"E01", "E01-F01"},
			wantType: ScopeFeature,
			wantKey:  "E01-F01",
		},

		// Task scope tests - full key
		{
			name:     "task scope - full key uppercase",
			args:     []string{"T-E01-F01-001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},
		{
			name:     "task scope - full key lowercase",
			args:     []string{"t-e01-f01-001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},
		{
			name:     "task scope - full key mixed case",
			args:     []string{"T-e01-F01-001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},

		// Task scope tests - short key (without T- prefix)
		{
			name:     "task scope - short key uppercase",
			args:     []string{"E01-F01-001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},
		{
			name:     "task scope - short key lowercase",
			args:     []string{"e01-f01-001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},

		// Task scope tests - three args
		{
			name:     "task scope - three args uppercase",
			args:     []string{"E01", "F01", "001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},
		{
			name:     "task scope - three args lowercase",
			args:     []string{"e01", "f01", "001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},
		{
			name:     "task scope - three args numeric",
			args:     []string{"E01", "F01", "1"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},
		{
			name:     "task scope - three args with full feature key",
			args:     []string{"E01", "E01-F01", "001"},
			wantType: ScopeTask,
			wantKey:  "T-E01-F01-001",
		},

		// Error cases
		{
			name:      "error - no args",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "error - invalid epic key (E1)",
			args:      []string{"E1"},
			wantError: true,
		},
		{
			name:      "error - invalid epic key (E001)",
			args:      []string{"E001"},
			wantError: true,
		},
		{
			name:      "error - invalid feature key",
			args:      []string{"E01-F1"},
			wantError: true,
		},
		{
			name:      "error - too many args",
			args:      []string{"E01", "F01", "001", "extra"},
			wantError: true,
		},
		{
			name:      "error - invalid task number (0)",
			args:      []string{"E01", "F01", "0"},
			wantError: true,
		},
		{
			name:      "error - invalid task number (1000)",
			args:      []string{"E01", "F01", "1000"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interpreter := NewInterpreter()
			scope, err := interpreter.ParseScope(tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseScope() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseScope() unexpected error: %v", err)
				return
			}

			if scope.Type != tt.wantType {
				t.Errorf("ParseScope() Type = %v, want %v", scope.Type, tt.wantType)
			}

			if scope.Key != tt.wantKey {
				t.Errorf("ParseScope() Key = %v, want %v", scope.Key, tt.wantKey)
			}
		})
	}
}

func TestInterpreter_ParseScope_EdgeCases(t *testing.T) {
	interpreter := NewInterpreter()

	t.Run("handles slugged keys", func(t *testing.T) {
		// Task with slug
		scope, err := interpreter.ParseScope([]string{"T-E01-F01-001-implement-feature"})
		if err != nil {
			t.Errorf("ParseScope() unexpected error: %v", err)
		}
		if scope.Type != ScopeTask {
			t.Errorf("ParseScope() Type = %v, want %v", scope.Type, ScopeTask)
		}
		if scope.Key != "T-E01-F01-001-IMPLEMENT-FEATURE" {
			t.Errorf("ParseScope() Key = %v, want %v", scope.Key, "T-E01-F01-001-IMPLEMENT-FEATURE")
		}
	})

	t.Run("nil interpreter", func(t *testing.T) {
		var nilInterpreter *Interpreter
		_, err := nilInterpreter.ParseScope([]string{"E01"})
		if err == nil {
			t.Error("ParseScope() on nil interpreter should return error")
		}
	})
}
