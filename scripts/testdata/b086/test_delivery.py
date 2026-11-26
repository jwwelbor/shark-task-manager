"""Delivery admission: the CI endpoint records a handoff only where a case enables it."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import host
from test_destination import observer, events

HERE = Path(__file__).resolve().parent


class Delivery(unittest.TestCase):
    def test_ci_records_only_in_delivery_cases_and_merge_stays_disabled(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            done = subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic',
                                   '--scenario', 'ship:delivery-defer', '--scenario', 'hold:delivery-refused'],
                                  capture_output=True)
            self.assertEqual(done.returncode, 0, done.stderr)
            manifest = json.loads((root / 'preparation.json').read_bytes())
            for name in ('ship', 'hold'):
                binding = manifest['bindings'][name]
                self.assertTrue(binding['delivery_ci'])
                self.assertIn('Delivery:', host.prompt(root / name))
                self.assertEqual(hashlib.sha256(host.prompt(root / name).encode()).hexdigest(),
                                 binding['prompt_sha256'])
                self.assertEqual(observer(root / name, 'ci').returncode, 0)
                self.assertNotEqual(observer(root / name, 'merge').returncode, 0)
                outcomes = [e['outcome'] for e in events(root / name) if e['event'] == 'result']
                self.assertEqual(outcomes, ['recorded', 'refused'])
            self.assertIn('create_refusal', manifest['bindings']['hold'])
            self.assertNotIn('create_refusal', manifest['bindings']['ship'])
            # The first-pair cases keep the endpoint disabled and the base prompt.
            self.assertNotEqual(observer(root / 'grant-a', 'ci').returncode, 0)
            self.assertNotIn('Delivery:', host.prompt(root / 'grant-a'))


if __name__ == '__main__':
    unittest.main()
