"""Runner 3 publication tests, using native-format recordings and separate processes."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from test_adaptive_review import HERE, fake_reviewer, review_packet, scope_fixture


class PublicationTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        repo, scope, scope_file = scope_fixture(self.root)
        self.packet = review_packet(scope)
        self.common = ['--project-root', repo, '--source-root', repo, '--diff-path', scope['diff_path'],
                       '--base-commit', scope['base_commit'], '--scope-file', scope_file]
        self.env = {**os.environ, 'PATH': str(self.root) + os.pathsep + os.environ['PATH']}
        codex = self.root / 'codex'
        codex.write_text('#!/bin/sh\nexit 7\n')
        codex.chmod(0o700)
        self.prompt = self.root / 'prompt'
        self.prompt.write_text('review fixture')
        self.published = 0

    def call(self, script, *args, expected=0):
        result = subprocess.run([sys.executable, HERE / script, *args], env=self.env,
                                capture_output=True, text=True)
        self.assertEqual(result.returncode, expected, result.stderr or result.stdout)
        return json.loads(result.stdout)

    def produce(self, name, host, request, tokens, cost, ending, direct=False):
        body = self.packet['report'] + ' café' + ending
        packet = {**self.packet, 'report': body}
        fake_reviewer(self.root / 'claude', {'session_id': name, 'model': 'recording-model',
                      'provider': 'anthropic', 'usage': {'input_tokens': tokens}, 'total_cost_usd': cost,
                      'result': json.dumps(packet)})
        args = [*self.common, '--host', host, '--prompt-file', self.prompt, '--require-verified-identity',
                '--no-workflow-available', '--no-agent-dispatch-available']
        if request is not None:
            args.extend(['--requested-model', request])
        if direct:
            result = self.call('adaptive_review.py', 'run-cli', *args, '--review-output-path', self.root / name)
        else:
            result = self.call('cli_review_job.py', 'produce', self.root / name, *args)
        reason = {'codex': 'Workflow and agent dispatch unavailable',
                  'unknown': 'host identity unknown; selected available alternate CLI',
                  'claude': 'Workflow and agent dispatch unavailable; codex: non-zero exit status 7; using claude CLI'}[host]
        self.assertEqual(result['fallback_reason'], reason)
        winner = json.loads(Path(result['retained_attempts']).read_bytes())[-1]
        self.assertEqual(winner.get('fallback_reason'), reason)
        self.assertEqual(winner['worker_id'], name)
        self.assertEqual(result['usage'], {'input_tokens': tokens})
        self.assertEqual(result['cost'], cost)
        self.assertEqual(result['effective_provider'], 'anthropic')
        self.assertEqual(result['effective_model'], 'recording-model')
        self.assertEqual(result['requested_model'], request)
        canonical = json.dumps(packet, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
        self.assertEqual(winner['packet_sha256'], hashlib.sha256(canonical).hexdigest())
        self.assertEqual(result['attempts_sha256'], hashlib.sha256(Path(result['retained_attempts']).read_bytes()).hexdigest())
        self.assertIn('requested_model', winner)
        self.assertEqual(winner['requested_model'], request)
        self.assertEqual(Path(result['retained_body']).read_bytes(), body.encode())
        self.assertIn('**Fallback reason:** ' + reason + '\n', Path(result['report_path']).read_text())
        return result

    def publish(self, job, producer, *args, expected=0):
        self.published += 1
        report = self.root / f'published-{self.published}.md'
        output = self.call('cli_review_job.py', 'publish', self.root / job, *self.common,
                           '--review-output-path', report, *args, expected=expected)
        header = report.read_text().split('\n---\n')[0]
        self.assertEqual(output['verdict'], 'PASS' if not expected else 'INCOMPLETE')
        self.assertIn('**Verdict:** ' + output['verdict'], header)
        if expected:
            for key in ('effective_model', 'effective_provider', 'usage', 'cost', 'retained_attempts', 'attempts'):
                self.assertIsNone(output.get(key))
            self.assertIn('**Verified effective route:** `unknown/unknown`', header)
            self.assertFalse(Path(str(report) + '.attempts.json').exists())
            self.assertIn(output['fallback_reason'], header)
            return output
        for field in ('attempts', 'usage', 'cost', 'effective_model', 'effective_provider', 'requested_model', 'fallback_reason'):
            self.assertEqual(output[field], producer[field])
        self.assertEqual(json.loads(Path(output['retained_attempts']).read_bytes()), producer['attempts'])
        self.assertIn('**Fallback reason:** ' + producer['fallback_reason'] + '\n', header)
        self.assertIn('**Requested model:** `' + (producer['requested_model'] or 'unknown') + '`', header)

    def test_producer_binds_exact_body_request_and_final_reason(self):
        self.produce('direct', 'claude', 'opus', 29, 0.07, ' \t\r\n\n', direct=True)

    def check_reason(self, job, producer):
        self.publish(job, producer)
        self.publish(job, producer, '--fallback-reason', producer['fallback_reason'])
        self.publish(job, producer, '--fallback-reason', 'preferred Codex succeeded', expected=2)
        self.publish(job, producer, '--fallback-reason', '', expected=2)

    def test_immediate_and_resumed_publications_preserve_request_and_reason(self):
        a = self.produce('A', 'codex', 'opus', 11, 0.02, ' \t\n\n')
        self.check_reason('A', a)
        b = self.produce('B', 'claude', None, 29, 0.07, '\n')
        self.check_reason('B', b)
        c = self.produce('C', 'unknown', '', 11, None, '\n')
        self.check_reason('C', c)
        # Each companion starts afresh. Fail visibly if publication invokes any reviewer.
        for cli in ('claude', 'codex'):
            (self.root / cli).write_text('#!/bin/sh\ntouch "$REVIEW_PROBE"\nexit 9\n')
        self.env['REVIEW_PROBE'] = str(self.root / 'unexpected-reviewer-call')
        for job, producer in [('C', c), ('B', b), ('A', a)]:
            self.check_reason(job, producer)
            self.publish(job, producer, '--fallback-reason', producer['fallback_reason'].upper(), expected=2)
            self.publish(job, producer, '--fallback-reason', producer['fallback_reason'] + ' ', expected=2)
            self.publish(job, producer, '--requested-model', 'recording-model', expected=2)
        self.publish('A', a, '--fallback-reason', c['fallback_reason'], expected=2)
        self.publish('A', a, '--requested-model', 'opus')
        self.publish('A', a, '--requested-model', 'haiku', expected=2)
        self.publish('C', c, '--requested-model', '')
        self.assertFalse(Path(self.env['REVIEW_PROBE']).exists())

    def test_publication_checks_the_selected_execution_even_for_identical_packets(self):
        a = self.produce('A', 'codex', 'opus', 11, 0.02, ' \t\n\n')
        d = self.produce('D', 'claude', 'opus', 29, 0.07, ' \t\n\n')
        self.publish('D', d)
        self.publish('A', a)
        path = Path(a['retained_attempts'])
        path.write_bytes(Path(d['retained_attempts']).read_bytes())
        self.publish('A', a, expected=2)

    def test_packet_checks_exact_body_and_complete_decisions_after_restart(self):
        a = self.produce('A', 'codex', 'opus', 11, 0.02, ' \t\n\n')
        b = self.produce('B', 'claude', 'opus', 29, 0.07, '\n')
        self.publish('B', b)
        self.publish('A', a)
        for field, other in [('retained_attempts', b), ('retained_body', b)]:
            path = Path(a[field])
            original = path.read_bytes()
            path.write_bytes(Path(other[field]).read_bytes())
            self.publish('A', a, expected=2)
            path.write_bytes(original)
        decisions = Path(a['retained_decisions'])
        changed = json.loads(decisions.read_bytes())
        changed['fixture'] = 'same valid scope, different decision object'
        decisions.write_text(json.dumps(changed))
        self.publish('A', a, expected=2)
        decisions.write_bytes(Path(b['retained_decisions']).read_bytes())
        body = Path(a['retained_body'])
        body.write_bytes(body.read_bytes()[:-1])
        self.publish('A', a, expected=2)
        body.unlink()
        self.publish('A', a, expected=2)

    def test_partial_producer_write_does_not_create_a_ready_job(self):
        job = self.root / 'interrupted'
        fake = self.root / 'claude'
        fake_reviewer(fake, {'session_id': 'interrupted-worker', 'provider': 'anthropic',
                             'model': 'recording-model', 'result': json.dumps(self.packet)})
        fake.write_text(fake.read_text().replace('#!/bin/sh\n', '#!/bin/sh\ntouch "$BODY_COLLISION"\n'))
        self.env['BODY_COLLISION'] = str(job / 'producer.md.body.txt')
        result = subprocess.run([sys.executable, HERE / 'cli_review_job.py', 'produce', job,
                                 *self.common, '--prompt-file', self.prompt, '--host', 'codex',
                                 '--no-workflow-available', '--no-agent-dispatch-available'],
                                env=self.env, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue((job / 'producer.md').exists())
        self.assertFalse((job / 'job.json').exists())
        self.publish('interrupted', {}, expected=2)

    def test_direct_writer_requires_the_selected_producer_digest(self):
        result = self.produce('A', 'codex', 'opus', 11, 0.02, '\n')
        for index, digest in enumerate((None, 'not-a-digest', '0' * 64)):
            destination = self.root / f'bad-reference-{index}.md'
            flags = [] if digest is None else ['--expected-attempts-sha256', digest]
            output = self.call('adaptive_review.py', 'write-report', *self.common,
                               '--review-output-path', destination, '--runner-mode', 'adversarial-cli',
                               '--adversarial-model', 'claude', '--specialists-completed', '0',
                               '--consolidator-completed', 'true', '--verdict', 'PASS',
                               '--body-file', result['retained_body'], '--review-decisions', result['retained_decisions'],
                               '--attempts-file', result['retained_attempts'], *flags, expected=2)
            self.assertEqual(output['verdict'], 'INCOMPLETE')
            self.assertIn('**Verdict:** INCOMPLETE', destination.read_text())
            self.assertNotIn('retained_attempts', output)
            self.assertIsNone(output['effective_model'])

    def test_legacy_receipts_and_missing_saved_digest_fail_without_attribution(self):
        producer = self.produce('A', 'codex', None, 11, None, '\n')
        reference = self.root / 'A/job.json'
        saved = json.loads(reference.read_bytes())
        receipt_path = Path(producer['retained_attempts'])
        # Each altered producer output has its own independently supplied execution checksum.
        for key, invalid in [('packet_sha256', ['absent']), ('requested_model', ['absent']),
                             ('fallback_reason', ['absent', None, '  ', 'unknown'])]:
            for value in invalid:
                receipts = json.loads(json.dumps(producer['attempts']))
                if value == 'absent':
                    del receipts[-1][key]
                else:
                    receipts[-1][key] = value
                raw = json.dumps(receipts).encode()
                receipt_path.write_bytes(raw)
                reference.write_text(json.dumps({**saved, 'attempts_sha256': hashlib.sha256(raw).hexdigest()}))
                self.publish('A', producer, '--fallback-reason', producer['fallback_reason'], expected=2)
        receipt_path.write_bytes((json.dumps(producer['attempts'], indent=2) + '\n').encode())
        for digest in (None, '', 'incorrect', '0' * 64):
            reference.write_text(json.dumps({**saved, 'attempts_sha256': digest}))
            self.publish('A', producer, expected=2)
        reference.unlink()
        rejected = self.publish('A', producer, expected=2)
        self.assertIn('producer reference unavailable', rejected['fallback_reason'])
