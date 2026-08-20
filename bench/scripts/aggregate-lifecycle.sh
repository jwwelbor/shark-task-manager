#!/usr/bin/env bash
# aggregate-lifecycle.sh --retention-root <retention_root>
#
# T-E40-F10-008 (spec.md REQ-F-007 (partial), REQ-F-009 (partial),
# REQ-F-010, REQ-F-011, REQ-F-017, ADR-F10-08, ADR-F10-11, REQ-NF-003;
# test-plan.md TC-084, TC-089). Pure lifecycle-v2 aggregator, modelled on
# `aggregate-runs.sh`'s header contract: prints one `aggregate.json`
# document to stdout, diagnostics to stderr, consults no clock, invokes no
# subprocess/provider/network, and touches nothing outside the named
# `--retention-root`.
#
# SLICE OWNERSHIP (component-changes row, this task's Scope): T-E40-F10-008
# built `identity`, `scenarios[]`, `time`, and `cost`. T-E40-F10-009 added
# `quality`, `review_value`, and `artifact_use`. T-E40-F10-010 (this
# extension) adds the final three blocks -- `noise_bands`, `comparisons`,
# and `invalid` -- completing all ten top-level blocks
# `bench/reports/lifecycle-baseline-schema.yaml` requires.
#
# ---------------------------------------------------------------------------
# Design decisions made by this task (documented here because spec.md does
# not fix them at this level of detail; each is a defensible reading of
# REQ-F-010/REQ-F-011/ADR-F10-04, not an invented business rule):
#
#   1. Whole-root Phase 1 refusal (ADR-F10-11, AC-T1). A file named exactly
#      `record.jsonl` ANYWHERE under the retention root (not just under
#      scenarios/) is Phase 1's fixed record filename -- F10 never retains
#      a file by that name (its own records are `lifecycle.jsonl` /
#      `evaluation.jsonl`) -- so its mere presence is refused with the
#      schema-owned `phase1_record_input_refused` reason before any other
#      work happens, per TC-089's resolution of its own deferred mixed-root
#      question: refuse the WHOLE root, never a partial aggregate.
#
#   2. `time.lifecycle_wall_seconds` / total provider cost are read
#      VERBATIM from each pair's retained `evaluation.jsonl` --
#      `metrics.elapsed_time.value` and `metrics.provider_cost.value`
#      (evaluate-lifecycle.sh's own `derive_metrics()` already computes
#      these as `sum(stage.elapsed_seconds)` / `sum(stage.cost_usd +
#      judge.cost_usd)`). ADR-F10-04 lists `metrics` among the fields F10
#      copies verbatim rather than recomputes; recomputing elapsed/cost
#      independently here would be exactly the "second, divergent
#      implementation" ADR-F10-04 forbids. The per-cell PARTITIONS
#      (stage_category / interval_category / share_partition), which I-08
#      does not carry, are computed here directly from retained I-07
#      `/stages[]/intervals` and reconciled against that verbatim total,
#      with any residual reported as `unattributed` -- this is what makes
#      the reconciliation in TC-084 a real check rather than a tautology.
#      A meaningful non-zero cost `unattributed` case is: a scenario with
#      an evaluated judge cost, since judge cost contributes to
#      `metrics.provider_cost.value` but is not attributable to any
#      (stage_category, interval_category) cell.
#
#   3. Cost-per-cell attribution rule. I-07 does not decompose a stage's
#      `cost_usd` per interval the way it decomposes elapsed time (a
#      stage's `intervals[]` durations already reconcile to its
#      `elapsed_seconds`, but nothing analogous exists for cost). Rather
#      than invent a pro-rata split across a stage's wait/tool/unclassified
#      time -- which would misrepresent provider spend as accruing during
#      idle wait -- this script attributes a stage's entire `cost_usd` to
#      the `(stage.category, "provider_active")` cell: cost is modelled as
#      accruing where the provider was doing the work, never during a wait
#      interval. This is why the REQ-F-011 `wait` share is always $0 for
#      cost by construction, which matches the domain reading "waiting
#      costs nothing" rather than being a computation gap.
#
#   4. `cost.ceiling_consumption.max_cost_usd` is the BATCH-declared
#      ceiling from `/identity/ceilings/max_cost_usd` (the single
#      `--max-cost-usd` value the operator acknowledged for the whole
#      batch, REQ-F-002), not a sum of each pair's own I-07
#      `limits.max_cost_usd` -- those are per-run resource-policy ceilings
#      (`run-lifecycle.sh`'s own `parse_limits()`, sourced from each
#      scenario package's `resource_policy` when `--limits` is not passed)
#      and may legitimately differ per scenario. `observed_cost_usd` is
#      the sum of every pair's I-07 `limits.observed_cost_usd`, i.e. what
#      was actually spent against that declared batch ceiling.
#
#   5. `retention_root_digest` is a digest of digests, not a full-tree
#      content hash: sha256 over the canonical JSON of `batch.json`'s own
#      raw-byte digest plus the sorted list of every retained pair's
#      `manifest.json` raw-byte digest. Every retained artifact's own byte
#      integrity is already covered by that pair's `manifest.json` sha256
#      entries (verify-retention-root.sh is the dedicated byte-preservation
#      checker); re-hashing the full evidence/transcripts payloads here
#      would cost O(retention root size) for no additional integrity
#      signal and would work against REQ-NF-004's streaming/scale
#      requirement for no benefit -- this aggregator does not otherwise
#      need to touch a single byte of evidence/ or transcripts/ content.
#
#   6. No per-scenario `time`/`cost` breakdown array. The `aggregate.json`
#      blocks table describes `time`/`cost` as "per scenario and rolled
#      up", but `bench/reports/lifecycle-baseline-schema.yaml`'s
#      `aggregate_required_fields` only requires the ROLLED-UP pointers
#      (`/time/lifecycle_wall_seconds`, `/time/stage_category`, ...) --
#      unlike `/quality/by_scenario[]`, which IS required. This task keeps
#      to the schema-required (rolled-up) shape rather than inventing an
#      untested `by_scenario` array; a later task may add one without
#      breaking this shape.
# ---------------------------------------------------------------------------
#
# Known upstream gap checked and found NOT BLOCKING for this task: T-007's
# retained finding that `run-lifecycle-batch.sh`'s `retain_pair` only
# populates manifest.json entries for 5 of 7 non-manifest artifacts (no
# evidence/transcripts entries) does not affect this script, because the
# identity/scenarios/time/cost blocks built here never read evidence/ or
# transcripts/ content or their manifest entries -- only package.yaml,
# lifecycle.jsonl, evaluation.jsonl, and manifest.json's entries for THOSE
# three names, all of which retain_pair already populates correctly with
# real source digests.
#
# A SEPARATE, previously undocumented gap in the same file WAS blocking and
# is fixed by this task (see run-lifecycle-batch.sh's batch.json writer):
# batch.json never retained the batch policy's declared `min_reps`, so
# `/identity/min_reps` (a field owned by THIS task) had no retained source
# to read from `--retention-root` alone. Fixed by re-parsing the same
# `batch_policy_path` that writer already opens for `batch_policy_digest`.
#
# ---------------------------------------------------------------------------
# T-E40-F10-009 design decisions (quality, review_value, artifact_use).
# spec.md REQ-F-012, REQ-F-013, ADR-F10-04, ADR-F10-10; test-plan.md TC-085,
# TC-086 (aggregator half); AC-T1, AC-T2, AC-T3.
#
#   7. `quality.by_scenario[].{structural,judge,execution_oracle}` are each
#      pair's I-08 block copied verbatim, presence-checked only -- their
#      interior is I-08's own vocabulary (ADR-F10-04); this script never
#      inspects `observed_result` or any nested field.
#      `quality.first_pass_yield` is the one DERIVED value the blocks table
#      names ("derived from the retained gate rounds"): among gates that
#      were reached (I-07 `review_gates[].state != not_reached`), the
#      fraction whose highest observed `round` is exactly 1. Zero reached
#      gates renders `unavailable`, never a `0/0` division or a bare `0`
#      (ADR-F10-10).
#
#   8. `review_value.gates[]` gate identity and `state` come from I-07
#      `review_gates[]` (byte-preserved, closed `gate_state` vocabulary:
#      findings / zero_findings / collection_failure / not_reached,
#      `bench/runs/i07-schema.yaml`) -- NEVER inferred from whether I-08
#      `metrics.quality.review_measures` happens to contain entries for that
#      gate, because a `zero_findings` gate is, by construction, absent from
#      `review_measures` (grouping is per finding) and inferring its state
#      from that absence would collapse it with `collection_failure` -- the
#      exact confusion AC-T2/ADR-F10-10 forbid. The seven finding measures
#      themselves (`review_value.gates[].counts`/`counts_by_severity`/
#      `counts_by_defect_class`) are read verbatim from I-08
#      `metrics.quality.review_measures[]`, joined to a gate on `gate`, using
#      I-08's OWN measure-key names (`normalized_unique`, not "unique" --
#      test-plan.md TC-085's ISO matrix: "keeps finding-measure vocabulary
#      I-08-owned").
#
#      `truth_set_available`/`precision`/`recall` are declared once PER
#      EVALUATION RECORD in I-08 (`metrics.quality.truth_set_status` /
#      `.precision` / `.recall`), not per gate -- `normalize-review-findings.sh`
#      computes them from the whole run's `truth_set`/`finding_ids`, with no
#      per-gate decomposition anywhere upstream. This script therefore
#      applies one pair's verbatim truth/precision/recall to every gate that
#      pair's `review_gates[]` contributed. This is the correct verbatim
#      reading of I-08's actual (not aspirational) shape, not an invented
#      per-gate rule -- documented here because a future reader may expect
#      per-gate precision and should know why it is not synthesized. Never
#      `0`; the metric's own `available` flag decides `unavailable`.
#
#      `elapsed_seconds`/`provider_cost_usd`/`resolution_cost_usd` per gate
#      render `unavailable` unconditionally: neither I-07 `review_gates[]`
#      (gate_id, state, round, findings, candidate_ref, policy_ref -- no
#      time/cost field) nor I-08 `metrics` carries a gate-level time/cost
#      figure, and `review_gates[].candidate_ref` is an opaque reference, not
#      a join key into `stages[].candidate` -- checked and confirmed absent
#      (grep across `bench/scripts/*.sh` and `i07-schema.yaml`). Attributing
#      a `review`/`qa`/`uat` stage's cost to "the" gate would invent a
#      one-to-one mapping I-07 does not assert (a run may carry several
#      review-category stages and several gates with no declared
#      correspondence), which ADR-F10-04 forbids. ADR-F10-10's rule --
#      absent measures render `unavailable`, never zero -- is stated
#      generally, not scoped to precision/recall alone, and applies here.
#
#   9. `artifact_use` produced/consumed/reused/orphan counts and typed
#      producer/consumer `edges` are computed directly from every pair's
#      retained I-07 `stages[].artifacts[]` (mirroring design decision 2's
#      pattern for time/cost: a verbatim I-08 scalar total, reconciled
#      against a finer breakdown this script computes from retained I-07
#      data). `artifact_type`/`edges[].edge_kind` are validated against
#      `bench/evidence/i05-schema.yaml`'s closed vocabularies (I-07 artifact
#      records reuse I-05's sets). Per `bench/README.md`'s stage-snapshot
#      field reference, an artifact's `consumers` key has THREE distinct,
#      never-coerced states this script keeps distinct: `consumers: []`
#      (a true orphan -- produced, no consumer observed), an ABSENT
#      `consumers` key (consumption evidence not collected -- neither
#      consumed nor a proven orphan), and a non-empty `consumers[]`
#      (consumed; `reused` when it names more than one DISTINCT
#      `consuming_stage`, since two edges from the very same consuming stage
#      is not reuse). I-08's own `metrics.artifact_use.orphaned` value
#      (`derive_metrics()`: `len(artifacts) - consumed`) CONFLATES the first
#      two states into one number -- this script's reconciliation therefore
#      compares I-08's verbatim `orphaned` value against
#      `(this script's orphan_count + evidence_missing_count)`, never against
#      `orphan_count` alone, so a real F09 gap is reported, never masked as
#      a mismatch. A genuine value MISMATCH is a non-fatal upstream contract
#      defect printed to stderr, exactly like design decision 2's
#      rework-mismatch precedent -- never a refusal. `metrics.artifact_use`
#      is OPTIONAL, unlike `elapsed_time`/`provider_cost`/`rework`
#      (verbatim_metric() hard-requires those); its mere absence -- true of
#      every T-E40-F10-008 fixture, which predates this task -- is silent,
#      matching TC-084's established "no stderr on a clean run" contract,
#      and the computed I-07-derived counts stand alone.
#
#  10. `artifact_use.replayed_interaction_proxy` (REQ-F-013's D01-D05 proxy
#      counts) is read, when present, from
#      `metrics.artifact_use.replayed_interaction_proxies` -- the shape
#      spec.md describes ("from the I-05/I-06 lineage carried in I-08
#      metrics.artifact_use") but the CURRENT `evaluate-lifecycle.sh` does
#      not yet populate (checked: no `replayed_interaction_proxies` key is
#      written anywhere in that script today). This is a genuine, checked
#      upstream gap, not a misreading on this script's part -- spec.md line
#      371-375 forbids F10 from reading I-06 directly, so there is no other
#      reachable source. Per ADR-F10-10, an unpopulated proxy block renders
#      every numeric field `unavailable` (never `0` or a synthesized
#      figure); its universal absence is silent, matching TC-084's
#      established "no stderr on a clean run" contract (a non-fatal
#      diagnostic fires only for PARTIAL coverage -- some retained pairs
#      carry the sub-object and others do not, an actual inconsistency, not
#      a uniform gap), while still emitting the schema-required single
#      `label` field so the block shape is always present (AC-T3). When the
#      sub-object IS present, this
#      script maps I-06's OWN field names 1:1 onto F10's schema names --
#      `authorized_request_count` -> `request_count`,
#      `authorized_response_count` -> `response_count`,
#      `revision_or_replacement_count` -> `revision_count`,
#      `unresolved_gate_count` -> `unresolved_gate_count` -- and derives
#      `payload_size_bytes` as `request_bytes_total + response_bytes_total`
#      (I-06 tracks the two directions separately; F10's schema names one
#      combined byte figure, so this script states the sum rule here rather
#      than leaving it to a reader to guess). Every field's replayed-proxy
#      identity is carried by the single container-level `label` field the
#      schema defines (`/artifact_use/replayed_interaction_proxy/label`) --
#      one label for the block, not one per numeric field, matching the
#      schema's own field inventory. This script never phrases any of this
#      as an observed-human measurement (`forbidden_effort_language`,
#      REQ-F-013): every count here is a replayed-interaction count, never
#      an observed-interaction measurement.
# ---------------------------------------------------------------------------
# T-E40-F10-010 design decisions (noise_bands, comparisons, invalid).
# spec.md REQ-F-007, REQ-F-008, REQ-F-016, ADR-F10-12; test-plan.md TC-088
# (aggregator half); AC-T1, AC-T2.
#
#  11. `noise_bands[]` metric set. REQ-F-007 requires "per-(scenario,
#      metric) noise bands" without naming which metrics; report-lifecycle.sh
#      (T-E40-F10-011) "computes no new statistic" -- every value it renders
#      for its three-dimension (quality/time/cost) paired-delta view must
#      already exist here. This script therefore publishes a band for
#      exactly three metrics, one per REQ-F-016 dimension, each read from an
#      ALREADY-VERBATIM I-08 metric value used elsewhere in this same
#      script -- never a second read or a different semantics:
#      `elapsed_time` (time) and `provider_cost` (cost) are `pair_elapsed`/
#      `pair_cost`, the same verbatim `metrics.elapsed_time`/
#      `metrics.provider_cost` values contributing to `/time`/`/cost`;
#      `confirmed_findings` (quality) is I-08's own
#      `metrics.quality.confirmed_findings` (design decision 8's
#      `quality_metric_value()` helper, same pattern as precision/recall) --
#      a genuine numeric, already-retained, direction-consistent (fewer
#      confirmed findings is better, same as less time/cost) quality proxy,
#      not an invented statistic. The metric identifiers are I-08's own
#      metric-key names verbatim (REQ-F-018), never a private rename.
#
#  12. Noise-band formula: reused verbatim from `aggregate-runs.sh`'s Class C
#      `compute_stats` (min/median/max/spread_abs, then an acceptance radius
#      equal to the observed spread when non-zero, else 10% of |median| when
#      the median itself is non-zero, else exactly zero) -- ADR-F10-12
#      explicitly reuses this "existing flag semantics" rather than
#      inventing a second statistic. What ADR-F10-12 does NOT reuse is
#      `aggregate-runs.sh`'s own hard-coded `n < 2` threshold: F10's
#      `insufficient_reps` (AC-T1) compares each band's contributing rep
#      count against the BATCH-declared `/identity/min_reps` (already read
#      for `/identity` above), because ADR-F10-12 fixes the mechanism, not
#      the number -- the number is operator batch policy. A metric whose
#      contributing pairs are all excluded (n=0 -- only reachable for
#      `confirmed_findings`, since `elapsed_time`/`provider_cost` are
#      hard-required by `verbatim_metric()` above) renders `min`/`median`/
#      `max`/`spread_abs`/`acceptance_interval` as `unavailable`, never a
#      fabricated zero-width band (ADR-F10-10), while still carrying
#      `rep_count: 0` and `insufficient_reps: true` so the gap is visible,
#      not silent.
#
#  13. `/comparisons` republishes each pair's OPTIONAL retained
#      `comparison.json` (`retention_optional_artifacts`,
#      `compare-lifecycle-evaluations.sh`'s own output shape) verbatim and
#      in full -- mode, both evaluation ids, the accept/reject verdict, the
#      `comparison` sub-object, and every `divergences[]` entry (REQ-F-015).
#      This script never re-derives `accepted` or filters divergences: the
#      comparator's own `candidate/*` and `workflow_policy/*` divergence
#      field-paths ARE the "candidate identity references, and policy
#      identity references" the blocks table names, so no second
#      identity-reference computation is needed or performed here
#      (ADR-F10-04).
#
#      `/invalid` has TWO sources, because `run-lifecycle-batch.sh` never
#      inspects I-08 eligibility (checked: no reference to
#      `aggregate_eligible`/`eligibility` anywhere in that script), so its
#      own `invalid/index.jsonl` ledger (source a) covers ONLY pairs that
#      never reached a retained `evaluation.jsonl` at all (dispatch/run/
#      eval failures) -- mirrored verbatim, per the blocks table ("mirroring
#      `invalid/index.jsonl`"), adding no reason code of this script's own.
#      A pair that DID dispatch successfully but that I-08 itself flagged
#      ineligible would otherwise be invisible to `/invalid` entirely, which
#      REQ-F-007's "an explicit invalid-run inventory carrying every
#      `invalidity_reasons` entry" (emphasis on EVERY) forbids. Source (b)
#      therefore adds one row per `scenarios[]` entry whose I-08
#      `eligibility.invalidity_reasons` is non-empty, reusing the SAME
#      `eligibility` object already read verbatim for `/scenarios` earlier
#      in this script (AC-T2: no recomputation, override, or softening --
#      not a second read of `evaluation.jsonl`). The two sources are
#      distinguished by a `source` field (`batch_dispatch_failure` /
#      `i08_eligibility`) rather than merged into one ambiguous shape,
#      because they carry genuinely different upstream provenance and the
#      dispatch-failure rows have no I-08 `eligibility` object to nest at
#      all.
# ---------------------------------------------------------------------------
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

