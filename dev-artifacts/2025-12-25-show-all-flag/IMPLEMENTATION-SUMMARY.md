# --show-all Flag Implementation Summary

**Date**: 2025-12-25
**Implemented By**: Claude Code (TDD Approach)
**Feature**: Hide completed tasks by default, add `--show-all` flag

## Overview

Implemented a new default behavior for `shark task list` that hides completed tasks unless explicitly requested. This reduces clutter and helps users focus on active work.

## Feature Description

### New Default Behavior
- `shark task list` now **hides completed tasks** by default
- Shows: `todo`, `in_progress`, `ready_for_review`, `blocked`
- Hides: `completed`

### New Flag
- `--show-all` - Include completed tasks in the output

### Filter Precedence
- **Explicit status filter** overrides default hiding
  - `--status=completed` shows only completed tasks
  - `--status=todo` shows only todo tasks
- **--show-all flag** includes all tasks
- **Default** (no flags) hides completed tasks

## Test-Driven Development Process

### 1. ✅ Tests Written First

Created `internal/cli/commands/task_list_filter_test.go` with comprehensive test coverage:

**Test Scenarios:**
- Default behavior hides completed tasks
- `--show-all` flag includes completed tasks
- Explicit `--status=completed` shows completed tasks
- Empty task list handling
- Mixed status task lists
- Status filter precedence rules

**Test Results:**
```
=== RUN   TestTaskListFiltering_HideCompletedByDefault
--- PASS: TestTaskListFiltering_HideCompletedByDefault (8 sub-tests)

=== RUN   TestTaskListFiltering_StatusFilterPrecedence
--- PASS: TestTaskListFiltering_StatusFilterPrecedence (5 sub-tests)
```

### 2. ✅ Implementation

**Files Modified:**

1. **`internal/cli/commands/task.go`**
   - Lines 48-65: Updated command help text with new default behavior
   - Lines 319-321: Added filtering call in `runTaskList()`
   - Lines 666-688: Implemented `filterTasksByCompletedStatus()` function
   - Line 1239: Added `--show-all` flag definition

2. **`docs/CLI_REFERENCE.md`**
   - Lines 543-590: Updated `shark task list` documentation
   - Added default behavior explanation
   - Added `--show-all` flag to flags list
   - Updated examples with new behavior

### 3. ✅ All Tests Pass

```bash
✅ All filtering tests pass (13/13 sub-tests)
✅ All command tests pass
✅ Full test suite passes
✅ Linter passes
✅ Build succeeds
```

## Implementation Details

### Core Filtering Function

```go
func filterTasksByCompletedStatus(tasks []*models.Task, showAll bool, statusFilter string) []*models.Task {
    // If an explicit status filter is set, don't apply default filtering
    if statusFilter != "" {
        return tasks
    }

    // If showAll is true, return all tasks
    if showAll {
        return tasks
    }

    // Default behavior: filter out completed tasks
    filtered := make([]*models.Task, 0, len(tasks))
    for _, task := range tasks {
        if task.Status != models.TaskStatusCompleted {
            filtered = append(filtered, task)
        }
    }
    return filtered
}
```

### Integration Point

```go
// In runTaskList() function, after all other filters:
showAll, _ := cmd.Flags().GetBool("show-all")
tasks = filterTasksByCompletedStatus(tasks, showAll, statusStr)
```

## Usage Examples

### Default Behavior (Hide Completed)
```bash
$ shark task list
Key                  Title                  Status           Priority
T-E01-F01-001       Build login form       todo             5
T-E01-F01-002       Add validation        in_progress       3
T-E01-F01-004       Write tests           ready_for_review  4
# T-E01-F01-003 (completed) is hidden
```

### Show All Tasks
```bash
$ shark task list --show-all
Key                  Title                  Status           Priority
T-E01-F01-001       Build login form       todo             5
T-E01-F01-002       Add validation        in_progress       3
T-E01-F01-003       Deploy to prod        completed         2
T-E01-F01-004       Write tests           ready_for_review  4
```

