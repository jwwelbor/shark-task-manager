package services

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/require"
)

type concurrentBugUpdateRepo struct{ BugRepository }

func (r concurrentBugUpdateRepo) UpdateIfStatus(ctx context.Context, bug *models.Bug, expected models.BugStatus) (bool, error) {
	if err := r.BugRepository.UpdateStatus(ctx, bug.ID, "repair_plan"); err != nil {
		return false, err
	}
	return r.BugRepository.UpdateIfStatus(ctx, bug, expected)
}

func TestBugUpdateDoesNotRestoreConcurrentStatus(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewBugRepository(repository.NewDB(test.NewIsolatedTestDB(t)))
	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: "B001", Title: "fixture"}, Status: "plan_review", Severity: models.BugSeverityMedium}
	require.NoError(t, repo.Create(ctx, bug))
	svc := &BugService{repo: concurrentBugUpdateRepo{repo}}
	title := "new title"
	_, err := svc.UpdateBug(ctx, "B001", BugUpdates{Title: &title})
	require.ErrorIs(t, err, ErrAdvanceGuardStaleFromStatus)
	got, err := repo.GetByKey(ctx, "B001")
	require.NoError(t, err)
	require.Equal(t, models.BugStatus("repair_plan"), got.Status)
	require.Equal(t, "fixture", got.Title)
}

func TestBugTriageDoesNotRestoreConcurrentStatus(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewBugRepository(repository.NewDB(test.NewIsolatedTestDB(t)))
	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: "B001", Title: "fixture"}, Status: "plan_review", Severity: models.BugSeverityMedium}
	require.NoError(t, repo.Create(ctx, bug))
	wf := workflow.NewService(t.TempDir())
	svc := NewBugService(concurrentBugUpdateRepo{repo}, NewEntityService(wf), nil, nil, nil, nil, "", nil)
	_, err := svc.TriageBug(ctx, "B001", TriageBugInput{Severity: "high"})
	require.ErrorIs(t, err, ErrAdvanceGuardStaleFromStatus)
	got, err := repo.GetByKey(ctx, "B001")
	require.NoError(t, err)
	require.Equal(t, models.BugStatus("repair_plan"), got.Status)
	require.Equal(t, models.BugSeverityMedium, got.Severity)
}
