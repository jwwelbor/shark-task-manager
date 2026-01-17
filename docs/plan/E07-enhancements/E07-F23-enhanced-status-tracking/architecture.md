# E07-F23: Enhanced Status Tracking and Visibility - Technical Architecture

**Feature:** E07-F23 - Enhanced Status Tracking and Visibility
**Epic:** E07 - Shark Enhancements
**Date:** 2026-01-16
**Author:** Architect Agent
**Status:** Design Proposal
**Depends On:** E07-F14 (Cascading Status Calculation)

---

## Executive Summary

This feature enhances status displays for features and epics to provide **quick, actionable visibility** into work state. It leverages the config-driven architecture from E07-F14 to calculate weighted progress, work breakdowns, and actionable insights.

**Key Innovations:**
- **Weighted Progress**: Progress reflects reality (90% for ready_for_approval)
- **Work Breakdown**: Shows agent work vs. human work vs. blocked
- **Action Items**: Surface tasks needing immediate attention
- **Config-Driven**: Uses `progress_weight`, `responsibility`, `blocks_feature` from config

**Design Principle:** Display enhancements only - NO changes to E07-F14 core logic, NO database schema changes.

---

## 1. Architecture Overview

### 1.1 System Context

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Commands                              │
│  shark feature get E07-F23                                   │
│  shark feature list E07                                      │
│  shark epic get E07                                          │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│              Command Handlers (CLI Layer)                    │
│  - Parse arguments                                           │
│  - Call repository methods                                   │
│  - Format output (JSON or table)                             │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│           Repository Layer (NEW METHODS)                     │
│  - GetTaskStatusBreakdown(featureID)                         │
│  - GetStatusInfo(featureID) → FeatureStatusInfo              │
│  - GetTasksWithActionItems(featureID)                        │
│  - GetFeatureStatusRollup(epicID)                            │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│           Status Package (NEW CALCULATIONS)                  │
│  - CalculateProgress(statusCounts, cfg)                      │
│  - CalculateWorkRemaining(statusCounts, cfg)                 │
│  - GetStatusContext(feature, statusCounts)                   │
│  - GetActionItems(tasks, cfg)                                │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│              Configuration (.sharkconfig.json)               │
│  - status_metadata.progress_weight                           │
│  - status_metadata.responsibility                            │
│  - status_metadata.blocks_feature                            │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Data Flow

**Feature Get Command:**
```
User: shark feature get E07-F23
  ↓
CLI: Parse args, get DB, load config
  ↓
Repository: GetStatusInfo(featureID)
  ├─ GetTaskStatusBreakdown(featureID) → map[status]count
  ├─ GetTasks(featureID) → []*Task
  └─ GetFeature(featureID) → *Feature
  ↓
Status Package: Calculate displays
  ├─ CalculateProgress(statusCounts, cfg) → ProgressInfo
  ├─ CalculateWorkRemaining(statusCounts, cfg) → WorkSummary
  ├─ GetStatusContext(feature, statusCounts) → string
  └─ GetActionItems(tasks, cfg) → ActionItems
  ↓
CLI: Format output (table or JSON)
  ↓
User: View enhanced status display
```

**Performance Characteristics:**
- Single database query for status breakdown (efficient GROUP BY)
- Single query for task list
- All calculations in-memory (no additional DB calls)
- **Target: < 100ms p95 latency**

---

## 2. Data Model

### 2.1 Existing Schema (No Changes)

**NO database schema changes are needed.** This feature uses existing tables and columns:

```sql
-- features table (existing)
CREATE TABLE features (
    id INTEGER PRIMARY KEY,
    epic_id INTEGER NOT NULL,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,              -- Calculated from tasks (E07-F14)
    status_override BOOLEAN DEFAULT 0,  -- E07-F14: manual override flag
    progress_pct REAL DEFAULT 0.0,      -- Legacy completion %
    ...
);

-- tasks table (existing)
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    feature_id INTEGER NOT NULL,
    key TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,              -- Drives feature status
    blocked_reason TEXT,               -- Used for action items
    ...
);
```

**Key Insight:** All needed data exists. We just need better queries and calculations.

### 2.2 New Data Structures (In-Memory)

**FeatureStatusInfo** (returned by repository):

```go
// FeatureStatusInfo contains comprehensive status information for a feature
type FeatureStatusInfo struct {
    Feature         *models.Feature           // Base feature data
    StatusBreakdown map[string]int           // Status -> count
    Tasks           []*models.Task            // All tasks for action items
    Progress        *ProgressInfo             // Weighted progress calculation
    WorkSummary     *WorkSummary              // Work breakdown
    StatusContext   string                    // "active (waiting)", "active (blocked)"
    ActionItems     *ActionItems              // Tasks needing attention
}
```