LIFECYCLE_SCHEMA="${LIFECYCLE_SCHEMA:-$BENCH_DIR/reports/lifecycle-baseline-schema.yaml}"
I05_SCHEMA="${I05_SCHEMA:-$BENCH_DIR/evidence/i05-schema.yaml}"
I07_SCHEMA="${I07_SCHEMA:-$BENCH_DIR/runs/i07-schema.yaml}"

usage() {
	echo "usage: aggregate-lifecycle.sh --retention-root <retention_root>" >&2
	exit 2
}

retention_root=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--retention-root)
		[[ $# -ge 2 ]] || usage
		retention_root="$2"
		shift 2
		;;
	--help)
		usage
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$retention_root" ]] || usage
[[ -d "$retention_root" ]] || {
	echo "aggregate-lifecycle: retention root not found: $retention_root" >&2
	exit 2
}
[[ -f "$LIFECYCLE_SCHEMA" ]] || {
	echo "aggregate-lifecycle: schema not found: $LIFECYCLE_SCHEMA" >&2
	exit 2
}
[[ -f "$I05_SCHEMA" ]] || {
	echo "aggregate-lifecycle: i05 schema not found: $I05_SCHEMA" >&2
	exit 2
}
[[ -f "$I07_SCHEMA" ]] || {
	echo "aggregate-lifecycle: i07 schema not found: $I07_SCHEMA" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "aggregate-lifecycle: python3 not found on PATH" >&2
	exit 2
}

RETENTION_ROOT_CANON="$(cd "$retention_root" && pwd)"

python3 - "$RETENTION_ROOT_CANON" "$LIFECYCLE_SCHEMA" "$I05_SCHEMA" "$I07_SCHEMA" <<'PYEOF'
import hashlib
import json
import os
import statistics
import sys

import yaml

retention_root, schema_path, i05_schema_path, i07_schema_path = sys.argv[1:5]


def fail(msg):
    print("aggregate-lifecycle: " + msg, file=sys.stderr)
    sys.exit(1)


def usage_fail(msg):
    print("aggregate-lifecycle: " + msg, file=sys.stderr)
    sys.exit(2)


# ---------------------------------------------------------------------------
# Phase 0: whole-root Phase 1 v1-record refusal (ADR-F10-11, AC-T1, TC-089).
# A recursive walk, not the scenarios/*/*/ pinned layout -- "anywhere under
# it" per REQ-F-017/AC-T1, including a record.jsonl injected outside the
# scenarios/ tree. This is the ONLY thing this script does before any other
# read, so an accidental Phase 1 root never gets even a partial aggregate.
# ---------------------------------------------------------------------------
v1_records = []
for current, dirs, files in os.walk(retention_root):
    dirs.sort()
    if "record.jsonl" in files:
        v1_records.append(os.path.relpath(os.path.join(current, "record.jsonl"), retention_root))
if v1_records:
    for rel in sorted(v1_records):
        print(
            f"aggregate-lifecycle: refusing entire aggregation: phase1_record_input_refused: "
            f"Phase 1 record.jsonl found at {rel} (ADR-F10-11: whole-root refusal, not a partial aggregate)",
            file=sys.stderr,
        )
    sys.exit(1)

# ---------------------------------------------------------------------------
# Closed vocabularies, read from schema (REQ-F-018) rather than duplicated.
# ---------------------------------------------------------------------------
with open(schema_path, encoding="utf-8") as f:
    schema = yaml.safe_load(f) or {}
with open(i05_schema_path, encoding="utf-8") as f:
    i05_schema = yaml.safe_load(f) or {}
with open(i07_schema_path, encoding="utf-8") as f:
    i07_schema = yaml.safe_load(f) or {}

SCHEMA_VERSION = schema.get("schema_version")
if not SCHEMA_VERSION:
    usage_fail(f"{schema_path}: missing schema_version")

PHASE_LABELS = schema.get("phase_label") or []
PHASE_LABEL = "lifecycle_v2"
if PHASE_LABEL not in PHASE_LABELS:
    usage_fail(f"{schema_path}: does not declare phase label {PHASE_LABEL!r}")

STAGE_CATEGORIES = i05_schema.get("stage_category") or []
INTERVAL_CATEGORIES = i05_schema.get("interval_category") or []
SHARE_CELLS = schema.get("share_partition_cell") or []
UNATTRIBUTED = schema.get("share_partition_residual", "unattributed")
if not STAGE_CATEGORIES or not INTERVAL_CATEGORIES or not SHARE_CELLS:
    usage_fail("schema vocabularies (stage_category/interval_category/share_partition_cell) are empty or missing")

# T-E40-F10-009: artifact/edge vocabularies (design decision 9) and the gate
# state vocabulary (design decision 8), all read from their single owners
# rather than duplicated (REQ-F-018).
ARTIFACT_TYPES = i05_schema.get("artifact_type") or []
EDGE_KINDS = i05_schema.get("edge_kind") or []
GATE_STATES = i07_schema.get("gate_state") or []
if not ARTIFACT_TYPES or not EDGE_KINDS or not GATE_STATES:
    usage_fail("schema vocabularies (artifact_type/edge_kind/gate_state) are empty or missing")

# REQ-F-012: the seven per-gate finding measures, using I-08's OWN measure
# key names verbatim (design decision 8) -- `evaluate-lifecycle.sh`'s
# `derive_metrics()` bucket keys, never a private renaming.
MEASURE_KEYS = (
    "emitted",
    "normalized_unique",
    "duplicate",
    "recurrent",
    "confirmed",
    "unconfirmed",
    "downstream_escape",
)


def zero_measures():
    return {key: 0 for key in MEASURE_KEYS}


# T-E40-F10-010, design decision 11: the three per-(scenario, metric) noise
# bands this aggregator publishes, one per REQ-F-016 dimension, each named
# with I-08's own metric-key name (REQ-F-018).
NOISE_BAND_METRICS = ("elapsed_time", "provider_cost", "confirmed_findings")

NOISE_BAND_DERIVATION_RULES = schema.get("noise_band_derivation_rule") or []
if not NOISE_BAND_DERIVATION_RULES:
    usage_fail("schema noise_band_derivation_rule is empty or missing")
NOISE_BAND_DERIVATION_RULE = NOISE_BAND_DERIVATION_RULES[0]

REPLAYED_PROXY_LABEL = (
    "replayed_interaction_proxy: a replayed D01-D05 product-design "
    "interaction count, never an observed-interaction measurement (REQ-F-013)"
)

# REQ-F-011's exact share-assignment rule.
WAIT_INTERVALS = {"queue_or_claim_wait", "replay_or_human_gate_wait", "retry_or_backoff"}
PRE_CODE_STAGES = {"discovery", "specification", "planning"}
REVIEW_STAGES = {"review", "qa", "uat"}


def share_for_cell(stage_category, interval_category, rework):
    if interval_category in WAIT_INTERVALS:
        return "wait"
    if stage_category in PRE_CODE_STAGES:
        return "pre_code"
    if stage_category in REVIEW_STAGES:
        return "review"
    if stage_category == "shipping":
        return "shipping"
    if stage_category == "code":
        return "rework" if rework else "first_pass_code"
    return None  # genuinely unmappable; see TC-084 edge-case note -- never
    # reached for the eight current stage_category values, but the
    # partition-building code below still routes it to unattributed rather
    # than silently dropping it, in case a future vocabulary addition ever
    # produces one.


def sha256_bytes(data):
    return hashlib.sha256(data).hexdigest()


def sha256_file(path):
    with open(path, "rb") as fh:
        return sha256_bytes(fh.read())


def require(d, key, ctx):
    if not isinstance(d, dict) or key not in d or d[key] in (None, ""):
        fail(f"{ctx}: missing required field {key!r}")
    return d[key]


def read_one_line_json(path, ctx):
    try:
        with open(path, encoding="utf-8") as fh:
            lines = [ln for ln in (raw.strip() for raw in fh.readlines()) if ln]
    except OSError as exc:
        fail(f"{ctx}: cannot read {path}: {exc}")
    if len(lines) != 1:
        fail(f"{ctx}: {path}: expected exactly one JSON record line, got {len(lines)}")
    try:
        obj = json.loads(lines[0])
    except json.JSONDecodeError as exc:
        fail(f"{ctx}: {path}: unparseable JSON: {exc}")
    if not isinstance(obj, dict):
        fail(f"{ctx}: {path}: record is not a JSON object")
    return obj


def is_number(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool)


# ---------------------------------------------------------------------------
# /identity -- from batch.json, the sole retained identity source
# (REQ-NF-003: pure function of --retention-root alone).
# ---------------------------------------------------------------------------
batch_path = os.path.join(retention_root, "batch.json")
if not os.path.isfile(batch_path):
    fail("batch.json not found under retention root -- required for /identity")
with open(batch_path, "rb") as fh:
    batch_bytes = fh.read()
try:
    batch = json.loads(batch_bytes)
except json.JSONDecodeError as exc:
    fail(f"batch.json is not valid JSON: {exc}")

batch_id = require(batch, "batch_id", "batch.json")
batch_phase = require(batch, "phase", "batch.json")
if batch_phase != PHASE_LABEL:
    fail(f"batch.json declares phase {batch_phase!r}, expected {PHASE_LABEL!r}")
batch_policy_digest = require(batch, "batch_policy_digest", "batch.json")
acknowledgement_ref = require(batch, "acknowledgement_ref", "batch.json")

min_reps = batch.get("min_reps")
if not isinstance(min_reps, int) or isinstance(min_reps, bool) or min_reps < 1:
    fail(
        "batch.json missing a valid min_reps (declared batch minimum reps) -- "
        "see run-lifecycle-batch.sh's batch.json writer (T-E40-F10-008 fix)"
    )


def ceiling_number(raw_ceilings, name):
    value = raw_ceilings.get(name)
    if value is None:
        fail(f"batch.json ceilings.{name} is missing")
    try:
        text = str(value)
        if "." in text or "e" in text.lower():
            return float(value)
        return int(value)
    except (TypeError, ValueError):
        fail(f"batch.json ceilings.{name} is not numeric: {value!r}")


raw_ceilings = batch.get("ceilings") or {}
ceilings = {
    "max_cost_usd": ceiling_number(raw_ceilings, "max_cost_usd"),
    "max_wall_clock_seconds": ceiling_number(raw_ceilings, "max_wall_clock_seconds"),
    "max_generated_tasks": ceiling_number(raw_ceilings, "max_generated_tasks"),
}

# ---------------------------------------------------------------------------
# Enumerate retained (scenario, rep) pairs under scenarios/<id>/<rep>/,
# mirroring verify-retention-root.sh's own layout walk.
# ---------------------------------------------------------------------------
scenarios_dir = os.path.join(retention_root, "scenarios")
pair_dirs = []
if os.path.isdir(scenarios_dir):
    for scenario_id in sorted(os.listdir(scenarios_dir)):
        scenario_dir = os.path.join(scenarios_dir, scenario_id)
        if not os.path.isdir(scenario_dir):
            continue
        for rep in sorted(os.listdir(scenario_dir)):
            rep_dir = os.path.join(scenario_dir, rep)
            if not os.path.isdir(rep_dir):
                continue
            pair_dirs.append((scenario_id, rep, rep_dir))

if not pair_dirs:
    fail(f"no retained (scenario, rep) pairs found under {scenarios_dir}")

# Per-cell accumulators for the rolled-up time/cost blocks.
stage_time_totals = {cat: 0.0 for cat in STAGE_CATEGORIES}
interval_time_totals = {cat: 0.0 for cat in INTERVAL_CATEGORIES}
share_time_totals = {cell: 0.0 for cell in SHARE_CELLS}
total_lifecycle_wall_seconds = 0.0

stage_cost_totals = {cat: 0.0 for cat in STAGE_CATEGORIES}
interval_cost_totals = {cat: 0.0 for cat in INTERVAL_CATEGORIES}
share_cost_totals = {cell: 0.0 for cell in SHARE_CELLS}
total_provider_cost_usd = 0.0
total_observed_ceiling_cost = 0.0

scenarios_list = []
manifest_digest_entries = []

# T-E40-F10-009 accumulators (design decisions 7-10).
quality_by_scenario = []
gate_records = {}  # gate_id -> {state, round, truth_set_available, precision, recall} (resolved after the pair loop)
gate_occurrences = {}  # gate_id -> list of per-pair occurrences, resolved after the pair loop
review_measures_by_gate = {}  # gate_id -> list of {severity, defect_class, measures}

artifact_produced_count = 0
artifact_consumed_count = 0
artifact_orphan_count = 0
artifact_evidence_missing_count = 0
artifact_reused_count = 0
artifact_edges = []

proxy_totals = {
    "authorized_request_count": 0,
    "authorized_response_count": 0,
    "request_bytes_total": 0,
    "response_bytes_total": 0,
    "revision_or_replacement_count": 0,
    "unresolved_gate_count": 0,
}
proxy_pairs_with_data = 0
proxy_pairs_total = 0

# T-E40-F10-010 accumulators (design decisions 11-13).
noise_metric_values = {}  # (scenario_id, metric_name) -> [contributing numeric values]
comparisons_list = []

for scenario_id, rep, rep_dir in pair_dirs:
    pair_label = f"{scenario_id}/{rep}"

    package_path = os.path.join(rep_dir, "package.yaml")
    lifecycle_path = os.path.join(rep_dir, "lifecycle.jsonl")
    evaluation_path = os.path.join(rep_dir, "evaluation.jsonl")
    manifest_path = os.path.join(rep_dir, "manifest.json")
    for name, path in (
        ("package.yaml", package_path),
        ("lifecycle.jsonl", lifecycle_path),
        ("evaluation.jsonl", evaluation_path),
        ("manifest.json", manifest_path),
    ):
        if not os.path.isfile(path):
            fail(f"{pair_label}: required retained artifact missing: {name}")

    try:
        with open(package_path, encoding="utf-8") as fh:
            package = yaml.safe_load(fh) or {}
    except (OSError, yaml.YAMLError) as exc:
        fail(f"{pair_label}: cannot read package.yaml: {exc}")
    if not isinstance(package, dict):
        fail(f"{pair_label}: package.yaml is not a YAML mapping")
    family = require(package, "entity_family", f"{pair_label} package.yaml")
    scenario_version = require(package, "scenario_version", f"{pair_label} package.yaml")

    lc = read_one_line_json(lifecycle_path, pair_label)
    ev = read_one_line_json(evaluation_path, pair_label)

    lc_identity = lc.get("identity") or {}
    if str(lc_identity.get("scenario_id")) != scenario_id:
        fail(
            f"{pair_label}: lifecycle.jsonl identity.scenario_id "
            f"{lc_identity.get('scenario_id')!r} disagrees with retention path"
        )

    outcome = require(lc, "outcome", f"{pair_label} lifecycle.jsonl")
    eligibility = require(ev, "eligibility", f"{pair_label} evaluation.jsonl")
    for key in ("aggregate_eligible", "publication_eligible", "invalidity_reasons"):
        if key not in eligibility:
            fail(f"{pair_label}: evaluation.jsonl eligibility.{key} missing")

    try:
        with open(manifest_path, encoding="utf-8") as fh:
            manifest = json.load(fh)
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"{pair_label}: manifest.json is not valid JSON: {exc}")
    artifacts_manifest = manifest.get("artifacts") or {}
    source_digests = {}
    for name, entry in sorted(artifacts_manifest.items()):
        if isinstance(entry, dict) and entry.get("sha256"):
            source_digests[name] = entry["sha256"]
    manifest_digest_entries.append(
        {"scenario_id": scenario_id, "rep": rep, "manifest_sha256": sha256_file(manifest_path)}
    )

    try:
        rep_int = int(rep)
    except ValueError:
        fail(f"{pair_label}: retention directory rep {rep!r} is not an integer")

    scenarios_list.append(
        {
            "scenario_id": scenario_id,
            "scenario_version": scenario_version,
            "family": family,
            "rep": rep_int,
            "retention_path": f"scenarios/{scenario_id}/{rep}",
            "eligibility": eligibility,
            "outcome": outcome,
            "source_digests": source_digests,
        }
    )

    # -----------------------------------------------------------------
    # /quality.by_scenario[] -- verbatim I-08 structural/judge/
    # execution_oracle verdicts (design decision 7). Presence-checked
    # only; interior vocabulary belongs to I-08.
    # -----------------------------------------------------------------
    structural = require(ev, "structural", f"{pair_label} evaluation.jsonl")
    judge = require(ev, "judge", f"{pair_label} evaluation.jsonl")
    execution_oracle = require(ev, "execution_oracle", f"{pair_label} evaluation.jsonl")
    quality_by_scenario.append(
        {
            "scenario_id": scenario_id,
            "rep": rep_int,
            "structural": structural,
            "judge": judge,
            "execution_oracle": execution_oracle,
        }
    )

    # -----------------------------------------------------------------
    # /review_value.gates[] gate identity/state -- verbatim I-07
    # review_gates[] (design decision 8), never inferred from I-08's
    # per-finding grouping (AC-T2).
    # -----------------------------------------------------------------
    review_gates = lc.get("review_gates")
    if not isinstance(review_gates, list):
        fail(f"{pair_label}: lifecycle.jsonl review_gates is not a list")

    metrics = ev.get("metrics") or {}
    quality_metrics = metrics.get("quality")
    if isinstance(quality_metrics, dict):
        truth_set_available = quality_metrics.get("truth_set_status") == "available"

        def quality_metric_value(name):
            entry = quality_metrics.get(name)
            if isinstance(entry, dict) and entry.get("available", False):
                return entry.get("value")
            return "unavailable"

        pair_precision = quality_metric_value("precision")
        pair_recall = quality_metric_value("recall")
        pair_confirmed_findings = quality_metric_value("confirmed_findings")
    else:
        truth_set_available = False
        pair_precision = "unavailable"
        pair_recall = "unavailable"
        pair_confirmed_findings = "unavailable"

    # T-E40-F10-010, design decision 11: the quality noise-band contribution
    # for this pair -- only a genuine retained integer contributes (mirrors
    # aggregate-runs.sh's "excluded reps never enter compute_stats" rule);
    # an "unavailable" pair_confirmed_findings is silently excluded, never
    # coerced to 0.
    if isinstance(pair_confirmed_findings, int) and not isinstance(pair_confirmed_findings, bool):
        noise_metric_values.setdefault((scenario_id, "confirmed_findings"), []).append(pair_confirmed_findings)

    pair_gate_ids = set()
    for g_idx, gate in enumerate(review_gates):
        if not isinstance(gate, dict):
            fail(f"{pair_label}: review_gates[{g_idx}] is not an object")
        gate_id = require(gate, "gate_id", f"{pair_label} review_gates[{g_idx}]")
        pair_gate_ids.add(gate_id)
        gate_state = gate.get("state")
        if gate_state not in GATE_STATES:
            fail(f"{pair_label}: review_gates[{g_idx}].state {gate_state!r} not in closed gate_state vocabulary")
        gate_round = gate.get("round")
        if gate_state != "not_reached":
            if not isinstance(gate_round, int) or isinstance(gate_round, bool) or gate_round < 1:
                fail(f"{pair_label}: review_gates[{g_idx}] (gate {gate_id!r}) round must be a positive integer for a reached gate, got {gate_round!r}")

        # A gate_id repeating across pairs is the NORMAL case for a
        # multi-rep batch (min_reps > 1 dispatches the same scenario's same
        # gates repeatedly) -- collected here, not merged in place, so
        # resolution after the full loop can tell an ordinary same-scenario
        # repetition apart from a genuine cross-scenario gate_id clash, and
        # can detect actual disagreement rather than diagnosing on mere
        # repetition (see resolution below, after the pair loop).
        gate_occurrences.setdefault(gate_id, []).append(
            {
                "scenario_id": scenario_id,
                "pair_label": pair_label,
                "state": gate_state,
                "round": gate_round,
                "truth_set_available": truth_set_available,
                "precision": pair_precision,
                "recall": pair_recall,
            }
        )

    review_measures = quality_metrics.get("review_measures") if isinstance(quality_metrics, dict) else None
    for m_idx, entry in enumerate(review_measures or []):
        if not isinstance(entry, dict):
            fail(f"{pair_label}: metrics.quality.review_measures[{m_idx}] is not an object")
        measure_gate_id = require(entry, "gate", f"{pair_label} metrics.quality.review_measures[{m_idx}]")
        severity = require(entry, "severity", f"{pair_label} metrics.quality.review_measures[{m_idx}]")
        defect_class = require(entry, "defect_class", f"{pair_label} metrics.quality.review_measures[{m_idx}]")
        raw_measures = require(entry, "measures", f"{pair_label} metrics.quality.review_measures[{m_idx}]")
        counted = {}
        for key in MEASURE_KEYS:
            measure_entry = raw_measures.get(key) if isinstance(raw_measures, dict) else None
            if not isinstance(measure_entry, dict) or not measure_entry.get("available", False):
                fail(
                    f"{pair_label}: metrics.quality.review_measures[{m_idx}].measures.{key} "
                    f"unavailable -- cannot build review_value.gates (REQ-F-012)"
                )
            value = measure_entry.get("value")
            if not isinstance(value, int) or isinstance(value, bool):
                fail(f"{pair_label}: metrics.quality.review_measures[{m_idx}].measures.{key}.value is not an integer")
            counted[key] = value
        if measure_gate_id not in pair_gate_ids:
            print(
                f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-012): {pair_label}: "
                f"metrics.quality.review_measures references gate {measure_gate_id!r} which is "
                f"absent from THIS pair's lifecycle.jsonl review_gates[]; measures skipped, not "
                f"resolved locally",
                file=sys.stderr,
            )
            continue
        review_measures_by_gate.setdefault(measure_gate_id, []).append(
            {"severity": severity, "defect_class": defect_class, "measures": counted}
        )

    # -----------------------------------------------------------------
    # /time, /cost -- verbatim I-08 totals (ADR-F10-04) reconciled
    # against per-cell sums computed here from retained I-07 stages.
    # -----------------------------------------------------------------

    def verbatim_metric(name):
        entry = metrics.get(name)
        if not isinstance(entry, dict) or not entry.get("available", False):
            fail(
                f"{pair_label}: evaluation.jsonl metrics.{name} unavailable -- "
                f"cannot build time/cost blocks (upstream contract gap, REQ-F-008)"
            )
        return entry["value"]

    pair_elapsed = verbatim_metric("elapsed_time")
    pair_cost = verbatim_metric("provider_cost")
    pair_rework_rollup = verbatim_metric("rework")
    if not is_number(pair_elapsed):
        fail(f"{pair_label}: metrics.elapsed_time.value is not numeric")
    if not is_number(pair_cost):
        fail(f"{pair_label}: metrics.provider_cost.value is not numeric")
    if not isinstance(pair_rework_rollup, int) or isinstance(pair_rework_rollup, bool):
        fail(f"{pair_label}: metrics.rework.value is not an integer")

    total_lifecycle_wall_seconds += pair_elapsed
    total_provider_cost_usd += pair_cost

    # T-E40-F10-010, design decision 11: elapsed_time/provider_cost noise-
    # band contributions -- the SAME verbatim values feeding /time, /cost.
    noise_metric_values.setdefault((scenario_id, "elapsed_time"), []).append(pair_elapsed)
    noise_metric_values.setdefault((scenario_id, "provider_cost"), []).append(pair_cost)

    limits = lc.get("limits") or {}
    observed_cost = limits.get("observed_cost_usd")
    if not is_number(observed_cost):
        fail(f"{pair_label}: lifecycle.jsonl limits.observed_cost_usd is not numeric")
    total_observed_ceiling_cost += observed_cost

    stages = lc.get("stages")
    if not isinstance(stages, list):
        fail(f"{pair_label}: lifecycle.jsonl stages is not a list")

    pair_rework_stage_count = 0
    pair_produced_count = 0
    pair_consumed_count = 0
    pair_orphan_count = 0
    pair_evidence_missing_count = 0
    pair_reused_count = 0
    for idx, stage in enumerate(stages):
        if not isinstance(stage, dict):
            fail(f"{pair_label}: stages[{idx}] is not an object")
        category = stage.get("category")
        if category not in STAGE_CATEGORIES:
            fail(f"{pair_label}: stages[{idx}].category {category!r} not in closed stage_category vocabulary")
        rework = stage.get("rework")
        if not isinstance(rework, bool):
            # ADR-F10-08: rework MUST be read as retained, never inferred --
            # an absent/non-boolean flag is a schema violation, not a
            # silent default to False (TC-084 Negative Cases).
            fail(f"{pair_label}: stages[{idx}].rework must be a retained boolean, got {rework!r}")
        cost_usd = stage.get("cost_usd")
        if not is_number(cost_usd):
            fail(f"{pair_label}: stages[{idx}].cost_usd is not numeric")
        intervals = stage.get("intervals")
        if not isinstance(intervals, list):
            fail(f"{pair_label}: stages[{idx}].intervals is not a list")

        if category == "code" and rework:
            pair_rework_stage_count += 1

        for j, interval in enumerate(intervals):
            if not isinstance(interval, dict):
                fail(f"{pair_label}: stages[{idx}].intervals[{j}] is not an object")
            icat = interval.get("category")
            if icat not in INTERVAL_CATEGORIES:
                fail(
                    f"{pair_label}: stages[{idx}].intervals[{j}].category {icat!r} "
                    f"not in closed interval_category vocabulary"
                )
            start = interval.get("start")
            end = interval.get("end")
            if not is_number(start) or not is_number(end):
                fail(f"{pair_label}: stages[{idx}].intervals[{j}] start/end must be numeric")
            duration = end - start
            if duration < 0:
                fail(f"{pair_label}: stages[{idx}].intervals[{j}] end precedes start")

            stage_time_totals[category] += duration
            interval_time_totals[icat] += duration
            share = share_for_cell(category, icat, rework)
            if share is not None:
                share_time_totals[share] += duration

        # Cost design decision 3 (see header): a stage's entire cost_usd
        # is attributed to (category, "provider_active"), never pro-rated
        # across its wait/tool/unclassified intervals.
        stage_cost_totals[category] += cost_usd
        interval_cost_totals["provider_active"] += cost_usd
        cost_share = share_for_cell(category, "provider_active", rework)
        if cost_share is not None:
            share_cost_totals[cost_share] += cost_usd

        # -------------------------------------------------------------
        # /artifact_use -- produced/consumed/reused/orphan counts and
        # typed producer/consumer edges, computed from retained I-07
        # artifacts (design decision 9). Three never-coerced consumer
        # states, per bench/README.md's stage-snapshot field reference:
        # `consumers` absent (evidence not collected), `consumers: []`
        # (a true orphan), and a non-empty `consumers[]` (consumed).
        # -------------------------------------------------------------
        artifacts = stage.get("artifacts")
        if not isinstance(artifacts, list):
            fail(f"{pair_label}: stages[{idx}].artifacts is not a list")
        producer_stage_name = stage.get("stage")
        for a_idx, artifact in enumerate(artifacts):
            if not isinstance(artifact, dict):
                fail(f"{pair_label}: stages[{idx}].artifacts[{a_idx}] is not an object")
            artifact_type = artifact.get("artifact_type")
            if artifact_type not in ARTIFACT_TYPES:
                fail(
                    f"{pair_label}: stages[{idx}].artifacts[{a_idx}].artifact_type "
                    f"{artifact_type!r} not in closed artifact_type vocabulary"
                )
            artifact_path = require(artifact, "path", f"{pair_label} stages[{idx}].artifacts[{a_idx}]")
            producer_stage = artifact.get("producer_stage", producer_stage_name)
            pair_produced_count += 1

            if "consumers" not in artifact:
                pair_evidence_missing_count += 1
                continue
            consumers = artifact["consumers"]
            if not isinstance(consumers, list):
                fail(f"{pair_label}: stages[{idx}].artifacts[{a_idx}].consumers is not a list")
            if not consumers:
                pair_orphan_count += 1
                continue

            pair_consumed_count += 1
            distinct_consuming_stages = set()
            for c_idx, consumer in enumerate(consumers):
                if not isinstance(consumer, dict):
                    fail(f"{pair_label}: stages[{idx}].artifacts[{a_idx}].consumers[{c_idx}] is not an object")
                consuming_stage = require(
                    consumer, "consuming_stage", f"{pair_label} stages[{idx}].artifacts[{a_idx}].consumers[{c_idx}]"
                )
                edge_kind = consumer.get("edge_kind")
                if edge_kind not in EDGE_KINDS:
                    fail(
                        f"{pair_label}: stages[{idx}].artifacts[{a_idx}].consumers[{c_idx}].edge_kind "
                        f"{edge_kind!r} not in closed edge_kind vocabulary"
                    )
                distinct_consuming_stages.add(consuming_stage)
                artifact_edges.append(
                    {
                        "producer_stage": producer_stage,
                        "artifact_path": artifact_path,
                        "artifact_type": artifact_type,
                        "consuming_stage": consuming_stage,
                        "edge_kind": edge_kind,
                    }
                )
            if len(distinct_consuming_stages) > 1:
                pair_reused_count += 1

    # REQ-F-011 / REQ-F-008: the rework share's contributing-stage count
    # must reconcile against I-08's metrics.rework rollup for THIS pair.
    # A mismatch is an upstream contract defect -- reported loudly, never
    # silently resolved by trusting either side (test-plan.md TC-084
    # Negative Cases).
    if pair_rework_stage_count != pair_rework_rollup:
        print(
            f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-008): {pair_label}: "
            f"rework stage count computed from retained I-07 stages "
            f"({pair_rework_stage_count}) does not match I-08 metrics.rework rollup "
            f"({pair_rework_rollup}); reported, not resolved locally",
            file=sys.stderr,
        )

    # REQ-F-013: reconcile the produced/consumed/(orphan+evidence_missing)
    # counts computed from retained I-07 artifacts against I-08's verbatim
    # metrics.artifact_use scalars for this pair (design decision 9).
    # I-08's own `orphaned` value conflates the true-orphan and
    # evidence-missing states, so it is compared against their SUM, never
    # against orphan_count alone. A missing metrics.artifact_use block
    # (every T-E40-F10-008 fixture) skips reconciliation with a diagnostic
    # rather than failing -- the computed I-07-derived counts stand alone.
    artifact_use_metric = metrics.get("artifact_use")
    if isinstance(artifact_use_metric, dict):
        def artifact_use_value(name):
            entry = artifact_use_metric.get(name)
            if isinstance(entry, dict) and entry.get("available", False):
                return entry.get("value")
            return None

        i08_produced = artifact_use_value("produced")
        i08_consumed = artifact_use_value("consumed")
        i08_orphaned = artifact_use_value("orphaned")
        mismatches = []
        if i08_produced is not None and i08_produced != pair_produced_count:
            mismatches.append(f"produced computed={pair_produced_count} i08={i08_produced}")
        if i08_consumed is not None and i08_consumed != pair_consumed_count:
            mismatches.append(f"consumed computed={pair_consumed_count} i08={i08_consumed}")
        pair_orphan_and_missing = pair_orphan_count + pair_evidence_missing_count
        if i08_orphaned is not None and i08_orphaned != pair_orphan_and_missing:
            mismatches.append(f"orphaned computed={pair_orphan_and_missing} i08={i08_orphaned}")
        if mismatches:
            print(
                f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-013): {pair_label}: "
                f"artifact_use reconciliation mismatch: {'; '.join(mismatches)}; reported, not resolved locally",
                file=sys.stderr,
            )
    # else: metrics.artifact_use is OPTIONAL (unlike elapsed_time/
    # provider_cost/rework, which verbatim_metric() hard-requires above) --
    # every T-E40-F10-008 fixture predates this task and carries no
    # metrics.artifact_use at all, so its mere absence is silent, matching
    # those fixtures' "no stderr on a clean run" contract (TC-084). Only a
    # genuine VALUE MISMATCH (both sides present, disagreeing) is loud,
    # exactly like design decision 2's rework-mismatch precedent.

    artifact_produced_count += pair_produced_count
    artifact_consumed_count += pair_consumed_count
    artifact_orphan_count += pair_orphan_count
    artifact_evidence_missing_count += pair_evidence_missing_count
    artifact_reused_count += pair_reused_count

    # -------------------------------------------------------------------
    # /artifact_use/replayed_interaction_proxy -- design decision 10. Read
    # only when I-08 actually carries the sub-object; absent for every
    # pair today (checked: evaluate-lifecycle.sh does not yet populate
    # it), so the block renders `unavailable` unless a fixture (or a
    # future evaluate-lifecycle.sh) supplies it.
    # -------------------------------------------------------------------
    proxy_pairs_total += 1
    proxy_source = artifact_use_metric.get("replayed_interaction_proxies") if isinstance(artifact_use_metric, dict) else None
    if isinstance(proxy_source, dict):
        proxy_pairs_with_data += 1
        for key in proxy_totals:
            value = proxy_source.get(key)
            if not isinstance(value, int) or isinstance(value, bool):
                fail(
                    f"{pair_label}: metrics.artifact_use.replayed_interaction_proxies.{key} "
                    f"is not an integer, got {value!r}"
                )
            proxy_totals[key] += value

    # -------------------------------------------------------------------
    # /comparisons -- design decision 13. comparison.json is OPTIONAL per
    # pair (retention_optional_artifacts); present only when
    # run-review-comparison.sh actually compared this (scenario, rep).
    # Republished verbatim and in full -- this script never re-derives
    # `accepted` or filters `divergences` (REQ-F-015, ADR-F10-04).
    # -------------------------------------------------------------------
    comparison_path = os.path.join(rep_dir, "comparison.json")
    if os.path.isfile(comparison_path):
        try:
            with open(comparison_path, encoding="utf-8") as fh:
                comparison_record = json.load(fh)
        except (OSError, json.JSONDecodeError) as exc:
            fail(f"{pair_label}: comparison.json is not valid JSON: {exc}")
        if not isinstance(comparison_record, dict):
            fail(f"{pair_label}: comparison.json is not a JSON object")
        for key in ("mode", "left_evaluation_id", "right_evaluation_id", "accepted", "comparison", "divergences"):
            if key not in comparison_record:
                fail(f"{pair_label}: comparison.json missing required field {key!r}")
        comparisons_list.append(
            {
                "scenario_id": scenario_id,
                "rep": rep_int,
                "retention_path": f"scenarios/{scenario_id}/{rep}",
                "mode": comparison_record["mode"],
                "left_evaluation_id": comparison_record["left_evaluation_id"],
                "right_evaluation_id": comparison_record["right_evaluation_id"],
                "accepted": comparison_record["accepted"],
                "comparison": comparison_record["comparison"],
                "divergences": comparison_record["divergences"],
            }
        )

