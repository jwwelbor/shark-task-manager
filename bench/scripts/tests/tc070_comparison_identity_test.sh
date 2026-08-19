#!/usr/bin/env bash
# TC-070: every comparison identity component must be exact and fail closed.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/shark-tc070.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

python3 - "$TMP_DIR" "$SCRIPT_DIR/../compare-lifecycle-evaluations.sh" <<'PY'
import copy
import hashlib
import json
import pathlib
import subprocess
import sys

root = pathlib.Path(sys.argv[1])
comparator = pathlib.Path(sys.argv[2])

def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()

candidate = {
    "base_commit": "a" * 40, "tree_digest": "b" * 64,
    "binary_diff_digest": "c" * 64, "changed_path_digest": "d" * 64,
    "dirty_untracked_manifest": "e" * 64, "test_suite_digest": "f" * 64,
    "scratch_content_digest": "0" * 64,
}
candidate["snapshot_digest"] = "9" * 64
candidate["identity_digest"] = digest({key: candidate[key] for key in ["base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest"]})
policy = {
    "enabled_gates": ["qa", "deep_review"], "gate_order": ["qa", "deep_review"],
    "reviewer": {"provider": "fixture", "model": "judge-1", "effort": "low"},
    "prompt_digest": "1" * 64, "review_bundle_digest": "2" * 64,
    "rendered_prompt_digest": "1" * 64, "deep_review_bundle_digest": "2" * 64,
    "fix_policy": "none",
    "fixes_allowed_between_gates": False,
}
policy["workflow_policy_identity_digest"] = digest(policy)
identity = {
    "scenario_id": "scenario-a", "scenario_version": "1", "fixture_id": "fixture-a",
    "fixture_digest": "3" * 64, "adapter_id": "go", "adapter_version": "1",
    "shark_binary_digest": "4" * 64, "shark_content_digest": "5" * 64,
    "rendered_prompt_digest": "6" * 64, "provider": "fixture", "model": "model-a",
    "effort": "low", "judge_digest": "7" * 64, "reference_digest": "8" * 64,
    "resource_policy_digest": "9" * 64,
    "toolchain_identity": [{"key": "go", "value": "1.24"}],
    "rendered_prompt_digests": ["6" * 64],
    "provider_identity": [{"stage": "code", "provider": "fixture", "model": "model-a", "effort": "low"}],
    "judge_identity": {"model": "judge", "configuration": "default"},
    "reference_digests": ["8" * 64],
}
left = {"evaluation_id": "left", "identity": identity, "candidate_snapshots": [{"candidate": candidate}], "workflow_policy": policy, "eligibility": {"aggregate_eligible": True}}
right = copy.deepcopy(left)

def write(name, record):
    path = root / f"{name}.jsonl"
    path.write_text(json.dumps(record, sort_keys=True) + "\n")
    return path

def run(left_record, right_record, expected, field=None):
    left_path = write("left", left_record)
    right_path = write("right", right_record)
    out = root / "comparison.json"
    proc = subprocess.run([str(comparator), "--left", str(left_path), "--right", str(right_path), "--mode", "independent_frozen_candidate", "--output", str(out)], text=True, capture_output=True)
    result = json.loads(out.read_text())
    assert result["accepted"] is expected, (field, result, proc.stderr)
    assert (proc.returncode == 0) is expected, (field, proc.returncode, result)
    if field:
        assert any(item["field"] == field for item in result["divergences"]), (field, result)

def mutate(value):
    """Type-generic one-factor mutation: flip a bool, extend a string, append
    a list entry, or perturb a dict's first nested value -- so every declared
    field (not just the string/digest-shaped ones) gets a real, distinct
    mutation rather than being special-cased or skipped."""
    if isinstance(value, bool):
        return not value
    if isinstance(value, dict):
        mutated_dict = copy.deepcopy(value)
        first_key = next(iter(mutated_dict))
        nested = mutated_dict[first_key]
        mutated_dict[first_key] = mutate(nested) if not isinstance(nested, (dict, list)) else "changed"
        return mutated_dict
    if isinstance(value, list):
        return list(value) + [value[-1] if value else "changed-item"]
    if isinstance(value, str):
        return value + "-changed"
    return value

