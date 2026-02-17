# Unified Entity Display Rendering

**Feature Key**: E07-F31

---

## Epic

- **Epic**: [E07 - Enhancements](/docs/plan/E07-enhancements/epic.md)

---

## Related Documents

- **[02-architecture.md](02-architecture.md)**: Component design, helper function signatures, EntityDisplayOptions struct
- **[06-security-performance.md](06-security-performance.md)**: Performance targets, security analysis, test strategy
- **[COMPLEXITY-TRIAGE-REPORT.md](COMPLEXITY-TRIAGE-REPORT.md)**: Complexity assessment and routing decision

---

## Goal

### Problem

The Shark CLI currently has fragmented rendering logic for displaying epics, features, and tasks:

1. **Code Duplication**: 6 separate render functions exist across epic.go, feature.go, and task.go (~500+ lines of duplicated logic), making maintenance difficult and introducing inconsistencies.

2. **Planning Mode Bug**: AI agents and users cannot see related documents when entities are in planning mode (statuses like `ready_for_decomposition`, `ready_for_refinement_ba`, etc.), blocking workflow transparency. Related documents only display in aggregation mode, creating a 50% visibility gap.

3. **Missing Valid Transitions**: Users must manually inspect workflow configuration to understand which status transitions are allowed from the current state. The CLI shows workflow position (linear path through complex state machine) but not actionable next steps.

4. **Inconsistent Display**: Different entity types show different information sections in different orders, creating cognitive overhead for users switching between epics, features, and tasks.

This impacts:
- **AI Agents** (primary users): Cannot see PRDs, research reports, and design documents during planning phases, requiring manual file discovery
- **Developers**: Spend time maintaining duplicate rendering code across 3 command files
- **Users**: Experience inconsistent output structure, missing critical workflow information

### Solution

Create unified rendering infrastructure (`internal/cli/commands/render_common.go`) with reusable helper functions for common display sections, while making **surgical fixes** to existing render functions to add missing sections (related docs, valid transitions).

This is a **two-phase approach**:
- **Phase 1 (E07-F31)**: Create infrastructure, fix immediate bugs with minimal changes (<30 lines across epic.go + feature.go)
- **Phase 2 (E15-F07)**: Full refactoring when service layer migration is complete

The feature establishes foundation patterns without forcing large-scale changes to command files that E15 will refactor anyway.

### Impact

**Immediate (E07-F31)**:
- Related documents visible in planning mode: 100% visibility (up from 50%)
- Valid transitions displayed: AI agents and users can see actionable next steps without reading workflow config
- Shared rendering functions: Single source of truth for common sections (8 helper functions)
- Zero breaking changes: All existing functionality preserved

**Foundation for E15-F07**:
- Consolidated rendering: `RenderEntity()` function ready for use when commands are refactored
- Reduced duplication: Command files can call helpers instead of implementing display logic inline
- Maintainability: Changes to common sections affect all entities automatically

**Measurable Outcomes**:
- Planning mode displays related docs: 0% → 100% of entities
- Lines of rendering code: Baseline for E15-F07 reduction target
- Developer maintenance time: Baseline for E15-F07 improvement measurement

---

## User Personas

### Persona 1: AI Agent (Orchestrator)

**Profile**:
- **Role/Title**: Automated agent orchestrating multi-phase feature development workflows
- **Experience Level**: Programmatic CLI interaction, reads JSON output exclusively
- **Key Characteristics**:
  - Executes commands based on orchestrator_action instructions
  - Requires complete context discovery via shark CLI (cannot browse filesystem manually)
  - Operates in planning phases: research, refinement, test planning, task generation
  - Depends on related documents to understand requirements and constraints

**Goals Related to This Feature**:
1. Discover all related documents (PRDs, research reports, architecture docs) for assigned feature/epic
2. Understand valid status transitions to advance workflow correctly
3. Access orchestrator actions to know which agent to spawn next

**Pain Points This Feature Addresses**:
- **Current**: Cannot see related documents when feature is in `ready_for_refinement_ba` status, must manually search filesystem
- **Current**: Must read workflow config YAML to determine valid transitions from current status
- **Current**: Orchestrator action visible but related docs hidden, forcing incomplete context

