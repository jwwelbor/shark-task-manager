#!/usr/bin/env bash
# bench/adapters/python/adapter.sh <capability> [args...]
#
# The I-04 execution adapter for bench/fixture-py (REQ-F-006, ADR-F05-03).
# This file and adapter.yaml are the ONLY place in bench/ that knows pytest,
# ruff, or black exist (REQ-F-007) -- no generic scenario, evidence, or
# admission script may branch on fixture language; they call this contract
# instead.
#
# Closed set of six capabilities, one function each below:
#   identity        --checkout <dir>
#   inject-tests    --checkout <dir> --files <path>...
#   test            --checkout <dir> [--include <path>...] [--exclude-id <id>...] [--only-id <id>...]
#   lint            --checkout <dir>
#   build           --checkout <dir>
#   format-check    --checkout <dir>
#
# Each capability writes exactly one JSON document to stdout, per the shape
# fixed in spec.md's "Adapter capability contract" table. Exit status 0 means
# "the capability ran" -- even when its *subject* is red (a failing test, a
# lint issue, an unformatted file): that outcome is reported IN the JSON, not
# via a non-zero exit. A non-zero exit means the toolchain itself could not
# be invoked (missing binary, crash, bad arguments) -- AC-T3.
#
# `test`'s ids are already normalized to `<module path>::<test name>` (dotted
# Python module path, matching pytest's own JUnit `classname`), so no
# consumer performs language-aware id parsing (REQ-F-007, AC-011).
# `--include`/`--exclude-id` carry a p2p_selection; `--only-id` names a
# predicate's own test ids. Both are translated here, and only here, into
# real pytest node ids / paths.
#
# Assumes the checkout's Python environment already has pytest, ruff, and
# black installed and on PATH (this adapter provisions nothing, mirroring
# the Go adapter's assumption that `go`/`golangci-lint` are already present).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
	echo "usage: adapter.sh <identity|inject-tests|test|lint|build|format-check> --checkout <dir> [...]" >&2
}

fail() {
	echo "adapter.sh: $1" >&2
	exit 1
}

require_tool() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 not found on PATH"
}

resolve_nodeids() {
	# Reconstructs real pytest node ids from normalized
	# "<dotted module/class path>::<test name>" ids (the inverse of the
	# mapping cmd_test applies when reading JUnit's classname/name
	# attributes). A blind dot-to-slash rewrite of the whole prefix is
	# wrong for a class-based test: JUnit's classname for a test inside a
	# class is itself dotted (e.g. "tests.test_x.TestGroup"), where only
	# "tests.test_x" is the file. This walks the dotted segments from
	# longest to shortest, taking the longest prefix that names a real
	# <prefix>.py file under $checkout as the file path and rejoining any
	# remaining segments as pytest's own "::"-nested class name(s).
	local checkout="$1"
	shift
	local out="$WORKDIR/resolve_out.$$.$RANDOM"
	local err="$WORKDIR/resolve_err.$$.$RANDOM"
	python3 - "$checkout" "$@" >"$out" 2>"$err" <<'PYEOF'
import os
import sys

checkout = sys.argv[1]
ids = sys.argv[2:]

for norm_id in ids:
    module_part, test_part = norm_id.split("::", 1)
    segments = module_part.split(".")
    resolved = None
    for split_at in range(len(segments), 0, -1):
        candidate_file = os.path.join(*segments[:split_at]) + ".py"
        if os.path.isfile(os.path.join(checkout, candidate_file)):
            resolved = "::".join([candidate_file] + segments[split_at:] + [test_part])
            break
    if resolved is None:
        sys.stderr.write("could not resolve id to a file under checkout: %s\n" % norm_id)
        sys.exit(1)
    print(resolved)
PYEOF
	local rc=$?
	if [[ $rc -ne 0 ]]; then
		cat "$err" >&2
		return 1
	fi
	cat "$out"
}

[[ $# -ge 1 ]] || {
	usage
	exit 1
}
CAPABILITY="$1"
shift

case "$CAPABILITY" in
identity | inject-tests | test | lint | build | format-check) ;;
*)
	fail "unknown capability '$CAPABILITY' (closed set: identity, inject-tests, test, lint, build, format-check)"
	;;