**ProgressInfo** (calculated from config):

```go
// ProgressInfo provides weighted and completion progress metrics
type ProgressInfo struct {
    WeightedPct    float64  // Weighted progress (e.g., 68.0)
    CompletionPct  float64  // Traditional completion % (e.g., 40.0)
    WeightedRatio  string   // "3.4/5" (weighted tasks complete)
    CompletionRatio string  // "2/5" (completed tasks / total)
    TotalTasks     int      // Total task count
}
```

**WorkSummary** (calculated from config `responsibility`):

```go
// WorkSummary breaks down work by responsibility
type WorkSummary struct {
    TotalTasks     int  // Total tasks
    CompletedTasks int  // Fully completed
    AgentWork      int  // Tasks with responsibility="agent"
    HumanWork      int  // Tasks with responsibility="human" or "qa_team"
    BlockedWork    int  // Tasks with blocks_feature=true
    NotStarted     int  // Tasks with progress_weight=0.0 (excluding completed)
}
```

**ActionItems** (filtered tasks):

```go
// ActionItems contains tasks requiring immediate attention
type ActionItems struct {
    AwaitingApproval []*TaskActionItem  // ready_for_approval, age_days
    Blocked          []*TaskActionItem  // blocked status, reason
    InProgress       []*TaskActionItem  // in_progress status
}

// TaskActionItem represents a single actionable task
type TaskActionItem struct {
    TaskKey        string   // E07-F23-003
    Title          string   // Task title
    Status         string   // Current status
    AgeDays        *int     // Days in current status (for waiting tasks)
    BlockedReason  *string  // Reason for blocking
}
```

---

## 3. Repository Layer

### 3.1 New Repository Methods

**Location:** `internal/repository/feature_repository.go`

#### 3.1.1 GetTaskStatusBreakdown (Already Exists in E07-F14)

```go
// GetTaskStatusBreakdown retrieves the count of tasks by status for a feature
// Already implemented in E07-F14
func (r *FeatureRepository) GetTaskStatusBreakdown(ctx context.Context, featureID int64) (map[models.TaskStatus]int, error)
```

**SQL Query:**
```sql
SELECT status, COUNT(*) as count
FROM tasks
WHERE feature_id = ?
GROUP BY status
```

**Performance:**
- Indexed query: `idx_tasks_feature_id_status` (composite index)
- O(1) scan with GROUP BY
- **< 5ms typical**

#### 3.1.2 GetStatusInfo (NEW)

```go
// GetStatusInfo retrieves comprehensive status information for a feature
// Includes status breakdown, tasks, and calculated metrics
func (r *FeatureRepository) GetStatusInfo(ctx context.Context, featureID int64) (*FeatureStatusInfo, error) {
    // 1. Get feature
    feature, err := r.GetByID(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get feature: %w", err)
    }

    // 2. Get status breakdown
    statusBreakdown, err := r.GetTaskStatusBreakdown(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get status breakdown: %w", err)
    }

    // 3. Get all tasks (for action items)
    tasks, err := r.taskRepo.ListByFeature(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get tasks: %w", err)
    }

    return &FeatureStatusInfo{
        Feature:         feature,
        StatusBreakdown: statusBreakdown,
        Tasks:           tasks,
        // Progress, WorkSummary, StatusContext, ActionItems calculated by status package
    }, nil
}
```

**Dependencies:**
- Requires injected `TaskRepository` for task queries
- Constructor: `NewFeatureRepository(db *DB, taskRepo *TaskRepository)`

#### 3.1.3 GetFeatureStatusRollup (NEW - for epic get)

```go
// GetFeatureStatusRollup returns a rollup of feature statuses for an epic
// Used for epic get display
func (r *FeatureRepository) GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
    query := `
        SELECT status, COUNT(*) as count
        FROM features
        WHERE epic_id = ?
        GROUP BY status
    `

    rows, err := r.db.QueryContext(ctx, query, epicID)
    if err != nil {
        return nil, fmt.Errorf("failed to get feature status rollup: %w", err)
    }
    defer rows.Close()

    counts := make(map[string]int)
    for rows.Next() {
        var status string
        var count int
        if err := rows.Scan(&status, &count); err != nil {
            return nil, err
        }
        counts[status] = count
    }

    return counts, nil
}
```

