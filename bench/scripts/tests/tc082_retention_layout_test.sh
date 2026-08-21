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
# (a3) Symlink write-through refusal (code-review-2026-08-21T0330-E40-F10.md
# finding 2, both codex-independent and this reviewer's own live repro):
# neither copy_artifact (single-file case) nor copy_dir_artifact (directory
# case) used to reject a pre-existing symlink planted at a destination
# artifact path before writing -- shutil.copyfile/Path.write_text followed
# the symlink and wrote through it, and Path.mkdir(exist_ok=True) treated a
# pre-existing symlink-to-directory as "already there, fine" and let the
# directory-walk copy real files INTO whatever the symlink resolved to.
# assert_within_out_root (the caller's only containment check) is never
# even called on individual artifact paths -- only on the rep-level `dest`
# directory -- so a symlink at a syntactically contained artifact name was
# never checked by anyone.
#
# Drives the REAL bench/scripts/lib/retain_pair binary directly (the exact
# production caller shape build_golden() above already uses) against a
# dest with a pre-existing symlink planted at BOTH a file-artifact path
# (package.yaml) and a directory-artifact path (evidence) pointing OUTSIDE
# the retention root, and proves:
#   1. the external file/directory the symlink pointed to is byte-for-byte
#      untouched (the actual attack this finding demonstrates: silent
#      overwrite of arbitrary filesystem content outside the retention
#      root);
#   2. the file case installs real retained content in place of the
#      symlink (retain_pair's normal job still gets done, just safely);
#   3. the directory case fails loudly (non-zero exit, a named diagnostic)
#      rather than silently writing into the symlinked-to directory --
#      rename() cannot atomically substitute a directory for a pre-existing
#      symlink, so install_atomic_dir's refusal is a structural property of
#      the fix, not a special-cased check that could itself drift.
# ===========================================================================
SYMLINK_OUTSIDE="$WORKDIR/symlink-outside"
mkdir -p "$SYMLINK_OUTSIDE/dir-target"
echo "SECRET FILE CONTENT OUTSIDE THE RETENTION ROOT -- must never be overwritten" >"$SYMLINK_OUTSIDE/secret-file.txt"
echo "SECRET DIR CONTENT OUTSIDE THE RETENTION ROOT -- must never be planted into" >"$SYMLINK_OUTSIDE/dir-target/leftover.txt"

SYMLINK_CASE_ROOT="$WORKDIR/symlink-case"
SYMLINK_DEST="$(pair_dir "$SYMLINK_CASE_ROOT")"
mkdir -p "$SYMLINK_DEST"
ln -s "$SYMLINK_OUTSIDE/secret-file.txt" "$SYMLINK_DEST/package.yaml"
ln -s "$SYMLINK_OUTSIDE/dir-target" "$SYMLINK_DEST/evidence"

symlink_rc=0
python3 "$RETAIN_PAIR" "$SCENARIO_ID" "$REP" "$SOURCES/package.yaml" \
	"$SOURCES/lifecycle.jsonl" "$SOURCES/evaluation.jsonl" "$I05_BUNDLE" "$SYMLINK_DEST" \
	>"$WORKDIR/symlink-case.out" 2>"$WORKDIR/symlink-case.err" || symlink_rc=$?

# (1) The external content the symlinks pointed to is untouched, regardless
# of how retain_pair itself reacted to the directory-case refusal.
[[ "$(cat "$SYMLINK_OUTSIDE/secret-file.txt")" == "SECRET FILE CONTENT OUTSIDE THE RETENTION ROOT -- must never be overwritten" ]] \
	|| fail "(a3) symlink write-through: external file outside the retention root was overwritten -- $(cat "$SYMLINK_OUTSIDE/secret-file.txt")"
[[ "$(ls "$SYMLINK_OUTSIDE/dir-target")" == "leftover.txt" ]] \
	|| fail "(a3) symlink write-through: external directory outside the retention root had content planted into it: $(ls "$SYMLINK_OUTSIDE/dir-target")"

# (2) The file case: the symlink is REPLACED with real retained content
# (never left as a symlink, never silently skipped).
[[ -L "$SYMLINK_DEST/package.yaml" ]] && fail "(a3) symlink write-through: dest/package.yaml is still a symlink after retain_pair -- expected it replaced with real content"
[[ -f "$SYMLINK_DEST/package.yaml" ]] || fail "(a3) symlink write-through: dest/package.yaml missing after retain_pair"
diff -u "$SOURCES/package.yaml" "$SYMLINK_DEST/package.yaml" >/dev/null \
	|| fail "(a3) symlink write-through: dest/package.yaml content is not the real retained source: $(diff -u "$SOURCES/package.yaml" "$SYMLINK_DEST/package.yaml")"

# (3) The directory case: rename() cannot atomically replace a symlink with
# a directory, so retain_pair MUST fail loudly (non-zero exit, a named
# diagnostic) rather than silently succeed by writing through it -- the
# symlink itself is left exactly as planted, never followed.
[[ "$symlink_rc" -ne 0 ]] || fail "(a3) symlink write-through: retain_pair exited 0 with a pre-existing symlink at a directory-artifact path -- expected a loud refusal"
grep -q "evidence" "$WORKDIR/symlink-case.err" || fail "(a3) symlink write-through: expected a diagnostic naming the refused artifact (evidence), got: $(cat "$WORKDIR/symlink-case.err")"
[[ -L "$SYMLINK_DEST/evidence" ]] || fail "(a3) symlink write-through: dest/evidence symlink was removed/replaced instead of left in place by the refusal"

echo "TC-082(a3): lib/retain_pair refuses to write through a pre-existing destination symlink -- external content untouched, file case installs real content in its place, directory case fails loudly (finding 2)"

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

# ===========================================================================
# classify_pair: symlink AT the rep-level directory itself (code-review-
# 2026-08-21T0459-E40-F10.md round-4 finding 2, Section E's "missing
# boundary case"): every prior symlink case in this suite plants the
# symlink at an ARTIFACT path inside an otherwise-real directory -- never
# at the directory-that-decides-skip-vs-dispatch itself. classify_pair's
# pre-round-4 "already retained, skip" fast path tested artifact PRESENCE
# (-f "$dir/evaluation.jsonl") through the rep directory before ever
# checking whether that directory is itself a symlink -- a pre-existing
# symlink pointed at a FOREIGN scenario's own legitimately-retained pair
# would be silently classified skipped_complete, with no diagnostic.
# ===========================================================================
SYMLINK_CLASSIFY_ROOT="$WORKDIR/symlink-classify"
mkdir -p "$SYMLINK_CLASSIFY_ROOT"

