#!/usr/bin/env bash
# smoke-lifecycle.sh [--out <dir>] [--scenario <scenario_id>] [--live]
#                     [--adapter <path>] [--fixture-root <dir>] [--run-id <id>]
#
# Operator convenience wrapper around the "Operator quick start: CLI
# dispatch smoke test" sequence documented in bench/README.md. It resolves
# every variable that sequence otherwise asks for by hand (scratch project
# path, seeded root entity key, scenario package path, shark binary) from
# `e40-benchmark.sh setup`'s own setup-result.json, then drives
# run-lifecycle.sh (dry-run, then optionally live) and verify-lifecycle-run.sh.
#
# It does not replace run-lifecycle.sh or e40-benchmark.sh -- it only chains
# their existing, unmodified interfaces so a single command reproduces the
# README sequence. See bench/README.md's "Operator quick start" section for
# what each step actually does and why.
#
# Each run-lifecycle.sh invocation this script drives gets its own I-05
# evidence bundle directory under $OUT/runs/<run-id>/i05 (T-E40-F12-006):
# required for the --live invocation (run-lifecycle.sh --mode live fails
# before the first dispatch without it), harmless to also pass for the
# dry-run.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# T-E40-F12-006 kickback fix: resolved the same way run-lifecycle-batch.sh
# and run-review-comparison.sh resolve their own driver binaries, so a test
# can substitute a stub and record the real argv this script hands them --
# an exit-code-only check would not have caught the missing
# --i05-bundle-dir pass-through this variable now carries.
E40_BENCHMARK_BIN="${E40_BENCHMARK_BIN:-$BENCH_DIR/scripts/e40-benchmark.sh}"
RUN_LIFECYCLE_BIN="${RUN_LIFECYCLE_BIN:-$SCRIPT_DIR/run-lifecycle.sh}"
VERIFY_LIFECYCLE_RUN_BIN="${VERIFY_LIFECYCLE_RUN_BIN:-$SCRIPT_DIR/verify-lifecycle-run.sh}"

command -v python3 >/dev/null 2>&1 || {
	echo "smoke-lifecycle: python3 not found on PATH" >&2
	exit 2
}

OUT="/tmp/e40-smoke"
SCENARIO="py-bug-due-date-boundary"
LIVE=0
ADAPTER="$SCRIPT_DIR/lifecycle-worker-adapter.sh"
RUN_ID=""
FIXTURE_ROOT=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--out)
		OUT="$2"
		shift 2
		;;
	--scenario)
		SCENARIO="$2"
		shift 2
		;;
	--live)
		LIVE=1
		shift
		;;
	--adapter)
		ADAPTER="$2"
		shift 2
		;;
	--run-id)
		RUN_ID="$2"
		shift 2
		;;
	--fixture-root)
		FIXTURE_ROOT="$2"
		shift 2
		;;
	-h | --help)
		echo "usage: smoke-lifecycle.sh [--out <dir>] [--scenario <scenario_id>] [--live] [--adapter <path>] [--fixture-root <dir>] [--run-id <id>]"
		exit 0
		;;
	*)
		echo "smoke-lifecycle: unrecognized argument: $1" >&2
		exit 2
		;;
	esac
done

echo "smoke-lifecycle: preparing scratch project under $OUT" >&2
"$E40_BENCHMARK_BIN" setup --out "$OUT" >"$OUT.setup-status.json.tmp" 2>&1 ||
	{
		cat "$OUT.setup-status.json.tmp" >&2
		rm -f "$OUT.setup-status.json.tmp"
		exit 1
	}
rm -f "$OUT.setup-status.json.tmp"

RESULT="$OUT/setup-result.json"
[[ -f "$RESULT" ]] || {
	echo "smoke-lifecycle: setup did not produce $RESULT" >&2
	exit 1
}

read -r SHARK_PATH SCRATCH_ROOT PACKAGE_PATH ROOT_KEY <<<"$(
	python3 - "$RESULT" "$SCENARIO" <<'PY'
import json
import sys

result_path, scenario_id = sys.argv[1], sys.argv[2]
with open(result_path, encoding="utf-8") as stream:
    result = json.load(stream)

entry = next((item for item in result["scenario_matrix"] if item["scenario_id"] == scenario_id), None)
if entry is None:
    known = ", ".join(sorted(item["scenario_id"] for item in result["scenario_matrix"]))
    print(f"smoke-lifecycle: unknown scenario {scenario_id!r}; known scenarios: {known}", file=sys.stderr)
    raise SystemExit(2)

root_key = result["root_keys"].get(entry["scenario_id"])
if not root_key:
    print(f"smoke-lifecycle: no seeded root_key for scenario {entry['scenario_id']!r}", file=sys.stderr)
    raise SystemExit(2)

print(result["shark_binary"]["path"], result["scratch_root"], entry["package_path"], root_key)
PY
)"

