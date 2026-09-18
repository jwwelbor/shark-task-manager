# Tech-debt delivery plan — 2026-09-17

Supersedes `2026-09-14-collection-plan.md`.

## Purpose

The tracker holds 35 non-terminal tech-debt records.  Two are blocked upstream;
the remaining 33 are grouped below by defect class and verification surface
rather than by key order.

Each row is one branch and one pull request.  A PR must not pull work from
another row without first updating this plan — the independent review and
rollback boundaries are intentional.

Delivered and `wont_fix` records are not tracked here.  Use the tracker for that
history.

## Reconciliation — done 2026-09-17

These nine were verified fixed in `main` but still sat at `identified`.  All
were force-set to `resolved` on 2026-09-17, each with a reason naming its PR.
No branch is needed; the table is kept for audit.

| Tech debt | Shipped in | Evidence |
| --- | --- | --- |
| TD-194, TD-195, TD-196 | PR #235 | `internal/keys/duplicate.go` now holds the shared custom-key duplicate check, wired into the epic/feature/task services with negative-path tests. |
| TD-197 | PR #236 | `VerifyRunIdentityOwner` no longer exists in `internal/gaterun/identity.go`; the stale comment went with it. |
| TD-200 | PR #234 | `internal/taskcreation/creator.go` + `internal/repository/entityrel` persist `--depends-on` with CLI-level regression coverage. |
| TD-201 | PR #236 | `internal/templates/includes_test.go:669` now reads "all seven required table columns". |
| TD-204 | PR #236 | `TestImpactServiceRecord_PropagatesWriterFailure` covers the note-write failure path via `impactNoteWriterStub`. |
| TD-205 | PR #236 | `E34-F07/test-plan.md` and `T-E34-F07-001.md` both updated. |
| TD-179 | PR #232 | `internal/services/impact_service.go` exists; `impact.go:143` calls `services.NewImpactService`. The extraction is done. |

Three records from the same PR ranges are **not** fixed and are scheduled below:
TD-198 (row 7), TD-199 (row 10), TD-202 (row 1).

TD-179 shipping in an unrelated PR shows the lag is not confined to #234–#236.
Before starting each row, re-verify its records against `HEAD` and drop anything
already fixed.

## Delivery order

| Order | Branch | Tech debt | Scope and PR boundary |
| --- | --- | --- | --- |
| 1 | `tech-debt/gate-bounded-input` | TD-178, TD-181, TD-193, TD-202 | One defect class plus one adjacent fidelity fix on the same read path: unbounded reads and captures evaluated before the envelope size bound, across `run_apply_result`/`impact`, `workercontrol/envelope.go`, and `dispatcher.go`; a bounded-text validator for question-kind fields; and `json.RawMessage` for the impact-file merge so byte/type fidelity survives. |
| 2 | `tech-debt/advance-guard-semantics` | TD-177, TD-183, TD-184, TD-191, TD-192 | `gatepersist` and `advance_guard` transition correctness: source-status/session guard on verify, `--session` ownership check ordering, the idempotency early-return that bypasses the CAS ledger, raw-vs-normalized status comparison, and kickback conflict detection ignoring reason mismatch. |
| 3 | `tech-debt/gate-lock-safety` | TD-176, TD-186, TD-187 | Lock lifecycle only: discarded lock-release errors, missing `RunLock.Release` ownership token, and the Windows-fallback TOCTOU symlink window. |
| 4 | `tech-debt/integration-capture-validation` | TD-207, TD-209 | Input validation and classification behavior: format allowlist for `epicRunID` in `AnalyzeHistory`/`replacementRecordPath`, and the fresh-init `overrides/.gitkeep` scaffold misclassified as orphaned. |
| 5 | `tech-debt/gate-runtime-dry` | TD-189, TD-190, TD-210 | Refactor and stale-comment cleanup over the same files rows 1–3 touch, so it lands **after** them: deduplicate the `run_resume.go` ingest/projection tail, correct the `buildGateCoordinator` comment, and drop the hardcoded status list from the F10 guard-test helpers in `run_cascade_integration_test.go`. |
| 6 | `tech-debt/gate-test-coverage` | TD-165, TD-185, TD-188, TD-218, TD-219, TD-220, TD-231 | Tests only, no behavior change: execution-level harness-wiring and `GateIngest` companions, the golden-testdata trailing blank line, the `cascadeIntegrationGuard` non-epic branch against the real guard type, `DirtyPathDigests` cardinality/rename/delete/clean-tree fixtures, the two untested `table_parser_test.go` columns, and consolidator failure diagnostics. |
| 7 | `tech-debt/docs-and-prompt-accuracy` | TD-180, TD-198, TD-217, TD-230, TD-232, TD-233 | Documentation and prompt text only: the `epic.integration_review` deferral note in `architecture.md`, the stale I-02/I-03 contract-test pointers in `E34-interaction-map.md`, the `integration_review.md` step 5 pointer to the populated candidate-level digests, a runnable coordinator-side `gate_result_v1` apply-result example, the user-global finish-feature skill's triage decision tree, and a single shared triage-routing contract reference. |
| 8 | `tech-debt/question-api-error-surface` | TD-234 | Standalone: the question owner-mismatch error leaks CLI flag syntax into HTTP API responses.  Keep separate — it is the only row touching the API error surface. |
| 9 | `tech-debt/workflow-progress-guard` | TD-153 | Standalone: `ExcludeFromProgress` fails open for custom workflows, plus the duplicated `DependsOn` guard in `sprint_service.go`.  Unrelated to every gate row above. |
| 10 | `tech-debt/i02-i03-recurrence-field` | TD-199 | Decision first, code second.  Whether I-02/I-03 get an explicit recurrence-classification field is an epic-owner call weighed against updating every consumer (E34-F07, E34-F08).  Record the decision; close as accepted-as-is if the churn is not worth it, and only then open a branch. |

## Blocked — not scheduled

| Tech debt | Blocker |
| --- | --- |
| TD-146, TD-147 | E40-F10 needs a stable per-gate join from upstream I-08 for elapsed/cost measurements and comparison deltas.  Both stay `triaged` until the upstream contract is extended or the F10 contract is renegotiated.  Do not open a branch. |

## Execution rules

1. Re-verify each row's records against `HEAD` before starting that row — the
   Reconciliation table shows the tracker has drifted from `main` before.
2. Work only from an isolated branch/worktree based on current `origin/main`.
3. Rows 1–4 are independent of each other and may run in parallel.  Row 5 lands
   after rows 1–3.  Row 6 lands after rows 2, 4, and 5 — TD-210 and TD-218 both
   edit `run_cascade_integration_test.go`.  Rows 7–10 are independent of
   everything.
4. Run the relevant focused test while developing, then `make fmt`, `make lint`,
   and `make test` before a PR is ready.
5. Run the full review/PR/CI/merge lifecycle through `/finish-feature` for each
   ready branch.  A row is closed when its PR is merged **and** its tracker
   records are advanced — not when either one alone is done.
