package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// SearchRepository defines data access for cross-entity search.
type SearchRepository interface {
	SearchAll(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error)
}

// SearchService provides cross-entity full-text search.
type SearchService struct {
	repo SearchRepository
}

// NewSearchService creates a SearchService with the given repository.
func NewSearchService(repo SearchRepository) *SearchService {
	return &SearchService{repo: repo}
}

// SearchAll searches across all entity types. Empty entityType means no filter.
func (s *SearchService) SearchAll(ctx context.Context, query string, entityType string) ([]*repository.EntitySearchResult, error) {
	var entityTypePtr *string
	if entityType != "" {
		entityTypePtr = &entityType
	}

	results, err := s.repo.SearchAll(ctx, query, entityTypePtr)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return results, nil
}
