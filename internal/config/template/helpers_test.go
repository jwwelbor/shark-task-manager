package template

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

	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F01-001",
		Title:       "Test Task",
		Slug:        &slug,
		Description: &description,

		FilePath: &filePath,

		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 16, 14, 45, 0, 0, time.UTC)}, Status: "todo",
		AgentType: &agentType,
		Priority:  5,

		ExecutionOrder:  &executionOrder,
		BlockedReason:   &blockedReason,
		DependsOn:       &dependsOn,
		CompletionNotes: &completionNotes,
		FilesChanged:    &filesChanged,
	}

	m := TaskPlaceholders(task)

	// Verify required fields
	if m["id"] != "T-E07-F01-001" {
		t.Errorf("id = %q, want %q", m["id"], "T-E07-F01-001")
	}
	if m["key"] != "T-E07-F01-001" {
		t.Errorf("key = %q, want %q", m["key"], "T-E07-F01-001")
	}
	if m["task_key"] != "T-E07-F01-001" {
		t.Errorf("task_key = %q, want %q", m["task_key"], "T-E07-F01-001")
	}
	if m["epic_key"] != "E07" {
		t.Errorf("epic_key = %q, want %q", m["epic_key"], "E07")
	}
	if m["feature_key"] != "E07-F01" {
		t.Errorf("feature_key = %q, want %q", m["feature_key"], "E07-F01")
	}
	// Verify backward-compatible aliases are present
	if m["task_id"] != "T-E07-F01-001" {
		t.Errorf("task_id (backward compat) = %q, want %q", m["task_id"], "T-E07-F01-001")
	}
	if m["epic_id"] != "E07" {
		t.Errorf("epic_id (backward compat) = %q, want %q", m["epic_id"], "E07")
	}
	if m["feature_id"] != "E07-F01" {
		t.Errorf("feature_id (backward compat) = %q, want %q", m["feature_id"], "E07-F01")
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
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F01-001",
		Title: "Test Task",

		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 16, 14, 45, 0, 0, time.UTC)}, Status: "todo",
		Priority: 5,
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

	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01",
		Title:       "Test Feature",
		Slug:        &slug,
		Description: &description,

		FilePath: &filePath,

		CreatedAt: time.Date(2025, 1, 10, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC)}, Status: "active",

		ExecutionOrder: &executionOrder,
	}

	m := FeaturePlaceholders(feature)

	// Verify required fields
	if m["id"] != "E07-F01" {
		t.Errorf("id = %q, want %q", m["id"], "E07-F01")
	}
	if m["key"] != "E07-F01" {
		t.Errorf("key = %q, want %q", m["key"], "E07-F01")
	}
	if m["epic_key"] != "E07" {
		t.Errorf("epic_key = %q, want %q", m["epic_key"], "E07")
	}
	// Verify backward-compatible aliases
	if m["feature_id"] != "E07-F01" {
		t.Errorf("feature_id (backward compat) = %q, want %q", m["feature_id"], "E07-F01")
	}
	if m["epic_id"] != "E07" {
		t.Errorf("epic_id (backward compat) = %q, want %q", m["epic_id"], "E07")
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

	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E07",
		Title:       "Test Epic",
		Slug:        &slug,
		Description: &description,

		FilePath:  &filePath,
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 20, 16, 0, 0, 0, time.UTC)}, Status: "active",
		Priority:      models.PriorityHigh,
		BusinessValue: &businessValue,
	}

	m := EpicPlaceholders(epic)

	// Verify required fields
	if m["id"] != "E07" {
		t.Errorf("id = %q, want %q", m["id"], "E07")
	}
	if m["key"] != "E07" {
		t.Errorf("key = %q, want %q", m["key"], "E07")
	}
	// Verify backward-compatible alias
	if m["epic_id"] != "E07" {
		t.Errorf("epic_id (backward compat) = %q, want %q", m["epic_id"], "E07")
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

// TestDeriveReviewBase covers the file_path → review_base derivation that
// feeds the <review-base> placeholder consumed by the code-review / QA / UAT
// partials. Regression for B025: shark next was exiting 3 on
// ready_for_code_review because no key populated <review-base>.
func TestDeriveReviewBase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{
			"task under feature",
			"docs/plan/E19-sprint/E19-F04-analytics/tasks/T-E19-F04-001.md",
			"docs/review/E19-sprint/E19-F04-analytics/",
		},
		{
			"feature spec",
			"docs/plan/E19-sprint/E19-F04-analytics/feature.md",
			"docs/review/E19-sprint/E19-F04-analytics/",
		},
		{
			"bug under docs/plan/bugs",
			"docs/plan/bugs/B025.md",
			"docs/review/bugs/",
		},
		{
			"epic top-level file",
			"docs/plan/E19-sprint/epic.md",
			"docs/review/E19-sprint/",
		},
		{
			"non-plan path passes through with trailing slash",
			"custom/E07-F01/tasks/T-E07-F01-001.md",
			"custom/E07-F01/",
		},
		{
			"leading ./ stripped",
			"./docs/plan/bugs/B025.md",
			"docs/review/bugs/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveReviewBase(tt.in); got != tt.want {
				t.Errorf("deriveReviewBase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestEntityPlaceholders_PopulatesReviewBase ensures the EntityPlaceholders
// shared builder injects review_base whenever file_path is present, so the
// agent-body renderer can substitute <review-base> via the dash-to-underscore
// alias path. Regression for B025.
func TestEntityPlaceholders_PopulatesReviewBase(t *testing.T) {
	filePath := "docs/plan/E07-feat/E07-F01-impl/tasks/T-E07-F01-001.md"
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E07-F01-001",
			Title:     "Test task",
			FilePath:  &filePath,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "ready_for_code_review",
		Priority: 5,
	}
	m := EntityPlaceholders(task)
	if m["review_base"] != "docs/review/E07-feat/E07-F01-impl/" {
		t.Errorf("review_base = %q, want docs/review/E07-feat/E07-F01-impl/", m["review_base"])
	}
}

// TestEntityPlaceholders_NoReviewBaseWithoutFilePath confirms review_base is
// omitted (not set to empty) when file_path is absent, so callers can detect
// "unknown" cleanly rather than rendering a stray bare slash.
func TestEntityPlaceholders_NoReviewBaseWithoutFilePath(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E07-F01-001",
			Title:     "Test task",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "draft",
		Priority: 5,
	}
	m := EntityPlaceholders(task)
	if _, ok := m["review_base"]; ok {
		t.Errorf("review_base should not be set when file_path is empty, got %q", m["review_base"])
	}
}

// TestDerivePlanDirs covers the file_path → epic_dir / feature_dir derivation
// that feeds the {{.epic_dir}} / {{.feature_dir}} placeholders consumed by the
// _resolve_spec_paths partial. Regression for B021: shark next was rendering
// docs/plan/E01/E01-F41/ (key-only) instead of the actual slug-suffixed
// docs/plan/E01-content-ingestion/E01-F41-lexicon-observation-staging.../
// because the partial concatenated EPIC_ID/FEATURE_ID (which are entity keys)
// instead of using the on-disk parent directories from file_path.
//
// Both helpers operate on file_path (the authoritative on-disk location) and
// preserve whatever convention that path uses — slug-suffixed or key-only.
func TestDerivePlanDirs(t *testing.T) {
	tests := []struct {
		name           string
		in             string
		wantEpicDir    string
		wantFeatureDir string
	}{
		{
			name:           "empty",
			in:             "",
			wantEpicDir:    "",
			wantFeatureDir: "",
		},
		{
			name:           "task under slug-suffixed feature",
			in:             "docs/plan/E01-content-ingestion/E01-F41-lexicon-observation/tasks/T-E01-F41-001.md",
			wantEpicDir:    "docs/plan/E01-content-ingestion",
			wantFeatureDir: "docs/plan/E01-content-ingestion/E01-F41-lexicon-observation",
		},
		{
			name:           "feature spec at slug-suffixed dir",
			in:             "docs/plan/E07-enhancements/E07-F40-file-logging/feature.md",
			wantEpicDir:    "docs/plan/E07-enhancements",
			wantFeatureDir: "docs/plan/E07-enhancements/E07-F40-file-logging",
		},
		{
			name:           "epic spec yields its own dir; feature_dir empty",
			in:             "docs/plan/E07-enhancements/epic.md",
			wantEpicDir:    "docs/plan/E07-enhancements",
			wantFeatureDir: "",
		},
		{
			name:           "key-only layout passes through unchanged",
			in:             "docs/plan/E07/E07-F02/feature.md",
			wantEpicDir:    "docs/plan/E07",
			wantFeatureDir: "docs/plan/E07/E07-F02",
		},
		{
			name:           "task under key-only feature",
			in:             "docs/plan/E07/E07-F02/tasks/T-E07-F02-001.md",
			wantEpicDir:    "docs/plan/E07",
			wantFeatureDir: "docs/plan/E07/E07-F02",
		},
		{
			name:           "leading ./ stripped",
			in:             "./docs/plan/E07-feat/E07-F01-impl/feature.md",
			wantEpicDir:    "docs/plan/E07-feat",
			wantFeatureDir: "docs/plan/E07-feat/E07-F01-impl",
		},
		{
			name:           "standalone bug under docs/plan/bugs",
			in:             "docs/plan/bugs/B021.md",
			wantEpicDir:    "docs/plan/bugs",
			wantFeatureDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveEpicDir(tt.in); got != tt.wantEpicDir {
				t.Errorf("deriveEpicDir(%q) = %q, want %q", tt.in, got, tt.wantEpicDir)
			}
			if got := deriveFeatureDir(tt.in); got != tt.wantFeatureDir {
				t.Errorf("deriveFeatureDir(%q) = %q, want %q", tt.in, got, tt.wantFeatureDir)
			}
		})
	}
}

