"""The first pair needs the review actually delivered, even for a zero-write escalation."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from observe import CONSOLIDATOR, REVIEW, TRIAGE, WORKFLOW
from test_packet import write_test_packet
from verify import read_children

HERE = Path(__file__).resolve().parent


class Intake(unittest.TestCase):
    def test_verified_policy_loads_and_staged_packet_alone_cannot_authorize_write(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            repo, scope, report, report_hash = write_test_packet(root)
            pair = root / 'pair'
            pair.mkdir()
            subprocess.run([sys.executable, HERE / 'prepare.py', pair, '--retained',
                            report, report_hash, repo, scope['candidate_commit']], check=True,
                           capture_output=True)
            manifest = json.loads((pair / 'preparation.json').read_bytes())
            for name in manifest['cases']:
                case = pair / name
                for policy in [WORKFLOW, TRIAGE, REVIEW, CONSOLIDATOR]:
                    if policy == WORKFLOW:
                        private_policy = policy
                    else:
                        private_policy = 'project/.agents/skills/shark-rider/' + policy.removeprefix(
                            'catalog/shark-rider/')
                    subprocess.run([sys.executable, HERE / 'observe.py', case, 'read', private_policy],
                                   check=True, capture_output=True)
                result = subprocess.run(['./shark', 'create', 'task', 'E01-F01', 'Example'],
                                        cwd=case / 'project', capture_output=True)
                with self.subTest(case=name):
                    self.assertNotEqual(result.returncode, 0, 'forwarded without a review intake')
                    self.assertIn(b'review intake', result.stderr)
                    self.assertEqual(read_children(case / 'project/shark-tasks.db',
                                                   manifest['bindings'][name]), [])

    def test_preparation_can_freeze_a_synthetic_review_from_applied_history(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            setup = subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic'],
                                   capture_output=True, text=True)
            self.assertEqual(setup.returncode, 0, setup.stderr)
            prepared = json.loads((root / 'preparation.json').read_bytes())
            pin = prepared['review_intake']
            self.assertEqual(pin['history']['entity'], 'B001')
            self.assertEqual(len(pin['history']['operation_digest']), 64)
            for name in prepared['cases']:
                case = root / name
                received = subprocess.run([sys.executable, HERE / 'observe.py', case, 'intake',
                                           case / 'packet', case / 'history.json'],
                                          check=True, capture_output=True)
                payload = json.loads(received.stdout)
                self.assertIn('synthetic', payload['review.md'].lower())
                self.assertIn('E01-F01', payload['review.md'])
                entry = json.loads((case / 'observations/tools.jsonl').read_text().splitlines()[-1])
                self.assertEqual(entry['intake'], pin)

    def test_late_retry_cannot_reuse_prior_intake_after_refusal(self):
        from intake import received_pin
        pin = {'bundle_sha256': 'original', 'history': {'entity': 'B001'}}
        events = [dict(run_id='one', tool='intake', invocation='first', event='entry'),
                  dict(run_id='one', tool='intake', invocation='first', event='result',
                       outcome='delivered', intake=pin),
                  dict(run_id='one', tool='intake', invocation='second', event='entry'),
                  dict(run_id='one', tool='intake', invocation='second', event='result', outcome='refused')]
        self.assertFalse(received_pin(events, 'one', pin))

    def test_offline_verifier_checks_review_again_after_intake(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic'],
                           check=True, capture_output=True)
            case = root / 'grant-b'
            subprocess.run([sys.executable, HERE / 'observe.py', case, 'intake',
                            case / 'packet', case / 'history.json'], check=True, capture_output=True)
            (case / 'packet/provenance.json').write_text('{}')
            checked = subprocess.run([sys.executable, HERE / 'verify.py', root,
                                      'docs/feature-a.md#DOCS#worked-example'], capture_output=True)
            self.assertIn('intake member differs: provenance.json',
                          json.loads(checked.stdout)['grant-b']['missing'])

    def test_zero_call_receipt_includes_generation_outside_the_writer(self):
        from intake import check_origin
        from synthetic import DISCLOSURE
        from hashlib import sha256
        with tempfile.TemporaryDirectory() as tmp:
            project = Path(tmp)
            inputs = project / 'fixture-inputs'
            inputs.mkdir()
            writer_trace = b'1 execve("/usr/bin/python3", ["python3", "write-report"], []) = 0\n'
            (inputs / 'writer.trace').write_bytes(writer_trace)
            # A setup call discarded before write-report must still invalidate zero-spend evidence.
            call = b'2 execve("/bin/claude", ["claude"], []) = 0\n'
            (inputs / 'setup.trace').write_bytes(call)
            provenance = dict(input_origin=DISCLOSURE, writer_trace_sha256=sha256(writer_trace).hexdigest(),
                              setup_trace_sha256=sha256(call).hexdigest())
            result = dict(evidence=[dict(kind='fixture_origin', summary=DISCLOSURE)])
            report = ('**Verified effective model:** ' + chr(96) + 'unknown' + chr(96) +
                      '\n\n---\n\n' + DISCLOSURE + ' The runner count represents six supplied angle records.')
            with self.assertRaisesRegex(ValueError, 'reviewer or provider call in setup'):
                check_origin({'review.md': report, 'provenance.json': json.dumps(provenance),
                              'review.md.decisions.json': '{"findings":[]}'}, result, project)
