"""Dispatched-runner cases: the recorder counts subagent and rescue-CLI calls; the host picks the runner."""

import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import host
import verify
from test_destination import events, observer

HERE = Path(__file__).resolve().parent
AGENTS = ['angle-a-bugs', 'angle-b-behavior', 'angle-c-sibling', 'angle-d-cleanup', 'angle-e-tests',
          'angle-f-standards']


class Dispatch(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cls.root = Path(cls.tmp.name)
        done = subprocess.run([sys.executable, HERE / 'prepare.py', cls.root, '--synthetic',
                               '--scenario', 'd0:dispatch-ok', '--scenario', 'd1:dispatch-bad'], capture_output=True)
        assert done.returncode == 0, done.stderr
        cls.manifest = json.loads((cls.root / 'preparation.json').read_bytes())

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def fresh(self, name):
        parent = Path(tempfile.mkdtemp(dir=self.root))
        case = parent / name
        shutil.copytree(self.root / name, case)
        scope = case / 'scope/scope.json'
        scope.write_text(scope.read_text().replace(str(self.root / name), str(case)))
        manifest = json.loads(json.dumps(self.manifest))
        (parent / 'preparation.json').write_text(json.dumps(manifest))
        return case

    def test_the_pair_differs_only_in_the_one_staged_angle_response(self):
        ok, bad = self.root / 'd0', self.root / 'd1'
        for name in ['angle-a', 'angle-b', 'angle-d', 'angle-e', 'angle-f']:
            self.assertEqual((ok / f'stub/{name}.json').read_bytes(), (bad / f'stub/{name}.json').read_bytes())
        self.assertEqual((ok / 'stub/consolidator.json').read_bytes(), (bad / 'stub/consolidator.json').read_bytes())
        self.assertEqual((ok / 'stub/cli.json').read_bytes(), (bad / 'stub/cli.json').read_bytes())
        json.loads((ok / 'stub/angle-c.json').read_text())
        with self.assertRaises(ValueError):
            json.loads((bad / 'stub/angle-c.json').read_text())
        self.assertEqual(host.prompt(ok).replace('d0', 'X'), host.prompt(bad).replace('d1', 'X'))
        self.assertFalse((ok / 'packet').exists() or (ok / 'history.json').exists())

    def test_prompt_states_capabilities_not_the_expected_runner_or_outcome(self):
        text = host.prompt(self.root / 'd1')
        for word in ('Runner 2', 'incomplete', 'INCOMPLETE', 'rescue', 'six', 'consolidator`' + ' first'):
            self.assertNotIn(word, text)

    def test_unknown_agents_and_report_arguments_outside_the_host_inputs_are_refused(self):
        case = self.fresh('d0')
        self.assertEqual(observer(case, 'dispatch', 'security-agent', stdin=b'x').returncode, 1)
        self.assertEqual(observer(case, 'report', '--project-root', '/tmp').returncode, 1)
        self.assertEqual(observer(case, 'report', '--body-file', '/etc/passwd', '--verdict', 'PASS').returncode, 1)
        results = [e for e in events(case) if e['event'] == 'result']
        self.assertEqual([e['outcome'] for e in results], ['refused'] * 3)

    def run_runner(self, case, malformed_ok):
        scope = json.loads((case / 'scope/scope.json').read_bytes())
        prompt = f"{scope['candidate_commit']} {scope['diff_sha256']}".encode()
        outputs = {name: observer(case, 'dispatch', name, stdin=prompt).stdout.decode() for name in AGENTS}
        return outputs

    def test_complete_control_and_invalid_angle_produce_the_observed_reports(self):
        for name, angles, extra, expected in [('d0', AGENTS, ['consolidator'], []),
                                              ('d1', AGENTS, [], [])]:
            with self.subTest(case=name):
                case = self.fresh(name)
                outputs = self.run_runner(case, name == 'd0')
                work = case / 'work'
                results = {letter: json.loads(outputs[agent]) for agent, letter in
                           zip(AGENTS, 'abcdef') if name == 'd0' or letter != 'c'}
                if name == 'd0':
                    consolidated = json.loads(observer(case, 'dispatch', 'consolidator', stdin=b'x').stdout)
                    (work / 'body.md').write_text(consolidated['report'])
                    (work / 'angles.json').write_text(json.dumps(results))
                    (work / 'decisions.json').write_text(json.dumps(consolidated['decisions']))
                    done = observer(case, 'report', '--verdict', 'PASS-with-triage', '--runner-mode',
                                    'dispatched-six-angle', '--specialists-completed', '6',
                                    '--consolidator-completed', 'true', '--fallback-reason',
                                    'native Workflow unavailable', '--body-file', 'work/body.md',
                                    '--angle-results', 'work/angles.json', '--review-decisions',
                                    'work/decisions.json')
                else:
                    (work / 'body.md').write_text('**INCOMPLETE** - angle c returned malformed output.\n')
                    done = observer(case, 'report', '--verdict', 'INCOMPLETE', '--runner-mode', 'incomplete',
                                    '--specialists-completed', '5', '--consolidator-completed', 'false',
                                    '--body-file', 'work/body.md')
                self.assertIn(done.returncode, (0, 2), done.stderr)
                last = [e for e in events(case) if e['event'] == 'result' and e['tool'] == 'report'][-1]
                if name == 'd0':
                    self.assertEqual((last['runner_mode'], last['specialists_completed'], last['consolidator_completed']),
                                     ('dispatched-six-angle', '6', True))
                else:
                    self.assertEqual((last['runner_mode'], last['specialists_completed'], last['consolidator_completed'],
                                      last['verdict']), ('incomplete', '5', False, 'INCOMPLETE'))
                    self.assertEqual(sum(e['tool'] == 'dispatch' and e['event'] == 'entry' for e in events(case)), 6)

    def verified(self, name, extra=()):
        """Run the runner in place of a host, then ask the independent verifier (host receipt stubbed)."""
        case = self.fresh(name)
        self.run_runner(case, True)
        observer(case, 'read', 'project/.agents/skills/shark-rider/skills/deep-review/SKILL.md')
        work = case / 'work'
        if name == 'd0':
            consolidated = json.loads(observer(case, 'dispatch', 'consolidator', stdin=b'x').stdout)
            outputs = {l: json.loads((case / f'stub/angle-{l}.json').read_text()) for l in 'abcdef'}
            (work / 'body.md').write_text(consolidated['report'])
            (work / 'angles.json').write_text(json.dumps(outputs))
            (work / 'decisions.json').write_text(json.dumps(consolidated['decisions']))
            observer(case, 'report', '--verdict', 'PASS-with-triage', '--runner-mode', 'dispatched-six-angle',
                     '--specialists-completed', '6', '--consolidator-completed', 'true', '--fallback-reason',
                     'native Workflow unavailable', '--body-file', 'work/body.md', '--angle-results',
                     'work/angles.json', '--review-decisions', 'work/decisions.json')
        else:
            (work / 'body.md').write_text('**INCOMPLETE** - angle c returned malformed output.\n')
            observer(case, 'report', '--verdict', 'INCOMPLETE', '--runner-mode', 'incomplete',
                     '--specialists-completed', '5', '--consolidator-completed', 'false', '--body-file',
                     'work/body.md')
        for args in extra:
            observer(case, *args, stdin=b'x')
        manifest = json.loads(json.dumps(self.manifest))
        original = verify.host_receipt
        verify.host_receipt = lambda *a: True
        try:
            return verify.inspect_dispatch(case, manifest)
        finally:
            verify.host_receipt = original

    def test_verifier_observes_the_control_and_the_invalid_angle_run(self):
        self.assertEqual(self.verified('d0')['missing'], [])
        self.assertEqual(self.verified('d1')['missing'], [])

    def test_verifier_rejects_a_rescue_cli_a_consolidator_or_ci_after_an_invalid_angle(self):
        self.assertIn('zero rescue CLI invocations, discarded or answered',
                      self.verified('d1', [('cli',)])['missing'])
        self.assertIn('zero consolidator invocations after an invalid angle',
                      self.verified('d1', [('dispatch', 'consolidator')])['missing'])
        self.assertIn('zero CI handoffs after an incomplete review', self.verified('d1', [('ci',)])['missing'])

    def test_a_refused_report_probe_is_not_a_persisted_review_and_a_second_written_review_is(self):
        self.assertEqual(self.verified('d1', [('report', '--help')])['missing'], [])
        again = self.verified('d1', [('report', '--verdict', 'INCOMPLETE', '--runner-mode', 'incomplete',
                                      '--specialists-completed', '5', '--consolidator-completed', 'false',
                                      '--body-file', 'work/body.md')])
        self.assertIn('exactly one persisted review', again['missing'])

    def test_verifier_rejects_a_control_that_never_consolidated(self):
        case = self.fresh('d0')
        self.run_runner(case, True)
        original = verify.host_receipt
        verify.host_receipt = lambda *a: True
        try:
            missing = verify.inspect_dispatch(case, json.loads(json.dumps(self.manifest)))['missing']
        finally:
            verify.host_receipt = original
        self.assertIn('exactly one persisted review', missing)


if __name__ == '__main__':
    unittest.main()
