#!/usr/bin/env bash
# TC-081 / T-E40-F10-006: pilot-ledger attestation gate (spec.md REQ-F-005,
# ADR-F10-09; test-plan.md TC-081 full body).
#
# Exercises the REAL pilot-ledger.sh binary and the REAL
# run-lifecycle-batch.sh CLI (which sources the REAL lib/spend-gate.sh)
# over a temp retention root with real retained-artifact fixtures and real
# sha256 digest computation. Per TC-081's Caller-Path Contract: "Do not
# mock digest comparison, family lookup, or the baseline-mode gate check."
#
# Preconditions built below (TC-081 Preconditions): three scenario
# families -- family-a (scenario-a) has no pilot attestation; family-b
# (scenario-b) will get an attestation later invalidated by mutating a
# retained artifact; family-c (scenario-c) gets a verified, current
# attestation.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
PILOT_LEDGER="$SCRIPTS_DIR/pilot-ledger.sh"
BATCH="$SCRIPTS_DIR/run-lifecycle-batch.sh"
SCHEMA="$REPO_ROOT/bench/reports/lifecycle-baseline-schema.yaml"

fail() {
	echo "TC-081 FAIL: $1" >&2
	exit 1
}

[[ -x "$PILOT_LEDGER" ]] || fail "bench/scripts/pilot-ledger.sh missing or not executable"
[[ -x "$BATCH" ]] || fail "bench/scripts/run-lifecycle-batch.sh missing or not executable"
[[ -f "$SCHEMA" ]] || fail "bench/reports/lifecycle-baseline-schema.yaml missing"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

ROOT="$WORKDIR/retention"
mkdir -p "$ROOT"

# ---------------------------------------------------------------------------
# Scenario index + three minimal admitted-package fixtures (scenario-a/
# family-a, scenario-b/family-b, scenario-c/family-c). Only the fields the
# matrix/family derivation reads (scenario_id, entity_family) are required.
# ---------------------------------------------------------------------------
INDEX_DIR="$WORKDIR/scenario-index"
mkdir -p "$INDEX_DIR/packages/scenario-a" "$INDEX_DIR/packages/scenario-b" "$INDEX_DIR/packages/scenario-c"

cat >"$INDEX_DIR/scenarios.yaml" <<'EOF'
schema_version: "1.0"
scenarios:
  - packages/scenario-a
  - packages/scenario-b
  - packages/scenario-c
EOF

write_package() {
	local dir="$1" scenario_id="$2" family="$3"
	cat >"$dir/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$scenario_id"
scenario_version: "1"
entity_family: "$family"
EOF
}
write_package "$INDEX_DIR/packages/scenario-a" "scenario-a" "family-a"
write_package "$INDEX_DIR/packages/scenario-b" "scenario-b" "family-b"
write_package "$INDEX_DIR/packages/scenario-c" "scenario-c" "family-c"

cat >"$WORKDIR/batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$INDEX_DIR/scenarios.yaml"
EOF

# ---------------------------------------------------------------------------
# Retained-artifact fixtures under the retention root, one scenario/rep
# directory per family, populated with all eight canonical retention
# artifacts (spec.md Data model) so pilot-ledger.sh has real bytes to
# digest.
# ---------------------------------------------------------------------------
retain_fixture() {
	local scenario_id="$1"
	local dest="$ROOT/scenarios/$scenario_id/1"
	mkdir -p "$dest/evidence" "$dest/transcripts"
	cp "$INDEX_DIR/packages/$scenario_id/package.yaml" "$dest/package.yaml"
	echo '{"stage": "code", "note": "fixture evidence"}' >"$dest/evidence/stage.json"
	echo "fixture transcript for $scenario_id" >"$dest/transcripts/stage.txt"
	echo '{"root_key": "ROOT-001", "entries": []}' >"$dest/entity-history.json"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/lifecycle.jsonl"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/evaluation.jsonl"
	echo '{"held_back": true}' >"$dest/oracle.json"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1, \"artifacts\": {}}" >"$dest/manifest.json"
}
retain_fixture scenario-a
retain_fixture scenario-b
retain_fixture scenario-c

