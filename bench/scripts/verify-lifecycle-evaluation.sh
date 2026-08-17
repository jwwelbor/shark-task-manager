#!/usr/bin/env bash
# verify-lifecycle-evaluation.sh <evaluation.jsonl> --schema <i08-schema.yaml>
# Offline, streaming I-08 verifier. It never invokes Shark, a provider, or an
# adapter and emits only a bounded verdict and retained invalidity inventory.
set -euo pipefail

usage() {
  echo "usage: verify-lifecycle-evaluation.sh <evaluation.jsonl> --schema <i08-schema.yaml>" >&2
  exit 2
}
[[ $# -eq 3 && "$2" == "--schema" ]] || usage
record_path="$1"
schema_path="$3"
[[ -f "$record_path" ]] || { echo "verify-lifecycle-evaluation: record not found: $record_path" >&2; exit 2; }
[[ -f "$schema_path" ]] || { echo "verify-lifecycle-evaluation: schema not found: $schema_path" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || { echo "verify-lifecycle-evaluation: python3 not found" >&2; exit 2; }

python3 - "$record_path" "$schema_path" <<'PYEOF'
import hashlib
import json
import re
import sys

try:
    import yaml
except ImportError as exc:
    print(f"verify-lifecycle-evaluation: PyYAML is required: {exc}", file=sys.stderr)
    raise SystemExit(2)

record_path, schema_path = sys.argv[1:3]
digest_re = re.compile(r"^[0-9a-f]{64}$")

def fail(code, path, detail):
    return {"code": code, "path": path, "detail": str(detail)[:240]}

try:
    with open(schema_path, encoding="utf-8") as stream:
        schema = yaml.safe_load(stream)
    if not isinstance(schema, dict) or not schema.get("schema_version"):
        print("verify-lifecycle-evaluation: schema must declare schema_version", file=sys.stderr)
        raise SystemExit(2)
    allowed = {"schema_version", "evaluation_id"} | set(schema.get("top_level_fields", []))
    reasons_allowed = set(schema.get("invalidity_reason", []))
    records = []
    with open(record_path, encoding="utf-8") as stream:
        for line_number, raw in enumerate(stream, 1):
            if not raw.strip():
                continue
            try:
                value = json.loads(raw)
            except json.JSONDecodeError as exc:
                print(json.dumps({"path": f"line[{line_number}]", "code": "malformed_record", "detail": exc.msg}, sort_keys=True), file=sys.stderr)
                raise SystemExit(1)
            if not isinstance(value, dict):
                print(json.dumps({"path": f"line[{line_number}]", "code": "malformed_record", "detail": "record must be an object"}, sort_keys=True), file=sys.stderr)
                raise SystemExit(1)
            records.append(value)
    if len(records) != 1:
        code = "duplicate_record" if len(records) > 1 else "malformed_record"
        print(json.dumps({"path": "/", "code": code, "detail": f"expected exactly one record, got {len(records)}"}, sort_keys=True), file=sys.stderr)
        raise SystemExit(1)
    record = records[0]
    errors = []
    for key in record:
        if key not in allowed:
            errors.append(fail("malformed_record", f"/{key}", "unexpected top-level field"))
    required = ["schema_version", "evaluation_id", "identity", "source_artifacts", "structural", "judge", "execution_oracle", "eligibility"]
    for key in required:
        if key not in record:
            errors.append(fail("malformed_record", f"/{key}", "required field is missing"))
    if record.get("schema_version") != schema["schema_version"]:
        errors.append(fail("unsupported_schema_version", "/schema_version", record.get("schema_version")))
    if not isinstance(record.get("evaluation_id"), str) or not record["evaluation_id"].strip():
        errors.append(fail("malformed_record", "/evaluation_id", "non-empty string required"))
    identity = record.get("identity")
    if not isinstance(identity, dict):
        errors.append(fail("malformed_record", "/identity", "object required"))
    else:
        for key in ("run_id", "scenario_id", "scenario_version", "fixture_id", "adapter_id", "adapter_version", "shark_binary_digest", "shark_content_digest"):
            if key not in identity or identity[key] is None:
                errors.append(fail("identity_missing", f"/identity/{key}", "required identity is missing"))
        def walk(value, path):
            if isinstance(value, dict):
                for key, child in value.items():
                    child_path = f"{path}/{key}"
                    if "digest" in key.lower() and child is not None and isinstance(child, str) and not digest_re.fullmatch(child):
                        errors.append(fail("malformed_digest", child_path, "lowercase SHA-256 digest required"))
                    walk(child, child_path)
            elif isinstance(value, list):
                for index, child in enumerate(value):
                    walk(child, f"{path}[{index}]")
        walk(identity, "/identity")
    for block in ("structural", "judge", "execution_oracle"):
        if not isinstance(record.get(block), dict):
            errors.append(fail("malformed_record", f"/{block}", "truth block must be an object"))
    eligibility = record.get("eligibility")
    if not isinstance(eligibility, dict):
        errors.append(fail("malformed_record", "/eligibility", "eligibility block must be an object"))
    else:
        reasons = eligibility.get("invalidity_reasons")
        if not isinstance(reasons, list):
            errors.append(fail("malformed_record", "/eligibility/invalidity_reasons", "array required"))
            reasons = []
        for index, item in enumerate(reasons):
            if not isinstance(item, dict) or not isinstance(item.get("code"), str):
                errors.append(fail("malformed_record", f"/eligibility/invalidity_reasons[{index}]", "named reason required"))
            elif reasons_allowed and item["code"] not in reasons_allowed:
                errors.append(fail("malformed_record", f"/eligibility/invalidity_reasons[{index}]/code", "schema-owned reason required"))
        if eligibility.get("aggregate_eligible") is True and reasons:
            errors.append(fail("aggregate_ineligible", "/eligibility/aggregate_eligible", "eligible record cannot retain invalidity reasons"))
        if eligibility.get("aggregate_eligible") is True:
            for block in ("structural", "judge", "execution_oracle"):
                observed = record.get(block, {}).get("observed_result")
                if block == "execution_oracle" and observed != "pass":
                    errors.append(fail("aggregate_ineligible", f"/{block}/observed_result", "eligible record requires oracle pass"))
                if block != "execution_oracle" and observed not in {"pass", "not_applicable"}:
                    errors.append(fail("aggregate_ineligible", f"/{block}/observed_result", "eligible record has incomplete truth"))
    if errors:
        print("verify-lifecycle-evaluation: " + json.dumps(errors, sort_keys=True, separators=(",", ":")), file=sys.stderr)
        raise SystemExit(1)
    retained = record.get("eligibility", {}).get("invalidity_reasons", [])
    verdict = {"evaluation_id": record["evaluation_id"], "aggregate_eligible": record["eligibility"].get("aggregate_eligible"), "invalidity_reasons": retained, "source_artifacts": record.get("source_artifacts", {})}
    print(json.dumps(verdict, sort_keys=True, separators=(",", ":")))
    if verdict["aggregate_eligible"] is not True:
        print("verify-lifecycle-evaluation: " + json.dumps(verdict, sort_keys=True, separators=(",", ":")), file=sys.stderr)
    raise SystemExit(0 if verdict["aggregate_eligible"] is True else 1)
except (OSError, yaml.YAMLError, json.JSONDecodeError, TypeError, KeyError) as exc:
    print(f"verify-lifecycle-evaluation: malformed_record: {exc}", file=sys.stderr)
    raise SystemExit(2)
PYEOF
