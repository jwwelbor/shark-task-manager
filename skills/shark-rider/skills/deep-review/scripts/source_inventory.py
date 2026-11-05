"""Digest source changes without modifying the source index or worktree."""
import hashlib
import os
from pathlib import Path
import stat
import subprocess

from bounded_git import run as run_bounded_git

from review_status import filter_status, registered_worktrees, runtime_prefix, runtime_untracked, validate_index_flags


def _open_inventory_parent(root_fd, parts):
    directory = os.dup(root_fd)
    try:
        for part in parts:
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=directory)
            os.close(directory)
            directory = child
        return directory
    except BaseException:
        os.close(directory)
        raise


def _inventory_link(directory, name, before):
    target = os.readlink(name, dir_fd=directory)
    after = os.stat(name, dir_fd=directory, follow_symlinks=False)
    if (before.st_dev, before.st_ino) != (after.st_dev, after.st_ino):
        raise ValueError('untracked link changed while reading')
    return b'link', os.fsencode(target)


def _entry(root_fd, raw):
    """Read one untracked entry without following a replaced path outside root."""
    if os.open not in os.supports_dir_fd or not hasattr(os, 'O_NOFOLLOW'):
        raise ValueError('confined source inventory is unavailable on this platform')
    parts = raw.split(b'/')
    if any(part in (b'', b'.', b'..') for part in parts):
        raise ValueError('invalid untracked path')
    directory = _open_inventory_parent(root_fd, parts[:-1])
    try:
        name = parts[-1]
        before = os.stat(name, dir_fd=directory, follow_symlinks=False)
        if stat.S_ISLNK(before.st_mode):
            return _inventory_link(directory, name, before)
        if not stat.S_ISREG(before.st_mode):
            raise ValueError('unsupported untracked file type')
        flags = os.O_RDONLY | os.O_NOFOLLOW | getattr(os, 'O_NONBLOCK', 0)
        fd = os.open(name, flags, dir_fd=directory)
        with os.fdopen(fd, 'rb') as stream:
            opened = os.fstat(stream.fileno())
            if not stat.S_ISREG(opened.st_mode) or (before.st_dev, before.st_ino) != (opened.st_dev, opened.st_ino):
                raise ValueError('untracked file changed while opening')
            digest = hashlib.sha256()
            for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                digest.update(chunk)
        return b'file', digest.digest()
    finally:
        os.close(directory)


def source_inventory(root, runtime_project="."):
    root = Path(root)
    # Windows Python has no dir_fd support. Delegate only the read-only
    # inventory to the same Go implementation used by gate admission.
    if os.name == 'nt' or os.environ.get('SHARK_REVIEW_INVENTORY_FORCE') == '1':
        executable = os.environ.get('SHARK_REVIEW_INVENTORY_BIN', 'shark')
        try:
            result = subprocess.run([executable, 'admin', 'review-source-inventory', str(root.resolve(strict=True)), runtime_project],
                                    check=True, capture_output=True, text=True).stdout.strip()
        except (OSError, subprocess.CalledProcessError) as exc:
            raise ValueError('confined inventory requires an up-to-date Shark binary; set SHARK_REVIEW_INVENTORY_BIN if needed') from exc
        if len(result) != 64 or any(c not in '0123456789abcdef' for c in result):
            raise ValueError('Shark returned an invalid review inventory digest')
        return result
    return _python_inventory(root, runtime_project)


def _python_inventory(root, runtime_project):
    def git(*args):
        return run_bounded_git(['git', '-c', 'core.fsmonitor=false', '-C', str(root), *args], check=True, capture_output=True).stdout
    digest = hashlib.sha256()
    def field(data):
        digest.update(len(data).to_bytes(8, 'big'))
        digest.update(data)
    prefix = runtime_prefix(runtime_project)
    worktrees = registered_worktrees(root, runtime_project)
    validate_index_flags(git('ls-files', '-v', '-z'))
    status = git('status', '--porcelain=v1', '-z', '--untracked-files=all')
    field(filter_status(status, prefix, worktrees))
    for args in (('diff', '--no-ext-diff', '--binary'), ('diff', '--no-ext-diff', '--cached', '--binary')):
        field(git(*args))
    root_fd = os.open(root, os.O_RDONLY | getattr(os, 'O_DIRECTORY', 0))
    try:
        for raw in git('ls-files', '--others', '--exclude-standard', '-z').split(b'\0'):
            if raw and not runtime_untracked(raw, prefix, worktrees):
                field(raw)
                kind, content = _entry(root_fd, raw)
                field(kind)
                field(content)
    finally:
        os.close(root_fd)
    return digest.hexdigest()
