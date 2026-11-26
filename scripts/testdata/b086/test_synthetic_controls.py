"""Exercise the writer, applied gates and private intake without impersonating a host."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

from intake import applied_review, consume, received_pin
from packet import read_input
from prepare import build_policy, digest, seed_project
from synthetic import applied_history, generate, write_json

HERE = Path(__file__).resolve().parent


class SyntheticControls(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.root = Path(os.environ.get('B086_CONTROL_ARTIFACTS', cls.temp.name)).resolve(strict=True)
        if not cls.root.is_relative_to(Path('/tmp')) or any(cls.root.iterdir()):
            raise ValueError('controls need an empty scratch directory')
        cls.source = cls.root / 'policy'
        cls.source.mkdir()
        cls.binary = cls.source / 'bin/shark'
        cls.seed = seed_project(cls.source, cls.root, build_policy(cls.source))

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def test_independent_attribution_and_scenario_controls(self):
        for variant, reason in [('report', 'report lacks synthetic runner disclosure'),
                                ('provenance', 'provenance lacks synthetic input disclosure'),
                                ('operation', 'applied evidence lacks synthetic input disclosure'),
                                ('model', 'effective model has no observed invocation'),
                                ('maintenance', 'optional Feature A worked example'),
                                ('fix-now', 'surviving actionable finding')]:
            project = self.root / variant
            with self.subTest(variant=variant):
                with self.assertRaisesRegex(ValueError, reason) as refusal:
                    generate(project, self.seed, self.source, variant)
                scope = json.loads((project / 'fixture-inputs/scope.json').read_bytes())
                report = project / 'fixture-review/review.md'
                if variant not in {'style', 'fix-now'}:
                    read_input(report, digest(report), project, scope['candidate_commit'])
                run, = (project / '.shark/runs').glob('fixture-code_review-*')
                state = json.loads((run / 'operation-state.json').read_bytes())
                history = dict(project=str(project), entity='B001', run_id=run.name,
                               operation_digest=state['operation_digest'])
                envelope = applied_review(history, self.binary)
                write_json(project / 'control-result.json', dict(reason=str(refusal.exception),
                           history=history, evidence=envelope['evidence'], report_sha256=digest(report)))

    def test_stylistic_downgrade_is_a_valid_report_only_packet(self):
        project = self.root / 'style-downgrade'
        artifacts, _, pin = generate(project, self.seed, self.source, 'style-downgrade')
        report = (project / 'fixture-review/review.md').read_text()
        decisions = json.loads((project / 'fixture-review/review.md.decisions.json').read_bytes())
        angles = json.loads((project / 'fixture-review/review.md.angles.json').read_bytes())
        self.assertEqual(decisions['findings'][0]['severity'], 'nit')
        self.assertEqual(decisions['findings'][0]['disposition'], 'no-action')
        self.assertEqual(angles['a']['findings'][0]['severity'], 'blocker')
        self.assertIn('current title already names Feature A', report)
        self.assertIn('no project standard requires a longer title', report)
        self.assertEqual(pin['history']['entity'], 'B001')
        self.assertIn('review.md.decisions.json', artifacts)
        provenance = json.loads(artifacts['provenance.json'])
        self.assertEqual(provenance['scenario'], 'stylistic downgrade report-only')

    def test_mandatory_fix_now_is_its_own_packet_class_and_triage_packets_still_refuse_it(self):
        project = self.root / 'mandatory'
        artifacts, _, pin = generate(project, self.seed, self.source, 'mandatory')
        decisions = json.loads(artifacts['review.md.decisions.json'])['findings']
        self.assertEqual([(d['disposition'], d['fingerprint']) for d in decisions],
                         [('fix-now', 'docs/feature-a.md#DOCS#incorrect-default')])
        self.assertEqual(json.loads(artifacts['provenance.json'])['scenario'],
                         'mandatory incorrect default order under Feature A')
        report = project / 'fixture-review/review.md'
        scope = json.loads((project / 'fixture-review/review.md.scope.json').read_bytes())
        with self.assertRaisesRegex(ValueError, 'surviving actionable finding'):
            read_input(report, digest(report), project, scope['candidate_commit'])
        self.assertEqual(pin['history']['entity'], 'B001')

    def test_each_intake_refuses_a_different_valid_packet_or_operation(self):
        root = self.root / 'intake-pair'
        root.mkdir()
        subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic'], check=True, capture_output=True)
        prepared = json.loads((root / 'preparation.json').read_bytes())
        pin = prepared['review_intake']
        project = Path(pin['history']['project'])
        packet = project / 'fixture-review'
        scope = json.loads((packet / 'review.md.scope.json').read_bytes())
        other = self.root / 'other-packet'
        _, _, alternate = generate(other, self.seed, self.source)
        self.assertNotEqual(alternate['bundle_sha256'], pin['bundle_sha256'])
        self.assertEqual(json.loads((other / 'fixture-review/review.md.scope.json').read_bytes())['candidate_commit'],
                         scope['candidate_commit'])
        alternate_op = applied_history(project, self.binary, packet, scope['base_commit'], None, 'B002')
        # Both alternatives must actually pass verification under their own identities.
        consume(packet, alternate_op, dict(pin, history=alternate_op), self.binary)
        self.assertNotEqual(alternate_op['operation_digest'], pin['history']['operation_digest'])
        ref = root / 'alternate-operation.json'
        write_json(ref, alternate_op)
        for name in prepared['cases']:
            case = root / name
            command = [sys.executable, HERE / 'observe.py', case, 'intake']
            for variant, bundle, history, reason in [
                ('bundle', other / 'fixture-review', case / 'history.json', 'intake bundle differs'),
                ('operation', case / 'packet', ref, 'intake history differs'),
            ]:
                with self.subTest(grant=name, variant=variant):
                    before = digest(case / 'project/shark-tasks.db')
                    refused = subprocess.run([*command, bundle, history], capture_output=True, text=True)
                    self.assertNotEqual(refused.returncode, 0)
                    self.assertIn(reason, refused.stderr)
                    self.assertEqual(before, digest(case / 'project/shark-tasks.db'))
                    events = [json.loads(line) for line in (case / 'observations/tools.jsonl').read_text().splitlines()]
                    self.assertEqual([event['tool'] for event in events], ['intake', 'intake'])
                    self.assertEqual(events[-1]['outcome'], 'refused')
                    (case / 'observations').rename(case / ('refused-' + variant))
            subprocess.run([*command, case / 'packet', case / 'history.json'], check=True, capture_output=True)
        def events(case):
            return [json.loads(line) for line in (root / case / 'observations/tools.jsonl').read_text().splitlines()]
        for name, other_name in [('grant-a', 'grant-b'), ('grant-b', 'grant-a')]:
            run = prepared['bindings'][name]['run_id']
            self.assertTrue(received_pin(events(name), run, pin))
            self.assertFalse(received_pin(events(name)[:1], run, pin))
            self.assertFalse(received_pin(events(other_name), run, pin))
        # Preserve a valid envelope while omitting its applied history on a disposable copy.
        copy = self.root / 'missing-transition'
        shutil.copytree(project, copy)
        db = copy / 'shark-tasks.db'
        import sqlite3
        with sqlite3.connect(db) as connection:
            connection.execute("delete from entity_history where notes like ?", ('%' + pin['history']['run_id'] + '%',))
        self.assertTrue((copy / '.shark/runs' / pin['history']['run_id'] / 'result.json').is_file())
        with self.assertRaisesRegex(ValueError, 'missing applied code-review transition'):
            applied_review(dict(pin['history'], project=str(copy)), self.binary)
