#!/usr/bin/env bash
# run-review-comparison.sh --candidate <candidate.yaml> --retention-root <root>
#                           --mode preview|pilot|baseline
#                           --comparison-mode independent_frozen_candidate|sequential_delivery
#                           [--acknowledge-provider-spend]
#                           [--max-cost-usd <n>] [--max-wall-clock-seconds <n>]
#                           [--max-generated-tasks <n>]
#
# T-E40-F10-005 (spec.md REQ-F-014, REQ-F-015, ADR-F10-04): operator
# review-comparison driver. Drives the feature-QA gate and the
# finish-feature deep-review gate for one declared candidate through the
# existing run-lifecycle.sh/evaluate-lifecycle.sh pair -- one dispatch per
# gate, never more -- then delegates the paired verdict entirely to the
# existing bench/scripts/compare-lifecycle-evaluations.sh (unchanged
# --left/--right/--mode/--output signature) and retains the accepted or
# rejected comparison record unchanged.
#
# --- candidate.yaml (this script's own input contract; not an upstream
# I-## shape) ---
#   schema_version: "1.0"
#   scenario_id: <admitted I-04 scenario id, shared by both gates -- the
#                 same fixture/candidate reviewed at two workflow points>
#   scenario_index: <path>          # optional, default
#                                    # bench/scenarios/scenarios.yaml
#   gates:
#     qa:
#       root_key: "<entity key>"    # existing entity key inside
#                                    # scratch_root (run-lifecycle.sh's
#                                    # --root contract, see
#                                    # run-lifecycle-batch.sh's Notes)
#       scratch_root: "<dir>"       # pre-built scratch Shark project
#                                    # template. Never mutated: every
#                                    # dispatch below copies it into a
#                                    # fresh ephemeral directory first.
#       i05_bundle_dir: "<dir>"     # optional; the I-05 stage-evidence
#                                    # bundle evaluate-lifecycle.sh reads
#                                    # via --i05, and the presence-only
#                                    # signal this driver reads for
#                                    # "truth-set availability" in preview.
#     deep_review:
#       root_key: "<entity key>"
#       scratch_root: "<dir>"
#       i05_bundle_dir: "<dir>"
#
# root_key/scratch_root/i05_bundle_dir are operator-prepared inputs, not
# something this driver bootstraps (REQ-NF-001, matching
# run-lifecycle-batch.sh's identical Notes-for-Agent boundary).
#
# --- Identity boundary (REQ-F-015, ADR-F10-04, AC-T3) ---
# This file performs NO field-by-field candidate or workflow-policy
# identity comparison of its own, and never re-derives or assembles a
# candidate_snapshots lineage: whatever a gate's own
# evaluate-lifecycle.sh run produces (one snapshot for a frozen candidate,
# several for a scenario with intervening fix rounds) reaches the
# comparator completely unmodified. The comparator's own MODES,
# candidate-identity field set, and workflow-policy digest field set
# remain the single identity authority; this file must never spell out
# that field-name vocabulary anywhere (including in comments) so a static
# scan for it (T-E40-F10-014) stays a meaningful signal. This driver never
# accepts a branch name or a bare `HEAD` as candidate identity itself --
# that rejection is the comparator's malformed-identity check, delegated,
# never special-cased here.
#
# Exit status (schema-owned, bench/reports/lifecycle-baseline-schema.yaml
# refusal_exit_status): 0 success (preview, or an accepted+published
# comparison), 2 usage error (including a comparator-reported invalid
# input), 3 spend-gate refusal (ADR-F10-02/03, before any subprocess), 4
# the comparator ran and rejected the pair -- never published
# (batch_partial_failure: "ran, outcome not publishable").
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# TD-077 defect-class precedent (run-batch.sh's own header comment, reused
# by run-lifecycle-batch.sh): resolved via the SIBLING path, never a bare
# PATH-resolved name, so a self-test can substitute a stub through one
# overridable variable without touching PATH resolution for anything else
# this script calls.
RUN_LIFECYCLE_BIN="${RUN_LIFECYCLE_BIN:-$SCRIPT_DIR/run-lifecycle.sh}"
EVALUATE_LIFECYCLE_BIN="${EVALUATE_LIFECYCLE_BIN:-$SCRIPT_DIR/evaluate-lifecycle.sh}"
COMPARATOR_BIN="${COMPARATOR_BIN:-$SCRIPT_DIR/compare-lifecycle-evaluations.sh}"
# UAT-R3-01 fix (T-E40-F10-005): the real producer for retain_pair's
# entity-history.json artifact -- see export-entity-history.sh's own header.
ENTITY_HISTORY_EXPORT_BIN="${ENTITY_HISTORY_EXPORT_BIN:-$SCRIPT_DIR/export-entity-history.sh}"
# UAT round-6 fix (T-E40-F10-005, coordinated with T-E40-F10-004's
# identical addition to run-lifecycle-batch.sh): the schema driving
# lib/verify_pair_retention's own retention_required_artifacts
# enumeration -- same override convention aggregate-lifecycle.sh's own
# LIFECYCLE_SCHEMA already establishes, never a private hardcoded copy of
# the schema path.
LIFECYCLE_SCHEMA="${LIFECYCLE_SCHEMA:-$BENCH_DIR/reports/lifecycle-baseline-schema.yaml}"

usage() {
	cat >&2 <<'EOF'
usage: run-review-comparison.sh --candidate <candidate.yaml> --retention-root <root>
                                 --mode preview|pilot|baseline
                                 --comparison-mode independent_frozen_candidate|sequential_delivery
                                 [--acknowledge-provider-spend]
                                 [--max-cost-usd <n>]
                                 [--max-wall-clock-seconds <n>]
                                 [--max-generated-tasks <n>]
EOF
	exit 2
}

