---
research_schema: 2
entity_key: E40-F03
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - data
  - workflow_operations
related_work: true
---

# Research report: Baseline report and noise band

## Scope

E40-F03 turns 30 I-02 JSONL artifacts (~10 corpus items × the default
variant × 3 reps) into two deliverables: the Phase 1 baseline report with a
published **noise band** per headline metric (G5), and the G7/UAT-07 replay
verification that re-invokes E40-F02's single-run command against a stored
manifest's pinned inputs and checks the result against that published band.
Both consumers read the same producer surface — **I-02**, produced by
E40-F02, which shipped and was UAT-approved **today** (2026-08-07). F03
consumes nothing else new: no Go code, no shark database change (ADR-002),
and no second implementation of anything F01 or F02 already built
(ADR-F02-06's "invoke, never re-derive" posture, which this research treats
as binding on F03 too).

Two things gate this research beyond the ordinary consumes-a-contract case.
First, a **hard deadline**: E40-F02's own UAT (`uat-20260807-E40-F02.md`,
finding F-1) found the shipped `sources` field narrower than
`spec.md`'s REQ-N-007 and I-02 prose require, and a decision note on E40-F03
(id 2242) says F03's specification step must surface this and
`task_generation` must not proceed with it unresolved. This research's job
is to gather the evidence needed to adjudicate it — not to adjudicate it
itself (that is `spec.md`'s "Durable unresolved decisions" mechanism, the
same one that closed Q001–Q003 upstream). Second, five more UAT findings
(F-3 through F-6) and one carried decision note (T-004) describe producer
quirks in the run driver and collector that F03's aggregator and report must
be designed around, not surprised by.

