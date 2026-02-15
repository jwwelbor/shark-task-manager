# Feature Research Report: E07-F29 - Template Variables for Related Docs and Tasks

**Feature**: Template Variables for Related Docs and Tasks
**Epic**: E07 - Enhancements
**Research Date**: 2026-02-13
**Researcher**: Research Agent

---

## Executive Summary

This feature will extend the existing template variable system to include `related_docs`, `related_tasks`, and similar relational fields for `instruction_template` substitution. Analysis reveals a well-established template infrastructure (`config.TaskPlaceholders`, `PopulateTemplate`) ready for extension, and an existing document relationship system with junction tables. **Recommendation: Extend existing patterns** rather than creating new code. Primary integration point: `internal/config/template_helpers.go`. Estimated complexity: **S (Small)** - straightforward extension of proven patterns.

---

## Research Questions

1. **How is the current template variable system implemented?**
2. **What related_docs/related_tasks data structures exist in the database?**
3. **Where are template variables populated and consumed?**
4. **What are the integration points for extending template placeholders?**
5. **What format should comma-separated document/task lists take?**

---

## Methodology

- Searched codebase for template-related code (`template*.go`, `instruction_template`, `PopulateTemplate`)
- Examined database schema for document/task relationship tables
- Analyzed existing placeholder factories (`TaskPlaceholders`, `FeaturePlaceholders`, `EpicPlaceholders`)
- Traced template population flow from repository to orchestrator action responses
- Reviewed related commands (`related-docs`, task context features)

---

## Findings

### Finding 1: Established Template Variable System

**Summary**: Shark has a mature template variable system for orchestrator actions using placeholder maps and string replacement.

**Evidence**:
- **File**: `internal/config/template_helpers.go` (lines 13-59, 64-92, 97-126)
  - `TaskPlaceholders(task *models.Task) map[string]string` - Factory for task template variables
  - `FeaturePlaceholders(feature *models.Feature) map[string]string` - Factory for feature variables
  - `EpicPlaceholders(epic *models.Epic) map[string]string` - Factory for epic variables
  - All return `map[string]string` with entity fields (id, title, status, file_path, etc.)

- **File**: `internal/config/orchestrator_action.go` (lines 134-149)
  - `PopulateTemplate(vars map[string]string) string` - String replacer using `strings.NewReplacer`
  - Replaces `{key}` patterns with values from vars map
  - Unknown placeholders left unchanged

- **File**: `internal/repository/task_repository.go` (line 1049)
  - Usage: `instruction := metadata.OrchestratorAction.PopulateTemplate(config.TaskPlaceholders(task))`
  - Demonstrates integration: fetch entity → build placeholder map → populate template

