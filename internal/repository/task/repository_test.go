package task

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/epic"
	"github.com/jwwelbor/shark-task-manager/internal/repository/feature"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepository_GetByIDClassifiesMissingEntity(t *testing.T) {
	repo := NewTaskRepository(dbconn.NewDB(test.GetTestDB()))

	_, err := repo.GetByID(context.Background(), -1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoerr.ErrNotFound), "error = %v", err)
}

// TestTaskRepository_Create_GeneratesAndStoresSlug verifies slug generation during task creation
func TestTaskRepository_Create_GeneratesAndStoresSlug(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E95-F01-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E95-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E95'")

	// Create dedicated epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E95",
		Title: "Test Epic for Slug Generation"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create dedicated feature for this test
	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E95-F01",
		Title: "Test Feature for Slug Generation"}, EpicID: testEpic.ID,

		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create task
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E95-F01-001",
		Title: "Implement User Authentication System"}, FeatureID: testFeature.ID,

		Status:   models.TaskStatus("todo"),
		Priority: 5,
	}

	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Verify slug was generated and stored
	assert.NotNil(t, task.Slug, "Slug should be generated")
	assert.Equal(t, "implement-user-authentication-system", *task.Slug)

	// Verify slug is persisted in database
	retrieved, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Slug, "Slug should be persisted")
	assert.Equal(t, "implement-user-authentication-system", *retrieved.Slug)
}

// TestTaskRepository_Create_SlugHandlesSpecialCharacters verifies slug handles special characters
func TestTaskRepository_Create_SlugHandlesSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key IN ('T-E97-F01-001', 'T-E97-F01-002', 'T-E97-F01-003')")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create a dedicated test epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E97",
		Title: "Test Epic for Task Slug Special Characters"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	// Create a dedicated test feature
	featureRepo := feature.NewFeatureRepository(db)
	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E97-F01",
		Title: "Test Feature for Task Slugs"}, EpicID: testEpic.ID,

		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() {
		if _, err := database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}()

	testCases := []struct {
		title        string
		expectedSlug string
	}{
		{
			title:        "Fix Bug: Memory Leak in Worker Pool",
			expectedSlug: "fix-bug-memory-leak-in-worker-pool",
		},
		{
			title:        "Upgrade PostgreSQL -> MongoDB",
			expectedSlug: "upgrade-postgresql-mongodb",
		},
		{
			title:        "Add Support for UTF-8 & Unicode 测试",
			expectedSlug: "add-support-for-utf-8-unicode",
		},
	}

	for i, tc := range testCases {
		task := &models.Task{BaseEntity: models.BaseEntity{Key: fmt.Sprintf("T-E97-F01-%03d", i+1),
			Title: tc.title}, FeatureID: testFeature.ID,

			Status:   models.TaskStatus("todo"),
			Priority: 5,
		}

		err := repo.Create(ctx, task)
		require.NoError(t, err, "Failed to create task with key %s, title: %s", task.Key, tc.title)

		assert.NotNil(t, task.Slug, "Slug should be generated for: %s", tc.title)
		assert.Equal(t, tc.expectedSlug, *task.Slug, "Slug mismatch for: %s", tc.title)

		// Cleanup
		defer func(id int64) {
			if _, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id); err != nil {
				t.Logf("Cleanup error: %v", err)
			}
		}(task.ID)
	}
}

// TestTaskRepository_UpdateCascadesOrder verifies that updating a task's execution order
// automatically resequences all other tasks in the same feature
func TestTaskRepository_UpdateCascadesOrder(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E98-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E98",
		Title: "Test Epic for Order Cascade"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create test feature
	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E98-F01",
		Title: "Test Feature for Order Cascade"}, EpicID: testEpic.ID,

		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	// Create four tasks with sequential orders: a-1, b-2, c-3, d-4
	order1, order2, order3, order4 := 1, 2, 3, 4
	taskA := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E98-F01-001",
		Title: "Task A"}, FeatureID: testFeature.ID,

		Status:         models.TaskStatus("todo"),
		Priority:       5,
		ExecutionOrder: &order1,
	}
	taskB := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E98-F01-002",
		Title: "Task B"}, FeatureID: testFeature.ID,

		Status:         models.TaskStatus("todo"),
		Priority:       5,
		ExecutionOrder: &order2,
	}
	taskC := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E98-F01-003",
		Title: "Task C"}, FeatureID: testFeature.ID,

		Status:         models.TaskStatus("todo"),
		Priority:       5,
		ExecutionOrder: &order3,
	}
	taskD := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E98-F01-004",
		Title: "Task D"}, FeatureID: testFeature.ID,

		Status:         models.TaskStatus("todo"),
		Priority:       5,
		ExecutionOrder: &order4,
	}

	err = taskRepo.Create(ctx, taskA)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskA.ID) }()

	err = taskRepo.Create(ctx, taskB)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskB.ID) }()

	err = taskRepo.Create(ctx, taskC)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskC.ID) }()

	err = taskRepo.Create(ctx, taskD)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskD.ID) }()

	// When: Update task D's order from 4 to 2
	newOrder := 2
	taskD.ExecutionOrder = &newOrder
	err = taskRepo.Update(ctx, taskD)
	require.NoError(t, err, "Failed to update task D's order")

	// Then: Verify cascade - expected order: a-1, d-2, b-3, c-4
	// Get all tasks for this feature
	tasks, err := taskRepo.ListByFeature(ctx, testFeature.ID)
	require.NoError(t, err, "Failed to list tasks by feature ID")
	require.Len(t, tasks, 4, "Should have 4 tasks")

	// Build a map for easy verification
	taskOrders := make(map[string]int)
	for _, task := range tasks {
		if task.ExecutionOrder != nil {
			taskOrders[task.Title] = *task.ExecutionOrder
		}
	}

	// Verify expected orders
	assert.Equal(t, 1, taskOrders["Task A"], "Task A should be at order 1")
	assert.Equal(t, 2, taskOrders["Task D"], "Task D should be at order 2 (moved)")
	assert.Equal(t, 3, taskOrders["Task B"], "Task B should be at order 3 (shifted)")
	assert.Equal(t, 4, taskOrders["Task C"], "Task C should be at order 4 (shifted)")
}

