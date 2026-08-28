package commands

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/status"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/pterm/pterm"
)

// Test Suite 2.4: buildFeaturePlanningBasicInfo()
func TestBuildFeaturePlanningBasicInfo(t *testing.T) {
	desc := "A feature description"

	tests := []struct {
		name     string
		info     *services.FeatureDisplayInfo
		wantKeys []string // Expected label names in order
	}{
		{
			name: "TC-planningBasicInfo-001: Minimal feature returns Title, Epic ID, Status",
			info: &services.FeatureDisplayInfo{
				Feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Auth Feature"}, EpicID: 7,
					Status: models.FeatureStatusActive,
				},
			},
			wantKeys: []string{"Title", "Epic ID", "Status"},
		},
		{
			name: "TC-planningBasicInfo-002: Feature with phase and description includes all fields",
			info: &services.FeatureDisplayInfo{
				Feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Auth Feature",

					Description: &desc}, EpicID: 7,
					Status: models.FeatureStatusDraft,
				},
				Phase:            "development",
				PhaseDescription: "Active development phase",
				ResolvedPath:     "docs/plan/E07/E07-F01",
			},
			wantKeys: []string{"Title", "Epic ID", "Status", "Phase", "Phase Description", "Path", "Description"},
		},
		{
			name: "TC-planningBasicInfo-003: Empty optional fields are omitted",
			info: &services.FeatureDisplayInfo{
				Feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Simple Feature"}, EpicID: 1,
					Status: models.FeatureStatusDraft,
				},
				Phase:            "",
				PhaseDescription: "",
				ResolvedPath:     "",
			},
			wantKeys: []string{"Title", "Epic ID", "Status"},
		},
		{
			name: "TC-planningBasicInfo-004: Nil description is omitted",
			info: &services.FeatureDisplayInfo{
				Feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "No Desc Feature",

					Description: nil}, EpicID: 3,
					Status: models.FeatureStatusActive,
				},
			},
			wantKeys: []string{"Title", "Epic ID", "Status"},
		},
		{
			name: "TC-planningBasicInfo-005: Empty string description is omitted",
			info: &services.FeatureDisplayInfo{
				Feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Empty Desc Feature",

					Description: strPtr("")}, EpicID: 3,
					Status: models.FeatureStatusActive,
				},
			},
			wantKeys: []string{"Title", "Epic ID", "Status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFeaturePlanningBasicInfo(tt.info)

			// Verify it returns [][]string
			if result == nil {
				t.Fatal("buildFeaturePlanningBasicInfo() returned nil, expected [][]string")
			}

			// Verify each row has exactly 2 columns (key-value pair)
			for i, row := range result {
				if len(row) != 2 {
					t.Errorf("Row %d has %d columns, want 2", i, len(row))
				}
			}

			// Verify expected keys are present in order
			gotKeys := make([]string, len(result))
			for i, row := range result {
				gotKeys[i] = row[0]
			}

			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("Keys = %v, want %v", gotKeys, tt.wantKeys)
			}
		})
	}
}

func TestBuildFeaturePlanningBasicInfo_Values(t *testing.T) {
	desc := "Test description"
	info := &services.FeatureDisplayInfo{
		Feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "User Auth",

			Description: &desc}, EpicID: 7,
			Status: models.FeatureStatusActive,
		},
		Phase:        "development",
		ResolvedPath: "docs/plan/E07/E07-F01",
	}

	result := buildFeaturePlanningBasicInfo(info)

	// Verify specific values
	assertInfoRow(t, result, "Title", "User Auth")
	assertInfoRow(t, result, "Epic ID", "7")
	assertInfoRow(t, result, "Status", "active (workflow)")
	assertInfoRow(t, result, "Phase", "development")
	assertInfoRow(t, result, "Path", "docs/plan/E07/E07-F01")
	assertInfoRow(t, result, "Description", "Test description")
}

