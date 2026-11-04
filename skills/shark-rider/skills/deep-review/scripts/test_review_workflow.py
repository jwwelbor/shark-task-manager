import json
from pathlib import Path
import subprocess
import unittest

SCRIPT = Path(__file__).with_name('review_workflow.js')


class NativeWorkflowTest(unittest.TestCase):
    def test_partial_and_malformed_results_never_reach_consolidator(self):
        harness = r'''
const fs = require('fs');
const src = fs.readFileSync(process.argv[1], 'utf8');
const body = src.slice(src.indexOf('// ─── Args'));
const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
const run = new AsyncFunction('args', 'phase', 'parallel', 'agent', 'log', body);
let calls = 0;
const mode = process.argv[2];
const angle = {candidate_identity:{commit:'abc',tree:'tree',diff_sha256:'hash'}, findings: [], reviewed_files: ['file']};
const parallel = async funcs => mode === 'context' ? Promise.all(funcs.map(f => f())) : mode === 'missing' ? [angle, angle, null, angle, angle, angle] :
  mode === 'malformed' ? [angle, angle, {}, angle, angle, angle] :
  mode === 'empty' ? [angle, angle, {...angle, reviewed_files: []}, angle, angle, angle] :
  mode === 'identity' ? [angle, angle, {...angle, candidate_identity: {...angle.candidate_identity, tree:'wrong'}}, angle, angle, angle] :
  mode === 'blocker' ? [{...angle, findings:[{file:'file', line:1, severity:'blocker', rule:'RISK', summary:'bad', diagnosis:'broken'}]}, angle, angle, angle, angle, angle] : Array(6).fill(angle);
const agent = async (prompt, options) => {
  calls++;
  if (mode === 'context' && options.label !== 'consolidator') {
    if (!prompt.includes('plan-id') || !prompt.includes('research-id') || !prompt.includes('CANDIDATE_TREE: tree')) throw Error('missing context');
    return angle;
  }
  if (mode === 'context' && (!prompt.includes('plan-id') || !prompt.includes('CANDIDATE_TREE: tree'))) throw Error('missing consolidation context');
  return mode === 'short' ? 'VERDICT: PASS' : 'Reviewed scope: all candidate files. Findings: none. Detailed checks establish complete coverage. VERDICT: PASS';
};
run({diff_path:'/tmp/diff', skill_dir:'/tmp/skill', candidate_commit:'abc', candidate_tree:'tree', diff_sha256:'hash', entity_type:'bug', repair_context:{plan:'plan-id', plan_revision:'revision', research:'research-id', coverage:'coverage-id', prior_findings:[]}, changed_files:['file']}, () => {}, parallel, agent, () => {})
  .then(report => process.stdout.write(JSON.stringify({report, calls}))).catch(e => { console.error(e); process.exit(1); });
'''
        for mode in ('missing', 'malformed', 'empty', 'identity', 'short', 'blocker', 'complete', 'context'):
            with self.subTest(mode=mode):
                proc = subprocess.run(['node', '-e', harness, str(SCRIPT), mode], capture_output=True, text=True, check=True)
                result = json.loads(proc.stdout)
                if mode in ('complete', 'context'):
                    self.assertIn('VERDICT: PASS', result['report']['report'])
                    self.assertEqual(len(result['report']['angle_results']), 6)
                    self.assertEqual(result['calls'], 7 if mode == 'context' else 1)
                else:
                    self.assertIn('INCOMPLETE', result['report']['report'])
                    self.assertEqual(result['calls'], 1 if mode in ('short', 'blocker') else 0)
