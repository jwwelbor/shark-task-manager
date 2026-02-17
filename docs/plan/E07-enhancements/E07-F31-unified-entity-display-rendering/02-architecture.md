# Architecture Design: E07-F31 - Unified Entity Display Rendering

**Feature**: E07-F31 - Unified Entity Display Rendering
**Created**: 2026-02-16
**Status**: Technical Refinement

---

## Overview

This document defines the technical architecture for creating shared rendering infrastructure while making surgical fixes to existing display logic to add missing sections (related documents, valid transitions).

**Design Principles**:
- **Minimal Changes**: <30 lines total changes to epic.go + feature.go
- **Additive Only**: No refactoring of existing render functions
- **Foundation Work**: Create infrastructure for E15-F07 to consume later
- **Zero Database Changes**: Pure display layer work

---

## Component Design

### 1. New File: `internal/cli/commands/render_common.go`

**Location**: `internal/cli/commands/render_common.go`
**Purpose**: Shared rendering helpers for epic, feature, and task display
**Estimated Lines**: ~200-300 lines

**Package Structure**:
```go
package commands

import (
    "fmt"
    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/services"
    "github.com/pterm/pterm"
)
```

---

## Data Structures

### EntityDisplayOptions

**Purpose**: Configuration struct for unified rendering (used by E15-F07)

```go
// EntityDisplayOptions configures entity display rendering.
// TODO E15-F07: Commands will use this struct when refactored to service layer pattern.
type EntityDisplayOptions struct {
    // Entity metadata
    EntityType string // "epic", "feature", or "task"
    Key        string
    Status     string

    // Common display sections (nil/empty values skip section)
    BasicInfo          [][]string                     // Key-value pairs for info table
    ValidTransitions   []string                       // Allowed next statuses
    OrchestratorAction *config.PopulatedAction        // Next action instruction
    RelatedDocs        []*models.Document             // Related documents
    Notes              []*models.EntityNote           // Entity notes
    ContextData        *models.ContextData            // Context information

    // Entity-specific content callback
    RenderSpecific func() // Callback for entity-specific sections (features table, tasks table, etc.)
}
```

**Design Rationale**:
- **Generic Types**: Uses common types (models.Document, config.PopulatedAction) to avoid entity-specific coupling
- **Nil-Safe**: All pointer/slice fields are optional; nil values skip rendering
- **Callback Pattern**: `RenderSpecific` allows entity-specific sections to be inserted at correct position
- **Extensible**: Easy to add new common sections in future without breaking existing callers

**E15-F07 Usage Pattern** (future):
```go
// Example of how E15-F07 will use this infrastructure
func runEpicGet(cmd *cobra.Command, args []string) error {
    svc := cli.GetEpicService()
    displayInfo, err := svc.GetDisplayInfo(ctx, args[0])
    if err != nil {
        return err
    }

    // Convert service result to display options
    opts := EntityDisplayOptions{
        EntityType:         "epic",
        Key:                displayInfo.Key,
        Status:             displayInfo.Status,
        BasicInfo:          displayInfo.BasicInfo,
        ValidTransitions:   displayInfo.ValidTransitions,
        OrchestratorAction: displayInfo.OrchestratorAction,
        RelatedDocs:        displayInfo.RelatedDocs,
        RenderSpecific: func() {
            renderEpicFeatureTable(displayInfo.Features)
            renderEpicRollups(displayInfo.FeatureRollup, displayInfo.TaskRollup)
        },
    }

    RenderEntity(opts)
    return nil
}
```

---

## Helper Functions

### 1. Header Rendering

```go
// renderHeader displays the entity header section.
// Uses pterm.DefaultSection for consistent styling.
//
// Parameters:
//   - entityType: "epic", "feature", or "task"
//   - key: entity key (e.g., "E07", "E07-F01", "E07-F01-001")
func renderHeader(entityType, key string) {
    pterm.DefaultSection.Printf("%s: %s", capitalize(entityType), key)
    fmt.Println()
}
```