// TestEntityPlaceholders_PopulatesPlanDirs ensures EntityPlaceholders emits
// epic_dir and feature_dir whenever file_path resolves to them, so the
// _resolve_spec_paths partial can render slug-suffixed paths instead of
// concatenating bare entity keys. Regression for B021.
func TestEntityPlaceholders_PopulatesPlanDirs(t *testing.T) {
	filePath := "docs/plan/E01-content-ingestion/E01-F41-lexicon-observation/tasks/T-E01-F41-001.md"
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E01-F41-001",
			Title:     "Test task",
			FilePath:  &filePath,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "in_development",
		Priority: 5,
	}
	m := EntityPlaceholders(task)
	wantEpic := "docs/plan/E01-content-ingestion"
	wantFeature := "docs/plan/E01-content-ingestion/E01-F41-lexicon-observation"
	if m["epic_dir"] != wantEpic {
		t.Errorf("epic_dir = %q, want %q", m["epic_dir"], wantEpic)
	}
	if m["feature_dir"] != wantFeature {
		t.Errorf("feature_dir = %q, want %q", m["feature_dir"], wantFeature)
	}
}

// TestEntityPlaceholders_NoPlanDirsWithoutFilePath confirms epic_dir and
// feature_dir are omitted (not set to empty) when file_path is absent, so the
// post-render guard in `shark next` can still distinguish "unknown" from
// "explicitly blank". Regression-companion for B021 mirroring the B025
// review_base behavior.
func TestEntityPlaceholders_NoPlanDirsWithoutFilePath(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E07-F01-001",
			Title:     "Test task",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "draft",
		Priority: 5,
	}
	m := EntityPlaceholders(task)
	if _, ok := m["epic_dir"]; ok {
		t.Errorf("epic_dir should not be set when file_path is empty, got %q", m["epic_dir"])
	}
	if _, ok := m["feature_dir"]; ok {
		t.Errorf("feature_dir should not be set when file_path is empty, got %q", m["feature_dir"])
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

	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F29-001",
		Title: "Test Task",

		ContextData: ptrString(`{"related_tasks":["E07-F05-001","E10-F05-002"]}`)}, Status: "todo",
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
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo, nil)

	if relDocs := result["related_docs"]; relDocs != "docs/a.md,docs/b.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md,docs/b.md")
	}

	if relTasks := result["related_tasks"]; relTasks != "E07-F05-001,E10-F05-002" {
		t.Errorf("related_tasks = %q, want %q", relTasks, "E07-F05-001,E10-F05-002")
	}
}

// TestTaskPlaceholdersWithRelated_NoData tests task placeholders with no data
func TestTaskPlaceholdersWithRelated_NoData(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F29-002",
		Title: "Empty Task"}, Status: "todo",
	}

	mockRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo, nil)

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
	result := TaskPlaceholdersWithRelated(ctx, nil, mockRepo, mockTaskRelRepo, nil)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestFeaturePlaceholdersWithRelated_HappyPath tests feature placeholders with docs and features
func TestFeaturePlaceholdersWithRelated_HappyPath(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F29",
		Title: "Test Feature"}, Status: "active",
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
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	if relDocs := result["related_docs"]; relDocs != "docs/a.md,docs/b.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md,docs/b.md")
	}

	if relFeatures := result["related_features"]; relFeatures != "E07-F05,E07-F21,E10-F05" {
		t.Errorf("related_features = %q, want %q", relFeatures, "E07-F05,E07-F21,E10-F05")
	}
}

// TestFeaturePlaceholdersWithRelated_NoData tests feature placeholders with no data
func TestFeaturePlaceholdersWithRelated_NoData(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F30",
		Title: "Empty Feature"}, Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

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
	result := FeaturePlaceholdersWithRelated(ctx, nil, mockDocRepo, mockRelRepo, nil)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relFeatures := result["related_features"]; relFeatures != "" {
		t.Errorf("related_features = %q, want empty string", relFeatures)
	}
}

// TestEpicPlaceholdersWithRelated_HappyPath tests epic placeholders with docs and epics
func TestEpicPlaceholdersWithRelated_HappyPath(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E07",
		Title: "Test Epic"}, Status: "active",
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
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx, nil)

	if relDocs := result["related_docs"]; relDocs != "docs/a.md,docs/b.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/a.md,docs/b.md")
	}

	if relEpics := result["related_epics"]; relEpics != "E01,E05" {
		t.Errorf("related_epics = %q, want %q", relEpics, "E01,E05")
	}
}

// TestEpicPlaceholdersWithRelated_NoData tests epic placeholders with no data
func TestEpicPlaceholdersWithRelated_NoData(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E08",
		Title: "Empty Epic"}, Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockEpicRelationshipRepository{
		epics: []string{},
	}

	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx, nil)

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
	result := EpicPlaceholdersWithRelated(nil, mockDocRepo, mockRelRepo, ctx, nil)

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
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F29-001",
		Title: "Test Task"}, Status: "todo",
	}

	mockRepo := &mockDocumentRepository{
		err: fmt.Errorf("database connection lost"),
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{"E07-F05-001"},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo, nil)

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
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F29-001",
		Title: "Test Task"}, Status: "todo",
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
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo, nil)

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
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F29-001",
		Title: "Test Task"}, Status: "todo",
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
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo, nil)

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
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F29",
		Title: "Test Feature"}, Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		err: fmt.Errorf("database connection lost"),
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{"E07-F05", "E07-F21"},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

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
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F29",
		Title: "Test Feature"}, Status: "active",
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
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

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
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F29",
		Title: "Test Feature"}, Status: "active",
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
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	// Should include cross-epic feature keys
	if relFeatures := result["related_features"]; relFeatures != "E01-F01,E07-F05,E10-F20" {
		t.Errorf("related_features with cross-epic = %q, want %q", relFeatures, "E01-F01,E07-F05,E10-F20")
	}
}

// TestEpicPlaceholdersWithRelated_DocRepoError tests epic placeholders when doc repo fails (TC-EPH-04)
func TestEpicPlaceholdersWithRelated_DocRepoError(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E07",
		Title: "Test Epic"}, Status: "active",
	}

	mockDocRepo := &mockDocumentRepository{
		err: fmt.Errorf("database connection lost"),
	}

	mockRelRepo := &mockEpicRelationshipRepository{
		epics: []string{"E01", "E05"},
	}

	ctx := context.Background()
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx, nil)

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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E07",
		Title: "Test Epic"}, Status: "active",
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
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx, nil)

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
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F29-001",
		Title: "Test Task",

		Slug:      &slug,
		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)}, Status: "in_progress",
		Priority: 7,
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
	result := TaskPlaceholdersWithRelated(ctx, task, mockRepo, mockTaskRelRepo, nil)

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
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F29",
		Title: "Test Feature",

		Slug: &slug}, Status: "active",
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
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	// Verify basic placeholders
	if result["id"] != "E07-F29" {
		t.Errorf("id placeholder missing or incorrect")
	}
	if result["key"] != "E07-F29" {
		t.Errorf("key placeholder missing or incorrect")
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
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E07",
		Title: "Test Epic",

		Slug: &slug}, Status: "active",
		Priority: models.PriorityHigh,
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
	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx, nil)

	// Verify basic placeholders
	if result["id"] != "E07" {
		t.Errorf("id placeholder missing or incorrect")
	}
	if result["key"] != "E07" {
		t.Errorf("key placeholder missing or incorrect")
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
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 100,
		Key:   "E07-F29-001",
		Title: "Test Task"}, Status: "todo",
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
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	if relDocs := result["related_docs"]; relDocs != "docs/spec.md,docs/design.md" {
		t.Errorf("related_docs = %q, want %q", relDocs, "docs/spec.md,docs/design.md")
	}

	if relTasks := result["related_tasks"]; relTasks != "E07-F05-001,E10-F05-002" {
		t.Errorf("related_tasks = %q, want %q", relTasks, "E07-F05-001,E10-F05-002")
	}
}

// TestTaskPlaceholdersWithRelated_Refactored_NoRelatedTasks tests empty array handling
func TestTaskPlaceholdersWithRelated_Refactored_NoRelatedTasks(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 200,
		Key:   "E07-F29-002",
		Title: "Test Task"}, Status: "todo",
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{}, // Empty array
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	if relDocs := result["related_docs"]; relDocs != "" {
		t.Errorf("related_docs = %q, want empty string", relDocs)
	}

	if relTasks := result["related_tasks"]; relTasks != "" {
		t.Errorf("related_tasks = %q, want empty string", relTasks)
	}
}

