# Feature Context Research: E07-F34 Template Variable Enrichment

**Date**: 2026-03-17
**Researcher**: Researcher Agent

---

## Executive Summary

E07-F34 enriches the orchestrator template system with five new variable groups (previous_status, parent_title, context_data structured fields, latest_note, sibling_progress). The primary implementation location is `internal/config/template_helpers.go`, specifically the three `*PlaceholdersWithRelated()` functions. The feature requires no schema changes and minimal new repository methods. A consolidated enrichment query is recommended for Turso round-trip efficiency.

---

## 1. Current Template Variable System

### How Placeholders Work

The template variable system is a flat `map[string]string` pipeline:

1. **Basic Placeholder Functions** (`TaskPlaceholders()`, `FeaturePlaceholders()`, `EpicPlaceholders()`) in `internal/config/template_helpers.go` extract model fields into a `map[string]string`.

2. **Extended Placeholder Functions** (`TaskPlaceholdersWithRelated()`, `FeaturePlaceholdersWithRelated()`, `EpicPlaceholdersWithRelated()`) wrap the basic functions and add relationship data by querying document and relationship repositories. They also call `extractContextDataMetadata()` to flatten `ContextData.Metadata` key-value pairs.

3. **Template Rendering** happens in two places:
   - `OrchestratorRenderer.Render()` in `internal/templates/orchestrator_renderer.go` uses Go's `text/template` with the flat map as data.
   - `PopulateTemplate()` on orchestrator actions uses simple string replacement: `{{.key}}` -> value.

4. **Call Sites** - The `*PlaceholdersWithRelated()` functions are called from:
   - `TaskService.resolveAction()` (line 761 of task_service.go)
   - `FeatureService.resolveAction()` (line 363 of feature_service.go)
   - `EpicService.resolveAction()` (line 359 of epic_service.go)
   - `DisplayService.ResolveTaskAction()` / `.ResolveFeatureAction()` / `.ResolveEpicAction()` (display_service.go lines 649, 622, 443)

### Function Signatures (Current)

```go
func TaskPlaceholdersWithRelated(
    ctx context.Context,
    task *models.Task,
    docRepo DocumentRepository,
    taskRelRepo TaskRelationshipRepository,
) map[string]string

func FeaturePlaceholdersWithRelated(
    ctx context.Context,
    feature *models.Feature,
    docRepoForFeature interface { ListForFeature(...) },
    featureRelRepo FeatureRelationshipRepository,
) map[string]string

func EpicPlaceholdersWithRelated(
    epic *models.Epic,
    docRepoForEpic interface { ListForEpic(...) },
    epicRelRepo EpicRelationshipRepository,
    ctx context.Context,
) map[string]string
```

### extractContextDataMetadata (Current)

Currently only extracts `ContextData.Metadata` (arbitrary key-value pairs). Does NOT extract the structured fields: `Progress`, `ImplementationDecisions`, `OpenQuestions`, `Blockers`.

---

## 2. Data Sources for Each Enrichment

### 2.1 `previous_status`

**Data source**: `task_history` table.
- Query: `SELECT old_status FROM task_history WHERE task_id = ? ORDER BY timestamp DESC LIMIT 1`
- Returns the `old_status` from the most recent history entry, which represents the status before the current one.
- **Existing repository**: `TaskHistoryRepository.ListByTask(ctx, taskID)` returns all history ordered DESC. Could use this (take first element) but wasteful for Turso. A new `GetMostRecentByTaskID(ctx, taskID)` method with `LIMIT 1` would be more efficient.
- **Feature/Epic**: No `feature_history` or `epic_history` tables exist. History is only tracked for tasks. Feature/epic `previous_status` would need either: (a) new history tables, or (b) storing previous_status directly on the entity. **Recommendation**: For v1, only implement for tasks since that is where the rejection-loop branching is most valuable. Features and epics can be added later.

### 2.2 `parent_title` (feature_title, epic_title)

**Data source**: `features` and `epics` tables via foreign key joins.
- Tasks have `feature_id` and `epic_id` columns (via feature -> epic relationship).
- Features have `epic_id`.
- **Existing repository methods**:
  - `EpicRepository.GetByID(ctx, id)` returns `*models.Epic`
  - `FeatureRepository.GetByID(ctx, id)` returns `*models.Feature`
