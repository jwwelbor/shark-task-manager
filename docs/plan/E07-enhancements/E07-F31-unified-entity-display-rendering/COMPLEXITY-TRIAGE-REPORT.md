# Complexity Triage Report: E07-F31

**Feature**: E07-F31 - Unified Entity Display Rendering
**Epic**: E07 - Enhancements
**Assessment Date**: 2026-02-16
**Assessed By**: Researcher Agent

---

## Executive Summary

**Complexity Tier**: STANDARD
**Complexity Score**: 4/18 points
**Routing Decision**: ready_for_refinement_ba

**One-Line Rationale**: Feature creates shared rendering infrastructure (new file + helpers) and makes surgical fixes to existing render functions without requiring data model changes, API modifications, or complex UI work.

---

## Dimension Scoring Matrix

### (1) File Impact: **2 points** (4-10 files)

**Files to Create:**
- `internal/cli/commands/render_common.go` (new, ~200-300 lines)
- `internal/cli/commands/render_common_test.go` (new test file)

**Files to Modify:**
- `internal/cli/commands/epic.go` (~10-15 lines per spec)
- `internal/cli/commands/feature.go` (~10-15 lines per spec)
- `docs/CLI_REFERENCE.md` (documentation update)

**Files for Reference/Testing:**
- `internal/services/display_service.go` (reference only)
- `internal/cli/commands/orchestrator_display.go` (reference pattern)

**Total Estimated Files**: 5-7 files
**Score**: 2 points (4-10 file range)

---

### (2) Pattern Novelty: **1 point** (adapt existing)

**Existing Patterns Identified:**
- ✅ `DisplayService` already exists with planning/aggregation mode detection
- ✅ `orchestrator_display.go` already has shared display functions (`displayOrchestratorAction`)
- ✅ Rendering follows established pterm patterns throughout codebase
- ✅ Helper function pattern used in other CLI command files

**New Patterns Introduced:**
- `EntityDisplayOptions` struct (configuration object for display)
- `RenderEntity()` function (unified renderer for future use in E15-F07)
- Helper functions: `renderHeader()`, `renderBasicInfo()`, `renderValidTransitions()`, etc.

**Assessment**: Adapting and consolidating existing patterns into reusable helpers. Not inventing new architecture - establishing infrastructure that follows current conventions.

**Score**: 1 point (adapt existing)

---

### (3) Data Model: **0 points** (no schema change)

**Database Impact:**
- ❌ No new tables
- ❌ No schema modifications
- ❌ No new columns or indexes
- ✅ Works with existing models: `Epic`, `Feature`, `Task`, `Document`, `ContextData`

**Score**: 0 points

---

### (4) API Surface: **0 points** (no API change)

**API Impact:**
- ❌ No HTTP endpoints (CLI-only feature)
- ❌ No command signature changes
- ❌ No new flags or arguments
- ✅ JSON output enhanced (adds fields, preserves existing - backward compatible)
  - Adds: `valid_transitions` field
  - Preserves: All existing JSON structure

**Backward Compatibility**: 100% - only additive changes

**Score**: 0 points

---

### (5) Cross-Feature Dependencies: **1 point** (1-2 feature deps)

**Direct Dependencies:**
1. **DisplayService** (`internal/services/display_service.go`)
   - Provides: Display mode detection (planning vs aggregation)
   - Provides: Orchestrator action resolution
   - Provides: Workflow position building

2. **workflow.Service** (`internal/workflow/`)
   - Provides: Valid transition calculation (`GetValidTransitions()`)
   - Provides: Status validation

3. **DocumentRepository** (`internal/repository/document_repository.go`)
   - Provides: Related documents data (`ListForEpic`, `ListForFeature`)

**Coordination Points:**
- E15-F07 will use `RenderEntity()` infrastructure later (foundation work)

**Dependency Graph:**
```
render_common.go
    ├─ DisplayService (existing)
    ├─ workflow.Service (existing)
    └─ DocumentRepository (existing)
```

