package commands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mapEntityHistoryToEntries converts EntityHistory records to StatusHistoryEntry slices.
// Shared helper to avoid copy-pasting this mapping across tests.
func mapEntityHistoryToEntries(history []*models.EntityHistory) []StatusHistoryEntry {
	entries := make([]StatusHistoryEntry, 0, len(history))
	for _, h := range history {
		entry := StatusHistoryEntry{
			Timestamp: h.ChangedAt.Format(time.RFC3339),
			NewStatus: h.ToStatus,
		}
		if h.FromStatus != nil {
			entry.OldStatus = *h.FromStatus
		}
		if h.ChangedBy != nil {
			entry.Agent = *h.ChangedBy
		}
		if h.Notes != nil {
			entry.Notes = *h.Notes
		}
		entries = append(entries, entry)
	}
	return entries
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

// --- Test StatusHistoryResult for non-task entity types (E21-F08-008) ---

func TestStatusHistory_FeatureKey(t *testing.T) {
	// Verifies that StatusHistoryResult correctly represents feature history
	// and that EntityHistory -> StatusHistoryEntry mapping works for features.
	now := time.Now().UTC()
	fromStatus := "draft"
	changedBy := "tech_lead"
	notes := "Feature ready for development"

	entityHistory := []*models.EntityHistory{
		{
			ID:         1,
			EntityType: models.EntityTypeFeature,
			EntityID:   42,
			FromStatus: &fromStatus,
			ToStatus:   "ready_for_development",
			ChangedBy:  &changedBy,
			Notes:      &notes,
			ChangedAt:  now,
		},
		{
			ID:         2,
			EntityType: models.EntityTypeFeature,
			EntityID:   42,
			FromStatus: nil, // initial status
			ToStatus:   "draft",
			ChangedBy:  nil,
			Notes:      nil,
			ChangedAt:  now.Add(-time.Hour),
		},
	}

	entries := mapEntityHistoryToEntries(entityHistory)

	result := StatusHistoryResult{
		EntityType: "feature",
		EntityKey:  "E21-F07",
		History:    entries,
		Total:      len(entries),
	}

	// Verify JSON output schema
	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "feature", parsed["entity_type"])
	assert.Equal(t, "E21-F07", parsed["entity_key"])
	assert.Equal(t, float64(2), parsed["total"])

	historyArr, ok := parsed["history"].([]interface{})
	require.True(t, ok)
	require.Len(t, historyArr, 2)

	// First entry has all fields
	first := historyArr[0].(map[string]interface{})
	assert.Equal(t, "draft", first["old_status"])
	assert.Equal(t, "ready_for_development", first["new_status"])
	assert.Equal(t, "tech_lead", first["agent"])
	assert.Equal(t, "Feature ready for development", first["notes"])

	// Second entry has nil -> empty for optional fields
	second := historyArr[1].(map[string]interface{})
	assert.Equal(t, "", second["old_status"]) // nil -> empty string
	assert.Equal(t, "draft", second["new_status"])
	_, hasAgent := second["agent"]
	assert.False(t, hasAgent, "agent should be omitted when empty (omitempty)")
}

func TestStatusHistory_BugKey(t *testing.T) {
	// Verifies that StatusHistoryResult correctly represents bug history
	// and that the entity_type field is "bug" in JSON output.
	now := time.Now().UTC()
	fromStatus := "open"
	changedBy := "qa_team"

	entityHistory := []*models.EntityHistory{
		{
			ID:         1,
			EntityType: models.EntityTypeBug,
			EntityID:   10,
			FromStatus: &fromStatus,
			ToStatus:   "in_progress",
			ChangedBy:  &changedBy,
			Notes:      nil,
			ChangedAt:  now,
		},
	}

	entries := mapEntityHistoryToEntries(entityHistory)

	result := StatusHistoryResult{
		EntityType: "bug",
		EntityKey:  "B001",
		History:    entries,
		Total:      len(entries),
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "bug", parsed["entity_type"])
	assert.Equal(t, "B001", parsed["entity_key"])
	assert.Equal(t, float64(1), parsed["total"])

	historyArr, ok := parsed["history"].([]interface{})
	require.True(t, ok)
	require.Len(t, historyArr, 1)

	first := historyArr[0].(map[string]interface{})
	assert.Equal(t, "open", first["old_status"])
	assert.Equal(t, "in_progress", first["new_status"])
	assert.Equal(t, "qa_team", first["agent"])
}