// TestTaskRepository_UpdateNoResequence_PreservesDuplicateOrders verifies that
// UpdateNoResequence sets the target task's execution_order without renumbering
// siblings, allowing intentional duplicate-order groups (parallel work) to be
// formed via `shark task update --parallel`.
func TestTaskRepository_UpdateNoResequence_PreservesDuplicateOrders(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E97-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E97-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E97'")

	// Create test epic
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E97",
		Title: "Test Epic for No-Resequence"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	require.NoError(t, epicRepo.Create(ctx, testEpic), "Failed to create test epic")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create test feature
	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E97-F01",
		Title: "Test Feature for No-Resequence"}, EpicID: testEpic.ID,
		Status: models.FeatureStatusDraft,
	}
	require.NoError(t, featureRepo.Create(ctx, testFeature), "Failed to create test feature")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	// Seed tasks at orders 1, 2, 3
	order1, order2, order3 := 1, 2, 3
	taskA := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-001", Title: "Task A"},
		FeatureID: testFeature.ID, Status: models.TaskStatus("todo"), Priority: 5,
		ExecutionOrder: &order1,
	}
	taskB := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-002", Title: "Task B"},
		FeatureID: testFeature.ID, Status: models.TaskStatus("todo"), Priority: 5,
		ExecutionOrder: &order2,
	}
	taskC := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E97-F01-003", Title: "Task C"},
		FeatureID: testFeature.ID, Status: models.TaskStatus("todo"), Priority: 5,
		ExecutionOrder: &order3,
	}
	require.NoError(t, taskRepo.Create(ctx, taskA))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskA.ID) }()
	require.NoError(t, taskRepo.Create(ctx, taskB))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskB.ID) }()
	require.NoError(t, taskRepo.Create(ctx, taskC))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskC.ID) }()

	// When: move Task C to order=1 WITHOUT cascade
	newOrder := 1
	taskC.ExecutionOrder = &newOrder
	require.NoError(t, taskRepo.UpdateNoResequence(ctx, taskC), "UpdateNoResequence should succeed")

	// Then: A and C share order=1 (parallel batch); B remains at 2
	tasks, err := taskRepo.ListByFeature(ctx, testFeature.ID)
	require.NoError(t, err, "Failed to list tasks by feature ID")
	require.Len(t, tasks, 3, "Should have 3 tasks")

	taskOrders := make(map[string]int)
	for _, task := range tasks {
		require.NotNil(t, task.ExecutionOrder, "Task %s should have an execution order", task.Title)
		taskOrders[task.Title] = *task.ExecutionOrder
	}

	assert.Equal(t, 1, taskOrders["Task A"], "Task A should remain at order 1")
	assert.Equal(t, 2, taskOrders["Task B"], "Task B should remain at order 2 (no cascade)")
	assert.Equal(t, 1, taskOrders["Task C"], "Task C should be at order 1 alongside Task A")
}

