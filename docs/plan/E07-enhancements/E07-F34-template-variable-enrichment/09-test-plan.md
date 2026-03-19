# E07-F34 Test Plan: Template Variable Enrichment

**Feature**: Template Variable Enrichment
**Date**: 2026-03-17
**Status**: Draft
**Complexity**: STANDARD (15/27)

---

## 1. Acceptance Criteria Test Matrix

This section maps each enrichment variable group to concrete test cases with inputs, expected outputs, and edge cases.

### 1.1 context_data Structured Fields (Zero-Query, In-Memory)

These variables are extracted from the existing `ContextData` JSON already present on the entity model. No new repository calls are needed.

| TC ID | Test Case | Input (ContextData JSON) | Expected Placeholders | Priority |
|-------|-----------|--------------------------|----------------------|----------|
| TC-CD-01 | Progress with all fields | `{"progress":{"current_step":"Implementing API","completed_steps":["Design","DB Schema"],"remaining_steps":["Tests","Review"]}}` | `current_step`=`"Implementing API"`, `completed_steps`=`"Design, DB Schema"`, `remaining_steps`=`"Tests, Review"`, `completed_steps_count`=`"2"`, `remaining_steps_count`=`"2"` | High |
| TC-CD-02 | Progress with only current_step | `{"progress":{"current_step":"Writing tests"}}` | `current_step`=`"Writing tests"`, `completed_steps_count`=`"0"`, `remaining_steps_count`=`"0"` | High |
| TC-CD-03 | Progress nil (no progress field) | `{"metadata":{"foo":"bar"}}` | No `current_step`, no `completed_steps`, `completed_steps_count` not set or `"0"` | High |
| TC-CD-04 | Open questions populated | `{"open_questions":["Auth provider?","Rate limiting?"]}` | `open_questions`=`"Auth provider?; Rate limiting?"`, `open_questions_count`=`"2"` | Medium |
| TC-CD-05 | Open questions empty | `{"open_questions":[]}` | `open_questions` absent or empty, `open_questions_count` absent or `"0"` | Medium |
| TC-CD-06 | Blockers populated | `{"blockers":[{"description":"Waiting on API spec","blocker_type":"external","blocked_since":"2026-03-01T00:00:00Z"}]}` | `blockers_count`=`"1"`, `latest_blocker`=`"Waiting on API spec"` | Medium |
| TC-CD-07 | Multiple blockers (latest = last in array) | `{"blockers":[{"description":"First","blocker_type":"internal","blocked_since":"..."},{"description":"Second","blocker_type":"external","blocked_since":"..."}]}` | `blockers_count`=`"2"`, `latest_blocker`=`"Second"` | Medium |
| TC-CD-08 | Implementation decisions count | `{"implementation_decisions":{"auth":"JWT","db":"PostgreSQL","cache":"Redis"}}` | `decisions_count`=`"3"` | Low |
| TC-CD-09 | Empty context_data string | `""` | No crash, no structured placeholders set | High |
| TC-CD-10 | Nil context_data pointer | `nil` | No crash, no structured placeholders set | High |
| TC-CD-11 | Malformed JSON | `"not json"` | Graceful skip, no crash, no placeholders set | High |
| TC-CD-12 | Metadata keys do not collide with structured fields | `{"metadata":{"current_step":"from-metadata"},"progress":{"current_step":"from-progress"}}` | Depends on extraction order. Document which wins. Metadata runs first per architecture doc, structured fields should NOT overwrite. | High |

### 1.2 previous_status (Task Only, Requires DB Query)

