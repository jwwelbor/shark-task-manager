#!/usr/bin/env bash
# TC-100 / T-E40-F11-002 (spec.md REQ-F-002, AC-F11-03/04/05; test-plan.md
# TC-03/04/05): evaluator disclosure and test-identity collection are bound
# to each package's admitted fixture.base_sha via an immutable checkout,
# never the repository submodule's incidental working-tree HEAD.
#
# Caller-Path Contract: real filesystem, real git, real fixture submodule
# (bench/fixture-py), real committed packages (py-change-priority-scale,
# py-techdebt-consolidate-validation -- the exact two packages the
# 2026-09-04 failure named). Nothing is stubbed except the one PATH-shimmed
# `python3` used to OBSERVE whether the test-identity collector process
# (id-collectors/python-collect-ids.sh) actually started -- the same
# "PATH-stubbed binary + invocation log" pattern tc043 case (d) uses for
# the provider dispatcher, generalized to this collector.
#
# AC-F11-03: checkout HEAD verified against fixture.base_sha BEFORE any
#   collector process starts; a mismatch aborts naming both SHAs, and the
#   collector's own invocation count is 0 (observed via the log, not
#   inferred from the guard's exit code alone).
# AC-F11-04: regardless of the LIVE submodule's own incidental HEAD, the
#   admitted checkout is always pinned at the admitted base_sha, so the two
#   real 2026-09-04 `ModuleNotFoundError` failures
#   (taskmanager.legacy_records, taskmanager.bulk_import) collect as
#   passes. The reverse is also proven directly against the collector: run
#   at the WRONG sha, the SAME two real oracle files reproduce the SAME two
#   real ModuleNotFoundErrors -- this is not a synthetic defect, it is the
#   actual gap the checkout binding closes.
# AC-F11-05: the live repository submodule is never touched (clean
#   `git status --porcelain`, byte-unchanged `git submodule status`), and
#   `fixture_checkout` is present in lib/e40_benchmark.py's
#   selected_matrix() output -- the exact field preflight-result.json's
#   scenario_matrix[] carries verbatim (e40_benchmark.py's
#   _cmd_preflight_locked writes `"scenario_matrix": matrix` unmodified, so
#   this is the production entrypoint's own field, not a parallel copy).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-scenario-fixture.sh"
GUARD="$SCRIPTS_DIR/verify-evidence-roots.sh"
COLLECTOR="$SCRIPTS_DIR/id-collectors/python-collect-ids.sh"
MODULE="$SCRIPTS_DIR/lib/e40_benchmark.py"
FIXTURE_SUBMODULE="$BENCH_DIR/fixture-py"

PKG_A_DIR="$BENCH_DIR/scenarios/packages/py-change-priority-scale"
PKG_A_ORACLE="evaluator/test_priority_scale_acceptance.py"
PKG_A_MODULE="taskmanager.legacy_records"

PKG_B_DIR="$BENCH_DIR/scenarios/packages/py-techdebt-consolidate-validation"
PKG_B_ORACLE="evaluator/test_consolidate_validation.py"
PKG_B_MODULE="taskmanager.bulk_import"

fail() {
	echo "TC-100 FAIL: $1" >&2
	exit 1
}