// Test Suite 2.4: buildFeatureBasicInfo() (aggregation mode)
func TestBuildFeatureBasicInfo(t *testing.T) {
	// Save and restore global config
	origNoColor := cli.GlobalConfig.NoColor
	defer func() { cli.GlobalConfig.NoColor = origNoColor }()

	// Force no-color for deterministic testing
	cli.GlobalConfig.NoColor = true

	desc := "Feature description"

	tests := []struct {
		name     string
		feature  *models.Feature
		data     *FeatureGetData
		wantKeys []string
	}{
		{
			name: "TC-featureBasicInfo-001: Minimal feature returns Title, Epic ID, Status, Progress",
			feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Auth Feature"}, EpicID: 7,
				Status:      models.FeatureStatusActive,
				ProgressPct: 50.0,
			},
			data: &FeatureGetData{
				ProgressInfo: &status.ProgressInfo{
					WeightedPct: 65.0,
				},
			},
			wantKeys: []string{"Title", "Epic ID", "Status", "Progress"},
		},
		{
			name: "TC-featureBasicInfo-002: Feature with all optional fields",
			feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Full Feature",

				Description: &desc}, EpicID: 7,
				Status:      models.FeatureStatusActive,
				ProgressPct: 75.0,
			},
			data: &FeatureGetData{
				DirPath:  "docs/plan/E07/E07-F01",
				Filename: "feature.md",
				ProgressInfo: &status.ProgressInfo{
					WeightedPct: 80.0,
				},
			},
			wantKeys: []string{"Title", "Epic ID", "Status", "Progress", "Path", "Filename", "Description"},
		},
		{
			name: "TC-featureBasicInfo-003: Status override shows '(manual override)' suffix",
			feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Override Feature"}, EpicID: 5,
				Status:         models.FeatureStatusCompleted,
				StatusOverride: true,
				ProgressPct:    100.0,
			},
			data: &FeatureGetData{
				ProgressInfo: &status.ProgressInfo{
					WeightedPct: 100.0,
				},
			},
			wantKeys: []string{"Title", "Epic ID", "Status", "Progress"},
		},
		{
			name: "TC-featureBasicInfo-004: Nil ProgressInfo falls back to feature.ProgressPct",
			feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Fallback Feature"}, EpicID: 2,
				Status:      models.FeatureStatusActive,
				ProgressPct: 42.0,
			},
			data: &FeatureGetData{
				ProgressInfo: nil,
			},
			wantKeys: []string{"Title", "Epic ID", "Status", "Progress"},
		},
		{
			name: "TC-featureBasicInfo-005: Nil description is omitted",
			feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "No Desc",

				Description: nil}, EpicID: 1,
				Status: models.FeatureStatusDraft,
			},
			data:     &FeatureGetData{},
			wantKeys: []string{"Title", "Epic ID", "Status", "Progress"},
		},
		{
			name: "TC-featureBasicInfo-006: Empty string description is omitted",
			feature: &models.Feature{BaseEntity: models.BaseEntity{Title: "Empty Desc",

				Description: strPtr("")}, EpicID: 1,
				Status: models.FeatureStatusDraft,
			},
			data:     &FeatureGetData{},
			wantKeys: []string{"Title", "Epic ID", "Status", "Progress"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFeatureBasicInfo(tt.feature, tt.data)

			if result == nil {
				t.Fatal("buildFeatureBasicInfo() returned nil")
			}

			// Verify each row has exactly 2 columns
			for i, row := range result {
				if len(row) != 2 {
					t.Errorf("Row %d has %d columns, want 2", i, len(row))
				}
			}

			// Verify expected keys
			gotKeys := make([]string, len(result))
			for i, row := range result {
				gotKeys[i] = row[0]
			}

			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("Keys = %v, want %v", gotKeys, tt.wantKeys)
			}
		})
	}
}

