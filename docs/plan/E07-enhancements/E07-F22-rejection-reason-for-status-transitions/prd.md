# E07-F22: Rejection Reason for Status Transitions (PRD)

**Feature:** E07-F22 - Rejection Reason for Status Transitions
**Epic:** E07 - Shark Enhancements
**Status:** draft
**Priority:** 5
**Created:** 2026-01-16

---

## Executive Summary

Implement **config-driven rejection reason** requirement for backward status transitions. When a task's `progress_weight` decreases (e.g., `ready_for_code_review` 85% → `in_development` 50%), require a rejection reason to document why work was sent back.

**Key Innovation:** Uses `progress_weight` from config (E07-F14) to automatically detect backward transitions—no hardcoded status lists needed!

---

## Problem Statement

### Current State
- Tasks can move backward in workflow (e.g., code review → development) without explanation
- No record of WHY work was rejected or sent back
- QA/review rejection reasons are lost
- Hard to analyze rejection patterns or identify quality issues

### User Pain Points
1. **Project Managers:** "Why was this task sent back? I can't find any notes."
2. **Developers:** "What specifically failed review? I have to ask the reviewer."
3. **Quality Teams:** "We need rejection metrics but can't track reasons systematically."
4. **Stakeholders:** "How many tasks are being rejected? What are the common issues?"

---

## Solution Overview

### Config-Driven Rejection Reason

**1. Automatic Detection**
```json
{
  "status_metadata": {
    "ready_for_code_review": {
      "progress_weight": 0.85  // 85% complete
    },
    "in_development": {
      "progress_weight": 0.50  // 50% complete
    }
  },
  "require_rejection_reason": true
}
```

**If `progress_weight` decreases → rejection reason required**

**2. Simple CLI Integration**
```bash
# Backward transition WITHOUT reason = ERROR
$ shark task update E07-F22-001 --status=in_development
Error: rejection reason required: transitioning from ready_for_code_review (85%) to in_development (50%) decreases progress

# Backward transition WITH reason = SUCCESS
$ shark task update E07-F22-001 --status=in_development \
  --rejection-reason="Failed code review: missing error handling for edge cases"
✓ Task updated with rejection reason recorded
```

**3. Stored in task_history**
```sql
SELECT task_id, from_status, to_status, rejection_reason, changed_at
FROM task_history
WHERE rejection_reason IS NOT NULL;
```

---

## User Stories

### Story 1: Code Reviewer Rejecting Work
**As a** code reviewer
**I want** to provide a rejection reason when sending code back to development
**So that** the developer knows exactly what needs to be fixed

**Acceptance Criteria:**
- [ ] CLI requires `--rejection-reason` flag for backward transitions
- [ ] Error message clearly states rejection reason is required
- [ ] Rejection reason stored in task_history
- [ ] Rejection reason visible in task history output

### Story 2: Project Manager Tracking Rejections
**As a** project manager
**I want** to see rejection reasons for tasks that were sent back
**So that** I can identify patterns and quality issues

**Acceptance Criteria:**
- [ ] `shark task history E07-F22-001` shows rejection reasons
- [ ] `shark feature get E07-F22` shows tasks with rejections
- [ ] JSON output includes rejection_reason field
- [ ] Can filter tasks by "has rejection"

### Story 3: Developer Understanding Feedback
**As a** developer
**I want** to see the rejection reason immediately when a task is sent back
**So that** I know what to fix without having to ask the reviewer

**Acceptance Criteria:**
- [ ] `shark task get` shows most recent rejection reason
- [ ] Task detail includes "Last Rejection: ..." section
- [ ] Rejection reason shown in task list (if recent)

### Story 4: Configurable Rejection Requirements
**As a** team lead
**I want** to enable/disable rejection reason requirements
**So that** I can adapt to my team's workflow

**Acceptance Criteria:**
- [ ] `require_rejection_reason: true` in config enables feature
- [ ] `require_rejection_reason: false` makes rejection reason optional
- [ ] Config change takes effect immediately (no restart)

---

## Technical Approach

### 1. Config-Driven Detection