FOREIGN_SCENARIO_CLASSIFY="scenario-tc082-foreign"
FOREIGN_DIR_CLASSIFY="$SYMLINK_CLASSIFY_ROOT/scenarios/$FOREIGN_SCENARIO_CLASSIFY/1"
mkdir -p "$FOREIGN_DIR_CLASSIFY"
cp "$SOURCES/evaluation.jsonl" "$FOREIGN_DIR_CLASSIFY/evaluation.jsonl"
python3 -c 'import json,sys
scenario_id, rep, dest = sys.argv[1:4]
manifest = {"scenario_id": scenario_id, "rep": int(rep), "artifacts": {}}
open(dest, "w").write(json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n")' "$FOREIGN_SCENARIO_CLASSIFY" 1 "$FOREIGN_DIR_CLASSIFY/manifest.json"

mkdir -p "$SYMLINK_CLASSIFY_ROOT/scenarios/$SCENARIO_ID"
ln -s "$FOREIGN_DIR_CLASSIFY" "$SYMLINK_CLASSIFY_ROOT/scenarios/$SCENARIO_ID/$REP"

SYMLINK_CLASSIFY_INDEX="$WORKDIR/symlink-classify-index"
mkdir -p "$SYMLINK_CLASSIFY_INDEX/packages/$SCENARIO_ID"
cat >"$SYMLINK_CLASSIFY_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$SYMLINK_CLASSIFY_INDEX/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/symlink-classify-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$SYMLINK_CLASSIFY_INDEX/scenarios.yaml"
EOF

classify_symlink_rc=0
"$BATCH" --batch "$WORKDIR/symlink-classify-batch-policy.yaml" --retention-root "$SYMLINK_CLASSIFY_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" >"$WORKDIR/classify-symlink.out" 2>&1 || classify_symlink_rc=$?
[[ "$classify_symlink_rc" -eq 4 ]] || fail "classify_pair symlink: expected exit 4 (pair recorded failed, never treated as complete), got $classify_symlink_rc: $(cat "$WORKDIR/classify-symlink.out")"
grep -q "skipped_complete" "$WORKDIR/classify-symlink.out" && fail "classify_pair symlink: the symlinked rep directory was classified skipped_complete -- expected a refusal instead: $(cat "$WORKDIR/classify-symlink.out")"
grep -q "pre-existing symlink" "$WORKDIR/classify-symlink.out" || fail "classify_pair symlink: expected a diagnostic naming the pre-existing symlink: $(cat "$WORKDIR/classify-symlink.out")"
[[ -L "$SYMLINK_CLASSIFY_ROOT/scenarios/$SCENARIO_ID/$REP" ]] || fail "classify_pair symlink: the pre-existing symlink at the rep directory was unexpectedly removed/replaced"
[[ "$(cat "$FOREIGN_DIR_CLASSIFY/evaluation.jsonl")" == "$(cat "$SOURCES/evaluation.jsonl")" ]] \
	|| fail "classify_pair symlink: the foreign scenario's own retained evaluation.jsonl was modified"
grep -q "pre_existing_symlink_at_rep_directory" "$SYMLINK_CLASSIFY_ROOT/invalid/index.jsonl" \
	|| fail "classify_pair symlink: invalid/index.jsonl did not record the pre_existing_symlink_at_rep_directory reason: $(cat "$SYMLINK_CLASSIFY_ROOT/invalid/index.jsonl" 2>/dev/null || echo MISSING)"

echo "TC-082(classify_pair symlink, finding 2 round-4): classify_pair refuses to treat a symlinked rep directory as skipped_complete, before ever trusting evaluation.jsonl presence through it"

# ===========================================================================
# classify_pair: provenance mismatch (real directory, not a symlink, but
# manifest.json's scenario_id/rep does not match) -- the read-side
# "confused deputy" shape codex's R4-5 reproduced. classify_pair had NO
# provenance check of any kind before this round's fix (the wider half of
# finding 2: run-review-comparison.sh's gate_dest_provenance_ok at least
# checked the `gate` field; classify_pair checked nothing at all).
# ===========================================================================
PROVENANCE_ROOT="$WORKDIR/provenance-mismatch"
mkdir -p "$PROVENANCE_ROOT"
PROVENANCE_DIR="$PROVENANCE_ROOT/scenarios/$SCENARIO_ID/$REP"
mkdir -p "$PROVENANCE_DIR"
cp "$SOURCES/evaluation.jsonl" "$PROVENANCE_DIR/evaluation.jsonl"
python3 -c 'import json,sys
dest = sys.argv[1]
manifest = {"scenario_id": "scenario-tc082-WRONG-OWNER", "rep": 999, "artifacts": {}}
open(dest, "w").write(json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n")' "$PROVENANCE_DIR/manifest.json"
PROVENANCE_BEFORE="$(cat "$PROVENANCE_DIR/evaluation.jsonl")"

PROVENANCE_INDEX="$WORKDIR/provenance-index"
mkdir -p "$PROVENANCE_INDEX/packages/$SCENARIO_ID"
cat >"$PROVENANCE_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$PROVENANCE_INDEX/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/provenance-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$PROVENANCE_INDEX/scenarios.yaml"
EOF

provenance_rc=0
"$BATCH" --batch "$WORKDIR/provenance-batch-policy.yaml" --retention-root "$PROVENANCE_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" >"$WORKDIR/provenance-mismatch.out" 2>&1 || provenance_rc=$?
[[ "$provenance_rc" -eq 4 ]] || fail "classify_pair provenance mismatch: expected exit 4 (pair recorded failed, never treated as complete), got $provenance_rc: $(cat "$WORKDIR/provenance-mismatch.out")"
grep -q "skipped_complete" "$WORKDIR/provenance-mismatch.out" && fail "classify_pair provenance mismatch: the mismatched-provenance pair was classified skipped_complete: $(cat "$WORKDIR/provenance-mismatch.out")"
grep -q "does not match this pair" "$WORKDIR/provenance-mismatch.out" || fail "classify_pair provenance mismatch: expected the provenance-mismatch diagnostic: $(cat "$WORKDIR/provenance-mismatch.out")"
grep -q "provenance_mismatch" "$PROVENANCE_ROOT/invalid/index.jsonl" \
	|| fail "classify_pair provenance mismatch: invalid/index.jsonl did not record the provenance_mismatch reason: $(cat "$PROVENANCE_ROOT/invalid/index.jsonl" 2>/dev/null || echo MISSING)"
[[ "$(cat "$PROVENANCE_DIR/evaluation.jsonl")" == "$PROVENANCE_BEFORE" ]] \
	|| fail "classify_pair provenance mismatch: the pre-existing (wrong-owner) evaluation.jsonl was modified"

echo "TC-082(classify_pair provenance mismatch, finding 2 round-4): classify_pair refuses to treat a rep directory as skipped_complete when its manifest.json scenario_id/rep does not match, even though it is a real (non-symlink) directory"

# ===========================================================================
# batch.json / invalid/index.jsonl symlink write-through (code-review-
# 2026-08-21T0459-E40-F10.md round-4 finding 3): these two top-level
# retention-root writers used to be plain open(path, "w")/os.makedirs(...,
# exist_ok=True) -- neither rejects a pre-existing symlink at that exact
# path, the same defect class this round's fix closed for every per-pair
# artifact via lib/retain_pair's install_atomic_file/install_atomic_dir.
# These two writers sit OUTSIDE lib/retain_pair entirely (they are this
# driver's own top-level aggregate outputs, not a (scenario, rep) pair
# artifact), so they were never brought under any prior round's sweep
# despite living in the SAME file the round-4 commit message claimed to
# have fully swept.
# ===========================================================================
SYMLINK_WRITE_INDEX_DIR="$WORKDIR/symlink-write-index"
mkdir -p "$SYMLINK_WRITE_INDEX_DIR/packages/$SCENARIO_ID"
cat >"$SYMLINK_WRITE_INDEX_DIR/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$SYMLINK_WRITE_INDEX_DIR/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/symlink-write-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$SYMLINK_WRITE_INDEX_DIR/scenarios.yaml"
EOF

# --- batch.json: pre-existing symlink to an OUTSIDE-root file ---
BATCH_JSON_ROOT="$WORKDIR/symlink-write-batch-json"
mkdir -p "$BATCH_JSON_ROOT"
BATCH_JSON_OUTSIDE="$WORKDIR/outside-batch-json-target.txt"
echo "SECRET CONTENT OUTSIDE THE RETENTION ROOT -- must never be overwritten" >"$BATCH_JSON_OUTSIDE"
ln -s "$BATCH_JSON_OUTSIDE" "$BATCH_JSON_ROOT/batch.json"

batch_json_rc=0
"$BATCH" --batch "$WORKDIR/symlink-write-batch-policy.yaml" --retention-root "$BATCH_JSON_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" >"$WORKDIR/batch-json-symlink.out" 2>&1 || batch_json_rc=$?
# root_key/scratch_root are not configured for this minimal fixture, so the
# pair is recorded failed and the batch itself exits 4 (overall_bad) --
# the SAME "clean run still writes batch.json/invalid" shape (a2) above
# already relies on, just with a failing pair instead of a succeeding one;
# what this section asserts is the WRITE SAFETY of that write, not the
# pair's own success/failure.
[[ "$batch_json_rc" -eq 4 ]] || fail "batch.json symlink: expected exit 4 (batch completed, pair recorded failed), got $batch_json_rc: $(cat "$WORKDIR/batch-json-symlink.out")"
[[ "$(cat "$BATCH_JSON_OUTSIDE")" == "SECRET CONTENT OUTSIDE THE RETENTION ROOT -- must never be overwritten" ]] \
	|| fail "batch.json symlink: the external file the symlink pointed to was overwritten -- $(cat "$BATCH_JSON_OUTSIDE")"
[[ -L "$BATCH_JSON_ROOT/batch.json" ]] && fail "batch.json symlink: batch.json is still a symlink after the run -- expected it replaced with real retained content"
[[ -f "$BATCH_JSON_ROOT/batch.json" ]] || fail "batch.json symlink: batch.json missing after the run"
grep -q '"phase": *"lifecycle_v2"' "$BATCH_JSON_ROOT/batch.json" || fail "batch.json symlink: batch.json does not contain real retained content: $(cat "$BATCH_JSON_ROOT/batch.json")"

echo "TC-082(batch.json symlink, finding 3 round-4): batch.json's writer replaces a pre-existing destination symlink instead of writing through it -- external content untouched"

# --- invalid/: pre-existing symlink to an OUTSIDE-root directory ---
INVALID_ROOT="$WORKDIR/symlink-write-invalid"
mkdir -p "$INVALID_ROOT"
INVALID_OUTSIDE="$WORKDIR/outside-invalid-target"
mkdir -p "$INVALID_OUTSIDE"
echo "leftover" >"$INVALID_OUTSIDE/leftover.txt"
ln -s "$INVALID_OUTSIDE" "$INVALID_ROOT/invalid"

invalid_rc=0
"$BATCH" --batch "$WORKDIR/symlink-write-batch-policy.yaml" --retention-root "$INVALID_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" >"$WORKDIR/invalid-symlink.out" 2>&1 || invalid_rc=$?
[[ "$invalid_rc" -ne 0 ]] || fail "invalid/ symlink: expected a nonzero exit (loud refusal) when invalid/ is a pre-existing symlink, got 0: $(cat "$WORKDIR/invalid-symlink.out")"
grep -q "pre-existing symlink" "$WORKDIR/invalid-symlink.out" || fail "invalid/ symlink: expected a diagnostic naming the pre-existing symlink at invalid/: $(cat "$WORKDIR/invalid-symlink.out")"
[[ -L "$INVALID_ROOT/invalid" ]] || fail "invalid/ symlink: the pre-existing symlink at invalid/ was unexpectedly removed/replaced"
[[ "$(ls "$INVALID_OUTSIDE")" == "leftover.txt" ]] \
	|| fail "invalid/ symlink: the external directory the symlink pointed to had content planted into it -- $(ls "$INVALID_OUTSIDE")"

echo "TC-082(invalid/ symlink, finding 3 round-4): the invalid/index.jsonl writer refuses to write through a pre-existing symlink at invalid/ -- external directory untouched"

# ===========================================================================
# --reps upper-bound validation (code-review-2026-08-21T0459-E40-F10.md
# round-4 tech-debt item 4 / downgraded codex R4-1): --reps must stay below
# the reserved gate-rep band run-review-comparison.sh's gate_rep() uses
# (GATE_REP_BASE=900000), so the two drivers' rep allocations structurally
# cannot collide regardless of the --reps value an operator supplies.
# ===========================================================================
reps_bound_rc=0
"$BATCH" --batch "$WORKDIR/reclaim-batch-policy.yaml" --retention-root "$WORKDIR/reps-bound-unused" \
	--mode preview --reps 900000 >"$WORKDIR/reps-bound.out" 2>&1 || reps_bound_rc=$?
[[ "$reps_bound_rc" -eq 2 ]] || fail "--reps upper bound: expected exit 2 (usage error) for --reps at the reserved gate-rep band, got $reps_bound_rc: $(cat "$WORKDIR/reps-bound.out")"
grep -q "reserved gate-rep band" "$WORKDIR/reps-bound.out" || fail "--reps upper bound: expected a diagnostic naming the reserved gate-rep band: $(cat "$WORKDIR/reps-bound.out")"

reps_ok_rc=0
"$BATCH" --batch "$WORKDIR/reclaim-batch-policy.yaml" --retention-root "$WORKDIR/reps-bound-ok" \
	--mode preview --reps 899999 >"$WORKDIR/reps-ok.out" 2>&1 || reps_ok_rc=$?
[[ "$reps_ok_rc" -eq 0 ]] || fail "--reps upper bound: a value just under the reserved band must still be accepted, got $reps_ok_rc: $(cat "$WORKDIR/reps-ok.out")"

echo "TC-082(--reps upper bound, tech-debt item 4): --reps at or above the reserved gate-rep band is rejected as a usage error; just under it is accepted"

# ===========================================================================
# --reps upper-bound validation, the two SIBLING inputs (advisor review
# after this pass's own completion, before declaring done): the --reps
# FLAG is only one of three inputs that feed the effective per-scenario
# reps count -- a batch policy's own top-level `min_reps` (reps defaults to
# min_reps when a scenario declares no override) and a per-scenario
# `scenarios.<id>.reps` override can reach the reserved GATE_REP_BASE band
# exactly as easily as --reps. Checking only the flag and leaving these two
# YAML-declared inputs unchecked would have been the SAME half-swept
# pattern this whole pass exists to close, just relocated one level down.
# ===========================================================================
MIN_REPS_BOUND_INDEX="$WORKDIR/min-reps-bound-index"
mkdir -p "$MIN_REPS_BOUND_INDEX/packages/$SCENARIO_ID"
cat >"$MIN_REPS_BOUND_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$MIN_REPS_BOUND_INDEX/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/min-reps-bound-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 900000
scenario_index: "$MIN_REPS_BOUND_INDEX/scenarios.yaml"
EOF

min_reps_bound_rc=0
"$BATCH" --batch "$WORKDIR/min-reps-bound-policy.yaml" --retention-root "$WORKDIR/min-reps-bound-unused" \
	--mode preview >"$WORKDIR/min-reps-bound.out" 2>&1 || min_reps_bound_rc=$?
[[ "$min_reps_bound_rc" -eq 1 ]] || fail "min_reps upper bound: expected exit 1 (matrix enumeration refused), got $min_reps_bound_rc: $(cat "$WORKDIR/min-reps-bound.out")"
grep -q "reserved gate-rep band" "$WORKDIR/min-reps-bound.out" || fail "min_reps upper bound: expected a diagnostic naming the reserved gate-rep band: $(cat "$WORKDIR/min-reps-bound.out")"
grep -q "min_reps" "$WORKDIR/min-reps-bound.out" || fail "min_reps upper bound: expected the diagnostic to name min_reps specifically: $(cat "$WORKDIR/min-reps-bound.out")"

echo "TC-082(min_reps upper bound, round-4 sweep): a batch policy's top-level min_reps at the reserved gate-rep band is rejected"

PER_SCENARIO_REPS_BOUND_INDEX="$WORKDIR/per-scenario-reps-bound-index"
mkdir -p "$PER_SCENARIO_REPS_BOUND_INDEX/packages/$SCENARIO_ID"
cat >"$PER_SCENARIO_REPS_BOUND_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$PER_SCENARIO_REPS_BOUND_INDEX/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/per-scenario-reps-bound-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$PER_SCENARIO_REPS_BOUND_INDEX/scenarios.yaml"
scenarios:
  $SCENARIO_ID:
    reps: 900000
EOF

per_scenario_reps_bound_rc=0
"$BATCH" --batch "$WORKDIR/per-scenario-reps-bound-policy.yaml" --retention-root "$WORKDIR/per-scenario-reps-bound-unused" \
	--mode preview >"$WORKDIR/per-scenario-reps-bound.out" 2>&1 || per_scenario_reps_bound_rc=$?
[[ "$per_scenario_reps_bound_rc" -eq 1 ]] || fail "per-scenario reps upper bound: expected exit 1 (matrix enumeration refused), got $per_scenario_reps_bound_rc: $(cat "$WORKDIR/per-scenario-reps-bound.out")"
grep -q "reserved gate-rep band" "$WORKDIR/per-scenario-reps-bound.out" || fail "per-scenario reps upper bound: expected a diagnostic naming the reserved gate-rep band: $(cat "$WORKDIR/per-scenario-reps-bound.out")"
grep -q "scenarios.$SCENARIO_ID.reps" "$WORKDIR/per-scenario-reps-bound.out" || fail "per-scenario reps upper bound: expected the diagnostic to name the specific scenario's reps override: $(cat "$WORKDIR/per-scenario-reps-bound.out")"

echo "TC-082(per-scenario reps upper bound, round-4 sweep): a batch policy's scenarios.<id>.reps override at the reserved gate-rep band is rejected"

# ===========================================================================
# quarantine_pair: pre-existing symlink at .incomplete/<scenario_id> (round-4
# sweep, found while auditing every "does this already exist" trust
# decision in run-lifecycle-batch.sh per code-review-2026-08-21T0459-
# E40-F10.md's escalation): `mkdir -p "$incomplete_root"` used to treat a
# pre-existing symlink-to-directory at .incomplete/<scenario_id> (or its
# parent, .incomplete) as "already there, fine" and silently no-op against
# it -- the subsequent `mv "$dir" "$dest"` in quarantine_pair() would then
# relocate a quarantined prior attempt INTO whatever the symlink resolved
# to (still in-root, so assert_within_out_root's containment check alone
# never caught it) instead of this scenario's own .incomplete/ area. Same
# defect class as retain_pair()'s own `dest` guard, at a write call this
# round's sweep found unswept by any prior round.
# ===========================================================================
QUARANTINE_SYMLINK_ROOT="$WORKDIR/quarantine-symlink"
mkdir -p "$QUARANTINE_SYMLINK_ROOT"

QUARANTINE_INCOMPLETE_DIR="$QUARANTINE_SYMLINK_ROOT/scenarios/$SCENARIO_ID/$REP"
mkdir -p "$QUARANTINE_INCOMPLETE_DIR"
echo "prior attempt marker" >"$QUARANTINE_INCOMPLETE_DIR/package.yaml"
QUARANTINE_BEFORE_DIGEST="$(sha256sum "$QUARANTINE_INCOMPLETE_DIR/package.yaml" | awk '{print $1}')"

# A foreign, unrelated real directory that .incomplete/<scenario_id> will
# be pre-symlinked to point at -- the "confused deputy" target the
# quarantine MOVE must never write into.
QUARANTINE_FOREIGN="$QUARANTINE_SYMLINK_ROOT/foreign-incomplete-target"
mkdir -p "$QUARANTINE_FOREIGN"
echo "leftover" >"$QUARANTINE_FOREIGN/leftover.txt"
mkdir -p "$QUARANTINE_SYMLINK_ROOT/.incomplete"
ln -s "$QUARANTINE_FOREIGN" "$QUARANTINE_SYMLINK_ROOT/.incomplete/$SCENARIO_ID"

QUARANTINE_INDEX="$WORKDIR/quarantine-symlink-index"
mkdir -p "$QUARANTINE_INDEX/packages/$SCENARIO_ID"
cat >"$QUARANTINE_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$QUARANTINE_INDEX/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/quarantine-symlink-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$QUARANTINE_INDEX/scenarios.yaml"
EOF

quarantine_symlink_rc=0
"$BATCH" --batch "$WORKDIR/quarantine-symlink-batch-policy.yaml" --retention-root "$QUARANTINE_SYMLINK_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" --reclaim-incomplete >"$WORKDIR/quarantine-symlink.out" 2>&1 || quarantine_symlink_rc=$?
[[ "$quarantine_symlink_rc" -eq 4 ]] || fail "quarantine_pair symlink: expected exit 4 (quarantine refused, pair recorded failed), got $quarantine_symlink_rc: $(cat "$WORKDIR/quarantine-symlink.out")"
grep -q "pre-existing symlink" "$WORKDIR/quarantine-symlink.out" || fail "quarantine_pair symlink: expected a diagnostic naming the pre-existing symlink: $(cat "$WORKDIR/quarantine-symlink.out")"
[[ -L "$QUARANTINE_SYMLINK_ROOT/.incomplete/$SCENARIO_ID" ]] || fail "quarantine_pair symlink: the pre-existing symlink at .incomplete/<scenario_id> was unexpectedly removed/replaced"
[[ "$(ls "$QUARANTINE_FOREIGN")" == "leftover.txt" ]] \
	|| fail "quarantine_pair symlink: the foreign directory the symlink pointed to had content planted into it -- $(ls "$QUARANTINE_FOREIGN")"
[[ -f "$QUARANTINE_INCOMPLETE_DIR/package.yaml" ]] || fail "quarantine_pair symlink: the original incomplete prior attempt directory was removed -- REQ-NF-007 forbids delete"
AFTER_QUARANTINE_DIGEST="$(sha256sum "$QUARANTINE_INCOMPLETE_DIR/package.yaml" | awk '{print $1}')"
[[ "$AFTER_QUARANTINE_DIGEST" == "$QUARANTINE_BEFORE_DIGEST" ]] || fail "quarantine_pair symlink: the original incomplete prior attempt's bytes were modified"

echo "TC-082(quarantine_pair symlink, round-4 sweep): quarantine_pair refuses to move an incomplete prior attempt through a pre-existing symlink at .incomplete/<scenario_id> -- foreign directory untouched, original attempt left in place"

# ===========================================================================
# STRUCTURAL FIX regression (code-review-2026-08-21T1335-E40-F10.md round 5,
# user decision 2026-08-21): three new cases proving the shared
# assert_no_symlink_in_chain / assert_source_not_symlink primitives
# (bench/scripts/lib/path-safety.sh) close the two gaps round 5 found in
# round 4's own "full audit sweep" -- an ANCESTOR-level symlink (not the
# rep-directory leaf every prior case here plants), a symlinked scratch_root
# defeating `cp -a` isolation, and a direct unit-level proof that the new
# primitive rejects a symlink at an arbitrary INTERMEDIATE chain component,
# not just the leaf or its immediate parent (the shape no existing test --
# including this suite's own leaf-level and ancestor-level cases -- exercises,
# since both of those only ever plant a symlink exactly one level above their
# own leaf).
# ===========================================================================

# ---------------------------------------------------------------------------
# (round-5 finding 1) classify_pair: symlink at an ANCESTOR of the
# rep-level directory -- scenarios/<scenario_id> itself, NOT
# scenarios/<scenario_id>/<rep> (every symlink case above, including
# "classify_pair symlink", plants the symlink exactly at the rep
# directory). This redirect lands fully inside out_root_canon
# (assert_within_out_root's containment check passes) and the rep-level
# directory ENTRY reached through it is real, not itself a symlink --
# invisible to a leaf-only -L test, since only the ANCESTOR redirects.
# Live-reproduced exactly as code-review-2026-08-21T1335-E40-F10.md's own
# "Confirmed reproduction, finding 1" section did against the real,
# unmodified (pre-fix) driver: a real run-lifecycle-batch.sh --mode pilot
# run silently reported "skipped_complete" for a scenario never actually
# retained.
# ---------------------------------------------------------------------------
SYMLINK_ANCESTOR_ROOT="$WORKDIR/symlink-ancestor"
mkdir -p "$SYMLINK_ANCESTOR_ROOT"

FOREIGN_SCENARIO_ANCESTOR="scenario-tc082-ancestor-foreign"
FOREIGN_DIR_ANCESTOR="$SYMLINK_ANCESTOR_ROOT/scenarios/$FOREIGN_SCENARIO_ANCESTOR"
mkdir -p "$FOREIGN_DIR_ANCESTOR/1"
cp "$SOURCES/evaluation.jsonl" "$FOREIGN_DIR_ANCESTOR/1/evaluation.jsonl"
# Manifest forged to claim the VICTIM scenario_id/rep -- content that would
# PASS pair_provenance_ok() if this directory were ever reached at all,
# isolating the ancestor-symlink gap specifically (not a provenance gap).
python3 -c 'import json,sys
scenario_id, rep, dest = sys.argv[1:4]
manifest = {"scenario_id": scenario_id, "rep": int(rep), "artifacts": {}}
open(dest, "w").write(json.dumps(manifest, sort_keys=True, separators=(",", ":")) + "\n")' "$SCENARIO_ID" "$REP" "$FOREIGN_DIR_ANCESTOR/1/manifest.json"

mkdir -p "$SYMLINK_ANCESTOR_ROOT/scenarios"
# The ANCESTOR symlink itself: scenarios/$SCENARIO_ID (one level above the
# rep directory), pointing at the foreign scenario's own real directory.
ln -s "$FOREIGN_DIR_ANCESTOR" "$SYMLINK_ANCESTOR_ROOT/scenarios/$SCENARIO_ID"

SYMLINK_ANCESTOR_INDEX="$WORKDIR/symlink-ancestor-index"
mkdir -p "$SYMLINK_ANCESTOR_INDEX/packages/$SCENARIO_ID"
cat >"$SYMLINK_ANCESTOR_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCENARIO_ID
EOF
cp "$SOURCES/package.yaml" "$SYMLINK_ANCESTOR_INDEX/packages/$SCENARIO_ID/package.yaml"
cat >"$WORKDIR/symlink-ancestor-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$SYMLINK_ANCESTOR_INDEX/scenarios.yaml"
EOF

symlink_ancestor_rc=0
"$BATCH" --batch "$WORKDIR/symlink-ancestor-batch-policy.yaml" --retention-root "$SYMLINK_ANCESTOR_ROOT" --mode pilot \
	"${GOOD_CEILINGS[@]}" >"$WORKDIR/symlink-ancestor.out" 2>&1 || symlink_ancestor_rc=$?
[[ "$symlink_ancestor_rc" -eq 4 ]] || fail "classify_pair ancestor symlink (round-5 finding 1): expected exit 4 (pair recorded failed, never treated as complete), got $symlink_ancestor_rc: $(cat "$WORKDIR/symlink-ancestor.out")"
grep -q "skipped_complete" "$WORKDIR/symlink-ancestor.out" && fail "classify_pair ancestor symlink (round-5 finding 1): the pair reached through an ancestor-level symlink was classified skipped_complete -- expected a refusal instead: $(cat "$WORKDIR/symlink-ancestor.out")"
grep -qi "symlink" "$WORKDIR/symlink-ancestor.out" || fail "classify_pair ancestor symlink (round-5 finding 1): expected a diagnostic naming the symlink: $(cat "$WORKDIR/symlink-ancestor.out")"
[[ -L "$SYMLINK_ANCESTOR_ROOT/scenarios/$SCENARIO_ID" ]] || fail "classify_pair ancestor symlink (round-5 finding 1): the pre-existing ANCESTOR symlink was unexpectedly removed/replaced"
[[ "$(cat "$FOREIGN_DIR_ANCESTOR/1/evaluation.jsonl")" == "$(cat "$SOURCES/evaluation.jsonl")" ]] \
	|| fail "classify_pair ancestor symlink (round-5 finding 1): the foreign scenario's own retained evaluation.jsonl was modified"
grep -q "pre_existing_symlink_at_rep_directory" "$SYMLINK_ANCESTOR_ROOT/invalid/index.jsonl" \
	|| fail "classify_pair ancestor symlink (round-5 finding 1): invalid/index.jsonl did not record the pre_existing_symlink_at_rep_directory reason: $(cat "$SYMLINK_ANCESTOR_ROOT/invalid/index.jsonl" 2>/dev/null || echo MISSING)"

echo "TC-082(classify_pair ancestor-level symlink, round-5 finding 1): classify_pair now refuses to treat a scenario reached through an ANCESTOR-level symlink (scenarios/<scenario_id> itself, one level above the rep directory) as skipped_complete -- structurally closed via assert_no_symlink_in_chain"

# ---------------------------------------------------------------------------
# (round-5 finding 2) dispatch_pair: a symlinked scratch_root must be
# refused BEFORE `cp -a`, not silently copied. GNU `cp -a` implies
# --no-dereference for a TOP-LEVEL symlink source argument, so a symlinked
# scratch_root would previously have produced an "ephemeral" copy that is
# itself a symlink to the SAME real template -- any write a worker makes
# into what it believes is an isolated copy mutates the operator's real
# template. This drives the REAL dispatch_pair() path (stubbed
# RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN, same TD-077 substitution
# convention as (a2) above) with a scratch_root that is a symlink to a real
# template directory, and proves: the batch refuses the pair (never
# silently proceeds), the stub is never invoked (no lifecycle/evaluation
# output is produced), and the real template's own content is provably
# untouched.
# ---------------------------------------------------------------------------
SCRATCH_SYMLINK_ROOT="$WORKDIR/scratch-symlink-root"
mkdir -p "$SCRATCH_SYMLINK_ROOT"

SCRATCH_REAL_TEMPLATE="$WORKDIR/scratch-real-template"
mkdir -p "$SCRATCH_REAL_TEMPLATE"
echo "ORIGINAL TEMPLATE CONTENT -- must never be mutated" >"$SCRATCH_REAL_TEMPLATE/marker.txt"
SCRATCH_TEMPLATE_BEFORE="$(sha256sum "$SCRATCH_REAL_TEMPLATE/marker.txt" | awk '{print $1}')"

SCRATCH_SYMLINK="$WORKDIR/scratch-symlink"
ln -s "$SCRATCH_REAL_TEMPLATE" "$SCRATCH_SYMLINK"

SCRATCH_SYMLINK_SCENARIO="scenario-tc082-scratch-symlink"
SCRATCH_SYMLINK_INDEX="$WORKDIR/scratch-symlink-index"
mkdir -p "$SCRATCH_SYMLINK_INDEX/packages/$SCRATCH_SYMLINK_SCENARIO"
cat >"$SCRATCH_SYMLINK_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCRATCH_SYMLINK_SCENARIO
EOF
cat >"$SCRATCH_SYMLINK_INDEX/packages/$SCRATCH_SYMLINK_SCENARIO/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$SCRATCH_SYMLINK_SCENARIO"
scenario_version: "1"
entity_family: "family-tc082-scratch-symlink"
EOF
cat >"$WORKDIR/scratch-symlink-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$SCRATCH_SYMLINK_INDEX/scenarios.yaml"
scenarios:
  $SCRATCH_SYMLINK_SCENARIO:
    root_key: "ROOT-TC082-SCRATCH-SYMLINK"
    scratch_root: "$SCRATCH_SYMLINK"
    reps: 1
EOF

# A stub that, if it were EVER invoked, proves this test would have caught a
# regression: it writes into its own --scratch-root argument (mirroring what
# a real lifecycle worker does) so a mutation would be observable on the
# real template if isolation were broken. Never expected to run at all
# (dispatch is refused before cp -a), but present so a future regression
# that removed the refusal (rather than merely making it non-loud) would
# still be caught by the digest assertion below, not just by an exit-code
# check.
SCRATCH_SYMLINK_STUB="$WORKDIR/scratch-symlink-stub.sh"
cat >"$SCRATCH_SYMLINK_STUB" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
scratch_root=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--scratch-root) scratch_root="$2"; shift 2 ;;
	*) shift ;;
	esac
