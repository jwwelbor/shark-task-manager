package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestTemplateEnrichmentRepository_GetTaskEnrichment_FullData tests task enrichment
// with all data present: history, notes, siblings.
func TestTemplateEnrichmentRepository_GetTaskEnrichment_FullData(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTemplateEnrichmentRepository(db)

	// Clean up any existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'TEST-ENRICH-%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE entity_type = 'task' AND entity_id IN (SELECT id FROM tasks WHERE key LIKE 'TEST-ENRICH-%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-ENRICH-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'TEST-ENRICH-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-ENRICH-%'")

	// Create test epic
	epicResult, err := database.ExecContext(ctx, "INSERT INTO epics (key, title, status, priority) VALUES ('TEST-ENRICH-E01', 'Test Epic', 'active', 'medium')")
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}
	epicID, _ := epicResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	// Create test feature
	featureResult, err := database.ExecContext(ctx, "INSERT INTO features (key, title, status, epic_id) VALUES ('TEST-ENRICH-E01-F01', 'Test Feature', 'active', ?)", epicID)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}
	featureID, _ := featureResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	// Create test task
	taskResult, err := database.ExecContext(ctx, "INSERT INTO tasks (key, title, status, feature_id) VALUES ('TEST-ENRICH-E01-F01-001', 'Test Task', 'in_development', ?)", featureID)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}
	taskID, _ := taskResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)

	// Create sibling tasks
	sibling1Result, err := database.ExecContext(ctx, "INSERT INTO tasks (key, title, status, feature_id) VALUES ('TEST-ENRICH-E01-F01-002', 'Completed Task', 'completed', ?)", featureID)
	if err != nil {
		t.Fatalf("Failed to create sibling task: %v", err)
	}
	sibling1ID, _ := sibling1Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", sibling1ID)

	sibling2Result, err := database.ExecContext(ctx, "INSERT INTO tasks (key, title, status, feature_id) VALUES ('TEST-ENRICH-E01-F01-003', 'Blocked Task', 'blocked', ?)", featureID)
	if err != nil {
		t.Fatalf("Failed to create sibling task: %v", err)
	}
	sibling2ID, _ := sibling2Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", sibling2ID)

	// Create task history
	_, err = database.ExecContext(ctx, "INSERT INTO task_history (task_id, old_status, new_status, timestamp) VALUES (?, 'todo', 'in_development', datetime('now'))", taskID)
	if err != nil {
		t.Fatalf("Failed to create task history: %v", err)
	}
	defer database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id = ?", taskID)

	// Create entity notes
	_, err = database.ExecContext(ctx, "INSERT INTO entity_notes (entity_type, entity_id, note_type, content, created_at) VALUES ('task', ?, 'comment', 'First comment', datetime('now', '-1 hour'))", taskID)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}
	_, err = database.ExecContext(ctx, "INSERT INTO entity_notes (entity_type, entity_id, note_type, content, created_at) VALUES ('task', ?, 'rejection', 'Needs rework', datetime('now'))", taskID)
	if err != nil {
		t.Fatalf("Failed to create rejection note: %v", err)
	}
	defer database.ExecContext(ctx, "DELETE FROM entity_notes WHERE entity_type = 'task' AND entity_id = ?", taskID)

	// Execute
	data, err := repo.GetTaskEnrichment(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskEnrichment() error = %v", err)
	}

	// Verify
	if data.PreviousStatus != "todo" {
		t.Errorf("PreviousStatus = %q, want %q", data.PreviousStatus, "todo")
	}
	if data.ParentTitle != "Test Feature" {
		t.Errorf("ParentTitle = %q, want %q", data.ParentTitle, "Test Feature")
	}
	if data.GrandparentTitle != "Test Epic" {
		t.Errorf("GrandparentTitle = %q, want %q", data.GrandparentTitle, "Test Epic")
	}
	if data.LatestNoteContent != "Needs rework" {
		t.Errorf("LatestNoteContent = %q, want %q", data.LatestNoteContent, "Needs rework")
	}
	if data.LatestNoteType != "rejection" {
		t.Errorf("LatestNoteType = %q, want %q", data.LatestNoteType, "rejection")
	}
	if data.NotesCount != 2 {
		t.Errorf("NotesCount = %d, want %d", data.NotesCount, 2)
	}
	if data.RejectionCount != 1 {
		t.Errorf("RejectionCount = %d, want %d", data.RejectionCount, 1)
	}
	if data.SiblingTotal != 3 {
		t.Errorf("SiblingTotal = %d, want %d", data.SiblingTotal, 3)
	}
	if data.SiblingCompleted != 1 {
		t.Errorf("SiblingCompleted = %d, want %d", data.SiblingCompleted, 1)
	}
	if data.SiblingBlocked != 1 {
		t.Errorf("SiblingBlocked = %d, want %d", data.SiblingBlocked, 1)
	}
}