- These are separate queries today. Could be consolidated into a single join query for Turso efficiency.
- The `task.EpicID` field may not be directly on the task model -- need to trace through `feature.EpicID`.

### 2.3 `context_data` Structured Fields

**Data source**: Already in-memory -- the `ContextData` JSON is already parsed by `extractContextDataMetadata()`. The structured fields are in `models.ContextData`:
- `Progress.CurrentStep` (*string)
- `Progress.CompletedSteps` ([]string)
- `Progress.RemainingSteps` ([]string)
- `OpenQuestions` ([]string)
- `Blockers` ([]BlockerContext)

**No new repository calls needed.** This is purely extending `extractContextDataMetadata()` to also extract the structured fields into the flat map.

### 2.4 `latest_note`

**Data source**: `entity_notes` table.
- Query: `SELECT content, note_type FROM entity_notes WHERE entity_type = ? AND entity_id = ? ORDER BY created_at DESC LIMIT 1`
- Count query: `SELECT COUNT(*) FROM entity_notes WHERE entity_type = ? AND entity_id = ?`
- Rejection count: `SELECT COUNT(*) FROM entity_notes WHERE entity_type = ? AND entity_id = ? AND note_type = 'rejection'`
- **Existing repository**: `EntityNoteRepository.GetByEntity(ctx, entityType, entityID)` returns ALL notes. No `LIMIT 1` method exists. A new method `GetLatestByEntity()` would minimize data transfer.

### 2.5 `sibling_progress`

**Data source**: `tasks` table (for features) and `features` table (for epics).
- Query: `SELECT status, COUNT(*) FROM tasks WHERE feature_id = ? GROUP BY status`
- **Existing interfaces**:
  - `FeatureTaskCounter.ListByFeature(ctx, featureID)` returns all tasks -- then count in-memory. Wasteful for Turso.
  - `FeatureTaskCounter.GetStatusBreakdownMapBatch(ctx, featureIDs)` does exactly what is needed but for batch. A single-feature variant or reuse of batch with single ID is viable.
  - `EpicFeatureCounter.ListByEpic(ctx, epicID)` returns all features. Same issue.
- **Recommendation**: Use existing batch methods with single-element arrays, or add lightweight count methods.

---

## 3. Integration Points

### Files That Call Placeholder Functions

| File | Function | Entity | Repositories Passed |
|------|----------|--------|---------------------|
| `internal/services/task_service.go:761` | `resolveAction()` | Task | `s.docRepo`, `s.relRepo` |
| `internal/services/feature_service.go:363` | `resolveAction()` | Feature | `s.docRepo`, `s.relRepo` |
| `internal/services/epic_service.go:359` | `resolveAction()` | Epic | `s.docRepo`, `s.relRepo` |
| `internal/services/display_service.go:649` | `ResolveTaskAction()` | Task | `s.deps.DocumentRepo`, `s.deps.TaskRelRepo` |
| `internal/services/display_service.go:622` | `ResolveFeatureAction()` | Feature | `s.deps.DocumentRepo`, nil |
| `internal/services/display_service.go:443` | `ResolveEpicAction()` | Epic | `s.deps.DocumentRepo`, nil |

### Template Files

The `shark-templates/` directory has been deleted from the working tree (git status shows deletions). Templates are loaded from a configurable `template_directory` path. The orchestrator renderer (`internal/templates/orchestrator_renderer.go`) finds templates by walking up from cwd.

### Template Rendering Pipeline

```
Service.resolveAction(ctx, entity, status)
  -> config.*PlaceholdersWithRelated(ctx, entity, repos...)  // builds map[string]string
  -> meta.OrchestratorAction.PopulateTemplate(placeholders)  // string substitution
  -> returns PopulatedAction with rendered Instruction
```

---

## 4. Extension Approach

### 4.1 context_data Structured Fields (Zero-cost, no new queries)

Extend `extractContextDataMetadata()` to also extract structured fields:

