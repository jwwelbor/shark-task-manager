"""Shared acceptance fixtures for native and persisted review decisions."""
import json
from pathlib import Path
import os
import shutil
import subprocess
import tempfile
import unittest

from adaptive_review import validate_review_decisions

HERE = Path(__file__).resolve().parent
FIXTURE = json.loads((HERE / 'testdata' / 'decisions.json').read_text())


class ReviewContractTests(unittest.TestCase):
    def test_native_and_standalone_consolidators_allow_report_only_nit_verdict(self):
        contract = (HERE.parent / 'references' / 'consolidator.md').read_text()
        workflow = (HERE / 'review_workflow.js').read_text()
        expected = 'pass-with-triage'
        for label, source in (('standalone', contract), ('native', workflow)):
            source = source.lower()
            with self.subTest(source=label):
                self.assertIn(expected, source)
                self.assertIn('report-only nits', source)
                self.assertIn('tracker work', source)
                self.assertIn('substantive non-blockers', source)

    def test_skill_helpers_use_loaded_installation_with_spaces_and_decoy_repo(self):
        skill = (HERE.parent / 'SKILL.md').read_text()
        self.assertIn('DEEP_REVIEW_SKILL_DIR', skill)
        self.assertIn('scripts/resolve_skill_dir.sh', skill)
        self.assertNotIn('/home/jwwel/.claude/skills/shark-rider/skills/deep-review', skill)
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            active = root / 'active skill installation with spaces' / 'deep-review'
            shutil.copytree(HERE.parent, active)
            repository = root / 'separate reviewed repository'
            decoy = repository / 'skills/shark-rider/skills/deep-review'
            (decoy / 'scripts').mkdir(parents=True)
            (decoy / 'SKILL.md').write_text('decoy')
            (decoy / 'scripts/get_diff.sh').write_text('decoy helper')
            result = subprocess.run(
                ['bash', str(active / 'scripts/resolve_skill_dir.sh')],
                cwd=repository,
                env={**os.environ, 'DEEP_REVIEW_SKILL_DIR': str(active)},
                capture_output=True,
                text=True,
                check=True,
            )
            resolved = Path(result.stdout.strip())
            self.assertEqual(resolved, active.resolve())
            self.assertNotEqual((resolved / 'scripts/get_diff.sh').read_text(), (decoy / 'scripts/get_diff.sh').read_text())
            missing_reference = active / 'references/angle-f-standards.md'
            missing_reference.unlink()
            missing = subprocess.run(
                ['bash', str(active / 'scripts/resolve_skill_dir.sh')],
                cwd=repository,
                env={**os.environ, 'DEEP_REVIEW_SKILL_DIR': str(active)},
                capture_output=True,
                text=True,
            )
            self.assertEqual(missing.returncode, 2)
            self.assertIn('missing required file', missing.stderr)

    def test_python_and_native_share_decision_contract(self):
        identity = FIXTURE['identity']
        scope = {'candidate_commit': identity['commit'], 'candidate_tree': identity['tree'],
                 'diff_sha256': identity['diff_sha256'], 'repair_plan_sha256': identity['repair_plan_sha256'],
                 'changed_files': FIXTURE['files']}
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
run(args,()=>{},parallel,agent,()=>{}).then(r => process.stdout.write(JSON.stringify({complete:r.runner_mode !== 'incomplete',decisions:r.decisions}))).catch(e => {console.error(e);process.exit(1)});
'''
        for case in FIXTURE['cases']:
            with self.subTest(case=case['name']):
                try:
                    validate_review_decisions(case['decisions'], scope, case['raw_findings'], case.get('verdict', 'PASS'), require_reviewed_files=False)
                    python_ok = True
                except ValueError:
                    python_ok = False
                proc = subprocess.run(['node', '-e', harness, str(HERE / 'review_workflow.js'), str(HERE / 'testdata' / 'decisions.json'), case['name']], check=True, capture_output=True, text=True)
                self.assertEqual(python_ok, case['valid'])
                native_result = json.loads(proc.stdout)
                self.assertEqual(native_result['complete'], case['valid'])
                if case['name'] == 'nit_report_only':
                    self.assertEqual(native_result['decisions']['findings'][0]['disposition'], 'report-only')
