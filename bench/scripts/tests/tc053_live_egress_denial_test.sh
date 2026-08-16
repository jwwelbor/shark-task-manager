#!/usr/bin/env bash
# TC-053 (test-plan.md AC test matrix row AC-003; Caller-Path Contracts row
# tc053; T-E40-F07-004 task spec Acceptance Criteria AC-T1/AC-T2/AC-T3).
#
# Proves both halves of REQ-F-004/REQ-F-005 independently (spec.md
# "Live-egress set and its two-half proof"):
#   (a) Structural (run-prelude.sh, belt-and-braces):
#       (a1) the real argument vector run-prelude.sh constructs -- captured
#            via a PATH-stubbed `claude` dispatcher recording its argv --
#            contains one --disallowedTools entry per
#            bench/replay/live-egress-tools.yaml member, and the prompt is
#            formed as the placeholder preamble followed by the Rider
#            action invocation (AC-T3's dispatch shape).
#       (a2) removing a member from a SCRATCH COPY of live-egress-tools.yaml
#            (never the committed file) changes the captured argv with no
#            edit to run-prelude.sh itself.
#       (a3) a stubbed, already-constructed argv that is missing a member
#            (bench/scripts/testdata/replay/argv/missing-member.json) fails
#            via run-prelude.sh's own --verify-argv mode BEFORE any dispatch
#            -- proven by a PATH-stubbed dispatcher recording ZERO
#            invocations, not merely inferred from the exit code -- and
#            names the missing member.
#       (a4) positive control: the SAME completeness check, fed the real
#            captured argv from (a1) via --verify-argv, passes -- so (a3)'s
#            failure is a real detection, not a check that always fails.
#   (b) Observational and binding (verify-replay-isolation.sh):
#       (b1) a retained-transcript fixture containing one WebSearch
#            tool-use record yields live_interaction_reached naming the
#            tool and the stage.
#       (b2) a clean transcript fixture passes.
#
# Caller-Path Contract (test-plan.md tc053 row): real argv construction by
# the real script, real transcript-file parsing. Never a hand-written
# expected-argv list computed independently of live-egress-tools.yaml, and
# never a boolean "contains a violation" flag standing in for a real
# transcript fixture -- every expected member set below is derived from the
# real, committed live-egress-tools.yaml at test run time.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"

RUN_PRELUDE="$SCRIPTS_DIR/run-prelude.sh"
VERIFY_ISOLATION="$SCRIPTS_DIR/verify-replay-isolation.sh"
LIVE_EGRESS_FILE="$BENCH_DIR/replay/live-egress-tools.yaml"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"
TESTDATA="$SCRIPTS_DIR/testdata/replay"

fail() {
	echo "TC-053 FAIL: $1" >&2
	exit 1
}

[[ -x "$RUN_PRELUDE" ]] || fail "run-prelude.sh missing or not executable: $RUN_PRELUDE"
[[ -x "$VERIFY_ISOLATION" ]] || fail "verify-replay-isolation.sh missing or not executable: $VERIFY_ISOLATION"
[[ -f "$LIVE_EGRESS_FILE" ]] || fail "live-egress-tools.yaml missing: $LIVE_EGRESS_FILE"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# read_live_egress_members -- the REAL, committed set, read fresh so this
# test's own expectations track the file rather than a value hardcoded here
# a second time.
read_live_egress_members() {
	python3 - "$LIVE_EGRESS_FILE" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
for tool in data["tools"]:
    print(tool["name"])
PYEOF
}
mapfile -t LIVE_EGRESS_MEMBERS < <(read_live_egress_members)
[[ "${#LIVE_EGRESS_MEMBERS[@]}" -ge 1 ]] || fail "could not read any tools[] entries from $LIVE_EGRESS_FILE"

# extract_argv <stub_log_jsonl> -- the stub claude's captured {"argv": [...]}
# JSONL log has exactly one line per invocation; this test always invokes at
# most once per case, so emit the first line's argv array, NUL-separated
# (never newline-separated: the prompt argument itself contains embedded
# newlines between the placeholder preamble and the Rider action
# invocation, which would otherwise fragment a single argv element across
# multiple mapfile entries).
extract_argv() {
	python3 - "$1" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    line = f.readline()
record = json.loads(line)
sys.stdout.write("\0".join(record["argv"]))
if record["argv"]:
    sys.stdout.write("\0")
PYEOF
}