done
echo "WRITTEN BY STUB (should never run)" >>"$scratch_root/marker.txt"
exit 0
EOF
chmod +x "$SCRATCH_SYMLINK_STUB"

scratch_symlink_rc=0
RUN_LIFECYCLE_BIN="$SCRATCH_SYMLINK_STUB" \
	"$BATCH" --batch "$WORKDIR/scratch-symlink-batch-policy.yaml" --retention-root "$SCRATCH_SYMLINK_ROOT" \
	--mode pilot "${GOOD_CEILINGS[@]}" >"$WORKDIR/scratch-symlink.out" 2>&1 || scratch_symlink_rc=$?
[[ "$scratch_symlink_rc" -eq 4 ]] || fail "dispatch_pair scratch_root symlink (round-5 finding 2): expected exit 4 (pair recorded failed, batch proceeds), got $scratch_symlink_rc: $(cat "$WORKDIR/scratch-symlink.out")"
grep -qi "symlink" "$WORKDIR/scratch-symlink.out" || fail "dispatch_pair scratch_root symlink (round-5 finding 2): expected a diagnostic naming the symlink: $(cat "$WORKDIR/scratch-symlink.out")"
grep -q "scratch_root_is_symlink" "$SCRATCH_SYMLINK_ROOT/invalid/index.jsonl" \
	|| fail "dispatch_pair scratch_root symlink (round-5 finding 2): invalid/index.jsonl did not record the scratch_root_is_symlink reason: $(cat "$SCRATCH_SYMLINK_ROOT/invalid/index.jsonl" 2>/dev/null || echo MISSING)"
