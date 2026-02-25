package repository

import (
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// NOTE: This file tests repository methods which directly interact with the database.
// Repository tests SHOULD use the real database to verify SQL queries work correctly.
//
// For other layers (services, CLI, etc.), MOCK the repository methods instead of using
// the real database. You already know what the repository will return, so mock it.
//
// Testing Philosophy:
// - Repository layer: Real database (tests SQL correctness)
// - Service layer: Mock repositories (tests business logic)
// - CLI layer: Mock services (tests command handling)
// - Integration tests: Real database (tests end-to-end)

// Helper to create epic, feature, and tasks for testing.
// Uses INSERT OR IGNORE pattern like SeedTestData() to handle shared test database.
// Epics E90-E99 are reserved for progress tests.
func setupProgressTest(t *testing.T, epicNum int, featureNum int, taskStatuses []models.TaskStatus) (int64, int64) {
	database := test.GetTestDB()

	// Use E90-E99 range for progress tests
	epicKey := fmt.Sprintf("E%02d", epicNum)
	featureKey := fmt.Sprintf("E%02d-F%02d", epicNum, featureNum)

	// Clean up any existing test data for this SPECIFIC feature to ensure test isolation
	_, _ = database.Exec("DELETE FROM tasks WHERE feature_id IN (SELECT id FROM features WHERE key = ?)", featureKey)
	_, _ = database.Exec("DELETE FROM features WHERE key = ?", featureKey)

	// Create epic via SQL using INSERT OR IGNORE (multiple features may use same epic)
	_, err := database.Exec(`
		INSERT OR IGNORE INTO epics (key, title, description, status, priority)
		VALUES (?, 'Progress Test Epic', 'Test epic', 'active', 'medium')
	`, epicKey)
	if err != nil {
		t.Fatalf("Failed to create epic %s: %v", epicKey, err)
	}

	// Get the epic ID (whether it was just created or already existed)
	var epicID int64
	err = database.QueryRow("SELECT id FROM epics WHERE key = ?", epicKey).Scan(&epicID)
	if err != nil {
		t.Fatalf("Failed to get epic ID for %s: %v", epicKey, err)
	}
	t.Logf("Using epic %s with ID=%d", epicKey, epicID)

	// Create feature via SQL
	_, err = database.Exec(`
		INSERT INTO features (epic_id, key, title, description, status)
		VALUES (?, ?, 'Progress Test Feature', 'Test feature', 'active')
	`, epicID, featureKey)
	if err != nil {
		t.Fatalf("Failed to create feature %s with epicID=%d: %v", featureKey, epicID, err)
	}

	// Get the feature ID
	var featureID int64
	err = database.QueryRow("SELECT id FROM features WHERE key = ?", featureKey).Scan(&featureID)
	if err != nil {
		t.Fatalf("Failed to get feature ID for %s: %v", featureKey, err)
	}

	// Create tasks
	for i, status := range taskStatuses {
		taskKey := fmt.Sprintf("%s-T%03d", featureKey, i+1)
		_, err := database.Exec(`
			INSERT INTO tasks (feature_id, key, title, status, agent_type, priority, depends_on)
			VALUES (?, ?, ?, ?, 'testing', 1, '[]')
		`, featureID, taskKey, fmt.Sprintf("Task %d", i+1), status)
		if err != nil {
			t.Fatalf("Failed to create task %s with status %s: %v", taskKey, status, err)
		}
	}

	return epicID, featureID
}

// NOTE: Feature-level progress calculation tests (TestFeatureProgress_*) were removed because
// FeatureRepository.CalculateProgress was migrated to FeatureService.CalculateProgress
// as part of E15-F06. Feature progress tests now live in epic_service_test.go and
// feature_service_test.go using mocked repositories.

// NOTE: Epic-level progress calculation tests (TestEpicProgress_*) were removed because
// EpicRepository.CalculateProgress was migrated to EpicService.CalculateProgress
// as part of E15-F05. Epic progress tests now live in epic_service_test.go
// using mocked repositories.