**Design Notes**:
- Uses existing `pterm.DefaultSection` styling for consistency
- Capitalizes entity type for display (Epic, Feature, Task)
- Adds newline for spacing (matches existing render functions)

---

### 2. Basic Info Table

```go
// renderBasicInfo renders key-value info table.
// Expects [][]string with format: []{"Label", "Value"}
//
// Parameters:
//   - info: key-value pairs for display (e.g., [["Title", "..."], ["Status", "..."]])
//
// Example:
//   info := [][]string{
//       {"Title", "User Authentication"},
//       {"Status", "active"},
//       {"Priority", "high"},
//   }
//   renderBasicInfo(info)
func renderBasicInfo(info [][]string) {
    if len(info) == 0 {
        return
    }
    _ = pterm.DefaultTable.WithData(info).Render()
    fmt.Println()
}
```

**Design Notes**:
- Generic table rendering (no entity-specific knowledge)
- Caller controls what rows to include
- Skips rendering if empty (graceful degradation)
- Uses existing `pterm.DefaultTable` for consistency

---

### 3. Valid Transitions Display

```go
// renderValidTransitions displays allowed next statuses.
// Shows a simple list of valid transitions from current status.
//
// Parameters:
//   - status: current status (for context display)
//   - transitions: list of allowed next statuses
//
// Example:
//   renderValidTransitions("in_progress", []string{"ready_for_review", "blocked"})
//   // Output:
//   // Valid Transitions
//   // ━━━━━━━━━━━━━━━━━━
//   //   - ready_for_review
//   //   - blocked
func renderValidTransitions(status string, transitions []string) {
    if len(transitions) == 0 {
        return
    }

    pterm.DefaultSection.Println("Valid Transitions")
    for _, transition := range transitions {
        fmt.Printf("  - %s\n", transition)
    }
    fmt.Println()
}
```

**Design Notes**:
- Simple bulleted list (no table overhead for 1-5 items)
- Skips section if no transitions (terminal statuses, missing config)
- Status parameter available for future enhancement (e.g., "From in_progress:")
- Consistent spacing with other sections

---

### 4. Orchestrator Action Display

```go
// renderOrchestratorAction displays next action instruction.
// Reuses existing displayOrchestratorAction implementation.
//
// Parameters:
//   - action: populated action from workflow config (nil if no action configured)
//
// Note: This is a thin wrapper around existing displayOrchestratorAction()
// to maintain consistency with current rendering while making it reusable.
func renderOrchestratorAction(action *config.PopulatedAction) {
    // Delegate to existing implementation in orchestrator_display.go
    displayOrchestratorAction(action)
}
```

**Design Notes**:
- Thin wrapper around existing `displayOrchestratorAction()` function
- Maintains backward compatibility
- Allows calls from both planning and aggregation mode renders
- No duplication of orchestrator display logic

---

### 5. Related Documents Display

```go
// renderRelatedDocuments displays list of related documents.
// Shows document title, type, and file path.
//
// Parameters:
//   - docs: list of related documents ([]*models.Document)
//
// Example:
//   docs := []*models.Document{
//       {Title: "PRD", Type: "prd", FilePath: "docs/plan/E07-F31/feature.md"},
//       {Title: "Research Report", Type: "research", FilePath: "docs/plan/E07-F31/research.md"},
//   }
//   renderRelatedDocuments(docs)
func renderRelatedDocuments(docs []*models.Document) {
    if len(docs) == 0 {
        return
    }

    pterm.DefaultSection.Println("Related Documents")
    for _, doc := range docs {
        fmt.Printf("  - %s (%s)\n", doc.Title, doc.FilePath)
    }
    fmt.Println()
}
```

