# Test Plan: E40-F06 - Stage evidence and evaluator isolation

**Created:** 2026-08-14
**Feature PRD:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/spec.md`
**Task Spec:** Not yet decomposed. This plan is written directly against
`spec.md` (Step 1-3 use `spec.md` as both the incremental spec and the
traceability target — the same posture E40-F05's test-plan.md used before
task decomposition, and the posture this epic has used for every feature
gated before its own task list exists).
**Status:** APPROVED, with two named, non-blocking coverage notes recorded
below (a repo-hygiene test case is inferred rather than named in `spec.md`'s
component table; REQ-NF-007's "without materially changing scenario wall
time" clause has no latency-threshold test) — see "Acceptance-criteria
review" and "Codex test-plan red-team (manual substitute)."

## Scope and drift analysis

`spec.md` is incremental over `feature.md` and
`architecture.md#stage-evidence-and-isolation-contract`, both already fixed
by the epic before this feature's own spec was written. Comparing `spec.md`
against `feature.md`:

- Every `spec.md` REQ-F-* and REQ-NF-* traces to a `feature.md` Scope,
  Acceptance boundary, or Contracts line: the three-root definition
  (REQ-F-002) → Scope bullet 1; the stage-snapshot field inventory
  (REQ-F-003, REQ-F-008, REQ-F-009) → Scope bullet 2 and the 2026-08-13
  value-attribution amendment; the eight-category `stage_category` closed set
  and six-category time ledger (REQ-F-003, REQ-F-005) → Scope bullet 3;
  candidate identity (REQ-F-006) → Scope bullet 4; artifact producer/consumer
  records and the empty-vs-absent `consumers` distinction (REQ-F-008) →
  Scope bullet 5; admission and dispatch-boundary isolation checks
  (REQ-F-010, REQ-F-011) → Scope bullet 6; ordered evaluator access
  (REQ-F-012) → Scope bullet 7; partial-evidence retention for stop outcomes
  (REQ-F-014) → Scope bullet 8. No REQ introduces a capability absent from
  `feature.md`'s Scope.
- Every `feature.md` Acceptance boundary bullet is covered: "exactly one
  addressable snapshot or a named missing-stage failure" → AC-003;
  "dispatch fails before provider spend if evaluator-only material is
  visible" → AC-009; "post-run oracle reads hidden inputs without copying
  into the worker checkout" → AC-011; "captured evidence re-evaluable
  without rerunning the worker" → AC-012, AC-013; "ledger reconciles without
  double counting, unknown time stays explicit" → AC-004; "replay detects
  changed tracked file, changed untracked file, changed test suite, or
  missing artifact-consumption record" → AC-012. No boundary bullet is left
  uncovered.
- `feature.md`'s Contracts section ("consume I-04"; "produce I-05… written by
  E40-F08 and consumed by E40-F09/F10"; "consume X-09… fail closed on
  missing comparison identity") matches `spec.md`'s Cross-feature
  interactions and Cross-epic integrations sections verbatim — no semantic
  drift in what is consumed vs. produced. The 2026-08-13 amendment
  ("F06 does not decide whether a gate or artifact was valuable") matches
  `spec.md`'s "Out of scope" bullets on scoring, comparison identity, and
  report layout exactly.
- `spec.md`'s own "Durable unresolved decisions" section resolves all five
  candidates with a materiality argument (X-09 field names reuse Q003;
  reconciliation epsilon magnitude and evidence-root location are
  non-material; the lifecycle-half "should have been dispatched" question is
  closed as undecidable by inherited ADR-F05-02; the E27-F15-merge
  envelope-survival question is newly proposed, non-blocking, and routed to
  the parent loop). No new Q### is warranted by this review; the fifth
  candidate is already correctly proposed rather than silently decided by
  F06.

**No drift found.** No BA or architecture refinement is required.

### Feature-level coverage check (component-changes table vs. AC/REQ)

`spec.md`'s Architecture "Component changes" table lists 15 new/modified
artifacts. Cross-referencing each against the AC list:

| Component | Owning REQ/AC |
|---|---|
| `bench/evidence/i05-schema.yaml` | REQ-F-017, AC-001, AC-020 |
| `bench/evidence/usage-mapping.yaml` | REQ-F-009, REQ-F-018, AC-007, AC-008, AC-021, AC-022 |
| `bench/scripts/verify-evidence-roots.sh` | REQ-F-010, REQ-F-011, AC-009, AC-010 |
| `bench/scripts/verify-stage-evidence.sh` | REQ-F-004..006, REQ-F-008, REQ-F-009, REQ-F-012, REQ-F-014, REQ-F-015; AC-004, AC-005, AC-006, AC-011 (via `adapter.sh inject-tests` ordering), AC-013, AC-014 |
| `bench/scripts/replay-stage-evidence.sh` | REQ-F-013, REQ-F-015, AC-012, AC-013 |
| `bench/scripts/canary-usagemapping.sh` | REQ-F-009, REQ-F-019, AC-007, AC-021 |
| `bench/scripts/testdata/evidence/` | fixture support for AC-004-AC-014 |
| `bench/scripts/tests/tc043_root_policy_isolation_test.sh` | AC-009, AC-010 |
| `bench/scripts/tests/tc044_time_ledger_reconciliation_test.sh` | AC-004 |
| `bench/scripts/tests/tc045_candidate_identity_test.sh` | AC-005 |
| `bench/scripts/tests/tc046_artifact_record_test.sh` | AC-006 |
| `bench/scripts/tests/tc047_usage_mapping_canary_test.sh` | AC-007, AC-021 |
| `bench/scripts/tests/tc048_evaluator_access_ordering_test.sh` | AC-011 |
| `bench/scripts/tests/tc049_snapshot_replay_test.sh` | AC-012, AC-013 |
| `bench/scripts/tests/tc050_partial_evidence_test.sh` | AC-014 |
| `bench/scripts/tests/tc051_evidence_offline_determinism_test.sh` | AC-016, AC-019 |
| `bench/scripts/tests/run-all.sh` | registration only, no independent AC |
| `bench/README.md` | documentation — reviewed, not executed (see "Integration scenarios") |
| `tests/contracts/e40_i05_stage_evidence_contract_test.go` (TC-042) | AC-001, AC-002, AC-003, AC-008, AC-015, AC-017, AC-020, AC-022 |
| `tests/contracts/testdata/e40_i05/{valid,invalid}/` | AC-001-003, AC-008, AC-015, AC-020 |

**Two gaps found, both non-blocking (named and owned below, same posture
F05's test-plan.md used for its own AC-020 gap):**

1. **AC-017's repo-hygiene assertion (`make fmt && make lint && make test`
   green, `go list ./...` clean, TC-042 passing without a populated
   submodule) names no dedicated `tcNNN` script in `spec.md`'s component
   table**, unlike F05's own AC-015/TC-037. This is a materially smaller
   check than F05's: F06 adds no new submodule, no new Go module boundary,
   and no fixture directory that could be mistaken for a `./...` package —
   its only Go addition is one file already exercised, and required to pass,
   by `make test` itself (`go test ./...` includes
   `tests/contracts/e40_i05_stage_evidence_contract_test.go`). A dedicated
   `tc052_repo_hygiene_test.sh` mirroring TC-037's populated/unpopulated
   submodule comparison would assert nothing F06 uniquely puts at risk,
   because F06 has no submodule state to compare. The verification method
   below (repo-root Makefile invocation, `go list ./...` grep, both run
   directly rather than through a dedicated wrapper script) is therefore
   recorded as this plan's resolution rather than as a missing deliverable;
   task decomposition may proceed from it. If a future feature in this epic
   *does* add a new submodule or fixture directory that risks entering
   `./...`, that feature — not F06 — should add the dedicated hygiene
   script, matching where F05 added TC-037 for its own two submodules.
2. **REQ-NF-007's "without materially changing scenario wall time" clause
   has no latency-threshold test.** The Verification plan table assigns
   REQ-NF-007 to "AC-004 (category-attribution branch)," which proves the
   dispatch-boundary check's own elapsed time is attributed to
   `tool_and_test` rather than `provider_active` — a correctness property —
   but proves nothing about the check's absolute or relative speed. No AC in
   `spec.md` states a numeric or comparative performance bound (e.g. "adds no
   more than N milliseconds per dispatch," "adds no more than X% to scenario
   wall time"), so no test could assert one without inventing a threshold
   `spec.md` does not authorize. This plan therefore treats REQ-NF-007 as
   covered for its testable half (category attribution, AC-004/TC-044) and
   records the untestable half (materiality of wall-time impact) as a
   qualitative code-review check — the guard must be a bounded filesystem
   walk over two roots plus a name-diff, not a network call or an
   unbounded scan — rather than a fabricated performance test. This mirrors
   how `spec.md`'s own Verification plan treats REQ-NF-005 (diff review, not
   a test).

This is why the plan below reaches **APPROVED** rather than
**NEEDS_REFINEMENT**: every AC has a test with a concrete oracle, and both
named coverage notes are resolved with an explicit, non-fabricated method
rather than left silent.

## Test tiers

Mirrors F05's tiering rationale, adapted: F06 needs no fixture submodule at
all for its Tier-1 validator (REQ-NF-003 — `bench/evidence/**` and
`tests/contracts/testdata/e40_i05/**` are committed, in-repo artifacts), but
its execution-based guards (isolation, replay, ledger, canary) genuinely need
real filesystem roots and, for the isolation guard specifically, a real I-04
package (which does require the `bench/fixture-py` submodule the guard's
`checkout-scenario-fixture.sh` call provisions).

