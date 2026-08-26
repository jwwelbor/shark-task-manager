#!/usr/bin/env bash
# smoke-lifecycle.sh [--out <dir>] [--scenario <scenario_id>] [--live]
#                     [--adapter <path>] [--run-id <id>]
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
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

command -v python3 >/dev/null 2>&1 || {
	echo "smoke-lifecycle: python3 not found on PATH" >&2
	exit 2
}

OUT="/tmp/e40-smoke"
SCENARIO="py-bug-due-date-boundary"
LIVE=0
ADAPTER="$SCRIPT_DIR/lifecycle-worker-adapter.sh"
RUN_ID=""

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
	-h | --help)
		echo "usage: smoke-lifecycle.sh [--out <dir>] [--scenario <scenario_id>] [--live] [--adapter <path>] [--run-id <id>]"
		exit 0
		;;
	*)
		echo "smoke-lifecycle: unrecognized argument: $1" >&2
		exit 2
		;;
	esac
done

echo "smoke-lifecycle: preparing scratch project under $OUT" >&2
python3 "$BENCH_DIR/scripts/lib/e40_benchmark.py" setup --out "$OUT" >"$OUT.setup-status.json.tmp" 2>&1 ||
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

root_key = result["root_keys"].get(entry["family"])
if not root_key:
    print(f"smoke-lifecycle: no seeded root_key for family {entry['family']!r}", file=sys.stderr)
    raise SystemExit(2)

print(result["shark_binary"]["path"], result["scratch_root"], entry["package_path"], root_key)
PY
)"

RUN_ID="${RUN_ID:-$SCENARIO-smoke}"
LIFECYCLE_OUT="$OUT/runs/$RUN_ID/lifecycle.jsonl"
mkdir -p "$(dirname "$LIFECYCLE_OUT")"

export SHARK_BIN="$SHARK_PATH"

echo "smoke-lifecycle: dry-run against scenario $SCENARIO, root $ROOT_KEY" >&2
DRY_RUN_STATUS=0
"$SCRIPT_DIR/run-lifecycle.sh" \
	--scenario "$PACKAGE_PATH" \
	--run-id "$RUN_ID" \
	--root "$ROOT_KEY" \
	--scratch-root "$SCRATCH_ROOT" \
	--output "$LIFECYCLE_OUT" \
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
"$SCRIPT_DIR/verify-lifecycle-run.sh" "$LIFECYCLE_OUT" --schema "$BENCH_DIR/runs/i07-schema.yaml" || VERIFY_STATUS=$?

if [[ "$DRY_RUN_STATUS" -eq 0 && "$VERIFY_STATUS" -eq 0 ]]; then
	echo "smoke-lifecycle: dry-run OK -- record at $LIFECYCLE_OUT" >&2
else
	echo "smoke-lifecycle: dry-run finished with issues (run exit=$DRY_RUN_STATUS, verify exit=$VERIFY_STATUS) -- see $LIFECYCLE_OUT" >&2
	exit 1
fi

if [[ "$LIVE" -eq 1 ]]; then
	LIVE_RUN_ID="$RUN_ID-live"
	LIVE_OUT="$OUT/runs/$LIVE_RUN_ID/lifecycle.jsonl"
	mkdir -p "$(dirname "$LIVE_OUT")"
	echo "smoke-lifecycle: live run against scenario $SCENARIO, root $ROOT_KEY (this spends provider credit)" >&2
	export LIFECYCLE_ADAPTER="$ADAPTER"
	"$SCRIPT_DIR/run-lifecycle.sh" \
		--scenario "$PACKAGE_PATH" \
		--run-id "$LIVE_RUN_ID" \
		--root "$ROOT_KEY" \
		--scratch-root "$SCRATCH_ROOT" \
		--output "$LIVE_OUT" \
		--mode live
	echo "smoke-lifecycle: live run complete -- record at $LIVE_OUT" >&2
fi
