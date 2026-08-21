#!/usr/bin/env bash
# TC-082 / T-E40-F10-007: retention-root layout and byte-preservation
# verification (spec.md REQ-F-004, REQ-NF-007; test-plan.md TC-082 full
# body; AC-005, AC-T1, AC-T2).
#
# Exercises the REAL verify-retention-root.sh binary (which delegates to
# the REAL verify-lifecycle-run.sh / verify-lifecycle-evaluation.sh) over a
# real retained-artifact fixture built with real sha256 digests, mirroring
# tc081's own "do not mock digest comparison" discipline.
#
# Caller-Path Contract (test-plan.md TC-082): "a root produced by one real
# run-lifecycle-batch.sh --mode pilot retention." Round-1 finding 2615
# (recurring into round-2's loop-guard, code-review-2026-08-20T2138-E40-F10.md
# Section I): this file's build_golden() used to hand-roll the retained
# directory and manifest.json with its own from-scratch digest_of_path/
# python inline script, never invoking any real production code path -- the
# exact gap that hid the empty-artifact digest divergence (finding 1) from
# this suite for two rework rounds. Two changes close it:
#   1. Test (a2) below drives the REAL "$BATCH" ("$BATCH --mode pilot ...")
#      end to end via stub RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN (real
#      matrix enumeration, real spend-gate.sh sourcing, real dispatch_pair/
#      retain_pair), satisfying the contract's literal text directly, and
#      asserts the REAL verifier accepts the result.
#   2. build_golden() (used for every damage-class case below, (b) through
#      (m)) now drives `bench/scripts/lib/retain_pair` directly -- the SAME
#      shared manifest builder run-lifecycle-batch.sh's own retain_pair()
#      (and run-review-comparison.sh's retain_gate()) call, byte-for-byte,
#      since its extraction in commit 63a7605a -- rather than a second,
#      hand-rolled digest/manifest implementation. This keeps the damage-
#      case matrix on real production retention logic without re-driving
#      the full batch orchestrator once per case; (a2) above independently
#      proves that orchestrator itself produces a root this validator
#      accepts. The bottom of this file separately drives "$BATCH" for real
#      a second time for the reclaim-incomplete assertions, which exercise
#      the driver's own classify_pair/quarantine_pair logic specifically.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
VERIFIER="$SCRIPTS_DIR/verify-retention-root.sh"
BATCH="$SCRIPTS_DIR/run-lifecycle-batch.sh"
RETAIN_PAIR="$SCRIPTS_DIR/lib/retain_pair"
SCHEMA="$REPO_ROOT/bench/reports/lifecycle-baseline-schema.yaml"
I07_FIXTURE="$REPO_ROOT/tests/contracts/testdata/e40_i07/valid/complete.jsonl"
I08_FIXTURE="$REPO_ROOT/bench/scripts/testdata/evaluation/eligible.jsonl"

fail() {
	echo "TC-082 FAIL: $1" >&2
	exit 1
}

[[ -x "$VERIFIER" ]] || fail "bench/scripts/verify-retention-root.sh missing or not executable"
[[ -x "$BATCH" ]] || fail "bench/scripts/run-lifecycle-batch.sh missing or not executable"
[[ -x "$RETAIN_PAIR" ]] || fail "bench/scripts/lib/retain_pair missing or not executable"
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
# after retention) so the golden root's manifest.json records source_path
# values that genuinely resolve, and the Edge Case fixture
# (source_path_missing) can delete exactly one of them on purpose. Laid out
# to match lib/retain_pair's own positional-argument shape exactly: a
# package.yaml, a lifecycle.jsonl, an evaluation.jsonl, an
# "<evaluation.jsonl>.oracle.json" sidecar (evaluate-lifecycle.sh's own
# naming convention retain_pair reuses, never re-derived), and an I-05
# bundle directory holding evidence content plus a transcripts/ subtree.
# entity-history.json has no source parameter in lib/retain_pair (no
# producer exists anywhere in this codebase today -- retain_pair always
# retains it as an honest empty placeholder, source_path=""); that is real
# production behavior, not a fixture gap, so this golden root reproduces it
# exactly rather than fabricating a source for it.
# ---------------------------------------------------------------------------
SOURCES="$WORKDIR/sources"
I05_BUNDLE="$SOURCES/i05-bundle"
mkdir -p "$I05_BUNDLE/transcripts"