scenarios_list.sort(key=lambda s: (s["scenario_id"], s["rep"]))
manifest_digest_entries.sort(key=lambda e: (e["scenario_id"], e["rep"]))
quality_by_scenario.sort(key=lambda q: (q["scenario_id"], q["rep"]))
artifact_edges.sort(key=lambda e: (e["producer_stage"], e["artifact_path"], e["consuming_stage"], e["edge_kind"]))
comparisons_list.sort(key=lambda c: (c["scenario_id"], c["rep"]))

# ---------------------------------------------------------------------------
# /review_value gate resolution (design decision 8 continued). A gate_id
# repeating across pairs is the NORMAL case for a multi-rep batch --
# `min_reps` (a required /identity field) dispatches the SAME scenario's
# SAME gates repeatedly, so every gate_id collides across reps on every
# real baseline root. This resolves each gate_id's occurrences (collected
# above, one per contributing pair) into a single row, distinguishing an
# ordinary same-scenario repetition (silent) from a genuine cross-scenario
# gate_id clash or an actual value DISAGREEMENT across reps (both loud,
# non-fatal diagnostics -- never a refusal, never a silently-picked side).
# ---------------------------------------------------------------------------
GATE_STATE_SEVERITY = {"collection_failure": 0, "findings": 1, "zero_findings": 2, "not_reached": 3}

