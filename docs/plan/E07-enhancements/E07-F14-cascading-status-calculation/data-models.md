# Data Models: Cascading Status Calculation

## Feature: E07-F14 - Automatic Status Calculation

**Date:** 2026-01-01
**Author:** Architect Agent

---

## 1. Modified Entity: Feature

### Current Schema

```sql
CREATE TABLE IF NOT EXISTS features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    epic_id INTEGER NOT NULL,
    key TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    slug TEXT,
    description TEXT,
    status TEXT NOT NULL,
    progress_pct REAL NOT NULL DEFAULT 0.0 CHECK (progress_pct >= 0.0 AND progress_pct <= 100.0),
    execution_order INTEGER NULL,
    file_path TEXT,
    custom_folder_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (epic_id) REFERENCES epics(id) ON DELETE CASCADE
);
```

### Schema Addition

```sql
-- New column for manual override control
ALTER TABLE features ADD COLUMN status_override BOOLEAN DEFAULT 0;

-- Index for efficient querying of non-overridden features
CREATE INDEX IF NOT EXISTS idx_features_status_override ON features(status_override);
```

### Updated Feature Model

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/models/feature.go`

```go
package models

import "time"

// FeatureStatus represents the status of a feature
type FeatureStatus string

const (
    FeatureStatusDraft     FeatureStatus = "draft"
    FeatureStatusActive    FeatureStatus = "active"
    FeatureStatusCompleted FeatureStatus = "completed"
    FeatureStatusArchived  FeatureStatus = "archived"
)

// Feature represents a mid-level unit within an epic
type Feature struct {
    ID               int64         `json:"id" db:"id"`
    EpicID           int64         `json:"epic_id" db:"epic_id"`
    Key              string        `json:"key" db:"key"`
    Title            string        `json:"title" db:"title"`
    Slug             *string       `json:"slug,omitempty" db:"slug"`
    Description      *string       `json:"description,omitempty" db:"description"`
    Status           FeatureStatus `json:"status" db:"status"`
    StatusOverride   bool          `json:"status_override" db:"status_override"`  // NEW
    ProgressPct      float64       `json:"progress_pct" db:"progress_pct"`
    ExecutionOrder   *int          `json:"execution_order,omitempty" db:"execution_order"`
    FilePath         *string       `json:"file_path,omitempty" db:"file_path"`
    CustomFolderPath *string       `json:"custom_folder_path,omitempty" db:"custom_folder_path"`
    CreatedAt        time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
}

// IsAutoStatus returns true if status is automatically derived from tasks
func (f *Feature) IsAutoStatus() bool {
    return !f.StatusOverride
}

// Validate validates the Feature fields
func (f *Feature) Validate() error {
    if err := ValidateFeatureKey(f.Key); err != nil {
        return err
    }
    if f.Title == "" {
        return ErrEmptyTitle
    }
    if err := ValidateFeatureStatus(string(f.Status)); err != nil {
        return err
    }
    if f.ProgressPct < 0.0 || f.ProgressPct > 100.0 {
        return ErrInvalidProgressPct
    }
    return nil
}
```

### Attribute Details

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | Unique identifier |
| epic_id | INTEGER | NOT NULL, FK | Reference to parent epic |
| key | TEXT | NOT NULL, UNIQUE | Human-readable key (E07-F14) |
| title | TEXT | NOT NULL | Feature title |
| slug | TEXT | | URL-friendly identifier |
| description | TEXT | | Optional description |
| status | TEXT | NOT NULL | Current status |
| **status_override** | BOOLEAN | DEFAULT 0 | **NEW: Manual override flag** |
| progress_pct | REAL | DEFAULT 0.0, CHECK 0-100 | Progress percentage |
| execution_order | INTEGER | | Ordering within epic |
| file_path | TEXT | | Associated file path |
| custom_folder_path | TEXT | | Custom folder for organization |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW | Last update timestamp |

### Business Rules

1. **status_override = false (default):**
   - Status is automatically derived from task statuses
   - Changed by `PropagateTaskChange()` when tasks update

2. **status_override = true:**
   - Status is manually controlled
   - Set when user explicitly sets status via CLI
   - `ApplyFeatureStatus()` skips this feature

3. **Clearing override:**
   - User calls `--auto-status` flag
   - Sets `status_override = false`
   - Immediately recalculates status from tasks

---

## 2. New Model: StatusChangeResult

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/status/models.go`

