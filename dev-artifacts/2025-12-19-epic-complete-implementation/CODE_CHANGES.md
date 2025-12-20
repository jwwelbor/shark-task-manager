# Code Changes for Epic Complete Implementation

## File Modified
`/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/epic.go`

## Summary of Changes

The `runEpicComplete` function was completely rewritten to implement comprehensive epic completion functionality with detailed warnings, per-feature breakdowns, and proper JSON output.

## Function: runEpicComplete (Lines 806-1066)

### Key Enhancements

#### 1. Per-Feature Tracking (Lines 859-886)
Added maps to track status breakdown per feature:
```go
featureTaskBreakdown := make(map[string]map[models.TaskStatus]int)
featureTaskCounts := make(map[string]int)
```

This enables detailed per-feature breakdown in warnings.

#### 2. Detailed Warning Output (Lines 902-995)
When incomplete tasks exist and `--force` is not used:

**Overall Status Breakdown** (Lines 908-921):
- Shows total task count
- Displays breakdown: "X todo, Y in_progress, Z blocked, W ready_for_review"

**Per-Feature Breakdown** (Lines 924-953):
- For each feature, shows:
  - Total tasks in feature
  - Count of incomplete tasks
  - Status breakdown for incomplete tasks
  - Different output for all-completed features

**Problematic Tasks** (Lines 956-990):
- Lists blocked tasks first (lines 960-965)
- Then lists other incomplete tasks (lines 967-972)
- Limited to 15 tasks maximum (lines 975-977)
- For blocked tasks, includes the blocking reason (lines 981-986)

**Exit Code**: Changed from 1 to 3 (line 994)
- 3 indicates "invalid state" per HTTP conventions

#### 3. Enhanced Task Completion (Lines 997-1025)
- Tracks affected tasks: `var affectedTaskKeys []string`
- Only updates non-completed tasks
- Collects task keys for JSON output (line 1015)
- Updates progress for all features (lines 1019-1024)

#### 4. Improved JSON Output (Lines 1029-1047)
Added required fields:
- `epic_key`: Epic key being completed
- `feature_count`: Number of features in epic
- `total_task_count`: Total tasks across all features
- `completed_count`: Final completion count
- `status_breakdown`: Object with counts by status
  ```go
  statusBreakdownMap := make(map[string]int)
  statusBreakdownMap["todo"] = totalStatusBreakdown[models.TaskStatusTodo]
  statusBreakdownMap["in_progress"] = ...
  statusBreakdownMap["blocked"] = ...
  statusBreakdownMap["ready_for_review"] = ...
  statusBreakdownMap["completed"] = completedCount + completedTaskCount
  ```
- `affected_tasks`: List of task keys completed
- `force_completed`: Boolean indicating if force was used

#### 5. Enhanced Human-Readable Output (Lines 1050-1063)
Different messages based on completion mode:
- **Force mode**: Shows force completion with status breakdown
- **Normal mode**: Shows completion with task counts and features

## Key Implementation Details

### Data Structures
- **totalStatusBreakdown**: `map[models.TaskStatus]int` - overall status counts
- **featureTaskBreakdown**: `map[string]map[models.TaskStatus]int` - per-feature status
- **featureTaskCounts**: `map[string]int` - task counts per feature
- **affectedTaskKeys**: `[]string` - tasks completed in this operation
- **problematicTasks**: `[]*models.Task` - blocked and incomplete tasks

### Status Classification
Tasks are classified as:
- **Done**: `TaskStatusCompleted` + `TaskStatusReadyForReview`
- **Incomplete**: All other statuses (todo, in_progress, blocked, archived)
- **Most Problematic**: Blocked tasks, then other incomplete tasks

### Error Handling
- Database error: `os.Exit(2)`
- Epic not found: `os.Exit(1)`
- Invalid state (incomplete without force): `os.Exit(3)`
- Success: `return nil`

### Progress Update
- Feature progress updated via `featureRepo.UpdateProgress()`
- Epic progress calculated on-demand (not stored, so no explicit update needed)

## Integration Points

### Used Methods
- `epicRepo.GetByKey()` - Get epic by key
- `featureRepo.ListByEpic()` - Get all features in epic
- `taskRepo.ListByFeature()` - Get all tasks in a feature
- `taskRepo.GetStatusBreakdown()` - Get status counts per feature
- `taskRepo.UpdateStatusForced()` - Update task status to completed
- `featureRepo.UpdateProgress()` - Update feature progress percentage

### Used Functions
- `getAgentIdentifier()` - Get current user/agent identifier
- `cli.GlobalConfig.JSON` - Check if JSON output requested
- `cli.OutputJSON()` - Output JSON with proper formatting
- `cli.Success()`, `cli.Warning()`, `cli.Info()`, `cli.Error()` - Output formatting

### Used Constants
- `models.TaskStatusCompleted` - Completed status
- `models.TaskStatusReadyForReview` - Ready for review status
- `models.TaskStatusTodo` - Todo status
- `models.TaskStatusInProgress` - In progress status
- `models.TaskStatusBlocked` - Blocked status

## Command Registration

No changes to command registration were needed. The command was already:
- Declared (line 84-98)
- Registered in init() (line 155)
- Flags added in init() (line 164)

## Output Examples

### Complete Example: Incomplete Tasks Warning
```
Warning: Cannot complete epic with incomplete tasks

Total tasks: 15
Status breakdown: 3 todo, 2 in_progress, 1 blocked, 9 ready_for_review

Feature breakdown:
  E07-F08: 5 tasks (1 incomplete) (1 todo, 1 in_progress)
  E07-F09: 4 tasks (3 incomplete) (2 todo, 1 blocked)
  E07-F10: 6 tasks (all ready_for_review)

Most problematic tasks:
  - T-E07-F09-001 (blocked) - Depends on external resource
  - T-E07-F09-002 (blocked) - Awaiting review
  - T-E07-F08-001 (todo)
  - T-E07-F08-002 (in_progress)

Use --force to complete all tasks regardless of status
```

### Force Complete Example
```
Epic E07 completed: Force-completed 5 tasks (1 todo, 1 in_progress, 1 blocked, 2 ready_for_review)
```

### All Completed Example
```
Epic E07 completed: 15/15 tasks completed across 3 features
```

## Testing Recommendations

1. **Scenario: No Features**
   - Epic with no features should exit gracefully

2. **Scenario: No Tasks**
   - Epic with features but no tasks should exit gracefully

3. **Scenario: All Completed**
   - Epic with all tasks completed should show success immediately

4. **Scenario: Mixed Status Without Force**
   - Should show detailed warning and exit with code 3

5. **Scenario: Mixed Status With Force**
   - Should complete all tasks and show summary

6. **Scenario: JSON Output**
   - Should include all required fields in JSON

7. **Scenario: Blocked Tasks**
   - Should list blocked tasks first
   - Should include blocking reasons when available

8. **Scenario: Large Task Count**
   - Should list up to 15 most problematic tasks
   - Should properly handle sorting

## Compatibility Notes

- No breaking changes to existing commands
- All changes are backward compatible
- Uses existing repository methods
- Follows existing code patterns and style
- Works with existing database schema
