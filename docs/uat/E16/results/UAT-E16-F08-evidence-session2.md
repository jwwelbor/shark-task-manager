# UAT Evidence: T-E16-F08-001 (Session 2)

**Date:** 2026-03-22
**Task:** Auto-update epic/feature status when child entities are added
**Evidence collector:** Claude (evidence only, no analysis)

---

## 1. Spec Quotes (Acceptance Criteria)

### AC-1 (task_spec lines 104-110)
> **Given** a feature `E01-F01` with status `completed`
> **When** a new task is created under `E01-F01` via `shark task create E01 F01 "New task"`
> **Then** the feature status changes to `active`
> **And** the task is created successfully with status `draft`
> **And** a history record is created for the feature with notes containing `"auto-reopened"`

### AC-2 (task_spec lines 112-117)
> **Given** a feature `E01-F01` with status `cancelled`
> **When** a new task is created under `E01-F01`
> **Then** the feature status changes to `active`
> **And** the task is created successfully

### AC-3 (task_spec lines 119-125)
> **Given** an epic `E01` with status `completed`
> **When** a new feature is created under `E01`
> **Then** the epic status changes to `active`
> **And** the feature is created successfully with status `draft`
> **And** a history record is created for the epic with notes containing `"auto-reopened"`

### AC-4 (task_spec lines 127-132)
> **Given** a feature `E01-F01` with status `active`
> **When** a new task is created under `E01-F01`
> **Then** the feature status remains `active` (unchanged)

### AC-5 (task_spec lines 134-138)
> **Given** a feature with status `in_refinement_ba` (non-terminal)
> **When** a new task is created
> **Then** the feature status remains unchanged

### AC-6 (task_spec lines 140-144)
> **Given** custom workflow where `_complete_` = `["done", "abandoned"]` and `_aggregation_` = `["tracking"]`
> **When** a task is created under a feature with status `done`
> **Then** the feature status changes to `tracking`

### AC-7 (task_spec lines 146-152)
> **Given** a feature with status `completed` and the feature repository update will fail
> **When** a new task is created
> **Then** the task is still created successfully
> **And** a warning is logged

### AC-8 (task_spec lines 154-160)
> **Given** a feature with status `completed`
> **When** a task is created via either creation path
> **Then** the feature auto-reopen logic executes in both cases

---

## 2. Implementation Code

### workflow/service.go - GetAggregationStatuses()
```go
func (s *Service) GetAggregationStatuses() []string {
    aggStatuses, exists := s.workflow.SpecialStatuses[config.AggregationStatusKey]
    if !exists || len(aggStatuses) == 0 {
        return []string{"active"}
    }
    return aggStatuses
}
```

### task_service.go:1195 - maybeReopenParentFeature()
```go
func (s *TaskService) maybeReopenParentFeature(ctx context.Context, featureKey string, taskKey string) {
    if s.featureService == nil {
        return
    }
    feature, err := s.featureService.GetFeature(ctx, featureKey)
    if err != nil {
        log.Printf("warning: auto-reopen check for feature %s failed: %v", featureKey, err)
        return
    }
    if feature == nil {
        return
    }
    featureWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelFeature)
    if !featureWf.IsTerminalStatus(string(feature.Status)) {
        return
    }
    aggStatuses := featureWf.GetAggregationStatuses()
    targetStatus := models.FeatureStatus(aggStatuses[0])
    _, err = s.featureService.UpdateFeature(ctx, feature.Key, FeatureUpdates{
        Status: &targetStatus,
    })
    if err != nil {
        log.Printf("warning: auto-reopen of feature %s failed: %v", featureKey, err)
    }
}
```

**Call sites:**
- task_service.go:346 (after creatorSvc path)
- task_service.go:408 (after fallback path)

### feature_service.go:793 - maybeReopenParentEpic()
```go
func (s *FeatureService) maybeReopenParentEpic(ctx context.Context, epic *models.Epic, featureKey string) {
    if s.epicLookupRepo == nil {
        return
    }
    epicWf := s.entitySvc.GetWorkflowService().ForLevel(workflow.LevelEpic)
    if !epicWf.IsTerminalStatus(string(epic.Status)) {
        return
    }
    aggStatuses := epicWf.GetAggregationStatuses()
    epic.Status = models.EpicStatus(aggStatuses[0])
    if err := s.epicLookupRepo.Update(ctx, epic); err != nil {
        log.Printf("warning: auto-reopen of epic %s failed: %v", epic.Key, err)
    }
}
```

