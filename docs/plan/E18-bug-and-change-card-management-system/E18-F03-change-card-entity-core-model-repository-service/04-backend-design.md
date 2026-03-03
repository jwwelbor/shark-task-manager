# E18-F03: Change-Card Entity Core -- Backend Design

**Feature**: E18-F03
**Date**: 2026-03-03

---

## 1. Repository Interface

**Defined in**: `internal/services/change_card_service.go` (consumer-side, per project convention)
**Implemented by**: `internal/repository/change_card_repository.go`

```go
// ChangeCardRepository defines the data access interface needed by ChangeCardService.
// This interface is satisfied by *repository.ChangeCardRepository.
type ChangeCardRepository interface {
    // CRUD operations
    Create(ctx context.Context, card *models.ChangeCard) error
    GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
    GetByID(ctx context.Context, id int64) (*models.ChangeCard, error)
    Update(ctx context.Context, card *models.ChangeCard) error
    Delete(ctx context.Context, id int64) error

    // Status operations
    UpdateStatus(ctx context.Context, id int64, status models.ChangeCardStatus) error

    // Query operations
    List(ctx context.Context, filter *ChangeCardRepoFilter) ([]*models.ChangeCard, error)
    ListByLinkedEntity(ctx context.Context, entityType, entityKey string) ([]*models.ChangeCard, error)
    CountByStatus(ctx context.Context) (map[string]int, error)

    // Key generation
    GetNextKey(ctx context.Context) (string, error)
}
```

### 1.1 Repository Filter

```go
// ChangeCardRepoFilter represents filtering options for listing change-cards.
// Defined in repository package alongside the concrete implementation.
type ChangeCardRepoFilter struct {
    Status          *models.ChangeCardStatus
    LinkedEntityKey *string
    IncludeTerminal bool // if false, excludes completed + declined
}
```

### 1.2 Repository Implementation Notes

**File**: `internal/repository/change_card_repository.go`

| Method | SQL Pattern | Notes |
|--------|-------------|-------|
| `Create` | INSERT INTO change_cards (...) VALUES (?, ...) | Calls `Validate()` before insert; sets `idea.ID` from `LastInsertId()` |
| `GetByKey` | SELECT ... WHERE key = ? (then try slug match) | Dual-key lookup: exact key first, then key+slug parse |
| `GetByID` | SELECT ... WHERE id = ? | Standard ID lookup |
| `Update` | UPDATE change_cards SET ... WHERE id = ? | Calls `Validate()` before update; checks `RowsAffected()` |
| `Delete` | DELETE FROM change_cards WHERE id = ? | Checks `RowsAffected()` for not-found |
| `UpdateStatus` | UPDATE change_cards SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? | Simple status update |
| `List` | SELECT ... [WHERE clauses] ORDER BY created_at DESC | Dynamic WHERE based on filter fields |
| `ListByLinkedEntity` | SELECT ... WHERE linked_entity_type = ? AND linked_entity_key = ? | Uses composite index |
| `CountByStatus` | SELECT status, COUNT(*) FROM change_cards GROUP BY status | For analytics/dashboard |
| `GetNextKey` | SELECT COALESCE(MAX(CAST(SUBSTR(key, 2) AS INTEGER)), 0) FROM change_cards | Returns formatted `C###` string |

**Dual-Key Lookup Strategy** (in `GetByKey`):
1. Try exact match: `WHERE key = ?` (handles `C001`)
2. If not found and input contains a hyphen after `C###`:
   - Parse numeric key: `C001` from `C001-add-dark-mode`
   - Parse slug: `add-dark-mode`
   - Query: `WHERE key = ? AND slug = ?`
3. Case-insensitive: uppercase the `C` prefix before matching

---

## 2. Service Design

**File**: `internal/services/change_card_service.go`

### 2.1 Service Struct and Constructor

