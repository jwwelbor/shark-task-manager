"""Prepare the optional Feature A example through the production writer and scratch gates."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import uuid

from intake import DISCLOSURE, MEMBERS, consume
from prepare import BASELINE, POLICY, digest

PLAN = 'docs/plan/bugs/B001.repair-plan.md'
APPROVAL = 'docs/plan/bugs/B001.plan-review.md'
FILE = 'docs/feature-a.md'
FINGERPRINT = FILE + '#DOCS#worked-example'
WHY = 'Feature A already documents the required command and default. A second worked example can wait.'


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')


def execute(args, cwd, env=None):
    trace = cwd / 'fixture-inputs/setup.trace.active'
    command = args
    if args[0] != 'strace':
        command = ['strace', '-f', '-qq', '-A', '-s', '256', '-e', 'trace=execve,connect', '-o', trace, *args]
    proc = subprocess.run(command, cwd=cwd, capture_output=True, text=True, env=env)
    if proc.returncode:
        raise ValueError(f'{args[0]} {args[1]} failed: {proc.stderr.strip()} {proc.stdout.strip()}')
    return proc.stdout


def git_fixture(project, variant):
    env = dict(os.environ, GIT_AUTHOR_NAME='B086 fixture', GIT_COMMITTER_NAME='B086 fixture',
               GIT_AUTHOR_EMAIL='fixture@example.invalid', GIT_COMMITTER_EMAIL='fixture@example.invalid',
               GIT_AUTHOR_DATE='2001-01-01T00:00:00Z', GIT_COMMITTER_DATE='2001-01-01T00:00:00Z',
               GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1')

    def git(*args):
        return execute(['git', '-c', 'core.hooksPath=/dev/null', *args], project, env).strip()

    git('init', '-b', 'fixture', '--object-format=sha1')
    (project / '.git/info/exclude').write_text('/*\n!/docs/\n/docs/plan/*\n!/docs/plan/bugs/\n/docs/plan/bugs/*\n')
    (project / FILE).parent.mkdir(exist_ok=True)
    (project / FILE).write_text('# Feature A\n\nRun feature-a list to list entries.\n')
    git('add', FILE)
    git('commit', '-m', 'Document the Feature A command')
    base = git('rev-parse', 'HEAD')
    (project / PLAN).parent.mkdir(parents=True, exist_ok=True)
    (project / PLAN).write_text(f'---\nreview_baseline: {base}\n---\n\nSynthetic fixture plan. Document the default order.\n')
    with (project / '.git/info/exclude').open('a') as exclude:
        exclude.write('!/docs/plan/bugs/B001.repair-plan.md\n')
    with (project / FILE).open('a') as doc:
        if variant == 'fix-now':
            doc.write('The default order is random.\n')
        else:
            doc.write('The default order is alphabetical.\n')
        if variant == 'maintenance':
            doc.write('By default the order is alphabetical.\n')
    git('add', FILE, PLAN)
    git('commit', '-m', 'Document the Feature A default')
    return base


def writer(project, bundle, scripts, base, variant):
    inputs = project / 'fixture-inputs'
    env = dict(os.environ, REVIEW_BASE=base, REVIEW_PLAN_SHA256=digest(project / PLAN))
    scope = json.loads(execute([scripts / 'get_diff.sh'], project, env))
    write_json(inputs / 'scope.json', scope)
    identity = {'commit': scope['candidate_commit'], 'tree': scope['candidate_tree'],
                'diff_sha256': scope['diff_sha256'], 'repair_plan_sha256': scope['repair_plan_sha256']}
    finding = dict(file=FILE, line=4, rule='DOCS', summary='Add a worked example under Feature A',
                   severity='non-blocker', diagnosis='A second example would help users apply the default order.',
                   fingerprint=FINGERPRINT)
    if variant == 'maintenance':
        finding.update(summary='Consolidate duplicate default descriptions', diagnosis='Duplicate instructions increase maintenance cost.',
                       fingerprint=FILE + '#DOCS#duplicate-default')
    elif variant == 'style':
        finding.update(summary='Prefer a longer section title', severity='nit', diagnosis='A stylistic preference only.',
                       fingerprint=FILE + '#DOCS#section-title')
    elif variant == 'fix-now':
        finding.update(summary='Correct the documented default order', diagnosis='The plan requires alphabetical order.',
                       fingerprint=FILE + '#DOCS#incorrect-default')
    decision = dict(finding, disposition='triage', reason=WHY)
    if variant == 'maintenance':
        decision['reason'] = 'The documented behavior is correct despite the duplication.'
    elif variant == 'style':
        decision.update(disposition='no-action', reason='The title already names the feature.')
    elif variant == 'fix-now':
        decision.update(disposition='fix-now', reason='The default is part of the accepted behavior.')
    write_json(inputs / 'angles.json', {angle: dict(candidate_identity=identity, reviewed_files=scope['changed_files'],
                                                     findings=[finding] if angle == 'a' else [])
                                        for angle in 'abcdef'})
    write_json(inputs / 'decisions.json', dict(candidate_identity=identity, findings=[decision]))
    body = f'VERDICT: PASS-with-triage\nScope: Feature A, E01-F01. Findings: {finding["summary"]}. {decision["reason"]}\n{finding["fingerprint"]}\n'
    if variant != 'report':
        body += DISCLOSURE + ' The runner count represents six supplied angle records.\n'
    (inputs / 'body.md').write_text(body)
    args = [sys.executable, scripts / 'adaptive_review.py', 'write-report', '--project-root', project,
            '--source-root', project, '--branch', scope['branch'], '--review-output-path', bundle / 'review.md',
            '--scope-file', inputs / 'scope.json', '--angle-results', inputs / 'angles.json',
            '--review-decisions', inputs / 'decisions.json', '--body-file', inputs / 'body.md',
            '--diff-path', scope['diff_path'], '--base-commit', base, '--verdict', 'PASS-with-triage',
            '--runner-mode', 'canonical-six-angle', '--specialists-completed', '6', '--consolidator-completed', 'true']
    if variant == 'model':
        args.extend(['--effective-model', 'unobserved-fixture-model'])
    trace = inputs / 'writer.trace'
    execute(['strace', '-f', '-qq', '-s', '256', '-e', 'trace=execve,connect', '-o', trace, *args], project)
    provenance = {'scenario': 'optional worked example under Feature A', 'policy_source': POLICY,
                  'b086_baseline': BASELINE, 'b086_plan_sha256': digest(Path(__file__).resolve().parents[3] /
                                                                       'docs/plan/bugs/B086.repair-plan.md'),
                  'writer_trace_sha256': digest(trace), 'candidate': scope['candidate_commit']}
    if variant != 'provenance':
        provenance['input_origin'] = DISCLOSURE
    write_json(bundle / 'provenance.json', provenance)
    return scope


def applied_history(project, binary, bundle, base, variant, expected_entity="B001"):
    def shark(*args):
        return json.loads(execute([binary, *args, '--db', project / 'shark-tasks.db',
                                   '--config', project / '.sharkconfig.json', '--json'], project))
    entity = shark('bug', 'create', 'Synthetic Feature A documentation review')['key']
    if entity != expected_entity:
        raise ValueError('unexpected scratch entity')

    def evidence(kind, path):
        return dict(kind=kind, pointer=str(path.relative_to(project)), summary='sha256:' + digest(path))

    plan = evidence('bug_repair_plan', project / PLAN)
    (project / APPROVAL).write_text(f'plan_path: {PLAN}\nplan_sha256: {plan["summary"]}\nSynthetic independent-review input.\n')
    for label, path in [('Fixture plan', PLAN), ('Fixture independent review', APPROVAL)]:
        shark('related-docs', 'add', label, path, '--bug', entity)
    shark('status', 'set', entity, 'plan_review', '--force', '--reason', 'Synthetic scenario setup')
    permanent = [plan, dict(kind='bug_review_baseline', pointer=base, summary='Fixture Git baseline')]
    files = [evidence(kind, bundle / name) for kind, name in MEMBERS.items()]
    session = shark('claim', entity, '--by', 'b086-scratch')['session_id']
    for stage, extra in [('plan_review', [evidence('plan_review_report', project / APPROVAL)]),
                         ('development', []), ('code_review', files), ('qa', files[:2])]:
        run_id = 'fixture-' + stage + '-' + uuid.uuid4().hex
        origin = [] if variant == 'operation' else [dict(kind='fixture_origin', pointer='fixture-inputs', summary=DISCLOSURE)]
        envelope = dict(kind='final', recommended_outcome='pass', evidence=permanent + extra + origin,
                        gate_result=dict(schema_version=1, summary=f'Synthetic fixture {stage} result.', findings=[], kickbacks=[]))
        path = project / 'fixture-inputs' / (stage + '.json')
        write_json(path, envelope)
        shark('run', entity, '--apply-result', path, '--workdir', project, '--session', session, '--run-id', run_id)
        state = json.loads((project / '.shark/runs' / run_id / 'operation-state.json').read_bytes())
        if state['gate'] != stage or state['persistence_state'] != 'transition_applied':
            raise ValueError(f'{stage} was not applied')
        if stage == 'code_review':
            history = dict(project=str(project), entity=entity, run_id=run_id, operation_digest=state['operation_digest'])
    return history


def generate(project, seed, source, variant=None):
    from packet import read_input
    shutil.copytree(seed, project)
    binary = source / 'bin/shark'
    bundle = project / 'fixture-review'
    bundle.mkdir()
    (project / 'fixture-inputs').mkdir()
    base = git_fixture(project, variant)
    scope = writer(project, bundle, source / 'skills/shark-rider/skills/deep-review/scripts', base, variant)
    history = applied_history(project, binary, bundle, base, variant)
    setup_trace = project / 'fixture-inputs/setup.trace'
    setup_trace.with_suffix('.trace.active').rename(setup_trace)
    provenance = json.loads((bundle / 'provenance.json').read_bytes())
    provenance['setup_trace_sha256'] = digest(setup_trace)
    write_json(bundle / 'provenance.json', provenance)
    artifacts, packet = read_input(bundle / 'review.md', digest(bundle / 'review.md'), project, scope['candidate_commit'])
    artifacts['provenance.json'] = (bundle / 'provenance.json').read_bytes()
    names = {name: digest(bundle / name) for name in artifacts}
    write_json(bundle / 'manifest.json', names)
    artifacts['manifest.json'] = (bundle / 'manifest.json').read_bytes()
    pin = dict(bundle_sha256=digest(bundle / 'manifest.json'), history=history)
    consume(bundle, history, pin, binary)
    return artifacts, packet, pin
