#!/usr/bin/env bash
# TC-051 (test-plan.md AC test matrix rows AC-016, AC-019; "Determinism and
# offline boundary" section; T-E40-F06-011 task spec Test Cases,
# AC-T1/AC-T2/AC-T3).
#
# Feature-level proof suite exercising the four guard scripts
# T-E40-F06-003 through -010 already built -- not co-locatable with any
# single implementation file (the task spec's own stated TDD exception).
# This script never edits verify-evidence-roots.sh, verify-stage-evidence.sh,
# replay-stage-evidence.sh, or canary-usagemapping.sh; it only proves
# properties about them, mechanically rather than by convention:
#
#   AC-T1 (REQ-NF-004, AC-016): each of the four guards, invoked twice
#     back to back over the same bundle/roots with the network disabled,
#     produces byte-identical stdout and exit code.
#   AC-T2 (REQ-NF-004, AC-016): a PATH-stubbed provider's invocation log is
#     empty across both runs of all four guards.
#   AC-T3 (REQ-F-007, AC-019): a repo-wide grep for forbidden generic-guard
#     tokens (python|pytest|pip|go test|golangci-lint|go build) over every
#     generic evidence/isolation/replay script and every tc04[3-9]/tc050
#     test returns zero hits, once the two named legitimate exceptions are
#     accounted for (see the "AC-T3" section below) -- this task's own
#     script and anything under bench/adapters/*/ are excluded from the
#     target file set entirely, per the task spec's own exclusion list.
#
# Network-isolation provisioning reuses F05's TC-039 pattern verbatim (see
# that script's own header for the full rationale): mechanism 1 is a real
# Linux network-namespace block (`unshare --user --net`), mechanism 2 is the
# portable `GOPROXY=off` fallback plus a poisoned proxy. Both are exercised
# independently -- REQ-NF-004's "byte-identical... zero provider calls" is
# the stronger of F05's two postures here: no I-05 guard is ever expected to
# attempt a network call, so this proves an absence, not merely a fallback
# path (task spec Notes for Agent).
#
# Caller-Path Contract: real subprocess execution of the four already-built
# guard scripts, against real fixture bundles/roots already committed by
# T-E40-F06-003 through -010 -- never a stubbed guard invocation and never a
# hand-computed "expected" digest/verdict substituted for the guard's own
# output. replay-stage-evidence.sh's real `adapter.sh test` subprocess
# writes .pytest_cache/__pycache__ side effects into whatever checkout it is
# pointed at (verified empirically while authoring this test), so each of
# its two runs gets its own FRESH copy of the committed bundle/checkout --
# same logical content, unpolluted by the other run's subprocess side
# effects -- while verify-evidence-roots.sh, verify-stage-evidence.sh, and
# canary-usagemapping.sh are proven read-only (by source inspection) and
# reused directly across both runs of the same mechanism.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

ROOTS_GUARD="$SCRIPTS_DIR/verify-evidence-roots.sh"
STAGE_GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
REPLAY_GUARD="$SCRIPTS_DIR/replay-stage-evidence.sh"
CANARY_GUARD="$SCRIPTS_DIR/canary-usagemapping.sh"
ADAPTER="$BENCH_DIR/adapters/python/adapter.sh"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"

ROOTS_FIXTURES="$SCRIPTS_DIR/testdata/evidence/root-policy"
STAGE_BUNDLE="$SCRIPTS_DIR/testdata/evidence/ledger/valid-disjoint"
REPLAY_BUNDLE_SRC="$SCRIPTS_DIR/testdata/evidence/replay/bundle"
REPLAY_CHECKOUT_SRC="$SCRIPTS_DIR/testdata/evidence/replay/checkout"

fail() {
	echo "TC-051 FAIL: $1" >&2
	exit 1
}

