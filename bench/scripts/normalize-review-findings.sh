#!/usr/bin/env bash
# TC-071: preserve I-07 review evidence and independently normalize findings.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
F09_SCRIPT_DIR="$SCRIPT_DIR" exec python3 - "$@" <<'PY'
import argparse
import hashlib
import json
import os
import sys
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--i07", required=True)
parser.add_argument("--output", required=True)
args = parser.parse_args()

def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)

def finding_id(finding):
    raw = {"fingerprint": finding["fingerprint"], "defect_class": finding["defect_class"]}
    return "f09-" + hashlib.sha256(canonical(raw).encode()).hexdigest()[:16]

def load_seeded_truth():
    path = Path(os.environ["F09_SCRIPT_DIR"]).parent / "evaluation" / "review-truth-set.json"
    seeded = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(seeded, dict) or seeded.get("schema_version") != "1.0":
        raise ValueError("seeded truth artifact has an unsupported schema")
    finding_ids = seeded.get("finding_ids")
    if not isinstance(finding_ids, list) or not finding_ids or any(not isinstance(item, str) or not item.startswith("f09-") for item in finding_ids):
        raise ValueError("seeded truth artifact requires non-empty F09 finding IDs")
    return seeded

try:
    source = json.loads(Path(args.i07).read_text(encoding="utf-8"))
    if not isinstance(source, dict):
        raise ValueError("I-07 review source must be an object")
    gates = source.get("review_gates")
    if not isinstance(gates, list):
        raise ValueError("review_gates must be an array")
    raw_truth = source.get("truth_set")
    seeded_truth = load_seeded_truth()
    truth_ids = set(seeded_truth["finding_ids"])
    raw = []
    normalized = []
    seen_by_fingerprint = {}
    for gate_record in gates:
        if not isinstance(gate_record, dict) or not isinstance(gate_record.get("gate_id"), str):
            raise ValueError("every review gate requires gate_id")
        gate = gate_record["gate_id"]
        candidate = gate_record.get("candidate_ref")
        findings = gate_record.get("findings")
        if not isinstance(findings, list):
            raise ValueError("every review gate requires a findings array")
        for finding in findings:
            if not isinstance(finding, dict):
                raise ValueError("every review finding must be an object")
            if not isinstance(finding.get("fingerprint"), str) or not finding["fingerprint"].strip():
                raise ValueError("every review finding requires a non-empty fingerprint")
            if not isinstance(finding.get("defect_class"), str) or not finding["defect_class"].strip():
                raise ValueError("every review finding requires a non-empty defect_class")
            raw.append({"gate": gate, "candidate_ref": candidate, "finding": finding})
            fingerprint = finding.get("fingerprint")
            recurrence_key = (fingerprint, finding.get("defect_class"))
            previous = seen_by_fingerprint.get(recurrence_key)
            link = "recurrent" if previous and previous["raw_source_ref"]["candidate_ref"] != candidate else ("duplicate" if previous else None)
            normalized_id = finding_id(finding)
            source_kind = "seeded_truth_set" if normalized_id in truth_ids else "independent_adjudication"
            confirmed = normalized_id in truth_ids
            normalized.append({
                "f09_finding_id": normalized_id,
                "raw_finding": finding,
                "raw_source_ref": {"gate": gate, "candidate_ref": candidate},
                "confirmation_source": source_kind,
                "first_seen_gate": previous["raw_source_ref"]["gate"] if previous else gate,
                "duplicate_or_recurrence": link,
                "resolution_candidate": candidate,
                "final_disposition": "confirmed" if confirmed else "unconfirmed",
            })
            if not previous:
                seen_by_fingerprint[recurrence_key] = normalized[-1]
    unique = [item for item in normalized if item["duplicate_or_recurrence"] != "duplicate"]
    confirmed_ids = {item["f09_finding_id"] for item in unique if item["final_disposition"] == "confirmed"}
    unconfirmed_ids = {item["f09_finding_id"] for item in unique if item["final_disposition"] == "unconfirmed"}
    counts = {"emitted": len(normalized), "normalized_unique": len(unique), "duplicate": sum(item["duplicate_or_recurrence"] == "duplicate" for item in normalized), "recurrent": sum(item["duplicate_or_recurrence"] == "recurrent" for item in normalized), "confirmed": len(confirmed_ids), "unconfirmed": len(unconfirmed_ids)}
    if truth_ids:
        counts["truth_set_status"] = "available"
        seeded = len(truth_ids)
        counts["precision"] = counts["confirmed"] / counts["normalized_unique"] if counts["normalized_unique"] else 0
        counts["recall"] = counts["confirmed"] / seeded if seeded else 0
    result = {"schema_version": "1.0", "raw_review_gates": source["review_gates"], "raw_source": {"i07_path": str(Path(args.i07))}, "raw_truth_set": raw_truth, "truth_set": {"available": True, "source": "bench/evaluation/review-truth-set.json", "digest": hashlib.sha256((Path(os.environ["F09_SCRIPT_DIR"]).parent / "evaluation" / "review-truth-set.json").read_bytes()).hexdigest()}, "normalized_findings": normalized, "derived_counts": counts}
    destination = Path(args.output); destination.parent.mkdir(parents=True, exist_ok=True); destination.write_text(json.dumps(result, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
    print(f"finding_normalization_invalid: {exc}", file=sys.stderr)
    raise SystemExit(2)
PY
