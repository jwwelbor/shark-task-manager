package status

import (
	"strings"
	"testing"
	"time"
)

// TestGetTerminalWidth tests terminal width detection
func TestGetTerminalWidth(t *testing.T) {
	width := getTerminalWidth()

	// Terminal width should be at least 80 (minimum usable width)
	if width < 80 {
		t.Errorf("Terminal width too small: got %d, expected at least 80", width)
	}

	// Should not exceed reasonable maximum (300 chars)
	if width > 300 {
		t.Errorf("Terminal width unreasonably large: got %d, expected <= 300", width)
	}
}

// TestRenderProgressBar tests progress bar rendering with color coding
func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name           string
		progress       float64
		width          int
		noColor        bool
		expectContains []string
		expectLength   int
	}{
		{
			name:           "zero progress",
			progress:       0.0,
			width:          20,
			noColor:        true,
			expectContains: []string{"[", "]", "0%"},
		},
		{
			name:           "half progress",
			progress:       50.0,
			width:          20,
			noColor:        true,
			expectContains: []string{"[", "]", "50%"},
		},
		{
			name:           "complete progress",
			progress:       100.0,
			width:          20,
			noColor:        true,
			expectContains: []string{"[", "]", "100%"},
		},
		{
			name:           "with color - low progress (red)",
			progress:       20.0,
			width:          20,
			noColor:        false,
			expectContains: []string{"[", "]", "20%"},
		},
		{
			name:           "with color - medium progress (yellow)",
			progress:       50.0,
			width:          20,
			noColor:        false,
			expectContains: []string{"[", "]", "50%"},
		},
		{
			name:           "with color - high progress (green)",
			progress:       90.0,
			width:          20,
			noColor:        false,
			expectContains: []string{"[", "]", "90%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderProgressBar(tt.progress, tt.width, tt.noColor)

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Progress bar missing expected content '%s': got '%s'", expected, result)
				}
			}

			// Verify percentage is displayed
			if !strings.Contains(result, "%") {
				t.Errorf("Progress bar missing percentage indicator: got '%s'", result)
			}
		})
	}
}

// TestFormatProjectSummary tests project summary display
func TestFormatProjectSummary(t *testing.T) {
	summary := &ProjectSummary{
		Epics:    &CountBreakdown{Total: 5, Active: 3},
		Features: &CountBreakdown{Total: 12, Active: 8},
		Tasks: &StatusBreakdown{
			Total:          45,
			Todo:           10,
			InProgress:     5,
			ReadyForReview: 3,
			Completed:      25,
			Blocked:        2,
		},
		OverallProgress: 55.5,
		BlockedCount:    2,
	}

	t.Run("with color", func(t *testing.T) {
		result := formatProjectSummary(summary, false)

		// Should contain all the summary data
		expectContains := []string{
			"PROJECT SUMMARY",
			"Epics",
			"5 total",
			"3 active",
			"Features",
			"12 total",
			"8 active",
			"Tasks",
			"45 total",
			"25 completed",
			"56%", // 55.5% rounds to 56%
			"2 blocked",
		}

		for _, expected := range expectContains {
			if !strings.Contains(result, expected) {
				t.Errorf("Summary missing expected content '%s': got output:\n%s", expected, result)
			}
		}
	})

	t.Run("without color", func(t *testing.T) {
		result := formatProjectSummary(summary, true)

		// Should not contain ANSI color codes
		if strings.Contains(result, "\033[") {
			t.Errorf("Summary contains color codes when noColor=true")
		}

		// Should still contain data (55.5% rounds to 56%)
		if !strings.Contains(result, "56%") {
			t.Errorf("Summary missing progress data: got output:\n%s", result)
		}
	})
}

