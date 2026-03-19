# E07-F34 Architecture: Template Variable Enrichment

**Feature**: Template Variable Enrichment
**Date**: 2026-03-17
**Status**: Draft
**Complexity**: STANDARD (15/27)

---

## 1. Problem Statement

The orchestrator template system currently exposes only basic entity fields and relationship data (related_docs, related_tasks/features/epics) as template variables. Agents frequently need additional context -- previous status (for rejection-loop branching), parent entity titles (for context), structured progress data, latest notes, and sibling progress counts -- but must issue separate CLI round-trips to obtain them. This feature enriches the `*PlaceholdersWithRelated()` functions to provide these variables natively, eliminating 2-4 extra CLI calls per template invocation.

## 2. Design Principles

- **Appropriate**: Extends the existing flat `map[string]string` placeholder pipeline. No new rendering engine or template syntax required.
- **Proven**: Follows the exact pattern established by E07-F29 (related_docs, related_tasks) and E07-F33 (context_data metadata extraction).
- **Simple**: Zero-query enrichments first; remaining enrichments consolidated into a single SQL query per entity type.

## 3. Architecture Overview

### 3.1 Current Pipeline

```
Service.resolveAction(ctx, entity, status)
  -> config.*PlaceholdersWithRelated(ctx, entity, repos...)
     -> *Placeholders(entity)              // basic fields
     -> query related docs/relationships   // existing DB queries
     -> extractContextDataMetadata(...)    // existing in-memory extraction
  -> meta.OrchestratorAction.PopulateTemplate(placeholders)
  -> returns PopulatedAction
```

### 3.2 Enriched Pipeline (After This Feature)

```
Service.resolveAction(ctx, entity, status)
  -> config.*PlaceholdersWithRelated(ctx, entity, repos..., enrichment)
     -> *Placeholders(entity)              // basic fields (unchanged)
     -> query related docs/relationships   // existing DB queries (unchanged)
     -> extractContextDataStructured(...)  // EXTENDED: structured fields
     -> applyEnrichmentData(enrichment)    // NEW: optional enrichment
  -> meta.OrchestratorAction.PopulateTemplate(placeholders)
  -> returns PopulatedAction
```

### 3.3 Layered Responsibility

| Layer | Responsibility | Changes |
|-------|---------------|---------|
| **Repository** | Single consolidated SQL query returning `TemplateEnrichmentData` | New method on a new or existing repository |
| **Service** | Construct enrichment data by calling repository, pass to placeholder function | Updated `resolveAction()` in all 3 entity services + DisplayService |
| **Config (template_helpers)** | Accept optional enrichment struct, merge into placeholder map; extend structured context extraction | Modified function signatures + new extraction logic |

---

## 4. Data Model: TemplateEnrichmentData

No database schema changes are required. All data sources already exist.

A new Go struct consolidates enrichment fields fetched from existing tables:

```go
// TemplateEnrichmentData contains pre-fetched enrichment data for template rendering.
// All fields are optional (zero-value means "not fetched" or "not applicable").
// This struct is constructed by the service layer and passed to *PlaceholdersWithRelated().
type TemplateEnrichmentData struct {
    // Previous status from task_history (tasks only in v1)
    PreviousStatus string

    // Parent entity titles for hierarchical context
    ParentTitle      string // feature.title for tasks, epic.title for features
    GrandparentTitle string // epic.title for tasks (empty for features/epics)

    // Latest note from entity_notes
    LatestNoteContent string
    LatestNoteType    string

    // Note counts
    NotesCount     int
    RejectionCount int

    // Sibling progress (children of the same parent)
    SiblingTotal     int
    SiblingCompleted int
    SiblingBlocked   int
}
```

**Design decision**: This is a flat value struct (not a pointer to a nested model) because it maps 1:1 to template variables. Using a struct rather than extending function signatures with additional repository interfaces keeps parameter count manageable and allows nil-safe optional usage.

---

## 5. Consolidated Repository Method

### 5.1 Interface

Define a new interface in `internal/config/template_helpers.go` alongside the existing repository interfaces:

