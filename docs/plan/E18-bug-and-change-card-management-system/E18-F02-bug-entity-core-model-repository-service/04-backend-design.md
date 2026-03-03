# E18-F02: Bug Entity Core -- Backend Design

**Feature**: Bug Entity Core (Model, Repository, Service)
**Date**: 2026-03-03

---

## 1. BugService API Contracts

### 1.1 Constructor

```go
// NewBugService creates a new BugService with required and optional dependencies.
//
// Required:
//   - db: database connection for transaction management
//   - repo: BugRepository for data access
//   - workflowSvc: workflow.Service (will be scoped to LevelBug internally)
//
// Optional (degrade gracefully when nil):
//   - epicRepo: for link validation of E## keys
//   - featureRepo: for link validation of E##-F## keys
//   - taskRepo: for link validation of E##-F##-### keys
func NewBugService(
    db *repository.DB,
    repo BugRepository,
    workflowSvc *workflow.Service,
    epicRepo LinkValidatorEpicRepo,
    featureRepo LinkValidatorFeatureRepo,
    taskRepo LinkValidatorTaskRepo,
) *BugService
```

**Internal state:**
```go
type BugService struct {
    db          *repository.DB
    repo        BugRepository
    workflowSvc *workflow.Service          // Scoped to LevelBug in constructor
    epicRepo    LinkValidatorEpicRepo      // For link validation (optional)
    featureRepo LinkValidatorFeatureRepo   // For link validation (optional)
    taskRepo    LinkValidatorTaskRepo      // For link validation (optional)
}
```

The constructor calls `workflowSvc.ForLevel(workflow.LevelBug)` to scope the workflow service.

---

### 1.2 CreateBug

```go
func (s *BugService) CreateBug(ctx context.Context, input CreateBugInput) (*models.Bug, error)
```

**Flow:**

