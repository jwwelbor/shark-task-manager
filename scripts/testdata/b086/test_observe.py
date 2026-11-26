"""Test the fixture's observation interfaces, without running a host scenario."""

import json
import os
from pathlib import Path
import sqlite3
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent


class RecorderTest(unittest.TestCase):
    def test_project_shark_records_normal_host_commands(self):
        with tempfile.TemporaryDirectory() as root:
            subprocess.run([sys.executable, str(HERE / 'prepare.py'), root],
                           check=True, capture_output=True)
            case = Path(root) / 'grant-a'
            project = case / 'project'
            env = {**os.environ, 'PATH': str(project) + os.pathsep + os.environ['PATH']}
            for entry in ('./shark', 'shark'):
                result = subprocess.run([entry, 'create', 'task', 'E01-F01', 'Usage example'],
                                        cwd=project, env=env, capture_output=True, text=True)
                self.assertNotEqual(result.returncode, 0,
                                    'ordinary scratch Shark bypassed the missing-packet check')
                self.assertIn('missing retained packet', result.stderr)
            with sqlite3.connect((project / 'shark-tasks.db').as_uri() + '?mode=ro', uri=True) as db:
                self.assertEqual(db.execute('SELECT count(*) FROM tasks').fetchone(), (0,))
            workflow = subprocess.run(['shark', 'skill', 'get', 'quality', 'workflows/review-code.md', '--raw'],
                                      cwd=project, env=env, check=True, capture_output=True)
            events = [json.loads(line) for line in (case / 'observations/tools.jsonl').read_text().splitlines()]
            results = [e for e in events if e['event'] == 'result']
            self.assertEqual([e['outcome'] for e in results], ['refused', 'refused', 'loaded'])
            manifest = json.loads((Path(root) / 'preparation.json').read_bytes())
            self.assertEqual(results[-1]['sha256'], manifest['embedded_workflow_sha256'])
            self.assertIn(b'unattended review disposition', workflow.stdout)

    def test_recorders_keep_attempts_when_packet_is_missing(self):
        with tempfile.TemporaryDirectory() as root:
            cmd = [sys.executable, str(HERE / 'prepare.py'), root]
            result = subprocess.run(cmd, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            case = Path(root) / 'grant-a'
            observer = [sys.executable, str(HERE / 'observe.py'), str(case)]
            for requested, reason in [(str(case / 'project/shark-tasks.db'), 'missing retained packet'),
                                      (str(Path(root) / 'grant-b/project/shark-tasks.db'), 'refusing --db')]:
                cmd = [*observer, 'shark', 'create', 'task', 'E01-F01', 'probe', '--db', requested]
                result = subprocess.run(cmd, capture_output=True, text=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue((case / 'observations/tools.jsonl').is_file(),
                                'Refused tracker attempts must be visible at tool entry')
                events = [json.loads(line) for line in (case / 'observations/tools.jsonl').read_text().splitlines()]
                self.assertEqual(events[-2]['event'], 'entry')
                self.assertEqual(events[-1]['outcome'], 'refused')
                self.assertFalse(events[-1]['forwarded'])
                self.assertIn(reason, events[-1]['reason'])
                self.assertEqual(events[-2]['argv'][-1], requested)
            policy = 'catalog/shark-rider/skills/triage/SKILL.md'
            loaded = subprocess.run([*observer, 'read', policy], capture_output=True)
            self.assertEqual(loaded.stdout, (case / policy).read_bytes())
            self.assertEqual(loaded.returncode, 0)
            (case / policy).write_text('stale installation')
            refused = subprocess.run([*observer, 'read', policy], capture_output=True)
            self.assertNotEqual(refused.returncode, 0)
            self.assertEqual(refused.stdout, b'')
            inbox = subprocess.run([*observer, 'parent'], input='{"fingerprint":"fixture#DOCS#example"}',
                                   capture_output=True, text=True)
            self.assertEqual(inbox.returncode, 0, inbox.stderr)
            received = json.loads((case / 'observations/parent.jsonl').read_text())
            self.assertEqual(received['message']['fingerprint'], 'fixture#DOCS#example')
            self.assertEqual(received['run_id'], events[0]['run_id'])
            for tool in ['ci', 'merge', 'review']:
                result = subprocess.run([*observer, tool], capture_output=True, text=True)
                self.assertNotEqual(result.returncode, 0)
            events = [json.loads(line) for line in (case / 'observations/tools.jsonl').read_text().splitlines()]
            self.assertEqual([e['tool'] for e in events if e['event'] == 'entry'],
                             ['shark', 'shark', 'read', 'read', 'parent', 'ci', 'merge', 'review'])


if __name__ == '__main__':
    unittest.main()