for gate_id in sorted(gate_occurrences):
    occurrences = gate_occurrences[gate_id]
    scenario_ids = {o["scenario_id"] for o in occurrences}
    if len(scenario_ids) > 1:
        print(
            f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-012): gate_id {gate_id!r} "
            f"appears under more than one scenario ({sorted(scenario_ids)}) -- gate_id is expected "
            f"to be scoped to one scenario's workflow_policy.enabled_gates; rows merged, not "
            f"resolved locally",
            file=sys.stderr,
        )

    states = {o["state"] for o in occurrences}
    if len(states) == 1:
        resolved_state = next(iter(states))
    else:
        # Reps of the same gate genuinely disagree on state -- pick the
        # most severe (most informative-of-a-problem) state deterministically
        # rather than an arbitrary "last one wins", and report the
        # disagreement; never silently average or hide it.
        resolved_state = min(states, key=lambda s: GATE_STATE_SEVERITY[s])
        print(
            f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-012): gate_id {gate_id!r} "
            f"reports disagreeing states across reps ({sorted(states)}); rendering the most "
            f"severe ({resolved_state!r}), not resolved locally",
            file=sys.stderr,
        )

    reached_rounds = [o["round"] for o in occurrences if o["state"] != "not_reached" and isinstance(o["round"], int)]
    resolved_round = max(reached_rounds) if reached_rounds else None

    truth_flags = {o["truth_set_available"] for o in occurrences}
    precisions = {o["precision"] for o in occurrences}
    recalls = {o["recall"] for o in occurrences}
    if len(truth_flags) == 1 and len(precisions) == 1 and len(recalls) == 1:
        resolved_truth_available = next(iter(truth_flags))
        resolved_precision = next(iter(precisions))
        resolved_recall = next(iter(recalls))
    else:
        # ADR-F10-10: never assert a value the contributing reps cannot
        # jointly support -- disagreement renders unavailable, exactly like
        # a genuinely absent truth set, rather than picking one rep's side.
        resolved_truth_available = False
        resolved_precision = "unavailable"
        resolved_recall = "unavailable"
        print(
            f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-012): gate_id {gate_id!r} "
            f"reports disagreeing truth_set_available/precision/recall across reps; rendering "
            f"unavailable rather than asserting an unsupportable value, not resolved locally",
            file=sys.stderr,
        )

    gate_records[gate_id] = {
        "state": resolved_state,
        "round": resolved_round,
        "truth_set_available": resolved_truth_available,
        "precision": resolved_precision,
        "recall": resolved_recall,
    }


