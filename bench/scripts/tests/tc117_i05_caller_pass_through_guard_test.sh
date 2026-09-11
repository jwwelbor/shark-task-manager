#!/usr/bin/env bash
# TC-117 / T-E40-F12-006: --i05-bundle-dir pass-through + guard-reorder
# caller-path coverage (TC-019/TC-020), and the T-E40-F12-006 kickback's own
# REWORK guard cases (smoke-lifecycle.sh pass-through proof + the
# defect-class structural static-scan guard over every bench/scripts/*.sh
# invocation site).
#
# Round-2 kickback (defect-class "test-placed-after-independently-failing-
# case-in-a-fail-fast-script"): TC-019/TC-020 and the two REWORK guard cases
# below all originally lived inside tc115_i05_producer_test.sh, physically
# positioned after TC-008 (owned by T-E40-F12-003). tc115 uses
# `set -euo pipefail` with a single `fail() { ...; exit 1; }` shared across
# ~28 TC-NNN cases in one process, so TC-008's current, separately-tracked,
# deterministic failure aborts the whole script before ever reaching any of
# these four cases -- they have never once executed in the checked-in suite.
# run-all.sh already runs each tcNNN_*_test.sh as an independent subprocess
# (its own PASS/FAIL line), so extracting these cases into their own file
# gives them real isolation from tc115's internal ordering instead of
# reordering them within the same fail-fast script (which would only buy
# immunity from TC-008, not from tc115's own earlier, more fragile cases
# such as TC-007's live-binary state-addition regression).
#
# This file intentionally duplicates only the minimal shared preamble
# (path resolution, `fail()`, the python3 precondition) tc115 also carries --
# see tc115_i05_producer_test.sh's own header for the full TC-001..TC-028
# case index this file does not duplicate.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
SCENARIO="$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml"
fail() { echo "TC-117 FAIL: $1" >&2; exit 1; }
[[ -f "$SCENARIO" ]] || fail "scenario package missing: $SCENARIO"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

# ---------------------------------------------------------------------------
# TC-019/TC-020: run-lifecycle-batch.sh dispatch_pair() and
# run-review-comparison.sh dispatch_gate() pass-through + guard reordering
# (T-E40-F12-006, REQ-F-013, ADR-F12-07, AC-018).
#
# Caller-Path Contract (test-plan.md TC-019/TC-020 rows): the REAL
# run-lifecycle-batch.sh / run-review-comparison.sh binaries, driven via the
# RUN_LIFECYCLE_BIN/EVALUATE_LIFECYCLE_BIN sibling-path stub convention
# tc082's (a2) driver-path section already establishes. RUN_LIFECYCLE_BIN
# here is a lightweight stub (not the real run-lifecycle.sh) that only
# records its own argv and writes a committed valid I-07 fixture to
# --output -- proving the pass-through/guard-ordering property does not
# require re-deriving a full live dispatch (that property, plus a real
# dispatch through the real run-lifecycle.sh, is separately proven for the
# batch driver by tc080's UAT-R2-01 regression section). EVALUATE_LIFECYCLE_BIN
# always fails deterministically, so both (a) and (b) below terminate at a
# clean, distinguishing classification without needing retain_pair/
# compare-lifecycle-evaluations.sh success.
# ---------------------------------------------------------------------------
BATCH="$SCRIPTS_DIR/run-lifecycle-batch.sh"
COMPARISON="$SCRIPTS_DIR/run-review-comparison.sh"
I07_FIXTURE="$REPO_ROOT/tests/contracts/testdata/e40_i07/valid/complete.jsonl"
[[ -x "$BATCH" ]] || fail "run-lifecycle-batch.sh missing or not executable"
[[ -x "$COMPARISON" ]] || fail "run-review-comparison.sh missing or not executable"
[[ -f "$I07_FIXTURE" ]] || fail "committed valid I-07 fixture missing: $I07_FIXTURE"

WORKDIR_19="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_19"' EXIT

