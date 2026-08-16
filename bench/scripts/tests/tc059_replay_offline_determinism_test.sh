#!/usr/bin/env bash
# TC-059 (test-plan.md AC test matrix rows AC-016, AC-019; "Determinism and
# offline boundary" section; T-E40-F07-012 task spec Acceptance Criteria
# AC-T1/AC-T2/AC-T3).
#
# Feature-level proof suite exercising the scripts T-E40-F07-004 through -011
# already built -- not co-locatable with any single implementation file (the
# task spec's own stated TDD exception, mirroring F06's TC-051). This script
# never edits run-prelude.sh, replay-answer.sh, verify-replay-result.sh, or
# verify-replay-isolation.sh; it only proves properties about them,
# mechanically rather than by convention:
#
#   AC-T1 (REQ-NF-004, AC-016): replay-answer.sh (the resolver),
#     verify-replay-result.sh, verify-replay-isolation.sh (both its
#     transcript-scan and bundle-disclosure shapes), and run-prelude.sh's
#     non-dispatch paths (--verify-argv completeness check and the
#     REQ-F-013 all-non-applicable short-circuit), each invoked twice back
#     to back over the same bundle/roots with the network disabled, produce
#     byte-identical stdout and exit code.
#   AC-T2 (REQ-NF-004, AC-016): a PATH-stubbed provider's invocation log is
#     empty across both runs of every guard/script above.
#   AC-T3 (REQ-NF-001, AC-019): a repo-wide grep for forbidden generic-guard
#     tokens (python|pytest|pip|go test|golangci-lint|go build) over exactly
#     the four generic F07 scripts (run-prelude.sh, replay-answer.sh,
#     verify-replay-result.sh, verify-replay-isolation.sh) returns zero
#     hits, once a token that is a substring of a real identifier (e.g.
#     "python3", the required interpreter binary these scripts invoke on
#     every call) is excluded via word-boundary matching, exactly as F06's
#     TC-051 does for the same four-guard shape. Unlike TC-051, no
#     "adapter" exception is needed here: F07's scripts name no
#     language-specific adapter entrypoint at all, so word-boundary matching
#     alone already yields zero hits (verified below with a synthetic
#     negative control proving the pattern is not vacuously blind).
#
# Network-isolation provisioning reuses F05's TC-039 / F06's TC-051 pattern
# verbatim (see TC-051's own header for the full rationale): mechanism 1 is
# a real Linux network-namespace block (`unshare --user --net`), mechanism 2
# is the portable `GOPROXY=off` fallback plus a poisoned proxy. Both are
# exercised independently -- no F07 script or resolver call is ever
# expected to attempt a network call, so this proves an absence, not merely
# a fallback path (task spec Notes for Agent: "do not introduce unordered
# map iteration in any script under test" -- byte-identity across both
# mechanisms is what makes that discipline mechanically checked here).
#
# Caller-Path Contract (test-plan.md tc059 row): real subprocess execution
# of the already-built scripts, against real fixture bundles/roots already
# committed by T-E40-F07-003 through -011 -- never a stubbed guard
# invocation and never a hand-computed "expected" digest/verdict
# substituted for the guard's own output. replay-answer.sh's real ledger
# side-file write means each of its two runs gets its own FRESH copy of the
# committed resolver bundle/responses fixture -- same logical content,
# unpolluted by the other run's own consumption record -- while
# verify-replay-result.sh, verify-replay-isolation.sh (both shapes), and
# run-prelude.sh's non-dispatch paths are read-only (verified by source
# inspection, matching TC-051's own verify-evidence-roots.sh/
# verify-stage-evidence.sh/canary-usagemapping.sh treatment) and reused
# directly across both runs of the same mechanism -- reuse is what makes
# verify-replay-result.sh's own embedded `result_path`/`bundle_path` stdout
# fields a meaningful byte-identity comparison rather than an incidental
# path mismatch, the same discipline TC-051 applied to
# verify-stage-evidence.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

RUN_PRELUDE="$SCRIPTS_DIR/run-prelude.sh"
RESOLVER="$SCRIPTS_DIR/replay-answer.sh"
VALIDATOR="$SCRIPTS_DIR/verify-replay-result.sh"
ISOLATION="$SCRIPTS_DIR/verify-replay-isolation.sh"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"

