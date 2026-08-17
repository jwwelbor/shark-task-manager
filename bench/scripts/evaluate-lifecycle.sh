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
    material = bytearray()
    for current, directories, files in os.walk(root, followlinks=False):
        directories.sort()
        files.sort()
        for name in files:
            path = os.path.join(current, name)
            relative = os.path.relpath(path, root).replace(os.sep, "/")
            material.extend(relative.encode("utf-8")); material.append(0)
            with open(path, "rb") as stream:
                material.extend(stream.read())
            material.append(0)
    return hashlib.sha256(bytes(material)).hexdigest()


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
    lifecycle_rows = jsonl(args.i07)
    lifecycle = lifecycle_rows[0] if lifecycle_rows else {}
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
        reasons.append(reason("contradictory_join", "/identity/scenario_id", "I-05/I-07 scenario identity disagrees"))

    # Preserve the complete comparison surface from I-07.  These are copied
    # as evidence, never reduced to branch/HEAD labels or terminal status.
    candidate_snapshots = []
    for stage in lifecycle.get("stages", []) if isinstance(lifecycle.get("stages"), list) else []:
        if isinstance(stage, dict) and isinstance(stage.get("candidate"), dict):
            candidate = dict(stage["candidate"])
            if "identity_digest" not in candidate:
                candidate["identity_digest"] = canonical_digest({key: value for key, value in candidate.items() if key not in {"identity_digest", "snapshot_digest"}})
            if "snapshot_digest" not in candidate:
                candidate["snapshot_digest"] = canonical_digest(candidate)
            candidate_snapshots.append({"stage": stage.get("stage"), "candidate": candidate})
    workflow_policy = dict(lifecycle.get("workflow_policy") or {})
    if workflow_policy:
        workflow_policy.setdefault("workflow_policy_identity_digest", canonical_digest(workflow_policy))
    declared_content_root = (i05.get("content_root") or i05.get("shark_data_root") or identity.get("content_root")) if isinstance(i05, dict) else None
    if declared_content_root:
        derived_content_digest = content_digest(declared_content_root)
        if derived_content_digest:
            identity["shark_content_digest"] = derived_content_digest

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

    prelude = (package.get("stage_matrix") or {}).get("prelude") or {}
    judge_applicable = any(isinstance(value, dict) and value.get("applicable") is True for value in prelude.values())
    judge = {"applicability": "applicable" if judge_applicable else "not_applicable", "observed_result": "fail" if judge_applicable else "not_applicable", "invalidity_reasons": []}
    if judge_applicable and not args.judge_result:
        reasons.append(reason("missing_judge", "/judge", "applicable planning/decomposition artifacts have no judge result"))
    elif judge_applicable:
        with open(args.judge_result, encoding="utf-8") as stream:
            supplied = json.load(stream)
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
        if process.returncode or oracle_result.get("observed_result") != "pass":
            reasons.extend(oracle_result.get("invalidity_reasons") or [reason("oracle_failure", "/execution_oracle", "held-back oracle failed")])

    sources = {"i05_path": args.i05, "i05_digest": file_digest(i05_json), "i07_path": args.i07, "i07_digest": file_digest(args.i07), "replay_lineage": [item.get("snapshot_digest") for item in lifecycle.get("stages", []) if isinstance(item, dict) and item.get("snapshot_digest")]}
    review_findings = {"raw_source_ref": None, "normalized_findings": [], "derived_counts": {"truth_set_status": "truth-set-unavailable"}}
    if args.review_findings:
        with open(args.review_findings, encoding="utf-8") as stream:
            supplied_findings = json.load(stream)
        if not isinstance(supplied_findings, dict) or not isinstance(supplied_findings.get("normalized_findings"), list):
            reasons.append(reason("review_findings_malformed", "/review_findings", "normalized findings artifact is malformed"))
        else:
            review_findings = supplied_findings
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
except (OSError, ValueError, TypeError, KeyError, yaml.YAMLError, json.JSONDecodeError) as exc:
    fail_input(str(exc))
PYEOF
