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
# UAT-R3-01 fix (T-E40-F10-004): the real producer for retain_pair's
# entity-history.json artifact -- see export-entity-history.sh's own header.
ENTITY_HISTORY_EXPORT_BIN="${ENTITY_HISTORY_EXPORT_BIN:-$SCRIPT_DIR/export-entity-history.sh}"
# UAT round-6 fix (T-E40-F10-004): the schema driving lib/verify_pair_
# retention's own retention_required_artifacts enumeration -- same override
# convention aggregate-lifecycle.sh's own LIFECYCLE_SCHEMA already
# establishes, never a private hardcoded copy of the schema path.
LIFECYCLE_SCHEMA="${LIFECYCLE_SCHEMA:-$BENCH_DIR/reports/lifecycle-baseline-schema.yaml}"

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
# GATE_REP_BASE mirrors run-review-comparison.sh's own reserved gate-rep
# band (code-review-2026-08-21T0330-E40-F10.md finding 1: gate reps live at
# GATE_REP_BASE+1/+2 precisely so this driver's sequential rep allocation
# below, `for ((rep = 1; rep <= reps; rep++))`, can never collide with
# them). codex R4-1 (code-review-2026-08-21T0459-E40-F10.md, downgraded to
# non-blocker tech-debt: reaching this requires an operator-supplied --reps
# five to six orders of magnitude beyond any plausible retained-pilot reps
# count) -- a one-line defensive validation costs nothing and removes the
# theoretical gap outright.
GATE_REP_BASE=900000
if [[ -n "$reps_flag" ]]; then
	[[ "$reps_flag" =~ ^[0-9]+$ && "$reps_flag" -ge 1 ]] || {
		echo "run-lifecycle-batch: --reps must be a positive integer, got '$reps_flag'" >&2
		exit 2
	}
	[[ "$reps_flag" -lt "$GATE_REP_BASE" ]] || {
		echo "run-lifecycle-batch: --reps must be less than the reserved gate-rep band ($GATE_REP_BASE, run-review-comparison.sh's gate_rep()), got '$reps_flag'" >&2
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
OPERATOR_LIMITS_FILE=""
if [[ "$mode" == "pilot" || "$mode" == "baseline" ]]; then
	# shellcheck source=lib/spend-gate.sh
	source "$SCRIPT_DIR/lib/spend-gate.sh"
	gate_rc=0
	spend_gate_check_all "$mode" "${ORIGINAL_ARGV[@]}" || gate_rc=$?
	if [[ "$gate_rc" -ne 0 ]]; then
		exit "$gate_rc"
	fi
	# UAT-R2-01 fix (uat-2026-08-21T185059Z-E40-F10.md): spend_gate_check_all
	# just proved the operator's ceilings are present, numeric, and strictly
	# positive -- the ONLY thing missing before this fix was carrying those
	# same, already-validated values forward to run-lifecycle.sh's own
	# --limits policy (the sole execution-time enforcer/recorder,
	# run-lifecycle.sh's limits_from()/lines ~299-314). Without this, every
	# dispatch below fell through to the scenario package's resource_policy
	# default instead, making the acknowledgement gate above pure theater.
	#
	# Re-reads ORIGINAL_ARGV via spend-gate.sh's OWN _spend_gate_flag_value
	# accessor -- the exact function spend_gate_check_ceiling just used to
	# validate these three flags -- rather than a second, independent argv
	# parser in this file's own arg-parsing loop above. A second parser with
	# different first-vs-last-duplicate-flag semantics would silently
	# reintroduce this same defect class (validated value != materialized
	# value) the moment an operator's argv repeats a ceiling flag; calling
	# the identical function against the identical array is closed by
	# construction instead. One file, materialized once per batch invocation
	# (the ceilings are constant across every (scenario, rep) pair in this
	# run) and reused by every dispatch_pair() call -- never a second,
	# divergent per-pair policy.
	# `|| ...=""` on each: _spend_gate_flag_value returns 1 with no output
	# when a flag is absent, and under this script's `set -e` a bare
	# `var="$(possibly-failing-cmd)"` would abort the whole script right
	# here with no diagnostic -- before the named-diagnostic check below
	# ever ran. Capturing empty-on-failure instead lets that check do its
	# job.
	uat_r2_01_cost="$(_spend_gate_flag_value "--max-cost-usd" "${ORIGINAL_ARGV[@]}")" || uat_r2_01_cost=""
	uat_r2_01_wall="$(_spend_gate_flag_value "--max-wall-clock-seconds" "${ORIGINAL_ARGV[@]}")" || uat_r2_01_wall=""
	uat_r2_01_tasks="$(_spend_gate_flag_value "--max-generated-tasks" "${ORIGINAL_ARGV[@]}")" || uat_r2_01_tasks=""
	# Structural, not merely positional: spend_gate_check_all above already
	# guarantees all three resolve non-empty (it would have refused/exited
	# otherwise), but re-asserting it here means a future reordering of this
	# block fails loudly with a named diagnostic instead of silently writing
	# an empty --limits field that run-lifecycle.sh's own limits_from() would
	# then reject far downstream with a much less specific error.
	if [[ -z "$uat_r2_01_cost" || -z "$uat_r2_01_wall" || -z "$uat_r2_01_tasks" ]]; then
		echo "run-lifecycle-batch: internal error: spend gate passed but a ceiling did not resolve from ORIGINAL_ARGV" >&2
		exit 2
	fi
	OPERATOR_LIMITS_FILE="$(mktemp)"
	printf 'max_cost_usd: %s\nmax_wall_clock_seconds: %s\nmax_generated_tasks: %s\n' \
		"$uat_r2_01_cost" "$uat_r2_01_wall" "$uat_r2_01_tasks" \
		>"$OPERATOR_LIMITS_FILE"
fi

out_root_canon="$(realpath -m -- "$retention_root")"
# shellcheck source=lib/path-safety.sh
source "$SCRIPT_DIR/lib/path-safety.sh"

[[ "$mode" == "preview" ]] || mkdir -p "$out_root_canon"

# A retention root is a single-writer publication boundary.  Without an
# atomic root-scoped claim, two operators can classify the same pair as
# pending and race its retained artifacts and batch metadata.
LOCK_DIR="$out_root_canon/.lifecycle-batch.lock"
if [[ "$mode" != "preview" ]]; then
	if ! mkdir "$LOCK_DIR"; then
		echo "run-lifecycle-batch: retention root is already locked by another batch: $out_root_canon" >&2
		exit 4
	fi
	printf '%s\n' "$$" >"$LOCK_DIR/pid"
fi

# ---------------------------------------------------------------------------
# Matrix enumeration (pure computation: reads the batch policy and the I-04
# scenario index only -- no subprocess, no Shark call). One row per
# requested scenario; `reps` is a count on the row, not N separate rows
# (REQ-F-001's "scenario matrix (scenario id, version, family, reps)").
# ---------------------------------------------------------------------------
MATRIX_TMP="$(mktemp)"
cleanup_matrix() {
	rm -f "$MATRIX_TMP" "$OPERATOR_LIMITS_FILE"
	if [[ "$mode" != "preview" ]]; then
		rm -f "$LOCK_DIR/pid"
		rmdir "$LOCK_DIR" 2>/dev/null || true
	fi
}
trap cleanup_matrix EXIT

set +e
python3 - "$BENCH_DIR" "$batch_policy" "$scenarios_filter" "$reps_flag" >"$MATRIX_TMP" <<'PYEOF'
import json
import os
import re
import sys
import uuid

import yaml

bench_dir, batch_policy_path, scenarios_filter, reps_override = sys.argv[1:5]
SCENARIO_ID_PATTERN = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")

# GATE_REP_BASE mirrors the bash-level constant above and run-review-
# comparison.sh's own reserved gate-rep band (code-review-2026-08-21T0459-
# E40-F10.md round-4 tech-debt item 4): the bash-level --reps flag check
# above only covers ONE of the three inputs that feed the effective
# per-scenario reps count below -- a batch policy's own top-level
# `min_reps` or a per-scenario `scenarios.<id>.reps` override can reach the
# reserved band just as easily as --reps, and reps defaults to min_reps
# when a scenario declares no override, so min_reps is checked too.
GATE_REP_BASE = 900000


def check_reps_bound(value, label):
    if value >= GATE_REP_BASE:
        print(
            f"run-lifecycle-batch: {label} must be less than the reserved gate-rep band "
            f"({GATE_REP_BASE}, run-review-comparison.sh's gate_rep()), got {value!r}",
            file=sys.stderr,
        )
        raise SystemExit(1)


def positive_integer(value, label):
    if isinstance(value, bool):
        valid = False
    elif isinstance(value, int):
        valid = True
    elif isinstance(value, float):
        valid = value.is_integer()
    elif isinstance(value, str):
        valid = value.isdecimal()
    else:
        valid = False
    if not valid:
        print(f"run-lifecycle-batch: {label} must be a positive integer, got {value!r}", file=sys.stderr)
        raise SystemExit(1)
    parsed = int(value)
    if parsed < 1:
        print(f"run-lifecycle-batch: {label} must be a positive integer, got {value!r}", file=sys.stderr)
        raise SystemExit(1)
    return parsed


with open(batch_policy_path) as f:
    policy = yaml.safe_load(f) or {}
if not isinstance(policy, dict):
    print(f"run-lifecycle-batch: batch policy must be a YAML mapping: {batch_policy_path}", file=sys.stderr)
    raise SystemExit(1)

min_reps_raw = policy.get("min_reps", 1)
min_reps = positive_integer(min_reps_raw, "batch policy min_reps")
check_reps_bound(min_reps, "batch policy min_reps")

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
    if not SCENARIO_ID_PATTERN.fullmatch(scenario_id):
        print(f"run-lifecycle-batch: scenario package has unsafe scenario_id: {scenario_id!r}", file=sys.stderr)
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
    reps_override_val = positive_integer(reps_override, "--reps")
    check_reps_bound(reps_override_val, "--reps")

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
        reps = positive_integer(reps_raw, f"scenarios.{scenario_id}.reps")
        check_reps_bound(reps, f"scenarios.{scenario_id}.reps")
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
    # No source-side symlink guard needed here (round-5 sweep,
    # code-review-2026-08-21T1335-E40-F10.md finding 2): unlike dispatch_pair's
    # `cp -a`, shutil.copytree does NOT preserve a top-level symlink source --
    # it lists the resolved directory's entries via os.scandir and creates a
    # genuinely new, independent `ephemeral` directory (empirically verified
    # for this fix: a symlinked scratch_root here produces a real directory
    # at `ephemeral`, never a symlink to the template). Different copy
    # primitive, not vulnerable to the finding-2 defect class.
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
trap 'cleanup_matrix; rm -f "$SUMMARY_TMP" "$INVALID_TMP" "$OPERATOR_LIMITS_FILE"' EXIT
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
	# retain_pair <scenario_id> <rep> <package_path> <lifecycle_jsonl> <evaluation_jsonl> <entity_history_json> <i05_bundle_dir>
	# Byte-preserving copy (ADR-F10-05): every artifact is copied with
	# shutil.copyfile/a real directory walk (no re-serialization) and
	# digested for manifest.json, per the schema's digest_rules
	# (bench/reports/lifecycle-baseline-schema.yaml: file_encoding
	# sha256_raw_bytes for a file, compact_json_sorted_keys_utf8 over the
	# sorted {path, sha256} file list for a directory, PLUS
	# digest_rules.empty_artifact_semantics for the "artifact exists but is
	# empty" case). The manifest builder itself now lives in ONE shared
	# script, `lib/retain_pair` (code-review-2026-08-20T2138-E40-F10.md
	# findings 1/2/7: four near-identical reimplementations of this same
	# digest/manifest logic is exactly the drift that let two blockers
	# ship) -- run-review-comparison.sh's retain_gate() calls the same
	# script, never a second copy.
	#
	# UAT-R3-01 (round 3): lib/retain_pair now refuses (nonzero exit, no
	# manifest.json written) rather than fabricating a placeholder when any
	# required artifact's source is absent -- callers MUST check this
	# function's return status; dispatch_pair below does.
	local scenario_id="$1" rep="$2" package_path="$3" lifecycle_jsonl="$4" evaluation_jsonl="$5" entity_history_json="$6" i05_bundle_dir="$7"
	local dest="$out_root_canon/scenarios/$scenario_id/$rep"
	assert_within_out_root "$dest" || return 1
	# Symlink write-through (code-review-2026-08-21T0330-E40-F10.md finding
	# 2; STRUCTURALLY closed round-5, code-review-2026-08-21T1335-E40-F10.md
	# finding 1): assert_within_out_root refuses a `dest` symlink that
	# escapes out_root_canon, but a symlink redirected to ANOTHER retained
	# pair's directory INSIDE the same root -- whether AT dest itself or at
	# an ANCESTOR of dest (e.g. scenarios/$scenario_id itself) -- still
	# passes containment, and mkdir -p would silently no-op against it. Walk
	# the WHOLE chain from out_root_canon down to dest, not just dest's own
	# leaf -L status, so an ancestor-level redirect can never slip past this
	# guard the way it did round 5's leaf-only symlink_dest() check. Refuse
	# loudly before mkdir -p ever runs.
	if ! assert_no_symlink_in_chain "$out_root_canon" "$dest"; then
		echo "run-lifecycle-batch: $scenario_id rep $rep: refusing to retain -- a symlink was found in the path to $dest" >&2
		return 1
	fi
	mkdir -p "$dest"
	python3 "$SCRIPT_DIR/lib/retain_pair" "$scenario_id" "$rep" "$package_path" "$lifecycle_jsonl" "$evaluation_jsonl" "$entity_history_json" "$i05_bundle_dir" "$dest"
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
	if [[ ! -d "$scratch_root" ]]; then
		echo "run-lifecycle-batch: $scenario_id rep $rep: configured scratch_root does not exist: $scratch_root" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "scratch_root_not_found"
		overall_bad="true"
		return 0
	fi
	local pair_work
	pair_work="$(mktemp -d)"
	local ephemeral="$pair_work/scratch"
	# Source-side symlink policy, invariant 2 (path-safety.sh scope-freeze
	# paragraph): a symlinked scratch_root -- top-level (code-review-2026-08-
	# 21T1335-E40-F10.md round-5 finding 2) or NESTED anywhere inside a real
	# scratch_root directory tree (code-review-2026-08-21T1141-E40-F10.md
	# round-6 finding 1) -- is closed BY CONSTRUCTION here: copy_tree_
	# dereferenced (lib/path-safety.sh) dereferences every symlink it
	# encounters, top-level source argument included, producing a genuinely
	# independent ephemeral copy in every case. This replaces the former
	# `assert_source_not_symlink` hard-refusal (fail-loud on a top-level
	# symlink) -- now redundant, since dereference-on-copy makes a top-level
	# symlinked scratch_root just as safe as a nested one, with no separate
	# guard needed.
	# copy_tree_dereferenced (lib/path-safety.sh) dereferences every symlink
	# in the copied tree, producing a genuinely independent ephemeral copy.
	#
	# Unlike `cp -a` (which silently succeeds on a DANGLING nested symlink,
	# preserving it as-is), shutil.copytree(..., symlinks=False) raises when
	# it tries to dereference a nested symlink whose target does not exist
	# (post-fix empirical verification: a `shutil.Error` traceback, not a
	# clean nonzero exit). Guarded here -- matching every other scratch_root
	# failure mode dispatch_pair already handles (scratch_root_not_found,
	# scratch_root_is_symlink) -- so a single scenario's stale template
	# symlink is recorded invalid and the batch proceeds to the next pair,
	# instead of a bare Python traceback aborting the entire batch under
	# this script's own `set -euo pipefail`.
	if ! copy_tree_dereferenced "$scratch_root" "$ephemeral"; then
		echo "run-lifecycle-batch: $scenario_id rep $rep: failed to copy scratch_root into an ephemeral working directory (a nested symlink may be dangling)" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "scratch_root_copy_failed"
		overall_bad="true"
		rm -rf "$pair_work"
		return 0
	fi
	local lifecycle_out="$pair_work/lifecycle.jsonl"

	echo "run-lifecycle-batch: dispatching $scenario_id rep $rep" >&2
	set +e
	# UAT-R2-01: --limits carries the operator-acknowledged ceilings
	# (OPERATOR_LIMITS_FILE, materialized above from the same values
	# spend_gate_check_all already validated) through to run-lifecycle.sh's
	# own execution-time enforcement/recording -- never left to fall through
	# to the scenario package's resource_policy default.
	"$RUN_LIFECYCLE_BIN" --scenario "$package_path" --run-id "${scenario_id}-rep${rep}" \
		--root "$root_key" --scratch-root "$ephemeral" --output "$lifecycle_out" \
		--limits "$OPERATOR_LIMITS_FILE" </dev/null
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

	# UAT-R3-01 (T-E40-F10-004 fix): export the ephemeral scratch project's
	# REAL entity_history before it is rm -rf'd below -- the real producer
	# retain_pair's entity-history.json now requires instead of the
	# hardcoded "" it used to pass. A failure here is a dispatch failure
	# (the same discipline as run-lifecycle.sh/evaluate-lifecycle.sh
	# failures above): this pair is never retained with a fabricated
	# placeholder.
	local entity_history_out="$pair_work/entity-history.json"
	set +e
	"$ENTITY_HISTORY_EXPORT_BIN" --scratch-root "$ephemeral" --root-key "$root_key" --output "$entity_history_out" </dev/null
	local entity_history_rc=$?
	set -e

	if [[ "$entity_history_rc" -ne 0 ]]; then
		echo "run-lifecycle-batch: $scenario_id rep $rep FAILED (export-entity-history.sh exit $entity_history_rc)" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "required_artifact_source_unavailable"
		overall_bad="true"
		rm -rf "$pair_work"
		return 0
	fi

	if ! retain_pair "$scenario_id" "$rep" "$package_path" "$lifecycle_out" "$evaluation_out" "$entity_history_out" "$i05_bundle_dir"; then
		echo "run-lifecycle-batch: $scenario_id rep $rep FAILED (retention refused -- a required artifact had no real source)" >&2
		append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
		record_invalid "$scenario_id" "$rep" "required_artifact_source_unavailable"
		overall_bad="true"
		rm -rf "$pair_work"
		return 0
	fi
	append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "$success_label"
	rm -rf "$pair_work"
}

