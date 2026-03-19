# F01: Entity Interface Foundation -- Implementation Architecture

**Feature**: E21-F01
**Author**: Architect
**Date**: 2026-03-19
**Status**: Approved
**Tier**: STANDARD

This document is the developer implementation guide for F01. It refines the epic-level architecture-design.md based on actual codebase inspection (commit 7299cc1, branch file-path-updates).

---

## 1. Implementation Plan -- Files and Changes

### New Files

| File | Purpose | Est. Lines |
|------|---------|------------|
| `internal/models/entity.go` | Entity interface, compile-time checks | ~60 |
| `internal/services/entity_repository.go` | EntityRepository interface | ~30 |
| `internal/services/entity_registry.go` | EntityRegistry struct | ~55 |
| `internal/services/epic_repo_adapter.go` | EpicRepositoryAdapter | ~55 |
| `internal/services/feature_repo_adapter.go` | FeatureRepositoryAdapter | ~60 |
| `internal/services/task_repo_adapter.go` | TaskRepositoryAdapter | ~65 |
| `internal/services/bug_repo_adapter.go` | BugRepositoryAdapter | ~65 |
| `internal/services/change_repo_adapter.go` | ChangeCardRepositoryAdapter | ~65 |
| `internal/models/entity_test.go` | Interface satisfaction + accessor tests | ~200 |
| `internal/services/entity_registry_test.go` | Registry tests | ~120 |
| `internal/services/entity_adapter_test.go` | Adapter tests with mocks | ~200 |

### Modified Files

| File | Change | Risk |
|------|--------|------|
| `internal/models/change_card.go` | `Slug string` -> `*string`, `FilePath string` -> `*string` | Low |
| `internal/models/epic.go` | Add 13 accessor methods (Validate already exists) | Zero |
| `internal/models/feature.go` | Add 13 accessor methods | Zero |
| `internal/models/task.go` | Add 13 accessor methods | Zero |
| `internal/models/bug.go` | Add 13 accessor methods | Zero |
| `internal/models/change_card.go` | Add 13 accessor methods | Zero |
| `internal/repository/change_card_repository.go` | Scan `Slug`/`FilePath` into `*string` (3 locations) | Low |
| `internal/services/change_card_service.go` | Dereference/set `Slug`/`FilePath` as `*string` (6 locations) | Low |
| `internal/cli/commands/change_card_commands.go` | Nil-check `FilePath` before display (1 location) | Low |
| `internal/cli/commands/change.go` | Assign `FilePath` as `*string` (1 location) | Low |

---

## 2. Entity Interface -- Final Go Code

File: `internal/models/entity.go`

```go
package models

import "time"

// Entity is the polymorphic interface implemented by all domain entity types.
// It provides accessor methods for the shared fields common to all entities.
//
// This interface is additive -- existing direct field access (e.g., epic.Key)
// continues to work unchanged. The interface is used only by cross-cutting
// services that need to operate on entities polymorphically.
type Entity interface {
	GetID() int64
	GetKey() string
	GetTitle() string
	GetSlug() string
	GetEntityType() EntityType
	GetStatus() string
	SetStatus(status string)
	GetDescription() string
	GetFilePath() string
	GetContextData() *string
	SetContextData(data *string)
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	Validate() error
}

// Compile-time interface satisfaction checks.
var (
	_ Entity = (*Epic)(nil)
	_ Entity = (*Feature)(nil)
	_ Entity = (*Task)(nil)
	_ Entity = (*Bug)(nil)
	_ Entity = (*ChangeCard)(nil)
)
```

**Note on EntityType**: The `EntityType` type and its 5 constants already exist in `internal/models/entity_note.go`. No new type definition is needed. The existing `EntityTypeEpic`, `EntityTypeFeature`, `EntityTypeTask`, `EntityTypeChange`, `EntityTypeBug` constants are used directly.

---

## 3. Model Implementations -- Per-Entity Accessor Methods

Each model gets 13 new methods (Validate already exists on all 5). The pattern is identical across models; only the type differences in field types require attention.

### Fields with type variance across models

