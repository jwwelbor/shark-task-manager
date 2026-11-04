"""Digest the source change inventory without modifying its index or files."""
import hashlib
import os
from pathlib import Path
import stat
import subprocess


def source_inventory(root):
    root = Path(root)
    def git(*args):
        return subprocess.run(['git', '-C', str(root), *args], check=True, capture_output=True).stdout
    digest = hashlib.sha256()
    def field(data):
        digest.update(len(data).to_bytes(8, 'big'))
        digest.update(data)
    for args in (('status', '--porcelain=v1', '-z', '--untracked-files=all'),
                 ('diff', '--no-ext-diff', '--binary'), ('diff', '--no-ext-diff', '--cached', '--binary')):
        field(git(*args))
    for raw in git('ls-files', '--others', '--exclude-standard', '-z').split(b'\0'):
        if not raw:
            continue
        field(raw)
        path = root / os.fsdecode(raw)
        info = path.lstat()
        if stat.S_ISLNK(info.st_mode):
            field(b'link')
            field(os.fsencode(os.readlink(path)))
        elif stat.S_ISREG(info.st_mode):
            field(b'file')
            file_digest = hashlib.sha256()
            with path.open('rb') as stream:
                for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                    file_digest.update(chunk)
            field(file_digest.digest())
        else:
            raise ValueError(f'unsupported untracked file type: {path}')
    return digest.hexdigest()
