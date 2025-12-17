# PM CLI Troubleshooting Guide

## Table of Contents

- [Initialization Issues](#initialization-issues)
- [Synchronization Issues](#synchronization-issues)
- [Database Issues](#database-issues)
- [File System Issues](#file-system-issues)
- [Performance Issues](#performance-issues)
- [Common Error Messages](#common-error-messages)

---

## Initialization Issues

### Init Command Hangs in CI/CD

**Symptom**: `pm init` hangs waiting for input in automated environments

**Cause**: Config file exists and init is prompting for overwrite confirmation

**Solution**: Use `--non-interactive` flag
```bash
pm init --non-interactive
```

**Prevention**: Always use `--non-interactive` in scripts and CI/CD pipelines

---

### Database Creation Fails with Permission Denied

**Symptom**:
```
Error: Cannot create database: permission denied
```

**Cause**: No write permission in current directory

**Solution**: Check directory permissions
```bash
ls -la
chmod 755 .  # Make directory writable
pm init
```

**Alternative**: Use custom database path
```bash
mkdir -p data
pm init --db-path=data/tasks.db
```

---

### Folder Creation Fails

**Symptom**:
```
Error: Cannot create folder docs/plan: permission denied
```

**Cause**: No write permission or parent directory doesn't exist

**Solution**:
```bash
# Check permissions
ls -la docs/

# Create parent directory
mkdir -p docs

# Retry init
pm init
```

---

### Templates Not Copied

**Symptom**: `templates/` directory exists but is empty

**Cause**: Init was run without `--force` and templates already existed (skipped)

**Solution**: Re-run with `--force`
```bash
pm init --force
```

---

### Config File Invalid JSON

**Symptom**:
```
Error: Cannot parse config file: invalid JSON
```

**Cause**: Manual edit introduced syntax error

**Solution**: Validate and fix JSON
```bash
# Check JSON syntax
cat .pmconfig.json | jq .

# If invalid, regenerate with force
pm init --force
```

**Example Valid Config**:
```json
{
  "default_epic": "E04",
  "default_agent": "backend",
  "color_enabled": true,
  "json_output": false
}
```

---

## Synchronization Issues

### Sync Finds No Files

**Symptom**:
```
Sync completed:
  Files scanned: 0
```

**Cause**: No task markdown files in expected locations

**Solution**: Verify file organization
```bash
# Check for task files
find . -name "T-*.md" -type f

# Verify folder structure
ls -la docs/plan/
```

**Expected Structure**:
```
docs/plan/<epic>/<feature>/T-<key>.md
```

---

### Tasks Not Imported

**Symptom**: Files scanned but tasks not imported
```
Files scanned: 10
Tasks imported: 0
Warnings: Task references non-existent feature
```

**Cause**: Epic/feature doesn't exist in database

**Solution**: Use `--create-missing` flag
```bash
pm sync --create-missing
```

**Alternative**: Create epic/feature manually
```bash
pm epic create --key=E04 --title="Epic Title"
pm feature create --epic=E04 --key=E04-F07 --title="Feature Title"
pm sync
```

---

### Invalid Frontmatter Errors

**Symptom**:
```
Warning: Invalid frontmatter in docs/plan/E04-cli/E04-F07/task.md
```

**Cause**: YAML syntax error in frontmatter

**Solution**: Fix YAML syntax
```bash
# View file
cat docs/plan/E04-cli/E04-F07/task.md

# Check for:
# - Missing closing ---
# - Invalid indentation
# - Special characters in strings (use quotes)
```

**Valid Frontmatter**:
```yaml
---
key: T-E04-F07-001
title: My task title
description: Task description here
---
```

**Invalid Frontmatter** (missing key):
```yaml
---
title: My task title
---
```

---

### Sync Transaction Rollback

**Symptom**:
```
Error: Database constraint violation
Transaction rolled back. No changes made.
```

**Cause**:
- Duplicate task key
- Invalid foreign key reference
- Database corruption

**Solution**:

1. **Check for duplicates**:
```bash
# Find duplicate task keys
grep -r "^key: T-E04-F07-001" docs/plan/
```

2. **Verify database integrity**:
```bash
sqlite3 shark-tasks.db "PRAGMA integrity_check;"
```

3. **Review sync output** for specific error

4. **Fix data issue** and retry

---

### Conflicts Not Resolving

**Symptom**: Conflicts reported but database not updated

**Cause**: Using `--strategy=database-wins`

**Solution**: Use `file-wins` strategy
```bash
pm sync --strategy=file-wins
# or shorthand:
pm sync --force
```

**Verify Strategy**:
```bash
# Preview with dry-run
pm sync --strategy=file-wins --dry-run
```

---

### Sync Ignoring File Changes

**Symptom**: File updated but database unchanged after sync

**Cause**: File and database already match (no conflict detected)

**Possible Issues**:
1. Wrong file edited
2. Frontmatter not changed
3. File cached/not saved

**Solution**:
```bash
# Verify file content
cat docs/plan/E04-cli/E04-F07/T-E04-F07-001.md

# Check database value
sqlite3 shark-tasks.db "SELECT key, title FROM tasks WHERE key='T-E04-F07-001';"

# Force sync
pm sync --force
```

---

## Database Issues

### Database Locked

**Symptom**:
```
Error: database is locked
```

**Cause**: Another process has database open or previous connection not closed

**Solution**:

1. **Check for other processes**:
```bash
lsof shark-tasks.db
# or
fuser shark-tasks.db
```

2. **Kill blocking process** (if safe):
```bash
kill <PID>
```

3. **Wait and retry**

4. **Verify WAL mode enabled**:
```bash
sqlite3 shark-tasks.db "PRAGMA journal_mode;"
# Should output: wal
```

5. **Enable WAL mode if not enabled**:
```bash
sqlite3 shark-tasks.db "PRAGMA journal_mode=WAL;"
```

---

### Database Corruption

**Symptom**:
```
Error: database disk image is malformed
```

**Cause**:
- Disk I/O error
- System crash during write
- File system corruption

**Solution**:

1. **Check integrity**:
```bash
sqlite3 shark-tasks.db "PRAGMA integrity_check;"
```

2. **Attempt repair** (create backup first):
```bash
# Backup
cp shark-tasks.db shark-tasks.db.backup

# Dump and restore
sqlite3 shark-tasks.db ".dump" | sqlite3 new-tasks.db
mv new-tasks.db shark-tasks.db
```

3. **Restore from backup** if repair fails:
```bash
cp shark-tasks.db.backup shark-tasks.db
```

4. **Re-sync from files**:
```bash
# If database unrecoverable, start fresh
rm shark-tasks.db
pm init
pm sync --create-missing
```

---

### Foreign Key Constraint Violation

**Symptom**:
```
Error: FOREIGN KEY constraint failed
```

**Cause**: Task references non-existent feature

**Solution**: Create missing epic/feature
```bash
pm sync --create-missing
```

**Or manually**:
```bash
pm epic create --key=E04 --title="Epic"
pm feature create --epic=E04 --key=E04-F07 --title="Feature"
pm sync
```

---

## File System Issues

### File Path Too Long

**Symptom**:
```
Error: File name too long
```

**Cause**: Deep directory nesting on systems with path length limits

**Solution**: Shorten epic/feature names or reorganize structure
```bash
# Bad (too long)
docs/plan/E04-very-long-epic-name-here/E04-F07-very-long-feature-name/T-E04-F07-001.md

# Good (shorter)
docs/plan/E04-epic/E04-F07-feature/T-E04-F07-001.md
```

---

### Permission Denied Reading Files

**Symptom**:
```
Error: Permission denied reading task file
```

**Cause**: File not readable

**Solution**: Fix file permissions
```bash
# Check permissions
ls -la docs/plan/E04-cli/E04-F07/T-E04-F07-001.md

# Fix permissions
chmod 644 docs/plan/E04-cli/E04-F07/T-E04-F07-001.md

# Or recursively
chmod -R 644 docs/plan/**/*.md
```

---

### Symlink Issues

**Symptom**:
```
Warning: Skipping symlink: /path/to/link
```

**Cause**: Sync rejects symlinks for security

**Solution**:
- This is expected behavior (security feature)
- Copy actual file instead of using symlink
```bash
cp /source/task.md docs/plan/E04-cli/E04-F07/task.md
```

---

## Performance Issues

### Sync Takes Too Long

**Symptom**: Sync operation exceeds 10 seconds for 100 files

**Possible Causes**:
1. Slow disk I/O
2. Large files
3. Many conflicts
4. Database not optimized

**Solutions**:

1. **Optimize database**:
```bash
sqlite3 shark-tasks.db "VACUUM;"
sqlite3 shark-tasks.db "ANALYZE;"
```

2. **Check file sizes**:
```bash
find docs/plan -name "*.md" -size +1M
```

3. **Use selective sync**:
```bash
pm sync --folder=docs/plan/E04-current-feature
```

4. **Enable WAL mode**:
```bash
sqlite3 shark-tasks.db "PRAGMA journal_mode=WAL;"
```

---

### Init Takes Too Long

**Symptom**: Init exceeds 5 seconds

**Possible Causes**:
1. Slow disk
2. Large template files
3. Many folders to create

**Solutions**:

1. **Use faster storage** (SSD vs HDD)

2. **Check disk space**:
```bash
df -h .
```

3. **Minimize templates** (if custom templates are large)

---

### High Memory Usage

**Symptom**: PM CLI consuming excessive memory

**Cause**: Processing many large files at once

**Solution**: Use selective sync
```bash
# Instead of syncing all files
pm sync

# Sync folder by folder
pm sync --folder=docs/plan/E04-epic/E04-F07-feature
pm sync --folder=docs/plan/E04-epic/E04-F08-feature
```

---

## Common Error Messages

### "Task key already exists"

**Full Error**:
```
Error: Task key T-E04-F07-001 already exists in database
```

**Cause**: Attempting to import task with duplicate key

**Solution**:
```bash
# Check existing task
pm task show T-E04-F07-001

# If duplicate file, rename or remove it
mv duplicate.md duplicate.md.bak

# Or update with sync
pm sync --strategy=file-wins
```

---

### "Missing required field: key"

**Full Error**:
```
Warning: Missing required field 'key' in frontmatter
File: docs/plan/E04-cli/E04-F07/task.md
```

**Cause**: Task file missing `key` field in frontmatter

**Solution**: Add key field
```yaml
---
key: T-E04-F07-001  ← Add this
title: My task
---
```

---

### "Cannot determine feature for task"

**Full Error**:
```
Warning: Cannot determine feature for task T-E99-F01-001
Please specify --epic=<key> --feature=<key> or use --create-missing
```

**Cause**:
- Task file not in feature folder structure
- Epic/feature doesn't exist in database

**Solution**:

**Option 1**: Use `--create-missing`
```bash
pm sync --create-missing
```

**Option 2**: Create epic/feature first
```bash
pm epic create --key=E99 --title="New Epic"
pm feature create --epic=E99 --key=E99-F01 --title="New Feature"
pm sync
```

**Option 3**: Move file to correct location
```bash
mkdir -p docs/plan/E99-epic/E99-F01-feature
mv orphan-task.md docs/plan/E99-epic/E99-F01-feature/T-E99-F01-001.md
```

---

### "Context canceled"

**Full Error**:
```
Error: context canceled
```

**Cause**:
- User pressed Ctrl+C
- Operation timeout
- System shutdown

**Solution**:
- This is expected if you intentionally canceled
- Check system resources if unexpected
- Retry operation

```bash
# Retry with longer timeout (if timeout issue)
pm sync  # No timeout flag in current version
```

---

### "YAML parse error"

**Full Error**:
```
Warning: YAML parse error in file: yaml: line 3: could not find expected ':'
```

**Cause**: Invalid YAML syntax

**Solution**: Fix YAML formatting

**Common Issues**:

1. **Missing colon**:
```yaml
# Bad
key T-E04-F07-001

# Good
key: T-E04-F07-001
```

2. **Invalid indentation**:
```yaml
# Bad
---
  key: T-E04-F07-001
title: Task
---

# Good
---
key: T-E04-F07-001
title: Task
---
```

3. **Unquoted special characters**:
```yaml
# Bad
title: Task: Implementation

# Good
title: "Task: Implementation"
```

---

## Getting Help

If you encounter an issue not covered here:

1. **Check verbose output**:
```bash
pm sync --dry-run  # See detailed preview
```

2. **Check database directly**:
```bash
sqlite3 shark-tasks.db
sqlite> .tables
sqlite> SELECT * FROM tasks WHERE key='T-E04-F07-001';
```

3. **Enable debug logging** (if available):
```bash
export PM_DEBUG=1
pm sync
```

4. **Check file with less**:
```bash
less docs/plan/E04-cli/E04-F07/T-E04-F07-001.md
```

5. **Validate JSON output**:
```bash
pm sync --json | jq .
```

6. **Review recent commits**:
```bash
git log --oneline -10
git diff HEAD~1 -- docs/plan/
```

---

## Preventive Measures

### 1. Regular Backups

```bash
# Backup database daily
cp shark-tasks.db backups/shark-tasks-$(date +%Y%m%d).db
```

### 2. Use Dry-Run

```bash
# Always preview before sync
pm sync --dry-run
pm sync
```

### 3. Commit Before Sync

```bash
git add .
git commit -m "Before sync"
pm sync
```

### 4. Keep Clean Frontmatter

Only include necessary fields:
```yaml
---
key: T-E04-F07-001
title: Task title
description: Task description
---
```

### 5. Enable WAL Mode

```bash
sqlite3 shark-tasks.db "PRAGMA journal_mode=WAL;"
```

### 6. Validate JSON Config

```bash
cat .pmconfig.json | jq .
```

---

## See Also

- [Initialization Guide](user-guide/initialization.md)
- [Synchronization Guide](user-guide/synchronization.md)
- [CLI Documentation](CLI.md)
