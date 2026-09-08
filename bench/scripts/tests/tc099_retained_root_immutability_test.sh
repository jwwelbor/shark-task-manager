#!/usr/bin/env bash
# TC-099 / T-E40-F11-001 (spec.md REQ-F-001, AC-F11-01/01a/01b/02;
# test-plan.md TC-01/TC-01a/TC-01b/TC-02): the retention-root immutability
# guard and its evidence registration.
#
# Caller-Path Contract (test-plan.md TC-01 row): production entrypoint is
# `bench/scripts/e40-benchmark.sh <seam>` for each of the 9 operator
# subcommands, including `prepare-replay` (T-E40-F11-006 landed it; this
# sweep now covers all nine per that row's own instruction to add it once
# it exists). Real registry file, real filesystem: `ensure_external_operator_root` and
# `retention_registry_lookup` are never mocked. The only thing this file
# stubs is a downstream `shark` binary for ONE case (see below), which is
# never the guard under test.
#
# AC-F11-01's own text: the guard fires unconditionally -- regardless of a
# registered root's recorded `status` or `candidate_identity.head`. Since
# the new guard never reads either field, the R1-R4 sweep below proves this
# by writing four different marker-file contents into the SAME registered
# root and asserting rejection every time: a regression to the old
# conjunction check (reject only when status != completed AND HEAD
# differs) would wrongly ALLOW R2-R4, and this sweep would catch it.
#
# This file also registers the real 2026-09-04 evidence in
# bench/retention-registry.yaml (a committed, non-test change made by this
# task, not by this test) and cross-checks it against feature.md's
# recorded digests (TC-02).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
OPERATOR="$SCRIPTS_DIR/e40-benchmark.sh"
MODULE="$SCRIPTS_DIR/lib/e40_benchmark.py"
VERIFY_MANIFEST="$SCRIPTS_DIR/verify-retention-manifest.sh"
REGISTRY="$BENCH_DIR/retention-registry.yaml"
FEATURE_DIR="$REPO_ROOT/docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F11-six-family-lifecycle-benchmark-readiness-and-cover"

fail() {
	echo "TC-099 FAIL: $1" >&2
	exit 1
}

[[ -x "$OPERATOR" ]] || fail "e40-benchmark.sh is missing or not executable"
[[ -f "$MODULE" ]] || fail "lib/e40_benchmark.py is missing"
[[ -x "$VERIFY_MANIFEST" ]] || fail "verify-retention-manifest.sh is missing or not executable"
[[ -f "$REGISTRY" ]] || fail "bench/retention-registry.yaml is missing (AC-F11-02 registration not applied)"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
REGISTRY_BACKUP="$WORKDIR/retention-registry.yaml.orig"
cp "$REGISTRY" "$REGISTRY_BACKUP"
cleanup() {
	cp "$REGISTRY_BACKUP" "$REGISTRY"
	rm -rf "$WORKDIR"
}
trap cleanup EXIT

GUARD_MARK="operator root is frozen as retained evidence"

# ---------------------------------------------------------------------------
# AC-T1 / ADR-F11-10 (choke point): exactly one call site invokes
# retention_registry_lookup -- the one inside ensure_external_operator_root.
# A second, parallel call site would mean a second guard was grown instead
# of reusing the one choke point.
# ---------------------------------------------------------------------------
call_sites="$(grep -c 'retention_registry_lookup(' "$MODULE")"
[[ "$call_sites" -eq 2 ]] \
	|| fail "expected exactly one definition + one call site for retention_registry_lookup, found $call_sites total occurrences"
def_and_call="$(grep -n 'retention_registry_lookup(' "$MODULE")"
echo "$def_and_call" | grep -q '^[0-9]*:def retention_registry_lookup' \
	|| fail "retention_registry_lookup definition not found where expected"

