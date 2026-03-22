# UAT Evidence Compilation: T-E16-F08-001

**Feature:** E16-F08 -- Enhancements and Maintenance
**Epic:** E16 -- Multi Level Workflow
**Task:** T-E16-F08-001 -- Auto-update epic/feature status when child entities are added
**Compiled by:** UAT Agent (Claude: evidence collection only)
**Date:** 2026-03-22

## Evidence Inventory

- Epic: E16 -- Multi Level Workflow (status: active)
- Feature: E16-F08 -- Enhancements and Maintenance (status: draft)
- Task: T-E16-F08-001 (status: in_approval)
- QA reports found: 2 (qa-results + exploratory-findings, timestamp 20260322-014235)
- Code reviews found: 1 (timestamp 20260322-013608)
- Test artifacts found: No dedicated test artifact directory
- Known issues from shark notes: 4 non-blocking observations documented in code review

## Acceptance Criteria and Evidence Mapping

### AC-1: Task creation reopens completed feature
- **Source:** Task spec lines 104-110
- **Evidence:** QA report AC-1 section (MET), Code review AC-1 (MET)
- **Test:** `TestTaskService_CreateTask_ReopensTerminalFeature`
- **Code location:** `task_service.go:364` (creatorSvc path), `task_service.go:428` (fallback path)
- **Note:** AC-1 also requires "a history record is created for the feature with notes containing auto-reopened". Code review Observation #4 and exploratory Finding #2 note that no feature/epic history table exists in the codebase. This is a pre-existing infrastructure gap.

### AC-2: Task creation reopens cancelled feature
- **Source:** Task spec lines 112-117
- **Evidence:** QA report AC-2 section (MET with note), Code review AC-2 (PARTIALLY MET)
- **Test:** `TestTaskService_CreateTask_ReopensArchivedFeature` (tests `archived`, not `cancelled`)
- **Conflict:** Code review Observation #1 notes that `cancelled` is NOT in the default `_complete_` list. Test uses `archived` instead. Logic path is identical for any terminal status. Custom aggregation test demonstrates custom `_complete_` values work.

### AC-3: Feature creation reopens completed epic
- **Source:** Task spec lines 119-125
- **Evidence:** QA report AC-3 section (MET), Code review AC-3 (MET)
- **Test:** `TestFeatureService_CreateFeature_ReopensTerminalEpic`
- **Code location:** `feature_service.go:860`
- **Note:** Same history record gap as AC-1 (see AC-1 note).

### AC-4: No-op when parent is already non-terminal
- **Source:** Task spec lines 127-132
- **Evidence:** QA report AC-4 section (MET), Code review AC-4 (MET)
- **Test:** `TestTaskService_CreateTask_NoReopenNonTerminalFeature`

### AC-5: No-op when parent is in planning status
- **Source:** Task spec lines 134-138
- **Evidence:** QA report AC-5 section (MET), Code review AC-5 (MET)
- **Test:** Covered by AC-4 test (same `IsTerminalStatus` check)

### AC-6: Works with custom _complete_ and _aggregation_ values
- **Source:** Task spec lines 140-144
- **Evidence:** QA report AC-6 section (MET), Code review AC-6 (MET)
- **Tests:** `TestTaskService_CreateTask_CustomAggregationStatus`, `TestFeatureService_CreateFeature_CustomAggregationStatus`

### AC-7: Graceful degradation on parent update failure
- **Source:** Task spec lines 146-152
- **Evidence:** QA report AC-7 section (MET), Code review AC-7 (MET)
- **Tests:** `TestTaskService_CreateTask_ReopenFailureDoesNotFailCreate`, `TestFeatureService_CreateFeature_ReopenFailureDoesNotFailCreate`

### AC-8: Both creation paths handled for tasks
- **Source:** Task spec lines 154-160
- **Evidence:** QA report AC-8 section (MET), Code review AC-8 (PARTIALLY MET)
- **Code:** `maybeReopenParentFeature` called at both line 364 and line 428
- **Gap:** No dedicated `FallbackPath` named test exists (code review Observation #2). However, all existing tests exercise the fallback path because `creatorSvc` is nil in test setup.

## Additional Evidence

### Quality Gates
- `make fmt`: PASS
- `make lint`: PASS (0 issues)
- `make test`: PASS (all 31 packages)
- 13 new tests, all passing

### E2E Reachability (from QA report)
- `GetAggregationStatuses()` called from `task_service.go:1287`, `feature_service.go:882`
- `maybeReopenParentFeature()` called from CLI entry points via `task.go:217`, `idea.go:594`
- `maybeReopenParentEpic()` called from CLI entry points via `feature.go:276`, `idea.go:574`
- `FeatureEpicLookup.Update()` called from `feature_service.go:885`, wired via `service_accessors.go:186`

### Non-Blocking Observations (from code review)
1. Test for AC-2 uses `archived` instead of `cancelled`
2. No dedicated `FallbackPath` test (coverage exists through other tests)
3. Unused `taskKey`/`featureKey` parameters in method bodies (reserved for future audit logging)
4. No audit/history records for auto-reopen (pre-existing infrastructure gap)

## Conflicts Flagged for Assessor

1. **AC-1 and AC-3 history requirement vs. implementation:** The acceptance criteria state "a history record is created... with notes containing auto-reopened". The implementation does NOT create history records because no feature/epic history table exists. Both code review and QA acknowledge this as a pre-existing infrastructure gap, not an implementation defect.

2. **AC-2 test specificity:** AC-2 specifies `cancelled` status but the test uses `archived`. The code review rates this as PARTIALLY MET. The QA report rates it as MET with a note.
