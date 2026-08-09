#!/usr/bin/env bash
# TC-011 (test-plan.md AC test matrix; T-E40-F01-009 task spec Test Cases).
#
# Exercises AC-010 / AC-T3 (REQ-F-010): `diff-ledgers.sh --toolchain-guard
# --base=<ledger.json>` -- the sole named toolchain-guard entrypoint -- fails
# independently on each of the four recorded-vs-observed mismatch axes (Go
# version, golangci-lint version, GOOS/GOARCH, .golangci.yml content hash),
# naming the specific mismatch on stderr, and never runs a diff mode
# afterward.
#
# Caller-Path Contract (test-plan.md TC-011): real subprocess invocation of
# whatever `go`/`golangci-lint` binary PATH resolves to -- the version
# string is never mocked in code. The Go-version and golangci-lint-version
# axes use a real, executable stub binary placed first on PATH that
# forwards every call to the real toolchain EXCEPT the one query this test
# wants to lie about (`go env GOVERSION` / `golangci-lint version`); the
# GOOS/GOARCH axis uses `go env`'s own real environment-variable override
# (a real toolchain behavior, not a script-level mock); the config-hash
# axis points diff-ledgers.sh at a mutated copy of the real .golangci.yml
# via DIFF_LEDGERS_GOLANGCI_CONFIG, so the tracked bench/fixture-repo
# submodule working tree is never touched.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

DIFF_SCRIPT="$SCRIPTS_DIR/diff-ledgers.sh"
BUILD_LEDGERS_SCRIPT="$SCRIPTS_DIR/build-ledgers.sh"
LEDGER="$BENCH_DIR/corpus/ledgers/4c24986844b09122e2d516f9bc1ec470b155b441/lint.json"
REAL_GOLANGCI_CONFIG="$BENCH_DIR/fixture-repo/.golangci.yml"

fail() {
	echo "TC-011 FAIL: $1" >&2
	exit 1
}

[[ -x "$DIFF_SCRIPT" ]] || fail "diff-ledgers.sh missing or not executable"
[[ -x "$BUILD_LEDGERS_SCRIPT" ]] || fail "build-ledgers.sh missing or not executable"
[[ -f "$LEDGER" ]] || fail "committed base lint ledger missing: $LEDGER"
[[ -f "$REAL_GOLANGCI_CONFIG" ]] || fail "fixture .golangci.yml missing: $REAL_GOLANGCI_CONFIG"
command -v go >/dev/null 2>&1 || fail "go toolchain not found on PATH in the test environment"
command -v golangci-lint >/dev/null 2>&1 || fail "golangci-lint not found on PATH in the test environment"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

REAL_GO="$(command -v go)"
REAL_GOLANGCI_LINT="$(command -v golangci-lint)"

STUBBIN="$WORKDIR/stubbin"
mkdir -p "$STUBBIN"

# assert_guard_fails <label> <extra_env...> -- runs diff-ledgers.sh
# --toolchain-guard with the given extra environment applied on top of the
# current one, asserting: non-zero exit, no stdout (no diff mode ran), and
# returns via global OUT_FILE/ERR_FILE for the caller to grep the specific
# mismatched field.
run_guard() {
	# Caller wraps each invocation in `set +e ... set -e`; errexit is a
	# shell-global flag, so this function must not toggle it itself -- doing
	# so would re-enable errexit for the caller before it captures $?.
	local out_file="$1" err_file="$2"
	shift 2
	env "$@" "$DIFF_SCRIPT" --toolchain-guard --base="$LEDGER" >"$out_file" 2>"$err_file"
}

assert_axis_failure() {
	local label="$1" out_file="$2" err_file="$3" code="$4" expected_field="$5"
	[[ "$code" -ne 0 ]] || fail "$label: diff-ledgers.sh exited 0, expected non-zero"
	[[ ! -s "$out_file" ]] || fail "$label: diff-ledgers.sh printed stdout despite a toolchain mismatch (a diff mode must never run): $(cat "$out_file")"
	grep -q "$expected_field" "$err_file" || fail "$label: stderr does not name '$expected_field': $(cat "$err_file")"
	echo "TC-011: $label PASS ($(cat "$err_file"))"
}

# --- Baseline: unmodified environment must pass (sanity check that the
# committed ledger and this host's toolchain genuinely agree before we
# start lying about individual axes). ---
baseline_out="$WORKDIR/baseline.out"
baseline_err="$WORKDIR/baseline.err"
set +e
"$DIFF_SCRIPT" --toolchain-guard --base="$LEDGER" >"$baseline_out" 2>"$baseline_err"
baseline_code=$?
set -e
[[ "$baseline_code" -eq 0 ]] || fail "baseline: diff-ledgers.sh --toolchain-guard failed against an unmodified environment: $(cat "$baseline_err")"
echo "TC-011: baseline PASS (unmodified environment agrees with the recorded ledger)"

