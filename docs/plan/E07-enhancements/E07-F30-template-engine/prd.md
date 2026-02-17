---
feature_key: E07-F30-template-engine
epic_key: E07
title: External Template Engine for Orchestrator Instructions
description: Move instruction templates to external files with Go text/template engine
status: in_refinement_ba
version: 1.0
last_updated: 2026-02-14
---

# Feature PRD: External Template Engine for Orchestrator Instructions

**Epic**: E07 - Enhancements
**Feature**: E07-F30 - Template Engine
**Complexity Tier**: STANDARD
**Version**: 1.0
**Status**: In Refinement (BA)
**Last Updated**: 2026-02-14

---

## 1. Goal

### Problem Statement

Orchestrator instruction templates are currently embedded as JSON strings in `.sharkconfig.json`. With 62 templates across 3 workflows and growing complexity (complexity tiers, relational variables), this approach has become unmaintainable:

**Critical Pain Points**:
- **Unreadable**: 200-500+ character single-line JSON strings with no formatting
- **Unmaintainable**: 62 templates in one massive JSON file, terrible git diffs
- **No conditionals**: Can't hide empty sections (e.g., "Related docs: " when no docs exist)
- **Copy-paste hell**: Common patterns (TDD process, READ sections) duplicated across 18+ templates
- **JSON escaping nightmare**: Quotes, braces, newlines all need escaping
- **No validation**: Typos in placeholders only discovered at runtime

**Real-World Example**: Feature BA refinement template needs different output depth based on complexity tier (SIMPLE/STANDARD/COMPLEX), but current inline strings can't adapt - agents must interpret tier manually.

### Solution

Move instruction templates to external files and use Go's stdlib `text/template` engine with rich conditionals, shared partials, and validation.

**Key Benefits**:
- ✅ Multiline formatting (readable!)
- ✅ Conditionals hide empty sections (if/else)
- ✅ Smart auto-numbering (conditional sections renumber automatically)
- ✅ Complexity tier scaling (different output per tier)
- ✅ Shared partials (change once, update everywhere)
- ✅ Clean git diffs (line-by-line changes)
- ✅ Pre-compilation catches errors at startup

### Success Metrics

**Quantitative**:
- All 62 templates externalized by Phase 3
- Template rendering performance < 5ms per action
- 95%+ test coverage for template engine
- Zero breaking changes (100% backward compatible)

**Qualitative**:
- Template authors report improved readability
- Git diffs show line-by-line changes (not 500-char single-line)
- New templates authored 2x faster with partials
- Onboarding time reduced (no more JSON escaping)

---

## 2. Key Personas

### Primary: Template Author (AI/Human)
**Description**: Creates and maintains orchestrator instruction templates
**Needs**:
- Readable multiline formatting with syntax highlighting
- Conditionals to show/hide sections based on data
- Shared partials to avoid duplication
- Clear validation errors when templates are malformed

### Secondary: Development Team
**Description**: Extends workflows and adds new statuses
**Needs**:
- Simple template authoring without JSON escaping
- Testable templates with mock data
- Gradual migration (no flag day)
- Backward compatibility with existing inline templates

### Tertiary: AI Orchestrator
**Description**: Receives rendered instructions for task/feature/epic actions
**Needs**:
- Correctly rendered instructions with all variables populated
- No runtime template errors
- Fast rendering (< 5ms)

---

## 3. User Stories (MoSCoW)

### Must Have

#### US-E07-F30-001: Backward Compatible Template Engine
**As a** development team,
**I want** the template engine to detect `.tmpl` filenames and use Go templates while preserving inline string templates,
**So that** existing workflows continue working unchanged during migration.

**Acceptance Criteria**:
- **AC-1.1**: If `instruction_template` ends with `.tmpl`, use template engine
- **AC-1.2**: If `instruction_template` doesn't end with `.tmpl`, use legacy string replacement
- **AC-1.3**: All existing inline templates render identically (zero breaking changes)
- **AC-1.4**: Template engine singleton initialized at startup with precompilation
- **AC-1.5**: Template syntax errors fail fast at startup with helpful messages

---

#### US-E07-F30-002: External Template Files with Conditionals
**As a** template author,
**I want** to write templates in external `.tmpl` files with if/else conditionals,
**So that** I can hide empty sections and adapt output based on complexity tier.

**Acceptance Criteria**:
- **AC-2.1**: Templates support `{{if .RelatedDocs}}` conditionals to hide empty sections
- **AC-2.2**: Templates support `{{if eq .ComplexityTier "SIMPLE"}}` for tier-specific output
- **AC-2.3**: Templates support `{{else if}}` and `{{else}}` for multiple branches
- **AC-2.4**: Whitespace control with `{{- if}}` trims leading/trailing whitespace
- **AC-2.5**: Templates can use `{{range}}` for looping over related items
- **AC-2.6**: Custom functions available: `eq`, `ne`, `and`, `or`, `isEmpty`, `join`