func TestBuildFeatureBasicInfo_Values(t *testing.T) {
	origNoColor := cli.GlobalConfig.NoColor
	defer func() { cli.GlobalConfig.NoColor = origNoColor }()
	cli.GlobalConfig.NoColor = true

	t.Run("calculated status shows (calculated) suffix", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Title: "Test Feature"}, EpicID: 7,
			Status:         models.FeatureStatusActive,
			StatusOverride: false,
			ProgressPct:    50.0,
		}
		data := &FeatureGetData{
			ProgressInfo: &status.ProgressInfo{WeightedPct: 65.0},
		}

		result := buildFeatureBasicInfo(feature, data)
		assertInfoRowContains(t, result, "Status", "(calculated)")
	})

	t.Run("manual override shows (manual override) suffix", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Title: "Override Feature"}, EpicID: 7,
			Status:         models.FeatureStatusCompleted,
			StatusOverride: true,
			ProgressPct:    100.0,
		}
		data := &FeatureGetData{
			ProgressInfo: &status.ProgressInfo{WeightedPct: 100.0},
		}

		result := buildFeatureBasicInfo(feature, data)
		assertInfoRowContains(t, result, "Status", "(manual override)")
	})

	t.Run("weighted progress is used when ProgressInfo available", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Title: "Progress Feature"}, EpicID: 7,
			Status:      models.FeatureStatusActive,
			ProgressPct: 50.0,
		}
		data := &FeatureGetData{
			ProgressInfo: &status.ProgressInfo{WeightedPct: 65.0},
		}

		result := buildFeatureBasicInfo(feature, data)
		assertInfoRow(t, result, "Progress", "65%")
	})

	t.Run("feature ProgressPct used as fallback when ProgressInfo nil", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Title: "Fallback Feature"}, EpicID: 7,
			Status:      models.FeatureStatusActive,
			ProgressPct: 42.0,
		}
		data := &FeatureGetData{
			ProgressInfo: nil,
		}

		result := buildFeatureBasicInfo(feature, data)
		assertInfoRow(t, result, "Progress", "42%")
	})
}

// Test Suite: Render callback functions (no-panic tests)
func TestRenderFeatureStatusBreakdown(t *testing.T) {
	tests := []struct {
		name            string
		statusBreakdown []workflow.StatusCount
		colorEnabled    bool
	}{
		{
			name:            "TC-statusBreakdown-001: Nil breakdown does nothing",
			statusBreakdown: nil,
			colorEnabled:    false,
		},
		{
			name:            "TC-statusBreakdown-002: Empty breakdown does nothing",
			statusBreakdown: []workflow.StatusCount{},
			colorEnabled:    false,
		},
		{
			name: "TC-statusBreakdown-003: Populated breakdown renders without panic",
			statusBreakdown: []workflow.StatusCount{
				{Status: "todo", Count: 3, Phase: "planning"},
				{Status: "in_progress", Count: 2, Phase: "development"},
				{Status: "completed", Count: 5, Phase: "done"},
			},
			colorEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderFeatureStatusBreakdown() panicked: %v", r)
				}
			}()
			renderFeatureStatusBreakdown(tt.statusBreakdown, nil, tt.colorEnabled)
		})
	}
}

func TestRenderFeatureProgressBreakdown(t *testing.T) {
	tests := []struct {
		name         string
		progressInfo *status.ProgressInfo
	}{
		{
			name:         "TC-progressBreakdown-001: Nil progress does nothing",
			progressInfo: nil,
		},
		{
			name: "TC-progressBreakdown-002: Populated progress renders without panic",
			progressInfo: &status.ProgressInfo{
				WeightedPct:     65.0,
				CompletionPct:   40.0,
				WeightedRatio:   "3.2/5",
				CompletionRatio: "2/5",
				TotalTasks:      5,
			},
		},
		{
			name: "TC-progressBreakdown-003: Zero values render without panic",
			progressInfo: &status.ProgressInfo{
				WeightedPct:     0.0,
				CompletionPct:   0.0,
				WeightedRatio:   "0/0",
				CompletionRatio: "0/0",
				TotalTasks:      0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderFeatureProgressBreakdown() panicked: %v", r)
				}
			}()
			renderFeatureProgressBreakdown(tt.progressInfo)
		})
	}
}

func TestRenderFeatureWorkSummary(t *testing.T) {
	tests := []struct {
		name        string
		workSummary *status.WorkSummary
	}{
		{
			name:        "TC-workSummary-001: Nil work summary does nothing",
			workSummary: nil,
		},
		{
			name:        "TC-workSummary-002: Zero total tasks does nothing",
			workSummary: &status.WorkSummary{TotalTasks: 0},
		},
		{
			name: "TC-workSummary-003: Populated work summary renders without panic",
			workSummary: &status.WorkSummary{
				TotalTasks:     10,
				CompletedTasks: 3,
				AgentWork:      4,
				HumanWork:      2,
				BlockedWork:    1,
				NotStarted:     0,
			},
		},
		{
			name: "TC-workSummary-004: Only completed tasks renders without panic",
			workSummary: &status.WorkSummary{
				TotalTasks:     5,
				CompletedTasks: 5,
			},
		},
		{
			name: "TC-workSummary-005: Only not-started tasks renders without panic",
			workSummary: &status.WorkSummary{
				TotalTasks: 3,
				NotStarted: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderFeatureWorkSummary() panicked: %v", r)
				}
			}()
			renderFeatureWorkSummary(tt.workSummary)
		})
	}
}