```go
// ChangeCardService provides business logic for change-card operations.
type ChangeCardService struct {
    db          *repository.DB       // for transaction management (CreateChangeCard)
    repo        ChangeCardRepository
    workflowSvc *workflow.Service
    epicRepo    EpicRepository       // for link validation (reuses existing interface)
    featureRepo FeatureRepository    // for link validation (reuses existing interface)
    projectRoot string               // for file path resolution
}

// NewChangeCardService creates a new ChangeCardService.
// The workflow service is automatically scoped to the change level.
//
// Required:
//   - db: database connection for transaction management
//   - repo: ChangeCardRepository for data access
//   - workflowSvc: workflow.Service (will be scoped to LevelChange internally)
//
// Optional (degrade gracefully when nil):
//   - epicRepo: for link validation of E## keys
//   - featureRepo: for link validation of E##-F## keys
//
// Panics:
//   - If db is nil (required dependency)
//   - If repo is nil (required dependency)
//   - If workflowSvc is nil (required dependency)
func NewChangeCardService(
    db *repository.DB,
    repo ChangeCardRepository,
    workflowSvc *workflow.Service,
    epicRepo EpicRepository,
    featureRepo FeatureRepository,
    projectRoot string,
) *ChangeCardService {
    requireNonNil(db, "ChangeCardService requires a non-nil *repository.DB")
    requireNonNil(repo, "ChangeCardService requires a non-nil ChangeCardRepository")
    requireNonNil(workflowSvc, "ChangeCardService requires a non-nil workflow.Service")
    return &ChangeCardService{
        db:          db,
        repo:        repo,
        workflowSvc: workflowSvc.ForLevel(workflow.LevelChange),
        epicRepo:    epicRepo,
        featureRepo: featureRepo,
        projectRoot: projectRoot,
    }
}
```

**Note**: `EpicRepository` and `FeatureRepository` interfaces are already defined in the services package (used by EpicService and FeatureService). The ChangeCardService reuses these interfaces -- it only needs `GetByKey` for link validation, which is part of both interfaces.

**Note**: `*repository.DB` is required because `CreateChangeCard` wraps DB insert + file write in a transaction (via `db.BeginTx()`). This matches the pattern used by BugService (F02).

### 2.2 DTOs

**File**: `internal/services/change_dto.go`

```go
// CreateChangeCardInput contains the parameters for creating a new change-card.
type CreateChangeCardInput struct {
    Title           string `json:"title"`
    LinkedEntityKey string `json:"linked_entity_key,omitempty"` // e.g., "E07" or "E07-F01"
}

// ChangeCardFilters contains filtering options for listing change-cards.
type ChangeCardFilters struct {
    Status          string `json:"status,omitempty"`
    LinkedEntityKey string `json:"linked_entity_key,omitempty"`
    ShowAll         bool   `json:"show_all,omitempty"` // include terminal statuses
}

// ChangeCardUpdates contains optional fields for updating a change-card.
type ChangeCardUpdates struct {
    Title           *string `json:"title,omitempty"`
    LinkedEntityKey *string `json:"linked_entity_key,omitempty"` // set to "" to unlink
}
```

### 2.3 Service Methods

#### CreateChangeCard

```
CreateChangeCard(ctx, input CreateChangeCardInput) (*models.ChangeCard, error)
```