### Explicit Status Filter
```bash
# Show only completed tasks
$ shark task list --status=completed
Key                  Title                  Status           Priority
T-E01-F01-003       Deploy to prod        completed         2

# Show only todo tasks
$ shark task list --status=todo
Key                  Title                  Status           Priority
T-E01-F01-001       Build login form       todo             5
```

### Combined with Other Filters
```bash
# Filter by epic, still hide completed
$ shark task list --epic=E04
# Shows non-completed tasks in E04

# Filter by epic, show all
$ shark task list --epic=E04 --show-all
# Shows ALL tasks in E04 including completed
```

## Design Decisions

### Why Hide Completed by Default?

1. **Focus on Active Work**: Users primarily care about what needs to be done
2. **Reduce Clutter**: Long-running projects accumulate many completed tasks
3. **Better UX**: Most task management tools hide completed items by default
4. **Easy Override**: `--show-all` provides quick access when needed

### Why Explicit Status Filter Takes Precedence?

When a user explicitly asks for `--status=completed`, they want to see completed tasks. This is more specific than the general "hide completed" rule, so it should win.

**Precedence Order:**
1. Explicit `--status` filter (highest)
2. `--show-all` flag
3. Default behavior (lowest)

### Why Not Filter at Repository Level?

The filtering happens **after** repository queries to:
1. Keep repository logic simple and focused on data access
2. Allow flexibility for future enhancements (e.g., `--hide-archived`)
3. Make the filter easy to test in isolation
4. Preserve existing repository API compatibility

## Backward Compatibility

### Breaking Change Warning

⚠️ **This is a breaking change** for users who rely on completed tasks appearing in default output.

**Migration Path:**
- Scripts/automation that need completed tasks: Add `--show-all` flag
- Users who want old behavior: Use `--show-all` or `--status=completed`

### JSON Output

The `--show-all` flag works with JSON output:

```bash
# Default: no completed tasks
shark task list --json

# Include completed tasks
shark task list --show-all --json
```

## Testing

### Unit Tests
- 13 test cases covering all scenarios
- Tests for filter precedence
- Tests for edge cases (empty lists, all completed, etc.)

### Manual Testing
```bash
# Build and test
make build
./bin/shark task list
./bin/shark task list --show-all
./bin/shark task list --status=completed
./bin/shark task list --help
```

### Regression Testing
- All existing tests pass
- No changes to other commands
- Filtering logic isolated in dedicated function

## Documentation Updates

### CLI Help Text
```bash
$ shark task list --help
List tasks with optional filtering by status, epic, feature, or agent.

By default, completed tasks are hidden. Use --show-all to include them.
...
      --show-all           Show all tasks including completed
```

### CLI Reference
Updated `docs/CLI_REFERENCE.md` with:
- Default behavior explanation
- `--show-all` flag documentation
- Updated examples showing both behaviors
- Precedence rules documented

## Quality Metrics

- **Test Coverage**: 100% of filtering logic covered
- **Tests Written First**: ✅ TDD approach followed
- **Backward Compatibility**: ⚠️ Breaking change (documented)
- **Documentation**: ✅ Complete
- **Linter**: ✅ Passes
- **Build**: ✅ Successful

## Files Changed

```
internal/cli/commands/task.go                    (+23 lines, modified)
internal/cli/commands/task_list_filter_test.go   (+211 lines, new file)
docs/CLI_REFERENCE.md                            (+31 lines, modified)
```

## Future Enhancements

Potential related features:
1. `--hide-archived` flag to hide archived tasks
2. `--completed-since <date>` to show recently completed tasks
3. `--hide <status>` to hide specific statuses
4. `--show-only <statuses>` for multi-status filtering

---

## Conclusion

The `--show-all` flag implementation successfully:
- ✅ Follows test-driven development
- ✅ Maintains code quality (linter, tests)
- ✅ Documents behavior completely
- ✅ Provides sensible defaults
- ✅ Allows easy override
- ✅ Preserves existing functionality

The new default behavior makes `shark task list` more useful for day-to-day work by focusing attention on active tasks.
