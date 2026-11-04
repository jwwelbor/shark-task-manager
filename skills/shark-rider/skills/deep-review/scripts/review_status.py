"""Filter untracked Shark runtime files while retaining every tracked change."""
import os
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


def runtime_untracked(path, prefix):
    return path.startswith(prefix)


def selection_overlaps_runtime(path, prefix):
    selected = path.rstrip(b'/')
    reserved = prefix.rstrip(b'/')
    return selected == reserved or selected.startswith(prefix) or reserved.startswith(selected + b'/')


def _status_record(records, index, prefix):
    record = records[index]
    index += 1
    if not record:
        return [], index
    if len(record) < 4 or record[2:3] != b' ':
        raise ValueError('malformed git status record')
    kind, path = record[:2], record[3:]
    if kind == b'??' and runtime_untracked(path, prefix):
        return [], index
    if b'R' not in kind and b'C' not in kind:
        return [record], index
    if index >= len(records) or not records[index]:
        raise ValueError('malformed git rename record')
    return [record, records[index]], index + 1


def filter_status(status, prefix):
    if status and not status.endswith(b'\0'):
        raise ValueError('malformed git status')
    records = status.split(b'\0')
    result = []
    index = 0
    while index < len(records):
        items, index = _status_record(records, index, prefix)
        result.extend(items)

    return b''.join(item + b'\0' for item in result)


def main():
    if len(sys.argv) != 2:
        raise ValueError('expected the relative Shark project path')
    return 1 if filter_status(sys.stdin.buffer.read(), runtime_prefix(sys.argv[1])) else 0


if __name__ == '__main__':
    sys.exit(main())
