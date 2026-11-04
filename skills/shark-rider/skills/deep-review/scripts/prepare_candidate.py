#!/usr/bin/env python3
"""Create an isolated, committed review snapshot from explicitly selected paths.

The source index and worktree are never changed. The caller must enumerate all
candidate paths, including relevant untracked files, and inspect the resulting
base-to-candidate manifest before dispatching review.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile

from source_inventory import source_inventory


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--project-root', required=True)
    parser.add_argument('--destination', required=True)
    parser.add_argument('--metadata-file', required=True)
    parser.add_argument('paths', nargs='+', help='explicit repository-relative candidate paths')
    args = parser.parse_args()
    root = Path(args.project_root).resolve(strict=True)
    destination = Path(args.destination).absolute()
    metadata_path = Path(args.metadata_file).absolute()
    if not metadata_path.parent.is_dir():
        parser.error('metadata-file parent must already exist')
    if metadata_path.exists() or metadata_path.resolve().is_relative_to(root) or metadata_path.resolve().is_relative_to(destination.resolve()):
        parser.error('metadata-file must be new and outside both worktrees')
    if destination.exists():
        parser.error('destination already exists')
    if destination.resolve().is_relative_to(root):
        parser.error('destination must be outside the source worktree')
    for path in args.paths:
        candidate = Path(path)
        if candidate.is_absolute() or '..' in candidate.parts or path in ('', '.'):
            parser.error(f'path must name a repository-relative candidate: {path!r}')
    def git(*argv, env=None, input=None):
        return subprocess.run(['git', '-C', str(root), *argv], env=env, input=input,
                              text=True, check=True, capture_output=True).stdout.strip()
    if Path(git('rev-parse', '--show-toplevel')).resolve() != root:
        parser.error('project-root must be the repository root')
    head = git('rev-parse', 'HEAD')
    # A single snapshot tree cannot represent two different versions of one
    # selected file. Never silently discard either the staged or worktree one.
    def changed_names(*options):
        raw = subprocess.run(['git', '-C', str(root), '--literal-pathspecs', 'diff', *options, '--name-only', '-z', '--no-renames', '--', *args.paths], check=True, capture_output=True).stdout
        return {os.fsdecode(path) for path in raw.split(b'\0') if path}
    staged = changed_names('--cached')
    unstaged = changed_names()
    divergent = staged & unstaged
    if divergent:
        parser.error('selected paths have divergent staged and worktree versions: ' + ', '.join(sorted(divergent)))
    inventory = source_inventory(root)
    staged_diff = subprocess.run(['git', '-C', str(root), '--literal-pathspecs', 'diff', '--no-ext-diff', '--cached', '--binary', head, '--', *args.paths], check=True, capture_output=True).stdout
    with tempfile.TemporaryDirectory(prefix='review-index-') as temp:
        env = dict(os.environ, GIT_INDEX_FILE=str(Path(temp) / 'index'),
                   GIT_AUTHOR_NAME='Review Snapshot', GIT_AUTHOR_EMAIL='review-snapshot@localhost',
                   GIT_COMMITTER_NAME='Review Snapshot', GIT_COMMITTER_EMAIL='review-snapshot@localhost')
        git('read-tree', head, env=env)
        git('--literal-pathspecs', 'add', '-A', '--', *args.paths, env=env)
        tree = git('write-tree', env=env)
        commit = git('-c', 'commit.gpgsign=false', 'commit-tree', tree, '-p', head,
                     env=env, input='Shark isolated review snapshot v1\n')
    git('worktree', 'add', '--detach', str(destination), commit)
    metadata = {'project_root': str(destination), 'source_root': str(root),
                'source_commit': head, 'candidate_commit': commit, 'candidate_tree': tree,
                'selected_paths': args.paths, 'selected_index_sha256': hashlib.sha256(staged_diff).hexdigest(),
                'source_inventory_sha256': inventory}
    # A worktree-local marker survives an ordinary amend/reword of the
    # synthetic commit; its subject alone is not durable provenance.
    git_dir = Path(subprocess.run(['git', '-C', str(destination), 'rev-parse', '--absolute-git-dir'],
                                  check=True, capture_output=True, text=True).stdout.strip())
    with (git_dir / 'shark-review-source.json').open('x', encoding='utf-8') as marker:
        json.dump(metadata, marker)
    with metadata_path.open('x', encoding='utf-8') as out:
        json.dump(metadata, out)
    print(json.dumps(metadata))


if __name__ == '__main__':
    main()
