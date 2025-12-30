# E11 Epic Requirements Assessment

**Date**: 2025-12-29
**Assessor**: ProductManager Agent
**Epic**: E11 - Configurable Status Workflow System

---

## Executive Summary

**Recommendation**: **Option B - Dispatch business-analyst to resolve documentation conflict**

The E11 epic has **comprehensive documentation**, but there is a **critical contradiction** between `scope.md` and `requirements.md` regarding data migration. This conflict has created confusion in the F03 task breakdown, with 3 migration-related tasks (005-007) that may or may not be valid depending on which document is authoritative.

**Critical Conflict**:
- `scope.md` (lines 17-25): "Legacy Task Migration" is **OUT OF SCOPE**
- `requirements.md` (REQ-NF-020): "Atomic Migration Transactions" is a **RELIABILITY REQUIREMENT**

**Recommended Actions**:
1. **DISPATCH business-analyst** to:
   - Resolve the scope.md vs requirements.md conflict
   - Make authoritative decision: Is data migration in or out of scope?
   - If OUT: Remove REQ-NF-020, delete F03 tasks 005-007, update scope
   - If IN: Write proper requirements for F03 tasks 005-007, update scope.md, justify the complexity
2. **KEEP** task T-E11-F03-008 (CLI command tests) regardless of decision
3. **UPDATE** all feature.md files to reflect actual implementations (currently all templates)

---

## Requirements Documentation Quality Assessment

### Epic-Level Documentation: ✅ EXCELLENT

**Files Reviewed**:
- `/docs/plan/E11-configurable-status-workflow-system/epic.md`
- `/docs/plan/E11-configurable-status-workflow-system/requirements.md`
- `/docs/plan/E11-configurable-status-workflow-system/scope.md`

**Strengths**:
1. **Clear Problem/Solution/Impact** - Epic clearly articulates the business need
2. **Comprehensive Requirements** - 22 functional + non-functional requirements with MoSCoW prioritization
3. **Explicit Scope Boundaries** - `scope.md` clearly defines what's OUT of scope
4. **Measurable Success Criteria** - Specific KPIs (40% custom workflow adoption, <5% force flag usage)
5. **Risk Documentation** - Alternative approaches documented and rejected with rationale

**Requirements Coverage**:
- ✅ Must-Have: REQ-F-001 through REQ-F-013 (13 requirements)
- ✅ Should-Have: REQ-F-014 through REQ-F-017 (4 requirements)
- ✅ Could-Have: REQ-F-018 through REQ-F-019 (2 requirements)
- ✅ Non-Functional: REQ-NF-001 through REQ-NF-041 (12 requirements)

**Total**: 31 requirements, all clearly documented

---

## Feature-Level Documentation Quality Assessment

### F01: Workflow Configuration & Validation - ❌ TEMPLATE ONLY
**Status**: Implementation complete (commit 3849a39), but feature.md is still template

### F02: Repository Integration - ❌ TEMPLATE ONLY
**Status**: Unknown implementation status, feature.md is template

### F03: CLI Commands & Migration - ❌ TEMPLATE ONLY + SCOPE CONFLICT
**Status**: Partial implementation (tasks 001-004 complete), but:
- Feature.md is still template
- Contains migration tasks that violate epic scope

### F04: Agent Targeting & Metadata - ❌ TEMPLATE ONLY
**Status**: Partial implementation (3 tasks complete per commit af3228d, c7f458b)

### F05: Workflow Visualization - ❌ TEMPLATE ONLY
**Status**: Not started

**Critical Gap**: All feature.md files are empty templates despite significant implementation work being complete

---

## The "Migration" Scope Conflict

### CRITICAL DOCUMENTATION CONFLICT DETECTED

There is a **direct contradiction** between `scope.md` and `requirements.md`:

#### What Scope.md Says (OUT OF SCOPE):

From `/docs/plan/E11-configurable-status-workflow-system/scope.md`, lines 17-25:

> **1. Legacy Task Migration**
> - **What**: Automatic or manual migration of existing tasks from old hardcoded statuses to new configurable workflow statuses
> - **Why It's Out of Scope**:
>   - This epic targets new projects or projects without existing tasks
>   - Migration adds significant complexity (data safety, rollback, testing)
>   - Current Shark installations can adopt new workflow for new tasks only
>   - Existing tasks can remain with legacy statuses (backward compatible)

**Conclusion from scope.md**: Data migration is explicitly OUT OF SCOPE.

#### What Requirements.md Says (IN SCOPE):

From `/docs/plan/E11-configurable-status-workflow-system/requirements.md`, lines 269-273:

> **REQ-NF-020**: Atomic Migration Transactions
> - **Description**: Workflow migration SHALL execute atomically (all tasks migrate or none do)
> - **Implementation**: Wrap migration in single SQLite transaction with `BEGIN TRANSACTION` / `COMMIT`
> - **Testing**: Test migration failure mid-process, verify rollback occurs
> - **Justification**: Prevents partial migrations that corrupt database state

**Conclusion from requirements.md**: Migration transactions are a RELIABILITY requirement.

#### Analysis of the Conflict

**Hypothesis**: REQ-NF-020 may refer to "schema migration" (database structure changes) rather than "data migration" (changing task status values).

However, the wording "all tasks migrate or none do" strongly suggests **data migration**, not schema migration.

**Implication**: Either:
1. The scope.md decision to exclude migration was made AFTER requirements.md was written (orphaned requirement)
2. REQ-NF-020 is about a different type of migration not covered by scope.md exclusion
3. There's a documentation error - one document is wrong

**Evidence from Implementation**: Commit b448446 shows NO migration code. This suggests scope.md is correct and REQ-NF-020 is an orphaned requirement.

---

### What F03 Tasks Say (IN SCOPE):

From task files:
- `T-E11-F03-005.md`: "Implement migration plan generation"
- `T-E11-F03-006.md`: "Implement shark migrate workflow command"
- `T-E11-F03-007.md`: "Implement migration rollback mechanism"

**Problem**: These tasks imply implementing a `shark workflow migrate` command to migrate existing task data, which directly contradicts the scope document.

---

### What Was Actually Implemented (ALIGNED WITH SCOPE):

From commit `b448446` (E11-F03 tasks 001-004):

**Implemented Commands**:
1. `shark workflow list` - Display configured workflow
2. `shark workflow validate` - Validate workflow config
3. `shark task set-status` - Generic status transition command
4. Updated `start`, `complete`, `approve` to use workflow validation

**No Migration Tooling**: The actual implementation contains ZERO migration functionality.

**Conclusion**: The codebase implementation is correct and aligns with scope. The task breakdown is wrong.

---

## What "Migration" Actually Means in E11 Context

There are **two types of migration** that could be considered:

### 1. **Data Migration** (Migrating Task Status Values)
- **Description**: Updating existing task records in database to use new workflow statuses
- **Epic Scope**: ❌ **EXPLICITLY OUT OF SCOPE** (scope.md lines 17-25)
- **Status**: Not implemented, should not be implemented

### 2. **Code Migration** (Updating Shark Codebase to Use Workflows)
- **Description**: Refactoring existing hardcoded status checks to use workflow config
- **Epic Scope**: ✅ **IN SCOPE** (this is the entire point of E11)
- **Status**: ✅ **ALREADY COMPLETE**
  - F01: Workflow config loading and validation ✅
  - F02: Repository layer integration with workflow validation ✅
  - F03: CLI commands using workflow system ✅
  - F04: Agent targeting via workflow metadata (partial) ✅

**Conclusion**: "Code migration" is already done. The system has been migrated to use configurable workflows.

---

## Task Status Analysis

### F03 Current Status

