# Task: E05-F01-T04 - Implement Rich Table Output Formatting

**Feature**: E05-F01 Status Dashboard & Reporting
**Epic**: E05 Task Management CLI Capabilities
**Task Key**: E05-F01-T04

## Description

Implement visual output formatting for the status dashboard using pterm library for rich tables, progress bars, and color coding. This task transforms the structured StatusDashboard data into attractive, readable terminal output with proper formatting for human consumption.

The formatter displays:
- Project Summary section with statistics
- Epic Breakdown table with ASCII progress bars
- Active Tasks section grouped by agent type
- Blocked Tasks section with reasons
- Recent Completions section with relative time

All output respects `--no-color` flag and terminal width constraints.

**Why This Matters**: Good formatting makes data accessible. Users need to quickly scan progress bars, identify blockers, and understand project health at a glance. Poor formatting defeats the dashboard's purpose.

## What You'll Build

Rich output formatter functions in `internal/cli/commands/status.go`:

```go
func outputRichTable(dashboard *StatusDashboard) error {
    outputProjectSummary(dashboard.Summary)
    outputEpicTable(dashboard.Epics)
    outputActiveTasks(dashboard.ActiveTasks)
    outputBlockedTasks(dashboard.BlockedTasks)
    outputRecentCompletions(dashboard.RecentCompletions)
    return nil
}

func outputProjectSummary(summary *ProjectSummary) { ... }
func outputEpicTable(epics []*EpicSummary) { ... }
func outputActiveTasks(tasks map[string][]*TaskInfo) { ... }
func outputBlockedTasks(tasks []*BlockedTaskInfo) { ... }
func outputRecentCompletions(tasks []*CompletionInfo) { ... }

// Helpers
func renderProgressBar(progress float64, health string) string { ... }
func getHealthColor(health string) string { ... }
func colorize(text string, color string) string { ... }
func truncateTitle(title string, maxWidth int) string { ... }
func getTerminalWidth() int { ... }
```

## Success Criteria

- [x] Project Summary section displays correct format with statistics
- [x] Epic Breakdown table with columns: Epic | Title | Progress | Tasks | Status
- [x] Progress bars are 20 characters wide: `[##########----------] 60.0%`
- [x] Epic health color coding: green (healthy), yellow (warning), red (critical)
- [x] Active Tasks grouped by agent type with counts in headers
- [x] Agent types in canonical order: frontend, backend, api, testing, devops, general, unassigned
- [x] Blocked Tasks displayed with blocking reasons
- [x] Recent Completions shown with relative time ("2 hours ago")
- [x] Empty sections show helpful messages: "No blocked tasks" (green)
- [x] Terminal width detection and title truncation for narrow terminals
- [x] `--no-color` flag strips all ANSI codes
- [x] Output fits in 80-column terminals with wrapping
- [x] All sections have clear headers and visual separation

## Implementation Notes

### Output Layout Example

```
PROJECT SUMMARY
===============
Epics: 5 (3 active, 2 completed)
Features: 23 (15 active, 8 completed)
Tasks: 127 (45 todo, 12 in_progress, 5 ready_for_review, 60 completed, 5 blocked)
Overall Progress: 47.3%
Blocked: 5 tasks


EPIC BREAKDOWN
==============
Epic   Title                        Progress                  Tasks    Status
─────────────────────────────────────────────────────────────────────────────
E01    Identity Platform            [############--------] 60%  30/50    active
E02    Task Management CLI          [########------------] 40%  20/50    active
E03    Documentation System         [####################] 100% 10/10    completed


ACTIVE TASKS (12)
=================
Frontend (3):
  • T-E01-F02-005: Build user profile component
  • T-E01-F02-007: Implement responsive navigation
  • T-E02-F01-003: Create task list UI

Backend (5):
  • T-E01-F01-002: Implement JWT validation
  • T-E01-F03-001: Build API authentication layer
  ...

BLOCKED TASKS (5)
=================
• T-E01-F02-003: User authentication flow
  Reason: Waiting for API specification from backend team

• T-E02-F01-007: Task dependency validation
  Reason: Missing dependency graph algorithm implementation

RECENT COMPLETIONS (Last 24 hours)
===================================
• T-E01-F01-003: JWT token generation - 2 hours ago
• T-E01-F02-001: Login form component - 5 hours ago
• T-E02-F01-001: Database connection setup - 18 hours ago
```

### Progress Bar Rendering

```go
func renderProgressBar(progress float64, health string) string {
    // Clamp progress to 0-100
    progress = math.Min(100.0, math.Max(0.0, progress))

    // Calculate filled characters (20-char bar)
    filled := int(progress / 5.0)  // 20 chars = 5% per char
    empty := 20 - filled

    // Build bar
    bar := "[" + strings.Repeat("#", filled) + strings.Repeat("-", empty) + "]"

    // Add color if not --no-color
    if !cli.GlobalConfig.NoColor {
        color := getHealthColor(health)
        return fmt.Sprintf("%s %s%5.1f%%[reset]", color, bar, progress)
    }

    return fmt.Sprintf("%s %5.1f%%", bar, progress)
}
```