---

#### US-E07-F30-003: Shared Partials
**As a** template author,
**I want** to define reusable partial templates (e.g., `_tdd_process.tmpl`),
**So that** I can change common sections once and update all templates using them.

**Acceptance Criteria**:
- **AC-3.1**: Partials use `_prefix.tmpl` naming convention (e.g., `_tdd_process.tmpl`)
- **AC-3.2**: Templates include partials with `{{template "tdd_process" .}}`
- **AC-3.3**: Partials stored in `templates/partials/` directory
- **AC-3.4**: At least 3 partials created: `_tdd_process.tmpl`, `_exit_gate.tmpl`, `_read_section.tmpl`
- **AC-3.5**: Changes to partials automatically affect all templates using them

---

#### US-E07-F30-004: High-Value Template Migration
**As a** development team,
**I want** 12 most complex templates converted to external files in Phase 2,
**So that** I can validate the approach and see immediate maintainability wins.

**Acceptance Criteria**:
- **AC-4.1**: Task execution templates (5): `ready_for_development`, `ready_for_code_review`, `ready_for_qa`, `ready_for_refinement_ba`, `ready_for_refinement_tech`
- **AC-4.2**: Feature planning templates (4): `ready_for_research`, `ready_for_refinement_ba`, `ready_for_refinement_tech`, `ready_for_test_planning`
- **AC-4.3**: Epic strategic templates (3): `ready_for_research`, `ready_for_feasibility_review_ba`, `ready_for_feasibility_review_tech`
- **AC-4.4**: `.sharkconfig.json` updated to reference `.tmpl` files instead of inline strings
- **AC-4.5**: All 12 templates render identically to inline versions (regression test)

---

### Should Have

#### US-E07-F30-005: Template Validation Command
**As a** template author,
**I want** a command to validate all template syntax,
**So that** I catch errors before runtime.

**Acceptance Criteria**:
- **AC-5.1**: `shark config validate --templates` checks syntax of all `.tmpl` files
- **AC-5.2**: Validation reports line numbers and specific syntax errors
- **AC-5.3**: Validation precompiles all templates to catch parse errors
- **AC-5.4**: Exit code 0 if all valid, 1 if any errors

---

#### US-E07-F30-006: Full Template Migration
**As a** development team,
**I want** all 62 templates externalized by Phase 3,
**So that** the entire system benefits from external template maintainability.

**Acceptance Criteria**:
- **AC-6.1**: All 62 orchestrator action templates converted to `.tmpl` files
- **AC-6.2**: Partial library expanded to 6-10 reusable partials
- **AC-6.3**: Documentation includes template authoring guide
- **AC-6.4**: Deprecation notice added for inline templates (warning, not error)
- **AC-6.5**: Full workflow regression tests pass

---

### Could Have

#### US-E07-F30-007: Migration Tool
**As a** template author,
**I want** a tool to auto-convert inline templates to external files,
**So that** I don't have to manually copy-paste 62 templates.

**Acceptance Criteria**:
- **AC-7.1**: `shark config migrate-templates` extracts inline templates to files
- **AC-7.2**: Tool creates `.tmpl` files in appropriate directories (task/, feature/, epic/)
- **AC-7.3**: Tool updates `.sharkconfig.json` to reference new files
- **AC-7.4**: Tool preserves template content exactly (no semantic changes)
- **AC-7.5**: Tool creates backup of `.sharkconfig.json` before modification

---

### Won't Have

#### US-E07-F30-008: Non-Go Template Engines
**Out of Scope**: Jinja2, Handlebars, or other template engines
**Rationale**: Go stdlib `text/template` is sufficient, already used in codebase

#### US-E07-F30-009: Visual Template Editor
**Out of Scope**: GUI or web-based template editor
**Rationale**: CLI/text-file workflow is sufficient for target audience

#### US-E07-F30-010: Template Marketplace
**Out of Scope**: Central repository for sharing templates
**Rationale**: Git-based sharing is sufficient

---

## 4. Functional Requirements

### FR-1: Backward Compatible Detection
**Description**: System automatically detects whether to use template engine or legacy string replacement
**Priority**: Must Have
**Verification**: Unit tests verify both `.tmpl` files and inline strings render correctly

