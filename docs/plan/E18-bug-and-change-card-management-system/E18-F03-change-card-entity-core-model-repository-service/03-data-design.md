# E18-F03: Change-Card Entity Core -- Data Design

**Feature**: E18-F03
**Date**: 2026-03-03

---

## 1. Entity: ChangeCard

**Description**: A lightweight enhancement or improvement proposal that can optionally link to an epic or feature.

**Table Name**: `change_cards` (created by E18-F01)

### 1.1 Database Schema (Reference -- Defined in F01)

The `change_cards` table is created by E18-F01. This section documents the expected schema that the repository and model depend on.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | Unique identifier |
| key | TEXT | NOT NULL, UNIQUE | Change-card key (C001, C002, ...) |
| title | TEXT | NOT NULL | Change-card title |
| status | TEXT | NOT NULL, DEFAULT 'proposed' | Current workflow status |
| slug | TEXT | | Human-readable key suffix, auto-generated from title |
| linked_entity_type | TEXT | | Type of linked entity: "epic" or "feature" (nullable) |
| linked_entity_key | TEXT | | Key of linked entity, e.g., "E07" or "E07-F01" (nullable) |
| file_path | TEXT | | Path to markdown file relative to project root |
| created_at | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

### 1.2 Expected Indexes (from F01)

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_change_cards_key | (key) | Primary lookup by key |
| idx_change_cards_slug | (slug) | Slug-based lookup |
| idx_change_cards_status | (status) | Status filtering |
| idx_change_cards_linked | (linked_entity_type, linked_entity_key) | Linked entity queries |

### 1.3 Expected Triggers (from F01)

- **update_change_cards_timestamp**: Auto-update `updated_at` on row UPDATE

---

## 2. Go Model: ChangeCard

**File**: `internal/models/change_card.go`

```go
package models

import (
    "fmt"
    "strings"
    "time"
)

// ChangeCardStatus represents the workflow status of a change-card
type ChangeCardStatus string

// ChangeCard represents a lightweight enhancement proposal
type ChangeCard struct {
    ID               int64            `json:"id" db:"id"`
    Key              string           `json:"key" db:"key"`
    Title            string           `json:"title" db:"title"`
    Status           ChangeCardStatus `json:"status" db:"status"`
    Slug             string           `json:"slug,omitempty" db:"slug"`
    LinkedEntityType *string          `json:"linked_entity_type,omitempty" db:"linked_entity_type"`
    LinkedEntityKey  *string          `json:"linked_entity_key,omitempty" db:"linked_entity_key"`
    FilePath         string           `json:"file_path,omitempty" db:"file_path"`
    CreatedAt        time.Time        `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
}

// Validate performs structural validation only.
// Does NOT validate status against workflow -- that is the service layer's responsibility.
func (c *ChangeCard) Validate() error {
    if strings.TrimSpace(c.Title) == "" {
        return fmt.Errorf("change-card title cannot be empty")
    }
    if strings.TrimSpace(string(c.Status)) == "" {
        return fmt.Errorf("change-card status cannot be empty")
    }
    return nil
}
```

**Design Notes**:
- `ChangeCardStatus` is a type alias for `string`, not an enum with constants. Status values come from the workflow engine configuration, not hardcoded constants. This matches the Task/Feature/Epic pattern where status validity is checked at the service layer.
- `LinkedEntityType` and `LinkedEntityKey` are pointer types (`*string`) because they are nullable -- a change-card can exist without a link.
- `Validate()` checks structural integrity only (non-empty title, non-empty status). It does NOT import workflow packages or hardcode status values.

---

## 3. Entity Type Extension

**File**: `internal/models/entity_note.go` (modification)

Add constant and map entry:

```go
const (
    EntityTypeEpic    EntityType = "epic"
    EntityTypeFeature EntityType = "feature"
    EntityTypeTask    EntityType = "task"
    EntityTypeChange  EntityType = "change"    // NEW
)

var ValidEntityTypes = map[EntityType]bool{
    EntityTypeEpic:    true,
    EntityTypeFeature: true,
    EntityTypeTask:    true,
    EntityTypeChange:  true,    // NEW
}
```

---

## 4. Workflow Level Extension

**File**: `internal/workflow/levels.go` (modification)

```go
const (
    LevelEpic    = "epic"
    LevelFeature = "feature"
    LevelTask    = "task"
    LevelChange  = "change"    // NEW
)
```

---

## 5. Key Format

| Aspect | Value |
|--------|-------|
| Format | `C###` (e.g., C001, C042, C999) |
| Regex | `^C\d{3}$` |
| Auto-increment | `MAX(CAST(SUBSTR(key, 2) AS INTEGER)) + 1` |
| Zero-padded | 3 digits |
| Reuse after delete | Never (monotonically increasing) |
| Case sensitivity | Case-insensitive lookup |
| Slug format | `C001-add-dark-mode` (key + slug joined by hyphen) |

---

## 6. Slug Generation

Slugs follow the existing pattern used for epics, features, and tasks:

1. Lowercase the title
2. Replace spaces and underscores with hyphens
3. Remove special characters (keep alphanumeric and hyphens)
4. Collapse multiple hyphens into one
5. Trim leading/trailing hyphens

**Examples**:
- "Add dark mode toggle" -> `add-dark-mode-toggle`
- "API_Design & Testing" -> `api-design-testing`
- "Fix: login redirect!!!" -> `fix-login-redirect`

---

## 7. File Path Convention

Change-card markdown files are stored flat (not nested under epics/features):

```
docs/changes/C001.md
docs/changes/C002.md
docs/changes/C042.md
```

The `docs/changes/` directory is created by the service if it does not exist (using `fileops.NewEntityFileWriter()` which auto-creates parent directories).

---

## 8. Markdown Template

The change-card markdown template generates files with frontmatter and body sections:

```markdown
---
change_card_key: C001
title: Add dark mode toggle
status: proposed
slug: add-dark-mode-toggle
linked_entity_type: epic
linked_entity_key: E07
---

# Add dark mode toggle

## Description

[Describe the proposed change]

## Justification

[Why is this change needed?]

## Linked Entity

- **Type**: epic
- **Key**: E07
```

If no linked entity, the `linked_entity_type` and `linked_entity_key` frontmatter fields are omitted, and the "Linked Entity" section shows "None".

---

## 9. Relationships

| Relationship | From | To | Type | Notes |
|-------------|------|----|------|-------|
| Optional link | ChangeCard | Epic | Reference (not FK) | Via `linked_entity_type` + `linked_entity_key` |
| Optional link | ChangeCard | Feature | Reference (not FK) | Via `linked_entity_type` + `linked_entity_key` |
| Notes | EntityNote | ChangeCard | FK-like (entity_type + entity_id) | Uses existing entity_notes table |

**Note**: Links are stored as soft references (type + key strings), not foreign keys. This is validated at the service layer, not enforced by database constraints. This matches the pattern used by the Bug entity (F02).