**Design Notes**:
- Matches existing related docs format in `renderEpicDetails()` (line 788-795)
- Omits type field to keep display concise (type usually clear from title)
- Skips section if no docs (graceful degradation)
- Consistent bullet list format

---

### 6. Notes Display

```go
// renderNotes displays entity notes with truncation.
// Shows most recent 10 notes by default.
//
// Parameters:
//   - notes: list of entity notes ([]*models.EntityNote)
//
// Format: [type] date  content (truncated to 80 chars)
func renderNotes(notes []*models.EntityNote) {
    if len(notes) == 0 {
        return
    }

    maxDisplay := 10
    totalNotes := len(notes)
    if totalNotes > maxDisplay {
        pterm.DefaultSection.Printf("Notes (showing %d of %d)", maxDisplay, totalNotes)
    } else {
        pterm.DefaultSection.Printf("Notes (%d)", totalNotes)
    }
    fmt.Println()

    displayCount := totalNotes
    if displayCount > maxDisplay {
        displayCount = maxDisplay
    }
    for i := totalNotes - displayCount; i < totalNotes; i++ {
        note := notes[i]
        dateStr := note.CreatedAt.Format("2006-01-02")
        content := note.Content
        if len(content) > 80 {
            content = content[:77] + "..."
        }
        fmt.Printf("  [%s] %s  %s\n", note.NoteType, dateStr, content)
    }
    fmt.Println()
}
```

**Design Notes**:
- Matches existing notes rendering in `renderEpicDetails()` (lines 864-889)
- Shows most recent notes first (reversed chronological)
- Truncates long content to 80 characters
- Clear indication when notes are truncated ("showing 10 of 25")

---

### 7. Context Data Display

```go
// renderContextData displays context information.
// Delegates to existing printContextData() implementation.
//
// Parameters:
//   - contextData: context data structure (*models.ContextData)
//
// Note: Only renders if context has actual content (progress, decisions, questions, etc.)
func renderContextData(contextData *models.ContextData) {
    if contextData == nil {
        return
    }

    // Check if context has any content
    hasContent := contextData.Progress != nil ||
        len(contextData.ImplementationDecisions) > 0 ||
        len(contextData.OpenQuestions) > 0 ||
        len(contextData.Blockers) > 0 ||
        len(contextData.AcceptanceCriteriaStatus) > 0

    if !hasContent {
        return
    }

    pterm.DefaultSection.Println("Context")
    fmt.Println()
    printContextData(contextData)
}
```

**Design Notes**:
- Reuses existing `printContextData()` function (defined elsewhere in commands package)
- Skips section if context is nil or empty
- Content check prevents rendering empty section headers

---

### 8. Valid Transitions Extraction Helper

```go
// GetValidTransitions extracts valid next statuses from workflow config.
// Returns empty array if status not found in status_flow or config is nil.
//
// Parameters:
//   - status: current status string
//   - workflow: workflow configuration (*config.WorkflowConfig)
//
// Returns:
//   - []string: list of allowed next statuses (empty if not found)
//
// Example:
//   cfg, _ := config.LoadWorkflowConfig(".sharkconfig.json")
//   transitions := GetValidTransitions("in_progress", cfg)
//   // Returns: ["ready_for_review", "blocked", "on_hold"]
func GetValidTransitions(status string, workflow *config.WorkflowConfig) []string {
    if workflow == nil {
        return []string{}
    }

    transitions, ok := workflow.StatusFlow[status]
    if !ok {
        return []string{}
    }

    return transitions
}
```

**Design Notes**:
- Simple lookup in `workflow.StatusFlow` map
- Nil-safe: returns empty array for nil config
- Returns empty array for terminal statuses (not in status_flow)
- No special error handling needed (empty array is valid response)

**Integration Pattern**:
```go
// How epic.go will use this helper
workflowCfg, _ := config.LoadWorkflowConfig(configPath)
validTransitions := GetValidTransitions(epic.Status, workflowCfg)
// Include validTransitions in JSON response
```

