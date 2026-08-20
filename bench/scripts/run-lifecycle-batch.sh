#!/usr/bin/env bash
# run-lifecycle-batch.sh --batch <policy.yaml> --retention-root <root>
#                          --mode preview|pilot|baseline [--dry-run]
#                          [--acknowledge-provider-spend]
#                          [--max-cost-usd <n>] [--max-wall-clock-seconds <n>]
#                          [--max-generated-tasks <n>]
#                          [--reps <n>] [--scenarios <id[,id...]>]
#                          [--reclaim-incomplete]
#
# T-E40-F10-004 (spec.md REQ-F-001..004, REQ-F-007, REQ-NF-007, ADR-F10-01):
# operator batch driver. Enumerates the (scenario, rep) matrix from an
# operator-supplied batch-policy YAML against the admitted I-04 scenario
# index, sources lib/spend-gate.sh for every provider-backed mode, and for
# non-preview modes invokes the existing run-lifecycle.sh once per pair and
# evaluate-lifecycle.sh once per completed pair, then retains their outputs
# byte-for-byte under the operator-declared --retention-root. In preview
# mode it invokes run-lifecycle.sh --mode dry-run once per requested
# scenario for stage resolution only and dispatches nothing else
# (ADR-F10-01) -- --dry-run is a REQ-F-001 alias for --mode preview, never
# a second flag convention.
#
# --- Batch-policy.yaml (this script's own input contract; not an upstream
# I-## shape) ---
#   schema_version: "1.0"
#   min_reps: <positive integer>            # REQ-F-007's declared minimum,
#                                            # read by aggregate-lifecycle.sh
#                                            # (T-E40-F10-008); also this
#                                            # driver's default --reps.
#   scenario_index: <path>                  # optional, default
#                                            # bench/scenarios/scenarios.yaml
#   scenarios:
#     <scenario_id>:
#       reps: <positive integer>            # optional per-scenario override
#       root_key: "<entity key>"            # the already-existing root
#                                            # entity key inside scratch_root
#                                            # (run-lifecycle.sh's --root
#                                            # contract: an existing entity
#                                            # key, not a key this driver
#                                            # creates -- spec.md Integration
#                                            # section, Notes for Agent).
#       scratch_root: "<dir>"                # a pre-built scratch Shark
#                                             # project template directory
#                                             # containing that root entity.
#                                             # This driver never mutates the
#                                             # template: every invocation
#                                             # (preview included) copies it
#                                             # into a fresh ephemeral
#                                             # directory first, so reps
#                                             # never share mutable DB state
#                                             # and a preview never leaves a
#                                             # side effect on the template.
#       i05_bundle_dir: "<dir>"              # optional; the I-05
#                                             # stage-evidence bundle
#                                             # directory evaluate-lifecycle.sh
#                                             # reads via --i05. F10 reuses
#                                             # I-05 read-only (ADR-F10-04)
#                                             # and does not produce it; a
#                                             # scenario without this
#                                             # configured is dispatched
#                                             # (run-lifecycle.sh still
#                                             # runs) but its pair cannot be
#                                             # evaluated and is classified
#                                             # `failed`, named as such.
#
# root_key/scratch_root/i05_bundle_dir are operator-prepared inputs, not
# something this driver bootstraps: creating a scratch Shark project and its
# seed entity is out of F10's scope (component-changes table: F10 invokes
# run-lifecycle.sh/evaluate-lifecycle.sh with their EXISTING signatures; it
# adds no Shark database, migration, or entity-creation surface, REQ-NF-001).
#
# Exit status (schema-owned, bench/reports/lifecycle-baseline-schema.yaml
# refusal_exit_status): 0 success, 2 usage error, 3 spend-gate refusal
# (ADR-F10-02/03, before any subprocess), 4 batch completed with at least
# one failed or unreclaimed-incomplete pair.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# TD-077 defect-class precedent (run-batch.sh's own header comment): resolved
# via the SIBLING path, never a bare PATH-resolved name, so a self-test can
# substitute a stub through one overridable variable without touching PATH
# resolution for anything else this script calls.
RUN_LIFECYCLE_BIN="${RUN_LIFECYCLE_BIN:-$SCRIPT_DIR/run-lifecycle.sh}"
EVALUATE_LIFECYCLE_BIN="${EVALUATE_LIFECYCLE_BIN:-$SCRIPT_DIR/evaluate-lifecycle.sh}"

