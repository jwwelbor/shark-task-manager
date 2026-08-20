#!/usr/bin/env bash
# TC-082 / T-E40-F10-007: retention-root layout and byte-preservation
# verification (spec.md REQ-F-004, REQ-NF-007; test-plan.md TC-082 full
# body; AC-005, AC-T1, AC-T2).
#
# Exercises the REAL verify-retention-root.sh binary (which delegates to
# the REAL verify-lifecycle-run.sh / verify-lifecycle-evaluation.sh) over a
# real retained-artifact fixture built with real sha256 digests, mirroring
# tc081's own "do not mock digest comparison" discipline.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
VERIFIER="$SCRIPTS_DIR/verify-retention-root.sh"
BATCH="$SCRIPTS_DIR/run-lifecycle-batch.sh"
SCHEMA="$REPO_ROOT/bench/reports/lifecycle-baseline-schema.yaml"
I07_FIXTURE="$REPO_ROOT/tests/contracts/testdata/e40_i07/valid/complete.jsonl"
I08_FIXTURE="$REPO_ROOT/bench/scripts/testdata/evaluation/eligible.jsonl"

fail() {
	echo "TC-082 FAIL: $1" >&2
	exit 1
}

[[ -x "$VERIFIER" ]] || fail "bench/scripts/verify-retention-root.sh missing or not executable"
[[ -x "$BATCH" ]] || fail "bench/scripts/run-lifecycle-batch.sh missing or not executable"
[[ -f "$SCHEMA" ]] || fail "bench/reports/lifecycle-baseline-schema.yaml missing"
[[ -f "$I07_FIXTURE" ]] || fail "committed valid I-07 fixture missing: $I07_FIXTURE"
[[ -f "$I08_FIXTURE" ]] || fail "committed valid I-08 fixture missing: $I08_FIXTURE"
command -v jq >/dev/null 2>&1 || fail "jq not found on PATH"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

SCENARIO_ID="scenario-tc082"
REP="1"

# ---------------------------------------------------------------------------
# Persistent "sources" staging area -- the pre-retention originals a real
# producer would digest against. Deliberately NOT ephemeral (unlike
# run-lifecycle-batch.sh's own pair_work, which it rm -rf's immediately
# after retention -- a known upstream gap this task's summary reports
# separately, see Notes below) so the golden root's manifest.json records
# source_path values that genuinely resolve, and the Edge Case fixture
# (source_path_missing) can delete exactly one of them on purpose.
# ---------------------------------------------------------------------------
SOURCES="$WORKDIR/sources"
mkdir -p "$SOURCES/evidence" "$SOURCES/transcripts"

cat >"$SOURCES/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$SCENARIO_ID"
scenario_version: "1"
entity_family: "family-tc082"
EOF
echo '{"stage": "code", "note": "tc082 fixture evidence"}' >"$SOURCES/evidence/stage.json"
echo "tc082 fixture transcript" >"$SOURCES/transcripts/stage.txt"
python3 -c '
import json
obj = {"root_key": "ROOT-001", "entries": [{"key": "T-001", "event": "created"}]}
with open("'"$SOURCES"'/entity-history.json", "w", encoding="utf-8") as f:
    f.write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")
'
cp "$I07_FIXTURE" "$SOURCES/lifecycle.jsonl"
jq -c '.metrics={quality:{},elapsed_time:{},provider_cost:{},rework:{},artifact_use:{}}' "$I08_FIXTURE" >"$SOURCES/evaluation.jsonl"
python3 -c '
import json
obj = {"held_back": True, "observed_result": "pass"}
with open("'"$SOURCES"'/oracle.json", "w", encoding="utf-8") as f:
    f.write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")
'

