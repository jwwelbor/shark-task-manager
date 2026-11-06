package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

type publicationContextValueKey struct{}

type publicationCleanupContextObservation struct {
	err         error
	deadline    time.Time
	hasDeadline bool
	traceValue  any
}

func observePublicationCleanupContext(ctx context.Context) publicationCleanupContextObservation {
	deadline, hasDeadline := ctx.Deadline()
	return publicationCleanupContextObservation{
		err:         ctx.Err(),
		deadline:    deadline,
		hasDeadline: hasDeadline,
		traceValue:  ctx.Value(publicationContextValueKey{}),
	}
}

func assertPublicationCleanupContext(t *testing.T, observation publicationCleanupContextObservation) {
	t.Helper()
	require.NoError(t, observation.err, "cleanup must not inherit request cancellation")
	require.True(t, observation.hasDeadline, "cleanup must be bounded")
	remaining := time.Until(observation.deadline)
	require.LessOrEqual(t, remaining, 5*time.Second, "cleanup must not outlive the configured bound")
	require.Equal(t, "trace-42", observation.traceValue, "cleanup must preserve request-scoped values")
}

func TestBugCreatePublicationFailureRollsBack(t *testing.T) {
	for _, tc := range []struct {
		name  string
		force bool
	}{
		{name: "atomic", force: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, "docs"), []byte("block directory"), 0600))
			row := false
			repo := &mockBugRepo{
				createFn: func(_ context.Context, bug *models.Bug) error {
					row = true
					bug.ID = 42
					return nil
				},
				deleteFn: func(_ context.Context, id int64) error {
					require.EqualValues(t, 42, id)
					row = false
					return nil
				},
			}
			tags := NewMockTagService()
			svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tags)
			svc.projectRoot = root
			index := &mockSearchIndexer{}
			svc.SetSearchIndexer(index)
			_, _, err := svc.CreateBug(context.Background(), CreateBugInput{Title: "failure", Tags: []string{"tag"}, Force: tc.force})
			require.ErrorContains(t, err, "failed to write bug file")
			require.ErrorContains(t, err, "rollback succeeded")
			var pathErr *os.PathError
			require.ErrorAs(t, err, &pathErr)
			require.Contains(t, pathErr.Path, "docs")
			require.False(t, row)
			require.Zero(t, tags.AttachManyCalls)
			require.Empty(t, index.indexCalls)
		})
	}
}

func TestChangeCreatePublicationFailureRollsBack(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs"), []byte("block directory"), 0600))
	row := false
	repo := &mockChangeCardRepo{createFn: func(_ context.Context, c *models.ChangeCard) error { row = true; c.ID = 42; return nil }, deleteFn: func(_ context.Context, id int64) error { require.EqualValues(t, 42, id); row = false; return nil }}
	tags := NewMockTagService()
	svc := newChangeCardServiceWithTagSvc(repo, tags)
	svc.projectRoot = root
	index := &mockSearchIndexer{}
	svc.SetSearchIndexer(index)
	_, _, err := svc.CreateChangeCard(context.Background(), CreateChangeCardInput{Title: "failure", Tags: []string{"tag"}})
	require.ErrorContains(t, err, "failed to write change-card file")
	require.ErrorContains(t, err, "rollback succeeded")
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
	require.Contains(t, pathErr.Path, "docs")
	require.False(t, row)
	require.Zero(t, tags.AttachManyCalls)
	require.Empty(t, index.indexCalls)
}

func TestBugCreatePublicationFailureRollsBackAfterRequestCancellation(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs"), []byte("block directory"), 0600))
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), publicationContextValueKey{}, "trace-42"))
	defer cancel()
	row := false
	var cleanupContext publicationCleanupContextObservation
	repo := &mockBugRepo{
		createFn: func(_ context.Context, bug *models.Bug) error {
			row = true
			bug.ID = 42
			cancel()
			return nil
		},
		deleteFn: func(deleteCtx context.Context, id int64) error {
			require.EqualValues(t, 42, id)
			cleanupContext = observePublicationCleanupContext(deleteCtx)
			if cleanupContext.err != nil {
				return cleanupContext.err
			}
			row = false
			return nil
		},
	}
	svc := newBugService(repo, nil, nil, nil)
	svc.projectRoot = root

	_, _, err := svc.CreateBug(ctx, CreateBugInput{Title: "canceled publication"})

	require.ErrorContains(t, err, "failed to write bug file")
	require.ErrorContains(t, err, "rollback succeeded")
	assertPublicationCleanupContext(t, cleanupContext)
	require.False(t, row, "the new bug row must be removed when publication fails")
}

