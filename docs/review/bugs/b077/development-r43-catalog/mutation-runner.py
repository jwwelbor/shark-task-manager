"""Run each changed comparison against the same normal-caller fixtures."""
import concurrent.futures
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

root = Path.cwd()
artifact = root / "docs/review/bugs/b077/development-r43-catalog"
suite = root / "tests/contracts/testdata/b077_codex_route.py"
def child(name, directory, test):
    sys.path.insert(0, str(suite.parent))
    import b077_codex_route as fixture
    from b077_catalog_cases import CatalogCases
    fixture.RIDER=Path(directory)
    ran=unittest.TextTestRunner().run(unittest.TestSuite([CatalogCases("test_"+test)]))
    return 0 if ran.wasSuccessful() else 1

if len(sys.argv)>1:
    sys.exit(child(*sys.argv[1:]))
matrix="catalog_stage_matrix"
raw="raw_catalog_descriptions"
lookup='roles = CAPABILITIES.get(facts["description"])'
facts='description=" ".join(description.casefold().split()).removesuffix(".")'
# Limit mutations to lookup or duplicate comparison without changing the policy oracle.
look_raw='rows[model] = dict(page=page, facts=facts)'
mutants={
    "missing-sonnet":(matrix, '"legacy balanced model for agentic coding": {"sonnet"}', '"legacy balanced model for agentic coding": set()'),
    "extra-opus":(matrix, '"legacy balanced model for agentic coding": {"sonnet"}', '"legacy balanced model for agentic coding": {"sonnet", "opus"}'),
    "all-roles":(matrix,'elif request["model"] not in roles:', 'elif not roles:'),
    "keep-period":(raw, lookup, 'roles = CAPABILITIES.get(" ".join(row["raw"].casefold().split()))'),
    "keep-case":(raw, lookup, 'roles = next((roles for text,roles in CAPABILITIES.items() if text.capitalize()==" ".join(row["raw"].split()).removesuffix(".")), None)'),
    "keep-exterior":(raw, lookup, 'roles = None if row["raw"] != row["raw"].strip() else CAPABILITIES.get(facts["description"])'),
    "keep-internal":(raw, lookup, 'roles = CAPABILITIES.get(row["raw"].strip().casefold().removesuffix("."))'),
    "raw-duplicates":(raw,'if facts != previous and "conflict" not in record:', 'if facts != previous or rows[model]["raw"] != row["description"]:'),
    "role-duplicates":(raw,'if facts != previous and "conflict" not in record:', 'if dict(facts, description=CAPABILITIES.get(facts["description"])) != dict(previous,description=CAPABILITIES.get(previous["description"])):'),
    "substring":(raw,lookup,'roles = next((roles for desc,roles in CAPABILITIES.items() if desc in facts["description"]), None)'),
    "drop-interior-punctuation":(raw,lookup,'roles = next((roles for desc,roles in CAPABILITIES.items() if " ".join(desc.replace(",", " ").split()) == " ".join(facts["description"].replace(","," ").split())), None)'),
    "ignore-supported":("duplicate_selection_facts",'if facts != previous and "conflict" not in record:','if dict(facts,supported_efforts=None) != dict(previous,supported_efforts=None):'),
    "ignore-default":("duplicate_selection_facts",'if facts != previous and "conflict" not in record:','if dict(facts,default_effort=None) != dict(previous,default_effort=None):'),
    "ignore-marker":("duplicate_selection_facts",'if facts != previous and "conflict" not in record:','if dict(facts,catalog_default=None) != dict(previous,catalog_default=None):'),
    "first-row":(raw,'if facts != previous and "conflict" not in record:', 'if False:'),
    "last-row":(raw,'if facts != previous and "conflict" not in record:', 'rows[model] = dict(page=page,facts=facts)\n                if False:'),
}
mutants.update({
    "keep-period-matrix":(matrix,lookup,mutants["keep-period"][2]),
    "ignore-effort-fit":("catalog_effort_and_tie_break",'elif effort not in facts["supported_efforts"]:',"elif False:"),
    "ignore-observed-effort":("catalog_effort_and_tie_break",'selection["selected_effort"] if selection.get("mapping_ref") else selection.get("effort")','selection.get("effort")'),
})
def run(entry):
    name,(test,old,new)=entry
    with tempfile.TemporaryDirectory() as d:
        local=Path(d)/"rider"
        shutil.copytree(root/"skills/shark-rider",local)
        reader=local/"scripts"/("observe-codex-turn.py" if name=="ignore-observed-effort" else "read-codex-route.py")
        s=reader.read_text()
        if old not in s: raise ValueError(name)
        if 'row["raw"]' in new or 'rows[model]["raw"]' in new:
            s=s.replace(look_raw,'rows[model] = dict(page=page, facts=facts, raw=row["description"])')
        reader.write_text(s.replace(old,new))
        trace=Path(d)/"trace"
        env=dict(os.environ,B077_CATALOG_TRACE=str(trace))
        done=subprocess.run([sys.executable,__file__,name,str(local),test],env=env,capture_output=True,text=True)
        records=[json.loads(l) for l in trace.read_text().splitlines()] if trace.exists() else []
        delta=[dict(stage=r["stage"], expected=r["expected"],actual=r["actual"]["route_status"],
                    claims=r["commands"].count("claim"),submissions=sum(c["method"]=="exec" for c in r["calls"]),
                    admissions=int(r["actual"]["route_status"]=="verified"),
                    inputs=[[row["description"] for row in page["data"]] for page in r["raw_pages"]]) for r in records if r["expected"]!=r["actual"]["route_status"]]
        out=dict(mutation=name,test=test,exit=done.returncode,cases=len(records),changed=delta)
        (artifact/(name+".log")).write_text(done.stderr)
        print(name,len(records),len(delta),flush=True)
        return out
with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
    out=list(pool.map(run,mutants.items()))
(artifact/"mutation-results.json").write_text(json.dumps(out,indent=2)+"\n")