RESOLVER_BUNDLE_SRC="$SCRIPTS_DIR/testdata/replay/resolver/bundle.json"
RESOLVER_RESPONSES_SRC="$SCRIPTS_DIR/testdata/replay/resolver/responses"
LINEAGE_BUNDLE_SRC="$SCRIPTS_DIR/testdata/replay/lineage/bundle.json"
LINEAGE_RESULT_VALID="$SCRIPTS_DIR/testdata/replay/lineage/result-valid.json"
CLEAN_TRANSCRIPT="$SCRIPTS_DIR/testdata/replay/transcript/clean.jsonl"
REFERENCE_BUNDLE="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json"
CLEAN_FIXTURE_CHECKOUT="$SCRIPTS_DIR/testdata/replay/roots/clean-fixture-checkout"
CLEAN_SCRATCH_PROJECT="$SCRIPTS_DIR/testdata/replay/roots/clean-scratch-project"
BUG_PKG="$BENCH_DIR/scenarios/packages/py-bug-due-date-boundary/package.yaml"

fail() {
	echo "TC-059 FAIL: $1" >&2
	exit 1
}

[[ -x "$RUN_PRELUDE" ]] || fail "run-prelude.sh missing or not executable: $RUN_PRELUDE"
[[ -x "$RESOLVER" ]] || fail "replay-answer.sh missing or not executable: $RESOLVER"
[[ -x "$VALIDATOR" ]] || fail "verify-replay-result.sh missing or not executable: $VALIDATOR"
[[ -x "$ISOLATION" ]] || fail "verify-replay-isolation.sh missing or not executable: $ISOLATION"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
[[ -f "$RESOLVER_BUNDLE_SRC" ]] || fail "resolver bundle fixture missing: $RESOLVER_BUNDLE_SRC"
[[ -d "$RESOLVER_RESPONSES_SRC" ]] || fail "resolver responses fixture missing: $RESOLVER_RESPONSES_SRC"
[[ -f "$LINEAGE_BUNDLE_SRC" ]] || fail "lineage bundle fixture missing: $LINEAGE_BUNDLE_SRC"
[[ -f "$LINEAGE_RESULT_VALID" ]] || fail "lineage result-valid fixture missing: $LINEAGE_RESULT_VALID"
[[ -f "$CLEAN_TRANSCRIPT" ]] || fail "clean transcript fixture missing: $CLEAN_TRANSCRIPT"
[[ -f "$REFERENCE_BUNDLE" ]] || fail "T-E40-F07-003 reference bundle missing: $REFERENCE_BUNDLE"
[[ -d "$CLEAN_FIXTURE_CHECKOUT" ]] || fail "clean-fixture-checkout fixture missing: $CLEAN_FIXTURE_CHECKOUT"
[[ -d "$CLEAN_SCRATCH_PROJECT" ]] || fail "clean-scratch-project fixture missing: $CLEAN_SCRATCH_PROJECT"
[[ -f "$BUG_PKG" ]] || fail "seed package missing: $BUG_PKG"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"
python3 -c 'import yaml' >/dev/null 2>&1 || fail "python3 module 'yaml' (PyYAML) not available"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

ORIGINAL_PATH="$PATH"

# ---------------------------------------------------------------------------
# Setup (once, outside any isolation mechanism -- these two steps build the
# real, stateful inputs each guard is exercised AGAINST below; they are not
# themselves the property being tested, matching TC-051's own precedent of
# copying fixtures before entering the isolated section):
#
#   1. A real consumption ledger for verify-replay-result.sh, built via one
#      genuine replay-answer.sh call (mirrors tc055's own setup).
#   2. A real, complete captured argv for run-prelude.sh's --verify-argv
#      completeness check, built via one genuine dispatch against the
#      PATH-stubbed claude (mirrors tc053's case a1/a4 relationship: a real
#      capture, positively re-verified).
# ---------------------------------------------------------------------------
LINEAGE_BUNDLE="$WORKDIR/lineage-bundle.json"
cp "$LINEAGE_BUNDLE_SRC" "$LINEAGE_BUNDLE"
"$RESOLVER" --bundle "$LINEAGE_BUNDLE" --stage D01 --kind human_question --topic vision_goal \
	>"$WORKDIR/lineage-setup.out" 2>"$WORKDIR/lineage-setup.err" \
	|| fail "setup: real D01 resolver call for the lineage ledger failed: $(cat "$WORKDIR/lineage-setup.err")"
