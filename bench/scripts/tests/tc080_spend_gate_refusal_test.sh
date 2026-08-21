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
# Captured before the refusal-path spy sections below permanently `export`
# a PATH prefix (their spy `git`/`shark` stubs are meant to prove a
# pre-dispatch refusal makes zero subprocess calls, and are never restored
# afterward within this file) -- the UAT-R2-01 section far below needs a
# real, unshadowed `git` for candidate_identity()'s own git calls, so it
# rebuilds its PATH from this clean snapshot rather than the ambient,
# by-then-spy-polluted $PATH.
ORIGINAL_PATH="$PATH"

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

# ---------------------------------------------------------------------------
# TC-080 driver-level half (T-E40-F10-004): the same conditions, run through
# the REAL run-lifecycle-batch.sh CLI end to end -- real argv parsing, real
# spend-gate.sh sourcing, real pre-dispatch short-circuit -- not the sourced
# library called directly as above. Header comment above flags this section
# as T-E40-F10-004's own addition (run-review-comparison.sh's comparison
# half is T-E40-F10-005's, not added here).
# ---------------------------------------------------------------------------
BATCH="$SCRIPTS_DIR/run-lifecycle-batch.sh"
[[ -x "$BATCH" ]] || fail "bench/scripts/run-lifecycle-batch.sh missing or not executable"

DRIVER_WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR" "$DRIVER_WORKDIR"' EXIT

DRIVER_SPY_LOG="$DRIVER_WORKDIR/spy.log"
: >"$DRIVER_SPY_LOG"
mkdir -p "$DRIVER_WORKDIR/spybin"
for bin in shark run-lifecycle.sh evaluate-lifecycle.sh git; do
	cat >"$DRIVER_WORKDIR/spybin/$bin" <<SPY
#!/usr/bin/env bash
echo "$bin \$*" >>"$DRIVER_SPY_LOG"
exit 1
SPY
	chmod +x "$DRIVER_WORKDIR/spybin/$bin"
done

DRIVER_ROOT="$DRIVER_WORKDIR/retention"
cat >"$DRIVER_WORKDIR/batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
EOF

assert_driver_refusal() {
	# assert_driver_refusal <label> <expected_reason_substring> <argv...>
	local label="$1" reason="$2"
	shift 2
	local out rc=0
	out="$(PATH="$DRIVER_WORKDIR/spybin:$PATH" "$BATCH" "$@" 2>&1)" || rc=$?
	[[ "$rc" -eq "$SPEND_GATE_EXIT_REFUSAL" ]] || fail "driver:$label: expected refusal exit $SPEND_GATE_EXIT_REFUSAL, got $rc: $out"
	echo "$out" | grep -q "$reason" || fail "driver:$label: expected refusal reason '$reason' in output: $out"
	if [[ -s "$DRIVER_SPY_LOG" ]]; then
		fail "driver:$label: unexpected subprocess/checkout/Shark call recorded: $(cat "$DRIVER_SPY_LOG")"
	fi
}

assert_driver_refusal "no-ack" "missing_acknowledgement" \
	--batch "$DRIVER_WORKDIR/batch-policy.yaml" --retention-root "$DRIVER_ROOT" --mode pilot \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10

assert_driver_refusal "zero-cost" "non_positive_max_cost_usd" \
	--batch "$DRIVER_WORKDIR/batch-policy.yaml" --retention-root "$DRIVER_ROOT" --mode pilot \
	--acknowledge-provider-spend --max-cost-usd 0 --max-wall-clock-seconds 600 --max-generated-tasks 10

assert_driver_refusal "no-retention-root" "missing_retention_root" \
	--batch "$DRIVER_WORKDIR/batch-policy.yaml" --mode pilot \
	--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10

