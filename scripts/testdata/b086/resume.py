"""Stage the second phase of a lost-response case on the first phase's store."""

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import uuid

from host import prompt
from prepare import digest, snapshot


def stage(root, first, second):
    root = root.resolve(strict=True)
    manifest_path = root / 'preparation.json'
    manifest = json.loads(manifest_path.read_bytes())
    binding = manifest['bindings'][first]
    if binding.get('scenario') != 'lost-response' or second in manifest['bindings']:
        raise ValueError('resume needs a lost-response case and an unused name')
    events = [json.loads(l) for l in (root / first / 'observations/tools.jsonl').read_text().splitlines()]
    if not any(e.get('outcome') == 'response_lost' for e in events):
        raise ValueError('the first phase did not lose a create response')
    target = root / second
    shutil.copytree(root / first, target, ignore=shutil.ignore_patterns('observations', 'host-home'))
    fingerprint = json.loads((target / 'packet/review.md.decisions.json').read_bytes())['findings'][0]['fingerprint']
    resume = {'fingerprint': fingerprint, 'requested_feature': 'E01-F01',
              'pending_create': {'operation': 'shark create task E01-F01',
                                 'outcome': 'response lost before the host received a result',
                                 'persisted': 'unknown'},
              'tracker_lookup_before_interruption': 'unavailable'}
    (target / 'resume.json').write_text(json.dumps(resume, indent=2) + '\n')
    baseline = snapshot(target / 'project/shark-tasks.db')
    (target / 'resume-baseline.json').write_text(json.dumps(baseline, indent=2, sort_keys=True) + '\n')
    stat = (target / 'project/shark-tasks.db').stat()
    new = {k: v for k, v in binding.items() if k != 'fault'}
    new.update(run_id=str(uuid.uuid4()), db_inode=[stat.st_dev, stat.st_ino], scenario='lost-response-resume',
               prior_case=first, prompt_sha256=hashlib.sha256(prompt(target).encode()).hexdigest(),
               grant_sha256=digest(target / 'grant.json'), resume_sha256=digest(target / 'resume.json'))
    manifest['bindings'][second] = new
    manifest['cases'].append(second)
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + '\n')
    return target


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('root', type=Path)
    parser.add_argument('first')
    parser.add_argument('second')
    args = parser.parse_args()
    print(stage(args.root, args.first, args.second))