# ---------------------------------------------------------------------------
# Case (a1): the real argv, captured via the PATH-stubbed dispatcher,
# contains one --disallowedTools entry per live-egress-tools.yaml member,
# and the prompt is the placeholder preamble followed by the Rider action
# invocation (AC-T3).
# ---------------------------------------------------------------------------
echo "TC-053: case (a1) - real constructed argv contains one denial argument per live-egress-set member"

LOG_A1="$WORKDIR/a1-claude-invocations.jsonl"
rm -f "$LOG_A1"
out="$WORKDIR/a1.out"
err="$WORKDIR/a1.err"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_A1" "$RUN_PRELUDE" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (a1): run-prelude.sh exited $code on a normal dispatch with the stub claude, want 0: $(cat "$err")"
[[ -s "$LOG_A1" ]] || fail "case (a1): stub claude recorded no invocation"
[[ "$(wc -l <"$LOG_A1")" -eq 1 ]] || fail "case (a1): stub claude recorded $(wc -l <"$LOG_A1") invocations, want exactly 1"

mapfile -d '' -t A1_ARGV < <(extract_argv "$LOG_A1")

for member in "${LIVE_EGRESS_MEMBERS[@]}"; do
	found=0
	for i in "${!A1_ARGV[@]}"; do
		if [[ "${A1_ARGV[$i]}" == "--disallowedTools" && "${A1_ARGV[$((i + 1))]:-}" == "$member" ]]; then
			found=1
			break
		fi
	done
	[[ "$found" -eq 1 ]] || fail "case (a1): captured argv has no --disallowedTools $member entry: ${A1_ARGV[*]}"
done

DISALLOWED_COUNT=0
for arg in "${A1_ARGV[@]}"; do
	[[ "$arg" == "--disallowedTools" ]] && DISALLOWED_COUNT=$((DISALLOWED_COUNT + 1))
done
[[ "$DISALLOWED_COUNT" -eq "${#LIVE_EGRESS_MEMBERS[@]}" ]] || fail "case (a1): captured argv has $DISALLOWED_COUNT --disallowedTools entries, want exactly ${#LIVE_EGRESS_MEMBERS[@]} (one per set member, no extras)"

echo "TC-053(case a1: real argv has one --disallowedTools per live-egress-set member, no extras) PASS"

echo "TC-053: case (a1b) - AC-T3 dispatch shape: PATH-resolved binary, non-interactive print mode, preamble + Rider action invocation prompt"

[[ "${A1_ARGV[0]}" == "-p" ]] || fail "case (a1b): argv[0] is not '-p' (non-interactive print mode): ${A1_ARGV[*]}"
PROMPT_ARG="${A1_ARGV[1]:-}"
[[ "$PROMPT_ARG" == *"preamble"* ]] || fail "case (a1b): prompt does not reference a preamble placeholder: $PROMPT_ARG"
[[ "$PROMPT_ARG" == *"/shark-rider project product-design"* ]] || fail "case (a1b): prompt does not contain the Rider product-design action invocation: $PROMPT_ARG"

echo "TC-053(case a1b: dispatch shape -- '-p' print mode, preamble-then-Rider-invocation prompt) PASS"

# ---------------------------------------------------------------------------
# Case (a2): removing a member from a SCRATCH COPY of live-egress-tools.yaml
# changes the captured argv, with no edit to run-prelude.sh.
# ---------------------------------------------------------------------------
echo "TC-053: case (a2) - removing a live-egress-tools.yaml member (scratch copy) changes the argv with no script edit"

REMOVED_MEMBER="${LIVE_EGRESS_MEMBERS[-1]}"
SCRATCH_YAML="$WORKDIR/live-egress-tools-scratch.yaml"
python3 - "$LIVE_EGRESS_FILE" "$SCRATCH_YAML" "$REMOVED_MEMBER" <<'PYEOF'
import sys
import yaml

