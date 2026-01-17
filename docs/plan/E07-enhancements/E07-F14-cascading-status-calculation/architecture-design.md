# Technical Architecture Design: Cascading Status Calculation (E07-F14)

**Date:** 2026-01-16
**Author:** Architect Agent
**Feature:** E07-F14 - Cascading Status Calculation
**Status:** Design Proposal

---

## Executive Summary

This document proposes the technical architecture for implementing automatic feature/epic status calculation based on child entity states. The design balances **simplicity**, **performance**, and **flexibility** by using:

1. **Hybrid computed status model** - status calculated on-demand with optional override
2. **Application-layer cascade logic** - controlled, testable state propagation
3. **Efficient SQL aggregation** - single-query status derivation
4. **Minimal schema changes** - single boolean column addition per table

---

## 1. Data Model Changes

### 1.1 Database Schema Additions

**Features Table:**
```sql
-- Add status override flag
ALTER TABLE features ADD COLUMN status_override BOOLEAN DEFAULT 0;

-- Index for efficient override filtering
CREATE INDEX IF NOT EXISTS idx_features_status_override ON features(status_override);

-- Composite index for cascade queries (epic -> features)
CREATE INDEX IF NOT EXISTS idx_features_epic_status ON features(epic_id, status);
```

**Epics Table:**
```sql
-- Add status override flag
ALTER TABLE epics ADD COLUMN status_override BOOLEAN DEFAULT 0;

-- Index for efficient override filtering
CREATE INDEX IF NOT EXISTS idx_epics_status_override ON epics(status_override);
```

**Tasks Table (existing indexes sufficient):**
```sql
-- Already exists: idx_tasks_feature_id ON tasks(feature_id)
-- Already exists: idx_tasks_status ON tasks(status)

-- OPTIONAL optimization: Composite index for aggregation queries
CREATE INDEX IF NOT EXISTS idx_tasks_feature_status ON tasks(feature_id, status);
```

### 1.2 Model Updates

**Feature Model** (`internal/models/feature.go`):
```go
type Feature struct {
    // ... existing fields ...
    Status         FeatureStatus `json:"status" db:"status"`
    StatusOverride bool          `json:"status_override" db:"status_override"`
    // ... rest of fields ...
}

// IsAutoStatus returns true if status is automatically calculated
func (f *Feature) IsAutoStatus() bool {
    return !f.StatusOverride
}

// StatusSource returns "calculated" or "manual" for display
func (f *Feature) StatusSource() string {
    if f.StatusOverride {
        return "manual"
    }
    return "calculated"
}
```

**Epic Model** (`internal/models/epic.go`):
```go
type Epic struct {
    // ... existing fields ...
    Status         EpicStatus `json:"status" db:"status"`
    StatusOverride bool       `json:"status_override" db:"status_override"`
    // ... rest of fields ...
}

// IsAutoStatus returns true if status is automatically calculated
func (e *Epic) IsAutoStatus() bool {
    return !e.StatusOverride
}

// StatusSource returns "calculated" or "manual" for display
func (e *Epic) StatusSource() string {
    if e.StatusOverride {
        return "manual"
    }
    return "calculated"
}
```

### 1.3 Migration Strategy

**Location:** `internal/db/db.go` in `runMigrations()` function

```go
func runMigrations(db *sql.DB) error {
    // ... existing migrations ...

    // Migration: Add status_override to features
    var columnExists int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM pragma_table_info('features')
        WHERE name = 'status_override'
    `).Scan(&columnExists)
    if err != nil {
        return fmt.Errorf("failed to check features schema: %w", err)
    }

    if columnExists == 0 {
        _, err = db.Exec(`ALTER TABLE features ADD COLUMN status_override BOOLEAN DEFAULT 0;`)
        if err != nil {
            return fmt.Errorf("failed to add status_override to features: %w", err)
        }

        _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_features_status_override ON features(status_override);`)
        if err != nil {
            return fmt.Errorf("failed to create features status_override index: %w", err)
        }
    }

    // Migration: Add status_override to epics
    err = db.QueryRow(`
        SELECT COUNT(*) FROM pragma_table_info('epics')
        WHERE name = 'status_override'
    `).Scan(&columnExists)
    if err != nil {
        return fmt.Errorf("failed to check epics schema: %w", err)
    }

    if columnExists == 0 {
        _, err = db.Exec(`ALTER TABLE epics ADD COLUMN status_override BOOLEAN DEFAULT 0;`)
        if err != nil {
            return fmt.Errorf("failed to add status_override to epics: %w", err)
        }

        _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_epics_status_override ON epics(status_override);`)
        if err != nil {
            return fmt.Errorf("failed to create epics status_override index: %w", err)
        }
    }

    // Optional: Composite indexes for performance
    _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_features_epic_status ON features(epic_id, status);`)
    if err != nil {
        return fmt.Errorf("failed to create features epic_status index: %w", err)
    }

    _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_feature_status ON tasks(feature_id, status);`)
    if err != nil {
        return fmt.Errorf("failed to create tasks feature_status index: %w", err)
    }

    return nil
}
```

**Migration Characteristics:**
- **Idempotent**: Safe to run multiple times
- **Non-destructive**: Adds columns with defaults, no data loss
- **Backward compatible**: Defaults to status_override=false (auto-calculation)
- **Performance**: < 100ms for databases with 1000+ entities

---

## 2. Status Calculation Logic

### 2.1 Core Algorithm Package

**New Package:** `internal/status/`

**Structure:**
```
internal/status/
├── derivation.go       # Pure calculation functions
├── derivation_test.go  # Comprehensive unit tests
└── models.go           # Result types and constants
```

**Key Design Principles:**
- **Pure functions**: No side effects, fully testable
- **No database dependencies**: Takes counts as input
- **Clear separation**: Algorithm isolated from infrastructure

### 2.2 Status Derivation Functions

**File:** `internal/status/derivation.go`

```go
package status

