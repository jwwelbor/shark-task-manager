# Data Models: E21-F07 BaseEntity Struct Embedding

**Feature**: E21-F07
**Author**: Architect
**Date**: 2026-03-20

---

## 1. BaseEntity Struct (New)

### Location

`internal/models/entity.go` (appended to existing file containing Entity interface)

### Definition

```go
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

### Field Specifications

| Field | Go Type | JSON Tag | DB Tag | Nullable | Description |
|-------|---------|----------|--------|----------|-------------|
| ID | `int64` | `"id"` | `"id"` | No | Auto-increment primary key |
| Key | `string` | `"key"` | `"key"` | No | Unique entity identifier (E01, E01-F01, etc.) |
| Title | `string` | `"title"` | `"title"` | No | Human-readable entity title |
| Slug | `*string` | `"slug,omitempty"` | `"slug"` | Yes | URL-safe title slug for dual-key lookup |
| Description | `*string` | `"description,omitempty"` | `"description"` | Yes | Detailed entity description |
| FilePath | `*string` | `"file_path,omitempty"` | `"file_path"` | Yes | Path to entity markdown file |
| ContextData | `*string` | `"context_data,omitempty"` | `"context_data"` | Yes | JSON-encoded context for resume workflow |
| CreatedAt | `time.Time` | `"created_at"` | `"created_at"` | No | Creation timestamp |
| UpdatedAt | `time.Time` | `"updated_at"` | `"updated_at"` | No | Last modification timestamp |

### Field NOT Included

| Field | Reason |
|-------|--------|
| Status | Each entity uses a distinct typed alias (EpicStatus, TaskStatus, etc.) with 2,178 occurrences across 156 files. Including it as `string` would require a massive migration. See architecture doc Section 2 for full rationale. |

---

## 2. Entity Struct Transformations

### 2.1 Epic

**Before**: 12 fields declared inline
**After**: BaseEntity embed + 4 entity-specific fields

```go
type Epic struct {
    BaseEntity                             // 9 fields promoted
    Status        EpicStatus             `json:"status" db:"status"`
    Priority      Priority               `json:"priority" db:"priority"`
    BusinessValue *Priority              `json:"business_value,omitempty" db:"business_value"`
    Metadata      map[string]interface{} `json:"metadata,omitempty" db:"-"`
}
```

**Entity-specific fields**:
| Field | Go Type | Description |
|-------|---------|-------------|
| Status | `EpicStatus` | Typed status enum (draft, active, completed, archived) |
| Priority | `Priority` | Epic priority (high, medium, low) |
| BusinessValue | `*Priority` | Optional business value rating |
| Metadata | `map[string]interface{}` | Derived data, not persisted to DB |

### 2.2 Feature

**Before**: 13 fields declared inline
**After**: BaseEntity embed + 6 entity-specific fields

```go
type Feature struct {
    BaseEntity                             // 9 fields promoted
    EpicID         int64                  `json:"epic_id" db:"epic_id"`
    Status         FeatureStatus          `json:"status" db:"status"`
    StatusOverride bool                   `json:"status_override" db:"status_override"`
    ProgressPct    float64                `json:"progress_pct" db:"progress_pct"`
    ExecutionOrder *int                   `json:"execution_order,omitempty" db:"execution_order"`
    Metadata       map[string]interface{} `json:"metadata,omitempty" db:"-"`
}
```

**Entity-specific fields**:
| Field | Go Type | Description |
|-------|---------|-------------|
| EpicID | `int64` | Foreign key to parent epic |
| Status | `FeatureStatus` | Typed status enum |
| StatusOverride | `bool` | Manual status override flag |
| ProgressPct | `float64` | Calculated progress percentage |
| ExecutionOrder | `*int` | Feature ordering within epic |
| Metadata | `map[string]interface{}` | Derived data, not persisted |

### 2.3 Task

**Before**: 22 fields declared inline (most complex entity)
**After**: BaseEntity embed + 19 entity-specific fields

```go
type Task struct {
    BaseEntity                                          // 9 fields promoted
    FeatureID          int64                            `json:"feature_id" db:"feature_id"`
    Status             TaskStatus                       `json:"status" db:"status"`
    AgentType          *string                          `json:"agent_type,omitempty" db:"agent_type"`
    Priority           int                              `json:"priority" db:"priority"`
    DependsOn          *string                          `json:"depends_on,omitempty" db:"depends_on"`
    AssignedAgent      *string                          `json:"assigned_agent,omitempty" db:"assigned_agent"`
    BlockedReason      *string                          `json:"blocked_reason,omitempty" db:"blocked_reason"`
    ExecutionOrder     *int                             `json:"execution_order,omitempty" db:"execution_order"`
    StartedAt          sql.NullTime                     `json:"started_at,omitempty" db:"started_at"`
    CompletedAt        sql.NullTime                     `json:"completed_at,omitempty" db:"completed_at"`
    BlockedAt          sql.NullTime                     `json:"blocked_at,omitempty" db:"blocked_at"`
    CompletedBy        *string                          `json:"completed_by,omitempty" db:"completed_by"`
    CompletionNotes    *string                          `json:"completion_notes,omitempty" db:"completion_notes"`
    FilesChanged       *string                          `json:"files_changed,omitempty" db:"files_changed"`
    TestsPassed        bool                             `json:"tests_passed" db:"tests_passed"`
    VerificationStatus *VerificationStatus              `json:"verification_status,omitempty" db:"verification_status"`
    TimeSpentMinutes   *int                             `json:"time_spent_minutes,omitempty" db:"time_spent_minutes"`
    Metadata           map[string]interface{}           `json:"metadata,omitempty" db:"-"`
    RejectionCount     int                              `json:"rejection_count" db:"-"`
    LastRejectionAt    *time.Time                       `json:"last_rejection_at,omitempty" db:"-"`
}
```

**Note**: Task.ContextData is currently declared as a standalone field in the current code. After embedding, it is promoted from BaseEntity. The existing `ContextData *string` field declaration on Task must be removed to avoid a compile error (ambiguous field reference).

### 2.4 Bug

**Before**: 12 fields declared inline
**After**: BaseEntity embed + 4 entity-specific fields

```go
type Bug struct {
    BaseEntity                             // 9 fields promoted
    Status           BugStatus            `json:"status" db:"status"`
    Severity         BugSeverity          `json:"severity" db:"severity"`
    LinkedEntityType *string              `json:"linked_entity_type,omitempty" db:"linked_entity_type"`
    LinkedEntityKey  *string              `json:"linked_entity_key,omitempty" db:"linked_entity_key"`
}
```

### 2.5 ChangeCard

**Before**: 16 fields declared inline
**After**: BaseEntity embed + 11 entity-specific fields

```go
type ChangeCard struct {
    BaseEntity                             // 9 fields promoted
    Status         ChangeCardStatus       `json:"status" db:"status"`
    Priority       int                    `json:"priority" db:"priority"`
    RequestedBy    *string                `json:"requested_by,omitempty" db:"requested_by"`
    AssignedTo     *string                `json:"assigned_to,omitempty" db:"assigned_to"`
    EpicID         *int64                 `json:"epic_id,omitempty" db:"epic_id"`
    FeatureID      *int64                 `json:"feature_id,omitempty" db:"feature_id"`
    RelatedTaskID  *int64                 `json:"related_task_id,omitempty" db:"related_task_id"`
    Justification  *string                `json:"justification,omitempty" db:"justification"`
    ImpactAnalysis *string                `json:"impact_analysis,omitempty" db:"impact_analysis"`
    RollbackPlan   *string                `json:"rollback_plan,omitempty" db:"rollback_plan"`
}
```

---

## 3. Per-Entity Method Retention

After embedding, each entity retains exactly 4 methods:

### 3.1 GetEntityType()

Returns the unique EntityType constant for each entity. Cannot be on BaseEntity because each entity type returns a different value.

```go
func (e *Epic) GetEntityType() EntityType       { return EntityTypeEpic }
func (f *Feature) GetEntityType() EntityType    { return EntityTypeFeature }
func (t *Task) GetEntityType() EntityType       { return EntityTypeTask }
func (b *Bug) GetEntityType() EntityType        { return EntityTypeBug }
func (c *ChangeCard) GetEntityType() EntityType { return EntityTypeChange }
```

### 3.2 GetStatus() / SetStatus()

Converts between typed status and string. Cannot be on BaseEntity because Status field is typed per-entity.

```go
// Epic
func (e *Epic) GetStatus() string       { return string(e.Status) }
func (e *Epic) SetStatus(status string) { e.Status = EpicStatus(status) }

