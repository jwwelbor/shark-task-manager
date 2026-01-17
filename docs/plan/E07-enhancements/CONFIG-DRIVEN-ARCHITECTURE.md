# Config-Driven Architecture: E07-F14 + E07-F22

**Date:** 2026-01-16
**Author:** Architect Agent
**Status:** Approved Design

---

## Executive Summary

This document explains the **unified config-driven architecture** for E07-F14 (Cascading Status Calculation) and E07-F22 (Rejection Reason for Status Transitions).

**Key Innovation:** All status semantics live in config. Code never mentions specific status names—it only queries config for semantic meaning.

---

## The Problem We Solved

### Before (Hardcoded Logic)

```go
// BAD - hardcoded status names everywhere
if task.Status == "ready_for_approval" {
    agentCompleteCount++
}

if task.Status == "blocked" {
    feature.Status = "blocked"
}

if task.Status == "in_development" || task.Status == "in_qa" {
    activeCount++
}

// E07-F22: Hardcoded rejection list
backwardTransitions := map[string][]string{
    "in_development": {"ready_for_code_review", "ready_for_qa", "ready_for_approval"},
    // ... 20 more lines of hardcoded transitions
}
```

**Problems:**
- ❌ Status meanings scattered across codebase
- ❌ Adding new workflow requires code changes
- ❌ Different teams can't customize workflows
- ❌ Hard to maintain and test

### After (Config-Driven)

```go
// GOOD - config-driven
meta := cfg.GetStatusMetadata(task.Status)

if meta.Responsibility == "human" {
    humanWorkCount++
}

if meta.BlocksFeature {
    feature.Status = "blocked"
}

if meta.ProgressWeight > 0.0 && meta.ProgressWeight < 1.0 {
    activeCount++
}

// E07-F22: Automatic rejection detection
if cfg.IsBackwardTransition(oldStatus, newStatus) {
    // Automatically detects progress decrease
}
```

**Benefits:**
- ✅ Status semantics in ONE place (config)
- ✅ Add new workflows with config changes only
- ✅ Per-project workflow customization
- ✅ Easy to maintain and test

---

## Enhanced Config Structure

### Complete Example

```json
{
  "status_metadata": {
    "draft": {
      "color": "gray",
      "phase": "planning",
      "description": "Task created but not yet refined",
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": false
    },
    "ready_for_development": {
      "color": "yellow",
      "phase": "development",
      "description": "Spec complete, ready for implementation",
      "agent_types": ["developer", "ai-coder"],
      "progress_weight": 0.0,
      "responsibility": "agent",
      "blocks_feature": false
    },
    "in_development": {
      "color": "yellow",
      "phase": "development",
      "description": "Code implementation in progress",
      "agent_types": ["developer", "ai-coder"],
      "progress_weight": 0.5,
      "responsibility": "agent",
      "blocks_feature": false
    },
    "ready_for_code_review": {
      "color": "magenta",
      "phase": "review",
      "description": "Code complete, awaiting review",
      "agent_types": ["tech-lead", "code-reviewer"],
      "progress_weight": 0.85,
      "responsibility": "human",
      "blocks_feature": false
    },
    "in_code_review": {
      "color": "magenta",
      "phase": "review",
      "description": "Under code review",
      "progress_weight": 0.85,
      "responsibility": "human",
      "blocks_feature": false
    },
    "ready_for_qa": {
      "color": "green",
      "phase": "qa",
      "description": "Ready for quality assurance testing",
      "agent_types": ["qa", "test-engineer"],
      "progress_weight": 0.80,
      "responsibility": "qa_team",
      "blocks_feature": false
    },
    "in_qa": {
      "color": "green",
      "phase": "qa",
      "description": "Being tested",
      "progress_weight": 0.80,
      "responsibility": "qa_team",
      "blocks_feature": false
    },
    "ready_for_approval": {
      "color": "purple",
      "phase": "approval",
      "description": "Awaiting final approval",
      "agent_types": ["product-manager", "client"],
      "progress_weight": 0.9,
      "responsibility": "human",
      "blocks_feature": false
    },
    "in_approval": {
      "color": "purple",
      "phase": "approval",
      "description": "Under final review",
      "progress_weight": 0.9,
      "responsibility": "human",
      "blocks_feature": false
    },
    "completed": {
      "color": "white",
      "phase": "done",
      "description": "Task finished and approved",
      "progress_weight": 1.0,
      "responsibility": "none",
      "blocks_feature": false
    },
    "blocked": {
      "color": "red",
      "phase": "any",
      "description": "Temporarily blocked by external dependency",
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": true
    },
    "on_hold": {
      "color": "orange",
      "phase": "any",
      "description": "Intentionally paused",
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": false
    },
    "cancelled": {
      "color": "gray",
      "phase": "done",
      "description": "Task abandoned or deprecated",
      "progress_weight": 1.0,
      "responsibility": "none",
      "blocks_feature": false
    }
  },

  "special_statuses": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"],
    "_agent_complete_": ["ready_for_approval", "ready_for_qa", "ready_for_code_review"],
    "_human_work_": ["in_approval", "ready_for_approval", "in_code_review", "ready_for_code_review", "in_qa", "ready_for_qa"],
    "_blocking_": ["blocked"],
    "_paused_": ["on_hold"]
  },

  "require_rejection_reason": true,

  "workflow": {
    "enable_auto_status_calculation": true,
    "cascade_to_epic": true,
    "cascade_to_feature": true
  }
}
```

