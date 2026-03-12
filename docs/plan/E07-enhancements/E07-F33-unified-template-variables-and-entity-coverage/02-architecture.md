# E07-F33 Architecture: Unified Template Variables and Entity Coverage

**Date**: 2026-03-11
**Tier**: STANDARD (score 8/27)
**Status**: Draft

---

## 1. Key Architectural Decisions

### ADR-1: Keep `map[string]string` as the Template Variable Type

**Decision**: Retain the existing `map[string]string` placeholder approach rather than introducing a unified Go struct.

**Rationale**:
- The `OrchestratorAction.PopulateTemplate(vars map[string]string)` method already works with `map[string]string` across all 7 call sites (TaskService, FeatureService, EpicService, BugService, ChangeCardService, DisplayService x2).
- The `OrchestratorRenderer.Render(templateName string, vars map[string]string)` also uses `map[string]string`.
- A struct would require changes to the template engine, `PopulateTemplate`, and all `.tmpl` files (they reference `{{.key}}` not `{{.Key}}`).
- The existing per-entity `XxxPlaceholders()` functions in `internal/config/template_helpers.go` are the right abstraction -- they convert typed models to `map[string]string`. The fix is to make them produce a consistent, complete key set.

**Consequence**: No template engine changes needed. Only the placeholder factory functions and `GetStatusActionPopulated` need updates.

### ADR-2: Canonical Variable Name Set

**Decision**: Standardize on lowercase_snake_case variable names that match the existing code (not the documentation). Fix the docs to match code.

**Current state of the bug**: `TaskPlaceholders()` sets `epic_id` and `feature_id` to the *task* key value (lines 22-24 of template_helpers.go). This means `{epic_id}` in a template resolves to `E07-F01-001` instead of `E07`.

**Canonical variable names** (superset across all entity types):

| Variable | Available For | Description |
|----------|--------------|-------------|
| `key` | All | The entity's own key (E07, E07-F01, E07-F01-001, B001, CC-001) |
| `id` | All | Alias for `key` (backward compat) |
| `title` | All | Entity title |
| `description` | All | Entity description (empty if nil) |
| `status` | All | Current workflow status |
| `slug` | All | URL-friendly slug |
| `file_path` | All | File path |
| `created_at` | All | RFC3339 timestamp |
| `updated_at` | All | RFC3339 timestamp |
| `epic_key` | Feature, Task | Parent epic key (e.g., `E07`) |
| `feature_key` | Task | Parent feature key (e.g., `E07-F01`) |
| `task_key` | Task | Alias for `key` when entity is a task |
| `agent_type` | Task | Agent assignment |
| `priority` | Task, Epic, ChangeCard | Priority value |
| `depends_on` | Task | Dependency string |
| `execution_order` | Task, Feature | Execution order |
| `blocked_reason` | Task | Blocked reason |
| `completion_notes` | Task | Completion notes |
| `files_changed` | Task | Files changed |
| `business_value` | Epic | Business value |
| `severity` | Bug | Bug severity level |
| `linked_entity_type` | Bug | Linked entity type |
| `linked_entity_key` | Bug | Linked entity key |
| `requested_by` | ChangeCard | Who requested |
| `assigned_to` | ChangeCard | Who is assigned |
| `justification` | ChangeCard | Change justification |
| `impact_analysis` | ChangeCard | Impact analysis text |
| `rollback_plan` | ChangeCard | Rollback plan text |
| `related_docs` | Task, Feature, Epic | CSV of related doc paths (from `WithRelated` variants) |
| `related_tasks` | Task | CSV of related task keys |
| `related_features` | Feature | CSV of related feature keys |
| `related_epics` | Epic | CSV of related epic keys |
| `complexity_tier` | Task, Feature | Complexity tier from metadata |

**Deprecation**: The `task_id`, `epic_id`, `feature_id` aliases are kept for backward compatibility but deprecated in favor of `task_key`, `epic_key`, `feature_key`. The old names were misleading (they implied database IDs but contained entity keys) and were all set to the same value. The aliases now resolve to the correct parent keys (e.g., `epic_id` resolves to `E07`, not the task key) but should be migrated to the canonical `_key` names in templates.

### ADR-3: Enrich Existing Placeholder Functions (Not a New Unified Struct)

**Decision**: Update the existing `TaskPlaceholders()`, `FeaturePlaceholders()`, `EpicPlaceholders()`, `BugPlaceholders()`, and `ChangeCardPlaceholders()` functions to populate the full canonical variable set. Do NOT create a new `EntityTemplateData` Go struct.

