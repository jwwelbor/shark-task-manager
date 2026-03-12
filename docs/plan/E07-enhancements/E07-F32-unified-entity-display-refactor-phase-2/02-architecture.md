# E07-F32: Unified Entity Display Refactor Phase 2 - Architecture

**Date:** 2026-03-11
**Status:** Approved
**Complexity Tier:** STANDARD (9/27 score)
**Dependency:** E07-F31 (Phase 1 - RenderEntity() API must be stable)

---

## 1. Current State Analysis

### 1.1 Architecture Overview

The entity display system has three layers:

1. **DisplayService** (`internal/services/display_service.go`) - Assembles all display data into `EpicDisplayInfo` / `FeatureDisplayInfo` structs via planning-mode and aggregation-mode populate functions.
2. **Helper Functions** (`internal/cli/commands/epic_helpers.go`, `feature_helpers.go`) - Render the display info structs into terminal output. Currently use manual rendering, not the unified `RenderEntity()` pattern.
3. **RenderEntity()** (`internal/cli/commands/render_common.go`) - The unified display function established in E07-F31. Currently only used by `task.go`.

### 1.2 Identified Issues

**Issue Category 1: Planning Mode Data Gaps (Tasks 001-002)**

The `populateEpicPlanningInfo()` (line 347) and `populateFeaturePlanningInfo()` (line 520) in `display_service.go` do NOT fetch:
- Related documents (via `DocumentRepo.ListForEpic/ListForFeature`)
- Notes (not fetched at all - no `Notes` field on `EpicDisplayInfo` or `FeatureDisplayInfo`)
- Context data (not fetched at all - no `ContextData` field on structs)

In contrast, the aggregation mode functions DO fetch related documents (lines 507 and 625 respectively). The structs themselves have a `RelatedDocs` field but lack `Notes` and `ContextData` fields entirely.

**Issue Category 2: UTF-8 Truncation Bug (Task 003)**

Two locations use byte-based string slicing for note content truncation:
- `epic_helpers.go` line 325: `content = content[:77] + "..."`
- `feature_helpers.go` line 528: `content = content[:77] + "..."`

This can corrupt multi-byte UTF-8 characters (e.g., CJK, emoji, accented characters) by slicing in the middle of a rune. The correct pattern already exists in `render_common.go` `renderNotes()` which uses `[]rune` conversion.

**Issue Category 3: Inconsistent Display Architecture (Tasks 004-008)**

The following functions bypass `RenderEntity()` and manually render all display sections:

| Function | File | Lines | Parameters |
|----------|------|-------|------------|
| `renderEpicPlanning()` | epic_helpers.go | 126-211 | 1 (struct) |
| `renderEpicDetails()` | epic_helpers.go | 214-400+ | 12 (parameter explosion) |
| `renderFeaturePlanning()` | feature_helpers.go | 111-182 | 1 (struct) |
| `renderFeatureDetails()` | feature_helpers.go | 356-560+ | 12 (parameter explosion) |

The reference implementation in `task.go` `runTaskGet()` (line 150) shows the target pattern: build a `[][]string` via `buildTaskBasicInfo()`, then call `RenderEntity(EntityDisplayOptions{...})`.

---

## 2. Target State

### 2.1 Data Completeness

Both planning and aggregation modes will return identical auxiliary data (related docs, notes, context). The display info structs will be extended:

```go
// Added to EpicDisplayInfo
Notes       []*models.EntityNote  `json:"notes,omitempty"`
ContextData *models.ContextData   `json:"context_data,omitempty"`

// Added to FeatureDisplayInfo
Notes       []*models.EntityNote  `json:"notes,omitempty"`
ContextData *models.ContextData   `json:"context_data,omitempty"`
```

### 2.2 Unified Rendering

All four manual render functions will be replaced by calls to `RenderEntity()` with entity-specific `RenderSpecific` callbacks:

```
renderEpicPlanning()    --> RenderEntity() + epicPlanningCallback
renderEpicDetails()     --> RenderEntity() + epicAggregationCallback
renderFeaturePlanning() --> RenderEntity() + featurePlanningCallback
renderFeatureDetails()  --> RenderEntity() + featureAggregationCallback
```

Each callback renders only the entity-specific sections (features table, task table, rollup stats, workflow position). The common sections (header, basic info, transitions, orchestrator action, related docs, notes, context) are handled by `RenderEntity()`.

### 2.3 Consistent UTF-8 Handling

All string truncation will use rune-safe operations via a shared helper.

---

## 3. Key Design Decisions

### ADR-001: Extend Display Structs vs. Fetch in CLI Layer