```go
// TemplateEnrichmentRepository provides consolidated enrichment data
// for template variable population. Implementations should fetch all
// data in a single query to minimize Turso round-trips.
type TemplateEnrichmentRepository interface {
    GetTaskEnrichment(ctx context.Context, taskID int64) (*TemplateEnrichmentData, error)
    GetFeatureEnrichment(ctx context.Context, featureID int64) (*TemplateEnrichmentData, error)
    GetEpicEnrichment(ctx context.Context, epicID int64) (*TemplateEnrichmentData, error)
}
```

### 5.2 Task Enrichment Query

Single query using subqueries and JOINs against existing tables:

```sql
SELECT
    -- previous_status: most recent old_status from task_history
    COALESCE(
        (SELECT old_status FROM task_history
         WHERE task_id = t.id ORDER BY timestamp DESC LIMIT 1),
        ''
    ) AS previous_status,

    -- parent_title: feature title
    COALESCE(f.title, '') AS parent_title,

    -- grandparent_title: epic title
    COALESCE(e.title, '') AS grandparent_title,

    -- latest_note_content
    COALESCE(
        (SELECT content FROM entity_notes
         WHERE entity_type = 'task' AND entity_id = t.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_content,

    -- latest_note_type
    COALESCE(
        (SELECT note_type FROM entity_notes
         WHERE entity_type = 'task' AND entity_id = t.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_type,

    -- notes_count
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'task' AND entity_id = t.id
    ) AS notes_count,

    -- rejection_count
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'task' AND entity_id = t.id
     AND note_type = 'rejection'
    ) AS rejection_count,

    -- sibling_total: all tasks in same feature
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = t.feature_id
    ) AS sibling_total,

    -- sibling_completed
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = t.feature_id AND status = 'completed'
    ) AS sibling_completed,

    -- sibling_blocked
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = t.feature_id AND status = 'blocked'
    ) AS sibling_blocked

FROM tasks t
LEFT JOIN features f ON t.feature_id = f.id
LEFT JOIN epics e ON f.epic_id = e.id
WHERE t.id = ?
```

This is a single round-trip to the database. All subqueries operate on indexed columns (`task_id`, `entity_type + entity_id`, `feature_id`, `status`).

### 5.3 Feature Enrichment Query

```sql
SELECT
    -- previous_status: not available for features in v1 (no feature_history table)
    '' AS previous_status,

    -- parent_title: epic title
    COALESCE(e.title, '') AS parent_title,

    -- grandparent_title: not applicable for features
    '' AS grandparent_title,

    -- latest_note_content
    COALESCE(
        (SELECT content FROM entity_notes
         WHERE entity_type = 'feature' AND entity_id = f.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_content,

    -- latest_note_type
    COALESCE(
        (SELECT note_type FROM entity_notes
         WHERE entity_type = 'feature' AND entity_id = f.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_type,

    -- notes_count
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'feature' AND entity_id = f.id
    ) AS notes_count,

    -- rejection_count
    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'feature' AND entity_id = f.id
     AND note_type = 'rejection'
    ) AS rejection_count,

    -- sibling_total: all tasks in this feature
    (SELECT COUNT(*) FROM tasks WHERE feature_id = f.id) AS sibling_total,

    -- sibling_completed
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = f.id AND status = 'completed'
    ) AS sibling_completed,

    -- sibling_blocked
    (SELECT COUNT(*) FROM tasks
     WHERE feature_id = f.id AND status = 'blocked'
    ) AS sibling_blocked

FROM features f
LEFT JOIN epics e ON f.epic_id = e.id
WHERE f.id = ?
```

### 5.4 Epic Enrichment Query

```sql
SELECT
    '' AS previous_status,
    '' AS parent_title,
    '' AS grandparent_title,

    COALESCE(
        (SELECT content FROM entity_notes
         WHERE entity_type = 'epic' AND entity_id = e.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_content,

    COALESCE(
        (SELECT note_type FROM entity_notes
         WHERE entity_type = 'epic' AND entity_id = e.id
         ORDER BY created_at DESC LIMIT 1),
        ''
    ) AS latest_note_type,

    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'epic' AND entity_id = e.id
    ) AS notes_count,

    (SELECT COUNT(*) FROM entity_notes
     WHERE entity_type = 'epic' AND entity_id = e.id
     AND note_type = 'rejection'
    ) AS rejection_count,

    -- sibling = features in this epic
    (SELECT COUNT(*) FROM features WHERE epic_id = e.id) AS sibling_total,
    (SELECT COUNT(*) FROM features
     WHERE epic_id = e.id AND status = 'completed'
    ) AS sibling_completed,
    (SELECT COUNT(*) FROM features
     WHERE epic_id = e.id AND status = 'blocked'
    ) AS sibling_blocked

FROM epics e
WHERE e.id = ?
```