### Config Field Reference

| Field | Type | Purpose | Used By |
|-------|------|---------|---------|
| `progress_weight` | float (0.0-1.0) | How much this status contributes to progress | E07-F14 progress calc, E07-F22 rejection detection |
| `responsibility` | string | Who's responsible (`agent`, `human`, `qa_team`, `none`) | E07-F14 work breakdown |
| `blocks_feature` | boolean | Should this status make feature blocked? | E07-F14 status calculation |
| `require_rejection_reason` | boolean | Require reason when progress decreases | E07-F22 validation |

---

## How E07-F14 Uses Config

### Feature Status Calculation

**Code:**
```go
func DeriveFeatureStatus(statusCounts map[string]int, cfg *config.Config) string {
    totalTasks := 0
    for _, count := range statusCounts {
        totalTasks += count
    }

    if totalTasks == 0 {
        return "draft"
    }

    // All complete? (from config)
    completeCount := 0
    for _, completeStatus := range cfg.SpecialStatuses["_complete_"] {
        completeCount += statusCounts[completeStatus]
    }
    if completeCount == totalTasks {
        return "completed"
    }

    // Any blocking tasks? (from config)
    for status, count := range statusCounts {
        if count > 0 {
            meta := cfg.GetStatusMetadata(status)
            if meta != nil && meta.BlocksFeature {
                return "blocked"
            }
        }
    }

    // Any paused? (from config)
    pausedCount := 0
    for _, pausedStatus := range cfg.SpecialStatuses["_paused_"] {
        pausedCount += statusCounts[pausedStatus]
    }
    if pausedCount == totalTasks {
        return "on_hold"
    }

    // All not started? (from config)
    notStartedCount := 0
    for _, startStatus := range cfg.SpecialStatuses["_start_"] {
        notStartedCount += statusCounts[startStatus]
    }
    if notStartedCount == totalTasks {
        return "draft"
    }

    return "active"
}
```

**Key Point:** Code never mentions "blocked", "completed", etc. by name. It queries config!

### Progress Calculation

**Code:**
```go
func CalculateProgress(statusCounts map[string]int, cfg *config.Config) float64 {
    totalTasks := 0
    weightedProgress := 0.0

    for status, count := range statusCounts {
        totalTasks += count

        // Get weight from config
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
```

**Example:**
```
5 tasks:
- 2 completed (weight 1.0) = 2.0
- 1 ready_for_approval (weight 0.9) = 0.9  ← Agent done!
- 1 in_development (weight 0.5) = 0.5
- 1 draft (weight 0.0) = 0.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 3.4 / 5 = 68% progress
```

### Work Breakdown by Responsibility

**Code:**
```go
func CalculateWorkRemaining(statusCounts map[string]int, cfg *config.Config) WorkSummary {
    summary := WorkSummary{}

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
```

**Output:**
```json
{
  "total_tasks": 25,
  "work_remaining": 13,
  "agent_work": 5,      // "responsibility": "agent"
  "human_work": 4,      // "responsibility": "human" or "qa_team"
  "blocked_work": 2,    // "blocks_feature": true
  "not_started": 2      // progress_weight = 0.0
}
```

---

## How E07-F22 Uses Config

### Automatic Backward Transition Detection

**Code:**
```go
// Config provides this method
func (c *Config) IsBackwardTransition(oldStatus, newStatus string) bool {
    oldMeta := c.GetStatusMetadata(oldStatus)
    newMeta := c.GetStatusMetadata(newStatus)

    if oldMeta == nil || newMeta == nil {
        return false
    }

    // If progress_weight decreases, it's backward!
    return newMeta.ProgressWeight < oldMeta.ProgressWeight
}
```

