"""Observe the fault at the real tracker entry without making a host decision."""

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from observe import CONSOLIDATOR, REVIEW, TRIAGE, WORKFLOW, tracker_args
import verify

HERE = Path(__file__).resolve().parent


class DefinitiveRefusal(unittest.TestCase):
    def test_verified_prior_pair_requires_both_grants_and_run_ids(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'pair.json'
            pair = {
                'grant-a': {'status': 'observed', 'run_id': 'a'},
                'grant-b': {'status': 'observed', 'run_id': 'b'},
                'pair_input': {'status': 'observed'},
            }
            path.write_text(json.dumps(pair))
            self.assertTrue(verify.verified_prior_pair(path))
            pair['grant-b']['status'] = 'incomplete'
            path.write_text(json.dumps(pair))
            self.assertFalse(verify.verified_prior_pair(path))

    def test_only_the_selected_create_is_refused_before_persistence(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            prepared = subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic',
                                       '--with-refusal-control'], capture_output=True)
            self.assertEqual(prepared.returncode, 0, prepared.stderr)
            manifest = json.loads((root / 'preparation.json').read_bytes())
            willing, refusing = root / 'grant-a', root / 'case-c'
            self.assertEqual((willing / 'grant.json').read_bytes(), (refusing / 'grant.json').read_bytes())
            self.assertEqual((willing / 'history.json').read_bytes(), (refusing / 'history.json').read_bytes())
            self.assertEqual((willing / 'packet/manifest.json').read_bytes(),
                             (refusing / 'packet/manifest.json').read_bytes())
            for case in (willing, refusing):
                observer = [sys.executable, case / 'libexec/observe.py', case]
                subprocess.run([*observer, 'intake', case / 'packet', case / 'history.json'],
                               check=True, capture_output=True)
                for name in (WORKFLOW, TRIAGE, REVIEW, CONSOLIDATOR):
                    private = name.replace('catalog/', 'project/.agents/skills/', 1)
                    subprocess.run([*observer, 'read', private], check=True, capture_output=True)
                project = case / 'project'
                subprocess.run(['./shark', 'list', 'E01-F01', '--json'], cwd=project,
                               check=True, capture_output=True)
                result = subprocess.run(['./shark', 'create', 'task', 'E01-F01', 'Worked example',
                                         '--json'], cwd=project, capture_output=True)
                events = [json.loads(line) for line in (case / 'observations/tools.jsonl').read_bytes().splitlines()]
                create = next(e for e in events if e.get('argv', [])[:2] == ['create', 'task'])
                attempts = [e for e in events if e['invocation'] == create['invocation']]
                binding = manifest['bindings'][case.name]
                children = verify.read_children(project / 'shark-tasks.db', binding)
                if case == willing:
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual([e['event'] for e in attempts], ['entry', 'forward', 'result'])
                    self.assertEqual([row['feature'] for row in children], ['E01-F01'])
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual([e['event'] for e in attempts], ['entry', 'result'])
                    self.assertEqual(attempts[-1]['outcome'], 'refused')
                    self.assertIs(attempts[-1]['forwarded'], False)
                    self.assertEqual(attempts[-1]['reason'], binding['create_refusal'])
                    self.assertIn(binding['create_refusal'].encode(), result.stderr)
                    self.assertEqual(children, [])

            # The parent's default verifier can still check the first two cases independently.
            command = [sys.executable, HERE / 'verify.py', root, 'docs/feature-a.md#DOCS#worked-example']
            first_pair = subprocess.run(command, capture_output=True)
            self.assertNotIn('case-c', json.loads(first_pair.stdout))
            with_refusal = subprocess.run([*command, '--include-refusal'], capture_output=True)
            self.assertNotEqual(with_refusal.returncode, 0)
            missing = json.loads(with_refusal.stdout)['case-c']['missing']
            self.assertIn('received unresolved fingerprint, failed create, feature, refusal and next action', missing)
            self.assertIn('verified first grant pair and matching-grant control', missing)

    def test_received_escalation_requires_failed_operation_target_reason_and_next_action(self):
        # These are diagnostic receiver inputs, separate from the live host case.
        reason = 'Tracker refused child creation under E01-F01'
        fp = 'docs/feature-a.md#DOCS#worked-example'
        message = {'fingerprint': fp, 'requested_feature': 'E01-F01',
                   'attempted_operation': 'shark create task E01-F01',
                   'refusal_reason': reason, 'requested_parent_action': 'Assign a writable owner.'}
        failed = {'event': 'result', 'tool': 'shark', 'invocation': 'one',
                  'outcome': 'refused', 'reason': reason, 'forwarded': False}
        received = {'event': 'result', 'tool': 'parent', 'invocation': 'two', 'outcome': 'received'}
        inbox = {'invocation': 'two', 'message': message}

        def observed(events, deliveries):
            return verify.received_create_refusal(events, deliveries, 'one', fp, reason)

        self.assertTrue(observed([failed, received], [inbox]))
        self.assertFalse(observed([failed, received], []))
        self.assertFalse(observed([received, failed], [inbox]))
        self.assertFalse(observed([dict(failed, forwarded=True), received], [inbox]))
        for field in message:
            incomplete = dict(message)
            del incomplete[field]
            with self.subTest(field=field):
                self.assertFalse(observed([failed, received], [dict(inbox, message=incomplete)]))

    def test_failed_or_late_duplicate_lookup_cannot_explain_an_attempted_create(self):
        fp = 'docs/feature-a.md#DOCS#worked-example'

        def lookup(key, argv, output='[]', code=0):
            return [{'event': 'entry', 'tool': 'shark', 'invocation': key, 'argv': argv},
                    {'event': 'result', 'tool': 'shark', 'invocation': key,
                     'exit_code': code, 'stdout': output}]
        exact = lookup('exact', ['notes', 'search', fp, '--json'])
        scope = lookup('scope', ['list', 'E01-F01', '--json'])
        create = {'event': 'entry', 'tool': 'shark', 'invocation': 'create', 'argv': ['create', 'task']}
        self.assertTrue(verify.empty_owner_searches([*exact, *scope, create], 'create', fp))
        # `shark search` returns JSON null for no untyped results.
        null_exact = lookup('exact', ['search', fp, '--json'], output='null')
        self.assertTrue(verify.empty_owner_searches([*null_exact, *scope, create], 'create', fp))
        textual_exact = lookup('exact', ['notes', 'search', fp],
                               output=f'No results found for "{fp}"')
        self.assertTrue(verify.empty_owner_searches([*textual_exact, *scope, create], 'create', fp))
        for wrong in [[], lookup('scope', ['list', 'E01-F02', '--json']),
                      lookup('scope', ['list', 'E01-F01'], code=1),
                      lookup('scope', ['list', 'E01-F01'], output='[{"key":"owned"}]')]:
            self.assertFalse(verify.empty_owner_searches([*exact, *wrong, create], 'create', fp))
        self.assertFalse(verify.empty_owner_searches([*scope, *exact, create], 'create', fp))
        self.assertFalse(verify.empty_owner_searches([*exact, create, *scope], 'create', fp))

    def test_refusal_target_follows_the_actual_create_parent(self):
        project = Path('/tmp/scratch')
        for args in [
                ['create', 'task', 'E01-F01', 'Title'],
                ['task', 'create', 'E01', 'F01', 'Title'],
                ['task', 'create', '--feature', 'F01', 'E01', 'Title'],
                ['task', 'create', '--feature=E01-F01', 'Title'],
                ['create', 'task', '--epic=e01', '--feature=f01', 'Title']]:
            with self.subTest(args=args):
                _, operation, feature = tracker_args(args, project, project / 'db', project / 'config')
                self.assertEqual((operation, feature), ('create', 'E01-F01'))
