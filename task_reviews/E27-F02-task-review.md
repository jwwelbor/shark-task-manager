---
feature: E27-F02
title: Viewer API Endpoints - Read-Only Dashboard Data Layer
reviewer: task-review-agent
date: 2026-04-11
verdict: PASS
---

# Task Review: E27-F02 — Viewer API Endpoints - Read-Only Dashboard Data Layer

## Verdict: PASS

All six tasks cover the feature requirements with correct sequencing, declared
dependencies, atomic scoping, and sufficient implementation detail for a
developer agent to execute without ambiguity.

---

## Feature Requirements Coverage

| Requirement | Source | Covered By |
|-------------|--------|-----------|
| `ListRecentAcrossEntities` on EntityHistoryRepository | spec §4.4 / REQ-F-006 | T-001 |
| `ViewerService` with 7 dashboard aggregate methods | spec §1.1 / REQ-F-001 through REQ-F-007 | T-002 |
| `Summary(ctx)` — entity counts + status colors | spec REQ-F-001 | T-002 |
| `Hierarchy(ctx)` — epic→feature tree, task/blocked counts | spec REQ-F-002 | T-002 |
| `History(ctx, key)` — unified audit trail, entity type detection | spec REQ-F-003 | T-002 |
| `File(ctx, key)` — DB-path-first + EvalSymlinks containment | spec REQ-F-004 / ADR-E27-008 | T-002 |
| `FeatureTasks(ctx, key, opts)` — filtered, paginated task list | spec REQ-F-005 | T-002 |
| `RecentActivity(ctx, opts)` — cross-entity recent changes | spec REQ-F-006 | T-002 |
| `WorkflowMeta(ctx)` — full workflow definition with direction | spec REQ-F-007 | T-002 |
| `ViewerServicer` interface + response DTOs | spec §1.1 | T-003 |
| `withLocalCORS` localhost-only middleware | spec REQ-NF-010 / ADR-E27-007 | T-003 |
| `ViewerHandler` with 7 endpoint methods + `RegisterRoutes` | spec §3 | T-003 |
| Input validation — key regexes, query param allowlists | spec REQ-NF-013 | T-003 |
| Error mapping (SecurityError→403, FileTooLargeError→413, etc.) | spec REQ-F-004 | T-003 |
| `ServiceContainer.ViewerService` + `WireServices` wiring | spec §1.1 / integration | T-004 |
| Routes mounted at `/api/v1/viewer/` | spec §3 | T-004 |
| ViewerService unit tests TC-S-001 through TC-S-063 | test-plan §4 | T-005 |
| Handler tests TC-H-001 through TC-H-074 | test-plan §5.2 | T-006 |
| Repository tests TC-R-001 through TC-R-007 | test-plan §5.1 | T-006 |

---

## Task-by-Task Assessment

### T-E27-F02-001 — Add ListRecentAcrossEntities to EntityHistoryRepository

**Status:** PASS

- Scope is atomic and correctly bounded (one new method + two new types).
- Functional requirements enumerate UNION ALL query structure, INNER JOIN for
  orphan omission, DESC ordering, and filter parameters — all traceable to
  spec REQ-F-006 and ADR requirements.
- Limit=0 / negative safety is explicitly called out.
- Parameterized queries are mandated (cross-reference to input-sanitization.md).
- Dependency on E27-F01 (DB handle pattern) is declared.
- No ambiguous deliverables.

### T-E27-F02-002 — Implement ViewerService

**Status:** PASS

- All 7 service methods are specified with exact signatures, return types, and
  edge-case behavior (fallback color/phase, limit clamping, orphan omission,
  path containment via EvalSymlinks, io.LimitReader at 2 MiB).
- Error types (`SecurityError`, `FileTooLargeError`) are called out for
  definition.
- Read-only invariant is explicit (no transactions, no mutations).
- `direction` computation algorithm (ordinal comparison) is unambiguous.
- Dependency on T-001 declared; coordination with T-003 for type declarations
  is noted.
- Note: the task correctly acknowledges that response types may be declared
  temporarily in viewer_service.go and moved to types.go during T-003 — this
  is a sensible coordination instruction for potentially parallel execution.

### T-E27-F02-003 — Implement viewer API handler, types, CORS middleware, routes

**Status:** PASS

- Four deliverable files are enumerated with distinct responsibilities.
- CORS middleware spec covers all required cases: localhost/127.0.0.1 echoing,
  OPTIONS preflight 204 short-circuit, non-local origin passthrough, no-origin
  passthrough.