### 5.5 Repository Implementation Location

Create a new file `internal/repository/template_enrichment_repository.go` containing the concrete implementation. This keeps enrichment queries separate from the existing entity repositories, which remain focused on CRUD operations.

---

## 6. Template Helper Extensions

### 6.1 Modified Function Signatures

Add an optional `*TemplateEnrichmentData` parameter to each `*PlaceholdersWithRelated()` function. Passing `nil` preserves backward compatibility -- callers that do not have enrichment data continue to work identically.

```go
func TaskPlaceholdersWithRelated(
    ctx context.Context,
    task *models.Task,
    docRepo DocumentRepository,
    taskRelRepo TaskRelationshipRepository,
    enrichment *TemplateEnrichmentData,  // NEW: optional, nil-safe
) map[string]string

func FeaturePlaceholdersWithRelated(
    ctx context.Context,
    feature *models.Feature,
    docRepoForFeature interface {
        ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error)
    },
    featureRelRepo FeatureRelationshipRepository,
    enrichment *TemplateEnrichmentData,  // NEW: optional, nil-safe
) map[string]string

func EpicPlaceholdersWithRelated(
    epic *models.Epic,
    docRepoForEpic interface {
        ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error)
    },
    epicRelRepo EpicRelationshipRepository,
    ctx context.Context,
    enrichment *TemplateEnrichmentData,  // NEW: optional, nil-safe
) map[string]string
```

**Call site update**: All 6 call sites pass `nil` initially. Services that construct enrichment data pass the populated struct.

### 6.2 Enrichment Application

A new helper function merges enrichment data into the placeholder map:

```go
func applyEnrichmentData(enrichment *TemplateEnrichmentData, placeholders map[string]string) {
    if enrichment == nil {
        return
    }

    if enrichment.PreviousStatus != "" {
        placeholders["previous_status"] = enrichment.PreviousStatus
    }
    if enrichment.ParentTitle != "" {
        placeholders["parent_title"] = enrichment.ParentTitle
    }
    if enrichment.GrandparentTitle != "" {
        placeholders["grandparent_title"] = enrichment.GrandparentTitle
    }
    if enrichment.LatestNoteContent != "" {
        placeholders["latest_note"] = enrichment.LatestNoteContent
        placeholders["latest_note_type"] = enrichment.LatestNoteType
    }

    placeholders["notes_count"] = fmt.Sprintf("%d", enrichment.NotesCount)
    placeholders["rejection_count"] = fmt.Sprintf("%d", enrichment.RejectionCount)
    placeholders["sibling_total"] = fmt.Sprintf("%d", enrichment.SiblingTotal)
    placeholders["sibling_completed"] = fmt.Sprintf("%d", enrichment.SiblingCompleted)
    placeholders["sibling_blocked"] = fmt.Sprintf("%d", enrichment.SiblingBlocked)
}
```

### 6.3 Extended Context Data Extraction

Extend `extractContextDataMetadata()` (or rename to `extractContextDataFields()`) to also extract structured fields from `ContextData`:

```go
func extractContextDataFields(contextData *string, placeholders map[string]string) {
    if contextData == nil || *contextData == "" {
        return
    }

    var cd models.ContextData
    if err := json.Unmarshal([]byte(*contextData), &cd); err != nil {
        return // Graceful degradation
    }

    // Existing: flatten Metadata key-value pairs
    for key, value := range cd.Metadata {
        str := stringifyMetadataValue(value)
        if str != "" {
            placeholders[key] = str
        }
    }

    // NEW: Extract structured progress fields
    if cd.Progress != nil {
        if cd.Progress.CurrentStep != nil {
            placeholders["current_step"] = *cd.Progress.CurrentStep
        }
        if len(cd.Progress.CompletedSteps) > 0 {
            placeholders["completed_steps"] = strings.Join(cd.Progress.CompletedSteps, ", ")
        }
        if len(cd.Progress.RemainingSteps) > 0 {
            placeholders["remaining_steps"] = strings.Join(cd.Progress.RemainingSteps, ", ")
        }
        placeholders["completed_steps_count"] = fmt.Sprintf("%d", len(cd.Progress.CompletedSteps))
        placeholders["remaining_steps_count"] = fmt.Sprintf("%d", len(cd.Progress.RemainingSteps))
    }

    // NEW: Extract open questions
    if len(cd.OpenQuestions) > 0 {
        placeholders["open_questions"] = strings.Join(cd.OpenQuestions, "; ")
        placeholders["open_questions_count"] = fmt.Sprintf("%d", len(cd.OpenQuestions))
    }

    // NEW: Extract blockers summary
    if len(cd.Blockers) > 0 {
        placeholders["blockers_count"] = fmt.Sprintf("%d", len(cd.Blockers))
        // First blocker description for quick reference
        placeholders["latest_blocker"] = cd.Blockers[len(cd.Blockers)-1].Description
    }

    // NEW: Extract implementation decisions count
    if len(cd.ImplementationDecisions) > 0 {
        placeholders["decisions_count"] = fmt.Sprintf("%d", len(cd.ImplementationDecisions))
    }
}
```

**Rename decision**: Rename `extractContextDataMetadata` to `extractContextDataFields` since it now extracts more than just metadata. Update the 3 call sites within template_helpers.go (internal callers only, no exported API change).

---

## 7. Variable Naming Convention

All new template variables follow the existing flat naming pattern using snake_case:

### 7.1 Complete Variable Reference

| Variable | Source | Entity Types | Example Value |
|----------|--------|-------------|---------------|
| `previous_status` | task_history | Task only (v1) | `"in_development"` |
| `parent_title` | Feature/Epic table JOIN | Task, Feature | `"User Authentication"` |
| `grandparent_title` | Epic table JOIN | Task only | `"Security Enhancements"` |
| `latest_note` | entity_notes (most recent) | All | `"Waiting on API spec"` |
| `latest_note_type` | entity_notes (most recent) | All | `"blocker"` |
| `notes_count` | entity_notes COUNT | All | `"5"` |
| `rejection_count` | entity_notes COUNT (type=rejection) | All | `"2"` |
| `sibling_total` | tasks/features COUNT | All | `"12"` |
| `sibling_completed` | tasks/features COUNT (status=completed) | All | `"8"` |
| `sibling_blocked` | tasks/features COUNT (status=blocked) | All | `"1"` |
| `current_step` | ContextData.Progress | All | `"Implementing API"` |
| `completed_steps` | ContextData.Progress | All | `"Design, DB Schema"` |
| `remaining_steps` | ContextData.Progress | All | `"Tests, Review"` |
| `completed_steps_count` | ContextData.Progress | All | `"2"` |
| `remaining_steps_count` | ContextData.Progress | All | `"2"` |
| `open_questions` | ContextData.OpenQuestions | All | `"Auth provider?; Rate limiting?"` |
| `open_questions_count` | ContextData.OpenQuestions | All | `"2"` |
| `blockers_count` | ContextData.Blockers | All | `"1"` |
| `latest_blocker` | ContextData.Blockers (last) | All | `"Waiting on API spec"` |
| `decisions_count` | ContextData.ImplementationDecisions | All | `"3"` |

### 7.2 Template Usage Examples

```
{{.previous_status}}            -> "in_code_review"
{{.parent_title}}               -> "Template Variable Enrichment"
{{.sibling_completed}}/{{.sibling_total}} tasks done
{{.latest_note}}                -> "Code review feedback addressed"
```

### 7.3 Naming Rationale

- `parent_title` and `grandparent_title` are generic names rather than `feature_title`/`epic_title` because the parent type depends on entity level (a feature's parent is an epic, a task's parent is a feature).
- `sibling_*` uses "sibling" rather than "child" or "task" because for features the siblings are tasks, for epics the siblings are features. The perspective is from the template's entity.
- Context data variables use their existing ContextData field names (snake_case) for consistency with the metadata extraction pattern.

