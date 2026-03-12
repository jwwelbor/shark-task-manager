# E07-F32: Unified Entity Display Refactor Phase 2 - Test Plan

**Feature**: E07-F32 Unified Entity Display Refactor - Phase 2
**Complexity**: STANDARD (9/27)
**Date**: 2026-03-11
**Status**: Approved

---

## 1. AC Test Matrix

### AC-001: Related docs displayed in planning mode

**Requirement**: REQ-F-001 - `populateEpicPlanningInfo` and `populateFeaturePlanningInfo` must fetch and populate `RelatedDocs`.

| TC ID | Description | Input | Expected Output | Edge Case |
|-------|------------|-------|-----------------|-----------|
| TC-001-01 | Epic with related docs in planning mode | Epic in `draft` status with 2 related docs linked | `shark get <epic> --json` returns `related_documents` array with 2 entries containing path and type | - |
| TC-001-02 | Feature with related docs in planning mode | Feature in `ready_for_refinement_ba` status with 1 related doc | `shark get <feature> --json` returns `related_documents` array with 1 entry | - |
| TC-001-03 | Epic with no related docs in planning mode | Epic in planning status, no docs linked | `related_documents` is null or empty array in JSON, no crash | Empty data |
| TC-001-04 | Epic related docs match aggregation mode format | Same epic tested in both planning and aggregation modes | `related_documents` field structure is identical between modes | Mode parity |
| TC-001-05 | Feature related docs match aggregation mode format | Same feature tested in both modes | `related_documents` field structure is identical between modes | Mode parity |
| TC-001-06 | DocumentRepo unavailable / returns error | DocumentRepo fetch fails | Epic/feature still renders; `related_documents` is empty; no crash | Error graceful degradation |

**Service layer unit tests** (mocked repos):
- `TestPopulateEpicPlanningInfo_FetchesRelatedDocs` - Verify `DocumentRepo.ListForEpic` is called and result assigned to `info.RelatedDocs`
- `TestPopulateFeaturePlanningInfo_FetchesRelatedDocs` - Verify `DocumentRepo.ListForFeature` is called and result assigned to `info.RelatedDocs`
- `TestPopulateEpicPlanningInfo_DocRepoError_GracefulDegradation` - Verify error does not propagate up

---

### AC-002: Notes and context displayed in planning mode

**Requirement**: REQ-F-002 - Planning mode populate functions must fetch notes and context data.

| TC ID | Description | Input | Expected Output | Edge Case |
|-------|------------|-------|-----------------|-----------|
| TC-002-01 | Epic with notes in planning mode | Epic with 2 notes added via `shark epic note add` | `shark get <epic>` terminal output shows Notes section with type, timestamp, truncated content | - |
| TC-002-02 | Epic with context in planning mode | Epic with context set via `shark epic context set` | `shark get <epic>` terminal output shows Context section with key-value pairs | - |
| TC-002-03 | Feature with notes in planning mode | Feature with notes added | `shark get <feature>` shows Notes section | - |
| TC-002-04 | Feature with context in planning mode | Feature with context set | `shark get <feature>` shows Context section | - |
| TC-002-05 | Epic with no notes or context in planning mode | Epic in planning status, no notes/context | No Notes or Context sections rendered; no crash | Empty data |
| TC-002-06 | Feature with no notes or context in planning mode | Feature in planning status, no notes/context | No Notes or Context sections rendered; no crash | Empty data |
| TC-002-07 | Notes/context JSON output in planning mode | Epic with notes and context, `--json` flag | JSON includes `notes` array and `context_data` object | JSON parity |
| TC-002-08 | Notes/context match aggregation mode format | Same entity in both modes | `notes` and `context_data` field structure is identical | Mode parity |

**Service layer unit tests** (mocked repos):
- `TestPopulateEpicPlanningInfo_FetchesNotes` - Verify NoteRepo called and Notes field populated
- `TestPopulateEpicPlanningInfo_FetchesContextData` - Verify ContextRepo called and ContextData populated
- `TestPopulateFeaturePlanningInfo_FetchesNotes` - Same for features
- `TestPopulateFeaturePlanningInfo_FetchesContextData` - Same for features

**Struct extension tests**:
- `TestEpicDisplayInfo_HasNotesField` - Verify `Notes` field exists on `EpicDisplayInfo`
- `TestEpicDisplayInfo_HasContextDataField` - Verify `ContextData` field exists on `EpicDisplayInfo`
- `TestFeatureDisplayInfo_HasNotesField` - Same for features
- `TestFeatureDisplayInfo_HasContextDataField` - Same for features

---

### AC-003: Multi-byte characters not corrupted in note truncation

