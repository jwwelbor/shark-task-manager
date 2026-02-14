package config

import (
	"context"
	"fmt"
	"strings"
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

// TestFormatDocPathsAsCSV_NilSlice tests formatting nil document slice (TC-FMT-01)
func TestFormatDocPathsAsCSV_NilSlice(t *testing.T) {
	result := formatDocPathsAsCSV(nil)
	if result != "" {
		t.Errorf("formatDocPathsAsCSV(nil) = %q, want empty string", result)
	}
}

// TestFormatDocPathsAsCSV_EmptySlice tests formatting empty document slice (TC-FMT-02)
func TestFormatDocPathsAsCSV_EmptySlice(t *testing.T) {
	result := formatDocPathsAsCSV([]*models.Document{})
	if result != "" {
		t.Errorf("formatDocPathsAsCSV([]) = %q, want empty string", result)
	}
}

// TestFormatDocPathsAsCSV_SingleDoc tests formatting single document (TC-FMT-03)
func TestFormatDocPathsAsCSV_SingleDoc(t *testing.T) {
	docs := []*models.Document{
		{FilePath: "docs/a.md"},
	}
	result := formatDocPathsAsCSV(docs)
	if result != "docs/a.md" {
		t.Errorf("formatDocPathsAsCSV(single) = %q, want %q", result, "docs/a.md")
	}
}

// TestFormatDocPathsAsCSV_MultipleDocs tests formatting multiple documents (TC-FMT-04)
func TestFormatDocPathsAsCSV_MultipleDocs(t *testing.T) {
	docs := []*models.Document{
		{FilePath: "docs/a.md"},
		{FilePath: "docs/b.md"},
		{FilePath: "docs/c.md"},
	}
	result := formatDocPathsAsCSV(docs)
	expected := "docs/a.md,docs/b.md,docs/c.md"
	if result != expected {
		t.Errorf("formatDocPathsAsCSV(three) = %q, want %q", result, expected)
	}
}

// TestFormatDocPathsAsCSV_DocsWithSpaces tests formatting documents with spaces in path (TC-FMT-05)
func TestFormatDocPathsAsCSV_DocsWithSpaces(t *testing.T) {
	docs := []*models.Document{
		{FilePath: "docs/My Doc.md"},
		{FilePath: "docs/Other File.md"},
	}
	result := formatDocPathsAsCSV(docs)
	expected := "docs/My Doc.md,docs/Other File.md"
	if result != expected {
		t.Errorf("formatDocPathsAsCSV(spaces) = %q, want %q", result, expected)
	}
}

// TestFormatDocPathsAsCSV_LargeList tests formatting 50 documents (TC-FMT-06)
func TestFormatDocPathsAsCSV_LargeList(t *testing.T) {
	docs := make([]*models.Document, 50)
	for i := 0; i < 50; i++ {
		path := fmt.Sprintf("docs/doc%d.md", i)
		docs[i] = &models.Document{FilePath: path}
	}
	result := formatDocPathsAsCSV(docs)

	// Verify all 50 paths are present
	for i := 0; i < 50; i++ {
		path := fmt.Sprintf("docs/doc%d.md", i)
		if !strings.Contains(result, path) {
			t.Errorf("formatDocPathsAsCSV(50) missing path %q", path)
		}
	}

	// Count commas (should be 49)
	commaCount := strings.Count(result, ",")
	if commaCount != 49 {
		t.Errorf("formatDocPathsAsCSV(50) has %d commas, want 49", commaCount)
	}
}

// TestExtractRelatedTasksFromContext tests the extractRelatedTasksFromContext function
func TestExtractRelatedTasksFromContext(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
		desc     string
	}{
		// TC-CTX-01: Nil context
		{
			name:     "nil_context",
			input:    nil,
			expected: "",
			desc:     "Nil context should return empty string",
		},
		// TC-CTX-02: Empty string
		{
			name:     "empty_string",
			input:    ptrString(""),
			expected: "",
			desc:     "Empty string should return empty string",
		},
		// TC-CTX-03: Valid JSON with 2 tasks (happy path)
		{
			name:     "valid_json_two_tasks",
			input:    ptrString(`{"related_tasks":["E01-F01","E02-F01"]}`),
			expected: "E01-F01,E02-F01",
			desc:     "Valid JSON with 2 tasks should extract both in CSV format",
		},
		// TC-CTX-04: Valid JSON with empty array
		{
			name:     "valid_json_empty_array",
			input:    ptrString(`{"related_tasks":[]}`),
			expected: "",
			desc:     "Valid JSON with empty array should return empty string",
		},
		// TC-CTX-05: Valid JSON without related_tasks field
		{
			name:     "valid_json_missing_field",
			input:    ptrString(`{"other_field":"value"}`),
			expected: "",
			desc:     "Valid JSON without related_tasks field should return empty string",
		},
		// TC-CTX-06: Malformed JSON
		{
			name:     "malformed_json",
			input:    ptrString(`"{invalid}"`),
			expected: "",
			desc:     "Malformed JSON should return empty string (no error, warning logged)",
		},
		// TC-CTX-07: Valid JSON with null related_tasks
		{
			name:     "valid_json_null_field",
			input:    ptrString(`{"related_tasks":null}`),
			expected: "",
			desc:     "Valid JSON with null related_tasks should return empty string",
		},
		// Additional: Single task
		{
			name:     "valid_json_single_task",
			input:    ptrString(`{"related_tasks":["E07-F05-001"]}`),
			expected: "E07-F05-001",
			desc:     "Valid JSON with single task should extract it",
		},
		// Additional: Multiple tasks with more complex format
		{
			name:     "valid_json_multiple_tasks",
			input:    ptrString(`{"related_tasks":["E01-F01-001","E07-F05-002","E10-F20-003"]}`),
			expected: "E01-F01-001,E07-F05-002,E10-F20-003",
			desc:     "Valid JSON with multiple tasks should extract all in order",
		},
		// Additional: JSON with extra fields
		{
			name:     "valid_json_with_extra_fields",
			input:    ptrString(`{"description":"test","related_tasks":["E01-F01"],"other":"data"}`),
			expected: "E01-F01",
			desc:     "Valid JSON with extra fields should extract only related_tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRelatedTasksFromContext(tt.input)
			if result != tt.expected {
				t.Errorf("%s: got %q, want %q", tt.desc, result, tt.expected)
			}
		})
	}
}

