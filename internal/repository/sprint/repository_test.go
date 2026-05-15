package sprint

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
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
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S901', 'S902', 'S903')")
	require.NoError(t, cleanupErr)

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
			_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key = ?", tt.sprint.Key)
			require.NoError(t, cleanupErr)

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
			_, cleanupErr = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = ?", tt.sprint.Key)
			require.NoError(t, cleanupErr)
		})
	}
}

func TestSprintRepository_GetByKey_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")
	require.NoError(t, cleanupErr)

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
				assert.ErrorIs(t, err, repoerr.ErrNotFound)
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
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")
	require.NoError(t, cleanupErr)

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
	assert.ErrorIs(t, err, repoerr.ErrNotFound)
}

func TestSprintRepository_Update(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")
	require.NoError(t, cleanupErr)

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
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")
	require.NoError(t, cleanupErr)

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
	assert.ErrorIs(t, err, repoerr.ErrNotFound)
}

func TestSprintRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Clean up
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")
	require.NoError(t, cleanupErr)

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
	assert.ErrorIs(t, err, repoerr.ErrNotFound)
}

func TestSprintRepository_GetNextKey_Monotonic(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Keep this test isolated from any existing sprint rows because GetNextKey()
	// uses the table-wide maximum key.
	cleanup := func() {
		_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints")
		require.NoError(t, cleanupErr)
	}
	cleanup()
	t.Cleanup(cleanup)

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
			cleanup()

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
				require.NoError(t, repo.Create(ctx, sprint))
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
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S701', 'S702', 'S703')")
	require.NoError(t, cleanupErr)

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
		require.NoError(t, repo.Create(ctx, sprint))
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
			wantStatus: func() *models.SprintStatus {
				s := models.SprintStatus("planning")
				return &s
			}(),
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
			wantStatus: func() *models.SprintStatus {
				s := models.SprintStatus("active")
				return &s
			}(),
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
			wantStatus: func() *models.SprintStatus {
				s := models.SprintStatus("closed")
				return &s
			}(),
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
			wantStatus: func() *models.SprintStatus {
				s := models.SprintStatus("archived")
				return &s
			}(),
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

			if tt.wantStatus != nil {
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
	_, cleanupErr := database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S901'")
	require.NoError(t, cleanupErr)

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

	// Verify Update also keeps user input parameterized across all string fields.
	retrieved.Goal = "Goal'; DROP TABLE sprints; --"
	retrieved.Name = "Updated'; UPDATE sprints SET status='closed'; --"
	retrieved.Slug = "slug'; DROP TABLE sprints; --"
	retrieved.FilePath = "/tmp/sprint'; DROP TABLE sprints; --.md"

	err = repo.Update(ctx, retrieved)
	assert.NoError(t, err)

	updated, err := repo.GetByKey(ctx, sprint.Key)
	assert.NoError(t, err)
	assert.Equal(t, "Updated'; UPDATE sprints SET status='closed'; --", updated.Name)
	assert.Equal(t, "Goal'; DROP TABLE sprints; --", updated.Goal)
	assert.Equal(t, "slug'; DROP TABLE sprints; --", updated.Slug)
	assert.Equal(t, "/tmp/sprint'; DROP TABLE sprints; --.md", updated.FilePath)

	// Verify the sprints table still exists
	var count int
	require.NoError(t, database.QueryRowContext(ctx, "SELECT COUNT(*) FROM sprints").Scan(&count))
	assert.Greater(t, count, 0)
}

// ---------------------------------------------------------------------------
// Test helpers for T-E19-F03-003 (backlog UNION query and entity ID helpers)
// ---------------------------------------------------------------------------

// seedTestSprint inserts a sprint row for use in backlog/assignment tests.
// Returns the inserted sprint's DB id.
func seedTestSprint(t *testing.T, database *sql.DB, key, status string) int64 {
	t.Helper()
	_, _ = database.ExecContext(context.Background(), "DELETE FROM sprints WHERE key = ?", key)
	result, err := database.ExecContext(context.Background(), `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		key, "Test Sprint "+key, "test goal",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		status,
	)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// seedEntityRow inserts a row into the appropriate entity table (tasks, bugs,
// change_cards, or tech_debts) and returns the inserted row's DB id.
// For entities that lack agent_type or priority, pass empty string / 0.
func seedEntityRow(t *testing.T, database *sql.DB, entityType, key, title, status string) int64 {
	t.Helper()
	ctx := context.Background()

	var result sql.Result
	var err error

	switch entityType {
	case "task":
		// Ensure a valid feature exists for FK constraint
		_, featureID := test.SeedTestData()
		// Clean up first
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", key)
		result, err = database.ExecContext(ctx, `
			INSERT INTO tasks (key, title, status, agent_type, priority, feature_id)
			VALUES (?, ?, ?, ?, ?, ?)`,
			key, title, status, "backend", 5, featureID,
		)
	case "bug":
		_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key = ?", key)
		result, err = database.ExecContext(ctx, `
			INSERT INTO bugs (key, title, status, severity)
			VALUES (?, ?, ?, ?)`,
			key, title, status, "medium",
		)
	case "change_card":
		_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", key)
		result, err = database.ExecContext(ctx, `
			INSERT INTO change_cards (key, title, status, priority)
			VALUES (?, ?, ?, ?)`,
			key, title, status, 5,
		)
	case "tech_debt":
		_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key = ?", key)
		result, err = database.ExecContext(ctx, `
			INSERT INTO tech_debts (key, title, status, category, severity)
			VALUES (?, ?, ?, ?, ?)`,
			key, title, status, "code-quality", "medium",
		)
	default:
		t.Fatalf("unsupported entityType: %q", entityType)
	}

	require.NoError(t, err, "seedEntityRow(%s, %s)", entityType, key)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// seedSprintAssignment inserts a sprint_assignments row directly into the DB.
// Returns the inserted row's id.
func seedSprintAssignment(t *testing.T, database *sql.DB, sprintID int64, entityType string, entityID int64) int64 {
	t.Helper()
	result, err := database.ExecContext(context.Background(), `
		INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		sprintID, entityType, entityID,
	)
	require.NoError(t, err)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// cleanupTestEntities removes seeded entity rows by key from all entity tables.
func cleanupTestEntities(t *testing.T, database *sql.DB, pairs [][2]string) {
	t.Helper()
	ctx := context.Background()
	tableMap := map[string]string{
		"task":        "tasks",
		"bug":         "bugs",
		"change_card": "change_cards",
		"tech_debt":   "tech_debts",
	}
	for _, pair := range pairs {
		entityType, key := pair[0], pair[1]
		table, ok := tableMap[entityType]
		if !ok {
			continue
		}
		//nolint:gosec // table is from a hardcoded map; no user input
		_, _ = database.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE key = ?", table), key)
	}
}

// cleanupSprintAndAssignments removes a sprint and all its assignments.
func cleanupSprintAndAssignments(t *testing.T, database *sql.DB, sprintID int64) {
	t.Helper()
	ctx := context.Background()
	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id = ?", sprintID)
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprintID)
}

// ---------------------------------------------------------------------------
// TC-B01: ListBacklog — tasks returned with entity_type="task"
// TC-B02: ListBacklog — bug entity type returned with entity_type="bug"
// TC-B03: ListBacklog — change_card entity type returned
// TC-B04: ListBacklog — tech_debt entity type returned
// ---------------------------------------------------------------------------

// TestListBacklog_AllEntityTypes verifies that the UNION ALL query returns rows
// from all four entity tables with correct entity_type labels (TC-B01..TC-B04).
func TestListBacklog_AllEntityTypes(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Seed a sprint
	sprintID := seedTestSprint(t, database, "S980", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	// Seed one entity of each type and assign to the sprint
	taskID := seedEntityRow(t, database, "task", "T-B01-F01-001", "Backlog Task", "todo")
	bugID := seedEntityRow(t, database, "bug", "B901", "Backlog Bug", "reported")
	ccID := seedEntityRow(t, database, "change_card", "CC-901", "Backlog CC", "proposed")
	tdID := seedEntityRow(t, database, "tech_debt", "TD-901", "Backlog TD", "identified")

	defer cleanupTestEntities(t, database, [][2]string{
		{"task", "T-B01-F01-001"},
		{"bug", "B901"},
		{"change_card", "CC-901"},
		{"tech_debt", "TD-901"},
	})

	_ = seedSprintAssignment(t, database, sprintID, "task", taskID)
	_ = seedSprintAssignment(t, database, sprintID, "bug", bugID)
	_ = seedSprintAssignment(t, database, sprintID, "change_card", ccID)
	_ = seedSprintAssignment(t, database, sprintID, "tech_debt", tdID)

	// Execute ListBacklog (nil entityType = all types)
	items, err := repo.ListBacklog(ctx, sprintID, nil, false)
	require.NoError(t, err)
	require.Len(t, items, 4, "expected 4 backlog items (one per entity type)")

	// Build a map from entity_type → item for assertions
	byType := make(map[string]*BacklogItem)
	for _, item := range items {
		byType[item.EntityType] = item
	}

	// TC-B01: task row is present with correct label and non-empty fields
	taskItem, ok := byType["task"]
	require.True(t, ok, "expected task row in backlog")
	assert.Equal(t, "task", taskItem.EntityType)
	assert.Equal(t, "T-B01-F01-001", taskItem.EntityKey)
	assert.Equal(t, "Backlog Task", taskItem.Title)
	assert.NotEmpty(t, taskItem.Status)
	assert.Equal(t, sprintID, taskItem.SprintID)

	// TC-B02: bug row
	bugItem, ok := byType["bug"]
	require.True(t, ok, "expected bug row in backlog")
	assert.Equal(t, "bug", bugItem.EntityType)
	assert.Equal(t, "B901", bugItem.EntityKey)
	assert.Equal(t, "Backlog Bug", bugItem.Title)

	// TC-B03: change_card row
	ccItem, ok := byType["change_card"]
	require.True(t, ok, "expected change_card row in backlog")
	assert.Equal(t, "change_card", ccItem.EntityType)
	assert.Equal(t, "CC-901", ccItem.EntityKey)

	// TC-B04: tech_debt row
	tdItem, ok := byType["tech_debt"]
	require.True(t, ok, "expected tech_debt row in backlog")
	assert.Equal(t, "tech_debt", tdItem.EntityType)
	assert.Equal(t, "TD-901", tdItem.EntityKey)
}

// TestListBacklog_FilterByEntityType verifies that the entityType filter limits
// results to the requested type only (part of TC-B06 / AC-5).
func TestListBacklog_FilterByEntityType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	sprintID := seedTestSprint(t, database, "S981", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	taskID := seedEntityRow(t, database, "task", "T-B06-F01-001", "Filter Task", "todo")
	bugID := seedEntityRow(t, database, "bug", "B902", "Filter Bug", "reported")
	defer cleanupTestEntities(t, database, [][2]string{
		{"task", "T-B06-F01-001"},
		{"bug", "B902"},
	})

	_ = seedSprintAssignment(t, database, sprintID, "task", taskID)
	_ = seedSprintAssignment(t, database, sprintID, "bug", bugID)

	entityType := "task"
	items, err := repo.ListBacklog(ctx, sprintID, &entityType, false)
	require.NoError(t, err)
	require.Len(t, items, 1, "filter by task should return only 1 item")
	assert.Equal(t, "task", items[0].EntityType)
	assert.Equal(t, "T-B06-F01-001", items[0].EntityKey)
}

// TestListBacklog_ExcludesSoftDeleted verifies that assignments with removed_at
// set are not returned by ListBacklog.
func TestListBacklog_ExcludesSoftDeleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	sprintID := seedTestSprint(t, database, "S982", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	taskID := seedEntityRow(t, database, "task", "T-B07-F01-001", "Removed Task", "todo")
	activeTaskID := seedEntityRow(t, database, "task", "T-B07-F01-002", "Active Task", "todo")
	defer cleanupTestEntities(t, database, [][2]string{
		{"task", "T-B07-F01-001"},
		{"task", "T-B07-F01-002"},
	})

	// Assign both, then soft-delete first
	assignID := seedSprintAssignment(t, database, sprintID, "task", taskID)
	_, err := database.ExecContext(ctx, "UPDATE sprint_assignments SET removed_at = CURRENT_TIMESTAMP WHERE id = ?", assignID)
	require.NoError(t, err)

	_ = seedSprintAssignment(t, database, sprintID, "task", activeTaskID)

	items, err := repo.ListBacklog(ctx, sprintID, nil, false)
	require.NoError(t, err)
	require.Len(t, items, 1, "soft-deleted assignment should not appear in backlog")
	assert.Equal(t, "T-B07-F01-002", items[0].EntityKey)
}

// TestListAssignmentsForCarryover_ExcludesCompleted verifies that entities in
// completed-equivalent statuses are not returned (TC-C07 carryover logic).
func TestListAssignmentsForCarryover_ExcludesCompleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	sprintID := seedTestSprint(t, database, "S983", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	// completed task — should NOT be returned by carryover query
	completedTaskID := seedEntityRow(t, database, "task", "T-CRY-F01-001", "Completed Task", "completed")
	// in_progress task — SHOULD be returned
	incompleteTaskID := seedEntityRow(t, database, "task", "T-CRY-F01-002", "Incomplete Task", "in_progress")
	// todo bug — SHOULD be returned
	todoBugID := seedEntityRow(t, database, "bug", "B-CRY-001", "Todo Bug", "reported")

	defer cleanupTestEntities(t, database, [][2]string{
		{"task", "T-CRY-F01-001"},
		{"task", "T-CRY-F01-002"},
		{"bug", "B-CRY-001"},
	})

	_ = seedSprintAssignment(t, database, sprintID, "task", completedTaskID)
	_ = seedSprintAssignment(t, database, sprintID, "task", incompleteTaskID)
	_ = seedSprintAssignment(t, database, sprintID, "bug", todoBugID)

	assignments, err := repo.ListAssignmentsForCarryover(ctx, sprintID)
	require.NoError(t, err)

	// Only the 2 incomplete should be returned; the completed task is excluded
	require.Len(t, assignments, 2, "carryover query must exclude completed entities")

	keys := make(map[string]bool)
	for _, a := range assignments {
		keys[fmt.Sprintf("%s:%d", a.EntityType, a.EntityID)] = true
	}
	assert.True(t, keys[fmt.Sprintf("task:%d", incompleteTaskID)], "incomplete task should be in carryover")
	assert.True(t, keys[fmt.Sprintf("bug:%d", todoBugID)], "todo bug should be in carryover")
	assert.False(t, keys[fmt.Sprintf("task:%d", completedTaskID)], "completed task must NOT be in carryover")
}

// TestListAssignmentsForCarryover_EmptyWhenAllCompleted verifies BVA lower
// bound: all entities completed → empty slice returned.
func TestListAssignmentsForCarryover_EmptyWhenAllCompleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	sprintID := seedTestSprint(t, database, "S984", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	taskID := seedEntityRow(t, database, "task", "T-CRY2-F01-001", "Done Task", "completed")
	defer cleanupTestEntities(t, database, [][2]string{{"task", "T-CRY2-F01-001"}})

	_ = seedSprintAssignment(t, database, sprintID, "task", taskID)

	assignments, err := repo.ListAssignmentsForCarryover(ctx, sprintID, "completed")
	require.NoError(t, err)
	assert.Empty(t, assignments, "all completed sprint should return empty carryover list")
}

// ---------------------------------------------------------------------------
// TC-B05: GetSprintBacklog — completion percentage BVA
// These test the UNION query's ability to return correct status data so the
// service can compute completion percentages.
// ---------------------------------------------------------------------------

// TestListBacklog_StatusFieldPresent verifies that BacklogItem.Status is
// populated correctly from each entity table (foundation for TC-B05).
func TestListBacklog_StatusFieldPresent(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	sprintID := seedTestSprint(t, database, "S985", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	taskID := seedEntityRow(t, database, "task", "T-B05-F01-001", "Status Task", "in_progress")
	bugID := seedEntityRow(t, database, "bug", "B905", "Status Bug", "reported")

	defer cleanupTestEntities(t, database, [][2]string{
		{"task", "T-B05-F01-001"},
		{"bug", "B905"},
	})

	_ = seedSprintAssignment(t, database, sprintID, "task", taskID)
	_ = seedSprintAssignment(t, database, sprintID, "bug", bugID)

	items, err := repo.ListBacklog(ctx, sprintID, nil, false)
	require.NoError(t, err)
	require.Len(t, items, 2)

	for _, item := range items {
		assert.NotEmpty(t, item.Status, "Status must be populated for entity_type=%s", item.EntityType)
	}

	// Verify specific status values
	byType := make(map[string]*BacklogItem)
	for _, item := range items {
		byType[item.EntityType] = item
	}
	assert.Equal(t, "in_progress", byType["task"].Status)
	assert.Equal(t, "reported", byType["bug"].Status)
}

// ---------------------------------------------------------------------------
// TC-B08: ListBacklog — blockedOnly=true returns only entities in blocked statuses
// TC-B09: ListBacklog — blockedOnly=true excludes entities NOT in blocked statuses
// ---------------------------------------------------------------------------

// TestListBacklog_BlockedOnly_ReturnsOnlyBlocked exercises the blockedOnly=true
// path of ListBacklog with the bug entity type (TC-B08). It verifies that only
// the entity with a status matching the provided blockedStatuses list is returned,
// and that the arg ordering bug (sprintID vs blockedStatuses order) is caught:
// if the args were inverted the query would either produce wrong results or a
// runtime error.
func TestListBacklog_BlockedOnly_ReturnsOnlyBlocked(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	sprintID := seedTestSprint(t, database, "S986", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	// One bug in a "blocked" status, one bug in a non-blocked status.
	blockedBugID := seedEntityRow(t, database, "bug", "B-BLK-001", "Blocked Bug", "blocked")
	activeBugID := seedEntityRow(t, database, "bug", "B-BLK-002", "Active Bug", "in_progress")
	defer cleanupTestEntities(t, database, [][2]string{
		{"bug", "B-BLK-001"},
		{"bug", "B-BLK-002"},
	})

	_ = seedSprintAssignment(t, database, sprintID, "bug", blockedBugID)
	_ = seedSprintAssignment(t, database, sprintID, "bug", activeBugID)

	// Call ListBacklog with blockedOnly=true and the "blocked" status in the set.
	items, err := repo.ListBacklog(ctx, sprintID, nil, true, "blocked")
	require.NoError(t, err)

	// Only the blocked bug should appear.
	require.Len(t, items, 1, "blockedOnly=true should return only the entity in a blocked status")
	assert.Equal(t, "bug", items[0].EntityType)
	assert.Equal(t, "B-BLK-001", items[0].EntityKey)
	assert.Equal(t, "blocked", items[0].Status)
}

// TestListBacklog_BlockedOnly_ExcludesNonBlocked exercises the blockedOnly=true
// path (TC-B09). It verifies that entities NOT in any of the provided blocked
// statuses are excluded even when they have active assignments in the sprint.
//
// Uses change_card entities (no feature_id FK cascade) to avoid the cascade-
// delete side-effect of calling seedEntityRow("task") twice (each call invokes
// test.SeedTestData() which deletes+recreates the E99 feature, cascade-deleting
// tasks created in the previous call). change_cards have no such FK dependency.
//
// The multi-placeholder blockedStatuses path ("blocked", "needs_info") is
// exercised here to confirm the arg-ordering fix works across both cardinalities.
func TestListBacklog_BlockedOnly_ExcludesNonBlocked(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	sprintID := seedTestSprint(t, database, "S987", "active")
	defer cleanupSprintAndAssignments(t, database, sprintID)

	// Two change_cards: one in a blocked-equivalent status, one not.
	// change_cards have no feature_id FK so calling seedEntityRow twice is safe.
	blockedCCID := seedEntityRow(t, database, "change_card", "CC-BLK-001", "Blocked CC", "blocked")
	activeCCID := seedEntityRow(t, database, "change_card", "CC-BLK-002", "Active CC", "in_progress")
	defer cleanupTestEntities(t, database, [][2]string{
		{"change_card", "CC-BLK-001"},
		{"change_card", "CC-BLK-002"},
	})

	_ = seedSprintAssignment(t, database, sprintID, "change_card", blockedCCID)
	_ = seedSprintAssignment(t, database, sprintID, "change_card", activeCCID)

	// blockedOnly=true with two blocked-status values exercises the multi-placeholder path.
	// Correct arg ordering: sprintID before blockedStatuses for each sub-select.
	items, err := repo.ListBacklog(ctx, sprintID, nil, true, "blocked", "needs_info")
	require.NoError(t, err)

	// Only the blocked change_card should appear; in_progress is excluded.
	require.Len(t, items, 1, "blockedOnly=true must exclude entities not in blocked statuses")
	assert.Equal(t, "change_card", items[0].EntityType)
	assert.Equal(t, "CC-BLK-001", items[0].EntityKey)

	// Verify blockedOnly=false returns both (sanity check).
	allItems, err := repo.ListBacklog(ctx, sprintID, nil, false)
	require.NoError(t, err)
	assert.Len(t, allItems, 2, "blockedOnly=false should return both change_cards")
}

// ---------------------------------------------------------------------------
// Entity ID helpers: GetTaskIDByKey, GetBugIDByKey, GetChangeCardIDByKey,
//                   GetTechDebtIDByKey
// ---------------------------------------------------------------------------

func TestGetTaskIDByKey_Found(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	taskID := seedEntityRow(t, database, "task", "T-GID-F01-001", "ID Lookup Task", "todo")
	defer cleanupTestEntities(t, database, [][2]string{{"task", "T-GID-F01-001"}})

	got, err := repo.GetTaskIDByKey(ctx, "T-GID-F01-001")
	require.NoError(t, err)
	assert.Equal(t, taskID, got)
}

func TestGetTaskIDByKey_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	taskID := seedEntityRow(t, database, "task", "T-GID-F01-002", "Case Task", "todo")
	defer cleanupTestEntities(t, database, [][2]string{{"task", "T-GID-F01-002"}})

	got, err := repo.GetTaskIDByKey(ctx, "t-gid-f01-002") // lowercase
	require.NoError(t, err)
	assert.Equal(t, taskID, got)
}

func TestGetTaskIDByKey_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, err := repo.GetTaskIDByKey(ctx, "T-NONEXISTENT-999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetBugIDByKey_Found(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	bugID := seedEntityRow(t, database, "bug", "B910", "ID Bug", "reported")
	defer cleanupTestEntities(t, database, [][2]string{{"bug", "B910"}})

	got, err := repo.GetBugIDByKey(ctx, "B910")
	require.NoError(t, err)
	assert.Equal(t, bugID, got)
}

func TestGetBugIDByKey_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, err := repo.GetBugIDByKey(ctx, "B99999")
	require.Error(t, err)
}

func TestGetChangeCardIDByKey_Found(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	ccID := seedEntityRow(t, database, "change_card", "CC-910", "ID CC", "proposed")
	defer cleanupTestEntities(t, database, [][2]string{{"change_card", "CC-910"}})

	got, err := repo.GetChangeCardIDByKey(ctx, "CC-910")
	require.NoError(t, err)
	assert.Equal(t, ccID, got)
}

func TestGetChangeCardIDByKey_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, err := repo.GetChangeCardIDByKey(ctx, "CC-99999")
	require.Error(t, err)
}

func TestGetTechDebtIDByKey_Found(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	tdID := seedEntityRow(t, database, "tech_debt", "TD-910", "ID TD", "identified")
	defer cleanupTestEntities(t, database, [][2]string{{"tech_debt", "TD-910"}})

	got, err := repo.GetTechDebtIDByKey(ctx, "TD-910")
	require.NoError(t, err)
	assert.Equal(t, tdID, got)
}

func TestGetTechDebtIDByKey_NotFound(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, err := repo.GetTechDebtIDByKey(ctx, "TD-99999")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Benchmark stubs: TC-P01, TC-P02
// Run with: go test -bench=. ./internal/repository/sprint/
// ---------------------------------------------------------------------------

func BenchmarkListBacklog_200Items(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	// Set up a sprint with 200 assignments (50 per entity type)
	sprintID := seedBenchmarkSprint(b, database)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id = ?", sprintID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprintID)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.ListBacklog(ctx, sprintID, nil, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// seedBenchmarkSprint seeds a sprint with 200 assignments for benchmark tests.
func seedBenchmarkSprint(b *testing.B, database *sql.DB) int64 {
	b.Helper()
	ctx := context.Background()

	// Create benchmark sprint
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S-BENCH-001'")
	result, err := database.ExecContext(ctx, `
		INSERT INTO sprints (key, name, goal, start_date, end_date, status)
		VALUES ('S-BENCH-001', 'Benchmark Sprint', 'perf test',
		        '2026-01-01', '2026-01-14', 'active')`)
	if err != nil {
		b.Fatalf("failed to create benchmark sprint: %v", err)
	}
	sprintID, _ := result.LastInsertId()

	// Seed 50 tasks
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("T-BENCH-F01-%03d", i+1)
		_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key = ?", key)
		res, err := database.ExecContext(ctx,
			`INSERT INTO tasks (key, title, status, priority, epic_id, feature_id) VALUES (?, ?, 'todo', 5, 1, 1)`,
			key, "Bench Task "+key)
		if err != nil {
			b.Fatalf("failed to seed task %s: %v", key, err)
		}
		entityID, _ := res.LastInsertId()
		_, _ = database.ExecContext(ctx,
			`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at) VALUES (?, 'task', ?, CURRENT_TIMESTAMP)`,
			sprintID, entityID)
	}

	// Seed 50 bugs
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("B-BENCH-%03d", i+1)
		_, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key = ?", key)
		res, err := database.ExecContext(ctx,
			`INSERT INTO bugs (key, title, status, severity) VALUES (?, ?, 'reported', 'medium')`,
			key, "Bench Bug "+key)
		if err != nil {
			b.Fatalf("failed to seed bug %s: %v", key, err)
		}
		entityID, _ := res.LastInsertId()
		_, _ = database.ExecContext(ctx,
			`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at) VALUES (?, 'bug', ?, CURRENT_TIMESTAMP)`,
			sprintID, entityID)
	}

	// Seed 50 change_cards
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("CC-BENCH-%03d", i+1)
		_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", key)
		res, err := database.ExecContext(ctx,
			`INSERT INTO change_cards (key, title, status, priority) VALUES (?, ?, 'proposed', 5)`,
			key, "Bench CC "+key)
		if err != nil {
			b.Fatalf("failed to seed change_card %s: %v", key, err)
		}
		entityID, _ := res.LastInsertId()
		_, _ = database.ExecContext(ctx,
			`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at) VALUES (?, 'change_card', ?, CURRENT_TIMESTAMP)`,
			sprintID, entityID)
	}

	// Seed 50 tech_debts
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("TD-BENCH-%03d", i+1)
		_, _ = database.ExecContext(ctx, "DELETE FROM tech_debts WHERE key = ?", key)
		res, err := database.ExecContext(ctx,
			`INSERT INTO tech_debts (key, title, status, category, severity) VALUES (?, ?, 'identified', 'code-quality', 'medium')`,
			key, "Bench TD "+key)
		if err != nil {
			b.Fatalf("failed to seed tech_debt %s: %v", key, err)
		}
		entityID, _ := res.LastInsertId()
		_, _ = database.ExecContext(ctx,
			`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at) VALUES (?, 'tech_debt', ?, CURRENT_TIMESTAMP)`,
			sprintID, entityID)
	}

	return sprintID
}