RUN_ID="${RUN_ID:-$SCENARIO-smoke}"
LIFECYCLE_OUT="$OUT/runs/$RUN_ID/lifecycle.jsonl"
mkdir -p "$(dirname "$LIFECYCLE_OUT")"
# T-E40-F12-006 kickback fix: each run gets its own I-05 bundle directory,
# named after that run's own run-id (mirroring LIFECYCLE_OUT's own
# per-run-id layout) -- the dry-run and live runs are separate
# run-lifecycle.sh invocations with separate identities, and the producer's
# reset is scoped to whatever directory it is given, so sharing one
# directory between them would let the live run reset the dry run's bundle.
I05_OUT="$OUT/runs/$RUN_ID/i05"
FIXTURE_ARGS=()
if [[ -n "$FIXTURE_ROOT" ]]; then
	FIXTURE_ARGS=(--fixture-root "$FIXTURE_ROOT")
fi

export SHARK_BIN="$SHARK_PATH"

echo "smoke-lifecycle: dry-run against scenario $SCENARIO, root $ROOT_KEY" >&2
DRY_RUN_STATUS=0
"$RUN_LIFECYCLE_BIN" \
	--scenario "$PACKAGE_PATH" \
	--run-id "$RUN_ID" \
	--root "$ROOT_KEY" \
	--scratch-root "$SCRATCH_ROOT" \
	--output "$LIFECYCLE_OUT" \
	--i05-bundle-dir "$I05_OUT" \
	"${FIXTURE_ARGS[@]}" \
	--mode dry-run || DRY_RUN_STATUS=$?

if [[ "$DRY_RUN_STATUS" -ne 0 ]]; then
	echo "smoke-lifecycle: run-lifecycle.sh exited $DRY_RUN_STATUS -- record's own outcome.reason:" >&2
	python3 -c "
import json
with open('$LIFECYCLE_OUT', encoding='utf-8') as f:
    print(json.dumps(json.load(f)['outcome'], indent=2))
" >&2 || echo "smoke-lifecycle: could not read $LIFECYCLE_OUT for detail" >&2
fi

echo "smoke-lifecycle: verifying $LIFECYCLE_OUT against the I-07 schema" >&2
VERIFY_STATUS=0
"$VERIFY_LIFECYCLE_RUN_BIN" "$LIFECYCLE_OUT" --schema "$BENCH_DIR/runs/i07-schema.yaml" || VERIFY_STATUS=$?

if [[ "$DRY_RUN_STATUS" -eq 0 && "$VERIFY_STATUS" -eq 0 ]]; then
	echo "smoke-lifecycle: dry-run OK -- record at $LIFECYCLE_OUT" >&2
elif [[ "$LIVE" -eq 0 ]]; then
	echo "smoke-lifecycle: dry-run finished with issues (run exit=$DRY_RUN_STATUS, verify exit=$VERIFY_STATUS) -- see $LIFECYCLE_OUT" >&2
	exit 1
else
	echo "smoke-lifecycle: dry-run is diagnostic-only before requested live capture (run exit=$DRY_RUN_STATUS, verify exit=$VERIFY_STATUS)" >&2
fi

if [[ "$LIVE" -eq 1 ]]; then
	LIVE_RUN_ID="$RUN_ID-live"
	LIVE_OUT="$OUT/runs/$LIVE_RUN_ID/lifecycle.jsonl"
	mkdir -p "$(dirname "$LIVE_OUT")"
	LIVE_I05_OUT="$OUT/runs/$LIVE_RUN_ID/i05"
	echo "smoke-lifecycle: live run against scenario $SCENARIO, root $ROOT_KEY (this spends provider credit)" >&2
	export LIFECYCLE_ADAPTER="$ADAPTER"
	LIVE_RUN_STATUS=0
	"$RUN_LIFECYCLE_BIN" \
		--scenario "$PACKAGE_PATH" \
		--run-id "$LIVE_RUN_ID" \
		--root "$ROOT_KEY" \
		--scratch-root "$SCRATCH_ROOT" \
		--output "$LIVE_OUT" \
		--i05-bundle-dir "$LIVE_I05_OUT" \
		"${FIXTURE_ARGS[@]}" \
		--mode live || LIVE_RUN_STATUS=$?
	if [[ "$LIVE_RUN_STATUS" -ne 0 ]]; then
		echo "smoke-lifecycle: live run-lifecycle.sh exited $LIVE_RUN_STATUS -- record's own outcome.reason:" >&2
		python3 -c "
import json
with open('$LIVE_OUT', encoding='utf-8') as f:
    print(json.dumps(json.load(f)['outcome'], indent=2))
" >&2 || echo "smoke-lifecycle: could not read $LIVE_OUT for detail" >&2
		exit "$LIVE_RUN_STATUS"
	fi
	echo "smoke-lifecycle: live run complete -- record at $LIVE_OUT, I-05 bundle at $LIVE_I05_OUT" >&2
fi
