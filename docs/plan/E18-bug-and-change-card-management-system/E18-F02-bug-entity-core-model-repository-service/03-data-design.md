# E18-F02: Bug Entity Core -- Data Design

**Feature**: Bug Entity Core (Model, Repository, Service)
**Date**: 2026-03-03

---

## 1. Bug Model (`internal/models/bug.go`)

### 1.1 Type Definitions

```go
package models

// BugStatus represents the workflow status of a bug.
// Valid values are determined by the workflow configuration, not hardcoded.
type BugStatus string

// BugSeverity represents the severity level of a bug.
type BugSeverity string

const (
    BugSeverityCritical BugSeverity = "critical"
    BugSeverityHigh     BugSeverity = "high"
    BugSeverityMedium   BugSeverity = "medium"
    BugSeverityLow      BugSeverity = "low"
)

// ValidBugSeverities is the set of valid severity values.
var ValidBugSeverities = map[BugSeverity]bool{
    BugSeverityCritical: true,
    BugSeverityHigh:     true,
    BugSeverityMedium:   true,
    BugSeverityLow:      true,
}
```

### 1.2 Bug Struct

```go
// Bug represents a bug report entity.
type Bug struct {
    ID               int64       `json:"id" db:"id"`
    Key              string      `json:"key" db:"key"`                               // Format: B###
    Title            string      `json:"title" db:"title"`
    Slug             *string     `json:"slug,omitempty" db:"slug"`
    Description      *string     `json:"description,omitempty" db:"description"`
    Status           BugStatus   `json:"status" db:"status"`
    Severity         BugSeverity `json:"severity" db:"severity"`
    LinkedEntityType *string     `json:"linked_entity_type,omitempty" db:"linked_entity_type"` // "epic", "feature", "task"
    LinkedEntityKey  *string     `json:"linked_entity_key,omitempty" db:"linked_entity_key"`
    ContextData      *string     `json:"context_data,omitempty" db:"context_data"`             // JSON
    FilePath         *string     `json:"file_path,omitempty" db:"file_path"`
    CreatedAt        time.Time   `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
}
```

**Design Notes:**
- `LinkedEntityType` and `LinkedEntityKey` are pointer types (nullable) because links are optional.
- `ContextData` stores arbitrary JSON (e.g., `{"assigned_to": "alice", "environment": "Safari 17.2"}`).
- `Slug` follows the same pattern as Task/Feature/Epic slugs -- auto-generated from title.
- `Description` is a pointer (optional field, may be nil).
- No `EpicID`/`FeatureID` foreign keys -- Bug is a standalone entity.

### 1.3 Struct Tag Mapping to Database

| Go Field | DB Column | DB Type | Nullable | Notes |
|----------|-----------|---------|----------|-------|
| ID | id | INTEGER | NO | PRIMARY KEY AUTOINCREMENT |
| Key | key | TEXT | NO | UNIQUE, format B### |
| Title | title | TEXT | NO | |
| Slug | slug | TEXT | YES | Auto-generated from title |
| Description | description | TEXT | YES | |
| Status | status | TEXT | NO | DEFAULT 'reported' |
| Severity | severity | TEXT | NO | DEFAULT 'medium' |
| LinkedEntityType | linked_entity_type | TEXT | YES | CHECK in epic/feature/task |
| LinkedEntityKey | linked_entity_key | TEXT | YES | |
| ContextData | context_data | TEXT | YES | JSON string |
| FilePath | file_path | TEXT | YES | |
| CreatedAt | created_at | DATETIME | NO | DEFAULT CURRENT_TIMESTAMP |
| UpdatedAt | updated_at | DATETIME | NO | DEFAULT CURRENT_TIMESTAMP |

### 1.4 Validation Method

```go
// Validate performs structural validation on the Bug model.
// It does NOT check workflow status validity (that is the service layer's job).
func (b *Bug) Validate() error {
    if err := ValidateBugKey(b.Key); err != nil {
        return err
    }
    if strings.TrimSpace(b.Title) == "" {
        return ErrEmptyTitle
    }
    if strings.TrimSpace(string(b.Status)) == "" {
        return errors.New("bug status cannot be empty")
    }
    if !ValidBugSeverities[b.Severity] {
        return fmt.Errorf("invalid severity %q: must be one of critical, high, medium, low", b.Severity)
    }
    return nil
}
```

### 1.5 Key Validation Function

```go
var bugKeyPattern = regexp.MustCompile(`^B\d{3}$`)

// ValidateBugKey validates the bug key format (B### where ### is 3 digits).
func ValidateBugKey(key string) error {
    if key == "" {
        return ErrEmptyKey
    }
    if !bugKeyPattern.MatchString(key) {
        return fmt.Errorf("invalid bug key format %q: must match B### (e.g., B001, B042)", key)
    }
    return nil
}
```

**Accepted**: `B001`, `B042`, `B999`
**Rejected**: `B0001` (4 digits), `B01` (2 digits), `b001` (lowercase), `bug001`, empty string

---

## 2. Entity Type Registration

### 2.1 EntityType Constant (`internal/models/entity_note.go`)

Add to existing constants:

```go
const (
    EntityTypeEpic    EntityType = "epic"
    EntityTypeFeature EntityType = "feature"
    EntityTypeTask    EntityType = "task"
    EntityTypeBug     EntityType = "bug"       // NEW
)

