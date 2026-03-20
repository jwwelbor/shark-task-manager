---
feature_key: E21-F07-baseentity-struct-embedding
epic_key: E21
title: BaseEntity Struct Embedding
description: Eliminate field and method duplication across 5 entity model structs using Go struct embedding
---

# BaseEntity Struct Embedding

**Feature Key**: E21-F07-baseentity-struct-embedding

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

All 5 entity types (Epic, Feature, Task, Bug, ChangeCard) declare the same 10 fields independently: ID, Key, Title, Slug, Description, Status, FilePath, ContextData, CreatedAt, UpdatedAt. Each also manually implements 14 Entity interface methods (GetID, GetKey, GetTitle, etc.). This results in **50 duplicated field declarations** and **70+ duplicated method implementations** across the model layer.

When a shared field needs to change (e.g., adding a `DeletedAt` field, changing ContextData from `*string` to a typed struct), the change must be made in 5 places with 5 sets of tests.

### Solution

Create a `BaseEntity` embedded struct containing the 10 shared fields and their Entity interface method implementations. Each entity type embeds `BaseEntity` and only declares entity-specific fields. The Entity interface is satisfied by the embedded methods, with entity-specific overrides where needed (e.g., `GetEntityType()`, `Validate()`).

### Impact

- Eliminate **50 duplicated field declarations** (10 fields x 5 types)
- Eliminate **~70 duplicated method implementations** (14 methods x 5 types, minus entity-specific overrides)
- Adding a new shared field requires changing 1 struct instead of 5
- Adding a new entity type requires only entity-specific fields + 2-3 method overrides

---

## User Personas

### Persona 1: Go Developer (Maintainer)

**Goals Related to This Feature**:
1. Add shared fields once, not five times
2. Understand the entity model from one struct definition
3. Trust that all entity types behave consistently for shared operations

**Pain Points This Feature Addresses**:
- Reviewing PRs that touch entity models requires checking 5 files for consistency
- Forgetting to update one entity type when adding a shared field causes subtle bugs
- 70+ nearly-identical getter/setter methods obscure the actual differences between entity types

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want shared entity fields defined in one place so that adding a new shared field requires one change, not five.

**Acceptance Criteria**:
- [ ] `BaseEntity` struct contains: ID, Key, Title, Slug, Description, Status (string), FilePath, ContextData, CreatedAt, UpdatedAt
- [ ] All 5 entity types embed `BaseEntity`
- [ ] Entity-specific status types (EpicStatus, TaskStatus, etc.) still work via accessor methods
- [ ] All existing tests pass without modification (backward compatible)

**Story 2**: As a developer, I want Entity interface methods implemented on `BaseEntity` so that new entity types get them automatically.

**Acceptance Criteria**:
- [ ] `BaseEntity` implements: GetID, GetKey, GetTitle, GetSlug, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt
- [ ] Each entity type only overrides: GetEntityType() (required), Validate() (entity-specific rules)
- [ ] Compile-time interface satisfaction checks still pass for all 5 types

**Story 3**: As a developer, I want direct field access to still work so that existing code doesn't break.

**Acceptance Criteria**:
- [ ] `epic.Key`, `task.Title`, `feature.Status` etc. still compile and work (Go promotes embedded fields)
- [ ] JSON serialization produces identical output (json tags on BaseEntity fields)
- [ ] Database scan operations work with embedded struct fields

---

## Requirements

### Functional Requirements

**Category: Model Layer Refactoring**

1. **REQ-F-001**: BaseEntity Struct Definition
   - **Description**: Create `BaseEntity` struct in `internal/models/entity.go` with the 10 shared fields and json/db tags
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Fields: ID (int64), Key (string), Title (string), Slug (*string), Description (*string), Status (string), FilePath (*string), ContextData (*string), CreatedAt (time.Time), UpdatedAt (time.Time)
     - [ ] JSON tags match current per-entity tags exactly
     - [ ] Status stored as string in BaseEntity; entity-specific typed status accessible via typed accessor on each entity

2. **REQ-F-002**: Entity Type Refactoring
   - **Description**: Refactor Epic, Feature, Task, Bug, ChangeCard to embed BaseEntity and remove duplicated fields
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Each entity struct embeds `BaseEntity` as first field
     - [ ] Entity-specific fields remain on the outer struct (e.g., Task.AgentType, Bug.Severity)
     - [ ] Remove per-entity implementations of shared getter/setter methods
     - [ ] Keep entity-specific `GetEntityType()` and `Validate()` implementations

3. **REQ-F-003**: Status Type Compatibility
   - **Description**: Maintain typed status enums (EpicStatus, TaskStatus, etc.) alongside string-based BaseEntity.Status
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] BaseEntity.Status is `string` type for polymorphic operations
     - [ ] Each entity provides typed accessor: `func (e *Epic) TypedStatus() EpicStatus`
     - [ ] Existing code using `epic.Status` (typed) migrated to use either `epic.TypedStatus()` or `epic.BaseEntity.Status`
     - [ ] OR: Status remains typed per-entity and BaseEntity.GetStatus() returns `string(e.Status)` — evaluate which approach causes fewer changes

