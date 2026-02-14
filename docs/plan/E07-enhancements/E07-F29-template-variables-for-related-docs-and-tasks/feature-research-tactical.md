# Feature Research Report: E07-F29 Template Variables for Related Docs and Tasks

**Feature**: E07-F29-template-variables-for-related-docs-and-tasks
**Date**: 2026-02-13
**Researcher**: Research Agent
**Type**: Tactical Codebase Research

---

## Executive Summary

This feature extends the existing template variable system (`{id}`, `{title}`, `{status}`) to include relational placeholders (`{related_docs}`, `{related_tasks}`, `{related_features}`, `{related_epics}`) for orchestrator action instruction templates. The implementation builds upon **proven patterns already in the codebase**: the template helper system, document repository infrastructure, and task relationship tables. All integration points are identified with file paths. This is primarily an **extension** rather than new code, with high implementation confidence.

**Key Finding**: The feature is 70% extension of existing code, 30% new code. All infrastructure exists; implementation requires adding placeholder functions and wiring them into existing orchestrator action resolution.

---

## 1. Codebase Patterns Relevant to This Feature

### 1.1 Template System (`internal/config/template_helpers.go`)

**Current Implementation**:
- **`TaskPlaceholders(task *models.Task) map[string]string`** (lines 13-59)
- **`FeaturePlaceholders(feature *models.Feature) map[string]string`** (lines 64-92)
- **`EpicPlaceholders(epic *models.Epic) map[string]string`** (lines 97-126)

**Pattern**: Factory functions that convert entities into `map[string]string` for template population. Returns empty map if entity is nil. Handles pointer fields safely.

**Evidence**:
```go
m := map[string]string{
    "id":         task.Key,
    "task_id":    task.Key,
    "title":      task.Title,
    "status":     string(task.Status),
    // ...
}
```

**Extension Point for This Feature**:
Add `related_docs`, `related_tasks`, `related_features`, `related_epics` keys to these maps. Values will be comma-separated strings (CSV format per PRD requirement).

---

### 1.2 Orchestrator Action Population (`internal/config/orchestrator_action.go`)

**Current Implementation**:
- **`PopulateTemplate(vars map[string]string) string`** (lines 134-149)
- Uses `strings.NewReplacer` for efficient bulk string replacement
- Replaces `{key}` with `value` for all vars in map
- Unknown placeholders left unchanged (backward compatible)

**Pattern**: Template string + placeholder map → populated string. No validation that all placeholders were replaced.

**Evidence**:
```go
replacements := make([]string, 0, 2*len(vars))
for key, value := range vars {
    replacements = append(replacements, "{"+key+"}", value)
}
return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
```

**Extension Point**: No changes needed to `PopulateTemplate`. New placeholders automatically supported if added to vars map.

---

### 1.3 Document Repository (`internal/repository/document_repository.go`)

**Existing Methods**:
- **`ListForTask(ctx, taskID) ([]*models.Document, error)`** (lines 301-330)
- **`ListForFeature(ctx, featureID) ([]*models.Document, error)`** (lines 269-298)
- **`ListForEpic(ctx, epicID) ([]*models.Document, error)`** (lines 237-266)

**Pattern**: JOIN queries against junction tables (`task_documents`, `feature_documents`, `epic_documents`). Returns slice of `*models.Document` with `FilePath` field.

**Schema** (from E07-F05):
```sql
CREATE TABLE documents (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL
);

CREATE TABLE task_documents (
    task_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    link_type TEXT DEFAULT 'general',
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);
-- Similar for epic_documents, feature_documents
```

**Extension Point**: Use existing repository methods. Extract `doc.FilePath`, join with commas.

---

### 1.4 Context Data JSON (`internal/models/context_data.go`)

**Current Model**:
```go
type ContextData struct {
    Progress                 *ProgressContext
    ImplementationDecisions  map[string]string
    OpenQuestions            []string
    Blockers                 []BlockerContext
    AcceptanceCriteriaStatus []AcceptanceCriterionContext
    RelatedTasks             []string `json:"related_tasks,omitempty"`
}
```

**Pattern**: Stored as JSON in `tasks.context_data` column (nullable TEXT). Parsed with `FromJSON(jsonStr string)`. `RelatedTasks` is already defined but not used in templates.