// ptrString is a helper to create string pointers for tests
func ptrString(s string) *string {
	return &s
}

// TestFormatFeatureKeysAsCSV_NilSlice tests nil feature keys slice
func TestFormatFeatureKeysAsCSV_NilSlice(t *testing.T) {
	result := formatFeatureKeysAsCSV(nil)
	if result != "" {
		t.Errorf("formatFeatureKeysAsCSV(nil) = %q, want empty string", result)
	}
}

// TestFormatFeatureKeysAsCSV_EmptySlice tests empty feature keys slice
func TestFormatFeatureKeysAsCSV_EmptySlice(t *testing.T) {
	result := formatFeatureKeysAsCSV([]string{})
	if result != "" {
		t.Errorf("formatFeatureKeysAsCSV([]) = %q, want empty string", result)
	}
}

// TestFormatFeatureKeysAsCSV_SingleKey tests single feature key formatting
func TestFormatFeatureKeysAsCSV_SingleKey(t *testing.T) {
	result := formatFeatureKeysAsCSV([]string{"E07-F05"})
	expected := "E07-F05"
	if result != expected {
		t.Errorf("formatFeatureKeysAsCSV() = %q, want %q", result, expected)
	}
}

// TestFormatFeatureKeysAsCSV_MultipleKeys tests multiple feature keys formatting
func TestFormatFeatureKeysAsCSV_MultipleKeys(t *testing.T) {
	result := formatFeatureKeysAsCSV([]string{"E07-F05", "E07-F21", "E10-F05"})
	expected := "E07-F05,E07-F21,E10-F05"
	if result != expected {
		t.Errorf("formatFeatureKeysAsCSV() = %q, want %q", result, expected)
	}
}

// TestFormatEpicKeysAsCSV_NilSlice tests nil epic keys slice
func TestFormatEpicKeysAsCSV_NilSlice(t *testing.T) {
	result := formatEpicKeysAsCSV(nil)
	if result != "" {
		t.Errorf("formatEpicKeysAsCSV(nil) = %q, want empty string", result)
	}
}