[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-scenario-fixture.sh missing or not executable"
[[ -x "$GUARD" ]] || fail "verify-evidence-roots.sh missing or not executable"
[[ -x "$COLLECTOR" ]] || fail "id-collectors/python-collect-ids.sh missing or not executable"
[[ -f "$MODULE" ]] || fail "lib/e40_benchmark.py missing"
[[ -e "$FIXTURE_SUBMODULE/.git" ]] || fail "bench/fixture-py submodule not initialized; run 'git submodule update --init'"
[[ -f "$PKG_A_DIR/package.yaml" ]] || fail "py-change-priority-scale/package.yaml missing"
[[ -f "$PKG_B_DIR/package.yaml" ]] || fail "py-techdebt-consolidate-validation/package.yaml missing"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

run_module() {
	# run_module <python-snippet-file> [args...] -- loads lib/e40_benchmark.py
	# as a module (the real, unmocked implementation) and runs the snippet.
	local snippet="$1"
	shift
	python3 - "$MODULE" "$snippet" "$@" <<'PY'
import importlib.util
import sys

module_path, snippet_path = sys.argv[1:3]
extra_args = sys.argv[3:]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
sys.argv = [snippet_path, *extra_args]
with open(snippet_path) as f:
    code = f.read()
exec(compile(code, snippet_path, "exec"), {"module": module, "sys": sys})
PY
}

read_pkg_field() {
	# read_pkg_field <package_yaml> <dotted.path>
	python3 - "$1" "$2" <<'PYEOF'
import sys
import yaml

path, dotted = sys.argv[1:3]
with open(path) as f:
    data = yaml.safe_load(f)
node = data
for part in dotted.split("."):
    node = node[part]
print(node)
PYEOF
}

BASE_SHA="$(read_pkg_field "$PKG_A_DIR/package.yaml" fixture.base_sha)"
FIXTURE_ID="$(read_pkg_field "$PKG_A_DIR/package.yaml" fixture.fixture_id)"
[[ -n "$BASE_SHA" && -n "$FIXTURE_ID" ]] || fail "could not derive fixture.fixture_id/base_sha from the real package.yaml"
B_BASE_SHA="$(read_pkg_field "$PKG_B_DIR/package.yaml" fixture.base_sha)"
[[ "$B_BASE_SHA" == "$BASE_SHA" ]] || fail "test assumes both packages admit the same base_sha (per spec.md, all committed packages do)"

# The live submodule's own incidental HEAD -- deliberately NOT assumed to
# equal BASE_SHA (AC-F11-04's whole point is that it does not have to).
SUBMODULE_HEAD="$(git -C "$FIXTURE_SUBMODULE" rev-parse HEAD)"
[[ -n "$SUBMODULE_HEAD" ]] || fail "could not resolve bench/fixture-py's current HEAD"

echo "TC-100: fixture_id=$FIXTURE_ID admitted base_sha=$BASE_SHA live submodule HEAD=$SUBMODULE_HEAD"

# A commit genuinely different from BASE_SHA, for constructing deliberately
# mismatched checkouts below (and for TC-04's "wrong SHA" reproduction,
# which needs the real 2026-09-04 ModuleNotFoundError state specifically).
# SUBMODULE_HEAD -- the live submodule's own incidental checked-out
# commit -- is NOT used here: an earlier test in a full-suite run (or an
# operator following this suite's own pin-before-running convention) can
# leave it pinned at exactly BASE_SHA, which would make a checkout of
# SUBMODULE_HEAD not actually differ at all, silently defeating the
# negative tests below, or (for TC-04) fail to reproduce the real bug.
# checkout-scenario-fixture.sh clones fresh from the submodule's own repo
# and checks out whatever SHA is requested, so it never depends on what's
# live-checked-out -- use the git-registered submodule pointer (this
# checkout's own committed bench/fixture-py entry, confirmed to reproduce
# the TC-04 ModuleNotFoundError), which is independent of ambient state.
DIFFERENT_FROM_BASE_SHA="$(git -C "$REPO_ROOT" rev-parse "HEAD:bench/fixture-py")"
[[ "$DIFFERENT_FROM_BASE_SHA" != "$BASE_SHA" ]] \
	|| fail "the registered bench/fixture-py submodule pointer equals BASE_SHA; cannot construct a genuinely different checkout"

# Baseline submodule cleanliness snapshot, taken BEFORE any of this test's
# own git operations, and re-checked at the very end (AC-F11-05).
submodule_status_before="$(git -C "$REPO_ROOT" submodule status -- bench/fixture-py)"
fixture_porcelain_before="$(git -C "$FIXTURE_SUBMODULE" status --porcelain)"

# ===========================================================================
# TC-03 / AC-F11-03: checkout-binding guard -- positive (match) and negative
# (mismatch, before any collector runs).
# ===========================================================================

# admitted_fixture_checkout() is the production entrypoint (REQ-F-002); it
# clones bench/fixture-py -- never touching it in place -- into a caller-
# supplied cache_root, verifies HEAD, and marks the tree read-only.
CACHE_ROOT="$WORKDIR/cache"
cat >"$WORKDIR/checkout_positive.py" <<'PY'
import sys
fixture_id, base_sha, cache_root = sys.argv[1:4]
from pathlib import Path
checkout = module.admitted_fixture_checkout(fixture_id, base_sha, Path(cache_root))
print(str(checkout))
PY
ADMITTED_CHECKOUT="$(run_module "$WORKDIR/checkout_positive.py" "$FIXTURE_ID" "$BASE_SHA" "$CACHE_ROOT")"
[[ -d "$ADMITTED_CHECKOUT" ]] || fail "admitted_fixture_checkout did not return an existing directory"
[[ "$(git -C "$ADMITTED_CHECKOUT" rev-parse HEAD)" == "$BASE_SHA" ]] \
	|| fail "admitted checkout HEAD does not equal the requested base_sha"
echo "TC-100: admitted_fixture_checkout() produced a checkout pinned at base_sha -- PASS"

# Read-only enforcement: an in-place edit of a tracked file must fail.
if ( echo "tamper" >>"$ADMITTED_CHECKOUT/taskmanager/manager.py" ) 2>/dev/null; then
	fail "admitted checkout file was writable; read-only marking did not apply"
fi
echo "TC-100: admitted checkout files are read-only -- PASS"

# rm -rf must still work (write bits are cleared on files only, never on
# directories -- unlinking a file needs write permission on its PARENT
# directory, not the file itself). If this regresses, every existing test
# that runs preflight/pilot/baseline against the real scenario index and
# then rm -rf's its own tmp WORKDIR would start failing on cleanup.
rm -rf "$ADMITTED_CHECKOUT"
[[ ! -e "$ADMITTED_CHECKOUT" ]] || fail "read-only checkout could not be removed by a plain rm -rf"
echo "TC-100: read-only checkout is still removable via plain rm -rf (directories stay writable) -- PASS"

# Re-derive it (the copy above was deleted) for the rest of the test, and
# separately prove the reuse path: a second request for the SAME
# (fixture_id, base_sha) must succeed WITHOUT re-invoking
# checkout-scenario-fixture.sh, which itself refuses a pre-existing
# dest_dir -- so a wrongly-reinvoked script would surface as a hard
# failure here, not silently.
ADMITTED_CHECKOUT="$(run_module "$WORKDIR/checkout_positive.py" "$FIXTURE_ID" "$BASE_SHA" "$CACHE_ROOT")"
REUSED_CHECKOUT="$(run_module "$WORKDIR/checkout_positive.py" "$FIXTURE_ID" "$BASE_SHA" "$CACHE_ROOT")"
[[ "$ADMITTED_CHECKOUT" == "$REUSED_CHECKOUT" ]] || fail "second admitted_fixture_checkout call returned a different path"
[[ -d "$REUSED_CHECKOUT" ]] || fail "reused checkout path does not exist"
echo "TC-100: admitted_fixture_checkout() reuses the existing checkout for a repeated (fixture_id, base_sha) -- PASS"

# A tampered cache entry (HEAD moved away from the path's own base_sha,
# simulating on-disk corruption between two calls in the same operator
# run) must be caught on reuse, not silently accepted.
TAMPERED_ID="tampered-py"
"$CHECKOUT_SCRIPT" "$FIXTURE_ID" "$DIFFERENT_FROM_BASE_SHA" "$CACHE_ROOT/$TAMPERED_ID/$BASE_SHA" >/dev/null 2>&1 || true
if [[ ! -d "$CACHE_ROOT/$TAMPERED_ID/$BASE_SHA" ]]; then
	mkdir -p "$CACHE_ROOT/$TAMPERED_ID"
	git clone --quiet -- "$FIXTURE_SUBMODULE" "$CACHE_ROOT/$TAMPERED_ID/$BASE_SHA" >/dev/null
	git -C "$CACHE_ROOT/$TAMPERED_ID/$BASE_SHA" -c advice.detachedHead=false checkout --quiet "$DIFFERENT_FROM_BASE_SHA" --
fi
cat >"$WORKDIR/checkout_tampered.py" <<'PY'
import sys
fixture_id, base_sha, cache_root = sys.argv[1:4]
from pathlib import Path
try:
    module.admitted_fixture_checkout(fixture_id, base_sha, Path(cache_root))
except module.OperatorError as exc:
    print(f"REJECTED: {exc}")
    sys.exit(0)
print("ACCEPTED (should not happen)")
sys.exit(1)
PY
tampered_out="$(run_module "$WORKDIR/checkout_tampered.py" "$TAMPERED_ID" "$BASE_SHA" "$CACHE_ROOT")" \
	|| fail "tampered cache entry was silently accepted on reuse"
echo "$tampered_out" | grep -q "REJECTED" || fail "tampered cache entry rejection not observed: $tampered_out"
echo "$tampered_out" | grep -q "$BASE_SHA" || fail "tampered-entry rejection did not name the expected base_sha: $tampered_out"
echo "$tampered_out" | grep -q "$DIFFERENT_FROM_BASE_SHA" || fail "tampered-entry rejection did not name the actual HEAD: $tampered_out"
echo "TC-100: admitted_fixture_checkout() rejects a tampered cache entry on reuse, naming both SHAs -- PASS"

# ---------------------------------------------------------------------------
# checkout-scenario-fixture.sh's own post-checkout assertion (spec.md
# §2.1.3): proven at the positive/success path above (every checkout this
# test builds passes through this script and its HEAD is asserted against
# the requested base_sha every time). A direct failure-injection test is
# not attempted: `git checkout <full-40-hex-sha>` either lands HEAD at
# exactly that commit or the checkout command itself fails first -- there
# is no code path in which git succeeds yet lands elsewhere for this
# script's own callers to reach. The assertion is defense-in-depth,
# matching the brownfield note that verify-evidence-roots.sh (below) is
# the guard actually exercised for the negative case.
# ---------------------------------------------------------------------------

# ===========================================================================
# TC-03 negative: verify-evidence-roots.sh aborts BEFORE any collector
# process starts when fixture_checkout's HEAD does not match the package's
# admitted fixture.base_sha -- proven via a real oracle_tests[] entry so a
# guard that only checked SHAs syntactically (without actually blocking
# collection) would still be caught by the invocation-count assertion.
# ===========================================================================
SCRATCH_PROJECT="$WORKDIR/scratch-project"
mkdir -p "$SCRATCH_PROJECT"
echo "unrelated scratch-project marker" >"$SCRATCH_PROJECT/README.md"

# A PATH-shimmed python3 that logs every invocation carrying "-m pytest" --
# python-collect-ids.sh's own unconditional `python3 -m pytest --version`
# toolchain probe is its FIRST action, before any collection heredoc runs,
# so this is a reliable "the collector script actually started" signal.
# verify-evidence-roots.sh's own internal python3 heredocs never invoke
# pytest, so this marker is unambiguous.
REAL_PYTHON3="$(command -v python3)"
mkdir -p "$WORKDIR/bin"
COLLECTOR_LOG="$WORKDIR/collector-invocations.log"
cat >"$WORKDIR/bin/python3" <<EOF
#!/usr/bin/env bash
case " \$* " in
*" -m pytest "*) echo "\$*" >>"$COLLECTOR_LOG" ;;
esac
exec "$REAL_PYTHON3" "\$@"
EOF
chmod +x "$WORKDIR/bin/python3"