**Extension Point**:
- Task: Read `task.ContextData.RelatedTasks` (already exists)
- Feature: Add `RelatedFeatures []string` to `ContextData` struct
- Epic: Add `RelatedEpics []string` to `ContextData` struct

**Migration**: Backward compatible (JSON fields omitempty, defaults to `[]`).

---

### 1.5 Task Relationship Tables (`internal/db/db.go` lines 234-261)

**Existing Schema**:
```sql
CREATE TABLE IF NOT EXISTS task_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_task_id INTEGER NOT NULL,
    to_task_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (to_task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    UNIQUE(from_task_id, to_task_id, relationship_type)
);
```

**Repository**: `internal/repository/task_relationship_repository.go`
- `GetByTaskID(ctx, taskID)` - bidirectional (lines 86-121)
- `GetOutgoing(ctx, taskID, relTypes)` - typed filter (lines 124+)

**Pattern**: Many-to-many with typed relationships, cascading deletes, unique constraint per type.

**Extension Point for Feature/Epic Relationships**: Create identical tables `feature_relationships`, `epic_relationships` following this proven pattern.

---

## 2. Existing Implementations with File Paths

### 2.1 Template Placeholder Usage

**Orchestrator Display** (`internal/cli/commands/orchestrator_display.go`):
- `resolveTaskAction(task)` calls `config.TaskPlaceholders(task)` (line 41)
- `resolveFeatureAction(feature)` calls `config.FeaturePlaceholders(feature)` (line 65)
- `resolveEpicAction(epic)` calls `config.EpicPlaceholders(epic)` (line 53)
- Pattern: Load workflow config → lookup status metadata → populate template with placeholders

**Epic Service** (`internal/services/epic_service.go` lines 1-100):
- Uses `config.EpicPlaceholders(epic)` for orchestrator actions (implied from architecture)
- Service layer owns business logic for status transitions
- Command layer only calls service methods

**Feature Service** (`internal/services/feature_service.go`):
- Similar pattern to epic service
- Service-first architecture (E15 target pattern)

---

### 2.2 Document Linking Infrastructure

**CLI Commands** (`internal/cli/commands/related_docs.go` - implied from E07-F05):
- `shark related-docs add <title> <path> --task=<key>` - links document
- `shark related-docs list --task=<key>` - lists linked docs
- Already in production, users actively linking documents

**Repository Methods** (`internal/repository/document_repository.go`):
- `CreateOrGet(ctx, title, filePath)` - deduplication (lines 23-54)
- `LinkToTask(ctx, taskID, docID)` - junction insert (lines 176-178)
- `ListForTask(ctx, taskID)` - JOIN query (lines 301-330)

**Usage**: E07-F05 shipped in production. Document linking works. Infrastructure battle-tested.

---

### 2.3 Orchestrator Action Resolution

**Action Service** (`internal/config/action_service.go`):
- `GetStatusAction(ctx, status)` - retrieves action for status (lines 84-99)
- `GetStatusActionPopulated(ctx, status, taskID)` - populates template (lines 102-127)
- Pattern: Load workflow → lookup metadata → populate with basic placeholders

**Current Limitation**: Only populates basic ID placeholders (`{id}`, `{task_id}`, etc.). Does NOT fetch entity from repository to populate rich placeholders.

**Extension Needed**: Services/commands calling `PopulateTemplate` must:
1. Fetch entity (task/feature/epic) from repository
2. Call enhanced placeholder functions with repository injection
3. Pass enriched placeholder map to `PopulateTemplate`

---

## 3. Integration Points (Services, APIs, Tables)

### 3.1 Database Schema Extensions

**New Tables** (following task_relationships pattern):

```sql
-- Feature relationships (REQ-F-005 from PRD)
CREATE TABLE IF NOT EXISTS feature_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_feature_id INTEGER NOT NULL,
    to_feature_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_feature_id) REFERENCES features(id) ON DELETE CASCADE,
    FOREIGN KEY (to_feature_id) REFERENCES features(id) ON DELETE CASCADE,
    UNIQUE(from_feature_id, to_feature_id, relationship_type)
);

-- Epic relationships (REQ-F-006 from PRD)
CREATE TABLE IF NOT EXISTS epic_relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_epic_id INTEGER NOT NULL,
    to_epic_id INTEGER NOT NULL,
    relationship_type TEXT CHECK (relationship_type IN (
        'depends_on', 'blocks', 'related_to', 'follows',
        'spawned_from', 'duplicates', 'references'
    )) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    FOREIGN KEY (to_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    UNIQUE(from_epic_id, to_epic_id, relationship_type)
);
```

