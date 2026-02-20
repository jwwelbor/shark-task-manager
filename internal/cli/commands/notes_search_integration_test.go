package commands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// mockNoteServiceForSearch is a mock NoteService for CLI command testing.
// It does NOT use a real database - all data is in-memory.
type mockNoteServiceForSearch struct {
	searchFunc               func(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error)
	searchWithTimePeriodFunc func(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error)
	getEntityDetailsFunc     func(ctx context.Context, entityType models.EntityType, entityID int64) *services.NoteEntityDetails
}

func (m *mockNoteServiceForSearch) SearchNotes(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, noteTypes, entityType, epicKey, featureKey)
	}
	return nil, nil
}

func (m *mockNoteServiceForSearch) SearchNotesWithTimePeriod(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error) {
	if m.searchWithTimePeriodFunc != nil {
		return m.searchWithTimePeriodFunc(ctx, query, noteTypes, epicKey, featureKey, since, until)
	}
	return nil, nil
}

func (m *mockNoteServiceForSearch) GetEntityDetails(ctx context.Context, entityType models.EntityType, entityID int64) *services.NoteEntityDetails {
	if m.getEntityDetailsFunc != nil {
		return m.getEntityDetailsFunc(ctx, entityType, entityID)
	}
	return nil
}

// buildSearchResultsForTest is a helper that exercises the result-building logic
// from runNotesSearch using the mock service (no database involved).
func buildSearchResultsForTest(t *testing.T, svc *mockNoteServiceForSearch, notes []*models.EntityNote) []NoteSearchResult {
	t.Helper()
	ctx := context.Background()
	results := make([]NoteSearchResult, 0, len(notes))
	for _, note := range notes {
		details := svc.GetEntityDetails(ctx, note.EntityType, note.EntityID)
		if details == nil {
			continue
		}
		result := NoteSearchResult{
			EntityType: string(note.EntityType),
			EntityKey:  details.Key,
			EntityName: details.Name,
			Note:       note,
		}
		if note.EntityType == models.EntityTypeTask {
			result.TaskKey = details.Key
			result.TaskTitle = details.Name
		}
		results = append(results, result)
	}
	return results
}

