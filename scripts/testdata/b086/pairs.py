"""Compare paired host cases: the inputs that differ must be exactly the ones the pair varies."""

import argparse
import json
from pathlib import Path

import verify
from common import digest

INPUT_FILES = ('grant.json', 'candidate-stats.json', 'boundaries.json', 'finding.json')


def case_inputs(case):
    return {name: digest(case / name) for name in INPUT_FILES if (case / name).exists()}


def delivered_prompt(case):
    """The prompt the host actually received, as recorded at dispatch, with the case path normalized."""
    try:
        with (case / 'observations/host.jsonl').open() as stream:
            stream.readline()
            observed = json.loads(stream.readline())
        if observed.get('type') != 'b086_host_input':
            return None
        return observed['prompt'].replace(str(case), '{case}')
    except (OSError, KeyError, ValueError):
        return None


def differing_inputs(a, b):
    left, right = case_inputs(a), case_inputs(b)
    return sorted(name for name in set(left) | set(right) if left.get(name) != right.get(name))


def compare_inputs(a, b):
    """Inputs that must be identical across a row 18 pair: grant, delivered prompt, finding and patch."""
    missing = []
    if (a / 'grant.json').read_bytes() != (b / 'grant.json').read_bytes():
        missing.append('identical grant (edit authority and budget)')
    prompts = [delivered_prompt(c) for c in (a, b)]
    if None in prompts:
        missing.append('delivered host prompt recorded at dispatch for both runs')
    elif prompts[0] != prompts[1]:
        missing.append('identical delivered host prompt')
    findings = []
    for case in (a, b):
        decision = json.loads((case / 'packet/review.md.decisions.json').read_bytes())['findings'][0]
        findings.append({k: decision.get(k) for k in ('fingerprint', 'diagnosis', 'summary', 'disposition', 'file', 'line',
                                                       'severity', 'rule', 'reason')})
    if findings[0] != findings[1]:
        missing.append('identical finding, fingerprint and proposed patch')
    commits = [json.loads((c / 'packet/review.md.scope.json').read_bytes())['candidate_commit'] for c in (a, b)]
    if commits[0] == commits[1]:
        missing.append('each input review bound to its own starting candidate')
    results = [json.loads((c / 'observations/host-result.json').read_bytes()) if (c / 'observations/host-result.json').exists() else {} for c in (a, b)]
    manifests = [json.loads((c.parent / 'preparation.json').read_bytes()) for c in (a, b)]
    context = [(m['policy_commit'], m['fixture_commit'], m['review_baseline'], r.get('model'), r.get('runtime'))
               for m, r in zip(manifests, results)]
    if context[0] != context[1]:
        missing.append('same policy, fixture, baseline, host model and runtime across the pair')
    return missing, findings[0]['fingerprint']


def fix_pair(root_a, root_b, name):
    """Row 18: same proposed patch, grant and prompt; only the unrelated diff differs. Both repairs are observed.

    `root_a` is the small candidate and `root_b` carries the large unrelated diff."""
    a, b = root_a / name, root_b / name
    missing, fingerprint = compare_inputs(a, b)
    manifests = [json.loads((r / 'preparation.json').read_bytes()) for r in (root_a, root_b)]
    states = [m['bindings'][name]['workspace'] for m in manifests]
    if states[0]['preserved'] or not states[1]['preserved']:
        missing.append('the first root is the small candidate and the second carries the unrelated diff to preserve')
    stats = [json.loads((c / 'packet/review.md.scope.json').read_bytes()) for c in (a, b)]
    if 'docs/unrelated.md' in stats[0]['changed_files'] or 'docs/unrelated.md' not in stats[1]['changed_files']:
        missing.append('only the large candidate diff holds the unrelated file')
    for root, manifest in zip((root_a, root_b), manifests):
        result = verify.inspect_case(root / name, manifest, fingerprint)
        if result['status'] != 'observed':
            missing.append(f'{root.name} case observed: {result["missing"]}')
    texts = [(c / 'candidate/docs/feature-a.md').read_bytes() for c in (a, b)]
    if texts[0] != texts[1]:
        missing.append('byte-identical resulting repair in both runs')
    if states[0]['expected_text'].encode() != texts[0]:
        missing.append('the resulting repair is the proposed patch')
    if any(digest(b / 'candidate' / n) != d for n, d in states[1]['preserved'].items()):
        missing.append('unrelated candidate files preserved')
    return {'status': 'incomplete' if missing else 'observed', 'missing': missing,
            'resulting_sha256': digest(a / 'candidate/docs/feature-a.md'), 'preserved': list(states[1]['preserved'])}


def effort_pair(root, a, b, allowed, fingerprint='docs/feature-a.md#DOCS#incorrect-default'):
    """Row 22: two repair-then-review cases differ only in the named input files, and both were observed."""
    differing = differing_inputs(root / a, root / b)
    missing = [] if differing == sorted(allowed) else [f'inputs differing {differing}, expected {sorted(allowed)}']
    prompts = [delivered_prompt(root / n) for n in (a, b)]
    if None in prompts or prompts[0] != prompts[1]:
        missing.append('identical delivered host prompt recorded at dispatch')
    manifest = json.loads((root / 'preparation.json').read_bytes())
    for name in (a, b):
        result = verify.inspect_case(root / name, manifest, fingerprint)
        if result['status'] != 'observed':
            missing.append(f'{name} case observed: {result["missing"]}')
    return {'status': 'incomplete' if missing else 'observed', 'missing': missing, 'differing': differing}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='kind', required=True)
    fix = sub.add_parser('fix')
    fix.add_argument('root_a', type=Path)
    fix.add_argument('root_b', type=Path)
    fix.add_argument('case')
    effort = sub.add_parser('effort')
    effort.add_argument('root', type=Path)
    effort.add_argument('a')
    effort.add_argument('b')
    effort.add_argument('--allow', nargs='+', required=True)
    args = parser.parse_args()
    result = (fix_pair(args.root_a, args.root_b, args.case) if args.kind == 'fix'
              else effort_pair(args.root, args.a, args.b, args.allow))
    print(json.dumps(result, indent=2))
    raise SystemExit(0 if result['status'] == 'observed' else 1)