# pair_provenance_ok <scenario_id> <rep> <dir> -- REQ-NF-007 (append-and-
# verify, never a silent overwrite/reuse): classify_pair's "already
# retained, skip real dispatch" fast path used to trust bare
# evaluation.jsonl presence as proof THIS exact (scenario_id, rep) pair was
# ever retained here at all (code-review-2026-08-21T0459-E40-F10.md round-4
# finding 2, the wider half of the gap: classify_pair had NO provenance
# check of any kind, unlike run-review-comparison.sh's gate_dest_
# provenance_ok, which at least checked the `gate` field). A pre-existing
# symlink or stray directory pointed at some OTHER scenario's own
# legitimately-retained pair would pass a bare presence check and be
# silently adopted as this scenario's own retained rep, with no diagnostic.
# manifest.json's scenario_id/rep fields (lib/retain_pair's own required
# manifest keys, both drivers) are the provenance check.
pair_provenance_ok() {
	local scenario_id="$1" rep="$2" dir="$3"
	[[ -f "$dir/manifest.json" ]] || return 1
	python3 -c '
import json, sys
scenario_id, rep, path = sys.argv[1:4]
try:
    manifest = json.load(open(path))
except Exception:
    sys.exit(1)
if not isinstance(manifest, dict):
    sys.exit(1)
sys.exit(0 if (str(manifest.get("scenario_id", "")) == scenario_id and str(manifest.get("rep", "")) == rep) else 1)
' "$scenario_id" "$rep" "$dir/manifest.json"
}