: >"$DRIVER_SPY_LOG"
DRIVER_ACK_ENV_OUT="$(PATH="$DRIVER_WORKDIR/spybin:$PATH" SHARK_BENCH_ACK=1 "$BATCH" \
	--batch "$DRIVER_WORKDIR/batch-policy.yaml" --retention-root "$DRIVER_ROOT" --mode pilot \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 2>&1)" && DRIVER_ACK_ENV_RC=0 || DRIVER_ACK_ENV_RC=$?
[[ "$DRIVER_ACK_ENV_RC" -eq "$SPEND_GATE_EXIT_REFUSAL" ]] || fail "driver:ack-via-env-var: expected refusal exit $SPEND_GATE_EXIT_REFUSAL, got $DRIVER_ACK_ENV_RC: $DRIVER_ACK_ENV_OUT"
echo "$DRIVER_ACK_ENV_OUT" | grep -q "missing_acknowledgement" || fail "driver:ack-via-env-var: expected missing_acknowledgement: $DRIVER_ACK_ENV_OUT"

echo "TC-080(driver half, T-E40-F10-004): run-lifecycle-batch.sh's real CLI refuses every missing/non-positive/env-only-ack condition before any subprocess -- PASS"

# Fully-satisfied invocation proceeds past the gate (proven by reaching
# dispatch logic, not by exit 0 -- an unconfigured scenario's pair is
# correctly classified `failed` per run-lifecycle-batch.sh's own honest
# fallback, giving batch_partial_failure (4), which is only reachable once
# the spend gate has already passed).
DRIVER_PASS_OUT="$(PATH="$DRIVER_WORKDIR/spybin:$PATH" "$BATCH" \
	--batch "$DRIVER_WORKDIR/batch-policy.yaml" --retention-root "$DRIVER_ROOT" --mode pilot \
	--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 2>&1)" && DRIVER_PASS_RC=0 || DRIVER_PASS_RC=$?
[[ "$DRIVER_PASS_RC" -eq 4 ]] || fail "driver:fully-satisfied: expected batch_partial_failure exit 4 (gate passed, dispatch reached, pair unconfigured), got $DRIVER_PASS_RC: $DRIVER_PASS_OUT"
echo "$DRIVER_PASS_OUT" | grep -q '"classification": "failed"' || fail "driver:fully-satisfied: expected dispatch to be reached and classify the unconfigured scenario pairs as failed: $DRIVER_PASS_OUT"
grep -q "root_key_or_scratch_root_not_configured" "$DRIVER_ROOT/invalid/index.jsonl" \
	|| fail "driver:fully-satisfied: invalid/index.jsonl did not name the unconfigured-scenario reason: $(cat "$DRIVER_ROOT/invalid/index.jsonl" 2>/dev/null)"

echo "TC-080(driver half, T-E40-F10-004): fully-satisfied invocation proceeds past the spend gate to real dispatch classification -- PASS"

# ---------------------------------------------------------------------------
# TC-080 comparison-driver half (T-E40-F10-005): the same conditions, run
# through the REAL run-review-comparison.sh CLI end to end -- real argv
# parsing, real spend-gate.sh sourcing, real pre-dispatch short-circuit.
# ---------------------------------------------------------------------------
COMPARISON="$SCRIPTS_DIR/run-review-comparison.sh"
[[ -x "$COMPARISON" ]] || fail "bench/scripts/run-review-comparison.sh missing or not executable"

CMP_DRIVER_WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR" "$DRIVER_WORKDIR" "$CMP_DRIVER_WORKDIR"' EXIT

CMP_DRIVER_SPY_LOG="$CMP_DRIVER_WORKDIR/spy.log"
: >"$CMP_DRIVER_SPY_LOG"
mkdir -p "$CMP_DRIVER_WORKDIR/spybin"
for bin in shark run-lifecycle.sh evaluate-lifecycle.sh compare-lifecycle-evaluations.sh git; do
	cat >"$CMP_DRIVER_WORKDIR/spybin/$bin" <<SPY