# Captured before any parsing: spend-gate.sh's AC-T2 contract requires the
# driver's *raw, unmodified* argv, so this array is built first and never
# mutated afterward.
ORIGINAL_ARGV=("$@")

candidate_decl=""
retention_root=""
mode=""
comparison_mode=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--candidate)
		[[ $# -ge 2 ]] || usage
		candidate_decl="$2"
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
	--comparison-mode)
		[[ $# -ge 2 ]] || usage
		comparison_mode="$2"
		shift 2
		;;
	--acknowledge-provider-spend)
		shift
		;;
	--max-cost-usd | --max-wall-clock-seconds | --max-generated-tasks)
		[[ $# -ge 2 ]] || usage
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

[[ -n "$candidate_decl" ]] || usage
[[ -f "$candidate_decl" ]] || {
	echo "run-review-comparison: candidate declaration not found: $candidate_decl" >&2
	exit 2
}
case "$mode" in
preview | pilot | baseline) ;;
*) usage ;;
esac
case "$comparison_mode" in
independent_frozen_candidate | sequential_delivery) ;;
*) usage ;;
esac
# --retention-root is a hard usage error only for --mode preview (mirrors
# run-lifecycle-batch.sh exactly: for --mode pilot|baseline its absence
# MUST be a spend-gate REFUSAL, exit 3, reason missing_retention_root, not
# a usage error -- so it is deliberately NOT checked here for those modes;
# spend_gate_check_all below is the single owner of that condition).
if [[ "$mode" == "preview" ]]; then
	[[ -n "$retention_root" ]] || usage
fi

command -v python3 >/dev/null 2>&1 || {
	echo "run-review-comparison: python3 not found on PATH" >&2
	exit 2
}

# ---------------------------------------------------------------------------
# ADR-F10-02/03: for provider-backed modes, spend_gate_check_all is called
# with the driver's untouched raw argv BEFORE anything below this point --
# no candidate-declaration parse, no retention-root creation, no gate
# dispatch. A refusal therefore costs zero subprocesses (REQ-F-002,
# REQ-F-003).
#
# UAT round-6 fix (uat-2026-08-21T233606Z-E40-F10.md, HIGH finding
# "comparison baseline can proceed without a verified family attestation"):
# this driver never passes --batch (it has no such flag; --candidate is its
# only matrix-shape declaration) -- ORIGINAL_ARGV therefore always carries
# --candidate. spend_gate_check_pilot_ledger_families (lib/spend-gate.sh,
# T-E40-F10-003's fix) now derives the requested family set from EITHER
# --batch OR --candidate and runs the real per-family pilot-ledger.sh
# --verify check unconditionally for both real driver shapes -- there is no
# separate --batch flag for this driver to add, and adding a redundant one
# would be meaningless since this file never enumerates a batch matrix.
# Sweep result: spend_gate_check_all is called exactly once, here, with the
# full untouched ORIGINAL_ARGV (including --candidate); no other call site
# in this file invokes any spend-gate function.
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
	# --limits policy (the sole execution-time enforcer/recorder). Without
	# this, both gate dispatches below fell through to the scenario
	# package's resource_policy default instead, making the acknowledgement
	# gate above pure theater.
	#
	# Re-reads ORIGINAL_ARGV via spend-gate.sh's OWN _spend_gate_flag_value
	# accessor -- the exact function spend_gate_check_ceiling just used to
	# validate these three flags -- rather than a second, independent argv
	# parser in this file's own arg-parsing loop above. A second parser with
	# different first-vs-last-duplicate-flag semantics would silently
	# reintroduce this same defect class (validated value != materialized
	# value) the moment an operator's argv repeats a ceiling flag; calling
	# the identical function against the identical array is closed by
	# construction instead. One file, materialized once per invocation (the
	# ceilings are constant across both gates of one candidate) and reused
	# by every dispatch_gate() call -- never a second, divergent per-gate
	# policy.
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
		echo "run-review-comparison: internal error: spend gate passed but a ceiling did not resolve from ORIGINAL_ARGV" >&2
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

# Batch and comparison share the scenarios/<id>/<rep> publication namespace;
# use the same root-scoped single-writer lock so the two producers cannot
# classify or publish the same pair concurrently. Recover only a lock whose
# recorded owner is no longer alive.
LOCK_DIR="$out_root_canon/.lifecycle-batch.lock"
if [[ "$mode" != "preview" ]]; then
	if ! mkdir "$LOCK_DIR"; then
		owner=""
		[[ -f "$LOCK_DIR/pid" ]] && owner="$(<"$LOCK_DIR/pid")"
		if [[ "$owner" =~ ^[0-9]+$ ]] && kill -0 "$owner"; then
			echo "run-review-comparison: retention root is already locked by process $owner: $out_root_canon" >&2
			exit 4
		fi
		if [[ ! "$owner" =~ ^[0-9]+$ ]]; then
			echo "run-review-comparison: retention root lock has no valid owner: $out_root_canon" >&2
			exit 4
		fi
		rm -f "$LOCK_DIR/pid"
		if ! rmdir "$LOCK_DIR" || ! mkdir "$LOCK_DIR"; then
			echo "run-review-comparison: retention root lock could not be recovered: $out_root_canon" >&2
			exit 4
		fi
	fi
	printf '%s\n' "$$" >"$LOCK_DIR/pid"
fi

