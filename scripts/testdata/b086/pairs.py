"""Compare paired host cases: the inputs that differ must be exactly the ones the pair varies."""

import argparse
import json
from pathlib import Path

import host
import verify
from common import digest

INPUT_FILES = ('grant.json', 'candidate-stats.json', 'boundaries.json', 'finding.json')


def case_inputs(case):
    return {name: digest(case / name) for name in INPUT_FILES if (case / name).exists()}


def normalized_prompt(case):
    return host.prompt(case).replace(str(case), '{case}')


def differing_inputs(a, b):
    left, right = case_inputs(a), case_inputs(b)
    return sorted(name for name in set(left) | set(right) if left.get(name) != right.get(name))


def fix_pair(root_a, root_b, name):
    """Row 18: same proposed patch, grant and prompt; only the unrelated diff differs. Both repairs are observed."""
    missing = []
    a, b = root_a / name, root_b / name
    if (a / 'grant.json').read_bytes() != (b / 'grant.json').read_bytes():
        missing.append('identical grant (edit authority and budget)')
    if normalized_prompt(a) != normalized_prompt(b):
        missing.append('identical normalized host prompt')
    findings = []
    for case in (a, b):
        decision = json.loads((case / 'packet/review.md.decisions.json').read_bytes())['findings'][0]
        findings.append({k: decision[k] for k in ('fingerprint', 'diagnosis', 'summary', 'disposition', 'file', 'line')})
    if findings[0] != findings[1]:
        missing.append('identical finding, fingerprint and proposed patch')
    commits = [json.loads((c / 'packet/review.md.scope.json').read_bytes())['candidate_commit'] for c in (a, b)]
    if commits[0] == commits[1]:
        missing.append('each input review bound to its own starting candidate')
    manifests = [json.loads((r / 'preparation.json').read_bytes()) for r in (root_a, root_b)]
    results = []
    for root, manifest in zip((root_a, root_b), manifests):
        result = verify.inspect_case(root / name, manifest, findings[0]['fingerprint'])
        results.append(result)
        if result['status'] != 'observed':
            missing.append(f'{root.name} case observed: {result["missing"]}')
    texts = [(c / 'candidate/docs/feature-a.md').read_bytes() for c in (a, b)]
    if texts[0] != texts[1]:
        missing.append('byte-identical resulting repair in both runs')
    states = [m['bindings'][name]['workspace'] for m in manifests]
    if states[0]['expected_text'].encode() != texts[0]:
        missing.append('the resulting repair is the proposed patch')
    if states[1]['preserved'] and any(digest(b / 'candidate' / n) != d for n, d in states[1]['preserved'].items()):
        missing.append('unrelated candidate files preserved')
    return {'status': 'incomplete' if missing else 'observed', 'missing': missing,
            'resulting_sha256': digest(a / 'candidate/docs/feature-a.md'), 'preserved': list(states[1]['preserved'])}


def effort_pair(root, a, b, allowed):
    """Row 22: two repair-then-review cases differ only in the named input files."""
    differing = differing_inputs(root / a, root / b)
    missing = [] if differing == sorted(allowed) else [f'inputs differing {differing}, expected {sorted(allowed)}']
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
