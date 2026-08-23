#!/usr/bin/env bash
# TC-090 / T-E40-F10-013: offline determinism and 100 MB retention-fixture
# scale test (spec.md REQ-NF-002, REQ-NF-003, REQ-NF-004; test-plan.md
# TC-090 full body; AC-013/AC-T1/AC-T2/AC-T3).
#
# Proves aggregate-lifecycle.sh and report-lifecycle.sh are byte-identical,
# denial-safe, streaming pure functions:
#   AC-T1 -- two runs against the same retention root produce byte-identical
#            aggregate.json and report markdown (both views).
#   AC-T2 -- zero denied calls across provider, network, DB, and
#            live-working-tree denial surfaces. Provider/network reuse
#            T-E40-F10-004's schema-driven PATH-shim harness
#            (tests/lib/path-shim-denial.sh); DB (shark/sqlite3/turso/...)
#            and live-working-tree-write denial are new here -- the DB leg
#            is a second, independently-named PATH-shim binary list (not
#            schema-driven, since REQ-NF-002's `provider_and_network_
#            binaries` vocabulary is deliberately scoped to provider/network
#            only); the live-tree leg snapshots `git status --porcelain`
#            across the run, matching both scripts' own header contracts
#            ("touches nothing outside the named --retention-root" /
#            "invokes no subprocess") -- neither script contains a single
#            write-mode `open()`/`.write()`/`mkdir`/`shutil.*` call, so a
#            real write would be a regression this test catches.
#   Negative case (REQ-NF-003) -- a distinguishable fake system clock (TZ
#            shift + a hostile `date` shim that would corrupt output if
#            consulted) must not change report output, plus a static scan
#            proving neither script imports a clock API.
#   AC-T3 -- a generated ~100 MB retention-shaped fixture (bulk padding
#            under evidence/transcripts, which aggregate-lifecycle.sh's own
#            header contract documents it never opens) completes within 60
#            seconds and peak RSS (via `/usr/bin/time -v`) does not exceed
#            25 MB, or largest-single-retained-file + 5 MB if a single file
#            exceeds 25 MB -- a concrete, numeric proxy for "streamed, not
#            fully loaded" (test-plan.md TC-090 Expected Output).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/.." && pwd)"
AGGREGATOR="$SCRIPTS_DIR/aggregate-lifecycle.sh"
REPORTER="$SCRIPTS_DIR/report-lifecycle.sh"
GEN="$SCRIPT_DIR/testdata/gen-100mb-retention-fixture.sh"
PATH_SHIM_LIB="$SCRIPT_DIR/lib/path-shim-denial.sh"

fail() {
	echo "TC-090 FAIL: $1" >&2
	exit 1
}

[[ -x "$AGGREGATOR" ]] || fail "bench/scripts/aggregate-lifecycle.sh missing or not executable"
[[ -x "$REPORTER" ]] || fail "bench/scripts/report-lifecycle.sh missing or not executable"
[[ -x "$GEN" ]] || fail "bench/scripts/tests/testdata/gen-100mb-retention-fixture.sh missing or not executable"
[[ -f "$PATH_SHIM_LIB" ]] || fail "tests/lib/path-shim-denial.sh missing"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"
command -v git >/dev/null 2>&1 || fail "git not found on PATH"
command -v /usr/bin/time >/dev/null 2>&1 || fail "/usr/bin/time not found (required for peak-RSS measurement)"
/usr/bin/time -v true >/dev/null 2>/tmp/.tc090-time-v-check.$$ || true
grep -q "Maximum resident set size" /tmp/.tc090-time-v-check.$$ 2>/dev/null || fail "/usr/bin/time does not support -v (GNU time required)"
rm -f /tmp/.tc090-time-v-check.$$

