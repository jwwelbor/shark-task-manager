"""Read the review bytes and the applied operation that will be delivered together."""

import hashlib
import json
from pathlib import Path
import re
import subprocess

MEMBERS = {'bug_review_report': 'review.md', 'bug_review_scope': 'review.md.scope.json',
           'bug_review_decisions': 'review.md.decisions.json', 'bug_review_angles': 'review.md.angles.json'}
DISCLOSURE = 'The six angle inputs and the consolidation decision are synthetic.'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def applied_review(history, binary):
    project = Path(history['project'])
    entity, run_id, operation = (history[key] for key in ('entity', 'run_id', 'operation_digest'))
    if not re.fullmatch(r'fixture-code_review-[0-9a-f]{32}', run_id):
        raise ValueError('invalid scratch code-review run')
    rows = json.loads(subprocess.check_output([
        binary, 'status', 'history', entity, '--json', '--config', project / '.sharkconfig.json',
        '--db', project / 'shark-tasks.db'], cwd=project))
    reason = f'gate code_review outcome pass run {run_id} operation {operation}'
    code_reviews = [row for row in rows['history'] if row['old_status'] == 'code_review']
    if (rows['entity_key'] != entity or not code_reviews or code_reviews[0]['notes'] != reason
            or code_reviews[0]['new_status'] != 'qa'):
        raise ValueError('missing applied code-review transition')
    run = project / '.shark/runs' / run_id
    state = json.loads((run / 'operation-state.json').read_bytes())
    identity = json.loads((run / 'identity.json').read_bytes())
    expected = dict(entity_key=entity, entity_type='bug', gate='code_review',
                    source_status='code_review', operation_digest=operation, run_id=run_id)
    if any(record.get(key) != value for record in [state, identity] for key, value in expected.items()):
        raise ValueError('applied review identity differs')
    if state['persistence_state'] != 'transition_applied' or state['admission_version'] != 1:
        raise ValueError('review was not applied')
    result = json.loads((run / 'result.json').read_bytes())
    bound = dict(entity_key=entity, entity_type='bug', source_status='code_review', gate='code_review', envelope=result)
    if sha(json.dumps(bound, separators=(',', ':'), sort_keys=True, ensure_ascii=False).encode()) != operation:
        raise ValueError('applied review envelope differs')
    return result


def consume(packet, history, pin, binary):
    manifest = (packet / 'manifest.json').read_bytes()
    if sha(manifest) != pin['bundle_sha256']:
        raise ValueError('intake bundle differs from frozen review')
    if history != pin['history']:
        raise ValueError('intake history differs from pinned applied operation')
    members = json.loads(manifest)
    full = {*MEMBERS.values(), 'review.md.diff', 'provenance.json'}
    if set(members) not in (full, full - {'review.md.angles.json'}):
        raise ValueError('incomplete review manifest')
    payload = {}
    for name, expected in members.items():
        data = (packet / name).read_bytes()
        if sha(data) != expected:
            raise ValueError(f'intake member differs: {name}')
        payload[name] = data.decode()
    result = applied_review(history, binary)
    for kind, name in MEMBERS.items():
        if name not in members:
            continue
        expected = 'sha256:' + members[name]
        if [ref['summary'] for ref in result['evidence'] if ref['kind'] == kind] != [expected]:
            raise ValueError(f'applied review differs: {name}')
    check_origin(payload, result, Path(history['project']))
    return dict(payload, history=history, manifest=manifest.decode())


def check_origin(payload, result, project):
    report = payload['review.md']
    header, body = report.split('\n\n---\n\n', 1)
    if DISCLOSURE not in body or not ('six supplied angle records' in body or 'no specialist records' in body):
        raise ValueError('report lacks synthetic runner disclosure')
    provenance = json.loads(payload['provenance.json'])
    if provenance.get('input_origin') != DISCLOSURE:
        raise ValueError('provenance lacks synthetic input disclosure')
    if [ref['summary'] for ref in result['evidence'] if ref['kind'] == 'fixture_origin'] != [DISCLOSURE]:
        raise ValueError('applied evidence lacks synthetic input disclosure')
    trace = (project / 'fixture-inputs/writer.trace').read_bytes()
    if sha(trace) != provenance['writer_trace_sha256']:
        raise ValueError('writer trace differs')
    invocations = re.findall(r'execve\("([^"]+)",', trace.decode())
    if not invocations or any(Path(path).name not in {'git', 'python3', 'python'} for path in invocations):
        raise ValueError('unexpected executable in writer trace')
    if b'sa_family=AF_INET' in trace or b'"write-report"' not in trace:
        raise ValueError('writer trace does not establish offline report generation')
    setup = (project / 'fixture-inputs/setup.trace').read_bytes()
    if sha(setup) != provenance['setup_trace_sha256']:
        raise ValueError('setup trace differs')
    programs = re.findall(r'execve\("([^"]+)",', setup.decode())
    if b'sa_family=AF_INET' in setup or any(Path(path).name in {'claude', 'codex'} for path in programs):
        raise ValueError('reviewer or provider call in setup')
    if not programs or b'"--apply-result"' not in setup or b'get_diff.sh"' not in setup:
        raise ValueError('missing scope capture or applied gate trace')
    models = re.findall(r'\*\*Verified effective model:\*\* [\x60]([^\x60]*)[\x60]', header)
    if models != ['unknown']:
        raise ValueError('effective model has no observed invocation')
    decisions = json.loads(payload['review.md.decisions.json'])['findings']
    expected = ({'severity': 'nit', 'disposition': 'no-action', 'file': 'docs/feature-a.md',
                 'fingerprint': 'docs/feature-a.md#DOCS#section-title',
                 'summary': 'Prefer a longer section title',
                 'reason': 'The current title names Feature A, and no project standard requires a longer title.'}
                if provenance.get('scenario') == 'stylistic downgrade report-only'
                else {'severity': 'non-blocker', 'disposition': 'triage', 'file': 'docs/feature-a.md',
                      'fingerprint': 'docs/feature-a.md#DOCS#worked-example',
                      'summary': 'Add a worked example under Feature A',
                      'reason': 'Feature A already documents the required command and default. A second worked example can wait.'})
    if len(decisions) != 1 or any(decisions[0].get(key) != value for key, value in expected.items()):
        raise ValueError('review does not supply the optional Feature A worked example')


def received_pin(events, run_id, pin):
    if not pin:
        return False
    sequence = [event for event in events if event.get('run_id') == run_id]
    intake = [(index, event) for index, event in enumerate(sequence) if event.get('tool') == 'intake']
    if len(intake) != 2:
        return False
    (_, entry), (end, delivery) = intake
    return (entry.get('event') == 'entry' and delivery.get('event') == 'result'
            and entry.get('invocation') and delivery.get('invocation') == entry['invocation']
            and delivery.get('outcome') == 'delivered' and delivery.get('intake') == pin
            and not any(event.get('tool') == 'shark' for event in sequence[:end]))
