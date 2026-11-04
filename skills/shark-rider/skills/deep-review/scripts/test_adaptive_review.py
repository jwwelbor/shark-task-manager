#!/usr/bin/env python3
import json
import os
import shlex
import stat
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
from adaptive_review import (  # noqa: E402
    Capabilities,
    cli_candidates,
    select_runner,
    valid_adversarial_output,
    validate_review_decisions,
)


def scope_fixture(root):
    repo = root / 'repo'
    repo.mkdir()
    def git(*args):
        return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], cwd=repo, check=True, capture_output=True, text=True).stdout.strip()
    git('init', '-b', 'main')
    git('config', 'commit.gpgsign', 'false')
    git('config', 'user.name', 'Test')
    git('config', 'user.email', 'test@example.invalid')
    (repo / 'source').write_text('old')
    git('add', 'source')
    git('commit', '-m', 'base')
    git('checkout', '-b', 'candidate')
    (repo / 'source').write_text('new')
    git('commit', '-am', 'candidate')
    capture = subprocess.run(['bash', str(HERE / 'get_diff.sh')], cwd=repo, check=True, capture_output=True, text=True)
    scope = json.loads(capture.stdout)
    scope_file = root / 'scope.json'
    scope_file.write_text(capture.stdout)
    return repo, scope, scope_file


def review_packet(scope):
    identity = {'commit': scope['candidate_commit'], 'tree': scope['candidate_tree'], 'diff_sha256': scope['diff_sha256']}
    return {'report': 'REVIEWED SCOPE: fixture files and coverage\nFINDINGS: none\nThe adversarial review is complete and evidence was checked.\nVERDICT: PASS',
            'decisions': {'candidate_identity': identity, 'findings': []}}


def fake_reviewer(path, packet):
    path.write_text("#!/bin/sh\nprintf '%s\\n' " + shlex.quote(json.dumps(packet)) + "\n")
    path.chmod(path.stat().st_mode | stat.S_IXUSR)