// TestFormatEpicTable tests epic table with progress bars
func TestFormatEpicTable(t *testing.T) {
	epics := []*EpicSummary{
		{
			Key:             "E01",
			Title:           "Test Epic One",
			ProgressPercent: 75.0,
			Health:          "healthy",
			TasksTotal:      20,
			TasksCompleted:  15,
			TasksBlocked:    0,
			FeaturesTotal:   5,
			FeaturesActive:  2,
		},
		{
			Key:             "E02",
			Title:           "Test Epic Two",
			ProgressPercent: 30.0,
			Health:          "warning",
			TasksTotal:      10,
			TasksCompleted:  3,
			TasksBlocked:    2,
			FeaturesTotal:   3,
			FeaturesActive:  2,
		},
		{
			Key:             "E03",
			Title:           "Test Epic Three",
			ProgressPercent: 10.0,
			Health:          "critical",
			TasksTotal:      15,
			TasksCompleted:  1,
			TasksBlocked:    5,
			FeaturesTotal:   4,
			FeaturesActive:  3,
		},
	}

	t.Run("with color", func(t *testing.T) {
		result := formatEpicTable(epics, false, 120)

		expectContains := []string{
			"EPICS",
			"E01",
			"Test Epic One",
			"75",
			"E02",
			"Test Epic Two",
			"30",
			"E03",
			"Test Epic Three",
			"10",
		}

		for _, expected := range expectContains {
			if !strings.Contains(result, expected) {
				t.Errorf("Epic table missing expected content '%s'", expected)
			}
		}
	})

	t.Run("without color", func(t *testing.T) {
		result := formatEpicTable(epics, true, 120)

		if strings.Contains(result, "\033[") {
			t.Errorf("Epic table contains color codes when noColor=true")
		}
	})
}

// TestFormatActiveTasks tests active tasks grouped by agent
func TestFormatActiveTasks(t *testing.T) {
	activeTasks := map[string][]*TaskInfo{
		"backend": {
			{
				Key:      "T-E01-F01-001",
				Title:    "Backend Task 1",
				Feature:  "E01-F01",
				Epic:     "E01",
				Priority: 5,
			},
			{
				Key:      "T-E01-F02-001",
				Title:    "Backend Task 2",
				Feature:  "E01-F02",
				Epic:     "E01",
				Priority: 3,
			},
		},
		"frontend": {
			{
				Key:      "T-E02-F01-001",
				Title:    "Frontend Task 1",
				Feature:  "E02-F01",
				Epic:     "E02",
				Priority: 8,
			},
		},
	}

	t.Run("with tasks", func(t *testing.T) {
		result := formatActiveTasks(activeTasks, false)

		expectContains := []string{
			"ACTIVE TASKS",
			"BACKEND", // Agent names are uppercased
			"T-E01-F01-001",
			"Backend Task 1",
			"T-E01-F02-001",
			"Backend Task 2",
			"FRONTEND", // Agent names are uppercased
			"T-E02-F01-001",
			"Frontend Task 1",
		}

		for _, expected := range expectContains {
			if !strings.Contains(result, expected) {
				t.Errorf("Active tasks missing expected content '%s': got output:\n%s", expected, result)
			}
		}
	})

	t.Run("empty tasks", func(t *testing.T) {
		emptyTasks := map[string][]*TaskInfo{}
		result := formatActiveTasks(emptyTasks, false)

		if !strings.Contains(result, "No active tasks") {
			t.Errorf("Empty active tasks should show 'No active tasks' message")
		}
	})
}

// TestFormatBlockedTasks tests blocked tasks with reasons
func TestFormatBlockedTasks(t *testing.T) {
	reason1 := "Waiting for API endpoint"
	reason2 := "Database schema not ready"

	blockedTasks := []*BlockedTaskInfo{
		{
			Key:           "T-E01-F01-001",
			Title:         "Blocked Task 1",
			Feature:       "E01-F01",
			Epic:          "E01",
			BlockedReason: &reason1,
		},
		{
			Key:           "T-E02-F01-001",
			Title:         "Blocked Task 2",
			Feature:       "E02-F01",
			Epic:          "E02",
			BlockedReason: &reason2,
		},
	}

	t.Run("with blocked tasks", func(t *testing.T) {
		result := formatBlockedTasks(blockedTasks, false)

		expectContains := []string{
			"BLOCKED TASKS",
			"T-E01-F01-001",
			"Blocked Task 1",
			"Waiting for API endpoint",
			"T-E02-F01-001",
			"Blocked Task 2",
			"Database schema not ready",
		}

		for _, expected := range expectContains {
			if !strings.Contains(result, expected) {
				t.Errorf("Blocked tasks missing expected content '%s'", expected)
			}
		}
	})

	t.Run("empty blocked tasks", func(t *testing.T) {
		emptyBlocked := []*BlockedTaskInfo{}
		result := formatBlockedTasks(emptyBlocked, false)

		if !strings.Contains(result, "No blocked tasks") {
			t.Errorf("Empty blocked tasks should show 'No blocked tasks' message")
		}
	})
}

