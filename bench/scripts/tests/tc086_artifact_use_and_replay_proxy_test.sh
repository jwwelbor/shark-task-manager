#!/usr/bin/env bash
# TC-086 / T-E40-F10-009 (aggregator half): artifact-use and
# replayed-interaction-proxy reporting (spec.md REQ-F-013, ADR-F10-04,
# ADR-F10-10; test-plan.md TC-086 full body; AC-009, AC-T3).
#
# Exercises the REAL aggregate-lifecycle.sh binary over two hand-built
# retention roots. Root A: one artifact produced-and-consumed downstream by
# TWO distinct stages (proving `reused`), one true orphan (`consumers: []`),
# one artifact with consumption evidence not collected at all (an absent
# `consumers` key -- a third, distinct state per bench/README.md), and a
# populated `metrics.artifact_use.replayed_interaction_proxies` sub-object
# (I-06's own field names) that this script must map onto F10's schema
# names and label. Root B: the same artifact shape but with NO
# `replayed_interaction_proxies` sub-object, proving the absent path
# renders every proxy field `unavailable`, never `0`.
#
# This is the aggregator half only: report-lifecycle.sh --view headline
# rendering and the forbidden_effort_language static scan across the
# enumerated F10 script/template set are T-E40-F10-011's and
# T-E40-F10-014's slices respectively.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
AGGREGATOR="$SCRIPTS_DIR/aggregate-lifecycle.sh"

fail() {
	echo "TC-086 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATOR" ]] || fail "bench/scripts/aggregate-lifecycle.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# ===========================================================================
