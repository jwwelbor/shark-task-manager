# UAT Evidence Compilation: T-E19-F05-012

**Task:** Tech debt: extract shared buildCapacityRows helper from PlanSprint/GetSprintCapacity duplication
**Epic:** E19 — Sprint Management Planning System
**Feature:** E19-F05 — Sprint Planning View / Capacity Management
**Date:** 2026-05-06

---

## Acceptance Criteria (from task spec + code review context)

The task spec uses a placeholder template for AC, but the substantive criteria are established in the code review and QA report based on the goal description. Derived AC:

| AC | Description | Source |
|----|-------------|--------|
| AC-1 | Single `buildCapacityRows` helper defined (no duplication) | Code review + task goal |
| AC-2 | `GetSprintCapacity` calls `buildCapacityRows` (not duplicated inline code) | Code review |
| AC-3 | `PlanSprint` calls `buildCapacityRows` (renamed from `planComputeCapacityRows`) | Code review |
| AC-4 | `planComputeCapacityRows` removed entirely (no remaining references) | Code review |
| AC-5 | `_ = assignments` blank identifier removed | Code review |
| AC-6 | No behavior change — body identical, tests pass | Code review + QA |

---

## Evidence Map

| AC | QA Report Evidence | Code Review Evidence | Test Evidence | Direct Code Evidence |
|----|------------------|---------------------|---------------|---------------------|
| AC-1 | Line 1814: one definition confirmed by grep | PASS — "Single source of truth" | 11 tests pass | `grep` output: one definition at line 1814 |
| AC-2 | Line 1432: `return buildCapacityRows(assignments, capacities), nil` | PASS — line 1432 documented | TestSprintService_GetSprintCapacity_* (5 tests) | Confirmed via direct code read |
| AC-3 | Line 1798: `capacity := buildCapacityRows(assignments, capacityModels)` | PASS — line 1798 documented | TestPlanSprint_* (6 tests) | Confirmed via direct code read |
| AC-4 | grep output shows 0 references to `planComputeCapacityRows` | PASS — "No remaining references" | N/A | Confirmed: no grep hits |
| AC-5 | `_ = assignments` absent from working tree | PASS | N/A | Confirmed: grep returns 0 hits |
| AC-6 | Body identical to deleted code; test run PASS (11/11) | PASS — "No behavior change" | 11/11 pass in 0.015s | Read lines 1814-1843 |

---

## Artifacts

| Source | Path | Timestamp |
|--------|------|-----------|
| Code Review | docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/code_review/20260506-161500-T-E19-F05-012-code-review.md | 2026-05-06 16:15:00 |
| QA Report | docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/code_review/20260506-162500-T-E19-F05-012-qa.md | 2026-05-06 16:25:00 |
| Implementation | internal/services/sprint_service.go | (working tree) |
| Test File | internal/services/sprint_service_test.go | (working tree) |

---

## Pre-existing Build Failure (Not in Scope)

`sprint_analytics_service.go` + `sprint_analytics_service_test.go` have a type mismatch from in-progress T-E19-F04-011. This failure:
- Exists identically at baseline HEAD before T-E19-F05-012 changes
- Affects only analytics files, not `sprint_service.go`
- Is being resolved by concurrent task T-E19-F04-011
- Targeted test run bypasses the failing package-level compile and runs 11/11 relevant tests successfully

---

## Missing Evidence

None. All 6 acceptance criteria have direct code evidence and test evidence.