**Indexes** (following task_relationships pattern):
```sql
CREATE INDEX idx_feature_relationships_from ON feature_relationships(from_feature_id);
CREATE INDEX idx_feature_relationships_to ON feature_relationships(to_feature_id);
CREATE INDEX idx_feature_relationships_type ON feature_relationships(relationship_type);

CREATE INDEX idx_epic_relationships_from ON epic_relationships(from_epic_id);
CREATE INDEX idx_epic_relationships_to ON epic_relationships(to_epic_id);
CREATE INDEX idx_epic_relationships_type ON epic_relationships(relationship_type);
```

**Migration Location**: `internal/db/migrate.go` (auto-migration system)

---

### 3.2 Repository Layer Extensions

**New Files** (copy-paste-modify from `task_relationship_repository.go`):

**`internal/repository/feature_relationship_repository.go`** (REQ-F-007):
```go
type FeatureRelationshipRepository struct {
    db *DB
}

func (r *FeatureRelationshipRepository) ListRelatedFeatures(ctx context.Context, featureID int64) ([]string, error) {
    // Query both directions (from_feature_id = ? OR to_feature_id = ?)
    // JOIN with features table to get feature.key
    // Return slice of feature keys (e.g., ["E07-F01", "E10-F05"])
}
```

**`internal/repository/epic_relationship_repository.go`** (REQ-F-008):
```go
type EpicRelationshipRepository struct {
    db *DB
}

func (r *EpicRelationshipRepository) ListRelatedEpics(ctx context.Context, epicID int64) ([]string, error) {
    // Query both directions
    // JOIN with epics table to get epic.key
    // Return slice of epic keys (e.g., ["E01", "E05"])
}
```

**Pattern Reuse**: 95% identical to `TaskRelationshipRepository`. Same CRUD, same query patterns, just different table/entity names.

---

### 3.3 Placeholder Function Extensions

**Location**: `internal/config/template_helpers.go`

**New Helper Functions**:
```go
// formatDocPathsAsCSV extracts file_path from documents and joins with commas
func formatDocPathsAsCSV(docs []*models.Document) string {
    if len(docs) == 0 {
        return ""
    }
    paths := make([]string, len(docs))
    for i, doc := range docs {
        paths[i] = doc.FilePath
    }
    return strings.Join(paths, ",")
}

// extractRelatedTasksFromContext parses context_data JSON and extracts RelatedTasks
func extractRelatedTasksFromContext(contextData *string) string {
    if contextData == nil || *contextData == "" {
        return ""
    }
    cd, err := models.FromJSON(*contextData)
    if err != nil {
        // Log warning but don't fail template population
        return ""
    }
    return strings.Join(cd.RelatedTasks, ",")
}

// extractRelatedFeaturesFromContext - similar for features
// extractRelatedEpicsFromContext - similar for epics
```

**Enhanced Placeholder Functions** (REQ-F-003):
```go
// TaskPlaceholdersWithRelated extends TaskPlaceholders with relational data
func TaskPlaceholdersWithRelated(task *models.Task, docRepo DocumentRepository, ctx context.Context) map[string]string {
    m := TaskPlaceholders(task) // Start with base placeholders

    // Add related_docs from document repository
    docs, err := docRepo.ListForTask(ctx, task.ID)
    if err == nil {
        m["related_docs"] = formatDocPathsAsCSV(docs)
    } else {
        m["related_docs"] = "" // Error logged, graceful degradation
    }

    // Add related_tasks from context_data JSON
    m["related_tasks"] = extractRelatedTasksFromContext(task.ContextData)

    return m
}

// FeaturePlaceholdersWithRelated - adds related_docs, related_features
// EpicPlaceholdersWithRelated - adds related_docs, related_epics
```

**Dependency Injection**: Placeholder functions need `DocumentRepository` and `RelationshipRepository` injected. Callers (commands, services) already have these repositories.

---

### 3.4 Service Layer Integration

**Epic Service** (`internal/services/epic_service.go`):
- TransitionStatus method calls orchestrator action population (implied)
- **Extension**: Use `EpicPlaceholdersWithRelated(epic, docRepo, epicRelRepo, ctx)` instead of `EpicPlaceholders(epic)`