MISMATCH_CHECKOUT="$WORKDIR/mismatch-checkout"
cp -r "$ADMITTED_CHECKOUT" "$MISMATCH_CHECKOUT"
chmod -R u+w "$MISMATCH_CHECKOUT"
git -C "$MISMATCH_CHECKOUT" -c advice.detachedHead=false checkout --quiet "$DIFFERENT_FROM_BASE_SHA" --
mismatch_head="$(git -C "$MISMATCH_CHECKOUT" rev-parse HEAD)"
[[ "$mismatch_head" == "$DIFFERENT_FROM_BASE_SHA" && "$mismatch_head" != "$BASE_SHA" ]] \
	|| fail "failed to construct a checkout whose HEAD genuinely differs from the admitted base_sha"

rm -f "$COLLECTOR_LOG"
neg_out="$WORKDIR/negative.out"
neg_rc=0
PATH="$WORKDIR/bin:$PATH" "$GUARD" "$PKG_A_DIR/package.yaml" "$MISMATCH_CHECKOUT" "$SCRATCH_PROJECT" "$PKG_A_DIR" >"$neg_out" 2>&1 || neg_rc=$?
[[ "$neg_rc" -eq 2 ]] || fail "mismatched checkout: expected exit 2 (script/precondition error), got $neg_rc: $(cat "$neg_out")"
grep -q "$BASE_SHA" "$neg_out" || fail "mismatch abort did not name the expected base_sha: $(cat "$neg_out")"
grep -q "$DIFFERENT_FROM_BASE_SHA" "$neg_out" || fail "mismatch abort did not name the actual checkout HEAD: $(cat "$neg_out")"
[[ ! -s "$COLLECTOR_LOG" ]] || fail "collector was invoked $(wc -l <"$COLLECTOR_LOG") time(s) on the mismatch path, want zero (observed, not inferred): $(cat "$COLLECTOR_LOG")"
echo "TC-100(TC-03 negative): mismatched checkout aborts naming both SHAs, zero collector invocations -- PASS"