```go
func extractContextDataMetadata(contextData *string, placeholders map[string]string) {
    // ... existing Metadata extraction ...

    // NEW: Extract structured progress fields
    if cd.Progress != nil {
        if cd.Progress.CurrentStep != nil {
            placeholders["current_step"] = *cd.Progress.CurrentStep
        }
        placeholders["completed_steps"] = strings.Join(cd.Progress.CompletedSteps, ", ")
        placeholders["remaining_steps"] = strings.Join(cd.Progress.RemainingSteps, ", ")
        placeholders["completed_steps_count"] = fmt.Sprintf("%d", len(cd.Progress.CompletedSteps))
        placeholders["remaining_steps_count"] = fmt.Sprintf("%d", len(cd.Progress.RemainingSteps))
    }

    // NEW: Extract open questions and blockers
    placeholders["open_questions"] = strings.Join(cd.OpenQuestions, ", ")
    placeholders["blockers_summary"] = fmt.Sprintf("%d blocker(s)", len(cd.Blockers))
}
```

**Impact**: Only `internal/config/template_helpers.go` changes. No signature changes.

### 4.2 Options for Remaining Enrichments (Require New Queries)

**Option A: Extend function signatures with additional repository interfaces**

Add new parameters to each `*PlaceholdersWithRelated()` function. This follows the existing pattern but increases parameter count further. Each call site in services must pass additional repos.

**Option B: Consolidated enrichment data struct (Recommended)**

Create an `EnrichmentData` struct that consolidates all enrichment fields, and a single function/method that populates it:

```go
type TemplateEnrichmentData struct {
    PreviousStatus  string
    ParentTitle     string  // feature_title for tasks, epic_title for features
    GrandparentTitle string // epic_title for tasks
    LatestNote      *models.EntityNote
    NotesCount      int
    RejectionCount  int
    ChildTotal      int
    ChildCompleted  int
    ChildBlocked    int
}

func TaskPlaceholdersWithRelated(
    ctx context.Context,
    task *models.Task,
    docRepo DocumentRepository,
    taskRelRepo TaskRelationshipRepository,
    enrichment *TemplateEnrichmentData,  // NEW: optional, nil-safe
) map[string]string
```

This keeps backward compatibility (pass nil for enrichment) and lets services construct the enrichment data however they choose (single query or multiple).

**Option C: Single consolidated SQL query per entity type**

A repository method like `GetTemplateEnrichmentData(entityType, entityID)` that uses JOINs and subqueries to fetch everything in one round-trip. Best for Turso performance. Example:

```sql
SELECT
    (SELECT old_status FROM task_history WHERE task_id = t.id ORDER BY timestamp DESC LIMIT 1) as previous_status,
    f.title as feature_title,
    e.title as epic_title,
    (SELECT content FROM entity_notes WHERE entity_type = 'task' AND entity_id = t.id ORDER BY created_at DESC LIMIT 1) as latest_note,
    (SELECT note_type FROM entity_notes WHERE entity_type = 'task' AND entity_id = t.id ORDER BY created_at DESC LIMIT 1) as latest_note_type,
    (SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'task' AND entity_id = t.id) as notes_count,
    (SELECT COUNT(*) FROM entity_notes WHERE entity_type = 'task' AND entity_id = t.id AND note_type = 'rejection') as rejection_count
FROM tasks t
LEFT JOIN features f ON t.feature_id = f.id
LEFT JOIN epics e ON f.epic_id = e.id
WHERE t.id = ?
```

---

## 5. Turso DB Consideration

The feature doc explicitly calls out minimizing round-trips. Current analysis:

| Enrichment | New Queries Needed | Can Consolidate? |
|------------|-------------------|-----------------|
| context_data fields | 0 (in-memory) | N/A |
| previous_status | 1 (task_history LIMIT 1) | Yes, via subquery |
| parent_title | 1-2 (feature + epic GetByID) | Yes, via JOIN |
| latest_note | 1-2 (latest note + counts) | Yes, via subquery |
| sibling_progress | 1 (GROUP BY status count) | Partially (different entity level) |

