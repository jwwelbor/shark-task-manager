"""Helpers the host-side recorder shares with the preparer. Holds no scenario expectations."""

import hashlib
from pathlib import Path
import sqlite3


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def tree_identity(root):
    """Identity of a whole candidate workspace: a lone file hashes as itself, several files as one listing."""
    files = sorted(p for p in Path(root).rglob('*') if p.is_file())
    if len(files) == 1:
        return digest(files[0])
    listing = ''.join(f'{p.relative_to(root)} {digest(p)}\n' for p in files)
    return hashlib.sha256(listing.encode()).hexdigest()


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