SCRATCH_TEMPLATE_AFTER="$(sha256sum "$SCRATCH_REAL_TEMPLATE/marker.txt" | awk '{print $1}')"
[[ "$SCRATCH_TEMPLATE_AFTER" == "$SCRATCH_TEMPLATE_BEFORE" ]] \
	|| fail "dispatch_pair scratch_root symlink (round-5 finding 2): the real template's content was mutated -- isolation was NOT preserved: $(cat "$SCRATCH_REAL_TEMPLATE/marker.txt")"
[[ -L "$SCRATCH_SYMLINK" ]] || fail "dispatch_pair scratch_root symlink (round-5 finding 2): the scratch_root symlink itself was unexpectedly removed/replaced"

echo "TC-082(dispatch_pair scratch_root symlink, round-5 finding 2): a symlinked scratch_root is refused before cp -a -- the real template is provably untouched, the stub lifecycle worker is never invoked"

# ---------------------------------------------------------------------------
# dispatch_pair scratch_root NESTED symlink (round-6 finding 1,
# code-review-2026-08-21T1141-E40-F10.md): a REAL (non-symlink) scratch_root
# directory containing a NESTED symlinked subdirectory used to survive
# `cp -a` as a live symlink at the same relative path in the "ephemeral"
# copy -- `cp -a` preserves every symlink it encounters during a recursive
# copy, not just a top-level symlink source argument, so a worker write
# through the believed-isolated ephemeral copy would silently reach
# whatever the nested symlink targets. Unlike the top-level case above
# (round-5 finding 2, which is refused outright before any copy runs),
# this case is a REAL scratch_root -- assert_source_not_symlink correctly
# allows it through -- and the driver is now expected to PROCEED
# (copy_tree_dereferenced dereferences the nested symlink into a real,
# independent copy) rather than refuse. Proves: the dispatch succeeds, the
# stub lifecycle worker's write through the nested-symlink-shaped path in
# its ephemeral scratch-root argument lands only in the ephemeral copy, and
# the external target the nested symlink points to is provably untouched.
# ---------------------------------------------------------------------------
SCRATCH_NESTED_ROOT="$WORKDIR/scratch-nested-root"
mkdir -p "$SCRATCH_NESTED_ROOT"