usage() {
	cat >&2 <<'EOF'
usage: run-lifecycle-batch.sh --batch <policy.yaml> --retention-root <root>
                               --mode preview|pilot|baseline [--dry-run]
                               [--acknowledge-provider-spend]
                               [--max-cost-usd <n>]
                               [--max-wall-clock-seconds <n>]
                               [--max-generated-tasks <n>]
                               [--reps <n>] [--scenarios <id[,id...]>]
                               [--reclaim-incomplete]

--dry-run is the sole alias for --mode preview (REQ-F-001); no other preview
flag convention is supported.
EOF
	exit 2
}

# Captured before any parsing: spend-gate.sh's AC-T2 contract requires the
# driver's *raw, unmodified* argv, so this array is built first and never
# mutated afterward.
ORIGINAL_ARGV=("$@")

batch_policy=""
retention_root=""
mode=""
dry_run="false"
reps_flag=""
scenarios_filter=""
reclaim_incomplete="false"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--batch)
		[[ $# -ge 2 ]] || usage
		batch_policy="$2"
		shift 2
		;;
	--retention-root)
		[[ $# -ge 2 ]] || usage
		retention_root="$2"
		shift 2
		;;
	--mode)
		[[ $# -ge 2 ]] || usage
		mode="$2"
		shift 2
		;;
	--dry-run)
		dry_run="true"
		shift
		;;
	--acknowledge-provider-spend)
		shift
		;;
	--max-cost-usd | --max-wall-clock-seconds | --max-generated-tasks)
		[[ $# -ge 2 ]] || usage
		shift 2
		;;
	--reps)
		[[ $# -ge 2 ]] || usage
		reps_flag="$2"
		shift 2
		;;
	--scenarios)
		[[ $# -ge 2 ]] || usage
		scenarios_filter="$2"
		shift 2
		;;
	--reclaim-incomplete)
		reclaim_incomplete="true"
		shift
		;;
	--help)
		usage
		;;
	*)
		usage
		;;
	esac
done

# REQ-F-001: --dry-run is a hard alias for --mode preview, applied
# unconditionally -- it is never a third convention alongside --mode.
[[ "$dry_run" == "true" ]] && mode="preview"

[[ -n "$batch_policy" ]] || usage
[[ -f "$batch_policy" ]] || {
	echo "run-lifecycle-batch: batch policy not found: $batch_policy" >&2
	exit 2
}
case "$mode" in
preview | pilot | baseline) ;;
*) usage ;;
esac
# --retention-root is a hard usage error only for --mode preview (REQ-F-001:
# preview must print "the resolved retention root", so it is not optional
# there, but preview is not spend-gated). For --mode pilot|baseline its
# absence MUST be a spend-gate REFUSAL (exit 3, reason
# missing_retention_root, REQ-F-002) rather than a usage error (exit 2) --
# so it is deliberately NOT checked here; spend_gate_check_all below is the
# single owner of that condition for provider-backed modes.
if [[ "$mode" == "preview" ]]; then
	[[ -n "$retention_root" ]] || usage
fi
if [[ -n "$reps_flag" ]]; then
	[[ "$reps_flag" =~ ^[0-9]+$ && "$reps_flag" -ge 1 ]] || {
		echo "run-lifecycle-batch: --reps must be a positive integer, got '$reps_flag'" >&2
		exit 2
	}
fi

command -v python3 >/dev/null 2>&1 || {
	echo "run-lifecycle-batch: python3 not found on PATH" >&2
	exit 2
}

# ---------------------------------------------------------------------------
# ADR-F10-02/03: for provider-backed modes, spend_gate_check_all is called
# with the driver's untouched raw argv BEFORE anything below this point --
# no batch-policy parse, no retention-root creation, no matrix enumeration,
# no subprocess/checkout/Shark call. A refusal therefore costs zero
# subprocesses (REQ-F-002, REQ-F-003).
# ---------------------------------------------------------------------------
if [[ "$mode" == "pilot" || "$mode" == "baseline" ]]; then
	# shellcheck source=lib/spend-gate.sh
	source "$SCRIPT_DIR/lib/spend-gate.sh"
	gate_rc=0
	spend_gate_check_all "$mode" "${ORIGINAL_ARGV[@]}" || gate_rc=$?
	if [[ "$gate_rc" -ne 0 ]]; then
		exit "$gate_rc"
	fi
fi