**Success Looks Like**:
Agent executes `shark feature get E15-F01 --json`, receives complete context including related_documents array and valid_transitions array, proceeds with refinement work using discovered PRD and architecture documents without requiring manual file path discovery.

---

### Persona 2: Developer (Manual CLI User)

**Profile**:
- **Role/Title**: Backend/frontend developer working on shark task implementation
- **Experience Level**: Comfortable with CLI tools, uses both --json and human-readable output
- **Key Characteristics**:
  - Works on multiple entities (epics, features, tasks) throughout the day
  - Needs quick visibility into status, next steps, and related documentation
  - Switches between planning artifacts (PRDs) and implementation work frequently

**Goals Related to This Feature**:
1. Quickly understand what status transitions are valid from current state
2. Access related documents (specs, PRDs, design docs) without leaving CLI
3. Consistent experience across epic/feature/task display commands

**Pain Points This Feature Addresses**:
- **Current**: Must remember or look up workflow config to know valid transitions
- **Current**: Related docs only visible in aggregation mode, not during planning phases
- **Current**: Display output varies between epic.go, feature.go, task.go implementations

**Success Looks Like**:
Developer runs `shark feature get E07-F31`, sees related documents section showing PRD and triage report, sees valid transitions showing `in_refinement_ba` and `blocked` as next options, makes informed decision about workflow progression without consulting external documentation.

---

### Persona 3: QA Engineer

**Profile**:
- **Role/Title**: Quality assurance engineer testing feature implementations
- **Experience Level**: Uses shark CLI to track test planning and execution status
- **Key Characteristics**:
  - Creates feature-level test plans during `in_test_planning` phase
  - Links test plan documents to features as related docs
  - Needs to verify orchestrator actions point to correct next agent

**Goals Related to This Feature**:
1. Register test plan documents and see them displayed in feature output
2. Verify workflow transitions make sense for QA handoff
3. Consistent display of test planning artifacts across features

**Pain Points This Feature Addresses**:
- **Current**: Test plans registered as related docs but not visible during planning mode
- **Current**: Cannot verify valid transitions include QA-relevant statuses
- **Current**: Orchestrator actions inconsistently displayed

**Success Looks Like**:
QA engineer runs `shark feature get E07-F31` after registering test plan, sees test plan in related documents section, verifies valid transitions include `ready_for_task_generation`, confirms orchestrator action correctly points to next workflow phase.

---

## User Stories

### Must-Have Stories

**Story 1**: As an **AI agent**, I want to see **related documents in planning mode** so that I can **access PRDs, research reports, and architecture docs without manual file discovery**.

**Acceptance Criteria**:
- [ ] When epic status is in planning mode (ready_for_decomposition, in_decomposition, etc.), `shark epic get E## --json` includes `related_documents` array
- [ ] When feature status is in planning mode (ready_for_refinement_ba, in_refinement_ba, etc.), `shark feature get E##-F## --json` includes `related_documents` array
- [ ] Related documents array structure matches existing aggregation mode format (backward compatible)
- [ ] Human-readable output displays "Related Documents" section with title, type, and file path
- [ ] Related documents displayed regardless of `is_planning` flag value

**Story 2**: As a **developer**, I want to see **valid status transitions** so that I can **understand which workflow actions are allowed from the current state**.

**Acceptance Criteria**:
- [ ] `shark epic get E## --json` includes `valid_transitions` array with allowed next statuses
- [ ] `shark feature get E##-F## --json` includes `valid_transitions` array with allowed next statuses
- [ ] `shark task get T-E##-F##-001 --json` includes `valid_transitions` array with allowed next statuses
- [ ] Valid transitions extracted from `.sharkconfig.json` status_flow configuration
- [ ] Human-readable output displays "Valid Transitions" section listing allowed next statuses
- [ ] If status not in status_flow (terminal state), valid_transitions is empty array

**Story 3**: As a **QA engineer**, I want **consistent display of orchestrator actions across planning and aggregation modes** so that I can **verify workflow routing without mode-dependent visibility**.