SCRATCH_NESTED_EXTERNAL="$WORKDIR/scratch-nested-external"
mkdir -p "$SCRATCH_NESTED_EXTERNAL"
echo "ORIGINAL EXTERNAL CONTENT -- must never be mutated" >"$SCRATCH_NESTED_EXTERNAL/marker.txt"
SCRATCH_NESTED_EXTERNAL_BEFORE="$(sha256sum "$SCRATCH_NESTED_EXTERNAL/marker.txt" | awk '{print $1}')"

# The scratch_root itself is a REAL directory (not a symlink) -- only a
# subdirectory nested inside it ("prompts") is a symlink to the external
# target, mirroring the shape production `run-lifecycle.sh` writes into
# (scratch/prompts/<ordinal>).
mkdir -p "$SCRATCH_NESTED_ROOT"
echo "real readme, not a symlink" >"$SCRATCH_NESTED_ROOT/readme.txt"
ln -s "$SCRATCH_NESTED_EXTERNAL" "$SCRATCH_NESTED_ROOT/prompts"

# i05_bundle_dir must be configured for dispatch_pair to reach a genuine
# exit-0 success (not "i05_bundle_not_configured") -- mirrors (a2)'s
# DRIVER_I05_BUNDLE above.
SCRATCH_NESTED_I05_BUNDLE="$WORKDIR/scratch-nested-i05-bundle"
mkdir -p "$SCRATCH_NESTED_I05_BUNDLE/transcripts"
echo '{"stage": "code", "note": "tc082 nested-symlink evidence"}' >"$SCRATCH_NESTED_I05_BUNDLE/stage.json"
echo "tc082 nested-symlink transcript" >"$SCRATCH_NESTED_I05_BUNDLE/transcripts/stage.txt"

