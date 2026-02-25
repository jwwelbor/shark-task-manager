package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// MockIdeaRepository is a mock implementation of IdeaRepository for testing.
type MockIdeaRepository struct {
	CreateFunc                 func(ctx context.Context, idea *models.Idea) error
	GetByIDFunc                func(ctx context.Context, id int64) (*models.Idea, error)
	GetByKeyFunc               func(ctx context.Context, key string) (*models.Idea, error)
	UpdateFunc                 func(ctx context.Context, idea *models.Idea) error
	DeleteFunc                 func(ctx context.Context, id int64) error
	ListFunc                   func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error)
	MarkAsConvertedFunc        func(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
	GetNextSequenceForDateFunc func(ctx context.Context, dateStr string) (int, error)
}

func (m *MockIdeaRepository) Create(ctx context.Context, idea *models.Idea) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, idea)
	}
	return fmt.Errorf("Create not implemented in mock")
}

func (m *MockIdeaRepository) GetByID(ctx context.Context, id int64) (*models.Idea, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("GetByID not implemented in mock")
}

func (m *MockIdeaRepository) GetByKey(ctx context.Context, key string) (*models.Idea, error) {
	if m.GetByKeyFunc != nil {
		return m.GetByKeyFunc(ctx, key)
	}
	return nil, fmt.Errorf("GetByKey not implemented in mock")
}

func (m *MockIdeaRepository) Update(ctx context.Context, idea *models.Idea) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, idea)
	}
	return fmt.Errorf("Update not implemented in mock")
}

func (m *MockIdeaRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return fmt.Errorf("Delete not implemented in mock")
}

func (m *MockIdeaRepository) List(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	return nil, fmt.Errorf("List not implemented in mock")
}

func (m *MockIdeaRepository) MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
	if m.MarkAsConvertedFunc != nil {
		return m.MarkAsConvertedFunc(ctx, ideaID, convertedToType, convertedToKey)
	}
	return fmt.Errorf("MarkAsConverted not implemented in mock")
}

func (m *MockIdeaRepository) GetNextSequenceForDate(ctx context.Context, dateStr string) (int, error) {
	if m.GetNextSequenceForDateFunc != nil {
		return m.GetNextSequenceForDateFunc(ctx, dateStr)
	}
	return 0, fmt.Errorf("GetNextSequenceForDate not implemented in mock")
}

// TC-IDEA-001: CreateIdea generates correct key format
func TestIdeaService_CreateIdea_GeneratesKey(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	mockRepo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			if dateStr != today {
				t.Errorf("expected dateStr %q, got %q", today, dateStr)
			}
			return 1, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			expectedKey := fmt.Sprintf("I-%s-01", today)
			if idea.Key != expectedKey {
				t.Errorf("expected key %q, got %q", expectedKey, idea.Key)
			}
			if idea.Status != models.IdeaStatusNew {
				t.Errorf("expected status %q, got %q", models.IdeaStatusNew, idea.Status)
			}
			idea.ID = 42
			return nil
		},
	}

	svc := NewIdeaService(mockRepo)
	idea, err := svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Test Idea"})

	if err != nil {
		t.Fatalf("CreateIdea() unexpected error: %v", err)
	}
	if idea.ID != 42 {
		t.Errorf("expected ID 42, got %d", idea.ID)
	}
	expectedKey := fmt.Sprintf("I-%s-01", today)
	if idea.Key != expectedKey {
		t.Errorf("expected key %q, got %q", expectedKey, idea.Key)
	}
}

// TC-IDEA-002: CreateIdea sets sequence number as 02 for second idea on same day
func TestIdeaService_CreateIdea_SequenceIncrement(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	callCount := 0
	mockRepo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			callCount++
			return callCount, nil // First call returns 1, second returns 2
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			idea.ID = int64(callCount)
			return nil
		},
	}

	svc := NewIdeaService(mockRepo)

	idea1, err := svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Idea 1"})
	if err != nil {
		t.Fatalf("first CreateIdea() unexpected error: %v", err)
	}
	if idea1.Key != fmt.Sprintf("I-%s-01", today) {
		t.Errorf("expected key I-%s-01, got %s", today, idea1.Key)
	}

	idea2, err := svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Idea 2"})
	if err != nil {
		t.Fatalf("second CreateIdea() unexpected error: %v", err)
	}
	if idea2.Key != fmt.Sprintf("I-%s-02", today) {
		t.Errorf("expected key I-%s-02, got %s", today, idea2.Key)
	}
}

// TC-IDEA-003: CreateIdea returns error for empty title
func TestIdeaService_CreateIdea_EmptyTitle(t *testing.T) {
	mockRepo := &MockIdeaRepository{}
	svc := NewIdeaService(mockRepo)

	_, err := svc.CreateIdea(context.Background(), CreateIdeaInput{Title: ""})
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Errorf("expected 'title is required' in error, got: %v", err)
	}
}

