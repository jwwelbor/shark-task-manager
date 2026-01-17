# Technical Architecture Design: Config-Driven Status Calculation (E07-F14 v2)

**Date:** 2026-01-16 (Updated)
**Author:** Architect Agent
**Feature:** E07-F14 - Cascading Status Calculation
**Status:** Design Proposal v2 (Config-Driven)

---

## Executive Summary

This document proposes a **fully config-driven architecture** for automatic feature/epic status calculation. Key changes from v1:

1. **❌ REMOVED: `status_override` column** - Feature/epic status is ALWAYS calculated from tasks
2. **✅ ADDED: Enhanced config metadata** - `progress_weight`, `responsibility`, `blocks_feature`
3. **✅ ADDED: Feature-level shortcuts** - `shark feature pause` applies to all tasks
4. **✅ SIMPLIFIED: No manual overrides** - Change tasks to change feature status

---

## 1. Configuration-Driven Design

### 1.1 Enhanced Status Metadata

**File:** `.sharkconfig.json`

```json
{
  "status_metadata": {
    "ready_for_approval": {
      "color": "purple",
      "phase": "approval",
      "description": "Awaiting final approval",
      "agent_types": ["product-manager", "client"],

      "progress_weight": 0.9,        // NEW: 90% complete (agent done!)
      "responsibility": "human",      // NEW: who's responsible
      "blocks_feature": false         // NEW: should feature show blocked?
    },
    "in_development": {
      "color": "yellow",
      "phase": "development",
      "description": "Code implementation in progress",
      "agent_types": ["developer", "ai-coder"],

      "progress_weight": 0.5,         // NEW: 50% complete (in progress)
      "responsibility": "agent",       // NEW: agent work
      "blocks_feature": false
    },
    "blocked": {
      "color": "red",
      "phase": "any",
      "description": "Temporarily blocked by external dependency",

      "progress_weight": 0.0,         // NEW: 0% (no progress while blocked)
      "responsibility": "none",        // NEW: waiting on external
      "blocks_feature": true           // NEW: YES - feature shows blocked
    },
    "completed": {
      "color": "white",
      "phase": "done",
      "description": "Task finished and approved",

      "progress_weight": 1.0,         // NEW: 100% complete
      "responsibility": "none",        // NEW: done
      "blocks_feature": false
    },
    "ready_for_code_review": {
      "color": "magenta",
      "phase": "review",
      "description": "Code complete, awaiting review",
      "agent_types": ["tech-lead", "code-reviewer"],

      "progress_weight": 0.85,        // NEW: 85% complete (dev done, needs review)
      "responsibility": "human",       // NEW: human reviewer
      "blocks_feature": false
    },
    "ready_for_qa": {
      "color": "green",
      "phase": "qa",
      "description": "Ready for quality assurance testing",
      "agent_types": ["qa", "test-engineer"],

      "progress_weight": 0.80,        // NEW: 80% complete (code/review done, needs QA)
      "responsibility": "qa_team",     // NEW: QA responsibility
      "blocks_feature": false
    },
    "draft": {
      "color": "gray",
      "phase": "planning",
      "description": "Task created but not yet refined",

      "progress_weight": 0.0,         // NEW: 0% (not started)
      "responsibility": "none",        // NEW: not assigned
      "blocks_feature": false
    },
    "on_hold": {
      "color": "orange",
      "phase": "any",
      "description": "Intentionally paused",

      "progress_weight": 0.0,         // NEW: 0% (paused)
      "responsibility": "none",        // NEW: waiting for decision
      "blocks_feature": false          // Different from blocked - this is intentional
    }
  },

  "special_statuses": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"],

    // NEW: Semantic groups for calculated status
    "_agent_complete_": ["ready_for_approval", "ready_for_qa", "ready_for_code_review"],
    "_human_work_": ["in_approval", "ready_for_approval", "in_code_review", "ready_for_code_review"],
    "_blocking_": ["blocked"],
    "_paused_": ["on_hold"]
  },

  // NEW: Top-level rejection reason config (E07-F22)
  "require_rejection_reason": true,

  // NEW: Top-level workflow config
  "workflow": {
    "enable_auto_status_calculation": true,
    "cascade_to_epic": true,
    "cascade_to_feature": true
  }
}
```

