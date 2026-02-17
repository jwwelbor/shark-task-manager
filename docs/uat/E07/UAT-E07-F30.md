# UAT Test Guide - Template Engine

**Feature:** E07-F30 - template engine
**Epic:** E07 - enhancements
**Generated:** 2026-02-15
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Provide a place to link enhancements for the shark CLI

**This Feature's Role:** Moves orchestrator instruction templates from embedded JSON strings to external `.tmpl` files with Go's text/template engine, enabling rich formatting, conditionals, partials, and maintainability for 62+ orchestration templates across epic/feature/task workflows.

**Related Features:**
- E07-F29 - Template Variables for Related Docs and Tasks (completed, 83.87%) - **Integration**: Provides template variables ({related_docs}, {related_tasks}, {related_features}, {related_epics}) that E07-F30 templates will use
- E07-F28 - Orchestration Action on Get (completed, 100%) - **Integration**: Populates orchestrator actions that E07-F30 templates will render
- E07-F27 - Embedded JSON Workflow Profiles and Init Merge (completed) - **Integration**: Provides workflow profile infrastructure that stores template references
- E07-F26 - Centralized Workflow Service Authority (active, 92.22%) - **Integration**: Workflow service validates statuses that trigger template rendering
- E07-F24 - Workflow Profile Support (active, 90%) - **Integration**: Workflow profiles define status metadata that references external templates

**Integration Points:**
- **E07-F29 → E07-F30**: Template variables from F29 populate template data in F30's rendering engine
- **E07-F28 → E07-F30**: Orchestrator actions from F28 specify which templates to render in F30
- **Template Files ↔ .sharkconfig.json**: Configuration references external `.tmpl` files via `instruction_template` field
- **Backward Compatibility**: Inline string templates continue working alongside external `.tmpl` files (gradual migration)

---

## Design Intent

**From Epic PRD:**
> Simple epic to hold enhancements for the shark CLI

**From Feature PRD:**
> **Problem**: Orchestrator instruction templates are currently embedded as JSON strings in `.sharkconfig.json`. This approach has hit critical pain points:
> 1. **Unreadable**: Templates are 200-500+ character single-line JSON strings with no formatting
> 2. **Unmaintainable**: 62 instruction templates across 3 workflows, all embedded in one massive JSON file
> 3. **No conditionals**: Can't hide empty sections (e.g., "Related docs: " when no docs exist)
> 4. **Copy-paste hell**: Common patterns (TDD process, READ sections, EXIT GATE) duplicated across 18+ templates

> **Solution**: Move instruction templates to external files and use Go's stdlib `text/template` engine.

**Key Design Decisions:**
- Use Go's `text/template` (stdlib, no external dependencies)
- Backward compatible: `.tmpl` suffix triggers engine, otherwise legacy string replacement
- Fail-fast on syntax errors at startup (precompilation)
- Singleton pattern for thread-safe template engine
- Partial templates for reusable sections (`_tdd_process`, `_exit_gate`, `_read_section`)
- Complexity tier scaling (SIMPLE/STANDARD/COMPLEX → different output depth)

---

## Cross-Feature Integration Tests

### Integration Scenario 1: Template Variables from E07-F29
**Features:** E07-F29 (template variables) + E07-F30 (template engine)
**Scenario:** External templates use {related_docs}, {related_tasks} variables to dynamically include related entity references

Steps:
1. Create task with related docs and tasks metadata
2. Render external template with template engine
3. Verify {related_docs} and {related_tasks} placeholders populate correctly
4. Verify conditionals hide empty sections when no related entities exist

Expected Result: Template renders with dynamic content based on relational data from E07-F29

### Integration Scenario 2: Orchestrator Actions from E07-F28
**Features:** E07-F28 (orchestrator action on get) + E07-F30 (template engine)
**Scenario:** `shark get <entity>` retrieves orchestrator action that references external `.tmpl` file

Steps:
1. Configure status metadata with `instruction_template: "task/ready_for_development.tmpl"`
2. Run `shark get T-E07-F30-001` (task in ready_for_approval status)
3. Verify orchestrator action renders using external template
4. Verify rendered instruction includes all placeholders populated

Expected Result: Orchestrator action shows fully rendered template content from external file

### Integration Scenario 3: Workflow Profiles and Template References
**Features:** E07-F24/E07-F27 (workflow profiles) + E07-F30 (template engine)
**Scenario:** Workflow profile configuration references external `.tmpl` files for status-specific instructions