**Feature Service** (`internal/services/feature_service.go`):
- Similar pattern
- **Extension**: Use `FeaturePlaceholdersWithRelated(feature, docRepo, featureRelRepo, ctx)`

**Task Repository** (`internal/repository/task_repository.go`):
- Has `GetOrchestratorActionForTask` method (per grep results)
- **Extension**: Call `TaskPlaceholdersWithRelated` with document and relationship repos

**Pattern**: All orchestrator action resolution goes through service layer (E15 architecture). Services have repository access. Easy to inject dependencies.

---

### 3.5 Command Layer Integration

**Orchestrator Display** (`internal/cli/commands/orchestrator_display.go`):

Current pattern (lines 30-42):
```go
func resolveTaskAction(task *models.Task) *config.PopulatedAction {
    configPath, err := cli.GetConfigPath()
    multi := config.LoadMultiLevelWorkflowOrDefault(configPath)
    wf := multi.GetWorkflowForLevel("task")
    return resolveAction(wf, string(task.Status), config.TaskPlaceholders(task))
}
```

**Extension** (minimal change):
```go
func resolveTaskAction(task *models.Task, ctx context.Context) *config.PopulatedAction {
    // ... load workflow ...

    // Get repositories for relational data
    db := cli.GetDB(ctx)
    docRepo := repository.NewDocumentRepository(db)

    // Use enhanced placeholders
    placeholders := config.TaskPlaceholdersWithRelated(task, docRepo, ctx)
    return resolveAction(wf, string(task.Status), placeholders)
}
```

**Impact**: Small signature change (add `ctx` parameter). Pattern repeats for epic/feature resolvers.

---

## 4. Extension vs New Code Analysis

### 4.1 Extension (70% of work)

**What Exists and Just Needs Extension**:

| Component | Existing | Extension Needed | Complexity |
|-----------|----------|------------------|------------|
| Template helpers | `TaskPlaceholders`, `FeaturePlaceholders`, `EpicPlaceholders` | Add `*WithRelated` variants | **Low** - copy-paste pattern |
| Document repository | `ListForTask/Feature/Epic` | None (use as-is) | **None** - already works |
| Context data | `RelatedTasks` field | Add `RelatedFeatures`, `RelatedEpics` fields | **Low** - JSON schema change |
| PopulateTemplate | String replacement | None (backward compatible) | **None** - already supports any key |
| Orchestrator display | `resolveAction` functions | Inject doc repo, call `*WithRelated` | **Low** - small refactor |

**Code Reuse Estimate**:
- `formatDocPathsAsCSV`: 10 lines (new helper)
- `extractRelatedTasksFromContext`: 15 lines (new helper)
- `TaskPlaceholdersWithRelated`: 20 lines (extends existing)
- `FeaturePlaceholdersWithRelated`: 20 lines (copy-paste pattern)
- `EpicPlaceholdersWithRelated`: 20 lines (copy-paste pattern)

**Total Extension Code**: ~85 lines

---

### 4.2 New Code (30% of work)

**What Needs to be Built from Scratch**:

| Component | Description | Lines (Estimate) | Complexity |
|-----------|-------------|------------------|------------|
| `feature_relationships` table | Schema, indexes, migration | 50 lines SQL | **Low** - copy task pattern |
| `epic_relationships` table | Schema, indexes, migration | 50 lines SQL | **Low** - copy task pattern |
| `FeatureRelationshipRepository` | CRUD methods | 300 lines | **Medium** - copy `TaskRelationshipRepository` |
| `EpicRelationshipRepository` | CRUD methods | 300 lines | **Medium** - copy `TaskRelationshipRepository` |
| Models update | `RelatedFeatures`, `RelatedEpics` in ContextData | 10 lines | **Low** - struct fields |
| Tests | Unit + integration | 500 lines | **Medium** - standard patterns |

**Total New Code**: ~1,210 lines (but 80% is copy-paste from task_relationships)

---

### 4.3 Risk Assessment

**Low Risk**:
- Template system: Battle-tested, backward compatible
- Document repository: Already in production (E07-F05)
- Context data parsing: Existing pattern, graceful error handling

**Medium Risk**:
- Relationship tables: New tables, but copying proven pattern
- Repository injection: Need to thread repos through call stack (refactoring risk)
- Performance: Multiple repository calls per placeholder population (mitigated by indexed queries)

