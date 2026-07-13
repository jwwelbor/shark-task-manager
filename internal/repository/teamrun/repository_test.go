package teamrun

import (
	"context"
	"database/sql"
	"errors"
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
	_, err := database.ExecContext(ctx, "PRAGMA busy_timeout = 0")
	require.NoError(t, err)

	lock, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = lock.ExecContext(ctx, `
		INSERT INTO team_runs
			(root_key, root_type, status, execution_mode, concurrency_limit, plan_hash)
		VALUES ('E38-LOCK', 'feature', 'planned', 'sequential', 1,
			'fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210')`)
	require.NoError(t, err)

	transactionStarted := make(chan struct{})
	release := make(chan struct{})
	repo.transactionBeginHook = func(attempt int) {
		if attempt == 0 {
			close(transactionStarted)
			<-release
		}
	}
	done := make(chan error, 1)
	go func() {
		run := testRun()
		run.RootKey = "E38-BUSY"
		done <- repo.CreateRunWithItems(ctx, run, []*TeamRunItem{testItem("T-E38-F01-001", 1)})
	}()

	<-transactionStarted
	require.NoError(t, lock.Rollback())
	close(release)
	require.NoError(t, <-done)

	var count int
	require.NoError(t, database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM team_runs WHERE root_key = 'E38-BUSY'").Scan(&count))
	require.Equal(t, 1, count)
}

func TestRepository_CreateRunWithItemsIfAbsent_IsAtomicUnderConcurrentConfirmation_TC006(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()
	start := make(chan struct{})
	results := make(chan struct {
		run        *TeamRun
		idempotent bool
		err        error
	}, 2)

	for i := 0; i < 2; i++ {
		go func() {
			<-start
			run, idempotent, err := repo.CreateRunWithItemsIfAbsent(ctx, testRun(), []*TeamRunItem{testItem("T-E38-F01-001", 1)})
			results <- struct {
				run        *TeamRun
				idempotent bool
				err        error
			}{run, idempotent, err}
		}()
	}
	close(start)

	var runs []*TeamRun
	var idempotentCount int
	for i := 0; i < 2; i++ {
		result := <-results
		require.NoError(t, result.err)
		runs = append(runs, result.run)
		if result.idempotent {
			idempotentCount++
		}
	}
	require.Equal(t, 1, idempotentCount)
	require.Equal(t, runs[0].ID, runs[1].ID)

	var count int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_runs").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_run_items").Scan(&count))
	require.Equal(t, 1, count)
}

func TestRepository_CreateRunWithItemsIfAbsent_ReturnsExistingRootForPlanDrift(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()

	first, idempotent, err := repo.CreateRunWithItemsIfAbsent(ctx, testRun(), []*TeamRunItem{testItem("T-E38-F01-001", 1)})
	require.NoError(t, err)
	require.False(t, idempotent)

	drifted := testRun()
	drifted.PlanHash = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	second, idempotent, err := repo.CreateRunWithItemsIfAbsent(ctx, drifted, []*TeamRunItem{testItem("T-E38-F01-001", 1)})
	require.NoError(t, err, "root-level uniqueness must return the existing snapshot")
	require.True(t, idempotent)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.PlanHash, second.PlanHash)

	var runCount, itemCount int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_runs WHERE root_type = 'feature' AND root_key = 'E38-F01'").Scan(&runCount))
	require.Equal(t, 1, runCount)
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_run_items WHERE team_run_id = ?", first.ID).Scan(&itemCount))
	require.Equal(t, 1, itemCount)
}

func TestRepository_CreateRunWithItemsIfAbsent_DoesNotExtendLegacyDuplicateRoot(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()
	hashA := testRun().PlanHash
	hashB := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	_, err := database.ExecContext(ctx, "DROP INDEX IF EXISTS idx_team_runs_confirmation")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "CREATE UNIQUE INDEX idx_team_runs_confirmation ON team_runs(root_type, root_key, plan_hash)")
	require.NoError(t, err)
	for _, hash := range []string{hashA, hashB} {
		_, err = database.ExecContext(ctx, `
			INSERT INTO team_runs
				(root_key, root_type, status, execution_mode, concurrency_limit, plan_hash)
			VALUES ('E38-LEGACY', 'feature', 'planned', 'sequential', 1, ?)`, hash)
		require.NoError(t, err)
	}

	requested := testRun()
	requested.RootKey = "E38-LEGACY"
	requested.PlanHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	confirmed, idempotent, err := repo.CreateRunWithItemsIfAbsent(ctx, requested, []*TeamRunItem{testItem("T-E38-F01-001", 1)})
	require.NoError(t, err)
	require.True(t, idempotent)
	require.NotEqual(t, requested.PlanHash, confirmed.PlanHash)

	var count int
	require.NoError(t, database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM team_runs WHERE root_type = 'feature' AND root_key = 'E38-LEGACY'").Scan(&count))
	require.Equal(t, 2, count)
}