// TestTaskRepository_UpdateNoResequence_ValidatesDependencies verifies that
// the --parallel update path (UpdateNoResequence / forceSkipCascade=true)
// still runs ValidateTaskDependencies.
//
// Before B048, this path could never change DependsOn (only --parallel's
// execution_order renumber used it), so validation was skipped as a TD-008
// optimization. B048 wired `--depends-on` through `shark task update`,
// including when combined with `--parallel`, so this path can now set
// tasks.depends_on too — and an unvalidated write would let a nonexistent,
// circular, or self dependency reach the column that gates status
// advancement (see docs/plan/bugs/B048.md). Both Update() and
// UpdateNoResequence() must now reject the same invalid depends_on.
func TestTaskRepository_UpdateNoResequence_ValidatesDependencies(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E93-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E93-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E93'")

	// Create test epic + feature
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E93",
		Title: "Test Epic TD-008"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E93-F01",
		Title: "Test Feature TD-008"}, EpicID: testEpic.ID,
		Status: models.FeatureStatusDraft,
	}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	order1 := 1
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E93-F01-001", Title: "Task X"},
		FeatureID: testFeature.ID, Status: models.TaskStatus("todo"), Priority: 5,
		ExecutionOrder: &order1,
	}
	require.NoError(t, taskRepo.Create(ctx, task))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID) }()
	prerequisite := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E93-F01-002", Title: "Prerequisite"},
		FeatureID: testFeature.ID, Status: models.TaskStatus("todo"), Priority: 5,
		ExecutionOrder: &order1,
	}
	require.NoError(t, taskRepo.Create(ctx, prerequisite))
	t.Cleanup(func() {
		_, err := database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", prerequisite.ID)
		require.NoError(t, err)
	})

	// Inject a depends_on pointing to a well-formed but non-existent task key.
	bogusDeps := `["T-E93-F01-999"]`
	task.DependsOn = &bogusDeps

	// Sanity check: regular Update() rejects.
	err := taskRepo.Update(ctx, task)
	require.Error(t, err, "regular Update should reject depends_on with non-existent key")
	assert.Contains(t, err.Error(), "dependency", "error should mention dependency validation")

	// B048: the fast path must reject the same invalid depends_on too, even
	// though it is only renumbering execution_order without cascade.
	newOrder := 2
	task.ExecutionOrder = &newOrder
	err = taskRepo.UpdateNoResequence(ctx, task)
	require.Error(t, err, "UpdateNoResequence must validate depends_on (B048); it must not silently persist a nonexistent dependency")
	assert.Contains(t, err.Error(), "dependency", "error should mention dependency validation")

	// Verify the row was NOT updated (rejected before the write).
	got, err := taskRepo.GetByKey(ctx, "T-E93-F01-001")
	require.NoError(t, err)
	require.NotNil(t, got.ExecutionOrder)
	assert.Equal(t, 1, *got.ExecutionOrder, "execution_order should be unchanged after a rejected update")

	// Clearing DependsOn on the same fast path must still succeed (nil/empty
	// deps are a no-op for ValidateTaskDependencies).
	task.DependsOn = nil
	require.NoError(t, taskRepo.UpdateNoResequence(ctx, task),
		"UpdateNoResequence must still succeed when depends_on is nil")

	got, err = taskRepo.GetByKey(ctx, "T-E93-F01-001")
	require.NoError(t, err)
	require.NotNil(t, got.ExecutionOrder)
	assert.Equal(t, 2, *got.ExecutionOrder, "execution_order should update once depends_on is valid (nil)")

	// A valid dependency must be persisted through the same --parallel path.
	validDeps := `["T-E93-F01-002"]`
	task.DependsOn = &validDeps
	newOrder = 3
	task.ExecutionOrder = &newOrder
	require.NoError(t, taskRepo.UpdateNoResequence(ctx, task))

	got, err = taskRepo.GetByKey(ctx, "T-E93-F01-001")
	require.NoError(t, err)
	require.NotNil(t, got.DependsOn)
	assert.Equal(t, validDeps, *got.DependsOn, "--parallel update must persist a valid depends_on value")
	require.NotNil(t, got.ExecutionOrder)
	assert.Equal(t, 3, *got.ExecutionOrder)
}

// TestTaskRepository_UpdateNoResequence_FastPath verifies the TD-008 fast path:
// when forceSkipCascade short-circuits, the update lands the new row state and
// leaves no connection in use afterwards.
//
// NOTE: this test does NOT prove a transaction was never opened — db.Stats()
// returns InUse==0 after completion for both the tx and non-tx paths, so it
// cannot distinguish them. Counting BeginTx invocations would require a driver
// wrapper, which is out of scope here; we deliberately scope this test to the
// observable post-conditions (row updated, pool drained) rather than overclaim.
func TestTaskRepository_UpdateNoResequence_FastPath(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E92-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E92-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E92'")

	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E92",
		Title: "Test Epic TD-008 NoTx"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E92-F01",
		Title: "Test Feature TD-008 NoTx"}, EpicID: testEpic.ID,
		Status: models.FeatureStatusDraft,
	}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	order1 := 1
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E92-F01-001", Title: "Task NoTx"},
		FeatureID: testFeature.ID, Status: models.TaskStatus("todo"), Priority: 5,
		ExecutionOrder: &order1,
	}
	require.NoError(t, taskRepo.Create(ctx, task))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID) }()

	statsBefore := database.Stats()

	// Exercise the fast path.
	newOrder := 7
	task.ExecutionOrder = &newOrder
	require.NoError(t, taskRepo.UpdateNoResequence(ctx, task))

	statsAfter := database.Stats()

	// Post-condition: the connection pool is drained (no connection left in use).
	// This holds for both the tx and non-tx paths, so it is a leak check, not a
	// proof that no transaction was opened — see the function doc.
	assert.Equal(t, 0, statsAfter.InUse, "no connection should be in-use after UpdateNoResequence returns")
	// Sanity: the call must have actually done some database work.
	assert.GreaterOrEqual(t, statsAfter.OpenConnections, statsBefore.OpenConnections,
		"OpenConnections should not have decreased")

	// Verify the row was actually updated.
	got, err := taskRepo.GetByKey(ctx, "T-E92-F01-001")
	require.NoError(t, err)
	require.NotNil(t, got.ExecutionOrder)
	assert.Equal(t, 7, *got.ExecutionOrder)
}

