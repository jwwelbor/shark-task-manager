# E07-F23: Enhanced Status Tracking - Implementation Plan

**Feature:** E07-F23 - Enhanced Status Tracking and Visibility
**Epic:** E07 - Shark Enhancements
**Date:** 2026-01-16
**Author:** Architect Agent
**Status:** Ready for Implementation

---

## Overview

This document provides a phased implementation plan for E07-F23, breaking down the work into incremental deliverables with clear task boundaries and estimates.

**Total Estimated Effort:** 17 hours
**Phases:** 4
**Tasks:** 11

---

## Phase 1: Core Calculations (6 hours)

**Goal:** Implement config-driven status calculations without display changes.

**Deliverables:**
- New `internal/status` package with core calculation functions
- Unit tests with >90% coverage
- Config-driven progress and work breakdown logic

### Task E07-F23-001: Create Status Package and Types (2 hours)

**Description:** Set up the new `internal/status` package with all core types and package structure.

**Acceptance Criteria:**
- [ ] Create `internal/status/` directory
- [ ] Define `ProgressInfo` struct with all fields
- [ ] Define `WorkSummary` struct with all fields
- [ ] Define `ActionItems` and `TaskActionItem` structs
- [ ] Create `types.go` with all type definitions
- [ ] Add package-level documentation
- [ ] No implementation yet - just type definitions

**Files to Create:**
- `internal/status/types.go`

**Example:**
```go
package status

// ProgressInfo provides weighted and completion progress metrics
type ProgressInfo struct {
    WeightedPct     float64  // Weighted progress (e.g., 68.0)
    CompletionPct   float64  // Traditional completion % (e.g., 40.0)
    WeightedRatio   string   // "3.4/5" (weighted tasks complete)
    CompletionRatio string   // "2/5" (completed tasks / total)
    TotalTasks      int      // Total task count
}

// ... other types ...
```

**Testing:**
- Compile check only (no logic to test yet)

---

### Task E07-F23-002: Implement Weighted Progress Calculation (2 hours)

**Description:** Implement `CalculateProgress()` function using `progress_weight` from config.

**Acceptance Criteria:**
- [ ] Implement `CalculateProgress(statusCounts, cfg)` function
- [ ] Use `cfg.GetStatusMetadata(status).ProgressWeight` for calculations
- [ ] Handle missing metadata gracefully (default 0.0)
- [ ] Calculate both weighted and completion percentages
- [ ] Handle empty status counts (return 0/0)
- [ ] Generate formatted ratio strings

**Files to Create:**
- `internal/status/progress.go`
- `internal/status/progress_test.go`

**Implementation Pattern:**
```go
func CalculateProgress(statusCounts map[string]int, cfg *config.Config) *ProgressInfo {
    totalTasks := 0
    weightedProgress := 0.0
    completedTasks := 0

    for status, count := range statusCounts {
        totalTasks += count
        meta := cfg.GetStatusMetadata(status)
        if meta != nil {
            weightedProgress += float64(count) * meta.ProgressWeight
            if meta.ProgressWeight >= 1.0 {
                completedTasks += count
            }
        }
    }

    // ... calculate percentages and ratios ...
}
```

**Test Cases:**
- All completed tasks → 100% weighted, 100% completion
- Mixed statuses → correct weighted calculation
- Empty task list → 0/0 gracefully
- Missing config metadata → defaults to 0.0
- Single task at ready_for_approval (0.9) → 90% weighted, 0% completion

**Dependencies:**
- E07-F14 config with `progress_weight` field

---

### Task E07-F23-003: Implement Work Breakdown Calculation (2 hours)

**Description:** Implement `CalculateWorkRemaining()` using `responsibility` and `blocks_feature` from config.

**Acceptance Criteria:**
- [ ] Implement `CalculateWorkRemaining(statusCounts, cfg)` function
- [ ] Categorize by `responsibility`: agent, human, qa_team, none
- [ ] Use `blocks_feature` to identify blocked work
- [ ] Calculate not started (progress_weight=0.0)
- [ ] Calculate completed (progress_weight>=1.0)
- [ ] Handle missing metadata gracefully

**Files to Create:**
- `internal/status/work_breakdown.go`
- `internal/status/work_breakdown_test.go`

**Implementation Pattern:**
```go
func CalculateWorkRemaining(statusCounts map[string]int, cfg *config.Config) *WorkSummary {
    summary := &WorkSummary{}

    for status, count := range statusCounts {
        meta := cfg.GetStatusMetadata(status)
        if meta == nil {
            continue
        }

        switch meta.Responsibility {
        case "agent":
            summary.AgentWork += count
        case "human", "qa_team":
            summary.HumanWork += count
        // ... handle other cases ...
        }
    }

    return summary
}
```

