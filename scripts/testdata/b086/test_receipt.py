"""Delivery gate over an already-recorded receipt: withheld reason and feature-note substitute."""

import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import host
import prepare
import verify
from test_destination import FINGERPRINT, events, observer

HERE = Path(__file__).resolve().parent
KINDS = ['receipt-ship', 'receipt-no-reason', 'receipt-note-only']


class Receipts(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cls.root = Path(cls.tmp.name)
        args = [a for k in KINDS for a in ('--scenario', f'{k}:{k}')]
        done = subprocess.run([sys.executable, HERE / 'prepare.py', cls.root, '--synthetic', *args],
                              capture_output=True)
        assert done.returncode == 0, done.stderr

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def receipt(self, name):
        return json.loads((self.root / name / 'receipt.json').read_bytes())

    def test_the_pair_differs_only_by_the_withheld_reason(self):
        ship, bare = self.receipt('receipt-ship'), self.receipt('receipt-no-reason')
        self.assertIn('reason', ship)
        drop = {'reason', 'tracker_project'}
        self.assertEqual({k: v for k, v in ship.items() if k not in drop}, {k: v for k, v in bare.items() if k not in drop})
        self.assertEqual((self.root / 'receipt-ship/grant.json').read_bytes(),
                         (self.root / 'receipt-no-reason/grant.json').read_bytes())

    def test_the_tracker_holds_the_owner_or_only_the_feature_note(self):
        for name, tasks, notes in (('receipt-ship', 1, 0), ('receipt-no-reason', 1, 0), ('receipt-note-only', 0, 1)):
            snap = prepare.snapshot(self.root / name / 'project/shark-tasks.db')
            self.assertEqual(len(snap['tasks']), tasks, name)
            self.assertEqual(len(snap['notes']), notes, name)
            self.assertTrue(all(FINGERPRINT in (t[3] or '') for t in snap['tasks']))
        self.assertEqual(self.receipt('receipt-note-only')['owner']['type'], 'feature-note')

    def test_prompt_makes_the_host_a_gate_and_enables_only_the_ci_endpoint(self):
        for name in KINDS:
            text = host.prompt(self.root / name)
            self.assertIn('You are only the delivery gate', text)
            self.assertIn('Delivery:', text)
            self.assertEqual(observer(self.root / name, 'merge').returncode, 1)

    def test_verifier_demands_zero_ci_and_a_named_gap_for_the_bare_receipt(self):
        name = 'receipt-no-reason'
        parent = Path(tempfile.mkdtemp(dir=self.root))
        case = parent / name
        shutil.copytree(self.root / name, case)
        manifest = json.loads((self.root / 'preparation.json').read_bytes())
        stat = (case / 'project/shark-tasks.db').stat()
        manifest['bindings'][name]['db_inode'] = [stat.st_dev, stat.st_ino]
        (parent / 'preparation.json').write_text(json.dumps(manifest))
        binding = manifest['bindings'][name]
        self.assertEqual(observer(case, 'ci').returncode, 0)
        ev = events(case)
        entries = [e for e in ev if e['event'] == 'entry']
        children = verify.read_children(case / 'project/shark-tasks.db', binding)
        missing = verify.receipt_checks(case, binding, name, entries, ev, [], [], children, [], FINGERPRINT,
                                        case / 'project')
        self.assertIn('zero CI handoffs for an unreconciled receipt', missing)
        self.assertIn('received pending finding naming the missing reason', missing)


class Policy(unittest.TestCase):
    def test_same_alternate_packet_and_receipt_differ_only_by_the_policy_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            done = subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic', '--synthetic-variant',
                                   'adversarial', '--scenario', 'allow:policy-allow',
                                   '--scenario', 'require:policy-require'], capture_output=True)
            self.assertEqual(done.returncode, 0, done.stderr)
            packet = lambda n: {p.name: p.read_bytes() for p in (root / n / 'packet').iterdir()}
            self.assertEqual(packet('allow'), packet('require'))
            self.assertIn('adversarial-cli', (root / 'allow/packet/review.md').read_text())
            self.assertNotEqual((root / 'allow/review-policy.json').read_bytes(),
                                (root / 'require/review-policy.json').read_bytes())
            strip = lambda n: {k: v for k, v in json.loads((root / n / 'receipt.json').read_bytes()).items()
                               if k != 'tracker_project'}
            self.assertEqual(strip('allow'), strip('require'))
            self.assertIn('review admission policy', host.prompt(root / 'allow'))


