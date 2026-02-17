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

// TestTaskPlaceholdersWithRelated_HappyPath tests task placeholders with docs and tasks (LEGACY - uses context data)
// NOTE: This test is deprecated and should be replaced with the refactored version
// that uses TaskRelationshipRepository. Keeping for backward compatibility during transition.
func TestTaskPlaceholdersWithRelated_HappyPath(t *testing.T) {
	t.Skip("DEPRECATED: This test uses the old context_data approach. Use TestTaskPlaceholdersWithRelated_Refactored_* instead.")

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

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{"E07-F05-001", "E10-F05-002"},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo)

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

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo)

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
	mockTaskRelRepo := &mockTaskRelationshipRepository{}
	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, nil, mockRepo, mockTaskRelRepo)

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

// Mock TaskRelationshipRepository for testing
type mockTaskRelationshipRepository struct {
	tasks []string
	err   error
}

func (m *mockTaskRelationshipRepository) ListRelatedTaskKeys(ctx context.Context, taskID int64) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}

// TestTaskPlaceholdersWithRelated_DocRepoError tests task placeholders when doc repo fails (TC-PH-06)
func TestTaskPlaceholdersWithRelated_DocRepoError(t *testing.T) {
	task := &models.Task{
		Key:    "T-E07-F29-001",
		Title:  "Test Task",
		Status: "todo",
	}

	mockRepo := &mockDocumentRepository{
		err: fmt.Errorf("database connection lost"),
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{"E07-F05-001"},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo)

	// Should return empty string for docs when repo fails (graceful degradation)
	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs on error = %q, want empty string", relDocs)
	}

	// Related tasks should still work from repository
	if relTasks := result["related_tasks"]; relTasks != "E07-F05-001" {
		t.Errorf("related_tasks = %q, want %q", relTasks, "E07-F05-001")
	}
}

// TestTaskPlaceholdersWithRelated_PartialData tests task with docs but no relationships (TC-PH-03)
func TestTaskPlaceholdersWithRelated_PartialData(t *testing.T) {
	task := &models.Task{
		Key:    "T-E07-F29-001",
		Title:  "Test Task",
		Status: "todo",
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
		},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo)

	if relDocs := result["related_docs"]; relDocs != "docs/spec.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/spec.md")
	}

	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestTaskPlaceholdersWithRelated_MalformedContext tests task when repo query fails (TC-PH-04)
// NOTE: This test was originally for malformed JSON context, but now tests repository error handling
func TestTaskPlaceholdersWithRelated_MalformedContext(t *testing.T) {
	task := &models.Task{
		Key:    "T-E07-F29-001",
		Title:  "Test Task",
		Status: "todo",
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
		},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		err: fmt.Errorf("query failed"),
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo)

	// Should still return docs
	if relDocs := result["related_docs"]; relDocs != "docs/spec.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/spec.md")
	}

	// Related tasks should be empty due to query error
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
		Key:       "T-E07-F29-001",
		Title:     "Test Task",
		Status:    "in_progress",
		Priority:  7,
		Slug:      &slug,
		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
		},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{"E01-F01"},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo)

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
// Tests for refactored TaskPlaceholdersWithRelated using TaskRelationshipRepository
// ============================================================================

// TestTaskPlaceholdersWithRelated_Refactored_WithRelatedTasks tests task placeholder with repository-fetched tasks
func TestTaskPlaceholdersWithRelated_Refactored_WithRelatedTasks(t *testing.T) {
	task := &models.Task{
		ID:     100,
		Key:    "E07-F29-001",
		Title:  "Test Task",
		Status: "todo",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
			{FilePath: "docs/design.md"},
		},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{"E07-F05-001", "E10-F05-002"},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	if relDocs := result["related_docs"]; relDocs != "docs/spec.md,docs/design.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/spec.md,docs/design.md")
	}

	if relTasks := result["related_tasks"]; relTasks != "E07-F05-001,E10-F05-002" {
		t.Errorf("related_tasks = %q, want %q", relTasks, "E07-F05-001,E10-F05-002")
	}
}

// TestTaskPlaceholdersWithRelated_Refactored_NoRelatedTasks tests empty array handling
func TestTaskPlaceholdersWithRelated_Refactored_NoRelatedTasks(t *testing.T) {
	task := &models.Task{
		ID:     200,
		Key:    "E07-F29-002",
		Title:  "Test Task",
		Status: "todo",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{}, // Empty array
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestTaskPlaceholdersWithRelated_Refactored_QueryError tests graceful degradation on repo error
func TestTaskPlaceholdersWithRelated_Refactored_QueryError(t *testing.T) {
	task := &models.Task{
		ID:     300,
		Key:    "E07-F29-003",
		Title:  "Test Task",
		Status: "todo",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{
			{FilePath: "docs/spec.md"},
		},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		err: fmt.Errorf("database connection failed"),
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	// Docs should still work
	if relDocs := result["related_docs"]; relDocs != "docs/spec.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/spec.md")
	}

	// Should return empty string on error (graceful degradation)
	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks on error = %q, want empty string", relTasks)
	}
}

// ptrString returns a pointer to a string (helper for tests)
func ptrString(s string) *string {
	return &s
}

// ========================
// Complexity Tier Tests (Task API-28 and API-29)
// ========================

