#!/usr/bin/env bash
# TC-071: raw I-07 findings remain evidence while F09 adjudicates independently.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
NORMALIZER="$SCRIPT_DIR/../normalize-review-findings.sh"
FIXTURE="$SCRIPT_DIR/../../../tests/contracts/testdata/e40_i08/valid/review-findings.json"
NO_TRUTH_SET="$SCRIPT_DIR/../../../tests/contracts/testdata/e40_i08/valid/review-findings-no-truth-set.json"
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

# Missing-truth-set partition (test-plan.md TC-071): a run whose I-07 source
# never declares a truth_set must not report truth_set.available or invent
# precision/recall -- the permanently-committed calibration fixture must not
# be treated as universally authoritative for every run.
"$NORMALIZER" --i07 "$NO_TRUTH_SET" --output "$WORKDIR/no-truth-set-out.json"
python3 - "$WORKDIR/no-truth-set-out.json" <<'PY'
import json, pathlib, sys
out = json.loads(pathlib.Path(sys.argv[1]).read_text())
counts = out["derived_counts"]
assert out["truth_set"]["available"] is False, out
assert counts["truth_set_status"] == "truth-set-unavailable", counts
assert "precision" not in counts and "recall" not in counts, counts
assert counts["emitted"] == 5 and counts["normalized_unique"] == 4, counts
assert counts["duplicate"] == 1 and counts["recurrent"] == 1, counts
assert counts["confirmed"] == 0, "a run with no truth set must not fabricate confirmations"
assert counts["unconfirmed"] == 3, counts
assert all(item["final_disposition"] == "unconfirmed" for item in out["normalized_findings"]), out["normalized_findings"]
PY

set +e
"$NORMALIZER" --i07 "$INVALID" --output "$WORKDIR/invalid.json" 2>"$WORKDIR/invalid.err"
code=$?
set -e
[[ "$code" -eq 2 ]] || { echo "TC-071 FAIL: malformed finding accepted with exit $code" >&2; exit 1; }
grep -q "every review finding must be an object" "$WORKDIR/invalid.err" || { echo "TC-071 FAIL: malformed finding diagnostic missing" >&2; exit 1; }

# Present-but-malformed truth_set partition (round-15 repro): a truth_set
# value that is merely non-null but does not satisfy the shape contract must
# fail closed with an error, never silently report available:true nor
# silently degrade to unavailable.
for case in empty-object empty-list wrong-type bad-digest unknown-field; do
  variant="$(python3 - "$FIXTURE" "$case" <<'PY'
import json, sys
source = json.loads(open(sys.argv[1]).read())
case = sys.argv[2]
bad = {
    "empty-object": {},
    "empty-list": [],
    "wrong-type": "seed-1",
    "bad-digest": {"digest": "not-hex", "finding_ids": ["seed-1"]},
    "unknown-field": {"digest": "1" * 64, "finding_ids": ["seed-1"], "extra": "x"},
}[case]
source["truth_set"] = bad
print(json.dumps(source))
PY
)"
  echo "$variant" > "$WORKDIR/malformed-truth-$case.json"
  set +e
  "$NORMALIZER" --i07 "$WORKDIR/malformed-truth-$case.json" --output "$WORKDIR/malformed-truth-$case-out.json" 2>"$WORKDIR/malformed-truth-$case.err"
  code=$?
  set -e
  [[ "$code" -eq 2 ]] || { echo "TC-071 FAIL: malformed-but-present truth_set ($case) accepted with exit $code" >&2; exit 1; }
  [[ ! -f "$WORKDIR/malformed-truth-$case-out.json" ]] || { echo "TC-071 FAIL: malformed-but-present truth_set ($case) still produced output" >&2; exit 1; }
  grep -q "truth_set" "$WORKDIR/malformed-truth-$case.err" || { echo "TC-071 FAIL: malformed truth_set ($case) diagnostic missing" >&2; exit 1; }
done

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
quality = evaluation["metrics"]["quality"]
assert quality["truth_set_status"] == "available" and quality["precision"]["available"] is True, quality
PY

