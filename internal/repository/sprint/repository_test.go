package sprint

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSprintRepository_Create(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up existing test data
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S901', 'S902', 'S903')")

	tests := []struct {
		name    string
		sprint  *models.Sprint
		wantErr bool
	}{
		{
			name: "valid sprint creation",
			sprint: &models.Sprint{
				Key:       "S901",
				Name:      "Sprint 1",
				Goal:      "Deliver features",
				StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
				Status:    "planning",
				Slug:      "sprint-1",
				FilePath:  "/path/to/sprint.md",
			},
			wantErr: false,
		},
		{
			name: "empty name fails validation",
			sprint: &models.Sprint{
				Key:       "S902",
				Name:      "   ",
				Goal:      "Deliver features",
				StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
				Status:    "planning",
			},
			wantErr: true,
		},
		{
			name: "invalid date range fails validation",
			sprint: &models.Sprint{
				Key:       "S903",
				Name:      "Sprint 3",
				Goal:      "Deliver features",
				StartDate: time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				Status:    "planning",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean before test
			_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = ?", tt.sprint.Key)

			err := repo.Create(ctx, tt.sprint)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.sprint.ID)

				// Verify created in database
				retrieved, err := repo.GetByKey(ctx, tt.sprint.Key)
				assert.NoError(t, err)
				assert.Equal(t, tt.sprint.Name, retrieved.Name)
				assert.Equal(t, tt.sprint.Key, retrieved.Key)
			}

			// Cleanup
			_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = ?", tt.sprint.Key)
		})
	}
}

func TestSprintRepository_GetByKey_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")

	// Create sprint
	sprint := &models.Sprint{
		Key:       "S901",
		Name:      "Sprint 1",
		Goal:      "Test sprint",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)

	tests := []struct {
		name   string
		key    string
		wantOK bool
	}{
		{"exact match", "S901", true},
		{"lowercase", "s901", true},
		{"uppercase", "S901", true},
		{"mixed case", "s901", true},
		{"not found", "S999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := repo.GetByKey(ctx, tt.key)
			if tt.wantOK {
				assert.NoError(t, err)
				assert.NotNil(t, retrieved)
				assert.Equal(t, sprint.Name, retrieved.Name)
			} else {
				assert.Error(t, err)
				assert.Nil(t, retrieved)
			}
		})
	}
}

func TestSprintRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")

	// Create sprint
	sprint := &models.Sprint{
		Key:       "S901",
		Name:      "Sprint 1",
		Goal:      "Test sprint",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)

	// Retrieve by ID
	retrieved, err := repo.GetByID(ctx, sprint.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, sprint.Key, retrieved.Key)
	assert.Equal(t, sprint.Name, retrieved.Name)

	// Not found
	_, err = repo.GetByID(ctx, 99999)
	assert.Error(t, err)
}

func TestSprintRepository_Update(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")

	// Create sprint
	sprint := &models.Sprint{
		Key:       "S901",
		Name:      "Sprint 1",
		Goal:      "Original goal",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)

	// Update
	sprint.Name = "Sprint 1 Updated"
	sprint.Goal = "Updated goal"
	sprint.Status = "active"
	err = repo.Update(ctx, sprint)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByKey(ctx, sprint.Key)
	assert.NoError(t, err)
	assert.Equal(t, "Sprint 1 Updated", retrieved.Name)
	assert.Equal(t, "Updated goal", retrieved.Goal)
	assert.Equal(t, models.SprintStatus("active"), retrieved.Status)
}

func TestSprintRepository_Delete(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")

	// Create sprint
	sprint := &models.Sprint{
		Key:       "S901",
		Name:      "Sprint 1",
		Goal:      "Test sprint",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(ctx, sprint.ID)
	assert.NoError(t, err)

	// Verify deleted
	_, err = repo.GetByID(ctx, sprint.ID)
	assert.Error(t, err)

	// Delete non-existent
	err = repo.Delete(ctx, 99999)
	assert.Error(t, err)
}

func TestSprintRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")

	// Create sprint
	sprint := &models.Sprint{
		Key:       "S901",
		Name:      "Sprint 1",
		Goal:      "Test sprint",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)

	// Update status atomically
	err = repo.UpdateStatus(ctx, sprint.ID, "active")
	assert.NoError(t, err)

	// Verify status changed
	retrieved, err := repo.GetByID(ctx, sprint.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.SprintStatus("active"), retrieved.Status)

	// Update status again
	err = repo.UpdateStatus(ctx, sprint.ID, "closed")
	assert.NoError(t, err)

	retrieved, err = repo.GetByID(ctx, sprint.ID)
	assert.NoError(t, err)
	assert.Equal(t, models.SprintStatus("closed"), retrieved.Status)

	// Update non-existent
	err = repo.UpdateStatus(ctx, 99999, "active")
	assert.Error(t, err)
}

