#!/usr/bin/env bash
# TC-047 (test-plan.md AC test matrix row tc047; T-E40-F06-009 task spec
# Test Cases).
#
# Exercises AC-007 (REQ-F-009) and AC-021 (REQ-F-019): the X-09 canary
# (bench/scripts/canary-usagemapping.sh) re-verifies bench/evidence/
# usage-mapping.yaml against a real captured provider envelope, detecting
# both named REQ-F-019 drift classes as distinct, correctly-attributed
# failures -- (a) a mapped envelope path going missing, and (b) the
# envelope source itself (the transcript's ---STDOUT--- block) becoming
# undecodable -- never misreported as nine independent per-slot failures.
#
# Caller-Path Contract (test-plan.md row tc047): real subprocess invocation
# of `bench/scripts/canary-usagemapping.sh [--transcript <path>]`, a real
# YAML parse of usage-mapping.yaml, and a real JSON decode of each
# transcript's ---STDOUT--- block. The agreeing case runs against the
# committed envelope fixture under bench/scripts/testdata/run/
# clean-completed/ (already asserted envelope-shaped by F02); the drifted
# and non-JSON cases run against T-E40-F06-009's own committed mutated-copy
# fixtures under bench/scripts/testdata/usagemapping/ -- never a
# hand-authored fixture shaped to make the mapping agree, and never a
# JSON-payload-wrapped-in-a-string simulation of the non-JSON case.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

CANARY="$SCRIPTS_DIR/canary-usagemapping.sh"
DEFAULT_FIXTURE="$SCRIPTS_DIR/testdata/run/clean-completed/run/transcripts/1-in_development-anthropic.log"
DRIFTED_FIXTURE="$SCRIPTS_DIR/testdata/usagemapping/drifted-cache-read-input-tokens.log"
NONJSON_FIXTURE="$SCRIPTS_DIR/testdata/usagemapping/non-json-transcript.log"

fail() {
	echo "TC-047 FAIL: $1" >&2
	exit 1
}

[[ -x "$CANARY" ]] || fail "canary-usagemapping.sh missing or not executable: $CANARY"
[[ -f "$DEFAULT_FIXTURE" ]] || fail "committed envelope fixture missing: $DEFAULT_FIXTURE"
[[ -f "$DRIFTED_FIXTURE" ]] || fail "drifted envelope fixture missing: $DRIFTED_FIXTURE"
[[ -f "$NONJSON_FIXTURE" ]] || fail "non-JSON transcript fixture missing: $NONJSON_FIXTURE"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# AC-T1 (AC-007 agreeing case): against the committed envelope fixtures
# under bench/scripts/testdata/run/, every anthropic_claude_cli slot
# resolves and the canary exits 0 with no --transcript override.
# ---------------------------------------------------------------------------
test_agreeing() {
	local out="$WORKDIR/agreeing.out" err="$WORKDIR/agreeing.err"
	set +e
	"$CANARY" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "agreeing: canary-usagemapping.sh exited $code, want 0: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == "PASS" ]] || fail "agreeing: last stderr line was $(printf '%q' "$last_line"), want exactly PASS: $(cat "$err")"

	echo "TC-047(agreeing) PASS"
}

# ---------------------------------------------------------------------------
# AC-T2 (AC-007 drifted case): against a copy of the real-capture fixture
# with usage.cache_read_input_tokens deleted, the canary fails naming
# exactly that slot and envelope path (drift class (a)) -- not a generic
# mismatch message.
# ---------------------------------------------------------------------------
test_drifted_field() {
	local out="$WORKDIR/drifted.out" err="$WORKDIR/drifted.err"
	set +e
	"$CANARY" --transcript "$DRIFTED_FIXTURE" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "drifted: canary-usagemapping.sh exited 0 against a fixture with cache_read_input_tokens deleted"

	grep -qF "cache_read_input_tokens" "$err" || fail "drifted: slot name cache_read_input_tokens not named on stderr: $(cat "$err")"
	grep -qF "usage.cache_read_input_tokens" "$err" || fail "drifted: envelope path usage.cache_read_input_tokens not named on stderr: $(cat "$err")"
	grep -qF "usage_slot_unavailable" "$err" || fail "drifted: failure not attributed to usage_slot_unavailable (drift class a): $(cat "$err")"
	! grep -qF "envelope_source_unavailable" "$err" || fail "drifted: field-only drift misattributed as envelope_source_unavailable: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == FAIL:* ]] || fail "drifted: last stderr line was $(printf '%q' "$last_line"), want a 'FAIL: <field>' line"

	echo "TC-047(drifted-field) PASS"
}

# ---------------------------------------------------------------------------
# AC-T3: --transcript accepts an operator-supplied real transcript (smoke
# path only, test-plan.md AC-007 -- not asserted against a specific value
# beyond a clean exit against a genuine real-capture-shaped envelope).
# ---------------------------------------------------------------------------
test_operator_transcript() {
	local out="$WORKDIR/operator.out" err="$WORKDIR/operator.err"
	set +e
	"$CANARY" --transcript "$DEFAULT_FIXTURE" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "operator-transcript: canary-usagemapping.sh --transcript exited $code against a real envelope, want 0: $(cat "$err")"

	echo "TC-047(operator-transcript) PASS"
}

# ---------------------------------------------------------------------------
# AC-T4 (AC-021): against a transcript whose ---STDOUT--- block is
# non-JSON, the canary fails as ONE envelope_source_unavailable naming the
# transcript path -- never nine per-slot usage_slot_unavailable failures
# (drift class (b)).
# ---------------------------------------------------------------------------
test_envelope_unavailable() {
	local out="$WORKDIR/nonjson.out" err="$WORKDIR/nonjson.err"
	set +e
	"$CANARY" --transcript "$NONJSON_FIXTURE" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "nonjson: canary-usagemapping.sh exited 0 against a non-JSON STDOUT block"

	grep -qF "envelope_source_unavailable" "$err" || fail "nonjson: failure not attributed to envelope_source_unavailable (drift class b): $(cat "$err")"
	grep -qF "$NONJSON_FIXTURE" "$err" || fail "nonjson: transcript path not named on stderr: $(cat "$err")"

	local per_slot_count
	per_slot_count="$(grep -cF "usage_slot_unavailable" "$err" || true)"
	[[ "$per_slot_count" -eq 0 ]] || fail "nonjson: reported per-slot usage_slot_unavailable failures instead of one whole-source failure: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == FAIL:* ]] || fail "nonjson: last stderr line was $(printf '%q' "$last_line"), want a 'FAIL: <field>' line"
	[[ "$last_line" == *envelope_source_unavailable* ]] || fail "nonjson: last stderr line was $(printf '%q' "$last_line"), want envelope_source_unavailable named"

	echo "TC-047(envelope-unavailable) PASS"
}

test_agreeing
test_drifted_field
test_operator_transcript
test_envelope_unavailable

echo "TC-047 PASS"
