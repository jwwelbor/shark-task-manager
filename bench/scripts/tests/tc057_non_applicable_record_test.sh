#!/usr/bin/env bash
# TC-057 (test-plan.md AC test matrix rows AC-010, AC-011; Caller-Path
# Contracts row tc057; T-E40-F07-010 task spec Acceptance Criteria
# AC-T1/AC-T2/AC-T3).
#
# Proves REQ-F-013 (an all-non-applicable I-04 package never invokes the
# Rider action and still gets an explicit not_applicable replay result, and
# an absent result for such a scenario is itself a named failure) and
# REQ-F-014 (a read-only pre-dispatch consistency assertion between a
# package's prelude matrix and its replay_reference) against real
# bench/scripts/run-prelude.sh:
#
#   AC-010 (REQ-F-013):
#     (i)/(ii)/(iii) for each of the three real, committed non-feature seed
#         packages (py-bug-due-date-boundary, py-change-priority-scale,
#         py-techdebt-consolidate-validation -- all prelude.D01-D05
#         applicable: false): a PATH-stubbed `claude` dispatcher records
#         ZERO invocations; the written result carries
#         terminal_outcome: not_applicable; each D01-D05's `reason` is
#         byte-diffed against the SAME package's own
#         stage_matrix.prelude.D0X.reason (never a loose "reason present"
#         check -- Caller-Path Contract tc057 row).
#     (iv) verify-replay-result.sh invoked against a result path that does
#         not exist is rejected (named failure), never an accepted absence.
#   AC-011 (REQ-F-014):
#     (i) the real py-feature-recurring-tasks package, unmodified, passes.
#     (ii) a scratch copy of that feature package with replay_reference
#         removed is rejected naming the field.
#     (iii) a scratch copy of the bug package with a replay_reference field
#         added is rejected naming the field.
#     (iv) a scratch copy of the feature package whose replay_reference
#         points at a committed fixture bundle carrying a disagreeing
#         scenario_binding.scenario_id is rejected naming both ids.
#     Every mutated case operates on a scratch copy this test creates
#     itself in $WORKDIR -- no file under bench/scenarios/ is modified.
#
# Caller-Path Contract (test-plan.md tc057 row): real I-04 package files
# (scratch copies for the mutated cases), real PATH-stubbed dispatcher
# recording invocations. "reason copied verbatim" is asserted by comparing
# the written result's reason string against the real package's own
# stage_matrix.prelude.D0X.reason value, never a boolean equality flag the
# test computes independently.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

RUN_PRELUDE="$SCRIPTS_DIR/run-prelude.sh"
VERIFY_RESULT="$SCRIPTS_DIR/verify-replay-result.sh"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"
TESTDATA="$SCRIPTS_DIR/testdata/replay"

PACKAGES_DIR="$BENCH_DIR/scenarios/packages"
BUG_PKG="$PACKAGES_DIR/py-bug-due-date-boundary/package.yaml"
CHANGE_PKG="$PACKAGES_DIR/py-change-priority-scale/package.yaml"
TECHDEBT_PKG="$PACKAGES_DIR/py-techdebt-consolidate-validation/package.yaml"
FEATURE_PKG="$PACKAGES_DIR/py-feature-recurring-tasks/package.yaml"

MISMATCH_BUNDLE="$TESTDATA/packages/mismatched-scenario-bundle.json"

fail() {
	echo "TC-057 FAIL: $1" >&2
	exit 1
}

[[ -x "$RUN_PRELUDE" ]] || fail "run-prelude.sh missing or not executable: $RUN_PRELUDE"
[[ -x "$VERIFY_RESULT" ]] || fail "verify-replay-result.sh missing or not executable: $VERIFY_RESULT"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
[[ -f "$BUG_PKG" ]] || fail "seed package not found: $BUG_PKG"
[[ -f "$CHANGE_PKG" ]] || fail "seed package not found: $CHANGE_PKG"
[[ -f "$TECHDEBT_PKG" ]] || fail "seed package not found: $TECHDEBT_PKG"
[[ -f "$FEATURE_PKG" ]] || fail "seed package not found: $FEATURE_PKG"
[[ -f "$MISMATCH_BUNDLE" ]] || fail "mismatched-scenario-bundle fixture not found: $MISMATCH_BUNDLE"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# Brownfield Context: "every mutated I-04 case operates on a scratch copy
# created by the test itself; no file under bench/scenarios/ is modified."
# Captured before any case runs and re-checked at the end of this file
# (never assumed clean going in -- compared, not asserted-empty).
SCENARIOS_STATUS_BEFORE="$(cd "$REPO_ROOT" && git status --porcelain -- bench/scenarios/)"

