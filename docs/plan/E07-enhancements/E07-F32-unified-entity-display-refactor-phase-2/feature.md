---
feature_key: E07-F32-unified-entity-display-refactor-phase-2
epic_key: E07
title: Unified Entity Display Refactor - Phase 2
description: Fix bugs in planning mode display and refactor epic/feature get commands to use the unified RenderEntity() pattern from render_common.go, matching the task get reference implementation.
---

# Unified Entity Display Refactor - Phase 2

**Feature Key**: E07-F32-unified-entity-display-refactor-phase-2
**Complexity**: STANDARD (9/27)
**Status**: ready_for_refinement_ba

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem

The epic and feature `get` commands use legacy monolithic render functions (`renderEpicDetails` with 12 parameters, `renderFeatureDetails` with 12 parameters) that do not leverage the unified `RenderEntity()` pattern established in `render_common.go`. This causes three categories of issues:

1. **Data Population Bugs (Planning Mode)**: `populateEpicPlanningInfo` and `populateFeaturePlanningInfo` in `display_service.go` do not fetch related documents, notes, or context data. The aggregation-mode counterparts (`populateEpicAggregationInfo`, `populateFeatureAggregationInfo`) correctly fetch these - the planning-mode functions were simply never updated to match.

2. **UTF-8 Corruption Bug**: `epic_helpers.go` (line 325) and `feature_helpers.go` (line 528) use byte-based truncation (`content[:77]`) which splits multi-byte UTF-8 characters, producing corrupted output. The fix already exists in `render_common.go:renderNotes()` which uses `[]rune` conversion.

3. **Inconsistent Display Architecture**: Epic and feature get commands bypass `RenderEntity()` entirely, manually rendering sections in ad-hoc order. This means they miss standardized section ordering, do not display valid transitions or orchestrator actions consistently, and require parallel maintenance of rendering logic across three entity types.

### Solution

Fix the identified bugs in planning mode data population, then refactor the epic and feature get commands to use `RenderEntity()` with entity-specific `RenderSpecific` callbacks, matching the task get reference implementation in `task.go`. Extract reusable helper functions (`buildEpicBasicInfo`, `buildFeatureBasicInfo`) following the `buildTaskBasicInfo` pattern, and consolidate JSON builders for consistent field output across all entity types.

### Impact

- Planning mode correctly shows related docs, notes, and context data for epics and features
- Safe Unicode handling in note truncation eliminates corrupted display output
- Consistent display output across all entity types (epic, feature, task)
- Reduced code duplication in CLI display layer (~200 lines eliminated)
- Easier maintenance via unified render pattern with entity-specific callbacks

---

## Requirements

### MoSCoW Classification

#### Must-Have (Critical path - blocks consistent entity display)

1. **REQ-F-001**: Add related docs fetch to planning mode populate functions
   - `populateEpicPlanningInfo` in `display_service.go` must call `DocumentRepo.ListByEntity()` for the epic and populate `RelatedDocs` field on `EpicDisplayInfo`, matching the pattern in `populateEpicAggregationInfo` (line 507)
   - `populateFeaturePlanningInfo` must do the same for features, matching `populateFeatureAggregationInfo` (line 625)
   - Both `EpicDisplayInfo` and `FeatureDisplayInfo` structs must include `RelatedDocs` field accessible in planning mode

2. **REQ-F-002**: Add Notes and ContextData fields to planning mode
   - `populateEpicPlanningInfo` must fetch notes via `NoteRepo` and context data via `ContextRepo` and populate corresponding fields on `EpicDisplayInfo`
   - `populateFeaturePlanningInfo` must do the same for features
   - `EpicDisplayInfo` and `FeatureDisplayInfo` structs must have `Notes` and `ContextData` fields (these fields may already exist for aggregation mode; if so, they simply need to be populated in planning mode too)

3. **REQ-F-003**: Fix byte-based note truncation to use rune-safe truncation
   - Replace `content[:77]` in `epic_helpers.go` (line 325) with `string([]rune(content)[:77])` following the pattern in `render_common.go:renderNotes()`
   - Replace `content[:77]` in `feature_helpers.go` (line 528) with the same rune-safe approach
   - Must handle edge case where rune length is less than 77

