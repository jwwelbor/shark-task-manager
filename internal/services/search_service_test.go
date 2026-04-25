package services

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
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

// ---------------------------------------------------------------------------
// Existing tests — updated for new SearchAll(ctx, query, entityType, tags) sig
// ---------------------------------------------------------------------------

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

	svc := NewSearchService(mock, nil)
	got, err := svc.SearchAll(context.Background(), "login", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SearchAll() mismatch:\ngot:  %+v\nwant: %+v", got, want)
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

	svc := NewSearchService(mock, nil)
	got, err := svc.SearchAll(context.Background(), "login", "bug", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SearchAll() mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestSearchService_SearchAll_EmptyQuery(t *testing.T) {
	mock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return []*repository.EntitySearchResult{}, nil
		},
	}

	svc := NewSearchService(mock, nil)
	got, err := svc.SearchAll(context.Background(), "", "", nil)
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

	svc := NewSearchService(mock, nil)
	_, err := svc.SearchAll(context.Background(), "anything", "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected wrapped repoErr, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// New tag post-filter tests — AC-17, AC-17b, AC-18, AC-19, AC-19b, AC-19c
// ---------------------------------------------------------------------------

// AC-17: tag filter reduces results; only rows whose (entity_type, id) is in
// the tagged-ID set are kept.
func TestSearchAll_TagFilterReducesResults(t *testing.T) {
	// Three raw results: two tasks and one bug.
	// After tag filter "voice": only task id=1 survives.
	task1ID := int64(1)
	task2ID := int64(2)
	bugID := int64(10)

	rawResults := []*repository.EntitySearchResult{
		{EntityType: "task", ID: task1ID, Key: "E07-F01-001", Title: "login endpoint", Status: "todo"},
		{EntityType: "task", ID: task2ID, Key: "E07-F01-002", Title: "fix login", Status: "in_progress"},
		{EntityType: "bug", ID: bugID, Key: "B001", Title: "login broken", Status: "triage"},
	}

	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return rawResults, nil
		},
	}

	entityIDsByTagsCalls := 0
	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
		entityIDsByTagsCalls++
		switch entityType {
		case models.EntityTypeTask:
			return []int64{task1ID}, nil // only task1 has tag "voice"
		case models.EntityTypeBug:
			return []int64{}, nil // no bugs have tag "voice"
		default:
			t.Errorf("unexpected entityType %q in EntityIDsByTags", entityType)
			return nil, nil
		}
	})

	svc := NewSearchService(searchMock, tagSvc)
	got, err := svc.SearchAll(context.Background(), "login", "", []string{"voice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Errorf("expected 1 result after tag filter, got %d: %+v", len(got), got)
	}
	if len(got) > 0 && got[0].Key != "E07-F01-001" {
		t.Errorf("expected E07-F01-001, got %q", got[0].Key)
	}
	// AC-T1: EntityIDsByTags called at most once per entity type (2 types: task + bug)
	if entityIDsByTagsCalls != 2 {
		t.Errorf("expected 2 EntityIDsByTags calls (one per entity-type bucket), got %d", entityIDsByTagsCalls)
	}
}

// AC-17b: EntityIDsByTags is called exactly once per distinct entity type,
// NOT once per result row.
func TestSearchAll_TagFilterCalledPerEntityTypeBucket(t *testing.T) {
	// 3 distinct entity types in results (epic, task, bug)
	rawResults := []*repository.EntitySearchResult{
		{EntityType: "epic", ID: 1, Key: "E07", Title: "my epic", Status: "active"},
		{EntityType: "task", ID: 10, Key: "E07-F01-001", Title: "task one", Status: "todo"},
		{EntityType: "task", ID: 11, Key: "E07-F01-002", Title: "task two", Status: "todo"},
		{EntityType: "bug", ID: 20, Key: "B001", Title: "some bug", Status: "triage"},
		{EntityType: "bug", ID: 21, Key: "B002", Title: "another bug", Status: "triage"},
	}

	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return rawResults, nil
		},
	}

	var callsByType = map[string]int{}
	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
		callsByType[string(entityType)]++
		return []int64{}, nil // no matches; we only care about call count
	})

	svc := NewSearchService(searchMock, tagSvc)
	_, err := svc.SearchAll(context.Background(), "query", "", []string{"voice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Exactly 3 entity types → exactly 3 EntityIDsByTags calls
	if len(callsByType) != 3 {
		t.Errorf("expected 3 distinct EntityIDsByTags calls, got %d: %v", len(callsByType), callsByType)
	}
	for _, et := range []string{"epic", "task", "bug"} {
		if callsByType[et] != 1 {
			t.Errorf("expected 1 call for entity type %q, got %d", et, callsByType[et])
		}
	}
}

// AC-18: when all EntityIDsByTags calls return empty slices, SearchAll returns
// an empty non-nil slice with no error.
func TestSearchAll_TagFilterAllZeroMatchesReturnsEmpty(t *testing.T) {
	rawResults := []*repository.EntitySearchResult{
		{EntityType: "task", ID: 1, Key: "E07-F01-001", Title: "task one", Status: "todo"},
		{EntityType: "bug", ID: 10, Key: "B001", Title: "bug one", Status: "triage"},
	}

	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return rawResults, nil
		},
	}

	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
		return []int64{}, nil // all zero matches
	})

	svc := NewSearchService(searchMock, tagSvc)
	got, err := svc.SearchAll(context.Background(), "anything", "", []string{"voice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 results when all tag filters return empty, got %d", len(got))
	}
}

