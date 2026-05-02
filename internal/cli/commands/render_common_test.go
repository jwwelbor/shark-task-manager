package commands

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// Test Suite 2.1.1: renderHeader()
func TestRenderHeader(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		key        string
	}{
		{
			name:       "TC-renderHeader-001: Epic type capitalizes correctly",
			entityType: "epic",
			key:        "E07",
		},
		{
			name:       "TC-renderHeader-002: Feature type capitalizes correctly",
			entityType: "feature",
			key:        "E07-F01",
		},
		{
			name:       "TC-renderHeader-003: Task type capitalizes correctly",
			entityType: "task",
			key:        "E07-F01-001",
		},
		{
			name:       "TC-renderHeader-004: Empty strings handled (no panic)",
			entityType: "",
			key:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that it doesn't panic - pterm output is difficult to capture
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderHeader() panicked: %v", r)
				}
			}()
			renderHeader(tt.entityType, tt.key)
		})
	}
}

// Test Suite 2.1.2: renderBasicInfo()
func TestRenderBasicInfo(t *testing.T) {
	tests := []struct {
		name string
		info [][]string
	}{
		{
			name: "TC-renderBasicInfo-001: Multiple rows render as table",
			info: [][]string{
				{"Title", "User Authentication"},
				{"Status", "active"},
				{"Priority", "high"},
			},
		},
		{
			name: "TC-renderBasicInfo-002: Single row renders correctly",
			info: [][]string{
				{"Title", "Single Task"},
			},
		},
		{
			name: "TC-renderBasicInfo-003: Empty array skips rendering",
			info: [][]string{},
		},
		{
			name: "TC-renderBasicInfo-004: Nil input skips rendering",
			info: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderBasicInfo() panicked: %v", r)
				}
			}()
			renderBasicInfo(tt.info)
		})
	}
}

// Test Suite 2.1.3: renderValidTransitions()
func TestRenderValidTransitions(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		transitions []string
	}{
		{
			name:        "TC-renderValidTransitions-001: Multiple transitions display as bulleted list",
			status:      "in_progress",
			transitions: []string{"ready_for_review", "blocked"},
		},
		{
			name:        "TC-renderValidTransitions-002: Single transition displays correctly",
			status:      "todo",
			transitions: []string{"in_progress"},
		},
		{
			name:        "TC-renderValidTransitions-003: Empty array skips section",
			status:      "completed",
			transitions: []string{},
		},
		{
			name:        "TC-renderValidTransitions-004: Nil array skips section",
			status:      "completed",
			transitions: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderValidTransitions() panicked: %v", r)
				}
			}()
			renderValidTransitions(tt.status, tt.transitions)
		})
	}
}

// Test Suite 2.1.4: renderRelatedDocuments()
func TestRenderRelatedDocuments(t *testing.T) {
	tests := []struct {
		name string
		docs []*models.Document
	}{
		{
			name: "TC-renderRelatedDocuments-001: Multiple docs display with title and path",
			docs: []*models.Document{
				{Title: "PRD", FilePath: "docs/plan/E07-F31/feature.md"},
				{Title: "Research Report", FilePath: "docs/plan/E07-F31/research.md"},
			},
		},
		{
			name: "TC-renderRelatedDocuments-002: Single doc displays correctly",
			docs: []*models.Document{
				{Title: "Design Doc", FilePath: "docs/design.md"},
			},
		},
		{
			name: "TC-renderRelatedDocuments-003: Empty array skips section",
			docs: []*models.Document{},
		},
		{
			name: "TC-renderRelatedDocuments-004: Nil array skips section",
			docs: nil,
		},
		{
			name: "TC-renderRelatedDocuments-005: Docs with missing fields handled",
			docs: []*models.Document{
				{Title: "", FilePath: "docs/empty-title.md"},
				{Title: "No Path", FilePath: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderRelatedDocuments() panicked: %v", r)
				}
			}()
			renderRelatedDocuments(tt.docs)
		})
	}
}

