#!/usr/bin/env python3
"""Fix compilation errors after migration."""

import re
from pathlib import Path

def fix_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    original = content
    fixed = False

    # Remove unused db import
    if '"github.com/jwwelbor/shark-task-manager/internal/db"' in content:
        if 'db.InitDB' not in content and 'db.Turso' not in content and 'db.SQLite' not in content:
            content = re.sub(
                r'\t"github://jwwelbor/shark-task-manager/internal/db"\n',
                '',
                content
            )
            fixed = True

    # Fix duplicate declarations: repoDb, err := cli.GetDB when repoDb already exists
    # Change to: repoDb, err = cli.GetDB
    content = re.sub(
        r'(\trepoDb), err := cli\.GetDB\(cmd\.Context\(\)\)',
        r'\1, err = cli.GetDB(cmd.Context())',
        content
    )
    if content != original:
        fixed = True
        original = content

    # Fix undefined database variable by replacing with repoDb
    content = re.sub(
        r'\brepository\.NewDB\(database\)',
        'repoDb',
        content
    )
    if content != original:
        fixed = True

    if fixed:
        with open(filepath, 'w') as f:
            f.write(content)
        return True
    return False

def main():
    commands_dir = Path('internal/cli/commands')
    fixed = 0

    for filepath in commands_dir.glob('*.go'):
        if fix_file(filepath):
            print(f"✓ Fixed: {filepath}")
            fixed += 1

    print(f"\nFixed {fixed} files")

if __name__ == '__main__':
    main()
