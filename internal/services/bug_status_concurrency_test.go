package services

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/require"
)

type staleBugUpdateRepo struct{ BugRepository }

func (r staleBugUpdateRepo) UpdateIfStatus(_ context.Context, _ *models.Bug, _ models.BugStatus) (bool, error) {
	return false, nil // The repository observed a concurrent status change.
}

func TestBugUpdateReportsConcurrentStatusChange(t *testing.T) {
	repo := &mockBugRepo{getByKeyFn: func(context.Context, string) (*models.Bug, error) {
		return &models.Bug{BaseEntity: models.BaseEntity{Key: "B001", Title: "fixture"}, Status: "plan_review", Severity: models.BugSeverityMedium}, nil
	}}
	wf := workflow.NewService(t.TempDir())
	svc := NewBugService(staleBugUpdateRepo{repo}, NewEntityService(wf), nil, nil, nil, nil, "", nil)
	title := "new title"
	_, err := svc.UpdateBug(context.Background(), "B001", BugUpdates{Title: &title})
	require.ErrorIs(t, err, ErrAdvanceGuardStaleFromStatus)
}

func TestBugTriageReportsConcurrentStatusChange(t *testing.T) {
	repo := &mockBugRepo{getByKeyFn: func(context.Context, string) (*models.Bug, error) {
		return &models.Bug{BaseEntity: models.BaseEntity{Key: "B001", Title: "fixture"}, Status: "plan_review", Severity: models.BugSeverityMedium}, nil
	}}
	wf := workflow.NewService(t.TempDir())
	svc := NewBugService(staleBugUpdateRepo{repo}, NewEntityService(wf), nil, nil, nil, nil, "", nil)
	_, err := svc.TriageBug(context.Background(), "B001", TriageBugInput{Severity: "high"})
	require.ErrorIs(t, err, ErrAdvanceGuardStaleFromStatus)
}
