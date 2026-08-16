#!/usr/bin/env bash
# TC-050 (test-plan.md AC test matrix row AC-014; Caller-Path Contracts row
# tc050; T-E40-F06-008 task spec Test Cases, AC-T1/AC-T2).
#
# Drives bench/scripts/verify-stage-evidence.sh (the stop-outcome
# eligibility portion T-E40-F06-008 implements, same entrypoint as
# tc044-tc046) against real bundle fixtures under
# bench/scripts/testdata/evidence/eligibility/ -- REQ-F-014:
#
#   AC-T1: for each of the ten named stop outcomes (resource_limit,
#     lease_loss, missing_outcome, unresolved_gate, pause, archive, error,
#     cancellation, worker_failure, timeout), partial stage snapshots stay
#     present/readable, publication_eligible is false, and
#     ineligibility_reasons[] is non-empty.
#   AC-T2: a bundle pairing any stop outcome with publication_eligible: true
#     is rejected naming the contradiction (publication_eligible_conflict,
#     bench/evidence/i05-schema.yaml error_kind vocabulary).
#
# Caller-Path Contract (test-plan.md tc050): "Do not represent 'partial
# evidence retained' as a boolean test-setup flag; the fixture must contain
# fewer stage snapshots than a complete run would, and the guard must read
# and report on the ones present." Each of the ten fixtures carries TWO
# real stage snapshots (discovery, specification) -- genuine JSON content
# on disk, not a stub -- and the fixture self-check below proves that count
# is strictly fewer than what the BUNDLE'S OWN `stage_matrix_source.prelude`
# (spec.md "Bundle layout (I-05)": "the I-04 halves this bundle's
# completeness is evaluated against", REQ-F-004) declares applicable --
# mirroring the exact stage_matrix_source shape TC-042's own fixtures use
# (tests/contracts/testdata/e40_i05/valid/prelude-lifecycle/bundle.json),
# not an indirect proxy off the unrelated closed stage_category vocabulary.
# So a guard that only ever asserted eligibility from stop_outcome's bare
# presence -- without confirming the partial snapshots are still readable
# -- would pass even if an implementation silently deleted partial evidence
# on a stop (this test also asserts the guard's own reported stages[] count
# and dispatch_ordinal set match what is genuinely on disk, so that
# silent-deletion regression is caught here).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
FIXTURES="$SCRIPTS_DIR/testdata/evidence/eligibility"

fail() {
	echo "TC-050 FAIL: $1" >&2
	exit 1
}

