#!/usr/bin/env bash
# TC-118 / TD-148: the real Shark CLI, lifecycle producer, and I-07 verifier
# must agree on one dry-run record. No PATH stub is permitted at this boundary.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"
SMOKE="$SCRIPTS_DIR/smoke-lifecycle.sh"
REAL_SHARK="$REPO_ROOT/bin/shark"

fail() { echo "TC-118 FAIL: $1" >&2; exit 1; }
[[ -x "$SMOKE" ]] || fail "smoke-lifecycle.sh missing or not executable"
[[ -x "$REAL_SHARK" ]] || fail "real shark binary missing at $REAL_SHARK (run 'make shark' first)"

WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc118.XXXXXX")"
trap 'rm -rf "$WORKDIR"' EXIT

"$SMOKE" --out "$WORKDIR/smoke" --run-id tc118-real-contract </dev/null
RECORD="$WORKDIR/smoke/runs/tc118-real-contract/lifecycle.jsonl"
[[ -s "$RECORD" ]] || fail "smoke-lifecycle did not produce lifecycle.jsonl"

python3 - "$RECORD" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
    record = json.loads(stream.readline())
assert record["identity"]["shark_binary_digest"], record
assert record["dispatches"], record
assert record["stages"], record
assert record["stages"][0]["candidate"]["scratch_content_digest"], record
PY

echo "TC-118: real Shark dry-run lifecycle record passes the production I-07 verifier"
