#!/usr/bin/env bash
# TC-013 (test-plan.md AC test matrix; T-E40-F01-007 task spec Test Cases).
#
# Exercises AC-012 / AC-T2-T3 (REQ-NF-005): `admit.sh bench/corpus/corpus.yaml`
# (full admitted set, no --item) run under genuine network isolation, once
# with the pinned `golangci-lint v2.9.0` present on PATH and once with it
# absent --
#   - present + offline: the full gate completes (exit 0), no outbound
#     network attempt occurs
#   - absent + offline: the gate fails loudly, naming `golangci-lint` as the
#     missing tool, and never falls back to root Makefile:108's curl-based
#     auto-installer
#
# Network isolation is real, not a code-level flag (test-plan.md Caller-Path
# Contract for TC-013): this script prefers a true network namespace via
# `unshare --user --net` (works unprivileged on this Linux host without
# needing `CAP_SYS_ADMIN` in the parent namespace, unlike bare
# `unshare --net`); where that is unavailable it falls back to the portable
# `GOPROXY=off GOFLAGS=-mod=readonly` + a poisoned local proxy pointed at a
# guaranteed-closed port, per test-plan.md's documented fallback.
#
# Caller-Path Contract (test-plan.md TC-013): drives the real admit.sh
# entrypoint under real subprocess execution and real network isolation --
# no network call is stubbed in code, and PATH is filtered by directory
# (not by an env flag admit.sh reads) to make golangci-lint genuinely
# unresolvable for the "absent" case.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ADMIT_SCRIPT="$SCRIPTS_DIR/admit.sh"
CHECKOUT_SCRIPT="$SCRIPTS_DIR/checkout-fixture.sh"
CORPUS_YAML="$BENCH_DIR/corpus/corpus.yaml"

fail() {
	echo "TC-013 FAIL: $1" >&2
	exit 1
}

[[ -x "$ADMIT_SCRIPT" ]] || fail "admit.sh missing or not executable"
[[ -x "$CHECKOUT_SCRIPT" ]] || fail "checkout-fixture.sh missing or not executable"
[[ -f "$CORPUS_YAML" ]] || fail "corpus.yaml missing: $CORPUS_YAML"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# --- Warm the Go module cache (REQ-NF-005 precondition: "once the fixture
# and its Go module cache are present"). The fixture module has no external
# dependencies, so this mainly proves the checkout/build path itself needs
# no network; a `go test` failure here is expected (the fixture ships one
# deliberately-failing regression probe) and is not fatal to warming.
BASE_SHA="$(python3 - "$CORPUS_YAML" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
print(data["fixture"]["base_sha"])
PYEOF
)"
[[ -n "$BASE_SHA" ]] || fail "could not derive base_sha from $CORPUS_YAML"

warm_dir="$WORKDIR/warm-checkout"
"$CHECKOUT_SCRIPT" "$BASE_SHA" "$warm_dir" >/dev/null
(cd "$warm_dir" && go build ./... && go test ./... >/dev/null 2>&1) || true
rm -rf "$warm_dir"

# --- Choose the network isolation mechanism ---
NET_WRAP=()
if unshare --user --net -- true >/dev/null 2>&1; then
	NET_WRAP=(unshare --user --net --)
	isolation_desc="unshare --user --net (real network-namespace block)"
else
	isolation_desc="GOPROXY=off + poisoned-proxy fallback (unshare --user --net unavailable)"
fi
echo "TC-013: network isolation mechanism: $isolation_desc"

run_isolated() {
	# run_isolated <out_file> <err_file> <path_value>
	# Runs admit.sh (full corpus.yaml, no --item) under the chosen
	# isolation mechanism and the given PATH. Returns admit.sh's exit code.
	local out_file="$1" err_file="$2" run_path="$3"
	"${NET_WRAP[@]}" env \
		PATH="$run_path" \
		GOPROXY=off \
		GOFLAGS=-mod=readonly \
		GOSUMDB=off \
		http_proxy="http://127.0.0.1:9" \
		https_proxy="http://127.0.0.1:9" \
		HTTP_PROXY="http://127.0.0.1:9" \
		HTTPS_PROXY="http://127.0.0.1:9" \
		"$ADMIT_SCRIPT" "$CORPUS_YAML" >"$out_file" 2>"$err_file"
}

