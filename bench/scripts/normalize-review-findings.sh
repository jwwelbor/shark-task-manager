#!/usr/bin/env bash
# TC-071: preserve I-07 review evidence and independently normalize findings.
set -euo pipefail
exec python3 - "$@" <<'PY'
import argparse, hashlib, json, sys
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("--i07", required=True)
parser.add_argument("--output", required=True)
args = parser.parse_args()

def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)

def finding_id(gate, finding):
    raw = {"gate": gate, "fingerprint": finding.get("fingerprint"), "criterion": finding.get("criterion"), "defect_class": finding.get("defect_class")}
    return "f09-" + hashlib.sha256(canonical(raw).encode()).hexdigest()[:16]

try:
    source = json.loads(Path(args.i07).read_text(encoding="utf-8"))
    gates = source.get("review_gates")
    if not isinstance(gates, list):
        raise ValueError("review_gates must be an array")
    truth = source.get("truth_set")
    truth_ids = set(truth.get("finding_ids", [])) if isinstance(truth, dict) else set()
    raw = []
    normalized = []
    seen_by_fingerprint = {}
    for gate_record in gates:
        if not isinstance(gate_record, dict) or not isinstance(gate_record.get("gate_id"), str):
            raise ValueError("every review gate requires gate_id")
        gate = gate_record["gate_id"]
        candidate = gate_record.get("candidate_ref")
        for finding in gate_record.get("findings", []):
            if not isinstance(finding, dict):
                raise ValueError("every review finding must be an object")
            raw.append({"gate": gate, "candidate_ref": candidate, "finding": finding})
            fingerprint = finding.get("fingerprint")
            previous = seen_by_fingerprint.get(fingerprint)
            link = "recurrent" if previous and previous["raw_source_ref"]["candidate_ref"] != candidate else ("duplicate" if previous else None)
            source_kind = "seeded_truth_set" if finding.get("truth_set_id") in truth_ids else "independent_adjudication"
            confirmed = finding.get("truth_set_id") in truth_ids
            normalized.append({
                "f09_finding_id": "f09-" + hashlib.sha256(canonical({"fingerprint": fingerprint, "defect_class": finding.get("defect_class")}).encode()).hexdigest()[:16],
                "raw_finding": finding,
                "raw_source_ref": {"gate": gate, "candidate_ref": candidate},
                "confirmation_source": source_kind,
                "first_seen_gate": previous["raw_source_ref"]["gate"] if previous else gate,
                "duplicate_or_recurrence": link,
                "resolution_candidate": candidate,
                "final_disposition": "confirmed" if confirmed else "unconfirmed",
            })
            if not previous:
                seen_by_fingerprint[fingerprint] = normalized[-1]
    unique = [item for item in normalized if item["duplicate_or_recurrence"] != "duplicate"]
    confirmed_ids = {item["f09_finding_id"] for item in unique if item["final_disposition"] == "confirmed"}
    unconfirmed_ids = {item["f09_finding_id"] for item in unique if item["final_disposition"] == "unconfirmed"}
    counts = {"emitted": len(normalized), "normalized_unique": len(unique), "duplicate": sum(item["duplicate_or_recurrence"] == "duplicate" for item in normalized), "recurrent": sum(item["duplicate_or_recurrence"] == "recurrent" for item in normalized), "confirmed": len(confirmed_ids), "unconfirmed": len(unconfirmed_ids)}
    if isinstance(truth, dict) and truth_ids:
        counts["truth_set_status"] = "available"
        seeded = len(truth_ids)
        counts["precision"] = counts["confirmed"] / counts["normalized_unique"] if counts["normalized_unique"] else 0
        counts["recall"] = counts["confirmed"] / seeded if seeded else 0
    else:
        counts["truth_set_status"] = "truth-set-unavailable"
    result = {"schema_version": "1.0", "raw_review_gates": source["review_gates"], "raw_source": {"i07_path": str(Path(args.i07))}, "truth_set": {"available": bool(truth_ids), "digest": truth.get("digest") if isinstance(truth, dict) else None}, "normalized_findings": normalized, "derived_counts": counts}
    destination = Path(args.output); destination.parent.mkdir(parents=True, exist_ok=True); destination.write_text(json.dumps(result, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
    print(f"finding_normalization_invalid: {exc}", file=sys.stderr)
    raise SystemExit(2)
PY
