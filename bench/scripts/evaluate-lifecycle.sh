#!/usr/bin/env bash
# TC-067 / TC-069: offline I-08 lifecycle evaluator.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
exec python3 - "$BENCH_DIR" "$SCRIPT_DIR/run-heldback-oracle.sh" "$@" <<'PYEOF'
import argparse
import hashlib
import json
import os
import subprocess
import sys
import tempfile

import yaml

bench_dir, oracle = sys.argv[1:3]
parser = argparse.ArgumentParser()
parser.add_argument("--i05", required=True)
parser.add_argument("--i07", required=True)
parser.add_argument("--scenario", required=True)
parser.add_argument("--output", required=True)
parser.add_argument("--judge-result")
parser.add_argument("--review-findings")
args = parser.parse_args(sys.argv[3:])


def file_digest(path):
    with open(path, "rb") as stream:
        return hashlib.sha256(stream.read()).hexdigest()


def jsonl(path):
    with open(path, encoding="utf-8") as stream:
        return [json.loads(line) for line in stream if line.strip()]


def digest(value):
    if not isinstance(value, str) or len(value) != 64 or value != value.lower():
        return False
    try:
        int(value, 16)
    except ValueError:
        return False
    return True


def canonical_digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")).hexdigest()


def content_digest(root):
    root = os.path.abspath(root)
    if not os.path.isdir(root):
        return None
    hasher = hashlib.sha256()
    for current, directories, files in os.walk(root, followlinks=False):
        directories.sort()
        files.sort()
        for name in files:
            path = os.path.join(current, name)
            relative = os.path.relpath(path, root).replace(os.sep, "/")
            hasher.update(relative.encode("utf-8")); hasher.update(b"\0")
            with open(path, "rb") as stream:
                for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                    hasher.update(chunk)
            hasher.update(b"\0")
    return hasher.hexdigest()


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
        if not os.path.isfile(path):
            return None
        hasher.update(relative.encode("utf-8")); hasher.update(b"\0")
        with open(path, "rb") as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b""):
                hasher.update(chunk)
        hasher.update(b"\0")
    return hasher.hexdigest()


def reason(code, path, detail):
    return {"code": code, "path": path, "detail": str(detail)[:240]}


