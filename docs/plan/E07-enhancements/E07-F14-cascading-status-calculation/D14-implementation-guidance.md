# Implementation Guidance: Enhanced Status Tracking UX

**Feature**: E07-F14 - Cascading Status Calculation
**Document Type**: Implementation Guide
**Author**: CXDesigner Agent
**Date**: 2026-01-16
**Status**: Draft

---

## Executive Summary

This document provides implementation guidance for the enhanced status tracking UX designed in [D12-ux-design-status-tracking.md](./D12-ux-design-status-tracking.md) and validated through the journey mapping in [D13-journey-map-status-experience.md](./D13-journey-map-status-experience.md).

### Key Deliverables

1. **Enhanced List Views**: `shark feature list`, `shark epic list`
2. **Enhanced Get Views**: `shark feature get`, `shark epic get`
3. **JSON Schema Extensions**: Additional fields for orchestrators
4. **Repository Methods**: Status calculation and health indicators

### Implementation Phases

| Phase | Tasks | Priority | Time Estimate |
|-------|-------|----------|---------------|
| Phase 1 | Core status calculation (T-E07-F14-001 to T-E07-F14-008) | High | Already done |
| Phase 2 | Feature get enhancement (T-E07-F14-009) | High | 4 hours |
| Phase 3 | Epic get enhancement (T-E07-F14-010) | High | 4 hours |
| Phase 4 | List views enhancement (T-E07-F14-011, T-E07-F14-012) | Medium | 6 hours |
| Phase 5 | JSON schema enhancement | Medium | 3 hours |
| **Total** | | | **17 hours** |

---

## Design Principles to Follow

### 1. Scannable First

**Principle**: Users should understand status in < 3 seconds

**Implementation Requirements**:
- Use color coding consistently
- Add visual indicators (⏳, ⚠️, ✓)
- Group related information
- Minimize clutter

**Code Pattern**:
```go
// Good: Scannable status with context
fmt.Printf("Status:   %s (%s) %s\n",
    colorize(status),
    phaseContext,
    visualIndicator)

// Bad: Wall of text
fmt.Printf("Status: %s\n", status)
```

---

### 2. Progressive Disclosure

**Principle**: Summary → Details → JSON for different use cases

**Implementation Requirements**:
- List views show summary only
- Get views show full details
- JSON includes all metadata
- Each level adds more context

**Code Pattern**:
```go
// List view: minimal
renderFeatureListRow(feature) // Shows status, progress, notes

// Get view: detailed
renderFeatureDetails(feature, tasks, breakdown, actions)

// JSON: complete
outputJSON(feature, tasks, breakdown, actions, health, metadata)
```

---

### 3. Color as Signal

**Principle**: Phase-based colors guide attention to actionable items

**Implementation Requirements**:
- Follow workflow phase color scheme
- Purple = human action needed (highest priority)
- Red = blocked (critical)
- Yellow = in progress (normal)
- Use `workflow.Service.FormatStatusForDisplay()`

**Code Pattern**:
```go
// Use workflow service for consistent colors
formatted := workflowService.FormatStatusForDisplay(status, true)
fmt.Printf("%s", formatted.Colored)

// Respect --no-color flag
if cli.GlobalConfig.NoColor {
    fmt.Printf("%s", status)
}
```

---

### 4. Status Context

**Principle**: Always explain why status has its current value

**Implementation Requirements**:
- Show calculated vs manual
- Explain what's driving status
- Show task breakdown by phase
- Add "because X" explanations

**Code Pattern**:
```go
// Good: Contextual status
fmt.Printf("Status:   %s (%s)\n", status, context)
fmt.Printf("          %s\n", explanation)

// Bad: Just the status
fmt.Printf("Status: %s\n", status)
```

---

### 5. Actionable Insights

**Principle**: Surface what needs attention immediately

**Implementation Requirements**:
- Add "Action Items" section
- Group by action type (waiting, blocked, next)
- Show specific task references
- Make it easy to act

**Code Pattern**:
```go
// Action items section
if len(actionItems) > 0 {
    renderSectionHeader("Action Items")
    for _, item := range actionItems {
        renderActionItem(item)
    }
}
```

---

## Implementation Task Breakdown

### Task Group 1: Repository Enhancements

