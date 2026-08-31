package models

import (
	"errors"
	"testing"
)

// TestValidateSprintAssignmentEntityType_Valid covers every allowlisted
// entity_type that may appear in a sprint_assignment row.
func TestValidateSprintAssignmentEntityType_Valid(t *testing.T) {
	valid := []string{"task", "bug", "change_card", "tech_debt"}
	for _, et := range valid {
		t.Run(et, func(t *testing.T) {
			if err := ValidateSprintAssignmentEntityType(et); err != nil {
				t.Errorf("ValidateSprintAssignmentEntityType(%q) = %v, want nil", et, err)
			}
		})
	}
}

// TestValidateSprintAssignmentEntityType_Invalid rejects all values outside
// the {task, bug, change_card, tech_debt} allowlist. Note: epic, feature,
// and idea are intentionally NOT allowlisted — sprints are for execution-level
// work items only.
func TestValidateSprintAssignmentEntityType_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"idea - not allowlisted", "idea"},
		{"feature - not allowlisted", "feature"},
		{"epic - not allowlisted", "epic"},
		{"sprint - sprints don't assign sprints", "sprint"},
		{"uppercase TASK", "TASK"},
		{"mixed case Task", "Task"},
		{"whitespace-padded task", " task "},
		{"prefix-padded -task", "-task"},
		{"hyphen variant change-card (vs change_card)", "change-card"},
		{"hyphen variant tech-debt (vs tech_debt)", "tech-debt"},
		{"unknown type", "story"},
		{"sql injection", "task' OR 1=1"},
		{"null byte", "task\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSprintAssignmentEntityType(tt.entityType)
			if err == nil {
				t.Errorf("ValidateSprintAssignmentEntityType(%q) = nil, want error", tt.entityType)
				return
			}
			if !errors.Is(err, ErrInvalidSprintAssignmentEntityType) {
				t.Errorf("ValidateSprintAssignmentEntityType(%q) error = %v, want errors.Is(ErrInvalidSprintAssignmentEntityType)", tt.entityType, err)
			}
		})
	}
}

// TestErrInvalidSprintAssignmentEntityType_Message verifies the sentinel
// error message lists the allowed values so callers know what is permitted.
func TestErrInvalidSprintAssignmentEntityType_Message(t *testing.T) {
	msg := ErrInvalidSprintAssignmentEntityType.Error()
	for _, want := range []string{"task", "bug", "change_card", "tech_debt"} {
		if !contains(msg, want) {
			t.Errorf("ErrInvalidSprintAssignmentEntityType message %q does not mention %q", msg, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
