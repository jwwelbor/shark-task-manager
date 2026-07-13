package team

import (
	"context"
	"errors"
	"strings"
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

func (m *ledgerRepositoryMock) CreateRunWithItemsIfAbsent(_ context.Context, run *teamrunrepo.TeamRun, items []*teamrunrepo.TeamRunItem) (*teamrunrepo.TeamRun, bool, error) {
	if m.findErr != nil {
		return nil, false, m.findErr
	}
	if m.run != nil {
		if m.run.PlanHash != run.PlanHash {
			return cloneRepoRun(m.run), true, nil
		}
		return cloneRepoRun(m.run), true, nil
	}
	if err := m.CreateRunWithItems(context.Background(), run, items); err != nil {
		return nil, false, err
	}
	return cloneRepoRun(run), false, nil
}

func (m *ledgerRepositoryMock) GetRun(context.Context, int64) (*teamrunrepo.TeamRun, error) {
	return cloneRepoRun(m.run), nil
}

func (m *ledgerRepositoryMock) ListItems(context.Context, int64) ([]*teamrunrepo.TeamRunItem, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	items := make([]*teamrunrepo.TeamRunItem, 0, len(m.items))
	for _, item := range m.items {
		copy := *item
		items = append(items, &copy)
	}
	return items, nil
}

func (m *ledgerRepositoryMock) UpdateRun(_ context.Context, run *teamrunrepo.TeamRun) error {
	m.run = cloneRepoRun(run)
	return nil
}

func (m *ledgerRepositoryMock) CompareAndSetItem(_ context.Context, item *teamrunrepo.TeamRunItem, expectedStatus string, expectedAttempt int) (bool, error) {
	if m.updateErr != nil {
		return false, m.updateErr
	}
	for index, existing := range m.items {
		if existing.ID != item.ID || existing.TeamRunID != item.TeamRunID || existing.ItemStatus != expectedStatus || existing.Attempt != expectedAttempt {
			continue
		}
		m.items[index] = item
		m.updatedItems++
		return true, nil
	}
	return false, nil
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
			repo := &ledgerRepositoryMock{run: &teamrunrepo.TeamRun{ID: 1}, items: []*teamrunrepo.TeamRunItem{{ID: int64(index + 1), TeamRunID: 1, ChildKey: "T-E38-F01-001", ChildType: "task", DependencyKeys: `[]`, ItemStatus: string(ItemStatusPlanned)}}}
			ledger := NewLedger(repo)
			got, err := ledger.RecordItemResult(context.Background(), ItemResultUpdate{RunID: 1, ItemID: int64(index + 1), Status: status, Outcome: string(status)})
			if err != nil || got.ItemStatus != status {
				t.Fatalf("result = %+v, error = %v, want status %q", got, err, status)
			}
		})
	}
}

func TestLedger_PersistRejectsRootAndItemIdentityMismatches(t *testing.T) {
	keysByType := map[models.EntityType]string{
		models.EntityTypeEpic:    "E38",
		models.EntityTypeFeature: "E38-F01",
		models.EntityTypeTask:    "T-E38-F01-001",
		models.EntityTypeBug:     "B001",
		models.EntityTypeChange:  "CC-001",
		models.EntityTypeSprint:  "S001",
	}
	for actualType, key := range keysByType {
		for declaredType := range models.ValidEntityTypes {
			if declaredType == actualType {
				continue
			}
			t.Run(string(actualType)+"_as_"+string(declaredType), func(t *testing.T) {
				plan := &TeamPlan{
					RootKey: "E38-F01", RootType: models.EntityTypeFeature,
					ExecutionMode: ExecutionModeSequential, ConcurrencyLimit: 1,
					PlanHash: strings.Repeat("a", 64),
					Items:    []TeamPlanItem{{ChildKey: key, ChildType: declaredType}},
				}
				ledger := NewLedger(&ledgerRepositoryMock{})
				if _, err := ledger.PersistConfirmedPlan(context.Background(), plan, "root-session-001"); err == nil {
					t.Fatalf("PersistConfirmedPlan accepted %s declared as %s", key, declaredType)
				}
			})
		}
	}
}

