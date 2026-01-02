# Test Plan: Cascading Status Calculation

## Feature: E07-F14 - Automatic Status Calculation

**Date:** 2026-01-01
**Author:** Architect Agent

---

## 1. Testing Strategy Overview

Following the project's testing architecture from CLAUDE.md:

| Test Type | Database | Location | Purpose |
|-----------|----------|----------|---------|
| Unit Tests | Mocked | `*_test.go` | Business logic, derivation algorithms |
| Repository Tests | Real DB | `*_test.go` in `repository/` | Database operations, migrations |
| Integration Tests | Real DB | `*_integration_test.go` | End-to-end cascading behavior |

---

## 2. Unit Tests (Mocked)

### 2.1 Status Derivation Logic

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/status/derivation_test.go`

#### Feature Status Derivation

```go
func TestDeriveFeatureStatus(t *testing.T) {
    tests := []struct {
        name     string
        counts   map[models.TaskStatus]int
        expected models.FeatureStatus
    }{
        {
            name: "all_completed_returns_completed",
            counts: map[models.TaskStatus]int{
                models.TaskStatusCompleted: 5,
            },
            expected: models.FeatureStatusCompleted,
        },
        {
            name: "all_archived_returns_completed",
            counts: map[models.TaskStatus]int{
                models.TaskStatusArchived: 3,
            },
            expected: models.FeatureStatusCompleted,
        },
        {
            name: "mixed_completed_archived_returns_completed",
            counts: map[models.TaskStatus]int{
                models.TaskStatusCompleted: 3,
                models.TaskStatusArchived:  2,
            },
            expected: models.FeatureStatusCompleted,
        },
        {
            name: "any_in_progress_returns_active",
            counts: map[models.TaskStatus]int{
                models.TaskStatusTodo:       2,
                models.TaskStatusInProgress: 1,
                models.TaskStatusCompleted:  3,
            },
            expected: models.FeatureStatusActive,
        },
        {
            name: "any_ready_for_review_returns_active",
            counts: map[models.TaskStatus]int{
                models.TaskStatusTodo:           2,
                models.TaskStatusReadyForReview: 1,
            },
            expected: models.FeatureStatusActive,
        },
        {
            name: "only_blocked_returns_active",
            counts: map[models.TaskStatus]int{
                models.TaskStatusBlocked: 2,
            },
            expected: models.FeatureStatusActive,
        },
        {
            name: "blocked_with_todo_returns_active",
            counts: map[models.TaskStatus]int{
                models.TaskStatusTodo:    3,
                models.TaskStatusBlocked: 1,
            },
            expected: models.FeatureStatusActive,
        },
        {
            name: "some_completed_some_todo_returns_active",
            counts: map[models.TaskStatus]int{
                models.TaskStatusTodo:      2,
                models.TaskStatusCompleted: 3,
            },
            expected: models.FeatureStatusActive,
        },
        {
            name: "all_todo_returns_draft",
            counts: map[models.TaskStatus]int{
                models.TaskStatusTodo: 5,
            },
            expected: models.FeatureStatusDraft,
        },
        {
            name: "empty_returns_empty_string",
            counts: map[models.TaskStatus]int{},
            expected: "",  // Indicates no change
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := deriveFeatureStatus(tt.counts)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### Epic Status Derivation

```go
func TestDeriveEpicStatus(t *testing.T) {
    tests := []struct {
        name     string
        counts   map[models.FeatureStatus]int
        expected models.EpicStatus
    }{
        {
            name: "all_completed_returns_completed",
            counts: map[models.FeatureStatus]int{
                models.FeatureStatusCompleted: 3,
            },
            expected: models.EpicStatusCompleted,
        },
        {
            name: "all_archived_returns_completed",
            counts: map[models.FeatureStatus]int{
                models.FeatureStatusArchived: 2,
            },
            expected: models.EpicStatusCompleted,
        },
        {
            name: "any_active_returns_active",
            counts: map[models.FeatureStatus]int{
                models.FeatureStatusDraft:     1,
                models.FeatureStatusActive:    1,
                models.FeatureStatusCompleted: 1,
            },
            expected: models.EpicStatusActive,
        },
        {
            name: "all_draft_returns_draft",
            counts: map[models.FeatureStatus]int{
                models.FeatureStatusDraft: 4,
            },
            expected: models.EpicStatusDraft,
        },
        {
            name: "mixed_draft_completed_returns_active",
            counts: map[models.FeatureStatus]int{
                models.FeatureStatusDraft:     2,
                models.FeatureStatusCompleted: 2,
            },
            expected: models.EpicStatusActive,
        },
        {
            name: "empty_returns_empty_string",
            counts: map[models.FeatureStatus]int{},
            expected: "",  // Indicates no change
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := deriveEpicStatus(tt.counts)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 2.2 Service Logic (Mocked Repositories)

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/status/calculation_test.go`

```go
// Mock interfaces
type MockFeatureRepository struct {
    GetByIDFunc             func(ctx context.Context, id int64) (*models.Feature, error)
    GetTaskStatusBreakdownFunc func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
    GetStatusOverrideFunc   func(ctx context.Context, featureID int64) (bool, error)
    UpdateStatusIfNotOverriddenFunc func(ctx context.Context, featureID int64, status models.FeatureStatus) (bool, error)
}

type MockEpicRepository struct {
    GetByIDFunc             func(ctx context.Context, id int64) (*models.Epic, error)
    GetFeatureStatusBreakdownFunc func(ctx context.Context, epicID int64) (map[models.FeatureStatus]int, error)
    UpdateStatusDerivedFunc func(ctx context.Context, epicID int64, status models.EpicStatus) error
}

type MockTaskRepository struct {
    GetByIDFunc func(ctx context.Context, id int64) (*models.Task, error)
}

func TestStatusCalculationService_ApplyFeatureStatus(t *testing.T) {
    tests := []struct {
        name            string
        featureID       int64
        currentStatus   models.FeatureStatus
        statusOverride  bool
        taskCounts      map[models.TaskStatus]int
        expectUpdate    bool
        expectedStatus  models.FeatureStatus
        expectError     bool
    }{
        {
            name:           "updates_status_when_not_overridden",
            featureID:      1,
            currentStatus:  models.FeatureStatusDraft,
            statusOverride: false,
            taskCounts: map[models.TaskStatus]int{
                models.TaskStatusInProgress: 1,
            },
            expectUpdate:   true,
            expectedStatus: models.FeatureStatusActive,
        },
        {
            name:           "skips_update_when_overridden",
            featureID:      1,
            currentStatus:  models.FeatureStatusDraft,
            statusOverride: true,
            taskCounts: map[models.TaskStatus]int{
                models.TaskStatusCompleted: 5,
            },
            expectUpdate:   false,
            expectedStatus: "",  // Not updated
        },
        {
            name:           "no_change_when_status_matches",
            featureID:      1,
            currentStatus:  models.FeatureStatusActive,
            statusOverride: false,
            taskCounts: map[models.TaskStatus]int{
                models.TaskStatusInProgress: 1,
            },
            expectUpdate:   false,  // Already active
            expectedStatus: models.FeatureStatusActive,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks
            featureRepo := &MockFeatureRepository{
                GetByIDFunc: func(ctx context.Context, id int64) (*models.Feature, error) {
                    return &models.Feature{
                        ID:             tt.featureID,
                        Status:         tt.currentStatus,
                        StatusOverride: tt.statusOverride,
                    }, nil
                },
                GetTaskStatusBreakdownFunc: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
                    return tt.taskCounts, nil
                },
                GetStatusOverrideFunc: func(ctx context.Context, featureID int64) (bool, error) {
                    return tt.statusOverride, nil
                },
                UpdateStatusIfNotOverriddenFunc: func(ctx context.Context, featureID int64, status models.FeatureStatus) (bool, error) {
                    if tt.statusOverride {
                        return false, nil
                    }
                    return true, nil
                },
            }

            svc := NewStatusCalculationService(nil, nil, featureRepo, nil)
            result, err := svc.ApplyFeatureStatus(context.Background(), tt.featureID)

            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectUpdate, result.WasChanged)
                if tt.expectUpdate {
                    assert.Equal(t, string(tt.expectedStatus), result.NewStatus)
                }
            }
        })
    }
}

func TestStatusCalculationService_PropagateTaskChange(t *testing.T) {
    // Test cascading from task -> feature -> epic
    t.Run("cascades_through_hierarchy", func(t *testing.T) {
        // Setup: Task belongs to Feature belongs to Epic
        task := &models.Task{ID: 1, FeatureID: 10}
        feature := &models.Feature{ID: 10, EpicID: 100, Status: models.FeatureStatusDraft}
        epic := &models.Epic{ID: 100, Status: models.EpicStatusDraft}

        taskCounts := map[models.TaskStatus]int{
            models.TaskStatusInProgress: 1,
            models.TaskStatusTodo:       2,
        }
        featureCounts := map[models.FeatureStatus]int{
            models.FeatureStatusActive: 1,  // After update
        }

        taskRepo := &MockTaskRepository{
            GetByIDFunc: func(ctx context.Context, id int64) (*models.Task, error) {
                return task, nil
            },
        }
        featureRepo := &MockFeatureRepository{
            GetByIDFunc: func(ctx context.Context, id int64) (*models.Feature, error) {
                return feature, nil
            },
            GetTaskStatusBreakdownFunc: func(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error) {
                return taskCounts, nil
            },
            GetStatusOverrideFunc: func(ctx context.Context, featureID int64) (bool, error) {
                return false, nil
            },
            UpdateStatusIfNotOverriddenFunc: func(ctx context.Context, featureID int64, status models.FeatureStatus) (bool, error) {
                return true, nil
            },
        }
        epicRepo := &MockEpicRepository{
            GetByIDFunc: func(ctx context.Context, id int64) (*models.Epic, error) {
                return epic, nil
            },
            GetFeatureStatusBreakdownFunc: func(ctx context.Context, epicID int64) (map[models.FeatureStatus]int, error) {
                return featureCounts, nil
            },
            UpdateStatusDerivedFunc: func(ctx context.Context, epicID int64, status models.EpicStatus) error {
                return nil
            },
        }

        svc := NewStatusCalculationService(nil, epicRepo, featureRepo, taskRepo)
        results, err := svc.PropagateTaskChange(context.Background(), task.ID)

        assert.NoError(t, err)
        assert.Len(t, results, 2)  // Feature + Epic updated
        assert.Equal(t, "feature", results[0].EntityType)
        assert.Equal(t, "epic", results[1].EntityType)
    })
}
```

---

## 3. Repository Tests (Real Database)

### 3.1 Feature Repository Tests

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/repository/feature_status_test.go`

```go
func TestFeatureRepository_StatusOverride(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    featureRepo := NewFeatureRepository(db)
    epicRepo := NewEpicRepository(db)

    // Cleanup before test
    _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E99-%'")
    _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E99'")

    // Create test epic and feature
    epic := &models.Epic{Key: "E99", Title: "Test Epic", Status: models.EpicStatusActive, Priority: models.PriorityMedium}
    err := epicRepo.Create(ctx, epic)
    require.NoError(t, err)

    feature := &models.Feature{
        EpicID: epic.ID,
        Key:    "E99-F01",
        Title:  "Test Feature",
        Status: models.FeatureStatusDraft,
    }
    err = featureRepo.Create(ctx, feature)
    require.NoError(t, err)

    t.Run("default_override_is_false", func(t *testing.T) {
        override, err := featureRepo.GetStatusOverride(ctx, feature.ID)
        assert.NoError(t, err)
        assert.False(t, override)
    })

    t.Run("set_override_to_true", func(t *testing.T) {
        err := featureRepo.SetStatusOverride(ctx, feature.ID, true)
        assert.NoError(t, err)

        override, err := featureRepo.GetStatusOverride(ctx, feature.ID)
        assert.NoError(t, err)
        assert.True(t, override)
    })

    t.Run("update_skipped_when_override_true", func(t *testing.T) {
        // Override is true from previous test
        updated, err := featureRepo.UpdateStatusIfNotOverridden(ctx, feature.ID, models.FeatureStatusActive)
        assert.NoError(t, err)
        assert.False(t, updated)

        // Verify status unchanged
        f, err := featureRepo.GetByID(ctx, feature.ID)
        assert.NoError(t, err)
        assert.Equal(t, models.FeatureStatusDraft, f.Status)
    })

    t.Run("update_applied_when_override_false", func(t *testing.T) {
        // Clear override
        err := featureRepo.SetStatusOverride(ctx, feature.ID, false)
        require.NoError(t, err)

        updated, err := featureRepo.UpdateStatusIfNotOverridden(ctx, feature.ID, models.FeatureStatusActive)
        assert.NoError(t, err)
        assert.True(t, updated)

        // Verify status changed
        f, err := featureRepo.GetByID(ctx, feature.ID)
        assert.NoError(t, err)
        assert.Equal(t, models.FeatureStatusActive, f.Status)
    })

    // Cleanup
    _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
    _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

func TestFeatureRepository_GetTaskStatusBreakdown(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    featureRepo := NewFeatureRepository(db)
    taskRepo := NewTaskRepository(db)

    // Cleanup and seed
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E99-%'")
    epicID, featureID := test.SeedTestData()  // Creates E99, E99-F99

    // Create tasks with different statuses
    tasks := []*models.Task{
        {FeatureID: featureID, Key: "T-E99-F99-001", Title: "Task 1", Status: models.TaskStatusTodo, Priority: 5},
        {FeatureID: featureID, Key: "T-E99-F99-002", Title: "Task 2", Status: models.TaskStatusTodo, Priority: 5},
        {FeatureID: featureID, Key: "T-E99-F99-003", Title: "Task 3", Status: models.TaskStatusInProgress, Priority: 5},
        {FeatureID: featureID, Key: "T-E99-F99-004", Title: "Task 4", Status: models.TaskStatusCompleted, Priority: 5},
        {FeatureID: featureID, Key: "T-E99-F99-005", Title: "Task 5", Status: models.TaskStatusCompleted, Priority: 5},
    }

    for _, task := range tasks {
        err := taskRepo.Create(ctx, task)
        require.NoError(t, err)
    }

    breakdown, err := featureRepo.GetTaskStatusBreakdown(ctx, featureID)
    assert.NoError(t, err)
    assert.Equal(t, 2, breakdown[models.TaskStatusTodo])
    assert.Equal(t, 1, breakdown[models.TaskStatusInProgress])
    assert.Equal(t, 2, breakdown[models.TaskStatusCompleted])
    assert.Equal(t, 0, breakdown[models.TaskStatusBlocked])

    // Cleanup
    for _, task := range tasks {
        _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
    }
}
```

### 3.2 Migration Test

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/db/migration_status_override_test.go`

```go
func TestMigration_StatusOverride(t *testing.T) {
    // Create temporary database
    tmpDB, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    defer tmpDB.Close()

    // Create schema WITHOUT status_override column
    _, err = tmpDB.Exec(`
        CREATE TABLE features (
            id INTEGER PRIMARY KEY,
            epic_id INTEGER NOT NULL,
            key TEXT NOT NULL UNIQUE,
            title TEXT NOT NULL,
            status TEXT NOT NULL
        )
    `)
    require.NoError(t, err)

    // Insert test data before migration
    _, err = tmpDB.Exec(`INSERT INTO features (id, epic_id, key, title, status) VALUES (1, 1, 'E01-F01', 'Test', 'draft')`)
    require.NoError(t, err)

    // Run migration
    err = migrateStatusOverride(tmpDB)
    assert.NoError(t, err)

    // Verify column exists
    var columnExists int
    err = tmpDB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'status_override'`).Scan(&columnExists)
    assert.NoError(t, err)
    assert.Equal(t, 1, columnExists)

    // Verify existing data has default value
    var override bool
    err = tmpDB.QueryRow(`SELECT status_override FROM features WHERE id = 1`).Scan(&override)
    assert.NoError(t, err)
    assert.False(t, override)

    // Verify migration is idempotent
    err = migrateStatusOverride(tmpDB)
    assert.NoError(t, err)
}
```

---

## 4. Integration Tests

### 4.1 End-to-End Cascading Test

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/status/calculation_integration_test.go`

```go
func TestStatusCalculation_EndToEnd_Cascading(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    database := test.GetTestDB()
    db := repository.NewDB(database)

    // Initialize repositories
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)

    // Initialize service
    calcService := NewStatusCalculationService(db, epicRepo, featureRepo, taskRepo)

    // Cleanup
    _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'T-E98-%'")
    _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'E98-%'")
    _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key = 'E98'")

    // Create hierarchy: Epic -> Feature -> Tasks
    epic := &models.Epic{Key: "E98", Title: "Integration Test Epic", Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
    err := epicRepo.Create(ctx, epic)
    require.NoError(t, err)

    feature := &models.Feature{EpicID: epic.ID, Key: "E98-F01", Title: "Integration Test Feature", Status: models.FeatureStatusDraft}
    err = featureRepo.Create(ctx, feature)
    require.NoError(t, err)

    tasks := []*models.Task{
        {FeatureID: feature.ID, Key: "T-E98-F01-001", Title: "Task 1", Status: models.TaskStatusTodo, Priority: 5},
        {FeatureID: feature.ID, Key: "T-E98-F01-002", Title: "Task 2", Status: models.TaskStatusTodo, Priority: 5},
        {FeatureID: feature.ID, Key: "T-E98-F01-003", Title: "Task 3", Status: models.TaskStatusTodo, Priority: 5},
    }
    for _, task := range tasks {
        err := taskRepo.Create(ctx, task)
        require.NoError(t, err)
    }

    t.Run("initial_state_is_draft", func(t *testing.T) {
        // Trigger initial calculation
        results, err := calcService.PropagateTaskChange(ctx, tasks[0].ID)
        require.NoError(t, err)

        // Feature should be draft (all tasks todo)
        f, _ := featureRepo.GetByID(ctx, feature.ID)
        assert.Equal(t, models.FeatureStatusDraft, f.Status)

        // Epic should be draft (all features draft)
        e, _ := epicRepo.GetByID(ctx, epic.ID)
        assert.Equal(t, models.EpicStatusDraft, e.Status)
    })

    t.Run("starting_task_activates_feature_and_epic", func(t *testing.T) {
        // Start a task
        err := taskRepo.UpdateStatus(ctx, tasks[0].ID, models.TaskStatusInProgress, nil, nil)
        require.NoError(t, err)

        // Propagate change
        results, err := calcService.PropagateTaskChange(ctx, tasks[0].ID)
        require.NoError(t, err)

        // Feature should now be active
        f, _ := featureRepo.GetByID(ctx, feature.ID)
        assert.Equal(t, models.FeatureStatusActive, f.Status)

        // Epic should now be active
        e, _ := epicRepo.GetByID(ctx, epic.ID)
        assert.Equal(t, models.EpicStatusActive, e.Status)

        // Verify results contain both updates
        assert.Len(t, results, 2)
    })

    t.Run("completing_all_tasks_completes_feature", func(t *testing.T) {
        // Complete all tasks
        for _, task := range tasks {
            // First move to in_progress if not already
            currentTask, _ := taskRepo.GetByID(ctx, task.ID)
            if currentTask.Status == models.TaskStatusTodo {
                taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, nil, nil)
            }
            taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusCompleted, nil, nil)
        }

        // Propagate final change
        results, err := calcService.PropagateTaskChange(ctx, tasks[2].ID)
        require.NoError(t, err)

        // Feature should be completed
        f, _ := featureRepo.GetByID(ctx, feature.ID)
        assert.Equal(t, models.FeatureStatusCompleted, f.Status)

        // Epic should be completed (only feature is completed)
        e, _ := epicRepo.GetByID(ctx, epic.ID)
        assert.Equal(t, models.EpicStatusCompleted, e.Status)
    })

    // Cleanup
    for _, task := range tasks {
        _, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)
    }
    _, _ = database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)
    _, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)
}

func TestStatusCalculation_Override_Prevents_Update(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    database := test.GetTestDB()
    db := repository.NewDB(database)

    // Initialize repositories
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)
    calcService := NewStatusCalculationService(db, epicRepo, featureRepo, taskRepo)

    // Create hierarchy
    epic := &models.Epic{Key: "E97", Title: "Override Test Epic", Status: models.EpicStatusDraft, Priority: models.PriorityMedium}
    err := epicRepo.Create(ctx, epic)
    require.NoError(t, err)
    defer database.ExecContext(ctx, "DELETE FROM epics WHERE id = ?", epic.ID)

    feature := &models.Feature{EpicID: epic.ID, Key: "E97-F01", Title: "Override Test Feature", Status: models.FeatureStatusDraft}
    err = featureRepo.Create(ctx, feature)
    require.NoError(t, err)
    defer database.ExecContext(ctx, "DELETE FROM features WHERE id = ?", feature.ID)

    // Set override
    err = featureRepo.SetStatusOverride(ctx, feature.ID, true)
    require.NoError(t, err)

    // Create and start task
    task := &models.Task{FeatureID: feature.ID, Key: "T-E97-F01-001", Title: "Task 1", Status: models.TaskStatusTodo, Priority: 5}
    err = taskRepo.Create(ctx, task)
    require.NoError(t, err)
    defer database.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", task.ID)

    err = taskRepo.UpdateStatus(ctx, task.ID, models.TaskStatusInProgress, nil, nil)
    require.NoError(t, err)

    // Propagate - feature should NOT change due to override
    results, err := calcService.PropagateTaskChange(ctx, task.ID)
    require.NoError(t, err)

    // Feature should still be draft
    f, _ := featureRepo.GetByID(ctx, feature.ID)
    assert.Equal(t, models.FeatureStatusDraft, f.Status)

    // Result should indicate skip
    assert.True(t, results[0].WasSkipped)
    assert.Equal(t, "status_override enabled", results[0].SkipReason)
}
```

---

## 5. CLI Tests (Mocked)

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature_status_test.go`

```go
func TestFeatureUpdateCommand_AutoStatus(t *testing.T) {
    // Mock setup
    mockFeatureRepo := &MockFeatureRepository{
        GetByKeyFunc: func(ctx context.Context, key string) (*models.Feature, error) {
            return &models.Feature{
                ID:             14,
                Key:            "E07-F14",
                Status:         models.FeatureStatusDraft,
                StatusOverride: true,
            }, nil
        },
        SetStatusOverrideFunc: func(ctx context.Context, id int64, override bool) error {
            return nil
        },
    }

    mockCalcService := &MockStatusCalculationService{
        ApplyFeatureStatusFunc: func(ctx context.Context, featureID int64) (*StatusChangeResult, error) {
            return &StatusChangeResult{
                EntityType:     "feature",
                EntityKey:      "E07-F14",
                PreviousStatus: "draft",
                NewStatus:      "active",
                WasChanged:     true,
            }, nil
        },
    }

    // Execute command
    output := executeFeatureUpdate("E07-F14", "--auto-status", "--json")

    // Verify output
    var result map[string]interface{}
    err := json.Unmarshal([]byte(output), &result)
    assert.NoError(t, err)
    assert.True(t, result["success"].(bool))
    assert.Equal(t, "active", result["feature"].(map[string]interface{})["status"])
    assert.False(t, result["feature"].(map[string]interface{})["status_override"].(bool))
}
```

---

## 6. Test Coverage Targets

| Component | Target Coverage |
|-----------|-----------------|
| status/derivation.go | 100% |
| status/calculation.go | 90% |
| repository/feature_repository.go (new methods) | 90% |
| repository/epic_repository.go (new methods) | 90% |
| cli/commands/feature.go (--auto-status) | 80% |

---

## 7. Test Execution

### Run All Tests

```bash
make test
```

### Run Status Package Tests Only

```bash
go test -v ./internal/status/...
```

### Run Integration Tests

```bash
go test -v ./internal/status/... -run Integration
```

### Run with Coverage

```bash
go test -cover ./internal/status/... ./internal/repository/...
```

---

*Document Version: 1.0*
*Last Updated: 2026-01-01*