# ---------------------------------------------------------------------------
# Parse candidate.yaml -> exactly two JSON rows, one per gate (qa,
# deep_review). Pure computation: reads the candidate declaration and the
# I-04 scenario index only -- no subprocess, no Shark call.
# ---------------------------------------------------------------------------
CANDIDATE_TMP="$(mktemp)"
cleanup_candidate() {
	rm -f "$CANDIDATE_TMP" "$OPERATOR_LIMITS_FILE"
	if [[ "$mode" != "preview" ]]; then
		rm -f "$LOCK_DIR/pid"
		rmdir "$LOCK_DIR" 2>/dev/null || true
	fi
}
trap cleanup_candidate EXIT

set +e
python3 - "$BENCH_DIR" "$candidate_decl" >"$CANDIDATE_TMP" <<'PYEOF'
import json
import os
import re
import sys

import yaml

bench_dir, candidate_path = sys.argv[1:3]

with open(candidate_path) as f:
    decl = yaml.safe_load(f) or {}
if not isinstance(decl, dict):
    print(f"run-review-comparison: candidate declaration must be a YAML mapping: {candidate_path}", file=sys.stderr)
    raise SystemExit(1)

scenario_id = str(decl.get("scenario_id", ""))
if not scenario_id:
    print("run-review-comparison: candidate declaration missing scenario_id", file=sys.stderr)
    raise SystemExit(1)
if not re.fullmatch(r"[a-z0-9]+(-[a-z0-9]+)*", scenario_id):
    print(f"run-review-comparison: candidate declaration has unsafe scenario_id: {scenario_id!r}", file=sys.stderr)
    raise SystemExit(1)

scenario_index_field = decl.get("scenario_index") or "scenarios/scenarios.yaml"
scenario_index_path = scenario_index_field if os.path.isabs(scenario_index_field) else os.path.join(bench_dir, scenario_index_field)
if not os.path.isfile(scenario_index_path):
    print(f"run-review-comparison: scenario index not found: {scenario_index_path}", file=sys.stderr)
    raise SystemExit(1)

with open(scenario_index_path) as f:
    index = yaml.safe_load(f) or {}
index_dir = os.path.dirname(scenario_index_path)

package_path = None
package = None
for rel in index.get("scenarios") or []:
    pkg_path = os.path.join(index_dir, rel, "package.yaml")
    if not os.path.isfile(pkg_path):
        continue
    with open(pkg_path) as f:
        pkg = yaml.safe_load(f) or {}
    if str(pkg.get("scenario_id", "")) == scenario_id:
        package_path = os.path.abspath(pkg_path)
        package = pkg
        break
if package_path is None:
    print(f"run-review-comparison: scenario id '{scenario_id}' did not resolve against the scenario index", file=sys.stderr)
    raise SystemExit(1)

gates_decl = decl.get("gates") or {}
if not isinstance(gates_decl, dict) or "qa" not in gates_decl or "deep_review" not in gates_decl:
    print("run-review-comparison: candidate declaration must declare both gates.qa and gates.deep_review", file=sys.stderr)
    raise SystemExit(1)

stage_matrix = package.get("stage_matrix") or {}
prelude = stage_matrix.get("prelude") or {}
lifecycle_mode = (stage_matrix.get("lifecycle") or {}).get("mode", "unknown")
family = str(package.get("entity_family", "unknown"))
scenario_version = str(package.get("scenario_version", "1"))

for gate_name in ("qa", "deep_review"):
    entry = gates_decl.get(gate_name) or {}
    if not isinstance(entry, dict):
        print(f"run-review-comparison: candidate declaration gates.{gate_name} must be a mapping", file=sys.stderr)
        raise SystemExit(1)
    root_key = str(entry.get("root_key") or "")
    scratch_root = str(entry.get("scratch_root") or "")
    if scratch_root and not os.path.isabs(scratch_root):
        scratch_root = os.path.join(bench_dir, scratch_root)
    i05_bundle_dir = str(entry.get("i05_bundle_dir") or "")
    if i05_bundle_dir and not os.path.isabs(i05_bundle_dir):
        i05_bundle_dir = os.path.join(bench_dir, i05_bundle_dir)
    row = {
        "gate": gate_name,
        "scenario_id": scenario_id,
        "scenario_version": scenario_version,
        "family": family,
        "root_key": root_key,
        "scratch_root": scratch_root,
        "i05_bundle_dir": i05_bundle_dir,
        "package_path": package_path,
        "prelude": prelude,
        "lifecycle_mode": lifecycle_mode,
    }
    print(json.dumps(row, sort_keys=True))
PYEOF
parse_rc=$?
set -e
if [[ "$parse_rc" -ne 0 ]]; then
	exit 2
fi

readarray -t GATE_ROWS <"$CANDIDATE_TMP"
[[ "${#GATE_ROWS[@]}" -eq 2 ]] || {
	echo "run-review-comparison: candidate declaration did not resolve exactly 2 gate rows (qa, deep_review)" >&2
	exit 2
}

row_field() {
	# row_field <row_json> <field>
	python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get(sys.argv[2], ""))' "$1" "$2"
}

# ---------------------------------------------------------------------------
# PREVIEW (ADR-F10-01-style, REQ-F-001/F-014, REQ-NF-002): calls
# run-lifecycle.sh --mode dry-run once per gate for real (zero-spend:
# candidate/workflow-policy identity is derived from git state, not a
# provider call) against an ephemeral copy of the gate's declared
# scratch_root template, and dispatches nothing else. Prints the raw
# identity/candidate/workflow-policy objects wholesale (never naming an
# individual digest field) so the reader sees everything the comparator
# will later see, without this file re-deriving or narrating any of it.
# ---------------------------------------------------------------------------
if [[ "$mode" == "preview" ]]; then
	PREVIEW_WORK="$(mktemp -d)"
	trap 'cleanup_candidate; rm -rf "$PREVIEW_WORK"' EXIT

	set +e
	python3 - "$RUN_LIFECYCLE_BIN" "$CANDIDATE_TMP" "$out_root_canon" "$PREVIEW_WORK" "$comparison_mode" \
		"${ORIGINAL_ARGV[*]}" <<'PYEOF'
