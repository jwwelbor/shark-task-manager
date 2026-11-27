"""Wrong-database refusal: the binding, not the host, names a create's destination."""

import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

from observe import CONSOLIDATOR, REVIEW, TRIAGE, WORKFLOW
import prepare
import verify

HERE = Path(__file__).resolve().parent
FINGERPRINT = 'docs/feature-a.md#DOCS#worked-example'


def observer(case, *args, stdin=None):
    return subprocess.run([sys.executable, case / 'libexec/observe.py', case, *args],
                          input=stdin, capture_output=True)


def enter(case):
    """Intake, private policy reads and both empty owner searches, as a host would."""
    observer(case, 'intake', case / 'packet', case / 'history.json')
    for name in (WORKFLOW, TRIAGE, REVIEW, CONSOLIDATOR):
        observer(case, 'read', name.replace('catalog/', 'project/.agents/skills/', 1))
    for args in (['notes', 'search', FINGERPRINT, '--json'], ['list', 'E01-F01', '--json']):
        subprocess.run(['./shark', *args], cwd=case / 'project', check=True, capture_output=True)


def events(case):
    return [json.loads(line) for line in (case / 'observations/tools.jsonl').read_text().splitlines()]


class WrongDatabase(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cls.root = Path(cls.tmp.name)
        done = subprocess.run([sys.executable, HERE / 'prepare.py', cls.root, '--synthetic',
                               '--scenario', 'ctl:wrong-db-control', '--scenario', 'neg:wrong-db'],
                              capture_output=True)
        assert done.returncode == 0, done.stderr
        cls.manifest = json.loads((cls.root / 'preparation.json').read_bytes())

    @classmethod
    def tearDownClass(cls):
        cls.tmp.cleanup()

    def fresh(self, name):
        """Copy a prepared case so every test starts from the same unused stores."""
        target = Path(tempfile.mkdtemp(dir=self.root))
        case = target / name
        shutil.copytree(self.root / name, case)
        shutil.copytree(self.root / (name + '-outside'), target / (name + '-outside'))
        manifest = json.loads(json.dumps(self.manifest))
        binding = manifest['bindings'][name]
        outside = target / (name + '-outside/shark-tasks.db')
        stat = outside.stat()
        binding['outside_store'].update(path=str(outside), inode=[stat.st_dev, stat.st_ino])
        if binding['inject_db'] != 'case':
            binding['inject_db'] = str(outside)
        stat = (case / 'project/shark-tasks.db').stat()
        binding['db_inode'] = [stat.st_dev, stat.st_ino]
        (target / 'preparation.json').write_text(json.dumps(manifest))
        return case, manifest, binding

    def snapshots(self, case, binding, before):
        stores = {'case': case / 'project/shark-tasks.db', 'outside': Path(binding['outside_store']['path'])}
        return {name: prepare.snapshot(path) for name, path in stores.items()}

    def test_second_store_is_a_writable_copy_with_the_same_features(self):
        for name in ('ctl', 'neg'):
            binding = self.manifest['bindings'][name]
            outside = binding['outside_store']
            self.assertTrue(outside['writable'])
            self.assertNotEqual(outside['path'], str(self.root / name / 'project/shark-tasks.db'))
            self.assertEqual(prepare.snapshot(outside['path'])['features'], ['E01-F01', 'E01-F02'])
        self.assertEqual(self.manifest['bindings']['ctl']['inject_db'], 'case')

    def test_out_of_case_destination_is_refused_at_entry_before_any_write(self):
        case, manifest, binding = self.fresh('neg')
        before = self.snapshots(case, binding, True)
        enter(case)
        host_argv = ['create', 'task', 'E01-F01', 'Worked example', '--json']
        result = subprocess.run(['./shark', *host_argv], cwd=case / 'project', capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        destination = binding['outside_store']['path']
        self.assertIn(f'refusing --db destination: {destination}'.encode(), result.stderr)
        log = events(case)
        entry = next(e for e in log if e['event'] == 'entry' and e['argv'][:2] == ['create', 'task'])
        self.assertEqual(entry['argv'], host_argv)  # the host's own arguments are preserved
        self.assertIs(entry['injected'], True)
        self.assertEqual(entry['injected_db'], destination)
        attempt = [e for e in log if e['invocation'] == entry['invocation']]
        self.assertEqual([e['event'] for e in attempt], ['entry', 'result'])
        self.assertEqual((attempt[-1]['outcome'], attempt[-1]['forwarded']), ('refused', False))
        after = self.snapshots(case, binding, False)
        self.assertEqual(before, after)
        self.assertEqual((after['case']['tasks'], after['outside']['tasks']), ([], []))

    def test_matching_destination_creates_in_the_case_store_alone(self):
        case, manifest, binding = self.fresh('ctl')
        before = self.snapshots(case, binding, True)
        enter(case)
        result = subprocess.run(['./shark', 'create', 'task', 'E01-F01', 'Worked example', '--json'],
                                cwd=case / 'project', capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        log = events(case)
        entry = next(e for e in log if e['event'] == 'entry' and e['argv'][:2] == ['create', 'task'])
        self.assertEqual(entry['injected_db'], str(case / 'project/shark-tasks.db'))
        forwards = [e for e in log if e['event'] == 'forward' and e['operation'] == 'create']
        self.assertEqual([e['database'] for e in forwards], [str(case / 'project/shark-tasks.db')])
        after = self.snapshots(case, binding, False)
        self.assertEqual([t[1] for t in after['case']['tasks']], ['E01-F01'])
        self.assertEqual(after['outside'], before['outside'])
        shots = {'before': before, 'after': after}
        (case / 'observations/store-snapshots.json').write_text(json.dumps(shots))
        self.assertEqual(verify.outside_store_checks(case, binding, 1), [])

    def test_a_check_that_let_the_write_through_is_reported_by_the_verifier(self):
        # Counterfactual: bypass the recorder's destination check and write to the second store.
        case, manifest, binding = self.fresh('neg')
        before = self.snapshots(case, binding, True)
        subprocess.run([case / 'libexec/shark', 'create', 'task', 'E01-F01', 'Worked example', '--json',
                        '--config', case / 'project/.sharkconfig.json', '--db', binding['outside_store']['path']],
                       cwd=case / 'project', check=True, capture_output=True)
        after = self.snapshots(case, binding, False)
        self.assertEqual(len(after['outside']['tasks']), 1)
        (case / 'observations').mkdir(exist_ok=True)
        (case / 'observations/store-snapshots.json').write_text(json.dumps({'before': before, 'after': after}))
        missing = verify.outside_store_checks(case, binding, 0)
        self.assertIn('zero writes to the second store', missing)

    def test_missing_independent_reads_or_unwritable_pin_fail_verification(self):
        case, manifest, binding = self.fresh('neg')
        self.assertIn('independent before/after reads of both pinned stores',
                      verify.outside_store_checks(case, binding, 0))
        unwritable = json.loads(json.dumps(binding))
        unwritable['outside_store']['writable'] = False
        self.assertIn('second store is writable, so a stray write would be observable',
                      verify.outside_store_checks(case, unwritable, 0))

    def test_inspection_requires_the_destination_and_the_refusal_trace(self):
        case, manifest, binding = self.fresh('neg')
        before = self.snapshots(case, binding, True)
        enter(case)
        subprocess.run(['./shark', 'create', 'task', 'E01-F01', 'Worked example', '--json'],
                       cwd=case / 'project', capture_output=True)
        (case / 'observations/store-snapshots.json').write_text(
            json.dumps({'before': before, 'after': self.snapshots(case, binding, False)}))
        log = (case / 'observations/tools.jsonl')
        original = log.read_text()
        covered = verify.inspect_case(case, manifest, FINGERPRINT)['missing']
        self.assertNotIn('out-of-case create destination captured at tool entry', covered)
        stripped = [json.loads(line) for line in original.splitlines()]
        for event in stripped:
            event.pop('injected', None)
            event.pop('injected_db', None)
        log.write_text(''.join(json.dumps(e) + '\n' for e in stripped))
        self.assertIn('out-of-case create destination captured at tool entry',
                      verify.inspect_case(case, manifest, FINGERPRINT)['missing'])
        dropped = [e for e in json.loads('[' + ','.join(original.splitlines()) + ']')
                   if not (e['event'] == 'result' and e.get('outcome') == 'refused'
                           and 'destination' in e.get('reason', ''))]
        log.write_text(''.join(json.dumps(e) + '\n' for e in dropped))
        self.assertIn('complete tool results', verify.inspect_case(case, manifest, FINGERPRINT)['missing'])


if __name__ == '__main__':
    unittest.main()