func TestSprintRepository_GetNextKey_Monotonic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S500', 'S501', 'S502', 'S503', 'S504')")

	tests := []struct {
		name     string
		existing []string
		want     string
	}{
		{
			name:     "no existing sprints in range",
			existing: []string{},
			want:     "S001",
		},
		{
			name:     "one sprint returns next",
			existing: []string{"S500"},
			want:     "S501",
		},
		{
			name:     "multiple sprints returns next sequential",
			existing: []string{"S500", "S501", "S502"},
			want:     "S503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean before test - preserve other sprints, clean only our test range
			_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S500', 'S501', 'S502', 'S503', 'S504')")

			// Create existing sprints
			for _, key := range tt.existing {
				sprint := &models.Sprint{
					Key:       key,
					Name:      "Sprint",
					Goal:      "Test",
					StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
					Status:    "planning",
				}
				_ = repo.Create(ctx, sprint)
			}

			// Get next key
			nextKey, err := repo.GetNextKey(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, nextKey)
		})
	}
}

func TestSprintRepository_List(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S701', 'S702', 'S703')")

	// Create test sprints
	sprints := []*models.Sprint{
		{
			Key:       "S701",
			Name:      "Planning Sprint",
			Goal:      "Kick off",
			StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
			Status:    "planning",
		},
		{
			Key:       "S702",
			Name:      "Active Sprint",
			Goal:      "Development",
			StartDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2025, 1, 28, 0, 0, 0, 0, time.UTC),
			Status:    "active",
		},
		{
			Key:       "S703",
			Name:      "Closed Sprint",
			Goal:      "Completed",
			StartDate: time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 12, 28, 0, 0, 0, 0, time.UTC),
			Status:    "closed",
		},
	}

	for _, sprint := range sprints {
		_ = repo.Create(ctx, sprint)
	}
	defer database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S701', 'S702', 'S703')")

	tests := []struct {
		name       string
		filters    *SprintListFilters
		wantCount  int
		wantStatus *models.SprintStatus
	}{
		{
			name:       "list all test sprints",
			filters:    nil,
			wantCount:  3,
			wantStatus: nil,
		},
		{
			name: "filter by status planning",
			filters: &SprintListFilters{
				Status: func() *models.SprintStatus {
					s := models.SprintStatus("planning")
					return &s
				}(),
			},
			wantCount: 1,
		},
		{
			name: "filter by status active",
			filters: &SprintListFilters{
				Status: func() *models.SprintStatus {
					s := models.SprintStatus("active")
					return &s
				}(),
			},
			wantCount: 1,
		},
		{
			name: "filter by status closed",
			filters: &SprintListFilters{
				Status: func() *models.SprintStatus {
					s := models.SprintStatus("closed")
					return &s
				}(),
			},
			wantCount: 1,
		},
		{
			name: "filter by non-existent status",
			filters: &SprintListFilters{
				Status: func() *models.SprintStatus {
					s := models.SprintStatus("archived")
					return &s
				}(),
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.List(ctx, tt.filters)
			assert.NoError(t, err)
			// Count only the sprints we created
			testCount := 0
			for _, r := range result {
				if r.Key == "S701" || r.Key == "S702" || r.Key == "S703" {
					testCount++
				}
			}
			assert.Equal(t, tt.wantCount, testCount)

			if tt.wantStatus != nil && len(result) > 0 {
				for _, s := range result {
					if s.Key == "S701" || s.Key == "S702" || s.Key == "S703" {
						assert.Equal(t, *tt.wantStatus, s.Status)
					}
				}
			}
		})
	}
}

