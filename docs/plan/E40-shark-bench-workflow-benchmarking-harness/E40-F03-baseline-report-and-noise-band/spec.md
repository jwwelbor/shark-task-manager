---
feature_key: E40-F03
epic_key: E40
title: "Baseline report and noise band"
type: combined-spec
tier: STANDARD (12/27)
date: 2026-08-07
related-docs:
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F03-baseline-report-and-noise-band/research-report.md
  - docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/spec.md
  - docs/review/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/uat-20260807-E40-F02.md
  - bench/README.md
---

# E40-F03 Specification — Baseline report and noise band

## Context (references, not restatement)

- Business context and Phase 1 scope: epic PRD [§1](../epic.md), [§2](../epic.md) (G5, G7 and the "Phase 1 exit owns G1–G5 and G7" line), [§3](../epic.md), [§4](../epic.md).
- System-level decisions: [architecture.md](../architecture.md) — "Metric collection and artifact schema", "Delivery boundaries and traceability" (F03's row), ADR-002.
- Comparison method and per-metric mechanics: [shark-bench-design.md §4](../shark-bench-design.md) (Defects row), [§5](../shark-bench-design.md) (3 reps, paired per-task comparison, the noise band as the Phase 1 deliverable).
- Feature scope and acceptance: [feature.md](feature.md).
- Capability reuse decisions: [research-report.md](research-report.md) "Capability map" and "Decisions" 1–9 — **binding on this spec** except where §"TD-076 adjudication" and ADR-F03-05 record a deliberate, evidenced narrowing.
- Producer surface: E40-F02's [spec.md](../E40-F02-bench-harness-run-driver-and-metric-collection/spec.md) (ADR-F02-01, ADR-F02-06, ADR-F02-11) and `bench/README.md` §"I-02 record schema field reference".
- Known producer gaps this feature reads around, not fixes: [uat-20260807-E40-F02.md](../../../review/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/uat-20260807-E40-F02.md) findings F-3 (TD-078), F-4 (TD-079), F-5 (TD-080), F-6 (TD-081).

This feature adds no business capability beyond what the PRD states. Everything below is incremental to the epic.

### Capabilities reused, not re-implemented

Named from the research report's Capability map; this feature builds none of them.

| Reused capability | Source | What F03 does instead of building it |
|---|---|---|
| Single-run driver | `bench/scripts/run-one.sh` (UAT-closed) | Invokes **unmodified**, for both the batch and the G7 replay (ADR-F02-06, ADR-F03-02) |
| I-02 record schema, goldens, contract test | `architecture.md#metric-collection-and-artifact-schema`; `bench/README.md` §"I-02 record schema field reference"; `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` + both goldens | Copies the shape source and contract test **verbatim** (decision note 2213); never re-derives the schema from `collect-run.sh` |
| Metric collection | `bench/scripts/collect-run.sh` | Never invoked directly by F03 and never re-implemented; F03 reads only the `record.jsonl` it produced |
| Ledger diff semantics / regression definition | `bench/scripts/diff-ledgers.sh`; `shark-bench-design.md §4` Defects row | Reads `oracle.p2p_regressions_count` as copied onto the record; never recomputes a diff (ADR-F03-06) |
| Base-SHA ledgers | `bench/corpus/ledgers/<base_sha>/{tests,lint}.json` | Read indirectly through `run-one.sh` on replay; F03 only asserts the directory's existence as a precondition (REQ-F-025) |
| Corpus manifest | `bench/corpus/corpus.yaml` | Read for the item list and for the seed/F2P paths a synthetic replay manifest must point at; never rewritten in place |
| Scratch provisioning + live-repo guardrail | `scripts/shark-scratch-env.sh` (via `run-one.sh`) | Never invoked directly; F03 runs no shark project-initialisation command |
| X-07 canary | `bench/scripts/canary-runsurface.sh` | Left to `run-one.sh`'s own default invocation; F03 adds no second canary path |

Explicitly **not** reused: `internal/reporting` (`ScanReport`) — a fixed `docs/plan` scan schema, unrelated to bench records (architecture.md "Scope and component design"; reconfirmed CONTRADICTS at F03 scope by this feature's research Capability map).

---

## TD-076 adjudication (binding, executed)

**Decision: option (b) — a sanctioned amendment narrowing REQ-N-007 and the I-02 prose. Executed in this specification step.**

### What was adjudicated

E40-F02's UAT finding F-1 (medium, `uat-20260807-E40-F02.md`) found the shipped `sources` block narrower than F02's `spec.md` requires: `collect-run.sh` emits `sources.oracle`/`.quality`/`.loc` (`:813,870,873,878`) plus `sources.stalled_stage` on the timeout path (`:916`), and nothing for the `timing`, `stages`, or `rejections` families — while REQ-N-007 says "every measurement", the data-model row says "per metric family", and the "Produces: I-02" section pins TC-001 as validating "every metric family declaring a source". The contract test enforces only value-membership of whatever keys are present, and treats the whole block as optional (`e40I02ValidateSources`, `:413-434`). Decision note 2242 makes adjudication a **hard deadline before F03's `task_generation`**, because after that point the divergence propagates into the consumer's contract.

### Why (b) and not (a)

The research report's Decisions §2 supplies the evidence; this step weighed it and finds it decisive on three independent grounds.

1. **The closed five-value set has no value that fits `timing`.** `timing.harness_wall_ns` is driver-measured from `meta.json`'s own t0/t1 — it is not `runresult`, `transcript`, `scratch_db`, `postrun`, or `liveness` under any reading. Option (a) therefore cannot be executed as "emit the static entries": it requires a sixth enum value in `e40I02SourceValues`, both committed goldens, `bench/README.md`'s closed-set sentence, and F02's data-model row. That is a schema amendment on a UAT-closed feature wearing a "fix" label — strictly more invasive than the amendment it was offered as the alternative to.
2. **`stages` and `rejections` are internally mixed-provenance already.** `stages[].duration_ns`/`.status`/`.exit_code` come from `RunResult` while `stages[].usage.*` comes from a transcript; `rejections.by_gate`/`.rework_loops` come from `RunResult` while `rejections.crosscheck.*` comes from the scratch DB — both documented per sub-field in `bench/README.md`. A single family-level string would either collapse two real provenances into one (losing information the README already records correctly) or require a nested per-sub-field `sources` structure that no spec designed.
3. **The shipped entries that exist are constant literals, so the pattern being generalised carries no information.** `sources.oracle`/`.quality`/`.loc` are always the literal `"postrun"` when present; there is no second producer for any of the three, so the tag says nothing the block's own presence does not. Extending a zero-information pattern to three more families costs real schema surface for no gain.

The principled boundary that survives is the one REQ-N-007's own rationale asks for — *tell one producer from another without re-deriving the provenance* — which only bites where more than one producer exists: today `sources.stalled_stage` (`"liveness"` vs `"scratch_db"`, driven by `resolve_timeout_detail()`) and, by the same logic, `manifest.model_id_source`'s presence/absence. The amendment states exactly that boundary.

This is the **third** use of an established mechanism, not a new one: F02's `timeout_detail` block and the five-value `sources` set both landed as sanctioned amendments (decision notes 2026-08-06 21:57 and 2026-08-07 01:36).

### What this adjudication changed

Four edits to `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/spec.md`, each tagged in place as sanctioned by this adjudication:

| Site | Change |
|---|---|
| REQ-N-007 (Non-functional) | "Every measurement" → "Every measurement **whose provenance is variable**", with the boundary defined and the single-producer exemption stated |
| Traceability table row | Label restated to match ("variable-provenance metrics name their source") so the restatement does not go stale |
| Data model, `sources` row | Narrowed to families with more than one real producer; the shipped emitting set enumerated exhaustively; `timing`/`stages`/`rejections` named as deliberately exempt with the reason |
| "Produces: I-02", TC-001 sentence | "every metric family declaring a source" → "every `sources` entry naming a value within the closed five-value set", with a note that the validator's existing check is the correct enforcement of the narrowed requirement |

**No code, golden, validator, or `bench/README.md` change follows.** The shipped collector, both goldens, `e40I02ValidateSources`, and the README field reference already implement and document exactly the narrowed requirement — that internal consistency is what made (b) available. `bench/README.md` is deliberately left untouched: it documents the shipped shape, frames every field as conditionally present, and never claims per-family completeness, so it was never part of the divergence.

**Consequence for this feature, stated for the exit gate:** the I-02 shape F03 consumes is **the shipped one**. `sources` is a conditionally-present object whose complete emitting set is `oracle`/`quality`/`loc` = `"postrun"` (present when the post-run phase ran) and `stalled_stage` = `"liveness"` \| `"scratch_db"` (present only on the timeout path). F03's aggregator must not treat a missing `sources.timing`, `sources.stages`, or `sources.rejections` as a defect, and must not derive family presence from `sources` — family presence is determined by the family block itself (REQ-F-007).

TD-076 is adjudicated and executed; the parent loop closes it.

---

## Requirements

### Functional

**Batch runner** (feature.md Scope 1; UAT-01)

- **REQ-F-001** — A single command executes the Phase 1 matrix — every admitted corpus item × one variant × N reps — unattended, never prompting, and terminates with a machine-readable summary of what it ran, skipped, and failed.
- **REQ-F-002** — The batch is resumable: a `(item, variant, rep)` whose artifact directory contains `record.jsonl` is skipped, not re-invoked. Re-running a completed batch performs zero runs and zero API spend.
- **REQ-F-003** — "Artifact directory present and non-empty, `record.jsonl` absent" is its own explicit state, never folded into "not yet run". The default action is to skip it, name it, and continue; `--reclaim-incomplete` first **moves** the stale directory to a quarantine path and only then re-invokes the driver. The runner never re-invokes the driver into a directory that already holds a prior attempt's `run/` or `post/`.
- **REQ-F-004** — A single pair's non-zero exit never aborts the batch: it is recorded and the batch proceeds to the next pair, so the worst case is a batch with named holes rather than a stopped batch.
- **REQ-F-005** — The item list comes from `corpus.yaml`'s `items:` only; `negative_items:` are never benched.
- **REQ-F-006** — The batch runner performs no destructive filesystem operation on any artifact directory: quarantine is a move, never a delete.

**Aggregator** (feature.md Scope 2–3; UAT-01, G5)

- **REQ-F-007** — The aggregator reads only `record.jsonl` files under the artifact root and nothing else — no batch log, no scratch project, no database, no network (ADR-002: reproducible from the artifact directory alone). A family's presence is determined by the family block itself, never by the `sources` block.
- **REQ-F-008** — Every record is classified by named rule as `complete`, `explained_absence`, or `anomaly`, and its classification is recorded against its `manifest.run_key`.
- **REQ-F-009** — A record missing a metric family is never averaged into that family's statistics. It is either excluded with a named reason (when the absence is explained) or bucketed as an `anomaly`, and any `anomaly` makes the aggregation exit non-zero with each anomalous `run_key` and missing family named. This is uat-plan.md's I-02 scenario made executable: "a record missing a metric family fails aggregation loudly instead of being silently averaged away."
- **REQ-F-010** — A structurally invalid record (unsupported `schema_version`, unparseable JSON, missing `manifest` or `outcome`) is a hard failure: the aggregator names the file and exits non-zero **without writing an aggregate**.
- **REQ-F-011** — Provenance uniformity is checked across the contributing records of a variant: `manifest.model_ids`, `.fixture_base_sha`, `.variant_bundle_sha256`, `.corpus_schema_version`, and `.shark_version`. A non-uniform batch is reported as **invalid as a baseline**, naming the differing values — never published as a result (uat-plan.md, Non-functional evidence: "a comparison spanning unpinned model versions is reported as invalid rather than as a result").
- **REQ-F-012** — For every `(item, variant, metric)` the aggregate records the contributing rep count, the excluded rep list with per-exclusion reasons, and — by metric class — the observed statistics defined in "Data model changes".
- **REQ-F-013** — Per-step token, cost, and duration aggregation is keyed on `stages[].status` (the workflow step), with repeat occurrences of the same step within one run summed, so a reworked step reports the step's total cost rather than one arbitrary occurrence.
- **REQ-F-014** — The regression signal is `oracle.p2p_regressions_count` and nothing else. No aggregate field named or derived as a regression may read `quality.tests_pass`.
- **REQ-F-015** — Absent keys within a **present** `rejections.by_gate` object count as zero for that gate, because the producer defines `rejections.rework_loops` as the sum of `by_gate`'s values. An absent `rejections` **block** remains unknown, never zero. The gate key universe is the union of keys observed across contributing records.
- **REQ-F-016** — Each `(item, metric)` publishes an acceptance interval alongside its observed spread, derived by the per-class rule in "Data model changes". A metric with fewer than two contributing reps publishes no interval and is flagged `insufficient_reps`.
- **REQ-F-017** — A task whose `oracle.f2p_resolved` is identical across every contributing rep (all true or all false) is flagged non-discriminative, as corpus feedback to E40-F01.
- **REQ-F-018** — A metric whose absolute spread exceeds its mean is flagged unusable at the current rep count.
- **REQ-F-019** — The aggregate carries an `input_digest`: a sha256 over the sorted `"<sha256>  <relative path>"` lines of every contributing `record.jsonl`, so a published report can be tied to the exact artifact set it was computed from.

**Report** (feature.md Scope 4; G5, UAT-01)

- **REQ-F-020** — The markdown report is a pure function of the aggregate JSON: it reads no artifact directory, consults no clock, and contains no value not present in or derivable from the aggregate.
- **REQ-F-021** — The report states the noise band explicitly per headline metric — observed `min`/`median`/`max`, absolute and relative spread, **and** the published acceptance interval with its derivation rule printed — never means alone.
- **REQ-F-022** — The report records the run provenance: exact model IDs, fixture base SHA, variant bundle sha256, corpus schema version, shark version, rep count, and `input_digest`.
- **REQ-F-023** — The report lists non-discriminative tasks under an explicit corpus-feedback heading naming E40-F01.
- **REQ-F-024** — The report carries the four named measurement caveats, each linked to its tracked producer item: the timeout-exclusion rule; the unexplained-absence anomaly bucket (F-4 / TD-079); `quality.fmt_clean`/`.vet_ok` not being provably agent-attributable until REQ-F-016's null-with-reason value ships (F-6 / TD-081); and `quality.tests_pass` not being a regression signal, because the fixture carries a deliberately permanently-failing regression probe (T-004).

**Replay verification** (feature.md Scope 5; G7, UAT-07)

- **REQ-F-025** — Replay reads a stored `record.jsonl` and a published aggregate, and nothing else about the original run. It re-invokes the driver with the **stored** `--rep` value against a distinct `--out` root, so the fresh record's `manifest.run_key` (`<item_id>::<variant_id>::rep<rep>`) is byte-identical to the stored one and is the join key for the comparison — no rep renumbering convention is introduced, and the distinct root is what prevents a collision with the original artifact.
- **REQ-F-026** — Replay pins the manifest's inputs by constructing a synthetic single-item `corpus.yaml` and passing it through `run-one.sh`'s existing `--corpus` flag. `run-one.sh` is invoked unmodified; no F03 change to it, `collect-run.sh`, or any other landed bench script is in scope.
- **REQ-F-027** — Preconditions are checked **before** dispatch, so a replay that cannot be valid costs no API spend: `bench/corpus/ledgers/<manifest.fixture_base_sha>/{tests,lint}.json` exist; the item id resolves in the source corpus; the published aggregate contains a band for that `(item, variant)`.
- **REQ-F-028** — Any divergence between the live corpus item's `fixture_base_sha` / `p2p_set` / top-level `schema_version` and the manifest's pinned values is recorded as a named `corpus_drift` entry. Replay always uses the **manifest's** values, never the live corpus's.
- **REQ-F-029** — After the replay run produces a fresh record, `manifest.variant_bundle_sha256` and `manifest.model_ids` are compared with the stored record's. A mismatch of either yields the verdict `invalid` with the expected and actual values named — the inputs were not reproduced, so the metric comparison is meaningless (ADR-F03-05).
- **REQ-F-030** — The verdict is three-valued — `pass`, `fail`, `invalid` — and is accompanied by a per-metric table giving the replayed value, the published acceptance interval, and the per-metric verdict.

### Non-functional

- **REQ-N-001** — Scripts match the existing `bench/scripts` conventions: `bash` entry point with `set -euo pipefail`, embedded `python3` (PyYAML available) for YAML/JSON work, machine-readable JSON to stdout, diagnostics to stderr, and a self-test under `bench/scripts/tests/tcNNN_*_test.sh` registered in `bench/scripts/tests/run-all.sh` (mirrors E40-F02 REQ-N-006).
- **REQ-N-002** — Nothing in this feature writes to the live repository, its `.sharkconfig.json`, or the live database, and no script invokes a shark project-initialisation command. All provisioning happens inside `run-one.sh`, which owns the `scripts/shark-scratch-env.sh` path (epic §4).
- **REQ-N-003** — Zero Go changes: no new or modified file under `internal/`, `cmd/`, or `tests/contracts/`. The epic's single Phase 1 Go change is spent on F04, and the one Go file in the bench surface — `tests/contracts/e40_i02_artifact_contract_test.go` — is owned by F02 and referenced, not extended (ADR-F03-09).
- **REQ-N-004** — Aggregate and report output is deterministic for a fixed input: JSON object keys emitted sorted, list-valued fields in a fixed stated order, no wall-clock value in either document (mirrors E40-F02 REQ-N-004, and is the property `input_digest` rests on).
- **REQ-N-005** — Fail loud everywhere a measurement could be fabricated: an absent metric is never read as zero, an unparseable record is refused rather than skipped, and a band is never published over a silently reduced rep set (mirrors E40-F02 REQ-N-005).
- **REQ-N-006** — Self-test fixtures are derived from the committed I-02 goldens (`tests/contracts/testdata/e40_i02_golden_record.jsonl`, `..._timeout.jsonl`), never hand-authored records. Hand-authoring a fixture would re-create UAT F-1's exact defect class one layer down: a test encoding an assumed shape rather than the shipped one.
- **REQ-N-007** — Two preconditions on the artifact and corpus surfaces are documented in `bench/README.md` and asserted where cheaply assertable: (i) `bench/corpus/ledgers/<sha>/` is never deleted for any SHA a published manifest references; (ii) a corpus item's seed file and held-back F2P test files are treated as immutable for any SHA a published manifest references (Q005).

### Acceptance criteria

| AC | Statement |
|---|---|
| AC-01 | A batch invocation over the 10-item corpus × 3 reps runs to completion unattended and prints a summary naming every pair's classification; no invocation prompts for input. |
| AC-02 | Re-invoking the same batch over a fully-populated artifact root performs zero `run-one.sh` invocations (asserted by a stubbed driver that records its call count) and exits zero. |
| AC-03 | Against an artifact root where one `rep-N/` holds `run/` and `post/` but no `record.jsonl`, the default batch invocation makes no `run-one.sh` call for that pair, names it in the summary as `incomplete_prior_attempt`, still runs every other pending pair, and exits non-zero. |
| AC-04 | The same fixture with `--reclaim-incomplete` moves that directory under the quarantine root, leaves its contents byte-identical there, and invokes `run-one.sh` exactly once for the pair against a now-absent target path. |
| AC-05 | With a stub driver whose exit status is non-zero for one pair, the batch still attempts every remaining pair. |
| AC-06 | The aggregator over a fixture root built from the committed goldens produces byte-identical output on two consecutive invocations. |
| AC-07 | A fixture record with an unsupported `schema_version` makes the aggregator exit non-zero, name the file, and write no aggregate file. |
| AC-08 | A non-timeout fixture record with `oracle`, `quality`, and `loc` all absent and `errors[]` empty is classified `anomaly`, appears in the aggregate's anomaly bucket with its `run_key`, contributes to no band, and makes the aggregator exit non-zero. |
| AC-09 | A timeout fixture record (the committed timeout golden) contributes to the outcome counts and the timeout rate, and appears in the `excluded[]` list — with reason `outcome_timeout` — of **every** registry metric applicable to its item type, including metrics whose family that record could not have carried; no band value equals the timeout cap, and `wall_clock_ns`'s `n` is one lower than the rep count. |
| AC-10 | A fixture record whose `quality.toolchain_guard` is not `"pass"` is classified `explained_absence` and excluded from the oracle/quality/LOC bands with that reason named; the aggregator exits zero when no anomaly exists. |
| AC-11 | A fixture set whose records carry two different `manifest.model_ids` values is reported as invalid as a baseline, naming both values, and exits non-zero. |
| AC-12 | For a task with three contributing reps, the aggregate's per-metric block carries `n=3`, `min`, `median`, `max`, `mean`, `spread_abs`, `spread_rel`, `accept_lo`, and `accept_hi`, and `accept_lo <= min` and `accept_hi >= max` hold for every Class B and Class C metric. |
| AC-13 | A metric with exactly one contributing rep is flagged `insufficient_reps` and publishes no `accept_lo`/`accept_hi`. |
| AC-14 | A task whose three reps all report `oracle.f2p_resolved: true` (and, separately, all `false`) is flagged non-discriminative and appears in the report's corpus-feedback section. |
| AC-15 | A metric whose fixture values give `max - min > mean` is flagged unusable at the current rep count. |
| AC-16 | Per-step aggregation over a fixture run whose `stages[]` contains the same `status` twice reports that step's token total as the sum of both occurrences, not either one alone. |
| AC-17 | No field in the aggregate whose name or documented meaning is a regression signal is computed from `quality.tests_pass`; the aggregate's regression field equals the record's `oracle.p2p_regressions_count`. |
| AC-18 | A fixture record whose `rejections` block is present but whose `by_gate` omits a gate observed elsewhere in the batch contributes `0` for that gate; a record whose whole `rejections` block is absent contributes nothing and is listed as excluded for the rejection metrics. |
| AC-19 | The report generator over a fixed aggregate produces byte-identical markdown on two consecutive invocations and contains no timestamp. |
| AC-20 | The report contains, per headline metric, both the observed spread and the acceptance interval with its derivation rule stated; and contains all four caveats of REQ-F-024, each naming its tracked item (TD-079, TD-081, T-004) or rule. |
| AC-21 | The report's provenance block reproduces the model IDs, fixture base SHA, bundle sha256, corpus schema version, shark version, and `input_digest` from the aggregate. |
| AC-22 | Replay against a stored record whose `fixture_base_sha` has no `bench/corpus/ledgers/<sha>/` directory exits non-zero naming the missing path, and makes no `run-one.sh` invocation. |
| AC-23 | The synthetic corpus file replay writes carries the manifest's `fixture_base_sha` in both `fixture.base_sha` and the item's `fixture_base_sha`, the manifest's `corpus_schema_version` as the top-level `schema_version`, the manifest's `p2p_set`, and absolute `seed_path` and `f2p.paths` entries that resolve to the source corpus's files. |
| AC-24 | With the live corpus item's `fixture_base_sha` deliberately edited away from the manifest's, replay records a `corpus_drift` entry naming both values and still invokes `run-one.sh` with a synthetic corpus carrying the **manifest's** SHA. |
| AC-25 | With a stub driver producing a fresh record whose `manifest.variant_bundle_sha256` differs from the stored record's, the verdict is `invalid` with reason `variant_bundle_drift` and both hashes named — not `fail`. |
| AC-26 | With a stub driver producing a fresh record whose `manifest.model_ids` differs from the stored record's while the bundle hash matches, the verdict is `invalid` with reason `model_version_drift` and both ID lists named. |
| AC-27 | With inputs matching and every headline metric inside its published acceptance interval, the verdict is `pass`; moving one metric outside its interval changes the verdict to `fail` and names that metric, its replayed value, and its interval. The fixture passes `run-one.sh` the **stored** `--rep` value with a distinct `--out` root, and the assertion includes that the fresh record's `manifest.run_key` is byte-identical to the stored record's. |
| AC-28 | `make fmt && make lint && make test` are unaffected by this feature — no Go file is added or changed — and `bench/scripts/tests/run-all.sh` passes with tc017, tc018, and tc019 registered. |

### Traceability

| Requirement | Acceptance criteria | Epic goal / UAT |
|---|---|---|
| REQ-F-001..006 batch runner, resumable, non-destructive | AC-01..AC-05 | G2, UAT-01 |
| REQ-F-007..011 artifact-only, classified, loud on gaps, uniformity | AC-06..AC-11 | G4, UAT-01, I-02 scenario |
| REQ-F-012..019 statistics, per-step keys, regression signal, flags, digest | AC-12..AC-19 | G5 |
| REQ-F-020..024 report content and caveats | AC-19..AC-21 | G5, UAT-01 |
| REQ-F-025..030 replay, preconditions, drift, three-valued verdict | AC-22..AC-27 | G7, UAT-07 |
| REQ-N-001 bench script conventions | AC-28 + tc017/tc018/tc019 | — |
| REQ-N-002 live repo/DB untouched | Inherited from `run-one.sh`; no F03 script invokes shark directly | epic §4 |
| REQ-N-003 zero Go changes | AC-28 | epic §4 (one Phase 1 Go change, spent on F04) |
| REQ-N-004 deterministic outputs | AC-06, AC-19 | G7 |
| REQ-N-005 fail loud, never fabricate | AC-07, AC-08, AC-09, AC-18 | G4 |
| REQ-N-006 fixtures derived from goldens | AC-06, AC-08, AC-09 | I-02 |
| REQ-N-007 retention and immutability preconditions | AC-22 (retention half) | G7, Q005 |

### Out of scope for this feature

- **Variant comparison and A/B delta reports** (G6, UAT-03, UAT-04) — Phase 2 by PRD §3 and Q002. Phase 1 publishes the band those scenarios later measure against; F03 computes no delta between two variants.
- **Statistical significance testing beyond min/median/max spread.** feature.md Out of Scope states this explicitly ("revisit if 3-rep bands prove too coarse"), and epic §2 carries the matching non-goal on sensitivity. Not an open question — a settled scope boundary (see "Durable unresolved decisions").
- **Fixing any E40-F02 producer gap.** F-3/TD-078, F-4/TD-079, F-5/TD-080, F-6/TD-081 are read around and surfaced in the report, never patched here. F-5/TD-080 in particular is the producer-side half of REQ-F-003; F03 defends against it from the consumer side without changing `run-one.sh`.
- **Any change to `run-one.sh`, `collect-run.sh`, `canary-runsurface.sh`, `admit.sh`, `build-ledgers.sh`, `diff-ledgers.sh`, `checkout-fixture.sh`, the corpus, or the goldens.**
- **Corpus rotation policy** — deferred to Phase 3 (epic.md:126, shark-bench-design.md:103). F03 states the retention precondition it depends on; it does not author the policy.
- **Acting on the corpus feedback.** F03 flags non-discriminative tasks; replacing or re-screening them is E40-F01's surface.
- **Publishing the baseline as CI, a dashboard, or a scheduled job** (epic §3, out of scope for E40 entirely).

---

## Architecture

### Component changes

| File | Change | Role |
|---|---|---|
| `bench/scripts/run-batch.sh` | New | Matrix driver: enumerate items × reps, classify each target directory, invoke `run-one.sh` for pending pairs, quarantine on `--reclaim-incomplete`, emit a JSON summary |
| `bench/scripts/aggregate-runs.sh` | New | Pure function of an artifact root → one aggregate JSON on stdout: classification, per-`(item, metric)` statistics, acceptance intervals, flags, exclusions, provenance, `input_digest` |
| `bench/scripts/report-baseline.sh` | New | Pure function of an aggregate JSON → the markdown baseline report on stdout |
| `bench/scripts/replay-manifest.sh` | New | G7: read a stored record + published aggregate, build the synthetic corpus, invoke `run-one.sh`, compare, emit the three-valued verification JSON |
| `bench/scripts/tests/tc017_run_batch_test.sh` | New | Batch enumeration, skip, incomplete-state, quarantine, failure-tolerance cases (AC-01..AC-05) |
| `bench/scripts/tests/tc018_aggregate_report_test.sh` | New | Aggregation classification, band math, flags, determinism, report content (AC-06..AC-21) |
| `bench/scripts/tests/tc019_replay_manifest_test.sh` | New | Preconditions, synthetic corpus shape, drift, three-valued verdict (AC-22..AC-27) |
| `bench/scripts/tests/run-all.sh` | Modify | Register the three new self-tests in sequence after `tc016` |
| `bench/scripts/testdata/aggregate/` | New | Fixture artifact roots, every record derived from the committed I-02 goldens (REQ-N-006) |
| `bench/README.md` | Modify | Add "Baseline aggregation, noise band, and replay": the aggregate schema, the band and acceptance-interval rules, the classification rules, the replay procedure, and the two preconditions of REQ-N-007 |
| `bench/baselines/<baseline_id>/aggregate.json`, `report.md` | New (published output) | The committed published baseline; `baseline_id` is `<variant_id>-<fixture_base_sha[:12]>-r<reps>` |
| `docs/plan/.../E40-F02-.../spec.md` | Modified by this spec step | TD-076 sanctioned amendment, four sites (see "TD-076 adjudication") |

No file under `internal/`, `cmd/`, or `tests/contracts/` is added or changed (REQ-N-003).

### Data model changes

**No shark database change** — no table, no column, no migration, no `CurrentSchemaVersion` bump (ADR-002). F03 introduces two derived documents, both reproducible from the artifact directory alone.

#### Metric registry and classes

Each aggregated metric is declared once with an id, a source expression over the I-02 record, and a class that determines its band rule.

| Metric id | Class | I-02 source |
|---|---|---|
| `oracle_f2p_resolved` | A | `oracle.f2p_resolved` |
| `oracle_repro_confirmed` | A | `oracle.repro_confirmed` (bug items only) |
| `quality_fmt_clean`, `quality_vet_ok`, `quality_tests_pass` | A | `quality.fmt_clean` / `.vet_ok` / `.tests_pass` (caveated, REQ-F-024) |
| `p2p_regressions_count` | B | `oracle.p2p_regressions_count` — the regression signal (REQ-F-014) |
| `p2p_removed_count` | B | `oracle.removed_count` |
| `lint_new_issues_count` | B | `quality.lint_new_issues_count` |
| `rejections_rework_loops` | B | `rejections.rework_loops` |
| `rejections_by_gate.<gate>` | B | `rejections.by_gate.<gate>`, absent key = 0 within a present block (REQ-F-015) |
| `loc_files_touched` | B | `loc.files_touched` |
| `loc_prod_added`, `loc_prod_deleted`, `loc_test_added`, `loc_test_deleted` | C | `loc.*` |
| `wall_clock_ns` | C | `timing.harness_wall_ns` |
| `run_total_duration_ns` | C | `runresult.total_duration_ns` |
| `tokens_input_total`, `tokens_output_total`, `cache_read_total`, `cache_creation_total` | C | sum over `stages[].usage.*` |
| `cost_usd_total` | C | sum over `stages[].usage.total_cost_usd` |
| `api_duration_ms_total` | C | sum over `stages[].usage.duration_api_ms` |
| `step.<status>.tokens_input` / `.tokens_output` / `.cost_usd` / `.duration_ns` | C | per-step, keyed on `stages[].status`, repeats summed (REQ-F-013) |

A Class C sum over `stages[].usage.*` is computed **only** when every agent stage in the run contributed that sub-field; if any agent stage's `usage` or that sub-field is absent, the run is excluded from that metric with reason `partial_usage`, never summed over a subset (REQ-N-005 — a partial sum reads as a low value, which is a fabricated measurement).

#### Record classification

| Classification | Rule |
|---|---|
| `complete` | Every family expected for the record's `outcome` is present |
| `explained_absence` | A family is absent **and** the record itself explains it: `outcome == "timeout"` (no post-run phase runs on that path); or `quality.toolchain_guard != "pass"`; or `errors[]` carries `postrun_check_aborted` |
| `anomaly` | A family is absent with no explanation on the record — specifically the F-4 case: `outcome != "timeout"` with `oracle`, `quality`, and `loc` all absent and no explaining error |

**Exclusion rules, applied per metric and always recorded:**

- `outcome == "timeout"` → the record contributes to the outcome distribution and the timeout rate **only**. It is excluded from every Class A, B, and C band, with reason `outcome_timeout`. This is load-bearing for `wall_clock_ns` in particular: a timeout record's `timing.harness_wall_ns` is the **cap**, not a measurement, so including it would make the wall-clock band an artifact of `--timeout` rather than of the workload. (The timeout golden confirms the record carries only `manifest`, `outcome`, `timeout_detail`, `timing`, `errors`, and `sources` — there is no `stages[]` or `runresult` block for the other bands to draw on either.)
- `explained_absence` → excluded from the absent families' bands with reason `toolchain_guard_abort` or `postrun_aborted`; the record still contributes to every family it does carry.
- `anomaly` → excluded from every post-run band with reason `unexplained_absence`, counted in the anomaly bucket, and the aggregation exits non-zero (REQ-F-009).
- Any exclusion appears in `excluded[]` for that `(item, metric)` with `run_key` and `reason`, so a band is never silently computed over fewer reps than the header implies. **An `excluded[]` entry is emitted for every metric in the registry applicable to the record's `item_type`, whether or not the record could have carried a value for that metric** — the list exists to answer "why is this band's `n` below the rep count", which is asked per metric, so a timeout record appears in the exclusion list of every applicable metric rather than only the ones whose family happened to be reachable on that path.

**Classification is not band participation.** A record classified `complete` may still be excluded from individual metrics — most commonly by `partial_usage`, where every family is present but one agent stage's `usage` sub-field is missing. `bench/README.md` documents that the same condition also produces an `envelope_parse_error` entry naming the field, so the exclusion is corroborated on the record itself rather than inferred. The report must therefore read the inventory counts as classification counts, never as band participation counts; per-metric `n` is the only participation number.

#### Band and acceptance interval

For each `(item_id, variant_id, metric)` over its contributing reps:

| Class | Observed statistics | Acceptance interval |
|---|---|---|
| A (binary) | `n`, `true_count`, `rate` | `accept_set`: the set of boolean values observed across reps |
| B (integer count) | `n`, `min`, `median`, `max`, `mean`, `spread_abs`, `spread_rel` | `[max(0, min − 1), max + 1]` |
| C (continuous) | `n`, `min`, `median`, `max`, `mean`, `spread_abs`, `spread_rel` | `r = max − min`; `r_eff = r` when `r > 0`, else `0.10 × abs(median)`; interval `[min − r_eff, max + r_eff]`, lower-clamped at 0 for non-negative metrics |

`spread_rel = spread_abs / median`, or `null` when `median == 0` (the report then states the absolute spread only). A metric that is identically zero across every rep has `r_eff = 0` and an exact `[0, 0]` interval — correct, because "the agent touched nothing" reproducing as "the agent touched something" is a real difference, not noise.

**Why the interval is derived rather than the raw range** (ADR-F03-04): for `n` independent samples the observed range covers a fresh sample with probability `(n−1)/(n+1)` — 50% at `n = 3`. Gating G7 on the raw `[min, max]` would therefore fail a *correct* replay about half the time per metric, and essentially always across the headline set. The interval is published **as part of the noise band**, in the same table and with its rule printed, so "within the published noise band" (epic G7, feature.md Success Metric) remains literally true of the number replay checks against.

**Corpus-level rollup** per metric: the `min`/`median`/`max` of the per-task `spread_rel`, plus for `oracle_f2p_resolved` a corpus pass rate (total passes over eligible runs) and a rep-slice band — the min/median/max of the three per-rep-index corpus pass rates. The rep index is an arbitrary slice, not a paired unit; the report states this, and the rollup is descriptive context only. The operative band for any comparison is the per-task one, because the comparison method is paired per task (`shark-bench-design.md §5`).

#### Aggregate document (`aggregate.json`)

Sorted keys, fixed list order, no timestamp (REQ-N-004).

| Block | Content |
|---|---|
| `schema_version` | Pinned; the aggregate's own version, distinct from I-02's |
| `baseline_id` | `<variant_id>-<fixture_base_sha[:12]>-r<reps>` |
| `input_digest` | sha256 over the sorted `"<sha256>  <relpath>"` lines of contributing records (REQ-F-019) |
| `provenance` | `model_ids[]`, `fixture_base_sha`, `variant_bundle_sha256`, `corpus_schema_version`, `shark_version`, `reps`, `uniform` (bool) and `divergences[]` when not |
| `inventory` | Per `run_key`: `classification`, `outcome`, and the families present |
| `outcomes` | Counts per outcome value, `timeout_rate`, `anomaly_count` |
| `anomalies[]` | `run_key` + missing families, for the F-4 bucket |
| `tasks[]` | Per item: `item_id`, `item_type`, `metrics{}` (statistics + interval + `excluded[]` + flags), `non_discriminative` |
| `corpus` | The corpus-level rollup described above |
| `flags` | `unusable_metrics[]`, `insufficient_reps[]`, `non_discriminative_tasks[]` |

#### Verification document (`verification.json`)

| Field | Content |
|---|---|
| `verdict` | `pass` \| `fail` \| `invalid` |
| `reasons[]` | Named conditions: `variant_bundle_drift`, `model_version_drift`, `corpus_drift`, `metric_outside_band` |
| `stored` / `replayed` | `run_key`, record paths, and the compared manifest identity fields |
| `baseline_id`, `input_digest` | The band the comparison was made against |
| `metrics[]` | Per metric: `replayed_value`, `accept_lo`/`accept_hi` (or `accept_set`), `verdict` |

### Interface contracts

```
bench/scripts/run-batch.sh --out <artifact_root>
                           [--corpus <corpus.yaml>] [--variant <id>]
                           [--reps <n>] [--timeout <seconds>]
                           [--items <id[,id...]>] [--reclaim-incomplete]
                           [--dry-run]           # prints classifications, invokes nothing

bench/scripts/aggregate-runs.sh --root <artifact_root> [--variant <id>]
                                # one aggregate JSON object to stdout

bench/scripts/report-baseline.sh --aggregate <aggregate.json>
                                # the markdown report to stdout

bench/scripts/replay-manifest.sh --record <path/to/record.jsonl>
                                 --band <aggregate.json>
                                 --out <replay_artifact_root>
                                 [--corpus <corpus.yaml>] [--skip-canary]
                                 # one verification JSON object to stdout
```

Exit statuses:

| Script | 0 | non-zero |
|---|---|---|
| `run-batch.sh` | Every pair completed or was skipped as already-complete | Any pair failed, or any `incomplete_prior_attempt` was skipped (named in the summary) |
| `aggregate-runs.sh` | Every record `complete` or `explained_absence`, provenance uniform | Any `anomaly` (aggregate still written, anomalies named) — or, for a structurally invalid record or non-uniform provenance, no aggregate written |
| `report-baseline.sh` | Report produced | Aggregate unreadable or unsupported `schema_version` |
| `replay-manifest.sh` | Verdict `pass` | Verdict `fail` or `invalid`, or a precondition failed before dispatch |

Artifact root layout, extending the layout `run-one.sh` already owns:

```
<artifact_root>/
├── <item_id>/<variant_id>/rep-<n>/record.jsonl   # produced by run-one.sh (I-02)
├── .incomplete/<item_id>/<variant_id>/rep-<n>-<seq>/   # quarantined prior attempts
├── batch-log.jsonl        # operator diagnostics only; never read by the aggregator
├── aggregate.json         # aggregate-runs.sh output
└── report.md              # report-baseline.sh output
```

Record enumeration is pinned to the bash glob `"$root"/*/*/rep-*/record.jsonl`, never `find` — the glob cannot match the dot-prefixed quarantine root, whereas `find` would descend into it and aggregate abandoned attempts.

**Synthetic replay corpus.** Written to a temp directory; `run-one.sh` resolves `seed_path` and `f2p.paths` against the corpus file's own directory (`os.path.join(corpus_dir, path)`), so both are emitted as absolute paths pointing into the source corpus tree, which `os.path.join` returns unchanged. The file carries the source corpus's `fixture:` and `p2p_sets:` blocks so it stays schema-shaped, with `fixture.base_sha` and the single item's `fixture_base_sha` both overwritten from the manifest — corpus.yaml's own INVARIANT requires those two to be byte-identical — the top-level `schema_version` set from `manifest.corpus_schema_version`, and the item's `p2p_set` from `manifest.p2p_set`. `items:` holds exactly the one replayed item; `negative_items:` is omitted.

The base-SHA ledger paths `run-one.sh` reads are derived from its **own** location (`$BENCH_DIR/corpus/ledgers/<sha>/`), not from the corpus file, so a synthetic corpus in a temp directory still resolves the real ledgers — which is exactly why REQ-N-007's retention precondition matters and why REQ-F-027 asserts the directory before dispatch.

### Key technical decisions

- **ADR-F03-01 — Four single-purpose scripts, each a pure function of its input.** Extends ADR-F02-01's driver/collector split: batch runner over run directories, aggregator over the artifact root, report over the aggregate, replay over a record plus a band. This is what makes AC-06 and AC-19 (byte-identical regeneration) statable at all, and it keeps ADR-002's "reproducible from the artifact directory alone" checkable rather than asserted. Rejected alternative: one `bench.sh` with subcommands — it would blur the pure-function boundary and make the determinism ACs about a mode rather than a program.
- **ADR-F03-02 — Replay pins inputs through a synthetic single-item `corpus.yaml`, never through a change to `run-one.sh`.** `run-one.sh` resolves `fixture_base_sha` from `corpus.yaml`'s current entry for `--item` (`:204-244`) with no per-field override; its only override surface is `--corpus`. A naive replay against the live corpus would silently use a *newer* SHA if a curator ever edited the item — defeating G7 with no warning. Building the manifest's pinned values into a synthetic corpus closes that hole using only the existing flag. `run-one.sh` is UAT-closed and ADR-F02-06's "invoke, never re-derive" posture binds F03 as a consumer.
- **ADR-F03-03 — Timeout and explained-absence records are excluded from bands by named rule; unexplained absence is an anomaly, not a datum.** Deviation from the naive read of "aggregate the artifacts": a timeout record's `timing.harness_wall_ns` is the cap, so averaging it in would make the published wall-clock band a function of the operator's `--timeout` choice. The F-4 anomaly bucket exists because a non-timeout record with every post-run family absent and `errors[]` empty is neither a pass nor a fail — the producer records no reason (TD-079), so the aggregator must not invent one. Rejected alternative: treat absent post-run families as failures — it would convert a harness defect into a measured quality regression, exactly the fabrication REQ-N-005 forbids.
- **ADR-F03-04 — The published band carries both the observed spread and a derived acceptance interval.** The observed `min`/`median`/`max` is the G5 deliverable and stays uninflated. The acceptance interval is derived from that same data by a printed rule, because a raw 3-sample range covers a fresh sample only `(n−1)/(n+1) = 50%` of the time, so gating G7 on it would fail correct replays. Both live in the same published table, so replay still checks against "the published noise band". The `0.10 × median` floor for zero-spread continuous metrics is an arbitrary-but-fixed constant, named as such in the report rather than presented as derived. Rejected alternative: more reps — feature.md and epic §2 place that beyond Phase 1 scope.
- **ADR-F03-05 — The replay verdict is three-valued, and input drift is `invalid`, not `fail`.** This **narrows** the research report's Decision 4 rather than overturning it: its operative claim — that no second hard-fail gate is needed for *bundle identity* — survives, and bundle-hash mismatch remains the identity gate. But a bundle pins the model *alias* (`OrchestratorAction.model`, e.g. `sonnet`) while `manifest.model_ids` carries the provider-resolved snapshot id from `modelUsage`; a hash match therefore does not imply the same model version, which is precisely why the manifest records both. uat-plan.md's non-functional evidence is explicit that "a comparison spanning unpinned model versions is reported as invalid rather than as a result", and the epic's risk list names model-version drift. Collapsing drift into `fail` would report an untested condition as a reproducibility failure. Both checks are necessarily **post-hoc** — the values come from the fresh record's manifest, so they are evaluated after the run and after its API spend; pre-checking would mean re-deriving `run-one.sh`'s install-and-hash step outside it, which ADR-F02-06 forbids.
- **ADR-F03-06 — `oracle.p2p_regressions_count` is the only regression signal.** `shark-bench-design.md §4`'s Defects row already defines it as the full-suite diff against the base-SHA ledger. `quality.tests_pass` reads `false` on essentially every run because F01's fixture carries a deliberately permanently-failing regression probe (T-004), so reading it as a regression signal would report a constant as a finding. It is still reported, under a gate-outcome heading with that caveat attached.
- **ADR-F03-07 — Batch resume treats "directory present, `record.jsonl` absent" as its own state, and quarantines rather than deletes.** `run-one.sh`'s overwrite refusal keys only on `record.jsonl` (`:197-200`) while it `mkdir -p`s the run directory (`:419,588`), so a crashed prior attempt (TD-078) leaves `run/` and `post/` populated but is invisible to that guard (TD-080). Re-invoking into it, then timing out, would leave the collector grafting the *prior* attempt's oracle/quality/LOC onto a fresh timeout record — a silently wrong datum in the band. Skip-and-report is the default so an unattended batch still completes (G2); quarantine is opt-in and is a move, never a delete, so evidence of the crash survives for diagnosis.
- **ADR-F03-08 — Self-test fixtures derive from the committed I-02 goldens.** Hand-authoring aggregator fixtures would encode an *assumed* record shape, which is UAT F-1's exact defect class (an enforcing test written to the shape someone believed shipped) reproduced one layer down in the consumer. Deriving from `e40_i02_golden_record.jsonl` and `..._timeout.jsonl` means a future producer change that breaks the goldens breaks F03's tests too, instead of leaving them green against a stale assumption.
- **ADR-F03-09 — No Go change, and no second I-02 contract test.** The epic's single Phase 1 Go change is spent on F04. `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` is F02's and stays the single enforcement point for the I-02 shape; F03's consumer-side task references it verbatim and adds no twin. Consumer-side behavioral coverage lives in `tc018`/`tc019`, which exercise F03's own scripts against golden-derived fixtures.

### Integration with existing code

F03 is additive at the filesystem boundary and touches no existing script's behavior.

- **`run-one.sh`** — invoked as a subprocess by `run-batch.sh` and `replay-manifest.sh` with its documented flags only (`--item`, `--variant`, `--rep`, `--timeout`, `--out`, `--corpus`, `--skip-canary`). Both callers resolve it as `$SCRIPT_DIR/run-one.sh`, matching the sibling-invocation convention every other bench script already uses, and both honor a `RUN_ONE_BIN` override for the self-tests' stub seam — the same testability pattern `run-one.sh` itself uses for `SHARK_BIN`, and the pattern whose *absence* of a `SCRIPT_DIR` default caused TD-077, so the default here is the sibling path, not a bare name.
- **`bench/corpus/corpus.yaml`** — read for the item list (`items:` only) and, on replay, for the source item's `seed_path` and `f2p` block. Never rewritten; the replay manifest is a separate temp file.
- **`bench/corpus/ledgers/<sha>/`** — never read or written directly by F03; only asserted to exist before a replay dispatch.
- **`tests/contracts/e40_i02_artifact_contract_test.go`** — referenced as the contract gate, not modified.
- **`bench/scripts/tests/run-all.sh`** — three registration lines appended after `tc016`, matching the existing sequential list convention.

---

## Cross-feature interactions

### Consumes: I-02 — Metric collection and artifact schema

| Property | Contract |
|---|---|
| Producer | E40-F02 Bench harness: run driver and metric collection |
| Shape source | `architecture.md#metric-collection-and-artifact-schema` |
| Contract test | `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` |
| Payload | One JSONL record per run: manifest block (item, variant, rep, SHAs, exact model IDs, timeout cap), per-stage records, post-run check results, and a rollup |
| Style | File artifact |
| Gate mode | `live`, as assigned by [the interaction map](../E40-interaction-map.md) |

Shape source and contract test are copied **verbatim** from E40-F02's `spec.md` "Produces: I-02" and the [interaction map](../E40-interaction-map.md)'s I-02 row. This discharges the spec-side half of the obligation recorded as a decision note on E40-F03 (2026-08-06 21:20); `task_generation` must still create at least one task declaring `I-02: consumes` that copies both pointers verbatim and owns the real caller path that aggregates records — `aggregate-runs.sh`.

**No twin test.** The consumer-side task references `TC-001` — the same test, in the same file, owned by F02. F03 adds no `tests/contracts/` file and no second I-02 validator (ADR-F03-09, REQ-N-003).

**The consumed shape is the shipped one.** Per this spec's TD-076 adjudication, the `sources` block is conditionally present with a complete emitting set of `oracle`/`quality`/`loc` (`"postrun"`) and `stalled_stage` (`"liveness"` \| `"scratch_db"`); `timing`, `stages`, and `rejections` carry no entry by design. `aggregate-runs.sh` determines family presence from the family block itself, never from `sources` (REQ-F-007), so this narrowing changes no aggregation behavior — it removes an expectation that would otherwise have been coded as a validity check against records that are correct.

**The uat-plan's I-02 scenario binds this feature:** "a record missing a metric family fails aggregation loudly instead of being silently averaged away." REQ-F-008/AC-08 make it executable, and the `explained_absence` classification is what keeps "loud" from meaning "unusable" on the timeout and toolchain-guard paths, where an absent family is the producer behaving correctly.

**Known producer gaps read, not fixed.** F03 consumes the I-02 surface as shipped, including four tracked defects: TD-078 (F-3, a run that fails mid-post-run emits no record — appears to F03 as a missing pair, handled by REQ-F-003), TD-079 (F-4, the anomaly bucket), TD-080 (F-5, the dirty-rerun window REQ-F-003 defends against from the consumer side), and TD-081 (F-6, fmt/vet attribution, caveated by REQ-F-024). None is in F03's scope to repair.

---

## Cross-epic integrations

**None assigned — verified, not omitted.** Both maps were checked row by row:

| Row | Owning / consuming feature per the maps | F03 involvement |
|---|---|---|
| X-07 (E22 → E40) | Owning feature **E40-F02**; consumer named as E40-F02 | None. F03 never parses `RunResult`, `StageLog`, or the transcript byte format — it reads only the collector's `record.jsonl`, and the canary that guards X-07 is invoked by `run-one.sh`, not by F03. |
| X-08 (E40-F04 → E22) | Owning feature **E40-F04**; activation owner E40-F04 itself | None. |
| X-09 (E27 → E40 Phase 2 G1) | Owning feature **TBD — Phase 2, no feature exists**; status `proposed` | None. Phase 2. |

[`E40-cross-epic-map.md`](../E40-cross-epic-map.md) and [`docs/product/cross-epic-integration-map.md`](../../../product/cross-epic-integration-map.md) agree on all three rows, and neither assigns a row to E40-F03. No row is mirrored here because none applies; this feature adds no new cross-epic seam, since every upstream surface it touches is reached through a bench script that already owns that seam.

---

## Durable unresolved decisions

**Q005 — Does a G7 replay detect a curator edit to a held-back F2P test or seed file?** (`draft`, minted by this specification step.)

Replay pins `fixture_base_sha`, `variant_bundle_sha256`, `corpus_schema_version`, and `p2p_set` from the stored manifest and passes them through the synthetic corpus. But `run-one.sh` resolves `seed_path` and `f2p.paths` against the corpus file's directory, so the held-back test files and the seed file are read from the **live** corpus tree at replay time, and the I-02 record carries no content hash of either. A curator edit between the original run and the replay would change the oracle silently.

- **Phase 1 posture, decided now**: option (ii) — corpus-item immutability for any SHA a published manifest references is a documented precondition (REQ-N-007), stated alongside the ledger-retention precondition in `bench/README.md`. Nothing violates it today: corpus rotation policy is deferred to Phase 3 (epic.md:126, shark-bench-design.md:103), which is the same reason the retention precondition is currently safe.
- **Closure options carried**: (i) a manifest content-hash field over the seed plus sorted held-back F2P files — a producer change, therefore Phase 2; (iii) accept as a named replay limitation.
- **Why not a Phase 1 producer change**: it would reopen a UAT-closed feature's record schema for a risk that the same deferred rotation policy currently prevents, and F03 owns no producer surface.
- **Closure owner**: Phase 2 planning, at the point corpus rotation is designed — the same decision point that must close the retention precondition.

**Not open, recorded so it is not re-litigated:** the coarseness of a 3-rep band is a *settled scope boundary*, not an unresolved decision. feature.md Out of Scope states "Statistical significance testing beyond min/median/max spread (revisit if 3-rep bands prove too coarse)" and epic §2 carries the matching explicit non-goal on sensitivity ("a 10×3 matrix detects coarse effects... the noise band exists precisely to stop such claims being made from this data"). ADR-F03-04 works within that boundary rather than reopening it.

**Q003 and Q004** are E40-F02 and Phase 2 concerns respectively and are not open decisions for this feature: Q003 was closed empirically by F02's REQ-F-021 capture and is recorded in `bench/README.md`; Q004 is constrained out of Phase 1 by entity-type scoping (ADR-005, tasks and bugs only).

---

*Last Updated*: 2026-08-07