Steps:
1. Verify .sharkconfig.json status_metadata entries reference `.tmpl` files
2. Trigger status transition (e.g., task to ready_for_code_review)
3. Verify correct template loaded based on new status
4. Verify template renders with status-specific context

Expected Result: Status transitions load and render correct external templates based on workflow profile config

---

## Epic Acceptance Validation

| Epic AC | Description | Feature Contribution | Status |
|---------|-------------|---------------------|--------|
| General enhancements | Collection of shark CLI improvements | Provides external template engine for orchestration instructions | [ ] |

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| AC-1.4 | Template engine singleton initialized at startup with precompilation | [ ] |
| AC-1.5 | Template syntax errors fail fast at startup with helpful messages | [ ] |
| AC-2.1 | External .tmpl files support multiline formatting | [ ] |
| AC-2.2 | Templates support if/else/else-if conditionals | [ ] |
| AC-2.3 | Templates support {{range}} loops | [ ] |
| AC-2.6 | Custom template functions available (eq, ne, isEmpty, tier helpers) | [ ] |
| AC-3.1 | Partials use _prefix.tmpl naming convention | [ ] |
| AC-3.2 | Partial include works via {{template "name" .}} | [ ] |
| AC-3.3 | Partials stored in templates/partials/ directory | [ ] |
| AC-3.4 | At least 3 partials created (TDD, exit gate, read section) | [ ] |
| AC-4.1 | Task execution templates (5) converted to external .tmpl | [ ] |
| AC-4.2 | Feature planning templates (4) converted to external .tmpl | [ ] |
| AC-4.3 | Epic strategic templates (3) converted to external .tmpl | [ ] |
| AC-4.5 | All converted templates render identically to inline versions | [ ] |
| AC-5.1 | .sharkconfig.json references Phase 2 .tmpl files | [ ] |
| AC-6.1 | Backward compatibility: inline templates still work | [ ] |
| AC-6.2 | Detection logic: .tmpl suffix triggers engine, else legacy | [ ] |

---

## Test Scenarios

### Scenario 1: Template Engine Initialization and Precompilation
**Tasks covered:** T-E07-F30-001 (Create OrchestratorRenderer)

**Steps:**
1. Start shark CLI (triggers template engine initialization)
2. Verify template engine singleton created
3. Verify all .tmpl files in templates/ precompiled without errors
4. Introduce syntax error in a .tmpl file → restart shark
5. Verify fail-fast error with filename and line number

**Success Criteria:**
- [ ] Engine initializes on first use (lazy loading)
- [ ] All valid templates precompile successfully
- [ ] Syntax errors fail fast at startup with clear error messages
- [ ] Singleton pattern enforced (same instance across calls)

---

### Scenario 2: External Template Rendering with Conditionals
**Tasks covered:** T-E07-F30-002 (.tmpl detection logic)

**Steps:**
1. Configure task status with external template: `"instruction_template": "task/ready_for_development.tmpl"`
2. Get task with no related_docs or related_tasks
3. Verify output has no empty "Related docs:" or "Related tasks:" lines
4. Update task with related_docs="prd.md"
5. Get task again → verify "Related docs: prd.md" appears
6. Add related_tasks="E07-F29-003"
7. Verify both sections appear with smart numbering (1, 2, 3)

**Success Criteria:**
- [ ] .tmpl files detected and render via engine
- [ ] Conditionals hide empty sections
- [ ] Smart numbering auto-adjusts based on conditional visibility
- [ ] Inline templates still work (backward compatibility)

---

### Scenario 3: Partial Template Includes
**Tasks covered:** T-E07-F30-003 (Create partials)

**Steps:**
1. Verify partials exist: _tdd_process.tmpl, _exit_gate.tmpl, _read_section.tmpl
2. Check templates/task/ready_for_development.tmpl includes {{template "_tdd_process" .}}
3. Render development template → verify TDD process steps appear
4. Modify _tdd_process.tmpl content
5. Restart engine → verify all templates using partial reflect changes

**Success Criteria:**
- [ ] Partials use _prefix.tmpl naming
- [ ] Partial includes work via {{template "name" .}}
- [ ] Changes to partials propagate to all including templates
- [ ] Context passed to partials correctly

---