// TestTaskPlaceholdersWithRelated_ComplexityTierWithValue tests task with complexity_tier in metadata (API-28)
func TestTaskPlaceholdersWithRelated_ComplexityTierWithValue(t *testing.T) {
	task := &models.Task{
		Key:      "E07-F30-001",
		Title:    "Test Task",
		Status:   "todo",
		Metadata: map[string]interface{}{"complexity_tier": "STANDARD"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	if tier := result["complexity_tier"]; tier != "STANDARD" {
		t.Errorf("complexity_tier = %q, want %q", tier, "STANDARD")
	}

	// Verify existing placeholders still present
	if id := result["task_id"]; id != "E07-F30-001" {
		t.Errorf("task_id = %q, want %q", id, "E07-F30-001")
	}
	if title := result["title"]; title != "Test Task" {
		t.Errorf("title = %q, want %q", title, "Test Task")
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierMissing tests task without complexity_tier (API-29)
func TestTaskPlaceholdersWithRelated_ComplexityTierMissing(t *testing.T) {
	task := &models.Task{
		Key:      "E07-F30-002",
		Title:    "Task without tier",
		Status:   "todo",
		Metadata: map[string]interface{}{}, // Empty metadata
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	// Should have empty string for complexity_tier when not set
	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierNilMetadata tests task with nil metadata
func TestTaskPlaceholdersWithRelated_ComplexityTierNilMetadata(t *testing.T) {
	task := &models.Task{
		Key:      "E07-F30-003",
		Title:    "Task with nil metadata",
		Status:   "todo",
		Metadata: nil,
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	// Should have empty string for complexity_tier when metadata is nil
	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierSimple tests task with SIMPLE tier
func TestTaskPlaceholdersWithRelated_ComplexityTierSimple(t *testing.T) {
	task := &models.Task{
		Key:      "E07-F30-004",
		Title:    "Simple task",
		Status:   "todo",
		Metadata: map[string]interface{}{"complexity_tier": "SIMPLE"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	if tier := result["complexity_tier"]; tier != "SIMPLE" {
		t.Errorf("complexity_tier = %q, want %q", tier, "SIMPLE")
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierComplex tests task with COMPLEX tier
func TestTaskPlaceholdersWithRelated_ComplexityTierComplex(t *testing.T) {
	task := &models.Task{
		Key:      "E07-F30-005",
		Title:    "Complex task",
		Status:   "todo",
		Metadata: map[string]interface{}{"complexity_tier": "COMPLEX"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	if tier := result["complexity_tier"]; tier != "COMPLEX" {
		t.Errorf("complexity_tier = %q, want %q", tier, "COMPLEX")
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierInvalidType tests task with non-string complexity_tier
func TestTaskPlaceholdersWithRelated_ComplexityTierInvalidType(t *testing.T) {
	task := &models.Task{
		Key:      "E07-F30-006",
		Title:    "Task with invalid tier type",
		Status:   "todo",
		Metadata: map[string]interface{}{"complexity_tier": 123}, // Integer instead of string
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo)

	// Should return empty string when tier is not a string (type assertion fails)
	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string when non-string", tier)
	}
}

// ========================
// Feature Complexity Tier Tests (API-30)
// ========================

// TestFeaturePlaceholdersWithRelated_ComplexityTierWithValue tests feature with complexity_tier (API-30)
func TestFeaturePlaceholdersWithRelated_ComplexityTierWithValue(t *testing.T) {
	feature := &models.Feature{
		Key:      "E07-F30",
		Title:    "Template engine",
		Status:   "active",
		Metadata: map[string]interface{}{"complexity_tier": "STANDARD"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	if tier := result["complexity_tier"]; tier != "STANDARD" {
		t.Errorf("complexity_tier = %q, want %q", tier, "STANDARD")
	}

	// Verify existing placeholders still present
	if id := result["feature_id"]; id != "E07-F30" {
		t.Errorf("feature_id = %q, want %q", id, "E07-F30")
	}
}

// TestFeaturePlaceholdersWithRelated_ComplexityTierMissing tests feature without complexity_tier
func TestFeaturePlaceholdersWithRelated_ComplexityTierMissing(t *testing.T) {
	feature := &models.Feature{
		Key:      "E07-F31",
		Title:    "Feature without tier",
		Status:   "active",
		Metadata: map[string]interface{}{},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestFeaturePlaceholdersWithRelated_ComplexityTierNilMetadata tests feature with nil metadata
func TestFeaturePlaceholdersWithRelated_ComplexityTierNilMetadata(t *testing.T) {
	feature := &models.Feature{
		Key:      "E07-F32",
		Title:    "Feature with nil metadata",
		Status:   "active",
		Metadata: nil,
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestFeaturePlaceholdersWithRelated_ComplexityTierComplex tests feature with COMPLEX tier
func TestFeaturePlaceholdersWithRelated_ComplexityTierComplex(t *testing.T) {
	feature := &models.Feature{
		Key:      "E07-F33",
		Title:    "Complex feature",
		Status:   "active",
		Metadata: map[string]interface{}{"complexity_tier": "COMPLEX"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo)

	if tier := result["complexity_tier"]; tier != "COMPLEX" {
		t.Errorf("complexity_tier = %q, want %q", tier, "COMPLEX")
	}
}
