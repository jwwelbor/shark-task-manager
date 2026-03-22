# E21-F07 Feature Research Report: BaseEntity Struct Embedding

**Date**: 2026-03-20
**Feature**: E21-F07 -- BaseEntity Struct Embedding
**Status**: Complete
**Recommendation**: GO -- with status field approach decision required before implementation

---

## 1. Codebase Patterns Relevant to This Feature

### 1.1 Current Entity Struct Pattern

All 5 entity types follow an identical pattern: declare shared fields inline, then implement 12-14 Entity interface methods per type. Verified across all model files:

| File | Shared Fields | Entity Methods | Entity-Specific Fields |
|------|--------------|----------------|----------------------|
| `internal/models/epic.go` | 10 (ID, Key, Title, Slug, Description, Status, FilePath, ContextData, CreatedAt, UpdatedAt) | 14 methods (lines 45-76) | Priority, BusinessValue, Metadata |
| `internal/models/feature.go` | 10 | 14 methods (lines 38-69) | EpicID, StatusOverride, ProgressPct, ExecutionOrder, Metadata |
| `internal/models/task.go` | 10 | 14 methods (lines 54-85) | FeatureID, AgentType, Priority(int), DependsOn, AssignedAgent, BlockedReason, ExecutionOrder, StartedAt, CompletedAt, BlockedAt, completion metadata fields, Metadata, RejectionCount, LastRejectionAt |
| `internal/models/bug.go` | 10 | 14 methods (lines 58-89) | Severity, LinkedEntityType, LinkedEntityKey |
| `internal/models/change_card.go` | 10 | 14 methods (lines 56-87) | Priority(int), RequestedBy, AssignedTo, EpicID, FeatureID, RelatedTaskID, Justification, ImpactAnalysis, RollbackPlan |

**Verified duplication**: 50 shared field declarations + 70 shared method implementations across 5 files.

### 1.2 Entity Interface (E21-F01 -- Completed)

The Entity interface was established in E21-F01 at `internal/models/entity.go`. It defines 13 methods:
- GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate

Compile-time satisfaction checks already exist (lines 29-35). All 5 types implement the interface manually today.

### 1.3 Pointer vs Value Type Consistency

The E21 research report (Section 2) identified a type inconsistency in ChangeCard. However, **examining the current code**, ChangeCard now uses `*string` for both Slug and FilePath (lines 47-48 of `internal/models/change_card.go`). This was likely normalized during E21-F01. No type inconsistency remains.

All 5 entities use identical types for shared fields:
- `ID` -- `int64`
- `Key` -- `string`
- `Title` -- `string`
- `Slug` -- `*string`
- `Description` -- `*string`
- `Status` -- typed string alias (EpicStatus, FeatureStatus, TaskStatus, BugStatus, ChangeCardStatus)
- `FilePath` -- `*string`
- `ContextData` -- `*string`
- `CreatedAt` -- `time.Time`
- `UpdatedAt` -- `time.Time`

### 1.4 Typed Status Pattern

Each entity uses a distinct string typedef for status:
- `EpicStatus string` (epic.go:8)
- `FeatureStatus string` (feature.go:8)
- `TaskStatus string` (task.go:9)
- `BugStatus string` (bug.go:13)
- `ChangeCardStatus string` (change_card.go:29)

These typed statuses appear in **2,178 occurrences across 156 files** (including tests). This is the highest-risk area of this feature.

### 1.5 JSON Serialization Tags

All shared fields use identical JSON tags across all 5 types:
- `json:"id"`, `json:"key"`, `json:"title"`, `json:"slug,omitempty"`, `json:"description,omitempty"`, `json:"status"`, `json:"file_path,omitempty"`, `json:"context_data,omitempty"`, `json:"created_at"`, `json:"updated_at"`

Go's `encoding/json` flattens embedded struct fields automatically when the embedded struct is not a pointer and has no conflicting tag name. Using `BaseEntity` (not `*BaseEntity`) as an anonymous field with these tags will produce identical JSON output.

### 1.6 Database Scanning Patterns

Repositories scan entity fields directly via positional `Scan()` calls. Example from `internal/repository/epic_repository.go` (lines 72-85):

