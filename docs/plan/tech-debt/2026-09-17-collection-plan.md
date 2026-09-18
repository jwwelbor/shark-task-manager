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

> **Status: executed and closed, 2026-09-18.**  All seven rows below are
> delivered; see [Outcome](#outcome--2026-09-18).  Two records remain
> non-terminal by design (TD-146/TD-147, blocked upstream) and one (TD-191) is
> parked awaiting an owner decision recorded on the record itself.  Nothing in
> the delivery table is still to do.

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

TD-179 shipping in an unrelated PR shows the lag is not confined to #234–#236.
Before starting each row, re-verify its records against `HEAD` and drop anything
already fixed.

## Second reconciliation — done 2026-09-17

Execution rule 1 was applied to every scheduled record before any branch was
opened.  Twenty-one more were verified fixed in `main` at `d8e29ba` and
force-set to `resolved`, leaving twelve genuinely open at the time this was
written.  All twelve have since been dispatched — see [Outcome](#outcome--2026-09-18).  Three rows (the
original 1, 4, and 5) emptied out entirely and need no branch.

| Tech debt | Evidence at `d8e29ba` |
| --- | --- |
| TD-176 | `gatepersist/coordinator.go:107` and `run_resume.go:424` both bind `releaseErr` and report it. |
| TD-177 | `gatepersist/reconcile.go:125` compares target status **or** digest; `kickback.go:176` records the round-2 UAT fix that folded reason into the digest. |
| TD-178 | `run_apply_result.go:94` and `impact.go:74` both read through `gaterun.ReadBoundedRegularFile(path, workercontrol.MaxEnvelopeBytes)`. |
| TD-181 | `workercontrol/question.go:36` bounds every `Options` element and `:49` bounds `recommendation` alongside `question`/`why_blocking`. |
| TD-183 | `gatepersist/adapters.go:150-153` passes `SessionID`, `FromStatus`, and `GuardAdvance: true`. |
| TD-184 | `run_apply_result.go:117` calls `verifyClaimSession` before any note/kickback/transition mutation. |
| TD-185 | Both `change/{code_review,qa}.golden` end `.\n` — no trailing blank line. |
| TD-187 | `gaterun/fsio_nofollow_windows.go:48-70` uses `windows.CreateFile` with `FILE_FLAG_OPEN_REPARSE_POINT` and a handle-attribute reparse check; no `Lstat`-then-open. |
| TD-189 | `run_resume.go:358` defines `finishGateIngest`; both ingest paths call it. |
| TD-190 | The `buildGateCoordinator` comment no longer mentions a placeholder or T-E34-F05-005. |
| TD-192 | `entity_service.go:451,455,468` normalize both sides before comparing. |
| TD-193 | `runner/dispatcher.go:198,201` use `readBoundedStream(..., MaxEnvelopeBytes)`; `:263` applies `io.LimitReader`. |
| TD-198 | `E34-interaction-map.md:45-49,64-65,77` carry the corrected I-02/I-03 pointers and hand the missing parity test to TD-211. |
| TD-202 | `impact.go:131` unmarshals into a typed `gateresult.ChangeImpactSet` and reconciles field-by-field; the `map[string]interface{}` merge is gone. |
| TD-207 | `integration/history.go:180` calls `ValidateEpicRunID(epicRunID)` first. |
| TD-209 | `sharkdata/overrides_status.go:177` excludes the installer's `overrides/.gitkeep`. |
| TD-210 | `run_cascade_integration_test.go` carries no hardcoded status list. |
| TD-217 | `epic/integration_review.md` step 5 names the candidate's `tracked_path_digests`/`untracked_path_digests` **in addition to** event-level paths. |
| TD-218 | `TestCascadeIntegrationGuard_NonEpicDoesNotCaptureBase` drives the real `cascadeIntegrationGuard` with a feature key. |
| TD-219 | `candidate_test.go:736,739` assert exact cardinality; `TestComputeDirtyPathDigests_RenameDeleteAndCleanTree` covers rename, delete, and clean-tree. |
| TD-220 | `table_parser_test.go:232-262` blanks/misvalues `Same-model gate`, `Separate QA`, and `Final UAT` independently. |

## Delivery order (all rows delivered — see Outcome)

Rows are renumbered after the second reconciliation.  Every remaining row is
independent of every other — the `run_cascade_integration_test.go` collision
that forced the original row-6-after-2/4/5 ordering is gone with TD-210 and
TD-218.

| Order | Branch | Tech debt | Scope and PR boundary |
| --- | --- | --- | --- |
| 1 | `tech-debt/advance-guard-idempotency` | TD-191 | The `TransitionStatus` idempotency early-return fires before `enforceAdvanceGuard`/`recordAdvanceGuard`, so a guarded transition landing on an already-at-target entity succeeds without touching the CAS replay ledger. |
| 2 | `tech-debt/run-lock-ownership` | TD-186 | `RunLock.Release` has no ownership token, so a late `Release` from a handle whose lock was reclaimed can unlink a different holder's lock file. |
| 3 | `tech-debt/gate-test-coverage` | TD-165, TD-188, TD-231 | Tests only, no behavior change: execution-level harness-wiring and `GateIngest` companions for the two source-scan guards, and consolidator failure diagnostics that stop dumping ~9KB into the failure message. |
| 4 | `tech-debt/docs-and-prompt-accuracy` | TD-180, TD-230, TD-232, TD-233 | Documentation and prompt text only: the `epic.integration_review` adoption-matrix note, a runnable coordinator-side `gate_result_v1` apply-result example with `run_id` semantics, the user-global finish-feature skill's triage decision tree, and a single shared triage-routing contract reference. |
| 5 | `tech-debt/question-api-error-surface` | TD-234 | Standalone: the question owner-mismatch error leaks CLI flag syntax into HTTP API responses.  Keep separate — it is the only row touching the API error surface. |
| 6 | `tech-debt/workflow-progress-guard` | TD-153 | Standalone: `ExcludeFromProgress` fails open for custom workflows, plus the duplicated `DependsOn` guard in `sprint_service.go`. |
| 7 | `tech-debt/i02-i03-recurrence-field` | TD-199 | Decision first, code second.  Whether I-02/I-03 get an explicit recurrence-classification field is an epic-owner call weighed against updating every consumer (E34-F07, E34-F08).  Record the decision; close as accepted-as-is if the churn is not worth it, and only then open a branch. |

## Outcome — 2026-09-18

Every row was executed through `/shark-rider run`.  Six pull requests merged.
A row is recorded closed here only when its PR merged **and** its tracker
records were advanced, per execution rule 5.

| Row | PR | Outcome |
| --- | --- | --- |
| 1 — TD-191 | [#266](https://github.com/jwwelbor/shark-task-manager/pull/266) (docs only) | **Not implemented — owner decision required.**  The defect is confirmed at `entity_service.go:273-282`, and research found a path the record lacked: `status_group.go:490` sets `GuardAdvance = true` for *every* `shark status advance`, which makes `configuration.md`'s documented "already consumed" rejection unreachable.  But TD-191's Resolution Notes carry an explicit architect instruction not to move guard enforcement ahead of the idempotency return.  Both positions and three options are recorded on the record.  Stays `identified`. |
| 2 — TD-186 | [#267](https://github.com/jwwelbor/shark-task-manager/pull/267) | **Resolved.**  `Release` now fences its unlink on a nonce written at acquire.  Inode/device fencing — the research report's own recommendation — was built first and rejected on evidence: ext4 reuses the freed inode on remove-then-create, so it gave zero protection.  The comment states plainly that this narrows rather than closes the race. |
| 3 — TD-165, TD-188, TD-231 | [#265](https://github.com/jwwelbor/shark-task-manager/pull/265) | **TD-231 resolved** (fixed all three affected tests, not the one named).  **TD-165 and TD-188 `wont_fix`**: both asked for work decision D7 already settled — `run_test.go:633-639` records that `test-plan.md` rejected seam-injecting `buildTransitioner`/`cli.Get*Service` and that D7 *sanctions* source-invariant checks for this property.  A candidate TD-165 test was withdrawn after review proved its mutation-detection delta over the existing suite was zero. |
| 4 — TD-180, TD-230, TD-232, TD-233 | [#268](https://github.com/jwwelbor/shark-task-manager/pull/268) | **All resolved.**  TD-232 was completed separately before this row ran.  TD-180's premise was stale in the opposite direction — the matrix claimed `integration_review` opts into `gate_result_v1` when the shipped step is `legacy`.  TD-233 deviated from its own plan: a single cross-tree reference is not resolvable at runtime, so one canonical statement per deployment tree. |
| 5 — TD-234 | [#269](https://github.com/jwwelbor/shark-task-manager/pull/269) | **Resolved.**  Typed `QuestionOwnerMismatchError` keeps the service layer transport-neutral; the CLI re-adds its flag hint via `errors.As`.  B062's assertions preserved, not weakened. |
| 6 — TD-153 | [#270](https://github.com/jwwelbor/shark-task-manager/pull/270) | **Resolved.**  Closed by a validator rule rather than a runtime selector, leaving all five progress consumers untouched.  The rule immediately caught a live shipped gap: `question.yaml` was crediting `withdrawn`/`superseded`/`archived` as complete progress. |
| 7 — TD-199 | — (no branch) | **`wont_fix`, accepted as-is.**  The architect decision already existed on the record (2026-09-15).  Re-verified that E34-F08 shipping — the named hypothetical trigger — did not create cross-feature demand: no `recurrence` field in `internal/gateresult`, no consumer queries one.  Row 7 directs closing without a branch in exactly this case. |

### Defects found that no record had filed

Re-verifying before implementing, rather than implementing from the record text,
surfaced four things the plan did not contain:

- `run.go`'s `--resume-run` flag help stated the per-stage `run_id` as
  `<run_id>-g<stage-iteration>`; `gateStageRunID` builds
  `<invocation run_id>-<entity-key>-g<stage-iteration>`.  Anyone following the
  help text would construct an id that does not match the sidecar.  Fixed in #268.
- `question.yaml`'s three abandonment terminals carried `progress_weight: 1.0`
  with no `exclude_from_progress`, so withdrawn and superseded Questions counted
  as fully-complete progress.  Fixed in #270.
- Inode/device fencing is a no-op on ext4 for the remove-then-create case, so the
  approach TD-186's research recommended would have shipped a fix that fixed
  nothing.  Avoided in #267.
- A new test with zero mutation-detection delta over the existing suite was
  withdrawn rather than shipped.  Avoided in #265.

## Superseded delivery order (pre-reconciliation, kept for audit)

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
| TD-191 | Awaiting an owner decision.  Its Resolution Notes carry an architect instruction not to move advance-guard enforcement ahead of the idempotency return; the 2026-09-18 research reaches the opposite conclusion and proposes a narrower in-path guard.  Three options are recorded on the record and in PR #266.  Latent here — `advance_guard` is not enabled in this repo.  Do not implement without that decision. |
| TD-146, TD-147 | E40-F10 needs a stable per-gate join from upstream I-08 for elapsed/cost measurements and comparison deltas.  Both stay `triaged` until the upstream contract is extended or the F10 contract is renegotiated.  Do not open a branch. |

## Execution rules

1. Re-verify each row's records against `HEAD` before starting that row — the
   Reconciliation table shows the tracker has drifted from `main` before.
2. Work only from an isolated branch/worktree based on current `origin/main`.
3. After the second reconciliation every remaining row is independent and may
   run in any order or in parallel.  The pre-reconciliation ordering
   constraints are recorded in the superseded table above.
4. Run the relevant focused test while developing, then `make fmt`, `make lint`,
   and `make test` before a PR is ready.
5. Run the full review/PR/CI/merge lifecycle through `/finish-feature` for each
   ready branch.  A row is closed when its PR is merged **and** its tracker
   records are advanced — not when either one alone is done.