# build_root <dest_root> <with_proxies:0|1>
# ===========================================================================
build_root() {
	local dest_root="$1" with_proxies="$2"
	python3 - "$dest_root" "$with_proxies" <<'PYEOF'
import hashlib
import json
import os
import sys

dest_root, with_proxies = sys.argv[1], sys.argv[2] == "1"

SCENARIO_ID = "scenario-tc086"
REP = "1"
pair_dir = os.path.join(dest_root, "scenarios", SCENARIO_ID, REP)
os.makedirs(pair_dir, exist_ok=True)

with open(os.path.join(pair_dir, "package.yaml"), "w", encoding="utf-8") as f:
    f.write(
        'schema_version: "1.0"\n'
        f'scenario_id: "{SCENARIO_ID}"\n'
        'scenario_version: "1"\n'
        'entity_family: "family-tc086"\n'
    )


def base_interval():
    return [{"category": "provider_active", "start": 0, "end": 1}]


planning_stage = {
    "dispatch_ordinal": 1, "stage": "planning", "category": "planning",
    "snapshot_digest": "a" * 64, "prompt_digest": "b" * 64,
    "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
    "usage": {"provider": "fixture", "model": "fixture-model"},
    "cost_usd": 1.0, "elapsed_seconds": 1.0, "errors": [], "rework": False,
    "intervals": base_interval(),
    "candidate": {}, "access_events": [], "evidence_refs": {},
    "artifacts": [
        {
            "artifact_type": "document", "path": "artifacts/design.md",
            "digest": "sha256:" + "1" * 64, "size_bytes": 100, "producer_stage": "planning",
            "consumers": [
                {"consuming_stage": "code", "edge_kind": "read", "observed_at": "2026-08-19T00:00:00Z"},
                {"consuming_stage": "review", "edge_kind": "referenced", "observed_at": "2026-08-19T00:00:01Z"},
            ],
        },
        {
            "artifact_type": "document", "path": "artifacts/orphan.md",
            "digest": "sha256:" + "2" * 64, "size_bytes": 50, "producer_stage": "planning",
            "consumers": [],
        },
        {
            "artifact_type": "other", "path": "artifacts/no-evidence.md",
            "digest": "sha256:" + "3" * 64, "size_bytes": 10, "producer_stage": "planning",
            # consumers key intentionally absent: consumption evidence not
            # collected -- a third, distinct state (bench/README.md).
        },
    ],
}
code_stage = {
    "dispatch_ordinal": 2, "stage": "code", "category": "code",
    "snapshot_digest": "c" * 64, "prompt_digest": "d" * 64,
    "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
    "usage": {"provider": "fixture", "model": "fixture-model"},
    "cost_usd": 1.0, "elapsed_seconds": 1.0, "errors": [], "rework": False,
    "intervals": base_interval(),
    "candidate": {}, "access_events": [], "evidence_refs": {},
    "artifacts": [
        {
            "artifact_type": "code_diff", "path": "artifacts/patch.diff",
            "digest": "sha256:" + "4" * 64, "size_bytes": 200, "producer_stage": "code",
            "consumers": [
                {"consuming_stage": "review", "edge_kind": "read", "observed_at": "2026-08-19T00:00:02Z"},
            ],
        },
    ],
}

lc = {
    "identity": {
        "schema_version": "1.0", "run_id": "run-tc086", "scenario_id": SCENARIO_ID,
        "scenario_version": "1", "fixture_id": "fixture-tc086", "fixture_digest": "e" * 64,
        "adapter_id": "fixture-adapter", "adapter_version": "1",
        "shark_binary_digest": "f" * 64, "shark_content_digest": "0" * 64, "roots": {},
    },
    "entity_graph": {}, "dispatches": [], "stages": [planning_stage, code_stage],
    "workflow_policy": {}, "review_gates": [], "questions": [],
    "limits": {
        "max_cost_usd": 5.0, "max_wall_clock_seconds": 60, "max_generated_tasks": 5,
        "observed_cost_usd": 2.0, "observed_wall_clock_seconds": 2.0,
        "observed_generated_tasks": 1, "first_exceeded": None,
    },
    "outcome": {"terminal": "complete", "reason": "tc086 fixture", "partial_evidence": False, "publication_eligible": True},
}
with open(os.path.join(pair_dir, "lifecycle.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(lc, sort_keys=True, separators=(",", ":")) + "\n")

# I-08's own conflated metrics.artifact_use: produced=4 (all artifacts
# regardless of consumer state), consumed=2 (design.md + patch.diff),
# orphaned = produced - consumed = 2 (conflates orphan.md AND
# no-evidence.md -- the F09 gap design decision 9 documents).
artifact_use_metric = {
    "produced": {"value": 4, "available": True},
    "consumed": {"value": 2, "available": True},
    "orphaned": {"value": 2, "available": True},
}
if with_proxies:
    artifact_use_metric["replayed_interaction_proxies"] = {
        "measurement_kind": "replayed_interaction_proxy",
        "authorized_request_count": 7,
        "authorized_response_count": 7,
        "request_bytes_total": 1200,
        "response_bytes_total": 3400,
        "revision_or_replacement_count": 2,
        "replay_wait_ns": 500000,
        "replay_wait_category": "replay_or_human_gate_wait",
        "unresolved_gate_count": 0,
    }

ev = {
    "schema_version": "1.0", "evaluation_id": "eval-tc086", "identity": {}, "source_artifacts": {},
    "structural": {}, "judge": {}, "execution_oracle": {},
    "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
    "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
    "metrics": {
        "elapsed_time": {"value": 2.0, "available": True},
        "provider_cost": {"value": 2.0, "available": True},
        "rework": {"value": 0, "available": True},
        "artifact_use": artifact_use_metric,
    },
}
with open(os.path.join(pair_dir, "evaluation.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(ev, sort_keys=True, separators=(",", ":")) + "\n")


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


manifest = {
    "scenario_id": SCENARIO_ID, "rep": int(REP),
    "artifacts": {
        "package.yaml": {"source_path": os.path.join(pair_dir, "package.yaml"), "sha256": sha(os.path.join(pair_dir, "package.yaml"))},
        "lifecycle.jsonl": {"source_path": os.path.join(pair_dir, "lifecycle.jsonl"), "sha256": sha(os.path.join(pair_dir, "lifecycle.jsonl"))},
        "evaluation.jsonl": {"source_path": os.path.join(pair_dir, "evaluation.jsonl"), "sha256": sha(os.path.join(pair_dir, "evaluation.jsonl"))},
    },
}
with open(os.path.join(pair_dir, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")

batch = {
    "phase": "lifecycle_v2", "batch_id": "batch-tc086", "mode": "pilot", "min_reps": 1,
    "batch_policy_digest": "1" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
with open(os.path.join(dest_root, "batch.json"), "w", encoding="utf-8") as f:
    json.dump(batch, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")
PYEOF
}

# ===========================================================================
# (a) Populated path: consumed/orphan/reused/evidence-missing distinguished,
# typed edges present, replayed_interaction_proxy carries mapped numeric
# values under its label.
# ===========================================================================
ROOT_A="$WORKDIR/root-a"
mkdir -p "$ROOT_A"
build_root "$ROOT_A" "1"

OUT_A="$("$AGGREGATOR" --retention-root "$ROOT_A" 2>"$WORKDIR/a.err")" || fail "(a) populated path: aggregator exited non-zero; stderr: $(cat "$WORKDIR/a.err")"
[[ ! -s "$WORKDIR/a.err" ]] || fail "(a) populated path: expected no stderr diagnostics, got: $(cat "$WORKDIR/a.err")"

python3 - "$OUT_A" <<'PYEOF'
import json
import sys

agg = json.loads(sys.argv[1])


def fail(msg):
    print(f"TC-086 FAIL (python check, root a): {msg}", file=sys.stderr)
    raise SystemExit(1)


au = agg["artifact_use"]

# produced=4 (design.md, orphan.md, no-evidence.md, patch.diff)
if au["produced_count"] != 4:
    fail(f"produced_count = {au['produced_count']}, expected 4")
# consumed=2 (design.md, patch.diff -- non-empty consumers)
if au["consumed_count"] != 2:
    fail(f"consumed_count = {au['consumed_count']}, expected 2")
# orphan=1 (orphan.md ONLY -- the true orphan state, consumers: [])
if au["orphan_count"] != 1:
    fail(f"orphan_count = {au['orphan_count']}, expected 1 (true orphan only, distinct from evidence-missing)")
# reused=1 (design.md consumed by two DISTINCT consuming stages)
if au["reused_count"] != 1:
    fail(f"reused_count = {au['reused_count']}, expected 1")

edges = au["edges"]
if len(edges) != 3:
    fail(f"edges has {len(edges)} entries, expected 3 (two for design.md, one for patch.diff)")
design_edges = [e for e in edges if e["artifact_path"] == "artifacts/design.md"]
if {e["consuming_stage"] for e in design_edges} != {"code", "review"}:
    fail(f"design.md edges consuming_stage set = {[e['consuming_stage'] for e in design_edges]}, expected {{'code','review'}}")
if {e["edge_kind"] for e in design_edges} != {"read", "referenced"}:
    fail(f"design.md edges edge_kind set = {[e['edge_kind'] for e in design_edges]}, expected {{'read','referenced'}}")
patch_edges = [e for e in edges if e["artifact_path"] == "artifacts/patch.diff"]
if len(patch_edges) != 1 or patch_edges[0]["consuming_stage"] != "review" or patch_edges[0]["edge_kind"] != "read":
    fail(f"patch.diff edges = {patch_edges}, expected exactly one review/read edge")
for e in edges:
    if e["artifact_path"] == "artifacts/orphan.md" or e["artifact_path"] == "artifacts/no-evidence.md":
        fail(f"orphan.md/no-evidence.md must not appear in edges, found {e}")

# --- AC-T3: every replay-proxy field carries its replayed-proxy label at
# the data level, with the mapped numeric values.
proxy = au["replayed_interaction_proxy"]
if "replayed_interaction_proxy" not in proxy["label"] and "replayed" not in proxy["label"].lower():
    fail(f"replayed_interaction_proxy.label does not identify itself as a replayed proxy: {proxy['label']!r}")
FORBIDDEN = ["human minute", "human hour", "human effort", "effort saved", "time equivalent",
             "time saved", "person-hour", "fte", "man hour", "man-hour", "person hour",
             "developer hour", "engineer hour", "manual hour", "hours saved"]
label_lower = proxy["label"].lower()
for term in FORBIDDEN:
    if term in label_lower:
        fail(f"replayed_interaction_proxy.label contains forbidden effort-language term {term!r}: {proxy['label']!r}")

if proxy["request_count"] != 7:
    fail(f"request_count = {proxy['request_count']}, expected 7 (authorized_request_count verbatim)")
if proxy["response_count"] != 7:
    fail(f"response_count = {proxy['response_count']}, expected 7 (authorized_response_count verbatim)")
if proxy["payload_size_bytes"] != 4600:
    fail(f"payload_size_bytes = {proxy['payload_size_bytes']}, expected 4600 (1200 request + 3400 response bytes)")
if proxy["revision_count"] != 2:
    fail(f"revision_count = {proxy['revision_count']}, expected 2 (revision_or_replacement_count verbatim)")
if proxy["unresolved_gate_count"] != 0:
    fail(f"unresolved_gate_count = {proxy['unresolved_gate_count']}, expected 0")

print("TC-086(a): consumed/orphan/reused/evidence-missing distinguished with typed edges; replayed_interaction_proxy carries mapped numeric values under its label")
PYEOF

# ===========================================================================
# (b) Absent-proxy path: metrics.artifact_use IS present (produced/consumed/
# orphaned), but replayed_interaction_proxies is NOT -- every proxy field
# must render "unavailable", never 0 (ADR-F10-10), while the label field
# stays present (AC-T3's shape requirement holds even when unpopulated).
# ===========================================================================
ROOT_B="$WORKDIR/root-b"
mkdir -p "$ROOT_B"
build_root "$ROOT_B" "0"

OUT_B="$("$AGGREGATOR" --retention-root "$ROOT_B" 2>"$WORKDIR/b.err")" || fail "(b) absent-proxy path: aggregator exited non-zero; stderr: $(cat "$WORKDIR/b.err")"
[[ ! -s "$WORKDIR/b.err" ]] || fail "(b) absent-proxy path: expected no stderr diagnostics (universal absence is silent, not a mismatch), got: $(cat "$WORKDIR/b.err")"

python3 - "$OUT_B" <<'PYEOF'
import json
import sys

agg = json.loads(sys.argv[1])


def fail(msg):
    print(f"TC-086 FAIL (python check, root b): {msg}", file=sys.stderr)
    raise SystemExit(1)


proxy = agg["artifact_use"]["replayed_interaction_proxy"]
if "label" not in proxy or not proxy["label"]:
    fail("replayed_interaction_proxy.label must be present even when the proxy sub-object is absent (AC-T3 shape)")
for field in ("request_count", "response_count", "payload_size_bytes", "revision_count", "unresolved_gate_count"):
    if proxy[field] != "unavailable":
        fail(f"{field} = {proxy[field]!r}, expected the literal string 'unavailable' (never 0) when the upstream proxy sub-object is absent")

# Produced/consumed/orphan/reused counts are unaffected by the absent proxy
# block -- same artifact shape as root (a).
au = agg["artifact_use"]
if au["produced_count"] != 4 or au["consumed_count"] != 2 or au["orphan_count"] != 1 or au["reused_count"] != 1:
    fail(f"artifact_use counts changed unexpectedly without proxy data: {au}")

print("TC-086(b): absent replayed_interaction_proxies renders every proxy field 'unavailable', never 0, while the block shape (label) stays present")
PYEOF

echo "TC-086 (aggregator half): PASS"