// TC-IDEA-004: CreateIdea returns error when repository fails
func TestIdeaService_CreateIdea_RepositoryFailure(t *testing.T) {
	mockRepo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(ctx context.Context, idea *models.Idea) error {
			return fmt.Errorf("database connection failed")
		},
	}

	svc := NewIdeaService(mockRepo)
	_, err := svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Test"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create idea") {
		t.Errorf("expected 'failed to create idea' in error, got: %v", err)
	}
}

// TC-IDEA-005: GetIdea returns idea for valid key
func TestIdeaService_GetIdea_HappyPath(t *testing.T) {
	expectedKey := "I-2026-01-15-01"
	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			if key != expectedKey {
				t.Errorf("expected key %q, got %q", expectedKey, key)
			}
			return &models.Idea{
				ID:     1,
				Key:    key,
				Title:  "Test Idea",
				Status: models.IdeaStatusNew,
			}, nil
		},
	}

	svc := NewIdeaService(mockRepo)
	idea, err := svc.GetIdea(context.Background(), expectedKey)

	if err != nil {
		t.Fatalf("GetIdea() unexpected error: %v", err)
	}
	if idea.Key != expectedKey {
		t.Errorf("expected key %q, got %q", expectedKey, idea.Key)
	}
}

// TC-IDEA-006: GetIdea returns error for non-existent idea
func TestIdeaService_GetIdea_NotFound(t *testing.T) {
	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return nil, fmt.Errorf("idea not found with key %q", key)
		},
	}

	svc := NewIdeaService(mockRepo)
	_, err := svc.GetIdea(context.Background(), "I-2026-01-15-99")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TC-IDEA-007: ListIdeas with no filter returns all ideas
func TestIdeaService_ListIdeas_NoFilter(t *testing.T) {
	ideas := []*models.Idea{
		{ID: 1, Key: "I-2026-01-15-01", Title: "Idea 1", Status: models.IdeaStatusNew},
		{ID: 2, Key: "I-2026-01-15-02", Title: "Idea 2", Status: models.IdeaStatusArchived},
	}
	mockRepo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			if filter != nil {
				t.Error("expected nil filter for unfiltered list")
			}
			return ideas, nil
		},
	}

	svc := NewIdeaService(mockRepo)
	result, err := svc.ListIdeas(context.Background(), IdeaFilters{})

	if err != nil {
		t.Fatalf("ListIdeas() unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 ideas, got %d", len(result))
	}
}

// TC-IDEA-008: ListIdeas with status filter passes filter to repository
func TestIdeaService_ListIdeas_StatusFilter(t *testing.T) {
	mockRepo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			if filter == nil {
				t.Fatal("expected non-nil filter")
			}
			if filter.Status == nil {
				t.Fatal("expected non-nil Status in filter")
			}
			if *filter.Status != models.IdeaStatusNew {
				t.Errorf("expected status %q, got %q", models.IdeaStatusNew, *filter.Status)
			}
			return []*models.Idea{
				{ID: 1, Key: "I-2026-01-15-01", Title: "New Idea", Status: models.IdeaStatusNew},
			}, nil
		},
	}

	svc := NewIdeaService(mockRepo)
	result, err := svc.ListIdeas(context.Background(), IdeaFilters{Status: "new"})

	if err != nil {
		t.Fatalf("ListIdeas() unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 idea, got %d", len(result))
	}
}

// TC-IDEA-009: UpdateIdea updates specified fields
func TestIdeaService_UpdateIdea_HappyPath(t *testing.T) {
	original := &models.Idea{
		ID:     1,
		Key:    "I-2026-01-15-01",
		Title:  "Original Title",
		Status: models.IdeaStatusNew,
	}
	newTitle := "Updated Title"

	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return original, nil
		},
		UpdateFunc: func(ctx context.Context, idea *models.Idea) error {
			if idea.Title != newTitle {
				t.Errorf("expected title %q, got %q", newTitle, idea.Title)
			}
			return nil
		},
	}

	svc := NewIdeaService(mockRepo)
	updated, err := svc.UpdateIdea(context.Background(), "I-2026-01-15-01", UpdateIdeaInput{
		Title: &newTitle,
	})

	if err != nil {
		t.Fatalf("UpdateIdea() unexpected error: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, updated.Title)
	}
}

// TC-IDEA-010: UpdateIdea returns error when idea not found
func TestIdeaService_UpdateIdea_NotFound(t *testing.T) {
	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return nil, fmt.Errorf("idea not found with key %q", key)
		},
	}

	svc := NewIdeaService(mockRepo)
	title := "New Title"
	_, err := svc.UpdateIdea(context.Background(), "I-2026-01-15-99", UpdateIdeaInput{Title: &title})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TC-IDEA-011: DeleteIdea deletes by key