4. **REQ-F-004**: Refactor `renderEpicDetails` to use `RenderEntity()` with callbacks
   - Replace the monolithic 12-parameter `renderEpicDetails` function with a call to `RenderEntity()` using `EntityDisplayOptions`
   - Epic-specific rendering (feature list, progress, impediments) moves to a `RenderSpecific` callback
   - Must produce equivalent visual output for both planning and aggregation modes

5. **REQ-F-005**: Refactor `renderFeatureDetails` to use `RenderEntity()` with callbacks
   - Replace the monolithic 12-parameter `renderFeatureDetails` function with a call to `RenderEntity()` using `EntityDisplayOptions`
   - Feature-specific rendering (task list, progress breakdown, work summary, action items) moves to a `RenderSpecific` callback
   - Must produce equivalent visual output for both planning and aggregation modes

#### Should-Have (Code quality improvements, not blocking)

6. **REQ-F-006**: Extract `buildEpicBasicInfo` helper function
   - Extract epic basic info assembly into a standalone function following the `buildTaskBasicInfo` pattern in `task.go`
   - Returns `[]KeyValuePair` for use with `RenderEntity()` BasicInfo field
   - Depends on REQ-F-004 completion

7. **REQ-F-007**: Extract `buildFeatureBasicInfo` helper function
   - Extract feature basic info assembly into a standalone function following the `buildTaskBasicInfo` pattern in `task.go`
   - Returns `[]KeyValuePair` for use with `RenderEntity()` BasicInfo field
   - Depends on REQ-F-005 completion

8. **REQ-F-008**: Consolidate JSON builders for consistent field output
   - Ensure `buildEpicGetJSON` and `buildFeatureGetJSON` include consistent fields matching `buildTaskGetJSON`: valid_transitions, orchestrator_action, related_docs, notes, context_data
   - Remove duplicate field assembly logic where present
   - Depends on REQ-F-004 through REQ-F-007 completion

#### Could-Have

None identified. All valuable improvements are captured in Must-Have and Should-Have.

#### Won't-Have (This Release)

- HTTP API display changes (CLI only)
- New display sections or features beyond bug fixes and consistency
- Changes to task get command (already uses RenderEntity as reference)

---

## Acceptance Criteria

### AC-001: Related docs displayed in planning mode (REQ-F-001)

**Given** an epic with related documents linked via `shark related-docs add`
**When** the epic is in a planning-phase status and `shark get <epic-key>` is executed
**Then** the Related Documents section is displayed with document paths and types, identical to aggregation mode output

**Given** a feature with related documents linked via `shark related-docs add`
**When** the feature is in a planning-phase status and `shark get <feature-key>` is executed
**Then** the Related Documents section is displayed with document paths and types, identical to aggregation mode output

### AC-002: Notes and context displayed in planning mode (REQ-F-002)

**Given** an epic with notes added via `shark epic note add` and context set via `shark epic context set`
**When** the epic is in a planning-phase status and `shark get <epic-key>` is executed
**Then** the Notes section shows all notes with type, timestamp, and truncated content
**And** the Context section shows all context key-value pairs

**Given** a feature with notes and context data
**When** the feature is in a planning-phase status and `shark get <feature-key>` is executed
**Then** Notes and Context sections are displayed, identical in format to aggregation mode output

### AC-003: Multi-byte characters not corrupted in note truncation (REQ-F-003)

**Given** a note containing multi-byte UTF-8 characters (e.g., emoji, CJK characters) longer than 80 characters
**When** the note is displayed in epic or feature get output
**Then** the truncated text ends with "..." and does not contain corrupted byte sequences
**And** the truncation point respects character boundaries (no partial runes)

### AC-004: Epic get uses RenderEntity pattern (REQ-F-004)

**Given** an epic in any status (planning or aggregation mode)
**When** `shark get <epic-key>` is executed
**Then** the output contains standardized sections in this order: Header, Basic Info, Valid Transitions, Orchestrator Action, Related Docs, Epic-Specific Details, Notes, Context Data
**And** the `renderEpicDetails` function no longer exists as a standalone 12-parameter function
**And** all existing tests pass with the refactored code

### AC-005: Feature get uses RenderEntity pattern (REQ-F-005)

