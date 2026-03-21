# Technical Architecture: E21-F07 BaseEntity Struct Embedding

**Feature**: E21-F07
**Complexity Tier**: COMPLEX
**Author**: Architect
**Date**: 2026-03-20
**Status**: Draft

---

## 1. Overview

### Problem

All 5 entity types (Epic, Feature, Task, Bug, ChangeCard) independently declare 10 shared fields and independently implement 14 Entity interface methods. This results in 50 duplicated field declarations and ~70 duplicated method implementations. Changes to shared fields require 5 synchronized modifications.

### Solution

Introduce a `BaseEntity` embedded struct containing 9 shared fields (all except Status) and 10 shared Entity interface method implementations. Each entity type embeds `BaseEntity` as an anonymous field, retaining entity-specific fields and the typed Status field on the outer struct.

### Design Decision: Option B (Status Excluded from BaseEntity)

**Decision**: BaseEntity does NOT include the Status field. Each entity retains its typed status field (EpicStatus, TaskStatus, etc.) and continues to implement GetStatus()/SetStatus() individually.

**Rationale**:
- Typed status fields appear in 2,178 occurrences across 156 files
- Changing all of these to `string` would dwarf the embedding effort itself
- Option B still eliminates 45 field declarations (9 fields x 5 types) and ~50 method implementations
- REQ-F-003 explicitly allows this approach: "OR: Status remains typed per-entity"
- Zero breaking changes to existing code

**Alternative rejected (Option A)**: BaseEntity.Status as `string`. Would achieve maximum deduplication (10 fields, 12 methods) but introduces a massive codebase-wide migration of 2,178 status type references with high regression risk.

---

## 2. Architecture

### 2.1 Component Diagram

```
internal/models/
  entity.go         <-- ADD BaseEntity struct + 10 shared methods here
  epic.go           <-- MODIFY: embed BaseEntity, remove 9 fields + 10 methods
  feature.go        <-- MODIFY: embed BaseEntity, remove 9 fields + 10 methods
  task.go           <-- MODIFY: embed BaseEntity, remove 9 fields + 10 methods
  bug.go            <-- MODIFY: embed BaseEntity, remove 9 fields + 10 methods
  change_card.go    <-- MODIFY: embed BaseEntity, remove 9 fields + 10 methods
  entity_test.go    <-- VERIFY: existing tests pass without modification
```

### 2.2 BaseEntity Struct Definition

**Location**: `internal/models/entity.go` (append to existing file)

```go
// BaseEntity contains the shared fields common to all domain entities.
// Entity types embed BaseEntity as an anonymous field to inherit these
// fields and their accessor methods.
//
// Status is intentionally excluded because each entity type uses a
// distinct typed status alias (EpicStatus, TaskStatus, etc.) that
// appears in ~2,178 call sites. Keeping Status per-entity avoids a
// massive codebase-wide migration.
type BaseEntity struct {
    ID          int64     `json:"id" db:"id"`
    Key         string    `json:"key" db:"key"`
    Title       string    `json:"title" db:"title"`
    Slug        *string   `json:"slug,omitempty" db:"slug"`
    Description *string   `json:"description,omitempty" db:"description"`
    FilePath    *string   `json:"file_path,omitempty" db:"file_path"`
    ContextData *string   `json:"context_data,omitempty" db:"context_data"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
```

**Fields included (9)**: ID, Key, Title, Slug, Description, FilePath, ContextData, CreatedAt, UpdatedAt

**Field excluded (1)**: Status -- remains typed per-entity

### 2.3 BaseEntity Method Implementations

```go
// Entity interface methods implemented by BaseEntity.
// These are promoted to embedding types via Go struct embedding.

func (b *BaseEntity) GetID() int64            { return b.ID }
func (b *BaseEntity) GetKey() string          { return b.Key }
func (b *BaseEntity) GetTitle() string        { return b.Title }
func (b *BaseEntity) GetCreatedAt() time.Time { return b.CreatedAt }
func (b *BaseEntity) GetUpdatedAt() time.Time { return b.UpdatedAt }

func (b *BaseEntity) GetSlug() string {
    if b.Slug != nil {
        return *b.Slug
    }
    return ""
}