**Requirement**: REQ-F-003 - Replace byte-based `content[:77]` with rune-safe truncation.

| TC ID | Description | Input | Expected Output | Edge Case |
|-------|------------|-------|-----------------|-----------|
| TC-003-01 | ASCII text longer than 80 chars | `"ABCDEFG..."` (100 ASCII chars) | Truncated to 77 chars + "..." suffix, no corruption | Baseline |
| TC-003-02 | CJK characters longer than 80 runes | String of 100 CJK characters (each 3 bytes UTF-8) | Truncated to 77 runes + "...", all characters intact | Multi-byte (3-byte) |
| TC-003-03 | Emoji characters longer than 80 runes | String with 100 emoji (each 4 bytes UTF-8) | Truncated to 77 runes + "...", all emoji intact | Multi-byte (4-byte) |
| TC-003-04 | Mixed ASCII and multi-byte | "Hello " + 90 CJK chars | Truncated at rune boundary, no partial runes | Mixed encoding |
| TC-003-05 | Exactly 77 runes | String of exactly 77 runes | No truncation, no "..." suffix | Boundary |
| TC-003-06 | Fewer than 77 runes | String of 50 runes | No truncation, no "..." suffix | Under limit |
| TC-003-07 | Empty string | `""` | Empty string, no panic | Edge: empty |
| TC-003-08 | Single character | `"A"` | `"A"`, no truncation | Edge: minimal |
| TC-003-09 | Exactly 78 runes | String of 78 runes | Truncated to 77 runes + "..." | Boundary: just over |

**Unit tests** (pure functions, no DB):
- `TestTruncateRunes_ASCII` - TC-003-01
- `TestTruncateRunes_CJK` - TC-003-02
- `TestTruncateRunes_Emoji` - TC-003-03
- `TestTruncateRunes_Mixed` - TC-003-04
- `TestTruncateRunes_ExactBoundary` - TC-003-05
- `TestTruncateRunes_UnderLimit` - TC-003-06
- `TestTruncateRunes_Empty` - TC-003-07
- `TestTruncateRunes_SingleChar` - TC-003-08
- `TestTruncateRunes_JustOverBoundary` - TC-003-09

These should be table-driven tests in `render_common_test.go` testing the extracted `truncateRunes()` helper.

---

### AC-004: Epic get uses RenderEntity pattern

**Requirement**: REQ-F-004 - Replace monolithic `renderEpicDetails` with `RenderEntity()` + callbacks.

| TC ID | Description | Input | Expected Output | Edge Case |
|-------|------------|-------|-----------------|-----------|
| TC-004-01 | Epic get in aggregation mode | Epic with features and tasks, aggregation status | Output contains sections in standard order: Header, Basic Info, Valid Transitions, Orchestrator Action, Related Docs, Epic-Specific (features table, rollup), Notes, Context | - |
| TC-004-02 | Epic get in planning mode | Epic in `draft` or `ready_for_refinement_ba` status | Output contains standard sections with planning-mode content (workflow position) | - |
| TC-004-03 | Valid transitions displayed | Epic with valid next statuses | "Valid Transitions" section appears in output listing available statuses | New section |
| TC-004-04 | Orchestrator action displayed | Epic in a status with configured orchestrator action | "Orchestrator Action" section appears | New section |
| TC-004-05 | `renderEpicDetails` removed | Codebase search | No standalone 12-parameter `renderEpicDetails` function exists | Architecture |
| TC-004-06 | JSON output unchanged | `shark get <epic> --json` | JSON structure matches pre-refactor output (additive changes only) | Backward compat |
| TC-004-07 | All existing epic tests pass | `make test` | Zero test failures in epic-related test files | Regression |

**Verification approach**: Primarily manual visual comparison + existing test suite. No new unit tests for rendering output (pterm output is difficult to capture). Verify TC-004-05 via codebase grep.

---

### AC-005: Feature get uses RenderEntity pattern

**Requirement**: REQ-F-005 - Replace monolithic `renderFeatureDetails` with `RenderEntity()` + callbacks.

| TC ID | Description | Input | Expected Output | Edge Case |
|-------|------------|-------|-----------------|-----------|
| TC-005-01 | Feature get in aggregation mode | Feature with tasks, aggregation status | Output in standard order: Header, Basic Info, Valid Transitions, Orchestrator Action, Related Docs, Feature-Specific (task list, progress, work summary), Notes, Context | - |
| TC-005-02 | Feature get in planning mode | Feature in planning status | Standard sections with planning-mode content | - |
| TC-005-03 | Valid transitions displayed | Feature with valid next statuses | "Valid Transitions" section appears | New section |
| TC-005-04 | Orchestrator action displayed | Feature in status with configured action | "Orchestrator Action" section appears | New section |
| TC-005-05 | `renderFeatureDetails` removed | Codebase search | No standalone 12-parameter `renderFeatureDetails` function exists | Architecture |
| TC-005-06 | JSON output unchanged | `shark get <feature> --json` | JSON structure matches pre-refactor (additive only) | Backward compat |
| TC-005-07 | All existing feature tests pass | `make test` | Zero test failures in feature-related test files | Regression |

