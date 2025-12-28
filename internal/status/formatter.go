package status

import (
	"fmt"
	"os"
	"sort"
	"strings"

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

	return sb.String()
}