// TestFormatEpicKeysAsCSV_EmptySlice tests empty epic keys slice
func TestFormatEpicKeysAsCSV_EmptySlice(t *testing.T) {
	result := formatEpicKeysAsCSV([]string{})
	if result != "" {
		t.Errorf("formatEpicKeysAsCSV([]) = %q, want empty string", result)
	}
}

// TestFormatEpicKeysAsCSV_SingleKey tests single epic key formatting
func TestFormatEpicKeysAsCSV_SingleKey(t *testing.T) {
	result := formatEpicKeysAsCSV([]string{"E01"})
	expected := "E01"
	if result != expected {
		t.Errorf("formatEpicKeysAsCSV() = %q, want %q", result, expected)
	}
}

// TestFormatEpicKeysAsCSV_MultipleKeys tests multiple epic keys formatting
func TestFormatEpicKeysAsCSV_MultipleKeys(t *testing.T) {
	result := formatEpicKeysAsCSV([]string{"E01", "E05", "E07"})
	expected := "E01,E05,E07"
	if result != expected {
		t.Errorf("formatEpicKeysAsCSV() = %q, want %q", result, expected)
	}
}

// TestTaskPlaceholdersWithRelated_HappyPath tests task placeholders with docs and tasks
func TestTaskPlaceholdersWithRelated_HappyPath(t *testing.T) {
	task := &models.Task{
		Key:         "T-E07-F29-001",
		Title:       "Test Task",
		Status:      "todo",
		ContextData: ptrString(`{"related_tasks":["E07-F05-001","E10-F05-002"]}`),
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
			{FilePath: "docs/b.md"},
		},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(task, mockRepo, ctx)

	if relDocs := result["related_docs"]; relDocs != "docs/a.md,docs/b.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md,docs/b.md")
	}

	if relTasks := result["related_tasks"]; relTasks != "E07-F05-001,E10-F05-002" {
		t.Errorf("related_tasks = %q, want %q", relTasks, "E07-F05-001,E10-F05-002")
	}
}

// TestTaskPlaceholdersWithRelated_NoData tests task placeholders with no data
func TestTaskPlaceholdersWithRelated_NoData(t *testing.T) {
	task := &models.Task{
		Key:    "T-E07-F29-002",
		Title:  "Empty Task",
		Status: "todo",
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(task, mockRepo, ctx)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestTaskPlaceholdersWithRelated_NilTask tests task placeholders with nil task
func TestTaskPlaceholdersWithRelated_NilTask(t *testing.T) {
	mockRepo := &mockDocumentRepository{}
	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(nil, mockRepo, ctx)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestFeaturePlaceholdersWithRelated_HappyPath tests feature placeholders with docs and features
func TestFeaturePlaceholdersWithRelated_HappyPath(t *testing.T) {
	feature := &models.Feature{
		Key:    "E07-F29",
		Title:  "Test Feature",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
			{FilePath: "docs/b.md"},
		},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{"E07-F05", "E07-F21", "E10-F05"},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	if relDocs := result["related_docs"]; relDocs != "docs/a.md,docs/b.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md,docs/b.md")
	}

	if relFeatures := result["related_features"]; relFeatures != "E07-F05,E07-F21,E10-F05" {
		t.Errorf("related_features = %q, want %q", relFeatures, "E07-F05,E07-F21,E10-F05")
	}
}

// TestFeaturePlaceholdersWithRelated_NoData tests feature placeholders with no data
func TestFeaturePlaceholdersWithRelated_NoData(t *testing.T) {
	feature := &models.Feature{
		Key:    "E07-F30",
		Title:  "Empty Feature",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relFeatures := result["related_features"]; relFeatures != "" {
		t.Errorf("related_features = %q, want empty string", relFeatures)
	}
}

// TestFeaturePlaceholdersWithRelated_NilFeature tests feature placeholders with nil feature
func TestFeaturePlaceholdersWithRelated_NilFeature(t *testing.T) {
	mockDocRepo := &mockDocumentRepository{}
	mockRelRepo := &mockFeatureRelationshipRepository{}
	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, nil, mockDocRepo, mockRelRepo)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relFeatures := result["related_features"]; relFeatures != "" {
		t.Errorf("related_features = %q, want empty string", relFeatures)
	}
}

// TestEpicPlaceholdersWithRelated_HappyPath tests epic placeholders with docs and epics
func TestEpicPlaceholdersWithRelated_HappyPath(t *testing.T) {
	epic := &models.Epic{
		Key:    "E07",
		Title:  "Test Epic",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
			{FilePath: "docs/b.md"},
		},
	}

	mockRelRepo := &mockEpicRelationshipRepository{
		epics: []string{"E01", "E05"},
	}

	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx)

	if relDocs := result["related_docs"]; relDocs != "docs/a.md,docs/b.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md,docs/b.md")
	}

	if relEpics := result["related_epics"]; relEpics != "E01,E05" {
		t.Errorf("related_epics = %q, want %q", relEpics, "E01,E05")
	}
}