// TestTaskPlaceholdersWithRelated_Refactored_QueryError tests graceful degradation on repo error
func TestTaskPlaceholdersWithRelated_Refactored_QueryError(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 300,
		Key:   "E07-F29-003",
		Title: "Test Task"}, Status: "todo",
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
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

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
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F30-001",
		Title: "Test Task"}, Status: "todo",
		Metadata: map[string]interface{}{"complexity_tier": "STANDARD"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	if tier := result["complexity_tier"]; tier != "STANDARD" {
		t.Errorf("complexity_tier = %q, want %q", tier, "STANDARD")
	}

	// Verify existing placeholders still present
	if id := result["key"]; id != "E07-F30-001" {
		t.Errorf("key = %q, want %q", id, "E07-F30-001")
	}
	if title := result["title"]; title != "Test Task" {
		t.Errorf("title = %q, want %q", title, "Test Task")
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierMissing tests task without complexity_tier (API-29)
func TestTaskPlaceholdersWithRelated_ComplexityTierMissing(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F30-002",
		Title: "Task without tier"}, Status: "todo",
		Metadata: map[string]interface{}{}, // Empty metadata
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// Should have empty string for complexity_tier when not set
	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierNilMetadata tests task with nil metadata
func TestTaskPlaceholdersWithRelated_ComplexityTierNilMetadata(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F30-003",
		Title: "Task with nil metadata"}, Status: "todo",
		Metadata: nil,
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// Should have empty string for complexity_tier when metadata is nil
	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierSimple tests task with SIMPLE tier
func TestTaskPlaceholdersWithRelated_ComplexityTierSimple(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F30-004",
		Title: "Simple task"}, Status: "todo",
		Metadata: map[string]interface{}{"complexity_tier": "SIMPLE"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	if tier := result["complexity_tier"]; tier != "SIMPLE" {
		t.Errorf("complexity_tier = %q, want %q", tier, "SIMPLE")
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierComplex tests task with COMPLEX tier
func TestTaskPlaceholdersWithRelated_ComplexityTierComplex(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F30-005",
		Title: "Complex task"}, Status: "todo",
		Metadata: map[string]interface{}{"complexity_tier": "COMPLEX"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	if tier := result["complexity_tier"]; tier != "COMPLEX" {
		t.Errorf("complexity_tier = %q, want %q", tier, "COMPLEX")
	}
}

// TestTaskPlaceholdersWithRelated_ComplexityTierInvalidType tests task with non-string complexity_tier
func TestTaskPlaceholdersWithRelated_ComplexityTierInvalidType(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F30-006",
		Title: "Task with invalid tier type"}, Status: "todo",
		Metadata: map[string]interface{}{"complexity_tier": 123}, // Integer instead of string
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockTaskRelRepo := &mockTaskRelationshipRepository{
		tasks: []string{},
	}

	ctx := context.Background()
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

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
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F30",
		Title: "Template engine"}, Status: "active",
		Metadata: map[string]interface{}{"complexity_tier": "STANDARD"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	if tier := result["complexity_tier"]; tier != "STANDARD" {
		t.Errorf("complexity_tier = %q, want %q", tier, "STANDARD")
	}

	// Verify existing placeholders still present
	if id := result["key"]; id != "E07-F30" {
		t.Errorf("key = %q, want %q", id, "E07-F30")
	}
}

// TestFeaturePlaceholdersWithRelated_ComplexityTierMissing tests feature without complexity_tier
func TestFeaturePlaceholdersWithRelated_ComplexityTierMissing(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F31",
		Title: "Feature without tier"}, Status: "active",
		Metadata: map[string]interface{}{},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestFeaturePlaceholdersWithRelated_ComplexityTierNilMetadata tests feature with nil metadata
func TestFeaturePlaceholdersWithRelated_ComplexityTierNilMetadata(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F32",
		Title: "Feature with nil metadata"}, Status: "active",
		Metadata: nil,
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	if tier := result["complexity_tier"]; tier != "" {
		t.Errorf("complexity_tier = %q, want empty string", tier)
	}
}

// TestFeaturePlaceholdersWithRelated_ComplexityTierComplex tests feature with COMPLEX tier
func TestFeaturePlaceholdersWithRelated_ComplexityTierComplex(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F33",
		Title: "Complex feature"}, Status: "active",
		Metadata: map[string]interface{}{"complexity_tier": "COMPLEX"},
	}

	mockDocRepo := &mockDocumentRepository{
		docs: []*models.Document{},
	}

	mockRelRepo := &mockFeatureRelationshipRepository{
		features: []string{},
	}

	ctx := context.Background()
	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	if tier := result["complexity_tier"]; tier != "COMPLEX" {
		t.Errorf("complexity_tier = %q, want %q", tier, "COMPLEX")
	}
}

// ===================================================================
// Tests for key-parsing helpers (E07-F33 T1)
// ===================================================================

func TestParseEpicKeyFromEntityKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"task key with T- prefix", "T-E07-F01-001", "E07"},
		{"task key without T- prefix", "E07-F01-001", "E07"},
		{"feature key", "E07-F01", "E07"},
		{"epic key only", "E07", "E07"},
		{"lowercase input", "t-e07-f01-001", "E07"},
		{"double-digit epic", "E12-F03-005", "E12"},
		{"empty string", "", ""},
		{"malformed no epic", "F01-001", ""},
		{"malformed gibberish", "xyz", ""},
		// TD-021: locks in slugged-key + case-variant behavior via KeyService.Parse.
		{"slugged epic", "E07-user-management", "E07"},
		{"slugged feature", "E07-F01-auth-module", "E07"},
		{"slugged task", "T-E07-F01-001-impl-jwt", "E07"},
		{"slugged short task", "E07-F01-001-impl-jwt", "E07"},
		{"mixed-case slugged task", "t-E07-f01-001-Impl-JWT", "E07"},
		{"feature suffix only (no epic)", "F01", ""},
		{"slugged feature suffix only (no epic)", "F01-some-feature", ""},
		{"bug key (no epic component)", "B001", ""},
		{"change-card key (no epic component)", "C001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEpicKeyFromEntityKey(tt.input)
			if got != tt.expected {
				t.Errorf("parseEpicKeyFromEntityKey(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseFeatureKeyFromTaskKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"task key with T- prefix", "T-E07-F01-001", "E07-F01"},
		{"task key without T- prefix", "E07-F01-001", "E07-F01"},
		{"feature key (no task num)", "E07-F01", "E07-F01"},
		{"lowercase input", "t-e07-f01-001", "E07-F01"},
		{"double-digit feature", "E12-F03-005", "E12-F03"},
		{"empty string", "", ""},
		{"epic only", "E07", ""},
		{"malformed", "xyz-abc", ""},
		{"single segment", "E07", ""},
		// TD-021: locks in slugged-key + case-variant behavior via KeyService.Parse.
		{"slugged task with T- prefix", "T-E07-F01-001-impl-jwt", "E07-F01"},
		{"slugged short task", "E07-F01-001-impl-jwt", "E07-F01"},
		{"slugged feature", "E07-F01-auth-module", "E07-F01"},
		{"mixed-case slugged task", "t-E07-f01-001-Impl-JWT", "E07-F01"},
		{"slugged epic (no feature)", "E07-user-management", ""},
		{"feature suffix only (no epic)", "F01", ""},
		{"bug key (no feature component)", "B001", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFeatureKeyFromTaskKey(tt.input)
			if got != tt.expected {
				t.Errorf("parseFeatureKeyFromTaskKey(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ===================================================================
// Tests for canonical variable names (E07-F33 T2-T6)
// ===================================================================

// TestTaskPlaceholders_CanonicalKeys verifies new canonical keys and absence of removed keys
func TestTaskPlaceholders_CanonicalKeys(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F01-001",
		Title: "Test Task",

		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 16, 14, 45, 0, 0, time.UTC)}, Status: "todo",
		Priority: 5,
	}

	m := TaskPlaceholders(task)

	// New canonical keys must be present
	requiredKeys := map[string]string{
		"id":          "T-E07-F01-001",
		"key":         "T-E07-F01-001",
		"task_key":    "T-E07-F01-001",
		"epic_key":    "E07",
		"feature_key": "E07-F01",
	}
	for k, want := range requiredKeys {
		if got := m[k]; got != want {
			t.Errorf("TaskPlaceholders[%q] = %q, want %q", k, got, want)
		}
	}

	// Old keys preserved as backward-compatible aliases
	backwardCompat := map[string]string{
		"task_id":    "T-E07-F01-001",
		"epic_id":    "E07",
		"feature_id": "E07-F01",
	}
	for k, want := range backwardCompat {
		if got := m[k]; got != want {
			t.Errorf("TaskPlaceholders[%q] (backward compat) = %q, want %q", k, got, want)
		}
	}
}

// TestTaskPlaceholders_ShortKeyFormat verifies parsing works with short task key format
func TestTaskPlaceholders_ShortKeyFormat(t *testing.T) {
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F01-001",
		Title: "Test Task",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "todo",
		Priority: 5,
	}

	m := TaskPlaceholders(task)

	if m["epic_key"] != "E07" {
		t.Errorf("epic_key = %q, want %q (short task key format)", m["epic_key"], "E07")
	}
	if m["feature_key"] != "E07-F01" {
		t.Errorf("feature_key = %q, want %q (short task key format)", m["feature_key"], "E07-F01")
	}
}

// TestFeaturePlaceholders_CanonicalKeys verifies new canonical keys and absence of removed keys
func TestFeaturePlaceholders_CanonicalKeys(t *testing.T) {
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01",
		Title: "Test Feature",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "active",
	}

	m := FeaturePlaceholders(feature)

	// New canonical keys
	if m["key"] != "E07-F01" {
		t.Errorf("key = %q, want %q", m["key"], "E07-F01")
	}
	if m["epic_key"] != "E07" {
		t.Errorf("epic_key = %q, want %q", m["epic_key"], "E07")
	}
	if m["id"] != "E07-F01" {
		t.Errorf("id = %q, want %q", m["id"], "E07-F01")
	}

	// Backward-compatible aliases
	if m["feature_id"] != "E07-F01" {
		t.Errorf("feature_id (backward compat) = %q, want %q", m["feature_id"], "E07-F01")
	}
	if m["epic_id"] != "E07" {
		t.Errorf("epic_id (backward compat) = %q, want %q", m["epic_id"], "E07")
	}
}

// TestEpicPlaceholders_CanonicalKeys verifies new canonical keys and backward-compatible aliases
func TestEpicPlaceholders_CanonicalKeys(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E07",
		Title: "Test Epic",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "active",
		Priority: models.PriorityHigh,
	}

	m := EpicPlaceholders(epic)

	// New canonical keys
	if m["key"] != "E07" {
		t.Errorf("key = %q, want %q", m["key"], "E07")
	}
	if m["id"] != "E07" {
		t.Errorf("id = %q, want %q", m["id"], "E07")
	}

	// Backward-compatible alias
	if m["epic_id"] != "E07" {
		t.Errorf("epic_id (backward compat) = %q, want %q", m["epic_id"], "E07")
	}
}

