"""Replay the saved positive and refusal records against the approved matrix."""
import collections
import gzip
import hashlib
import json
import re
import subprocess
from pathlib import Path

artifacts = Path(__file__).resolve().parent
plan = Path("docs/plan/bugs/B077.repair-plan.md").read_text()
matrix = plan.split("## Catalog eligibility cases for C02")[1].split("### Raw catalog descriptions")[0]
table = re.findall(r"^\| ([^|]+?) \| ([AR] \| [AR] \| [AR] \| [AR] \| [AR] \| [AR]) \|$", matrix, re.M)
stages = ("research","repair_plan","plan_review","development","code_review","qa")
cells = [(description+".",stage,cell) for description,outcomes in table for stage,cell in zip(stages,outcomes.split(" | "), strict=True)]
assert len(cells) == 48
trace = artifacts/"final-controls.jsonl"
raw = trace.read_bytes() if trace.exists() else gzip.decompress(trace.with_suffix(".jsonl.gz").read_bytes())
records = [json.loads(line) for line in raw.splitlines()]
def matches(row,cell):
    pages=row["raw_pages"]
    return (len(pages)==1 and len(pages[0]["data"])==1 and row["stage"]==cell[1]
            and pages[0]["data"][0]["description"]==cell[0] and row["requested_effort"]=="high")

starts = [i for i in range(len(records)-len(cells)+1) if all(matches(records[i+j], cell) for j,cell in enumerate(cells))]
assert len(starts)==1,starts
counts=collections.Counter()
for index,(_,_,cell) in enumerate(cells,starts[0]):
    row=records[index]
    assert row["actual"]["route_status"]==("verified" if cell=="A" else "unsupported")
    assert row["commands"].count("claim")==int(cell=="A")
    assert sum(c["method"]=="exec" for c in row["calls"])==int(cell=="A")
    counts[cell]+=1
for row in records:
    final=row["actual"]
    for ref,sha in (("evidence_ref","attempt_sha256"),("decision_ref","decision_sha256")):
        data=row["graph"][final[ref]]
        assert hashlib.sha256(data.encode()).hexdigest()==final[sha]
    for data in row["graph"].values():
        assert not any(x in data for x in ("/fixture/initial-only","/fixture/confirmation-only"))
summary=dict(candidate=json.loads((artifacts/"candidate.json").read_bytes())["candidate"],records=len(records),
             outcomes=collections.Counter(row["actual"]["route_status"] for row in records),
             matrix_counts=counts,matrix_offset=starts[0],verified_graphs=len(records))
(artifacts/"replay-summary.json").write_text(json.dumps(summary,indent=2)+"\n")
(artifacts/"final-controls.jsonl.gz").write_bytes(gzip.compress(raw,mtime=0))
print(summary)
