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
const angle = {findings: [], reviewed_files: ['file']};
const parallel = async funcs => mode === 'missing' ? [angle, angle, null, angle, angle, angle] :
  mode === 'malformed' ? [angle, angle, {}, angle, angle, angle] : Array(6).fill(angle);
const agent = async () => { calls++; return 'VERDICT: PASS'; };
run({diff_path:'/tmp/diff', skill_dir:'/tmp/skill', changed_files:['file']}, () => {}, parallel, agent, () => {})
  .then(report => process.stdout.write(JSON.stringify({report, calls}))).catch(e => { console.error(e); process.exit(1); });
'''
        for mode in ('missing', 'malformed', 'complete'):
            with self.subTest(mode=mode):
                proc = subprocess.run(['node', '-e', harness, str(SCRIPT), mode], capture_output=True, text=True, check=True)
                result = json.loads(proc.stdout)
                if mode == 'complete':
                    self.assertEqual(result['report'], 'VERDICT: PASS')
                    self.assertEqual(result['calls'], 1)
                else:
                    self.assertIn('INCOMPLETE', result['report'])
                    self.assertEqual(result['calls'], 0)