Terms carried unchanged from the epic and F02 reports: **I-02** (one JSONL
record per run — manifest block, per-stage records, post-run check results,
rollup), **noise band** (the Phase 1 baseline's own run-to-run spread per
metric, the threshold a future config delta must clear), **envelope** (the
raw claude JSON stdout object), and **sources** (I-02's per-metric-family
provenance object, closed five-value set `runresult`/`transcript`/
`scratch_db`/`postrun`/`liveness`).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md` (Goal, Scope items 1–5,
  Acceptance Criteria, Out of Scope), `epic.md` §2 (G5, G7, and the "Phase 1
  exit owns G1–G5 and G7" line), `shark-bench-design.md` §5 (Comparison
  method — 3 reps, paired per-task comparison, the noise band as the
  Phase 1 deliverable), `uat-plan.md` (UAT-01, UAT-07, and the "I-02
  (F02 → F03)" cross-feature scenario), `architecture.md`'s Delivery
  boundaries table (F03's row: trigger, observable result, "no new I-## is
  needed since replay reuses the I-01/I-02 shapes already produced"), and
  the three decision notes on E40-F03 (ids 2213, 2242, 2244) define F03's
  scope, its consumer obligation, and the TD-076 blocking constraint.
- [x] `affected_implementation_or_contract` — Evidence: `bench/README.md`
  §"I-02 record schema field reference" (the shipped, documented shape F03
  must aggregate over — copied verbatim into F03's own spec per decision
  note 2213); `tests/contracts/e40_i02_artifact_contract_test.go`
  (`TestTC001_I02ArtifactContract`, and directly read
  `e40I02ValidateSources` at lines 412–430 — confirms UAT F-1 at the code
  level, independent of the UAT report's own citation); both committed
  golden records (`tests/contracts/testdata/e40_i02_golden_record.jsonl`,
  `..._timeout.jsonl` — concrete shapes for a completed and a timed-out
  run); `bench/scripts/collect-run.sh` (`sources` emission at lines 813,
  870, 873, 878, 916; outcome resolution at 881–922); `bench/scripts/run-one.sh`
  (`fixture_base_sha` resolved from `corpus.yaml` by `--item` lookup at
  lines 204–244, no override flag; `checkout-fixture.sh` invocation at 379;
  base-ledger paths at 593–594).
- [x] `related_work` — Evidence: epic-level `research-report.md` (Decision 5
  — F03's aggregator is new, bench-only tooling, not a repurposing of
  `internal/reporting`); `E40-F02-.../research-report.md` (Finding 6 — B053
  does not reach F02's diff path, so it does not reach F03 either; Decision
  5 — `RunResult.Outcome`'s five-value taxonomy F03 must also respect);
  `E40-F01-.../research-report.md` (Finding 5 — F01's ledgers have F03 as a
  second-order consumer via G7 replay, corroborating the discriminative-band
  feedback loop is an expected, not novel, obligation); `E40-F04-.../research-report.md`
  (I-03 is F02's dependency, already discharged — not a direct F03
  concern); `docs/review/.../E40-F02-.../uat-20260807-E40-F02.md` (primary
  source for findings F-1 through F-6, the AC-by-AC table, and the "Contract
  surfaces" I-02 row); E40-F02's 28-note decision trail, in particular the
  T-004 note (2026-08-07 00:16) and the two prior sanctioned-amendment notes
  (2026-08-06 21:57 `timeout_detail` + 5-value `sources`; 2026-08-07 01:36
  `model_id_source` amendment); TD-076 through TD-079 (tech-debt items filed
  from the same UAT round).
- [x] `pattern_contract` — Evidence: E40-F02's `spec.md` ADR-F02-01 (driver
  and collector are separate scripts; the collector is "a pure function of a
  completed run directory" — the pattern F03's aggregator and the G7 replay
  path should both follow, since a re-run over a stored artifact directory
  is exactly this same operation) and ADR-F02-06 (invoke F01's/F02's
  scripts, never re-derive — extended here to F03 reusing `run-one.sh`
  unmodified for replay); `architecture.md`'s Delivery boundaries table
  (G7's own framing: "no new I-## is needed since replay reuses the I-01/
  I-02 shapes already produced by F01 and F02"); the two prior sanctioned
  schema-amendment precedents on E40-F02 (`timeout_detail`, the 5-value
  `sources` set) as the concrete mechanism a TD-076 amendment would reuse a
  third time, not invent.
- [x] `dependency_impact` — Evidence: `bench/corpus/corpus.yaml` (read
  directly via PyYAML: 10 admitted items — 5 `task`, 5 `bug` — the
  concrete population F03's discriminative-band flagging runs over; 3
  `negative_items`, irrelevant to F03); `collect-run.sh`'s exact `sources`
  emission sites (dependency for the TD-076 adjudication — see Findings 2–3);
  `run-one.sh`'s CLI surface (`--item`/`--variant`/`--rep`/`--timeout`/
  `--out`/`--corpus`/`--keep-scratch`/`--skip-canary` — no per-field SHA
  override, a dependency for how F03's replay command must be built —
  Finding 6); `epic.md:126` and `shark-bench-design.md:103` (corpus rotation
  policy explicitly deferred to Phase 3 — the precondition behind Finding 7's
  ledger-retention gap); `uat-20260807-E40-F02.md` findings F-3 and F-5 read
  together against `feature.md` Scope item 1's resume rule (Finding 9).

## Capability map

| Capability | Evidence | Decision | F03 responsibility |
|---|---|---|---|
| I-02 JSONL record contract (schema, goldens, contract test) | `bench/README.md` §"I-02 record schema field reference"; `tests/contracts/e40_i02_artifact_contract_test.go#TC-001`; both golden records | REUSE, verbatim | Copy the shape source `architecture.md#metric-collection-and-artifact-schema` and `TC-001` verbatim into a task declaring "I-02: consumes" (decision note 2213) — do not re-derive the schema from the collector source. |
| `sources` per-family provenance (REQ-N-007) | `collect-run.sh:813,870,873,878,916`; `e40I02ValidateSources` (`...contract_test.go:412-430`, value-membership only, whole block optional) | **Adjudicate — recommend narrowing amendment over collector rework** (Decisions 1–2) | Do not copy the I-02 shape's `sources` prose verbatim without resolving TD-076 first (hard deadline, decision note 2242). This research recommends option (b); the specification step makes the binding call. |
| `run-one.sh` single-run command | `bench/scripts/run-one.sh` interface (`--item`/`--variant`/`--rep`/`--timeout`/`--out`/`--corpus`/`--keep-scratch`/`--skip-canary`) | REUSE, unmodified | F03's G7 replay command re-invokes this exact script (ADR-F02-06's reuse posture). Because it resolves `fixture_base_sha` from `corpus.yaml` by `--item` lookup with no per-field override (Finding 6), replay must go through the existing `--corpus <path>` flag with a synthetic single-item file, never a code change to `run-one.sh` itself. |
| `bench/corpus/ledgers/<base_sha>/{tests,lint}.json` | `bench/README.md`; `epic.md:126`, `shark-bench-design.md:103` (corpus rotation deferred to Phase 3) | REUSE, with a precondition to pin | Nothing today deletes an old SHA's ledger directory, so replay is safe as long as that stays true. F03's spec should state the precondition explicitly (Finding 7) rather than leave it implicit. |
| Regression-signal definition (`p2p_regressions[]`, not `tests_pass`) | `shark-bench-design.md` §4 "Defects" row; E40-F02 decision note 2026-08-07 00:16 (T-004) | REUSE, as already-designed semantics | Implement the aggregator's defect/regression metric exactly as the design doc already defines it; do not read `quality.tests_pass` as a regression signal (Finding 10, binding). |
| Batch runner "skip completed pairs" (feature.md Scope item 1) | `feature.md` Scope item 1; `uat-20260807-E40-F02.md` findings F-3, F-5 | **NEW**, with a constraint from upstream findings | F03 builds this; it does not exist anywhere upstream. Finding 9 below is a concrete design constraint on it, not a reuse decision. |
| `internal/reporting` (`ScanReport`) | Epic and F02 research reports' Capability map rows (both CONTRADICTS) | CONTRADICTS | Reconfirmed out of scope at F03 scope too — F03's report is new, bench-only markdown + machine-readable-aggregate tooling (ADR-002: JSONL artifacts are the only store), not a repurposing of the docs/plan scan schema. |
| Discriminative-band flagging → F01 feedback (feature.md AC) | `feature.md` Acceptance Criteria ("Tasks that every rep aces or every rep fails are flagged... corpus feedback to E40-F01"); `E40-F01-.../research-report.md` Finding 5 (F03 already anticipated as F01's second-order consumer); `bench/corpus/corpus.yaml` (10 admitted items today) | **NEW** | F03-owned logic with no upstream precedent to reuse; F01's own research already expected this feedback path to exist, so it is not a scope surprise. |

## Findings

1. **The I-02 contract is real, shipped, internally consistent, and the
   right thing for F03 to copy verbatim.** `bench/README.md`'s schema
   reference, `collect-run.sh`'s emission code, both golden records, and
   `TestTC001_I02ArtifactContract` all agree with each other field-for-field
   (independently confirmed by direct reads, not by trusting the UAT
   report's claim alone). This corroborates decision note 2213's instruction
   to copy the shape source and `TC-001` verbatim rather than re-derive.

2. **The closed five-value `sources` set has no value that fits
   `timing.harness_wall_ns`.** That field is driver-measured from
   `meta.json`'s own t0/t1 (`bench/README.md` row `timing.harness_wall_ns`)
   — it is not `runresult`/`transcript`/`scratch_db`/`postrun`/`liveness`
   under any reading. Extending "every metric family declares a source" to
   the `timing` family therefore is not a one-line addition: it requires a
   sixth enum value in the validator (`e40I02SourceValues`), both goldens,
   the README's closed-set sentence, and `spec.md`'s data-model row — a
   schema amendment in every respect except being labeled one.

3. **`stages[]` and `rejections` are each internally mixed-provenance
   across sub-fields already.** `stages[].duration_ns`/`.status`/`.exit_code`
   come from `RunResult` while `stages[].usage.*` comes from a transcript
   (`bench/README.md`'s own per-sub-field table for `usage.*`);
   `rejections.by_gate`/`.rework_loops` come from `RunResult` while
   `rejections.crosscheck.*` comes from the scratch DB. A single
   family-level `sources.stages` or `sources.rejections` string would
   either collapse two real provenances into one (losing information) or
   require a nested per-sub-field `sources` structure `spec.md` never
   designed — new schema surface, not a static-string fix.

4. **The shipped `sources` values that *are* emitted are constant strings
   with exactly one producer path each — they do not disambiguate between
   candidates the way REQ-N-007 describes.** `sources["oracle"]`,
   `["quality"]`, and `["loc"]` are always the literal `"postrun"` when
   present (`collect-run.sh:870,813,873,878` — the guard-fail branch at 813
   and the normal branch at 873 both write the same literal); there is no
   second possible producer for any of the three, so the tag adds nothing a
   consumer could not already infer from the block's own presence. The one
   family whose value genuinely varies across real alternative producers is
   `sources.stalled_stage` (`"liveness"` vs. `"scratch_db"`,
   `collect-run.sh:916`, driven by `resolve_timeout_detail()`) — exactly the
   REQ-N-007 case of "tell a liveness-derived number from a DB-derived one
   without re-deriving the provenance," because which one fired changes how
   much to trust the timeout attribution.

5. **`e40I02ValidateSources` treats the entire `sources` block as optional
   and checks only value-membership of whatever keys happen to be
   present** (`...contract_test.go:412-430`: returns with zero errors if the
   `sources` key is absent at all; for present keys, only checks
   `e40I02SourceValues[s]`). This is an independent, code-level
   confirmation of UAT finding F-1 — the contract test could not catch a
   collector that stopped emitting `sources` entirely, let alone one
   missing a specific family.

6. **G7 replay has an unaddressed mechanism gap: `run-one.sh` cannot be
   told to use a specific pinned `fixture_base_sha`.** `run-one.sh` resolves
   `fixture_base_sha` from `corpus.yaml`'s current entry for `--item <id>`
   (`run-one.sh:204-244`), then passes it to `checkout-fixture.sh`
   (`:379`) and to the base-ledger paths (`:593-594`). The only override
   surface is `--corpus <path>`, which swaps the whole manifest file, not a
   single field. If a corpus curator ever edits `corpus.yaml`'s
   `fixture_base_sha` for an item (ordinary maintenance, not forbidden
   anywhere), a naive replay invocation against the live `corpus.yaml` would
   silently use the *new* SHA instead of the stored manifest's pinned one —
   defeating G7's entire premise with no warning, because nothing in the
   current tooling compares the two.

7. **Ledger retention is a latent, currently-safe, unaddressed
   precondition.** `bench/corpus/ledgers/<base_sha>/` is keyed by SHA and
   nothing in F01 or F02 deletes an old SHA's directory; corpus rotation
   policy is explicitly deferred to Phase 3 (`epic.md:126`,
   `shark-bench-design.md:103`), so this cannot bite today. It is cheap to
   state as a spec-level precondition now and expensive to discover once
   rotation actually starts.

8. **`variant_bundle_sha256` needs an explicit fail-loud comparison on
   replay; `model_ids` does not need a separate one.** The manifest records
   a content hash of the installed bundle, but `--variant <id>` re-installs
   "the variant" by id, not by hash — if bundle content has drifted since
   the original run (e.g. the default bundle's prompt or model assignment
   was tuned), a replay would silently install different content. Model
   selection lives inside that same bundle content (`provider`/`model`/
   `effort` per step, per `OrchestratorAction`), so a bundle-hash match
   already implies a model-ID match — no second, independent model-ID check
   is needed on top of the hash comparison.

9. **F-3 and F-5 interact with F03's own resume rule and change its
   design, not just its caveats.** A crashed prior attempt (UAT F-3: a
   post-run subprocess failure after the agent completed, with no
   `record.jsonl` written) leaves `run/` and `post/` populated in the
   artifact directory. `feature.md` Scope item 1's resume rule — "skip
   already-completed (task, rep) pairs by scanning existing artifacts" —
   as literally stated would read "no `record.jsonl`" as "not yet run" and
   re-invoke `run-one.sh` into that same directory. `run-one.sh`'s overwrite
   refusal keys only on `record.jsonl`'s existence (UAT F-5 evidence,
   `run-one.sh:197-200`, `mkdir -p` at `:419,588`), so the rerun proceeds.
   If that rerun then times out, the driver skips the post-run pipeline
   entirely and the collector has no freshness check on `post/*`
   (`collect-run.sh:796-878`) — it would graft the **prior attempt's** stale
   oracle/quality/loc onto the fresh timeout record, F-5's exact worst case.
   F03's batch runner must therefore treat "artifact directory non-empty,
   `record.jsonl` absent" as its own explicit state, not fold it into
   "not yet run."

10. **T-004 is confirmation that F03 must implement the already-documented
    design, not evidence of a new problem to solve.**
    `shark-bench-design.md` §4's own "Defects" row already defines the
    regression signal as the full-suite diff against the base-SHA ledger
    (`p2p_regressions[]`), not `tests_pass`. T-004 (E40-F02 decision note,
    2026-08-07 00:16) explains *why* the naive read is tempting and wrong:
    F01's fixture repo carries a deliberately permanently-failing regression
    probe, so `quality.tests_pass` reads `false` on essentially every real
    run, orthogonal to whether the agent introduced a real regression.

11. **F-4 and F-6 leave gaps in the current producer that F03's report
    must surface rather than silently absorb.** F-4: a cap-fired-but-
    RunResult-delivered race can produce a non-timeout-outcome record with
    `oracle`/`quality`/`loc` all absent and `errors[]` empty — no field on
    the record explains why. F-6: the driver can never emit REQ-F-016's
    `null`-with-reason quality-gate value, so an environmental gate failure
    (broken tool, not agent-caused) is recorded as `fail`, indistinguishable
    from a real one. Both are open, tracked findings on E40-F02 (not F03's
    to fix), but F03's aggregator and report read their output.

12. **Discriminative-band flagging is genuinely new F03-owned logic with a
    real, already-sized input population.** `bench/corpus/corpus.yaml`
    (read directly) has 10 admitted items — 5 `task`, 5 `bug` — the
    population feature.md's acceptance criterion ("tasks that every rep
    aces or every rep fails are flagged... corpus feedback to E40-F01")
    runs over. F01's own research report (Finding 5) already anticipated
    F03 as a second-order consumer of its ledger format via G7 replay,
    corroborating that this feedback loop was expected at design time, not
    a late addition.

## Decisions

1. **Rigor stays STANDARD (12/27).** Nothing found argues the tier is
   wrong — F03's remaining unknowns are contract-verification and
   mechanism-design questions against a producer surface (I-02) that is now
   fully built and UAT-approved, not open cross-boundary or alternatives
   questions of the kind that would push this to COMPLEX.

2. **Recommend TD-076 option (b) — a sanctioned amendment narrowing
   REQ-N-007/the I-02 prose — over option (a), collector rework.** The
   decisive evidence is Finding 2: the closed five-value set has no value
   for `timing` (a single, fixed, driver-measured producer), so "every
   family" cannot be satisfied without inventing a sixth enum value and
   touching the validator, both goldens, the README, and `spec.md` — option
   (a) is a schema amendment wearing a "fix" label. Finding 3 compounds
   this: `stages`/`rejections` are internally mixed-provenance, so a
   family-level string there either loses information or needs new nested
   schema surface. Finding 4 supplies the principled narrower boundary the
   amendment should state: **families with more than one real producer,
   where the recorded value disambiguates which one fired** — today that is
   exactly `sources.stalled_stage` (`liveness` vs. `scratch_db`) plus, by
   the same logic, `manifest.model_id_source`'s presence/absence
   (`modelUsage` resolved or not). `oracle`/`quality`/`loc` have exactly one
   producer each and their `sources` values are constant literals that add
   no information beyond the block's own presence — extending the same
   low-information pattern to `timing`/`stages`/`rejections` would cost
   real schema surface for no corresponding gain. This is not a new
   mechanism: the 5-value `sources` set itself and `timeout_detail` both
   landed as sanctioned amendments on E40-F02 (decision note, 2026-08-06
   21:57); option (b) is the third use of the same tool, not a departure
   from it. **This research recommends; it does not adjudicate** — the
   binding call belongs to F03's specification step, per decision note
   2242.

3. **F03's spec must design G7 replay as: construct a synthetic
   single-item `corpus.yaml`** carrying the stored manifest's pinned
   `fixture_base_sha`/`corpus_schema_version`/`p2p_set`, and invoke
   `run-one.sh --corpus <synthetic-file> ...` unmodified (Finding 6). Do
   not modify `run-one.sh`'s resolution logic — it is UAT-approved and
   closed, and ADR-F02-06's reuse-not-modify posture extends to F03 as a
   consumer, not an owner, of that script. State the ledger-retention
   precondition (Finding 7) explicitly in the spec — "never delete
   `bench/corpus/ledgers/<sha>/` for any SHA a published manifest
   references" — even though nothing violates it today.

4. **Replay must fail loud on a `variant_bundle_sha256` mismatch** between
   the freshly-installed bundle and the stored manifest's value (Finding
   8), reusing the same re-install-and-hash mechanism `run-one.sh` already
   performs for a fresh run — no new hashing method needed. No separate
   model-ID re-check: it is covered transitively by the bundle-hash
   comparison, since model selection lives inside bundle content.

5. **F03's batch runner must treat "artifact directory present,
   `record.jsonl` absent" as its own explicit state** — clean the stale
   `run/`/`post/` before rerunning, or refuse and report, rather than
   silently folding it into "not yet run" (Finding 9). This is a direct,
   concrete design consequence of two open E40-F02 findings (F-3, F-5)
   interacting with feature.md's own Scope item 1 wording, not a general
   caveat.

6. **F03's aggregator must compute the regression signal exactly as
   `shark-bench-design.md` §4 already defines it** — `p2p_regressions[]`
   from the full-suite ledger diff — and must never read
   `quality.tests_pass` as agent-caused regression (Finding 10, binding per
   E40-F02 decision note 2026-08-07 00:16 / T-004).

7. **F03's report must carry two explicit measurement-noise caveats**
   rather than presenting raw pass/fail counts as clean signal: (a) a
   non-timeout record with `oracle`/`quality`/`loc` all absent and
   `errors[]` empty is its own distinguishable anomaly bucket, not a pass
   or a fail (F-4); (b) `quality.fmt_clean`/`.vet_ok` reading `false` is not
   yet provably agent-attributable until F02 ships REQ-F-016's
   null-with-reason value (F-6) — state this as a known, tracked gap in the
   producer, not something for F03 to work around silently.

8. **F03's `task_generation` must create at least one task declaring
   "I-02: consumes,"** copying `architecture.md#metric-collection-and-artifact-schema`
   and `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` verbatim
   (decision note 2213), and must not proceed until TD-076 is adjudicated
   (decision note 2242's hard deadline). This research supplies the
   adjudication evidence and a recommendation (Decision 2); it does not
   itself close the tech-debt item — that is the specification step's or a
   dedicated task's action.

9. **Discriminative-band flagging is confirmed new, F03-owned logic** with
   no upstream capability to reuse (Finding 12) — `feature.md`'s own
   acceptance criterion, run over the current 10-item corpus.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F03-baseline-report-and-noise-band/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md` (§2 Goals/Success criteria, §4 Constraints)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (Metric collection and artifact schema; Delivery boundaries and traceability)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md` (§4 Per-metric mechanics — Defects row; §5 Comparison method)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md` (UAT-01, UAT-07, I-02 cross-feature scenario)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` (I-02 row)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (epic-level report)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/research-report.md` (sibling, Finding 5)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/research-report.md` (sibling, Findings 6, Decision 5)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/spec.md` (ADR-F02-01, ADR-F02-06, ADR-F02-11, REQ-N-007, "Produces: I-02")
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F04-shark-run-live-progress-and-per-run-log/research-report.md` (sibling)
- `docs/review/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/uat-20260807-E40-F02.md` (findings F-1 through F-6, AC-by-AC table, Contract surfaces I-02 row)
- `shark feature notes E40-F02` / `shark feature get E40-F02 --json` (28-note decision trail, incl. T-004 note 2026-08-07 00:16, sanctioned-amendment notes 2026-08-06 21:57 and 2026-08-07 01:36)
- `shark feature notes E40-F03` / `shark feature get E40-F03 --json` (decision notes 2213, 2242, 2244)
- `docs/plan/tech-debt/TD-076.md`, `TD-077.md`, `TD-078.md`, `TD-079.md`
- `bench/README.md` (§"I-02 record schema field reference", §"Confirmed claude CLI JSON envelope field names")
- `bench/scripts/collect-run.sh` (lines 780-938: `compute_postrun`, `sources` emission at 813/870/873/878/916, outcome resolution at 881-922)
- `bench/scripts/run-one.sh` (`fixture_base_sha` resolution at 204-244, checkout invocation at 379, base-ledger paths at 593-594)
- `tests/contracts/e40_i02_artifact_contract_test.go` (`TestTC001_I02ArtifactContract`, `e40I02ValidateSources` at lines 412-430)
- `tests/contracts/testdata/e40_i02_golden_record.jsonl`, `e40_i02_golden_record_timeout.jsonl` (read directly)
- `bench/corpus/corpus.yaml` (read via PyYAML: 10 admitted items, 3 negative items)
- `internal/sharkdata/default_data/research/recipes.yaml` (the actual v2 catalog read for this report; `shark-data/research/recipes.yaml` named in the dispatch prompt does not exist — the same discrepancy F02's and F04's reports already documented, now a third independent observation)

RECOMMENDED OUTCOME: standard
