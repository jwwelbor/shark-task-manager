#!/usr/bin/env bash
# TC-F13-11 / TC-F13-13 (AC-F13-11, AC-F13-13).
# Unit and contract tests for E40 model routing profiles and workflow routing identity.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_PY="$SCRIPT_DIR/tc122_copilot_routing_identity_test.py"

fail() {
	echo "TC-F13-11/13 FAIL: $1" >&2
	exit 1
}

[[ -f "$TEST_PY" ]] || fail "test script missing: $TEST_PY"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

if python3 -c "import pytest" >/dev/null 2>&1; then
	python3 -m pytest -q "$TEST_PY"
else
	python3 "$TEST_PY"
fi

echo "TC-F13-11/13: pass"