**File:** `internal/config/config.go`
```go
type Config struct {
    RequireRejectionReason bool `json:"require_rejection_reason"`
    StatusMetadata map[string]StatusMetadata `json:"status_metadata"`
}

type StatusMetadata struct {
    ProgressWeight float64 `json:"progress_weight"`  // E07-F14
    // ... other fields
}

// IsBackwardTransition checks if progress decreases
func (c *Config) IsBackwardTransition(oldStatus, newStatus string) bool {
    oldMeta := c.GetStatusMetadata(oldStatus)
    newMeta := c.GetStatusMetadata(newStatus)

    if oldMeta == nil || newMeta == nil {
        return false
    }

    return newMeta.ProgressWeight < oldMeta.ProgressWeight
}
```

### 2. Repository Integration

**File:** `internal/repository/task_repository.go`
```go
type UpdateStatusOptions struct {
    RejectionReason string
    Notes           string
    Agent           string
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, taskKey string, newStatus string, opts UpdateStatusOptions) error {
    task, err := r.GetByKey(ctx, taskKey)
    if err != nil {
        return err
    }

    cfg := config.Get()

    // Check if rejection reason required
    if cfg.RequireRejectionReason && cfg.IsBackwardTransition(task.Status, newStatus) {
        if opts.RejectionReason == "" {
            oldMeta := cfg.GetStatusMetadata(task.Status)
            newMeta := cfg.GetStatusMetadata(newStatus)
            return &RejectionReasonRequiredError{
                FromStatus:     task.Status,
                ToStatus:       newStatus,
                FromProgress:   oldMeta.ProgressWeight * 100,
                ToProgress:     newMeta.ProgressWeight * 100,
            }
        }
    }

    // Update status
    // ...

    // Record history with rejection reason
    return r.recordHistory(ctx, task.ID, task.Status, newStatus, opts)
}

type RejectionReasonRequiredError struct {
    FromStatus   string
    ToStatus     string
    FromProgress float64
    ToProgress   float64
}

func (e *RejectionReasonRequiredError) Error() string {
    return fmt.Sprintf(
        "rejection reason required: transitioning from %s (%.0f%%) to %s (%.0f%%) decreases progress",
        e.FromStatus, e.FromProgress,
        e.ToStatus, e.ToProgress,
    )
}
```

### 3. Database Schema

**Migration:** `internal/db/db.go`
```sql
-- Add rejection_reason to task_history (if not exists)
ALTER TABLE task_history ADD COLUMN rejection_reason TEXT;
CREATE INDEX IF NOT EXISTS idx_task_history_rejection ON task_history(rejection_reason);
```

### 4. CLI Commands

**File:** `internal/cli/commands/task_update.go`
```go
var taskUpdateCmd = &cobra.Command{
    Use:   "update <task-key>",
    Short: "Update task properties",
    Args:  cobra.ExactArgs(1),
    RunE:  runTaskUpdate,
}

var (
    taskUpdateStatus         string
    taskUpdateRejectionReason string
)

func init() {
    taskUpdateCmd.Flags().StringVar(&taskUpdateStatus, "status", "", "New status")
    taskUpdateCmd.Flags().StringVar(&taskUpdateRejectionReason, "rejection-reason", "", "Reason for backward transition")
    taskCmd.AddCommand(taskUpdateCmd)
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    taskKey := args[0]

    repoDb, err := cli.GetDB(ctx)
    if err != nil {
        return err
    }

    taskRepo := repository.NewTaskRepository(repoDb)

    opts := repository.UpdateStatusOptions{
        RejectionReason: taskUpdateRejectionReason,
    }

    err = taskRepo.UpdateStatus(ctx, taskKey, taskUpdateStatus, opts)
    if err != nil {
        var rejErr *repository.RejectionReasonRequiredError
        if errors.As(err, &rejErr) {
            // User-friendly error message
            cli.Error(fmt.Sprintf("Rejection reason required"))
            cli.Info(fmt.Sprintf("Transitioning from %s (%.0f%% complete) to %s (%.0f%% complete) moves backward in the workflow.",
                rejErr.FromStatus, rejErr.FromProgress,
                rejErr.ToStatus, rejErr.ToProgress))
            cli.Info(fmt.Sprintf("Please provide a reason using --rejection-reason flag:"))
            cli.Info(fmt.Sprintf("  shark task update %s --status=%s --rejection-reason=\"<reason>\"", taskKey, taskUpdateStatus))
            os.Exit(1)
        }
        return err
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(map[string]interface{}{
            "task_key": taskKey,
            "status": taskUpdateStatus,
            "rejection_reason": taskUpdateRejectionReason,
        })
    }

    cli.Success(fmt.Sprintf("Task %s updated to %s", taskKey, taskUpdateStatus))
    if taskUpdateRejectionReason != "" {
        cli.Info(fmt.Sprintf("Rejection reason: %s", taskUpdateRejectionReason))
    }

    return nil
}
```