```go
err := r.db.QueryRowContext(ctx, query, id).Scan(
    &epic.ID, &epic.Key, &epic.Title, &epic.Description,
    &epic.Status, &epic.Priority, &epic.BusinessValue,
    &epic.Slug, &epic.FilePath, &epic.ContextData,
    &epic.CreatedAt, &epic.UpdatedAt,
)
```

Bug repository uses a centralized `scanBug()` helper function (`internal/repository/bug_repository.go:29-52`). Other repositories inline their scan calls.

**Key finding**: Go promotes embedded struct fields, so `&epic.ID` continues to work after embedding. The `Scan()` calls do NOT need modification for BaseEntity embedding -- field promotion handles it transparently.

---

## 2. Existing Implementations with File Paths

### 2.1 Model Files (Direct Modification Targets)

| File | Lines | Action |
|------|-------|--------|
| `/home/jwwel/projects/shark-task-manager/internal/models/entity.go` | 36 | Add BaseEntity struct definition + shared methods |
| `/home/jwwel/projects/shark-task-manager/internal/models/epic.go` | 98 | Replace shared fields with `BaseEntity` embed, remove shared methods |
| `/home/jwwel/projects/shark-task-manager/internal/models/feature.go` | 91 | Replace shared fields with `BaseEntity` embed, remove shared methods |
| `/home/jwwel/projects/shark-task-manager/internal/models/task.go` | 113 | Replace shared fields with `BaseEntity` embed, remove shared methods |
| `/home/jwwel/projects/shark-task-manager/internal/models/bug.go` | 118 | Replace shared fields with `BaseEntity` embed, remove shared methods |
| `/home/jwwel/projects/shark-task-manager/internal/models/change_card.go` | 105 | Replace shared fields with `BaseEntity` embed, remove shared methods |

### 2.2 Test Files (Verification, Not Modification)

| File | Description |
|------|-------------|
| `/home/jwwel/projects/shark-task-manager/internal/models/entity_test.go` | 431 lines -- tests all Entity interface methods across all 5 types. These tests should pass WITHOUT modification after embedding, validating backward compatibility. |
| `/home/jwwel/projects/shark-task-manager/internal/models/bug_test.go` | Bug-specific validation tests |
| `/home/jwwel/projects/shark-task-manager/internal/models/change_card_test.go` | ChangeCard-specific validation tests |

### 2.3 Validation File

| File | Description |
|------|-------------|
| `/home/jwwel/projects/shark-task-manager/internal/models/validation.go` | Shared validation functions (ValidateEpicKey, ValidateFeatureKey, etc.). Not directly affected by embedding but called by entity-specific Validate() methods which remain on each entity. |

### 2.4 Repository Files (Indirect -- Scan Compatibility)

| File | Scan Calls | Impact |
|------|-----------|--------|
| `/home/jwwel/projects/shark-task-manager/internal/repository/epic_repository.go` | ~11 | No change needed (field promotion) |
| `/home/jwwel/projects/shark-task-manager/internal/repository/feature_repository.go` | ~14 | No change needed (field promotion) |
| `/home/jwwel/projects/shark-task-manager/internal/repository/task_repository.go` | ~14 | No change needed (field promotion) |
| `/home/jwwel/projects/shark-task-manager/internal/repository/bug_repository.go` | ~4 (via scanBug helper) | No change needed (field promotion) |
| `/home/jwwel/projects/shark-task-manager/internal/repository/change_card_repository.go` | ~3 | No change needed (field promotion) |

---

## 3. Integration Points

### 3.1 Service Layer -- Entity Interface Usage

The Entity interface is already used polymorphically by cross-cutting services established in E21-F02/F03:

| Service | File | Usage Pattern |
|---------|------|---------------|
| EntityService | `internal/services/entity_service.go` | `ResolveActionFn func(entity models.Entity, status string)` -- consumes Entity interface |
| NoteService | `internal/services/note_service.go` | Uses `models.Entity` via repository adapters |
| BugRepositoryAdapter | `internal/services/bug_repo_adapter.go` | Returns `models.Entity` from `GetByKey()`/`GetByID()` |
| FeatureRepositoryAdapter | `internal/services/feature_repo_adapter.go` | Returns `models.Entity` from `GetByKey()`/`GetByID()` |