**Examples:**
```
ready_for_code_review (0.85) → in_development (0.50) = BACKWARD (requires rejection reason)
in_development (0.50) → ready_for_code_review (0.85) = FORWARD (no rejection reason)
ready_for_approval (0.90) → in_qa (0.80) = BACKWARD (requires rejection reason)
in_qa (0.80) → ready_for_approval (0.90) = FORWARD (no rejection reason)
```

### Repository Validation

**Code:**
```go
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
            return &RejectionReasonRequiredError{
                FromStatus:   task.Status,
                ToStatus:     newStatus,
                FromProgress: oldMeta.ProgressWeight * 100,
                ToProgress:   newMeta.ProgressWeight * 100,
            }
        }
    }

    // ... rest of update logic
}
```

### CLI Error Messages

```bash
$ shark task update E07-F22-001 --status=in_development
Error: rejection reason required: transitioning from ready_for_code_review (85%) to in_development (50%) decreases progress

Please provide a reason using --rejection-reason flag:
  shark task update E07-F22-001 --status=in_development --rejection-reason="<reason>"
```

---

## Benefits of Config-Driven Approach

### 1. No Hardcoded Status Logic

**Before:** 50+ places in code checking specific status names
**After:** 0 places checking specific status names

All logic queries config for semantic meaning.

### 2. Easy Workflow Customization

**Add a new status:**
```json
{
  "status_metadata": {
    "waiting_for_dependencies": {
      "color": "orange",
      "phase": "development",
      "description": "Waiting for external dependencies",
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": true
    }
  }
}
```

**Code automatically handles it** - no code changes needed!

### 3. Automatic Rejection Detection

**No hardcoded transition lists:**
```go
// BAD - hardcoded (v1 approach)
backwardTransitions := map[string][]string{
    "in_development": {"ready_for_code_review", "ready_for_qa", "ready_for_approval"},
    "in_qa": {"ready_for_approval"},
    "in_code_review": {"ready_for_qa", "ready_for_approval"},
}

// GOOD - config-driven (v2 approach)
if cfg.IsBackwardTransition(oldStatus, newStatus) {
    // Automatically detects based on progress_weight
}
```

### 4. Progress Recognition Before Completion

**Task at `ready_for_approval` (90% complete):**
- Agent work: DONE ✅
- Human work: PENDING ⏳
- Progress: 90% (not 0%!)

This recognizes agent work is complete even though task isn't approved yet.

### 5. Per-Project Customization

**Different teams can have different configs:**

**Team A (strict workflow):**
```json
{
  "require_rejection_reason": true,
  "status_metadata": {
    "in_development": {"progress_weight": 0.3},  // Lower weight = more gates
    "ready_for_code_review": {"progress_weight": 0.7}
  }
}
```

**Team B (lightweight workflow):**
```json
{
  "require_rejection_reason": false,
  "status_metadata": {
    "in_development": {"progress_weight": 0.5},  // Higher weight = fewer gates
    "ready_for_code_review": {"progress_weight": 0.9}
  }
}
```

---

## Implementation Simplicity

### No Schema Changes

**E07-F14:**
- ✅ Feature/epic status already exists in database
- ❌ No `status_override` column needed
- ✅ Status always calculated from tasks

**E07-F22:**
- ✅ task_history table already exists
- ✅ Just add `rejection_reason` column
- ✅ Index on rejection_reason for filtering

### Minimal Code Changes

**E07-F14:**
- Add config fields (1 file: config.go)
- Implement calculation functions (1 file: derivation.go)
- Update repositories to call calculation (2 files)
- Total: ~4 files changed

**E07-F22:**
- Add IsBackwardTransition method (1 file: config.go)
- Add validation to UpdateStatus (1 file: task_repository.go)
- Add CLI flag (1 file: task_update.go)
- Total: ~3 files changed

### High Maintainability

**Adding new workflow:**
1. Update config.json (1 file)
2. Done!

**No code changes needed** for new statuses.

---

## Example Workflows

### Workflow 1: Simple (3 Statuses)

```json
{
  "status_metadata": {
    "todo": {
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": false
    },
    "in_progress": {
      "progress_weight": 0.5,
      "responsibility": "agent",
      "blocks_feature": false
    },
    "done": {
      "progress_weight": 1.0,
      "responsibility": "none",
      "blocks_feature": false
    }
  },
  "require_rejection_reason": true
}
```

**Backward transitions:**
- `done` (1.0) → `in_progress` (0.5) = requires reason
- `in_progress` (0.5) → `todo` (0.0) = requires reason

### Workflow 2: Detailed (12 Statuses)

