#!/usr/bin/env bash
# TC-101 (test-plan.md TC-06/TC-06a/TC-06b; spec.md AC-F11-06, AC-F11-06a,
# AC-F11-06b; T-E40-F11-006 task spec "Spend-gated I-06 replay preparation
# flow").
#
# Exercises the NEW `e40-benchmark.sh prepare-replay` subcommand's own
# decision table (spec.md §7.2's "Invocation | Result" table, reproduced as
# TC-06 below) and the 12-class ceiling BVA (spec.md §7.3.2 / test-plan.md
# §7.3.2) through the real top-level CLI, never an internal Python
# function -- the same "no test may call an internal function directly"
# discipline spec.md §7.2 states for every seam.
#
# AC-F11-06a's own 8-seam per-revert mutation matrix (S1-S8, the fourth
# ceiling threaded through pilot/baseline/variant/preflight's own argv and
# projections) is T-E40-F11-003's production change and is NOT re-asserted
# here as a mutation sweep -- this feature's own call_plan.totals evidence
# field does not exist until T-E40-F11-012 lands, so there is nowhere yet
# to observe "accepted at S1 but dropped before S8" from the outside. What
# THIS file proves, that no earlier task could, is AC-F11-06a's S2 seam --
# `prepare-replay`'s own parser-optional/handler-required ceiling
# validation -- since that subcommand did not exist before this task. The
# 12-class hardening (AC-F11-06b) is exercised here via S2 because S2 is
# the only seam a ceiling-free (no-acknowledgement) invocation can reach at
# all without an OperatorError firing first.
#
# Caller-Path Contract (test-plan.md TC-06/06a/06b row): real
# `bench/scripts/e40-benchmark.sh prepare-replay` CLI invocations against a
# real `setup`-produced operator root and a real admitted scenario
# package. require_positive is never mocked -- every BVA case below drives
# the actual numeric parser through the CLI's own argv. Zero-provider-call
# proof uses the shared PATH-shim denial harness (schema-driven binary
# list, tests/lib/path-shim-denial.sh) plus a positive control using
# testdata/stubs/claude (STUB_CLAUDE_LOG), proving the counting mechanism
# itself is not vacuously blind.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"

