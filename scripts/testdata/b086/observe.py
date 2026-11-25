"""Tool-entry recorders for the first grant pair. The host owns every disposition."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys
import uuid

from prepare import digest
from intake import consume, received_pin

TRIAGE = 'catalog/shark-rider/skills/triage/SKILL.md'
REVIEW = 'catalog/shark-rider/skills/deep-review/SKILL.md'
CONSOLIDATOR = 'catalog/shark-rider/skills/deep-review/references/consolidator.md'
WORKFLOW = 'embedded/quality/workflows/review-code.md'
HANDOFF = {TRIAGE, REVIEW, CONSOLIDATOR, WORKFLOW}
WORKFLOW_ARGS = ['skill', 'get', 'quality', 'workflows/review-code.md', '--raw']


def append(path, record):
    with path.open('a', encoding='utf-8') as stream:
        stream.write(json.dumps(record) + '\n')


def pinned(case, relative, manifest):
    path = (case / relative).resolve(strict=True)
    if not path.is_relative_to(case) or digest(path) != manifest['prepared_files'][relative]:
        raise ValueError(f'prepared file changed: {relative}')
    return path


def read_policy(case, relative, manifest):
    if relative == WORKFLOW:
        binary = pinned(case, 'libexec/shark', manifest)
        content = subprocess.check_output([binary, *WORKFLOW_ARGS], cwd=case / 'project')
        expected = manifest['embedded_workflow_sha256']
        origin = ' '.join([str(binary), *WORKFLOW_ARGS])
    elif relative.startswith('project/.agents/skills/'):
        path = (case / relative).resolve(strict=True)
        if not path.is_relative_to(case / 'project/.agents/skills'):
            raise ValueError('private skill path escapes the case catalog')
        tail = relative.removeprefix('project/.agents/skills/')
        if tail.startswith('shark-rider/'):
            catalog_relative = 'catalog/shark-rider/' + tail.removeprefix('shark-rider/')
        elif tail.startswith('finish-feature/'):
            catalog_relative = 'catalog/finish-feature/' + tail.removeprefix('finish-feature/')
        else:
            raise ValueError('skill is outside the private catalog')
        if catalog_relative not in manifest['prepared_files']:
            raise ValueError('private skill file is not pinned')
        content = path.read_bytes()
        expected = manifest['prepared_files'][relative]
        if expected != manifest['prepared_files'][catalog_relative]:
            raise ValueError('private skill digest differs from pinned catalog')
        origin = str(path)
        relative = catalog_relative
    else:
        if not relative.startswith('catalog/') or relative not in manifest['prepared_files']:
            raise ValueError('file is outside the private catalog')
        path = pinned(case, relative, manifest)
        origin, content, expected = str(path), path.read_bytes(), manifest['prepared_files'][relative]
    if hashlib.sha256(content).hexdigest() != expected:
        raise ValueError('loaded policy differs from prepared source')
    return content, {'file': relative, 'origin': origin, 'sha256': expected}


def has_pinned_handoff(case, manifest, run_id, events):
    expected = {}
    for name in HANDOFF - {WORKFLOW}:
        tail = name.removeprefix('catalog/')
        expected[name] = (str(case / 'project/.agents/skills' / tail),
                          manifest['prepared_files'][name])
    expected[WORKFLOW] = (' '.join([str(case / 'libexec/shark'), *WORKFLOW_ARGS]),
                          manifest['embedded_workflow_sha256'])
    loaded = set()
    for event in events:
        if (event.get('run_id') != run_id or event.get('tool') != 'read'
                or event.get('event') != 'result' or event.get('outcome') != 'loaded'):
            continue
        name = event.get('file')
        if name in expected:
            if (event.get('origin'), event.get('sha256')) != expected[name]:
                return False
            loaded.add(name)
    return loaded == HANDOFF


def tracker_args(argv, project, database, config):
    # The first pair needs ordinary reads and task creation only. Record every refused invocation.
    words = []
    forwarded = []
    values = {'--config': config, '--db': database}
    seen = set()
    args = iter(argv)
    for arg in args:
        name, equals, inline = arg.partition('=')
        if name in values or name in ('--description', '--title', '--field', '--priority', '--epic', '--feature', '--type', '--note-type'):
            value = inline if equals else next(args, None)
            if not value or value.startswith('-'):
                raise ValueError(f'missing {name} value')
            if name in values:
                if name in seen or (project / value).resolve() != values[name]:
                    raise ValueError(f'refusing {name} destination: {value}')
                seen.add(name)
            else:
                forwarded.extend([name, value])
        elif arg in ('--json', '--no-color'):
            forwarded.append(arg)
        elif arg.startswith('-'):
            raise ValueError(f'unsupported first-pair option: {arg}')
        else:
            words.append(arg)
            forwarded.append(arg)
    if not words:
        raise ValueError('missing tracker operation')
    create = words[:2] in (['task', 'create'], ['create', 'task'])
    read = words[0] in {'list', 'get', 'progress', 'history', 'search'} or words[:2] == ['notes', 'search']
    read = read or words[:2] == ['status', 'history']
    if len(words) > 1 and words[0] in {'bug', 'task', 'feature', 'epic', 'change', 'tech-debt', 'idea'}:
        read = read or words[1] in {'list', 'get'}
    association = words[:2] == ['notes', 'add'] or words[:3] == ['task', 'note', 'add']
    if not (create or read or association):
        raise ValueError(f'unsupported first-pair command: {words[:2]}')
    operation = 'create' if create else 'associate' if association else 'read'
    return [*forwarded, '--config', str(config), '--db', str(database)], operation


def tracker(case, manifest, binding, argv, log, event):
    project = case / 'project'
    config = pinned(case, 'project/.sharkconfig.json', manifest)
    pinned(case, 'project/shark', manifest)
    binary = pinned(case, 'libexec/shark', manifest)
    database = (project / 'shark-tasks.db').resolve(strict=True)
    stat = database.stat()
    if database.parent != project or [stat.st_dev, stat.st_ino] != binding['db_inode']:
        raise ValueError('database no longer matches prepared case')
    args, operation = tracker_args(argv, project, database, config)
    if not manifest['packet']:
        raise ValueError('missing retained packet')
    for name, expected in manifest['packet']['files'].items():
        path = (case / 'packet' / name).resolve(strict=True)
        if not path.is_relative_to(case / 'packet') or digest(path) != expected:
            raise ValueError('retained packet changed')
    loads = [json.loads(line) for line in log.read_text().splitlines()]
    if not has_pinned_handoff(case, manifest, binding['run_id'], loads):
        raise ValueError('private review handoff reads missing or inconsistent with pinned sources')
    pin = manifest.get('review_intake')
    if (pin or operation != 'read') and not received_pin(loads, binding['run_id'], pin):
        raise ValueError('verified review intake required before forwarding')
    for relative in HANDOFF - {WORKFLOW}:
        pinned(case, relative, manifest)
    append(log, dict(event, event='forward', database=str(database), operation=operation))
    try:
        result = subprocess.run([binary, *args], cwd=project, capture_output=True)
    except OSError as error:
        append(log, dict(event, event='result', outcome='launch_failed', reason=str(error)))
        return 1
    append(log, dict(event, event='result', outcome='exited', forwarded=True,
                     exit_code=result.returncode, stdout=result.stdout.decode(),
                     stderr=result.stderr.decode()))
    sys.stdout.buffer.write(result.stdout)
    sys.stderr.buffer.write(result.stderr)
    return result.returncode


def run(case, tool, argv):
    case = case.resolve(strict=True)
    manifest = json.loads((case.parent / 'preparation.json').read_bytes())
    binding = manifest['bindings'][case.name]
    observations = case / 'observations'
    observations.mkdir(exist_ok=True)
    log = observations / 'tools.jsonl'
    workflow_read = tool == 'shark' and argv == WORKFLOW_ARGS
    event = {'run_id': binding['run_id'], 'invocation': str(uuid.uuid4()),
             'tool': 'read' if workflow_read else tool}
    append(log, dict(event, event='entry', argv=argv))
    try:
        if workflow_read or (tool == 'read' and len(argv) == 1):
            content, detail = read_policy(case, WORKFLOW if workflow_read else argv[0], manifest)
            sys.stdout.buffer.write(content)
            sys.stdout.buffer.flush()
            append(log, dict(event, event='result', outcome='loaded', **detail))
        elif tool == 'intake' and len(argv) == 2:
            pin = manifest.get('review_intake')
            if not pin:
                raise ValueError('no independently pinned review intake')
            history = json.loads(Path(argv[1]).read_bytes())
            payload = consume(Path(argv[0]), history, pin, pinned(case, 'libexec/shark', manifest))
            print(json.dumps(payload), flush=True)
            observed = dict(bundle_sha256=hashlib.sha256(payload['manifest'].encode()).hexdigest(),
                            history=payload['history'])
            append(log, dict(event, event='result', outcome='delivered', intake=observed))
        elif tool in {'parent', 'disposition'}:
            message = json.load(sys.stdin)
            append(observations / (tool + '.jsonl'), dict(event, message=message))
            append(log, dict(event, event='result', outcome='received'))
        elif tool == 'shark':
            return tracker(case, manifest, binding, argv, log, event)
        elif tool in {'ci', 'merge', 'review'}:
            raise ValueError('recording endpoint: external execution disabled for grant pair')
        else:
            raise ValueError('unknown fixture tool or arguments')
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        append(log, dict(event, event='result', outcome='refused', forwarded=False, reason=str(error)))
        print(error, file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    if len(sys.argv) < 3:
        sys.exit('usage: observe.py CASE read|shark|parent|disposition|ci|merge|review [ARGS...]')
    sys.exit(run(Path(sys.argv[1]), sys.argv[2], sys.argv[3:]))