# The full declared one-factor mutation matrix (test-plan.md TC-070): every
# `identity` field the comparator enumerates, every candidate identity field,
# and every workflow-policy field -- not a representative sample. A field
# silently dropped from (or misspelled in) the comparator's field lists would
# go uncaught by a partial matrix; this loop is parameterized by the exact
# same field lists the comparator itself uses.
IDENTITY_MUTATION_FIELDS = [
    "scenario_id", "scenario_version", "fixture_id", "fixture_digest", "adapter_id", "adapter_version",
    "toolchain_identity", "shark_binary_digest", "shark_content_digest", "rendered_prompt_digest",
    "rendered_prompt_digests", "provider", "model", "effort", "provider_identity", "judge_digest",
    "judge_identity", "reference_digest", "reference_digests", "resource_policy_digest",
]
CANDIDATE_MUTATION_FIELDS = ["base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest"]
POLICY_MUTATION_FIELDS = ["enabled_gates", "gate_order", "reviewer", "prompt_digest", "rendered_prompt_digest", "review_bundle_digest", "deep_review_bundle_digest", "fixes_allowed_between_gates", "fix_policy"]

run(left, right, True)
for field in [f"identity/{f}" for f in IDENTITY_MUTATION_FIELDS] + [f"candidate/{f}" for f in CANDIDATE_MUTATION_FIELDS] + [f"workflow_policy/{f}" for f in POLICY_MUTATION_FIELDS]:
    mutated = copy.deepcopy(right)
    target = mutated
    key = field.split("/", 1)[1]
    if field.startswith("identity/"):
        target["identity"][key] = mutate(target["identity"][key])
    elif field.startswith("candidate/"):
        target["candidate_snapshots"][0]["candidate"][key] = "z" * (40 if key == "base_commit" else 64)
    else:
        target["workflow_policy"][key] = mutate(target["workflow_policy"][key])
    run(left, mutated, False, field)

branch_only = copy.deepcopy(right)
branch_only["candidate_snapshots"][0]["candidate"] = {"base_commit": "a" * 40}
run(left, branch_only, False, "candidate/tree_digest")

# Counterfactuals: equal supplied digests are still rejected when they are not
# the canonical digest of the fields they claim to identify.
false_equal_left = copy.deepcopy(left)
false_equal_right = copy.deepcopy(right)
false_equal_left["candidate_snapshots"][0]["candidate"]["identity_digest"] = "0" * 64
false_equal_right["candidate_snapshots"][0]["candidate"]["identity_digest"] = "0" * 64
false_equal_left["workflow_policy"]["workflow_policy_identity_digest"] = "0" * 64
false_equal_right["workflow_policy"]["workflow_policy_identity_digest"] = "0" * 64
left_path = write("false-equal-left", false_equal_left)
right_path = write("false-equal-right", false_equal_right)
out = root / "false-equal.json"
proc = subprocess.run([str(comparator), "--left", str(left_path), "--right", str(right_path), "--mode", "independent_frozen_candidate", "--output", str(out)], text=True, capture_output=True)
result = json.loads(out.read_text())
assert proc.returncode != 0 and result["accepted"] is False, result
assert any(item["reason"] == "identity_mismatch" for item in result["divergences"]), result

malformed = copy.deepcopy(left)
malformed["candidate_snapshots"][0]["candidate"]["identity_digest"] = "not-a-digest"
malformed_path = write("malformed", malformed)
out = root / "malformed.json"
proc = subprocess.run([str(comparator), "--left", str(malformed_path), "--right", str(right_path), "--mode", "independent_frozen_candidate", "--output", str(out)], text=True, capture_output=True)
result = json.loads(out.read_text())
assert proc.returncode != 0 and any(item["reason"] == "malformed_digest" for item in result["divergences"]), result
print("TC-070 PASS")
PY