**Tasks**: T-E07-F14-009, T-E07-F14-010

**Goal**: Add repository methods for status calculation and health indicators

#### New Repository Methods

```go
// FeatureRepository enhancements
type FeatureStatusInfo struct {
    Status            string
    StatusSource      string // "calculated" or "manual"
    StatusExplanation StatusExplanation
    ProgressBreakdown ProgressBreakdown
    ActionItems       ActionItems
    Health            HealthIndicators
}

func (r *FeatureRepository) GetStatusInfo(ctx context.Context, featureID int64) (*FeatureStatusInfo, error)

// EpicRepository enhancements
type EpicStatusInfo struct {
    Status                string
    StatusSource          string
    StatusExplanation     StatusExplanation
    ProgressBreakdown     ProgressBreakdown
    FeatureDistribution   FeatureDistribution
    ActionItems           ActionItems
    Health                HealthIndicators
}

func (r *EpicRepository) GetStatusInfo(ctx context.Context, epicID int64) (*EpicStatusInfo, error)
```

#### Status Explanation Logic

```go
type StatusExplanation struct {
    PrimaryPhase string // "approval", "development", "planning", "done", "mixed"
    Reason       string // "waiting_for_approval", "in_development", "blocked", etc.
    Details      string // Human-readable explanation
}

func calculateFeatureStatusExplanation(tasks []*models.Task) StatusExplanation {
    // Count tasks by phase
    phaseCount := make(map[string]int)
    for _, task := range tasks {
        phase := getWorkflowPhase(task.Status)
        phaseCount[phase]++
    }

    // Determine primary phase
    if phaseCount["approval"] > 0 {
        return StatusExplanation{
            PrimaryPhase: "approval",
            Reason:       "waiting_for_approval",
            Details:      fmt.Sprintf("%d task(s) ready for approval", phaseCount["approval"]),
        }
    }

    if phaseCount["blocked"] > 0 {
        return StatusExplanation{
            PrimaryPhase: "any",
            Reason:       "blocked",
            Details:      fmt.Sprintf("%d task(s) blocked", phaseCount["blocked"]),
        }
    }

    if phaseCount["development"] > 0 {
        return StatusExplanation{
            PrimaryPhase: "development",
            Reason:       "in_development",
            Details:      fmt.Sprintf("%d task(s) in development", phaseCount["development"]),
        }
    }

    // ... continue for other phases
}
```

#### Progress Breakdown Logic

```go
type ProgressBreakdown struct {
    Completed int
    Total     int
    ByPhase   map[string]int
}

func calculateProgressBreakdown(tasks []*models.Task) ProgressBreakdown {
    breakdown := ProgressBreakdown{
        Total:   len(tasks),
        ByPhase: make(map[string]int),
    }

    for _, task := range tasks {
        phase := getWorkflowPhase(task.Status)
        breakdown.ByPhase[phase]++

        if task.Status == "completed" || task.Status == "cancelled" {
            breakdown.Completed++
        }
    }

    return breakdown
}
```

#### Action Items Logic

```go
type ActionItems struct {
    WaitingForApproval []*ActionItem
    Blocked            []*ActionItem
    InDevelopment      []*ActionItem
}

type ActionItem struct {
    TaskKey   string
    Title     string
    Status    string
    Reason    string // For blocked items
}

func extractActionItems(tasks []*models.Task) ActionItems {
    items := ActionItems{}

    for _, task := range tasks {
        switch task.Status {
        case "ready_for_approval", "in_approval":
            items.WaitingForApproval = append(items.WaitingForApproval, &ActionItem{
                TaskKey: task.Key,
                Title:   task.Title,
                Status:  task.Status,
            })

        case "blocked":
            items.Blocked = append(items.Blocked, &ActionItem{
                TaskKey: task.Key,
                Title:   task.Title,
                Status:  task.Status,
                Reason:  getBlockReason(task), // From task history
            })

        case "in_development":
            items.InDevelopment = append(items.InDevelopment, &ActionItem{
                TaskKey: task.Key,
                Title:   task.Title,
                Status:  task.Status,
            })
        }
    }

    return items
}
```

#### Health Indicators Logic

