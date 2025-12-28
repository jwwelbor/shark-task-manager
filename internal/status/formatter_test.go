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
