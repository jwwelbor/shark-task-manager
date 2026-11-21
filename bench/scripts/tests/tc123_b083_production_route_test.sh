#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
command -v python3 >/dev/null 2>&1 || {
	echo "TC-123 B083 FAIL: python3 not found on PATH" >&2
	exit 1
}
python3 -m pytest -q \
	"$SCRIPT_DIR/test_b083_codex_cli.py" \
	"$SCRIPT_DIR/test_b083_codex_catalog.py" \
	"$SCRIPT_DIR/test_b083_codex_catalog_identity.py"
python3 "$SCRIPT_DIR/tc123_b083_production_route_test.py"