import "github.com/jwwelbor/shark-task-manager/internal/models"

// TaskStatusCounts holds task status distribution
type TaskStatusCounts struct {
    Todo            int
    InProgress      int
    ReadyForReview  int
    Blocked         int
    Completed       int
    Archived        int
}

// FeatureStatusCounts holds feature status distribution
type FeatureStatusCounts struct {
    Draft       int
    Active      int
    Blocked     int
    Completed   int
    Archived    int
}

// DeriveFeatureStatus calculates feature status from task counts
func DeriveFeatureStatus(counts TaskStatusCounts) models.FeatureStatus {
    total := counts.Todo + counts.InProgress + counts.ReadyForReview +
             counts.Blocked + counts.Completed + counts.Archived

    // No tasks = draft
    if total == 0 {
        return models.FeatureStatusDraft
    }

    // All completed/archived = completed
    completedCount := counts.Completed + counts.Archived
    if completedCount == total {
        return models.FeatureStatusCompleted
    }

    // Any active work = active
    activeCount := counts.InProgress + counts.ReadyForReview + counts.Blocked
    if activeCount > 0 {
        return models.FeatureStatusActive
    }

    // Mix of todo + completed (without active) = active
    // Rationale: Work has started but not all tasks begun
    if completedCount > 0 && counts.Todo > 0 {
        return models.FeatureStatusActive
    }

    // All todo = draft
    return models.FeatureStatusDraft
}

// DeriveEpicStatus calculates epic status from feature counts
func DeriveEpicStatus(counts FeatureStatusCounts) models.EpicStatus {
    total := counts.Draft + counts.Active + counts.Blocked +
             counts.Completed + counts.Archived

    // No features = draft
    if total == 0 {
        return models.EpicStatusDraft
    }

    // All completed/archived = completed
    completedCount := counts.Completed + counts.Archived
    if completedCount == total {
        return models.EpicStatusCompleted
    }

    // Any active/blocked = active
    activeCount := counts.Active + counts.Blocked
    if activeCount > 0 {
        return models.EpicStatusActive
    }

    // All draft = draft
    return models.EpicStatusDraft
}

// ParseTaskStatusCounts converts SQL GROUP BY results to TaskStatusCounts
func ParseTaskStatusCounts(rows map[string]int) TaskStatusCounts {
    return TaskStatusCounts{
        Todo:           rows["todo"],
        InProgress:     rows["in_progress"],
        ReadyForReview: rows["ready_for_review"],
        Blocked:        rows["blocked"],
        Completed:      rows["completed"],
        Archived:       rows["archived"],
    }
}

// ParseFeatureStatusCounts converts SQL GROUP BY results to FeatureStatusCounts
func ParseFeatureStatusCounts(rows map[string]int) FeatureStatusCounts {
    return FeatureStatusCounts{
        Draft:     rows["draft"],
        Active:    rows["active"],
        Blocked:   rows["blocked"],
        Completed: rows["completed"],
        Archived:  rows["archived"],
    }
}
```

### 2.3 Result Models

**File:** `internal/status/models.go`

```go
package status

import "time"

// StatusChangeResult represents the outcome of a status recalculation
type StatusChangeResult struct {
    EntityType     string    `json:"entity_type"`      // "feature" or "epic"
    EntityKey      string    `json:"entity_key"`       // e.g., "E07-F14"
    EntityID       int64     `json:"entity_id"`
    PreviousStatus string    `json:"previous_status"`
    NewStatus      string    `json:"new_status"`
    Changed        bool      `json:"changed"`          // true if status actually changed
    Skipped        bool      `json:"skipped"`          // true if override prevented update
    SkipReason     string    `json:"skip_reason,omitempty"`
    CalculatedAt   time.Time `json:"calculated_at"`
}

