import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import sys
import unittest

SCRIPT = Path(__file__).with_name('get_diff.sh').resolve()


class CaptureScopeTest(unittest.TestCase):
    def test_scope_binds_baseline_candidate_and_refuses_dirty_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', *args], cwd=root, check=True,
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
            self.assertEqual(scope['base_commit'], base)
            self.assertEqual(scope['candidate_tree'], git('rev-parse', 'HEAD^{tree}'))
            self.assertEqual(scope['changed_files'], ['change.txt'])
            self.assertEqual(scope['diff_sha256'], hashlib.sha256(Path(scope['diff_path']).read_bytes()).hexdigest())
            with tempfile.TemporaryDirectory() as evidence_dir:
                evidence = Path(evidence_dir)
                scope_path = evidence / 'scope.json'
                scope_path.write_text(result.stdout)
                body_path = evidence / 'body.md'
                body_path.write_text('Reviewed all files.')
                report = evidence / 'report.md'
                writer = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'write-report',
                          '--body-file', str(body_path), '--review-output-path', str(report),
                          '--runner-mode', 'dispatched-six-angle', '--specialists-completed', '6',
                          '--consolidator-completed', 'true', '--base-commit', base,
                          '--diff-path', scope['diff_path'], '--scope-file', str(scope_path), '--verdict', 'PASS']
                unbound = writer.copy()
                scope_index = unbound.index('--scope-file')
                del unbound[scope_index:scope_index + 2]
                missing = subprocess.run(unbound, capture_output=True, text=True)
                self.assertEqual(missing.returncode, 2)
                self.assertEqual(json.loads(missing.stdout)['verdict'], 'INCOMPLETE')
                valid = subprocess.run(writer, capture_output=True, text=True)
                self.assertEqual(valid.returncode, 0, valid.stderr)
                self.assertIn(scope['candidate_tree'], report.read_text())
                partial = writer.copy()
                partial[partial.index('6')] = '5'
                incomplete = subprocess.run(partial, capture_output=True, text=True)
                self.assertEqual(incomplete.returncode, 2)
                self.assertEqual(json.loads(incomplete.stdout)['verdict'], 'INCOMPLETE')
                original_diff = Path(scope['diff_path']).read_bytes()
                Path(scope['diff_path']).write_bytes(original_diff + b'tampered')
                tampered = subprocess.run(writer, capture_output=True, text=True)
                self.assertEqual(tampered.returncode, 2)
                self.assertEqual(json.loads(tampered.stdout)['verdict'], 'INCOMPLETE')
                Path(scope['diff_path']).write_bytes(original_diff)
                (root / 'new.txt').write_text('untracked\n')
                stale = subprocess.run(writer, capture_output=True, text=True)
                self.assertEqual(stale.returncode, 2)
                self.assertEqual(json.loads(stale.stdout)['verdict'], 'INCOMPLETE')
            dirty = subprocess.run(['bash', str(SCRIPT)], cwd=root, text=True, capture_output=True)
            self.assertNotEqual(dirty.returncode, 0)
            self.assertIn('not clean', dirty.stderr)

    def test_snapshot_preserves_unrelated_work_and_index(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'source'
            root.mkdir()
            def git(*args):
                return subprocess.run(['git', '-c', 'commit.gpgsign=false', *args], cwd=root, check=True, text=True, capture_output=True).stdout.strip()
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
            (root / 'changed').write_text('final')
            (root / 'new').write_text('new candidate')
            (root / 'unrelated').write_text('preserve me')
            (root / 'scratch').write_text('unrelated scratch')
            before = git('status', '--porcelain')
            staged = git('diff', '--cached', '--binary')
            destination = Path(tmp) / 'snapshot'
            result = subprocess.run([sys.executable, str(SCRIPT.with_name('prepare_candidate.py')),
                                     '--project-root', str(root), '--destination', str(destination),
                                     '--metadata-file', str(Path(tmp) / 'snapshot.json'), 'changed', 'new'], capture_output=True, text=True, check=True)
            self.assertEqual(git('status', '--porcelain'), before)
            self.assertEqual(git('diff', '--cached', '--binary'), staged)
            self.assertEqual((root / 'unrelated').read_text(), 'preserve me')
            self.assertEqual((destination / 'changed').read_text(), 'final')
            self.assertEqual((destination / 'new').read_text(), 'new candidate')
            self.assertEqual((destination / 'unrelated').read_text(), 'old')
            self.assertFalse((destination / 'scratch').exists())
            scope = subprocess.run(['bash', str(SCRIPT)], cwd=destination, check=True,
                                   capture_output=True, text=True, env={**os.environ, 'REVIEW_BASE': base, 'REVIEW_SOURCE_METADATA': str(Path(tmp) / 'snapshot.json')})
            self.assertEqual(json.loads(scope.stdout)['changed_files'], ['changed', 'new'])
            self.assertEqual(json.loads(result.stdout)['candidate_tree'], json.loads(scope.stdout)['candidate_tree'])
            evidence = Path(tmp) / 'evidence'
            evidence.mkdir()
            scope_file = evidence / 'scope.json'
            scope_file.write_text(scope.stdout)
            body = evidence / 'body.md'
            body.write_text('VERDICT: PASS')
            scope_data = json.loads(scope.stdout)
            writer = [sys.executable, str(SCRIPT.with_name('adaptive_review.py')), 'write-report',
                      '--body-file', str(body), '--review-output-path', str(evidence / 'report.md'),
                      '--runner-mode', 'dispatched-six-angle', '--specialists-completed', '6',
                      '--consolidator-completed', 'true', '--base-commit', base,
                      '--diff-path', scope_data['diff_path'], '--scope-file', str(scope_file), '--verdict', 'PASS']
            self.assertEqual(subprocess.run(writer, capture_output=True).returncode, 0)
            (root / 'changed').write_text('newer change')
            stale = subprocess.run(writer, capture_output=True, text=True)
            self.assertEqual(stale.returncode, 2)
            self.assertEqual(json.loads(stale.stdout)['verdict'], 'INCOMPLETE')


if __name__ == '__main__':
    unittest.main()