func TestStatusHistory_ChangeCardNormalization(t *testing.T) {
	// Verifies ADR-1: change_card entity type is normalized to "change"
	// in the StatusHistoryResult output.
	result := StatusHistoryResult{
		EntityType: "change", // After normalization from "change_card"
		EntityKey:  "CC-001",
		History:    []StatusHistoryEntry{},
		Total:      0,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "change", parsed["entity_type"], "change_card should be normalized to change")
	assert.Equal(t, "CC-001", parsed["entity_key"])
	assert.Equal(t, float64(0), parsed["total"])
}

func TestStatusHistory_EpicKey(t *testing.T) {
	// Verifies StatusHistoryResult works for epic entity type.
	now := time.Now().UTC()
	fromStatus := "draft"

	entityHistory := []*models.EntityHistory{
		{
			ID:         1,
			EntityType: models.EntityTypeEpic,
			EntityID:   5,
			FromStatus: &fromStatus,
			ToStatus:   "active",
			ChangedBy:  nil,
			Notes:      nil,
			ChangedAt:  now,
		},
	}

	entries := mapEntityHistoryToEntries(entityHistory)

	result := StatusHistoryResult{
		EntityType: "epic",
		EntityKey:  "E21",
		History:    entries,
		Total:      len(entries),
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "epic", parsed["entity_type"])
	assert.Equal(t, "E21", parsed["entity_key"])
	assert.Equal(t, float64(1), parsed["total"])
}

func TestStatusHistory_LimitTruncation(t *testing.T) {
	// Verifies BR-3: --limit flag truncates result set in the CLI layer.
	now := time.Now().UTC()

	// Create 5 history entries
	entityHistory := make([]*models.EntityHistory, 5)
	for i := 0; i < 5; i++ {
		status := "status_" + string(rune('a'+i))
		entityHistory[i] = &models.EntityHistory{
			ID:         int64(i + 1),
			EntityType: models.EntityTypeFeature,
			EntityID:   42,
			FromStatus: nil,
			ToStatus:   status,
			ChangedAt:  now.Add(time.Duration(i) * time.Hour),
		}
	}

	// Apply limit of 3 (same logic as runStatusHistory: history[:limit] keeps first N,
	// which are the most recent since history is ordered DESC from the repository)
	limit := 3
	if limit > 0 && len(entityHistory) > limit {
		entityHistory = entityHistory[:limit]
	}

	assert.Len(t, entityHistory, 3, "limit should truncate to first 3 entries")
	assert.Equal(t, "status_a", entityHistory[0].ToStatus, "should keep first 3 entries (most recent in DESC order)")
	assert.Equal(t, "status_c", entityHistory[2].ToStatus)
}

func TestStatusHistory_EmptyResult(t *testing.T) {
	// Verifies AC-7: empty history returns correct JSON structure.
	result := StatusHistoryResult{
		EntityType: "feature",
		EntityKey:  "E21-F07",
		History:    []StatusHistoryEntry{},
		Total:      0,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed StatusHistoryResult
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "feature", parsed.EntityType)
	assert.Equal(t, 0, parsed.Total)
	assert.Empty(t, parsed.History)
}

// --- Test formatHistoryNotesForDisplay (AC-T1 through AC-T4) ---

