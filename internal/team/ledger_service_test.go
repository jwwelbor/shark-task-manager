package team

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	teamrunrepo "github.com/jwwelbor/shark-task-manager/internal/repository/teamrun"
)

type ledgerRepositoryMock struct {
	run          *teamrunrepo.TeamRun
	items        []*teamrunrepo.TeamRunItem
	createdRuns  int
	updatedItems int
	findErr      error
	listErr      error
	createErr    error
	updateErr    error
}

func (m *ledgerRepositoryMock) FindRunByRoot(context.Context, string, string) (*teamrunrepo.TeamRun, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if m.run == nil {
		return nil, ErrRepositoryNotFound
	}
	return cloneRepoRun(m.run), nil
}

func (m *ledgerRepositoryMock) CreateRunWithItems(_ context.Context, run *teamrunrepo.TeamRun, items []*teamrunrepo.TeamRunItem) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdRuns++
	run.ID = int64(m.createdRuns)
	m.run = cloneRepoRun(run)
	m.run.ID = run.ID
	m.items = items
	for index, item := range items {
		item.ID = int64(index + 1)
		item.TeamRunID = run.ID
	}
	return nil
}

func (m *ledgerRepositoryMock) GetRun(context.Context, int64) (*teamrunrepo.TeamRun, error) {
	return cloneRepoRun(m.run), nil
}

func (m *ledgerRepositoryMock) ListItems(context.Context, int64) ([]*teamrunrepo.TeamRunItem, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *ledgerRepositoryMock) UpdateRun(_ context.Context, run *teamrunrepo.TeamRun) error {
	m.run = cloneRepoRun(run)
	return nil
}

func (m *ledgerRepositoryMock) UpdateItem(_ context.Context, item *teamrunrepo.TeamRunItem) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updatedItems++
	for index, existing := range m.items {
		if existing.ID == item.ID {
			m.items[index] = item
			return nil
		}
	}
	return errors.New("item not found")
}

func TestLedger_ContextCancellationAndWrappedErrors_TC015(t *testing.T) {
	repoError := errors.New("repository unavailable")
	ledger := NewLedger(&ledgerRepositoryMock{findErr: repoError})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ledger.PersistConfirmedPlan(ctx, ledgerTestPlan(), "root-session-001"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled persistence error = %v, want context.Canceled", err)
	}

	ledger = NewLedger(&ledgerRepositoryMock{findErr: repoError})
	if _, err := ledger.PersistConfirmedPlan(context.Background(), ledgerTestPlan(), "root-session-001"); !errors.Is(err, repoError) {
		t.Fatalf("wrapped persistence error = %v, want repository error", err)
	}
}

func TestLedger_RecordItemResult_AllTerminalOutcomes_TC007(t *testing.T) {
	statuses := []ItemStatus{ItemStatusCompleted, ItemStatusFailed, ItemStatusBlocked, ItemStatusPaused, ItemStatusSkipped, ItemStatusCancelled}
	for index, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			repo := &ledgerRepositoryMock{run: &teamrunrepo.TeamRun{ID: 1}, items: []*teamrunrepo.TeamRunItem{{ID: int64(index + 1), TeamRunID: 1, ChildKey: "T-E38-F01-001", ChildType: "task", ItemStatus: string(ItemStatusPlanned)}}}
			ledger := NewLedger(repo)
			got, err := ledger.RecordItemResult(context.Background(), ItemResultUpdate{RunID: 1, ItemID: int64(index + 1), Status: status, Outcome: string(status)})
			if err != nil || got.ItemStatus != status {
				t.Fatalf("result = %+v, error = %v, want status %q", got, err, status)
			}
		})
	}
}

func TestLedger_Idempotency_TC007(t *testing.T) {
	repo := &ledgerRepositoryMock{}
	ledger := NewLedger(repo)
	plan := ledgerTestPlan()

	first, err := ledger.PersistConfirmedPlan(context.Background(), plan, "root-session-001")
	if err != nil {
		t.Fatalf("first persistence error = %v", err)
	}
	second, err := ledger.PersistConfirmedPlan(context.Background(), plan, "root-session-001")
	if err != nil {
		t.Fatalf("idempotent persistence error = %v", err)
	}
	if first.ID != second.ID || repo.createdRuns != 1 {
		t.Fatalf("idempotent persistence created duplicate: first=%+v second=%+v runs=%d", first, second, repo.createdRuns)
	}

	drifted := ledgerTestPlan()
	drifted.PlanHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := ledger.PersistConfirmedPlan(context.Background(), drifted, "root-session-001"); !errors.Is(err, ErrPlanDrift) {
		t.Fatalf("drift error = %v, want ErrPlanDrift", err)
	}

	completed := ItemResultUpdate{RunID: first.ID, ItemID: 1, Attempt: 0, Status: ItemStatusCompleted, Outcome: "passed", Evidence: "done", ArtifactRefs: []string{"artifacts/result.md"}}
	result, err := ledger.RecordItemResult(context.Background(), completed)
	if err != nil {
		t.Fatalf("terminal result error = %v", err)
	}
	repeated, err := ledger.RecordItemResult(context.Background(), completed)
	if err != nil || repeated.Attempt != result.Attempt || repo.updatedItems != 1 {
		t.Fatalf("terminal repeat = result=%+v err=%v updates=%d", repeated, err, repo.updatedItems)
	}

	conflict := completed
	conflict.Status = ItemStatusFailed
	if _, err := ledger.RecordItemResult(context.Background(), conflict); !errors.Is(err, ErrConflictingTerminalResult) {
		t.Fatalf("conflict error = %v, want ErrConflictingTerminalResult", err)
	}

	retry := completed
	retry.Attempt = 1
	retry.ExplicitRetry = true
	retry.Outcome = "failed"
	updated, err := ledger.RecordItemResult(context.Background(), retry)
	if err != nil || updated.Attempt != 1 || repo.updatedItems != 2 {
		t.Fatalf("explicit retry = result=%+v err=%v updates=%d", updated, err, repo.updatedItems)
	}
}

func TestLedger_RejectsInvalidResultBeforePersistence_TC015(t *testing.T) {
	repo := &ledgerRepositoryMock{run: &teamrunrepo.TeamRun{ID: 1}, items: []*teamrunrepo.TeamRunItem{{ID: 1, TeamRunID: 1, ChildKey: "T-E38-F01-001", ChildType: "task", ItemStatus: "planned"}}}
	ledger := NewLedger(repo)

	_, err := ledger.RecordItemResult(context.Background(), ItemResultUpdate{RunID: 1, ItemID: 1, Attempt: -1, Status: ItemStatusCompleted})
	if !errors.Is(err, ErrInvalidAttempt) || repo.updatedItems != 0 {
		t.Fatalf("invalid result error = %v, updates=%d", err, repo.updatedItems)
	}
}

func ledgerTestPlan() *TeamPlan {
	return &TeamPlan{
		RootKey:          "E38-F01",
		RootType:         models.EntityTypeFeature,
		ExecutionMode:    ExecutionModeSequential,
		ConcurrencyLimit: 1,
		PlanHash:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Items: []TeamPlanItem{{
			ChildKey: "T-E38-F01-001", ChildType: models.EntityTypeTask,
			ExecutionOrder: 1, Wave: 0, Planned: DispatchMetadata{AgentType: "developer"}, Eligible: true,
		}},
	}
}

func cloneRepoRun(run *teamrunrepo.TeamRun) *teamrunrepo.TeamRun {
	if run == nil {
		return nil
	}
	copy := *run
	return &copy
}
