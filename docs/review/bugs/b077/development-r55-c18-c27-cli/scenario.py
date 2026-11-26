"""C18/C27 disposable CLI replay against the supplied B077 candidate binary."""
import json
import os
import re
from pathlib import Path
import sqlite3
import subprocess
import sys

project = Path('/tmp/b077-c18-c27-proof/project')
project.mkdir(parents=True, exist_ok=True)
binary = Path(os.environ['B077_SHARK']).resolve(strict=True)
(project / 'shark').symlink_to(binary)
subprocess.run(['git', 'init', '-q'], cwd=project, check=True)
log = Path('/tmp/b077-c18-c27-proof/commands.jsonl')
db_path = project / 'shark-tasks.db'

def call(*args, expected=0):
    result = subprocess.run(['./shark', *args], cwd=project, text=True, capture_output=True)
    row = {'argv': list(args), 'exit_code': result.returncode, 'stdout': result.stdout, 'stderr': result.stderr}
    with log.open('a') as f:
        f.write(json.dumps(row) + '\n')
    if (result.returncode == 0) != (expected == 0):
        raise AssertionError(f'command {args} exit={result.returncode}; stderr={result.stderr}; stdout={result.stdout}')
    return result

def snapshot(key):
    with sqlite3.connect(f'file:{db_path}?mode=ro', uri=True) as db:
        row = db.execute('SELECT id,status,context_data FROM questions WHERE key=?', (key,)).fetchone()
        if row is None:
            raise AssertionError(f'{key} missing')
        qid, status, context = row
        notes = db.execute("SELECT count(*) FROM entity_notes WHERE entity_type='question' AND entity_id=?", (qid,)).fetchone()[0]
        history = db.execute("SELECT count(*) FROM entity_history WHERE entity_type='question' AND entity_id=?", (qid,)).fetchone()[0]
        return {'status': status, 'context_data': context, 'notes': notes, 'history': history}

def create(title):
    created = call('--json', 'question', 'create', title, '--summary', 'C18/C27 disposable session authority case', '--requester', 'asker').stdout
    match = re.search(r'\"key\"\s*:\s*\"(Q[0-9]+)\"', created)
    if not match:
        raise AssertionError(f'question create output omitted key: {created}')
    key = match.group(1)
    call('question', 'configure-workflow', key, '--resolution-owner', 'owner', '--responder', 'alice')
    return key

call('admin', 'init', '--non-interactive')
call('admin', 'install-shark-data')

# C18: prior responder session cannot write after same-responder lease takeover.
q1 = create('C18 stale response session')
call('claim', q1, '--by', 'alice', '--session', 'c18-old')
call('claim', q1, '--by', 'alice', '--session', 'c18-current', '--force')
before = snapshot(q1)
call('question', 'respond', q1, '--session', 'c18-old', '--responder', 'alice', '--summary', 'stale answer', '--evidence-pointer', 'notes/stale.md', expected=1)
if snapshot(q1) != before:
    raise AssertionError('C18 stale response changed Question state, notes, or history')
call('question', 'respond', q1, '--session', 'c18-current', '--responder', 'alice', '--summary', 'accepted answer', '--evidence-pointer', 'notes/accepted.md')
answered = snapshot(q1)
if answered['status'] != 'ready_for_resolution' or 'c18-current' not in (answered['context_data'] or ''):
    raise AssertionError(f'C18 matching response not persisted: {answered}')

# C27: resolution accepts only the live session that recorded the final response.
call('question', 'resolve', q1, '--resolution-owner', 'owner', '--resolution-kind', 'no_lasting_consequence', '--session', 'c18-old', expected=1)
if snapshot(q1) != answered:
    raise AssertionError('C27 wrong-session resolution changed state, notes, or history')
call('question', 'resolve', q1, '--resolution-owner', 'owner', '--resolution-kind', 'no_lasting_consequence', '--session', 'c18-current')
resolved = snapshot(q1)
if resolved['status'] != 'resolved' or resolved['history'] != answered['history'] + 1:
    raise AssertionError(f'C27 current final session did not resolve exactly once: {resolved}')

# C27 write-boundary guard: a takeover between response and resolution rejects both
# the successor session and the now-stale original, with no Question mutation.
q2 = create('C27 takeover after final response')
call('claim', q2, '--by', 'alice', '--session', 'c27-final')
call('question', 'respond', q2, '--session', 'c27-final', '--responder', 'alice', '--summary', 'final answer', '--evidence-pointer', 'notes/final.md')
call('claim', q2, '--by', 'alice', '--session', 'c27-successor', '--force')
pre_resolve = snapshot(q2)
call('question', 'resolve', q2, '--resolution-owner', 'owner', '--resolution-kind', 'no_lasting_consequence', '--session', 'c27-successor', expected=1)
if snapshot(q2) != pre_resolve:
    raise AssertionError('C27 successor session resolved predecessor answer')
call('question', 'resolve', q2, '--resolution-owner', 'owner', '--resolution-kind', 'no_lasting_consequence', '--session', 'c27-final', expected=1)
if snapshot(q2) != pre_resolve:
    raise AssertionError('C27 stale final session wrote after lease takeover')

print(json.dumps({'candidate_binary': str(binary), 'scenario': 'B077 C18/C27 session-bound Question writes',
                  'evidence': 'real Shark CLI commands; independent read-only SQLite snapshots',
                  'C18': {'question': q1, 'stale_session_refused_without_write': True,
                          'current_session_response_committed': True},
                  'C27': {'current_final_session_resolved': True, 'wrong_session_refused_without_write': True,
                          'successor_and_stale_sessions_refused_after_takeover': True, 'question': q2},
                  'command_log': str(log)}, indent=2))
