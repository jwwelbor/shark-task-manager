package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// SearchRepository defines the repository interface needed by SearchService.
// This interface is satisfied by *repository.SearchRepository.
type SearchRepository interface {
	// SearchAll performs a LIKE-based full-text search across all entity types.
	// If entityType is non-nil and non-empty, results are filtered to that type only.
	SearchAll(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error)
}

// SearchService provides business logic for cross-entity search operations.
// It wraps the search repository and can apply additional filtering or
// transformation logic as needed.
type SearchService struct {
	repo SearchRepository
}

// NewSearchService creates a new SearchService with the given repository.
func NewSearchService(repo SearchRepository) *SearchService {
	return &SearchService{repo: repo}
}

// SearchAll searches across all entity types (epics, features, tasks, bugs, change-cards).
// If entityType is non-empty, results are filtered to that entity type only.
// An empty query returns an empty result set.
func (s *SearchService) SearchAll(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
	results, err := s.repo.SearchAll(ctx, query, entityType)
	if err != nil {
		return nil, fmt.Errorf("search failed for query %q: %w", query, err)
	}
	return results, nil
}