**Flow**:
1. Validate title is non-empty
2. If `LinkedEntityKey` is provided:
   a. Detect entity type from key format (E## = epic, E##-F## = feature)
   b. Validate entity exists via `epicRepo.GetByKey()` or `featureRepo.GetByKey()`
   c. Reject task keys (E##-F##-###) with clear error
3. Generate next key via `repo.GetNextKey()`
4. Generate slug from title
5. Get default status from `workflowSvc.GetDefaultStatus()`
6. Build `ChangeCard` model
7. Call `model.Validate()`
8. **Begin transaction** via `s.db.BeginTx(ctx, nil)` (defer rollback)
9. Insert DB record via `repo.Create()`
10. Generate markdown content from template
11. Write file via `fileops.NewEntityFileWriter()`
12. If file write fails: rollback transaction, return error
13. Commit transaction
14. Return created change-card

#### GetChangeCard

```
GetChangeCard(ctx, key string) (*models.ChangeCard, error)
```

**Flow**: Delegate to `repo.GetByKey()` with error wrapping.

#### ListChangeCards

```
ListChangeCards(ctx, filters ChangeCardFilters) ([]*models.ChangeCard, error)
```

**Flow**:
1. Build repo filter from service filters
2. If `ShowAll` is false, set `IncludeTerminal = false` (excludes completed + declined)
3. Delegate to `repo.List()`
4. Return results (never nil, empty slice if no matches)

#### UpdateChangeCard

```
UpdateChangeCard(ctx, key string, updates ChangeCardUpdates) (*models.ChangeCard, error)
```

**Flow**:
1. Get existing change-card via `repo.GetByKey()`
2. Apply non-nil updates
3. If linked entity key changed, validate new link exists
4. Call `model.Validate()`
5. Save via `repo.Update()`
6. Return updated change-card

#### DeleteChangeCard

```
DeleteChangeCard(ctx, key string) error
```

**Flow**:
1. Get change-card via `repo.GetByKey()` (returns NotFoundError if missing)
2. Delete DB record via `repo.Delete()`
3. Delete markdown file (best-effort; log warning if file missing)

#### ApproveChangeCard

```
ApproveChangeCard(ctx, key string) (*models.ChangeCard, error)
```

**Flow**:
1. Get change-card via `repo.GetByKey()`
2. Validate transition `proposed -> approved` via `workflowSvc.ValidateTransition()`
3. If current status is not `proposed`, return error: "cannot approve change-card %s: current status is '%s', must be 'proposed'"
4. Update status via `repo.UpdateStatus()`
5. Return updated change-card

#### AdvanceChangeCardStatus

```
AdvanceChangeCardStatus(ctx, key string) (*models.ChangeCard, error)
```

**Flow**:
1. Get change-card via `repo.GetByKey()`
2. Get next status via `workflowSvc.GetNextStatus()`
3. If terminal (no next status), return error: "change-card %s is in terminal status '%s'; no further transitions available"
4. Validate transition via `workflowSvc.ValidateTransition()`
5. Update status via `repo.UpdateStatus()`
6. Return updated change-card

#### SetChangeCardStatus

```
SetChangeCardStatus(ctx, key string, targetStatus string) (*models.ChangeCard, error)
```

**Flow**:
1. Get change-card via `repo.GetByKey()`
2. Validate transition via `workflowSvc.ValidateTransition(current, target)`
3. Update status via `repo.UpdateStatus()`
4. Return updated change-card

#### CountByStatus

```
CountByStatus(ctx) (map[string]int, error)
```

**Flow**: Delegate to `repo.CountByStatus()`.

---

## 3. Service Accessor

**File**: `internal/cli/services_global.go` (addition)

```go
// GetChangeCardService returns a ChangeCardService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing GetDB pattern for CLI entry points).
func GetChangeCardService() *services.ChangeCardService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    changeCardRepo := repository.NewChangeCardRepository(db)
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    workflowSvc := GetWorkflowService()

    projectRoot, _ := FindProjectRoot()
    if projectRoot == "" {
        projectRoot = "."
    }

    return services.NewChangeCardService(db, changeCardRepo, workflowSvc, epicRepo, featureRepo, projectRoot)
}
```

---

## 4. Dependency Graph

```
ChangeCardService
├── *repository.DB (for transaction management)
├── ChangeCardRepository (interface)
│   └── *repository.ChangeCardRepository (implementation)
│       └── *repository.DB
├── *workflow.Service (scoped to LevelChange)
│   └── .sharkconfig.json (config file)
├── EpicRepository (interface, for link validation)
│   └── *repository.EpicRepository
│       └── *repository.DB
├── FeatureRepository (interface, for link validation)
│   └── *repository.FeatureRepository
│       └── *repository.DB
└── projectRoot string (for file path resolution)
```

---

## 5. Link Validation Logic

The service detects entity type from key format and validates existence:

```go
func (s *ChangeCardService) validateLink(ctx context.Context, key string) (entityType, entityKey string, err error) {
    key = strings.TrimSpace(key)
    if key == "" {
        return "", "", nil // no link -- valid
    }

    // Detect entity type from key format
    switch {
    case isTaskKey(key):  // E##-F##-### or T-E##-F##-###
        return "", "", fmt.Errorf("task linking is not supported for change-cards; link to epic or feature instead")
    case isFeatureKey(key):  // E##-F## or F##
        _, err := s.featureRepo.GetByKey(ctx, key)
        if err != nil {
            return "", "", fmt.Errorf("feature %s not found: %w", key, err)
        }
        return "feature", key, nil
    case isEpicKey(key):  // E##
        _, err := s.epicRepo.GetByKey(ctx, key)
        if err != nil {
            return "", "", fmt.Errorf("epic %s not found: %w", key, err)
        }
        return "epic", key, nil
    default:
        return "", "", fmt.Errorf("unrecognized entity key format: %s (expected E## for epic or E##-F## for feature)", key)
    }
}
```

Key format detection can reuse existing key parsing utilities from `internal/patterns/` or `internal/cli/commands/validators.go`.

---

## 6. Error Types

The service uses existing error patterns:

| Error | When | HTTP Status (future) |
|-------|------|---------------------|
| Repository "not found" error | GetByKey returns no rows | 404 |
| `fmt.Errorf("cannot approve...")` | ApproveChangeCard on wrong status | 422 |
| `fmt.Errorf("...terminal status...")` | AdvanceChangeCardStatus on completed/declined | 422 |
| `fmt.Errorf("feature %s not found...")` | Link validation fails | 400 |
| `fmt.Errorf("task linking not supported...")` | Task key provided as link | 400 |
| `fmt.Errorf("change-card title cannot be empty")` | Empty title in CreateChangeCard | 400 |

---

## 7. Test Plan Summary

### Model Tests (`internal/models/change_card_test.go`)

| Test | Input | Expected |
|------|-------|----------|
| Valid change-card | Title + status set | No error |
| Empty title | Title = "" | Error "title cannot be empty" |
| Empty status | Status = "" | Error "status cannot be empty" |
| Whitespace title | Title = "  " | Error "title cannot be empty" |

### Repository Tests (`internal/repository/change_card_repository_test.go`)

| Test | Description |
|------|-------------|
| Create + GetByKey | Round-trip CRUD |
| Key auto-generation | First = C001, next = C002 |
| Dual key lookup (numeric) | GetByKey("C001") |
| Dual key lookup (slugged) | GetByKey("C001-add-dark-mode") |
| List with status filter | Filter by "proposed" |
| List excluding terminal | ShowAll=false excludes completed/declined |
| ListByLinkedEntity | Filter by epic/feature link |
| Update | Modify title, verify updated_at changes |
| Delete | Delete and verify GetByKey returns not-found |
| UpdateStatus | Status change and verify |
| CountByStatus | Counts per status |

### Service Tests (`internal/services/change_card_service_test.go`)

| Test | Mocks | Verifies |
|------|-------|----------|
| CreateChangeCard (happy) | MockRepo.Create, MockRepo.GetNextKey | Key assigned, status = default, file created |
| CreateChangeCard (invalid link) | MockEpicRepo.GetByKey returns error | Error returned, no Create called |
| CreateChangeCard (task link rejected) | None | Error "task linking not supported" |
| ApproveChangeCard (happy) | MockRepo.GetByKey (status=proposed), MockWorkflow.ValidateTransition | Status updated to approved |
| ApproveChangeCard (wrong status) | MockRepo.GetByKey (status=in_progress) | Error with current status |
| AdvanceChangeCardStatus | MockRepo.GetByKey, MockWorkflow.GetNextStatus | Status advanced |
| AdvanceChangeCardStatus (terminal) | MockWorkflow.GetNextStatus returns error | Terminal status error |
| SetChangeCardStatus (valid) | MockWorkflow.ValidateTransition ok | Status updated |
| SetChangeCardStatus (invalid) | MockWorkflow.ValidateTransition error | Transition error |
| ListChangeCards (filters) | MockRepo.List | Correct filter passed to repo |
| GetChangeCard | MockRepo.GetByKey | Change-card returned |
| DeleteChangeCard | MockRepo.GetByKey + MockRepo.Delete | Both called |
| DeleteChangeCard (not found) | MockRepo.GetByKey returns error | Not-found error |

---

## 8. Context and Note Service Updates

### 8.1 ContextService Changes

In `internal/services/context_service.go`, the `getContextJSON` and `setContextJSON` functions switch on entity type. Add the change-card case:

```go
// In getContextJSON:
case models.EntityTypeChange:
    repo := s.changeCardRepo // ChangeCardContextRepo interface
    card, err := repo.GetByKey(ctx, entityKey)
    if err != nil {
        return nil, err
    }
    return card.ContextData, nil

// In setContextJSON:
case models.EntityTypeChange:
    repo := s.changeCardRepo // ChangeCardContextRepo interface
    return repo.UpdateContextData(ctx, entityKey, contextJSON)
```

**New interface for ContextService:**
```go
// ChangeCardContextRepo provides change-card context data access for ContextService.
type ChangeCardContextRepo interface {
    GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
    UpdateContextData(ctx context.Context, key string, contextData *string) error
}
```

The `UpdateContextData` method should be added to the concrete `ChangeCardRepository` implementation. It updates only the `context_data` column: `UPDATE change_cards SET context_data = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?`.

### 8.2 NoteService Changes

In `internal/services/note_service.go`, entity existence validation needs to accept "change". Add:

```go
case models.EntityTypeChange:
    _, err := s.changeCardRepo.GetByKey(ctx, entityKey)
    if err != nil {
        return fmt.Errorf("change-card not found: %s", entityKey)
    }
```

**New interface for NoteService:**
```go
// ChangeCardEntityValidator provides change-card existence checks for NoteService.
type ChangeCardEntityValidator interface {
    GetByKey(ctx context.Context, key string) (*models.ChangeCard, error)
}
```

### 8.3 Wiring for Context and Note Services

The `GetContextService` and `GetNoteService` functions in `services_global.go` need to be updated to inject a `ChangeCardRepository` so the services can handle the change entity type.

```go
// In GetContextService:
changeCardRepo := repository.NewChangeCardRepository(db)
globalContextService = services.NewContextService(epicRepo, featureRepo, taskRepo, bugRepo, changeCardRepo)

// In GetNoteService:
changeCardRepo := repository.NewChangeCardRepository(db)
globalNoteService = services.NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo, bugRepo, changeCardRepo)
```

**Note**: Both F02 (Bug) and F03 (Change-Card) add parameters to these constructors. If F02 is implemented first, F03 appends `changeCardRepo` as the last argument. If implemented simultaneously, coordinate the constructor signatures to include both `bugRepo` and `changeCardRepo`.

---

## 9. Implementation Order

Tasks should be created in this order within F03:

1. **Model**: `change_card.go` + `change_card_test.go` + entity_note.go modification + levels.go modification
2. **Repository**: `change_card_repository.go` + `change_card_repository_test.go` (include `UpdateContextData` method)
3. **Service DTOs**: `change_dto.go`
4. **Service**: `change_card_service.go` + `change_card_service_test.go`
5. **Service Accessor**: `services_global.go` addition (`GetChangeCardService`)
6. **Template**: Change-card markdown template (inline or .tmpl file)
7. **Context/Note Integration**: ContextService + NoteService switch-case additions + `services_global.go` wiring updates

Each step builds on the previous. Steps 1-2 can be verified independently. Steps 3-6 require F01 workflow engine to be in place for integration testing. Step 7 coordinates with F02 (Bug) which adds similar modifications for `entity_type = "bug"`.
