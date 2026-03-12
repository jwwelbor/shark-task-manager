package commands

import (
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// Test Suite 2.4: buildEpicPlanningBasicInfo() - pure function tests

func TestBuildEpicPlanningBasicInfo(t *testing.T) {
	tests := []struct {
		name string
		info *services.EpicDisplayInfo
		want [][]string
	}{
		{
			name: "TC-buildEpicPlanningBasicInfo-001: All fields populated",
			info: func() *services.EpicDisplayInfo {
				desc := "Epic description"
				bv := models.PriorityHigh
				return &services.EpicDisplayInfo{
					Epic: &models.Epic{
						Title:         "User Authentication",
						Status:        models.EpicStatusActive,
						Priority:      models.PriorityHigh,
						Description:   &desc,
						BusinessValue: &bv,
					},
					Phase:            "development",
					PhaseDescription: "Implementation in progress",
					ResolvedPath:     "docs/plan/E07-user-auth/epic.md",
				}
			}(),
			want: [][]string{
				{"Title", "User Authentication"},
				{"Status", "active (workflow)"},
				{"Phase", "development"},
				{"Phase Description", "Implementation in progress"},
				{"Priority", "high"},
				{"Path", "docs/plan/E07-user-auth/epic.md"},
				{"Description", "Epic description"},
				{"Business Value", "high"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-002: Minimal fields (only required)",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:  "Minimal Epic",
					Status: models.EpicStatusDraft,
				},
			},
			want: [][]string{
				{"Title", "Minimal Epic"},
				{"Status", "draft (workflow)"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-003: Nil description omitted",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:       "No Description",
					Status:      models.EpicStatusActive,
					Priority:    models.PriorityMedium,
					Description: nil,
				},
			},
			want: [][]string{
				{"Title", "No Description"},
				{"Status", "active (workflow)"},
				{"Priority", "medium"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-004: Empty description omitted",
			info: func() *services.EpicDisplayInfo {
				empty := ""
				return &services.EpicDisplayInfo{
					Epic: &models.Epic{
						Title:       "Empty Description",
						Status:      models.EpicStatusActive,
						Description: &empty,
					},
				}
			}(),
			want: [][]string{
				{"Title", "Empty Description"},
				{"Status", "active (workflow)"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-005: Empty priority omitted",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:    "No Priority",
					Status:   models.EpicStatusDraft,
					Priority: "",
				},
			},
			want: [][]string{
				{"Title", "No Priority"},
				{"Status", "draft (workflow)"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-006: Empty path omitted",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:  "No Path",
					Status: models.EpicStatusDraft,
				},
				ResolvedPath: "",
			},
			want: [][]string{
				{"Title", "No Path"},
				{"Status", "draft (workflow)"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-007: Nil business value omitted",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:         "No BV",
					Status:        models.EpicStatusActive,
					BusinessValue: nil,
				},
			},
			want: [][]string{
				{"Title", "No BV"},
				{"Status", "active (workflow)"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-008: Phase present without phase description",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:  "Phase Only",
					Status: models.EpicStatusActive,
				},
				Phase: "planning",
			},
			want: [][]string{
				{"Title", "Phase Only"},
				{"Status", "active (workflow)"},
				{"Phase", "planning"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEpicPlanningBasicInfo(tt.info)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildEpicPlanningBasicInfo() =\n  %v\nwant:\n  %v", got, tt.want)
			}
		})
	}
}

// Test Suite 2.4: buildEpicAggregationBasicInfo() - pure function tests

func TestBuildEpicAggregationBasicInfo(t *testing.T) {
	tests := []struct {
		name     string
		epic     *models.Epic
		progress float64
		path     string
		filename string
		want     [][]string
	}{
		{
			name: "TC-buildEpicAggregationBasicInfo-001: All fields populated",
			epic: func() *models.Epic {
				desc := "Full epic"
				bv := models.PriorityHigh
				return &models.Epic{
					Title:         "Full Epic",
					Status:        models.EpicStatusActive,
					Priority:      models.PriorityHigh,
					Description:   &desc,
					BusinessValue: &bv,
				}
			}(),
			progress: 75.0,
			path:     "docs/plan/E07-full/",
			filename: "epic.md",
			want: [][]string{
				{"Title", "Full Epic"},
				{"Status", "active (calculated)"},
				{"Priority", "high"},
				{"Progress", "75%"},
				{"Path", "docs/plan/E07-full/"},
				{"Filename", "epic.md"},
				{"Description", "Full epic"},
				{"Business Value", "high"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-002: Minimal fields",
			epic: &models.Epic{
				Title:    "Minimal",
				Status:   models.EpicStatusDraft,
				Priority: models.PriorityLow,
			},
			progress: 0.0,
			path:     "",
			filename: "",
			want: [][]string{
				{"Title", "Minimal"},
				{"Status", "draft (calculated)"},
				{"Priority", "low"},
				{"Progress", "0%"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-003: 100% progress",
			epic: &models.Epic{
				Title:    "Complete",
				Status:   models.EpicStatusCompleted,
				Priority: models.PriorityMedium,
			},
			progress: 100.0,
			path:     "",
			filename: "",
			want: [][]string{
				{"Title", "Complete"},
				{"Status", "completed (calculated)"},
				{"Priority", "medium"},
				{"Progress", "100%"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-004: Nil description omitted",
			epic: &models.Epic{
				Title:       "No Desc",
				Status:      models.EpicStatusActive,
				Priority:    models.PriorityMedium,
				Description: nil,
			},
			progress: 50.0,
			path:     "",
			filename: "",
			want: [][]string{
				{"Title", "No Desc"},
				{"Status", "active (calculated)"},
				{"Priority", "medium"},
				{"Progress", "50%"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-005: Empty description omitted",
			epic: func() *models.Epic {
				empty := ""
				return &models.Epic{
					Title:       "Empty Desc",
					Status:      models.EpicStatusActive,
					Priority:    models.PriorityMedium,
					Description: &empty,
				}
			}(),
			progress: 25.0,
			path:     "",
			filename: "",
			want: [][]string{
				{"Title", "Empty Desc"},
				{"Status", "active (calculated)"},
				{"Priority", "medium"},
				{"Progress", "25%"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-006: Path without filename",
			epic: &models.Epic{
				Title:    "Path Only",
				Status:   models.EpicStatusActive,
				Priority: models.PriorityMedium,
			},
			progress: 10.0,
			path:     "docs/plan/E07/",
			filename: "",
			want: [][]string{
				{"Title", "Path Only"},
				{"Status", "active (calculated)"},
				{"Priority", "medium"},
				{"Progress", "10%"},
				{"Path", "docs/plan/E07/"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-007: Filename without path",
			epic: &models.Epic{
				Title:    "Filename Only",
				Status:   models.EpicStatusActive,
				Priority: models.PriorityMedium,
			},
			progress: 10.0,
			path:     "",
			filename: "epic.md",
			want: [][]string{
				{"Title", "Filename Only"},
				{"Status", "active (calculated)"},
				{"Priority", "medium"},
				{"Progress", "10%"},
				{"Filename", "epic.md"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-008: Nil business value omitted",
			epic: &models.Epic{
				Title:         "No BV",
				Status:        models.EpicStatusActive,
				Priority:      models.PriorityMedium,
				BusinessValue: nil,
			},
			progress: 50.0,
			path:     "",
			filename: "",
			want: [][]string{
				{"Title", "No BV"},
				{"Status", "active (calculated)"},
				{"Priority", "medium"},
				{"Progress", "50%"},
			},
		},
		{
			name: "TC-buildEpicAggregationBasicInfo-009: Fractional progress rounds correctly",
			epic: &models.Epic{
				Title:    "Fractional",
				Status:   models.EpicStatusActive,
				Priority: models.PriorityMedium,
			},
			progress: 33.333,
			path:     "",
			filename: "",
			want: [][]string{
				{"Title", "Fractional"},
				{"Status", "active (calculated)"},
				{"Priority", "medium"},
				{"Progress", "33%"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildEpicAggregationBasicInfo(tt.epic, tt.progress, tt.path, tt.filename)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildEpicAggregationBasicInfo() =\n  %v\nwant:\n  %v", got, tt.want)
			}
		})
	}
}

// Test Suite 2.4: renderEpicPlanningSpecific() - no-panic tests
// These test that the rendering callbacks don't panic with various inputs.

func TestRenderEpicPlanningSpecific_NoPanic(t *testing.T) {
	tests := []struct {
		name string
		info *services.EpicDisplayInfo
	}{
		{
			name: "TC-renderEpicPlanningSpecific-001: With workflow position and features",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:  "Test Epic",
					Status: models.EpicStatusActive,
				},
				WorkflowPosition: &services.WorkflowPosition{
					Statuses:     []string{"draft", "active", "completed"},
					CurrentIndex: 1,
				},
				Features: []services.FeatureDisplayItem{
					{
						Feature:   &models.Feature{Key: "E07-F01", Title: "Feature 1", Status: "active"},
						TaskCount: 5,
						Phase:     "development",
					},
				},
			},
		},
		{
			name: "TC-renderEpicPlanningSpecific-002: No workflow position",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:  "No Position",
					Status: models.EpicStatusDraft,
				},
				WorkflowPosition: nil,
				Features:         []services.FeatureDisplayItem{},
			},
		},
		{
			name: "TC-renderEpicPlanningSpecific-003: No features",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:  "No Features",
					Status: models.EpicStatusDraft,
				},
				Features: nil,
			},
		},
		{
			name: "TC-renderEpicPlanningSpecific-004: Feature with empty phase",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{
					Title:  "Empty Phase Feature",
					Status: models.EpicStatusActive,
				},
				Features: []services.FeatureDisplayItem{
					{
						Feature:   &models.Feature{Key: "E07-F01", Title: "Feature 1", Status: "draft"},
						TaskCount: 0,
						Phase:     "",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderEpicPlanningSpecific() panicked: %v", r)
				}
			}()
			renderEpicPlanningSpecific(tt.info)
		})
	}
}

// Test Suite 2.4: renderEpicAggregationSpecific() - no-panic tests

func TestRenderEpicAggregationSpecific_NoPanic(t *testing.T) {
	tests := []struct {
		name                 string
		featureRollup        map[string]int
		taskRollup           map[string]int
		blockedTasks         []*models.Task
		approvalBacklogCount int
		features             []FeatureWithDetails
	}{
		{
			name:          "TC-renderEpicAggregationSpecific-001: All sections populated",
			featureRollup: map[string]int{"active": 3, "completed": 2},
			taskRollup:    map[string]int{"todo": 5, "in_progress": 3, "completed": 10},
			blockedTasks: []*models.Task{
				{Key: "E07-F01-001", Title: "Blocked Task"},
			},
			approvalBacklogCount: 2,
			features: []FeatureWithDetails{
				{
					Feature:   &models.Feature{Key: "E07-F01", Title: "Feature 1", Status: "active"},
					TaskCount: 5,
				},
			},
		},
		{
			name:                 "TC-renderEpicAggregationSpecific-002: Empty rollups",
			featureRollup:        map[string]int{},
			taskRollup:           map[string]int{},
			blockedTasks:         nil,
			approvalBacklogCount: 0,
			features:             nil,
		},
		{
			name:          "TC-renderEpicAggregationSpecific-003: Nil rollups",
			featureRollup: nil,
			taskRollup:    nil,
			blockedTasks:  nil,
			features:      nil,
		},
		{
			name:          "TC-renderEpicAggregationSpecific-004: Blocked task with reason and age",
			featureRollup: nil,
			taskRollup:    nil,
			blockedTasks: func() []*models.Task {
				reason := "Waiting on API"
				return []*models.Task{
					{Key: "E07-F01-001", Title: "Blocked", BlockedReason: &reason},
				}
			}(),
			approvalBacklogCount: 0,
			features:             nil,
		},
		{
			name:                 "TC-renderEpicAggregationSpecific-005: Only approval backlog",
			featureRollup:        nil,
			taskRollup:           nil,
			blockedTasks:         nil,
			approvalBacklogCount: 5,
			features:             nil,
		},
		{
			name:          "TC-renderEpicAggregationSpecific-006: Features with planning mode",
			featureRollup: nil,
			taskRollup:    nil,
			blockedTasks:  nil,
			features: []FeatureWithDetails{
				{
					Feature:    &models.Feature{Key: "E07-F01", Title: "Planning Feature", Status: "draft"},
					TaskCount:  0,
					IsPlanning: true,
					Phase:      "planning",
				},
				{
					Feature:    &models.Feature{Key: "E07-F02", Title: "Active Feature with a very long title that should be truncated by the renderer", Status: "active"},
					TaskCount:  10,
					IsPlanning: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderEpicAggregationSpecific() panicked: %v", r)
				}
			}()
			renderEpicAggregationSpecific(tt.featureRollup, tt.taskRollup, tt.blockedTasks, tt.approvalBacklogCount, tt.features)
		})
	}
}

// Test sortEpics helper
func TestSortEpics(t *testing.T) {
	tests := []struct {
		name     string
		epics    []EpicWithProgress
		sortBy   string
		wantKeys []string
	}{
		{
			name: "Sort by key (default)",
			epics: []EpicWithProgress{
				{Epic: &models.Epic{Key: "E03"}, ProgressPct: 50},
				{Epic: &models.Epic{Key: "E01"}, ProgressPct: 75},
				{Epic: &models.Epic{Key: "E02"}, ProgressPct: 25},
			},
			sortBy:   "key",
			wantKeys: []string{"E01", "E02", "E03"},
		},
		{
			name: "Sort by progress",
			epics: []EpicWithProgress{
				{Epic: &models.Epic{Key: "E01"}, ProgressPct: 75},
				{Epic: &models.Epic{Key: "E02"}, ProgressPct: 25},
				{Epic: &models.Epic{Key: "E03"}, ProgressPct: 50},
			},
			sortBy:   "progress",
			wantKeys: []string{"E02", "E03", "E01"},
		},
		{
			name: "Sort by status",
			epics: []EpicWithProgress{
				{Epic: &models.Epic{Key: "E01", Status: models.EpicStatusCompleted}},
				{Epic: &models.Epic{Key: "E02", Status: models.EpicStatusDraft}},
				{Epic: &models.Epic{Key: "E03", Status: models.EpicStatusActive}},
			},
			sortBy:   "status",
			wantKeys: []string{"E02", "E03", "E01"},
		},
		{
			name: "Default sort (empty string) sorts by key",
			epics: []EpicWithProgress{
				{Epic: &models.Epic{Key: "E03"}},
				{Epic: &models.Epic{Key: "E01"}},
			},
			sortBy:   "",
			wantKeys: []string{"E01", "E03"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortEpics(tt.epics, tt.sortBy)
			gotKeys := make([]string, len(tt.epics))
			for i, e := range tt.epics {
				gotKeys[i] = e.Key
			}
			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("sortEpics() got keys = %v, want %v", gotKeys, tt.wantKeys)
			}
		})
	}
}

// Test getRelativePath helper
func TestGetRelativePath(t *testing.T) {
	tests := []struct {
		name        string
		absPath     string
		projectRoot string
		want        string
	}{
		{
			name:        "Valid relative path",
			absPath:     "/home/user/project/docs/epic.md",
			projectRoot: "/home/user/project",
			want:        "docs/epic.md",
		},
		{
			name:        "Same directory",
			absPath:     "/home/user/project/file.md",
			projectRoot: "/home/user/project",
			want:        "file.md",
		},
		{
			name:        "Path outside project root returns absolute",
			absPath:     "/other/path/file.md",
			projectRoot: "/home/user/project",
			want:        "../../../other/path/file.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRelativePath(tt.absPath, tt.projectRoot)
			if got != tt.want {
				t.Errorf("getRelativePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
