---
feature_key: E40-F03-baseline-report-and-noise-band
epic_key: E40
type: test-plan
tier: STANDARD (12/27)
date: 2026-08-07
---

# Test Plan: E40-F03 — Baseline report and noise band

**Created:** 2026-08-07
**Feature spec:** [spec.md](spec.md) (`type: combined-spec`, STANDARD 12/27; 30 REQ-F, 7 REQ-N, 28 ACs, 9 ADRs)
**Research report:** [research-report.md](research-report.md)
**Parent epic UAT plan:** [../uat-plan.md](../uat-plan.md)
**Interaction map:** [../E40-interaction-map.md](../E40-interaction-map.md)
**Sibling test plans (pattern precedent):** [../E40-F02-.../test-plan.md](../E40-F02-bench-harness-run-driver-and-metric-collection/test-plan.md), [../E40-F04-.../test-plan.md](../E40-F04-shark-run-live-progress-and-per-run-log/test-plan.md)
**Task specs:** none exist yet — this is feature-level test planning ahead of `task_generation`, mirroring F02's and F04's own treatment. `spec.md`'s 28-AC table is the AC list; task-spec-drift steps (Step 2/3 of the generic workflow, task-spec vs. feature spec) have no input and are skipped.
**Status:** APPROVED — see "Red-team review" and "Verdict" at the end.

---

## Scope and drift analysis (Steps 1–3)

`spec.md` is a `type: combined-spec` for E40-F03, already checked for drift
against its own upstream sources (`epic.md`, `architecture.md`,
`shark-bench-design.md`, `E40-interaction-map.md`, `E40-cross-epic-map.md`,
`uat-plan.md`, F02's `spec.md`) inside its own "Context" and "Traceability"
sections. This test plan re-derives that traceability independently rather
than copying it, and adds one thing `spec.md`'s own traceability table does
not do: a REQ-indexed check, not just an AC-indexed one (see "REQ-indexed
gaps closed by this plan" below — an AC-indexed pass alone would have missed
several of these).

- Every REQ-F-/REQ-N-* in `spec.md`'s Requirements section (lines 93–147)
  traces to at least one AC in the spec's own Traceability table (lines
  184–197). Cross-checked directly: no REQ found without a matching AC row,
  no AC found that names no REQ.
- Every epic goal F03 claims to serve (G2, G4, G5, G7) is named in
  `epic.md` §2 and in `uat-plan.md`'s criterion column (UAT-01, UAT-07).
  Cross-checked against `uat-plan.md`'s table directly: UAT-01 ("Unattended
  baseline batch produces a report with a noise band") and UAT-07 ("Results
  reproduce from a stored manifest") map one-to-one onto F03's Scope items
  1–4 and Scope item 5, respectively. No scope creep (no REQ claims a Phase
  2 capability — variant comparison, statistical significance beyond
  min/median/max — that `feature.md` Out of Scope and `epic.md` §2 both
  exclude) and no scope narrowing found.
- **TD-076 is adjudicated and executed, not open.** `spec.md`'s own "TD-076
  adjudication" section states option (b) is decided and the four sites in
  F02's `spec.md` are already edited. This test plan treats the *shipped*
  `sources` shape (the narrowed, five-value set) as the one to test against
  — verified directly against both committed goldens (below), not assumed
  from prose. No test in this plan requires a `sources.timing`,
  `sources.stages`, or `sources.rejections` entry to exist.
- **Q005 is a documented Phase 1 precondition, not spec drift.** "Durable
  unresolved decisions" records corpus-item immutability (REQ-N-007) as
  option (ii), closed for Phase 1. This plan covers REQ-N-007's two
  preconditions as a `bench/README.md` content check (TC-018v), the same
  treatment F02's test plan gave its own precondition-documentation
  obligations (mirrors F02 TC-017).
- No BA or architecture refinement is required. `spec.md`'s Architecture
  section already names every script this plan needs a test surface for
  (`run-batch.sh`, `aggregate-runs.sh`, `report-baseline.sh`,
  `replay-manifest.sh`) with exact reserved self-test file names
  (`tc017`/`tc018`/`tc019`), so this plan does not invent new deliverables
  the way F01's plan had to — it fills in test design against deliverables
  `spec.md` already authorizes, matching F02's own "no PRD completeness
  gap" finding.

### REQ-indexed gaps closed by this plan

`spec.md`'s own Traceability table is AC-indexed and every REQ maps to at
least one AC, but several REQ **sentences** — narrower than the REQ's own
AC — have no dedicated test anywhere in the AC list. Found by rereading
every REQ's full text against its cited AC's literal wording, not by
trusting the table row. Each is closed here with an owner (this plan) and a
TC, not deferred:

