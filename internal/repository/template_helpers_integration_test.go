package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// Mock Feature Relationship Repository
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

// Mock Epic Relationship Repository
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

// testDocumentRepository implements config.DocumentRepository for integration testing
type testDocumentRepository struct {
	db *sql.DB
}

// testTaskRelationshipRepository implements config.TaskRelationshipRepository for integration testing
type testTaskRelationshipRepository struct {
	db *sql.DB
}

func (r *testTaskRelationshipRepository) ListRelatedTaskKeys(ctx context.Context, taskID int64) ([]string, error) {
	query := `
		SELECT DISTINCT t.key
		FROM tasks t
		JOIN task_relationships tr ON (t.id = tr.to_task_id OR t.id = tr.from_task_id)
		WHERE (tr.from_task_id = ? OR tr.to_task_id = ?) AND t.id != ?
		ORDER BY t.key
	`
	rows, err := r.db.QueryContext(ctx, query, taskID, taskID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (r *testDocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
	query := `
		SELECT DISTINCT d.id, d.title, d.file_path
		FROM documents d
		JOIN task_documents td ON d.id = td.document_id
		WHERE td.task_id = ?
		ORDER BY d.id
	`
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.FilePath); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (r *testDocumentRepository) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error) {
	query := `
		SELECT DISTINCT d.id, d.title, d.file_path
		FROM documents d
		JOIN feature_documents fd ON d.id = fd.document_id
		WHERE fd.feature_id = ?
		ORDER BY d.id
	`
	rows, err := r.db.QueryContext(ctx, query, featureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.FilePath); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (r *testDocumentRepository) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error) {
	query := `
		SELECT DISTINCT d.id, d.title, d.file_path
		FROM documents d
		JOIN epic_documents ed ON d.id = ed.document_id
		WHERE ed.epic_id = ?
		ORDER BY d.id
	`
	rows, err := r.db.QueryContext(ctx, query, epicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.FilePath); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// Integration Test: Task with Related Documents
func TestIntegrationTemplateTaskWithRelatedDocs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	defer templateIntegrationCleanup(database)

	// Setup
	_, featureID, taskID := templateSetupEpicFeatureTask(t, database, "TMPL-E01", "TMPL-E01-F01", "TMPL-E01-F01-001")
	doc1ID := templateCreateDocument(t, database, "Spec", "docs/spec.md")
	doc2ID := templateCreateDocument(t, database, "Design", "docs/design.md")
	templateLinkDocumentToTask(t, database, taskID, doc1ID)
	templateLinkDocumentToTask(t, database, taskID, doc2ID)

	docRepo := &testDocumentRepository{db: database}
	taskRelRepo := &testTaskRelationshipRepository{db: database}
	task := &models.Task{BaseEntity: models.BaseEntity{ID: taskID,
		Key:   "TMPL-E01-F01-001",
		Title: "Test Task"}, Status: "todo",
		Priority:  5,
		FeatureID: featureID,
	}

	// Execute
	placeholders := config.TaskPlaceholdersWithRelated(ctx, task, docRepo, taskRelRepo, nil)

	// Verify
	relatedDocs := placeholders["related_docs"]
	if relatedDocs != "docs/spec.md,docs/design.md" {
		t.Errorf("related_docs incorrect: got %q, want %q", relatedDocs, "docs/spec.md,docs/design.md")
	}
}

// Integration Test: Task with Related Tasks from Database Relationships
func TestIntegrationTemplateTaskWithRelatedTasks(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	defer templateIntegrationCleanup(database)

	// Setup: Create main task
	epicID, featureID, taskID := templateSetupEpicFeatureTask(t, database, "TMPL-E02", "TMPL-E02-F01", "TMPL-E02-F01-001")

	// Create related tasks
	relatedTask1ID := templateCreateTask(t, database, featureID, epicID, "TMPL-E02-F01-002", "Related Task 1")
	relatedTask2ID := templateCreateTask(t, database, featureID, epicID, "TMPL-E02-F01-003", "Related Task 2")

	// Create relationships from main task to related tasks
	_, err := database.Exec(
		`INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, ?)`,
		taskID, relatedTask1ID, "depends_on",
	)
	if err != nil {
		t.Fatalf("Failed to create relationship 1: %v", err)
	}

	_, err = database.Exec(
		`INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type) VALUES (?, ?, ?)`,
		taskID, relatedTask2ID, "blocks",
	)
	if err != nil {
		t.Fatalf("Failed to create relationship 2: %v", err)
	}

	docRepo := &testDocumentRepository{db: database}
	taskRelRepo := &testTaskRelationshipRepository{db: database}
	task := &models.Task{BaseEntity: models.BaseEntity{ID: taskID,
		Key:   "TMPL-E02-F01-001",
		Title: "Test Task"}, Status: "todo",
		Priority:  5,
		FeatureID: featureID,
	}

	// Execute
	placeholders := config.TaskPlaceholdersWithRelated(ctx, task, docRepo, taskRelRepo, nil)

	// Verify - should have both related task keys
	relatedTasks := placeholders["related_tasks"]
	if !strings.Contains(relatedTasks, "TMPL-E02-F01-002") || !strings.Contains(relatedTasks, "TMPL-E02-F01-003") {
		t.Errorf("related_tasks incorrect: got %q, expected both TMPL-E02-F01-002 and TMPL-E02-F01-003", relatedTasks)
	}
}

// Integration Test: Large Document List (55 docs)
func TestIntegrationTemplateLargeDocumentList(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	defer templateIntegrationCleanup(database)

	// Setup
	_, featureID, taskID := templateSetupEpicFeatureTask(t, database, "TMPL-E03", "TMPL-E03-F01", "TMPL-E03-F01-001")
	for i := 0; i < 55; i++ {
		docID := templateCreateDocument(t, database, fmt.Sprintf("Doc%d", i), fmt.Sprintf("docs/doc%d.md", i))
		templateLinkDocumentToTask(t, database, taskID, docID)
	}

	docRepo := &testDocumentRepository{db: database}
	taskRelRepo := &testTaskRelationshipRepository{db: database}
	task := &models.Task{BaseEntity: models.BaseEntity{ID: taskID,
		Key:   "TMPL-E03-F01-001",
		Title: "Test Task"}, Status: "todo",
		Priority:  5,
		FeatureID: featureID,
	}

	// Execute
	placeholders := config.TaskPlaceholdersWithRelated(ctx, task, docRepo, taskRelRepo, nil)

	// Verify
	relatedDocs := placeholders["related_docs"]
	for i := 0; i < 55; i++ {
		expectedPath := fmt.Sprintf("docs/doc%d.md", i)
		if !strings.Contains(relatedDocs, expectedPath) {
			t.Errorf("related_docs missing doc%d.md", i)
		}
	}
	commaCount := strings.Count(relatedDocs, ",")
	if commaCount != 54 {
		t.Errorf("expected 54 commas for 55 items, got %d", commaCount)
	}
}

// Integration Test: Dynamic Document Lookup
func TestIntegrationTemplateDynamicDocumentLookup(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	defer templateIntegrationCleanup(database)

	// Setup
	_, featureID, taskID := templateSetupEpicFeatureTask(t, database, "TMPL-E04", "TMPL-E04-F01", "TMPL-E04-F01-001")
	doc1ID := templateCreateDocument(t, database, "Doc1", "docs/doc1.md")
	doc2ID := templateCreateDocument(t, database, "Doc2", "docs/doc2.md")
	templateLinkDocumentToTask(t, database, taskID, doc1ID)
	templateLinkDocumentToTask(t, database, taskID, doc2ID)

	docRepo := &testDocumentRepository{db: database}
	taskRelRepo := &testTaskRelationshipRepository{db: database}
	task := &models.Task{BaseEntity: models.BaseEntity{ID: taskID,
		Key:   "TMPL-E04-F01-001",
		Title: "Test Task"}, Status: "todo",
		Priority:  5,
		FeatureID: featureID,
	}

	// Initial lookup
	placeholders1 := config.TaskPlaceholdersWithRelated(ctx, task, docRepo, taskRelRepo, nil)
	if !strings.Contains(placeholders1["related_docs"], "docs/doc1.md") {
		t.Errorf("Initial lookup missing doc1.md")
	}

	// Unlink document
	_, err := database.Exec(`DELETE FROM task_documents WHERE task_id = ? AND document_id = ?`, taskID, doc1ID)
	if err != nil {
		t.Fatalf("Failed to unlink document: %v", err)
	}

	// Second lookup (dynamic)
	placeholders2 := config.TaskPlaceholdersWithRelated(ctx, task, docRepo, taskRelRepo, nil)
	if strings.Contains(placeholders2["related_docs"], "docs/doc1.md") {
		t.Errorf("Dynamic lookup should not include unlinked document")
	}
	if placeholders2["related_docs"] != "docs/doc2.md" {
		t.Errorf("Dynamic lookup incorrect: got %q", placeholders2["related_docs"])
	}
}

// Integration Test: Feature-Level Placeholders
func TestIntegrationTemplateFeaturePlaceholdersWithDocs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	defer templateIntegrationCleanup(database)

	// Setup
	epicID := templateCreateEpic(t, database, "TMPL-E05", "Test Epic")
	featureID := templateCreateFeature(t, database, epicID, "TMPL-E05-F01", "Test Feature")
	doc1ID := templateCreateDocument(t, database, "Arch", "docs/arch.md")
	doc2ID := templateCreateDocument(t, database, "PRD", "docs/prd.md")
	templateLinkDocumentToFeature(t, database, featureID, doc1ID)
	templateLinkDocumentToFeature(t, database, featureID, doc2ID)

	docRepo := &testDocumentRepository{db: database}
	mockRelRepo := &mockFeatureRelationshipRepository{features: []string{}}
	feature := &models.Feature{BaseEntity: models.BaseEntity{ID: featureID,
		Key:   "TMPL-E05-F01",
		Title: "Test Feature",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: models.FeatureStatusActive,
	}

	// Execute
	placeholders := config.FeaturePlaceholdersWithRelated(ctx, feature, docRepo, mockRelRepo, nil)

	// Verify
	relatedDocs := placeholders["related_docs"]
	if !strings.Contains(relatedDocs, "docs/arch.md") || !strings.Contains(relatedDocs, "docs/prd.md") {
		t.Errorf("related_docs missing documents: got %q", relatedDocs)
	}
}

// Integration Test: Epic-Level Placeholders
func TestIntegrationTemplateEpicPlaceholdersWithDocs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	defer templateIntegrationCleanup(database)

	// Setup
	epicID := templateCreateEpic(t, database, "TMPL-E06", "Test Epic")
	doc1ID := templateCreateDocument(t, database, "Summary", "docs/summary.md")
	doc2ID := templateCreateDocument(t, database, "Roadmap", "docs/roadmap.md")
	templateLinkDocumentToEpic(t, database, epicID, doc1ID)
	templateLinkDocumentToEpic(t, database, epicID, doc2ID)

	docRepo := &testDocumentRepository{db: database}
	mockRelRepo := &mockEpicRelationshipRepository{epics: []string{}}
	epic := &models.Epic{BaseEntity: models.BaseEntity{ID: epicID,
		Key:   "TMPL-E06",
		Title: "Test Epic",

		CreatedAt: time.Now(),
		UpdatedAt: time.Now()}, Status: models.EpicStatusActive,
		Priority: models.PriorityHigh,
	}

	// Execute
	placeholders := config.EpicPlaceholdersWithRelated(epic, docRepo, mockRelRepo, ctx, nil)

	// Verify
	relatedDocs := placeholders["related_docs"]
	if relatedDocs != "docs/summary.md,docs/roadmap.md" {
		t.Errorf("related_docs incorrect: got %q", relatedDocs)
	}
}

