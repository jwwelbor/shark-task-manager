#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
VERIFIER="$REPO_ROOT/bench/scripts/verify-lifecycle-evaluation.sh"
[[ -x "$VERIFIER" ]] || { echo "TC-076: verifier missing" >&2; exit 1; }
if rg -n 'curl|wget|shark-tasks\.db|\.sharkconfig|go test|pytest|npm test|read_text\(' "$VERIFIER" >/dev/null; then
  echo "TC-076: forbidden egress/product path/language branch/read detected" >&2; exit 1
fi
[[ "$(wc -c < "$VERIFIER")" -lt 50000 ]]
echo "TC-076: static safety and language neutrality pass"
