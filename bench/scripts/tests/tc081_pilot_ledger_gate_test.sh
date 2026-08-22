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
	# UAT-R3-01 rework: manifest.json's `artifacts` map now needs a real,
	# non-empty `source_path` per required artifact (pilot-ledger.sh
	# --record/--verify both refuse a required artifact with no source
	# provenance, regardless of digest agreement) -- a bare
	# "artifacts": {} manifest (what --record legitimately produced before
	# this fixture never called the real retain_pair) is no longer
	# accepted. The source_path VALUES here do not need to resolve on disk
	# (pilot-ledger.sh's own check is non-emptiness, not existence -- that
	# stricter existence check belongs to verify-retention-root.sh's
	# digest-mismatch path only); they only need to be real, non-empty
	# strings, matching what a real retain_pair would have recorded.
	python3 - "$dest" <<'PY'
import hashlib
import json
import os
import sys

dest = sys.argv[1]


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": hashlib.sha256(fh.read()).hexdigest()})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return hashlib.sha256(canonical).hexdigest()
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


artifacts = {}
for name in ("package.yaml", "evidence", "transcripts", "entity-history.json", "lifecycle.jsonl", "evaluation.jsonl", "oracle.json"):
    artifacts[name] = {
        "source_path": f"/fixture-source/{name}",
        "sha256": digest_of_path(os.path.join(dest, name)),
    }