out_root_canon="$(realpath -m -- "$retention_root")"
# shellcheck source=lib/path-safety.sh
source "$SCRIPT_DIR/lib/path-safety.sh"

[[ "$mode" == "preview" ]] || mkdir -p "$out_root_canon"

# ---------------------------------------------------------------------------
# Matrix enumeration (pure computation: reads the batch policy and the I-04
# scenario index only -- no subprocess, no Shark call). One row per
# requested scenario; `reps` is a count on the row, not N separate rows
# (REQ-F-001's "scenario matrix (scenario id, version, family, reps)").
# ---------------------------------------------------------------------------
MATRIX_TMP="$(mktemp)"
cleanup_matrix() { rm -f "$MATRIX_TMP"; }
trap cleanup_matrix EXIT

set +e
python3 - "$BENCH_DIR" "$batch_policy" "$scenarios_filter" "$reps_flag" >"$MATRIX_TMP" <<'PYEOF'
import json
import os
import sys

import yaml

bench_dir, batch_policy_path, scenarios_filter, reps_override = sys.argv[1:5]

with open(batch_policy_path) as f:
    policy = yaml.safe_load(f) or {}
if not isinstance(policy, dict):
    print(f"run-lifecycle-batch: batch policy must be a YAML mapping: {batch_policy_path}", file=sys.stderr)
    raise SystemExit(1)

min_reps_raw = policy.get("min_reps", 1)
try:
    min_reps = int(min_reps_raw)
    if min_reps < 1:
        raise ValueError
except (TypeError, ValueError):
    print(f"run-lifecycle-batch: batch policy min_reps must be a positive integer, got {min_reps_raw!r}", file=sys.stderr)
    raise SystemExit(1)

scenario_index_field = policy.get("scenario_index") or "scenarios/scenarios.yaml"
scenario_index_path = scenario_index_field if os.path.isabs(scenario_index_field) else os.path.join(bench_dir, scenario_index_field)
if not os.path.isfile(scenario_index_path):
    print(f"run-lifecycle-batch: scenario index not found: {scenario_index_path}", file=sys.stderr)
    raise SystemExit(1)

with open(scenario_index_path) as f:
    index = yaml.safe_load(f) or {}
scenario_dirs = index.get("scenarios") or []
if not scenario_dirs:
    print(f"run-lifecycle-batch: no scenarios found in {scenario_index_path}", file=sys.stderr)
    raise SystemExit(1)
index_dir = os.path.dirname(scenario_index_path)

packages = []
for rel in scenario_dirs:
    pkg_path = os.path.join(index_dir, rel, "package.yaml")
    if not os.path.isfile(pkg_path):
        print(f"run-lifecycle-batch: scenario package not found: {pkg_path}", file=sys.stderr)
        raise SystemExit(1)
    with open(pkg_path) as f:
        pkg = yaml.safe_load(f) or {}
    scenario_id = str(pkg.get("scenario_id", ""))
    if not scenario_id:
        print(f"run-lifecycle-batch: scenario package missing scenario_id: {pkg_path}", file=sys.stderr)
        raise SystemExit(1)
    packages.append((scenario_id, pkg_path, pkg))

requested_ids = [s for s in scenarios_filter.split(",") if s] if scenarios_filter else []
if requested_ids:
    by_id = {sid: (sid, path, pkg) for sid, path, pkg in packages}
    unresolved = [rid for rid in requested_ids if rid not in by_id]
    for rid in unresolved:
        print(f"run-lifecycle-batch: --scenarios id '{rid}' did not resolve against the scenario index", file=sys.stderr)
    selected = [by_id[rid] for rid in requested_ids if rid in by_id]
    if not selected:
        print("run-lifecycle-batch: --scenarios resolved to zero admitted scenarios; nothing to run", file=sys.stderr)
        raise SystemExit(1)
else:
    selected = packages

policy_scenarios = policy.get("scenarios") or {}
if not isinstance(policy_scenarios, dict):
    print("run-lifecycle-batch: batch policy 'scenarios' must be a mapping", file=sys.stderr)
    raise SystemExit(1)

reps_override_val = None
if reps_override:
    reps_override_val = int(reps_override)

