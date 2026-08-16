# Test Plan: E40-F07 - Replayable product-design prelude

**Created:** 2026-08-15
**Feature PRD:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/feature.md`
**Task Spec:** Not yet decomposed. This plan is written directly against
`spec.md` (the same posture F05's and F06's test-plan.md used before task
decomposition — Step 1-3 use `spec.md` as both the incremental spec and the
traceability target).
**Status:** APPROVED, with two named, non-blocking coverage notes recorded
below (AC-017's repo-hygiene assertion names no dedicated `tcNNN` script,
same posture F06's plan recorded for its own AC-017; and AC-019 has no REQ
entry in `spec.md`'s own Verification plan table, a spec-side traceability
gap this plan records rather than silently working around) — see
"Acceptance-criteria review" and "Codex test-plan red-team (manual
substitute)."

## Scope and drift analysis

`spec.md` is incremental over `feature.md` and
`architecture.md#product-design-replay-contract`, both already fixed by the
epic before this feature's own spec was written. Comparing `spec.md` against
`feature.md`:

- Every `spec.md` REQ-F-* and REQ-NF-* traces to a `feature.md` Scope,
  Acceptance boundary, or Contracts line: "wrap, don't fork" (REQ-NF-006(a))
  → Scope bullet 1; the versioned replay bundle (REQ-F-001, REQ-F-002) →
  Scope bullet 2; routing human questions/research through the adapter and
  disabling live input (REQ-F-004, REQ-F-005, REQ-F-006, REQ-F-007) → Scope
  bullet 3; per-stage response/artifact/digest lineage (REQ-F-003, REQ-F-009,
  REQ-F-010) → Scope bullet 4; the replayed-interaction-proxy field set
  (REQ-F-011) and its 2026-08-13 amendment → Scope bullet 5; downstream
  artifact-consumption edges (REQ-F-010's `edge_kind`) → Scope bullet 6;
  `unresolved_gate` (REQ-F-008) → Scope bullet 7. No REQ introduces a
  capability absent from `feature.md`'s Scope.
- Every `feature.md` Acceptance boundary bullet is covered: "two runs consume
  the same response sequence and produce complete lineage" → AC-004, AC-005;
  "a scored run cannot reach live research or an interactive question
  surface" → AC-003; "missing replay input stops the scenario and never
  invents an answer" → AC-004, AC-006; "non-feature scenarios bypass the
  prelude and retain explicit non-applicable stage records" → AC-010;
  "replaying the same bundle reproduces interaction counts and artifact
  consumption edges, and the report cannot label them as human minutes" →
  AC-008, AC-016. No boundary bullet is left uncovered.
- `feature.md`'s Contracts section ("Consumes I-04: use the feature
  scenario's stage matrix and replay references"; "Consumes X-10: invoke the
  existing E36-F02 product-design action … rather than defining a second
  product-design flow"; "Produces I-06 … consumed by E40-F08") matches
  `spec.md`'s Cross-feature interactions and Cross-epic integrations sections
  verbatim — no semantic drift in what is consumed vs. produced. The
  2026-08-13 amendment ("does not convert scripted responses into a claim
  about real stakeholder effort") matches `spec.md`'s REQ-F-011/ADR-F07-06
  human-time prohibition exactly.
- `spec.md`'s own "Durable unresolved decisions" section resolves all five
  candidates with a materiality argument (provider-CLI denial-mechanism
  coverage is non-material because the gate is observational; resolver
  routing compliance is non-material because ledger reconciliation converts
  non-compliance into a named failure; the `replay_wait_ns` ceiling is a
  non-material tuning constant, matching F06's reconciliation-epsilon
  disposition; future-bundle location is already settled by I-04; the
  `evaluator_only`-relocation alternative is closed as incorrect, not
  pending). No new Q### is warranted by this review.

**No drift found.** No BA or architecture refinement is required.

### Feature-level coverage check (component-changes table vs. AC/REQ)

`spec.md`'s Architecture "Component changes" table lists 15 new/modified
artifacts (Go and non-Go). Cross-referencing each against the AC list:

| Component | Owning REQ/AC |
|---|---|
| `bench/replay/i06-schema.yaml` | REQ-F-018, AC-001, AC-015 |
| `bench/replay/live-egress-tools.yaml` | REQ-F-004, AC-003 |
| `bench/replay/preamble.md` | REQ-F-016, AC-012 |
| `bench/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json` | REQ-NF-006(c) carve-out; fixture support for AC-004-AC-012 |
| `bench/scripts/replay-answer.sh` | REQ-F-006, REQ-F-007, AC-004, AC-005 |
| `bench/scripts/run-prelude.sh` | REQ-F-004, REQ-F-013, REQ-F-014, REQ-F-015, REQ-F-016; AC-003(a), AC-010, AC-011, AC-012 (REQ-F-012 is `verify-replay-isolation.sh`'s row below, invoked by the caller's dispatch loop, not by `run-prelude.sh` itself) |
| `bench/scripts/verify-replay-result.sh` | REQ-F-009; AC-006 (REQ-F-003/010/011/013/017 and AC-002/007/008/013 are TC-052's row below — this script implements only REQ-F-009's lineage reconciliation) |
| `bench/scripts/verify-replay-isolation.sh` | REQ-F-005, REQ-F-012; AC-003(b), AC-009 |
| `bench/scripts/testdata/replay/` | fixture support for AC-003-AC-012, AC-016 |
| `bench/scripts/tests/tc053_live_egress_denial_test.sh` | AC-003 |
| `bench/scripts/tests/tc054_replay_resolver_test.sh` | AC-004, AC-005 |
| `bench/scripts/tests/tc055_lineage_reconciliation_test.sh` | AC-006 |
| `bench/scripts/tests/tc056_bundle_disclosure_test.sh` | AC-009 |
| `bench/scripts/tests/tc057_non_applicable_record_test.sh` | AC-010, AC-011 |
| `bench/scripts/tests/tc058_prelude_placement_test.sh` | AC-012 |
| `bench/scripts/tests/tc059_replay_offline_determinism_test.sh` | AC-016, AC-019 |
| `bench/scripts/tests/run-all.sh` | registration only, no independent AC |
| `bench/README.md` | documentation — reviewed, not executed (see "Integration scenarios") |
| `tests/contracts/e40_i06_product_design_replay_contract_test.go` (TC-052) | AC-001, AC-002, AC-007, AC-008, AC-013, AC-014, AC-015, AC-017 |
| `tests/contracts/testdata/e40_i06/{valid,invalid}/` | AC-001, AC-002, AC-007, AC-008, AC-013, AC-014, AC-015 |

**One gap found, non-blocking, the same shape F06's plan recorded for its own
AC-017:**

1. **AC-017's repo-hygiene assertion (`make fmt && make lint && make test`
   green, `go list ./...` clean, TC-052 passing without a populated
   submodule) names no dedicated `tcNNN` script** in `spec.md`'s component
   table, unlike F05's own AC-015/TC-037. This is materially smaller than
   F05's check for the same reason F06's was: F07 adds no new submodule and
   no new Go module boundary — its only Go addition is one file already
   exercised, and required to pass, by `make test` itself (`go test ./...`
   includes `tests/contracts/e40_i06_product_design_replay_contract_test.go`).
   A dedicated `tc060_repo_hygiene_test.sh` mirroring TC-037's
   populated/unpopulated submodule comparison would assert nothing F07
   uniquely puts at risk, because F07 has no submodule state to compare (its
   fixture and script trees are plain files and bash, not a separate Go
   module or a language runtime needing population). The verification method
   below (repo-root Makefile invocation, `go list ./...` grep, both run
   directly rather than through a dedicated wrapper script) is therefore
   recorded as this plan's resolution rather than as a missing deliverable;
   task decomposition may proceed from it. If a future feature in this epic
   adds a new submodule or fixture directory that risks entering `./...`,
   that feature should add the dedicated hygiene script, matching where F05
   added TC-037.

This is why the plan below reaches **APPROVED** rather than
**NEEDS_REFINEMENT**: every AC has a test with a concrete oracle, and both
named coverage notes (here, and AC-019's REQ-traceability gap recorded in
"Acceptance-criteria review") are resolved with an explicit, non-fabricated
method rather than left silent.

## Test tiers

Mirrors F06's tiering rationale, adapted: F07 needs no fixture submodule at
all for its Tier-1 validator (REQ-NF-003 — `bench/replay/**` and
`tests/contracts/testdata/e40_i06/**` are committed, in-repo artifacts), but
its execution-based guards (resolver, isolation, non-applicable, placement,
determinism) genuinely need real filesystem roots and, for the placement
guard specifically, a **live scratch Shark project** — the one new
infrastructure need this feature introduces that neither F05 nor F06 had,
stood up only through `scripts/shark-scratch-env.sh` (REQ-NF-005).

| Tier | Runs | Needs submodule? | Needs live scratch Shark project? | Where |
|---|---|---|---|---|
| **Tier 1** | `make test` (CI + every dev machine) | No — reads only committed I-06 schema, bundle carve-out, and fixture files under `bench/replay/**` and `tests/contracts/testdata/e40_i06/**` | No | `tests/contracts/e40_i06_product_design_replay_contract_test.go` (TC-052) |
| **Tier 1b** | Curator, manually or via `bench/scripts/tests/run-all.sh` | Some cases yes (resolver, isolation, non-applicable, and lineage cases run against committed bundle/result fixtures with no live checkout); tc058 specifically needs a real `py-feature-recurring-tasks` fixture checkout for `run-prelude.sh`'s REQ-F-014 consistency check | tc058 only, via `scripts/shark-scratch-env.sh` | `bench/scripts/tests/tc053_*.sh` through `tc059_*.sh` |
| **Tier 2** | Curator, at replay-schema or bundle-carve-out change time | Yes for tc058 (placement guard against a real fixture checkout and a real scratch project) | Yes, for tc058 | `bench/scripts/{run-prelude,verify-replay-result,verify-replay-isolation}.sh` against real roots |

Tier 2 is what REQ-NF-004's "byte-identical verdicts … at an unchanged
bundle, fixture SHA, and toolchain identity" exercises for the execution
guards. It is **not** gated by root `make test`; `bench/README.md`'s new
"I-06 product-design replay contract" section must name the exact Tier 2
invocation sequence (matching F05's and F06's own precedent) so "curator
re-runs it" is a real, documented action.

## Determinism and offline boundary (REQ-NF-004, AC-016)

Same class of incidental non-determinism F01, F05, and F06 all had to name
explicitly, now applied to the replay resolver, guards, and the prelude
adapter itself rather than to admission or evidence:

- No guard's or the resolver's verdict may depend on wall-clock time beyond
  what a fixture already records. `replay_wait_ns` is measured, not modelled
  (REQ-F-011), and AC-016's two-run comparison would catch a resolver that
  padded its own latency with a sleep or a clock read folded into output.
- `verify-replay-result.sh`'s and `verify-replay-isolation.sh`'s output
  (verdicts, named failing fields) must be emitted in a fixed order — by
  stage, then by entry ordinal within a stage — so two runs' full output is
  byte-comparable, the same discipline F05's `admit-scenario.sh` and F06's
  `verify-stage-evidence.sh` used.
- `run-prelude.sh` is the single provider-calling path in this feature
  (REQ-NF-004) and is exercised by no test through a real provider; every
  test in this feature drives it (or the scripts it delegates to) against a
  PATH-stubbed dispatcher binary that records its invocations, so
  "zero provider calls" is observed rather than assumed, the same discipline
  F06's TC-043/TC-049 used.
- TC-059 provisions the network-disabled state the same way F05's TC-039 and
  F06's TC-051 did (Linux `unshare --net`, or the portable
  `GOPROXY=off`/poisoned-proxy fallback), because every guard, the resolver,
  and `run-prelude.sh`'s non-dispatch paths are pure filesystem and digest
  computation and must never depend on the network at all.