#### 3.1.4 GetTasksWithActionItems (NEW)

```go
// GetTasksWithActionItems retrieves tasks that require immediate attention
// Filters by: ready_for_approval, blocked, in_progress
func (r *TaskRepository) GetTasksWithActionItems(ctx context.Context, featureID int64) ([]*models.Task, error) {
    query := `
        SELECT id, key, title, status, blocked_reason, created_at, updated_at
        FROM tasks
        WHERE feature_id = ?
          AND status IN ('ready_for_approval', 'blocked', 'in_progress')
        ORDER BY
            CASE
                WHEN status = 'blocked' THEN 1
                WHEN status = 'ready_for_approval' THEN 2
                ELSE 3
            END,
            created_at ASC
    `

    // Implementation similar to ListByFeature...
}
```

**Performance Optimization:**
- Uses composite index: `idx_tasks_feature_id_status`
- IN clause with 3 statuses is efficient
- Sorted by priority (blocked > awaiting approval > in progress)

### 3.2 Epic Repository Additions

**Location:** `internal/repository/epic_repository.go`

```go
// GetTaskStatusRollup returns task status counts across all features in an epic
func (r *EpicRepository) GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
    query := `
        SELECT t.status, COUNT(*) as count
        FROM tasks t
        JOIN features f ON t.feature_id = f.id
        WHERE f.epic_id = ?
        GROUP BY t.status
    `

    // Implementation similar to GetFeatureStatusRollup...
}

// GetFeaturesWithImpediments returns features with blockers or risks
func (r *EpicRepository) GetFeaturesWithImpediments(ctx context.Context, epicID int64) ([]*FeatureImpediment, error) {
    // Query features with blocked tasks or long-running approval tasks
    // Used for epic get "Impediments & Risks" section
}
```

---

## 4. Status Package (NEW)

**Location:** `internal/status/` (new package)

### 4.1 Package Structure

```
internal/status/
├── progress.go          # CalculateProgress()
├── work_breakdown.go    # CalculateWorkRemaining()
├── context.go          # GetStatusContext()
├── action_items.go     # GetActionItems()
├── types.go            # ProgressInfo, WorkSummary, ActionItems
└── status_test.go      # Comprehensive tests
```

### 4.2 Core Calculations

#### 4.2.1 CalculateProgress

```go
package status

import (
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

// CalculateProgress calculates weighted and completion progress from status counts
// Uses progress_weight from config to recognize partial completion
func CalculateProgress(statusCounts map[string]int, cfg *config.Config) *ProgressInfo {
    totalTasks := 0
    weightedProgress := 0.0
    completedTasks := 0

    for status, count := range statusCounts {
        totalTasks += count

        // Get metadata from config
        meta := cfg.GetStatusMetadata(status)
        if meta != nil {
            // Weighted progress (config-driven)
            weightedProgress += float64(count) * meta.ProgressWeight

            // Completion count (traditional)
            if meta.ProgressWeight >= 1.0 {
                completedTasks += count
            }
        }
    }

    if totalTasks == 0 {
        return &ProgressInfo{
            WeightedPct:     0.0,
            CompletionPct:   0.0,
            WeightedRatio:   "0/0",
            CompletionRatio: "0/0",
            TotalTasks:      0,
        }
    }

    weightedPct := (weightedProgress / float64(totalTasks)) * 100.0
    completionPct := (float64(completedTasks) / float64(totalTasks)) * 100.0

    return &ProgressInfo{
        WeightedPct:     weightedPct,
        CompletionPct:   completionPct,
        WeightedRatio:   fmt.Sprintf("%.1f/%d", weightedProgress, totalTasks),
        CompletionRatio: fmt.Sprintf("%d/%d", completedTasks, totalTasks),
        TotalTasks:      totalTasks,
    }
}
```

**Example:**
```go
// Given tasks:
// - 2 completed (progress_weight=1.0)
// - 1 ready_for_approval (progress_weight=0.9)
// - 1 in_development (progress_weight=0.5)
// - 1 draft (progress_weight=0.0)

statusCounts := map[string]int{
    "completed":          2,  // 2.0
    "ready_for_approval": 1,  // 0.9
    "in_development":     1,  // 0.5
    "draft":              1,  // 0.0
}

progress := CalculateProgress(statusCounts, cfg)
// WeightedPct: 68.0% = (2.0 + 0.9 + 0.5 + 0.0) / 5 * 100
// CompletionPct: 40.0% = 2 / 5 * 100
// WeightedRatio: "3.4/5"
// CompletionRatio: "2/5"
```

