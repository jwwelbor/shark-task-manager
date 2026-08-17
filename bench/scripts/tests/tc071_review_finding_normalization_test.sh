#!/usr/bin/env bash
# TC-071: raw I-07 findings remain evidence while F09 adjudicates independently.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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

without_truth = dict(source)
without_truth.pop("truth_set")
path = pathlib.Path(sys.argv[1]).with_name("no-truth.json")
path.write_text(json.dumps(without_truth))
PY

"$NORMALIZER" --i07 "$WORKDIR/no-truth.json" --output "$WORKDIR/no-truth-out.json"
python3 - "$WORKDIR/no-truth-out.json" <<'PY'
import json, pathlib, sys
counts = json.loads(pathlib.Path(sys.argv[1]).read_text())["derived_counts"]
assert counts["truth_set_status"] == "truth-set-unavailable", counts
assert "precision" not in counts and "recall" not in counts, counts
PY

set +e
"$NORMALIZER" --i07 "$INVALID" --output "$WORKDIR/invalid.json" 2>"$WORKDIR/invalid.err"
code=$?
set -e
[[ "$code" -eq 2 ]] || { echo "TC-071 FAIL: malformed finding accepted with exit $code" >&2; exit 1; }
grep -q "every review finding must be an object" "$WORKDIR/invalid.err" || { echo "TC-071 FAIL: malformed finding diagnostic missing" >&2; exit 1; }

echo "TC-071 PASS"