class AdaptiveReviewTests(unittest.TestCase):
    def test_capability_ladder(self):
        self.assertEqual(select_runner(Capabilities(True, True, True, True))['runner_mode'], 'canonical-six-angle')
        self.assertEqual(select_runner(Capabilities(False, True, True, True))['runner_mode'], 'dispatched-six-angle')
        self.assertEqual(select_runner(Capabilities(False, False, True, False), 'claude')['adversarial_model'], 'codex')
        self.assertEqual(select_runner(Capabilities(False, False, False, True), 'codex')['adversarial_model'], 'claude')
        self.assertEqual(select_runner(Capabilities(False, False, True, False), 'codex')['adversarial_model'], 'codex')
        self.assertEqual(select_runner(Capabilities(False, False, False, False))['runner_mode'], 'incomplete')

    def test_cli_candidates_retry_host_cli_after_preferred_alternate(self):
        self.assertEqual(
            cli_candidates(Capabilities(False, False, True, True), 'codex', 'claude'),
            ['claude', 'codex'],
        )

    def test_output_validation_rejects_partial(self):
        good = 'REVIEWED SCOPE: 2 files\nFINDINGS: none\nThe change is safe and complete.\nVERDICT: PASS'
        self.assertTrue(valid_adversarial_output(good))
        for bad in ('', 'VERDICT: PASS', 'authentication failed', 'REVIEWED SCOPE\nFINDINGS'):
            self.assertFalse(valid_adversarial_output(bad))

    def test_review_decisions_require_explicit_refutation(self):
        scope = {'candidate_commit': 'c', 'candidate_tree': 't', 'diff_sha256': 'd'}
        blocker = {'file': 'a.go', 'line': 3, 'summary': 'broken', 'severity': 'blocker', 'rule': 'BUG', 'diagnosis': 'wrong'}
        decisions = {'candidate_identity': {'commit': 'c', 'tree': 't', 'diff_sha256': 'd'}, 'findings': []}
        with self.assertRaises(ValueError):
            validate_review_decisions(decisions, scope, [blocker], 'PASS')
        decisions['findings'] = [{**blocker, 'disposition': 'open', 'reason': 'not refuted'}]
        with self.assertRaises(ValueError):
            validate_review_decisions(decisions, scope, [blocker], 'PASS')
        decisions['findings'][0]['disposition'] = 'refuted'
        decisions['findings'][0]['reason'] = 'evidence shows the caller cannot reach this branch'
        validate_review_decisions(decisions, scope, [blocker], 'PASS')

    def test_cli_success_persists_metadata(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            fake = root / 'fake-reviewer'
            fake_reviewer(fake, review_packet(scope))
            report = root / 'report.md'
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'claude',
                '--codex-command', str(fake), '--timeout', '2', '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            metadata = json.loads(proc.stdout)
            self.assertEqual(metadata['runner_mode'], 'adversarial-cli')
            self.assertIn('Base commit:', report.read_text())
            self.assertIn(scope['base_commit'], report.read_text())
            self.assertIn('Adversarial model:', report.read_text())
            self.assertIn('`codex`', report.read_text())

    def test_cli_pass_with_open_blocker_is_incomplete(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            packet = review_packet(scope)
            packet['report'] = 'REVIEWED SCOPE: all files. FINDINGS: blockers remain despite a thorough check of all callers. VERDICT: PASS'
            packet['decisions']['findings'] = [{'file': 'source', 'line': 1, 'severity': 'blocker', 'rule': 'BUG', 'summary': 'broken', 'diagnosis': 'wrong', 'disposition': 'open', 'reason': 'not refuted'}]
            fake = root / 'reviewer'
            fake_reviewer(fake, packet)
            result = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'],
                '--review-output-path', str(root / 'report.md'), '--host', 'claude',
                '--codex-command', str(fake), '--claude-command', str(fake),
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)
            self.assertEqual(json.loads(result.stdout)['verdict'], 'INCOMPLETE')

    def test_cli_rejects_missing_scope_and_wrong_execution_root(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            called = root / 'reviewer-called'
            fake = root / 'reviewer'
            fake.write_text(f'#!/bin/sh\ntouch "{called}"\n')
            fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
            common = [sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                      '--prompt-file', str(prompt), '--diff-path', scope['diff_path'],
                      '--base-commit', scope['base_commit'], '--review-output-path', str(root / 'report.md'),
                      '--host', 'claude', '--codex-command', str(fake),
                      '--no-workflow-available', '--no-agent-dispatch-available']
            for extra in (['--project-root', str(repo)],
                          ['--project-root', str(root), '--scope-file', str(scope_file)]):
                result = subprocess.run(common + extra, capture_output=True, text=True)
                self.assertEqual(result.returncode, 2, result.stderr)
                self.assertEqual(json.loads(result.stdout)['verdict'], 'INCOMPLETE')
                self.assertFalse(called.exists())

    def test_cli_failures_are_incomplete(self):
        cases = {
            "empty": "#!/bin/sh\nexit 0\n",
            "malformed": "#!/bin/sh\nprintf '%s\\n' 'VERDICT: PASS'\n",
            "auth": "#!/bin/sh\nprintf '%s\\n' 'authentication failed' >&2\nexit 1\n",
            "timeout": "#!/bin/sh\nsleep 2\n",
        }
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            for name, contents in cases.items():
                fake = root / name
                fake.write_text(contents)
                fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
                report = root / f'{name}.md'
                proc = subprocess.run([
                    sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                    '--prompt-file', str(prompt), '--project-root', str(repo), '--diff-path', scope['diff_path'],
                    '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'claude',
                    '--codex-command', str(fake), '--timeout', '0.1', '--no-workflow-available', '--no-agent-dispatch-available',
                ], capture_output=True, text=True, check=False)
                self.assertEqual(proc.returncode, 2, name)
                self.assertIn('INCOMPLETE', report.read_text(), name)

    def test_cli_retries_comparable_host_cli_after_preferred_failure(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            failed = root / 'failed-reviewer'
            failed.write_text('#!/bin/sh\nexit 1\n')
            failed.chmod(failed.stat().st_mode | stat.S_IXUSR)
            successful = root / 'codex-reviewer'
            fake_reviewer(successful, review_packet(scope))
            report = root / 'report.md'
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'codex',
                '--claude-command', str(failed), '--codex-command', str(successful), '--timeout', '2',
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            metadata = json.loads(proc.stdout)
            self.assertEqual(metadata['adversarial_model'], 'codex')
            self.assertIn('using codex CLI', report.read_text())

    def test_no_automated_runner_persists_incomplete(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            report = root / 'report.md'
            env = {**os.environ, 'PATH': '/nonexistent'}
            proc = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'unknown',
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], env=env, capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 2)
            self.assertIn('runner_mode=incomplete', report.read_text())


if __name__ == '__main__':
    unittest.main()