// Helper functions

func templateSetupEpicFeatureTask(t *testing.T, database *sql.DB, epicKey, featureKey, taskKey string) (int64, int64, int64) {
	epicID := templateCreateEpic(t, database, epicKey, "Test Epic")
	featureID := templateCreateFeature(t, database, epicID, featureKey, "Test Feature")
	taskID := templateCreateTask(t, database, featureID, epicID, taskKey, "Test Task")
	return epicID, featureID, taskID
}

func templateCreateEpic(t *testing.T, database *sql.DB, key, title string) int64 {
	result, err := database.Exec(
		`INSERT INTO epics (key, title, description, status, priority) VALUES (?, ?, ?, 'active', 'high')`,
		key, title, "Test epic",
	)
	if err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get epic ID: %v", err)
	}
	return id
}

func templateCreateFeature(t *testing.T, database *sql.DB, epicID int64, key, title string) int64 {
	result, err := database.Exec(
		`INSERT INTO features (epic_id, key, title, description, status) VALUES (?, ?, ?, ?, 'active')`,
		epicID, key, title, "Test feature",
	)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get feature ID: %v", err)
	}
	return id
}

func templateCreateTask(t *testing.T, database *sql.DB, featureID, epicID int64, key, title string) int64 {
	result, err := database.Exec(
		`INSERT INTO tasks (feature_id, key, title, description, status, priority) VALUES (?, ?, ?, ?, 'todo', 5)`,
		featureID, key, title, "Test task",
	)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get task ID: %v", err)
	}
	return id
}