| TC ID | Test Case | Setup | Expected | Priority |
|-------|-----------|-------|----------|----------|
| TC-PS-01 | Task with history entries | Task has history: todo -> in_development -> in_code_review -> in_development | `previous_status`=`"in_code_review"` | High |
| TC-PS-02 | Task with no history | New task, no task_history entries | `previous_status`=`""` (empty string) | High |
| TC-PS-03 | Task with single history entry | Task transitioned once: todo -> in_development | `previous_status`=`"todo"` | High |
| TC-PS-04 | Feature entity (no history table) | Feature passed to enrichment | `previous_status`=`""` (features have no history table in v1) | Medium |
| TC-PS-05 | Epic entity (no history table) | Epic passed to enrichment | `previous_status`=`""` | Medium |

### 1.3 parent_title (Requires DB JOIN)

| TC ID | Test Case | Setup | Expected | Priority |
|-------|-----------|-------|----------|----------|
| TC-PT-01 | Task with valid feature and epic parents | Task in feature "User Auth" in epic "Security" | `parent_title`=`"User Auth"`, `grandparent_title`=`"Security"` | High |
| TC-PT-02 | Task with orphaned feature (epic deleted) | Task.feature_id valid but feature.epic_id points to missing epic | `parent_title`=feature title, `grandparent_title`=`""` (COALESCE in query) | Medium |
| TC-PT-03 | Feature with valid epic parent | Feature in epic "Enhancements" | `parent_title`=`"Enhancements"`, `grandparent_title`=`""` | High |
| TC-PT-04 | Epic (no parent) | Epic entity | `parent_title`=`""`, `grandparent_title`=`""` | Medium |

### 1.4 latest_note (Requires DB Query)

| TC ID | Test Case | Setup | Expected | Priority |
|-------|-----------|-------|----------|----------|
| TC-LN-01 | Entity with multiple notes | 3 notes: comment, decision, rejection (most recent) | `latest_note`=rejection content, `latest_note_type`=`"rejection"`, `notes_count`=`"3"`, `rejection_count`=`"1"` | High |
| TC-LN-02 | Entity with no notes | No entity_notes rows | `latest_note`=`""`, `latest_note_type`=`""`, `notes_count`=`"0"`, `rejection_count`=`"0"` | High |
| TC-LN-03 | Entity with only rejection notes | 2 rejection notes | `rejection_count`=`"2"`, `notes_count`=`"2"` | Medium |
| TC-LN-04 | Works for all entity types | Task, Feature, Epic each have notes | All three entity types produce correct counts | Medium |

### 1.5 sibling_progress (Requires DB COUNT Query)

| TC ID | Test Case | Setup | Expected | Priority |
|-------|-----------|-------|----------|----------|
| TC-SP-01 | Feature with mixed task statuses | Feature with 7 tasks: 3 completed, 1 blocked, 3 other | `sibling_total`=`"7"`, `sibling_completed`=`"3"`, `sibling_blocked`=`"1"` | High |
| TC-SP-02 | Feature with zero tasks | Feature has no tasks | `sibling_total`=`"0"`, `sibling_completed`=`"0"`, `sibling_blocked`=`"0"` | High |
| TC-SP-03 | Epic with mixed feature statuses | Epic with 4 features: 2 completed, 1 blocked | `sibling_total`=`"4"`, `sibling_completed`=`"2"`, `sibling_blocked`=`"1"` | High |
| TC-SP-04 | Task sibling context | Task enrichment includes sibling counts from same feature | Sibling counts reflect all tasks in the same feature | Medium |
| TC-SP-05 | Epic with zero features | Epic has no features | `sibling_total`=`"0"`, `sibling_completed`=`"0"`, `sibling_blocked`=`"0"` | Medium |

---

## 2. Component Test Strategy

### 2.1 Unit Tests: extractContextDataFields (internal/config/template_helpers_test.go)

**What**: Pure function tests for the renamed/extended `extractContextDataMetadata` -> `extractContextDataFields` function.

**Pattern**: Matches the existing `TestExtractContextDataMetadata_*` tests in `template_helpers_test.go`. Table-driven tests, no database, no mocks needed.

**Test file**: `internal/config/template_helpers_test.go` (extend existing file)