```go
package status

import "time"

// StatusChangeResult represents the outcome of a status calculation
type StatusChangeResult struct {
    EntityType     string    `json:"entity_type"`     // "feature" or "epic"
    EntityKey      string    `json:"entity_key"`      // e.g., "E07-F14"
    EntityID       int64     `json:"entity_id"`       // Database ID
    PreviousStatus string    `json:"previous_status"` // Status before change
    NewStatus      string    `json:"new_status"`      // Status after change
    WasChanged     bool      `json:"was_changed"`     // true if status actually changed
    WasSkipped     bool      `json:"was_skipped"`     // true if override prevented update
    SkipReason     string    `json:"skip_reason,omitempty"`
    CalculatedAt   time.Time `json:"calculated_at"`
}

// StatusBreakdown holds counts of entities by status
type StatusBreakdown struct {
    Total          int            `json:"total"`
    ByStatus       map[string]int `json:"by_status"`
    CalculatedAt   time.Time      `json:"calculated_at"`
}

// FeatureStatusBreakdown provides task distribution for a feature
type FeatureStatusBreakdown struct {
    FeatureID      int64          `json:"feature_id"`
    FeatureKey     string         `json:"feature_key"`
    StatusBreakdown
    DerivedStatus  string         `json:"derived_status"`
    CurrentStatus  string         `json:"current_status"`
    OverrideActive bool           `json:"override_active"`
}

// EpicStatusBreakdown provides feature distribution for an epic
type EpicStatusBreakdown struct {
    EpicID         int64          `json:"epic_id"`
    EpicKey        string         `json:"epic_key"`
    StatusBreakdown
    DerivedStatus  string         `json:"derived_status"`
    CurrentStatus  string         `json:"current_status"`
}

// RecalculationSummary summarizes a batch recalculation
type RecalculationSummary struct {
    EpicsUpdated    int                   `json:"epics_updated"`
    FeaturesUpdated int                   `json:"features_updated"`
    FeaturesSkipped int                   `json:"features_skipped"`
    Changes         []StatusChangeResult  `json:"changes"`
    StartedAt       time.Time             `json:"started_at"`
    CompletedAt     time.Time             `json:"completed_at"`
    DurationMs      int64                 `json:"duration_ms"`
}
```

---

## 3. Status Derivation Lookup Tables

### Feature Status from Tasks

| Task Status | Weight | Priority |
|-------------|--------|----------|
| in_progress | Active | 1 (highest) |
| ready_for_review | Active | 2 |
| blocked | Active | 3 |
| completed | Count toward completion | - |
| archived | Count toward completion | - |
| todo | Default | 4 (lowest) |

**Derivation Algorithm:**

```
IF all_tasks_are(completed OR archived):
    RETURN completed
ELSE IF any_task_is(in_progress):
    RETURN active
ELSE IF any_task_is(ready_for_review):
    RETURN active
ELSE IF any_task_is(blocked) AND no_task_is(in_progress):
    RETURN active
ELSE IF some_tasks_are(completed) AND some_tasks_are(todo):
    RETURN active
ELSE IF all_tasks_are(todo):
    RETURN draft
ELSE:
    RETURN active  // default fallback
```

### Epic Status from Features

| Feature Status | Weight | Priority |
|----------------|--------|----------|
| active | Active | 1 (highest) |
| completed | Count toward completion | - |
| archived | Count toward completion | - |
| draft | Default | 2 (lowest) |

**Derivation Algorithm:**

```
IF all_features_are(completed OR archived):
    RETURN completed
ELSE IF any_feature_is(active):
    RETURN active
ELSE IF all_features_are(draft):
    RETURN draft
ELSE:
    RETURN active  // default fallback (mixed states)
```

---

## 4. Database Query Specifications

### Query: Get Task Status Counts for Feature

```sql
-- Input: feature_id
-- Returns: status -> count mapping

SELECT
    status,
    COUNT(*) as count
FROM tasks
WHERE feature_id = :feature_id
GROUP BY status;
```