// TestTemplateEnrichmentRepository_GetTaskEnrichment_EmptyHistory tests task with no history.
func TestTemplateEnrichmentRepository_GetTaskEnrichment_EmptyHistory(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTemplateEnrichmentRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'TEST-ENRICH-NOHIST-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'TEST-ENRICH-NOHIST-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'TEST-ENRICH-NOHIST-E01'")

	epicResult, _ := database.ExecContext(ctx, "INSERT INTO epics (key, title, status, priority) VALUES ('TEST-ENRICH-NOHIST-E01', 'Epic', 'active', 'medium')")
	epicID, _ := epicResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	featureResult, _ := database.ExecContext(ctx, "INSERT INTO features (key, title, status, epic_id) VALUES ('TEST-ENRICH-NOHIST-F01', 'Feature', 'active', ?)", epicID)
	featureID, _ := featureResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	taskResult, _ := database.ExecContext(ctx, "INSERT INTO tasks (key, title, status, feature_id) VALUES ('TEST-ENRICH-NOHIST-001', 'Task', 'todo', ?)", featureID)
	taskID, _ := taskResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)

	data, err := repo.GetTaskEnrichment(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTaskEnrichment() error = %v", err)
	}

	if data.PreviousStatus != "" {
		t.Errorf("PreviousStatus = %q, want empty string", data.PreviousStatus)
	}
}

// TestTemplateEnrichmentRepository_GetFeatureEnrichment_FullData tests feature enrichment.
func TestTemplateEnrichmentRepository_GetFeatureEnrichment_FullData(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTemplateEnrichmentRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE entity_type = 'feature' AND entity_id IN (SELECT id FROM features WHERE key LIKE 'TEST-ENRICH-FE-%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-ENRICH-FE-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'TEST-ENRICH-FE-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-ENRICH-FE-%'")

	epicResult, _ := database.ExecContext(ctx, "INSERT INTO epics (key, title, status, priority) VALUES ('TEST-ENRICH-FE-E01', 'Feature Test Epic', 'active', 'medium')")
	epicID, _ := epicResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	featureResult, _ := database.ExecContext(ctx, "INSERT INTO features (key, title, status, epic_id) VALUES ('TEST-ENRICH-FE-F01', 'Test Feature', 'active', ?)", epicID)
	featureID, _ := featureResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	// Create tasks in the feature
	task1Result, _ := database.ExecContext(ctx, "INSERT INTO tasks (key, title, status, feature_id) VALUES ('TEST-ENRICH-FE-001', 'T1', 'completed', ?)", featureID)
	task1ID, _ := task1Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task1ID)

	task2Result, _ := database.ExecContext(ctx, "INSERT INTO tasks (key, title, status, feature_id) VALUES ('TEST-ENRICH-FE-002', 'T2', 'blocked', ?)", featureID)
	task2ID, _ := task2Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task2ID)

	task3Result, _ := database.ExecContext(ctx, "INSERT INTO tasks (key, title, status, feature_id) VALUES ('TEST-ENRICH-FE-003', 'T3', 'in_development', ?)", featureID)
	task3ID, _ := task3Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task3ID)

	data, err := repo.GetFeatureEnrichment(ctx, featureID)
	if err != nil {
		t.Fatalf("GetFeatureEnrichment() error = %v", err)
	}

	if data.PreviousStatus != "" {
		t.Errorf("PreviousStatus = %q, want empty (features have no history in v1)", data.PreviousStatus)
	}
	if data.ParentTitle != "Feature Test Epic" {
		t.Errorf("ParentTitle = %q, want %q", data.ParentTitle, "Feature Test Epic")
	}
	if data.GrandparentTitle != "" {
		t.Errorf("GrandparentTitle = %q, want empty (features have no grandparent)", data.GrandparentTitle)
	}
	if data.SiblingTotal != 3 {
		t.Errorf("SiblingTotal = %d, want %d", data.SiblingTotal, 3)
	}
	if data.SiblingCompleted != 1 {
		t.Errorf("SiblingCompleted = %d, want %d", data.SiblingCompleted, 1)
	}
	if data.SiblingBlocked != 1 {
		t.Errorf("SiblingBlocked = %d, want %d", data.SiblingBlocked, 1)
	}
}