TC19_SCENARIO_ID="tc115-tc019-scenario"
mkdir -p "$WORKDIR_19/index/packages/$TC19_SCENARIO_ID" "$WORKDIR_19/scratch"
cat >"$WORKDIR_19/index/packages/$TC19_SCENARIO_ID/package.yaml" <<EOF
schema_version: "1.0"
scenario_id: "$TC19_SCENARIO_ID"
scenario_version: "1"
entity_family: "family-tc019"
fixture:
  fixture_id: "fixture-tc019-scenario"
  base_sha: "fixture-base-tc019-scenario"
EOF
cat >"$WORKDIR_19/index/scenarios.yaml" <<EOF
schema_version: "1.0"
scenarios:
  - packages/$TC19_SCENARIO_ID
EOF

# dispatch_pair() (run-lifecycle-batch.sh) checks out the declared fixture
# via CHECKOUT_SCENARIO_FIXTURE_BIN before ever invoking RUN_LIFECYCLE_BIN --
# stubbed the same way tc082_retention_layout_test.sh's DRIVER_CHECKOUT_STUB
# stubs it, so TC-019's real dispatch_pair() invocation reaches its
# RUN_LIFECYCLE_BIN stub instead of failing during fixture checkout.
TC19_CHECKOUT_STUB="$WORKDIR_19/checkout-scenario-fixture-stub.sh"
cat >"$TC19_CHECKOUT_STUB" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$3"
printf 'gitdir: fixture-only-test-double\n' >"$3/.git"
EOF
chmod +x "$TC19_CHECKOUT_STUB"

# Records its own argv (one line per invocation) and writes a committed,
# schema-valid I-07 fixture to --output, so a caller downstream of a
# successful invocation (dispatch_pair's own run_rc==0 check) sees a real
# record -- never invoked at all in the misconfigured (b) case below.
TC19_RUN_LOG="$WORKDIR_19/run-lifecycle-argv.log"
: >"$TC19_RUN_LOG"
TC19_RUN_STUB="$WORKDIR_19/run-lifecycle-stub.sh"
cat >"$TC19_RUN_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "\$*" >>"$TC19_RUN_LOG"
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
cp "$I07_FIXTURE" "\$output"
EOF
chmod +x "$TC19_RUN_STUB"

# Always fails deterministically -- dispatch_pair/dispatch_gate must reach
# this call (proving the real dispatch happened) and then classify the
# pair/gate invalid via the SAME evaluation_failed path REQ-F-013 leaves
# untouched, never mocking evaluate-lifecycle.sh's own verdict logic.
TC19_EVAL_STUB="$WORKDIR_19/evaluate-lifecycle-stub.sh"
cat >"$TC19_EVAL_STUB" <<'EOF'
#!/usr/bin/env bash
echo "tc115 TC-019/TC-020 fixture: deliberate evaluation failure" >&2
exit 1
EOF
chmod +x "$TC19_EVAL_STUB"

# ===========================================================================
# TC-019(a): dispatch_pair(), i05_bundle_dir CONFIGURED -- the real
# RUN_LIFECYCLE_BIN invocation must include --i05-bundle-dir, proven by
# recording argv and grepping for the flag (not merely the exit code).
# ===========================================================================
TC19A_I05="$WORKDIR_19/i05-a"
TC19A_POLICY="$WORKDIR_19/policy-a.yaml"
cat >"$TC19A_POLICY" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$WORKDIR_19/index/scenarios.yaml"
scenarios:
  $TC19_SCENARIO_ID:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR_19/scratch"
    i05_bundle_dir: "$TC19A_I05"
    reps: 1
EOF

: >"$TC19_RUN_LOG"
tc19a_rc=0
RUN_LIFECYCLE_BIN="$TC19_RUN_STUB" EVALUATE_LIFECYCLE_BIN="$TC19_EVAL_STUB" \
	CHECKOUT_SCENARIO_FIXTURE_BIN="$TC19_CHECKOUT_STUB" \
	"$BATCH" --batch "$TC19A_POLICY" --retention-root "$WORKDIR_19/retention-a" \
	--mode pilot --acknowledge-provider-spend \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 \
	>"$WORKDIR_19/tc19a.out" 2>"$WORKDIR_19/tc19a.err" || tc19a_rc=$?