# Production integration, missing-truth-set partition: the I-08 evaluation
# record's metrics.quality must also report truth-set-unavailable end to
# end, not just the intermediate normalizer artifact.
python3 - "$NO_TRUTH_SET" "$WORKDIR/i07-no-truth.jsonl" <<'PY'
import json, pathlib, sys
pathlib.Path(sys.argv[2]).write_text(json.dumps(json.loads(pathlib.Path(sys.argv[1]).read_text())) + "\n")
PY
set +e
"$evaluator" --i05 "$WORKDIR/i05" --i07 "$WORKDIR/i07-no-truth.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$WORKDIR/evaluation-no-truth.jsonl" >/dev/null 2>/dev/null
set -e
python3 - "$WORKDIR/evaluation-no-truth.jsonl" <<'PY'
import json, pathlib, sys
evaluation = json.loads(pathlib.Path(sys.argv[1]).read_text())
quality = evaluation["metrics"]["quality"]
assert quality["truth_set_status"] == "truth-set-unavailable", quality
assert quality["precision"]["available"] is False and quality["precision"]["value"] is None, quality
assert quality["recall"]["available"] is False and quality["recall"]["value"] is None, quality
PY

# Production integration, complete decision-table row (QA1-002): TC-067's
# truth decision table requires an explicit "structural-pass/judge-pass/
# oracle-pass (eligible)" row, but every place `aggregate_eligible: true` was
# previously asserted (TC-070/072/075) fed a hand-built JSONL straight to
# compare-lifecycle-evaluations.sh -- never derived from a real
# evaluate-lifecycle.sh run. Build a genuinely complete identity/policy/
# oracle input (real fixture-py checkout, real reference patch, real
# adapter-backed held-back oracle, real repo-computed deep-review bundle
# digest) and assert the terminal eligibility flag through the real
# evaluator entrypoint, closing that row for real.
CHECKOUT_SCRIPT="$SCRIPT_DIR/../checkout-scenario-fixture.sh"
mkdir -p "$WORKDIR/eligible/i05"
"$CHECKOUT_SCRIPT" py 964fa68e4c9e0c4e0f3756d9efd78b888c558fd9 "$WORKDIR/eligible/checkout" >/dev/null
git -C "$WORKDIR/eligible/checkout" apply "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/evaluator/reference.patch"

python3 - "$REPO_ROOT" "$WORKDIR/eligible" <<'PY'
import hashlib, json, os, sys

repo_root, out_dir = sys.argv[1:3]


def sha(label):
    return hashlib.sha256(label.encode()).hexdigest()


def canonical_digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")).hexdigest()


# Mirrors evaluate-lifecycle.sh's own DEEP_REVIEW_FILES/deep_review_digest so
# the fixture's deep_review_bundle_digest is independently, really computed
# from the repository's own files rather than copied from the evaluator.
DEEP_REVIEW_FILES = (
    "skills/shark-rider/skills/deep-review/SKILL.md",
    "skills/shark-rider/skills/deep-review/references/angle-a-bugs.md",
    "skills/shark-rider/skills/deep-review/references/angle-b-behavior.md",
    "skills/shark-rider/skills/deep-review/references/angle-c-sibling.md",
    "skills/shark-rider/skills/deep-review/references/angle-d-cleanup.md",
    "skills/shark-rider/skills/deep-review/references/angle-e-tests.md",
    "skills/shark-rider/skills/deep-review/references/angle-f-standards.md",
    "skills/shark-rider/skills/deep-review/references/consolidator.md",
    "skills/shark-rider/skills/deep-review/scripts/get_diff.sh",
)


def deep_review_digest(root):
    hasher = hashlib.sha256()
    for relative in DEEP_REVIEW_FILES:
        path = os.path.join(root, relative)
        if not os.path.isfile(path):
            return None
        hasher.update(relative.encode("utf-8"))
        hasher.update(b"\0")
        with open(path, "rb") as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                hasher.update(chunk)
        hasher.update(b"\0")
    return hasher.hexdigest()


bundle_digest = deep_review_digest(repo_root)
assert bundle_digest, "repository deep-review bundle is incomplete; cannot build an eligible fixture"

