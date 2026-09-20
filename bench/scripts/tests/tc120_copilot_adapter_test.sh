#!/usr/bin/env bash
# TC-F13-01 / TC-F13-02 / TC-F13-12 (AC-F13-01, AC-F13-02, AC-F13-12).
# Unit tests for GitHub Copilot CLI command assembly and sandbox confinement in lifecycle-worker-adapter.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_PY="$SCRIPT_DIR/tc120_copilot_adapter_test.py"

fail() {
	echo "TC-F13-01/02/12 FAIL: $1" >&2
	exit 1
}

[[ -f "$TEST_PY" ]] || fail "test script missing: $TEST_PY"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

if python3 -c "import pytest" >/dev/null 2>&1; then
	python3 -m pytest -q "$TEST_PY"
else
	python3 "$TEST_PY"
fi

echo "TC-F13-01/02/12: pass"