```go
type HealthIndicators struct {
    Overall          string // "good", "caution", "critical"
    HasBlockers      bool
    AwaitingApproval bool
    InProgress       bool
}

func calculateHealthIndicators(statusInfo *FeatureStatusInfo) HealthIndicators {
    health := HealthIndicators{
        HasBlockers:      len(statusInfo.ActionItems.Blocked) > 0,
        AwaitingApproval: len(statusInfo.ActionItems.WaitingForApproval) > 0,
        InProgress:       len(statusInfo.ActionItems.InDevelopment) > 0,
    }

    // Determine overall health
    if health.HasBlockers {
        health.Overall = "critical"
    } else if health.AwaitingApproval {
        health.Overall = "caution"
    } else if health.InProgress {
        health.Overall = "good"
    } else {
        health.Overall = "good"
    }

    return health
}
```

---

### Task Group 2: CLI Command Enhancements

**Tasks**: T-E07-F14-009, T-E07-F14-010, T-E07-F14-011, T-E07-F14-012

**Goal**: Update CLI commands to show enhanced status information

#### Feature Get Enhancement (T-E07-F14-009)

**File**: `internal/cli/commands/feature.go`

**Changes Needed**:

1. **Fetch status info from repository**:
```go
func runFeatureGet(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    // Get status info
    statusInfo, err := featureRepo.GetStatusInfo(ctx, feature.ID)
    if err != nil {
        // Handle error
    }

    // Output
    if cli.GlobalConfig.JSON {
        return outputFeatureJSON(feature, tasks, statusInfo)
    }

    renderFeatureDetails(feature, tasks, statusInfo, workflowService)
}
```

2. **Update renderFeatureDetails function**:
```go
func renderFeatureDetails(feature *models.Feature, tasks []*models.Task, statusInfo *FeatureStatusInfo, workflowService *workflow.Service) {
    // Print header
    pterm.DefaultSection.Printf("Feature: %s - %s", feature.Key, feature.Title)
    fmt.Println()

    // Status section
    renderStatusSection(feature.Status, statusInfo.StatusExplanation, statusInfo.StatusSource, workflowService)

    // Progress section
    renderProgressSection(statusInfo.ProgressBreakdown)

    // Status distribution
    renderStatusDistribution(statusInfo.StatusBreakdown, workflowService)

    // Action items (if any)
    if hasActionItems(statusInfo.ActionItems) {
        renderActionItemsSection(statusInfo.ActionItems)
    }

    // Task list (workflow-ordered)
    renderTaskList(tasks, workflowService)
}
```

3. **Add helper rendering functions**:
```go
func renderStatusSection(status string, explanation StatusExplanation, source string, ws *workflow.Service) {
    fmt.Printf("Status:   ")

    // Format status with color
    if !cli.GlobalConfig.NoColor && ws != nil {
        formatted := ws.FormatStatusForDisplay(status, true)
        fmt.Printf("%s", formatted.Colored)
    } else {
        fmt.Printf("%s", status)
    }

    // Add context
    fmt.Printf(" (%s)", explanation.Reason)

    // Add visual indicator
    indicator := getVisualIndicator(explanation.Reason)
    if indicator != "" {
        fmt.Printf(" %s", indicator)
    }

    fmt.Println()

    // Show source
    fmt.Printf("          %s from task statuses\n", source)
    fmt.Println()
}

func getVisualIndicator(reason string) string {
    switch reason {
    case "waiting_for_approval":
        return "⏳"
    case "blocked":
        return "⚠️"
    case "completed":
        return "✓"
    default:
        return ""
    }
}

func renderProgressSection(breakdown ProgressBreakdown) {
    fmt.Printf("Progress: %d%% complete (%d of %d tasks)\n",
        int(float64(breakdown.Completed)/float64(breakdown.Total)*100),
        breakdown.Completed,
        breakdown.Total)

    // Show breakdown by phase
    for phase, count := range breakdown.ByPhase {
        if count > 0 && phase != "done" {
            fmt.Printf("          %d task(s) in %s\n", count, phase)
        }
    }

    fmt.Println()
}

func renderActionItemsSection(actionItems ActionItems) {
    pterm.DefaultSection.Println("Action Items")
    fmt.Println()

    // Waiting for approval (highest priority)
    if len(actionItems.WaitingForApproval) > 0 {
        fmt.Printf("⏳ %d task(s) ready for approval:\n", len(actionItems.WaitingForApproval))
        for _, item := range actionItems.WaitingForApproval {
            fmt.Printf("   • %s: %s\n", item.TaskKey, item.Title)
        }
        fmt.Println()
    }

    // Blocked (critical)
    if len(actionItems.Blocked) > 0 {
        fmt.Printf("⚠️  %d task(s) blocked:\n", len(actionItems.Blocked))
        for _, item := range actionItems.Blocked {
            fmt.Printf("   • %s: %s", item.TaskKey, item.Title)
            if item.Reason != "" {
                fmt.Printf(" (Reason: %s)", item.Reason)
            }
            fmt.Println()
        }
        fmt.Println()
    }

    // In development (informational)
    if len(actionItems.InDevelopment) > 0 {
        fmt.Printf("⚙  %d task(s) in development:\n", len(actionItems.InDevelopment))
        for _, item := range actionItems.InDevelopment {
            fmt.Printf("   • %s: %s\n", item.TaskKey, item.Title)
        }
        fmt.Println()
    }
}
```