#!/usr/bin/env bash
echo "$bin \$*" >>"$CMP_DRIVER_SPY_LOG"
exit 1
SPY
	chmod +x "$CMP_DRIVER_WORKDIR/spybin/$bin"
done

CMP_DRIVER_ROOT="$CMP_DRIVER_WORKDIR/retention"
cat >"$CMP_DRIVER_WORKDIR/candidate.yaml" <<EOF
schema_version: "1.0"
scenario_id: "py-bug-due-date-boundary"
gates:
  qa:
    root_key: "ROOT-001"
    scratch_root: "$CMP_DRIVER_WORKDIR/scratch-template"
  deep_review:
    root_key: "ROOT-001"
    scratch_root: "$CMP_DRIVER_WORKDIR/scratch-template"
EOF

assert_cmp_driver_refusal() {
	# assert_cmp_driver_refusal <label> <expected_reason_substring> <argv...>
	local label="$1" reason="$2"
	shift 2
	local out rc=0
	out="$(PATH="$CMP_DRIVER_WORKDIR/spybin:$PATH" "$COMPARISON" "$@" 2>&1)" || rc=$?
	[[ "$rc" -eq "$SPEND_GATE_EXIT_REFUSAL" ]] || fail "cmp-driver:$label: expected refusal exit $SPEND_GATE_EXIT_REFUSAL, got $rc: $out"
	echo "$out" | grep -q "$reason" || fail "cmp-driver:$label: expected refusal reason '$reason' in output: $out"
	if [[ -s "$CMP_DRIVER_SPY_LOG" ]]; then
		fail "cmp-driver:$label: unexpected subprocess/checkout/Shark/comparator call recorded: $(cat "$CMP_DRIVER_SPY_LOG")"
	fi
}

assert_cmp_driver_refusal "no-ack" "missing_acknowledgement" \
	--candidate "$CMP_DRIVER_WORKDIR/candidate.yaml" --retention-root "$CMP_DRIVER_ROOT" \
	--mode pilot --comparison-mode independent_frozen_candidate \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10

assert_cmp_driver_refusal "zero-cost" "non_positive_max_cost_usd" \
	--candidate "$CMP_DRIVER_WORKDIR/candidate.yaml" --retention-root "$CMP_DRIVER_ROOT" \
	--mode pilot --comparison-mode independent_frozen_candidate \
	--acknowledge-provider-spend --max-cost-usd 0 --max-wall-clock-seconds 600 --max-generated-tasks 10

assert_cmp_driver_refusal "no-retention-root" "missing_retention_root" \
	--candidate "$CMP_DRIVER_WORKDIR/candidate.yaml" \
	--mode pilot --comparison-mode independent_frozen_candidate \
	--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10

: >"$CMP_DRIVER_SPY_LOG"
CMP_DRIVER_ACK_ENV_OUT="$(PATH="$CMP_DRIVER_WORKDIR/spybin:$PATH" SHARK_BENCH_ACK=1 "$COMPARISON" \
	--candidate "$CMP_DRIVER_WORKDIR/candidate.yaml" --retention-root "$CMP_DRIVER_ROOT" \
	--mode pilot --comparison-mode independent_frozen_candidate \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 2>&1)" && CMP_DRIVER_ACK_ENV_RC=0 || CMP_DRIVER_ACK_ENV_RC=$?
[[ "$CMP_DRIVER_ACK_ENV_RC" -eq "$SPEND_GATE_EXIT_REFUSAL" ]] || fail "cmp-driver:ack-via-env-var: expected refusal exit $SPEND_GATE_EXIT_REFUSAL, got $CMP_DRIVER_ACK_ENV_RC: $CMP_DRIVER_ACK_ENV_OUT"
echo "$CMP_DRIVER_ACK_ENV_OUT" | grep -q "missing_acknowledgement" || fail "cmp-driver:ack-via-env-var: expected missing_acknowledgement: $CMP_DRIVER_ACK_ENV_OUT"