SCRATCH_NESTED_SCENARIO="scenario-tc082-scratch-nested-symlink"
SCRATCH_NESTED_INDEX="$WORKDIR/scratch-nested-index"
mkdir -p "$SCRATCH_NESTED_INDEX/packages/$SCRATCH_NESTED_SCENARIO"
cat >"$SCRATCH_NESTED_INDEX/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$SCRATCH_NESTED_SCENARIO
EOF
cat >"$SCRATCH_NESTED_INDEX/packages/$SCRATCH_NESTED_SCENARIO/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$SCRATCH_NESTED_SCENARIO"
scenario_version: "1"
entity_family: "family-tc082-scratch-nested-symlink"
EOF
cat >"$WORKDIR/scratch-nested-batch-policy.yaml" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$SCRATCH_NESTED_INDEX/scenarios.yaml"
scenarios:
  $SCRATCH_NESTED_SCENARIO:
    root_key: "ROOT-TC082-SCRATCH-NESTED-SYMLINK"
    scratch_root: "$SCRATCH_NESTED_ROOT"
    i05_bundle_dir: "$SCRATCH_NESTED_I05_BUNDLE"
    reps: 1
EOF

SCRATCH_NESTED_ROOT_OUT="$WORKDIR/scratch-nested-root-out"
mkdir -p "$SCRATCH_NESTED_ROOT_OUT"

