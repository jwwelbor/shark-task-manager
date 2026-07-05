package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type searchIndexCall struct {
	entityType models.EntityType
	entityID   int64
}

type mockSearchIndexer struct {
	indexErr    error
	removeErr   error
	indexCalls  []searchIndexCall
	removeCalls []searchIndexCall
}

func (m *mockSearchIndexer) IndexEntity(_ context.Context, entityType models.EntityType, entityID int64) error {
	m.indexCalls = append(m.indexCalls, searchIndexCall{entityType: entityType, entityID: entityID})
	return m.indexErr
}

func (m *mockSearchIndexer) RemoveEntity(_ context.Context, entityType models.EntityType, entityID int64) error {
	m.removeCalls = append(m.removeCalls, searchIndexCall{entityType: entityType, entityID: entityID})
	return m.removeErr
}

func requireSearchIndexCall(t *testing.T, calls []searchIndexCall, entityType models.EntityType, entityID int64) {
	t.Helper()
	require.Len(t, calls, 1)
	assert.Equal(t, entityType, calls[0].entityType)
	assert.Equal(t, entityID, calls[0].entityID)
}

func TestNoteService_AddNoteIndexesParentAfterCreate(t *testing.T) {
	indexer := &mockSearchIndexer{}
	noteRepo := &mockNoteEntityNoteRepo{
		createFunc: func(_ context.Context, note *models.EntityNote) error {
			note.ID = 99
			return nil
		},
	}
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	_, err = svc.AddNote(context.Background(), models.EntityTypeTask, "E01-F01-001", "comment", "Index the parent", "dev")
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.indexCalls, models.EntityTypeTask, 3)
	assert.Empty(t, indexer.removeCalls)
}

func TestNoteService_AddNoteDoesNotIndexWhenCreateFails(t *testing.T) {
	indexer := &mockSearchIndexer{}
	noteRepo := &mockNoteEntityNoteRepo{
		createFunc: func(_ context.Context, _ *models.EntityNote) error {
			return fmt.Errorf("insert failed")
		},
	}
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	_, err = svc.AddNote(context.Background(), models.EntityTypeTask, "E01-F01-001", "comment", "Do not index", "dev")
	require.Error(t, err)

	assert.Empty(t, indexer.indexCalls)
	assert.Empty(t, indexer.removeCalls)
}

func TestNoteService_AddNoteReturnsSuccessWhenIndexingFailsAfterCreate(t *testing.T) {
	indexer := &mockSearchIndexer{indexErr: fmt.Errorf("fts temporarily unavailable")}
	noteRepo := &mockNoteEntityNoteRepo{
		createFunc: func(_ context.Context, note *models.EntityNote) error {
			note.ID = 99
			return nil
		},
	}
	svc, err := NewNoteService(noteRepo, newNoteTestRegistry())
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	_, err = svc.AddNote(context.Background(), models.EntityTypeTask, "E01-F01-001", "comment", "Index later", "dev")
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.indexCalls, models.EntityTypeTask, 3)
}