```json
{
  "status_metadata": {
    "draft": {"progress_weight": 0.0},
    "ready_for_refinement": {"progress_weight": 0.0},
    "in_refinement": {"progress_weight": 0.1},
    "ready_for_development": {"progress_weight": 0.2},
    "in_development": {"progress_weight": 0.5},
    "ready_for_code_review": {"progress_weight": 0.85},
    "in_code_review": {"progress_weight": 0.85},
    "ready_for_qa": {"progress_weight": 0.80},
    "in_qa": {"progress_weight": 0.80},
    "ready_for_approval": {"progress_weight": 0.9},
    "in_approval": {"progress_weight": 0.9},
    "completed": {"progress_weight": 1.0}
  }
}
```

**All backward transitions automatically detected based on weight decrease.**

---

## Testing Strategy

### Config-Driven Tests

```go
func TestFeatureStatus_ConfigDriven(t *testing.T) {
    cfg := &config.Config{
        StatusMetadata: map[string]config.StatusMetadata{
            "in_development": {ProgressWeight: 0.5, BlocksFeature: false},
            "blocked": {ProgressWeight: 0.0, BlocksFeature: true},
            "completed": {ProgressWeight: 1.0, BlocksFeature: false},
        },
        SpecialStatuses: map[string][]string{
            "_complete_": {"completed"},
            "_blocking_": {"blocked"},
        },
    }

    tests := []struct {
        name           string
        statusCounts   map[string]int
        expectedStatus string
        expectedProgress float64
    }{
        {
            name: "blocked task makes feature blocked",
            statusCounts: map[string]int{
                "in_development": 3,
                "blocked": 1,
            },
            expectedStatus: "blocked",
            expectedProgress: 37.5,  // (3*0.5 + 1*0.0) / 4 * 100
        },
        {
            name: "all completed",
            statusCounts: map[string]int{
                "completed": 5,
            },
            expectedStatus: "completed",
            expectedProgress: 100.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test status
            status := status.DeriveFeatureStatus(tt.statusCounts, cfg)
            if status != tt.expectedStatus {
                t.Errorf("status: expected %s, got %s", tt.expectedStatus, status)
            }

            // Test progress
            progress := status.CalculateProgress(tt.statusCounts, cfg)
            if progress != tt.expectedProgress {
                t.Errorf("progress: expected %.1f, got %.1f", tt.expectedProgress, progress)
            }
        })
    }
}
```

**Key Point:** Tests use config object, not hardcoded statuses!

---

## Migration Path

### Phase 1: Update Config

```bash
# Backup current config
cp .sharkconfig.json .sharkconfig.json.backup

# Add new fields with defaults
cat >> .sharkconfig.json <<EOF
{
  "require_rejection_reason": false,
  "workflow": {
    "enable_auto_status_calculation": true,
    "cascade_to_epic": true,
    "cascade_to_feature": true
  },
  "status_metadata": {
    "draft": {
      ...,
      "progress_weight": 0.0,
      "responsibility": "none",
      "blocks_feature": false
    },
    ...
  }
}
EOF
```

### Phase 2: Deploy Code

- No database migrations needed (status already exists)
- Deploy E07-F14 + E07-F22 code
- Backward compatible (defaults preserve current behavior)

### Phase 3: Enable Features

```bash
# Enable rejection reason requirement
shark config set require_rejection_reason true

# Test it
shark task update E07-F22-001 --status=in_development
# Error: rejection reason required

shark task update E07-F22-001 --status=in_development \
  --rejection-reason="Failed code review"
# Success!
```

---

## Summary

### What We Built

1. **E07-F14 (Cascading Status Calculation)**
   - Feature/epic status calculated from tasks
   - Progress weighted by `progress_weight`
   - Work breakdown by `responsibility`
   - Blocking detection by `blocks_feature`

2. **E07-F22 (Rejection Reason)**
   - Automatic backward transition detection
   - Based on `progress_weight` decrease
   - Configurable via `require_rejection_reason`
   - No hardcoded transition lists

### Why It's Better

**Before:**
- ❌ 50+ places checking specific status names
- ❌ Hardcoded transition lists for rejection detection
- ❌ Code changes required for new workflows
- ❌ Can't customize per-project

**After:**
- ✅ 0 places checking specific status names
- ✅ Automatic rejection detection via config
- ✅ Add workflows with config only
- ✅ Per-project customization easy

### The Power of Config-Driven Design

**Single Source of Truth:**
All status semantics live in `.sharkconfig.json`. Code queries config for meaning.

**Automatic Features:**
- Rejection reason detection
- Progress calculation
- Work breakdown
- Status cascading

All work automatically for ANY workflow defined in config!

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
**Status:** Approved