#### 4.2.2 CalculateWorkRemaining

```go
// CalculateWorkRemaining breaks down work by responsibility
// Uses responsibility and blocks_feature from config
func CalculateWorkRemaining(statusCounts map[string]int, cfg *config.Config) *WorkSummary {
    summary := &WorkSummary{}

    for status, count := range statusCounts {
        meta := cfg.GetStatusMetadata(status)
        if meta == nil {
            continue
        }

        // Count by responsibility
        switch meta.Responsibility {
        case "agent":
            summary.AgentWork += count
        case "human", "qa_team":
            summary.HumanWork += count
        case "none":
            // Check if this status blocks the feature
            if meta.BlocksFeature {
                summary.BlockedWork += count
            } else if meta.ProgressWeight == 0.0 {
                summary.NotStarted += count
            } else if meta.ProgressWeight >= 1.0 {
                summary.CompletedTasks += count
            }
        }

        summary.TotalTasks += count
    }

    return summary
}
```

**Example:**
```go
// Same tasks as above:
// - completed (responsibility="none", progress=1.0) → completed
// - ready_for_approval (responsibility="human") → human work
// - in_development (responsibility="agent") → agent work
// - draft (responsibility="none", progress=0.0) → not started

workSummary := CalculateWorkRemaining(statusCounts, cfg)
// TotalTasks: 5
// CompletedTasks: 2
// AgentWork: 1
// HumanWork: 1
// BlockedWork: 0
// NotStarted: 1
```

#### 4.2.3 GetStatusContext

```go
// GetStatusContext returns contextual status suffix for display
// Examples: "active (waiting)", "active (blocked)", "active (development)"
func GetStatusContext(feature *models.Feature, statusCounts map[string]int, cfg *config.Config) string {
    if feature.Status != "active" {
        return string(feature.Status)
    }

    // Check for waiting (tasks awaiting approval)
    if statusCounts["ready_for_approval"] > 0 || statusCounts["ready_for_code_review"] > 0 {
        return "active (waiting)"
    }

    // Check for blocked
    if statusCounts["blocked"] > 0 {
        return "active (blocked)"
    }

    // Check for development
    if statusCounts["in_development"] > 0 {
        return "active (development)"
    }

    // Default active
    return "active"
}
```

#### 4.2.4 GetActionItems

```go
// GetActionItems categorizes tasks requiring immediate attention
func GetActionItems(tasks []*models.Task, cfg *config.Config) *ActionItems {
    items := &ActionItems{
        AwaitingApproval: []*TaskActionItem{},
        Blocked:          []*TaskActionItem{},
        InProgress:       []*TaskActionItem{},
    }

    now := time.Now()

    for _, task := range tasks {
        meta := cfg.GetStatusMetadata(string(task.Status))
        if meta == nil {
            continue
        }

        switch {
        case meta.Responsibility == "human" && task.Status == "ready_for_approval":
            // Calculate age in days
            ageDays := int(now.Sub(task.UpdatedAt).Hours() / 24)
            items.AwaitingApproval = append(items.AwaitingApproval, &TaskActionItem{
                TaskKey:  task.Key,
                Title:    task.Title,
                Status:   string(task.Status),
                AgeDays:  &ageDays,
            })

        case meta.BlocksFeature:
            items.Blocked = append(items.Blocked, &TaskActionItem{
                TaskKey:       task.Key,
                Title:         task.Title,
                Status:        string(task.Status),
                BlockedReason: task.BlockedReason,
            })

        case task.Status == "in_development" || task.Status == "in_progress":
            items.InProgress = append(items.InProgress, &TaskActionItem{
                TaskKey: task.Key,
                Title:   task.Title,
                Status:  string(task.Status),
            })
        }
    }

    return items
}
```

---

## 5. CLI Enhancements

### 5.1 Feature Get Command

**Location:** `internal/cli/commands/feature.go`

**Enhanced Output:**

