#!/usr/bin/env python3
import argparse
import contextlib
import io
import json
import os
import shlex
import stat
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import adaptive_review  # noqa: E402
from adaptive_review import (  # noqa: E402
    Capabilities,
    cli_candidates,
    select_runner,
    valid_adversarial_output,
    validate_review_decisions,
    run_bounded_reviewer,
)


class BoundedReviewerTest(unittest.TestCase):
    def test_windows_fails_closed_before_launch(self):
        with patch('adaptive_review.os.name', 'nt'), patch('adaptive_review.subprocess.Popen') as process:
            with self.assertRaisesRegex(OSError, 'process-tree cleanup'):
                run_bounded_reviewer(['reviewer'], None, 1)
            process.assert_not_called()

    @unittest.skipIf(os.name == 'nt', 'uses POSIX process groups')
    def test_excess_output_is_bounded_and_rejected(self):
        script = "import sys; sys.stdout.write('x' * (9 * 1024 * 1024)); sys.stdout.flush()"
        result = run_bounded_reviewer([sys.executable, '-c', script], None, 5)
        self.assertNotEqual(result.returncode, 0)
        self.assertLessEqual(len(result.stdout.encode()), 8 * 1024 * 1024)

    @unittest.skipIf(os.name == 'nt', 'uses POSIX process groups')
    def test_closed_output_pipes_do_not_bypass_timeout(self):
        script = "import os,time; os.close(1); os.close(2); time.sleep(10)"
        start = time.monotonic()
        result = run_bounded_reviewer([sys.executable, '-c', script], None, 0.2)
        self.assertNotEqual(result.returncode, 0)
        self.assertLess(time.monotonic() - start, 3)

    @unittest.skipIf(os.name == 'nt', 'uses POSIX process groups')
    def test_parent_exit_with_child_holding_stderr_is_bounded(self):
        with tempfile.TemporaryDirectory() as tmp:
            marker = Path(tmp) / 'child-survived'
            child = "import time,pathlib; time.sleep(1); pathlib.Path(" + repr(str(marker)) + ").write_text('alive')"
            parent = "import subprocess,sys; subprocess.Popen([sys.executable, '-c', " + repr(child) + "], stdout=subprocess.DEVNULL); print('parent exited')"
            result = run_bounded_reviewer([sys.executable, '-c', parent], tmp, 0.2)
            self.assertNotEqual(result.returncode, 0)
            time.sleep(1.1)
            self.assertFalse(marker.exists())


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
    @unittest.skipIf(os.name == 'nt', 'uses shell executable fixture')
    def test_scope_timeout_produces_structured_failure_and_report(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            fake_bin = root / 'bin'
            fake_bin.mkdir()
            git = fake_bin / 'git'
            git.write_text('#!/bin/sh\nsleep 3\n')
            git.chmod(0o700)
            env = {**os.environ, 'PATH': str(fake_bin) + os.pathsep + os.environ['PATH'], 'SHARK_REVIEW_GIT_TIMEOUT': '0.2'}
            script = str(HERE / 'adaptive_review.py')
            validate = subprocess.run([sys.executable, script, 'validate-scope', '--scope-file', str(scope_file), '--project-root', str(repo), '--source-root', str(repo)], env=env, capture_output=True, text=True)
            self.assertEqual(validate.returncode, 2, validate.stderr)
            self.assertFalse(json.loads(validate.stdout)['valid'])
            body = root / 'body'
            body.write_text('Reviewed scope: all files and callers. Findings: none. Detailed verification completed. VERDICT: PASS')
            report = root / 'report.md'
            writer = subprocess.run([sys.executable, script, 'write-report', '--body-file', str(body), '--project-root', str(repo), '--source-root', str(repo), '--review-output-path', str(report), '--runner-mode', 'adversarial-cli', '--specialists-completed', '0', '--consolidator-completed', 'true', '--fallback-reason', 'native unavailable', '--adversarial-model', 'codex', '--base-commit', scope['base_commit'], '--diff-path', scope['diff_path'], '--verdict', 'PASS', '--scope-file', str(scope_file)], env=env, capture_output=True, text=True)
            self.assertEqual(writer.returncode, 2, writer.stderr)
            self.assertEqual(json.loads(writer.stdout)['verdict'], 'INCOMPLETE')
            self.assertTrue(report.exists())
            fake_shark = fake_bin / 'shark'
            fake_shark.write_text('#!/bin/sh\nsleep 3\n')
            fake_shark.chmod(0o700)
            env.update({'SHARK_REVIEW_IO_FORCE': '1', 'SHARK_REVIEW_INVENTORY_BIN': str(fake_shark)})
            second_report = root / 'io-timeout-report.md'
            args = writer.args[:]
            args[args.index('--review-output-path') + 1] = str(second_report)
            failed_io = subprocess.run(args, env=env, capture_output=True, text=True)
            self.assertEqual(failed_io.returncode, 2, failed_io.stderr)
            self.assertEqual(json.loads(failed_io.stdout)['verdict'], 'INCOMPLETE')
            self.assertTrue(second_report.exists())

    def test_failed_report_can_describe_product_finding(self):
        body = ('CURRENT_REVIEW_STATUS: COMPLETE\n'
                'FINDING_DETAIL: application has incomplete validation of external input.\n'
                'Scope: all candidate files reviewed. Findings: validation bug. VERDICT: FAIL')
        self.assertTrue(adaptive_review.valid_review_report(body, 'FAIL'))
        self.assertFalse(adaptive_review.valid_review_report(body + '\nThe current review is incomplete.', 'FAIL'))

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

    def test_failed_review_keeps_every_raw_finding(self):
        scope = {'candidate_commit': 'c', 'candidate_tree': 't', 'diff_sha256': 'd'}
        finding = {'file': 'a.go', 'line': 3, 'summary': 'broken', 'severity': 'blocker', 'rule': 'BUG', 'diagnosis': 'wrong'}
        decisions = {'candidate_identity': {'commit': 'c', 'tree': 't', 'diff_sha256': 'd'}, 'findings': []}
        with self.assertRaisesRegex(ValueError, 'no explicit disposition'):
            validate_review_decisions(decisions, scope, [finding], 'FAIL')
        decisions['findings'] = [{**finding, 'disposition': 'open', 'reason': 'still reproducible'}]
        validate_review_decisions(decisions, scope, [finding], 'FAIL')

    def test_report_retains_validated_bytes_after_source_replacement(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            identity = review_packet(scope)['decisions']['candidate_identity']
            angles = root / 'angles.json'
            angles.write_text(json.dumps({name: {'candidate_identity': identity, 'reviewed_files': scope['changed_files'], 'findings': []} for name in 'abcdef'}))
            decisions = root / 'decisions.json'
            decisions.write_text(json.dumps({'candidate_identity': identity, 'findings': []}))
            body = root / 'body'
            body.write_text(review_packet(scope)['report'])
            report = root / 'report'
            diff = Path(scope['diff_path'])
            expected = {str(path): path.read_bytes() for path in (angles, diff)}
            original_read = adaptive_review.read_review_file
            def replace_after_read(path, limit):
                data = original_read(path, limit)
                if str(path) in expected:
                    Path(path).write_bytes(b'replaced')
                return data
            args = argparse.Namespace(runner_mode='dispatched-six-angle', specialists_completed=6, consolidator_completed=True,
                adversarial_model=None, fallback_reason='native Workflow unavailable', base_commit=scope['base_commit'], diff_path=str(diff), verdict='PASS',
                requested_model=None, effective_model=None, scope_file=str(scope_file), project_root=str(repo), source_root=str(repo),
                shark_project_root=None, angle_results=str(angles), review_decisions=str(decisions), body_file=str(body),
                review_output_path=str(report), branch='fixture')
            with patch('adaptive_review.read_review_file', side_effect=replace_after_read), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(adaptive_review.write_report(args), 0)
            self.assertEqual(Path(str(report) + '.angles.json').read_bytes(), expected[str(angles)])
            self.assertEqual(Path(str(report) + '.diff').read_bytes(), expected[str(diff)])

    def test_retained_artifacts_reject_output_above_shared_limit(self):
        with tempfile.TemporaryDirectory() as tmp, patch('adaptive_review.MAX_ARTIFACT_BYTES', 4):
            path = Path(tmp) / 'evidence'
            with self.assertRaisesRegex(ValueError, 'exceeds size limit'):
                adaptive_review._write_new_bytes(path, b'12345')
            self.assertFalse(path.exists())

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
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
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

    def test_cli_writer_requires_model_and_fallback_reason(self):
        for model, reason in [('', 'unavailable'), ('unknown', 'unavailable'), ('codex', ''), ('codex', 'none'), ('codex', '`unknown`'), ('codex', '` none `')]:
            args = argparse.Namespace(runner_mode='adversarial-cli', specialists_completed=0,
                                      consolidator_completed=True, adversarial_model=model, fallback_reason=reason, verdict='PASS')
            metadata = {'runner_mode': args.runner_mode, 'verdict': 'PASS', 'fallback_reason': reason}
            self.assertTrue(adaptive_review._writer_runner_state(args, metadata, None, False))
            self.assertEqual(metadata['verdict'], 'INCOMPLETE')
        args = argparse.Namespace(runner_mode='dispatched-six-angle', specialists_completed=6,
                                  consolidator_completed=True, adversarial_model=None, fallback_reason=None, verdict='PASS')
        metadata = {'runner_mode': args.runner_mode, 'verdict': 'PASS', 'fallback_reason': None}
        self.assertTrue(adaptive_review._writer_runner_state(args, metadata, None, False))
        self.assertEqual(metadata['verdict'], 'INCOMPLETE')

    def test_writer_rejects_unsupported_verdict(self):
        for verdict in ('BANANA', 'INCOMPLETE', ''):
            args = argparse.Namespace(runner_mode='adversarial-cli', specialists_completed=0,
                                      consolidator_completed=True, adversarial_model='codex',
                                      fallback_reason='native review unavailable', verdict=verdict)
            metadata = {'runner_mode': args.runner_mode, 'verdict': verdict, 'fallback_reason': args.fallback_reason}
            self.assertTrue(adaptive_review._writer_runner_state(args, metadata, None, False))
            self.assertEqual(metadata['verdict'], 'INCOMPLETE')
            self.assertEqual(metadata['runner_mode'], 'incomplete')

    def test_cli_rejects_scope_replacement_during_review(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, scope_file = scope_fixture(root)
            prompt = root / 'prompt.txt'
            prompt.write_text('review')
            fake = root / 'reviewer'
            changed = {**scope, 'repair_plan_sha256': 'a' * 64}
            fake.write_text("#!/bin/sh\nprintf '%s' " + shlex.quote(json.dumps(changed)) + ' > ' + shlex.quote(str(scope_file)) +
                            "\nprintf '%s\\n' " + shlex.quote(json.dumps(review_packet(scope))) + "\n")
            fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
            result = subprocess.run([
                sys.executable, str(HERE / 'adaptive_review.py'), 'run-cli',
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(root / 'report'),
                '--host', 'claude', '--codex-command', str(fake), '--no-workflow-available', '--no-agent-dispatch-available',
            ], capture_output=True, text=True)
            self.assertEqual(result.returncode, 2, result.stderr)
            self.assertEqual(json.loads(result.stdout)['verdict'], 'INCOMPLETE')

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
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
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
                      '--prompt-file', str(prompt), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                      '--base-commit', scope['base_commit'], '--review-output-path', str(root / 'report.md'),
                      '--host', 'claude', '--codex-command', str(fake),
                      '--no-workflow-available', '--no-agent-dispatch-available']
            for index, extra in enumerate((['--project-root', str(repo)],
                                           ['--project-root', str(root), '--scope-file', str(scope_file)])):
                invocation = common.copy()
                invocation[invocation.index('--review-output-path') + 1] = str(root / f'report-{index}.md')
                result = subprocess.run(invocation + extra, capture_output=True, text=True)
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
                    '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
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
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
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
                '--prompt-file', str(prompt), '--project-root', str(repo), '--source-root', str(repo), '--diff-path', scope['diff_path'],
                '--scope-file', str(scope_file), '--base-commit', scope['base_commit'], '--review-output-path', str(report), '--host', 'unknown',
                '--no-workflow-available', '--no-agent-dispatch-available',
            ], env=env, capture_output=True, text=True, check=False)
            self.assertEqual(proc.returncode, 2)
            self.assertIn('runner_mode=incomplete', report.read_text())


if __name__ == '__main__':
    unittest.main()