From database query:
```
T-E11-F03-001: Implement shark workflow list command (completed)
T-E11-F03-002: Implement shark workflow validate command (completed)
T-E11-F03-003: Implement shark task set-status command (completed)
T-E11-F03-004: Update existing commands (start, complete, approve) (completed)
T-E11-F03-005: Implement migration plan generation (todo) ← DELETE
T-E11-F03-006: Implement shark migrate workflow command (todo) ← DELETE
T-E11-F03-007: Implement migration rollback mechanism (todo) ← DELETE
T-E11-F03-008: Write CLI command tests with mocked repositories (todo) ← KEEP
```

**Duplicate Tasks**: Database shows duplicate task keys (T-E11-F03-009 through T-E11-F03-019) with same titles. Likely sync issue.

---

## Recommendation: Delete Migration Tasks

### Rationale

1. **Scope Violation**: Tasks 005-007 contradict the explicit scope boundary in `scope.md`
2. **Already Complete**: The "code migration" work these tasks might have intended is already done
3. **No Requirements**: There are no requirements (REQ-F-xxx) in `requirements.md` for data migration tooling
4. **Implementation Evidence**: Commit b448446 shows NO migration code was implemented, only workflow commands
5. **Backward Compatibility**: Epic design uses default workflow fallback, making data migration unnecessary

### What Scope.md Recommends Instead (lines 24-25)

> - **Workaround**: For existing projects, use default workflow (matches legacy statuses) or manually update task statuses as needed

Translation: Don't migrate data; just use the default workflow which matches existing statuses.

---

## Recommended Action Plan

### Immediate Actions (ProductManager)

1. **Delete Redundant Tasks**:
   ```bash
   shark task delete T-E11-F03-005  # migration plan generation
   shark task delete T-E11-F03-006  # shark migrate workflow command
   shark task delete T-E11-F03-007  # migration rollback mechanism
   ```

2. **Keep Testing Task**:
   - T-E11-F03-008: "Write CLI command tests with mocked repositories" is legitimate
   - Aligns with REQ-NF-040 (test coverage >85%)
   - Testing is always in scope

3. **Update F03 Feature Documentation**:
   - Replace template content in `E11-F03-cli-commands-migration/feature.md`
   - Document actual implementation:
     - User stories for workflow list/validate/set-status commands
     - Acceptance criteria matching REQ-F-010, REQ-F-011, REQ-F-012, REQ-F-013
     - Remove "Migration" from feature title (rename to "CLI Commands & Workflow Enforcement")

4. **Fix Task Duplicates**:
   - Investigate why database has duplicate tasks (T-E11-F03-009 through T-E11-F03-019)
   - Likely sync issue; run `shark sync --dry-run` to diagnose

### Follow-Up Actions (Business-Analyst - NOT NEEDED)

**Business-analyst is NOT needed** because:
- Requirements are complete and comprehensive (31 requirements documented)
- Scope is clearly defined with explicit boundaries
- Problem is task breakdown error, not missing requirements
- Solution is to delete erroneous tasks, not write new requirements

### Optional Enhancement (Future)

If there's genuine user demand for data migration (unlikely given scope rationale), address in a **future epic**:
- Epic E12: "Workflow Migration Tooling" (not currently planned)
- Would require separate PRD with data safety, rollback, testing requirements
- Estimated 2-3 weeks of work (see scope.md, lines 193-199 for context)

---

## F03 Completion Criteria

Once tasks 005-007 are deleted, F03 will have 5 total tasks:

1. ✅ T-E11-F03-001: Implement shark workflow list command (completed)
2. ✅ T-E11-F03-002: Implement shark workflow validate command (completed)
3. ✅ T-E11-F03-003: Implement shark task set-status command (completed)
4. ✅ T-E11-F03-004: Update existing commands (start, complete, approve) (completed)
5. ⏳ T-E11-F03-008: Write CLI command tests with mocked repositories (todo)

**Remaining Work**: Only task 008 (CLI tests)

**F03 Progress**: 4/5 complete (80%) → Will be 100% when T-E11-F03-008 is done

---

## E11 Epic Overall Status

### Features Implementation Status