manifest = {
    "scenario_id": os.path.basename(os.path.dirname(dest)),
    "rep": 1,
    "artifacts": artifacts,
}
with open(os.path.join(dest, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
PY
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

# ===========================================================================
# T-E40-F10-006 rework (code-review-2026-08-20T1731-E40-F10.md, defects 1
# and 13): re-tests exercising the REAL pilot-ledger.sh binary again, same
# Caller-Path Contract as the rest of this file (no mocking of digest
# computation, grammar validation, or containment checking).
# ===========================================================================

LEDGER_LINES_BEFORE_REWORK="$(wc -l <"$ROOT/pilot-ledger.jsonl")"

# ---------------------------------------------------------------------------
# Findings 1 and 7 (code-review-2026-08-20T2138-E40-F10.md;
# bench/reports/lifecycle-baseline-schema.yaml digest_rules.
# empty_artifact_semantics), CORRECTED by UAT-R3-01 (round 3): this section
# used to assert that an empty artifact with manifest.json source_path==""
# was an "honest, accepted gap" that both --record and --verify accepted.
# UAT-R3-01 found that acceptance is exactly the fabrication defect: a
# required artifact with no real source must NEVER be attestable, regardless
# of whether its (real, present) empty-content digest happens to be
# internally consistent. The one case that remains legitimate and
# UNCHANGED by this fix is case (ii) below: a real, non-empty source_path
# whose content happens to be empty (a genuine upstream producer defect,
# distinct from "no source was ever available").
# ---------------------------------------------------------------------------
retain_fixture_dirs() {
	# retain_fixture_dirs <scenario_id> <family> <evidence_has_content> <transcripts_has_content> [<evidence_source_path> <transcripts_source_path>]
	local scenario_id="$1" family="$2" evidence_has_content="$3" transcripts_has_content="$4"
	local evidence_source_path="${5-}" transcripts_source_path="${6-}"
	local dest="$ROOT/scenarios/$scenario_id/1"
	mkdir -p "$dest/evidence" "$dest/transcripts"
	cat >"$dest/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$scenario_id"
scenario_version: "1"
entity_family: "$family"
EOF
	[[ "$evidence_has_content" != "1" ]] || echo '{"stage": "code", "note": "fixture evidence"}' >"$dest/evidence/stage.json"
	[[ "$transcripts_has_content" != "1" ]] || echo "fixture transcript for $scenario_id" >"$dest/transcripts/stage.txt"
	echo '{"root_key": "ROOT-001", "entries": []}' >"$dest/entity-history.json"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/lifecycle.jsonl"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/evaluation.jsonl"
	echo '{"held_back": true}' >"$dest/oracle.json"
	# package.yaml/entity-history.json/lifecycle.jsonl/evaluation.jsonl/
	# oracle.json carry a real non-empty source_path unconditionally --
	# these cases are specifically about evidence/transcripts, not about
	# re-testing the other five artifacts' source provenance.
	#
	# T-E40-F10-006 UAT round 6 (2026-08-21T233606Z): pilot-ledger.sh's
	# --record/--verify now delegate to lib/verify_pair_retention, which
	# independently recomputes each artifact's digest and compares it
	# against manifest.json's OWN recorded `sha256` claim -- a literal
	# "placeholder" string (this fixture's pre-fix shape) would now be
	# flagged as digest_mismatch for every artifact, contaminating the
	# evidence/transcripts-specific axis these two cases exist to test.
	# Real digests, computed the same way retain_pair/pilot-ledger.sh
	# themselves compute them, keep this fixture's manifest self-consistent
	# so only the evidence/transcripts source_path axis is under test.
	python3 - "$dest" "$evidence_source_path" "$transcripts_source_path" <<'PY'
import hashlib
import json
import os
import sys

dest, evidence_source_path, transcripts_source_path = sys.argv[1:4]


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": hashlib.sha256(fh.read()).hexdigest()})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return hashlib.sha256(canonical).hexdigest()
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


artifacts = {}
for name in ("package.yaml", "entity-history.json", "lifecycle.jsonl", "evaluation.jsonl", "oracle.json"):
    artifacts[name] = {
        "source_path": f"/fixture-source/{name}",
        "sha256": digest_of_path(os.path.join(dest, name)),
    }
artifacts["evidence"] = {
    "source_path": evidence_source_path,
    "sha256": digest_of_path(os.path.join(dest, "evidence")),
}
artifacts["transcripts"] = {
    "source_path": transcripts_source_path,
    "sha256": digest_of_path(os.path.join(dest, "transcripts")),
}

manifest = {
    "scenario_id": os.path.basename(os.path.dirname(dest)),
    "rep": 1,
    "artifacts": artifacts,
}
with open(os.path.join(dest, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
PY
}

# Case (i), UAT-R3-01 counterfactual (was "honest gap" before this fix):
# both evidence/ and transcripts/ genuinely empty, with manifest.json
# recording source_path=="" for both -- exactly retain_pair's pre-fix
# fabrication shape. --record MUST now refuse (never attest over a
# required artifact with no source provenance), naming both artifacts and
# the schema-owned required_artifact_source_unavailable reason. This test
# fails against the pre-fix "honest gap, --record succeeds" behavior and
# passes only now that --record checks source_path before ever computing a
# ledger row.
retain_fixture_dirs scenario-empty-both family-empty-both 0 0 "" ""
rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario scenario-empty-both --rep 1 \
	--operator "operator-3@example.com" --checklist "$CHECKLIST" 2>&1)" || rc=$?
[[ "$rc" -eq 2 ]] || fail "UAT-R3-01 (both empty, no source provenance): expected --record to refuse with exit 2, got rc=$rc: $out"
echo "$out" | grep -q "required_artifact_source_unavailable" || fail "UAT-R3-01 (both empty, no source provenance): refusal did not name required_artifact_source_unavailable: $out"
echo "$out" | grep -q "evidence" || fail "UAT-R3-01 (both empty, no source provenance): refusal did not name evidence: $out"
echo "$out" | grep -q "transcripts" || fail "UAT-R3-01 (both empty, no source provenance): refusal did not name transcripts: $out"

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-empty-both 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "UAT-R3-01 (both empty, no source provenance): --verify unexpectedly succeeded for a family that was never attestable, got: $out"
echo "$out" | grep -q "no_attestation" || fail "UAT-R3-01 (both empty, no source provenance): expected no_attestation (the refused --record above wrote no ledger row), got: $out"

# Case (ii), UNCHANGED by UAT-R3-01: evidence/ is empty but manifest.json
# records a REAL, non-empty source_path for it (state (b): a real source
# WAS checked and found empty -- round-1 finding 3's original concern, now
# caught via a different axis than digest_of_path). transcripts/ has real
# content, so its emptiness check never applies regardless of its own
# source_path. --record still succeeds (present, non-None digest, AND real
# source provenance for every required artifact); --verify must FAIL,
# naming evidence with a reason distinct from stale_digest/no_attestation/
# incomplete_attestation/missing_source_provenance.
retain_fixture_dirs scenario-empty-evidence family-empty-evidence 0 1 "$WORKDIR/checked-i05-bundle" "/fixture-source/transcripts"
rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario scenario-empty-evidence --rep 1 \
	--operator "operator-3@example.com" --checklist "$CHECKLIST" 2>&1)" || rc=$?
[[ "$rc" -eq 0 ]] || fail "findings 1/7 (evidence empty, real source checked): recording should succeed (present, not missing, real source provenance), got rc=$rc: $out"

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-empty-evidence 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "findings 1/7 (evidence empty, real source checked): --verify should flag the found-but-empty evidence/ as a defect, got rc=0: $out"
echo "$out" | grep -q "family=family-empty-evidence: FAILED" || fail "findings 1/7 (evidence empty, real source checked): expected a FAILED verdict, got: $out"
echo "$out" | grep -q "empty_source_artifact" || fail "findings 1/7 (evidence empty, real source checked): expected the empty_source_artifact reason, got: $out"
echo "$out" | grep -q "evidence" || fail "findings 1/7 (evidence empty, real source checked): reason did not name 'evidence', got: $out"
echo "$out" | grep -q "stale_digest" && fail "findings 1/7 (evidence empty, real source checked): should not ALSO report stale_digest (digest matches; the defect is source_path-based, not staleness): $out"
echo "$out" | grep -q "missing_source_provenance" && fail "findings 1/7 (evidence empty, real source checked): should not ALSO report missing_source_provenance (source_path IS real and non-empty): $out"

# Only case (ii)'s --record call above succeeded and appended a ledger row
# -- case (i)'s --record now refuses (UAT-R3-01) and appends nothing.
LEDGER_LINES_AFTER_EMPTY_ARTIFACT_CASES="$(wc -l <"$ROOT/pilot-ledger.jsonl")"
[[ "$LEDGER_LINES_AFTER_EMPTY_ARTIFACT_CASES" -eq "$((LEDGER_LINES_BEFORE_REWORK + 1))" ]] || fail "findings 1/7: expected exactly 1 new ledger row (only case (ii)'s --record succeeded), before=$LEDGER_LINES_BEFORE_REWORK after=$LEDGER_LINES_AFTER_EMPTY_ARTIFACT_CASES"

echo "TC-081 (rework, findings 1/7, corrected by UAT-R3-01): pass (a required artifact with no real source provenance now refuses --record and --verify, never an accepted honest gap; a real-source-checked-but-empty artifact remains a --verify defect, distinct from staleness and from missing provenance)"

# ---------------------------------------------------------------------------
# UAT-R3-01 (round 3) counterfactual: a family with a PREVIOUSLY VERIFIED,
# digest-current attestation is invalidated when its retained manifest.json
# is later mutated to carry an empty source_path for a required artifact --
# the same "mutation after attestation invalidates a previously-verified
# attestation" discipline ADR-F10-09 already requires for byte mutation
# (tested above via family-b's lifecycle.jsonl append), now proven for
# provenance mutation too. This exercises --verify's missing_source_provenance
# check independently of --record's own refusal (case (i) above), against a
# family that DID successfully attest before the mutation.
# ---------------------------------------------------------------------------
mkdir -p "$INDEX_DIR/packages/scenario-provenance-mutated"
write_package "$INDEX_DIR/packages/scenario-provenance-mutated" "scenario-provenance-mutated" "family-provenance-mutated"
retain_fixture scenario-provenance-mutated

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario scenario-provenance-mutated --rep 1 \
	--operator "operator-3@example.com" --checklist "$CHECKLIST" 2>&1)" || rc=$?
[[ "$rc" -eq 0 ]] || fail "UAT-R3-01 (provenance mutated after attestation): precondition --record failed, got rc=$rc: $out"

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-provenance-mutated 2>&1)" || rc=$?
[[ "$rc" -eq 0 ]] || fail "UAT-R3-01 (provenance mutated after attestation): --verify should accept the fresh, real-provenance attestation, got rc=$rc: $out"
echo "$out" | grep -q "family=family-provenance-mutated: verified" || fail "UAT-R3-01 (provenance mutated after attestation): expected verified before mutation, got: $out"

