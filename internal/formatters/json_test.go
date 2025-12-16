package formatters

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatEpicListJSON(t *testing.T) {
	tests := []struct {
		name     string
		epics    []*EpicWithProgress
		validate func(t *testing.T, output string)
	}{
		{
			name: "empty list",
			epics: []*EpicWithProgress{},
			validate: func(t *testing.T, output string) {
				var result struct {
					Results []interface{} `json:"results"`
					Count   int           `json:"count"`
				}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, 0, result.Count)
				assert.Empty(t, result.Results)
			},
		},
		{
			name: "single epic",
			epics: []*EpicWithProgress{
				{
					Epic: &models.Epic{
						ID:          1,
						Key:         "E01",
						Title:       "Test Epic",
						Description: stringPtr("Test Description"),
						Status:      models.EpicStatusActive,
						Priority:    models.PriorityHigh,
						CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					ProgressPct: 75.5,
				},
			},
			validate: func(t *testing.T, output string) {
				var result struct {
					Results []map[string]interface{} `json:"results"`
					Count   int                      `json:"count"`
				}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, 1, result.Count)
				require.Len(t, result.Results, 1)

				epic := result.Results[0]
				assert.Equal(t, float64(1), epic["id"])
				assert.Equal(t, "E01", epic["key"])
				assert.Equal(t, "Test Epic", epic["title"])
				assert.Equal(t, "Test Description", epic["description"])
				assert.Equal(t, "active", epic["status"])
				assert.Equal(t, "high", epic["priority"])
				assert.Equal(t, 75.5, epic["progress_pct"])
				assert.NotNil(t, epic["created_at"])
				assert.NotNil(t, epic["updated_at"])
			},
		},
		{
			name: "multiple epics",
			epics: []*EpicWithProgress{
				{
					Epic: &models.Epic{
						ID:       1,
						Key:      "E01",
						Title:    "Epic 1",
						Status:   models.EpicStatusActive,
						Priority: models.PriorityHigh,
					},
					ProgressPct: 50.0,
				},
				{
					Epic: &models.Epic{
						ID:       2,
						Key:      "E02",
						Title:    "Epic 2",
						Status:   models.EpicStatusCompleted,
						Priority: models.PriorityMedium,
					},
					ProgressPct: 100.0,
				},
			},
			validate: func(t *testing.T, output string) {
				var result struct {
					Results []map[string]interface{} `json:"results"`
					Count   int                      `json:"count"`
				}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, 2, result.Count)
				require.Len(t, result.Results, 2)
				assert.Equal(t, "E01", result.Results[0]["key"])
				assert.Equal(t, "E02", result.Results[1]["key"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := FormatEpicListJSON(tt.epics)
			require.NoError(t, err)
			tt.validate(t, output)
		})
	}
}

func TestFormatEpicGetJSON(t *testing.T) {
	tests := []struct {
		name     string
		epic     *EpicWithProgress
		features []*FeatureWithTaskCount
		validate func(t *testing.T, output string)
	}{
		{
			name: "epic with no features",
			epic: &EpicWithProgress{
				Epic: &models.Epic{
					ID:       1,
					Key:      "E01",
					Title:    "Test Epic",
					Status:   models.EpicStatusActive,
					Priority: models.PriorityHigh,
				},
				ProgressPct: 0.0,
			},
			features: []*FeatureWithTaskCount{},
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, "E01", result["key"])
				assert.Equal(t, 0.0, result["progress_pct"])
				features, ok := result["features"].([]interface{})
				require.True(t, ok)
				assert.Empty(t, features)
			},
		},
		{
			name: "epic with features",
			epic: &EpicWithProgress{
				Epic: &models.Epic{
					ID:       1,
					Key:      "E01",
					Title:    "Test Epic",
					Status:   models.EpicStatusActive,
					Priority: models.PriorityHigh,
				},
				ProgressPct: 75.0,
			},
			features: []*FeatureWithTaskCount{
				{
					Feature: &models.Feature{
						ID:          1,
						EpicID:      1,
						Key:         "E01-F01",
						Title:       "Feature 1",
						Status:      models.FeatureStatusActive,
						ProgressPct: 50.0,
					},
					TaskCount: 10,
				},
				{
					Feature: &models.Feature{
						ID:          2,
						EpicID:      1,
						Key:         "E01-F02",
						Title:       "Feature 2",
						Status:      models.FeatureStatusCompleted,
						ProgressPct: 100.0,
					},
					TaskCount: 5,
				},
			},
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, "E01", result["key"])
				assert.Equal(t, 75.0, result["progress_pct"])

				features, ok := result["features"].([]interface{})
				require.True(t, ok)
				require.Len(t, features, 2)

				feature1 := features[0].(map[string]interface{})
				assert.Equal(t, "E01-F01", feature1["key"])
				assert.Equal(t, 50.0, feature1["progress_pct"])
				assert.Equal(t, float64(10), feature1["task_count"])

				feature2 := features[1].(map[string]interface{})
				assert.Equal(t, "E01-F02", feature2["key"])
				assert.Equal(t, 100.0, feature2["progress_pct"])
				assert.Equal(t, float64(5), feature2["task_count"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := FormatEpicGetJSON(tt.epic, tt.features)
			require.NoError(t, err)
			tt.validate(t, output)
		})
	}
}

