#!/usr/bin/env bash
# TC-080 / T-E40-F10-003: provider-spend gate refusal (spec.md REQ-F-002,
# REQ-F-003; test-plan.md TC-080 full body and Caller-Path Contract row).
#
# This task owns bench/scripts/lib/spend-gate.sh only. TC-080's
# driver-level assertions (running the flags through the real
# run-lifecycle-batch.sh / run-review-comparison.sh CLIs) belong to
# T-E40-F10-004/005; this test exercises the sourced library directly,
# which is TC-080's own Caller-Path Contract instruction: "real
# spend-gate.sh sourced library and real CLI parsing... Do not mock flag
# parsing, spend-gate.sh refusal logic, or the pre-dispatch short-circuit."
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
LIB="$SCRIPTS_DIR/lib/spend-gate.sh"
SCHEMA="$REPO_ROOT/bench/reports/lifecycle-baseline-schema.yaml"
RUNLIFECYCLE="$SCRIPTS_DIR/run-lifecycle.sh"

fail() {
	echo "TC-080 FAIL: $1" >&2
	exit 1
}

[[ -f "$LIB" ]] || fail "bench/scripts/lib/spend-gate.sh missing"
[[ -f "$SCHEMA" ]] || fail "bench/reports/lifecycle-baseline-schema.yaml missing"

# shellcheck source=/dev/null
source "$LIB"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
ROOT="$WORKDIR/retention"
mkdir -p "$ROOT"

# ---------------------------------------------------------------------------
# AC-T1 pre-dispatch spy: sentinel binaries for every subprocess/checkout/
# Shark call a driver could plausibly make. If spend_gate_check_all (or any
# function it calls) ever shells out to one of these, the spy log below
# grows -- proving the refusal happened before dispatch, not just alongside
# a fast-failing one (TC-080 Caller-Path Contract "note").
# ---------------------------------------------------------------------------
SPY_LOG="$WORKDIR/spy.log"
: >"$SPY_LOG"
mkdir -p "$WORKDIR/spybin"
for bin in shark run-lifecycle.sh evaluate-lifecycle.sh git; do
	cat >"$WORKDIR/spybin/$bin" <<SPY
#!/usr/bin/env bash
echo "$bin \$*" >>"$SPY_LOG"
exit 1
SPY
	chmod +x "$WORKDIR/spybin/$bin"
done

assert_no_spy_calls() {
	local label="$1"
	if [[ -s "$SPY_LOG" ]]; then
		fail "$label: unexpected subprocess/checkout/Shark call recorded: $(cat "$SPY_LOG")"
	fi
	return 0
}

# All refusal-path calls below run with the spy binaries first on PATH and a
# real-source spend-gate.sh -- proving AC-T1 for every case in one pass.
export PATH="$WORKDIR/spybin:$PATH"

ACK="--acknowledge-provider-spend"
ROOT_FLAG=(--retention-root "$ROOT")
GOOD_CEILINGS=(--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10)

assert_refusal() {
	# assert_refusal <label> <expected_reason> <mode> <argv...>
	local label="$1" expected_reason="$2" mode="$3"
	shift 3
	local rc=0
	spend_gate_check_all "$mode" "$@" || rc=$?
	[[ "$rc" -eq "$SPEND_GATE_EXIT_REFUSAL" ]] || fail "$label: expected refusal exit $SPEND_GATE_EXIT_REFUSAL, got $rc"
	[[ "$SPEND_GATE_REFUSAL_REASON" == "$expected_reason" ]] || fail "$label: expected reason '$expected_reason', got '$SPEND_GATE_REFUSAL_REASON'"
	assert_no_spy_calls "$label"
}

