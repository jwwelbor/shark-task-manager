#!/usr/bin/env bash
# bench/adapters/go/adapter.sh <capability> [args...]
#
# The I-04 execution adapter for bench/fixture-repo (REQ-F-006, ADR-F05-03) --
# the I-01 *compatibility* adapter (research-report.md Decision 3): it wraps
# the existing, unmodified bench/scripts/build-ledgers.sh and the fixture's
# own Makefile rather than introducing new Go-specific tooling. This file and
# adapter.yaml are the ONLY place in bench/ that knows `go test`,
# `golangci-lint`, or `gofmt` exist (REQ-F-007) -- no generic scenario,
# evidence, or admission script may branch on fixture language; they call
# this contract instead.
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
# be invoked (missing binary, crash, bad arguments) -- AC-T4.
#
# `test`/`lint` reshape build-ledgers.sh's tests.json/lint.json rather than
# re-deriving them (AC-T1) -- build-ledgers.sh and diff-ledgers.sh are
# invoked/left byte-unchanged by this adapter (AC-T3); this adapter never
# edits either script or bench/fixture-repo. diff-ledgers.sh's base-vs-post
# diffing is deliberately NOT invoked here: I-04's final predicates read a
# single `test`/`lint` snapshot (REQ-F-010), never a base ledger (ADR-F05-10,
# REQ-F-017) -- that comparison role stays E40-F02's, over I-01's own
# corpus.yaml ledgers.
#
# `test`'s ids are already normalized to `<package import path>::<test
# name>` (AC-T2), matching build-ledgers.sh's own ledger convention. Since
# build-ledgers.sh has no selection arguments (its `<checkout_dir>
# <output_dir>` interface is frozen, same posture as
# checkout-fixture.sh -- ADR-F05-06), `--include`/`--exclude-id`/`--only-id`
# are applied here, AFTER running the full unmodified ledger build, by
# filtering the resulting entries down to the requested selection -- the
# language-specific translation of a p2p_selection into "which of this run's
# results count" lives here and nowhere else (REQ-F-017).
#
# Assumes the checkout's Go environment already has `go`, `golangci-lint`,
# `gofmt`, and `make` installed and on PATH (this adapter provisions nothing,
# mirroring the Python adapter's assumption that pytest/ruff/black are
# already present).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_LEDGERS="$BENCH_DIR/scripts/build-ledgers.sh"

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

write_lines() {
	# Writes each remaining arg as its own line to file $1. Used to hand
	# variable-length bash arrays to the python3 heredocs below without
	# fighting shell word-splitting on positional args.
	local file="$1"
	shift
	: >"$file"
	local x
	for x in "$@"; do
		printf '%s\n' "$x" >>"$file"
	done
}

resolve_import_paths() {
	# Translates one fixture-relative p2p_selection.include path into the
	# Go import path(s) it names -- a directory resolves to itself and
	# everything under it (`./<path>/...`); a file resolves to its
	# containing package. Returns non-zero (no message; the caller names
	# the offending path) if the path does not exist under the checkout.
	local rel="$1"
	rel="${rel#./}"
	local abs="$CHECKOUT/$rel"
	[[ -e "$abs" ]] || return 1
	local pattern
	if [[ -d "$abs" ]]; then
		pattern="./${rel%/}/..."
	else
		local dir
		dir="$(dirname "$rel")"
		pattern="./$dir"
	fi
	(cd "$CHECKOUT" && go list -f '{{.ImportPath}}' "$pattern")
}

find_package_dir() {
	# Locates an existing directory under the checkout whose non-test .go
	# file(s) declare `package <pkg>` -- the language-specific "where does
	# the toolchain discover a test file" rule inject-tests exists to
	# express (a Go test must be colocated with the package it tests;
	# there is no single fixed directory the way pytest's rootdir allows).
	local pkg="$1"
	local f
	while IFS= read -r -d '' f; do
		if grep -m1 -qE "^package[[:space:]]+${pkg}([[:space:]]|\$)" "$f"; then
			dirname "$f"
			return 0
		fi
	done < <(find "$CHECKOUT" -name '*.go' ! -name '*_test.go' -print0 | sort -z)
	return 1
}