// --------------------------------------------------------------------------
// Helpers for Tx method tests
// --------------------------------------------------------------------------

// createTestSprintForTx inserts a sprint into the DB and returns its ID.
func createTestSprintForTx(t *testing.T, database *sql.DB, repo *SprintRepository, key, status string) int64 {
	t.Helper()
	ctx := context.Background()
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = ?", key)
	sprint := &models.Sprint{
		Key:       key,
		Name:      "Test Sprint " + key,
		Goal:      "Test goal",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    models.SprintStatus(status),
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err, "createTestSprintForTx: %s", key)
	return sprint.ID
}

// insertAssignment inserts an active sprint_assignments row and returns its ID.
func insertAssignment(t *testing.T, database *sql.DB, sprintID int64, entityType string, entityID int64) int64 {
	t.Helper()
	ctx := context.Background()
	result, err := database.ExecContext(ctx,
		`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		sprintID, entityType, entityID,
	)
	require.NoError(t, err, "insertAssignment: sprint=%d type=%s entity=%d", sprintID, entityType, entityID)
	id, err := result.LastInsertId()
	require.NoError(t, err)
	return id
}

// --------------------------------------------------------------------------
// TC-C08 subset: ReassignToSprintTx updates sprint_id for listed IDs
// --------------------------------------------------------------------------

func TestSprintRepository_ReassignToSprintTx_UpdatesSprintID(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key IN ('S910', 'S911'))")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S910', 'S911')")

	sourceID := createTestSprintForTx(t, database, repo, "S910", "active")
	destID := createTestSprintForTx(t, database, repo, "S911", "planning")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (?, ?)", sourceID, destID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S910', 'S911')")
	}()

	a1 := insertAssignment(t, database, sourceID, "task", 99901)
	a2 := insertAssignment(t, database, sourceID, "task", 99902)

	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	err = repo.ReassignToSprintTx(ctx, tx, []int64{a1, a2}, destID)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	rows, err := database.QueryContext(ctx,
		`SELECT sprint_id FROM sprint_assignments WHERE id IN (?, ?)`, a1, a2)
	require.NoError(t, err)
	defer rows.Close()

	var sprintIDs []int64
	for rows.Next() {
		var sid int64
		require.NoError(t, rows.Scan(&sid))
		sprintIDs = append(sprintIDs, sid)
	}
	require.NoError(t, rows.Err())
	assert.Len(t, sprintIDs, 2)
	for _, sid := range sprintIDs {
		assert.Equal(t, destID, sid, "expected sprint_id to be updated to destID")
	}
}

// --------------------------------------------------------------------------
// TC-C08 subset: DropAssignmentsTx sets removed_at for listed IDs
// --------------------------------------------------------------------------

func TestSprintRepository_DropAssignmentsTx_SetsRemovedAt(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S912')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S912'")

	sprintID := createTestSprintForTx(t, database, repo, "S912", "active")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id = ?", sprintID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S912'")
	}()

	a1 := insertAssignment(t, database, sprintID, "task", 99903)
	a2 := insertAssignment(t, database, sprintID, "bug", 99904)

	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	err = repo.DropAssignmentsTx(ctx, tx, []int64{a1, a2})
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	rows, err := database.QueryContext(ctx,
		`SELECT removed_at FROM sprint_assignments WHERE id IN (?, ?)`, a1, a2)
	require.NoError(t, err)
	defer rows.Close()

	var count int
	for rows.Next() {
		var ra sql.NullString
		require.NoError(t, rows.Scan(&ra))
		assert.True(t, ra.Valid, "removed_at should be set (not NULL)")
		count++
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, 2, count)
}

// --------------------------------------------------------------------------
// TC-C08: CreateCompletionTx inserts row and is queryable after commit
// --------------------------------------------------------------------------

func TestSprintRepository_CreateCompletionTx_InsertsRow(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_completions WHERE sprint_id IN (SELECT id FROM sprints WHERE key IN ('S913', 'S919'))")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S913', 'S919')")

	// Create the sprint that will be closed
	sprintID := createTestSprintForTx(t, database, repo, "S913", "active")
	// Create a real next sprint to satisfy the FK constraint on next_sprint_id
	nextSprintID := createTestSprintForTx(t, database, repo, "S919", "planning")

	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_completions WHERE sprint_id = ?", sprintID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S913', 'S919')")
	}()

	plannedSize := 20.5
	completedSize := 12.0

	completion := &models.SprintCompletion{
		SprintID:             sprintID,
		CompletedAt:          time.Now().UTC(),
		PlannedEntityCount:   8,
		CompletedEntityCount: 5,
		CarriedOverCount:     3,
		DroppedCount:         0,
		PlannedSizeSum:       &plannedSize,
		CompletedSizeSum:     &completedSize,
		CarryoverMode:        "next",
		NextSprintID:         &nextSprintID,
	}

	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	err = repo.CreateCompletionTx(ctx, tx, completion)
	require.NoError(t, err)
	assert.NotZero(t, completion.ID, "CreateCompletionTx must set completion.ID")

	err = tx.Commit()
	require.NoError(t, err)

	var (
		gotSprintID         int64
		gotPlannedCount     int
		gotCompletedCount   int
		gotCarriedOverCount int
		gotDroppedCount     int
		gotCarryoverMode    string
		gotNextSprintID     sql.NullInt64
	)
	err = database.QueryRowContext(ctx,
		`SELECT sprint_id, planned_entity_count, completed_entity_count,
		        carried_over_count, dropped_count, carryover_mode, next_sprint_id
		 FROM sprint_completions WHERE id = ?`, completion.ID,
	).Scan(
		&gotSprintID,
		&gotPlannedCount,
		&gotCompletedCount,
		&gotCarriedOverCount,
		&gotDroppedCount,
		&gotCarryoverMode,
		&gotNextSprintID,
	)
	require.NoError(t, err)

	assert.Equal(t, sprintID, gotSprintID)
	assert.Equal(t, 8, gotPlannedCount)
	assert.Equal(t, 5, gotCompletedCount)
	assert.Equal(t, 3, gotCarriedOverCount)
	assert.Equal(t, 0, gotDroppedCount)
	assert.Equal(t, "next", gotCarryoverMode)
	assert.True(t, gotNextSprintID.Valid)
	assert.Equal(t, nextSprintID, gotNextSprintID.Int64)
}

// --------------------------------------------------------------------------
// Edge: ReassignToSprintTx with empty slice is a no-op
// --------------------------------------------------------------------------

func TestSprintRepository_ReassignToSprintTx_EmptySliceNoOp(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S914'")
	sprintID := createTestSprintForTx(t, database, repo, "S914", "active")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S914'")
	}()
	_ = sprintID

	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	err = repo.ReassignToSprintTx(ctx, tx, []int64{}, sprintID)
	assert.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, err)
}

// --------------------------------------------------------------------------
// Edge: DropAssignmentsTx with empty slice is a no-op
// --------------------------------------------------------------------------

func TestSprintRepository_DropAssignmentsTx_EmptySliceNoOp(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S915'")
	_ = createTestSprintForTx(t, database, repo, "S915", "active")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S915'")
	}()

	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	err = repo.DropAssignmentsTx(ctx, tx, []int64{})
	assert.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, err)
}

// --------------------------------------------------------------------------
// Edge: CreateCompletionTx with nil optional fields
// --------------------------------------------------------------------------

func TestSprintRepository_CreateCompletionTx_NilOptionalFields(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_completions WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S916')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S916'")

	sprintID := createTestSprintForTx(t, database, repo, "S916", "active")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_completions WHERE sprint_id = ?", sprintID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S916'")
	}()

	completion := &models.SprintCompletion{
		SprintID:             sprintID,
		CompletedAt:          time.Now().UTC(),
		PlannedEntityCount:   3,
		CompletedEntityCount: 3,
		CarriedOverCount:     0,
		DroppedCount:         0,
		PlannedSizeSum:       nil,
		CompletedSizeSum:     nil,
		CarryoverMode:        "backlog",
		NextSprintID:         nil,
	}

	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	err = repo.CreateCompletionTx(ctx, tx, completion)
	require.NoError(t, err)
	assert.NotZero(t, completion.ID)

	err = tx.Commit()
	require.NoError(t, err)

	var gotNextSprintID sql.NullInt64
	var gotPlannedSize, gotCompletedSize sql.NullFloat64
	err = database.QueryRowContext(ctx,
		`SELECT next_sprint_id, planned_size_sum, completed_size_sum
		 FROM sprint_completions WHERE id = ?`, completion.ID,
	).Scan(&gotNextSprintID, &gotPlannedSize, &gotCompletedSize)
	require.NoError(t, err)

	assert.False(t, gotNextSprintID.Valid, "next_sprint_id should be NULL for backlog mode")
	assert.False(t, gotPlannedSize.Valid, "planned_size_sum should be NULL for unsized")
	assert.False(t, gotCompletedSize.Valid, "completed_size_sum should be NULL for unsized")
}

// --------------------------------------------------------------------------
// Rollback verification: Tx methods do not persist on rollback
// --------------------------------------------------------------------------

func TestSprintRepository_TxMethods_RollbackLeavesNoTrace(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key IN ('S917', 'S918'))")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S917', 'S918')")

	sourceID := createTestSprintForTx(t, database, repo, "S917", "active")
	destID := createTestSprintForTx(t, database, repo, "S918", "planning")
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (?, ?)", sourceID, destID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('S917', 'S918')")
	}()

	a1 := insertAssignment(t, database, sourceID, "task", 99905)

	tx, err := database.BeginTx(ctx, nil)
	require.NoError(t, err)

	err = repo.ReassignToSprintTx(ctx, tx, []int64{a1}, destID)
	require.NoError(t, err)

	// Rollback instead of commit
	err = tx.Rollback()
	require.NoError(t, err)

	// Verify sprint_id was NOT changed (still sourceID)
	var gotSprintID int64
	err = database.QueryRowContext(ctx,
		`SELECT sprint_id FROM sprint_assignments WHERE id = ?`, a1).Scan(&gotSprintID)
	require.NoError(t, err)
	assert.Equal(t, sourceID, gotSprintID, "rollback must not persist sprint_id change")
}

// --------------------------------------------------------------------------
// TC-P03 stub: BenchmarkSprintClose_200Entities
// --------------------------------------------------------------------------

// BenchmarkSprintClose_200Entities is the performance benchmark stub for TC-P03.
// Run with: go test -bench=BenchmarkSprintClose_200Entities -benchtime=1x ./internal/repository/sprint/
//
// This stub exercises ReassignToSprintTx + CreateCompletionTx against 200
// assignments and validates the <2s target per REQ-NF-001.
func BenchmarkSprintClose_200Entities(b *testing.B) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key IN ('SB01', 'SB02'))")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_completions WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'SB01')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key IN ('SB01', 'SB02')")

	srcSprint := &models.Sprint{
		Key:       "SB01",
		Name:      "Bench Sprint 1",
		Goal:      "Benchmark",
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}
	err := repo.Create(ctx, srcSprint)
	if err != nil {
		b.Fatalf("benchmark setup: create source sprint: %v", err)
	}

	dstSprint := &models.Sprint{
		Key:       "SB02",
		Name:      "Bench Sprint 2",
		Goal:      "Benchmark",
		StartDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err = repo.Create(ctx, dstSprint)
	if err != nil {
		b.Fatalf("benchmark setup: create dest sprint: %v", err)
	}

	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (?, ?)", srcSprint.ID, dstSprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_completions WHERE sprint_id = ?", srcSprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id IN (?, ?)", srcSprint.ID, dstSprint.ID)
	}()

	assignmentIDs := make([]int64, 0, 200)
	for i := 0; i < 200; i++ {
		result, err := database.ExecContext(ctx,
			`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at)
			 VALUES (?, 'task', ?, CURRENT_TIMESTAMP)`,
			srcSprint.ID, int64(90000+i),
		)
		if err != nil {
			b.Fatalf("benchmark setup: insert assignment %d: %v", i, err)
		}
		id, _ := result.LastInsertId()
		assignmentIDs = append(assignmentIDs, id)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			b.Fatalf("BeginTx: %v", err)
		}

		if err := repo.ReassignToSprintTx(ctx, tx, assignmentIDs, dstSprint.ID); err != nil {
			_ = tx.Rollback()
			b.Fatalf("ReassignToSprintTx: %v", err)
		}

		completion := &models.SprintCompletion{
			SprintID:             srcSprint.ID,
			CompletedAt:          time.Now().UTC(),
			PlannedEntityCount:   200,
			CompletedEntityCount: 100,
			CarriedOverCount:     100,
			DroppedCount:         0,
			CarryoverMode:        "next",
			NextSprintID:         &dstSprint.ID,
		}
		if err := repo.CreateCompletionTx(ctx, tx, completion); err != nil {
			_ = tx.Rollback()
			b.Fatalf("CreateCompletionTx: %v", err)
		}

		// Rollback to keep DB clean between benchmark iterations
		if err := tx.Rollback(); err != nil {
			b.Fatalf("Rollback: %v", err)
		}
	}

	b.StopTimer()
}

// ─── E19-F05-002: Capacity CRUD and backlog query tests ───────────────────────

// TC-014-01: SetCapacity creates new row; GetCapacity returns it.
func TestSprintRepository_SetCapacity_Insert(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_capacity WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S961')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S961'")
	sprint := &models.Sprint{
		Key:       "S961",
		Name:      "Capacity Test Sprint",
		Goal:      "Test capacity",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_capacity WHERE sprint_id = ?", sprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)
	}()

	capacity := &models.SprintCapacity{
		SprintID:       sprint.ID,
		AgentType:      "backend",
		CapacityPoints: 21,
	}
	err = repo.SetCapacity(ctx, capacity)
	require.NoError(t, err)

	rows, err := repo.GetCapacity(ctx, sprint.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, sprint.ID, rows[0].SprintID)
	assert.Equal(t, "backend", rows[0].AgentType)
	assert.InDelta(t, 21.0, rows[0].CapacityPoints, 0.001)
}

// TC-014-02: SetCapacity upserts — calling twice keeps only one row with updated value.
func TestSprintRepository_SetCapacity_Upsert(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_capacity WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S962')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S962'")
	sprint := &models.Sprint{
		Key:       "S962",
		Name:      "Upsert Test Sprint",
		Goal:      "Upsert",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_capacity WHERE sprint_id = ?", sprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)
	}()

	err = repo.SetCapacity(ctx, &models.SprintCapacity{
		SprintID: sprint.ID, AgentType: "backend", CapacityPoints: 21,
	})
	require.NoError(t, err)

	err = repo.SetCapacity(ctx, &models.SprintCapacity{
		SprintID: sprint.ID, AgentType: "backend", CapacityPoints: 34,
	})
	require.NoError(t, err)

	rows, err := repo.GetCapacity(ctx, sprint.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1, "upsert should not create duplicate row")
	assert.InDelta(t, 34.0, rows[0].CapacityPoints, 0.001)
}

// TC-014-03 (adapted): GetCapacity returns empty slice (not error) when no rows exist.
func TestSprintRepository_GetCapacity_Empty(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S963'")
	sprint := &models.Sprint{
		Key:       "S963",
		Name:      "Empty Capacity Sprint",
		Goal:      "Empty",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err := repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)

	rows, err := repo.GetCapacity(ctx, sprint.ID)
	assert.NoError(t, err)
	assert.NotNil(t, rows, "should return non-nil empty slice")
	assert.Empty(t, rows, "should be empty when no capacity rows")
}

// TC-011-05: ListUnassignedBacklog completes < 500ms for 500 tasks.
func TestSprintRepository_ListUnassignedBacklog_Performance(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'PERF-E01-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'PERF-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'PERF-E01'")

	var epicID int64
	err := database.QueryRowContext(ctx,
		`INSERT INTO epics (key, title, status, priority) VALUES ('PERF-E01', 'Perf Epic', 'active', 'medium') RETURNING id`).Scan(&epicID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	var featureID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO features (epic_id, key, title, status) VALUES (?, 'PERF-F01', 'Perf Feature', 'in_progress') RETURNING id`,
		epicID).Scan(&featureID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("PERF-E01-F01-%03d", i+1)
		_, err = database.ExecContext(ctx,
			`INSERT OR IGNORE INTO tasks (feature_id, key, title, status, priority) VALUES (?, ?, ?, 'ready_for_development', 5)`,
			featureID, key, "Perf Task "+key)
		require.NoError(t, err)
	}
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'PERF-E01-F01-%'")

	start := time.Now()
	items, err := repo.ListUnassignedBacklog(ctx, []string{"task"})
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(items), 500,
		"should return at least 500 items (the seeded tasks)")
	assert.Less(t, elapsed.Milliseconds(), int64(500),
		"ListUnassignedBacklog must complete in < 500ms for 500 entities, got %dms", elapsed.Milliseconds())
}

