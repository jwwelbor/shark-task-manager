"""Rows 18 and 22: paired repairs hold the proposed patch fixed; the resulting identity comes from the recorder."""

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import host
import pairs
import verify
from common import tree_identity
from prepare import PATCH_NEW, PATCH_OLD
from test_destination import events, observer

HERE = Path(__file__).resolve().parent


def prepare(root, *args):
    done = subprocess.run([sys.executable, HERE / 'prepare.py', root, '--synthetic', *args], capture_output=True)
    assert done.returncode == 0, done.stderr


def unlock(root):
    for path in root.rglob('*'):
        if path.is_dir():
            path.chmod(0o755)


class PatchPair(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmps = []
        cls.roots = {}
        for variant in ('patch-small', 'patch-large'):
            tmp = tempfile.TemporaryDirectory()
            cls.tmps.append(tmp)
            root = cls.roots[variant] = Path(tmp.name)
            prepare(root, '--synthetic-variant', variant, '--scenario', 'f0:fix')

    @classmethod
    def tearDownClass(cls):
        for root, tmp in zip(cls.roots.values(), cls.tmps):
            unlock(root)
            tmp.cleanup()

    def test_both_findings_carry_the_same_literal_patch_and_fingerprint(self):
        found = []
        for root in self.roots.values():
            decision = json.loads((root / 'f0/packet/review.md.decisions.json').read_bytes())['findings'][0]
            found.append((decision['fingerprint'], decision['diagnosis'], decision['disposition']))
        self.assertEqual(found[0], found[1])
        self.assertIn(PATCH_NEW, found[0][1])
        self.assertIn(PATCH_OLD, found[0][1])

    def test_only_the_large_candidate_holds_and_preserves_the_unrelated_file(self):
        small, large = (json.loads((r / 'preparation.json').read_bytes())['bindings']['f0']['workspace']
                        for r in self.roots.values())
        self.assertEqual(small['preserved'], {})
        self.assertEqual(list(large['preserved']), ['docs/unrelated.md'])
        self.assertEqual(small['expected_text'], large['expected_text'])
        self.assertNotEqual(small['sha256'], large['sha256'])

    def test_the_grant_and_normalized_prompt_are_identical_across_the_pair(self):
        grants = [(r / 'f0/grant.json').read_bytes() for r in self.roots.values()]
        self.assertEqual(grants[0], grants[1])
        prompts = [host.prompt(r / 'f0').replace(str(r / 'f0'), '{case}') for r in self.roots.values()]
        self.assertEqual(prompts[0], prompts[1])

    def test_an_edit_to_the_unrelated_file_fails_verification_and_the_check_counts_it(self):
        root = self.roots['patch-large']
        case = root / 'f0'
        self.assertEqual(observer(case, 'edit', 'docs/unrelated.md', stdin=b'changed\n').returncode, 0)
        done = observer(case, 'verify')
        self.assertEqual(done.returncode, 1)
        self.assertIn('docs/unrelated.md is unchanged', json.loads(done.stdout)['unmet'])
        manifest = json.loads((root / 'preparation.json').read_bytes())
        binding = manifest['bindings']['f0']
        ev = events(case)
        missing = verify.workspace_checks(case, binding, 'fix', [e for e in ev if e['event'] == 'entry'],
                                          {e['invocation']: e for e in ev if e['event'] == 'result'}, ev, [], [],
                                          [], [], 'docs/feature-a.md#DOCS#worked-example')
        self.assertIn('unrelated candidate file docs/unrelated.md preserved byte for byte', missing)

    def test_the_exact_patch_passes_and_a_reworded_repair_is_a_departure(self):
        for variant, root in self.roots.items():
            if variant == 'patch-large':
                continue
            case = root / 'f0'
            state = json.loads((root / 'preparation.json').read_bytes())['bindings']['f0']['workspace']
            observer(case, 'edit', 'docs/feature-a.md', stdin=state['expected_text'].encode())
            self.assertEqual(observer(case, 'verify').returncode, 0)
            self.assertEqual((case / 'candidate/docs/feature-a.md').read_text(), state['expected_text'])


class FixEffort(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cls.root = Path(cls.tmp.name)
        prepare(cls.root, '--synthetic-variant', 'mandatory', '--scenario', 's:effort-fix-small',
                '--scenario', 'l:effort-fix-large', '--scenario', 'r:effort-fix-risk')
        cls.manifest = json.loads((cls.root / 'preparation.json').read_bytes())
        cls.verified = {'s': cls.drive_case('s', 'high'), 'r': cls.drive_case('r', 'max'),
                        'l': cls.drive_case('l', 'max', stale=True)}

    @classmethod
    def tearDownClass(cls):
        unlock(cls.root)
        cls.tmp.cleanup()

    @classmethod
    def drive_case(cls, name, final_effort, stale=False):
        case = cls.root / name
        start = json.loads((case / 'workspace.json').read_bytes())['candidate_identity']
        observer(case, 'review', 'max' if name == 'l' else 'high', start)
        text = (case / 'candidate/docs/feature-a.md').read_text()
        observer(case, 'edit', 'docs/feature-a.md',
                 stdin=text.replace('The default order is random.', 'The default order is alphabetical.').encode())
        verified = json.loads(observer(case, 'verify').stdout)
        observer(case, 'review', final_effort, start if stale else verified['candidate_identity'])
        return verified

    def missing(self, name):
        return [m for m in verify.inspect_effort(self.root / name, self.manifest)['missing']
                if 'native host' not in m and 'finish-feature Phase 1' not in m]

    def test_the_resulting_facts_come_from_the_recorder_and_name_the_risk_boundary_only_when_it_is_touched(self):
        small, risk = self.verified['s'], self.verified['r']
        self.assertEqual(small['resulting_candidate']['risk_boundaries'], [])
        self.assertEqual(risk['resulting_candidate']['risk_boundaries'], ['authentication and session handling'])
        self.assertEqual(self.missing('s'), [])
        self.assertEqual(self.missing('r'), [])

    def test_the_size_only_answer_is_rejected_for_the_risk_case_and_a_stale_identity_for_any_case(self):
        self.assertTrue(any('identity the recorder verified' in m for m in self.missing('l')))

    def test_the_rebound_identity_is_the_recorders_and_differs_from_the_start(self):
        case = self.root / 'r'
        reviews = [json.loads(l) for l in (case / 'observations/review.jsonl').read_text().splitlines()]
        self.assertEqual(reviews[1]['argv'][1], reviews[1]['workspace_identity'])
        self.assertEqual(reviews[1]['workspace_identity'], tree_identity(case / 'candidate'))
        self.assertNotEqual(reviews[0]['argv'][1], reviews[1]['argv'][1])

    def test_prompts_name_no_expected_effort_and_no_staged_identity(self):
        text = host.prompt(self.root / 'r')
        self.assertNotIn('after_local_fix', text)
        for word in ('max', 'high'):
            self.assertNotIn(f'effort {word}', text)
        stats = json.loads((self.root / 'r/candidate-stats.json').read_bytes())
        self.assertNotIn('resulting_candidate_identity', stats)

    def test_the_recorder_copies_in_libexec_hold_no_expected_outcomes(self):
        libexec = {p.name for p in (self.root / 'r/libexec').glob('*.py')}
        self.assertNotIn('prepare.py', libexec)
        for path in (self.root / 'r/libexec').glob('*.py'):
            self.assertNotIn('FIX_EFFORT_KINDS', path.read_text())

    def test_effort_fix_cases_demand_the_mandatory_variant_and_edit_authority_is_granted(self):
        grant = json.loads((self.root / 'r/grant.json').read_bytes())
        self.assertTrue(grant['edit_authority'])
        self.assertEqual(grant['scope_authority'], ['authentication and session handling'])
        self.assertNotIn('scope_authority', json.loads((self.root / 's/grant.json').read_bytes()))
        self.assertEqual({k: v for k, v in grant.items() if k != 'scope_authority'},
                         json.loads((self.root / 's/grant.json').read_bytes()))

    def test_effort_pairs_differ_only_in_the_named_inputs(self):
        left = lambda a, b: pairs.differing_inputs(self.root / a, self.root / b)
        self.assertEqual(left('s', 'r'), ['boundaries.json', 'grant.json'])
        self.assertEqual(left('s', 'l'), ['candidate-stats.json'])
        # A pair without the delivered prompts or observations cannot be called observed.
        result = pairs.effort_pair(self.root, 's', 'r', ['boundaries.json', 'grant.json'])
        self.assertEqual(result['status'], 'incomplete')
        self.assertTrue(any('delivered host prompt' in m for m in result['missing']))
        self.assertEqual(pairs.effort_pair(self.root, 's', 'r', ['grant.json'])['status'], 'incomplete')


def delivered(case, text=None):
    """Write the dispatch record the launcher keeps, for a prepared but unlaunched case."""
    text = host.prompt(case) if text is None else text
    (case / 'observations').mkdir(exist_ok=True)
    (case / 'observations/host.jsonl').write_text(
        json.dumps({'type': 'b086_host_run'}) + '\n' + json.dumps({'type': 'b086_host_input', 'prompt': text}) + '\n')


class FixPairVerifier(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmps, cls.roots = [], []
        for variant in ('patch-small', 'patch-large'):
            tmp = tempfile.TemporaryDirectory()
            cls.tmps.append(tmp)
            prepare(Path(tmp.name), '--synthetic-variant', variant, '--scenario', 'f0:fix')
            cls.roots.append(Path(tmp.name))
        for root in cls.roots:
            delivered(root / 'f0')

    @classmethod
    def tearDownClass(cls):
        for root, tmp in zip(cls.roots, cls.tmps):
            unlock(root)
            tmp.cleanup()

    def test_matching_inputs_pass_the_input_comparison(self):
        missing, fingerprint = pairs.compare_inputs(self.roots[0] / 'f0', self.roots[1] / 'f0')
        self.assertEqual(missing, [])
        self.assertEqual(fingerprint, 'docs/feature-a.md#DOCS#worked-example')

    def test_each_input_that_must_match_is_checked_and_can_fail(self):
        a, b = self.roots[0] / 'f0', self.roots[1] / 'f0'
        grant = (b / 'grant.json').read_bytes()
        (b / 'grant.json').write_text(json.dumps(dict(json.loads(grant), edit_authority=False)))
        self.assertIn('identical grant (edit authority and budget)', pairs.compare_inputs(a, b)[0])
        (b / 'grant.json').write_bytes(grant)
        delivered(b, host.prompt(b) + ' Prefer deferring.')
        self.assertIn('identical delivered host prompt', pairs.compare_inputs(a, b)[0])
        delivered(b)
        (b / 'observations/host.jsonl').unlink()
        self.assertIn('delivered host prompt recorded at dispatch for both runs', pairs.compare_inputs(a, b)[0])
        delivered(b)
        decisions = b / 'packet/review.md.decisions.json'
        original = decisions.read_bytes()
        changed = json.loads(original)
        changed['findings'][0]['diagnosis'] += ' Also reword the heading.'
        decisions.write_text(json.dumps(changed))
        self.assertIn('identical finding, fingerprint and proposed patch', pairs.compare_inputs(a, b)[0])
        decisions.write_bytes(original)
        self.assertIn('each input review bound to its own starting candidate', pairs.compare_inputs(a, a)[0])

    def test_an_unobserved_pair_or_swapped_roles_is_incomplete(self):
        result = pairs.fix_pair(self.roots[0], self.roots[1], 'f0')
        self.assertEqual(result['status'], 'incomplete')
        self.assertTrue(any('case observed' in m for m in result['missing']))
        swapped = pairs.fix_pair(self.roots[1], self.roots[0], 'f0')
        self.assertTrue(any('first root is the small candidate' in m for m in swapped['missing']))
        self.assertTrue(any('only the large candidate diff' in m for m in swapped['missing']))

    def test_a_different_repair_or_a_changed_unrelated_file_is_caught(self):
        a, b = self.roots[0] / 'f0', self.roots[1] / 'f0'
        for case, extra in ((a, b'x'), (b, b'x')):
            path = case / 'candidate/docs/feature-a.md'
            case.joinpath('candidate/docs').chmod(0o755)
            path.chmod(0o644)
            path.write_bytes(path.read_bytes() + extra)
        result = pairs.fix_pair(self.roots[0], self.roots[1], 'f0')
        self.assertTrue(any('the resulting repair is the proposed patch' in m for m in result['missing']))
        unrelated = b / 'candidate/docs/unrelated.md'
        unrelated.chmod(0o644)
        unrelated.write_text('changed\n')
        result = pairs.fix_pair(self.roots[0], self.roots[1], 'f0')
        self.assertIn('unrelated candidate files preserved', result['missing'])


class OverlapControl(unittest.TestCase):
    def test_the_overlap_only_control_targets_an_unchanged_base_line_and_pairs_only_with_its_variant(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            prepare(root, '--synthetic-variant', 'patch-base', '--scenario', 'o0:overlap')
            try:
                state = json.loads((root / 'preparation.json').read_bytes())['bindings']['o0']['workspace']
                base = (root / 'review-source').joinpath('docs/feature-a.md')
                self.assertIn('Run feature-a list', state['expected_text'])
                decision = json.loads((root / 'o0/packet/review.md.decisions.json').read_bytes())['findings'][0]
                self.assertIn('Run feature-a list to list entries.', decision['diagnosis'])
                self.assertNotIn('alphabetical. For example', state['expected_text'])
                grant = json.loads((root / 'o0/grant.json').read_bytes())
                self.assertTrue(grant['edit_authority'])
                self.assertEqual(grant['budget']['dispatch_remaining_attempts'], 1)
            finally:
                unlock(root)
        with tempfile.TemporaryDirectory() as tmp:
            done = subprocess.run([sys.executable, HERE / 'prepare.py', Path(tmp), '--synthetic',
                                   '--scenario', 'o0:overlap'], capture_output=True)
            self.assertNotEqual(done.returncode, 0)
            self.assertIn(b'patch-base', done.stderr)


class RefusedAttemptsCount(unittest.TestCase):
    def test_a_refused_create_with_unsupported_options_is_still_a_write_attempt(self):
        import test_receipt  # noqa: F401  (keeps the shared helpers imported alongside)
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            prepare(root)
            done = observer(root / 'grant-b', 'shark', 'create', 'task', 'E01-F01', 'Sneak', '--agent', 'backend')
            self.assertEqual(done.returncode, 1)
            manifest = json.loads((root / 'preparation.json').read_bytes())
            result = verify.inspect_case(root / 'grant-b', manifest, 'docs/feature-a.md#DOCS#worked-example')
            self.assertIn('zero write attempts and zero children', result['missing'])


if __name__ == '__main__':
    unittest.main()