| Tier | Runs | Needs submodule? | Where |
|---|---|---|---|
| **Tier 1** | `make test` (CI + every dev machine) | No — reads only committed I-05 schema, mapping, and bundle fixtures under `bench/evidence/**` and `tests/contracts/testdata/e40_i05/**` | `tests/contracts/e40_i05_stage_evidence_contract_test.go` (TC-042) |
| **Tier 1b** | Curator, manually or via `bench/scripts/tests/run-all.sh` | Some cases yes (root-isolation tests against a real I-04 package fixture checkout), some no (ledger, artifact-record, and stop-outcome cases run against committed bundle fixtures with no live checkout) | `bench/scripts/tests/tc043_*.sh` through `tc051_*.sh` |
| **Tier 2** | Curator, at evidence-schema or usage-mapping change time | Yes for `tc043` (root-isolation guard against a real fixture checkout and scratch project) and `tc047` (canary against a real or operator-supplied transcript) | `bench/scripts/{verify-evidence-roots,canary-usagemapping}.sh` against real roots |

Tier 2 is what REQ-NF-004's "byte-identical verdicts... at an unchanged
bundle, fixture SHA, and toolchain identity" exercises for the execution
guards. It is **not** gated by root `make test`; `bench/README.md`'s new
"I-05 stage evidence and isolation contract" section must name the exact
Tier 2 invocation sequence (matching F05's own precedent for its own
section) so "curator re-runs it" is a real, documented action.

## Determinism and offline boundary (REQ-NF-004, AC-016)

Same class of incidental non-determinism F01 and F05 both had to name
explicitly, now applied to evidence guards rather than admission:

- No guard's verdict may depend on wall-clock time beyond what the bundle
  itself already records (`stage_start`, `stage_end`, digests). A guard that
  reads the current system time and folds it into a comparison (e.g.
  "warn if this bundle is older than N days") would break AC-016's
  byte-identical-across-repeated-runs requirement; no such behavior is
  specified, and TC-051 explicitly re-runs every guard twice over the same
  bundle to catch one if introduced.
- `verify-stage-evidence.sh`'s JSON output (verdicts, named failing fields)
  must be emitted in a fixed order — by `dispatch_ordinal`, then by field
  name within a stage — so two runs' full output is byte-comparable, the
  same discipline F05's `admit-scenario.sh` used (sorted by `scenario_id`).
- `canary-usagemapping.sh` must not depend on which of the committed
  envelope fixtures under `bench/scripts/testdata/run/` happens to be listed
  first in a directory read; TC-047 and TC-051 both assert this by running
  the canary twice and diffing stdout byte-for-byte.
- TC-051 provisions the network-disabled state the same way F05's TC-039
  did (Linux `unshare --net`, or the portable `GOPROXY=off`/poisoned-proxy
  fallback), because every guard under test here is pure filesystem and
  digest computation and must never depend on the network at all — unlike
  F05's admission gate, no I-05 guard is expected to need network access in
  any circumstance, so REQ-NF-004's "byte-identical… zero provider calls"
  claim is the stronger of the two postures (no attempted call to detect,
  only an absence to prove).

## AC test matrix