// TestBugPlaceholders_ExpandedFields verifies all expanded fields on Bug
func TestBugPlaceholders_ExpandedFields(t *testing.T) {
	slug := "test-bug"
	description := "Bug description"
	filePath := "docs/bug.md"
	linkedType := "task"
	linkedKey := "E07-F01-001"

	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: "B001",
		Title: "Test Bug",

		Slug:        &slug,
		Description: &description,
		FilePath:    &filePath,

		CreatedAt: time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 3, 2, 11, 0, 0, 0, time.UTC)}, Status: "open",
		Severity: models.BugSeverityHigh,

		LinkedEntityType: &linkedType,
		LinkedEntityKey:  &linkedKey,
	}

	m := BugPlaceholders(bug)

	expected := map[string]string{
		"id":                 "B001",
		"key":                "B001",
		"title":              "Test Bug",
		"status":             "open",
		"severity":           "high",
		"slug":               "test-bug",
		"description":        "Bug description",
		"file_path":          "docs/bug.md",
		"linked_entity_type": "task",
		"linked_entity_key":  "E07-F01-001",
		"created_at":         "2025-03-01T10:00:00Z",
		"updated_at":         "2025-03-02T11:00:00Z",
	}

	for k, want := range expected {
		if got := m[k]; got != want {
			t.Errorf("BugPlaceholders[%q] = %q, want %q", k, got, want)
		}
	}
}

// TestBugPlaceholders_NilOptionalFields verifies nil pointer fields are omitted
func TestBugPlaceholders_NilOptionalFields(t *testing.T) {
	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: "B002",
		Title: "Minimal Bug",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "open",
		Severity: models.BugSeverityLow,
	}

	m := BugPlaceholders(bug)

	// Required fields present
	if m["key"] != "B002" {
		t.Errorf("key = %q, want %q", m["key"], "B002")
	}

	// Optional fields absent
	for _, k := range []string{"slug", "description", "file_path", "linked_entity_type", "linked_entity_key"} {
		if _, exists := m[k]; exists {
			t.Errorf("BugPlaceholders should not contain %q when nil", k)
		}
	}
}

// TestChangeCardPlaceholders_ExpandedFields verifies all expanded fields on ChangeCard
func TestChangeCardPlaceholders_ExpandedFields(t *testing.T) {
	description := "Change description"
	requestedBy := "alice"
	assignedTo := "bob"
	justification := "Needed for performance"
	impactAnalysis := "Low risk"
	rollbackPlan := "Revert commit"

	card := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001",
		Title: "Test Change",

		Slug:        ptrString("test-change"),
		FilePath:    ptrString("docs/change.md"),
		Description: &description,

		CreatedAt: time.Date(2025, 3, 5, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 3, 6, 10, 0, 0, 0, time.UTC)}, Status: "draft",
		Priority: 3,

		RequestedBy:    &requestedBy,
		AssignedTo:     &assignedTo,
		Justification:  &justification,
		ImpactAnalysis: &impactAnalysis,
		RollbackPlan:   &rollbackPlan,
	}

	m := ChangeCardPlaceholders(card)

	expected := map[string]string{
		"id":              "CC-001",
		"key":             "CC-001",
		"title":           "Test Change",
		"status":          "draft",
		"priority":        "3",
		"slug":            "test-change",
		"file_path":       "docs/change.md",
		"description":     "Change description",
		"requested_by":    "alice",
		"assigned_to":     "bob",
		"justification":   "Needed for performance",
		"impact_analysis": "Low risk",
		"rollback_plan":   "Revert commit",
		"created_at":      "2025-03-05T09:00:00Z",
		"updated_at":      "2025-03-06T10:00:00Z",
	}

	for k, want := range expected {
		if got := m[k]; got != want {
			t.Errorf("ChangeCardPlaceholders[%q] = %q, want %q", k, got, want)
		}
	}
}

// TestChangeCardPlaceholders_NilOptionalFields verifies nil pointer fields are omitted
func TestChangeCardPlaceholders_NilOptionalFields(t *testing.T) {
	card := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-002",
		Title: "Minimal Change",

		Slug:      ptrString("minimal-change"),
		FilePath:  ptrString("docs/change2.md"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "draft",
		Priority: 5,
	}

	m := ChangeCardPlaceholders(card)

	// Required fields present
	if m["key"] != "CC-002" {
		t.Errorf("key = %q, want %q", m["key"], "CC-002")
	}

	// Optional fields absent
	for _, k := range []string{"description", "requested_by", "assigned_to", "justification", "impact_analysis", "rollback_plan"} {
		if _, exists := m[k]; exists {
			t.Errorf("ChangeCardPlaceholders should not contain %q when nil", k)
		}
	}
}

// ===================================================================
// Tests for ContextData.Metadata extraction (E07-F30-004)
// ===================================================================

// TestStringifyMetadataValue verifies type conversion for metadata values
func TestStringifyMetadataValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string value", "STANDARD", "STANDARD"},
		{"empty string", "", ""},
		{"integer via float64 (JSON)", float64(42), "42"},
		{"float value", float64(3.14), "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int value", int(8), "8"},
		{"int64 value", int64(100), "100"},
		{"nil value", nil, ""},
		{"unsupported slice", []string{"a"}, ""},
		{"unsupported map", map[string]string{"a": "b"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringifyMetadataValue(tt.input)
			if got != tt.expected {
				t.Errorf("stringifyMetadataValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestExtractContextDataFields_StringField verifies string metadata extraction
func TestExtractContextDataFields_StringField(t *testing.T) {
	contextData := `{"metadata": {"complexity_tier": "STANDARD", "assignee": "john"}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&contextData, placeholders)

	if placeholders["complexity_tier"] != "STANDARD" {
		t.Errorf("complexity_tier = %q, want %q", placeholders["complexity_tier"], "STANDARD")
	}
	if placeholders["assignee"] != "john" {
		t.Errorf("assignee = %q, want %q", placeholders["assignee"], "john")
	}
}

// TestExtractContextDataFields_IntegerField verifies integer metadata extraction
func TestExtractContextDataFields_IntegerField(t *testing.T) {
	contextData := `{"metadata": {"estimated_hours": 8}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&contextData, placeholders)

	if placeholders["estimated_hours"] != "8" {
		t.Errorf("estimated_hours = %q, want %q", placeholders["estimated_hours"], "8")
	}
}

// TestExtractContextDataFields_FloatField verifies float metadata extraction
func TestExtractContextDataFields_FloatField(t *testing.T) {
	contextData := `{"metadata": {"velocity": 3.14}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&contextData, placeholders)

	if placeholders["velocity"] != "3.14" {
		t.Errorf("velocity = %q, want %q", placeholders["velocity"], "3.14")
	}
}

// TestExtractContextDataFields_BoolField verifies bool metadata extraction
func TestExtractContextDataFields_BoolField(t *testing.T) {
	contextData := `{"metadata": {"is_critical": true}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&contextData, placeholders)

	if placeholders["is_critical"] != "true" {
		t.Errorf("is_critical = %q, want %q", placeholders["is_critical"], "true")
	}
}

// TestExtractContextDataFields_NilContextData verifies nil handling
func TestExtractContextDataFields_NilContextData(t *testing.T) {
	placeholders := make(map[string]string)
	extractContextDataFields(nil, placeholders)

	if len(placeholders) != 0 {
		t.Errorf("expected 0 placeholders from nil context data, got %d", len(placeholders))
	}
}

// TestExtractContextDataFields_EmptyContextData verifies empty string handling
func TestExtractContextDataFields_EmptyContextData(t *testing.T) {
	empty := ""
	placeholders := make(map[string]string)
	extractContextDataFields(&empty, placeholders)

	if len(placeholders) != 0 {
		t.Errorf("expected 0 placeholders from empty context data, got %d", len(placeholders))
	}
}

// TestExtractContextDataFields_NoMetadataField verifies JSON without metadata field.
// With the extractContextDataFields extension, structured fields like progress are
// now extracted even without a metadata field.
func TestExtractContextDataFields_NoMetadataField(t *testing.T) {
	noMeta := `{"progress": {"current_step": "testing"}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&noMeta, placeholders)

	// The function now extracts structured fields too, so progress fields should appear
	if placeholders["current_step"] != "testing" {
		t.Errorf("expected current_step='testing', got %q", placeholders["current_step"])
	}
	if placeholders["completed_steps_count"] != "0" {
		t.Errorf("expected completed_steps_count='0', got %q", placeholders["completed_steps_count"])
	}
	if placeholders["remaining_steps_count"] != "0" {
		t.Errorf("expected remaining_steps_count='0', got %q", placeholders["remaining_steps_count"])
	}
}

// TestExtractContextDataFields_EmptyMetadata verifies empty metadata map
func TestExtractContextDataFields_EmptyMetadata(t *testing.T) {
	emptyMeta := `{"metadata": {}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&emptyMeta, placeholders)

	if len(placeholders) != 0 {
		t.Errorf("expected 0 placeholders from empty metadata, got %d", len(placeholders))
	}
}

// TestTaskPlaceholdersWithRelated_ContextDataMetadata verifies task placeholder extraction
// with metadata from ContextData (the primary path introduced by T-E07-F30-004)
func TestTaskPlaceholdersWithRelated_ContextDataMetadata(t *testing.T) {
	ctx := context.Background()

	contextData := `{"metadata": {"complexity_tier": "STANDARD", "custom_field": "value123", "estimated_hours": 8, "is_critical": true}}`
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F30-001",
		Title: "Test Task",

		ContextData: &contextData,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now()}, Status: "in_development",
		Priority: 5,
	}

	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}

	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// Verify ContextData.Metadata fields are available
	if result["complexity_tier"] != "STANDARD" {
		t.Errorf("complexity_tier = %q, want %q", result["complexity_tier"], "STANDARD")
	}
	if result["custom_field"] != "value123" {
		t.Errorf("custom_field = %q, want %q", result["custom_field"], "value123")
	}
	if result["estimated_hours"] != "8" {
		t.Errorf("estimated_hours = %q, want %q", result["estimated_hours"], "8")
	}
	if result["is_critical"] != "true" {
		t.Errorf("is_critical = %q, want %q", result["is_critical"], "true")
	}

	// Verify existing placeholders are still present
	if result["task_key"] != "T-E07-F30-001" {
		t.Errorf("task_key = %q, want %q", result["task_key"], "T-E07-F30-001")
	}
}

// TestTaskPlaceholdersWithRelated_NilContextData verifies no crash with nil ContextData
func TestTaskPlaceholdersWithRelated_NilContextData(t *testing.T) {
	ctx := context.Background()

	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F30-001",
		Title: "Test Task",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "todo",
		Priority: 5,
	}

	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}

	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// Should not crash, existing placeholders still work
	if result["task_key"] != "T-E07-F30-001" {
		t.Errorf("task_key = %q, want %q", result["task_key"], "T-E07-F30-001")
	}
	// complexity_tier defaults to empty
	if result["complexity_tier"] != "" {
		t.Errorf("complexity_tier = %q, want empty string", result["complexity_tier"])
	}
}