- Input validation requirements cover all param types: key regexes, blocked
  allowlist, entity_type allowlist, since RFC3339 parsing, integer limit/offset
  with clamping.
- Error HTTP status mappings are complete.
- Structured logging requirements (slog.Error, no SQL leakage) included.
- References existing `internal/api/` patterns for handler structure.
- Dependency on T-002 declared; note about compile-time satisfaction of
  ViewerServicer interface is accurate.

### T-E27-F02-004 — Wire ViewerService into ServiceContainer

**Status:** PASS

- Scope is minimal and correct: only two files modified (services.go + route
  registration).
- Checklist covers all wiring concerns: ServiceContainer field, WireServices
  construction, dependency availability audit, projectRoot passing,
  RegisterRoutes call, CORS scoping.
- Explicitly calls out that statusCalc may need construction if not already
  present — prevents a common wiring oversight.
- Acceptance criteria includes a curl smoke test at `/api/v1/viewer/summary`.
- Dependencies on T-002 and T-003 are declared.
- No dedicated tests required (integration covered by T-006 smoke test) is a
  reasonable engineering decision given the narrow wiring scope.

### T-E27-F02-005 — ViewerService unit tests (TC-S-001 through TC-S-063)

**Status:** PASS

- Full mock-only test coverage for all 7 service methods.
- Mock structs follow the function-field pattern per services/testing.md.
- File system tests use t.TempDir() — no hard-coded paths.
- TC-S-032 (symlink escape) conditionally skips on unsupported OS.
- TC-S-034 (2 MiB+1 byte) boundary test is explicit.
- TC-S-061 direction computation test uses controlled ordinal data.
- Coverage target ≥ 85% stated for viewer_service.go.
- Dependency on T-002 only (no DB needed).

### T-E27-F02-006 — Handler tests + repository tests

**Status:** PASS

- Splits cleanly into two test files with appropriate test strategies:
  handler_test.go (httptest + MockViewerServicer, no DB) and
  repository_test.go extension (real test DB with cleanup).
- All TC-H-* groups cover happy paths, input validation errors (400), not-found
  (404), and CORS middleware in isolation.
- CORS OPTIONS test correctly checks that downstream handler body was NOT
  written (TC-H-073).
- Repository TC-R-004 (orphan omission) uses direct DELETE to trigger INNER
  JOIN behavior — correct approach.
- TC-R-003 (since filter) uses controlled time offsets — avoids wall-clock
  flakiness.
- Coverage targets stated: ≥85% for handler.go/cors.go, 100% branch for
  ListRecentAcrossEntities.
- Dependencies on T-001 and T-003 declared.

---

## Execution Order Verification

```
T-001 (repository method)
  └─ T-002 (service, depends on T-001)
       ├─ T-003 (handler package, depends on T-002)
       │    └─ T-004 (wiring, depends on T-002 + T-003)
       └─ T-005 (service tests, depends on T-002)
T-001 + T-003 ──► T-006 (handler + repo tests)
```

Execution orders in shark (1 through 6) match this dependency graph. No
circularity. T-005 and T-006 can proceed in parallel once T-002/T-003 exist.

---

## Security Requirements Coverage

| Security Requirement | Task |
|----------------------|------|
| CORS localhost-only (ADR-E27-007) | T-003 (impl) + T-006 (tests TC-H-070–074) |
| File path containment + EvalSymlinks (ADR-E27-008) | T-002 (impl) + T-005 (TC-S-031, TC-S-032) |
| No user input in file path resolution | T-002 (DB-path-first spec) |
| Key regex validation before DB lookup (REQ-NF-013) | T-003 (impl) + T-006 (TC-H-020-024) |
| Read-only invariant (ADR-E27-003) | T-002 (explicit), T-005 (no mutation mocks) |
| No SQL/FS detail in user-facing error messages (REQ-NF-021) | T-003 (slog.Error requirement) |

---

## Issues Found

None. All six tasks are well-specified, correctly sequenced, and collectively
cover all 7 endpoints, all functional requirements, all non-functional security
requirements, and both unit and integration test layers.

---

## Summary

The task set for E27-F02 is complete and ready for implementation. The tasks
decompose the feature cleanly: repository extension (T-001), service
implementation (T-002), handler/types/CORS package (T-003), server wiring
(T-004), service unit tests (T-005), and handler + repository tests (T-006).
All seven required endpoints are accounted for across T-002 and T-003. Security
requirements (CORS, path containment) are in-scope and tested. Execution order
is correct.