[[ "$tc19a_rc" -eq 4 ]] || fail "TC-019(a): expected exit 4 (real dispatch reached, evaluation deliberately failed), got $tc19a_rc: $(cat "$WORKDIR_19/tc19a.out") $(cat "$WORKDIR_19/tc19a.err")"
[[ -s "$TC19_RUN_LOG" ]] || fail "TC-019(a): RUN_LIFECYCLE_BIN was never invoked despite a configured i05_bundle_dir"
grep -q -- "--i05-bundle-dir $TC19A_I05" "$TC19_RUN_LOG" \
	|| fail "TC-019(a): dispatch_pair's real invocation did not include --i05-bundle-dir $TC19A_I05: $(cat "$TC19_RUN_LOG")"
grep -q "evaluation_failed" "$WORKDIR_19/retention-a/invalid/index.jsonl" \
	|| fail "TC-019(a): expected the pair classified evaluation_failed (proving the real dispatch succeeded before eval ran), got: $(cat "$WORKDIR_19/retention-a/invalid/index.jsonl" 2>/dev/null)"

echo "TC-115 TC-019(a): pass (dispatch_pair passes --i05-bundle-dir through to the real run-lifecycle.sh invocation)"

# ===========================================================================
# TC-019(b): dispatch_pair(), i05_bundle_dir MISCONFIGURED (unset) -- the
# guard-reorder half of ADR-F12-07: the pair is recorded invalid BEFORE
# RUN_LIFECYCLE_BIN is invoked at all (no provider spend burned on a doomed
# run), proven by the argv log staying empty.
# ===========================================================================
TC19B_POLICY="$WORKDIR_19/policy-b.yaml"
cat >"$TC19B_POLICY" <<EOF
schema_version: "1.0"
min_reps: 1
scenario_index: "$WORKDIR_19/index/scenarios.yaml"
scenarios:
  $TC19_SCENARIO_ID:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR_19/scratch"
    reps: 1
EOF

: >"$TC19_RUN_LOG"
tc19b_rc=0
RUN_LIFECYCLE_BIN="$TC19_RUN_STUB" EVALUATE_LIFECYCLE_BIN="$TC19_EVAL_STUB" \
	CHECKOUT_SCENARIO_FIXTURE_BIN="$TC19_CHECKOUT_STUB" \
	"$BATCH" --batch "$TC19B_POLICY" --retention-root "$WORKDIR_19/retention-b" \
	--mode pilot --acknowledge-provider-spend \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 \
	>"$WORKDIR_19/tc19b.out" 2>"$WORKDIR_19/tc19b.err" || tc19b_rc=$?
[[ "$tc19b_rc" -eq 4 ]] || fail "TC-019(b): expected exit 4 (pair recorded invalid pre-dispatch), got $tc19b_rc: $(cat "$WORKDIR_19/tc19b.out") $(cat "$WORKDIR_19/tc19b.err")"
[[ ! -s "$TC19_RUN_LOG" ]] || fail "TC-019(b): RUN_LIFECYCLE_BIN was invoked despite an unconfigured i05_bundle_dir -- the guard must fire BEFORE dispatch: $(cat "$TC19_RUN_LOG")"
grep -q "i05_bundle_not_configured" "$WORKDIR_19/retention-b/invalid/index.jsonl" \
	|| fail "TC-019(b): expected the pair classified i05_bundle_not_configured, got: $(cat "$WORKDIR_19/retention-b/invalid/index.jsonl" 2>/dev/null)"

echo "TC-115 TC-019(b): pass (dispatch_pair's guard reorder refuses an unconfigured pair before any dispatch is attempted)"

# ===========================================================================
# TC-020(a)/(b): run-review-comparison.sh dispatch_gate() -- same two
# assertions, independent call site (spec.md Component-changes table lists
# run-lifecycle-batch.sh and run-review-comparison.sh as two separately
# modified files).
# ===========================================================================
TC20A_I05="$WORKDIR_19/i05-cmp-a"
TC20A_CANDIDATE="$WORKDIR_19/candidate-a.yaml"
cat >"$TC20A_CANDIDATE" <<EOF
schema_version: "1.0"
scenario_id: "$TC19_SCENARIO_ID"
scenario_index: "$WORKDIR_19/index/scenarios.yaml"
gates:
  qa:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR_19/scratch"
    i05_bundle_dir: "$TC20A_I05"
  deep_review:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR_19/scratch"
    i05_bundle_dir: "$TC20A_I05"
