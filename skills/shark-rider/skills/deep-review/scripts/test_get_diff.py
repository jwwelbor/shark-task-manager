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
                return subprocess.run(['git', *args], cwd=root, check=True,
                                      text=True, capture_output=True).stdout.strip()
            git('init', '-b', 'main')
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


if __name__ == '__main__':
    unittest.main()
