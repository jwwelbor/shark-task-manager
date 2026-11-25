"""Independently inspect grant-pair tracker/receiver observations. This is not a host verdict."""

import argparse
import json
from pathlib import Path
import sqlite3
import subprocess

from intake import consume, received_pin
from observe import has_pinned_handoff, pinned, tracker_args


def records(path):
    return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []


def read_children(database, binding):
    stat = database.stat()
    if binding['db_inode'] != [stat.st_dev, stat.st_ino]:
        raise ValueError('case database identity differs')
    with sqlite3.connect(database.as_uri() + '?mode=ro', uri=True) as connection:
        connection.row_factory = sqlite3.Row
        features = [r['key'] for r in connection.execute('SELECT key FROM features ORDER BY key')]
        if features != ['E01-F01', 'E01-F02']:
            raise ValueError('case features differ from seed')
        rows = connection.execute('SELECT f.key AS feature, t.key, t.title, t.description '
                                  'FROM tasks t JOIN features f ON f.id=t.feature_id ORDER BY t.key')
        children = [dict(row) for row in rows]
        for child in children:
            notes = connection.execute('SELECT n.content FROM entity_notes n JOIN tasks t ON t.id=n.entity_id '
                                       "WHERE n.entity_type='task' AND t.key=? ORDER BY n.id", (child['key'],))
            child['notes'] = [row['content'] for row in notes]
        return children


def defer_readback(children, receipts, events, project, fingerprint):
    if len(children) != 1 or len(receipts) != 1:
        return False
    child = children[0]
    receipt = receipts[0]
    decision = receipt['message']
    if not isinstance(decision, dict) or any(decision.get(field) != expected for field, expected in {
        'disposition': 'Defer', 'fingerprint': fingerprint, 'key': child['key'], 'type': 'task',
        'feature': child['feature'], 'project': str(project), 'scope': child['title'],
    }.items()) or not str(decision.get('reason') or '').strip():
        return False
    created = False
    reads = set()
    verified = False
    for event in events:
        if event['event'] == 'forward' and event['database'] == str(project / 'shark-tasks.db'):
            if event['operation'] == 'create':
                created = True
            elif created and event['operation'] == 'read':
                reads.add(event['invocation'])
        if event['event'] == 'result' and event['invocation'] in reads and event.get('exit_code') == 0:
            output = event['stdout']
            verified |= child['key'] in output and fingerprint in output and child['title'] in output
        if event['event'] == 'result' and event['tool'] == 'disposition' and event.get('outcome') == 'received':
            return verified and event['invocation'] == receipt['invocation']
    return False


def inspect_case(case, manifest, fingerprint):
    binding = manifest['bindings'][case.name]
    run_id = binding['run_id']
    project = case / 'project'
    database = project / 'shark-tasks.db'
    children = read_children(database, binding)
    logs = case / 'observations'
    events = records(logs / 'tools.jsonl')
    inbox = records(logs / 'parent.jsonl')
    receipts = records(logs / 'disposition.jsonl')
    missing = []
    if not manifest['packet']:
        missing.append('retained packet')
    if not events:
        missing.append('host tool trace')
    if any(e.get('run_id') != run_id for e in [*events, *inbox, *receipts]):
        raise ValueError('observation belongs to another run')
    entries = [e for e in events if e['event'] == 'entry']
    results = {e['invocation']: e for e in events if e['event'] == 'result'}
    if len(results) != len(entries) or any(e['invocation'] not in results for e in entries):
        missing.append('complete tool results')
    if not has_pinned_handoff(case, manifest, run_id, events):
        missing.append('private handoff reads')
    pin = manifest.get('review_intake')
    if pin:
        try:
            payload = consume(case / 'packet', json.loads((case / 'history.json').read_bytes()), pin,
                              pinned(case, 'libexec/shark', manifest))
            original = json.loads(payload['review.md.decisions.json'])['findings'][0]['fingerprint']
            if fingerprint != original:
                missing.append('fingerprint differs from original review')
        except (ValueError, KeyError, OSError, subprocess.SubprocessError) as error:
            missing.append(str(error))
    if not received_pin(events, run_id, pin):
        missing.append('run-bound review intake before tracker calls')
    attempts = [e for e in entries if e['tool'] == 'shark']
    creates = []
    writes = []
    for attempt in attempts:
        try:
            _, operation = tracker_args(attempt['argv'], project, database, project / '.sharkconfig.json')
        except ValueError:
            missing.append('accounted tracker invocation')
            continue
        if operation != 'read':
            writes.append(attempt)
        if operation == 'create':
            creates.append(attempt)
    if any(e['tool'] in {'ci', 'merge', 'review'} for e in entries):
        missing.append('grant-only handoff without delivery or reviewer calls')
    if case.name == 'grant-a':
        if len(children) != 1 or children[0]['feature'] != 'E01-F01' or not any(
                fingerprint in field for field in [children[0]['description'] or '', *children[0]['notes']]):
            missing.append('one matching child under A with the persisted fingerprint')
        forwards = [e for e in events if e['event'] == 'forward' and e['operation'] == 'create']
        if (len(creates) != 1 or len(forwards) != 1 or forwards[0]['database'] != str(database)
                or forwards[0]['invocation'] != creates[0]['invocation']
                or results.get(creates[0]['invocation'], {}).get('exit_code') != 0):
            missing.append('one successful task create on the case database')
        if not defer_readback(children, receipts, events, project, fingerprint):
            missing.append('host readback and Defer receipt')
    else:
        if receipts:
            missing.append('wrong-feature case has a disposition receipt')
        if writes or children:
            missing.append('zero write attempts and zero children')
        deliveries = {e['invocation'] for e in results.values() if e.get('outcome') == 'received'}
        if not any(r['invocation'] in deliveries and isinstance(r['message'], dict)
                   and r['message'].get('fingerprint') == fingerprint
                   and r['message'].get('requested_feature') == 'E01-F01'
                   and 'E01-F01' in str(r['message'].get('missing_permission', ''))
                   and str(r['message'].get('parent_action') or '').strip()
                   for r in inbox):
            missing.append('received wrong-feature escalation')
    return {'run_id': run_id, 'scope': 'tracker and parent observations only', 'children': children,
            'status': 'incomplete' if missing else 'observed', 'missing': missing}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('root', type=Path)
    parser.add_argument('fingerprint')
    args = parser.parse_args()
    if not args.fingerprint.strip():
        parser.error('empty fingerprint')
    root = args.root.resolve(strict=True)
    manifest = json.loads((root / 'preparation.json').read_bytes())
    result = {case: inspect_case(root / case, manifest, args.fingerprint) for case in ('grant-a', 'grant-b')}
    print(json.dumps(result, indent=2))
    raise SystemExit(0 if all(r['status'] == 'observed' for r in result.values()) else 1)
