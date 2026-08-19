#!/usr/bin/env bash
# TC-069: judge evidence is eligible only with complete, disjoint calibration
# provenance; no default score may be fabricated.
#
# QA2-001 (round 2, note 2586): the committed test previously exercised only
# "missing --judge-result entirely" (missing_judge). evaluate-lifecycle.sh's
# run_judge() `elif judge_applicable:` block (REQ-F-005's real calibration
# validation) implements 8 distinct invalidity-reason codes and was never
# invoked by any committed test. This file drives that branch directly
# through the real evaluator CLI for every AC-004/test-plan-declared
# partition, plus a positive control proving "only the complete calibrated
# applicable case is eligible" end to end.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
EVALUATOR="$REPO_ROOT/bench/scripts/evaluate-lifecycle.sh"
CHECKOUT_SCRIPT="$REPO_ROOT/bench/scripts/checkout-scenario-fixture.sh"
FEATURE_SCENARIO="$REPO_ROOT/bench/scenarios/packages/py-feature-recurring-tasks/package.yaml"
BUG_SCENARIO="$REPO_ROOT/bench/scenarios/packages/py-bug-due-date-boundary/package.yaml"

[[ -x "$EVALUATOR" ]] || { echo "TC-069: evaluator missing or not executable" >&2; exit 1; }
[[ -f "$FEATURE_SCENARIO" ]] || { echo "TC-069: feature scenario fixture missing" >&2; exit 1; }
[[ -f "$BUG_SCENARIO" ]] || { echo "TC-069: bug scenario fixture missing" >&2; exit 1; }

tmp="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc069.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

python3 - "$tmp" "$EVALUATOR" "$CHECKOUT_SCRIPT" "$FEATURE_SCENARIO" "$BUG_SCENARIO" <<'PY'
import copy
import hashlib
import json
import os
import pathlib
import subprocess
import sys

root, evaluator, checkout_script, feature_scenario, bug_scenario = (
    pathlib.Path(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5],
)


def sha(label):
    return hashlib.sha256(label.encode()).hexdigest()


def base_judge():
    """A complete, valid, disjoint calibrated-judge result."""
    return {
        "rubric_digest": sha("rubric-v1"),
        "prompt_digest": sha("judge-prompt-v1"),
        "judge_model": "claude-judge-1",
        "configuration": "calibration-v1",
        "calibration_set_digest": sha("calibration-set-v1"),
        "calibration_examples": [
            {"example_id": "cal-001", "label": 1},
            {"example_id": "cal-002", "label": 0},
        ],
        "evaluation_set_ids": ["eval-001", "eval-002"],
        "reference_digest": sha("reference-v1"),
        "rationale": "Candidate output matches the calibrated reference solution across every rubric dimension.",
        "score": 0.87,
        "usage": {"input_tokens": 1450, "output_tokens": 310},
        "cost_usd": 0.0421,
    }


def omit(d, *keys):
    return {k: v for k, v in d.items() if k not in keys}


CASE_COUNTER = [0]


def build_minimal_case_dir():
    """A minimal harness that forces judge_applicable=True (D01 applicable)
    without a real fixture checkout -- structural/oracle/identity reasons
    accumulate too, but judge.observed_result and judge.invalidity_reasons
    are derived solely from run_judge(), so this is sufficient to exercise
    every partition of that branch in isolation."""
    CASE_COUNTER[0] += 1
    case_dir = root / f"case-{CASE_COUNTER[0]}"
    i05_dir = case_dir / "i05"
    i05_dir.mkdir(parents=True)
    (i05_dir / "bundle.json").write_text(json.dumps({
        "roots": {"agent_fixture_checkout": "/does/not/exist"},
        "stages": [{"stage_path": "stages/code.json"}],
    }))
    i07_path = case_dir / "i07.jsonl"
    i07_path.write_text(json.dumps({
        "identity": {"run_id": "tc069", "scenario_id": "tc069-calibration"},
        "entity_graph": {"nodes": ["T"]},
        "dispatches": [{"transition": "development"}],
        "stages": [{"category": "code", "input_lineage": "input"}],
        "outcome": {"terminal": "complete"},
    }) + "\n")
    scenario_path = case_dir / "scenario.yaml"
    scenario_path.write_text(
        "scenario_id: tc069-calibration\n"
        "scenario_version: 1\n"
        "stage_matrix:\n"
        "  prelude:\n"
        "    D01: {applicable: true}\n"
        "fixture: {fixture_id: py}\n"
        "adapter: {name: python, version: \"1.0.0\"}\n"
    )
    return case_dir, i05_dir, i07_path, scenario_path


