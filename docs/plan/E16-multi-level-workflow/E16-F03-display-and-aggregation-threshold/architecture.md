# E16-F03: Display and Aggregation Threshold - Architecture Plan

**Feature**: E16-F03-display-and-aggregation-threshold
**Epic**: E16 - Multi-Level Workflow System
**Depends On**: E16-F01 (Core Workflow Engine - provides level-specific workflow parsing)

---

## Overview

This feature modifies `shark epic get`, `shark feature get`, and `shark status` commands to respect the `is_planning` / aggregation threshold concept. Entities in planning states show their own workflow position and phase. Entities in aggregation states (e.g., `active`) show aggregated child progress (current behavior, unchanged).

---

## Architecture Approach

### Design Principle: Display Logic in Service, Rendering in Commands

The current codebase has a **fat controller** pattern in `epic.go` and `feature.go` - commands directly orchestrate repository calls and build display data. Per the architecture rules (E15 target), new code should use the **service layer pattern**:

- **Service**: Determines display mode (planning vs aggregation), assembles data model
- **Command**: Calls service, routes to appropriate renderer based on mode
- **Repository**: No changes needed for this feature

### Key Design Decision: DisplayInfo Pattern

Rather than modifying every existing method, introduce a new **DisplayInfo** type that wraps entity details with display-mode awareness. The service determines the mode, and the command renders accordingly.

---

## Layer-by-Layer Design

### 1. Config Layer (`internal/config/`)

**Prerequisite**: E16-F01 must add `epic_workflow` and `feature_workflow` parsing to `.sharkconfig.json`. This feature depends on being able to load level-specific workflow configs.

**New fields on `StatusMetadata`** (added by E16-F01):
```go
// Already in StatusMetadata struct:
// Phase, Color, Description, ProgressWeight, etc.

// NEW fields (E16-F01 adds these):
IsPlanning     bool   `json:"is_planning,omitempty"`
AggregatesFrom string `json:"aggregates_from,omitempty"` // "features" or "tasks"
```

**New constant** (E16-F01 adds):
```go
const AggregationStatusKey = "_aggregation_"
```

**What E16-F03 needs from E16-F01**:
- `LoadEpicWorkflow(configPath) (*WorkflowConfig, error)` - load epic workflow section
- `LoadFeatureWorkflow(configPath) (*WorkflowConfig, error)` - load feature workflow section
- `StatusMetadata.IsPlanning` field accessible
- `StatusMetadata.AggregatesFrom` field accessible
- `SpecialStatuses["_aggregation_"]` accessible

If E16-F01 is not yet built, E16-F03 can stub these by reading the raw config directly and looking for `epic_workflow` / `feature_workflow` sections.

---

### 2. Service Layer (`internal/services/`)

**New file**: `internal/services/display_service.go`

This service encapsulates the planning-vs-aggregation logic. It is the **single point** where the display mode decision is made.

```go
package services

// DisplayMode indicates whether an entity is in planning or aggregation mode
type DisplayMode string

const (
    DisplayModePlanning    DisplayMode = "planning"
    DisplayModeAggregation DisplayMode = "aggregation"
)

// EpicDisplayInfo contains all data needed to render an epic's details
type EpicDisplayInfo struct {
    Epic            *models.Epic
    Mode            DisplayMode

    // Planning mode fields (populated when Mode == Planning)
    Phase           string          // Current workflow phase
    PhaseDescription string         // Human-readable description
    WorkflowPosition *WorkflowPosition // Position in workflow chain

    // Aggregation mode fields (populated when Mode == Aggregation)
    Progress        float64
    Features        []FeatureDisplayItem
    FeatureRollup   map[string]int
    TaskRollup      map[string]int
    BlockedTasks    []*models.Task
    ApprovalBacklog int
    RelatedDocs     []*models.Document

    // Common fields
    ResolvedPath    string
    StatusSource    string // "workflow" for planning, "calculated" for aggregation
}

// FeatureDisplayItem represents a feature in the epic's feature list
type FeatureDisplayItem struct {
    Feature     *models.Feature
    TaskCount   int
    Mode        DisplayMode     // Each feature has its own mode
    Phase       string          // Workflow phase if in planning mode
}

// FeatureDisplayInfo contains all data needed to render a feature's details
type FeatureDisplayInfo struct {
    Feature         *models.Feature
    Mode            DisplayMode

    // Planning mode fields
    Phase           string
    PhaseDescription string
    WorkflowPosition *WorkflowPosition

    // Aggregation mode fields
    Tasks           []*models.Task
    StatusBreakdown []workflow.StatusCount
    ProgressInfo    *status.ProgressInfo
    WorkSummary     *status.WorkSummary
    ActionItems     *status.ActionItems
    RelatedDocs     []*models.Document

    // Common fields
    ResolvedPath    string
    StatusSource    string
}

// WorkflowPosition represents where an entity is in its workflow
type WorkflowPosition struct {
    Statuses      []string // Ordered list of all statuses in workflow
    CurrentIndex  int      // Index of current status
    CurrentStatus string   // Current status name
    Phases        []PhaseInfo // Phase groupings for display
}

// PhaseInfo groups statuses by phase for workflow position display
type PhaseInfo struct {
    Name     string
    Statuses []string
    IsCurrent bool
}
```

