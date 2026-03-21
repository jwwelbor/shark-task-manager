# Migration Strategy: E21-F07 BaseEntity Struct Embedding

**Feature**: E21-F07
**Author**: Architect
**Date**: 2026-03-20

---

## 1. Migration Approach: Sequential Per-Entity

### Strategy

Migrate one entity type at a time, in order of increasing complexity. After each entity migration, run the full quality gate (`make fmt && make lint && make test`) to catch regressions immediately.

### Rationale

- Early detection of systematic issues (if embedding breaks something, it shows on the first entity)
- Smaller, reviewable changesets
- Easy rollback (revert one entity, not all five)
- Validates the pattern before applying to the most complex entity (Task)

---

## 2. Execution Order

### Phase 1: Foundation

**Task 1: Define BaseEntity struct and methods**

File: `internal/models/entity.go`

Steps:
1. Add `BaseEntity` struct definition with 9 fields after the Entity interface definition
2. Add 10 method implementations on `*BaseEntity`
3. Add dedicated tests for BaseEntity methods in `entity_test.go`
4. Run quality gate

**Estimated size**: ~45 lines added to entity.go, ~40 lines tests

**Exit criteria**: BaseEntity compiles, tests pass, no existing tests broken

### Phase 2: Entity Migrations (One at a Time)

**Task 2: Migrate Epic** (simplest -- 3 entity-specific fields)

File: `internal/models/epic.go`

Steps:
1. Replace 9 shared field declarations with `BaseEntity` embed as first field
2. Keep Status, Priority, BusinessValue, Metadata on outer struct
3. Remove 10 shared method implementations (GetID, GetKey, GetTitle, GetSlug, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt)
4. Keep 4 methods: GetEntityType, GetStatus, SetStatus, Validate
5. Run quality gate

**Lines removed**: ~35 (9 fields + 10 methods)
**Lines added**: 1 (BaseEntity embed line)

**Task 3: Migrate Feature** (4 entity-specific fields + StatusOverride, ProgressPct)

File: `internal/models/feature.go`

Steps: Same pattern as Epic. Keep EpicID, Status, StatusOverride, ProgressPct, ExecutionOrder, Metadata on outer struct. Also keep `IsAutoStatus()` helper method.

**Task 4: Migrate Bug** (3 entity-specific fields)

File: `internal/models/bug.go`

Steps: Same pattern. Keep Status, Severity, LinkedEntityType, LinkedEntityKey on outer struct.

**Task 5: Migrate ChangeCard** (10 entity-specific fields)

File: `internal/models/change_card.go`

Steps: Same pattern. Keep all ChangeCard-specific fields on outer struct.

**Task 6: Migrate Task** (19 entity-specific fields -- highest risk)

File: `internal/models/task.go`

Steps: Same pattern. Special attention:
- Task has a standalone `ContextData *string` field declaration that must be removed (it will be promoted from BaseEntity)
- Task has the most entity-specific fields (19), so careful field ordering is needed
- Task's Validate() method is the most complex -- it remains unchanged

Migrate Task last because:
1. It has the most fields, creating the highest risk of a missed field
2. By this point, the pattern is proven on 4 simpler entities
3. Task has the most repository scan calls (~14), so it validates the scan compatibility claim most thoroughly

### Phase 3: Cleanup

**Task 7: Verification and cleanup**

Steps:
1. Run full test suite: `make test`
2. Run linting: `make lint`
3. Spot-check JSON output for each entity type via CLI:
   - `shark epic get E21 --json | head -5`
   - `shark feature get E21-F07 --json | head -5`
   - `shark task list E21 F01 --json | head -5`
4. Remove any dead code or comments referencing old field declarations
5. Verify compile-time interface satisfaction checks still present

---

## 3. Per-Entity Migration Checklist

Use this checklist for each entity (Tasks 2-6):

```
[ ] Replace shared field declarations with `BaseEntity` embed
    [ ] ID int64
    [ ] Key string
    [ ] Title string
    [ ] Slug *string
    [ ] Description *string
    [ ] FilePath *string
    [ ] ContextData *string
    [ ] CreatedAt time.Time
    [ ] UpdatedAt time.Time
[ ] Keep entity-specific fields on outer struct
[ ] Keep typed Status field on outer struct
[ ] Remove shared method implementations:
    [ ] GetID()
    [ ] GetKey()
    [ ] GetTitle()
    [ ] GetSlug()
    [ ] GetDescription()
    [ ] GetFilePath()
    [ ] GetContextData()
    [ ] SetContextData()
    [ ] GetCreatedAt()
    [ ] GetUpdatedAt()
[ ] Keep entity-specific methods:
    [ ] GetEntityType()
    [ ] GetStatus()
    [ ] SetStatus()
    [ ] Validate()
[ ] Run: make fmt && make lint && make test
[ ] All tests pass without modification
```

---

## 4. Field Ordering Convention

After embedding, entity structs follow this field ordering convention:

```go
type Entity struct {
    BaseEntity                   // 1. Embedded struct (always first)
    // Parent references
    ParentID     int64           // 2. Foreign keys to parent entities
    // Core entity fields
    Status       TypedStatus     // 3. Status (typed per-entity)
    // Entity-specific fields
    Field1       Type            // 4. Entity-specific fields
    Field2       Type
    // Derived/non-persisted fields
    Metadata     map[...]        // 5. Metadata (last, db:"-")
}
```

This ordering ensures:
- BaseEntity is always the first field (convention for embedded structs)
- Status immediately follows BaseEntity for visual clarity
- Entity-specific fields are grouped logically
- Non-persisted derived fields are at the end

---

## 5. Rollback Plan

### Per-Entity Rollback

If a single entity migration causes issues:
1. Revert the changes to that entity's model file
2. Re-add the 9 shared field declarations
3. Re-add the 10 shared method implementations
4. Run quality gate to confirm rollback

### Full Rollback

If the entire approach is found to be problematic:
1. Revert all entity model files to pre-migration state
2. Remove BaseEntity struct and methods from entity.go
3. Remove BaseEntity-specific tests
4. Run quality gate

BaseEntity in entity.go can remain as dead code temporarily without causing issues (it has no callers if no entity embeds it).

---

## 6. Verification Checklist

### After All Migrations Complete

```
[ ] make fmt passes (no formatting changes)
[ ] make lint passes (no linting errors)
[ ] make test passes (all tests green)
[ ] JSON output unchanged for each entity type
[ ] Compile-time interface satisfaction checks present in entity.go
[ ] No duplicate field declarations remain across model files
[ ] BaseEntity has exactly 9 fields and 10 methods
[ ] Each entity has exactly 4 per-entity methods (GetEntityType, GetStatus, SetStatus, Validate)
[ ] entity_test.go passes without modification (431 lines of existing tests)
```

---

*Last Updated*: 2026-03-20