### FR-2: Template Engine Implementation
**Description**: Go `text/template` engine with custom functions and partial support
**Priority**: Must Have
**Details**:
- Singleton engine initialized at startup
- Precompilation of all templates catches syntax errors early
- Custom functions: `eq`, `ne`, `and`, `or`, `isEmpty`, `join`, `isSimple`, `isStandard`, `isComplex`
- Partial template support with `{{template "name" .}}`

**Verification**: Integration tests verify conditionals, partials, and custom functions work

### FR-3: Directory Structure
**Description**: Templates organized by entity type with shared partials
**Priority**: Must Have
**Structure**:
```
templates/
├── epic/
│   ├── ready_for_research.tmpl
│   └── ready_for_refinement_ba.tmpl
├── feature/
│   ├── ready_for_refinement_ba.tmpl
│   └── ready_for_test_planning.tmpl
├── task/
│   ├── ready_for_development.tmpl
│   └── ready_for_code_review.tmpl
└── partials/
    ├── _tdd_process.tmpl
    ├── _exit_gate.tmpl
    └── _read_section.tmpl
```

**Verification**: Directory structure exists, templates load correctly

### FR-4: Config Schema (No Changes)
**Description**: Existing `instruction_template` field works for both inline and file references
**Priority**: Must Have
**Details**:
- No new config fields required
- Filename suffix `.tmpl` triggers template engine
- Non-`.tmpl` values use legacy string replacement

**Verification**: Config validation tests pass for both formats

### FR-5: Template Variables
**Description**: All existing placeholder variables available to templates
**Priority**: Must Have
**Details**:
- Reuse `*PlaceholdersWithRelated` functions from `internal/config/template_helpers.go`
- Convert `map[string]string` to template-compatible data structure
- Include complexity tier, related docs/tasks/features/epics

**Verification**: Template variables match existing placeholder output

---

## 5. Non-Functional Requirements

### NFR-1: Performance
- Template rendering must complete in < 5ms per action
- Precompiled templates cached in memory
- No performance degradation vs. string replacement

### NFR-2: Backward Compatibility
- Zero breaking changes to existing workflows
- All existing inline templates continue working
- Gradual migration at user's pace

### NFR-3: Error Handling
- Template syntax errors fail fast at startup (not runtime)
- Clear error messages with line numbers
- Graceful degradation if template file missing (log error, return empty)

### NFR-4: Testability
- 95%+ test coverage for template engine code
- Unit tests for all custom functions
- Integration tests for full workflow rendering

---

## 6. Acceptance Criteria Summary

### Phase 1: Foundation (MVP)
- [ ] Engine supports both inline and file templates (US-E07-F30-001)
- [ ] Conditionals and custom functions work (US-E07-F30-002)
- [ ] Shared partials work (US-E07-F30-003)
- [ ] Zero breaking changes
- [ ] Template rendering tests passing

### Phase 2: High-Value Templates
- [ ] 12 high-value templates converted (US-E07-F30-004)
- [ ] 3+ reusable partials created
- [ ] `.sharkconfig.json` references `.tmpl` files
- [ ] Regression tests pass (all templates render identically)

### Phase 3: Full Migration
- [ ] All 62 templates externalized (US-E07-F30-006)
- [ ] 6-10 partials in library
- [ ] Template validation command available (US-E07-F30-005)
- [ ] Migration tool available (US-E07-F30-007, optional)
- [ ] Documentation complete

---

## 7. Out of Scope

### Explicitly NOT Included

1. **Non-Go Template Engines**: Jinja2, Handlebars, Mustache - Go stdlib sufficient
2. **Visual Template Editor**: CLI/text-file workflow only
3. **Template Marketplace**: Just git for sharing
4. **YAML Config Migration**: Keep `.sharkconfig.json` as JSON
5. **Template Inheritance**: Base templates with overrides (future enhancement)
6. **Hot Reload**: Watch `templates/` directory for changes (future enhancement)
7. **Template Linting**: Enforce conventions, max complexity (future enhancement)

---

## 8. Dependencies

### Required (Blocking)
- **E07-F29: Template Variables for Related Docs** (✅ COMPLETED 2026-02-14)
  - Provides `{related_docs}`, `{related_tasks}`, `{related_features}`, `{related_epics}` variables
  - Integration point: `*PlaceholdersWithRelated()` functions
  - Status: Satisfied - can reference related entities in templates immediately

### Optional (Non-Blocking)
- **E07-F21: Add Actions to Status Transition** (ACTIVE, 78%)
  - Establishes orchestrator action patterns
  - E07-F30 makes E07-F21 actions more maintainable
  - No direct code dependency

### Infrastructure
- Go stdlib `text/template` (already available)
- Existing `PopulateTemplate` function (for backward compatibility)
- Existing template infrastructure in `internal/templates/`

---

## 9. Integration Points