---

### 9. Unified Entity Renderer (Foundation for E15-F07)

```go
// RenderEntity renders a complete entity display using EntityDisplayOptions.
// This function orchestrates section rendering in standard order.
//
// TODO E15-F07: Commands will use this function when refactored to service layer pattern.
// For now, this establishes the pattern and section ordering for future consolidation.
//
// Standard section order:
//   1. Header (entity type + key)
//   2. Basic Info (key-value table)
//   3. Valid Transitions (allowed next statuses)
//   4. Orchestrator Action (next action instruction)
//   5. Related Documents (linked artifacts)
//   6. [Entity-Specific Sections via RenderSpecific callback]
//   7. Notes (entity notes)
//   8. Context Data (progress, decisions, questions)
//
// Parameters:
//   - opts: EntityDisplayOptions struct with all display configuration
//
// Example:
//   opts := EntityDisplayOptions{
//       EntityType: "feature",
//       Key: "E07-F01",
//       Status: "active",
//       BasicInfo: [][]string{{"Title", "..."}},
//       ValidTransitions: []string{"completed", "on_hold"},
//       RenderSpecific: func() {
//           renderFeatureTasksTable(tasks)
//       },
//   }
//   RenderEntity(opts)
func RenderEntity(opts EntityDisplayOptions) {
    // 1. Header
    renderHeader(opts.EntityType, opts.Key)

    // 2. Basic Info
    renderBasicInfo(opts.BasicInfo)

    // 3. Valid Transitions
    renderValidTransitions(opts.Status, opts.ValidTransitions)

    // 4. Orchestrator Action
    renderOrchestratorAction(opts.OrchestratorAction)

    // 5. Related Documents
    renderRelatedDocuments(opts.RelatedDocs)

    // 6. Entity-Specific Sections (callback)
    if opts.RenderSpecific != nil {
        opts.RenderSpecific()
    }

    // 7. Notes
    renderNotes(opts.Notes)

    // 8. Context Data
    renderContextData(opts.ContextData)
}
```

**Design Notes**:
- Establishes standard section ordering for all entities
- Graceful degradation: skips nil/empty sections automatically
- Callback pattern allows entity-specific content at correct position
- Clear TODO comments for E15-F07 adoption
- Not required to be used immediately (infrastructure only)

---

## Surgical Changes to Existing Render Functions

### Epic Planning Mode: Add Related Docs

**File**: `internal/cli/commands/epic.go`
**Function**: `renderEpicPlanning()`
**Line**: After orchestrator action display (~line 715)
**Change**: Add 1 function call

```go
// After displayOrchestratorAction(info.OrchestratorAction) at line 715
// Add this:
renderRelatedDocuments(info.RelatedDocs)
```

**Justification**:
- Planning mode currently doesn't show related docs (50% visibility gap)
- Single line addition uses new helper function
- No refactoring of existing logic
- Maintains existing section order

---

### Epic JSON Output: Add Valid Transitions

**File**: `internal/cli/commands/epic.go`
**Function**: `runEpicGet()` JSON output path
**Line**: In JSON response construction (~line 619-644)
**Change**: Add 2-3 lines for valid transitions

```go
// In runEpicGet(), before return cli.OutputJSON(result) at line 645
// Add these lines after line 644:

// Get valid transitions for epic status
workflowCfg, _ := config.LoadWorkflowConfig(configPath) // configPath already available in function
validTransitions := GetValidTransitions(string(epic.Status), workflowCfg)

result := map[string]interface{}{
    // ... existing fields ...
    "orchestrator_action":    displaySvc.ResolveEpicAction(ctx, epic),
    "valid_transitions":      validTransitions, // NEW FIELD
}
```

**Justification**:
- Adds missing valid_transitions field to JSON
- Uses new GetValidTransitions() helper
- Backward compatible (additive only)
- ~3 lines of code

---