func TestRenderFeatureActionItems(t *testing.T) {
	ageDays := 3
	reason := "Waiting on API"

	tests := []struct {
		name        string
		actionItems *status.ActionItems
	}{
		{
			name:        "TC-actionItems-001: Nil action items does nothing",
			actionItems: nil,
		},
		{
			name: "TC-actionItems-002: Empty action items does nothing",
			actionItems: &status.ActionItems{
				AwaitingApproval: []*status.TaskActionItem{},
				Blocked:          []*status.TaskActionItem{},
				InProgress:       []*status.TaskActionItem{},
			},
		},
		{
			name: "TC-actionItems-003: Awaiting approval renders without panic",
			actionItems: &status.ActionItems{
				AwaitingApproval: []*status.TaskActionItem{
					{TaskKey: "E07-F01-001", Title: "Review task", AgeDays: &ageDays},
				},
				Blocked:    []*status.TaskActionItem{},
				InProgress: []*status.TaskActionItem{},
			},
		},
		{
			name: "TC-actionItems-004: Blocked items with reason renders without panic",
			actionItems: &status.ActionItems{
				AwaitingApproval: []*status.TaskActionItem{},
				Blocked: []*status.TaskActionItem{
					{TaskKey: "E07-F01-002", Title: "Blocked task", BlockedReason: &reason},
				},
				InProgress: []*status.TaskActionItem{},
			},
		},
		{
			name: "TC-actionItems-005: In-progress items renders without panic",
			actionItems: &status.ActionItems{
				AwaitingApproval: []*status.TaskActionItem{},
				Blocked:          []*status.TaskActionItem{},
				InProgress: []*status.TaskActionItem{
					{TaskKey: "E07-F01-003", Title: "Active task"},
					{TaskKey: "E07-F01-004", Title: "Another active task"},
				},
			},
		},
		{
			name: "TC-actionItems-006: Nil AgeDays and nil BlockedReason handled",
			actionItems: &status.ActionItems{
				AwaitingApproval: []*status.TaskActionItem{
					{TaskKey: "E07-F01-001", Title: "No age", AgeDays: nil},
				},
				Blocked: []*status.TaskActionItem{
					{TaskKey: "E07-F01-002", Title: "No reason", BlockedReason: nil},
				},
				InProgress: []*status.TaskActionItem{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderFeatureActionItems() panicked: %v", r)
				}
			}()
			renderFeatureActionItems(tt.actionItems)
		})
	}
}

func TestRenderFeatureWorkflowPosition(t *testing.T) {
	tests := []struct {
		name string
		wp   *services.WorkflowPosition
	}{
		{
			name: "TC-workflowPos-001: Nil position does nothing",
			wp:   nil,
		},
		{
			name: "TC-workflowPos-002: Populated position renders without panic",
			wp: &services.WorkflowPosition{
				Statuses:     []string{"draft", "active", "completed"},
				CurrentIndex: 1,
			},
		},
		{
			name: "TC-workflowPos-003: First position renders without panic",
			wp: &services.WorkflowPosition{
				Statuses:     []string{"draft", "active"},
				CurrentIndex: 0,
			},
		},
		{
			name: "TC-workflowPos-004: Last position renders without panic",
			wp: &services.WorkflowPosition{
				Statuses:     []string{"draft", "active", "completed"},
				CurrentIndex: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderFeatureWorkflowPosition() panicked: %v", r)
				}
			}()
			renderFeatureWorkflowPosition(tt.wp)
		})
	}
}

func TestRenderFeatureTasksSection(t *testing.T) {
	tests := []struct {
		name  string
		tasks []*models.Task
	}{
		{
			name:  "TC-tasksSection-001: Nil tasks renders info message",
			tasks: nil,
		},
		{
			name:  "TC-tasksSection-002: Empty tasks renders info message",
			tasks: []*models.Task{},
		},
		// Note: Non-empty tasks require workflow service (cli.GetWorkflowService()),
		// so we only test empty/nil cases here. Full integration tests cover the
		// non-empty case.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderFeatureTasksSection() panicked: %v", r)
				}
			}()
			renderFeatureTasksSection(tt.tasks)
		})
	}
}