CHECKLIST="$WORKDIR/checklist.json"
echo '{"items": [{"id": "structural-spotcheck", "result": "pass"}]}' >"$CHECKLIST"

# ---------------------------------------------------------------------------
# Preconditions: attest family-b then invalidate it by mutating a retained
# artifact (ADR-F10-09: a later mutation invalidates a previously-verified
# attestation, not silently carried forward -- also covers TC-081's Edge
# Case about a reclaimed/re-run pair, which this same digest-recomputation
# mechanism catches with no special-case code). Attest family-c and leave
# it untouched (verified, current).
# ---------------------------------------------------------------------------
"$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario scenario-b --rep 1 \
	--operator "operator-1@example.com" --checklist "$CHECKLIST" \
	>/dev/null || fail "record for family-b (precondition) failed"

echo '{"mutated": true}' >>"$ROOT/scenarios/scenario-b/1/lifecycle.jsonl"

"$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario scenario-c --rep 1 \
	--operator "operator-1@example.com" --checklist "$CHECKLIST" \
	>/dev/null || fail "record for family-c (precondition) failed"

# Negative case (TC-081): a pilot attestation recorded for one family must
# not satisfy a DIFFERENT family's gate -- family-a has recorded nothing
# yet, but family-b/family-c HAVE; verifying family-a in isolation must
# still fail.
rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-a 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "negative case: family-a verified despite no attestation ever being recorded for it (sibling families' attestations must not leak): $out"
echo "$out" | grep -q "no_attestation" || fail "negative case: expected no_attestation condition for family-a, got: $out"

GOOD_CEILINGS=(--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10)

# batch.json must not exist yet -- proves every refusal below is genuinely
# pre-dispatch, not just a fast-failing run that still reached the
# classify/dispatch/summary stage (which is the only thing that writes it).
[[ ! -f "$ROOT/batch.json" ]] || fail "batch.json already exists before any invocation was expected to succeed"

# ---------------------------------------------------------------------------
# Invocation (0): the operator DEFAULT -- no --scenarios filter at all, so
# the family set is derived from every scenario in the index (mirrors
# run-lifecycle-batch.sh's own matrix enumeration default: no filter means
# every admitted package, TC-081's own family-derivation "operator
# default" path). Must still refuse as a whole command, naming both
# family-a (no attestation) and family-b (stale).
# ---------------------------------------------------------------------------
out0="$("$BATCH" --batch "$WORKDIR/batch-policy.yaml" --retention-root "$ROOT" --mode baseline \
	"${GOOD_CEILINGS[@]}" 2>&1)" && rc0=0 || rc0=$?
[[ "$rc0" -eq 3 ]] || fail "invocation(0) no --scenarios filter (operator default): expected spend-gate refusal exit 3, got $rc0: $out0"
echo "$out0" | grep -q "family-a" || fail "invocation(0): refusal did not name family-a: $out0"
echo "$out0" | grep -q "family-b" || fail "invocation(0): refusal did not name family-b: $out0"
echo "$out0" | grep -q '"classification"' && fail "invocation(0): output contains dispatch/classification evidence -- refusal was not pre-dispatch: $out0"
[[ ! -f "$ROOT/batch.json" ]] || fail "invocation(0): batch.json was written despite a whole-command refusal"

# ---------------------------------------------------------------------------
# Invocation (1): family-a-only matrix -- must refuse as a whole command
# before any dispatch, naming family-a, its specific condition (no
# attestation), and the schema-owned refusal-reason code (REQ-F-018).
# ---------------------------------------------------------------------------
out1="$("$BATCH" --batch "$WORKDIR/batch-policy.yaml" --retention-root "$ROOT" --mode baseline \
	"${GOOD_CEILINGS[@]}" --scenarios scenario-a 2>&1)" && rc1=0 || rc1=$?