// AC-19: when EntityIDsByTags returns *UnregisteredTagError, SearchAll propagates it.
func TestSearchAll_TagFilterUnregisteredPropagates(t *testing.T) {
	rawResults := []*repository.EntitySearchResult{
		{EntityType: "task", ID: 1, Key: "E07-F01-001", Title: "task one", Status: "todo"},
	}

	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return rawResults, nil
		},
	}

	unregErr := &UnregisteredTagError{Name: "does-not-exist"}
	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
		return nil, unregErr
	})

	svc := NewSearchService(searchMock, tagSvc)
	got, err := svc.SearchAll(context.Background(), "anything", "", []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil results on error, got %v", got)
	}

	var unregTagErr *UnregisteredTagError
	if !errors.As(err, &unregTagErr) {
		t.Errorf("expected *UnregisteredTagError, got %T: %v", err, err)
	}
}

// AC-19b: when tagSvc is nil and tags are requested, no panic occurs;
// returns *TagFilterUnavailableError.
// (Per AC-T3 in task spec: tagSvc==nil with non-empty tags → TagFilterUnavailableError)
func TestSearchAll_NilTagSvcIgnoresTagFilter(t *testing.T) {
	rawResults := []*repository.EntitySearchResult{
		{EntityType: "task", ID: 1, Key: "E07-F01-001", Title: "task one", Status: "todo"},
	}

	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return rawResults, nil
		},
	}

	// tagSvc is nil — simulates un-wired service
	svc := NewSearchService(searchMock, nil)
	got, err := svc.SearchAll(context.Background(), "anything", "", []string{"voice"})

	// With nil tagSvc + non-empty tags, should return TagFilterUnavailableError (AC-T3)
	if err == nil {
		t.Fatal("expected TagFilterUnavailableError, got nil error")
	}
	if got != nil {
		t.Errorf("expected nil results on error, got %v", got)
	}
	var unavailErr *TagFilterUnavailableError
	if !errors.As(err, &unavailErr) {
		t.Errorf("expected *TagFilterUnavailableError, got %T: %v", err, err)
	}
}

// AC-19d (B014): when FTS returns zero rows, tag validation must still fire.
// An unregistered tag must return *UnregisteredTagError regardless of FTS hit count.
func TestSearchService_SearchAll_UnregisteredTag_WithEmptyFTS(t *testing.T) {
	// FTS returns no results at all — the byType bucket map will be empty.
	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return []*repository.EntitySearchResult{}, nil // zero FTS hits
		},
	}

	unregErr := &UnregisteredTagError{Name: "bogus"}
	tagSvc := NewMockTagService().WithEntityIDsByTagsFn(func(ctx context.Context, entityType models.EntityType, names []string, op TagQueryOp) ([]int64, error) {
		return nil, unregErr
	})

	svc := NewSearchService(searchMock, tagSvc)
	got, err := svc.SearchAll(context.Background(), "qwertyuiop-no-such-string", "", []string{"bogus"})
	if err == nil {
		t.Fatal("expected *UnregisteredTagError, got nil error (B014: tag validation skipped on empty FTS)")
	}
	if got != nil {
		t.Errorf("expected nil results on error, got %v", got)
	}

	var unregTagErr *UnregisteredTagError
	if !errors.As(err, &unregTagErr) {
		t.Errorf("expected *UnregisteredTagError, got %T: %v", err, err)
	}
}

// AC-T3-emptyFTS (B014): when tagSvc is nil and FTS returns zero rows,
// *TagFilterUnavailableError must still be returned (not silent empty result).
func TestSearchService_SearchAll_NilTagSvc_WithEmptyFTS(t *testing.T) {
	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return []*repository.EntitySearchResult{}, nil // zero FTS hits
		},
	}

	// tagSvc is nil — simulates un-wired service
	svc := NewSearchService(searchMock, nil)
	got, err := svc.SearchAll(context.Background(), "no-matches", "", []string{"voice"})
	if err == nil {
		t.Fatal("expected *TagFilterUnavailableError, got nil error (B014: AC-T3 check skipped on empty FTS)")
	}
	if got != nil {
		t.Errorf("expected nil results on error, got %v", got)
	}
	var unavailErr *TagFilterUnavailableError
	if !errors.As(err, &unavailErr) {
		t.Errorf("expected *TagFilterUnavailableError, got %T: %v", err, err)
	}
}

// AC-19c: when tags is nil/empty, EntityIDsByTags is never called and results
// are returned unchanged.
func TestSearchAll_EmptyTagsNoExtraCall(t *testing.T) {
	rawResults := []*repository.EntitySearchResult{
		{EntityType: "task", ID: 1, Key: "E07-F01-001", Title: "task one", Status: "todo"},
		{EntityType: "bug", ID: 10, Key: "B001", Title: "bug one", Status: "triage"},
	}

	searchMock := &mockSearchRepository{
		SearchAllFunc: func(ctx context.Context, query string, entityType *string) ([]*repository.EntitySearchResult, error) {
			return rawResults, nil
		},
	}

	tagSvc := NewMockTagService() // no fn overrides; call counter starts at 0

	svc := NewSearchService(searchMock, tagSvc)
	got, err := svc.SearchAll(context.Background(), "anything", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Results returned unchanged
	if !reflect.DeepEqual(got, rawResults) {
		t.Errorf("expected rawResults unchanged, got %v", got)
	}
	// EntityIDsByTags must NOT have been called
	if tagSvc.EntityIDsByTagsCalls != 0 {
		t.Errorf("expected 0 EntityIDsByTags calls with nil tags, got %d", tagSvc.EntityIDsByTagsCalls)
	}
}