// Test Suite 2.1.5: GetValidTransitions()
func TestGetValidTransitions(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		workflow *config.WorkflowConfig
		want     []string
	}{
		{
			name:   "TC-GetValidTransitions-001: Valid status returns transitions",
			status: "in_progress",
			workflow: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_progress": {"ready_for_review", "blocked"},
				},
			},
			want: []string{"ready_for_review", "blocked"},
		},
		{
			name:   "TC-GetValidTransitions-002: Terminal status returns empty array",
			status: "completed",
			workflow: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_progress": {"completed"},
					"completed":   {},
				},
			},
			want: []string{},
		},
		{
			name:     "TC-GetValidTransitions-003: Nil workflow returns empty array",
			status:   "in_progress",
			workflow: nil,
			want:     []string{},
		},
		{
			name:   "TC-GetValidTransitions-004: Empty status_flow map returns empty array",
			status: "in_progress",
			workflow: &config.WorkflowConfig{
				StatusFlow: map[string][]string{},
			},
			want: []string{},
		},
		{
			name:   "TC-GetValidTransitions-005: Status not in status_flow returns empty array",
			status: "unknown_status",
			workflow: &config.WorkflowConfig{
				StatusFlow: map[string][]string{
					"in_progress": {"ready_for_review"},
				},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetValidTransitions(tt.status, tt.workflow)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValidTransitions() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test Suite 2.2: renderOrchestratorAction()
func TestRenderOrchestratorAction(t *testing.T) {
	tests := []struct {
		name   string
		action *config.PopulatedAction
	}{
		{
			name: "TC-renderOrchestratorAction-001: Non-nil action delegates correctly",
			action: &config.PopulatedAction{
				Action:      "review",
				AgentType:   "tech_lead",
				Instruction: "Review the code changes",
			},
		},
		{
			name:   "TC-renderOrchestratorAction-002: Nil action shows 'None configured'",
			action: nil,
		},
		{
			name: "TC-renderOrchestratorAction-003: Action with all fields displays completely",
			action: &config.PopulatedAction{
				Action:      "start",
				AgentType:   "developer",
				Skills:      []string{"go", "testing"},
				Instruction: "Start implementing the feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderOrchestratorAction() panicked: %v", r)
				}
			}()
			renderOrchestratorAction(tt.action)
		})
	}
}

// Test Suite 2.3: renderNotes() and renderContextData()
func TestRenderNotes(t *testing.T) {
	// Create sample notes with timestamps
	now := time.Now()
	tests := []struct {
		name  string
		notes []*models.EntityNote
	}{
		{
			name: "TC-renderNotes-001: Displays most recent 10 notes",
			notes: func() []*models.EntityNote {
				notes := make([]*models.EntityNote, 15)
				for i := 0; i < 15; i++ {
					notes[i] = &models.EntityNote{
						NoteType:  models.NoteTypeComment,
						Content:   "Note " + string(rune('A'+i)),
						CreatedAt: now.Add(-time.Duration(15-i) * time.Hour),
					}
				}
				return notes
			}(),
		},
		{
			name: "TC-renderNotes-002: Truncates notes > 80 chars",
			notes: []*models.EntityNote{
				{
					NoteType:  models.NoteTypeComment,
					Content:   strings.Repeat("a", 100), // 100 characters
					CreatedAt: now,
				},
			},
		},
		{
			name:  "TC-renderNotes-003: Empty notes skips section",
			notes: []*models.EntityNote{},
		},
		{
			name: "TC-renderNotes-004: Less than 10 notes displays all",
			notes: []*models.EntityNote{
				{
					NoteType:  models.NoteTypeComment,
					Content:   "Note 1",
					CreatedAt: now,
				},
				{
					NoteType:  models.NoteTypeDecision,
					Content:   "Note 2",
					CreatedAt: now.Add(-1 * time.Hour),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderNotes() panicked: %v", r)
				}
			}()
			renderNotes(tt.notes)
		})
	}
}

func TestRenderContextData(t *testing.T) {
	currentStep := "Step 2"
	tests := []struct {
		name        string
		contextData *models.ContextData
	}{
		{
			name: "TC-renderContextData-001: Displays context fields",
			contextData: &models.ContextData{
				Progress: &models.ProgressContext{
					CompletedSteps: []string{"Step 1"},
					CurrentStep:    &currentStep,
					RemainingSteps: []string{"Step 3"},
				},
			},
		},
		{
			name:        "TC-renderContextData-002: Nil context skips section",
			contextData: nil,
		},
		{
			name:        "TC-renderContextData-003: Empty context skips section",
			contextData: &models.ContextData{},
		},
		{
			name: "TC-renderContextData-004: Context with implementation decisions",
			contextData: &models.ContextData{
				ImplementationDecisions: map[string]string{
					"framework": "React",
					"database":  "PostgreSQL",
				},
			},
		},
		{
			name: "TC-renderContextData-005: Context with open questions",
			contextData: &models.ContextData{
				OpenQuestions: []string{"How to handle authentication?", "Which API version?"},
			},
		},
		{
			name: "TC-renderContextData-006: Context with blockers",
			contextData: &models.ContextData{
				Blockers: []models.BlockerContext{
					{
						Description:  "Waiting for API access",
						BlockerType:  "external",
						BlockedSince: time.Now(),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("renderContextData() panicked: %v", r)
				}
			}()
			renderContextData(tt.contextData)
		})
	}
}

