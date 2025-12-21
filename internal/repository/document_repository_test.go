package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestCreateOrGetNewDocument creates a new document
func TestCreateOrGetNewDocument(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	doc, err := docRepo.CreateOrGet(ctx, "OAuth Spec", "docs/oauth.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	if doc.ID == 0 {
		t.Error("Expected document ID to be set")
	}
	if doc.Title != "OAuth Spec" {
		t.Errorf("Expected title 'OAuth Spec', got %q", doc.Title)
	}
	if doc.FilePath != "docs/oauth.md" {
		t.Errorf("Expected file path 'docs/oauth.md', got %q", doc.FilePath)
	}
}

// TestCreateOrGetExistingDocument reuses existing document
func TestCreateOrGetExistingDocument(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	// Create first
	doc1, err := docRepo.CreateOrGet(ctx, "Architecture", "docs/arch.md")
	if err != nil {
		t.Fatalf("First CreateOrGet failed: %v", err)
	}

	// Get existing (same title and path)
	doc2, err := docRepo.CreateOrGet(ctx, "Architecture", "docs/arch.md")
	if err != nil {
		t.Fatalf("Second CreateOrGet failed: %v", err)
	}

	if doc1.ID != doc2.ID {
		t.Errorf("Expected same document ID, got %d and %d", doc1.ID, doc2.ID)
	}
}

// TestGetByID retrieves document by ID
func TestGetByID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	created, err := docRepo.CreateOrGet(ctx, "API Design", "docs/api.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	retrieved, err := docRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.Title != "API Design" {
		t.Errorf("Expected title 'API Design', got %q", retrieved.Title)
	}
}