1. Validate input (title non-empty, severity valid if provided)
2. Set defaults: severity = `medium` if empty, status = `workflowSvc.GetDefaultStatus()`
3. If `LinkedEntityKey` is provided:
   a. Detect entity type from key format (E## = epic, E##-F## = feature, E##-F##-### = task)
   b. Call appropriate repository's `GetByKey` to verify existence
   c. Set `LinkedEntityType` accordingly
   d. If entity not found, return error: `"linked entity not found: <key>"`
4. Generate next key via `repo.GetNextKey(ctx)`
5. Generate slug from title
6. Begin database transaction
7. Create Bug model and call `repo.Create(ctx, bug)`
8. Render markdown template and write file via `fileops.WriteEntityFile`
9. If file write fails, transaction rolls back automatically (defer)
10. Commit transaction
11. Return created bug

**Error Cases:**
- Empty title: `"title is required"`
- Invalid severity: `"invalid severity '<value>': must be one of critical, high, medium, low"`
- Linked entity not found: `"linked entity not found: <key>"`
- Unrecognized key format for link: `"unrecognized entity key format: <key>"`
- File creation failure: `"failed to create bug file: <reason> (database changes rolled back)"`

---

### 1.3 GetBug

```go
func (s *BugService) GetBug(ctx context.Context, key string) (*models.Bug, error)
```

**Flow:**
1. Call `repo.GetByKey(ctx, key)`
2. Return bug or propagate `NotFoundError`

---

### 1.4 UpdateBug

```go
func (s *BugService) UpdateBug(ctx context.Context, key string, updates BugUpdates) (*models.Bug, error)
```

**Flow:**
1. Get existing bug via `repo.GetByKey`
2. Apply non-nil fields from `BugUpdates`
3. Validate updated model via `bug.Validate()`
4. Call `repo.Update(ctx, bug)`
5. Return updated bug

**Constraints:**
- Cannot change bug key
- Cannot change status via UpdateBug (use SetBugStatus or AdvanceBugStatus)

---

### 1.5 DeleteBug

```go
func (s *BugService) DeleteBug(ctx context.Context, key string) error
```

**Flow:**
1. Call `repo.Delete(ctx, key)`
2. Propagate `NotFoundError` if bug does not exist

---

### 1.6 ListBugs

```go
func (s *BugService) ListBugs(ctx context.Context, filters BugFilters) ([]*models.Bug, error)
```

**Flow:**
1. Convert `BugFilters` DTO to `BugListFilters` repository struct
2. Call `repo.List(ctx, repoFilters)`
3. Return results (empty slice if no matches, never nil)

---

### 1.7 AdvanceBugStatus

```go
func (s *BugService) AdvanceBugStatus(ctx context.Context, key string) (*models.Bug, error)
```

**Flow:**
1. Get bug via `repo.GetByKey`
2. Check if current status is terminal: `workflowSvc.IsTerminalStatus(bug.Status)`
   - If terminal, return error: `"cannot advance bug <key>: status '<status>' is terminal"`
3. Get next status: `workflowSvc.GetNextStatus(bug.Status)`
4. Update status: `repo.UpdateStatus(ctx, bug.ID, nextStatus)`
5. Reload and return updated bug

---

### 1.8 SetBugStatus

```go
func (s *BugService) SetBugStatus(ctx context.Context, key string, targetStatus string) (*models.Bug, error)
```

**Flow:**
1. Get bug via `repo.GetByKey`
2. Validate transition: `workflowSvc.ValidateTransition(bug.Status, targetStatus)`
   - If invalid, return workflow error
3. Update status: `repo.UpdateStatus(ctx, bug.ID, targetStatus)`
4. Reload and return updated bug

---

### 1.9 TriageBug

```go
func (s *BugService) TriageBug(ctx context.Context, input TriageBugInput) (*models.Bug, error)
```

**Flow:**
1. Get bug via `repo.GetByKey(ctx, input.Key)`
2. Validate current status is `reported`:
   - If not, return: `"cannot triage bug <key>: current status is '<status>', must be 'reported'"`
3. Begin transaction
4. If `input.Severity` is provided and valid, update severity on the bug model
5. If `input.AssignedTo` is provided, merge into `context_data` JSON:
   ```go
   contextMap := parseContextData(bug.ContextData)
   contextMap["assigned_to"] = input.AssignedTo
   bug.ContextData = marshalContextData(contextMap)
   ```
6. Call `repo.Update(ctx, bug)` to persist severity + context_data changes
7. Advance status from `reported` to `triaged`:
   - Validate transition via workflow service
   - Call `repo.UpdateStatus(ctx, bug.ID, "triaged")`
8. Commit transaction
9. Reload and return updated bug

**Error Cases:**
- Bug not found: propagated `NotFoundError`
- Wrong status: `"cannot triage bug <key>: current status is '<status>', must be 'reported'"`
- Invalid severity: `"invalid severity '<value>': must be one of critical, high, medium, low"`
- Workflow transition error: propagated from workflow service
- Transaction failure: all changes rolled back

---

## 2. Service DTOs (`internal/services/bug_dto.go`)

```go
package services

// CreateBugInput contains the parameters for creating a new bug.
type CreateBugInput struct {
    // Title is the bug title (required).
    Title string `json:"title"`

    // Description is an optional longer description of the bug.
    Description string `json:"description,omitempty"`

    // Severity sets the initial severity (optional, defaults to "medium").
    // Must be one of: critical, high, medium, low.
    Severity string `json:"severity,omitempty"`

    // LinkedEntityKey optionally links the bug to an epic, feature, or task.
    // Entity type is auto-detected from key format.
    LinkedEntityKey string `json:"linked_entity_key,omitempty"`
}

// TriageBugInput contains the parameters for the triage operation.
type TriageBugInput struct {
    // Key is the bug key to triage (required).
    Key string `json:"key"`

    // Severity optionally updates the severity during triage.
    Severity string `json:"severity,omitempty"`

    // AssignedTo optionally sets the assigned person as a context field.
    AssignedTo string `json:"assigned_to,omitempty"`
}

// BugUpdates contains the fields that can be updated on an existing bug.
// Only non-nil pointer fields are applied (partial update pattern).
type BugUpdates struct {
    // Title updates the bug title if non-nil.
    Title *string `json:"title,omitempty"`

    // Description updates the description if non-nil.
    Description *string `json:"description,omitempty"`

    // Severity updates the severity if non-nil.
    // Must be one of: critical, high, medium, low.
    Severity *string `json:"severity,omitempty"`
}

// BugFilters defines filtering options for listing bugs.
type BugFilters struct {
    // Status filters bugs by workflow status.
    Status string `json:"status,omitempty"`

    // Severity filters bugs by severity level.
    Severity string `json:"severity,omitempty"`

    // LinkedEntityKey filters bugs linked to a specific entity.
    LinkedEntityKey string `json:"linked_entity_key,omitempty"`

    // ShowAll includes resolved/terminal bugs (default: false, hides terminal).
    ShowAll bool `json:"show_all,omitempty"`
}
```

---

## 3. Link Validation Logic

The entity type detection logic for `LinkedEntityKey`:

```go
func (s *BugService) detectEntityType(key string) (string, error) {
    // Try each pattern in order of specificity (most specific first)
    if models.ValidateTaskKey(key) == nil || isShortTaskKey(key) {
        return "task", nil
    }
    if models.ValidateFeatureKey(key) == nil {
        return "feature", nil
    }
    if models.ValidateEpicKey(key) == nil {
        return "epic", nil
    }
    return "", fmt.Errorf("unrecognized entity key format: %s", key)
}

func (s *BugService) validateLinkedEntity(ctx context.Context, entityType, key string) error {
    switch entityType {
    case "epic":
        if s.epicRepo == nil {
            return fmt.Errorf("link validation unavailable: epic repository not configured")
        }
        _, err := s.epicRepo.GetByKey(ctx, key)
        if err != nil {
            return fmt.Errorf("linked entity not found: %s", key)
        }
    case "feature":
        if s.featureRepo == nil {
            return fmt.Errorf("link validation unavailable: feature repository not configured")
        }
        _, err := s.featureRepo.GetByKey(ctx, key)
        if err != nil {
            return fmt.Errorf("linked entity not found: %s", key)
        }
    case "task":
        if s.taskRepo == nil {
            return fmt.Errorf("link validation unavailable: task repository not configured")
        }
        _, err := s.taskRepo.GetByKey(ctx, key)
        if err != nil {
            return fmt.Errorf("linked entity not found: %s", key)
        }
    }
    return nil
}
```

**Note on short task keys**: The existing codebase supports short task key format `E##-F##-###` (without `T-` prefix). The `isShortTaskKey` helper should check for this pattern as well. Consult the existing key parsing utilities in the codebase.

---

## 4. GetBugService Accessor (`internal/cli/services_global.go`)

```go
// GetBugService returns a BugService instance.
// Creates a new instance each call with the global DB connection and workflow service.
// Panics on DB failure (matching existing CLI entry point pattern).
func GetBugService() *services.BugService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }
    workflowSvc := GetWorkflowService()
    bugRepo := repository.NewBugRepository(db)
    epicRepo := repository.NewEpicRepository(db)
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)

    return services.NewBugService(db, bugRepo, workflowSvc, epicRepo, featureRepo, taskRepo)
}
```

---

## 5. Bug Markdown Template

### 5.1 Template Location

`templates/bug/bug.md.tmpl`

### 5.2 Template Content

```markdown
---
key: {{.Key}}
title: "{{.Title}}"
status: {{.Status}}
severity: {{.Severity}}
{{- if .LinkedEntityKey}}
linked_entity: {{.LinkedEntityKey}}
{{- end}}
---

# {{.Title}}

## Reproduction Steps

1. <!-- Describe the steps to reproduce the issue -->

## Expected Behavior

<!-- Describe what should happen -->

## Actual Behavior

<!-- Describe what actually happens -->

## Environment

<!-- Browser, OS, device, version, or other relevant environment details -->

## Additional Context

<!-- Screenshots, logs, error messages, or any other relevant information -->
```

### 5.3 Template Registration

The template should be loadable via the existing templates package. Check whether templates use `embed.FS` or filesystem loading, and follow the established pattern for registering the new bug template directory.

---

## 6. Context and Note Service Updates

### 6.1 ContextService Changes

In `internal/services/context_service.go`, the `getContextJSON` and `setContextJSON` functions switch on entity type. Add the bug case:

```go
// In getContextJSON:
case models.EntityTypeBug:
    repo := s.bugRepo // BugContextRepo interface
    bug, err := repo.GetByKey(ctx, entityKey)
    if err != nil {
        return nil, err
    }
    return bug.ContextData, nil

// In setContextJSON:
case models.EntityTypeBug:
    repo := s.bugRepo // BugContextRepo interface
    return repo.UpdateContextData(ctx, entityKey, contextJSON)
```

**New interface for ContextService:**
```go
// BugContextRepo provides bug context data access for ContextService.
type BugContextRepo interface {
    GetByKey(ctx context.Context, key string) (*models.Bug, error)
    UpdateContextData(ctx context.Context, key string, contextData *string) error
}
```

**Alternative approach**: Instead of adding a new interface to ContextService, the BugRepository could implement a method `UpdateContextData(ctx, key, contextJSON)` that only updates the `context_data` column. The ContextService's bug case calls this method.

### 6.2 NoteService Changes

In `internal/services/note_service.go`, entity existence validation needs to accept "bug". Add:

```go
case models.EntityTypeBug:
    _, err := s.bugRepo.GetByKey(ctx, entityKey)
    if err != nil {
        return fmt.Errorf("bug not found: %s", entityKey)
    }
```

**New interface for NoteService:**
```go
// BugEntityValidator provides bug existence checks for NoteService.
type BugEntityValidator interface {
    GetByKey(ctx context.Context, key string) (*models.Bug, error)
}
```

### 6.3 Wiring for Context and Note Services

The `GetContextService` and `GetNoteService` functions in `services_global.go` need to be updated to also inject a `BugRepository` so the services can handle the bug entity type.

```go
// In GetContextService:
bugRepo := repository.NewBugRepository(db)
globalContextService = services.NewContextService(epicRepo, featureRepo, taskRepo, bugRepo)

// In GetNoteService:
bugRepo := repository.NewBugRepository(db)
globalNoteService = services.NewNoteService(noteRepo, epicRepo, featureRepo, taskRepo, bugRepo)
```

**Note**: These constructors gain an additional parameter. Since both are initialized via `sync.Once`, the change is isolated to the initialization block. The new parameter can be added as the last argument to maintain backward compatibility.

---

## 7. Bug Workflow Definition (F01 Provides)

For reference, the expected bug workflow from F01:

**Statuses**: `reported`, `triaged`, `in_fix`, `in_verification`, `resolved`, `wont_fix`, `duplicate`

**Status Flow**:
```
reported --> triaged --> in_fix --> in_verification --> resolved
                    \-> wont_fix
                    \-> duplicate
```

**Terminal statuses**: `resolved`, `wont_fix`, `duplicate`

**Default status**: `reported`

The BugService does NOT hardcode these statuses. All status logic flows through `workflowSvc.ForLevel("bug")`.

---

## 8. Error Types

Bug-specific errors follow existing patterns:

```go
// NotFoundError already exists in repository package -- reuse it:
// &repository.NotFoundError{Entity: "bug", Key: key}

// Workflow errors are returned from workflow.Service -- propagate them.

// Validation errors use fmt.Errorf with descriptive messages:
// fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", severity)
// fmt.Errorf("cannot triage bug %s: current status is '%s', must be 'reported'", key, status)
// fmt.Errorf("linked entity not found: %s", key)
```

No new custom error types needed. The existing `NotFoundError` and workflow error types are sufficient.

---

## 9. Testing Strategy

### 9.1 Model Tests (`internal/models/bug_test.go`)

- ValidateBugKey: accepted and rejected formats
- Bug.Validate(): empty title, empty key, invalid key, invalid severity, empty status
- BugSeverity constants: verify all 4 values

### 9.2 Repository Tests (`internal/repository/bug_repository_test.go`)

Real database with cleanup. Test prefix: `TEST-B` keys or use B900-B999 range.

- Create + GetByKey round-trip
- GetByKey case insensitivity
- GetByKey not found returns NotFoundError
- GetNextKey with empty table returns B001
- GetNextKey with existing bugs increments correctly
- GetNextKey after deletion does not reuse keys
- Update mutable fields
- UpdateStatus
- Delete existing and non-existing
- List with no filters
- List with status filter
- List with severity filter
- List with linked entity filter
- List with combined filters
- CountByStatus, CountBySeverity

### 9.3 Service Tests (`internal/services/bug_service_test.go`)

Mocked repositories. Table-driven where applicable.

- CreateBug happy path (default severity, auto-key)
- CreateBug with severity and linked entity
- CreateBug with invalid linked entity (not found)
- CreateBug with unrecognized key format
- CreateBug with invalid severity
- CreateBug with empty title
- GetBug happy path
- GetBug not found
- UpdateBug partial update
- UpdateBug key immutability
- DeleteBug happy path
- DeleteBug not found
- ListBugs with various filter combinations
- ListBugs empty result returns empty slice
- AdvanceBugStatus happy path through workflow
- AdvanceBugStatus from terminal status returns error
- SetBugStatus valid transition
- SetBugStatus invalid transition
- TriageBug happy path (severity + assigned_to + status advance)
- TriageBug from wrong status returns error
- TriageBug rollback on failure

---

*Last Updated: 2026-03-03*
