"""Compare frozen Git objects in the repository used for review capture."""
import hashlib
import re
import subprocess

from bounded_git import OutputLimitExceeded, run as run_bounded_git
from review_status import git_path_text


def _git(root, *args):
    return run_bounded_git(['git', '-c', 'core.fsmonitor=false', '-C', str(root), *args],
                           check=True, capture_output=True).stdout


def changed_paths(root, start, end):
    names = _git(root, 'diff', '--name-only', '-z', start, end)
    return [git_path_text(raw) for raw in names.split(b'\0') if raw]


def successor_delta(root, base, predecessor, candidate):
    """A comparison supplies inputs to review. It cannot authenticate an inspection."""
    if not isinstance(predecessor, str) or not re.fullmatch(r'(?:[0-9a-f]{40}|[0-9a-f]{64})', predecessor):
        return None
    try:
        if _git(root, 'cat-file', '-t', predecessor).strip() != b'commit':
            return None
        _git(root, 'merge-base', '--is-ancestor', base, predecessor)
        _git(root, 'merge-base', '--is-ancestor', predecessor, candidate)
        tree = _git(root, 'rev-parse', predecessor + '^{tree}').decode('ascii').strip()
        patch = _git(root, 'diff', '--no-ext-diff', '--binary', predecessor, candidate)
        return {
            'predecessor_commit': predecessor,
            'predecessor_tree': tree,
            'changed_files': changed_paths(root, predecessor, candidate),
            'diff_sha256': hashlib.sha256(patch).hexdigest(),
        }
    except (OSError, subprocess.CalledProcessError, subprocess.TimeoutExpired, OutputLimitExceeded):
        # The complete baseline comparison remains available for a fresh review.
        return None