// CascadeResult represents the outcome of a cascade operation
type CascadeResult struct {
    FeatureChange *StatusChangeResult `json:"feature_change,omitempty"`
    EpicChange    *StatusChangeResult `json:"epic_change,omitempty"`
}
```

---

## 3. Repository Layer Changes

### 3.1 Feature Repository Additions

**File:** `internal/repository/feature_repository.go`

```go
// GetTaskStatusBreakdown returns task counts grouped by status
// Returns map[status]count for efficient status calculation
func (r *FeatureRepository) GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[string]int, error) {
    query := `
        SELECT status, COUNT(*) as count
        FROM tasks
        WHERE feature_id = ?
        GROUP BY status
    `

    rows, err := r.db.QueryContext(ctx, query, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get task status breakdown: %w", err)
    }
    defer rows.Close()

    breakdown := make(map[string]int)
    for rows.Next() {
        var status string
        var count int
        if err := rows.Scan(&status, &count); err != nil {
            return nil, fmt.Errorf("failed to scan row: %w", err)
        }
        breakdown[status] = count
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rows: %w", err)
    }

    return breakdown, nil
}

// CalculateFeatureStatus computes the derived status from tasks
// Does NOT update database - use RecalculateAndUpdateStatus for that
func (r *FeatureRepository) CalculateFeatureStatus(ctx context.Context, featureID int64) (models.FeatureStatus, error) {
    breakdown, err := r.GetTaskStatusBreakdown(ctx, featureID)
    if err != nil {
        return "", err
    }

    counts := status.ParseTaskStatusCounts(breakdown)
    return status.DeriveFeatureStatus(counts), nil
}

// RecalculateAndUpdateStatus recalculates feature status and updates database
// Respects status_override flag - returns skipped=true if override is active
// Also triggers epic status recalculation cascade
func (r *FeatureRepository) RecalculateAndUpdateStatus(ctx context.Context, featureID int64) (*status.CascadeResult, error) {
    // Get current feature
    feature, err := r.GetByID(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get feature: %w", err)
    }

    // Calculate new status
    newStatus, err := r.CalculateFeatureStatus(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to calculate status: %w", err)
    }

    result := &status.CascadeResult{
        FeatureChange: &status.StatusChangeResult{
            EntityType:     "feature",
            EntityKey:      feature.Key,
            EntityID:       feature.ID,
            PreviousStatus: string(feature.Status),
            NewStatus:      string(newStatus),
            Changed:        false,
            Skipped:        false,
            CalculatedAt:   time.Now(),
        },
    }

    // Check if override is active
    if feature.StatusOverride {
        result.FeatureChange.Skipped = true
        result.FeatureChange.SkipReason = "manual override active"
        // Still cascade to epic even if feature skipped
        epicRepo := NewEpicRepository(r.db)
        epicResult, err := epicRepo.RecalculateAndUpdateStatus(ctx, feature.EpicID)
        if err != nil {
            return result, fmt.Errorf("failed to cascade to epic: %w", err)
        }
        result.EpicChange = epicResult.EpicChange
        return result, nil
    }

    // Check if status actually changed
    if feature.Status == newStatus {
        // No change, but still cascade (other features might have changed)
        epicRepo := NewEpicRepository(r.db)
        epicResult, err := epicRepo.RecalculateAndUpdateStatus(ctx, feature.EpicID)
        if err != nil {
            return result, fmt.Errorf("failed to cascade to epic: %w", err)
        }
        result.EpicChange = epicResult.EpicChange
        return result, nil
    }

    // Update status in database
    updateQuery := `
        UPDATE features
        SET status = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err = r.db.ExecContext(ctx, updateQuery, string(newStatus), featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to update feature status: %w", err)
    }

    result.FeatureChange.Changed = true

    // Cascade to epic
    epicRepo := NewEpicRepository(r.db)
    epicResult, err := epicRepo.RecalculateAndUpdateStatus(ctx, feature.EpicID)
    if err != nil {
        return result, fmt.Errorf("failed to cascade to epic: %w", err)
    }
    result.EpicChange = epicResult.EpicChange

    return result, nil
}

// SetStatusManual sets feature status with manual override
func (r *FeatureRepository) SetStatusManual(ctx context.Context, featureID int64, newStatus models.FeatureStatus) error {
    query := `
        UPDATE features
        SET status = ?, status_override = 1, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err := r.db.ExecContext(ctx, query, string(newStatus), featureID)
    if err != nil {
        return fmt.Errorf("failed to set manual status: %w", err)
    }
    return nil
}

