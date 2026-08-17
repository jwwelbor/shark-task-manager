#!/usr/bin/env bash
# TC-071: raw I-07 findings remain evidence while F09 adjudicates independently.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
NORMALIZER="$SCRIPT_DIR/../normalize-review-findings.sh"
FIXTURE="$SCRIPT_DIR/../../../tests/contracts/testdata/e40_i08/valid/review-findings.json"
INVALID="$SCRIPT_DIR/../../../tests/contracts/testdata/e40_i08/invalid/malformed-review-findings.json"
WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc071.XXXXXX")"
trap 'rm -rf "$WORKDIR"' EXIT

[[ -x "$NORMALIZER" ]] || { echo "TC-071 FAIL: normalizer missing" >&2; exit 1; }
"$NORMALIZER" --i07 "$FIXTURE" --output "$WORKDIR/findings.json"

python3 - "$WORKDIR/findings.json" "$FIXTURE" <<'PY'
import json, pathlib, sys
out = json.loads(pathlib.Path(sys.argv[1]).read_text())
source = json.loads(pathlib.Path(sys.argv[2]).read_text())
assert out["truth_set"]["available"] is True, out
assert out["derived_counts"]["emitted"] == 5, out
assert out["derived_counts"]["normalized_unique"] == 4, out
assert out["derived_counts"]["duplicate"] == 1, out
assert out["derived_counts"]["recurrent"] == 1, out
assert out["derived_counts"]["confirmed"] == 2, out
assert out["derived_counts"]["unconfirmed"] == 1, out
assert "precision" in out["derived_counts"] and "recall" in out["derived_counts"], out
assert out["raw_review_gates"] == source["review_gates"], "raw evidence was altered"
normalized = out["normalized_findings"]
assert all(item["f09_finding_id"].startswith("f09-") for item in normalized)
assert all("confirmation_source" in item and "final_disposition" in item for item in normalized)
assert any(item["duplicate_or_recurrence"] == "duplicate" for item in normalized)
assert any(item["duplicate_or_recurrence"] == "recurrent" for item in normalized)
assert any(item["raw_finding"].get("disposition") == "open" and item["final_disposition"] == "confirmed" for item in normalized), "raw disposition was trusted instead of independently adjudicated"

forged = json.loads(json.dumps(source))
for gate in forged["review_gates"]:
    for finding in gate["findings"]:
        finding["truth_set_id"] = "f09-3987da435c531171"
path = pathlib.Path(sys.argv[1]).with_name("forged-reviewer-truth.json")
path.write_text(json.dumps(forged))
PY

"$NORMALIZER" --i07 "$WORKDIR/forged-reviewer-truth.json" --output "$WORKDIR/forged-out.json"
python3 - "$WORKDIR/forged-out.json" <<'PY'
import json, pathlib, sys
out = json.loads(pathlib.Path(sys.argv[1]).read_text())
counts = out["derived_counts"]
assert counts["truth_set_status"] == "available", counts
assert counts["confirmed"] == 2 and counts["unconfirmed"] == 1, counts
clean = [item for item in out["normalized_findings"] if item["raw_finding"]["fingerprint"] == "clean"]
assert clean and clean[0]["final_disposition"] == "unconfirmed", clean
PY

set +e
"$NORMALIZER" --i07 "$INVALID" --output "$WORKDIR/invalid.json" 2>"$WORKDIR/invalid.err"
code=$?
set -e
[[ "$code" -eq 2 ]] || { echo "TC-071 FAIL: malformed finding accepted with exit $code" >&2; exit 1; }
grep -q "every review finding must be an object" "$WORKDIR/invalid.err" || { echo "TC-071 FAIL: malformed finding diagnostic missing" >&2; exit 1; }

# Production integration: evaluate-lifecycle.sh must own normalization.
mkdir -p "$WORKDIR/i05"
printf '%s\n' '{"roots":{"agent_fixture_checkout":"/does/not/exist"},"stages":[{"stage_path":"stages/code.json"}]}' > "$WORKDIR/i05/bundle.json"
python3 - "$FIXTURE" "$WORKDIR/i07.jsonl" <<'PY'
import json, pathlib, sys
pathlib.Path(sys.argv[2]).write_text(json.dumps(json.loads(pathlib.Path(sys.argv[1]).read_text())) + "\n")
PY
evaluator="$SCRIPT_DIR/../evaluate-lifecycle.sh"
set +e
"$evaluator" --i05 "$WORKDIR/i05" --i07 "$WORKDIR/i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$WORKDIR/evaluation.jsonl" >/dev/null 2>/dev/null
set -e
python3 - "$WORKDIR/evaluation.jsonl" "$FIXTURE" <<'PY'
import json, pathlib, sys
evaluation = json.loads(pathlib.Path(sys.argv[1]).read_text())
source = json.loads(pathlib.Path(sys.argv[2]).read_text())
findings = evaluation["review_findings"]
assert findings["raw_source_ref"]["i07_digest"], findings
assert findings["raw_review_gates"] == source["review_gates"], "evaluator did not retain raw I-07 review gates"
assert len(findings["normalized_findings"]) == 5, findings
assert not any(item["code"] == "review_findings_bypass" for item in evaluation["eligibility"]["invalidity_reasons"]), evaluation
PY

echo "TC-071 PASS"
