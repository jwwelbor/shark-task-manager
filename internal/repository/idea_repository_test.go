package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestIdeaRepository_Create tests creating a new idea
func TestIdeaRepository_Create(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key LIKE 'I-2026-01-01-%'")

	// Create test idea
	createdDate := time.Now()
	priority := 5
	status := models.IdeaStatusNew

	idea := &models.Idea{
		Key:         "I-2026-01-01-01",
		Title:       "Test Idea",
		CreatedDate: createdDate,
		Priority:    &priority,
		Status:      status,
	}

	err := repo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}

	if idea.ID == 0 {
		t.Error("Expected ID to be set after create")
	}

	// Clean up
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea.ID)
	}()
}

// TestIdeaRepository_GetByKey tests retrieving an idea by its key
func TestIdeaRepository_GetByKey(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key = 'I-2026-01-01-02'")

	// Create test idea
	createdDate := time.Now()
	priority := 7
	description := "Test Description"
	status := models.IdeaStatusNew

	idea := &models.Idea{
		Key:         "I-2026-01-01-02",
		Title:       "Test Idea for GetByKey",
		Description: &description,
		CreatedDate: createdDate,
		Priority:    &priority,
		Status:      status,
	}

	err := repo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea.ID)
	}()

	// Retrieve by key
	retrieved, err := repo.GetByKey(ctx, "I-2026-01-01-02")
	if err != nil {
		t.Fatalf("Failed to get idea by key: %v", err)
	}

	if retrieved.Key != idea.Key {
		t.Errorf("Expected key %s, got %s", idea.Key, retrieved.Key)
	}
	if retrieved.Title != idea.Title {
		t.Errorf("Expected title %s, got %s", idea.Title, retrieved.Title)
	}
	if *retrieved.Priority != *idea.Priority {
		t.Errorf("Expected priority %d, got %d", *idea.Priority, *retrieved.Priority)
	}
}

// TestIdeaRepository_GetByID tests retrieving an idea by its ID
func TestIdeaRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key = 'I-2026-01-01-03'")

	// Create test idea
	createdDate := time.Now()
	status := models.IdeaStatusNew

	idea := &models.Idea{
		Key:         "I-2026-01-01-03",
		Title:       "Test Idea for GetByID",
		CreatedDate: createdDate,
		Status:      status,
	}

	err := repo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea.ID)
	}()

	// Retrieve by ID
	retrieved, err := repo.GetByID(ctx, idea.ID)
	if err != nil {
		t.Fatalf("Failed to get idea by ID: %v", err)
	}

	if retrieved.ID != idea.ID {
		t.Errorf("Expected ID %d, got %d", idea.ID, retrieved.ID)
	}
	if retrieved.Title != idea.Title {
		t.Errorf("Expected title %s, got %s", idea.Title, retrieved.Title)
	}
}