func templateCreateDocument(t *testing.T, database *sql.DB, title, filePath string) int64 {
	result, err := database.Exec(
		`INSERT INTO documents (title, file_path) VALUES (?, ?)`,
		title, filePath,
	)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get document ID: %v", err)
	}
	return id
}

func templateLinkDocumentToTask(t *testing.T, database *sql.DB, taskID, documentID int64) {
	_, err := database.Exec(
		`INSERT INTO task_documents (task_id, document_id) VALUES (?, ?)`,
		taskID, documentID,
	)
	if err != nil {
		t.Fatalf("Failed to link document to task: %v", err)
	}
}

func templateLinkDocumentToFeature(t *testing.T, database *sql.DB, featureID, documentID int64) {
	_, err := database.Exec(
		`INSERT INTO feature_documents (feature_id, document_id) VALUES (?, ?)`,
		featureID, documentID,
	)
	if err != nil {
		t.Fatalf("Failed to link document to feature: %v", err)
	}
}

func templateLinkDocumentToEpic(t *testing.T, database *sql.DB, epicID, documentID int64) {
	_, err := database.Exec(
		`INSERT INTO epic_documents (epic_id, document_id) VALUES (?, ?)`,
		epicID, documentID,
	)
	if err != nil {
		t.Fatalf("Failed to link document to epic: %v", err)
	}
}

func templateIntegrationCleanup(database *sql.DB) {
	// Clean up in reverse order
	database.Exec("DELETE FROM task_documents WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'TMPL-%')")
	database.Exec("DELETE FROM feature_documents WHERE feature_id IN (SELECT id FROM features WHERE key LIKE 'TMPL-%')")
	database.Exec("DELETE FROM epic_documents WHERE epic_id IN (SELECT id FROM epics WHERE key LIKE 'TMPL-%')")
	database.Exec("DELETE FROM tasks WHERE key LIKE 'TMPL-%'")
	database.Exec("DELETE FROM features WHERE key LIKE 'TMPL-%'")
	database.Exec("DELETE FROM epics WHERE key LIKE 'TMPL-%'")
	database.Exec("DELETE FROM documents WHERE title LIKE 'TMPL-%' OR title LIKE 'Spec%' OR title LIKE 'Design%' OR title LIKE 'Doc%' OR title LIKE 'Arch%' OR title LIKE 'PRD%' OR title LIKE 'Summary%' OR title LIKE 'Roadmap%'")
}