**Assessment**: Limited dependencies on existing, stable services. No circular dependencies. Coordination with E15-F07 is one-way (this creates foundation, E15-F07 consumes it later).

**Score**: 1 point

---

### (6) UI Complexity: **0 points** (modify existing components)

**Display Work:**
- ✅ Modifying existing render functions (`renderEpicPlanning`, `renderFeaturePlanning`)
- ✅ Adding helper functions to existing patterns
- ✅ Terminal output only (pterm library already in use)
- ❌ No new interactive UI
- ❌ No complex state management
- ❌ No new display frameworks

**Changes Summary:**
- Add related documents section to planning mode (existing pattern)
- Add valid transitions display (new helper, simple list output)
- Create reusable helpers for common sections

**Score**: 0 points

---

## Total Complexity Score

| Dimension | Score | Weight | Description |
|-----------|-------|--------|-------------|
| File Impact | 2 | / 3 | 5-7 files affected |
| Pattern Novelty | 1 | / 3 | Adapting existing patterns |
| Data Model | 0 | / 3 | No schema changes |
| API Surface | 0 | / 3 | CLI-only, backward compatible |
| Cross-Feature Deps | 1 | / 3 | 1-2 existing service deps |
| UI Complexity | 0 | / 3 | Modify existing terminal output |
| **TOTAL** | **4** | **/ 18** | **STANDARD tier** |

---

## Tier Assignment

```
Scoring Tiers:
├─ 0-3 points  → SIMPLE
├─ 4-7 points  → STANDARD  ✓ (E07-F31 = 4 points)
└─ 8+ points   → COMPLEX
```

**Assigned Tier**: **STANDARD**

---

## Routing Decision

**Status Transition**: `ready_for_triage` → `ready_for_refinement_ba`

**Rationale**:
- **NOT SIMPLE** (≤3 points): Involves creating shared infrastructure and adapting patterns across multiple files
- **NOT COMPLEX** (≥8 points): No data model changes, API changes, or novel architecture
- **STANDARD** (4-7 points): Well-defined scope, adapts existing patterns, establishes foundation for future work

**Next Phase**: Business Analyst refinement
- Validate acceptance criteria
- Clarify edge cases for rendering functions
- Document integration test scenarios
- Review JSON output structure changes

---

## Integration Points Analysis

### 1. DisplayService Integration
**File**: `internal/services/display_service.go`
**Used For**:
- Display mode detection (planning vs aggregation)
- Orchestrator action resolution
- Workflow position calculation

**Integration Pattern**:
```go
// New helper in render_common.go will call:
displaySvc := cli.GetDisplayService()
action := displaySvc.ResolveEpicAction(ctx, epic)
renderOrchestratorAction(action)
```

### 2. Workflow Service Integration
**File**: `internal/workflow/service.go`
**Used For**:
- Valid transition calculation
- Status flow traversal

**Integration Pattern**:
```go
// New helper function:
func GetValidTransitions(status string, workflow *config.WorkflowConfig) []string {
    transitions, ok := workflow.StatusFlow[status]
    if !ok {
        return []string{}
    }
    return transitions
}
```

### 3. Document Repository Integration
**File**: `internal/repository/document_repository.go`
**Used For**:
- Fetching related documents for display

**Integration Pattern**:
```go
// Already used in DisplayService:
relatedDocs, err := docRepo.ListForFeature(ctx, featureID)
// render_common.go will format for display:
renderRelatedDocuments(relatedDocs)
```

### 4. Command File Integration
**Files**: `epic.go`, `feature.go`
**Changes**: Minimal (10-15 lines each)

**Integration Pattern**:
```go
// In renderEpicPlanning() - add ONE call:
renderRelatedDocuments(info.RelatedDocs)

// In runEpicGet() JSON output - add ONE field:
validTransitions := GetValidTransitions(epic.Status, workflow)
// Include in JSON response
```

---

## File Impact Estimate (Detailed)

### New Files (2)
1. **internal/cli/commands/render_common.go**
   - Lines: ~200-300
   - Contains: EntityDisplayOptions, RenderEntity(), 8 helper functions