def run_judge_case(judge_result):
    case_dir, i05_dir, i07_path, scenario_path = build_minimal_case_dir()
    judge_path = case_dir / "judge.json"
    judge_path.write_text(json.dumps(judge_result))
    output_path = case_dir / "evaluation.jsonl"
    subprocess.run(
        [evaluator, "--i05", str(i05_dir), "--i07", str(i07_path), "--scenario", str(scenario_path),
         "--output", str(output_path), "--judge-result", str(judge_path)],
        capture_output=True, text=True, check=False,
    )
    return json.loads(output_path.read_text())


def assert_judge(name, record, expected):
    """expected: list of (code, path) tuples the judge sub-object must carry
    exactly (empty list means a fully valid, judge-pass result)."""
    judge = record["judge"]
    actual = {(item["code"], item["path"]) for item in judge["invalidity_reasons"]}
    assert actual == set(expected), (name, "reasons mismatch", actual, expected)
    expected_result = "fail" if expected else "pass"
    assert judge["observed_result"] == expected_result, (name, judge["observed_result"], expected_result)
    assert judge["applicability"] == "applicable", (name, judge["applicability"])


# --- AC-004 / test-plan TC-069 mutation matrix: one factor mutated per case
# against the complete, valid baseline. Every AC-004-named partition (valid
# calibrated result, disjoint sets, missing calibration, overlapping example,
# changed rubric/prompt digest, changed judge model/configuration, malformed
# usage) plus the test-plan's fuller enumeration (empty calibration set,
# duplicate example ID, held-out item present in calibration, negative/
# unknown usage, judge identity omitted, score/rationale empty and
# out-of-range, zero-cost/zero-usage boundaries, unknown judge fields,
# malformed reference-set membership, duplicate calibration IDs, missing
# reference) is named below.
CASES = [
    ("baseline_valid_disjoint_calibration", lambda d: d, []),
    ("missing_calibration_key_absent", lambda d: omit(d, "calibration_examples"),
     [("missing_calibration", "/judge/calibration_examples")]),
    ("missing_calibration_empty_list", lambda d: {**d, "calibration_examples": []},
     [("missing_calibration", "/judge/calibration_examples")]),
    ("duplicate_calibration_example_id", lambda d: {**d, "calibration_examples": [
        {"example_id": "cal-001", "label": 1}, {"example_id": "cal-001", "label": 0}]},
     [("duplicate_calibration_example", "/judge/calibration_examples")]),
    ("calibration_overlap_held_out_in_calibration", lambda d: {**d, "evaluation_set_ids": ["eval-001", "cal-001"]},
     [("calibration_overlap", "/judge/evaluation_set_ids")]),
    ("changed_rubric_digest_malformed", lambda d: {**d, "rubric_digest": "not-a-valid-digest"},
     [("malformed_digest", "/judge/rubric_digest")]),
    ("rubric_digest_missing", lambda d: omit(d, "rubric_digest"),
     [("judge_identity_missing", "/judge/rubric_digest")]),
    ("changed_prompt_digest_malformed", lambda d: {**d, "prompt_digest": "z" * 64},
     [("malformed_digest", "/judge/prompt_digest")]),
    ("calibration_set_digest_malformed", lambda d: {**d, "calibration_set_digest": "Z" * 64},
     [("malformed_digest", "/judge/calibration_set_digest")]),
    ("reference_digest_missing", lambda d: omit(d, "reference_digest"),
     [("judge_identity_missing", "/judge/reference_digest")]),
    ("reference_digest_malformed", lambda d: {**d, "reference_digest": "not-hex"},
     [("malformed_digest", "/judge/reference_digest")]),
    ("judge_model_missing", lambda d: omit(d, "judge_model"),
     [("judge_identity_missing", "/judge/judge_model")]),
    ("configuration_missing", lambda d: omit(d, "configuration"),
     [("judge_identity_missing", "/judge/judge_model")]),
    ("judge_identity_omitted_both", lambda d: omit(d, "judge_model", "configuration"),
     [("judge_identity_missing", "/judge/judge_model")]),
    ("usage_negative_input_tokens", lambda d: {**d, "usage": {"input_tokens": -5, "output_tokens": 310}},
     [("judge_usage_invalid", "/judge/usage")]),
    ("usage_missing_key", lambda d: omit(d, "usage"),
     [("judge_usage_invalid", "/judge/usage")]),
    ("usage_unknown_type", lambda d: {**d, "usage": "unknown"},
     [("judge_usage_invalid", "/judge/usage")]),
    ("usage_zero_boundary_valid", lambda d: {**d, "usage": {"input_tokens": 0, "output_tokens": 0}}, []),
    ("cost_negative", lambda d: {**d, "cost_usd": -0.01},
     [("judge_usage_invalid", "/judge/cost_usd")]),
    ("cost_zero_boundary_valid", lambda d: {**d, "cost_usd": 0}, []),
    ("score_out_of_range_high", lambda d: {**d, "score": 1.4},
     [("judge_score_invalid", "/judge/score")]),
    ("score_out_of_range_negative", lambda d: {**d, "score": -0.2},
     [("judge_score_invalid", "/judge/score")]),
    ("score_missing", lambda d: omit(d, "score"),
     [("judge_score_invalid", "/judge/score")]),
    ("score_boundary_zero_valid", lambda d: {**d, "score": 0}, []),
    ("score_boundary_one_valid", lambda d: {**d, "score": 1}, []),
    ("rationale_empty", lambda d: {**d, "rationale": ""},
     [("judge_score_invalid", "/judge/rationale")]),
    ("rationale_missing", lambda d: omit(d, "rationale"),
     [("judge_score_invalid", "/judge/rationale")]),
    ("malformed_reference_set_membership_missing_ids", lambda d: {**d, "calibration_examples": [
        {"label": 1}, {"label": 0}]},
     [("duplicate_calibration_example", "/judge/calibration_examples")]),
]

