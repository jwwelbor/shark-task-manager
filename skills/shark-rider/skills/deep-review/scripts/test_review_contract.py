"""Shared acceptance fixtures for native and persisted review decisions."""
import json
from pathlib import Path
import subprocess
import unittest

from adaptive_review import validate_review_decisions

HERE = Path(__file__).resolve().parent
FIXTURE = json.loads((HERE / 'testdata' / 'decisions.json').read_text())


class ReviewContractTests(unittest.TestCase):
    def test_python_and_native_share_decision_contract(self):
        identity = FIXTURE['identity']
        scope = {'candidate_commit': identity['commit'], 'candidate_tree': identity['tree'],
                 'diff_sha256': identity['diff_sha256'], 'repair_plan_sha256': identity['repair_plan_sha256']}
        harness = r'''
const fs = require('fs');
const source = fs.readFileSync(process.argv[1], 'utf8');
const run = new (Object.getPrototypeOf(async function(){}).constructor)('args','phase','parallel','agent','log', source.slice(source.indexOf('// ─── Args')));
const fixture = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const test = fixture.cases.find(c => c.name === process.argv[3]);
const id = fixture.identity;
const angle = {candidate_identity:id, reviewed_files:['file','context'], findings:[]};
const parallel = async () => [{...angle,findings:test.raw_findings}, angle, angle, angle, angle, angle];
const agent = async () => ({report:'Scope: all candidate files and production callers. Findings: reviewed with evidence and disposition. VERDICT: '+(test.verdict || 'PASS'), decisions:test.decisions});
const args = {diff_path:'diff',skill_dir:'skill',candidate_commit:id.commit,candidate_tree:id.tree,diff_sha256:id.diff_sha256,repair_plan_sha256:id.repair_plan_sha256,entity_type:'bug',repair_context:{plan:'plan',plan_revision:'sha256:'+id.repair_plan_sha256,research:'research',coverage:'coverage',prior_findings:[]},changed_files:fixture.files};
run(args,()=>{},parallel,agent,()=>{}).then(r => process.stdout.write(JSON.stringify(r.runner_mode !== 'incomplete'))).catch(e => {console.error(e);process.exit(1)});
'''
        for case in FIXTURE['cases']:
            with self.subTest(case=case['name']):
                try:
                    validate_review_decisions(case['decisions'], scope, case['raw_findings'], case.get('verdict', 'PASS'))
                    python_ok = True
                except ValueError:
                    python_ok = False
                proc = subprocess.run(['node', '-e', harness, str(HERE / 'review_workflow.js'), str(HERE / 'testdata' / 'decisions.json'), case['name']], check=True, capture_output=True, text=True)
                self.assertEqual(python_ok, case['valid'])
                self.assertEqual(json.loads(proc.stdout), case['valid'])