// TestEpicPlaceholdersWithRelated_NoData tests epic placeholders with no data
func TestEpicPlaceholdersWithRelated_NoData(t *testing.T) {
	epic := &models.Epic{
		Key:    "E08",
		Title:  "Empty Epic",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockEpicRelationshipRepository{
		epics: []string{},
	}

	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relEpics := result["related_epics"]; relEpics != "" {
		t.Errorf("related_epics = %q, want empty string", relEpics)
	}
}

// TestEpicPlaceholdersWithRelated_NilEpic tests epic placeholders with nil epic
func TestEpicPlaceholdersWithRelated_NilEpic(t *testing.T) {
	mockDocRepo := &mockDocumentRepository{}
	mockRelRepo := &mockEpicRelationshipRepository{}
	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(nil, mockDocRepo, mockRelRepo, ctx)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relEpics := result["related_epics"]; relEpics != "" {
		t.Errorf("related_epics = %q, want empty string", relEpics)
	}
}

// Mock DocumentRepository for testing
type mockDocumentRepository struct {
	docs []*models.Document
	err  error
}

func (m *mockDocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.docs, nil
}

func (m *mockDocumentRepository) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.docs, nil
}

func (m *mockDocumentRepository) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.docs, nil
}

// Mock FeatureRelationshipRepository for testing
type mockFeatureRelationshipRepository struct {
	features []string
	err      error
}

func (m *mockFeatureRelationshipRepository) ListRelatedFeatures(ctx context.Context, featureID int64) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.features, nil
}

// Mock EpicRelationshipRepository for testing
type mockEpicRelationshipRepository struct {
	epics []string
	err   error
}

func (m *mockEpicRelationshipRepository) ListRelatedEpics(ctx context.Context, epicID int64) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.epics, nil
}

// TestTaskPlaceholdersWithRelated_DocRepoError tests task placeholders when doc repo fails (TC-PH-06)
func TestTaskPlaceholdersWithRelated_DocRepoError(t *testing.T) {
	task := &models.Task{
		Key:         "T-E07-F29-001",
		Title:       "Test Task",
		Status:      "todo",
		ContextData: ptrString(`{"related_tasks":["E07-F05-001"]}`),
	}

	mockRepo := &mockDocumentRepository{
		err: fmt.Errorf("database connection lost"),
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(task, mockRepo, ctx)

	// Should return empty string for docs when repo fails (graceful degradation)
	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs on error = %q, want empty string", relDocs)
	}

	// Related tasks should still work from context data
	if relTasks := result["related_tasks"]; relTasks != "E07-F05-001" {
		t.Errorf("related_tasks = %q, want %q", relTasks, "E07-F05-001")
	}
}

// TestTaskPlaceholdersWithRelated_PartialData tests task with docs but no context data (TC-PH-03)
func TestTaskPlaceholdersWithRelated_PartialData(t *testing.T) {
	task := &models.Task{
		Key:    "T-E07-F29-001",
		Title:  "Test Task",
		Status: "todo",
		// No context data
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
		},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(task, mockRepo, ctx)

	if relDocs := result["related_docs"]; relDocs != "docs/spec.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/spec.md")
	}

	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestTaskPlaceholdersWithRelated_MalformedContext tests task with malformed JSON context (TC-PH-04)