rows = []
for scenario_id, pkg_path, pkg in selected:
    scenario_version = str(pkg.get("scenario_version", "1"))
    family = str(pkg.get("entity_family", "unknown"))
    entry = policy_scenarios.get(scenario_id) or {}
    if not isinstance(entry, dict):
        print(f"run-lifecycle-batch: batch policy scenarios.{scenario_id} must be a mapping", file=sys.stderr)
        raise SystemExit(1)
    if reps_override_val is not None:
        reps = reps_override_val
    else:
        reps_raw = entry.get("reps", min_reps)
        try:
            reps = int(reps_raw)
            if reps < 1:
                raise ValueError
        except (TypeError, ValueError):
            print(f"run-lifecycle-batch: scenarios.{scenario_id}.reps must be a positive integer, got {reps_raw!r}", file=sys.stderr)
            raise SystemExit(1)
    root_key = str(entry.get("root_key") or "")
    scratch_root = str(entry.get("scratch_root") or "")
    if scratch_root and not os.path.isabs(scratch_root):
        scratch_root = os.path.join(bench_dir, scratch_root)
    i05_bundle_dir = str(entry.get("i05_bundle_dir") or "")
    if i05_bundle_dir and not os.path.isabs(i05_bundle_dir):
        i05_bundle_dir = os.path.join(bench_dir, i05_bundle_dir)
    stage_matrix = pkg.get("stage_matrix") or {}
    prelude = stage_matrix.get("prelude") or {}
    lifecycle_mode = (stage_matrix.get("lifecycle") or {}).get("mode", "unknown")
    rows.append({
        "scenario_id": scenario_id,
        "scenario_version": scenario_version,
        "family": family,
        "reps": reps,
        "root_key": root_key,
        "scratch_root": scratch_root,
        "i05_bundle_dir": i05_bundle_dir,
        "package_path": os.path.abspath(pkg_path),
        "prelude": prelude,
        "lifecycle_mode": lifecycle_mode,
        "min_reps": min_reps,
    })

for row in rows:
    print(json.dumps(row, sort_keys=True))
PYEOF
enumerate_rc=$?
set -e
if [[ "$enumerate_rc" -ne 0 ]]; then
	exit 1
fi

readarray -t MATRIX_ROWS <"$MATRIX_TMP"
[[ "${#MATRIX_ROWS[@]}" -gt 0 ]] || {
	echo "run-lifecycle-batch: matrix enumeration produced zero scenario rows" >&2
	exit 1
}

# ---------------------------------------------------------------------------
# PREVIEW (ADR-F10-01, REQ-F-001, REQ-NF-002): calls run-lifecycle.sh
# --mode dry-run once per requested scenario for stage resolution only,
# against an ephemeral copy of the scenario's declared scratch_root template
# (never the template itself), and dispatches nothing else. Never aborts on
# one scenario's resolution failure (matches run-batch.sh's non-aborting
# discipline) -- an unconfigured or failing scenario is named, not fatal to
# the preview as a whole.
# ---------------------------------------------------------------------------
if [[ "$mode" == "preview" ]]; then
	PREVIEW_WORK="$(mktemp -d)"
	trap 'cleanup_matrix; rm -rf "$PREVIEW_WORK"' EXIT

	set +e
	python3 - "$RUN_LIFECYCLE_BIN" "$MATRIX_TMP" "$out_root_canon" "$PREVIEW_WORK" \
		"${ORIGINAL_ARGV[*]}" <<'PYEOF'
import json
import os
import re
import shutil
import subprocess
import sys

run_lifecycle_bin, matrix_path, retention_root, work_dir, argv_joined = sys.argv[1:6]


def flag_value(argv_str, flag):
    parts = argv_str.split(" ") if argv_str else []
    for i, tok in enumerate(parts):
        if tok == flag and i + 1 < len(parts):
            return parts[i + 1]
    return None


ceilings = {
    "max_cost_usd": flag_value(argv_joined, "--max-cost-usd"),
    "max_wall_clock_seconds": flag_value(argv_joined, "--max-wall-clock-seconds"),
    "max_generated_tasks": flag_value(argv_joined, "--max-generated-tasks"),
}

rows = []
with open(matrix_path) as f:
    for line in f:
        line = line.strip()
        if line:
            rows.append(json.loads(line))

families = sorted({row["family"] for row in rows})

print("=== run-lifecycle-batch: operator preview (zero provider calls) ===")
print(f"retention root: {retention_root}")
print("ceilings:")
for name, value in ceilings.items():
    print(f"  {name}: {value if value is not None else 'not declared'}")

