package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	sharkdb "github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const progressCoherencyWorkflow = `{
	"advance_guard": {"enabled": true, "mode": "session_from_status"},
	"task_workflow": {
		"special_statuses": {"_start_": ["todo"], "_complete_": ["completed"]},
		"status_flow": {"todo": ["completed"], "completed": ["todo"]},
		"status_metadata": {
			"todo": {"phase": "planning", "progress_weight": 0.0},
			"completed": {"phase": "done", "progress_weight": 1.0}
		}
	},
	"feature_workflow": {
		"special_statuses": {"_start_": ["draft"], "_complete_": ["completed", "archived"], "_aggregation_": ["active"]},
		"status_flow": {"draft": ["active"], "review": ["active"], "active": ["completed"], "completed": ["active", "archived"], "archived": []},
		"status_metadata": {
			"draft": {"phase": "planning"},
			"review": {"phase": "planning"},
			"active": {"phase": "development", "orchestrator_action": {"action": "cascade", "instruction_template": "advance from child state"}},
			"completed": {"phase": "done"},
			"archived": {"phase": "done"}
		}
	},
	"epic_workflow": {
		"special_statuses": {"_start_": ["draft"], "_complete_": ["completed", "archived"], "_aggregation_": ["active"]},
		"status_flow": {"draft": ["active"], "review": ["active"], "active": ["completed"], "completed": ["active", "archived"], "archived": []},
		"status_metadata": {
			"draft": {"phase": "planning"},
			"review": {"phase": "planning"},
			"active": {"phase": "development", "orchestrator_action": {"action": "cascade", "instruction_template": "advance from child state"}},
			"completed": {"phase": "done"},
			"archived": {"phase": "done"}
		}
	}
}`

var errInjectedAggregateFailure = errors.New("injected aggregate failure")

type aggregateFailurePoint string

const (
	failFeatureRead aggregateFailurePoint = "feature_read"
	failEpicRead    aggregateFailurePoint = "epic_read"
)

type failingAggregateRepository struct {
	services.AggregateMutationRepository
	failAt aggregateFailurePoint
}

func (r *failingAggregateRepository) GetFeatureForProgressTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Feature, error) {
	if r.failAt == failFeatureRead {
		return nil, errInjectedAggregateFailure
	}
	return r.AggregateMutationRepository.GetFeatureForProgressTx(ctx, tx, id)
}

func (r *failingAggregateRepository) GetEpicForStatusTx(ctx context.Context, tx *sql.Tx, id int64) (*models.Epic, error) {
	if r.failAt == failEpicRead {
		return nil, errInjectedAggregateFailure
	}
	return r.AggregateMutationRepository.GetEpicForStatusTx(ctx, tx, id)
}

type progressCoherencyFixture struct {
	db          *repository.DB
	services    *ServiceContainer
	projectRoot string
	epicID      int64
	featureID   int64
}

func newProgressCoherencyFixture(t *testing.T, epicStatus, featureStatus string, progress float64) *progressCoherencyFixture {
	t.Helper()
	config.ClearWorkflowCache()
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, ".sharkconfig.json"), []byte(progressCoherencyWorkflow), 0o644))
	sqlDB, err := sharkdb.InitDB(filepath.Join(projectRoot, "progress-coherency.db"))
	require.NoError(t, err)
	database := repository.NewDB(sqlDB)
	t.Cleanup(func() {
		config.ClearWorkflowCache()
		require.NoError(t, database.Close())
	})

	fixture := &progressCoherencyFixture{
		db:          database,
		services:    WireServices(database, projectRoot),
		projectRoot: projectRoot,
	}
	if epicStatus != "" {
		fixture.epicID = insertProgressEpic(t, database, epicStatus)
	}
	if featureStatus != "" {
		fixture.featureID = insertProgressFeature(t, database, fixture.epicID, featureStatus, progress)
	}
	return fixture
}

func (f *progressCoherencyFixture) injectAggregateFailure(point aggregateFailurePoint) {
	repo := &failingAggregateRepository{
		AggregateMutationRepository: repository.NewProgressMutationRepository(),
		failAt:                      point,
	}
	coordinator := services.NewAggregateMutationCoordinator(repo, workflow.NewService(f.projectRoot))
	f.services.TaskService.SetAggregateMutationCoordinator(coordinator)
	f.services.FeatureService.SetAggregateMutationCoordinator(coordinator)
	f.services.EpicService.SetAggregateMutationCoordinator(coordinator)
}

func insertProgressEpic(t *testing.T, database *repository.DB, status string) int64 {
	t.Helper()
	result, err := database.Exec(`
		INSERT INTO epics (key, title, status, priority, file_path)
		VALUES ('E01', 'Progress epic', ?, 'high', 'docs/plan/E01/epic.md')`, status)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

func insertProgressFeature(t *testing.T, database *repository.DB, epicID int64, status string, progress float64) int64 {
	t.Helper()
	result, err := database.Exec(`
		INSERT INTO features (epic_id, key, title, status, progress_pct, file_path)
		VALUES (?, 'E01-F01', 'Progress feature', ?, ?, 'docs/plan/E01/E01-F01/feature.md')`, epicID, status, progress)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

func insertProgressTask(t *testing.T, database *repository.DB, featureID int64, status string) int64 {
	t.Helper()
	result, err := database.Exec(`
		INSERT INTO tasks (feature_id, key, title, status, priority, agent_type)
		VALUES (?, 'T-E01-F01-001', 'Progress task', ?, 5, 'general')`, featureID, status)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

func rowCount(t *testing.T, database *repository.DB, table, predicate string, args ...interface{}) int {
	t.Helper()
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, predicate)
	var count int
	require.NoError(t, database.QueryRow(query, args...).Scan(&count))
	return count
}

func entityStatus(t *testing.T, database *repository.DB, table string, id int64) string {
	t.Helper()
	query := fmt.Sprintf("SELECT status FROM %s WHERE id = ?", table)
	var status string
	require.NoError(t, database.QueryRow(query, id).Scan(&status))
	return status
}

func featureProgress(t *testing.T, database *repository.DB, id int64) float64 {
	t.Helper()
	var progress float64
	require.NoError(t, database.QueryRow("SELECT progress_pct FROM features WHERE id = ?", id).Scan(&progress))
	return progress
}

func TestProgressCoherency_TaskCreateSuccessAndRollback(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "completed", "completed", 100)

		task, _, err := fixture.services.TaskService.CreateTask(context.Background(), services.CreateTaskInput{
			EpicKey: "E01", FeatureKey: "E01-F01", Title: "Created task",
		})

		require.NoError(t, err)
		require.NotNil(t, task)
		assert.Equal(t, 1, rowCount(t, fixture.db, "tasks", "feature_id = ?", fixture.featureID))
		assert.Equal(t, "active", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 0.0, featureProgress(t, fixture.db, fixture.featureID))
		assert.Equal(t, "active", entityStatus(t, fixture.db, "epics", fixture.epicID))
	})

	t.Run("aggregate failure rolls back task and creator history", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "completed", "completed", 100)
		fixture.injectAggregateFailure(failFeatureRead)

		_, _, err := fixture.services.TaskService.CreateTask(context.Background(), services.CreateTaskInput{
			EpicKey: "E01", FeatureKey: "E01-F01", Title: "Rolled back task",
		})

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, 0, rowCount(t, fixture.db, "tasks", "feature_id = ?", fixture.featureID))
		assert.Equal(t, 0, rowCount(t, fixture.db, "task_history", "1 = 1"))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 100.0, featureProgress(t, fixture.db, fixture.featureID))
	})
}

func TestProgressCoherency_TaskCreationReopensAncestorsFromHistory(t *testing.T) {
	fixture := newProgressCoherencyFixture(t, "completed", "completed", 100)
	_, err := fixture.db.Exec(`
		INSERT INTO entity_history (entity_type, entity_id, from_status, to_status)
		VALUES ('feature', ?, 'draft', 'review'), ('epic', ?, 'draft', 'review')`,
		fixture.featureID, fixture.epicID)
	require.NoError(t, err)

	_, _, err = fixture.services.TaskService.CreateTask(context.Background(), services.CreateTaskInput{
		EpicKey: "E01", FeatureKey: "E01-F01", Title: "Regression task",
	})

	require.NoError(t, err)
	assert.Equal(t, "review", entityStatus(t, fixture.db, "features", fixture.featureID))
	assert.Equal(t, "review", entityStatus(t, fixture.db, "epics", fixture.epicID))
}

func TestProgressCoherency_TaskDeleteSuccessAndRollback(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 50)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "completed")

		require.NoError(t, fixture.services.TaskService.DeleteTask(context.Background(), "T-E01-F01-001"))

		assert.Equal(t, 0, rowCount(t, fixture.db, "tasks", "id = ?", taskID))
		assert.Equal(t, 0.0, featureProgress(t, fixture.db, fixture.featureID))
	})

	t.Run("aggregate failure rolls back delete", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 50)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "completed")
		fixture.injectAggregateFailure(failFeatureRead)

		err := fixture.services.TaskService.DeleteTask(context.Background(), "T-E01-F01-001")

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, 1, rowCount(t, fixture.db, "tasks", "id = ?", taskID))
		assert.Equal(t, 50.0, featureProgress(t, fixture.db, fixture.featureID))
	})
}

func TestProgressCoherency_TaskTransitionSuccessAndRollback(t *testing.T) {
	t.Run("success commits task, aggregates, and histories together", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")

		_, err := fixture.services.TaskService.TransitionStatus(context.Background(), "T-E01-F01-001", "completed", services.TransitionOptions{})

		require.NoError(t, err)
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 100.0, featureProgress(t, fixture.db, fixture.featureID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
		assert.Equal(t, 3, rowCount(t, fixture.db, "entity_history", "1 = 1"))
	})

	t.Run("aggregate failure rolls back generic transition side effects", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")
		fixture.injectAggregateFailure(failFeatureRead)

		_, err := fixture.services.TaskService.TransitionStatus(context.Background(), "T-E01-F01-001", "completed", services.TransitionOptions{})

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, "todo", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, "active", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 0, rowCount(t, fixture.db, "entity_history", "1 = 1"))
		assert.Equal(t, 0, rowCount(t, fixture.db, "task_history", "1 = 1"))
	})
}

func TestProgressCoherency_TaskTransitionAuxiliaryWritesShareTransaction(t *testing.T) {
	t.Run("forced reason note commits with successful mutation", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		insertProgressTask(t, fixture.db, fixture.featureID, "todo")

		_, err := fixture.services.TaskService.TransitionStatus(context.Background(), "T-E01-F01-001", "completed", services.TransitionOptions{
			Force: true, Reason: "verified rejection note transaction", Agent: "test-agent",
		})

		require.NoError(t, err)
		assert.Equal(t, 1, rowCount(t, fixture.db, "entity_notes", "entity_type = 'task' AND note_type = 'rejection'"))
	})

	t.Run("aggregate failure rolls back rejection note", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")
		fixture.injectAggregateFailure(failFeatureRead)

		_, err := fixture.services.TaskService.TransitionStatus(context.Background(), "T-E01-F01-001", "completed", services.TransitionOptions{
			Force: true, Reason: "must roll back", Agent: "test-agent",
		})

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, "todo", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, 0, rowCount(t, fixture.db, "entity_notes", "entity_type = 'task' AND note_type = 'rejection'"))
	})

	guardOpts := services.TransitionOptions{
		GuardAdvance: true,
		SessionID:    "session-1",
		FromStatus:   "todo",
		Outcome:      "pass",
	}
	t.Run("advance guard token commits with successful mutation", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		insertProgressTask(t, fixture.db, fixture.featureID, "todo")

		_, err := fixture.services.TaskService.TransitionStatus(context.Background(), "T-E01-F01-001", "completed", guardOpts)

		require.NoError(t, err)
		assert.Equal(t, 1, rowCount(t, fixture.db, "advance_guard_consumptions", "session_id = 'session-1'"))
	})

	t.Run("aggregate failure rolls back advance guard token", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")
		fixture.injectAggregateFailure(failFeatureRead)

		_, err := fixture.services.TaskService.TransitionStatus(context.Background(), "T-E01-F01-001", "completed", guardOpts)

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, "todo", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, 0, rowCount(t, fixture.db, "advance_guard_consumptions", "session_id = 'session-1'"))
	})
}

func TestProgressCoherency_FeatureCreateDeleteSuccessAndRollback(t *testing.T) {
	t.Run("create success reopens terminal epic", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "completed", "", 0)

		feature, err := fixture.services.FeatureService.CreateFeature(context.Background(), services.CreateFeatureInput{
			EpicKey: "E01", Title: "Created feature",
		})

		require.NoError(t, err)
		require.NotNil(t, feature)
		assert.Equal(t, "active", entityStatus(t, fixture.db, "epics", fixture.epicID))
		assert.Equal(t, 1, rowCount(t, fixture.db, "entity_history", "entity_type = 'epic'"))
	})

	t.Run("creating an already completed feature preserves terminal epic", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "completed", "", 0)

		feature, err := fixture.services.FeatureService.CreateFeature(context.Background(), services.CreateFeatureInput{
			EpicKey: "E01", Title: "Imported completed feature", Status: "completed",
		})

		require.NoError(t, err)
		require.NotNil(t, feature)
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
		assert.Equal(t, 0, rowCount(t, fixture.db, "entity_history", "entity_type = 'epic'"))
	})

	t.Run("create aggregate failure rolls back feature", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "completed", "", 0)
		fixture.injectAggregateFailure(failEpicRead)

		_, err := fixture.services.FeatureService.CreateFeature(context.Background(), services.CreateFeatureInput{
			EpicKey: "E01", Title: "Rolled back feature",
		})

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, 0, rowCount(t, fixture.db, "features", "epic_id = ?", fixture.epicID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
	})

	t.Run("delete success", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)

		require.NoError(t, fixture.services.FeatureService.DeleteFeature(context.Background(), "E01-F01"))

		assert.Equal(t, 0, rowCount(t, fixture.db, "features", "id = ?", fixture.featureID))
	})

	t.Run("delete aggregate failure rolls back feature", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		fixture.injectAggregateFailure(failEpicRead)

		err := fixture.services.FeatureService.DeleteFeature(context.Background(), "E01-F01")

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, 1, rowCount(t, fixture.db, "features", "id = ?", fixture.featureID))
	})
}

func TestProgressCoherency_FeatureStatusMutationsSuccessAndRollback(t *testing.T) {
	mutations := []struct {
		name   string
		mutate func(context.Context, *ServiceContainer) error
	}{
		{
			name: "transition",
			mutate: func(ctx context.Context, container *ServiceContainer) error {
				_, err := container.FeatureService.TransitionStatus(ctx, "E01-F01", "completed", services.TransitionOptions{})
				return err
			},
		},
		{
			name: "explicit update",
			mutate: func(ctx context.Context, container *ServiceContainer) error {
				status := models.FeatureStatusCompleted
				_, err := container.FeatureService.UpdateFeature(ctx, "E01-F01", services.FeatureUpdates{Status: &status})
				return err
			},
		},
		{
			name: "conditional update",
			mutate: func(ctx context.Context, container *ServiceContainer) error {
				updated, err := container.FeatureService.UpdateFeatureStatusIfNotOverridden(ctx, "E01-F01", models.FeatureStatusCompleted)
				if err == nil && !updated {
					return errors.New("conditional feature update was unexpectedly skipped")
				}
				return err
			},
		},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name+" success", func(t *testing.T) {
			fixture := newProgressCoherencyFixture(t, "active", "active", 0)

			require.NoError(t, mutation.mutate(context.Background(), fixture.services))

			assert.Equal(t, "completed", entityStatus(t, fixture.db, "features", fixture.featureID))
			assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
		})

		t.Run(mutation.name+" aggregate failure rolls back status", func(t *testing.T) {
			fixture := newProgressCoherencyFixture(t, "active", "active", 0)
			fixture.injectAggregateFailure(failEpicRead)

			err := mutation.mutate(context.Background(), fixture.services)

			require.ErrorIs(t, err, errInjectedAggregateFailure)
			assert.Equal(t, "active", entityStatus(t, fixture.db, "features", fixture.featureID))
			assert.Equal(t, "active", entityStatus(t, fixture.db, "epics", fixture.epicID))
			assert.Equal(t, 0, rowCount(t, fixture.db, "entity_history", "1 = 1"))
		})
	}
}

func TestProgressCoherency_FeatureCompleteSuccessAndRollback(t *testing.T) {
	t.Run("no-task success", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)

		_, err := fixture.services.FeatureService.CompleteFeature(context.Background(), "E01-F01", false)

		require.NoError(t, err)
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
	})

	t.Run("no-task aggregate failure rolls back feature completion", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		fixture.injectAggregateFailure(failEpicRead)

		_, err := fixture.services.FeatureService.CompleteFeature(context.Background(), "E01-F01", false)

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, "active", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, "active", entityStatus(t, fixture.db, "epics", fixture.epicID))
	})

	t.Run("success", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")

		result, err := fixture.services.FeatureService.CompleteFeature(context.Background(), "E01-F01", true)

		require.NoError(t, err)
		require.Len(t, result.AffectedTasks, 1)
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 100.0, featureProgress(t, fixture.db, fixture.featureID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
	})

	t.Run("aggregate failure rolls back task and feature completion", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")
		fixture.injectAggregateFailure(failFeatureRead)

		_, err := fixture.services.FeatureService.CompleteFeature(context.Background(), "E01-F01", true)

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, "todo", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, "active", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 0.0, featureProgress(t, fixture.db, fixture.featureID))
	})
}

func TestProgressCoherency_FeatureCascadeAndRecalculateSuccessAndRollback(t *testing.T) {
	mutations := []struct {
		name       string
		taskStatus string
		mutate     func(context.Context, *ServiceContainer, int64) error
	}{
		{
			name:       "task cascade",
			taskStatus: "todo",
			mutate: func(ctx context.Context, container *ServiceContainer, _ int64) error {
				return container.FeatureService.CascadeFeatureStatusToTasks(ctx, "E01-F01", models.TaskStatus("completed"))
			},
		},
		{
			name:       "progress recalculation",
			taskStatus: "completed",
			mutate: func(ctx context.Context, container *ServiceContainer, featureID int64) error {
				return container.FeatureService.RecalculateAndSetProgress(ctx, featureID)
			},
		},
		{
			name:       "progress recalculation by key",
			taskStatus: "completed",
			mutate: func(ctx context.Context, container *ServiceContainer, _ int64) error {
				return container.FeatureService.RecalculateAndSetProgressByKey(ctx, "E01-F01")
			},
		},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name+" success", func(t *testing.T) {
			fixture := newProgressCoherencyFixture(t, "active", "active", 0)
			taskID := insertProgressTask(t, fixture.db, fixture.featureID, mutation.taskStatus)

			require.NoError(t, mutation.mutate(context.Background(), fixture.services, fixture.featureID))

			assert.Equal(t, "completed", entityStatus(t, fixture.db, "tasks", taskID))
			assert.Equal(t, "completed", entityStatus(t, fixture.db, "features", fixture.featureID))
			assert.Equal(t, 100.0, featureProgress(t, fixture.db, fixture.featureID))
			assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
		})

		t.Run(mutation.name+" aggregate failure rolls back", func(t *testing.T) {
			fixture := newProgressCoherencyFixture(t, "active", "active", 0)
			taskID := insertProgressTask(t, fixture.db, fixture.featureID, mutation.taskStatus)
			fixture.injectAggregateFailure(failFeatureRead)

			err := mutation.mutate(context.Background(), fixture.services, fixture.featureID)

			require.ErrorIs(t, err, errInjectedAggregateFailure)
			assert.Equal(t, mutation.taskStatus, entityStatus(t, fixture.db, "tasks", taskID))
			assert.Equal(t, "active", entityStatus(t, fixture.db, "features", fixture.featureID))
			assert.Equal(t, 0.0, featureProgress(t, fixture.db, fixture.featureID))
		})
	}
}

func TestProgressCoherency_EpicForcedCompletionSuccessAndRollback(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")

		result, err := fixture.services.EpicService.CompleteEpic(context.Background(), "E01", true, "test-agent")

		require.NoError(t, err)
		assert.True(t, result.ForceCompleted)
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 100.0, featureProgress(t, fixture.db, fixture.featureID))
		assert.Equal(t, "completed", entityStatus(t, fixture.db, "epics", fixture.epicID))
	})

	t.Run("aggregate failure rolls back entire hierarchy", func(t *testing.T) {
		fixture := newProgressCoherencyFixture(t, "active", "active", 0)
		taskID := insertProgressTask(t, fixture.db, fixture.featureID, "todo")
		fixture.injectAggregateFailure(failFeatureRead)

		_, err := fixture.services.EpicService.CompleteEpic(context.Background(), "E01", true, "test-agent")

		require.ErrorIs(t, err, errInjectedAggregateFailure)
		assert.Equal(t, "todo", entityStatus(t, fixture.db, "tasks", taskID))
		assert.Equal(t, "active", entityStatus(t, fixture.db, "features", fixture.featureID))
		assert.Equal(t, 0.0, featureProgress(t, fixture.db, fixture.featureID))
		assert.Equal(t, "active", entityStatus(t, fixture.db, "epics", fixture.epicID))
	})
}
