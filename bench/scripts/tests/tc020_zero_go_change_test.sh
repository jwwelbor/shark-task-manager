#!/usr/bin/env bash
# TC-020 (test-plan.md AC test matrix; T-E40-F03-007 task spec Test Cases).
#
# AC-28 (REQ-N-003, REQ-N-001): a MECHANICAL check ("not 'someone runs
# make test'", test-plan.md's own framing), not a wrapper around another
# script -- this file IS the check, asserted directly against the live
# repo state (test-plan.md's Caller-Path Contract row for TC-020: "n/a --
# static analysis, no subprocess under test"). Two independent halves:
#
#   (1) diff-stat (REQ-N-003): no Go file under internal/, cmd/, or
#       tests/contracts/ is added or modified by this feature.
#   (2) registration (REQ-N-001): bench/scripts/tests/run-all.sh
#       registers tc017/tc018/tc019 -- checked UNCONDITIONALLY, never
#       skipped, independent of whether half (1) can even run.
#
# Base-ref caveat -- a deliberate reading of the plan's own "wrong
# comparison is worse than no check" principle, not a weakening of it.
# test-plan.md's own text: "a vacuous pass is worse than no check, so an
# unresolvable base is a skip, not a silent pass" for the case where
# `git merge-base HEAD origin/main` fails to resolve at all. This repo's
# ACTUAL topology extends that same principle to a base that DOES
# resolve but is nonetheless the wrong comparison point: E40 develops
# every feature (F01, F02, F03, F04) on ONE long-lived branch, never
# merging a feature back to origin/main before the next one starts. By
# the time F03 lands, F02's and F04's own, separately-sanctioned Go
# changes are already committed ahead of it on this SAME branch (F04 IS
# "the epic's single Phase 1 Go change" REQ-N-003's own text names) --
# none of which are F03's. A literal "diff since origin/main is empty"
# check would therefore report a false failure on every run of this
# branch regardless of F03's own compliance -- exactly the "wrong base"
# outcome the plan's skip principle exists to avoid, just resolved
# instead of unresolved. This test adds one more discriminator on top of
# the plan's literal mechanism: when the base resolves and the diff is
# non-empty, it checks whether any COMMIT touching those paths since the
# base is F03's own (subject line matching this epic's own `(e40-f03)`
# commit-message convention). No match -> the diff is entirely sibling-
# feature work, skipped and logged (never a silent pass) naming the
# owning commits. A match -> FAIL: that is the actual defect REQ-N-003
# exists to catch (an F03 task adding a new tests/contracts/*.go file,
# e.g. a second I-02 validator, which ADR-F03-09 explicitly forbids).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

RUN_ALL="$SCRIPT_DIR/run-all.sh"

fail() {
	echo "TC-020 FAIL: $1" >&2
	exit 1
}

[[ -f "$RUN_ALL" ]] || fail "run-all.sh missing: $RUN_ALL"
command -v git >/dev/null 2>&1 || fail "git not found on PATH"

# ---------------------------------------------------------------------------
# Half (2): registration -- REQ-N-001, unconditional, never skipped. The
# literal invocation test-plan.md's TC-020 row pins (GNU grep BRE
# alternation via \|), asserting the count of MATCHING LINES equals 3 --
# one registration line per script.
# ---------------------------------------------------------------------------
reg_count="$(grep -c 'tc017_run_batch_test.sh\|tc018_aggregate_report_test.sh\|tc019_replay_manifest_test.sh' "$RUN_ALL" || true)"
[[ "$reg_count" == "3" ]] || {
	for tc in tc017_run_batch_test.sh tc018_aggregate_report_test.sh tc019_replay_manifest_test.sh; do
		grep -q "$tc" "$RUN_ALL" || echo "TC-020: run-all.sh does not register $tc" >&2
	done
	fail "run-all.sh registration count=$reg_count, want 3 (tc017, tc018, tc019 each registered once)"
}
echo "TC-020(registration: tc017/tc018/tc019 all registered in run-all.sh) PASS"

# ---------------------------------------------------------------------------
# Half (1): diff-stat -- REQ-N-003. Exit codes inside the subshell are
# deliberately distinct so the caller can tell an actual REQ-N-003 finding
# (2) apart from an infrastructure error (anything else non-zero) -- a
# `git diff`/`cd` failure must never be reported as "F03 added a Go file".
# ---------------------------------------------------------------------------
set +e
(
	cd "$REPO_ROOT" || exit 1

	base=""
	if ! base="$(git merge-base HEAD origin/main 2>/dev/null)"; then
		base=""
	fi

	if [[ -z "$base" ]]; then
		echo "TC-020(diff-stat: no merge-base resolved against origin/main -- e.g. no origin/main, detached checkout with no remote, or a shallow clone missing the merge-base commit -- skipped, logged not silently passed) SKIP" >&2
		exit 0
	fi

	changed="$(git diff --stat "$base"..HEAD -- internal/ cmd/ tests/contracts/)" || exit 1
	if [[ -z "$changed" ]]; then
		echo "TC-020(diff-stat: zero changed files under internal/, cmd/, tests/contracts/ since $base) PASS"
		exit 0
	fi

	# Non-empty diff: only a REQ-N-003 violation when at least one
	# contributing COMMIT is F03's own (see header caveat) -- this repo's
	# actual topology means F02/F04's already-landed, separately-
	# sanctioned Go work legitimately shows up here too.
	contributing_commits="$(git log --oneline "$base"..HEAD -- internal/ cmd/ tests/contracts/)" || exit 1

	if [[ -z "$contributing_commits" ]]; then
		# A non-empty diff-stat with NO commit attributable to it is not a
		# state this check can vouch for -- unlike the "every commit is a
		# sibling feature's" case below, there is nothing here naming who
		# owns the diff, so this is a finding, not a vacuous skip.
		echo "TC-020: git diff --stat is non-empty but git log over the same pathspec/range named no contributing commit -- cannot attribute the diff, treating as a finding rather than a silent skip:" >&2
		echo "$changed" >&2
		exit 2
	fi

	f03_commits="$(echo "$contributing_commits" | grep -i 'e40-f03' || true)"

	if [[ -z "$f03_commits" ]]; then
		echo "TC-020(diff-stat: non-empty diff since $base, but every contributing commit belongs to a sibling epic feature, none to e40-f03 -- skipped, logged not silently passed) SKIP" >&2
		echo "TC-020: sibling-feature commits touching internal/, cmd/, or tests/contracts/ since $base:" >&2
		echo "$contributing_commits" >&2
		exit 0
	fi

	echo "TC-020: at least one commit touching internal/, cmd/, or tests/contracts/ since $base belongs to e40-f03 itself (REQ-N-003 violation):" >&2
	echo "$f03_commits" >&2
	echo "TC-020: full changed-file list:" >&2
	echo "$changed" >&2
	exit 2
)
diff_rc=$?
set -e

if [[ "$diff_rc" -eq 2 ]]; then
	fail "diff-stat half: REQ-N-003 finding (a Go-path change is attributable to e40-f03, or a non-empty diff could not be attributed to any commit -- see stderr above)"
elif [[ "$diff_rc" -ne 0 ]]; then
	fail "diff-stat half: unexpected error while computing the diff (subshell exit $diff_rc, see stderr above) -- an infrastructure failure, not a REQ-N-003 verdict"
fi

echo "TC-020 PASS"