[[ -x "$ROOTS_GUARD" ]] || fail "verify-evidence-roots.sh missing or not executable: $ROOTS_GUARD"
[[ -x "$STAGE_GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable: $STAGE_GUARD"
[[ -x "$REPLAY_GUARD" ]] || fail "replay-stage-evidence.sh missing or not executable: $REPLAY_GUARD"
[[ -x "$CANARY_GUARD" ]] || fail "canary-usagemapping.sh missing or not executable: $CANARY_GUARD"
[[ -x "$ADAPTER" ]] || fail "python adapter missing or not executable: $ADAPTER"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
[[ -f "$ROOTS_FIXTURES/package/package.yaml" ]] || fail "root-policy fixtures missing: $ROOTS_FIXTURES"
[[ -f "$STAGE_BUNDLE/bundle.json" ]] || fail "ledger fixture missing: $STAGE_BUNDLE/bundle.json"
[[ -f "$REPLAY_BUNDLE_SRC/bundle.json" ]] || fail "replay bundle fixture missing: $REPLAY_BUNDLE_SRC/bundle.json"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"
python3 -c 'import yaml' >/dev/null 2>&1 || fail "python3 module 'yaml' (PyYAML) not available"
python3 -m pytest --version >/dev/null 2>&1 || fail "pytest not runnable on PATH (cache-warm precondition, mirrors TC-039)"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

ORIGINAL_PATH="$PATH"

# ---------------------------------------------------------------------------
# AC-T1/AC-T2 helpers.
#
# net_wrap is a plain bash array, always set by the caller (run_mechanism)
# as a `local` immediately before calling run_guard_twice -- bash resolves
# unqualified variable references through the live call stack (dynamic
# scope), so run_guard_twice sees run_mechanism's net_wrap without it being
# passed explicitly, the same way a nested function in this shell always
# can. No nameref indirection needed.
# ---------------------------------------------------------------------------

# run_guard_twice <label> <out1> <out2> <cmd...> -- runs <cmd...> (prefixed
# by the enclosing mechanism's net_wrap, if any) twice, capturing stdout and
# exit code each time, and asserts both runs are byte-identical on both
# axes (AC-T1).
run_guard_twice() {
	local label="$1" out1="$2" out2="$3"
	shift 3

	local code1 code2
	set +e
	if [[ "${#net_wrap[@]}" -gt 0 ]]; then
		"${net_wrap[@]}" "$@" >"$out1" 2>/dev/null
		code1=$?
		"${net_wrap[@]}" "$@" >"$out2" 2>/dev/null
		code2=$?
	else
		"$@" >"$out1" 2>/dev/null
		code1=$?
		"$@" >"$out2" 2>/dev/null
		code2=$?
	fi
	set -e

	[[ "$code1" -eq "$code2" ]] || fail "$label: exit codes differ across two back-to-back runs: $code1 vs $code2"
	diff -q "$out1" "$out2" >/dev/null || fail "$label: stdout differs across two back-to-back runs (AC-016 byte-identity violated): $(diff "$out1" "$out2" || true)"
	echo "TC-051: $label - two back-to-back runs produced byte-identical stdout (exit $code1)"
}

# run_mechanism <mech_label> <net_wrap_cmd...> -- exercises AC-T1 (byte
# identity) and AC-T2 (zero provider invocations) for all four guards under
# one network-isolation mechanism.
run_mechanism() {
	local mech_label="$1"
	shift
	local -a net_wrap=("$@")

	local claude_log="$WORKDIR/$mech_label-claude.log"
	rm -f "$claude_log"
	export PATH="$STUBBIN:$ORIGINAL_PATH"
	export STUB_CLAUDE_LOG="$claude_log"

	# --- verify-evidence-roots.sh: read-only walk, offline root-policy
	# fixtures (T-E40-F06-003), reused directly across both runs. ---
	run_guard_twice "$mech_label/verify-evidence-roots.sh" \
		"$WORKDIR/$mech_label-roots-1.out" "$WORKDIR/$mech_label-roots-2.out" \
		"$ROOTS_GUARD" \
		"$ROOTS_FIXTURES/package/package.yaml" \
		"$ROOTS_FIXTURES/clean/fixture-checkout" \
		"$ROOTS_FIXTURES/clean/scratch-project" \
		"$ROOTS_FIXTURES/package"

	# --- verify-stage-evidence.sh: read-only over a valid ledger bundle
	# (T-E40-F06-004), reused directly -- its stdout embeds the bundle_dir
	# absolute path, so reusing the SAME directory for both runs (rather
	# than two independently-named fresh copies) is what makes this a
	# meaningful byte-identity comparison, not an incidental path mismatch. ---
	run_guard_twice "$mech_label/verify-stage-evidence.sh" \
		"$WORKDIR/$mech_label-stage-1.out" "$WORKDIR/$mech_label-stage-2.out" \
		"$STAGE_GUARD" "$STAGE_BUNDLE"

	# --- replay-stage-evidence.sh: its real adapter.sh test subprocess
	# writes .pytest_cache/__pycache__ into the checkout it is pointed at
	# (empirically confirmed while authoring this test), so each run gets
	# its own fresh copy of the committed bundle/checkout -- same content,
	# no cross-run pollution. Neither guard's stdout embeds a path (verified
	# by inspection), so two differently-named fresh copies are still a
	# valid byte-identity check, unlike verify-stage-evidence.sh above. ---
	local replay_bundle_1="$WORKDIR/$mech_label-replay-1-bundle"
	local replay_checkout_1="$WORKDIR/$mech_label-replay-1-checkout"
	local replay_bundle_2="$WORKDIR/$mech_label-replay-2-bundle"
	local replay_checkout_2="$WORKDIR/$mech_label-replay-2-checkout"
	cp -r "$REPLAY_BUNDLE_SRC" "$replay_bundle_1"
	cp -r "$REPLAY_CHECKOUT_SRC" "$replay_checkout_1"
	cp -r "$REPLAY_BUNDLE_SRC" "$replay_bundle_2"
	cp -r "$REPLAY_CHECKOUT_SRC" "$replay_checkout_2"

	local replay_out_1="$WORKDIR/$mech_label-replay-1.out"
	local replay_out_2="$WORKDIR/$mech_label-replay-2.out"
	local replay_code_1 replay_code_2
	set +e
	if [[ "${#net_wrap[@]}" -gt 0 ]]; then
		"${net_wrap[@]}" "$REPLAY_GUARD" "$replay_bundle_1" --checkout "$replay_checkout_1" --adapter "$ADAPTER" \
			>"$replay_out_1" 2>/dev/null
		replay_code_1=$?
		"${net_wrap[@]}" "$REPLAY_GUARD" "$replay_bundle_2" --checkout "$replay_checkout_2" --adapter "$ADAPTER" \
			>"$replay_out_2" 2>/dev/null
		replay_code_2=$?
	else
		"$REPLAY_GUARD" "$replay_bundle_1" --checkout "$replay_checkout_1" --adapter "$ADAPTER" \
			>"$replay_out_1" 2>/dev/null
		replay_code_1=$?
		"$REPLAY_GUARD" "$replay_bundle_2" --checkout "$replay_checkout_2" --adapter "$ADAPTER" \
			>"$replay_out_2" 2>/dev/null
		replay_code_2=$?
	fi
	set -e
	[[ "$replay_code_1" -eq "$replay_code_2" ]] || fail "$mech_label/replay-stage-evidence.sh: exit codes differ across two fresh-copy runs: $replay_code_1 vs $replay_code_2"
	diff -q "$replay_out_1" "$replay_out_2" >/dev/null || fail "$mech_label/replay-stage-evidence.sh: stdout differs across two fresh-copy runs of the same committed bundle/checkout content (AC-016 byte-identity violated): $(diff "$replay_out_1" "$replay_out_2" || true)"
	echo "TC-051: $mech_label/replay-stage-evidence.sh - two fresh-copy runs of the same committed content produced byte-identical stdout (exit $replay_code_1)"

	# --- canary-usagemapping.sh: read-only over committed run fixtures
	# (T-E40-F06-009), no args, reused directly. ---
	run_guard_twice "$mech_label/canary-usagemapping.sh" \
		"$WORKDIR/$mech_label-canary-1.out" "$WORKDIR/$mech_label-canary-2.out" \
		"$CANARY_GUARD"

	# --- AC-T2: the stubbed provider recorded zero invocations across all
	# four guards' two runs each (eight invocations total this mechanism). ---
	[[ ! -s "$claude_log" ]] || fail "$mech_label: stubbed provider recorded $(wc -l <"$claude_log") invocation(s) across the four guards, want zero (REQ-NF-004 'zero provider calls'): $(cat "$claude_log")"
	echo "TC-051: $mech_label - stubbed provider invocation log empty across both runs of all four guards (AC-T2)"

	export PATH="$ORIGINAL_PATH"
	unset STUB_CLAUDE_LOG
}

# ---------------------------------------------------------------------------
# AC-T1/AC-T2: mechanism 1 - unshare --user --net (real Linux
# network-namespace block, same instantiation TC-039 established in this
# repo for bare `unshare --net`, which requires a privilege this
# environment does not have).
# ---------------------------------------------------------------------------
echo "TC-051: mechanism 1 - unshare --user --net (real Linux network-namespace block)"
if unshare --user --net -- true >/dev/null 2>&1; then
	run_mechanism "unshare-net" unshare --user --net --
else
	fail "unshare --user --net is unavailable on this host -- TC-051 requires both isolation mechanisms to run independently, not a fallback to only one"
fi

# ---------------------------------------------------------------------------
# AC-T1/AC-T2: mechanism 2 - portable GOPROXY=off/PIP_NO_INDEX=1 fallback
# with a poisoned proxy (TC-039's own portable-fallback technique, extended
# to this feature's four guards).
# ---------------------------------------------------------------------------
echo "TC-051: mechanism 2 - portable GOPROXY=off/PIP_NO_INDEX=1 fallback with a poisoned proxy"
run_mechanism "portable-fallback" env \
	GOPROXY=off \
	GOFLAGS=-mod=readonly \
	GOSUMDB=off \
	PIP_NO_INDEX=1 \
	PIP_INDEX_URL="http://127.0.0.1:9/simple" \
	http_proxy="http://127.0.0.1:9" \
	https_proxy="http://127.0.0.1:9" \
	HTTP_PROXY="http://127.0.0.1:9" \
	HTTPS_PROXY="http://127.0.0.1:9"

echo "TC-051: both isolation mechanisms independently completed all four guards, twice each, offline (AC-T1, AC-T2)"

# ---------------------------------------------------------------------------
# AC-T3 (REQ-F-007, AC-019): repo-wide forbidden-token sweep over every
# generic evidence/isolation/replay script and tc04[3-9]/tc050 test.
#
# Two legitimate exceptions, per the task spec's own exclusion list and the
# note this task's own spec quotes verbatim from T-E40-F06-010:
#   (1) this task's own script (tc051) and anything under bench/adapters/*/
#       are excluded from the TARGET FILE SET entirely -- never grepped at
#       all.
#   (2) within the target files that ARE grepped, a line that also names
#       the sanctioned adapter entrypoint -- containing the case-insensitive
#       substring "adapter" (matching either a `bench/adapters/<name>/
#       adapter.sh` path literal, an `$ADAPTER`/`ADAPTER=` variable
#       reference, or an error message describing that variable, tc048's
#       and tc049's established precedent for naming a REAL adapter
#       invocation) -- is not a language-branch leak. REQ-F-007's own
#       design channels all language-specific work through exactly that
#       entrypoint, so a forbidden token co-occurring with "adapter" on the
#       same line is naming the abstraction, not invoking the tool inline.
#       A FORBIDDEN_TOKENS='...' pattern-definition line inside a sibling
#       tcNNN test (tc045, tc049 -- this task's own predecessors) is the
#       same kind of exception: it is the grep PATTERN proving the
#       property's absence elsewhere in that file, not a real usage.
# Every other hit is a genuine violation.
# ---------------------------------------------------------------------------
echo "TC-051: AC-T3 - repo-wide grep for forbidden generic-guard tokens"

FORBIDDEN_TOKENS='\<python\>|\<pytest\>|\<pip\>|\<go[[:space:]]+test\>|\<golangci-lint\>|\<go[[:space:]]+build\>'
FORBIDDEN_TOKENS_DEFINITION_LINE='^FORBIDDEN_TOKENS='

SWEEP_TARGETS=(
	"$ROOTS_GUARD"
	"$STAGE_GUARD"
	"$REPLAY_GUARD"
	"$CANARY_GUARD"
)
shopt -s nullglob
for f in "$SCRIPT_DIR"/tc04[3-9]_*.sh "$SCRIPT_DIR"/tc050_*.sh; do
	SWEEP_TARGETS+=("$f")
done
shopt -u nullglob

[[ "${#SWEEP_TARGETS[@]}" -ge 12 ]] || fail "AC-T3: sweep target set has only ${#SWEEP_TARGETS[@]} files, expected at least 12 (4 guards + tc043..tc050) -- fixture/build-order regression"

raw_hit_count=0
legitimate_hit_count=0
violation_found=0
for target in "${SWEEP_TARGETS[@]}"; do
	[[ -f "$target" ]] || fail "AC-T3: sweep target does not exist: $target"

	while IFS= read -r hit_line; do
		[[ -n "$hit_line" ]] || continue
		raw_hit_count=$((raw_hit_count + 1))
		content="${hit_line#*:}"
		content_lower="${content,,}"
		if [[ "$content_lower" == *adapter* ]] || [[ "$content" =~ $FORBIDDEN_TOKENS_DEFINITION_LINE ]]; then
			legitimate_hit_count=$((legitimate_hit_count + 1))
			continue
		fi
		echo "TC-051: AC-T3 VIOLATION - forbidden generic-guard token found outside the two legitimate exceptions: $target:$hit_line" >&2
		violation_found=1
	done < <(grep -nE "$FORBIDDEN_TOKENS" "$target" || true)
done

[[ "$violation_found" -eq 0 ]] || fail "AC-T3: forbidden generic-guard token(s) found outside bench/adapters/*/ and the two named legitimate exceptions (see stderr above) -- REQ-F-007 language-neutrality violated"

# The filter must be doing real discriminating work, not vacuously passing
# because nothing matched at all -- the known legitimate hits (tc048/tc049
# naming the python adapter path, tc045/tc049's own FORBIDDEN_TOKENS
# pattern-definition lines) are asserted present, so a future change that
# silently widened the exception regex to swallow a real violation would
# also change this count and be visible in review.
[[ "$raw_hit_count" -gt 0 ]] || fail "AC-T3: sweep found zero raw hits at all -- the two legitimate-exception patterns would then be vacuously unexercised, not proven to correctly discriminate"
[[ "$legitimate_hit_count" -eq "$raw_hit_count" ]] || fail "AC-T3: internal check failed: $legitimate_hit_count of $raw_hit_count raw hits classified legitimate, but violation_found=0 -- counts should agree"

echo "TC-051: AC-T3 - zero forbidden-token hits outside bench/adapters/*/ and the two named legitimate exceptions ($raw_hit_count known-legitimate hit(s) correctly discriminated) PASS"

echo "TC-051: PASS"