# A stub RUN_LIFECYCLE_BIN that writes through its own --scratch-root
# argument's nested "prompts" path (mirroring what a real lifecycle worker
# does), and also produces a well-formed lifecycle.jsonl (via cp of the
# committed I-07 fixture) plus a matching EVALUATE_LIFECYCLE_BIN stub so
# this run reaches retain_pair and a full, real success path -- proving
# isolation holds all the way through a genuine dispatch, not merely up to
# a refusal.
SCRATCH_NESTED_STUB="$WORKDIR/scratch-nested-stub.sh"
cat >"$SCRATCH_NESTED_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
scratch_root=""
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--scratch-root) scratch_root="\$2"; shift 2 ;;
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
echo "WRITTEN BY STUB THROUGH NESTED PATH" >>"\$scratch_root/prompts/marker.txt"
cp "$I07_FIXTURE" "\$output"
exit 0
EOF
chmod +x "$SCRATCH_NESTED_STUB"

SCRATCH_NESTED_EVAL_STUB="$WORKDIR/scratch-nested-eval-stub.sh"
cat >"$SCRATCH_NESTED_EVAL_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
cp "$I08_FIXTURE" "\$output"
touch "\$output.oracle.json"
exit 0
EOF
chmod +x "$SCRATCH_NESTED_EVAL_STUB"