// TestFormatRecentCompletions tests recent completions with relative time
func TestFormatRecentCompletions(t *testing.T) {
	now := time.Now()
	ago2h := "2 hours ago"
	ago1d := "1 day ago"

	completions := []*CompletionInfo{
		{
			Key:          "T-E01-F01-001",
			Title:        "Completed Task 1",
			Feature:      "E01-F01",
			Epic:         "E01",
			CompletedAt:  now.Add(-2 * time.Hour),
			CompletedAgo: &ago2h,
		},
		{
			Key:          "T-E02-F01-001",
			Title:        "Completed Task 2",
			Feature:      "E02-F01",
			Epic:         "E02",
			CompletedAt:  now.Add(-24 * time.Hour),
			CompletedAgo: &ago1d,
		},
	}

	t.Run("with completions", func(t *testing.T) {
		result := formatRecentCompletions(completions, false)

		expectContains := []string{
			"RECENT COMPLETIONS",
			"T-E01-F01-001",
			"Completed Task 1",
			"2 hours ago",
			"T-E02-F01-001",
			"Completed Task 2",
			"1 day ago",
		}

		for _, expected := range expectContains {
			if !strings.Contains(result, expected) {
				t.Errorf("Recent completions missing expected content '%s'", expected)
			}
		}
	})

	t.Run("empty completions", func(t *testing.T) {
		emptyCompletions := []*CompletionInfo{}
		result := formatRecentCompletions(emptyCompletions, false)

		if result != "" {
			t.Errorf("Empty completions should return empty string, got: %s", result)
		}
	})
}

// TestFormatLinkedBugs tests the linked bug summary renderer for feature-level status.
// Covers TC-F07-009 (bugs present) and TC-F07-010 (nil bugs → empty string).
func TestFormatLinkedBugs(t *testing.T) {
	tests := []struct {
		name           string
		bugs           *BugFeatureSummary
		noColor        bool
		expectEmpty    bool
		expectContains []string
	}{
		{
			name:        "nil bugs returns empty string (TC-F07-010)",
			bugs:        nil,
			noColor:     true,
			expectEmpty: true,
		},
		{
			name: "bugs with severity breakdown (TC-F07-009)",
			bugs: &BugFeatureSummary{
				TotalLinked: 3,
				OpenCount:   2,
				OpenBySeverity: map[string]int{
					"high":   1,
					"medium": 1,
				},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"Linked Bugs: 3",
				"2 open",
				"high: 1",
				"medium: 1",
			},
		},
		{
			name: "all bugs resolved - open count zero",
			bugs: &BugFeatureSummary{
				TotalLinked:    2,
				OpenCount:      0,
				OpenBySeverity: map[string]int{},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"Linked Bugs: 2",
				"0 open",
			},
		},
		{
			name: "single open bug",
			bugs: &BugFeatureSummary{
				TotalLinked: 1,
				OpenCount:   1,
				OpenBySeverity: map[string]int{
					"critical": 1,
				},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"Linked Bugs: 1",
				"1 open",
				"critical: 1",
			},
		},
		{
			name: "with color flag - still contains bug info",
			bugs: &BugFeatureSummary{
				TotalLinked: 3,
				OpenCount:   2,
				OpenBySeverity: map[string]int{
					"high":   1,
					"medium": 1,
				},
			},
			noColor:     false,
			expectEmpty: false,
			expectContains: []string{
				"Linked Bugs: 3",
				"2 open",
				"high: 1",
				"medium: 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLinkedBugs(tt.bugs, tt.noColor)

			if tt.expectEmpty {
				if result != "" {
					t.Errorf("expected empty string, got: %q", result)
				}
				return
			}

			if result == "" {
				t.Fatal("expected non-empty result, got empty string")
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("output missing %q: got %q", expected, result)
				}
			}
		})
	}
}