class Effort(unittest.TestCase):
    def test_review_entry_records_only_the_chosen_effort_and_the_verifier_checks_it(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            done = subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic',
                                   '--scenario', 's:effort-small', '--scenario', 'l:effort-large',
                                   '--scenario', 'r:effort-risk'], capture_output=True)
            self.assertEqual(done.returncode, 0, done.stderr)
            manifest = json.loads((root / 'preparation.json').read_bytes())
            small, large, risk = (json.loads((root / n / 'candidate-stats.json').read_bytes()) for n in 'slr')
            self.assertEqual((small['risk_boundaries'], large['risk_boundaries']), ([], []))
            self.assertGreater(large['changed_lines'], small['changed_lines'])
            self.assertEqual(risk['changed_lines'], small['changed_lines'])
            self.assertTrue(risk['risk_boundaries'])
            self.assertIn('candidate-stats.json', host.prompt(root / 's'))
            self.assertNotIn('retained unattended review handoff', host.prompt(root / 's'))
            self.assertEqual(observer(root / 's', 'review', 'high', 'extra').returncode, 1)
            self.assertEqual(observer(root / 's', 'review', 'high').returncode, 0)
            self.assertEqual(observer(root / 's', 'ci').returncode, 1)
            invoked = [json.loads(l)['argv'] for l in (root / 's/observations/review.jsonl').read_text().splitlines()]
            self.assertEqual(invoked, [['high']])
            result = verify.inspect_effort(root / 's', manifest)
            self.assertIn('run-bound native host and private-skill receipt', result['missing'])
            self.assertNotIn('one deep-review entry with effort high', ' '.join(result['missing']))
            wrong = verify.inspect_effort(root / 'l', manifest)  # no review call yet
            self.assertTrue(any('effort max' in m for m in wrong['missing']))

    def test_rebound_candidate_after_a_local_fix_names_its_identity_and_its_effort(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            done = subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic',
                                   '--scenario', 'q:effort-fix-small', '--scenario', 'w:effort-fix-risk'],
                                  capture_output=True)
            self.assertEqual(done.returncode, 0, done.stderr)
            manifest = json.loads((root / 'preparation.json').read_bytes())
            small, risk = (json.loads((root / n / 'candidate-stats.json').read_bytes()) for n in 'qw')
            self.assertEqual(small['after_local_fix']['resulting_candidate_identity'],
                             risk['after_local_fix']['resulting_candidate_identity'])
            self.assertEqual((small['risk_boundaries'], bool(risk['risk_boundaries'])), ([], True))
            self.assertIn('resulting candidate identity', host.prompt(root / 'q'))
            self.assertNotIn('authorized local Fix', host.prompt(root / 'q').split('Recorder:')[1])
            identity = small['after_local_fix']['resulting_candidate_identity']
            self.assertEqual(observer(root / 'q', 'review', 'high').returncode, 1)  # identity is required
            self.assertEqual(observer(root / 'q', 'review', 'high', identity).returncode, 0)
            self.assertNotIn('one deep-review entry with effort high', ' '.join(
                verify.inspect_effort(root / 'q', manifest)['missing']))
            observer(root / 'w', 'review', 'high', identity)  # the size-only answer misses the risk boundary
            wrong = verify.inspect_effort(root / 'w', manifest)
            self.assertTrue(any('effort max on the resulting candidate identity' in m for m in wrong['missing']))
            observer(root / 'w', 'review', 'max', identity)
            self.assertTrue(any('one deep-review entry' in m for m in verify.inspect_effort(root / 'w', manifest)['missing']))


if __name__ == '__main__':
    unittest.main()