These services call Entity interface methods, not direct fields. They are **unaffected** by BaseEntity embedding since the interface contract does not change.

### 3.2 Service Layer -- Direct Field Access

Services access entity fields directly in typed contexts (after type assertion from Entity interface or when working with concrete types):

| Pattern | Occurrence Count | Example |
|---------|-----------------|---------|
| `.Status` on typed entities | 120 in services/ | `task.Status`, `epic.Status`, etc. |
| `.Key` on typed entities | 59 in services/ | `task.Key`, `epic.Key` |
| `.Title` on typed entities | 13 in services/ | `task.Title` |

All of these continue to work via Go field promotion. **No changes needed in services.**

### 3.3 CLI Layer -- Direct Field Access

| Pattern | Occurrence Count |
|---------|-----------------|
| `.Status` on typed entities | 98 in cli/commands/ |
| `.Key` on typed entities | 48 in cli/commands/ |

All continue to work via Go field promotion. **No changes needed in CLI.**

### 3.4 JSON Serialization

JSON serialization occurs at approximately 15 points across the codebase (`json.Marshal`, `json.NewEncoder`). The `cli.OutputJSON()` function in `internal/cli/root.go:386` and `internal/formatters/json.go:53` handle the primary JSON output paths.

Go's `encoding/json` embeds anonymous struct fields inline (no wrapping object). The JSON tags on BaseEntity fields will produce identical output to current per-entity tags. **No changes needed.**

### 3.5 Database Layer

No direct integration with BaseEntity. Repositories continue to use concrete types (`*models.Epic`, etc.) and scan fields positionally. Field promotion ensures `&epic.Key` resolves to `&epic.BaseEntity.Key` transparently.

---

## 4. Extension vs New Code Analysis

### 4.1 Code to Add (New)

| Item | Location | Lines (est.) |
|------|----------|-------------|
| BaseEntity struct definition | `internal/models/entity.go` | ~15 |
| BaseEntity shared methods (12 methods) | `internal/models/entity.go` | ~30 |
| **Total new code** | | **~45 lines** |

### 4.2 Code to Remove

| Item | Per Entity | x5 Entities | Notes |
|------|-----------|-------------|-------|
| Shared field declarations | ~10 lines | ~50 lines | Replaced by `BaseEntity` embed (1 line) |
| Shared getter/setter methods | ~12-14 methods (~25 lines) | ~125 lines | Promoted from BaseEntity |
| **Total removed code** | | **~175 lines** | |

### 4.3 Code to Modify

| Item | Modification | Impact |
|------|-------------|--------|
| Each entity struct | Replace 10 shared fields with `BaseEntity` embed line | 5 changes |
| Each entity's GetEntityType() | Keep as-is (entity-specific, cannot be on BaseEntity) | 0 changes |
| Each entity's Validate() | Keep as-is (entity-specific validation) | 0 changes |
| Each entity's SetStatus() | Depends on status approach chosen (see Section 5.1) | 0-5 changes |

### 4.4 Net Impact

- **Net code reduction**: ~130 lines (175 removed - 45 added)
- **Method count reduction**: 60 methods eliminated (12 shared methods x 5 types, minus 12 on BaseEntity)
- **Compile-time safety**: Maintained via existing `var _ Entity = (*Epic)(nil)` checks

---

## 5. Critical Design Decisions

### 5.1 Status Field Strategy (DECISION REQUIRED)

This is the single most impactful design decision for this feature. The feature description identifies two options:

**Option A: BaseEntity.Status is `string`**

```go
type BaseEntity struct {
    // ...
    Status string `json:"status" db:"status"`
    // ...
}

func (b *BaseEntity) GetStatus() string  { return b.Status }
func (b *BaseEntity) SetStatus(s string) { b.Status = s }

// Each entity adds a typed accessor
func (e *Epic) TypedStatus() EpicStatus { return EpicStatus(e.Status) }
```

- Pros: Maximum deduplication; BaseEntity fully implements Entity interface for status
- Cons: **Breaking change** -- all code using `epic.Status` as `EpicStatus` (2,178 occurrences across 156 files) would need migration. `if task.Status == models.TaskStatus("todo")` becomes `if task.Status == "todo"` or `if task.TypedStatus() == models.TaskStatus("todo")`.
- Risk: Very high touch count, high risk of regression

