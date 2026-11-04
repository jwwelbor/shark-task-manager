"""Filter untracked Shark runtime files while retaining every tracked change."""
import os
import subprocess
from pathlib import Path, PurePosixPath
import sys


def runtime_project(repo_root, shark_root):
    root = Path(repo_root).resolve(strict=True)
    project = Path(shark_root).resolve(strict=True)
    try:
        return project.relative_to(root).as_posix()
    except ValueError as exc:
        raise ValueError('Shark project must be inside the candidate repository') from exc


def runtime_prefix(project_relative):
    path = PurePosixPath(project_relative)
    if path.is_absolute() or '..' in path.parts or not project_relative or '\\' in project_relative or str(path) != project_relative:
        raise ValueError('invalid Shark project location')
    parts = [] if project_relative == '.' else list(path.parts)
    return os.fsencode('/'.join([*parts, '.shark', 'runs']) + '/')


def validate_index_flags(data):
    if data and not data.endswith(b'\0'):
        raise ValueError('malformed Git index flags')
    for entry in data.split(b'\0'):
        if not entry:
            continue
        if len(entry) < 3 or entry[1:2] != b' ':
            raise ValueError('malformed Git index flags')
        flag = entry[0]
        if flag == ord('S') or ord('a') <= flag <= ord('z'):
            raise ValueError(f'tracked path {os.fsdecode(entry[2:])!r} has assume-unchanged or skip-worktree set; clear it before review')


def registered_worktrees(repo_root, project_relative):
    root = Path(repo_root).resolve(strict=True)
    base = root / project_relative / '.shark-run-worktrees'
    raw = subprocess.run(['git', '-c', 'core.fsmonitor=false', '-C', str(root), 'worktree', 'list', '--porcelain', '-z'],
                         check=True, capture_output=True).stdout
    result = []
    for field in raw.split(b'\0'):
        if not field.startswith(b'worktree '):
            continue
        worktree = Path(os.fsdecode(field[len(b'worktree '):]))
        if worktree.parent != base:
            continue
        result.append(os.fsencode(worktree.relative_to(root).as_posix()) + b'/')
    return result


def runtime_untracked(path, prefix, worktrees=()):
    return path.startswith(prefix) or any(path.rstrip(b'/') == item.rstrip(b'/') or path.startswith(item) for item in worktrees)


def selection_overlaps_runtime(path, prefix):
    selected = path.rstrip(b'/')
    reserved = prefix.rstrip(b'/')
    return selected == reserved or selected.startswith(prefix) or reserved.startswith(selected + b'/')


def _status_record(records, index, prefix, worktrees):
    record = records[index]
    index += 1
    if not record:
        return [], index
    if len(record) < 4 or record[2:3] != b' ':
        raise ValueError('malformed git status record')
    kind, path = record[:2], record[3:]
    if kind == b'??' and runtime_untracked(path, prefix, worktrees):
        return [], index
    if b'R' not in kind and b'C' not in kind:
        return [record], index
    if index >= len(records) or not records[index]:
        raise ValueError('malformed git rename record')
    return [record, records[index]], index + 1


def filter_status(status, prefix, worktrees=()):
    if status and not status.endswith(b'\0'):
        raise ValueError('malformed git status')
    records = status.split(b'\0')
    result = []
    index = 0
    while index < len(records):
        items, index = _status_record(records, index, prefix, worktrees)
        result.extend(items)

    return b''.join(item + b'\0' for item in result)


def main():
    if len(sys.argv) != 2:
        raise ValueError('expected the relative Shark project path')
    root = subprocess.run(['git', '-c', 'core.fsmonitor=false', 'rev-parse', '--show-toplevel'],
                          check=True, capture_output=True).stdout.decode().strip()
    flags = subprocess.run(['git', '-c', 'core.fsmonitor=false', '-C', root, 'ls-files', '-v', '-z'],
                           check=True, capture_output=True).stdout
    validate_index_flags(flags)
    return 1 if filter_status(sys.stdin.buffer.read(), runtime_prefix(sys.argv[1]), registered_worktrees(root, sys.argv[1])) else 0


if __name__ == '__main__':
    sys.exit(main())
