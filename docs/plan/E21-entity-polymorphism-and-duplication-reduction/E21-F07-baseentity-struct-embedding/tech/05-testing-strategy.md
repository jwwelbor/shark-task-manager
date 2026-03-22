# Testing Strategy: E21-F07 BaseEntity Struct Embedding

**Feature**: E21-F07
**Author**: Architect
**Date**: 2026-03-20

---

## 1. Testing Philosophy

This feature is a pure internal refactoring. The primary testing strategy is **existing test validation** -- all existing tests must pass without modification. New tests are limited to validating the BaseEntity struct itself.

### Test Categories

| Category | Action | Rationale |
|----------|--------|-----------|
| Existing model tests (entity_test.go) | Run without modification | Validates backward compatibility of all Entity interface methods across all 5 types |
| Existing repository tests | Run without modification | Validates database scan compatibility with embedded fields |
| Existing service tests | Run without modification | Validates Entity interface usage in polymorphic contexts |
| Existing CLI tests | Run without modification | Validates field access and JSON serialization in commands |
| New BaseEntity tests | Write new tests | Validates BaseEntity struct and methods in isolation |

---

## 2. New Tests: BaseEntity

### 2.1 Location

`internal/models/entity_test.go` (append to existing file)

### 2.2 Test Cases

**Test 1: BaseEntity field access**

```go
func TestBaseEntity_FieldAccess(t *testing.T) {
    slug := "test-slug"
    desc := "test description"
    filePath := "/path/to/file.md"
    contextData := `{"key":"value"}`

    base := BaseEntity{
        ID:          42,
        Key:         "E01",
        Title:       "Test Entity",
        Slug:        &slug,
        Description: &desc,
        FilePath:    &filePath,
        ContextData: &contextData,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    assert.Equal(t, int64(42), base.GetID())
    assert.Equal(t, "E01", base.GetKey())
    assert.Equal(t, "Test Entity", base.GetTitle())
    assert.Equal(t, "test-slug", base.GetSlug())
    assert.Equal(t, "test description", base.GetDescription())
    assert.Equal(t, "/path/to/file.md", base.GetFilePath())
    assert.Equal(t, &contextData, base.GetContextData())
    assert.False(t, base.GetCreatedAt().IsZero())
    assert.False(t, base.GetUpdatedAt().IsZero())
}
```

**Test 2: BaseEntity nil pointer fields**

```go
func TestBaseEntity_NilFields(t *testing.T) {
    base := BaseEntity{
        ID:    1,
        Key:   "E01",
        Title: "Test",
        // Slug, Description, FilePath, ContextData are nil
    }

    assert.Equal(t, "", base.GetSlug())
    assert.Equal(t, "", base.GetDescription())
    assert.Equal(t, "", base.GetFilePath())
    assert.Nil(t, base.GetContextData())
}
```

**Test 3: BaseEntity SetContextData**

```go
func TestBaseEntity_SetContextData(t *testing.T) {
    base := BaseEntity{}

    assert.Nil(t, base.GetContextData())

    data := `{"step":"implementing"}`
    base.SetContextData(&data)
    assert.Equal(t, &data, base.GetContextData())
    assert.Equal(t, `{"step":"implementing"}`, *base.GetContextData())

    base.SetContextData(nil)
    assert.Nil(t, base.GetContextData())
}
```

**Test 4: Embedding promotes fields correctly**

```go
func TestBaseEntity_EmbeddingPromotion(t *testing.T) {
    // Verify that embedding works as expected with Epic
    epic := &Epic{
        BaseEntity: BaseEntity{
            ID:    1,
            Key:   "E01",
            Title: "Test Epic",
        },
        Status:   EpicStatusDraft,
        Priority: PriorityMedium,
    }

    // Field promotion: direct access should work
    assert.Equal(t, int64(1), epic.ID)
    assert.Equal(t, "E01", epic.Key)
    assert.Equal(t, "Test Epic", epic.Title)

    // Method promotion: interface methods should work
    var entity Entity = epic
    assert.Equal(t, int64(1), entity.GetID())
    assert.Equal(t, "E01", entity.GetKey())
    assert.Equal(t, "Test Epic", entity.GetTitle())

    // Entity-specific methods should take precedence
    assert.Equal(t, EntityTypeEpic, entity.GetEntityType())
    assert.Equal(t, "draft", entity.GetStatus())
}
```

**Test 5: JSON serialization produces flat output**

```go
func TestBaseEntity_JSONSerialization(t *testing.T) {
    slug := "test-epic"
    epic := &Epic{
        BaseEntity: BaseEntity{
            ID:    1,
            Key:   "E01",
            Title: "Test Epic",
            Slug:  &slug,
        },
        Status:   EpicStatusDraft,
        Priority: PriorityMedium,
    }

    data, err := json.Marshal(epic)
    assert.NoError(t, err)

    // Verify flat JSON (no nested "BaseEntity" object)
    var result map[string]interface{}
    err = json.Unmarshal(data, &result)
    assert.NoError(t, err)

    assert.Equal(t, float64(1), result["id"])
    assert.Equal(t, "E01", result["key"])
    assert.Equal(t, "Test Epic", result["title"])
    assert.Equal(t, "test-epic", result["slug"])
    assert.Equal(t, "draft", result["status"])
    assert.Equal(t, "medium", result["priority"])

    // Verify no nested BaseEntity key
    _, hasBaseEntity := result["BaseEntity"]
    assert.False(t, hasBaseEntity, "JSON should not contain nested BaseEntity object")
    _, hasBase := result["base_entity"]
    assert.False(t, hasBase, "JSON should not contain nested base_entity object")
}
```