4. **REQ-F-004**: Repository Scan Compatibility
   - **Description**: Ensure database row scanning works with embedded struct fields
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] All repository Scan() calls updated to reference embedded fields correctly
     - [ ] No behavioral change in repository operations
     - [ ] All repository tests pass

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: Zero API Surface Change
   - **Description**: All existing code that accesses entity fields or calls Entity interface methods must continue to work without modification
   - **Measurement**: All existing tests pass without changes
   - **Justification**: This is a pure internal refactoring; callers should not need to change

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Field Access**
- **Given** an Epic struct with embedded BaseEntity
- **When** code accesses `epic.Key` or `epic.Title`
- **Then** Go field promotion makes these accessible identically to before

**Scenario 2: Interface Satisfaction**
- **Given** all 5 entity types embed BaseEntity
- **When** compile-time interface checks run
- **Then** all types satisfy the Entity interface

**Scenario 3: Serialization**
- **Given** a Task struct with embedded BaseEntity
- **When** serialized to JSON
- **Then** output is identical to current format (no nested "base_entity" object)

**Scenario 4: New Shared Field**
- **Given** a developer needs to add `DeletedAt *time.Time` to all entities
- **When** they add it to BaseEntity
- **Then** all 5 entity types automatically have the field and its accessor

---

## Out of Scope

### Explicitly Excluded

1. **Generic Repository**
   - **Why**: Repository refactoring is a separate concern (different SQL per entity type)
   - **Future**: May be addressed if entity tables are unified

2. **Status Type Unification**
   - **Why**: Merging EpicStatus/TaskStatus/etc. into a single type is a larger change
   - **Future**: Could happen in a separate feature if needed

---

## Design Notes

### Go Embedding Pattern

```go
// internal/models/entity.go
type BaseEntity struct {
    ID          int64     `json:"id"`
    Key         string    `json:"key"`
    Title       string    `json:"title"`
    Slug        *string   `json:"slug,omitempty"`
    Description *string   `json:"description,omitempty"`
    Status      string    `json:"status"`
    FilePath    *string   `json:"file_path,omitempty"`
    ContextData *string   `json:"context_data,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

func (b *BaseEntity) GetID() int64            { return b.ID }
func (b *BaseEntity) GetKey() string           { return b.Key }
func (b *BaseEntity) GetTitle() string         { return b.Title }
func (b *BaseEntity) GetSlug() string          { if b.Slug != nil { return *b.Slug }; return "" }
func (b *BaseEntity) GetStatus() string        { return b.Status }
func (b *BaseEntity) SetStatus(s string)       { b.Status = s }
func (b *BaseEntity) GetDescription() string   { if b.Description != nil { return *b.Description }; return "" }
func (b *BaseEntity) GetFilePath() string      { if b.FilePath != nil { return *b.FilePath }; return "" }
func (b *BaseEntity) GetContextData() *string  { return b.ContextData }
func (b *BaseEntity) SetContextData(d *string) { b.ContextData = d }
func (b *BaseEntity) GetCreatedAt() time.Time  { return b.CreatedAt }
func (b *BaseEntity) GetUpdatedAt() time.Time  { return b.UpdatedAt }

// internal/models/epic.go
type Epic struct {
    BaseEntity                    // Embeds all shared fields + methods
    Priority      Priority        `json:"priority"`
    BusinessValue *Priority       `json:"business_value,omitempty"`
}

func (e *Epic) GetEntityType() EntityType { return EntityTypeEpic }
func (e *Epic) Validate() error { /* epic-specific validation */ }
```

### Risk: Status Field Type Conflict

The biggest design decision is how to handle typed status fields. Current state:
- `Epic.Status` is `EpicStatus` (string typedef)
- `BaseEntity.Status` would be `string`

**Option A**: BaseEntity.Status is `string`, each entity adds a typed overlay method. Breaking change for code using `epic.Status` directly.

**Option B**: BaseEntity does NOT include Status. Each entity keeps its typed Status field and implements GetStatus()/SetStatus() individually. Less duplication reduction but zero breaking changes.

**Recommendation**: Option A with a migration. The typed status enums add little value since status validation happens in workflow.Service at runtime, not via type system.

---

## Dependencies & Integrations

### Dependencies

- **E21-F01**: Entity interface definition (completed) — this feature refactors the implementation of that interface
- **Repository layer**: All 5 repositories need Scan() updates

### Risks

- JSON serialization with embedded structs can produce unexpected output if tags conflict
- Database scanning with `sql.Scan()` needs careful field ordering
- Heavy use of direct field access (`task.Key`) throughout codebase means high touch-count

---

## Success Metrics

### Primary Metrics

1. **Field Declaration Reduction**
   - **Target**: 50 field declarations reduced to 10 (in BaseEntity) + entity-specific fields
   - **Measurement**: Count of shared field declarations across model files

2. **Method Implementation Reduction**
   - **Target**: ~70 method implementations reduced to 12 (in BaseEntity) + 10 overrides (GetEntityType x5, Validate x5)
   - **Measurement**: Count of Entity interface method implementations

3. **Zero Test Regression**
   - **Target**: All existing tests pass without modification
   - **Measurement**: `make test` green

---

*Last Updated*: 2026-03-20
