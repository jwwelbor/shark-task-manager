"""Check a retained grant-pair input against its original Git objects, without running a reviewer."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]

sys.path.insert(0, str(ROOT / 'skills/shark-rider/skills/deep-review/scripts'))
import adaptive_review as review


def read_input(report, report_hash, repository, commit, require_actionable=True):
    artifacts = {'review.md': review.read_review_file(report, review.MAX_ARTIFACT_BYTES)}
    if hashlib.sha256(artifacts['review.md']).hexdigest() != report_hash:
        raise ValueError('retained report digest differs')
    header, body = artifacts['review.md'].decode().split('\n\n---\n\n', 1)
    verdict = 'PASS-with-triage'
    if (f'**Verdict:** {verdict}'.lower() != header.splitlines()[-1].lower()
            or not review.valid_passing_report(body, verdict)):
        raise ValueError('grant pair requires a complete PASS-with-triage report')

    def sidecar(suffix, label):
        data = review.read_review_file(str(report) + suffix, review.MAX_ARTIFACT_BYTES)
        if f'**{label} SHA-256:** `{hashlib.sha256(data).hexdigest()}`' not in header:
            raise ValueError(f'report does not bind the supplied {label}')
        artifacts['review.md' + suffix] = data
        return data

    scope = json.loads(sidecar('.scope.json', 'Retained scope'))
    decisions = json.loads(sidecar('.decisions.json', 'Review decisions'))
    if not re.fullmatch(r'[0-9a-f]{40}|[0-9a-f]{64}', commit) or scope['candidate_commit'] != commit:
        raise ValueError('scope identifies a different replay candidate')
    base = scope['base_commit']
    review._validate_scope_identity(scope)
    review._scope_git(repository, 'merge-base', '--is-ancestor', base, commit)
    tree = review._scope_git(repository, 'rev-parse', commit + '^{tree}').decode().strip()
    if tree != scope['candidate_tree']:
        raise ValueError('candidate tree differs from retained scope')
    artifacts['review.md.diff'] = review._validate_scope_diff(
        scope, base, str(report) + '.diff', repository, commit)
    identity_line = (f'**Candidate:** `{commit}` · **Tree:** `{tree}` · '
                     f'**Diff SHA-256:** `{scope["diff_sha256"]}`')
    if identity_line not in header.splitlines():
        raise ValueError('report identifies a different replay candidate')
    metadata = re.search(r'^\*\*Runner metadata:\*\* `runner_mode=([a-z-]+)` '
                         r'`specialists_completed=(\d+)` `consolidator_completed=True`$',
                         header, re.MULTILINE | re.IGNORECASE)
    if not metadata:
        raise ValueError('missing complete runner metadata')
    mode, count = metadata.groups()
    if mode in {'canonical-six-angle', 'dispatched-six-angle'} and count == '6':
        raw = review.validate_six_angles(sidecar('.angles.json', 'Angle results'), scope)
    elif mode == 'adversarial-cli' and count == '0':
        raw = []
    else:
        raise ValueError('runner did not complete the review')
    review.validate_review_decisions(decisions, scope, raw, verdict)
    review._require_finding_decision(body, decisions)
    actionable = any(f['disposition'] == 'triage' and f['severity'] == 'non-blocker'
                     for f in decisions['findings'])
    report_only = (len(decisions['findings']) == 1
                   and decisions['findings'][0]['severity'] == 'nit'
                   and decisions['findings'][0]['disposition'] == 'no-action'
                   and raw and any(f['severity'] == 'blocker' for f in raw))
    if (require_actionable and not actionable) or (not require_actionable and not report_only):
        raise ValueError('grant pair requires a surviving actionable finding')
    receipt = {'report_sha256': report_hash, 'replay_candidate': commit, 'replay_tree': tree,
               'replay_diff_sha256': scope['diff_sha256'], 'runner_mode': mode,
               'files': {name: hashlib.sha256(data).hexdigest() for name, data in artifacts.items()}}
    return artifacts, receipt


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('report', type=Path)
    parser.add_argument('report_sha256')
    parser.add_argument('original_repository', type=Path)
    parser.add_argument('candidate_commit')
    args = parser.parse_args()
    try:
        _, receipt = read_input(args.report, args.report_sha256, args.original_repository, args.candidate_commit)
    except (KeyError, ValueError, OSError, TypeError, subprocess.SubprocessError) as error:
        parser.exit(1, f'B086 packet refused: {error}\n')
    print(json.dumps(receipt, indent=2))