EOF

: >"$TC19_RUN_LOG"
tc20a_rc=0
RUN_LIFECYCLE_BIN="$TC19_RUN_STUB" EVALUATE_LIFECYCLE_BIN="$TC19_EVAL_STUB" \
	"$COMPARISON" --candidate "$TC20A_CANDIDATE" --retention-root "$WORKDIR_19/cmp-retention-a" \
	--mode pilot --comparison-mode independent_frozen_candidate --acknowledge-provider-spend \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 \
	>"$WORKDIR_19/tc20a.out" 2>"$WORKDIR_19/tc20a.err" || tc20a_rc=$?
[[ "$tc20a_rc" -eq 4 ]] || fail "TC-020(a): expected exit 4 (real gate dispatch reached, evaluation deliberately failed), got $tc20a_rc: $(cat "$WORKDIR_19/tc20a.out") $(cat "$WORKDIR_19/tc20a.err")"
[[ -s "$TC19_RUN_LOG" ]] || fail "TC-020(a): RUN_LIFECYCLE_BIN was never invoked despite a configured i05_bundle_dir"
grep -q -- "--i05-bundle-dir $TC20A_I05" "$TC19_RUN_LOG" \
	|| fail "TC-020(a): dispatch_gate's real invocation did not include --i05-bundle-dir $TC20A_I05: $(cat "$TC19_RUN_LOG")"
grep -q "evaluate-lifecycle.sh exit" "$WORKDIR_19/tc20a.err" \
	|| fail "TC-020(a): expected the gate to fail at evaluate-lifecycle.sh (proving the real dispatch succeeded first), got: $(cat "$WORKDIR_19/tc20a.err")"

echo "TC-115 TC-020(a): pass (dispatch_gate passes --i05-bundle-dir through to the real run-lifecycle.sh invocation)"

TC20B_CANDIDATE="$WORKDIR_19/candidate-b.yaml"
cat >"$TC20B_CANDIDATE" <<EOF
schema_version: "1.0"
scenario_id: "$TC19_SCENARIO_ID"
scenario_index: "$WORKDIR_19/index/scenarios.yaml"
gates:
  qa:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR_19/scratch"
  deep_review:
    root_key: "ROOT-001"
    scratch_root: "$WORKDIR_19/scratch"
EOF

: >"$TC19_RUN_LOG"
tc20b_rc=0
RUN_LIFECYCLE_BIN="$TC19_RUN_STUB" EVALUATE_LIFECYCLE_BIN="$TC19_EVAL_STUB" \
	"$COMPARISON" --candidate "$TC20B_CANDIDATE" --retention-root "$WORKDIR_19/cmp-retention-b" \
	--mode pilot --comparison-mode independent_frozen_candidate --acknowledge-provider-spend \
	--max-cost-usd 5 --max-wall-clock-seconds 600 --max-generated-tasks 10 \
	>"$WORKDIR_19/tc20b.out" 2>"$WORKDIR_19/tc20b.err" || tc20b_rc=$?
[[ "$tc20b_rc" -eq 4 ]] || fail "TC-020(b): expected exit 4 (gate refused pre-dispatch), got $tc20b_rc: $(cat "$WORKDIR_19/tc20b.out") $(cat "$WORKDIR_19/tc20b.err")"
[[ ! -s "$TC19_RUN_LOG" ]] || fail "TC-020(b): RUN_LIFECYCLE_BIN was invoked despite an unconfigured i05_bundle_dir -- the guard must fire BEFORE dispatch: $(cat "$TC19_RUN_LOG")"
grep -q "i05_bundle_dir not configured; cannot evaluate" "$WORKDIR_19/tc20b.err" \
	|| fail "TC-020(b): expected the standard pre-dispatch i05_bundle_dir refusal message, got: $(cat "$WORKDIR_19/tc20b.err")"

echo "TC-115 TC-020(b): pass (dispatch_gate's guard reorder refuses an unconfigured gate before any dispatch is attempted)"