func TestFormatFeatureListJSON(t *testing.T) {
	tests := []struct {
		name     string
		features []*FeatureWithTaskCount
		validate func(t *testing.T, output string)
	}{
		{
			name:     "empty list",
			features: []*FeatureWithTaskCount{},
			validate: func(t *testing.T, output string) {
				var result struct {
					Results []interface{} `json:"results"`
					Count   int           `json:"count"`
				}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, 0, result.Count)
				assert.Empty(t, result.Results)
			},
		},
		{
			name: "single feature",
			features: []*FeatureWithTaskCount{
				{
					Feature: &models.Feature{
						ID:          1,
						EpicID:      1,
						Key:         "E01-F01",
						Title:       "Test Feature",
						Description: stringPtr("Test Description"),
						Status:      models.FeatureStatusActive,
						ProgressPct: 75.5,
						CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					TaskCount: 10,
				},
			},
			validate: func(t *testing.T, output string) {
				var result struct {
					Results []map[string]interface{} `json:"results"`
					Count   int                      `json:"count"`
				}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, 1, result.Count)
				require.Len(t, result.Results, 1)

				feature := result.Results[0]
				assert.Equal(t, float64(1), feature["id"])
				assert.Equal(t, float64(1), feature["epic_id"])
				assert.Equal(t, "E01-F01", feature["key"])
				assert.Equal(t, "Test Feature", feature["title"])
				assert.Equal(t, "Test Description", feature["description"])
				assert.Equal(t, "active", feature["status"])
				assert.Equal(t, 75.5, feature["progress_pct"])
				assert.Equal(t, float64(10), feature["task_count"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := FormatFeatureListJSON(tt.features)
			require.NoError(t, err)
			tt.validate(t, output)
		})
	}
}

func TestFormatFeatureGetJSON(t *testing.T) {
	tests := []struct {
		name      string
		feature   *models.Feature
		tasks     []*models.Task
		breakdown map[models.TaskStatus]int
		validate  func(t *testing.T, output string)
	}{
		{
			name: "feature with no tasks",
			feature: &models.Feature{
				ID:          1,
				EpicID:      1,
				Key:         "E01-F01",
				Title:       "Test Feature",
				Status:      models.FeatureStatusActive,
				ProgressPct: 0.0,
			},
			tasks:     []*models.Task{},
			breakdown: map[models.TaskStatus]int{},
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, "E01-F01", result["key"])
				assert.Equal(t, 0.0, result["progress_pct"])
				tasks, ok := result["tasks"].([]interface{})
				require.True(t, ok)
				assert.Empty(t, tasks)
				breakdown, ok := result["status_breakdown"].(map[string]interface{})
				require.True(t, ok)
				assert.Empty(t, breakdown)
			},
		},
		{
			name: "feature with tasks",
			feature: &models.Feature{
				ID:          1,
				EpicID:      1,
				Key:         "E01-F01",
				Title:       "Test Feature",
				Status:      models.FeatureStatusActive,
				ProgressPct: 70.0,
			},
			tasks: []*models.Task{
				{
					ID:        1,
					FeatureID: 1,
					Key:       "E01-F01-T01",
					Title:     "Task 1",
					Status:    models.TaskStatusCompleted,
					Priority:  5,
				},
				{
					ID:        2,
					FeatureID: 1,
					Key:       "E01-F01-T02",
					Title:     "Task 2",
					Status:    models.TaskStatusInProgress,
					Priority:  3,
				},
			},
			breakdown: map[models.TaskStatus]int{
				models.TaskStatusCompleted:  1,
				models.TaskStatusInProgress: 1,
			},
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, "E01-F01", result["key"])
				assert.Equal(t, 70.0, result["progress_pct"])

				tasks, ok := result["tasks"].([]interface{})
				require.True(t, ok)
				require.Len(t, tasks, 2)

				task1 := tasks[0].(map[string]interface{})
				assert.Equal(t, "E01-F01-T01", task1["key"])
				assert.Equal(t, "completed", task1["status"])

				breakdown, ok := result["status_breakdown"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, float64(1), breakdown["completed"])
				assert.Equal(t, float64(1), breakdown["in_progress"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := FormatFeatureGetJSON(tt.feature, tt.tasks, tt.breakdown)
			require.NoError(t, err)
			tt.validate(t, output)
		})
	}
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}