**New service**: `DisplayService`

```go
type DisplayService struct {
    epicRepo       *repository.EpicRepository
    featureRepo    *repository.FeatureRepository
    taskRepo       *repository.TaskRepository
    documentRepo   *repository.DocumentRepository
    workflowSvc    *workflow.Service
    epicWorkflow   *config.WorkflowConfig    // Level-specific, may be nil
    featureWorkflow *config.WorkflowConfig   // Level-specific, may be nil
    taskWorkflow   *config.WorkflowConfig    // Existing task workflow
    configPath     string
}

func NewDisplayService(
    db *repository.DB,
    configPath string,
    projectRoot string,
) *DisplayService
```

**Key methods**:

```go
// GetEpicDisplayInfo assembles all data needed to display an epic
func (s *DisplayService) GetEpicDisplayInfo(ctx context.Context, epicKey string) (*EpicDisplayInfo, error)

// GetFeatureDisplayInfo assembles all data needed to display a feature
func (s *DisplayService) GetFeatureDisplayInfo(ctx context.Context, featureKey string) (*FeatureDisplayInfo, error)

// determineDisplayMode checks entity status against workflow config
// Returns Planning if is_planning=true for current status, Aggregation otherwise
func (s *DisplayService) determineEpicDisplayMode(epic *models.Epic) DisplayMode

func (s *DisplayService) determineFeatureDisplayMode(feature *models.Feature) DisplayMode

// buildWorkflowPosition constructs the workflow position visualization data
func (s *DisplayService) buildWorkflowPosition(
    currentStatus string,
    workflowCfg *config.WorkflowConfig,
) *WorkflowPosition
```

**Display mode determination logic** (core algorithm):

```go
func (s *DisplayService) determineEpicDisplayMode(epic *models.Epic) DisplayMode {
    // If no epic workflow configured, always use aggregation (backward compat)
    if s.epicWorkflow == nil {
        return DisplayModeAggregation
    }

    // Check if current status is in _aggregation_ special statuses
    aggStatuses, exists := s.epicWorkflow.SpecialStatuses[config.AggregationStatusKey]
    if exists {
        for _, aggStatus := range aggStatuses {
            if strings.EqualFold(aggStatus, string(epic.Status)) {
                return DisplayModeAggregation
            }
        }
    }

    // Check is_planning from status metadata
    meta, found := s.epicWorkflow.GetStatusMetadata(string(epic.Status))
    if found && meta.IsPlanning {
        return DisplayModePlanning
    }

    // Default: aggregation (includes terminal statuses like completed)
    // For completed/cancelled, the display should show final aggregated state
    return DisplayModeAggregation
}
```

---

### 3. Command Layer (`internal/cli/commands/`)

**Modified files**: `epic.go`, `feature.go`, `status.go`

The commands become thin wrappers that call `DisplayService` and route to the appropriate renderer.

#### Epic Get Command Changes

```go
func runEpicGet(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    epicKey := args[0]

    // Get database and config
    repoDb, err := cli.GetDB(ctx)
    if err != nil { ... }

    configPath, _ := cli.GetConfigPath()
    projectRoot, _ := os.Getwd()

    // Create display service
    displaySvc := services.NewDisplayService(repoDb, configPath, projectRoot)

    // Get display info (service determines planning vs aggregation)
    info, err := displaySvc.GetEpicDisplayInfo(ctx, epicKey)
    if err != nil { ... }

    // Output
    if cli.GlobalConfig.JSON {
        return outputEpicJSON(info)
    }

    // Route to appropriate renderer
    switch info.Mode {
    case services.DisplayModePlanning:
        renderEpicPlanning(info)
    case services.DisplayModeAggregation:
        renderEpicAggregation(info)  // Existing renderEpicDetails refactored
    }
    return nil
}
```