4. **Update task list rendering** (workflow-ordered):
```go
func renderTaskList(tasks []*models.Task, ws *workflow.Service) {
    pterm.DefaultSection.Println("Task List (grouped by workflow phase)")
    fmt.Println()

    // Group tasks by phase
    tasksByPhase := groupTasksByPhase(tasks)

    // Render in phase order: approval → development → planning → done
    phaseOrder := []string{"approval", "review", "qa", "development", "planning", "done"}

    for _, phase := range phaseOrder {
        phaseTasks := tasksByPhase[phase]
        if len(phaseTasks) == 0 {
            continue
        }

        // Phase header
        fmt.Printf("%s %s PHASE\n", getPhaseIndicator(phase), strings.ToUpper(phase))

        // Tasks
        for _, task := range phaseTasks {
            fmt.Printf("  %s  %-20s  ", task.Key, truncate(task.Title, 50))

            if !cli.GlobalConfig.NoColor && ws != nil {
                formatted := ws.FormatStatusForDisplay(task.Status, true)
                fmt.Printf("%s\n", formatted.Colored)
            } else {
                fmt.Printf("%s\n", task.Status)
            }
        }

        fmt.Println()
    }
}

func groupTasksByPhase(tasks []*models.Task) map[string][]*models.Task {
    groups := make(map[string][]*models.Task)

    for _, task := range tasks {
        phase := getWorkflowPhase(task.Status)
        groups[phase] = append(groups[phase], task)
    }

    return groups
}

func getPhaseIndicator(phase string) string {
    switch phase {
    case "approval":
        return "⏳"
    case "development":
        return "⚙"
    case "planning":
        return "📝"
    case "done":
        return "✓"
    default:
        return "•"
    }
}
```

---

#### Epic Get Enhancement (T-E07-F14-010)

**File**: `internal/cli/commands/epic.go`

**Changes Needed**: Similar to feature get, but with epic-specific logic

1. **Fetch epic status info**
2. **Render epic status section**
3. **Add feature distribution section**
4. **Add aggregated action items**
5. **Render feature list (urgency-ordered)**

**Key Differences**:
- Epic status is calculated from feature statuses (not tasks)
- Feature distribution shows count per phase
- Action items are aggregated across all features
- Feature list is ordered by urgency (waiting → blocked → in progress → planning → completed)

---

#### List View Enhancements (T-E07-F14-011, T-E07-F14-012)

**Files**: `internal/cli/commands/feature.go`, `internal/cli/commands/epic.go`

**Changes Needed**:

