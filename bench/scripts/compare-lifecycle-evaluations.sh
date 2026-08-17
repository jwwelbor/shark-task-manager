#!/usr/bin/env bash
# TC-070/TC-075: fail-closed comparison of retained I-08 evaluations.
set -euo pipefail
exec python3 - "$@" <<'PY'
import argparse
import hashlib
import json
import re
import sys
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--left", required=True)
parser.add_argument("--right", required=True)
parser.add_argument("--mode", required=True)
parser.add_argument("--output", required=True)
args = parser.parse_args()

MODES = {"independent_frozen_candidate", "sequential_delivery"}
CANDIDATE_FIELDS = ("base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest")
POLICY_DIGEST_FIELDS = ("prompt_digest", "rendered_prompt_digest", "review_bundle_digest", "deep_review_bundle_digest", "workflow_policy_identity_digest")
DIGEST_RE = re.compile(r"^[0-9a-f]{64}$")
COMMIT_RE = re.compile(r"^[0-9a-f]{40}$")

def load_record(path):
    rows = [json.loads(line) for line in Path(path).read_text(encoding="utf-8").splitlines() if line.strip()]
    if len(rows) != 1:
        raise ValueError(f"{path}: exactly one evaluation record is required")
    if not isinstance(rows[0], dict):
        raise ValueError(f"{path}: evaluation record must be an object")
    return rows[0]

def value(record, path):
    current = record
    for part in path.split("/"):
        if not isinstance(current, dict) or part not in current:
            return None
        current = current[part]
    return current

def canonical_digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")).hexdigest()

def validate_record(record, side, divergences):
    snapshots = record.get("candidate_snapshots")
    if not isinstance(snapshots, list) or not snapshots:
        divergences.append({"field": f"{side}/candidate_snapshots", "left": None, "right": "non-empty list", "reason": "identity_missing"})
    else:
        for index, snapshot in enumerate(snapshots):
            candidate = snapshot.get("candidate") if isinstance(snapshot, dict) else None
            prefix = f"{side}/candidate_snapshots[{index}]/candidate"
            if not isinstance(candidate, dict):
                divergences.append({"field": prefix, "left": None, "right": "object", "reason": "identity_missing"})
                continue
            missing = [field for field in CANDIDATE_FIELDS if candidate.get(field) in (None, "")]
            for field in missing:
                divergences.append({"field": f"{prefix}/{field}", "left": None, "right": "required", "reason": "identity_missing"})
            if not isinstance(candidate.get("base_commit"), str) or not COMMIT_RE.fullmatch(candidate.get("base_commit", "")):
                divergences.append({"field": f"{prefix}/base_commit", "left": candidate.get("base_commit"), "right": "40 lowercase hex", "reason": "malformed_digest"})
            for field in CANDIDATE_FIELDS[1:]:
                if field in candidate and (not isinstance(candidate[field], str) or not DIGEST_RE.fullmatch(candidate[field])):
                    divergences.append({"field": f"{prefix}/{field}", "left": candidate.get(field), "right": "64 lowercase hex", "reason": "malformed_digest"})
            supplied = candidate.get("identity_digest")
            expected = canonical_digest({field: candidate[field] for field in CANDIDATE_FIELDS if field in candidate}) if not missing else None
            if not isinstance(supplied, str) or not DIGEST_RE.fullmatch(supplied):
                divergences.append({"field": f"{prefix}/identity_digest", "left": supplied, "right": "64 lowercase hex", "reason": "malformed_digest"})
            elif expected and supplied != expected:
                divergences.append({"field": f"{prefix}/identity_digest", "left": supplied, "right": expected, "reason": "identity_mismatch"})
            snapshot_digest = candidate.get("snapshot_digest")
            if not isinstance(snapshot_digest, str) or not DIGEST_RE.fullmatch(snapshot_digest):
                divergences.append({"field": f"{prefix}/snapshot_digest", "left": snapshot_digest, "right": "64 lowercase hex", "reason": "malformed_digest"})

    policy = record.get("workflow_policy")
    if not isinstance(policy, dict):
        divergences.append({"field": f"{side}/workflow_policy", "left": None, "right": "object", "reason": "identity_missing"})
    else:
        required = ("enabled_gates", "gate_order", "reviewer", "prompt_digest", "rendered_prompt_digest", "deep_review_bundle_digest", "fixes_allowed_between_gates", "workflow_policy_identity_digest")
        for field in required:
            if field not in policy or policy[field] in (None, "", []):
                divergences.append({"field": f"{side}/workflow_policy/{field}", "left": None, "right": "required", "reason": "identity_missing"})
        reviewer = policy.get("reviewer")
        if not isinstance(reviewer, dict) or not all(reviewer.get(field) for field in ("provider", "model", "effort")):
            divergences.append({"field": f"{side}/workflow_policy/reviewer", "left": reviewer, "right": "provider/model/effort", "reason": "identity_missing"})
        for field in POLICY_DIGEST_FIELDS:
            if field in policy and (not isinstance(policy[field], str) or not DIGEST_RE.fullmatch(policy[field])):
                divergences.append({"field": f"{side}/workflow_policy/{field}", "left": policy.get(field), "right": "64 lowercase hex", "reason": "malformed_digest"})
        supplied = policy.get("workflow_policy_identity_digest")
        expected = canonical_digest({key: value for key, value in policy.items() if key != "workflow_policy_identity_digest"})
        if isinstance(supplied, str) and DIGEST_RE.fullmatch(supplied) and supplied != expected:
            divergences.append({"field": f"{side}/workflow_policy/workflow_policy_identity_digest", "left": supplied, "right": expected, "reason": "identity_mismatch"})