func TestTaskPlaceholdersWithRelated_MalformedContext(t *testing.T) {
	task := &models.Task{
		Key:         "T-E07-F29-001",
		Title:       "Test Task",
		Status:      "todo",
		ContextData: ptrString(`{invalid json}`),
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
		},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(task, mockRepo, ctx)

	// Should still return docs even with malformed context
	if relDocs := result["related_docs"]; relDocs != "docs/spec.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/spec.md")
	}

	// Related tasks should be empty due to malformed JSON
	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestFeaturePlaceholdersWithRelated_DocRepoError tests feature placeholders when doc repo fails (TC-FPH-04)
func TestFeaturePlaceholdersWithRelated_DocRepoError(t *testing.T) {
	feature := &models.Feature{
		Key:    "E07-F29",
		Title:  "Test Feature",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		err: fmt.Errorf("database connection lost"),
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{"E07-F05", "E07-F21"},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	// Should return empty string for docs when repo fails
	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs on error = %q, want empty string", relDocs)
	}

	// Related features should still work
	if relFeatures := result["related_features"]; relFeatures != "E07-F05,E07-F21" {
		t.Errorf("related_features = %q, want %q", relFeatures, "E07-F05,E07-F21")
	}
}

// TestFeaturePlaceholdersWithRelated_FeatureRelRepoError tests feature placeholders when feature rel repo fails (TC-FPH-04)
func TestFeaturePlaceholdersWithRelated_FeatureRelRepoError(t *testing.T) {
	feature := &models.Feature{
		Key:    "E07-F29",
		Title:  "Test Feature",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
		},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		err: fmt.Errorf("relationship table query failed"),
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	// Related docs should still work
	if relDocs := result["related_docs"]; relDocs != "docs/a.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md")
	}

	// Related features should be empty on error
	if relFeatures := result["related_features"]; relFeatures != "" {
		t.Errorf("related_features on error = %q, want empty string", relFeatures)
	}
}

// TestFeaturePlaceholdersWithRelated_CrossEpic tests feature placeholders with cross-epic relationships (TC-FPH-05)
func TestFeaturePlaceholdersWithRelated_CrossEpic(t *testing.T) {
	feature := &models.Feature{
		Key:    "E07-F29",
		Title:  "Test Feature",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
		},
	}

	// Cross-epic relationships
	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{"E01-F01", "E07-F05", "E10-F20"},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	// Should include cross-epic feature keys
	if relFeatures := result["related_features"]; relFeatures != "E01-F01,E07-F05,E10-F20" {
		t.Errorf("related_features with cross-epic = %q, want %q", relFeatures, "E01-F01,E07-F05,E10-F20")
	}
}

// TestEpicPlaceholdersWithRelated_DocRepoError tests epic placeholders when doc repo fails (TC-EPH-04)
func TestEpicPlaceholdersWithRelated_DocRepoError(t *testing.T) {
	epic := &models.Epic{
		Key:    "E07",
		Title:  "Test Epic",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		err: fmt.Errorf("database connection lost"),
	}

	mockRelRepo := &mockEpicRelationshipRepository{
		epics: []string{"E01", "E05"},
	}

	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx)

	// Should return empty string for docs when repo fails
	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs on error = %q, want empty string", relDocs)
	}

	// Related epics should still work
	if relEpics := result["related_epics"]; relEpics != "E01,E05" {
		t.Errorf("related_epics = %q, want %q", relEpics, "E01,E05")
	}
}

// TestEpicPlaceholdersWithRelated_EpicRelRepoError tests epic placeholders when epic rel repo fails (TC-EPH-04)
func TestEpicPlaceholdersWithRelated_EpicRelRepoError(t *testing.T) {
	epic := &models.Epic{
		Key:    "E07",
		Title:  "Test Epic",
		Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
		},
	}

	mockRelRepo := &mockEpicRelationshipRepository{
		err: fmt.Errorf("epic relationship table query failed"),
	}

	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx)

	// Related docs should still work
	if relDocs := result["related_docs"]; relDocs != "docs/a.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md")
	}

	// Related epics should be empty on error
	if relEpics := result["related_epics"]; relEpics != "" {
		t.Errorf("related_epics on error = %q, want empty string", relEpics)
	}
}