print("scenario matrix:")
for row in rows:
    print(f"  - scenario_id={row['scenario_id']} scenario_version={row['scenario_version']} family={row['family']} reps={row['reps']}")

    prelude = row["prelude"] or {}
    print("    applicable stages (I-04 prelude D01-D05):")
    if prelude:
        for stage_id in sorted(prelude):
            entry = prelude[stage_id] or {}
            applicable = bool(entry.get("applicable"))
            reason = entry.get("reason", "")
            call_line = "1 planned provider call" if applicable else "replayed (0 planned provider calls)"
            print(f"      {stage_id}: applicable={applicable} -> {call_line} ({reason})")
    else:
        print("      (no prelude entries in scenario package)")

    print(f"    lifecycle stage_matrix.mode: {row['lifecycle_mode']}")

    root_key = row["root_key"]
    scratch_root = row["scratch_root"]
    if not root_key or not scratch_root:
        print("    stage resolution: skipped (root_key/scratch_root not configured for this scenario in the batch policy)")
        continue
    if not os.path.isdir(scratch_root):
        print(f"    stage resolution: skipped (configured scratch_root does not exist: {scratch_root})")
        continue

    ephemeral = os.path.join(work_dir, re.sub(r"[^A-Za-z0-9._-]", "_", row["scenario_id"]))
    shutil.copytree(scratch_root, ephemeral)
    output_path = os.path.join(work_dir, f"{row['scenario_id']}-preview-lifecycle.jsonl")

    process = subprocess.run(
        [
            run_lifecycle_bin,
            "--scenario", row["package_path"],
            "--run-id", f"preview-{row['scenario_id']}",
            "--root", root_key,
            "--scratch-root", ephemeral,
            "--output", output_path,
            "--mode", "dry-run",
        ],
        text=True, capture_output=True, check=False,
    )
    if process.returncode != 0:
        detail = (process.stderr or process.stdout).strip().splitlines()
        detail = detail[-1] if detail else "no diagnostic output"
        print(f"    stage resolution: FAILED (run-lifecycle.sh --mode dry-run exited {process.returncode}: {detail})")
        continue

    with open(output_path) as f:
        record = json.loads(f.readline())
    stage_count = len(record.get("stages") or [])
    dispatch_count = len(record.get("dispatches") or [])
    terminal = (record.get("outcome") or {}).get("terminal", "unknown")
    print(f"    stage resolution: OK -- {dispatch_count} dispatch(es), {stage_count} stage(s) resolved, terminal={terminal}, planned provider calls=0 (dry-run short-circuit)")