// TestSprintRepository_ListUnassignedBacklog_ExcludesAssigned checks that tasks
// assigned to active/planning sprints are excluded.
func TestSprintRepository_ListUnassignedBacklog_ExcludesAssigned(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S971')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S971'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'BL-E01-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'BL-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'BL-E01'")

	var epicID int64
	err := database.QueryRowContext(ctx,
		`INSERT INTO epics (key, title, status, priority) VALUES ('BL-E01', 'Backlog Epic', 'active', 'medium') RETURNING id`).Scan(&epicID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	var featureID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO features (epic_id, key, title, status) VALUES (?, 'BL-F01', 'Backlog Feature', 'in_progress') RETURNING id`,
		epicID).Scan(&featureID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	// Task A: unassigned — should appear in backlog.
	var taskAID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority) VALUES (?, 'BL-E01-F01-001', 'Task A', 'ready_for_development', 5) RETURNING id`,
		featureID).Scan(&taskAID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskAID)

	// Task B: assigned to active sprint — should be excluded.
	var taskBID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority) VALUES (?, 'BL-E01-F01-002', 'Task B', 'ready_for_development', 5) RETURNING id`,
		featureID).Scan(&taskBID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskBID)

	sprint := &models.Sprint{
		Key: "S971", Name: "Active Sprint", Goal: "Active",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}
	err = repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id = ?", sprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)
	}()

	_, err = database.ExecContext(ctx,
		`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at) VALUES (?, 'task', ?, CURRENT_TIMESTAMP)`,
		sprint.ID, taskBID)
	require.NoError(t, err)

	items, err := repo.ListUnassignedBacklog(ctx, []string{"task"})
	require.NoError(t, err)

	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	assert.Contains(t, keys, "BL-E01-F01-001", "unassigned task should be in backlog")
	assert.NotContains(t, keys, "BL-E01-F01-002", "assigned task should not be in backlog")
}

// TestSprintRepository_ListUnassignedBacklog_ExcludesCompleted checks that tasks
// in terminal statuses (completed, archived, cancelled) are excluded.
func TestSprintRepository_ListUnassignedBacklog_ExcludesCompleted(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'BLC-E01-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'BLC-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'BLC-E01'")

	var epicID int64
	err := database.QueryRowContext(ctx,
		`INSERT INTO epics (key, title, status, priority) VALUES ('BLC-E01', 'Completed Epic', 'active', 'medium') RETURNING id`).Scan(&epicID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	var featureID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO features (epic_id, key, title, status) VALUES (?, 'BLC-F01', 'Completed Feature', 'in_progress') RETURNING id`,
		epicID).Scan(&featureID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	var eligibleID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority) VALUES (?, 'BLC-E01-F01-001', 'Eligible', 'ready_for_development', 5) RETURNING id`,
		featureID).Scan(&eligibleID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", eligibleID)

	var completedID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority) VALUES (?, 'BLC-E01-F01-002', 'Done', 'completed', 5) RETURNING id`,
		featureID).Scan(&completedID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", completedID)

	items, err := repo.ListUnassignedBacklog(ctx, []string{"task"})
	require.NoError(t, err)

	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	assert.Contains(t, keys, "BLC-E01-F01-001")
	assert.NotContains(t, keys, "BLC-E01-F01-002", "completed task should not be in backlog")
}

