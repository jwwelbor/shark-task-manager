#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
VERIFIER="$REPO_ROOT/bench/scripts/verify-lifecycle-evaluation.sh"
SCHEMA="$REPO_ROOT/bench/evaluation/i08-schema.yaml"
VALID="$REPO_ROOT/tests/contracts/testdata/e40_i08/valid/eligible.json"
[[ -x "$VERIFIER" ]] || { echo "TC-076: verifier missing" >&2; exit 1; }

for script in evaluate-lifecycle.sh run-heldback-oracle.sh compare-lifecycle-evaluations.sh normalize-review-findings.sh verify-lifecycle-evaluation.sh; do
  path="$REPO_ROOT/bench/scripts/$script"
  [[ -x "$path" ]] || { echo "TC-076: missing executable surface: $script" >&2; exit 1; }
  if rg -n 'curl|wget|(^|[^[:alnum:]_])(nc|netcat)([^[:alnum:]_]|$)|https?://|shark-tasks\.db|\.sharkconfig' "$path" >/dev/null; then
    echo "TC-076: forbidden egress/product path detected in $script" >&2
    exit 1
  fi
done
[[ "$(wc -c < "$VERIFIER")" -lt 50000 ]]

tmp="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc076.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT
python3 - "$VALID" "$tmp/100mb.jsonl" <<'PY'
import json
import pathlib
import sys

valid = json.dumps(json.loads(pathlib.Path(sys.argv[1]).read_text()), separators=(",", ":")).encode() + b"\n"
out = pathlib.Path(sys.argv[2])
with out.open("wb") as stream:
    remaining = 100 * 1024 * 1024
    chunk = b"\n" * (1024 * 1024)
    while remaining >= len(chunk):
        stream.write(chunk)
        remaining -= len(chunk)
    stream.write(b"\n" * remaining)
    stream.write(valid)
PY
timeout 30s "$VERIFIER" "$tmp/100mb.jsonl" --schema "$SCHEMA" >"$tmp/verdict" 2>"$tmp/error" || {
  echo "TC-076: 100 MB streaming verification exceeded 30 seconds or rejected the valid tail" >&2
  cat "$tmp/error" >&2
  exit 1
}
grep -q 'aggregate_eligible' "$tmp/verdict"
echo "TC-076 PASS: full-surface safety scan and 100 MB streaming boundary pass"