### 1.2 New Config Fields Explained

| Field | Type | Purpose | Example |
|-------|------|---------|---------|
| `progress_weight` | float (0.0-1.0) | How much this status contributes to progress | `0.9` = 90% done |
| `responsibility` | string | Who's responsible for this status | `agent`, `human`, `qa_team`, `none` |
| `blocks_feature` | boolean | Should this status make feature show "blocked"? | `true` for blocked status |
| `require_rejection_reason` | boolean | Require reason when progress_weight decreases | `true` = require reasons |

---

## 2. Simplified Data Model

### 2.1 NO Schema Changes Needed

**✅ Feature/Epic Status Already Exists:**
- `features.status` - Already stored
- `epics.status` - Already stored

**❌ NO status_override Column:**
- Feature/epic status is ALWAYS calculated
- No manual overrides at feature/epic level
- Change tasks to change feature status

**✅ Use Existing Infrastructure:**
- task_history - Already tracks status changes
- Existing indexes - Already optimized

### 2.2 Model Updates

**Feature Model** (`internal/models/feature.go`):
```go
type Feature struct {
    ID          int64         `json:"id" db:"id"`
    Key         string        `json:"key" db:"key"`
    EpicID      int64         `json:"epic_id" db:"epic_id"`
    Title       string        `json:"title" db:"title"`
    Status      FeatureStatus `json:"status" db:"status"`
    // NO status_override field!
    CreatedAt   time.Time     `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
}

