#!/usr/bin/env bash
# lib/spend-gate.sh -- single sourced library owning refusal semantics for
# both provider-backed operator drivers (run-lifecycle-batch.sh,
# run-review-comparison.sh; T-E40-F10-004/005). spec.md REQ-F-002,
# REQ-F-003, ADR-F10-02, ADR-F10-03.
#
# Implements, in this file only, so neither driver invents its own copy:
#   - the explicit --acknowledge-provider-spend check (AC-T2: satisfiable
#     ONLY by the literal CLI flag -- never an environment variable, a
#     config default, or a stored prior acknowledgement);
#   - the max-cost-usd / max-wall-clock-seconds / max-generated-tasks
#     presence+positivity pre-check (AC-T3: the same positivity boundary
#     `run-lifecycle.sh`'s `limits_from()` already enforces -- value must be
#     present, numeric, and > 0 -- re-checked early so a refusal costs zero
#     subprocesses; this library adds no second, divergent validator and
#     never re-derives per-run ceiling *semantics*, only presence/positivity
#     *shape*);
#   - the --retention-root presence check;
#   - the pilot-ledger gate hook for --mode baseline (REQ-F-005).
#     spend_gate_check_pilot_ledger (T-E40-F10-003) is presence-only (does
#     a pilot-ledger.jsonl exist under the retention root) and is called
#     unconditionally from spend_gate_check_all, unchanged, so every
#     existing caller keeps its exact prior behavior (including
#     tc080_spend_gate_refusal_test.sh's direct spend_gate_check_all
#     assertions). spend_gate_check_pilot_ledger_families (T-E40-F10-006)
#     is the REAL per-family, digest-current verification ADR-F10-09
#     requires; it is layered on top, also called unconditionally from
#     spend_gate_check_all, but only ACTS when the caller's argv contains
#     a --batch flag (run-lifecycle-batch.sh's own signature) -- it
#     derives the requested scenario matrix's family set independently
#     from that --batch policy + scenario index (mirroring, but never
#     calling into, run-lifecycle-batch.sh's own matrix enumeration) so
#     REQ-F-005's whole-command refusal runs at this file's existing
#     pre-matrix call site with no change to run-lifecycle-batch.sh
#     itself. A caller with no --batch flag (e.g. run-review-comparison.sh)
#     falls through unchanged -- still gated by the presence-only check
#     above only, per that driver's own scope.
#
# Refusal-reason strings and exit statuses mirror
# bench/reports/lifecycle-baseline-schema.yaml `refusal_reason` /
# `refusal_exit_status` (schema is the vocabulary owner, REQ-F-018; this
# file's literals are checked against that schema by
# tc080_spend_gate_refusal_test.sh so the two cannot silently drift, and
# this is a `MODES`-style precedent already used by
# bench/scripts/compare-lifecycle-evaluations.sh).
#
# AC-T1: every check below is pure bash (regex/string comparison only) --
# no subprocess, scenario checkout, or Shark call is made anywhere in this
# file, so a refusal is unconditionally pre-dispatch.
#
# Caller contract: source this file, then call spend_gate_check_all with
# the driver's *raw, unmodified* argv (mode first). Passing anything other
# than the literal argv (e.g. a value pre-merged from an environment
# variable or config file) would defeat AC-T2's structural guarantee.

SPEND_GATE_EXIT_USAGE_ERROR=2
SPEND_GATE_EXIT_REFUSAL=3

# Resolved once at source time via the SIBLING path (TD-077 defect-class
# precedent, matched by run-lifecycle-batch.sh's own RUN_LIFECYCLE_BIN /
# EVALUATE_LIFECYCLE_BIN), never a bare PATH-resolved name -- so a
# self-test can substitute a stub through one overridable variable without
# touching PATH resolution for anything else spend-gate.sh's callers do.
SPEND_GATE_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PILOT_LEDGER_BIN="${PILOT_LEDGER_BIN:-$SPEND_GATE_LIB_DIR/../pilot-ledger.sh}"
SPEND_GATE_BENCH_DIR="${SPEND_GATE_BENCH_DIR:-$(cd "$SPEND_GATE_LIB_DIR/../.." && pwd)}"

# Set by the last spend_gate_check_* call that refused; empty after a
# passing check or a usage error (a usage error is not a refusal reason).
SPEND_GATE_REFUSAL_REASON=""