func TestChangeCardCreatePublicationFailureRollsBackAfterRequestCancellation(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs"), []byte("block directory"), 0600))
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), publicationContextValueKey{}, "trace-42"))
	defer cancel()
	row := false
	var cleanupContext publicationCleanupContextObservation
	repo := &mockChangeCardRepo{
		createFn: func(_ context.Context, card *models.ChangeCard) error {
			row = true
			card.ID = 42
			cancel()
			return nil
		},
		deleteFn: func(deleteCtx context.Context, id int64) error {
			require.EqualValues(t, 42, id)
			cleanupContext = observePublicationCleanupContext(deleteCtx)
			if cleanupContext.err != nil {
				return cleanupContext.err
			}
			row = false
			return nil
		},
	}
	svc := newChangeCardServiceWithTagSvc(repo, nil)
	svc.projectRoot = root

	_, _, err := svc.CreateChangeCard(ctx, CreateChangeCardInput{Title: "canceled publication"})

	require.ErrorContains(t, err, "failed to write change-card file")
	require.ErrorContains(t, err, "rollback succeeded")
	assertPublicationCleanupContext(t, cleanupContext)
	require.False(t, row, "the new change-card row must be removed when publication fails")
}

func TestBugCreatePublicationFailureReportsRollbackFailure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs"), []byte("block directory"), 0600))
	rollbackErr := errors.New("database unavailable")
	repo := &mockBugRepo{
		createFn: func(_ context.Context, bug *models.Bug) error {
			bug.ID = 42
			return nil
		},
		deleteFn: func(context.Context, int64) error { return rollbackErr },
	}
	svc := newBugService(repo, nil, nil, nil)
	svc.projectRoot = root

	_, _, err := svc.CreateBug(context.Background(), CreateBugInput{Title: "failure"})
	require.Error(t, err)
	require.ErrorIs(t, err, rollbackErr)
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
	require.Contains(t, pathErr.Path, "docs")
	require.ErrorContains(t, err, "rollback failed")
}

func TestChangeCardCreatePublicationFailureReportsRollbackFailure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs"), []byte("block directory"), 0600))
	rollbackErr := errors.New("database unavailable")
	repo := &mockChangeCardRepo{
		createFn: func(_ context.Context, card *models.ChangeCard) error {
			card.ID = 42
			return nil
		},
		deleteFn: func(context.Context, int64) error { return rollbackErr },
	}
	svc := newChangeCardServiceWithTagSvc(repo, nil)
	svc.projectRoot = root

	_, _, err := svc.CreateChangeCard(context.Background(), CreateChangeCardInput{Title: "failure"})
	require.Error(t, err)
	require.ErrorIs(t, err, rollbackErr)
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
	require.Contains(t, pathErr.Path, "docs")
	require.ErrorContains(t, err, "rollback failed")
}

func TestBugForcePublicationFailureRemainsBestEffort(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "docs"), []byte("block directory"), 0600))
	row := false
	repo := &mockBugRepo{createFn: func(_ context.Context, bug *models.Bug) error { row = true; bug.ID = 42; return nil }, deleteFn: func(context.Context, int64) error { row = false; return nil }}
	tags := NewMockTagService()
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tags)
	svc.projectRoot = root
	index := &mockSearchIndexer{}
	svc.SetSearchIndexer(index)
	bug, linked, err := svc.CreateBug(context.Background(), CreateBugInput{Title: "failure", Tags: []string{"tag"}, Force: true})
	require.NoError(t, err)
	require.NotNil(t, bug)
	require.False(t, linked)
	require.True(t, row)
	require.Equal(t, 1, tags.AttachManyCalls)
	requireSearchIndexCall(t, index.indexCalls, models.EntityTypeBug, 42)
}

func TestBugCreateSuccessfulPublicationIndexes(t *testing.T) {
	root := t.TempDir()
	repo := &mockBugRepo{createFn: func(_ context.Context, bug *models.Bug) error { bug.ID = 42; return nil }}
	tags := NewMockTagService()
	svc := newBugServiceWithTagSvc(repo, nil, nil, nil, tags)
	svc.projectRoot = root
	index := &mockSearchIndexer{}
	svc.SetSearchIndexer(index)
	bug, linked, err := svc.CreateBug(context.Background(), CreateBugInput{Title: "success", Tags: []string{"tag"}})
	require.NoError(t, err)
	require.False(t, linked)
	_, err = os.ReadFile(filepath.Join(root, *bug.FilePath))
	require.NoError(t, err)
	require.Equal(t, 1, tags.AttachManyCalls)
	requireSearchIndexCall(t, index.indexCalls, models.EntityTypeBug, 42)
}

func TestChangeCreateSuccessfulPublicationIndexes(t *testing.T) {
	root := t.TempDir()
	repo := &mockChangeCardRepo{createFn: func(_ context.Context, card *models.ChangeCard) error { card.ID = 42; return nil }}
	tags := NewMockTagService()
	svc := newChangeCardServiceWithTagSvc(repo, tags)
	svc.projectRoot = root
	index := &mockSearchIndexer{}
	svc.SetSearchIndexer(index)
	card, linked, err := svc.CreateChangeCard(context.Background(), CreateChangeCardInput{Title: "success", Tags: []string{"tag"}})
	require.NoError(t, err)
	require.False(t, linked)
	_, err = os.ReadFile(filepath.Join(root, *card.FilePath))
	require.NoError(t, err)
	require.Equal(t, 1, tags.AttachManyCalls)
	requireSearchIndexCall(t, index.indexCalls, models.EntityTypeChange, 42)
}
