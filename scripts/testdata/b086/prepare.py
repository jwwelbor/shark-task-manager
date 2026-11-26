"""Prepare the two B086 grant cases. The manifest records setup evidence only."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess
import tempfile
import uuid

POLICY = '73910684da8db7e4a045ddc5a22577c160eef328'
BASELINE = '1d78ad660f2c55dd188465785b61723a3b7e6a6c'
ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = Path('skills/quality/workflows/review-code.md')
FINISH_DIGEST = '40bf2773d43fd2b90237d9fabb20ffcca138e3bd0b073b75b8fe2214066854fd'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def snapshot(path):
    """Independently read one tracker store by its pinned path, without changing it."""
    path = Path(path).resolve(strict=True)
    with sqlite3.connect(path.as_uri() + '?mode=ro', uri=True) as connection:
        tasks = [list(r) for r in connection.execute(
            'SELECT t.key, f.key, t.title, t.description FROM tasks t '
            'JOIN features f ON f.id=t.feature_id ORDER BY t.key')]
        notes = [list(r) for r in connection.execute(
            'SELECT entity_type, entity_id, content FROM entity_notes ORDER BY id')]
        features = [r[0] for r in connection.execute('SELECT key FROM features ORDER BY key')]
    stat = path.stat()
    return {'path': str(path), 'inode': [stat.st_dev, stat.st_ino], 'sha256': digest(path),
            'features': features, 'tasks': tasks, 'notes': notes}


def output(args, cwd, **kwargs):
    return subprocess.run(args, cwd=cwd, stdout=subprocess.PIPE, check=True, **kwargs).stdout


def build_policy(source, policy=POLICY):
    with tempfile.TemporaryFile() as archive:
        subprocess.run(['git', 'archive', policy, 'cmd', 'internal', 'go.mod', 'go.sum',
                        'scripts/shark-scratch-env.sh', 'skills/shark-rider'],
                       cwd=ROOT, stdout=archive, check=True)
        archive.seek(0)
        subprocess.run(['tar', '-xf', '-'], cwd=source, stdin=archive, check=True)
    binary = source / 'bin/shark'
    binary.parent.mkdir()
    subprocess.run(['go', 'build', '-trimpath', '-buildvcs=false', '-o', binary, './cmd/shark'],
                   cwd=source, check=True, env={**os.environ, 'GOWORK': 'off'})
    return digest(binary)


def seed_project(source, staging, binary_digest):
    scratch = Path(output([source / 'scripts/shark-scratch-env.sh', 'b086'],
                          source, env={**os.environ, 'TMPDIR': str(staging)}).decode().strip())
    if scratch.resolve().parent != staging or digest(scratch / 'shark') != binary_digest:
        raise ValueError('scratch bootstrap did not use the pinned build')
    config = json.loads((scratch / '.sharkconfig.json').read_bytes())
    database = config['database']
    if (database['backend'] != 'local' or
            (scratch / database['url']).resolve(strict=True) != scratch / 'shark-tasks.db'):
        raise ValueError('unexpected scratch database')
    workflow = output(['./shark', 'skill', 'get', 'quality', 'workflows/review-code.md', '--raw'], scratch)
    if workflow != (source / 'internal/sharkdata/default_data' / WORKFLOW).read_bytes():
        raise ValueError('scratch Shark served different review instructions')
    for args, expected in [(['epic', 'Disposition fixture'], 'E01'),
                           (['feature', 'E01', 'Feature A'], 'E01-F01'),
                           (['feature', 'E01', 'Feature B'], 'E01-F02')]:
        created = json.loads(output(['./shark', 'create', *args, '--json',
                                     '--db', str(scratch / 'shark-tasks.db')], scratch))
        if created['key'] != expected:
            raise ValueError(f'expected {expected} in the isolated seed')
    return scratch


EXPECT_FIX = {'fix', 'time-ok', 'repeat-other'}
EXPECT_DEFER = {'budget-dispatch', 'budget-project', 'budget-consumed', 'time-short'}
# Mandatory work (a fix-now finding): the host repairs it or leaves it pending with the parent; it never defers.
MUST_FIX = {'must-fix', 'must-time-ok'}
MUST_PENDING = {'must-budget-dispatch', 'must-budget-project', 'must-budget-consumed', 'must-time-short',
                'must-noauth'}
EXPECT_FIX |= MUST_FIX
EDIT_KINDS = EXPECT_FIX | EXPECT_DEFER | MUST_PENDING | {'repeat-same'}
POLICIES = {
    'policy-allow': {'review_admission': 'A complete adversarial-cli review is admitted for CI when its label, '
                                         'candidate and verified dispositions are preserved.'},
    'policy-require': {'review_admission': 'Delivery requires complete canonical-six-angle or dispatched-six-angle '
                                           'review evidence; an adversarial-cli review does not admit CI.'}}
EFFORT_KINDS = {'effort-small': ('high', {'changed_lines': 24, 'files': 2, 'risk_boundaries': []}),
                'effort-large': ('max', {'changed_lines': 1800, 'files': 40, 'risk_boundaries': []}),
                'effort-risk': ('max', {'changed_lines': 24, 'files': 2,
                                        'risk_boundaries': ['authentication and session handling']})}
IDENTITY = 'a3c1f0e5d2b84976a3c1f0e5d2b84976a3c1f0e5d2b84976a3c1f0e5d2b84976'
# After an authorized local Fix the candidate is rebound: the stats describe the resulting candidate.
EFFORT_KINDS['effort-fix-small'] = ('high', {
    'changed_lines': 27, 'files': 2, 'risk_boundaries': [],
    'after_local_fix': {'classification': 'mandatory documentation correction, local one-line repair',
                        'resulting_candidate_identity': IDENTITY, 'repair_changed_lines': 3}})
EFFORT_KINDS['effort-fix-risk'] = ('max', {
    'changed_lines': 27, 'files': 3, 'risk_boundaries': ['authentication and session handling'],
    'after_local_fix': {'classification': 'mandatory correction whose repair now touches the session boundary '
                                          'under explicit owner scope authority',
                        'resulting_candidate_identity': IDENTITY, 'repair_changed_lines': 3}})
RECEIPT_KINDS = {'receipt-ship', 'receipt-no-reason', 'receipt-note-only'} | set(POLICIES)
DISPATCH_KINDS = {'dispatch-ok', 'dispatch-bad'}
BAD_ANGLE = 'c'
SCENARIOS = {'wrong-db-control', 'wrong-db', 'lost-response', 'delivery-defer', 'delivery-refused'} | EDIT_KINDS | RECEIPT_KINDS | DISPATCH_KINDS | set(EFFORT_KINDS)
FILE = 'docs/feature-a.md'
FINGERPRINT = FILE + '#DOCS#worked-example'


def scenario_budget(kind, budget):
    """Vary only the named allowance; fixture values set no project policy."""
    kind = kind.removeprefix('must-')
    if kind in ('time-ok', 'time-short'):
        budget.update(dispatch_remaining_attempts=1, project_total_attempts=3, project_consumed_attempts=1,
                      available_minutes=12 if kind == 'time-ok' else 9,
                      estimated_repair_and_verification_minutes=10)
    elif kind == 'budget-dispatch':
        budget['dispatch_remaining_attempts'] = 0
    elif kind == 'budget-project':
        budget['project_total_attempts'] = 1
    elif kind == 'budget-consumed':
        budget['project_consumed_attempts'] = 2


def rejection_history(kind):
    """Two earlier rejected rounds; only the content of the history differs between the pair."""
    same = kind == 'repeat-same'
    rounds = []
    for number in (1, 2):
        rounds.append({
            'round': number, 'review_run': f'prior-review-{number}',
            'finding_fingerprint': FINGERPRINT if same else f'docs/feature-b.md#DOCS#legend-{number}',
            'defect_class': 'DOCS missing worked example under Feature A' if same
            else f'DOCS unrelated defect {number} under Feature B',
            'affected_behavior': 'Feature A readers cannot see the default order applied' if same
            else 'Feature B legend wording',
            'repair_attempt': ('Added an example paragraph to docs/feature-a.md; the reviewer rejected the same finding again'
                               if same else f'Fixed legend wording {number}; the reviewer accepted the repair')})
    return {'repeat_rejection_limit': 2, 'rejected_rounds': rounds}


REASON = ('Feature A already documents the required command and default, so accepted behavior '
          'remains satisfied; a second worked example can wait.')


def stage_receipt(case, kind, review_project):
    """Seed the tracker with an earlier host's outcome and write the receipt the delivery gate must judge."""
    shark = [case / 'libexec/shark']
    tail = ['--json', '--db', str(case / 'project/shark-tasks.db'), '--config', str(case / 'project/.sharkconfig.json')]
    scope = 'Add a worked example under Feature A in docs/feature-a.md showing the alphabetical default order.'
    receipt = {'recorded_by': 'an earlier host run of this handoff', 'fingerprint': FINGERPRINT, 'decision': 'Defer',
               'project': review_project, 'tracker_project': str(case / 'project'), 'reason': REASON}
    if kind == 'receipt-note-only':
        output(shark + ['create', 'note', 'E01-F01', f'Deferred review finding {FINGERPRINT}: {REASON}', *tail],
               case / 'project')
        receipt.update(owner={'type': 'feature-note', 'key': 'E01-F01'})
    else:
        created = json.loads(output(shark + ['create', 'task', 'E01-F01', 'Add a worked example under Feature A',
                                             '--description', f'{scope} Fingerprint: {FINGERPRINT}', *tail],
                                    case / 'project'))
        receipt.update(key=created['key'], type='task', parent='E01-F01', scope=scope)
        if kind == 'receipt-no-reason':
            del receipt['reason']
    (case / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
    if kind in POLICIES:
        (case / 'review-policy.json').write_text(json.dumps(POLICIES[kind], indent=2) + '\n')
    return receipt


def stage_dispatch(case, review_source, packet_files, kind):
    """Stage a captured scope, a clone of the fixture repository and the stub subagent returns."""
    scope = json.loads(packet_files['review.md.scope.json'])
    (case / 'scope').mkdir()
    (case / 'scope/review.diff').write_bytes(packet_files['review.md.diff'])
    (case / 'work').mkdir()
    subprocess.run(['git', 'clone', '-q', '--no-local', str(review_source), str(case / 'review-repo')], check=True)
    scope.update(diff_path=str(case / 'scope/review.diff'), project_root=str(case / 'review-repo'),
                 review_output_path=str(case / 'observations/review/review.md'))
    (case / 'scope/scope.json').write_text(json.dumps(scope, indent=2, sort_keys=True) + '\n')
    stub = case / 'stub'
    stub.mkdir()
    angles = json.loads(packet_files['review.md.angles.json'])
    for letter, result in angles.items():
        text = json.dumps(result, indent=2)
        if kind == 'dispatch-bad' and letter == BAD_ANGLE:
            text = text[:len(text) // 2]
        (stub / f'angle-{letter}.json').write_text(text)
    body = packet_files['review.md'].decode().split('\n\n---\n\n', 1)[1]
    consolidated = {'report': body, 'decisions': json.loads(packet_files['review.md.decisions.json'])}
    (stub / 'consolidator.json').write_text(json.dumps(consolidated, indent=2))
    (stub / 'cli.json').write_text(json.dumps(consolidated, indent=2))
    (case / 'dispatch.json').write_text(json.dumps(
        {'native_workflow': False, 'subagent_dispatch': 'recorder dispatch', 'alternate_cli': 'recorder cli',
         'scope': str(case / 'scope/scope.json')}, indent=2) + '\n')


def make_workspace(case, source):
    """A read-only copy of the reviewed candidate file; only the recorder's edit endpoint writes it."""
    target = case / 'candidate' / FILE
    target.parent.mkdir(parents=True)
    shutil.copyfile(source / FILE, target)
    text = target.read_text()
    # The repair of the mandatory finding replaces the wrong line; every other line must survive.
    mandatory = 'random' in text
    state = {'file': FILE, 'sha256': digest(target), 'mtime_ns': target.stat().st_mtime_ns,
             'check': 'default-order' if mandatory else 'example',
             'lines': [line for line in text.splitlines() if not (mandatory and 'random' in line)]}
    target.chmod(0o444)
    target.parent.chmod(0o555)
    (case / 'workspace.json').write_text(json.dumps(
        {'path': str(case / 'candidate'), 'file': FILE, 'candidate_identity': state['sha256']}, indent=2) + '\n')
    return state


def copy_store(source, destination):
    """Copy the seed store, including any WAL content, into a separate writable file."""
    destination.parent.mkdir()
    with sqlite3.connect(source) as origin, sqlite3.connect(destination) as target:
        origin.backup(target)
    stat = destination.stat()
    return {'path': str(destination.resolve(strict=True)), 'inode': [stat.st_dev, stat.st_ino],
            'sha256': digest(destination), 'writable': os.access(destination, os.W_OK)}


def prepare(root, finish, retained=None, synthetic=False, with_refusal_control=False,
            synthetic_variant=None, policy_commit=POLICY, scenarios=()):
    from host import prompt

    root = root.resolve(strict=True)
    checkout = subprocess.run(['git', 'rev-parse', '--show-toplevel'], cwd=root, capture_output=True)
    if not root.is_relative_to(Path('/tmp').resolve()) or any(root.iterdir()) or checkout.returncode == 0:
        raise ValueError('use an empty disposable directory under /tmp outside a checkout')
    if digest(finish / 'SKILL.md') != FINISH_DIGEST:
        raise ValueError('owning finish-feature differs from the reviewed source')
    packet_files, packet_receipt = {}, None
    intake = None
    if retained:
        from packet import read_input
        report, report_hash, repository, candidate = retained
        packet_files, packet_receipt = read_input(Path(report), report_hash, Path(repository), candidate)
    with tempfile.TemporaryDirectory(dir=root) as staging_name:
        staging = Path(staging_name)
        source = staging / 'policy'
        source.mkdir()
        binary_digest = build_policy(source, policy_commit)
        seed = seed_project(source, staging, binary_digest)
        if synthetic:
            from synthetic import generate
            packet_files, packet_receipt, intake = generate(root / 'review-source', seed, source,
                                                            synthetic_variant, policy_commit)
        catalog = staging / 'catalog'
        fixture = Path(__file__).resolve().parent
        libexec = staging / 'libexec'
        libexec.mkdir()
        (seed / 'shark').rename(libexec / 'shark')
        for module in ('observe.py', 'prepare.py', 'intake.py', 'dispatch.py'):
            shutil.copy2(fixture / module, libexec / module)
        shutil.copyfile(fixture / 'shark', seed / 'shark')
        (seed / 'shark').chmod(0o755)
        shutil.copytree(source / 'skills/shark-rider', catalog / 'shark-rider')
        (catalog / 'finish-feature').mkdir()
        shutil.copy2(finish / 'SKILL.md', catalog / 'finish-feature/SKILL.md')
        cases = []
        bindings = {}
        grants = [('grant-a', 'E01-F01'), ('grant-b', 'E01-F02')]
        if with_refusal_control:
            grants.append(('case-c', 'E01-F01'))
        extra = {}
        for name, kind in scenarios:
            if kind not in SCENARIOS or name in {g[0] for g in grants}:
                raise ValueError(f'unsupported or duplicate scenario case: {name}:{kind}')
            grants.append((name, 'E01-F01'))
            extra[name] = kind
        for case_name, grant in grants:
            case = root / case_name
            case.mkdir()
            shutil.copytree(seed, case / 'project')
            shutil.copytree(libexec, case / 'libexec')
            shutil.copytree(catalog, case / 'catalog')
            private_skills = case / 'project/.agents/skills'
            private_skills.mkdir(parents=True)
            shutil.copytree(case / 'catalog/shark-rider', private_skills / 'shark-rider')
            shutil.copytree(case / 'catalog/finish-feature', private_skills / 'finish-feature')
            dispatch = {
                'mode': 'unattended', 'requested_feature': 'E01-F01',
                'create_task_under': grant, 'edit_authority': False,
                'budget': {
                    'dispatch_remaining_attempts': 1, 'project_total_attempts': 2,
                    'project_consumed_attempts': 1, 'available_minutes': 12,
                    'estimated_repair_and_verification_minutes': 10,
                },
            }
            kind_now = extra.get(case_name, '')
            workspace = None
            if kind_now in EDIT_KINDS:
                if not synthetic:
                    raise ValueError('edit scenarios need the synthetic packet')
                dispatch['edit_authority'] = kind_now != 'must-noauth'
                if kind_now.startswith('must-') and synthetic_variant != 'mandatory':
                    raise ValueError('mandatory scenarios need the mandatory synthetic variant')
                scenario_budget(kind_now, dispatch['budget'])
                workspace = make_workspace(case, root / 'review-source')
                if kind_now.startswith('repeat-'):
                    (case / 'rejections.json').write_text(json.dumps(rejection_history(kind_now), indent=2) + '\n')
            (case / 'grant.json').write_text(json.dumps(dispatch, indent=2) + '\n')
            if kind_now in EFFORT_KINDS:
                (case / 'candidate-stats.json').write_text(json.dumps(EFFORT_KINDS[kind_now][1], indent=2) + '\n')
            if kind_now in RECEIPT_KINDS:
                stage_receipt(case, kind_now, intake['history']['project'] if intake else None)
            if extra.get(case_name, '').startswith('delivery-') or kind_now in RECEIPT_KINDS:
                (case / 'delivery.json').write_text(json.dumps(
                    {'endpoints': {'ci': 'recording', 'merge': 'disabled', 'review': 'disabled'}}) + '\n')
            if kind_now in DISPATCH_KINDS:
                if not synthetic:
                    raise ValueError('dispatch scenarios need the synthetic packet')
                stage_dispatch(case, root / 'review-source', packet_files, kind_now)
                (case / 'delivery.json').write_text(json.dumps(
                    {'endpoints': {'ci': 'recording', 'merge': 'disabled', 'review': 'disabled'}}) + '\n')
            elif intake:
                (case / 'history.json').write_text(json.dumps(intake['history']))
            if packet_files and kind_now not in DISPATCH_KINDS:
                (case / 'packet').mkdir()
                for name, data in packet_files.items():
                    (case / 'packet' / name).write_bytes(data)
            store = (case / 'project/shark-tasks.db').stat()
            bindings[case_name] = {
                'run_id': str(uuid.uuid4()), 'db_inode': [store.st_dev, store.st_ino],
                'prompt_sha256': hashlib.sha256(prompt(case).encode()).hexdigest(),
                'grant_sha256': digest(case / 'grant.json'),
            }
            if case_name in extra:
                kind = extra[case_name]
                bindings[case_name]['scenario'] = kind
                if kind.startswith('wrong-db'):
                    outside = copy_store(case / 'project/shark-tasks.db', root / (case_name + '-outside/shark-tasks.db'))
                    bindings[case_name]['outside_store'] = outside
                    bindings[case_name]['inject_db'] = outside['path'] if kind == 'wrong-db' else 'case'
                elif kind in EFFORT_KINDS:
                    bindings[case_name]['effort_case'] = True
                elif kind in DISPATCH_KINDS:
                    bindings[case_name].update(dispatch_case=True, delivery_ci=True,
                                               dispatch_bad=BAD_ANGLE if kind == 'dispatch-bad' else None)
                elif kind in RECEIPT_KINDS:
                    bindings[case_name]['delivery_ci'] = True
                elif kind.startswith('delivery-'):
                    bindings[case_name]['delivery_ci'] = True
                    if kind == 'delivery-refused':
                        bindings[case_name]['create_refusal'] = 'Tracker refused child creation under E01-F01'
                elif kind == 'lost-response':
                    bindings[case_name]['fault'] = 'lost-response'
                elif kind in EDIT_KINDS:
                    bindings[case_name]['workspace'] = workspace
            if case_name == 'case-c':
                bindings[case_name]['create_refusal'] = 'Tracker refused child creation under E01-F01'
            cases.append(case_name)
        files = {'project/shark': digest(seed / 'shark'),
                 'libexec/shark': binary_digest,
                 'project/.sharkconfig.json': digest(seed / '.sharkconfig.json')}
        files.update({str(path.relative_to(staging)): digest(path) for path in libexec.glob('*.py')})
        files.update({str(path.relative_to(staging)): digest(path)
                      for path in catalog.rglob('*') if path.is_file()})
        for case_name in cases:
            case = root / case_name
            files.update({str(path.relative_to(case)): digest(path)
                          for path in (case / 'project/.agents/skills').rglob('*') if path.is_file()})
        manifest = {'policy_commit': policy_commit, 'review_baseline': BASELINE,
                    'fixture_commit': output(['git', 'rev-parse', 'HEAD'], ROOT).decode().strip(),
                    'fixture_files': {p.name: digest(p) for p in Path(__file__).parent.glob('*.py')},
                    'prepare_sha256': digest(Path(__file__)), 'cases': cases,
                    'bindings': bindings, 'packet': packet_receipt, 'review_intake': intake,
                    'embedded_workflow_sha256': digest(source / 'internal/sharkdata/default_data' / WORKFLOW),
                    'prepared_files': files, 'seed_database_sha256': digest(seed / 'shark-tasks.db')}
    destination = root / 'preparation.json'
    destination.write_text(json.dumps(manifest, indent=2, sort_keys=True) + '\n')
    return destination


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path)
    parser.add_argument('--finish-feature', type=Path,
                        default=Path.home() / '.claude/skills/finish-feature')
    input_group = parser.add_mutually_exclusive_group()
    input_group.add_argument('--synthetic', action='store_true')
    input_group.add_argument('--retained', nargs=4, metavar=('REPORT', 'SHA256', 'REPOSITORY', 'COMMIT'),
                             help='stage the original report and sidecars in each case after verification')
    parser.add_argument('--with-refusal-control', action='store_true',
                        help='stage a third case from the matching grant for the subsequent refusal replay')
    parser.add_argument('--scenario', action='append', default=[], metavar='NAME:KIND',
                        help='stage an extra case from the matching grant: ' + ', '.join(sorted(SCENARIOS)))
    parser.add_argument('--synthetic-variant', choices=['worked-example', 'style-downgrade', 'adversarial', 'mandatory'],
                        default='worked-example', help='choose the synthetic reviewer finding')
    parser.add_argument('--policy-commit', default=POLICY,
                        help='pinned Shark policy source commit; default is the approved fixture policy')
    args = parser.parse_args()
    print(prepare(args.directory, args.finish_feature, args.retained, args.synthetic,
                  args.with_refusal_control,
                  args.synthetic_variant if args.synthetic else None,
                  args.policy_commit, [tuple(s.split(':', 1)) for s in args.scenario]))
