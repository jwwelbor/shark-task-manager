"""Lost create response: committed write, suspended lookups, then a resumed run."""

import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import host
import prepare
import resume
import verify
from test_destination import FINGERPRINT, enter, events

HERE = Path(__file__).resolve().parent


class LostResponse(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cls.base = Path(cls.tmp.name) / 'base'
        cls.base.mkdir()
        done = subprocess.run([sys.executable, HERE / 'prepare.py', cls.base, '--synthetic',
                               '--scenario', 'lost:lost-response'], capture_output=True)
        assert done.returncode == 0, done.stderr

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def staged(self):
        root = Path(tempfile.mkdtemp(dir=self.tmp.name))
        target = root / 'root'
        shutil.copytree(self.base, target)
        manifest = json.loads((target / 'preparation.json').read_bytes())
        stat = (target / 'lost/project/shark-tasks.db').stat()
        manifest['bindings']['lost']['db_inode'] = [stat.st_dev, stat.st_ino]
        (target / 'preparation.json').write_text(json.dumps(manifest))
        return target, target / 'lost', manifest

    def create(self, case):
        return subprocess.run(['./shark', 'create', 'task', 'E01-F01', 'Worked example',
                               '--description', FINGERPRINT, '--json'], cwd=case / 'project', capture_output=True)

    def test_commit_is_hidden_from_the_host_and_lookups_stay_suspended(self):
        root, case, manifest = self.staged()
        enter(case)
        result = self.create(case)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn(b'T-E01-F01', result.stdout + result.stderr)  # the host never sees the key
        self.assertEqual([t[1] for t in prepare.snapshot(case / 'project/shark-tasks.db')['tasks']], ['E01-F01'])
        lookup = subprocess.run(['./shark', 'search', FINGERPRINT, '--json'], cwd=case / 'project', capture_output=True)
        self.assertNotEqual(lookup.returncode, 0)
        self.assertIn(b'lookup unavailable', lookup.stderr)
        log = events(case)
        shark = [e for e in log if e['event'] == 'result' and e.get('tool') == 'shark']
        self.assertEqual([e['outcome'] for e in shark][-2:], ['response_lost', 'refused'])
        lost = [e for e in log if e.get('outcome') == 'response_lost']
        self.assertEqual(len(lost), 1)
        self.assertTrue(lost[0]['forwarded'])

    def test_a_repeat_create_stays_callable_and_is_counted_and_reported(self):
        root, case, manifest = self.staged()
        enter(case)
        self.create(case)
        self.create(case)
        entries = [e for e in events(case) if e['event'] == 'entry' and e['argv'][:2] == ['create', 'task']]
        results = {e['invocation']: e for e in events(case) if e['event'] == 'result'}
        self.assertEqual(len(entries), 2)
        self.assertEqual(len(prepare.snapshot(case / 'project/shark-tasks.db')['tasks']), 2)
        missing = verify.lost_phase_checks(entries, results, entries, [], [], [], events(case), FINGERPRINT, None)
        self.assertIn('exactly one create whose response was lost and no repeat create', missing)

    def test_resumed_phase_reads_the_original_key_and_forbids_a_second_create(self):
        root, case, manifest = self.staged()
        enter(case)
        self.create(case)
        second = resume.stage(root, 'lost', 'resumed')
        manifest = json.loads((root / 'preparation.json').read_bytes())
        binding = manifest['bindings']['resumed']
        self.assertNotIn('fault', binding)
        self.assertNotEqual(binding['run_id'], manifest['bindings']['lost']['run_id'])
        self.assertIn('resume.json', host.prompt(second))
        enter(second)
        found = subprocess.run(['./shark', 'search', FINGERPRINT, '--json'], cwd=second / 'project',
                               capture_output=True)
        self.assertEqual(found.returncode, 0, found.stderr)
        self.assertIn(b'T-E01-F01-001', found.stdout)
        shots = json.loads((second / 'resume-baseline.json').read_bytes())
        self.assertEqual(len(shots['tasks']), 1)
        # A further create is possible; the verifier must report it.
        self.create(second)
        log = events(second)
        entries = [e for e in log if e['event'] == 'entry']
        creates = [e for e in entries if e['argv'][:2] == ['create', 'task']]
        results = {e['invocation']: e for e in log if e['event'] == 'result'}
        children = verify.read_children(second / 'project/shark-tasks.db', binding)
        missing = verify.resumed_phase_checks(second, binding, entries, results, creates, creates, children,
                                              [], log, second / 'project', FINGERPRINT)
        self.assertIn('exactly one forwarded create across both phases', missing)
        self.assertIn('zero creates and zero writes after resumption', missing)
        self.assertIn('the original item is unchanged and still the only child', missing)


if __name__ == '__main__':
    unittest.main()
