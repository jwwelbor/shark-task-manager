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
#   - the pilot-ledger gate hook for --mode baseline (REQ-F-005). This task
#     (T-E40-F10-003) provides presence-only gating (does a
#     pilot-ledger.jsonl exist under the retention root); T-E40-F10-006
#     (pilot-ledger.sh) extends spend_gate_check_pilot_ledger with
#     per-family digest verification once the ledger format and retained
#     artifacts it verifies against exist.
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

	SPEND_GATE_REFUSAL_REASON=""
	return 0
}
