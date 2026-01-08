#!/bin/bash
set -e

COMMANDS_DIR="internal/cli/commands"
BACKUP_DIR="dev-artifacts/2026-01-08-database-initialization-architecture/backup"

echo "=== Database Initialization Migration Script ==="
echo ""
echo "This script will:"
echo "  1. Backup all command files"
echo "  2. Replace old database init pattern with cli.GetDB()"
echo "  3. Update error handling"
echo "  4. Remove manual database Close() calls"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Counter for files migrated
MIGRATED_COUNT=0

# Find all command files (excluding tests and db_*.go files)
find "$COMMANDS_DIR" -name "*.go" -type f | while read -r file; do
    # Skip test files and db_global.go
    if [[ "$file" == *_test.go ]] || [[ "$file" == */db_global.go ]] || [[ "$file" == */db_init.go ]]; then
        continue
    fi

    # Check if file contains old pattern
    if ! grep -q "cli.GetDBPath()\|db.InitDB\|repository.NewDB" "$file"; then
        continue
    fi

    # Check if file already uses cli.GetDB
    if grep -q "cli.GetDB(" "$file"; then
        echo "Skipping: $file (already migrated)"
        continue
    fi

    echo "Processing: $file"

    # Backup original
    cp "$file" "$BACKUP_DIR/$(basename $file).backup"

    # Pattern 1: Standard pattern with cli.GetDBPath()
    # Before:
    #   dbPath, err := cli.GetDBPath()
    #   if err != nil { ... }
    #   database, err := db.InitDB(dbPath)
    #   if err != nil { ... }
    #   repoDb := repository.NewDB(database)
    #   defer repoDb.Close()
    #
    # After:
    #   repoDb, err := cli.GetDB(cmd.Context())
    #   if err != nil {
    #       return fmt.Errorf("failed to get database: %w", err)
    #   }
    #   // Note: Database will be closed automatically by PersistentPostRunE hook

    perl -i -0777 -pe 's/dbPath,\s*err\s*:=\s*cli\.GetDBPath\(\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{\s*\n\s*(?:cli\.Error\([^)]+\)\s*\n\s*)?(?:if\s+cli\.GlobalConfig\.Verbose\s*\{[^}]*\}\s*\n\s*)?(?:os\.Exit\(\d+\)\s*\n\s*)?(?:return[^\n]*\n\s*)?\}\s*\n\s*database,\s*err\s*:=\s*db\.InitDB\(dbPath\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{\s*\n\s*(?:cli\.Error\([^)]+\)\s*\n\s*)?(?:if\s+cli\.GlobalConfig\.Verbose\s*\{[^}]*\}\s*\n\s*)?(?:os\.Exit\(\d+\)\s*\n\s*)?(?:return[^\n]*\n\s*)?\}\s*\n\s*repoDb\s*:=\s*repository\.NewDB\(database\)\s*\n\s*defer\s+repoDb\.Close\(\)/repoDb, err := cli.GetDB(cmd.Context())\n\tif err != nil {\n\t\treturn fmt.Errorf("failed to get database: %w", err)\n\t}\n\t\/\/ Note: Database will be closed automatically by PersistentPostRunE hook/gs' "$file"

    # Pattern 2: Simpler pattern without GetDBPath
    # Before:
    #   database, err := db.InitDB(cli.GlobalConfig.DBPath)
    #   if err != nil { ... }
    #   dbWrapper := repository.NewDB(database)
    #   defer dbWrapper.Close()
    #
    # After:
    #   dbWrapper, err := cli.GetDB(cmd.Context())
    #   if err != nil {
    #       return fmt.Errorf("failed to get database: %w", err)
    #   }
    #   // Note: Database will be closed automatically by PersistentPostRunE hook

    perl -i -0777 -pe 's/database,\s*err\s*:=\s*db\.InitDB\([^)]+\)\s*\n\s*if\s*err\s*!=\s*nil\s*\{\s*\n\s*(?:cli\.Error\([^)]+\)\s*\n\s*)?(?:if\s+cli\.GlobalConfig\.Verbose\s*\{[^}]*\}\s*\n\s*)?(?:os\.Exit\(\d+\)\s*\n\s*)?(?:return[^\n]*\n\s*)?\}\s*\n\s*(repoDb|dbWrapper)\s*:=\s*repository\.NewDB\(database\)\s*\n\s*defer\s+\1\.Close\(\)/$1, err := cli.GetDB(cmd.Context())\n\tif err != nil {\n\t\treturn fmt.Errorf("failed to get database: %w", err)\n\t}\n\t\/\/ Note: Database will be closed automatically by PersistentPostRunE hook/gs' "$file"

    # Pattern 3: Remove standalone defer repoDb.Close() or defer database.Close()
    perl -i -pe 's/^\s*defer\s+(repoDb|dbWrapper|database)\.Close\(\)\s*$/\t\/\/ Note: Database will be closed automatically by PersistentPostRunE hook\n/g' "$file"

    echo "  ✓ Updated: $file"
    MIGRATED_COUNT=$((MIGRATED_COUNT + 1))
done

echo ""
echo "=== Migration Complete ==="
echo ""
echo "Files migrated: $MIGRATED_COUNT"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff internal/cli/commands/"
echo "  2. Run tests: make test"
echo "  3. If tests pass: Continue to Phase 4"
echo "  4. If issues occur: git checkout internal/cli/commands/"
echo ""
echo "Backups stored in: $BACKUP_DIR"