# read_reason <package.yaml> <stage> -- the package's OWN reason string for
# that D0X stage, read fresh from the real file so this test's expectation
# tracks the committed package rather than a value re-typed here.
read_reason() {
	python3 - "$1" "$2" <<'PYEOF'
import sys
import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)
stage = sys.argv[2]
sys.stdout.write(data["stage_matrix"]["prelude"][stage]["reason"])
PYEOF
}

# result_field <result.json> <dotted.field> -- reads one field out of a
# written replay result (top-level or dotted, "." not supported for
# arrays -- only used for terminal_outcome here).
result_field() {
	python3 - "$1" "$2" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
sys.stdout.write(str(doc.get(sys.argv[2], "")))
PYEOF
}

# stage_record <result.json> <stage> -- prints "applicable\treason" for one
# D0X stage record out of a written replay result's stages[] array.
stage_record() {
	python3 - "$1" "$2" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
stage = sys.argv[2]
for record in doc.get("stages", []):
    if record.get("stage") == stage:
        sys.stdout.write(f"{record.get('applicable')}\t{record.get('reason', '')}")
        sys.exit(0)
sys.stderr.write(f"stage {stage} not found in result stages[]\n")
sys.exit(1)
PYEOF
}

# ---------------------------------------------------------------------------
# AC-010 cases (i)/(ii)/(iii): for each of the three non-feature seed
# packages, zero dispatcher invocations, terminal_outcome: not_applicable,
# and each D01-D05's reason copied verbatim.
# ---------------------------------------------------------------------------
run_non_applicable_case() {
	local name="$1" pkg="$2"
	echo "TC-057: case AC-010 ($name) - all-non-applicable package never dispatches, writes verbatim not_applicable record"

	local log="$WORKDIR/$name-claude-invocations.jsonl"
	local result="$WORKDIR/$name-result.json"
	local err="$WORKDIR/$name.err"
	rm -f "$log" "$result"

	set +e
	PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$log" \
		"$RUN_PRELUDE" --package "$pkg" --result-out "$result" >/dev/null 2>"$err"
	local code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "$name: run-prelude.sh exited $code for an all-non-applicable package, want 0: $(cat "$err")"
	[[ ! -s "$log" ]] || fail "$name: stub claude recorded $(wc -l <"$log") invocation(s) for an all-non-applicable package, want zero: $(cat "$log")"
	[[ -f "$result" ]] || fail "$name: no replay result was written for an all-non-applicable package"

	local outcome
	outcome="$(result_field "$result" terminal_outcome)"
	[[ "$outcome" == "not_applicable" ]] || fail "$name: result terminal_outcome=$outcome, want not_applicable"

	local stage applicable_got reason_got reason_want line
	for stage in D01 D02 D03 D04 D05; do
		line="$(stage_record "$result" "$stage")" || fail "$name: result stages[] missing $stage record"
		applicable_got="${line%%$'\t'*}"
		reason_got="${line#*$'\t'}"
		reason_want="$(read_reason "$pkg" "$stage")"
		[[ "$applicable_got" == "False" ]] || fail "$name: result stage $stage applicable=$applicable_got, want False"
		[[ "$reason_got" == "$reason_want" ]] || fail "$name: result stage $stage reason=$reason_got, want byte-identical to package's own reason=$reason_want"
	done

	echo "TC-057(case AC-010 $name: zero dispatch, not_applicable, verbatim reasons D01-D05) PASS"
}

run_non_applicable_case "bug" "$BUG_PKG"
run_non_applicable_case "change_card" "$CHANGE_PKG"
run_non_applicable_case "tech_debt" "$TECHDEBT_PKG"

# ---------------------------------------------------------------------------
# AC-010 case (iv): producing no result for a non-applicable scenario is
# itself a named failure -- verify-replay-result.sh against an absent
# result file is rejected, never an accepted absence.
# ---------------------------------------------------------------------------
echo "TC-057: case AC-010 (iv) - absent replay result is a named failure, never an accepted absence"

ABSENT_RESULT="$WORKDIR/does-not-exist-result.json"
[[ ! -e "$ABSENT_RESULT" ]] || fail "test setup bug: $ABSENT_RESULT unexpectedly exists"
err="$WORKDIR/absent.err"
set +e
"$VERIFY_RESULT" "$ABSENT_RESULT" "$WORKDIR/placeholder-bundle.json" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (iv): verify-replay-result.sh exited 0 against an absent result file, want non-zero"
grep -q "$ABSENT_RESULT" "$err" || fail "case (iv): failure message does not name the absent result path: $(cat "$err")"

echo "TC-057(case AC-010 iv: absent result file -> named, non-zero failure) PASS"