# Positive control for the same PATH-shimmed collector log: the match path
# above (ADMITTED_CHECKOUT) must actually invoke the collector, so the zero
# reading above is meaningful evidence and not an artifact of the guard
# never reaching collection for an unrelated reason.
rm -f "$COLLECTOR_LOG"
pos_out="$WORKDIR/positive.out"
PATH="$WORKDIR/bin:$PATH" "$GUARD" "$PKG_A_DIR/package.yaml" "$ADMITTED_CHECKOUT" "$SCRATCH_PROJECT" "$PKG_A_DIR" >"$pos_out" 2>&1
grep -q "^CLEAN$" "$pos_out" || fail "matched checkout did not report CLEAN: $(cat "$pos_out")"
[[ -s "$COLLECTOR_LOG" ]] || fail "collector was never invoked on the match path (positive control)"
echo "TC-100(TC-03 positive control): matched checkout reports CLEAN with the collector genuinely invoked -- PASS"

# ===========================================================================
# TC-04 / AC-F11-04: the exact two real 2026-09-04 ModuleNotFoundError
# failures, reproduced as passes at the admitted checkout and as failures
# at the wrong SHA -- proving the fix is the checkout binding, not that the
# underlying import gap stopped existing.
# ===========================================================================
WRONG_CHECKOUT="$WORKDIR/wrong-sha-checkout"
"$CHECKOUT_SCRIPT" "$FIXTURE_ID" "$DIFFERENT_FROM_BASE_SHA" "$WRONG_CHECKOUT" >/dev/null