assert_usage_error() {
	# assert_usage_error <label> <mode> <argv...>
	local label="$1" mode="$2"
	shift 2
	local rc=0
	spend_gate_check_all "$mode" "$@" || rc=$?
	[[ "$rc" -eq "$SPEND_GATE_EXIT_USAGE_ERROR" ]] || fail "$label: expected usage-error exit $SPEND_GATE_EXIT_USAGE_ERROR, got $rc"
	[[ -z "$SPEND_GATE_REFUSAL_REASON" ]] || fail "$label: usage error must not set a refusal reason, got '$SPEND_GATE_REFUSAL_REASON'"
	assert_no_spy_calls "$label"
}

# ---------------------------------------------------------------------------
# (a) No acknowledgement flag at all.
# ---------------------------------------------------------------------------
assert_refusal "no-ack" "missing_acknowledgement" pilot \
	"${ROOT_FLAG[@]}" "${GOOD_CEILINGS[@]}"

# ---------------------------------------------------------------------------
# (b) Acknowledgement present, one ceiling zero/negative/absent at a time
# (BVA), for all three ceiling flags. Each other ceiling stays valid so the
# refusal is attributable to exactly the ceiling under test.
# ---------------------------------------------------------------------------
declare -A CEILING_MISSING=(
	["--max-cost-usd"]="missing_max_cost_usd"
	["--max-wall-clock-seconds"]="missing_max_wall_clock_seconds"
	["--max-generated-tasks"]="missing_max_generated_tasks"
)
declare -A CEILING_NONPOSITIVE=(
	["--max-cost-usd"]="non_positive_max_cost_usd"
	["--max-wall-clock-seconds"]="non_positive_max_wall_clock_seconds"
	["--max-generated-tasks"]="non_positive_max_generated_tasks"
)