| Field | Epic | Feature | Task | Bug | ChangeCard (after normalization) |
|-------|------|---------|------|-----|----------------------------------|
| Slug | `*string` | `*string` | `*string` | `*string` | `*string` |
| FilePath | `*string` | `*string` | `*string` | `*string` | `*string` |
| Description | `*string` | `*string` | `*string` | `*string` | `*string` |
| ContextData | `*string` | `*string` | `*string` | `*string` | `*string` |
| Status | `EpicStatus` | `FeatureStatus` | `TaskStatus` | `BugStatus` | `ChangeCardStatus` |

All pointer-type fields use the same accessor pattern:

```go
func (e *Epic) GetSlug() string {
    if e.Slug != nil { return *e.Slug }
    return ""
}
```

### Epic Implementation (added to `internal/models/epic.go`)

```go
func (e *Epic) GetID() int64              { return e.ID }
func (e *Epic) GetKey() string            { return e.Key }
func (e *Epic) GetTitle() string          { return e.Title }
func (e *Epic) GetSlug() string {
    if e.Slug != nil { return *e.Slug }
    return ""
}
func (e *Epic) GetEntityType() EntityType { return EntityTypeEpic }
func (e *Epic) GetStatus() string         { return string(e.Status) }
func (e *Epic) SetStatus(status string)   { e.Status = EpicStatus(status) }
func (e *Epic) GetDescription() string {
    if e.Description != nil { return *e.Description }
    return ""
}
func (e *Epic) GetFilePath() string {
    if e.FilePath != nil { return *e.FilePath }
    return ""
}
func (e *Epic) GetContextData() *string     { return e.ContextData }
func (e *Epic) SetContextData(data *string) { e.ContextData = data }
func (e *Epic) GetCreatedAt() time.Time     { return e.CreatedAt }
func (e *Epic) GetUpdatedAt() time.Time     { return e.UpdatedAt }
```

### Feature, Task, Bug -- Same pattern

Feature, Task, and Bug follow the identical pattern with their respective status types (`FeatureStatus`, `TaskStatus`, `BugStatus`) and entity type constants. All `*string` fields use the nil-safe accessor shown above.

### ChangeCard -- Same pattern AFTER normalization

After ChangeCard Slug/FilePath are normalized to `*string` (see section 4), ChangeCard uses the same nil-safe accessor pattern as all other models.

---

## 4. ChangeCard Normalization -- Exact Changes

### 4.1 Model Change (`internal/models/change_card.go`)

```go
// BEFORE (lines 47-48)
Slug     string  `json:"slug,omitempty" db:"slug"`
FilePath string  `json:"file_path,omitempty" db:"file_path"`

// AFTER
Slug     *string `json:"slug,omitempty" db:"slug"`
FilePath *string `json:"file_path,omitempty" db:"file_path"`
```

### 4.2 Repository Scan (`internal/repository/change_card_repository.go`)

The `Scan` calls at lines 43, 63, and 150 already scan into `card.Slug` and `card.FilePath`. Since these become `*string`, SQLite's NULL handling works automatically with `database/sql` -- `sql.Scan` into a `*string` produces `nil` for NULL and a valid pointer for non-NULL values. **No scan logic changes needed** beyond the type change propagation.

The `Create` (line 63) and `Update` (line 150) INSERT/UPDATE statements pass `card.Slug` and `card.FilePath` as bind parameters. `database/sql` automatically maps `*string` nil to SQL NULL and `*string` non-nil to the string value. **No SQL changes needed**.

### 4.3 Service Changes (`internal/services/change_card_service.go`)

Six locations need updating:

| Line | Current | After |
|------|---------|-------|
| 152 | `card.FilePath = filePath` | `card.FilePath = &filePath` |
| 237 | `card.Slug = utils.GenerateSlug(card.Title)` | `slug := utils.GenerateSlug(card.Title); card.Slug = &slug` |
| 260-261 | `card.FilePath = *updates.FilePath` | `card.FilePath = updates.FilePath` (already `*string`) |
| 287 | `if card.FilePath != "" && s.projectRoot != ""` | `if card.FilePath != nil && *card.FilePath != "" && s.projectRoot != ""` |
| 288 | `absPath := filepath.Join(s.projectRoot, card.FilePath)` | `absPath := filepath.Join(s.projectRoot, *card.FilePath)` |
| 374 | `sb.WriteString(fmt.Sprintf("slug: %s\n", card.Slug))` | `if card.Slug != nil { sb.WriteString(fmt.Sprintf("slug: %s\n", *card.Slug)) }` |