python3 -c '
import json
manifest_path = "'"$ROOT"'/scenarios/scenario-provenance-mutated/1/manifest.json"
manifest = json.load(open(manifest_path, encoding="utf-8"))
manifest["artifacts"]["entity-history.json"]["source_path"] = ""
json.dump(manifest, open(manifest_path, "w", encoding="utf-8"), sort_keys=True, separators=(",", ":"))
'

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-provenance-mutated 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "UAT-R3-01 (provenance mutated after attestation): --verify should fail once source_path is emptied post-attestation, got rc=0: $out"
echo "$out" | grep -q "family=family-provenance-mutated: FAILED" || fail "UAT-R3-01 (provenance mutated after attestation): expected a FAILED verdict, got: $out"
echo "$out" | grep -q "missing_source_provenance" || fail "UAT-R3-01 (provenance mutated after attestation): expected missing_source_provenance, got: $out"
echo "$out" | grep -q "entity-history.json" || fail "UAT-R3-01 (provenance mutated after attestation): reason did not name entity-history.json, got: $out"

echo "TC-081 (UAT-R3-01 counterfactual): pass (a previously-verified attestation is invalidated by a post-attestation source_path mutation, distinctly as missing_source_provenance)"

# Re-checkpoint the ledger line count (this block's one successful --record,
# scenario-provenance-mutated, appended one more row) so the defect-13
# "a refused --record appended nothing" comparisons below still hold.
LEDGER_LINES_AFTER_EMPTY_ARTIFACT_CASES="$(wc -l <"$ROOT/pilot-ledger.jsonl")"