func TestIdeaService_CreateIdeaIndexesAfterPersist(t *testing.T) {
	indexer := &mockSearchIndexer{}
	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(_ context.Context, _ string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(_ context.Context, idea *models.Idea) error {
			idea.ID = 42
			return nil
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	_, err = svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Indexable idea"})
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.indexCalls, models.EntityTypeIdea, 42)
}

func TestIdeaService_CreateIdeaReturnsSuccessWhenIndexingFailsAfterPersist(t *testing.T) {
	indexer := &mockSearchIndexer{indexErr: fmt.Errorf("fts temporarily unavailable")}
	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(_ context.Context, _ string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(_ context.Context, idea *models.Idea) error {
			idea.ID = 42
			return nil
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	_, err = svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Indexable idea"})
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.indexCalls, models.EntityTypeIdea, 42)
}

func TestIdeaService_CreateIdeaDoesNotIndexWhenRepositoryFails(t *testing.T) {
	indexer := &mockSearchIndexer{}
	repo := &MockIdeaRepository{
		GetNextSequenceForDateFunc: func(_ context.Context, _ string) (int, error) {
			return 1, nil
		},
		CreateFunc: func(_ context.Context, _ *models.Idea) error {
			return fmt.Errorf("insert failed")
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	_, err = svc.CreateIdea(context.Background(), CreateIdeaInput{Title: "Do not index"})
	require.Error(t, err)

	assert.Empty(t, indexer.indexCalls)
}

func TestIdeaService_UpdateIdeaIndexesAfterPersist(t *testing.T) {
	indexer := &mockSearchIndexer{}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 43, Key: key, Title: "Old title", Status: models.IdeaStatusNew}, nil
		},
		UpdateFunc: func(_ context.Context, idea *models.Idea) error {
			assert.Equal(t, "New title", idea.Title)
			return nil
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)
	title := "New title"

	_, err = svc.UpdateIdea(context.Background(), "I-2026-07-05-01", UpdateIdeaInput{Title: &title})
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.indexCalls, models.EntityTypeIdea, 43)
}

func TestIdeaService_UpdateIdeaDoesNotIndexWhenRepositoryFails(t *testing.T) {
	indexer := &mockSearchIndexer{}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 44, Key: key, Title: "Old title", Status: models.IdeaStatusNew}, nil
		},
		UpdateFunc: func(_ context.Context, _ *models.Idea) error {
			return fmt.Errorf("update failed")
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)
	title := "New title"

	_, err = svc.UpdateIdea(context.Background(), "I-2026-07-05-02", UpdateIdeaInput{Title: &title})
	require.Error(t, err)

	assert.Empty(t, indexer.indexCalls)
}

func TestIdeaService_DeleteIdeaRemovesIndexAfterPersist(t *testing.T) {
	indexer := &mockSearchIndexer{}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 45, Key: key, Title: "Deleted idea", Status: models.IdeaStatusNew}, nil
		},
		DeleteFunc: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(45), id)
			return nil
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	err = svc.DeleteIdea(context.Background(), "I-2026-07-05-03")
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.removeCalls, models.EntityTypeIdea, 45)
	assert.Empty(t, indexer.indexCalls)
}

func TestIdeaService_DeleteIdeaReturnsSuccessWhenIndexRemoveFailsAfterPersist(t *testing.T) {
	indexer := &mockSearchIndexer{removeErr: fmt.Errorf("fts temporarily unavailable")}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 45, Key: key, Title: "Deleted idea", Status: models.IdeaStatusNew}, nil
		},
		DeleteFunc: func(_ context.Context, id int64) error {
			assert.Equal(t, int64(45), id)
			return nil
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	err = svc.DeleteIdea(context.Background(), "I-2026-07-05-03")
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.removeCalls, models.EntityTypeIdea, 45)
	assert.Empty(t, indexer.indexCalls)
}

func TestIdeaService_DeleteIdeaDoesNotRemoveIndexWhenRepositoryFails(t *testing.T) {
	indexer := &mockSearchIndexer{}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 46, Key: key, Title: "Not deleted", Status: models.IdeaStatusNew}, nil
		},
		DeleteFunc: func(_ context.Context, _ int64) error {
			return fmt.Errorf("delete failed")
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	err = svc.DeleteIdea(context.Background(), "I-2026-07-05-04")
	require.Error(t, err)

	assert.Empty(t, indexer.removeCalls)
}

func TestIdeaService_ConvertIdeaIndexesAfterPersist(t *testing.T) {
	indexer := &mockSearchIndexer{}
	repo := &MockIdeaRepository{
		GetByKeyFunc: func(_ context.Context, key string) (*models.Idea, error) {
			return &models.Idea{ID: 47, Key: key, Title: "Convertible", Status: models.IdeaStatusNew}, nil
		},
		MarkAsConvertedFunc: func(_ context.Context, ideaID int64, convertedToType, convertedToKey string) error {
			assert.Equal(t, int64(47), ideaID)
			assert.Equal(t, "task", convertedToType)
			assert.Equal(t, "E01-F01-001", convertedToKey)
			return nil
		},
	}
	svc, err := NewIdeaService(repo)
	require.NoError(t, err)
	svc.SetSearchIndexer(indexer)

	err = svc.ConvertIdea(context.Background(), "I-2026-07-05-05", "task", "E01-F01-001")
	require.NoError(t, err)

	requireSearchIndexCall(t, indexer.indexCalls, models.EntityTypeIdea, 47)
}
