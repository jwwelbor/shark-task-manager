#!/usr/bin/env bash
# TC-037 (test-plan.md AC test matrix; T-E40-F05-015 task spec Test Cases).
#
# Exercises AC-015 (REQ-NF-001/REQ-NF-002): repo root `make fmt && make lint
# && make test` is green, and `go list ./...` lists no fixture or
# evaluator-only package, in BOTH submodule states -- with `bench/fixture-py`
# and `bench/fixture-repo` populated (this checkout's own live state) and
# with them absent entirely (CI-like, matching `actions/checkout@v4`'s
# default of never initializing a submodule).
#
# The "populated" state runs directly against this live checkout -- exactly
# the quality-gate command this whole feature is required to keep green.
# The "unpopulated" state is built from an ISOLATED rsync copy of the whole
# working tree (including uncommitted/untracked files, so in-flight feature
# work is actually exercised, not a stale committed-only snapshot) with the
# two submodule directories excluded from the copy entirely -- never by
# `git submodule deinit`/removing content from the live checkout itself,
# which would mutate shared state this checkout may not be the sole writer
# of (this repo's primary checkout is sometimes shared by concurrent
# sessions on the same path).
#
# Both states exist as separate Go modules from the top-level module
# (`bench/fixture-repo/go.mod`, `bench/scripts/testenum/go.mod`) or are not
# Go at all (`bench/fixture-py`), so `go list ./...`'s module boundary
# already excludes them structurally -- this test asserts that fact
# explicitly rather than merely trusting it holds in both filesystem states.
#
# Caller-Path Contract (test-plan.md TC-037): real `go`, `gofmt`,
# `golangci-lint` subprocesses via the real Makefile targets, run against
# the full repository, never a hand-picked package subset.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

fail() {
	echo "TC-037 FAIL: $1" >&2
	exit 1
}

[[ -f "$REPO_ROOT/Makefile" ]] || fail "Makefile not found at repo root: $REPO_ROOT"
command -v rsync >/dev/null 2>&1 || fail "rsync not found on PATH"
command -v go >/dev/null 2>&1 || fail "go not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

FORBIDDEN_GO_LIST_PATTERN='bench/(fixture-py|fixture-repo|scenarios)'

# assert_hygiene <label> <root_dir>
assert_hygiene() {
	local label="$1" root_dir="$2"
	local make_out="$WORKDIR/${label}-make.out"
	local make_code=0

	echo "TC-037: $label - make fmt && make lint && make test"
	set +e
	(cd "$root_dir" && make fmt && make lint && make test) >"$make_out" 2>&1
	make_code=$?
	set -e
	[[ "$make_code" -eq 0 ]] || fail "$label: 'make fmt && make lint && make test' exited $make_code; tail of output: $(tail -60 "$make_out")"
	echo "TC-037: $label - make fmt/lint/test green"

	local go_list_out="$WORKDIR/${label}-go-list.out"
	(cd "$root_dir" && go list ./...) >"$go_list_out" 2>&1 || fail "$label: 'go list ./...' failed: $(cat "$go_list_out")"
	if grep -qE "$FORBIDDEN_GO_LIST_PATTERN" "$go_list_out"; then
		fail "$label: 'go list ./...' lists a fixture/scenario package it must not: $(grep -E "$FORBIDDEN_GO_LIST_PATTERN" "$go_list_out")"
	fi
	echo "TC-037: $label - go list ./... lists no fixture or scenario package ($(grep -c . "$go_list_out") package(s) total)"
}

echo "TC-037: state 1 - submodules populated (this checkout's live state)"
[[ -e "$REPO_ROOT/bench/fixture-py/.git" ]] || fail "bench/fixture-py submodule not initialized in this checkout; run 'git submodule update --init' before running TC-037"
[[ -e "$REPO_ROOT/bench/fixture-repo/.git" ]] || fail "bench/fixture-repo submodule not initialized in this checkout; run 'git submodule update --init' before running TC-037"
assert_hygiene "populated" "$REPO_ROOT"

echo "TC-037: state 2 - submodules unpopulated (CI-like, actions/checkout@v4 default)"
UNPOPULATED_COPY="$WORKDIR/unpopulated-copy"
mkdir -p "$UNPOPULATED_COPY"
rsync -a \
	--exclude='/bench/fixture-py/' \
	--exclude='/bench/fixture-repo/' \
	"$REPO_ROOT/" "$UNPOPULATED_COPY/"

[[ ! -e "$UNPOPULATED_COPY/bench/fixture-py/.git" ]] || fail "isolated copy still carries bench/fixture-py/.git -- rsync exclude did not take effect"
[[ ! -e "$UNPOPULATED_COPY/bench/fixture-repo/.git" ]] || fail "isolated copy still carries bench/fixture-repo/.git -- rsync exclude did not take effect"

assert_hygiene "unpopulated" "$UNPOPULATED_COPY"

echo "TC-037: PASS"
