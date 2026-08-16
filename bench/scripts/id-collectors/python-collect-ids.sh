#!/usr/bin/env bash
# python-collect-ids.sh --checkout <dir> --file <path>
#
# F06-owned test-identity collector for the "python" I-04 adapter
# (T-E40-F06-003 round-4 code-review rework). Lives outside bench/adapters/**
# on purpose: REQ-NF-006 names that tree part of I-04's frozen, byte-
# unchanged surface (round-3 code-review finding F.5 -- a prior fix edited
# bench/adapters/python/adapter.sh to add a "collect-ids" capability there,
# which an earlier rework reverted). verify-evidence-roots.sh resolves a
# package's declared adapter.name to THIS script via id-collectors/
# registry.yaml, a registry this feature owns -- never a raw path join of
# the untrusted, candidate-controlled adapter.name string.
#
# STRUCTURAL identity derivation, not text-parsing (round-4 code-review
# mandate). The previous implementation reconstructed a test's identity by
# string-splitting pytest's own human-readable `--collect-only -q` stdout on
# "::". Two consecutive rounds each closed one instance of that text
# parser's fragility while leaving (round 3) or introducing (round 4) an
# adjacent one:
#   - round 3: the auto-generated "[param]" node-id suffix never appears as
#     literal source text ("def test_foo(n): ..." never contains the
#     runtime string "test_foo[1]") -- fixed by stripping the suffix.
#   - round 4: a `@pytest.mark.parametrize(..., ids=[...])` CUSTOM id
#     containing a space caused the line-shape filter added THAT SAME round
#     (to suppress an unrelated warnings-summary false positive) to
#     silently discard the entire line -- no identity derived at all; a
#     custom id containing "::" caused the "::"-split itself to take the
#     wrong segment as the name, emitting a garbage identity.
# Both failure modes share one root cause: pytest does not escape or bound
# `ids=` values in its free-form text reporter output, so no string-split/
# filter heuristic keyed to the SHAPE of a normal node id can be made robust
# against arbitrary custom-id content.
#
# This version never reads that text output at all. It runs a small,
# in-process pytest plugin object (passed to pytest.main(..., plugins=[...]),
# no conftest.py) implementing pytest_collection_modifyitems -- a hook that
# fires after collection completes but strictly before any test's body runs
# (--collect-only stops the session before the run phase begins, so no held-
# back oracle test is ever executed; independently confirmed empirically
# while authoring this script: a synthetic probe test whose body writes a
# marker file and asserts False is collected with the marker file never
# created). For each collected item the plugin reads pytest's OWN,
# already-computed identity fields directly -- never re-derived here by
# splitting text:
#   - item.nodeid        -- the full "<file>::<test>[<params>]" id pytest
#                            itself computes.
#   - item.originalname   -- the literal `def` function name pytest itself
#                            computes (_pytest/python.py's Function.__init__:
#                            `self.originalname = originalname or name`),
#                            GUARANTEED to exclude any parametrize suffix --
#                            auto-generated OR custom `ids=`-supplied -- and
#                            to never contain custom-id content, because
#                            pytest derives it from the `def` statement, not
#                            from any rendered/escaped text form. This is the
#                            field the round-4 report specifically
#                            recommends and the one this script searches
#                            against; it eliminates the entire class of
#                            "pytest's text output has an ambiguous/lossy
#                            shape for X" bugs the last two rounds each hit a
#                            different instance of, because both fields are
#                            pytest's own internal, already-parsed values --
#                            there is no free-form text to misinterpret.
#
# pytest's own collection-phase console output is captured and discarded (it
# is never parsed for anything); only this script's own final JSON reaches
# real stdout.
#
# -p no:cacheprovider and PYTHONDONTWRITEBYTECODE=1 keep this read-only
# against --checkout (no .pytest_cache/__pycache__ written) so a generic
# isolation guard can call this against a root it is about to scan for leaks
# without poisoning that same root. -p no:warnings is no longer load-bearing
# for identity derivation (there is no warnings-summary text to misparse --
# nothing is parsed at all), but is kept to avoid unrelated collection
# noise on stderr.
#
# Emits exactly one JSON document to stdout: {"ids": [{"id": "<node id>",
# "name": "<originalname>"}, ...]}. Every "name" is guaranteed a non-empty
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
(cd "$CHECKOUT" && PYTHONDONTWRITEBYTECODE=1 python3 - "$FILE" <<'PYEOF') >"$collect_out" 2>"$collect_err" || collect_exit=$?
import contextlib
import io
import json
import sys

import pytest

file_arg = sys.argv[1]


class _IdentityCollector:
    """A plain plugin OBJECT (not a conftest.py module) -- pluggy discovers
    its pytest_collection_modifyitems method by name when passed via
    pytest.main(..., plugins=[...]), exactly like a conftest.py's top-level
    functions. This hook fires once, after the collection phase has fully
    computed every item's identity fields but strictly before the (never
    reached, because of --collect-only) test-run phase."""

    def __init__(self):
        self.entries = []

    def pytest_collection_modifyitems(self, session, config, items):
        for item in items:
            # originalname is only set on Function-like items; fall back to
            # item.name (still parametrize-suffix-free for any item type
            # that has no originalname concept at all, e.g. a bare Module).
            name = getattr(item, "originalname", None) or item.name
            self.entries.append({"id": item.nodeid, "name": name})


collector = _IdentityCollector()
captured = io.StringIO()
with contextlib.redirect_stdout(captured), contextlib.redirect_stderr(captured):
    exit_code = pytest.main(
        ["--collect-only", "-p", "no:cacheprovider", "-p", "no:warnings", file_arg],
        plugins=[collector],
    )

# pytest.ExitCode.OK (0) = tests collected; NO_TESTS_COLLECTED (5) = a
# legitimate empty result (e.g. a helper module with no test_* functions) --
# both are a *successful* run of the toolchain, matching this script's
# original exit-code contract. Anything else means collection itself could
# not complete (e.g. an import error in the file under test).
if int(exit_code) not in (0, 5):
    sys.stderr.write(f"collect-only could not run (exit {int(exit_code)}): {captured.getvalue()}\n")
    sys.exit(1)

for entry in collector.entries:
    if not isinstance(entry["name"], str) or not entry["name"]:
        sys.stderr.write(f"collector produced a non-string or empty name for node id {entry['id']!r}\n")
        sys.exit(1)

print(json.dumps({"ids": collector.entries}))
PYEOF

# Collection exits 0 (tests found) or 5 ("no tests collected" -- a
# legitimate empty result). Anything else means collection itself could not
# complete (e.g. an import error in the file under test) -- the toolchain
# failed, not "zero ids". The in-process driver above already applies this
# same rule to pytest's own exit code and always exits 0/1 accordingly, so
# collect_exit here is just that driver's own exit status.
if [[ "$collect_exit" -ne 0 ]]; then
	echo "python-collect-ids: collect-only could not run (exit $collect_exit): $(cat "$collect_err") $(cat "$collect_out")" >&2
	exit 1
fi

cat "$collect_out"