**Call site:** feature_service.go:781

### feature_service.go - FeatureEpicLookup interface
```go
type FeatureEpicLookup interface {
    GetByKey(ctx context.Context, key string) (*models.Epic, error)
    GetByFilePath(ctx context.Context, filePath string) (*models.Epic, error)
    UpdateFilePath(ctx context.Context, epicKey string, newFilePath *string) error
    List(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error)
    Update(ctx context.Context, epic *models.Epic) error  // Added for auto-reopen
}
```

---

## 3. Test Code

### Workflow tests (3 tests)
- `TestService_GetAggregationStatuses` - Returns configured values
- `TestService_GetAggregationStatuses_NotConfigured` - Falls back to `["active"]`
- `TestService_GetAggregationStatuses_Empty` - Falls back to `["active"]`

### Task service tests (6 tests)
- `TestTaskService_CreateTask_ReopensTerminalFeature` - completed → active
- `TestTaskService_CreateTask_ReopensArchivedFeature` - archived → active (tests `archived`, not `cancelled`)
- `TestTaskService_CreateTask_NoReopenNonTerminalFeature` - active stays active
- `TestTaskService_CreateTask_NoReopenNilFeatureService` - graceful skip
- `TestTaskService_CreateTask_ReopenFailureDoesNotFailCreate` - best-effort
- `TestTaskService_CreateTask_CustomAggregationStatus` - custom _complete_ and _aggregation_

### Feature service tests (4 tests)
- `TestFeatureService_CreateFeature_ReopensTerminalEpic` - completed → active
- `TestFeatureService_CreateFeature_NoReopenNonTerminalEpic` - active stays active
- `TestFeatureService_CreateFeature_ReopenFailureDoesNotFailCreate` - best-effort
- `TestFeatureService_CreateFeature_CustomAggregationStatus` - custom values

---

## 4. Test Execution Output

All 14 auto-reopen tests: **PASS**
Full test suite: 31/32 packages pass. 1 failure in `internal/config` (4 template reference tests unrelated to E16-F08-001 -- caused by E22 branch merge).

---

## 5. DI Wiring Verification

### CLI (services_global.go)
- `GetTaskService()` calls `SetFeatureService(GetFeatureService())` at line ~201
- `GetFeatureService()` receives `epicRepo` for auto-reopen wiring

### HTTP (cmd/server/services.go)
- `WireServices()` wires `featureService` into `taskService` and `epicRepo` into `featureService`

---

## 6. Audit Trail Infrastructure Status

### entity_history table EXISTS (from E21)
```sql
CREATE TABLE IF NOT EXISTS entity_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL,
    changed_by TEXT,
    notes TEXT,
    forced INTEGER NOT NULL DEFAULT 0,
    rejection_reason TEXT,
    changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### entity_history_repository.go EXISTS
- `EntityHistoryRepository` with `Record()` method for creating history entries
- Used by `EntityService.StatusUpdateRaw()` for status transitions

### BUT: auto-reopen does NOT create entity_history records
- `maybeReopenParentFeature` calls `featureService.UpdateFeature()` which calls `repo.Update()` -- NO history record created
- `maybeReopenParentEpic` calls `epicLookupRepo.Update()` -- NO history record created
- No database trigger creates history on feature/epic status changes (triggers only update `updated_at`)
- The infrastructure now EXISTS but is not wired into the auto-reopen path

---

## 7. Parameter Usage in Log Messages

- `featureKey` IS used in log messages at task_service.go:1202 and :1221
- `taskKey` is NOT used in any log message or record (passed but unused in method body)
- `featureKey` param in `maybeReopenParentEpic` is NOT used (epic.Key is used instead)

---

## 8. Prior UAT Conditions Status

| Condition | Prior Status | Current Status |
|-----------|-------------|----------------|
| 1. Descope audit trail from AC-1/AC-3 | Required | entity_history infra now EXISTS (E21), but NOT wired into auto-reopen path |
| 2. Add test for `cancelled` status | Optional | NOT added |
| 3. Wire unused parameters | Optional | `taskKey` and `featureKey` (in epic method) still unused |