check_collects_clean() {
	local label="$1" checkout="$2" pkg_dir="$3" oracle_rel="$4"
	local out="$WORKDIR/collect-$label.out" err="$WORKDIR/collect-$label.err"
	"$COLLECTOR" --checkout "$checkout" --file "$pkg_dir/$oracle_rel" >"$out" 2>"$err" \
		|| fail "$label: collector failed at the admitted checkout: $(cat "$err")"
	grep -q '"ids"' "$out" || fail "$label: collector produced no ids: $(cat "$out")"
	grep -qi "ModuleNotFoundError" "$err" && fail "$label: unexpected ModuleNotFoundError at the admitted checkout: $(cat "$err")"
	return 0
}

check_collects_module_not_found() {
	local label="$1" checkout="$2" oracle_abs="$3" module_name="$4"
	local err="$WORKDIR/collect-$label.err"
	local rc=0
	"$COLLECTOR" --checkout "$checkout" --file "$oracle_abs" >/dev/null 2>"$err" || rc=$?
	[[ "$rc" -ne 0 ]] || fail "$label: expected collection failure at the wrong SHA, got exit 0"
	grep -q "ModuleNotFoundError: No module named '$module_name'" "$err" \
		|| fail "$label: expected ModuleNotFoundError for $module_name, got: $(cat "$err")"
}

check_collects_clean "legacy-records" "$ADMITTED_CHECKOUT" "$PKG_A_DIR" "$PKG_A_ORACLE"
check_collects_clean "bulk-import" "$ADMITTED_CHECKOUT" "$PKG_B_DIR" "$PKG_B_ORACLE"
echo "TC-100(AC-F11-04, admitted base_sha): both taskmanager.legacy_records and taskmanager.bulk_import collect successfully -- PASS"

check_collects_module_not_found "legacy-records-wrong-sha" "$WRONG_CHECKOUT" "$PKG_A_DIR/$PKG_A_ORACLE" "$PKG_A_MODULE"
check_collects_module_not_found "bulk-import-wrong-sha" "$WRONG_CHECKOUT" "$PKG_B_DIR/$PKG_B_ORACLE" "$PKG_B_MODULE"
echo "TC-100(AC-F11-04, wrong submodule HEAD): both real 2026-09-04 ModuleNotFoundError failures reproduce -- PASS"