build_ceilings_without() {
	# Prints GOOD_CEILINGS with the named flag's pair removed.
	local skip="$1" out=() i
	for ((i = 0; i < ${#GOOD_CEILINGS[@]}; i += 2)); do
		if [[ "${GOOD_CEILINGS[$i]}" != "$skip" ]]; then
			out+=("${GOOD_CEILINGS[$i]}" "${GOOD_CEILINGS[$((i + 1))]}")
		fi
	done
	printf '%s\n' "${out[@]}"
}

for flag in "--max-cost-usd" "--max-wall-clock-seconds" "--max-generated-tasks"; do
	mapfile -t others < <(build_ceilings_without "$flag")

	assert_refusal "absent:$flag" "${CEILING_MISSING[$flag]}" pilot \
		"$ACK" "${ROOT_FLAG[@]}" "${others[@]}"

	assert_refusal "zero:$flag" "${CEILING_NONPOSITIVE[$flag]}" pilot \
		"$ACK" "${ROOT_FLAG[@]}" "${others[@]}" "$flag" 0

	assert_refusal "negative:$flag" "${CEILING_NONPOSITIVE[$flag]}" pilot \
		"$ACK" "${ROOT_FLAG[@]}" "${others[@]}" "$flag" -1

	# Non-numeric ceiling value: usage error, NOT a spend-gate refusal
	# (TC-080 Edge Cases: "distinguish exit codes").
	assert_usage_error "nonnumeric:$flag" pilot \
		"$ACK" "${ROOT_FLAG[@]}" "${others[@]}" "$flag" "not-a-number"
done

# ---------------------------------------------------------------------------
# (c) No --retention-root.
# ---------------------------------------------------------------------------
assert_refusal "no-retention-root" "missing_retention_root" pilot \
	"$ACK" "${GOOD_CEILINGS[@]}"

# ---------------------------------------------------------------------------
# (d) AC-T2: acknowledgement supplied only via an environment variable, or
# only via a config-file default, with no CLI flag. Both must refuse
# identically to case (a) -- spend_gate_check_acknowledgement scans only
# the argv it is given and never consults the environment or a config
# file, so this is a structural (not incidental) guarantee.
# ---------------------------------------------------------------------------
CONFIG_FILE="$WORKDIR/bench-config.json"
printf '{"acknowledge_provider_spend": true}\n' >"$CONFIG_FILE"

SHARK_BENCH_ACK=1 assert_refusal "ack-via-env-var" "missing_acknowledgement" pilot \
	"${ROOT_FLAG[@]}" "${GOOD_CEILINGS[@]}"

assert_refusal "ack-via-config-default" "missing_acknowledgement" pilot \
	--bench-config "$CONFIG_FILE" "${ROOT_FLAG[@]}" "${GOOD_CEILINGS[@]}"

# A stored prior acknowledgement (e.g. a previous successful run against
# the same retention root) must not satisfy the gate on a new invocation --
# spend_gate_check_acknowledgement reads no retention-root state at all, so
# recording an on-disk "prior ack" marker and re-checking without the flag
# proves the new invocation still refuses.
echo '{"acknowledged": true}' >"$ROOT/prior-ack.json"
assert_refusal "ack-not-carried-forward" "missing_acknowledgement" pilot \
	"${ROOT_FLAG[@]}" "${GOOD_CEILINGS[@]}"

# ---------------------------------------------------------------------------
# Fully-satisfied invocation (TC-080 Caller-Path Contract baseline): only
# this one proceeds past the gate. Both --mode pilot and --mode baseline
# are exercised; baseline additionally requires the pilot-ledger presence
# gate, tested by the fixture below.
# ---------------------------------------------------------------------------
rc=0
spend_gate_check_all pilot "$ACK" "${ROOT_FLAG[@]}" "${GOOD_CEILINGS[@]}" || rc=$?
[[ "$rc" -eq 0 ]] || fail "fully-satisfied pilot invocation refused unexpectedly (rc=$rc, reason=$SPEND_GATE_REFUSAL_REASON)"
[[ -z "$SPEND_GATE_REFUSAL_REASON" ]] || fail "fully-satisfied invocation left a stale refusal reason: $SPEND_GATE_REFUSAL_REASON"
assert_no_spy_calls "fully-satisfied-pilot"

# ---------------------------------------------------------------------------
# Additional library-level coverage requested for this task (beyond
# TC-080's own body, which TC-081 will exercise end-to-end once
# pilot-ledger.sh exists in T-E40-F10-006): the --mode baseline
# pilot-ledger presence gate hook.
# ---------------------------------------------------------------------------
assert_refusal "baseline-no-pilot-ledger" "missing_pilot_attestation" baseline \
	"$ACK" "${ROOT_FLAG[@]}" "${GOOD_CEILINGS[@]}"

: >"$ROOT/pilot-ledger.jsonl"
rc=0
spend_gate_check_all baseline "$ACK" "${ROOT_FLAG[@]}" "${GOOD_CEILINGS[@]}" || rc=$?
[[ "$rc" -eq 0 ]] || fail "baseline invocation with a present pilot-ledger.jsonl refused unexpectedly (rc=$rc, reason=$SPEND_GATE_REFUSAL_REASON)"
assert_no_spy_calls "baseline-with-pilot-ledger"

# ---------------------------------------------------------------------------
# Schema-owned refusal reason (REQ-F-018): every reason string this test
# has asserted above must be a member of the schema's closed vocabulary --
# spend-gate.sh must not invent a private refusal-reason string.
# ---------------------------------------------------------------------------
for reason in missing_acknowledgement missing_max_cost_usd non_positive_max_cost_usd \
	missing_max_wall_clock_seconds non_positive_max_wall_clock_seconds \
	missing_max_generated_tasks non_positive_max_generated_tasks \
	missing_retention_root missing_pilot_attestation; do
	grep -qE "^\s*-\s*${reason}\s*\$" "$SCHEMA" || fail "refusal reason '$reason' is not present in $SCHEMA's refusal_reason vocabulary"
done

# Distinct refusal exit status vs. usage-error status, cross-checked
# against the schema's own refusal_exit_status block so the two constants
# in spend-gate.sh cannot silently drift from the schema.
[[ "$SPEND_GATE_EXIT_USAGE_ERROR" != "$SPEND_GATE_EXIT_REFUSAL" ]] || fail "usage-error and spend-gate refusal exit statuses must be distinct"
grep -qE "^\s*usage_error:\s*${SPEND_GATE_EXIT_USAGE_ERROR}\s*\$" "$SCHEMA" || fail "SPEND_GATE_EXIT_USAGE_ERROR ($SPEND_GATE_EXIT_USAGE_ERROR) does not match schema's refusal_exit_status.usage_error"
grep -qE "^\s*spend_gate_refusal:\s*${SPEND_GATE_EXIT_REFUSAL}\s*\$" "$SCHEMA" || fail "SPEND_GATE_EXIT_REFUSAL ($SPEND_GATE_EXIT_REFUSAL) does not match schema's refusal_exit_status.spend_gate_refusal"

# ---------------------------------------------------------------------------
# AC-T3: ceiling positivity delegates to run-lifecycle.sh's existing
# positivity check (research-report.md `pattern_contract`, ~lines 300-314;
# the Python `limits_from()` embedded in run-lifecycle.sh's heredoc) rather
# than adding a second, divergent validator. Proven functionally: feeding
# run-lifecycle.sh the exact boundary values spend-gate.sh refuses (0, -1)
# must trip run-lifecycle.sh's OWN positivity check with the same verdict,
# not some other failure. limits_from() runs before run-lifecycle.sh even
# resolves the `shark` binary, so this needs no shark/adapter fixture --
# only a real admitted scenario package (reused from tc062's fixture).
# ---------------------------------------------------------------------------
[[ -x "$RUNLIFECYCLE" ]] || fail "bench/scripts/run-lifecycle.sh missing or not executable"
SCENARIO="$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml"
[[ -f "$SCENARIO" ]] || fail "fixture scenario package missing: $SCENARIO"

for boundary in 0 -1; do
	LIMITS_FILE="$WORKDIR/limits-${boundary}.yaml"
	cat >"$LIMITS_FILE" <<EOF
max_cost_usd: ${boundary}
max_wall_clock_seconds: 600
max_generated_tasks: 10
EOF
	ERR_FILE="$WORKDIR/limits-${boundary}.err"
	# Deliberately outside the spy PATH: this call exercises the real
	# run-lifecycle.sh authority, not spend-gate.sh's pre-check.
	if env PATH="/usr/bin:/bin:/usr/local/bin" "$RUNLIFECYCLE" \
		--scenario "$SCENARIO" --run-id "tc080-boundary-${boundary}" --root ROOT-001 \
		--scratch-root "$WORKDIR/scratch-${boundary}" --limits "$LIMITS_FILE" \
		--output "$WORKDIR/lifecycle-${boundary}.jsonl" --mode dry-run \
		>/dev/null 2>"$ERR_FILE"; then
		fail "run-lifecycle.sh accepted a non-positive max_cost_usd ($boundary); spend-gate.sh's pre-check would diverge from its authority"
	fi
	grep -q "resource ceilings must be positive" "$ERR_FILE" \
		|| fail "run-lifecycle.sh rejected max_cost_usd=$boundary for an unexpected reason: $(cat "$ERR_FILE")"

	# spend-gate.sh must refuse this same boundary value too, with the
	# non_positive_max_cost_usd reason (same directionality, distinct
	# owner: presence/positivity shape here, per-run semantics there).
	assert_refusal "ac-t3-boundary-${boundary}" "non_positive_max_cost_usd" pilot \
		"$ACK" "${ROOT_FLAG[@]}" --max-cost-usd "$boundary" \
		--max-wall-clock-seconds 600 --max-generated-tasks 10
done

echo "TC-080: pass (spend-gate.sh refuses on every missing/non-positive condition before any subprocess, ack is CLI-flag-only, ceiling positivity matches run-lifecycle.sh's own authority)"