func TestTaskRepository_UpdateNoResequence_IgnoresPersistedLegacyDependencies(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	taskRepo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	highPriority := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E91", Title: "TD-062 epic"}, Status: models.EpicStatusActive, Priority: models.PriorityHigh, BusinessValue: &highPriority}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()
	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E91-F01", Title: "TD-062 feature"}, EpicID: testEpic.ID, Status: models.FeatureStatusDraft}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	order := 1
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E91-F01-001", Title: "legacy dependency"}, FeatureID: testFeature.ID, Status: "todo", Priority: 5, ExecutionOrder: &order}
	require.NoError(t, taskRepo.Create(ctx, task))
	legacyDeps := `["missing-task"]`
	_, err := database.ExecContext(ctx, "UPDATE tasks SET depends_on = ? WHERE id = ?", legacyDeps, task.ID)
	require.NoError(t, err)

	newOrder := 2
	task.ExecutionOrder = &newOrder
	task.DependsOn = nil
	require.NoError(t, taskRepo.UpdateNoResequence(ctx, task))

	updated, err := taskRepo.GetByKey(ctx, task.Key)
	require.NoError(t, err)
	require.NotNil(t, updated.ExecutionOrder)
	assert.Equal(t, 2, *updated.ExecutionOrder)
}

// TestTaskRepository_UpdateStatus_BackwardTransitionRequiresReason tests rejection reason validation
func TestTaskRepository_UpdateStatus_BackwardTransitionRequiresReason(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Clean up test data first (use unique numbers to avoid conflicts)
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-F98%%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-F98%%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

	// Create test epic
	highValue := models.PriorityHigh
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E98",
		Title: "Test Rejection Reasons Epic"}, Status: models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highValue,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	// Create test feature
	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E98-F98",
		Title: "Test Rejection Reasons Feature"}, EpicID: testEpic.ID,

		Status: models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	// Create a test task in in_progress status (development phase)
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E98-F98-001",
		Title: "Test Rejection Reason Task"}, FeatureID: testFeature.ID,

		Status:   models.TaskStatus("in_progress"),
		Priority: 5,
	}
	err = repo.Create(ctx, task)
	require.NoError(t, err)
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID) }()

	// Get the initial task status
	initialTask, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(initialTask.Status))

	t.Run("backward transition without reason succeeds at repo layer", func(t *testing.T) {
		// NOTE: Backward transition validation (reason requirement) has moved to the service layer.
		// The repository layer no longer validates transitions or requires reasons.

		// Ensure task starts in in_progress
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(current.Status))

		// Update to ready_for_review
		reason := "Test review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err, "Forward transition should succeed")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("ready_for_review"), models.TaskStatus(updated.Status))

		// Backward transition without reason now succeeds at repo layer
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil)
		assert.NoError(t, err, "Backward transition without reason should succeed at repo layer (validation moved to service)")

		// Verify status changed
		updated, err = repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(updated.Status))
	})

	t.Run("backward transition with reason should succeed", func(t *testing.T) {
		// Reset task to ready_for_review first
		reason := "Initial review"
		err := repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err, "Forward transition should succeed")

		// Verify task is in correct status
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, models.TaskStatus("ready_for_review"), models.TaskStatus(current.Status))

		// Now try backward transition WITH reason (ready_for_review -> in_progress)
		rejectionReason := "Missing error handling"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &rejectionReason)
		assert.NoError(t, err, "Backward transition with reason should succeed")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(updated.Status))
	})

	t.Run("backward transition with force flag bypasses reason requirement", func(t *testing.T) {
		// Ensure task is in in_progress first
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Reset task to ready_for_review
		reason := "Review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err)

		// Try backward transition with force but no reason
		err = repo.UpdateStatusForced(ctx, task.ID, models.TaskStatus("in_progress"), nil, nil, nil, nil, true)
		assert.NoError(t, err, "Backward transition with force should bypass reason requirement")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("in_progress"), models.TaskStatus(updated.Status))
	})

	t.Run("forward transition without reason should succeed", func(t *testing.T) {
		// Ensure task is in in_progress first
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Forward transitions (planning -> development) should not require reason
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, nil)
		assert.NoError(t, err, "Forward transition should succeed without reason")

		// Verify status changed
		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, models.TaskStatus("ready_for_review"), models.TaskStatus(updated.Status))
	})

	t.Run("empty reason string succeeds at repo layer for backward transition", func(t *testing.T) {
		// NOTE: Backward transition reason validation has moved to the service layer.
		// The repository layer no longer validates reasons.
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Reset to ready_for_review
		reason := "Review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err)

		// Try with empty reason string - repo allows it now
		emptyReason := ""
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &emptyReason)
		assert.NoError(t, err, "Backward transition with empty reason should succeed at repo layer (validation moved to service)")
	})

	t.Run("whitespace-only reason succeeds at repo layer for backward transition", func(t *testing.T) {
		// NOTE: Backward transition reason validation has moved to the service layer.
		current, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		if current.Status != models.TaskStatus("in_progress") {
			resetReason := "Reset for test"
			_ = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &resetReason)
		}

		// Reset to ready_for_review
		reason := "Review"
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("ready_for_review"), nil, &reason)
		require.NoError(t, err)

		// Try with whitespace-only reason - repo allows it now
		whitespacedReason := "   \t\n  "
		err = repo.UpdateStatus(ctx, task.ID, models.TaskStatus("in_progress"), nil, &whitespacedReason)
		assert.NoError(t, err, "Backward transition with whitespace-only reason should succeed at repo layer (validation moved to service)")
	})
}

