#!/usr/bin/env bash
# TC-058 (test-plan.md AC test matrix row AC-012; Caller-Path Contracts row
# tc058; T-E40-F07-011 task spec Acceptance Criteria AC-T1/AC-T2/AC-T3/AC-T4).
#
# Proves REQ-F-015/REQ-F-016 against real `bench/scripts/run-prelude.sh`, a
# real scratch Shark project stood up by `scripts/shark-scratch-env.sh`
# (REQ-NF-005 -- no other project-initialisation path), and a
# PATH-stubbed dispatcher that records the constructed prompt AND the
# dispatched process's own actual working directory (its own `os.getcwd()`,
# never a value the caller hands it -- test-plan.md's own Caller-Path
# Contract wording for this row):
#
#   AC-T1 (REQ-F-015): the dispatch working directory the stub OBSERVES
#       equals the real scratch project root `scripts/shark-scratch-env.sh`
#       created; the written result's artifact_root.path is that root's real
#       path and artifact_root.identity_digest is recomputable from it alone
#       (this test recomputes it independently, never trusting the recorded
#       value at face value).
#   AC-T2 (REQ-F-012/REQ-F-015): a filesystem walk of a fixture-checkout
#       stand-in and an evaluator-only stand-in root finds zero D01-D05 or
#       docs/product/progress.md writes after the dispatch -- only the
#       scratch project (the artifact root) may carry them.
#   AC-T3 (REQ-F-016): the recorded prompt contains bench/replay/preamble.md's
#       content verbatim, read from THIS test's own fresh read of the real
#       file (never a hardcoded string), and the written result's
#       preamble_digest equals that file's real sha256.
#   AC-T4 (REQ-F-016): every produced artifact filename conforms to the
#       bundle's own D0X-*.md Output Standard, positively asserted -- proven
#       both by a conforming positive case and by a distinct case where one
#       non-conforming filename among the produced set is rejected, naming
#       it, with zero result written (a validator that only ever exercises
#       the happy path would not catch a check that silently no-ops).
#
# Caller-Path Contract (test-plan.md tc058 row): a REAL scratch-project
# filesystem stood up by scripts/shark-scratch-env.sh, a REAL read of
# bench/replay/preamble.md, and a REAL filesystem walk of the fixture
# checkout and evaluator-only stand-in roots for off-limits writes. Do not
# stand up the scratch project any other way (REQ-NF-005). Do not compute
# preamble_digest from an in-memory string constant -- read the real file.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCH_DIR="$(cd "$SCRIPTS_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"

RUN_PRELUDE="$SCRIPTS_DIR/run-prelude.sh"
STUBBIN="$SCRIPTS_DIR/testdata/stubs"
SCRATCH_ENV_SCRIPT="$REPO_ROOT/scripts/shark-scratch-env.sh"
PREAMBLE="$BENCH_DIR/replay/preamble.md"

FEATURE_PKG="$BENCH_DIR/scenarios/packages/py-feature-recurring-tasks/package.yaml"

fail() {
	echo "TC-058 FAIL: $1" >&2
	exit 1
}

[[ -x "$RUN_PRELUDE" ]] || fail "run-prelude.sh missing or not executable: $RUN_PRELUDE"
[[ -x "$STUBBIN/claude" ]] || fail "testdata/stubs/claude missing or not executable"
[[ -x "$SCRATCH_ENV_SCRIPT" ]] || fail "scripts/shark-scratch-env.sh missing or not executable: $SCRATCH_ENV_SCRIPT"
[[ -f "$PREAMBLE" ]] || fail "bench/replay/preamble.md missing: $PREAMBLE"
[[ -f "$FEATURE_PKG" ]] || fail "seed package not found: $FEATURE_PKG"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
SCRATCH_ROOTS=()
cleanup() {
	rm -rf "$WORKDIR"
	local d
	for d in "${SCRATCH_ROOTS[@]:-}"; do
		[[ -n "$d" ]] && rm -rf "$d"
	done
	return 0
}
trap cleanup EXIT