**Verification approach**: Same as AC-004 -- manual visual comparison + existing test suite.

---

### AC-006: JSON output includes consistent fields

**Requirement**: REQ-F-008 - All entity JSON builders include consistent common fields.

| TC ID | Description | Input | Expected Output | Edge Case |
|-------|------------|-------|-----------------|-----------|
| TC-006-01 | Epic JSON includes all common fields | `shark get <epic> --json` | JSON contains: `key`, `title`, `status`, `valid_transitions`, `orchestrator_action`, `related_documents`, `notes`, `context_data` | - |
| TC-006-02 | Feature JSON includes all common fields | `shark get <feature> --json` | JSON contains same common fields as above | - |
| TC-006-03 | Task JSON still includes all fields | `shark get <task> --json` | JSON contains all existing fields (no regression) | Regression |
| TC-006-04 | Field names consistent across types | Compare JSON keys for epic, feature, task | Common field names are identical (e.g., `valid_transitions` not `validTransitions` or `next_statuses`) | Naming consistency |
| TC-006-05 | `--field valid_transitions` works for epics | `shark get <epic> --field valid_transitions` | Returns array of valid transition strings | Field extraction |
| TC-006-06 | `--field valid_transitions` works for features | `shark get <feature> --field valid_transitions` | Returns array of valid transition strings | Field extraction |
| TC-006-07 | Empty collections use empty array not null | Entity with no notes/docs | `notes: []`, `related_documents: []` (not `null`) | Null safety |
| TC-006-08 | Backward compatibility | Existing JSON consumers | No fields removed or renamed from current output | Breaking change check |

**Verification approach**: JSON output comparison tests. Use `shark get <key> --json | jq '.field'` to extract and validate.

---

## 2. Component Test Strategy

### 2.1 DisplayService Planning Mode Population (Tasks 001-002)

**Component**: `internal/services/display_service.go` - `populateEpicPlanningInfo()` and `populateFeaturePlanningInfo()`

**Test type**: Service unit tests with mocked repositories

**Key behaviors to test**:
1. `DocumentRepo.ListForEpic/ListForFeature` is called in planning mode (currently only called in aggregation mode)
2. `NoteRepo` is called and result populates new `Notes` field on display info struct
3. `ContextService/ContextRepo` is called and result populates new `ContextData` field
4. Error from any repo fetch does not fail the entire populate function (graceful degradation)
5. Existing fields (`Phase`, `PhaseDescription`, `WorkflowPosition`) remain populated correctly

**Mock setup**:
- `MockDocumentRepo` with `ListForEpicFunc` / `ListForFeatureFunc`
- `MockNoteRepo` returning test notes
- `MockContextService` returning test context data

**Test file**: `internal/services/display_service_test.go` (extend existing or create)

### 2.2 Rune-Safe Truncation Helper (Task 003)

**Component**: `internal/cli/commands/render_common.go` - `truncateRunes()`

**Test type**: Pure unit tests (no dependencies)

**Key behaviors to test**:
1. Correct rune-based truncation (not byte-based)
2. Appends "..." suffix only when truncation occurs
3. Handles empty string without panic
4. Handles strings shorter than max length (no truncation)
5. Handles exact boundary (77 runes = no truncation, 78 runes = truncation)
6. Multi-byte characters preserved at truncation boundary

**Test file**: `internal/cli/commands/render_common_test.go` (extend existing)

### 2.3 RenderEntity Callbacks (Tasks 004-005)

**Component**: `internal/cli/commands/epic_helpers.go` and `feature_helpers.go` - `RenderSpecific` callback functions

**Test type**: Primarily regression testing via existing test suite + manual verification

**Key behaviors to verify**:
1. `RenderEntity()` is called (not legacy manual rendering)
2. `RenderSpecific` callback renders entity-specific sections (feature table, task table, rollups)
3. Section ordering matches `EntityDisplayOptions` standard order
4. Both planning and aggregation modes work through the same code path
5. No new panics introduced (nil checks on optional fields)

**Verification**: Run `make test` to ensure all existing `epic_get_integration_test.go` and `feature_get_integration_test.go` tests pass. Manual comparison of terminal output before/after.