**Tests to add**:
- `TestExtractContextDataFields_ProgressAllFields` -- TC-CD-01
- `TestExtractContextDataFields_ProgressPartial` -- TC-CD-02
- `TestExtractContextDataFields_NoProgress` -- TC-CD-03
- `TestExtractContextDataFields_OpenQuestions` -- TC-CD-04, TC-CD-05
- `TestExtractContextDataFields_Blockers` -- TC-CD-06, TC-CD-07
- `TestExtractContextDataFields_Decisions` -- TC-CD-08
- `TestExtractContextDataFields_EmptyAndNil` -- TC-CD-09, TC-CD-10
- `TestExtractContextDataFields_MalformedJSON` -- TC-CD-11
- `TestExtractContextDataFields_MetadataCollision` -- TC-CD-12

**Approach**: Build JSON strings, call `extractContextDataFields()`, inspect the resulting `map[string]string`.

### 2.2 Unit Tests: applyEnrichmentData (internal/config/template_helpers_test.go)

**What**: Pure function tests for the new `applyEnrichmentData()` helper.

**Tests to add**:
- `TestApplyEnrichmentData_AllFieldsPopulated` -- All fields set, all placeholders appear
- `TestApplyEnrichmentData_NilEnrichment` -- Pass nil, no crash, no changes to map
- `TestApplyEnrichmentData_ZeroValues` -- Struct with all zero-values, counts render as "0"
- `TestApplyEnrichmentData_PartialData` -- Only some fields populated (e.g., previous_status set but no notes)
- `TestApplyEnrichmentData_EmptyStringsNotOverwritten` -- Empty PreviousStatus does NOT add `previous_status` key

**Approach**: Construct `TemplateEnrichmentData` struct, call `applyEnrichmentData()`, inspect placeholder map.

### 2.3 Unit Tests: Signature Backward Compatibility (internal/config/template_helpers_test.go)

**What**: Verify that passing `nil` as the enrichment parameter to `*PlaceholdersWithRelated()` produces identical output to the current (pre-change) behavior.

**Tests to add**:
- `TestTaskPlaceholdersWithRelated_NilEnrichment` -- Pass nil enrichment, verify all existing placeholders unchanged
- `TestFeaturePlaceholdersWithRelated_NilEnrichment` -- Same for features
- `TestEpicPlaceholdersWithRelated_NilEnrichment` -- Same for epics

**Approach**: Call the function with nil enrichment and mock repos (or nil repos matching existing nil-safety behavior). Compare output keys/values against the current baseline.

### 2.4 Repository Integration Tests: TemplateEnrichmentRepository (internal/repository/template_enrichment_repository_test.go)

**What**: Integration tests using real database for the new consolidated enrichment queries.

**Pattern**: Follows existing repository test patterns -- use `test.GetTestDB()`, clean before each test, use `TEST-` prefixed keys, defer cleanup.

**Test file**: `internal/repository/template_enrichment_repository_test.go` (new file)

**Tests to add**:

```
TestTemplateEnrichmentRepository_GetTaskEnrichment_FullData
  - Seed: epic, feature, task, task_history entries, entity_notes (mix of types), sibling tasks
  - Verify: previous_status, parent_title, grandparent_title, latest_note_*, notes_count, rejection_count, sibling_*

TestTemplateEnrichmentRepository_GetTaskEnrichment_EmptyHistory
  - Seed: task with no history entries
  - Verify: previous_status = ""

TestTemplateEnrichmentRepository_GetTaskEnrichment_NoNotes
  - Seed: task with no entity_notes
  - Verify: latest_note = "", notes_count = 0, rejection_count = 0

TestTemplateEnrichmentRepository_GetTaskEnrichment_NoSiblings
  - Seed: single task in feature (no siblings)
  - Verify: sibling_total = 1 (includes self), sibling_completed/blocked counts correct

TestTemplateEnrichmentRepository_GetFeatureEnrichment_FullData
  - Seed: epic, feature, multiple tasks with various statuses, entity_notes
  - Verify: parent_title (epic title), sibling counts (task counts in feature)

TestTemplateEnrichmentRepository_GetFeatureEnrichment_NoTasks
  - Seed: feature with zero tasks
  - Verify: sibling_total = 0

TestTemplateEnrichmentRepository_GetEpicEnrichment_FullData
  - Seed: epic, multiple features with various statuses, entity_notes
  - Verify: sibling counts (feature counts in epic)

TestTemplateEnrichmentRepository_GetEpicEnrichment_Empty
  - Seed: epic with no features, no notes
  - Verify: All counts = 0, empty strings for note fields

TestTemplateEnrichmentRepository_NonExistentEntity
  - Pass non-existent ID
  - Verify: Appropriate error or zero-valued struct returned
```