# ---------------------------------------------------------------------------
# TC-019/TC-020 regression check (AC-018's own wording): a real
# preflight readiness invocation against a scenario whose policy now
# configures i05_bundle_dir no longer reports the P5
# "i05_bundle_dir is not configured" blocker. Drives the real, unmodified
# `runtime_readiness()` (bench/scripts/lib/e40_benchmark.py) -- the same
# pure function `e40-benchmark.sh preflight` itself calls -- rather than
# re-deriving the P5 check.
# ---------------------------------------------------------------------------
mkdir -p "$TC19A_I05"
set +e
python3 - "$SCRIPTS_DIR/lib/e40_benchmark.py" "$WORKDIR_19" "$TC19A_I05" "$TC19_SCENARIO_ID" <<'PY'
import importlib.util
import sys
from pathlib import Path

module_path, config_base, i05_dir, scenario_id = sys.argv[1:5]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
e40_benchmark = importlib.util.module_from_spec(spec)
spec.loader.exec_module(e40_benchmark)

config = {
    "runtime": {"lifecycle_adapter": sys.executable},
    "scenario_roots": {scenario_id: {"i05_bundle_dir": i05_dir}},
}
matrix = [{"scenario_id": scenario_id}]
ready, blockers = e40_benchmark.runtime_readiness(config, Path(config_base), matrix)
p5_blockers = [b for b in blockers if b["requirement"] == "P5"]
assert not p5_blockers, f"P5 blocker still reported for a configured i05_bundle_dir: {p5_blockers}"
assert ready, f"runtime_readiness reported not-ready with no P5 blocker: {blockers}"
PY
tc19_regression_rc=$?
set -e
[[ "$tc19_regression_rc" -eq 0 ]] || fail "TC-019/TC-020 regression check: runtime_readiness() still reports the P5 i05_bundle_dir blocker for a configured scenario"

echo "TC-115 TC-019/TC-020 regression check: pass (preflight's real runtime_readiness() reports no P5 blocker once i05_bundle_dir is configured)"

rm -rf "$WORKDIR_19"
trap - EXIT


# ---------------------------------------------------------------------------
# REWORK T-E40-F12-006 kickback (code-review defect class
# "cli-contract-change-missed-caller"): these two cases are NOT test-plan.md
# TC-NNN rows -- test-plan.md never scoped smoke-lifecycle.sh at all, which
# is exactly how it was missed the first time this task ran. They are a
# defect-class-sweep addition: the review's own sweep found three real
# invokers of run-lifecycle.sh (`grep -rln 'RUN_LIFECYCLE_BIN\|run-lifecycle
# \.sh' bench/scripts/*.sh`, comment-only hits excluded) --
# run-lifecycle-batch.sh and run-review-comparison.sh (already covered by
# TC-019/TC-020 above) and bench/scripts/smoke-lifecycle.sh, which was
# missed. A second, requirement-keyed re-sweep (`--mode live` across
# bench/, Makefile, .github/, plus *.py/*.yaml/Makefile mentions of
# `run-lifecycle`) found exactly one more site carrying the same defect in
# prose: bench/README.md's manual "Go live" step (fixed alongside the
# script; a doc fix, not a test target).
#
# Caller-Path Contract: the REAL bench/scripts/smoke-lifecycle.sh binary,
# invoked with its real CLI (--out/--live/--run-id). Its own e40-benchmark.sh
# setup call, run-lifecycle.sh dispatch, and verify-lifecycle-run.sh check
# are each substituted via the RUN_LIFECYCLE_BIN-style sibling-path stub
# convention this task's own kickback fix added
# (E40_BENCHMARK_BIN/RUN_LIFECYCLE_BIN/VERIFY_LIFECYCLE_RUN_BIN) so the case
# runs in milliseconds with no real Shark binary, provider, or scratch
# project -- but the assertion below is on the REAL recorded argv
# smoke-lifecycle.sh hands RUN_LIFECYCLE_BIN, per the kickback's own
# "a real invocation check, not just an exit-code check" requirement.
# ---------------------------------------------------------------------------
WORKDIR_SMOKE="$(mktemp -d)"
trap 'rm -rf "$WORKDIR_SMOKE"' EXIT

