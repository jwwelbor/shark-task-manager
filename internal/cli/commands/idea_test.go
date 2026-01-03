package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// TestGenerateIdeaKey tests the idea key generation logic
func TestGenerateIdeaKey(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	dateStr := now.Format("2006-01-02")

	tests := []struct {
		name          string
		existingIdeas []*models.Idea
		expectedKey   string
		expectedError bool
	}{
		{
			name:          "First idea of the day",
			existingIdeas: []*models.Idea{},
			expectedKey:   fmt.Sprintf("I-%s-01", dateStr),
			expectedError: false,
		},
		{
			name: "Second idea of the day",
			existingIdeas: []*models.Idea{
				{Key: fmt.Sprintf("I-%s-01", dateStr), Title: "First idea"},
			},
			expectedKey:   fmt.Sprintf("I-%s-02", dateStr),
			expectedError: false,
		},
		{
			name: "Multiple existing ideas",
			existingIdeas: []*models.Idea{
				{Key: fmt.Sprintf("I-%s-01", dateStr), Title: "First"},
				{Key: fmt.Sprintf("I-%s-02", dateStr), Title: "Second"},
				{Key: fmt.Sprintf("I-%s-03", dateStr), Title: "Third"},
			},
			expectedKey:   fmt.Sprintf("I-%s-04", dateStr),
			expectedError: false,
		},
		{
			name: "Ideas from different days (should ignore)",
			existingIdeas: []*models.Idea{
				{Key: "I-2025-12-31-01", Title: "Old idea"},
				{Key: fmt.Sprintf("I-%s-01", dateStr), Title: "Today's idea"},
			},
			expectedKey:   fmt.Sprintf("I-%s-02", dateStr),
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &MockIdeaRepository{
				ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
					return tt.existingIdeas, nil
				},
				GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
					// Count existing ideas for today
					maxSeq := 0
					prefix := fmt.Sprintf("I-%s-", dateStr)
					for _, idea := range tt.existingIdeas {
						if len(idea.Key) >= len(prefix) && idea.Key[:len(prefix)] == prefix {
							// Extract sequence number from key (last 2 digits)
							var seq int
							_, err := fmt.Sscanf(idea.Key, prefix+"%d", &seq)
							if err == nil && seq > maxSeq {
								maxSeq = seq
							}
						}
					}
					return maxSeq + 1, nil
				},
			}

			// Generate key
			key, err := generateIdeaKey(ctx, mockRepo)

			// Check error expectation
			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check key matches expected
			if !tt.expectedError && key != tt.expectedKey {
				t.Errorf("Expected key %s, got %s", tt.expectedKey, key)
			}
		})
	}
}

// TestIdeaCreate_Success tests successful idea creation
func TestIdeaCreate_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	expectedKey := fmt.Sprintf("I-%s-01", dateStr)

	var capturedIdea *models.Idea

	mockRepo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			return []*models.Idea{}, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			capturedIdea = idea
			idea.ID = 1
			return nil
		},
	}

	// Test basic create
	title := "Test Idea"
	key, err := generateIdeaKey(ctx, mockRepo)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	idea := &models.Idea{
		Key:         key,
		Title:       title,
		CreatedDate: now,
		Status:      models.IdeaStatusNew,
	}

	err = mockRepo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}

	// Verify captured idea
	if capturedIdea == nil {
		t.Fatal("Idea was not captured by mock")
	}
	if capturedIdea.Key != expectedKey {
		t.Errorf("Expected key %s, got %s", expectedKey, capturedIdea.Key)
	}
	if capturedIdea.Title != title {
		t.Errorf("Expected title %s, got %s", title, capturedIdea.Title)
	}
	if capturedIdea.Status != models.IdeaStatusNew {
		t.Errorf("Expected status %s, got %s", models.IdeaStatusNew, capturedIdea.Status)
	}
}

