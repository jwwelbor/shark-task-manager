#!/usr/bin/env bash
# export-entity-history.sh --scratch-root <dir> --root-key <key> --output <path>
#
# T-E40-F10-004/005 (UAT-R3-01 fix; spec.md REQ-F-004 "the scratch-project
# entity history"): the real producer for retain_pair's entity-history.json
# artifact. Runs `shark history <root-key> --json` with the scratch shark
# project (--scratch-root) as the working directory -- the same `shark`
# resolution convention run-lifecycle.sh's own run_command() uses
# (SHARK_BIN env var, default "shark") -- and writes its stdout verbatim to
# --output. `shark history` auto-detects entity type from the key format
# (task/feature/epic/bug/change/tech-debt), so this script adds no
# per-family branch (REQ-NF-006).
#
# Both operator drivers invoke this once per (scenario, rep)/gate pair,
# after a successful run-lifecycle.sh dispatch and before retain_pair, and
# pass its output as retain_pair's entity_history_json source (never the
# empty string retain_pair used to hardcode).
#
# TD-077 sibling-path override precedent (matches RUN_LIFECYCLE_BIN /
# EVALUATE_LIFECYCLE_BIN / PILOT_LEDGER_BIN): both drivers resolve this
# binary via ENTITY_HISTORY_EXPORT_BIN (default: this file, sibling to
# run-lifecycle.sh/evaluate-lifecycle.sh), so a test can substitute a stub
# that writes real-but-synthetic history content without needing a real
# shark project at scratch_root.
#
# Exit status: 0 with --output written on success. Nonzero and --output NOT
# written on any failure (shark binary not found, project root not
# resolvable at scratch-root, command failure, non-JSON output) -- the
# caller MUST treat this as a dispatch failure (record_invalid). retain_pair
# never fabricates a placeholder for a missing entity-history.json source
# either (UAT-R3-01): a failure here refuses the whole pair.
set -euo pipefail

usage() {
	echo "usage: export-entity-history.sh --scratch-root <dir> --root-key <key> --output <path>" >&2
	exit 2
}

scratch_root=""
root_key=""
output=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--scratch-root)
		[[ $# -ge 2 ]] || usage
		scratch_root="$2"
		shift 2
		;;
	--root-key)
		[[ $# -ge 2 ]] || usage
		root_key="$2"
		shift 2
		;;
	--output)
		[[ $# -ge 2 ]] || usage
		output="$2"
		shift 2
		;;
	--help)
		usage
		;;
	*)
		usage
		;;
	esac
done

[[ -n "$scratch_root" && -n "$root_key" && -n "$output" ]] || usage
[[ -d "$scratch_root" ]] || {
	echo "export-entity-history: scratch root not found: $scratch_root" >&2
	exit 1
}

SHARK_BIN_RESOLVED="${SHARK_BIN:-shark}"
command -v "$SHARK_BIN_RESOLVED" >/dev/null 2>&1 || {
	echo "export-entity-history: shark executable not found on PATH: $SHARK_BIN_RESOLVED" >&2
	exit 1
}

TMP_OUT="$(mktemp)"
TMP_ERR="$(mktemp)"
trap 'rm -f "$TMP_OUT" "$TMP_ERR"' EXIT

run_rc=0
(cd "$scratch_root" && "$SHARK_BIN_RESOLVED" history "$root_key" --json) >"$TMP_OUT" 2>"$TMP_ERR" || run_rc=$?
if [[ "$run_rc" -ne 0 ]]; then
	echo "export-entity-history: shark history $root_key --json failed in $scratch_root (exit $run_rc): $(cat "$TMP_ERR")" >&2
	exit 1
fi

python3 -c 'import json,sys; json.load(open(sys.argv[1], encoding="utf-8"))' "$TMP_OUT" 2>/dev/null || {
	echo "export-entity-history: shark history $root_key --json returned non-JSON output" >&2
	exit 1
}

mkdir -p "$(dirname "$output")"
cp "$TMP_OUT" "$output"