# ---------------------------------------------------------------------------
# TC-01a: no operator-specific absolute path appears anywhere in bench/
# SOURCE (AC-F11-01a) -- only in the committed retention-registry.yaml DATA
# file. Excludes this test file itself (it necessarily names the pattern
# it greps for, which is not a hardcoded operator path) and the registry
# data file (the one permitted carrier of the path, per AC-F11-01a).
# ---------------------------------------------------------------------------
HOME_DIR_PATTERN="/home/"
home_hits="$(grep -rln "$HOME_DIR_PATTERN" "$BENCH_DIR" --include='*.py' --include='*.sh' --include='*.yaml' --include='*.yml' 2>/dev/null \
	| grep -v '/retention-registry\.yaml$' \
	| grep -v '/tc099_retained_root_immutability_test\.sh$' || true)"
[[ -z "$home_hits" ]] \
	|| fail "operator-specific absolute path leaked into bench/ source: $home_hits"
grep -q "$HOME_DIR_PATTERN" "$REGISTRY" \
	|| fail "retention-registry.yaml does not carry the registered root's absolute path"

# ===========================================================================
# Fixture construction shared by the reject (registered) and allow
# (unregistered) sweeps. Every config below is deliberately the MINIMUM
# that reaches each seam's ensure_external_operator_root call -- nothing
# past the guard is exercised on the reject path, and the allow path is
# free to fail later for an unrelated, safe reason (readiness, missing
# scratch_template, etc.) as long as it is never THIS guard's reason.
# ===========================================================================
mkdir -p "$WORKDIR/adapter-bin" "$WORKDIR/i05"
cat >"$WORKDIR/adapter-bin/lifecycle-adapter" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$WORKDIR/adapter-bin/lifecycle-adapter"

mkdir -p "$WORKDIR/compare-baseline" "$WORKDIR/compare-variant"
echo '{}' >"$WORKDIR/compare-baseline/benchmark-manifest.json"
echo '{}' >"$WORKDIR/compare-baseline/aggregate.json"
echo '{}' >"$WORKDIR/compare-variant/benchmark-manifest.json"
echo '{}' >"$WORKDIR/compare-variant/aggregate.json"

# write_configs_for_root <root> <prefix>
# Emits <prefix>-baseline.yaml (usable for setup/preflight/validate-variant/
# pilot/baseline) and <prefix>-variant.yaml (profile.role=variant, for the
# `variant` seam) with operator_root pinned to <root>.
write_configs_for_root() {
	local root="$1" prefix="$2"
	cat >"$WORKDIR/$prefix-baseline.yaml" <<EOF
schema_version: "1.0"
operator_root: "$root"
runtime:
  lifecycle_adapter: "$WORKDIR/adapter-bin/lifecycle-adapter"
scenario_roots:
  py-bug-due-date-boundary:
    i05_bundle_dir: "$WORKDIR/i05"
EOF
	cat >"$WORKDIR/$prefix-variant.yaml" <<EOF
schema_version: "1.0"
operator_root: "$root"
profile:
  role: variant
runtime:
  lifecycle_adapter: "$WORKDIR/adapter-bin/lifecycle-adapter"
scenario_roots:
  py-bug-due-date-boundary:
    i05_bundle_dir: "$WORKDIR/i05"
EOF
}

EXEC_FLAGS=(--scenario py-bug-due-date-boundary --acknowledge-provider-spend \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 \
	--max-provider-calls 15)