| REQ text with no AC-level test | AC it hides under | Gap | Closed by |
|---|---|---|---|
| REQ-F-007, second sentence: "A family's presence is determined by the family block itself, never by the `sources` block." | AC-06 (only tests artifact-only reading) | This is the operative half of the TD-076 adjudication's consumer-side consequence — no AC exercises a record with a family present but no matching `sources` entry, or `sources` absent entirely | TC-018q |
| REQ-F-011's five uniformity fields (`model_ids`, `.fixture_base_sha`, `.variant_bundle_sha256`, `.corpus_schema_version`, `.shark_version`) | AC-11 (states only "two different `manifest.model_ids` values") | An implementation hardcoding only the `model_ids` check would pass AC-11 as literally written | TC-018f, table-driven over all five |
| REQ-F-019 (`input_digest` is a **computed** sha256 over sorted `"<sha256>  <relpath>"` lines) | AC-21 (only asserts the aggregate's `input_digest` is echoed into the report) | No AC proves the digest is actually *computed* from content — a constant string passes AC-21 by construction | TC-018r |
| REQ-F-023 ("naming E40-F01") | AC-14 (only requires the section exist) | An empty or genetically-named "corpus feedback" heading would pass AC-14 as written | TC-018i |
| REQ-N-007's corpus-item-immutability half (seed/F2P files) | No AC at all — only the ledger-retention half is under AC-22 | Immutability is a documented precondition (Q005), asserted where cheaply assertable (README content) | TC-018v |
| `report-baseline.sh`'s Interface-contracts exit table: non-zero when "Aggregate unreadable or unsupported `schema_version`" | No AC — AC-07 covers the **record's** `schema_version`, not the **aggregate's** | Distinct failure mode, same table, no AC row | TC-018u |
| Record-enumeration glob (`"$root"/*/*/rep-*/record.jsonl`, "never `find`") | No AC exercises the discriminating case | A `find`-based implementation would pass every other AC and still descend into `.incomplete/` | TC-018s |
| `batch-log.jsonl`: "operator diagnostics only; never read by the aggregator" | No AC | A batch log the aggregator can't even open must not fail aggregation | TC-018t |
| RUN_ONE_BIN default resolution — "`SCRIPT_DIR` sibling" (binding decision note on E40-F03; the exact defect class TD-077 named for `CANARY_BIN`) | AC-01..05, AC-22..27 (all use an *overridden* `RUN_ONE_BIN`) | Every AC test overrides `RUN_ONE_BIN`; a script defaulting to a bare PATH name would pass every one of them and fail in every real environment | TC-017f, TC-019h |
| Class C acceptance-interval's three branches (`r>0`; `r==0, median≠0`; identically-zero) and `spread_rel=null` when `median==0` | AC-12 (asserts the invariant, not each branch) | A single-branch implementation (e.g., always `r_eff=r`) could pass a naive AC-12 test that only exercises the `r>0` case | TC-018g, table-driven over three branches |
| AC-25/AC-26's uncovered fourth combination: bundle **and** model_ids both drift | AC-25, AC-26 (only test one axis differing at a time) | Two implementers could disagree on whether both-drift reports one reason or two | TC-019g, resolved below |

**Resolution for the both-drift-differ combination (TC-019g), pinned here
for task generation, matching F02's own `stage_join_error`/
`transcript_missing` resolution pattern (a plan-level reading, not a spec
defect):** `verification.json`'s `reasons[]` field is documented as an
array, and REQ-F-029 says a mismatch "of either" yields `invalid`. Read
together, when both `variant_bundle_sha256` and `model_ids` differ, the
verdict is `invalid` and `reasons[]` contains **both**
`variant_bundle_drift` and `model_version_drift`, each naming its own
expected/actual pair — never one reason silently subsuming the other. This
is the natural reading of REQ-F-029's per-field comparison plus the
array-typed `reasons[]` field; it is not a new rule invented by this plan.

### Ambiguity findings (Step 5)

**No AC is an open-ended robustness assertion.** Every AC names a concrete
input shape, a concrete rule, and a concrete observable output (a file's
presence/absence, an exit code, a named field, a specific numeric
invariant). The closest candidates and why each is closed, not open:

- **AC-01 "runs to completion unattended... no invocation prompts for
  input."** Closed by REQ-F-001's own concrete mechanism: the test asserts
  the invocation is run with stdin redirected from `/dev/null` and still
  completes (same technique F02's TC-014a uses for its own "no stdin read"
  assertion) — not a vague "never blocks."
- **AC-15 "unusable at the current rep count."** Closed by REQ-F-018's
  concrete formula (`spread_abs > mean`), not a subjective judgment.
- **AC-24 "corpus_drift entry naming both values."** Closed by REQ-F-028's
  named field shape; "both values" is the live value and the manifest
  value, not an open list.

**One clarity note, resolved by context, not a defect.** REQ-F-024's third
caveat text ("`quality.fmt_clean`/`.vet_ok` not being provably
agent-attributable until REQ-F-016's null-with-reason value ships") cites
"REQ-F-016" — but *this* spec (F03) also defines its own REQ-F-016 (the
acceptance-interval publication rule, unrelated to quality gates). Read in
context (the parenthetical "F-6 / TD-081" ties it to E40-F02's UAT finding
F-6), this is unambiguously a cross-feature reference to **F02's**
REQ-F-016 (F02's quality-gate null-with-reason requirement, still
unshipped), not F03's own. Flagged here so the report-content task (TC-018o)
and its implementer do not mistakenly hyperlink the caveat to F03's own
REQ-F-016 section. Not a spec defect — the caveat text is correct, only
the two same-numbered REQs across sibling features could confuse a reader
skimming this plan out of context.

---

## AC Test Matrix

Every AC in `spec.md` has at least one test case below. `TC` values point
into one of the four new bash self-tests (`tc017_run_batch_test.sh` /
`tc018_aggregate_report_test.sh` / `tc019_replay_manifest_test.sh`,
matching `spec.md`'s own reserved file names) or the meta-check
`tc020_zero_go_change_test.sh` (AC-28, this plan's own addition — `spec.md`
does not reserve a fourth file name for it, but AC-28's assertion needs a
runnable home; registering a fourth, small script is proportionate rather
than folding a `git diff --stat` check into an unrelated script). Full
Caller-Path Contracts are in their own section below to avoid repeating the
same entrypoint 28 times.

### `run-batch.sh` (AC-01..AC-05) — `tc017_run_batch_test.sh`

| AC | Requirement(s) | TC | Technique | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|
| AC-01 | REQ-F-001, REQ-F-005 | TC-017a | Contract-surface enumeration | `run-batch.sh --out <tmp> --corpus bench/corpus/corpus.yaml --reps 3` against the **real** 10-item `corpus.yaml` (5+ per-type mix, `negative_items:` present too), `RUN_ONE_BIN` pointed at a stub emitting a `completed`-outcome record (derived from the golden) on every call, invoked with `</dev/null` | Exits 0; stdout/stderr summary names all 30 `(item, variant, rep)` classifications; stub invocation count is exactly 30 (one per pair); none of the 3 `negative_items:` entries appear in the summary (REQ-F-005). Negative: a `negative_items:`-derived id passed via `--items` is rejected/ignored, never dispatched. |
| AC-02 | REQ-F-002 | TC-017b | State transition (resume) | Same fixture, `--out` pointed at a root **pre-populated** with all 30 pairs' `record.jsonl` already present (from AC-01's own output, reused) | Second invocation: stub invocation count is exactly **0** (the stub's own `STUB_RUN_ONE_LOG` line count, the literal mechanism AC-02's text names: "asserted by a stubbed driver that records its call count"); exits 0. Negative: any non-zero stub call count fails the test outright. |
| AC-03 | REQ-F-003 | TC-017c | Decision table (directory state × `--reclaim-incomplete`) | `--out` root with exactly one pair's directory containing `run/` and `post/` subdirectories but no `record.jsonl` (hand-constructed, matching `run-one.sh`'s real on-disk shape after `mkdir -p run post`), default flags (no `--reclaim-incomplete`) | Stub invocation count excludes that pair (29, not 30); the pair is named in the summary with classification `incomplete_prior_attempt`; every other pending pair still runs; exit code is **non-zero**. Negative: the stale `run/`/`post/` directory's contents are byte-unchanged after the invocation (no silent cleanup without the flag). |
| AC-04 | REQ-F-003, REQ-F-006 | TC-017d | Decision table (same fixture, flag flipped) | Identical fixture to AC-03, `--reclaim-incomplete` added | The stale directory is **moved** (not copied, not deleted) to `.incomplete/<item>/<variant>/rep-<n>-<seq>/`; its contents are byte-identical there (`diff -r` old snapshot vs. new location); the pair's original target path no longer exists; stub invocation count for that pair is exactly 1, invoked against the now-absent (fresh) target path. Negative: the source `.incomplete/` entry is never itself a target `run-one.sh` writes back into. |
| AC-05 | REQ-F-004 | TC-017e | Attack-class enumeration (fault tolerance) | 3-item synthetic corpus, `RUN_ONE_BIN` stub configured to exit non-zero for exactly one `(item, variant, rep)` triple (env-var-selected) and succeed for the rest | Stub invocation count equals the full pending pair count (the batch attempted every pair, including the two after the failing one in enumeration order); the failing pair is named in the summary with a failure classification; exit code non-zero (per the Interface-contracts exit table: "Any pair failed... named in the summary"). Negative: a batch run with the failure at the **last** enumerated pair (not the first) still shows every prior pair completed — proves the loop doesn't short-circuit regardless of failure position. |
| — | RUN_ONE_BIN default resolution (binding decision note; TD-077's defect class) | TC-017f | Contract-surface enumeration (constructor default) | `run-batch.sh` invoked with `RUN_ONE_BIN` **unset**, `PATH` containing only a directory holding a script named `run-one.sh` placed as `run-batch.sh`'s own sibling (`$SCRIPT_DIR/run-one.sh` — i.e., `bench/scripts/run-one.sh` itself, or a same-directory stand-in for the test) | The batch resolves and invokes the sibling-path script, not a bare-PATH-name lookup that could resolve to nothing or to an unrelated binary. Negative: a `run-batch.sh` implementation defaulting `RUN_ONE_BIN` to the bare string `run-one.sh` (relying on `PATH`) fails this test in an environment where `bench/scripts/` is not on `PATH` — exactly TD-077's `CANARY_BIN` defect, one script over. |

### `aggregate-runs.sh` (AC-06..AC-18) + `report-baseline.sh` (AC-19..AC-21) — `tc018_aggregate_report_test.sh`

| AC | Requirement(s) | TC | Technique | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|
| AC-06 | REQ-F-007, REQ-N-004 | TC-018a | State transition (re-run stability) + Boundary (locale) | A golden-derived fixture root (3 completed reps of one item, see "Fixture derivation" below), `aggregate-runs.sh` invoked **twice** as two fully separate process invocations under `LC_ALL=C`; a second pair of invocations under `LC_ALL=en_US.UTF-8` runs only when `locale -a` (case-insensitively) lists that locale, else that half is skipped and logged, mirroring `transcript_test.go`'s Windows-skip convention — a container with only `C` installed must not read as a silent pass | The `LC_ALL=C` pair is byte-identical, including key order, on every environment. The `LC_ALL=en_US.UTF-8` pair, when run, is byte-identical to the `LC_ALL=C` pair. Negative: an implementation relying on the shell's default glob/sort collation (rather than an explicit sort) would diverge between the two locale pairs when both run — the discriminating case a same-locale-only rerun test would miss; a skipped UTF-8 half is recorded as skipped, not silently passed. |
| AC-07 | REQ-F-010, REQ-N-005 | TC-018b | Attack-class enumeration (malformed input) | A fixture root with one record whose `schema_version` is `"99.0"` (an unsupported value) | `aggregate-runs.sh` exits non-zero, names the offending file path on stderr, and writes **no** `aggregate.json` (`ls` confirms absence). Negative: a partial/truncated `aggregate.json` left on disk from a failed run is itself a failure (must be absent, not empty or partial). |
| AC-08 | REQ-F-008, REQ-F-009, REQ-N-005 | TC-018c | Decision table (F-4 anomaly class) | A fixture: `outcome: "completed"`, `oracle`/`quality`/`loc` blocks **and** `sources.oracle`/`.quality`/`.loc` all absent, `errors[]` empty (the exact F-4 shape) | Record classified `anomaly`, appears in `aggregate.json`'s `anomalies[]` with its `run_key` and the three missing family names; contributes to **no** band for any metric; aggregator exits non-zero. Negative: the same fixture with one family present (e.g. `oracle` populated, `quality`/`loc` still absent) still classifies `anomaly` — the rule is "no explanation," not "zero families," and the record contributes only to families it actually carries excluded elsewhere per REQ-F-009's own text ("still contributes to every family it does carry" — this sub-case additionally proves partial-anomaly is still anomaly-classified, not silently downgraded to `explained_absence`). |
| AC-09 | REQ-F-009 | TC-018d | Decision table (outcome=timeout exclusion) | One item, 3 reps: rep-1/rep-2 derived from the **completed** golden (item id/rep/run_key rewritten to match), rep-3 derived from the **timeout** golden (same item id, rep 3) | Rep-3 contributes to `outcomes` counts and `timeout_rate` only; appears in `excluded[]` with reason `outcome_timeout` for **every** registry metric applicable to a `task` item — including `oracle_f2p_resolved`/`quality_*` families the timeout record structurally never carried (per REQ-F-009's "whether or not the record could have carried a value"); no band value for any metric equals `60`s (the fixture's `timeout_cap_s`, matching the timeout golden's carried cap) or any value coincidentally equal to the cap; `wall_clock_ns`'s `n == 2` (3 reps minus the one timeout). |
| AC-10 | REQ-F-009 | TC-018e | Decision table (explained_absence: toolchain_guard) | A fixture: `outcome: "completed"`, `quality.toolchain_guard` overwritten from `"pass"` to `"go_version_mismatch"`, `oracle`/`loc`/`quality.lint_new_issues*` blocks absent (matching what `run-one.sh`'s pinned order actually produces on a guard abort) | Classified `explained_absence`; excluded from the oracle/quality/LOC bands with reason `toolchain_guard_abort`; aggregator exits **zero** (no anomaly present in this fixture set). Negative: a fixture set that also contains one genuine anomaly record alongside this one still exits non-zero overall — proves exit status is a whole-aggregation property, not per-record. |
| AC-11 | REQ-F-011 | TC-018f | Boundary/equivalence (5-axis table-drive) | **Five** two-record fixture pairs, one per uniformity field (`manifest.model_ids`, `.fixture_base_sha`, `.variant_bundle_sha256`, `.corpus_schema_version`, `.shark_version`), each pair identical except the one field under test differs between the two records | Each of the five: `provenance.uniform == false`, `provenance.divergences[]` names the differing field and both values, exits non-zero, **no** other block (`tasks[]`, `corpus`) is populated as if the batch were valid (REQ-F-011: "never published as a result"). Negative: a sixth fixture pair identical on all five fields but differing on an **unlisted** field (e.g. `manifest.rep`, which is expected to vary) does **not** trigger a uniformity violation — proves the check is scoped to exactly the five named fields, not "any manifest field differs." |
| AC-12 | REQ-F-012, ADR-F03-04 | TC-018g | Boundary Value Analysis (3 interval branches) + invariant check | Three 3-rep fixture sets varying one Class C metric: (i) spread present (`loc.prod_added`: 10, 12, 15 → `r=5>0`); (ii) zero spread, nonzero median (`timing.harness_wall_ns` held identical and nonzero across reps → `r=0`, `median≠0`); (iii) identically-zero (`loc.test_deleted`: 0, 0, 0 → `median=0`) | (i): `accept_lo = min−r_eff = 5`, `accept_hi = max+r_eff = 20` (`r_eff=r=5`). (ii): `r_eff = 0.10 × |median|`, interval centered symmetric on the shared value. (iii): `accept_lo = accept_hi = 0` exactly (not `±0.10×0`, same numeric result but asserted as the *documented* zero-spread rule, not a coincidence of the formula); `spread_rel` is `null` in this fixture (median 0). For every Class B/C metric across all three: `accept_lo <= min` and `accept_hi >= max` holds, and `n=3`/`min`/`median`/`max`/`mean`/`spread_abs`/`spread_rel` are all present. Class A metric (`oracle_f2p_resolved`) in the same fixture set carries `accept_set` (the observed boolean set), not a numeric interval — negative: no Class A metric ever carries `accept_lo`/`accept_hi` keys. |
| AC-13 | REQ-F-016 | TC-018h | Boundary Value Analysis (n=1 boundary) | A fixture where exactly one metric family has only 1 contributing rep (the other two reps' records are `explained_absence`-excluded from that one family only, still contributing elsewhere) | That metric's block carries the `insufficient_reps` flag and **no** `accept_lo`/`accept_hi`/`accept_set` keys; `n == 1`. Negative: every *other* metric in the same task, still at `n=3`, publishes its interval normally — proves the flag is per-metric, not per-task. |
| AC-14 | REQ-F-017, REQ-F-023 | TC-018i | Equivalence Partitioning (all-true / all-false) | Two 3-rep fixture sets for the same item: one with `oracle.f2p_resolved: true` on every rep, one with `false` on every rep | Both flagged `non_discriminative` in `aggregate.json`'s `flags.non_discriminative_tasks[]`; `report-baseline.sh`'s rendered markdown contains a heading whose text **names "E40-F01"** verbatim (REQ-F-023's own wording, not merely "a corpus-feedback section exists" — the gap this plan's REQ-indexed check found). Negative: a 3-rep set with a mixed `true`/`true`/`false` result is **not** flagged non-discriminative. |
| AC-15 | REQ-F-018 | TC-018j | Boundary Value Analysis (`spread_abs > mean`) | A fixture with fixture values `1, 1, 100` for a Class B metric (`spread_abs = 99`, `mean ≈ 34`, `99 > 34`) | That metric flagged in `flags.unusable_metrics[]`. Negative: a boundary fixture with `spread_abs == mean` exactly (not `>`) is **not** flagged — proves the comparison is strict, not `>=`. |
| AC-16 | REQ-F-013 | TC-018k | Decision table (repeat `stages[].status`) | A fixture whose `stages[]` contains two entries both with `status: "in_development"` (a rework loop — status re-entered), each carrying distinct nonzero `usage.input_tokens`/`.output_tokens`/`.total_cost_usd` | The `step.in_development.tokens_input`/`.tokens_output`/`.cost_usd`/`.duration_ns` metrics equal the **sum** of both occurrences, not either one alone. Negative: a third stage with a *different* status (`in_qa`) in the same run is not folded into the `in_development` step's total. |
| AC-17 | REQ-F-014, ADR-F03-06 | TC-018l | Contract-surface enumeration (source-field lock) | A fixture where `oracle.p2p_regressions_count` and `quality.tests_pass` are deliberately set to **different, uncorrelated** values across reps (e.g. `p2p_regressions_count: 0,1,0` while `tests_pass: false,false,false` on every rep, matching T-004's real-world shape) | The aggregate's regression field tracks `p2p_regressions_count`'s per-rep values exactly; no field anywhere in the aggregate is computed from `quality.tests_pass`. Negative: re-aggregate the identical fixture with only `tests_pass` flipped (`true` on every rep instead of `false`) and diff the two `aggregate.json` outputs — the **regression field** (`p2p_regressions_count`-derived) must be byte-identical across both runs; only the `quality_tests_pass` Class A metric's own block (`true_count`/`rate`/`accept_set`, which legitimately tracks `tests_pass` as its own registry entry) may differ. A whole-document byte-identity assertion would be self-defeating here, since `quality_tests_pass` is itself a registered Class A metric and is expected to change when its source value changes — this TC scopes the assertion to the regression field alone, which is the actual property REQ-F-014 states. |
| AC-18 | REQ-F-015 | TC-018m | Decision table (present-with-omitted-key vs. whole-block-absent) | Two fixtures: (a) 3 reps, `rejections.by_gate` present on every rep but one rep omits a gate key (`in_qa`) that another rep carries; (b) one rep's whole `rejections` block absent | (a): the gate-omitting rep contributes `0` for that gate to `rejections_by_gate.in_qa`'s statistics (not excluded, not treated as missing data). (b): that rep contributes nothing to any `rejections_*` metric and is listed in `excluded[]` for those metrics with a named reason. Negative: case (a)'s zero-contribution must still count toward `n` (the metric's `n` includes the omitting rep, since the block itself was present) — distinguishing "0" from "excluded" is the point of this AC. |
| AC-19 | REQ-N-004 | TC-018n | State transition (re-run stability) | A fixed `aggregate.json` (from AC-12's fixture), `report-baseline.sh` invoked twice as separate processes | Byte-identical markdown output both times; `grep -E '[0-9]{4}-[0-9]{2}-[0-9]{2}|[0-9]{2}:[0-9]{2}:[0-9]{2}'` over the output finds no date/time-shaped string. Negative (F02's counter-factual, reused): a version that stamps a "generated at" line fails this test by construction. |
| AC-20 | REQ-F-021, REQ-F-024 | TC-018o | Contract-surface enumeration (closed 4-caveat set) | The AC-12 aggregate, rendered via `report-baseline.sh` | For each headline metric: both the observed `min`/`median`/`max`/spread **and** the `accept_lo`/`accept_hi` (or `accept_set`) **and** a printed sentence naming the derivation rule (e.g. "derived as `[min−r, max+r]`" or the 0.10-median-floor sentence, matching whichever branch that metric's fixture hit). All **four** REQ-F-024 caveats present by content: the timeout-exclusion rule (prose, no tracked item), `TD-079` (F-4 anomaly bucket), `TD-081` (F-6, fmt/vet attribution), `T-004` (tests_pass not a regression signal). Negative: a report missing even one of the four caveat mentions fails — this is a closed-set check (grep for all four tokens), not "some caveats exist." |
| AC-21 | REQ-F-022 | TC-018p | Contract-surface enumeration | The AC-12 aggregate (with a full `provenance` block and a computed `baseline_id`) | Report's provenance section reproduces `model_ids`, `fixture_base_sha`, `variant_bundle_sha256`, `corpus_schema_version`, `shark_version`, `reps`, and `input_digest` **verbatim** from the aggregate; separately, the aggregate's own `baseline_id` field matches the documented format `<variant_id>-<fixture_base_sha[:12]>-r<reps>` exactly (12 hex chars of the SHA, literal `r` + integer reps) — the format-string gap this plan's REQ-indexed check found (no AC states it explicitly). |
| — | REQ-F-007 2nd sentence (family presence ≠ `sources` presence) | TC-018q | Contract-surface enumeration (positive/negative pair) | (a) A fixture with `oracle` block **present** and populated, but `sources.oracle` key **absent** entirely (a hypothetically-narrower collector); (b) a fixture with the whole `sources` object **absent** while every family block is present | (a): the `oracle` family still contributes to its metrics' statistics — presence is read from the block, not from `sources`. (b): the record is **not** classified `anomaly` and no metric is excluded — the absent `sources` block alone is not itself an anomaly signal. Negative: an implementation that gates family inclusion on `sources.<family>` being present would exclude fixture (a)'s `oracle` metrics and would misclassify fixture (b) — this TC is designed to catch exactly that implementation. |
| — | REQ-F-019 (`input_digest` is computed, not echoed) | TC-018r | Attack-class enumeration (mutation sensitivity) + invariant (order independence) | (a) The AC-12 fixture root, aggregated once; then one byte of one contributing `record.jsonl` is mutated (e.g. a digit in `loc.prod_added`) and aggregated again. (b) The same fixture root's files re-aggregated after the directory listing order is perturbed (e.g. touch the files in reverse mtime order — the sorted-lines rule must make this irrelevant) | (a): `input_digest` differs between the two runs. (b): `input_digest` is **identical** regardless of on-disk enumeration order (the sorted-lines rule, REQ-F-019's own text). Negative: an implementation using an unsorted concatenation would fail (b) — its digest would vary with directory listing order, which is exactly the property REQ-F-019 exists to prevent (reproducibility from the artifact set, not from the enumeration order). |
| — | Record enumeration: pinned glob vs. `find` | TC-018s | Attack-class enumeration (silent-corruption surface) | A fixture root with one normal complete pair under `<item>/<variant>/rep-1/record.jsonl`, plus a **structurally identical, well-formed** `record.jsonl` placed under `<root>/.incomplete/<item>/<variant>/rep-1-1/record.jsonl` (a quarantined prior attempt that happens to be well-formed) | The quarantined record contributes to **no** band, **no** `inventory`, and **no** count anywhere in `aggregate.json` — the dot-prefixed top-level component is excluded by the glob's own semantics (`*` does not match a leading-dot path component), not by a runtime skip-rule. Negative: an implementation using `find "$root" -name record.jsonl` (unscoped) would pick up the quarantined file and double-count the pair — this TC is designed to fail exactly that implementation. |
| — | `batch-log.jsonl` never read by the aggregator | TC-018t | Attack-class enumeration (fail-soft boundary) | A fixture root with a valid, complete set of records **and** a `batch-log.jsonl` file `chmod 000`'d (unreadable) | Aggregation succeeds identically to the same fixture root with `batch-log.jsonl` absent entirely — byte-identical `aggregate.json` in both cases. Negative: any error, warning, or behavior difference caused by the unreadable `batch-log.jsonl` fails this test — REQ-F-007's "reads only `record.jsonl` files... and nothing else" is the property under test. |
| — | `report-baseline.sh` on an aggregate with unsupported `schema_version` | TC-018u | Attack-class enumeration (malformed input, aggregate side) | An `aggregate.json` with `schema_version: "99.0"` | `report-baseline.sh` exits non-zero, per the Interface-contracts exit table's `report-baseline.sh` non-zero row ("Aggregate unreadable or unsupported `schema_version`") — distinct from AC-07's record-level check. Negative: no partial/truncated markdown is written to stdout before the failure is detected (the whole input is validated before any output begins, matching REQ-F-020's "pure function" framing). |
| — | REQ-N-007 immutability precondition (Q005) | TC-018v | Contract-surface enumeration (content-only) | Direct read of `bench/README.md`'s new "Baseline aggregation, noise band, and replay" section | The section states both REQ-N-007 preconditions verbatim in substance: (i) never delete `bench/corpus/ledgers/<sha>/` for any SHA a published manifest references; (ii) a corpus item's seed file and held-back F2P files are immutable for any SHA a published manifest references. Content-only — no decision-table or mutation test simulates the meaning of the prose itself (matches F02's TC-017 treatment of its own documentation AC). |

### `replay-manifest.sh` (AC-22..AC-27) — `tc019_replay_manifest_test.sh`

| AC | Requirement(s) | TC | Technique | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|
| AC-22 | REQ-F-027 | TC-019a | Decision table (3 preconditions × pass/fail) | Three sub-cases, one per REQ-F-027 precondition, each isolating that single precondition's failure while the other two hold: (i) `bench/corpus/ledgers/<manifest.fixture_base_sha>/` directory absent; (ii) the stored record's `manifest.item_id` does not resolve in the `--corpus` source file; (iii) the `--band` aggregate has no entry for `(item_id, variant_id)` | Each sub-case: exits non-zero, names the specific missing precondition (the missing path / the unresolved item id / the missing band entry) on stderr, and `RUN_ONE_BIN` stub's invocation count is **0** — no API spend before dispatch. Negative: a fourth sub-case where all three preconditions hold invokes the stub exactly once (proves the checks are gating, not always-failing). |
| AC-23 | REQ-F-026, ADR-F03-02 | TC-019b | Contract-surface enumeration (synthetic corpus shape) | A stored record (item existing in the real `bench/corpus/corpus.yaml`, e.g. `cart-remove-item-last-match`) whose manifest carries `fixture_base_sha`/`corpus_schema_version`/`p2p_set` values, plus a `RUN_ONE_BIN` stub configured to **echo its received `--corpus` file's contents** to a capture path rather than fabricate a record (so the synthetic file itself is inspected, not just its effect) | The synthetic corpus file: top-level `fixture.base_sha` **and** the single item's `fixture_base_sha` both equal the manifest's value (byte-identical to each other, per corpus.yaml's own INVARIANT); top-level `schema_version` equals `manifest.corpus_schema_version`; the item's `p2p_set` equals `manifest.p2p_set`; `seed_path` and every `f2p.paths` entry are **absolute** paths resolving to files that exist in the source corpus tree; `items:` holds exactly the one item; `negative_items:` is omitted or empty. |
| AC-24 | REQ-F-028, ADR-F03-02 | TC-019c | Attack-class enumeration (curator-edit simulation) | A **temp copy** of `corpus.yaml` (never the tracked file) with the target item's `fixture_base_sha` edited to a different value than the stored record's manifest, passed via `--corpus` | `verification.json` carries a `corpus_drift` entry naming both the live (temp-corpus) value and the manifest's pinned value; the synthetic replay corpus built for dispatch still carries the **manifest's** SHA (verified by the same echo-capture technique as TC-019b), not the live/edited one; `RUN_ONE_BIN` stub is still invoked (drift is recorded, not a precondition failure — REQ-F-027's preconditions don't include "corpus matches live", only that the item **id** resolves). |
| AC-25 | REQ-F-029, ADR-F03-05 | TC-019d | Decision table (one axis differs) | `RUN_ONE_BIN` stub returns a fresh record whose `manifest.variant_bundle_sha256` differs from the stored record's, `model_ids` **matching** | Verdict `invalid`; `reasons[]` contains exactly `variant_bundle_drift`; both the expected (stored) and actual (fresh) hash values are named. Negative: verdict is **not** `fail` — a drifted-input comparison is never reported as a measured result. |
| AC-26 | REQ-F-029, ADR-F03-05 | TC-019e | Decision table (other axis differs) | Same as AC-25 but `variant_bundle_sha256` **matching**, `model_ids` differing | Verdict `invalid`; `reasons[]` contains exactly `model_version_drift`; both ID lists named in full (not just "differs"). |
| AC-27 | REQ-F-025, REQ-F-030 | TC-019f | Boundary Value Analysis (inside/outside band) + State transition (run_key identity) | (a) Fresh record's every headline metric inside its published `accept_lo`/`accept_hi` (or `accept_set`) → expect `pass`. (b) The same fixture with exactly one metric's fresh value moved **outside** its interval → expect `fail`, naming that metric, its replayed value, and its interval. Both sub-cases assert: the invocation passed `RUN_ONE_BIN` the manifest's **stored** `--rep` value (captured via the stub's argv log); `--out` is a **distinct** root from the original artifact's (never the same path); the fresh record's `manifest.run_key` is byte-identical to the stored record's `run_key` | (a) verdict `pass`, per-metric table shows every metric's verdict `pass`. (b) verdict `fail`, the moved metric named with its replayed value and interval; every other metric still individually `pass` in the per-metric table (fail is metric-scoped in the table even though the overall verdict is `fail`). |
| — | Both `variant_bundle_sha256` **and** `model_ids` differ (AC-25/AC-26's uncovered 2×2 cell) | TC-019g | Decision table (both axes) | `RUN_ONE_BIN` stub returns a fresh record differing from the stored one on **both** fields simultaneously | Verdict `invalid`; `reasons[]` contains **both** `variant_bundle_drift` and `model_version_drift`, each with its own named expected/actual pair — per this plan's resolution above, not one reason silently subsuming the other. |
| — | RUN_ONE_BIN default resolution (same defect class as TC-017f) | TC-019h | Contract-surface enumeration (constructor default) | `replay-manifest.sh` invoked with `RUN_ONE_BIN` unset, same sibling-resolution setup as TC-017f | Resolves and invokes `$SCRIPT_DIR/run-one.sh` (or the test's same-directory stand-in), never a bare-PATH-name lookup. |

### `make fmt && make lint && make test`, `run-all.sh` registration (AC-28) — `tc020_zero_go_change_test.sh`

| AC | Requirement(s) | TC | Technique | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|
| AC-28 | REQ-N-003, REQ-N-001 | TC-020 | Contract-surface enumeration (mechanical, not "someone runs `make test`") | Base resolution: `base="$(git merge-base HEAD origin/main 2>/dev/null)"`; if that fails (no `origin/main`, detached checkout with no remote, shallow clone missing the merge-base commit), the diff-stat half of TC-020 **skips cleanly with a logged reason** rather than comparing against a wrong or empty base — a vacuous pass is worse than no check, so an unresolvable base is a skip, not a silent pass. When resolved: `git diff --stat "$base"..HEAD -- internal/ cmd/ tests/contracts/`. Independently (never skipped): `grep -c 'tc017_run_batch_test.sh\|tc018_aggregate_report_test.sh\|tc019_replay_manifest_test.sh' bench/scripts/tests/run-all.sh` | When the base resolves: `git diff --stat` reports **zero** changed files under those three paths (the mechanical proof of REQ-N-003's "no Go file is added or changed" — same technique F04's plan used for its own REQ-N-004 code-review check). `run-all.sh` contains all three new registration lines, checked unconditionally. Negative: a change that adds a new file under `tests/contracts/` (e.g. a second I-02 validator, which ADR-F03-09 explicitly forbids) fails the diff-stat half whenever it runs, even if `make test` itself would still pass; the registration-grep half alone still gives partial AC-28 coverage on any checkout where the merge-base can't be resolved (e.g. a shallow CI clone), so the check degrades rather than going dark entirely. |

ACs without a TC: none. Every AC in `spec.md`'s 28-row table has at least
one row above; nine additional rows close the REQ-indexed gaps this plan's
own drift analysis found.

---

## ISTQB Technique Application (per AC)

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-01 | Contract-surface enumeration | TC-017a | The batch command is F03's own external interface over the real corpus; every pair and the negative-items exclusion must be enumerated, not sampled. |
| AC-02 | State transition (resume) | TC-017b | "Already complete" is a state the batch must recognize from disk alone; the stub's own call-count is the literal mechanism AC-02's text names. |
| AC-03, AC-04 | Decision table (directory state × flag) | TC-017c, TC-017d | Two binary conditions (stale directory present, `--reclaim-incomplete` on/off) determine three distinct behaviors (skip-and-name, quarantine-and-rerun, and — untested here because it can't occur — "not stale, nothing to do"); a decision table forces both reachable rows explicitly. |
| AC-05 | Attack-class enumeration (fault tolerance) | TC-017e | "One pair's failure never aborts the batch" is a defensive property against a specific fault class (an unhandled non-zero exit propagating up), the class attack-class enumeration exists to force. |
| AC-06 | State transition (re-run stability) + Boundary (locale) | TC-018a | Determinism is a re-run-stability claim (same family as F02's TC-015j); the locale axis is the boundary that actually falsifies "sorted explicitly" vs. "sorted by inherited collation." |
| AC-07 | Attack-class enumeration | TC-018b | An unsupported schema version is a malformed-input attack surface; "no aggregate written" is the defensive property under test, not just the exit code. |
| AC-08 | Decision table (F-4 anomaly shape) | TC-018c | The anomaly rule is itself a 3-way decision (timeout / explained / unexplained-absence); AC-08 isolates the third row and a decision table is what distinguishes it from the other two, which AC-09/AC-10 separately test. |
| AC-09, AC-10 | Decision table (same 3-way rule, other two rows) | TC-018d, TC-018e | Same decision table as AC-08, remaining rows — grouped by technique, not by TC number, because all three ACs are one classification rule read at three points. |
| AC-11 | Boundary/equivalence (5-field table-drive) | TC-018f | REQ-F-011 names five fields; equivalence partitioning across five is required to avoid an implementation that hardcodes one and silently ignores the rest — the gap this plan's own drift analysis found. |
| AC-12 | Boundary Value Analysis (3 interval branches) | TC-018g | ADR-F03-04's formula has three reachable branches (`r>0`, `r==0∧median≠0`, identically-zero); BVA is exactly the technique for a piecewise formula, and a single happy-path test would miss two of the three branches. |
| AC-13 | Boundary Value Analysis (n=1) | TC-018h | `n=1` is the literal boundary REQ-F-016 names ("fewer than two contributing reps"). |
| AC-14 | Equivalence Partitioning (all-true / all-false) | TC-018i | `f2p_resolved`'s discriminative check is a closed boolean-uniformity partition; both uniform classes need representation, and a mixed class is the negative. |
| AC-15 | Boundary Value Analysis (`>` vs `>=`) | TC-018j | REQ-F-018's comparison operator is a strict inequality; BVA on the exact-equal case is what proves it's not `>=`. |
| AC-16 | Decision table (repeat status × distinct status) | TC-018k | Summing-vs-not is a 2-condition decision (same status repeats: sum; different status: separate step) that a single-occurrence-only test can't distinguish. |
| AC-17 | Contract-surface enumeration (source-field lock) | TC-018l | "Never reads `tests_pass` as a regression signal" is a closed-surface claim about which one field feeds a named output; enumerating the alternate (wrong) source and proving it's uncorrelated with the output is the only way to falsify a hidden fallback. |
| AC-18 | Decision table (present-omitted-key vs. absent-block) | TC-018m | Two structurally different "missing" states (a present object missing one key vs. the object itself missing) resolve to two different rules (0 vs. excluded); a decision table is required to keep them from collapsing into one code path. |
| AC-19 | State transition (re-run stability) | TC-018n | Same re-run-stability claim as AC-06, applied to the report generator. |
| AC-20 | Contract-surface enumeration (closed 4-caveat set) | TC-018o | REQ-F-024 names exactly four caveats; enumerating all four by content (not "a caveats section exists") is the only way to catch a report that ships three. |
| AC-21 | Contract-surface enumeration | TC-018p | The provenance block is a fixed, named field set copied verbatim from the aggregate — enumeration, not sampling. |
| REQ-F-007 (2nd sentence) | Contract-surface enumeration (positive/negative pair) | TC-018q | "Presence from the block, not from `sources`" needs both directions tested — a block present with no `sources` entry, and `sources` absent with every block present — to distinguish it from the two plausible wrong implementations. |
| REQ-F-019 | Attack-class enumeration (mutation) + invariant (order) | TC-018r | A digest's correctness is provable only by mutation-sensitivity (changes when input changes) and order-invariance (doesn't change when enumeration order changes) — neither is provable by inspecting one static output. |
| Record enumeration (glob) | Attack-class enumeration | TC-018s | The glob-vs-`find` distinction is a silent-corruption attack surface (double-counting an abandoned attempt); the discriminating fixture (a well-formed record under `.incomplete/`) is the only way to falsify a `find`-based implementation. |
| `batch-log.jsonl` non-read | Attack-class enumeration (fail-soft boundary) | TC-018t | "Never read" is a defensive property against a specific fault (an unreadable diagnostics file breaking the aggregator); the `chmod 000` injection is the class of attack that would expose a violation. |
| `report-baseline.sh` schema check | Attack-class enumeration | TC-018u | Same class as AC-07, one script over — a malformed aggregate is the input class to enumerate, not the record's own malformedness (already covered). |
| REQ-N-007 (Q005) | Contract-surface enumeration (content-only) | TC-018v | A documentation precondition has no runtime behavior to test — content presence is the only checkable property, matching F02's own treatment of its analogous documentation ACs. |
| AC-22 | Decision table (3 preconditions) | TC-019a | Three independent precondition checks combine; a decision table forces each isolated failure plus the all-pass row, rather than testing only the "everything fails together" case. |
| AC-23 | Contract-surface enumeration | TC-019b | The synthetic corpus is itself an interface contract (what `run-one.sh` will read); every field REQ-F-026/ADR-F03-02 name must be enumerated, not sampled. |
| AC-24 | Attack-class enumeration (curator-edit simulation) | TC-019c | Corpus drift is exactly the attack class Finding 6 (research report) names: an ordinary, non-malicious curator edit silently defeating reproducibility if unguarded. |
| AC-25, AC-26 | Decision table (2-axis: bundle × model_ids) | TC-019d, TC-019e, TC-019g | Two independently-driftable fields combine into a 2×2 table (match/match, bundle-only, model-only, both) — AC-25/AC-26 name two of the four cells explicitly; this plan's own drift analysis found and closed the third (both-differ) cell via TC-019g. |
| AC-27 | Boundary Value Analysis (inside/outside interval) + State transition (run_key identity) | TC-019f | Pass/fail is a boundary condition on the published interval; run_key identity is a state-transition property (the fresh run must join under the *same* key as the stored one, not a renumbered one). |
| RUN_ONE_BIN default | Contract-surface enumeration (constructor default) | TC-017f, TC-019h | A default-resolution path is a contract surface distinct from every overridden-stub test in this plan — TD-077's exact defect class. |
| AC-28 | Contract-surface enumeration (mechanical) | TC-020 | "No Go file added or changed" and "three tests registered" are both enumerable, mechanically checkable facts, not review judgments. |

ACs without a technique annotation: none. Every AC and every REQ-indexed
addition above has at least one named technique.

---

## ISO 25010 Coverage Matrix

`N/A` justifications follow `uat-plan.md`'s own framing for this epic
("Not a product concern here — the harness is offline tooling") and F02's
precedent: Performance and Portability are out of scope for offline
curator/CI tooling with no cross-OS claim in `spec.md`.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-01 | ✅ TC-017a | N/A | N/A | ✅ TC-017a (machine-readable summary, unattended) | N/A | N/A | N/A | N/A |
| AC-02 | ✅ TC-017b | N/A | N/A | N/A | ✅ TC-017b (zero re-run cost) | N/A | N/A | N/A |
| AC-03 | ✅ TC-017c | N/A | N/A | ✅ TC-017c (named classification) | ✅ TC-017c (no silent data loss) | N/A | N/A | N/A |
| AC-04 | ✅ TC-017d | N/A | N/A | N/A | ✅ TC-017d (byte-identical quarantine, move not delete) | ✅ TC-017d (REQ-F-006 non-destructive) | N/A | N/A |
| AC-05 | ✅ TC-017e | N/A | N/A | ✅ TC-017e (named failure) | ✅ TC-017e (fault-tolerant loop) | N/A | N/A | N/A |
| AC-06 | ✅ TC-018a | N/A | N/A | N/A | ✅ TC-018a (re-run stability) | N/A | ✅ TC-018a (explicit sort, not collation-dependent) | N/A |
| AC-07 | ✅ TC-018b | N/A | N/A | ✅ TC-018b (named file) | ✅ TC-018b (no partial output) | N/A | N/A | N/A |
| AC-08 | ✅ TC-018c | N/A | N/A | ✅ TC-018c (named run_key + families) | ✅ TC-018c (loud, not silently averaged) | N/A | N/A | N/A |
| AC-09 | ✅ TC-018d | N/A | N/A | ✅ TC-018d (named exclusion reason) | ✅ TC-018d (cap not conflated with measurement) | N/A | N/A | N/A |
| AC-10 | ✅ TC-018e | N/A | N/A | N/A | ✅ TC-018e (named reason) | N/A | N/A | N/A |
| AC-11 | ✅ TC-018f | N/A | N/A | ✅ TC-018f (both values named) | ✅ TC-018f (never published as a result) | N/A | N/A | N/A |
| AC-12 | ✅ TC-018g | N/A | N/A | N/A | ✅ TC-018g (interval invariant holds) | N/A | ✅ TC-018g (formula branch coverage) | N/A |
| AC-13 | ✅ TC-018h | N/A | N/A | ✅ TC-018h (flag visible) | N/A | N/A | N/A | N/A |
| AC-14 | ✅ TC-018i | N/A | N/A | ✅ TC-018i (named E40-F01) | N/A | N/A | N/A | N/A |
| AC-15 | ✅ TC-018j | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-16 | ✅ TC-018k | N/A | N/A | N/A | ✅ TC-018k (correct rework accounting) | N/A | N/A | N/A |
| AC-17 | ✅ TC-018l | N/A | N/A | N/A | N/A | ✅ TC-018l (no fabricated regression signal) | N/A | N/A |
| AC-18 | ✅ TC-018m | N/A | N/A | N/A | ✅ TC-018m (correct n, no silent miscount) | N/A | N/A | N/A |
| AC-19 | ✅ TC-018n | N/A | N/A | N/A | ✅ TC-018n (re-run stability) | N/A | N/A | N/A |
| AC-20 | ✅ TC-018o | N/A | N/A | ✅ TC-018o (derivation rule printed) | N/A | N/A | N/A | N/A |
| AC-21 | ✅ TC-018p | N/A | N/A | N/A | N/A | N/A | ✅ TC-018p (baseline_id format checkable) | N/A |
| AC-22 | ✅ TC-019a | N/A | N/A | ✅ TC-019a (named missing precondition) | ✅ TC-019a (no dispatch, zero spend) | N/A | N/A | N/A |
| AC-23 | ✅ TC-019b | N/A | ✅ TC-019b (schema-shaped synthetic corpus) | N/A | N/A | N/A | N/A | N/A |
| AC-24 | ✅ TC-019c | N/A | N/A | ✅ TC-019c (both values named) | N/A | N/A | N/A | N/A |
| AC-25 | ✅ TC-019d | N/A | N/A | ✅ TC-019d (both hashes named) | N/A | ✅ TC-019d (identity gate, not silently trusted) | N/A | N/A |
| AC-26 | ✅ TC-019e | N/A | N/A | ✅ TC-019e (both ID lists named) | N/A | ✅ TC-019e (model version pinned) | N/A | N/A |
| AC-27 | ✅ TC-019f | N/A | N/A | ✅ TC-019f (per-metric table) | ✅ TC-019f (run_key identity preserved) | N/A | N/A | N/A |
| AC-28 | ✅ TC-020 | N/A | ✅ TC-020 (zero Go diff) | N/A | N/A | N/A | ✅ TC-020 (mechanical, CI-checkable) | N/A |

No empty cells. `N/A` cells follow the epic-wide justification stated above.
Every non-`N/A` cell cites a TC present in the AC Test Matrix.

---

## Observability Design (per behavior)

This is offline curator/CI tooling with no production request path (same
posture as F01's and F02's own test plans). "Observability" means the
scripts' own machine-readable output, which a curator or a future Phase 2
comparison report depends on without re-deriving it.

| Behavior | Log/stdout evidence | Trace/metric | Test assertion |
|---|---|---|---|
| Batch progress and per-pair classification | `run-batch.sh`'s JSON summary on stdout names every pair's outcome (`pending_run`/`skipped_complete`/`incomplete_prior_attempt`/`quarantined_and_rerun`/`failed`) | N/A — internal, no request path | TC-017a/b/c/d/e each assert the specific classification token, not just "the batch ran" |
| Aggregation anomaly loudness | `aggregate.json`'s `anomalies[]` names `run_key` + missing families; aggregator's own non-zero exit is the loud signal REQ-F-009/UAT's I-02 scenario names | N/A | TC-018c |
| Provenance-uniformity violation | `provenance.divergences[]` names the differing field and both values, on every one of the five REQ-F-011 fields | N/A | TC-018f |
| Corpus feedback (non-discriminative tasks) | Report's named heading, `E40-F01` cited by name | N/A | TC-018i |
| Measurement-noise caveats | Report's four REQ-F-024 caveat sentences, each naming its tracked item (`TD-079`/`TD-081`/`T-004`) or rule | N/A | TC-018o |
| Replay precondition failure | `replay-manifest.sh` names the specific failed precondition on stderr before any dispatch | N/A | TC-019a |
| Replay drift | `verification.json`'s `reasons[]` names `corpus_drift`/`variant_bundle_drift`/`model_version_drift` with expected/actual values, never a bare "invalid" | N/A | TC-019c, TC-019d, TC-019e, TC-019g |
| Replay per-metric verdict | `verification.json`'s `metrics[]` array: per-metric `replayed_value`, interval, and verdict — never a single pass/fail bit alone | N/A | TC-019f |
| Input reproducibility | `input_digest`, computed (not echoed) and order-invariant | N/A | TC-018r |

No new metrics/traces are required — this is deliberately not a
runtime-service observability surface, matching `uat-plan.md`'s own framing
("Not a product concern here").

---

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-02 | E40-F02 Bench harness: run driver and metric collection | E40-F03 (this feature) | `architecture.md#metric-collection-and-artifact-schema` | `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` (Go function: `TestTC001_I02ArtifactContract`) | `TestTC001_I02ArtifactContract` (reused, not reimplemented) |

**I-02 (consumes).** Shape source and contract-test pointer are copied
**verbatim** from `spec.md`'s "Cross-feature interactions" section, which
itself copies them verbatim from E40-F02's `spec.md` "Produces: I-02" per
decision note 2213. `TestTC001_I02ArtifactContract` is reused **as-is — no
twin test, no second I-02 validator** (ADR-F03-09, REQ-N-003). This test
plan does not modify, extend, or duplicate that function or its file.

**F03's own real caller path for I-02** is `aggregate-runs.sh`, exercised
by every `TC-018*` case in this plan — the consumer-side task
`task_generation` must create (per `spec.md`'s own note: "`task_generation`
must still create at least one task declaring `I-02: consumes` that copies
both pointers verbatim and owns the real caller path... `aggregate-runs.sh`")
discharges its I-02 obligation by pointing at this plan's `tc018` suite,
not by writing new Go.

**Gate mode:** `live`, per `E40-interaction-map.md`'s I-02 row — not
staged, not `contract-only`. No staged-edge fields (activation owner,
closure key) apply; not applicable is recorded here, not silently omitted,
matching F04's own treatment of its I-03 row.

**The consumed shape is the shipped, TD-076-adjudicated one.** Every
`TC-018*` fixture in this plan is derived from the **current** committed
goldens (`sources` narrowed to the five-value set, `timeout_detail` present
on the timeout path) — never from the pre-adjudication four-value/every-family
reading `spec.md`'s TD-076 section describes as rejected.

---

## Cross-epic integrations

**None assigned — verified against `spec.md`'s own table, not re-derived.**
`spec.md`'s "Cross-epic integrations" section checked all three
cross-epic rows (X-07, X-08, X-09) against `E40-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md` and found none assigned to
E40-F03. This test plan mirrors that table rather than re-verifying the
two source maps a third time (they were already cross-checked by F02's and
F04's own test plans, and `spec.md`'s specification step re-checked them a
third time for F03):

| Row | Owning/consuming feature | F03 involvement | Test coverage |
|---|---|---|---|
| X-07 (E22 → E40) | E40-F02 | None — F03 never parses `RunResult`/`StageLog`/transcript bytes; the canary is `run-one.sh`'s concern | N/A — no F03 test needed |
| X-08 (E40-F04 → E22) | E40-F04 | None | N/A |
| X-09 (E27 → E40 Phase 2 G1) | TBD, Phase 2 | None | N/A |

No F03 test case exists for any X-## row, matching the "no seam" finding
above — an absent row is not an untested row, since there is nothing here
for F03 to own.

---

## Integration Scenarios

| Scenario | Components | Boundary verified | Epic UAT reference |
|---|---|---|---|
| Batch → aggregate → report chain, unattended | `run-batch.sh` → 30 `record.jsonl` files → `aggregate-runs.sh` → `aggregate.json` → `report-baseline.sh` → `report.md` | Each stage consumes only its documented predecessor's output, never reaching around it (ADR-002, "reproducible from the artifact directory alone") | UAT-01 |
| Resume after partial completion | `run-batch.sh` re-invoked over a mixed root (some pairs complete, one `incomplete_prior_attempt`, rest pending) | Skip-vs-rerun-vs-quarantine classification is stable across re-invocations | UAT-01 ("Re-running the batch skips already-completed pairs instead of duplicating them") |
| A record missing a metric family, aggregated | I-02 record (F02's shape) → `aggregate-runs.sh`'s classification rule → loud non-zero exit | The exact scenario `uat-plan.md` names for I-02: "a record missing a metric family fails aggregation loudly instead of being silently averaged away" | I-02 cross-feature scenario, UAT-01 |
| Replay reproduces a stored manifest within its published band | `replay-manifest.sh` → synthetic corpus → `run-one.sh` (stubbed in self-tests) → fresh record → comparison against `aggregate.json`'s band | "The report regenerates from the artifact directory alone, with no state outside it" — replay's inputs are exactly the stored record + published aggregate, nothing else | UAT-07 |
| Curator edits a corpus item between original run and replay | Live `corpus.yaml` (temp copy in tests) vs. stored manifest's pinned values | Replay always uses the manifest's values; drift is recorded, not silently absorbed or silently blocking | UAT-07 (non-functional evidence: reproducibility under drift), Finding 6 (research report) |
| Live repo/DB never touched by any F03 script | `run-batch.sh`/`replay-manifest.sh` invoke `run-one.sh` (real or stubbed) only; `aggregate-runs.sh`/`report-baseline.sh` touch no subprocess at all | REQ-N-002: no script invokes a shark project-initialisation command directly | Non-functional evidence, "the live repo... untouched" |

This feature contributes to **UAT-01** and **UAT-07** per `uat-plan.md`'s
own criterion column (G2/G4/G5 and G7 respectively). No other epic UAT
scenario names F03.

---

## Caller-Path Contracts (Step 5.8)

Every runtime test case in this plan drives a real bash subprocess against
real or synthetic files — no test in this plan mocks a script's own
internal control flow; only the external `run-one.sh` binary is ever
substituted (via `RUN_ONE_BIN`), matching the dispatch prompt's Zero-API-spend
rule.

| TC | Production entrypoint (exact invocation) | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-017 (a–f) | `bench/scripts/run-batch.sh --out <dir> --corpus <corpus.yaml> --reps <n> [--items <id,...>] [--reclaim-incomplete] [--timeout <s>]` | `RUN_ONE_BIN` pointed at a stub script (`bench/scripts/testdata/stubs/run-one`) for the duration of the test only | Do not stub or bypass `run-batch.sh`'s own enumeration, classification, or quarantine logic — only the external `run-one.sh` invocation is replaced | An implementation that hardcodes "every pair succeeds" internally instead of actually invoking `RUN_ONE_BIN` per pair would pass a test that stubs too high (e.g. mocking `run-batch.sh` itself); stubbing only the driver binary keeps the enumeration/classification logic on the hook |
| TC-017f, TC-019h specifically | Same entrypoints, `RUN_ONE_BIN` **unset** | Real filesystem resolution: a real `run-one.sh`-named file placed as the caller script's sibling | Do not set `PATH` to include the sibling directory (that would test PATH-resolution, not sibling-path-default resolution — the two are different failure modes, and TD-077's actual defect was a bare-name default relying on `PATH`) | A `RUN_ONE_BIN="${RUN_ONE_BIN:-run-one.sh}"` default (bare name, PATH-dependent) passes in any dev shell where `bench/scripts/` happens to be on `PATH` and fails in CI or any other shell — exactly TD-077's defect, reproduced as a passing-then-failing test depending on environment, which this TC is designed to catch deterministically instead |
| TC-018 (a–v) | `bench/scripts/aggregate-runs.sh --root <dir> [--variant <id>]` and `bench/scripts/report-baseline.sh --aggregate <path>` | Real filesystem reads of committed-goldens-derived fixture files under a per-test-case root; no subprocess is invoked by either script at all (REQ-F-007/REQ-F-020: pure functions of their input) | Do not hand-type expected `aggregate.json`/`report.md` fragments from what the scripts are expected to produce and diff against a hand-typed oracle for the *statistics themselves* — assert against the documented formula (ADR-F03-04) applied to the fixture's own known input values, computed independently in the test (e.g., in the test script's own `python3` snippet), not copied from the implementation | An aggregator that reads its own previous output as ground truth (self-referential) would pass a test whose oracle was hand-copied from a prior run of the same implementation; computing the expected statistic independently from the fixture's raw values in the test itself closes that hole |
| TC-018a specifically | Same entrypoint, invoked under two `LC_ALL` values | Real process-level locale environment variable, not a mocked collation function | n/a | An aggregator relying on `sort` (or a language runtime's default map/dict iteration) without an explicit `LC_ALL=C`/`sort -k` invocation would diverge between locales — this is exactly the property `LC_ALL=en_US.UTF-8` vs. `LC_ALL=C` is chosen to falsify |
| TC-018g specifically | Same entrypoint | n/a | Do not hardcode `accept_lo`/`accept_hi` expected values as literals copied from a prior run — compute them in the test from the fixture's own `min`/`max`/`median` via ADR-F03-04's formula, independently | An implementation with an off-by-one in the interval clamp (e.g. always applying the non-negative-clamp even when unneeded) would still pass a hand-copied-literal oracle if the literal was copied from that same buggy implementation's own prior output |
| TC-018r specifically | Same `aggregate-runs.sh` entrypoint, invoked with a mutated byte in one contributing record and with directory-entry ordering perturbed | Real filesystem mutation (`sed -i` one byte) and real `touch -d` mtime reordering | Do not simulate "order changed" by passing a differently-sorted file list on the command line — `aggregate-runs.sh` takes a `--root` directory, not a file list, so ordering must be perturbed at the filesystem level (mtime/inode order), the same way a real re-run after a partial batch resume would produce a different on-disk enumeration order | A digest implementation that iterates `os.walk` (or bash glob expansion) without an explicit sort would produce a different digest under reordered mtimes — exactly what this TC is designed to catch |
| TC-018s specifically | Same entrypoint, a fixture root with a well-formed `record.jsonl` under `.incomplete/<item>/<variant>/rep-1-1/` | Real filesystem placement; no glob/find call is mocked — the production script's own enumeration logic runs unmodified against a real directory tree | Do not pre-filter the fixture root before invoking the script (that would test the fixture, not the script) | A `find`-based enumeration (unscoped by the leading-dot exclusion the bash glob provides implicitly) picks up the quarantined file and silently double-counts — the exact defect this TC exists to catch, not reachable by any test using only well-placed, non-adversarial fixtures |
| TC-019 (a–h) | `bench/scripts/replay-manifest.sh --record <path> --band <aggregate.json> --out <dir> --corpus <corpus.yaml> [--skip-canary]` | `RUN_ONE_BIN` pointed at a stub, same technique as TC-017; real `bench/corpus/corpus.yaml` (or a temp copy with one field edited, for TC-019c) as `--corpus` | Do not stub `replay-manifest.sh`'s own synthetic-corpus construction, precondition checks, or comparison logic — only the external `run-one.sh` invocation is replaced, matching ADR-F03-02's "invoke, never re-derive" | An implementation that hardcodes the synthetic corpus's shape as a static template instead of actually reading the stored manifest's fields would pass a test that only checks the *fact* a corpus file exists; TC-019b's echo-capture technique inspects the synthetic file's actual field values, closing that hole |
| TC-019b, TC-019c specifically | Same entrypoint, `RUN_ONE_BIN` stub configured to copy its received `--corpus <path>` file to a capture location instead of fabricating a record | Real file copy inside the stub (a real, executable script, matching TC-011's/TC-014's PATH-stub-style seam, here applied to a positional-arg-resolved binary rather than a `PATH`-resolved one) | Do not parse the synthetic corpus file's content from inside `replay-manifest.sh`'s own test assertions by re-implementing YAML parsing separately from what the real script would read — use the same `python3`+PyYAML read path the production script itself uses (REQ-N-001's own stated convention), so a schema-shape bug in the test's own parser can't mask a real one | A test that hand-parses the synthetic corpus with a regex instead of PyYAML could pass even if the file were YAML-invalid in a way a real consumer (`run-one.sh`, which also uses PyYAML) would reject |
| TC-020 | `git merge-base HEAD origin/main` (or logged skip) → `git diff --stat`, `grep` over `bench/scripts/tests/run-all.sh` | n/a — static analysis, no subprocess under test | n/a | A feature branch that adds a new `tests/contracts/*.go` file (violating ADR-F03-09) or forgets to register one of the three new self-tests both fail this mechanical check whenever the diff-stat half can run, independent of whether `make test` happens to still pass; an unresolvable merge-base degrades the check to registration-only rather than reporting a false pass on the diff-stat half |

**Implementation hook:** every developer's red-phase test must drive the
listed entrypoint with the listed argument shape; a test that mocks above
the entrypoint (e.g., a shell function stubbing out `aggregate-runs.sh`'s
own statistics computation instead of running the real script against a
real fixture directory) is rejected at code review, matching REQ-N-001's
"self-test... registered in `run-all.sh`" expectation of real bash
self-tests, not simulations of bash behavior.

---

## Test Infrastructure

### Existing patterns to reuse (with file paths)

| Pattern | File | Reused for |
|---|---|---|
| PATH-stub external binary, real executable, forwards to real tool except the one faked axis | `bench/scripts/tests/tc011_toolchain_guard_test.sh` | Conceptual precedent for the `RUN_ONE_BIN`-stub technique below (variable-override seam rather than `PATH`-resolution, since `run-one.sh` is invoked via `$RUN_ONE_BIN`/sibling-path, never bare-name-on-`PATH`) |
| Env-var-configurable stub binary with an invocation log | `bench/scripts/testdata/stubs/shark` (`STUB_SHARK_LOG`, canned outcomes, canned exit codes) | Direct structural precedent for the new `bench/scripts/testdata/stubs/run-one` stub — same config-via-env-var, same JSONL invocation log shape |
| Real subprocess invocation of the driver under test, only the outermost binary replaced | `bench/scripts/tests/tc014_run_one_smoke_test.sh` | TC-017's and TC-019's Caller-Path Contract style |
| `tests=(...)` array + PASS/FAIL wrapper | `bench/scripts/tests/run-all.sh` | Registration point for `tc017`/`tc018`/`tc019` (three new lines after `tc016`, per `spec.md`'s Component-changes table); `tc020` (this plan's own addition for AC-28) registered as a fourth line, since it needs to run in CI the same as the other three |
| Committed synthetic ledger convention, small hand-authored files shaped like real tool output | `bench/scripts/testdata/{lint,test}/*.json` | Conceptual precedent for `bench/scripts/testdata/aggregate/`'s fixture roots — except F03's fixtures are *generated*, not hand-authored, per REQ-N-006 (see below) |
| `bench/README.md`'s "I-02 record schema field reference" | `bench/README.md:255-296` | The exact field set and closed-value sets (`outcome`, `errors[].kind`, `sources.<family>`) every TC-018/TC-019 fixture must match — read directly, not assumed, for every derived fixture's structural shape |

### Fixture derivation: a committed generator, not a hand-copy (REQ-N-006, ADR-F03-08)

**Finding, closed here.** The two committed goldens' `manifest.item_id`
values (`cart-apply-discount-code`, `checkout-flow-timeout-demo`) do **not**
appear in `bench/corpus/corpus.yaml`'s `items:` list (verified directly:
neither string matches any of the 10 admitted or 3 negative item ids). "Derive
from the goldens" therefore cannot mean a literal copy for most of this
plan's scenarios — AC-09 alone requires three reps of *one* item where one
rep is a timeout, which means at minimum `manifest.item_id`/`.rep`/
`.run_key` must be rewritten from whichever golden is being adapted. Left
unstated, ADR-F03-08's claim ("a future producer change that breaks the
goldens breaks F03's tests too") is not actually true of a set of
one-time, hand-copied-then-edited fixture files — a hand-edit only reflects
the goldens' shape as of the moment it was made.

**Resolution, pinned as a hard requirement on task generation:**

1. **A committed generator script**, `bench/scripts/testdata/aggregate/gen_fixtures.py`
   (`python3`, matching REQ-N-001's embedded-python3 convention), reads
   **both** committed goldens — `tests/contracts/testdata/e40_i02_golden_record.jsonl`
   and `..._timeout.jsonl` — **fresh at test-run time** (not a pre-committed
   static output checked in once), for every `tc018`/`tc019` invocation.
2. **Fields a derived fixture MAY rewrite:** `manifest.item_id`,
   `manifest.item_type` (only if the scenario needs the other type),
   `manifest.rep`, `manifest.run_key` (recomputed from the other two per
   `<item_id>::<variant_id>::rep<rep>`), and the specific metric-under-test
   field(s) named by that scenario's row in the AC Test Matrix above (e.g.
   `loc.prod_added`'s three values for TC-018g(i), `quality.toolchain_guard`
   for TC-018e). No other field is touched.
3. **Fields a derived fixture MUST preserve structurally, unedited:** every
   key name, every value's type, the `sources` block's shape and closed
   value set, the family-presence/absence pattern implied by `outcome`
   (i.e., a scenario that wants a `completed`-outcome fixture starts from
   the completed golden and never fabricates a `timeout_detail` block; a
   timeout-outcome fixture starts from the timeout golden and never
   fabricates a `runresult` block), and `schema_version` (except in TC-018b/
   TC-018u, whose whole point is mutating it).
4. **Consequence, restoring ADR-F03-08's guarantee:** because the generator
   parses the goldens' actual current field names and structure at run
   time (not a snapshot), a future producer change that renames or removes
   a field the generator reads (e.g. `stages[].usage.input_tokens`) breaks
   the generator itself — loudly, at test-collection time — rather than
   leaving a stale fixture silently green against a shape the collector no
   longer produces.
5. **Fixtures needing real corpus membership vs. not:** `aggregate-runs.sh`
   never reads `corpus.yaml` (REQ-F-007: "reads only `record.jsonl`
   files... and nothing else"), so `tc018`'s generated `item_id` values
   need **not** exist in the real corpus — a synthetic id like
   `f03-fixture-<scenario>` is fine and keeps each scenario's fixture
   self-describing. `replay-manifest.sh`'s `tc019` fixtures are different:
   REQ-F-027's second precondition requires the item id to **resolve in
   the source corpus**, so `tc019`'s stored-record fixtures reuse an
   `item_id` that genuinely exists in `bench/corpus/corpus.yaml` (e.g.
   `cart-remove-item-last-match`, already used by F02's own TC-014a) — the
   generator's `--item-id` argument is set to a real corpus id for every
   `tc019` case, and to a synthetic id for every `tc018` case.

This is recorded as binding on task generation exactly as F02's plan
pinned its own `timeout_detail` schema resolution and F04's plan pinned its
tick-method requirement — not deferred as an open question.

### New test infrastructure needed (this feature's own deliverables)

- **`bench/scripts/testdata/stubs/run-one`** — new PATH-stub-style
  executable (though invoked via `$RUN_ONE_BIN`, not `PATH`, since
  `run-batch.sh`/`replay-manifest.sh` resolve it by variable, matching
  `run-one.sh`'s own `$SHARK_BIN` pattern). Accepts `run-one.sh`'s real CLI
  surface (`--item`, `--variant`, `--rep`, `--timeout`, `--out`,
  `--corpus`, `--skip-canary`), env-var-configurable: which
  golden-derived fixture record to write at the computed output path
  (`STUB_RUN_ONE_RECORD_FILE`), exit status (`STUB_RUN_ONE_EXIT`), an
  invocation log (`STUB_RUN_ONE_LOG`, same JSONL `{"argv":[...], "cwd":...}`
  shape as the existing `shark` stub — this is the literal mechanism
  AC-02's and AC-22's text both name: "asserted by a stubbed driver that
  records its call count"), and an echo-capture mode
  (`STUB_RUN_ONE_ECHO_CORPUS_TO`) that copies the received `--corpus` file
  verbatim to a capture path for TC-019b/TC-019c's synthetic-corpus
  inspection.
- **`bench/scripts/testdata/aggregate/gen_fixtures.py`** — the fixture
  generator described above, plus its own committed unit coverage is not
  required (it is test infrastructure, not production code) but its
  correctness is transitively proven by every `tc018`/`tc019` case that
  consumes its output — a generator bug producing a structurally wrong
  fixture would make the corresponding `tc018`/`tc019` case itself fail or
  assert something false, which is caught the same way a bad test fixture
  is always caught: by the test it drives misbehaving.
- **`bench/scripts/tests/tc017_run_batch_test.sh`**,
  **`tc018_aggregate_report_test.sh`**, **`tc019_replay_manifest_test.sh`**
  — the three self-tests `spec.md`'s Component-changes table reserves by
  name.
- **`bench/scripts/tests/tc020_zero_go_change_test.sh`** — this plan's own
  addition (not reserved by `spec.md`, since AC-28 needs a runnable home
  and folding a `git diff --stat` check into an unrelated script would be
  worse than a small fourth script). Registered in `run-all.sh` alongside
  the other three.
- **`bench/README.md` "Baseline aggregation, noise band, and replay"
  section** — new, per `spec.md`'s component-changes table: the aggregate
  schema, the band/acceptance-interval rules, the classification rules,
  the replay procedure, and REQ-N-007's two preconditions — drives
  TC-018v.

### Test tiers

All three (four, counting `tc020`) new self-tests are **Tier 1b**: no
submodule, no scratch project, no real `shark run` dispatch — every
external-binary seam (`RUN_ONE_BIN`) is stubbed. This is a **lighter**
tier than F02's own `tc014`/`tc015` (which still provision a real scratch
project via `scripts/shark-scratch-env.sh` for the driver smoke test),
because F03's scripts never invoke `shark` directly at all (REQ-N-002) —
the only subprocess any F03 script ever spawns is `run-one.sh` itself, and
every self-test in this plan replaces that one seam.

| Tier | Runs | Needs submodule/scratch project? | Where |
|---|---|---|---|
| Tier 1b | Curator, manually or via `bench/scripts/tests/run-all.sh` | No — synthetic golden-derived fixtures + `RUN_ONE_BIN`-stubbed driver only | `tc017_run_batch_test.sh`, `tc018_aggregate_report_test.sh`, `tc019_replay_manifest_test.sh`, `tc020_zero_go_change_test.sh` |

No Tier 2 script is added by this feature — F03 never dispatches a real
agent or provisions a real scratch project in its own self-tests (that
would violate the Zero-API-spend rule the dispatch prompt states
explicitly, and the corpus/canary/driver surfaces F03 reuses are already
proven at Tier 2 by F01's and F02's own suites).

---

## Red-team review (Step 7.5)

**Codex and gemini were both quota-blocked per the dispatch prompt's own
explicit instruction** ("codex AND gemini are both quota-blocked (codex
until 2026-08-07 23:30, gemini ~2026-08-13) — do NOT attempt either"), so
neither CLI was invoked — unlike F04's plan (which attempted codex twice
and hit the same quota wall live) or F02's plan (which substituted
`gemini-3.1-pro-high`), this plan's dispatch prompt pre-empted both
attempts as known-futile and asked for a documented substitution instead.

### Pass 1 — pre-draft design consultation (design input, not a review of this document)

Before this plan's tables were written, the `advisor` tool (an independent
stronger-reviewer mechanism available in this harness) was consulted on the
test-design **approach**, evaluated against the same seven Step 7.5
criteria. **This was not a review of the finalized 28-AC document — the
document did not exist yet.** Its nine findings were incorporated *during*
authoring, as design inputs, not caught after the fact:

1. Enumeration completeness — REQ-F-011's five uniformity fields (drafted
   as `model_ids`-only) → TC-018f, table-driven over all five.
2. Enumeration completeness — REQ-F-007's second sentence (family presence
   independent of `sources`) had no planned test → TC-018q.
3. Enumeration completeness — the both-drift-differ cell of AC-25/AC-26's
   2×2 → TC-019g.
4. Negative-case coverage — REQ-F-019's `input_digest` needed a
   mutation-sensitivity + order-invariance pair, not a value-echo check →
   TC-018r.
5. Observability — `batch-log.jsonl`'s "never read" claim needed a
   falsifying test → TC-018t.
6. Caller-path/fixture-provenance — neither committed golden's `item_id`
   exists in `corpus.yaml`, so "derive from the goldens" needed an explicit
   rewrite/preserve field list and a committed generator, not a hand-copy →
   the "Fixture derivation" subsection.
7. The RUN_ONE_BIN default-resolution gap (TD-077's defect class) → TC-017f,
   TC-019h.
8. The record-enumeration pinned-glob-vs-`find` discriminating case →
   TC-018s.
9. AC-28's mechanical form (not "someone runs `make test`") → TC-020.

These are recorded here as design inputs baked into the document below, per
Rule 12 (fail loud about what actually happened) — not presented as an
independent pass that read and passed judgment on the finished plan.

### Pass 2 — post-draft self-red-team, against the finalized document

With both CLI reviewers blocked, a second, genuinely independent-in-time
pass was run **after** the full 28-AC document was drafted, checking
specifically for defects a pre-draft design consultation structurally
cannot catch: internal inconsistencies, unstated environmental assumptions,
and self-defeating assertions introduced during the writing itself. This
pass found three:

- **CONCERN, fixed:** `TC-020`'s `git diff --stat <merge-base>..HEAD`
  named no way to resolve `<merge-base>`. Run from an arbitrary checkout
  (no `origin/main`, a shallow clone, a detached HEAD), the comparison base
  is undefined — the AC-28 check would be red or vacuously green depending
  on where it runs. Fixed: `git merge-base HEAD origin/main` with a logged
  clean skip of the diff-stat half (not a silent pass) when it can't
  resolve; the `run-all.sh`-registration half runs unconditionally, so the
  check degrades to partial coverage instead of going dark.
- **CONCERN, fixed:** `TC-018a`'s `LC_ALL=en_US.UTF-8` half assumed that
  locale exists in the test environment. A `C`-only CI container would
  make that half fail for an environmental reason unrelated to the
  property under test. Fixed: gated on `locale -a` listing the locale,
  else skipped and logged, mirroring `transcript_test.go`'s Windows-skip
  convention — the `LC_ALL=C` half always runs and is the one that must
  never skip.
- **CONCERN, fixed:** `TC-018l`'s (AC-17) negative case asserted the two
  aggregates were byte-identical "except wherever `quality.tests_pass` is
  echoed" — but `quality_tests_pass` is itself a registered Class A metric
  (per the Metric registry table), so flipping the source value on every
  rep *legitimately* changes that metric's `true_count`/`rate`/`accept_set`.
  A whole-document byte-identity assertion was self-defeating as written.
  Fixed: the assertion is scoped to the regression field alone (the actual
  REQ-F-014 property), not the whole aggregate.

No BLOCKER-class finding. All three CONCERNs are fixed in place above, in
the AC Test Matrix and Caller-Path Contracts sections, not left as
follow-ups.

**Issues raised (both passes):** 12 (9 pre-draft design inputs, 3 post-draft
document defects).
**Issues addressed:** 12 of 12 — none deferred, none left as a dangling
"flagged for later" without a named TC or an in-place text correction.
**Issues deferred:** 0.

This two-pass structure is recorded explicitly because the two passes catch
different defect classes: a pre-draft consultation shapes what gets built
but cannot review a document that doesn't exist yet; only a pass against
the finished artifact can catch defects the writing itself introduces (an
unresolvable merge-base, an environment-dependent locale assumption, a
self-defeating negative-case scope). Collapsing the two into one "red-team
found 9 things" claim would have overstated what the pre-draft consultation
actually verified.

---

## Recommendations

- [x] **Ready for development** — every AC in `spec.md` has a named test
  case, ISTQB technique, ISO 25010 row, and Caller-Path Contract. The one
  declared cross-feature interaction (I-02) reuses its fixed-name contract
  test verbatim with no twin, and this plan's `tc018` suite is the real
  caller path that discharges F03's consumer-side task obligation. The one
  feature area with no cross-epic seam (X-07/X-08/X-09) is verified
  against `spec.md`'s own table, not silently skipped. Nine REQ-indexed
  gaps this plan's own drift analysis and red-team pass found (five-field
  uniformity check, family-presence-vs-`sources` independence, digest
  computed-not-echoed, corpus-feedback naming E40-F01 by content, the
  RUN_ONE_BIN default-resolution defect class, the pinned-glob-vs-`find`
  discriminating case, `batch-log.jsonl`'s non-read property, the
  both-drift-differ replay cell, and the aggregate-side unsupported-schema
  exit path) are all closed with a named TC, not deferred. Fixture
  provenance (REQ-N-006/ADR-F03-08) is pinned to a committed generator with
  an explicit rewrite/preserve field list, closing the gap between "derived
  from the goldens" as prose and as a falsifiable property.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.

One implementation-shape item is flagged for task generation to resolve
explicitly, matching F02's/F04's own precedent of naming such items rather
than leaving them implicit: the exact reachable-precondition ordering
inside `replay-manifest.sh`'s three REQ-F-027 checks (TC-019a's decision
table assumes independent, isolatable failures — task generation should
confirm the implementation checks all three and reports the *first*
failing one deterministically, or names every failing precondition at
once; either is testable against this plan's TC-019a as written, but the
task spec should state which).

## Verdict

**APPROVED.**

---

*Last updated*: 2026-08-07