**Test Cases:**
- All agent work → AgentWork count correct
- All human work → HumanWork count correct
- Blocked tasks → BlockedWork count correct
- Mixed responsibilities → all counts correct
- Missing metadata → graceful handling

---

## Phase 2: Repository Enhancements (4 hours)

**Goal:** Add repository methods to fetch status information efficiently.

**Deliverables:**
- `GetStatusInfo()` method in FeatureRepository
- `GetFeatureStatusRollup()` method in FeatureRepository
- `GetTaskStatusRollup()` method in EpicRepository
- Repository tests with real database

### Task E07-F23-004: Add Feature Repository Methods (2.5 hours)

**Description:** Implement `GetStatusInfo()` to fetch comprehensive feature status data.

**Acceptance Criteria:**
- [ ] Add `TaskRepository` dependency to FeatureRepository
- [ ] Update constructor: `NewFeatureRepository(db *DB, taskRepo *TaskRepository)`
- [ ] Implement `GetStatusInfo(ctx, featureID)` method
- [ ] Fetch feature, status breakdown, and tasks
- [ ] Return `FeatureStatusInfo` struct
- [ ] Handle errors gracefully

**Files to Modify:**
- `internal/repository/feature_repository.go`
- `internal/repository/feature_repository_test.go`

**Implementation Pattern:**
```go
// Add field to struct
type FeatureRepository struct {
    db       *DB
    taskRepo *TaskRepository  // NEW
}

// Update constructor
func NewFeatureRepository(db *DB, taskRepo *TaskRepository) *FeatureRepository {
    return &FeatureRepository{
        db:       db,
        taskRepo: taskRepo,
    }
}

// New method
func (r *FeatureRepository) GetStatusInfo(ctx context.Context, featureID int64) (*FeatureStatusInfo, error) {
    // 1. Get feature
    feature, err := r.GetByID(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get feature: %w", err)
    }

    // 2. Get status breakdown (already exists in E07-F14)
    statusBreakdown, err := r.GetTaskStatusBreakdown(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get status breakdown: %w", err)
    }

    // 3. Get tasks
    tasks, err := r.taskRepo.ListByFeature(ctx, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get tasks: %w", err)
    }

    return &FeatureStatusInfo{
        Feature:         feature,
        StatusBreakdown: statusBreakdown,
        Tasks:           tasks,
    }, nil
}
```

**Test Cases:**
- Feature with tasks → returns complete info
- Feature with no tasks → returns empty breakdown
- Feature not found → returns error
- Database error → wrapped error

**Dependencies:**
- E07-F14: `GetTaskStatusBreakdown()` already exists

---

### Task E07-F23-005: Add Epic Repository Methods (1.5 hours)

**Description:** Add rollup methods for epic status tracking.

**Acceptance Criteria:**
- [ ] Implement `GetFeatureStatusRollup(ctx, epicID)` method
- [ ] Implement `GetTaskStatusRollup(ctx, epicID)` method
- [ ] Use efficient GROUP BY queries
- [ ] Return map[string]int for counts
- [ ] Handle empty epics gracefully

**Files to Modify:**
- `internal/repository/epic_repository.go`
- `internal/repository/epic_repository_test.go`

**Implementation Pattern:**
```go
// GetFeatureStatusRollup returns feature status counts for an epic
func (r *EpicRepository) GetFeatureStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
    query := `
        SELECT status, COUNT(*) as count
        FROM features
        WHERE epic_id = ?
        GROUP BY status
    `

    rows, err := r.db.QueryContext(ctx, query, epicID)
    if err != nil {
        return nil, fmt.Errorf("failed to get feature rollup: %w", err)
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

// GetTaskStatusRollup returns task status counts across all features in epic
func (r *EpicRepository) GetTaskStatusRollup(ctx context.Context, epicID int64) (map[string]int, error) {
    query := `
        SELECT t.status, COUNT(*) as count
        FROM tasks t
        JOIN features f ON t.feature_id = f.id
        WHERE f.epic_id = ?
        GROUP BY t.status
    `

    // ... similar implementation ...
}
```

**Test Cases:**
- Epic with features → returns correct counts
- Epic with no features → returns empty map
- Epic with multiple features and tasks → rollup correct
- Database error → wrapped error

---

## Phase 3: CLI Display Enhancements (5 hours)

**Goal:** Enhance feature get, feature list, and epic get commands with new displays.