# ---------------------------------------------------------------------------
# Defect 13 (codex red-team finding 13): --scenario must be validated
# against REQ-F-002's closed lowercase-kebab grammar (the same grammar
# tests/contracts/e40_i04_scenario_contract_test.go enforces for I-04
# admission) before any read or write, for --record; and the retention-
# relative path built from a scenario_id -- whether a freshly-validated
# argv or one read back out of the (hand-editable) ledger for --verify --
# must be containment-checked before any read or write.
# ---------------------------------------------------------------------------

# (a) Grammar rejection for --record: uppercase, underscore, and an
# embedded path separator are all outside the closed lowercase-kebab
# grammar and must be refused before pilot-ledger.sh even looks at the
# filesystem for that scenario.
for bad_scenario in "Scenario-A" "scenario_a" "scenario/a" "../scenario-a" "/etc"; do
	rc=0
	out="$("$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario "$bad_scenario" --rep 1 \
		--operator "operator-4@example.com" --checklist "$CHECKLIST" 2>&1)" || rc=$?
	[[ "$rc" -eq 2 ]] || fail "defect 13: --scenario '$bad_scenario' should be refused (exit 2) by the grammar check, got rc=$rc: $out"
	echo "$out" | grep -qi "grammar\|lowercase-kebab" || fail "defect 13: --scenario '$bad_scenario' refusal did not name the grammar/lowercase-kebab reason: $out"
done
LEDGER_LINES_AFTER_GRAMMAR="$(wc -l <"$ROOT/pilot-ledger.jsonl")"
[[ "$LEDGER_LINES_AFTER_GRAMMAR" -eq "$LEDGER_LINES_AFTER_EMPTY_ARTIFACT_CASES" ]] || fail "defect 13: a grammar-refused --record appended to pilot-ledger.jsonl anyway"