import json
import os
import re
import shutil
import subprocess
import sys

run_lifecycle_bin, matrix_path, retention_root, work_dir, comparison_mode, argv_joined = sys.argv[1:7]


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

print("=== run-review-comparison: operator preview (zero provider calls) ===")
print(f"retention root: {retention_root}")
print(f"comparison mode: {comparison_mode}")
print("ceilings:")
for name, value in ceilings.items():
    print(f"  {name}: {value if value is not None else 'not declared'}")

for row in rows:
    print(f"gate: {row['gate']}")
    print(f"  scenario_id={row['scenario_id']} scenario_version={row['scenario_version']} family={row['family']}")

    prelude = row["prelude"] or {}
    print("  expected provider-call inventory (I-04 prelude D01-D05):")
    if prelude:
        for stage_id in sorted(prelude):
            entry = prelude[stage_id] or {}
            applicable = bool(entry.get("applicable"))
            reason = entry.get("reason", "")
            call_line = "1 planned provider call" if applicable else "replayed (0 planned provider calls)"
            print(f"    {stage_id}: applicable={applicable} -> {call_line} ({reason})")
    else:
        print("    (no prelude entries in scenario package)")
    print(f"  lifecycle stage_matrix.mode: {row['lifecycle_mode']}")

    i05_bundle_dir = row["i05_bundle_dir"]
    truth_set_available = bool(i05_bundle_dir) and os.path.isdir(i05_bundle_dir) and any(os.scandir(i05_bundle_dir))
    print(f"  truth-set availability: {truth_set_available}")

    root_key = row["root_key"]
    scratch_root = row["scratch_root"]
    if not root_key or not scratch_root:
        print("    identity resolution: skipped (root_key/scratch_root not configured for this gate)")
        continue
    if not os.path.isdir(scratch_root):
        print(f"    identity resolution: skipped (configured scratch_root does not exist: {scratch_root})")
        continue

    ephemeral = os.path.join(work_dir, re.sub(r"[^A-Za-z0-9._-]", "_", row["gate"]))
    # No source-side symlink guard needed here (round-5 sweep,
    # code-review-2026-08-21T1335-E40-F10.md finding 2): unlike dispatch_gate's
    # `cp -a`, shutil.copytree does NOT preserve a top-level symlink source --
    # it lists the resolved directory's entries via os.scandir and creates a
    # genuinely new, independent `ephemeral` directory (empirically verified
    # for this fix: a symlinked scratch_root here produces a real directory
    # at `ephemeral`, never a symlink to the template). Different copy
    # primitive, not vulnerable to the finding-2 defect class.
    shutil.copytree(scratch_root, ephemeral)
    output_path = os.path.join(work_dir, f"{row['gate']}-preview-lifecycle.jsonl")

    process = subprocess.run(
        [
            run_lifecycle_bin,
            "--scenario", row["package_path"],
            "--run-id", f"preview-{row['gate']}",
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
        print(f"    identity resolution: FAILED (run-lifecycle.sh --mode dry-run exited {process.returncode}: {detail})")
        continue

    with open(output_path) as f:
        record = json.loads(f.readline())
    identity = record.get("identity") or {}
    policy = record.get("workflow_policy") or {}
    stages = record.get("stages") or []
    candidate = stages[0].get("candidate") if stages and isinstance(stages[0], dict) else None

    print("    candidate identity (from the real dry-run record, zero-spend):")
    print("      identity: " + json.dumps(identity, sort_keys=True))
    if candidate is not None:
        print("      candidate: " + json.dumps(candidate, sort_keys=True))
    else:
        print("      candidate: (no dispatch resolved during dry-run)")
    print("    workflow-policy identity: " + json.dumps(policy, sort_keys=True))
    print(f"    fix rules: fixes_allowed_between_gates={policy.get('fixes_allowed_between_gates')}")

print("=== end preview: zero provider or network calls made ===")
PYEOF
	preview_rc=$?
	set -e
	exit "$preview_rc"
fi

# ---------------------------------------------------------------------------
# PILOT / BASELINE dispatch. One dispatch per gate, never more
# (run-lifecycle-batch.sh's dispatch_pair/skipped_complete discipline
# reused): if a gate's evaluation.jsonl is already retained under the
# root, dispatch is skipped entirely and the already-retained record feeds
# the comparator directly -- this is also what lets an operator re-run a
# comparison after only one gate's evidence changed without re-spending
# the other gate.
#
# Retention layout (code-review-2026-08-20T2138-E40-F10.md finding 2): a
# gate's retained output is a FULL (scenario, rep) pair --
# `scenarios/<scenario_id>/<rep>/` with all eight artifacts -- the SAME
# shape and the SAME shared manifest builder (`lib/retain_pair`)
# run-lifecycle-batch.sh's retain_pair() uses, per spec.md's Data model
# table. `scenario_id` is the real, shared I-04 scenario id (candidate.yaml
# declares it explicitly shared by both gates) -- never a synthesized id.
# `rep` is a fixed, documented per-gate mapping (gate_rep() below): the two
# gates of one candidate are not repeated identical runs, they are two
# distinct retained pairs of the SAME scenario_id, and REQ-F-004 already
# treats "for each (scenario, rep): ... any review-comparison record" as
# one retention model shared by both drivers.
#
# Rep-namespace collision (code-review-2026-08-21T0330-E40-F10.md finding
# 1): this driver and run-lifecycle-batch.sh both write into the SAME
# `scenarios/<scenario_id>/<rep>/` namespace for a scenario_id they
# legitimately share, but each allocates `rep` independently.
# run-lifecycle-batch.sh allocates sequentially from rep 1
# (`for ((rep = 1; rep <= reps; rep++))`, run-lifecycle-batch.sh:683) with
# no configured upper bound. gate_rep() previously hardcoded qa=rep 1,
# deep_review=rep 2 -- inside that exact low range -- so whichever driver
# wrote a given (scenario_id, rep) slot SECOND collided with the other:
# a gate dispatched before a batch run silently satisfied the batch's
# classify_pair() "skipped_complete" check for a pair the batch itself
# never produced, with no diagnostic (the confirmed-live-repro direction;
# the reverse order was already caught loudly by
# gate_dest_provenance_ok() below). GATE_REP_BASE reserves a numbering
# band no realistic `--reps` value reaches, so the two producers' rep
# allocations structurally cannot collide regardless of dispatch order --
# the narrowest fix, entirely on this driver's side, with no change to
# run-lifecycle-batch.sh's classify_pair() or aggregate-lifecycle.sh's pair
# enumeration required.
#
# OPEN QUESTION (not resolved here, flagged for spec.md/schema): whether a
# gate-produced rep SHOULD count toward a scenario's ordinary
# quality/time/cost/noise-band aggregate statistics at all, since it is a
# different kind of artifact (a review-comparison gate run, not an
# ordinary pilot/baseline rep) than a real batch rep. REQ-F-016's "three
# separate dimensions, no blending" spirit argues it should not; nothing in
# spec.md says so explicitly today, and this fix deliberately leaves that
# question open rather than silently deciding it as a side effect.
# ---------------------------------------------------------------------------
GATE_REP_BASE=900000

gate_rep() {
	# gate_rep <gate> -- fixed, documented mapping within the reserved
	# GATE_REP_BASE band (never operator-supplied, never in the low
	# sequential range run-lifecycle-batch.sh allocates from): qa=BASE+1
	# (feature-QA gate, REQ-F-014's "left" side), deep_review=BASE+2
	# (finish-feature deep-review gate, the "right"/terminal side).
	case "$1" in
	qa) echo "$((GATE_REP_BASE + 1))" ;;
	deep_review) echo "$((GATE_REP_BASE + 2))" ;;
	*)
		echo "run-review-comparison: internal error: unknown gate '$1'" >&2
		return 1
		;;
	esac
}

