package status

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"golang.org/x/term"
)

// getTerminalWidth detects the current terminal width
func getTerminalWidth() int {
	// Try to get actual terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 80 {
		// Default to 120 if we can't detect or it's too small
		return 120
	}

	// Cap at reasonable maximum
	if width > 300 {
		return 300
	}

	return width
}

// renderProgressBar creates a visual progress bar with color coding
func renderProgressBar(progress float64, width int, noColor bool) string {
	if width < 10 {
		width = 10 // Minimum bar width
	}

	// Calculate filled portion (leave room for brackets and percentage)
	barWidth := width - 7 // "[ ]" + " XX%"
	if barWidth < 5 {
		barWidth = 5
	}

	filled := int((progress / 100.0) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	empty := barWidth - filled

	// Build bar components
	filledBar := strings.Repeat("█", filled)
	emptyBar := strings.Repeat("░", empty)

	// Color coding based on progress
	var bar string
	if noColor {
		bar = fmt.Sprintf("[%s%s] %.0f%%", filledBar, emptyBar, progress)
	} else {
		var coloredFilled string
		if progress >= 75.0 {
			coloredFilled = pterm.Green(filledBar)
		} else if progress >= 25.0 {
			coloredFilled = pterm.Yellow(filledBar)
		} else {
			coloredFilled = pterm.Red(filledBar)
		}
		bar = fmt.Sprintf("[%s%s] %.0f%%", coloredFilled, emptyBar, progress)
	}

	return bar
}

// formatProjectSummary formats the project summary section
func formatProjectSummary(summary *ProjectSummary, noColor bool) string {
	var sb strings.Builder

	// Header
	if noColor {
		sb.WriteString("=== PROJECT SUMMARY ===\n\n")
	} else {
		sb.WriteString(pterm.DefaultHeader.WithFullWidth().Sprint("PROJECT SUMMARY"))
		sb.WriteString("\n\n")
	}

	// Epics
	sb.WriteString(fmt.Sprintf("Epics:    %d total, %d active\n",
		summary.Epics.Total, summary.Epics.Active))

	// Features
	sb.WriteString(fmt.Sprintf("Features: %d total, %d active\n",
		summary.Features.Total, summary.Features.Active))

	// Tasks
	sb.WriteString(fmt.Sprintf("Tasks:    %d total, %d completed, %d in progress, %d blocked\n",
		summary.Tasks.Total, summary.Tasks.Completed, summary.Tasks.InProgress, summary.Tasks.Blocked))

	// Progress bar
	sb.WriteString("\nOverall Progress: ")
	progressBar := renderProgressBar(summary.OverallProgress, 50, noColor)
	sb.WriteString(progressBar)
	sb.WriteString("\n")

	// Blocked warning if applicable
	if summary.BlockedCount > 0 {
		if noColor {
			sb.WriteString(fmt.Sprintf("\n⚠ %d blocked tasks require attention\n", summary.BlockedCount))
		} else {
			sb.WriteString(fmt.Sprintf("\n%s\n", pterm.Warning.Sprintf("%d blocked tasks require attention", summary.BlockedCount)))
		}
	}

	return sb.String()
}

// formatEpicTable formats the epic breakdown table with progress bars
func formatEpicTable(epics []*EpicSummary, noColor bool, termWidth int) string {
	var sb strings.Builder

	if len(epics) == 0 {
		return ""
	}

	// Header
	if noColor {
		sb.WriteString("\n=== EPICS ===\n\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(pterm.DefaultHeader.WithFullWidth().Sprint("EPICS"))
		sb.WriteString("\n\n")
	}

	// Calculate column widths
	progressBarWidth := 40

	// Build table data
	tableData := pterm.TableData{
		{"Key", "Title", "Progress", "Health", "Tasks", "Features"},
	}

	for _, epic := range epics {
		// Check if epic is in planning mode
		if epic.IsPlanning {
			// Planning mode: show phase and status instead of progress
			phaseStr := epic.Phase
			if phaseStr == "" {
				phaseStr = "planning"
			}

			var modeStr string
			if noColor {
				modeStr = "(planning)"
			} else {
				modeStr = pterm.Cyan("(planning)")
			}

			tableData = append(tableData, []string{
				epic.Key,
				epic.Title,
				modeStr,
				phaseStr,
				"-",
				"-",
			})
			continue
		}

		// Aggregation mode: show progress and task counts
		// Format health indicator
		var healthStr string
		if noColor {
			healthStr = epic.Health
		} else {
			switch epic.Health {
			case "healthy":
				healthStr = pterm.Green("●") + " healthy"
			case "warning":
				healthStr = pterm.Yellow("●") + " warning"
			case "critical":
				healthStr = pterm.Red("●") + " critical"
			default:
				healthStr = epic.Health
			}
		}

		// Progress bar
		progressBar := renderProgressBar(epic.ProgressPercent, progressBarWidth, noColor)

		// Task info
		tasksStr := fmt.Sprintf("%d/%d", epic.TasksCompleted, epic.TasksTotal)
		if epic.TasksBlocked > 0 {
			tasksStr += fmt.Sprintf(" (%d blocked)", epic.TasksBlocked)
		}

		// Features info
		featuresStr := fmt.Sprintf("%d/%d active", epic.FeaturesActive, epic.FeaturesTotal)

		tableData = append(tableData, []string{
			epic.Key,
			epic.Title,
			progressBar,
			healthStr,
			tasksStr,
			featuresStr,
		})
	}

	// Render table
	if noColor {
		// Simple text table for no-color mode
		for i, row := range tableData {
			if i == 0 {
				sb.WriteString(strings.Join(row, " | "))
				sb.WriteString("\n")
				sb.WriteString(strings.Repeat("-", 80))
				sb.WriteString("\n")
			} else {
				sb.WriteString(strings.Join(row, " | "))
				sb.WriteString("\n")
			}
		}
	} else {
		tableStr, _ := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()
		sb.WriteString(tableStr)
	}

	return sb.String()
}

// formatActiveTasks formats active tasks grouped by agent type
func formatActiveTasks(activeTasks map[string][]*TaskInfo, noColor bool) string {
	var sb strings.Builder

	if len(activeTasks) == 0 {
		sb.WriteString("\nNo active tasks\n")
		return sb.String()
	}

	// Header
	if noColor {
		sb.WriteString("\n=== ACTIVE TASKS ===\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(pterm.DefaultHeader.WithFullWidth().Sprint("ACTIVE TASKS"))
		sb.WriteString("\n")
	}

	// Sort agent types for consistent output
	agentTypes := make([]string, 0, len(activeTasks))
	for agentType := range activeTasks {
		agentTypes = append(agentTypes, agentType)
	}
	sort.Strings(agentTypes)

	// Display tasks grouped by agent
	for _, agentType := range agentTypes {
		tasks := activeTasks[agentType]

		// Agent section header
		sb.WriteString("\n")
		if noColor {
			sb.WriteString(fmt.Sprintf("## %s (%d)\n", strings.ToUpper(agentType), len(tasks)))
		} else {
			sb.WriteString(pterm.DefaultSection.Sprintf("%s (%d)", strings.ToUpper(agentType), len(tasks)))
			sb.WriteString("\n")
		}

		// List tasks
		for _, task := range tasks {
			var priorityStr string
			if task.Priority >= 8 {
				priorityStr = "!!!"
			} else if task.Priority >= 5 {
				priorityStr = "!!"
			} else {
				priorityStr = "!"
			}

			if noColor {
				sb.WriteString(fmt.Sprintf("  [%s] %s: %s (%s)\n",
					priorityStr, task.Key, task.Title, task.Feature))
			} else {
				var coloredPriority string
				if task.Priority >= 8 {
					coloredPriority = pterm.Red(priorityStr)
				} else if task.Priority >= 5 {
					coloredPriority = pterm.Yellow(priorityStr)
				} else {
					coloredPriority = pterm.LightWhite(priorityStr)
				}

				sb.WriteString(fmt.Sprintf("  %s %s: %s %s\n",
					coloredPriority,
					pterm.Cyan(task.Key),
					task.Title,
					pterm.Gray(fmt.Sprintf("(%s)", task.Feature))))
			}
		}
	}

	return sb.String()
}

// formatBlockedTasks formats blocked tasks with their blocking reasons
func formatBlockedTasks(blockedTasks []*BlockedTaskInfo, noColor bool) string {
	var sb strings.Builder

	if len(blockedTasks) == 0 {
		sb.WriteString("\nNo blocked tasks\n")
		return sb.String()
	}

	// Header
	if noColor {
		sb.WriteString("\n=== BLOCKED TASKS ===\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(pterm.DefaultHeader.WithFullWidth().Sprint("BLOCKED TASKS"))
		sb.WriteString("\n")
	}

	// List blocked tasks
	for i, task := range blockedTasks {
		sb.WriteString("\n")

		reason := "No reason provided"
		if task.BlockedReason != nil && *task.BlockedReason != "" {
			reason = *task.BlockedReason
		}

		if noColor {
			sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, task.Key, task.Title))
			sb.WriteString(fmt.Sprintf("   Feature: %s\n", task.Feature))
			sb.WriteString(fmt.Sprintf("   Reason: %s\n", reason))
		} else {
			sb.WriteString(fmt.Sprintf("%d. %s: %s\n",
				i+1,
				pterm.Red(task.Key),
				task.Title))
			sb.WriteString(fmt.Sprintf("   Feature: %s\n", pterm.Gray(task.Feature)))
			sb.WriteString(fmt.Sprintf("   Reason: %s\n", pterm.Yellow(reason)))
		}
	}

	return sb.String()
}

// formatRecentCompletions formats recently completed tasks with relative time
func formatRecentCompletions(completions []*CompletionInfo, noColor bool) string {
	if len(completions) == 0 {
		return ""
	}

	var sb strings.Builder

	// Header
	if noColor {
		sb.WriteString("\n=== RECENT COMPLETIONS ===\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(pterm.DefaultHeader.WithFullWidth().Sprint("RECENT COMPLETIONS"))
		sb.WriteString("\n")
	}

	// List completions
	for i, completion := range completions {
		timeAgo := "recently"
		if completion.CompletedAgo != nil {
			timeAgo = *completion.CompletedAgo
		}

		if noColor {
			sb.WriteString(fmt.Sprintf("\n%d. %s: %s\n", i+1, completion.Key, completion.Title))
			sb.WriteString(fmt.Sprintf("   Feature: %s\n", completion.Feature))
			sb.WriteString(fmt.Sprintf("   Completed: %s\n", timeAgo))
		} else {
			sb.WriteString(fmt.Sprintf("\n%d. %s: %s\n",
				i+1,
				pterm.Green(completion.Key),
				completion.Title))
			sb.WriteString(fmt.Sprintf("   Feature: %s\n", pterm.Gray(completion.Feature)))
			sb.WriteString(fmt.Sprintf("   Completed: %s\n", pterm.Cyan(timeAgo)))
		}
	}

	return sb.String()
}

// formatDurationFromSecs converts a duration in seconds to a human-readable string.
// Durations of 24 hours or more are formatted as "Xd Yh"; shorter durations as "Xh Ym".
func formatDurationFromSecs(secs float64) string {
	d := time.Duration(secs * float64(time.Second))
	hours := int(d.Hours())
	if hours >= 24 {
		days := hours / 24
		remainingHours := hours % 24
		return fmt.Sprintf("%dd %dh", days, remainingHours)
	}
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// formatBugSummary renders the BUGS section of the status dashboard.
// Returns an empty string when bugs is nil (conditional display).
func formatBugSummary(bugs *BugDashboardSummary, noColor bool) string {
	if bugs == nil {
		return ""
	}

	var sb strings.Builder

	// Header
	if noColor {
		sb.WriteString("\n=== BUGS ===\n\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(pterm.DefaultHeader.WithFullWidth().Sprint("BUGS"))
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("Total: %d\n\n", bugs.Total))

	// Status breakdown in defined order
	sb.WriteString("By Status:\n")
	statusOrder := []string{"reported", "triaged", "in_fix", "in_verification", "resolved", "wont_fix", "duplicate"}
	for _, status := range statusOrder {
		if count, ok := bugs.ByStatus[status]; ok && count > 0 {
			sb.WriteString(fmt.Sprintf("  %-18s %d\n", status+":", count))
		}
	}

	// Severity breakdown (open bugs only) -- omit section when all counts are zero
	if len(bugs.OpenBySeverity) > 0 {
		hasOpen := false
		for _, count := range bugs.OpenBySeverity {
			if count > 0 {
				hasOpen = true
				break
			}
		}
		if hasOpen {
			sb.WriteString("\nOpen Bug Severity:\n")
			severityOrder := []string{"critical", "high", "medium", "low"}
			for _, sev := range severityOrder {
				count := bugs.OpenBySeverity[sev]
				sb.WriteString(fmt.Sprintf("  %-18s %d\n", sev+":", count))
			}
		}
	}

	return sb.String()
}

// formatChangeCardSummary renders the CHANGE CARDS section of the status dashboard.
// Returns an empty string when cards is nil (conditional display).
func formatChangeCardSummary(cards *ChangeCardDashboardSummary, noColor bool) string {
	if cards == nil {
		return ""
	}

	var sb strings.Builder

	// Header
	if noColor {
		sb.WriteString("\n=== CHANGE CARDS ===\n\n")
	} else {
		sb.WriteString("\n")
		sb.WriteString(pterm.DefaultHeader.WithFullWidth().Sprint("CHANGE CARDS"))
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("Total: %d\n\n", cards.Total))

	// Status breakdown in defined order
	sb.WriteString("By Status:\n")
	statusOrder := []string{"proposed", "approved", "in_progress", "completed", "declined"}
	for _, status := range statusOrder {
		if count, ok := cards.ByStatus[status]; ok && count > 0 {
			sb.WriteString(fmt.Sprintf("  %-18s %d\n", status+":", count))
		}
	}

	return sb.String()
}

// FormatLinkedBugs formats the linked bug summary for feature-level status output.
// Returns an empty string when bugs is nil (no bugs linked to the feature).
// Terminal output format: "Linked Bugs: N (M open -- sev1: X, sev2: Y)"
func FormatLinkedBugs(bugs *BugFeatureSummary, noColor bool) string {
	if bugs == nil {
		return ""
	}

	// Build severity breakdown string from open bugs only
	// Sort severity names for deterministic output
	severities := make([]string, 0, len(bugs.OpenBySeverity))
	for sev := range bugs.OpenBySeverity {
		severities = append(severities, sev)
	}
	sort.Strings(severities)

	var sevParts []string
	for _, sev := range severities {
		count := bugs.OpenBySeverity[sev]
		if count > 0 {
			sevParts = append(sevParts, fmt.Sprintf("%s: %d", sev, count))
		}
	}

	var openDetail string
	if len(sevParts) > 0 {
		openDetail = fmt.Sprintf("%d open -- %s", bugs.OpenCount, strings.Join(sevParts, ", "))
	} else {
		openDetail = fmt.Sprintf("%d open", bugs.OpenCount)
	}

	line := fmt.Sprintf("Linked Bugs: %d (%s)", bugs.TotalLinked, openDetail)

	if noColor {
		return line
	}
	return pterm.Yellow(line)
}

// FormatDashboard formats the complete dashboard for terminal output
func FormatDashboard(dashboard *StatusDashboard, noColor bool) string {
	var sb strings.Builder

	termWidth := getTerminalWidth()

	// Project summary
	sb.WriteString(formatProjectSummary(dashboard.Summary, noColor))
	sb.WriteString("\n")

	// Epic table
	sb.WriteString(formatEpicTable(dashboard.Epics, noColor, termWidth))
	sb.WriteString("\n")

	// Active tasks
	sb.WriteString(formatActiveTasks(dashboard.ActiveTasks, noColor))
	sb.WriteString("\n")

	// Blocked tasks
	if len(dashboard.BlockedTasks) > 0 {
		sb.WriteString(formatBlockedTasks(dashboard.BlockedTasks, noColor))
		sb.WriteString("\n")
	}

	// Recent completions
	if len(dashboard.RecentCompletions) > 0 {
		sb.WriteString(formatRecentCompletions(dashboard.RecentCompletions, noColor))
		sb.WriteString("\n")
	}

	// Bug summary (conditional: rendered only when BugSummary is non-nil)
	if bugSection := formatBugSummary(dashboard.BugSummary, noColor); bugSection != "" {
		sb.WriteString(bugSection)
	}

	// Change-card summary (conditional: rendered only when ChangeCardSummary is non-nil)
	if ccSection := formatChangeCardSummary(dashboard.ChangeCardSummary, noColor); ccSection != "" {
		sb.WriteString(ccSection)
	}

	return sb.String()
}
