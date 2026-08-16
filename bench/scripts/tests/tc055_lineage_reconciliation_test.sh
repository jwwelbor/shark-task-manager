#!/usr/bin/env bash
# TC-055 (test-plan.md AC test matrix row AC-006; Caller-Path Contracts row
# tc055; T-E40-F07-007 task spec Acceptance Criteria AC-T1/AC-T2/AC-T3).
#
# Proves REQ-F-009's two distinct verdicts by driving the real
# bench/scripts/verify-replay-result.sh entrypoint directly against real
# result/bundle JSON and a REAL resolver consumption ledger produced by a
# real bench/scripts/replay-answer.sh call -- there is no caller above
# verify-replay-result.sh yet (test-plan.md tc055 row), and the ledger is
# never hand-tagged by this test:
#   (i)   a result whose D04 artifact claims a consumption
#         ({entry_id: D04-01, entry_digest: <the bundle's own real digest
#         for that entry>}) that the resolver ledger never recorded (D04-01
#         is a real, valid bundle entry that this test deliberately never
#         calls replay-answer.sh for) fails unattributed_artifact naming
#         the entry (AC-006 case i).
#   (ii)  a result whose D03 artifact was produced with an empty
#         consumed_entries[] against a stage that declares a required=true
#         entry fails unattributed_artifact naming the stage (AC-006 case
#         ii).
#   (iii) a result whose D02 stage produced NO artifact at all -- the real
#         shape a genuine unresolved_gate takes (REQ-F-008 stops the
#         prelude before an artifact is produced) -- is accepted, and the
#         script's own JSON output reports terminal_outcome: unresolved_gate
#         verbatim, never unattributed_artifact (AC-006 case iii).
#   (iv)  a clean D01 stage, whose artifact's claimed consumption exactly
#         matches a REAL ledger record from an actual replay-answer.sh
#         call, is accepted (baseline: the validator does not simply reject
#         everything).
#   (v)   a combined result carrying BOTH a clean D01 stage and a gated D02
#         stage, submitted with D02 listed FIRST in the source JSON, proves
#         the script's own output re-sorts stages into fixed order
#         (D01 then D02) rather than preserving input order (REQ-NF-004).
#
# Caller-Path Contract (test-plan.md tc055 row): real result/bundle JSON,
# real reconciliation against the resolver's own recorded ledger. This test
# never represents "consumed nothing required" / "no entry available" as a
# boolean flag it supplies itself -- every ledger record below is written by
# a real replay-answer.sh invocation, and the verdict for every case is the
# real exit code and stderr/stdout text verify-replay-result.sh itself
# produces from that real JSON content.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

RESOLVER="$SCRIPTS_DIR/replay-answer.sh"
VALIDATOR="$SCRIPTS_DIR/verify-replay-result.sh"
FIXTURE_DIR="$SCRIPTS_DIR/testdata/replay/lineage"
FIXTURE_BUNDLE="$FIXTURE_DIR/bundle.json"

fail() {
	echo "TC-055 FAIL: $1" >&2
	exit 1
}

[[ -x "$RESOLVER" ]] || fail "replay-answer.sh missing or not executable: $RESOLVER"
[[ -x "$VALIDATOR" ]] || fail "verify-replay-result.sh missing or not executable: $VALIDATOR"
[[ -f "$FIXTURE_BUNDLE" ]] || fail "lineage fixture bundle missing: $FIXTURE_BUNDLE"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

BUNDLE="$WORKDIR/bundle.json"
cp "$FIXTURE_BUNDLE" "$BUNDLE"
LEDGER="$BUNDLE.consumption.jsonl"

# ---------------------------------------------------------------------------
# Build a REAL consumption ledger: consume D01's sole entry via the real
# resolver. D02-01, D03-01, and D04-01 are deliberately left unconsumed by
# this test -- their bundle entries are real and valid, but the ledger holds
# no record for them, which is exactly the ground truth each case below
# reconciles against.
# ---------------------------------------------------------------------------
"$RESOLVER" --bundle "$BUNDLE" --stage D01 --kind human_question --topic vision_goal \
	>"$WORKDIR/resolve-d01.out" 2>"$WORKDIR/resolve-d01.err" \
	|| fail "setup: real D01 resolver call failed: $(cat "$WORKDIR/resolve-d01.err")"
