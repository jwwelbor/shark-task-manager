package services

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// mockSearchRepository is a test double for SearchRepository.
type mockSearchRepository struct {
	SearchAllFunc func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error)
}

func (m *mockSearchRepository) SearchAll(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
	if m.SearchAllFunc != nil {
		return m.SearchAllFunc(ctx, query, entityType)
	}
	return nil, errors.New("SearchAll not implemented in mock")
}

func TestSearchService_SearchAll_NoFilter(t *testing.T) {
	want := []*repository.EntitySearchResult{
		{EntityType: "task", Key: "E01-F01-001", Title: "login endpoint", Status: "todo"},
		{EntityType: "epic", Key: "E01", Title: "login feature", Status: "in_progress"},
	}

	mock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			if query != "login" {
				t.Errorf("unexpected query %q", query)
			}
			if entityType != nil {
				t.Errorf("expected nil entityType, got %q", *entityType)
			}
			return want, nil
		},
	}

	svc := NewSearchService(mock)
	got, err := svc.SearchAll(context.Background(), "login", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d results, got %d", len(want), len(got))
	}
	for i, r := range got {
		if r.Key != want[i].Key {
			t.Errorf("result[%d].Key = %q, want %q", i, r.Key, want[i].Key)
		}
	}
}

func TestSearchService_SearchAll_WithEntityTypeFilter(t *testing.T) {
	want := []*repository.EntitySearchResult{
		{EntityType: "bug", Key: "B001", Title: "login crash", Status: "triage"},
	}

	mock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			if entityType == nil || *entityType != "bug" {
				t.Errorf("expected entityType=bug, got %v", entityType)
			}
			return want, nil
		},
	}

	svc := NewSearchService(mock)
	got, err := svc.SearchAll(context.Background(), "login", "bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Key != "B001" {
		t.Errorf("expected key B001, got %q", got[0].Key)
	}
}

func TestSearchService_SearchAll_EmptyQuery(t *testing.T) {
	mock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return []*repository.EntitySearchResult{}, nil
		},
	}

	svc := NewSearchService(mock)
	got, err := svc.SearchAll(context.Background(), "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(got))
	}
}

func TestSearchService_SearchAll_RepoError(t *testing.T) {
	repoErr := errors.New("database unavailable")
	mock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return nil, repoErr
		},
	}

	svc := NewSearchService(mock)
	_, err := svc.SearchAll(context.Background(), "anything", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected wrapped repoErr, got: %v", err)
	}
}