# (b) Containment check catches what the grammar check structurally cannot:
# a scenario_id that IS valid lowercase-kebab but whose retention-relative
# path resolves outside the retention root via a symlinked path component
# (a real path-traversal primitive a lexical grammar check is blind to,
# which is exactly why assert_within_out_root canonicalizes with realpath
# before comparing -- the same guard run-lifecycle-batch.sh and
# run-review-comparison.sh already use for their own retention paths).
OUTSIDE_ROOT="$WORKDIR/secret-outside"
mkdir -p "$OUTSIDE_ROOT/1/evidence" "$OUTSIDE_ROOT/1/transcripts"
cat >"$OUTSIDE_ROOT/1/package.yaml" <<'EOF'
schema_version: "1.0"
scenario_id: "escape-link"
scenario_version: "1"
entity_family: "leaked-family"
EOF
echo '{"stage": "code"}' >"$OUTSIDE_ROOT/1/evidence/stage.json"
echo "leaked transcript" >"$OUTSIDE_ROOT/1/transcripts/stage.txt"
echo '{"root_key": "ROOT-001", "entries": []}' >"$OUTSIDE_ROOT/1/entity-history.json"
echo '{"scenario_id": "escape-link", "rep": 1}' >"$OUTSIDE_ROOT/1/lifecycle.jsonl"
echo '{"scenario_id": "escape-link", "rep": 1}' >"$OUTSIDE_ROOT/1/evaluation.jsonl"
echo '{"held_back": true}' >"$OUTSIDE_ROOT/1/oracle.json"
echo '{"scenario_id": "escape-link", "rep": 1, "artifacts": {}}' >"$OUTSIDE_ROOT/1/manifest.json"

mkdir -p "$ROOT/scenarios"
ln -s "$OUTSIDE_ROOT" "$ROOT/scenarios/escape-link"

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario escape-link --rep 1 \
	--operator "operator-5@example.com" --checklist "$CHECKLIST" 2>&1)" || rc=$?
[[ "$rc" -eq 2 ]] || fail "defect 13 (symlink escape): recording through a scenario dir symlinked outside the retention root should be refused (exit 2), got rc=$rc: $out"
echo "$out" | grep -qi "outside\|refusing" || fail "defect 13 (symlink escape): refusal did not name a containment reason: $out"
grep -q "leaked-family" "$ROOT/pilot-ledger.jsonl" && fail "defect 13 (symlink escape): the ledger was poisoned with the outside-root scenario's family -- containment check did not prevent the read"
LEDGER_LINES_AFTER_SYMLINK_RECORD="$(wc -l <"$ROOT/pilot-ledger.jsonl")"
[[ "$LEDGER_LINES_AFTER_SYMLINK_RECORD" -eq "$LEDGER_LINES_AFTER_GRAMMAR" ]] || fail "defect 13 (symlink escape): a refused --record appended to pilot-ledger.jsonl anyway"

# (c) The same containment check applies on the --verify read side against
# a ledger entry (append-only, hand-editable -- not a freshly-validated
# argv) that names the same symlinked scenario_id. Hand-append a forged row
# for a family --record never legitimately wrote, mirroring what a pre-fix
# --record (or direct hand-editing) could have produced.
python3 - "$ROOT/pilot-ledger.jsonl" <<'PYEOF'
import json
import sys

path = sys.argv[1]
row = {
    "family": "leaked-family",
    "run_reference": {"scenario_id": "escape-link", "rep": 1, "retention_path": "scenarios/escape-link/1"},
    "operator_identity": "attacker@example.com",
    "checklist_results": {"items": []},
    "inspected_artifact_digests": {},
    "recorded_at": "2026-08-20T00:00:00Z",
}
with open(path, "a", encoding="utf-8") as f:
    f.write(json.dumps(row, sort_keys=True) + "\n")
PYEOF

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family leaked-family 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "defect 13 (verify symlink escape): verifying a family whose ledger entry points outside the retention root should fail, got rc=0: $out"
echo "$out" | grep -q "family=leaked-family: FAILED" || fail "defect 13 (verify symlink escape): expected a FAILED verdict for leaked-family: $out"
echo "$out" | grep -q "unsafe_scenario_reference" || fail "defect 13 (verify symlink escape): expected the unsafe_scenario_reference reason, got: $out"

