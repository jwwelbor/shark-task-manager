#!/usr/bin/env bash
# verify-lifecycle-run.sh <lifecycle.jsonl> --schema <i07-schema.yaml>
#
# Offline, fail-closed validation for one I-07 lifecycle run record. The
# script is intentionally independent of Shark, providers, and databases so
# contract and dry-run checks cannot spend provider credits or observe live
# workflow state.
set -euo pipefail

usage() {
	echo "usage: verify-lifecycle-run.sh <lifecycle.jsonl> --schema <i07-schema.yaml>" >&2
	exit 2
}

[[ $# -eq 3 && "$2" == "--schema" ]] || usage
run_path="$1"
schema_path="$3"
[[ -f "$run_path" ]] || { echo "verify-lifecycle-run: run not found: $run_path" >&2; exit 2; }
[[ -f "$schema_path" ]] || { echo "verify-lifecycle-run: schema not found: $schema_path" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || { echo "verify-lifecycle-run: python3 not found" >&2; exit 2; }

python3 - "$run_path" "$schema_path" <<'PYEOF'
import hashlib
import json
import re
import sys

try:
    import yaml
except ImportError as exc:
    print(f"verify-lifecycle-run: PyYAML is required: {exc}", file=sys.stderr)
    sys.exit(2)

run_path, schema_path = sys.argv[1:3]
DIGEST = re.compile(r"^[0-9a-f]{64}$")


class ContractError(Exception):
    def __init__(self, kind, path, detail):
        super().__init__(detail)
        self.kind = kind
        self.path = path
        self.detail = detail


def fail(kind, path, detail):
    raise ContractError(kind, path, detail)


def load_schema():
    try:
        with open(schema_path, encoding="utf-8") as stream:
            schema = yaml.safe_load(stream)
    except (OSError, yaml.YAMLError) as exc:
        print(f"verify-lifecycle-run: cannot read schema {schema_path}: {exc}", file=sys.stderr)
        sys.exit(2)
    if not isinstance(schema, dict) or not schema.get("schema_version"):
        print("verify-lifecycle-run: schema must declare schema_version", file=sys.stderr)
        sys.exit(2)
    return schema


def load_run():
    records = []
    try:
        with open(run_path, encoding="utf-8") as stream:
            for line_number, raw in enumerate(stream, 1):
                if not raw.strip():
                    continue
                try:
                    records.append(json.loads(raw))
                except json.JSONDecodeError as exc:
                    fail("malformed_field", f"line[{line_number}]", f"invalid JSON: {exc.msg}")
    except OSError as exc:
        print(f"verify-lifecycle-run: cannot read run {run_path}: {exc}", file=sys.stderr)
        sys.exit(2)
    if len(records) != 1:
        fail("malformed_field", "/", f"expected exactly one lifecycle record, got {len(records)}")
    if not isinstance(records[0], dict):
        fail("malformed_field", "/", "lifecycle record must be a JSON object")
    return records[0]


def pointer_values(document, pointer):
    """Return (path, value) pairs, expanding [] segments in schema paths."""
    parts = [part for part in pointer.split("/") if part]
    current = [("", document)]
    for part in parts:
        expanded = []
        is_array = part.endswith("[]")
        key = part[:-2] if is_array else part
        for path, value in current:
            if not isinstance(value, dict) or key not in value:
                continue
            child = value[key]
            child_path = f"{path}/{key}"
            if is_array:
                if not isinstance(child, list):
                    expanded.append((child_path, child))
                else:
                    expanded.extend((f"{child_path}[{index}]", item) for index, item in enumerate(child))
            else:
                expanded.append((child_path, child))
        current = expanded
    return current


def pointer_exists(document, pointer):
    return bool(pointer_values(document, pointer))


def required_pointer_state(document, pointer):
    """Return True for present, False for missing, and None when an empty
    collection makes a wildcard child not applicable."""
    parts = [part for part in pointer.split("/") if part]

    def visit(value, remaining):
        if not remaining:
            return True
        if not isinstance(value, dict):
            return False
        part = remaining[0]
        is_array = part.endswith("[]")
        key = part[:-2] if is_array else part
        if key not in value:
            return False
        child = value[key]
        if not is_array:
            return visit(child, remaining[1:])
        if not isinstance(child, list):
            return False
        if not child:
            return None
        states = [visit(item, remaining[1:]) for item in child]
        if any(state is False for state in states):
            return False
        if all(state is None for state in states):
            return None
        return True

    return visit(document, parts)


def type_matches(value, type_name):
    if type_name == "string":
        return isinstance(value, str)
    if type_name == "digest":
        return isinstance(value, str) and bool(DIGEST.fullmatch(value))
    if type_name == "object":
        return isinstance(value, dict)
    if type_name == "array":
        return isinstance(value, list)
    if type_name == "boolean":
        return isinstance(value, bool)
    if type_name == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    if type_name == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if type_name == "nullable_string":
        return value is None or isinstance(value, str)
    return True


def validate_declared_fields(record, schema):
    for pointer in schema.get("required_fields", []):
        state = required_pointer_state(record, pointer)
        if state is None:
            continue
        if state is False:
            kind = "artifact_consumption_record_missing" if pointer.endswith("/consumers") else "malformed_field"
            fail(kind, pointer, "artifact consumption evidence is missing" if kind != "malformed_field" else "required field is missing")
        values = pointer_values(record, pointer)
        for actual_path, value in values:
            if value is None:
                if pointer == "/outcome/reason" and record.get("outcome", {}).get("terminal") != "complete":
                    continue
                if schema.get("properties", {}).get(pointer) == "nullable_string":
                    continue
                fail("malformed_field", actual_path, "required field is null")
            if isinstance(value, str) and not value:
                if pointer == "/outcome/reason" and record.get("outcome", {}).get("terminal") != "complete":
                    continue
                fail("malformed_field", actual_path, "required field is empty")
            type_name = schema.get("properties", {}).get(pointer)
            if type_name and not type_matches(value, type_name):
                fail("malformed_field", actual_path, f"expected {type_name}, observed {type(value).__name__}")


def validate_vocabulary(value, vocabulary, path, kind="unsupported_outcome"):
    if value not in vocabulary:
        fail(kind, path, f"unsupported value {value!r}; allowed values are {sorted(vocabulary)!r}")


def canonical_digest(value):
    encoded = json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def validate_record(record, schema):
    allowed_top_level = set(schema.get("top_level_fields", []))
    for field in record:
        if field not in allowed_top_level:
            fail("malformed_field", f"/{field}", "unexpected top-level field")
    declared_version = schema["schema_version"]
    identity = record["identity"]
    if identity["schema_version"] != declared_version:
        fail("unsupported_schema_version", "/identity/schema_version", f"{identity['schema_version']!r} does not match {declared_version!r}")

    graph = record["entity_graph"]
    validate_vocabulary(graph["resolved_via"], schema.get("resolved_via", []), "/entity_graph/resolved_via")
    for root_name in ("agent_fixture_checkout", "scratch_shark_project", "evaluator_only"):
        if root_name not in identity["roots"]:
            fail("identity_mismatch", "/identity/roots", f"missing required root {root_name!r}")

    terminal_outcomes = schema.get("terminal_outcome", [])
    validate_vocabulary(record["outcome"]["terminal"], terminal_outcomes, "/outcome/terminal")
    if record["outcome"]["terminal"] == "complete":
        if record["outcome"]["publication_eligible"] is not True:
            fail("publication_eligible_conflict", "/outcome/publication_eligible", "complete run must be publication eligible")
    else:
        if record["outcome"]["publication_eligible"] is not False:
            fail("publication_eligible_conflict", "/outcome/publication_eligible", "stop outcome must set publication_eligible=false")
        if not record["outcome"]["reason"].strip():
            fail("missing_stop_reason", "/outcome/reason", "stop outcome requires a non-empty reason")

    dispatches = record["dispatches"]
    ordinals = []
    for index, dispatch in enumerate(dispatches):
        path = f"/dispatches[{index}]"
        ordinal = dispatch["ordinal"]
        if ordinal in ordinals:
            fail("duplicate_dispatch_ordinal", f"{path}/ordinal", f"ordinal {ordinal} occurs more than once")
        if ordinals and ordinal <= ordinals[-1]:
            fail("non_monotonic_dispatch_ordinal", f"{path}/ordinal", f"ordinal {ordinal} is not greater than {ordinals[-1]}")
        ordinals.append(ordinal)
        response = dispatch["response"]
        validate_vocabulary(dispatch["outcome"], schema.get("dispatch_outcome", []), f"{path}/outcome")
        if "resolved_via" in response:
            validate_vocabulary(response["resolved_via"], schema.get("resolved_via", []), f"{path}/response/resolved_via", "malformed_field")
        if response["entity_key"] != dispatch["requested_key"] and dispatch["requested_key"] != graph["root_key"]:
            fail("identity_mismatch", f"{path}/response/entity_key", "returned entity does not match requested key")
        if not response["model"].strip() or not response["provider"].strip():
            fail("missing_usage_or_model", f"{path}/response", "response must name provider and model")
        if not DIGEST.fullmatch(response["prompt_sha256"]):
            fail("malformed_field", f"{path}/response/prompt_sha256", "prompt_sha256 is not a lowercase SHA-256 digest")
        if response["prompt_bytes"] <= 0:
            fail("malformed_field", f"{path}/response/prompt_bytes", "prompt_bytes must be positive")
        evidence = dispatch["evidence_refs"]
        if evidence["prompt_sha256"] != response["prompt_sha256"] or evidence["prompt_bytes"] != response["prompt_bytes"]:
            fail("prompt_digest_mismatch", f"{path}/evidence_refs", "prompt_sha256 or prompt_bytes disagrees with the keyed response")

    stages_by_ordinal = {stage["dispatch_ordinal"]: stage for stage in record["stages"]}
    for index, stage in enumerate(record["stages"]):
        path = f"/stages[{index}]"
        validate_vocabulary(stage["category"], schema.get("stage_category", []), f"{path}/category", "malformed_field")
        usage = stage["usage"]
        if not isinstance(usage.get("model"), str) or not usage["model"].strip() or not isinstance(usage.get("provider"), str) or not usage["provider"].strip():
            fail("missing_usage_or_model", f"{path}/usage", "stage usage must name provider and model")
        candidate = stage["candidate"]
        expected_identity = canonical_digest({
            key: value for key, value in candidate.items()
            if key not in {"identity_digest", "snapshot_digest"}
        })
        if candidate["identity_digest"] != expected_identity:
            fail("identity_mismatch", f"{path}/candidate/identity_digest", "candidate identity digest does not match its identity components")
        if candidate["snapshot_digest"] != stage["evidence_refs"]["candidate_snapshot_digest"]:
            fail("candidate_snapshot_mismatch", f"{path}/evidence_refs/candidate_snapshot_digest", "candidate snapshot digest disagrees with the stage snapshot")
        for artifact_index, artifact in enumerate(stage["artifacts"]):
            if "consumers" not in artifact:
                fail("artifact_consumption_record_missing", f"{path}/artifacts[{artifact_index}]/consumers", "artifact consumption evidence is missing; consumers: [] is the explicit empty value")

    if set(stages_by_ordinal) != set(ordinals):
        fail("identity_mismatch", "/stages/dispatch_ordinal", "stage and dispatch ordinal sets disagree")

    policy = record["workflow_policy"]
    if not isinstance(policy["reviewer"].get("provider"), str) or not policy["reviewer"].get("provider") or not policy["reviewer"].get("model"):
        fail("missing_usage_or_model", "/workflow_policy/reviewer", "reviewer policy must name provider and model")
    for index, gate in enumerate(record["review_gates"]):
        validate_vocabulary(gate["state"], schema.get("gate_state", []), f"/review_gates[{index}]/state", "malformed_field")

    return {
        "result": "accepted",
        "schema_version": declared_version,
        "run_id": identity["run_id"],
        "publication_eligible": record["outcome"]["publication_eligible"],
        "dispatch_count": len(dispatches),
    }


schema = load_schema()
try:
    record = load_run()
    validate_declared_fields(record, schema)
    verdict = validate_record(record, schema)
    print(json.dumps(verdict, sort_keys=True, separators=(",", ":")))
except ContractError as exc:
    print(f"verify-lifecycle-run: {exc.kind}: {exc.path}: {exc.detail}", file=sys.stderr)
    sys.exit(1)
PYEOF
