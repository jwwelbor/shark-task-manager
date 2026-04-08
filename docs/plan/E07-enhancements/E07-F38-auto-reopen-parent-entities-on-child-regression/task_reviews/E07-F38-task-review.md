---
feature_key: E07-F38
review_type: task_decomposition
verdict: PASS
reviewed: 2026-04-06
---

# E07-F38 Task Decomposition Review

## Verdict: PASS

All 7 generated tasks fully cover the spec requirements and acceptance criteria. Ordering and dependencies form a valid DAG. Task quality meets project conventions (each file ≤44 lines, no code blocks, references spec.md and test-plan.md).

---

## Requirements Coverage Matrix

### Functional Requirements

| Requirement | Description | Covered By |
|---|---|---|
| REQ-F-001 | Task backward transition reopens feature + epic | T-002 (impl), T-003 (post-hook wiring), T-006 (tests) |
| REQ-F-002 | Feature backward transition reopens epic only | T-002 (impl), T-003 (post-hook wiring), T-006 (tests) |
| REQ-F-003 | Refactor existing maybeReopen helpers onto unified cascade | T-003 (refactor), T-007 (regression tests) |
| REQ-F-004 | Three-step reopen target lookup (history → aggregation → initial) | T-002 (resolveReopenTarget), T-006 (AC-07/08 tests) |
| REQ-F-005 | No hardcoded status names; profile-agnostic | T-001 (parameterized NOT IN), T-002 (uses workflow.Service.IsTerminalStatus) |
| REQ-F-006 | Bugs/change-cards do not trigger cascade | Structural (no hook in BugService/ChangeCardService); T-007 AC-11 verifies |
| REQ-F-007 | History row notes prefix `auto_reopen:` | T-002 (buildAutoReopenNotes), T-006 (AC-06 test) |
| REQ-F-008 | Idempotent (re-fetch inside Tx, skip if non-terminal) | T-002 (impl), T-006 (AC-04/05/09 tests) |
| REQ-F-009 | shark status history formatter labels auto-reopen rows | T-005 (impl), T-007 (AC-14 test) |
| REQ-F-010 | Rollup commands reflect reopened state | Implicit via existing rollup paths; AC-12 integration tests in T-006/T-007 |

### Non-Functional Requirements

| Requirement | Description | Covered By |
|---|---|---|
| REQ-N-001 | ≤50ms P95 cascade overhead | T-006 (BenchmarkCascade_BothLegs, AC-15) |
| REQ-N-002 | Atomic across parent chain | T-001 (UpdateStatusTx, CreateTx), T-002 (single Tx envelope) |
| REQ-N-003 | Cascade failure is non-blocking, slog.Warn | T-002 (AC-T3), T-006 (AC-13 test) |
| REQ-N-004 | Backward compatibility (no CLI/API/exit-code change) | Structural; nil-safe optional deps in T-003 |
| REQ-N-005 | No schema change unless index missing | T-001 (AC-T4 conditional migration with version bump) |
| REQ-N-006 | ≥80% line coverage; explicit error-path tests | T-006 (cascade tests), T-007 (repository tests) |
| REQ-N-007 | make fmt/lint/test pass | T-007 (AC-T5 / AC-16) |

### Acceptance Criteria