### 4.4 CLI Command Changes

**`internal/cli/commands/change_card_commands.go`** (line 95-96):
```go
// BEFORE
if card.FilePath != "" {
    info = append(info, []string{"File", card.FilePath})

// AFTER
if card.FilePath != nil && *card.FilePath != "" {
    info = append(info, []string{"File", *card.FilePath})
```

**`internal/cli/commands/change.go`** (line 732):
```go
// BEFORE
updates.FilePath = &v

// AFTER -- no change needed if updates.FilePath is already *string in the DTO
updates.FilePath = &v
```

### 4.5 Test Updates

`internal/services/change_card_service_test.go` and `internal/models/change_card_test.go` -- update any string literal assignments to use `&stringVar` or helper `strPtr("value")` pattern.

---

## 5. EntityRepository Interface -- Final Go Code

File: `internal/services/entity_repository.go`

```go
package services

import (
    "context"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// EntityRepository provides polymorphic data access for any entity type.
// It wraps typed repositories to support cross-cutting operations.
//
// Implementations are thin adapters that delegate to typed repositories
// and perform type assertions where needed.
type EntityRepository interface {
    // GetByKey retrieves an entity by its key.
    GetByKey(ctx context.Context, key string) (models.Entity, error)

    // GetByID retrieves an entity by its database ID.
    GetByID(ctx context.Context, id int64) (models.Entity, error)

    // UpdateStatus updates the status field of an entity.
    UpdateStatus(ctx context.Context, id int64, status string) error

    // Update persists all fields of the entity.
    // The entity parameter must be the correct concrete type for this adapter.
    Update(ctx context.Context, entity models.Entity) error

    // GetContextData retrieves the context_data JSON string.
    GetContextData(ctx context.Context, id int64) (*string, error)

    // UpdateContextData updates the context_data JSON string.
    UpdateContextData(ctx context.Context, id int64, data *string) error
}
```

---

## 6. Adapter Pattern -- Concrete Example and Variations

### 6.1 Epic Adapter (full example)

File: `internal/services/epic_repo_adapter.go`

```go
package services

import (
    "context"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicRepositoryAdapter wraps EpicRepository to satisfy EntityRepository.
type EpicRepositoryAdapter struct {
    repo EpicRepository
}

func NewEpicRepositoryAdapter(repo EpicRepository) *EpicRepositoryAdapter {
    return &EpicRepositoryAdapter{repo: repo}
}

func (a *EpicRepositoryAdapter) GetByKey(ctx context.Context, key string) (models.Entity, error) {
    return a.repo.GetByKey(ctx, key)
}

func (a *EpicRepositoryAdapter) GetByID(ctx context.Context, id int64) (models.Entity, error) {
    return a.repo.GetByID(ctx, id)
}

func (a *EpicRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
    return a.repo.UpdateStatus(ctx, id, models.EpicStatus(status))
}

func (a *EpicRepositoryAdapter) Update(ctx context.Context, entity models.Entity) error {
    epic, ok := entity.(*models.Epic)
    if !ok {
        return fmt.Errorf("EpicRepositoryAdapter.Update: expected *models.Epic, got %T", entity)
    }
    return a.repo.Update(ctx, epic)
}

func (a *EpicRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
    return a.repo.GetContextData(ctx, id)
}

func (a *EpicRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
    return a.repo.UpdateContextData(ctx, id, data)
}
```

### 6.2 Variations by Entity

**Feature adapter**: Same as Epic but calls `repo.UpdateStatus(ctx, id, models.FeatureStatus(status))`. Note: Feature repository has `UpdateStatusIfNotOverridden` instead of plain `UpdateStatus`. The adapter should call the existing `Update` method after setting the status field, or the `FeatureRepository` interface in `feature_service.go` needs an `UpdateStatus` method added. Decision: Use the `Update` approach (get-set-update) which is consistent and safe.

