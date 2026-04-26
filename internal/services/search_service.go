package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// SearchRepository defines data access for cross-entity search.
type SearchRepository interface {
	SearchAll(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error)
}

// SearchService provides cross-entity full-text search with optional tag
// post-filtering (AND intersection).
type SearchService struct {
	repo   SearchRepository
	tagSvc TagQuerier // optional; nil disables --tag on search (REQ-F-011)
}

// NewSearchService constructs a SearchService. tagSvc is optional; pass
// nil to disable tag filtering.
func NewSearchService(repo SearchRepository, tagSvc TagQuerier) *SearchService {
	return &SearchService{repo: repo, tagSvc: tagSvc}
}

// SearchAll searches across all entity types. Empty entityType means no type
// filter. When len(tags) > 0, results are post-filtered to only rows whose
// (entity_type, id) is in the AND-intersection of the given tag names per
// entity-type bucket (REQ-F-011, REQ-NF-002).
//
// Algorithm (spec §2.6.2):
//  1. Call repo.SearchAll → raw results.
//  2. If len(tags) == 0 → return raw results unchanged.
//  3. If tagSvc == nil → return *TagFilterUnavailableError (AC-T3).
//  4. Bucket raw results by entity_type.
//  5. For each bucket, call tagSvc.EntityIDsByTags once → tagged-ID set.
//  6. Walk results; keep rows whose (EntityType, ID) is in the set.
//  7. Return filtered slice (non-nil, possibly empty).
func (s *SearchService) SearchAll(ctx context.Context, query string, entityType string, tags []string) ([]*repository.EntitySearchResult, error) {
	var entityTypePtr *string
	if entityType != "" {
		entityTypePtr = &entityType
	}

	results, err := s.repo.SearchAll(ctx, query, entityTypePtr)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Step 2: no tag filter → return unchanged.
	if len(tags) == 0 {
		return results, nil
	}

	// Step 3: tag filter requested but service not wired.
	if s.tagSvc == nil {
		return nil, &TagFilterUnavailableError{}
	}

	// Step 4: bucket results by entity_type to avoid N+1 calls.
	byType := map[string][]*repository.EntitySearchResult{}
	for _, r := range results {
		byType[r.EntityType] = append(byType[r.EntityType], r)
	}

	// Step 4b (B014): when FTS returned zero rows the byType map is empty and
	// the per-bucket loop below never executes, silently skipping tag-name
	// validation. To preserve the eager-validation contract (unregistered tags
	// must always produce *UnregisteredTagError), run a sentinel validation
	// call using models.EntityTypeTask when there are no buckets to iterate.
	if len(byType) == 0 {
		if _, idErr := s.tagSvc.EntityIDsByTags(ctx, models.EntityTypeTask, tags, TagQueryOpAnd); idErr != nil {
			return nil, idErr
		}
		// All tag names are valid; return empty non-nil slice.
		return make([]*repository.EntitySearchResult, 0), nil
	}

	// Step 5: for each entity-type bucket, fetch the full tagged-ID set once.
	taggedIDs := map[string]map[int64]struct{}{} // entityType → set of IDs
	for et := range byType {
		ids, idErr := s.tagSvc.EntityIDsByTags(ctx, models.EntityType(et), tags, TagQueryOpAnd)
		if idErr != nil {
			return nil, idErr
		}
		set := make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			set[id] = struct{}{}
		}
		taggedIDs[et] = set
	}

	// Step 6: walk results and keep only rows in the tagged-ID sets.
	filtered := make([]*repository.EntitySearchResult, 0)
	for _, r := range results {
		if set, ok := taggedIDs[r.EntityType]; ok {
			if _, inSet := set[r.ID]; inSet {
				filtered = append(filtered, r)
			}
		}
	}

	return filtered, nil
}