**Deliverables:**
- Enhanced feature get with progress breakdown, action items, work summary
- Enhanced feature list with health indicators and notes
- Enhanced epic get with rollups and impediments

### Task E07-F23-006: Enhance Feature Get Command (2.5 hours)

**Description:** Add progress breakdown, action items, and work summary sections to feature get.

**Acceptance Criteria:**
- [ ] Add status package imports
- [ ] Call `CalculateProgress()` for progress info
- [ ] Call `CalculateWorkRemaining()` for work summary
- [ ] Call `GetStatusContext()` for status context
- [ ] Implement `GetActionItems()` function in status package
- [ ] Update table formatter with new sections
- [ ] Update JSON output with new fields
- [ ] Add visual indicators (⏳, 🚫, ✅)

**Files to Modify:**
- `internal/cli/commands/feature.go`
- `internal/status/action_items.go` (NEW)
- `internal/status/context.go` (NEW)

**Implementation Pattern:**
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
    featureID := ... // parse from args
    statusInfo, err := featureRepo.GetStatusInfo(ctx, featureID)
    if err != nil {
        return err
    }

    // Calculate displays
    progress := status.CalculateProgress(statusInfo.StatusBreakdown, cfg)
    workSummary := status.CalculateWorkRemaining(statusInfo.StatusBreakdown, cfg)
    statusContext := status.GetStatusContext(statusInfo.Feature, statusInfo.StatusBreakdown, cfg)
    actionItems := status.GetActionItems(statusInfo.Tasks, cfg)

    // Output
    if cli.GlobalConfig.JSON {
        return outputFeatureJSON(statusInfo, progress, workSummary, statusContext, actionItems)
    }

    return outputFeatureTable(statusInfo, progress, workSummary, statusContext, actionItems)
}
```

**Test Cases:**
- Feature with all statuses → all sections display correctly
- Feature with no tasks → "No tasks" message
- Feature with waiting tasks → action items shown
- JSON output → all fields present
- Table output → formatting correct

**New Files:**
- `internal/status/action_items.go` - GetActionItems() implementation
- `internal/status/context.go` - GetStatusContext() implementation

---

### Task E07-F23-007: Enhance Feature List Command (1.5 hours)

**Description:** Add health indicators and notes column to feature list.

**Acceptance Criteria:**
- [ ] Add Health column with 🟢 🟡 🔴 indicators
- [ ] Add Notes column with status summaries
- [ ] Calculate health based on blockers and approval age
- [ ] Format progress with weighted ratio
- [ ] Update table headers and column widths
- [ ] JSON output includes health info

**Files to Modify:**
- `internal/cli/commands/feature.go`

**Implementation Pattern:**
```go
func runFeatureListCommand(cmd *cobra.Command, args []string) error {
    // ... get features ...

    rows := [][]string{}
    for _, feature := range features {
        // Get status info
        statusBreakdown, _ := featureRepo.GetTaskStatusBreakdown(ctx, feature.ID)

        // Calculate health
        health := calculateHealthIndicator(statusBreakdown, cfg)

        // Generate notes
        notes := generateNotesColumn(statusBreakdown)

        // Calculate progress
        progress := status.CalculateProgress(statusBreakdown, cfg)

        rows = append(rows, []string{
            feature.Key,
            feature.Title,
            health,
            fmt.Sprintf("%d%% (%.1f/%d)", int(progress.WeightedPct), progress.WeightedRatio, progress.TotalTasks),
            notes,
        })
    }

    // Output table
    headers := []string{"KEY", "TITLE", "HEALTH", "PROGRESS", "NOTES"}
    cli.OutputTable(headers, rows)
}

// Health calculation
func calculateHealthIndicator(statusBreakdown map[string]int, cfg *config.Config) string {
    blockedCount := statusBreakdown["blocked"]
    if blockedCount >= 3 {
        return "🔴" // At risk
    }
    if blockedCount >= 1 || statusBreakdown["ready_for_approval"] > 0 {
        return "🟡" // Attention
    }
    return "🟢" // Healthy
}

// Notes generation
func generateNotesColumn(statusBreakdown map[string]int) string {
    parts := []string{}
    if ready := statusBreakdown["ready_for_approval"]; ready > 0 {
        parts = append(parts, fmt.Sprintf("%d ready", ready))
    }
    if blocked := statusBreakdown["blocked"]; blocked > 0 {
        parts = append(parts, fmt.Sprintf("%d blocked", blocked))
    }
    if len(parts) == 0 {
        return "[all on track]"
    }
    return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}