**Acceptance Criteria**:
- [ ] Orchestrator action displayed in both planning and aggregation modes
- [ ] Orchestrator action section appears in consistent position (after Valid Transitions, before Related Documents)
- [ ] JSON output includes `orchestrator_action` object in both modes
- [ ] Orchestrator action displays agent_type, skills, and instruction text
- [ ] Missing orchestrator action shows clear "None configured" message

---

### Should-Have Stories

**Story 4**: As a **developer**, I want **helper functions for common display sections** so that **future rendering code can avoid duplication**.

**Acceptance Criteria**:
- [ ] `render_common.go` created in `internal/cli/commands/`
- [ ] Helper functions implemented: renderHeader(), renderBasicInfo(), renderValidTransitions(), renderOrchestratorAction(), renderRelatedDocuments(), renderNotes(), renderContextData()
- [ ] Each helper function has unit test coverage
- [ ] GetValidTransitions() helper extracts transitions from workflow config
- [ ] Helpers accept standardized data structures (no entity-specific types)

**Story 5**: As an **architect**, I want **EntityDisplayOptions struct and RenderEntity() function** so that **E15-F07 can consolidate rendering without reimplementing infrastructure**.

**Acceptance Criteria**:
- [ ] EntityDisplayOptions struct defined with all common display fields
- [ ] RenderEntity() function implements unified rendering flow
- [ ] RenderSpecific callback allows entity-specific sections to be inserted
- [ ] Documentation includes TODO comments: "E15-F07 will use this infrastructure"
- [ ] Infrastructure available but not required to be used in this feature

---

### Could-Have Stories

**Story 6**: As a **user**, I want **consistent section ordering across all entity types** so that I can **quickly find information without cognitive overhead**.

**Acceptance Criteria**:
- [ ] All entity renders follow order: Header, Basic Info, Valid Transitions, Orchestrator Action, Related Docs, [Entity-Specific], Notes, Context Data
- [ ] Section ordering documented in render_common.go
- [ ] Integration tests verify section presence and order

---

### Edge Case & Error Stories

**Error Story 1**: As a **developer**, when **workflow config is missing or malformed**, I want to **see empty valid_transitions array** so that I can **handle gracefully without crashes**.

**Acceptance Criteria**:
- [ ] Missing `.sharkconfig.json` returns empty valid_transitions array
- [ ] Status not in status_flow returns empty valid_transitions array
- [ ] Malformed workflow config logs warning but doesn't crash display
- [ ] JSON output valid even with empty valid_transitions

**Error Story 2**: As an **AI agent**, when **entity has no related documents**, I want to **see empty related_documents array** so that I can **differentiate between "no docs" and "display error"**.

**Acceptance Criteria**:
- [ ] Entities with no related docs return empty related_documents array in JSON
- [ ] Human-readable output shows "Related Documents: None" instead of omitting section
- [ ] Consistent behavior across epic, feature, task displays

---

## Requirements

### Functional Requirements

**Category: Rendering Infrastructure**

1. **REQ-F-001**: Create render_common.go with Helper Functions
   - **Description**: Create new file `internal/cli/commands/render_common.go` with reusable helper functions for common display sections
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] File created with package `commands`
     - [ ] Imports: models, services, config, pterm
     - [ ] 8 helper functions implemented (see detailed list in AC)
     - [ ] Each function accepts standardized parameters (no entity-specific coupling)

2. **REQ-F-002**: Implement GetValidTransitions Helper
   - **Description**: Extract valid next statuses from workflow.Service status_flow configuration
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Function signature: `GetValidTransitions(status string, workflow *config.WorkflowConfig) []string`
     - [ ] Returns empty array if status not found in status_flow
     - [ ] Returns empty array if workflow config is nil/malformed
     - [ ] Unit tests cover: valid status, missing status, nil config, empty status_flow