def build_partition(totals, categories, total_value):
    block = {cat: round(totals.get(cat, 0.0), 6) for cat in categories}
    matched = sum(totals.get(cat, 0.0) for cat in categories)
    block[UNATTRIBUTED] = round(total_value - matched, 6)
    return block


time_block = {
    "lifecycle_wall_seconds": round(total_lifecycle_wall_seconds, 6),
    "stage_category": build_partition(stage_time_totals, STAGE_CATEGORIES, total_lifecycle_wall_seconds),
    "interval_category": build_partition(interval_time_totals, INTERVAL_CATEGORIES, total_lifecycle_wall_seconds),
    "share_partition": build_partition(share_time_totals, SHARE_CELLS, total_lifecycle_wall_seconds),
}

cost_block = {
    "stage_category": build_partition(stage_cost_totals, STAGE_CATEGORIES, total_provider_cost_usd),
    "interval_category": build_partition(interval_cost_totals, INTERVAL_CATEGORIES, total_provider_cost_usd),
    "share_partition": build_partition(share_cost_totals, SHARE_CELLS, total_provider_cost_usd),
    "ceiling_consumption": {
        "observed_cost_usd": round(total_observed_ceiling_cost, 6),
        "max_cost_usd": ceilings["max_cost_usd"],
    },
}