for name, mutate, expected in CASES:
    judge_input = mutate(copy.deepcopy(base_judge()))
    record = run_judge_case(judge_input)
    assert_judge(name, record, expected)
    if name == "score_missing":
        assert "score" not in record["judge"], ("no fabricated score allowed", record["judge"])
print(f"TC-069: {len(CASES)} calibration mutation cases PASS")

# --judge-result omitted entirely on an applicable-judge scenario (the sole
# partition the pre-round-2 test covered): must remain the 8th named code,
# "missing_judge" at "/judge", never silently subsumed by the mutation matrix
# above (every CASES entry always supplies --judge-result).
missing_case_dir, missing_i05_dir, missing_i07_path, missing_scenario_path = build_minimal_case_dir()
missing_output = missing_case_dir / "evaluation.jsonl"
subprocess.run(
    [evaluator, "--i05", str(missing_i05_dir), "--i07", str(missing_i07_path),
     "--scenario", str(missing_scenario_path), "--output", str(missing_output)],
    capture_output=True, text=True, check=False,
)
missing_judge_record = json.loads(missing_output.read_text())
# missing_judge is recorded only at the top-level eligibility.invalidity_reasons
# (run_judge() never overwrites judge["invalidity_reasons"] in this branch --
# it stays []), matching the pre-round-2 test's own assertion shape.
assert missing_judge_record["judge"]["applicability"] == "applicable", missing_judge_record["judge"]
assert missing_judge_record["judge"]["observed_result"] == "fail", missing_judge_record["judge"]
assert missing_judge_record["judge"]["invalidity_reasons"] == [], missing_judge_record["judge"]
assert any(item["code"] == "missing_judge" and item["path"] == "/judge" for item in missing_judge_record["eligibility"]["invalidity_reasons"]), missing_judge_record["eligibility"]
assert missing_judge_record["eligibility"]["aggregate_eligible"] is False, missing_judge_record["eligibility"]
print("TC-069: --judge-result omitted entirely is still named missing_judge")

# Unknown judge fields (test-plan fixture matrix): an unrecognized field must
# be silently dropped, never rejected and never persisted into the record.
unknown_field_input = {**base_judge(), "extra_unknown_field": "should be dropped"}
unknown_record = run_judge_case(unknown_field_input)
assert_judge("unknown_judge_fields_dropped", unknown_record, [])
assert "extra_unknown_field" not in unknown_record["judge"], unknown_record["judge"]
print("TC-069: unknown judge fields are dropped, not rejected")

# Applicability/result contradiction (test-plan fixture matrix): a caller
# cannot spoof "observed_result": "pass" / "applicability": "not_applicable"
# to bypass a genuinely invalid (missing-calibration) judge result -- the
# evaluator must derive both fields itself, never trust supplied ones.
contradiction_input = omit({**base_judge(), "observed_result": "pass", "applicability": "not_applicable"},
                            "calibration_examples")