### 2.3 Test Count

| Test | Lines (est.) | Purpose |
|------|-------------|---------|
| TestBaseEntity_FieldAccess | 20 | All getter methods return correct values |
| TestBaseEntity_NilFields | 10 | Nil-safe behavior for optional pointer fields |
| TestBaseEntity_SetContextData | 12 | Mutator method correctness |
| TestBaseEntity_EmbeddingPromotion | 18 | Go field/method promotion works with Entity interface |
| TestBaseEntity_JSONSerialization | 22 | JSON output is flat, not nested |
| **Total** | **~82** | |

---

## 3. Existing Test Coverage (No Modification Required)

### 3.1 Model Tests

**File**: `internal/models/entity_test.go` (431 lines)

Coverage includes:
- Entity interface satisfaction for all 5 types
- GetID, GetKey, GetTitle, GetSlug, GetEntityType tests for each type
- GetStatus, SetStatus round-trip tests for each type
- GetDescription, GetFilePath, GetContextData with nil and non-nil values
- SetContextData mutation tests
- GetCreatedAt, GetUpdatedAt tests
- Validate() tests for each entity with valid and invalid inputs

**Why these pass without modification**: Tests create entity structs with field assignments like `epic.Key = "E01"`. After embedding, Go field promotion resolves `epic.Key` to `epic.BaseEntity.Key` transparently. Test assertions call Entity interface methods which are either promoted from BaseEntity or defined on the outer type.

### 3.2 Repository Tests

**Files**: `internal/repository/*_test.go`

Coverage includes:
- CRUD operations with database scan/insert for all entity types
- Field value round-trip through database
- Slug lookup
- Status update operations

**Why these pass without modification**: Repository Scan() calls use `&entity.Field` syntax. Field promotion handles `&epic.Key` identically to `&epic.BaseEntity.Key`.

### 3.3 Service Tests

**Files**: `internal/services/*_test.go`

Coverage includes:
- Entity interface method calls in polymorphic contexts
- Status transition validation
- Note, context, resume operations across entity types

**Why these pass without modification**: Services interact via Entity interface methods or typed concrete fields. Both pathways work unchanged after embedding.

---

## 4. Verification Protocol

### 4.1 Per-Entity Verification

After each entity migration (Tasks 2-6), run:

```bash
make fmt && make lint && make test
```

This must pass before proceeding to the next entity.

### 4.2 Full Verification (After All Migrations)

```bash
# Full quality gate
make fmt && make lint && make test

# JSON output spot-checks (manual)
./bin/shark epic get E21 --json 2>/dev/null | python3 -m json.tool | head -10
./bin/shark feature get E21-F07 --json 2>/dev/null | python3 -m json.tool | head -10
```

Verify JSON output has:
- Flat structure (no nested `base_entity` or `BaseEntity` object)
- All fields present with correct values
- `status` field present and correctly typed

### 4.3 Regression Risk Matrix

| Test Category | # Tests | Risk of Failure | Reason |
|---------------|---------|-----------------|--------|
| Model unit tests | ~50 | Very Low | Field promotion is transparent |
| Repository integration tests | ~30 | Very Low | Scan() works with promoted fields |
| Service unit tests | ~40 | Very Low | Entity interface contract unchanged |
| CLI integration tests | ~20 | Very Low | Field access and JSON unchanged |
| **Total** | **~140** | | |

---

## 5. Edge Cases to Validate

### 5.1 Task.ContextData Removal

Task currently declares ContextData as a standalone field. After embedding, this declaration must be removed because BaseEntity provides it. If both are present, Go reports a compile error for ambiguous field access.

**Validation**: Compile succeeds after removing Task.ContextData. Existing tests that set `task.ContextData` continue to work via promotion.

### 5.2 Struct Literal Initialization

Existing code that creates entity structs with field names will need to either:
- Use `BaseEntity: BaseEntity{...}` for shared fields (in new code)
- Continue using promoted field assignment `epic.Key = "E01"` (no change needed)

Struct literals like `Epic{Key: "E01", ...}` continue to work because Go allows setting promoted fields by name in struct literals.

**Important**: If any test or code uses positional (unnamed) struct initialization like `Epic{1, "E01", ...}`, it will break. However, this pattern is rare in Go and not used in this codebase (verified: all entity construction uses named fields).

### 5.3 Zero-Value Initialization

`Epic{}` creates an Epic with a zero-valued BaseEntity embed. All fields are zero-valued (0, "", nil, zero time). This is identical to the current behavior where `Epic{}` produces zero-valued fields.

---

## 6. Test Execution Order

1. **Before any code changes**: Run `make test` to establish baseline (all green)
2. **After Task 1** (BaseEntity definition): Run `make test` -- new tests pass, existing unchanged
3. **After Task 2** (Epic migration): Run `make test` -- all existing Epic tests still pass
4. **After Task 3** (Feature migration): Run `make test`
5. **After Task 4** (Bug migration): Run `make test`
6. **After Task 5** (ChangeCard migration): Run `make test`
7. **After Task 6** (Task migration): Run `make test`
8. **After Task 7** (cleanup): Final `make fmt && make lint && make test`

---

*Last Updated*: 2026-03-20