**Examples**:
- 60% progress (healthy): `[############--------] 60.0%` (green)
- 40% progress (warning): `[########------------] 40.0%` (yellow)
- 20% progress (critical): `[####----------------] 20.0%` (red)
- 100% progress (healthy): `[####################] 100.0%` (green)
- 0% progress (critical): `[--------------------]  0.0%` (red)

### Color Coding

Map health status to pterm color codes:

```go
func getHealthColor(health string) string {
    switch health {
    case "healthy":
        return "[green]"
    case "warning":
        return "[yellow]"
    case "critical":
        return "[red]"
    default:
        return "[white]"
    }
}

func colorize(text string, color string) string {
    if cli.GlobalConfig.NoColor {
        return text
    }
    return fmt.Sprintf("%s%s[reset]", color, text)
}
```

**Color Usage**:
- Green: Completed tasks, healthy epics (≥75%)
- Yellow: In-progress tasks, warning epics (25-74%)
- Red: Blocked tasks, critical epics (<25% or >3 blocked)
- Blue: Ready-for-review tasks
- Gray: Todo tasks

### Terminal Width Handling

```go
func getTerminalWidth() int {
    width, _, err := term.GetSize(int(os.Stdout.Fd()))
    if err != nil || width < 80 {
        return 80  // Default to 80 columns
    }
    return width
}

func truncateTitle(title string, maxWidth int) string {
    if len(title) <= maxWidth {
        return title
    }
    return title[:maxWidth-3] + "..."
}
```

**Usage in table rendering**:
```go
width := getTerminalWidth()
titleMaxWidth := width - 20  // Account for other columns

truncated := truncateTitle(epic.Title, titleMaxWidth)
// Use truncated in table cell
```

### Section Formatters

#### Project Summary

```go
func outputProjectSummary(summary *ProjectSummary) {
    fmt.Println("\nPROJECT SUMMARY")
    fmt.Println(strings.Repeat("=", 50))
    fmt.Printf("Epics: %d (%d active, %d completed)\n",
        summary.Epics.Total, summary.Epics.Active, summary.Epics.Completed)
    fmt.Printf("Features: %d (%d active, %d completed)\n",
        summary.Features.Total, summary.Features.Active, summary.Features.Completed)
    fmt.Printf("Tasks: %d (%d todo, %d in_progress, %d ready_for_review, %d completed, %d blocked)\n",
        summary.Tasks.Total, summary.Tasks.Todo, summary.Tasks.InProgress,
        summary.Tasks.ReadyForReview, summary.Tasks.Completed, summary.Tasks.Blocked)
    fmt.Printf("Overall Progress: %.1f%%\n", summary.OverallProgress)
    fmt.Printf("Blocked: %d tasks\n", summary.BlockedCount)
}
```

#### Epic Breakdown Table

Use pterm library for nice tables:

```go
func outputEpicTable(epics []*EpicSummary) {
    if len(epics) == 0 {
        fmt.Println("\n" + colorize("No epics found", "[yellow]"))
        return
    }

    fmt.Println("\n\nEPIC BREAKDOWN")
    fmt.Println(strings.Repeat("=", 80))

    // Create table with pterm
    tableData := [][]string{
        {"Epic", "Title", "Progress", "Tasks", "Status"},
    }

    for _, epic := range epics {
        progressBar := renderProgressBar(epic.Progress, epic.Health)
        tasksStr := fmt.Sprintf("%d/%d", epic.TasksCompleted, epic.TasksTotal)

        title := truncateTitle(epic.Title, 25)
        tableData = append(tableData, []string{
            epic.Key,
            title,
            progressBar,
            tasksStr,
            epic.Status,
        })
    }

    table := pterm.TableData(tableData)
    pterm.DefaultTable.WithData(table).Render()
}
```

#### Active Tasks by Agent

```go
func outputActiveTasks(tasks map[string][]*TaskInfo) {
    count := 0
    for _, agents := range tasks {
        count += len(agents)
    }

    fmt.Printf("\n\nACTIVE TASKS (%d)\n", count)
    fmt.Println(strings.Repeat("=", 50))

    if count == 0 {
        fmt.Println(colorize("No tasks currently in progress", "[green]"))
        return
    }

    // Use canonical agent order
    for _, agent := range status.AgentTypesOrder {
        taskList, exists := tasks[agent]
        if !exists || len(taskList) == 0 {
            continue
        }

        fmt.Printf("\n%s (%d):\n", capitalizeAgent(agent), len(taskList))
        for _, task := range taskList {
            fmt.Printf("  • %s: %s\n", task.Key, task.Title)
        }
    }
}

func capitalizeAgent(agent string) string {
    if agent == "unassigned" {
        return "Unassigned"
    }
    return strings.ToUpper(agent[:1]) + agent[1:]
}
```

