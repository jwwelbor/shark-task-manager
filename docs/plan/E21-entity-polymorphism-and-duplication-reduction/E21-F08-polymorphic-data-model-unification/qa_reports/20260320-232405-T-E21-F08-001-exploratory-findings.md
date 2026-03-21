# Exploratory Testing Findings: T-E21-F08-001

**Task:** Schema Migration: Create Polymorphic Tables and Migrate Data
**Tested by:** QA Agent
**Date:** 2026-03-20 23:24:05

---

## Exploratory Testing Charter

"Explore the polymorphic table migration to discover data integrity issues, idempotency failures, or backward compatibility regressions."

---

## Findings

### Finding 1: Compatibility Views Are a Strong Addition (Positive)

The implementation goes beyond the spec by creating backward-compatible views with INSTEAD OF triggers for the dropped document tables. This means existing repository code that references `epic_documents`, `feature_documents`, etc. continues to work transparently. This was not in the original spec but is an excellent engineering decision that reduces risk for downstream tasks.

**Severity**: N/A (positive finding)

### Finding 2: task_history Retention is Well-Documented

The decision to retain `task_history` as a live TABLE (rather than dropping it per the spec) is documented in:
- Code comments in db.go
- Code review report
- Task notes (note #626)

The rationale is clear: TaskHistoryRepository still queries task_history directly. Dropping it before T-E21-F08-004 would break status history commands. Data is dual-written (existing data copied to entity_history; task_history retained for existing consumers).

**Severity**: N/A (noted deviation, already approved)
**Downstream**: T-E21-F08-004 must address this by migrating TaskHistoryRepository to use entity_history and then dropping task_history.

### Finding 3: No Display View Breakage Observed

Display data views (`epic_display_data`, `feature_display_data`, `task_display_data`) were updated to reference `entity_documents` instead of the old per-entity tables. The full test suite passes, confirming no display regressions.

### Finding 4: EC-4 FK Filter Prevents Silent Data Loss

The migration correctly filters `WHERE document_id IN (SELECT id FROM documents)` during both INSERT and verification phases. This means orphaned document references (where the parent document was deleted but the link record remains) are not migrated -- which is correct behavior. The verification uses the same filter, so counts match and the migration succeeds.

---

## Bugs Found

None.

---

## Recommendations for Downstream Tasks

1. **T-E21-F08-004**: Must migrate TaskHistoryRepository to use entity_history, then drop the task_history TABLE and replace it with a compatibility view (similar to what was done for document tables).

2. **T-E21-F08-003**: Should leverage the compatibility views during the transition period but ultimately query entity_documents directly.

3. **Future migrations**: The pattern established here (create new table -> copy data -> verify -> drop old -> create compat view) is reusable for F11 (polymorphic relationships) and F12 (polymorphic acceptance criteria).