// Test Suite AC-003: truncateRunes() - rune-safe string truncation
func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxRunes int
		want     string
	}{
		{
			name:     "TC-003-01: ASCII text longer than limit truncated with ellipsis",
			input:    strings.Repeat("a", 100),
			maxRunes: 77,
			want:     strings.Repeat("a", 77) + "...",
		},
		{
			name:     "TC-003-02: CJK characters (3-byte UTF-8) truncated at rune boundary",
			input:    strings.Repeat("\u4e16", 100), // 100 CJK chars (each 3 bytes)
			maxRunes: 77,
			want:     strings.Repeat("\u4e16", 77) + "...",
		},
		{
			name:     "TC-003-03: Emoji characters (4-byte UTF-8) truncated at rune boundary",
			input:    strings.Repeat("\U0001F600", 100), // 100 emoji (each 4 bytes)
			maxRunes: 77,
			want:     strings.Repeat("\U0001F600", 77) + "...",
		},
		{
			name:     "TC-003-04: Mixed ASCII and multi-byte truncated at rune boundary",
			input:    "Hello " + strings.Repeat("\u4e16", 80), // 6 ASCII + 80 CJK = 86 runes
			maxRunes: 77,
			want:     "Hello " + strings.Repeat("\u4e16", 71) + "...",
		},
		{
			name:     "TC-003-05: Exactly maxRunes - no truncation no ellipsis",
			input:    strings.Repeat("a", 77),
			maxRunes: 77,
			want:     strings.Repeat("a", 77),
		},
		{
			name:     "TC-003-06: Fewer than maxRunes - no truncation",
			input:    "short string",
			maxRunes: 77,
			want:     "short string",
		},
		{
			name:     "TC-003-07: Empty string - no panic",
			input:    "",
			maxRunes: 77,
			want:     "",
		},
		{
			name:     "TC-003-08: Single character - no truncation",
			input:    "x",
			maxRunes: 77,
			want:     "x",
		},
		{
			name:     "TC-003-09: Exactly maxRunes+1 - truncated with ellipsis",
			input:    strings.Repeat("b", 78),
			maxRunes: 77,
			want:     strings.Repeat("b", 77) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateRunes(tt.input, tt.maxRunes)
			if got != tt.want {
				t.Errorf("truncateRunes() = %q (len=%d), want %q (len=%d)",
					got, len([]rune(got)), tt.want, len([]rune(tt.want)))
			}
		})
	}
}

// TestFitColumn covers the truncate-or-pad helper used by Title columns
// in list views. Each output must be exactly maxLen runes so pterm renders
// the column at full reserved width.
func TestFitColumn(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short padded with spaces", "abc", 6, "abc   "},
		{"exact length unchanged", "abcdef", 6, "abcdef"},
		{"longer truncated with ellipsis", "abcdefghij", 6, "abc..."},
		{"unicode short padded", "café", 6, "café  "},
		{"unicode truncated", "caférestaurant", 6, "caf..."},
		{"empty padded", "", 4, "    "},
		{"empty maxLen returns empty", "anything", 0, ""},
		{"negative maxLen returns empty", "anything", -1, ""},
		{"maxLen 1 truncates without ellipsis", "abcdef", 1, "a"},
		{"maxLen 3 truncates without ellipsis", "abcdef", 3, "abc"},
		{"maxLen 4 truncates with ellipsis", "abcdef", 4, "a..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fitColumn(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("fitColumn(%q, %d) = %q (len %d), want %q (len %d)",
					tt.input, tt.maxLen, got, len([]rune(got)), tt.want, len([]rune(tt.want)))
			}
			// Invariant: output length equals maxLen (or 0 when maxLen <= 0).
			gotLen := len([]rune(got))
			wantLen := tt.maxLen
			if wantLen <= 0 {
				wantLen = 0
			}
			if gotLen != wantLen {
				t.Errorf("fitColumn(%q, %d): output len %d, want %d",
					tt.input, tt.maxLen, gotLen, wantLen)
			}
		})
	}
}

// Test capitalize helper function
func TestCapitalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase word",
			input: "epic",
			want:  "Epic",
		},
		{
			name:  "already capitalized",
			input: "Feature",
			want:  "Feature",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single character",
			input: "t",
			want:  "T",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := capitalize(tt.input)
			if got != tt.want {
				t.Errorf("capitalize() = %q, want %q", got, tt.want)
			}
		})
	}
}