// TestTaskRepository_UpdateStatusForced_StoresRejectionReason verifies that rejection reasons are stored in task_history
func TestTaskRepository_UpdateStatusForced_StoresRejectionReason(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	historyRepo := NewTaskHistoryRepository(db)

	// Seed test data
	_, featureID := test.SeedTestData()

	// Clean up any leftover tasks from previous runs before creating
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-F99-%'")

	// Create task in ready_for_review status with unique key
	timestamp := time.Now().UnixNano() % 1000
	taskKey := fmt.Sprintf("T-E99-F99-%03d", timestamp)
	priority := 5
	task := &models.Task{BaseEntity: models.BaseEntity{Key: taskKey,
		Title: "Test Rejection Reason Storage"}, Status: models.TaskStatus("ready_for_review"),
		FeatureID: featureID,
		Priority:  priority,
	}

	err := repo.Create(ctx, task)
	require.NoError(t, err, "Failed to create test task")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}()

	// Update status with rejection reason (backward transition)
	rejectionReason := "Missing error handling on line 67"
	agent := "test-agent"
	notes := "This needs to be fixed"

	// Call UpdateStatusForced with rejection reason
	err = repo.UpdateStatusForced(
		ctx,
		task.ID,
		models.TaskStatus("in_progress"),
		&agent,
		&notes,
		&rejectionReason, // Now passing rejection reason
		nil,              // documentPath
		false,
	)
	require.NoError(t, err, "UpdateStatusForced should succeed")

	// Verify task status was updated
	updatedTask, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err, "Failed to get updated task")
	require.Equal(t, models.TaskStatus("in_progress"), updatedTask.Status, "Task status should be updated")

	// Verify rejection reason was stored in history
	history, err := historyRepo.ListByTask(ctx, task.ID)
	require.NoError(t, err, "Failed to retrieve task history")
	require.NotEmpty(t, history, "History should have at least one entry")

	// Get the most recent history entry (most recent first in list)
	lastEntry := history[len(history)-1]
	require.NotNil(t, lastEntry.OldStatus, "Old status should be present")
	require.Equal(t, string(models.TaskStatus("ready_for_review")), *lastEntry.OldStatus, "Old status should match")
	require.Equal(t, string(models.TaskStatus("in_progress")), lastEntry.NewStatus, "New status should match")

	// THIS IS THE CRITICAL ASSERTION - rejection reason should be stored
	// This will FAIL until we implement Step 1-2
	require.NotNil(t, lastEntry.RejectionReason, "Rejection reason should be stored in history")
	require.Equal(t, rejectionReason, *lastEntry.RejectionReason, "Rejection reason should match what was provided")
	require.NotNil(t, lastEntry.Agent, "Agent should be stored")
	require.Equal(t, agent, *lastEntry.Agent, "Agent should be stored")
	require.NotNil(t, lastEntry.Notes, "Notes should be stored")
	require.Equal(t, notes, *lastEntry.Notes, "Notes should be stored")
}