echo "TC-080(comparison driver half, T-E40-F10-005): run-review-comparison.sh's real CLI refuses every missing/non-positive/env-only-ack/no-retention-root condition before any subprocess -- PASS"

# Fully-satisfied invocation proceeds past the gate (proven by reaching
# real gate dispatch: the unconfigured scratch_root -- it does not exist on
# disk -- causes an honest per-gate dispatch failure, giving exit 4 "ran,
# comparison not attempted/published", which is only reachable once the
# spend gate has already passed).
CMP_DRIVER_PASS_OUT="$(PATH="$CMP_DRIVER_WORKDIR/spybin:$PATH" "$COMPARISON" \
	--candidate "$CMP_DRIVER_WORKDIR/candidate.yaml" --retention-root "$CMP_DRIVER_ROOT" \
	--mode pilot --comparison-mode independent_frozen_candidate \
	--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 2>&1)" && CMP_DRIVER_PASS_RC=0 || CMP_DRIVER_PASS_RC=$?
[[ "$CMP_DRIVER_PASS_RC" -eq 4 ]] || fail "cmp-driver:fully-satisfied: expected exit 4 (gate passed, dispatch reached, gate scratch_root missing), got $CMP_DRIVER_PASS_RC: $CMP_DRIVER_PASS_OUT"
echo "$CMP_DRIVER_PASS_OUT" | grep -q "one or both gates failed to dispatch/evaluate" || fail "cmp-driver:fully-satisfied: expected dispatch to be reached and fail on the unconfigured scratch_root: $CMP_DRIVER_PASS_OUT"

echo "TC-080(comparison driver half, T-E40-F10-005): fully-satisfied invocation proceeds past the spend gate to real gate dispatch -- PASS"

# ===========================================================================
# UAT-R2-01 regression (uat-2026-08-21T185059Z-E40-F10.md, T-E40-F10-004/005):
# the acknowledgement + ceiling-positivity gate above used to be pure theater
# -- both drivers validated the operator's raw --max-cost-usd/
# --max-wall-clock-seconds/--max-generated-tasks via spend_gate_check_all,
# then silently discarded those values (a bare `shift 2`) instead of ever
# passing them to run-lifecycle.sh's own --limits policy, so every real
# dispatch fell through to the scenario package's resource_policy default
# instead. This section proves the fix with the REAL run-lifecycle.sh (not
# a stub) executed through a thin logging wrapper -- the sole thing under
# test is whether the driver's own invocation carries --limits and whether
# the resulting real I-07 record reflects the OPERATOR's tighter values, not
# the scenario's own (deliberately larger) resource_policy defaults.
# ===========================================================================
UAT_R2_01_WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR" "$DRIVER_WORKDIR" "$CMP_DRIVER_WORKDIR" "$UAT_R2_01_WORKDIR"' EXIT
mkdir -p "$UAT_R2_01_WORKDIR/bin"

# A minimal real shark stub: ROOT-001 resolves directly to itself (no fork),
# spawn_agent once, adapter returns a final pass with zero cost, then
# release. Enough for run-lifecycle.sh's own dispatch loop to run to
# completion for real, with no provider/network call (LIFECYCLE_ADAPTER
# below is a local fixture script, not a real provider).
cat >"$UAT_R2_01_WORKDIR/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
python3 - "$@" <<'PY'
import hashlib, json, sys
args = sys.argv[1:]
if args[:2] == ["next", "ROOT-001"]:
    prompt = "work ROOT-001\n"
    response = {
        "entity_key": "ROOT-001", "entity_type": "task", "status": "development",
        "action": "spawn_agent", "agent_type": "developer", "provider": "fixture",
        "model": "fixture-model", "effort": "medium", "prompt": prompt,
        "prompt_sha256": hashlib.sha256(prompt.encode()).hexdigest(),
        "prompt_bytes": len(prompt.encode()), "resolved_via": ["ROOT-001"],
        "unresolved_placeholders": [], "error": "", "question_block": None,
        "current_responder": "",
    }
    path = args[args.index("--prompt-out") + 1]
    open(path, "wb").write(prompt.encode())
    print(json.dumps(response, separators=(",", ":")))