---

## Benefits

### 1. Automatic Detection
- No hardcoded status lists
- Works with any workflow (defined in config)
- Detects backward transitions automatically based on `progress_weight`

### 2. Quality Insights
- Track rejection reasons systematically
- Identify patterns (e.g., "missing error handling" appears often)
- Measure rejection rates per developer/feature

### 3. Developer Experience
- Clear error messages
- Immediate feedback on what failed
- No need to ask reviewer for details

### 4. Audit Trail
- Complete history of rejections
- Who rejected, when, and why
- Can be analyzed for quality metrics

---

## Examples

### Example 1: Code Review Rejection

```bash
# Task ready for review
$ shark task get E07-F22-001
Task: E07-F22-001 - Implement JWT validation
Status: ready_for_code_review (85% complete)

# Reviewer sends back to development
$ shark task update E07-F22-001 --status=in_development
Error: rejection reason required: transitioning from ready_for_code_review (85%) to in_development (50%) decreases progress

# Reviewer provides reason
$ shark task update E07-F22-001 --status=in_development \
  --rejection-reason="Missing error handling for invalid JWT format. Added TODO comments in code."
✓ Task E07-F22-001 updated to in_development
ℹ Rejection reason: Missing error handling for invalid JWT format. Added TODO comments in code.

# Developer sees rejection reason
$ shark task get E07-F22-001
Task: E07-F22-001 - Implement JWT validation
Status: in_development (50% complete)
Last Rejection: Missing error handling for invalid JWT format. Added TODO comments in code.
  Rejected at: 2026-01-16 14:30:00
  Rejected from: ready_for_code_review
```

### Example 2: QA Rejection

```bash
# Task ready for approval
$ shark task get E07-F22-002
Task: E07-F22-002 - Add user profile page
Status: ready_for_approval (90% complete)

# QA rejects back to development
$ shark task update E07-F22-002 --status=in_development \
  --rejection-reason="Browser compatibility issue: layout broken in Safari. Screenshot attached to task file."
✓ Task E07-F22-002 updated to in_development
ℹ Rejection reason: Browser compatibility issue: layout broken in Safari. Screenshot attached to task file.
```

### Example 3: Optional Rejection Reason (Config Disabled)

```json
{
  "require_rejection_reason": false
}
```

```bash
# Backward transition without reason = OK
$ shark task update E07-F22-003 --status=in_development
✓ Task E07-F22-003 updated to in_development
⚠ Consider adding --rejection-reason for better tracking
```

---

## Acceptance Criteria

### Phase 1: Core Functionality
- [ ] Config option `require_rejection_reason` added
- [ ] `progress_weight` used to detect backward transitions
- [ ] CLI requires `--rejection-reason` for backward transitions
- [ ] `rejection_reason` stored in task_history
- [ ] Clear error message when rejection reason missing

### Phase 2: CLI Integration
- [ ] `shark task update --status=X --rejection-reason="..."` works
- [ ] `shark task get` shows last rejection reason
- [ ] `shark task history` includes rejection reasons
- [ ] JSON output includes `rejection_reason` field

### Phase 3: Reporting
- [ ] `shark task list --with-rejections` shows rejected tasks
- [ ] `shark feature get` shows tasks with recent rejections
- [ ] `shark report rejections` generates rejection summary

---

## Non-Goals

- ❌ Rejection approval workflow (out of scope)
- ❌ Rejection notifications (future feature)
- ❌ Rejection templates (future feature)
- ❌ Multi-level approvals (out of scope)