print("pilot-ledger state (per family, REQ-F-005 presence check):")
ledger_path = os.path.join(retention_root, "pilot-ledger.jsonl")
attested_families = set()
if os.path.isfile(ledger_path):
    with open(ledger_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                entry = json.loads(line)
            except json.JSONDecodeError:
                continue
            fam = entry.get("family")
            if fam:
                attested_families.add(fam)
for fam in families:
    state = "attestation present" if fam in attested_families else "no attestation"
    print(f"  {fam}: {state}")

print("=== end preview: zero provider or network calls made ===")
PYEOF
	preview_rc=$?
	set -e
	exit "$preview_rc"
fi

# ---------------------------------------------------------------------------
# PILOT / BASELINE dispatch (REQ-F-004, REQ-NF-007). Classification mirrors
# run-batch.sh's discipline exactly (ADR-F03-07 precedent):
#   skipped_complete         -- <retention_root>/scenarios/<id>/<rep>/
#                                evaluation.jsonl already present.
#   incomplete_prior_attempt -- directory present, non-empty, no
#                                evaluation.jsonl. Skipped by default
#                                (never re-invoked into the stale
#                                directory); --reclaim-incomplete quarantines
#                                it under .incomplete/ (a MOVE, never a
#                                delete, REQ-NF-007) before dispatch.
#   quarantined_and_rerun    -- same state, quarantined then dispatched.
#   pending_run               -- no prior directory; dispatched.
#   failed                    -- dispatched but run-lifecycle.sh or
#                                evaluate-lifecycle.sh did not complete
#                                (recorded, batch proceeds -- one pair's
#                                failure never aborts the batch).
# ---------------------------------------------------------------------------
SUMMARY_TMP="$(mktemp)"
INVALID_TMP="$(mktemp)"
trap 'cleanup_matrix; rm -f "$SUMMARY_TMP" "$INVALID_TMP"' EXIT
overall_bad="false"

append_summary() {
	# append_summary <scenario_id> <scenario_version> <family> <rep> <classification>
	python3 -c '
import json, sys
sid, ver, fam, rep, cls = sys.argv[1:6]
print(json.dumps({"scenario_id": sid, "scenario_version": ver, "family": fam, "rep": int(rep), "classification": cls}))
' "$1" "$2" "$3" "$4" "$5" >>"$SUMMARY_TMP"
}

record_invalid() {
	# record_invalid <scenario_id> <rep> <reason>
	python3 -c '
import json, sys
sid, rep, reason = sys.argv[1:4]
print(json.dumps({"scenario_id": sid, "rep": int(rep), "retention_path": f"scenarios/{sid}/{rep}", "invalidity_reasons": [reason]}))
' "$1" "$2" "$3" >>"$INVALID_TMP"
}

retain_pair() {
	# retain_pair <scenario_id> <rep> <package_path> <lifecycle_jsonl> <evaluation_jsonl>
	# Byte-preserving copy (ADR-F10-05): every artifact below is copied with
	# shutil.copyfile (no re-serialization) and digested for manifest.json.
	local scenario_id="$1" rep="$2" package_path="$3" lifecycle_jsonl="$4" evaluation_jsonl="$5"
	local dest="$out_root_canon/scenarios/$scenario_id/$rep"
	assert_within_out_root "$dest" || return 1
	mkdir -p "$dest/evidence" "$dest/transcripts"
	python3 - "$scenario_id" "$rep" "$package_path" "$lifecycle_jsonl" "$evaluation_jsonl" "$dest" <<'PYEOF'
import hashlib
import json
import shutil
import sys
from pathlib import Path

scenario_id, rep, package_path, lifecycle_jsonl, evaluation_jsonl, dest = sys.argv[1:7]
dest = Path(dest)

artifacts = {}


def copy_artifact(name, source):
    source_path = Path(source)
    target = dest / name
    if source_path.is_file():
        shutil.copyfile(source_path, target)
        digest = hashlib.sha256(target.read_bytes()).hexdigest()
        artifacts[name] = {"source_path": str(source_path), "sha256": digest}
    else:
        # Not yet available from this driver's own inputs (e.g. I-05 bundle
        # not configured, entity-history export not wired) -- an honest
        # placeholder, never a fabricated byte-identical claim.
        target.write_text("", encoding="utf-8")
        artifacts[name] = {"source_path": "", "sha256": hashlib.sha256(b"").hexdigest()}


copy_artifact("package.yaml", package_path)
copy_artifact("lifecycle.jsonl", lifecycle_jsonl)
copy_artifact("evaluation.jsonl", evaluation_jsonl)
copy_artifact("entity-history.json", "")
copy_artifact("oracle.json", "")

manifest = {
    "scenario_id": scenario_id,
    "rep": int(rep),
    "artifacts": artifacts,
}
(dest / "manifest.json").write_text(json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
PYEOF
}

dispatch_pair() {
	# dispatch_pair <scenario_id> <scenario_version> <family> <rep> <package_path> <root_key> <scratch_root> <i05_bundle_dir> <success_label>
	local scenario_id="$1" scenario_version="$2" family="$3" rep="$4" package_path="$5"
	local root_key="$6" scratch_root="$7" i05_bundle_dir="$8" success_label="$9"

	if [[ -z "$root_key" || -z "$scratch_root" ]]; then
		echo "run-lifecycle-batch: $scenario_id rep $rep: root_key/scratch_root not configured; recorded failed, batch proceeds" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "root_key_or_scratch_root_not_configured"
		overall_bad="true"
		return 0
	fi

	local pair_work
	pair_work="$(mktemp -d)"
	local ephemeral="$pair_work/scratch"
	cp -a "$scratch_root" "$ephemeral"
	local lifecycle_out="$pair_work/lifecycle.jsonl"

	echo "run-lifecycle-batch: dispatching $scenario_id rep $rep" >&2
	set +e
	"$RUN_LIFECYCLE_BIN" --scenario "$package_path" --run-id "${scenario_id}-rep${rep}" \
		--root "$root_key" --scratch-root "$ephemeral" --output "$lifecycle_out" </dev/null
	local run_rc=$?
	set -e

	if [[ "$run_rc" -ne 0 ]]; then
		echo "run-lifecycle-batch: $scenario_id rep $rep FAILED (run-lifecycle.sh exit $run_rc)" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "lifecycle_run_failed"
		overall_bad="true"
		rm -rf "$pair_work"
		return 0
	fi

	if [[ -z "$i05_bundle_dir" ]]; then
		echo "run-lifecycle-batch: $scenario_id rep $rep: i05_bundle_dir not configured; cannot evaluate, recorded failed" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "i05_bundle_not_configured"
		overall_bad="true"
		rm -rf "$pair_work"
		return 0
	fi

	local evaluation_out="$pair_work/evaluation.jsonl"
	set +e
	"$EVALUATE_LIFECYCLE_BIN" --i05 "$i05_bundle_dir" --i07 "$lifecycle_out" \
		--scenario "$package_path" --output "$evaluation_out" </dev/null
	local eval_rc=$?
	set -e

	if [[ "$eval_rc" -ne 0 ]]; then
		echo "run-lifecycle-batch: $scenario_id rep $rep FAILED (evaluate-lifecycle.sh exit $eval_rc)" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "evaluation_failed"
		overall_bad="true"
		rm -rf "$pair_work"
		return 0
	fi

	retain_pair "$scenario_id" "$rep" "$package_path" "$lifecycle_out" "$evaluation_out"
	append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "$success_label"
	rm -rf "$pair_work"
}

# classify_pair <scenario_id> <rep> -- prints skipped_complete /
# incomplete_prior_attempt / pending, mirroring run-batch.sh classify_pair.
classify_pair() {
	local scenario_id="$1" rep="$2"
	local dir="$out_root_canon/scenarios/$scenario_id/$rep"
	if [[ -f "$dir/evaluation.jsonl" ]]; then
		echo "skipped_complete"
		return 0
	fi
	[[ -d "$dir" ]] || {
		echo "pending"
		return 0
	}
	if [[ -n "$(ls -A "$dir" 2>/dev/null)" ]]; then
		echo "incomplete_prior_attempt"
	else
		echo "pending"
	fi
}

# quarantine_pair <scenario_id> <rep> -- REQ-NF-007: a MOVE, never a delete.
quarantine_pair() {
	local scenario_id="$1" rep="$2"
	local dir="$out_root_canon/scenarios/$scenario_id/$rep"
	local incomplete_root="$out_root_canon/.incomplete/$scenario_id"
	assert_within_out_root "$dir" || return 1
	assert_within_out_root "$incomplete_root" || return 1
	mkdir -p "$incomplete_root"
	local seq=1
	while [[ -e "$incomplete_root/rep-$rep-$seq" ]]; do
		seq=$((seq + 1))
	done
	local dest="$incomplete_root/rep-$rep-$seq"
	assert_within_out_root "$dest" || return 1
	mv "$dir" "$dest"
	echo "run-lifecycle-batch: quarantined incomplete prior attempt: $dir -> $dest" >&2
}

for row_json in "${MATRIX_ROWS[@]}"; do
	scenario_id="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["scenario_id"])' "$row_json")"
	scenario_version="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["scenario_version"])' "$row_json")"
	family="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["family"])' "$row_json")"
	reps="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["reps"])' "$row_json")"
	root_key="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["root_key"])' "$row_json")"
	scratch_root="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["scratch_root"])' "$row_json")"
	i05_bundle_dir="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["i05_bundle_dir"])' "$row_json")"
	package_path="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["package_path"])' "$row_json")"

	for ((rep = 1; rep <= reps; rep++)); do
		dir="$out_root_canon/scenarios/$scenario_id/$rep"
		if ! assert_within_out_root "$dir"; then
			append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
			record_invalid "$scenario_id" "$rep" "path_containment_check_failed"
			overall_bad="true"
			continue
		fi

		classification="$(classify_pair "$scenario_id" "$rep")"
		case "$classification" in
		skipped_complete)
			echo "run-lifecycle-batch: $scenario_id rep $rep already retained; skipped_complete" >&2
			append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "skipped_complete"
			;;
		incomplete_prior_attempt)
			if [[ "$reclaim_incomplete" == "true" ]]; then
				quarantine_rc=0
				quarantine_pair "$scenario_id" "$rep" || quarantine_rc=$?
				if [[ "$quarantine_rc" -ne 0 ]]; then
					echo "run-lifecycle-batch: failed to quarantine $scenario_id rep $rep; skipping, batch proceeds" >&2
					append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
					overall_bad="true"
				else
					dispatch_pair "$scenario_id" "$scenario_version" "$family" "$rep" "$package_path" \
						"$root_key" "$scratch_root" "$i05_bundle_dir" "quarantined_and_rerun"
				fi
			else
				echo "run-lifecycle-batch: $scenario_id rep $rep is an incomplete prior attempt (directory present, evaluation.jsonl absent); skipping (pass --reclaim-incomplete to quarantine and re-run)" >&2
				append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "incomplete_prior_attempt"
				overall_bad="true"
			fi
			;;
		pending)
			dispatch_pair "$scenario_id" "$scenario_version" "$family" "$rep" "$package_path" \
				"$root_key" "$scratch_root" "$i05_bundle_dir" "pending_run"
			;;
		esac
	done