# pair_retention_verified <scenario_id> <rep> <dir> -- UAT round-6 fix (T-E40-
# F10-004, defect class: "treating a present file, digest field, or non-empty
# provenance string as proof of verified source derivation"). Called ONLY
# after pair_provenance_ok already confirmed manifest.json's own identity
# matches -- this is the completeness half of the reuse-fast-path gap:
# pair_provenance_ok alone never checked that every retention_required_
# artifacts entry was actually present, and never recomputed a single
# digest, so a pair with a superficially valid manifest.json but missing,
# truncated, or tampered artifacts was indistinguishable from a real,
# fully byte-preserved retention. Delegates to the shared lib/verify_pair_
# retention (same "single owner" precedent as lib/retain_pair itself) so
# run-review-comparison.sh's own gate_dest_provenance_ok (T-E40-F10-005)
# can apply the IDENTICAL completeness+digest strengthening rather than a
# second, potentially divergent reimplementation. On refusal, prints the
# `<reason>:<artifact>` token verify_pair_retention writes to its own
# stdout (captured here, NOT printed to this function's stdout, so the
# caller can fold it into classify_pair's own single-line classification)
# and returns nonzero; the human diagnostic already went to stderr from
# verify_pair_retention itself.
pair_retention_verified() {
	local scenario_id="$1" rep="$2" dir="$3"
	set +e
	PAIR_RETENTION_VERIFIED_REASON="$(python3 "$SCRIPT_DIR/lib/verify_pair_retention" "$scenario_id" "$rep" "$dir" "$LIFECYCLE_SCHEMA")"
	local rc=$?
	set -e
	if [[ "$rc" -ne 0 && -z "$PAIR_RETENTION_VERIFIED_REASON" ]]; then
		PAIR_RETENTION_VERIFIED_REASON="verification_failed:unknown"
	fi
	return "$rc"
}