# ---------------------------------------------------------------------------
# AC-011 case (i): the real feature package, unmodified, passes REQ-F-014's
# consistency check (and, since it has applicable stages, falls through to
# the unmodified dispatch path -- proven by exactly one stub invocation).
# ---------------------------------------------------------------------------
echo "TC-057: case AC-011 (i) - real feature package passes the consistency check"

LOG_I="$WORKDIR/feature-claude-invocations.jsonl"
RESULT_I="$WORKDIR/feature-result.json"
err="$WORKDIR/feature.err"
rm -f "$LOG_I"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_I" \
	"$RUN_PRELUDE" --package "$FEATURE_PKG" --result-out "$RESULT_I" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (i): run-prelude.sh exited $code for the real, unmodified feature package, want 0: $(cat "$err")"
[[ "$(wc -l <"$LOG_I")" -eq 1 ]] || fail "case (i): stub claude recorded $(wc -l <"$LOG_I") invocation(s) for a package with applicable stages, want exactly 1 (consistency check passed, fell through to dispatch)"

echo "TC-057(case AC-011 i: real feature package passes, falls through to dispatch) PASS"

# ---------------------------------------------------------------------------
# AC-011 case (ii): a scratch copy of the feature package with
# replay_reference removed is rejected naming the field. Never reaches
# dispatch (zero stub invocations).
# ---------------------------------------------------------------------------
echo "TC-057: case AC-011 (ii) - scratch feature copy missing replay_reference is rejected naming the field"

SCRATCH_MISSING_REF="$WORKDIR/feature-missing-ref.yaml"
python3 - "$FEATURE_PKG" "$SCRATCH_MISSING_REF" <<'PYEOF'
import sys
import yaml

src, dst = sys.argv[1:3]
with open(src) as f:
    data = yaml.safe_load(f)
assert "replay_reference" in data, "test setup bug: real feature package has no replay_reference to remove"
del data["replay_reference"]
with open(dst, "w") as f:
    yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False)
PYEOF

LOG_II="$WORKDIR/missing-ref-claude-invocations.jsonl"
err="$WORKDIR/missing-ref.err"
rm -f "$LOG_II"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_II" \
	"$RUN_PRELUDE" --package "$SCRATCH_MISSING_REF" --result-out "$WORKDIR/missing-ref-result.json" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (ii): run-prelude.sh exited 0 against a scratch package with an applicable stage but no replay_reference, want non-zero"
grep -q "replay_reference" "$err" || fail "case (ii): failure message does not name replay_reference: $(cat "$err")"
[[ ! -s "$LOG_II" ]] || fail "case (ii): stub claude recorded $(wc -l <"$LOG_II") invocation(s) despite the read-only consistency check, want zero (checked BEFORE any dispatch): $(cat "$LOG_II")"
[[ ! -e "$WORKDIR/missing-ref-result.json" ]] || fail "case (ii): a replay result was written despite the consistency violation"

echo "TC-057(case AC-011 ii: missing replay_reference on an applicable package -> rejected naming the field, zero dispatch) PASS"

# ---------------------------------------------------------------------------
# AC-011 case (iii): a scratch copy of the bug package with a
# replay_reference field added is rejected naming the field.
# ---------------------------------------------------------------------------
echo "TC-057: case AC-011 (iii) - scratch bug copy with an unexpected replay_reference is rejected naming the field"

SCRATCH_EXTRA_REF="$WORKDIR/bug-extra-ref.yaml"
python3 - "$BUG_PKG" "$SCRATCH_EXTRA_REF" <<'PYEOF'
import sys
import yaml

src, dst = sys.argv[1:3]
with open(src) as f:
    data = yaml.safe_load(f)
assert "replay_reference" not in data, "test setup bug: real bug package already has a replay_reference"
data["replay_reference"] = "does-not-need-to-exist.json"
with open(dst, "w") as f:
    yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False)
PYEOF

LOG_III="$WORKDIR/extra-ref-claude-invocations.jsonl"
err="$WORKDIR/extra-ref.err"
rm -f "$LOG_III"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_III" \
	"$RUN_PRELUDE" --package "$SCRATCH_EXTRA_REF" --result-out "$WORKDIR/extra-ref-result.json" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (iii): run-prelude.sh exited 0 against a non-applicable scratch package carrying an unexpected replay_reference, want non-zero"
grep -q "replay_reference" "$err" || fail "case (iii): failure message does not name replay_reference: $(cat "$err")"
[[ ! -s "$LOG_III" ]] || fail "case (iii): stub claude recorded $(wc -l <"$LOG_III") invocation(s) despite the read-only consistency check, want zero: $(cat "$LOG_III")"
[[ ! -e "$WORKDIR/extra-ref-result.json" ]] || fail "case (iii): a replay result was written despite the consistency violation"

