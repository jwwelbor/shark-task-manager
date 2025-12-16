# Task: Add context.Context to Repository Interfaces

**Task ID**: E04-F09-T001
**Feature**: E04-F09 - Recommended Architecture Improvements
**Priority**: P0 (Critical)
**Estimated Effort**: 4 hours
**Status**: Todo

---

## Objective

Add `context.Context` as the first parameter to all repository method signatures to enable request cancellation, timeout management, and distributed tracing support.

## Background

Currently, repository methods don't accept context, which means:
- Cannot cancel long-running database operations
- Cannot set timeouts on queries
- Cannot propagate request context from HTTP handlers
- Missing standard Go idiom for I/O operations

## Acceptance Criteria

- [ ] All repository method signatures include `context.Context` as first parameter
- [ ] Context parameter is consistently named `ctx`
- [ ] Method signatures follow Go convention: `(ctx context.Context, ...other params)`
- [ ] Interface documentation updated with context usage guidelines
- [ ] No functional changes to method behavior (signature only)

## Implementation Details

### Files to Modify

```
internal/repository/
├── epic_repository.go
├── feature_repository.go
├── task_repository.go
├── task_history_repository.go
└── repository.go
```

### Method Signature Changes

#### Example: TaskRepository

**Before**:
```go
func (r *TaskRepository) Create(task *models.Task) error
func (r *TaskRepository) GetByID(id int64) (*models.Task, error)
func (r *TaskRepository) GetByKey(key string) (*models.Task, error)
func (r *TaskRepository) Update(task *models.Task) error
func (r *TaskRepository) Delete(id int64) error
func (r *TaskRepository) ListByFeature(featureID int64) ([]*models.Task, error)
func (r *TaskRepository) UpdateStatus(taskID int64, status models.TaskStatus, agent, notes *string) error
```

**After**:
```go
func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error)
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error)
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error
func (r *TaskRepository) Delete(ctx context.Context, id int64) error
func (r *TaskRepository) ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID int64, status models.TaskStatus, agent, notes *string) error
```

### Complete Method List

#### TaskRepository (17 methods)
- `Create(ctx context.Context, task *models.Task) error`
- `GetByID(ctx context.Context, id int64) (*models.Task, error)`
- `GetByKey(ctx context.Context, key string) (*models.Task, error)`
- `Update(ctx context.Context, task *models.Task) error`
- `Delete(ctx context.Context, id int64) error`
- `ListByFeature(ctx context.Context, featureID int64) ([]*models.Task, error)`
- `ListByEpic(ctx context.Context, epicKey string) ([]*models.Task, error)`
- `FilterByStatus(ctx context.Context, status models.TaskStatus) ([]*models.Task, error)`
- `FilterByAgentType(ctx context.Context, agentType models.AgentType) ([]*models.Task, error)`
- `FilterCombined(ctx context.Context, status *models.TaskStatus, epicKey *string, agentType *models.AgentType, maxPriority *int) ([]*models.Task, error)`
- `List(ctx context.Context) ([]*models.Task, error)`
- `UpdateStatus(ctx context.Context, taskID int64, newStatus models.TaskStatus, agent *string, notes *string) error`
- `BlockTask(ctx context.Context, taskID int64, reason string, agent *string) error`
- `UnblockTask(ctx context.Context, taskID int64, agent *string) error`
- `ReopenTask(ctx context.Context, taskID int64, agent *string, notes *string) error`
- `GetStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)`
- `queryTasks(ctx context.Context, query string, args ...interface{}) ([]*models.Task, error)`

#### EpicRepository (7 methods)
- `Create(ctx context.Context, epic *models.Epic) error`
- `GetByID(ctx context.Context, id int64) (*models.Epic, error)`
- `GetByKey(ctx context.Context, key string) (*models.Epic, error)`
- `List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)`
- `Update(ctx context.Context, epic *models.Epic) error`
- `Delete(ctx context.Context, id int64) error`
- `CalculateProgress(ctx context.Context, epicID int64) (float64, error)`
- `CalculateProgressByKey(ctx context.Context, key string) (float64, error)`

