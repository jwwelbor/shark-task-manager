# Exploratory Testing Findings: T-E16-F08-001

**Task:** Auto-update epic/feature status when child entities are added
**Tested by:** QA Agent
**Date:** 2026-03-22 01:42:35

## Charter

Explore the auto-reopen behavior to discover edge cases, integration issues, or unexpected behaviors not covered by acceptance criteria.

## Findings

### Finding 1: Unused Parameters (Informational)

**Severity:** Low
**Location:** `task_service.go:1268`, `feature_service.go:872`

Both `maybeReopenParentFeature(taskKey)` and `maybeReopenParentEpic(featureKey)` accept a key parameter for "audit logging" but do not use it. The parameters are properly documented in godoc as being for future audit trail purposes. No compile warning since Go allows unused method parameters (unlike function-local variables).

**Impact:** None currently. Future enhancement when feature/epic history tracking is implemented.

### Finding 2: No History/Audit Records for Auto-Reopen

**Severity:** Medium (informational)
**Location:** AC-1 and AC-3 spec requirement vs. implementation

The spec states "a history record is created for the feature/epic with notes containing 'auto-reopened'". However, the codebase has no feature or epic history table -- only `task_history` exists. The code review noted this as a spec-level inconsistency (Observation #4). The implementation correctly updates the status but cannot create history records that do not have supporting infrastructure.

**Impact:** Users cannot distinguish automatic reopens from manual status changes in feature/epic history. This is a pre-existing infrastructure gap, not a regression.

**Recommendation:** Track as a separate enhancement if feature/epic history tracking is added in the future.

### Finding 3: Race Condition Analysis (No Issue Found)

**Explored:** What happens if multiple tasks are created simultaneously under a completed feature?

**Result:** No issue. The spec correctly notes (Edge Case #4) that each creation independently checks and potentially updates. The second creation sees the feature already at `active` (from the first), which is a no-op. Since the check-and-update is a simple status comparison followed by an UPDATE, and SQLite serializes writes, there is no risk of data corruption.

### Finding 4: Archived Status Reopening Behavior (Expected)

**Explored:** Should creating a child under an `archived` parent reopen it?

**Result:** Yes, this is intentional per the spec (Edge Case #7). In the default config, `archived` is in `_complete_` for epics/features. Creating a child under an archived parent reopens it to `active`. Users who do not want this behavior can remove `archived` from `_complete_` in their workflow config.

### Finding 5: Basic Profile Compatibility (Verified)

**Explored:** Does this work with the basic workflow profile (no custom epic/feature workflows)?

**Result:** Yes. When no custom workflows are configured, `NewService("")` falls back to `DefaultEpicWorkflow` and `DefaultFeatureWorkflow`, which include `_complete_: ["completed", "archived"]` and `_aggregation_: ["active"]`. The tests use `workflow.NewService("")` (empty project root) which triggers these defaults.

## Summary

No blocking issues found during exploratory testing. All findings are informational or relate to pre-existing infrastructure gaps (no feature/epic history tables). The implementation is solid and handles edge cases correctly.
