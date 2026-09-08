#!/usr/bin/env bash
# verify-retention-manifest.sh <registry_id> [--root <path>]
#
# Standalone, operator-invocable verifier for AC-F11-01b (spec.md 2.1.2a):
# recomputes a registered retention root's whole-tree digest manifest and
# diffs it against the manifest captured at registration time
# (bench/retention-manifests/<registry_id>.sha256). Read-only -- it never
# writes to the registered root, the registry, or any manifest file.
#
# `--root <path>` recomputes against a DIFFERENT path (a disposable copy)
# while still diffing against the real registry entry's recorded manifest.
# tc099's 5-mutation sweep uses this to prove the check catches a modified,
# added, removed, renamed, or retargeted-symlink file without ever mutating
# the real registered root.
#
# Prints one JSON line to stdout:
#   {"status": "match"|"mismatch"|"not_present_on_host",
#    "registry_id": ..., "root_path": ..., "differences": [...]}
# ("differences" is present only for "mismatch", each entry naming the
# differing relative path and its change: "added"|"removed"|"modified".)
#
# Exit status: 0 match, 1 mismatch, 3 not_present_on_host, 2 usage/registry
# error (e.g. unknown registry_id, malformed registry entry).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE="$SCRIPT_DIR/lib/e40_benchmark.py"

usage() {
	echo "usage: verify-retention-manifest.sh <registry_id> [--root <path>]" >&2
	exit 2
}

[[ $# -ge 1 ]] || usage
registry_id="$1"
shift
root_override=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--root)
		[[ $# -ge 2 ]] || usage
		root_override="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

[[ -f "$MODULE" ]] || {
	echo "verify-retention-manifest: e40_benchmark.py not found: $MODULE" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "verify-retention-manifest: python3 not found on PATH" >&2
	exit 2
}

python3 - "$MODULE" "$registry_id" "$root_override" <<'PY'
import importlib.util
import json
import sys

module_path, registry_id, root_override = sys.argv[1:4]
spec = importlib.util.spec_from_file_location("e40_benchmark", module_path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

try:
    result = module.verify_tree_manifest(
        registry_id,
        root_override=module.Path(root_override) if root_override else None,
    )
except module.OperatorError as exc:
    print(f"verify-retention-manifest: {exc}", file=sys.stderr)
    raise SystemExit(2)

print(json.dumps(result, sort_keys=True))
status = result["status"]
if status == "match":
    raise SystemExit(0)
if status == "not_present_on_host":
    raise SystemExit(3)
raise SystemExit(1)
PY
