#!/usr/bin/env bash
# Safe top-level operator entrypoint for repeatable E40 lifecycle benchmarks.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

command -v python3 >/dev/null 2>&1 || {
	echo "e40-benchmark: python3 not found on PATH" >&2
	exit 2
}

exec python3 "$SCRIPT_DIR/lib/e40_benchmark.py" "$@"
