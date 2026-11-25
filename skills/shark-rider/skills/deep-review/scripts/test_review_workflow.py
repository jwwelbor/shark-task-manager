import itertools
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

from adaptive_review import valid_adversarial_output, valid_review_report

SCRIPT = Path(__file__).with_name('review_workflow.js')


class NativeWorkflowTest(unittest.TestCase):
    def test_consolidator_receives_ordered_inspections_and_observations_by_angle(self):
        paths = ['retired.txt', 'source', 'tests']
        identity = {'commit': 'commit', 'tree': 'tree', 'diff_sha256': 'digest'}
        packets = {}
        for label, order in zip('abcdef', itertools.permutations(paths)):
            packets[label] = {
                'review_contract_version': 2, 'candidate_identity': identity, 'reviewed_files': order,
                'findings': [{'file': 'source', 'line': 1, 'severity': 'blocker',
                              'rule': 'CORRECTNESS', 'summary': 'wrong total',
                              'diagnosis': 'returned 1', 'impact': 'consumer gets the wrong total', 'evidence': f'operation {label}'}],
            }
        harness = r'''
const fs = require('fs');
const source = fs.readFileSync(process.argv[1], 'utf8');
const run = new (Object.getPrototypeOf(async function(){}).constructor)(
    'args', 'phase', 'parallel', 'agent', 'log', source.slice(source.indexOf('// ─── Args')));
const input = JSON.parse(fs.readFileSync(0, 'utf8'));
const calls = [];
const agent = async (prompt, options) => {
    calls.push({prompt, options});
    if (options.label !== 'consolidator') return input.packets[options.label[0].toLowerCase()];
    return {report: 'Scope: all three changed files inspected. Findings: 0 defects found after checking all callers. VERDICT: PASS',
            decisions: {review_contract_version: 2, candidate_identity: input.packets.a.candidate_identity, findings: []}};
};
run(input.args, ()=>{}, tasks => Promise.all(tasks.map(task => task())), agent, ()=>{})
    .then(result => process.stdout.write(JSON.stringify({calls, result})));
'''
        def exercise(angles, **options):
            args = dict(diff_path='diff', skill_dir='skill', candidate_commit='commit',
                        candidate_tree='tree', diff_sha256='digest', changed_files=paths, **options)
            result = subprocess.run(['node', '-e', harness, str(SCRIPT)],
                                    input=json.dumps({'args': args, 'packets': angles}),
                                    capture_output=True, text=True, check=True)
            return json.loads(result.stdout)

        observed = exercise(packets, effort='medium')
        calls = observed['calls']
        self.assertEqual([call['options']['effort'] for call in calls], ['medium'] * 7)
        prompt = calls[-1]['prompt']
        marker = '- ANGLE_RESULTS:'
        with self.subTest(handoff='labeled specialist results'):
            self.assertIn(marker, prompt)
            delivered, _ = json.JSONDecoder().raw_decode(prompt.split(marker, 1)[1].lstrip())
            self.assertEqual(delivered, json.loads(json.dumps(packets)))
        clean = {label: dict(packet, findings=[]) for label, packet in packets.items()}
        observed = exercise(clean)
        self.assertEqual(len(observed['calls']), 7)
        self.assertTrue(all('effort' not in call['options'] for call in observed['calls']))
        with self.subTest(handoff='consolidation schema'):
            schema = observed['calls'][-1]['options']['schema']['properties']['decisions']
            self.assertNotIn('reviewed_files', schema['required'])
            self.assertEqual(schema['properties']['reviewed_files'], {'type': 'array', 'items': {'type': 'string'}})
        with self.subTest(handoff='six angle declarations suffice'):
            self.assertEqual(observed['result']['runner_mode'], 'canonical-six-angle')
            self.assertTrue(observed['result']['consolidator_completed'])
        for path in paths:
            with self.subTest(missing=path):
                incomplete = dict(clean, b=dict(clean['b'], reviewed_files=[p for p in paths if p != path]))
                observed = exercise(incomplete)
                self.assertEqual(len(observed['calls']), 6)
                result = observed['result']
                self.assertEqual(result['runner_mode'], 'incomplete')
                self.assertEqual(result['specialists_completed'], 5)
                self.assertFalse(result['consolidator_completed'])
                self.assertIn('angle b', result['report'].lower())
                self.assertIn(path, result['report'])

    def test_optional_consolidator_manifest_preserves_literal_git_names(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.check_output(['git', '-c', 'user.name=Test', '-c',
                    'user.email=test@example.invalid', '-c', 'commit.gpgsign=false', *args], cwd=root)
            git('init', '-q')
            git('commit', '--allow-empty', '-qm', 'base')
            (root / 'ordinary').write_text('supplement')
            git('add', '--', 'ordinary')
            git('commit', '-qm', 'tracked supplement')
            (root / ' ').write_text('changed')
            git('add', '--', ' ')
            git('commit', '-qm', 'candidate')
            names = git('diff', '--name-only', '-z', 'HEAD^', 'HEAD')
            self.assertEqual(names, b' \0')
            paths = names.decode().split('\0')[:-1]
            args = {'changed_files': paths, 'candidate_commit': git('rev-parse', 'HEAD').decode().strip(),
                    'candidate_tree': git('rev-parse', 'HEAD^{tree}').decode().strip(),
                    'diff_path': 'diff', 'diff_sha256': 'digest', 'skill_dir': str(SCRIPT.parent.parent)}
            harness = r'''
const fs = require('fs');
const src = fs.readFileSync(process.argv[1], 'utf8');
const run = new (Object.getPrototypeOf(async function() {}).constructor)(
    'args', 'phase', 'parallel', 'agent', 'log', src.slice(src.indexOf('// ─── Args')));
const {args, optional, bFiles} = JSON.parse(fs.readFileSync(0, 'utf8'));
const identity = {commit: args.candidate_commit, tree: args.candidate_tree, diff_sha256: args.diff_sha256};
const calls = [];
const agent = async (_, options) => {
    calls.push(options.label);
    if (options.label === 'consolidator') {
        return {report: 'Scope: the frozen single-space path was inspected. Findings: 0 defects found in its changed content. Verdict: PASS',
                decisions: {review_contract_version: 2, candidate_identity: identity, findings: [], ...optional}};
    }
    return {review_contract_version: 2, candidate_identity: identity, findings: [],
            reviewed_files: options.label.startsWith('B:') ? bFiles : args.changed_files};
};
run(args, () => {}, tasks => Promise.all(tasks.map(task => task())), agent, () => {})
    .then(result => process.stdout.write(JSON.stringify({calls, result})));
'''
            def check(optional, complete=True, missing_b=False):
                trial = {'args': args, 'optional': optional, 'bFiles': ['ordinary'] if missing_b else paths}
                output = json.loads(subprocess.check_output(['node', '-e', harness, str(SCRIPT)],
                                    input=json.dumps(trial), text=True))
                result = output['result']
                self.assertEqual(len(output['calls']), 6 if missing_b else 7)
                self.assertEqual(output['calls'].count('consolidator'), 0 if missing_b else 1)
                self.assertEqual(result['specialists_completed'], 5 if missing_b else 6)
                self.assertEqual(result['consolidator_completed'], complete)
                self.assertEqual(result['runner_mode'], 'canonical-six-angle' if complete else 'incomplete')
                if complete:
                    self.assertIn('Verdict: PASS', result['report'])
                    self.assertEqual({k: v for k, v in result['decisions'].items() if k == 'reviewed_files'}, optional)
                else:
                    self.assertIn('INCOMPLETE', result['report'])
                if not complete:
                    self.assertIn('angle b' if missing_b else 'consolidator', result['report'].lower())
                    if missing_b or optional.get('reviewed_files') in ([], ['wrong']):
                        self.assertIn('" "', result['report'])

            check({})
            check({'reviewed_files': [' ']})
            check({'reviewed_files': [' ', 'ordinary']})
            check({'reviewed_files': ['ordinary', ' ']})
            for value in ([], ['wrong'], None, [' ', None], [' ', '../unsafe']):
                with self.subTest(consolidator_paths=value):
                    check({'reviewed_files': value}, complete=False)
            check({'reviewed_files': [' ']}, complete=False, missing_b=True)
            check({'reviewed_files': [' ']})

    def test_partial_and_malformed_results_never_reach_consolidator(self):
        harness = r'''
const fs = require('fs');
const src = fs.readFileSync(process.argv[1], 'utf8');
const body = src.slice(src.indexOf('// ─── Args'));
const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
const run = new AsyncFunction('args', 'phase', 'parallel', 'agent', 'log', body);
let calls = 0;
const mode = process.argv[2];
const angle = {review_contract_version:2, candidate_identity:{commit:'abc',tree:'tree',diff_sha256:'hash',repair_plan_sha256:'a'.repeat(64)}, findings: [], reviewed_files: ['file']};
const parallel = async funcs => mode === 'context' ? Promise.all(funcs.map(f => f())) : mode === 'missing' ? [angle, angle, null, angle, angle, angle] :
  mode === 'malformed' ? [angle, angle, {}, angle, angle, angle] :
  mode === 'empty' ? [angle, angle, {...angle, reviewed_files: []}, angle, angle, angle] :
  mode === 'identity' ? [angle, angle, {...angle, candidate_identity: {...angle.candidate_identity, tree:'wrong'}}, angle, angle, angle] :
  ['blocker','open','refuted','fail_missing','fail_open'].includes(mode) ? [{...angle, findings:[{file:'file', line:1, severity:'blocker', rule:'RISK', summary:'bad', diagnosis:'broken', impact:'wrong total delivered', evidence:'adding 7 and 3 returned 7'}]}, angle, angle, angle, angle, angle] : Array(6).fill(angle);
const agent = async (prompt, options) => {
  calls++;
  if (mode === 'context' && options.label !== 'consolidator') {
    if (!prompt.includes('plan-id') || !prompt.includes('research-id') || !prompt.includes('CANDIDATE_TREE: tree')) throw Error('missing context');
    return angle;
  }
  if (mode === 'context' && (!prompt.includes('plan-id') || !prompt.includes('CANDIDATE_TREE: tree'))) throw Error('missing consolidation context');
  if (mode === 'consolidator_error') throw Error('failed consolidation');
  const report = mode.startsWith('detail_') ? ('CURRENT_REVIEW_STATUS: COMPLETE\nScope: all files. Findings: validation issue retained. VERDICT: PASS-WITH-TRIAGE\nFINDING_DETAIL: INCOMPLETE VALIDATION | F-1 | non-blocker | triage' + (mode === 'detail_multi' ? '\nHISTORICAL_REVIEW_STATUS: DID_NOT_FINISH\nFINDING_DETAIL: INCOMPLETE PRODUCT BEHAVIOR | F-2 | non-blocker | triage' : '')) : mode === 'fixture' ? process.argv[3] : mode === 'short' ? 'VERDICT: PASS' : mode === 'conflict' ? 'Scope and coverage: all files and callers inspected. Findings: conflicting report. VERDICT: PASS then VERDICT: FAIL' : mode.startsWith('fail_') ? 'Reviewed scope: all candidate files. Findings: a blocking defect requires repair. **Verdict:** FAIL' : 'Reviewed scope: all candidate files. Findings: none. Detailed checks establish complete coverage. **Verdict:** PASS';
  const findings = mode.startsWith('detail_') ? [{finding_id:mode === 'detail_unrelated' ? 'F-2' : 'F-1', file:'file', line:1, severity:'non-blocker', rule:'RISK', summary:'validation issue', diagnosis:'missing input check', impact:'invalid input accepted', evidence:'empty input returned success', disposition:mode === 'detail_case' ? 'Triage' : 'triage', reason:'tracked'}, ...(mode === 'detail_multi' ? [{finding_id:'F-2', file:'file', line:2, severity:'non-blocker', rule:'LOGGING', summary:'no diagnostic', diagnosis:'no log on denial', impact:'operator cannot identify the denied input', evidence:'empty log after denial', disposition:'triage', reason:'tracked'}] : [])] : ['open','refuted','fail_open'].includes(mode) ? [{file:'file', line:1, severity:'blocker', rule:'RISK', summary:'bad', diagnosis:'broken', impact:'wrong total delivered', evidence:'adding 7 and 3 returned 7', disposition:mode === 'fail_open' ? 'open' : mode, reason:'evidence'}] : [];
  return {report, decisions:{review_contract_version:2, candidate_identity:angle.candidate_identity, findings}};
};
run({diff_path:'/tmp/diff', skill_dir:'/tmp/skill', candidate_commit:'abc', candidate_tree:'tree', diff_sha256:'hash', repair_plan_sha256:'a'.repeat(64), entity_type:'bug', repair_context:{plan:'plan-id', plan_revision:'sha256:'+'a'.repeat(64), research:'research-id', coverage:'coverage-id', prior_findings:[]}, changed_files:['file']}, () => {}, parallel, agent, () => {})
  .then(report => process.stdout.write(JSON.stringify({report, calls}))).catch(e => { console.error(e); process.exit(1); });
'''
        fixtures = json.loads((SCRIPT.parent / 'testdata' / 'report_bodies.json').read_text())
        for fixture in fixtures:
            with self.subTest(report_fixture=fixture['name']):
                proc = subprocess.run(['node', '-e', harness, str(SCRIPT), 'fixture', fixture['body']], capture_output=True, text=True, check=True)
                result = json.loads(proc.stdout)['report']
                self.assertEqual(result['runner_mode'] == 'canonical-six-angle', fixture['valid'])
                self.assertEqual(valid_review_report(fixture['body'], 'PASS'), fixture['valid'])
                self.assertEqual(valid_adversarial_output(fixture['body']), fixture['valid'])
        for mode, expected in [('detail_match', 'canonical-six-angle'), ('detail_unrelated', 'incomplete'), ('detail_case', 'incomplete'), ('detail_multi', 'canonical-six-angle')]:
            with self.subTest(mode=mode):
                proc = subprocess.run(['node', '-e', harness, str(SCRIPT), mode], capture_output=True, text=True, check=True)
                self.assertEqual(json.loads(proc.stdout)['report']['runner_mode'], expected)
        for mode in ('missing', 'malformed', 'empty', 'identity', 'short', 'blocker', 'open', 'conflict', 'consolidator_error', 'refuted', 'complete', 'context', 'fail_missing', 'fail_open'):
            with self.subTest(mode=mode):
                proc = subprocess.run(['node', '-e', harness, str(SCRIPT), mode], capture_output=True, text=True, check=True)
                result = json.loads(proc.stdout)
                if mode in ('complete', 'context', 'refuted', 'fail_open'):
                    self.assertIn('Verdict:** FAIL' if mode == 'fail_open' else 'Verdict:** PASS', result['report']['report'])
                    self.assertEqual(len(result['report']['angle_results']), 6)
                    self.assertEqual(result['calls'], 7 if mode == 'context' else 1)
                else:
                    self.assertIn('INCOMPLETE', result['report']['report'])
                    self.assertEqual(result['calls'], 1 if mode in ('short', 'blocker', 'open', 'conflict', 'consolidator_error', 'fail_missing') else 0)
                    if mode == 'consolidator_error':
                        self.assertEqual(len(result['report']['angle_results']), 6)