echo "TC-057(case AC-011 iii: unexpected replay_reference on a non-applicable package -> rejected naming the field, zero dispatch) PASS"

# ---------------------------------------------------------------------------
# AC-011 case (iv): a scratch copy of the feature package whose
# replay_reference points at a committed fixture bundle carrying a
# disagreeing scenario_binding.scenario_id is rejected naming both ids.
# ---------------------------------------------------------------------------
echo "TC-057: case AC-011 (iv) - scratch feature copy pointing at a bundle with a disagreeing scenario_id is rejected naming both ids"

SCRATCH_MISMATCH_DIR="$WORKDIR/mismatch-scenario"
mkdir -p "$SCRATCH_MISMATCH_DIR"
SCRATCH_MISMATCH_PKG="$SCRATCH_MISMATCH_DIR/package.yaml"
python3 - "$FEATURE_PKG" "$SCRATCH_MISMATCH_PKG" <<'PYEOF'
import sys
import yaml

src, dst = sys.argv[1:3]
with open(src) as f:
    data = yaml.safe_load(f)
# scenario_id is left UNCHANGED (still py-feature-recurring-tasks) -- only
# the replay_reference is repointed at the mismatched-scenario fixture
# bundle, so the disagreement is entirely in the BUNDLE's own
# scenario_binding.scenario_id, per AC-011's own case (iv) framing.
data["replay_reference"] = "mismatched-scenario-bundle.json"
with open(dst, "w") as f:
    yaml.safe_dump(data, f, default_flow_style=False, sort_keys=False)
PYEOF
cp "$MISMATCH_BUNDLE" "$SCRATCH_MISMATCH_DIR/mismatched-scenario-bundle.json"

BUNDLE_SCENARIO_ID="$(python3 -c "import json; print(json.load(open('$MISMATCH_BUNDLE'))['scenario_binding']['scenario_id'])")"
PKG_SCENARIO_ID="$(python3 -c "import yaml; print(yaml.safe_load(open('$FEATURE_PKG'))['scenario_id'])")"
[[ "$BUNDLE_SCENARIO_ID" != "$PKG_SCENARIO_ID" ]] || fail "test setup bug: mismatched-scenario-bundle fixture's scenario_id equals the real feature package's own scenario_id -- fixture no longer mismatched"

LOG_IV="$WORKDIR/mismatch-claude-invocations.jsonl"
err="$WORKDIR/mismatch.err"
rm -f "$LOG_IV"
set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_IV" \
	"$RUN_PRELUDE" --package "$SCRATCH_MISMATCH_PKG" --result-out "$WORKDIR/mismatch-result.json" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (iv): run-prelude.sh exited 0 against a package/bundle pair with disagreeing scenario ids, want non-zero"
grep -q "$PKG_SCENARIO_ID" "$err" || fail "case (iv): failure message does not name the package's scenario_id ($PKG_SCENARIO_ID): $(cat "$err")"
grep -q "$BUNDLE_SCENARIO_ID" "$err" || fail "case (iv): failure message does not name the bundle's disagreeing scenario_id ($BUNDLE_SCENARIO_ID): $(cat "$err")"
[[ ! -s "$LOG_IV" ]] || fail "case (iv): stub claude recorded $(wc -l <"$LOG_IV") invocation(s) despite the read-only consistency check, want zero: $(cat "$LOG_IV")"
[[ ! -e "$WORKDIR/mismatch-result.json" ]] || fail "case (iv): a replay result was written despite the consistency violation"

echo "TC-057(case AC-011 iv: disagreeing scenario_binding.scenario_id -> rejected naming both ids, zero dispatch) PASS"

# ---------------------------------------------------------------------------
# Repo-hygiene regression (Brownfield Context): none of the mutated cases
# above touched a real, committed file under bench/scenarios/.
# ---------------------------------------------------------------------------
echo "TC-057: repo-hygiene regression - no file under bench/scenarios/ was modified"

SCENARIOS_STATUS_AFTER="$(cd "$REPO_ROOT" && git status --porcelain -- bench/scenarios/)"
[[ "$SCENARIOS_STATUS_BEFORE" == "$SCENARIOS_STATUS_AFTER" ]] || fail "bench/scenarios/ git status changed during this test run (before=[$SCENARIOS_STATUS_BEFORE] after=[$SCENARIOS_STATUS_AFTER])"

echo "TC-057(repo-hygiene regression: bench/scenarios/ untouched) PASS"

echo "TC-057: PASS"