```

**Test Cases:**
- Feature with blockers → 🔴 indicator
- Feature with approval backlog → 🟡 indicator
- Feature on track → 🟢 indicator
- Notes column → correct summaries

---

### Task E07-F23-008: Enhance Epic Get Command (1 hour)

**Description:** Add feature rollup, task rollup, and impediments sections to epic get.

**Acceptance Criteria:**
- [ ] Add feature status rollup section
- [ ] Add task rollup section
- [ ] Add impediments section with blocked features
- [ ] Show approval backlog with age
- [ ] Update table formatter
- [ ] Update JSON output

**Files to Modify:**
- `internal/cli/commands/epic.go`

**Implementation Pattern:**
```go
func runEpicGetCommand(cmd *cobra.Command, args []string) error {
    // ... get epic ...

    // Get feature rollup
    featureRollup, err := featureRepo.GetFeatureStatusRollup(ctx, epic.ID)
    if err != nil {
        return err
    }

    // Get task rollup
    taskRollup, err := epicRepo.GetTaskStatusRollup(ctx, epic.ID)
    if err != nil {
        return err
    }

    // Output with new sections
    if cli.GlobalConfig.JSON {
        return outputEpicJSON(epic, featureRollup, taskRollup)
    }

    return outputEpicTable(epic, featureRollup, taskRollup)
}
```

**Test Cases:**
- Epic with features → rollup correct
- Epic with no features → empty sections
- JSON output → all rollup fields present

---

## Phase 4: Testing & Documentation (2 hours)

**Goal:** Comprehensive tests and updated documentation.

**Deliverables:**
- Integration tests for all commands
- End-to-end tests
- Updated CLI documentation
- Architecture validation

### Task E07-F23-009: Add Integration Tests (1 hour)

**Description:** Add integration tests for all enhanced commands.

**Acceptance Criteria:**
- [ ] Test feature get with real database
- [ ] Test feature list with real database
- [ ] Test epic get with real database
- [ ] Test all calculation functions end-to-end
- [ ] Test JSON output format
- [ ] Test table output format

**Files to Create:**
- `internal/cli/commands/feature_integration_test.go`
- `internal/cli/commands/epic_integration_test.go`

**Test Pattern:**
```go
func TestFeatureGetIntegration(t *testing.T) {
    ctx := context.Background()
    db := test.GetTestDB()
    defer test.CleanupDB(db)

    // Seed test data
    epicID, featureID := test.SeedTestData()
    test.SeedTasks(featureID, map[string]int{
        "completed":          2,
        "ready_for_approval": 1,
        "in_development":     1,
        "draft":              1,
    })

    // Run command
    cmd := buildFeatureGetCommand()
    output := captureOutput(cmd, []string{"E07-F23"})

    // Verify output
    assert.Contains(t, output, "Progress Breakdown")
    assert.Contains(t, output, "68%") // Expected weighted progress
    assert.Contains(t, output, "Action Items")
    assert.Contains(t, output, "Work Summary")
}
```

---

### Task E07-F23-010: Add End-to-End Tests (0.5 hours)

**Description:** Add shell script tests for full command execution.

**Acceptance Criteria:**
- [ ] Test feature get command output
- [ ] Test feature list command output
- [ ] Test epic get command output
- [ ] Test JSON output parsing

**Files to Create:**
- `test/e2e/feature_get_test.sh`
- `test/e2e/feature_list_test.sh`
- `test/e2e/epic_get_test.sh`

**Test Pattern:**
```bash
#!/bin/bash

# Test feature get
output=$(./bin/shark feature get E07-F23 --json)
weighted_pct=$(echo "$output" | jq -r '.progress.weighted_pct')

if [ "$weighted_pct" != "68.0" ]; then
    echo "FAIL: Expected weighted_pct=68.0, got $weighted_pct"
    exit 1
fi