candidate_fields = {
    "base_commit": sha("base_commit"),
    "tree_digest": sha("tree_digest"),
    "binary_diff_digest": sha("binary_diff_digest"),
    "changed_path_digest": sha("changed_path_digest"),
    "test_suite_digest": sha("test_suite_digest"),
}
candidate = dict(candidate_fields)
candidate["dirty_untracked_manifest"] = []
candidate["identity_digest"] = canonical_digest({**candidate_fields, "dirty_untracked_manifest": []})
candidate["snapshot_digest"] = sha("snapshot_digest")

workflow_policy = {
    "enabled_gates": ["code", "review"],
    "gate_order": ["code", "review"],
    "reviewer": {"provider": "anthropic", "model": "claude-sonnet", "effort": "high"},
    "prompt_digest": sha("prompt_digest"),
    "rendered_prompt_digest": sha("rendered_prompt_digest"),
    "deep_review_bundle_digest": bundle_digest,
    "fixes_allowed_between_gates": True,
}
workflow_policy["workflow_policy_identity_digest"] = canonical_digest(workflow_policy)

identity = {
    "run_id": "tc071-eligible-run",
    "scenario_id": "py-bug-due-date-boundary",
    "scenario_version": "1",
    "fixture_id": "py",
    "fixture_digest": sha("fixture_digest"),
    "adapter_id": "python",
    "adapter_version": "1.0.0",
    "dispatch_id": "dispatch-1",
    "dispatch_ordinal": 1,
    "toolchain_identity": [{"key": "python_version", "value": "3.12.3"}],
    "shark_binary_digest": sha("shark_binary_digest"),
    "shark_content_digest": sha("shark_content_digest"),
    "rendered_prompt_digests": [sha("rendered_prompt_1")],
    "provider_identity": [{"stage": "code", "provider": "anthropic", "model": "claude-sonnet", "effort": "high"}],
    "judge_identity": {"model": "not-applicable", "configuration": "not-applicable"},
    "reference_digests": [sha("reference_digest_1")],
    "resource_policy_digest": sha("resource_policy_digest"),
}

dispatches = [{"dispatch_id": "dispatch-1", "dispatch_ordinal": 1, "transition": "advance"}]

lifecycle = {
    "identity": identity,
    "outcome": {"terminal": "complete"},
    "entity_graph": {"nodes": ["run"]},
    "dispatches": dispatches,
    "workflow_policy": workflow_policy,
    "review_gates": [],
    "stages": [{"stage": "code", "category": "code", "candidate": candidate, "input_lineage": []}],
}

with open(os.path.join(out_dir, "i07.jsonl"), "w", encoding="utf-8") as stream:
    stream.write(json.dumps(lifecycle))
    stream.write("\n")

i05 = {
    "identity": {"run_id": "tc071-eligible-run", "scenario_id": "py-bug-due-date-boundary", "scenario_version": "1", "dispatch_id": "dispatch-1", "dispatch_ordinal": 1},
    "roots": {"agent_fixture_checkout": os.path.join(out_dir, "checkout")},
    "terminal_status": {"reached": True, "reached_at": "2020-01-01T00:00:00Z"},
    "stages": [{"stage_path": "stages/code.json"}],
    "dispatches": dispatches,
}
with open(os.path.join(out_dir, "i05", "bundle.json"), "w", encoding="utf-8") as stream:
    json.dump(i05, stream)
PY

"$evaluator" --i05 "$WORKDIR/eligible/i05" --i07 "$WORKDIR/eligible/i07.jsonl" --scenario "$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml" --output "$WORKDIR/eligible/evaluation.jsonl"
python3 - "$WORKDIR/eligible/evaluation.jsonl" <<'PY'
import json, sys
evaluation = json.load(open(sys.argv[1], encoding="utf-8"))
assert evaluation["eligibility"]["aggregate_eligible"] is True, evaluation["eligibility"]
assert evaluation["eligibility"]["invalidity_reasons"] == [], evaluation["eligibility"]
assert evaluation["structural"]["observed_result"] == "pass", evaluation["structural"]
assert evaluation["judge"]["observed_result"] == "not_applicable", evaluation["judge"]
assert evaluation["execution_oracle"]["observed_result"] == "pass", evaluation["execution_oracle"]
PY
echo "TC-071: a genuinely complete real-evaluator run reaches eligibility.aggregate_eligible: true (QA1-002)"

echo "TC-071 PASS"