// StatusSource always returns "calculated"
func (f *Feature) StatusSource() string {
    return "calculated"
}
```

**Epic Model** (`internal/models/epic.go`):
```go
type Epic struct {
    ID          int64      `json:"id" db:"id"`
    Key         string     `json:"key" db:"key"`
    Title       string     `json:"title" db:"title"`
    Status      EpicStatus `json:"status" db:"status"`
    // NO status_override field!
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// StatusSource always returns "calculated"
func (e *Epic) StatusSource() string {
    return "calculated"
}
```

---

## 3. Config-Driven Status Calculation

### 3.1 Status Derivation Logic

**New Package:** `internal/status/`

**File:** `internal/status/derivation.go`

```go
package status

import (
    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// DeriveFeatureStatus calculates feature status from task breakdown (config-driven)
func DeriveFeatureStatus(statusCounts map[string]int, cfg *config.Config) string {
    totalTasks := 0
    for _, count := range statusCounts {
        totalTasks += count
    }

    // No tasks = draft
    if totalTasks == 0 {
        return "draft"
    }

    // Check if all tasks are in "complete" statuses (from config)
    completeCount := 0
    for _, completeStatus := range cfg.SpecialStatuses["_complete_"] {
        completeCount += statusCounts[completeStatus]
    }
    if completeCount == totalTasks {
        return "completed"
    }

    // Check if feature is blocked (any tasks with blocks_feature=true)
    for status, count := range statusCounts {
        if count > 0 {
            meta := cfg.GetStatusMetadata(status)
            if meta != nil && meta.BlocksFeature {
                return "blocked"
            }
        }
    }

    // Check if feature is paused (all tasks on_hold)
    pausedCount := 0
    for _, pausedStatus := range cfg.SpecialStatuses["_paused_"] {
        pausedCount += statusCounts[pausedStatus]
    }
    if pausedCount == totalTasks {
        return "on_hold"
    }

    // Check if any work has started (not all draft/todo)
    startStatuses := cfg.SpecialStatuses["_start_"]
    notStartedCount := 0
    for _, startStatus := range startStatuses {
        notStartedCount += statusCounts[startStatus]
    }

    if notStartedCount == totalTasks {
        return "draft"  // All tasks not started
    }

    // Default: work in progress
    return "active"
}

// DeriveEpicStatus calculates epic status from feature breakdown (config-driven)
func DeriveEpicStatus(featureStatusCounts map[string]int, cfg *config.Config) string {
    totalFeatures := 0
    for _, count := range featureStatusCounts {
        totalFeatures += count
    }

    // No features = draft
    if totalFeatures == 0 {
        return "draft"
    }

    // All complete = completed
    completeCount := featureStatusCounts["completed"] + featureStatusCounts["archived"]
    if completeCount == totalFeatures {
        return "completed"
    }

    // Any blocked features = blocked
    if featureStatusCounts["blocked"] > 0 {
        return "blocked"
    }

    // Any active features = active
    if featureStatusCounts["active"] > 0 {
        return "active"
    }

    // All draft = draft
    if featureStatusCounts["draft"] == totalFeatures {
        return "draft"
    }

    // Default: active
    return "active"
}
```

### 3.2 Progress Calculation (Config-Driven)

**File:** `internal/status/progress.go`

```go
package status

import (
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

// CalculateProgress computes weighted progress from task statuses
func CalculateProgress(statusCounts map[string]int, cfg *config.Config) float64 {
    totalTasks := 0
    weightedProgress := 0.0

    for status, count := range statusCounts {
        totalTasks += count

        // Get progress weight from config
        meta := cfg.GetStatusMetadata(status)
        if meta != nil {
            weightedProgress += float64(count) * meta.ProgressWeight
        }
    }

    if totalTasks == 0 {
        return 0.0
    }

    return (weightedProgress / float64(totalTasks)) * 100.0
}

// CalculateWorkRemaining computes work breakdown by responsibility
func CalculateWorkRemaining(statusCounts map[string]int, cfg *config.Config) WorkSummary {
    summary := WorkSummary{
        TotalTasks:      0,
        WorkRemaining:   0,
        AgentWork:       0,
        HumanWork:       0,
        BlockedWork:     0,
        NotStarted:      0,
    }

    for status, count := range statusCounts {
        summary.TotalTasks += count

        meta := cfg.GetStatusMetadata(status)
        if meta == nil {
            continue
        }

        // Not complete = work remaining
        if meta.ProgressWeight < 1.0 {
            summary.WorkRemaining += count
        }

        // Categorize by responsibility
        switch meta.Responsibility {
        case "agent":
            summary.AgentWork += count
        case "human", "qa_team":
            summary.HumanWork += count
        case "none":
            if meta.BlocksFeature {
                summary.BlockedWork += count
            } else if meta.ProgressWeight == 0.0 {
                summary.NotStarted += count
            }
        }
    }

    return summary
}

type WorkSummary struct {
    TotalTasks      int     `json:"total_tasks"`
    WorkRemaining   int     `json:"work_remaining"`
    AgentWork       int     `json:"agent_work"`
    HumanWork       int     `json:"human_work"`
    BlockedWork     int     `json:"blocked_work"`
    NotStarted      int     `json:"not_started"`
}
```

---

## 4. Feature-Level Shortcuts

### 4.1 Bulk Task Operations

Feature-level commands apply to all constituent tasks:

**File:** `internal/cli/commands/feature_bulk.go`

```go
// Pause feature: set all active tasks to on_hold
func PauseFeature(ctx context.Context, featureKey string) error {
    featureRepo := repository.NewFeatureRepository(db)
    taskRepo := repository.NewTaskRepository(db)

    // Get feature
    feature, err := featureRepo.GetByKey(ctx, featureKey)
    if err != nil {
        return err
    }

    // Get all tasks not in complete statuses
    tasks, err := taskRepo.GetByFeatureID(ctx, feature.ID)
    if err != nil {
        return err
    }

    cfg := config.Get()
    completeStatuses := cfg.SpecialStatuses["_complete_"]

    // Update active tasks to on_hold
    for _, task := range tasks {
        if !contains(completeStatuses, task.Status) {
            err = taskRepo.UpdateStatus(ctx, task.Key, "on_hold")
            if err != nil {
                return fmt.Errorf("failed to pause task %s: %w", task.Key, err)
            }
        }
    }

    // Feature status recalculates automatically
    return nil
}

// Resume feature: set all on_hold tasks back to previous status
func ResumeFeature(ctx context.Context, featureKey string) error {
    // Similar implementation, restoring from task_history
}

// Cancel feature: set all incomplete tasks to cancelled
func CancelFeature(ctx context.Context, featureKey string) error {
    // Similar implementation, setting to "cancelled"
}
```

### 4.2 CLI Commands

```bash
# Pause feature (all tasks → on_hold)
shark feature pause E07-F22

# Resume feature (restore from on_hold)
shark feature resume E07-F22

# Cancel feature (all incomplete → cancelled)
shark feature cancel E07-F22

# Archive epic (all tasks → archived)
shark epic archive E07
```

---

## 5. Repository Layer (Simplified)

### 5.1 Feature Repository

**File:** `internal/repository/feature_repository.go`

```go
// GetTaskStatusBreakdown returns task counts grouped by status
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
            return nil, err
        }
        breakdown[status] = count
    }

    return breakdown, nil
}