[[ -f "$LINEAGE_BUNDLE.consumption.jsonl" ]] || fail "setup: no consumption ledger written for the lineage bundle"

COMPLETE_ARGV_FIXTURE="$WORKDIR/setup-argv-capture.jsonl"
PATH="$STUBBIN:$ORIGINAL_PATH" STUB_CLAUDE_LOG="$COMPLETE_ARGV_FIXTURE" \
	"$RUN_PRELUDE" >/dev/null 2>"$WORKDIR/setup-argv.err" \
	|| fail "setup: real dispatch to capture a complete argv fixture failed: $(cat "$WORKDIR/setup-argv.err")"
PATH="$ORIGINAL_PATH"
[[ -s "$COMPLETE_ARGV_FIXTURE" ]] || fail "setup: no argv captured for the complete-argv fixture"

# ---------------------------------------------------------------------------
# AC-T1/AC-T2 helpers.
#
# net_wrap is a plain bash array, always set by the caller (run_mechanism)
# as a `local` immediately before calling run_guard_twice/run_resolver_twice
# -- bash resolves unqualified variable references through the live call
# stack (dynamic scope), so both see run_mechanism's net_wrap without it
# being passed explicitly, the same discipline TC-051 established.
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
	echo "TC-059: $label - two back-to-back runs produced byte-identical stdout (exit $code1)"
}

# run_resolver_twice <mech_label> -- replay-answer.sh is stateful (writes a
# per-bundle consumption ledger side-file), so reusing one bundle copy
# across both runs would make run 2 observe run 1's own consumption and
# diverge from run 1's supplied-response output -- not what AC-016 tests
# (two INDEPENDENT executions from the identical starting state). Each run
# gets its own fresh copy of the committed resolver bundle/responses
# fixture, same logical content, mirroring TC-051's own treatment of
# replay-stage-evidence.sh.
run_resolver_twice() {
	local mech_label="$1"
	local dir1="$WORKDIR/$mech_label-resolver-1" dir2="$WORKDIR/$mech_label-resolver-2"
	mkdir -p "$dir1" "$dir2"
	cp "$RESOLVER_BUNDLE_SRC" "$dir1/bundle.json"
	cp -r "$RESOLVER_RESPONSES_SRC" "$dir1/responses"
	cp "$RESOLVER_BUNDLE_SRC" "$dir2/bundle.json"
	cp -r "$RESOLVER_RESPONSES_SRC" "$dir2/responses"

	local out1="$WORKDIR/$mech_label-resolver-1.out" out2="$WORKDIR/$mech_label-resolver-2.out"
	local code1 code2
	set +e
	if [[ "${#net_wrap[@]}" -gt 0 ]]; then
		"${net_wrap[@]}" "$RESOLVER" --bundle "$dir1/bundle.json" --stage D01 --kind human_question --topic problem_statement \
			>"$out1" 2>/dev/null
		code1=$?
		"${net_wrap[@]}" "$RESOLVER" --bundle "$dir2/bundle.json" --stage D01 --kind human_question --topic problem_statement \
			>"$out2" 2>/dev/null
		code2=$?
	else
		"$RESOLVER" --bundle "$dir1/bundle.json" --stage D01 --kind human_question --topic problem_statement \
			>"$out1" 2>/dev/null
		code1=$?
		"$RESOLVER" --bundle "$dir2/bundle.json" --stage D01 --kind human_question --topic problem_statement \
			>"$out2" 2>/dev/null
		code2=$?
	fi
	set -e

	[[ "$code1" -eq "$code2" ]] || fail "$mech_label/replay-answer.sh: exit codes differ across two fresh-copy runs: $code1 vs $code2"
	[[ "$code1" -eq 0 ]] || fail "$mech_label/replay-answer.sh: setup expectation broken -- a matching D01/problem_statement call should succeed (exit 0), got $code1"
	diff -q "$out1" "$out2" >/dev/null || fail "$mech_label/replay-answer.sh: stdout differs across two fresh-copy runs of the same committed bundle content (AC-016 byte-identity violated)"
	echo "TC-059: $mech_label/replay-answer.sh - two fresh-copy runs of the same committed bundle content produced byte-identical stdout (exit $code1)"
}

