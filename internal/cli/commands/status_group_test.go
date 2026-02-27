package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test capitalizeEntityType ---

func TestCapitalizeEntityType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "epic", input: "epic", expected: "Epic"},
		{name: "feature", input: "feature", expected: "Feature"},
		{name: "task", input: "task", expected: "Task"},
		{name: "empty string", input: "", expected: ""},
		{name: "single char", input: "a", expected: "A"},
		{name: "already capitalized", input: "Epic", expected: "Epic"},
		{name: "all uppercase", input: "TASK", expected: "TASK"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capitalizeEntityType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Test truncateString ---

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string no truncation",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length string",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated with ellipsis",
			input:    "this is a very long string",
			maxLen:   10,
			expected: "this is...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "maxLen equals 3",
			input:    "abcdef",
			maxLen:   3,
			expected: "abc",
		},
		{
			name:     "maxLen less than 3",
			input:    "abcdef",
			maxLen:   2,
			expected: "ab",
		},
		{
			name:     "maxLen of 1",
			input:    "abcdef",
			maxLen:   1,
			expected: "a",
		},
		{
			name:     "maxLen of 4 with long string",
			input:    "abcdef",
			maxLen:   4,
			expected: "a...",
		},
		{
			name:     "maxLen of 0",
			input:    "abc",
			maxLen:   0,
			expected: "", // len("abc") > 0, maxLen <= 3, so s[:0] = ""
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Test entityTransitionerFunc adapter ---

func TestEntityTransitionerFunc(t *testing.T) {
	t.Run("delegates to wrapped function", func(t *testing.T) {
		called := false
		f := entityTransitionerFunc(func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			called = true
			assert.Equal(t, "E07-F01-001", key)
			assert.Equal(t, "in_progress", targetStatus)
			assert.True(t, opts.Force)
			assert.Equal(t, "testing reason", opts.Reason)
			return &services.TransitionResult{
				FromStatus: "todo",
				ToStatus:   "in_progress",
				EntityKey:  "E07-F01-001",
			}, nil
		})

		result, err := f.TransitionStatus(context.Background(), "E07-F01-001", "in_progress", services.TransitionOptions{
			Force:  true,
			Reason: "testing reason",
		})

		assert.NoError(t, err)
		assert.True(t, called)
		require.NotNil(t, result)
		assert.Equal(t, "todo", result.FromStatus)
		assert.Equal(t, "in_progress", result.ToStatus)
		assert.Equal(t, "E07-F01-001", result.EntityKey)
	})

	t.Run("propagates errors", func(t *testing.T) {
		f := entityTransitionerFunc(func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return nil, assert.AnError
		})

		result, err := f.TransitionStatus(context.Background(), "E07-F01-001", "in_progress", services.TransitionOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// --- Test StatusSetResult JSON serialization ---

func TestStatusSetResult_JSON(t *testing.T) {
	t.Run("changed true with all fields", func(t *testing.T) {
		result := StatusSetResult{
			EntityType:     "task",
			EntityKey:      "E07-F01-001",
			Status:         "in_progress",
			Changed:        true,
			Message:        "todo -> in_progress",
			PreviousStatus: "todo",
			IsBackward:     false,
			IsForced:       true,
			Reason:         "admin override",
			ChildCount:     3,
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "task", parsed["entity_type"])
		assert.Equal(t, "E07-F01-001", parsed["entity_key"])
		assert.Equal(t, "in_progress", parsed["status"])
		assert.Equal(t, true, parsed["changed"])
		assert.Equal(t, "todo -> in_progress", parsed["message"])
		assert.Equal(t, "todo", parsed["previous_status"])
		assert.Equal(t, true, parsed["is_forced"])
		assert.Equal(t, "admin override", parsed["reason"])
		assert.Equal(t, float64(3), parsed["child_count"])
	})

	t.Run("changed false omits optional fields", func(t *testing.T) {
		result := StatusSetResult{
			EntityType: "epic",
			EntityKey:  "E07",
			Status:     "active",
			Changed:    false,
			Message:    "Entity already at status 'active'",
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "epic", parsed["entity_type"])
		assert.Equal(t, "E07", parsed["entity_key"])
		assert.Equal(t, "active", parsed["status"])
		assert.Equal(t, false, parsed["changed"])
		assert.Equal(t, "Entity already at status 'active'", parsed["message"])

		// Optional fields should be omitted when zero/empty
		_, hasPrevious := parsed["previous_status"]
		assert.False(t, hasPrevious, "previous_status should be omitted when empty")
		_, hasReason := parsed["reason"]
		assert.False(t, hasReason, "reason should be omitted when empty")
		_, hasIsBackward := parsed["is_backward"]
		assert.False(t, hasIsBackward, "is_backward should be omitted when false")
		_, hasIsForced := parsed["is_forced"]
		assert.False(t, hasIsForced, "is_forced should be omitted when false")
		_, hasChildCount := parsed["child_count"]
		assert.False(t, hasChildCount, "child_count should be omitted when zero")
		_, hasAction := parsed["orchestrator_action"]
		assert.False(t, hasAction, "orchestrator_action should be omitted when nil")
	})

	t.Run("roundtrip deserialization", func(t *testing.T) {
		original := StatusSetResult{
			EntityType:     "feature",
			EntityKey:      "E07-F01",
			Status:         "active",
			Changed:        true,
			Message:        "draft -> active",
			PreviousStatus: "draft",
			IsBackward:     true,
			Reason:         "rework needed",
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var deserialized StatusSetResult
		err = json.Unmarshal(data, &deserialized)
		require.NoError(t, err)

		assert.Equal(t, original.EntityType, deserialized.EntityType)
		assert.Equal(t, original.EntityKey, deserialized.EntityKey)
		assert.Equal(t, original.Status, deserialized.Status)
		assert.Equal(t, original.Changed, deserialized.Changed)
		assert.Equal(t, original.Message, deserialized.Message)
		assert.Equal(t, original.PreviousStatus, deserialized.PreviousStatus)
		assert.Equal(t, original.IsBackward, deserialized.IsBackward)
		assert.Equal(t, original.Reason, deserialized.Reason)
	})
}

// --- Test StatusHistoryResult JSON serialization ---

func TestStatusHistoryResult_JSON(t *testing.T) {
	t.Run("with entries", func(t *testing.T) {
		result := StatusHistoryResult{
			EntityType: "task",
			EntityKey:  "E07-F01-001",
			History: []StatusHistoryEntry{
				{
					Timestamp: "2026-02-25T10:00:00Z",
					OldStatus: "todo",
					NewStatus: "in_progress",
					Agent:     "developer",
					Notes:     "Starting work",
				},
				{
					Timestamp: "2026-02-25T12:00:00Z",
					OldStatus: "in_progress",
					NewStatus: "ready_for_review",
					Notes:     "Implementation done",
				},
			},
			Total: 2,
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed StatusHistoryResult
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "task", parsed.EntityType)
		assert.Equal(t, "E07-F01-001", parsed.EntityKey)
		assert.Equal(t, 2, parsed.Total)
		require.Len(t, parsed.History, 2)
		assert.Equal(t, "todo", parsed.History[0].OldStatus)
		assert.Equal(t, "in_progress", parsed.History[0].NewStatus)
		assert.Equal(t, "developer", parsed.History[0].Agent)
	})

	t.Run("with empty history", func(t *testing.T) {
		result := StatusHistoryResult{
			EntityType: "task",
			EntityKey:  "E07-F01-001",
			History:    []StatusHistoryEntry{},
			Total:      0,
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed StatusHistoryResult
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, 0, parsed.Total)
		assert.Empty(t, parsed.History)
	})

	t.Run("nil history serializes as null", func(t *testing.T) {
		result := StatusHistoryResult{
			EntityType: "task",
			EntityKey:  "E07-F01-001",
			Total:      0,
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		// nil slice serializes as null in JSON
		assert.Nil(t, parsed["history"])
	})
}

// --- Test StatusHistoryEntry JSON serialization ---

func TestStatusHistoryEntry_JSON(t *testing.T) {
	t.Run("all fields populated", func(t *testing.T) {
		entry := StatusHistoryEntry{
			Timestamp: "2026-02-25T10:00:00Z",
			OldStatus: "todo",
			NewStatus: "in_progress",
			Agent:     "developer",
			Notes:     "Starting implementation",
		}

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "2026-02-25T10:00:00Z", parsed["timestamp"])
		assert.Equal(t, "todo", parsed["old_status"])
		assert.Equal(t, "in_progress", parsed["new_status"])
		assert.Equal(t, "developer", parsed["agent"])
		assert.Equal(t, "Starting implementation", parsed["notes"])
	})

	t.Run("optional fields omitted when empty", func(t *testing.T) {
		entry := StatusHistoryEntry{
			Timestamp: "2026-02-25T10:00:00Z",
			OldStatus: "",
			NewStatus: "todo",
		}

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "2026-02-25T10:00:00Z", parsed["timestamp"])
		assert.Equal(t, "todo", parsed["new_status"])

		// Agent and Notes have omitempty, so they should be absent
		_, hasAgent := parsed["agent"]
		assert.False(t, hasAgent, "agent should be omitted when empty")
		_, hasNotes := parsed["notes"]
		assert.False(t, hasNotes, "notes should be omitted when empty")

		// old_status does NOT have omitempty, so it should be present even if empty
		_, hasOldStatus := parsed["old_status"]
		assert.True(t, hasOldStatus, "old_status should be present even when empty (no omitempty)")
	})

	t.Run("roundtrip deserialization", func(t *testing.T) {
		original := StatusHistoryEntry{
			Timestamp: "2026-02-25T14:30:00Z",
			OldStatus: "in_progress",
			NewStatus: "blocked",
			Agent:     "qa",
			Notes:     "Waiting on API endpoint",
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var deserialized StatusHistoryEntry
		err = json.Unmarshal(data, &deserialized)
		require.NoError(t, err)

		assert.Equal(t, original, deserialized)
	})
}