func TestSprintRepository_Create_Parameterized(t *testing.T) {
	// This test verifies parameterized queries prevent SQL injection.
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")

	// Try to create a sprint with malicious SQL in the name
	sprint := &models.Sprint{
		Key:       "S901",
		Name:      "Sprint'; DROP TABLE sprints; --",
		Goal:      "Test",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}

	// Should succeed and store the literal string (not execute SQL)
	err := repo.Create(ctx, sprint)
	assert.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)

	// Verify the table still exists and the string was stored literally
	retrieved, err := repo.GetByKey(ctx, sprint.Key)
	assert.NoError(t, err)
	assert.Equal(t, "Sprint'; DROP TABLE sprints; --", retrieved.Name)

	// Verify the sprints table still exists
	var count int
	_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM sprints").Scan(&count)
	assert.Greater(t, count, 0)
}

// ─────────────────────────────────────────────────────────────────────────────
// Assignment test helpers
// ─────────────────────────────────────────────────────────────────────────────

// seedTestSprint creates a sprint for assignment tests and returns its ID.
// Uses the S9xx key range reserved for assignment tests.
func seedTestSprint(t *testing.T, repo *SprintRepository, key string) int64 {
	t.Helper()
	ctx := context.Background()
	s := &models.Sprint{
		Key:       key,
		Name:      "Assignment Test Sprint " + key,
		Goal:      "Test goal",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, s)
	require.NoError(t, err, "failed to seed sprint %s", key)
	return s.ID
}