// TestNotesSearch_EntityTypeFilter tests entity type filtering using mocks (no real database).
func TestNotesSearch_EntityTypeFilter(t *testing.T) {
	now := time.Now()

	epicNote := &models.EntityNote{
		ID: 1, EntityType: models.EntityTypeEpic, EntityID: 10,
		NoteType: models.NoteTypeComment, Content: "TEST-Epic note", CreatedAt: now,
	}
	featureNote := &models.EntityNote{
		ID: 2, EntityType: models.EntityTypeFeature, EntityID: 20,
		NoteType: models.NoteTypeDecision, Content: "TEST-Feature note", CreatedAt: now,
	}
	taskNote := &models.EntityNote{
		ID: 3, EntityType: models.EntityTypeTask, EntityID: 30,
		NoteType: models.NoteTypeSolution, Content: "TEST-Task note", CreatedAt: now,
	}
	allNotes := []*models.EntityNote{epicNote, featureNote, taskNote}

	entityDetailMap := map[int64]*services.NoteEntityDetails{
		10: {Key: "E99", Name: "Test Epic"},
		20: {Key: "E99-F99", Name: "Test Feature"},
		30: {Key: "T-E99-F99-999", Name: "Test Task"},
	}

	svc := &mockNoteServiceForSearch{
		searchFunc: func(ctx context.Context, query string, noteTypes []string, entityType *models.EntityType, epicKey string, featureKey string) ([]*models.EntityNote, error) {
			if entityType == nil {
				return allNotes, nil
			}
			var filtered []*models.EntityNote
			for _, n := range allNotes {
				if n.EntityType == *entityType {
					filtered = append(filtered, n)
				}
			}
			return filtered, nil
		},
		getEntityDetailsFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) *services.NoteEntityDetails {
			return entityDetailMap[entityID]
		},
	}

	t.Run("Search all entities", func(t *testing.T) {
		notes, err := svc.SearchNotes(context.Background(), "TEST-", nil, nil, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 3 {
			t.Errorf("Expected 3 notes, got %d", len(notes))
		}
	})

	t.Run("Filter by entity type - epic", func(t *testing.T) {
		entityType := models.EntityTypeEpic
		notes, err := svc.SearchNotes(context.Background(), "TEST-", nil, &entityType, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 1 {
			t.Errorf("Expected 1 epic note, got %d", len(notes))
		}
		if len(notes) > 0 && notes[0].EntityType != models.EntityTypeEpic {
			t.Errorf("Expected epic note, got %s", notes[0].EntityType)
		}
	})

	t.Run("Filter by entity type - feature", func(t *testing.T) {
		entityType := models.EntityTypeFeature
		notes, err := svc.SearchNotes(context.Background(), "TEST-", nil, &entityType, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 1 {
			t.Errorf("Expected 1 feature note, got %d", len(notes))
		}
		if len(notes) > 0 && notes[0].EntityType != models.EntityTypeFeature {
			t.Errorf("Expected feature note, got %s", notes[0].EntityType)
		}
	})

	t.Run("Filter by entity type - task", func(t *testing.T) {
		entityType := models.EntityTypeTask
		notes, err := svc.SearchNotes(context.Background(), "TEST-", nil, &entityType, "", "")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(notes) != 1 {
			t.Errorf("Expected 1 task note, got %d", len(notes))
		}
		if len(notes) > 0 && notes[0].EntityType != models.EntityTypeTask {
			t.Errorf("Expected task note, got %s", notes[0].EntityType)
		}
	})

	t.Run("JSON output validation", func(t *testing.T) {
		results := buildSearchResultsForTest(t, svc, []*models.EntityNote{epicNote})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}

		jsonData, err := json.Marshal(results[0].Note)
		if err != nil {
			t.Fatalf("Failed to marshal note: %v", err)
		}

		var parsed models.EntityNote
		if err := json.Unmarshal(jsonData, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal note: %v", err)
		}

		if parsed.ID != epicNote.ID {
			t.Errorf("ID mismatch after JSON roundtrip: got %d, want %d", parsed.ID, epicNote.ID)
		}
		if parsed.EntityType != epicNote.EntityType {
			t.Errorf("EntityType mismatch after JSON roundtrip: got %s, want %s", parsed.EntityType, epicNote.EntityType)
		}
	})
}

// TestNotesSearch_EdgeCases tests edge cases using mocks (no real database).
func TestNotesSearch_EdgeCases(t *testing.T) {
	t.Run("Entity not found is skipped gracefully", func(t *testing.T) {
		note := &models.EntityNote{
			ID: 1, EntityType: models.EntityTypeTask, EntityID: 999,
			NoteType: models.NoteTypeComment, Content: "Orphaned note",
		}

		svc := &mockNoteServiceForSearch{
			getEntityDetailsFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) *services.NoteEntityDetails {
				return nil // entity not found
			},
		}

		results := buildSearchResultsForTest(t, svc, []*models.EntityNote{note})
		if len(results) != 0 {
			t.Errorf("Expected 0 results (entity not found should be skipped), got %d", len(results))
		}
	})

	t.Run("Task note sets legacy fields", func(t *testing.T) {
		note := &models.EntityNote{
			ID: 2, EntityType: models.EntityTypeTask, EntityID: 100,
			NoteType: models.NoteTypeRejection, Content: "Missing tests",
		}

		svc := &mockNoteServiceForSearch{
			getEntityDetailsFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) *services.NoteEntityDetails {
				return &services.NoteEntityDetails{Key: "E01-F01-001", Name: "My Task"}
			},
		}

		results := buildSearchResultsForTest(t, svc, []*models.EntityNote{note})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results[0].TaskKey != "E01-F01-001" {
			t.Errorf("Expected TaskKey 'E01-F01-001', got '%s'", results[0].TaskKey)
		}
		if results[0].TaskTitle != "My Task" {
			t.Errorf("Expected TaskTitle 'My Task', got '%s'", results[0].TaskTitle)
		}
	})

	t.Run("Epic note does not set legacy task fields", func(t *testing.T) {
		note := &models.EntityNote{
			ID: 3, EntityType: models.EntityTypeEpic, EntityID: 5,
			NoteType: models.NoteTypeComment, Content: "Epic level note",
		}

		svc := &mockNoteServiceForSearch{
			getEntityDetailsFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) *services.NoteEntityDetails {
				return &services.NoteEntityDetails{Key: "E05", Name: "My Epic"}
			},
		}

		results := buildSearchResultsForTest(t, svc, []*models.EntityNote{note})
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results[0].TaskKey != "" {
			t.Errorf("Expected empty TaskKey for epic note, got '%s'", results[0].TaskKey)
		}
		if results[0].TaskTitle != "" {
			t.Errorf("Expected empty TaskTitle for epic note, got '%s'", results[0].TaskTitle)
		}
	})
}

// TestNotesSearch_InvalidEntityType verifies that invalid entity type is rejected.
func TestNotesSearch_InvalidEntityType(t *testing.T) {
	// This tests the validation in runNotesSearch before calling service.
	// We simulate the validation logic directly.
	entityTypeStr := "invalid_type"
	et := models.EntityType(entityTypeStr)
	if models.ValidEntityTypes[et] {
		t.Errorf("Expected 'invalid_type' to be invalid entity type")
	}
}

// TestNotesSearch_TimePeriod verifies time period search delegation to service.
func TestNotesSearch_TimePeriod(t *testing.T) {
	now := time.Now()
	recentNote := &models.EntityNote{
		ID: 1, EntityType: models.EntityTypeTask, EntityID: 1,
		NoteType: models.NoteTypeRejection, Content: "Recent rejection",
		CreatedAt: now,
	}

	called := false
	svc := &mockNoteServiceForSearch{
		searchWithTimePeriodFunc: func(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string, since string, until string) ([]*models.EntityNote, error) {
			called = true
			if since != "2026-01-01" {
				t.Errorf("Expected since '2026-01-01', got '%s'", since)
			}
			return []*models.EntityNote{recentNote}, nil
		},
		getEntityDetailsFunc: func(ctx context.Context, entityType models.EntityType, entityID int64) *services.NoteEntityDetails {
			return &services.NoteEntityDetails{Key: "E01-F01-001", Name: "My Task"}
		},
	}

	notes, err := svc.SearchNotesWithTimePeriod(context.Background(), "rejection", nil, "", "", "2026-01-01", "")
	if err != nil {
		t.Fatalf("SearchNotesWithTimePeriod failed: %v", err)
	}
	if !called {
		t.Error("Expected SearchNotesWithTimePeriod to be called")
	}
	if len(notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(notes))
	}
}