// TC-012-09: BulkAssign inserts all assignments; GetAssignmentsWithSize returns them.
func TestSprintRepository_BulkAssign_InsertsAll(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S981')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S981'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'BA-E01-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'BA-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'BA-E01'")

	var epicID int64
	err := database.QueryRowContext(ctx,
		`INSERT INTO epics (key, title, status, priority) VALUES ('BA-E01', 'BulkAssign Epic', 'active', 'medium') RETURNING id`).Scan(&epicID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	var featureID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO features (epic_id, key, title, status) VALUES (?, 'BA-F01', 'BulkAssign Feature', 'in_progress') RETURNING id`,
		epicID).Scan(&featureID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	var taskID1, taskID2 int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority, size) VALUES (?, 'BA-E01-F01-001', 'Task 1', 'ready_for_development', 5, 3) RETURNING id`,
		featureID).Scan(&taskID1)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID1)

	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority, size) VALUES (?, 'BA-E01-F01-002', 'Task 2', 'ready_for_development', 5, 5) RETURNING id`,
		featureID).Scan(&taskID2)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID2)

	sprint := &models.Sprint{
		Key: "S981", Name: "BulkAssign Sprint", Goal: "Bulk",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err = repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id = ?", sprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)
	}()

	assignments := []models.SprintAssignment{
		{SprintID: sprint.ID, EntityType: "task", EntityID: taskID1},
		{SprintID: sprint.ID, EntityType: "task", EntityID: taskID2},
	}

	count, err := repo.BulkAssign(ctx, sprint.ID, assignments)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	assigned, err := repo.GetAssignmentsWithSize(ctx, sprint.ID)
	require.NoError(t, err)
	assert.Len(t, assigned, 2)
}

// TC-012-10: BulkAssign skips duplicate assignment without error; returns 0.
func TestSprintRepository_BulkAssign_SkipsDuplicate(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S982')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S982'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'DUP-E01-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'DUP-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'DUP-E01'")

	var epicID int64
	err := database.QueryRowContext(ctx,
		`INSERT INTO epics (key, title, status, priority) VALUES ('DUP-E01', 'Dup Epic', 'active', 'medium') RETURNING id`).Scan(&epicID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	var featureID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO features (epic_id, key, title, status) VALUES (?, 'DUP-F01', 'Dup Feature', 'in_progress') RETURNING id`,
		epicID).Scan(&featureID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	var taskID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority) VALUES (?, 'DUP-E01-F01-001', 'Dup Task', 'ready_for_development', 5) RETURNING id`,
		featureID).Scan(&taskID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)

	sprint := &models.Sprint{
		Key: "S982", Name: "Dup Sprint", Goal: "Dup",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err = repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id = ?", sprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)
	}()

	// First assignment.
	count, err := repo.BulkAssign(ctx, sprint.ID, []models.SprintAssignment{
		{SprintID: sprint.ID, EntityType: "task", EntityID: taskID},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Re-assign same entity — should skip, return 0, no error.
	count, err = repo.BulkAssign(ctx, sprint.ID, []models.SprintAssignment{
		{SprintID: sprint.ID, EntityType: "task", EntityID: taskID},
	})
	assert.NoError(t, err, "duplicate assignment should not return error")
	assert.Equal(t, 0, count, "should skip duplicate and return 0 inserted")
}

// TestSprintRepository_GetAssignmentsWithSize_MultiType verifies tasks with
// size values are returned correctly from GetAssignmentsWithSize.
func TestSprintRepository_GetAssignmentsWithSize_MultiType(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewSprintRepository(db)

	_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id IN (SELECT id FROM sprints WHERE key = 'S983')")
	_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE key = 'S983'")
	_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'AWS-E01-F01-%'")
	_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key = 'AWS-F01'")
	_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'AWS-E01'")

	var epicID int64
	err := database.QueryRowContext(ctx,
		`INSERT INTO epics (key, title, status, priority) VALUES ('AWS-E01', 'AWS Epic', 'active', 'medium') RETURNING id`).Scan(&epicID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epicID)

	var featureID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO features (epic_id, key, title, status) VALUES (?, 'AWS-F01', 'AWS Feature', 'in_progress') RETURNING id`,
		epicID).Scan(&featureID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", featureID)

	var taskID int64
	err = database.QueryRowContext(ctx,
		`INSERT INTO tasks (feature_id, key, title, status, priority, size, agent_type) VALUES (?, 'AWS-E01-F01-001', 'Sized Task', 'ready_for_development', 5, 5, 'backend') RETURNING id`,
		featureID).Scan(&taskID)
	require.NoError(t, err)
	defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", taskID)

	sprint := &models.Sprint{
		Key: "S983", Name: "AWS Sprint", Goal: "AWS",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
		Status:    "planning",
	}
	err = repo.Create(ctx, sprint)
	require.NoError(t, err)
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM sprint_assignments WHERE sprint_id = ?", sprint.ID)
		_, _ = database.ExecContext(ctx, "DELETE FROM sprints WHERE id = ?", sprint.ID)
	}()

	_, err = database.ExecContext(ctx,
		`INSERT INTO sprint_assignments (sprint_id, entity_type, entity_id, assigned_at) VALUES (?, 'task', ?, CURRENT_TIMESTAMP)`,
		sprint.ID, taskID)
	require.NoError(t, err)

	assigned, err := repo.GetAssignmentsWithSize(ctx, sprint.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 1)

	size5 := 5
	assert.Equal(t, "task", assigned[0].EntityType)
	assert.Equal(t, taskID, assigned[0].EntityID)
	assert.Equal(t, "AWS-E01-F01-001", assigned[0].Key)
	assert.Equal(t, &size5, assigned[0].Size)
	require.NotNil(t, assigned[0].AgentType)
	assert.Equal(t, "backend", *assigned[0].AgentType)
}
