package commands

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
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
					Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "User Authentication",

						Description: &desc}, Status: models.EpicStatusActive,
						Priority: models.PriorityHigh,

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
				Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Minimal Epic"}, Status: models.EpicStatusDraft},
			},
			want: [][]string{
				{"Title", "Minimal Epic"},
				{"Status", "draft (workflow)"},
			},
		},
		{
			name: "TC-buildEpicPlanningBasicInfo-003: Nil description omitted",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "No Description",

					Description: nil}, Status: models.EpicStatusActive,
					Priority: models.PriorityMedium,
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
					Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Empty Description",

						Description: &empty}, Status: models.EpicStatusActive,
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
				Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "No Priority"}, Status: models.EpicStatusDraft,
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
				Epic:         &models.Epic{BaseEntity: models.BaseEntity{Title: "No Path"}, Status: models.EpicStatusDraft},
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
				Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "No BV"}, Status: models.EpicStatusActive,
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
				Epic:  &models.Epic{BaseEntity: models.BaseEntity{Title: "Phase Only"}, Status: models.EpicStatusActive},
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
				return &models.Epic{BaseEntity: models.BaseEntity{Title: "Full Epic",

					Description: &desc}, Status: models.EpicStatusActive,
					Priority: models.PriorityHigh,

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
			epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Minimal"}, Status: models.EpicStatusDraft,
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
			epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Complete"}, Status: models.EpicStatusCompleted,
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
			epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "No Desc",

				Description: nil}, Status: models.EpicStatusActive,
				Priority: models.PriorityMedium,
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
				return &models.Epic{BaseEntity: models.BaseEntity{Title: "Empty Desc",

					Description: &empty}, Status: models.EpicStatusActive,
					Priority: models.PriorityMedium,
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
			epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Path Only"}, Status: models.EpicStatusActive,
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
			epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Filename Only"}, Status: models.EpicStatusActive,
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
			epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "No BV"}, Status: models.EpicStatusActive,
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
			epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Fractional"}, Status: models.EpicStatusActive,
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
				Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Test Epic"}, Status: models.EpicStatusActive},
				WorkflowPosition: &services.WorkflowPosition{
					Statuses:     []string{"draft", "active", "completed"},
					CurrentIndex: 1,
				},
				Features: []services.FeatureDisplayItem{
					{
						Feature:   &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01", Title: "Feature 1"}, Status: "active"},
						TaskCount: 5,
						Phase:     "development",
					},
				},
			},
		},
		{
			name: "TC-renderEpicPlanningSpecific-002: No workflow position",
			info: &services.EpicDisplayInfo{
				Epic:             &models.Epic{BaseEntity: models.BaseEntity{Title: "No Position"}, Status: models.EpicStatusDraft},
				WorkflowPosition: nil,
				Features:         []services.FeatureDisplayItem{},
			},
		},
		{
			name: "TC-renderEpicPlanningSpecific-003: No features",
			info: &services.EpicDisplayInfo{
				Epic:     &models.Epic{BaseEntity: models.BaseEntity{Title: "No Features"}, Status: models.EpicStatusDraft},
				Features: nil,
			},
		},
		{
			name: "TC-renderEpicPlanningSpecific-004: Feature with empty phase",
			info: &services.EpicDisplayInfo{
				Epic: &models.Epic{BaseEntity: models.BaseEntity{Title: "Empty Phase Feature"}, Status: models.EpicStatusActive},
				Features: []services.FeatureDisplayItem{
					{
						Feature:   &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01", Title: "Feature 1"}, Status: "draft"},
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
				{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: "Blocked Task"}},
			},
			approvalBacklogCount: 2,
			features: []FeatureWithDetails{
				{
					Feature:   &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01", Title: "Feature 1"}, Status: "active"},
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
					{BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: "Blocked"}, BlockedReason: &reason},
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
					Feature:    &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F01", Title: "Planning Feature"}, Status: "draft"},
					TaskCount:  0,
					IsPlanning: true,
					Phase:      "planning",
				},
				{
					Feature:    &models.Feature{BaseEntity: models.BaseEntity{Key: "E07-F02", Title: "Active Feature with a very long title that should be truncated by the renderer"}, Status: "active"},
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
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E03"}}, ProgressPct: 50},
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E01"}}, ProgressPct: 75},
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E02"}}, ProgressPct: 25},
			},
			sortBy:   "key",
			wantKeys: []string{"E01", "E02", "E03"},
		},
		{
			name: "Sort by progress",
			epics: []EpicWithProgress{
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E01"}}, ProgressPct: 75},
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E02"}}, ProgressPct: 25},
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E03"}}, ProgressPct: 50},
			},
			sortBy:   "progress",
			wantKeys: []string{"E02", "E03", "E01"},
		},
		{
			name: "Sort by status",
			epics: []EpicWithProgress{
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E01"}, Status: models.EpicStatusCompleted}},
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E02"}, Status: models.EpicStatusDraft}},
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E03"}, Status: models.EpicStatusActive}},
			},
			sortBy:   "status",
			wantKeys: []string{"E02", "E03", "E01"},
		},
		{
			name: "Default sort (empty string) sorts by key",
			epics: []EpicWithProgress{
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E03"}}},
				{Epic: &models.Epic{BaseEntity: models.BaseEntity{Key: "E01"}}},
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

// TestBuildEpicListRows_TitleScalesWithConsoleWidth verifies that
// buildEpicListRows truncates the Title column at a width derived from
// the resolved console width (CC-036). Mirrors the rendering-path test
// in internal/formatters/task_table_test.go to lock in the contract that
// `cli.TitleColumnWidth(70)` flows through this list view.
//
// At the 120-col baseline the truncation cap is 50 chars (120 - 70),
// matching the historical hardcoded behavior. At a 60-col terminal the
// cap clamps to the 20-char floor in TitleColumnWidth. At 200 cols the
// cap widens to 130 chars so longer titles render in full.
func TestBuildEpicListRows_TitleScalesWithConsoleWidth(t *testing.T) {
	// 200 chars — long enough that every test case truncates except the
	// widest terminal.
	longTitle := strings.Repeat("A", 200)

	tests := []struct {
		name              string
		stubbedWidth      int
		wantTruncatedLen  int  // expected len(title) in the rendered row
		wantHasEllipsis   bool // expected to end in "..."
		wantUntruncatedAt int  // sanity: TitleColumnWidth(70) at this width
	}{
		{
			name:              "narrow terminal yields min title width (clamped to 20)",
			stubbedWidth:      60,
			wantTruncatedLen:  20, // 60-70 < 20 floor → 20
			wantHasEllipsis:   true,
			wantUntruncatedAt: 20,
		},
		{
			name:              "baseline 120 reproduces historical 50-char cap",
			stubbedWidth:      120,
			wantTruncatedLen:  50, // 120-70 = 50
			wantHasEllipsis:   true,
			wantUntruncatedAt: 50,
		},
		{
			name:              "wide terminal yields wider title (130 chars)",
			stubbedWidth:      200,
			wantTruncatedLen:  130, // 200-70 = 130
			wantHasEllipsis:   true,
			wantUntruncatedAt: 130,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := cli.SetConsoleWidthForTesting(tt.stubbedWidth)
			defer restore()

			epics := []EpicWithProgress{
				{
					Epic: &models.Epic{
						BaseEntity: models.BaseEntity{Key: "E07", Title: longTitle},
						Status:     models.EpicStatusActive,
						Priority:   models.PriorityMedium,
					},
					ProgressPct: 50.0,
				},
			}

			rows := buildEpicListRows(epics)
			if len(rows) != 1 {
				t.Fatalf("buildEpicListRows returned %d rows, want 1", len(rows))
			}

			gotTitle := rows[0][1] // Title is column index 1
			if len(gotTitle) != tt.wantTruncatedLen {
				t.Errorf("title length = %d, want %d (console width %d, title=%q)",
					len(gotTitle), tt.wantTruncatedLen, tt.stubbedWidth, gotTitle)
			}
			if tt.wantHasEllipsis && !strings.HasSuffix(gotTitle, "...") {
				t.Errorf("title %q should end in '...' when truncated (console width %d)",
					gotTitle, tt.stubbedWidth)
			}
			if got := cli.TitleColumnWidth(70); got != tt.wantUntruncatedAt {
				t.Errorf("cli.TitleColumnWidth(70) = %d, want %d (console width %d)",
					got, tt.wantUntruncatedAt, tt.stubbedWidth)
			}
		})
	}

	// Sanity check: differing widths produce differing truncated lengths.
	// This is the headline contract — width-sensitive rendering.
	t.Run("narrow vs wide produce different lengths", func(t *testing.T) {
		longRow := func(width int) string {
			restore := cli.SetConsoleWidthForTesting(width)
			defer restore()
			rows := buildEpicListRows([]EpicWithProgress{
				{
					Epic: &models.Epic{
						BaseEntity: models.BaseEntity{Key: "E07", Title: longTitle},
						Status:     models.EpicStatusActive,
						Priority:   models.PriorityMedium,
					},
				},
			})
			return rows[0][1]
		}

		narrow := longRow(60)
		wide := longRow(200)
		if len(narrow) == len(wide) {
			t.Errorf("expected narrow (%d) and wide (%d) titles to differ in length, got both %d",
				60, 200, len(narrow))
		}
	})
}