**Decision:** Extend `EpicDisplayInfo` and `FeatureDisplayInfo` with `Notes` and `ContextData` fields, and populate them in the service layer.

**Rationale:** The DisplayService already fetches related docs in aggregation mode. Adding notes and context follows the same pattern and keeps data assembly in the service layer where it belongs. CLI commands should not need to make additional service calls after getting display info.

**Alternative Rejected:** Having CLI commands fetch notes/context separately. This would scatter data assembly logic across the CLI layer and make JSON output harder to consolidate.

### ADR-002: Callback Pattern for Entity-Specific Rendering

**Decision:** Use the existing `RenderSpecific func()` callback field in `EntityDisplayOptions` for entity-specific sections (features table, task table, rollups, workflow position).

**Rationale:** This is the proven pattern from `task.go`. The callback captures display info in a closure and renders entity-specific content at the correct position in the section ordering. No new types or interfaces needed.

**Example:**
```go
RenderEntity(EntityDisplayOptions{
    // ... common fields ...
    RenderSpecific: func() {
        renderWorkflowPosition(info.WorkflowPosition)
        renderFeaturesTable(info.Features)
    },
})
```

### ADR-003: Extract Shared Truncation Helper

**Decision:** Create a `truncateRunes(s string, maxLen int) string` helper in `render_common.go` and use it from all truncation sites.

**Rationale:** The correct pattern (`[]rune` conversion) already exists in `renderNotes()`. Extracting it prevents future regressions and makes the fix trivially verifiable. The function is small (5-6 lines) and belongs in `render_common.go` alongside other display helpers.

### ADR-004: buildEpicBasicInfo / buildFeatureBasicInfo Pattern

**Decision:** Create `buildEpicBasicInfo()` and `buildFeatureBasicInfo()` functions that return `[][]string`, mirroring the existing `buildTaskBasicInfo()` pattern.

**Rationale:** This is the established pattern for the `BasicInfo` field in `EntityDisplayOptions`. Each function extracts entity-specific key-value pairs (title, status, phase, description, priority, path, etc.) from the display info struct.

### ADR-005: Preserve Existing JSON Builders

**Decision:** Leave `buildEpicGetJSON()` and `buildFeatureGetJSON()` functions intact. JSON output path is a separate concern from terminal rendering and will be consolidated in Task 008 after all other changes are stable.

**Rationale:** Changing JSON structure and terminal rendering simultaneously increases risk. The JSON builders already work correctly. Task 008 can consolidate them to use the extended display info structs without rushing.

---

## 4. File Change Map

### Phase 1: Bug Fixes (Tasks 001-003, parallel)

| File | Change | Task |
|------|--------|------|
| `internal/services/display_service.go` | Add `Notes` and `ContextData` fields to `EpicDisplayInfo` struct (line 40) | 001 |
| `internal/services/display_service.go` | Populate `RelatedDocs`, `Notes`, `ContextData` in `populateEpicPlanningInfo()` (line 347) | 001 |
| `internal/services/display_service.go` | Populate `Notes`, `ContextData` in `populateEpicAggregationInfo()` (line 419) | 001 |
| `internal/services/display_service.go` | Add `Notes` and `ContextData` fields to `FeatureDisplayInfo` struct (line 67) | 002 |
| `internal/services/display_service.go` | Populate `RelatedDocs`, `Notes`, `ContextData` in `populateFeaturePlanningInfo()` (line 520) | 002 |
| `internal/services/display_service.go` | Populate `Notes`, `ContextData` in `populateFeatureAggregationInfo()` (line 598) | 002 |
| `internal/cli/commands/render_common.go` | Extract `truncateRunes()` helper function | 003 |
| `internal/cli/commands/render_common.go` | Update `renderNotes()` to use `truncateRunes()` | 003 |
| `internal/cli/commands/epic_helpers.go` | Replace byte-based truncation at line 325 with `truncateRunes()` | 003 |
| `internal/cli/commands/feature_helpers.go` | Replace byte-based truncation at line 528 with `truncateRunes()` | 003 |

### Phase 2: Render Refactors (Tasks 004-005, parallel)

| File | Change | Task |
|------|--------|------|
| `internal/cli/commands/epic_helpers.go` | Create `buildEpicBasicInfo()` returning `[][]string` | 004 |
| `internal/cli/commands/epic_helpers.go` | Refactor `renderEpicPlanning()` to use `RenderEntity()` with callback | 004 |
| `internal/cli/commands/epic_helpers.go` | Refactor `renderEpicDetails()` to use `RenderEntity()` with callback | 004 |
| `internal/cli/commands/feature_helpers.go` | Create `buildFeatureBasicInfo()` returning `[][]string` | 005 |
| `internal/cli/commands/feature_helpers.go` | Refactor `renderFeaturePlanning()` to use `RenderEntity()` with callback | 005 |
| `internal/cli/commands/feature_helpers.go` | Refactor `renderFeatureDetails()` to use `RenderEntity()` with callback | 005 |