// TestTaskRepository_GetByIDs tests the batch GetByIDs method
func TestTaskRepository_GetByIDs(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Clean up test data first
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E95-F17-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E95-F17'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E95'")

	// Create dedicated epic for this test
	highPriority := models.PriorityHigh
	testEpic := &models.Epic{
		BaseEntity:    models.BaseEntity{Key: "E95", Title: "Test Epic for GetByIDs"},
		Status:        models.EpicStatusActive,
		Priority:      models.PriorityHigh,
		BusinessValue: &highPriority,
	}
	err := epicRepo.Create(ctx, testEpic)
	require.NoError(t, err, "Failed to create test epic")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	// Create dedicated feature for this test
	testFeature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E95-F17", Title: "Test Feature for GetByIDs"},
		EpicID:     testEpic.ID,
		Status:     models.FeatureStatusDraft,
	}
	err = featureRepo.Create(ctx, testFeature)
	require.NoError(t, err, "Failed to create test feature")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID)
	}()

	// Create 3 test tasks
	task1 := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E95-F17-011", Title: "Batch Test Task One"},
		FeatureID:  testFeature.ID,
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	task2 := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E95-F17-012", Title: "Batch Test Task Two"},
		FeatureID:  testFeature.ID,
		Status:     models.TaskStatus("in_progress"),
		Priority:   3,
	}
	task3 := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E95-F17-013", Title: "Batch Test Task Three"},
		FeatureID:  testFeature.ID,
		Status:     models.TaskStatus("todo"),
		Priority:   7,
	}

	for _, task := range []*models.Task{task1, task2, task3} {
		err = repo.Create(ctx, task)
		require.NoError(t, err, "Failed to create test task %s", task.Key)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E95-F17-%'")
	}()

	t.Run("MultipleTasks", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{task1.ID, task2.ID, task3.ID})
		require.NoError(t, err)
		assert.Len(t, tasks, 3)

		// Build a map for easy lookup
		tasksByKey := make(map[string]*models.Task)
		for _, task := range tasks {
			tasksByKey[task.Key] = task
		}
		assert.Contains(t, tasksByKey, "T-E95-F17-011")
		assert.Contains(t, tasksByKey, "T-E95-F17-012")
		assert.Contains(t, tasksByKey, "T-E95-F17-013")
	})

	t.Run("EmptyInput", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{})
		require.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.Len(t, tasks, 0)
	})

	t.Run("NilInput", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.Len(t, tasks, 0)
	})

	t.Run("SomeMissing", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{task1.ID, -999, task3.ID})
		require.NoError(t, err)
		assert.Len(t, tasks, 2)

		tasksByKey := make(map[string]*models.Task)
		for _, task := range tasks {
			tasksByKey[task.Key] = task
		}
		assert.Contains(t, tasksByKey, "T-E95-F17-011")
		assert.Contains(t, tasksByKey, "T-E95-F17-013")
	})

	t.Run("AllMissing", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{-1, -2, -3})
		require.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.Len(t, tasks, 0)
	})

	t.Run("SingleID", func(t *testing.T) {
		tasks, err := repo.GetByIDs(ctx, []int64{task2.ID})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "T-E95-F17-012", tasks[0].Key)
		assert.Equal(t, "Batch Test Task Two", tasks[0].Title)
		assert.Equal(t, models.TaskStatus("in_progress"), tasks[0].Status)
	})
}

// ptrIntTask returns a pointer to n; helper for size round-trip tests.
func ptrIntTask(n int) *int { return &n }

// TestTaskRepository_SizeRoundTrip verifies that Size persists through Create,
// GetByKey (via GetByID), and Update without information loss (TC-F010-C).
func TestTaskRepository_SizeRoundTrip(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	epicKey := "E96"
	featureKey := "E96-F01"
	taskKey := "T-E96-F01-001"

	// Clean up before test
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", taskKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = ?", featureKey)
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = ?", epicKey)

	// Create parent epic
	testEpic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: epicKey, Title: "Task Size RT Epic"},
		Status:     models.EpicStatusDraft,
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, epicRepo.Create(ctx, testEpic), "Failed to create parent epic")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID)
	}()

	// Create parent feature
	testFeature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: featureKey, Title: "Task Size RT Feature"},
		EpicID:     testEpic.ID,
		Status:     models.FeatureStatusDraft,
	}
	require.NoError(t, featureRepo.Create(ctx, testFeature), "Failed to create parent feature")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID)
	}()

	// Step 1: Create task with Size = ptr(5)
	task := &models.Task{
		BaseEntity: models.BaseEntity{
			Key:   taskKey,
			Title: "Size Round Trip Task",
			Size:  ptrIntTask(5),
		},
		FeatureID: testFeature.ID,
		Status:    models.TaskStatus("todo"),
		Priority:  5,
	}
	require.NoError(t, repo.Create(ctx, task), "Create() failed")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
	}()

	// Read back and assert Size == 5
	got, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err, "GetByID() failed")
	if got.Size == nil {
		t.Fatal("expected Size to be non-nil after Create")
	}
	assert.Equal(t, 5, *got.Size, "expected Size=5 after Create")

	// Step 2: Update Size = ptr(1)
	got.Size = ptrIntTask(1)
	require.NoError(t, repo.Update(ctx, got), "Update() failed")

	got2, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err, "GetByID() after update failed")
	if got2.Size == nil {
		t.Fatal("expected Size to be non-nil after Update to 1")
	}
	assert.Equal(t, 1, *got2.Size, "expected Size=1 after update")

	// Step 3: Update Size = nil
	got2.Size = nil
	require.NoError(t, repo.Update(ctx, got2), "Update() to nil failed")

	got3, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err, "GetByID() after nil update failed")
	assert.Nil(t, got3.Size, "expected Size=nil after clearing")
}

