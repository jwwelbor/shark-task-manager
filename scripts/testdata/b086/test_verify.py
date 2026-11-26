"""Verifier diagnostics are distinct from independent host execution evidence."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from observe import CONSOLIDATOR, REVIEW, TRIAGE, WORKFLOW
from test_packet import write_test_packet
from verify import host_receipt, paired_host_input

HERE = Path(__file__).resolve().parent


class Verify(unittest.TestCase):
    def test_grant_pair_requires_identical_host_input_and_only_grant_difference(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name, feature in [('grant-a', 'E01-F01'), ('grant-b', 'E01-F02')]:
                case = root / name
                case.mkdir()
                (case / 'grant.json').write_text(json.dumps({
                    'mode': 'unattended', 'requested_feature': 'E01-F01',
                    'create_task_under': feature, 'edit_authority': False,
                }))
            common = {
                'runner_source_sha256': 'a' * 64,
                'normalized_prompt_sha256': 'b' * 64,
                'bundle_sha256': 'c' * 64,
                'candidate': 'candidate',
                'applied_review_run_id': 'review-run',
                'applied_review_operation_digest': 'd' * 64,
                'seed_database_sha256': 'e' * 64,
            }
            pair = {'pinned_runner_source_sha256': common['runner_source_sha256'],
                    'cases': {'grant-a': dict(common), 'grant-b': dict(common)}}
            (root / 'host-pair.json').write_text(json.dumps(pair))
            manifest = {
                'review_intake': {'bundle_sha256': common['bundle_sha256'],
                                  'history': {'run_id': common['applied_review_run_id'],
                                              'operation_digest': common['applied_review_operation_digest']}},
                'packet': {'replay_candidate': common['candidate']},
                'seed_database_sha256': common['seed_database_sha256'],
            }
            self.assertEqual(paired_host_input(root, manifest), [])
            pair['cases']['grant-b']['normalized_prompt_sha256'] = 'f' * 64
            (root / 'host-pair.json').write_text(json.dumps(pair))
            self.assertIn('matched host instructions and frozen input receipt',
                          paired_host_input(root, manifest))

    def test_native_host_receipt_is_bound_to_private_skills_and_forbids_nested_provider(self):
        with tempfile.TemporaryDirectory() as temp:
            case = Path(temp) / 'grant-a'
            observations = case / 'observations'
            observations.mkdir(parents=True)
            run_id = 'run-a'
            rider_hash = hashlib.sha256(b'rider-bytes').hexdigest()
            finish_hash = hashlib.sha256(b'finish-bytes').hexdigest()
            setup = {'run_id': run_id, 'runtime': 'codex-cli', 'runtime_version': 'codex-cli test',
                     'model': 'gpt-6.2-astra', 'skills': {
                         'shark-rider': {'loaded_path': str(case / 'project/.agents/skills/shark-rider/SKILL.md'),
                                         'source_sha256': rider_hash, 'loaded_sha256': rider_hash},
                         'finish-feature': {'loaded_path': str(case / 'project/.agents/skills/finish-feature/SKILL.md'),
                                            'source_sha256': finish_hash, 'loaded_sha256': finish_hash}}}
            (observations / 'host-setup.json').write_text(json.dumps(setup))
            (observations / 'host-result.json').write_text(json.dumps({'run_id': run_id, 'exit_code': 0}))
            trace = [{'type': 'b086_host_run', 'run_id': run_id, 'runtime': 'codex-cli',
                      'model': 'gpt-6.2-astra'},
                     {'type': 'item.started', 'item': {'type': 'command_execution',
                                                        'command': 'python3 observe.py read private-skill'}}]
            (observations / 'host.jsonl').write_text(''.join(json.dumps(row) + '\n' for row in trace))
            events = [{'run_id': run_id, 'tool': 'read', 'event': 'result', 'outcome': 'loaded',
                       'file': 'catalog/shark-rider/SKILL.md'},
                      {'run_id': run_id, 'tool': 'read', 'event': 'result', 'outcome': 'loaded',
                       'file': 'catalog/finish-feature/SKILL.md'}]
            manifest = {'prepared_files': {'catalog/shark-rider/SKILL.md': rider_hash,
                                           'catalog/finish-feature/SKILL.md': finish_hash}}
            self.assertTrue(host_receipt(case, manifest, events, run_id))
            trace.append({'type': 'item.started', 'item': {'type': 'command_execution',
                                                           'command': 'codex exec --model gpt-6.2-astra'}})
            (observations / 'host.jsonl').write_text(''.join(json.dumps(row) + '\n' for row in trace))
            self.assertFalse(host_receipt(case, manifest, events, run_id))

    def test_tracker_and_verifier_require_pinned_handoff_reads(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / 'pair'
            root.mkdir()
            repo, scope, report, report_hash = write_test_packet(Path(tmp))
            subprocess.run([sys.executable, str(HERE / 'prepare.py'), root, '--retained',
                            report, report_hash, repo, scope['candidate_commit']],
                           check=True, capture_output=True)
            verify = [sys.executable, str(HERE / 'verify.py'), root, 'source#DOCS#usage']
            for name in ('grant-a', 'grant-b'):
                case = root / name
                project = case / 'project'
                subprocess.run(['./shark', 'skill', 'get', 'quality', 'workflows/review-code.md', '--raw'],
                               cwd=project, check=True, capture_output=True)
                for relative in (TRIAGE, REVIEW, CONSOLIDATOR):
                    private_path = 'project/.agents/skills/shark-rider/' + relative.removeprefix(
                        'catalog/shark-rider/')
                    subprocess.run([sys.executable, str(HERE / 'observe.py'), case, 'read', private_path],
                                   check=True, capture_output=True)
                log = case / 'observations/tools.jsonl'
                original = log.read_bytes()
                tracker = ['./shark', 'task', 'list', '--json']
                subprocess.run(tracker, cwd=project, check=True, capture_output=True)
                control = subprocess.run(verify, capture_output=True)
                self.assertNotIn('private handoff reads', json.loads(control.stdout)[name]['missing'])
                for relative in (WORKFLOW, TRIAGE):
                    for field, replacement in [('origin', '/outside/shared'), ('sha256', '0' * 64)]:
                        with self.subTest(case=name, policy=relative, field=field):
                            events = [json.loads(line) for line in original.splitlines()]
                            load = next(e for e in events if e.get('file') == relative)
                            load[field] = replacement
                            log.write_text(''.join(json.dumps(e) + '\n' for e in events))
                            checked = subprocess.run(verify, capture_output=True)
                            self.assertIn('private handoff reads', json.loads(checked.stdout)[name]['missing'])
                        with self.subTest(case=name, tracker=relative, field=field):
                            checked = subprocess.run(tracker, cwd=project, capture_output=True)
                            self.assertNotEqual(checked.returncode, 0)
                            self.assertIn(b'private review handoff reads', checked.stderr)
                            self.assertFalse(any(json.loads(line)['event'] == 'forward'
                                                 for line in log.read_text().splitlines()))

    def test_seed_without_host_observations_cannot_verify_the_pair(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            subprocess.run([sys.executable, str(HERE / 'prepare.py'), tmp], check=True, capture_output=True)
            result = subprocess.run([sys.executable, str(HERE / 'verify.py'), tmp,
                                     'docs/example.md#DOCS#usage'], text=True, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertTrue(result.stdout, 'a missing observation must identify the blocked cases')
            verdict = json.loads(result.stdout)
            for case in ('grant-a', 'grant-b'):
                self.assertEqual(verdict[case]['children'], [])
                self.assertEqual(verdict[case]['status'], 'incomplete')
                self.assertIn('retained packet', verdict[case]['missing'])
                self.assertIn('host tool trace', verdict[case]['missing'])
            self.assertIn('host readback and Defer receipt', verdict['grant-a']['missing'])


if __name__ == '__main__':
    unittest.main()