# Same proof through the production caller-path (verify-evidence-roots.sh,
# which preflight invokes per scenario), not just the raw collector binary.
PATH="$WORKDIR/bin:$PATH" "$GUARD" "$PKG_B_DIR/package.yaml" "$ADMITTED_CHECKOUT" "$SCRATCH_PROJECT" "$PKG_B_DIR" >"$WORKDIR/guard-b.out" 2>&1 \
	|| fail "verify-evidence-roots.sh failed for py-techdebt-consolidate-validation at the admitted checkout: $(cat "$WORKDIR/guard-b.out")"
grep -q "^CLEAN$" "$WORKDIR/guard-b.out" || fail "py-techdebt-consolidate-validation did not report CLEAN: $(cat "$WORKDIR/guard-b.out")"
echo "TC-100(AC-F11-04, production caller-path): verify-evidence-roots.sh collects py-techdebt-consolidate-validation cleanly at the admitted checkout -- PASS"

# ===========================================================================
# TC-05 / AC-F11-05: fixture_checkout surfaces in selected_matrix() (the
# exact field preflight-result.json.scenario_matrix[] carries verbatim),
# and the live submodule is never touched by any of the above.
# ===========================================================================
cat >"$WORKDIR/matrix_field.py" <<'PY'
import sys
from pathlib import Path

cache_root, scenario_id = sys.argv[1:3]
config = {"scenario_index": None}
matrix = module.selected_matrix(config, Path("."), [scenario_id], 1, Path(cache_root))
assert len(matrix) == 1, f"expected exactly one selected row, got {len(matrix)}"
row = matrix[0]
assert row["fixture_checkout"], f"fixture_checkout field missing/empty: {row}"
print(row["fixture_checkout"])

# Without a cache_root, the field is present but null -- never silently
# omitted, and never a stale/incidental path.
matrix_no_cache = module.selected_matrix(config, Path("."), [scenario_id], 1)
row_no_cache = matrix_no_cache[0]
assert "fixture_checkout" in row_no_cache, "fixture_checkout key missing entirely when cache_root is omitted"
assert row_no_cache["fixture_checkout"] is None, f"expected None without cache_root, got {row_no_cache['fixture_checkout']!r}"
PY
MATRIX_CACHE="$WORKDIR/matrix-cache"
matrix_checkout_path="$(run_module "$WORKDIR/matrix_field.py" "$MATRIX_CACHE" "py-change-priority-scale")"
[[ "$matrix_checkout_path" == "$MATRIX_CACHE/$FIXTURE_ID/$BASE_SHA" ]] \
	|| fail "selected_matrix()'s fixture_checkout did not resolve to <cache_root>/<fixture_id>/<base_sha>: $matrix_checkout_path"
[[ -d "$matrix_checkout_path" ]] || fail "selected_matrix()'s fixture_checkout path does not exist on disk"
[[ "$(git -C "$matrix_checkout_path" rev-parse HEAD)" == "$BASE_SHA" ]] \
	|| fail "selected_matrix()'s fixture_checkout is not pinned at the admitted base_sha"
echo "TC-100(AC-F11-05): selected_matrix() carries fixture_checkout -- <cache_root>/<fixture_id>/<base_sha>, pinned; null when cache_root is omitted -- PASS"

submodule_status_after="$(git -C "$REPO_ROOT" submodule status -- bench/fixture-py)"
fixture_porcelain_after="$(git -C "$FIXTURE_SUBMODULE" status --porcelain)"
[[ "$fixture_porcelain_before" == "$fixture_porcelain_after" && -z "$fixture_porcelain_after" ]] \
	|| fail "bench/fixture-py has a dirty working tree after this test's checkouts: before=[$fixture_porcelain_before] after=[$fixture_porcelain_after]"
[[ "$submodule_status_before" == "$submodule_status_after" ]] \
	|| fail "git submodule status bench/fixture-py changed: before=[$submodule_status_before] after=[$submodule_status_after]"
echo "TC-100(AC-F11-05): bench/fixture-py submodule byte-unchanged (clean status, unchanged submodule pointer) after every checkout above -- PASS"

echo "TC-100: admitted-fixture checkout binding for evaluator collection -- ALL PASS"