// Test calculateHealthIndicator (pure function)
func TestCalculateHealthIndicator(t *testing.T) {
	tests := []struct {
		name         string
		statusCounts map[string]int
		cfg          *config.WorkflowConfig
		want         string
	}{
		{
			name:         "TC-health-001: Nil config with no blocked returns healthy",
			statusCounts: map[string]int{"todo": 3, "in_progress": 2},
			cfg:          nil,
			want:         "\U0001F7E2",
		},
		{
			name:         "TC-health-002: Nil config with 1 blocked returns attention",
			statusCounts: map[string]int{"todo": 3, "blocked": 1},
			cfg:          nil,
			want:         "\U0001F7E1",
		},
		{
			name:         "TC-health-003: Nil config with 3+ blocked returns at risk",
			statusCounts: map[string]int{"todo": 3, "blocked": 3},
			cfg:          nil,
			want:         "\U0001F534",
		},
		{
			name:         "TC-health-004: Config-driven with blocking statuses",
			statusCounts: map[string]int{"todo": 3, "ready_for_approval": 1},
			cfg:          featureTestWorkflowConfig(),
			want:         "\U0001F7E1",
		},
		{
			name:         "TC-health-005: Config-driven with no blocking statuses returns healthy",
			statusCounts: map[string]int{"todo": 3, "in_progress": 2},
			cfg:          featureTestWorkflowConfig(),
			want:         "\U0001F7E2",
		},
		{
			name:         "TC-health-006: Empty status counts returns healthy",
			statusCounts: map[string]int{},
			cfg:          nil,
			want:         "\U0001F7E2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateHealthIndicator(tt.statusCounts, tt.cfg)
			if got != tt.want {
				t.Errorf("calculateHealthIndicator() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test generateNotesColumn (pure function)
func TestGenerateNotesColumn(t *testing.T) {
	tests := []struct {
		name         string
		statusCounts map[string]int
		cfg          *config.WorkflowConfig
		wantContains string
	}{
		{
			name:         "TC-notes-001: No blocking statuses returns on track",
			statusCounts: map[string]int{"todo": 3},
			cfg:          nil,
			wantContains: "[on track]",
		},
		{
			name:         "TC-notes-002: Blocked tasks shown in notes",
			statusCounts: map[string]int{"blocked": 2},
			cfg:          nil,
			wantContains: "2 blocked",
		},
		{
			name:         "TC-notes-003: Empty counts returns on track",
			statusCounts: map[string]int{},
			cfg:          nil,
			wantContains: "[on track]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNotesColumn(tt.statusCounts, tt.cfg)
			if got == "" {
				t.Error("generateNotesColumn() returned empty string")
			}
			if tt.wantContains != "" && !strings.Contains(got, tt.wantContains) {
				t.Errorf("generateNotesColumn() = %q, want to contain %q", got, tt.wantContains)
			}
		})
	}
}

// Test filterFeaturesByCompletedStatus (pure function)
func TestFilterFeaturesByCompletedStatus(t *testing.T) {
	features := []FeatureWithTaskCount{
		{Feature: &models.Feature{Status: models.FeatureStatusActive}},
		{Feature: &models.Feature{Status: models.FeatureStatusCompleted}},
		{Feature: &models.Feature{Status: models.FeatureStatusDraft}},
	}

	t.Run("default hides completed", func(t *testing.T) {
		result := filterFeaturesByCompletedStatus(features, false, "")
		if len(result) != 2 {
			t.Errorf("Expected 2 features, got %d", len(result))
		}
		for _, f := range result {
			if f.Status == models.FeatureStatusCompleted {
				t.Error("Completed feature should be filtered out")
			}
		}
	})

	t.Run("showAll returns all", func(t *testing.T) {
		result := filterFeaturesByCompletedStatus(features, true, "")
		if len(result) != 3 {
			t.Errorf("Expected 3 features, got %d", len(result))
		}
	})

	t.Run("explicit status filter returns all", func(t *testing.T) {
		result := filterFeaturesByCompletedStatus(features, false, "completed")
		if len(result) != 3 {
			t.Errorf("Expected 3 features with explicit filter, got %d", len(result))
		}
	})

	t.Run("empty list returns empty", func(t *testing.T) {
		result := filterFeaturesByCompletedStatus(nil, false, "")
		if len(result) != 0 {
			t.Errorf("Expected 0 features, got %d", len(result))
		}
	})
}

// Test sortFeatures (pure function)
func TestSortFeatures(t *testing.T) {
	t.Run("sort by key", func(t *testing.T) {
		features := []FeatureWithTaskCount{
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F03"}}},
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01"}}},
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F02"}}},
		}
		sortFeatures(features, "key", nil, nil)
		if features[0].Key != "E07-F01" || features[1].Key != "E07-F02" || features[2].Key != "E07-F03" {
			t.Errorf("Sort by key failed: got %s, %s, %s", features[0].Key, features[1].Key, features[2].Key)
		}
	})

	t.Run("sort by progress", func(t *testing.T) {
		features := []FeatureWithTaskCount{
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01"}, ProgressPct: 80.0}},
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F02"}, ProgressPct: 20.0}},
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F03"}, ProgressPct: 50.0}},
		}
		sortFeatures(features, "progress", nil, nil)
		if features[0].ProgressPct != 20.0 || features[1].ProgressPct != 50.0 || features[2].ProgressPct != 80.0 {
			t.Errorf("Sort by progress failed: got %.0f, %.0f, %.0f",
				features[0].ProgressPct, features[1].ProgressPct, features[2].ProgressPct)
		}
	})

	t.Run("sort by status", func(t *testing.T) {
		features := []FeatureWithTaskCount{
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01"}, Status: models.FeatureStatusCompleted}},
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F02"}, Status: models.FeatureStatusDraft}},
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F03"}, Status: models.FeatureStatusActive}},
		}
		sortFeatures(features, "status", nil, nil)
		if features[0].Status != models.FeatureStatusDraft ||
			features[1].Status != models.FeatureStatusActive ||
			features[2].Status != models.FeatureStatusCompleted {
			t.Errorf("Sort by status failed: got %s, %s, %s",
				features[0].Status, features[1].Status, features[2].Status)
		}
	})

	t.Run("default sort is by key", func(t *testing.T) {
		features := []FeatureWithTaskCount{
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F03"}}},
			{Feature: &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01"}}},
		}
		sortFeatures(features, "", nil, nil)
		if features[0].Key != "E07-F01" {
			t.Errorf("Default sort failed: first key is %s, want E07-F01", features[0].Key)
		}
	})
}

// TestSortFeatures_ProgressUsesLiveComputedValue covers the remaining half of
// B047 AC5: "Feature list display and sort order use the same progress
// value." renderFeatureListTable and outputFeatureListJSON both render
// live-computed weighted progress (status.CalculateProgress over the
// status-breakdown batch), not the persisted ProgressPct cache. Before the
// fix, sortFeatures sorted by the stale cache, so a feature whose displayed
// percentage (computed) was higher than another's could still sort below it
// (cache says otherwise) — the exact "sort order disagrees with the
// displayed value" symptom the AC calls out.
//
// F01 has a stale low cache (10%) but a fully-completed live breakdown
// (100% computed). F02 has a stale high cache (90%) but an all-todo live
// breakdown (0% computed). Sorting by the cache would put F01 before F02;
// sorting by the live-computed value (matching the table/JSON output) must
// put F02 before F01.
func TestSortFeatures_ProgressUsesLiveComputedValue(t *testing.T) {
	cfg := &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"todo":      {ProgressWeight: 0.0},
			"completed": {ProgressWeight: 1.0},
		},
	}

	features := []FeatureWithTaskCount{
		{Feature: &models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: "E07-F01"}, ProgressPct: 10.0}},
		{Feature: &models.Feature{BaseEntity: models.BaseEntity{ID: 2, Key: "E07-F02"}, ProgressPct: 90.0}},
	}
	statusBreakdownBatch := map[int64]map[models.TaskStatus]int{
		1: {models.TaskStatus("completed"): 1}, // live-computed: 100%
		2: {models.TaskStatus("todo"): 1},      // live-computed: 0%
	}

	sortFeatures(features, "progress", statusBreakdownBatch, cfg)

	if features[0].Key != "E07-F02" || features[1].Key != "E07-F01" {
		t.Errorf("sortFeatures did not use live-computed progress: got order %s, %s; want E07-F02, E07-F01",
			features[0].Key, features[1].Key)
	}
}

// TestRenderFeatureAggregation_ReadinessMessage covers B047 AC5: "feature get
// readiness messaging uses computed progress rather than a stale cached
// field." feature.ProgressPct is the persisted cache; data.ProgressInfo is
// computed live from the current task status breakdown. The two normally
// agree once the aggregate coordinator keeps the cache current, but the
// readiness banner must key off the same computed source the rest of the
// page uses (see buildFeatureBasicInfo's "Progress" row), not the raw
// cached column, so a stale cache can never show a readiness banner that
// disagrees with the progress the operator is looking at.
func TestRenderFeatureAggregation_ReadinessMessage(t *testing.T) {
	origNoColor := cli.GlobalConfig.NoColor
	defer func() { cli.GlobalConfig.NoColor = origNoColor }()
	cli.GlobalConfig.NoColor = true

	// pterm.Success caches its output writer at package-init time (a copy of
	// os.Stdout taken once), so it does not observe captureOutput's temporary
	// os.Stdout swap. Redirect it explicitly for this test and restore after.
	origSuccessWriter := pterm.Success
	defer func() { pterm.Success = origSuccessWriter }()

	task := &models.Task{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: "Only task"}}

	t.Run("computed progress at 100 shows readiness banner even when cached field lags", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Title: "Feature"}, EpicID: 7,
			Status:      models.FeatureStatusActive,
			ProgressPct: 90.0, // stale cache: not yet refreshed
		}
		data := &FeatureGetData{
			Tasks:        []*models.Task{task},
			ProgressInfo: &status.ProgressInfo{WeightedPct: 100.0}, // computed: current
		}

		out := captureOutput(t, func() {
			pterm.Success = *origSuccessWriter.WithWriter(os.Stdout)
			renderFeatureAggregationWithTags(feature, data, nil, nil)
		})

		if !strings.Contains(string(out), "ready for approval") {
			t.Errorf("expected readiness banner when computed progress is 100%%, got output:\n%s", out)
		}
	})

	t.Run("computed progress below 100 hides readiness banner even when cached field is stale-high", func(t *testing.T) {
		feature := &models.Feature{BaseEntity: models.BaseEntity{Title: "Feature"}, EpicID: 7,
			Status:      models.FeatureStatusActive,
			ProgressPct: 100.0, // stale cache: overstates progress
		}
		data := &FeatureGetData{
			Tasks:        []*models.Task{task},
			ProgressInfo: &status.ProgressInfo{WeightedPct: 80.0}, // computed: current
		}

		out := captureOutput(t, func() {
			pterm.Success = *origSuccessWriter.WithWriter(os.Stdout)
			renderFeatureAggregationWithTags(feature, data, nil, nil)
		})

		if strings.Contains(string(out), "ready for approval") {
			t.Errorf("did not expect readiness banner when computed progress is below 100%%, got output:\n%s", out)
		}
	})
}