```
Feature: E07-F23 - Enhanced Status Tracking
Status: active (waiting) ⏳
Created: 2026-01-16 | Updated: 2026-01-16 14:30

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Progress Breakdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Overall: 68% (3.4/5 tasks)
  • Completed: 2 tasks (40%)
  • Ready for Approval: 1 task (18%) ⏳
  • In Development: 1 task (10%)
  • Draft: 1 task (0%)

Progress Bar:
[████████████████████░░░░░░░░] 68% Weighted Progress
[████████████░░░░░░░░░░░░░░░░] 40% Completed

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Action Items (What needs your attention)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏳ Waiting for Approval (1):
  • E07-F23-003 - Add status breakdown display
    Agent work complete, awaiting your review

    👉 Review: shark task approve E07-F23-003

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Work Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 5 tasks
Remaining: 3 tasks (60%)

Breakdown:
  ✅ Completed: 2 tasks
  🏃 Agent Work: 1 task (in progress)
  ⏳ Human Work: 1 task (awaiting approval)
  📋 Not Started: 1 task

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Tasks (5)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏳ Approval Phase:
  E07-F23-003  Add status breakdown display    ready_for_approval

🏃 Development Phase:
  E07-F23-002  Implement work breakdown         in_development

✅ Completed:
  E07-F23-001  Create config metadata           completed
  E07-F23-004  Add unit tests                   completed

📋 Planning:
  E07-F23-005  Update documentation             draft
```

**Implementation:**

```go
func runFeatureGetCommand(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Get database and config
    repoDb, err := cli.GetDB(ctx)
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }

    cfg, err := config.LoadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Initialize repositories
    taskRepo := repository.NewTaskRepository(repoDb)
    featureRepo := repository.NewFeatureRepository(repoDb, taskRepo)

    // Get feature status info
    statusInfo, err := featureRepo.GetStatusInfo(ctx, featureID)
    if err != nil {
        return err
    }

    // Calculate displays using status package
    progress := status.CalculateProgress(statusInfo.StatusBreakdown, cfg)
    workSummary := status.CalculateWorkRemaining(statusInfo.StatusBreakdown, cfg)
    statusContext := status.GetStatusContext(statusInfo.Feature, statusInfo.StatusBreakdown, cfg)
    actionItems := status.GetActionItems(statusInfo.Tasks, cfg)

    // Output JSON or table
    if cli.GlobalConfig.JSON {
        return outputFeatureJSON(statusInfo, progress, workSummary, statusContext, actionItems)
    }

    return outputFeatureTable(statusInfo, progress, workSummary, statusContext, actionItems)
}
```

### 5.2 Feature List Command

**Enhanced Output:**

```
FEATURES (E07)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
KEY      TITLE                    HEALTH  PROGRESS              NOTES
E07-F22  Rejection Reasons        🟡      45% (2.2/5)          [3 ready, 2 blocked]
E07-F23  Enhanced Status          🟡      68% (3.4/5)          [1 waiting approval]
E07-F24  Next Feature             🟢      0% (0/8)             [all todo]
```

**Health Indicators:**
- 🟢 Healthy: No blockers, on track
- 🟡 Attention: 1-2 blockers OR tasks awaiting approval > 7 days
- 🔴 At Risk: 3+ blockers OR >30% tasks blocked

**Implementation:**

```go
func runFeatureListCommand(cmd *cobra.Command, args []string) error {
    // ... get features ...

    for _, feature := range features {
        // Get status breakdown
        statusBreakdown, _ := featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)

        // Calculate health
        health := calculateHealthIndicator(statusBreakdown, cfg)

        // Calculate notes
        notes := generateNotesColumn(statusBreakdown)

        // Calculate progress
        progress := status.CalculateProgress(statusBreakdown, cfg)

        // Add to table row
        rows = append(rows, []string{
            feature.Key,
            feature.Title,
            health,
            fmt.Sprintf("%d%% (%.1f/%d)", progress.WeightedPct, progress.WeightedRatio),
            notes,
        })
    }

    // Output table
}
```

### 5.3 Epic Get Command

**Enhanced Output:**

```
Epic: E07 - Shark Enhancements
Status: active (calculated)
Overall Progress: 60% (12 of 20 features complete)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Feature Status Summary (20 features)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Planning:    3 features [draft: 3]
Development: 8 features [active: 6, blocked: 2]
Review:      4 features [active: 2, waiting: 2]
Done:       12 features [completed: 12]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Task Rollup (250 tasks across all features)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✅ Completed: 150 (60%)
  🏃 In Progress: 45 (18%)
  ⏳ Awaiting Approval: 20 (8%)
  🚫 Blocked: 15 (6%)
  📋 To Do: 20 (8%)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️ Impediments & Risks
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚫 Blocked Features (2):
  • E07-F05: 3 tasks blocked (waiting on API design)
  • E07-F12: 2 tasks blocked (dependency not ready)

⏳ Approval Backlog (2 features):
  • E07-F22: 3 tasks awaiting approval (age: 5 days)
  • E07-F23: 1 task awaiting approval (age: 2 days)
```