network_attempt_pattern='dial tcp|connection refused|no such host|network is unreachable|could not resolve|temporary failure in name resolution'

echo "TC-013: case 1 - pinned golangci-lint present, network isolated"

golangci_bin="$(command -v golangci-lint || true)"
[[ -n "$golangci_bin" ]] || fail "golangci-lint not found on PATH in the test environment -- TC-013 case 1 requires the pinned binary to be installed to exercise the 'present' branch (an environment precondition, not admit.sh's own check)"

out1="$WORKDIR/case1.out"
err1="$WORKDIR/case1.err"
set +e
run_isolated "$out1" "$err1" "$PATH"
code1=$?
set -e
[[ "$code1" -eq 0 ]] || fail "case 1: admit.sh exited $code1 with golangci-lint present and network isolated (expected 0); stderr: $(cat "$err1")"
[[ -s "$out1" ]] || fail "case 1: admit.sh produced no stdout verdicts"
if grep -qiE "$network_attempt_pattern" "$err1"; then
	fail "case 1: stderr shows an outbound network attempt: $(cat "$err1")"
fi
echo "TC-013: case 1 PASS ($(grep -c . "$out1") verdict line(s), offline, golangci-lint present)"

echo "TC-013: case 2 - pinned golangci-lint absent from PATH, network isolated"

golangci_dir="$(dirname "$golangci_bin")"
filtered_path=""
IFS=':' read -ra path_entries <<<"$PATH"
for p in "${path_entries[@]}"; do
	[[ "$p" == "$golangci_dir" ]] && continue
	filtered_path="${filtered_path:+$filtered_path:}$p"
done
[[ "$filtered_path" != "$PATH" ]] || fail "filtering $golangci_dir out of PATH had no effect -- golangci-lint's directory was not isolated"
if PATH="$filtered_path" command -v golangci-lint >/dev/null 2>&1; then
	fail "golangci-lint is still resolvable after filtering $golangci_dir from PATH -- a second copy exists elsewhere on PATH, TC-013 cannot exercise the 'absent' branch"
fi

out2="$WORKDIR/case2.out"
err2="$WORKDIR/case2.err"
set +e
run_isolated "$out2" "$err2" "$filtered_path"
code2=$?
set -e
[[ "$code2" -ne 0 ]] || fail "case 2: admit.sh exited 0 with golangci-lint absent from PATH (expected non-zero failure); stdout: $(cat "$out2")"
grep -qi 'golangci-lint' "$err2" || fail "case 2: stderr does not name golangci-lint as the missing tool: $(cat "$err2")"
if grep -qiE 'curl|install\.sh|raw\.githubusercontent\.com/golangci' "$err2"; then
	fail "case 2: stderr references the curl-based auto-installer -- must never fall back to it: $(cat "$err2")"
fi
[[ ! -s "$out2" ]] || fail "case 2: admit.sh printed verdict output despite the missing pinned tool -- it must fail before evaluating any candidate: $(cat "$out2")"
if grep -qiE "$network_attempt_pattern" "$err2"; then
	fail "case 2: stderr shows an outbound network attempt around the missing-tool guard: $(cat "$err2")"
fi
echo "TC-013: case 2 PASS (missing-tool failure named on stderr, no fallback, no network attempt)"

# Structural proof (independent of what happened to run above) that the
# missing-tool guard cannot silently trigger root Makefile:108's installer:
# admit.sh must never itself invoke curl or make as a subprocess. Comment
# lines are excluded so this checks actual invocations (argv-style quoted
# tokens or a bare shell/subprocess call), not prose that merely mentions
# "curl" or "make" (e.g. this script's own header, or admit.sh's doc
# comments about the guarantee it upholds).
if grep -vE '^\s*#' "$ADMIT_SCRIPT" | grep -qE '["'"'"']curl["'"'"']|["'"'"']make["'"'"']|(^|[^A-Za-z0-9_./-])curl[[:space:]]|(^|[^A-Za-z0-9_./-])make[[:space:]]'; then
	fail "admit.sh itself invokes curl or make outside a comment -- it must never be able to trigger Makefile:108's auto-installer"
fi

echo "TC-013: PASS"