### Service Layer
**Files**: `internal/services/epic_service.go`, `internal/services/feature_service.go`, `internal/services/display_service.go`
**Method**: `resolveAction(ctx, entity, status)`
**Integration**: Calls `action.PopulateTemplate(vars)` - single integration point

### Config Layer
**File**: `internal/config/orchestrator_action.go`
**Method**: `OrchestratorAction.PopulateTemplate(vars map[string]string) string`
**Changes**:
```go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    // If template ends with .tmpl, use template engine
    if strings.HasSuffix(oa.InstructionTemplate, ".tmpl") {
        return templateEngine.Render(oa.InstructionTemplate, vars)
    }

    // Otherwise, use legacy string replacement
    return legacyPopulateTemplate(oa.InstructionTemplate, vars)
}
```

### Template Helpers
**File**: `internal/config/template_helpers.go`
**Functions**: `TaskPlaceholdersWithRelated`, `FeaturePlaceholdersWithRelated`, `EpicPlaceholdersWithRelated`
**Changes**: Add complexity tier to returned placeholder maps (minimal extension)

### Existing Template Infrastructure
**File**: `internal/templates/renderer.go`
**Pattern**: Replicate existing task markdown template rendering for orchestrator templates
**Reuse**: Template loading, `text/template` usage, custom functions pattern

---

## 10. Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Breaking existing inline templates | High | Low | Filename detection (`.tmpl` suffix) maintains backward compatibility |
| Template syntax errors at runtime | Medium | Medium | Precompile templates at startup, fail fast with helpful errors |
| Performance degradation | Low | Low | Cache compiled templates, benchmark against string replacement |
| Template complexity explosion | Medium | Medium | Lint templates, enforce max complexity limits (Phase 4) |
| Partial template conflicts | Low | Low | Use `_` prefix convention, namespace by entity type |

---

## 11. Implementation Phases

### Phase 1: Foundation (2-3 days)
- Create `internal/templates/orchestrator_renderer.go`
- Add `.tmpl` detection to `OrchestratorAction.PopulateTemplate()`
- Create `templates/` directory structure
- Write unit tests for renderer
- Add template precompilation at startup

**Deliverable**: Engine ready, 100% backward compatible

---

### Phase 2: High-Value Templates (3-4 days)
- Convert 12 most complex templates to external files
- Create 3 reusable partials
- Update `.sharkconfig.json` to reference `.tmpl` files
- Write integration tests

**Deliverable**: 12 external templates, immediate readability wins

---

### Phase 3: Full Migration (4-5 days)
- Convert remaining 50 templates
- Expand partial library to 6-10 partials
- Add `shark config validate --templates` command
- Create optional migration tool
- Update documentation
- Add deprecation notice for inline templates

**Deliverable**: Complete external template system

---

### Phase 4: Advanced Features (Future)
- Template inheritance (base templates with overrides)
- Template testing framework (`shark config test-templates`)
- Hot reload in dev mode (watch templates/, reload on change)
- Template linting (enforce conventions, max complexity)
- Template library (shareable across projects)

---

## 12. Related Documents

**Research**:
- Feature Research Report: `docs/plan/E07-enhancements/E07-F30-template-engine/feature-research-tactical.md`
- External Proposal: `/home/jwwel/.claude/docs/external-prompt-templates-proposal.md`
- Variable Guide: `/home/jwwel/.claude/docs/instruction-template-variable-usage-guide.md`

**Codebase References**:
- Orchestrator Action: `internal/config/orchestrator_action.go`
- Template Helpers: `internal/config/template_helpers.go`
- Epic Service: `internal/services/epic_service.go` (lines 202-238)
- Feature Service: `internal/services/feature_service.go` (lines 210-238)
- Existing Template Renderer: `internal/templates/renderer.go`

**Configuration**:
- Current Config: `.sharkconfig.json`
- Advanced Profile: `internal/init/profiles/advanced.json`

---

## 13. Requirements Traceability

### Epic E07 Requirements
**Epic Goal**: Provide enhancements to Shark CLI
**Epic Requirement**: Vary by feature (placeholder epic)

**Feature E07-F30 Traceability**:
- Enhances CLI workflow by improving template maintainability
- Builds on existing infrastructure (E07-F29 template variables)
- Aligns with epic's purpose of incremental CLI improvements

### Parent Epic Alignment
- No specific epic requirements to trace (E07 is a placeholder epic)
- Feature is self-contained enhancement
- Does not conflict with other E07 features
- Complements E07-F21 (orchestrator actions) and E07-F29 (template variables)

---

*Last Updated*: 2026-02-14
*Status*: Ready for Technical Refinement
*Next Step*: Architect review for technical design approval