# --- Axis 1: Go version mismatch (stub `go`) ---
cat >"$STUBBIN/go" <<EOF
#!/usr/bin/env bash
if [[ "\$1" == "env" && "\$2" == "GOVERSION" ]]; then
	echo "go1.99.0"
	exit 0
fi
exec "$REAL_GO" "\$@"
EOF
chmod +x "$STUBBIN/go"

out1="$WORKDIR/axis1.out"
err1="$WORKDIR/axis1.err"
set +e
run_guard "$out1" "$err1" PATH="$STUBBIN:$PATH"
code1=$?
set -e
assert_axis_failure "Go version mismatch" "$out1" "$err1" "$code1" "go_version"
rm -f "$STUBBIN/go"

# --- Axis 2: golangci-lint version mismatch (stub `golangci-lint`) ---
cat >"$STUBBIN/golangci-lint" <<EOF
#!/usr/bin/env bash
if [[ "\$1" == "version" ]]; then
	echo "golangci-lint has version 9.9.9 built with go1.26.0 from deadbeef on 2026-01-01T00:00:00Z"
	exit 0
fi
exec "$REAL_GOLANGCI_LINT" "\$@"
EOF
chmod +x "$STUBBIN/golangci-lint"

out2="$WORKDIR/axis2.out"
err2="$WORKDIR/axis2.err"
set +e
run_guard "$out2" "$err2" PATH="$STUBBIN:$PATH"
code2=$?
set -e
assert_axis_failure "golangci-lint version mismatch" "$out2" "$err2" "$code2" "golangci_lint_version"
rm -f "$STUBBIN/golangci-lint"

# --- Axis 3: GOOS/GOARCH mismatch (real `go env` environment-variable
# override -- not a stub) ---
out3="$WORKDIR/axis3.out"
err3="$WORKDIR/axis3.err"
observed_goos="$(go env GOOS)"
mismatched_goos="linux"
[[ "$observed_goos" == "linux" ]] && mismatched_goos="darwin"
set +e
run_guard "$out3" "$err3" GOOS="$mismatched_goos"
code3=$?
set -e
assert_axis_failure "GOOS mismatch" "$out3" "$err3" "$code3" "goos"

out3b="$WORKDIR/axis3b.out"
err3b="$WORKDIR/axis3b.err"
observed_goarch="$(go env GOARCH)"
mismatched_goarch="amd64"
[[ "$observed_goarch" == "amd64" ]] && mismatched_goarch="arm64"
set +e
run_guard "$out3b" "$err3b" GOARCH="$mismatched_goarch"
code3b=$?
set -e
assert_axis_failure "GOARCH mismatch" "$out3b" "$err3b" "$code3b" "goarch"

# --- Axis 4: .golangci.yml content hash mismatch (mutated copy, real
# submodule file untouched) ---
mutated_config_dir="$WORKDIR/mutated-config"
mkdir -p "$mutated_config_dir"
cp "$REAL_GOLANGCI_CONFIG" "$mutated_config_dir/.golangci.yml"
printf '\n# TC-011 one-byte mutation\n' >>"$mutated_config_dir/.golangci.yml"
cmp -s "$REAL_GOLANGCI_CONFIG" "$mutated_config_dir/.golangci.yml" &&
	fail "mutated config copy is byte-identical to the real config -- mutation had no effect"

out4="$WORKDIR/axis4.out"
err4="$WORKDIR/axis4.err"
set +e
run_guard "$out4" "$err4" DIFF_LEDGERS_GOLANGCI_CONFIG="$mutated_config_dir/.golangci.yml"
code4=$?
set -e
assert_axis_failure ".golangci.yml hash mismatch" "$out4" "$err4" "$code4" "golangci_config_sha256"

# Confirm the real submodule working tree was never touched by axis 4.
cmp -s "$REAL_GOLANGCI_CONFIG" "$BENCH_DIR/fixture-repo/.golangci.yml" ||
	fail "the real bench/fixture-repo/.golangci.yml was modified by this test -- must never mutate the tracked submodule"

# --- Ownership: build-ledgers.sh must never accept or implement
# --toolchain-guard (REQ-F-010's single-owner requirement). Comment lines
# are excluded so this checks actual flag handling, not doc-comment prose
# that merely mentions the flag name to describe the ownership boundary
# (see build-ledgers.sh's own header). ---
if grep -vE '^\s*#' "$BUILD_LEDGERS_SCRIPT" | grep -q -- '--toolchain-guard'; then
	fail "build-ledgers.sh appears to handle --toolchain-guard outside a comment -- it must never own this comparison"
fi
echo "TC-011: build-ledgers.sh does not implement --toolchain-guard (ownership confirmed)"

echo "TC-011: PASS"