#### New Renderers

**Epic Planning Mode Renderer** (`epic.go`):

```go
func renderEpicPlanning(info *EpicDisplayInfo) {
    // Epic: E16 - Multi-Level Workflow System
    // Status: in_refinement (Refinement phase)
    // Phase: refinement
    //
    // Workflow Position:
    //   draft -> research -> [in_refinement] -> decomposition -> active -> completed
    //                         ^^^^^^^^^^^
    //                         YOU ARE HERE
    //
    // No features yet (epic is still being refined).
}
```

**Feature Planning Mode Renderer** (`feature.go`):

```go
func renderFeaturePlanning(info *FeatureDisplayInfo) {
    // Feature: E16-F01 - Core Workflow Engine
    // Status: in_refinement_tech (Refinement phase)
    // Phase: refinement
    //
    // Workflow Position:
    //   draft -> refinement_ba -> [refinement_tech] -> task_gen -> build -> active -> completed
    //
    // No tasks yet (feature is still being refined).
}
```

#### Aggregation Mode: Feature-Level Planning Labels

When an epic is in aggregation mode but has features in planning states, the feature list shows `(planning)` instead of progress:

```go
// In renderEpicAggregation, when rendering features table:
for _, item := range info.Features {
    if item.Mode == services.DisplayModePlanning {
        // Show (planning) instead of progress bar
        progressCol = fmt.Sprintf("(%s)", item.Phase)  // e.g., "(refinement)"
    } else {
        progressCol = fmt.Sprintf("%.1f%%", item.Feature.ProgressPct)
    }
}
```

---

### 4. JSON Output Changes

#### Epic JSON - Planning Mode

```json
{
    "key": "E16",
    "title": "Multi-Level Workflow System",
    "status": "in_refinement",
    "display_mode": "planning",
    "is_planning": true,
    "phase": "refinement",
    "phase_description": "BA refining epic requirements",
    "workflow_position": {
        "statuses": ["draft", "ready_for_research", "in_research", ...],
        "current_index": 3,
        "current_status": "in_refinement"
    }
}
```

#### Epic JSON - Aggregation Mode (current behavior + enhancements)

```json
{
    "key": "E16",
    "title": "Multi-Level Workflow System",
    "status": "active",
    "display_mode": "aggregation",
    "is_planning": false,
    "progress_pct": 62.0,
    "features": [...],
    "feature_status_rollup": {...},
    "task_status_rollup": {...},
    "impediments": [...]
}
```

#### Feature JSON - Planning Mode

```json
{
    "key": "E16-F01",
    "title": "Core Workflow Engine",
    "status": "ready_for_refinement_tech",
    "display_mode": "planning",
    "is_planning": true,
    "phase": "refinement",
    "workflow_position": {
        "statuses": ["draft", "ready_for_refinement_ba", ...],
        "current_index": 3,
        "current_status": "ready_for_refinement_tech"
    }
}
```

---

### 5. Status Command Changes (`status.go`)

The `shark status` command uses a separate `StatusService` and `StatusDashboard` model. Changes needed:

**In `StatusService.GetDashboard()`**: When building `EpicSummary` entries, check if the epic is in planning mode:

```go
type EpicSummary struct {
    // ... existing fields ...
    DisplayMode     string  `json:"display_mode"`      // NEW: "planning" or "aggregation"
    Phase           string  `json:"phase,omitempty"`    // NEW: workflow phase if planning
    IsPlanning      bool    `json:"is_planning"`        // NEW
}
```

When an epic is in planning mode, `ProgressPercent`, `TasksTotal`, `TasksCompleted` should be 0 (or omitted), and `Phase` should reflect the workflow position.

---

### 6. Status Package Changes (`internal/status/`)

**Modified file**: `calculation_service.go`

The cascading calculation service currently derives epic/feature statuses from children. With E16-F03, this logic must respect the display mode:

- **If entity has level-specific workflow with `is_planning: true`**: Status is NOT derived from children. It's the entity's own status (set explicitly via workflow transitions).
- **If entity is in aggregation mode**: Current behavior - derive from children.

```go
// RecalculateFeatureStatus - MODIFIED
func (s *CalculationService) RecalculateFeatureStatus(ctx context.Context, featureID int64) (*StatusChangeResult, error) {
    feature, err := s.featureRepo.GetByID(ctx, featureID)
    if err != nil { return nil, err }

    // NEW: Check if feature is in planning mode
    if s.isFeatureInPlanningMode(feature) {
        // Don't derive status from tasks - feature has its own workflow
        return &StatusChangeResult{
            EntityType:     "feature",
            EntityKey:      feature.Key,
            PreviousStatus: string(feature.Status),
            NewStatus:      string(feature.Status),
            WasChanged:     false,
            WasSkipped:     true,
            SkipReason:     "feature in planning mode (is_planning=true)",
        }, nil
    }

    // ... existing aggregation logic unchanged ...
}
```

**New helper methods**:

```go
func (s *CalculationService) isFeatureInPlanningMode(feature *models.Feature) bool {
    // Load feature workflow config
    // Check if current status has is_planning: true
    // If no feature_workflow configured, return false (backward compat)
}

func (s *CalculationService) isEpicInPlanningMode(epic *models.Epic) bool {
    // Same pattern for epics
}
```

---

### 7. Derivation Logic Changes (`internal/status/derivation.go`)

**No changes to `DeriveFeatureStatus` or `DeriveEpicStatus`**. These functions still work correctly for aggregation mode. The planning mode bypass happens at the `CalculationService` level, which simply skips calling derivation when the entity is in planning mode.

---

## Backward Compatibility

### No Epic/Feature Workflow Configured

When `.sharkconfig.json` has no `epic_workflow` or `feature_workflow`:
- `determineDisplayMode()` returns `DisplayModeAggregation`
- All existing behavior is 100% preserved
- No new fields appear in output (or they appear with default values)
- Zero performance impact (the config check is a nil pointer test)

### Existing Entities at `active` Status

Entities at `active` with or without custom workflows:
- `active` is in `_aggregation_` special statuses
- `determineDisplayMode()` returns `DisplayModeAggregation`
- Current rendering behavior is identical

### `draft -> active` Shortcut

The shortcut path remains valid. When an entity goes directly from `draft` to `active`:
- `draft` has `is_planning: true` -> planning mode display
- `active` has `is_planning: false` + `aggregates_from` -> aggregation mode display
- Transition is seamless

---

## Performance Considerations

### REQ-NF-002: Planning mode display adds no additional database queries

Planning mode display is **cheaper** than aggregation mode because:
- No `CalculateProgress()` call needed
- No `ListByEpic()` / `ListByFeature()` to get children
- No `GetStatusBreakdown()` or rollup queries
- Only the entity itself is fetched + workflow config lookup (cached)

### Config Loading

Level-specific workflow configs are loaded once per command execution and cached:
- `LoadEpicWorkflow()` - cached via the same mechanism as `LoadWorkflowConfig()`
- `LoadFeatureWorkflow()` - cached similarly
- No additional file reads after first access

---

## File Changes Summary

### New Files

| File | Purpose |
|------|---------|
| `internal/services/display_service.go` | Display mode logic, data assembly |
| `internal/services/display_service_test.go` | Unit tests with mocked repos |

### Modified Files

| File | Changes |
|------|---------|
| `internal/cli/commands/epic.go` | Refactor `runEpicGet` to use DisplayService, add `renderEpicPlanning` |
| `internal/cli/commands/feature.go` | Refactor `runFeatureGet` to use DisplayService, add `renderFeaturePlanning` |
| `internal/cli/commands/status.go` | Add planning mode awareness to dashboard |
| `internal/status/calculation_service.go` | Add planning mode bypass in cascade logic |
| `internal/status/models.go` | Add `DisplayMode` and `IsPlanning` to `EpicSummary` |
| `internal/status/status.go` | Add `StatusService` planning mode awareness |

### Files NOT Modified