| AC | Requirement(s) | Tier | Technique | Test case | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|---|
| AC-001 | REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-005, REQ-F-006, REQ-F-008, REQ-F-009, REQ-F-017 | 1 | Contract-surface enumeration | TC-042 | `TestTC042_I05StageEvidenceContract` reads every fixture bundle under `tests/contracts/testdata/e40_i05/valid/` (`bundle.json` plus its `stages/*.json`) via `os.ReadFile` and validates against `bench/evidence/i05-schema.yaml`'s field inventory and closed vocabularies | Every REQ-F-002/003/005/006/008/009 field is present and well-typed in every valid fixture; `schema_version` matches the version the validator supports; every closed-vocabulary value used (`stage_category`, interval category, `artifact_type`, `edge_kind`, `evaluator_access.phase`, stop outcome, error kind) resolves against `i05-schema.yaml`. Negative: a fixture using a vocabulary value not present in `i05-schema.yaml` is rejected naming the field and the offending value, not accepted by convention because the real committed fixtures happen not to use it. |
| AC-002 | REQ-F-002 | 1 | Decision table (3 required roots × {present-correct, missing, nested-pair}) | TC-042 | Table-driven subtests: a valid fixture with all three `roots` entries pairwise disjoint; a fixture declaring only two roots; a fixture whose `roots.scratch_shark_project.path` is nested inside `roots.agent_fixture_checkout.path` | Valid fixture accepted with all three `worker_access` modes (`read_write`, `authorized_surfaces_only`, `never_during_dispatch`) verified against their declared root. Negative: the two-root fixture is rejected naming the missing root; the nested-pair fixture is rejected naming both members of the offending pair — not a generic "invalid roots" message. |
| AC-003 | REQ-F-004 | 1 | Decision table (prelude oracle × lifecycle oracle, evaluated independently) | TC-042 | Table-driven subtests: (a) a prelude-half fixture with `stage_matrix.prelude.D03.applicable: true` and no `D03` entry in `stages[]`; (b) a lifecycle-half fixture with one observed dispatch (a stage snapshot present) and no matching `stages[]` index entry; (c) a lifecycle-half fixture with two `stages[]` entries sharing one `dispatch_ordinal`; (d) a lifecycle-half fixture with a fully consistent one-snapshot-per-dispatch set and no missing-stage claim | (a) rejected as a named `missing_stage` failure identifying `D03`. (b) rejected as an unmatched-dispatch failure. (c) rejected as a duplicate-dispatch-ordinal failure. (d) accepted with **no** `missing_stage` verdict available for the lifecycle half — proving the two halves are evaluated by genuinely different oracles rather than one shared rule. Negative: a validator implementation that emits a `missing_stage` verdict for the lifecycle half on case (d) (e.g. by wrongly treating `all_dispatched` as an enumerable set) is itself the defect REQ-F-004's last sentence exists to catch, and TC-042 asserts its *absence*, not merely its correctness when present. |
| AC-004 | REQ-F-005, REQ-NF-007 (category-attribution branch) | 1b | Boundary value analysis (`reconciliation_epsilon_ns`) + attack-class enumeration (interval overlap, window escape) | `bench/scripts/tests/tc044_time_ledger_reconciliation_test.sh` | `verify-stage-evidence.sh` against dedicated bundle fixtures under `bench/scripts/testdata/evidence/`: (i) six disjoint half-open intervals whose union reconciles exactly within epsilon; (ii) two intervals from different categories overlapping by one nanosecond; (iii) one interval whose `end` exceeds `stage_end`; (iv) a residual larger than `reconciliation_epsilon_ns` left unassigned; (v) a residual **within** epsilon, asserted to land in `unclassified`, never `provider_active` | (i) accepted. (ii) rejected naming both overlapping categories. (iii) rejected naming the escaping interval and stage window. (iv) rejected naming the residual magnitude. (v) accepted with the residual visible in the fixture's `unclassified` interval list and absent from `provider_active`. This is also REQ-NF-007's testable half: the dispatch-boundary check's own synthetic elapsed interval in a companion fixture is asserted to be categorized `tool_and_test`, never `provider_active`. Negative: an implementation assigning an unattributed residual to `provider_active` "to be safe" is exactly the defect case (v) exists to catch. |
| AC-005 | REQ-F-006 | 1b | Decision table (5 required `candidate` fields × missing-field rejection) + attack-class enumeration (`base_commit`-alone identity) | `bench/scripts/tests/tc045_candidate_identity_test.sh` | `verify-stage-evidence.sh` against a `code`-category snapshot fixture missing exactly one of `tree_digest`, `binary_diff_digest`, `changed_path_digest`, `dirty_untracked_manifest`, `test_suite_digest` (five independent fixtures, one per field), plus two snapshot fixtures sharing one `base_commit` value but differing in `tree_digest` | Each of the five missing-field fixtures is rejected naming that exact field. The two same-`base_commit` snapshots are reported as **distinct** candidates (not merged/deduplicated), proving `base_commit` alone never establishes identity per ADR-009. Negative: an implementation treating two snapshots with equal `base_commit` as "the same candidate" without checking the other four fields would silently conflate a dirty-tree edit with a clean one — this is the exact regression the two-snapshot case exists to catch. |
| AC-006 | REQ-F-008 | 1b | Equivalence partitioning (`consumers: []` vs. absent `consumers` — two distinct, both-legal states) | `bench/scripts/tests/tc046_artifact_record_test.sh` | `verify-stage-evidence.sh` against one bundle fixture whose `artifacts[]` contains one entry with `consumers: []` and one entry with the `consumers` key entirely omitted | The empty-array entry yields verdict `orphan`; the absent-key entry yields verdict `consumption_evidence_missing`. Neither verdict is produced for the other entry — asserted by checking both directions of non-coercion (the empty-array entry is never reported `consumption_evidence_missing`, and the absent-key entry is never reported `orphan`). Negative: a YAML/JSON decoder that defaults an absent key to an empty slice (a common Go `omitempty`/zero-value trap) would collapse the two verdicts into one — TC-046 fails loudly on exactly that collapse. |
| AC-007 | REQ-F-009 | 2 | Attack-class enumeration (drift class (a): mapped path absent from a real envelope) | `bench/scripts/tests/tc047_usage_mapping_canary_test.sh` | `bench/scripts/canary-usagemapping.sh`, once against the committed envelope fixtures under `bench/scripts/testdata/run/` (agreeing case) and once against a copy of the same fixture with one mapped path (e.g. `usage.cache_read_input_tokens`) deleted (drifted case); an `--transcript <path>` invocation against an operator-supplied real transcript is exercised as a smoke path, not asserted against a specific value | Agreeing case: every `anthropic_claude_cli` slot in `usage-mapping.yaml` resolves against the committed envelope, canary exits `0`. Drifted case: canary fails naming the exact slot (`cache_read_input_tokens`) and envelope path (`usage.cache_read_input_tokens`) — not a generic "mapping mismatch." Negative: a canary that reports success merely because the file parses as JSON, without checking every individual mapped path resolves, would pass the agreeing case but falsely pass the drifted case too — TC-047 requires the drifted case to fail specifically. |
| AC-008 | REQ-F-009, REQ-F-018 | 1 | Decision table (unmapped-provider fail-closed × `verification_tier` presentation) | TC-042 | Table-driven subtests: (a) a snapshot fixture claiming decoded `usage` values under provider `openai_codex_cli`, which `usage-mapping.yaml` declares `unmapped`; (b) a snapshot fixture recording the same provider's usage slots as absent with a matching `usage_slot_unavailable` error entry; (c) a snapshot fixture presenting a slot whose mapping-declared `verification_tier` is `unverified` (e.g. `provider_session_id`) as part of required/verified identity; (d) the same slot recorded as opportunistic, non-identity evidence | (a) rejected naming the provider `openai_codex_cli` as unmapped. (b) accepted. (c) rejected naming the slot and its `unverified` tier. (d) accepted. Negative: an implementation that treats "provider declared unmapped" as merely a warning rather than a hard rejection of any decoded value under that provider fails case (a)'s assertion that rejection is unconditional. |
| AC-009 | REQ-F-011 | 1b/2 | Attack-class enumeration (planted-leak locations, exhaustively: fixture checkout, scratch project, clean-roots control, pre-dispatch timing) | `bench/scripts/tests/tc043_root_policy_isolation_test.sh` | `bench/scripts/verify-evidence-roots.sh <package.yaml> <fixture_checkout> <scratch_project> <evaluator_root>`, run four times: (a) an evaluator-only file (from the package's `evaluator_only.reference_solution`) planted inside the fixture checkout; (b) the same file planted inside the scratch Shark project instead; (c) both roots left clean; (d) case (a) repeated with a PATH-stubbed dispatcher binary that records every invocation to a log file, verifying the log is empty after the guard's failing exit | (a) fails naming the fixture-checkout root, the exact planted path, and the matched `evaluator_only` source. (b) fails the same way but naming the scratch-project root — proving a guard walking only `--workdir` cannot pass this case. (c) exits `0` — clean roots produce no false positive. (d) fails exactly as (a) does, **and** the stubbed dispatcher log contains zero invocations, so "before provider spend" is observed via a non-invoked dispatcher rather than inferred from the guard's own non-zero exit status alone. Negative: a guard that only checks `--workdir` (agent-visible fixture checkout) and silently ignores the scratch project passes case (a) but fails to catch case (b) — TC-043 requires both to independently fail. |
| AC-010 | REQ-F-010 | 1b/2 | Contract-surface enumeration (dynamic name derivation, not a hardcoded list) | `bench/scripts/tests/tc043_root_policy_isolation_test.sh` | The admission-time check invoked twice against two scratch copies of the same package: once with the package's original `oracle_tests[]` entry name, once with that entry renamed in the scratch copy (no edit to the guard script itself) | Both invocations succeed at detecting the (renamed or original) evaluator-only test identity — the guard's search target changes to match whichever name the package under test declares, proving REQ-F-010's "names MUST be derived from the package at call time, never from a hardcoded list." Negative: a guard with the original test name compiled or grepped in as a literal string would fail to detect the renamed entry's presence in a planted-leak variant of this same test — TC-043 exercises exactly that renamed case. |
| AC-011 | REQ-F-012 | 1b | State transition (ordered access: pre-terminal reject, post-terminal accept, in-place-read accept, copy-before-completion reject) | `bench/scripts/tests/tc048_evaluator_access_ordering_test.sh` | Four cases against `verify-stage-evidence.sh` and a real `bench/adapters/<name>/adapter.sh inject-tests` invocation: (a) `inject-tests` invoked while the scenario's terminal status has not yet been reached; (b) the same invocation after terminal status; (c) an oracle read of `evaluator_only.reference_solution` performed in place from the evaluator-only root, after terminal status; (d) the same read performed by first copying the file into the worker checkout before execution completes | (a) rejected as `isolation_violation`. (b) accepted, and the bundle's `access.jsonl` gains one `evaluator_access` event with `{accessor, artifact_path, digest, phase, granted_at}` all populated. (c) accepted with the same event-append guarantee. (d) rejected naming the violation — the pre-completion copy, not the eventual read. Negative: an implementation that only checks *whether* evaluator material was eventually read, and not *when* relative to terminal status/execution completion, would accept case (a) and (d) — TC-048 requires both to fail. |
| AC-012 | REQ-F-013 | 1b | State transition (four independent drift kinds) | `bench/scripts/tests/tc049_snapshot_replay_test.sh` | `bench/scripts/replay-stage-evidence.sh` against a stored bundle fixture and its named roots, with a PATH-stubbed provider binary that exits non-zero and logs any invocation, run once per mutation: (a) a tracked file in the roots edited after the snapshot was taken; (b) an untracked file added; (c) the adapter's normalized test-id set changed (one test id added or removed); (d) one artifact's `consumers` key deleted from the stored snapshot | (a) reported as `tracked_file_changed` naming the path. (b) reported as `untracked_file_changed` naming the path. (c) reported as `test_suite_changed` naming the differing test id. (d) reported as `artifact_consumption_record_missing` naming the artifact. All four replays complete with **zero** invocations recorded in the stubbed-provider log, proving "no worker rerun and no provider call" empirically rather than assuming it. Negative: a replay implementation that shells out to the real worker/provider to "double-check" a drift would be caught by the non-zero stub-invocation-count assertion. |
| AC-013 | REQ-F-015 | 1b | Boundary value analysis (one-byte mutation) + state transition (recompute-reproduces-recorded) | `bench/scripts/tests/tc049_snapshot_replay_test.sh` | `replay-stage-evidence.sh` recomputing `snapshot_digest` over (i) an unmodified stored snapshot and (ii) the same snapshot with exactly one byte of one field (e.g. `rework_count`) changed | (i) the recomputed digest equals the recorded `snapshot_digest`. (ii) recomputation yields a different digest and the guard reports `snapshot_mutated` naming the stage. Negative: an implementation that recomputes the digest **including** the `snapshot_digest` field itself would produce a spurious mismatch on case (i) even with no real mutation — TC-049 asserts case (i) passes, which fails for that implementation bug. |
| AC-014 | REQ-F-014 | 1b | Decision table (10 named stop outcomes × {partial evidence retained, `publication_eligible: false`, `ineligibility_reasons[]` non-empty}) + attack-class enumeration (eligible-with-stop-outcome rejection) | `bench/scripts/tests/tc050_partial_evidence_test.sh` | `verify-stage-evidence.sh` against ten bundle fixtures, one per stop outcome (`resource_limit`, `lease_loss`, `missing_outcome`, `unresolved_gate`, `pause`, `archive`, `error`, `cancellation`, `worker_failure`, `timeout`), each carrying at least one partial stage snapshot; plus one fixture pairing a stop outcome with `publication_eligible: true` | All ten fixtures: partial snapshots present and readable, `publication_eligible: false`, `ineligibility_reasons[]` non-empty. The eleventh (eligible-with-stop-outcome) fixture is rejected naming the contradiction. Negative: an implementation that discards partial snapshots for a stopped run (rather than merely marking the bundle ineligible) fails the "partial evidence retained" assertion for every one of the ten cases, not just one. |
| AC-015 | REQ-F-016 | 1 | Decision table (11 named malformed-field rejection cases) | TC-042 | Table-driven subtests over `tests/contracts/testdata/e40_i05/invalid/`, one fixture per REQ-F-016 case: missing/overlapping root; unknown `stage_category`; unknown interval category; overlapping ledger; non-reconciling ledger; `code`/`review` snapshot missing a `candidate` field; artifact record missing `producer_stage`; artifact record missing `digest`; zero-valued usage slot where the mapping reports the field absent; `evaluator_access` event out of authorized order; stop outcome with `publication_eligible: true`; duplicate dispatch ordinal; unsupported `schema_version` | Each of the 11 named cases exits non-zero with the failing field named in the error, matching that case's own defect, not a generic "invalid bundle" message. Negative: a fixture correcting exactly one field passes, proving the validator is not rejecting the whole file for an unrelated reason. |
| AC-016 | REQ-NF-004 | 1b/2 | State transition (repeated-run byte identity, network-disabled) | `bench/scripts/tests/tc051_evidence_offline_determinism_test.sh` | Every guard (`verify-evidence-roots.sh`, `verify-stage-evidence.sh`, `replay-stage-evidence.sh`, `canary-usagemapping.sh`) invoked twice, back to back, over the same bundle/roots, with the network disabled per the "Determinism and offline boundary" section above and a PATH-stubbed provider recording invocations | Both invocations of every guard produce byte-identical stdout/exit code, and the stubbed-provider invocation log is empty after all four guards complete. Negative: a guard reading wall-clock time or directory-listing order into its output would fail the byte-identity comparison; a guard silently attempting a network call would either hang (caught by a timeout) or fail under the isolation mechanism. |
| AC-017 | REQ-NF-001, REQ-NF-002, REQ-NF-003 | 1 | Boundary/state enumeration (submodule-absent CI state vs. this checkout's live state) | TC-042 (repo-root Makefile gate; no dedicated `tcNNN` script — see "Feature-level coverage check" gap note above) | Repo root `make fmt && make lint && make test`, run against this checkout's live state; separately, `go test ./tests/contracts/...` run against an isolated copy of the tree with no fixture submodule initialized at all (CI-like, matching `actions/checkout@v4`'s default), confirming `tests/contracts/e40_i05_stage_evidence_contract_test.go` needs no submodule; `go list ./...` checked for any evidence-fixture or evaluator-only package | `make fmt && make lint && make test` is green; `go test ./tests/contracts/...` passes identically with no submodule present; `go list ./...` lists no `bench/evidence`, `bench/scripts/testdata/evidence`, or evaluator-only package (none of these are Go packages to begin with, so this is a structural, not incidental, guarantee); `.github/workflows/ci.yml` is confirmed byte-unchanged (folded into AC-018's diff review). Negative: a future change accidentally adding a `.go` file under `bench/scripts/testdata/evidence/` would surface here as a new `go list` entry. |
| AC-018 | REQ-NF-006 | N/A (diff review, not an automated TC) | Attack-class enumeration (frozen-interface / non-regression, reviewed not executed) | Diff review | `git diff` against `bench/corpus/corpus.yaml`, `bench/scripts/collect-run.sh`, `bench/scripts/verify-clean-checkout.sh`, `bench/scripts/canary-runsurface.sh`, `tests/contracts/e40_i01_corpus_contract_test.go`, and every I-04 artifact REQ-NF-006 names (`bench/scenarios/**`, `bench/adapters/**`, `bench/scripts/admit-scenario.sh`, `eval-predicate.sh`, `checkout-scenario-fixture.sh`, `tests/contracts/e40_i04_scenario_contract_test.go`); a repo-wide check that no file under `internal/` or `cmd/` is touched | Every named file's diff is empty (byte-unchanged); no file under `internal/` or `cmd/` appears in the feature's changeset. This is the same non-automatable method `spec.md`'s own Verification plan uses for REQ-NF-005 — a diff/grep review performed at code-review time, not a test asserting behavior. |
| AC-019 | REQ-F-007 | 1b | Attack-class enumeration (forbidden-token leak surface, generic-component language neutrality) | `bench/scripts/tests/tc051_evidence_offline_determinism_test.sh` | `grep -rE 'python|pytest|pip|go test|golangci-lint|go build'` over every generic evidence/isolation/replay script (`verify-evidence-roots.sh`, `verify-stage-evidence.sh`, `replay-stage-evidence.sh`, `canary-usagemapping.sh`, `bench/scripts/tests/tc04[3-9]_*.sh`, `tc05[01]_*.sh`, excluding `tc051` itself and anything under `bench/adapters/*/`) | Zero hits outside `bench/adapters/*/`, mechanically proving REQ-F-007's "no generic evidence component branches on Python, Go, or a package manager" — the same mechanical proof F05's AC-012/AC-019 established for I-04's admission tooling. Negative: a future edit inlining a `pytest` invocation into `verify-stage-evidence.sh` to compute `test_suite_digest` "the easy way" is caught by this same grep, mechanically rather than by code-review convention alone. |
| AC-020 | REQ-F-017 | 1 | Contract-surface enumeration (single-owner vocabulary agreement, bidirectional) | TC-042 | `TestTC042_I05StageEvidenceContract` cross-checks that every closed-vocabulary value the validator accepts appears in `bench/evidence/i05-schema.yaml`, plus a fixture whose bundle uses a vocabulary value present in `i05-schema.yaml` but absent from any committed bundle fixture, and a fixture whose bundle uses a value present in neither | The schema-only value surfaces as a named, non-fatal "declared but unexercised" note (schema and bundle fixtures may legitimately diverge in coverage), while the value present in neither is rejected as an unknown vocabulary entry. Negative: a validator embedding its own private copy of the vocabulary (rather than reading `i05-schema.yaml`) would silently diverge if the YAML file changed without the Go code changing — TC-042 reads the vocabulary from the YAML file at test time specifically to catch that divergence, not from a Go constant. |
| AC-021 | REQ-F-019 | 2 | Attack-class enumeration (drift class (b): envelope-availability drift, decoded not inferred) | `bench/scripts/tests/tc047_usage_mapping_canary_test.sh` | `canary-usagemapping.sh` against a transcript fixture whose `---STDOUT---` block is arbitrary non-JSON text (e.g. plain assistant prose, matching the shape the E27-F15 branch would actually produce per ADR-F06-11, but asserted as a decoder-robustness case, not a claim about that branch's real output) | Fails as `envelope_source_unavailable` naming the transcript path — a single, whole-source failure, **not** nine independent `usage_slot_unavailable` entries, one per slot. Negative: an implementation that tries to decode each semantic slot independently against non-JSON input and reports nine separate per-slot failures instead of recognizing "the source itself isn't an envelope" fails this assertion — this is precisely the confusion REQ-F-019/ADR-F06-11 exists to prevent, and TC-047 checks the failure count and kind, not merely that failure occurs. |
| AC-022 | REQ-F-018 | 1 | Decision table (`required_identity_slots` declaration validity × per-snapshot completeness) | TC-042 | Table-driven subtests: (a) `usage-mapping.yaml` as committed, asserting every entry in `required_identity_slots` has `verification_tier: real_capture`; (b) a mutated copy of the mapping listing `provider_session_id` (`unverified`) as required; (c) a snapshot fixture carrying every slot in `required_identity_slots`; (d) a snapshot fixture missing one required slot | (a) passes — every required slot is `real_capture`. (b) rejected naming `provider_session_id` as the offending required-but-unverified slot. (c) accepted as identity-complete. (d) rejected naming the missing slot. Negative: an implementation that only checks the *count* of required slots present, not which specific slots, would pass case (d) if the snapshot happened to carry a different, non-required slot as a substitute — TC-042 asserts the specific slot name, not a count. |

## Acceptance-criteria review

Every AC above is unambiguous, testable, traceable to a `spec.md` REQ, and
specifies an exact expected output (a named failing field, an exact
boundary state, a byte-identical comparison, a specific verdict string)
rather than "works correctly" or "handles errors gracefully." No AC is an
open-ended robustness assertion. REQ-NF-004's "byte-identical" is the
closest candidate to open-endedness and is closed by the "Determinism and
offline boundary" section above, the same way F05's plan closed the
analogous claim. Every runtime AC above has at least one explicit negative
case in the matrix.

**Two coverage notes, both non-blocking (repeated here from "Feature-level
coverage check" for visibility):**

1. AC-017 has no dedicated `tcNNN_*.sh` wrapper script; the verification
   method is a direct repo-root Makefile invocation plus a `go list`/diff
   check. This is deliberately thinner than F05's own AC-015/TC-037 because
   F06 introduces no new submodule, module boundary, or fixture directory
   for a hygiene script to meaningfully compare — the risk TC-037 exists to
   catch (a fixture package accidentally entering `go list ./...`) has no
   analog surface in this feature's deliverables.
2. REQ-NF-007's "without materially changing scenario wall time" clause is
   covered only for its testable half (time-category attribution, AC-004);
   no AC states a numeric performance bound, so none is tested. A
   qualitative code-review note (bounded filesystem walk, no network call)
   is recorded instead of an invented threshold.

Both notes were reached independently of any external red-team pass — see
"Codex test-plan red-team (manual substitute)" below for the self-critique
process that surfaced them, since Codex CLI is unavailable in this
environment.

## ISTQB technique application

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-001 | Contract-surface enumeration | TC-042 | I-05 is a cross-feature interaction surface (F08/F09/F10 read it); every field and every closed-vocabulary value must be enumerated against valid fixtures, not sampled. |
| AC-002 | Decision table | TC-042 | Three required roots × {present-correct, missing, nested} is a small, fully enumerable combinatorial space; a decision table forces every cell. |
| AC-003 | Decision table | TC-042 | REQ-F-004 states two genuinely different completeness oracles; a decision table with an explicit "no verdict available" cell is the only technique that can prove the two are actually different rather than one masquerading as two. |
| AC-004 | Boundary value analysis + attack-class enumeration | tc044 | The epsilon is an explicit numeric boundary (BVA: at/over the boundary); overlap and window-escape are defensive/leak-surface properties over interval pairs (attack-class enumeration), not a single happy-path assertion. |
| AC-005 | Decision table + attack-class enumeration | tc045 | Five independently-required fields is a decision table (one row per missing field); "commit id alone is not identity" is a leak-surface property an attacker (or a lazy implementation) could exploit by conflating two distinct trees under one commit. |
| AC-006 | Equivalence partitioning | tc046 | `consumers: []` and absent `consumers` are exactly two partitions the schema declares distinct; the technique's job is proving the boundary between them holds, not merely that one of the two states works. |
| AC-007 | Attack-class enumeration | tc047 | A drifted upstream envelope is a supply-chain-shaped attack/failure surface on the mapping's own correctness; the canary must detect the drift, not merely confirm the happy path. |
| AC-008 | Decision table | TC-042 | Fail-closed posture is a small decision table (mapped/unmapped × verified/unverified × claimed/recorded-absent); every cell must have its own named verdict. |
| AC-009 | Attack-class enumeration | tc043 | "Never visible to the worker" is a defensive property; the two independent planted-leak locations plus the pre-dispatch timing check are the enumerated leak surface, not a single happy-path check. |
| AC-010 | Contract-surface enumeration | tc043 | The guard's obligation is to derive names from the package's own declared contract at call time, not from a hardcoded list — proven by varying the contract's declared name, not the guard's code. |
| AC-011 | State transition | tc048 | "Absent before, present only after, in place not by copy" is inherently an ordered sequence of states; state-transition testing is the technique built to enumerate exactly that ordering. |
| AC-012 | State transition | tc049 | Replay drift detection is a before/after state comparison over four independent dimensions (tracked file, untracked file, test suite, artifact record). |
| AC-013 | Boundary value analysis + state transition | tc049 | A one-byte mutation is the minimal boundary a content-addressed digest must detect; recompute-reproduces-recorded is a two-state (unmodified/mutated) transition check. |
| AC-014 | Decision table + attack-class enumeration | tc050 | Ten named stop outcomes is a decision table by definition; "eligible with a stop outcome" is the leak/inconsistency case attack-class enumeration is built to catch. |
| AC-015 | Decision table | TC-042 | REQ-F-016 lists 11 distinct named rejection cases; each is a row a table-driven test must exercise independently with its own expected failing field. |
| AC-016 | State transition | tc051 | Reproducibility is a claim about two executions of the same state under network isolation — the technique built to enumerate exactly that. |
| AC-017 | Boundary/state enumeration | TC-042 | Submodule-present vs. submodule-absent is the two states worth distinguishing for REQ-NF-003's CI claim. |
| AC-018 | Attack-class enumeration (reviewed, not executed) | Diff review | Frozen-interface regression is a defensive property against silent scope creep into files this feature must not touch. |
| AC-019 | Attack-class enumeration | tc051 | "No generic component branches on language" is a leak-surface property; grep is the mechanical enumeration of that surface, not a convention trusted by inspection. |
| AC-020 | Contract-surface enumeration | TC-042 | Single-owner vocabulary agreement is a bidirectional contract-surface check (schema→bundle and bundle→schema), not a one-directional sanity check. |
| AC-021 | Attack-class enumeration | tc047 | Whole-source availability loss is a distinct failure mode from per-field drift; enumerating it as its own attack class is what stops it from being misdiagnosed as nine unrelated field errors. |
| AC-022 | Decision table | TC-042 | `required_identity_slots` validity and per-snapshot completeness combine into a small decision table (declaration-valid/invalid × snapshot-complete/incomplete), each cell independently asserted. |

## Caller-Path Contracts

This feature is deterministic runtime tooling (bash scripts executing real
filesystem checks, digest computation, and one real adapter subprocess call,
plus one Go contract test), matching F05's posture exactly. `content-only`
opt-outs apply **only** to `bench/README.md`'s documentation additions (see
"Integration scenarios"); every other row below drives its real production
entrypoint. Unlike F05, most of this feature's scripts have **no
application-level caller above them at all** — F08 has not been decomposed,
so today the shell script or Go test function under test IS the
entrypoint, not a wrapper invoked by some other Shark surface. This is
stated explicitly per row below, not left implicit, per the workflow's
instruction not to invent a caller-path fiction where none exists.

| TC | Entrypoint (exact invocation) | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-042 | `TestTC042_I05StageEvidenceContract` (the contract test function itself; internal — no caller above the Go test binary) calling `os.ReadFile` + real YAML/JSON unmarshal against `bench/evidence/i05-schema.yaml`, `bench/evidence/usage-mapping.yaml`, and `tests/contracts/testdata/e40_i05/{valid,invalid}/*` | Real filesystem read of committed YAML/JSON, real parser | Do not substitute an in-memory struct for the committed schema/mapping/fixture files; must parse the real files | A validator reading a hand-built in-memory manifest would stay green even if the real committed schema or a real fixture bundle were malformed — the same trap F05's TC-030 Caller-Path Contract names. |
| tc043 | `bench/scripts/verify-evidence-roots.sh <package.yaml> <fixture_checkout> <scratch_project> <evaluator_root>` invoked directly (internal — `verify-evidence-roots.sh` IS the entrypoint; no F08 dispatcher exists yet to call it in production) | Real filesystem walk of two live roots; real content-digest computation | Do not stub the planted-leak file's presence with a metadata flag; the file must actually exist on disk at the path under test. A PATH-stubbed dispatcher binary is the one **permitted** mock, used only to prove zero invocations occurred — never to simulate the guard's own filesystem check | A guard asserting isolation from a config flag instead of a real filesystem walk would report "clean" even with a real leaked file present. |
| tc044 | `bench/scripts/verify-stage-evidence.sh <bundle_dir>` (ledger-reconciliation portion) invoked directly against real bundle fixtures under `bench/scripts/testdata/evidence/` (internal — no caller above the script exists yet) | Real fixture files on disk, real interval-arithmetic computation | Do not hand-compute the expected reconciliation result inside the test harness and compare guard output against it without the guard itself performing the arithmetic; the guard must do the summation | A test harness that pre-computes "should reconcile" and merely checks the guard agrees, without the guard doing real interval math, would pass even if the guard's own arithmetic were wrong on a case not covered by the harness's pre-computation. |
| tc045 | `bench/scripts/verify-stage-evidence.sh <bundle_dir>` (candidate-identity portion), same entrypoint as tc044 | Real fixture files, real field-presence and field-equality checks | Do not stub `base_commit` comparison logic separately from the real snapshot JSON; must read the actual fixture fields | A test that asserts the guard's *design* rejects `base_commit`-alone identity, without an actual fixture that has matching `base_commit` and differing `tree_digest`, would not catch an implementation that silently merges the two. |
| tc046 | `bench/scripts/verify-stage-evidence.sh <bundle_dir>` (artifact-record portion), same entrypoint as tc044/tc045 | Real fixture bundle with one empty-array and one absent-key `consumers` entry | Do not represent the two states as two separate boolean flags in test setup instead of the real JSON shapes (`[]` vs. omitted key) — the distinction under test is a JSON-encoding distinction, not a semantic label the test harness assigns | A harness that tags entries "orphan"/"missing" itself, rather than letting the guard derive the verdict from the JSON shape, would pass even if the guard's own JSON-shape detection were broken. |
| tc047 | `bench/scripts/canary-usagemapping.sh [--transcript <path>]` invoked directly against real committed envelope fixtures and, for the smoke path, an operator-supplied real transcript (internal — the canary script IS the entrypoint; no F08 runtime caller exists yet) | Real envelope fixture files under `bench/scripts/testdata/run/`, real YAML parse of `usage-mapping.yaml`, real JSON decode of the transcript's `---STDOUT---` block | Do not hand-author a JSON envelope shaped to make the mapping "pass" — must use the committed real-capture fixture (or a mutated copy of it for the drifted case); do not simulate the non-JSON case with a JSON payload wrapped in a string, which would defeat the decoder-robustness assertion | A canary asserting against a hand-authored fixture that happens to agree with the mapping would pass even if the mapping disagreed with the real captured envelope shape `bench/README.md` documents. |
| tc048 | `bench/scripts/verify-stage-evidence.sh <bundle_dir>` (access-ordering portion) plus a real `bench/adapters/<name>/adapter.sh inject-tests --checkout <dir>` invocation (internal — no F08/F09 caller exists yet; `adapter.sh inject-tests` is I-04's real capability, invoked directly, not stubbed) | Real `adapter.sh` subprocess invocation for the post-terminal injection case; real file copy for the negative pre-completion-copy case | Do not stub `adapter.sh inject-tests`'s effect with a direct file write bypassing the adapter — the ordering guard must observe the adapter's real placement mechanism, per REQ-F-012's explicit instruction to reuse `inject-tests` rather than a new copy path | A guard that only checks a boolean "was terminal status reached" flag, without observing the real adapter invocation or the real evaluator_access event append, would pass a case where the placement bypassed `inject-tests` entirely. |
| tc049 | `bench/scripts/replay-stage-evidence.sh <bundle_dir>` invoked directly against a stored bundle fixture and a PATH-stubbed provider binary (internal — the replay script IS the entrypoint; PATH-stubbed provider is the one **permitted** mock, used only to prove zero invocations, not to fabricate a "successful rerun") | Real stored-bundle JSON, real digest recomputation, real filesystem diff of the named roots against the recorded lineage | Do not let the replay script actually invoke a real provider or a real worker process — REQ-F-013's whole claim is that it never needs to; the PATH-stubbed provider recording zero invocations is what proves this rather than assumes it | A replay implementation that silently reruns the worker "to be sure" and only reports drift it observed from the rerun (rather than from the stored evidence) would still "work" for the happy path but would violate REQ-F-013's "without rerunning the worker" claim, caught only by the stub-invocation-count assertion. |
| tc050 | `bench/scripts/verify-stage-evidence.sh <bundle_dir>` (stop-outcome/eligibility portion), same entrypoint pattern as tc044-tc046 (internal — no caller above the script exists yet) | Real fixture bundles, one per named stop outcome, with real partial-snapshot content | Do not represent "partial evidence retained" as a boolean test-setup flag; the fixture must contain fewer stage snapshots than a complete run would, and the guard must read and report on the ones present | A guard asserting eligibility purely from `stop_outcome`'s presence, without confirming partial snapshots are still readable, would pass even if an implementation silently deleted partial evidence on a stop. |
| tc051 | Every guard script invoked twice under network isolation, plus a static `grep` over the generic evidence scripts (internal — no caller above the scripts/grep exists) | Real subprocess execution with genuine network isolation (Linux `unshare --net` or the portable proxy-poison fallback, per F05's TC-039 precedent); real source grep | Do not simulate "offline" with a code-level flag; the environment itself must be offline. Do not scope the grep to a hand-picked line range | An implementation checking only a config flag for "offline mode" while still depending on a network fallback in production would pass a fake test while remaining unsafe; a narrowly-scoped grep would miss a forbidden token reintroduced outside the scanned range. |

## ISO 25010 coverage matrix

`N/A` cells are justified the same way F05's plan justified them: this is
offline curator/CI tooling with no production request path or end-user
journey (`uat-plan.md`'s "Not a product concern here"), consistent with
REQ-NF-003/004/005. Security coverage here is unusually dense relative to
F05 because F06's core purpose — evaluator-material isolation and fail-closed
usage identity — is itself a security-shaped property (a leak-surface and a
trust boundary), not an incidental one.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-042 | N/A | ✅ TC-042 (schema-version gate) | N/A | N/A | N/A | ✅ TC-042 (single-owner vocabulary read) | N/A |
| AC-002 | ✅ TC-042 | N/A | N/A | ✅ TC-042 (named offending root pair) | ✅ TC-042 | ✅ TC-042 (nested-root leak surface) | N/A | N/A |
| AC-003 | ✅ TC-042 | N/A | N/A | N/A | ✅ TC-042 (dual-oracle correctness) | N/A | ✅ TC-042 (oracle-shape distinction) | N/A |
| AC-004 | ✅ tc044 | ✅ tc044 (REQ-NF-007 category attribution) | N/A | ✅ tc044 (named category pair / residual magnitude) | ✅ tc044 | N/A | N/A | N/A |
| AC-005 | ✅ tc045 | N/A | N/A | ✅ tc045 (named missing field) | N/A | ✅ tc045 (`base_commit`-alone identity spoof) | N/A | N/A |
| AC-006 | ✅ tc046 | N/A | N/A | N/A | ✅ tc046 | ✅ tc046 (silent coercion trap) | ✅ tc046 (schema-level distinguishability) | N/A |
| AC-007 | ✅ tc047 | N/A | N/A | ✅ tc047 (named slot/path on drift) | ✅ tc047 | ✅ tc047 (upstream drift detection) | N/A | N/A |
| AC-008 | ✅ TC-042 | N/A | N/A | N/A | ✅ TC-042 | ✅ TC-042 (unmapped-provider fail-closed) | N/A | N/A |
| AC-009 | ✅ tc043 | ✅ tc043 (before-dispatch timing, via zero-invocation log) | N/A | ✅ tc043 (named root/path/source) | ✅ tc043 | ✅ tc043 (dual planted-leak locations) | N/A | N/A |
| AC-010 | ✅ tc043 | N/A | N/A | N/A | N/A | ✅ tc043 (dynamic name derivation, no hardcoded list) | ✅ tc043 | N/A |
| AC-011 | ✅ tc048 | N/A | N/A | N/A | ✅ tc048 | ✅ tc048 (ordering violation) | N/A | N/A |
| AC-012 | ✅ tc049 | N/A | N/A | ✅ tc049 (named drift kind/path/test id/artifact) | ✅ tc049 | N/A | N/A | N/A |
| AC-013 | ✅ tc049 | N/A | N/A | N/A | ✅ tc049 (immutability) | ✅ tc049 (tamper detection) | N/A | N/A |
| AC-014 | ✅ tc050 | N/A | N/A | N/A | ✅ tc050 (partial evidence retained) | ✅ tc050 (eligible-with-stop-outcome rejection) | N/A | N/A |
| AC-015 | ✅ TC-042 | N/A | N/A | ✅ TC-042 (named failing field per case) | N/A | N/A | N/A | N/A |
| AC-016 | ✅ tc051 | N/A | N/A | N/A | ✅ tc051 (byte-identical reruns) | ✅ tc051 (zero provider calls, offline) | N/A | ✅ tc051 (both isolation mechanisms, per F05 precedent) |
| AC-017 | ✅ TC-042 | N/A | ✅ TC-042 (submodule-absent CI state) | N/A | N/A | N/A | ✅ TC-042 (repo hygiene) | N/A |
| AC-018 | ✅ Diff review | N/A | N/A | N/A | ✅ Diff review (I-01/I-04 callers unaffected) | N/A | ✅ Diff review (frozen interface) | N/A |
| AC-019 | ✅ tc051 | N/A | N/A | N/A | N/A | ✅ tc051 (language-branch leak surface) | ✅ tc051 | N/A |
| AC-020 | ✅ TC-042 | N/A | N/A | N/A | N/A | N/A | ✅ TC-042 (bidirectional single-owner agreement) | N/A |
| AC-021 | ✅ tc047 | N/A | N/A | ✅ tc047 (named `envelope_source_unavailable`) | N/A | ✅ tc047 (whole-source drift, not misdiagnosed as per-field) | N/A | N/A |
| AC-022 | ✅ TC-042 | N/A | N/A | ✅ TC-042 (named missing/offending slot) | N/A | ✅ TC-042 (unverified-slot-as-required rejection) | N/A | N/A |

No coverage gaps: every non-`N/A` cell cites a TC or the named diff review;
every `N/A` cell is justified by this feature's offline-tooling,
no-user-journey nature or by the absence of a stated performance bound
(recorded explicitly for AC-004/AC-009's Performance cells rather than left
as an unexplained N/A, since those two ACs are the closest this feature
comes to a latency-shaped property).

## Observability design

Same posture as F05: no metrics/trace spans, because this is offline
curator/CI tooling with no production runtime (REQ-NF-001's "adds no service"
applies transitively — there is nothing running that could emit a metric).
Observability means the guards' own machine-readable stdout/exit status,
which F08/F09/F10 and a human curator both depend on. This is stated
per-behavior below as "internal — no observability (test-only script, no
production runtime)" is the correct justification for every row; the table
instead names *what* each script's terminal output carries, since that
output is the entire observability surface for this feature.

| Behavior | Log / stdout evidence | Trace/metric | Test assertion |
|---|---|---|---|
| Schema/bundle validation failure | `TestTC042_...` subtests report the exact failing field per REQ-F-016 case, not a generic "invalid" message | N/A — internal, test-only Go binary | TC-042 |
| Root-policy isolation violation | `verify-evidence-roots.sh` prints the offending root, path, and matched evaluator-only source; exit non-zero | N/A — internal, no production runtime | tc043 |
| Ledger reconciliation failure | `verify-stage-evidence.sh` prints the offending category pair or residual magnitude | N/A — internal | tc044 |
| Candidate identity failure | `verify-stage-evidence.sh` prints the missing `candidate` field or the two distinct-candidate digests | N/A — internal | tc045 |
| Artifact record verdict | `verify-stage-evidence.sh` prints `orphan` or `consumption_evidence_missing` per artifact, distinctly | N/A — internal | tc046 |
| Usage-mapping drift | `canary-usagemapping.sh` prints the drifted slot and envelope path (field drift) or `envelope_source_unavailable` and the transcript path (source drift) | N/A — internal | tc047 |
| Evaluator-access ordering violation | `verify-stage-evidence.sh` prints `isolation_violation` naming the phase and artifact | N/A — internal | tc048 |
| Replay drift | `replay-stage-evidence.sh` prints one of the four named drift kinds plus the specific path/test id/artifact | N/A — internal | tc049 |
| Snapshot mutation | `replay-stage-evidence.sh` prints `snapshot_mutated` naming the stage | N/A — internal | tc049 |
| Partial-evidence eligibility | `verify-stage-evidence.sh` prints `publication_eligible` and the full `ineligibility_reasons[]` list | N/A — internal | tc050 |
| Offline/determinism failure | Any guard fails naming the specific byte offset or field that differed between the two runs, not a generic "non-deterministic" message | N/A — internal | tc051 |

No new instrumentation beyond structured script/test output is required or
permitted, matching REQ-NF-001's "adds no service, no schema, no migration."

## Cross-feature contract tests (I-05, I-04)

### Produces: I-05

Carried verbatim from `spec.md`'s Cross-feature interactions section:

| I-## | Producer | Consumers | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-05 | E40-F06 | E40-F08, E40-F09, E40-F10 | `architecture.md#stage-evidence-and-isolation-contract` | `tests/contracts/e40_i05_stage_evidence_contract_test.go#TC-042` | TC-042 |

Gate mode: `contract-only`, staged by
`E40-interaction-map.md#i-05-staged-edge` — F06's producer role necessarily
runs before its consumers (F08/F09/F10, execution order 8-10) are
decomposed. Activation owner: E40-F08, E40-F09, E40-F10, each closing its own
consumption independently at its own UAT. Closure key: E40-F08 / E40-F09 /
E40-F10, respectively. Counterpart status: read live from Shark at
review/UAT time, not copied here as a fact that would go stale. Review
basis: this test-plan and `spec.md`, present together at F06 task_review.
Demonstrability disposition: `pending-integration` until each consumer's
live wiring closes.

**Judgment call on the I-05 contract-test pointer, recorded explicitly.**
The interaction map instructs F06's spec.md to "name the shared
contract-test pointer at specification time, the same way F05's spec.md
named TC-030 for I-04" — `spec.md` does this (§Cross-feature interactions,
"Shared contract test" row: `TC-042`). Because F06 is the *producer* side of
a staged edge whose consumers (E40-F08, E40-F09, E40-F10) have not yet been
decomposed into their own spec.md/test-plan.md, **TC-042 today has exactly
one owner and one caller: this feature's own Go contract test, reading only
this feature's own committed fixtures.** There is no fabricated shared
proof against unbuilt consumer code, and no attempt to write a "F08-side"
test in advance of F08's own spec. This test-plan asserts the same posture
`spec.md` itself states plainly: "E40-F08, E40-F09, and E40-F10 must copy
the shape source and the contract-test pointer above verbatim; the same test
proves every side of this contract and no twin test is created." When F08
is decomposed, its own test-plan.md must reuse `TC-042` verbatim as the
shape source for its runtime-writer obligations rather than writing a second
reader/writer validator — this plan records that obligation rather than
attempting to discharge it early on F08's behalf.

### Consumes: I-04

| I-## | Producer | Consumer | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-04 | E40-F05 | E40-F06 | `architecture.md#lifecycle-scenario-package-contract` | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` | TC-030 (existing, unmodified) |

F06's consumption slice is `evaluator_only`, `toolchain_identity`, and both
`stage_matrix` halves, assigned verbatim by
`E40-interaction-map.md#i-04-staged-edge` and `spec.md`'s own Consumes: I-04
row. F06 does not extend or re-run TC-030 — the non-regression obligation
instead is that every I-04 artifact REQ-NF-006 names is byte-unchanged by
this feature, verified by the AC-018 diff review, not by a second F06-owned
assertion over I-04's shape. tc043 (root-isolation) and the ledger/candidate
guards read `evaluator_only`, `toolchain_identity`, and `stage_matrix`
through real I-04 package files (e.g. `py-feature-recurring-tasks`), so
TC-030's own passing status is a real precondition for tc043 through tc051
running against a real package — this is stated as a dependency, not
duplicated as a second contract test. Gate mode: `contract-only`, staged by
`E40-interaction-map.md#i-04-staged-edge`; E40-F06 is the activation owner
for its own slice and closes it at its own UAT (UAT-09) with a real caller
chain (tc043's direct `verify-evidence-roots.sh` invocation against a real
package), shared-contract evidence (TC-030 passing), a production-path
integration test (tc043 against `py-feature-recurring-tasks`, the same
package F05's own AC-004 already exercises), and a wiring-removal
counterfactual (a guard that stopped reading `evaluator_only` from the real
package and instead hardcoded a name would fail AC-010's renamed-entry
case). Closure key: E40-F06, at its own UAT.

No twin test is created for I-04: TC-030 remains the single shared proof of
that contract, and F06's guards consume the artifacts TC-030 validates.

## Cross-epic integration tests (X-##)

### Consumes and validates: X-09 — Provider-usage field mapping

Carried verbatim from `spec.md`'s Cross-epic integrations section:

| Property | Contract |
|---|---|
| Producer epic / feature | E27 — Shark Status Viewer (E27-F15) |
| Consumer epic / feature | E40 — Shark Bench. E40-F06 is the contract owner; E40-F08 is the runtime writer |
| What F06 produces for it | `bench/evidence/usage-mapping.yaml` plus `canary-usagemapping.sh` |
| What F06 validates | The mapping resolves against a real capture; an unresolvable slot yields `usage_slot_unavailable` and an absent field, never a zero; an unmapped provider fails closed (AC-007, AC-008) |
| Test coverage | UAT-09, UAT-14; `canary-usagemapping.sh` driven by tc047 (AC-007, AC-021) and TC-042's fail-closed cases (AC-008, AC-022) |
| Deferral | None. No X-09 obligation is deferred to `docs/product/progress.md` |

**Judgment call on scope, recorded explicitly.** `spec.md` states F06
"produces, consumes, or validates **no other X-## row**." This test-plan
does not invent a contract test for X-07 (F02), X-08 (F04), X-10 (F07),
X-11/X-13 (F08), or X-12 (F09) — each belongs to its own owning feature's
test plan. The one adjacency worth naming: I-05 *records* a Shark-data
content digest field only when E40-F09 later supplies it under X-12; no
fixture bundle in this feature's own test data carries that field populated,
and TC-042's valid fixtures are not required to include it (it is not in
REQ-F-002/003/005/006/008/009's field inventory this feature owns).

## Integration scenarios

| Scenario | Boundary | Epic UAT contribution | Test evidence |
|---|---|---|---|
| Stage evidence bundle → future lifecycle consumers (I-05) | `bench/evidence/*` schema and `bundle.json`/`stages/*.json` shape → E40-F08/F09/F10's not-yet-built readers/writers | UAT-09 | TC-042 pins the shape those features must consume with no manual step |
| Three-root isolation → real dispatch boundary | `verify-evidence-roots.sh` → planted-leak fixture roots and a real I-04 package | UAT-09 ("prove references, answer keys, patches, and hidden tests absent") | tc043 |
| Ordered evaluator access → real `adapter.sh inject-tests` | `verify-stage-evidence.sh` + `adapter.sh inject-tests` → real post-terminal placement | UAT-09 ("grant recorded evaluator access") | tc048 |
| Time ledger → coordination-vs-work separation | `verify-stage-evidence.sh` → six-category interval reconciliation | UAT-16 | tc044 |
| Artifact use → orphan detection | `verify-stage-evidence.sh` → typed producer/consumer edges | UAT-18 | tc046 |
| Candidate identity → workflow-policy comparison | `verify-stage-evidence.sh` → `code`/`review` snapshot `candidate` block | UAT-19 | tc045 |
| Replay → re-evaluation without a worker rerun | `replay-stage-evidence.sh` → stored bundle | UAT-09, UAT-19 | tc049 |
| Provider-usage mapping → X-09 verification | `canary-usagemapping.sh` → real captured envelope fixtures | UAT-09, UAT-14 | tc047 |
| Fixture and I-05 tree → shark's own quality gate | `bench/evidence/*` tree → root `make fmt && make lint && make test` and `go list ./...` | Non-functional: repo hygiene, not a UAT scenario | TC-042 (repo-root gate; see AC-017) |
| Frozen I-01/I-04 interfaces → new sibling tooling | `bench/corpus/**`, `bench/scenarios/**` (unchanged) vs. `bench/evidence/**` (new) | Non-functional: no regression to Phase 1 or F05 (UAT-01, UAT-02, UAT-05-08 transitively) | AC-018 diff review |

Two verification-plan rows are intentionally **not** test cases, matching
`spec.md`'s own Verification plan table's stated method:

- **REQ-NF-005** (evidence tooling never touches the live shark database,
  `.sharkconfig.json`, or the live repository working tree, and never
  invokes shark project-initialisation commands) — verified by code review
  of `bench/scripts/{verify-evidence-roots,verify-stage-evidence,
  replay-stage-evidence,canary-usagemapping}.sh` confirming no such
  invocation exists, matching `spec.md`'s own "Diff review" method, not an
  automated test.
- **`bench/README.md`'s new "I-05 stage evidence and isolation contract"
  section** — this is prose describing an already-tested shape (TC-042,
  tc043-tc051 assert the real shape); the documentation itself is reviewed
  for accuracy against the shape, not independently tested, per the
  workflow's "Prompt-only changes" guidance applied to a docs-only delta.

## Test infrastructure

**Existing patterns to reuse:**
- `tests/contracts/e40_i04_scenario_contract_test.go` and
  `tests/contracts/e40_i01_corpus_contract_test.go` establish the
  repository-root-relative artifact-reading helper style (`filepath.Abs`,
  then `os.ReadFile`) and the `TestTC0NN_...` naming convention — TC-042
  follows this exactly, continuing the epic's sequential TC numbering (F05
  used TC-030-TC-041; the highest TC number in committed epic docs today is
  TC-041, per `bench/scripts/tests/tc041_predicate_argument_trace_test.sh`
  and `tests/contracts/e40_i04_scenario_contract_test.go`, so TC-042 is the
  next free slot, matching `spec.md`'s own explicit reference).
- `bench/scripts/tests/run-all.sh` and its `tcNNN_<description>_test.sh`
  naming convention (`tc003_clean_checkout_test.sh` … `tc041_predicate_
  argument_trace_test.sh`) is the pattern tc043 through tc051's bash test
  scripts follow, registered in the same `run-all.sh` — the only existing
  bench file this feature edits (REQ-F-017's table entry).
- `bench/scripts/canary-runsurface.sh`'s real-invocation-over-re-derivation
  discipline is the pattern `canary-usagemapping.sh` follows structurally,
  though it is a new script, not an extension of `canary-runsurface.sh`
  (ADR-F06-04's Rejected alternative (b) explicitly declines source-scraping
  `collect-run.sh`'s Python constants in favor of a shared-fixture coupling).
- `internal/runner/dispatcher.go`'s `DefaultDisallowedTools` is the pattern
  (not code) `verify-evidence-roots.sh` mirrors — nothing under
  `internal/runner` is imported or tested by this feature; it is cited here
  only because `spec.md`/`research-report.md` name it as the guard's design
  precedent (ADR-F06-03), not because F06 adds a test against it.

**New test infrastructure needed (this feature's own deliverables, already
named in `spec.md`'s component table, cross-checked against the AC test
matrix above with no unnamed gap remaining after the two notes recorded in
"Feature-level coverage check"):**
- `tests/contracts/e40_i05_stage_evidence_contract_test.go` — one Go file,
  `package contracts`, containing `TestTC042_I05StageEvidenceContract` and
  its table-driven valid/invalid-fixture subtests (AC-001-003, AC-008,
  AC-015, AC-017, AC-020, AC-022). Per REQ-NF-001, this is the **only** Go
  file this feature adds.
- `tests/contracts/testdata/e40_i05/{valid,invalid}/` — table-driven bundle
  fixtures. The 11 named REQ-F-016 rejection cases (AC-015) plus the root
  and completeness cases (AC-002, AC-003) plus the unmapped-provider and
  `verification_tier` cases (AC-008) plus the vocabulary-agreement cases
  (AC-020) plus the `required_identity_slots` cases (AC-022) — task
  decomposition must enumerate these as an explicit fixture-authoring task,
  the same posture F05's plan took for its own 14-row malformed-package
  table.
- `bench/scripts/testdata/evidence/` — bundle fixtures for the
  bench-script test cases: clean roots, each planted-leak case (tc043), each
  ledger case (tc044), each candidate-identity case (tc045), each
  artifact-record case (tc046), each access-ordering case (tc048), each
  replay-drift case (tc049), each of the ten stop-outcome cases (tc050).
- A copy of `py-feature-recurring-tasks` (or another admitted I-04 package)
  for tc043's renamed-`oracle_tests[]`-entry case (AC-010) — a scratch,
  test-time-mutated copy, the same "transient candidate" technique F05's
  TC-033 used for its own rejection-branch coverage, not a second committed
  fixture package.
- A PATH-stubbed dispatcher/provider binary reused across tc043, tc049, and
  tc051 to prove zero invocations — one shared shell helper under
  `bench/scripts/tests/` (e.g. `_stub_dispatcher.sh`, invocation-count
  logging to a file) rather than three independent implementations, to
  guarantee the "records zero invocations" assertion means the same thing
  in all three places.
- `bench/README.md`'s new "I-05 stage evidence and isolation contract
  (E40-F06)" section must name the exact Tier 2 curator invocation
  sequence (at minimum: run `verify-evidence-roots.sh` before every
  dispatch, `verify-stage-evidence.sh` after every stage, `canary-
  usagemapping.sh` on any usage-mapping change, `replay-stage-evidence.sh`
  for re-evaluation) so REQ-NF-004's reproducibility claim has one
  documented invocation, not an implied one — mirroring the role F05's
  README section played for its own Tier 2 sequence.

### Test infrastructure gaps

No unnamed script gap remains: unlike F05's own AC-020
(`verify-fixture-py-base.sh`, discovered missing from the component table
during this review), F06's `spec.md` component-changes table already names
every script this AC test matrix depends on (`verify-evidence-roots.sh`,
`verify-stage-evidence.sh`, `replay-stage-evidence.sh`,
`canary-usagemapping.sh`, and all nine `tc04[3-9]`/`tc05[01]` test wrappers).
The two notes recorded above (AC-017's absent dedicated hygiene script;
REQ-NF-007's untested latency claim) are not missing scripts — they are
scope decisions this plan makes explicit rather than silently assuming, and
neither blocks task decomposition:

- **AC-017 repo-hygiene verification.** Owner: not applicable — this plan's
  resolution (direct Makefile/`go list`/`go test` invocation, no dedicated
  wrapper) is final unless a later feature in this epic adds a new
  submodule or fixture directory, at which point that feature should add
  its own hygiene script, matching where F05 added TC-037.
- **REQ-NF-007 latency claim.** Owner: the E40-F06 spec/architecture owner,
  if a numeric threshold is ever wanted. Trigger: only if a future UAT or
  operator complaint surfaces the dispatch-boundary check as measurably
  slow in practice — until then, the qualitative code-review constraint
  (bounded filesystem walk, no network call) recorded in the AC-004 row
  above is this plan's authoritative resolution, and task decomposition may
  proceed from it without a fabricated performance test.

## Codex test-plan red-team (manual substitute)

**Verdict:** SKIPPED. Codex CLI is unavailable in this environment; a
manual, adversarial self-review was substituted in its place, applying the
same posture F05's plan's two-pass codex run used (find genuine coverage
gaps, mismatched case counts, unproven "distinct" assertions, and
open-ended robustness language), but performed by re-reading this plan
against `spec.md` line by line rather than by an independent tool
invocation. Findings are recorded honestly below rather than a fabricated
PASS.

**Findings from the manual pass:**

1. **AC-017 repo-hygiene verification has no dedicated `tcNNN` script**,
   unlike every other AC in this matrix and unlike F05's own AC-015/TC-037.
   Resolved above (Feature-level coverage check, item 1; Acceptance-criteria
   review, note 1): this is a legitimate scope difference (F06 adds no
   submodule or new module boundary), not an oversight, and is recorded as
   such rather than silently matched to a script that doesn't need to exist.
2. **REQ-NF-007's "without materially changing scenario wall time" has no
   latency-threshold test.** Resolved above (Feature-level coverage check,
   item 2; Acceptance-criteria review, note 2): `spec.md` states no numeric
   bound, so this plan does not invent one; the testable half (category
   attribution) is covered by AC-004/tc044, and the untestable half is
   recorded as a code-review constraint.
3. **AC-021's negative-case fixture (arbitrary non-JSON `---STDOUT---`
   text) must not be re-labeled as "the E27-F15 branch's actual output"
   anywhere in test code or fixture naming.** `spec.md`'s own text is
   explicit that this is a decoder-robustness assertion making no shape
   claim about any real or predicted upstream artifact (ADR-F06-04(c)'s
   prohibition on testing against an unobserved shape). This plan's AC-021
   row and Caller-Path Contract row both state this constraint explicitly
   rather than leaving it to be inferred by whoever authors the fixture —
   checked and confirmed present, not a new finding requiring a plan change.
4. **The I-05 contract-test pointer (TC-042) has exactly one real caller
   today** (this feature's own Go test), because F08/F09/F10 do not exist
   yet as committed code. This plan states that explicitly (Cross-feature
   contract tests §"Produces: I-05," "Judgment call") rather than
   fabricating a shared test that spans code that has not been written —
   checked and confirmed present, not a new finding requiring a plan change.
5. **No case in the AC test matrix asserts a negative on the *admission-time*
   check (REQ-F-010) independent of the *dispatch-boundary* check
   (REQ-F-011) beyond AC-010's dynamic-derivation case.** Re-reading
   REQ-F-010 and REQ-F-011 side by side: REQ-F-010 is "prove absent from a
   fresh checkout before any dispatch of that scenario" (a one-time,
   scenario-level admission gate) while REQ-F-011 is "fail the dispatch
   before any provider call, checked at every dispatch" (a per-dispatch
   guard). AC-009's four cases are explicitly framed as the dispatch-boundary
   check (REQ-F-011); AC-010 is explicitly framed as the admission-time
   check's *derivation* property (REQ-F-010), but does not independently
   assert admission catches a leak in a *fresh checkout that has never been
   dispatched at all* — only that its name-derivation is dynamic. This is a
   real, narrow gap: `spec.md`'s own AC-010 text states "the admission-time
   check... derives evaluator-only names from the I-04 package at call
   time," which is what this plan's tc043 second case (AC-010) tests, but
   neither AC-009 nor AC-010 as written in `spec.md` states a case where
   `verify-evidence-roots.sh` is run in its REQ-F-010 admission mode (fresh
   checkout, before *any* dispatch) with a planted leak and must fail. Given
   `spec.md`'s own component table describes
   `verify-evidence-roots.sh <package.yaml> <fixture_checkout>
   <scratch_project> <evaluator_root>` as one script serving both REQ-F-010
   and REQ-F-011 with the same underlying path-presence logic, and AC-009's
   four cases already exercise that logic exhaustively against both
   agent-visible roots, this plan treats AC-009's coverage as **sufficient
   for both call sites of the same underlying check** — the guard's logic
   does not change between an admission-time call and a dispatch-boundary
   call, only when it is invoked — and records this as a reasoned inference
   rather than silently assuming it. If task decomposition finds a behavioral
   difference between the two call sites (e.g., admission-time additionally
   checking test-identity absence via `adapter.sh test`'s normalized ids,
   which REQ-F-010's text mentions and REQ-F-011's does not), a dedicated
   admission-time-only negative case should be added to tc043 at that point;
   this plan flags the ambiguity rather than asserting false confidence that
   it is fully closed.

No finding above is blocking: findings 1-2 were already resolved with a
named method before this red-team pass; findings 3-4 confirm the plan
already states the required constraint; finding 5 is a genuine, narrow
scope question recorded for task decomposition's attention rather than
silently resolved either way.

## Recommendations

- [x] Ready for development — every AC in `spec.md` has a named test case,
  technique, ISO 25010 row, and caller-path contract. Codex CLI was
  unavailable, so a manual adversarial self-review substituted for the
  automated red-team pass; its five findings are recorded above with none
  blocking. The two genuine coverage notes found (AC-017's absent dedicated
  hygiene script; REQ-NF-007's untested latency claim) are resolved with an
  explicit, non-fabricated method rather than left silent, matching the
  posture F05's plan used for its own AC-020 gap. The one open scope
  question (finding 5 above: whether REQ-F-010's admission-time call site
  needs a dedicated negative case beyond AC-009's dispatch-boundary
  coverage) is flagged for task decomposition's attention, not treated as
  either resolved or blocking, because `spec.md` names one guard script
  serving both call sites and this plan's inference that shared logic needs
  shared coverage is reasonable but not proven by a `spec.md` statement.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.