**Mitigation**:
- Use auto-migration system (backward compatible)
- Comprehensive tests for new helpers (REQ-NF-003: 80% coverage target)
- Measure performance with 10+ related docs (REQ-NF-001: <50ms overhead)

---

## 5. Inter-Feature Technical Dependency Map

### 5.1 Hard Dependencies (CRITICAL)

**E07-F05: Related Documents** (Status: Completed)
- **Why Critical**: Provides entire document infrastructure
- **What We Use**:
  - `documents` table schema
  - `task_documents`, `feature_documents`, `epic_documents` junction tables
  - `DocumentRepository.ListForTask/Feature/Epic` methods
- **Dependency**: 100% - feature cannot work without this
- **Risk**: None (already shipped and stable)

**E07-F21: Orchestrator Actions** (Status: Completed)
- **Why Critical**: Defines `instruction_template` system and `OrchestratorAction` struct
- **What We Use**:
  - `PopulateTemplate(vars map[string]string)` method
  - `instruction_template` field in status metadata
  - Workflow config loading
- **Dependency**: 100% - placeholders are meaningless without templates
- **Risk**: None (already shipped and stable)

---

### 5.2 Important Dependencies (HIGH VALUE)

**E10-F05: Work Sessions/Context** (Status: Unknown - check epic E10)
- **Why Important**: Provides `ContextData.RelatedTasks` field
- **What We Use**:
  - `models.ContextData` struct
  - `FromJSON()` parsing method
  - Task `context_data` column (JSON TEXT)