# ---------------------------------------------------------------------------
# /quality -- design decision 7. first_pass_yield: among gates that were
# reached (state != not_reached), the fraction whose highest observed round
# is exactly 1. No reached gates -> "unavailable", never a 0/0 division or a
# bare 0 (ADR-F10-10).
# ---------------------------------------------------------------------------
reached_gates = [g for g in gate_records.values() if g["state"] != "not_reached"]
if reached_gates:
    first_pass_count = sum(1 for g in reached_gates if g["round"] == 1)
    first_pass_yield = round(first_pass_count / len(reached_gates), 6)
else:
    first_pass_yield = "unavailable"

quality_block = {
    "by_scenario": quality_by_scenario,
    "first_pass_yield": first_pass_yield,
}

if gate_records:
    print(
        "aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-012): "
        "review_value.gates[].elapsed_seconds/provider_cost_usd/resolution_cost_usd have no "
        "upstream join key in I-07 review_gates[] or I-08 metrics -- rendered as unavailable, "
        "never zero (ADR-F10-10)",
        file=sys.stderr,
    )

# ---------------------------------------------------------------------------
# /review_value -- design decision 8. Gate identity/state verbatim from
# I-07 review_gates[]; the seven finding measures, broken out by severity
# and defect class, verbatim from I-08 metrics.quality.review_measures[].
# ---------------------------------------------------------------------------
review_value_gates = []
for gate_id in sorted(gate_records):
    record = gate_records[gate_id]
    measures_for_gate = review_measures_by_gate.get(gate_id, [])
    counts = zero_measures()
    counts_by_severity = {}
    counts_by_defect_class = {}
    for entry in measures_for_gate:
        for key in MEASURE_KEYS:
            value = entry["measures"][key]
            counts[key] += value
            counts_by_severity.setdefault(entry["severity"], zero_measures())[key] += value
            counts_by_defect_class.setdefault(entry["defect_class"], zero_measures())[key] += value
    review_value_gates.append(
        {
            "gate_id": gate_id,
            "state": record["state"],
            "counts": counts,
            "counts_by_severity": counts_by_severity,
            "counts_by_defect_class": counts_by_defect_class,
            "elapsed_seconds": "unavailable",
            "provider_cost_usd": "unavailable",
            "resolution_cost_usd": "unavailable",
            "truth_set_available": record["truth_set_available"],
            "precision": record["precision"],
            "recall": record["recall"],
        }
    )