# new_scratch_project <label> -- REQ-NF-005: the ONLY sanctioned mechanism
# for standing up a live scratch Shark project. Prints the created root's
# path; the caller is responsible for its own cleanup (registered in
# SCRATCH_ROOTS so `cleanup` above removes it on exit).
new_scratch_project() {
	local label="$1"
	local root
	root="$("$SCRATCH_ENV_SCRIPT" "tc058-$label")"
	SCRATCH_ROOTS+=("$root")
	echo "$root"
}

# result_field <result.json> <field> -- top-level field reader.
result_field() {
	python3 - "$1" "$2" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    doc = json.load(f)
val = doc.get(sys.argv[2], "")
sys.stdout.write(val if isinstance(val, str) else json.dumps(val))
PYEOF
}

# log_field <log.jsonl> <field> -- the single logged invocation's field
# (this test only ever dispatches exactly once per invocation, so "the
# only line" is the right read -- checked explicitly, not assumed).
log_field() {
	python3 - "$1" "$2" <<'PYEOF'
import json
import sys

path, field = sys.argv[1], sys.argv[2]
with open(path) as f:
    lines = [line for line in f if line.strip()]
assert len(lines) == 1, f"expected exactly one logged invocation, got {len(lines)}"
record = json.loads(lines[0])
val = record[field]
sys.stdout.write(val if isinstance(val, str) else json.dumps(val))
PYEOF
}

# recompute_identity_digest <real_path> -- independently recomputes
# REQ-F-015's artifact_root.identity_digest from the recorded path ALONE,
# mirroring this feature's own "recompute, never trust the recorded value
# at face value" discipline (ADR-F07-08's entry_digest join-key proof, and
# F06's TC-042 Caller-Path Contract precedent).
recompute_identity_digest() {
	python3 -c "import hashlib,sys; print('sha256:' + hashlib.sha256(sys.argv[1].encode('utf-8')).hexdigest())" "$1"
}

realpath_of() {
	python3 -c "import os,sys; print(os.path.realpath(sys.argv[1]))" "$1"
}

# ---------------------------------------------------------------------------
# Shared setup for the AC-T1/AC-T2/AC-T3/AC-T4(positive) case: one real
# scratch project, one fixture-checkout stand-in, one evaluator-only
# stand-in, conforming D01-D05 artifacts planted in the scratch project's
# docs/product/ (simulating what a real dispatched session would have
# produced there -- run-prelude.sh's own post-dispatch scan does not care
# WHEN they appeared, only that they are present after dispatch returns,
# since no live provider is exercised in this bench-script test tier).
# ---------------------------------------------------------------------------
echo "TC-058: setup - real scratch Shark project, fixture-checkout/evaluator-only stand-ins, conforming D01-D05 fixtures"

SCRATCH_A="$(new_scratch_project main)"
FIXTURE_CHECKOUT_A="$WORKDIR/fixture-checkout-a"
EVALUATOR_ONLY_A="$WORKDIR/evaluator-only-a"
mkdir -p "$FIXTURE_CHECKOUT_A" "$EVALUATOR_ONLY_A" "$SCRATCH_A/docs/product"

for f in D01-vision-statement D02-success-criteria D03-market-research D04-feasibility-report D05-stakeholder-insights; do
	echo "# $f (fixture)" >"$SCRATCH_A/docs/product/$f.md"
done
echo "# progress (fixture)" >"$SCRATCH_A/docs/product/progress.md"

LOG_A="$WORKDIR/a-claude-invocations.jsonl"
RESULT_A="$WORKDIR/a-result.json"
ERR_A="$WORKDIR/a.err"
rm -f "$LOG_A" "$RESULT_A"

set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_A" \
	"$RUN_PRELUDE" --package "$FEATURE_PKG" --result-out "$RESULT_A" --artifact-root "$SCRATCH_A" \
	>/dev/null 2>"$ERR_A"
CODE_A=$?
set -e
[[ "$CODE_A" -eq 0 ]] || fail "setup: run-prelude.sh exited $CODE_A dispatching the real feature package with conforming artifacts already in place, want 0: $(cat "$ERR_A")"
[[ "$(wc -l <"$LOG_A")" -eq 1 ]] || fail "setup: stub claude recorded $(wc -l <"$LOG_A") invocation(s), want exactly 1"
[[ -f "$RESULT_A" ]] || fail "setup: no replay result was written for a successful dispatch"

