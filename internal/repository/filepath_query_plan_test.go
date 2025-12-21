package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestGetByFilePath_EpicQueryPlan(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Check GetByFilePath query plan for epics
	rows, err := database.QueryContext(ctx,
		"EXPLAIN QUERY PLAN SELECT id, key, title, description, status, priority, business_value, file_path, created_at, updated_at FROM epics WHERE file_path = ?",
		"test.md")
	assert.NoError(t, err)
	defer rows.Close()

	var plans []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		assert.NoError(t, err)
		plans = append(plans, detail)
		t.Logf("Epic GetByFilePath Query Plan: %s", detail)
	}

	// Verify that the index is being used
	planText := strings.Join(plans, " ")
	assert.Contains(t, planText, "idx_epics_file_path", "Query plan should use idx_epics_file_path index")
}

func TestGetByFilePath_FeatureQueryPlan(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Check GetByFilePath query plan for features
	rows, err := database.QueryContext(ctx,
		"EXPLAIN QUERY PLAN SELECT id, epic_id, key, title, description, status, progress_pct, execution_order, file_path, created_at, updated_at FROM features WHERE file_path = ?",
		"test.md")
	assert.NoError(t, err)
	defer rows.Close()

	var plans []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		assert.NoError(t, err)
		plans = append(plans, detail)
		t.Logf("Feature GetByFilePath Query Plan: %s", detail)
	}

	// Verify that the index is being used
	planText := strings.Join(plans, " ")
	assert.Contains(t, planText, "idx_features_file_path", "Query plan should use idx_features_file_path index")
}

func TestUpdateFilePath_EpicQueryPlan(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Check UpdateFilePath query plan for epics
	rows, err := database.QueryContext(ctx,
		"EXPLAIN QUERY PLAN UPDATE epics SET file_path = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?",
		"test.md", "E01")
	assert.NoError(t, err)
	defer rows.Close()

	var plans []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		assert.NoError(t, err)
		plans = append(plans, detail)
		t.Logf("Epic UpdateFilePath Query Plan: %s", detail)
	}

	// Verify that the key index is being used
	planText := strings.Join(plans, " ")
	assert.Contains(t, planText, "idx_epics_key", "Query plan should use idx_epics_key index for WHERE clause")
}

func TestUpdateFilePath_FeatureQueryPlan(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()

	// Check UpdateFilePath query plan for features
	rows, err := database.QueryContext(ctx,
		"EXPLAIN QUERY PLAN UPDATE features SET file_path = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?",
		"test.md", "E01-F01")
	assert.NoError(t, err)
	defer rows.Close()

	var plans []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		assert.NoError(t, err)
		plans = append(plans, detail)
		t.Logf("Feature UpdateFilePath Query Plan: %s", detail)
	}

	// Verify that the key index is being used
	planText := strings.Join(plans, " ")
	assert.Contains(t, planText, "idx_features_key", "Query plan should use idx_features_key index for WHERE clause")
}