| File | Reason |
|------|--------|
| `internal/config/workflow_schema.go` | E16-F01 adds `IsPlanning`, `AggregatesFrom` fields |
| `internal/config/workflow_parser.go` | E16-F01 adds level-specific parsing |
| `internal/models/epic.go` | No model changes needed |
| `internal/models/feature.go` | No model changes needed |
| `internal/repository/*.go` | No repository changes needed |
| `internal/status/derivation.go` | Derivation logic unchanged |

---

## Task Breakdown

### Task 1: DisplayService Core (S)
- Create `internal/services/display_service.go`
- Implement `DisplayMode` type and mode determination logic
- Implement `buildWorkflowPosition()` for workflow chain visualization
- Handle nil workflow config (backward compat)
- Tests with mocked repos

### Task 2: Epic Get - Planning Mode (S)
- Implement `GetEpicDisplayInfo()` in DisplayService
- Add `renderEpicPlanning()` renderer in `epic.go`
- Modify `runEpicGet()` to use DisplayService
- Preserve existing aggregation rendering (refactor to `renderEpicAggregation`)
- Update JSON output with `display_mode`, `is_planning`, `phase`, `workflow_position`
- Tests

### Task 3: Feature Get - Planning Mode (S)
- Implement `GetFeatureDisplayInfo()` in DisplayService
- Add `renderFeaturePlanning()` renderer in `feature.go`
- Modify `runFeatureGet()` to use DisplayService
- Preserve existing aggregation rendering
- Update JSON output
- Tests

### Task 4: Epic Aggregation - Planning Feature Labels (XS)
- When epic is in aggregation mode, features in planning states show `(planning)` or `(phase_name)` instead of progress bar
- Modify feature list rendering in epic aggregation view
- Tests

### Task 5: Calculation Service - Planning Mode Bypass (S)
- Modify `RecalculateFeatureStatus()` to skip derivation for planning-mode features
- Modify `RecalculateEpicStatus()` to skip derivation for planning-mode epics
- Add `isFeatureInPlanningMode()` and `isEpicInPlanningMode()` helpers
- Tests

### Task 6: Status Command - Planning Awareness (S)
- Modify `StatusDashboard` / `EpicSummary` to include display mode
- Update `StatusService.GetDashboard()` to check planning mode
- Update `FormatDashboard()` to render planning mode epics differently
- Tests

### Task 7: Integration Testing & Edge Cases (M)
- Test with no custom workflow (backward compat)
- Test entity at each planning status
- Test entity at aggregation status with mixed planning/active children
- Test `draft -> active` shortcut path
- Test completed/cancelled entities
- Test JSON output for both modes

---

## Dependency Graph

```
E16-F01 (Core Workflow Engine)
    │
    ├── Provides: level-specific workflow config parsing
    ├── Provides: is_planning, aggregates_from in StatusMetadata
    └── Provides: _aggregation_ special status key
         │
         ▼
E16-F03 Task 1: DisplayService Core
    │
    ├──► Task 2: Epic Get - Planning Mode
    ├──► Task 3: Feature Get - Planning Mode
    ├──► Task 5: Calculation Service Bypass
    │
    Task 2 ──► Task 4: Planning Feature Labels (in epic aggregation view)
    │
    Task 5 ──► Task 6: Status Command Awareness
    │
    All ──► Task 7: Integration Testing
```

---

## Open Design Questions

1. **Workflow Position Visualization**: The epic PRD shows a detailed ASCII workflow position display (arrows with `[current]` marker). Should E16-F03 implement a simplified version, or defer to E16-F06 (Workflow Visualization)? **Recommendation**: Implement a basic text-based position indicator in E16-F03 (e.g., `[draft] -> [research] -> [>> in_refinement <<] -> ...`), and let E16-F06 add the rich visualization later.

2. **Feature Status in Aggregation Mode with Custom Workflow**: When a feature has a `feature_workflow` configured and is at `active`, should it still use the existing `DeriveFeatureStatus()` logic or a new config-driven approach? **Recommendation**: Keep existing derivation for aggregation mode. The `feature_workflow` is for planning-phase navigation; once in aggregation, task-based derivation takes over.

3. **DisplayService vs Modifying Existing Commands In-Place**: Creating a new `DisplayService` is cleaner architecturally but requires more refactoring of existing commands. **Recommendation**: Use `DisplayService` for the new planning mode path, but keep existing aggregation logic in commands initially. Migrate aggregation to service in a follow-up task (part of E15 service layer migration).

---

*Last Updated*: 2026-02-09