elif args[0] == "claim":
    print('{"session_id":"SID-uat-r2-01"}')
elif args[0] == "heartbeat":
    print('{"ok":true}')
elif args[:2] == ["status", "advance"]:
    print('{"advanced":true}')
elif args[0] == "release":
    print('{"released":true}')
else:
    raise SystemExit("unexpected shark argv: " + repr(args))
PY
SHARK
chmod +x "$UAT_R2_01_WORKDIR/bin/shark"

cat >"$UAT_R2_01_WORKDIR/adapter.sh" <<'ADAPTER'
#!/usr/bin/env bash
set -euo pipefail
python3 -c 'import json,sys; request=json.load(sys.stdin); print(json.dumps({"worker_id":"worker-uat-r2-01","session_id":request["session_id"],"kind":"final","recommended_outcome":"pass","cost_usd":0.0,"evidence":{"summary":"fixture"}}))'
ADAPTER
chmod +x "$UAT_R2_01_WORKDIR/adapter.sh"

# A thin logging wrapper substituted for RUN_LIFECYCLE_BIN (the same
# sibling-path override convention every driver in this suite already
# uses): logs the exact argv it was invoked with, snapshots the --limits
# file's content at invocation time (before the driver's own EXIT trap can
# remove it), execs the REAL run-lifecycle.sh unmodified, then snapshots the
# resulting real --output content too. Never mocks run-lifecycle.sh's own
# limits_from()/enforcement logic -- that stays real end to end.
cat >"$UAT_R2_01_WORKDIR/run-lifecycle-wrapper.sh" <<'WRAPPER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$SPY_ARGV_LOG"
limits_val="" output_val="" prev=""
for arg in "$@"; do
	[[ "$prev" == "--limits" ]] && limits_val="$arg"
	[[ "$prev" == "--output" ]] && output_val="$arg"
	prev="$arg"
done
if [[ -n "$limits_val" && -f "$limits_val" ]]; then
	cp "$limits_val" "$CAPTURED_LIMITS_FILE"
fi
rc=0
"$REAL_RUN_LIFECYCLE_BIN" "$@" || rc=$?
if [[ -n "$output_val" && -f "$output_val" ]]; then
	cp "$output_val" "$CAPTURED_LIFECYCLE_OUT"
fi
exit "$rc"
WRAPPER
chmod +x "$UAT_R2_01_WORKDIR/run-lifecycle-wrapper.sh"

# A real admitted scenario package (copied from the shared fixture, which
# already carries a real resource_policy block) with its resource_policy
# deliberately widened far past the operator ceilings this test passes on
# the CLI below -- the exact "CLI ceiling stricter than the scenario
# default" shape UAT-R2-01 names as the gap TC-080's existing coverage
# never constructed.
UAT_R2_01_SCENARIO_ID="scenario-uat-r2-01"
mkdir -p "$UAT_R2_01_WORKDIR/index/packages/$UAT_R2_01_SCENARIO_ID"
python3 - "$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml" \
	"$UAT_R2_01_WORKDIR/index/packages/$UAT_R2_01_SCENARIO_ID/package.yaml" "$UAT_R2_01_SCENARIO_ID" <<'PY'
import sys
import yaml

src, dst, scenario_id = sys.argv[1:4]
with open(src) as f:
    data = yaml.safe_load(f)
data["scenario_id"] = scenario_id
# Deliberately far larger than the operator's CLI ceilings below, so a
# record that reflects the SCENARIO default rather than the OPERATOR
# ceiling is unambiguously distinguishable.
data["resource_policy"] = {
    "max_cost_usd": 999.0,
    "max_wall_clock_seconds": 999999,
    "max_generated_tasks": 999,
}
with open(dst, "w") as f:
    yaml.safe_dump(data, f, sort_keys=False)
