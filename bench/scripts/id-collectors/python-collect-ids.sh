#!/usr/bin/env bash
# python-collect-ids.sh --checkout <dir> --file <path>
#
# F06-owned test-identity collector for the "python" I-04 adapter
# (T-E40-F06-003 round-3 code-review rework). Lives outside bench/adapters/**
# on purpose: REQ-NF-006 names that tree part of I-04's frozen, byte-
# unchanged surface (round-3 code-review finding F.5 -- a prior fix edited
# bench/adapters/python/adapter.sh to add a "collect-ids" capability there,
# which this rework reverts). verify-evidence-roots.sh resolves a package's
# declared adapter.name to THIS script via id-collectors/registry.yaml, a
# registry this feature owns -- never a raw path join of the untrusted,
# candidate-controlled adapter.name string.
#
# Enumerates, via pytest's own --collect-only machinery (module-level code
# runs, but no test function's body is ever called -- REQ-F-010's "a test
# identity those files define... without running them", independently
# confirmed empirically while authoring this script: a synthetic probe test
# whose body writes a marker file and asserts False is collected with the
# marker file never created), every test node id <file>::<name> the given
# --file defines, plus -- for a @pytest.mark.parametrize-decorated test --
# the BASE identity with the trailing [param] suffix stripped. That suffix
# is a pytest-RUNTIME-generated string ("test_foo[1]") that never appears as
# literal source text in the file itself ("def test_foo(n): ..."), so
# without the base identity a renamed, stem-stripped copy of a parametrized
# oracle test would defeat every signal the caller has (T-E40-F06-003
# round-3 code-review finding F.2 -- the exact renamed-and-stripped attack
# this feature exists to close).
#
# -p no:cacheprovider and PYTHONDONTWRITEBYTECODE=1 keep this read-only
# against --checkout (no .pytest_cache/__pycache__ written) so a generic
# isolation guard can call this against a root it is about to scan for leaks
# without poisoning that same root. -p no:warnings suppresses pytest's own
# warnings-summary section entirely, rather than trying to parse spurious
# "::"-containing summary/echoed-source lines back out of it afterward
# (T-E40-F06-003 round-3 code-review finding F.8 -- a prior version's naive
# "any line containing '::' is a node id" parse treated a warnings-summary
# line, and even the literal echoed `warnings.warn(...)` source line, as a
# spurious derived identity). The line-shape filter below (first segment
# ends in .py, no segment contains whitespace) is a second, independent
# defense against the same class of false positive, in case some other
# pytest plugin ever emits "::"-bearing prose to stdout during collection.
#
# Emits exactly one JSON document to stdout: {"ids": [{"id": "<node id>",
# "name": "<derived name>"}, ...]}. Every "name" is guaranteed a non-empty
# string (T-E40-F06-003 round-3 code-review finding F.7 -- an empty or
# non-string derived name would let the caller compile a search pattern that
# zero-width-matches almost any content, a false isolation_violation on
# nearly any file in an otherwise-clean root). Exit 0 on success (including
# "collection ran, zero tests found" -- pytest's own exit 5), non-zero with
# a message on stderr if collection itself could not run at all (e.g. an
# import error in the file under test).
set -euo pipefail

usage() {
	echo "usage: python-collect-ids.sh --checkout <dir> --file <path>" >&2
	exit 2
}

CHECKOUT=""
FILE=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--checkout)
		CHECKOUT="${2:-}"
		shift 2
		;;
	--file)
		FILE="${2:-}"
		shift 2
		;;
	*)
		echo "python-collect-ids: unknown argument: $1" >&2
		usage
		;;
	esac
done

[[ -n "$CHECKOUT" ]] || usage
[[ -d "$CHECKOUT" ]] || {
	echo "python-collect-ids: --checkout dir not found: $CHECKOUT" >&2
	exit 2
}
[[ -n "$FILE" ]] || usage
[[ -f "$FILE" ]] || {
	echo "python-collect-ids: --file source not found: $FILE" >&2
	exit 2
}
command -v python3 >/dev/null 2>&1 || {
	echo "python-collect-ids: python3 not found on PATH" >&2
	exit 2
}
python3 -m pytest --version >/dev/null 2>&1 || {
	echo "python-collect-ids: pytest not runnable on PATH" >&2
	exit 2
}

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

collect_out="$WORKDIR/collect.out"
collect_err="$WORKDIR/collect.err"
collect_exit=0
(cd "$CHECKOUT" && PYTHONDONTWRITEBYTECODE=1 python3 -m pytest --collect-only -q -p no:cacheprovider -p no:warnings "$FILE") \
	>"$collect_out" 2>"$collect_err" || collect_exit=$?

# Collection exits 0 (tests found) or 5 ("no tests collected" -- a
# legitimate empty result). Anything else means collection itself could not
# complete (e.g. an import error in the file under test) -- the toolchain
# failed, not "zero ids".
if [[ "$collect_exit" -ne 0 && "$collect_exit" -ne 5 ]]; then
	echo "python-collect-ids: collect-only could not run (exit $collect_exit): $(cat "$collect_err") $(cat "$collect_out")" >&2
	exit 1
fi

python3 - "$collect_out" <<'PYEOF'
import json
import sys

out_path = sys.argv[1]
with open(out_path) as f:
    lines = f.read().splitlines()

ids = []
seen = set()


def emit(node_id, name):
    if not isinstance(name, str) or not name:
        return
    key = (node_id, name)
    if key in seen:
        return
    seen.add(key)
    ids.append({"id": node_id, "name": name})


for line in lines:
    line = line.strip()
    if not line or "::" not in line:
        continue
    segments = line.split("::")
    # A real pytest --collect-only -q node-id line's first segment is the
    # collected file's own path (always ends in .py) and no segment
    # contains whitespace -- rules out prose/summary lines that merely
    # happen to contain "::" (finding F.8), independent of -p no:warnings
    # above.
    if not segments[0].endswith(".py"):
        continue
    if any(" " in segment for segment in segments):
        continue
    name = segments[-1]
    emit(line, name)
    # Base identity: strip a trailing pytest-generated [param] suffix
    # (finding F.2) so a parametrized test's identity is still findable as
    # literal source text -- "def test_foo(n): ..." never contains the
    # runtime string "test_foo[1]".
    base_name = name.split("[", 1)[0]
    if base_name and base_name != name:
        emit(line, base_name)

print(json.dumps({"ids": ids}))
PYEOF