3. **REQ-F-003**: Implement EntityDisplayOptions Struct
   - **Description**: Define configuration struct for unified rendering (used by E15-F07)
   - **User Story**: Links to Story 5
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Struct includes: EntityType, Key, Status, BasicInfo, ValidTransitions, OrchestratorAction, RelatedDocs, Notes, ContextData
     - [ ] RenderSpecific field is `func()` callback for entity-specific sections
     - [ ] All fields use common types (models.Document, services.OrchestratorAction, etc.)
     - [ ] Documentation comment explains E15-F07 future use

4. **REQ-F-004**: Implement RenderEntity Function
   - **Description**: Unified renderer that orchestrates section display in standard order
   - **User Story**: Links to Story 5, Story 6
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Function signature: `RenderEntity(opts EntityDisplayOptions)`
     - [ ] Calls helpers in order: renderHeader, renderBasicInfo, renderValidTransitions, renderOrchestratorAction, renderRelatedDocuments, [callback], renderNotes, renderContextData
     - [ ] Skips sections with nil/empty data (graceful degradation)
     - [ ] Documentation includes TODO: "E15-F07 will use this when refactoring commands"

**Category: Bug Fixes - Epic Display**

5. **REQ-F-005**: Add Related Documents to Epic Planning Mode
   - **Description**: Call renderRelatedDocuments() in renderEpicPlanning() function
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Modify epic.go: Add single call to renderRelatedDocuments(info.RelatedDocs) in renderEpicPlanning()
     - [ ] Related docs displayed after Orchestrator Action, before entity-specific sections
     - [ ] Lines changed in epic.go: < 15
     - [ ] Existing renderEpicPlanning() logic unchanged (no refactoring)

6. **REQ-F-006**: Add Valid Transitions to Epic JSON Output
   - **Description**: Include valid_transitions field in JSON response for epic get command
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Modify epic.go: Call GetValidTransitions() in runEpicGet() for JSON path
     - [ ] Add valid_transitions field to JSON response structure
     - [ ] All existing JSON fields preserved (backward compatible)
     - [ ] Lines changed in epic.go: < 10

**Category: Bug Fixes - Feature Display**

7. **REQ-F-007**: Add Related Documents to Feature Planning Mode
   - **Description**: Call renderRelatedDocuments() in renderFeaturePlanning() function
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Modify feature.go: Add single call to renderRelatedDocuments(info.RelatedDocs) in renderFeaturePlanning()
     - [ ] Related docs displayed after Orchestrator Action, before entity-specific sections
     - [ ] Lines changed in feature.go: < 15
     - [ ] Existing renderFeaturePlanning() logic unchanged (no refactoring)

8. **REQ-F-008**: Add Valid Transitions to Feature JSON Output
   - **Description**: Include valid_transitions field in JSON response for feature get command
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Modify feature.go: Call GetValidTransitions() in runFeatureGet() for JSON path
     - [ ] Add valid_transitions field to JSON response structure
     - [ ] All existing JSON fields preserved (backward compatible)
     - [ ] Lines changed in feature.go: < 10

**Category: Testing**

9. **REQ-F-009**: Unit Tests for Render Helpers
   - **Description**: Comprehensive test coverage for all helper functions in render_common.go
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Test file: `internal/cli/commands/render_common_test.go`
     - [ ] Each helper function has tests: renderHeader, renderBasicInfo, renderValidTransitions, renderOrchestratorAction, renderRelatedDocuments, renderNotes, renderContextData
     - [ ] GetValidTransitions() tested: valid status, missing status, nil config, malformed config
     - [ ] Edge cases covered: nil inputs, empty arrays, missing data

10. **REQ-F-010**: Integration Tests for Planning Mode Display
    - **Description**: Verify related docs and valid transitions display correctly in planning mode
    - **User Story**: Links to Story 1, Story 2
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] Test: Epic in ready_for_decomposition shows related docs and valid transitions
      - [ ] Test: Feature in ready_for_refinement_ba shows related docs and valid transitions
      - [ ] Test: JSON output includes both new fields
      - [ ] Regression test: Aggregation mode unchanged

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: No Additional Database Queries
   - **Description**: Rendering must not introduce new database calls beyond existing data fetching
   - **Measurement**: Database query count before/after feature
   - **Target**: 0 new queries introduced
   - **Justification**: Display logic should only format data already retrieved by DisplayService