rm -f "$ROOT/scenarios/escape-link"

echo "TC-081 (rework, defect 13): pass (--scenario is refused before any read/write for both an out-of-grammar value and a grammar-valid value whose path escapes the retention root via a symlink, on both --record and --verify)"

# ---------------------------------------------------------------------------
# Defect 3 (advisor review of this rework, code-review-2026-08-20T1731-E40-F10.md
# finding 3's own stated failure mode): --verify's own comment says a missing
# artifact must never compare None == None against a recorded digest and be
# reported "verified" -- but the empty-dir fix alone does not close this for
# a ledger row whose inspected_artifact_digests is missing keys entirely (a
# forged/hand-edited row, or one from a pre-fix --record that never wrote a
# key for a since-deleted artifact). digest_of_path(missing_path) is None,
# and dict.get() on an absent key is also None -- None == None passes the
# staleness check with zero real evidence inspected.
# ---------------------------------------------------------------------------
scenario_id="scenario-incomplete-attestation"
family="family-incomplete-attestation"
# Deliberately no retained scenario directory created under $ROOT/scenarios
# for this scenario_id/rep -- digest_of_path() over a nonexistent path
# returns None for every one of the eight required artifacts, exactly
# mirroring recorded_digests.get(name) on a row with an empty digests map.
# If the verify loop ever compares None == None as a pass, this is the
# fixture that catches it.
python3 - "$ROOT/pilot-ledger.jsonl" "$scenario_id" "$family" <<'PYEOF'
import json
import sys

path, scenario_id, family = sys.argv[1:4]
row = {
    "family": family,
    "run_reference": {"scenario_id": scenario_id, "rep": 1, "retention_path": f"scenarios/{scenario_id}/1"},
    "operator_identity": "attacker@example.com",
    "checklist_results": {"items": []},
    # Deliberately empty -- no digests recorded for any of the eight
    # required artifacts, simulating a forged or pre-fix row.
    "inspected_artifact_digests": {},
    "recorded_at": "2026-08-20T00:00:00Z",
}
with open(path, "a", encoding="utf-8") as f:
    f.write(json.dumps(row, sort_keys=True) + "\n")
PYEOF

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family "$family" 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "defect 3 (incomplete attestation): verifying a family whose recorded digests are missing entirely should fail, got rc=0: $out"
echo "$out" | grep -q "family=$family: FAILED" || fail "defect 3 (incomplete attestation): expected a FAILED verdict for $family: $out"
echo "$out" | grep -q "incomplete_attestation" || fail "defect 3 (incomplete attestation): expected the incomplete_attestation reason, got: $out"

echo "TC-081 (rework, defect 3): pass (a ledger row missing recorded digests for required artifacts is never treated as verified via a None == None comparison)"

# ===========================================================================
# T-E40-F10-006 UAT round 6 (2026-08-21T233606Z, defect class: "treating a
# present file, digest field, or non-empty provenance string as proof of
# verified source derivation"): before this fix, pilot-ledger.sh's own
# `source_path` checks (--record's requirement-1 loop and --verify's
# missing_source_provenance loop) only ever asked "is source_path a
# non-empty string" -- neither ever read manifest.json's own per-artifact
# `sha256` claim and compared it against the actual retained bytes. A
# manifest.json carrying a fabricated, non-existent source_path alongside a
# `sha256` value forged to agree with the real retained bytes therefore
# passed both --record and --verify undetected. These two cases build such
# a manifest directly (bypassing lib/retain_pair entirely, exactly as a
# hand-crafted or forged manifest would) and prove the new
# lib/verify_pair_retention delegation closes the gap on both the record
# side and the verify side.
# ===========================================================================