var ValidEntityTypes = map[EntityType]bool{
    EntityTypeEpic:    true,
    EntityTypeFeature: true,
    EntityTypeTask:    true,
    EntityTypeBug:     true,  // NEW
}
```

---

## 3. Bug Repository Interface (`internal/services/bug_service.go`)

The BugRepository interface is defined in the services package (consumer-side pattern) and implemented by `*repository.BugRepository`.

```go
// BugRepository defines the data access interface needed by BugService.
// Implemented by *repository.BugRepository.
type BugRepository interface {
    // CRUD
    Create(ctx context.Context, bug *models.Bug) error
    GetByKey(ctx context.Context, key string) (*models.Bug, error)
    GetByID(ctx context.Context, id int64) (*models.Bug, error)
    Update(ctx context.Context, bug *models.Bug) error
    Delete(ctx context.Context, key string) error
    UpdateStatus(ctx context.Context, id int64, status models.BugStatus) error

    // Key generation
    GetNextKey(ctx context.Context) (string, error)

    // Query
    List(ctx context.Context, filters BugListFilters) ([]*models.Bug, error)

    // Analytics
    CountByStatus(ctx context.Context) (map[string]int, error)
    CountBySeverity(ctx context.Context) (map[string]int, error)
}

// BugListFilters contains the filter parameters for List queries.
// Used by the repository to build WHERE clauses.
type BugListFilters struct {
    Status          string
    Severity        string
    LinkedEntityKey string
}
```

### Link Validation Interfaces

Minimal interfaces for the repositories needed only for link validation:

```go
// LinkValidatorEpicRepo provides epic existence checks for link validation.
type LinkValidatorEpicRepo interface {
    GetByKey(ctx context.Context, key string) (*models.Epic, error)
}

// LinkValidatorFeatureRepo provides feature existence checks for link validation.
type LinkValidatorFeatureRepo interface {
    GetByKey(ctx context.Context, key string) (*models.Feature, error)
}

// LinkValidatorTaskRepo provides task existence checks for link validation.
type LinkValidatorTaskRepo interface {
    GetByKey(ctx context.Context, key string) (*models.Task, error)
}
```

---

## 4. Bug Repository Implementation (`internal/repository/bug_repository.go`)

### 4.1 Constructor

```go
type BugRepository struct {
    db *DB
}

func NewBugRepository(db *DB) *BugRepository {
    return &BugRepository{db: db}
}
```

### 4.2 Key Methods

**GetNextKey**: Generates the next B### key.

```sql
SELECT COALESCE(MAX(CAST(SUBSTR(key, 2) AS INTEGER)), 0) + 1 FROM bugs
```

Returns `fmt.Sprintf("B%03d", nextNum)`. If no bugs exist, returns `"B001"`.

**GetByKey**: Case-insensitive lookup.

```sql
SELECT id, key, title, slug, description, status, severity,
       linked_entity_type, linked_entity_key, context_data,
       file_path, created_at, updated_at
FROM bugs
WHERE UPPER(key) = UPPER(?)
```

Returns `NotFoundError{Entity: "bug", Key: key}` if not found.

**Create**: Inserts a bug and sets `bug.ID` from `LastInsertId()`.

**Update**: Updates mutable fields (title, description, severity, linked_entity_type, linked_entity_key, context_data, file_path). Does NOT update key or status (status has its own method).

**UpdateStatus**: Updates only the status and updated_at fields.

```sql
UPDATE bugs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
```

**Delete**: Deletes by key. Returns `NotFoundError` if the key does not exist (checks affected rows).

**List**: Builds dynamic WHERE clause from filters.

```go
func (r *BugRepository) List(ctx context.Context, filters BugListFilters) ([]*models.Bug, error) {
    query := `SELECT ... FROM bugs WHERE 1=1`
    args := []interface{}{}

    if filters.Status != "" {
        query += " AND status = ?"
        args = append(args, filters.Status)
    }
    if filters.Severity != "" {
        query += " AND severity = ?"
        args = append(args, filters.Severity)
    }
    if filters.LinkedEntityKey != "" {
        query += " AND linked_entity_key = ?"
        args = append(args, filters.LinkedEntityKey)
    }

    query += " ORDER BY created_at DESC"
    // ... execute and scan
}
```

**CountByStatus / CountBySeverity**: Aggregate queries.

```sql
SELECT status, COUNT(*) FROM bugs GROUP BY status
SELECT severity, COUNT(*) FROM bugs GROUP BY severity
```

### 4.3 Column Scan Order

All SELECT queries must scan columns in this order to match the Bug struct:

```go
row.Scan(
    &bug.ID,
    &bug.Key,
    &bug.Title,
    &bug.Slug,
    &bug.Description,
    &bug.Status,
    &bug.Severity,
    &bug.LinkedEntityType,
    &bug.LinkedEntityKey,
    &bug.ContextData,
    &bug.FilePath,
    &bug.CreatedAt,
    &bug.UpdatedAt,
)
```

Extract a private `scanBug(scanner) (*models.Bug, error)` helper to avoid repetition.

---

## 5. Slug Generation

Bug slugs follow the same algorithm as existing entities:

1. Lowercase the title
2. Replace spaces and underscores with hyphens
3. Remove characters that are not alphanumeric or hyphens
4. Collapse multiple hyphens into one
5. Trim leading/trailing hyphens

Example: `"Button misaligned on mobile"` becomes `"button-misaligned-on-mobile"`

Reuse the existing `GenerateSlug()` function from wherever it exists in the codebase (check `internal/repository/` or `internal/models/` for the slug generation utility).

---

*Last Updated: 2026-03-03*