// seedTaskRow inserts a minimal tasks row for tests that need a real task ID.
// Returns the task's database ID.
func seedTaskRow(ctx context.Context, t *testing.T, db *dbconn.DB, key, title, status string) int64 {
	t.Helper()
	// Resolve a feature to satisfy FK constraints.
	// SeedTestData returns (epicID, featureID) but tasks only need feature_id.
	_, featureID := test.SeedTestData()
	result, err := db.ExecContext(ctx,
		`INSERT INTO tasks (key, title, status, feature_id, agent_type, priority, execution_order, depends_on)
		 VALUES (?, ?, ?, ?, 'backend', 5, 1, '[]')`,
		key, title, status, featureID,
	)
	require.NoError(t, err, "failed to seed task row %s", key)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// seedBugRow inserts a minimal bugs row for tests that need a real bug ID.
// Bugs table: id, key, title, status, severity (NOT NULL), ...
func seedBugRow(ctx context.Context, t *testing.T, db *dbconn.DB, key, title, status string) int64 {
	t.Helper()
	result, err := db.ExecContext(ctx,
		`INSERT INTO bugs (key, title, status, severity)
		 VALUES (?, ?, ?, 'medium')`,
		key, title, status,
	)
	require.NoError(t, err, "failed to seed bug row %s", key)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// seedChangeCardRow inserts a minimal change_cards row.
// change_cards table: id, key, title, status, priority (optional), ...
func seedChangeCardRow(ctx context.Context, t *testing.T, db *dbconn.DB, key, title, status string) int64 {
	t.Helper()
	result, err := db.ExecContext(ctx,
		`INSERT INTO change_cards (key, title, status)
		 VALUES (?, ?, ?)`,
		key, title, status,
	)
	require.NoError(t, err, "failed to seed change_card row %s", key)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// seedTechDebtRow inserts a minimal tech_debts row.
// tech_debts table: id, key, title, status, category (NOT NULL), severity (NOT NULL), ...
func seedTechDebtRow(ctx context.Context, t *testing.T, db *dbconn.DB, key, title, status string) int64 {
	t.Helper()
	result, err := db.ExecContext(ctx,
		`INSERT INTO tech_debts (key, title, status, category, severity)
		 VALUES (?, ?, ?, 'code-quality', 'medium')`,
		key, title, status,
	)
	require.NoError(t, err, "failed to seed tech_debt row %s", key)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// cleanupSprints removes test sprints (and cascades to sprint_assignments).
func cleanupSprints(ctx context.Context, t *testing.T, db *dbconn.DB, keys ...string) {
	t.Helper()
	for _, k := range keys {
		_, _ = db.ExecContext(ctx, `DELETE FROM sprints WHERE key = ?`, k)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R01: AddAssignment — task entity type succeeds
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_TaskEntityType_Succeeds(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up before test.
	cleanupSprints(ctx, t, db, "S910")
	_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'T-E01-F01-911'`)

	// Seed sprint in planning status.
	sprintID := seedTestSprint(t, repo, "S910")
	defer cleanupSprints(ctx, t, db, "S910")

	// Seed a real task row to satisfy the polymorphic reference.
	taskID := seedTaskRow(ctx, t, db, "T-E01-F01-911", "Assignment Test Task", "todo")
	defer db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID)

	// TC-R01: AddAssignment with entity_type="task" should succeed.
	assignment := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "task",
		EntityID:   taskID,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, assignment)
	assert.NoError(t, err, "TC-R01: AddAssignment should succeed for task entity type")
	assert.NotZero(t, assignment.ID, "TC-R01: AddAssignment should set the ID field")

	// Verify via GetActiveAssignment.
	got, err := repo.GetActiveAssignment(ctx, "task", taskID)
	require.NoError(t, err)
	require.NotNil(t, got, "TC-R01: GetActiveAssignment should return the assignment")
	assert.Equal(t, sprintID, got.SprintID)
	assert.Equal(t, "task", got.EntityType)
	assert.Equal(t, taskID, got.EntityID)
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R02: AddAssignment — bug entity type succeeds
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_BugEntityType_Succeeds(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S911")
	_, _ = db.ExecContext(ctx, `DELETE FROM bugs WHERE key = 'B911'`)

	sprintID := seedTestSprint(t, repo, "S911")
	defer cleanupSprints(ctx, t, db, "S911")

	bugID := seedBugRow(ctx, t, db, "B911", "Assignment Test Bug", "open")
	defer db.ExecContext(ctx, `DELETE FROM bugs WHERE id = ?`, bugID)

	assignment := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "bug",
		EntityID:   bugID,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, assignment)
	assert.NoError(t, err, "TC-R02: AddAssignment should succeed for bug entity type")

	got, err := repo.GetActiveAssignment(ctx, "bug", bugID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "bug", got.EntityType)
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R03: AddAssignment — change_card entity type succeeds
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_ChangeCardEntityType_Succeeds(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S912")
	_, _ = db.ExecContext(ctx, `DELETE FROM change_cards WHERE key = 'CC-912'`)

	sprintID := seedTestSprint(t, repo, "S912")
	defer cleanupSprints(ctx, t, db, "S912")

	ccID := seedChangeCardRow(ctx, t, db, "CC-912", "Assignment Test CC", "open")
	defer db.ExecContext(ctx, `DELETE FROM change_cards WHERE id = ?`, ccID)

	assignment := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "change_card",
		EntityID:   ccID,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, assignment)
	assert.NoError(t, err, "TC-R03: AddAssignment should succeed for change_card entity type")

	got, err := repo.GetActiveAssignment(ctx, "change_card", ccID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "change_card", got.EntityType)
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R04: AddAssignment — tech_debt entity type succeeds
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_TechDebtEntityType_Succeeds(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S913")
	_, _ = db.ExecContext(ctx, `DELETE FROM tech_debts WHERE key = 'TD-913'`)

	sprintID := seedTestSprint(t, repo, "S913")
	defer cleanupSprints(ctx, t, db, "S913")

	tdID := seedTechDebtRow(ctx, t, db, "TD-913", "Assignment Test TD", "open")
	defer db.ExecContext(ctx, `DELETE FROM tech_debts WHERE id = ?`, tdID)

	assignment := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "tech_debt",
		EntityID:   tdID,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, assignment)
	assert.NoError(t, err, "TC-R04: AddAssignment should succeed for tech_debt entity type")

	got, err := repo.GetActiveAssignment(ctx, "tech_debt", tdID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "tech_debt", got.EntityType)
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R05: AddAssignment — invalid entity type rejected
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_InvalidEntityType_Rejected(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S914")
	sprintID := seedTestSprint(t, repo, "S914")
	defer cleanupSprints(ctx, t, db, "S914")

	assignment := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "epic", // invalid
		EntityID:   1,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, assignment)
	assert.Error(t, err, "TC-R05: AddAssignment with invalid entity_type should return error")
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R06: AddAssignment — empty entity type rejected
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_EmptyEntityType_Rejected(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S915")
	sprintID := seedTestSprint(t, repo, "S915")
	defer cleanupSprints(ctx, t, db, "S915")

	assignment := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "", // empty
		EntityID:   1,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, assignment)
	assert.Error(t, err, "TC-R06: AddAssignment with empty entity_type should return error")
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R07: RemoveAssignment — task entity removed (soft-delete)
// ─────────────────────────────────────────────────────────────────────────────

func TestRemoveAssignment_TaskEntity_SetsRemovedAt(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S916")
	_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'T-E01-F01-916'`)

	sprintID := seedTestSprint(t, repo, "S916")
	defer cleanupSprints(ctx, t, db, "S916")

	taskID := seedTaskRow(ctx, t, db, "T-E01-F01-916", "Remove Test Task", "todo")
	defer db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID)

	// Add the assignment first.
	assignment := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "task",
		EntityID:   taskID,
		AssignedAt: time.Now(),
	}
	require.NoError(t, repo.AddAssignment(ctx, assignment))

	// TC-R07: Remove the assignment.
	err := repo.RemoveAssignment(ctx, sprintID, "task", taskID)
	assert.NoError(t, err, "TC-R07: RemoveAssignment should succeed for an active assignment")

	// After removal GetActiveAssignment should return nil.
	got, err := repo.GetActiveAssignment(ctx, "task", taskID)
	assert.NoError(t, err)
	assert.Nil(t, got, "TC-R07: GetActiveAssignment should return nil after removal")

	// Verify the row still exists in the DB with removed_at set (soft-delete).
	var removedAt *time.Time
	scanErr := database.QueryRowContext(ctx,
		`SELECT removed_at FROM sprint_assignments
		 WHERE sprint_id = ? AND entity_type = ? AND entity_id = ?`,
		sprintID, "task", taskID,
	).Scan(&removedAt)
	assert.NoError(t, scanErr)
	assert.NotNil(t, removedAt, "TC-R07: removed_at should be set (soft-delete, not hard-delete)")
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R08: RemoveAssignment — no active assignment returns error
// ─────────────────────────────────────────────────────────────────────────────

func TestRemoveAssignment_NoActiveAssignment_ReturnsError(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S917")
	sprintID := seedTestSprint(t, repo, "S917")
	defer cleanupSprints(ctx, t, db, "S917")

	// TC-R08: Remove a non-existent assignment (entityID 99999 not assigned).
	err := repo.RemoveAssignment(ctx, sprintID, "task", 99999)
	assert.Error(t, err, "TC-R08: RemoveAssignment on non-existent assignment should return error")
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R09: AddAssignment — duplicate active assignment returns conflict error
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_DuplicateActiveAssignment_ReturnsError(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S918", "S919")
	_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'T-E01-F01-918'`)

	sprintID1 := seedTestSprint(t, repo, "S918")
	sprintID2 := seedTestSprint(t, repo, "S919")
	defer cleanupSprints(ctx, t, db, "S918", "S919")

	taskID := seedTaskRow(ctx, t, db, "T-E01-F01-918", "Duplicate Assign Task", "todo")
	defer db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID)

	// First assignment — to sprint S918.
	a1 := &models.SprintAssignment{
		SprintID:   sprintID1,
		EntityType: "task",
		EntityID:   taskID,
		AssignedAt: time.Now(),
	}
	require.NoError(t, repo.AddAssignment(ctx, a1))

	// Second assignment — to sprint S919 (same entity, already active in S918).
	a2 := &models.SprintAssignment{
		SprintID:   sprintID2,
		EntityType: "task",
		EntityID:   taskID,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, a2)
	assert.Error(t, err, "TC-R09: duplicate active assignment should return error")

	// The error should reference the existing sprint key.
	// The partial unique index fires at the DB level; the repo wraps it with
	// context that includes the conflicting sprint information.
	assert.True(t, strings.Contains(strings.ToLower(err.Error()), "already") ||
		strings.Contains(err.Error(), "UNIQUE") ||
		strings.Contains(err.Error(), "conflict"),
		"TC-R09: error should indicate a uniqueness conflict, got: %v", err)
}

// ─────────────────────────────────────────────────────────────────────────────
// TC-R10: AddAssignment — re-add after removal succeeds (partial-unique-index
// only covers WHERE removed_at IS NULL)
// ─────────────────────────────────────────────────────────────────────────────

func TestAddAssignment_AllowsAfterRemoval(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S920")
	_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'T-E01-F01-920'`)

	sprintID := seedTestSprint(t, repo, "S920")
	defer cleanupSprints(ctx, t, db, "S920")

	taskID := seedTaskRow(ctx, t, db, "T-E01-F01-920", "Re-Assign Task", "todo")
	defer db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID)

	// Add first time.
	a := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "task",
		EntityID:   taskID,
		AssignedAt: time.Now(),
	}
	require.NoError(t, repo.AddAssignment(ctx, a))

	// Remove it (sets removed_at).
	require.NoError(t, repo.RemoveAssignment(ctx, sprintID, "task", taskID))

	// Add again — should succeed because the partial unique index only
	// enforces uniqueness WHERE removed_at IS NULL.
	a2 := &models.SprintAssignment{
		SprintID:   sprintID,
		EntityType: "task",
		EntityID:   taskID,
		AssignedAt: time.Now(),
	}
	err := repo.AddAssignment(ctx, a2)
	assert.NoError(t, err, "TC-R10: re-adding after removal should succeed")

	got, err := repo.GetActiveAssignment(ctx, "task", taskID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, taskID, got.EntityID)
}

// ─────────────────────────────────────────────────────────────────────────────
// ListAssignments tests
// ─────────────────────────────────────────────────────────────────────────────

func TestListAssignments_FilterByEntityType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S921")
	_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE key = 'T-E01-F01-921'`)
	_, _ = db.ExecContext(ctx, `DELETE FROM bugs WHERE key = 'B921'`)

	sprintID := seedTestSprint(t, repo, "S921")
	defer cleanupSprints(ctx, t, db, "S921")

	taskID := seedTaskRow(ctx, t, db, "T-E01-F01-921", "List Filter Task", "todo")
	defer db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID)

	bugID := seedBugRow(ctx, t, db, "B921", "List Filter Bug", "open")
	defer db.ExecContext(ctx, `DELETE FROM bugs WHERE id = ?`, bugID)

	// Add both a task and a bug assignment.
	require.NoError(t, repo.AddAssignment(ctx, &models.SprintAssignment{
		SprintID: sprintID, EntityType: "task", EntityID: taskID, AssignedAt: time.Now(),
	}))
	require.NoError(t, repo.AddAssignment(ctx, &models.SprintAssignment{
		SprintID: sprintID, EntityType: "bug", EntityID: bugID, AssignedAt: time.Now(),
	}))

	taskType := "task"
	assignments, err := repo.ListAssignments(ctx, sprintID, &taskType)
	require.NoError(t, err)
	assert.Len(t, assignments, 1, "ListAssignments with task filter should return 1 item")
	assert.Equal(t, "task", assignments[0].EntityType)
}

func TestListAssignments_IgnoresSoftDeleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	cleanupSprints(ctx, t, db, "S922")
	_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE key IN ('T-E01-F01-922', 'T-E01-F01-923')`)

	sprintID := seedTestSprint(t, repo, "S922")
	defer cleanupSprints(ctx, t, db, "S922")

	taskID1 := seedTaskRow(ctx, t, db, "T-E01-F01-922", "Active Task", "todo")
	defer db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID1)

	taskID2 := seedTaskRow(ctx, t, db, "T-E01-F01-923", "Removed Task", "todo")
	defer db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID2)

	// Add both assignments.
	require.NoError(t, repo.AddAssignment(ctx, &models.SprintAssignment{
		SprintID: sprintID, EntityType: "task", EntityID: taskID1, AssignedAt: time.Now(),
	}))
	require.NoError(t, repo.AddAssignment(ctx, &models.SprintAssignment{
		SprintID: sprintID, EntityType: "task", EntityID: taskID2, AssignedAt: time.Now(),
	}))

	// Remove the second one.
	require.NoError(t, repo.RemoveAssignment(ctx, sprintID, "task", taskID2))

	// ListAssignments should return only the active one.
	all, err := repo.ListAssignments(ctx, sprintID, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1, "ListAssignments should ignore soft-deleted rows")
	assert.Equal(t, taskID1, all[0].EntityID)
}