# invoke_seam <seam> <root> <config_baseline> <config_variant> <out_file>
# Runs one of the 8 shipped seams against <root> (either the registered
# disposable root, for the reject sweep, or a fresh unregistered one, for
# the allow sweep), capturing combined stdout+stderr to <out_file> and
# returning the exit code via $?.
invoke_seam() {
	local seam="$1" root="$2" cfg_baseline="$3" cfg_variant="$4" out_file="$5"
	set +e
	case "$seam" in
	setup)
		timeout 60 "$OPERATOR" setup --out "$root" >"$out_file" 2>&1
		;;
	preflight)
		timeout 60 "$OPERATOR" preflight --config "$cfg_baseline" >"$out_file" 2>&1
		;;
	validate-variant)
		timeout 60 "$OPERATOR" validate-variant --config "$cfg_baseline" --variant-name tc099-variant >"$out_file" 2>&1
		;;
	pilot)
		timeout 60 "$OPERATOR" pilot --config "$cfg_baseline" --run-id tc099-run \
			"${EXEC_FLAGS[@]}" >"$out_file" 2>&1
		;;
	pilot-retry-incomplete)
		timeout 60 "$OPERATOR" pilot --config "$cfg_baseline" --run-id tc099-run --retry-incomplete \
			"${EXEC_FLAGS[@]}" >"$out_file" 2>&1
		;;
	baseline)
		timeout 60 "$OPERATOR" baseline --config "$cfg_baseline" --run-id tc099-run \
			"${EXEC_FLAGS[@]}" >"$out_file" 2>&1
		;;
	variant)
		timeout 60 "$OPERATOR" variant --config "$cfg_variant" --run-id tc099-run \
			"${EXEC_FLAGS[@]}" >"$out_file" 2>&1
		;;
	compare)
		timeout 60 "$OPERATOR" compare --baseline "$WORKDIR/compare-baseline" \
			--variant "$WORKDIR/compare-variant" --out "$root/tc099-comparison" >"$out_file" 2>&1
		;;
	demo)
		timeout 60 "$OPERATOR" demo --out "$root" >"$out_file" 2>&1
		;;
	prepare-replay)
		timeout 60 "$OPERATOR" prepare-replay --config "$cfg_baseline" \
			--scenario py-bug-due-date-boundary >"$out_file" 2>&1
		;;
	*)
		fail "unknown seam in invoke_seam: $seam"
		;;
	esac
	local rc=$?
	set -e
	return "$rc"
}

# The 9 seams named by AC-F11-01, CLI subcommand names. `prepare-replay`
# (T-E40-F11-006) is the 9th and reaches the same
# `ensure_external_operator_root` choke point as its very first action
# (ADR-F11-10), before any scenario resolution or ceiling check -- so a
# minimal `--config <cfg> --scenario <any>` invocation (no acknowledgement
# or ceilings needed) already exercises the guard.
SEAMS=(setup preflight validate-variant pilot baseline variant compare demo prepare-replay)

# ===========================================================================
# TC-01: R1-R4 x 9 seams = 36 rejection cases. Four marker-file variants
# stand in for the four retained
# status/HEAD combinations the pre-rework criterion distinguished; the new
# guard must reject all four identically because it never reads either
# field (AC-F11-01's "Supersedes the pre-rework criterion").
# ===========================================================================
DISPOSABLE_ID="tc099-disposable-root"
DISPOSABLE_ROOT="$WORKDIR/disposable-root"
DISPOSABLE_MANIFEST="$WORKDIR/disposable-root.sha256"
mkdir -p "$DISPOSABLE_ROOT"

cat >>"$REGISTRY" <<EOF
  - registry_id: "$DISPOSABLE_ID"
    root_path: "$DISPOSABLE_ROOT"
    registered_at: "2026-09-06T00:00:00Z"
    classification: "incomplete"
    tree_manifest_path: "$DISPOSABLE_MANIFEST"
EOF

write_configs_for_root "$DISPOSABLE_ROOT" "reject"

tree_snapshot() {
	find "$1" -type f -exec sha256sum {} \; | sort
}

reject_count=0
declare -a CONDITIONS=(
	"R1:incomplete:differs"
	"R2:incomplete:matches"
	"R3:completed:differs"
	"R4:completed:matches"
)
for condition in "${CONDITIONS[@]}"; do
	IFS=":" read -r row status head <<<"$condition"
	cat >"$DISPOSABLE_ROOT/operator-result.json" <<EOF
{"status": "$status", "candidate_identity": {"head": "$head-from-current-head"}}
EOF
	before="$(tree_snapshot "$DISPOSABLE_ROOT")"
	for seam in "${SEAMS[@]}"; do
		out_file="$WORKDIR/reject-$row-$seam.out"
		rc=0
		invoke_seam "$seam" "$DISPOSABLE_ROOT" "$WORKDIR/reject-baseline.yaml" "$WORKDIR/reject-variant.yaml" "$out_file" || rc=$?
		[[ "$rc" -ne 0 ]] \
			|| fail "$row/$seam: expected non-zero exit against a registered root, got 0: $(cat "$out_file")"
		grep -q "$GUARD_MARK" "$out_file" \
			|| fail "$row/$seam: retention guard did not fire: $(cat "$out_file")"
		grep -q "registry_id=$DISPOSABLE_ID" "$out_file" \
			|| fail "$row/$seam: stderr missing registry_id: $(cat "$out_file")"
		grep -q "root_path=$DISPOSABLE_ROOT" "$out_file" \
			|| fail "$row/$seam: stderr missing root_path: $(cat "$out_file")"
		reject_count=$((reject_count + 1))
	done
	after="$(tree_snapshot "$DISPOSABLE_ROOT")"
	[[ "$before" == "$after" ]] \
		|| fail "$row: registered root was not byte-identical after the seam sweep"