1. **Update table columns**:
```go
func renderFeatureListTable(features []FeatureWithTaskCount, workflowService *workflow.Service) {
    tableData := pterm.TableData{
        {"Key", "Title", "Status", "Progress", "Tasks", "Notes"},
    }

    for _, feature := range features {
        // Status with context
        statusDisplay := formatStatusWithContext(feature.Status, feature.StatusInfo)

        // Progress with ratio
        progressDisplay := fmt.Sprintf("%d%% (%d/%d)",
            int(feature.ProgressPct),
            feature.StatusInfo.ProgressBreakdown.Completed,
            feature.StatusInfo.ProgressBreakdown.Total)

        // Tasks column
        tasksDisplay := fmt.Sprintf("%d done", feature.StatusInfo.ProgressBreakdown.Completed)

        // Notes column
        notesDisplay := formatNotes(feature.StatusInfo.ActionItems)

        tableData = append(tableData, []string{
            feature.Key,
            truncate(feature.Title, 25),
            statusDisplay,
            progressDisplay,
            tasksDisplay,
            notesDisplay,
        })
    }

    _ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

func formatStatusWithContext(status string, statusInfo *FeatureStatusInfo) string {
    context := ""

    switch statusInfo.StatusExplanation.Reason {
    case "waiting_for_approval":
        context = " (waiting) ⏳"
    case "blocked":
        context = " (blocked) ⚠️"
    case "in_development":
        context = " (dev)"
    case "completed":
        context = " ✓"
    }

    return status + context
}

func formatNotes(actionItems ActionItems) string {
    notes := []string{}

    if len(actionItems.WaitingForApproval) > 0 {
        notes = append(notes, fmt.Sprintf("%d waiting", len(actionItems.WaitingForApproval)))
    }

    if len(actionItems.Blocked) > 0 {
        notes = append(notes, fmt.Sprintf("%d blocked", len(actionItems.Blocked)))
    }

    if len(notes) == 0 {
        return ""
    }

    return strings.Join(notes, ", ")
}
```

---

### Task Group 3: JSON Schema Enhancements

**Goal**: Add enhanced fields to JSON output

**Changes Needed**:

1. **Feature JSON enhancement**:
```go
func outputFeatureJSON(feature *models.Feature, tasks []*models.Task, statusInfo *FeatureStatusInfo) error {
    result := map[string]interface{}{
        // Existing fields
        "id":          feature.ID,
        "key":         feature.Key,
        "title":       feature.Title,
        "status":      feature.Status,
        "progress_pct": feature.ProgressPct,

        // NEW: Enhanced fields
        "status_source":     statusInfo.StatusSource,
        "status_explanation": statusInfo.StatusExplanation,
        "progress_breakdown": statusInfo.ProgressBreakdown,
        "status_distribution": statusInfo.StatusBreakdown,
        "action_items":       statusInfo.ActionItems,
        "health":             statusInfo.Health,

        // Existing fields
        "tasks": tasks,
    }

    return cli.OutputJSON(result)
}
```

2. **Epic JSON enhancement**: Similar structure with epic-specific fields

---

## Testing Strategy

### Unit Tests

**Test Files**:
- `internal/repository/feature_repository_test.go`
- `internal/repository/epic_repository_test.go`

**Test Cases**:

```go
func TestFeatureRepository_GetStatusInfo(t *testing.T) {
    tests := []struct {
        name           string
        tasks          []*models.Task
        wantStatus     string
        wantPhase      string
        wantReason     string
        wantActionItems int
    }{
        {
            name: "feature with waiting task",
            tasks: []*models.Task{
                {Status: "completed"},
                {Status: "completed"},
                {Status: "ready_for_approval"},
            },
            wantStatus:      "active",
            wantPhase:       "approval",
            wantReason:      "waiting_for_approval",
            wantActionItems: 1,
        },
        {
            name: "feature with blocked task",
            tasks: []*models.Task{
                {Status: "completed"},
                {Status: "blocked"},
            },
            wantStatus:      "active",
            wantPhase:       "any",
            wantReason:      "blocked",
            wantActionItems: 1,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

**Test Files**:
- `internal/cli/commands/feature_test.go`
- `internal/cli/commands/epic_test.go`

**Test Cases**:

```go
func TestFeatureGetCommand_EnhancedOutput(t *testing.T) {
    // Setup test database with feature and tasks
    // Run feature get command
    // Verify output contains:
    //   - Status explanation
    //   - Progress breakdown
    //   - Action items section
    //   - Workflow-ordered task list
}