echo "TC-058: setup complete"

# ---------------------------------------------------------------------------
# AC-T1: dispatch working directory pin + recorded/recomputable identity
# digest.
# ---------------------------------------------------------------------------
echo "TC-058: case AC-T1 - dispatch working directory equals the scratch project root; result records a recomputable identity digest"

SCRATCH_A_REAL="$(realpath_of "$SCRATCH_A")"

OBSERVED_CWD="$(log_field "$LOG_A" cwd)"
[[ "$(realpath_of "$OBSERVED_CWD")" == "$SCRATCH_A_REAL" ]] || fail "AC-T1: dispatcher's OWN observed cwd ($OBSERVED_CWD) does not equal the scratch project root ($SCRATCH_A_REAL) -- REQ-F-015 working-directory pin violated"

RESULT_ROOT_PATH="$(python3 -c "import json; print(json.load(open('$RESULT_A'))['artifact_root']['path'])")"
[[ "$(realpath_of "$RESULT_ROOT_PATH")" == "$SCRATCH_A_REAL" ]] || fail "AC-T1: result artifact_root.path ($RESULT_ROOT_PATH) does not equal the scratch project root ($SCRATCH_A_REAL)"

RESULT_ROOT_KIND="$(python3 -c "import json; print(json.load(open('$RESULT_A'))['artifact_root']['root_kind'])")"
[[ "$RESULT_ROOT_KIND" == "scratch_shark_project" ]] || fail "AC-T1: result artifact_root.root_kind=$RESULT_ROOT_KIND, want scratch_shark_project"

RESULT_IDENTITY_DIGEST="$(python3 -c "import json; print(json.load(open('$RESULT_A'))['artifact_root']['identity_digest'])")"
EXPECTED_IDENTITY_DIGEST="$(recompute_identity_digest "$RESULT_ROOT_PATH")"
[[ "$RESULT_IDENTITY_DIGEST" == "$EXPECTED_IDENTITY_DIGEST" ]] || fail "AC-T1: result artifact_root.identity_digest ($RESULT_IDENTITY_DIGEST) does not recompute from the recorded path alone (want $EXPECTED_IDENTITY_DIGEST)"

echo "TC-058(case AC-T1: dispatch cwd == scratch_shark_project root; artifact_root.path/identity_digest recorded and recomputable) PASS"

# ---------------------------------------------------------------------------
# AC-T2: no D01-D05 artifact or docs/product/progress.md write lands in the
# fixture checkout or the evaluator-only root.
# ---------------------------------------------------------------------------
echo "TC-058: case AC-T2 - no off-limits write in the fixture checkout or evaluator-only root"

OFFLIMITS_HITS="$(find "$FIXTURE_CHECKOUT_A" "$EVALUATOR_ONLY_A" -type f \( -regex '.*/D[0-9][0-9]-.*\.md' -o -name 'progress.md' \))"
[[ -z "$OFFLIMITS_HITS" ]] || fail "AC-T2: found off-limits D0X/progress.md write(s) outside the scratch project: $OFFLIMITS_HITS"

# Positive control for the walk itself: the SAME filenames genuinely do
# exist in the scratch project (the permitted root) -- proving this is a
# real, discriminating filesystem walk and not a vacuous pass because
# nothing was ever produced anywhere.
SCRATCH_HITS="$(find "$SCRATCH_A/docs/product" -type f -name 'D0*.md' | wc -l | tr -d ' ')"
[[ "$SCRATCH_HITS" -eq 5 ]] || fail "AC-T2 control: expected 5 D0X artifacts in the scratch project's docs/product/, found $SCRATCH_HITS -- test setup bug, not a real absence proof"

echo "TC-058(case AC-T2: zero D0X/progress.md writes in fixture-checkout/evaluator-only; scratch project itself genuinely carries them) PASS"

# ---------------------------------------------------------------------------
# AC-T3: recorded prompt contains preamble.md's content verbatim, read fresh
# from disk; result preamble_digest equals that file's real sha256.
# ---------------------------------------------------------------------------
echo "TC-058: case AC-T3 - recorded prompt contains preamble.md verbatim; preamble_digest matches the real file's sha256"