| AC | Description | Owning Task | Test Function |
|---|---|---|---|
| AC-01 | Task backward → feature reopen to prior non-terminal | T-006 | TestCascade_TaskBackwardReopensFeature |
| AC-02 | Task backward → epic reopen | T-006 | TestCascade_TaskBackwardReopensEpic |
| AC-03 | Feature backward → epic reopen only | T-006 | TestCascade_FeatureBackwardReopensEpic |
| AC-04 | Non-terminal feature skipped, epic still checked | T-006 | TestCascade_NonTerminalFeatureContinuesToEpic |
| AC-05 | All ancestors non-terminal → no-op | T-006 | TestCascade_AllAncestorsNonTerminalNoOp |
| AC-06 | History row format `auto_reopen:` prefix | T-006 | TestCascade_HistoryRowFormat |
| AC-07 | Fallback to aggregation status | T-006 | TestResolveReopenTarget_FallbackAggregation |
| AC-08 | Fallback to initial status | T-006 | TestResolveReopenTarget_FallbackInitial |
| AC-09 | Idempotent on second regression | T-006 | TestCascade_IdempotentOnSecondRegression |
| AC-10 | Existing maybeReopen tests still pass | T-007 | Existing test suite (regression) |
| AC-11 | Bug regression does not cascade | T-007 | TestCascade_BugDoesNotTriggerCascade |
| AC-12 | Basic + advanced profile parameterization | T-006 / T-007 | TestCascade_BasicProfile, TestCascade_AdvancedProfile |
| AC-13 | Tx failure non-blocking, WARN log | T-006 | TestCascade_TxFailureIsNonBlocking |
| AC-14 | Status history formatter renders auto label | T-007 | TestStatusHistoryFormatter_AutoReopenLabel |
| AC-15 | ≤50ms P95 benchmark | T-006 | BenchmarkCascade_BothLegs |
| AC-16 | make fmt/lint/test exits 0 | T-007 | CI gate |

**Coverage: 16/16 ACs, 10/10 functional REQs, 7/7 non-functional REQs.**

---

## Task Quality

| Check | Result |
|---|---|
| Each task ≤50 lines (excl. frontmatter) | PASS — all 7 tasks are 28–44 lines |
| No code blocks in task files | PASS |
| References spec.md / test-plan.md sections | PASS — every task references specific spec sections and ACs |
| Task scopes coherent | PASS — each task modifies a related set of files (repo / service / wiring / formatter / tests) |
| Task titles reflect content | PASS |

---

## Ordering & Dependencies

```
T-001 (repo methods) ────┬─→ T-002 (cascade_reopen.go) ─┬─→ T-003 (wire post-hook) ──┬─→ T-004 (CLI/HTTP wiring)
                         │                              │                            │
                         │                              └─→ T-006 (cascade tests) ───┤
                         │                                                           │
                         │                              ┌─→ T-005 (formatter) ───────┤
                         │                              │                            │
                         └──────────────────────────────┴───────────────────────────→ T-007 (repo + integration tests)
```

| Check | Result |
|---|---|
| Foundation tasks first | PASS — T-001 (repo) → T-002 (helper) → T-003 (wiring) |
| DAG validity (no cycles) | PASS |
| Execution order matches dependencies | PASS — declared `dependencies:` field consistent with execution_order |
| Tasks unblocking | PASS — T-006 can run in parallel with T-003/T-004/T-005 once T-002 lands |

---

## Scope Alignment

| Check | Result |
|---|---|
| Stays within feature scope | PASS — no scope creep beyond cascade reopen |
| No tasks unmapped to spec requirements | PASS — every task maps to one or more REQ/AC |
| Granularity appropriate | PASS — 7 tasks, each ~½–1 day of focused work |

---

## Gaps Identified

**None.** All spec requirements and acceptance criteria are covered.

---

## Minor Observations (non-blocking)

1. T-005 (formatter) declares `dependencies: [T-E07-F38-003]` but is functionally independent of T-003 — it could ship in parallel. Not a blocker; the dependency is harmless and may reflect a desire to land cascade behavior before its UI label.
2. AC-12 (basic vs advanced profile parameterization) is referenced by both T-006 and T-007 (`backward_transition_test.go`). The split is acceptable per spec.md Section 2.1.2; either task may host the parameterized table-driven tests. Recommend T-007 for the cross-cutting integration tests and T-006 for the unit-level mocks.
3. REQ-F-010 has no dedicated AC in the spec (the spec explicitly notes it is a regression-prevention guard). No task explicitly targets it, which matches the spec's intent. Consider adding a smoke assertion in T-007 that calls a rollup function after cascade fires.

---

## Recommendations

- **Advance to next status.** Tasks are ready for implementation.
- During T-001, perform the index check called out in REQ-N-005 first; if no index addition is needed, no migration changes are required (preferred path).
- During T-006, ensure the table-driven AC-12 tests use real workflow profile config files (basic + advanced) loaded via `workflow.NewService`, not hand-rolled status maps, to keep the "no hardcoded status names" guarantee end-to-end.
