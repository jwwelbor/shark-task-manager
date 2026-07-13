package teamrun_test

import (
	"context"
	"encoding/json"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	teamrunrepo "github.com/jwwelbor/shark-task-manager/internal/repository/teamrun"
	"github.com/jwwelbor/shark-task-manager/internal/team"
	sharktest "github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTC014_PersistedRunResultConversion(t *testing.T) {
	db := sharktest.NewIsolatedTestDB(t)
	repo := teamrunrepo.NewTeamRunRepository(dbconn.NewDB(db))
	started := time.Date(2026, time.July, 13, 15, 0, 0, 0, time.UTC)
	outcome, agent := "passed", "developer"
	evidence, err := json.Marshal(map[string]any{"summary": "artifact recorded", "artifact_refs": []string{"docs/result.md"}})
	require.NoError(t, err)
	run := &teamrunrepo.TeamRun{RootKey: "E38-F01-fixture", RootType: "feature", Status: "completed", ExecutionMode: "parallel", ConcurrencyLimit: 2, PlanHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", AggregateOutcome: &outcome, StartedAt: &started}
	item := &teamrunrepo.TeamRunItem{ChildKey: "T-E38-F01-001", ChildType: "task", Wave: 1, ExecutionOrder: 2, DependencyKeys: `["T-E38-F01-000"]`, PlannedAgentType: &agent, ItemStatus: "completed", Outcome: &outcome, Evidence: stringPtr(string(evidence)), Attempt: 1, StartedAt: &started}
	require.NoError(t, repo.CreateRunWithItems(context.Background(), run, []*teamrunrepo.TeamRunItem{item}))
	result, err := team.NewLedgerService(repo, t.TempDir()).GetRunResult(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, run.ID, result.RunID)
	require.Equal(t, run.PlanHash, result.PlanHash)
	require.Equal(t, team.RunStatusCompleted, result.Status)
	require.Len(t, result.Items, 1)
	require.Equal(t, "artifact recorded", result.Items[0].Evidence)
	require.Equal(t, []string{"docs/result.md"}, result.Items[0].ArtifactRefs)
}

func TestSchedulerPreClaimDiagnosticsLeavePlannedRows_TC003(t *testing.T) {
	db := sharktest.NewIsolatedTestDB(t)
	repo := teamrunrepo.NewTeamRunRepository(dbconn.NewDB(db))
	coordinator := "root-session-001"
	run := &teamrunrepo.TeamRun{
		RootKey: "E38-F02", RootType: "feature", Status: "planned",
		ExecutionMode: "sequential", ConcurrencyLimit: 1,
		PlanHash:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		RootSessionID: &coordinator,
	}
	items := []*teamrunrepo.TeamRunItem{
		{ChildKey: "T-E38-F02-001", ChildType: "task", DependencyKeys: `[]`, ItemStatus: "planned", Attempt: 0},
		{ChildKey: "T-E38-F02-002", ChildType: "task", DependencyKeys: `[]`, ItemStatus: "planned", Attempt: 0},
	}
	require.NoError(t, repo.CreateRunWithItems(context.Background(), run, items))
	ledger := team.NewLedgerService(repo, t.TempDir())

	_, err := ledger.RecordPreClaimResult(context.Background(), team.ItemResultUpdate{
		RunID: run.ID, ItemID: items[0].ID, Attempt: 0, Status: team.ItemStatusBlocked,
		SkipReason: "dependency_not_satisfied",
	}, coordinator)
	require.NoError(t, err)
	_, err = ledger.RecordPreClaimResult(context.Background(), team.ItemResultUpdate{
		RunID: run.ID, ItemID: items[1].ID, Attempt: 0, Status: team.ItemStatusSkipped,
		SkipReason: "unresolved_workflow",
	}, coordinator)
	require.NoError(t, err)

	got, err := repo.ListItems(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, "blocked", got[0].ItemStatus)
	require.NotNil(t, got[0].SkipReason)
	require.Equal(t, "dependency_not_satisfied", *got[0].SkipReason)
	require.Equal(t, "skipped", got[1].ItemStatus)
	require.NotNil(t, got[1].SkipReason)
	require.Equal(t, "unresolved_workflow", *got[1].SkipReason)
}

func stringPtr(value string) *string { return &value }