cat >"$SOURCES/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$SCENARIO_ID"
scenario_version: "1"
entity_family: "family-tc082"
EOF
echo '{"stage": "code", "note": "tc082 fixture evidence"}' >"$I05_BUNDLE/stage.json"
echo "tc082 fixture transcript" >"$I05_BUNDLE/transcripts/stage.txt"
cp "$I07_FIXTURE" "$SOURCES/lifecycle.jsonl"
jq -c '.metrics={quality:{},elapsed_time:{},provider_cost:{},rework:{},artifact_use:{}}' "$I08_FIXTURE" >"$SOURCES/evaluation.jsonl"
# jq's `-c` output is compact but NOT necessarily sorted-key -- re-canonicalize
# so the RETAINED raw bytes are already this repository's canonical form
# (compact, sorted keys, trailing newline). Case (k) below re-serializes this
# same content differently (pretty-printed) and needs the ORIGINAL retained
# bytes' recorded digest to equal the canonical-content digest for the
# re_serialized (not digest_mismatch) reason to fire correctly.
python3 -c '
import json
path = "'"$SOURCES"'/evaluation.jsonl"
obj = json.load(open(path, encoding="utf-8"))
with open(path, "w", encoding="utf-8") as f:
    f.write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")
'
ORACLE_SOURCE="$SOURCES/evaluation.jsonl.oracle.json"
python3 -c '
import json
obj = {"held_back": True, "observed_result": "pass"}
with open("'"$ORACLE_SOURCE"'", "w", encoding="utf-8") as f:
    f.write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")
'