retain_gate() {
	# retain_gate <scenario_id> <rep> <gate> <package_path> <lifecycle_jsonl> <evaluation_jsonl> <entity_history_json> <i05_bundle_dir>
	# UAT-R3-01 (round 3): lib/retain_pair now refuses (nonzero exit, no
	# manifest.json written) rather than fabricating a placeholder when any
	# required artifact's source is absent -- dispatch_gate below checks
	# this function's return status.
	local scenario_id="$1" rep="$2" gate="$3" package_path="$4" lifecycle_jsonl="$5" evaluation_jsonl="$6" entity_history_json="$7" i05_bundle_dir="$8"
	local dest="$out_root_canon/scenarios/$scenario_id/$rep"
	assert_within_out_root "$dest" || return 1
	# Symlink write-through (code-review-2026-08-21T0330-E40-F10.md finding
	# 2; STRUCTURALLY closed round-5, code-review-2026-08-21T1335-E40-F10.md
	# finding 1): assert_within_out_root resolves an existing symlink at
	# `dest` and refuses it if it escapes out_root_canon, but a symlink
	# redirected to ANOTHER retained pair's directory INSIDE the same root
	# -- whether AT dest itself or at an ANCESTOR of dest (e.g.
	# scenarios/$scenario_id itself) -- still passes containment, and
	# mkdir -p would then silently no-op against it. Walk the WHOLE chain
	# from out_root_canon down to dest, not just dest's own leaf -L status,
	# so an ancestor-level redirect can never slip past this guard the way
	# it did round 5's leaf-only symlink_dest() check.
	if ! assert_no_symlink_in_chain "$out_root_canon" "$dest"; then
		echo "run-review-comparison: $gate gate: refusing to retain -- a symlink was found in the path to $dest" >&2
		return 1
	fi
	mkdir -p "$dest"
	python3 "$SCRIPT_DIR/lib/retain_pair" "$scenario_id" "$rep" "$package_path" "$lifecycle_jsonl" "$evaluation_jsonl" "$entity_history_json" "$i05_bundle_dir" "$dest" "$gate"
}

# Set by gate_dest_provenance_ok on a completeness/digest refusal only (the
# `<reason>:<artifact>` token lib/verify_pair_retention writes to its own
# stdout); left unset/empty for an identity-mismatch refusal (the
# gate/scenario_id/rep check below), which already has its own named
# diagnostic at the call site.
GATE_DEST_PROVENANCE_REASON=""