// TestIdeaCreate_WithAllFields tests creating an idea with all optional fields
func TestIdeaCreate_WithAllFields(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	priority := 8
	order := 1
	description := "Test description"
	notes := "Test notes"
	relatedDocs := `["doc1.md","doc2.md"]`
	dependencies := `["I-2026-01-01-01"]`

	var capturedIdea *models.Idea

	mockRepo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			return []*models.Idea{}, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			capturedIdea = idea
			idea.ID = 1
			return nil
		},
	}

	key, _ := generateIdeaKey(ctx, mockRepo)

	idea := &models.Idea{
		Key:          key,
		Title:        "Full Idea",
		Description:  &description,
		CreatedDate:  now,
		Priority:     &priority,
		Order:        &order,
		Notes:        &notes,
		RelatedDocs:  &relatedDocs,
		Dependencies: &dependencies,
		Status:       models.IdeaStatusNew,
	}

	err := mockRepo.Create(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to create idea: %v", err)
	}

	// Verify all fields
	if *capturedIdea.Priority != priority {
		t.Errorf("Expected priority %d, got %d", priority, *capturedIdea.Priority)
	}
	if *capturedIdea.Order != order {
		t.Errorf("Expected order %d, got %d", order, *capturedIdea.Order)
	}
	if *capturedIdea.Description != description {
		t.Errorf("Expected description %s, got %s", description, *capturedIdea.Description)
	}
}

// TestIdeaList_AllIdeas tests listing all ideas
func TestIdeaList_AllIdeas(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	priority1 := 5
	priority2 := 8

	existingIdeas := []*models.Idea{
		{
			ID:          1,
			Key:         "I-2026-01-01-01",
			Title:       "First Idea",
			Priority:    &priority1,
			CreatedDate: now,
			Status:      models.IdeaStatusNew,
		},
		{
			ID:          2,
			Key:         "I-2026-01-01-02",
			Title:       "Second Idea",
			Priority:    &priority2,
			CreatedDate: now,
			Status:      models.IdeaStatusOnHold,
		},
	}

	mockRepo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			return existingIdeas, nil
		},
	}

	ideas, err := mockRepo.List(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list ideas: %v", err)
	}

	if len(ideas) != 2 {
		t.Errorf("Expected 2 ideas, got %d", len(ideas))
	}
}

// TestIdeaList_FilterByStatus tests filtering ideas by status
func TestIdeaList_FilterByStatus(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	existingIdeas := []*models.Idea{
		{
			ID:          1,
			Key:         "I-2026-01-01-01",
			Title:       "New Idea",
			CreatedDate: now,
			Status:      models.IdeaStatusNew,
		},
		{
			ID:          2,
			Key:         "I-2026-01-01-02",
			Title:       "On Hold Idea",
			CreatedDate: now,
			Status:      models.IdeaStatusOnHold,
		},
	}

	mockRepo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			if filter != nil && filter.Status != nil {
				// Filter in mock
				filtered := []*models.Idea{}
				for _, idea := range existingIdeas {
					if idea.Status == *filter.Status {
						filtered = append(filtered, idea)
					}
				}
				return filtered, nil
			}
			return existingIdeas, nil
		},
	}

	// Test filtering by "new" status
	statusNew := models.IdeaStatusNew
	filter := &repository.IdeaFilter{Status: &statusNew}
	ideas, err := mockRepo.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list ideas: %v", err)
	}

	if len(ideas) != 1 {
		t.Errorf("Expected 1 idea with status 'new', got %d", len(ideas))
	}
	if len(ideas) > 0 && ideas[0].Status != models.IdeaStatusNew {
		t.Errorf("Expected status 'new', got '%s'", ideas[0].Status)
	}
}

// TestIdeaGet_Success tests successfully retrieving an idea
func TestIdeaGet_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	priority := 7

	expectedIdea := &models.Idea{
		ID:          1,
		Key:         "I-2026-01-01-01",
		Title:       "Test Idea",
		Priority:    &priority,
		CreatedDate: now,
		Status:      models.IdeaStatusNew,
	}

	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			if key == "I-2026-01-01-01" {
				return expectedIdea, nil
			}
			return nil, fmt.Errorf("idea not found")
		},
	}

	idea, err := mockRepo.GetByKey(ctx, "I-2026-01-01-01")
	if err != nil {
		t.Fatalf("Failed to get idea: %v", err)
	}

	if idea.Key != "I-2026-01-01-01" {
		t.Errorf("Expected key I-2026-01-01-01, got %s", idea.Key)
	}
	if idea.Title != "Test Idea" {
		t.Errorf("Expected title 'Test Idea', got %s", idea.Title)
	}
}

// TestIdeaGet_NotFound tests retrieving a non-existent idea
func TestIdeaGet_NotFound(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return nil, fmt.Errorf("idea not found with key %q", key)
		},
	}

	_, err := mockRepo.GetByKey(ctx, "I-9999-99-99-99")
	if err == nil {
		t.Error("Expected error for non-existent idea, got none")
	}
}