## AC test matrix

| AC | Requirement(s) | Tier | Technique | Test case | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|---|
| AC-001 | REQ-F-001, REQ-F-002, REQ-F-010, REQ-F-011, REQ-F-017 | 1 | Contract-surface enumeration + decision table (bundle vs. result document kind) | TC-052 | `TestTC052_I06ProductDesignReplayContract` reads every fixture bundle and result under `tests/contracts/testdata/e40_i06/valid/` via `os.ReadFile` and validates against `bench/replay/i06-schema.yaml`'s field inventory and closed vocabularies; a fixture supplies a result document where a bundle is expected, and the reverse | Every REQ-F-001/002/010/011/017 field is present and well-typed in every valid fixture; `schema_version` matches the version the validator supports; every closed-vocabulary value used (`stage`, `request_kind`, `artifact_type`, `edge_kind`, `terminal_outcome`, error kind) resolves against `i06-schema.yaml`; the bundle and the result are recognized as distinct document kinds. Negative: a result document supplied where a bundle is expected (or the reverse) is rejected naming the expected kind, not silently half-validated against the wrong shape. |
| AC-002 | REQ-F-003 | 1 | Boundary value analysis (one-byte mutation) + attack-class enumeration (join-key spoof) | TC-052 | Table-driven subtests: (i) every entry's `entry_digest` recomputed from the stored bundle fixture; (ii) the same bundle with exactly one byte of one entry field changed; (iii) a result fixture whose `stages[].consumed_entries[].entry_digest` values are checked as a subset of the bundle's recomputed digest set, plus a fourth fixture where one consumed-entry digest is **not** in that set | (i) recomputed digest equals the stored `entry_digest` for every entry. (ii) recomputation yields a different digest and the validator reports `replay_bundle_mutated` naming the entry. (iii) the subset check passes for the valid result; the fourth fixture is rejected naming the offending `entry_digest`. Negative: a validator that trusts the result's recorded `entry_digest` values without recomputing from the bundle would accept case (ii)'s mutated bundle silently — TC-052 asserts recomputation, not trust. |
| AC-003 | REQ-F-004, REQ-F-005 | 1b/2 | Decision table (structural vs. observational, evaluated independently) + attack-class enumeration (missing denial-argument member, planted live-egress transcript record) | `bench/scripts/tests/tc053_live_egress_denial_test.sh` | (a) Structural: `run-prelude.sh`'s constructed argument vector captured via a PATH-stubbed dispatcher that records its argv, run once against `live-egress-tools.yaml` as committed and once against a copy with one member removed, plus one case where the stubbed argv is missing a member entirely. (b) Observational: `verify-replay-isolation.sh` against a retained-transcript fixture containing one `WebSearch` tool-use record, and a second, clean transcript fixture | (a) The argv contains one denial argument per set member; removing a member from the file changes the argv with no script edit; an argv missing a member fails before dispatch, naming the missing member. (b) The transcript with a `WebSearch` record yields `live_interaction_reached` naming the tool and stage; the clean transcript passes. Negative: an implementation that treats the denial-argument construction alone as proof (skipping the transcript scan) would pass a transcript that actually contains a live tool-use record — TC-053's case (b) requires the observational half to independently catch what the structural half cannot verify (whether the provider actually honored the denial). |
| AC-004 | REQ-F-006, REQ-F-007, REQ-F-008 | 1b | Decision table (3 named outcomes: supply, `unresolved_gate`, `replay_desync`) + attack-class enumeration (no nearest/partial match) | `bench/scripts/tests/tc054_replay_resolver_test.sh` | `replay-answer.sh --bundle <path> --stage <D0X> --kind <kind> --topic <key>` against a fixture bundle: (i) a matching call at the lowest unconsumed ordinal; (ii) the same stage/kind/topic called again after that stage's entries are exhausted; (iii) a call whose `--topic` disagrees with the entry at the current ordinal; (iv) a call whose `--topic` is a near-miss (e.g. differing by one character) of an unconsumed entry's topic, asserting no fuzzy match is attempted | (i) supplies the lowest unconsumed ordinal's response and appends exactly one consumption record. (ii) fails `unresolved_gate` naming stage, kind, and topic — the resolver never invents, paraphrases, or degrades to a default answer (REQ-F-008). (iii) fails `replay_desync` naming both the expected and supplied topic. (iv) fails `replay_desync` or `unresolved_gate` — never a supplied response — proving no nearest-match path exists. Negative: an implementation that falls back to "closest topic" when no exact match exists would supply a response for case (iv), which is exactly the fabrication path REQ-F-007/REQ-F-008 forbid and this case is designed to catch. |
| AC-005 | REQ-F-007 | 1b | State transition (two-pass byte identity) | `bench/scripts/tests/tc054_replay_resolver_test.sh` | Two consecutive resolver-driven passes over the same bundle and the same recorded call sequence (same stage/kind/topic calls in the same order) | Both passes produce byte-identical supplied response sequences and byte-identical consumption ledgers (same entry ids, same order, same digests). Negative: a resolver that consumes entries by first-match-in-file-order rather than lowest-unconsumed-ordinal would diverge between passes if two entries in the same stage share a topic but not an ordinal — TC-054's reproducibility case is run against a bundle constructed to have that ambiguity, so ordinal-primacy is actually exercised, not assumed absent from the fixture. |
| AC-006 | REQ-F-009 | 1b | Decision table (2 distinct verdicts) + attack-class enumeration (fabricated/bypassed lineage) | `bench/scripts/tests/tc055_lineage_reconciliation_test.sh` | `verify-replay-result.sh` against three result fixtures: (i) an artifact record claiming a `consumed_entries` reference absent from the resolver's own ledger; (ii) a stage producing an artifact having consumed zero `required: true` entries; (iii) a run that stopped because no unconsumed entry remained for a stage (a genuine `unresolved_gate` case, with no fabricated consumption claim) | (i) rejected as `unattributed_artifact` naming the entry. (ii) rejected as `unattributed_artifact` naming the stage. (iii) reported as `unresolved_gate` — never `unattributed_artifact`, proving the two verdicts are genuinely distinguishable and not one masquerading as the other. Negative: a validator that collapses "consumed nothing required" and "no entry was available" into one generic "incomplete lineage" verdict would pass (i)/(ii)/(iii) with an identical message, defeating downstream E40-F10's need to distinguish "corpus incomplete" from "session bypassed the resolver" — TC-055 asserts the three cases produce three distinguishable outcomes. |
| AC-007 | REQ-F-010 | 1 | Equivalence partitioning (`consumers: []` vs. absent `consumers`) | TC-052 | `TestTC052_...` against one result fixture whose `stages[].artifacts[]` contains one entry with `consumers: []` and one entry with the `consumers` key entirely omitted, plus a third entry carrying one `{consuming_stage, edge_kind: "referenced", observed_at}` edge from a later stage | The empty-array entry yields verdict `orphan`; the absent-key entry yields verdict `consumption_evidence_missing`; neither verdict is produced for the other entry. The third entry's downstream edge is recorded with its `edge_kind` intact and readable. Negative: a decoder that defaults an absent key to an empty slice (a common `omitempty`/zero-value trap, the same one F06's TC-046 was designed to catch for I-05) would collapse the two verdicts into one — TC-052 fails loudly on exactly that collapse. |
| AC-008 | REQ-F-008, REQ-F-011, REQ-NF-007 (`replay_wait_category` branch) | 1 | Decision table (closed field set × discriminator × human-time-name prohibition) + boundary value analysis (`replay_wait_ns` plausibility ceiling) | TC-052 | Table-driven subtests: (a) a proxy block missing `measurement_kind`; (b) `measurement_kind` set to a value other than `replayed_interaction_proxy`; (c) a proxy block carrying a field outside the closed set (e.g. `human_minutes`); (d) a valid proxy block asserting `replay_wait_category` equals `replay_or_human_gate_wait` (REQ-NF-007); (e) a proxy block whose `replay_wait_ns` exceeds a declared plausibility ceiling for a local file read, that ceiling owned by `bench/replay/i06-schema.yaml` per REQ-F-018's single-owner rule rather than hardcoded in the validator, so AC-015's bidirectional check covers it too; (f) a result fixture whose `terminal_outcome` is `unresolved_gate` and whose `unresolved_gate_count` equals the number of unresolved-gate stage events recorded in `stages[]`; (g) a fixture pairing `terminal_outcome: unresolved_gate` with `unresolved_gate_count: 0` | (a)/(b)/(c) rejected naming the missing/offending field. (d) accepted, category confirmed. (e) rejected as a synthesized-delay case. (f) accepted, counter value confirmed to match the recorded gate events. (g) rejected naming the zero-count contradiction. Negative: a proxy field named `stakeholder_minutes` or `cognitive_effort_estimate` (rather than a numeric field alone) is rejected specifically because its **name** expresses human-attributed duration/effort, not merely because it is unlisted. A second negative, closing REQ-F-008's counter obligation: a validator that checks only the proxy block's field *inventory* would accept a run that stopped on a missing entry while reporting `unresolved_gate_count: 0`, making G12's gate-count reporting silently wrong for E40-F10 — case (g) exists to catch exactly that. |
| AC-009 | REQ-F-012 | 1b/2 | Attack-class enumeration (planted-leak locations, exhaustively: fixture checkout, scratch project, clean-roots control, pre-dispatch timing) | `bench/scripts/tests/tc056_bundle_disclosure_test.sh` | `verify-replay-isolation.sh <bundle_path> <fixture_checkout> <scratch_project>`, run four times: (a) the replay bundle planted inside the fixture checkout; (b) a copy of the bundle under a different filename but the same content digest planted inside the scratch Shark project; (c) both roots left clean while the resolver still supplies one entry to the working directory, in place, one entry at a time; (d) case (a) repeated with a PATH-stubbed dispatcher recording every invocation, verifying the log is empty after the guard's failing exit | (a) fails `bundle_bulk_disclosure` naming the fixture-checkout root and the exact path. (b) fails the same way but naming the scratch-project root and the digest-matched copy — proving a guard comparing only filenames cannot pass this case. (c) exits `0` — a single supplied response present in the working directory is not bulk disclosure. (d) fails exactly as (a) does, **and** the stubbed dispatcher log contains zero invocations. Negative: a guard that only checks for the bundle's exact filename (rather than content digest) would miss case (b)'s renamed copy — TC-056 requires the digest-matched renamed copy to independently fail. |
| AC-010 | REQ-F-013 | 1b | Decision table (4 cases: never-invoked, result-written, reason-verbatim, missing-result-is-failure) + attack-class enumeration (silent absence) | `bench/scripts/tests/tc057_non_applicable_record_test.sh` | `run-prelude.sh` against each of the three non-feature seed packages (all `prelude.D01`-`.D05` entries `applicable: false`): (i) a PATH-stubbed dispatcher records invocations across the run; (ii) the written replay result is checked for `terminal_outcome: not_applicable`; (iii) each of D01-D05's `reason` string is diffed byte-for-byte against the package's own `stage_matrix.prelude.D0X.reason`; (iv) a fourth case runs `verify-replay-result.sh` against a directory where the expected result file is absent entirely | (i) zero dispatcher invocations for all three packages. (ii)/(iii) accepted, reasons match verbatim. (iv) rejected as a named failure — an absent result is not treated as "nothing to check," matching REQ-F-013's "explicit non-applicable record is the deliverable." Negative: an implementation that silently exits `0` with no result file when every prelude stage is non-applicable (treating "nothing to do" as success) is exactly the defect case (iv) exists to catch. |
| AC-011 | REQ-F-014 | 1b/2 | Decision table (pass × 3 named violations) | `bench/scripts/tests/tc057_non_applicable_record_test.sh` | `run-prelude.sh`'s pre-dispatch consistency check against: (i) the real feature package, unmodified; (ii) a scratch copy of the feature package with `replay_reference` removed; (iii) a scratch copy of a non-feature (bug) package with a `replay_reference` field added; (iv) a scratch copy of the feature package whose bundle fixture's `scenario_binding.scenario_id` disagrees with the package's own `scenario_id` | (i) passes. (ii) rejected naming the missing `replay_reference` field. (iii) rejected naming the unexpected `replay_reference` field on a package with no applicable prelude stage. (iv) rejected naming both the package's and the bundle's disagreeing scenario ids. No file under `bench/scenarios/` is modified by the test — every mutated case operates on a scratch copy created by the test itself. Negative: an implementation that checks only "does `replay_reference` exist" without cross-checking `scenario_binding.scenario_id` would pass case (iv), which is the exact cross-package mismatch this case exists to catch. |
| AC-012 | REQ-F-015, REQ-F-016 | 1b/2 | Contract-surface enumeration (5 independent placement/identity assertions) + state transition (preamble prepended before dispatch) | `bench/scripts/tests/tc058_prelude_placement_test.sh` | `run-prelude.sh` against a scratch Shark project created by `scripts/shark-scratch-env.sh`, with a PATH-stubbed dispatcher recording the full constructed prompt and working directory: (i) the recorded dispatch working directory; (ii) the written result's `artifact_root.path`/`identity_digest`; (iii) a filesystem walk of the fixture checkout and evaluator-only root for any D01-D05 artifact or `docs/product/progress.md` write; (iv) the recorded prompt's content compared against `bench/replay/preamble.md`'s bytes, and the result's `preamble_digest` compared against that file's `sha256`; (v) the produced artifact filenames compared against the bundle's `D0X-*.md` Output Standard | (i) equals `roots.scratch_shark_project`. (ii) the result records that root's real path and a digest that is recomputable. (iii) zero D01-D05 or `progress.md` writes found in either off-limits root. (iv) the recorded prompt contains `preamble.md`'s content verbatim and `preamble_digest` matches. (v) every produced filename matches the `D0X-*.md` pattern. Negative: an implementation that computes `preamble_digest` from its own in-memory string constant rather than reading `bench/replay/preamble.md` at dispatch time would pass today but silently diverge the moment the file changes without the constant being updated — TC-058 reads the real file, not a constant, to catch that drift. |
| AC-013 | REQ-F-008, REQ-F-017 | 1 | Decision table (12 named stop/terminal outcomes × `i07_stop_mapping`) + attack-class enumeration (eligible-with-stop-outcome rejection) | TC-052 | Table-driven subtests over twelve result fixtures, one per `terminal_outcome` value; for each of the four I-06-local diagnostics (`replay_desync`, `live_interaction_reached`, `unattributed_artifact`, `bundle_bulk_disclosure`) a companion fixture checks `i07_stop_mapping`; plus one fixture pairing a stop outcome with `publication_eligible: true`, one fixture with `i07_stop_mapping` absent on a stop outcome, and one fixture whose `i07_stop_mapping` names a value outside I-07's own stop vocabulary | Every non-`complete`/`not_applicable` fixture retains partial evidence, `publication_eligible: false`, non-empty `ineligibility_reasons[]`. `replay_desync` maps to `unresolved_gate`; the other three I-06-local diagnostics map to `error`; the eight remaining values map to themselves. The eligible-with-stop-outcome fixture is rejected naming the contradiction. The absent-`i07_stop_mapping` fixture is rejected naming the field. The out-of-vocabulary-mapping fixture is rejected naming the offending value. Negative: an implementation that widens I-07's own stop vocabulary by passing an I-06-local value straight through (rather than mapping it) is exactly the defect the out-of-vocabulary case catches. |
| AC-014 | REQ-F-019 | 1 | Decision table (14 named malformed-field rejection fixtures, one per case) | TC-052 | Table-driven subtests over `tests/contracts/testdata/e40_i06/invalid/`, one **fixture** per REQ-F-019 case, 14 fixtures total: unsupported `schema_version`; duplicate `ordinal` within a stage; unknown `stage`; unknown `request_kind`; unknown `edge_kind`; unknown `terminal_outcome`; unknown error kind (five separate fixtures, one per closed vocabulary — not one shared "enum-family" fixture, so a validator that only checks one vocabulary correctly cannot pass by accident on the other four); an entry whose `entry_digest` does not recompute; a response `{path, digest}` that does not resolve or does not match; a proxy block with an extra field or a non-discriminator `measurement_kind` (two fixtures); an artifact record missing `digest` or producer-stage identity (two fixtures); a stop outcome paired with `publication_eligible: true`; a `not_applicable` result missing a per-stage `reason`; a consumption claim absent from the resolver ledger | Each of the 14 named fixtures exits non-zero with the failing field named in the error, matching that fixture's own defect, not a generic "invalid bundle/result" message. Negative: a fixture correcting exactly one field passes, proving the validator is not rejecting the whole file for an unrelated reason; the five separate vocabulary fixtures also prove no single "any enum wrong" catch-all is standing in for per-vocabulary rejection. |
| AC-015 | REQ-F-018 | 1 | Contract-surface enumeration (single-owner vocabulary agreement, bidirectional) | TC-052 | `TestTC052_...` cross-checks that every closed-vocabulary value the validator accepts appears in `bench/replay/i06-schema.yaml`, plus a fixture whose bundle/result uses a vocabulary value present in `i06-schema.yaml` but absent from any committed fixture, and a fixture using a value present in neither | The schema-only value surfaces as a named, non-fatal "declared but unexercised" note, while the value present in neither is rejected as an unknown vocabulary entry. Negative: a validator embedding its own private copy of the vocabulary (rather than reading `i06-schema.yaml`) would silently diverge if the YAML file changed without the Go code changing — TC-052 reads the vocabulary from the YAML file at test time specifically to catch that divergence. |
| AC-016 | REQ-NF-004 | 1b/2 | State transition (repeated-run byte identity, network-disabled) | `bench/scripts/tests/tc059_replay_offline_determinism_test.sh` | Every guard, the resolver, and `run-prelude.sh`'s non-dispatch paths (`replay-answer.sh`, `verify-replay-result.sh`, `verify-replay-isolation.sh`, `run-prelude.sh`'s pre-dispatch checks) invoked twice, back to back, over the same bundle/roots, with the network disabled per the "Determinism and offline boundary" section above and a PATH-stubbed provider recording invocations | Both invocations of every guard/script produce byte-identical stdout/exit code, and the stubbed-provider invocation log is empty after every guard and pre-dispatch check completes. Negative: a script reading wall-clock time or directory-listing order into its output would fail the byte-identity comparison; a script silently attempting a network call would either hang (caught by a timeout) or fail under the isolation mechanism. |
| AC-017 | REQ-NF-002, REQ-NF-003 | 1 | Boundary/state enumeration (submodule-absent CI state vs. this checkout's live state) | TC-052 (repo-root Makefile gate; no dedicated `tcNNN` script — see "Feature-level coverage check" gap note above) | Repo root `make fmt && make lint && make test`, run against this checkout's live state; separately, `go test ./tests/contracts/...` run against an isolated copy of the tree with no fixture submodule initialized at all (CI-like, matching `actions/checkout@v4`'s default), confirming `tests/contracts/e40_i06_product_design_replay_contract_test.go` needs no submodule; `go list ./...` checked for any replay-fixture package | `make fmt && make lint && make test` is green; `go test ./tests/contracts/...` passes identically with no submodule present; `go list ./...` lists no `bench/replay`, `bench/scripts/testdata/replay`, or evaluator-only package (none of these are Go packages to begin with, so this is a structural, not incidental, guarantee); `.github/workflows/ci.yml` is confirmed byte-unchanged (folded into AC-018's diff review). Negative: a future change accidentally adding a `.go` file under `bench/scripts/testdata/replay/` would surface here as a new `go list` entry. |
| AC-018 | REQ-NF-001, REQ-NF-006 | N/A (diff review, not an automated TC) | Attack-class enumeration (frozen-interface / non-regression, reviewed not executed) | Diff review | `git diff` against `skills/shark-rider/verbs/product-design.md`, every file under `internal/sharkdata/default_data/skills/product-design/**`, every Phase 1 / I-04 / I-05 file REQ-NF-006(b)/(c)/(d) names, `package.yaml` (unchanged while `evaluator/replay/reference-bundle.json` is the single carve-out written), and a repo-wide check that no file under `internal/` or `cmd/` is touched | Every named file's diff is empty (byte-unchanged) except the one declared `reference-bundle.json` carve-out; no file under `internal/` or `cmd/` appears in the feature's changeset. This is the same non-automatable method F05's and F06's own Verification plans use for their frozen-interface requirements — a diff/grep review performed at code-review time, not a test asserting behavior. |
| AC-019 | REQ-NF-001 (generic, language-neutral tooling posture; not independently named in `spec.md`'s Verification plan table — see "Acceptance-criteria review" traceability note) | 1b | Attack-class enumeration (forbidden-token leak surface, generic-component language neutrality) | `bench/scripts/tests/tc059_replay_offline_determinism_test.sh` | `grep -rE 'python|pytest|pip|go test|golangci-lint|go build'` over every generic F07 script (`run-prelude.sh`, `replay-answer.sh`, `verify-replay-result.sh`, `verify-replay-isolation.sh`) | Zero hits, mechanically proving F07 adds no language-aware generic component — the same mechanical proof F05's AC-012 and F06's AC-019 established. Negative: a future edit inlining a `pytest` invocation into `verify-replay-result.sh` to compute an artifact digest "the easy way" is caught by this same grep, mechanically rather than by code-review convention alone. |

## Acceptance-criteria review

Every AC above is unambiguous, testable, traceable to a `spec.md` REQ, and
specifies an exact expected output (a named failing field, an exact
boundary state, a byte-identical comparison, a specific verdict string)
rather than "works correctly" or "handles errors gracefully." No AC is an
open-ended robustness assertion. Every runtime AC above has at least one
explicit negative case in the matrix.

**Two coverage notes, non-blocking:**

1. AC-017 has no dedicated `tcNNN_*.sh` wrapper script (repeated here from
   "Feature-level coverage check" for visibility); the verification method
   is a direct repo-root Makefile invocation plus a `go list`/diff check.
   This is deliberately thinner than F05's own AC-015/TC-037 for the same
   reason F06's plan recorded for its own AC-017: F07 introduces no new
   submodule, module boundary, or fixture directory for a hygiene script to
   meaningfully compare.
2. **AC-019 has no REQ entry in `spec.md`'s own Verification plan table** —
   the table maps every other REQ-F/REQ-NF to at least one AC, but no row
   names AC-019, even though the Acceptance criteria table itself defines
   it and the Architecture component table assigns it to `tc059`. This is a
   traceability gap in `spec.md` itself, not an invention by this test-plan:
   AC-019's own text ("proving F07 adds no language-aware generic
   component") most plausibly traces to REQ-NF-001's "adds no service …
   generic, language-neutral tooling" posture, and this plan's AC test
   matrix records that inference explicitly (AC-019 row) rather than
   silently treating AC-019 as REQ-less. Recorded here so `spec.md`'s next
   revision can close the gap with an explicit Verification plan row rather
   than this plan quietly working around it.

This note was reached independently of any external red-team pass — see
"Codex test-plan red-team (manual substitute)" below for the self-critique
process that surfaced it, since Codex CLI is unavailable in this
environment.

## ISTQB technique application

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-001 | Contract-surface enumeration + decision table | TC-052 | I-06 is a cross-feature interaction surface (F08 reads it); every field, every closed-vocabulary value, and the bundle/result document-kind distinction must be enumerated against valid fixtures, not sampled. |
| AC-002 | Boundary value analysis + attack-class enumeration | TC-052 | A one-byte mutation is the minimal boundary a content-addressed digest must detect; a spoofed join key across the I-05/I-06 boundary is a leak/trust-boundary property, not a happy-path check. |
| AC-003 | Decision table + attack-class enumeration | tc053 | Structural denial and observational detection are two genuinely different halves (REQ-F-004 vs. REQ-F-005); a decision table forces both to be checked independently, and the missing-member/planted-record cases are the leak surface an attacker (or a lazy implementation) could exploit. |
| AC-004 | Decision table + attack-class enumeration | tc054 | Three named outcomes is a small enumerable combinatorial space; "no fuzzy match" is a defensive property against fabrication, the exact failure mode this feature exists to prevent. |
| AC-005 | State transition | tc054 | Reproducibility is a claim about two executions consuming the same recorded call sequence — the technique built to enumerate exactly that. |
| AC-006 | Decision table + attack-class enumeration | tc055 | Two named failures (`unattributed_artifact`, `unresolved_gate`) with different causes is a decision table by definition; conflating them is the exact leak-surface a resolver-bypassing session could exploit. |
| AC-007 | Equivalence partitioning | TC-052 | `consumers: []` and absent `consumers` are exactly two partitions the schema declares distinct, inherited verbatim from I-05's own rule; the technique's job is proving the boundary between them holds. |
| AC-008 | Decision table + boundary value analysis | TC-052 | Closed field set × discriminator is a small decision table; the `replay_wait_ns` plausibility ceiling is an explicit numeric boundary (BVA: at/over the boundary). |
| AC-009 | Attack-class enumeration | tc056 | "Absent from both agent-visible roots" is a defensive property; the two independent planted-leak locations plus the pre-dispatch timing check are the enumerated leak surface, mirroring F06's AC-009. |
| AC-010 | Decision table + attack-class enumeration | tc057 | Four named cases (never-invoked, written, verbatim, missing-is-failure) is a decision table; silent absence masquerading as success is the leak-surface an inattentive implementation could fall into. |
| AC-011 | Decision table | tc057 | Pass × 3 named violations of a read-only consistency assertion is a small, fully enumerable combinatorial space. |
| AC-012 | Contract-surface enumeration + state transition | tc058 | Five independent placement/identity assertions form a contract surface; the preamble-prepended-before-dispatch requirement is inherently an ordered state (prompt constructed before dispatch, not after). |
| AC-013 | Decision table + attack-class enumeration | TC-052 | Twelve named terminal outcomes plus a required mapping field is a decision table by definition; "eligible with a stop outcome" is the leak/inconsistency case attack-class enumeration is built to catch. |
| AC-014 | Decision table | TC-052 | REQ-F-019 lists ten distinct named rejection cases; each is a row a table-driven test must exercise independently with its own expected failing field. |
| AC-015 | Contract-surface enumeration | TC-052 | Single-owner vocabulary agreement is a bidirectional contract-surface check (schema→fixture and fixture→schema), not a one-directional sanity check. |
| AC-016 | State transition | tc059 | Reproducibility is a claim about two executions of the same state under network isolation — the technique built to enumerate exactly that. |
| AC-017 | Boundary/state enumeration | TC-052 | Submodule-present vs. submodule-absent is the two states worth distinguishing for REQ-NF-003's CI claim. |
| AC-018 | Attack-class enumeration (reviewed, not executed) | Diff review | Frozen-interface regression is a defensive property against silent scope creep into files this feature must not touch. |
| AC-019 | Attack-class enumeration | tc059 | "No generic component branches on language" is a leak-surface property; grep is the mechanical enumeration of that surface, not a convention trusted by inspection. |

ACs without a technique annotation = untestable spec; none exist in this
plan — every AC above carries at least one named technique.

## ISO 25010 coverage matrix

`N/A` cells are justified the same way F05's and F06's plans justified them:
this is offline curator/CI tooling and a host-side prelude adapter with no
production request path or end-user journey. Security coverage here is
unusually dense relative to F05, in the same proportion F06's was, because
F07's core purpose — proving a scored run cannot reach live input and cannot
fabricate lineage — is itself a security-shaped property (a leak surface and
a trust boundary), not an incidental one.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-052 | N/A | ✅ TC-052 (schema-version gate) | N/A | N/A | N/A | ✅ TC-052 (single-owner vocabulary read) | N/A |
| AC-002 | ✅ TC-052 | N/A | N/A | ✅ TC-052 (named offending entry) | ✅ TC-052 | ✅ TC-052 (join-key spoof detection) | N/A | N/A |
| AC-003 | ✅ tc053 | N/A | N/A | ✅ tc053 (named tool/stage on violation) | ✅ tc053 | ✅ tc053 (live-egress leak surface, structural + observational) | N/A | N/A |
| AC-004 | ✅ tc054 | N/A | N/A | ✅ tc054 (named stage/kind/topic) | ✅ tc054 | ✅ tc054 (no fuzzy-match fabrication path) | N/A | N/A |
| AC-005 | ✅ tc054 | N/A | N/A | N/A | ✅ tc054 (reproducibility) | N/A | N/A | N/A |
| AC-006 | ✅ tc055 | N/A | N/A | ✅ tc055 (named entry/stage) | ✅ tc055 | ✅ tc055 (resolver-bypass detection) | N/A | N/A |
| AC-007 | ✅ TC-052 | N/A | N/A | N/A | ✅ TC-052 | ✅ TC-052 (silent-coercion trap) | ✅ TC-052 (schema-level distinguishability) | N/A |
| AC-008 | ✅ TC-052 | ✅ TC-052 (`replay_wait_ns` ceiling) | N/A | ✅ TC-052 (named offending field) | ✅ TC-052 | ✅ TC-052 (human-time-name prohibition) | N/A | N/A |
| AC-009 | ✅ tc056 | ✅ tc056 (before-dispatch timing, via zero-invocation log) | N/A | ✅ tc056 (named root/path) | ✅ tc056 | ✅ tc056 (dual planted-leak locations, digest-matched renamed copy) | N/A | N/A |
| AC-010 | ✅ tc057 | N/A | N/A | N/A | ✅ tc057 (partial-outcome retention) | ✅ tc057 (silent-absence detection) | N/A | N/A |
| AC-011 | ✅ tc057 | N/A | N/A | ✅ tc057 (named offending field/ids) | N/A | ✅ tc057 (cross-package scenario-id mismatch) | N/A | N/A |
| AC-012 | ✅ tc058 | N/A | N/A | N/A | ✅ tc058 | ✅ tc058 (root-placement enforcement) | N/A | ✅ tc058 (real scratch env via `shark-scratch-env.sh`) |
| AC-013 | ✅ TC-052 | N/A | N/A | N/A | ✅ TC-052 (partial evidence retained) | ✅ TC-052 (eligible-with-stop-outcome rejection; I-07 vocabulary containment) | N/A | N/A |
| AC-014 | ✅ TC-052 | N/A | N/A | ✅ TC-052 (named failing field per case) | N/A | N/A | N/A | N/A |
| AC-015 | ✅ TC-052 | N/A | N/A | N/A | N/A | N/A | ✅ TC-052 (bidirectional single-owner agreement) | N/A |
| AC-016 | ✅ tc059 | N/A | N/A | N/A | ✅ tc059 (byte-identical reruns) | ✅ tc059 (zero provider calls, offline) | N/A | ✅ tc059 (network-isolation mechanism, per F05/F06 precedent) |
| AC-017 | ✅ TC-052 | N/A | ✅ TC-052 (submodule-absent CI state) | N/A | N/A | N/A | ✅ TC-052 (repo hygiene) | N/A |
| AC-018 | ✅ Diff review | N/A | N/A | N/A | ✅ Diff review (I-04/I-05/X-10 callers unaffected) | N/A | ✅ Diff review (frozen interface) | N/A |
| AC-019 | ✅ tc059 | N/A | N/A | N/A | N/A | ✅ tc059 (language-branch leak surface) | ✅ tc059 | N/A |

No coverage gaps: every non-`N/A` cell cites a TC or the named diff review;
every `N/A` cell is justified by this feature's offline-tooling/host-adapter,
no-user-journey nature.

## Observability design

Same posture as F05/F06: no metrics/trace spans, because this is
offline/curator-side tooling and a host-side adapter with no production
runtime (REQ-NF-001's "adds no service" applies transitively). Observability
means the scripts' own machine-readable stdout/exit status, which F08 and a
human curator both depend on. The table names *what* each script's terminal
output carries, since that output is the entire observability surface for
this feature.

| Behavior | Log / stdout evidence | Trace/metric | Test assertion |
|---|---|---|---|
| Schema/document-kind validation failure | `TestTC052_...` subtests report the exact failing field or the mismatched document kind, not a generic "invalid" message | N/A — internal, test-only Go binary | TC-052 |
| Join-key mutation | `TestTC052_...` reports `replay_bundle_mutated` naming the entry | N/A — internal | TC-052 |
| Live-egress structural denial | `run-prelude.sh` fails before dispatch naming the missing denial-set member | N/A — internal, no production runtime | tc053 |
| Live-egress observational violation | `verify-replay-isolation.sh` prints `live_interaction_reached` naming the tool and stage | N/A — internal | tc053 |
| Resolver outcome | `replay-answer.sh` prints the supplied response on success, or `replay_desync`/`unresolved_gate` naming stage/kind/topic on failure | N/A — internal | tc054 |
| Lineage reconciliation failure | `verify-replay-result.sh` prints `unattributed_artifact` naming the entry or stage, distinctly from `unresolved_gate` | N/A — internal | tc055 |
| Artifact record verdict | `verify-replay-result.sh` prints `orphan` or `consumption_evidence_missing` per artifact, distinctly | N/A — internal | TC-052 |
| Proxy block violation | `verify-replay-result.sh` prints the offending field name, including a human-time-named field | N/A — internal | TC-052 |
| Bundle-disclosure violation | `verify-replay-isolation.sh` prints `bundle_bulk_disclosure` naming the root and path | N/A — internal | tc056 |
| Non-applicable record verdict | `run-prelude.sh`/`verify-replay-result.sh` print `not_applicable` and confirm per-stage `reason` matched verbatim, or name the mismatch | N/A — internal | tc057 |
| Read-only consistency violation | `run-prelude.sh` prints the offending package/bundle field before any dispatch occurs | N/A — internal | tc057 |
| Placement/identity verdict | `run-prelude.sh` result records `artifact_root.path`/`identity_digest`, `preamble_digest`; violations name the off-limits root and path | N/A — internal | tc058 |
| Terminal-outcome/eligibility verdict | `verify-replay-result.sh` prints `publication_eligible`, `ineligibility_reasons[]`, and `i07_stop_mapping` | N/A — internal | TC-052 |
| Offline/determinism failure | Any guard/script fails naming the specific byte offset or field that differed between the two runs, not a generic "non-deterministic" message | N/A — internal | tc059 |

No new instrumentation beyond structured script/test output is required or
permitted, matching REQ-NF-001's "adds no service, no schema, no migration."

## Caller-Path Contracts

This feature is deterministic host-side tooling (bash/python3 scripts
executing real filesystem checks, digest computation, one real Rider-action
subprocess invocation behind a PATH-stubbed dispatcher in tests, and one real
`shark-scratch-env.sh` invocation), matching F06's posture. `content-only`
opt-outs apply **only** to `bench/README.md`'s documentation additions (see
"Integration scenarios"); every other row below drives its real production
entrypoint. As with F06, most of this feature's scripts have **no
application-level caller above them at all** — E40-F08 has not been
decomposed, so today the shell script or Go test function under test IS the
entrypoint. This is stated explicitly per row, not left implicit.

| TC | Entrypoint (exact invocation) | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-052 | `TestTC052_I06ProductDesignReplayContract` (the contract test function itself; internal — no caller above the Go test binary) calling `os.ReadFile` + real YAML/JSON unmarshal against `bench/replay/i06-schema.yaml` and `tests/contracts/testdata/e40_i06/{valid,invalid}/*` | Real filesystem read of committed YAML/JSON, real parser | Do not substitute an in-memory struct for the committed schema/fixture files; must parse the real files | A validator reading a hand-built in-memory manifest would stay green even if the real committed schema or a real fixture were malformed — the same trap F06's TC-042 Caller-Path Contract names. |
| tc053 | `bench/scripts/run-prelude.sh` invoked directly with a PATH-stubbed dispatcher binary recording its argv (structural half); `bench/scripts/verify-replay-isolation.sh <transcript_path>` invoked directly (observational half). Both internal — no F08 dispatcher exists yet to call either in production | Real argv construction by the real script; real transcript-file parsing | Do not stub the denial-argument construction with a hand-written expected-argv list computed independently of `live-egress-tools.yaml`; the script must read the real file. Do not simulate the transcript scan with a boolean "contains a violation" flag — must parse a real transcript fixture's tool-use records | A script asserting isolation from a hardcoded tool list instead of reading `live-egress-tools.yaml` at call time would silently drift the moment the file changed without the script being updated. |
| tc054 | `bench/scripts/replay-answer.sh --bundle <path> --stage <D0X> --kind <kind> --topic <key>` invoked directly against a real bundle fixture (internal — `replay-answer.sh` IS the entrypoint; no F08 caller exists yet) | Real bundle-file read, real ordinal/topic matching logic, real consumption-record append | Do not hand-compute the "expected" lowest-unconsumed-ordinal response inside the test harness and merely compare the script's output against it without the script performing the lookup itself; do not stub the consumption-ledger append | A test harness that pre-computes "should supply entry N" and merely checks the script agrees, without the script doing the real ordinal-primary lookup, would pass even if the script's own matching logic were wrong on a case the harness didn't pre-compute. |
| tc055 | `bench/scripts/verify-replay-result.sh <result_path> <bundle_path>` invoked directly against real fixture files (internal — no caller above the script exists yet) | Real result/bundle JSON, real reconciliation of claimed consumption against the resolver's own recorded ledger | Do not represent "consumed nothing required" and "no entry available" as two separate boolean flags supplied by the test harness instead of the real JSON shapes — the distinction under test must be derived by the script from real fixture content | A harness that tags fixtures "unattributed"/"unresolved" itself, rather than letting the guard derive the verdict from the JSON shape, would pass even if the guard's own derivation were broken. |
| tc056 | `bench/scripts/verify-replay-isolation.sh <bundle_path> <fixture_checkout> <scratch_project>` invoked directly (internal — no F08 dispatcher exists yet to call it in production) | Real filesystem walk of two live roots; real content-digest computation for the renamed-copy case | Do not stub the planted-leak file's presence with a metadata flag; the file must actually exist on disk at the path under test, including the digest-matched renamed-copy case. A PATH-stubbed dispatcher binary is the one **permitted** mock, used only to prove zero invocations occurred | A guard comparing only filenames (not content digests) would miss a renamed copy of the bundle planted in a scratch project — the same trap a filename-only leak check would fall into. |
| tc057 | `bench/scripts/run-prelude.sh` invoked directly against each of the three non-feature seed packages, plus `bench/scripts/verify-replay-result.sh` for the missing-result and reason-verbatim checks (internal — no F08 caller exists yet) | Real I-04 package files (scratch copies for the mutated cases), real PATH-stubbed dispatcher recording invocations | Do not represent "reason copied verbatim" as a boolean equality flag set by the test; must byte-diff the written result's `reason` field against the real package's `stage_matrix.prelude.D0X.reason` | An implementation that regenerates a paraphrased `reason` string instead of copying the package's own text verbatim would pass a loose "reason present" check but fail this byte-diff. |
| tc058 | `bench/scripts/run-prelude.sh` invoked directly against a real scratch Shark project stood up by `scripts/shark-scratch-env.sh`, with a PATH-stubbed dispatcher recording the constructed prompt and working directory (internal — the script IS the entrypoint; no F08 caller exists yet) | Real scratch-project filesystem, real `preamble.md` file read, real filesystem walk of the fixture checkout and evaluator-only root for off-limits writes | Do not stand up the scratch project by any mechanism other than `scripts/shark-scratch-env.sh` — REQ-NF-005 forbids any other project-initialisation path in this feature's own scripts and tests. Do not compute `preamble_digest` from an in-memory string constant; must read `bench/replay/preamble.md` from disk | A test that fabricates a fake scratch-project directory structure by hand (rather than using the sanctioned scratch-env script) would not catch a `run-prelude.sh` bug that depends on a real Shark project's actual on-disk shape (e.g. `.sharkconfig.json` presence). |
| tc059 | Every guard/resolver/adapter script invoked twice under network isolation, plus a static `grep` over the generic replay scripts (internal — no caller above the scripts/grep exists) | Real subprocess execution with genuine network isolation (Linux `unshare --net` or the portable proxy-poison fallback, per F05's TC-039/F06's TC-051 precedent); real source grep | Do not simulate "offline" with a code-level flag; the environment itself must be offline. Do not scope the grep to a hand-picked line range | An implementation checking only a config flag for "offline mode" while still depending on a network fallback in production would pass a fake test while remaining unsafe; a narrowly-scoped grep would miss a forbidden token reintroduced outside the scanned range. |

## Cross-feature contract tests (I-06, I-04)

### Produces: I-06

Carried verbatim from `spec.md`'s Cross-feature interactions section:

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-06 | E40-F07 | E40-F08 | `architecture.md#product-design-replay-contract` | `tests/contracts/e40_i06_product_design_replay_contract_test.go#TC-052` | TC-052 |

Gate mode: `contract-only`, staged by
`E40-interaction-map.md#i-06-staged-edge` — F07's producer role necessarily
runs before its sole consumer (E40-F08, execution order 8) is decomposed.
Activation owner: E40-F08, closing its own consumption at its own UAT.
Closure key: E40-F08, at its own UAT. Counterpart status: read live from
Shark at review/UAT time, not copied here as a fact that would go stale.
Review basis: this test-plan and `spec.md`, present together at F07
task_review. Demonstrability disposition: `pending-integration` until
E40-F08's live wiring closes.

**Judgment call on the I-06 contract-test pointer, recorded explicitly**, the
same posture F06's plan recorded for I-05. Because F07 is the *producer* side
of a staged edge whose sole consumer (E40-F08) has not yet been decomposed
into its own spec.md/test-plan.md, **TC-052 today has exactly one owner and
one caller: this feature's own Go contract test, reading only this feature's
own committed fixtures.** There is no fabricated shared proof against unbuilt
consumer code, and no attempt to write an "F08-side" test in advance of F08's
own spec. This test-plan asserts the same posture `spec.md` itself states
plainly: "E40-F08 must copy the shape source and the contract-test pointer
above verbatim; the same test proves every side of this contract and no twin
test is created." When F08 is decomposed, its own test-plan.md must reuse
`TC-052` verbatim as the shape source for its runtime-reader obligations
rather than writing a second reader/writer validator — this plan records that
obligation rather than attempting to discharge it early on F08's behalf.

### Consumes: I-04

| I-## | Producer | Consumer | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-04 | E40-F05 | E40-F07 | `architecture.md#lifecycle-scenario-package-contract` | `tests/contracts/e40_i04_scenario_contract_test.go#TC-030` | TC-030 (existing, unmodified) |

F07's consumption slice is `entity_family`, `stage_matrix.prelude`, and
`replay_reference`, assigned verbatim by
`E40-interaction-map.md#i-04-staged-edge` and `spec.md`'s own Consumes: I-04
row. F07 does not extend or re-run TC-030 — the non-regression obligation
instead is that every I-04 artifact REQ-NF-006(c) names is byte-unchanged by
this feature except the single declared `reference-bundle.json` carve-out
(verified by the AC-018 diff review, not by a second F07-owned assertion over
I-04's shape). tc057 and tc058 read `stage_matrix.prelude` and
`replay_reference` through the real `py-feature-recurring-tasks` package, so
TC-030's own passing status is a real precondition for those tests running
against a real package — this is stated as a dependency, not duplicated as a
second contract test. Gate mode: `contract-only`, staged by
`E40-interaction-map.md#i-04-staged-edge`; E40-F07 is the activation owner
for its own slice and closes it at its own UAT (UAT-10) with a real caller
chain (tc057/tc058's direct `run-prelude.sh` invocation against the real
package), shared-contract evidence (TC-030 passing), a production-path
integration test (tc058 against `py-feature-recurring-tasks`, the same
package F05's own AC-004 and F06's tc043 already exercise), and a
wiring-removal counterfactual (a script that stopped reading
`stage_matrix.prelude` from the real package and instead hardcoded
applicability would fail AC-010's non-applicable-package case). Closure key:
E40-F07, at its own UAT.

**Carve-out safety.** `spec.md`'s own review (§"Consumes: I-04", "Carve-out
safety, verified 2026-08-15") confirms TC-030 asserts nothing about
`reference-bundle.json`'s content (no digest, no `_note` key, no byte
length) and no recorded `admission:` digest covers the package directory —
so writing the bundle's interior invalidates no committed TC-030 assertion
and requires no TC-030 edit. This test-plan does not re-verify that claim
with a new test; it is a structural fact about TC-030's existing field
inventory, checked by direct reading, and stated here so a future reader
does not assume it needs independent test coverage.

No twin test is created for I-04: TC-030 remains the single shared proof of
that contract, and F07's scripts consume the artifacts TC-030 validates.

## Cross-epic integration tests (X-10)

### Consumes: X-10 — Shark Rider product-design action and progress record

Carried verbatim from `spec.md`'s Cross-epic integrations section:

| Property | Contract |
|---|---|
| Producer epic / feature | E36 — Project Layer and Consult Bridge (E36-F02) |
| Consumer epic / feature | E40 — Shark Bench (E40-F07), sole owner of this seam |
| What F07 supplies | A digest-pinned interaction-routing preamble, a live-egress denial set, and a local resolver — all outside the frozen adapter and bundle trees |
| Test coverage | UAT-10, UAT-18; tc053 (AC-003), tc054 (AC-004, AC-005), tc058 (AC-012), TC-052's structural cases |
| Deferral | None. No X-10 obligation is deferred to `docs/product/progress.md` |

**Judgment call on scope, recorded explicitly.** `spec.md` states F07
"produces, consumes, or validates **no other X-## row**." This test-plan does
not invent a contract test for X-07 (F02), X-08 (F04), X-09 (F06), X-11/X-13
(F08), or X-12 (F09) — each belongs to its own owning feature's test plan.
The REQ-NF-006(a) byte-freeze plus AC-018's diff review is the mechanical
proof that the wrapped X-10 methodology was not forked — this is X-10's
verification mechanism precisely because X-10 is not a code-callable
contract in the I-## sense (it is "invoke the existing action unmodified,"
verified by non-regression, not by a shared reader/writer test). No entry in
`docs/product/progress.md` records a deferral for X-10 or I-06 as of this
plan's writing (`docs/product/progress.md`'s only E40-relevant row is the
2026-08-11 cross-epic-integration-map entry assigning X-10 to E40-F07; it
records no outstanding deferral).

## Integration scenarios

| Scenario | Boundary | Epic UAT contribution | Test evidence |
|---|---|---|---|
| Replay bundle/result → future E40-F08 reader (I-06) | `bench/replay/*` schema and bundle/result shape → E40-F08's not-yet-built lifecycle runner | UAT-10 | TC-052 pins the shape F08 must consume with no manual step |
| Live-egress denial → real Rider dispatch | `run-prelude.sh` + `verify-replay-isolation.sh` → planted live-egress transcript record | UAT-10 ("scored run cannot reach live research or an interactive question surface") | tc053 |
| Ordinal-primary resolver → real elicitation/research routing | `replay-answer.sh` → real bundle entries | UAT-10 ("two runs consume the same response sequence") | tc054 |
| Lineage reconciliation → resolver-bypass detection | `verify-replay-result.sh` → resolver ledger | UAT-10, UAT-18 | tc055 |
| Entry-at-a-time disclosure → real dispatch boundary | `verify-replay-isolation.sh` → planted-leak fixture roots | UAT-10 | tc056 |
| Non-applicable families → explicit record | `run-prelude.sh` → three non-feature seed packages | UAT-10 ("non-feature scenarios … retain explicit non-applicable stage records") | tc057 |
| Prelude placement → real scratch Shark project | `run-prelude.sh` → `scripts/shark-scratch-env.sh` and the real Rider action's dispatch shape | UAT-10 | tc058 |
| Artifact-consumption edges → orphan/reuse detection | `verify-replay-result.sh` → typed producer/consumer edges | UAT-18 | TC-052 |
| Replayed interaction proxies → human-time prohibition | `verify-replay-result.sh` → closed proxy field set | UAT-18 | TC-052 |
| Fixture and I-06 tree → shark's own quality gate | `bench/replay/*` tree → root `make fmt && make lint && make test` and `go list ./...` | Non-functional: repo hygiene, not a UAT scenario | TC-052 (repo-root gate; see AC-017) |
| Frozen X-10/I-04/I-05 interfaces → new sibling tooling | `skills/shark-rider/verbs/product-design.md`, `internal/sharkdata/default_data/skills/product-design/**`, `bench/scenarios/**`, `bench/evidence/**` (unchanged except the declared carve-out) vs. `bench/replay/**` (new) | Non-functional: no regression to X-10, Phase 1, F05, or F06 (UAT-01, UAT-02, UAT-05-09 transitively) | AC-018 diff review |

Two verification-plan rows are intentionally **not** test cases, matching
`spec.md`'s own Verification plan table's stated method:

- **REQ-NF-005** (replay tooling never touches the live shark database,
  `.sharkconfig.json`, or the live repository working tree, and stands up its
  one needed live scratch project only through `scripts/shark-scratch-env.sh`)
  — verified by code review of `bench/scripts/{run-prelude,replay-answer,
  verify-replay-result,verify-replay-isolation}.sh` confirming no other
  project-initialisation invocation exists, matching `spec.md`'s own "diff
  review" method for the analogous F05/F06 requirement, plus tc058's positive
  case that the one sanctioned scratch-env call succeeds.
- **`bench/README.md`'s new "I-06 product-design replay contract" section** —
  this is prose describing an already-tested shape (TC-052, tc053-tc059
  assert the real shape); the documentation itself is reviewed for accuracy
  against the shape, not independently tested, per the workflow's
  "Prompt-only changes" guidance applied to a docs-only delta.

## Test infrastructure

**Existing patterns to reuse:**
- `tests/contracts/e40_i05_stage_evidence_contract_test.go` and
  `tests/contracts/e40_i04_scenario_contract_test.go` establish the
  repository-root-relative artifact-reading helper style (`filepath.Abs`,
  then `os.ReadFile`) and the `TestTC0NN_...` naming convention — TC-052
  follows this exactly, continuing the epic's sequential TC numbering (F06
  used TC-042; TC-043 through TC-051 are its execution-guard tests; the
  highest TC number in committed epic docs today is TC-051, per
  `bench/scripts/tests/tc051_evidence_offline_determinism_test.sh`, so
  TC-052 is the next free slot, matching `spec.md`'s own explicit
  reference).
