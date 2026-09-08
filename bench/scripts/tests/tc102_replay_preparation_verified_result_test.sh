#!/usr/bin/env bash
# TC-102 (test-plan.md TC-07/TC-08; spec.md AC-F11-07, AC-F11-08;
# T-E40-F11-006 task spec "Spend-gated I-06 replay preparation flow").
#
# TC-07 exercises the acknowledged, fully-ceilinged `prepare-replay` path
# through the real top-level CLI: the genuine replay producer
# (run-prelude.sh, wrapping X-10's Rider product-design action) runs
# against the real PATH-stubbed `claude` (offline, per §7.7's positive
# control), its result is verified by the real, unmodified
# `verify-replay-result.sh` against the package's admitted
# `replay_reference` bundle, and the verified absolute result path is
# printed. A distinct negative case proves a result that FAILS
# verification prints no path and exits non-zero.
#
# TC-08 exercises AC-F11-08's closed inventory (spec.md §7.1a row 08, N=11):
# `preflight`'s own four-flag parser surface plus the three named
# prepare-replay/setup-only flags that must remain unreachable from it.
#
# Caller-Path Contract: real `bench/scripts/e40-benchmark.sh` CLI
# invocations against a real `setup`-produced operator root and the real
# `py-feature-recurring-tasks` admitted package. verify-replay-result.sh is
# never mocked -- both the positive case (this file re-invokes it
# independently over prepare-replay's own printed result/bundle, never
# trusting prepare-replay's own validation.json artifact at face value)
# and the negative case exercise the real, unmodified script.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
VALIDATOR="$SCRIPTS_DIR/verify-replay-result.sh"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"
FEATURE_PKG="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks/package.yaml"
REFERENCE_BUNDLE="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json"