### Feature Planning Mode: Add Related Docs

**File**: `internal/cli/commands/feature.go`
**Function**: `renderFeaturePlanning()`
**Line**: After orchestrator action display (~line 865)
**Change**: Add 1 function call

```go
// After displayOrchestratorAction(info.OrchestratorAction) at line 865
// Add this:
renderRelatedDocuments(info.RelatedDocs)
```

**Justification**:
- Same as epic planning mode
- Planning mode doesn't show related docs currently
- Single line addition
- No refactoring needed

---

### Feature JSON Output: Add Valid Transitions

**File**: `internal/cli/commands/feature.go`
**Function**: `runFeatureGet()` JSON output path
**Line**: In JSON response construction (~line 772-796)
**Change**: Add 2-3 lines for valid transitions

```go
// In runFeatureGet(), before return cli.OutputJSON(result) at line 797
// Add these lines after line 796:

// Get valid transitions for feature status
validTransitions := GetValidTransitions(string(feature.Status), workflowCfg) // workflowCfg already loaded at line 697

result := map[string]interface{}{
    // ... existing fields ...
    "orchestrator_action": displaySvc.ResolveFeatureAction(ctx, feature),
    "valid_transitions":   validTransitions, // NEW FIELD
}
```

**Justification**:
- Adds missing valid_transitions field to JSON
- workflowCfg already loaded in function (line 697)
- Backward compatible
- ~2 lines of code (config already available)

---

## Integration Points

### 1. DisplayService

**Location**: `internal/services/display_service.go`
**Usage**: Provides `OrchestratorAction` data for rendering
**Interface**:
```go
type DisplayService interface {
    ResolveEpicAction(ctx context.Context, epic *models.Epic) *config.PopulatedAction
    ResolveFeatureAction(ctx context.Context, feature *models.Feature) *config.PopulatedAction
    GetEpicDisplayInfo(ctx context.Context, key string) (*EpicDisplayInfo, error)
    GetFeatureDisplayInfo(ctx context.Context, key string) (*FeatureDisplayInfo, error)
}
```

**Changes Needed**: **NONE**
**Reason**: DisplayService already provides all needed data. Rendering helpers just format it.

---

### 2. workflow.Service

**Location**: `internal/workflow/service.go`
**Usage**: Provides status flow configuration
**Interface**:
```go
type Service struct {
    workflow *config.WorkflowConfig
}

func (s *Service) GetWorkflow() *config.WorkflowConfig
```

**Changes Needed**: **NONE**
**Reason**: `GetValidTransitions()` helper accepts `*config.WorkflowConfig` directly. Commands load config and pass it in.

---

### 3. DocumentRepository

**Location**: `internal/repository/document_repository.go`
**Usage**: Fetches related documents (already called by DisplayService)
**Interface**:
```go
type DocumentRepository interface {
    ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error)
    ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error)
}
```

**Changes Needed**: **NONE**
**Reason**: DisplayService already fetches related docs. Rendering helpers just display them.

---

### 4. pterm Library

**Usage**: Terminal styling and table rendering
**Functions Used**:
- `pterm.DefaultSection.Printf()` - Section headers
- `pterm.DefaultTable.WithData().Render()` - Info tables
- Existing usage pattern maintained

**Changes Needed**: **NONE**
**Reason**: Helpers use same pterm patterns as existing render functions.

---

## Performance Considerations

### No New Database Queries

**Constraint**: REQ-NF-001 - No additional database queries
**Compliance**:
- All helpers format data already fetched
- `GetValidTransitions()` reads from in-memory config (no DB access)
- Related docs already fetched by DisplayService
- Zero new repository calls