**Rationale**:
- The PRD suggests a unified `EntityTemplateData` struct (REQ-F-001), but after analyzing the codebase, the existing `map[string]string` approach is simpler and requires fewer changes.
- Each placeholder function already knows its entity type and can set the correct hierarchy keys (`epic_key`, `feature_key`, `task_key`).
- The `WithRelated` variants already extend the base functions -- this pattern works well.
- Entity-specific fields that don't apply are simply absent from the map (templates handle missing keys gracefully via the `{placeholder}` pass-through behavior).

**Consequence**: Smaller changeset, no new types, no template engine changes.

### ADR-4: Fix `GetStatusActionPopulated` Signature

**Decision**: Change `GetStatusActionPopulated` to accept `map[string]string` placeholders directly instead of a single `taskID string`.

**Current signature** (broken):
```go
GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error)
```

**New signature**:
```go
GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error)
```

**Rationale**:
- The current implementation builds a 4-entry map where all values are `taskID` -- this is the core bug.
- Callers already have access to the entity (task/feature/epic/bug/change-card) and should call the appropriate `XxxPlaceholders()` function to build the full variable map.
- Most production call sites (7 of them in services) already bypass `GetStatusActionPopulated` and call `PopulateTemplate(placeholders)` directly on the `OrchestratorAction`. This signature change only affects the `ActionService` interface and its callers.

**Impact**: The `ActionService` interface, `DefaultActionService`, `MockActionService`, and any direct callers of `GetStatusActionPopulated` need updates. Tests need updating.

### ADR-5: Task Placeholders Need Parent Key Resolution

**Decision**: `TaskPlaceholders()` needs access to parent epic/feature keys to correctly populate `epic_key` and `feature_key`.

**Options considered**:
1. **Pass parent keys as parameters** -- Add `epicKey` and `featureKey` params to `TaskPlaceholders()`.
2. **Parse from task key** -- Extract `E07` and `E07-F01` from `T-E07-F01-001` by string parsing.
3. **Look up from database** -- Query epic/feature by ID.

**Decision**: Option 2 (parse from task key). The task key format `T-E##-F##-###` or `E##-F##-###` deterministically encodes the parent keys. This avoids adding database dependencies to a pure utility function and avoids changing the function signature (which has 10+ callers).

For bugs and change-cards, `epic_key` and `feature_key` are not applicable (they use `linked_entity_type`/`linked_entity_key` instead).

---

## 2. Component Design

### Files to Modify

#### `internal/config/template_helpers.go` (Primary changes)

1. **`TaskPlaceholders(task *models.Task) map[string]string`**:
   - Add `key` (same as `id`, alias for backward compat)
   - Parse `epic_key` and `feature_key` from `task.Key` (e.g., `T-E07-F01-001` -> `E07`, `E07-F01`)
   - Add `task_key` as alias for `key`
   - Keep `epic_id` and `feature_id` as deprecated aliases (now correctly resolved to parent keys)
   - Add `description` from `task.Description`

2. **`FeaturePlaceholders(feature *models.Feature) map[string]string`**:
   - Add `key` (same as `id`)
   - Parse `epic_key` from `feature.Key` (e.g., `E07-F01` -> `E07`)
   - Keep `feature_id` and `epic_id` as deprecated aliases (correctly resolved)
   - Add `description` from `feature.Description`

3. **`EpicPlaceholders(epic *models.Epic) map[string]string`**:
   - Add `key` (same as `id`)
   - Remove misleading `epic_id` alias (it was identical to `id`)

4. **`BugPlaceholders(bug *models.Bug) map[string]string`**:
   - Expand with full common fields: `description`, `slug`, `created_at`, `updated_at`
   - Add `linked_entity_type`, `linked_entity_key`

5. **`ChangeCardPlaceholders(card *models.ChangeCard) map[string]string`**:
   - Expand with full fields: `slug`, `created_at`, `updated_at`
   - Add `requested_by`, `assigned_to`, `justification`, `impact_analysis`, `rollback_plan`

6. **New helper**: `parseEpicKeyFromEntityKey(entityKey string) string` -- extracts `E##` from task or feature keys.
7. **New helper**: `parseFeatureKeyFromTaskKey(taskKey string) string` -- extracts `E##-F##` from task keys.

#### `internal/config/action_service.go`

1. **`ActionService` interface**: Change `GetStatusActionPopulated` signature from `(ctx, status, taskID string)` to `(ctx, status string, vars map[string]string)`.
2. **`DefaultActionService.GetStatusActionPopulated`**: Remove the hardcoded 4-var map. Pass `vars` directly to `PopulateTemplate`.

#### `internal/config/mock_action_service.go`

1. Update `MockActionService.GetStatusActionPopulatedFunc` signature to match new interface.

#### `internal/config/action_service_test.go`

1. Update test callers to pass `map[string]string` instead of `taskID string`.

#### `internal/templates/renderer.go` (No changes needed for MVP)