---

## Dependencies

- **E07-F14 (Cascading Status Calculation):** Provides `progress_weight` in config
- **Existing config system:** Uses `.sharkconfig.json`
- **task_history table:** Already exists, just add column

---

## Implementation Tasks

See `tasks/` directory for detailed implementation tasks:
- T-E07-F22-001: Add rejection_reason to task_history
- T-E07-F22-002: Implement IsBackwardTransition in config
- T-E07-F22-003: Add rejection reason validation to UpdateStatus
- T-E07-F22-004: Add CLI flag --rejection-reason
- T-E07-F22-005: Add rejection reason to task history display
- T-E07-F22-006: Add config option require_rejection_reason
- T-E07-F22-007: Add tests for rejection reason validation

**Estimated Effort:** 7 tasks, ~10-15 hours

---

## Testing Strategy

### Unit Tests
```go
func TestIsBackwardTransition(t *testing.T) {
    cfg := &config.Config{
        StatusMetadata: map[string]config.StatusMetadata{
            "ready_for_code_review": {ProgressWeight: 0.85},
            "in_development": {ProgressWeight: 0.50},
            "completed": {ProgressWeight: 1.0},
        },
    }

    tests := []struct {
        name       string
        oldStatus  string
        newStatus  string
        isBackward bool
    }{
        {"review to dev", "ready_for_code_review", "in_development", true},
        {"dev to review", "in_development", "ready_for_code_review", false},
        {"review to complete", "ready_for_code_review", "completed", false},
        {"complete to review", "completed", "ready_for_code_review", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := cfg.IsBackwardTransition(tt.oldStatus, tt.newStatus)
            if result != tt.isBackward {
                t.Errorf("expected %v, got %v", tt.isBackward, result)
            }
        })
    }
}
```

### Integration Tests
```go
func TestUpdateStatus_RequiresRejectionReason(t *testing.T) {
    // Setup
    db := test.GetTestDB()
    repo := repository.NewTaskRepository(db)

    // Create task in ready_for_code_review
    task := createTestTask(t, repo, "ready_for_code_review")

    // Attempt backward transition without reason
    opts := repository.UpdateStatusOptions{} // No rejection reason
    err := repo.UpdateStatus(ctx, task.Key, "in_development", opts)

    // Should fail
    var rejErr *repository.RejectionReasonRequiredError
    if !errors.As(err, &rejErr) {
        t.Fatal("expected RejectionReasonRequiredError")
    }

    // With rejection reason should succeed
    opts.RejectionReason = "Failed code review"
    err = repo.UpdateStatus(ctx, task.Key, "in_development", opts)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Verify history recorded
    history := getTaskHistory(t, repo, task.ID)
    if history[0].RejectionReason != "Failed code review" {
        t.Errorf("expected rejection reason in history")
    }
}
```

---

## Success Metrics

### Functional Metrics
- ✅ 100% of backward transitions have rejection reasons (if config enabled)
- ✅ Zero "why was this rejected?" questions in team communication
- ✅ Rejection reasons visible in all relevant CLI commands

### Quality Metrics
- ✅ Rejection pattern analysis possible
- ✅ Average rejection reasons > 10 words (meaningful feedback)
- ✅ Rejection rate per developer trackable

### User Satisfaction
- ✅ Developers can understand rejection without asking reviewer
- ✅ Reviewers find it easy to provide rejection reasons
- ✅ Project managers have visibility into rejection patterns

---

## Future Enhancements

### Phase 2: Rejection Analytics
- `shark report rejections --by-developer`
- `shark report rejections --by-reason`
- `shark report rejections --by-feature`
- Rejection rate trends over time

### Phase 3: Rejection Templates
```json
{
  "rejection_templates": {
    "missing_tests": "Missing unit tests for new functionality",
    "error_handling": "Insufficient error handling for edge cases",
    "code_quality": "Code quality issues: {specific_issues}"
  }
}
```

```bash
shark task update E07-F22-001 --status=in_development \
  --rejection-template=missing_tests
```

### Phase 4: Rejection Notifications
- Notify developer when task rejected
- Include rejection reason in notification
- Slack/email integration

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
**Status:** Ready for Implementation