```go
func (a *FeatureRepositoryAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
    feature, err := a.repo.GetByID(ctx, id)
    if err != nil {
        return fmt.Errorf("failed to get feature for status update: %w", err)
    }
    feature.Status = models.FeatureStatus(status)
    return a.repo.Update(ctx, feature)
}
```

**Task adapter**: Task repository's `UpdateStatus` has a different signature (`key string, status string, ...`). The adapter normalizes this by doing get-set-update like Feature.

**Bug and ChangeCard adapters**: Their repos have `UpdateStatus(ctx, id, TypedStatus)` which maps cleanly. For `GetContextData`/`UpdateContextData`, these repos lack dedicated methods, so the adapter uses the get-field-update pattern:

```go
func (a *BugRepositoryAdapter) GetContextData(ctx context.Context, id int64) (*string, error) {
    bug, err := a.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    return bug.ContextData, nil
}

func (a *BugRepositoryAdapter) UpdateContextData(ctx context.Context, id int64, data *string) error {
    bug, err := a.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }
    bug.ContextData = data
    return a.repo.Update(ctx, bug)
}
```

### 6.3 Interface Requirements on Typed Repos

Each adapter depends on the existing service-level typed repository interface (e.g., `EpicRepository` in `epic_service.go`). These interfaces already include `GetByKey`, `GetByID`, and `Update`. Some may need minor additions:

| Typed Interface | Methods Needed by Adapter | Already Present? |
|-----------------|--------------------------|------------------|
| `EpicRepository` | GetByKey, GetByID, Update, UpdateStatus, GetContextData, UpdateContextData | Yes (all) |
| `FeatureRepository` | GetByKey, GetByID, Update, GetContextData, UpdateContextData | Yes (all) |
| `TaskRepository` | GetByKey, GetByID, Update | Yes |
| `BugRepository` | GetByKey, GetByID, Update, UpdateStatus | Yes |
| `ChangeCardRepository` | GetByKey, GetByID, Update, UpdateStatus, UpdateContextData | Yes |

No typed repository interface changes are required.

---

## 7. EntityRegistry -- Integration with services_global.go

File: `internal/services/entity_registry.go`

The implementation matches the epic architecture doc exactly (Section 3). The registry is thread-safe with `sync.RWMutex` and provides `Register`, `GetRepository`, `MustGetRepository`, and `RegisteredTypes`.

### CLI Initialization Pattern

The registry is NOT added to `services_global.go` in F01. F01 only creates the registry code and tests. Integration into `services_global.go` happens in F02 when NoteService/ContextService/ResumeService are refactored to accept the registry.

However, the initialization pattern is designed now for F02 to use:

```go
// Future: internal/cli/services_global.go (F02 scope)
var (
    globalRegistry *services.EntityRegistry
    registryOnce   sync.Once
)

func GetEntityRegistry() *services.EntityRegistry {
    registryOnce.Do(func() {
        db, err := GetDB(context.Background())
        if err != nil {
            panic(fmt.Sprintf("failed to get database for EntityRegistry: %v", err))
        }

        globalRegistry = services.NewEntityRegistry()
        globalRegistry.Register(models.EntityTypeEpic,
            services.NewEpicRepositoryAdapter(repository.NewEpicRepository(db)))
        globalRegistry.Register(models.EntityTypeFeature,
            services.NewFeatureRepositoryAdapter(repository.NewFeatureRepository(db)))
        globalRegistry.Register(models.EntityTypeTask,
            services.NewTaskRepositoryAdapter(repository.NewTaskRepository(db)))
        globalRegistry.Register(models.EntityTypeBug,
            services.NewBugRepositoryAdapter(repository.NewBugRepository(db)))
        globalRegistry.Register(models.EntityTypeChange,
            services.NewChangeCardRepositoryAdapter(repository.NewChangeCardRepository(db)))
    })
    return globalRegistry
}
```

---

## 8. Test Strategy

### 8.1 Entity Interface Tests (`internal/models/entity_test.go`)

**What to test:**
- Compile-time satisfaction (handled by `var _ Entity = (*Epic)(nil)` in entity.go)
- Each accessor returns the correct value for each of the 5 models
- Nil-safe accessors return empty string for nil `*string` fields
- SetStatus mutates the status field correctly
- SetContextData mutates correctly