contradiction_record = run_judge_case(contradiction_input)
assert_judge("applicability_result_contradiction", contradiction_record,
             [("missing_calibration", "/judge/calibration_examples")])
print("TC-069: caller-supplied applicability/observed_result cannot override validation")

# Non-applicable planning artifact (test-plan setup/input partition): when no
# prelude item is applicable, the judge is not_applicable and --judge-result
# is never even opened -- supply a path that does not exist to prove it.
case_dir, i05_dir, i07_path, _ = build_minimal_case_dir()
bug_output = case_dir / "evaluation.jsonl"
subprocess.run(
    [evaluator, "--i05", str(i05_dir), "--i07", str(i07_path), "--scenario", bug_scenario,
     "--output", str(bug_output), "--judge-result", str(case_dir / "does-not-exist.json")],
    capture_output=True, text=True, check=False,
)
not_applicable_record = json.loads(bug_output.read_text())
assert not_applicable_record["judge"] == {"applicability": "not_applicable", "observed_result": "not_applicable", "invalidity_reasons": []}, not_applicable_record["judge"]
print("TC-069: a non-applicable planning artifact never opens --judge-result")

# --- Positive control (AC-004's own required text: "only the complete
# calibrated applicable case is eligible"). Build a genuinely complete
# real-evaluator run -- real checkout, real reference patch, real adapter-
# backed held-back oracle, real repo-computed deep-review bundle digest,
# same pattern as TC-071's QA1-002 fix -- against a scenario whose prelude
# IS applicable (py-feature-recurring-tasks), with a complete valid
# --judge-result, and assert aggregate_eligible: true end to end.
eligible_dir = root / "eligible"
checkout_dir = eligible_dir / "checkout"
eligible_dir.mkdir(parents=True)
subprocess.run([checkout_script, "py", "964fa68e4c9e0c4e0f3756d9efd78b888c558fd9", str(checkout_dir)],
                check=True, capture_output=True, text=True)
subprocess.run(["git", "-C", str(checkout_dir), "apply", str(pathlib.Path(feature_scenario).parent / "evaluator" / "reference.patch")],
                check=True, capture_output=True, text=True)

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


def deep_review_digest(repo_root):
    hasher = hashlib.sha256()
    for relative in DEEP_REVIEW_FILES:
        path = os.path.join(repo_root, relative)
        assert os.path.isfile(path), path
        hasher.update(relative.encode()); hasher.update(b"\0")
        with open(path, "rb") as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                hasher.update(chunk)
        hasher.update(b"\0")
    return hasher.hexdigest()


def canonical_digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")).hexdigest()


repo_root = pathlib.Path(evaluator).parents[2]
bundle_digest = deep_review_digest(str(repo_root))

candidate_fields = {
    "base_commit": sha("base_commit"), "tree_digest": sha("tree_digest"),
    "binary_diff_digest": sha("binary_diff_digest"), "changed_path_digest": sha("changed_path_digest"),
    "test_suite_digest": sha("test_suite_digest"),
}
candidate = dict(candidate_fields)
candidate["dirty_untracked_manifest"] = []
candidate["identity_digest"] = canonical_digest({**candidate_fields, "dirty_untracked_manifest": []})
candidate["snapshot_digest"] = sha("snapshot_digest")

workflow_policy = {
    "enabled_gates": ["code", "review"], "gate_order": ["code", "review"],
    "reviewer": {"provider": "anthropic", "model": "claude-sonnet", "effort": "high"},
    "prompt_digest": sha("prompt_digest"), "rendered_prompt_digest": sha("rendered_prompt_digest"),
    "deep_review_bundle_digest": bundle_digest, "fixes_allowed_between_gates": True,
}
workflow_policy["workflow_policy_identity_digest"] = canonical_digest(workflow_policy)

identity = {
    "run_id": "tc069-eligible-run", "scenario_id": "py-feature-recurring-tasks", "scenario_version": "1",
    "fixture_id": "py", "fixture_digest": sha("fixture_digest"), "adapter_id": "python", "adapter_version": "1.0.0",
    "dispatch_id": "dispatch-1", "dispatch_ordinal": 1,
    "toolchain_identity": [{"key": "python_version", "value": "3.12.3"}],
    "shark_binary_digest": sha("shark_binary_digest"), "shark_content_digest": sha("shark_content_digest"),
    "rendered_prompt_digests": [sha("rendered_prompt_1")],
    "provider_identity": [{"stage": "code", "provider": "anthropic", "model": "claude-sonnet", "effort": "high"}],
    "judge_identity": {"model": "claude-judge-1", "configuration": "calibration-v1"},
    "reference_digests": [sha("reference_digest_1")], "resource_policy_digest": sha("resource_policy_digest"),
}