**Given** a feature in any status (planning or aggregation mode)
**When** `shark get <feature-key>` is executed
**Then** the output contains standardized sections in this order: Header, Basic Info, Valid Transitions, Orchestrator Action, Related Docs, Feature-Specific Details, Notes, Context Data
**And** the `renderFeatureDetails` function no longer exists as a standalone 12-parameter function
**And** all existing tests pass with the refactored code

### AC-006: JSON output includes consistent fields (REQ-F-008)

**Given** any entity type (epic, feature, or task)
**When** `shark get <key> --json` is executed
**Then** the JSON output includes these common fields: key, title, status, valid_transitions, orchestrator_action, related_docs, notes, context_data
**And** field names and nesting structure are consistent across all three entity types

---

## Task Breakdown

8 tasks organized in 4 phases with explicit dependencies:

| Task | Title | Priority | MoSCoW | Dependencies |
|------|-------|----------|--------|--------------|
| T-E07-F32-001 | Fix related docs fetch in planning mode | 9 | Must-Have | None |
| T-E07-F32-002 | Fix Notes/ContextData fields in planning mode | 9 | Must-Have | None |
| T-E07-F32-003 | Fix byte-based note truncation | 8 | Must-Have | None |
| T-E07-F32-004 | Refactor renderEpicDetails to use RenderEntity | 7 | Must-Have | 001, 002, 003 |
| T-E07-F32-005 | Refactor renderFeatureDetails to use RenderEntity | 7 | Must-Have | 001, 002, 003 |
| T-E07-F32-006 | Extract buildEpicBasicInfo helper | 6 | Should-Have | 004 |
| T-E07-F32-007 | Extract buildFeatureBasicInfo helper | 6 | Should-Have | 005 |
| T-E07-F32-008 | Consolidate JSON builders | 5 | Should-Have | 004, 005, 006, 007 |

**Phase 1** (001-003) can be executed in parallel - independent bug fixes.
**Phase 2** (004-005) can be executed in parallel after Phase 1 - both refactors are independent of each other.
**Phase 3** (006-007) can be executed in parallel after their respective Phase 2 task.
**Phase 4** (008) requires all prior phases complete.

---

## Key Files

| File | Role |
|------|------|
| `internal/services/display_service.go` | Planning mode populate functions to fix (001, 002) |
| `internal/cli/commands/render_common.go` | Unified `RenderEntity()` pattern - target architecture |
| `internal/cli/commands/epic_helpers.go` | Epic rendering, truncation bug, JSON builder (003, 004, 006, 008) |
| `internal/cli/commands/feature_helpers.go` | Feature rendering, truncation bug, JSON builder (003, 005, 007, 008) |
| `internal/cli/commands/epic.go` | Epic get command call site |
| `internal/cli/commands/feature.go` | Feature get command call site |
| `internal/cli/commands/task.go` | Reference implementation - `buildTaskBasicInfo`, `RenderEntity` usage |

---

## Dependencies & Integrations

### Internal Dependencies

| Dependency | Status | Impact |
|-----------|--------|--------|
| E07-F31 (Unified Entity Display Rendering) | Active (not yet completed) | Provides stable `RenderEntity()` API in `render_common.go`. Phase 2 tasks (004-005) should not begin until F31 is completed and the API is stable. Phase 1 bug fixes (001-003) can proceed independently. |

### External Dependencies

None. All changes are internal to the CLI display layer.

---

## Out of Scope

### Explicitly Excluded

1. **HTTP API display changes** - This feature covers CLI `get` command display only
2. **Task get refactoring** - Task get already uses `RenderEntity()` and serves as the reference implementation
3. **New display features** - No new display sections beyond what currently exists; this is strictly bug fixes and architectural consistency
4. **Display service structural refactoring** - Display service internals beyond fixing the planning mode data population gaps are out of scope
5. **Performance optimization** - No changes to query patterns or caching; the additional fetches in planning mode are consistent with existing aggregation mode patterns

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| E07-F31 API changes during development | Medium | High | Phase 1 (bug fixes) proceeds independently. Phase 2 tasks (refactors) wait for F31 completion. |
| Display output regression after refactor | Medium | Medium | Existing tests must pass. Manual comparison of output before/after refactor. |
| Struct field additions break serialization | Low | Low | Fields added as pointer types or with `omitempty` JSON tags to maintain backward compatibility. |

---

*Last Updated*: 2026-03-11