# ---------------------------------------------------------------------------
# Golden retention root, built by driving the REAL bench/scripts/lib/
# retain_pair (T-E40-F10-007 rework, code-review-2026-08-20T2138-E40-F10.md
# Section I / round-1 finding 2615): the exact same shared manifest builder
# run-lifecycle-batch.sh's own retain_pair() calls -- byte-preserving copies
# plus a manifest.json built the same way a real `run-lifecycle-batch.sh
# --mode pilot` retention builds one, not a second hand-rolled digest
# implementation that can silently drift from the production one (the gap
# that hid finding 1's empty-artifact divergence from this suite).
# ---------------------------------------------------------------------------
build_golden() {
	local root="$1"
	local dest="$root/scenarios/$SCENARIO_ID/$REP"
	mkdir -p "$dest"
	python3 "$RETAIN_PAIR" "$SCENARIO_ID" "$REP" "$SOURCES/package.yaml" \
		"$SOURCES/lifecycle.jsonl" "$SOURCES/evaluation.jsonl" "$I05_BUNDLE" "$dest"
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
# (a2) Caller-Path Contract literal satisfaction (test-plan.md TC-082: "a
# root produced by one real run-lifecycle-batch.sh --mode pilot retention"):
# drives the REAL "$BATCH" end to end -- real matrix enumeration, real
# spend-gate.sh sourcing, real dispatch_pair/retain_pair -- via stub
# RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN that write real, valid I-07/I-08
# fixture content to the exact --output paths the driver hands them
# (mirroring the report's own finding-1 repro invocation shape), then
# verifies the resulting root with the REAL verifier. This is IN ADDITION
# to (not a replacement for) the retain_pair-driven golden root above: that
# root drives every damage-class case below through a single shared
# fixture-building call, while this section proves the full, unmodified
# driver path (batch-policy parsing, root_key/scratch_root plumbing,
# dispatch_pair, retain_pair) independently produces a root this validator
# accepts, per Section I's loop-guard.
# ===========================================================================
DRIVER_WORKDIR="$WORKDIR/driver"
DRIVER_I05_BUNDLE="$DRIVER_WORKDIR/i05-bundle"
mkdir -p "$DRIVER_I05_BUNDLE/transcripts"
echo '{"stage": "code", "note": "tc082 driver-path evidence"}' >"$DRIVER_I05_BUNDLE/stage.json"
echo "tc082 driver-path transcript" >"$DRIVER_I05_BUNDLE/transcripts/stage.txt"

DRIVER_SCRATCH="$DRIVER_WORKDIR/scratch-template"
mkdir -p "$DRIVER_SCRATCH"
echo "placeholder scratch project (never mutated -- run-lifecycle-batch.sh copies it)" >"$DRIVER_SCRATCH/marker.txt"

DRIVER_INDEX_DIR="$DRIVER_WORKDIR/index"
DRIVER_SCENARIO_ID="scenario-tc082-driver"
mkdir -p "$DRIVER_INDEX_DIR/packages/$DRIVER_SCENARIO_ID"
cat >"$DRIVER_INDEX_DIR/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$DRIVER_SCENARIO_ID
EOF
cat >"$DRIVER_INDEX_DIR/packages/$DRIVER_SCENARIO_ID/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$DRIVER_SCENARIO_ID"
scenario_version: "1"
entity_family: "family-tc082-driver"
EOF
cat >"$DRIVER_WORKDIR/policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$DRIVER_INDEX_DIR/scenarios.yaml"
scenarios:
  $DRIVER_SCENARIO_ID:
    root_key: "ROOT-TC082-DRIVER"
    scratch_root: "$DRIVER_SCRATCH"
    i05_bundle_dir: "$DRIVER_I05_BUNDLE"
    reps: 1
EOF

# Stubs write REAL, valid committed fixture content to the --output path the
# driver hands them (TD-077 defect-class precedent: substituted via
# RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN, the same sibling-path override
# convention the driver's own header documents -- never a bare PATH
# substitution). No provider/network call is made; these are the same
# committed I-07/I-08 fixtures the retain_pair-driven golden root above
# uses, so phase 4's real upstream-delegate check has genuinely valid
# content to accept.
DRIVER_RUN_STUB="$DRIVER_WORKDIR/run-lifecycle-stub.sh"
cat >"$DRIVER_RUN_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
cp "$I07_FIXTURE" "\$output"
EOF
chmod +x "$DRIVER_RUN_STUB"

DRIVER_EVAL_STUB="$DRIVER_WORKDIR/evaluate-lifecycle-stub.sh"
cat >"$DRIVER_EVAL_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
jq -c '.metrics={quality:{},elapsed_time:{},provider_cost:{},rework:{},artifact_use:{}}' "$I08_FIXTURE" >"\$output"
# evaluate-lifecycle.sh's own oracle sidecar naming convention
# (<output>.oracle.json), which lib/retain_pair's copy_artifact("oracle.json",
# evaluation_jsonl + ".oracle.json") reads verbatim -- never re-derived here.
python3 -c 'import json,sys; obj={"held_back": True, "observed_result": "pass"}; open(sys.argv[1], "w", encoding="utf-8").write(json.dumps(obj, sort_keys=True, separators=(",", ":")) + "\n")' "\$output.oracle.json"
EOF
chmod +x "$DRIVER_EVAL_STUB"

DRIVER_ROOT="$WORKDIR/driver-retention-root"
driver_batch_rc=0
RUN_LIFECYCLE_BIN="$DRIVER_RUN_STUB" EVALUATE_LIFECYCLE_BIN="$DRIVER_EVAL_STUB" \
	"$BATCH" --batch "$DRIVER_WORKDIR/policy.yaml" --retention-root "$DRIVER_ROOT" \
	--mode pilot --acknowledge-provider-spend --max-cost-usd 5 \
	--max-wall-clock-seconds 600 --max-generated-tasks 10 \
	>"$WORKDIR/driver-batch.out" 2>"$WORKDIR/driver-batch.err" || driver_batch_rc=$?
[[ "$driver_batch_rc" -eq 0 ]] || fail "(a2) driver-path: real run-lifecycle-batch.sh --mode pilot invocation failed: exit $driver_batch_rc; stdout: $(cat "$WORKDIR/driver-batch.out"); stderr: $(cat "$WORKDIR/driver-batch.err")"
[[ -f "$DRIVER_ROOT/scenarios/$DRIVER_SCENARIO_ID/1/manifest.json" ]] || fail "(a2) driver-path: real driver did not retain the expected pair directory"
# invalid/index.jsonl is always created (even on a clean run) but must stay
# EMPTY here -- a non-empty entry would mean the driver itself classified
# this pair invalid (root_key_or_scratch_root_not_configured/
# lifecycle_run_failed/evaluation_failed/i05_bundle_not_configured), which
# would make this "driver-path" test pass verify-retention-root.sh against a
# pair the driver itself never considers successfully retained -- not a
# genuine Caller-Path Contract satisfaction.
[[ -f "$DRIVER_ROOT/invalid/index.jsonl" ]] || fail "(a2) driver-path: expected invalid/index.jsonl to exist (even if empty)"
[[ ! -s "$DRIVER_ROOT/invalid/index.jsonl" ]] || fail "(a2) driver-path: driver classified the pair invalid -- not a genuine success: $(cat "$DRIVER_ROOT/invalid/index.jsonl")"

out_a2=""
rc_a2=0
out_a2="$("$VERIFIER" --retention-root "$DRIVER_ROOT" --schema "$SCHEMA" 2>"$WORKDIR/a2.err")" || rc_a2=$?
[[ "$rc_a2" -eq 0 ]] || fail "(a2) driver-path: verify-retention-root.sh rejected a real run-lifecycle-batch.sh --mode pilot retention: exit $rc_a2; stderr: $(cat "$WORKDIR/a2.err")"
echo "$out_a2" | grep -q '"verdict":"pass"' || fail "(a2) driver-path: expected a pass verdict over a real driver-produced root, got: $out_a2"
echo "$out_a2" | grep -q '"failures":\[\]' || fail "(a2) driver-path: expected an empty failures array, got: $out_a2"

echo "TC-082: a root produced by one real run-lifecycle-batch.sh --mode pilot retention verifies cleanly (Caller-Path Contract)"

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
# evaluation.jsonl, not entity-history.json: lib/retain_pair always retains
# entity-history.json as an honest empty placeholder (no producer exists for
# it anywhere in this codebase -- digest_rules.empty_artifact_semantics), so
# it has no real re-serializable content to mutate. evaluation.jsonl is a
# real, JSON-shaped retained artifact (JSON_ARTIFACTS in
# verify-retention-root.sh) copied byte-for-byte from a real I-08 fixture by
# retain_pair, giving this case genuine content to re-encode.
root_k="$(damaged_root "re-serialized")"
dir_k="$(pair_dir "$root_k")"
ORIGINAL_DIGEST_K="$(sha256sum "$SOURCES/evaluation.jsonl" | awk '{print $1}')"
python3 -c '
import json
obj = json.load(open("'"$SOURCES"'/evaluation.jsonl", encoding="utf-8"))
with open("'"$dir_k"'/evaluation.jsonl", "w", encoding="utf-8") as f:
    json.dump(obj, f, indent=2, sort_keys=False)
    f.write("\n")
'
RESERIALIZED_DIGEST_K="$(sha256sum "$dir_k/evaluation.jsonl" | awk '{print $1}')"
[[ "$ORIGINAL_DIGEST_K" != "$RESERIALIZED_DIGEST_K" ]] || fail "(k) re-serialized: fixture setup produced byte-identical content; test is not exercising re-serialization"
out_k=""
rc_k=0
out_k="$("$VERIFIER" --retention-root "$root_k" --schema "$SCHEMA" 2>"$WORKDIR/k.err")" || rc_k=$?
[[ "$rc_k" -ne 0 ]] || fail "(k) re-serialized: expected nonzero exit, got 0"
grep -q "evaluation.jsonl: re_serialized" "$WORKDIR/k.err" || fail "(k) re-serialized: stderr did not name evaluation.jsonl and re_serialized: $(cat "$WORKDIR/k.err")"
echo "$out_k" | grep -q '"reason":"re_serialized"' || fail "(k) re-serialized: stdout verdict missing re_serialized reason: $out_k"
echo "$out_k" | grep -q 'digest_mismatch' && fail "(k) re-serialized: expected re_serialized only, not also digest_mismatch, for evaluation.jsonl: $out_k"

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

# ===========================================================================
# Phase 4 (delegated upstream schema validity): a retained artifact whose
# manifest-recorded digest matches its (malformed) bytes exactly -- so phase
# 3's digest check passes and does NOT short-circuit -- but whose content is
# not valid JSON must still fail, via delegation to the real
# verify-lifecycle-run.sh / verify-lifecycle-evaluation.sh, naming
# upstream_lifecycle_invalid / upstream_evaluation_invalid respectively. This
# is not one of TC-082's lettered cases, but it is the only way to exercise
# the Integration-with-existing-code delegation path (spec.md; Brownfield
# Context: "Delegates upstream schema validity ... rather than duplicating
# either validator") -- a digest-mismatch case alone (j) never reaches phase
# 4, since verify-retention-root.sh reports digest_mismatch and moves on
# without invoking either delegate for that artifact.
# ===========================================================================
corrupt_but_digest_matching() {
	# Overwrites $2 (an artifact path) with $3 (malformed content), then
	# patches $1's manifest.json so the recorded sha256 for that artifact
	# equals the new (malformed) bytes' digest -- keeping phase 3 (digest
	# equality) green so phase 4 (upstream delegation) is what fires.
	local dir="$1" artifact="$2" content="$3"
	printf '%s' "$content" >"$dir/$artifact"
	local new_digest
	new_digest="$(sha256sum "$dir/$artifact" | awk '{print $1}')"
	python3 -c '
import json
path, artifact, digest = __import__("sys").argv[1:4]
manifest = json.load(open(path, encoding="utf-8"))
manifest["artifacts"][artifact]["sha256"] = digest
with open(path, "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")
' "$dir/manifest.json" "$artifact" "$new_digest"
}

root_l="$(damaged_root "upstream-lifecycle-invalid")"
dir_l="$(pair_dir "$root_l")"
corrupt_but_digest_matching "$dir_l" "lifecycle.jsonl" 'not-valid-json-at-all'
out_l=""
rc_l=0
out_l="$("$VERIFIER" --retention-root "$root_l" --schema "$SCHEMA" 2>"$WORKDIR/l.err")" || rc_l=$?
[[ "$rc_l" -ne 0 ]] || fail "(l) upstream_lifecycle_invalid: expected nonzero exit, got 0"
grep -q "lifecycle.jsonl: upstream_lifecycle_invalid" "$WORKDIR/l.err" || fail "(l) upstream_lifecycle_invalid: stderr did not name lifecycle.jsonl and upstream_lifecycle_invalid: $(cat "$WORKDIR/l.err")"
echo "$out_l" | grep -q '"reason":"upstream_lifecycle_invalid"' || fail "(l) upstream_lifecycle_invalid: stdout verdict missing upstream_lifecycle_invalid reason: $out_l"
echo "$out_l" | grep -q '"reason":"digest_mismatch"' && fail "(l) upstream_lifecycle_invalid: expected no digest_mismatch entry (digest was made to match), got: $out_l"

root_m="$(damaged_root "upstream-evaluation-invalid")"
dir_m="$(pair_dir "$root_m")"
corrupt_but_digest_matching "$dir_m" "evaluation.jsonl" 'not-valid-json-at-all'
out_m=""
rc_m=0
out_m="$("$VERIFIER" --retention-root "$root_m" --schema "$SCHEMA" 2>"$WORKDIR/m.err")" || rc_m=$?
[[ "$rc_m" -ne 0 ]] || fail "(m) upstream_evaluation_invalid: expected nonzero exit, got 0"
grep -q "evaluation.jsonl: upstream_evaluation_invalid" "$WORKDIR/m.err" || fail "(m) upstream_evaluation_invalid: stderr did not name evaluation.jsonl and upstream_evaluation_invalid: $(cat "$WORKDIR/m.err")"
echo "$out_m" | grep -q '"reason":"upstream_evaluation_invalid"' || fail "(m) upstream_evaluation_invalid: stdout verdict missing upstream_evaluation_invalid reason: $out_m"
echo "$out_m" | grep -q '"reason":"digest_mismatch"' && fail "(m) upstream_evaluation_invalid: expected no digest_mismatch entry (digest was made to match), got: $out_m"

echo "TC-082: upstream schema-validity delegation (lifecycle + evaluation) fails distinctly on malformed-but-digest-matching content"

echo "TC-082: layout completeness, digest equality, re-serialization detection, source-path/lineage checks, and AC-T2 ordering all pass"

# ===========================================================================
# Comparison-driven coexistence (code-review-2026-08-20T2138-E40-F10.md
# finding 2 / this task's rework item 2): a comparison-driven (gate) pair
# retained in the SAME root as a real batch pair must not be misread as a
# broken/incomplete lifecycle pair. run-review-comparison.sh's retain_gate()
# (T-E40-F10-005, commit 63a7605a) retains through the exact SAME
# (scenario, rep) eight-artifact pair layout retain_pair() uses --
# gate_rep() maps qa -> rep 1, deep_review -> rep 2 -- plus an ADDITIVE
# manifest.json `gate` field (not one of the schema's
# retention_manifest_required_fields, so verify-retention-root.sh must
# ignore it, not reject it as an unexpected key) and, on an accepted
# comparator verdict, a 9th OPTIONAL comparison.json artifact dropped
# directly into the deep_review rep's directory (retention_optional_
# artifacts; never recorded in manifest.json's `artifacts` map, matching
# run-review-comparison.sh's exact comparison_dest write -- verify-
# retention-root.sh's phase 1 only checks retention_required_artifacts, and
# phase 3 only walks manifest.json's own `artifacts` map, so comparison.json
# is correctly neither required nor digest-checked here). This builds that
# exact shape with the SAME lib/retain_pair a real run-review-comparison.sh
# --mode baseline invocation drives (the `producer_gate` positional arg),
# for a scenario distinct from the batch pair's scenario_id, coexisting in
# one retention root, then verifies the WHOLE root passes cleanly.
# ===========================================================================
COEXIST_ROOT="$WORKDIR/coexist"
cp -a "$GOLDEN_ROOT" "$COEXIST_ROOT"

CMP_SCENARIO_ID="scenario-tc082-cmp"
QA_DEST="$COEXIST_ROOT/scenarios/$CMP_SCENARIO_ID/1"
DEEP_REVIEW_DEST="$COEXIST_ROOT/scenarios/$CMP_SCENARIO_ID/2"
mkdir -p "$QA_DEST" "$DEEP_REVIEW_DEST"
python3 "$RETAIN_PAIR" "$CMP_SCENARIO_ID" 1 "$SOURCES/package.yaml" \
	"$SOURCES/lifecycle.jsonl" "$SOURCES/evaluation.jsonl" "$I05_BUNDLE" "$QA_DEST" "qa"
python3 "$RETAIN_PAIR" "$CMP_SCENARIO_ID" 2 "$SOURCES/package.yaml" \
	"$SOURCES/lifecycle.jsonl" "$SOURCES/evaluation.jsonl" "$I05_BUNDLE" "$DEEP_REVIEW_DEST" "deep_review"
echo '{"verdict":"accepted","divergences":[]}' >"$DEEP_REVIEW_DEST/comparison.json"

grep -q '"gate":"qa"' "$QA_DEST/manifest.json" || fail "coexistence: fixture setup did not record the qa gate field -- test is not exercising the additive gate key"
grep -q '"gate":"deep_review"' "$DEEP_REVIEW_DEST/manifest.json" || fail "coexistence: fixture setup did not record the deep_review gate field -- test is not exercising the additive gate key"

out_cmp=""
rc_cmp=0
out_cmp="$("$VERIFIER" --retention-root "$COEXIST_ROOT" --schema "$SCHEMA" 2>"$WORKDIR/coexist.err")" || rc_cmp=$?
[[ "$rc_cmp" -eq 0 ]] || fail "coexistence: a comparison-driven qa/deep_review pair alongside a real batch pair unexpectedly failed verification: exit $rc_cmp; stderr: $(cat "$WORKDIR/coexist.err")"
[[ ! -s "$WORKDIR/coexist.err" ]] || fail "coexistence: expected no stderr diagnostics for an all-clean root, got: $(cat "$WORKDIR/coexist.err")"
cmp_pass_count="$(echo "$out_cmp" | grep -c '"verdict":"pass"')"
[[ "$cmp_pass_count" -eq 3 ]] || fail "coexistence: expected 3 passing pairs (1 batch pair + qa gate + deep_review gate), got $cmp_pass_count in: $out_cmp"
echo "$out_cmp" | grep -q '"scenario_id":"'"$CMP_SCENARIO_ID"'","verdict":"pass"' || fail "coexistence: qa/deep_review gate pairs were not both reported as passing: $out_cmp"
echo "$out_cmp" | grep -q '"verdict":"fail"' && fail "coexistence: expected zero failing pairs, got: $out_cmp"

echo "TC-082: comparison-driven qa/deep_review pairs (with an additive manifest.json 'gate' field and an unrecorded comparison.json) coexisting with a real batch pair in the same retention root verify cleanly, not misread as broken lifecycle pairs"

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