// TestFormatBugSummary tests the bug section renderer for the dashboard.
// Covers TC-F07-001 (status breakdown), TC-F07-005 (severity open only),
// TC-F07-006 (severity omitted when all terminal), Section 3.3 formatter tests.
func TestFormatBugSummary(t *testing.T) {
	tests := []struct {
		name           string
		bugs           *BugDashboardSummary
		noColor        bool
		expectEmpty    bool
		expectContains []string
		expectMissing  []string
	}{
		{
			name:        "nil input returns empty string",
			bugs:        nil,
			noColor:     true,
			expectEmpty: true,
		},
		{
			name: "basic data with status breakdown (TC-F07-001)",
			bugs: &BugDashboardSummary{
				Total: 3,
				ByStatus: map[string]int{
					"reported": 1,
					"in_fix":   1,
					"resolved": 1,
				},
				OpenBySeverity: map[string]int{
					"high":   1,
					"medium": 1,
				},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"BUGS",
				"Total: 3",
				"reported:",
				"in_fix:",
				"resolved:",
			},
		},
		{
			name: "severity section shows open bugs only (TC-F07-005)",
			bugs: &BugDashboardSummary{
				Total: 5,
				ByStatus: map[string]int{
					"reported": 1,
					"triaged":  2,
					"resolved": 1,
					"wont_fix": 1,
				},
				OpenBySeverity: map[string]int{
					"critical": 1,
					"high":     2,
					"medium":   0,
					"low":      0,
				},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"BUGS",
				"Total: 5",
				"Open Bug Severity:",
				"critical:",
				"high:",
			},
		},
		{
			name: "severity subsection omitted when all zero (TC-F07-006)",
			bugs: &BugDashboardSummary{
				Total: 3,
				ByStatus: map[string]int{
					"resolved": 2,
					"wont_fix": 1,
				},
				OpenBySeverity: map[string]int{
					"critical": 0,
					"high":     0,
					"medium":   0,
					"low":      0,
				},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"BUGS",
				"Total: 3",
			},
			expectMissing: []string{
				"Open Bug Severity:",
			},
		},
		{
			name: "status ordering is reported triaged in_fix in_verification resolved wont_fix duplicate",
			bugs: &BugDashboardSummary{
				Total: 7,
				ByStatus: map[string]int{
					"duplicate":       1,
					"wont_fix":        1,
					"resolved":        1,
					"in_verification": 1,
					"in_fix":          1,
					"triaged":         1,
					"reported":        1,
				},
				OpenBySeverity: map[string]int{},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"reported:",
				"triaged:",
				"in_fix:",
				"in_verification:",
				"resolved:",
				"wont_fix:",
				"duplicate:",
			},
		},
		{
			name: "noColor false still contains bug info",
			bugs: &BugDashboardSummary{
				Total:          2,
				ByStatus:       map[string]int{"reported": 2},
				OpenBySeverity: map[string]int{"high": 2},
			},
			noColor:     false,
			expectEmpty: false,
			expectContains: []string{
				"BUGS",
				"Total: 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBugSummary(tt.bugs, tt.noColor)

			if tt.expectEmpty {
				if result != "" {
					t.Errorf("expected empty string, got: %q", result)
				}
				return
			}

			if result == "" {
				t.Fatal("expected non-empty result, got empty string")
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("output missing %q:\n%s", expected, result)
				}
			}

			for _, missing := range tt.expectMissing {
				if strings.Contains(result, missing) {
					t.Errorf("output should NOT contain %q:\n%s", missing, result)
				}
			}
		})
	}
}