gate_dest_provenance_ok() {
	# gate_dest_provenance_ok <dest> <gate> <scenario_id> <rep> -- REQ-NF-007
	# (append-and-verify, never a silent overwrite/reuse): a bare
	# "evaluation.jsonl exists" check is not proof THIS gate retained it for
	# THIS scenario/rep -- the same (scenario_id, rep) path could be
	# occupied by an unrelated run-lifecycle-batch.sh pair, by the OTHER
	# gate if gate_rep() were ever misconfigured, or (code-review-
	# 2026-08-21T0459-E40-F10.md round-4 finding 2, the wider half of the
	# gap: a `gate`-only check passes a cross-scenario confused-deputy read
	# -- a symlink or stray directory pointed at ANOTHER scenario's own
	# legitimately-retained gate pair with a matching `gate` field but a
	# different scenario_id/rep) by a symlink or stray directory at this
	# exact path. manifest.json's scenario_id/rep/gate fields (all written
	# by lib/retain_pair) are therefore checked together; any mismatch or
	# absence refuses loudly instead of silently adopting someone else's
	# retained evaluation.jsonl as this gate's comparator input.
	#
	# UAT round-6 fix (uat-2026-08-21T233606Z-E40-F10.md, HIGH finding
	# "fabricated or unverifiable provenance can enter pilot approval and
	# aggregates", defect class: "treating a present file, digest field, or
	# non-empty provenance string as proof of verified source derivation"):
	# an identity-matching manifest.json (gate/scenario_id/rep all correct)
	# is necessary but NOT sufficient -- it never checked that every
	# retention_required_artifacts entry was actually present, and never
	# recomputed a single digest, so a pair with a superficially valid
	# manifest.json but missing, truncated, or tampered artifacts was
	# indistinguishable from a real, fully byte-preserved retention. Once
	# the identity check below passes, this delegates to the SAME shared
	# lib/verify_pair_retention T-E40-F10-004's run-lifecycle-batch.sh
	# pair_retention_verified() uses, so both drivers enforce IDENTICAL
	# reuse-completeness semantics rather than two divergent
	# reimplementations of "is this pair actually complete."
	local dest="$1" gate="$2" scenario_id="$3" rep="$4"
	[[ -f "$dest/manifest.json" ]] || return 1
	if ! python3 -c '
import json, sys
gate, scenario_id, rep, path = sys.argv[1:5]
try:
    manifest = json.load(open(path))
except Exception:
    sys.exit(1)
ok = (
    manifest.get("gate", "") == gate
    and str(manifest.get("scenario_id", "")) == scenario_id
    and str(manifest.get("rep", "")) == rep
)
sys.exit(0 if ok else 1)
' "$gate" "$scenario_id" "$rep" "$dest/manifest.json"; then
		return 1
	fi

	GATE_DEST_PROVENANCE_REASON=""
	set +e
	GATE_DEST_PROVENANCE_REASON="$(python3 "$SCRIPT_DIR/lib/verify_pair_retention" "$scenario_id" "$rep" "$dest" "$LIFECYCLE_SCHEMA")"
	local rc=$?
	set -e
	if [[ "$rc" -ne 0 ]]; then
		[[ -n "$GATE_DEST_PROVENANCE_REASON" ]] || GATE_DEST_PROVENANCE_REASON="verification_failed:unknown"
		return 1
	fi
	return 0
}