SMOKE_BIN="$SCRIPTS_DIR/smoke-lifecycle.sh"
[[ -x "$SMOKE_BIN" ]] || fail "REWORK smoke-lifecycle.sh missing or not executable"

SMOKE_RUN_LOG="$WORKDIR_SMOKE/run-lifecycle-argv.log"
: >"$SMOKE_RUN_LOG"

SMOKE_SETUP_STUB="$WORKDIR_SMOKE/e40-benchmark-stub.sh"
cat >"$SMOKE_SETUP_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
[[ "\$1" == "setup" ]] || { echo "stub e40-benchmark: unsupported command \$1" >&2; exit 2; }
shift
out=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--out) out="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
mkdir -p "\$out"
cat >"\$out/setup-result.json" <<JSON
{
  "shark_binary": {"path": "/bin/true"},
  "scratch_root": "\$out/scratch",
  "scenario_matrix": [{"scenario_id": "py-bug-due-date-boundary", "package_path": "$SCENARIO"}],
  "root_keys": {"py-bug-due-date-boundary": "ROOT-001"}
}
JSON
EOF
chmod +x "$SMOKE_SETUP_STUB"

# Records its own argv (one line per invocation, matching TC-019/TC-020's
# TC19_RUN_STUB convention) and writes a minimal fake record to --output --
# smoke-lifecycle.sh only reads that file's `outcome` field, and only on a
# non-zero exit, which this stub never produces.
SMOKE_RUN_STUB="$WORKDIR_SMOKE/run-lifecycle-stub.sh"
cat >"$SMOKE_RUN_STUB" <<EOF
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "\$*" >>"$SMOKE_RUN_LOG"
output=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output) output="\$2"; shift 2 ;;
	*) shift ;;
	esac
done
mkdir -p "\$(dirname "\$output")"
printf '{"outcome":{"reason":"stub"}}' >"\$output"
EOF
chmod +x "$SMOKE_RUN_STUB"

# Always passes -- this case is about the argv smoke-lifecycle.sh hands
# RUN_LIFECYCLE_BIN, not about verify-lifecycle-run.sh's own schema check
# (already covered elsewhere against real run-lifecycle.sh output).
SMOKE_VERIFY_STUB="$WORKDIR_SMOKE/verify-lifecycle-run-stub.sh"
cat >"$SMOKE_VERIFY_STUB" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$SMOKE_VERIFY_STUB"

SMOKE_OUT="$WORKDIR_SMOKE/out"
smoke_rc=0
E40_BENCHMARK_BIN="$SMOKE_SETUP_STUB" RUN_LIFECYCLE_BIN="$SMOKE_RUN_STUB" \
	VERIFY_LIFECYCLE_RUN_BIN="$SMOKE_VERIFY_STUB" \
	"$SMOKE_BIN" --out "$SMOKE_OUT" --live --run-id smoke-tc \
	>"$WORKDIR_SMOKE/smoke.out" 2>"$WORKDIR_SMOKE/smoke.err" || smoke_rc=$?
[[ "$smoke_rc" -eq 0 ]] || fail "REWORK smoke-lifecycle.sh --live (stubbed) exited $smoke_rc: $(cat "$WORKDIR_SMOKE/smoke.out") $(cat "$WORKDIR_SMOKE/smoke.err")"

python3 - "$SMOKE_RUN_LOG" "$SMOKE_OUT" <<'PY' || fail "REWORK smoke-lifecycle.sh pass-through assertions failed"
import shlex
import sys

log_path, out_dir = sys.argv[1:3]
with open(log_path, encoding="utf-8") as f:
    lines = [line for line in f.read().splitlines() if line.strip()]
assert len(lines) == 2, f"expected 2 RUN_LIFECYCLE_BIN invocations (dry-run + live), got {len(lines)}: {lines}"


def flag(tokens, name):
    assert name in tokens, f"invocation missing {name}: {tokens}"
    return tokens[tokens.index(name) + 1]


dry_tokens = shlex.split(lines[0])
live_tokens = shlex.split(lines[1])

