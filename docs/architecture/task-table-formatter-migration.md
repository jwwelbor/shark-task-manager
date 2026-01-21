# Task Table Formatter - Migration Guide

This guide shows step-by-step how to migrate existing task rendering code to use the centralized `formatters.TaskTable` functionality.

## Migration Examples

### Example 1: Migrate `shark task list` Command

**File**: `internal/cli/commands/task.go`

#### Before (Lines 489-545)

```go
// Human-readable table output
if len(tasks) == 0 {
    cli.Info("No tasks found")
    return nil
}

// Get project root for WorkflowService
projectRoot, err := os.Getwd()
if err != nil {
    projectRoot = ""
}
workflowService := workflow.NewService(projectRoot)

headers := []string{"Key", "Title", "Status", "Priority", "Agent Type", "Order"}
rows := [][]string{}
for _, task := range tasks {
    agentTypeStr := "-"
    if task.AgentType != nil {
        agentTypeStr = string(*task.AgentType)
    }

    // Truncate title if too long
    title := task.Title
    if len(title) > 40 {
        title = title[:37] + "..."
    }

    // Format execution_order (show "-" if NULL)
    execOrder := "-"
    if task.ExecutionOrder != nil {
        execOrder = fmt.Sprintf("%d", *task.ExecutionOrder)
    }

    // Add rejection indicator to key if task has rejections
    keyDisplay := task.Key
    if task.RejectionCount > 0 {
        keyDisplay = task.Key + " " + formatRejectionIndicator(task.RejectionCount)
    }

    // Apply color coding to status using workflow service
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

#### After (5 Lines)

```go
// Human-readable table output
if len(tasks) == 0 {
    cli.Info("No tasks found")
    return nil
}

// Get project root for WorkflowService
projectRoot, err := os.Getwd()
if err != nil {
    projectRoot = ""
}
workflowService := workflow.NewService(projectRoot)

// Use centralized task table formatter
config := formatters.DefaultTaskTableConfig()
config.ColorEnabled = !cli.GlobalConfig.NoColor

if err := formatters.RenderTaskTable(tasks, workflowService, config); err != nil {
    return fmt.Errorf("failed to render task table: %w", err)
}
```

#### Changes Summary
- **55 lines → 9 lines** (83% reduction)
- Removed all manual formatting logic
- Removed manual iteration over tasks
- Removed duplicate title truncation logic
- Removed duplicate status color formatting
- Removed duplicate agent type formatting
- Removed duplicate execution order formatting
- Removed duplicate rejection indicator logic

### Example 2: Migrate `shark feature get` Task Table

**File**: `internal/cli/commands/feature.go`

#### Before (Lines 1100-1131)

```go
for _, task := range tasks {
    // Widen title column to 60 characters for better readability
    title := task.Title
    if len(title) > 60 {
        title = title[:57] + "..."
    }

    // Get agent type
    agent := "none"
    if task.AgentType != nil {
        agent = string(*task.AgentType)
    }

    // Format task status with color if available
    taskStatusDisplay := string(task.Status)
    if colorEnabled {
        formatted := workflowService.FormatStatusForDisplay(string(task.Status), true)
        taskStatusDisplay = formatted.Colored
    }

    tableData = append(tableData, []string{
        task.Key,
        title,
        taskStatusDisplay,
        fmt.Sprintf("%d", task.Priority),
        agent,
    })
}

// Render tasks table
_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
```

#### After (5 Lines)

```go
// Use centralized task table formatter with feature get config
config := formatters.FeatureGetTaskTableConfig()
config.ColorEnabled = colorEnabled