dispatch_gate() {
	# dispatch_gate <scenario_id> <gate> <package_path> <root_key> <scratch_root> <i05_bundle_dir>
	# Prints the resolved evaluation.jsonl path on success (stdout).
	local scenario_id="$1" gate="$2" package_path="$3" root_key="$4" scratch_root="$5" i05_bundle_dir="$6"
	local rep
	rep="$(gate_rep "$gate")" || return 1
	local dest="$out_root_canon/scenarios/$scenario_id/$rep"

	# Symlink-anywhere-in-chain (code-review-2026-08-21T0459-E40-F10.md
	# round-4 finding 2; STRUCTURALLY extended round-5, code-review-
	# 2026-08-21T1335-E40-F10.md finding 1, to walk the full chain rather
	# than the leaf-level directory alone): the "already retained, skip real
	# dispatch" fast path below used to test artifact PRESENCE
	# (-f "$dest/evaluation.jsonl") through `dest` before this driver's own
	# -L guard (retain_gate(), reached only on the "not yet retained, must
	# create" branch) or the provenance check ever ran. round 5 found the
	# identical gap one directory level up: a symlink at
	# scenarios/$scenario_id itself (fully inside out_root_canon, so
	# assert_within_out_root's containment check alone does not catch it)
	# would silently alias this scenario's gate pair to whatever it resolves
	# to, invisible to a leaf-only -L test since `dest` itself is a real
	# directory entry. Refuse before ever trusting `dest`'s presence, on
	# both branches (already-retained skip AND pending-dispatch), using the
	# same shared chain-walking predicate retain_gate() uses.
	if ! assert_no_symlink_in_chain "$out_root_canon" "$dest"; then
		echo "run-review-comparison: $gate gate: refusing to reuse or dispatch -- a symlink was found in the path to $dest" >&2
		return 1
	fi

	if [[ -f "$dest/evaluation.jsonl" ]]; then
		GATE_DEST_PROVENANCE_REASON=""
		if ! gate_dest_provenance_ok "$dest" "$gate" "$scenario_id" "$rep"; then
			if [[ -n "$GATE_DEST_PROVENANCE_REASON" ]]; then
				# UAT round-6 fix: identity matched (gate/scenario_id/rep),
				# but lib/verify_pair_retention found the retained pair
				# itself is not actually complete/byte-identical -- a
				# required artifact is absent, has no recorded source_path,
				# or its recomputed digest disagrees with the
				# manifest-recorded value. Never silently treated as
				# complete: fails closed with the named artifact:reason,
				# same discipline run-lifecycle-batch.sh's
				# provenance_incomplete classification applies.
				echo "run-review-comparison: $gate gate: retention path $dest is already occupied but failed verification ($GATE_DEST_PROVENANCE_REASON) -- refusing to reuse or overwrite" >&2
			else
				echo "run-review-comparison: $gate gate: retention path $dest is already occupied by a retained pair this driver did not produce for gate '$gate'/scenario '$scenario_id'/rep '$rep' (missing/mismatched manifest.json provenance) -- refusing to reuse or overwrite" >&2
			fi
			return 1
		fi
		echo "run-review-comparison: $gate gate already retained under $dest; skipped_complete" >&2
		printf '%s\n' "$dest/evaluation.jsonl"
		return 0
	fi

	if [[ -z "$root_key" || -z "$scratch_root" ]]; then
		echo "run-review-comparison: $gate gate: root_key/scratch_root not configured" >&2
		return 1
	fi
	if [[ ! -d "$scratch_root" ]]; then
		echo "run-review-comparison: $gate gate: configured scratch_root does not exist: $scratch_root" >&2
		return 1
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
	if ! copy_tree_dereferenced "$scratch_root" "$ephemeral"; then
		echo "run-review-comparison: $gate gate: failed to copy scratch_root into an ephemeral working directory" >&2
		rm -rf "$pair_work"
		return 1
	fi
	local lifecycle_out="$pair_work/lifecycle.jsonl"

	echo "run-review-comparison: dispatching $gate gate" >&2
	set +e
	# UAT-R2-01: --limits carries the operator-acknowledged ceilings
	# (OPERATOR_LIMITS_FILE, materialized above from the same values
	# spend_gate_check_all already validated) through to run-lifecycle.sh's
	# own execution-time enforcement/recording -- never left to fall through
	# to the scenario package's resource_policy default.
	"$RUN_LIFECYCLE_BIN" --scenario "$package_path" --run-id "cmp-${scenario_id}-${gate}" \
		--root "$root_key" --scratch-root "$ephemeral" --output "$lifecycle_out" \
		--limits "$OPERATOR_LIMITS_FILE" </dev/null
	local run_rc=$?
	set -e
	if [[ "$run_rc" -ne 0 ]]; then
		echo "run-review-comparison: $gate gate FAILED (run-lifecycle.sh exit $run_rc)" >&2
		rm -rf "$pair_work"
		return 1
	fi

	if [[ -z "$i05_bundle_dir" ]]; then
		echo "run-review-comparison: $gate gate: i05_bundle_dir not configured; cannot evaluate" >&2
		rm -rf "$pair_work"
		return 1
	fi

	local evaluation_out="$pair_work/evaluation.jsonl"
	set +e
	"$EVALUATE_LIFECYCLE_BIN" --i05 "$i05_bundle_dir" --i07 "$lifecycle_out" \
		--scenario "$package_path" --output "$evaluation_out" </dev/null
	local eval_rc=$?
	set -e
	if [[ "$eval_rc" -ne 0 ]]; then
		echo "run-review-comparison: $gate gate FAILED (evaluate-lifecycle.sh exit $eval_rc)" >&2
		rm -rf "$pair_work"
		return 1
	fi

	# UAT-R3-01 (T-E40-F10-005 fix): export the ephemeral scratch project's
	# REAL entity_history before it is rm -rf'd below -- the real producer
	# retain_pair's entity-history.json now requires instead of the
	# hardcoded "" it used to pass. A failure here refuses the whole gate,
	# same discipline as run-lifecycle.sh/evaluate-lifecycle.sh above.
	local entity_history_out="$pair_work/entity-history.json"
	set +e
	"$ENTITY_HISTORY_EXPORT_BIN" --scratch-root "$ephemeral" --root-key "$root_key" --output "$entity_history_out" </dev/null
	local entity_history_rc=$?
	set -e
	if [[ "$entity_history_rc" -ne 0 ]]; then
		echo "run-review-comparison: $gate gate FAILED (export-entity-history.sh exit $entity_history_rc)" >&2
		rm -rf "$pair_work"
		return 1
	fi

	if ! retain_gate "$scenario_id" "$rep" "$gate" "$package_path" "$lifecycle_out" "$evaluation_out" "$entity_history_out" "$i05_bundle_dir"; then
		rm -rf "$pair_work"
		return 1
	fi
	rm -rf "$pair_work"
	printf '%s\n' "$dest/evaluation.jsonl"
	return 0
}

scenario_id_global=""
left_eval=""
right_eval=""
dispatch_failed="false"

for row_json in "${GATE_ROWS[@]}"; do
	gate="$(row_field "$row_json" gate)"
	scenario_id="$(row_field "$row_json" scenario_id)"
	root_key="$(row_field "$row_json" root_key)"
	scratch_root="$(row_field "$row_json" scratch_root)"
	i05_bundle_dir="$(row_field "$row_json" i05_bundle_dir)"
	package_path="$(row_field "$row_json" package_path)"
	scenario_id_global="$scenario_id"

	rep=""
	if ! rep="$(gate_rep "$gate")"; then
		dispatch_failed="true"
		continue
	fi
	dest="$out_root_canon/scenarios/$scenario_id/$rep"
	if ! assert_within_out_root "$dest"; then
		dispatch_failed="true"
		continue
	fi

	eval_path=""
	if eval_path="$(dispatch_gate "$scenario_id" "$gate" "$package_path" "$root_key" "$scratch_root" "$i05_bundle_dir")"; then
		case "$gate" in
		qa) left_eval="$eval_path" ;;
		deep_review) right_eval="$eval_path" ;;
		esac
	else
		dispatch_failed="true"
	fi
done

if [[ "$dispatch_failed" == "true" || -z "$left_eval" || -z "$right_eval" ]]; then
	echo "run-review-comparison: one or both gates failed to dispatch/evaluate; comparison not attempted" >&2
	exit 4
fi