fail() {
	echo "TC-101 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh missing or not executable"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
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

# ===========================================================================
# TC-06: the decision table (spec.md §7.2).
# ===========================================================================

# (a) No acknowledgement, no ceilings: preview printed, exit non-zero, zero
# provider calls, writes nothing outside the preview directory.
path_shim_denial_setup "$WORKDIR/deny-a"
BEFORE_A="$(find "$OPROOT" -type f | sort)"
OUT_A="$WORKDIR/a.out"
RC_A=0
PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" prepare-replay \
	--config "$CONFIG" --scenario "$SCENARIO" >"$OUT_A" 2>&1 || RC_A=$?
[[ "$RC_A" -ne 0 ]] || fail "(a) no-ack/no-ceilings: expected non-zero exit, got 0: $(cat "$OUT_A")"
# §7.3.2 C1c: must not be indistinguishable from argparse's own exit 2 or
# from a handler-required OperatorError (also 2 by default) -- the preview
# refusal uses a third, distinct code.
[[ "$RC_A" -ne 2 ]] || fail "(a) no-ack/no-ceilings: exit 2 collides with argparse/OperatorError -- §7.3.2 C1c requires a distinguishable refusal"
for field in proposed_provider_calls proposed_provider_call_count model provider effort resource_ceilings_required; do
	grep -q "\"$field\"" "$OUT_A" || fail "(a) preview JSON missing field: $field: $(cat "$OUT_A")"
done
path_shim_denial_assert_empty "(a) no-ack/no-ceilings preview" \
	|| fail "(a) no-ack/no-ceilings: zero-provider-call proof failed (denial harness recorded an invocation)"
AFTER_A="$(find "$OPROOT" -type f | sort)"
NEW_FILES_A="$(comm -13 <(printf '%s\n' "$BEFORE_A") <(printf '%s\n' "$AFTER_A"))"
[[ -n "$NEW_FILES_A" ]] || fail "(a) no-ack/no-ceilings: expected at least the preview artifact to be written"
while IFS= read -r new_file; do
	[[ -n "$new_file" ]] || continue
	case "$new_file" in
	"$OPROOT"/replay-preview/*) ;;
	*) fail "(a) no-ack/no-ceilings: wrote outside its preview directory: $new_file" ;;
	esac
done <<<"$NEW_FILES_A"
echo "TC-101(a): no-ack/no-ceilings previews (exit $RC_A), zero provider calls, writes confined to the preview directory -- PASS"

# (b) No acknowledgement, SOME ceilings supplied: supplying ceilings never
# substitutes for acknowledgement (§7.1 row 06) -- must refuse identically.
OUT_B="$WORKDIR/b.out"
RC_B=0
PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" prepare-replay \
	--config "$CONFIG" --scenario "$SCENARIO" --max-cost-usd 5 --max-provider-calls 5 \
	>"$OUT_B" 2>&1 || RC_B=$?
[[ "$RC_B" -eq "$RC_A" ]] || fail "(b) no-ack/some-ceilings: exit ($RC_B) differs from the no-ceilings preview refusal ($RC_A)"
grep -q '"proposed_provider_calls"' "$OUT_B" || fail "(b) no-ack/some-ceilings: expected the same preview to print: $(cat "$OUT_B")"
path_shim_denial_assert_empty "(b) no-ack/some-ceilings" \
	|| fail "(b) no-ack/some-ceilings: supplying ceilings must never authorize a provider call"
echo "TC-101(b): no-ack/some-ceilings still refuses (supplying ceilings never substitutes for acknowledgement) -- PASS"

# (c) Acknowledgement, each single ceiling absent in turn: OperatorError
# naming resource_policy.<field>, NOT an argparse error (§7.3.2 C1b).
declare -A CEILING_FIELD=(
	["--max-cost-usd"]="resource_policy.max_cost_usd"
	["--max-wall-clock-seconds"]="resource_policy.max_wall_clock_seconds"
	["--max-generated-tasks"]="resource_policy.max_generated_tasks"
	["--max-provider-calls"]="resource_policy.max_provider_calls"
)
build_ceilings_without() {
	local skip="$1" out=() i
	for ((i = 0; i < ${#GOOD_CEILINGS[@]}; i += 2)); do
		if [[ "${GOOD_CEILINGS[$i]}" != "$skip" ]]; then
			out+=("${GOOD_CEILINGS[$i]}" "${GOOD_CEILINGS[$((i + 1))]}")
		fi
	done
	printf '%s\n' "${out[@]}"
}
for flag in "${!CEILING_FIELD[@]}"; do
	mapfile -t others < <(build_ceilings_without "$flag")
	out_c="$WORKDIR/c-${flag#--}.out"
	rc_c=0
	path_shim_denial_setup "$WORKDIR/deny-c-${flag#--}"
	PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" prepare-replay \
		--config "$CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend "${others[@]}" \
		>"$out_c" 2>&1 || rc_c=$?
	[[ "$rc_c" -eq 2 ]] || fail "(c) ack, $flag absent: expected OperatorError exit 2, got $rc_c: $(cat "$out_c")"
	grep -q "${CEILING_FIELD[$flag]}" "$out_c" \
		|| fail "(c) ack, $flag absent: stderr does not name ${CEILING_FIELD[$flag]}: $(cat "$out_c")"
	if grep -q '"proposed_provider_calls"' "$out_c"; then
		fail "(c) ack, $flag absent: unexpectedly printed the no-ack preview instead of an OperatorError: $(cat "$out_c")"
	fi
	path_shim_denial_assert_empty "(c) ack, $flag absent" \
		|| fail "(c) ack, $flag absent: a missing ceiling must never reach a provider call"
done
echo "TC-101(c): ack + each single absent ceiling -> OperatorError naming resource_policy.<field>, not argparse, zero provider calls -- PASS"

# (d) Acknowledgement, all four ceilings strictly positive: proceeds past
# the gate (the deep artifact/verification assertions are TC-102's own).
OUT_D="$WORKDIR/d.out"
LOG_D="$WORKDIR/d-claude.log"
RC_D=0
PATH="$STUBBIN:$ORIGINAL_PATH" STUB_CLAUDE_LOG="$LOG_D" "$OPERATOR" prepare-replay \
	--config "$CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend "${GOOD_CEILINGS[@]}" \
	>"$OUT_D" 2>&1 || RC_D=$?
[[ "$RC_D" -eq 0 ]] || fail "(d) ack + all four ceilings: expected exit 0, got $RC_D: $(cat "$OUT_D")"
[[ -f "$(cat "$OUT_D")" ]] || fail "(d) ack + all four ceilings: stdout is not an existing result path: $(cat "$OUT_D")"
[[ -s "$LOG_D" ]] || fail "(d) ack + all four ceilings: positive control -- stub claude recorded no invocation"
echo "TC-101(d): ack + all four positive ceilings proceeds past the gate and prints an existing result path -- PASS"

# C1a (§7.3.2): the execution seams' ceilings stay required=True and
# unchanged by this task -- omitting the fourth ceiling on pilot/baseline/
# variant is still an argparse error, exit 2, never reaching the handler.
# Minimal argv: only enough to reach add_execution_arguments' own parsing,
# never a real dispatch (the missing required flag fails before that).
for seam in pilot baseline variant; do
	out_c1a="$WORKDIR/c1a-$seam.out"
	rc_c1a=0
	"$OPERATOR" "$seam" --config "$CONFIG" --run-id "tc101-c1a-$seam" \
		--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 \
		>"$out_c1a" 2>&1 || rc_c1a=$?
	[[ "$rc_c1a" -eq 2 ]] || fail "C1a/$seam: expected argparse exit 2 for an omitted --max-provider-calls, got $rc_c1a: $(cat "$out_c1a")"
	grep -q "required" "$out_c1a" || fail "C1a/$seam: expected an argparse 'required' rejection: $(cat "$out_c1a")"
done
echo "TC-101(BVA C1a): --max-provider-calls omitted on pilot/baseline/variant is still an unchanged argparse exit 2 -- PASS"

# ===========================================================================
# AC-F11-06a/§2.3.6: proposed_provider_calls reflects the applicable-and-
# unreplayed prelude-stage set, and --max-provider-calls BOUNDS it (not
# merely validated as positive). No admitted, committed package can
# exercise this: §2.3.3a's own point is that a package passing the replay
# requirement always yields fixed_prelude_calls == 0 (every admitted
# package's bundle already covers every applicable stage). A synthetic
# fixture -- a copy of py-feature-recurring-tasks with two stages' bundle
# entries removed -- is required to observe either half of this property.
# ===========================================================================
SCRATCH_SCENARIOS="$WORKDIR/scratch-scenarios"
SCRATCH_PKG="$SCRATCH_SCENARIOS/pkg"
mkdir -p "$SCRATCH_PKG"
cp -r "$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks/." "$SCRATCH_PKG/"
python3 - "$SCRATCH_PKG/evaluator/replay/reference-bundle.json" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path) as f:
    bundle = json.load(f)
bundle["entries"] = [e for e in bundle["entries"] if e.get("stage") not in ("D01", "D02")]
with open(path, "w") as f:
    json.dump(bundle, f)
PY
cat >"$SCRATCH_SCENARIOS/scenarios.yaml" <<'EOF'
schema_version: "1.0"
scenarios:
  - pkg
EOF
SCRATCH_CONFIG="$WORKDIR/scratch-config.yaml"
python3 - "$CONFIG" "$SCRATCH_CONFIG" "$SCRATCH_SCENARIOS/scenarios.yaml" <<'PY'
import sys

import yaml

config_path, out_path, index_path = sys.argv[1:4]
with open(config_path) as f:
    config = yaml.safe_load(f)
config["scenario_index"] = index_path
with open(out_path, "w") as f:
    yaml.safe_dump(config, f)
PY

OUT_NONZERO="$WORKDIR/nonzero-preview.out"
RC_NONZERO=0
"$OPERATOR" prepare-replay --config "$SCRATCH_CONFIG" --scenario "$SCENARIO" \
	>"$OUT_NONZERO" 2>&1 || RC_NONZERO=$?
[[ "$RC_NONZERO" -ne 0 ]] || fail "nonzero preview: expected non-zero (preview-only) exit, got 0: $(cat "$OUT_NONZERO")"
python3 - "$OUT_NONZERO" <<'PY'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
assert doc["proposed_provider_call_count"] == 2, doc
assert sorted(doc["proposed_provider_calls"]) == ["D01", "D02"], doc
PY
echo "TC-101(nonzero preview): a package missing two stages' bundle entries previews proposed_provider_call_count=2, naming D01/D02 -- PASS"

OUT_BOUND_REJECT="$WORKDIR/bound-reject.out"
RC_BOUND_REJECT=0
path_shim_denial_setup "$WORKDIR/deny-bound-reject"
PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" prepare-replay \
	--config "$SCRATCH_CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 --max-provider-calls 1 \
	>"$OUT_BOUND_REJECT" 2>&1 || RC_BOUND_REJECT=$?
[[ "$RC_BOUND_REJECT" -ne 0 ]] || fail "bound reject: expected refusal when 2 proposed calls exceed --max-provider-calls=1, got 0"
grep -q "exceed resource_policy.max_provider_calls" "$OUT_BOUND_REJECT" \
	|| fail "bound reject: expected the bounding error naming resource_policy.max_provider_calls: $(cat "$OUT_BOUND_REJECT")"
path_shim_denial_assert_empty "bound reject" \
	|| fail "bound reject: exceeding the ceiling must never reach a provider call"
echo "TC-101(AC-F11-06a bound): --max-provider-calls=1 refuses when 2 provider calls are proposed, zero calls made -- PASS"

OUT_BOUND_ACCEPT="$WORKDIR/bound-accept.out"
LOG_BOUND_ACCEPT="$WORKDIR/bound-accept-claude.log"
RC_BOUND_ACCEPT=0
PATH="$STUBBIN:$ORIGINAL_PATH" STUB_CLAUDE_LOG="$LOG_BOUND_ACCEPT" "$OPERATOR" prepare-replay \
	--config "$SCRATCH_CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 --max-provider-calls 5 \
	>"$OUT_BOUND_ACCEPT" 2>&1 || RC_BOUND_ACCEPT=$?
[[ "$RC_BOUND_ACCEPT" -eq 0 ]] || fail "bound accept: expected proceeding when 2 proposed calls are within --max-provider-calls=5, got $RC_BOUND_ACCEPT: $(cat "$OUT_BOUND_ACCEPT")"
[[ -s "$LOG_BOUND_ACCEPT" ]] || fail "bound accept: expected the genuine dispatch to actually run"
echo "TC-101(AC-F11-06a bound): --max-provider-calls=5 proceeds when 2 provider calls are proposed -- PASS"

# ===========================================================================
# TC-06b: the 12-class ceiling BVA (spec.md §7.3.2), applied through
# require_positive's own two branches (integer=False for cost/wall-clock,
# integer=True for generated-tasks/provider-calls) -- the branch, not the
# flag name, is what determines each class's behavior, so one table per
# branch covers all four ceilings without four textually-identical copies.
# ===========================================================================

# reject_case <label> <flag> <value> <message_substring>
reject_case() {
	local label="$1" flag="$2" value="$3" want="$4"
	local others=() i
	for ((i = 0; i < ${#GOOD_CEILINGS[@]}; i += 2)); do
		if [[ "${GOOD_CEILINGS[$i]}" != "$flag" ]]; then
			others+=("${GOOD_CEILINGS[$i]}" "${GOOD_CEILINGS[$((i + 1))]}")
		fi
	done
	local out_file="$WORKDIR/bva-reject-$(echo "$flag$label" | tr -cd 'A-Za-z0-9_').out"
	local rc=0
	path_shim_denial_setup "$WORKDIR/deny-$(echo "$flag$label" | tr -cd 'A-Za-z0-9_')"
	# "$flag=$value" (one token) avoids argparse's negative-number-like-flag
	# ambiguity for values such as -inf that its own heuristic cannot tell
	# apart from an unrecognized option.
	PATH="$PATH_SHIM_DENIAL_BIN_DIR:$ORIGINAL_PATH" "$OPERATOR" prepare-replay \
		--config "$CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend \
		"${others[@]}" "$flag=$value" >"$out_file" 2>&1 || rc=$?
	[[ "$rc" -eq 2 ]] || fail "$flag/$label ($value): expected OperatorError exit 2, got $rc: $(cat "$out_file")"
	grep -q "$want" "$out_file" || fail "$flag/$label ($value): expected message containing '$want': $(cat "$out_file")"
	path_shim_denial_assert_empty "$flag/$label" || fail "$flag/$label ($value): rejected ceiling must never reach a provider call"
}

# accept_case <label> <flag> <value>
accept_case() {
	local label="$1" flag="$2" value="$3"
	local others=() i
	for ((i = 0; i < ${#GOOD_CEILINGS[@]}; i += 2)); do
		if [[ "${GOOD_CEILINGS[$i]}" != "$flag" ]]; then
			others+=("${GOOD_CEILINGS[$i]}" "${GOOD_CEILINGS[$((i + 1))]}")
		fi
	done
	local out_file="$WORKDIR/bva-accept-$(echo "$flag$label" | tr -cd 'A-Za-z0-9_').out"
	local log_file="$WORKDIR/bva-accept-$(echo "$flag$label" | tr -cd 'A-Za-z0-9_').log"
	local rc=0
	PATH="$STUBBIN:$ORIGINAL_PATH" STUB_CLAUDE_LOG="$log_file" "$OPERATOR" prepare-replay \
		--config "$CONFIG" --scenario "$SCENARIO" --acknowledge-provider-spend \
		"${others[@]}" "$flag=$value" >"$out_file" 2>&1 || rc=$?
	[[ "$rc" -eq 0 ]] || fail "$flag/$label ($value): expected acceptance (exit 0), got $rc: $(cat "$out_file")"
}

# Common reject classes (C2-C9), identical for every ceiling regardless of
# its integer=True/False branch.
for flag in --max-cost-usd --max-wall-clock-seconds --max-generated-tasks --max-provider-calls; do
	reject_case "boolean" "$flag" "true" "must be numeric and strictly positive"
	reject_case "nonnumeric" "$flag" "abc" "must be numeric and strictly positive"
	reject_case "negative" "$flag" "-1" "must be strictly positive"
	reject_case "zero" "$flag" "0" "must be strictly positive"
	reject_case "nan" "$flag" "nan" "must be a finite number"
	reject_case "posinf" "$flag" "inf" "must be a finite number"
	reject_case "overflow" "$flag" "1e400" "must be a finite number"
	reject_case "neginf" "$flag" "-inf" "must be a finite number"
done
echo "TC-101(BVA C2-C9): boolean/non-numeric/negative/zero/NaN/+inf/overflow/-inf rejected identically across all four ceilings -- PASS"

# C10: fractional under integer=True -- rejected ONLY for the two integer
# ceilings; for the two float ceilings the identical value "2.5" is a
# perfectly ordinary (C12) accepted input, not a class 10 case at all
# (require_positive's integer= branch, not the flag name, decides this).
reject_case "fractional" "--max-generated-tasks" "2.5" "must be a whole number, got 2.5"
reject_case "fractional" "--max-provider-calls" "2.5" "must be a whole number, got 2.5"
accept_case "fractional-is-valid-for-float-ceiling" "--max-cost-usd" "2.5"
accept_case "fractional-is-valid-for-float-ceiling" "--max-wall-clock-seconds" "2.5"
echo "TC-101(BVA C10): fractional value rejected under integer=True, accepted under integer=False -- PASS"

# C11: lexical variants (accepted deliberately, AC-F11-06b) -- surrounding
# whitespace and the PEP 515 digit separator, for all four ceilings.
for flag in --max-cost-usd --max-wall-clock-seconds --max-generated-tasks --max-provider-calls; do
	accept_case "lexical-whitespace" "$flag" " 3 "
	accept_case "lexical-separator" "$flag" "1_0"
done
echo "TC-101(BVA C11): lexical variants (' 3 ', '1_0') accepted for all four ceilings -- PASS"

# C12: minimum positive / typical accepted values, split by branch (an
# integer ceiling's minimum positive value is 1; a float ceiling also
# accepts a small positive fraction, which C10's own float half already
# proved -- this adds the plain-integer/typical-float shape).
accept_case "typical" "--max-cost-usd" "5.0"
accept_case "typical" "--max-wall-clock-seconds" "5.0"
accept_case "minimum" "--max-generated-tasks" "1"
accept_case "minimum" "--max-provider-calls" "1"
echo "TC-101(BVA C12): minimum-positive/typical values accepted for all four ceilings -- PASS"

echo "TC-101: spend-gated I-06 replay preparation preview/gate decision table and 12-class ceiling BVA -- ALL PASS"
