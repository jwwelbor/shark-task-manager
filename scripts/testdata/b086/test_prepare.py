"""Setup tests. These never stand in for host replay evidence."""

import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess
import sys
import tempfile
import unittest


HERE = Path(__file__).resolve().parent
DRIVER = HERE / 'prepare.py'
POLICY = '73910684da8db7e4a045ddc5a22577c160eef328'


class GrantPairSetup(unittest.TestCase):
    def test_pair_starts_from_one_seed_and_pins_private_policy(self):
        with tempfile.TemporaryDirectory(prefix='b086-setup-test-') as tmp:
            setup = subprocess.run([sys.executable, str(DRIVER), tmp],
                                   capture_output=True, text=True)
            self.assertEqual(setup.returncode, 0, setup.stderr)
            manifest = json.loads(Path(setup.stdout.strip()).read_text())
            self.assertEqual(manifest['policy_commit'], POLICY)
            cases = [Path(tmp) / case for case in manifest['cases']]
            self.assertEqual(len(cases), 2)
            inputs = []
            stores = []
            common_dispatches = []
            for case in cases:
                dispatch = json.loads((case / 'grant.json').read_text())
                inputs.append(dispatch.pop('create_task_under'))
                common_dispatches.append(dispatch)
                self.assertEqual(dispatch['mode'], 'unattended')
                self.assertEqual(dispatch['requested_feature'], 'E01-F01')
                self.assertIs(dispatch.get('edit_authority'), False,
                              'the host needs the scope of its grant before considering a Fix')
                self.assertEqual(dispatch.get('budget'), {
                    'dispatch_remaining_attempts': 1, 'project_total_attempts': 2,
                    'project_consumed_attempts': 1, 'available_minutes': 12,
                    'estimated_repair_and_verification_minutes': 10,
                })
                db = case / 'project/shark-tasks.db'
                stores.append(db)
                with sqlite3.connect(db.as_uri() + '?mode=ro', uri=True) as con:
                    self.assertEqual(con.execute('select key from features order by key').fetchall(),
                                     [('E01-F01',), ('E01-F02',)])
                    self.assertEqual(con.execute('select count(*) from tasks').fetchone(), (0,))
                for relative, digest in manifest['prepared_files'].items():
                    private = (case / relative).resolve(strict=True)
                    self.assertTrue(private.is_relative_to(case))
                    self.assertEqual(hashlib.sha256(private.read_bytes()).hexdigest(), digest)
                self.assertIn('catalog/shark-rider/skills/triage/SKILL.md', manifest['prepared_files'])
                self.assertIn('catalog/finish-feature/SKILL.md', manifest['prepared_files'])
                self.assertIn('project/shark', manifest['prepared_files'])
            self.assertEqual(inputs, ['E01-F01', 'E01-F02'])
            self.assertEqual(*common_dispatches)
            self.assertFalse(stores[0].samefile(stores[1]))
            self.assertEqual(stores[0].read_bytes(), stores[1].read_bytes())


if __name__ == '__main__':
    unittest.main()