// TestFeaturePlaceholdersWithRelated_ContextDataMetadata verifies feature metadata extraction
func TestFeaturePlaceholdersWithRelated_ContextDataMetadata(t *testing.T) {
	ctx := context.Background()

	contextData := `{"metadata": {"complexity_tier": "COMPLEX", "owner": "team-alpha"}}`
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "E07-F30",
		Title: "Template Engine",

		ContextData: &contextData,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now()}, Status: "active",
	}

	// mockDocumentRepository implements ListForFeature via the same mock
	mockDocRepo := &mockDocumentRepository{}
	mockRelRepo := &mockFeatureRelationshipRepository{}

	result := FeaturePlaceholdersWithRelated(ctx, feature, mockDocRepo, mockRelRepo, nil)

	if result["complexity_tier"] != "COMPLEX" {
		t.Errorf("complexity_tier = %q, want %q", result["complexity_tier"], "COMPLEX")
	}
	if result["owner"] != "team-alpha" {
		t.Errorf("owner = %q, want %q", result["owner"], "team-alpha")
	}
}

// TestEpicPlaceholdersWithRelated_ContextDataMetadata verifies epic metadata extraction
func TestEpicPlaceholdersWithRelated_ContextDataMetadata(t *testing.T) {
	ctx := context.Background()

	contextData := `{"metadata": {"phase": "development", "budget": 50000}}`
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "E07",
		Title: "Enhancements",

		ContextData: &contextData,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now()}, Status: "active",
		Priority: models.PriorityHigh,
	}

	// mockDocumentRepository implements ListForEpic via the same mock
	mockDocRepo := &mockDocumentRepository{}
	mockRelRepo := &mockEpicRelationshipRepository{}

	result := EpicPlaceholdersWithRelated(epic, mockDocRepo, mockRelRepo, ctx, nil)

	if result["phase"] != "development" {
		t.Errorf("phase = %q, want %q", result["phase"], "development")
	}
	if result["budget"] != "50000" {
		t.Errorf("budget = %q, want %q", result["budget"], "50000")
	}
}

// TestContextDataMetadata_BackwardCompatFallback verifies that entity Metadata
// is used as fallback when ContextData.Metadata doesn't have the field
func TestContextDataMetadata_BackwardCompatFallback(t *testing.T) {
	ctx := context.Background()

	// Task with entity Metadata but no ContextData
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F30-001",
		Title: "Test Task",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "todo",
		Priority: 5,
		Metadata: map[string]interface{}{"complexity_tier": "SIMPLE"},
	}

	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}

	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// Should fall back to entity Metadata
	if result["complexity_tier"] != "SIMPLE" {
		t.Errorf("complexity_tier = %q, want %q (from entity Metadata fallback)", result["complexity_tier"], "SIMPLE")
	}
}

// TestContextDataMetadata_ContextDataTakesPrecedence verifies ContextData.Metadata
// takes precedence over entity Metadata for the same field
func TestContextDataMetadata_ContextDataTakesPrecedence(t *testing.T) {
	ctx := context.Background()

	contextData := `{"metadata": {"complexity_tier": "COMPLEX"}}`
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F30-001",
		Title: "Test Task",

		// Entity metadata says SIMPLE
		ContextData: &contextData, // ContextData says COMPLEX
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now()}, Status: "todo",
		Priority: 5,
		Metadata: map[string]interface{}{"complexity_tier": "SIMPLE"},
	}

	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}

	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// ContextData.Metadata should win
	if result["complexity_tier"] != "COMPLEX" {
		t.Errorf("complexity_tier = %q, want %q (ContextData should take precedence)", result["complexity_tier"], "COMPLEX")
	}
}

// ============================================================================
// Tests for extractContextDataFields structured field extraction (E07-F34-001)
// ============================================================================