_spend_gate_flag_present() {
	local flag="$1"
	shift
	local arg
	for arg in "$@"; do
		[[ "$arg" == "$flag" ]] && return 0
	done
	return 1
}

# Prints the value immediately following the first occurrence of $1 in the
# remaining arguments. Returns 1 (nothing printed) if the flag is absent or
# is the final argument with no value following it.
_spend_gate_flag_value() {
	local flag="$1"
	shift
	local args=("$@")
	local i
	for ((i = 0; i < ${#args[@]}; i++)); do
		if [[ "${args[$i]}" == "$flag" ]]; then
			if ((i + 1 < ${#args[@]})); then
				printf '%s' "${args[$((i + 1))]}"
				return 0
			fi
			return 1
		fi
	done
	return 1
}

_spend_gate_is_numeric() {
	[[ "$1" =~ ^-?[0-9]+(\.[0-9]+)?$ ]]
}

# Caller must already know $1 is numeric (_spend_gate_is_numeric). Returns
# 0 when strictly positive, 1 when zero or negative. Pure string
# comparison -- no subprocess (e.g. awk/bc) -- to keep AC-T1 trivially true.
_spend_gate_is_positive() {
	local value="$1"
	[[ "$value" == -* ]] && return 1
	local digits="${value//./}"
	[[ "$digits" =~ ^0+$ ]] && return 1
	return 0
}

spend_gate_refuse() {
	local reason="$1" message="$2"
	SPEND_GATE_REFUSAL_REASON="$reason"
	echo "spend-gate: refused ($reason): $message" >&2
	return "$SPEND_GATE_EXIT_REFUSAL"
}

spend_gate_usage_error() {
	local message="$1"
	SPEND_GATE_REFUSAL_REASON=""
	echo "spend-gate: usage error: $message" >&2
	return "$SPEND_GATE_EXIT_USAGE_ERROR"
}

# spend_gate_check_acknowledgement <argv...>
# AC-T2: scans only the argv passed to this function -- never consults
# SHARK_BENCH_ACK or any other environment variable, a bench config file,
# or state recorded under a retention root from a prior invocation.
spend_gate_check_acknowledgement() {
	if ! _spend_gate_flag_present "--acknowledge-provider-spend" "$@"; then
		spend_gate_refuse "missing_acknowledgement" \
			"--acknowledge-provider-spend was not supplied on the command line"
		return $?
	fi
	SPEND_GATE_REFUSAL_REASON=""
	return 0
}

# spend_gate_check_ceiling <flag> <missing_reason> <nonpositive_reason> <argv...>
# Absent -> refusal ($missing_reason). Present but non-numeric -> usage
# error (distinct exit status, not a spend-gate refusal). Present, numeric,
# <= 0 -> refusal ($nonpositive_reason). Present, numeric, > 0 -> pass.
spend_gate_check_ceiling() {
	local flag="$1" missing_reason="$2" nonpositive_reason="$3"
	shift 3
	local value
	if ! value="$(_spend_gate_flag_value "$flag" "$@")" || [[ -z "$value" ]]; then
		spend_gate_refuse "$missing_reason" \
			"$flag was not supplied or had no value"
		return $?
	fi
	if ! _spend_gate_is_numeric "$value"; then
		spend_gate_usage_error "$flag value '$value' is not numeric"
		return $?
	fi
	if ! _spend_gate_is_positive "$value"; then
		spend_gate_refuse "$nonpositive_reason" \
			"$flag value '$value' is not strictly positive"
		return $?
	fi
	SPEND_GATE_REFUSAL_REASON=""
	return 0
}

# spend_gate_check_retention_root <argv...>
spend_gate_check_retention_root() {
	local value
	if ! value="$(_spend_gate_flag_value "--retention-root" "$@")" || [[ -z "$value" ]]; then
		spend_gate_refuse "missing_retention_root" \
			"--retention-root was not supplied on the command line or had no value"
		return $?
	fi
	SPEND_GATE_REFUSAL_REASON=""
	return 0
}

# spend_gate_check_pilot_ledger <mode> <retention_root>
# REQ-F-005: only --mode baseline requires a verified attestation; preview
# and pilot are ungated here. Presence-only for now (see file header);
# T-E40-F10-006 extends this with per-family digest verification.
spend_gate_check_pilot_ledger() {
	local mode="$1" retention_root="$2"
	if [[ "$mode" != "baseline" ]]; then
		SPEND_GATE_REFUSAL_REASON=""
		return 0
	fi
	if [[ ! -f "$retention_root/pilot-ledger.jsonl" ]]; then
		spend_gate_refuse "missing_pilot_attestation" \
			"no pilot-ledger.jsonl found under retention root '$retention_root'"
		return $?
	fi
	SPEND_GATE_REFUSAL_REASON=""
	return 0
}

# _spend_gate_families_from_batch <batch_policy_path> <scenarios_filter>
# Prints one family name per line: the entity_family (per admitted I-04
# package.yaml) of every scenario the batch policy's matrix would request,
# filtered by <scenarios_filter> (a comma-separated --scenarios value, or
# empty for "every scenario in the index") -- this MIRRORS, but does not
# call into, run-lifecycle-batch.sh's own matrix-enumeration Python (family
# derivation only; reps/root_key/scratch_root are irrelevant here). Kept
# independent so REQ-F-005's whole-command refusal can run at this file's
# existing pre-matrix call site with no change to run-lifecycle-batch.sh.
#
# Deliberately fails OPEN (prints nothing, never a hard error) on any
# parse problem: a malformed batch policy or scenario index is
# run-lifecycle-batch.sh's own matrix-enumeration step's failure to
# report (it re-parses the same file moments later), not this library's.
_spend_gate_families_from_batch() {
	local batch_policy_path="$1" scenarios_filter="$2"
	[[ -f "$batch_policy_path" ]] || return 0
	command -v python3 >/dev/null 2>&1 || return 0
	python3 - "$SPEND_GATE_BENCH_DIR" "$batch_policy_path" "$scenarios_filter" <<'PYEOF' 2>/dev/null || true
import os
import sys

try:
    import yaml
except ImportError:
    raise SystemExit(0)

bench_dir, batch_policy_path, scenarios_filter = sys.argv[1:4]

try:
    with open(batch_policy_path) as f:
        policy = yaml.safe_load(f) or {}
except Exception:
    raise SystemExit(0)
if not isinstance(policy, dict):
    raise SystemExit(0)

scenario_index_field = policy.get("scenario_index") or "scenarios/scenarios.yaml"
scenario_index_path = (
    scenario_index_field if os.path.isabs(scenario_index_field) else os.path.join(bench_dir, scenario_index_field)
)
if not os.path.isfile(scenario_index_path):
    raise SystemExit(0)

try:
    with open(scenario_index_path) as f:
        index = yaml.safe_load(f) or {}
except Exception:
    raise SystemExit(0)
scenario_dirs = index.get("scenarios") or []
index_dir = os.path.dirname(scenario_index_path)

requested_ids = [s for s in scenarios_filter.split(",") if s] if scenarios_filter else []

families = set()
for rel in scenario_dirs:
    pkg_path = os.path.join(index_dir, rel, "package.yaml")
    if not os.path.isfile(pkg_path):
        continue
    try:
        with open(pkg_path) as f:
            pkg = yaml.safe_load(f) or {}
    except Exception:
        continue
    scenario_id = str(pkg.get("scenario_id", ""))
    if not scenario_id:
        continue
    if requested_ids and scenario_id not in requested_ids:
        continue
    family = str(pkg.get("entity_family") or "")
    if family:
        families.add(family)

for fam in sorted(families):
    print(fam)
PYEOF
}

# spend_gate_check_pilot_ledger_families <mode> <argv...>
# REQ-F-005 whole-command refusal (AC-T1), ADR-F10-09: for --mode baseline
# invocations whose argv contains --batch (run-lifecycle-batch.sh's own
# signature), derive the requested matrix's family set and ask the single
# digest-verification authority (pilot-ledger.sh --verify --family) once
# per family. ANY failing family fails the WHOLE command -- this runs
# before matrix enumeration/dispatch, never as a per-family dispatch
# filter -- and the refusal names every failing family with its specific
# per-family condition (no_attestation vs. stale_digest), even though the
# single schema-owned SPEND_GATE_REFUSAL_REASON code can only carry one of
# the two (missing_pilot_attestation takes precedence when ANY family has
# no attestation at all; stale_pilot_attestation_digest otherwise).
#
# A caller whose argv has no --batch flag (e.g. run-review-comparison.sh)
# is a no-op pass here -- gated only by the presence-only check above,
# which remains this file's sole owner of that driver's baseline gate.
spend_gate_check_pilot_ledger_families() {
	local mode="$1"
	shift
	local argv=("$@")

	if [[ "$mode" != "baseline" ]]; then
		SPEND_GATE_REFUSAL_REASON=""
		return 0
	fi

	local batch_policy
	batch_policy="$(_spend_gate_flag_value "--batch" "${argv[@]}")" || batch_policy=""
	if [[ -z "$batch_policy" ]]; then
		SPEND_GATE_REFUSAL_REASON=""
		return 0
	fi

	local retention_root
	retention_root="$(_spend_gate_flag_value "--retention-root" "${argv[@]}")" || retention_root=""
	if [[ -z "$retention_root" ]]; then
		SPEND_GATE_REFUSAL_REASON=""
		return 0
	fi

	local scenarios_filter
	scenarios_filter="$(_spend_gate_flag_value "--scenarios" "${argv[@]}")" || scenarios_filter=""

	local families=()
	local fam
	while IFS= read -r fam; do
		[[ -n "$fam" ]] && families+=("$fam")
	done < <(_spend_gate_families_from_batch "$batch_policy" "$scenarios_filter")

	if [[ "${#families[@]}" -eq 0 ]]; then
		SPEND_GATE_REFUSAL_REASON=""
		return 0
	fi

	if [[ ! -x "$PILOT_LEDGER_BIN" ]]; then
		SPEND_GATE_REFUSAL_REASON=""
		return 0
	fi

	local failing=() any_missing="false"
	local out rc condition
	for fam in "${families[@]}"; do
		rc=0
		out="$("$PILOT_LEDGER_BIN" --retention-root "$retention_root" --verify --family "$fam" 2>&1)" || rc=$?
		if [[ "$rc" -ne 0 ]]; then
			condition="unknown"
			[[ "$out" =~ FAILED\ \(([a-z_]+) ]] && condition="${BASH_REMATCH[1]}"
			[[ "$condition" == "no_attestation" ]] && any_missing="true"
			failing+=("$fam ($condition)")
		fi
	done

	if [[ "${#failing[@]}" -eq 0 ]]; then
		SPEND_GATE_REFUSAL_REASON=""
		return 0
	fi

	local reason="stale_pilot_attestation_digest"
	[[ "$any_missing" == "true" ]] && reason="missing_pilot_attestation"

	spend_gate_refuse "$reason" \
		"pilot attestation gate failed for the following families: ${failing[*]}"
	return $?
}

# spend_gate_check_all <mode> <argv...>
# Single entrypoint both provider-backed drivers call for --mode
# pilot|baseline, before any subprocess/checkout/Shark call. <argv...> MUST
# be the driver's raw, unmodified argv (AC-T2). Checks run in a fixed
# order and short-circuit on the first failure; the caller reads the exit
# status (SPEND_GATE_EXIT_REFUSAL vs SPEND_GATE_EXIT_USAGE_ERROR) and
# SPEND_GATE_REFUSAL_REASON.
spend_gate_check_all() {
	local mode="$1"
	shift
	local argv=("$@")

	spend_gate_check_acknowledgement "${argv[@]}" || return $?
	spend_gate_check_ceiling "--max-cost-usd" "missing_max_cost_usd" "non_positive_max_cost_usd" "${argv[@]}" || return $?
	spend_gate_check_ceiling "--max-wall-clock-seconds" "missing_max_wall_clock_seconds" "non_positive_max_wall_clock_seconds" "${argv[@]}" || return $?
	spend_gate_check_ceiling "--max-generated-tasks" "missing_max_generated_tasks" "non_positive_max_generated_tasks" "${argv[@]}" || return $?
	spend_gate_check_retention_root "${argv[@]}" || return $?

	local retention_root
	retention_root="$(_spend_gate_flag_value "--retention-root" "${argv[@]}")"
	spend_gate_check_pilot_ledger "$mode" "$retention_root" || return $?
	spend_gate_check_pilot_ledger_families "$mode" "${argv[@]}" || return $?

	SPEND_GATE_REFUSAL_REASON=""
	return 0
}