assert flag(dry_tokens, "--mode") == "dry-run", f"expected first invocation mode dry-run: {dry_tokens}"
assert flag(live_tokens, "--mode") == "live", f"expected second invocation mode live: {live_tokens}"

dry_i05 = flag(dry_tokens, "--i05-bundle-dir")
live_i05 = flag(live_tokens, "--i05-bundle-dir")
assert dry_i05 != live_i05, f"dry-run and live invocations share one --i05-bundle-dir: {dry_i05}"
assert dry_i05 == f"{out_dir}/runs/smoke-tc/i05", f"unexpected dry-run --i05-bundle-dir: {dry_i05}"
assert live_i05 == f"{out_dir}/runs/smoke-tc-live/i05", f"unexpected live --i05-bundle-dir: {live_i05}"
PY

echo "TC-115 REWORK(smoke-lifecycle pass-through): pass (both dry-run and live RUN_LIFECYCLE_BIN invocations carry distinct --i05-bundle-dir values, proven via recorded argv, not exit code)"

rm -rf "$WORKDIR_SMOKE"
trap - EXIT

# ---------------------------------------------------------------------------
# REWORK T-E40-F12-006 kickback, defect-class structural guard: rather than
# re-trusting a name-based grep (the reviewer's own `run-lifecycle\.sh`
# pattern missed smoke-lifecycle.sh because it never mentioned
# RUN_LIFECYCLE_BIN before this kickback), statically enumerate every real
# invocation of run-lifecycle.sh under bench/scripts/*.sh -- both the
# RUN_LIFECYCLE_BIN convention and any lingering raw
# "$SCRIPT_DIR/run-lifecycle.sh"-style path -- and require --i05-bundle-dir
# on every one whose --mode is "live" or omitted (parse_args()'s own
# default is "live", so an absent --mode is exactly as live as an explicit
# one). This is checked against the joined logical statement (backslash
# continuations collapsed), matching how these multi-line invocations are
# actually written. This guard is verified to reproduce this kickback's own
# defect: run against the pre-fix smoke-lifecycle.sh (raw-path invocation,
# --mode live, no --i05-bundle-dir) it flags exactly that line.
# ---------------------------------------------------------------------------
python3 - "$SCRIPTS_DIR" <<'PY' || fail "REWORK defect-class guard: a live-mode run-lifecycle.sh invocation is missing --i05-bundle-dir (see stdout above)"
import re
import sys
from pathlib import Path

scripts_dir = Path(sys.argv[1])
raw_path_re = re.compile(r'^"[^"]*run-lifecycle\.sh"(?:\s|\\|$)')

checked = []
violations = []
for path in sorted(scripts_dir.glob("*.sh")):
    if path.name == "run-lifecycle.sh":
        continue  # the definer, not a caller of itself
    text = path.read_text(encoding="utf-8")
    joined = re.sub(r"\\\n[ \t]*", " ", text)
    for line in joined.splitlines():
        stripped = line.strip()
        is_bin_var = stripped.startswith('"$RUN_LIFECYCLE_BIN"')
        is_raw_path = bool(raw_path_re.match(stripped))
        if not (is_bin_var or is_raw_path):
            continue
        checked.append(f"{path.name}: {stripped[:160]}")
        mode_match = re.search(r'--mode\s+"?([\w-]+)"?', line)
        effective_mode = mode_match.group(1) if mode_match else "live"
        if effective_mode == "live" and "--i05-bundle-dir" not in line:
            violations.append(f"{path.name}: {stripped[:200]}")

assert checked, "no run-lifecycle.sh invocation sites found under bench/scripts/*.sh -- guard is not exercising anything"
print(f"REWORK defect-class guard: checked {len(checked)} run-lifecycle.sh invocation site(s):")
for line in checked:
    print(f"  {line}")
if violations:
    print("VIOLATIONS (live-mode invocation missing --i05-bundle-dir):")
    for v in violations:
        print(f"  {v}")
    raise SystemExit(1)
PY

echo "TC-115 REWORK(defect-class guard): pass (every live-mode run-lifecycle.sh invocation site under bench/scripts/*.sh carries --i05-bundle-dir)"

echo "TC-117: pass"