### Phase 3: Helper Extraction (Tasks 006-007, parallel)

| File | Change | Task |
|------|--------|------|
| `internal/cli/commands/epic_helpers.go` | Extract shared epic rendering helpers (workflow position, features table) | 006 |
| `internal/cli/commands/feature_helpers.go` | Extract shared feature rendering helpers (task table, status breakdown) | 007 |

### Phase 4: JSON Consolidation (Task 008)

| File | Change | Task |
|------|--------|------|
| `internal/cli/commands/epic_helpers.go` | Update `buildEpicGetJSON()` to use extended `EpicDisplayInfo` fields | 008 |
| `internal/cli/commands/feature_helpers.go` | Update `buildFeatureGetJSON()` to use extended `FeatureDisplayInfo` fields | 008 |

---

## 5. Implementation Sequence

```
Phase 1 (parallel):
  Task 001: Fix epic planning mode data gaps
  Task 002: Fix feature planning mode data gaps
  Task 003: Fix UTF-8 truncation bug
    |
Phase 2 (parallel, depends on Phase 1):
  Task 004: Refactor epic render functions to RenderEntity()
  Task 005: Refactor feature render functions to RenderEntity()
    |
Phase 3 (parallel, depends on Phase 2):
  Task 006: Extract shared epic rendering helpers
  Task 007: Extract shared feature rendering helpers
    |
Phase 4 (depends on Phase 3):
  Task 008: Consolidate JSON output
```

**Critical path:** 001/002/003 -> 004/005 -> 006/007 -> 008

**Estimated total tasks:** 8 tasks across 4 phases.

---

## 6. Risk Mitigations

### Risk 1: Visual Regression in Terminal Output

**Likelihood:** Medium
**Impact:** Medium (broken display for users)

**Mitigation:**
- Capture current terminal output for all entity types (epic planning, epic aggregation, feature planning, feature aggregation) as baseline snapshots before refactoring.
- After each phase, compare output visually to baseline. The section ordering in `RenderEntity()` must match current manual ordering exactly.
- Run `shark epic get`, `shark feature get` for both planning and aggregation mode entities after each change.

### Risk 2: DisplayService Repository Dependencies

**Likelihood:** Low
**Impact:** Low (graceful degradation)

**Mitigation:**
- The `NoteRepo` and `ContextService` may not be wired into `DisplayService` yet. Follow the existing pattern in `display_service.go` where `DocumentRepo` is used: fetch data, ignore errors gracefully, default to nil/empty.
- Check `DisplayServiceDeps` struct for available repositories before assuming they exist.

### Risk 3: JSON Output Breaking Changes

**Likelihood:** Low
**Impact:** High (breaks agent consumers)

**Mitigation:**
- Task 008 (JSON consolidation) is deliberately last, after all other changes are stable.
- New fields (`notes`, `context_data`) in display structs are `omitempty` so they only appear when populated.
- Existing JSON structure must not change for fields that are already present.

### Risk 4: Phase Dependency Violations

**Likelihood:** Low
**Impact:** Medium (merge conflicts, test failures)

**Mitigation:**
- Phase 1 tasks (001-003) are independent and can be merged in any order.
- Phase 2 tasks (004-005) modify different files (epic_helpers.go vs feature_helpers.go) so can merge in parallel.
- Each task must pass `make fmt && make lint && make test` before merging.

---

## 7. Testing Strategy

### Unit Tests

- **Task 003:** Test `truncateRunes()` with ASCII, multi-byte UTF-8 (CJK), emoji, empty string, and strings shorter than max length.
- **Tasks 001-002:** Test that `populateEpicPlanningInfo()` and `populateFeaturePlanningInfo()` populate `RelatedDocs`, `Notes`, and `ContextData` fields. Use mocked repositories.
- **Tasks 004-005:** No new unit tests needed for rendering (visual output). Existing tests should continue to pass.

### Integration Tests

- After each phase, verify `shark epic get <key>` and `shark feature get <key>` produce correct output in both `--json` and terminal modes.
- Test with entities in both planning mode (workflow statuses) and aggregation mode (derived statuses) to cover both code paths.

### Regression Tests

- Verify that `shark task get <key>` still works correctly (no changes to task rendering).
- Verify JSON output structure is backward-compatible (no removed fields, only additions).