func (b *BaseEntity) GetDescription() string {
    if b.Description != nil {
        return *b.Description
    }
    return ""
}

func (b *BaseEntity) GetFilePath() string {
    if b.FilePath != nil {
        return *b.FilePath
    }
    return ""
}

func (b *BaseEntity) GetContextData() *string     { return b.ContextData }
func (b *BaseEntity) SetContextData(data *string) { b.ContextData = data }
```

**Methods on BaseEntity (10)**: GetID, GetKey, GetTitle, GetSlug, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt

**Methods remaining per-entity (4)**: GetEntityType, GetStatus, SetStatus, Validate

### 2.4 Entity Type After Refactoring

Example: Epic (simplest entity)

```go
// Before (11 fields, 14 methods):
type Epic struct {
    ID            int64       `json:"id" db:"id"`
    Key           string      `json:"key" db:"key"`
    Title         string      `json:"title" db:"title"`
    Description   *string     `json:"description,omitempty" db:"description"`
    Status        EpicStatus  `json:"status" db:"status"`
    Priority      Priority    `json:"priority" db:"priority"`
    BusinessValue *Priority   `json:"business_value,omitempty" db:"business_value"`
    Slug          *string     `json:"slug,omitempty" db:"slug"`
    FilePath      *string     `json:"file_path,omitempty" db:"file_path"`
    ContextData   *string     `json:"context_data,omitempty" db:"context_data"`
    Metadata      map[string]interface{} `json:"metadata,omitempty" db:"-"`
    CreatedAt     time.Time   `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time   `json:"updated_at" db:"updated_at"`
}

// After (BaseEntity + 4 entity-specific fields, 4 methods):
type Epic struct {
    BaseEntity                             // 9 shared fields + 10 methods
    Status        EpicStatus             `json:"status" db:"status"`
    Priority      Priority               `json:"priority" db:"priority"`
    BusinessValue *Priority              `json:"business_value,omitempty" db:"business_value"`
    Metadata      map[string]interface{} `json:"metadata,omitempty" db:"-"`
}

// Only entity-specific methods remain:
func (e *Epic) GetEntityType() EntityType { return EntityTypeEpic }
func (e *Epic) GetStatus() string         { return string(e.Status) }
func (e *Epic) SetStatus(status string)   { e.Status = EpicStatus(status) }
func (e *Epic) Validate() error           { /* unchanged */ }
```

### 2.5 All Entity Structs After Refactoring

| Entity | BaseEntity | Status Field | Entity-Specific Fields |
|--------|-----------|-------------|----------------------|
| Epic | embedded | `Status EpicStatus` | Priority, BusinessValue, Metadata |
| Feature | embedded | `Status FeatureStatus` | EpicID, StatusOverride, ProgressPct, ExecutionOrder, Metadata |
| Task | embedded | `Status TaskStatus` | FeatureID, AgentType, Priority(int), DependsOn, AssignedAgent, BlockedReason, ExecutionOrder, StartedAt, CompletedAt, BlockedAt, completion metadata (6 fields), Metadata, RejectionCount, LastRejectionAt |
| Bug | embedded | `Status BugStatus` | Severity, LinkedEntityType, LinkedEntityKey |
| ChangeCard | embedded | `Status ChangeCardStatus` | Priority(int), RequestedBy, AssignedTo, EpicID, FeatureID, RelatedTaskID, Justification, ImpactAnalysis, RollbackPlan |

---

## 3. Go Embedding Mechanics

### 3.1 Field Promotion

Go promotes fields from embedded structs. After embedding:

```go
epic := &models.Epic{}
epic.Key = "E01"          // Works: promoted from BaseEntity
epic.Title = "My Epic"    // Works: promoted from BaseEntity
epic.Status = "active"    // Works: on Epic directly (not in BaseEntity)
epic.Priority = "high"    // Works: on Epic directly
```

All existing code accessing `entity.Key`, `entity.Title`, etc. continues to work without modification.

### 3.2 Method Promotion

Methods on `BaseEntity` are promoted to the embedding type:

```go
var entity models.Entity = epic
entity.GetKey()    // Calls BaseEntity.GetKey() via promotion
entity.GetStatus() // Calls Epic.GetStatus() (outer method takes precedence)
```

When the outer type declares a method with the same name as an embedded method, the outer method takes precedence. This is how GetStatus()/SetStatus() work correctly -- they are defined on each entity, not on BaseEntity.

### 3.3 JSON Serialization

Go's `encoding/json` flattens anonymous embedded struct fields:

```go
// Produces flat JSON, NOT nested:
{
  "id": 1,
  "key": "E01",
  "title": "My Epic",
  "status": "active",    // From Epic.Status (outer field)
  "priority": "high",    // From Epic.Priority
  "slug": null,          // From BaseEntity.Slug
  ...
}
```

There is no `"base_entity": {}` wrapper. JSON tags on BaseEntity fields produce the same output as the current per-entity tags because:
1. The tags are identical (verified in research)
2. `encoding/json` inlines anonymous struct fields
3. Status is on the outer struct, so no tag conflict occurs

### 3.4 Database Scanning

Repository `Scan()` calls use `&entity.FieldName` syntax. Go field promotion means:

```go
// This works identically before and after embedding:
err := row.Scan(
    &epic.ID,          // Resolves to &epic.BaseEntity.ID
    &epic.Key,         // Resolves to &epic.BaseEntity.Key
    &epic.Title,       // Resolves to &epic.BaseEntity.Title
    &epic.Description, // Resolves to &epic.BaseEntity.Description
    &epic.Status,      // Resolves to &epic.Status (outer field, not BaseEntity)
    &epic.Priority,    // Resolves to &epic.Priority (outer field)
    ...
)
```

No repository code changes are required. The research report confirmed this across all 5 repositories (~46 total scan sites).

---

## 4. Interface Satisfaction

### 4.1 Compile-Time Checks

The existing compile-time checks in `entity.go` remain unchanged:

```go
var (
    _ Entity = (*Epic)(nil)
    _ Entity = (*Feature)(nil)
    _ Entity = (*Task)(nil)
    _ Entity = (*Bug)(nil)
    _ Entity = (*ChangeCard)(nil)
)
```

After embedding, each type satisfies Entity through a combination of:
- **10 promoted methods** from BaseEntity (GetID, GetKey, GetTitle, GetSlug, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt)
- **4 per-entity methods** (GetEntityType, GetStatus, SetStatus, Validate)

### 4.2 Method Resolution Order

| Method | Source | Notes |
|--------|--------|-------|
| GetID() | BaseEntity (promoted) | Same implementation for all entities |
| GetKey() | BaseEntity (promoted) | Same implementation for all entities |
| GetTitle() | BaseEntity (promoted) | Same implementation for all entities |
| GetSlug() | BaseEntity (promoted) | Same implementation for all entities |
| GetEntityType() | Per-entity (required) | Returns unique EntityType constant |
| GetStatus() | Per-entity (required) | Converts typed status to string |
| SetStatus() | Per-entity (required) | Converts string to typed status |
| GetDescription() | BaseEntity (promoted) | Same implementation for all entities |
| GetFilePath() | BaseEntity (promoted) | Same implementation for all entities |
| GetContextData() | BaseEntity (promoted) | Same implementation for all entities |
| SetContextData() | BaseEntity (promoted) | Same implementation for all entities |
| GetCreatedAt() | BaseEntity (promoted) | Same implementation for all entities |
| GetUpdatedAt() | BaseEntity (promoted) | Same implementation for all entities |
| Validate() | Per-entity (required) | Entity-specific validation rules |

---

## 5. Impact Analysis

### 5.1 Files Modified Directly

| File | Change | Risk |
|------|--------|------|
| `internal/models/entity.go` | Add BaseEntity struct + 10 methods (~45 lines) | Low - additive only |
| `internal/models/epic.go` | Remove 9 fields + 10 methods, add BaseEntity embed | Medium - structural change |
| `internal/models/feature.go` | Remove 9 fields + 10 methods, add BaseEntity embed | Medium - structural change |
| `internal/models/task.go` | Remove 9 fields + 10 methods, add BaseEntity embed | Medium - structural change |
| `internal/models/bug.go` | Remove 9 fields + 10 methods, add BaseEntity embed | Medium - structural change |
| `internal/models/change_card.go` | Remove 9 fields + 10 methods, add BaseEntity embed | Medium - structural change |

**Total**: 6 files modified

### 5.2 Files NOT Modified (Verified Safe)

| Category | Files | Why Safe |
|----------|-------|----------|
| Repository layer | 5 files | Field promotion handles `&entity.Field` transparently |
| Service layer | ~15 files | Uses Entity interface or direct field access (both work via promotion) |
| CLI commands | ~20 files | Direct field access works via promotion |
| Tests | All existing | Go field promotion is transparent; tests pass without modification |

### 5.3 Quantitative Impact

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Shared field declarations | 50 (10 x 5) | 9 (in BaseEntity) | -41 declarations |
| Shared method implementations | 70 (14 x 5) | 10 (in BaseEntity) + 20 (4 per-entity x 5) | -40 implementations |
| Total lines in model files | ~485 | ~355 | -130 lines |
| Entity interface method sources | 5 files | 1 file (BaseEntity) + 5 files (per-entity) | Consolidated |
| Cost to add new shared field | 5 files | 1 file (BaseEntity) | 80% reduction |
| Cost to add new entity type | ~25 lines shared + specific | ~1 line embed + specific | ~90% reduction |

---

## 6. Cross-Feature Consistency

### 6.1 Alignment with E21 Architecture

The E21 epic architecture (architecture-design.md) was written before F07 was fully scoped. Key alignment points:

- **E21-F01 (completed)**: Established the Entity interface. F07 consolidates the _implementations_ of that interface without changing the interface itself.
- **E21-F02/F03 (completed/active)**: Use Entity interface for polymorphic operations. Unaffected by BaseEntity because they call interface methods, not struct fields directly.
- **E21-F08 (draft)**: Polymorphic Data Model Unification may build on BaseEntity for shared table schema patterns.

### 6.2 Architecture Decision Note

The E21 epic research report (Section 1) stated: "Struct Embedding (Rejected in scope.md): Embedding a BaseEntity struct would eliminate accessor boilerplate but breaks existing direct field access."

**This assessment was incorrect.** Go struct embedding with anonymous fields DOES promote fields transparently. `epic.Key` after embedding resolves to `epic.BaseEntity.Key` at compile time with zero runtime cost. The research report for F07 specifically verified this. The earlier rejection was based on a misunderstanding of Go's field promotion mechanics.

F07 uses Option B (Status excluded) which makes the refactoring fully backward-compatible with zero breaking changes.

---

## 7. Risks and Mitigations

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| 1 | JSON serialization produces nested output | Very Low | High | Go `encoding/json` flattens anonymous embeds. Verified in research. Validated by existing entity_test.go tests which check JSON output. |
| 2 | Database scan breaks after field reorder | Very Low | High | Field promotion is name-based, not order-based. `&epic.Key` resolves identically before and after. Repository tests cover all scan paths. |
| 3 | Method promotion conflict with future BaseEntity methods | Low | Medium | Only add methods to BaseEntity that are truly shared. Keep entity-specific methods on outer types. Compile-time checks catch conflicts immediately. |
| 4 | Merge conflicts with concurrent E21 work | Low | Low | F07 only modifies model struct definitions. No other active feature is modifying these struct field lists. |
| 5 | Test regression from subtle field ordering change | Very Low | Medium | Run `make test` after each entity migration. Entity_test.go has comprehensive coverage (431 lines) covering all 5 types. |

---

## 8. Constraints and Non-Functional Requirements

### 8.1 Backward Compatibility (REQ-NF-001)

- All existing field access patterns (`epic.Key`, `task.Status`, etc.) must continue to compile and work
- JSON serialization must produce identical output
- Database scanning must work without repository changes
- All existing tests must pass without modification

### 8.2 Performance

- Zero runtime cost. Go struct embedding is a compile-time mechanism. Method dispatch is identical (no virtual dispatch overhead vs. direct calls).
- Memory layout may differ slightly due to struct field ordering, but the difference is negligible (padding optimization is unchanged for the current field types).

### 8.3 Maintainability

- Adding a new shared field: Change 1 file (entity.go) instead of 5
- Adding a new entity type: Embed BaseEntity + add 4 methods (GetEntityType, GetStatus, SetStatus, Validate) instead of 14 methods
- Code review: Shared behavior centralized in one location

---

*Last Updated*: 2026-03-20