**Option B: Status stays per-entity, BaseEntity excludes Status**

```go
type BaseEntity struct {
    ID          int64     `json:"id"`
    Key         string    `json:"key"`
    Title       string    `json:"title"`
    // NO Status field
    Slug        *string   `json:"slug,omitempty"`
    // ... other fields
}

// Each entity keeps its typed Status and implements GetStatus/SetStatus
type Epic struct {
    BaseEntity
    Status   EpicStatus `json:"status" db:"status"`
    Priority Priority   `json:"priority" db:"priority"`
    // ...
}
func (e *Epic) GetStatus() string  { return string(e.Status) }
func (e *Epic) SetStatus(s string) { e.Status = EpicStatus(s) }
```

- Pros: **Zero breaking changes**; all existing `epic.Status` usage works as-is; JSON output identical
- Cons: Reduces duplication less (9 fields shared instead of 10; 10 methods shared instead of 12)
- Risk: Very low. GetStatus/SetStatus remain on each entity (10 methods not eliminated), but the other 50 field declarations and 60 method implementations are still eliminated.

**Recommendation: Option B**

The typed status field appears 2,178 times across 156 files. Migrating this would be a separate, massive effort that would dwarf the BaseEntity embedding work itself. Option B still eliminates 9 shared field declarations x 5 = 45 fields and ~50 shared method implementations, achieving the feature's primary goal with zero risk.

The feature description's REQ-F-003 explicitly allows this: "OR: Status remains typed per-entity and BaseEntity.GetStatus() returns `string(e.Status)` -- evaluate which approach causes fewer changes."

### 5.2 JSON Tag Conflict Risk

When embedding, if BaseEntity has `json:"status"` and an outer struct also has a Status field with `json:"status"`, Go's encoding/json resolves this by preferring the outer struct's field. This means Option B (where Status stays on each entity) produces correct JSON without any special handling.

For Option A, no conflict exists because BaseEntity owns the Status field.

---

## 6. Inter-Feature Technical Dependency Map

```
E21-F01 (Entity Interface Foundation) [COMPLETED]
  |
  |-- Provides: Entity interface, compile-time checks, EntityType enum
  |-- BaseEntity must satisfy Entity interface (minus GetEntityType/Validate)
  |
  +---> E21-F07 (BaseEntity Struct Embedding) [THIS FEATURE]
          |
          |-- Provides: BaseEntity struct, field deduplication
          |-- Consumed by: All 5 entity model files
          |
          +---> E21-F08 (Polymorphic Data Model Unification) [draft]
          |       |-- May build on BaseEntity for shared table schema
          |
          +---> E21-F09 (Entity Service Delegation Completion) [draft]
                  |-- Unaffected; services use Entity interface, not BaseEntity directly

E21-F02 (Cross-Cutting Service Unification) [active via F06]
  |-- Uses Entity interface polymorphically
  |-- UNAFFECTED by BaseEntity (interface contract unchanged)

E21-F03 (Status Transition Unification) [completed via entity_service.go]
  |-- EntityService operates on Entity interface
  |-- UNAFFECTED by BaseEntity

E21-F06 (Enhancements and Maintenance) [active, 2/6 tasks done]
  |-- No dependency on or conflict with F07

E21-F10 (CLI Command Consolidation) [draft]
  |-- May benefit from BaseEntity for generic entity display
  |-- No blocking dependency

E21-F11 (Polymorphic Entity Relationships) [draft]
E21-F12 (Remove Unused Acceptance Criteria System) [draft]
  |-- No dependency on F07
```

### Parallel Safety

F07 can be developed in parallel with F06, F09, F10, F11, and F12 because:
1. F07 only modifies model struct definitions and their accessor methods
2. No other active feature is modifying the same model struct fields
3. The Entity interface contract does not change
4. Repository scan patterns are unaffected

### Blocking Relationships

- F07 **depends on** F01 (completed) -- Entity interface must exist
- F08 **may depend on** F07 -- if data model unification builds on BaseEntity
- No features **block on** F07 for their own progress