# shellcheck source=lib/path-shim-denial.sh
source "$PATH_SHIM_LIB"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# ===========================================================================
# Shared small-root fixture builder -- same field shapes as test-plan.md
# TC-084's own proven-good fixture builder (already exercised against the
# real aggregate-lifecycle.sh/report-lifecycle.sh, both views). Used for
# AC-T1 and AC-T2; the 100 MB fixture below is reserved for AC-T3 alone.
# ===========================================================================
build_small_root() {
	local dest_root="$1"
	python3 - "$dest_root" <<'PYEOF'
import hashlib
import json
import os
import sys

dest_root = sys.argv[1]
SCENARIO_ID = "scenario-tc090"
REP = "1"
pair_dir = os.path.join(dest_root, "scenarios", SCENARIO_ID, REP)
os.makedirs(pair_dir, exist_ok=True)

with open(os.path.join(pair_dir, "package.yaml"), "w", encoding="utf-8") as f:
    f.write(
        'schema_version: "1.0"\n'
        f'scenario_id: "{SCENARIO_ID}"\n'
        'scenario_version: "1"\n'
        'entity_family: "family-tc090"\n'
    )

INTERVALS = [
    ("provider_active", 10), ("tool_and_test", 5), ("queue_or_claim_wait", 3),
    ("replay_or_human_gate_wait", 2), ("retry_or_backoff", 1), ("unclassified", 4),
]
STAGE_SECONDS = sum(d for _, d in INTERVALS)
STAGE_COST = 10.0
STAGE_CATEGORIES = ["discovery", "specification", "planning", "code", "review", "qa", "uat", "shipping"]


def make_intervals():
    out = []
    cursor = 0
    for category, duration in INTERVALS:
        out.append({"category": category, "start": cursor, "end": cursor + duration})
        cursor += duration
    return out


def make_stage(ordinal, category, rework):
    return {
        "dispatch_ordinal": ordinal, "stage": category, "category": category,
        "snapshot_digest": "a" * 64, "prompt_digest": "b" * 64,
        "input_lineage": [], "replay_lineage": [], "output_paths": [], "output_digests": [],
        "usage": {"provider": "fixture", "model": "fixture-model"},
        "cost_usd": STAGE_COST, "elapsed_seconds": float(STAGE_SECONDS),
        "errors": [], "intervals": make_intervals(), "rework": rework,
        "candidate": {}, "artifacts": [], "access_events": [], "evidence_refs": {},
    }


stages = [make_stage(i + 1, cat, rework=False) for i, cat in enumerate(STAGE_CATEGORIES)]
total_elapsed = float(STAGE_SECONDS * len(stages))
total_cost = STAGE_COST * len(stages)

lc = {
    "identity": {
        "schema_version": "1.0", "run_id": "run-tc090", "scenario_id": SCENARIO_ID,
        "scenario_version": "1", "fixture_id": "fixture-tc090", "fixture_digest": "c" * 64,
        "adapter_id": "fixture-adapter", "adapter_version": "1",
        "shark_binary_digest": "d" * 64, "shark_content_digest": "e" * 64, "roots": {},
    },
    "entity_graph": {}, "dispatches": [], "stages": stages,
    "workflow_policy": {}, "review_gates": [], "questions": [],
    "limits": {
        "max_cost_usd": 100.0, "max_wall_clock_seconds": 3600, "max_generated_tasks": 20,
        "observed_cost_usd": total_cost, "observed_wall_clock_seconds": total_elapsed,
        "observed_generated_tasks": 1, "first_exceeded": None,
    },
    "outcome": {"terminal": "complete", "reason": "tc090 fixture", "partial_evidence": False, "publication_eligible": True},
}
with open(os.path.join(pair_dir, "lifecycle.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(lc, sort_keys=True, separators=(",", ":")) + "\n")

ev = {
    "schema_version": "1.0", "evaluation_id": "eval-tc090", "identity": {}, "source_artifacts": {},
    "structural": {}, "judge": {}, "execution_oracle": {},
    "eligibility": {"aggregate_eligible": True, "publication_eligible": True, "invalidity_reasons": []},
    "candidate_snapshots": [], "workflow_policy": {}, "comparison": {},
    "metrics": {
        "elapsed_time": {"value": total_elapsed, "available": True, "detail": "sum of retained stage elapsed_seconds"},
        "provider_cost": {"value": total_cost, "available": True, "detail": "sum of retained stage cost_usd"},
        "rework": {"value": 0, "available": True, "detail": "count of retained stages marked rework"},
    },
}
with open(os.path.join(pair_dir, "evaluation.jsonl"), "w", encoding="utf-8") as f:
    f.write(json.dumps(ev, sort_keys=True, separators=(",", ":")) + "\n")


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


manifest = {
    "scenario_id": SCENARIO_ID, "rep": int(REP),
    "artifacts": {
        "package.yaml": {"source_path": os.path.join(pair_dir, "package.yaml"), "sha256": sha(os.path.join(pair_dir, "package.yaml"))},
        "lifecycle.jsonl": {"source_path": os.path.join(pair_dir, "lifecycle.jsonl"), "sha256": sha(os.path.join(pair_dir, "lifecycle.jsonl"))},
        "evaluation.jsonl": {"source_path": os.path.join(pair_dir, "evaluation.jsonl"), "sha256": sha(os.path.join(pair_dir, "evaluation.jsonl"))},
    },
}
with open(os.path.join(pair_dir, "manifest.json"), "w", encoding="utf-8") as f:
    json.dump(manifest, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")

batch = {
    "phase": "lifecycle_v2", "batch_id": "batch-tc090", "mode": "pilot", "min_reps": 1,
    "batch_policy_digest": "f" * 64,
    "ceilings": {"max_cost_usd": "100", "max_wall_clock_seconds": "3600", "max_generated_tasks": "20"},
    "acknowledgement_ref": {"flag": "--acknowledge-provider-spend", "present": True},
}
with open(os.path.join(dest_root, "batch.json"), "w", encoding="utf-8") as f:
    json.dump(batch, f, sort_keys=True, separators=(",", ":"))
    f.write("\n")
PYEOF
}

SMALL_ROOT="$WORKDIR/small-root"
mkdir -p "$SMALL_ROOT"
build_small_root "$SMALL_ROOT"

# ===========================================================================
# AC-T1: byte-identical double run (aggregate.json + both report views)
# ===========================================================================
AGG_1="$WORKDIR/agg-1.json"
AGG_2="$WORKDIR/agg-2.json"
"$AGGREGATOR" --retention-root "$SMALL_ROOT" >"$AGG_1" 2>"$WORKDIR/agg-1.err" ||
	fail "AC-T1: first aggregate run failed: $(cat "$WORKDIR/agg-1.err")"
"$AGGREGATOR" --retention-root "$SMALL_ROOT" >"$AGG_2" 2>"$WORKDIR/agg-2.err" ||
	fail "AC-T1: second aggregate run failed: $(cat "$WORKDIR/agg-2.err")"
cmp -s "$AGG_1" "$AGG_2" || fail "AC-T1: aggregate.json differs across two runs against the same retention root"

for view in headline stage_diagnostic; do
	"$REPORTER" --aggregate "$AGG_1" --view "$view" >"$WORKDIR/report-1-$view.md" 2>"$WORKDIR/report-1-$view.err" ||
		fail "AC-T1: first report ($view) run failed: $(cat "$WORKDIR/report-1-$view.err")"
	"$REPORTER" --aggregate "$AGG_1" --view "$view" >"$WORKDIR/report-2-$view.md" 2>"$WORKDIR/report-2-$view.err" ||
		fail "AC-T1: second report ($view) run failed: $(cat "$WORKDIR/report-2-$view.err")"
	cmp -s "$WORKDIR/report-1-$view.md" "$WORKDIR/report-2-$view.md" ||
		fail "AC-T1: report markdown ($view) differs across two runs over the same aggregate.json"
done
echo "TC-090 (AC-T1): byte-identical double run PASS (aggregate.json + headline + stage_diagnostic)"

# ===========================================================================
# Negative case (REQ-NF-003): fake-clock injection must not affect output.
# (a) Static scan: neither script may import/call a wall-clock API.
# (b) Runtime: a shifted TZ plus a hostile `date` shim (logs the call,
#     prints a corrupting sentinel, exits non-zero) must not change output
#     and must never actually be invoked.
# ===========================================================================
if grep -nE 'datetime\.(now|today)|time\.time\(|time\.localtime|time\.gmtime|date\.today' "$AGGREGATOR" "$REPORTER"; then
	fail "fake-clock static scan: found a wall-clock API reference in aggregate-lifecycle.sh/report-lifecycle.sh"
fi

CLOCK_BIN_DIR="$WORKDIR/clockbin"
CLOCK_LOG="$WORKDIR/clockbin.log"
mkdir -p "$CLOCK_BIN_DIR"
: >"$CLOCK_LOG"
cat >"$CLOCK_BIN_DIR/date" <<SHIM
#!/usr/bin/env bash
printf 'date %s\n' "\$*" >>"$CLOCK_LOG"
echo "Thu Jan  1 00:00:00 UTC 1970 (FAKE-CLOCK-SENTINEL-TC090)"
exit 1
SHIM
chmod +x "$CLOCK_BIN_DIR/date"

AGG_CLOCK="$WORKDIR/agg-clock.json"
TZ="Pacific/Kiritimati" PATH="$CLOCK_BIN_DIR:$PATH" "$AGGREGATOR" --retention-root "$SMALL_ROOT" \
	>"$AGG_CLOCK" 2>"$WORKDIR/agg-clock.err" ||
	fail "fake-clock leg: aggregate run failed under shifted TZ / hostile date shim: $(cat "$WORKDIR/agg-clock.err")"
cmp -s "$AGG_1" "$AGG_CLOCK" ||
	fail "fake-clock leg: aggregate.json changed under a shifted TZ / hostile date shim (REQ-NF-003 violation)"

REPORT_CLOCK="$WORKDIR/report-clock-headline.md"
TZ="Pacific/Kiritimati" PATH="$CLOCK_BIN_DIR:$PATH" "$REPORTER" --aggregate "$AGG_1" --view headline \
	>"$REPORT_CLOCK" 2>"$WORKDIR/report-clock.err" ||
	fail "fake-clock leg: report run failed under shifted TZ / hostile date shim: $(cat "$WORKDIR/report-clock.err")"
cmp -s "$WORKDIR/report-1-headline.md" "$REPORT_CLOCK" ||
	fail "fake-clock leg: headline report changed under a shifted TZ / hostile date shim (REQ-NF-003 violation)"
[[ ! -s "$CLOCK_LOG" ]] || fail "fake-clock leg: date binary was invoked (should never be consulted): $(cat "$CLOCK_LOG")"
echo "TC-090 (negative case): fake-clock injection has no effect on output; date never invoked PASS"

# ===========================================================================
# AC-T2: zero denied calls across provider, network, DB, and live-tree
# denial surfaces, exercised together with a second byte-identical double
# run (the same properties AC-T1 proved, now proved again under denial).
# ===========================================================================
path_shim_denial_setup "$WORKDIR/provnet" || fail "path-shim-denial setup failed"

# DB-connection-attempt denial: a second, independently-named PATH-shim
# binary list (not schema-driven -- REQ-NF-002's provider_and_network_
# binaries vocabulary is scoped to provider/network only). Any of these
# ever being invoked by a pure aggregator/reporter would itself be the
# defect this leg exists to catch.
DB_DENY_BINARIES=(shark sqlite3 turso psql mysql libsql)
DB_BIN_DIR="$WORKDIR/dbdenybin"
DB_LOG="$WORKDIR/dbdeny.log"
mkdir -p "$DB_BIN_DIR"
: >"$DB_LOG"
for name in "${DB_DENY_BINARIES[@]}"; do
	cat >"$DB_BIN_DIR/$name" <<SHIM
#!/usr/bin/env bash
printf '%s %s\n' "$name" "\$*" >>"$DB_LOG"
exit 1
SHIM
	chmod +x "$DB_BIN_DIR/$name"
done

# Live-working-tree-write-denial: snapshot the real repo's working tree
# across the whole denial run. Neither script contains a write-mode
# open()/.write()/mkdir/shutil call (confirmed statically below too), so
# this is a real regression catch, not a tautology.
if grep -nE "open\([^)]*['\"]w[b]?['\"]|\.write\(|os\.makedirs|os\.mkdir|shutil\." "$AGGREGATOR" "$REPORTER"; then
	fail "live-tree static scan: found a write-capable call in aggregate-lifecycle.sh/report-lifecycle.sh"
fi
GIT_STATUS_BEFORE="$(git -C "$REPO_ROOT" status --porcelain)"

export PATH="$PATH_SHIM_DENIAL_BIN_DIR:$DB_BIN_DIR:$PATH"

DENY_AGG_1="$WORKDIR/deny-agg-1.json"
DENY_AGG_2="$WORKDIR/deny-agg-2.json"
"$AGGREGATOR" --retention-root "$SMALL_ROOT" >"$DENY_AGG_1" 2>"$WORKDIR/deny-agg-1.err" ||
	fail "AC-T2: first aggregate run under denial failed: $(cat "$WORKDIR/deny-agg-1.err")"
"$AGGREGATOR" --retention-root "$SMALL_ROOT" >"$DENY_AGG_2" 2>"$WORKDIR/deny-agg-2.err" ||
	fail "AC-T2: second aggregate run under denial failed: $(cat "$WORKDIR/deny-agg-2.err")"
cmp -s "$DENY_AGG_1" "$DENY_AGG_2" || fail "AC-T2: aggregate.json differs across two denied-PATH runs"
cmp -s "$AGG_1" "$DENY_AGG_1" || fail "AC-T2: aggregate.json under denial differs from the undenied baseline"

for view in headline stage_diagnostic; do
	"$REPORTER" --aggregate "$DENY_AGG_1" --view "$view" >"$WORKDIR/deny-report-1-$view.md" 2>"$WORKDIR/deny-report-1-$view.err" ||
		fail "AC-T2: first report ($view) run under denial failed: $(cat "$WORKDIR/deny-report-1-$view.err")"
	"$REPORTER" --aggregate "$DENY_AGG_1" --view "$view" >"$WORKDIR/deny-report-2-$view.md" 2>"$WORKDIR/deny-report-2-$view.err" ||
		fail "AC-T2: second report ($view) run under denial failed: $(cat "$WORKDIR/deny-report-2-$view.err")"
	cmp -s "$WORKDIR/deny-report-1-$view.md" "$WORKDIR/deny-report-2-$view.md" ||
		fail "AC-T2: report markdown ($view) differs across two denied-PATH runs"
	cmp -s "$WORKDIR/report-1-$view.md" "$WORKDIR/deny-report-1-$view.md" ||
		fail "AC-T2: report markdown ($view) under denial differs from the undenied baseline"
done

GIT_STATUS_AFTER="$(git -C "$REPO_ROOT" status --porcelain)"
[[ "$GIT_STATUS_BEFORE" == "$GIT_STATUS_AFTER" ]] ||
	fail "AC-T2 (live-working-tree denial): repo working tree changed during the run:
before: $GIT_STATUS_BEFORE
after:  $GIT_STATUS_AFTER"

path_shim_denial_assert_empty "TC-090 provider/network denial" || fail "provider/network invocation detected"
[[ ! -s "$DB_LOG" ]] || fail "AC-T2 (DB denial): a DB-related binary was invoked: $(cat "$DB_LOG")"
echo "TC-090 (AC-T2): zero denied calls across provider/network/DB/live-tree surfaces PASS"

# ===========================================================================
# AC-T3: generated ~100 MB retention fixture -- 60s wall-time bound and
# 25 MB (or largest-file+5MB) peak-RSS bound (REQ-NF-004).
# ===========================================================================

# Generator determinism self-check (Scope: "same seed produces the same
# retention tree byte-for-byte"). Uses a small --total-bytes override so
# this check stays cheap; the full ~100 MB fixture below is generated once,
# per the task's "not regenerated per test run" instruction.
manifest_of() {
	local root="$1"
	(cd "$root" && find . -type f | sort | xargs -r sha256sum)
}

DET_DIR="$WORKDIR/det-fixture"
"$GEN" --seed "tc090-det-seed" --out "$DET_DIR" --total-bytes 2097152 >/dev/null 2>"$WORKDIR/gen-det-1.err" ||
	fail "generator determinism check: first small generation failed: $(cat "$WORKDIR/gen-det-1.err")"
manifest_of "$DET_DIR" >"$WORKDIR/det-manifest-1.txt"
rm -rf "$DET_DIR"
"$GEN" --seed "tc090-det-seed" --out "$DET_DIR" --total-bytes 2097152 >/dev/null 2>"$WORKDIR/gen-det-2.err" ||
	fail "generator determinism check: second small generation failed: $(cat "$WORKDIR/gen-det-2.err")"
manifest_of "$DET_DIR" >"$WORKDIR/det-manifest-2.txt"
cmp -s "$WORKDIR/det-manifest-1.txt" "$WORKDIR/det-manifest-2.txt" ||
	fail "generator determinism check: same --seed produced a different retention tree across two generations"
rm -rf "$DET_DIR"
echo "TC-090 (generator determinism): same seed -> byte-identical retention tree PASS"

# The real ~100 MB fixture, generated once (per task Notes: "run once at
# test-fixture setup time, not regenerated per test run"), into a /tmp
# scratch location (never the repo), cleaned up by this script's EXIT trap
# regardless of pass/fail.
FIXTURE_ROOT="$WORKDIR/fixture-100mb"
GEN_SUMMARY="$("$GEN" --seed "tc090-scale-seed" --out "$FIXTURE_ROOT" 2>"$WORKDIR/gen-scale.err")" ||
	fail "100 MB fixture generation failed: $(cat "$WORKDIR/gen-scale.err")"

LARGEST_FILE_BYTES="$(python3 -c "import json,sys; print(json.loads(sys.argv[1])['largest_padding_file_bytes'])" "$GEN_SUMMARY")"
[[ -n "$LARGEST_FILE_BYTES" ]] || fail "could not read largest_padding_file_bytes from generator summary: $GEN_SUMMARY"

FIXTURE_BYTES="$(du -sb "$FIXTURE_ROOT" | cut -f1)"
[[ "$FIXTURE_BYTES" -ge 100000000 ]] || fail "generated fixture is smaller than 100 MB ($FIXTURE_BYTES bytes)"

# REQ-NF-004 / TC-090 Expected Output: 25 MB flat, or largest-single-file+5MB
# if a single retained file exceeds 25 MB.
BOUND_FLAT=$((25 * 1024 * 1024))
BOUND_FALLBACK=$((LARGEST_FILE_BYTES + 5 * 1024 * 1024))
if [[ "$BOUND_FALLBACK" -gt "$BOUND_FLAT" ]]; then
	RSS_BOUND_BYTES="$BOUND_FALLBACK"
else
	RSS_BOUND_BYTES="$BOUND_FLAT"
fi

AGG_100="$WORKDIR/agg-100mb.json"
TIME_LOG="$WORKDIR/time-100mb.log"
START_EPOCH="$(date +%s)"
/usr/bin/time -v "$AGGREGATOR" --retention-root "$FIXTURE_ROOT" >"$AGG_100" 2>"$TIME_LOG" ||
	fail "AC-T3: aggregate run over the 100 MB fixture failed; see $TIME_LOG: $(cat "$TIME_LOG")"
END_EPOCH="$(date +%s)"
WALL_SECONDS=$((END_EPOCH - START_EPOCH))

[[ "$WALL_SECONDS" -le 60 ]] || fail "AC-T3: aggregate run over the 100 MB fixture took ${WALL_SECONDS}s, exceeding the 60s bound (REQ-NF-004)"

PEAK_RSS_KB="$(grep "Maximum resident set size" "$TIME_LOG" | grep -oE '[0-9]+')"
[[ -n "$PEAK_RSS_KB" ]] || fail "AC-T3: could not parse peak RSS from /usr/bin/time -v output: $(cat "$TIME_LOG")"
PEAK_RSS_BYTES=$((PEAK_RSS_KB * 1024))

[[ "$PEAK_RSS_BYTES" -le "$RSS_BOUND_BYTES" ]] ||
	fail "AC-T3: peak RSS ${PEAK_RSS_BYTES} bytes exceeds the ${RSS_BOUND_BYTES}-byte bound (100 MB fixture; largest single retained file ${LARGEST_FILE_BYTES} bytes) -- aggregator is loading retained payloads instead of streaming (REQ-NF-004)"

# report-lifecycle.sh only ever reads the (small) aggregate.json this
# aggregator just produced -- never the retention root -- so it is
# structurally unaffected by fixture scale; still exercised end-to-end here
# to prove the whole scale pipeline completes.
for view in headline stage_diagnostic; do
	"$REPORTER" --aggregate "$AGG_100" --view "$view" >"$WORKDIR/report-100mb-$view.md" 2>"$WORKDIR/report-100mb-$view.err" ||
		fail "AC-T3: report ($view) over the 100 MB fixture's aggregate.json failed: $(cat "$WORKDIR/report-100mb-$view.err")"
done

echo "TC-090 (AC-T3): 100 MB fixture (${FIXTURE_BYTES} bytes) aggregated in ${WALL_SECONDS}s, peak RSS ${PEAK_RSS_KB} KiB (bound $((RSS_BOUND_BYTES / 1024)) KiB) PASS"

echo "TC-090: offline determinism and 100 MB retention-fixture scale test PASS"
