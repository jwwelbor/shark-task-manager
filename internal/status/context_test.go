package status

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestGetStatusContext(t *testing.T) {
	tests := []struct {
		name            string
		statusCounts    map[string]int
		cfg             *config.WorkflowConfig
		expectedContext string
	}{
		{
			name:            "waiting: ready_for_approval",
			statusCounts:    map[string]int{"ready_for_approval": 1},
			cfg:             nil,
			expectedContext: "active (waiting)",
		},
		{
			name:            "waiting: ready_for_code_review",
			statusCounts:    map[string]int{"ready_for_code_review": 2},
			cfg:             nil,
			expectedContext: "active (waiting)",
		},
		{
			name:            "waiting: both approval and review",
			statusCounts:    map[string]int{"ready_for_approval": 1, "ready_for_code_review": 1},
			cfg:             nil,
			expectedContext: "active (waiting)",
		},
		{
			name:            "blocked",
			statusCounts:    map[string]int{"blocked": 1},
			cfg:             nil,
			expectedContext: "active (blocked)",
		},
		{
			name:            "blocked takes precedence over development",
			statusCounts:    map[string]int{"blocked": 1, "in_progress": 2},
			cfg:             nil,
			expectedContext: "active (blocked)",
		},
		{
			name:            "development: in_progress",
			statusCounts:    map[string]int{"in_progress": 3},
			cfg:             nil,
			expectedContext: "active (development)",
		},
		{
			name:            "development: in_development",
			statusCounts:    map[string]int{"in_development": 2},
			cfg:             nil,
			expectedContext: "active (development)",
		},
		{
			name:            "development: both in_progress and in_development",
			statusCounts:    map[string]int{"in_progress": 1, "in_development": 1},
			cfg:             nil,
			expectedContext: "active (development)",
		},
		{
			name:            "active: todo status only",
			statusCounts:    map[string]int{"todo": 5},
			cfg:             nil,
			expectedContext: "active",
		},
		{
			name:            "active: completed status only",
			statusCounts:    map[string]int{"completed": 3},
			cfg:             nil,
			expectedContext: "active",
		},
		{
			name:            "waiting takes precedence over development",
			statusCounts:    map[string]int{"ready_for_approval": 1, "in_progress": 2},
			cfg:             nil,
			expectedContext: "active (waiting)",
		},
		{
			name:            "waiting takes precedence over blocked",
			statusCounts:    map[string]int{"ready_for_approval": 1, "blocked": 1},
			cfg:             nil,
			expectedContext: "active (waiting)",
		},
		{
			name:            "empty status counts",
			statusCounts:    map[string]int{},
			cfg:             nil,
			expectedContext: "active",
		},
		{
			name:            "development with multiple other statuses",
			statusCounts:    map[string]int{"todo": 2, "in_progress": 1, "completed": 1},
			cfg:             nil,
			expectedContext: "active (development)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := GetStatusContext(tt.statusCounts, tt.cfg)
			if context != tt.expectedContext {
				t.Errorf("GetStatusContext() = %q, want %q", context, tt.expectedContext)
			}
		})
	}
}

func TestGetStatusContextPriority(t *testing.T) {
	// Test priority: waiting > blocked > development > active
	t.Run("priority: waiting > blocked > development", func(t *testing.T) {
		statusCounts := map[string]int{
			"ready_for_approval": 1,
			"blocked":            1,
			"in_progress":        1,
		}

		context := GetStatusContext(statusCounts, nil)
		if context != "active (waiting)" {
			t.Errorf("Expected 'active (waiting)', got %q", context)
		}
	})

	t.Run("priority: blocked > development", func(t *testing.T) {
		statusCounts := map[string]int{
			"blocked":     1,
			"in_progress": 2,
		}

		context := GetStatusContext(statusCounts, nil)
		if context != "active (blocked)" {
			t.Errorf("Expected 'active (blocked)', got %q", context)
		}
	})

	t.Run("priority: development > other active", func(t *testing.T) {
		statusCounts := map[string]int{
			"in_progress": 1,
			"todo":        2,
			"completed":   1,
		}

		context := GetStatusContext(statusCounts, nil)
		if context != "active (development)" {
			t.Errorf("Expected 'active (development)', got %q", context)
		}
	})
}
