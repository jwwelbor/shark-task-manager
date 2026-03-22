# Assessment & Triage Summary - E21-F17

**Date:** 2026-03-22
**Feature:** Task Deps Service Layer Cleanup
**Previous Status:** ready_for_assessment
**Current Status:** in_specification
**Next Agent:** Architect (for specification writing)

---

## Work Completed

### 1. Scope Validation ✅

**Finding:** Correctly scoped as FEATURE (not misclassified task)

**Evidence:**
- 3 distinct, sequenced subtasks (move, batch, unify)
- Design decisions required (where to place relationship logic)
- 5+ files affected (task_deps.go, services, repository, tests)
- Cross-layer architectural refactoring (CLI → service → repository)

### 2. Complexity Triage ✅

**Complexity Score: 11/27 → STANDARD TIER**

**Technical Assessment (7/6 points):**
- File Impact: 2/3 (5 files, localized to task dependencies)
- Pattern Novelty: 1/3 (follows E15/E21 service layer pattern)
- Data Model: 1/3 (no schema changes, existing models)
- API Surface: 2/3 (4 new public methods)
- Cross-Feature Deps: 1/3 (self-contained, minimal ecosystem impact)
- UI Complexity: 0/3 (refactoring only, no new UI)

**Execution Assessment (4/3 points):**
- Task Estimation: 1/3 (straightforward, ~40-60 LOC per task)
- Regression Risk: 2/3 (moderate, N+1 query changes need validation)
- Execution Effort: 1/3 (clear steps, established patterns)

**Tier Justification:**
- **Why STANDARD (not SIMPLE):** 3 distinct tasks, design decisions, moderate regression risk, cross-layer impact
- **Why not COMPLEX:** Follows established patterns, no schema changes, localized scope, straightforward steps

### 3. Routing Decision ✅

**Route:** ready_for_specification

Per triage framework:
- SIMPLE (0-6) → ready_for_task_generation
- **STANDARD (7-15) → ready_for_specification** ← This feature
- COMPLEX (16+) → ready_for_research

---

## Context for Architecture Phase

### Problem Statement
The task dependencies command (`task_deps.go`) contains ~150 lines of business logic in CLI command handlers:
- `getTaskRelationshipsViaEntityRel` - filters and resolves relationships
- `getTaskBlockedByViaEntityRel` - fetches task dependencies
- `getTaskBlocksViaEntityRel` - fetches tasks that depend on this one
- Tree-building functions - uses N+1 query patterns

### Current Issues
1. **Fat controller anti-pattern:** Business logic in CLI commands (violates E15 architecture)
2. **N+1 queries:** Loop calls `taskSvc.GetTaskByID()` once per relationship
3. **Duplication:** Three nearly-identical resolve-and-build loops

### Solution Scope
**Task 1:** Move relationship logic to service layer
- Add `GetTaskRelationships(ctx, taskKey, typeFilter)` to EntityRelationshipService
- Add `GetTaskBlockedBy(ctx, taskKey)` service method
- Add `GetTaskBlocks(ctx, taskKey)` service method
- Refactor CLI commands to call service methods
- Remove helper functions from task_deps.go

**Task 2:** Add batch GetTasksByIDs to repository
- Eliminate N+1 queries
- Single batch fetch instead of loop

**Task 3:** Unify duplicate resolve patterns
- Extract shared `resolveTaskRelationships()` helper
- All three helpers delegate to shared logic

### Files Affected
- `internal/cli/commands/task_deps.go` - CLI command refactoring
- `internal/services/entity_relationship_service.go` - new service methods
- `internal/repository/task_repository.go` - new batch method
- `internal/services/*_test.go` - new tests with mocks
- `internal/repository/*_test.go` - batch method repository tests

### Architecture Alignment
- **Thin CLI pattern:** Commands become parse → call service → format
- **Service layer:** EntityRelationshipService owns relationship orchestration
- **Repository layer:** TaskRepository owns data access, batch fetching
- **No schema changes:** Works with existing models and tables

---

## Exit Gate Verification

All assessment exit gate criteria met:

- ✅ Scope clearly defined (3 sequenced tasks)
- ✅ Complexity assessed with justification (STANDARD = 11/27)
- ✅ Architecture decisions reference existing patterns (E15, E21)
- ✅ File list provided (4 primary files + tests)
- ✅ Problem statement clear with current code examples
- ✅ Solution approach well-scoped per task breakdown

---

## Next Steps for Architect

1. **Write detailed specifications** per task
   - Acceptance criteria for each task
   - Method signatures with documentation
   - Test plan (happy path, error cases, N+1 elimination verification)

2. **Design review:**
   - Where should methods live? (EntityRelationshipService vs dedicated service)
   - Batch fetch strategy for tree-building
   - Backward compatibility with existing CLI

3. **Test strategy:**
   - Service tests with mocks (no DB)
   - Repository tests with real DB for batch method
   - Integration verification of N+1 elimination

---

## Assessment Artifacts

- `/ASSESSMENT.md` - Detailed complexity triage breakdown
- `/ASSESSMENT_SUMMARY.md` - This summary document
- Task files already created with initial requirements:
  - `tasks/T-E21-F17-001.md` - Move helpers to service layer
  - `tasks/T-E21-F17-002.md` - Add batch GetTasksByIDs
  - `tasks/T-E21-F17-003.md` - Unify duplicate resolve patterns