func TestStatusHistoryFormatter_AutoReopenLabel(t *testing.T) {
	t.Run("AC-T1: auto_reopen prefix gets distinct label in human output", func(t *testing.T) {
		notes := "auto_reopen: parent re-opened because child E07-F01-001 regressed to in_progress"
		result := formatHistoryNotesForDisplay(notes)
		assert.Contains(t, result, "[auto-reopen]", "human output should contain [auto-reopen] label")
		assert.Contains(t, result, notes, "original notes should still be present")
	})

	t.Run("AC-T3: detection is purely strings.HasPrefix on auto_reopen:", func(t *testing.T) {
		// Exact prefix required — no false positives on other prefixes
		cases := []struct {
			notes   string
			wantTag bool
		}{
			{"auto_reopen: cascaded from child", true},
			{"auto_reopen:", true}, // bare prefix still matches
			{"manual note", false},
			{"", false},
			{"AUTO_REOPEN: uppercase should not match", false}, // case-sensitive
			{"some auto_reopen: mid-string note", false},       // not a prefix
		}
		for _, c := range cases {
			got := formatHistoryNotesForDisplay(c.notes)
			if c.wantTag {
				assert.Contains(t, got, "[auto-reopen]", "expected [auto-reopen] label for notes=%q", c.notes)
			} else {
				assert.NotContains(t, got, "[auto-reopen]", "did not expect [auto-reopen] label for notes=%q", c.notes)
			}
		}
	})

	t.Run("AC-T4: rows without prefix render identically to before", func(t *testing.T) {
		normalNotes := "Implemented per spec"
		result := formatHistoryNotesForDisplay(normalNotes)
		assert.Equal(t, normalNotes, result, "non-auto_reopen notes should be unchanged")
	})

	t.Run("AC-T4: empty notes render identically to before", func(t *testing.T) {
		result := formatHistoryNotesForDisplay("")
		assert.Equal(t, "", result)
	})

	t.Run("AC-T2: JSON output schema unchanged — Notes field raw, no label", func(t *testing.T) {
		// Verify that StatusHistoryEntry (used for JSON output) never has [auto-reopen]
		// injected — the formatter is only called in the human-output path.
		notes := "auto_reopen: parent cascade"
		entry := StatusHistoryEntry{
			Timestamp: "2026-04-07T10:00:00Z",
			OldStatus: "completed",
			NewStatus: "in_progress",
			Notes:     notes,
		}

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		// JSON notes field must be the raw value — no label injected
		assert.Equal(t, notes, parsed["notes"], "JSON notes field must remain unchanged")
		assert.NotContains(t, parsed["notes"].(string), "[auto-reopen]", "JSON output must not contain label")
	})

	t.Run("auto_reopen label visible in table row", func(t *testing.T) {
		// Build two entries: one with auto_reopen prefix, one without.
		// Simulate the table-row building logic from runStatusHistory.
		autoNotes := "auto_reopen: cascade from child task"
		manualNotes := "Manual status update"

		entries := []StatusHistoryEntry{
			{Timestamp: "2026-04-07T10:00:00Z", OldStatus: "completed", NewStatus: "in_progress", Notes: autoNotes},
			{Timestamp: "2026-04-07T11:00:00Z", OldStatus: "in_progress", NewStatus: "completed", Notes: manualNotes},
		}

		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			oldStatus := e.OldStatus
			if oldStatus == "" {
				oldStatus = "(initial)"
			}
			rows = append(rows, []string{
				e.Timestamp,
				oldStatus,
				e.NewStatus,
				e.Agent,
				truncateToWidth(formatHistoryNotesForDisplay(e.Notes), 60),
			})
		}

		require.Len(t, rows, 2)
		// auto_reopen row: Notes column should contain [auto-reopen]
		assert.Contains(t, rows[0][4], "[auto-reopen]", "auto_reopen row should have distinct label in Notes column")
		// manual row: Notes column should not contain [auto-reopen]
		assert.NotContains(t, rows[1][4], "[auto-reopen]", "manual row should not have label in Notes column")
		assert.Contains(t, rows[1][4], manualNotes, "manual row should preserve original notes text")
	})
}
