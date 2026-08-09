#!/usr/bin/env bash
# TC-004 (test-plan.md AC test matrix; T-E40-F01-006 task spec Test Cases).
#
# Exercises AC-003 / AC-T1: `admit.sh <corpus_yaml> ` (no `--item`) over the
# full admitted set --
#   - all five admission checks pass for each of the >=10 admitted items
#   - >=1 `bug`-type item's manifest-named repro test is confirmed
#   - output is exactly one JSON verdict line per item, sorted by item id
#   - admit.sh's own exit code is its summary assertion (0 == gate satisfied)
#
# Caller-Path Contract (test-plan.md TC-004): drives the real
# `bench/scripts/admit.sh` entrypoint, which itself shells out to real `git
# apply` and real `go test` inside a fresh checkout-fixture.sh checkout --
# no subprocess result is stubbed and no per-item boolean is hardcoded here.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-004 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

stdout_file="$WORKDIR/stdout.jsonl"
set +e
"$ADMIT_SCRIPT" "$CORPUS_YAML" >"$stdout_file"
exit_code=$?
set -e

[[ "$exit_code" -eq 0 ]] || fail "admit.sh exited $exit_code over the full admitted set (expected 0); output: $(cat "$stdout_file")"

expected_ids="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
for item in data["items"]:
    print(item["id"])
PYEOF
)"
expected_count="$(printf '%s\n' "$expected_ids" | grep -c .)"
[[ "$expected_count" -ge 10 ]] || fail "corpus.yaml has only $expected_count items, need >= 10 to test AC-003 meaningfully"

actual_count="$(grep -c . "$stdout_file" || true)"
[[ "$actual_count" -eq "$expected_count" ]] || fail "expected exactly $expected_count verdict lines (one per admitted item), got $actual_count"

python3 - "$stdout_file" "$expected_count" <<'PYEOF'
import json
import sys

stdout_path, expected_count = sys.argv[1], int(sys.argv[2])

with open(stdout_path) as f:
    lines = [line for line in f.read().splitlines() if line]

verdicts = [json.loads(line) for line in lines]

ids = [v["item_id"] for v in verdicts]
if ids != sorted(ids):
    sys.exit(f"TC-004 FAIL: verdict lines are not sorted by item_id: {ids}")

not_admitted = [v for v in verdicts if v["status"] != "admitted"]
if not_admitted:
    sys.exit(
        "TC-004 FAIL: not every admitted-set item passed all five checks: "
        + ", ".join(f"{v['item_id']} ({v['failing_check']})" for v in not_admitted)
    )

for v in verdicts:
    for check_name, result in v["checks"].items():
        if result is not True:
            sys.exit(f"TC-004 FAIL: {v['item_id']} check {check_name!r} was {result!r}, want True")

admitted_count = sum(1 for v in verdicts if v["status"] == "admitted")
if admitted_count < 10:
    sys.exit(f"TC-004 FAIL: only {admitted_count} admitted items, need >= 10 (REQ-F-006)")

bug_confirmed = [v for v in verdicts if v["type"] == "bug" and v.get("repro_confirmed") is True]
if not bug_confirmed:
    sys.exit("TC-004 FAIL: no admitted bug-type item carries a confirmed repro-test verdict")

print(f"TC-004: {admitted_count} admitted items, sorted, all five checks pass;")
print(f"        {len(bug_confirmed)} bug item(s) with repro_confirmed=true: "
      + ", ".join(v["item_id"] for v in bug_confirmed))
PYEOF

echo "TC-004: PASS"