func TestFeatureListCommand_EnhancedColumns(t *testing.T) {
    // Setup test database with features
    // Run feature list command
    // Verify table contains:
    //   - Status with context
    //   - Progress with ratio
    //   - Notes column with action items
}
```

### Manual Testing Checklist

- [ ] Feature get shows status explanation
- [ ] Feature get shows progress breakdown
- [ ] Feature get shows action items section
- [ ] Feature get shows workflow-ordered task list
- [ ] Epic get shows status explanation
- [ ] Epic get shows feature distribution
- [ ] Epic get shows aggregated action items
- [ ] Epic get shows urgency-ordered feature list
- [ ] Feature list shows status context
- [ ] Feature list shows progress ratio
- [ ] Feature list shows notes column
- [ ] Epic list shows status context
- [ ] Epic list shows feature summary
- [ ] JSON output includes all enhanced fields
- [ ] Colors follow workflow configuration
- [ ] --no-color flag works correctly
- [ ] Visual indicators display correctly

---

## Rollout Plan

### Phase 1: Repository Layer (Week 1)

**Deliverables**:
- [ ] Add `GetStatusInfo()` methods to FeatureRepository
- [ ] Add `GetStatusInfo()` methods to EpicRepository
- [ ] Add status explanation calculation logic
- [ ] Add progress breakdown calculation logic
- [ ] Add action items extraction logic
- [ ] Add health indicators calculation logic
- [ ] Write unit tests for all new methods

**Success Criteria**:
- All unit tests pass
- 100% code coverage for new methods
- Performance < 100ms for features with 50 tasks

---

### Phase 2: Feature Get Enhancement (Week 1)

**Deliverables**:
- [ ] Update `runFeatureGet()` to use status info
- [ ] Update `renderFeatureDetails()` with new sections
- [ ] Add status section rendering
- [ ] Add progress section rendering
- [ ] Add action items section rendering
- [ ] Add workflow-ordered task list rendering
- [ ] Update JSON output with enhanced fields
- [ ] Write integration tests

**Success Criteria**:
- Feature get output matches design mockups
- JSON output includes all enhanced fields
- Colors follow workflow configuration
- All integration tests pass

---

### Phase 3: Epic Get Enhancement (Week 2)

**Deliverables**:
- [ ] Update `runEpicGet()` to use status info
- [ ] Update `renderEpicDetails()` with new sections
- [ ] Add epic status section rendering
- [ ] Add feature distribution section rendering
- [ ] Add aggregated action items section rendering
- [ ] Add urgency-ordered feature list rendering
- [ ] Update JSON output with enhanced fields
- [ ] Write integration tests

**Success Criteria**:
- Epic get output matches design mockups
- JSON output includes all enhanced fields
- Feature list is urgency-ordered
- All integration tests pass

---

### Phase 4: List Views Enhancement (Week 2)

**Deliverables**:
- [ ] Update `renderFeatureListTable()` with new columns
- [ ] Update `renderEpicListTable()` with new columns
- [ ] Add status context formatting
- [ ] Add progress ratio formatting
- [ ] Add notes column formatting
- [ ] Write integration tests

**Success Criteria**:
- List outputs match design mockups
- Tables remain scannable (< 3 seconds to assess)
- Visual indicators display correctly
- All integration tests pass

---

### Phase 5: Documentation & Release (Week 3)

**Deliverables**:
- [ ] Update CLI_REFERENCE.md with new output format
- [ ] Add migration guide for JSON consumers
- [ ] Update CHANGELOG.md
- [ ] Create demo video/screenshots
- [ ] Write release notes

**Success Criteria**:
- All documentation is accurate
- Migration guide is clear
- Release notes highlight benefits

---

## Performance Considerations

### Query Optimization

**Problem**: Multiple queries to calculate status info

**Solution**: Use CTEs (Common Table Expressions) to aggregate in single query

```sql
WITH task_phase_counts AS (
    SELECT
        feature_id,
        CASE
            WHEN status IN ('ready_for_approval', 'in_approval') THEN 'approval'
            WHEN status IN ('ready_for_development', 'in_development') THEN 'development'
            WHEN status IN ('completed', 'cancelled') THEN 'done'
            WHEN status = 'blocked' THEN 'blocked'
            ELSE 'planning'
        END as phase,
        COUNT(*) as count
    FROM tasks
    WHERE feature_id = ?
    GROUP BY feature_id, phase
)
SELECT * FROM task_phase_counts;
```

### Caching Strategy

**Problem**: Status info recalculated on every request

**Solution**: Cache status info with TTL

```go
type StatusInfoCache struct {
    cache map[int64]*CacheEntry
    mu    sync.RWMutex
    ttl   time.Duration
}