src, dst, removed = sys.argv[1:4]
with open(src) as f:
    data = yaml.safe_load(f)
before = len(data["tools"])
data["tools"] = [t for t in data["tools"] if t["name"] != removed]
assert len(data["tools"]) == before - 1, f"expected to remove exactly one tool named {removed!r}"
with open(dst, "w") as f:
    yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False)
PYEOF

LOG_A2="$WORKDIR/a2-claude-invocations.jsonl"
rm -f "$LOG_A2"
err="$WORKDIR/a2.err"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_A2" "$RUN_PRELUDE" --live-egress-file "$SCRATCH_YAML" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (a2): run-prelude.sh exited $code against the scratch (member-removed) file, want 0: $(cat "$err")"

mapfile -d '' -t A2_ARGV < <(extract_argv "$LOG_A2")
A2_DISALLOWED_COUNT=0
for arg in "${A2_ARGV[@]}"; do
	[[ "$arg" == "--disallowedTools" ]] && A2_DISALLOWED_COUNT=$((A2_DISALLOWED_COUNT + 1))
done
[[ "$A2_DISALLOWED_COUNT" -eq $((${#LIVE_EGRESS_MEMBERS[@]} - 1)) ]] || fail "case (a2): scratch-file argv has $A2_DISALLOWED_COUNT --disallowedTools entries, want $((${#LIVE_EGRESS_MEMBERS[@]} - 1)) (one member removed)"

for i in "${!A2_ARGV[@]}"; do
	if [[ "${A2_ARGV[$i]}" == "--disallowedTools" && "${A2_ARGV[$((i + 1))]:-}" == "$REMOVED_MEMBER" ]]; then
		fail "case (a2): removed member $REMOVED_MEMBER still present in the scratch-file argv: ${A2_ARGV[*]}"
	fi
done

echo "TC-053(case a2: removing $REMOVED_MEMBER from a scratch copy shrinks the argv accordingly, unmodified run-prelude.sh) PASS"

# ---------------------------------------------------------------------------
# Case (a3): a stubbed, already-constructed argv missing a member fails via
# --verify-argv BEFORE any dispatch -- proven by zero stub invocations, not
# merely the exit code.
# ---------------------------------------------------------------------------
echo "TC-053: case (a3) - stubbed argv missing a member fails before dispatch, naming the missing member"

MISSING_MEMBER_FIXTURE="$TESTDATA/argv/missing-member.json"
[[ -f "$MISSING_MEMBER_FIXTURE" ]] || fail "missing-member argv fixture not found: $MISSING_MEMBER_FIXTURE"

LOG_A3="$WORKDIR/a3-claude-invocations.jsonl"
rm -f "$LOG_A3"
err="$WORKDIR/a3.err"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_A3" "$RUN_PRELUDE" --verify-argv "$MISSING_MEMBER_FIXTURE" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (a3): run-prelude.sh --verify-argv exited 0 on an argv missing a member, want non-zero"
grep -q "argv_incomplete" "$err" || fail "case (a3): failure message does not name argv_incomplete: $(cat "$err")"
grep -q "WebFetch" "$err" || fail "case (a3): failure message does not name the missing member WebFetch: $(cat "$err")"
[[ ! -s "$LOG_A3" ]] || fail "case (a3): stub claude recorded $(wc -l <"$LOG_A3") invocation(s) for a --verify-argv check, want zero (fails BEFORE dispatch): $(cat "$LOG_A3")"

echo "TC-053(case a3: stubbed argv missing WebFetch -> argv_incomplete naming it, zero dispatcher invocations) PASS"

# ---------------------------------------------------------------------------
# Case (a4): positive control -- the SAME --verify-argv check, fed the real
# complete argv captured in case (a1), passes. Proves (a3)'s failure is a
# real detection and not a check that always fails.
# ---------------------------------------------------------------------------
echo "TC-053: case (a4) - positive control: a complete captured argv passes --verify-argv"

out="$WORKDIR/a4.out"
err="$WORKDIR/a4.err"
set +e
"$RUN_PRELUDE" --verify-argv "$LOG_A1" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (a4): run-prelude.sh --verify-argv exited $code on the real, complete captured argv, want 0: $(cat "$err")"
[[ "$(cat "$out")" == "ARGV_COMPLETE" ]] || fail "case (a4): stdout was $(cat "$out"), want ARGV_COMPLETE"

echo "TC-053(case a4: complete argv -> ARGV_COMPLETE, positive control for case a3) PASS"

# ---------------------------------------------------------------------------
# Case (b1): a retained-transcript fixture containing one WebSearch
# tool-use record yields live_interaction_reached naming the tool and stage.
# ---------------------------------------------------------------------------
echo "TC-053: case (b1) - retained transcript with a WebSearch tool-use record yields live_interaction_reached"

VIOLATION_TRANSCRIPT="$TESTDATA/transcript/websearch-violation.jsonl"
[[ -f "$VIOLATION_TRANSCRIPT" ]] || fail "websearch-violation transcript fixture not found: $VIOLATION_TRANSCRIPT"

err="$WORKDIR/b1.err"
set +e
"$VERIFY_ISOLATION" "$VIOLATION_TRANSCRIPT" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (b1): verify-replay-isolation.sh exited 0 on a transcript containing a WebSearch record, want non-zero"
grep -q "live_interaction_reached" "$err" || fail "case (b1): failure message does not name live_interaction_reached: $(cat "$err")"
grep -q "tool=WebSearch" "$err" || fail "case (b1): failure message does not name the tool WebSearch: $(cat "$err")"
grep -q "stage=D03" "$err" || fail "case (b1): failure message does not name the stage D03: $(cat "$err")"

echo "TC-053(case b1: WebSearch record -> live_interaction_reached naming tool=WebSearch stage=D03) PASS"

# ---------------------------------------------------------------------------
# Case (b2): a clean transcript passes.
# ---------------------------------------------------------------------------
echo "TC-053: case (b2) - a clean retained transcript passes"

CLEAN_TRANSCRIPT="$TESTDATA/transcript/clean.jsonl"
[[ -f "$CLEAN_TRANSCRIPT" ]] || fail "clean transcript fixture not found: $CLEAN_TRANSCRIPT"

out="$WORKDIR/b2.out"
err="$WORKDIR/b2.err"
set +e
"$VERIFY_ISOLATION" "$CLEAN_TRANSCRIPT" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (b2): verify-replay-isolation.sh exited $code on a clean transcript, want 0: $(cat "$err")"
[[ "$(cat "$out")" == "CLEAN" ]] || fail "case (b2): stdout was $(cat "$out"), want CLEAN"

echo "TC-053(case b2: clean transcript -> exit 0, CLEAN) PASS"

# ---------------------------------------------------------------------------
# Negative-half regression (test-plan.md AC-003 negative case): an
# implementation that treats the denial-argument construction alone as
# proof (skipping the transcript scan) would pass a transcript that
# actually contains a live tool-use record. Re-assert here, independently
# of case (b1), that a successful structural dispatch (case a1, argv
# correct) says NOTHING about whether the SAME transcript would pass the
# observational half -- the violation transcript in (b1) is scanned on its
# own, with no dependency on run-prelude.sh's argv outcome.
# ---------------------------------------------------------------------------
echo "TC-053: negative-half regression - structural pass does not imply observational pass"

[[ "$DISALLOWED_COUNT" -eq "${#LIVE_EGRESS_MEMBERS[@]}" ]] || fail "negative-half regression: test setup assumption (case a1 structural pass) no longer holds"
err="$WORKDIR/neg.err"
set +e
"$VERIFY_ISOLATION" "$VIOLATION_TRANSCRIPT" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "negative-half regression: verify-replay-isolation.sh passed a transcript with a live WebSearch record even though the structural denial-argument construction (case a1) was itself correct -- the observational half must independently catch what the structural half cannot verify"

echo "TC-053(negative-half regression: structural correctness alone does not mask a live-egress transcript record) PASS"

echo "TC-053: PASS"