done

python3 - "$SUMMARY_TMP" "$INVALID_TMP" "$mode" "$out_root_canon" "$batch_policy" "$reclaim_incomplete" \
	"${ORIGINAL_ARGV[*]}" <<'PYEOF'
import hashlib
import json
import sys
import time

import yaml

summary_path, invalid_path, mode, retention_root, batch_policy_path, reclaim, argv_joined = sys.argv[1:8]


def flag_value(argv_str, flag):
    parts = argv_str.split(" ") if argv_str else []
    for i, tok in enumerate(parts):
        if tok == flag and i + 1 < len(parts):
            return parts[i + 1]
    return None


pairs = []
with open(summary_path) as f:
    for line in f:
        line = line.strip()
        if line:
            pairs.append(json.loads(line))

invalid = []
with open(invalid_path) as f:
    for line in f:
        line = line.strip()
        if line:
            invalid.append(json.loads(line))

counts = {}
for p in pairs:
    counts[p["classification"]] = counts.get(p["classification"], 0) + 1

with open(batch_policy_path, "rb") as f:
    policy_bytes = f.read()
    policy_digest = hashlib.sha256(policy_bytes).hexdigest()

# T-E40-F10-008 fix: batch.json is the ONLY retained identity source
# aggregate-lifecycle.sh reads (its API contract takes just
# --retention-root, REQ-NF-003 pure-function-of-the-root), but this
# writer previously never echoed the batch policy's declared `min_reps`
# into batch.json even though it already re-reads batch_policy_path here
# for the policy digest -- aggregate.json's required `/identity/min_reps`
# (spec.md "aggregate.json blocks" table) had no retained source to read
# it from. Re-parsing the same policy file already opened above for its
# digest, mirroring the earlier min_reps extraction/validation in this
# script's preview-matrix step, closes that gap without adding a new
# operator input.
policy_for_min_reps = yaml.safe_load(policy_bytes) or {}
min_reps_raw = policy_for_min_reps.get("min_reps", 1) if isinstance(policy_for_min_reps, dict) else 1
try:
    min_reps = int(min_reps_raw)
    if min_reps < 1:
        raise ValueError