fail() {
	echo "TC-102 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh missing or not executable"
[[ -x "$VALIDATOR" ]] || fail "verify-replay-result.sh missing or not executable"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
[[ -f "$FEATURE_PKG" ]] || fail "seed package missing: $FEATURE_PKG"
[[ -f "$REFERENCE_BUNDLE" ]] || fail "reference bundle missing: $REFERENCE_BUNDLE"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

# shellcheck source=lib/path-shim-denial.sh
source "$SCRIPT_DIR/lib/path-shim-denial.sh"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
ORIGINAL_PATH="$PATH"

OPROOT="$WORKDIR/operator"
SETUP_OUT="$WORKDIR/setup.out"
"$OPERATOR" setup --out "$OPROOT" >"$SETUP_OUT" 2>&1 || fail "setup: $(cat "$SETUP_OUT")"
CONFIG="$OPROOT/e40-demo.yaml"
[[ -f "$CONFIG" ]] || fail "setup did not produce a config: $CONFIG"

SCENARIO="py-feature-recurring-tasks"
GOOD_CEILINGS=(--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 --max-provider-calls 5)

json_field() {
	python3 -c "import json,sys; print(json.load(open(sys.argv[1])).get(sys.argv[2], ''))" "$1" "$2"
}

# ===========================================================================
# TC-07 (positive): ack + all four ceilings -> genuine producer runs
# against the real bundle-consistency/dispatch path, verified, absolute
# path printed. Five retained artifacts all present.
# ===========================================================================
OUT_POS="$WORKDIR/pos.out"
LOG_POS="$WORKDIR/pos-claude.log"
RC_POS=0
PATH="$STUBBIN:$ORIGINAL_PATH" STUB_CLAUDE_LOG="$LOG_POS" "$OPERATOR" prepare-replay \
	--config "$CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend "${GOOD_CEILINGS[@]}" \
	>"$OUT_POS" 2>&1 || RC_POS=$?
[[ "$RC_POS" -eq 0 ]] || fail "TC-07 positive: expected exit 0, got $RC_POS: $(cat "$OUT_POS")"

RESULT_PATH="$(tr -d '\n' <"$OUT_POS")"
[[ -n "$RESULT_PATH" ]] || fail "TC-07 positive: no path printed on stdout"
[[ "$RESULT_PATH" = /* ]] || fail "TC-07 positive: printed path is not absolute: $RESULT_PATH"
[[ -f "$RESULT_PATH" ]] || fail "TC-07 positive: printed path does not exist: $RESULT_PATH"

ATTEMPT_DIR="$(dirname "$RESULT_PATH")"
for artifact in result.json transcript.txt usage.json limits.json validation.json; do
	[[ -f "$ATTEMPT_DIR/$artifact" ]] || fail "TC-07 positive: retained attempt is missing $artifact under $ATTEMPT_DIR"
done
case "$ATTEMPT_DIR" in
"$OPROOT"/replay/"$SCENARIO"/*) ;;
*) fail "TC-07 positive: attempt directory is not under <operator_root>/replay/<scenario_id>/: $ATTEMPT_DIR" ;;
esac

[[ -s "$LOG_POS" ]] || fail "TC-07 positive: positive control -- stub claude recorded no invocation"

# Never trust prepare-replay's own validation.json at face value -- recompute
# by re-invoking the real, unmodified verify-replay-result.sh independently
# over the printed result path and the package's own bundle.
INDEPENDENT_VERIFY_OUT="$WORKDIR/independent-verify.out"
INDEPENDENT_VERIFY_RC=0
"$VALIDATOR" "$RESULT_PATH" "$REFERENCE_BUNDLE" >"$INDEPENDENT_VERIFY_OUT" 2>&1 || INDEPENDENT_VERIFY_RC=$?
[[ "$INDEPENDENT_VERIFY_RC" -eq 0 ]] \
	|| fail "TC-07 positive: independent re-verification of the printed result failed (rc=$INDEPENDENT_VERIFY_RC): $(cat "$INDEPENDENT_VERIFY_OUT")"

RESULT_TERMINAL_OUTCOME="$(json_field "$RESULT_PATH" terminal_outcome)"
[[ "$RESULT_TERMINAL_OUTCOME" == "complete" ]] \
	|| fail "TC-07 positive: expected terminal_outcome=complete for a package with an applicable prelude stage, got $RESULT_TERMINAL_OUTCOME"

USAGE_PROVIDER_CALLS="$(json_field "$ATTEMPT_DIR/usage.json" provider_calls)"
[[ "$USAGE_PROVIDER_CALLS" == "1" ]] \
	|| fail "TC-07 positive: usage.json provider_calls=$USAGE_PROVIDER_CALLS, want 1 (one genuine dispatch was attempted)"

echo "TC-102(TC-07 positive): genuine producer runs, five artifacts retained under <operator_root>/replay/<scenario>/<attempt>/, independent re-verification accepts, absolute path printed -- PASS"

# ===========================================================================
# TC-07 (negative): the producer exits 0, but the RESULT it wrote fails
# verify-replay-result.sh -- prepare-replay must exit non-zero and print no
# path. Constructed with a purpose-built stub `claude` that plants a
# conforming D0X artifact into its own cwd (pinned to the artifact root by
# run-prelude.sh) before returning -- an artifact whose consumed_entries[]
# is necessarily empty (run-prelude.sh's own write_complete_result never
# populates lineage), which the bundle's required=true entries for every
# D0X stage turn into a genuine "unattributed_artifact" verdict.
# ===========================================================================
BADBIN="$WORKDIR/badbin"
mkdir -p "$BADBIN"
cat >"$BADBIN/claude" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p docs/product
echo "# fabricated artifact, never resolver-attributed" >docs/product/D01-fabricated.md
cat <<'JSON'
{"result": "bad stub: plants an unattributed D01 artifact", "model": "claude-stub-model"}
JSON
STUB
chmod +x "$BADBIN/claude"

# stdout and stderr are captured SEPARATELY here (unlike the positive
# case's combined capture) because AC-F11-07's own requirement is specific
# to stdout: "a result failing verify-replay-result.sh -> non-zero and no
# path printed" names the stream, and a merged capture would only ever
# reflect whichever stream happened to interleave first.
OUT_NEG="$WORKDIR/neg.out"
ERR_NEG="$WORKDIR/neg.err"
RC_NEG=0
PATH="$BADBIN:$ORIGINAL_PATH" "$OPERATOR" prepare-replay \
	--config "$CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend "${GOOD_CEILINGS[@]}" \
	>"$OUT_NEG" 2>"$ERR_NEG" || RC_NEG=$?
[[ "$RC_NEG" -ne 0 ]] || fail "TC-07 negative: expected non-zero exit for a producer result failing verification, got 0: $(cat "$ERR_NEG")"
[[ ! -s "$OUT_NEG" ]] \
	|| fail "TC-07 negative: a path was printed on stdout despite a failing verification: $(cat "$OUT_NEG")"
grep -q "unattributed_artifact" "$ERR_NEG" \
	|| fail "TC-07 negative: expected verify-replay-result.sh's unattributed_artifact verdict to be surfaced: $(cat "$ERR_NEG")"

echo "TC-102(TC-07 negative): a producer result failing verify-replay-result.sh exits non-zero and prints no path -- PASS"

# ===========================================================================
# TC-08 (spec.md §7.1a row 08, N=11): preflight's parser declares exactly
# four options; prepare-replay's own flags (and setup's --scenario-index)
# are unreachable from it, proven as argparse rejections rather than
# merely unused.
# ===========================================================================
path_shim_denial_setup "$WORKDIR/deny-preflight"
BEFORE_PREFLIGHT="$(find "$OPROOT" -type d -name replay | sort)"

run_preflight_positive() {
	local label="$1"
	shift
	local out_file="$WORKDIR/preflight-${label}.out"
	local rc=0
	PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" preflight --config "$CONFIG" "$@" \
		>"$out_file" 2>&1 || rc=$?
	if grep -qE 'error: (unrecognized arguments|the following arguments are required)' "$out_file"; then
		fail "preflight/$label: parser rejected a declared flag combination: $(cat "$out_file")"
	fi
}

# --config alone; each of the other three added singly; all four together.
run_preflight_positive "config-alone"
run_preflight_positive "with-out" --out "$WORKDIR/preflight-out-a"
run_preflight_positive "with-reps" --reps 1
run_preflight_positive "with-scenario" --scenario "$SCENARIO"
run_preflight_positive "all-four" --out "$WORKDIR/preflight-out-b" --reps 1 --scenario "$SCENARIO"
echo "TC-102(TC-08 positive domain): preflight's own four declared flags (--config/--out/--reps/--scenario), singly and together, are never rejected by the parser -- PASS"

# The three unknown flags that would reach the replay/setup path: each must
# be an argparse rejection, proving the preparation path is UNREACHABLE
# from preflight, not merely unused.
declare -A UNREACHABLE_FLAGS=(
	["--acknowledge-provider-spend"]=""
	["--max-provider-calls"]="5"
	["--scenario-index"]="$BENCH_DIR/scenarios/scenarios.yaml"
)
for flag in "${!UNREACHABLE_FLAGS[@]}"; do
	value="${UNREACHABLE_FLAGS[$flag]}"
	out_file="$WORKDIR/preflight-unknown-$(echo "$flag" | tr -cd 'A-Za-z0-9_').out"
	rc=0
	if [[ -n "$value" ]]; then
		PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" preflight --config "$CONFIG" "$flag" "$value" \
			>"$out_file" 2>&1 || rc=$?
	else
		PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" preflight --config "$CONFIG" "$flag" \
			>"$out_file" 2>&1 || rc=$?
	fi
	[[ "$rc" -eq 2 ]] || fail "preflight/$flag: expected argparse exit 2 (unreachable), got $rc: $(cat "$out_file")"
	grep -q "unrecognized arguments" "$out_file" \
		|| fail "preflight/$flag: expected an 'unrecognized arguments' argparse rejection: $(cat "$out_file")"
done
echo "TC-102(TC-08 closed inventory): --acknowledge-provider-spend, --max-provider-calls, and --scenario-index are all argparse-unreachable from preflight -- PASS"

# AC-F11-08's own core clause: preflight never creates a replay result and
# never makes a provider call, in any of the above combinations.
path_shim_denial_assert_empty "TC-08 preflight sweep" \
	|| fail "TC-08: preflight made a provider call in at least one of the above invocations"
AFTER_PREFLIGHT="$(find "$OPROOT" -type d -name replay | sort)"
[[ "$BEFORE_PREFLIGHT" == "$AFTER_PREFLIGHT" ]] \
	|| fail "TC-08: a replay/ directory appeared under the operator root after the preflight sweep -- preflight must never create a replay result"

echo "TC-102: verified spend-gated replay preparation result, and preflight's closed AC-F11-08 flag inventory -- ALL PASS"
