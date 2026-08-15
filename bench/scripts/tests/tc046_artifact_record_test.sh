#!/usr/bin/env bash
# TC-046 (test-plan.md AC test matrix row AC-006; Caller-Path Contracts row
# tc046; T-E40-F06-006 task spec Test Cases, AC-T1 through AC-T3).
#
# Drives bench/scripts/verify-stage-evidence.sh (the artifact-record portion
# T-E40-F06-006 implements, same entrypoint as tc044/tc045) against one real
# bundle fixture under bench/scripts/testdata/evidence/artifacts/ --
# REQ-F-008, ADR-F06-07:
#
#   AC-T1: an artifact whose `consumers` key is present but an empty array
#     (`consumers: []`) yields verdict `orphan`.
#   AC-T2: an artifact whose `consumers` key is entirely absent yields
#     verdict `consumption_evidence_missing`.
#   AC-T3: neither verdict is produced for the other entry, in either
#     direction -- no coercion of absent-to-empty or empty-to-absent.
#
# A third fixture entry carries a POPULATED `consumers` array (mirroring
# the real typed-edge shape `tests/contracts/testdata/e40_i05/valid/
# lifecycle-only/stages/1-develop.json` already uses for TC-042) and must
# yield a third, distinct verdict (`consumed`). Without this third state,
# AC-T3's non-coercion assertions would be entailed by AC-T1/AC-T2 alone
# (with only two possible verdicts, "not consumption_evidence_missing"
# follows trivially from "is orphan") and an implementation that assigned
# verdicts by array position rather than by the real JSON shape could pass
# undetected -- the third state makes the non-coercion claim, and the
# guard's actual shape-driven derivation, non-trivial to fake.
#
# Caller-Path Contract (test-plan.md tc046): the fixture snapshot on disk
# encodes the states as real JSON shapes (`"consumers": []`, the key
# omitted entirely, and a populated typed-edge list) -- this test never
# tags entries "orphan"/"missing"/"consumed" itself; it only reads the
# guard's own live verdict output and asserts against it, so a broken
# JSON-shape detection inside the guard cannot pass vacuously here.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

GUARD="$SCRIPTS_DIR/verify-stage-evidence.sh"
FIXTURE="$SCRIPTS_DIR/testdata/evidence/artifacts/mixed-consumers"

fail() {
	echo "TC-046 FAIL: $1" >&2
	exit 1
}

[[ -x "$GUARD" ]] || fail "verify-stage-evidence.sh missing or not executable: $GUARD"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# Fixture self-check first: confirm the two source entries on disk genuinely
# encode the two distinct JSON shapes this test depends on -- one entry
# carries `consumers: []` and the other omits the `consumers` key entirely
# (not, e.g., `consumers: null`, which would be a different, uncovered
# state) -- mirrors tc044/tc045's own fixture self-check style.
python3 -c "
import json
with open('$FIXTURE/stages/1-plan.json') as f:
    snapshot = json.load(f)
artifacts = snapshot['artifacts']
assert len(artifacts) == 3, f'fixture bug: expected exactly 3 artifacts, got {len(artifacts)}'
empty_array_entry = next(a for a in artifacts if a['path'] == 'artifacts/empty-array-entry.md')
absent_key_entry = next(a for a in artifacts if a['path'] == 'artifacts/absent-key-entry.md')
populated_entry = next(a for a in artifacts if a['path'] == 'artifacts/populated-entry.md')
assert 'consumers' in empty_array_entry and empty_array_entry['consumers'] == [], (
    'fixture bug: empty-array-entry must carry consumers: [] (present, empty)'
)
assert 'consumers' not in absent_key_entry, (
    'fixture bug: absent-key-entry must omit the consumers key entirely, not set it to null or []'
)
assert 'consumers' in populated_entry and len(populated_entry['consumers']) > 0, (
    'fixture bug: populated-entry must carry a non-empty consumers array'
)
" || fail "fixture self-check failed (see message above)"

echo "TC-046: mixed-consumers bundle -> distinct, non-coerced verdicts per artifact"

out="$WORKDIR/out.json"
err="$WORKDIR/err.txt"
set +e
"$GUARD" "$FIXTURE" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "guard exited $code on a well-formed mixed-consumers bundle, want 0: $(cat "$err")"

# AC-T1/AC-T2/AC-T3: read the guard's own live verdict per artifact path --
# never a hand-computed expectation independent of the guard's output --
# and assert the two verdicts land on the correct entries and ONLY there.
python3 -c "
import json
with open('$out') as f:
    result = json.load(f)
artifacts = result['stages'][0]['artifacts']
by_path = {a['path']: a['verdict'] for a in artifacts}

assert by_path.get('artifacts/empty-array-entry.md') == 'orphan', (
    f'AC-T1 VIOLATION: consumers: [] entry did not yield verdict orphan, got {by_path.get(\"artifacts/empty-array-entry.md\")!r}'
)
assert by_path.get('artifacts/absent-key-entry.md') == 'consumption_evidence_missing', (
    f'AC-T2 VIOLATION: absent-consumers-key entry did not yield verdict consumption_evidence_missing, got {by_path.get(\"artifacts/absent-key-entry.md\")!r}'
)
# Third state (populated consumers): a genuinely distinct verdict from
# both orphan and consumption_evidence_missing -- makes the AC-T3
# non-coercion checks below non-trivial (see header comment) rather than
# an assertion entailed by AC-T1/AC-T2 alone.
assert by_path.get('artifacts/populated-entry.md') == 'consumed', (
    f'populated-consumers entry did not yield verdict consumed, got {by_path.get(\"artifacts/populated-entry.md\")!r}'
)

# AC-T3: non-coercion in BOTH directions -- the empty-array entry must
# never be reported consumption_evidence_missing, and the absent-key entry
# must never be reported orphan. With three distinct verdicts available,
# these checks are no longer entailed by AC-T1/AC-T2's equality assertions.
assert by_path['artifacts/empty-array-entry.md'] != 'consumption_evidence_missing', (
    'AC-T3 VIOLATION: empty-array entry was coerced to consumption_evidence_missing'
)
assert by_path['artifacts/absent-key-entry.md'] != 'orphan', (
    'AC-T3 VIOLATION: absent-key entry was coerced to orphan'
)
assert by_path['artifacts/populated-entry.md'] not in ('orphan', 'consumption_evidence_missing'), (
    'AC-T3 VIOLATION: populated-consumers entry was coerced into one of the two no-consumer verdicts'
)
" || fail "verdict assertion failed (see message above)"

echo "TC-046(AC-T1: consumers: [] -> orphan; AC-T2: absent consumers key -> consumption_evidence_missing; AC-T3: neither verdict coerced into the other in either direction) PASS"

echo "TC-046: PASS"