**Verification**:
```go
// Before E07-F31:
// runEpicGet() calls:
//   - epicRepo.GetByKey()         ✓ existing
//   - documentRepo.ListForEpic()  ✓ existing
//   - displaySvc.GetEpicDisplayInfo() ✓ existing

// After E07-F31:
// runEpicGet() calls:
//   - epicRepo.GetByKey()         ✓ existing
//   - documentRepo.ListForEpic()  ✓ existing
//   - displaySvc.GetEpicDisplayInfo() ✓ existing
//   - GetValidTransitions()       ✓ in-memory config read only

// Zero new database queries introduced
```

---

### Rendering Time Overhead

**Constraint**: REQ-NF-002 - < 5ms overhead
**Compliance**:
- Helpers are pure formatting (string operations)
- No I/O operations (just fmt.Printf)
- Same pterm calls as existing code (just reorganized)
- Overhead: ~0.1ms per helper function

**Expected Impact**:
- Related docs display: +1-2ms (same as existing section in aggregation mode)
- Valid transitions: +0.1ms (simple list iteration)
- Total overhead: <3ms (well under 5ms target)

---

## Backward Compatibility

### JSON Output Changes

**Changes**: Additive only
- **New Fields**: `valid_transitions` (array of strings)
- **Existing Fields**: All preserved, unchanged

**Backward Compatibility**:
```json
// Before E07-F31:
{
  "id": 1,
  "key": "E07",
  "status": "active",
  "orchestrator_action": {...}
}

// After E07-F31:
{
  "id": 1,
  "key": "E07",
  "status": "active",
  "orchestrator_action": {...},
  "valid_transitions": ["completed", "on_hold"]  // NEW FIELD (additive)
}
```

**Consumer Impact**:
- Existing JSON parsers: No changes needed (ignores new field)
- New consumers: Can use valid_transitions if available
- Schema versioning: Not required (additive change)

---

### Human-Readable Output Changes

**Changes**: Additive sections only
- **New Sections**:
  - "Related Documents" in planning mode (epic and feature)
  - "Valid Transitions" in all modes (not yet implemented, pending Story 2)
- **Existing Sections**: All preserved in same order

**User Impact**:
- More information visible (improved UX)
- No sections removed or reordered
- Consistent display patterns

---

## Error Handling

### Graceful Degradation

All helper functions handle missing/nil data gracefully:

```go
// Example: renderRelatedDocuments()
func renderRelatedDocuments(docs []*models.Document) {
    if len(docs) == 0 {
        return  // Skips section silently
    }
    // ... render section
}

// Example: GetValidTransitions()
func GetValidTransitions(status string, workflow *config.WorkflowConfig) []string {
    if workflow == nil {
        return []string{}  // Empty array, not nil
    }
    transitions, ok := workflow.StatusFlow[status]
    if !ok {
        return []string{}  // Terminal status, no transitions
    }
    return transitions
}
```

**Error Scenarios**:
1. **Missing config**: Returns empty valid_transitions array
2. **No related docs**: Skips related docs section
3. **Nil orchestrator action**: Displays "None configured"
4. **Terminal status**: Returns empty valid_transitions (valid scenario)

**No Crashes**: All helpers are nil-safe and fail gracefully.

---

## File Impact Summary

### New Files (1)
- `internal/cli/commands/render_common.go` (~200-300 lines)

### Modified Files (2)
- `internal/cli/commands/epic.go` (~10-15 lines changed)
  - Add `renderRelatedDocuments()` call in planning mode
  - Add `valid_transitions` to JSON output
- `internal/cli/commands/feature.go` (~10-15 lines changed)
  - Add `renderRelatedDocuments()` call in planning mode
  - Add `valid_transitions` to JSON output

### Total Impact
- **New code**: ~250 lines (render_common.go)
- **Modified code**: ~25 lines (epic.go + feature.go)
- **Files touched**: 3 files
- **Risk**: LOW (additive changes, no refactoring)

---

## Testing Strategy

### Unit Tests (render_common_test.go)

**Coverage Target**: 100% of helper functions