// TestIdeaRepository_List tests listing all ideas
func TestIdeaRepository_List(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key LIKE 'I-2026-01-02-%'")

	// Create multiple test ideas
	createdDate := time.Now()
	status1 := models.IdeaStatusNew
	status2 := models.IdeaStatusOnHold

	idea1 := &models.Idea{
		Key:         "I-2026-01-02-01",
		Title:       "First Test Idea",
		CreatedDate: createdDate,
		Status:      status1,
	}

	idea2 := &models.Idea{
		Key:         "I-2026-01-02-02",
		Title:       "Second Test Idea",
		CreatedDate: createdDate,
		Status:      status2,
	}

	err := repo.Create(ctx, idea1)
	if err != nil {
		t.Fatalf("Failed to create idea1: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea1.ID) }()

	err = repo.Create(ctx, idea2)
	if err != nil {
		t.Fatalf("Failed to create idea2: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea2.ID) }()

	// List all ideas
	ideas, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list ideas: %v", err)
	}

	// Should have at least our 2 test ideas
	if len(ideas) < 2 {
		t.Errorf("Expected at least 2 ideas, got %d", len(ideas))
	}

	// Verify our test ideas are in the list
	foundIdea1 := false
	foundIdea2 := false
	for _, idea := range ideas {
		if idea.Key == "I-2026-01-02-01" {
			foundIdea1 = true
		}
		if idea.Key == "I-2026-01-02-02" {
			foundIdea2 = true
		}
	}

	if !foundIdea1 {
		t.Error("Expected to find idea1 in list")
	}
	if !foundIdea2 {
		t.Error("Expected to find idea2 in list")
	}
}

// TestIdeaRepository_ListWithStatusFilter tests filtering ideas by status
func TestIdeaRepository_ListWithStatusFilter(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key LIKE 'I-2026-01-03-%'")

	// Create test ideas with different statuses
	createdDate := time.Now()
	statusNew := models.IdeaStatusNew
	statusOnHold := models.IdeaStatusOnHold

	idea1 := &models.Idea{
		Key:         "I-2026-01-03-01",
		Title:       "New Idea",
		CreatedDate: createdDate,
		Status:      statusNew,
	}

	idea2 := &models.Idea{
		Key:         "I-2026-01-03-02",
		Title:       "On Hold Idea",
		CreatedDate: createdDate,
		Status:      statusOnHold,
	}

	err := repo.Create(ctx, idea1)
	if err != nil {
		t.Fatalf("Failed to create idea1: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea1.ID) }()

	err = repo.Create(ctx, idea2)
	if err != nil {
		t.Fatalf("Failed to create idea2: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea2.ID) }()

	// Filter by status "new"
	filter := &IdeaFilter{Status: &statusNew}
	ideas, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list ideas with filter: %v", err)
	}

	// All returned ideas should have status "new"
	for _, idea := range ideas {
		if idea.Status != statusNew {
			t.Errorf("Expected all ideas to have status 'new', got '%s'", idea.Status)
		}
	}
}

// TestIdeaRepository_Update tests updating an idea
func TestIdeaRepository_Update(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key = 'I-2026-01-04-01'")

	// Create test idea
	createdDate := time.Now()
	priority := 5
	status := models.IdeaStatusNew

	idea := &models.Idea{
		Key:         "I-2026-01-04-01",
		Title:       "Original Title",
		CreatedDate: createdDate,
		Priority:    &priority,
		Status:      status,
	}

	err := repo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea.ID) }()

	// Update idea
	idea.Title = "Updated Title"
	newPriority := 8
	idea.Priority = &newPriority
	newStatus := models.IdeaStatusOnHold
	idea.Status = newStatus

	err = repo.Update(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to update idea: %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.GetByID(ctx, idea.ID)
	if err != nil {
		t.Fatalf("Failed to get updated idea: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", retrieved.Title)
	}
	if *retrieved.Priority != 8 {
		t.Errorf("Expected priority 8, got %d", *retrieved.Priority)
	}
	if retrieved.Status != newStatus {
		t.Errorf("Expected status '%s', got '%s'", newStatus, retrieved.Status)
	}
}

// TestIdeaRepository_Delete tests deleting an idea
func TestIdeaRepository_Delete(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key = 'I-2026-01-05-01'")

	// Create test idea
	createdDate := time.Now()
	status := models.IdeaStatusNew

	idea := &models.Idea{
		Key:         "I-2026-01-05-01",
		Title:       "Idea to Delete",
		CreatedDate: createdDate,
		Status:      status,
	}

	err := repo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}

	// Delete idea
	err = repo.Delete(ctx, idea.ID)
	if err != nil {
		t.Fatalf("Failed to delete idea: %v", err)
	}

	// Verify it's deleted
	_, err = repo.GetByID(ctx, idea.ID)
	if err == nil {
		t.Error("Expected error when getting deleted idea, got none")
	}
}

// TestIdeaRepository_MarkAsConverted tests marking an idea as converted
func TestIdeaRepository_MarkAsConverted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE key = 'I-2026-01-06-01'")

	// Create test idea
	createdDate := time.Now()
	status := models.IdeaStatusNew

	idea := &models.Idea{
		Key:         "I-2026-01-06-01",
		Title:       "Idea to Convert",
		CreatedDate: createdDate,
		Status:      status,
	}

	err := repo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM ideas WHERE id = ?", idea.ID) }()

	// Mark as converted to epic
	err = repo.MarkAsConverted(ctx, idea.ID, "epic", "E15")
	if err != nil {
		t.Fatalf("Failed to mark idea as converted: %v", err)
	}

	// Retrieve and verify conversion tracking
	retrieved, err := repo.GetByID(ctx, idea.ID)
	if err != nil {
		t.Fatalf("Failed to get converted idea: %v", err)
	}

	// Verify status changed to converted
	if retrieved.Status != models.IdeaStatusConverted {
		t.Errorf("Expected status 'converted', got '%s'", retrieved.Status)
	}

	// Verify converted_to_type is set
	if retrieved.ConvertedToType == nil {
		t.Fatal("Expected ConvertedToType to be set, got nil")
	}
	if *retrieved.ConvertedToType != "epic" {
		t.Errorf("Expected ConvertedToType 'epic', got '%s'", *retrieved.ConvertedToType)
	}

	// Verify converted_to_key is set
	if retrieved.ConvertedToKey == nil {
		t.Fatal("Expected ConvertedToKey to be set, got nil")
	}
	if *retrieved.ConvertedToKey != "E15" {
		t.Errorf("Expected ConvertedToKey 'E15', got '%s'", *retrieved.ConvertedToKey)
	}

	// Verify converted_at is set
	if retrieved.ConvertedAt == nil {
		t.Fatal("Expected ConvertedAt to be set, got nil")
	}

	// ConvertedAt should be recent (within last minute)
	if time.Since(*retrieved.ConvertedAt) > time.Minute {
		t.Errorf("ConvertedAt timestamp seems too old: %v", *retrieved.ConvertedAt)
	}
}

// TestIdeaRepository_MarkAsConverted_NonExistentIdea tests error handling
func TestIdeaRepository_MarkAsConverted_NonExistentIdea(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	dbWrapper := NewDB(database)
	repo := NewIdeaRepository(dbWrapper)

	// Try to mark non-existent idea as converted
	err := repo.MarkAsConverted(ctx, 999999999, "epic", "E99")
	if err == nil {
		t.Error("Expected error when marking non-existent idea as converted, got none")
	}
}
