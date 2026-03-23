package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// mockSearchRepository implements SearchRepository for testing.
type mockSearchRepository struct {
	searchAllFunc func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error)
}

func (m *mockSearchRepository) SearchAll(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
	if m.searchAllFunc != nil {
		return m.searchAllFunc(ctx, query, entityType)
	}
	return nil, fmt.Errorf("SearchAll not implemented in mock")
}

func TestSearchService_SearchAll_ReturnsResults(t *testing.T) {
	expected := []*repository.EntitySearchResult{
		{EntityType: "task", Key: "E07-F01-001", Title: "Implement auth", Status: "todo"},
		{EntityType: "epic", Key: "E07", Title: "Auth Epic", Status: "in_progress"},
	}

	mockRepo := &mockSearchRepository{
		searchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			if query != "auth" {
				t.Errorf("expected query 'auth', got %q", query)
			}
			if entityType != nil {
				t.Errorf("expected nil entityType, got %q", *entityType)
			}
			return expected, nil
		},
	}

	svc := NewSearchService(mockRepo)
	results, err := svc.SearchAll(context.Background(), "auth", nil)
	if err != nil {
		t.Fatalf("SearchAll() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if results[0].Key != "E07-F01-001" {
		t.Errorf("expected first result key E07-F01-001, got %s", results[0].Key)
	}
}

func TestSearchService_SearchAll_WithEntityTypeFilter(t *testing.T) {
	entityType := "task"
	expected := []*repository.EntitySearchResult{
		{EntityType: "task", Key: "E07-F01-001", Title: "Implement auth", Status: "todo"},
	}

	mockRepo := &mockSearchRepository{
		searchAllFunc: func(ctx context.Context, query string, entityTypePtr *string) ([]*repository.EntitySearchResult, error) {
			if entityTypePtr == nil {
				t.Error("expected non-nil entityType, got nil")
			} else if *entityTypePtr != "task" {
				t.Errorf("expected entityType 'task', got %q", *entityTypePtr)
			}
			return expected, nil
		},
	}

	svc := NewSearchService(mockRepo)
	results, err := svc.SearchAll(context.Background(), "auth", &entityType)
	if err != nil {
		t.Fatalf("SearchAll() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearchService_SearchAll_PropagatesRepositoryError(t *testing.T) {
	mockRepo := &mockSearchRepository{
		searchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}

	svc := NewSearchService(mockRepo)
	results, err := svc.SearchAll(context.Background(), "auth", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if results != nil {
		t.Errorf("expected nil results on error, got %v", results)
	}
}

func TestSearchService_SearchAll_EmptyResults(t *testing.T) {
	mockRepo := &mockSearchRepository{
		searchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return []*repository.EntitySearchResult{}, nil
		},
	}

	svc := NewSearchService(mockRepo)
	results, err := svc.SearchAll(context.Background(), "nonexistent", nil)
	if err != nil {
		t.Fatalf("SearchAll() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