2. **REQ-NF-002**: Rendering Time Unchanged
   - **Description**: Helper function overhead must not measurably increase display time
   - **Measurement**: Benchmark epic/feature get commands before/after
   - **Target**: < 5ms difference (within measurement noise)
   - **Justification**: Helpers should be zero-cost abstractions (just reorganizing existing code)

**Maintainability**

3. **REQ-NF-003**: Code Reusability
   - **Description**: Helper functions must be reusable across epic, feature, task rendering
   - **Implementation**: No entity-specific types in helper signatures
   - **Target**: 100% of helpers callable from any entity type
   - **Justification**: Foundation for E15-F07 consolidation

4. **REQ-NF-004**: Minimal Changes to Command Files
   - **Description**: Epic.go and feature.go changes must be surgical (<30 lines total)
   - **Implementation**: Add calls to helpers, avoid refactoring existing functions
   - **Target**: < 15 lines changed per file
   - **Justification**: Avoid disrupting code E15 will refactor anyway

**Backward Compatibility**

5. **REQ-NF-005**: JSON Structure Preservation
   - **Description**: All existing JSON fields must remain unchanged
   - **Implementation**: Only ADD fields (valid_transitions, ensure related_documents in planning mode)
   - **Compliance**: JSON schema versioning not required (additive changes only)
   - **Risk Mitigation**: Existing consumers (AI agents, scripts) continue working

6. **REQ-NF-006**: Human-Readable Output Preservation
   - **Description**: All existing display sections must remain visible and in same relative position
   - **Implementation**: Add new sections, don't remove or reorder existing
   - **Target**: Zero user complaints about missing information
   - **Justification**: Display changes should be purely additive

**Testing**

7. **REQ-NF-007**: Test Coverage for Helpers
   - **Description**: All helper functions must have unit test coverage
   - **Target**: 100% of exported helper functions tested
   - **Measurement**: Go test coverage report for render_common.go
   - **Justification**: Infrastructure code must be reliable for E15-F07 adoption

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: AI Agent Discovers Related Docs in Planning Mode**
- **Given** feature E15-F01 is in status `ready_for_refinement_ba`
- **And** feature has PRD and research report registered as related documents
- **When** agent executes `shark feature get E15-F01 --json`
- **Then** response includes `related_documents` array with 2 items
- **And** each item has `title`, `type`, `file_path` fields
- **And** orchestrator_action field is present
- **And** valid_transitions array includes ["in_refinement_ba", "blocked", "on_hold"]

**Scenario 2: Developer Sees Valid Transitions in Human Output**
- **Given** epic E07 is in status `active`
- **When** developer runs `shark epic get E07`
- **Then** output includes "Valid Transitions" section
- **And** section lists allowed next statuses: "completed", "on_hold"
- **And** section appears before "Related Documents" section
- **And** all existing sections (features table, rollups) still display

**Scenario 3: QA Verifies Test Plan Visibility**
- **Given** feature E07-F31 is in status `ready_for_test_planning`
- **And** QA has registered test plan as related document
- **When** QA runs `shark feature get E07-F31`
- **Then** "Related Documents" section displays test plan with title and path
- **And** "Orchestrator Action" section shows next agent is QA
- **And** "Valid Transitions" section shows ["in_test_planning", "blocked"]

**Scenario 4: Helper Functions Are Reusable**
- **Given** render_common.go is created with helper functions
- **When** developer calls renderRelatedDocuments([]*models.Document{...})
- **Then** function renders documents regardless of entity type (epic, feature, task)
- **And** function handles nil input gracefully (no panic)
- **And** function handles empty array (displays "None")

**Scenario 5: Error Handling - Missing Workflow Config**
- **Given** `.sharkconfig.json` is missing or malformed
- **When** user runs `shark feature get E07-F31 --json`
- **Then** response includes `valid_transitions: []` (empty array)
- **And** command completes successfully (no crash)
- **And** other sections display normally