// TestFormatDocPathsAsCSV_AllNilDocuments tests formatting documents with nil pointers
func TestFormatDocPathsAsCSV_AllNilDocuments(t *testing.T) {
	docs := []*models.Document{nil, nil, nil}
	result := formatDocPathsAsCSV(docs)
	if result != "" {
		t.Errorf("formatDocPathsAsCSV(all nil) = %q, want empty string", result)
	}
}

// TestFormatDocPathsAsCSV_MixedValidAndEmpty tests formatting documents with empty file paths
func TestFormatDocPathsAsCSV_MixedValidAndEmpty(t *testing.T) {
	docs := []*models.Document{
		{FilePath: "docs/a.md"},
		{FilePath: ""},
		{FilePath: "docs/b.md"},
	}
	result := formatDocPathsAsCSV(docs)
	// Should skip empty paths
	expected := "docs/a.md,docs/b.md"
	if result != expected {
		t.Errorf("formatDocPathsAsCSV(mixed) = %q, want %q", result, expected)
	}
}

// TestTaskPlaceholdersWithRelated_BasicPlaceholders tests that basic placeholders are still present
func TestTaskPlaceholdersWithRelated_BasicPlaceholders(t *testing.T) {
	slug := "test-task"
	task := &models.Task{
		Key:         "T-E07-F29-001",
		Title:       "Test Task",
		Status:      "in_progress",
		Priority:    7,
		Slug:        &slug,
		ContextData: ptrString(`{"related_tasks":["E01-F01"]}`),
		CreatedAt:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
		},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(task, mockRepo, ctx)

	// Verify basic placeholders are still there
	if result["id"] != "T-E07-F29-001" {
		t.Errorf("id placeholder missing or incorrect")
	}
	if result["title"] != "Test Task" {
		t.Errorf("title placeholder missing or incorrect")
	}
	if result["status"] != "in_progress" {
		t.Errorf("status placeholder missing or incorrect")
	}
	if result["priority"] != "7" {
		t.Errorf("priority placeholder missing or incorrect")
	}
	if result["slug"] != "test-task" {
		t.Errorf("slug placeholder missing or incorrect")
	}

	// Verify new placeholders are added
	if result["related_docs"] != "docs/spec.md" {
		t.Errorf("related_docs placeholder missing or incorrect")
	}
	if result["related_tasks"] != "E01-F01" {
		t.Errorf("related_tasks placeholder missing or incorrect")
	}
}

// TestFeaturePlaceholdersWithRelated_BasicPlaceholders tests that basic placeholders are still present
func TestFeaturePlaceholdersWithRelated_BasicPlaceholders(t *testing.T) {
	slug := "test-feature"
	feature := &models.Feature{
		Key:    "E07-F29",
		Title:  "Test Feature",
		Status: "active",
		Slug:   &slug,
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
		},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{"E07-F05"},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	// Verify basic placeholders
	if result["id"] != "E07-F29" {
		t.Errorf("id placeholder missing or incorrect")
	}
	if result["feature_id"] != "E07-F29" {
		t.Errorf("feature_id placeholder missing or incorrect")
	}
	if result["title"] != "Test Feature" {
		t.Errorf("title placeholder missing or incorrect")
	}
	if result["slug"] != "test-feature" {
		t.Errorf("slug placeholder missing or incorrect")
	}

	// Verify new placeholders
	if result["related_docs"] != "docs/a.md" {
		t.Errorf("related_docs placeholder missing or incorrect")
	}
	if result["related_features"] != "E07-F05" {
		t.Errorf("related_features placeholder missing or incorrect")
	}
}

