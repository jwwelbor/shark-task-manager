#!/usr/bin/env bash
# TC-063 / test-plan.md TC-006 and TC-009.
#
# Caller-path contract: invoke the real review-capture.sh entrypoint with
# committed JSON fixtures. No serializer, raw-note parser, or join is mocked.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CAPTURE="$SCRIPTS_DIR/review-capture.sh"
FIXTURES="$SCRIPTS_DIR/testdata/lifecycle/review"

fail() { echo "TC-063 FAIL: $1" >&2; exit 1; }

[[ -x "$CAPTURE" ]] || fail "review-capture.sh missing or not executable"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

for state in findings zero-findings collector-failure unreached; do

  "$CAPTURE" --input "$FIXTURES/$state.json" --output "$WORKDIR/$state.json" \
    || fail "$state fixture was not captured"
done

python3 - "$WORKDIR" "$FIXTURES" <<'PY'
import json
import pathlib
import sys

out_dir, fixture_dir = map(pathlib.Path, sys.argv[1:])

findings = json.loads((out_dir / "findings.json").read_text())
assert len(findings["review_gates"]) == 1
gate = findings["review_gates"][0]
assert gate["state"] == "findings", gate
assert gate["candidate_ref"] == {"snapshot_digest": "candidate-a"}, gate
assert gate["policy_ref"] == {"policy_digest": "policy-a"}, gate
assert len(gate["findings"]) == 1, gate
finding = gate["findings"][0]
assert finding["gate"] == "qa"
assert finding["round"] == 2
assert finding["severity"] == "medium"
assert finding["defect_class"] == "boundary validation"
assert finding["fingerprint"] == "src/example.py:parse:boundary-validation"
assert finding["criterion"] == "AC-006"
assert finding["test"] == "TC-063"
assert finding["disposition"] == "open"
assert finding["metadata"] == {"owner": "reviewer-a", "custom": ["keep", 7]}
assert finding["raw_bytes"] == json.loads((fixture_dir / "findings.json").read_text())["gates"][0]["findings"][0]["raw_bytes"]

zero = json.loads((out_dir / "zero-findings.json").read_text())["review_gates"][0]
assert zero["state"] == "zero_findings" and zero["findings"] == [], zero

failed = json.loads((out_dir / "collector-failure.json").read_text())["review_gates"][0]
assert failed["state"] == "collection_failure" and failed["findings"] == [], failed
assert failed["collector_error"] == "review command exited 17", failed

unreached = json.loads((out_dir / "unreached.json").read_text())["review_gates"][0]
assert unreached["state"] == "not_reached" and unreached["findings"] == [], unreached
assert unreached["candidate_ref"] == {"snapshot_digest": "candidate-a"}, unreached

PY

set +e
"$CAPTURE" --input "$FIXTURES/duplicate-gate.json" --output "$WORKDIR/duplicate.json" \
  >"$WORKDIR/duplicate.out" 2>"$WORKDIR/duplicate.err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "duplicate gate accepted with exit $code"
grep -q "duplicate gate_id" "$WORKDIR/duplicate.err" || fail "duplicate diagnostic omitted gate_id"

echo "TC-063(TC-006/TC-009: distinct gate states, raw finding preservation, and exact joins) PASS"