---

## 8. Service Layer Integration

### 8.1 Wiring Pattern

Each service that calls `*PlaceholdersWithRelated()` needs access to the enrichment repository. The enrichment repository is added as an optional dependency (nil-safe), following the established pattern for `docRepo` and `relRepo`.

**TaskService constructor change** (example):

```go
type TaskService struct {
    // ... existing fields ...
    enrichRepo config.TemplateEnrichmentRepository // optional
}

func NewTaskService(
    // ... existing params ...
    enrichRepo config.TemplateEnrichmentRepository,
) *TaskService {
    return &TaskService{
        // ... existing assignments ...
        enrichRepo: enrichRepo,
    }
}
```

**resolveAction() update** (example for TaskService):

```go
func (s *TaskService) resolveAction(ctx context.Context, task *models.Task, status string) *config.PopulatedAction {
    // ... existing workflow/metadata lookup ...

    // Fetch enrichment data (optional, graceful degradation)
    var enrichment *config.TemplateEnrichmentData
    if s.enrichRepo != nil {
        data, err := s.enrichRepo.GetTaskEnrichment(ctx, task.ID)
        if err != nil {
            log.Printf("WARNING: Failed to fetch enrichment data for task %s: %v", task.Key, err)
        } else {
            enrichment = data
        }
    }

    var placeholders map[string]string
    if s.docRepo != nil && s.relRepo != nil {
        placeholders = config.TaskPlaceholdersWithRelated(ctx, task, s.docRepo, s.relRepo, enrichment)
    } else {
        placeholders = config.TaskPlaceholders(task)
    }

    // ... existing action population ...
}
```

### 8.2 CLI Wiring (services_global.go)

The `GetTaskService()`, `GetFeatureService()`, and `GetEpicService()` global accessors construct the enrichment repository from the shared DB:

```go
func GetTaskService() *services.TaskService {
    db, _ := GetDB(context.Background())
    // ... existing repos ...
    enrichRepo := repository.NewTemplateEnrichmentRepository(db)
    return services.NewTaskService(/* existing params */, enrichRepo)
}
```

### 8.3 DisplayService Wiring

The `DisplayService.Dependencies` struct gains a `TemplateEnrichmentRepo` field. The three `Resolve*Action()` methods pass enrichment data the same way as the entity services.

---

## 9. Implementation Order

Tasks are ordered to maximize incremental value and minimize risk:

### Phase 1: Zero-Query Enrichment (context_data structured fields)

**Task 1**: Extend `extractContextDataMetadata()` to extract structured fields (Progress, OpenQuestions, Blockers, ImplementationDecisions). Rename to `extractContextDataFields()`.

- **Files**: `internal/config/template_helpers.go`, `internal/config/template_helpers_test.go`
- **Dependencies**: None
- **Risk**: Low (internal function, no signature changes)
- **Impact**: High (eliminates CLI round-trips for progress/blocker context in templates)

### Phase 2: Enrichment Infrastructure

**Task 2**: Create `TemplateEnrichmentData` struct and `TemplateEnrichmentRepository` interface in `internal/config/template_helpers.go`. Create concrete implementation in `internal/repository/template_enrichment_repository.go` with the consolidated queries.

- **Files**: `internal/config/template_helpers.go`, `internal/repository/template_enrichment_repository.go` (new), `internal/repository/template_enrichment_repository_test.go` (new)
- **Dependencies**: None
- **Risk**: Low (new code, no changes to existing)

**Task 3**: Modify `*PlaceholdersWithRelated()` signatures to accept optional `*TemplateEnrichmentData`. Add `applyEnrichmentData()` helper. Update all 6 call sites to pass `nil` (backward-compatible no-op).

- **Files**: `internal/config/template_helpers.go`, `internal/config/template_helpers_test.go`, `internal/services/task_service.go`, `internal/services/feature_service.go`, `internal/services/epic_service.go`, `internal/services/display_service.go`
- **Dependencies**: Task 2
- **Risk**: Medium (signature change touches 6 call sites, but nil pass is safe)

### Phase 3: Wire Enrichment

**Task 4**: Wire enrichment repository into TaskService, FeatureService, EpicService, DisplayService. Update `resolveAction()` methods to fetch and pass enrichment data. Update global accessors.

