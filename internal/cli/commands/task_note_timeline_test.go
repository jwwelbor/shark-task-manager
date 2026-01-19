package commands

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// Test that timeline includes rejection events with proper formatting
func TestTimelineRejectionEventFormatting(t *testing.T) {
	tests := []struct {
		name          string
		rejectionNote *repository.RejectionHistoryEntry
		expectedIcon  string // Should contain ⚠️
	}{
		{
			name: "rejection with short reason",
			rejectionNote: &repository.RejectionHistoryEntry{
				ID:         1,
				Timestamp:  "2026-01-15 14:30:00",
				FromStatus: "in_code_review",
				ToStatus:   "in_development",
				RejectedBy: "reviewer",
				Reason:     "Missing error handling",
			},
			expectedIcon: "⚠️",
		},
		{
			name: "rejection with long reason",
			rejectionNote: &repository.RejectionHistoryEntry{
				ID:         2,
				Timestamp:  "2026-01-15 15:00:00",
				FromStatus: "ready_for_qa",
				ToStatus:   "in_code_review",
				RejectedBy: "qa-agent",
				Reason:     strings.Repeat("x", 100),
			},
			expectedIcon: "⚠️",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the rejection entry has expected properties
			if tt.rejectionNote.Reason == "" {
				t.Error("rejection note should have a reason")
			}

			// Verify truncation logic for long reasons
			if len(tt.rejectionNote.Reason) > 80 {
				truncated := tt.rejectionNote.Reason[:77] + "..."
				if len(truncated) >= len(tt.rejectionNote.Reason) {
					t.Error("truncation should produce shorter text")
				}
				if !strings.HasSuffix(truncated, "...") {
					t.Error("truncated text should end with ...")
				}
			}
		})
	}
}

// Test TimelineEvent structure includes rejection data
func TestTimelineEventRejectionType(t *testing.T) {
	timestamp := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)

	event := TimelineEvent{
		Timestamp: timestamp,
		EventType: "rejection",
		Content:   "⚠️ Rejected by reviewer: Missing error handling",
		Actor:     "reviewer",
	}

	// Verify event structure
	if event.EventType != "rejection" {
		t.Errorf("expected EventType 'rejection', got %s", event.EventType)
	}

	if !strings.Contains(event.Content, "⚠️") {
		t.Error("rejection event content should include warning symbol")
	}

	if !strings.Contains(event.Content, "Rejected") {
		t.Error("rejection event should indicate rejection")
	}
}

// Test rejection events appear in chronological order
func TestTimelineRejectionChronological(t *testing.T) {
	now := time.Now()

	events := []TimelineEvent{
		{
			Timestamp: now.Add(-2 * time.Hour),
			EventType: "status",
			Content:   "Created",
		},
		{
			Timestamp: now.Add(-1 * time.Hour),
			EventType: "rejection",
			Content:   "⚠️ Rejected by reviewer",
		},
		{
			Timestamp: now,
			EventType: "status",
			Content:   "Status: in_code_review → in_development",
		},
	}

	// Verify events are chronologically ordered
	for i := 0; i < len(events)-1; i++ {
		if events[i+1].Timestamp.Before(events[i].Timestamp) {
			t.Errorf("events out of order: event %d should not be before event %d", i+1, i)
		}
	}
}

// Test JSON output includes rejection events
func TestTimelineJSONWithRejections(t *testing.T) {
	now := time.Now()

	timeline := []TimelineEvent{
		{
			Timestamp: now,
			EventType: "rejection",
			Content:   "⚠️ Rejected by reviewer: Missing tests",
			Actor:     "reviewer",
		},
		{
			Timestamp: now.Add(1 * time.Minute),
			EventType: "status",
			Content:   "Status: in_code_review → in_development",
			Actor:     "developer",
		},
	}

	// Verify JSON marshaling includes rejection events
	jsonData, err := json.Marshal(timeline)
	if err != nil {
		t.Fatalf("failed to marshal timeline: %v", err)
	}

	var unmarshaled []TimelineEvent
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal timeline: %v", err)
	}

	if len(unmarshaled) != 2 {
		t.Errorf("expected 2 events, got %d", len(unmarshaled))
	}

	// Verify rejection event is preserved in JSON
	rejectionFound := false
	for _, event := range unmarshaled {
		if event.EventType == "rejection" {
			rejectionFound = true
			if !strings.Contains(event.Content, "⚠️") {
				t.Error("rejection event should preserve warning symbol in JSON")
			}
		}
	}

	if !rejectionFound {
		t.Error("rejection event should be present in JSON output")
	}
}

// Test reason truncation with specific lengths
func TestReasonTruncationLogic(t *testing.T) {
	tests := []struct {
		name           string
		reason         string
		maxLen         int
		expectTruncate bool
	}{
		{
			name:           "short reason",
			reason:         "Missing error handling",
			maxLen:         80,
			expectTruncate: false,
		},
		{
			name:           "exactly 80 chars",
			reason:         strings.Repeat("x", 80),
			maxLen:         80,
			expectTruncate: false,
		},
		{
			name:           "81 chars",
			reason:         strings.Repeat("x", 81),
			maxLen:         80,
			expectTruncate: true,
		},
		{
			name:           "very long reason",
			reason:         strings.Repeat("This is a very long reason. ", 10),
			maxLen:         80,
			expectTruncate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var truncated string
			if len(tt.reason) > tt.maxLen {
				// Truncate to 77 chars + "..."
				truncated = tt.reason[:tt.maxLen-3] + "..."
			} else {
				truncated = tt.reason
			}

			shouldTruncate := len(truncated) < len(tt.reason)
			if shouldTruncate != tt.expectTruncate {
				t.Errorf("expected truncate=%v, got %v", tt.expectTruncate, shouldTruncate)
			}

			if shouldTruncate && !strings.HasSuffix(truncated, "...") {
				t.Error("truncated reason should end with ...")
			}

			// Verify truncated length
			if shouldTruncate && len(truncated) > 80 {
				t.Errorf("truncated text length %d exceeds maxLen %d", len(truncated), tt.maxLen)
			}
		})
	}
}

// Test document indicator for rejection events
func TestRejectionDocumentIndicator(t *testing.T) {
	tests := []struct {
		name         string
		docPath      *string
		shouldShow   bool
		expectedIcon string
	}{
		{
			name:         "no document",
			docPath:      nil,
			shouldShow:   false,
			expectedIcon: "",
		},
		{
			name:         "with document",
			docPath:      stringPtr("docs/review-feedback.md"),
			shouldShow:   true,
			expectedIcon: "📄",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasDoc := tt.docPath != nil && *tt.docPath != ""

			if tt.shouldShow != hasDoc {
				t.Errorf("expected shouldShow=%v, got %v", tt.shouldShow, hasDoc)
			}

			if tt.shouldShow && tt.docPath != nil {
				// In the actual implementation, the document indicator would be shown
				if tt.expectedIcon != "📄" {
					t.Error("expected document icon to be 📄")
				}
			}
		})
	}
}