2. **internal/cli/commands/render_common_test.go**
   - Lines: ~300-400
   - Contains: Unit tests for all helper functions

### Modified Files (2-3)
1. **internal/cli/commands/epic.go**
   - Lines changed: ~10-15
   - Changes: Add related docs call, add valid transitions to JSON

2. **internal/cli/commands/feature.go**
   - Lines changed: ~10-15
   - Changes: Add related docs call, add valid transitions to JSON

3. **docs/CLI_REFERENCE.md** (possibly)
   - Lines changed: ~10-20
   - Changes: Document `valid_transitions` field in JSON output

### Reference Files (No Changes, Review Only)
- `internal/services/display_service.go` (existing patterns)
- `internal/cli/commands/orchestrator_display.go` (existing helper pattern)
- `internal/templates/renderer.go` (template system reference)

### Total Impact
- **New code**: ~500-700 lines (render_common.go + tests)
- **Modified code**: ~30-50 lines (epic.go + feature.go + docs)
- **Files touched**: 5-7 files
- **Risk**: LOW (surgical changes, no refactoring of existing functions per spec)

---

## Pattern Analysis

### Current Rendering Patterns (Identified)

**Duplication Issues** (per feature spec):
- 6 separate render functions across 3 entity types:
  - `renderEpicPlanning()` + `renderEpicDetails()` in epic.go
  - `renderFeaturePlanning()` + `renderFeatureDetails()` in feature.go
  - Inline rendering in task.go
- ~500+ lines of duplicated rendering logic

**Rendering Split**:
- Two render paths per entity: planning mode vs aggregation mode
- Based on `is_planning` flag from DisplayService
- Related documents inconsistently shown across modes

**Current Helper Pattern** (orchestrator_display.go):
```go
// Existing shared helper:
func displayOrchestratorAction(action *config.PopulatedAction) {
    if action == nil {
        fmt.Println("Next Action: None configured")
        return
    }
    fmt.Println("\nNext Action:")
    fmt.Printf("  Type: %s\n", action.Action)
    // ... formatting logic
}
```

### New Pattern (render_common.go)

**Infrastructure Pattern**:
```go
// Configuration object for unified rendering (E15-F07 will use this)
type EntityDisplayOptions struct {
    EntityType         string
    Key                string
    Status             string
    BasicInfo          [][]string
    ValidTransitions   []string
    OrchestratorAction *services.OrchestratorAction
    RelatedDocs        []*models.Document
    Notes              []*models.EntityNote
    ContextData        *models.ContextData
    RenderSpecific     func()  // Callback for entity-specific sections
}

// Unified renderer (foundation for E15-F07)
func RenderEntity(opts EntityDisplayOptions) {
    // Common section order:
    // 1. Header
    // 2. Basic Info
    // 3. Valid Transitions
    // 4. Orchestrator Action
    // 5. Related Documents
    // [Entity-specific callback inserted here]
    // 6. Notes
    // 7. Context Data
}
```

**Helper Pattern**:
```go
// Reusable helpers (available immediately for use)
func renderHeader(entityType, key string) { /* ... */ }
func renderBasicInfo(info [][]string) { /* ... */ }
func renderValidTransitions(status string, transitions []string) { /* ... */ }
func renderOrchestratorAction(action *services.OrchestratorAction) { /* ... */ }
func renderRelatedDocuments(docs []*models.Document) { /* ... */ }
func renderNotes(notes []*models.EntityNote) { /* ... */ }
func renderContextData(data *models.ContextData) { /* ... */ }
```

**Extraction Helper**:
```go
// Workflow integration helper
func GetValidTransitions(status string, workflow *config.WorkflowConfig) []string {
    transitions, ok := workflow.StatusFlow[status]
    if !ok {
        return []string{}
    }
    return transitions
}
```

### Pattern Adaptation Strategy

**Phase 1 (This Feature - E07-F31)**:
- Create infrastructure (EntityDisplayOptions, RenderEntity, helpers)
- Make minimal changes to existing render functions
- Add missing sections (related docs, valid transitions)
- Infrastructure available but not required to use yet