**Test Cases**:
1. **renderHeader()**: Entity type capitalization, key display
2. **renderBasicInfo()**: Empty array, single row, multiple rows
3. **renderValidTransitions()**: Empty array, single transition, multiple transitions
4. **renderRelatedDocuments()**: Nil array, empty array, multiple docs
5. **renderNotes()**: Empty, <10 notes, >10 notes (truncation)
6. **renderContextData()**: Nil data, empty data, populated data
7. **GetValidTransitions()**: Nil config, missing status, valid status, terminal status

**Example Test**:
```go
func TestGetValidTransitions(t *testing.T) {
    tests := []struct {
        name     string
        status   string
        workflow *config.WorkflowConfig
        want     []string
    }{
        {
            name:     "nil config returns empty",
            status:   "in_progress",
            workflow: nil,
            want:     []string{},
        },
        {
            name:   "terminal status returns empty",
            status: "completed",
            workflow: &config.WorkflowConfig{
                StatusFlow: map[string][]string{
                    "in_progress": {"ready_for_review", "blocked"},
                },
            },
            want: []string{},
        },
        {
            name:   "valid status returns transitions",
            status: "in_progress",
            workflow: &config.WorkflowConfig{
                StatusFlow: map[string][]string{
                    "in_progress": {"ready_for_review", "blocked"},
                },
            },
            want: []string{"ready_for_review", "blocked"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := GetValidTransitions(tt.status, tt.workflow)
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

### Integration Tests

**Test Scenarios**:
1. Epic in planning mode displays related docs
2. Feature in planning mode displays related docs
3. JSON output includes valid_transitions field
4. Related docs section skipped when empty
5. Valid transitions section skipped when empty

**Not Required** (deferred to E15-F07):
- Full `RenderEntity()` integration tests (infrastructure not used yet)
- Consistency tests across epic/feature/task (E15-F07 scope)

---

## Migration Path to E15-F07

### Phase 1 (E07-F31 - This Feature)
1. Create `render_common.go` with all helpers and `RenderEntity()`
2. Add surgical fixes to epic.go and feature.go
3. Infrastructure available but not required to be used

### Phase 2 (E15-F07 - Service Layer Refactoring)
1. Refactor epic/feature/task commands to call services
2. Services return `EntityDisplayOptions` structs
3. Commands call `RenderEntity()` instead of custom render functions
4. Delete `renderEpicPlanning()`, `renderEpicDetails()`, etc.
5. 500+ lines of duplicate code eliminated

**E15-F07 Refactoring Pattern**:
```go
// Before E15-F07 (current):
func runEpicGet(cmd *cobra.Command, args []string) error {
    // 50+ lines of repository calls, logic, display construction
    if planning {
        renderEpicPlanning(info)
    } else {
        renderEpicDetails(epic, ...)
    }
}

// After E15-F07 (target):
func runEpicGet(cmd *cobra.Command, args []string) error {
    svc := cli.GetEpicService()
    opts, err := svc.GetDisplayOptions(ctx, args[0])
    if err != nil {
        return err
    }
    RenderEntity(opts)  // Single unified renderer
}
```

---

## Appendix: Helper Function Signatures Reference

```go
// Header
func renderHeader(entityType, key string)

// Basic Info
func renderBasicInfo(info [][]string)

// Valid Transitions
func renderValidTransitions(status string, transitions []string)

// Orchestrator Action
func renderOrchestratorAction(action *config.PopulatedAction)

// Related Documents
func renderRelatedDocuments(docs []*models.Document)

// Notes
func renderNotes(notes []*models.EntityNote)

// Context Data
func renderContextData(contextData *models.ContextData)

// Valid Transitions Extraction
func GetValidTransitions(status string, workflow *config.WorkflowConfig) []string

// Unified Renderer (E15-F07 target)
func RenderEntity(opts EntityDisplayOptions)
```

---

**Document Status**: Ready for Technical Review
**Next Step**: Create 06-security-performance.md