**Implications**:
The existing pattern is simple and extensible. Adding new placeholders requires:
1. Extending placeholder factory functions (e.g., `TaskPlaceholders`)
2. Fetching relational data (related docs/tasks) when building placeholder map
3. Formatting as comma-separated file paths
4. No changes to `PopulateTemplate` itself (it's generic)

---

### Finding 2: Document Relationship System (E07-F05)

**Summary**: Shark has a complete document relationship system with many-to-many junction tables and a `DocumentRepository` for CRUD operations.

**Evidence**:
- **Database Schema** (`internal/db/db.go`, `internal/db/migrate.go`):
  ```sql
  CREATE TABLE documents (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      title TEXT NOT NULL,
      file_path TEXT NOT NULL,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE epic_documents (
      epic_id INTEGER NOT NULL,
      document_id INTEGER NOT NULL,
      PRIMARY KEY (epic_id, document_id),
      FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE,
      FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
  );

  CREATE TABLE feature_documents (
      feature_id INTEGER NOT NULL,
      document_id INTEGER NOT NULL,
      PRIMARY KEY (feature_id, document_id),
      FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE,
      FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
  );

  CREATE TABLE task_documents (
      task_id INTEGER NOT NULL,
      document_id INTEGER NOT NULL,
      PRIMARY KEY (task_id, document_id),
      FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
      FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
  );
  ```

- **Repository** (`internal/repository/document_repository.go`):
  - `CreateOrGet(ctx, title, filePath)` - Create or retrieve document
  - `ListForEpic(ctx, epicID)` - Get all docs for epic
  - `ListForFeature(ctx, featureID)` - Get all docs for feature
  - `ListForTask(ctx, taskID)` - Get all docs for task
  - `LinkToEpic`, `LinkToFeature`, `LinkToTask` - Junction table operations

- **CLI Commands** (`internal/cli/commands/related_docs.go`):
  - `shark related-docs add <title> <path> --epic=E01`
  - `shark related-docs list --task=E01-F01-001`
  - Demonstrates full integration with entity relationship model

**Implications**:
Related documents are already queryable via `DocumentRepository`. For template variables:
1. Fetch related documents when building placeholder map
2. Extract `file_path` from each `*models.Document`
3. Format as comma-separated list: `"docs/spec.md,docs/design.md,docs/notes.md"`
4. Add to placeholder map as `related_docs` key

---

### Finding 3: Related Tasks in Context System (E10-F05)

**Summary**: The context system (`models.ContextData`) already includes a `RelatedTasks []string` field for cross-task references.

**Evidence**:
- **File**: `internal/models/context_data.go` (lines 9-17)
  ```go
  type ContextData struct {
      Progress                 *ProgressContext             `json:"progress,omitempty"`
      ImplementationDecisions  map[string]string            `json:"implementation_decisions,omitempty"`
      OpenQuestions            []string                     `json:"open_questions,omitempty"`
      Blockers                 []BlockerContext             `json:"blockers,omitempty"`
      AcceptanceCriteriaStatus []AcceptanceCriterionContext `json:"acceptance_criteria_status,omitempty"`
      RelatedTasks             []string                     `json:"related_tasks,omitempty"`
  }
  ```

- **File**: `internal/services/context_service.go`
  - Context service manages structured resume context for tasks
  - `RelatedTasks` field stores task keys for cross-references

- **Usage**: Work session resume and task context commands (`task resume`, `task context`)

**Implications**:
Related tasks exist in the `ContextData` JSON blob, NOT as database foreign keys. To expose in templates:
1. Parse task's `context_data` JSON field (if populated)
2. Extract `ContextData.RelatedTasks` array
3. Format as comma-separated list: `"E07-F01-001,E07-F02-003"`
4. Add to placeholder map as `related_tasks` key

**Alternative**: Create a `task_relationships` junction table for formal task-to-task links (not currently implemented). Current context approach is lightweight but less queryable.

---

### Finding 4: Template Population Flow

**Summary**: Template variables are populated at the **repository layer** when fetching entities, ensuring all data is available for orchestrator action responses.

**Evidence**:
- **Task Repository** (`internal/repository/task_repository.go`, line 1049):
  ```go
  func (r *TaskRepository) GetOrchestratorActionForTask(...) {
      task, err := r.GetByKey(ctx, taskKey)
      // ... lookup status metadata ...
      instruction := metadata.OrchestratorAction.PopulateTemplate(
          config.TaskPlaceholders(task)
      )
      return &config.PopulatedAction{...}
  }
  ```

- **Epic Service** (`internal/services/epic_service.go`):
  - Uses `config.EpicPlaceholders(epic)` when building orchestrator action responses
  - Populated at time of status transition

- **Feature Service** (`internal/services/feature_service.go`):
  - Uses `config.FeaturePlaceholders(feature)` for feature-level workflow actions

**Integration Points**:
1. **Primary**: `internal/config/template_helpers.go`
   - Extend `TaskPlaceholders`, `FeaturePlaceholders`, `EpicPlaceholders`
   - Add logic to fetch related docs/tasks and format as CSV strings

2. **Secondary**: Repository layer (if performance optimization needed)
   - Pre-fetch related documents when loading entity
   - Cache in placeholder map for reuse

**Implications**:
- Clean separation: placeholder logic in `template_helpers.go`
- Minimal changes to existing repository/service code
- Template population happens once per entity fetch, minimal performance impact

---

### Finding 5: Existing Relational Field Precedent (Ideas)

**Summary**: The `Idea` entity already uses JSON-serialized arrays for `related_docs` and `dependencies`, demonstrating a working pattern.

**Evidence**:
- **File**: `internal/models/idea.go` (lines 29-30)
  ```go
  type Idea struct {
      // ... other fields ...
      RelatedDocs  *string    `json:"related_docs,omitempty" db:"related_docs"` // JSON array of document paths
      Dependencies *string    `json:"dependencies,omitempty" db:"dependencies"` // JSON array of idea keys
      // ...
  }
  ```

- **Validation** (`internal/models/idea.go`, lines 66-77):
  ```go
  if i.Dependencies != nil {
      if err := ValidateJSONArray(*i.Dependencies); err != nil {
          return fmt.Errorf("invalid dependencies JSON: %w", err)
      }
  }
  if i.RelatedDocs != nil {
      if err := ValidateJSONArray(*i.RelatedDocs); err != nil {
          return fmt.Errorf("invalid related_docs JSON: %w", err)
      }
  }
  ```

**Implications**:
Two approaches exist in codebase:
1. **Junction tables** (tasks/features/epics): More relational, queryable, supports many-to-many
2. **JSON arrays** (ideas): Simpler, fewer tables, harder to query

For template variables, the **data source** differs:
- Epic/Feature/Task: Fetch from junction tables → convert to CSV
- Idea: Already have JSON string → parse → convert to CSV

Both can produce the same template variable format: `"path1,path2,path3"`.

---

## Codebase Patterns Relevant to This Feature

### Pattern 1: Placeholder Factory Pattern

**Location**: `internal/config/template_helpers.go`

**Current Implementation**:
```go
func TaskPlaceholders(task *models.Task) map[string]string {
    m := map[string]string{
        "id":         task.Key,
        "task_id":    task.Key,
        "title":      task.Title,
        "status":     string(task.Status),
        "file_path":  *task.FilePath, // if not nil
        // ... ~15 more fields ...
    }
    return m
}
```

**Extension for Related Docs/Tasks**:
```go
func TaskPlaceholders(task *models.Task) map[string]string {
    m := map[string]string{
        // ... existing fields ...
    }

    // NEW: Add related_docs placeholder
    // NOTE: Requires passing DocumentRepository or pre-fetched docs
    // if relatedDocs := getRelatedDocs(task.ID); len(relatedDocs) > 0 {
    //     paths := make([]string, len(relatedDocs))
    //     for i, doc := range relatedDocs {
    //         paths[i] = doc.FilePath
    //     }
    //     m["related_docs"] = strings.Join(paths, ",")
    // }

    // NEW: Add related_tasks placeholder
    // if task.ContextData != nil {
    //     if ctxData, err := models.FromJSON(*task.ContextData); err == nil {
    //         if len(ctxData.RelatedTasks) > 0 {
    //             m["related_tasks"] = strings.Join(ctxData.RelatedTasks, ",")
    //         }
    //     }
    // }

    return m
}
```

**Challenge**: Placeholder factories currently receive only the entity object, not repository access. Options:
1. **Pass repository as parameter**: `TaskPlaceholders(task, docRepo)`
2. **Pre-fetch related data**: Caller fetches docs before calling placeholder factory
3. **Add method to entity**: `task.GetRelatedDocPaths()` (breaks separation of concerns)

**Recommendation**: Option 2 (pre-fetch) - maintains function simplicity while enabling extension.

---

### Pattern 2: Repository Query Pattern

**Location**: `internal/repository/document_repository.go`

**Current Methods**:
```go
func (r *DocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error)
func (r *DocumentRepository) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error)
func (r *DocumentRepository) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error)
```

**Usage Pattern** (from `related_docs.go`):
```go
docRepo := repository.NewDocumentRepository(dbWrapper)
taskRepo := repository.NewTaskRepository(dbWrapper)

task, err := taskRepo.GetByKey(ctx, taskKey)
docs, err := docRepo.ListForTask(ctx, task.ID)

// Extract file paths
for _, doc := range docs {
    fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
}
```

**Integration for Templates**:
```go
// In template_helpers.go or new helper
func TaskPlaceholdersWithDocs(task *models.Task, docRepo *repository.DocumentRepository, ctx context.Context) map[string]string {
    m := TaskPlaceholders(task) // Start with base placeholders

    // Fetch related documents
    if docRepo != nil && task.ID > 0 {
        docs, err := docRepo.ListForTask(ctx, task.ID)
        if err == nil && len(docs) > 0 {
            paths := make([]string, len(docs))
            for i, doc := range docs {
                paths[i] = doc.FilePath
            }
            m["related_docs"] = strings.Join(paths, ",")
        }
    }

    return m
}
```

---

### Pattern 3: Template Variable Format

**Location**: `internal/config/orchestrator_action.go`

**Current Format**: `{placeholder_name}`

**Example Templates**:
```json
{
  "instruction_template": "Work on task {id}: {title}. Status: {status}. File: {file_path}"
}
```

**After Population**:
```
"Work on task E07-F01-001: Implement Auth. Status: in_development. File: docs/plan/E07/E07-F01/tasks/E07-F01-001.md"
```

**Proposed Extension**:
```json
{
  "instruction_template": "Work on task {id}: {title}.\n\nRelated Docs:\n{related_docs}\n\nRelated Tasks:\n{related_tasks}"
}
```

**After Population**:
```
"Work on task E07-F01-001: Implement Auth.

Related Docs:
docs/spec/auth-spec.md,docs/design/oauth-flow.md

Related Tasks:
E07-F02-001,E07-F02-003"
```

**Format Decision**:
- **Comma-separated list** (as specified in feature description): `"path1,path2,path3"`
- **Pro**: Simple, compact, easy to parse in AI prompts
- **Con**: No newlines/formatting (but can be added in template itself)
- **Alternative**: JSON array string `"[\"path1\",\"path2\"]"` (more structured but harder to read)

**Recommendation**: Use comma-separated format as specified. Template authors can add formatting:
```
Related Docs: {related_docs}
```
becomes:
```
Related Docs: docs/a.md,docs/b.md
```

---

## Existing Implementations with File Paths

### Implementation 1: Document Repository Integration

**File**: `internal/repository/document_repository.go`
**Lines**: 23-54, 82-100

**What it does**:
- `CreateOrGet(title, filePath)` - Stores documents with unique file_path
- `getByTitleAndPath(title, filePath)` - Looks up by composite key
- `ListForTask/Feature/Epic(entityID)` - Fetches all linked documents

**Code Pattern**:
```go
func (r *DocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
    query := `
        SELECT d.id, d.title, d.file_path, d.created_at
        FROM documents d
        INNER JOIN task_documents td ON d.id = td.document_id
        WHERE td.task_id = ?
        ORDER BY d.created_at DESC
    `
    // ... scan rows into []*models.Document ...
}
```

**Relevance**: Direct data source for `related_docs` template variable.

---

### Implementation 2: Related Docs CLI Command

**File**: `internal/cli/commands/related_docs.go`
**Lines**: 358-436

**What it does**:
- `shark related-docs list --task=E07-F01-001`
- Fetches documents via `docRepo.ListForTask()`
- Outputs title and file_path

**Code Pattern**:
```go
docs, err := docRepo.ListForTask(ctx, task.ID)
for _, doc := range docs {
    fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
}
```

**Relevance**: Demonstrates how to extract file paths from `*models.Document` list.

---

### Implementation 3: Context Data Related Tasks

**File**: `internal/models/context_data.go`
**Lines**: 9-17

**What it does**:
- Stores `RelatedTasks []string` in JSON context blob
- Serialized to `task.context_data` column

**Code Pattern**:
```go
type ContextData struct {
    RelatedTasks []string `json:"related_tasks,omitempty"`
    // ...
}

func FromJSON(jsonStr string) (*ContextData, error) {
    var cd ContextData
    json.Unmarshal([]byte(jsonStr), &cd)
    return &cd, nil
}
```

**Relevance**: Data source for `related_tasks` template variable (if task has context_data populated).

---

## Integration Points

### 1. Primary Integration: Template Helpers

**File**: `internal/config/template_helpers.go`

**Current Functions**:
- `TaskPlaceholders(task)` → map[string]string
- `FeaturePlaceholders(feature)` → map[string]string
- `EpicPlaceholders(epic)` → map[string]string

**Extension Strategy**:
```go
// Option A: Add overloaded functions with repository access
func TaskPlaceholdersWithRelated(task *models.Task, docRepo DocumentListerer) map[string]string

// Option B: Add separate helper for relational fields
func AddRelatedDocsPlaceholder(m map[string]string, taskID int64, docRepo DocumentListerer)

// Option C: Extend existing functions with optional context parameter
func TaskPlaceholders(task *models.Task, opts ...PlaceholderOption) map[string]string
```

**Recommendation**: Option A (new function variant) - cleanest API, backwards compatible.

---

### 2. Repository Layer: Task/Feature/Epic Repositories

**Files**:
- `internal/repository/task_repository.go`
- `internal/repository/feature_repository.go`
- `internal/repository/epic_repository.go`

**Current Integration Point** (Task example, line 1049):
```go
func (r *TaskRepository) GetOrchestratorActionForTask(...) {
    task, err := r.GetByKey(ctx, taskKey)
    instruction := metadata.OrchestratorAction.PopulateTemplate(
        config.TaskPlaceholders(task) // ← CHANGE HERE
    )
}
```

**Modification**:
```go
func (r *TaskRepository) GetOrchestratorActionForTask(...) {
    task, err := r.GetByKey(ctx, taskKey)

    // NEW: Create document repository for fetching related docs
    docRepo := NewDocumentRepository(r.db)

    // NEW: Use extended placeholder function
    instruction := metadata.OrchestratorAction.PopulateTemplate(
        config.TaskPlaceholdersWithRelated(task, docRepo, ctx)
    )
}
```

**Impact**: ~3 files (task/feature/epic repositories), ~10 lines of code each.

---

### 3. Service Layer: Epic/Feature Services

**Files**:
- `internal/services/epic_service.go`
- `internal/services/feature_service.go`

**Current Usage**:
```go
placeholders := config.EpicPlaceholders(epic)
instruction := action.PopulateTemplate(placeholders)
```

**Modification**:
```go
// NEW: Inject document repository into service
type EpicService struct {
    repo     EpicRepository
    docRepo  DocumentRepository // ← ADD
    workflow *workflow.Service
}

// In transition method:
placeholders := config.EpicPlaceholdersWithRelated(epic, s.docRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Impact**: ~2 service files, constructor updates, ~5 lines per service.

---

### 4. Database Schema: No Changes Required

**Analysis**: Junction tables (`epic_documents`, `feature_documents`, `task_documents`) already exist. No schema migration needed.

**Verification**:
```sql
-- Existing tables (confirmed in internal/db/db.go):
CREATE TABLE documents (...);
CREATE TABLE epic_documents (...);
CREATE TABLE feature_documents (...);
CREATE TABLE task_documents (...);
```

**Conclusion**: Zero database changes required. Feature is purely code-based.

---

### 5. CLI Commands: Display Integration (Optional)

**Files**:
- `internal/cli/commands/task.go`
- `internal/cli/commands/feature.go`
- `internal/cli/commands/epic.go`

**Current Behavior**: `shark get E07-F01-001 --json` shows orchestrator_action with populated template.

**Enhancement Opportunity**:
```json
{
  "key": "E07-F01-001",
  "title": "Implement Auth",
  "status": "in_development",
  "orchestrator_action": {
    "action": "spawn_agent",
    "instruction": "Work on task E07-F01-001: Implement Auth.\n\nRelated Docs:\ndocs/spec/auth.md,docs/design/flow.md"
  }
}
```

**Impact**: None required for MVP. Template variables work automatically once placeholder factories extended.

---

## Extension vs New Code Analysis

### Extension Opportunities (Preferred)

| Component | Action | Rationale |
|-----------|--------|-----------|
| `template_helpers.go` | Extend `TaskPlaceholders` family | Existing pattern, proven API |
| `document_repository.go` | Use existing `ListForTask/Feature/Epic` | Already implemented, tested |
| Repository orchestrator methods | Call extended placeholder factories | Minimal change to existing flow |
| `PopulateTemplate` | No changes | Generic, handles any placeholder |
| Database schema | No changes | Junction tables already exist |

**Code Impact**:
- **New code**: ~100-150 lines (placeholder extensions, helper functions)
- **Modified code**: ~30-50 lines (repository/service updates)
- **Test code**: ~200-300 lines (unit tests for new placeholders)

**Risk**: Low - extends proven patterns, no breaking changes.

---

### New Code Required

| Component | Purpose | Complexity |
|-----------|---------|------------|
| `PlaceholdersWithRelated` functions | Fetch docs/tasks and format | Low - straightforward queries |
| Helper: `formatDocPathsAsCSV` | Extract paths from `[]*Document` | Trivial - `strings.Join` |
| Helper: `extractRelatedTasksFromContext` | Parse `ContextData.RelatedTasks` | Low - JSON parsing |
| Tests: Template helper tests | Verify placeholder population | Medium - mock repositories |
| Tests: Integration tests | End-to-end template variable flow | Medium - database fixtures |

**Total New Code**: ~300-400 lines (including tests).

---

### Architectural Decision: Where to Fetch Related Data?

**Option 1: In Placeholder Factory (with repo injection)**
```go
func TaskPlaceholdersWithRelated(task *models.Task, docRepo *DocumentRepository, ctx context.Context) map[string]string {
    m := TaskPlaceholders(task)
    docs, _ := docRepo.ListForTask(ctx, task.ID)
    m["related_docs"] = formatDocPaths(docs)
    return m
}
```
**Pros**: Encapsulated, simple API
**Cons**: Placeholder factory does I/O (not pure function)

**Option 2: Pre-fetch in Repository Layer**
```go
func (r *TaskRepository) GetOrchestratorActionForTask(...) {
    task, _ := r.GetByKey(ctx, taskKey)
    docs, _ := r.docRepo.ListForTask(ctx, task.ID)

    m := config.TaskPlaceholders(task)
    m["related_docs"] = formatDocPaths(docs)
    instruction := action.PopulateTemplate(m)
}
```
**Pros**: Keeps placeholder factory pure
**Cons**: Duplicates logic across task/feature/epic repos

**Option 3: Add Related Fields to Entity Models**
```go
type TaskWithRelations struct {
    *models.Task
    RelatedDocs []*models.Document
}
```
**Pros**: Clean separation, supports multiple use cases
**Cons**: More changes, new types to maintain

**Recommendation**: **Option 1** (repo injection) - balances simplicity with extensibility. Pure function goal sacrificed for practicality.

---

## Inter-Feature Technical Dependency Map

### Direct Dependencies

| Feature | Relationship | Impact on E07-F29 |
|---------|--------------|-------------------|
| **E07-F05** (Related Documents) | Provides document junction tables | **CRITICAL** - Data source for `related_docs` |
| **E07-F21** (Orchestrator Actions) | Defined `instruction_template` system | **CRITICAL** - Template system being extended |
| **E10-F05** (Work Sessions/Context) | Stores `RelatedTasks` in `ContextData` | **IMPORTANT** - Data source for `related_tasks` |
| **E07-F28** (Orchestration Action on Get) | Displays populated actions | **MINOR** - Beneficiary of new variables |

### Dependency Graph

```
E07-F29 (Template Variables)
    ├── READS FROM: E07-F05 (document_repository, junction tables)
    ├── EXTENDS: E07-F21 (instruction_template, PopulateTemplate)
    ├── READS FROM: E10-F05 (ContextData.RelatedTasks)
    └── ENHANCES: E07-F28 (orchestrator action responses)
```

### Sibling Feature Interactions

**E07-F16** (Workflow Config Integration): May define which statuses have orchestrator actions → affects when templates are populated.

**E07-F23** (Enhanced Status Tracking): Displays task status changes → could benefit from showing related docs/tasks in status views.

**E07-F26** (Centralized Workflow Service): Manages workflow authority → no direct impact, but good reference for service patterns.

### External System Dependencies

**None** - Feature is entirely internal to Shark codebase.

---

## Implementation Approach Recommendations

### Recommended Approach: Incremental Extension

**Phase 1: Extend Placeholder Factories** (2-3 hours)
1. Create `TaskPlaceholdersWithRelated(task, docRepo, ctx)` in `template_helpers.go`
2. Implement `formatDocPathsAsCSV([]*models.Document) string` helper
3. Implement `extractRelatedTasksFromContext(*string) string` helper
4. Write unit tests with mock repositories

**Phase 2: Integrate into Repository Layer** (1-2 hours)
1. Update `TaskRepository.GetOrchestratorActionForTask()` to use new placeholder function
2. Update `EpicService` orchestrator action population
3. Update `FeatureService` orchestrator action population
4. Add integration tests

**Phase 3: Documentation and Examples** (1 hour)
1. Add template variable reference to `docs/guides/template-system.md`
2. Create example templates in `.sharkconfig.json` examples
3. Update CLI reference docs

**Total Effort**: 4-6 hours (including tests and docs)

---

### Alternative Approach: Pure Extension with Builder Pattern

**Concept**: Use builder pattern to avoid repo injection in placeholder factories.

```go
type PlaceholderBuilder struct {
    entity  interface{}
    docRepo *DocumentRepository
    ctx     context.Context
}

func (b *PlaceholderBuilder) WithRelatedDocs() *PlaceholderBuilder {
    // Fetch and add related_docs placeholder
    return b
}

func (b *PlaceholderBuilder) Build() map[string]string {
    // Return final placeholder map
}
```

**Usage**:
```go
placeholders := NewPlaceholderBuilder(task).
    WithRelatedDocs(docRepo, ctx).
    WithRelatedTasks().
    Build()
```

**Pros**: More flexible, cleaner API
**Cons**: More code, higher complexity for simple use case

**Verdict**: Not recommended for MVP. Current approach is simpler.

---

### Anti-Patterns to Avoid

**❌ Don't**: Store related docs as denormalized CSV in task table
```sql
ALTER TABLE tasks ADD COLUMN related_docs_csv TEXT; -- WRONG
```
**Why**: Breaks normalization, hard to maintain, duplicates data.

**❌ Don't**: Fetch related docs on every placeholder factory call without caching
```go
func TaskPlaceholders(task) {
    // This would be called multiple times per request
    docs, _ := fetchDocsFromDB(task.ID) // INEFFICIENT
}
```
**Why**: N+1 query problem, performance degradation.

**❌ Don't**: Hardcode document paths in templates
```json
{
  "instruction_template": "Read docs/spec.md then work on {id}"
}
```
**Why**: Not dynamic, breaks when docs move, defeats purpose of feature.

**✅ Do**: Use relational data fetched once, formatted consistently
```go
docs := docRepo.ListForTask(ctx, task.ID) // Once per request
placeholders["related_docs"] = formatDocPaths(docs)
```

---

## Technical Challenges and Mitigations

### Challenge 1: Performance - N+1 Queries

**Problem**: Fetching related docs for every task during list operations could cause N queries.

**Example**:
```go
// BAD: N+1 pattern
for _, task := range tasks {
    docs := docRepo.ListForTask(ctx, task.ID) // N queries
}
```

**Mitigation**:
1. **Lazy Loading**: Only fetch related docs when template actually used (not during list)
2. **Batch Loading**: Add `docRepo.ListForTasks([]int64{taskIDs})` for bulk fetch
3. **Caching**: Cache document lists in placeholder builder

**Implementation**:
```go
// Add to DocumentRepository
func (r *DocumentRepository) ListForTasks(ctx context.Context, taskIDs []int64) (map[int64][]*Document, error) {
    query := `
        SELECT td.task_id, d.id, d.title, d.file_path
        FROM documents d
        JOIN task_documents td ON d.id = td.document_id
        WHERE td.task_id IN (?)
    `
    // ... use sqlx.In for IN clause ...
}
```

**Priority**: Medium (optimize if performance testing shows need).

---

### Challenge 2: Missing Related Data

**Problem**: Tasks may have zero related docs/tasks. Template should handle gracefully.

**Example**:
```
Template: "Read {related_docs} before starting {id}"
With no docs: "Read  before starting E07-F01-001" // Empty space
```

**Mitigation**:
1. **Empty string for missing data**: `m["related_docs"] = ""`
2. **Template conditional logic**: Not supported by simple string replacer
3. **Pre-formatted message**: `m["related_docs"] = "(no related documents)"`

**Recommended Solution**:
```go
if len(docs) == 0 {
    m["related_docs"] = "" // Let template author handle with formatting
} else {
    m["related_docs"] = formatDocPaths(docs)
}
```

**Template Author Best Practice**:
```
{{if related_docs}}
Related Docs: {related_docs}
{{end}}
```
(But current system doesn't support conditionals - just string replacement)

**Practical Solution**: Document that empty values mean "no related items" in template guide.

---

### Challenge 3: Context Data Parsing

**Problem**: `task.context_data` is a JSON blob. Parsing could fail or return nil.

**Example**:
```go
contextData, err := models.FromJSON(*task.ContextData)
if err != nil {
    // What to do? Fail silently? Log? Error?
}
```

**Mitigation**:
```go
func extractRelatedTasksFromContext(contextJSON *string) string {
    if contextJSON == nil || strings.TrimSpace(*contextJSON) == "" {
        return "" // No context data
    }

    ctxData, err := models.FromJSON(*contextJSON)
    if err != nil {
        // Log warning but don't fail
        log.Printf("Failed to parse context_data: %v", err)
        return ""
    }

    if len(ctxData.RelatedTasks) == 0 {
        return ""
    }

    return strings.Join(ctxData.RelatedTasks, ",")
}
```

**Key**: Fail gracefully, return empty string, don't break template population.

---

## Testing Strategy

### Unit Tests

**File**: `internal/config/template_helpers_test.go`

**Test Cases**:
1. ✅ `TestTaskPlaceholdersWithRelated_WithDocs` - Task with 2+ related documents
2. ✅ `TestTaskPlaceholdersWithRelated_NoDocs` - Task with zero related documents
3. ✅ `TestTaskPlaceholdersWithRelated_WithRelatedTasks` - Task with context_data containing related tasks
4. ✅ `TestTaskPlaceholdersWithRelated_NoContext` - Task with nil context_data
5. ✅ `TestTaskPlaceholdersWithRelated_InvalidJSON` - Task with malformed context_data JSON
6. ✅ `TestFormatDocPathsAsCSV` - Document path formatting helper
7. ✅ `TestExtractRelatedTasksFromContext` - Context parsing helper

**Mock Strategy**:
```go
type MockDocumentRepository struct {
    ListForTaskFunc func(context.Context, int64) ([]*models.Document, error)
}

func (m *MockDocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
    if m.ListForTaskFunc != nil {
        return m.ListForTaskFunc(ctx, taskID)
    }
    return nil, nil
}
```

---

### Integration Tests

**File**: `internal/repository/task_repository_test.go`

**Test Cases**:
1. ✅ `TestGetOrchestratorActionForTask_WithRelatedDocs` - End-to-end template population
2. ✅ `TestTaskStatusUpdate_OrchestratorActionIncludesRelatedDocs` - Action on transition
3. ✅ `TestFeatureOrchestratorAction_WithRelatedDocs` - Feature-level template population
4. ✅ `TestEpicOrchestratorAction_WithRelatedDocs` - Epic-level template population

**Database Setup**:
```go
func setupTestTaskWithDocs(t *testing.T) (*models.Task, []*models.Document) {
    db := test.GetTestDB()

    // Create task
    task := createTestTask(db)

    // Create documents
    docRepo := repository.NewDocumentRepository(db)
    doc1, _ := docRepo.CreateOrGet(ctx, "Spec", "docs/spec.md")
    doc2, _ := docRepo.CreateOrGet(ctx, "Design", "docs/design.md")

    // Link to task
    docRepo.LinkToTask(ctx, task.ID, doc1.ID)
    docRepo.LinkToTask(ctx, task.ID, doc2.ID)

    return task, []*models.Document{doc1, doc2}
}
```

---

### Manual Testing Scenarios

**Scenario 1: Task with Related Docs**
```bash
# Setup
shark related-docs add "Auth Spec" docs/auth-spec.md --task=E07-F01-001
shark related-docs add "OAuth Flow" docs/oauth-flow.md --task=E07-F01-001

# Trigger orchestrator action
shark task update-status E07-F01-001 ready_for_development --json

# Verify output includes:
# "instruction": "... Related Docs: docs/auth-spec.md,docs/oauth-flow.md ..."
```

**Scenario 2: Task with Related Tasks (via Context)**
```bash
# Setup context with related tasks
shark task context update E07-F01-001 --related-tasks="E07-F02-001,E07-F02-003"

# Trigger orchestrator action
shark task update-status E07-F01-001 ready_for_development --json

# Verify output includes:
# "instruction": "... Related Tasks: E07-F02-001,E07-F02-003 ..."
```

**Scenario 3: Task with No Related Items**
```bash
# Create task without related docs/tasks
shark task create E07 F01 "Standalone Task"

# Trigger orchestrator action
shark task update-status E07-F01-002 ready_for_development --json

# Verify output:
# - `related_docs` placeholder replaced with empty string
# - Template remains valid
```

---

## Code Volume Estimates

### New Code

| File | Lines | Purpose |
|------|-------|---------|
| `template_helpers.go` | 50 | `TaskPlaceholdersWithRelated` and variants |
| `template_helpers.go` | 20 | `formatDocPathsAsCSV` helper |
| `template_helpers.go` | 30 | `extractRelatedTasksFromContext` helper |
| `task_repository.go` | 10 | Updated orchestrator action call |
| `epic_service.go` | 10 | Updated placeholder usage |
| `feature_service.go` | 10 | Updated placeholder usage |
| **Total** | **130** | Production code |

### Test Code

| File | Lines | Purpose |
|------|-------|---------|
| `template_helpers_test.go` | 150 | Unit tests for placeholder factories |
| `task_repository_test.go` | 80 | Integration test for task templates |
| `epic_service_test.go` | 40 | Integration test for epic templates |
| `feature_service_test.go` | 40 | Integration test for feature templates |
| **Total** | **310** | Test code |

### Documentation

| File | Lines | Purpose |
|------|-------|---------|
| `template-system.md` | 50 | Template variable reference |
| `CLI_REFERENCE.md` | 20 | Updated orchestrator action examples |
| `feature.md` | 30 | Feature completion documentation |
| **Total** | **100** | Documentation |

**Grand Total**: ~540 lines (130 prod + 310 test + 100 docs)

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Performance degradation (N+1 queries) | Medium | Medium | Batch fetch, lazy loading, caching |
| Context JSON parsing errors | Low | Low | Fail gracefully, return empty string |
| Template variable name conflicts | Low | Low | Use namespaced names (`related_docs` vs `docs`) |
| Breaking changes to existing templates | Very Low | High | Additive only - no removal of existing placeholders |

### Integration Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Repository dependency injection complexity | Low | Medium | Use simple function parameters, not DI framework |
| Cross-feature coupling (E07-F05, E10-F05) | Low | Low | Well-defined interfaces, existing data structures |
| Testing coverage gaps | Medium | Medium | Comprehensive unit + integration tests (target 80%+) |

### Operational Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Missing related docs data | High | Low | Empty string handling, documentation |
| Large number of related items | Low | Medium | Truncation strategy (e.g., first 10 docs) |

**Overall Risk**: **Low** - Extends proven patterns, minimal breaking change potential.

---

## Success Criteria

### Functional Success

✅ **Template Variable Support**:
- `{related_docs}` populates with comma-separated file paths
- `{related_tasks}` populates with comma-separated task keys
- Empty values handled gracefully (empty string)

✅ **Entity Coverage**:
- Works for tasks (`TaskPlaceholders`)
- Works for features (`FeaturePlaceholders`)
- Works for epics (`EpicPlaceholders`)

✅ **Integration**:
- Populated in orchestrator action responses (`shark task update-status --json`)
- Available in all contexts where `instruction_template` is used
- Backward compatible (existing templates unaffected)

### Technical Success

✅ **Performance**:
- No measurable performance degradation on `task next` (< 50ms overhead)
- Bulk operations (list 100+ tasks) remain performant

✅ **Code Quality**:
- Test coverage > 80% for new code
- All existing tests pass
- Linter and vet pass (no new warnings)

✅ **Documentation**:
- Template variable reference published
- Example templates provided
- CLI reference updated

---

## Next Steps for BA and Architect

### For Business Analyst

**Input Needed**:
1. **User Stories**: How should AI agents use `related_docs` and `related_tasks` in practice?
   - Example: "As an AI agent starting a task, I want to see related documentation so I can understand context."

2. **Edge Case Handling**: What should happen when:
   - A task has 50+ related documents? (Truncate? Paginate? Error?)
   - Related task keys are invalid/deleted?
   - Context JSON is corrupted?

3. **Template Examples**: Draft sample `instruction_template` strings showing:
   - How to use `{related_docs}` in developer workflow
   - How to use `{related_tasks}` for cross-task coordination
   - Formatting best practices

**Recommended Sections**:
- Acceptance Criteria (Given/When/Then)
- User Stories (Must-Have vs Should-Have)
- Error Handling Stories

---

### For Architect

**Technical Decisions Needed**:
1. **Placeholder Factory Signature**: Approve repo injection approach?
   ```go
   func TaskPlaceholdersWithRelated(task *models.Task, docRepo *DocumentRepository, ctx context.Context) map[string]string
   ```

2. **Performance Optimization**: Should we implement batch loading in MVP or defer to later?
   ```go
   func (r *DocumentRepository) ListForTasks(ctx context.Context, taskIDs []int64) (map[int64][]*Document, error)
   ```

3. **Empty Value Handling**: Confirm strategy:
   - Empty string for no related items? ✅
   - Placeholder text like "(none)"? ❌
   - Omit placeholder entirely? ❌

4. **Related Tasks Source**: Use `ContextData.RelatedTasks` or create formal junction table?
   - **Current**: Lightweight, JSON-based (recommended for MVP)
   - **Future**: Queryable `task_relationships` table (if needed)

**Review Points**:
- Extension pattern alignment with existing architecture
- Service layer dependency injection strategy
- Test coverage approach (unit vs integration)

---

## Appendices

### A. Related Document Schema

```sql
-- Core tables
CREATE TABLE documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Junction tables
CREATE TABLE epic_documents (
    epic_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (epic_id, document_id),
    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE feature_documents (
    feature_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (feature_id, document_id),
    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE task_documents (
    task_id INTEGER NOT NULL,
    document_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, document_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);
```

### B. Context Data Structure

```go
type ContextData struct {
    Progress                 *ProgressContext             `json:"progress,omitempty"`
    ImplementationDecisions  map[string]string            `json:"implementation_decisions,omitempty"`
    OpenQuestions            []string                     `json:"open_questions,omitempty"`
    Blockers                 []BlockerContext             `json:"blockers,omitempty"`
    AcceptanceCriteriaStatus []AcceptanceCriterionContext `json:"acceptance_criteria_status,omitempty"`
    RelatedTasks             []string                     `json:"related_tasks,omitempty"` // ← SOURCE
}
```

Stored in `tasks.context_data` as JSON TEXT.

### C. Template Variable Reference

| Placeholder | Entity Level | Data Source | Format |
|-------------|-------------|-------------|--------|
| `{id}` | Task/Feature/Epic | `entity.Key` | String (e.g., "E07-F01-001") |
| `{title}` | Task/Feature/Epic | `entity.Title` | String |
| `{status}` | Task/Feature/Epic | `entity.Status` | String (e.g., "in_development") |
| `{file_path}` | Task/Feature/Epic | `entity.FilePath` | String path |
| `{related_docs}` | Task/Feature/Epic | `document_repository.ListFor*()` | **CSV: "path1,path2,path3"** |
| `{related_tasks}` | Task | `task.context_data.RelatedTasks` | **CSV: "key1,key2,key3"** |

### D. Example Template

```json
{
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["coding", "testing"],
        "instruction_template": "Implement task {id}: {title}\n\nStatus: {status}\nFile: {file_path}\n\n### Related Documentation\n{related_docs}\n\n### Related Tasks\n{related_tasks}\n\nPlease review related materials before starting implementation."
      }
    }
  }
}
```

**Populated Output**:
```
Implement task E07-F01-001: JWT Token Validation

