# Test Fixes Needed for Progress Tests

## Summary

Most progress tests are passing (10/14), but 4 tests have issues with the shared database pattern.

## Failing Tests

1. `TestEpicProgress_WeightedAverage` - Epic E97 being created twice somehow
2. `TestEpicProgress_TaskCountWeighting` - Similar issue with E98
3. `TestEpicProgress_EmptyFeatures` - Getting 0% instead of expected 50%
4. `TestEpicProgressPerformance` - Feature E84-F02 foreign key constraint

## Root Cause

The `INSERT OR IGNORE` pattern with `LastInsertId()` has subtle bugs when called multiple times for the same epic within a test. The logs show epic E97 being "created" twice with different IDs (1 and 10), which shouldn't happen with UNIQUE constraints.

## Recommended Fix

Instead of using raw SQL with INSERT OR IGNORE, use the repository methods properly:

```go
func setupProgressTest(t *testing.T, epicNum int, featureNum int, taskStatuses []models.TaskStatus) (int64, int64) {
    database := test.GetTestDB()
    db := NewDB(database)
    epicRepo := NewEpicRepository(db)
    featureRepo := NewFeatureRepository(db)
    taskRepo := NewTaskRepository(db)

    epicKey := fmt.Sprintf("E%02d", epicNum)
    featureKey := fmt.Sprintf("E%02d-F%02d", epicNum, featureNum)

    // Try to get existing epic
    epic, err := epicRepo.GetByKey(epicKey)
    if err != nil {
        // Create new epic if doesn't exist
        epic = &models.Epic{
            Key: epicKey,
            Title: "Progress Test Epic",
            Description: stringPtr("Test epic"),
            Status: models.EpicStatusActive,
            Priority: models.PriorityMedium,
        }
        err = epicRepo.Create(epic)
        if err != nil {
            t.Fatalf("Failed to create epic %s: %v", epicKey, err)
        }
    }

    // Try to get existing feature
    feature, err := featureRepo.GetByKey(featureKey)
    if err != nil {
        // Create new feature if doesn't exist
        feature = &models.Feature{
            EpicID: epic.ID,
            Key: featureKey,
            Title: "Progress Test Feature",
            Description: stringPtr("Test feature"),
            Status: models.FeatureStatusActive,
        }
        err = featureRepo.Create(feature)
        if err != nil {
            t.Fatalf("Failed to create feature %s: %v", featureKey, err)
        }
    }

    // Delete and recreate tasks
    database.Exec("DELETE FROM tasks WHERE feature_id = ?", feature.ID)

    for i, status := range taskStatuses {
        taskKey := fmt.Sprintf("%s-T%03d", featureKey, i+1)
        task := &models.Task{
            FeatureID: feature.ID,
            Key: taskKey,
            Title: fmt.Sprintf("Task %d", i+1),
            Status: status,
            Priority: 1,
            AgentType: agentTypePtr(models.AgentTypeTesting),
            DependsOn: stringPtr("[]"),
        }
        err := taskRepo.Create(task)
        if err != nil {
            t.Fatalf("Failed to create task %s: %v", taskKey, err)
        }
    }

    return epic.ID, feature.ID
}

func stringPtr(s string) *string {
    return &s
}

func agentTypePtr(a models.AgentType) *models.AgentType {
    return &a
}
```

This approach uses repository methods which handle validation and avoid the INSERT OR IGNORE issues.

## Alternative Quick Fix

Run tests sequentially and clean database between test packages:
```bash
find . -name "test-shark-tasks.db*" -delete && go test -p 1 ./...
```