**Recommendation**: Use **Option C** (consolidated query) for items 1, 2, and 4 since they all relate to the same entity and can be fetched in a single SQL query with JOINs and subqueries. Item 5 (sibling_progress) operates at a different level (parent entity's children) but can still be added as a subquery.

**Total queries needed with consolidation**:
- **1 query** for task enrichment (previous_status + parent titles + latest note + counts)
- **1 query** for sibling_progress (child status counts)
- **0 queries** for context_data structured fields

Down from potentially 5-7 separate queries without consolidation.

---

## 6. File Impact List

### Must Change

| File | Changes |
|------|---------|
| `internal/config/template_helpers.go` | Extend `extractContextDataMetadata()` for structured fields. Modify `*PlaceholdersWithRelated()` signatures to accept enrichment data. Add enrichment data to placeholder maps. |
| `internal/config/template_helpers_test.go` | Tests for new placeholder variables |

### Likely Change (Repository Layer)

| File | Changes |
|------|---------|
| `internal/repository/task_history_repository.go` | Add `GetMostRecentByTaskID(ctx, taskID)` method returning single history entry |
| `internal/repository/entity_note_repository.go` | Add `GetLatestByEntity(ctx, entityType, entityID)` method and `CountByEntity()`, `CountByEntityAndType()` |
| *New file or existing repo* | Add consolidated `GetTemplateEnrichmentData()` if going with Option C |

### Likely Change (Service Layer)

| File | Changes |
|------|---------|
| `internal/services/task_service.go` | Update `resolveAction()` to construct and pass enrichment data |
| `internal/services/feature_service.go` | Update `resolveAction()` to construct and pass enrichment data |
| `internal/services/epic_service.go` | Update `resolveAction()` to construct and pass enrichment data |
| `internal/services/display_service.go` | Update `ResolveTaskAction()`, `ResolveFeatureAction()`, `ResolveEpicAction()` to pass enrichment data |

### No Change Expected

| File | Reason |
|------|--------|
| `internal/templates/orchestrator_renderer.go` | Template rendering unchanged (still flat map) |
| `internal/templates/loader.go` | Entity file templates unchanged |
| `internal/models/context_data.go` | Structs already have the fields needed |
| Database schema | No new tables or columns needed |

---

## 7. Risks and Unknowns

### Risks

- **Signature changes to `*PlaceholdersWithRelated()`**: These functions are called from 6+ locations. Any signature change requires updating all call sites. Using an optional struct parameter (nil-safe) mitigates this.
- **Feature/Epic history**: Only `task_history` exists. `previous_status` for features/epics requires new tables or a different approach. Recommend deferring to task-only for v1.
- **Turso latency**: If enrichment queries are not consolidated, each template render could add 3-5 remote round-trips. The consolidated query approach is important.

### Unknowns

- Whether `task.EpicID` is directly accessible on the Task model or requires going through `feature.EpicID`. Need to verify the Task model struct.
- The exact performance impact of adding subqueries to the enrichment query on Turso. Should be measured with a simple benchmark.

---

## 8. Recommended Implementation Order

1. **Item 3: context_data structured fields** -- Zero new queries, highest impact (eliminates PLAN GATE pattern in 8+ templates), pure code change in one function.
2. **Consolidated enrichment query** -- Build the repository method that fetches previous_status + parent titles + latest note in one query.
3. **Item 1: previous_status** -- Uses the enrichment query.
4. **Item 2: parent_title** -- Uses the enrichment query.
5. **Item 4: latest_note** -- Uses the enrichment query.
6. **Item 5: sibling_progress** -- Separate count query, medium impact.

---

## References

- `internal/config/template_helpers.go` -- Primary implementation location (lines 84-632)
- `internal/config/template_helpers_test.go` -- Existing test suite
- `internal/models/context_data.go` -- ContextData struct definition
- `internal/services/display_service.go` -- DisplayService call sites
- `internal/services/task_service.go` -- TaskService.resolveAction()
- `internal/repository/task_history_repository.go` -- Task history queries
- `internal/repository/entity_note_repository.go` -- Entity note queries
- Feature E07-F29 -- Prior related work (template variables for related docs and tasks)