PY
cat >"$UAT_R2_01_WORKDIR/index/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$UAT_R2_01_SCENARIO_ID
EOF

UAT_R2_01_OPERATOR_COST=5
UAT_R2_01_OPERATOR_WALL=600
UAT_R2_01_OPERATOR_TASKS=10

verify_operator_limits_won() {
	# verify_operator_limits_won <label> <limits_file> <lifecycle_out>
	local label="$1" limits_file="$2" lifecycle_out="$3"
	[[ -f "$limits_file" ]] || fail "$label: no --limits file was captured from the real run-lifecycle.sh invocation"
	[[ -f "$lifecycle_out" ]] || fail "$label: no real I-07 output was captured from the real run-lifecycle.sh invocation"
	python3 -c '
import json, sys
import yaml

label, limits_path, lifecycle_path, cost, wall, tasks = sys.argv[1:7]
cost, wall, tasks = float(cost), float(wall), int(tasks)

with open(limits_path) as f:
    limits = yaml.safe_load(f)
got_cost = limits["max_cost_usd"]
got_wall = limits["max_wall_clock_seconds"]
got_tasks = limits["max_generated_tasks"]
assert float(got_cost) == cost, label + ": materialized --limits file max_cost_usd=" + repr(got_cost) + ", expected the operator ceiling " + repr(cost)
assert float(got_wall) == wall, label + ": materialized --limits file max_wall_clock_seconds=" + repr(got_wall) + ", expected " + repr(wall)
assert int(got_tasks) == tasks, label + ": materialized --limits file max_generated_tasks=" + repr(got_tasks) + ", expected " + repr(tasks)

with open(lifecycle_path) as f:
    record = json.loads(f.readline())
recorded = record["limits"]
rec_cost = recorded["max_cost_usd"]
rec_wall = recorded["max_wall_clock_seconds"]
rec_tasks = recorded["max_generated_tasks"]
assert float(rec_cost) == cost, label + ": I-07 recorded max_cost_usd=" + repr(rec_cost) + ", expected the operator ceiling " + repr(cost) + " (not the scenario resource_policy default 999.0)"
assert float(rec_wall) == wall, label + ": I-07 recorded max_wall_clock_seconds=" + repr(rec_wall) + ", expected " + repr(wall) + " (not the scenario default 999999)"
assert int(rec_tasks) == tasks, label + ": I-07 recorded max_generated_tasks=" + repr(rec_tasks) + ", expected " + repr(tasks) + " (not the scenario default 999)"
' "$label" "$limits_file" "$lifecycle_out" "$UAT_R2_01_OPERATOR_COST" "$UAT_R2_01_OPERATOR_WALL" "$UAT_R2_01_OPERATOR_TASKS" \
		|| fail "$label: operator ceiling did not win over the scenario resource_policy default"
}

# ---------------------------------------------------------------------------
# Batch driver (T-E40-F10-004): real run-lifecycle-batch.sh --mode pilot,
# real spend-gate.sh sourcing, real dispatch_pair -- i05_bundle_dir is left
# unconfigured on purpose so the pair is honestly classified `failed` AFTER
# the real run-lifecycle.sh dispatch already completed (exit 4, mirroring
# this file's own "fully-satisfied" convention above); the wrapper has
# already captured the real argv/--limits/--output by then.
# ---------------------------------------------------------------------------
BATCH_R2_WORKDIR="$UAT_R2_01_WORKDIR/batch"
mkdir -p "$BATCH_R2_WORKDIR/scratch"
cat >"$BATCH_R2_WORKDIR/policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$UAT_R2_01_WORKDIR/index/scenarios.yaml"
scenarios:
  $UAT_R2_01_SCENARIO_ID:
    root_key: "ROOT-001"
    scratch_root: "$BATCH_R2_WORKDIR/scratch"
    reps: 1
EOF