// TestFormatChangeCardSummary tests the change-card section renderer.
// Covers TC-F07-007 (status counts), Section 3.3 formatter tests.
func TestFormatChangeCardSummary(t *testing.T) {
	tests := []struct {
		name           string
		cards          *ChangeCardDashboardSummary
		noColor        bool
		expectEmpty    bool
		expectContains []string
	}{
		{
			name:        "nil input returns empty string",
			cards:       nil,
			noColor:     true,
			expectEmpty: true,
		},
		{
			name: "basic data with status breakdown (TC-F07-007)",
			cards: &ChangeCardDashboardSummary{
				Total: 5,
				ByStatus: map[string]int{
					"proposed":    2,
					"approved":    1,
					"in_progress": 1,
					"completed":   1,
				},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"CHANGE CARDS",
				"Total: 5",
				"proposed:",
				"approved:",
				"in_progress:",
				"completed:",
			},
		},
		{
			name: "status ordering is proposed approved in_progress completed declined",
			cards: &ChangeCardDashboardSummary{
				Total: 5,
				ByStatus: map[string]int{
					"declined":    1,
					"completed":   1,
					"in_progress": 1,
					"approved":    1,
					"proposed":    1,
				},
			},
			noColor:     true,
			expectEmpty: false,
			expectContains: []string{
				"proposed:",
				"approved:",
				"in_progress:",
				"completed:",
				"declined:",
			},
		},
		{
			name: "noColor false still renders",
			cards: &ChangeCardDashboardSummary{
				Total:    3,
				ByStatus: map[string]int{"proposed": 3},
			},
			noColor:     false,
			expectEmpty: false,
			expectContains: []string{
				"CHANGE CARDS",
				"Total: 3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatChangeCardSummary(tt.cards, tt.noColor)

			if tt.expectEmpty {
				if result != "" {
					t.Errorf("expected empty string, got: %q", result)
				}
				return
			}

			if result == "" {
				t.Fatal("expected non-empty result, got empty string")
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("output missing %q:\n%s", expected, result)
				}
			}
		})
	}
}

// TestFormatDurationFromSecs tests the duration helper (Section 3.3 formatter tests).
func TestFormatDurationFromSecs(t *testing.T) {
	tests := []struct {
		name     string
		secs     float64
		expected string
	}{
		{
			name:     "zero seconds",
			secs:     0.0,
			expected: "0h 0m",
		},
		{
			name:     "minutes only",
			secs:     3660.0, // 1h 1m
			expected: "1h 1m",
		},
		{
			name:     "hours (TC-F07-015: 4h 0m)",
			secs:     14400.0, // 4 hours
			expected: "4h 0m",
		},
		{
			name:     "days (TC: 3d 0h)",
			secs:     259200.0, // 3 days
			expected: "3d 0h",
		},
		{
			name:     "days with remaining hours",
			secs:     90000.0, // 25h = 1d 1h
			expected: "1d 1h",
		},
		{
			name:     "fractional seconds rounds to minute",
			secs:     3661.9, // 1h 1m 1.9s -> 1h 1m
			expected: "1h 1m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationFromSecs(tt.secs)
			if result != tt.expected {
				t.Errorf("formatDurationFromSecs(%v) = %q, want %q", tt.secs, result, tt.expected)
			}
		})
	}
}

