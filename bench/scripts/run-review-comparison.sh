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
# Parse candidate.yaml -> exactly two JSON rows, one per gate (qa,
# deep_review). Pure computation: reads the candidate declaration and the
# I-04 scenario index only -- no subprocess, no Shark call.
# ---------------------------------------------------------------------------
CANDIDATE_TMP="$(mktemp)"
cleanup_candidate() { rm -f "$CANDIDATE_TMP"; }
trap cleanup_candidate EXIT

set +e
python3 - "$BENCH_DIR" "$candidate_decl" >"$CANDIDATE_TMP" <<'PYEOF'
import json
import os
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
	# retain_gate <scenario_id> <rep> <gate> <package_path> <lifecycle_jsonl> <evaluation_jsonl> <i05_bundle_dir>
	local scenario_id="$1" rep="$2" gate="$3" package_path="$4" lifecycle_jsonl="$5" evaluation_jsonl="$6" i05_bundle_dir="$7"
	local dest="$out_root_canon/scenarios/$scenario_id/$rep"
	assert_within_out_root "$dest" || return 1
	# Symlink write-through, dest-level (code-review-2026-08-21T0330-E40-F10.md
	# finding 2, swept to this call site's own "rep/gate-provenance
	# assumption"): assert_within_out_root resolves an existing symlink at
	# `dest` and refuses it if it escapes out_root_canon, but a symlink
	# redirected to ANOTHER retained pair's directory INSIDE the same root
	# still passes containment -- mkdir -p would then silently no-op
	# against it, and every artifact lib/retain_pair writes would land in
	# that unrelated in-root directory instead of this gate's own. Refuse
	# loudly before mkdir -p ever runs, rather than let a symlinked `dest`
	# masquerade as this gate's real directory.
	if [[ -L "$dest" ]]; then
		echo "run-review-comparison: $gate gate: refusing to retain -- $dest is a pre-existing symlink, not a real directory" >&2
		return 1
	fi
	mkdir -p "$dest"
	python3 "$SCRIPT_DIR/lib/retain_pair" "$scenario_id" "$rep" "$package_path" "$lifecycle_jsonl" "$evaluation_jsonl" "$i05_bundle_dir" "$dest" "$gate"
}

gate_dest_provenance_ok() {
	# gate_dest_provenance_ok <dest> <gate> -- REQ-NF-007 (append-and-verify,
	# never a silent overwrite/reuse): a bare "evaluation.jsonl exists"
	# check is not proof THIS gate retained it -- the same (scenario_id,
	# rep) path could be occupied by an unrelated run-lifecycle-batch.sh
	# pair, or by the OTHER gate if gate_rep() were ever misconfigured.
	# manifest.json's additive `gate` field (lib/retain_pair) is the
	# provenance check; a mismatch or absence refuses loudly instead of
	# silently adopting someone else's retained evaluation.jsonl as this
	# gate's comparator input.
	local dest="$1" gate="$2"
	[[ -f "$dest/manifest.json" ]] || return 1
	local recorded_gate
	recorded_gate="$(python3 -c 'import json,sys
try:
    print(json.load(open(sys.argv[1])).get("gate",""))
except Exception:
    print("")' "$dest/manifest.json" 2>/dev/null || true)"
	[[ "$recorded_gate" == "$gate" ]]
}

dispatch_gate() {
	# dispatch_gate <scenario_id> <gate> <package_path> <root_key> <scratch_root> <i05_bundle_dir>
	# Prints the resolved evaluation.jsonl path on success (stdout).
	local scenario_id="$1" gate="$2" package_path="$3" root_key="$4" scratch_root="$5" i05_bundle_dir="$6"
	local rep
	rep="$(gate_rep "$gate")" || return 1
	local dest="$out_root_canon/scenarios/$scenario_id/$rep"

	if [[ -f "$dest/evaluation.jsonl" ]]; then
		if ! gate_dest_provenance_ok "$dest" "$gate"; then
			echo "run-review-comparison: $gate gate: retention path $dest is already occupied by a retained pair this driver did not produce for gate '$gate' (missing/mismatched manifest.json 'gate' field) -- refusing to reuse or overwrite" >&2
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
	if ! cp -a "$scratch_root" "$ephemeral"; then
		echo "run-review-comparison: $gate gate: failed to copy scratch_root into an ephemeral working directory" >&2
		rm -rf "$pair_work"
		return 1
	fi
	local lifecycle_out="$pair_work/lifecycle.jsonl"

	echo "run-review-comparison: dispatching $gate gate" >&2
	set +e
	"$RUN_LIFECYCLE_BIN" --scenario "$package_path" --run-id "cmp-${scenario_id}-${gate}" \
		--root "$root_key" --scratch-root "$ephemeral" --output "$lifecycle_out" </dev/null
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

	if ! retain_gate "$scenario_id" "$rep" "$gate" "$package_path" "$lifecycle_out" "$evaluation_out" "$i05_bundle_dir"; then
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
	# Symlink write-through (code-review-2026-08-21T0330-E40-F10.md finding
	# 2, swept into every write call this driver owns): assert_within_out_root
	# above already refuses a symlink at comparison_dest that escapes
	# out_root_canon entirely (realpath -m resolves an existing symlink's
	# final component, so an outside target already fails containment). It
	# canNOT catch the narrower "confused deputy" variant: a symlink at
	# comparison_dest redirected to ANOTHER retained artifact inside the
	# SAME root -- that resolves within out_root_canon and passes
	# containment, yet a plain `cp --` would still silently overwrite that
	# unrelated in-root file. Staging into a temp file inside the SAME
	# directory (guaranteeing the same filesystem, so the install below is
	# a real atomic rename) and installing with `mv --` replaces WHATEVER
	# is at comparison_dest -- symlink or not, in-root target or out --
	# instead of writing through it, matching lib/retain_pair's
	# install_atomic_file/install_atomic_dir pattern and this codebase's
	# established tempfile+os.replace atomic-install convention
	# (lifecycle-prelude.sh, review-capture.sh, lifecycle-worker-adapter.sh).
	comparison_install_tmp="$(mktemp "$comparison_dest_dir/.comparison.json.XXXXXX")"
	cp -- "$COMPARISON_ATTEMPT" "$comparison_install_tmp"
	mv -- "$comparison_install_tmp" "$comparison_dest"
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
