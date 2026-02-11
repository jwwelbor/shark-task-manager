package config

import (
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestTaskPlaceholders_AllFields validates all keys from a populated Task
func TestTaskPlaceholders_AllFields(t *testing.T) {
	slug := "test-task"
	description := "Test description"
	agentType := "developer"
	filePath := "docs/task.md"
	executionOrder := 5
	blockedReason := "Waiting on API"
	dependsOn := `["T-E01-F01-001"]`
	completionNotes := "Done"
	filesChanged := `["file1.go","file2.go"]`

	task := &models.Task{
		Key:             "T-E07-F01-001",
		Title:           "Test Task",
		Slug:            &slug,
		Description:     &description,
		Status:          "todo",
		AgentType:       &agentType,
		Priority:        5,
		FilePath:        &filePath,
		ExecutionOrder:  &executionOrder,
		BlockedReason:   &blockedReason,
		DependsOn:       &dependsOn,
		CompletionNotes: &completionNotes,
		FilesChanged:    &filesChanged,
		CreatedAt:       time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2025, 1, 16, 14, 45, 0, 0, time.UTC),
	}

	m := TaskPlaceholders(task)

	// Verify required fields
	if m["id"] != "T-E07-F01-001" {
		t.Errorf("id = %q, want %q", m["id"], "T-E07-F01-001")
	}
	if m["task_id"] != "T-E07-F01-001" {
		t.Errorf("task_id = %q, want %q", m["task_id"], "T-E07-F01-001")
	}
	if m["epic_id"] != "T-E07-F01-001" {
		t.Errorf("epic_id = %q, want %q", m["epic_id"], "T-E07-F01-001")
	}
	if m["feature_id"] != "T-E07-F01-001" {
		t.Errorf("feature_id = %q, want %q", m["feature_id"], "T-E07-F01-001")
	}
	if m["title"] != "Test Task" {
		t.Errorf("title = %q, want %q", m["title"], "Test Task")
	}
	if m["status"] != "todo" {
		t.Errorf("status = %q, want %q", m["status"], "todo")
	}
	if m["priority"] != "5" {
		t.Errorf("priority = %q, want %q", m["priority"], "5")
	}
	if m["created_at"] != "2025-01-15T10:30:00Z" {
		t.Errorf("created_at = %q, want %q", m["created_at"], "2025-01-15T10:30:00Z")
	}
	if m["updated_at"] != "2025-01-16T14:45:00Z" {
		t.Errorf("updated_at = %q, want %q", m["updated_at"], "2025-01-16T14:45:00Z")
	}

	// Verify optional pointer fields
	if m["slug"] != "test-task" {
		t.Errorf("slug = %q, want %q", m["slug"], "test-task")
	}
	if m["description"] != "Test description" {
		t.Errorf("description = %q, want %q", m["description"], "Test description")
	}
	if m["agent_type"] != "developer" {
		t.Errorf("agent_type = %q, want %q", m["agent_type"], "developer")
	}
	if m["file_path"] != "docs/task.md" {
		t.Errorf("file_path = %q, want %q", m["file_path"], "docs/task.md")
	}
	if m["execution_order"] != "5" {
		t.Errorf("execution_order = %q, want %q", m["execution_order"], "5")
	}
	if m["blocked_reason"] != "Waiting on API" {
		t.Errorf("blocked_reason = %q, want %q", m["blocked_reason"], "Waiting on API")
	}
	if m["depends_on"] != `["T-E01-F01-001"]` {
		t.Errorf("depends_on = %q, want %q", m["depends_on"], `["T-E01-F01-001"]`)
	}
	if m["completion_notes"] != "Done" {
		t.Errorf("completion_notes = %q, want %q", m["completion_notes"], "Done")
	}
	if m["files_changed"] != `["file1.go","file2.go"]` {
		t.Errorf("files_changed = %q, want %q", m["files_changed"], `["file1.go","file2.go"]`)
	}
}

// TestTaskPlaceholders_NilPointers validates nil pointer fields are omitted
func TestTaskPlaceholders_NilPointers(t *testing.T) {
	task := &models.Task{
		Key:       "T-E07-F01-001",
		Title:     "Test Task",
		Status:    "todo",
		Priority:  5,
		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 16, 14, 45, 0, 0, time.UTC),
	}

	m := TaskPlaceholders(task)

	// Required fields should be present
	if m["id"] != "T-E07-F01-001" {
		t.Errorf("id = %q, want %q", m["id"], "T-E07-F01-001")
	}

	// Nil pointer fields should not be in map
	if _, exists := m["slug"]; exists {
		t.Error("slug should not be in map when nil")
	}
	if _, exists := m["description"]; exists {
		t.Error("description should not be in map when nil")
	}
	if _, exists := m["agent_type"]; exists {
		t.Error("agent_type should not be in map when nil")
	}
	if _, exists := m["file_path"]; exists {
		t.Error("file_path should not be in map when nil")
	}
}