// --- GetRecent tests (T-E07-F17-002) ---

// seedTasksWithTimestamps creates n tasks under the given feature with created_at
// staggered by 1 second each (oldest first in the slice, so tasks[0] is the oldest).
// Uses direct SQL INSERT to bypass key-format validation and enable staggered timestamps.
// Keys follow the valid format T-E90-F01-NNN (e.g. T-E90-F01-001 … T-E90-F01-010).
// Returns task IDs for deferred cleanup.
func seedTasksWithTimestamps(t *testing.T, _ *TaskRepository, db *dbconn.DB, _ int64, featureID int64, n int) []int64 {
	t.Helper()
	ctx := context.Background()
	baseTime := time.Now().UTC().Add(-time.Duration(n) * time.Second)
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("T-E90-F01-%03d", i+1)
		ts := baseTime.Add(time.Duration(i) * time.Second)
		result, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO tasks (feature_id, key, title, status, priority, created_at, updated_at)
			 VALUES (?, ?, ?, 'todo', 5, ?, CURRENT_TIMESTAMP)`,
			featureID, key, fmt.Sprintf("Recent Task %d", i+1), ts.Format("2006-01-02T15:04:05Z"),
		)
		require.NoError(t, err, "seedTasksWithTimestamps: INSERT failed for key %s", key)
		id, err := result.LastInsertId()
		require.NoError(t, err)
		ids = append(ids, id)
	}
	return ids
}

// TestTaskRepository_GetRecent_OrdersByCreatedAtDesc seeds 5 tasks with distinct timestamps
// and asserts that GetRecent returns them in created_at DESC order.
func TestTaskRepository_GetRecent_OrdersByCreatedAtDesc(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E90-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E90-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E90", Title: "Recent Test Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E90-F01", Title: "Recent Test Feature"}, EpicID: testEpic.ID, Status: models.FeatureStatusActive}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	ids := seedTasksWithTimestamps(t, repo, db, testEpic.ID, testFeature.ID, 5)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
		}
	}()

	tasks, err := repo.GetRecent(ctx, 5)
	require.NoError(t, err)
	require.Len(t, tasks, 5)

	// Assert descending order by created_at
	for i := 1; i < len(tasks); i++ {
		assert.True(t, !tasks[i-1].CreatedAt.Before(tasks[i].CreatedAt),
			"expected tasks[%d].CreatedAt >= tasks[%d].CreatedAt, got %v < %v",
			i-1, i, tasks[i-1].CreatedAt, tasks[i].CreatedAt)
	}
}

// TestTaskRepository_GetRecent_LimitRespected seeds 10 tasks and asserts that
// GetRecent(ctx, 3) returns exactly 3 rows.
func TestTaskRepository_GetRecent_LimitRespected(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E90-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E90-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E90", Title: "Recent Test Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E90-F01", Title: "Recent Test Feature"}, EpicID: testEpic.ID, Status: models.FeatureStatusActive}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	ids := seedTasksWithTimestamps(t, repo, db, testEpic.ID, testFeature.ID, 10)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
		}
	}()

	tasks, err := repo.GetRecent(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, tasks, 3, "GetRecent(3) must return exactly 3 rows")
}

// TestTaskRepository_GetRecent_EmptyTable asserts that GetRecent returns a non-nil
// empty slice when no tasks exist.
func TestTaskRepository_GetRecent_EmptyTable(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)

	// Ensure no RECENT-tagged rows exist
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E90-F01-%'")

	// We cannot guarantee the whole tasks table is empty (other tests may run concurrently),
	// but we can verify the method works correctly by deleting our test prefix and
	// calling with limit 0 is not valid per spec; instead just verify no panic and
	// that when we know there are no rows the slice is non-nil.
	// Use a dedicated cleanup to make this as isolated as possible.
	tasks, err := repo.GetRecent(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, tasks, "GetRecent must return a non-nil slice even when empty")
}

// TestTaskRepository_GetRecent_LimitExceedsRowCount seeds 2 tasks and asserts that
// GetRecent(ctx, 100) returns exactly 2 (all rows, not an error).
func TestTaskRepository_GetRecent_LimitExceedsRowCount(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Pre-cleanup
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E90-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E90-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E90", Title: "Recent Test Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E90-F01", Title: "Recent Test Feature"}, EpicID: testEpic.ID, Status: models.FeatureStatusActive}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	ids := seedTasksWithTimestamps(t, repo, db, testEpic.ID, testFeature.ID, 2)
	defer func() {
		for _, id := range ids {
			_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
		}
	}()

	tasks, err := repo.GetRecent(ctx, 100)
	require.NoError(t, err)
	// At least our 2 seeded rows; DB may have other rows so we only check >= 2
	assert.GreaterOrEqual(t, len(tasks), 2, "GetRecent(100) must return at least the 2 seeded tasks")
}

// BenchmarkTaskRepository_GetRecent measures GetRecent(ctx, 100) against a pre-seeded DB.
// Seeds 10000 rows once, then benchmarks. Expectation: < 150ms per operation (REQ-NF-001).
func BenchmarkTaskRepository_GetRecent(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)
	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)

	// Cleanup before seeding
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E90-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E90-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E90'")

	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E90", Title: "Bench Recent Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	if err := epicRepo.Create(ctx, testEpic); err != nil {
		b.Fatalf("create epic: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E90-F01", Title: "Bench Recent Feature"}, EpicID: testEpic.ID, Status: models.FeatureStatusActive}
	if err := featureRepo.Create(ctx, testFeature); err != nil {
		b.Fatalf("create feature: %v", err)
	}
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	// Seed 10000 tasks in batches using direct SQL for speed.
	// Keys are arbitrary strings for the benchmark — we bypass model validation
	// by using INSERT directly and ensure cleanup via the feature_id foreign key.
	batchSize := 500
	totalRows := 10000
	for batch := 0; batch < totalRows/batchSize; batch++ {
		for i := 0; i < batchSize; i++ {
			rowNum := batch*batchSize + i + 1
			key := fmt.Sprintf("bench-recent-%05d", rowNum)
			_, err := database.ExecContext(ctx,
				"INSERT OR IGNORE INTO tasks (feature_id, key, title, status, priority, created_at, updated_at) VALUES (?, ?, ?, 'todo', 5, datetime('now', ?), CURRENT_TIMESTAMP)",
				testFeature.ID, key, fmt.Sprintf("Bench Task %d", rowNum), fmt.Sprintf("-%d seconds", totalRows-rowNum))
			if err != nil {
				b.Fatalf("seed task %d: %v", rowNum, err)
			}
		}
	}
	// Cleanup via feature_id cascade (epic+feature deferred deletes above cover tasks)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetRecent(ctx, 100)
	}
}

// TestTaskRepository_StatusUpdateRawWithTx_Guarded_RejectsStaleStatus proves
// the Guarded compare-and-swap actually rejects a write whose OldStatus no
// longer matches the row's current status, catching a caller that raced past
// a status change it didn't observe (e.g. advance_guard's replay/staleness
// protection).
func TestTaskRepository_StatusUpdateRawWithTx_Guarded_RejectsStaleStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewTaskRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = 'T-E96-F01-001'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'E96-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E96'")

	epicRepo := epic.NewEpicRepository(db)
	featureRepo := feature.NewFeatureRepository(db)
	testEpic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E96", Title: "Guarded Update Epic"}, Status: models.EpicStatusActive, Priority: models.PriorityMedium}
	require.NoError(t, epicRepo.Create(ctx, testEpic))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", testEpic.ID) }()

	testFeature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E96-F01", Title: "Guarded Update Feature"}, EpicID: testEpic.ID, Status: models.FeatureStatusActive}
	require.NoError(t, featureRepo.Create(ctx, testFeature))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", testFeature.ID) }()

	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E96-F01-001", Title: "Guarded Update Task"}, FeatureID: testFeature.ID, Status: models.TaskStatus("todo"), Priority: 5}
	require.NoError(t, repo.Create(ctx, task))
	defer func() { _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID) }()

	// First guarded update: todo -> in_progress, OldStatus correctly "todo".
	// This must succeed and actually change the row.
	tx1, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	_, err = repo.StatusUpdateRawWithTx(ctx, tx1, models.StatusUpdateParams{
		TaskID: task.ID, NewStatus: models.TaskStatus("in_progress"),
		OldStatus: "todo", TaskKey: task.Key, Guarded: true,
	})
	require.NoError(t, err, "first guarded update from the true current status must succeed")
	require.NoError(t, tx1.Commit())

	updated, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("in_progress"), updated.Status, "status must actually have changed after the guarded update")

	// Second guarded update: simulates a caller that raced past the first
	// update — it still believes the status is "todo" (stale) and tries the
	// same guarded transition again. This is exactly the replay/race
	// scenario advance_guard exists to prevent, and must be rejected without
	// mutating the row a second time.
	tx2, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	_, err = repo.StatusUpdateRawWithTx(ctx, tx2, models.StatusUpdateParams{
		TaskID: task.ID, NewStatus: models.TaskStatus("in_progress"),
		OldStatus: "todo", TaskKey: task.Key, Guarded: true,
	})
	assert.ErrorIs(t, err, ErrGuardedUpdateStale, "a guarded update against a stale OldStatus must be rejected")
	_ = tx2.Rollback()

	// UpdateStatusIfCurrent (the repository-level convenience wrapper) must
	// exhibit the same behavior: false, nil on a stale expected status.
	ok, err := repo.UpdateStatusIfCurrent(ctx, task.ID, "todo", "blocked")
	require.NoError(t, err)
	assert.False(t, ok, "UpdateStatusIfCurrent must report false for a stale expected status, not silently succeed")

	final, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatus("in_progress"), final.Status, "the rejected guarded updates must not have mutated the row")
}