# classify_pair <scenario_id> <rep> -- prints symlinked_dest /
# provenance_mismatch / provenance_incomplete:<reason> / skipped_complete /
# incomplete_prior_attempt / pending, mirroring run-batch.sh classify_pair
# (extended with the two new read-side provenance classifications, round-4
# finding 2; STRUCTURALLY extended round-5, code-review-2026-08-21T1335-
# E40-F10.md finding 1, to walk the whole chain rather than the leaf-level
# directory alone; extended round-6, UAT-2026-08-21T233606Z, with
# provenance_incomplete -- an identity-matching manifest.json is no longer
# sufficient on its own, every required artifact must be present with a
# matching digest).
classify_pair() {
	local scenario_id="$1" rep="$2"
	local dir="$out_root_canon/scenarios/$scenario_id/$rep"
	# Symlink anywhere from out_root_canon down to $dir, inclusive (round-4
	# finding 2 closed the LEAF case; round-5 finding 1 found the identical
	# gap one level up: a symlink at scenarios/$scenario_id itself -- fully
	# inside out_root_canon, so assert_within_out_root's containment check
	# alone does not catch it -- silently aliases this scenario's retained
	# pair to whatever the symlink resolves to. The pre-round-5 checks below
	# tested artifact PRESENCE through `dir` before retain_pair()'s own -L
	# guard (reached only on the "not yet retained, must create" branch)
	# ever ran, and that guard itself only ever inspected the LEAF. Refuse
	# to classify ANY chain-compromised path as complete (or as a normal
	# incomplete/pending directory) before ever trusting its presence.
	if ! assert_no_symlink_in_chain "$out_root_canon" "$dir"; then
		echo "symlinked_dest"
		return 0
	fi
	if [[ -f "$dir/evaluation.jsonl" ]]; then
		if ! pair_provenance_ok "$scenario_id" "$rep" "$dir"; then
			echo "provenance_mismatch"
			return 0
		fi
		# UAT round-6 fix: identity-matching manifest.json is necessary but
		# not sufficient -- verify every required artifact is present with a
		# manifest-recorded source_path AND a digest that recomputes to the
		# manifest-recorded value before ever trusting this pair as complete.
		if ! pair_retention_verified "$scenario_id" "$rep" "$dir"; then
			echo "provenance_incomplete:$PAIR_RETENTION_VERIFIED_REASON"
			return 0
		fi
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
	local incomplete_top="$out_root_canon/.incomplete"
	local incomplete_root="$incomplete_top/$scenario_id"
	assert_within_out_root "$dir" || return 1
	assert_within_out_root "$incomplete_root" || return 1
	# Symlink-anywhere-in-chain (round-4 sweep, code-review-2026-08-21T0459-
	# E40-F10.md; STRUCTURALLY extended round-5, code-review-2026-08-21T1335-
	# E40-F10.md finding 1, to walk the full chain rather than two named leaf
	# components): `mkdir -p` treats a pre-existing symlink-to-directory at
	# ANY path component as "already there, fine" and silently no-ops
	# against it, exactly the same defect class as retain_pair()/
	# retain_gate()'s own `dest` guard -- the subsequent `mv "$dir" "$dest"`
	# below would then relocate a quarantined prior attempt INTO whatever
	# the symlink resolves to (still in-root, so assert_within_out_root
	# alone does not catch it) instead of this scenario's own `.incomplete/`
	# area. A single chain walk from out_root_canon down to incomplete_root
	# structurally covers every component `mkdir -p` would otherwise walk
	# past (.incomplete, .incomplete/<scenario_id>, and any future nesting),
	# not just the two components named by round 4's own fix. Also walked
	# on `dir` itself (the source directory about to be moved): classify_pair
	# already chain-checks this same path before quarantine_pair is ever
	# reached, but re-checking here removes any implicit dependency on
	# caller ordering.
	if ! assert_no_symlink_in_chain "$out_root_canon" "$dir"; then
		echo "run-lifecycle-batch: $scenario_id rep $rep: refusing to quarantine -- a symlink was found in the path to $dir" >&2
		return 1
	fi
	if ! assert_no_symlink_in_chain "$out_root_canon" "$incomplete_root"; then
		echo "run-lifecycle-batch: $scenario_id rep $rep: refusing to quarantine -- a pre-existing symlink was found in the path to $incomplete_root" >&2
		return 1
	fi
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
		symlinked_dest)
			# round-5 finding 1: $dir itself is not necessarily the literal
			# symlink -- a symlink at any ANCESTOR component (e.g.
			# scenarios/$scenario_id itself) between out_root_canon and $dir
			# also lands here now that classify_pair() walks the whole
			# chain. classify_pair()'s own assert_no_symlink_in_chain call
			# already printed the exact symlinked path component above this
			# line; this message stays accurate for both the leaf and
			# ancestor shapes by not claiming $dir itself is the symlink.
			echo "run-lifecycle-batch: $scenario_id rep $rep: a symlink was found in the path to $dir; refusing to treat as complete or dispatch into it" >&2
			append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
			record_invalid "$scenario_id" "$rep" "pre_existing_symlink_at_rep_directory"
			overall_bad="true"
			;;
		provenance_mismatch)
			echo "run-lifecycle-batch: $scenario_id rep $rep: retention path $dir is already occupied by a retained pair whose manifest.json scenario_id/rep does not match this pair -- refusing to reuse or overwrite" >&2
			append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
			record_invalid "$scenario_id" "$rep" "provenance_mismatch"
			overall_bad="true"
			;;
		provenance_incomplete:*)
			# UAT round-6 fix (defect class: "treating a present file, digest
			# field, or non-empty provenance string as proof of verified
			# source derivation"): manifest.json's own identity matched
			# (scenario_id/rep), but lib/verify_pair_retention found the
			# retained pair itself is not actually complete/byte-identical --
			# a required artifact is absent, has no recorded source_path, or
			# its recomputed digest disagrees with the manifest-recorded
			# value. Never silently treated as complete: fails closed with
			# the named artifact:reason, same discipline as
			# provenance_mismatch above (an operator must inspect and
			# --reclaim-incomplete or otherwise repair the retention path;
			# this driver never overwrites a pre-existing, occupied
			# destination on its own).
			pir_reason="${classification#provenance_incomplete:}"
			echo "run-lifecycle-batch: $scenario_id rep $rep: retention path $dir is already occupied but failed verification ($pir_reason) -- refusing to reuse or overwrite" >&2
			append_summary "$scenario_id" "$scenario_version" "$family" "$rep" "failed"
			record_invalid "$scenario_id" "$rep" "provenance_incomplete:$pir_reason"
			overall_bad="true"
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
import os
import sys
import tempfile
import time
import uuid

