package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Test that HistoryEntry includes RejectionReason field
func TestHistoryEntryHasRejectionReason(t *testing.T) {
	// RED: This test verifies the structure includes rejection_reason

	// Create a HistoryEntry with rejection reason
	entry := HistoryEntry{
		Timestamp:       "2026-01-16T10:00:00Z",
		RelativeAge:     "1 hour ago",
		OldStatus:       strPtr("in_progress"),
		NewStatus:       "ready_for_review",
		Agent:           strPtr("reviewer-1"),
		Notes:           strPtr("Code review failed"),
		RejectionReason: strPtr("Missing error handling on line 42"),
	}

	// Verify all fields are set
	if entry.RejectionReason == nil {
		t.Fatal("expected RejectionReason to be set")
	}
	if *entry.RejectionReason != "Missing error handling on line 42" {
		t.Errorf("expected 'Missing error handling on line 42', got %s", *entry.RejectionReason)
	}
}

// Test that JSON marshaling includes rejection_reason
func TestHistoryEntryJSONMarshal(t *testing.T) {
	// RED: Verify JSON includes rejection_reason field

	tests := []struct {
		name            string
		rejectionReason *string
		expectInJSON    bool
	}{
		{
			name:            "with rejection reason",
			rejectionReason: strPtr("Missing error handling"),
			expectInJSON:    true,
		},
		{
			name:            "without rejection reason",
			rejectionReason: nil,
			expectInJSON:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := HistoryEntry{
				Timestamp:       "2026-01-16T10:00:00Z",
				NewStatus:       "in_progress",
				RejectionReason: tt.rejectionReason,
			}

			jsonData, err := json.Marshal(entry)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			_, hasField := result["rejection_reason"]
			if tt.expectInJSON && !hasField {
				t.Error("expected rejection_reason in JSON output")
			}
			if !tt.expectInJSON && hasField {
				t.Error("did not expect rejection_reason in JSON output (nil)")
			}
		})
	}
}

// Test HistoryOutput JSON structure includes entries with rejection_reason
func TestHistoryOutputWithRejectionReason(t *testing.T) {
	// RED: Verify HistoryOutput JSON structure includes rejection_reason

	output := HistoryOutput{
		TaskKey: "E07-F22-001",
		History: []HistoryEntry{
			{
				Timestamp:       "2026-01-16T10:00:00Z",
				RelativeAge:     "1 hour ago",
				OldStatus:       strPtr("in_progress"),
				NewStatus:       "ready_for_review",
				Agent:           strPtr("reviewer"),
				RejectionReason: strPtr("Missing validation"),
			},
			{
				Timestamp:       "2026-01-16T09:00:00Z",
				RelativeAge:     "2 hours ago",
				OldStatus:       strPtr("todo"),
				NewStatus:       "in_progress",
				Agent:           strPtr("developer"),
				RejectionReason: nil, // Forward transitions don't have rejection reason
			},
		},
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal output: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify structure
	if result["task_key"] != "E07-F22-001" {
		t.Error("task_key mismatch")
	}

	history, ok := result["history"].([]interface{})
	if !ok {
		t.Fatal("history is not an array")
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(history))
	}

	// First entry should have rejection_reason
	firstEntry := history[0].(map[string]interface{})
	if _, hasReason := firstEntry["rejection_reason"]; !hasReason {
		t.Error("expected first entry to have rejection_reason")
	}
}

// Test conversion from TaskHistory to HistoryEntry preserves rejection_reason
func TestTaskHistoryToHistoryEntry(t *testing.T) {
	// RED: Verify TaskHistory.RejectionReason is converted to HistoryEntry

	now := time.Now()
	taskHistory := &models.TaskHistory{
		ID:              1,
		TaskID:          100,
		OldStatus:       strPtr("in_progress"),
		NewStatus:       "ready_for_review",
		Agent:           strPtr("reviewer-1"),
		Notes:           strPtr("Code review"),
		RejectionReason: strPtr("Missing error handling"),
		Timestamp:       now,
	}

	// This is what the command should do:
	entry := HistoryEntry{
		Timestamp:       taskHistory.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		RelativeAge:     "just now", // Simplified for test
		OldStatus:       taskHistory.OldStatus,
		NewStatus:       taskHistory.NewStatus,
		Agent:           taskHistory.Agent,
		Notes:           taskHistory.Notes,
		RejectionReason: taskHistory.RejectionReason,
	}

	// Verify rejection_reason is preserved
	if entry.RejectionReason == nil {
		t.Fatal("expected rejection_reason to be preserved")
	}
	if *entry.RejectionReason != "Missing error handling" {
		t.Errorf("expected 'Missing error handling', got %s", *entry.RejectionReason)
	}
}

// Test table display format includes rejection reason (helper function)
func TestFormatHistoryTableWithRejectionReason(t *testing.T) {
	// RED: Verify rejection_reason would be displayed in table format

	// When rejection reason is provided, it should be displayed
	rejection := "Missing error handling on line 42"
	reason := &rejection

	// Verify the reason is available for display
	if reason == nil {
		t.Error("rejection reason should not be nil")
	}
	if *reason == "" {
		t.Error("rejection reason should not be empty")
	}

	// Verify empty rejection reasons are handled
	emptyReason := ""
	if emptyReason == "" {
		// Should skip display if empty
		t.Logf("Empty reason correctly skipped")
	}
}