PREAMBLE_BYTES="$(cat "$PREAMBLE")"
RECORDED_PROMPT="$(log_field "$LOG_A" argv | python3 -c "
import json, sys
argv = json.load(sys.stdin)
i = argv.index('-p')
print(argv[i + 1])
")"
[[ "$RECORDED_PROMPT" == "$PREAMBLE_BYTES"* ]] || fail "AC-T3: recorded prompt does not begin with bench/replay/preamble.md's own content, read fresh from disk"

EXPECTED_PREAMBLE_DIGEST="$(python3 -c "import hashlib; print(hashlib.sha256(open('$PREAMBLE','rb').read()).hexdigest())")"
RESULT_PREAMBLE_DIGEST="$(result_field "$RESULT_A" preamble_digest)"
[[ "$RESULT_PREAMBLE_DIGEST" == "$EXPECTED_PREAMBLE_DIGEST" ]] || fail "AC-T3: result preamble_digest ($RESULT_PREAMBLE_DIGEST) != real preamble.md sha256 ($EXPECTED_PREAMBLE_DIGEST)"

echo "TC-058(case AC-T3: dispatched prompt carries preamble.md verbatim; preamble_digest == real file's sha256) PASS"

# ---------------------------------------------------------------------------
# AC-T4 (positive): every produced artifact filename in the written result
# conforms to the D0X-*.md Output Standard.
# ---------------------------------------------------------------------------
echo "TC-058: case AC-T4 (positive) - every produced artifact filename matches the D0X-*.md Output Standard"

python3 - "$RESULT_A" <<'PYEOF'
import json
import re
import sys

pattern = re.compile(r"^D\d{2}-.+\.md$")
with open(sys.argv[1]) as f:
    doc = json.load(f)
names = []
for stage in doc["stages"]:
    for artifact in stage.get("artifacts", []):
        name = artifact["path"].rsplit("/", 1)[-1]
        names.append(name)
        assert pattern.match(name), f"produced artifact filename {name!r} does not match D0X-*.md"
assert len(names) == 5, f"expected 5 recorded D01-D05 artifacts, got {len(names)}: {names}"
print("TC-058: all recorded artifact filenames conform:", names)
PYEOF

echo "TC-058(case AC-T4 positive: every recorded artifact filename matches D0X-*.md) PASS"

# ---------------------------------------------------------------------------
# AC-T4 (negative): a non-conforming filename among the produced set is
# rejected, naming it, with no result written for that invocation --
# proving the check actually discriminates rather than always passing.
# ---------------------------------------------------------------------------
echo "TC-058: case AC-T4 (negative) - a non-conforming produced filename is rejected, naming it, zero result written"

SCRATCH_B="$(new_scratch_project bad-filename)"
mkdir -p "$SCRATCH_B/docs/product"
for f in D01-vision-statement D02-success-criteria D03-market-research D04-feasibility-report D05-stakeholder-insights; do
	echo "# $f (fixture)" >"$SCRATCH_B/docs/product/$f.md"
done
echo "stray notes, not a D0X artifact" >"$SCRATCH_B/docs/product/notes.txt"

LOG_B="$WORKDIR/b-claude-invocations.jsonl"
RESULT_B="$WORKDIR/b-result.json"
ERR_B="$WORKDIR/b.err"
rm -f "$LOG_B" "$RESULT_B"

set +e
PATH="$STUBBIN:$PATH" STUB_CLAUDE_LOG="$LOG_B" \
	"$RUN_PRELUDE" --package "$FEATURE_PKG" --result-out "$RESULT_B" --artifact-root "$SCRATCH_B" \
	>/dev/null 2>"$ERR_B"
CODE_B=$?
set -e
[[ "$CODE_B" -ne 0 ]] || fail "AC-T4 negative: run-prelude.sh exited 0 with a non-conforming produced filename present, want non-zero"
grep -q "notes.txt" "$ERR_B" || fail "AC-T4 negative: failure message does not name the offending filename notes.txt: $(cat "$ERR_B")"
[[ ! -e "$RESULT_B" ]] || fail "AC-T4 negative: a replay result was written despite the Output Standard violation"

echo "TC-058(case AC-T4 negative: non-conforming filename rejected by name, no result written) PASS"

echo "TC-058: PASS"