// TestDeleteDocument removes document
func TestDeleteDocument(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	doc, err := docRepo.CreateOrGet(ctx, "ToDelete", "docs/delete.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	err = docRepo.Delete(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = docRepo.GetByID(ctx, doc.ID)
	if err == nil {
		t.Error("Expected error for deleted document")
	}
}

// TestLinkToEpic links document to epic
func TestLinkToEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)

	// Create or get test epic (handles duplicate key case in test reruns)
	testEpic := &models.Epic{
		Key:         "E70",
		Title:       "Test Link to Epic",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	doc, err := docRepo.CreateOrGet(ctx, "Epic Doc", "docs/epic.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	err = docRepo.LinkToEpic(ctx, testEpic.ID, doc.ID)
	if err != nil {
		t.Fatalf("LinkToEpic failed: %v", err)
	}

	// Verify linked
	docs, err := docRepo.ListForEpic(ctx, testEpic.ID)
	if err != nil {
		t.Fatalf("ListForEpic failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}
	if docs[0].ID != doc.ID {
		t.Errorf("Expected document ID %d, got %d", doc.ID, docs[0].ID)
	}
}

// TestLinkToFeature links document to feature
func TestLinkToFeature(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create or get test epic
	testEpic := &models.Epic{
		Key:         "E76",
		Title:       "Test Link Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Create or get feature (handles duplicate key case)
	feature := &models.Feature{
		EpicID:      testEpic.ID,
		Key:         "E76-F01",
		Title:       "Test Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
	}
	feature, _, err = featureRepo.CreateIfNotExists(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	doc, err := docRepo.CreateOrGet(ctx, "Feature Doc", "docs/feature.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	err = docRepo.LinkToFeature(ctx, feature.ID, doc.ID)
	if err != nil {
		t.Fatalf("LinkToFeature failed: %v", err)
	}

	docs, err := docRepo.ListForFeature(ctx, feature.ID)
	if err != nil {
		t.Fatalf("ListForFeature failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}
}

// TestLinkToTask links document to task
func TestLinkToTask(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	taskRepo := NewTaskRepository(db)

	test.SeedTestData()
	tasks, err := taskRepo.ListByEpic(ctx, "E99")
	if err != nil {
		t.Fatalf("ListByEpic failed: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("No tasks found")
	}

	doc, err := docRepo.CreateOrGet(ctx, "Task Doc", "docs/task.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	err = docRepo.LinkToTask(ctx, tasks[0].ID, doc.ID)
	if err != nil {
		t.Fatalf("LinkToTask failed: %v", err)
	}

	docs, err := docRepo.ListForTask(ctx, tasks[0].ID)
	if err != nil {
		t.Fatalf("ListForTask failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}
}

// TestUnlinkFromEpic removes document link from epic
func TestUnlinkFromEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)

	// Create or get test epic (handles duplicate key case in test reruns)
	testEpic := &models.Epic{
		Key:         "E71",
		Title:       "Test Unlink Epic",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	doc, err := docRepo.CreateOrGet(ctx, "Unlink Doc", "docs/unlink.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	_ = docRepo.LinkToEpic(ctx, testEpic.ID, doc.ID)

	err = docRepo.UnlinkFromEpic(ctx, testEpic.ID, doc.ID)
	if err != nil {
		t.Fatalf("UnlinkFromEpic failed: %v", err)
	}

	docs, err := docRepo.ListForEpic(ctx, testEpic.ID)
	if err != nil {
		t.Fatalf("ListForEpic failed: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

// TestUnlinkFromFeature removes document link from feature
func TestUnlinkFromFeature(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create or get test epic
	testEpic := &models.Epic{
		Key:         "E77",
		Title:       "Test Unlink Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Create or get feature (handles duplicate key case)
	feature := &models.Feature{
		EpicID:      testEpic.ID,
		Key:         "E77-F01",
		Title:       "Test Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
	}
	feature, _, err = featureRepo.CreateIfNotExists(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	doc, err := docRepo.CreateOrGet(ctx, "Unlink Feature Doc", "docs/unlink-f.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	_ = docRepo.LinkToFeature(ctx, feature.ID, doc.ID)

	err = docRepo.UnlinkFromFeature(ctx, feature.ID, doc.ID)
	if err != nil {
		t.Fatalf("UnlinkFromFeature failed: %v", err)
	}

	docs, err := docRepo.ListForFeature(ctx, feature.ID)
	if err != nil {
		t.Fatalf("ListForFeature failed: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

// TestUnlinkFromTask removes document link from task
func TestUnlinkFromTask(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Create or get a unique test epic and feature and task
	testEpic := &models.Epic{
		Key:         "E72",
		Title:       "Test Unlink Task",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	testFeature := &models.Feature{
		EpicID:      testEpic.ID,
		Key:         "E72-F01",
		Title:       "Test Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
	}
	testFeature, _, err = featureRepo.CreateIfNotExists(ctx, testFeature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	testTask := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E72-F01-999",
		Title:     "Test Task",
		Status:    "todo",
		Priority:  1,
	}
	if err := taskRepo.Create(ctx, testTask); err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	doc, err := docRepo.CreateOrGet(ctx, "Unlink Task Doc", "docs/unlink-t.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	_ = docRepo.LinkToTask(ctx, testTask.ID, doc.ID)

	err = docRepo.UnlinkFromTask(ctx, testTask.ID, doc.ID)
	if err != nil {
		t.Fatalf("UnlinkFromTask failed: %v", err)
	}

	docs, err := docRepo.ListForTask(ctx, testTask.ID)
	if err != nil {
		t.Fatalf("ListForTask failed: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

// TestListForEpic returns all documents for epic
func TestListForEpic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)

	// Create or get test epic (handles duplicate key case in test reruns)
	testEpic := &models.Epic{
		Key:         "E73",
		Title:       "Test List Epic",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	// Link multiple documents
	doc1, _ := docRepo.CreateOrGet(ctx, "Doc1", "docs/doc1.md")
	doc2, _ := docRepo.CreateOrGet(ctx, "Doc2", "docs/doc2.md")
	if err := docRepo.LinkToEpic(ctx, testEpic.ID, doc1.ID); err != nil {
		t.Fatalf("LinkToEpic failed: %v", err)
	}
	if err := docRepo.LinkToEpic(ctx, testEpic.ID, doc2.ID); err != nil {
		t.Fatalf("LinkToEpic failed: %v", err)
	}

	docs, err := docRepo.ListForEpic(ctx, testEpic.ID)
	if err != nil {
		t.Fatalf("ListForEpic failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

// TestListForFeature returns all documents for feature
func TestListForFeature(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)

	// Create or get test epic and feature (handles duplicate key case in test reruns)
	testEpic := &models.Epic{
		Key:         "E74",
		Title:       "Test List Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	testFeature := &models.Feature{
		EpicID:      testEpic.ID,
		Key:         "E74-F01",
		Title:       "Test Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
	}
	testFeature, _, err = featureRepo.CreateIfNotExists(ctx, testFeature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	// Link multiple documents
	doc1, _ := docRepo.CreateOrGet(ctx, "FDoc1", "docs/fdoc1.md")
	doc2, _ := docRepo.CreateOrGet(ctx, "FDoc2", "docs/fdoc2.md")
	if err := docRepo.LinkToFeature(ctx, testFeature.ID, doc1.ID); err != nil {
		t.Fatalf("LinkToFeature failed: %v", err)
	}
	if err := docRepo.LinkToFeature(ctx, testFeature.ID, doc2.ID); err != nil {
		t.Fatalf("LinkToFeature failed: %v", err)
	}

	docs, err := docRepo.ListForFeature(ctx, testFeature.ID)
	if err != nil {
		t.Fatalf("ListForFeature failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

// TestListForTask returns all documents for task
func TestListForTask(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)
	featureRepo := NewFeatureRepository(db)
	taskRepo := NewTaskRepository(db)

	// Create or get a unique test epic, feature, and task
	testEpic := &models.Epic{
		Key:         "E75",
		Title:       "Test List Task",
		Description: test.StringPtr("Test"),
		Status:      "active",
		Priority:    "high",
	}
	var err error
	testEpic, _, err = epicRepo.CreateIfNotExists(ctx, testEpic)
	if err != nil {
		t.Fatalf("Failed to create test epic: %v", err)
	}

	testFeature := &models.Feature{
		EpicID:      testEpic.ID,
		Key:         "E75-F01",
		Title:       "Test Feature",
		Description: test.StringPtr("Test"),
		Status:      "active",
	}
	testFeature, _, err = featureRepo.CreateIfNotExists(ctx, testFeature)
	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	testTask := &models.Task{
		FeatureID: testFeature.ID,
		Key:       "T-E75-F01-999",
		Title:     "Test Task",
		Status:    "todo",
		Priority:  1,
	}
	if err := taskRepo.Create(ctx, testTask); err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Link multiple documents
	doc1, _ := docRepo.CreateOrGet(ctx, "TDoc1", "docs/tdoc1.md")
	doc2, _ := docRepo.CreateOrGet(ctx, "TDoc2", "docs/tdoc2.md")
	if err := docRepo.LinkToTask(ctx, testTask.ID, doc1.ID); err != nil {
		t.Fatalf("LinkToTask failed: %v", err)
	}
	if err := docRepo.LinkToTask(ctx, testTask.ID, doc2.ID); err != nil {
		t.Fatalf("LinkToTask failed: %v", err)
	}

	docs, err := docRepo.ListForTask(ctx, testTask.ID)
	if err != nil {
		t.Fatalf("ListForTask failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

// TestDocumentReuseSameTitlePath reuses document with same title and path
func TestDocumentReuseSameTitlePath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)
	epicRepo := NewEpicRepository(db)

	_, _ = test.SeedTestData()
	epic, _ := epicRepo.GetByKey(ctx, "E99")

	// Link same document to different parents (via CreateOrGet)
	doc1, _ := docRepo.CreateOrGet(ctx, "SharedDoc", "docs/shared.md")
	_ = docRepo.LinkToEpic(ctx, epic.ID, doc1.ID)

	doc2, _ := docRepo.CreateOrGet(ctx, "SharedDoc", "docs/shared.md")
	if doc1.ID != doc2.ID {
		t.Error("Expected document reuse for same title and path")
	}
}

// TestGetByIDNotFound returns error for missing document
func TestGetByIDNotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	_, err := docRepo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
}

// TestGetByTitle retrieves document by title only
func TestGetByTitle(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	created, err := docRepo.CreateOrGet(ctx, "Title Only Doc", "docs/titleonly.md")
	if err != nil {
		t.Fatalf("CreateOrGet failed: %v", err)
	}

	retrieved, err := docRepo.GetByTitle(ctx, "Title Only Doc")
	if err != nil {
		t.Fatalf("GetByTitle failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, retrieved.ID)
	}
	if retrieved.Title != "Title Only Doc" {
		t.Errorf("Expected title 'Title Only Doc', got %q", retrieved.Title)
	}
	if retrieved.FilePath != "docs/titleonly.md" {
		t.Errorf("Expected file path 'docs/titleonly.md', got %q", retrieved.FilePath)
	}
}

// TestGetByTitleNotFound returns error for missing document title
func TestGetByTitleNotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := NewDB(database)
	docRepo := NewDocumentRepository(db)

	_, err := docRepo.GetByTitle(ctx, "NonexistentTitle")
	if err == nil {
		t.Error("Expected error for non-existent document title")
	}
}