[[ "$rc1" -eq 3 ]] || fail "invocation(1) family-a-only: expected spend-gate refusal exit 3, got $rc1: $out1"
echo "$out1" | grep -q "family-a" || fail "invocation(1): refusal did not name family-a: $out1"
echo "$out1" | grep -q "no_attestation" || fail "invocation(1): refusal did not report the no_attestation condition: $out1"
echo "$out1" | grep -q "missing_pilot_attestation" || fail "invocation(1): refusal did not carry the schema-owned missing_pilot_attestation reason code: $out1"
echo "$out1" | grep -q '"classification"' && fail "invocation(1): output contains dispatch/classification evidence -- refusal was not pre-dispatch: $out1"
[[ ! -f "$ROOT/batch.json" ]] || fail "invocation(1): batch.json was written despite a refusal"

# ---------------------------------------------------------------------------
# Invocation (2): family-b-only matrix -- must refuse, naming family-b, its
# specific condition (stale digest), and the schema-owned reason code.
# ---------------------------------------------------------------------------
out2="$("$BATCH" --batch "$WORKDIR/batch-policy.yaml" --retention-root "$ROOT" --mode baseline \
	"${GOOD_CEILINGS[@]}" --scenarios scenario-b 2>&1)" && rc2=0 || rc2=$?
[[ "$rc2" -eq 3 ]] || fail "invocation(2) family-b-only: expected spend-gate refusal exit 3, got $rc2: $out2"
echo "$out2" | grep -q "family-b" || fail "invocation(2): refusal did not name family-b: $out2"
echo "$out2" | grep -q "stale_digest" || fail "invocation(2): refusal did not report the stale_digest condition: $out2"
echo "$out2" | grep -q "stale_pilot_attestation_digest" || fail "invocation(2): refusal did not carry the schema-owned stale_pilot_attestation_digest reason code: $out2"
echo "$out2" | grep -q '"classification"' && fail "invocation(2): output contains dispatch/classification evidence -- refusal was not pre-dispatch: $out2"
[[ ! -f "$ROOT/batch.json" ]] || fail "invocation(2): batch.json was written despite a refusal"

# Schema-owned refusal reasons (REQ-F-018): both codes asserted above must
# be members of the schema's closed refusal_reason vocabulary, cross-checked
# the same way tc080_spend_gate_refusal_test.sh checks its own reasons.
for reason in missing_pilot_attestation stale_pilot_attestation_digest; do
	grep -qE "^\s*-\s*${reason}\s*\$" "$SCHEMA" || fail "refusal reason '$reason' is not present in $SCHEMA's refusal_reason vocabulary"
done

# ---------------------------------------------------------------------------
# Invocation (3): family-c-only matrix -- must proceed PAST the gate.
# Proven the same way tc080 proves it: the classification/dispatch stage is
# reached at all (exit 3, spend-gate refusal, would prove the opposite).
# The retention fixture already has an evaluation.jsonl in place for
# scenario-c rep 1 (built directly, not via a real dispatch), so
# classify_pair honestly reports skipped_complete and the batch exits 0 --
# still proof the gate was reached and passed, since classify_pair only
# runs after spend_gate_check_all has already returned success.
# ---------------------------------------------------------------------------
out3="$("$BATCH" --batch "$WORKDIR/batch-policy.yaml" --retention-root "$ROOT" --mode baseline \
	"${GOOD_CEILINGS[@]}" --scenarios scenario-c 2>&1)" && rc3=0 || rc3=$?
[[ "$rc3" -eq 0 ]] || fail "invocation(3) family-c-only: expected the gate to pass and dispatch/classification to be reached (exit 0), got $rc3: $out3"
echo "$out3" | grep -q '"classification": "skipped_complete"' || fail "invocation(3): expected classification to be reached and report skipped_complete: $out3"
[[ -f "$ROOT/batch.json" ]] || fail "invocation(3): batch.json was not written despite a successful, gate-passing invocation"
BATCH_JSON_DIGEST_AFTER_3="$(sha256sum "$ROOT/batch.json" | awk '{print $1}')"