dispatches = [{"dispatch_id": "dispatch-1", "dispatch_ordinal": 1, "transition": "advance"}]

lifecycle = {
    "identity": identity, "outcome": {"terminal": "complete"}, "entity_graph": {"nodes": ["run"]},
    "dispatches": dispatches, "workflow_policy": workflow_policy, "review_gates": [],
    "stages": [{"stage": "code", "category": "code", "candidate": candidate, "input_lineage": []}],
}
(eligible_dir / "i07.jsonl").write_text(json.dumps(lifecycle) + "\n")

i05 = {
    "identity": {"run_id": "tc069-eligible-run", "scenario_id": "py-feature-recurring-tasks", "scenario_version": "1",
                 "dispatch_id": "dispatch-1", "dispatch_ordinal": 1},
    "roots": {"agent_fixture_checkout": str(checkout_dir)},
    "terminal_status": {"reached": True, "reached_at": "2020-01-01T00:00:00Z"},
    "stages": [{"stage_path": "stages/code.json"}],
    "dispatches": dispatches,
}
(eligible_dir / "i05").mkdir()
(eligible_dir / "i05" / "bundle.json").write_text(json.dumps(i05))

judge_result_path = eligible_dir / "judge.json"
judge_result_path.write_text(json.dumps(base_judge()))

eligible_output = eligible_dir / "evaluation.jsonl"
subprocess.run(
    [evaluator, "--i05", str(eligible_dir / "i05"), "--i07", str(eligible_dir / "i07.jsonl"),
     "--scenario", feature_scenario, "--output", str(eligible_output),
     "--judge-result", str(judge_result_path)],
    check=True, capture_output=True, text=True,
)
eligible_record = json.loads(eligible_output.read_text())
assert eligible_record["eligibility"]["aggregate_eligible"] is True, eligible_record["eligibility"]
assert eligible_record["eligibility"]["invalidity_reasons"] == [], eligible_record["eligibility"]
assert eligible_record["judge"]["applicability"] == "applicable", eligible_record["judge"]
assert eligible_record["judge"]["observed_result"] == "pass", eligible_record["judge"]
assert eligible_record["judge"]["invalidity_reasons"] == [], eligible_record["judge"]
# test-plan TC-069 Expected: "The record retains rubric/prompt/model/
# configuration/calibration/reference identity, rationale, score, usage,
# cost, and applicability" -- every supplied judge field must round-trip,
# not just the two spot-checked earlier.
valid_judge_input = base_judge()
for key, value in valid_judge_input.items():
    assert eligible_record["judge"].get(key) == value, (key, eligible_record["judge"].get(key), value)
print("TC-069: only the complete calibrated applicable case is eligible (positive control)")

# Contrast pair proving "ONLY the calibrated case is eligible": re-run the
# identical, otherwise-eligible fixture with one judge field mutated
# (calibration/evaluation-set overlap) and confirm it flips to ineligible --
# a same-fixture pair is stronger evidence than eligible-true and judge-fail
# observed as two unrelated facts.
overlap_judge_input = {**base_judge(), "evaluation_set_ids": ["eval-001", "cal-001"]}
overlap_judge_path = eligible_dir / "judge-overlap.json"
overlap_judge_path.write_text(json.dumps(overlap_judge_input))
overlap_output = eligible_dir / "evaluation-overlap.jsonl"
subprocess.run(
    [evaluator, "--i05", str(eligible_dir / "i05"), "--i07", str(eligible_dir / "i07.jsonl"),
     "--scenario", feature_scenario, "--output", str(overlap_output),
     "--judge-result", str(overlap_judge_path)],
    capture_output=True, text=True, check=False,
)
overlap_record = json.loads(overlap_output.read_text())
assert overlap_record["judge"]["observed_result"] == "fail", overlap_record["judge"]
assert any(item["code"] == "calibration_overlap" for item in overlap_record["judge"]["invalidity_reasons"]), overlap_record["judge"]
assert overlap_record["eligibility"]["aggregate_eligible"] is False, overlap_record["eligibility"]
assert any(item["code"] == "calibration_overlap" for item in overlap_record["eligibility"]["invalidity_reasons"]), overlap_record["eligibility"]
print("TC-069: the same otherwise-eligible fixture becomes ineligible on a calibration-overlap judge mutation")

print("TC-069 PASS")
PY