#### Blocked Tasks with Reasons

```go
func outputBlockedTasks(tasks []*BlockedTaskInfo) {
    fmt.Printf("\n\nBLOCKED TASKS (%d)\n", len(tasks))
    fmt.Println(strings.Repeat("=", 50))

    if len(tasks) == 0 {
        fmt.Println(colorize("No blocked tasks", "[green]"))
        return
    }

    for _, task := range tasks {
        fmt.Printf("%s• %s: %s\n", colorize("", "[red]"), task.Key, task.Title)

        reason := "No reason provided"
        if task.BlockedReason != nil {
            reason = *task.BlockedReason
        }
        fmt.Printf("  %sReason: %s[reset]\n", colorize("", "[red]"), reason)
        fmt.Println()
    }
}
```

#### Recent Completions

```go
func outputRecentCompletions(tasks []*CompletionInfo) {
    fmt.Printf("\n\nRECENT COMPLETIONS (Last 24 hours)\n")
    fmt.Println(strings.Repeat("=", 60))

    if len(tasks) == 0 {
        fmt.Println(colorize("No tasks completed in last 24 hours", "[gray]"))
        return
    }

    for i, task := range tasks {
        if i >= 20 {  // Limit display to 20 items
            fmt.Printf("  ... and %d more\n", len(tasks)-i)
            break
        }
        fmt.Printf("%s• %s: %s - %s[reset]\n",
            colorize("", "[green]"), task.Key, task.Title, task.CompletedAgo)
    }
}
```

### No-Color Mode

Always check `cli.GlobalConfig.NoColor` before applying colors:

```go
func colorize(text string, color string) string {
    if cli.GlobalConfig.NoColor {
        return text  // Strip color codes
    }
    return fmt.Sprintf("%s%s[reset]", color, text)
}
```

This ensures `--no-color` produces clean, ANSI-code-free output.

## Dependencies

- pterm: Terminal rendering library (already in project)
- Go standard library: fmt, strings, math, os, golang.org/x/term
- Internal: `internal/cli` for GlobalConfig.NoColor
- `internal/status` for data structures

## Related Tasks

- **E05-F01-T01**: Service Data Structures - Defines StatusDashboard
- **E05-F01-T03**: CLI Command - Calls outputRichTable

## Acceptance Criteria

**Functional**:
- [ ] Project Summary displays correct format
- [ ] Epic table with proper column alignment
- [ ] Progress bars render correctly: 20 characters wide
- [ ] Progress percentages accurate to 0.1%
- [ ] Active tasks grouped by agent_type
- [ ] Agent types in canonical order
- [ ] Blocked tasks show reasons
- [ ] Recent completions show relative time
- [ ] Empty sections show helpful messages
- [ ] Color coding follows spec: green/yellow/red for health

**Formatting**:
- [ ] Output readable in 80-column terminal
- [ ] Long titles truncated with "..."
- [ ] Progress bars aligned with proper spacing
- [ ] Section headers clear and consistent
- [ ] Proper spacing between sections

**Flags**:
- [ ] `--no-color` removes all ANSI codes
- [ ] Color shows by default (not --no-color)
- [ ] JSON output not affected by color flag

**Edge Cases**:
- [ ] Empty project shows "No epics found"
- [ ] No active tasks shows "No tasks currently in progress"
- [ ] No blocked tasks shows "No blocked tasks" in green
- [ ] No recent completions shows appropriate message
- [ ] Very narrow terminal (<80 cols) handled gracefully

**Testing**:
- [ ] Unit test: Progress bar rendering
- [ ] Unit test: Color application
- [ ] Unit test: Title truncation
- [ ] Integration test: Full output formatting
- [ ] All tests pass: `go test ./internal/cli/commands -run Output -v`

## Verification Steps

```bash
# Test full output
./bin/shark status

# Test with colors disabled
./bin/shark status --no-color | grep -v "\\[" | head -30
# Should have no ANSI codes

# Test progress bar rendering
go test ./internal/cli/commands -run TestProgressBar -v

# Verify output in narrow terminal
# (Manually test with resize, or mock terminal width)

# Test with large dataset
./bin/shark status | less
# Should be properly formatted and scrollable
```

## Implementation Checklist

See Phase 4 in implementation-checklist.md:
- [ ] Task 4.1: Rich Table Formatter Setup
- [ ] Task 4.2: Project Summary Section
- [ ] Task 4.3: Epic Breakdown Table
- [ ] Task 4.4: Active Tasks Section
- [ ] Task 4.5: Blocked Tasks Section
- [ ] Task 4.6: Recent Completions Section
- [ ] Task 4.7: Color Coding Implementation
- [ ] Task 4.8: Terminal Width Handling