func TestRepository_CompareAndSetItem_RejectsStaleWriter_TC007(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()
	run := testRun()
	item := testItem("T-E38-F01-001", 1)
	require.NoError(t, repo.CreateRunWithItems(ctx, run, []*TeamRunItem{item}))

	item.ItemStatus = "completed"
	item.Attempt = 0
	updated, err := repo.CompareAndSetItem(ctx, item, "planned", 0)
	require.NoError(t, err)
	require.True(t, updated)

	item.Outcome = stringPtr("different")
	updated, err = repo.CompareAndSetItem(ctx, item, "planned", 0)
	require.NoError(t, err)
	require.False(t, updated, "a stale writer must not overwrite the terminal result")

	got, err := repo.ListItems(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", got[0].ItemStatus)
	require.Nil(t, got[0].Outcome)
}

func TestRepository_CompareAndSetItem_PreservesConfirmedSnapshot(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))
	ctx := context.Background()
	run := testRun()
	item := testItem("T-E38-F01-001", 1)
	item.PlannedAction = stringPtr("develop")
	item.PlannedModel = stringPtr("model-a")
	require.NoError(t, repo.CreateRunWithItems(ctx, run, []*TeamRunItem{item}))

	item.Wave = 99
	item.ExecutionOrder = 99
	item.DependencyKeys = `["T-E38-F01-999"]`
	item.PlannedAction = stringPtr("rewrite")
	item.PlannedModel = stringPtr("model-b")
	item.ItemStatus = "claimed"
	updated, err := repo.CompareAndSetItem(ctx, item, "planned", 0)
	require.NoError(t, err)
	require.True(t, updated)

	got, err := repo.ListItems(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, 0, got[0].Wave)
	require.Equal(t, 1, got[0].ExecutionOrder)
	require.Equal(t, "[]", got[0].DependencyKeys)
	require.Equal(t, "develop", *got[0].PlannedAction)
	require.Equal(t, "model-a", *got[0].PlannedModel)
	require.Equal(t, "claimed", got[0].ItemStatus)
}

func TestRepository_TransactionRetry_CoversBeginCallbackRollbackAndCancel(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))

	t.Run("busy begin retries", func(t *testing.T) {
		attempts := 0
		repo.transactionBeginErrorHook = func(attempt int) error {
			attempts++
			if attempt == 0 {
				return errors.New("database is locked")
			}
			return nil
		}
		callbackCalls := 0
		require.NoError(t, repo.withTransactionRetry(context.Background(), func(*sql.Tx) error {
			callbackCalls++
			return nil
		}))
		require.Equal(t, 2, attempts)
		require.Equal(t, 1, callbackCalls)
		repo.transactionBeginErrorHook = nil
	})

	t.Run("begin callback runs after transaction starts", func(t *testing.T) {
		callbackAttempt := -1
		repo.transactionBeginHook = func(attempt int) { callbackAttempt = attempt }
		require.NoError(t, repo.withTransactionRetry(context.Background(), func(*sql.Tx) error { return nil }))
		require.Equal(t, 0, callbackAttempt)
		repo.transactionBeginHook = nil
	})

	t.Run("rollback error is joined with callback error", func(t *testing.T) {
		callbackErr := errors.New("callback failed")
		rollbackErr := errors.New("rollback failed")
		repo.transactionRollbackHook = func(error) error { return rollbackErr }
		err := repo.withTransactionRetry(context.Background(), func(*sql.Tx) error { return callbackErr })
		require.ErrorIs(t, err, callbackErr)
		require.ErrorIs(t, err, rollbackErr)
		repo.transactionRollbackHook = nil
	})

	t.Run("cancelled context does not retry", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		attempts := 0
		repo.transactionBeginErrorHook = func(int) error { attempts++; return nil }
		err := repo.withTransactionRetry(ctx, func(*sql.Tx) error { return nil })
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, attempts)
		repo.transactionBeginErrorHook = nil
	})
}

func TestRepository_TransactionRetry_CommitAndBusyExhaustion(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	repo := NewRepository(dbconn.NewDB(database))

	t.Run("commit error is not retried", func(t *testing.T) {
		commitErr := errors.New("commit failed")
		calls := 0
		repo.transactionCommitHook = func(error) error { return commitErr }
		err := repo.withTransactionRetry(context.Background(), func(*sql.Tx) error { calls++; return nil })
		require.ErrorIs(t, err, commitErr)
		require.Equal(t, 1, calls)
		repo.transactionCommitHook = nil
	})

	t.Run("busy callback exhausts retries", func(t *testing.T) {
		calls := 0
		err := repo.withTransactionRetry(context.Background(), func(*sql.Tx) error {
			calls++
			return errors.New("database is locked")
		})
		require.Equal(t, transactionAttempts, calls)
		require.ErrorContains(t, err, "database is locked")
	})
}

func stringPtr(value string) *string { return &value }