: >"$UAT_R2_01_WORKDIR/batch-spy-argv.log"
batch_r2_rc=0
PATH="$UAT_R2_01_WORKDIR/bin:$ORIGINAL_PATH" \
	LIFECYCLE_ADAPTER="$UAT_R2_01_WORKDIR/adapter.sh" \
	RUN_LIFECYCLE_BIN="$UAT_R2_01_WORKDIR/run-lifecycle-wrapper.sh" \
	REAL_RUN_LIFECYCLE_BIN="$SCRIPTS_DIR/run-lifecycle.sh" \
	SPY_ARGV_LOG="$UAT_R2_01_WORKDIR/batch-spy-argv.log" \
	CAPTURED_LIMITS_FILE="$UAT_R2_01_WORKDIR/batch-captured-limits.yaml" \
	CAPTURED_LIFECYCLE_OUT="$UAT_R2_01_WORKDIR/batch-captured-lifecycle.jsonl" \
	"$BATCH" --batch "$BATCH_R2_WORKDIR/policy.yaml" --retention-root "$BATCH_R2_WORKDIR/retention" \
	--mode pilot --acknowledge-provider-spend \
	--max-cost-usd "$UAT_R2_01_OPERATOR_COST" --max-wall-clock-seconds "$UAT_R2_01_OPERATOR_WALL" \
	--max-generated-tasks "$UAT_R2_01_OPERATOR_TASKS" \
	>"$UAT_R2_01_WORKDIR/batch-r2.out" 2>"$UAT_R2_01_WORKDIR/batch-r2.err" || batch_r2_rc=$?
[[ "$batch_r2_rc" -eq 4 ]] || fail "uat-r2-01 batch: expected exit 4 (real dispatch reached, i05_bundle_dir deliberately unconfigured), got $batch_r2_rc; stdout: $(cat "$UAT_R2_01_WORKDIR/batch-r2.out"); stderr: $(cat "$UAT_R2_01_WORKDIR/batch-r2.err")"
grep -q "i05_bundle_not_configured" "$BATCH_R2_WORKDIR/retention/invalid/index.jsonl" \
	|| fail "uat-r2-01 batch: expected the pair to be classified i05_bundle_not_configured AFTER a real run-lifecycle.sh dispatch, got: $(cat "$BATCH_R2_WORKDIR/retention/invalid/index.jsonl" 2>/dev/null)"

grep -q -- "--limits" "$UAT_R2_01_WORKDIR/batch-spy-argv.log" \
	|| fail "uat-r2-01 batch: real run-lifecycle-batch.sh invocation of run-lifecycle.sh did not include --limits: $(cat "$UAT_R2_01_WORKDIR/batch-spy-argv.log")"
verify_operator_limits_won "uat-r2-01 batch" \
	"$UAT_R2_01_WORKDIR/batch-captured-limits.yaml" "$UAT_R2_01_WORKDIR/batch-captured-lifecycle.jsonl"

echo "TC-080(UAT-R2-01 regression, batch driver, T-E40-F10-004): real run-lifecycle-batch.sh --mode pilot invocation of the real run-lifecycle.sh carries the operator's CLI ceilings via --limits, and the real I-07 record's limits reflect those operator values -- not the scenario's own (much larger) resource_policy default -- PASS"

# ---------------------------------------------------------------------------
# Comparison driver (T-E40-F10-005): real run-review-comparison.sh --mode
# pilot, both gates dispatched for real (i05_bundle_dir left unconfigured on
# both so the run ends honestly at exit 4 -- comparison never attempted --
# only after each real run-lifecycle.sh dispatch already completed).
# ---------------------------------------------------------------------------
CMP_R2_WORKDIR="$UAT_R2_01_WORKDIR/cmp"
mkdir -p "$CMP_R2_WORKDIR/scratch"
cat >"$CMP_R2_WORKDIR/candidate.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$UAT_R2_01_SCENARIO_ID"
scenario_index: "$UAT_R2_01_WORKDIR/index/scenarios.yaml"
gates:
  qa:
    root_key: "ROOT-001"
    scratch_root: "$CMP_R2_WORKDIR/scratch"
  deep_review:
    root_key: "ROOT-001"
    scratch_root: "$CMP_R2_WORKDIR/scratch"
