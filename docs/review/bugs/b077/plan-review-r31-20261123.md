# B077 independent repair plan review — revision 31

plan_path: docs/plan/bugs/B077.repair-plan.md
plan_sha256: sha256:26972249df3c6b15255859c58265b3145eec0cfdd0ca787bceb531720a78e402

**Decision: replan.** Revision 31 closes the two revision 30 findings. Two caller-level gaps remain. One allows a failed discovery to be attributed to a model that was never selected. The other allows the exact Codex CLI route to lose its documented default behavior. Both can be resolved in repair planning under the existing decisions.

**Reviewer:** Codex, independently dispatched in the Tech Lead role. **Commit reviewed:** `a35a2ed0c42ed31997c3aca9525a1937c77b479f`. **Code review baseline:** `bbf490c6ec2bd597d6f3e6e4cdd93e68ca54fa50`. The working file and committed blob have the digest above. Git resolves the declared baseline as a commit, as the parent of the first B077 implementation `dd77e079`, and as an ancestor of the reviewed commit.

## Open findings

### B077-R31-01 — An empty discovery can leave a false selected model in the rejection record

C04 requires a successful mapped configuration read with a missing or empty model to yield a preflight `unverified` record. The same row preserves the selected pair for that status. The adapter's normative matrix requires both `selected_provider` and `selected_model` for preflight `unverified`. Its glossary defines the latter as the model explicitly selected. Rider's mapped procedure permits recording a concrete model only after a nonempty discovery. In this case the provider is known and no model was selected. [Coverage row C04](../../../plan/bugs/B077.repair-plan.md#coverage-index) · [Adapter matrix](../../../../skills/shark-rider/context/host-adapter-contract.md#phasestatus-matrix) · [Mapped discovery](../../../../skills/shark-rider/verbs/run.md#b077-default-bug-stage-codex-fallback-mappings)

Consider a parent that correctly refuses the dispatch, records the bounded error and performs zero claims, launches, admissions or advances, but copies the previous model A into the rejection record. The C02 successes and all the new C04 action counts still pass. C05 never receives a worker. The receipt now attributes a selected model to an attempt that selected none. This is distinct from using the stale model to launch.

**Required change.** Define the bounded parent rejection and its required and forbidden field values when discovery establishes no concrete model. Reconcile it with the adapter matrix and the existing case where a concrete model was selected but its observation channel is unavailable. In the warm-cache control, inspect the actual record emitted by Rider. Require the original request, attributable discovery evidence, and absence of any claimed concrete selection or observation. Substituting A only in the rejection metadata must fail. Keep the existing zero-action assertions and the selected-but-unverified positive field control. Decisions 4008 and 4011 supply the authority for truthful selection and bounded rejection. The revision needs no exception to either.

### B077-R31-02 — A Claude default can satisfy C03 while the exact Codex CLI default regresses

C03 specifies the outcome for “a supported default” at a generic worker boundary. C01 points its empty Claude case there. C02's named exact Codex cases have a concrete requested model. None requires the receiving Codex CLI to exercise the evidenced-default branch with an empty keyed request. The current empty-model contract test checks Claude instructions and generic contract text. The Codex recording test uses four nonempty requests. [C01–C04](../../../plan/bugs/B077.repair-plan.md#coverage-index) · [Recorder cases and default test](../../../../tests/contracts/b077_codex_route_attestation_test.go) · [Exact Codex launch](../../../../skills/shark-rider/verbs/run.md#long-running-codex-cli-adapter)

I compared the actual Bash launch excerpts from the delivered revision 27 commit and the reviewed checkout. Each ran against a temporary local recording executable with the same empty `REQUESTED_MODEL`, selected provider and prompt. No provider or account was contacted. The two values for `SELECTED_MODEL` distinguish omitting the override from knowing the concrete default for comparison.

| Launch excerpt | Selected-model input | Exit | Recorder submissions | Recorder received `--model` |
|---|---|---:|---:|---|
| Delivered `c8a36d58` | Empty | 0 | 1 | No |
| Delivered `c8a36d58` | Concrete fixture default | 0 | 1 | No |
| Reviewed checkout | Empty | 1 | 0 | No process received argv |
| Reviewed checkout | Concrete fixture default | 0 | 1 | Yes |

The surrounding procedure still directs Rider to omit the flag for an evidenced default. The caller cannot satisfy that rule in either branch. A Claude-only C03 control and the new mapped-discovery checks can all succeed with this defect present.

**Required change.** Bind the C03 positive control to each affected launch path, including exact Codex CLI. Supply a recording host with a documented, independently observable default. Require one submission of the final keyed bytes, no model flag at the receiving executable, an empty requested model in evidence and the concrete identity from the producing turn. Without default or observation capability, require refusal before claim. Keep C02's explicit pin and C04's invalid mapped discovery as distinct controls. This preserves the bug's expected behavior, earlier craft finding B2 and the delivered caller contract. A fixture can establish assertion quality here. A live support claim still needs installed-host evidence.

## Authority, prior findings and retained evidence

- Parent-project Shark lookup returned the registered plan (420), research (413) and earlier independent reviews, including revision 30 (539). The original plan reviews and craft reviews are retained in the B077 worktree archive. I compared their findings with C09/C11 and the successor reports for revisions 28–30. For example, R3-04 survives in C04's failed non-force claim and zero-launch assertion. Each of the thirty rows has its six required columns and resolving section links into the separate mutable evidence record.
- At the declared baseline the keyed response supplied independent route fields, the native handoff omitted model selection, and Rider's long-running Codex example omitted a model argument. The separate Go dispatcher already supplied its own selector. Metadata from the six cited WWGM children confirms Luna/xhigh, and the three parent spawn calls omit model overrides. The B100/B101 inspection records lack a keyed join to those children. The research diagnosis holds.
- Owner decision 3576 supersedes capture-only scope. Adjudication 3609 and Architect advice 3619 retain the preferred exact Claude route, the six eligible fallback tuples and producing-worker correlation. Owner instruction 4008 on B083 preserves explicit user pins. B077 Architect correction 4011 requires same-CLI discovery, an explicit mapped selection, observation and rejection without another launch on access failure. Owner-directed Tech Director disposition 4273 chooses delivered revision 27 as the planning basis. I verified revision 27 in the PR #291 tree and revision 25 in the retained approval snapshot. The later disposition in the project owner's decision brief resolves its earlier open question. B083's E40 discovery remains separately owned.
- The E34-F01 implementation addendum retains scalar request fields. The E38-F09 owner gates govern host capability and parent control, and owner note 1977 assigns Question state to E39. The product X-05 assignment, F09 disposition 2041 and verification 2097 retain embedded distribution despite gate 003's earlier manual-only direction. Owner decision 3965 governs routed child rejection and recovery. The successor has no authority to change those decisions. CC-044's owner defers separate Claude transport to TD-238.
- The WWGM B100/B101 and B110 decisions and S010 review scope/retry decisions govern their owning workflows. The AgentMap routing brief is a recommendation. The archived B099 F-02 and B110 F-02 show independent ways to lose the earlier attempt or its known model and cost. C02/C12/C13 require zero preflight inference and preserve each attempt's own observed identity, usage and reported cost, including denial and missing amounts.

## Disposition of the preceding review

**4339 / R30-01** is covered for successful selection and suppression of stale dispatch. C02 changes only the selected CLI's independent response from A to B with a distinguishable ambient executable and profile. C16 checks the actual claim and stored model, C05 the producing turn, and C14 the legacy actor at the flag, storage and history readback. A cached A on the second successful request fails. R31-01 concerns the contents of the rejection record after an empty read.

**4340 / R30-02** is covered. C02/C05/C13 pair successful discovery with denial at the selected invocation, expose another available route and count one attempted dispatch, zero alternates, zero admissions and zero advances. They check stale final output, scoped cleanup and both known and missing accounting values. The matching control admits and advances once.

C06–C10 and C12–C30 retain prompt bytes, nested reviewer and reporter identity, schema and source checks, harness rendering, working directory, Question ownership, lease-qualified writes, recovery, configured outcomes, and child routing. Examples include C20's recovery without repeat inference, C24's original operation and lease at the child write, and C30's time-of-renewal check against the stored lease. The two findings above concern observable behavior within the authorized successor scope.

Return to repair planning for a new committed revision and independent review. Carry forward the diagnosis, declared code baseline, owner dispositions and unchanged prior evidence. This review records plan findings only. Candidate review, QA and live provider verification remain downstream.
