"""Verifier diagnostics are distinct from independent host execution evidence."""

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from observe import CONSOLIDATOR, REVIEW, TRIAGE, WORKFLOW
from test_packet import write_test_packet

HERE = Path(__file__).resolve().parent


class Verify(unittest.TestCase):
    def test_tracker_and_verifier_require_pinned_handoff_reads(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'pair'
            root.mkdir()
            repo, scope, report, report_hash = write_test_packet(Path(tmp))
            subprocess.run([sys.executable, str(HERE / 'prepare.py'), root, '--retained',
                            report, report_hash, repo, scope['candidate_commit']],
                           check=True, capture_output=True)
            verify = [sys.executable, str(HERE / 'verify.py'), root, 'source#DOCS#usage']
            for name in ('grant-a', 'grant-b'):
                case = root / name
                project = case / 'project'
                subprocess.run(['./shark', 'skill', 'get', 'quality', 'workflows/review-code.md', '--raw'],
                               cwd=project, check=True, capture_output=True)
                for relative in (TRIAGE, REVIEW, CONSOLIDATOR):
                    subprocess.run([sys.executable, str(HERE / 'observe.py'), case, 'read', relative],
                                   check=True, capture_output=True)
                log = case / 'observations/tools.jsonl'
                original = log.read_bytes()
                tracker = ['./shark', 'task', 'list', '--json']
                subprocess.run(tracker, cwd=project, check=True, capture_output=True)
                control = subprocess.run(verify, capture_output=True)
                self.assertNotIn('private handoff reads', json.loads(control.stdout)[name]['missing'])
                for relative in (WORKFLOW, TRIAGE):
                    for field, replacement in [('origin', '/outside/shared'), ('sha256', '0' * 64)]:
                        with self.subTest(case=name, policy=relative, field=field):
                            events = [json.loads(line) for line in original.splitlines()]
                            load = next(e for e in events if e.get('file') == relative)
                            load[field] = replacement
                            log.write_text(''.join(json.dumps(e) + '\n' for e in events))
                            checked = subprocess.run(verify, capture_output=True)
                            self.assertIn('private handoff reads', json.loads(checked.stdout)[name]['missing'])
                        with self.subTest(case=name, tracker=relative, field=field):
                            checked = subprocess.run(tracker, cwd=project, capture_output=True)
                            self.assertNotEqual(checked.returncode, 0)
                            self.assertIn(b'private review handoff reads', checked.stderr)
                            self.assertFalse(any(json.loads(line)['event'] == 'forward'
                                                 for line in log.read_text().splitlines()))

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
