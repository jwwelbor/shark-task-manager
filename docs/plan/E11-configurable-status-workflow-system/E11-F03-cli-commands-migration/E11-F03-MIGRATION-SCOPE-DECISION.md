# E11-F03 Migration Scope Decision Memo

**Date**: 2025-12-29
**Decision Maker**: BusinessAnalyst Agent
**Epic**: E11 - Configurable Status Workflow System
**Feature**: F03 - CLI Commands & Workflow Enforcement

---

## Executive Summary

**Decision**: Tasks T-E11-F03-005, T-E11-F03-006, and T-E11-F03-007 should be **REMOVED** from the feature scope.

**Rationale**: The term "migration" caused confusion. The epic scope document explicitly excludes "task data migration" but the feature tasks were created for data migration tooling. The actual "code migration" (refactoring Shark's codebase to use workflow config) is **already complete** in features F01-F04.

**Impact**:
- F03 scope reduced from 8 tasks to 5 tasks
- F03 completion: 4 of 5 tasks done (80% complete)
- Remaining work: Only T-E11-F03-008 (CLI command tests)
- Epic timeline unchanged (migration was never implemented)

---

## The Documentation Conflict

### What Was Unclear

There was a direct contradiction in E11 documentation:

1. **scope.md (lines 17-25)**: "Legacy Task Migration" is **OUT OF SCOPE**
   - "This epic targets new projects or projects without existing tasks"
   - "Migration adds significant complexity (data safety, rollback, testing)"
   - "Existing tasks can remain with legacy statuses (backward compatible)"

2. **requirements.md (REQ-NF-020)**: "Atomic Migration Transactions" is a **RELIABILITY REQUIREMENT**
   - "Workflow migration SHALL execute atomically (all tasks migrate or none do)"
   - Implementation details for migration transactions

3. **F03 Tasks (005-007)**: Migration tooling tasks exist
   - T-E11-F03-005: "Implement migration plan generation"
   - T-E11-F03-006: "Implement shark migrate workflow command"
   - T-E11-F03-007: "Implement migration rollback mechanism"

4. **Implementation Evidence**: Commit b448446 shows **NO migration code**
   - Only workflow commands implemented (list, validate, set-status)
   - No migration plan, no migrate command, no rollback mechanism

### User Clarification

User clarified the intent:
> "migration to update shark code" IS in scope
> "migration of existing task data" is OUT of scope

This confirms the scope document is correct: **data migration is out of scope**.

---

## Root Cause Analysis

The confusion stems from using "migration" to describe two different concepts:

### Type 1: Code Migration (IN SCOPE, COMPLETED)
**Definition**: Refactoring Shark's codebase to use configurable workflow system instead of hardcoded statuses

**Scope**: ✅ IN SCOPE for E11
**Status**: ✅ COMPLETED

**What Was Done**:
- F01: Workflow configuration loading from `.sharkconfig.json`
- F01: Workflow validation logic
- F01: Default workflow fallback for backward compatibility
- F02: Repository layer integration with workflow validation
- F02: Task status update methods using workflow config
- F03: CLI commands (workflow list, workflow validate, task set-status)
- F03: Refactored convenience commands (start, complete, approve) to use workflow validation

**Result**: Shark's codebase has been successfully migrated to use configurable workflows. The system no longer has hardcoded status checks.

---

### Type 2: Data Migration (OUT OF SCOPE, NOT IMPLEMENTED)
**Definition**: Migrating existing task records from old hardcoded statuses to new workflow-compatible statuses

**Scope**: ❌ OUT OF SCOPE for E11
**Status**: ❌ NOT IMPLEMENTED (correctly)

**Why Out of Scope**:
1. **Complexity**: Requires status mapping strategy, dry-run mode, rollback mechanism, extensive testing
2. **Risk**: Data safety concerns (partial migration corrupts database)
3. **Value**: Limited - default workflow matches legacy statuses (backward compatible)
4. **Target Audience**: E11 targets new projects or uses default workflow for existing projects
5. **Workaround**: Projects can use default workflow (no migration needed) or manually update statuses using `shark task set-status`

**What Was NOT Done** (correctly):
- No migration plan generation
- No `shark workflow migrate` command
- No migration rollback mechanism
- No status mapping configuration
- No dry-run mode for testing migrations

---

## Decision: Remove Tasks 005-007

### Tasks to Remove

1. **T-E11-F03-005: Implement migration plan generation**
   - Status: Todo (never implemented)
   - Action: DELETE from feature scope
   - Reason: Data migration is out of scope per `scope.md`

2. **T-E11-F03-006: Implement shark migrate workflow command**
   - Status: Todo (never implemented)
   - Action: DELETE from feature scope
   - Reason: Data migration is out of scope per `scope.md`

3. **T-E11-F03-007: Implement migration rollback mechanism**
   - Status: Todo (never implemented)
   - Action: DELETE from feature scope
   - Reason: Data migration is out of scope per `scope.md`

### Tasks to Keep

4. **T-E11-F03-001: Implement shark workflow list command**
   - Status: ✅ Completed (commit b448446)
   - Action: KEEP (implements REQ-F-011)

5. **T-E11-F03-002: Implement shark workflow validate command**
   - Status: ✅ Completed (commit b448446)
   - Action: KEEP (implements REQ-F-012)

6. **T-E11-F03-003: Implement shark task set-status command**
   - Status: ✅ Completed (commit b448446)
   - Action: KEEP (implements REQ-F-010)

7. **T-E11-F03-004: Update existing commands (start, complete, approve)**
   - Status: ✅ Completed (commit b448446)
   - Action: KEEP (implements REQ-F-013)

8. **T-E11-F03-008: Write CLI command tests with mocked repositories**
   - Status: ⏳ Todo
   - Action: KEEP (implements REQ-NF-040: >85% test coverage)
   - Priority: High (required for quality gates)

---

## Documentation Updates Made

### 1. Updated `requirements.md`

**Change**: Marked REQ-NF-020 as REMOVED with explanation

**Before**:
```
REQ-NF-020: Atomic Migration Transactions
- Description: Workflow migration SHALL execute atomically...
```

**After**:
```
REQ-NF-020: ~~Atomic Migration Transactions~~ REMOVED - OUT OF SCOPE
- Original Description: Workflow migration SHALL execute atomically...
- Decision: Data migration is explicitly OUT OF SCOPE per scope.md
- Rationale: E11 targets new projects or uses default workflow (backward compatible)
- Alternative: Use default workflow or manually update statuses with set-status
- Note: This requirement was orphaned from earlier design phase
```

**Rationale**: Keeps historical record of requirement decision while clarifying it's no longer applicable.

---

### 2. Updated `scope.md`

**Change**: Added clarification about "code migration" vs "data migration"

**Before**:
```
1. Legacy Task Migration
- What: Automatic or manual migration of existing tasks...
```

**After**:
```
1. Legacy Task Data Migration
- What: Automatic or manual migration of existing tasks...
- Important Clarification: "Code migration" (refactoring Shark's codebase to use
  workflow config) IS in scope and has been completed in F01-F04. This exclusion
  refers ONLY to task data migration.
```

**Rationale**: Eliminates ambiguity about what "migration" means in the context of E11.

---

### 3. Rewrote `feature.md` (Complete Replacement)

**Change**: Replaced template with comprehensive feature documentation

**What Was Added**:
- **Problem Statement**: Why CLI commands are needed
- **Solution Overview**: What was built (4 commands + refactored existing commands)
- **User Personas**: 3 personas (AI Agent, Project Manager, Tech Lead)
- **User Stories**: 6 stories with acceptance criteria (all marked complete)
- **Requirements Traceability**: Maps to REQ-F-010, 011, 012, 013
- **Out of Scope Section**: Explicitly documents migration removal decision
- **Implementation Summary**: Lists completed tasks (001-004), removed tasks (005-007), remaining task (008)
- **Success Metrics**: Command adoption, validation error prevention, force flag usage
- **Testing Strategy**: Unit tests (T-008), integration tests, regression tests

**Rationale**: Feature was 80% implemented but documentation was 0% complete (still template). This brings documentation in line with implementation reality.

---

## Implementation Guidance

### For Task Tracking System

Execute the following operations to align task database with feature scope:

```bash
# Delete migration tasks (no longer in scope)
shark task delete T-E11-F03-005  # migration plan generation
shark task delete T-E11-F03-006  # shark migrate workflow command
shark task delete T-E11-F03-007  # migration rollback mechanism

# Verify remaining tasks
shark task list --feature=E11-F03

# Expected output (5 tasks):
# T-E11-F03-001: Implement shark workflow list command (completed)
# T-E11-F03-002: Implement shark workflow validate command (completed)
# T-E11-F03-003: Implement shark task set-status command (completed)
# T-E11-F03-004: Update existing commands (completed)
# T-E11-F03-008: Write CLI command tests (todo)
```

**Note**: If task deletion isn't supported, mark tasks as "won't implement" or "cancelled" with notes referencing this decision memo.

---

### For F03 Completion

**Current Status**: 4 of 5 tasks complete (80%)

**Remaining Work**: Only T-E11-F03-008 (CLI command tests)

**To Complete Feature**:
1. Write unit tests for workflow commands (list, validate, set-status)
2. Write integration tests for refactored commands (start, complete, approve)
3. Use mocked repositories (no real database in CLI tests)
4. Achieve >85% test coverage (REQ-NF-040)
5. Test error handling, validation failures, force flag behavior

**Estimated Effort**: 4-8 hours (Medium complexity)

**Dependencies**: None (all implementation complete)

**Acceptance Criteria**:
- [ ] All workflow commands have unit tests with mocked repositories
- [ ] All refactored convenience commands have integration tests
- [ ] Test coverage >85% for workflow package
- [ ] Error handling tests cover invalid transitions, missing config, force flag
- [ ] Tests follow Shark testing patterns (see CLAUDE.md "Testing Architecture")

---

### For Future Data Migration (If Requested)

If user demand emerges for task data migration tooling, create a **separate epic**:

**Epic Concept**: E12 - Workflow Migration Tooling (tentative)

**Scope**:
- Automated status mapping (old status → new status)
- Dry-run mode (preview changes without applying)
- Atomic migration transactions (all or nothing)
- Rollback mechanism (undo migration)
- Migration audit trail
- Status mapping configuration
- Pre-migration validation (detect unmapped statuses)

**Estimated Effort**: 2-3 weeks (substantial complexity)

**Prerequisites**:
- E11 must be complete and stable
- Real-world use cases demonstrating need
- User stories from teams with existing projects

**Rationale**: Data migration is a separate concern with different risks, requirements, and value proposition. Keeping it separate maintains E11 focus on core workflow system.

---

## Lessons Learned

### Documentation Clarity

**Issue**: The term "migration" was ambiguous, causing confusion about scope.

**Improvement**: When using potentially ambiguous terms, define them explicitly upfront. For example:
- "Code migration" = refactoring codebase
- "Data migration" = changing existing records
- "Schema migration" = database structure changes

**Action**: Add glossary to epic.md for future features with technical terms.

---

### Requirements Lifecycle

**Issue**: REQ-NF-020 was orphaned when scope changed but not removed from requirements.md.

**Improvement**: When scope changes during planning:
1. Update requirements.md to mark removed requirements
2. Update scope.md to clarify exclusions
3. Cross-reference both documents (bidirectional links)
4. Document decision rationale

**Action**: Always review requirements.md when updating scope.md and vice versa.

---

### Task Planning

**Issue**: Tasks 005-007 were created despite explicit scope exclusion.

**Improvement**: Before creating tasks:
1. Verify feature is in scope per scope.md
2. Verify requirements exist for planned tasks
3. Check implementation commits for actual work done
4. Reconcile planned vs. actual implementation

**Action**: Use feature.md as single source of truth for task planning (reconciles scope, requirements, implementation).

---

## Conclusion

**Final Decision**: Remove tasks T-E11-F03-005, T-E11-F03-006, T-E11-F03-007 from F03 scope.

**Rationale**:
1. Data migration is explicitly out of scope per epic scope document
2. Code migration (refactoring Shark) is complete
3. No user demand for automated data migration
4. Default workflow provides backward compatibility
5. Manual migration via `set-status` is sufficient workaround

**Feature Status**: F03 is 80% complete (4 of 5 tasks done)

**Next Steps**:
1. Delete migration tasks from tracking system
2. Complete T-E11-F03-008 (CLI tests) to finish F03
3. Update epic progress tracking (F03: 80% → 100% when T-008 done)

**Documentation Status**: All documentation now consistent and actionable
- ✅ scope.md clarified
- ✅ requirements.md updated
- ✅ feature.md completed (no longer template)
- ✅ Decision rationale documented (this memo)

---

*Decision finalized by BusinessAnalyst Agent on 2025-12-29*