- `bench/scripts/tests/run-all.sh` and its `tcNNN_<description>_test.sh`
  naming convention (`tc003_clean_checkout_test.sh` … `tc051_evidence_
  offline_determinism_test.sh`) is the pattern tc053 through tc059's bash
  test scripts follow, registered in the same `run-all.sh` — the only
  existing bench file this feature edits (REQ-F-018's implicit table entry,
  matching how F06's REQ-F-017 table entry named the same file).
- `bench/scripts/verify-evidence-roots.sh`'s and
  `bench/scripts/verify-stage-evidence.sh`'s self-contained `python3`
  heredoc form, `ScriptError`-versus-violation exit-code split, and
  `resolve_within`-style containment for bundle-relative paths is the
  pattern `replay-answer.sh`, `run-prelude.sh`, `verify-replay-result.sh`,
  and `verify-replay-isolation.sh` all follow, per `spec.md`'s own
  component-table note under `replay-answer.sh`.
- `scripts/shark-scratch-env.sh` is the sanctioned mechanism for standing up
  the one live scratch Shark project this feature needs (REQ-NF-005) — no
  new scratch-environment tooling is introduced.
- `internal/runner/dispatcher.go`'s `DefaultDisallowedTools` is the pattern
  (not code) `run-prelude.sh`'s `--disallowedTools`-style argument
  construction mirrors — nothing under `internal/runner` is imported or
  tested by this feature; it is cited here only because `spec.md`/research
  name it as the mechanism's design precedent (ADR-F07-03), not because F07
  adds a test against it.

**New test infrastructure needed (this feature's own deliverables, already
named in `spec.md`'s component table, cross-checked against the AC test
matrix above with no unnamed gap remaining after the one note recorded in
"Feature-level coverage check"):**
- `tests/contracts/e40_i06_product_design_replay_contract_test.go` — one Go
  file, `package contracts`, containing `TestTC052_I06ProductDesignReplayContract`
  and its table-driven valid/invalid-fixture subtests (AC-001, AC-002,
  AC-007, AC-008, AC-013, AC-014, AC-015, AC-017). Per REQ-NF-001, this is
  the **only** Go file this feature adds.
- `tests/contracts/testdata/e40_i06/{valid,invalid}/` — table-driven bundle
  and result fixtures. The ten named REQ-F-019 rejection cases (AC-014) plus
  the document-kind-confusion and vocabulary cases (AC-001, AC-015) plus the
  `consumers` empty-vs-absent and proxy-closure cases (AC-007, AC-008) plus
  the twelve terminal-outcome/`i07_stop_mapping` cases (AC-013) — task
  decomposition must enumerate these as an explicit fixture-authoring task,
  the same posture F05's and F06's plans took for their own malformed-fixture
  tables.
- `bench/scripts/testdata/replay/` — bundle, result, transcript, and root
  fixtures for the bench-script test cases: a conforming bundle/result pair,
  each resolver-outcome case (tc054), each lineage-reconciliation case
  (tc055), each planted-leak case (tc056), each of the three non-applicable
  seed-package cases (tc057), and the live-egress-violation transcript
  (tc053).
- The real `py-feature-recurring-tasks` I-04 package (the same fixture F05's
  AC-004 and F06's tc043 already use) for tc057's read-only-consistency
  cases and tc058's placement case — scratch, test-time-mutated copies for
  the negative cases, not a second committed fixture package.
- A live scratch Shark project stood up per-test by
  `scripts/shark-scratch-env.sh` for tc058, torn down after the test
  (matching that script's own documented lifecycle contract).
- A PATH-stubbed dispatcher/provider binary reused across tc053, tc056,
  tc057, tc058, and tc059 to prove zero (or exactly-recorded) invocations —
  one shared shell helper (mirroring F06's `_stub_dispatcher.sh`) rather than
  five independent implementations, so "records zero invocations" and "the
  recorded argv/prompt matches" mean the same thing in all five places.
- `bench/README.md`'s new "I-06 product-design replay contract (E40-F07)"
  section must name the exact Tier 2 curator invocation sequence (at
  minimum: run `run-prelude.sh`'s pre-dispatch checks before every scored
  dispatch, `verify-replay-result.sh` after every prelude run,
  `verify-replay-isolation.sh` against the retained transcript and roots) so
  REQ-NF-004's reproducibility claim has one documented invocation, not an
  implied one — mirroring the role F05's and F06's README sections played
  for their own Tier 2 sequences.

### Test infrastructure gaps

No unnamed script gap remains: `spec.md`'s component-changes table already
names every script this AC test matrix depends on (`replay-answer.sh`,
`run-prelude.sh`, `verify-replay-result.sh`, `verify-replay-isolation.sh`,
and all seven `tc05[3-9]` test wrappers). The one note recorded above
(AC-017's absent dedicated hygiene script) is not a missing script — it is a
scope decision this plan makes explicit rather than silently assuming, and
does not block task decomposition:

- **AC-017 repo-hygiene verification.** Owner: not applicable — this plan's
  resolution (direct Makefile/`go list`/`go test` invocation, no dedicated
  wrapper) is final unless a later feature in this epic adds a new submodule
  or fixture directory, at which point that feature should add its own
  hygiene script, matching where F05 added TC-037.

## Codex test-plan red-team (manual substitute)

**Verdict:** SKIPPED. Codex CLI is unavailable in this environment; a
manual, adversarial self-review was substituted in its place, applying the
same posture F05's and F06's plans' substitute reviews used (find genuine
coverage gaps, mismatched case counts, unproven "distinct" assertions, and
open-ended robustness language), but performed by re-reading this plan
against `spec.md` line by line rather than by an independent tool
invocation. Findings are recorded honestly below rather than a fabricated
PASS.

**Findings from the manual pass:**

1. **AC-017 repo-hygiene verification has no dedicated `tcNNN` script**,
   unlike every other AC in this matrix and unlike F05's own AC-015/TC-037.
   Resolved above (Feature-level coverage check, item 1; Acceptance-criteria
   review, note 1): this is a legitimate scope difference (F07 adds no
   submodule or new module boundary), not an oversight.
2. **The I-06 contract-test pointer (TC-052) has exactly one real caller
   today** (this feature's own Go test), because E40-F08 does not exist yet
   as committed code. This plan states that explicitly (Cross-feature
   contract tests §"Produces: I-06," "Judgment call") rather than fabricating
   a shared test that spans code that has not been written — checked and
   confirmed present, not a new finding requiring a plan change.
3. **AC-009's tc056 must distinguish a filename-identical planted bundle from
   a content-digest-identical renamed copy, and the matrix must make both
   independently fail.** Re-reading REQ-F-012's "any copy or transformation
   of it" against AC-009's own text ("a copy of it under a different name
   with the same content digest") confirmed the matrix's case (b) already
   requires the digest-matched-not-filename-matched case explicitly — the
   Caller-Path Contract's Forbidden-mocks column for tc056 states this too.
   Checked and confirmed present, not a new finding requiring a plan change.
4. **AC-004's "no fuzzy match" case needs a topic that is a genuine near-miss
   (differs by one character), not merely a different topic altogether**,
   or the negative case would trivially pass any implementation, fuzzy-match
   or not, by both correctly rejecting an obviously-wrong topic. The matrix's
   case (iv) is written with this exact constraint ("differing by one
   character") — checked and confirmed present, not a new finding requiring
   a plan change.
5. **No case in the AC test matrix independently exercises `run-prelude.sh`'s
   REQ-F-014 consistency check against a package whose `stage_matrix.prelude`
   is a *mixture* of `applicable: true` and `applicable: false` entries** —
   every REQ-F-013 case (AC-010) uses a package with all five stages
   `applicable: false`, and REQ-F-014's own cases (AC-011) vary
   `replay_reference` and `scenario_binding.scenario_id`, not the stage
   matrix's applicability pattern itself. `spec.md`'s own text does not
   describe a partial-applicability scenario package in the current corpus
   (F05's seed packages are either "all D01-D05 applicable" — the feature
   family — or "all non-applicable" — the other three families), so this gap
   is **narrow and corpus-shaped, not a scope gap in this plan**: no fixture
   package with a mixed prelude matrix exists to test against, and inventing
   one would test a corpus shape this epic's F05 seed set does not currently
   produce. This is recorded for task decomposition's attention: if a future
   scenario package is admitted with a genuinely mixed `prelude.D0X.applicable`
   pattern, tc057 or tc058 should gain a case exercising it; until then this
   plan's coverage of the two corpus shapes that actually exist (all-true,
   all-false) is complete.

No finding above is blocking: findings 1-4 confirm the plan already states
the required constraint or scope difference; finding 5 is a genuine, narrow
scope question tied to the current corpus's shape, recorded for task
decomposition's attention rather than silently resolved either way.

**Independent second-pass review, applied after the manual self-review
above.** A stronger-reviewer pass against this plan's full transcript
surfaced one genuine, previously-unfound blocker and three real
inconsistencies, all now fixed in the plan above rather than left as
findings-only text:

6. **REQ-F-008 was absent from every `Requirement(s)` cell that should have
   carried it, and one of its MUSTs — "increment `unresolved_gate_count`" —
   had no assertion anywhere.** AC-004 and AC-013 tested REQ-F-008's other
   halves (never inventing an answer; partial retention;
   `publication_eligible: false`; non-empty `ineligibility_reasons[]`) by
   virtue of testing REQ-F-006/007/017, but no case asserted the counter's
   *value* on a run that actually hit an unresolved gate — an
   implementation that declared the field and always left it `0` would have
   passed every case originally written. **Fixed**: REQ-F-008 added to
   AC-004's and AC-013's `Requirement(s)` cells; AC-008's TC-052 case set
   gained cases (f) and (g) asserting `unresolved_gate_count` against the
   recorded gate events and rejecting a zero-count/`unresolved_gate` pairing.
7. **REQ-NF-007 was missing from AC-008's `Requirement(s)` cell**, even
   though AC-008's case (d) already tests exactly the branch `spec.md`'s
   Verification plan assigns it to, and F06's sibling plan traced the
   analogous requirement explicitly in its own AC-004 row. **Fixed**: added.
8. **AC-014's case count was self-contradictory** — "10 named … cases" and
   "one fixture per REQ-F-019 case" against an input column that grouped
   five closed-vocabulary rejections into "one enum-family case," which is
   14 fixtures, not 10. Left ambiguous, a task author building "ten
   fixtures" would under-cover four of the five vocabularies. **Fixed**:
   restated as 14 named fixtures, the five vocabulary cases separated
   explicitly.
9. **AC-019 was traced to an invented REQ-F-018 reading** ("generic-component
   posture") that does not match REQ-F-018's actual text (the single-owner
   vocabulary rule, not language neutrality) — and `spec.md`'s own
   Verification plan table has no row for AC-019 at all. **Fixed**: recorded
   as a genuine spec-side traceability gap (Acceptance-criteria review, note
   2) with this plan's best-supported inference (REQ-NF-001) stated
   explicitly rather than silently asserted as settled fact, and the header
   "No drift found" framing left intact only for scope/requirement drift —
   this is a REQ-to-AC traceability gap in `spec.md`, not scope drift.

Findings 6-9 are now incorporated into the plan above, not merely recorded
as unresolved text. Finding 6 was a real blocker (closed); findings 7-9 are
non-blocking precision fixes (closed).

## Recommendations

- [x] Ready for development — every AC in `spec.md` has a named test case,
  technique, ISO 25010 row, and caller-path contract; every REQ-F-008
  sub-obligation (including `unresolved_gate_count`) now has an assertion.
  Codex CLI was unavailable, so a manual adversarial self-review substituted
  for the automated red-team pass, followed by an independent second-pass
  review; nine findings total are recorded above, of which one (REQ-F-008's
  counter) was a real blocker and is now closed, and three (REQ-NF-007's
  missing trace, AC-014's ambiguous case count, AC-019's invented REQ
  mapping) are precision fixes now closed. Two coverage notes remain
  non-blocking and are resolved with an explicit method rather than left
  silent: AC-017's absent dedicated hygiene script (matching F05's and F06's
  own analogous gaps), and AC-019's absent REQ row in `spec.md`'s own
  Verification plan table (a spec-side traceability gap, not a plan gap).
  The one open scope question (finding 5 above: whether a future
  mixed-applicability scenario package needs a dedicated negative case) is
  flagged for task decomposition's attention, not treated as either resolved
  or blocking, because no such package exists in the current corpus for this
  plan to test against.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.