**Scenario 6: Backward Compatibility - Existing JSON Consumers**
- **Given** AI agent expects existing JSON fields (epic, display_mode, phase, etc.)
- **When** agent parses `shark epic get E07 --json` response after this feature
- **Then** all existing fields are present with same types
- **And** new fields (valid_transitions) are additive
- **And** agent's existing parsing logic works unchanged

**Scenario 7: Minimal Code Changes**
- **Given** epic.go and feature.go before this feature
- **When** code review compares line diffs
- **Then** epic.go has < 15 lines changed
- **And** feature.go has < 15 lines changed
- **And** task.go has 0 lines changed
- **And** changes are additive calls to helpers (no refactoring)

---

## Out of Scope

### Explicitly Excluded

1. **Consolidating renderEpicPlanning() + renderEpicDetails() into single function**
   - **Why**: E15-F07 will perform full command refactoring when service layer is ready; doing this now creates duplicate work
   - **Future**: E15-F07 (service layer migration) will consolidate using RenderEntity()
   - **Workaround**: Infrastructure exists (RenderEntity function), just not used yet

2. **Consolidating renderFeaturePlanning() + renderFeatureDetails() into single function**
   - **Why**: Same as above - E15-F07 scope
   - **Future**: E15-F07 will use RenderEntity() infrastructure created here
   - **Workaround**: renderFeaturePlanning and renderFeatureDetails remain separate for now

3. **Refactoring task.go rendering**
   - **Why**: Task rendering has inline logic that needs larger refactor (E15-F07 scope)
   - **Future**: E15-F07 will apply same RenderEntity() pattern to tasks
   - **Workaround**: Task rendering unchanged; this feature establishes pattern for future application

4. **Removing workflow position display**
   - **Why**: Workflow position (linear path through statuses) is established output; removing it is breaking change
   - **Future**: Could be removed in major version bump if users prefer valid_transitions only
   - **Workaround**: Both workflow_position and valid_transitions displayed; users can ignore workflow_position

5. **CLI command layer calling helpers directly from existing render functions**
   - **Why**: Existing functions (renderEpicPlanning, renderFeaturePlanning) are monolithic; calling individual helpers mid-function requires refactoring
   - **Future**: E15-F07 will replace monolithic functions with RenderEntity() calls
   - **Workaround**: Helpers added at specific insertion points (related docs section, valid transitions in JSON)

6. **Converting all entity rendering to use EntityDisplayOptions pattern**
   - **Why**: Requires rewriting runEpicGet, runFeatureGet, runTaskGet functions - large change E15 will handle
   - **Future**: E15-F07 will convert all entity get commands to service calls that return EntityDisplayOptions
   - **Workaround**: Pattern exists, ready for E15-F07 adoption

---

### Alternative Approaches Rejected

**Alternative 1: Full Rendering Consolidation Now**
- **Description**: Consolidate renderEpicPlanning + renderEpicDetails, renderFeaturePlanning + renderFeatureDetails, refactor task.go into single RenderEntity() pattern across all entities
- **Why Rejected**:
  - E15 is actively refactoring CLI command layer to use service layer pattern
  - Doing full consolidation now creates merge conflicts and wasted work
  - Risk of breaking existing display output is higher with large changes
  - Minimal changes approach establishes foundation with lower risk
- **Trade-off**: Delayed consolidation benefits, but avoids duplicate refactoring work