### 2.4 Basic Info Helpers (Tasks 006-007)

**Component**: `buildEpicBasicInfo()` and `buildFeatureBasicInfo()` in helper files

**Test type**: Unit tests (pure function, no DB)

**Key behaviors to test**:
1. Returns `[][]string` with correct key-value pairs
2. Handles nil/empty optional fields without panic
3. Status field is included and correctly formatted
4. Path field is included when available
5. Output matches `buildTaskBasicInfo()` pattern (consistent format)

**Test file**: Can be added to existing test files for epic_helpers and feature_helpers.

---

## 3. Integration Scenarios

### 3.1 End-to-End: Planning Mode Display Completeness

**Scenario**: Verify that an epic/feature in planning mode shows all data sections that aggregation mode shows.

**Steps**:
1. Create an epic in `draft` status (planning mode)
2. Add 2 related docs via `shark related-docs add`
3. Add 2 notes via `shark epic note add`
4. Set context via `shark epic context set`
5. Run `shark get <epic>` and verify Related Docs, Notes, and Context sections appear
6. Run `shark get <epic> --json` and verify all fields populated
7. Repeat steps 1-6 for a feature

**Pass criteria**: All sections visible in both terminal and JSON output modes.

### 3.2 End-to-End: RenderEntity Section Ordering

**Scenario**: Verify section ordering matches the standard defined in `RenderEntity()`.

**Steps**:
1. Use an epic/feature with all possible data populated (related docs, notes, context, features/tasks)
2. Run `shark get <key>` and capture terminal output
3. Verify sections appear in order: Header > Basic Info > Valid Transitions > Orchestrator Action > Related Documents > Entity-Specific > Notes > Context Data

**Pass criteria**: Section ordering matches the standard for all three entity types (epic, feature, task).

### 3.3 End-to-End: JSON Field Consistency

**Scenario**: Verify `--json` output has consistent field names across entity types.

**Steps**:
1. Run `shark get <epic> --json`, `shark get <feature> --json`, `shark get <task> --json`
2. Extract top-level keys from each
3. Verify common keys are identical: `valid_transitions`, `orchestrator_action`, `related_documents`, `notes`, `context_data`
4. Verify `--field` extraction works for each common field on each entity type

**Pass criteria**: Common field names match exactly; `--field` extraction returns correct values.

### 3.4 Regression: Task Get Unchanged

**Scenario**: Verify `shark task get` is completely unaffected by the refactor.

**Steps**:
1. Run `shark get <task>` and `shark get <task> --json` before any F32 changes (baseline)
2. After all F32 changes, run same commands
3. Compare output

**Pass criteria**: Task get output is identical (zero changes to task rendering).

### 3.5 Cross-Component: Display Service to CLI Rendering

**Scenario**: Verify data flows correctly from DisplayService through to CLI rendering.

**Steps**:
1. Create epic with notes, context, and related docs
2. Set epic to planning status
3. Call `DisplayService.GetEpicDisplayInfo()` - verify Notes, ContextData, RelatedDocs fields populated
4. Run `shark get <epic>` - verify all three sections render in terminal output
5. Run `shark get <epic> --json` - verify all three fields in JSON

**Pass criteria**: Data populated in service layer correctly appears in both terminal and JSON output.

---

## 4. Quality Gates

| Gate | Criteria | Phase |
|------|----------|-------|
| Unit tests pass | `make test` returns 0 exit code | After each task |
| Lint passes | `make lint` returns 0 exit code | After each task |
| Format clean | `make fmt` produces no changes | After each task |
| No new panics | All nil-safety edge cases covered in tests | After Tasks 003, 004, 005 |
| JSON backward compatible | No fields removed or renamed | After Task 008 |
| UTF-8 correctness | `truncateRunes` tests cover all multi-byte scenarios | After Task 003 |
| Visual parity | Terminal output visually matches pre-refactor (except new sections) | After Tasks 004, 005 |

---

## 5. Test Execution Order

Tests should be executed in phase order matching the implementation sequence:

1. **Phase 1 tests** (Tasks 001-003): Service unit tests for planning mode data population + `truncateRunes` unit tests
2. **Phase 2 tests** (Tasks 004-005): Run full `make test` to verify regression; manual visual comparison
3. **Phase 3 tests** (Tasks 006-007): Unit tests for `buildEpicBasicInfo` / `buildFeatureBasicInfo`
4. **Phase 4 tests** (Task 008): JSON consistency tests across all entity types
5. **Final regression**: Full `make test` + integration scenarios 3.1-3.5

---

*Last Updated*: 2026-03-11
