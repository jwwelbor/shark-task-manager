# Task Review: E27-F05 — Server Wiring and Integration - End-to-End Assembly

**Date**: 2026-04-11
**Reviewer**: Task Review Agent
**Feature Status at Review**: in_task_review
**Verdict**: PASS

---

## Feature Summary

E27-F05 is the integration assembly feature for the E27 shark status viewer epic. It modifies `cmd/server/main.go` and `cmd/server/services.go`, adds a CORS middleware, and introduces a new `internal/viewer/server` package with an exported `StartServer()` entry point callable by the `shark web` CLI command (F04).

4 tasks were generated. All are reviewed below.

---

## Dependency Chain Validation

| Prerequisite Feature | Status at Review | Required By |
|---|---|---|
| E27-F01 (dbinit package) | active (in development) | T-001, T-004 |
| E27-F02 (ViewerService + handler) | active (in development) | T-002, T-004 |
| E27-F03 (SPA assets) | in_task_generation | T-002, T-004 |
| E27-F04 (shark web CLI) | active (in development) | T-004 (interface contract) |

All prerequisites are in-progress or have tasks in generation. F05 tasks are correctly gated behind completion of F01-F04 before final integration. F04 depends on F05 (StartServer), so the feature-level ordering is F01→F02→F03→F05 (with F04 consuming F05's exported function). This is consistent with the stated dependency chain.

---

## Task-by-Task Review

### T-E27-F05-001 — Replace hardcoded DB init in cmd/server/main.go with dbinit package

**Execution Order**: 1
**Verdict**: PASS

**Coverage**: Directly addresses feature spec requirement: "call `dbinit.Init` (from E27-F01) instead of the hardcoded `db.InitDB("shark-tasks.db")`". Scope is minimal and surgical — one import, one call swap. Preservation requirements for integrity-check logging and `defer db.Close()` are explicitly stated. Acceptance criteria are testable and concrete (TC-001, TC-002, build/test gates). Dependencies correctly identified (F01 must be merged first). This task is correctly sequenced first since T-004 explicitly depends on its DB init logic.

**Issues**: None.

---

### T-E27-F05-002 — Wire ViewerService into ServiceContainer and register viewer routes with SPA and CORS

**Execution Order**: 2
**Verdict**: PASS with note on execution ordering

**Coverage**: Addresses the three main integration points from the feature spec: (1) `ViewerService` field added to `ServiceContainer`, (2) viewer routes registered on the mux, (3) SPA served at `GET /`. Requirements include all 7 viewer endpoints, CORS scoping to viewer routes only (not CRUD), and the sub-mux pattern. Acceptance criteria map cleanly to test cases (TC-003, TC-004, TC-005, TC-011c, TC-015). Dependencies on F02 and F03 correctly noted.

**Ordering note**: T-002 (execution order 2) explicitly depends on T-003 (WithLocalCORS, execution order 3) being complete "before final integration." T-003 has zero dependencies and is independently implementable. The execution order as specified (002 before 003) creates a dependency inversion — a developer implementing strictly in order would attempt T-002 before the CORS middleware it requires exists. This is a documentation issue, not a blocking defect: developers can implement T-003 first (it is independent), then complete T-002. The task notes in T-002 correctly flag this: "T-E27-F05-003 (WithLocalCORS) should be complete before final integration." Recommend developer treats T-003 as a prerequisite of T-002 regardless of listed order numbers.

**Issues**: Non-blocking. Execution order numbers 2 and 3 are inverted relative to dependency direction. Developer should implement T-003 before T-002.

---

### T-E27-F05-003 — Implement WithLocalCORS middleware in internal/api/viewer/cors.go

**Execution Order**: 3
**Verdict**: PASS

**Coverage**: Complete, self-contained, zero-dependency task. All CORS behaviors are enumerated (localhost/127.0.0.1 origin echoing, OPTIONS preflight short-circuit returning 204, Vary header, no external origins). Test cases TC-006 through TC-011b are explicitly listed with concrete httptest assertions. Stdlib-only constraint correctly specified. Package placement (`internal/api/viewer`) is consistent with where the viewer handler lives. This task is independently implementable and could be done in any order relative to T-001 and T-002.

**Issues**: None. (See ordering note in T-002 review.)

---

### T-E27-F05-004 — Implement StartServer entry point in internal/viewer/server/server.go

**Execution Order**: 4
**Verdict**: PASS

**Coverage**: This is the keystone task. The `Options` struct and `StartServer` signature are fully specified with types. All behavioral requirements are enumerated: DB init path when `opts.DB` is nil, caller-owned DB does not get closed, ready channel semantics, graceful 30-second shutdown, `http.ErrServerClosed` treated as nil, `otelhttp` wrapper. The "thin cmd/server/main.go" constraint (~15 lines) ensures the refactor is complete. Test cases TC-012, TC-013, TC-014b, TC-018 (500ms timing) cover the four critical behaviors. The no-circular-import constraint (`internal/viewer/server` must not import `internal/cli`) is explicit and verified by TC-014 build check. The design note about `WireServices` migration (move to this package or call via hook) is correctly deferred to implementation as an allowed choice.

**Issues**: None.

---

## Specification Coverage Check

| Feature Spec Requirement | Covered By Task | Status |
|---|---|---|
| Replace `db.InitDB` with `dbinit.Init` | T-001 | COVERED |
| `ViewerService` added to `ServiceContainer` | T-002 | COVERED |
| `WireServices()` constructs `ViewerService` | T-002 | COVERED |
| 7 viewer routes registered on mux | T-002 | COVERED |
| CORS only on `/api/v1/viewer/*` | T-002 + T-003 | COVERED |
| `GET /` serves embedded `viewer.html` | T-002 | COVERED |
| `internal/viewer/assets.go` with go:embed | T-002 (consumes F03 asset) | COVERED (F03 produces the asset) |
| `StartServer(addr, db)` exportable entry point | T-004 | COVERED |
| `cmd/server/main.go` calls `StartServer` | T-004 | COVERED |
| Integration smoke tests (GET /, /api/v1/viewer/summary) | T-002 + T-004 (cmd/server/main_test.go) | COVERED |
| No circular import from `internal/viewer/server` | T-004 | COVERED |

All feature spec requirements have task coverage.

---

## Execution Order Assessment

Stated order: T-001 → T-002 → T-003 → T-004

Correct dependency order: T-001 and T-003 are independent of each other; T-003 must precede T-002; T-004 depends on all three.

Suggested implementation order: **T-001 → T-003 → T-002 → T-004**

The listed execution orders do not prevent successful implementation — T-003 can simply be done before T-002 regardless of the ordinal numbers. The developer instructions on T-002 already note T-003 as a prerequisite, mitigating the risk.

---

## Risk Assessment

| Risk | Severity | Mitigation |
|---|---|---|
| T-002/T-003 order inversion | Low | T-002 notes T-003 as prerequisite; no functional blocker |
| F03 still in task generation (SPA not ready) | Medium | F05 tasks are correctly deferred — T-002 and T-004 list F03 as a dependency |
| WireServices migration decision (move vs. hook) | Low | T-004 explicitly allows either approach; developer can choose |
| Circular import risk (`internal/viewer/server` → `internal/cli`) | Low | Explicitly called out with build-time verification (TC-014) |

---

## Conclusion

All 4 tasks are complete, correctly scoped, and provide sufficient specification for a developer to implement without ambiguity. Acceptance criteria are testable. The one structural concern (T-002/T-003 execution order inversion) is non-blocking and is already self-documented in T-002's dependency notes. All feature spec requirements are covered.

**Verdict: PASS** — Feature is ready to advance to active.