The `TemplateData` struct is used only by `Renderer.Render()` for task markdown file generation (not orchestrator actions). It is separate from the `map[string]string` placeholder system. If future work wants to unify task file rendering with the same placeholder system, that can be a follow-up.

#### Service files (Minimal changes)

The 7 service call sites that call `meta.OrchestratorAction.PopulateTemplate(placeholders)` directly are already correct -- they pass entity-specific placeholders. No changes needed for those.

Only call sites that use `GetStatusActionPopulated` on the `ActionService` interface need updating (currently appears to be only tests and the mock).

### Files NOT Modified

- `internal/templates/orchestrator_renderer.go` -- Already works with `map[string]string`, no changes.
- `internal/templates/renderer.go` -- Separate system for task file generation, out of scope.
- `internal/models/bug.go`, `internal/models/change_card.go` -- Models are fine as-is.
- Service `resolveAction()` methods -- Already use per-entity `XxxPlaceholders()` functions correctly.

---

## 3. Interface Changes

### `ActionService` Interface (Breaking Change)

```go
// Before
type ActionService interface {
    GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error)
    GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error)
    GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error)
    ValidateActions(ctx context.Context) (*ValidationResult, error)
    Reload(ctx context.Context) error
}

// After
type ActionService interface {
    GetStatusAction(ctx context.Context, status string) (*OrchestratorAction, error)
    GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error)
    GetAllActions(ctx context.Context) (map[string]*OrchestratorAction, error)
    ValidateActions(ctx context.Context) (*ValidationResult, error)
    Reload(ctx context.Context) error
}
```

### Placeholder Function Signatures (No Change)

All `XxxPlaceholders()` functions keep their existing signatures. The change is in what keys/values they return, not their Go signatures.

---

## 4. Migration Approach for Variable Name Alignment

### Removed Variables (Breaking)

| Old Name | Replacement | Reason |
|----------|-------------|--------|
| `task_id` | `key` or `task_key` | Was set to task key, name implied database ID |
| `epic_id` (in TaskPlaceholders) | `epic_key` | Was incorrectly set to task key; now correctly set to epic key |
| `feature_id` (in TaskPlaceholders) | `feature_key` | Was incorrectly set to task key; now correctly set to feature key |
| `epic_id` (in EpicPlaceholders) | `key` | Was redundant with `id` |
| `feature_id` (in FeaturePlaceholders) | `key` | Was redundant with `id` |

### Added Variables

| New Name | Entity Types | Value |
|----------|-------------|-------|
| `key` | All | Entity's own key (was only in Bug/ChangeCard before) |
| `epic_key` | Feature, Task | Parsed from entity key |
| `feature_key` | Task | Parsed from task key |
| `task_key` | Task | Alias for `key` |
| `description` | Bug (expanded) | Bug description field |
| `slug` | Bug, ChangeCard (expanded) | Slug field |
| `created_at` | Bug (expanded) | Timestamp |
| `updated_at` | Bug (expanded) | Timestamp |
| `linked_entity_type` | Bug | Linked entity type |
| `linked_entity_key` | Bug | Linked entity key |
| `requested_by` | ChangeCard | Requested by field |
| `assigned_to` | ChangeCard | Assigned to field |
| `justification` | ChangeCard | Justification text |
| `impact_analysis` | ChangeCard | Impact analysis text |
| `rollback_plan` | ChangeCard | Rollback plan text |

### Template Migration

Search all `.tmpl` files and `.sharkconfig.json` for `{task_id}`, `{epic_id}`, `{feature_id}` and replace:
- `{task_id}` -> `{key}` (or `{task_key}` if in a context where multiple entity types are referenced)
- `{epic_id}` -> `{epic_key}`
- `{feature_id}` -> `{feature_key}`

The `{id}` alias is kept for backward compatibility.

---

## 5. Task Breakdown

| Task | Complexity | Description |
|------|-----------|-------------|
| T1 | S | Add key-parsing helpers (`parseEpicKeyFromEntityKey`, `parseFeatureKeyFromTaskKey`) with tests |
| T2 | M | Update `TaskPlaceholders` to populate full canonical variable set with correct hierarchy keys |
| T3 | S | Update `FeaturePlaceholders` to add `key`, `epic_key`, remove misleading `feature_id` |
| T4 | XS | Update `EpicPlaceholders` to add `key`, remove misleading `epic_id` |
| T5 | S | Expand `BugPlaceholders` with full field set |
| T6 | S | Expand `ChangeCardPlaceholders` with full field set |
| T7 | S | Change `GetStatusActionPopulated` signature and implementation |
| T8 | XS | Update `MockActionService` and tests |
| T9 | S | Update `.tmpl` files and config for variable name changes |
| T10 | S | Update documentation (`docs/guides/template-system.md`, `docs/cli-reference/template-system.md`) |

---

*Last Updated*: 2026-03-11