# run_mechanism <mech_label> <net_wrap_cmd...> -- exercises AC-T1 (byte
# identity) and AC-T2 (zero provider invocations) for the resolver, both
# guards, and run-prelude.sh's non-dispatch paths under one
# network-isolation mechanism.
run_mechanism() {
	local mech_label="$1"
	shift
	local -a net_wrap=("$@")

	local claude_log="$WORKDIR/$mech_label-claude.log"
	rm -f "$claude_log"
	export PATH="$STUBBIN:$ORIGINAL_PATH"
	export STUB_CLAUDE_LOG="$claude_log"

	# --- replay-answer.sh (the resolver, REQ-F-006/REQ-F-007, T-E40-F07-005): fresh copies per run. ---
	run_resolver_twice "$mech_label"

	# --- verify-replay-result.sh (REQ-F-009 lineage reconciliation, T-E40-F07-007): read-only, reused directly. ---
	run_guard_twice "$mech_label/verify-replay-result.sh" \
		"$WORKDIR/$mech_label-lineage-1.out" "$WORKDIR/$mech_label-lineage-2.out" \
		"$VALIDATOR" "$LINEAGE_RESULT_VALID" "$LINEAGE_BUNDLE"

	# --- verify-replay-isolation.sh, transcript-scan shape (REQ-F-005, T-E40-F07-004): read-only, reused directly. ---
	run_guard_twice "$mech_label/verify-replay-isolation.sh(transcript)" \
		"$WORKDIR/$mech_label-transcript-1.out" "$WORKDIR/$mech_label-transcript-2.out" \
		"$ISOLATION" "$CLEAN_TRANSCRIPT"

	# --- verify-replay-isolation.sh, bundle-disclosure shape (REQ-F-012, T-E40-F07-006): read-only, reused directly. ---
	run_guard_twice "$mech_label/verify-replay-isolation.sh(disclosure)" \
		"$WORKDIR/$mech_label-disclosure-1.out" "$WORKDIR/$mech_label-disclosure-2.out" \
		"$ISOLATION" "$REFERENCE_BUNDLE" "$CLEAN_FIXTURE_CHECKOUT" "$CLEAN_SCRATCH_PROJECT"

	# --- run-prelude.sh --verify-argv (REQ-F-004 completeness check, non-dispatch, T-E40-F07-004): read-only, reused directly. ---
	run_guard_twice "$mech_label/run-prelude.sh(--verify-argv)" \
		"$WORKDIR/$mech_label-argv-1.out" "$WORKDIR/$mech_label-argv-2.out" \
		"$RUN_PRELUDE" --verify-argv "$COMPLETE_ARGV_FIXTURE"

	# --- run-prelude.sh --package (REQ-F-013 all-non-applicable short-circuit, non-dispatch, T-E40-F07-010): reused output path (idempotent overwrite). ---
	local na_result="$WORKDIR/$mech_label-not-applicable-result.json"
	run_guard_twice "$mech_label/run-prelude.sh(--package non-applicable)" \
		"$WORKDIR/$mech_label-na-1.out" "$WORKDIR/$mech_label-na-2.out" \
		"$RUN_PRELUDE" --package "$BUG_PKG" --result-out "$na_result"

	# --- AC-T2: the stubbed provider recorded zero invocations across the
	# resolver, both guards, and both of run-prelude.sh's non-dispatch paths'
	# two runs each this mechanism. ---
	[[ ! -s "$claude_log" ]] || fail "$mech_label: stubbed provider recorded $(wc -l <"$claude_log") invocation(s) across this mechanism's scripts, want zero (REQ-NF-004 'zero provider calls'): $(cat "$claude_log")"
	echo "TC-059: $mech_label - stubbed provider invocation log empty across both runs of every guard/script (AC-T2)"

	export PATH="$ORIGINAL_PATH"
	unset STUB_CLAUDE_LOG
}