# ---------------------------------------------------------------------------
# Direct pilot-ledger.sh record/verify exercise for family-a (AC-T2): a
# fresh attestation verifies clean, then mutating a retained artifact
# invalidates it on the next --verify.
# ---------------------------------------------------------------------------
"$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario scenario-a --rep 1 \
	--operator "operator-2@example.com" --checklist "$CHECKLIST" \
	>/dev/null || fail "record for family-a failed"

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-a 2>&1)" || rc=$?
[[ "$rc" -eq 0 ]] || fail "family-a verify immediately after record should pass, got rc=$rc: $out"
echo "$out" | grep -q "family=family-a: verified" || fail "family-a verify did not report verified: $out"

# AC-T2: mutate a retained artifact for family-a after attestation.
echo '{"mutated": true}' >>"$ROOT/scenarios/scenario-a/1/evidence/stage.json"

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-a 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "AC-T2: family-a verify after mutating a retained artifact should fail, but it passed"
echo "$out" | grep -q "family=family-a: FAILED" || fail "AC-T2: expected a FAILED verdict for family-a after mutation: $out"
echo "$out" | grep -q "stale_digest" || fail "AC-T2: expected stale_digest condition for family-a after mutation: $out"

# ---------------------------------------------------------------------------
# Invocation (4): multi-family matrix (family-a, family-b, family-c
# together) -- REQ-F-005's whole-command refusal must NOT be silently
# downgraded to per-family skipping. family-a is now stale (just mutated
# above) and family-b is still stale (precondition); family-c remains
# verified. The refusal must name BOTH family-a and family-b.
#
# Exit code + family names alone cannot distinguish "whole command
# refused" from "family-c was quietly dispatched/skipped while a and b
# were filtered out" -- an implementation that did per-family filtering
# could still exit 3 and mention both names in a log line while having
# already run classify_pair/dispatch_pair for family-c. The discriminator
# is that NOTHING past the gate ever ran for family-c either: no
# classification evidence in the output, and batch.json (only written by
# the post-dispatch summary step) is byte-identical to its state after
# invocation (3) -- i.e. this invocation never reached that step at all.
# ---------------------------------------------------------------------------
out4="$("$BATCH" --batch "$WORKDIR/batch-policy.yaml" --retention-root "$ROOT" --mode baseline \
	"${GOOD_CEILINGS[@]}" --scenarios scenario-a,scenario-b,scenario-c 2>&1)" && rc4=0 || rc4=$?
[[ "$rc4" -eq 3 ]] || fail "invocation(4) multi-family: expected spend-gate refusal exit 3 as a WHOLE COMMAND, got $rc4: $out4"
echo "$out4" | grep -q "family-a" || fail "invocation(4): refusal did not name family-a: $out4"
echo "$out4" | grep -q "family-b" || fail "invocation(4): refusal did not name family-b: $out4"
echo "$out4" | grep -q '"classification"' && fail "invocation(4): output contains dispatch/classification evidence -- family-c was NOT filtered out of a whole-command refusal, it was dispatched: $out4"
[[ -f "$ROOT/batch.json" ]] || fail "invocation(4): batch.json unexpectedly disappeared"
BATCH_JSON_DIGEST_AFTER_4="$(sha256sum "$ROOT/batch.json" | awk '{print $1}')"
[[ "$BATCH_JSON_DIGEST_AFTER_4" == "$BATCH_JSON_DIGEST_AFTER_3" ]] || fail "invocation(4): batch.json changed after the multi-family invocation, proving the post-dispatch summary step ran -- this was per-family filtering, not a whole-command refusal"

echo "TC-081: pass (pilot-ledger attestation gate refuses --mode baseline as a whole command naming every failing family, verified digest-current attestations proceed, and a post-attestation mutation invalidates a previously-verified attestation)"