- **Files**: `internal/services/task_service.go`, `internal/services/feature_service.go`, `internal/services/epic_service.go`, `internal/services/display_service.go`, `internal/cli/services_global.go`
- **Dependencies**: Task 3
- **Risk**: Low (additive, optional dependency, graceful degradation on error)

### Phase 4: Testing and Validation

**Task 5**: Integration tests for enrichment repository queries. Unit tests for enrichment application. End-to-end validation with sample templates.

- **Files**: Test files
- **Dependencies**: Task 4

---

## 10. Turso Performance Analysis

### Query Budget

| Operation | Queries (Before) | Queries (After) | Delta |
|-----------|-----------------|-----------------|-------|
| Template render (task) | 2 (docs + rels) | 3 (docs + rels + enrichment) | +1 |
| Template render (feature) | 2 (docs + rels) | 3 (docs + rels + enrichment) | +1 |
| Template render (epic) | 2 (docs + rels) | 3 (docs + rels + enrichment) | +1 |

**Net cost**: +1 query per template render. This single query replaces what would be 5-7 separate queries (previous_status, feature lookup, epic lookup, latest note, note counts, sibling counts) without consolidation.

### Index Coverage

All subqueries in the consolidated query hit existing indexes:
- `task_history(task_id)` -- `idx_task_history_task_id`
- `entity_notes(entity_type, entity_id, note_type)` -- `idx_entity_notes_entity_type`
- `tasks(feature_id)` -- implied by foreign key or existing composite indexes
- `features(epic_id)` -- implied by foreign key

### Estimated Latency

Single consolidated query with indexed subqueries: ~5-15ms on Turso (comparable to existing display view queries). No performance regression expected.

---

## 11. Backward Compatibility

### Template Compatibility

- **Existing templates**: Unchanged. New variables are additive; templates that do not reference `{{.previous_status}}` etc. are unaffected.
- **Missing variables**: If enrichment data is not available (nil enrichment, nil enrichment repo), the variables are simply absent from the placeholder map. Go's `text/template` renders missing keys as `<no value>` (or empty with `{{with}}`), and `PopulateTemplate()` leaves `{{.key}}` unreplaced. This matches existing behavior for optional variables.

### API Compatibility

- **Function signatures**: `*PlaceholdersWithRelated()` gains a new trailing parameter. All existing call sites must be updated to pass `nil`. This is a compile-time break that is trivially fixed.
- **Service constructors**: Gain an optional `enrichRepo` parameter. Passing `nil` preserves existing behavior.
- **No database schema changes**: No migrations needed.

---

## 12. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Signature change breaks call sites | Certain | Low | All 6 call sites are internal; update in same PR with `nil` as default |
| Consolidated query too slow on Turso | Low | Medium | All subqueries use indexed columns; can fall back to separate queries if needed |
| `previous_status` unavailable for features/epics | N/A | Low | Explicitly scoped to tasks in v1; features/epics get empty string |
| Context data fields conflict with Metadata keys | Low | Low | Structured fields use distinct names (`current_step`, `completed_steps`) that are unlikely to collide with user metadata; Metadata extraction runs first, structured fields do not overwrite existing keys |

---

## 13. Future Considerations

- **Feature/Epic history**: When `feature_history` and `epic_history` tables are added, the enrichment queries can be extended to provide `previous_status` for those entity types.
- **Template variable documentation**: A `shark template vars <entity-type>` command could list all available variables for a given entity type.
- **Enrichment caching**: If template rendering is called multiple times for the same entity in a single command, the enrichment data could be cached in the service. Not needed for v1 since each `resolveAction()` call is per-entity.

---

## References

- `internal/config/template_helpers.go` -- Primary implementation target
- `internal/models/context_data.go` -- ContextData struct definition
- `internal/repository/task_history_repository.go` -- task_history queries
- `internal/repository/entity_note_repository.go` -- entity_notes queries
- Feature E07-F29 -- Prior template variable work (related_docs, related_tasks)
- Feature E07-F33 -- Prior template variable work (context_data metadata)
- Research report: `docs/plan/E07-enhancements/E07-F34-template-variable-enrichment/F01-feature-context.md`