echo "PASS: Feature get e2e test"
```

---

### Task E07-F23-011: Update Documentation (0.5 hours)

**Description:** Update CLI reference and architecture docs.

**Acceptance Criteria:**
- [ ] Update `docs/CLI_REFERENCE.md` with new output examples
- [ ] Add examples for feature get, feature list, epic get
- [ ] Document JSON API response changes
- [ ] Update architecture summary in README

**Files to Modify:**
- `docs/CLI_REFERENCE.md`

**Content to Add:**
- Feature get output example with all sections
- Feature list output example with health indicators
- Epic get output example with rollups
- JSON schema for enhanced responses

---

## Task Summary

| Phase | Task | Effort | Status |
|-------|------|--------|--------|
| 1 | E07-F23-001: Create Status Package | 2h | Pending |
| 1 | E07-F23-002: Weighted Progress | 2h | Pending |
| 1 | E07-F23-003: Work Breakdown | 2h | Pending |
| 2 | E07-F23-004: Feature Repository | 2.5h | Pending |
| 2 | E07-F23-005: Epic Repository | 1.5h | Pending |
| 3 | E07-F23-006: Feature Get Display | 2.5h | Pending |
| 3 | E07-F23-007: Feature List Display | 1.5h | Pending |
| 3 | E07-F23-008: Epic Get Display | 1h | Pending |
| 4 | E07-F23-009: Integration Tests | 1h | Pending |
| 4 | E07-F23-010: E2E Tests | 0.5h | Pending |
| 4 | E07-F23-011: Documentation | 0.5h | Pending |
| **Total** | **11 tasks** | **17h** | |

---

## Implementation Order

**Sequential Dependencies:**

1. **Phase 1 (Core)** → Must complete before Phase 2
   - Status package is foundation for everything
   - Can work in parallel on 001, 002, 003

2. **Phase 2 (Repository)** → Must complete before Phase 3
   - Repository methods needed by CLI
   - Can work in parallel on 004, 005

3. **Phase 3 (CLI)** → Can start after Phase 2
   - Display enhancements build on repository
   - Can work in parallel on 006, 007, 008

4. **Phase 4 (Testing)** → Can start after Phase 3
   - Tests validate full integration
   - Can work in parallel on 009, 010, 011

**Parallel Work Opportunities:**
- Phase 1: All 3 tasks can be parallel (different files)
- Phase 2: Tasks 004 and 005 can be parallel
- Phase 3: Tasks 006, 007, 008 can be parallel after repository work
- Phase 4: All 3 tasks can be parallel

**Critical Path:** 1 → 2 → 6 (longest sequential chain)

---

## Risk Assessment

### Low Risk

✅ **No database schema changes** - Uses existing tables
✅ **Additive changes only** - No breaking API changes
✅ **Config-driven** - Behavior defined in config, easy to adjust
✅ **No external dependencies** - All internal packages

### Medium Risk

⚠️ **Performance** - Multiple queries per feature in list view
- **Mitigation:** Optimize with batch queries in Phase 2
- **Monitoring:** Add latency tracking

⚠️ **Config missing metadata** - Missing progress_weight in config
- **Mitigation:** Graceful defaults (0.0)
- **Validation:** Config load-time validation

### Minimal Risk

🟢 **E07-F14 dependency** - Already implemented and stable
🟢 **Repository tests** - Use real DB (well-established pattern)
🟢 **CLI tests** - Use mocks (no DB dependencies)

---

## Rollback Plan

**If issues arise:**

1. **Phase 1 issues:** No impact - status package not yet used
2. **Phase 2 issues:** Revert repository changes, no CLI impact
3. **Phase 3 issues:** Feature flag to disable new displays (fall back to old output)
4. **Phase 4 issues:** Tests don't affect production

**Rollback Command:**
```bash
git revert <commit-range>
make build && make test
```

**Compatibility:** Old CLI binaries continue working (no breaking changes).

---

## Success Metrics

### Completion Criteria

- [ ] All 11 tasks completed
- [ ] All unit tests passing (>90% coverage)
- [ ] All integration tests passing
- [ ] All E2E tests passing
- [ ] Documentation updated
- [ ] Code review approved
- [ ] Performance targets met (p95 < 100ms)

### Performance Validation

```bash
# Feature get latency
time ./bin/shark feature get E07-F23
# Target: < 100ms

# Feature list latency (20 features)
time ./bin/shark feature list E07
# Target: < 200ms

# Epic get latency
time ./bin/shark epic get E07
# Target: < 500ms
```

### User Acceptance

- [ ] PM can answer "what needs attention?" in < 3 seconds
- [ ] Developer can see agent work recognized (90% progress)
- [ ] Orchestrator gets full context in single API call

---

## Post-Implementation

### Monitoring

**After deployment, track:**
- Command latency (p50, p95, p99)
- Query count per command
- Error rates
- User feedback on display clarity

### Iteration Plan

**Phase 2 (Future):**
- Add caching if latency issues
- Add filtering flags: `--awaiting-approval`, `--with-blockers`
- Add velocity metrics

**Phase 3 (Future):**
- Real-time WebSocket updates
- Push notifications
- Dashboard UI

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
**Status:** Ready for Implementation
**Next Step:** Begin Phase 1 - Task E07-F23-001