// TestIdeaUpdate_Success tests successfully updating an idea
func TestIdeaUpdate_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	originalPriority := 5
	updatedPriority := 8

	originalIdea := &models.Idea{
		ID:          1,
		Key:         "I-2026-01-01-01",
		Title:       "Original Title",
		Priority:    &originalPriority,
		CreatedDate: now,
		Status:      models.IdeaStatusNew,
	}

	var capturedIdea *models.Idea

	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return originalIdea, nil
		},
		UpdateFunc: func(ctx context.Context, idea *models.Idea) error {
			capturedIdea = idea
			return nil
		},
	}

	// Get and update
	idea, err := mockRepo.GetByKey(ctx, "I-2026-01-01-01")
	if err != nil {
		t.Fatalf("Failed to get idea: %v", err)
	}

	idea.Title = "Updated Title"
	idea.Priority = &updatedPriority
	idea.Status = models.IdeaStatusOnHold

	err = mockRepo.Update(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to update idea: %v", err)
	}

	// Verify captured update
	if capturedIdea.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", capturedIdea.Title)
	}
	if *capturedIdea.Priority != updatedPriority {
		t.Errorf("Expected priority %d, got %d", updatedPriority, *capturedIdea.Priority)
	}
	if capturedIdea.Status != models.IdeaStatusOnHold {
		t.Errorf("Expected status 'on_hold', got %s", capturedIdea.Status)
	}
}

// TestIdeaDelete_HardDelete tests permanently deleting an idea
func TestIdeaDelete_HardDelete(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	existingIdea := &models.Idea{
		ID:          1,
		Key:         "I-2026-01-01-01",
		Title:       "Idea to Delete",
		CreatedDate: now,
		Status:      models.IdeaStatusNew,
	}

	var deletedID int64

	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return existingIdea, nil
		},
		DeleteFunc: func(ctx context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}

	// Get and delete
	idea, err := mockRepo.GetByKey(ctx, "I-2026-01-01-01")
	if err != nil {
		t.Fatalf("Failed to get idea: %v", err)
	}

	err = mockRepo.Delete(ctx, idea.ID)
	if err != nil {
		t.Fatalf("Failed to delete idea: %v", err)
	}

	if deletedID != 1 {
		t.Errorf("Expected deleted ID 1, got %d", deletedID)
	}
}

// TestIdeaDelete_SoftDelete tests archiving an idea (soft delete)
func TestIdeaDelete_SoftDelete(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	existingIdea := &models.Idea{
		ID:          1,
		Key:         "I-2026-01-01-01",
		Title:       "Idea to Archive",
		CreatedDate: now,
		Status:      models.IdeaStatusNew,
	}

	var capturedIdea *models.Idea

	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return existingIdea, nil
		},
		UpdateFunc: func(ctx context.Context, idea *models.Idea) error {
			capturedIdea = idea
			return nil
		},
	}

	// Get and archive
	idea, err := mockRepo.GetByKey(ctx, "I-2026-01-01-01")
	if err != nil {
		t.Fatalf("Failed to get idea: %v", err)
	}

	idea.Status = models.IdeaStatusArchived

	err = mockRepo.Update(ctx, idea)
	if err != nil {
		t.Fatalf("Failed to archive idea: %v", err)
	}

	if capturedIdea.Status != models.IdeaStatusArchived {
		t.Errorf("Expected status 'archived', got %s", capturedIdea.Status)
	}
}

// TestIdeaJSONMarshaling tests JSON marshaling of related docs and dependencies
func TestIdeaJSONMarshaling(t *testing.T) {
	// Test related docs
	relatedDocs := []string{"doc1.md", "doc2.md"}
	docsJSON, err := json.Marshal(relatedDocs)
	if err != nil {
		t.Fatalf("Failed to marshal related docs: %v", err)
	}

	expectedDocsJSON := `["doc1.md","doc2.md"]`
	if string(docsJSON) != expectedDocsJSON {
		t.Errorf("Expected JSON %s, got %s", expectedDocsJSON, string(docsJSON))
	}

	// Test dependencies
	dependencies := []string{"I-2026-01-01-01", "I-2026-01-01-02"}
	depsJSON, err := json.Marshal(dependencies)
	if err != nil {
		t.Fatalf("Failed to marshal dependencies: %v", err)
	}

	expectedDepsJSON := `["I-2026-01-01-01","I-2026-01-01-02"]`
	if string(depsJSON) != expectedDepsJSON {
		t.Errorf("Expected JSON %s, got %s", expectedDepsJSON, string(depsJSON))
	}
}