def candidate(record):
    snapshots = record.get("candidate_snapshots")
    if not isinstance(snapshots, list) or not snapshots:
        return None
    first = snapshots[0]
    return first.get("candidate") if isinstance(first, dict) and isinstance(first.get("candidate"), dict) else first

def digest(record, path):
    current = value(record, path)
    return current

def main():
    divergences = []
    if args.mode not in MODES:
        divergences.append({"field": "comparison/mode", "left": args.mode, "right": None, "reason": "unsupported_mode"})
        left = right = {}
    else:
        left, right = load_record(args.left), load_record(args.right)
        for side, record in (("left", left), ("right", right)):
            eligibility = record.get("eligibility")
            if not isinstance(eligibility, dict) or eligibility.get("aggregate_eligible") is not True:
                divergences.append({"field": f"{side}/eligibility/aggregate_eligible", "left": eligibility.get("aggregate_eligible") if isinstance(eligibility, dict) else None, "right": True, "reason": "evaluation_ineligible"})

    identity_fields = [
        "scenario_id", "scenario_version", "fixture_id", "fixture_digest", "adapter_id", "adapter_version",
        "toolchain_identity", "shark_binary_digest", "shark_content_digest", "rendered_prompt_digest",
        "rendered_prompt_digests", "provider", "model", "effort", "provider_identity", "judge_digest",
        "judge_identity", "reference_digest", "reference_digests", "resource_policy_digest",
    ]
    for field in identity_fields:
        left_value = digest(left, "identity/" + field)
        right_value = digest(right, "identity/" + field)
        if left_value is None or right_value is None:
            divergences.append({"field": "identity/" + field, "left": left_value, "right": right_value, "reason": "identity_missing"})
        elif left_value != right_value:
            divergences.append({"field": "identity/" + field, "left": left_value, "right": right_value, "reason": "identity_mismatch"})

    validate_record(left, "left", divergences)
    validate_record(right, "right", divergences)
    candidate_fields = [*CANDIDATE_FIELDS, "identity_digest"]
    left_candidate, right_candidate = candidate(left), candidate(right)
    for field in candidate_fields:
        left_value = left_candidate.get(field) if isinstance(left_candidate, dict) else None
        right_value = right_candidate.get(field) if isinstance(right_candidate, dict) else None
        if left_value is None or right_value is None:
            divergences.append({"field": "candidate/" + field, "left": left_value, "right": right_value, "reason": "identity_missing"})
        elif left_value != right_value:
            divergences.append({"field": "candidate/" + field, "left": left_value, "right": right_value, "reason": "identity_mismatch"})

    policy_fields = [
        "enabled_gates", "gate_order", "reviewer", "prompt_digest", "rendered_prompt_digest",
        "review_bundle_digest", "deep_review_bundle_digest", "fixes_allowed_between_gates", "fix_policy",
    ]
    for field in policy_fields:
        left_value = value(left, "workflow_policy/" + field)
        right_value = value(right, "workflow_policy/" + field)
        if left_value is None or right_value is None:
            divergences.append({"field": "workflow_policy/" + field, "left": left_value, "right": right_value, "reason": "identity_missing"})
        elif left_value != right_value:
            divergences.append({"field": "workflow_policy/" + field, "left": left_value, "right": right_value, "reason": "identity_mismatch"})

    left_policy_digest = value(left, "workflow_policy/workflow_policy_identity_digest")
    right_policy_digest = value(right, "workflow_policy/workflow_policy_identity_digest")
    if left_policy_digest != right_policy_digest:
        divergences.append({"field": "workflow_policy/workflow_policy_identity_digest", "left": left_policy_digest, "right": right_policy_digest, "reason": "identity_mismatch"})

    if args.mode == "sequential_delivery":
        for side, record in (("left", left), ("right", right)):
            lineage = record.get("candidate_snapshots")
            if side == "right" and (not isinstance(lineage, list) or len(lineage) < 2):
                divergences.append({"field": f"{side}/candidate_snapshots", "left": len(lineage) if isinstance(lineage, list) else None, "right": 2, "reason": "candidate_lineage_missing"})
    elif args.mode == "independent_frozen_candidate":
        for side, record in (("left", left), ("right", right)):
            lineage = record.get("candidate_snapshots")
            if not isinstance(lineage, list) or len(lineage) != 1:
                divergences.append({"field": f"{side}/candidate_snapshots", "left": len(lineage) if isinstance(lineage, list) else None, "right": 1, "reason": "intervening_fix_detected"})

    left_findings = {item.get("f09_finding_id") for item in (value(left, "review_findings/normalized_findings") or []) if isinstance(item, dict) and item.get("final_disposition") == "confirmed"}
    right_findings = [item for item in (value(right, "review_findings/normalized_findings") or []) if isinstance(item, dict) and item.get("final_disposition") == "confirmed"]
    newly_confirmed = sorted({item.get("f09_finding_id") for item in right_findings if item.get("f09_finding_id") not in left_findings})
    intervening = right.get("candidate_snapshots") if args.mode == "sequential_delivery" else []

    accepted = not divergences
    result = {
        "schema_version": "1.0",
        "mode": args.mode,
        "left_evaluation_id": left.get("evaluation_id"),
        "right_evaluation_id": right.get("evaluation_id"),
        "accepted": accepted,
        "comparison": {"accepted": accepted, "quality_delta": None, "causal_claim": None, "newly_confirmed_findings": newly_confirmed if args.mode == "sequential_delivery" else [], "intervening_candidates": intervening},
        "divergences": divergences,
    }
    Path(args.output).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output).write_text(json.dumps(result, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
    if not accepted:
        print("comparison_divergence fields=" + ",".join(item["field"] for item in divergences), file=sys.stderr)
    return 0 if accepted else 1

try:
    raise SystemExit(main())
except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
    print(f"comparison_invalid: {exc}", file=sys.stderr)
    raise SystemExit(2)
PY