// TestExtractContextDataFields_ProgressAllFields verifies extraction of all progress fields (TC-CD-01)
func TestExtractContextDataFields_ProgressAllFields(t *testing.T) {
	cd := `{"progress":{"current_step":"Implementing API","completed_steps":["Design","DB Schema"],"remaining_steps":["Tests","Review"]}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	if placeholders["current_step"] != "Implementing API" {
		t.Errorf("current_step = %q, want %q", placeholders["current_step"], "Implementing API")
	}
	if placeholders["completed_steps"] != "Design, DB Schema" {
		t.Errorf("completed_steps = %q, want %q", placeholders["completed_steps"], "Design, DB Schema")
	}
	if placeholders["remaining_steps"] != "Tests, Review" {
		t.Errorf("remaining_steps = %q, want %q", placeholders["remaining_steps"], "Tests, Review")
	}
	if placeholders["completed_steps_count"] != "2" {
		t.Errorf("completed_steps_count = %q, want %q", placeholders["completed_steps_count"], "2")
	}
	if placeholders["remaining_steps_count"] != "2" {
		t.Errorf("remaining_steps_count = %q, want %q", placeholders["remaining_steps_count"], "2")
	}
}

// TestExtractContextDataFields_ProgressPartial verifies partial progress (TC-CD-02)
func TestExtractContextDataFields_ProgressPartial(t *testing.T) {
	cd := `{"progress":{"current_step":"Writing tests"}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	if placeholders["current_step"] != "Writing tests" {
		t.Errorf("current_step = %q, want %q", placeholders["current_step"], "Writing tests")
	}
	if placeholders["completed_steps_count"] != "0" {
		t.Errorf("completed_steps_count = %q, want %q", placeholders["completed_steps_count"], "0")
	}
	if placeholders["remaining_steps_count"] != "0" {
		t.Errorf("remaining_steps_count = %q, want %q", placeholders["remaining_steps_count"], "0")
	}
}

// TestExtractContextDataFields_NoProgress verifies no progress field (TC-CD-03)
func TestExtractContextDataFields_NoProgress(t *testing.T) {
	cd := `{"metadata":{"foo":"bar"}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	if placeholders["foo"] != "bar" {
		t.Errorf("foo = %q, want %q", placeholders["foo"], "bar")
	}
	// current_step should not be set when progress is nil
	if _, exists := placeholders["current_step"]; exists {
		t.Errorf("current_step should not exist when progress is nil")
	}
}

// TestExtractContextDataFields_OpenQuestions verifies open question extraction (TC-CD-04)
func TestExtractContextDataFields_OpenQuestions(t *testing.T) {
	cd := `{"open_questions":["Auth provider?","Rate limiting?"]}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	if placeholders["open_questions"] != "Auth provider?; Rate limiting?" {
		t.Errorf("open_questions = %q, want %q", placeholders["open_questions"], "Auth provider?; Rate limiting?")
	}
	if placeholders["open_questions_count"] != "2" {
		t.Errorf("open_questions_count = %q, want %q", placeholders["open_questions_count"], "2")
	}
}

// TestExtractContextDataFields_OpenQuestionsEmpty verifies empty open questions (TC-CD-05)
func TestExtractContextDataFields_OpenQuestionsEmpty(t *testing.T) {
	cd := `{"open_questions":[]}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	if _, exists := placeholders["open_questions"]; exists {
		t.Errorf("open_questions should not exist for empty slice")
	}
}

// TestExtractContextDataFields_Blockers verifies blocker extraction (TC-CD-06, TC-CD-07)
func TestExtractContextDataFields_Blockers(t *testing.T) {
	cd := `{"blockers":[{"description":"First","blocker_type":"internal","blocked_since":"2026-03-01T00:00:00Z"},{"description":"Second","blocker_type":"external","blocked_since":"2026-03-02T00:00:00Z"}]}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	if placeholders["blockers_count"] != "2" {
		t.Errorf("blockers_count = %q, want %q", placeholders["blockers_count"], "2")
	}
	if placeholders["latest_blocker"] != "Second" {
		t.Errorf("latest_blocker = %q, want %q (should be last element)", placeholders["latest_blocker"], "Second")
	}
}

// TestExtractContextDataFields_Decisions verifies implementation decisions count (TC-CD-08)
func TestExtractContextDataFields_Decisions(t *testing.T) {
	cd := `{"implementation_decisions":{"auth":"JWT","db":"PostgreSQL","cache":"Redis"}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	if placeholders["decisions_count"] != "3" {
		t.Errorf("decisions_count = %q, want %q", placeholders["decisions_count"], "3")
	}
}

// TestExtractContextDataFields_EmptyAndNil verifies nil/empty safety (TC-CD-09, TC-CD-10)
func TestExtractContextDataFields_EmptyAndNil(t *testing.T) {
	// nil pointer
	placeholders := make(map[string]string)
	extractContextDataFields(nil, placeholders)
	if len(placeholders) != 0 {
		t.Errorf("expected 0 placeholders for nil, got %d", len(placeholders))
	}

	// empty string
	empty := ""
	placeholders = make(map[string]string)
	extractContextDataFields(&empty, placeholders)
	if len(placeholders) != 0 {
		t.Errorf("expected 0 placeholders for empty, got %d", len(placeholders))
	}
}

// TestExtractContextDataFields_MalformedJSON verifies graceful skip (TC-CD-11)
func TestExtractContextDataFields_MalformedJSON(t *testing.T) {
	bad := "not json"
	placeholders := make(map[string]string)
	extractContextDataFields(&bad, placeholders)
	if len(placeholders) != 0 {
		t.Errorf("expected 0 placeholders for malformed JSON, got %d", len(placeholders))
	}
}

// TestExtractContextDataFields_MetadataCollision verifies metadata vs structured field precedence (TC-CD-12)
func TestExtractContextDataFields_MetadataCollision(t *testing.T) {
	cd := `{"metadata":{"current_step":"from-metadata"},"progress":{"current_step":"from-progress"}}`
	placeholders := make(map[string]string)
	extractContextDataFields(&cd, placeholders)

	// Metadata runs first, structured fields should NOT overwrite
	if placeholders["current_step"] != "from-metadata" {
		t.Errorf("current_step = %q, want %q (metadata should take precedence)", placeholders["current_step"], "from-metadata")
	}
}

// ============================================================================
// Tests for ApplyEnrichmentData (E07-F34-002/003)
// ============================================================================

// TestApplyEnrichmentData_AllFieldsPopulated verifies all enrichment fields
func TestApplyEnrichmentData_AllFieldsPopulated(t *testing.T) {
	enrichment := &TemplateEnrichmentData{
		PreviousStatus:    "in_code_review",
		ParentTitle:       "User Auth",
		GrandparentTitle:  "Security",
		LatestNoteContent: "Review feedback addressed",
		LatestNoteType:    "comment",
		NotesCount:        5,
		RejectionCount:    1,
		SiblingTotal:      12,
		SiblingCompleted:  8,
		SiblingBlocked:    1,
	}
	placeholders := make(map[string]string)
	ApplyEnrichmentData(enrichment, placeholders)

	checks := map[string]string{
		"previous_status":   "in_code_review",
		"parent_title":      "User Auth",
		"grandparent_title": "Security",
		"latest_note":       "Review feedback addressed",
		"latest_note_type":  "comment",
		"notes_count":       "5",
		"rejection_count":   "1",
		"sibling_total":     "12",
		"sibling_completed": "8",
		"sibling_blocked":   "1",
	}
	for key, want := range checks {
		if placeholders[key] != want {
			t.Errorf("%s = %q, want %q", key, placeholders[key], want)
		}
	}
}

// TestApplyEnrichmentData_NilEnrichment verifies nil safety
func TestApplyEnrichmentData_NilEnrichment(t *testing.T) {
	placeholders := map[string]string{"existing": "value"}
	ApplyEnrichmentData(nil, placeholders)
	if len(placeholders) != 1 {
		t.Errorf("expected 1 placeholder unchanged, got %d", len(placeholders))
	}
	if placeholders["existing"] != "value" {
		t.Errorf("existing placeholder modified")
	}
}

// TestApplyEnrichmentData_ZeroValues verifies zero-valued enrichment
func TestApplyEnrichmentData_ZeroValues(t *testing.T) {
	enrichment := &TemplateEnrichmentData{}
	placeholders := make(map[string]string)
	ApplyEnrichmentData(enrichment, placeholders)

	// Counts should render as "0"
	if placeholders["sibling_total"] != "0" {
		t.Errorf("sibling_total = %q, want %q", placeholders["sibling_total"], "0")
	}
	if placeholders["notes_count"] != "0" {
		t.Errorf("notes_count = %q, want %q", placeholders["notes_count"], "0")
	}
	// Empty strings should NOT create keys
	if _, exists := placeholders["previous_status"]; exists {
		t.Errorf("previous_status should not exist for empty string")
	}
	if _, exists := placeholders["parent_title"]; exists {
		t.Errorf("parent_title should not exist for empty string")
	}
}

// TestApplyEnrichmentData_PartialData verifies partial enrichment
func TestApplyEnrichmentData_PartialData(t *testing.T) {
	enrichment := &TemplateEnrichmentData{
		PreviousStatus:   "todo",
		SiblingTotal:     5,
		SiblingCompleted: 3,
	}
	placeholders := make(map[string]string)
	ApplyEnrichmentData(enrichment, placeholders)

	if placeholders["previous_status"] != "todo" {
		t.Errorf("previous_status = %q, want %q", placeholders["previous_status"], "todo")
	}
	if placeholders["sibling_total"] != "5" {
		t.Errorf("sibling_total = %q, want %q", placeholders["sibling_total"], "5")
	}
	// latest_note should not be set
	if _, exists := placeholders["latest_note"]; exists {
		t.Errorf("latest_note should not exist when LatestNoteContent is empty")
	}
}

// TestTaskPlaceholdersWithRelated_NilEnrichment verifies backward compatibility
func TestTaskPlaceholdersWithRelated_NilEnrichment_BackwardCompat(t *testing.T) {
	ctx := context.Background()
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F01-001",
		Title: "Test Task",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "todo",
		Priority: 5,
	}
	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}

	// Call with nil enrichment - should produce same output as basic + relationship placeholders
	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// Basic placeholders should still be present
	if result["key"] != "T-E07-F01-001" {
		t.Errorf("key = %q, want %q", result["key"], "T-E07-F01-001")
	}
	if result["title"] != "Test Task" {
		t.Errorf("title = %q, want %q", result["title"], "Test Task")
	}
	// Enrichment placeholders should NOT be present
	if _, exists := result["previous_status"]; exists {
		t.Errorf("previous_status should not exist with nil enrichment")
	}
	if _, exists := result["parent_title"]; exists {
		t.Errorf("parent_title should not exist with nil enrichment")
	}
}

// TestTaskPlaceholdersWithRelated_WithEnrichment verifies enrichment integration
func TestTaskPlaceholdersWithRelated_WithEnrichment(t *testing.T) {
	ctx := context.Background()
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E07-F01-001",
		Title: "Test Task",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: "todo",
		Priority: 5,
	}
	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}

	enrichment := &TemplateEnrichmentData{
		PreviousStatus:   "in_development",
		ParentTitle:      "Feature One",
		GrandparentTitle: "Epic One",
		SiblingTotal:     10,
		SiblingCompleted: 7,
	}

	result := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, enrichment)

	if result["previous_status"] != "in_development" {
		t.Errorf("previous_status = %q, want %q", result["previous_status"], "in_development")
	}
	if result["parent_title"] != "Feature One" {
		t.Errorf("parent_title = %q, want %q", result["parent_title"], "Feature One")
	}
	if result["grandparent_title"] != "Epic One" {
		t.Errorf("grandparent_title = %q, want %q", result["grandparent_title"], "Epic One")
	}
	if result["sibling_total"] != "10" {
		t.Errorf("sibling_total = %q, want %q", result["sibling_total"], "10")
	}
}

// TestEntityPlaceholders_WithTask verifies EntityPlaceholders extracts shared fields from a Task.
func TestEntityPlaceholders_WithTask(t *testing.T) {
	slug := "my-task"
	desc := "A description"
	fp := "docs/task.md"
	task := &models.Task{BaseEntity: models.BaseEntity{ID: 42,
		Key:         "T-E07-F01-001",
		Title:       "My Task",
		Slug:        &slug,
		Description: &desc,

		FilePath:  &fp,
		CreatedAt: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC)}, Status: "in_progress",
	}

	m := EntityPlaceholders(task)

	checks := map[string]string{
		"id":          "T-E07-F01-001",
		"key":         "T-E07-F01-001",
		"entity_type": "task",
		"title":       "My Task",
		"status":      "in_progress",
		"slug":        "my-task",
		"description": "A description",
		"file_path":   "docs/task.md",
		"created_at":  "2025-06-01T10:00:00Z",
		"updated_at":  "2025-06-02T12:00:00Z",
	}
	for k, want := range checks {
		if got := m[k]; got != want {
			t.Errorf("EntityPlaceholders(task)[%q] = %q, want %q", k, got, want)
		}
	}
}

// TestEntityPlaceholders_WithEpic verifies EntityPlaceholders extracts shared fields from an Epic.
func TestEntityPlaceholders_WithEpic(t *testing.T) {
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "E07",
		Title: "My Epic",

		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)}, Status: "active",
	}

	m := EntityPlaceholders(epic)

	if m["entity_type"] != "epic" {
		t.Errorf("entity_type = %q, want %q", m["entity_type"], "epic")
	}
	if m["key"] != "E07" {
		t.Errorf("key = %q, want %q", m["key"], "E07")
	}
	if m["title"] != "My Epic" {
		t.Errorf("title = %q, want %q", m["title"], "My Epic")
	}
	// slug should not be present when nil
	if _, ok := m["slug"]; ok {
		t.Error("slug should not be present when nil")
	}
	// description should not be present when nil
	if _, ok := m["description"]; ok {
		t.Error("description should not be present when nil")
	}
	// file_path should not be present when nil
	if _, ok := m["file_path"]; ok {
		t.Error("file_path should not be present when nil")
	}
}

// TestEntityPlaceholders_WithBug verifies EntityPlaceholders extracts shared fields from a Bug.
func TestEntityPlaceholders_WithBug(t *testing.T) {
	slug := "crash-on-login"
	bug := &models.Bug{BaseEntity: models.BaseEntity{ID: 10,
		Key:   "B001",
		Title: "Crash on Login",
		Slug:  &slug,

		CreatedAt: time.Date(2025, 3, 15, 8, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 3, 15, 9, 0, 0, 0, time.UTC)}, Status: "open",
		Severity: "critical",
	}

	m := EntityPlaceholders(bug)

	if m["entity_type"] != "bug" {
		t.Errorf("entity_type = %q, want %q", m["entity_type"], "bug")
	}
	if m["key"] != "B001" {
		t.Errorf("key = %q, want %q", m["key"], "B001")
	}
	if m["slug"] != "crash-on-login" {
		t.Errorf("slug = %q, want %q", m["slug"], "crash-on-login")
	}
	// Bug-specific fields like severity should NOT be in base placeholders
	if _, ok := m["severity"]; ok {
		t.Error("severity should not be in base EntityPlaceholders")
	}
}

// TestEntityPlaceholders_WithChangeCard verifies EntityPlaceholders with a ChangeCard.
func TestEntityPlaceholders_WithChangeCard(t *testing.T) {
	card := &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 5,
		Key:   "CC-001",
		Title: "Migrate DB",

		CreatedAt: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)}, Status: "draft",
		Priority: 3,
	}

	m := EntityPlaceholders(card)

	if m["entity_type"] != "change" {
		t.Errorf("entity_type = %q, want %q", m["entity_type"], "change")
	}
	if m["key"] != "CC-001" {
		t.Errorf("key = %q, want %q", m["key"], "CC-001")
	}
	// priority should NOT be in base (it's entity-specific formatting)
	if _, ok := m["priority"]; ok {
		t.Error("priority should not be in base EntityPlaceholders")
	}
}

// TestEntityPlaceholders_NilEntity verifies EntityPlaceholders returns empty map for nil.
func TestEntityPlaceholders_NilEntity(t *testing.T) {
	m := EntityPlaceholders(nil)
	if len(m) != 0 {
		t.Errorf("EntityPlaceholders(nil) returned %d entries, want 0", len(m))
	}
}

// TestEntityPlaceholders_OptionalFieldsOmittedWhenEmpty verifies that slug, description,
// and file_path are omitted when the Entity returns empty strings for them.
func TestEntityPlaceholders_OptionalFieldsOmittedWhenEmpty(t *testing.T) {
	// Feature with no optional pointer fields set
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: 2,
		Key:   "E01-F01",
		Title: "Feature",

		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, Status: "todo",
	}

	m := EntityPlaceholders(feature)

	// These should NOT be present since the pointer fields are nil
	for _, key := range []string{"slug", "description", "file_path"} {
		if _, ok := m[key]; ok {
			t.Errorf("%q should not be present when the underlying field is nil", key)
		}
	}

	// These should always be present
	for _, key := range []string{"id", "key", "entity_type", "title", "status", "created_at", "updated_at"} {
		if _, ok := m[key]; !ok {
			t.Errorf("%q should always be present in EntityPlaceholders output", key)
		}
	}
}

// TestTechDebtPlaceholders_ExpandedFields verifies all expanded fields on TechDebt
func TestTechDebtPlaceholders_ExpandedFields(t *testing.T) {
	slug := "legacy-api-cleanup"
	description := "Remove deprecated v1 API endpoints"
	filePath := "docs/plan/tech_debt/TD-001.md"
	effortEstimate := "m"

	td := &models.TechDebt{
		BaseEntity: models.BaseEntity{
			Key:         "TD-001",
			Title:       "Legacy API Cleanup",
			Slug:        &slug,
			Description: &description,
			FilePath:    &filePath,
			CreatedAt:   time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC),
		},
		Status:         "identified",
		Category:       models.TechDebtCategoryCodeQuality,
		Severity:       models.TechDebtSeverityHigh,
		EffortEstimate: &effortEstimate,
	}

	m := TechDebtPlaceholders(td)

	expected := map[string]string{
		"id":              "TD-001",
		"key":             "TD-001",
		"entity_type":     "tech_debt",
		"title":           "Legacy API Cleanup",
		"status":          "identified",
		"category":        "code-quality",
		"severity":        "high",
		"effort_estimate": "m",
		"slug":            "legacy-api-cleanup",
		"description":     "Remove deprecated v1 API endpoints",
		"file_path":       "docs/plan/tech_debt/TD-001.md",
		"created_at":      "2026-04-01T10:00:00Z",
		"updated_at":      "2026-04-02T11:00:00Z",
	}

	for k, want := range expected {
		if got := m[k]; got != want {
			t.Errorf("TechDebtPlaceholders[%q] = %q, want %q", k, got, want)
		}
	}
}

// TestTechDebtPlaceholders_NilOptionalFields verifies nil pointer fields are omitted
func TestTechDebtPlaceholders_NilOptionalFields(t *testing.T) {
	td := &models.TechDebt{
		BaseEntity: models.BaseEntity{
			Key:       "TD-002",
			Title:     "Minimal Tech Debt",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "triaged",
		Category: models.TechDebtCategoryArchitecture,
		Severity: models.TechDebtSeverityLow,
	}

	m := TechDebtPlaceholders(td)

	// Required fields present
	if m["key"] != "TD-002" {
		t.Errorf("key = %q, want %q", m["key"], "TD-002")
	}
	if m["category"] != "architecture" {
		t.Errorf("category = %q, want %q", m["category"], "architecture")
	}
	if m["severity"] != "low" {
		t.Errorf("severity = %q, want %q", m["severity"], "low")
	}

	// Optional fields absent
	for _, k := range []string{"slug", "description", "file_path", "effort_estimate"} {
		if _, exists := m[k]; exists {
			t.Errorf("TechDebtPlaceholders should not contain %q when nil", k)
		}
	}
}

// TestTechDebtPlaceholders_NilEntity verifies nil entity returns empty map
func TestTechDebtPlaceholders_NilEntity(t *testing.T) {
	m := TechDebtPlaceholders(nil)
	if len(m) != 0 {
		t.Errorf("TechDebtPlaceholders(nil) returned %d entries, want 0", len(m))
	}
}

// ============================================================================
// E07-F42 — Size placeholder tests (TC-F011-A, TC-F011-B, TC-F011-C,
//           TC-F012-A, TC-F012-B, IS-4)
// ============================================================================

// ptrInt returns a pointer to an int value (test helper for Size fields).
func ptrInt(n int) *int { return &n }

// TC-F011-C: applySizePlaceholders helper — tested in isolation.
func TestApplySizePlaceholders_NonNilSize(t *testing.T) {
	tests := []struct {
		size      int
		wantSize  string
		wantLabel string
	}{
		{1, "1", "XS"},
		{2, "2", "S"},
		{3, "3", "M"},
		{5, "5", "L"},
		{8, "8", "XL"},
		{13, "13", "XXL"},
	}
	for _, tt := range tests {
		t.Run(tt.wantLabel, func(t *testing.T) {
			m := make(map[string]string)
			applySizePlaceholders(ptrInt(tt.size), m)
			if got := m["size"]; got != tt.wantSize {
				t.Errorf("applySizePlaceholders(%d): size = %q, want %q", tt.size, got, tt.wantSize)
			}
			if got := m["size_label"]; got != tt.wantLabel {
				t.Errorf("applySizePlaceholders(%d): size_label = %q, want %q", tt.size, got, tt.wantLabel)
			}
		})
	}
}

// TC-F011-C: applySizePlaceholders helper — nil produces empty strings.
func TestApplySizePlaceholders_NilSize(t *testing.T) {
	m := make(map[string]string)
	applySizePlaceholders(nil, m)
	if got := m["size"]; got != "" {
		t.Errorf("applySizePlaceholders(nil): size = %q, want empty string", got)
	}
	if got := m["size_label"]; got != "" {
		t.Errorf("applySizePlaceholders(nil): size_label = %q, want empty string", got)
	}
}

// TC-F011-C: applySizePlaceholders — both keys always written (not just absent when nil).
func TestApplySizePlaceholders_NilSizeKeysExplicitlySet(t *testing.T) {
	// Ensure that after calling applySizePlaceholders(nil, m), the keys
	// exist in the map with empty-string values (no "<nil>" rendered).
	m := make(map[string]string)
	applySizePlaceholders(nil, m)
	if _, exists := m["size"]; !exists {
		t.Error("applySizePlaceholders(nil): key 'size' must be present (empty string), got absent")
	}
	if _, exists := m["size_label"]; !exists {
		t.Error("applySizePlaceholders(nil): key 'size_label' must be present (empty string), got absent")
	}
}

// TC-F011-A: TaskPlaceholders populates size and size_label for a task with Size set.
func TestTaskPlaceholders_SizePopulated(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E07-F01-001",
			Title:     "Sized Task",
			Size:      ptrInt(5),
			CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, 1, 16, 14, 45, 0, 0, time.UTC),
		},
		Status:   "todo",
		Priority: 5,
	}

	m := TaskPlaceholders(task)

	if got := m["size"]; got != "5" {
		t.Errorf("TaskPlaceholders size = %q, want %q", got, "5")
	}
	if got := m["size_label"]; got != "L" {
		t.Errorf("TaskPlaceholders size_label = %q, want %q", got, "L")
	}
}

// TC-F011-B: TaskPlaceholders produces empty-string placeholders when Size is nil.
func TestTaskPlaceholders_SizeNil(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E07-F01-002",
			Title:     "Unsized Task",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "todo",
		Priority: 3,
	}

	m := TaskPlaceholders(task)

	if got := m["size"]; got != "" {
		t.Errorf("TaskPlaceholders size (nil) = %q, want empty string", got)
	}
	if got := m["size_label"]; got != "" {
		t.Errorf("TaskPlaceholders size_label (nil) = %q, want empty string", got)
	}
}

// TC-F011-A: FeaturePlaceholders populates size and size_label for a feature with Size set.
func TestFeaturePlaceholders_SizePopulated(t *testing.T) {
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			Key:       "E07-F01",
			Title:     "Sized Feature",
			Size:      ptrInt(8),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status: "active",
	}

	m := FeaturePlaceholders(feature)

	if got := m["size"]; got != "8" {
		t.Errorf("FeaturePlaceholders size = %q, want %q", got, "8")
	}
	if got := m["size_label"]; got != "XL" {
		t.Errorf("FeaturePlaceholders size_label = %q, want %q", got, "XL")
	}
}

// TC-F011-B: FeaturePlaceholders produces empty-string placeholders when Size is nil.
func TestFeaturePlaceholders_SizeNil(t *testing.T) {
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{
			Key:       "E07-F02",
			Title:     "Unsized Feature",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status: "active",
	}

	m := FeaturePlaceholders(feature)

	if got := m["size"]; got != "" {
		t.Errorf("FeaturePlaceholders size (nil) = %q, want empty string", got)
	}
	if got := m["size_label"]; got != "" {
		t.Errorf("FeaturePlaceholders size_label (nil) = %q, want empty string", got)
	}
}

// TC-F011-A: EpicPlaceholders populates size and size_label for an epic with Size set.
func TestEpicPlaceholders_SizePopulated(t *testing.T) {
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:       "E07",
			Title:     "Sized Epic",
			Size:      ptrInt(13),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "active",
		Priority: models.PriorityHigh,
	}

	m := EpicPlaceholders(epic)

	if got := m["size"]; got != "13" {
		t.Errorf("EpicPlaceholders size = %q, want %q", got, "13")
	}
	if got := m["size_label"]; got != "XXL" {
		t.Errorf("EpicPlaceholders size_label = %q, want %q", got, "XXL")
	}
}

// TC-F011-B: EpicPlaceholders produces empty-string placeholders when Size is nil.
func TestEpicPlaceholders_SizeNil(t *testing.T) {
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:       "E08",
			Title:     "Unsized Epic",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "active",
		Priority: models.PriorityLow,
	}

	m := EpicPlaceholders(epic)

	if got := m["size"]; got != "" {
		t.Errorf("EpicPlaceholders size (nil) = %q, want empty string", got)
	}
	if got := m["size_label"]; got != "" {
		t.Errorf("EpicPlaceholders size_label (nil) = %q, want empty string", got)
	}
}

// TC-F011-A: BugPlaceholders populates size and size_label for a bug with Size set.
func TestBugPlaceholders_SizePopulated(t *testing.T) {
	bug := &models.Bug{
		BaseEntity: models.BaseEntity{
			Key:       "B001",
			Title:     "Sized Bug",
			Size:      ptrInt(3),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "open",
		Severity: models.BugSeverityHigh,
	}

	m := BugPlaceholders(bug)

	if got := m["size"]; got != "3" {
		t.Errorf("BugPlaceholders size = %q, want %q", got, "3")
	}
	if got := m["size_label"]; got != "M" {
		t.Errorf("BugPlaceholders size_label = %q, want %q", got, "M")
	}
}

// TC-F011-B: BugPlaceholders produces empty-string placeholders when Size is nil.
func TestBugPlaceholders_SizeNil(t *testing.T) {
	bug := &models.Bug{
		BaseEntity: models.BaseEntity{
			Key:       "B002",
			Title:     "Unsized Bug",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "open",
		Severity: models.BugSeverityLow,
	}

	m := BugPlaceholders(bug)

	if got := m["size"]; got != "" {
		t.Errorf("BugPlaceholders size (nil) = %q, want empty string", got)
	}
	if got := m["size_label"]; got != "" {
		t.Errorf("BugPlaceholders size_label (nil) = %q, want empty string", got)
	}
}

// TC-F011-A: ChangeCardPlaceholders populates size and size_label when Size set.
func TestChangeCardPlaceholders_SizePopulated(t *testing.T) {
	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{
			Key:       "CC-001",
			Title:     "Sized Change",
			Size:      ptrInt(2),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "draft",
		Priority: 3,
	}

	m := ChangeCardPlaceholders(card)

	if got := m["size"]; got != "2" {
		t.Errorf("ChangeCardPlaceholders size = %q, want %q", got, "2")
	}
	if got := m["size_label"]; got != "S" {
		t.Errorf("ChangeCardPlaceholders size_label = %q, want %q", got, "S")
	}
}

// TC-F011-B: ChangeCardPlaceholders produces empty-string placeholders when Size is nil.
func TestChangeCardPlaceholders_SizeNil(t *testing.T) {
	card := &models.ChangeCard{
		BaseEntity: models.BaseEntity{
			Key:       "CC-002",
			Title:     "Unsized Change",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "draft",
		Priority: 1,
	}

	m := ChangeCardPlaceholders(card)

	if got := m["size"]; got != "" {
		t.Errorf("ChangeCardPlaceholders size (nil) = %q, want empty string", got)
	}
	if got := m["size_label"]; got != "" {
		t.Errorf("ChangeCardPlaceholders size_label (nil) = %q, want empty string", got)
	}
}

// TC-F012-A: complexity_tier in Metadata is preserved (not overwritten) when Size is nil.
func TestTaskPlaceholdersWithRelated_ComplexityTierPreservedWhenSizeNil(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E07-F42-001",
			Title:     "Task with complexity_tier only",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "todo",
		Metadata: map[string]interface{}{"complexity_tier": "L"},
	}

	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}
	ctx := context.Background()
	m := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// complexity_tier should be "L" from Metadata
	if got := m["complexity_tier"]; got != "L" {
		t.Errorf("complexity_tier = %q, want %q (TC-F012-A)", got, "L")
	}
	// size should be empty (Size is nil)
	if got := m["size"]; got != "" {
		t.Errorf("size = %q, want empty string when Size is nil (TC-F012-A)", got)
	}
	if got := m["size_label"]; got != "" {
		t.Errorf("size_label = %q, want empty string when Size is nil (TC-F012-A)", got)
	}
}

// TC-F012-B: Both complexity_tier and size placeholders populated independently.
func TestTaskPlaceholdersWithRelated_BothSizeAndComplexityTierPopulated(t *testing.T) {
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:       "T-E07-F42-002",
			Title:     "Task with both fields",
			Size:      ptrInt(8),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Status:   "todo",
		Metadata: map[string]interface{}{"complexity_tier": "L"},
	}

	mockDocRepo := &mockDocumentRepository{}
	mockTaskRelRepo := &mockTaskRelationshipRepository{}
	ctx := context.Background()
	m := TaskPlaceholdersWithRelated(ctx, task, mockDocRepo, mockTaskRelRepo, nil)

	// complexity_tier = "L" from Metadata — unchanged
	if got := m["complexity_tier"]; got != "L" {
		t.Errorf("complexity_tier = %q, want %q (TC-F012-B)", got, "L")
	}
	// size = "8" from Size field — independent
	if got := m["size"]; got != "8" {
		t.Errorf("size = %q, want %q (TC-F012-B)", got, "8")
	}
	if got := m["size_label"]; got != "XL" {
		t.Errorf("size_label = %q, want %q (TC-F012-B)", got, "XL")
	}
}

// IS-4: Template rendering integration — all per-entity placeholder builders
// produce "size" and "size_label" independently of complexity_tier.
func TestTemplatePlaceholders_SizeIndependentOfComplexityTier(t *testing.T) {
	// Confirm that for TaskPlaceholders (base builder), size and complexity_tier
	// are fully independent: one can be set without affecting the other.
	task5 := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:   "T-E07-F42-003",
			Title: "Task size=5 no tier",
			Size:  ptrInt(5),
		},
		Status: "todo",
	}
	m5 := TaskPlaceholders(task5)
	if m5["size"] != "5" || m5["size_label"] != "L" {
		t.Errorf("size=5 case: size=%q size_label=%q", m5["size"], m5["size_label"])
	}
	// complexity_tier key should not appear (no Metadata, no ContextData)
	if _, exists := m5["complexity_tier"]; exists {
		t.Errorf("complexity_tier should not be in TaskPlaceholders output (only set by WithRelated)")
	}

	// Nil size — both placeholders are empty strings, not absent
	taskNil := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:   "T-E07-F42-004",
			Title: "Task nil size",
		},
		Status: "todo",
	}
	mNil := TaskPlaceholders(taskNil)
	if mNil["size"] != "" {
		t.Errorf("size (nil) = %q, want empty string", mNil["size"])
	}
	if mNil["size_label"] != "" {
		t.Errorf("size_label (nil) = %q, want empty string", mNil["size_label"])
	}
}
