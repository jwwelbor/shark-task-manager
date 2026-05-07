# UAT Evidence Compilation — T-E19-F05-003

**Task:** SprintService: extend struct, interfaces, constructor, and default-capacity on create
**Epic:** E19 — Sprint Management Planning System
**Feature:** E19-F05 — Sprint Planning View / Capacity Management
**Compiled:** 2026-05-06

---

## Evidence Sources

| Source | File | Timestamp |
|--------|------|-----------|
| Task Spec | docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/tasks/T-E19-F05-003.md | 2026-05-06 |
| Code Review | docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/code_review/20260506-001002-T-E19-F05-003-code-review.md | 2026-05-06 |
| QA Report | docs/plan/E19-sprint-management-planning-system/E19-F05-sprint-planning-view-capacity-management/code_review/T-E19-F05-003-qa.md | 2026-05-06 |
| Implementation | internal/services/sprint_service.go | current |
| Tests | internal/services/sprint_service_test.go | current |
| Wiring | internal/cli/services_global.go | current |
| Config | internal/config/config.go | current |

---

## Acceptance Criteria Evidence Mapping

| # | Criterion | Source | Evidence Available | Notes |
|---|-----------|--------|-------------------|-------|
| AC-1 | SprintCapacityRepository and SprintAssignmentQueryRepository interfaces defined in services package (consumer side) | Task spec / Code review §4 | Yes | Both interfaces defined at lines 52–65 of sprint_service.go; consumer-side placement confirmed |
| AC-2 | NewSprintService 5-arg constructor: repo and workflowSvc required (panic on nil), other 3 optional (nil-safe) | Task spec / Code review §5 | Partial | Implementation uses 5+variadic form (6th variadic `db ...*repository.DB`); required panic guards present; optional args nil-safe. Code review calls it "5-arg" but implementation is technically 5+1 variadic. |
| AC-3 | CreateSprint 4-condition nil guard, insert failure non-fatal | Task spec / Code review §6 | Yes | Lines 190–199 of sprint_service.go; 4-condition guard confirmed; `_ =` discard confirmed |
| AC-4 | TC-015-01 and TC-015-02 tests defined and logically correct | Task spec / QA report §4 | Partial | Tests exist and logic is correct per code inspection. HOWEVER: package fails to compile due to unused imports in sprint_service_test.go (lines 13–15: `repository` and `internaltesthelper` imported but not used). Tests cannot execute. |
| AC-5 | Existing callers not broken | Task spec / Code review §5.4 | Yes | GetSprintService() wiring updated; all test callers updated to 5+variadic form with nil optional args |
| AC-6 | GetSprintService() passes sprintRepo for all three interfaces and cfg | Task spec / Code review §7 | Yes | services_global.go line 614: `NewSprintService(sprintRepo, workflowSvc, sprintRepo, sprintRepo, cfg)` |

---

## Known Issues / Conflicts

1. **BLOCKER — Build failure:** `go test ./internal/services/... -run TestSprintService` fails to compile with:
   - `sprint_service_test.go:13:2: "github.com/jwwelbor/shark-task-manager/internal/repository" imported and not used`
   - `sprint_service_test.go:15:2: "github.com/jwwelbor/shark-task-manager/internal/test" imported as internaltesthelper and not used`
   - These are different from the WIP-stub errors QA reported (those were resolved when GetSprintBacklog was implemented), but the package still does not compile.

2. **Discrepancy — Constructor arg count:** Code review and task spec say "5-arg constructor" but the actual signature is `NewSprintService(repo, workflowSvc, assignmentRepo, capacityRepo, cfg, db ...*repository.DB)` — a 5-arg form plus a variadic 6th arg. The variadic `db` is for CloseSprintWithCarryover transaction support. Callers using the 5-arg form still work (variadic is backward compatible). Not a blocker but a spec/impl mismatch.

3. **Non-blocker — Missing INFO log:** CreateSprint default-capacity path silently discards SetCapacity errors without emitting the observability log line. Filed in code review as TD-OBS-001.