// Feature
func (f *Feature) GetStatus() string       { return string(f.Status) }
func (f *Feature) SetStatus(status string) { f.Status = FeatureStatus(status) }

// Task
func (t *Task) GetStatus() string       { return string(t.Status) }
func (t *Task) SetStatus(status string) { t.Status = TaskStatus(status) }

// Bug
func (b *Bug) GetStatus() string       { return string(b.Status) }
func (b *Bug) SetStatus(status string) { b.Status = BugStatus(status) }

// ChangeCard
func (c *ChangeCard) GetStatus() string       { return string(c.Status) }
func (c *ChangeCard) SetStatus(status string) { c.Status = ChangeCardStatus(status) }
```

### 3.3 Validate()

Entity-specific validation logic. Cannot be on BaseEntity because each entity has different required fields and validation rules.

Validate() methods remain unchanged from their current implementations.

---

## 4. Database Schema (No Changes)

This feature does NOT modify the database schema. The database tables remain identical:

| Table | Schema Change | Reason |
|-------|---------------|--------|
| epics | None | Go struct embedding is a code-level concern only |
| features | None | Same |
| tasks | None | Same |
| bugs | None | Same |
| change_cards | None | Same |

No migrations are required. No `CurrentSchemaVersion` bump is needed.

---

## 5. JSON Tag Conflict Analysis

### 5.1 Potential Conflict: Overlapping Tags

When an outer struct has a field with the same JSON tag as an embedded struct field, Go's `encoding/json` applies these rules:

1. If only one field at a given level has a tag, it wins
2. If multiple fields at the same level have the same tag, all are ignored (hidden)
3. Outer struct fields shadow embedded struct fields

### 5.2 Analysis for Each Entity

**Status field**: BaseEntity does NOT have a Status field (Option B), so no conflict exists. Each entity's `Status` field with `json:"status"` tag is the only source.

**All other shared fields**: Are ONLY in BaseEntity after refactoring. No conflict because they are removed from the outer struct.

**Metadata field**: Only on outer struct (`json:"metadata,omitempty" db:"-"`). Not in BaseEntity. No conflict.

### 5.3 Conclusion

Zero JSON tag conflicts. The JSON output is identical before and after embedding.

---

*Last Updated*: 2026-03-20
