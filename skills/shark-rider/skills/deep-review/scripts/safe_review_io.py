"""Bounded, nonblocking regular-file reads for review evidence."""
import os
from pathlib import Path
import stat
import subprocess

from bounded_git import run as run_bounded_git

MAX_ARTIFACT_BYTES = 128 * 1024 * 1024


def _read_with_shark(path, limit):
    executable = os.environ.get('SHARK_REVIEW_INVENTORY_BIN', 'shark')
    try:
        data = run_bounded_git([executable, 'admin', 'review-read-artifact', str(path), str(limit)],
                              check=True, capture_output=True).stdout
    except (OSError, subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
        raise ValueError('safe review reads require an up-to-date Shark binary') from exc
    if len(data) > limit:
        raise ValueError('review artifact exceeds size limit')
    return data


def _read_native(path, limit):
    flags = os.O_RDONLY | os.O_NOFOLLOW | getattr(os, 'O_NONBLOCK', 0)
    fd = os.open(path, flags)
    with os.fdopen(fd, 'rb') as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_size > limit:
            raise ValueError('review artifact must be a bounded regular file')
        data = stream.read(limit + 1)
        if len(data) > limit:
            raise ValueError('review artifact exceeds size limit')
        return data


def read_review_file(path, limit):
    if limit <= 0 or limit > MAX_ARTIFACT_BYTES:
        raise ValueError('invalid review artifact size limit')
    path = Path(path).absolute()
    if os.name == 'nt' or not hasattr(os, 'O_NOFOLLOW') or os.environ.get('SHARK_REVIEW_IO_FORCE') == '1':
        return _read_with_shark(path, limit)
    return _read_native(path, limit)