---

## 6. JSON API Response Format

### 6.1 Enhanced Feature Response

```json
{
  "id": 23,
  "key": "E07-F23",
  "title": "Enhanced Status Tracking",
  "status": "active",
  "status_context": "waiting",
  "status_explanation": "1 task awaiting approval",

  "progress": {
    "weighted_pct": 68.0,
    "completion_pct": 40.0,
    "weighted_ratio": "3.4/5",
    "completion_ratio": "2/5",
    "total_tasks": 5
  },

  "work_summary": {
    "total_tasks": 5,
    "completed_tasks": 2,
    "work_remaining": 3,
    "agent_work": 1,
    "human_work": 1,
    "blocked_work": 0,
    "not_started": 1
  },

  "status_breakdown": {
    "completed": 2,
    "ready_for_approval": 1,
    "in_development": 1,
    "draft": 1
  },

  "status_breakdown_by_phase": {
    "approval": {
      "statuses": [
        {"status": "ready_for_approval", "count": 1, "color": "purple"}
      ],
      "total": 1
    },
    "development": {
      "statuses": [
        {"status": "in_development", "count": 1, "color": "yellow"}
      ],
      "total": 1
    },
    "done": {
      "statuses": [
        {"status": "completed", "count": 2, "color": "white"}
      ],
      "total": 2
    },
    "planning": {
      "statuses": [
        {"status": "draft", "count": 1, "color": "gray"}
      ],
      "total": 1
    }
  },

  "action_items": {
    "awaiting_approval": [
      {
        "task_key": "E07-F23-003",
        "title": "Add status breakdown display",
        "status": "ready_for_approval",
        "age_days": 2
      }
    ],
    "blocked": [],
    "in_progress": [
      {
        "task_key": "E07-F23-002",
        "title": "Implement work breakdown",
        "status": "in_development"
      }
    ]
  },

  "health": {
    "indicator": "attention",
    "level": "yellow",
    "reasons": [
      "1 task awaiting approval"
    ]
  }
}
```

### 6.2 Epic Response with Rollup

```json
{
  "id": 7,
  "key": "E07",
  "title": "Shark Enhancements",
  "status": "active",
  "progress_pct": 60.0,

  "feature_rollup": {
    "total_features": 20,
    "by_status": {
      "draft": 3,
      "active": 6,
      "blocked": 2,
      "waiting": 2,
      "completed": 12
    },
    "by_phase": {
      "planning": 3,
      "development": 8,
      "review": 4,
      "done": 12
    }
  },

  "task_rollup": {
    "total_tasks": 250,
    "completed": 150,
    "in_progress": 45,
    "awaiting_approval": 20,
    "blocked": 15,
    "todo": 20
  },

  "impediments": {
    "blocked_features": [
      {
        "feature_key": "E07-F05",
        "blocked_tasks": 3,
        "reason": "waiting on API design"
      }
    ],
    "approval_backlog": [
      {
        "feature_key": "E07-F22",
        "tasks_awaiting_approval": 3,
        "age_days": 5
      }
    ]
  }
}
```

---

## 7. Performance Considerations

### 7.1 Query Optimization

**Problem:** N+1 queries when listing features with status info.

**Solution:** Batch queries with single JOIN.

```sql
-- BAD: N+1 queries
SELECT * FROM features WHERE epic_id = ?;
-- Then for each feature:
SELECT status, COUNT(*) FROM tasks WHERE feature_id = ? GROUP BY status;

-- GOOD: Single query with JOIN
SELECT
    f.id, f.key, f.title, f.status,
    t.status as task_status,
    COUNT(t.id) as task_count
FROM features f
LEFT JOIN tasks t ON f.id = t.feature_id
WHERE f.epic_id = ?
GROUP BY f.id, t.status
ORDER BY f.execution_order;
```

**Performance:**
- **Before:** 1 + N queries (21 queries for 20 features)
- **After:** 1 query
- **Latency:** < 20ms for 20 features with 250 tasks

### 7.2 Caching Strategy

**Not needed for v1** - calculations are fast enough.

**Future optimization (if needed):**
- Cache FeatureStatusInfo for 5 seconds (stale-while-revalidate)
- Invalidate on task status change
- Use Redis or in-memory LRU cache

### 7.3 Index Requirements

**Existing indexes (already present):**
```sql
CREATE INDEX idx_tasks_feature_id ON tasks(feature_id);
CREATE INDEX idx_tasks_feature_id_status ON tasks(feature_id, status);
CREATE INDEX idx_features_epic_id ON features(epic_id);
```