review_value_block = {"gates": review_value_gates}

# ---------------------------------------------------------------------------
# /artifact_use -- design decision 9/10.
# ---------------------------------------------------------------------------
if proxy_pairs_with_data == 0:
    replayed_interaction_proxy = {
        "label": REPLAYED_PROXY_LABEL,
        "request_count": "unavailable",
        "response_count": "unavailable",
        "payload_size_bytes": "unavailable",
        "revision_count": "unavailable",
        "unresolved_gate_count": "unavailable",
    }
    # No diagnostic when zero pairs carry replayed_interaction_proxies: this
    # sub-object is OPTIONAL (design decision 10 -- evaluate-lifecycle.sh
    # does not populate it today), so its universal absence is silent,
    # matching every T-E40-F10-008 fixture's "no stderr on a clean run"
    # contract (TC-084). Only PARTIAL coverage (some pairs have it, some do
    # not) is loud, below.
else:
    if proxy_pairs_with_data != proxy_pairs_total:
        print(
            f"aggregate-lifecycle: UPSTREAM CONTRACT DEFECT (REQ-F-013): only "
            f"{proxy_pairs_with_data}/{proxy_pairs_total} retained pairs carry "
            f"metrics.artifact_use.replayed_interaction_proxies -- summing the pairs that have "
            f"it, not rendered as a complete total",
            file=sys.stderr,
        )
    replayed_interaction_proxy = {
        "label": REPLAYED_PROXY_LABEL,
        "request_count": proxy_totals["authorized_request_count"],
        "response_count": proxy_totals["authorized_response_count"],
        "payload_size_bytes": proxy_totals["request_bytes_total"] + proxy_totals["response_bytes_total"],
        "revision_count": proxy_totals["revision_or_replacement_count"],
        "unresolved_gate_count": proxy_totals["unresolved_gate_count"],
    }