done
[[ "$reject_count" -eq 36 ]] \
	|| fail "expected 36 reject cases (4 conditions x 9 seams), got $reject_count"
echo "TC-01 (reject sweep): $reject_count/36 R1-R4 x seam cases rejected, registered root byte-identical -- PASS"

# AC-F11-01: "plus --retry-incomplete on any of them" -- the flag must
# never bypass the guard. pilot-retry-incomplete is not a 9th seam; it is
# the same `pilot` seam with the flag added, checked once here.
retry_out="$WORKDIR/reject-retry-incomplete.out"
retry_rc=0
invoke_seam "pilot-retry-incomplete" "$DISPOSABLE_ROOT" "$WORKDIR/reject-baseline.yaml" "$WORKDIR/reject-variant.yaml" "$retry_out" || retry_rc=$?
[[ "$retry_rc" -ne 0 ]] || fail "pilot --retry-incomplete: expected non-zero exit against a registered root, got 0"
grep -q "$GUARD_MARK" "$retry_out" || fail "pilot --retry-incomplete: retention guard did not fire: $(cat "$retry_out")"
echo "TC-01 (--retry-incomplete): flag does not bypass the guard -- PASS"

# ===========================================================================
# TC-01: R5 -- an unregistered root is allowed by every seam (the guard
# specifically does not fire; each seam is free to fail afterward for its
# own, unrelated reason).
# ===========================================================================
FRESH_ROOT="$WORKDIR/fresh-root"
mkdir -p "$FRESH_ROOT"
write_configs_for_root "$FRESH_ROOT" "allow"