---

## 7. Implementation Approach Recommendations

### Recommended Task Breakdown

1. **Task 1: Define BaseEntity struct and methods** (~45 lines new code)
   - Add to `internal/models/entity.go`
   - Include 9 shared fields (all except Status, per Option B)
   - Implement 10 shared methods (GetID, GetKey, GetTitle, GetSlug, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt)
   - Add tests for BaseEntity methods directly

2. **Task 2: Refactor Epic to embed BaseEntity**
   - Replace 9 shared fields with `BaseEntity` embed
   - Remove 10 shared method implementations
   - Keep GetEntityType(), GetStatus(), SetStatus(), Validate() on Epic
   - Keep entity-specific fields (Priority, BusinessValue, Metadata)
   - Verify: `make fmt && make lint && make test`

3. **Task 3: Refactor Feature to embed BaseEntity**
   - Same pattern as Task 2
   - Keep entity-specific fields (EpicID, StatusOverride, ProgressPct, ExecutionOrder, Metadata)

4. **Task 4: Refactor Task to embed BaseEntity**
   - Same pattern as Task 2
   - Highest risk due to most entity-specific fields (~20 fields beyond shared ones)
   - Keep all completion metadata, dependency, and timing fields

5. **Task 5: Refactor Bug to embed BaseEntity**
   - Same pattern as Task 2
   - Keep entity-specific fields (Severity, LinkedEntityType, LinkedEntityKey)

6. **Task 6: Refactor ChangeCard to embed BaseEntity**
   - Same pattern as Task 2
   - Keep entity-specific fields (Priority, RequestedBy, AssignedTo, EpicID, FeatureID, RelatedTaskID, Justification, ImpactAnalysis, RollbackPlan)

7. **Task 7: Validation and cleanup**
   - Run full test suite (`make test`)
   - Verify JSON output compatibility (spot-check CLI JSON for each entity type)
   - Verify repository scan compatibility (run repository integration tests)
   - Remove any dead code

### Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| JSON serialization change | Go flattens anonymous struct fields; tags are identical. Verify with existing entity_test.go tests. |
| Database scan breakage | Field promotion ensures `&epic.Key` still works. Repository tests cover this. |
| Status type conflicts | Use Option B (keep typed Status per entity). Zero changes to 2,178 status field references. |
| Embedded field name collision | Audit: no entity has a field named `BaseEntity`. No collision risk. |
| Method promotion shadowing | GetEntityType() and Validate() are NOT on BaseEntity, so no shadowing occurs. GetStatus()/SetStatus() remain per-entity (Option B). |

### Sequencing Recommendation

Refactor one entity at a time, running `make fmt && make lint && make test` after each. Start with Epic (simplest, fewest entity-specific fields) and end with Task (most complex, most entity-specific fields). This allows early detection of any systematic issues.

---

## References

All file paths and line counts verified against the codebase on branch E21 at commit c3a0978.

- Entity interface: `/home/jwwel/projects/shark-task-manager/internal/models/entity.go`
- Entity tests: `/home/jwwel/projects/shark-task-manager/internal/models/entity_test.go`
- Epic model: `/home/jwwel/projects/shark-task-manager/internal/models/epic.go`
- Feature model: `/home/jwwel/projects/shark-task-manager/internal/models/feature.go`
- Task model: `/home/jwwel/projects/shark-task-manager/internal/models/task.go`
- Bug model: `/home/jwwel/projects/shark-task-manager/internal/models/bug.go`
- ChangeCard model: `/home/jwwel/projects/shark-task-manager/internal/models/change_card.go`
- Validation: `/home/jwwel/projects/shark-task-manager/internal/models/validation.go`
- EntityService: `/home/jwwel/projects/shark-task-manager/internal/services/entity_service.go`
- Bug scan helper: `/home/jwwel/projects/shark-task-manager/internal/repository/bug_repository.go`
- Epic repository: `/home/jwwel/projects/shark-task-manager/internal/repository/epic_repository.go`
- Parent epic research: `/home/jwwel/projects/shark-task-manager/docs/plan/E21-entity-polymorphism-and-duplication-reduction/research-report.md`

*Last Updated*: 2026-03-20