// TestEpicPlaceholdersWithRelated_BasicPlaceholders tests that basic placeholders are still present
func TestEpicPlaceholdersWithRelated_BasicPlaceholders(t *testing.T) {
	slug := "test-epic"
	epic := &models.Epic{
		Key:      "E07",
		Title:    "Test Epic",
		Status:   "active",
		Priority: models.PriorityHigh,
		Slug:     &slug,
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/a.md"},
		},
	}

	mockRelRepo := &mockEpicRelationshipRepository{
		epics: []string{"E01"},
	}

	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx)

	// Verify basic placeholders
	if result["id"] != "E07" {
		t.Errorf("id placeholder missing or incorrect")
	}
	if result["epic_id"] != "E07" {
		t.Errorf("epic_id placeholder missing or incorrect")
	}
	if result["title"] != "Test Epic" {
		t.Errorf("title placeholder missing or incorrect")
	}
	if result["slug"] != "test-epic" {
		t.Errorf("slug placeholder missing or incorrect")
	}

	// Verify new placeholders
	if result["related_docs"] != "docs/a.md" {
		t.Errorf("related_docs placeholder missing or incorrect")
	}
	if result["related_epics"] != "E01" {
		t.Errorf("related_epics placeholder missing or incorrect")
	}
}

// ============================================================================
// Tests for extractRelatedFeaturesFromContext (new helper function)
// ============================================================================

// TestExtractRelatedFeaturesFromContext_Nil tests extracting features from nil context
func TestExtractRelatedFeaturesFromContext_Nil(t *testing.T) {
	result := extractRelatedFeaturesFromContext(nil)
	if result != "" {
		t.Errorf("extractRelatedFeaturesFromContext(nil) = %q, want empty string", result)
	}
}

// TestExtractRelatedFeaturesFromContext_Empty tests extracting features from empty context
func TestExtractRelatedFeaturesFromContext_Empty(t *testing.T) {
	result := extractRelatedFeaturesFromContext(ptrString(""))
	if result != "" {
		t.Errorf("extractRelatedFeaturesFromContext(empty) = %q, want empty string", result)
	}
}

// TestExtractRelatedFeaturesFromContext_HappyPath tests extracting features from valid JSON
func TestExtractRelatedFeaturesFromContext_HappyPath(t *testing.T) {
	contextJSON := `{"related_features":["E07-F05","E07-F21","E10-F05"]}`
	result := extractRelatedFeaturesFromContext(ptrString(contextJSON))
	expected := "E07-F05,E07-F21,E10-F05"
	if result != expected {
		t.Errorf("extractRelatedFeaturesFromContext(valid JSON) = %q, want %q", result, expected)
	}
}

// TestExtractRelatedFeaturesFromContext_SingleFeature tests extracting single feature
func TestExtractRelatedFeaturesFromContext_SingleFeature(t *testing.T) {
	contextJSON := `{"related_features":["E01-F05"]}`
	result := extractRelatedFeaturesFromContext(ptrString(contextJSON))
	expected := "E01-F05"
	if result != expected {
		t.Errorf("extractRelatedFeaturesFromContext(single) = %q, want %q", result, expected)
	}
}

// TestExtractRelatedFeaturesFromContext_EmptyArray tests extracting features from empty array
func TestExtractRelatedFeaturesFromContext_EmptyArray(t *testing.T) {
	contextJSON := `{"related_features":[]}`
	result := extractRelatedFeaturesFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedFeaturesFromContext(empty array) = %q, want empty string", result)
	}
}

// TestExtractRelatedFeaturesFromContext_MissingField tests extracting features when field is missing
func TestExtractRelatedFeaturesFromContext_MissingField(t *testing.T) {
	contextJSON := `{"other_field":"value"}`
	result := extractRelatedFeaturesFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedFeaturesFromContext(missing field) = %q, want empty string", result)
	}
}

// TestExtractRelatedFeaturesFromContext_NullField tests extracting features when field is null
func TestExtractRelatedFeaturesFromContext_NullField(t *testing.T) {
	contextJSON := `{"related_features":null}`
	result := extractRelatedFeaturesFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedFeaturesFromContext(null field) = %q, want empty string", result)
	}
}

// TestExtractRelatedFeaturesFromContext_MalformedJSON tests extracting features from malformed JSON
func TestExtractRelatedFeaturesFromContext_MalformedJSON(t *testing.T) {
	contextJSON := `{invalid json}`
	result := extractRelatedFeaturesFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedFeaturesFromContext(malformed) = %q, want empty string", result)
	}
}