// RecalculateStatus recalculates feature status from tasks (always, no override)
func (r *FeatureRepository) RecalculateStatus(ctx context.Context, featureID int64) (*StatusChangeResult, error) {
    // Get current feature
    feature, err := r.GetByID(ctx, featureID)
    if err != nil {
        return nil, err
    }

    // Get task breakdown
    breakdown, err := r.GetTaskStatusBreakdown(ctx, featureID)
    if err != nil {
        return nil, err
    }

    // Calculate new status (config-driven)
    cfg := config.Get()
    newStatus := status.DeriveFeatureStatus(breakdown, cfg)

    result := &StatusChangeResult{
        EntityType:     "feature",
        EntityKey:      feature.Key,
        EntityID:       feature.ID,
        PreviousStatus: feature.Status,
        NewStatus:      newStatus,
        Changed:        feature.Status != newStatus,
        CalculatedAt:   time.Now(),
    }

    // Update if changed
    if result.Changed {
        updateQuery := `
            UPDATE features
            SET status = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        `
        _, err = r.db.ExecContext(ctx, updateQuery, newStatus, featureID)
        if err != nil {
            return nil, fmt.Errorf("failed to update feature status: %w", err)
        }
    }

    // Cascade to epic
    epicRepo := NewEpicRepository(r.db)
    epicResult, err := epicRepo.RecalculateStatus(ctx, feature.EpicID)
    if err != nil {
        return result, fmt.Errorf("failed to cascade to epic: %w", err)
    }

    result.EpicChange = epicResult
    return result, nil
}
```

### 5.2 Task Repository Integration

**File:** `internal/repository/task_repository.go`

```go
// UpdateStatus updates task status and cascades to feature/epic
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskKey string, newStatus string, opts UpdateStatusOptions) error {
    // Get task
    task, err := r.GetByKey(ctx, taskKey)
    if err != nil {
        return err
    }

    // Check if rejection reason needed (E07-F22)
    cfg := config.Get()
    if cfg.RequireRejectionReason {
        if err := r.validateRejectionReason(task.Status, newStatus, opts.RejectionReason, cfg); err != nil {
            return err
        }
    }

    // Update task status
    updateQuery := `
        UPDATE tasks
        SET status = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    _, err = r.db.ExecContext(ctx, updateQuery, newStatus, task.ID)
    if err != nil {
        return fmt.Errorf("failed to update task status: %w", err)
    }

    // Record history with rejection reason
    if err := r.recordHistory(ctx, task.ID, task.Status, newStatus, opts); err != nil {
        return fmt.Errorf("failed to record history: %w", err)
    }

    // Cascade to feature (which cascades to epic)
    featureRepo := NewFeatureRepository(r.db)
    _, err = featureRepo.RecalculateStatus(ctx, task.FeatureID)
    if err != nil {
        return fmt.Errorf("failed to cascade status change: %w", err)
    }

    return nil
}

// validateRejectionReason checks if rejection reason is required (E07-F22)
func (r *TaskRepository) validateRejectionReason(oldStatus, newStatus string, reason string, cfg *config.Config) error {
    // Get progress weights from config
    oldMeta := cfg.GetStatusMetadata(oldStatus)
    newMeta := cfg.GetStatusMetadata(newStatus)

    if oldMeta == nil || newMeta == nil {
        return nil  // Unknown status, skip validation
    }

    // If progress_weight decreases, it's a backward transition
    if newMeta.ProgressWeight < oldMeta.ProgressWeight {
        if reason == "" {
            return fmt.Errorf("rejection reason required for backward transition (%s → %s)", oldStatus, newStatus)
        }
    }

    return nil
}

type UpdateStatusOptions struct {
    RejectionReason string  // E07-F22: Required if progress decreases
    Notes           string  // Optional notes
    Agent           string  // Agent ID
}
```

---

## 6. Config Package Enhancements

### 6.1 Config Struct

**File:** `internal/config/config.go`

```go
type Config struct {
    StatusMetadata    map[string]StatusMetadata `json:"status_metadata"`
    SpecialStatuses   map[string][]string       `json:"special_statuses"`
    StatusFlow        map[string][]string       `json:"status_flow"`

    // NEW: Top-level config options
    RequireRejectionReason bool `json:"require_rejection_reason"`

    Workflow WorkflowConfig `json:"workflow"`
}

type StatusMetadata struct {
    Color          string   `json:"color"`
    Phase          string   `json:"phase"`
    Description    string   `json:"description"`
    AgentTypes     []string `json:"agent_types,omitempty"`

    // NEW: Calculation metadata
    ProgressWeight float64  `json:"progress_weight"`  // 0.0 to 1.0
    Responsibility string   `json:"responsibility"`    // agent, human, qa_team, none
    BlocksFeature  bool     `json:"blocks_feature"`    // true if feature should show blocked
}

type WorkflowConfig struct {
    EnableAutoStatusCalculation bool `json:"enable_auto_status_calculation"`
    CascadeToEpic              bool `json:"cascade_to_epic"`
    CascadeToFeature           bool `json:"cascade_to_feature"`
}

// GetStatusMetadata retrieves metadata for a status
func (c *Config) GetStatusMetadata(status string) *StatusMetadata {
    if meta, ok := c.StatusMetadata[status]; ok {
        return &meta
    }
    return nil
}

// IsBackwardTransition checks if transition decreases progress
func (c *Config) IsBackwardTransition(oldStatus, newStatus string) bool {
    oldMeta := c.GetStatusMetadata(oldStatus)
    newMeta := c.GetStatusMetadata(newStatus)

    if oldMeta == nil || newMeta == nil {
        return false
    }

    return newMeta.ProgressWeight < oldMeta.ProgressWeight
}
```

---

## 7. E07-F22 Integration: Rejection Reason

### 7.1 Automatic Detection

**With config-driven approach, rejection reason detection is automatic:**

```go
// In task_repository.go
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskKey string, newStatus string, opts UpdateStatusOptions) error {
    task, err := r.GetByKey(ctx, taskKey)
    if err != nil {
        return err
    }

    cfg := config.Get()

    // E07-F22: Check if rejection reason required
    if cfg.RequireRejectionReason && cfg.IsBackwardTransition(task.Status, newStatus) {
        if opts.RejectionReason == "" {
            oldMeta := cfg.GetStatusMetadata(task.Status)
            newMeta := cfg.GetStatusMetadata(newStatus)
            return fmt.Errorf(
                "rejection reason required: transitioning from %s (%.0f%%) to %s (%.0f%%) decreases progress",
                task.Status, oldMeta.ProgressWeight*100,
                newStatus, newMeta.ProgressWeight*100,
            )
        }
    }

    // ... rest of update logic
}
```

### 7.2 CLI Integration

```bash
# Backward transition without reason = ERROR
$ shark task update E07-F22-001 --status=in_development
Error: rejection reason required: transitioning from ready_for_code_review (85%) to in_development (50%) decreases progress

# Backward transition WITH reason = SUCCESS
$ shark task update E07-F22-001 --status=in_development --rejection-reason="Failed code review: missing error handling"
✓ Task E07-F22-001 updated: ready_for_code_review → in_development
ℹ Rejection reason: Failed code review: missing error handling
```

### 7.3 Task History with Rejection Reason

```sql
-- task_history table (existing)
CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    from_status TEXT NOT NULL,
    to_status TEXT NOT NULL,
    rejection_reason TEXT,  -- NEW: E07-F22
    notes TEXT,
    changed_by TEXT,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
```

---

## 8. Benefits of Config-Driven Approach

### 8.1 No Hardcoded Status Logic

**Before (hardcoded):**
```go
// BAD - hardcoded status meanings
if task.Status == "ready_for_approval" {
    agentCompleteCount++
}
if task.Status == "blocked" {
    feature.Status = "blocked"
}
```

**After (config-driven):**
```go
// GOOD - config-driven
meta := cfg.GetStatusMetadata(task.Status)
if meta.Responsibility == "human" {
    humanWorkCount++
}
if meta.BlocksFeature {
    feature.Status = "blocked"
}
```

### 8.2 Easy Workflow Customization

**Add new workflow:**
```json
{
  "status_metadata": {
    "waiting_for_qa": {
      "color": "cyan",
      "phase": "qa",
      "description": "Waiting for QA team",
      "progress_weight": 0.85,
      "responsibility": "qa_team",
      "blocks_feature": false
    }
  }
}
```

**Code automatically handles it** - no code changes needed!

### 8.3 Automatic Rejection Reason Detection

**With `progress_weight` in config:**
- `ready_for_code_review` (0.85) → `in_development` (0.50) = backward transition
- `in_qa` (0.80) → `in_development` (0.50) = backward transition
- `in_development` (0.50) → `ready_for_code_review` (0.85) = forward transition

**Detection is automatic based on config, no hardcoded status lists!**

---

## 9. Migration Strategy

### 9.1 Phase 1: Update Config Schema

```bash
# Add new config fields with defaults
cat >> .sharkconfig.json <<EOF
{
  "require_rejection_reason": false,
  "workflow": {
    "enable_auto_status_calculation": true,
    "cascade_to_epic": true,
    "cascade_to_feature": true
  },
  "status_metadata": {
    "ready_for_approval": {
      ...existing fields...,
      "progress_weight": 0.9,
      "responsibility": "human",
      "blocks_feature": false
    },
    ...
  }
}
EOF
```

### 9.2 Phase 2: Deploy Config-Driven Logic

- No database migrations needed!
- Feature/epic status already exists
- Just deploy new calculation logic
- Backward compatible

### 9.3 Phase 3: Enable Features

```bash
# Enable rejection reason requirement
shark config set require_rejection_reason true

# Test backward transition
shark task update E07-F22-001 --status=in_development --rejection-reason="Test rejection"
```

---

## 10. Performance

### 10.1 Same Performance as v1

- Single SQL query per entity
- Composite indexes for fast GROUP BY
- No additional overhead from config reads (cached in memory)

**Benchmarks:** (same as v1)
- Feature with 100 tasks: < 10ms (p95)
- Epic with 50 features: < 10ms (p95)
- Full cascade: < 30ms (p95)

### 10.2 Config Caching

```go
// Config is loaded once at startup, cached in memory
var globalConfig *Config

func Get() *Config {
    if globalConfig == nil {
        globalConfig = Load()
    }
    return globalConfig
}
```

---

## 11. Testing Strategy

### 11.1 Config-Driven Tests

```go
func TestDeriveFeatureStatus_ConfigDriven(t *testing.T) {
    cfg := &config.Config{
        StatusMetadata: map[string]config.StatusMetadata{
            "ready_for_approval": {ProgressWeight: 0.9, BlocksFeature: false},
            "blocked": {ProgressWeight: 0.0, BlocksFeature: true},
        },
        SpecialStatuses: map[string][]string{
            "_complete_": {"completed"},
        },
    }

    tests := []struct {
        name           string
        statusCounts   map[string]int
        expectedStatus string
    }{
        {
            name: "blocked task makes feature blocked",
            statusCounts: map[string]int{
                "in_development": 3,
                "blocked": 1,
            },
            expectedStatus: "blocked",
        },
        {
            name: "all ready_for_approval = active",
            statusCounts: map[string]int{
                "ready_for_approval": 5,
            },
            expectedStatus: "active",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := status.DeriveFeatureStatus(tt.statusCounts, cfg)
            if result != tt.expectedStatus {
                t.Errorf("expected %s, got %s", tt.expectedStatus, result)
            }
        })
    }
}
```

---

## 12. Summary of Changes from v1

| Aspect | v1 (Manual Override) | v2 (Config-Driven) |
|--------|---------------------|-------------------|
| **Schema** | Add `status_override` column | NO schema changes |
| **Feature Status** | Can be manual or calculated | ALWAYS calculated |
| **Manual Control** | `--status=active` sets override | Use feature shortcuts |
| **Status Meanings** | Hardcoded in Go code | Defined in config |
| **Progress Calc** | Hardcoded weights | `progress_weight` in config |
| **Rejection Reason** | Hardcoded transition list | Automatic (progress decrease) |
| **Workflow Changes** | Code changes required | Config changes only |
| **Complexity** | Medium (override logic) | Low (pure calculation) |
| **Maintainability** | Split logic (code + config) | Single source (config) |

---

## 13. Implementation Tasks

### Phase 1: Config Enhancement (E07-F14)
1. T-E07-F14-001: Add enhanced config fields (progress_weight, responsibility, blocks_feature)
2. T-E07-F14-002: Create config-driven status derivation logic
3. T-E07-F14-003: Implement GetTaskStatusBreakdown (repository)
4. T-E07-F14-004: Implement RecalculateStatus (no override check)
5. T-E07-F14-005: Create epic status calculation (config-driven)

### Phase 2: Progress Calculation (E07-F14)
6. T-E07-F14-006: Implement weighted progress calculation
7. T-E07-F14-007: Implement work remaining breakdown
8. T-E07-F14-008: Add status breakdown API

### Phase 3: Cascade Integration (E07-F14)
9. T-E07-F14-009: Integrate cascade into task status commands
10. T-E07-F14-010: Add feature-level shortcuts (pause/resume/cancel)

### Phase 4: Rejection Reason (E07-F22)
11. T-E07-F22-001: Add rejection_reason to task_history
12. T-E07-F22-002: Implement backward transition detection
13. T-E07-F22-003: Add CLI support for --rejection-reason flag
14. T-E07-F22-004: Add config option require_rejection_reason

**Estimated Total:** 14 tasks, ~30-40 hours

---

## 14. Success Criteria

### Functional Requirements
- [ ] Feature status calculated from task breakdown using config
- [ ] Epic status calculated from feature breakdown using config
- [ ] Progress uses weighted calculation from config
- [ ] Rejection reason required when progress_weight decreases
- [ ] Feature shortcuts (pause/resume/cancel) update tasks
- [ ] CLI shows status source as "calculated"

### Non-Functional Requirements
- [ ] No hardcoded status logic (all config-driven)
- [ ] Performance same as v1 (< 30ms cascade)
- [ ] Zero database migrations needed
- [ ] Backward compatible with existing configs
- [ ] Easy to add new workflows (config only)

---

**Document Version:** 2.0 (Config-Driven)
**Last Updated:** 2026-01-16
**Status:** Ready for Review