func TestIdeaService_DeleteIdea_HappyPath(t *testing.T) {
	deletedID := int64(0)
	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 5, Key: key, Title: "To Delete", Status: models.IdeaStatusNew}, nil
		},
		DeleteFunc: func(ctx context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}

	svc := NewIdeaService(mockRepo)
	err := svc.DeleteIdea(context.Background(), "I-2026-01-15-01")

	if err != nil {
		t.Fatalf("DeleteIdea() unexpected error: %v", err)
	}
	if deletedID != 5 {
		t.Errorf("expected deletedID 5, got %d", deletedID)
	}
}

// TC-IDEA-012: ConvertIdea marks idea as converted
func TestIdeaService_ConvertIdea_HappyPath(t *testing.T) {
	var capturedType, capturedKey string
	var capturedID int64

	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 3, Key: key, Title: "Convert Me", Status: models.IdeaStatusNew}, nil
		},
		MarkAsConvertedFunc: func(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error {
			capturedID = ideaID
			capturedType = convertedToType
			capturedKey = convertedToKey
			return nil
		},
	}

	svc := NewIdeaService(mockRepo)
	err := svc.ConvertIdea(context.Background(), "I-2026-01-15-01", "epic", "E15")

	if err != nil {
		t.Fatalf("ConvertIdea() unexpected error: %v", err)
	}
	if capturedID != 3 {
		t.Errorf("expected ID 3, got %d", capturedID)
	}
	if capturedType != "epic" {
		t.Errorf("expected type 'epic', got %q", capturedType)
	}
	if capturedKey != "E15" {
		t.Errorf("expected key 'E15', got %q", capturedKey)
	}
}

// TC-IDEA-013: ConvertIdea returns error if already converted
func TestIdeaService_ConvertIdea_AlreadyConverted(t *testing.T) {
	mockRepo := &MockIdeaRepository{
		GetByKeyFunc: func(ctx context.Context, key string) (*models.Idea, error) {
			return &models.Idea{
				ID:     3,
				Key:    key,
				Title:  "Already Converted",
				Status: models.IdeaStatusConverted, // Already converted
			}, nil
		},
	}

	svc := NewIdeaService(mockRepo)
	err := svc.ConvertIdea(context.Background(), "I-2026-01-15-01", "feature", "E15-F01")

	if err == nil {
		t.Fatal("expected error for already-converted idea, got nil")
	}
	if !strings.Contains(err.Error(), "already converted") {
		t.Errorf("expected 'already converted' in error, got: %v", err)
	}
}

// TC-IDEA-014: Table-driven test for key generation with different sequences
func TestIdeaService_CreateIdea_KeySequences(t *testing.T) {
	tests := []struct {
		name           string
		sequence       int
		expectedSuffix string
	}{
		{"sequence 1", 1, "01"},
		{"sequence 9", 9, "09"},
		{"sequence 10", 10, "10"},
		{"sequence 99", 99, "99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := tt.sequence
			today := time.Now().Format("2006-01-02")
			expectedKey := fmt.Sprintf("I-%s-%s", today, tt.expectedSuffix)

			mockRepo := &MockIdeaRepository{
				GetNextSequenceForDateFunc: func(ctx context.Context, dateStr string) (int, error) {
					return seq, nil
				},
				CreateFunc: func(ctx context.Context, idea *models.Idea) error {
					if idea.Key != expectedKey {
						t.Errorf("expected key %q, got %q", expectedKey, idea.Key)
					}
					idea.ID = 1
					return nil
				},
			}

			svc := NewIdeaService(mockRepo)
			idea, err := svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Test"})
			if err != nil {
				t.Fatalf("CreateIdea() unexpected error: %v", err)
			}
			if idea.Key != expectedKey {
				t.Errorf("expected key %q, got %q", expectedKey, idea.Key)
			}
		})
	}
}

// Test that NewIdeaService panics on nil repo
func TestNewIdeaService_PanicsOnNilRepo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil repo, but no panic occurred")
		}
	}()
	NewIdeaService(nil)
}

// Test ListIdeas returns empty slice not nil when repository returns nil
func TestIdeaService_ListIdeas_ReturnsEmptySlice(t *testing.T) {
	mockRepo := &MockIdeaRepository{
		ListFunc: func(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error) {
			return nil, nil // Repository returns nil
		},
	}

	svc := NewIdeaService(mockRepo)
	result, err := svc.ListIdeas(context.Background(), IdeaFilters{})

	if err != nil {
		t.Fatalf("ListIdeas() unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 ideas, got %d", len(result))
	}
}