allow_count=0
for seam in "${SEAMS[@]}"; do
	out_file="$WORKDIR/allow-$seam.out"
	invoke_seam "$seam" "$FRESH_ROOT" "$WORKDIR/allow-baseline.yaml" "$WORKDIR/allow-variant.yaml" "$out_file" || true
	grep -q "$GUARD_MARK" "$out_file" \
		&& fail "allow/$seam: retention guard fired against an unregistered root: $(cat "$out_file")"
	allow_count=$((allow_count + 1))
	rm -rf "${FRESH_ROOT:?}"/*
done
[[ "$allow_count" -eq 9 ]] || fail "expected 9 allow cases (all seams), got $allow_count"
echo "TC-01 (allow sweep / R5): $allow_count/9 seams allowed an unregistered root -- PASS"

# ===========================================================================
# TC-01b: whole-tree manifest invariance, 5 mutation kinds, on a DISPOSABLE
# COPY -- never on the registered root itself (Caller-Path Contract: "Do
# not stub the manifest file").
# ===========================================================================
MUTATION_ID="tc099-mutation-fixture"
MUTATION_ROOT="$WORKDIR/mutation-root"
MUTATION_MANIFEST="$WORKDIR/mutation-root.sha256"
mkdir -p "$MUTATION_ROOT/a/b"
echo "one" >"$MUTATION_ROOT/file1.txt"
echo "two" >"$MUTATION_ROOT/a/file2.txt"
echo "three" >"$MUTATION_ROOT/a/b/file3.txt"
ln -s "file1.txt" "$MUTATION_ROOT/link-to-file1"

python3 - "$MODULE" "$MUTATION_ROOT" "$MUTATION_MANIFEST" <<'PY'
import importlib.util
import pathlib
import sys

module_path, root, manifest_path = sys.argv[1:4]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
pathlib.Path(manifest_path).write_text(
    module.compute_tree_manifest(pathlib.Path(root)), encoding="utf-8"
)
PY
[[ -f "$MUTATION_MANIFEST" ]] || fail "failed to generate the mutation-fixture manifest"

cat >>"$REGISTRY" <<EOF
  - registry_id: "$MUTATION_ID"
    root_path: "$MUTATION_ROOT"
    registered_at: "2026-09-06T00:00:00Z"
    classification: "incomplete"
    tree_manifest_path: "$MUTATION_MANIFEST"
EOF

# Unmutated: must match.
match_out="$WORKDIR/mutation-match.out"
set +e
"$VERIFY_MANIFEST" "$MUTATION_ID" >"$match_out" 2>&1
match_rc=$?
set -e
[[ "$match_rc" -eq 0 ]] || fail "unmutated fixture reported mismatch: $(cat "$match_out")"
grep -q '"status": "match"' "$match_out" || fail "unmutated fixture did not report status match: $(cat "$match_out")"

assert_mutation_detected() {
	local label="$1" mutated_root="$2" expected_path="$3"
	local out_file="$WORKDIR/mutation-$label.out"
	set +e
	"$VERIFY_MANIFEST" "$MUTATION_ID" --root "$mutated_root" >"$out_file" 2>&1
	local rc=$?
	set -e
	[[ "$rc" -eq 1 ]] || fail "$label: expected exit 1 (mismatch), got $rc: $(cat "$out_file")"
	grep -q '"status": "mismatch"' "$out_file" || fail "$label: did not report status mismatch: $(cat "$out_file")"
	grep -q "\"$expected_path\"" "$out_file" || fail "$label: differing path '$expected_path' not named: $(cat "$out_file")"
}

# (1) file modified
copy1="$WORKDIR/mutation-copy-modified"
cp -r "$MUTATION_ROOT" "$copy1"
echo "one-modified" >"$copy1/file1.txt"
assert_mutation_detected "modified" "$copy1" "file1.txt"

# (2) file added
copy2="$WORKDIR/mutation-copy-added"
cp -r "$MUTATION_ROOT" "$copy2"
echo "new" >"$copy2/a/new-file.txt"
assert_mutation_detected "added" "$copy2" "a/new-file.txt"

# (3) file removed
copy3="$WORKDIR/mutation-copy-removed"
cp -r "$MUTATION_ROOT" "$copy3"
rm "$copy3/a/b/file3.txt"
assert_mutation_detected "removed" "$copy3" "a/b/file3.txt"

# (4) file renamed (surfaces as one removed + one added path)
copy4="$WORKDIR/mutation-copy-renamed"
cp -r "$MUTATION_ROOT" "$copy4"
mv "$copy4/a/file2.txt" "$copy4/a/file2-renamed.txt"
assert_mutation_detected "renamed" "$copy4" "a/file2.txt"
grep -q '"a/file2-renamed.txt"' "$WORKDIR/mutation-renamed.out" \
	|| fail "renamed: new path a/file2-renamed.txt not named: $(cat "$WORKDIR/mutation-renamed.out")"

# (5) symlink retargeted
copy5="$WORKDIR/mutation-copy-symlink"
cp -r "$MUTATION_ROOT" "$copy5"
rm "$copy5/link-to-file1"
ln -s "a/file2.txt" "$copy5/link-to-file1"
assert_mutation_detected "symlink-retargeted" "$copy5" "link-to-file1"

# The real fixture root itself must never have been touched by any of the
# above -- re-verifying it directly (no --root override) must still match.
set +e
"$VERIFY_MANIFEST" "$MUTATION_ID" >"$WORKDIR/mutation-final-check.out" 2>&1
final_rc=$?
set -e
[[ "$final_rc" -eq 0 ]] || fail "mutation fixture root was altered by the sweep: $(cat "$WORKDIR/mutation-final-check.out")"

echo "TC-01b: 5/5 mutation kinds detected on disposable copies, fixture root untouched -- PASS"

# ===========================================================================
# TC-02: evidence/2026-09-04-retained-root.md cross-checked against
# feature.md's §Observed evidence digests, plus classification and
# registry_id cross-link.
# ===========================================================================
EVIDENCE_FILE="$FEATURE_DIR/evidence/2026-09-04-retained-root.md"
FEATURE_FILE="$FEATURE_DIR/feature.md"
[[ -f "$EVIDENCE_FILE" ]] || fail "evidence/2026-09-04-retained-root.md is missing"
[[ -f "$FEATURE_FILE" ]] || fail "feature.md is missing"

# The label "Preflight result SHA-256" also appears inside each file's own
# cross-reference prose, so pattern-anchoring on that label is ambiguous.
# Both files record exactly one SHA-256 value; take the first 64-hex-char
# match from each rather than anchoring on the label line.
feature_digest="$(grep -o '[0-9a-f]\{64\}' "$FEATURE_FILE" | head -n1)"
evidence_digest="$(grep -o '[0-9a-f]\{64\}' "$EVIDENCE_FILE" | head -n1)"
[[ -n "$feature_digest" ]] || fail "could not extract preflight SHA-256 from feature.md"
[[ "$feature_digest" == "$evidence_digest" ]] \
	|| fail "evidence file's preflight SHA-256 ($evidence_digest) does not match feature.md's ($feature_digest)"

grep -qi 'classification.*`incomplete`' "$EVIDENCE_FILE" \
	|| fail "evidence file does not record classification: incomplete"
grep -q 'registry_id: 2026-09-04-e40-full-family' "$EVIDENCE_FILE" \
	|| fail "evidence file does not cross-link its registry_id"

# The registry_id named in the evidence file must resolve to a real entry
# whose classification is genuinely incomplete. AC-F11-02 is a document-vs-
# document check (row 02's own Seam column: "n/a (static)") -- the digest
# equality asserted above (evidence file vs feature.md) is what the AC
# actually requires, and that check is host-independent (both are text
# files committed to the repo). The retained root itself is machine-local
# operator evidence (AC-F11-01a): on a host that never held it, re-digesting
# the live file would wrongly fail this test everywhere except the
# machine that captured the evidence. So the live re-digest below is only
# an ADDITIONAL cross-check, run only when the root is actually present on
# this host; when it is not, `verify_tree_manifest` must report
# `not_present_on_host` (never `mismatch`) for the same reason.
python3 - "$MODULE" "$feature_digest" <<'PY'
import importlib.util
import pathlib
import sys

module_path, expected_digest = sys.argv[1:3]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

entry = module.retention_registry_entry("2026-09-04-e40-full-family")
if entry["classification"] != "incomplete":
    raise SystemExit(f"registry entry classification is {entry['classification']!r}, expected 'incomplete'")

root = pathlib.Path(entry["root_path"])
if not root.exists():
    # AC-F11-01a: machine-local evidence absent on this host. The only
    # thing provable here is that verify_tree_manifest correctly reports
    # not_present_on_host rather than silently mismatching or matching.
    result = module.verify_tree_manifest("2026-09-04-e40-full-family")
    if result["status"] != "not_present_on_host":
        raise SystemExit(
            f"registered root is absent on this host but verify_tree_manifest reported "
            f"{result['status']!r}, expected 'not_present_on_host'"
        )
else:
    # Root present on this host (e.g. the machine that captured the
    # evidence): additionally cross-check the live preflight-result.json
    # digest against feature.md's recorded value, verbatim.
    preflight_result = root / "preflight" / "baseline" / "preflight-result.json"
    if not preflight_result.is_file():
        raise SystemExit(f"retained preflight result not found: {preflight_result}")
    digest = module.file_digest(preflight_result)
    if digest != expected_digest:
        raise SystemExit(
            f"retained preflight-result.json digest {digest} does not match feature.md's {expected_digest}"
        )
PY

echo "TC-02: evidence file cross-checks feature.md's digest, classification, and registry_id -- PASS"

echo "TC-099: retention-root immutability guard and evidence registration -- ALL PASS"
