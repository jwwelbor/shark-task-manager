"""Stub subagent, rescue-CLI and report endpoints for the dispatched-runner cases (B086 rows 10 and 19).

The recorder serves staged subagent returns and counts every invocation; the host decides which runner to use,
what to dispatch and how to persist the result. Only `report` runs real code (the pinned production writer).
"""

import json
from pathlib import Path
import subprocess
import sys

ANGLES = {'angle-a-bugs': 'a', 'angle-b-behavior': 'b', 'angle-c-sibling': 'c',
          'angle-d-cleanup': 'd', 'angle-e-tests': 'e', 'angle-f-standards': 'f'}
SCRIPT = 'project/.agents/skills/shark-rider/skills/deep-review/scripts/adaptive_review.py'
REPORT_FLAGS = {'--verdict', '--runner-mode', '--specialists-completed', '--consolidator-completed',
                '--body-file', '--angle-results', '--review-decisions', '--adversarial-model',
                '--fallback-reason'}
FILE_FLAGS = {'--body-file', '--angle-results', '--review-decisions'}


def parse_json(text):
    try:
        json.loads(text)
        return True
    except ValueError:
        return False


def serve(case, name, stdin):
    """Return (text, detail) for one subagent dispatch; refuse an unknown agent."""
    if name in ANGLES:
        path = case / 'stub' / f'angle-{ANGLES[name]}.json'
        detail = {'agent': name, 'angle': ANGLES[name]}
    elif name == 'consolidator':
        path = case / 'stub/consolidator.json'
        detail = {'agent': name}
    else:
        raise ValueError(f'unknown subagent: {name}')
    scope = json.loads((case / 'scope/scope.json').read_bytes())
    text = path.read_text()
    prompt = stdin.decode('utf-8', 'replace')
    detail.update(prompt_bytes=len(stdin), names_candidate=scope['candidate_commit'] in prompt,
                  names_diff=scope['diff_sha256'] in prompt, served_valid_json=parse_json(text))
    return text, detail


def serve_cli(case):
    return (case / 'stub/cli.json').read_text()


def report(case, argv):
    """Run the pinned writer with the case's captured scope fixed; the host supplies only its own inputs."""
    scope = json.loads((case / 'scope/scope.json').read_bytes())
    allowed = []
    index = 0
    while index < len(argv):
        flag = argv[index]
        if flag not in REPORT_FLAGS or index + 1 >= len(argv):
            raise ValueError(f'report argument not supplied by the host: {flag}')
        value = argv[index + 1]
        if flag in FILE_FLAGS and not (case / value).resolve(strict=True).is_relative_to(case / 'work'):
            raise ValueError(f'{flag} must name a file under work/')
        allowed += [flag, str((case / value).resolve(strict=True)) if flag in FILE_FLAGS else value]
        index += 2
    output = case / 'observations/review/review.md'
    output.parent.mkdir(parents=True, exist_ok=True)
    repository = str(case / 'review-repo')
    command = [sys.executable, str(case / SCRIPT), 'write-report', '--project-root', repository,
               '--source-root', repository, '--branch', scope['branch'], '--scope-file',
               str(case / 'scope/scope.json'), '--diff-path', str(case / 'scope/review.diff'),
               '--base-commit', scope['base_commit'], '--review-output-path', str(output), *allowed]
    done = subprocess.run(command, capture_output=True, text=True, cwd=repository)
    return done, output


def read_metadata(report_path):
    """Runner metadata line of a persisted review, read independently of the host."""
    import re
    head = report_path.read_text().split('\n\n---\n\n', 1)[0]
    found = re.search(r'`runner_mode=([a-z-]+)` `specialists_completed=(\w+)` `consolidator_completed=(\w+)`',
                      head)
    verdict = re.search(r'\*\*Verdict:\*\* (\S+)', head)
    return {'runner_mode': found.group(1) if found else None,
            'specialists_completed': found.group(2) if found else None,
            'consolidator_completed': (found.group(3).lower() == 'true') if found else None,
            'verdict': verdict.group(1) if verdict else None}