scratch_nested_rc=0
RUN_LIFECYCLE_BIN="$SCRATCH_NESTED_STUB" EVALUATE_LIFECYCLE_BIN="$SCRATCH_NESTED_EVAL_STUB" \
	"$BATCH" --batch "$WORKDIR/scratch-nested-batch-policy.yaml" --retention-root "$SCRATCH_NESTED_ROOT_OUT" \
	--mode pilot "${GOOD_CEILINGS[@]}" >"$WORKDIR/scratch-nested.out" 2>&1 || scratch_nested_rc=$?
[[ "$scratch_nested_rc" -eq 0 ]] || fail "dispatch_pair scratch_root NESTED symlink (round-6 finding 1): expected exit 0 (a real scratch_root with a nested symlink must be dispatched, not refused), got $scratch_nested_rc: $(cat "$WORKDIR/scratch-nested.out")"

SCRATCH_NESTED_EXTERNAL_AFTER="$(sha256sum "$SCRATCH_NESTED_EXTERNAL/marker.txt" | awk '{print $1}')"
[[ "$SCRATCH_NESTED_EXTERNAL_AFTER" == "$SCRATCH_NESTED_EXTERNAL_BEFORE" ]] \
	|| fail "dispatch_pair scratch_root NESTED symlink (round-6 finding 1): the external target of the nested symlink was mutated -- isolation was NOT preserved: $(cat "$SCRATCH_NESTED_EXTERNAL/marker.txt")"
[[ -L "$SCRATCH_NESTED_ROOT/prompts" ]] || fail "dispatch_pair scratch_root NESTED symlink (round-6 finding 1): the original scratch_root's own nested symlink was unexpectedly removed/replaced"

echo "TC-082(dispatch_pair scratch_root NESTED symlink, round-6 finding 1): a real scratch_root containing a nested symlink is dispatched successfully (not refused), the worker's write through the nested path lands only in the dereferenced ephemeral copy, and the external target is provably untouched"

# ---------------------------------------------------------------------------
# Unit-level proof (round-5 report, Section E "Missing boundary cases" and
# this dispatch's own explicit instruction): assert_no_symlink_in_chain
# must reject a symlink at an ARBITRARY INTERMEDIATE path component -- not
# just the leaf (the "classify_pair symlink" case above) or the leaf's
# immediate parent/ancestor (the "ancestor symlink" case above) -- since
# both of THOSE cases only ever plant a symlink exactly one level above
# their own leaf. This drives bench/scripts/lib/path-safety.sh's shared
# primitive directly (sourced in a subshell, no driver invocation needed)
# against a synthetic root/leaf pair four levels deep, with the symlink
# planted at level 2 of 4 -- neither the leaf (level 4) nor its immediate
# parent (level 3).
# ---------------------------------------------------------------------------
CHAIN_UNIT_ROOT="$WORKDIR/chain-unit-root"
mkdir -p "$CHAIN_UNIT_ROOT/level1"
CHAIN_UNIT_FOREIGN="$WORKDIR/chain-unit-foreign"
mkdir -p "$CHAIN_UNIT_FOREIGN"
echo "leftover" >"$CHAIN_UNIT_FOREIGN/leftover.txt"
# level2 (an INTERMEDIATE component: two levels below root, two levels
# above the leaf) is the symlink -- neither the leaf nor its immediate
# parent.
ln -s "$CHAIN_UNIT_FOREIGN" "$CHAIN_UNIT_ROOT/level1/level2"

chain_unit_rc=0
bash -c '
	source "'"$SCRIPTS_DIR"'/lib/path-safety.sh"
	assert_no_symlink_in_chain "'"$CHAIN_UNIT_ROOT"'" "'"$CHAIN_UNIT_ROOT"'/level1/level2/level3/level4"
' >"$WORKDIR/chain-unit.out" 2>&1 || chain_unit_rc=$?
[[ "$chain_unit_rc" -ne 0 ]] || fail "assert_no_symlink_in_chain unit test: expected a nonzero (refused) exit for a symlink at an intermediate chain component (level2 of a 4-level path), got 0: $(cat "$WORKDIR/chain-unit.out")"
grep -q "level1/level2" "$WORKDIR/chain-unit.out" || fail "assert_no_symlink_in_chain unit test: expected the diagnostic to name the exact intermediate symlinked path (level1/level2): $(cat "$WORKDIR/chain-unit.out")"

# Counter-proof: the identical chain WITHOUT the intermediate symlink (a
# genuinely deep, real path) must be reported safe -- proving the refusal
# above is caused by the planted symlink, not by chain depth itself.
CHAIN_UNIT_ROOT_CLEAN="$WORKDIR/chain-unit-root-clean"
mkdir -p "$CHAIN_UNIT_ROOT_CLEAN/level1/level2/level3"
chain_unit_clean_rc=0
bash -c '
	source "'"$SCRIPTS_DIR"'/lib/path-safety.sh"
	assert_no_symlink_in_chain "'"$CHAIN_UNIT_ROOT_CLEAN"'" "'"$CHAIN_UNIT_ROOT_CLEAN"'/level1/level2/level3/level4"
' >"$WORKDIR/chain-unit-clean.out" 2>&1 || chain_unit_clean_rc=$?
[[ "$chain_unit_clean_rc" -eq 0 ]] || fail "assert_no_symlink_in_chain unit test counter-proof: a genuinely deep path with no symlink anywhere must be reported safe, got exit $chain_unit_clean_rc: $(cat "$WORKDIR/chain-unit-clean.out")"

echo "TC-082(assert_no_symlink_in_chain intermediate-component unit test, round-5 structural fix): the shared primitive rejects a symlink at an arbitrary intermediate path component (neither the leaf nor its immediate parent), and does not falsely reject an equally deep symlink-free path"
