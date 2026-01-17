package formatters

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// HistoryExportRecord represents a single history record for export
type HistoryExportRecord struct {
	Timestamp       time.Time `json:"timestamp"`
	TaskKey         string    `json:"task_key"`
	OldStatus       *string   `json:"old_status,omitempty"`
	NewStatus       string    `json:"new_status"`
	Agent           *string   `json:"agent,omitempty"`
	Notes           *string   `json:"notes,omitempty"`
	RejectionReason *string   `json:"rejection_reason,omitempty"`
}

// HistoryWithTask combines a history record with its associated task key
type HistoryWithTask struct {
	History *models.TaskHistory
	TaskKey string
}

// FormatHistoryCSV formats history records as CSV
func FormatHistoryCSV(records []HistoryExportRecord) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	// Write header
	header := []string{"timestamp", "task_key", "old_status", "new_status", "agent", "rejection_reason", "notes"}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, record := range records {
		row := []string{
			record.Timestamp.Format(time.RFC3339),
			record.TaskKey,
			stringValue(record.OldStatus),
			record.NewStatus,
			stringValue(record.Agent),
			stringValue(record.RejectionReason),
			stringValue(record.Notes),
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	return builder.String(), nil
}

// FormatHistoryJSON formats history records as JSON
func FormatHistoryJSON(records []HistoryExportRecord) (string, error) {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// ConvertToExportRecords converts TaskHistory records to HistoryExportRecord for a single task
func ConvertToExportRecords(histories []*models.TaskHistory, taskKey string) []HistoryExportRecord {
	records := make([]HistoryExportRecord, len(histories))
	for i, h := range histories {
		records[i] = HistoryExportRecord{
			Timestamp:       h.Timestamp,
			TaskKey:         taskKey,
			OldStatus:       h.OldStatus,
			NewStatus:       h.NewStatus,
			Agent:           h.Agent,
			Notes:           h.Notes,
			RejectionReason: h.RejectionReason,
		}
	}
	return records
}

// ConvertMultipleTasksToExportRecords converts history records from multiple tasks
func ConvertMultipleTasksToExportRecords(historyWithTasks []HistoryWithTask) []HistoryExportRecord {
	records := make([]HistoryExportRecord, len(historyWithTasks))
	for i, ht := range historyWithTasks {
		records[i] = HistoryExportRecord{
			Timestamp:       ht.History.Timestamp,
			TaskKey:         ht.TaskKey,
			OldStatus:       ht.History.OldStatus,
			NewStatus:       ht.History.NewStatus,
			Agent:           ht.History.Agent,
			Notes:           ht.History.Notes,
			RejectionReason: ht.History.RejectionReason,
		}
	}
	return records
}

// stringValue safely extracts the value from a string pointer, returning empty string if nil
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