# ---------------------------------------------------------------------------
# Delegate entirely to compare-lifecycle-evaluations.sh (REQ-F-015). Its
# --output is written verbatim, byte-for-byte, to stdout and (only on
# acceptance) to the published retention path; a rejected comparison is
# never copied to the published path, and its divergence reasons are
# printed exactly as the comparator wrote them.
# ---------------------------------------------------------------------------
COMPARISON_ATTEMPT="$(mktemp)"
set +e
"$COMPARATOR_BIN" --left "$left_eval" --right "$right_eval" --mode "$comparison_mode" --output "$COMPARISON_ATTEMPT"
comparator_rc=$?
set -e

case "$comparator_rc" in
0)
	# Published under the deep_review pair's retained rep directory (the
	# terminal/"right" side of REQ-F-014's "feature-QA gate versus the
	# finish-feature deep-review gate" ordering) as the 9th, optional
	# per-pair artifact -- exactly the shape spec.md's Data model table
	# documents and aggregate-lifecycle.sh's pair loop already reads
	# (comparison_path = os.path.join(rep_dir, "comparison.json")). Never
	# duplicated into the qa pair's directory too: aggregate-lifecycle.sh
	# appends one /comparisons row per rep_dir it finds a comparison.json
	# in, so publishing to one location keeps one comparator verdict as
	# exactly one row.
	comparison_rep="$(gate_rep "deep_review")"
	comparison_dest_dir="$out_root_canon/scenarios/$scenario_id_global/$comparison_rep"
	comparison_dest="$comparison_dest_dir/comparison.json"
	assert_within_out_root "$comparison_dest" || exit 1
	# Ancestor-chain check (code-review-2026-08-21T1335-E40-F10.md round-5
	# finding 1), scoped to comparison_dest_dir -- deliberately NOT to
	# comparison_dest itself (the comparison.json FILE), whose own
	# pre-existing-symlink-at-the-leaf case is handled below by design (the
	# `mv -T --` atomic replace tolerates and REPLACES a symlink AT this
	# exact leaf, per TC-087 cases F/G -- that is correct, intended
	# behavior, not a gap). What round 5 found is a DIFFERENT path: a
	# symlink at an ANCESTOR of comparison_dest_dir (e.g.
	# scenarios/$scenario_id_global itself), redirecting the whole directory
	# this file is about to be installed into. dispatch_gate() above already
	# chain-checked this identical directory (comparison_dest_dir ==
	# dispatch_gate's own `dest` for the deep_review gate) earlier in this
	# same single-threaded run, but re-checking here removes any implicit
	# dependency on call-order reasoning between two independent write call
	# sites in this file.
	assert_no_symlink_in_chain "$out_root_canon" "$comparison_dest_dir" || exit 1
	# Symlink write-through (code-review-2026-08-21T0459-E40-F10.md round-4
	# finding 1; originally flagged code-review-2026-08-21T0330-E40-F10.md
	# finding 2): assert_within_out_root above already refuses a symlink at
	# comparison_dest that escapes out_root_canon entirely (realpath -m
	# resolves an existing symlink's final component, so an outside target
	# already fails containment). It canNOT catch the narrower "confused
	# deputy" variant: a symlink at comparison_dest redirected to ANOTHER
	# retained artifact inside the SAME root -- that resolves within
	# out_root_canon and passes containment.
	#
	# The PREVIOUS fix here staged into a temp file in the same directory
	# and installed with plain `mv -- tmp dest`. That was still wrong: GNU
	# coreutils `mv <src> <dst>` (no `-T`) FIRST checks (via a `stat()` that
	# FOLLOWS symlinks) whether `<dst>` is or resolves through a directory;
	# if so, it treats the call as "move <src> INTO the directory named
	# <dst>/basename(<src>)" rather than "replace whatever directory entry
	# is named <dst>" -- so a symlink-to-directory at comparison_dest was
	# never replaced at all: the temp file was silently deposited INSIDE
	# the symlink's target directory, and the driver reported success
	# (empirically confirmed: exit 0, "comparison ACCEPTED and published",
	# while comparison_dest remained a symlink and a stray temp file was
	# planted inside whatever it pointed at -- a direct REQ-NF-007
	# violation). `mv -T --` disables that directory-following stat/detour
	# entirely and goes straight to a rename()-equivalent atomic
	# replacement of the DIRECTORY ENTRY named comparison_dest, regardless
	# of whether that entry is a plain file, a symlink-to-file, or a
	# symlink-to-directory (verified empirically for this fix, both shapes)
	# -- matching POSIX rename(2)/os.replace() semantics, which never
	# dereference the final path component of either operand. This mirrors
	# lib/retain_pair's install_atomic_file pattern and this codebase's
	# established tempfile+os.replace atomic-install convention
	# (lifecycle-prelude.sh, review-capture.sh, lifecycle-worker-adapter.sh)
	# without a second, differently-behaved primitive.
	comparison_install_tmp="$(mktemp "$comparison_dest_dir/.comparison.json.XXXXXX")"
	cp -- "$COMPARISON_ATTEMPT" "$comparison_install_tmp"
	mv -T -- "$comparison_install_tmp" "$comparison_dest"
	echo "run-review-comparison: comparison ACCEPTED and published: $comparison_dest" >&2
	cat "$COMPARISON_ATTEMPT"
	rm -f "$COMPARISON_ATTEMPT"
	exit 0
	;;
1)
	echo "run-review-comparison: comparison REJECTED by compare-lifecycle-evaluations.sh; not published" >&2
	cat "$COMPARISON_ATTEMPT"
	rm -f "$COMPARISON_ATTEMPT"
	exit 4
	;;
*)
	echo "run-review-comparison: compare-lifecycle-evaluations.sh reported an invalid comparison input (exit $comparator_rc)" >&2
	rm -f "$COMPARISON_ATTEMPT"
	exit 2
	;;
esac