**Example Result:**

| status | count |
|--------|-------|
| todo | 3 |
| in_progress | 2 |
| completed | 5 |

### Query: Get Feature Status Counts for Epic

```sql
-- Input: epic_id
-- Returns: status -> count mapping

SELECT
    status,
    COUNT(*) as count
FROM features
WHERE epic_id = :epic_id
GROUP BY status;
```

### Query: Update Feature Status (Conditional on Override)

```sql
-- Only updates if status_override is false or null
UPDATE features
SET
    status = :new_status,
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = :feature_id
    AND (status_override = 0 OR status_override IS NULL);

-- Returns rows_affected: 1 if updated, 0 if skipped
```

### Query: Get Features Needing Recalculation

```sql
-- Get all features without manual override
SELECT
    f.id,
    f.key,
    f.status,
    f.epic_id
FROM features f
WHERE f.status_override = 0 OR f.status_override IS NULL
ORDER BY f.epic_id, f.id;
```

---

## 5. Index Specifications

### Existing Indexes (Relevant)

```sql
CREATE UNIQUE INDEX idx_features_key ON features(key);
CREATE INDEX idx_features_epic_id ON features(epic_id);
CREATE INDEX idx_features_status ON features(status);
CREATE INDEX idx_tasks_feature_id ON tasks(feature_id);
CREATE INDEX idx_tasks_status ON tasks(status);
```

### New Indexes

```sql
-- For efficient override queries
CREATE INDEX idx_features_status_override ON features(status_override);

-- Composite index for cascade queries
CREATE INDEX idx_features_epic_status ON features(epic_id, status);

-- Composite index for task aggregation
CREATE INDEX idx_tasks_feature_status ON tasks(feature_id, status);
```

---

## 6. Migration Plan

### Migration: Add status_override Column

**File:** `/home/jwwelbor/projects/shark-task-manager/internal/db/db.go`

Add to `runMigrations()`:

```go
// Check if features table has status_override column
err = db.QueryRow(`
    SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'status_override'
`).Scan(&columnExists)
if err != nil {
    return fmt.Errorf("failed to check features schema for status_override: %w", err)
}

if columnExists == 0 {
    if _, err := db.Exec(`ALTER TABLE features ADD COLUMN status_override BOOLEAN DEFAULT 0;`); err != nil {
        return fmt.Errorf("failed to add status_override to features: %w", err)
    }
    if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_features_status_override ON features(status_override);`); err != nil {
        return fmt.Errorf("failed to create features status_override index: %w", err)
    }
}
```

### Migration Rollback (Manual)

```sql
-- SQLite doesn't support DROP COLUMN easily
-- To rollback, would need to recreate table without the column
-- For now, column can remain with default=0 (no-op for existing behavior)
```

### Data Migration

No data migration needed:
- `status_override` defaults to `0` (false)
- All existing features will have automatic status calculation enabled
- Preserves current behavior until user explicitly sets override

---

## 7. Validation Rules

### Feature Status Validation

```go
func ValidateFeatureStatus(status string) error {
    validStatuses := map[string]bool{
        "draft":     true,
        "active":    true,
        "completed": true,
        "archived":  true,
    }
    if !validStatuses[status] {
        return fmt.Errorf("invalid feature status: %s", status)
    }
    return nil
}
```

### Status Override Validation

- `status_override` must be boolean (0 or 1 in SQLite)
- Setting `status_override = true` requires explicit `status` value
- Clearing override (`--auto-status`) triggers immediate recalculation

---

## 8. Entity Relationship Updates

### Current Relationships

```
Epic (1) --< (N) Feature (1) --< (N) Task
```

### Status Flow (New)

```
Task Status Change
       |
       v
Feature.RecalculateStatus() <-- respects status_override
       |
       v
Epic.RecalculateStatus()
```

### Cascade Direction

- **Upward only:** Task -> Feature -> Epic
- **Never downward:** Changing epic status does NOT change feature/task statuses
- **Respects boundaries:** Override on feature stops propagation to that feature

---

*Document Version: 1.0*
*Last Updated: 2026-01-01*