import yaml

summary_path, invalid_path, mode, retention_root, batch_policy_path, reclaim, argv_joined = sys.argv[1:8]


# ---------------------------------------------------------------------------
# Top-level retention-root writes (code-review-2026-08-21T0459-E40-F10.md
# round-4 finding 3): batch.json and invalid/index.jsonl used to be plain
# `open(path, "w")` (which follows a pre-existing FILE symlink at that exact
# path and truncates whatever it points at) and `os.makedirs(...,
# exist_ok=True)` (which treats a pre-existing symlink-to-directory as
# "already there, fine" and lets the subsequent open() write through it).
# This is the SAME defect class lib/retain_pair's install_atomic_file
# closed for every per-pair artifact -- these two writers just sit outside
# that shared script entirely (they are this driver's own top-level
# aggregate outputs, not a (scenario, rep) pair artifact), so they were
# never brought under the sweep despite living in the same file. Reusing
# the identical tempfile-sibling + os.replace() primitive here (never a
# second, differently-behaved mechanism) means a pre-existing symlink at
# either destination is REPLACED atomically as a directory entry, never
# opened for writing/followed, matching lib/retain_pair's own
# install_atomic_file (files) and the loud-refusal-on-directory-shaped-
# collision posture for `invalid/` (directories) below.
def install_atomic_file(target_path: str, data: bytes):
    parent = os.path.dirname(target_path) or "."
    os.makedirs(parent, exist_ok=True)
    fd, temp_name = tempfile.mkstemp(prefix=f".{os.path.basename(target_path)}.", dir=parent)
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temp_name, target_path)
    except OSError:
        try:
            os.unlink(temp_name)
        except OSError:
            pass
        raise


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
if isinstance(min_reps_raw, bool) or not isinstance(min_reps_raw, (int, float, str)):
    print(f"run-lifecycle-batch: batch policy min_reps must be a positive integer, got {min_reps_raw!r}", file=sys.stderr)
    raise SystemExit(1)