if err := formatters.RenderTaskTable(tasks, workflowService, config); err != nil {
    return fmt.Errorf("failed to render task table: %w", err)
}
```

#### Changes Summary
- **32 lines → 5 lines** (84% reduction)
- Removed manual iteration over tasks
- Removed duplicate title truncation (60 chars)
- Removed duplicate agent type formatting
- Removed duplicate status color formatting
- Used predefined `FeatureGetTaskTableConfig()` for consistency

## Step-by-Step Migration Process

### Step 1: Import the Formatter Package

Add to imports:
```go
import (
    // ... existing imports ...
    "github.com/jwwelbor/shark-task-manager/internal/formatters"
)
```

### Step 2: Identify Existing Task Rendering Code

Look for patterns like:
- Loops iterating over `[]*models.Task`
- Manual title truncation: `if len(title) > X { title = title[:X-3] + "..." }`
- Status color formatting: `workflowService.FormatStatusForDisplay(...)`
- Agent type formatting: `if task.AgentType != nil { ... }`
- Execution order formatting: `if task.ExecutionOrder != nil { ... }`
- Table rendering: `cli.OutputTable(...)` or `pterm.DefaultTable.WithData(...).Render()`

### Step 3: Choose Configuration

**Use DefaultTaskTableConfig() if:**
- Command is a list/query command
- Needs all standard columns (Key, Title, Status, Priority, Agent Type, Order)
- Title truncation at 40 characters is acceptable
- Using `cli.OutputTable` for rendering

**Use FeatureGetTaskTableConfig() if:**
- Command is a detail/get command
- Showing tasks within a feature context
- Need wider title column (60 characters)
- Don't need execution order or rejection indicators
- Using `pterm.DefaultTable` for rendering

**Create custom config if:**
- Need specific column combination
- Need custom title truncation length
- Need different rendering method

### Step 4: Replace Rendering Code

Replace the entire rendering block with:

```go
// Get workflow service if not already created
workflowService := workflow.NewService(projectRoot)

// Choose config
config := formatters.DefaultTaskTableConfig() // or FeatureGetTaskTableConfig()
config.ColorEnabled = !cli.GlobalConfig.NoColor // respect global flag

// Render
if err := formatters.RenderTaskTable(tasks, workflowService, config); err != nil {
    return fmt.Errorf("failed to render task table: %w", err)
}
```

### Step 5: Remove Helper Functions

If the command has helper functions that are now obsolete, remove them:

```go
// DELETE: formatRejectionIndicator() if only used for task rendering
// DELETE: any custom title truncation functions
// DELETE: any custom agent type formatting functions
```

These are now handled internally by the formatter.

### Step 6: Test

Run the command and verify:
- ✅ Table renders correctly
- ✅ Colors work when not using `--no-color`
- ✅ Title truncation works at expected length
- ✅ All columns display correctly
- ✅ Null handling works (agent type, execution order)
- ✅ Rejection indicators show (if applicable)

```bash
# Test basic rendering
./bin/shark task list

# Test color disabled
./bin/shark task list --no-color

# Test JSON output (should not use formatter)
./bin/shark task list --json

# Test with filters
./bin/shark task list --status=todo
./bin/shark task list E07 F01
```

## Common Migration Patterns

### Pattern 1: Conditional Rendering (List vs Empty)

**Before:**
```go
if len(tasks) == 0 {
    cli.Info("No tasks found")
    return nil
}

// ... 50 lines of rendering code ...
```

**After:**
```go
if len(tasks) == 0 {
    cli.Info("No tasks found")
    return nil
}