esac

CHECKOUT=""
FILES=()
INCLUDE=()
EXCLUDE_IDS=()
ONLY_IDS=()

while [[ $# -gt 0 ]]; do
	case "$1" in
	--checkout)
		CHECKOUT="${2:-}"
		shift 2
		;;
	--files)
		shift
		while [[ $# -gt 0 && "$1" != --* ]]; do
			FILES+=("$1")
			shift
		done
		;;
	--include)
		shift
		while [[ $# -gt 0 && "$1" != --* ]]; do
			INCLUDE+=("$1")
			shift
		done
		;;
	--exclude-id)
		shift
		while [[ $# -gt 0 && "$1" != --* ]]; do
			EXCLUDE_IDS+=("$1")
			shift
		done
		;;
	--only-id)
		shift
		while [[ $# -gt 0 && "$1" != --* ]]; do
			ONLY_IDS+=("$1")
			shift
		done
		;;
	*)
		fail "unrecognized argument: $1"
		;;
	esac
done

[[ -n "$CHECKOUT" ]] || fail "--checkout is required"
[[ -d "$CHECKOUT" ]] || fail "checkout dir not found: $CHECKOUT"
CHECKOUT="$(cd "$CHECKOUT" && pwd)"

require_tool python3

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

cmd_identity() {
	local python_version_raw python_version
	python_version_raw="$(python3 --version 2>&1)"
	python_version="$(printf '%s' "$python_version_raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
	[[ -n "$python_version" ]] || fail "could not parse python3 version from: $python_version_raw"

	local pytest_version_raw pytest_version
	pytest_version_raw="$(python3 -m pytest --version 2>&1)" || fail "pytest not runnable: $pytest_version_raw"
	pytest_version="$(printf '%s' "$pytest_version_raw" | head -n1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
	[[ -n "$pytest_version" ]] || fail "could not parse pytest version from: $pytest_version_raw"

	require_tool ruff
	local ruff_version_raw ruff_version
	ruff_version_raw="$(ruff --version 2>&1)"
	ruff_version="$(printf '%s' "$ruff_version_raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
	[[ -n "$ruff_version" ]] || fail "could not parse ruff version from: $ruff_version_raw"

	require_tool black
	local black_version_raw black_version
	black_version_raw="$(black --version 2>&1 | head -n1)"
	black_version="$(printf '%s' "$black_version_raw" | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -n1)"
	[[ -n "$black_version" ]] || fail "could not parse black version from: $black_version_raw"

	local pyproject_path="$CHECKOUT/pyproject.toml"
	[[ -f "$pyproject_path" ]] || fail "pyproject.toml not found: $pyproject_path"
	local pyproject_sha256
	pyproject_sha256="$(sha256sum "$pyproject_path" | awk '{print $1}')"

	local adapter_name adapter_version
	adapter_name="$(grep -E '^name:' "$SCRIPT_DIR/adapter.yaml" | sed -E 's/^name:[[:space:]]*"?([^"[:space:]]+)"?.*/\1/')"
	adapter_version="$(grep -E '^version:' "$SCRIPT_DIR/adapter.yaml" | sed -E 's/^version:[[:space:]]*"?([^"[:space:]]+)"?.*/\1/')"

	python3 - "$adapter_name" "$adapter_version" "$python_version" "$pytest_version" "$ruff_version" "$black_version" "$pyproject_sha256" <<'PYEOF'
import json
import sys

adapter_name, adapter_version, python_version, pytest_version, ruff_version, black_version, pyproject_sha256 = sys.argv[1:8]

toolchain_identity = [
    {"key": "python_version", "value": python_version},
    {"key": "pytest_version", "value": pytest_version},
    {"key": "ruff_version", "value": ruff_version},
    {"key": "black_version", "value": black_version},
    {"key": "pyproject_sha256", "value": pyproject_sha256},
]

print(json.dumps({"adapter": adapter_name, "version": adapter_version, "toolchain_identity": toolchain_identity}))
PYEOF
}

cmd_inject_tests() {
	[[ ${#FILES[@]} -gt 0 ]] || fail "inject-tests requires at least one --files path"

	local dest_dir="$CHECKOUT/tests"
	mkdir -p "$dest_dir"

	local pair_args=()
	local src base dest
	for src in "${FILES[@]}"; do
		[[ -f "$src" ]] || fail "inject-tests source file not found: $src"
		base="$(basename "$src")"
		dest="$dest_dir/$base"
		cp "$src" "$dest"
		pair_args+=("$src" "tests/$base")
	done

	python3 - "${pair_args[@]}" <<'PYEOF'
import json
import sys

args = sys.argv[1:]
pairs = list(zip(args[0::2], args[1::2]))
injected = [{"source": s, "destination": d} for s, d in pairs]
print(json.dumps({"injected": injected}))
PYEOF
}

cmd_test() {
	python3 -m pytest --version >/dev/null 2>&1 || fail "pytest not runnable"

	local pos_args=()
	local extra_args=()

	if [[ ${#ONLY_IDS[@]} -gt 0 ]]; then
		local resolved_only
		resolved_only="$(resolve_nodeids "$CHECKOUT" "${ONLY_IDS[@]}")" || exit 1
		mapfile -t pos_args <<<"$resolved_only"
	else
		if [[ ${#INCLUDE[@]} -gt 0 ]]; then
			pos_args=("${INCLUDE[@]}")
		fi
		if [[ ${#EXCLUDE_IDS[@]} -gt 0 ]]; then
			# NOTE: unlike --only-id above, an --exclude-id that resolves to
			# an existing file but a test name pytest never collects is not
			# separately validated here -- `--deselect` silently matches
			# nothing rather than erroring, so a stale/typo'd exclude id
			# widens the effective P2P set instead of naming the problem.
			# resolve_nodeids still fails loudly when the id's *file* can't
			# be found on disk.
			local resolved_excludes nid
			resolved_excludes="$(resolve_nodeids "$CHECKOUT" "${EXCLUDE_IDS[@]}")" || exit 1
			while IFS= read -r nid; do
				[[ -n "$nid" ]] || continue
				extra_args+=("--deselect" "$nid")
			done <<<"$resolved_excludes"
		fi
	fi

	local junit_out="$WORKDIR/junit.xml"
	local run_out="$WORKDIR/pytest.out"
	local run_err="$WORKDIR/pytest.err"
	local pytest_args=(-q "--junit-xml=$junit_out")
	[[ ${#pos_args[@]} -eq 0 ]] || pytest_args+=("${pos_args[@]}")
	[[ ${#extra_args[@]} -eq 0 ]] || pytest_args+=("${extra_args[@]}")

	local pytest_exit=0
	(cd "$CHECKOUT" && python3 -m pytest "${pytest_args[@]}") >"$run_out" 2>"$run_err" || pytest_exit=$?

	# pytest exit 0 = all passed, 1 = ran fine but some failed -- both are a
	# successful capability invocation (AC-T3). Exit 5 ("no tests ran") means
	# the resolved selection (an --include path with nothing under it, or an
	# --exclude-id set that deselected everything) legitimately matched zero
	# tests -- a valid empty {"entries": []} result, not a toolchain failure;
	# AC-007 check (b) needs to see this as "empty set" so admit-scenario.sh
	# can reject it by name, not have it swallowed here as "could not run."
	# Anything else (2: interrupted, 3: internal error, 4: usage error, e.g.
	# an --include/--only-id path that does not exist at all) means the
	# toolchain itself could not run the requested selection.
	if [[ "$pytest_exit" -gt 1 && "$pytest_exit" -ne 5 ]]; then
		fail "pytest could not run (exit $pytest_exit): $(cat "$run_err") $(cat "$run_out")"
	fi
	[[ -f "$junit_out" ]] || fail "pytest exited $pytest_exit but produced no junit report"

	python3 - "$junit_out" <<'PYEOF'
import json
import sys
import xml.etree.ElementTree as ET

junit_path = sys.argv[1]
root = ET.parse(junit_path).getroot()

entries = []
for testcase in root.iter("testcase"):
    node_id = f"{testcase.get('classname')}::{testcase.get('name')}"
    if testcase.find("failure") is not None or testcase.find("error") is not None:
        outcome = "fail"
    elif testcase.find("skipped") is not None:
        outcome = "skip"
    else:
        outcome = "pass"
    entries.append({"id": node_id, "outcome": outcome})

entries.sort(key=lambda e: e["id"])
print(json.dumps({"entries": entries}))
PYEOF
}

cmd_lint() {
	require_tool ruff

	local raw_out="$WORKDIR/ruff.json"
	local raw_err="$WORKDIR/ruff.err"
	local ruff_exit=0
	(cd "$CHECKOUT" && ruff check --output-format=json .) >"$raw_out" 2>"$raw_err" || ruff_exit=$?

	# ruff exits 0 (clean) or 1 (issues found) on a normal run -- both are a
	# successful invocation; anything else means ruff itself could not run.
	if [[ "$ruff_exit" -gt 1 ]]; then
		fail "ruff check could not run (exit $ruff_exit): $(cat "$raw_err")"
	fi

	python3 - "$raw_out" "$CHECKOUT" <<'PYEOF'
import json
import os
import sys

raw_path, checkout = sys.argv[1:3]

with open(raw_path) as f:
    text = f.read().strip()
data = json.loads(text) if text else []

issues = []
for item in data:
    try:
        rel = os.path.relpath(item["filename"], checkout)
    except ValueError:
        rel = item["filename"]
    issues.append({"rule": item["code"], "file": rel, "text": item["message"]})

# A stable multiset ordering: identity excludes line/column so it is stable
# under position shifts (ADR-F01-03 precedent), and repeated runs over an
# unchanged checkout must be byte-identical (REQ-NF-004).
issues.sort(key=lambda i: (i["rule"], i["file"], i["text"]))
print(json.dumps({"issues": issues}))
PYEOF
}

cmd_build() {
	local compile_out="$WORKDIR/compileall.out"
	local compile_exit=0
	(cd "$CHECKOUT" && python3 -m compileall -q -x '(\.git|\.egg-info|\.pytest_cache|__pycache__)' .) \
		>"$compile_out" 2>&1 || compile_exit=$?

	local import_out="$WORKDIR/import.out"
	local import_exit=0
	(cd "$CHECKOUT" && PYTHONPATH="$CHECKOUT" python3 -c 'import taskmanager') \
		>"$import_out" 2>&1 || import_exit=$?

	python3 - "$compile_exit" "$compile_out" "$import_exit" "$import_out" <<'PYEOF'
import json
import sys

compile_exit, compile_out_path, import_exit, import_out_path = sys.argv[1:5]
compile_exit = int(compile_exit)
import_exit = int(import_exit)

diagnostics = []
if compile_exit != 0:
    with open(compile_out_path) as f:
        text = f.read().strip()
    if text:
        diagnostics.append(text)
if import_exit != 0:
    with open(import_out_path) as f:
        text = f.read().strip()
    if text:
        diagnostics.append(text)

print(json.dumps({"ok": compile_exit == 0 and import_exit == 0, "diagnostics": diagnostics}))
PYEOF
}

cmd_format_check() {
	require_tool black

	local err_out="$WORKDIR/black.err"
	local black_exit=0
	(cd "$CHECKOUT" && black --check .) >/dev/null 2>"$err_out" || black_exit=$?

	# black --check exits 0 (nothing to reformat) or 1 (some files would
	# change) on a normal run; 123 signals an internal black error.
	if [[ "$black_exit" -gt 1 ]]; then
		fail "black --check could not run (exit $black_exit): $(cat "$err_out")"
	fi

	python3 - "$err_out" "$CHECKOUT" <<'PYEOF'
import json
import os
import re
import sys

err_path, checkout = sys.argv[1:3]

with open(err_path) as f:
    text = f.read()

offending = []
for line in text.splitlines():
    m = re.match(r"^would reformat (.+)$", line)
    if m:
        try:
            rel = os.path.relpath(m.group(1), checkout)
        except ValueError:
            rel = m.group(1)
        offending.append(rel)

offending.sort()
print(json.dumps({"ok": len(offending) == 0, "offending_files": offending}))
PYEOF
}

case "$CAPABILITY" in
identity) cmd_identity ;;
inject-tests) cmd_inject_tests ;;
test) cmd_test ;;
lint) cmd_lint ;;
build) cmd_build ;;
format-check) cmd_format_check ;;
esac