// Helper functions for test assertions

// assertInfoRow asserts that an info table has a row with the given key and exact value.
func assertInfoRow(t *testing.T, info [][]string, key, wantValue string) {
	t.Helper()
	for _, row := range info {
		if row[0] == key {
			if row[1] != wantValue {
				t.Errorf("Row %q = %q, want %q", key, row[1], wantValue)
			}
			return
		}
	}
	t.Errorf("Key %q not found in info table", key)
}

// assertInfoRowContains asserts that an info table has a row whose value contains the substring.
func assertInfoRowContains(t *testing.T, info [][]string, key, wantSubstr string) {
	t.Helper()
	for _, row := range info {
		if row[0] == key {
			if !strings.Contains(row[1], wantSubstr) {
				t.Errorf("Row %q = %q, want to contain %q", key, row[1], wantSubstr)
			}
			return
		}
	}
	t.Errorf("Key %q not found in info table", key)
}

// featureTestWorkflowConfig returns a minimal workflow config for testing feature helpers.
func featureTestWorkflowConfig() *config.WorkflowConfig {
	return &config.WorkflowConfig{
		StatusMetadata: map[string]config.StatusMetadata{
			"todo":               {BlocksFeature: false},
			"in_progress":        {BlocksFeature: false},
			"blocked":            {BlocksFeature: true},
			"ready_for_approval": {BlocksFeature: true},
			"completed":          {BlocksFeature: false},
		},
		StatusFlow: map[string][]string{
			"todo":               {"in_progress"},
			"in_progress":        {"ready_for_approval", "blocked"},
			"ready_for_approval": {"completed"},
			"blocked":            {"in_progress"},
		},
	}
}