// TestFeaturePlaceholders_AllFields validates all keys from a populated Feature
func TestFeaturePlaceholders_AllFields(t *testing.T) {
	slug := "test-feature"
	description := "Feature description"
	filePath := "docs/feature.md"
	executionOrder := 3

	feature := &models.Feature{
		Key:            "E07-F01",
		Title:          "Test Feature",
		Slug:           &slug,
		Description:    &description,
		Status:         "active",
		FilePath:       &filePath,
		ExecutionOrder: &executionOrder,
		CreatedAt:      time.Date(2025, 1, 10, 9, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC),
	}

	m := FeaturePlaceholders(feature)

	// Verify required fields
	if m["id"] != "E07-F01" {
		t.Errorf("id = %q, want %q", m["id"], "E07-F01")
	}
	if m["feature_id"] != "E07-F01" {
		t.Errorf("feature_id = %q, want %q", m["feature_id"], "E07-F01")
	}
	if m["title"] != "Test Feature" {
		t.Errorf("title = %q, want %q", m["title"], "Test Feature")
	}
	if m["status"] != "active" {
		t.Errorf("status = %q, want %q", m["status"], "active")
	}
	if m["created_at"] != "2025-01-10T09:00:00Z" {
		t.Errorf("created_at = %q, want %q", m["created_at"], "2025-01-10T09:00:00Z")
	}
	if m["updated_at"] != "2025-01-15T12:30:00Z" {
		t.Errorf("updated_at = %q, want %q", m["updated_at"], "2025-01-15T12:30:00Z")
	}

	// Verify optional fields
	if m["slug"] != "test-feature" {
		t.Errorf("slug = %q, want %q", m["slug"], "test-feature")
	}
	if m["description"] != "Feature description" {
		t.Errorf("description = %q, want %q", m["description"], "Feature description")
	}
	if m["file_path"] != "docs/feature.md" {
		t.Errorf("file_path = %q, want %q", m["file_path"], "docs/feature.md")
	}
	if m["execution_order"] != "3" {
		t.Errorf("execution_order = %q, want %q", m["execution_order"], "3")
	}
}

// TestEpicPlaceholders_AllFields validates all keys from a populated Epic
func TestEpicPlaceholders_AllFields(t *testing.T) {
	slug := "test-epic"
	description := "Epic description"
	filePath := "docs/epic.md"
	businessValue := models.PriorityHigh

	epic := &models.Epic{
		Key:           "E07",
		Title:         "Test Epic",
		Slug:          &slug,
		Description:   &description,
		Status:        "active",
		Priority:      models.PriorityHigh,
		BusinessValue: &businessValue,
		FilePath:      &filePath,
		CreatedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2025, 1, 20, 16, 0, 0, 0, time.UTC),
	}

	m := EpicPlaceholders(epic)

	// Verify required fields
	if m["id"] != "E07" {
		t.Errorf("id = %q, want %q", m["id"], "E07")
	}
	if m["epic_id"] != "E07" {
		t.Errorf("epic_id = %q, want %q", m["epic_id"], "E07")
	}
	if m["title"] != "Test Epic" {
		t.Errorf("title = %q, want %q", m["title"], "Test Epic")
	}
	if m["status"] != "active" {
		t.Errorf("status = %q, want %q", m["status"], "active")
	}
	if m["priority"] != "high" {
		t.Errorf("priority = %q, want %q", m["priority"], "high")
	}
	if m["created_at"] != "2025-01-01T00:00:00Z" {
		t.Errorf("created_at = %q, want %q", m["created_at"], "2025-01-01T00:00:00Z")
	}
	if m["updated_at"] != "2025-01-20T16:00:00Z" {
		t.Errorf("updated_at = %q, want %q", m["updated_at"], "2025-01-20T16:00:00Z")
	}

	// Verify optional fields
	if m["slug"] != "test-epic" {
		t.Errorf("slug = %q, want %q", m["slug"], "test-epic")
	}
	if m["description"] != "Epic description" {
		t.Errorf("description = %q, want %q", m["description"], "Epic description")
	}
	if m["file_path"] != "docs/epic.md" {
		t.Errorf("file_path = %q, want %q", m["file_path"], "docs/epic.md")
	}
	if m["business_value"] != "high" {
		t.Errorf("business_value = %q, want %q", m["business_value"], "high")
	}
}