**Phase 2 (Future - E15-F07)**:
- Consolidate `renderEpicPlanning()` + `renderEpicDetails()` using `RenderEntity()`
- Consolidate `renderFeaturePlanning()` + `renderFeatureDetails()` using `RenderEntity()`
- Refactor task.go rendering
- Remove duplicated logic

**Why Two Phases?**
- E07-F31: Fix immediate bugs, establish patterns (LOW RISK)
- E15-F07: Major refactoring when service layer is ready (ALIGNED WITH ARCHITECTURE CHANGES)

---

## Risk Assessment

### Low Risks ✅

1. **Backward Compatibility**: Only additive JSON changes
2. **Existing Functionality**: Minimal changes to existing render functions (<20 lines each)
3. **Pattern Reuse**: Adapting proven patterns from existing code
4. **Dependencies**: Using stable, existing services

### Medium Risks ⚠️

1. **Consistency**: Need to ensure helper functions produce consistent output
   - **Mitigation**: Comprehensive unit tests for each helper

2. **Adoption**: Infrastructure created but not immediately used everywhere
   - **Mitigation**: Document TODO comments pointing to E15-F07 for full adoption

### Negligible Risks ✓

1. **Performance**: No new queries, same data access patterns
2. **Data Loss**: No database changes, no migration needed
3. **Breaking Changes**: Backward compatible by design

---

## Recommendations

### For BA Refinement Phase (Next Step)

1. **Validate Acceptance Criteria**:
   - [ ] Confirm related docs display format (list, table, inline?)
   - [ ] Clarify valid transitions display format (list, inline?)
   - [ ] Review JSON structure changes with consumers

2. **Edge Case Documentation**:
   - [ ] How to display when workflow config is missing/malformed?
   - [ ] How to handle entities with no related documents?
   - [ ] How to format transitions for statuses not in status_flow?

3. **Integration Testing Scenarios**:
   - [ ] Epic in planning mode with related docs
   - [ ] Feature in planning mode with no related docs
   - [ ] JSON output validation
   - [ ] Regression: Aggregation mode unchanged

4. **Documentation Review**:
   - [ ] CLI_REFERENCE.md updates needed?
   - [ ] Example JSON responses documented?
   - [ ] TODO comments for E15-F07 clearly marked?

### For Tech Lead Refinement (After BA)

1. **Technical Architecture Review**:
   - [ ] Validate helper function signatures
   - [ ] Review EntityDisplayOptions struct design
   - [ ] Confirm pterm usage patterns

2. **Test Coverage Requirements**:
   - [ ] Unit tests for each helper function (8 functions)
   - [ ] Integration tests for planning mode rendering
   - [ ] Regression tests for aggregation mode

3. **Code Review Focus**:
   - [ ] Helper function reusability
   - [ ] Error handling in render functions
   - [ ] TODO comment placement for E15-F07

---

## Appendix: Command Execution Log

```bash
# Triage assessment completed
$ ./bin/shark feature get E07-F31 --json
# Status: ready_for_triage

# Routing to BA refinement (STANDARD tier)
$ ./bin/shark feature update E07-F31 --status=ready_for_refinement_ba
SUCCESS: Feature E07-F31 updated successfully

# Verification
$ ./bin/shark feature get E07-F31 --json | jq -r '.feature.status'
ready_for_refinement_ba
```

---

## Conclusion

Feature E07-F31 is **STANDARD complexity** (4/18 points) and has been routed to `ready_for_refinement_ba` for Business Analyst review. The feature creates foundational infrastructure for unified rendering without requiring data model changes, API modifications, or complex UI work. It makes surgical fixes to existing render functions while establishing patterns for E15-F07 to consume later.

**Key Success Factors**:
- Well-scoped: Creates infrastructure, makes minimal changes
- Low risk: No refactoring, only additive changes
- Foundation work: Establishes patterns for future consolidation
- Backward compatible: Preserves all existing functionality

**Next Phase**: Business Analyst to validate acceptance criteria, clarify edge cases, and document integration test scenarios.
