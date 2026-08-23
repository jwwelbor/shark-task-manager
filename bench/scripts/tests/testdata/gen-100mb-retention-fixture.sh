#!/usr/bin/env bash
# gen-100mb-retention-fixture.sh --seed <n> --out <dir> [--total-bytes <n>]
#
# T-E40-F10-013 (spec.md REQ-NF-004; test-plan.md TC-090; AC-T3) committed
# GENERATOR for a retention-shaped ~100 MB fixture -- not a committed binary
# blob. No committed 100 MB retention fixture exists anywhere in the repo
# today (confirmed by search at task-decomposition time) and this generator
# is run once at test-fixture setup time, not regenerated per test run
# (task Goal).
#
# Deterministic: invoking this script twice with the same --seed and the
# same --total-bytes produces byte-identical content at every relative
# path. Every byte comes from hashlib.shake_256(seed + ":" + relpath),
# never os.urandom/wall-clock/PID -- shake_256's arbitrary-length digest()
# is a fast (C-implemented), pure function of its key, so re-running this
# generator is a real reproducibility proof, not merely "looks similar".
#
# Layout (matches run-lifecycle-batch.sh's retain_pair() output shape, the
# real production retention layout this fixture stands in for):
#   <out>/batch.json
#   <out>/scenarios/<scenario_id>/<rep>/package.yaml
#   <out>/scenarios/<scenario_id>/<rep>/lifecycle.jsonl
#   <out>/scenarios/<scenario_id>/<rep>/evaluation.jsonl
#   <out>/scenarios/<scenario_id>/<rep>/manifest.json
#   <out>/scenarios/<scenario_id>/<rep>/evidence/blob-*.bin      (padding)
#   <out>/scenarios/<scenario_id>/<rep>/transcripts/blob-*.bin   (padding)
#
# The four small files (package.yaml/lifecycle.jsonl/evaluation.jsonl/
# manifest.json) are schema-valid retained-pair content, shaped identically
# to test-plan.md TC-084's own proven-good fixture builder -- the same
# shapes aggregate-lifecycle.sh/report-lifecycle.sh already pass over
# successfully in TC-084. The bulk padding lives under evidence/ and
# transcripts/, which aggregate-lifecycle.sh's own header contract (design
# decision 5) documents it never opens -- this is what makes TC-090's
# peak-RSS bound a real streaming proof rather than a fixture that is
# coincidentally small.
#
# --out must not already exist with content -- callers pass a fresh scratch
# directory (per parent-loop guidance: generate into /tmp, not the repo,
# and clean up afterward); this script never overwrites unrelated content.
set -euo pipefail

usage() {
	echo "usage: gen-100mb-retention-fixture.sh --seed <n> --out <dir> [--total-bytes <n>]" >&2
	exit 2
}

seed=""
out=""
total_bytes=104857600 # 100 MiB default (REQ-NF-004)