def write(record):
    os.makedirs(os.path.dirname(os.path.abspath(args.output)), exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as stream:
        json.dump(record, stream, sort_keys=True, separators=(",", ":"))
        stream.write("\n")
    eligible = record["eligibility"]["aggregate_eligible"]
    if not eligible:
        print("evaluation_invalid evaluation_id=" + record["evaluation_id"] + " reasons=" + ",".join(item["code"] for item in record["eligibility"]["invalidity_reasons"]), file=sys.stderr)
    raise SystemExit(0 if eligible else 1)


def fail_input(detail):
    write({"schema_version": "1.0", "evaluation_id": "invalid-input", "identity": {}, "source_artifacts": {}, "structural": {"applicability": "applicable", "checks": [], "observed_result": "fail"}, "judge": {"applicability": "applicable", "observed_result": "fail", "invalidity_reasons": []}, "execution_oracle": {"observed_result": "not_run", "invalidity_reasons": []}, "eligibility": {"structural_valid": False, "judge_valid": False, "oracle_valid": False, "aggregate_eligible": False, "publication_eligible": False, "invalidity_reasons": [reason("source_malformed", "/input", detail)]}})


try:
    with open(args.scenario, encoding="utf-8") as stream:
        package = yaml.safe_load(stream)
    if not isinstance(package, dict):
        fail_input("scenario package is not an object")
    i05_json = os.path.join(args.i05, "bundle.json") if os.path.isdir(args.i05) else args.i05
    with open(i05_json, encoding="utf-8") as stream:
        i05 = json.load(stream)
    if not isinstance(i05, dict):
        fail_input("I-05 bundle must be a JSON object")
    if i05.get("roots") is not None and not isinstance(i05.get("roots"), dict):
        fail_input("I-05 roots must be an object")
    if package.get("stage_matrix") is not None and not isinstance(package.get("stage_matrix"), dict):
        fail_input("scenario stage_matrix must be an object")
    lifecycle_rows = jsonl(args.i07)
    if any(not isinstance(row, dict) for row in lifecycle_rows):
        fail_input("every I-07 JSONL record must be an object")
    lifecycle = lifecycle_rows[0] if lifecycle_rows else {}
    if not isinstance(lifecycle.get("identity"), (dict, type(None))):
        fail_input("I-07 identity must be an object")
    for key in ("fixture", "adapter"):
        if key in package and not isinstance(package[key], (dict, type(None))):
            fail_input(f"scenario {key} must be an object")
    if lifecycle.get("stages") is not None and not isinstance(lifecycle.get("stages"), list):
        fail_input("I-07 stages must be an array")
    run_id = ((lifecycle.get("identity") or {}).get("run_id")) or "unknown-run"
    evaluation_id = run_id + "-" + file_digest(args.i07)[:12]
    reasons = []
    if len(lifecycle_rows) == 0:
        reasons.append(reason("missing_join", "/i07", "I-07 lifecycle record is missing"))
    elif len(lifecycle_rows) != 1:
        reasons.append(reason("duplicate_join", "/i07", "exactly one I-07 lifecycle record is required"))
    scenario_id = package.get("scenario_id")
    identity = dict(lifecycle.get("identity") or {})
    identity.setdefault("scenario_id", scenario_id)
    identity.setdefault("scenario_version", str(package.get("scenario_version", "")))
    identity.setdefault("fixture_id", (package.get("fixture") or {}).get("fixture_id"))
    identity.setdefault("adapter_id", (package.get("adapter") or {}).get("name"))
    identity.setdefault("adapter_version", (package.get("adapter") or {}).get("version"))
    if lifecycle_rows and identity.get("scenario_id") not in {None, scenario_id}:
        reasons.append(reason("contradictory_join", "/identity/scenario_id", "I-07/scenario package identity disagrees"))

    # I-05 and I-07 are a join, not two independently plausible inputs. Keep
    # the check schema-shaped and compare every reference the producers
    # record; never infer a relationship from the filename or terminal state.
    i05_identity = dict(i05.get("identity") or {}) if isinstance(i05, dict) else {}
    i05_records = i05.get("records") or i05.get("runs") if isinstance(i05, dict) else None
    if isinstance(i05_records, list) and len(i05_records) != 1:
        reasons.append(reason("duplicate_join", "/i05/records", "I-05 must contain exactly one joined evidence record"))
    join_fields = ("run_id", "scenario_id", "scenario_version", "dispatch_id", "dispatch_ordinal")
    for field in join_fields:
        i07_value = identity.get(field)
        if i07_value is None:
            i07_value = lifecycle.get(field)
        i05_value = i05_identity.get(field)
        if i05_value is None:
            i05_value = i05.get(field) if isinstance(i05, dict) else None
        if field == "scenario_version" and i07_value is not None:
            i07_value = str(i07_value)
        if field == "scenario_version" and i05_value is not None:
            i05_value = str(i05_value)
        if i07_value is None or i05_value is None:
            reasons.append(reason("missing_join", "/join/" + field, "I-05 and I-07 must both record this join reference"))
        elif i07_value != i05_value:
            reasons.append(reason("contradictory_join", "/join/" + field, "I-05 and I-07 join references disagree"))
    i07_dispatches = lifecycle.get("dispatches")
    i05_dispatches = i05.get("dispatches") or i05.get("dispatch_refs") if isinstance(i05, dict) else None
    if i07_dispatches is None or i05_dispatches is None:
        reasons.append(reason("missing_join", "/join/dispatches", "both I-05 and I-07 must retain dispatch references"))
    elif canonical_digest(i07_dispatches) != canonical_digest(i05_dispatches):
        reasons.append(reason("contradictory_join", "/join/dispatches", "I-05 and I-07 dispatch references disagree"))

    # Preserve the complete comparison surface from I-07.  These are copied
    # as evidence, never reduced to branch/HEAD labels or terminal status.
    raw_workflow_policy = lifecycle.get("workflow_policy")
    if raw_workflow_policy is not None and not isinstance(raw_workflow_policy, dict):
        fail_input("I-07 workflow_policy must be an object")
    candidate_snapshots = []
    for stage_index, stage in enumerate(lifecycle.get("stages", []) if isinstance(lifecycle.get("stages"), list) else []):
        if not isinstance(stage, dict) or (stage.get("candidate") is not None and not isinstance(stage.get("candidate"), dict)):
            fail_input(f"I-07 stage {stage_index} and candidate must be objects")
        if not isinstance(stage, dict) or not isinstance(stage.get("candidate"), dict):
            reasons.append(reason("identity_missing", f"/stages/{stage_index}/candidate", "upstream I-07 candidate identity is required"))
            continue
        candidate = dict(stage["candidate"])
        candidate_fields = ("base_commit", "tree_digest", "binary_diff_digest", "changed_path_digest", "dirty_untracked_manifest", "test_suite_digest")
        for field in candidate_fields:
            if field not in candidate or candidate[field] in (None, ""):
                reasons.append(reason("identity_missing", f"/stages/{stage_index}/candidate/{field}", "complete upstream candidate identity is required"))
        if "identity_digest" not in candidate or "snapshot_digest" not in candidate:
            reasons.append(reason("identity_missing", f"/stages/{stage_index}/candidate", "F09 must not synthesize candidate identity digests"))
        else:
            expected_identity = canonical_digest({key: candidate[key] for key in candidate_fields if key in candidate})
            if candidate.get("identity_digest") != expected_identity:
                reasons.append(reason("identity_mismatch", f"/stages/{stage_index}/candidate/identity_digest", "upstream candidate identity digest disagrees with its six fields"))
        candidate_snapshots.append({"stage": stage.get("stage"), "candidate": candidate})
    workflow_policy = dict(raw_workflow_policy or {})
    declared_content_root = (i05.get("content_root") or i05.get("shark_data_root") or identity.get("content_root")) if isinstance(i05, dict) else None
    if declared_content_root:
        derived_content_digest = content_digest(declared_content_root)
        if derived_content_digest:
            identity["shark_content_digest"] = derived_content_digest

    # Producer-side identity validation is deliberately strict. The
    # comparator cannot repair an incomplete record after it has been emitted:
    # a single evaluation with partial policy/content identity must already be
    # ineligible for aggregation.
    required_identity = (
        "run_id", "scenario_id", "scenario_version", "fixture_id",
        "fixture_digest", "adapter_id", "adapter_version", "toolchain_identity",
        "shark_binary_digest", "shark_content_digest", "rendered_prompt_digests",
        "provider_identity", "judge_identity", "reference_digests",
        "resource_policy_digest",
    )
    for field in required_identity:
        value = identity.get(field)
        if value is None or value == "" or value == []:
            reasons.append(reason("identity_missing", "/identity/" + field, "producer must retain the complete I-08 identity surface"))
    for field in ("fixture_digest", "shark_binary_digest", "shark_content_digest", "resource_policy_digest"):
        if field in identity and not digest(identity[field]):
            reasons.append(reason("malformed_digest", "/identity/" + field, "identity digest must be lowercase SHA-256"))
    for field in ("rendered_prompt_digests", "reference_digests"):
        values = identity.get(field)
        if not isinstance(values, list) or not values or any(not digest(value) for value in values):
            reasons.append(reason("identity_missing", "/identity/" + field, "identity digest list must be non-empty and valid"))
    if not isinstance(identity.get("provider_identity"), list) or not identity["provider_identity"]:
        reasons.append(reason("identity_missing", "/identity/provider_identity", "provider/model/effort provenance is required"))
    elif any(not isinstance(item, dict) or not all(item.get(key) for key in ("stage", "provider", "model", "effort")) for item in identity["provider_identity"]):
        reasons.append(reason("identity_missing", "/identity/provider_identity", "every provider identity must name stage, provider, model, and effort"))
    if not isinstance(identity.get("toolchain_identity"), list) or not identity["toolchain_identity"]:
        reasons.append(reason("identity_missing", "/identity/toolchain_identity", "toolchain identity is required"))
    if not isinstance(identity.get("judge_identity"), dict) or not identity["judge_identity"].get("model") or not identity["judge_identity"].get("configuration"):
        reasons.append(reason("identity_missing", "/identity/judge_identity", "judge model and configuration identity are required even when the judge is not applicable"))

    required_policy = (
        "enabled_gates", "gate_order", "reviewer", "prompt_digest",
        "rendered_prompt_digest", "deep_review_bundle_digest",
        "fixes_allowed_between_gates", "workflow_policy_identity_digest",
    )
    for field in required_policy:
        value = workflow_policy.get(field)
        if value is None or value == "" or value == []:
            reasons.append(reason("identity_missing", "/workflow_policy/" + field, "producer must retain complete workflow-policy identity"))
    reviewer = workflow_policy.get("reviewer")
    if not isinstance(reviewer, dict) or not all(reviewer.get(key) for key in ("provider", "model", "effort")):
        reasons.append(reason("identity_missing", "/workflow_policy/reviewer", "reviewer provider/model/effort identity is required"))
    for field in ("prompt_digest", "rendered_prompt_digest", "deep_review_bundle_digest", "workflow_policy_identity_digest"):
        if field in workflow_policy and not digest(workflow_policy[field]):
            reasons.append(reason("malformed_digest", "/workflow_policy/" + field, "workflow-policy digest must be lowercase SHA-256"))
    expected_policy_digest = canonical_digest({key: value for key, value in workflow_policy.items() if key != "workflow_policy_identity_digest"})
    if workflow_policy.get("workflow_policy_identity_digest") and workflow_policy["workflow_policy_identity_digest"] != expected_policy_digest:
        reasons.append(reason("identity_mismatch", "/workflow_policy/workflow_policy_identity_digest", "workflow-policy digest does not match its fields"))
    expected_bundle_digest = deep_review_digest(os.path.dirname(bench_dir))
    if expected_bundle_digest is None:
        reasons.append(reason("source_missing", "/workflow_policy/deep_review_bundle_digest", "repository deep-review bundle is incomplete"))
    elif workflow_policy.get("deep_review_bundle_digest") != expected_bundle_digest:
        reasons.append(reason("identity_mismatch", "/workflow_policy/deep_review_bundle_digest", "deep-review bundle digest disagrees with repository-owned files"))

    # Deterministic structural checks are separate from status and worker
    # claims. A missing check remains a failure rather than an inferred pass.
    checks = {
        "required_artifacts": bool(i05.get("stages") or i05.get("artifacts")),
        "ownership": bool(i05.get("roots")),
        "links": bool(i05.get("access_events") or i05.get("artifacts") or lifecycle.get("stages")),
        "dependencies": bool(lifecycle.get("entity_graph")),
        "status_transitions": bool(lifecycle.get("dispatches")) and all(isinstance(item, dict) and item.get("transition") is not None for item in lifecycle.get("dispatches", [])),
        "traceability": bool(lifecycle.get("stages")) and all(isinstance(item, dict) and (item.get("input_lineage") is not None or item.get("evidence_refs") is not None) for item in lifecycle.get("stages", [])),
        "executable_task": bool(lifecycle.get("stages")) and any(item.get("category") == "code" for item in lifecycle.get("stages", []) if isinstance(item, dict)),
    }
    structural_checks = [{"check_id": key, "applicability": "applicable", "stage": "lifecycle", "entity": "run", "result": "pass" if value else "fail", "evidence_refs": ["i05", "i07"] if value else [], "reason": None if value else "required evidence is absent"} for key, value in checks.items()]
    structural_pass = bool(lifecycle_rows) and all(item["result"] == "pass" for item in structural_checks)
    structural = {"applicability": "applicable", "checks": structural_checks, "observed_result": "pass" if structural_pass else "fail"}
    if not structural_pass:
        reasons.append(reason("structural_failure", "/structural/checks", "one or more deterministic structural checks failed"))

    raw_prelude = (package.get("stage_matrix") or {}).get("prelude")
    if raw_prelude is not None and not isinstance(raw_prelude, dict):
        fail_input("scenario stage_matrix.prelude must be an object")
    prelude = raw_prelude or {}
    judge_applicable = any(isinstance(value, dict) and value.get("applicable") is True for value in prelude.values())
    judge = {"applicability": "applicable" if judge_applicable else "not_applicable", "observed_result": "fail" if judge_applicable else "not_applicable", "invalidity_reasons": []}
    if judge_applicable and not args.judge_result:
        reasons.append(reason("missing_judge", "/judge", "applicable planning/decomposition artifacts have no judge result"))
    elif judge_applicable:
        with open(args.judge_result, encoding="utf-8") as stream:
            supplied = json.load(stream)
        if not isinstance(supplied, dict):
            fail_input("judge result must be a JSON object")
        for key in ["rubric_digest", "prompt_digest", "judge_model", "configuration", "calibration_set_digest", "calibration_examples", "evaluation_set_ids", "reference_digest", "rationale", "score", "usage", "cost_usd"]:
            if key in supplied:
                judge[key] = supplied[key]
        judge_reasons = []
        for key in ["rubric_digest", "prompt_digest", "calibration_set_digest", "reference_digest"]:
            if not digest(supplied.get(key)):
                judge_reasons.append(reason("judge_identity_missing" if not supplied.get(key) else "malformed_digest", "/judge/" + key, "judge provenance digest is missing or malformed"))
        if not supplied.get("judge_model") or not supplied.get("configuration"):
            judge_reasons.append(reason("judge_identity_missing", "/judge/judge_model", "judge model/configuration identity is required"))
        examples = supplied.get("calibration_examples")
        example_ids = [item.get("example_id") for item in examples] if isinstance(examples, list) else []
        if not examples:
            judge_reasons.append(reason("missing_calibration", "/judge/calibration_examples", "calibration examples are required"))
        if len(example_ids) != len(set(example_ids)):
            judge_reasons.append(reason("duplicate_calibration_example", "/judge/calibration_examples", "calibration example IDs must be unique"))
        if set(supplied.get("evaluation_set_ids") or []).intersection(example_ids):
            judge_reasons.append(reason("calibration_overlap", "/judge/evaluation_set_ids", "calibration and evaluation sets overlap"))
        if not isinstance(supplied.get("rationale"), str) or not supplied["rationale"].strip():
            judge_reasons.append(reason("judge_score_invalid", "/judge/rationale", "rationale is required"))
        if not isinstance(supplied.get("score"), (int, float)) or not 0 <= supplied["score"] <= 1:
            judge_reasons.append(reason("judge_score_invalid", "/judge/score", "score must be between 0 and 1"))
        usage = supplied.get("usage")
        if not isinstance(usage, dict) or any(not isinstance(usage.get(key), int) or usage.get(key) < 0 for key in ["input_tokens", "output_tokens"]):
            judge_reasons.append(reason("judge_usage_invalid", "/judge/usage", "usage token counts must be non-negative integers"))
        if not isinstance(supplied.get("cost_usd"), (int, float)) or supplied["cost_usd"] < 0:
            judge_reasons.append(reason("judge_usage_invalid", "/judge/cost_usd", "cost must be non-negative"))
        judge["observed_result"] = "pass" if not judge_reasons else "fail"
        judge["invalidity_reasons"] = judge_reasons
        reasons.extend(judge_reasons)

    checkout = (i05.get("roots") or {}).get("agent_fixture_checkout")
    if checkout is not None and not isinstance(checkout, str):
        fail_input("I-05 roots.agent_fixture_checkout must be a path string")
    if not checkout or not os.path.isdir(checkout):
        oracle_result = {"observed_result": "not_run", "invalidity_reasons": [reason("missing_oracle", "/execution_oracle", "agent fixture checkout is unavailable")]}
        reasons.append(reason("missing_oracle", "/execution_oracle", "agent fixture checkout is unavailable"))
    else:
        oracle_output = args.output + ".oracle.json"
        process = subprocess.run([oracle, "--scenario", args.scenario, "--i07", args.i07, "--stage-bundle", args.i05, "--checkout", checkout, "--output", oracle_output], capture_output=True, text=True)
        try:
            with open(oracle_output, encoding="utf-8") as stream:
                oracle_result = json.load(stream)
        except (OSError, json.JSONDecodeError):
            oracle_result = {"observed_result": "not_run", "invalidity_reasons": [reason("missing_oracle", "/execution_oracle", "oracle did not produce a result")]}
        if not isinstance(oracle_result, dict):
            oracle_result = {"observed_result": "not_run", "invalidity_reasons": [reason("source_malformed", "/execution_oracle", "oracle result must be an object")]}
        if process.returncode or oracle_result.get("observed_result") != "pass":
            reasons.extend(oracle_result.get("invalidity_reasons") or [reason("oracle_failure", "/execution_oracle", "held-back oracle failed")])

    sources = {"i05_path": args.i05, "i05_digest": file_digest(i05_json), "i07_path": args.i07, "i07_digest": file_digest(args.i07), "replay_lineage": [item.get("snapshot_digest") for item in lifecycle.get("stages", []) if isinstance(item, dict) and item.get("snapshot_digest")]}
    # Normalization is an evaluator-owned boundary. A caller may not inject a
    # pre-adjudicated normalized artifact and thereby bypass raw I-07 review
    # evidence and the seeded truth-set adjudication.
    review_findings = {"raw_source_ref": None, "normalized_findings": [], "derived_counts": {"truth_set_status": "truth-set-unavailable"}}
    if args.review_findings:
        reasons.append(reason("review_findings_bypass", "/review_findings", "caller-supplied normalized findings are not accepted; evaluator owns normalization"))
    normalizer = os.path.join(os.path.dirname(oracle), "normalize-review-findings.sh")
    with tempfile.TemporaryDirectory(prefix="f09-normalize-") as normalize_dir:
        normalized_path = os.path.join(normalize_dir, "findings.json")
        process = subprocess.run([normalizer, "--i07", args.i07, "--output", normalized_path], capture_output=True, text=True)
        if process.returncode != 0:
            reasons.append(reason("review_findings_malformed", "/review_findings/raw_source", "evaluator-owned normalization failed: " + process.stderr.strip()))
        else:
            try:
                with open(normalized_path, encoding="utf-8") as stream:
                    normalized = json.load(stream)
                if not isinstance(normalized, dict) or not isinstance(normalized.get("normalized_findings"), list):
                    raise ValueError("normalizer output has no normalized_findings list")
                normalized["raw_source_ref"] = {"i07_path": args.i07, "i07_digest": file_digest(args.i07)}
                review_findings = normalized
            except (OSError, ValueError, json.JSONDecodeError) as exc:
                reasons.append(reason("review_findings_malformed", "/review_findings", str(exc)))
    identity["source_identity_digest"] = canonical_digest(identity)
    unique = []
    seen = set()
    for item in reasons:
        key = (item["code"], item["path"])
        if key not in seen:
            unique.append(item)
            seen.add(key)
    aggregate = not unique and structural["observed_result"] == "pass" and judge["observed_result"] in {"pass", "not_applicable"} and oracle_result.get("observed_result") == "pass"
    write({"schema_version": "1.0", "evaluation_id": evaluation_id, "identity": identity, "source_artifacts": sources, "structural": structural, "judge": judge, "execution_oracle": oracle_result, "review_findings": review_findings, "candidate_snapshots": candidate_snapshots, "workflow_policy": workflow_policy, "comparison": {"mode": None, "accepted": False, "quality_delta": None}, "eligibility": {"structural_valid": structural["observed_result"] == "pass", "judge_valid": judge["observed_result"] in {"pass", "not_applicable"}, "oracle_valid": oracle_result.get("observed_result") == "pass", "aggregate_eligible": aggregate, "publication_eligible": aggregate, "invalidity_reasons": unique}})
except (OSError, ValueError, TypeError, AttributeError, IndexError, KeyError, yaml.YAMLError, json.JSONDecodeError) as exc:
    fail_input(str(exc))
PYEOF