**Pattern**: Table-driven test over all 5 model types using a factory function.

```go
func TestEntityAccessors(t *testing.T) {
    entities := map[string]models.Entity{
        "epic":        &models.Epic{ID: 1, Key: "E01", Title: "Epic", Status: "active", ...},
        "feature":     &models.Feature{ID: 2, Key: "E01-F01", Title: "Feature", Status: "active", ...},
        "task":        &models.Task{ID: 3, Key: "T-E01-F01-001", Title: "Task", Status: "todo", ...},
        "bug":         &models.Bug{ID: 4, Key: "B001", Title: "Bug", Status: "open", ...},
        "change_card": &models.ChangeCard{ID: 5, Key: "CC-001", Title: "Change", Status: "draft", ...},
    }
    for name, e := range entities {
        t.Run(name, func(t *testing.T) {
            assert.NotZero(t, e.GetID())
            assert.NotEmpty(t, e.GetKey())
            assert.NotEmpty(t, e.GetTitle())
            assert.NotEmpty(t, e.GetStatus())
            // ... etc
        })
    }
}
```

### 8.2 Registry Tests (`internal/services/entity_registry_test.go`)

**What to test:**
- Register + GetRepository returns correct adapter
- GetRepository for unregistered type returns error
- MustGetRepository panics for unregistered type
- Duplicate Register panics
- RegisteredTypes returns all registered types
- Thread safety (concurrent Register/GetRepository)

**Pattern**: Unit tests with mock EntityRepository (simple struct satisfying the interface).

### 8.3 Adapter Tests (`internal/services/entity_adapter_test.go`)

**What to test:**
- Each adapter's GetByKey delegates to typed repo and returns models.Entity
- Each adapter's UpdateStatus calls typed repo with correct typed status
- Each adapter's Update rejects wrong concrete type with clear error
- Each adapter's GetContextData/UpdateContextData works (direct or get-set-update pattern)

**Pattern**: Mock typed repositories with function fields. Test one adapter thoroughly (Epic), then parameterize for the other 4.

### 8.4 ChangeCard Normalization Tests

Update existing tests in `internal/models/change_card_test.go` and `internal/services/change_card_service_test.go` to use `*string` for Slug/FilePath. Add a helper:

```go
func strPtr(s string) *string { return &s }
```

---

## 9. Implementation Order -- Recommended Task Sequence

### Task 1: ChangeCard Slug/FilePath Normalization (prerequisite)

**Files**: `change_card.go`, `change_card_repository.go`, `change_card_service.go`, `change_card_commands.go`, `change.go`, related tests
**Why first**: All other tasks assume uniform `*string` types across all models.
**Validation**: `make fmt && make lint && make test` -- all existing tests must pass.

### Task 2: Entity Interface + Implementations

**Files**: `entity.go` (new), `epic.go`, `feature.go`, `task.go`, `bug.go`, `change_card.go` (accessor additions), `entity_test.go` (new)
**Why second**: Pure additions -- zero risk to existing code. Creates the foundation for tasks 3-4.
**Validation**: `make build` (compile-time checks), `make test` (accessor tests).

### Task 3: EntityRepository Interface + 5 Adapters

**Files**: `entity_repository.go` (new), 5 adapter files (new), `entity_adapter_test.go` (new)
**Why third**: Depends on Entity interface (task 2) and typed repo interfaces (already exist).
**Validation**: Adapter tests with mocked typed repos.

### Task 4: EntityRegistry

**Files**: `entity_registry.go` (new), `entity_registry_test.go` (new)
**Why last**: Depends on EntityRepository interface (task 3). Smallest piece, cleanly isolated.
**Validation**: Registry unit tests.

### Dependency Graph

```
Task 1 (ChangeCard normalization)
    |
    v
Task 2 (Entity interface + implementations)
    |
    v
Task 3 (EntityRepository + adapters)
    |
    v
Task 4 (EntityRegistry)
```

All tasks are strictly sequential. Total estimated effort: M (4 tasks, each S-sized).

---

*Derived from epic architecture-design.md and validated against codebase at commit 7299cc1.*
