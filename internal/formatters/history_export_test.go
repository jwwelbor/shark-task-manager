package formatters

import (
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatHistoryCSV tests CSV export of history records
func TestFormatHistoryCSV(t *testing.T) {
	// Create test data
	timestamp1 := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)
	timestamp2 := time.Date(2025, 12, 27, 11, 30, 0, 0, time.UTC)

	oldStatus := "todo"
	agent1 := "backend-agent-1"
	notes1 := "Started implementation"
	notes2 := "Ready for review"

	histories := []HistoryExportRecord{
		{
			Timestamp: timestamp1,
			TaskKey:   "T-E05-F03-001",
			OldStatus: &oldStatus,
			NewStatus: "in_progress",
			Agent:     &agent1,
			Notes:     &notes1,
		},
		{
			Timestamp: timestamp2,
			TaskKey:   "T-E05-F03-001",
			OldStatus: stringPtr("in_progress"),
			NewStatus: "ready_for_review",
			Agent:     &agent1,
			Notes:     &notes2,
		},
	}

	// Test CSV export
	csv, err := FormatHistoryCSV(histories)
	require.NoError(t, err)
	require.NotEmpty(t, csv)

	// Verify CSV header
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	assert.Len(t, lines, 3, "Should have header + 2 data rows")

	// Check header
	expectedHeader := "timestamp,task_key,old_status,new_status,agent,notes"
	assert.Equal(t, expectedHeader, lines[0])

	// Check first data row
	expectedRow1 := "2025-12-27T10:00:00Z,T-E05-F03-001,todo,in_progress,backend-agent-1,Started implementation"
	assert.Equal(t, expectedRow1, lines[1])

	// Check second data row
	expectedRow2 := "2025-12-27T11:30:00Z,T-E05-F03-001,in_progress,ready_for_review,backend-agent-1,Ready for review"
	assert.Equal(t, expectedRow2, lines[2])
}

// TestFormatHistoryCSVWithEmptyFields tests CSV export with empty/nil fields
func TestFormatHistoryCSVWithEmptyFields(t *testing.T) {
	timestamp := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)

	histories := []HistoryExportRecord{
		{
			Timestamp: timestamp,
			TaskKey:   "T-E05-F03-002",
			OldStatus: nil, // Initial creation
			NewStatus: "todo",
			Agent:     nil,
			Notes:     nil,
		},
	}

	csv, err := FormatHistoryCSV(histories)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(csv), "\n")
	assert.Len(t, lines, 2, "Should have header + 1 data row")

	// Check data row with empty fields (only old_status, agent, notes are empty)
	expectedRow := "2025-12-27T10:00:00Z,T-E05-F03-002,,todo,,"
	assert.Equal(t, expectedRow, lines[1])
}

// TestFormatHistoryCSVWithCommasInNotes tests CSV export with commas in notes field
func TestFormatHistoryCSVWithCommasInNotes(t *testing.T) {
	timestamp := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)
	notes := "Fixed bug, added tests, updated docs"

	histories := []HistoryExportRecord{
		{
			Timestamp: timestamp,
			TaskKey:   "T-E05-F03-003",
			OldStatus: stringPtr("todo"),
			NewStatus: "completed",
			Agent:     stringPtr("dev-agent"),
			Notes:     &notes,
		},
	}

	csv, err := FormatHistoryCSV(histories)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(csv), "\n")
	assert.Len(t, lines, 2)

	// Notes should be quoted because they contain commas
	expectedRow := `2025-12-27T10:00:00Z,T-E05-F03-003,todo,completed,dev-agent,"Fixed bug, added tests, updated docs"`
	assert.Equal(t, expectedRow, lines[1])
}

// TestFormatHistoryCSVWithQuotesInNotes tests CSV export with quotes in notes field
func TestFormatHistoryCSVWithQuotesInNotes(t *testing.T) {
	timestamp := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)
	notes := `Updated "main" function`

	histories := []HistoryExportRecord{
		{
			Timestamp: timestamp,
			TaskKey:   "T-E05-F03-004",
			OldStatus: stringPtr("todo"),
			NewStatus: "completed",
			Agent:     stringPtr("dev-agent"),
			Notes:     &notes,
		},
	}

	csv, err := FormatHistoryCSV(histories)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(csv), "\n")
	assert.Len(t, lines, 2)

	// Quotes should be escaped as double quotes
	expectedRow := `2025-12-27T10:00:00Z,T-E05-F03-004,todo,completed,dev-agent,"Updated ""main"" function"`
	assert.Equal(t, expectedRow, lines[1])
}