# ---------------------------------------------------------------------------
# Golden retention root: byte-preserving copies of every source artifact
# under scenarios/<scenario_id>/<rep>/, plus a manifest.json built the same
# way retain_pair builds one (source_path + sha256 per artifact), except
# this fixture covers all SEVEN non-manifest artifacts the schema's
# retention_manifest_required_fields lists -- run-lifecycle-batch.sh's own
# retain_pair currently only populates five (no evidence/transcripts
# entries), a gap this task reports as an upstream finding rather than
# fixing here (out of this task's scope; no unrelated changes).
# ---------------------------------------------------------------------------
build_golden() {
	local root="$1"
	local dest="$root/scenarios/$SCENARIO_ID/$REP"
	mkdir -p "$dest"
	cp "$SOURCES/package.yaml" "$dest/package.yaml"
	cp -a "$SOURCES/evidence" "$dest/evidence"
	cp -a "$SOURCES/transcripts" "$dest/transcripts"
	cp "$SOURCES/entity-history.json" "$dest/entity-history.json"
	cp "$SOURCES/lifecycle.jsonl" "$dest/lifecycle.jsonl"
	cp "$SOURCES/evaluation.jsonl" "$dest/evaluation.jsonl"
	cp "$SOURCES/oracle.json" "$dest/oracle.json"

	python3 - "$SOURCES" "$dest" "$SCENARIO_ID" "$REP" <<'PYEOF'
import hashlib
import json
import os
import sys

sources, dest, scenario_id, rep = sys.argv[1:5]


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def digest_of_path(path):
    if os.path.isdir(path):
        entries = []
        for root, dirs, files in os.walk(path):
            dirs.sort()
            for fname in sorted(files):
                fpath = os.path.join(root, fname)
                relpath = os.path.relpath(fpath, path).replace(os.sep, "/")
                with open(fpath, "rb") as fh:
                    entries.append({"path": relpath, "sha256": sha256_bytes(fh.read())})
        entries.sort(key=lambda e: e["path"])
        canonical = json.dumps(entries, sort_keys=True, separators=(",", ":")).encode("utf-8")
        return sha256_bytes(canonical)
    with open(path, "rb") as fh:
        return sha256_bytes(fh.read())


names = ["package.yaml", "evidence", "transcripts", "entity-history.json",
         "lifecycle.jsonl", "evaluation.jsonl", "oracle.json"]
artifacts = {}
for name in names:
    source_path = os.path.join(sources, name)
    artifacts[name] = {"source_path": source_path, "sha256": digest_of_path(source_path)}

manifest = {"scenario_id": scenario_id, "rep": int(rep), "artifacts": artifacts}
with open(os.path.join(dest, "manifest.json"), "w", encoding="utf-8") as f:
    f.write(json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n")
PYEOF
}

GOLDEN_ROOT="$WORKDIR/golden"
build_golden "$GOLDEN_ROOT"

# damaged_root <name> -- makes an independent full copy of the golden root
# under $WORKDIR/case-<name> and prints its path.
damaged_root() {
	local name="$1"
	local dest="$WORKDIR/case-$name"
	cp -a "$GOLDEN_ROOT" "$dest"
	echo "$dest"
}

pair_dir() {
	echo "$1/scenarios/$SCENARIO_ID/$REP"
}

# ===========================================================================
# (a) Complete, undamaged root verifies cleanly.
# ===========================================================================
out_a=""
rc_a=0
out_a="$("$VERIFIER" --retention-root "$GOLDEN_ROOT" --schema "$SCHEMA" 2>"$WORKDIR/a.err")" || rc_a=$?
[[ "$rc_a" -eq 0 ]] || fail "(a) complete root: expected exit 0, got $rc_a; stderr: $(cat "$WORKDIR/a.err")"
echo "$out_a" | grep -q '"verdict":"pass"' || fail "(a) complete root: expected a pass verdict, got: $out_a"
echo "$out_a" | grep -q '"failures":\[\]' || fail "(a) complete root: expected an empty failures array, got: $out_a"
[[ ! -s "$WORKDIR/a.err" ]] || fail "(a) complete root: expected no stderr diagnostics, got: $(cat "$WORKDIR/a.err")"

# ===========================================================================
# (b)-(i): each of the eight retained artifacts, deleted in turn, fails
# distinctly naming that artifact and "missing" -- not one representative
# case (TC-082's own instruction).
# ===========================================================================
ARTIFACTS=(package.yaml evidence transcripts entity-history.json lifecycle.jsonl evaluation.jsonl oracle.json manifest.json)
for artifact in "${ARTIFACTS[@]}"; do
	root="$(damaged_root "missing-$artifact")"
	dir="$(pair_dir "$root")"
	rm -rf "${dir:?}/$artifact"

	out=""
	rc=0
	out="$("$VERIFIER" --retention-root "$root" --schema "$SCHEMA" 2>"$WORKDIR/missing-$artifact.err")" || rc=$?
	[[ "$rc" -ne 0 ]] || fail "missing $artifact: expected nonzero exit, got 0"
	grep -q "$artifact: missing" "$WORKDIR/missing-$artifact.err" || fail "missing $artifact: stderr did not name the artifact and 'missing': $(cat "$WORKDIR/missing-$artifact.err")"
	echo "$out" | grep -q '"verdict":"fail"' || fail "missing $artifact: expected a fail verdict, got: $out"

	# AC-T2 (generalizes to every missing artifact, manifest.json included):
	# a layout failure short-circuits before phase 2/3 ever runs -- no
	# digest-phase or upstream-delegate reason ever appears for this pair.
	for later_reason in digest_mismatch re_serialized source_path_missing lineage_mismatch upstream_lifecycle_invalid upstream_evaluation_invalid; do
		echo "$out" | grep -q "$later_reason" && fail "missing $artifact: output contains a phase-2/3 reason ('$later_reason') -- layout-completeness did not short-circuit before the digest/upstream phase: $out"
	done
done

# AC-T2, stated explicitly: the manifest.json-missing case above already
# proves the ordering; assert it once more by name for traceability.
grep -q "manifest.json: missing" "$WORKDIR/missing-manifest.json.err" || fail "AC-T2: manifest.json-missing case did not name manifest.json as missing"

# ===========================================================================
# (j) One artifact's bytes mutated (not valid JSON) -> digest_mismatch,
# naming the artifact and reason.
# ===========================================================================
root_j="$(damaged_root "digest-mismatch")"
dir_j="$(pair_dir "$root_j")"
printf '\x00not-json-garbage\x00' >>"$dir_j/lifecycle.jsonl"
out_j=""
rc_j=0
out_j="$("$VERIFIER" --retention-root "$root_j" --schema "$SCHEMA" 2>"$WORKDIR/j.err")" || rc_j=$?
[[ "$rc_j" -ne 0 ]] || fail "(j) digest-mismatch: expected nonzero exit, got 0"
grep -q "lifecycle.jsonl: digest_mismatch" "$WORKDIR/j.err" || fail "(j) digest-mismatch: stderr did not name lifecycle.jsonl and digest_mismatch: $(cat "$WORKDIR/j.err")"
echo "$out_j" | grep -q '"reason":"digest_mismatch"' || fail "(j) digest-mismatch: stdout verdict missing digest_mismatch reason: $out_j"

# ===========================================================================
# (k) An artifact re-serialized (re-encoded JSON, same logical content,
# different bytes) fails distinctly as re_serialized, not digest_mismatch --
# the discriminator is a canonical-content recompute that still matches the
# manifest-recorded digest even though the raw bytes differ.
# ===========================================================================
root_k="$(damaged_root "re-serialized")"
dir_k="$(pair_dir "$root_k")"
ORIGINAL_DIGEST_K="$(sha256sum "$SOURCES/entity-history.json" | awk '{print $1}')"
python3 -c '
import json
obj = json.load(open("'"$SOURCES"'/entity-history.json", encoding="utf-8"))
with open("'"$dir_k"'/entity-history.json", "w", encoding="utf-8") as f:
    json.dump(obj, f, indent=2, sort_keys=False)
    f.write("\n")
'
RESERIALIZED_DIGEST_K="$(sha256sum "$dir_k/entity-history.json" | awk '{print $1}')"
[[ "$ORIGINAL_DIGEST_K" != "$RESERIALIZED_DIGEST_K" ]] || fail "(k) re-serialized: fixture setup produced byte-identical content; test is not exercising re-serialization"
out_k=""
rc_k=0
out_k="$("$VERIFIER" --retention-root "$root_k" --schema "$SCHEMA" 2>"$WORKDIR/k.err")" || rc_k=$?
[[ "$rc_k" -ne 0 ]] || fail "(k) re-serialized: expected nonzero exit, got 0"
grep -q "entity-history.json: re_serialized" "$WORKDIR/k.err" || fail "(k) re-serialized: stderr did not name entity-history.json and re_serialized: $(cat "$WORKDIR/k.err")"
echo "$out_k" | grep -q '"reason":"re_serialized"' || fail "(k) re-serialized: stdout verdict missing re_serialized reason: $out_k"
echo "$out_k" | grep -q 'digest_mismatch' && fail "(k) re-serialized: expected re_serialized only, not also digest_mismatch, for entity-history.json: $out_k"

# ===========================================================================
# Edge Case: a manifest listing a source path that no longer exists on disk
# must fail distinctly from a plain digest mismatch (source_path_missing).
# Digest equality alone is the authoritative byte-preservation proof (a
# digest match must NOT fail just because provenance metadata went stale --
# see verify-retention-root.sh's own header on why), so this case must
# BOTH corrupt the retained bytes (so the digest already disagrees) AND
# move the recorded source out from under it, to prove source_path is
# consulted only once a mismatch already exists, to sharpen its reason.
# ===========================================================================
root_edge="$(damaged_root "source-path-missing")"
dir_edge="$(pair_dir "$root_edge")"
printf '\x00not-json-garbage\x00' >>"$dir_edge/oracle.json"
ORACLE_SOURCE="$SOURCES/oracle.json"
mv "$ORACLE_SOURCE" "$ORACLE_SOURCE.moved"
out_edge=""
rc_edge=0
out_edge="$("$VERIFIER" --retention-root "$root_edge" --schema "$SCHEMA" 2>"$WORKDIR/edge.err")" || rc_edge=$?
mv "$ORACLE_SOURCE.moved" "$ORACLE_SOURCE"
[[ "$rc_edge" -ne 0 ]] || fail "edge case: expected nonzero exit when a recorded source_path no longer exists, got 0"
grep -q "oracle.json: source_path_missing" "$WORKDIR/edge.err" || fail "edge case: stderr did not name oracle.json and source_path_missing: $(cat "$WORKDIR/edge.err")"
echo "$out_edge" | grep -q '"reason":"source_path_missing"' || fail "edge case: stdout verdict missing source_path_missing reason: $out_edge"
echo "$out_edge" | grep -q '"reason":"digest_mismatch"' && fail "edge case: expected source_path_missing only, not also a plain digest_mismatch entry, for oracle.json: $out_edge"

# Counter-proof for the fix above: a digest MATCH must pass even when its
# recorded source_path is separately gone (provenance staleness alone is
# not a preservation defect) -- moving oracle.json's source with the
# retained bytes left untouched must verify cleanly.
root_source_stale_only="$(damaged_root "source-path-stale-only-digest-ok")"
mv "$ORACLE_SOURCE" "$ORACLE_SOURCE.moved2"
out_stale=""
rc_stale=0
out_stale="$("$VERIFIER" --retention-root "$root_source_stale_only" --schema "$SCHEMA" 2>"$WORKDIR/stale.err")" || rc_stale=$?
mv "$ORACLE_SOURCE.moved2" "$ORACLE_SOURCE"
[[ "$rc_stale" -eq 0 ]] || fail "counter-proof: a digest match must pass even when the recorded source_path is separately gone, got exit $rc_stale; stderr: $(cat "$WORKDIR/stale.err")"
echo "$out_stale" | grep -q '"verdict":"pass"' || fail "counter-proof: expected a pass verdict when digest matches despite a stale source_path, got: $out_stale"

# ===========================================================================
# lineage_mismatch: manifest.json's own declared scenario_id/rep must agree
# with the scenarios/<scenario_id>/<rep>/ directory it was found in.
# ===========================================================================
root_lineage="$(damaged_root "lineage-mismatch")"
dir_lineage="$(pair_dir "$root_lineage")"
python3 -c '
import json
path = "'"$dir_lineage"'/manifest.json"
manifest = json.load(open(path, encoding="utf-8"))
manifest["scenario_id"] = "scenario-tc082-WRONG"
with open(path, "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")
'
out_lineage=""
rc_lineage=0
out_lineage="$("$VERIFIER" --retention-root "$root_lineage" --schema "$SCHEMA" 2>"$WORKDIR/lineage.err")" || rc_lineage=$?
[[ "$rc_lineage" -ne 0 ]] || fail "lineage_mismatch: expected nonzero exit, got 0"
grep -q "manifest.json: lineage_mismatch" "$WORKDIR/lineage.err" || fail "lineage_mismatch: stderr did not name manifest.json and lineage_mismatch: $(cat "$WORKDIR/lineage.err")"
echo "$out_lineage" | grep -q '"reason":"lineage_mismatch"' || fail "lineage_mismatch: stdout verdict missing lineage_mismatch reason: $out_lineage"

# ===========================================================================
# Negative Case (redundant, explicit form of AC-T2): a root missing
# manifest.json entirely fails layout-completeness before any digest check.
# ===========================================================================
root_neg="$(damaged_root "negative-no-manifest")"
dir_neg="$(pair_dir "$root_neg")"
rm -f "$dir_neg/manifest.json"
out_neg=""
rc_neg=0
out_neg="$("$VERIFIER" --retention-root "$root_neg" --schema "$SCHEMA" 2>"$WORKDIR/neg.err")" || rc_neg=$?
[[ "$rc_neg" -ne 0 ]] || fail "negative case: expected nonzero exit, got 0"
grep -q "manifest.json: missing" "$WORKDIR/neg.err" || fail "negative case: stderr did not name manifest.json and missing: $(cat "$WORKDIR/neg.err")"
for later_reason in digest_mismatch re_serialized source_path_missing lineage_mismatch upstream_lifecycle_invalid upstream_evaluation_invalid; do
	echo "$out_neg" | grep -q "$later_reason" && fail "negative case: output contains a phase-2/3 reason ('$later_reason') despite manifest.json being absent: $out_neg"
done

echo "TC-082: layout completeness, digest equality, re-serialization detection, source-path/lineage checks, and AC-T2 ordering all pass"

# ===========================================================================
# TC-082 Input tail: "repeat retention of the same (scenario, rep) against
# the same root, once plainly and once with --reclaim-incomplete." This
# exercises run-lifecycle-batch.sh's OWN classify_pair/quarantine_pair
# discipline (T-E40-F10-004, REQ-NF-007 append-and-verify), not
# verify-retention-root.sh itself -- included here because it is part of
# TC-082's full body. Per REQ-NF-007: a plain repeat of an already-retained
# (scenario, rep) must be classified and skipped (no overwrite); only
# --reclaim-incomplete quarantines it (a MOVE under .incomplete/, never a
# delete) and reruns.
# ===========================================================================
RECLAIM_ROOT="$WORKDIR/reclaim"
mkdir -p "$RECLAIM_ROOT"
INDEX_DIR="$WORKDIR/reclaim-index"
mkdir -p "$INDEX_DIR/packages/$SCENARIO_ID"
cat >"$INDEX_DIR/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$INDEX_DIR/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/reclaim-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$INDEX_DIR/scenarios.yaml"
EOF

# Pre-existing INCOMPLETE prior attempt: directory present, evaluation.jsonl
# absent (classify_pair's own definition of incomplete_prior_attempt).
INCOMPLETE_DIR="$RECLAIM_ROOT/scenarios/$SCENARIO_ID/$REP"
mkdir -p "$INCOMPLETE_DIR"
echo "prior attempt marker" >"$INCOMPLETE_DIR/package.yaml"
BEFORE_DIGEST="$(sha256sum "$INCOMPLETE_DIR/package.yaml" | awk '{print $1}')"

GOOD_CEILINGS=(--acknowledge-provider-spend --max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10)

# Plain repeat: no --reclaim-incomplete. classify_pair reports
# incomplete_prior_attempt and the driver skips it untouched -- no
# quarantine, no overwrite.
out_plain="$("$BATCH" --batch "$WORKDIR/reclaim-batch-policy.yaml" --retention-root "$RECLAIM_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" 2>&1)" || true
echo "$out_plain" | grep -q "incomplete_prior_attempt" || fail "reclaim: plain repeat did not report incomplete_prior_attempt: $out_plain"
[[ ! -d "$RECLAIM_ROOT/.incomplete" ]] || fail "reclaim: plain repeat (no --reclaim-incomplete) unexpectedly quarantined the prior attempt"
[[ -f "$INCOMPLETE_DIR/package.yaml" ]] || fail "reclaim: plain repeat removed the prior attempt directory -- REQ-NF-007 forbids delete/overwrite"
AFTER_PLAIN_DIGEST="$(sha256sum "$INCOMPLETE_DIR/package.yaml" | awk '{print $1}')"
[[ "$AFTER_PLAIN_DIGEST" == "$BEFORE_DIGEST" ]] || fail "reclaim: plain repeat modified the prior attempt's bytes"

# --reclaim-incomplete repeat: quarantined (moved under .incomplete/), never
# silently deleted. Prior bytes must survive intact at the quarantine
# destination. The subsequent rerun itself is expected to fail (no
# root_key/scratch_root configured in this minimal fixture) -- only the
# quarantine MOVE is asserted here.
"$BATCH" --batch "$WORKDIR/reclaim-batch-policy.yaml" --retention-root "$RECLAIM_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" --reclaim-incomplete >"$WORKDIR/reclaim.out" 2>&1 || true
QUARANTINE_DIR="$RECLAIM_ROOT/.incomplete/$SCENARIO_ID/rep-$REP-1"
[[ -d "$QUARANTINE_DIR" ]] || fail "reclaim: --reclaim-incomplete did not create the expected quarantine directory $QUARANTINE_DIR: $(cat "$WORKDIR/reclaim.out")"
[[ -f "$QUARANTINE_DIR/package.yaml" ]] || fail "reclaim: quarantined directory is missing the prior attempt's package.yaml"
QUARANTINE_DIGEST="$(sha256sum "$QUARANTINE_DIR/package.yaml" | awk '{print $1}')"
[[ "$QUARANTINE_DIGEST" == "$BEFORE_DIGEST" ]] || fail "reclaim: quarantined bytes do not match the original prior attempt (REQ-NF-007: never silently deleting/mutating)"

echo "TC-082: reclaim-incomplete classification (plain skip vs. quarantine-and-rerun) passes"