[[ -x "$GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable: $GUARD"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

STOP_OUTCOMES=(
	resource_limit
	lease_loss
	missing_outcome
	unresolved_gate
	pause
	archive
	error
	cancellation
	worker_failure
	timeout
)

# ---------------------------------------------------------------------------
# Fixture self-check: every one of the ten fixtures genuinely carries fewer
# stage snapshots on disk than its OWN stage_matrix_source.prelude declares
# applicable -- "fewer stage snapshots than a complete run would" made
# concrete against the bundle's own REQ-F-004 completeness reference, not a
# private assumption or an indirect vocabulary-size proxy in this test.
# ---------------------------------------------------------------------------
for outcome in "${STOP_OUTCOMES[@]}"; do
	python3 -c "
import json

with open('$FIXTURES/$outcome/bundle.json') as f:
    bundle = json.load(f)
stages = bundle['stages']
prelude = bundle['stage_matrix_source']['prelude']
applicable_count = sum(1 for entry in prelude.values() if entry.get('applicable'))
assert len(stages) < applicable_count, (
    f'fixture bug: $outcome declares {len(stages)} stage(s), which is not fewer than the '
    f'{applicable_count} applicable prelude stages its own stage_matrix_source declares -- '
    'AC-T1 requires a genuinely partial (fewer-than-complete) stage count'
)
assert bundle.get('stop_outcome') == '$outcome', (
    f'fixture bug: $outcome bundle.json declares stop_outcome={bundle.get(\"stop_outcome\")!r}, want $outcome'
)
for entry in stages:
    snapshot_path = '$FIXTURES/$outcome/' + entry['snapshot_path']
    with open(snapshot_path) as sf:
        json.load(sf)  # must be real, parseable JSON on disk -- not a stub
" || fail "$outcome: fixture self-check failed (see message above)"
done

# ---------------------------------------------------------------------------
# AC-T1: each of the ten named stop outcomes -> partial snapshots
# present/readable, publication_eligible: false, ineligibility_reasons[]
# non-empty.
# ---------------------------------------------------------------------------
for outcome in "${STOP_OUTCOMES[@]}"; do
	echo "TC-050: AC-T1 - $outcome -> partial evidence retained, publication_eligible: false"

	out="$WORKDIR/$outcome.out"
	err="$WORKDIR/$outcome.err"
	set +e
	"$GUARD" "$FIXTURES/$outcome" >"$out" 2>"$err"
	code=$?
	set -e
	[[ "$code" -eq 0 ]] || fail "$outcome: guard exited $code on a valid partial-evidence bundle, want 0: $(cat "$err")"

	python3 -c "
import json
with open('$out') as f:
    result = json.load(f)

# Partial snapshots stay present/readable: the guard's own reported
# stages[] reflects the real, on-disk partial set -- two entries, matching
# the two genuine snapshot files this fixture carries, not merely
# 'nonzero'.
stages = result['stages']
assert len(stages) == 2, (
    f'AC-T1 VIOLATION ($outcome): guard reported {len(stages)} stage(s), want 2 -- partial '
    'snapshots were not fully read and reported'
)
ordinals = sorted(s['dispatch_ordinal'] for s in stages)
assert ordinals == [1, 2], (
    f'AC-T1 VIOLATION ($outcome): guard reported dispatch_ordinal set {ordinals}, want [1, 2]'
)

eligibility = result.get('eligibility')
assert eligibility is not None, (
    f'AC-T1 VIOLATION ($outcome): guard output carries no eligibility block for a stop_outcome bundle'
)
assert eligibility.get('stop_outcome') == '$outcome', (
    f'AC-T1 VIOLATION ($outcome): eligibility.stop_outcome={eligibility.get(\"stop_outcome\")!r}, want $outcome'
)
assert eligibility.get('publication_eligible') is False, (
    f'AC-T1 VIOLATION ($outcome): eligibility.publication_eligible={eligibility.get(\"publication_eligible\")!r}, want False'
)
reasons = eligibility.get('ineligibility_reasons')
assert isinstance(reasons, list) and len(reasons) > 0, (
    f'AC-T1 VIOLATION ($outcome): eligibility.ineligibility_reasons={reasons!r}, want a non-empty list'
)
" || fail "$outcome: guard output assertion failed (see message above)"

	echo "TC-050(AC-T1: $outcome -> 2 partial snapshots present/readable, publication_eligible: false, ineligibility_reasons[] non-empty) PASS"
done

# ---------------------------------------------------------------------------
# AC-T2: a bundle pairing a stop outcome with publication_eligible: true is
# rejected naming the contradiction.
# ---------------------------------------------------------------------------
echo "TC-050: AC-T2 - stop_outcome paired with publication_eligible: true -> rejected naming the contradiction"

err="$WORKDIR/conflict.err"
set +e
"$GUARD" "$FIXTURES/eligible-with-stop-outcome" >/dev/null 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "conflict fixture: guard exited $code, want 1 (rejection): $(cat "$err")"
grep -q "publication_eligible_conflict" "$err" || fail "conflict fixture: stderr does not name publication_eligible_conflict: $(cat "$err")"
grep -q "stop_outcome='error'" "$err" || fail "conflict fixture: stderr does not name the offending stop_outcome: $(cat "$err")"

echo "TC-050(AC-T2: stop_outcome + publication_eligible: true -> rejected naming publication_eligible_conflict) PASS"

echo "TC-050: PASS"