except (TypeError, ValueError):
    print(f"run-lifecycle-batch: batch policy min_reps must be a positive integer, got {min_reps_raw!r}", file=sys.stderr)
    raise SystemExit(1)

batch = {
    "phase": "lifecycle_v2",
    "batch_id": f"batch-{int(time.time())}",
    "mode": mode,
    "retention_root": retention_root,
    "batch_policy_digest": policy_digest,
    "min_reps": min_reps,
    "ceilings": {
        "max_cost_usd": flag_value(argv_joined, "--max-cost-usd"),
        "max_wall_clock_seconds": flag_value(argv_joined, "--max-wall-clock-seconds"),
        "max_generated_tasks": flag_value(argv_joined, "--max-generated-tasks"),
    },
    "acknowledgement_ref": {
        "flag": "--acknowledge-provider-spend",
        "present": "--acknowledge-provider-spend" in (argv_joined.split(" ") if argv_joined else []),
    },
    "reclaim_incomplete": reclaim == "true",
    "counts": counts,
    "pairs": pairs,
}
with open(f"{retention_root}/batch.json", "w") as f:
    json.dump(batch, f, indent=2, sort_keys=True)
    f.write("\n")

import os

os.makedirs(f"{retention_root}/invalid", exist_ok=True)
with open(f"{retention_root}/invalid/index.jsonl", "w") as f:
    for row in invalid:
        f.write(json.dumps(row, sort_keys=True) + "\n")

print(json.dumps(batch, indent=2, sort_keys=True))
PYEOF

if [[ "$overall_bad" == "true" ]]; then
	exit 4
fi
exit 0