artifact_use_block = {
    "produced_count": artifact_produced_count,
    "consumed_count": artifact_consumed_count,
    "reused_count": artifact_reused_count,
    "orphan_count": artifact_orphan_count,
    "edges": artifact_edges,
    "replayed_interaction_proxy": replayed_interaction_proxy,
}


# ---------------------------------------------------------------------------
# /noise_bands -- design decision 11/12. Reuses aggregate-runs.sh's Class C
# min/median/max/spread_abs/accept-interval formula verbatim (ADR-F10-12) --
# NOT its own private `n < 2` threshold; F10's threshold is the batch-
# declared /identity/min_reps (AC-T1). Every contributing value is the SAME
# already-verbatim I-08 metric value used to build /time, /cost, and
# /quality elsewhere in this script -- no metric is read a second time or
# with different semantics.
# ---------------------------------------------------------------------------
def compute_noise_band(band_scenario_id, metric_name, values, declared_min_reps):
    n = len(values)
    if n == 0:
        # ADR-F10-10: absent measures render `unavailable`, never zero or a
        # fabricated band. Only reachable for `confirmed_findings` --
        # `elapsed_time`/`provider_cost` are hard-required by
        # verbatim_metric() above and can never reach n==0 here.
        return {
            "scenario_id": band_scenario_id,
            "metric": metric_name,
            "min": "unavailable",
            "median": "unavailable",
            "max": "unavailable",
            "spread_abs": "unavailable",
            "acceptance_interval": {"lower_bound": "unavailable", "upper_bound": "unavailable"},
            "derivation_rule": NOISE_BAND_DERIVATION_RULE,
            "rep_count": 0,
            "insufficient_reps": True,
        }

    mn = min(values)
    mx = max(values)
    median = statistics.median(values)
    spread_abs = mx - mn
    if spread_abs > 0:
        r_eff = spread_abs
    elif median != 0:
        r_eff = 0.10 * abs(median)
    else:
        # Identically the same value across every contributing rep --
        # the documented exact rule (aggregate-runs.sh precedent), not a
        # coincidence of the 0.10*median formula.
        r_eff = 0
    lower_bound = max(0, mn - r_eff)
    upper_bound = mx + r_eff

    return {
        "scenario_id": band_scenario_id,
        "metric": metric_name,
        "min": round(mn, 6),
        "median": round(median, 6),
        "max": round(mx, 6),
        "spread_abs": round(spread_abs, 6),
        "acceptance_interval": {
            "lower_bound": round(lower_bound, 6),
            "upper_bound": round(upper_bound, 6),
        },
        "derivation_rule": NOISE_BAND_DERIVATION_RULE,
        "rep_count": n,
        # AC-T1: compared against the declared BATCH minimum, never a
        # private threshold -- ADR-F10-12 fixes the mechanism, the batch
        # policy fixes the number.
        "insufficient_reps": n < declared_min_reps,
    }


noise_bands_list = [
    compute_noise_band(sid, metric_name, noise_metric_values.get((sid, metric_name), []), min_reps)
    for sid in sorted({s["scenario_id"] for s in scenarios_list})
    for metric_name in NOISE_BAND_METRICS
]

# ---------------------------------------------------------------------------
# /invalid -- design decision 13. Two sources, both loud never silent
# (REQ-F-007's "every invalidity_reasons entry"):
#   (a) retention_root/invalid/index.jsonl -- run-lifecycle-batch.sh's own
#       dispatch-failure ledger for pairs that never reached a retained
#       evaluation.jsonl at all. Mirrored verbatim (spec.md blocks table:
#       "mirroring invalid/index.jsonl").
#   (b) retained-but-ineligible pairs -- every scenarios_list[] entry whose
#       I-08 eligibility.invalidity_reasons is non-empty, reusing the SAME
#       eligibility object already read verbatim for /scenarios (AC-T2: no
#       recomputation, override, or softening).
# ---------------------------------------------------------------------------
invalid_list = []

invalid_index_path = os.path.join(retention_root, "invalid", "index.jsonl")
if os.path.isfile(invalid_index_path):
    with open(invalid_index_path, encoding="utf-8") as fh:
        for line_no, raw_line in enumerate(fh, start=1):
            stripped_line = raw_line.strip()
            if not stripped_line:
                continue
            try:
                invalid_row = json.loads(stripped_line)
            except json.JSONDecodeError as exc:
                fail(f"invalid/index.jsonl:{line_no}: unparseable JSON: {exc}")
            if not isinstance(invalid_row, dict):
                fail(f"invalid/index.jsonl:{line_no}: row is not a JSON object")
            for key in ("scenario_id", "rep", "retention_path", "invalidity_reasons"):
                if key not in invalid_row:
                    fail(f"invalid/index.jsonl:{line_no}: missing required field {key!r}")
            invalid_list.append(
                {
                    "scenario_id": invalid_row["scenario_id"],
                    "rep": invalid_row["rep"],
                    "retention_path": invalid_row["retention_path"],
                    "source": "batch_dispatch_failure",
                    "invalidity_reasons": invalid_row["invalidity_reasons"],
                }
            )

for scenario_entry in scenarios_list:
    entry_reasons = scenario_entry["eligibility"].get("invalidity_reasons")
    if entry_reasons:
        invalid_list.append(
            {
                "scenario_id": scenario_entry["scenario_id"],
                "rep": scenario_entry["rep"],
                "retention_path": scenario_entry["retention_path"],
                "source": "i08_eligibility",
                "eligibility": scenario_entry["eligibility"],
            }
        )

invalid_list.sort(key=lambda r: (r["scenario_id"], r["rep"], r["source"]))

# /identity.retention_root_digest -- digest of digests (design decision 5).
retention_root_digest = sha256_bytes(
    json.dumps(
        {"batch_json_sha256": sha256_bytes(batch_bytes), "pairs": manifest_digest_entries},
        sort_keys=True,
        separators=(",", ":"),
    ).encode("utf-8")
)

identity = {
    "schema_version": SCHEMA_VERSION,
    "batch_id": batch_id,
    "phase": PHASE_LABEL,
    "retention_root_digest": retention_root_digest,
    "batch_policy_digest": batch_policy_digest,
    "ceilings": ceilings,
    "acknowledgement_ref": acknowledgement_ref,
    "min_reps": min_reps,
}

aggregate = {
    "identity": identity,
    "scenarios": scenarios_list,
    "time": time_block,
    "cost": cost_block,
    "quality": quality_block,
    "review_value": review_value_block,
    "artifact_use": artifact_use_block,
    "noise_bands": noise_bands_list,
    "comparisons": comparisons_list,
    "invalid": invalid_list,
}

print(json.dumps(aggregate, indent=2, sort_keys=True))
PYEOF
