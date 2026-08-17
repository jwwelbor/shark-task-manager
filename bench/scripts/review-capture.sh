#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "--input" || "${3:-}" != "--output" || $# -ne 4 ]]; then
  printf 'usage: review-capture.sh --input <review.json> --output <capture.json>\n' >&2; exit 2
fi
python3 - "$2" "$4" <<'PY'
import json, os, sys, tempfile
from pathlib import Path
source, destination = map(Path, sys.argv[1:])
try:
    document = json.loads(source.read_text(encoding="utf-8"))
except (OSError, json.JSONDecodeError) as exc:
    print(f"review-capture: cannot read input: {exc}", file=sys.stderr); raise SystemExit(2)
gates = document.get("gates")
if not isinstance(gates, list):
    print("review-capture: input gates must be an array", file=sys.stderr); raise SystemExit(1)
seen, captured = set(), []
for gate in gates:
    if not isinstance(gate, dict) or not isinstance(gate.get("gate_id"), str) or not gate["gate_id"].strip():
        print("review-capture: every gate requires a gate_id", file=sys.stderr); raise SystemExit(1)
    gate_id = gate["gate_id"]
    if gate_id in seen:
        print(f"review-capture: duplicate gate_id {gate_id}", file=sys.stderr); raise SystemExit(1)
    seen.add(gate_id)
    reached, status = gate.get("reached") is True, gate.get("collector_status")
    if not reached:
        state, findings = "not_reached", []
    elif status == "complete":
        findings = gate.get("findings")
        if not isinstance(findings, list):
            print(f"review-capture: findings must be an array for {gate_id}", file=sys.stderr); raise SystemExit(1)
        state = "findings" if findings else "zero_findings"
    elif status == "failure":
        if gate.get("findings") not in ([], None):
            print(f"review-capture: failed gate {gate_id} cannot contain findings", file=sys.stderr); raise SystemExit(1)
        state, findings = "collection_failure", []
    else:
        print(f"review-capture: unsupported collector status for {gate_id}: {status!r}", file=sys.stderr); raise SystemExit(1)
    record = {"gate_id": gate_id, "state": state, "round": gate.get("round"), "candidate_ref": gate.get("candidate_ref"), "policy_ref": gate.get("policy_ref"), "findings": findings}
    if state == "collection_failure": record["collector_error"] = gate.get("collector_error", "")
    captured.append(record)
result = {"review_gates": captured}
destination = destination.resolve()
destination.parent.mkdir(parents=True, exist_ok=True)
fd, temporary = tempfile.mkstemp(prefix=f".{destination.name}.", dir=destination.parent)
try:
    with os.fdopen(fd, "w", encoding="utf-8") as stream:
        json.dump(result, stream, sort_keys=True, separators=(",", ":")); stream.write("\n"); stream.flush(); os.fsync(stream.fileno())
    os.replace(temporary, destination)
except OSError:
    try: os.unlink(temporary)
    except OSError: pass
    raise
PY
