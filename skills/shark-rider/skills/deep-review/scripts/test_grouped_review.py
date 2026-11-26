import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from test_adaptive_review import HERE, review_packet, scope_fixture
from testdata.grouped_findings import grouped_cases


class GroupedReviewTest(unittest.TestCase):
    def test_consolidation_and_persisted_decisions_account_for_both_observations(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            identity = review_packet(scope)['decisions']['candidate_identity']
            files = scope['changed_files']
            args = dict(diff_path=scope['diff_path'], skill_dir=str(HERE.parent), candidate_commit=identity['commit'],
                        candidate_tree=identity['tree'], diff_sha256=identity['diff_sha256'], changed_files=files)
            harness = """
const fs = require('node:fs');
const source = fs.readFileSync(process.argv[1], 'utf8');
const run = new (Object.getPrototypeOf(async function(){}).constructor)(
    'args','phase','parallel','agent','log', source.slice(source.indexOf('// ─── Args')));
const {args, angles, response} = JSON.parse(fs.readFileSync(0,'utf8'));
const agent = async (_, options) => options.label === 'consolidator' ? response : angles[options.label[0].toLowerCase()];
run(args, ()=>{}, tasks=>Promise.all(tasks.map(task=>task())), agent, ()=>{})
    .then(result=>process.stdout.write(JSON.stringify(result)));
"""
            for case, angles, report, decisions, error in grouped_cases(identity, files, files[0]):
                with self.subTest(case=case, route='native'):
                    payload = dict(args=args, angles=angles, response=dict(report=report, decisions=decisions))
                    result = json.loads(subprocess.check_output(['node', '-e', harness, HERE / 'review_workflow.js'],
                                        input=json.dumps(payload), text=True))
                    self.assertEqual(result['angle_results'], angles)
                    self.assertEqual(result['specialists_completed'], 6)
                    self.assertEqual(result['consolidator_completed'], not error, result['report'])
                    if error:
                        self.assertIn(error, result['report'])
                    else:
                        self.assertEqual((result['report'], result['decisions']), (report, decisions))
                for mode in ('canonical-six-angle', 'dispatched-six-angle'):
                    with self.subTest(case=case, route=mode):
                        (root / 'body').write_text(report)
                        (root / 'decisions').write_text(json.dumps(decisions))
                        (root / 'angles').write_text(json.dumps(angles))
                        output = root / f'{case}-{mode}.md'
                        result = subprocess.run([sys.executable, HERE / 'adaptive_review.py', 'write-report',
                            '--project-root', repo, '--source-root', repo, '--body-file', root / 'body',
                            '--review-output-path', output, '--scope-file', scope_file,
                            '--base-commit', scope['base_commit'], '--diff-path', scope['diff_path'],
                            '--runner-mode', mode, '--specialists-completed', '6', '--consolidator-completed', 'true',
                            '--fallback-reason', 'native unavailable', '--angle-results', root / 'angles',
                            '--review-decisions', root / 'decisions', '--verdict', 'PASS-WITH-TRIAGE'],
                            text=True, capture_output=True)
                        metadata = json.loads(result.stdout)
                        self.assertEqual(result.returncode == 0, not error, metadata)
                        self.assertEqual(metadata['verdict'], 'INCOMPLETE' if error else 'PASS-WITH-TRIAGE')
                        if error:
                            self.assertIn(error, output.read_text())
                        else:
                            self.assertIn(report, output.read_text())
                            self.assertEqual(json.loads(Path(metadata['retained_decisions']).read_text()), decisions)