**Alternative 2: Modify DisplayService Instead of CLI Commands**
- **Description**: Add valid_transitions and ensure related_docs in planning mode at DisplayService layer
- **Why Rejected**:
  - DisplayService is in services layer (business logic); display formatting belongs in CLI layer
  - Would violate architecture pattern (services shouldn't know about display logic)
  - CLI commands already receive DisplayInfo structs; formatting should happen at command layer
- **Trade-off**: More surgical changes to epic.go/feature.go, but respects architecture boundaries

**Alternative 3: Create Separate Planning/Aggregation Renderers**
- **Description**: Create unified renderPlanning() and renderAggregation() functions instead of EntityDisplayOptions pattern
- **Why Rejected**:
  - Still duplicates logic across entity types (need 6 functions: epic planning/aggregation, feature planning/aggregation, task planning/aggregation)
  - Doesn't solve root cause (duplication across entities)
  - EntityDisplayOptions pattern is more flexible for E15-F07 service layer integration
- **Trade-off**: Simpler immediate implementation, but doesn't establish foundation for E15-F07

---

## Success Metrics

### Primary Metrics

1. **Related Documents Visibility in Planning Mode**
   - **What**: Percentage of entity get commands in planning mode that display related documents
   - **Baseline**: 0% (currently hidden in planning mode)
   - **Target**: 100% (all entities in planning mode show related docs)
   - **Timeline**: Immediate (upon feature completion)
   - **Measurement**: Integration test verification + manual spot checks on E15, E07 entities

2. **Valid Transitions Availability**
   - **What**: Percentage of entity get commands that include valid_transitions in JSON output
   - **Baseline**: 0% (field doesn't exist)
   - **Target**: 100% (all entity types return valid_transitions array)
   - **Timeline**: Immediate
   - **Measurement**: Unit tests for epic/feature/task JSON responses

3. **Code Change Scope**
   - **What**: Total lines changed in epic.go + feature.go + task.go
   - **Target**: < 30 lines total (< 15 per file)
   - **Timeline**: Feature implementation
   - **Measurement**: Git diff line count

---

### Secondary Metrics

- **Helper Function Test Coverage**: 100% of helper functions have unit tests
- **Backward Compatibility**: 0 breaking changes to JSON output (additive only)
- **Regression Issues**: 0 reported display bugs in aggregation mode after changes
- **E15-F07 Adoption Readiness**: RenderEntity() infrastructure available and documented for E15-F07 use

---

## Dependencies & Integrations

### Dependencies

- **DisplayService** (`internal/services/display_service.go`): Provides EpicDisplayInfo and FeatureDisplayInfo structs containing related_documents data
- **workflow.Service** (`internal/workflow/service.go`): Provides status_flow configuration for valid transition extraction
- **DocumentRepository** (`internal/repository/document_repository.go`): Already fetched by DisplayService; no new queries needed
- **config.WorkflowConfig** (`internal/config/`): Contains status_flow map for GetValidTransitions() helper

### Integration Requirements

- **CLI Command Files**: epic.go, feature.go must call new helpers from render_common.go
- **JSON Output**: Epic and feature JSON responses must include valid_transitions field
- **pterm Library**: Helpers use existing pterm functions (PrintTable, box rendering, etc.)

---

## Open Questions & Assumptions

### Resolved During PRD Creation

All requirements are clear based on:
1. Existing feature.md with detailed implementation approach
2. Complexity triage report confirming integration points
3. Architecture documentation establishing helper pattern

### Remaining Open Questions

**NONE** - All requirements are clear. The following assumptions have been validated:

**Assumptions Validated**:
1. **Assumption**: DisplayService already fetches related documents for all modes
   - **Validation**: Triage report confirms DisplayService.GetEpicDisplayInfo and GetFeatureDisplayInfo already populate RelatedDocs field
   - **Impact**: No new repository calls needed

2. **Assumption**: workflow.Service exposes status_flow configuration
   - **Validation**: Architecture docs confirm workflow.Service provides access to status_flow map
   - **Impact**: GetValidTransitions() can extract transitions directly

3. **Assumption**: E15-F07 timeline allows for this infrastructure to be created first
   - **Validation**: Feature explicitly scoped as "foundation for E15-F07"
   - **Impact**: RenderEntity() infrastructure created but not required to be used yet

4. **Assumption**: Minimal changes (<30 lines) is achievable
   - **Validation**: Triage report estimates 10-15 lines per file (epic.go, feature.go)
   - **Impact**: Surgical approach confirmed feasible

5. **Assumption**: Related docs should display in all modes (planning and aggregation)
   - **Validation**: User story explicitly requires planning mode visibility
   - **Impact**: Related docs section added to both renderEpicPlanning and renderEpicDetails patterns

---

*Last Updated*: 2026-02-16