// ClearStatusOverride clears manual override and recalculates status
func (r *FeatureRepository) ClearStatusOverride(ctx context.Context, featureID int64) (*status.CascadeResult, error) {
    // Clear override flag first
    query := `
        UPDATE features
        SET status_override = 0, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err := r.db.ExecContext(ctx, query, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to clear override: %w", err)
    }

    // Recalculate and update status
    return r.RecalculateAndUpdateStatus(ctx, featureID)
}
```

### 3.2 Epic Repository Additions

**File:** `internal/repository/epic_repository.go`

```go
// GetFeatureStatusBreakdown returns feature counts grouped by status
func (r *EpicRepository) GetFeatureStatusBreakdown(ctx context.Context, epicID int64) (map[string]int, error) {
    query := `
        SELECT status, COUNT(*) as count
        FROM features
        WHERE epic_id = ?
        GROUP BY status
    `

    rows, err := r.db.QueryContext(ctx, query, epicID)
    if err != nil {
        return nil, fmt.Errorf("failed to get feature status breakdown: %w", err)
    }
    defer rows.Close()

    breakdown := make(map[string]int)
    for rows.Next() {
        var status string
        var count int
        if err := rows.Scan(&status, &count); err != nil {
            return nil, fmt.Errorf("failed to scan row: %w", err)
        }
        breakdown[status] = count
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rows: %w", err)
    }

    return breakdown, nil
}

// CalculateEpicStatus computes the derived status from features
func (r *EpicRepository) CalculateEpicStatus(ctx context.Context, epicID int64) (models.EpicStatus, error) {
    breakdown, err := r.GetFeatureStatusBreakdown(ctx, epicID)
    if err != nil {
        return "", err
    }

    counts := status.ParseFeatureStatusCounts(breakdown)
    return status.DeriveEpicStatus(counts), nil
}

// RecalculateAndUpdateStatus recalculates epic status and updates database
// Respects status_override flag - returns skipped=true if override is active
func (r *EpicRepository) RecalculateAndUpdateStatus(ctx context.Context, epicID int64) (*status.CascadeResult, error) {
    // Get current epic
    epic, err := r.GetByID(ctx, epicID)
    if err != nil {
        return nil, fmt.Errorf("failed to get epic: %w", err)
    }

    // Calculate new status
    newStatus, err := r.CalculateEpicStatus(ctx, epicID)
    if err != nil {
        return nil, fmt.Errorf("failed to calculate status: %w", err)
    }

    result := &status.CascadeResult{
        EpicChange: &status.StatusChangeResult{
            EntityType:     "epic",
            EntityKey:      epic.Key,
            EntityID:       epic.ID,
            PreviousStatus: string(epic.Status),
            NewStatus:      string(newStatus),
            Changed:        false,
            Skipped:        false,
            CalculatedAt:   time.Now(),
        },
    }

    // Check if override is active
    if epic.StatusOverride {
        result.EpicChange.Skipped = true
        result.EpicChange.SkipReason = "manual override active"
        return result, nil
    }

    // Check if status actually changed
    if epic.Status == newStatus {
        return result, nil
    }

    // Update status in database
    updateQuery := `
        UPDATE epics
        SET status = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err = r.db.ExecContext(ctx, updateQuery, string(newStatus), epicID)
    if err != nil {
        return nil, fmt.Errorf("failed to update epic status: %w", err)
    }

    result.EpicChange.Changed = true
    return result, nil
}

