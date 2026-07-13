package db_test

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/require"
)

func TestMigrateTeamRunTables_IsAdditiveAndIdempotent_TC011(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	ctx := context.Background()

	_, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS team_run_items")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DROP TABLE IF EXISTS team_runs")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM schema_version")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (27)")
	require.NoError(t, err)

	_, err = database.ExecContext(ctx, `
		INSERT INTO epics (key, title, status, priority)
		VALUES ('E38-MIG', 'Migration fixture', 'active', 'high')`)
	require.NoError(t, err)

	_, err = db.ApplySchemaIfNeeded(database)
	require.NoError(t, err)
	_, err = db.ApplySchemaIfNeeded(database)
	require.NoError(t, err)
	var schemaVersion int
	require.NoError(t, database.QueryRowContext(ctx,
		"SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&schemaVersion))
	require.Equal(t, db.CurrentSchemaVersion, schemaVersion)

	for _, table := range []string{"team_runs", "team_run_items"} {
		var count int
		require.NoError(t, database.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count))
		require.Equal(t, 1, count, "migration should create %s", table)
	}

	var epicCount int
	require.NoError(t, database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM epics WHERE key = 'E38-MIG'").Scan(&epicCount))
	require.Equal(t, 1, epicCount, "migration must not rewrite existing entity rows")

	for _, index := range []string{
		"idx_team_runs_root_status",
		"idx_team_runs_status",
		"idx_team_runs_plan_hash",
		"idx_team_runs_confirmation",
		"idx_team_run_items_run_wave_status",
		"idx_team_run_items_child",
		"idx_team_run_items_claim_session",
		"idx_team_run_items_worker_session",
	} {
		var count int
		require.NoError(t, database.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", index).Scan(&count))
		require.Equal(t, 1, count, "migration should create %s", index)
	}
}

func TestMigrateTeamRunTables_RepairsPartialMigrationAndEnforcesCascade_TC011(t *testing.T) {
	database := test.NewIsolatedTestDB(t)
	ctx := context.Background()

	_, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS team_run_items")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "DELETE FROM schema_version")
	require.NoError(t, err)
	_, err = database.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (27)")
	require.NoError(t, err)
	_, err = db.ApplySchemaIfNeeded(database)
	require.NoError(t, err)

	_, err = database.ExecContext(ctx, `
		INSERT INTO team_runs
			(root_key, root_type, status, execution_mode, concurrency_limit, plan_hash)
		VALUES ('E38-F01', 'feature', 'planned', 'sequential', 1,
			'0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef')`)
	require.NoError(t, err)

	var runID int64
	require.NoError(t, database.QueryRowContext(ctx,
		"SELECT id FROM team_runs WHERE root_key = 'E38-F01'").Scan(&runID))
	_, err = database.ExecContext(ctx, `
		INSERT INTO team_run_items
			(team_run_id, child_key, child_type, wave, execution_order, dependency_keys, item_status)
		VALUES (?, 'T-E38-F01-001', 'task', 0, 1, '[]', 'planned')`, runID)
	require.NoError(t, err)

	_, err = database.ExecContext(ctx, "DELETE FROM team_runs WHERE id = ?", runID)
	require.NoError(t, err)

	var itemCount int
	require.NoError(t, database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM team_run_items WHERE team_run_id = ?", runID).Scan(&itemCount))
	require.Zero(t, itemCount, "team_run_items must cascade when the run is deleted")

	_, err = database.ExecContext(ctx, `
		INSERT INTO team_runs
			(root_key, root_type, status, execution_mode, concurrency_limit, plan_hash)
		VALUES ('E38-F01', 'invalid', 'planned', 'sequential', 1,
			'0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef')`)
	require.Error(t, err)
}