**NO new indexes needed.**

### 7.4 Performance Targets

| Operation | Target (p95) | Typical |
|-----------|--------------|---------|
| Feature Get (single) | < 100ms | ~30ms |
| Feature List (20) | < 200ms | ~50ms |
| Epic Get (rollup) | < 500ms | ~150ms |
| Status calculation | < 1ms | ~0.1ms |

---

## 8. Testing Strategy

### 8.1 Unit Tests

**Status Package Tests:**
```go
// internal/status/progress_test.go
func TestCalculateProgress(t *testing.T) {
    tests := []struct {
        name          string
        statusCounts  map[string]int
        config        *config.Config
        wantWeighted  float64
        wantCompleted float64
    }{
        {
            name: "all completed",
            statusCounts: map[string]int{"completed": 5},
            wantWeighted: 100.0,
            wantCompleted: 100.0,
        },
        {
            name: "mixed statuses with weights",
            statusCounts: map[string]int{
                "completed": 2,           // 2.0
                "ready_for_approval": 1,  // 0.9
                "in_development": 1,      // 0.5
                "draft": 1,               // 0.0
            },
            wantWeighted: 68.0,   // (2.0 + 0.9 + 0.5 + 0.0) / 5 * 100
            wantCompleted: 40.0,  // 2 / 5 * 100
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            progress := CalculateProgress(tt.statusCounts, tt.config)
            assert.Equal(t, tt.wantWeighted, progress.WeightedPct)
            assert.Equal(t, tt.wantCompleted, progress.CompletionPct)
        })
    }
}
```

### 8.2 Repository Tests

```go
// internal/repository/feature_repository_test.go
func TestGetStatusInfo(t *testing.T) {
    ctx := context.Background()
    db := test.GetTestDB()
    repo := repository.NewFeatureRepository(db, taskRepo)

    // Seed test data
    epicID, featureID := test.SeedTestData()
    test.SeedTasks(featureID, []string{"completed", "ready_for_approval", "in_development"})

    // Get status info
    info, err := repo.GetStatusInfo(ctx, featureID)
    require.NoError(t, err)

    // Verify
    assert.Equal(t, 3, len(info.StatusBreakdown))
    assert.Equal(t, 1, info.StatusBreakdown["completed"])
    assert.NotNil(t, info.Tasks)
}
```

### 8.3 CLI Integration Tests

```go
// internal/cli/commands/feature_test.go
func TestFeatureGetCommandEnhanced(t *testing.T) {
    // Mock repository
    mockRepo := &MockFeatureRepository{
        GetStatusInfoFunc: func(ctx context.Context, id int64) (*FeatureStatusInfo, error) {
            return &FeatureStatusInfo{
                StatusBreakdown: map[string]int{
                    "completed": 2,
                    "ready_for_approval": 1,
                },
            }, nil
        },
    }

    // Run command
    cmd := buildFeatureGetCommand(mockRepo)
    output := captureOutput(cmd, []string{"E07-F23"})

    // Verify output
    assert.Contains(t, output, "Progress Breakdown")
    assert.Contains(t, output, "68%")
    assert.Contains(t, output, "Action Items")
}
```

### 8.4 End-to-End Tests

```bash
# Test feature get
./bin/shark feature get E07-F23 --json | jq '.progress.weighted_pct'
# Expected: 68.0

# Test feature list
./bin/shark feature list E07 | grep "E07-F23"
# Expected: Shows health indicator and notes

# Test epic get
./bin/shark epic get E07 --json | jq '.task_rollup.total_tasks'
# Expected: 250
```

---

## 9. Error Handling

### 9.1 Missing Config Metadata

**Scenario:** Status exists in DB but not in config.

**Handling:**
```go
meta := cfg.GetStatusMetadata(status)
if meta == nil {
    // Default values
    meta = &config.StatusMetadata{
        ProgressWeight: 0.0,
        Responsibility: "none",
        BlocksFeature:  false,
    }
}
```

**Impact:** Graceful degradation - missing statuses treated as 0% progress.

### 9.2 Empty Task List

**Scenario:** Feature has no tasks.

**Handling:**
```go
if len(statusInfo.StatusBreakdown) == 0 {
    return &ProgressInfo{
        WeightedPct:     0.0,
        CompletionPct:   0.0,
        WeightedRatio:   "0/0",
        CompletionRatio: "0/0",
        TotalTasks:      0,
    }
}
```

