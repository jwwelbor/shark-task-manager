#!/usr/bin/env python3
"""
Complete the migration to cli.GetDB() pattern by replacing remaining db.InitDB patterns.
"""

import re
import sys
from pathlib import Path

def migrate_file(filepath):
    """Migrate a single file to use cli.GetDB pattern."""
    with open(filepath, 'r') as f:
        content = f.read()

    original = content

    # Pattern 1: Standard GetDBPath + InitDB + NewDB pattern
    # Before:
    #   dbPath, err := cli.GetDBPath()
    #   if err != nil {
    #       return fmt.Errorf("failed to get database path: %w", err)
    #   }
    #
    #   database, err := db.InitDB(dbPath)
    #   if err != nil {
    #       return fmt.Errorf("failed to initialize database: %w", err)
    #   }
    #
    # After:
    #   repoDb, err := cli.GetDB(cmd.Context())
    #   if err != nil {
    #       return fmt.Errorf("failed to get database: %w", err)
    #   }

    pattern1 = re.compile(
        r'\tdbPath, err := cli\.GetDBPath\(\)\s*\n'
        r'\tif err != nil \{\s*\n'
        r'\t\treturn fmt\.Errorf\("failed to get database path: %w", err\)\s*\n'
        r'\t\}\s*\n'
        r'\s*\n'
        r'\tdatabase, err := db\.InitDB\(dbPath\)\s*\n'
        r'\tif err != nil \{\s*\n'
        r'\t\treturn fmt\.Errorf\("failed to initialize database: %w", err\)\s*\n'
        r'\t\}',
        re.MULTILINE
    )

    replacement1 = (
        '\trepoDb, err := cli.GetDB(cmd.Context())\n'
        '\tif err != nil {\n'
        '\t\treturn fmt.Errorf("failed to get database: %w", err)\n'
        '\t}'
    )

    content = pattern1.sub(replacement1, content)

    # Pattern 2: Replace repository.NewDB(database) with repoDb after migration
    if 'cli.GetDB(cmd.Context())' in content:
        content = re.sub(
            r'repository\.NewDB\(database\)',
            'repoDb',
            content
        )

    if content != original:
        with open(filepath, 'w') as f:
            f.write(content)
        return True
    return False

def main():
    commands_dir = Path('internal/cli/commands')
    migrated = 0

    for filepath in commands_dir.glob('*.go'):
        # Skip test files
        if filepath.name.endswith('_test.go'):
            continue

        # Skip if already using cli.GetDB
        with open(filepath) as f:
            content = f.read()
            if 'cli.GetDB(' in content and 'db.InitDB' not in content:
                continue

        if migrate_file(filepath):
            print(f"✓ Migrated: {filepath}")
            migrated += 1

    print(f"\nMigrated {migrated} files")

if __name__ == '__main__':
    main()