if isinstance(min_reps_raw, float) and not min_reps_raw.is_integer():
    print(f"run-lifecycle-batch: batch policy min_reps must be a positive integer, got {min_reps_raw!r}", file=sys.stderr)
    raise SystemExit(1)
try:
    min_reps = int(min_reps_raw)
    if min_reps < 1:
        raise ValueError
except (TypeError, ValueError):
    print(f"run-lifecycle-batch: batch policy min_reps must be a positive integer, got {min_reps_raw!r}", file=sys.stderr)
    raise SystemExit(1)

batch = {
    "phase": "lifecycle_v2",
    "batch_id": f"batch-{uuid.uuid4().hex}",
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
batch_json_bytes = (json.dumps(batch, indent=2, sort_keys=True) + "\n").encode("utf-8")
# No ancestor-chain check needed here (round-5 sweep, code-review-
# 2026-08-21T1335-E40-F10.md finding 1): batch.json/invalid/ are direct,
# single-level children of retention_root itself (the SAME trusted,
# already-canonicalized anchor out_root_canon establishes) -- there is no
# intermediate ancestor component between root and these two leaves for a
# chain walk to add coverage for. install_atomic_file's tempfile+os.replace
# already handles a pre-existing symlink AT batch.json itself (replaced
# atomically, matching lib/retain_pair's own file-artifact convention); the
# os.path.islink check below does the same for invalid/ as a directory
# (refused, matching lib/retain_pair's directory-artifact convention).
install_atomic_file(f"{retention_root}/batch.json", batch_json_bytes)

invalid_dir = f"{retention_root}/invalid"
if os.path.islink(invalid_dir):
    print(
        f"run-lifecycle-batch: refusing to write invalid/index.jsonl -- {invalid_dir} is a "
        "pre-existing symlink, not a real directory",
        file=sys.stderr,
    )
    raise SystemExit(1)
os.makedirs(invalid_dir, exist_ok=True)
invalid_jsonl_bytes = "".join(json.dumps(row, sort_keys=True) + "\n" for row in invalid).encode("utf-8")
install_atomic_file(f"{invalid_dir}/index.jsonl", invalid_jsonl_bytes)

print(json.dumps(batch, indent=2, sort_keys=True))
PYEOF

if [[ "$overall_bad" == "true" ]]; then
	exit 4
fi
exit 0
