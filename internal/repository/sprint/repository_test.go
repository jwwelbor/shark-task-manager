package sprint

import (
	"context"
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