while [[ $# -gt 0 ]]; do
	case "$1" in
	--seed)
		[[ $# -ge 2 ]] || usage
		seed="$2"
		shift 2
		;;
	--out)
		[[ $# -ge 2 ]] || usage
		out="$2"
		shift 2
		;;
	--total-bytes)
		[[ $# -ge 2 ]] || usage
		total_bytes="$2"
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

[[ -n "$seed" ]] || usage
[[ -n "$out" ]] || usage
command -v python3 >/dev/null 2>&1 || {
	echo "gen-100mb-retention-fixture: python3 not found on PATH" >&2
	exit 2
}

mkdir -p "$out"
if [[ -n "$(ls -A "$out" 2>/dev/null)" ]]; then
	echo "gen-100mb-retention-fixture: --out $out is not empty; pass a fresh directory" >&2
	exit 2
fi

python3 - "$seed" "$out" "$total_bytes" <<'PYEOF'
import hashlib
import json
import os
import sys

seed, out, total_bytes = sys.argv[1], sys.argv[2], int(sys.argv[3])

SCENARIO_ID = "scale-fixture-scenario"
REP = "1"
pair_dir = os.path.join(out, "scenarios", SCENARIO_ID, REP)
evidence_dir = os.path.join(pair_dir, "evidence")
transcripts_dir = os.path.join(pair_dir, "transcripts")
os.makedirs(evidence_dir, exist_ok=True)
os.makedirs(transcripts_dir, exist_ok=True)


def det_bytes(relpath, n):
    key = f"{seed}:{relpath}".encode("utf-8")
    return hashlib.shake_256(key).digest(n)


# ---------------------------------------------------------------------------
# Small, schema-valid retained pair -- the aggregator/reporter under test
# read ONLY these four small files; everything else below is inert padding
# they never open.
# ---------------------------------------------------------------------------
with open(os.path.join(pair_dir, "package.yaml"), "w", encoding="utf-8") as f:
    f.write(
        'schema_version: "1.0"\n'
        f'scenario_id: "{SCENARIO_ID}"\n'
        'scenario_version: "1"\n'
        'entity_family: "family-scale-fixture"\n'
    )

INTERVALS = [
    ("provider_active", 10),
    ("tool_and_test", 5),
    ("queue_or_claim_wait", 3),
    ("replay_or_human_gate_wait", 2),
    ("retry_or_backoff", 1),
    ("unclassified", 4),
]
STAGE_SECONDS = sum(d for _, d in INTERVALS)
STAGE_COST = 10.0
STAGE_CATEGORIES = ["discovery", "specification", "planning", "code", "review", "qa", "uat", "shipping"]


def make_intervals():
    result = []
    cursor = 0
    for category, duration in INTERVALS:
        result.append({"category": category, "start": cursor, "end": cursor + duration})
        cursor += duration
    return result


def make_stage(ordinal, category, rework):
    return {
        "dispatch_ordinal": ordinal,
        "stage": category,
        "category": category,
        "snapshot_digest": "a" * 64,
        "prompt_digest": "b" * 64,
        "input_lineage": [],
        "replay_lineage": [],
        "output_paths": [],
        "output_digests": [],
        "usage": {"provider": "fixture", "model": "fixture-model"},
        "cost_usd": STAGE_COST,
        "elapsed_seconds": float(STAGE_SECONDS),
        "errors": [],
        "intervals": make_intervals(),
        "rework": rework,
        "candidate": {},
        "artifacts": [],
        "access_events": [],
        "evidence_refs": {},
    }


stages = [make_stage(i + 1, cat, rework=False) for i, cat in enumerate(STAGE_CATEGORIES)]
total_elapsed = float(STAGE_SECONDS * len(stages))
total_cost = STAGE_COST * len(stages)

lc = {
    "identity": {
        "schema_version": "1.0",
        "run_id": "run-scale-fixture",
        "scenario_id": SCENARIO_ID,
        "scenario_version": "1",
        "fixture_id": "fixture-scale",
        "fixture_digest": "c" * 64,
        "adapter_id": "fixture-adapter",
        "adapter_version": "1",
        "shark_binary_digest": "d" * 64,
        "shark_content_digest": "e" * 64,
        "roots": {},
    },
    "entity_graph": {},
    "dispatches": [],
    "stages": stages,
    "workflow_policy": {},
    "review_gates": [],
    "questions": [],
    "limits": {
        "max_cost_usd": 100.0,
        "max_wall_clock_seconds": 3600,
        "max_generated_tasks": 20,
        "observed_cost_usd": total_cost,
        "observed_wall_clock_seconds": total_elapsed,
        "observed_generated_tasks": 1,
        "first_exceeded": None,
    },
    "outcome": {"terminal": "complete", "reason": "scale fixture", "partial_evidence": False, "publication_eligible": True},
}
with open(os.path.join(pair_dir, "lifecycle.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(lc, sort_keys=True, separators=(",", ":")) + "\n")

ev = {
    "schema_version": "1.0",
    "evaluation_id": "eval-scale-fixture",
    "identity": {},
    "source_artifacts": {},
    "structural": {},
    "judge": {},
    "execution_oracle": {},
    "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
    "candidate_snapshots": [],
    "workflow_policy": {},
    "comparison": {},
    "metrics": {
        "elapsed_time": {"value": total_elapsed, "available": True, "detail": "sum of retained stage elapsed_seconds"},
        "provider_cost": {"value": total_cost, "available": True, "detail": "sum of retained stage cost_usd"},
        "rework": {"value": 0, "available": True, "detail": "count of retained stages marked rework"},
    },
}
with open(os.path.join(pair_dir, "evaluation.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(ev, sort_keys=True, separators=(",", ":")) + "\n")


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


manifest = {
    "scenario_id": SCENARIO_ID,
    "rep": int(REP),
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
    "phase": "lifecycle_v2",
    "batch_id": "batch-scale-fixture",
    "mode": "pilot",
    "min_reps": 1,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
with open(os.path.join(out, "batch.json"), "w", encoding="utf-8") as f:
    json.dump(batch, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")

# ---------------------------------------------------------------------------
# Bulk padding: aggregate-lifecycle.sh never opens evidence/ or transcripts/
# content (its own header's design decision 5). This is where the ~100 MB
# lives. The plan keeps every single file under 25 MB so TC-090's RSS bound
# stays the flat 25 MB case (never the largest-file+5MB fallback) at the
# default --total-bytes, while still including one file clearly larger than
# its siblings so the Edge Case ("a very large single evidence file...
# must not cause a memory spike proportional to that one file's size") is
# genuinely exercised.
# ---------------------------------------------------------------------------
FILE_PLAN = [
    ("evidence", "blob-000.bin", 20 * 1024 * 1024),
    ("evidence", "blob-001.bin", 16 * 1024 * 1024),
    ("evidence", "blob-002.bin", 16 * 1024 * 1024),
    ("transcripts", "blob-000.bin", 16 * 1024 * 1024),
    ("transcripts", "blob-001.bin", 16 * 1024 * 1024),
    ("transcripts", "blob-002.bin", 16 * 1024 * 1024),
]
planned_total = sum(size for _, _, size in FILE_PLAN)
if total_bytes != planned_total:
    # A caller overriding --total-bytes (e.g. a fast determinism self-test)
    # still gets deterministic, proportionally-scaled padding rather than a
    # plan that silently ignores the flag.
    scale = total_bytes / planned_total
    FILE_PLAN = [(d, n, max(1, int(size * scale))) for d, n, size in FILE_PLAN]

dir_map = {"evidence": evidence_dir, "transcripts": transcripts_dir}
largest_file_bytes = 0
written_total = 0
for subdir, name, size in FILE_PLAN:
    rel = f"scenarios/{SCENARIO_ID}/{REP}/{subdir}/{name}"
    data = det_bytes(rel, size)
    dest = os.path.join(dir_map[subdir], name)
    with open(dest, "wb") as f:
        f.write(data)
    written_total += len(data)
    largest_file_bytes = max(largest_file_bytes, len(data))

print(json.dumps({
    "pair_dir": pair_dir,
    "padding_bytes_written": written_total,
    "largest_padding_file_bytes": largest_file_bytes,
}))
PYEOF