// TestFormatDashboard_WithBugAndChangeCard tests FormatDashboard integration
// with the new bug and change-card sections.
func TestFormatDashboard_WithBugAndChangeCard(t *testing.T) {
	baseDashboard := func() *StatusDashboard {
		return &StatusDashboard{
			Summary: &ProjectSummary{
				Epics:    &CountBreakdown{Total: 1, Active: 1},
				Features: &CountBreakdown{Total: 2, Active: 1},
				Tasks: &StatusBreakdown{
					Total:      5,
					InProgress: 2,
					Completed:  3,
				},
				OverallProgress: 60.0,
			},
			Epics: []*EpicSummary{
				{Key: "E01", Title: "Epic", ProgressPercent: 60.0, Health: "healthy"},
			},
			ActiveTasks:       map[string][]*TaskInfo{},
			BlockedTasks:      []*BlockedTaskInfo{},
			RecentCompletions: []*CompletionInfo{},
		}
	}

	t.Run("dashboard includes BUGS section when BugSummary non-nil", func(t *testing.T) {
		d := baseDashboard()
		d.BugSummary = &BugDashboardSummary{
			Total:          2,
			ByStatus:       map[string]int{"reported": 1, "triaged": 1},
			OpenBySeverity: map[string]int{"high": 1, "medium": 1},
		}

		result := FormatDashboard(d, true)

		if !strings.Contains(result, "BUGS") {
			t.Errorf("dashboard output should contain BUGS section when BugSummary is set")
		}
		if !strings.Contains(result, "Total: 2") {
			t.Errorf("dashboard output should contain bug total")
		}
	})

	t.Run("dashboard excludes BUGS section when BugSummary nil", func(t *testing.T) {
		d := baseDashboard()
		d.BugSummary = nil

		result := FormatDashboard(d, true)

		if strings.Contains(result, "BUGS") {
			t.Errorf("dashboard should NOT contain BUGS section when BugSummary is nil")
		}
	})

	t.Run("dashboard includes CHANGE CARDS section when ChangeCardSummary non-nil", func(t *testing.T) {
		d := baseDashboard()
		d.ChangeCardSummary = &ChangeCardDashboardSummary{
			Total:    3,
			ByStatus: map[string]int{"proposed": 2, "approved": 1},
		}

		result := FormatDashboard(d, true)

		if !strings.Contains(result, "CHANGE CARDS") {
			t.Errorf("dashboard output should contain CHANGE CARDS section when ChangeCardSummary is set")
		}
		if !strings.Contains(result, "Total: 3") {
			t.Errorf("dashboard output should contain change card total")
		}
	})

	t.Run("dashboard excludes CHANGE CARDS section when ChangeCardSummary nil", func(t *testing.T) {
		d := baseDashboard()
		d.ChangeCardSummary = nil

		result := FormatDashboard(d, true)

		if strings.Contains(result, "CHANGE CARDS") {
			t.Errorf("dashboard should NOT contain CHANGE CARDS section when ChangeCardSummary is nil")
		}
	})

	t.Run("dashboard renders both sections together", func(t *testing.T) {
		d := baseDashboard()
		d.BugSummary = &BugDashboardSummary{
			Total:          1,
			ByStatus:       map[string]int{"reported": 1},
			OpenBySeverity: map[string]int{"critical": 1},
		}
		d.ChangeCardSummary = &ChangeCardDashboardSummary{
			Total:    1,
			ByStatus: map[string]int{"proposed": 1},
		}

		result := FormatDashboard(d, true)

		for _, expected := range []string{"BUGS", "CHANGE CARDS", "PROJECT SUMMARY"} {
			if !strings.Contains(result, expected) {
				t.Errorf("dashboard missing section %q", expected)
			}
		}
	})
}

// TestFormatDashboard tests complete dashboard formatting
func TestFormatDashboard(t *testing.T) {
	dashboard := &StatusDashboard{
		Summary: &ProjectSummary{
			Epics:    &CountBreakdown{Total: 2, Active: 2},
			Features: &CountBreakdown{Total: 5, Active: 3},
			Tasks: &StatusBreakdown{
				Total:          20,
				Todo:           5,
				InProgress:     3,
				ReadyForReview: 2,
				Completed:      8,
				Blocked:        2,
			},
			OverallProgress: 40.0,
			BlockedCount:    2,
		},
		Epics: []*EpicSummary{
			{
				Key:             "E01",
				Title:           "Test Epic",
				ProgressPercent: 50.0,
				Health:          "warning",
				TasksTotal:      10,
				TasksCompleted:  5,
				TasksBlocked:    1,
			},
		},
		ActiveTasks: map[string][]*TaskInfo{
			"backend": {
				{
					Key:      "T-E01-F01-001",
					Title:    "Active Task",
					Feature:  "E01-F01",
					Epic:     "E01",
					Priority: 5,
				},
			},
		},
		BlockedTasks:      []*BlockedTaskInfo{},
		RecentCompletions: []*CompletionInfo{},
	}

	t.Run("complete dashboard", func(t *testing.T) {
		result := FormatDashboard(dashboard, false)

		// Should contain all sections
		expectContains := []string{
			"PROJECT SUMMARY",
			"EPICS",
			"ACTIVE TASKS",
		}

		for _, expected := range expectContains {
			if !strings.Contains(result, expected) {
				t.Errorf("Dashboard missing expected section '%s'", expected)
			}
		}

		// Should be non-empty
		if len(result) < 100 {
			t.Errorf("Dashboard output suspiciously short: %d chars", len(result))
		}
	})

	t.Run("no color mode", func(t *testing.T) {
		result := FormatDashboard(dashboard, true)

		// Should not contain ANSI codes
		if strings.Contains(result, "\033[") {
			t.Errorf("Dashboard contains color codes when noColor=true")
		}
	})
}