### Scenario 4: Complexity Tier Scaling
**Tasks covered:** T-E07-F30-004 (Extend placeholder builders with complexity_tier)

**Steps:**
1. Create task with complexity_tier="SIMPLE"
2. Render template with tier-aware conditionals
3. Verify SIMPLE output (brief instructions)
4. Update task to complexity_tier="STANDARD"
5. Re-render → verify STANDARD output (focused instructions)
6. Update task to complexity_tier="COMPLEX"
7. Re-render → verify COMPLEX output (comprehensive instructions)

**Success Criteria:**
- [ ] complexity_tier field available as template variable
- [ ] Templates adapt output based on tier value
- [ ] isSimple(), isStandard(), isComplex() helpers work
- [ ] Tier scaling reduces/expands instruction depth appropriately

---

### Scenario 5: Phase 2 Task Template Conversion
**Tasks covered:** T-E07-F30-005 (Convert 5 task execution templates)

**Steps:**
1. Verify 5 task templates exist in templates/task/:
   - ready_for_development.tmpl
   - ready_for_code_review.tmpl
   - ready_for_qa.tmpl
   - ready_for_refinement_ba.tmpl
   - ready_for_refinement_tech.tmpl
2. For each template:
   a. Render with test data
   b. Compare output to inline template baseline
   c. Verify semantic equivalence (ignoring whitespace)
3. Verify templates include partials where appropriate
4. Verify conditionals hide empty sections

**Success Criteria:**
- [ ] All 5 task templates created
- [ ] Templates render identically to inline versions (semantic equivalence)
- [ ] Multiline formatting improves readability
- [ ] No blank lines for empty conditional sections

---

### Scenario 6: Phase 2 Feature and Epic Template Conversion
**Tasks covered:** T-E07-F30-006 (4 feature templates), T-E07-F30-007 (3 epic templates)

**Steps:**
1. Verify 4 feature templates exist in templates/feature/:
   - ready_for_research.tmpl
   - ready_for_refinement_ba.tmpl
   - ready_for_refinement_tech.tmpl
   - ready_for_test_planning.tmpl
2. Verify 3 epic templates exist in templates/epic/:
   - ready_for_research.tmpl
   - ready_for_feasibility_review_ba.tmpl
   - ready_for_feasibility_review_tech.tmpl
3. For each template:
   a. Render with entity-appropriate test data
   b. Verify semantic equivalence to inline version
4. Check feature/epic templates use shared partials (_exit_gate, _read_section)

**Success Criteria:**
- [ ] All 4 feature templates created
- [ ] All 3 epic templates created
- [ ] Templates render identically to inline versions
- [ ] Shared partials used across entity types

---

### Scenario 7: Config Update for Phase 2 Templates
**Tasks covered:** T-E07-F30-008 (Update .sharkconfig.json)

**Steps:**
1. Open .sharkconfig.json
2. Verify status_metadata entries for converted statuses reference .tmpl files:
   - `"instruction_template": "task/ready_for_development.tmpl"`
   - `"instruction_template": "feature/ready_for_refinement_ba.tmpl"`
   - etc.
3. Verify legacy inline templates still exist for unconverted statuses
4. Run `shark get <entity>` for status with external template
5. Verify orchestrator action renders using external file
6. Run `shark get <entity>` for status with inline template
7. Verify orchestrator action renders using legacy string replacement

**Success Criteria:**
- [ ] Config references all Phase 2 .tmpl files
- [ ] External templates render correctly
- [ ] Inline templates still work (backward compatibility)
- [ ] No broken template references

---

### Scenario 8: Regression Testing - Backward Compatibility
**Tasks covered:** All tasks (general compatibility validation)

**Steps:**
1. Create baseline renders for all inline templates (before migration)
2. Convert templates to external files
3. Re-render with same test data
4. Compare outputs:
   - Semantic content must match exactly
   - Whitespace/formatting differences acceptable
5. Verify no changes to existing shark commands
6. Verify no changes to task/feature/epic data structures
7. Run full test suite → all tests pass

**Success Criteria:**
- [ ] External templates semantically identical to inline versions
- [ ] No breaking changes to CLI commands
- [ ] No breaking changes to data models
- [ ] All existing tests pass
- [ ] Gradual migration possible (no flag day required)

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | (none yet) |
| Result | - |
| Results File | - |

**Previous Sessions:** None
