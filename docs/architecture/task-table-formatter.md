# Task Table Formatter

## Overview

The `internal/formatters` package provides a centralized, reusable function for rendering task lists across all CLI commands. This eliminates duplicate task rendering logic and ensures consistent formatting throughout the application.

## Location

**Package**: `internal/formatters`
**File**: `internal/formatters/task_table.go`

## Problem Solved

Previously, task list rendering was duplicated across multiple commands:
1. `shark task list` (internal/cli/commands/task.go:489-543)
2. `shark feature get` (internal/cli/commands/feature.go:1100-1131)

This led to:
- ~60 lines of duplicate code per usage
- Inconsistent title truncation (40 chars vs 60 chars)
- Inconsistent formatting
- Difficulty maintaining color coding logic

## Solution

A single, configurable function that handles all task table rendering:

```go
func FormatTaskTable(
    tasks []*models.Task,
    workflowService *workflow.Service,
    config TaskTableConfig,
) TaskTableResult
```

## Configuration

### TaskTableConfig

Provides fine-grained control over table rendering:

```go
type TaskTableConfig struct {
    // Column visibility
    ShowKey            bool
    ShowTitle          bool
    ShowStatus         bool
    ShowPriority       bool
    ShowAgentType      bool
    ShowExecutionOrder bool
    ShowRejections     bool

    // Formatting options
    TitleMaxLength int
    ColorEnabled   bool
    UseHeader      bool

    // Table renderer
    UsePterm bool // If true, uses pterm.DefaultTable
}
```

### Predefined Configurations

Two built-in configurations match existing command behaviors:

**DefaultTaskTableConfig** - Used by `shark task list`:
```go
config := formatters.DefaultTaskTableConfig()
// Shows: Key, Title, Status, Priority, Agent Type, Order
// Title truncated at 40 chars
// Uses cli.OutputTable
```

**FeatureGetTaskTableConfig** - Used by `shark feature get`:
```go
config := formatters.FeatureGetTaskTableConfig()
// Shows: Key, Title, Status, Priority, Agent Type
// Title truncated at 60 chars (wider for better readability)
// Uses pterm.DefaultTable
```

## Usage Examples

### Basic Usage

```go
import (
    "github.com/jwwelbor/shark-task-manager/internal/formatters"
    "github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// Get tasks from repository
tasks, err := taskRepo.List(ctx, filters)
if err != nil {
    return err
}

// Create workflow service for color formatting
workflowService := workflow.NewService(projectRoot)

// Render with default config
config := formatters.DefaultTaskTableConfig()
config.ColorEnabled = !cli.GlobalConfig.NoColor

err = formatters.RenderTaskTable(tasks, workflowService, config)
```

### Custom Configuration

```go
// Create custom config for specific view
config := formatters.TaskTableConfig{
    ShowKey:            true,
    ShowTitle:          true,
    ShowStatus:         true,
    ShowPriority:       false,  // Hide priority
    ShowAgentType:      false,  // Hide agent type
    ShowExecutionOrder: false,  // Hide execution order
    ShowRejections:     true,
    TitleMaxLength:     50,     // Custom truncation
    ColorEnabled:       true,
    UseHeader:          true,
    UsePterm:           false,
}

err = formatters.RenderTaskTable(tasks, workflowService, config)
```

### Advanced: Get Formatted Data Without Rendering

```go
// Format without rendering (useful for custom output)
config := formatters.DefaultTaskTableConfig()
result := formatters.FormatTaskTable(tasks, workflowService, config)

// result.Headers: []string{"Key", "Title", "Status", "Priority", "Agent Type", "Order"}
// result.Rows: [][]string{{"E07-F01-001", "Task Title", "todo", "5", "backend", "1"}}

// Now you can process the data or render it yourself
for _, row := range result.Rows {
    fmt.Println(strings.Join(row, " | "))
}
```

## Features

### 1. Title Truncation

Automatically truncates long titles based on `TitleMaxLength`:

```go
config := formatters.DefaultTaskTableConfig()
config.TitleMaxLength = 40

// "This is a very long task title that should be truncated to fit"
// becomes: "This is a very long task title that s..."
```

### 2. Status Color Formatting

Integrates with workflow service for status color coding:

```go
config := formatters.DefaultTaskTableConfig()
config.ColorEnabled = true

// Status column will be colored based on .sharkconfig.json status_metadata
// Example: "todo" -> gray, "in_progress" -> yellow, "completed" -> green
```

### 3. Rejection Indicators

Shows rejection count when tasks have been rejected in review:

```go
config := formatters.DefaultTaskTableConfig()
config.ShowRejections = true

// Key column: "E07-F01-001 🔴×3" (task rejected 3 times)
```

### 4. Null Handling

Gracefully handles nullable fields:

- `AgentType` is `nil` → displays "none"
- `ExecutionOrder` is `nil` → displays "-"

### 5. Flexible Rendering

