#!/usr/bin/env python3
import json
import os
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
from adaptive_review import Capabilities, select_runner, valid_adversarial_output  # noqa: E402


class AdaptiveReviewTests(unittest.TestCase):
    def test_capability_ladder(self):
        self.assertEqual(select_runner(Capabilities(True, True, True, True))['runner_mode'], 'canonical-six-angle')
        self.assertEqual(select_runner(Capabilities(False, True, True, True))['runner_mode'], 'dispatched-six-angle')
        self.assertEqual(select_runner(Capabilities(False, False, True, False), 'claude')['adversarial_model'], 'codex')
        self.assertEqual(select_runner(Capabilities(False, False, False, True), 'codex')['adversarial_model'], 'claude')
        self.assertEqual(select_runner(Capabilities(False, False, False, False))['runner_mode'], 'incomplete')

    def test_output_validation_rejects_partial(self):
        good = 'REVIEWED SCOPE: 2 files\nFINDINGS: none\nThe change is safe and complete.\nVERDICT: PASS'
        self.assertTrue(valid_adversarial_output(good))
        for bad in ('', 'VERDICT: PASS', 'authentication failed', 'REVIEWED SCOPE\nFINDINGS'):
            self.assertFalse(valid_adversarial_output(bad))

    def test_cli_success_persists_metadata(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            fake = root / 'fake-reviewer'
            fake.write_text("#!/bin/sh\nprintf '%s\\n' 'REVIEWED SCOPE: fixture files and coverage' 'FINDINGS: none' 'The adversarial review is complete and evidence was checked.' 'VERDICT: PASS'\n")
            fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
            report = root / 'report.md'
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(root), '--diff-path', '/tmp/diff',
                '--base-commit', 'abc123', '--review-output-path', str(report), '--host', 'claude',
                '--codex-command', str(fake), '--timeout', '2', '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            metadata = json.loads(proc.stdout)
            self.assertEqual(metadata['runner_mode'], 'adversarial-cli')
            self.assertIn('Base commit:', report.read_text())
            self.assertIn('`abc123`', report.read_text())
            self.assertIn('Adversarial model:', report.read_text())
            self.assertIn('`codex`', report.read_text())

    def test_cli_failures_are_incomplete(self):
        cases = {
            "empty": "#!/bin/sh\nexit 0\n",
            "malformed": "#!/bin/sh\nprintf '%s\\n' 'VERDICT: PASS'\n",
            "auth": "#!/bin/sh\nprintf '%s\\n' 'authentication failed' >&2\nexit 1\n",
            "timeout": "#!/bin/sh\nsleep 2\n",
        }
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            for name, contents in cases.items():
                fake = root / name
                fake.write_text(contents)
                fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
                report = root / f'{name}.md'
                proc = subprocess.run([
                    sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                    '--prompt-file', str(prompt), '--project-root', str(root), '--diff-path', '/tmp/diff',
                    '--base-commit', 'abc123', '--review-output-path', str(report), '--host', 'claude',
                    '--codex-command', str(fake), '--timeout', '0.1', '--no-workflow-available', '--no-agent-dispatch-available',
                ], capture_output=True, text=True, check=False)
                self.assertEqual(proc.returncode, 2, name)
                self.assertIn('INCOMPLETE', report.read_text(), name)

    def test_no_automated_runner_persists_incomplete(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            report = root / 'report.md'
            env = {**os.environ, 'PATH': '/nonexistent'}
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(root), '--diff-path', '/tmp/diff',
                '--base-commit', 'abc123', '--review-output-path', str(report), '--host', 'unknown',
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], env=env, capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 2)
            self.assertIn('runner_mode=incomplete', report.read_text())


if __name__ == '__main__':
    unittest.main()