[[ -f "$LEDGER" ]] || fail "setup: no consumption ledger written after the D01 resolver call"
[[ "$(wc -l <"$LEDGER")" -eq 1 ]] || fail "setup: ledger has $(wc -l <"$LEDGER") records after one supply, want exactly 1"

# ---------------------------------------------------------------------------
# Case (i) / AC-006 case i: an artifact claims a consumption absent from the
# resolver ledger -> unattributed_artifact naming the entry.
# ---------------------------------------------------------------------------
echo "TC-055: case (i) - claimed consumption absent from the ledger -> unattributed_artifact naming the entry"

out_i="$WORKDIR/case-i.out"
err_i="$WORKDIR/case-i.err"
set +e
"$VALIDATOR" "$FIXTURE_DIR/result-case-i-unattributed-entry.json" "$BUNDLE" >"$out_i" 2>"$err_i"
code_i=$?
set -e
[[ "$code_i" -eq 1 ]] || fail "case (i): exited $code_i, want 1: $(cat "$err_i")"
[[ ! -s "$out_i" ]] || fail "case (i): printed stdout on a rejection, want empty: $(cat "$out_i")"
grep -q "unattributed_artifact" "$err_i" || fail "case (i): failure message does not name unattributed_artifact: $(cat "$err_i")"
grep -q "stage=D04" "$err_i" || fail "case (i): failure message does not name the stage: $(cat "$err_i")"
grep -q "entry_id=D04-01" "$err_i" || fail "case (i): failure message does not name the entry: $(cat "$err_i")"
grep -q "unresolved_gate" "$err_i" && fail "case (i): failure message also names unresolved_gate -- the two verdicts must not blur" || true

echo "TC-055(case i: fabricated consumption claim rejected unattributed_artifact naming the entry) PASS"

# ---------------------------------------------------------------------------
# Case (ii) / AC-006 case ii: a stage produced an artifact having consumed
# zero required=true entries -> unattributed_artifact naming the stage.
# ---------------------------------------------------------------------------
echo "TC-055: case (ii) - artifact consumed zero required entries -> unattributed_artifact naming the stage"

out_ii="$WORKDIR/case-ii.out"
err_ii="$WORKDIR/case-ii.err"
set +e
"$VALIDATOR" "$FIXTURE_DIR/result-case-ii-zero-required-consumed.json" "$BUNDLE" >"$out_ii" 2>"$err_ii"
code_ii=$?
set -e
[[ "$code_ii" -eq 1 ]] || fail "case (ii): exited $code_ii, want 1: $(cat "$err_ii")"
[[ ! -s "$out_ii" ]] || fail "case (ii): printed stdout on a rejection, want empty: $(cat "$out_ii")"
grep -q "unattributed_artifact" "$err_ii" || fail "case (ii): failure message does not name unattributed_artifact: $(cat "$err_ii")"
grep -q "stage=D03" "$err_ii" || fail "case (ii): failure message does not name the stage: $(cat "$err_ii")"
grep -q "unresolved_gate" "$err_ii" && fail "case (ii): failure message also names unresolved_gate -- the two verdicts must not blur" || true

echo "TC-055(case ii: artifact with zero required consumption rejected unattributed_artifact naming the stage) PASS"

# ---------------------------------------------------------------------------
# Case (iii) / AC-006 case iii: a genuine unresolved_gate stage (no artifact
# produced at all) is accepted and REPORTED as unresolved_gate -- never
# rejected as unattributed_artifact. Proves the two verdicts are genuinely
# distinguishable, not one masquerading as the other.
# ---------------------------------------------------------------------------
echo "TC-055: case (iii) - genuine unresolved_gate (no artifact produced) -> accepted, reported as unresolved_gate"

