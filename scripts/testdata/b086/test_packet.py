"""Exercise packet intake with the existing review writer and a disposable Git repo."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
SCRIPTS = HERE.parents[2] / 'skills/shark-rider/skills/deep-review/scripts'
sys.path.insert(0, str(SCRIPTS))
import adaptive_review as review
from test_adaptive_review import scope_fixture


class PacketIntake(unittest.TestCase):
    def test_only_the_original_scope_and_report_can_enter_replay(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo, scope, _ = scope_fixture(root)
            report = root / 'original.md'
            body = ('VERDICT: PASS-with-triage\nReviewed scope includes the source file. '
                    'Findings include an optional usage example for that feature.')
            decision = dict(file='source', line=1, rule='DOCS', severity='non-blocker',
                            summary='Missing usage example', diagnosis='Existing example omits the new option',
                            reason='Existing usage remains documented', disposition='triage')
            meta = dict(scope, generated='fixture', runner_mode='adversarial-cli',
                        specialists_completed=0, consolidator_completed=True, verdict='PASS-with-triage',
                        adversarial_model='test stub', _scope=scope,
                        _diff_bytes=Path(scope['diff_path']).read_bytes(),
                        _decisions={'candidate_identity': review._review_identity(scope),
                                    'findings': [decision]})
            review.persist_report(report, body, meta)
            report_hash = hashlib.sha256(report.read_bytes()).hexdigest()
            command = [sys.executable, str(HERE / 'packet.py'), str(report), report_hash,
                       str(repo), scope['candidate_commit']]
            result = subprocess.run(command, text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            receipt = json.loads(result.stdout)
            self.assertEqual(receipt['report_sha256'], report_hash)
            self.assertEqual(receipt['replay_candidate'], scope['candidate_commit'])
            for label, args in [('wrong report', [*command[:3], '0' * 64, *command[4:]]),
                                ('wrong candidate', [*command[:-1], scope['base_commit']])]:
                with self.subTest(label=label):
                    refusal = subprocess.run(args, capture_output=True, text=True)
                    self.assertNotEqual(refusal.returncode, 0)
                    self.assertEqual(refusal.stdout, '')
            Path(str(report) + '.diff').write_text('different diff')
            refused = subprocess.run(command, capture_output=True, text=True)
            self.assertNotEqual(refused.returncode, 0)
            self.assertIn('diff', refused.stderr)


if __name__ == '__main__':
    unittest.main()