**Display:** "No tasks" message instead of progress breakdown.

### 9.3 Invalid Progress Weights

**Scenario:** Config has `progress_weight` > 1.0 or < 0.0.

**Handling:**
```go
func validateStatusMetadata(meta *StatusMetadata) error {
    if meta.ProgressWeight < 0.0 || meta.ProgressWeight > 1.0 {
        return fmt.Errorf("progress_weight must be between 0.0 and 1.0, got %.2f", meta.ProgressWeight)
    }
    return nil
}
```

**When:** Config load time (fail fast).

---

## 10. Migration & Backwards Compatibility

### 10.1 No Breaking Changes

- Existing commands work unchanged
- JSON API is additive (new fields, no removals)
- Old configs work (missing fields default gracefully)

### 10.2 Feature Flag

**Not needed** - all enhancements are opt-in by using new display sections.

**Old behavior:** Still available via `--json` without new fields.

**New behavior:** Displayed automatically in table output.

---

## 11. Dependencies

### 11.1 Required

- **E07-F14 (Cascading Status Calculation):** Provides status calculation and config metadata
  - Required fields: `progress_weight`, `responsibility`, `blocks_feature`
  - Already implemented

### 11.2 Optional

- **E07-F22 (Rejection Reason):** Rejection reasons can be shown in action items
  - If available: Show rejection reason in reopened tasks
  - If not: Show generic "reopened" message

---

## 12. Security Considerations

### 12.1 SQL Injection

**Protection:** Parameterized queries everywhere.

```go
// GOOD
query := "SELECT status, COUNT(*) FROM tasks WHERE feature_id = ? GROUP BY status"
rows, err := db.QueryContext(ctx, query, featureID)

// BAD (never do this)
query := fmt.Sprintf("SELECT * FROM tasks WHERE feature_id = %d", featureID)
```

### 12.2 Information Disclosure

**Risk:** Expose blocked reasons that may contain sensitive info.

**Mitigation:**
- Blocked reasons are already stored in DB (existing risk)
- No new exposure - same as `shark task get`
- Follow existing access control patterns

---

## 13. Monitoring & Observability

### 13.1 Metrics to Track (Future)

- Feature get latency (p50, p95, p99)
- Status calculation time
- Cache hit rate (if caching added)
- Query count per command

### 13.2 Logging

**Log at DEBUG level:**
```go
logger.Debug("Calculating progress",
    "featureID", featureID,
    "taskCount", len(statusInfo.StatusBreakdown),
    "weightedPct", progress.WeightedPct,
)
```

**No logging at INFO level** (performance sensitive).

---

## 14. Future Enhancements

### 14.1 Phase 2: Real-Time Updates

- WebSocket API for live status updates
- Push notifications when task needs approval
- Dashboard with live progress bars

### 14.2 Phase 3: Advanced Analytics

- Velocity metrics (tasks completed per week)
- Cycle time (time in each status)
- Bottleneck identification (slowest phases)

### 14.3 Phase 4: Predictive Analytics

- Estimate completion date based on velocity
- Risk prediction (likely to miss deadline)
- Resource allocation recommendations

---

## 15. Acceptance Criteria

### 15.1 Feature Get Display

- [ ] Progress breakdown shows weighted and completion %
- [ ] Progress shows ratio: "68% (3.4/5)"
- [ ] Action items section shows tasks awaiting approval
- [ ] Work summary shows agent/human/blocked/not started
- [ ] Status context shown: "active (waiting)"
- [ ] All sections render correctly in table mode
- [ ] JSON output includes all new fields

### 15.2 Feature List Display

- [ ] Health indicators: 🟢 🟡 🔴
- [ ] Notes column: "[3 ready, 2 blocked]"
- [ ] Progress bar with weighted progress
- [ ] Can filter: `--awaiting-approval`, `--with-blockers`

### 15.3 Epic Get Display

- [ ] Feature status rollup by phase
- [ ] Task rollup across all features
- [ ] Impediments section with blocked features
- [ ] Approval backlog shown with age

### 15.4 Performance

- [ ] Feature get < 100ms (p95)
- [ ] Feature list < 200ms for 20 features (p95)
- [ ] Epic get < 500ms (p95)
- [ ] All queries use indexes

### 15.5 Testing

- [ ] Unit tests for all status calculations
- [ ] Repository tests for all new methods
- [ ] CLI tests with mocked repositories
- [ ] End-to-end tests for display formatting

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
**Status:** Ready for Implementation