// SetStatusManual sets epic status with manual override
func (r *EpicRepository) SetStatusManual(ctx context.Context, epicID int64, newStatus models.EpicStatus) error {
    query := `
        UPDATE epics
        SET status = ?, status_override = 1, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err := r.db.ExecContext(ctx, query, string(newStatus), epicID)
    if err != nil {
        return fmt.Errorf("failed to set manual status: %w", err)
    }
    return nil
}

// ClearStatusOverride clears manual override and recalculates status
func (r *EpicRepository) ClearStatusOverride(ctx context.Context, epicID int64) (*status.CascadeResult, error) {
    // Clear override flag first
    query := `
        UPDATE epics
        SET status_override = 0, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err := r.db.ExecContext(ctx, query, epicID)
    if err != nil {
        return nil, fmt.Errorf("failed to clear override: %w", err)
    }

    // Recalculate and update status
    return r.RecalculateAndUpdateStatus(ctx, epicID)
}
```

### 3.3 Task Repository Integration

**File:** `internal/repository/task_repository.go`

Add cascade trigger to status update methods:

```go
// UpdateStatus updates task status with cascade to feature/epic
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskKey string, newStatus models.TaskStatus) (*status.CascadeResult, error) {
    // Get task
    task, err := r.GetByKey(ctx, taskKey)
    if err != nil {
        return nil, err
    }

    // Update task status (existing logic)
    updateQuery := `
        UPDATE tasks
        SET status = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err = r.db.ExecContext(ctx, updateQuery, string(newStatus), task.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to update task status: %w", err)
    }

    // Cascade to feature (which cascades to epic)
    featureRepo := NewFeatureRepository(r.db)
    result, err := featureRepo.RecalculateAndUpdateStatus(ctx, task.FeatureID)
    if err != nil {
        return nil, fmt.Errorf("failed to cascade status change: %w", err)
    }

    return result, nil
}
```

---

## 4. CLI Changes

### 4.1 Feature Update Command

**File:** `internal/cli/commands/feature.go`

Add status update support:

```go
var featureUpdateCmd = &cobra.Command{
    Use:   "update <feature-key>",
    Short: "Update feature properties",
    Args:  cobra.ExactArgs(1),
    RunE:  runFeatureUpdate,
}

var (
    featureUpdateStatus string
)

func init() {
    featureUpdateCmd.Flags().StringVar(&featureUpdateStatus, "status", "",
        "Set feature status (draft, active, completed, archived, or 'auto' to enable automatic calculation)")
    featureCmd.AddCommand(featureUpdateCmd)
}

func runFeatureUpdate(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    featureKey := args[0]

    repoDb, err := cli.GetDB(ctx)
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }

    featureRepo := repository.NewFeatureRepository(repoDb)

    // Get feature
    feature, err := featureRepo.GetByKey(ctx, featureKey)
    if err != nil {
        return fmt.Errorf("failed to get feature: %w", err)
    }

    // Handle status update
    if featureUpdateStatus != "" {
        if featureUpdateStatus == "auto" {
            // Clear override and recalculate
            result, err := featureRepo.ClearStatusOverride(ctx, feature.ID)
            if err != nil {
                return fmt.Errorf("failed to enable auto status: %w", err)
            }

            if cli.GlobalConfig.JSON {
                return cli.OutputJSON(result)
            }

            cli.Success(fmt.Sprintf("Feature %s: status override cleared", feature.Key))
            if result.FeatureChange.Changed {
                cli.Info(fmt.Sprintf("  Status changed: %s → %s",
                    result.FeatureChange.PreviousStatus,
                    result.FeatureChange.NewStatus))
            }
            if result.EpicChange != nil && result.EpicChange.Changed {
                cli.Info(fmt.Sprintf("  Epic %s status changed: %s → %s",
                    result.EpicChange.EntityKey,
                    result.EpicChange.PreviousStatus,
                    result.EpicChange.NewStatus))
            }
            return nil
        }

        // Set manual status
        newStatus := models.FeatureStatus(featureUpdateStatus)
        if err := newStatus.Validate(); err != nil {
            return fmt.Errorf("invalid status: %w", err)
        }

        err = featureRepo.SetStatusManual(ctx, feature.ID, newStatus)
        if err != nil {
            return fmt.Errorf("failed to set status: %w", err)
        }

        // Trigger epic recalculation
        epicRepo := repository.NewEpicRepository(repoDb)
        epicResult, err := epicRepo.RecalculateAndUpdateStatus(ctx, feature.EpicID)
        if err != nil {
            return fmt.Errorf("failed to cascade to epic: %w", err)
        }

        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(map[string]interface{}{
                "feature_change": map[string]interface{}{
                    "key": feature.Key,
                    "previous_status": string(feature.Status),
                    "new_status": featureUpdateStatus,
                    "override": true,
                },
                "epic_change": epicResult.EpicChange,
            })
        }

        cli.Success(fmt.Sprintf("Feature %s status set to %s (manual override)", feature.Key, featureUpdateStatus))
        if epicResult.EpicChange != nil && epicResult.EpicChange.Changed {
            cli.Info(fmt.Sprintf("  Epic %s status changed: %s → %s",
                epicResult.EpicChange.EntityKey,
                epicResult.EpicChange.PreviousStatus,
                epicResult.EpicChange.NewStatus))
        }
        return nil
    }

    return fmt.Errorf("no update options specified")
}
```

### 4.2 Feature Get Command Enhancement

Update to show status source:

```go
func displayFeature(feature *models.Feature) {
    // ... existing code ...

    // Show status with source indicator
    statusSource := "(calculated)"
    if feature.StatusOverride {
        statusSource = "(manual override)"
    }
    cli.Info(fmt.Sprintf("Status: %s %s", feature.Status, statusSource))

    // ... rest of display ...
}
```

### 4.3 Epic Update Command

**File:** `internal/cli/commands/epic.go`

Similar implementation to feature update command.

---

## 5. Performance Considerations

### 5.1 Query Optimization

**Aggregation Queries:**
- Single query per entity (no N+1 problem)
- Uses `GROUP BY` with `COUNT(*)` - highly optimized by SQLite
- Composite indexes reduce query time from O(n) to O(log n)

**Benchmark Targets:**
- Feature with 100 tasks: < 10ms
- Epic with 50 features: < 10ms
- Full cascade (task → feature → epic): < 30ms

### 5.2 Caching Strategy

**Current Design: No caching**

**Rationale:**
- Status stored in database (already cached)
- Recalculation only on status changes (not on reads)
- SQLite WAL mode handles concurrent reads efficiently
- Premature optimization risk

**Future Optimization (if needed):**
- Add `calculated_status` column alongside `status`
- Update `calculated_status` on child changes
- Use `status_override ? status : calculated_status` for reads
- Trades write performance for read performance

### 5.3 Denormalization Trade-offs

**Option A: Fully Computed (Current Design)**
- Status stored in database, updated on cascade
- ✅ Fast reads (no calculation)
- ✅ Simple to understand
- ❌ Requires cascade on every change

**Option B: Fully Normalized**
- Status calculated on every read
- ❌ Slower reads (aggregation query)
- ✅ No cascade complexity
- ❌ Inconsistent if not cached

**Option C: Hybrid (Future Enhancement)**
- Store both `status` and `calculated_status`
- Use override flag to choose which to display
- ✅ Best of both worlds
- ❌ More complex schema

**Decision: Option A (Fully Computed)**
- Aligns with existing progress calculation pattern
- Acceptable performance for typical use cases
- Can upgrade to Option C if performance becomes issue

### 5.4 Scalability Limits

**Expected Limits:**
- Epics: 100+
- Features per epic: 100+
- Tasks per feature: 200+

**Worst Case Cascade:**
- Epic with 100 features
- Feature with 200 tasks
- Task status change triggers:
  - 1 query to update task
  - 1 query to aggregate feature's tasks
  - 1 query to update feature
  - 1 query to aggregate epic's features
  - 1 query to update epic
- Total: 5 queries, ~30ms

**Mitigation for Large Projects:**
- Batch recalculation command (recalculate all at once)
- Optional: Async cascade with background worker
- Optional: Rate limiting on cascade triggers

---

## 6. Testing Strategy

### 6.1 Unit Tests

**Status Derivation (`internal/status/derivation_test.go`):**
- Test all task composition scenarios (27 cases from test-plan.md)
- Test all feature composition scenarios (9 cases)
- Test edge cases (empty, single entity, all same status)
- **Target: 100% coverage**

**Repository Methods:**
- Test `GetTaskStatusBreakdown` with various task distributions
- Test `GetFeatureStatusBreakdown` with various feature distributions
- Test `RecalculateAndUpdateStatus` with and without override
- Test cascade chain (task → feature → epic)
- **Target: 90%+ coverage**

### 6.2 Integration Tests

**Cascade Integration (`internal/repository/cascade_test.go`):**
- Create epic with features with tasks
- Update task status, verify feature + epic recalculated
- Set feature override, verify cascade stops at feature
- Clear feature override, verify recalculation resumes
- Test concurrent updates (pessimistic locking)

### 6.3 CLI Tests

**Feature Update Command:**
- Test `--status=active` sets manual override
- Test `--status=auto` clears override
- Test JSON output format
- Test error cases (invalid status, not found)

**Feature Get Command:**
- Test displays "(calculated)" or "(manual override)"
- Test JSON includes `status_source` field

### 6.4 Performance Tests

**Benchmark Suite:**
```go
func BenchmarkFeatureStatusCalculation(b *testing.B) {
    // Feature with 100 tasks
    // Measure recalculation time
}

func BenchmarkEpicStatusCalculation(b *testing.B) {
    // Epic with 50 features
    // Measure recalculation time
}

func BenchmarkFullCascade(b *testing.B) {
    // Task status change through epic
    // Measure end-to-end time
}
```

**Acceptance Criteria:**
- Feature calculation: < 10ms (p95)
- Epic calculation: < 10ms (p95)
- Full cascade: < 30ms (p95)

---

## 7. Migration Path

### 7.1 Phase 1: Database Migration
- Run `shark init` to apply migrations
- Add `status_override` columns with defaults
- Create indexes
- No user action required

### 7.2 Phase 2: Feature Implementation
- Implement status calculation package
- Add repository methods
- No user-visible changes yet

### 7.3 Phase 3: Manual Trigger (Temporary)
- Add `shark feature recalculate <key>` command
- Add `shark epic recalculate <key>` command
- Users can manually trigger recalculation

### 7.4 Phase 4: Automatic Cascade
- Integrate cascade into task status commands
- Status updates automatically propagate
- Users see immediate status updates

### 7.5 Phase 5: Manual Override
- Add `--status` flag to update commands
- Add `--status=auto` to clear override
- Complete feature rollout

---

## 8. Rollback Strategy

### 8.1 Immediate Rollback (Pre-Release)
- Revert code changes
- Schema changes remain (benign - defaults to false)
- No data loss

### 8.2 Post-Release Rollback
- Disable automatic cascade (feature flag)
- Users continue using manual status updates
- `status_override` column remains (all set to true)

### 8.3 Emergency Rollback
```sql
-- Disable automatic calculation for all entities
UPDATE features SET status_override = 1;
UPDATE epics SET status_override = 1;
```

---

## 9. Open Questions & Decisions

### 9.1 Progress Calculation Enhancement

**Question:** Should progress include ready_for_review as partial completion?

**Current:** `progress = completed_count / total_count * 100`

**Proposed:**
```
progress = (completed_count + ready_for_review_count * 0.9) / total_count * 100
```

**Decision:** OUT OF SCOPE for E07-F14
- Orthogonal concern (progress vs status)
- Existing progress calculation works well
- Can be enhanced in separate feature

### 9.2 Blocked Status Propagation

**Question:** Should blocked features make epic blocked?

**Current Design:** Blocked features count as "active" for epic

**Alternative:** Epic becomes "blocked" if any feature is blocked

**Decision:** CURRENT DESIGN (blocked = active)
- Rationale: Blocked is temporary state, not permanent
- Epic may have other active features
- Manual override available if needed

### 9.3 Archived Status Behavior

**Question:** Should archived be auto-calculated or manual-only?

**Current Design:** Archived tasks count as "completed" for feature
- Feature can become "completed" if all tasks archived
- Epic can become "completed" if all features archived

**Alternative:** Archived always requires manual override

**Decision:** CURRENT DESIGN
- Rationale: Archived is a form of completion (work done, just deprecated)
- Matches user expectation that "all work complete" = completed
- Manual override available for exceptions

### 9.4 History Tracking

**Question:** Should we track feature/epic status history?

**Current:** Only task_history table exists

**Decision:** OUT OF SCOPE for E07-F14
- Add in future feature if needed
- Can be added without breaking changes
- Focus on core calculation first

---

## 10. Success Criteria

### 10.1 Functional Requirements
- [ ] Feature status automatically updates when tasks change
- [ ] Epic status automatically updates when features change
- [ ] Manual override with `--status=<value>` works
- [ ] Clear override with `--status=auto` works
- [ ] CLI shows "(calculated)" or "(manual override)"
- [ ] JSON output includes `status_source` field

### 10.2 Non-Functional Requirements
- [ ] Feature calculation < 10ms (p95)
- [ ] Epic calculation < 10ms (p95)
- [ ] Full cascade < 30ms (p95)
- [ ] All existing tests pass
- [ ] New tests have 90%+ coverage
- [ ] Zero breaking changes to existing commands

### 10.3 Documentation Requirements
- [ ] Architecture document (this file) ✅
- [ ] API documentation updated
- [ ] CLI help text updated
- [ ] Migration guide for users

---

## 11. Implementation Task Breakdown

Based on existing task files in E07-F14:

### Phase 1: Foundation (Tasks T-E07-F14-001 to T-E07-F14-005)
1. T-E07-F14-001: Add status_override column to features
2. T-E07-F14-002: Create status derivation logic
3. T-E07-F14-003: Add GetTaskStatusBreakdown to FeatureRepository
4. T-E07-F14-004: Add CalculateFeatureStatus method
5. T-E07-F14-005: Add status_override column to epics

### Phase 2: Calculation Service (Tasks T-E07-F14-006 to T-E07-F14-008)
6. T-E07-F14-006: Create StatusCalculationService
7. T-E07-F14-007: Implement RecalculateFeatureStatus
8. T-E07-F14-008: Implement RecalculateEpicStatus

### Phase 3: Cascade Integration (Tasks T-E07-F14-009 to T-E07-F14-010)
9. T-E07-F14-009: Integrate cascade into task status commands
10. T-E07-F14-010: Integrate cascade into feature create/delete

### Phase 4: Manual Override (Tasks T-E07-F14-011 to T-E07-F14-012)
11. T-E07-F14-011: Add feature update command with status override
12. T-E07-F14-012: Add epic update command with status override

**Estimated Total:** 12 tasks, ~40-60 hours of implementation

---

## 12. Risk Assessment

### 12.1 Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Performance degradation on large projects | High | Medium | Benchmark early, optimize indexes, add caching if needed |
| Race conditions in concurrent updates | Medium | Low | SQLite WAL mode + transactions handle this |
| Migration failure on existing databases | High | Low | Idempotent migrations, extensive testing |
| Breaking existing workflows | High | Low | Defaults preserve current behavior, backward compatible |

### 12.2 User Experience Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Users confused by automatic status changes | Medium | Medium | Clear "(calculated)" indicator, documentation |
| Users unable to manually control status | Low | Low | Manual override with `--status` flag |
| Status changes unexpectedly | Medium | Low | Verbose logging, change notifications |

### 12.3 Mitigation Summary

**Overall Risk Level:** LOW to MEDIUM

**Key Mitigations:**
1. Extensive testing before release
2. Performance benchmarks on large datasets
3. Clear documentation and user communication
4. Gradual rollout with feature flags
5. Easy rollback strategy

---

## 13. Alternatives Considered

### 13.1 Alternative A: Database Triggers

**Approach:** Use SQLite triggers to update status automatically

**Pros:**
- Guaranteed consistency (atomic with transaction)
- No application code changes needed
- Works even with direct database access

**Cons:**
- Complex trigger logic (hard to debug)
- Logic split between app and database
- Harder to test
- Harder to add conditions (override logic)

**Decision:** REJECTED - Too complex, less testable

### 13.2 Alternative B: Event-Driven Architecture

**Approach:** Publish task status change events, subscribers update status

**Pros:**
- Loose coupling
- Easy to add new subscribers
- Async processing possible

**Cons:**
- Over-engineered for current scale
- Adds complexity (event queue, handlers)
- Harder to debug (eventual consistency)
- No event infrastructure exists

**Decision:** REJECTED - Premature optimization

### 13.3 Alternative C: Periodic Recalculation

**Approach:** Cron job recalculates all statuses every N minutes

**Pros:**
- Simple implementation
- No cascade complexity
- Batched updates (efficient)

**Cons:**
- Stale data (up to N minutes)
- Wastes CPU on unchanged entities
- Poor user experience (delayed updates)

**Decision:** REJECTED - Poor UX, doesn't meet requirements

### 13.4 Selected Approach: Application-Layer Cascade

**Rationale:**
- Balance of simplicity and correctness
- Testable and debuggable
- Immediate updates (good UX)
- Flexible (easy to add conditions)
- Aligns with existing patterns

---

## 14. Future Enhancements

### 14.1 Phase 2 Features (Post-E07-F14)

**Status History Tracking:**
- Add `feature_history` and `epic_history` tables
- Track who/when/why status changed
- Enable audit trail and analytics

**Notification System:**
- Notify stakeholders when epic/feature completes
- Slack/email integration
- Webhook support

**Bulk Recalculation:**
- `shark recalculate-all` command
- Recalculate all epics/features at once
- Progress bar, dry-run mode

**Advanced Status Rules:**
- Configurable status calculation rules
- Custom status values per project
- Weight-based status calculation

### 14.2 Performance Enhancements

**Async Cascade:**
- Background worker for cascade updates
- Queue status changes
- Batch updates every N seconds

**Caching Layer:**
- Redis cache for calculated statuses
- Cache invalidation on updates
- Reduces database load

**Read Replicas:**
- Separate read/write databases
- Status calculation on replicas
- Reduces contention

---

## Appendix A: SQL Query Reference

### A.1 Feature Status Calculation Query

```sql
-- Single-query status calculation for feature
SELECT
    CASE
        WHEN COUNT(*) = 0 THEN 'draft'
        WHEN COUNT(*) = SUM(CASE WHEN status IN ('completed', 'archived') THEN 1 ELSE 0 END) THEN 'completed'
        WHEN SUM(CASE WHEN status IN ('in_progress', 'ready_for_review', 'blocked') THEN 1 ELSE 0 END) > 0 THEN 'active'
        ELSE 'draft'
    END as calculated_status
FROM tasks
WHERE feature_id = ?;
```

### A.2 Epic Status Calculation Query

```sql
-- Single-query status calculation for epic
SELECT
    CASE
        WHEN COUNT(*) = 0 THEN 'draft'
        WHEN COUNT(*) = SUM(CASE WHEN status IN ('completed', 'archived') THEN 1 ELSE 0 END) THEN 'completed'
        WHEN SUM(CASE WHEN status IN ('active', 'blocked') THEN 1 ELSE 0 END) > 0 THEN 'active'
        ELSE 'draft'
    END as calculated_status
FROM features
WHERE epic_id = ?;
```

### A.3 Batch Recalculation Query

```sql
-- Find all features needing recalculation
SELECT
    f.id,
    f.key,
    f.status as current_status,
    CASE
        WHEN COUNT(t.id) = 0 THEN 'draft'
        WHEN COUNT(t.id) = SUM(CASE WHEN t.status IN ('completed', 'archived') THEN 1 ELSE 0 END) THEN 'completed'
        WHEN SUM(CASE WHEN t.status IN ('in_progress', 'ready_for_review', 'blocked') THEN 1 ELSE 0 END) > 0 THEN 'active'
        ELSE 'draft'
    END as calculated_status
FROM features f
LEFT JOIN tasks t ON t.feature_id = f.id
WHERE (f.status_override = 0 OR f.status_override IS NULL)
GROUP BY f.id, f.key, f.status
HAVING calculated_status != current_status;
```

---

## Appendix B: Example CLI Session

```bash
# Initial state
$ shark task list E07 F14
T-E07-F14-001  Draft implementation         todo
T-E07-F14-002  Add unit tests               todo
T-E07-F14-003  Integration tests            todo

$ shark feature get E07-F14
Feature: E07-F14 - Status Calculation
Status: draft (calculated)
Progress: 0%
Tasks: 3

# Start first task
$ shark task start E07-F14-001
✓ Task E07-F14-001 started
ℹ Feature E07-F14 status changed: draft → active
ℹ Epic E07 status changed: draft → active

# Complete tasks
$ shark task complete E07-F14-001
✓ Task E07-F14-001 marked ready for review
ℹ Feature E07-F14 remains: active (1 in review, 2 todo)

$ shark task approve E07-F14-001
✓ Task E07-F14-001 completed
ℹ Feature E07-F14 remains: active (1 completed, 2 todo)

# Complete all tasks
$ shark task start E07-F14-002 && shark task complete E07-F14-002 && shark task approve E07-F14-002
$ shark task start E07-F14-003 && shark task complete E07-F14-003 && shark task approve E07-F14-003
✓ All tasks completed
ℹ Feature E07-F14 status changed: active → completed
ℹ Epic E07 status changed: active → completed

# Manual override
$ shark feature update E07-F14 --status=active
✓ Feature E07-F14 status set to active (manual override)
ℹ Epic E07 status changed: completed → active

$ shark feature get E07-F14
Feature: E07-F14 - Status Calculation
Status: active (manual override)
Progress: 100%
Tasks: 3 (all completed)

# Clear override
$ shark feature update E07-F14 --status=auto
✓ Feature E07-F14: status override cleared
ℹ Status changed: active → completed
ℹ Epic E07 status changed: active → completed
```

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
**Status:** Ready for Review
