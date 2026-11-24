"""Verifier diagnostics are distinct from independent host execution evidence."""

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent


class Verify(unittest.TestCase):
    def test_seed_without_host_observations_cannot_verify_the_pair(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            subprocess.run([sys.executable, str(HERE / 'prepare.py'), tmp], check=True, capture_output=True)
            result = subprocess.run([sys.executable, str(HERE / 'verify.py'), tmp,
                                     'docs/example.md#DOCS#usage'], text=True, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertTrue(result.stdout, 'a missing observation must identify the blocked cases')
            verdict = json.loads(result.stdout)
            for case in ('grant-a', 'grant-b'):
                self.assertEqual(verdict[case]['children'], [])
                self.assertEqual(verdict[case]['status'], 'incomplete')
                self.assertIn('retained packet', verdict[case]['missing'])
                self.assertIn('host tool trace', verdict[case]['missing'])
            self.assertIn('host readback and Defer receipt', verdict['grant-a']['missing'])


if __name__ == '__main__':
    unittest.main()