func TestLedger_ResultLookupRejectsPersistedIdentityMismatch(t *testing.T) {
	repo := &ledgerRepositoryMock{
		run:   &teamrunrepo.TeamRun{ID: 1, RootKey: "E38-F01", RootType: "feature", PlanHash: strings.Repeat("a", 64), Status: "planned", ExecutionMode: "sequential", ConcurrencyLimit: 1},
		items: []*teamrunrepo.TeamRunItem{{ID: 1, TeamRunID: 1, ChildKey: "B001", ChildType: "task", DependencyKeys: `[]`, ItemStatus: "planned"}},
	}
	_, err := NewLedger(repo).RecordItemResult(context.Background(), ItemResultUpdate{RunID: 1, ItemID: 1, Status: ItemStatusCompleted})
	if err == nil || !errors.Is(err, ErrInvalidEntityKey) {
		t.Fatalf("RecordItemResult() error = %v, want persisted identity validation error", err)
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

func TestLedger_ListItems_RejectsMalformedDependencyJSON_TC014(t *testing.T) {
	repo := &ledgerRepositoryMock{items: []*teamrunrepo.TeamRunItem{{
		ID: 1, TeamRunID: 1, ChildKey: "T-E38-F01-001", ChildType: "task",
		DependencyKeys: `[1]`, ItemStatus: string(ItemStatusPlanned),
	}}}
	ledger := NewLedger(repo)

	if _, err := ledger.ListItems(context.Background(), 1); !errors.Is(err, ErrMalformedDependency) {
		t.Fatalf("malformed dependency error = %v, want ErrMalformedDependency", err)
	}
}

func TestLedger_ListItems_RejectsMalformedEvidenceJSON_TC014(t *testing.T) {
	repo := &ledgerRepositoryMock{items: []*teamrunrepo.TeamRunItem{{
		ID: 1, TeamRunID: 1, ChildKey: "T-E38-F01-001", ChildType: "task",
		DependencyKeys: `[]`, Evidence: stringPtr(`{bad`), ItemStatus: string(ItemStatusCompleted),
	}}}
	ledger := NewLedger(repo)

	if _, err := ledger.ListItems(context.Background(), 1); !errors.Is(err, ErrInvalidEvidence) {
		t.Fatalf("malformed evidence error = %v, want ErrInvalidEvidence", err)
	}
}

func TestLedger_RecordItemResult_UsesAllowedArtifactBase_TC009(t *testing.T) {
	projectRoot := t.TempDir()
	repo := &ledgerRepositoryMock{items: []*teamrunrepo.TeamRunItem{{
		ID: 1, TeamRunID: 1, ChildKey: "T-E38-F01-001", ChildType: "task", DependencyKeys: `[]`, ItemStatus: string(ItemStatusPlanned),
	}}}
	ledger := NewLedger(repo, projectRoot)

	result, err := ledger.RecordItemResult(context.Background(), ItemResultUpdate{
		RunID: 1, ItemID: 1, Attempt: 0, Status: ItemStatusCompleted,
		ArtifactRefs: []string{"artifacts/./result.md"},
	})
	if err != nil || len(result.ArtifactRefs) != 1 || result.ArtifactRefs[0] != "artifacts/result.md" {
		t.Fatalf("canonical artifact result = %+v, error = %v", result, err)
	}
	updatesBeforeReject := repo.updatedItems

	_, err = ledger.RecordItemResult(context.Background(), ItemResultUpdate{
		RunID: 1, ItemID: 1, Attempt: 0, Status: ItemStatusCompleted,
		ArtifactRefs: []string{"../../outside.md"},
	})
	if !errors.Is(err, ErrInvalidArtifactPath) || repo.updatedItems != updatesBeforeReject {
		t.Fatalf("outside-base artifact error = %v, updates=%d", err, repo.updatedItems)
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