Status: ready_for_development
File: docs/plan/E07/E07-F01/tasks/E07-F01-001.md

### Related Documentation
docs/spec/oauth-2.0-spec.md,docs/design/auth-flow.md,docs/security/jwt-best-practices.md

### Related Tasks
E07-F02-001,E07-F02-003

Please review related materials before starting implementation.
```

---

## References

### Codebase Files Analyzed

**Template System**:
- `/internal/config/template_helpers.go` (lines 13-126)
- `/internal/config/orchestrator_action.go` (lines 134-149)
- `/internal/templates/renderer.go` (lines 11-84)

**Document System**:
- `/internal/repository/document_repository.go` (lines 1-463)
- `/internal/cli/commands/related_docs.go` (lines 1-463)
- `/internal/db/db.go` (schema lines 266-318)

**Context System**:
- `/internal/models/context_data.go` (lines 9-147)
- `/internal/services/context_service.go`

**Orchestrator Integration**:
- `/internal/repository/task_repository.go` (line 1049)
- `/internal/services/epic_service.go`
- `/internal/services/feature_service.go`

### Related Features

- **E07-F05**: Add Related Documents (provides junction tables)
- **E07-F21**: Add Actions to Status Transition (defines instruction_template)
- **E10-F05**: Work Sessions Resume Context (provides RelatedTasks field)
- **E07-F28**: Orchestration Action on Get (displays populated actions)

---

**Research Complete**: All sections populated with codebase evidence. File paths cited. Integration points identified. Ready for BA refinement and architect review.

**Recommended Next Action**: Advance feature to BA refinement for user story development and acceptance criteria definition.