type CacheEntry struct {
    StatusInfo *FeatureStatusInfo
    ExpiresAt  time.Time
}

func (c *StatusInfoCache) Get(featureID int64) (*FeatureStatusInfo, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, exists := c.cache[featureID]
    if !exists || time.Now().After(entry.ExpiresAt) {
        return nil, false
    }

    return entry.StatusInfo, true
}

func (c *StatusInfoCache) Set(featureID int64, info *FeatureStatusInfo) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.cache[featureID] = &CacheEntry{
        StatusInfo: info,
        ExpiresAt:  time.Now().Add(c.ttl),
    }
}
```

**Note**: Caching may be out of scope for initial implementation. Consider for future enhancement.

---

## Accessibility Checklist

- [ ] All colors paired with text/symbols
- [ ] Visual indicators (⏳, ⚠️, ✓) always present
- [ ] Phase labels always shown
- [ ] `--no-color` flag preserves all information
- [ ] Structured output for screen readers
- [ ] Consistent section headers
- [ ] Action items section at top
- [ ] Tables have clear headers
- [ ] Responsive layout for narrow terminals

---

## Backward Compatibility

### JSON Schema Compatibility

**Guarantee**: All new fields are additive, existing fields unchanged

**Strategy**:
- Add new fields (status_explanation, progress_breakdown, etc.)
- Never remove or rename existing fields
- Existing JSON parsers continue working

**Migration Guide**:
```markdown
# Migration Guide: Enhanced JSON Schema

## What Changed

Enhanced JSON output now includes additional fields:
- `status_explanation`: Why status has current value
- `progress_breakdown`: Detailed progress information
- `action_items`: Tasks needing attention
- `health`: Health indicators

## Backward Compatibility

All existing fields remain unchanged. Existing parsers will:
- ✅ Continue working without changes
- ✅ Safely ignore new fields
- ✅ Parse existing fields as before

## Upgrading

Optional: Update parsers to use new fields:

```python
# Before
status = feature['status']

# After (optional enhancement)
status = feature['status']
explanation = feature.get('status_explanation', {})
reason = explanation.get('reason', 'unknown')
```
```

---

## Success Metrics (Post-Launch)

### User Experience Metrics

**Measurement Method**: User surveys (1 week after launch)

- [ ] "I can assess feature health in < 5 seconds" → Target: 90% agree
- [ ] "I know what to do next" → Target: 95% agree
- [ ] "Action items are clear" → Target: 95% agree
- [ ] "Status explanations are helpful" → Target: 90% agree

### Performance Metrics

**Measurement Method**: Automated monitoring

- [ ] Feature get response time: < 100ms (p95)
- [ ] Epic get response time: < 200ms (p95)
- [ ] List command response time: < 150ms (p95)
- [ ] API call reduction: > 50% (orchestrators)

### Business Impact Metrics

**Measurement Method**: Time tracking (1 month after launch)

- [ ] Project manager time saved: > 5 hours/month
- [ ] Developer time saved: > 2 hours/month
- [ ] Orchestrator API calls: < 50% of baseline
- [ ] Support tickets related to status confusion: < 20% of baseline

---

## Related Documents

- [UX Design: Status Tracking](./D12-ux-design-status-tracking.md) - Detailed design mockups
- [Journey Map: Status Experience](./D13-journey-map-status-experience.md) - User journey analysis
- [E07-F14 Feature PRD](./prd.md) - Complete requirements
- [E07-F14 Feature Specification](./feature.md) - Feature summary

---

## Conclusion

This implementation guide provides a clear path from design to implementation for enhanced status tracking in Shark Task Manager. By following the phased approach and adhering to the design principles, we can deliver a dramatically improved user experience while maintaining backward compatibility and performance.

**Key Takeaways**:
1. Implement in phases (repository → feature get → epic get → list views)
2. Follow design principles consistently (scannable, progressive disclosure, color as signal)
3. Test thoroughly (unit, integration, manual)
4. Maintain backward compatibility (additive changes only)
5. Measure success (UX metrics, performance, business impact)

**Estimated Timeline**: 3 weeks (17 hours of development + testing + documentation)

**Risk Mitigation**: Phased rollout allows for early feedback and course correction.