# ---------------------------------------------------------------------------
# AC-T1/AC-T2: mechanism 1 - unshare --user --net (real Linux
# network-namespace block, same instantiation TC-039/TC-051 established in
# this repo for bare `unshare --net`, which requires a privilege this
# environment does not have).
# ---------------------------------------------------------------------------
echo "TC-059: mechanism 1 - unshare --user --net (real Linux network-namespace block)"
if unshare --user --net -- true >/dev/null 2>&1; then
	run_mechanism "unshare-net" unshare --user --net --
else
	fail "unshare --user --net is unavailable on this host -- TC-059 requires both isolation mechanisms to run independently, not a fallback to only one"
fi

# ---------------------------------------------------------------------------
# AC-T1/AC-T2: mechanism 2 - portable GOPROXY=off/PIP_NO_INDEX=1 fallback
# with a poisoned proxy (TC-039's/TC-051's own portable-fallback technique).
# ---------------------------------------------------------------------------
echo "TC-059: mechanism 2 - portable GOPROXY=off/PIP_NO_INDEX=1 fallback with a poisoned proxy"
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

echo "TC-059: both isolation mechanisms independently completed all scripts, twice each, offline (AC-T1, AC-T2)"

# ---------------------------------------------------------------------------
# AC-T3 (REQ-NF-001, AC-019): repo-wide forbidden-token sweep over exactly
# the four generic F07 scripts. Word-boundary matching excludes "python3"
# (the required interpreter binary every one of these scripts legitimately
# invokes on every call) while still catching a bare "python", "pytest",
# "pip", "go test", "golangci-lint", or "go build" token -- with these four
# scripts, that yields zero hits with NO exceptions needed (unlike F06's
# TC-051, which needed an "adapter" carve-out because its guards name a real
# language-specific adapter entrypoint; F07 has none).
# ---------------------------------------------------------------------------
echo "TC-059: AC-T3/AC-019 - repo-wide grep for forbidden generic-guard tokens"

FORBIDDEN_TOKENS='\<python\>|\<pytest\>|\<pip\>|\<go[[:space:]]+test\>|\<golangci-lint\>|\<go[[:space:]]+build\>'

SWEEP_TARGETS=(
	"$RUN_PRELUDE"
	"$RESOLVER"
	"$VALIDATOR"
	"$ISOLATION"
)

for target in "${SWEEP_TARGETS[@]}"; do
	[[ -f "$target" ]] || fail "AC-T3: sweep target does not exist: $target"
	if hit="$(grep -nE "$FORBIDDEN_TOKENS" "$target")"; then
		fail "AC-T3: forbidden generic-guard token found in $target -- REQ-NF-001/AC-019 language-neutrality violated: $hit"
	fi
done

echo "TC-059: AC-T3/AC-019 - zero forbidden-token hits across the four generic F07 scripts PASS"

# Negative control: the pattern above must not be vacuously blind -- prove
# it actually catches a real violation, on a SCRATCH copy only, never a
# committed file. Without this, a future accidental widening of the
# exception surface (e.g. broadening the word-boundary regex) could silence
# a real leak with nothing here to notice, the same "filter must do real
# discriminating work" property TC-051's own raw_hit_count check names.
echo "TC-059: AC-T3/AC-019 - negative control: the sweep pattern catches a real violation"

SCRATCH_VIOLATION="$WORKDIR/scratch-violation.sh"
cp "$RUN_PRELUDE" "$SCRATCH_VIOLATION"
printf '\n# scratch-only test injection: pytest is a forbidden generic-guard token\n' >>"$SCRATCH_VIOLATION"
grep -qE "$FORBIDDEN_TOKENS" "$SCRATCH_VIOLATION" || fail "AC-T3: negative control failed -- the sweep pattern did not catch an injected bare 'pytest' token on a scratch copy, meaning the real zero-hit result above could be a vacuously blind pattern rather than a real absence"

echo "TC-059: AC-T3/AC-019 - negative control confirms the sweep pattern is not vacuously blind PASS"

echo "TC-059: PASS"
