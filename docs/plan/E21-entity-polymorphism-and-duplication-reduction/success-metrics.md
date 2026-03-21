# Success Metrics

**Epic**: [Entity Polymorphism and Duplication Reduction](./epic.md)

---

## Overview

This document defines the measurable indicators that determine whether the entity polymorphism refactoring has achieved its goals. Since this is an internal architecture epic with no end-user-facing changes, all metrics are developer-experience and code-quality metrics.

**Measurement Timeline**: Metrics are evaluated at each feature boundary (F01 through F05) and at epic completion.

---

## Primary Success Metrics

### Metric 1: Service Layer Line Reduction

**Type**: Lagging

**What We're Measuring**:
Net reduction in total lines of code across entity service files (`internal/services/*_service.go`). This is the primary quantitative measure of duplication elimination.

**How We'll Measure**:
- **Data Source**: `wc -l internal/services/*_service.go` before and after each feature
- **Calculation Method**: (lines before) - (lines after) = net reduction. New shared files (entity_service.go) are counted in the "after" total.
- **Measurement Frequency**: At each feature boundary

**Success Criteria**:
- **Baseline**: Current service layer total (measured at epic start, estimated ~5,806 lines across 8 files)
- **Target**: Net reduction of 800+ lines (conservative estimate; 1,255 lines of identified duplication minus ~455 lines of new shared code)
- **Timeline**: At F03 completion (status transition unification accounts for ~400 of the duplicated lines)
- **Minimum Viable**: Net reduction of 500+ lines

**Relates To**:
- **Requirement(s)**: REQ-NF-007
- **User Journey**: Journey 2 (cross-cutting bug fix), Journey 3 (cross-cutting feature)
- **Business Value**: Less code to maintain, fewer places for bugs to hide

---

### Metric 2: Files Required for New Entity Type

**Type**: Leading

**What We're Measuring**:
The number of files that must be created or modified to add a fully functional entity type with CRUD, status transitions, notes, context, and document linking.

**How We'll Measure**:
- **Data Source**: Documentation review of the "new entity checklist" after F01+F02 completion. Optionally validated by a spike that adds a placeholder entity.
- **Calculation Method**: Count files created + files modified for a hypothetical 6th entity type
- **Measurement Frequency**: After F02 completion (registry), verified at epic completion

**Success Criteria**:
- **Baseline**: 15+ files (documented in epic.md Current State Analysis, validated by E18 experience)
- **Target**: 5 or fewer files
- **Timeline**: After F02 completion
- **Minimum Viable**: 8 or fewer files (50% reduction)

**Relates To**:
- **Requirement(s)**: REQ-NF-008
- **User Journey**: Journey 1 (adding a new entity type)
- **Business Value**: Future entity types (Milestone, Sprint, etc.) become feasible for a single developer in 1-2 days

---

### Metric 3: Switch Statement Elimination

**Type**: Lagging

**What We're Measuring**:
The count of per-entity switch/case branches in cross-cutting services (NoteService, ContextService, ResumeService). Each switch statement is a maintenance liability that must be extended when a new entity type is added.

**How We'll Measure**:
- **Data Source**: `grep -c 'case models.EntityType' internal/services/note_service.go internal/services/context_service.go internal/services/resume_service.go`
- **Calculation Method**: Count of case branches that dispatch based on entity type
- **Measurement Frequency**: After F02 completion

**Success Criteria**:
- **Baseline**: ~14 switch branches across 3 services (2 in NoteService x 5 entities + 2 in ContextService x 5 entities, partially overlapping)
- **Target**: 0 switch branches dispatching by entity type in cross-cutting services
- **Timeline**: After F02 completion
- **Minimum Viable**: 0 (this is binary -- partial elimination still leaves the extension problem)

**Relates To**:
- **Requirement(s)**: REQ-F-005, REQ-F-006, REQ-F-007
- **User Journey**: Journey 1 (new entity author does not need to add switch branches)
- **Business Value**: Eliminates the most error-prone extension point for new entity types

---

### Metric 4: Test Pass Rate at Phase Boundaries

**Type**: Leading

**What We're Measuring**:
Whether all existing tests pass at each feature boundary without modification to test assertions or expected behavior. This confirms the refactoring is safe and backward-compatible.

**How We'll Measure**:
- **Data Source**: `make test` output at each feature boundary
- **Calculation Method**: (passing tests) / (total tests) x 100%
- **Measurement Frequency**: After each feature (F01, F02, F03, F04, F05)

**Success Criteria**:
- **Baseline**: Current test pass rate (should be 100%)
- **Target**: 100% pass rate at every feature boundary
- **Timeline**: Continuous throughout epic
- **Minimum Viable**: 100% (no exceptions; failing tests indicate behavioral regression)

**Relates To**:
- **Requirement(s)**: REQ-NF-001, REQ-NF-009
- **User Journey**: All journeys (the refactoring must not break existing behavior)
- **Business Value**: Confidence that the refactoring is safe; no regressions shipped

---

## Secondary Success Metrics

### Metric 5: Setter Method Elimination

**Type**: Lagging

**What We're Measuring**:
The count of `SetXxxRepo()` methods on cross-cutting services that exist solely to wire per-entity repository dependencies (e.g., `SetBugRepo()`, `SetChangeCardRepo()`).

**How We'll Measure**:
- **Data Source**: `grep -c 'func.*Set.*Repo' internal/services/note_service.go internal/services/context_service.go internal/services/resume_service.go`
- **Calculation Method**: Count of setter methods
- **Measurement Frequency**: After F02 completion

**Success Criteria**:
- **Baseline**: ~10-15 setter methods across cross-cutting services
- **Target**: 0 setter methods (replaced by registry injection via constructor)
- **Timeline**: After F02 completion
- **Minimum Viable**: 0

**Relates To**:
- **Requirement(s)**: REQ-F-005, REQ-F-006, REQ-F-012
- **User Journey**: Journey 1 (no setter methods to add for new entity types)

---

### Metric 6: Interface Satisfaction Compile Checks

**Type**: Leading

**What We're Measuring**:
Whether all 5 entity types have compile-time interface satisfaction checks (the `var _ Entity = (*Epic)(nil)` pattern).

**How We'll Measure**:
- **Data Source**: `grep 'var _ Entity' internal/models/*.go`
- **Calculation Method**: Count of compile-time checks (should be 5, one per entity type)
- **Measurement Frequency**: After F01 completion

**Success Criteria**:
- **Baseline**: 0 (no Entity interface exists)
- **Target**: 5 compile-time checks (one per entity type)
- **Timeline**: After F01 completion
- **Minimum Viable**: 5

**Relates To**:
- **Requirement(s)**: REQ-F-002, REQ-NF-005
- **User Journey**: Journey 4 (new contributors see clear interface contract)

---

## Success Criteria Summary

The epic is considered **successful** if:

1. Net reduction of 800+ lines in the service layer (Metric 1)
2. A new entity type requires 5 or fewer files to add (Metric 2)
3. Zero entity-type switch statements remain in cross-cutting services (Metric 3)
4. 100% test pass rate maintained at every feature boundary (Metric 4)
5. Zero `SetXxxRepo()` setter methods remain on cross-cutting services (Metric 5)
6. All 5 entity types have compile-time Entity interface satisfaction checks (Metric 6)

The epic is considered **minimally successful** if metrics 2, 3, and 4 are met (the architectural goal of extensibility is achieved even if line reduction falls short of the 800-line target).

---

*See also*: [Requirements](./requirements.md)
