package teamrun

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/require"
)

func testRun() *TeamRun {
	return &TeamRun{
		RootKey:          "E38-F01",
		RootType:         "feature",
		Status:           "planned",
		ExecutionMode:    "sequential",
		ConcurrencyLimit: 1,
		PlanHash:         "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}

func testItem(key string, order int) *TeamRunItem {
	return &TeamRunItem{
		ChildKey:       key,
		ChildType:      "task",
		Wave:           order / 2,
		ExecutionOrder: order,
		DependencyKeys: "[]",
		ItemStatus:     "planned",
		Attempt:        0,
	}
}

func TestRepository_CreateRunWithItems_IsAtomicAndOrdered_TC006(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()

	run := testRun()
	items := []*TeamRunItem{
		testItem("T-E38-F01-001", 1),
		testItem("T-E38-F01-002", 2),
		testItem("T-E38-F01-002", 3),
		testItem("T-E38-F01-004", 4),
	}
	require.Error(t, repo.CreateRunWithItems(ctx, run, items),
		"a duplicate membership must roll back the run and every prior item")

	var count int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_runs").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_run_items").Scan(&count))
	require.Zero(t, count)

	run = testRun()
	items = []*TeamRunItem{
		testItem("T-E38-F01-004", 4),
		testItem("T-E38-F01-001", 1),
		testItem("T-E38-F01-003", 3),
		testItem("T-E38-F01-002", 2),
	}
	require.NoError(t, repo.CreateRunWithItems(ctx, run, items))
	require.NotZero(t, run.ID)

	got, err := repo.ListItems(ctx, run.ID)
	require.NoError(t, err)
	require.Len(t, got, 4)
	require.Equal(t, "T-E38-F01-001", got[0].ChildKey)
	require.Equal(t, "T-E38-F01-002", got[1].ChildKey)
	require.Equal(t, "T-E38-F01-003", got[2].ChildKey)
	require.Equal(t, "T-E38-F01-004", got[3].ChildKey)
	for _, item := range got {
		require.Equal(t, run.ID, item.TeamRunID)
	}
}

func TestRepository_ConstraintsAndNullableFields_TC011(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()

	run := testRun()
	completed := time.Date(2026, 7, 13, 10, 30, 0, 0, time.UTC)
	run.StartedAt = &completed
	require.NoError(t, repo.CreateRunWithItems(ctx, run, []*TeamRunItem{testItem("T-E38-F01-001", 1)}))

	got, err := repo.GetRun(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, run.RootKey, got.RootKey)
	require.NotNil(t, got.StartedAt)
	require.Nil(t, got.CompletedAt)

	_, err = database.ExecContext(ctx, `
		INSERT INTO team_run_items
			(team_run_id, child_key, child_type, wave, execution_order, dependency_keys)
		VALUES (?, 'T-E38-F01-001', 'task', 0, 2, '[]')`, run.ID)
	require.Error(t, err, "membership must be unique per run, child type, and key")
}

func TestRepository_CreateRunWithItems_RetriesTransientBusyWithoutDuplicates_TC012(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()

	lock, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lock.ExecContext(ctx, `
		INSERT INTO team_runs
			(root_key, root_type, status, execution_mode, concurrency_limit, plan_hash)
		VALUES ('E38-LOCK', 'feature', 'planned', 'sequential', 1,
			'fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210')`)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		run := testRun()
		run.RootKey = "E38-BUSY"
		done <- repo.CreateRunWithItems(ctx, run, []*TeamRunItem{testItem("T-E38-F01-001", 1)})
	}()

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, lock.Rollback())
	require.NoError(t, <-done)

	var count int
	require.NoError(t, database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM team_runs WHERE root_key = 'E38-BUSY'").Scan(&count))
	require.Equal(t, 1, count)
}