// TestFormatHistoryCSVEmpty tests CSV export with no records
func TestFormatHistoryCSVEmpty(t *testing.T) {
	histories := []HistoryExportRecord{}

	csv, err := FormatHistoryCSV(histories)
	require.NoError(t, err)

	// Should still have header
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	assert.Len(t, lines, 1)
	assert.Equal(t, "timestamp,task_key,old_status,new_status,agent,notes", lines[0])
}

// TestFormatHistoryJSON tests JSON export of history records
func TestFormatHistoryJSON(t *testing.T) {
	timestamp1 := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)
	timestamp2 := time.Date(2025, 12, 27, 11, 30, 0, 0, time.UTC)

	agent1 := "backend-agent-1"
	notes1 := "Started implementation"

	histories := []HistoryExportRecord{
		{
			Timestamp: timestamp1,
			TaskKey:   "T-E05-F03-001",
			OldStatus: stringPtr("todo"),
			NewStatus: "in_progress",
			Agent:     &agent1,
			Notes:     &notes1,
		},
		{
			Timestamp: timestamp2,
			TaskKey:   "T-E05-F03-001",
			OldStatus: stringPtr("in_progress"),
			NewStatus: "ready_for_review",
			Agent:     &agent1,
			Notes:     stringPtr("Ready for review"),
		},
	}

	json, err := FormatHistoryJSON(histories)
	require.NoError(t, err)
	require.NotEmpty(t, json)

	// Verify it's valid JSON and contains expected fields
	assert.Contains(t, json, `"timestamp"`)
	assert.Contains(t, json, `"task_key"`)
	assert.Contains(t, json, `"old_status"`)
	assert.Contains(t, json, `"new_status"`)
	assert.Contains(t, json, `"agent"`)
	assert.Contains(t, json, `"notes"`)
	assert.Contains(t, json, `"T-E05-F03-001"`)
	assert.Contains(t, json, `"in_progress"`)
	assert.Contains(t, json, `"backend-agent-1"`)
	assert.Contains(t, json, `"Started implementation"`)
}

// TestFormatHistoryJSONEmpty tests JSON export with no records
func TestFormatHistoryJSONEmpty(t *testing.T) {
	histories := []HistoryExportRecord{}

	json, err := FormatHistoryJSON(histories)
	require.NoError(t, err)

	// Should return empty array
	assert.Equal(t, "[]", strings.TrimSpace(json))
}

// TestConvertToExportRecords tests conversion from TaskHistory to HistoryExportRecord
func TestConvertToExportRecords(t *testing.T) {
	timestamp := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)

	taskHistories := []*models.TaskHistory{
		{
			ID:        1,
			TaskID:    100,
			OldStatus: stringPtr("todo"),
			NewStatus: "in_progress",
			Agent:     stringPtr("backend-agent"),
			Notes:     stringPtr("Starting work"),
			Timestamp: timestamp,
		},
	}

	taskKey := "T-E05-F03-001"
	records := ConvertToExportRecords(taskHistories, taskKey)

	require.Len(t, records, 1)
	assert.Equal(t, timestamp, records[0].Timestamp)
	assert.Equal(t, taskKey, records[0].TaskKey)
	assert.Equal(t, "todo", *records[0].OldStatus)
	assert.Equal(t, "in_progress", records[0].NewStatus)
	assert.Equal(t, "backend-agent", *records[0].Agent)
	assert.Equal(t, "Starting work", *records[0].Notes)
}

// TestConvertMultipleTasksToExportRecords tests conversion of multiple tasks' history
func TestConvertMultipleTasksToExportRecords(t *testing.T) {
	timestamp1 := time.Date(2025, 12, 27, 10, 0, 0, 0, time.UTC)
	timestamp2 := time.Date(2025, 12, 27, 11, 0, 0, 0, time.UTC)

	historyWithTasks := []HistoryWithTask{
		{
			History: &models.TaskHistory{
				ID:        1,
				TaskID:    100,
				OldStatus: stringPtr("todo"),
				NewStatus: "in_progress",
				Timestamp: timestamp1,
			},
			TaskKey: "T-E05-F03-001",
		},
		{
			History: &models.TaskHistory{
				ID:        2,
				TaskID:    101,
				OldStatus: nil,
				NewStatus: "todo",
				Timestamp: timestamp2,
			},
			TaskKey: "T-E05-F03-002",
		},
	}

	records := ConvertMultipleTasksToExportRecords(historyWithTasks)

	require.Len(t, records, 2)
	assert.Equal(t, "T-E05-F03-001", records[0].TaskKey)
	assert.Equal(t, "T-E05-F03-002", records[1].TaskKey)
	assert.Equal(t, "in_progress", records[0].NewStatus)
	assert.Equal(t, "todo", records[1].NewStatus)
}