**Cleanup pattern**:
```go
_, _ = database.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")
_, _ = database.ExecContext(ctx, "DELETE FROM features WHERE key LIKE 'TEST-%'")
_, _ = database.ExecContext(ctx, "DELETE FROM epics WHERE key LIKE 'TEST-%'")
_, _ = database.ExecContext(ctx, "DELETE FROM task_history WHERE task_id IN (SELECT id FROM tasks WHERE key LIKE 'TEST-%')")
_, _ = database.ExecContext(ctx, "DELETE FROM entity_notes WHERE entity_id IN (SELECT id FROM tasks WHERE key LIKE 'TEST-%')")
```

### 2.5 Service Tests: resolveAction Enrichment Wiring (internal/services/*_service_test.go)

**What**: Verify that each service's `resolveAction()` method correctly fetches enrichment data and passes it to the placeholder function. Uses mocked repositories.

**Pattern**: Follows existing service test patterns with `MockTaskRepository`, `MockWorkflowService`, etc.

**Tests to add** (in each service's test file):

```
TestTaskService_ResolveAction_WithEnrichment
  - Mock enrichment repo returns populated TemplateEnrichmentData
  - Verify: enrichment data appears in populated action template variables

TestTaskService_ResolveAction_NilEnrichmentRepo
  - Construct service with nil enrichment repo
  - Verify: No crash, template still renders with basic + relationship placeholders

TestTaskService_ResolveAction_EnrichmentRepoError
  - Mock enrichment repo returns error
  - Verify: Graceful degradation, template renders without enrichment data, warning logged

TestFeatureService_ResolveAction_WithEnrichment
  - Same pattern for FeatureService

TestEpicService_ResolveAction_WithEnrichment
  - Same pattern for EpicService

TestDisplayService_ResolveTaskAction_WithEnrichment
  - Same pattern for DisplayService task resolution

TestDisplayService_ResolveFeatureAction_WithEnrichment
  - Same pattern for DisplayService feature resolution

TestDisplayService_ResolveEpicAction_WithEnrichment
  - Same pattern for DisplayService epic resolution
```

**Mock interface needed**:
```go
type MockTemplateEnrichmentRepository struct {
    GetTaskEnrichmentFunc    func(ctx context.Context, taskID int64) (*config.TemplateEnrichmentData, error)
    GetFeatureEnrichmentFunc func(ctx context.Context, featureID int64) (*config.TemplateEnrichmentData, error)
    GetEpicEnrichmentFunc    func(ctx context.Context, epicID int64) (*config.TemplateEnrichmentData, error)
}
```

---

## 3. Integration Scenarios

### 3.1 Cross-Feature Touchpoints (6 Call Sites)

The `*PlaceholdersWithRelated()` function signatures change from 4 parameters to 5 parameters (adding `*TemplateEnrichmentData`). All 6 call sites must be updated.

| Call Site | File | Line | Entity | Verification |
|-----------|------|------|--------|-------------|
| TaskService.resolveAction | `internal/services/task_service.go` | ~761 | Task | Service test with mock enrichment repo |
| FeatureService.resolveAction | `internal/services/feature_service.go` | ~363 | Feature | Service test with mock enrichment repo |
| EpicService.resolveAction | `internal/services/epic_service.go` | ~359 | Epic | Service test with mock enrichment repo |
| DisplayService.ResolveTaskAction | `internal/services/display_service.go` | ~649 | Task | DisplayService test with mock enrichment repo |
| DisplayService.ResolveFeatureAction | `internal/services/display_service.go` | ~622 | Feature | DisplayService test with mock enrichment repo |
| DisplayService.ResolveEpicAction | `internal/services/display_service.go` | ~443 | Epic | DisplayService test with mock enrichment repo |

**Compile-time check**: After the signature change, `make build` must succeed. Any missed call site will be a compile error -- this is by design.

### 3.2 Constructor Wiring Verification

The enrichment repository must be wired into each service constructor and into the CLI global accessors.

| Wiring Point | File | Verification |
|-------------|------|-------------|
| TaskService constructor | `internal/services/task_service.go` | Accepts optional enrichRepo, nil-safe |
| FeatureService constructor | `internal/services/feature_service.go` | Same |
| EpicService constructor | `internal/services/epic_service.go` | Same |
| DisplayService.Dependencies | `internal/services/display_service.go` | TemplateEnrichmentRepo field added |
| GetTaskService() | `internal/cli/services_global.go` | Constructs and passes enrichment repo |
| GetFeatureService() | `internal/cli/services_global.go` | Same |
| GetEpicService() | `internal/cli/services_global.go` | Same |
| GetDisplayService() | `internal/cli/services_global.go` | Same |

**Test**: Build succeeds with `make build`. Existing `make test` passes (all tests still compile and run).

### 3.3 End-to-End Template Rendering

After all components are wired, a manual or scripted integration test should verify that a template containing the new variables renders correctly.

**Scenario**: Create a task, add history entries and notes, set context_data with progress fields, then trigger template rendering via `shark status advance` or `shark task resume`. Verify the populated action contains the enrichment variables.

**Validation approach**: Use `shark task resume <key> --json` and inspect the output for enrichment variable presence. Alternatively, create a test `.tmpl` file with `{{.previous_status}}`, `{{.parent_title}}`, `{{.latest_note}}` and verify they render to non-empty values.

---

## 4. Edge Cases and Defensive Testing

### 4.1 Nil Safety

| Scenario | Component | Expected Behavior |
|----------|-----------|-------------------|
| `enrichment` parameter is nil | `applyEnrichmentData()` | Early return, no placeholders modified |
| `enrichment` parameter is nil | `*PlaceholdersWithRelated()` | Produces same output as pre-change behavior |
| `enrichRepo` field is nil on service | `resolveAction()` | Skips enrichment fetch, passes nil to placeholder function |
| `contextData` is nil pointer | `extractContextDataFields()` | Early return, no crash |
| `contextData` is empty string | `extractContextDataFields()` | Early return, no crash |

### 4.2 Missing Parent Entities

| Scenario | Component | Expected Behavior |
|----------|-----------|-------------------|
| Task's feature_id points to deleted feature | Enrichment query | LEFT JOIN returns NULL, COALESCE returns `""` |
| Task's feature.epic_id points to deleted epic | Enrichment query | LEFT JOIN returns NULL, COALESCE returns `""` |
| Feature's epic_id points to deleted epic | Enrichment query | LEFT JOIN returns NULL, COALESCE returns `""` |

### 4.3 Empty Data Sets

| Scenario | Component | Expected Behavior |
|----------|-----------|-------------------|
| Task has zero history entries | Enrichment query | `previous_status` = `""` |
| Entity has zero notes | Enrichment query | `latest_note` = `""`, `notes_count` = `"0"`, `rejection_count` = `"0"` |
| Feature has zero tasks | Enrichment query | `sibling_total` = `"0"`, `sibling_completed` = `"0"`, `sibling_blocked` = `"0"` |
| Epic has zero features | Enrichment query | Same pattern |
| ContextData.Progress is nil | `extractContextDataFields()` | No progress placeholders set |
| ContextData.OpenQuestions is empty slice | `extractContextDataFields()` | No `open_questions` placeholder or empty string |
| ContextData.Blockers is empty slice | `extractContextDataFields()` | No `blockers_count` placeholder or `"0"` |

### 4.4 Enrichment Repository Errors

| Scenario | Component | Expected Behavior |
|----------|-----------|-------------------|
| Database connection failure | `resolveAction()` | Log warning, proceed without enrichment, template renders with basic placeholders |
| Query timeout | `resolveAction()` | Same graceful degradation |
| Non-existent entity ID passed | Enrichment repository | Return error or zero-valued struct (no crash) |

### 4.5 Data Ordering and Correctness

| Scenario | Component | Expected Behavior |
|----------|-----------|-------------------|
| Multiple task_history entries: verify most recent is used | Repository query | `ORDER BY timestamp DESC LIMIT 1` returns the correct row |
| Multiple entity_notes: verify most recent by created_at is used | Repository query | `ORDER BY created_at DESC LIMIT 1` returns the correct row |
| Notes with same created_at timestamp | Repository query | Deterministic result (ORDER BY id DESC as tiebreaker, or accept either) |

---

## 5. Quality Gates

### 5.1 Build Gate
- `make build` succeeds (all 6 call sites updated, no compile errors)
- `make fmt` produces no changes
- `make lint` passes with no new warnings

### 5.2 Test Coverage Gate
- All new functions have unit tests: `applyEnrichmentData()`, `extractContextDataFields()` extension
- Repository integration tests cover all 3 entity types (task, feature, epic)
- Service tests verify enrichment wiring for all 6 call sites
- Nil-safety tests for all optional parameters
- Edge case tests for empty data sets and error conditions

### 5.3 Backward Compatibility Gate
- Passing nil enrichment to `*PlaceholdersWithRelated()` produces identical output to pre-change
- Existing service tests pass without modification (other than adding nil enrichment parameter)
- No database schema changes required (no migration)

### 5.4 Performance Gate
- Enrichment query adds at most 1 additional DB round-trip per template render
- All subqueries in the consolidated query hit existing indexes
- No N+1 query patterns introduced

---

## 6. Test Execution Order

Implementation and testing should proceed in this order to minimize risk:

1. **Phase 1 (Zero-Query)**: Extend `extractContextDataMetadata` -> `extractContextDataFields`. Unit tests only. No signature changes, no wiring changes. Lowest risk.

2. **Phase 2 (Infrastructure)**: Create `TemplateEnrichmentData` struct, `TemplateEnrichmentRepository` interface, concrete implementation. Add repository integration tests. Still no signature changes to existing functions.

3. **Phase 3 (Signature Change)**: Modify `*PlaceholdersWithRelated()` signatures, add `applyEnrichmentData()`. Update all 6 call sites to pass nil. Add nil-safety unit tests. Run `make build` and `make test` to verify backward compat.

4. **Phase 4 (Wiring)**: Wire enrichment repo into services and CLI global accessors. Service tests with mock enrichment repos. Full `make test` pass.

5. **Phase 5 (Validation)**: End-to-end manual validation with a real template containing new variables.

---

## References

- Architecture: `docs/plan/E07-enhancements/E07-F34-template-variable-enrichment/02-architecture.md`
- Research: `docs/plan/E07-enhancements/E07-F34-template-variable-enrichment/F01-feature-context.md`
- Existing tests: `internal/config/template_helpers_test.go`
- ContextData model: `internal/models/context_data.go`
- Template helpers: `internal/config/template_helpers.go`
- Call sites: `internal/services/task_service.go`, `internal/services/feature_service.go`, `internal/services/epic_service.go`, `internal/services/display_service.go`