// TestExtractRelatedFeaturesFromContext_CrossEpicFeatures tests extracting cross-epic feature keys
func TestExtractRelatedFeaturesFromContext_CrossEpicFeatures(t *testing.T) {
	contextJSON := `{"related_features":["E01-F01","E07-F05","E10-F20"]}`
	result := extractRelatedFeaturesFromContext(ptrString(contextJSON))
	expected := "E01-F01,E07-F05,E10-F20"
	if result != expected {
		t.Errorf("extractRelatedFeaturesFromContext(cross-epic) = %q, want %q", result, expected)
	}
}

// ============================================================================
// Tests for extractRelatedEpicsFromContext (new helper function)
// ============================================================================

// TestExtractRelatedEpicsFromContext_Nil tests extracting epics from nil context
func TestExtractRelatedEpicsFromContext_Nil(t *testing.T) {
	result := extractRelatedEpicsFromContext(nil)
	if result != "" {
		t.Errorf("extractRelatedEpicsFromContext(nil) = %q, want empty string", result)
	}
}

// TestExtractRelatedEpicsFromContext_Empty tests extracting epics from empty context
func TestExtractRelatedEpicsFromContext_Empty(t *testing.T) {
	result := extractRelatedEpicsFromContext(ptrString(""))
	if result != "" {
		t.Errorf("extractRelatedEpicsFromContext(empty) = %q, want empty string", result)
	}
}

// TestExtractRelatedEpicsFromContext_HappyPath tests extracting epics from valid JSON
func TestExtractRelatedEpicsFromContext_HappyPath(t *testing.T) {
	contextJSON := `{"related_epics":["E01","E05","E07"]}`
	result := extractRelatedEpicsFromContext(ptrString(contextJSON))
	expected := "E01,E05,E07"
	if result != expected {
		t.Errorf("extractRelatedEpicsFromContext(valid JSON) = %q, want %q", result, expected)
	}
}

// TestExtractRelatedEpicsFromContext_SingleEpic tests extracting single epic
func TestExtractRelatedEpicsFromContext_SingleEpic(t *testing.T) {
	contextJSON := `{"related_epics":["E01"]}`
	result := extractRelatedEpicsFromContext(ptrString(contextJSON))
	expected := "E01"
	if result != expected {
		t.Errorf("extractRelatedEpicsFromContext(single) = %q, want %q", result, expected)
	}
}

// TestExtractRelatedEpicsFromContext_EmptyArray tests extracting epics from empty array
func TestExtractRelatedEpicsFromContext_EmptyArray(t *testing.T) {
	contextJSON := `{"related_epics":[]}`
	result := extractRelatedEpicsFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedEpicsFromContext(empty array) = %q, want empty string", result)
	}
}

// TestExtractRelatedEpicsFromContext_MissingField tests extracting epics when field is missing
func TestExtractRelatedEpicsFromContext_MissingField(t *testing.T) {
	contextJSON := `{"other_field":"value"}`
	result := extractRelatedEpicsFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedEpicsFromContext(missing field) = %q, want empty string", result)
	}
}

// TestExtractRelatedEpicsFromContext_NullField tests extracting epics when field is null
func TestExtractRelatedEpicsFromContext_NullField(t *testing.T) {
	contextJSON := `{"related_epics":null}`
	result := extractRelatedEpicsFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedEpicsFromContext(null field) = %q, want empty string", result)
	}
}

// TestExtractRelatedEpicsFromContext_MalformedJSON tests extracting epics from malformed JSON
func TestExtractRelatedEpicsFromContext_MalformedJSON(t *testing.T) {
	contextJSON := `{invalid json}`
	result := extractRelatedEpicsFromContext(ptrString(contextJSON))
	if result != "" {
		t.Errorf("extractRelatedEpicsFromContext(malformed) = %q, want empty string", result)
	}
}

// TestExtractRelatedEpicsFromContext_MultipleEpics tests extracting multiple epic keys
func TestExtractRelatedEpicsFromContext_MultipleEpics(t *testing.T) {
	contextJSON := `{"related_epics":["E01","E02","E03","E04","E05"]}`
	result := extractRelatedEpicsFromContext(ptrString(contextJSON))
	expected := "E01,E02,E03,E04,E05"
	if result != expected {
		t.Errorf("extractRelatedEpicsFromContext(multiple) = %q, want %q", result, expected)
	}
}