// TestTemplateEnrichmentRepository_GetEpicEnrichment_FullData tests epic enrichment.
func TestTemplateEnrichmentRepository_GetEpicEnrichment_FullData(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTemplateEnrichmentRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE entity_type = 'epic' AND entity_id IN (SELECT id FROM epics WHERE key LIKE 'TEST-ENRICH-EP-%')")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'TEST-ENRICH-EP-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-ENRICH-EP-%'")

	epicResult, _ := database.ExecContext(ctx, "INSERT INTO epics (key, title, status, priority) VALUES ('TEST-ENRICH-EP-E01', 'Epic Test', 'active', 'medium')")
	epicID, _ := epicResult.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	// Create features in the epic
	f1Result, _ := database.ExecContext(ctx, "INSERT INTO features (key, title, status, epic_id) VALUES ('TEST-ENRICH-EP-F01', 'F1', 'completed', ?)", epicID)
	f1ID, _ := f1Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", f1ID)

	f2Result, _ := database.ExecContext(ctx, "INSERT INTO features (key, title, status, epic_id) VALUES ('TEST-ENRICH-EP-F02', 'F2', 'blocked', ?)", epicID)
	f2ID, _ := f2Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", f2ID)

	f3Result, _ := database.ExecContext(ctx, "INSERT INTO features (key, title, status, epic_id) VALUES ('TEST-ENRICH-EP-F03', 'F3', 'active', ?)", epicID)
	f3ID, _ := f3Result.LastInsertId()
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", f3ID)

	// Add a note to the epic
	_, _ = database.ExecContext(ctx, "INSERT INTO entity_notes (entity_type, entity_id, note_type, content, created_at) VALUES ('epic', ?, 'comment', 'Epic note', datetime('now'))", epicID)
	defer database.ExecContext(ctx, "DELETE FROM entity_notes WHERE entity_type = 'epic' AND entity_id = ?", epicID)

	data, err := repo.GetEpicEnrichment(ctx, epicID)
	if err != nil {
		t.Fatalf("GetEpicEnrichment() error = %v", err)
	}

	if data.ParentTitle != "" {
		t.Errorf("ParentTitle = %q, want empty (epics have no parent)", data.ParentTitle)
	}
	if data.SiblingTotal != 3 {
		t.Errorf("SiblingTotal = %d, want %d", data.SiblingTotal, 3)
	}
	if data.SiblingCompleted != 1 {
		t.Errorf("SiblingCompleted = %d, want %d", data.SiblingCompleted, 1)
	}
	if data.SiblingBlocked != 1 {
		t.Errorf("SiblingBlocked = %d, want %d", data.SiblingBlocked, 1)
	}
	if data.NotesCount != 1 {
		t.Errorf("NotesCount = %d, want %d", data.NotesCount, 1)
	}
	if data.LatestNoteContent != "Epic note" {
		t.Errorf("LatestNoteContent = %q, want %q", data.LatestNoteContent, "Epic note")
	}
}

// TestTemplateEnrichmentRepository_NonExistentEntity tests non-existent entity.
func TestTemplateEnrichmentRepository_NonExistentEntity(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	repo := NewTemplateEnrichmentRepository(db)

	// Non-existent task
	data, err := repo.GetTaskEnrichment(ctx, 999999)
	if err != nil {
		t.Fatalf("GetTaskEnrichment() for non-existent should not error, got %v", err)
	}
	if data.PreviousStatus != "" || data.ParentTitle != "" || data.SiblingTotal != 0 {
		t.Errorf("Expected zero-valued struct for non-existent entity, got %+v", data)
	}

	// Non-existent feature
	fData, fErr := repo.GetFeatureEnrichment(ctx, 999999)
	if fErr != nil {
		t.Fatalf("GetFeatureEnrichment() for non-existent should not error, got %v", fErr)
	}
	if fData.SiblingTotal != 0 {
		t.Errorf("Expected zero-valued struct for non-existent feature, got sibling_total=%d", fData.SiblingTotal)
	}

	// Non-existent epic
	eData, eErr := repo.GetEpicEnrichment(ctx, 999999)
	if eErr != nil {
		t.Fatalf("GetEpicEnrichment() for non-existent should not error, got %v", eErr)
	}
	if eData.SiblingTotal != 0 {
		t.Errorf("Expected zero-valued struct for non-existent epic, got sibling_total=%d", eData.SiblingTotal)
	}
}