EOF

: >"$UAT_R2_01_WORKDIR/cmp-spy-argv.log"
cmp_r2_rc=0
PATH="$UAT_R2_01_WORKDIR/bin:$ORIGINAL_PATH" \
	LIFECYCLE_ADAPTER="$UAT_R2_01_WORKDIR/adapter.sh" \
	RUN_LIFECYCLE_BIN="$UAT_R2_01_WORKDIR/run-lifecycle-wrapper.sh" \
	REAL_RUN_LIFECYCLE_BIN="$SCRIPTS_DIR/run-lifecycle.sh" \
	SPY_ARGV_LOG="$UAT_R2_01_WORKDIR/cmp-spy-argv.log" \
	CAPTURED_LIMITS_FILE="$UAT_R2_01_WORKDIR/cmp-captured-limits.yaml" \
	CAPTURED_LIFECYCLE_OUT="$UAT_R2_01_WORKDIR/cmp-captured-lifecycle.jsonl" \
	"$COMPARISON" --candidate "$CMP_R2_WORKDIR/candidate.yaml" --retention-root "$CMP_R2_WORKDIR/retention" \
	--mode pilot --comparison-mode independent_frozen_candidate --acknowledge-provider-spend \
	--max-cost-usd "$UAT_R2_01_OPERATOR_COST" --max-wall-clock-seconds "$UAT_R2_01_OPERATOR_WALL" \
	--max-generated-tasks "$UAT_R2_01_OPERATOR_TASKS" \
	>"$UAT_R2_01_WORKDIR/cmp-r2.out" 2>"$UAT_R2_01_WORKDIR/cmp-r2.err" || cmp_r2_rc=$?
[[ "$cmp_r2_rc" -eq 4 ]] || fail "uat-r2-01 comparison: expected exit 4 (real gate dispatch reached, i05_bundle_dir deliberately unconfigured), got $cmp_r2_rc; stdout: $(cat "$UAT_R2_01_WORKDIR/cmp-r2.out"); stderr: $(cat "$UAT_R2_01_WORKDIR/cmp-r2.err")"
grep -q "one or both gates failed to dispatch/evaluate" "$UAT_R2_01_WORKDIR/cmp-r2.out" "$UAT_R2_01_WORKDIR/cmp-r2.err" \
	|| fail "uat-r2-01 comparison: expected the standard post-gate-dispatch failure message, got stdout: $(cat "$UAT_R2_01_WORKDIR/cmp-r2.out"); stderr: $(cat "$UAT_R2_01_WORKDIR/cmp-r2.err")"

# Both gates (qa, deep_review) must have carried --limits -- the fix is not
# satisfied by fixing only the first dispatch call.
[[ "$(grep -c -- "--limits" "$UAT_R2_01_WORKDIR/cmp-spy-argv.log")" -eq 2 ]] \
	|| fail "uat-r2-01 comparison: expected both the qa and deep_review real run-lifecycle.sh invocations to include --limits, got: $(cat "$UAT_R2_01_WORKDIR/cmp-spy-argv.log")"
verify_operator_limits_won "uat-r2-01 comparison" \
	"$UAT_R2_01_WORKDIR/cmp-captured-limits.yaml" "$UAT_R2_01_WORKDIR/cmp-captured-lifecycle.jsonl"

echo "TC-080(UAT-R2-01 regression, comparison driver, T-E40-F10-005): real run-review-comparison.sh --mode pilot invocations (both qa and deep_review gates) of the real run-lifecycle.sh carry the operator's CLI ceilings via --limits, and the real I-07 record's limits reflect those operator values -- not the scenario's own (much larger) resource_policy default -- PASS"