| Feature | Tasks Total | Tasks Complete | Progress | Status |
|---------|-------------|----------------|----------|--------|
| F01: Workflow Config & Validation | 10 | ~5 | ~50% | In Progress |
| F02: Repository Integration | 10 | ~5 | ~50% | In Progress |
| F03: CLI Commands | 5* | 4 | 80% | Nearly Complete |
| F04: Agent Targeting & Metadata | 10 | 3 | 30% | In Progress |
| F05: Workflow Visualization | 8 | 0 | 0% | Not Started |

*After deleting tasks 005-007

### Epic Completion Estimate

**Must-Have Requirements Status**:
- REQ-F-001 through REQ-F-007: ✅ Complete (F01, F02)
- REQ-F-010 through REQ-F-013: ✅ Complete (F03)
- REQ-F-014 through REQ-F-017: ⏳ Partial (F04, 3/4 complete)

**Estimated Completion**:
- Core functionality (Must-Have): ~75% complete
- Should-Have (F04): ~75% complete
- Could-Have (F05): 0% complete (optional visualization)

**Recommendation**: Focus on completing F04 (agent targeting). F05 (visualization) is "Could-Have" and can be deferred.

---

## Conclusion

**Answer to Original Question**: Business-analyst IS needed to resolve documentation conflict.

**Root Cause of Confusion**: Documentation conflict between `scope.md` (migration OUT) and `requirements.md` (migration IN via REQ-NF-020).

**Path Forward - Dispatch Business-Analyst**:

Business-analyst should receive the following directive:

---

### Business-Analyst Task: Resolve E11 Migration Scope Conflict

**Context**: E11 epic has conflicting documentation:
- File: `/docs/plan/E11-configurable-status-workflow-system/scope.md` (lines 17-25) says "Legacy Task Migration" is OUT OF SCOPE
- File: `/docs/plan/E11-configurable-status-workflow-system/requirements.md` (REQ-NF-020) includes "Atomic Migration Transactions" as a reliability requirement

**Your Mission**:

1. **Review Evidence**:
   - Read scope.md section on "Legacy Task Migration"
   - Read REQ-NF-020 in requirements.md
   - Review git commit b448446 (shows NO migration code implemented)
   - Review tasks T-E11-F03-005, 006, 007 (migration tasks with template content)

2. **Make Authoritative Decision**:
   - **Option 1 - Migration is OUT OF SCOPE** (recommended based on implementation evidence):
     - Remove REQ-NF-020 from requirements.md
     - Delete tasks T-E11-F03-005, T-E11-F03-006, T-E11-F03-007
     - Update F03 feature title to "CLI Commands & Workflow Enforcement" (remove "Migration")
     - Document decision rationale in scope.md

   - **Option 2 - Migration is IN SCOPE**:
     - Update scope.md to include migration as in-scope
     - Write complete requirements for T-E11-F03-005, 006, 007 tasks
     - Update F03 feature.md with:
       - User stories for migration commands
       - Acceptance criteria for data safety, rollback, dry-run
       - Migration strategy (status mapping, edge cases)
     - Justify why scope changed from original decision
     - Estimate additional 2-3 weeks of work

3. **Update Documentation**:
   - Ensure scope.md and requirements.md are consistent
   - Update F03 feature.md (currently template) to reflect decision
   - If migration IN scope: Create detailed migration specification document

4. **Deliverables**:
   - Updated scope.md (consistent with decision)
   - Updated requirements.md (add or remove REQ-NF-020)
   - Updated F03 feature.md (remove template, add actual content)
   - Decision memo explaining rationale

---

**ProductManager Follow-Up Actions** (after business-analyst completes):
- If migration OUT: Delete tasks 005-007, focus on completing F04
- If migration IN: Review migration requirements, adjust E11 timeline, create new tasks if needed
- Complete T-E11-F03-008 (CLI tests) regardless of decision
- Update all other feature.md files to replace templates with actual implementation docs

---

*Assessment completed by ProductManager Agent on 2025-12-29*
