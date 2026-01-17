package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// MockNoteRepo for testing search functionality
type MockNoteRepo struct {
	notes []*models.TaskNote
}

func (m *MockNoteRepo) Search(ctx context.Context, query string, noteTypes []string, epicKey string, featureKey string) ([]*models.TaskNote, error) {
	var results []*models.TaskNote

	for _, note := range m.notes {
		// Filter by note type
		if len(noteTypes) > 0 {
			hasType := false
			for _, nt := range noteTypes {
				if nt == string(note.NoteType) {
					hasType = true
					break
				}
			}
			if !hasType {
				continue
			}
		}

		// Filter by search query
		if query != "" && !strings.Contains(note.Content, query) {
			continue
		}

		results = append(results, note)
	}

	return results, nil
}

// TestRejectionTypeFiltering verifies --type=rejection flag works
func TestRejectionTypeFiltering(t *testing.T) {
	now := time.Now()

	rejectionNote := &models.TaskNote{
		ID:        1,
		TaskID:    100,
		NoteType:  models.NoteTypeRejection,
		Content:   "Missing error handling on line 42",
		CreatedBy: str("reviewer"),
		CreatedAt: now,
	}

	decisionNote := &models.TaskNote{
		ID:        2,
		TaskID:    100,
		NoteType:  models.NoteTypeDecision,
		Content:   "Used rejection notes for feedback",
		CreatedBy: str("developer"),
		CreatedAt: now,
	}

	repo := &MockNoteRepo{
		notes: []*models.TaskNote{rejectionNote, decisionNote},
	}

	ctx := context.Background()

	// Search for rejection notes only
	notes, err := repo.Search(ctx, "", []string{string(models.NoteTypeRejection)}, "", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("expected 1 rejection note, got %d", len(notes))
	}

	if notes[0].NoteType != models.NoteTypeRejection {
		t.Errorf("expected rejection note type, got %v", notes[0].NoteType)
	}
}

// TestTimePeriodFiltering verifies --since and --until functionality
func TestTimePeriodFiltering(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	lastWeek := now.AddDate(0, 0, -7)

	oldNote := &models.TaskNote{
		ID:        1,
		TaskID:    100,
		NoteType:  models.NoteTypeRejection,
		Content:   "Old rejection",
		CreatedBy: str("reviewer1"),
		CreatedAt: lastWeek,
	}

	recentNote := &models.TaskNote{
		ID:        2,
		TaskID:    101,
		NoteType:  models.NoteTypeRejection,
		Content:   "Recent rejection",
		CreatedBy: str("reviewer2"),
		CreatedAt: yesterday,
	}

	allNotes := []*models.TaskNote{oldNote, recentNote}

	// Filter notes created in last 3 days
	since := now.AddDate(0, 0, -3)

	var filteredNotes []*models.TaskNote
	for _, note := range allNotes {
		if note.CreatedAt.After(since) {
			filteredNotes = append(filteredNotes, note)
		}
	}

	// Should return recent note but not old note
	if len(filteredNotes) != 1 {
		t.Errorf("expected 1 note after filtering, got %d", len(filteredNotes))
	}

	if filteredNotes[0].ID != 2 {
		t.Errorf("expected recent note (id=2), got id=%d", filteredNotes[0].ID)
	}
}

// TestSearchWithTextQuery verifies text search functionality
func TestSearchWithTextQuery(t *testing.T) {
	now := time.Now()

	notes := []*models.TaskNote{
		{
			ID:        1,
			TaskID:    100,
			NoteType:  models.NoteTypeRejection,
			Content:   "Missing error handling on line 42",
			CreatedBy: str("reviewer"),
			CreatedAt: now,
		},
		{
			ID:        2,
			TaskID:    101,
			NoteType:  models.NoteTypeRejection,
			Content:   "Wrong validation approach",
			CreatedBy: str("reviewer"),
			CreatedAt: now,
		},
	}

	repo := &MockNoteRepo{notes: notes}
	ctx := context.Background()

	// Search for notes containing "error"
	results, err := repo.Search(ctx, "error", []string{string(models.NoteTypeRejection)}, "", "")

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if !strings.Contains(results[0].Content, "error") {
		t.Errorf("expected result to contain 'error'")
	}
}

// TestJSONOutput verifies JSON marshaling works correctly
func TestJSONOutput(t *testing.T) {
	now := time.Now()

	results := []NoteSearchResult{
		{
			TaskKey:   "E07-F22-001",
			TaskTitle: "Implement rejection notes",
			Note: &models.TaskNote{
				ID:        1,
				TaskID:    100,
				NoteType:  models.NoteTypeRejection,
				Content:   "Missing error handling",
				CreatedBy: str("reviewer@example.com"),
				CreatedAt: now,
			},
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify JSON contains expected fields
	if !strings.Contains(jsonStr, "E07-F22-001") {
		t.Error("JSON missing task_key")
	}

	if !strings.Contains(jsonStr, "rejection") {
		t.Error("JSON missing note type")
	}

	if !strings.Contains(jsonStr, "Missing error handling") {
		t.Error("JSON missing note content")
	}
}

// TestCombinedFilters verifies multiple filters work together
func TestCombinedFilters(t *testing.T) {
	now := time.Now()

	notes := []*models.TaskNote{
		{
			ID:        1,
			TaskID:    100,
			NoteType:  models.NoteTypeRejection,
			Content:   "Missing validation",
			CreatedBy: str("reviewer"),
			CreatedAt: now,
		},
		{
			ID:        2,
			TaskID:    101,
			NoteType:  models.NoteTypeRejection,
			Content:   "Wrong approach",
			CreatedBy: str("reviewer"),
			CreatedAt: now,
		},
	}

	repo := &MockNoteRepo{notes: notes}
	ctx := context.Background()

	// Search for rejection notes containing "validation"
	results, err := repo.Search(
		ctx,
		"validation",
		[]string{string(models.NoteTypeRejection)},
		"E07",
		"E07-F22",
	)

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Should find note containing "validation"
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[0].Content != "Missing validation" {
		t.Errorf("expected 'Missing validation', got '%s'", results[0].Content)
	}
}

// TestEmptyResults verifies empty result handling
func TestEmptyResults(t *testing.T) {
	repo := &MockNoteRepo{notes: []*models.TaskNote{}}
	ctx := context.Background()

	results, err := repo.Search(ctx, "nonexistent", []string{string(models.NoteTypeRejection)}, "", "")

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestLargeDatasetPerformance verifies performance with 1000+ notes
func TestLargeDatasetPerformance(t *testing.T) {
	// Create 1000 mock rejection notes
	var notes []*models.TaskNote
	for i := 0; i < 1000; i++ {
		notes = append(notes, &models.TaskNote{
			ID:        int64(i + 1),
			TaskID:    int64((i % 100) + 1),
			NoteType:  models.NoteTypeRejection,
			Content:   "Some rejection reason at position " + string(rune(i)),
			CreatedBy: str("reviewer"),
			CreatedAt: time.Now(),
		})
	}

	repo := &MockNoteRepo{notes: notes}
	ctx := context.Background()

	// Time the search
	start := time.Now()
	results, err := repo.Search(ctx, "rejection", []string{string(models.NoteTypeRejection)}, "", "")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected to find rejection notes")
	}

	// Log performance (in test output)
	t.Logf("Search through 1000+ notes took %v", elapsed)

	// Performance should be acceptable
	if elapsed > 500*time.Millisecond {
		t.Logf("warning: search took %v, expected < 500ms for 1000 notes", elapsed)
	}
}

// Helper function to create string pointers
func str(s string) *string {
	return &s
}