#### FeatureRepository (8 methods)
- `Create(ctx context.Context, feature *models.Feature) error`
- `GetByID(ctx context.Context, id int64) (*models.Feature, error)`
- `GetByKey(ctx context.Context, key string) (*models.Feature, error)`
- `ListByEpic(ctx context.Context, epicID int64) ([]*models.Feature, error)`
- `ListByEpicKey(ctx context.Context, epicKey string) ([]*models.Feature, error)`
- `Update(ctx context.Context, feature *models.Feature) error`
- `Delete(ctx context.Context, id int64) error`
- `CalculateProgress(ctx context.Context, featureID int64) (float64, error)`
- `UpdateProgress(ctx context.Context, featureID int64) error`

#### TaskHistoryRepository (3 methods)
- `Create(ctx context.Context, history *models.TaskHistory) error`
- `GetByTaskID(ctx context.Context, taskID int64) ([]*models.TaskHistory, error)`
- `GetRecentHistory(ctx context.Context, limit int) ([]*models.TaskHistory, error)`

### Implementation Steps

1. **Update method signatures** (mechanical change)
   ```bash
   # Use sed or manual editing to add ctx parameter
   # Example for TaskRepository:
   sed -i 's/func (r \*TaskRepository) Create(/func (r *TaskRepository) Create(ctx context.Context, /g' internal/repository/task_repository.go
   ```

2. **Update internal method calls** (queryTasks, etc.)
   ```go
   // Before
   func (r *TaskRepository) queryTasks(query string, args ...interface{}) ([]*models.Task, error)

   // After
   func (r *TaskRepository) queryTasks(ctx context.Context, query string, args ...interface{}) ([]*models.Task, error)
   ```

3. **Verify compilation** (will fail until callers updated)
   ```bash
   go build ./...  # Expected to fail - callers not updated yet
   ```

4. **Document changes**
   - Add comment to each repository file explaining context usage
   - Update package-level documentation

### Documentation

Add to each repository file:

```go
// Package repository provides data access layer with context support.
//
// All repository methods accept context.Context as the first parameter to support:
// - Request cancellation
// - Timeout management
// - Distributed tracing
// - Request-scoped values
//
// Callers should create contexts appropriately:
// - HTTP handlers: Use r.Context() from http.Request
// - CLI commands: Use context.WithTimeout(context.Background(), timeout)
// - Tests: Use context.Background() or context.WithTimeout()
//
// Example:
//   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//   defer cancel()
//
//   task, err := repo.GetByID(ctx, taskID)
//   if err != nil {
//       return err
//   }
package repository
```

## Testing

**Note**: This task only updates signatures. Tests will be updated in E04-F09-T008.

### Verification Steps

1. **Compile check**: `go build ./internal/repository/...` (will fail with caller errors - expected)
2. **Method count**: Verify all methods updated
   ```bash
   # Should find 0 methods without ctx parameter
   grep -n "func (r \*.*Repository)" internal/repository/*.go | grep -v "context.Context"
   ```

## Dependencies

### Blocks
- E04-F09-T002: Update TaskRepository implementation
- E04-F09-T003: Update EpicRepository implementation
- E04-F09-T004: Update FeatureRepository implementation
- E04-F09-T005: Update TaskHistoryRepository implementation

### Depends On
- None (first task in sequence)

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Missing method updates | Low | Compiler will catch all issues |
| Inconsistent naming | Low | Use `ctx` everywhere consistently |
| Documentation gaps | Medium | Review each file for doc comments |

## Success Criteria

- ✅ All repository methods have `context.Context` as first parameter
- ✅ Context parameter consistently named `ctx`
- ✅ Package documentation updated
- ✅ No compilation errors in repository package itself
- ✅ Ready for implementation updates in T002-T005

## Notes

- This is a **signature-only** change - no implementation changes yet
- Code will not compile until callers are updated (T002-T008)
- This follows Go's standard library pattern (database/sql)
- Context should be named `ctx` not `context` to avoid shadowing package name

## Completion Checklist

- [ ] All TaskRepository methods updated (17 methods)
- [ ] All EpicRepository methods updated (8 methods)
- [ ] All FeatureRepository methods updated (9 methods)
- [ ] All TaskHistoryRepository methods updated (3 methods)
- [ ] Package documentation added
- [ ] Method documentation reviewed
- [ ] Git commit created with message: "Add context parameter to repository method signatures"
