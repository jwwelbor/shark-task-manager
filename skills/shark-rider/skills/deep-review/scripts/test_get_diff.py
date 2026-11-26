import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import sys
import unittest
import uuid

SCRIPT = Path(__file__).with_name('get_diff.sh').resolve()


def run_writer(command, **kwargs):
    command = command.copy()
    index = command.index('--review-output-path') + 1
    path = Path(command[index])
    if path.exists():
        command[index] = str(path.with_name(path.name + '.' + uuid.uuid4().hex))
    return subprocess.run(command, **kwargs)


class CaptureScopeTest(unittest.TestCase):
    @unittest.skipUnless(os.name == 'posix', 'requires byte-valued Git paths')
    def test_scope_path_bytes_survive_locale_changes_and_snapshot_reconstruction(self):
        # R40 Capture: each selected input must keep its identity across processes.
        names = [b'caf\xc3\xa9.txt', b'caf\xe9.txt', b'caf\xef\xbf\xbd.txt',
                 br'caf\udce9.txt', 'caf\U000100e9.txt'.encode('utf-8')]
        wire_names = [name.decode('utf-8', 'surrogateescape') for name in names]
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'repo'
            root.mkdir()

            def git(*args, cwd=root):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false',
                                       '-c', 'core.hooksPath=/dev/null', *args],
                                      cwd=cwd, capture_output=True, check=True).stdout

            git('init', '-b', 'main')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            git('commit', '--allow-empty', '-m', 'base')
            base = git('rev-parse', 'HEAD').decode().strip()
            for index, name in enumerate(names):
                (root / os.fsdecode(name)).write_bytes(bytes([index + 1]))
            git('add', '-A')
            git('commit', '-m', 'distinct paths')
            expected = git('diff', '--name-only', '-z', base, 'HEAD').split(b'\0')[:-1]
            self.assertCountEqual(expected, names)
            env = dict(os.environ, REVIEW_BASE=base, LC_ALL='C',
                       PYTHONCOERCECLOCALE='0', PYTHONUTF8='0')
            scope_file = Path(tmp) / 'scope.json'

            def capture(cwd, environment):
                result = subprocess.run(['bash', str(SCRIPT)], cwd=cwd, env=environment,
                                        check=True, capture_output=True)
                return json.loads(result.stdout)

            def validate(scope, cwd, environment):
                scope_file.write_text(json.dumps(scope), encoding='utf-8')
                return subprocess.run([sys.executable, str(SCRIPT.with_name('adaptive_review.py')),
                                       'validate-scope', '--project-root', str(cwd),
                                       '--source-root', str(root), '--scope-file', str(scope_file)],
                                      env=environment, capture_output=True)

            for utf8_mode, reader_mode in [('1', '0'), ('0', '1')]:
                with self.subTest(capture_utf8=utf8_mode):
                    scope = capture(root, dict(env, PYTHONUTF8=utf8_mode))
                    self.assertEqual(scope['changed_files'],
                                     [name.decode('utf-8', 'surrogateescape') for name in expected])
                    self.assertEqual(scope['changed_file_count'], len(names))
                    result = validate(scope, root, dict(env, PYTHONUTF8=reader_mode))
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

            # Keep both spellings dirty. Select one and verify Git captured exactly its bytes.
            for index in (0, 1):
                (root / os.fsdecode(names[index])).write_bytes(b'edited-' + bytes([index]))
            for index in (0, 1):
                with self.subTest(selected=names[index]):
                    snapshot = Path(tmp) / f'snapshot-{index}'
                    metadata = Path(tmp) / f'metadata-{index}.json'
                    subprocess.run([sys.executable, str(SCRIPT.with_name('prepare_candidate.py')),
                                    '--project-root', str(root), '--destination', str(snapshot),
                                    '--metadata-file', str(metadata), names[index]],
                                   capture_output=True, check=True, env=env)
                    source = json.loads(metadata.read_bytes())
                    self.assertEqual(source['selected_paths'], [wire_names[index]])
                    marker = Path(git('rev-parse', '--absolute-git-dir', cwd=snapshot)
                                  .decode().strip()) / 'shark-review-source.json'
                    self.assertEqual(json.loads(marker.read_bytes()), source)
                    self.assertEqual(git('show', b'HEAD:' + names[index], cwd=snapshot),
                                     b'edited-' + bytes([index]))
                    self.assertEqual(git('diff', '--name-only', '-z', 'HEAD^', 'HEAD', cwd=snapshot),
                                     names[index] + b'\0')
                    scope = capture(snapshot, dict(env, REVIEW_SOURCE_METADATA=str(metadata)))
                    for mode in ('1', '0'):
                        result = validate(scope, snapshot, dict(env, PYTHONUTF8=mode))
                        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

                    wrong = dict(source, selected_paths=[wire_names[1 - index]])
                    for swap_metadata, swap_marker in ((True, False), (False, True), (True, True)):
                        challenged = dict(scope, source_snapshot=wrong if swap_metadata else source)
                        marker.write_text(json.dumps(wrong if swap_marker else source), encoding='utf-8')
                        result = validate(challenged, snapshot, env)
                        self.assertNotEqual(result.returncode, 0)
                        diagnostic = (b'source candidate changed' if swap_metadata and swap_marker
                                      else b'marker does not match')
                        self.assertIn(diagnostic, result.stdout + result.stderr)
                    marker.write_text(json.dumps(source), encoding='utf-8')
                    result = validate(scope, snapshot, env)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_successor_capture_keeps_baseline_scope_and_restored_policy_delta(self):
        # R40 Capture: removing a policy from the current manifest still changes its caller.
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp) / 'repo'
            repo.mkdir()

            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false',
                                       '-c', 'core.hooksPath=/dev/null', *args],
                                      cwd=repo, check=True, capture_output=True).stdout

            def capture(predecessor=None):
                env = dict(os.environ, REVIEW_BASE=base, REVIEW_PREDECESSOR_COMMIT=predecessor or '')
                return json.loads(subprocess.run(['bash', str(SCRIPT)], cwd=repo, env=env,
                                                 capture_output=True, check=True).stdout)

            def check_manifest(scope, start, end):
                expected = git('diff', '--name-only', '-z', start, end).split(b'\0')[:-1]
                self.assertEqual(scope['changed_files'], [p.decode() for p in expected])
                patch = git('diff', '--no-ext-diff', '--binary', start, end)
                self.assertEqual(scope['diff_sha256'], hashlib.sha256(patch).hexdigest())

            git('init', '-b', 'main')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            for name in ('wrapper', 'policy', 'leaf'):
                (repo / name).write_text('base')
            git('add', '.')
            git('commit', '-m', 'base')
            base = git('rev-parse', 'HEAD').decode().strip()
            for name in ('wrapper', 'policy', 'leaf'):
                (repo / name).write_text('A-' + name)
            git('commit', '-am', 'first review')
            first = git('rev-parse', 'HEAD').decode().strip()
            round_a = capture()
            check_manifest(round_a, base, first)
            self.assertNotIn('successor_delta', round_a)

            git('restore', '--source', base, '--', 'policy')
            git('commit', '-am', 'restore policy')
            scope = capture(first)
            self.assertEqual(scope['base_commit'], base)
            self.assertEqual(scope['changed_files'], ['leaf', 'wrapper'])
            check_manifest(scope, base, 'HEAD')
            self.assertIn('successor_delta', scope)
            delta = scope['successor_delta']
            self.assertEqual(delta['predecessor_commit'], first)
            self.assertEqual(delta['predecessor_tree'], git('rev-parse', first + '^{tree}').decode().strip())
            self.assertEqual(delta['changed_files'], ['policy'])
            check_manifest(delta, first, 'HEAD')

            scope_file = Path(tmp) / 'scope.json'
            command = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope',
                       '--project-root', str(repo), '--source-root', str(repo), '--scope-file', str(scope_file)]
            for key, replacement in [('changed_files', []), ('predecessor_commit', base),
                                     ('predecessor_tree', scope['candidate_tree']), ('diff_sha256', '0' * 64)]:
                with self.subTest(field=key):
                    scope_file.write_text(json.dumps(dict(scope, successor_delta=dict(delta, **{key: replacement}))))
                    result = subprocess.run(command, capture_output=True)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn(b'successor delta', result.stdout)
            scope_file.write_text(json.dumps(scope))
            result = subprocess.run(command, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

            # Another branch has a valid object, but cannot supply this successor's delta.
            git('checkout', '-b', 'unrelated', base)
            git('commit', '--allow-empty', '-m', 'sibling')
            sibling = git('rev-parse', 'HEAD').decode().strip()
            git('checkout', 'main')
            for unavailable in [sibling, 'f' * 40]:
                full = capture(unavailable)
                self.assertNotIn('successor_delta', full)
                check_manifest(full, base, 'HEAD')

    def test_registered_runtime_worktree_is_excluded_but_other_files_are_not(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'repo'
            root.mkdir()
            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', '-C', str(root), *args], check=True, capture_output=True, text=True).stdout.strip()
            git('init', '-b', 'main')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            (root / 'file').write_text('base')
            git('add', 'file')
            git('commit', '-m', 'base')
            base = git('rev-parse', 'HEAD')
            (root / 'file').write_text('candidate')
            git('commit', '-am', 'candidate')
            worktree = root / '.shark-run-worktrees' / 'B001'
            git('worktree', 'add', '-b', 'shark-run-B001-test', str(worktree))
            result = subprocess.run(['bash', str(SCRIPT)], cwd=root, env={**os.environ, 'REVIEW_BASE': base}, capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            (root / '.shark-run-worktrees' / 'unrelated.txt').write_text('do not hide')
            result = subprocess.run(['bash', str(SCRIPT)], cwd=root, env={**os.environ, 'REVIEW_BASE': base}, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)

    def test_scope_rejects_git_flags_that_hide_source_edits(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.run(['git', '-C', str(root), *args], check=True, capture_output=True, text=True).stdout.strip()
            git('init', '-b', 'main')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            git('config', 'commit.gpgsign', 'false')
            (root / 'file').write_text('base')
            git('add', 'file')
            git('commit', '-m', 'base')
            base = git('rev-parse', 'HEAD')
            (root / 'file').write_text('candidate')
            git('commit', '-am', 'candidate')
            for flag in ('assume-unchanged', 'skip-worktree'):
                with self.subTest(flag=flag):
                    git('update-index', '--' + flag, '--', 'file')
                    (root / 'file').write_text('hidden edit')
                    result = subprocess.run(['bash', str(SCRIPT)], cwd=root, env={**os.environ, 'REVIEW_BASE': base}, capture_output=True, text=True)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('assume-unchanged or skip-worktree', result.stderr)
                    git('update-index', '--no-' + flag, '--', 'file')
                    (root / 'file').write_text('candidate')

    def test_scope_rejects_untrusted_root_before_git(self):
        with tempfile.TemporaryDirectory() as tmp:
            trusted = Path(tmp) / 'trusted'
            untrusted = Path(tmp) / 'untrusted'
            trusted.mkdir()
            untrusted.mkdir()
            subprocess.run(['git', 'init', str(untrusted)], check=True, capture_output=True)
            sentinel = Path(tmp) / 'executed'
            hook = Path(tmp) / 'monitor'
            hook.write_text('#!/bin/sh\ntouch "' + str(sentinel) + '"\n')
            hook.chmod(0o755)
            subprocess.run(['git', '-C', str(untrusted), 'config', 'core.fsmonitor', str(hook)], check=True)
            scope = Path(tmp) / 'scope.json'
            scope.write_text(json.dumps({'project_root': str(untrusted), 'base_commit': '0' * 40, 'diff_path': str(Path(tmp) / 'diff')}))
            result = subprocess.run([sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope', '--project-root', str(trusted), '--source-root', str(trusted), '--scope-file', str(scope)], capture_output=True, text=True, timeout=5)
            self.assertEqual(result.returncode, 2)
            self.assertIn('root does not match', result.stdout)
            self.assertFalse(sentinel.exists())

    @unittest.skipUnless(hasattr(os, 'mkfifo'), 'requires FIFO support')
    def test_scope_rejects_fifo_diff_without_blocking(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'repo'
            root.mkdir()
            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', '-C', str(root), *args], check=True, capture_output=True, text=True).stdout.strip()
            git('init', '-b', 'main')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            (root / 'file').write_text('base')
            git('add', 'file')
            git('commit', '-m', 'base')
            base = git('rev-parse', 'HEAD')
            (root / 'file').write_text('candidate')
            git('commit', '-am', 'candidate')
            capture = subprocess.run(['bash', str(SCRIPT)], cwd=root, env={**os.environ, 'REVIEW_BASE': base}, check=True, capture_output=True, text=True)
            scope = json.loads(capture.stdout)
            fifo = Path(tmp) / 'fifo'
            os.mkfifo(fifo)
            scope['diff_path'] = str(fifo)
            scope_path = Path(tmp) / 'scope.json'
            scope_path.write_text(json.dumps(scope))
            result = subprocess.run([sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope', '--project-root', str(root), '--source-root', str(root), '--scope-file', str(scope_path)], capture_output=True, timeout=5)
            self.assertEqual(result.returncode, 2)

    def test_explicit_baseline_works_without_main_or_master(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], cwd=root, check=True, capture_output=True, text=True).stdout.strip()
            git('init', '-b', 'trunk')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            (root / 'file').write_text('base')
            git('add', 'file')
            git('commit', '-m', 'base')
            base = git('rev-parse', 'HEAD')
            (root / 'file').write_text('candidate')
            git('commit', '-am', 'candidate')
            result = subprocess.run(['bash', str(SCRIPT)], cwd=root, env={**os.environ, 'REVIEW_BASE': base}, check=True, capture_output=True, text=True)
            self.assertEqual(json.loads(result.stdout)['base_commit'], base)

    def test_scope_binds_baseline_candidate_and_refuses_dirty_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], cwd=root, check=True,
                                      text=True, capture_output=True).stdout.strip()
            git('init', '-b', 'main')
            git('config', 'commit.gpgsign', 'false')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            (root / 'change.txt').write_text('base\n')
            git('add', 'change.txt')
            git('commit', '-m', 'base')
            base = git('rev-parse', 'HEAD')
            git('checkout', '-b', 'repair')
            (root / 'change.txt').write_text('fixed\n')
            git('commit', '-am', 'repair')
            result = subprocess.run(['bash', str(SCRIPT)], cwd=root, check=True,
                                    text=True, capture_output=True,
                                    env={**os.environ, 'REVIEW_BASE': base})
            scope = json.loads(result.stdout)
            second = subprocess.run(['bash', str(SCRIPT)], cwd=root, check=True,
                                    text=True, capture_output=True,
                                    env={**os.environ, 'REVIEW_BASE': base})
            self.assertNotEqual(scope['review_output_path'], json.loads(second.stdout)['review_output_path'])
            self.assertEqual(scope['base_commit'], base)
            self.assertEqual(scope['candidate_tree'], git('rev-parse', 'HEAD^{tree}'))
            self.assertEqual(scope['changed_files'], ['change.txt'])
            self.assertEqual(scope['diff_sha256'], hashlib.sha256(Path(scope['diff_path']).read_bytes()).hexdigest())
            with tempfile.TemporaryDirectory() as evidence_dir:
                evidence = Path(evidence_dir)
                scope_path = evidence / 'scope.json'
                scope_path.write_text(result.stdout)
                sentinel = evidence / 'fsmonitor-executed'
                hook = evidence / 'fsmonitor'
                hook.write_text('#!/bin/sh\ntouch "' + str(sentinel) + '"\n')
                hook.chmod(0o755)
                git('config', 'core.fsmonitor', str(hook))
                validate = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope', '--project-root', str(root), '--source-root', str(root), '--scope-file', str(scope_path)]
                self.assertEqual(subprocess.run(validate, capture_output=True).returncode, 0)
                self.assertFalse(sentinel.exists())
                git('config', '--unset', 'core.fsmonitor')
                body_path = evidence / 'body.md'
                body_path.write_text('Reviewed scope: all candidate files. Findings: none. Detailed checks confirmed complete coverage. VERDICT: PASS')
                report = evidence / 'report.md'
                angles = evidence / 'angles.json'
                identity = {'commit': scope['candidate_commit'], 'tree': scope['candidate_tree'], 'diff_sha256': scope['diff_sha256']}
                angles.write_text(json.dumps({name: {'candidate_identity': identity, 'reviewed_files': scope['changed_files'], 'findings': []} for name in 'abcdef'}))
                decisions = evidence / 'decisions.json'
                decisions.write_text(json.dumps({'candidate_identity': identity, 'findings': []}))
                writer = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'write-report',
                          '--project-root', str(root), '--source-root', str(root), '--body-file', str(body_path), '--review-output-path', str(report),
                          '--runner-mode', 'dispatched-six-angle', '--specialists-completed', '6',
                          '--consolidator-completed', 'true', '--fallback-reason', 'native Workflow unavailable', '--base-commit', base,
                          '--diff-path', scope['diff_path'], '--scope-file', str(scope_path), '--angle-results', str(angles), '--review-decisions', str(decisions), '--verdict', 'PASS']
                no_angles = writer.copy()
                index = no_angles.index('--angle-results')
                del no_angles[index:index + 2]
                self.assertEqual(run_writer(no_angles, capture_output=True).returncode, 2)
                original_angles = angles.read_text()
                partial_angles = json.loads(original_angles)
                partial_angles['b']['reviewed_files'] = []
                angles.write_text(json.dumps(partial_angles))
                self.assertEqual(run_writer(writer, capture_output=True).returncode, 2)
                angles.write_text(original_angles)
                unbound = writer.copy()
                scope_index = unbound.index('--scope-file')
                del unbound[scope_index:scope_index + 2]
                missing = run_writer(unbound, capture_output=True, text=True)
                self.assertEqual(missing.returncode, 2)
                self.assertEqual(json.loads(missing.stdout)['verdict'], 'INCOMPLETE')
                for invalid_body in ('', 'VERDICT: PASS', 'Reviewed scope: all. Findings: blockers remain. VERDICT: FAIL',
                                     'Reviewed scope: all files. Findings: incomplete coverage. VERDICT: PASS',
                                     'Reviewed scope: all files and callers carefully. Findings: blockers remain and are not resolved. VERDICT: PASS',
                                     'Reviewed scope: all files and callers carefully. Findings: file:1 blocker is NOT REFUTED. VERDICT: PASS'):
                    body_path.write_text(invalid_body)
                    invalid = run_writer(writer, capture_output=True, text=True)
                    self.assertEqual(invalid.returncode, 2, invalid_body)
                    self.assertEqual(json.loads(invalid.stdout)['verdict'], 'INCOMPLETE')
                body_path.write_text('Reviewed scope: all candidate files. Findings: none. Detailed checks confirmed complete coverage. VERDICT: PASS')
                valid = run_writer(writer, capture_output=True, text=True)
                self.assertEqual(valid.returncode, 0, valid.stderr)
                report = Path(json.loads(valid.stdout)['report_path'])
                self.assertIn(scope['candidate_tree'], report.read_text())
                report_bytes = report.read_bytes()
                collision = writer.copy()
                collision[collision.index('--review-output-path') + 1] = str(report)
                self.assertNotEqual(subprocess.run(collision, capture_output=True).returncode, 0)
                self.assertEqual(report.read_bytes(), report_bytes)
                link = evidence / 'report-link'
                link.symlink_to(report)
                collision[collision.index('--review-output-path') + 1] = str(link)
                self.assertNotEqual(subprocess.run(collision, capture_output=True).returncode, 0)
                self.assertEqual(report.read_bytes(), report_bytes)
                default_writer = writer.copy()
                default_writer[default_writer.index('--review-output-path') + 1] = scope['review_output_path']
                default = subprocess.run(default_writer, capture_output=True, text=True)
                self.assertEqual(default.returncode, 0, default.stderr)
                retained = json.loads(default.stdout)['retained_scope']
                validate = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope', '--project-root', str(root), '--source-root', str(root), '--scope-file', retained]
                self.assertEqual(subprocess.run(validate, capture_output=True).returncode, 0)
                self.assertEqual(git('status', '--porcelain'), '')
                failed_decisions = {'candidate_identity': identity, 'findings': [{'file': 'change.txt', 'line': 1, 'severity': 'blocker', 'rule': 'BUG', 'summary': 'broken', 'diagnosis': 'wrong', 'disposition': 'open', 'reason': 'still reproducible'}]}
                body_path.write_text('Reviewed scope: all candidate files and callers. Findings: a blocking bug remains and requires repair. VERDICT: FAIL')
                failed_writer = writer.copy()
                failed_writer[failed_writer.index('--verdict') + 1] = 'FAIL'
                failed_angles = json.loads(original_angles)
                failed_angles['a']['findings'] = [{k: v for k, v in failed_decisions['findings'][0].items() if k not in ('disposition', 'reason')}]
                angles.write_text(json.dumps(failed_angles))
                decisions.write_text(json.dumps({'candidate_identity': identity, 'findings': []}))
                omitted = run_writer(failed_writer, capture_output=True, text=True)
                self.assertEqual(omitted.returncode, 2)
                self.assertEqual(json.loads(omitted.stdout)['verdict'], 'INCOMPLETE')
                decisions.write_text(json.dumps(failed_decisions))
                failed = run_writer(failed_writer, capture_output=True, text=True)
                self.assertEqual(failed.returncode, 0, failed.stderr)
                self.assertTrue(Path(json.loads(failed.stdout)['retained_decisions']).is_file())
                angles.write_text(original_angles)
                body_path.write_text('Reviewed scope: all candidate files and callers. Findings: none. Detailed checks confirmed complete coverage. VERDICT: PASS')
                contradictory_fail = run_writer(failed_writer, capture_output=True, text=True)
                self.assertEqual(contradictory_fail.returncode, 2)
                self.assertEqual(json.loads(contradictory_fail.stdout)['verdict'], 'INCOMPLETE')
                decisions.write_text(json.dumps({'candidate_identity': identity, 'findings': []}))
                body_path.write_text('Reviewed scope: all candidate files. Findings: none. Detailed checks confirmed complete coverage. VERDICT: PASS')
                partial = writer.copy()
                partial[partial.index('6')] = '5'
                incomplete = run_writer(partial, capture_output=True, text=True)
                self.assertEqual(incomplete.returncode, 2)
                self.assertEqual(json.loads(incomplete.stdout)['verdict'], 'INCOMPLETE')
                partial[partial.index('5')] = 'unknown'
                packet_arg = partial.index('--angle-results')
                del partial[packet_arg:packet_arg + 2]
                unknown = run_writer(partial, capture_output=True, text=True)
                self.assertEqual(unknown.returncode, 2)
                self.assertEqual(json.loads(unknown.stdout)['specialists_completed'], 'unknown')
                original_diff = Path(scope['diff_path']).read_bytes()
                Path(scope['diff_path']).write_bytes(original_diff + b'tampered')
                tampered = run_writer(writer, capture_output=True, text=True)
                self.assertEqual(tampered.returncode, 2)
                self.assertEqual(json.loads(tampered.stdout)['verdict'], 'INCOMPLETE')
                Path(scope['diff_path']).write_bytes(original_diff)
                (root / 'new.txt').write_text('untracked\n')
                stale = run_writer(writer, capture_output=True, text=True)
                self.assertEqual(stale.returncode, 2)
                self.assertEqual(json.loads(stale.stdout)['verdict'], 'INCOMPLETE')
            dirty = subprocess.run(['bash', str(SCRIPT)], cwd=root, text=True, capture_output=True)
            self.assertNotEqual(dirty.returncode, 0)
            self.assertIn('not clean', dirty.stderr)

    def test_reworded_snapshot_in_second_worktree_cannot_replace_trusted_source(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'source'
            root.mkdir()
            def git(where, *args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', '-C', str(where), *args], check=True, capture_output=True, text=True).stdout.strip()
            git(root, 'init', '-b', 'main')
            git(root, 'config', 'user.name', 'Test')
            git(root, 'config', 'user.email', 'test@example.invalid')
            (root / 'file').write_text('base')
            git(root, 'add', 'file')
            git(root, 'commit', '-m', 'base')
            base = git(root, 'rev-parse', 'HEAD')
            (root / 'file').write_text('candidate')
            first = Path(tmp) / 'first'
            metadata = Path(tmp) / 'metadata.json'
            subprocess.run([sys.executable, str(SCRIPT.with_name('prepare_candidate.py')), '--project-root', str(root), '--destination', str(first), '--metadata-file', str(metadata), 'file'], check=True, capture_output=True)
            git(first, 'commit', '--amend', '-m', 'ordinary subject')
            amended = git(first, 'rev-parse', 'HEAD')
            second = Path(tmp) / 'second'
            git(root, 'worktree', 'add', '--detach', str(second), amended)
            capture = subprocess.run(['bash', str(SCRIPT)], cwd=second, env={**os.environ, 'REVIEW_BASE': base}, check=True, capture_output=True, text=True)
            scope = Path(tmp) / 'scope.json'
            scope.write_text(capture.stdout)
            result = subprocess.run([sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope', '--project-root', str(second), '--source-root', str(root), '--scope-file', str(scope)], capture_output=True, text=True, timeout=5)
            self.assertEqual(result.returncode, 2)
            self.assertIn('source root does not match trusted source', result.stdout)

    def test_snapshot_preserves_unrelated_work_and_index(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'source'
            root.mkdir()
            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', *args], cwd=root, check=True, text=True, capture_output=True).stdout.strip()
            git('init', '-b', 'main')
            git('config', 'commit.gpgsign', 'false')
            git('config', 'user.name', 'Test')
            git('config', 'user.email', 'test@example.invalid')
            (root / 'changed').write_text('old')
            (root / 'unrelated').write_text('old')
            git('add', 'changed', 'unrelated')
            git('commit', '-m', 'base')
            base = git('rev-parse', 'HEAD')
            (root / 'changed').write_text('staged')
            git('add', 'changed')
            (root / 'new').write_text('new candidate')
            (root / 'unrelated').write_text('preserve me')
            (root / 'scratch').write_text('unrelated scratch')
            # A staged-only version with the worktree restored to HEAD is
            # ambiguous and must not be silently discarded.
            (root / 'changed').write_text('old')
            conflict = subprocess.run([sys.executable, str(SCRIPT.with_name('prepare_candidate.py')),
                                       '--project-root', str(root), '--destination', str(Path(tmp) / 'rejected'),
                                       '--metadata-file', str(Path(tmp) / 'rejected.json'), 'changed'],
                                      capture_output=True, text=True)
            self.assertNotEqual(conflict.returncode, 0)
            self.assertIn('divergent staged and worktree', conflict.stderr)
            (root / 'changed').write_text('staged')
            before = git('status', '--porcelain')
            staged = git('diff', '--cached', '--binary')
            destination = Path(tmp) / 'snapshot'
            result = subprocess.run([sys.executable, str(SCRIPT.with_name('prepare_candidate.py')),
                                     '--project-root', str(root), '--destination', str(destination),
                                     '--metadata-file', str(Path(tmp) / 'snapshot.json'), 'changed', 'new'], capture_output=True, text=True, check=True)
            self.assertEqual(git('status', '--porcelain'), before)
            self.assertEqual(git('diff', '--cached', '--binary'), staged)
            self.assertEqual((root / 'unrelated').read_text(), 'preserve me')
            self.assertEqual((destination / 'changed').read_text(), 'staged')
            self.assertEqual((destination / 'new').read_text(), 'new candidate')
            self.assertEqual((destination / 'unrelated').read_text(), 'old')
            self.assertFalse((destination / 'scratch').exists())
            missing_source = subprocess.run(['bash', str(SCRIPT)], cwd=destination, capture_output=True, text=True, env={**os.environ, 'REVIEW_BASE': base})
            self.assertNotEqual(missing_source.returncode, 0)
            self.assertIn('requires REVIEW_SOURCE_METADATA', missing_source.stderr)
            scope = subprocess.run(['bash', str(SCRIPT)], cwd=destination, check=True,
                                   capture_output=True, text=True, env={**os.environ, 'REVIEW_BASE': base, 'REVIEW_SOURCE_METADATA': str(Path(tmp) / 'snapshot.json')})
            self.assertEqual(json.loads(scope.stdout)['changed_files'], ['changed', 'new'])
            self.assertEqual(json.loads(result.stdout)['candidate_tree'], json.loads(scope.stdout)['candidate_tree'])
            evidence = Path(tmp) / 'evidence'
            evidence.mkdir()
            scope_file = evidence / 'scope.json'
            scope_file.write_text(scope.stdout)
            body = evidence / 'body.md'
            body.write_text('Reviewed scope: all candidate files. Findings: none. Detailed checks confirmed complete coverage. VERDICT: PASS')
            scope_data = json.loads(scope.stdout)
            angles = evidence / 'angles.json'
            identity = {'commit': scope_data['candidate_commit'], 'tree': scope_data['candidate_tree'], 'diff_sha256': scope_data['diff_sha256']}
            angles.write_text(json.dumps({name: {'candidate_identity': identity, 'reviewed_files': scope_data['changed_files'], 'findings': []} for name in 'abcdef'}))
            decisions = evidence / 'decisions.json'
            decisions.write_text(json.dumps({'candidate_identity': identity, 'findings': []}))
            writer = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'write-report',
                      '--project-root', str(destination), '--source-root', str(root), '--body-file', str(body), '--review-output-path', str(evidence / 'report.md'),
                      '--runner-mode', 'dispatched-six-angle', '--specialists-completed', '6',
                      '--consolidator-completed', 'true', '--fallback-reason', 'native Workflow unavailable', '--base-commit', base,
                      '--diff-path', scope_data['diff_path'], '--scope-file', str(scope_file), '--angle-results', str(angles), '--review-decisions', str(decisions), '--verdict', 'PASS']
            accepted = run_writer(writer, capture_output=True, text=True)
            self.assertEqual(accepted.returncode, 0, accepted.stderr)
            retained_scope = json.loads(accepted.stdout)['retained_scope']
            validate = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope', '--project-root', str(destination), '--source-root', str(root), '--scope-file', retained_scope]
            self.assertEqual(subprocess.run(validate, capture_output=True).returncode, 0)
            (destination / 'untracked').write_text('changed snapshot')
            self.assertEqual(subprocess.run(validate, capture_output=True).returncode, 2)
            (destination / 'untracked').unlink()
            destination.rename(destination.with_name('snapshot-missing'))
            self.assertEqual(subprocess.run(validate, capture_output=True).returncode, 2)
            destination.with_name('snapshot-missing').rename(destination)
            default_writer = writer.copy()
            default_writer[default_writer.index('--review-output-path') + 1] = scope_data['review_output_path']
            default = subprocess.run(default_writer, capture_output=True, text=True)
            self.assertEqual(default.returncode, 0, default.stderr)
            default_validate = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'validate-scope', '--project-root', str(destination), '--source-root', str(root), '--scope-file', json.loads(default.stdout)['retained_scope']]
            self.assertEqual(subprocess.run(default_validate, capture_output=True).returncode, 0)
            (root / 'scratch').write_text('changed unrelated scratch')
            changed_scratch = subprocess.run(validate, capture_output=True)
            self.assertEqual(changed_scratch.returncode, 2)
            (root / 'scratch').write_text('unrelated scratch')
            (root / 'later').write_text('new source path')
            inventory_changed = run_writer(writer, capture_output=True, text=True)
            self.assertEqual(inventory_changed.returncode, 2)
            self.assertIn('inventory changed', inventory_changed.stdout)
            self.assertEqual(subprocess.run(validate, capture_output=True).returncode, 2)
            (root / 'later').unlink()
            git('reset', 'HEAD', '--', 'changed')
            index_changed = run_writer(writer, capture_output=True, text=True)
            self.assertEqual(index_changed.returncode, 2)
            self.assertTrue('source index changed' in index_changed.stdout or 'inventory changed' in index_changed.stdout)
            git('add', 'changed')
            no_source = dict(scope_data)
            no_source.pop('source_snapshot')
            scope_file.write_text(json.dumps(no_source))
            self.assertEqual(run_writer(writer, capture_output=True).returncode, 2)
            scope_file.write_text(scope.stdout)
            (root / 'changed').write_text('newer change')
            stale = run_writer(writer, capture_output=True, text=True)
            self.assertEqual(stale.returncode, 2)
            self.assertEqual(json.loads(stale.stdout)['verdict'], 'INCOMPLETE')
            subprocess.run(['git', '-C', str(destination), '-c', 'commit.gpgsign=false', '-c', 'core.hooksPath=/dev/null', 'commit', '--amend', '-m', 'rewritten snapshot'], check=True, capture_output=True)
            amended = subprocess.run(['bash', str(SCRIPT)], cwd=destination, capture_output=True, text=True, env={**os.environ, 'REVIEW_BASE': base})
            self.assertNotEqual(amended.returncode, 0)
            self.assertIn('requires REVIEW_SOURCE_METADATA', amended.stderr)


if __name__ == '__main__':
    unittest.main()