build_forged_manifest_fixture() {
	# build_forged_manifest_fixture <scenario_id> <family> -- real byte
	# content for all eight required artifacts, but manifest.json's `sha256`
	# field for oracle.json is a FORGED value that does not match
	# oracle.json's real bytes, while source_path is a fabricated, non-empty,
	# non-existent path -- the exact shape the UAT rejection describes.
	local scenario_id="$1" family="$2"
	local dest="$ROOT/scenarios/$scenario_id/1"
	mkdir -p "$dest/evidence" "$dest/transcripts"
	cat >"$dest/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$scenario_id"
scenario_version: "1"
entity_family: "$family"
EOF
	echo '{"stage": "code", "note": "fixture evidence"}' >"$dest/evidence/stage.json"
	echo "fixture transcript for $scenario_id" >"$dest/transcripts/stage.txt"
	echo '{"root_key": "ROOT-001", "entries": []}' >"$dest/entity-history.json"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/lifecycle.jsonl"
	echo "{\"scenario_id\": \"$scenario_id\", \"rep\": 1}" >"$dest/evaluation.jsonl"
	echo '{"held_back": true}' >"$dest/oracle.json"
	python3 - "$dest" <<'PY'
import hashlib
import json
import os
import sys

dest = sys.argv[1]


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": hashlib.sha256(fh.read()).hexdigest()})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return hashlib.sha256(canonical).hexdigest()
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


artifacts = {}
for name in ("package.yaml", "entity-history.json", "lifecycle.jsonl", "evaluation.jsonl", "evidence", "transcripts"):
    artifacts[name] = {
        "source_path": f"/fixture-source/{name}",
        "sha256": digest_of_path(os.path.join(dest, name)),
    }
# The forgery: a non-empty, plausible-looking, but entirely fabricated
# source_path that was never a real derivation source, paired with a
# `sha256` claim that does NOT match oracle.json's real bytes -- exactly
# what pilot-ledger.sh's pre-fix "source_path is a non-empty string" check
# could never distinguish from a legitimate retain_pair-written entry.
artifacts["oracle.json"] = {
    "source_path": "/totally/fabricated/never-real/oracle.json",
    "sha256": "0" * 64,
}

manifest = {
    "scenario_id": os.path.basename(os.path.dirname(dest)),
    "rep": 1,
    "artifacts": artifacts,
}
with open(os.path.join(dest, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
PY
}

# ---------------------------------------------------------------------------
# Case A: --record must refuse over a forged manifest.json at record time --
# every artifact is genuinely present and every source_path is a non-empty
# string, but oracle.json's manifest-recorded `sha256` disagrees with its
# real retained bytes. The pre-fix check (non-emptiness only) would have
# recorded this attestation; the fix must refuse it.
# ---------------------------------------------------------------------------
build_forged_manifest_fixture scenario-forged-manifest-record family-forged-manifest-record
rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --record --scenario scenario-forged-manifest-record --rep 1 \
	--operator "operator-6@example.com" --checklist "$CHECKLIST" 2>&1)" || rc=$?
[[ "$rc" -eq 2 ]] || fail "UAT round 6 (forged manifest, --record): expected --record to refuse (exit 2) over a manifest.json whose recorded digest disagrees with the actual retained bytes, got rc=$rc: $out"
echo "$out" | grep -q "digest_mismatch" || fail "UAT round 6 (forged manifest, --record): refusal did not name digest_mismatch: $out"
echo "$out" | grep -q "oracle.json" || fail "UAT round 6 (forged manifest, --record): refusal did not name oracle.json: $out"
grep -q "family-forged-manifest-record" "$ROOT/pilot-ledger.jsonl" && fail "UAT round 6 (forged manifest, --record): a refused --record appended to pilot-ledger.jsonl anyway"

echo "TC-081 (UAT round 6, forged manifest at --record): pass (--record refuses a manifest whose per-artifact digest claim disagrees with the actual retained bytes, not just a non-empty source_path string)"

# ---------------------------------------------------------------------------
# Case B: a HAND-FORGED ledger row (never produced by a real --record call,
# same "append-only, hand-editable" attack class already exercised by defect
# 13(c) and defect 3 above) whose `inspected_artifact_digests` are set to the
# REAL, CURRENT bytes on disk -- so pilot-ledger's own pre-existing staleness
# check (ledger digest vs. current bytes) passes cleanly for every one of the
# eight artifacts, INCLUDING manifest.json itself (its bytes are never
# mutated after the forged row is planted, so `stale_digest: manifest.json`
# never fires either). source_path is a non-empty string for every artifact,
# so the pre-fix `missing_source_provenance` check also never fires. The one
# thing wrong is that manifest.json's OWN recorded `sha256` claim for
# oracle.json does not match oracle.json's real bytes -- exactly the "manifest
# was never legitimately produced by lib/retain_pair" shape a hand-planted
# retention directory (or a --record gate that only checked non-emptiness)
# could never distinguish from a real one. Before this fix, --verify never
# read manifest.json's own `sha256` field at all, so this family would have
# reported "verified" despite the demonstrably self-inconsistent manifest.
# ---------------------------------------------------------------------------
build_forged_manifest_fixture scenario-forged-manifest-verify family-forged-manifest-verify

python3 - "$ROOT/pilot-ledger.jsonl" "$ROOT/scenarios/scenario-forged-manifest-verify/1" <<'PYEOF'
import hashlib
import json
import os
import sys

ledger_path, scenario_dir = sys.argv[1:3]


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": hashlib.sha256(fh.read()).hexdigest()})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return hashlib.sha256(canonical).hexdigest()
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


# The forger reads the REAL current bytes and reports honest digests for
# every artifact -- defeating pilot-ledger's own staleness check entirely --
# without ever running --record (so lib/verify_pair_retention's manifest
# self-consistency gate was never applied to this retention directory).
digests = {
    name: digest_of_path(os.path.join(scenario_dir, name))
    for name in (
        "package.yaml", "evidence", "transcripts", "entity-history.json",
        "lifecycle.jsonl", "evaluation.jsonl", "oracle.json", "manifest.json",
    )
}
row = {
    "family": "family-forged-manifest-verify",
    "run_reference": {
        "scenario_id": "scenario-forged-manifest-verify",
        "rep": 1,
        "retention_path": "scenarios/scenario-forged-manifest-verify/1",
    },
    "operator_identity": "attacker@example.com",
    "checklist_results": {"items": []},
    "inspected_artifact_digests": digests,
    "recorded_at": "2026-08-21T00:00:00Z",
}
with open(ledger_path, "a", encoding="utf-8") as f:
    f.write(json.dumps(row, sort_keys=True) + "\n")
PYEOF

rc=0
out="$("$PILOT_LEDGER" --retention-root "$ROOT" --verify --family family-forged-manifest-verify 2>&1)" || rc=$?
[[ "$rc" -ne 0 ]] || fail "UAT round 6 (forged manifest, --verify): --verify should fail for a hand-forged ledger row backed by a self-inconsistent manifest.json, got rc=0: $out"
echo "$out" | grep -q "family=family-forged-manifest-verify: FAILED" || fail "UAT round 6 (forged manifest, --verify): expected a FAILED verdict, got: $out"
echo "$out" | grep -q "digest_mismatch" || fail "UAT round 6 (forged manifest, --verify): expected digest_mismatch (manifest's own recorded provenance disagrees with the actual bytes), got: $out"
echo "$out" | grep -q "oracle.json" || fail "UAT round 6 (forged manifest, --verify): reason did not name oracle.json, got: $out"
echo "$out" | grep -q "stale_digest" && fail "UAT round 6 (forged manifest, --verify): should not ALSO report stale_digest (the forged ledger row's digests were set to match the real current bytes exactly): $out"
echo "$out" | grep -q "missing_source_provenance" && fail "UAT round 6 (forged manifest, --verify): should not ALSO report missing_source_provenance (every source_path is a non-empty string): $out"

echo "TC-081 (UAT round 6, forged manifest at --verify): pass (a hand-forged ledger row whose digests match the real current bytes -- defeating pilot-ledger's own staleness check -- is still caught by cross-checking manifest.json's own recorded per-artifact digest against those same bytes)"
