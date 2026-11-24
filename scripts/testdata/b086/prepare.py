"""Prepare the two B086 grant cases. The manifest records setup evidence only."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
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


def output(args, cwd, **kwargs):
    return subprocess.run(args, cwd=cwd, stdout=subprocess.PIPE, check=True, **kwargs).stdout


def build_policy(source):
    with tempfile.TemporaryFile() as archive:
        subprocess.run(['git', 'archive', POLICY, 'cmd', 'internal', 'go.mod', 'go.sum',
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


def prepare(root, finish, retained=None):
    root = root.resolve(strict=True)
    checkout = subprocess.run(['git', 'rev-parse', '--show-toplevel'], cwd=root, capture_output=True)
    if not root.is_relative_to(Path('/tmp').resolve()) or any(root.iterdir()) or checkout.returncode == 0:
        raise ValueError('use an empty disposable directory under /tmp outside a checkout')
    if digest(finish / 'SKILL.md') != FINISH_DIGEST:
        raise ValueError('owning finish-feature differs from the reviewed source')
    packet_files, packet_receipt = {}, None
    if retained:
        from packet import read_input
        report, report_hash, repository, candidate = retained
        packet_files, packet_receipt = read_input(Path(report), report_hash, Path(repository), candidate)
    with tempfile.TemporaryDirectory(dir=root) as staging_name:
        staging = Path(staging_name)
        source = staging / 'policy'
        source.mkdir()
        binary_digest = build_policy(source)
        seed = seed_project(source, staging, binary_digest)
        catalog = staging / 'catalog'
        fixture = Path(__file__).resolve().parent
        libexec = staging / 'libexec'
        libexec.mkdir()
        (seed / 'shark').rename(libexec / 'shark')
        for module in ('observe.py', 'prepare.py'):
            shutil.copy2(fixture / module, libexec / module)
        shutil.copyfile(fixture / 'shark', seed / 'shark')
        (seed / 'shark').chmod(0o755)
        shutil.copytree(source / 'skills/shark-rider', catalog / 'shark-rider')
        (catalog / 'finish-feature').mkdir()
        shutil.copy2(finish / 'SKILL.md', catalog / 'finish-feature/SKILL.md')
        cases = []
        bindings = {}
        for case_name, grant in [('grant-a', 'E01-F01'), ('grant-b', 'E01-F02')]:
            case = root / case_name
            case.mkdir()
            shutil.copytree(seed, case / 'project')
            shutil.copytree(libexec, case / 'libexec')
            shutil.copytree(catalog, case / 'catalog')
            dispatch = {
                'mode': 'unattended', 'requested_feature': 'E01-F01',
                'create_task_under': grant, 'edit_authority': False,
                'budget': {
                    'dispatch_remaining_attempts': 1, 'project_total_attempts': 2,
                    'project_consumed_attempts': 1, 'available_minutes': 12,
                    'estimated_repair_and_verification_minutes': 10,
                },
            }
            (case / 'grant.json').write_text(json.dumps(dispatch, indent=2) + '\n')
            if packet_files:
                (case / 'packet').mkdir()
                for name, data in packet_files.items():
                    (case / 'packet' / name).write_bytes(data)
            store = (case / 'project/shark-tasks.db').stat()
            bindings[case_name] = {'run_id': str(uuid.uuid4()), 'db_inode': [store.st_dev, store.st_ino]}
            cases.append(case_name)
        files = {'project/shark': digest(seed / 'shark'),
                 'libexec/shark': binary_digest,
                 'project/.sharkconfig.json': digest(seed / '.sharkconfig.json')}
        files.update({str(path.relative_to(staging)): digest(path) for path in libexec.glob('*.py')})
        files.update({str(path.relative_to(staging)): digest(path)
                      for path in catalog.rglob('*') if path.is_file()})
        manifest = {'policy_commit': POLICY, 'review_baseline': BASELINE,
                    'fixture_commit': output(['git', 'rev-parse', 'HEAD'], ROOT).decode().strip(),
                    'fixture_files': {p.name: digest(p) for p in Path(__file__).parent.glob('*.py')},
                    'prepare_sha256': digest(Path(__file__)), 'cases': cases,
                    'bindings': bindings, 'packet': packet_receipt,
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
    parser.add_argument('--retained', nargs=4, metavar=('REPORT', 'SHA256', 'REPOSITORY', 'COMMIT'),
                        help='stage the original report and sidecars in each case after verification')
    args = parser.parse_args()
    print(prepare(args.directory, args.finish_feature, args.retained))