out_iii="$WORKDIR/case-iii.out"
err_iii="$WORKDIR/case-iii.err"
set +e
"$VALIDATOR" "$FIXTURE_DIR/result-case-iii-genuine-unresolved-gate.json" "$BUNDLE" >"$out_iii" 2>"$err_iii"
code_iii=$?
set -e
[[ "$code_iii" -eq 0 ]] || fail "case (iii): exited $code_iii, want 0 (never unattributed_artifact): $(cat "$err_iii")"
[[ ! -s "$err_iii" ]] || fail "case (iii): a genuine unresolved_gate case printed to stderr, want silent accept: $(cat "$err_iii")"
grep -q "unattributed_artifact" "$out_iii" && fail "case (iii): output names unattributed_artifact for a genuine gate -- the two verdicts must not blur: $(cat "$out_iii")" || true
python3 -c "
import json, sys
doc = json.load(open(sys.argv[1]))
assert doc['terminal_outcome'] == 'unresolved_gate', f\"terminal_outcome={doc.get('terminal_outcome')!r}, want unresolved_gate\"
stages = doc['stages']
assert len(stages) == 1 and stages[0]['stage'] == 'D02', f\"stages={stages!r}\"
assert stages[0]['artifacts_produced'] == 0, f\"artifacts_produced={stages[0]['artifacts_produced']!r}, want 0\"
" "$out_iii" || fail "case (iii): output JSON does not report terminal_outcome=unresolved_gate for the gated D02 stage"

echo "TC-055(case iii: genuine gate accepted and reported as unresolved_gate, never rejected as unattributed_artifact) PASS"

# ---------------------------------------------------------------------------
# Case (iv): a clean D01 stage whose claimed consumption exactly matches a
# REAL ledger record is accepted -- the validator does not reject
# everything indiscriminately.
# ---------------------------------------------------------------------------
echo "TC-055: case (iv) - clean stage matching the real ledger is accepted"

out_iv="$WORKDIR/case-iv.out"
err_iv="$WORKDIR/case-iv.err"
set +e
"$VALIDATOR" "$FIXTURE_DIR/result-valid.json" "$BUNDLE" >"$out_iv" 2>"$err_iv"
code_iv=$?
set -e
[[ "$code_iv" -eq 0 ]] || fail "case (iv): exited $code_iv, want 0: $(cat "$err_iv")"
python3 -c "
import json, sys
doc = json.load(open(sys.argv[1]))
stages = doc['stages']
assert len(stages) == 1 and stages[0]['stage'] == 'D01', f\"stages={stages!r}\"
assert stages[0]['consumed_required_entry_ids'] == ['D01-01'], f\"consumed_required_entry_ids={stages[0]['consumed_required_entry_ids']!r}\"
assert stages[0]['result'] == 'accepted'
" "$out_iv" || fail "case (iv): output JSON does not report D01 accepted with D01-01 attributed"

echo "TC-055(case iv: clean stage reconciling against the real ledger is accepted) PASS"

# ---------------------------------------------------------------------------
# Case (v): a combined result carrying a clean D01 stage and a gated D02
# stage, with D02 listed FIRST in the source file, is accepted, and the
# script's own output re-sorts stages into fixed order (D01 then D02) --
# REQ-NF-004 byte-identical, order-independent-of-input verdicts.
# ---------------------------------------------------------------------------
echo "TC-055: case (v) - combined result, fixed-order output regardless of input order"

out_v="$WORKDIR/case-v.out"
err_v="$WORKDIR/case-v.err"
set +e
"$VALIDATOR" "$FIXTURE_DIR/result-combined-valid-and-gate.json" "$BUNDLE" >"$out_v" 2>"$err_v"
code_v=$?
set -e
[[ "$code_v" -eq 0 ]] || fail "case (v): exited $code_v, want 0: $(cat "$err_v")"
python3 -c "
import json, sys
doc = json.load(open(sys.argv[1]))
stages = doc['stages']
assert [s['stage'] for s in stages] == ['D01', 'D02'], f\"stage order={[s['stage'] for s in stages]!r}, want ['D01', 'D02'] (fixed order, not input order)\"
" "$out_v" || fail "case (v): output JSON stage order does not match fixed-order discipline"

echo "TC-055(case v: combined result accepted, output stage order fixed regardless of input order) PASS"

echo "TC-055 PASS"