Choose between two rendering methods:

**cli.OutputTable** (default):
```go
config.UsePterm = false
formatters.RenderTaskTable(tasks, workflowService, config)
// Uses pterm's table rendering under the hood via cli.OutputTable
```

**pterm.DefaultTable** (feature get):
```go
config.UsePterm = true
formatters.RenderTaskTable(tasks, workflowService, config)
// Directly uses pterm for rendering
```

## Migration Guide

### Before (task.go)

```go
// 55 lines of duplicate code
headers := []string{"Key", "Title", "Status", "Priority", "Agent Type", "Order"}
rows := [][]string{}
for _, task := range tasks {
    agentTypeStr := "-"
    if task.AgentType != nil {
        agentTypeStr = string(*task.AgentType)
    }

    title := task.Title
    if len(title) > 40 {
        title = title[:37] + "..."
    }

    execOrder := "-"
    if task.ExecutionOrder != nil {
        execOrder = fmt.Sprintf("%d", *task.ExecutionOrder)
    }

    keyDisplay := task.Key
    if task.RejectionCount > 0 {
        keyDisplay = task.Key + " " + formatRejectionIndicator(task.RejectionCount)
    }

    statusDisplay := string(task.Status)
    if !cli.GlobalConfig.NoColor {
        formatted := workflowService.FormatStatusForDisplay(string(task.Status), true)
        statusDisplay = formatted.Colored
    }

    rows = append(rows, []string{
        keyDisplay,
        title,
        statusDisplay,
        fmt.Sprintf("%d", task.Priority),
        agentTypeStr,
        execOrder,
    })
}

cli.OutputTable(headers, rows)
```

### After (task.go)

```go
// 5 lines - cleaner and reusable
workflowService := workflow.NewService(projectRoot)
config := formatters.DefaultTaskTableConfig()
config.ColorEnabled = !cli.GlobalConfig.NoColor

formatters.RenderTaskTable(tasks, workflowService, config)
```

**Benefits:**
- **55 lines → 5 lines** (90% reduction)
- Consistent formatting across commands
- Single point of maintenance
- Easy to add new columns or change formatting

## Testing

Comprehensive test suite at `internal/formatters/task_table_test.go`:

- ✅ Empty task lists
- ✅ Single and multiple tasks
- ✅ Title truncation at various lengths
- ✅ Rejection indicators
- ✅ Agent type (set and nil)
- ✅ Execution order (set and nil)
- ✅ Column visibility toggling
- ✅ Color formatting
- ✅ Header construction

Run tests:
```bash
go test -v ./internal/formatters -run TestFormatTaskTable
```

## Architecture Alignment

This implementation follows Shark's architecture patterns:

1. **Centralized Formatters** - Matches existing `internal/formatters` package pattern
2. **Workflow Integration** - Uses `workflow.Service` for status formatting
3. **Configuration-Driven** - Flexible `TaskTableConfig` for different views
4. **Testability** - Comprehensive test coverage with mocked dependencies
5. **Single Responsibility** - Formatters only format, commands handle business logic

## Future Enhancements

Potential additions:
- [ ] CSV export format
- [ ] Markdown table format
- [ ] JSON structured output
- [ ] Custom column ordering
- [ ] Sort by column
- [ ] Pagination support
- [ ] Grouping by epic/feature/status

## Related Files

- `internal/formatters/task_table.go` - Main implementation
- `internal/formatters/task_table_test.go` - Test suite
- `internal/workflow/service.go` - Status color formatting
- `internal/cli/commands/task.go` - Task list command (consumer)
- `internal/cli/commands/feature.go` - Feature get command (consumer)

## API Reference

### Functions

**FormatTaskTable**
```go
func FormatTaskTable(
    tasks []*models.Task,
    workflowService *workflow.Service,
    config TaskTableConfig,
) TaskTableResult
```
Formats tasks into table headers and rows. Returns `TaskTableResult` for custom rendering.

**RenderTaskTable**
```go
func RenderTaskTable(
    tasks []*models.Task,
    workflowService *workflow.Service,
    config TaskTableConfig,
) error
```
Formats and renders tasks using the configured renderer (cli.OutputTable or pterm).

**DefaultTaskTableConfig**
```go
func DefaultTaskTableConfig() TaskTableConfig
```
Returns configuration matching `shark task list` behavior.

**FeatureGetTaskTableConfig**
```go
func FeatureGetTaskTableConfig() TaskTableConfig
```
Returns configuration matching `shark feature get` behavior.

### Types

**TaskTableConfig**
```go
type TaskTableConfig struct {
    ShowKey, ShowTitle, ShowStatus, ShowPriority,
    ShowAgentType, ShowExecutionOrder, ShowRejections bool
    TitleMaxLength int
    ColorEnabled, UseHeader, UsePterm bool
}
```

**TaskTableResult**
```go
type TaskTableResult struct {
    Headers []string
    Rows    [][]string
}
```
