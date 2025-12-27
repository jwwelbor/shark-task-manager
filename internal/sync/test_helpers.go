package sync

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/stretchr/testify/require"
)

// setupTestDatabase creates and initializes a test database using db.InitDB to get all migrations
func setupTestDatabase(tb testing.TB, dbPath string) *sql.DB {
	database, err := db.InitDB(dbPath)
	require.NoError(tb, err)
	return database
}

// setupTestEpicAndFeature creates a test epic and feature
func setupTestEpicAndFeature(tb testing.TB, db *sql.DB) {
	ctx := context.Background()
	repoDb := repository.NewDB(db)
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)

	// Create epic
	epic := &models.Epic{
		Key:      "E01",
		Title:    "Test Epic",
		Status:   models.EpicStatusActive,
		Priority: models.PriorityMedium,
	}
	err := epicRepo.Create(ctx, epic)
	require.NoError(tb, err)

	// Create feature
	feature := &models.Feature{
		EpicID: epic.ID,
		Key:    "E01-F01",
		Title:  "Test Feature",
		Status: models.FeatureStatusActive,
	}
	err = featureRepo.Create(ctx, feature)
	require.NoError(tb, err)
}

// createTestTaskFile creates a task file with the given key and title
func createTestTaskFile(tb testing.TB, filePath, taskKey, title string) {
	content := fmt.Sprintf(`---
task_key: %s
status: todo
---

# Task: %s

## Description

Test task description for %s.
`, taskKey, title, taskKey)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(tb, err)
}

// updateTestTaskFile updates a task file with new content
func updateTestTaskFile(tb testing.TB, filePath, taskKey, title string) {
	content := fmt.Sprintf(`---
task_key: %s
status: todo
---

# Task: %s

## Description

Updated test task description for %s.
`, taskKey, title, taskKey)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(tb, err)
}
