package repository

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSearchAllTestDB creates an in-memory DB for SearchAll tests.
func setupSearchAllTestDB(t *testing.T) *DB {
	t.Helper()
	testDB, err := db.InitDB(":memory:")
	require.NoError(t, err)
	return &DB{DB: testDB}
}

// seedSearchAllTestData inserts minimal epics/features/tasks/bugs/change_cards.
func seedSearchAllTestData(t *testing.T, repoDb *DB) {
	t.Helper()
	ctx := context.Background()

	// Epic
	epicRepo := NewEpicRepository(repoDb)
	epic := &models.Epic{BaseEntity: models.BaseEntity{Key: "E01", Title: "Login System"}, Status: "active", Priority: "high"}
	require.NoError(t, epicRepo.Create(ctx, epic))

	// Feature
	featureRepo := NewFeatureRepository(repoDb)
	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: "E01-F01", Title: "Authentication"}, EpicID: epic.ID, Status: "active"}
	require.NoError(t, featureRepo.Create(ctx, feature))

	// Task
	taskRepo := NewTaskRepository(repoDb)
	taskDesc := "Implement login endpoint"
	agent := "backend"
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E01-F01-001",
		Title:       "Implement login flow",
		Description: &taskDesc}, FeatureID: feature.ID,

		Status:    "todo",
		Priority:  5,
		AgentType: &agent,
	}
	require.NoError(t, taskRepo.Create(ctx, task))

	// Bug
	bugRepo := NewBugRepository(repoDb)
	bugDesc := "Login button fails on mobile"
	bug := &models.Bug{BaseEntity: models.BaseEntity{Key: "B001",
		Title:       "Login button broken",
		Description: &bugDesc}, Status: models.BugStatus("reported"),
		Severity: models.BugSeverityHigh,
	}
	require.NoError(t, bugRepo.Create(ctx, bug))

	// Change-Card
	ccRepo := NewChangeCardRepository(repoDb)
	ccDesc := "Add dark mode support to login page"
	cc := &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001",
		Title:       "Dark mode login",
		Description: &ccDesc}, Status: models.ChangeCardStatus("proposed"),
		Priority: 5,
	}
	require.NoError(t, ccRepo.Create(ctx, cc))
}

// --- SearchAll tests ---

func TestSearchAll_ReturnsAllEntityTypes(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	// "login" matches the task title, bug title, and change-card description
	results, err := repo.SearchAll(ctx, "login", nil)
	require.NoError(t, err)

	typesSeen := map[string]bool{}
	for _, r := range results {
		typesSeen[r.EntityType] = true
	}

	assert.True(t, typesSeen["task"], "expected task in results")
	assert.True(t, typesSeen["bug"], "expected bug in results")
	assert.True(t, typesSeen["change"], "expected change-card in results")
}

func TestSearchAll_EmptyQueryReturnsEmpty(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	results, err := repo.SearchAll(ctx, "", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchAll_TypeFilterBug(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	entityType := "bug"
	results, err := repo.SearchAll(ctx, "login", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.Equal(t, "bug", r.EntityType)
	}
}

func TestSearchAll_TypeFilterChange(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	entityType := "change"
	results, err := repo.SearchAll(ctx, "dark mode", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.Equal(t, "change", r.EntityType)
	}
}

func TestSearchAll_TypeFilterTask(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	entityType := "task"
	results, err := repo.SearchAll(ctx, "login", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.Equal(t, "task", r.EntityType)
	}
}

func TestSearchAll_BugResultIncludesSeverity(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	entityType := "bug"
	results, err := repo.SearchAll(ctx, "login", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		if r.EntityType == "bug" {
			assert.NotEmpty(t, r.Severity, "bug result should have severity")
		}
	}
}

func TestSearchAll_ChangeResultHasNoSeverity(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	entityType := "change"
	results, err := repo.SearchAll(ctx, "dark mode", &entityType)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		if r.EntityType == "change" {
			assert.Empty(t, r.Severity, "change-card result should not have severity")
		}
	}
}

func TestSearchAll_ResultsHaveRequiredFields(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	results, err := repo.SearchAll(ctx, "login", nil)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.NotEmpty(t, r.EntityType, "EntityType must be set")
		assert.NotEmpty(t, r.Key, "Key must be set")
		assert.NotEmpty(t, r.Title, "Title must be set")
		assert.NotEmpty(t, r.Status, "Status must be set")
	}
}

func TestSearchAll_NoMatchReturnsEmpty(t *testing.T) {
	repoDb := setupSearchAllTestDB(t)
	defer repoDb.Close()

	seedSearchAllTestData(t, repoDb)
	repo := NewSearchRepository(repoDb)
	ctx := context.Background()

	results, err := repo.SearchAll(ctx, "xyznonexistent12345", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}