// Render with formatter
config := formatters.DefaultTaskTableConfig()
config.ColorEnabled = !cli.GlobalConfig.NoColor
formatters.RenderTaskTable(tasks, workflowService, config)
```

### Pattern 2: Custom Column Selection

**Before:**
```go
// Show only key, title, status
for _, task := range tasks {
    rows = append(rows, []string{
        task.Key,
        task.Title,
        task.Status,
    })
}
```

**After:**
```go
config := formatters.DefaultTaskTableConfig()
config.ShowPriority = false
config.ShowAgentType = false
config.ShowExecutionOrder = false
config.ColorEnabled = !cli.GlobalConfig.NoColor
formatters.RenderTaskTable(tasks, workflowService, config)
```

### Pattern 3: Custom Title Length

**Before:**
```go
title := task.Title
if len(title) > 50 {
    title = title[:47] + "..."
}
```

**After:**
```go
config := formatters.DefaultTaskTableConfig()
config.TitleMaxLength = 50
config.ColorEnabled = !cli.GlobalConfig.NoColor
formatters.RenderTaskTable(tasks, workflowService, config)
```

### Pattern 4: Different Table Renderer

**Before (using pterm directly):**
```go
tableData := [][]string{headers}
for _, task := range tasks {
    // ... format row ...
    tableData = append(tableData, row)
}
_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
```

**After:**
```go
config := formatters.DefaultTaskTableConfig()
config.UsePterm = true // Use pterm instead of cli.OutputTable
config.ColorEnabled = !cli.GlobalConfig.NoColor
formatters.RenderTaskTable(tasks, workflowService, config)
```

## Benefits Checklist

After migration, you should have:

- ✅ Significantly fewer lines of code
- ✅ Consistent task rendering across commands
- ✅ Single point of maintenance for task table logic
- ✅ Automatic color formatting based on workflow config
- ✅ Consistent title truncation
- ✅ Consistent null handling (agent type, execution order)
- ✅ Consistent rejection indicators
- ✅ Easy to add new columns in future
- ✅ Easy to change formatting rules globally
- ✅ Better testability (formatters have comprehensive tests)

## Commands to Migrate

### Priority 1 (Duplicate Logic)
- ✅ `shark task list` - Already analyzed, ready to migrate
- ✅ `shark feature get` - Already analyzed, ready to migrate

### Priority 2 (Similar Rendering)
- [ ] `shark epic get` - May have task table rendering
- [ ] `shark task next` - May show task details
- [ ] Any custom commands that render task lists

### Priority 3 (Future)
- [ ] Consider creating similar formatters for:
  - Feature tables (`FeatureTableFormatter`)
  - Epic tables (`EpicTableFormatter`)
  - Task history tables (`TaskHistoryTableFormatter`)

## Testing After Migration

### Unit Tests

Ensure existing command tests still pass:
```bash
go test -v ./internal/cli/commands -run TestTask
go test -v ./internal/cli/commands -run TestFeature
```

### Integration Tests

Test actual command execution:
```bash
# Build
make build

# Test task list
./bin/shark task list
./bin/shark task list --no-color
./bin/shark task list --json
./bin/shark task list E07 F01

# Test feature get
./bin/shark feature get E07-F01
./bin/shark feature get E07-F01 --no-color
./bin/shark feature get E07-F01 --json
```

### Visual Verification

Compare before/after screenshots:
1. Take screenshot before migration
2. Migrate code
3. Take screenshot after migration
4. Verify they look identical (or intentionally improved)

## Rollback Plan

If issues arise:

1. **Git revert**: Revert the migration commit
   ```bash
   git revert <commit-hash>
   ```

2. **Selective rollback**: Keep formatter but revert command changes
   ```bash
   git checkout HEAD~1 -- internal/cli/commands/task.go
   ```

3. **Report issues**: File bug report with:
   - What command was migrated
   - What broke
   - Screenshots of before/after
   - Steps to reproduce

## Questions & Answers

**Q: What if I need a column that's not in TaskTableConfig?**

A: Add it to `TaskTableConfig` struct, update `buildHeaders()` and `formatTaskRow()`, add tests.

**Q: What if I need custom formatting for a specific command?**

A: Create a custom config or extend `TaskTableConfig` with new options.

**Q: Should I remove `formatRejectionIndicator()` from task.go?**

A: Only if it's not used elsewhere. The formatter has its own implementation.

**Q: What about JSON output?**

A: JSON output bypasses the formatter. Keep existing `cli.OutputJSON(tasks)` logic.

**Q: Can I mix formatter with custom rendering?**

A: Yes. Use `FormatTaskTable()` to get headers/rows, then render yourself:
```go
result := formatters.FormatTaskTable(tasks, workflowService, config)
// Custom rendering with result.Headers and result.Rows
```

## Support

For migration help:
- Review `docs/architecture/task-table-formatter.md`
- Check test examples in `internal/formatters/task_table_test.go`
- Look at successful migrations in this guide
- Ask in project discussions or issues