run_build_ledgers() {
	# Invokes the unmodified build-ledgers.sh (AC-T3) into $1, surfacing
	# its own diagnostic on failure rather than a bare non-zero exit.
	local out_dir="$1"
	local log_out="$WORKDIR/build-ledgers.out"
	local log_err="$WORKDIR/build-ledgers.err"
	if ! "$BUILD_LEDGERS" "$CHECKOUT" "$out_dir" >"$log_out" 2>"$log_err"; then
		fail "build-ledgers.sh could not run: $(cat "$log_err") $(cat "$log_out")"
	fi
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
	require_tool go
	require_tool golangci-lint

	local golangci_config="$CHECKOUT/.golangci.yml"
	[[ -f "$golangci_config" ]] || fail "fixture .golangci.yml not found: $golangci_config"

	local go_version goos goarch
	go_version="$(go env GOVERSION)"
	goos="$(go env GOOS)"
	goarch="$(go env GOARCH)"

	local golangci_lint_raw golangci_lint_version
	golangci_lint_raw="$(golangci-lint version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)"
	[[ -n "$golangci_lint_raw" ]] || fail "could not parse golangci-lint version from 'golangci-lint version'"
	golangci_lint_version="v${golangci_lint_raw}"

	local golangci_config_sha256
	golangci_config_sha256="$(sha256sum "$golangci_config" | awk '{print $1}')"

	local adapter_name adapter_version
	adapter_name="$(grep -E '^name:' "$SCRIPT_DIR/adapter.yaml" | sed -E 's/^name:[[:space:]]*"?([^"[:space:]]+)"?.*/\1/')"
	adapter_version="$(grep -E '^version:' "$SCRIPT_DIR/adapter.yaml" | sed -E 's/^version:[[:space:]]*"?([^"[:space:]]+)"?.*/\1/')"

	python3 - "$adapter_name" "$adapter_version" "$go_version" "$golangci_lint_version" "$goos" "$goarch" "$golangci_config_sha256" <<'PYEOF'
import json
import sys

adapter_name, adapter_version, go_version, golangci_lint_version, goos, goarch, golangci_config_sha256 = sys.argv[1:8]

toolchain_identity = [
    {"key": "go_version", "value": go_version},
    {"key": "golangci_lint_version", "value": golangci_lint_version},
    {"key": "goos", "value": goos},
    {"key": "goarch", "value": goarch},
    {"key": "golangci_config_sha256", "value": golangci_config_sha256},
]

print(json.dumps({"adapter": adapter_name, "version": adapter_version, "toolchain_identity": toolchain_identity}))
PYEOF
}

cmd_inject_tests() {
	[[ ${#FILES[@]} -gt 0 ]] || fail "inject-tests requires at least one --files path"
	require_tool go

	local pair_args=()
	local src pkg_name dest_dir base dest rel_dest
	for src in "${FILES[@]}"; do
		[[ -f "$src" ]] || fail "inject-tests source file not found: $src"
		pkg_name="$(grep -m1 -E '^package[[:space:]]+[A-Za-z0-9_]+' "$src" | awk '{print $2}')"
		[[ -n "$pkg_name" ]] || fail "inject-tests: could not parse a 'package' declaration from $src"
		dest_dir="$(find_package_dir "$pkg_name")" || fail "inject-tests: no existing package '$pkg_name' found under checkout for $src"
		base="$(basename "$src")"
		dest="$dest_dir/$base"
		cp "$src" "$dest"
		rel_dest="${dest#"$CHECKOUT"/}"
		pair_args+=("$src" "$rel_dest")
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
	require_tool go

	local ledger_dir="$WORKDIR/ledgers"
	mkdir -p "$ledger_dir"
	run_build_ledgers "$ledger_dir"
	local tests_json="$ledger_dir/tests.json"
	[[ -f "$tests_json" ]] || fail "build-ledgers.sh did not produce tests.json"

	# --include carries a p2p_selection (REQ-F-017): resolve each
	# fixture-relative path to the Go import path(s) it names.
	local include_pkgs=()
	if [[ ${#INCLUDE[@]} -gt 0 ]]; then
		local rel resolved p
		for rel in "${INCLUDE[@]}"; do
			resolved="$(resolve_import_paths "$rel")" || fail "--include path not found under checkout: $rel"
			while IFS= read -r p; do
				[[ -n "$p" ]] || continue
				include_pkgs+=("$p")
			done <<<"$resolved"
		done
	fi

	local include_file="$WORKDIR/include_pkgs.txt"
	local exclude_file="$WORKDIR/exclude_ids.txt"
	local only_file="$WORKDIR/only_ids.txt"
	write_lines "$include_file" "${include_pkgs[@]}"
	write_lines "$exclude_file" "${EXCLUDE_IDS[@]}"
	write_lines "$only_file" "${ONLY_IDS[@]}"
	# Distinct from include_pkgs being empty: --include was given but
	# resolved to zero Go packages (e.g. a directory with no .go files)
	# is a legitimate EMPTY p2p_selection, not "no --include given" --
	# REQ-F-012 check (b) must be able to see and reject that empty set
	# by name, the same "empty is a valid result" contract the Python
	# adapter documents for pytest exit 5.
	local include_given=0
	[[ ${#INCLUDE[@]} -gt 0 ]] && include_given=1

	python3 - "$tests_json" "$include_file" "$exclude_file" "$only_file" "$include_given" <<'PYEOF'
import json
import sys

tests_path, include_path, exclude_path, only_path, include_given = sys.argv[1:6]
include_given = include_given == "1"


def read_lines(p):
    with open(p) as f:
        return [line.strip() for line in f if line.strip()]


with open(tests_path) as f:
    ledger = json.load(f)
all_ids = {e["identity"]: e["action"] for e in ledger.get("entries", [])}

only_ids = read_lines(only_path)
if only_ids:
    missing = [i for i in only_ids if i not in all_ids]
    if missing:
        sys.exit("adapter.sh: --only-id names id(s) not present in the test ledger: %s" % missing)
    selected = {i: all_ids[i] for i in only_ids}
else:
    if include_given:
        include_pkgs = set(read_lines(include_path))
        selected = {i: a for i, a in all_ids.items() if i.split("::", 1)[0] in include_pkgs}
    else:
        selected = dict(all_ids)
    exclude_ids = set(read_lines(exclude_path))
    for i in exclude_ids:
        selected.pop(i, None)

entries = [{"id": i, "outcome": selected[i]} for i in sorted(selected)]
print(json.dumps({"entries": entries}))
PYEOF
}

cmd_lint() {
	require_tool golangci-lint

	local ledger_dir="$WORKDIR/ledgers"
	mkdir -p "$ledger_dir"
	run_build_ledgers "$ledger_dir"
	local lint_json="$ledger_dir/lint.json"
	[[ -f "$lint_json" ]] || fail "build-ledgers.sh did not produce lint.json"

	python3 - "$lint_json" <<'PYEOF'
import json
import sys

lint_path = sys.argv[1]

with open(lint_path) as f:
    ledger = json.load(f)

issues = [
    {"rule": e.get("from_linter", ""), "file": e.get("path", ""), "text": e.get("text", "")}
    for e in ledger.get("entries", [])
]
issues.sort(key=lambda i: (i["rule"], i["file"], i["text"]))
print(json.dumps({"issues": issues}))
PYEOF
}

cmd_build() {
	require_tool make

	local out="$WORKDIR/make-build.out"
	local exit_code=0
	(cd "$CHECKOUT" && make build) >"$out" 2>&1 || exit_code=$?

	python3 - "$exit_code" "$out" <<'PYEOF'
import json
import sys

exit_code = int(sys.argv[1])
with open(sys.argv[2]) as f:
    text = f.read().strip()

diagnostics = [text] if (exit_code != 0 and text) else []
print(json.dumps({"ok": exit_code == 0, "diagnostics": diagnostics}))
PYEOF
}

cmd_format_check() {
	# The fixture Makefile has no "format-check" target -- its only
	# formatting target is `fmt` (`go fmt ./...`), which REWRITES files
	# and so cannot be reused for a non-destructive check without
	# mutating bench/fixture-repo (REQ-F-005 requires it stay unchanged).
	# `gofmt -l` is go fmt's own underlying, non-destructive "list files
	# that would be reformatted" primitive, mirroring the Python
	# adapter's `black --check` shape exactly.
	require_tool gofmt

	local out="$WORKDIR/gofmt.out"
	local err="$WORKDIR/gofmt.err"
	local exit_code=0
	(cd "$CHECKOUT" && gofmt -l .) >"$out" 2>"$err" || exit_code=$?

	# gofmt -l exits 0 whether or not it lists offending files; a non-zero
	# exit means gofmt itself could not run.
	if [[ "$exit_code" -ne 0 ]]; then
		fail "gofmt -l could not run (exit $exit_code): $(cat "$err")"
	fi

	python3 - "$out" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    offending = sorted(line.strip().removeprefix("./") for line in f if line.strip())

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