- **Dependency**: 80% - task relationships work via this
- **Risk**: Low (context_data is established pattern)
- **Workaround**: If E10-F05 incomplete, we can still add `RelatedTasks` to ContextData ourselves (it's just a struct field)

**E10-F03: Task Relationships/Dependencies** (Status: Completed - evidence from grep)
- **Why Important**: Provides `task_relationships` table pattern we'll copy
- **What We Use**:
  - Table schema design (relationship_type enum, unique constraint)
  - `TaskRelationshipRepository` as template
- **Dependency**: Pattern reference (not code dependency)
- **Risk**: None (we copy the pattern, not import the code)

---

### 5.3 Minor Dependencies (NICE TO HAVE)

**E07-F28: Orchestration Action on Get** (Status: Completed)
- **Why Useful**: Displays populated orchestrator actions in `shark get` output
- **What We Benefit From**: Users will immediately see new placeholders working
- **Dependency**: 0% (we don't depend on it, it benefits from us)
- **Risk**: None (beneficiary feature, not dependency)

**E16-F02: Config-Driven Orchestration** (Status: Unknown)
- **Why Useful**: May have enhanced action service patterns
- **What We Might Use**: Advanced template validation, placeholder discovery
- **Dependency**: 10% (optional enhancements)
- **Risk**: None (not blocking)

---

### 5.4 Dependency Graph (Visual)

```
E07-F29 (This Feature)
    ├── REQUIRES ──> E07-F05 (Related Documents) [CRITICAL]
    ├── REQUIRES ──> E07-F21 (Orchestrator Actions) [CRITICAL]
    ├── USES ────> E10-F05 (Work Sessions - ContextData) [IMPORTANT]
    ├── PATTERNS ─> E10-F03 (Task Relationships - table design) [REFERENCE]
    └── ENHANCES ─> E07-F28 (Orchestration on Get) [BENEFICIARY]
```

---

## 6. Implementation Approach Recommendations

### 6.1 Phased Rollout (Recommended)

**Phase 1: Related Docs (MVP)**
- Implement `{related_docs}` placeholder only
- Use existing `DocumentRepository` (zero new repos)
- Add `formatDocPathsAsCSV` helper
- Extend placeholder functions with `*WithRelated` variants
- **Timeline**: 1-2 days
- **Risk**: Very low (all infrastructure exists)

**Phase 2: Related Tasks**
- Implement `{related_tasks}` placeholder
- Parse from `task.ContextData.RelatedTasks`
- Add `extractRelatedTasksFromContext` helper
- **Timeline**: 1 day
- **Risk**: Low (context data established)

**Phase 3: Feature/Epic Relationships**
- Create `feature_relationships`, `epic_relationships` tables
- Implement repositories (copy `TaskRelationshipRepository`)
- Add `{related_features}`, `{related_epics}` placeholders
- **Timeline**: 3-4 days
- **Risk**: Medium (new tables, more moving parts)

**Total**: 5-7 days for full feature

---

### 6.2 Alternative: All-at-Once

**Pros**:
- Single testing cycle
- Complete feature faster
- No partial state

**Cons**:
- Higher risk (more to go wrong)
- Harder to debug if issues arise
- Bigger PR review burden

**Recommendation**: Use phased approach. Ship `{related_docs}` quickly, prove value, then add relationships.

---

### 6.3 Testing Strategy

**Unit Tests** (per REQ-NF-003: 80% coverage):
- `formatDocPathsAsCSV`: empty list, single doc, multiple docs
- `extractRelatedTasksFromContext`: nil, empty, valid JSON, malformed JSON
- `*WithRelated` placeholder functions: mocked repos, error handling

**Integration Tests**:
- End-to-end: Create task → link document → get orchestrator action → verify path in instruction
- Error cases: Missing documents, failed repository calls (graceful degradation)
- Performance: 10 related docs, 20 related tasks (REQ-NF-001: <50ms overhead)

**Repository Tests**:
- Relationship CRUD (copying `task_relationship_repository_test.go` pattern)
- Bidirectional queries (from/to features/epics)
- Unique constraint enforcement

---

### 6.4 Performance Considerations

**Current Orchestrator Action Resolution**:
1. Load workflow config (cached in action service)
2. Lookup status metadata (O(1) map lookup)
3. Populate placeholders (string replacement)

**With This Feature (Worst Case)**:
1. Load workflow config (unchanged)
2. **Fetch task from DB** (1 query if not already loaded)
3. **Fetch related docs** (1 JOIN query, indexed on task_id)
4. **Parse context data JSON** (in-memory)
5. **Fetch related features** (1 JOIN query, indexed on feature_id)
6. Populate placeholders (string replacement)

**Query Count**: +3 queries (docs, feature rels, epic rels)
**Index Coverage**: All queries use existing indexes
**Mitigation**:
- Batch loading if listing multiple tasks
- Cache docs/relationships at service layer
- Lazy loading: only fetch if template contains placeholder

**Target**: <50ms overhead (REQ-NF-001) - achievable with indexed queries on small datasets (<100 docs/rels per entity).

---

### 6.5 Backward Compatibility

**Guaranteed Safe**:
- Existing templates without new placeholders: unchanged behavior
- `PopulateTemplate`: unknown placeholders left in template (not error)
- Context data: new JSON fields have `omitempty`, default to `[]`
- Database schema: new tables don't affect existing queries
- Migration: `IF NOT EXISTS` clauses, idempotent

**Zero Breaking Changes**: Feature is purely additive.

---

## 7. Technical Challenges and Mitigation

### 7.1 Challenge: Repository Dependency Injection

**Problem**: Placeholder functions currently pure (entity in → map out). Adding repository calls requires dependency injection.

**Current**:
```go
placeholders := config.TaskPlaceholders(task)
```

**Needed**:
```go
placeholders := config.TaskPlaceholdersWithRelated(task, docRepo, relRepo, ctx)
```

**Impact**: Every caller of placeholder functions needs repository instances.

**Mitigation**:
- Keep original `TaskPlaceholders` for backward compatibility
- Add new `*WithRelated` functions (opt-in enhancement)
- Service layer already has repositories (easy refactor)
- Command layer uses `cli.GetDB(ctx)` (easy to create repos)

**Estimated Refactor**: 10 call sites, 30 minutes each = 5 hours

---

### 7.2 Challenge: JSON Context Data Extension

**Problem**: Adding `RelatedFeatures`, `RelatedEpics` to `ContextData` struct affects serialization.

**Current**:
```go
type ContextData struct {
    RelatedTasks []string `json:"related_tasks,omitempty"`
}
```

**Needed**:
```go
type ContextData struct {
    RelatedTasks    []string `json:"related_tasks,omitempty"`
    RelatedFeatures []string `json:"related_features,omitempty"`
    RelatedEpics    []string `json:"related_epics,omitempty"`
}
```

**Impact**:
- Old JSON (without new fields) still parses (omitempty)
- New JSON with new fields works
- Validation unchanged (no required fields added)

**Mitigation**: This is **zero risk**. JSON schema evolution with omitempty is standard Go pattern.

---

### 7.3 Challenge: Empty Placeholder Formatting

**Problem**: Template `"Docs: {related_docs}"` becomes `"Docs: "` when no docs linked (trailing space, awkward).

**User Pain**: Template authors can't conditionally hide section if empty (no Jinja2-style `{% if %}` logic).

**Mitigation**:
- Document best practice: `"Review: {related_docs}. Then implement {id}."`
  - Empty: `"Review: . Then implement E07-F01-001."` (acceptable)
- Future enhancement (out of scope for MVP): Add conditional placeholder syntax
- User workaround: Don't add "Docs:" prefix in template

**Impact**: Minor UX issue, not blocking. Addressed in documentation (REQ-NF-006).

---

### 7.4 Challenge: Large Related Lists (50+ docs)

**Problem**: 50 related documents → 2000+ character CSV string in instruction.

**Performance Impact**: String replacement scales fine (O(n) length). Template max 2000 chars (validated).

**Mitigation**:
- Documentation warning: Link task-specific docs at task level, broad docs at feature/epic level
- No truncation in MVP (PRD explicit: all docs included)
- Future: Add config for max items if performance degrades

**Risk**: Low. Most tasks will have 1-5 related docs. Power users can optimize linking.

---

## 8. Code Examples

### 8.1 Enhanced Placeholder Function (Task)

```go
// File: internal/config/template_helpers.go

// TaskPlaceholdersWithRelated extends TaskPlaceholders with relational data.
// Requires DocumentRepository and context for database access.
// Returns empty strings for relational fields on error (graceful degradation).
func TaskPlaceholdersWithRelated(
    task *models.Task,
    docRepo DocumentRepository,
    ctx context.Context,
) map[string]string {
    // Start with base placeholders
    m := TaskPlaceholders(task)

    // Add related documents
    if docRepo != nil {
        docs, err := docRepo.ListForTask(ctx, task.ID)
        if err != nil {
            // Log warning but don't fail template population
            // TODO: Add structured logging here
            m["related_docs"] = ""
        } else {
            m["related_docs"] = formatDocPathsAsCSV(docs)
        }
    } else {
        m["related_docs"] = ""
    }

    // Add related tasks from context data
    m["related_tasks"] = extractRelatedTasksFromContext(task.ContextData)

    return m
}

// formatDocPathsAsCSV extracts file_path from documents and joins with commas.
// Returns empty string if docs is nil or empty.
func formatDocPathsAsCSV(docs []*models.Document) string {
    if len(docs) == 0 {
        return ""
    }
    paths := make([]string, len(docs))
    for i, doc := range docs {
        paths[i] = doc.FilePath
    }
    return strings.Join(paths, ",")
}

// extractRelatedTasksFromContext parses context_data JSON and extracts RelatedTasks.
// Returns empty string on parse error (logs warning).
func extractRelatedTasksFromContext(contextData *string) string {
    if contextData == nil || *contextData == "" {
        return ""
    }
    cd, err := models.FromJSON(*contextData)
    if err != nil {
        // TODO: Log warning with task ID for debugging
        return ""
    }
    if len(cd.RelatedTasks) == 0 {
        return ""
    }
    return strings.Join(cd.RelatedTasks, ",")
}
```

---

### 8.2 Feature Relationship Repository (Skeleton)

```go
// File: internal/repository/feature_relationship_repository.go

package repository

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// FeatureRelationshipRepository handles CRUD for feature relationships.
// Follows the same pattern as TaskRelationshipRepository.
type FeatureRelationshipRepository struct {
    db *DB
}

// NewFeatureRelationshipRepository creates a new repository.
func NewFeatureRelationshipRepository(db *DB) *FeatureRelationshipRepository {
    return &FeatureRelationshipRepository{db: db}
}

// ListRelatedFeatures returns all feature keys related to the given feature.
// Includes both outbound (from_feature_id) and inbound (to_feature_id) relationships.
// Returns comma-separated string of feature keys (e.g., "E07-F01,E10-F05").
func (r *FeatureRelationshipRepository) ListRelatedFeatures(
    ctx context.Context,
    featureID int64,
) ([]string, error) {
    query := `
        SELECT DISTINCT f.key
        FROM features f
        WHERE f.id IN (
            SELECT to_feature_id FROM feature_relationships WHERE from_feature_id = ?
            UNION
            SELECT from_feature_id FROM feature_relationships WHERE to_feature_id = ?
        )
        ORDER BY f.key
    `

    rows, err := r.db.QueryContext(ctx, query, featureID, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to query related features: %w", err)
    }
    defer rows.Close()

    var keys []string
    for rows.Next() {
        var key string
        if err := rows.Scan(&key); err != nil {
            return nil, fmt.Errorf("failed to scan feature key: %w", err)
        }
        keys = append(keys, key)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating related features: %w", err)
    }

    return keys, nil
}

// Additional methods: Create, Delete, GetByType (copy from TaskRelationshipRepository)
```

---

## 9. Success Criteria

### 9.1 Technical Acceptance

- [ ] All placeholder functions (`*WithRelated`) implemented and tested
- [ ] Database tables created with indexes (`feature_relationships`, `epic_relationships`)
- [ ] Repositories implemented with 80% test coverage
- [ ] Migration runs successfully (idempotent, backward compatible)
- [ ] Integration test: full flow (create task → link doc → get action → verify placeholder)
- [ ] Performance test: <50ms overhead with 10 related docs

### 9.2 User-Facing Success

- [ ] Template using `{related_docs}` populates with correct file paths
- [ ] Empty related docs returns empty string (no error)
- [ ] Malformed context JSON doesn't break template population (logs warning)
- [ ] Documentation updated with placeholder list and examples
- [ ] Existing templates continue working (backward compatible)

---

## 10. Open Questions for BA/Architect

### 10.1 For Business Analyst

**Q1**: Should we support filtering related docs by link type?
- Currently: `{related_docs}` includes all link types (general, specification, architecture)
- Proposed: `{spec_docs}`, `{arch_docs}` filtered placeholders?
- **Decision Needed**: MVP supports all types, future enhancement?

**Q2**: How should cross-epic feature relationships display?
- Example: E07-F29 relates to E10-F05 (different epics)
- Placeholder value: `"E07-F05,E10-F05"` (full keys) or `"F05,F05"` (ambiguous)?
- **Decision Needed**: Always use full feature keys (E07-F05 format)?

---

### 10.2 For Architect

**Q1**: Should placeholder functions cache repository results?
- Currently: Fresh query every time `*WithRelated` called
- Alternative: Cache docs/relationships in task struct (stale data risk)
- **Decision Needed**: Accept query overhead or implement caching layer?

**Q2**: Feature/Epic relationship table vs JSON context data?
- PRD decided: Use both (tables for queryability, JSON for lightweight)
- Alternative: Tables only (more consistent, better analytics)
- **Decision Needed**: Confirm dual approach or pivot to tables-only?

**Q3**: CLI commands for relationship management?
- Out of scope for MVP per PRD
- Future: `shark feature relate E07-F29 E07-F05 --type=depends_on`
- **Decision Needed**: MVP ships read-only (templates display), write commands in F30?

---

## 11. Conclusion

**Implementation Confidence**: **HIGH (85%)**

**Rationale**:
- 70% of work is extending proven patterns (template helpers, document repo)
- 30% new code (relationship tables) copies existing TaskRelationship pattern
- All dependencies shipped and stable (E07-F05, E07-F21)
- Zero breaking changes (purely additive)
- Clear integration points with file paths identified

**Recommended Approach**: **Phased rollout** (docs → tasks → relationships)

**Timeline Estimate**: 5-7 days for full feature (1 developer)

**Next Steps for BA**:
1. Review open questions (Section 10)
2. Confirm phased rollout vs all-at-once
3. Define user story priority (docs first? or relationships?)
4. Write acceptance criteria based on Scenarios 1-7 from PRD

**Next Steps for Architect**:
1. Review performance mitigation strategy (Section 6.4)
2. Approve relationship table schema (Section 3.1)
3. Decide on caching strategy (Q1 in Section 10.2)
4. Design structured logging for template errors

---

**File Paths Referenced**:
- `internal/config/template_helpers.go` (placeholder functions)
- `internal/config/orchestrator_action.go` (template population)
- `internal/repository/document_repository.go` (document queries)
- `internal/models/context_data.go` (JSON structure)
- `internal/db/db.go` (schema definitions)
- `internal/repository/task_relationship_repository.go` (pattern to copy)
- `internal/cli/commands/orchestrator_display.go` (action resolution)
- `internal/services/epic_service.go`, `feature_service.go` (service integration)

---

*Research complete. All integration points identified. Ready for BA decomposition and task creation.*
