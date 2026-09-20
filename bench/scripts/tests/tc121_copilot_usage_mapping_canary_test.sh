#!/usr/bin/env bash
# TC-F13-08 (test-plan.md AC-F13-08; T-E40-F13-005 task spec).
#
# Exercises AC-F13-08 (REQ-F-006, REQ-NF-002, X-09): the canary
# (bench/scripts/canary-copilot-usagemapping.sh) re-verifies bench/evidence/
# usage-mapping.yaml against captured Copilot CLI JSONL provider envelopes,
# detecting both drift classes as distinct, correctly-attributed failures:
#   (a) a mapped envelope slot missing or drifted in the envelope
#   (b) the envelope source itself (the transcript's ---STDOUT--- block)
#       becoming undecodable or non-JSON
#
# Caller-Path Contract: real subprocess invocation of
# `bench/scripts/canary-copilot-usagemapping.sh [--transcript <path>]`, real
# YAML parse of usage-mapping.yaml, and JSONL stream validation.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

CANARY="$SCRIPTS_DIR/canary-copilot-usagemapping.sh"
CLEAN_FIXTURE="$SCRIPTS_DIR/testdata/usagemapping/copilot/clean-transcript.log"
DRIFTED_FIXTURE="$SCRIPTS_DIR/testdata/usagemapping/copilot/drifted-cache-read-input-tokens.log"
NONJSON_FIXTURE="$SCRIPTS_DIR/testdata/usagemapping/copilot/non-json-transcript.log"

fail() {
	echo "TC-F13-08 FAIL: $1" >&2
	exit 1
}

[[ -x "$CANARY" ]] || fail "canary-copilot-usagemapping.sh missing or not executable: $CANARY"
[[ -f "$CLEAN_FIXTURE" ]] || fail "clean fixture missing: $CLEAN_FIXTURE"
[[ -f "$DRIFTED_FIXTURE" ]] || fail "drifted fixture missing: $DRIFTED_FIXTURE"
[[ -f "$NONJSON_FIXTURE" ]] || fail "non-JSON fixture missing: $NONJSON_FIXTURE"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Test 1 (agreeing case): against committed clean fixture under
# testdata/usagemapping/copilot/, all github_copilot_cli slots resolve and the
# canary exits 0 with no --transcript override.
# ---------------------------------------------------------------------------
test_agreeing_default() {
	local out="$WORKDIR/agreeing.out" err="$WORKDIR/agreeing.err"
	set +e
	"$CANARY" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "agreeing-default: canary exited $code, want 0: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == "PASS" ]] || fail "agreeing-default: last stderr line was $(printf '%q' "$last_line"), want PASS: $(cat "$err")"

	echo "TC-F13-08(agreeing-default) PASS"
}

# ---------------------------------------------------------------------------
# Test 2: --transcript accepts an operator-supplied transcript path.
# ---------------------------------------------------------------------------
test_operator_transcript() {
	local out="$WORKDIR/operator.out" err="$WORKDIR/operator.err"
	set +e
	"$CANARY" --transcript "$CLEAN_FIXTURE" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "operator-transcript: canary --transcript exited $code, want 0: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == "PASS" ]] || fail "operator-transcript: last stderr line was $(printf '%q' "$last_line"), want PASS: $(cat "$err")"

	echo "TC-F13-08(operator-transcript) PASS"
}

# ---------------------------------------------------------------------------
# Test 3 (drifted case): against a fixture with cache_read missing from
# promptCacheBreakState, canary fails naming cache_read_input_tokens and
# usage_slot_unavailable.
# ---------------------------------------------------------------------------
test_drifted_field() {
	local out="$WORKDIR/drifted.out" err="$WORKDIR/drifted.err"
	set +e
	"$CANARY" --transcript "$DRIFTED_FIXTURE" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "drifted: canary exited 0 against fixture missing cache_read"

	grep -qF "cache_read_input_tokens" "$err" || fail "drifted: cache_read_input_tokens not named: $(cat "$err")"
	grep -qF "usage.cache_read_input_tokens" "$err" || fail "drifted: envelope path not named: $(cat "$err")"
	grep -qF "usage_slot_unavailable" "$err" || fail "drifted: usage_slot_unavailable not in error: $(cat "$err")"
	! grep -qF "envelope_source_unavailable" "$err" || fail "drifted: misattributed as envelope_source_unavailable: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == FAIL:* ]] || fail "drifted: last stderr line was $(printf '%q' "$last_line"), want FAIL: line"

	echo "TC-F13-08(drifted-field) PASS"
}

# ---------------------------------------------------------------------------
# Test 4 (non-JSON case): against a transcript whose STDOUT is non-JSON prose,
# canary fails as envelope_source_unavailable.
# ---------------------------------------------------------------------------
test_envelope_unavailable() {
	local out="$WORKDIR/nonjson.out" err="$WORKDIR/nonjson.err"
	set +e
	"$CANARY" --transcript "$NONJSON_FIXTURE" >"$out" 2>"$err"
	local code=$?
	set -e
	[[ "$code" -ne 0 ]] || fail "nonjson: canary exited 0 against non-JSON transcript"

	grep -qF "envelope_source_unavailable" "$err" || fail "nonjson: envelope_source_unavailable not found: $(cat "$err")"
	grep -qF "$NONJSON_FIXTURE" "$err" || fail "nonjson: transcript path not named: $(cat "$err")"

	local per_slot_count
	per_slot_count="$(grep -cF "usage_slot_unavailable" "$err" || true)"
	[[ "$per_slot_count" -eq 0 ]] || fail "nonjson: reported per-slot failures instead of whole-source failure: $(cat "$err")"

	local last_line
	last_line="$(tail -n1 "$err")"
	[[ "$last_line" == FAIL:* ]] || fail "nonjson: last stderr line was $(printf '%q' "$last_line"), want FAIL: line"

	echo "TC-F13-08(envelope-unavailable) PASS"
}

test_agreeing_default
test_operator_transcript
test_drifted_field
test_envelope_unavailable

echo "TC-F13-08 PASS"
