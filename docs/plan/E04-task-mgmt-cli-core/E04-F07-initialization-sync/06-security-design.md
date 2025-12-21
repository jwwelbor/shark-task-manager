# Security Design: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: security-architect

## Purpose

This document defines security considerations, threat models, and mitigation strategies for initialization and synchronization operations.

---

## Threat Model

### Assets to Protect

1. **Database File** (`shark-tasks.db`): Contains all task, epic, and feature data
2. **Config File** (`.pmconfig.json`): May contain sensitive configuration
3. **Task Files**: Markdown files with frontmatter metadata
4. **File System Integrity**: Prevent malicious file writes
5. **Database Integrity**: Prevent SQL injection, data corruption

### Threat Actors

1. **Malicious File**: Crafted markdown file with exploit attempts
2. **Path Traversal Attack**: File paths attempting to escape allowed directories
3. **SQL Injection**: Malicious frontmatter data targeting database
4. **Symlink Attack**: Symlinks pointing to sensitive files
5. **Race Condition**: Concurrent modifications during sync

---

## Security Controls

### 1. File System Security

#### Database File Permissions

**Requirement**: Database file must be readable/writable only by owner (Unix: 600).

**Implementation**:
```go
import (
    "os"
    "runtime"
)

func setDatabasePermissions(dbPath string) error {
    if runtime.GOOS == "windows" {
        // Windows uses ACLs, not POSIX permissions
        // Default file creation permissions are sufficient
        return nil
    }

    // Set permissions to 0600 (read/write owner only)
    if err := os.Chmod(dbPath, 0600); err != nil {
        return fmt.Errorf("failed to set database permissions: %w", err)
    }

    return nil
}
```

**Verification**:
```bash
# Unix systems
ls -l shark-tasks.db
# Expected: -rw------- 1 user user 12345 Dec 16 10:00 shark-tasks.db
```

#### Config File Permissions

**Requirement**: Config file should be readable/writable only by owner (Unix: 600).

**Implementation**:
```go
func writeConfigFile(path string, data []byte) error {
    // Write to temp file first
    tmpPath := path + ".tmp"
    if err := ioutil.WriteFile(tmpPath, data, 0600); err != nil {
        return err
    }

    // Atomic rename
    if err := os.Rename(tmpPath, path); err != nil {
        os.Remove(tmpPath)
        return err
    }

    return nil
}
```

#### File Path Validation

**Threat**: Path traversal attack (e.g., `../../../etc/passwd` in file_path).

**Mitigation**:
```go
func validateFilePath(filePath string) error {
    // Convert to absolute path
    absPath, err := filepath.Abs(filePath)
    if err != nil {
        return fmt.Errorf("invalid file path: %w", err)
    }

    // Define allowed root directories
    allowedRoots := []string{
        filepath.Join(workingDir, "docs/plan"),
        filepath.Join(workingDir, "docs/tasks"),
        filepath.Join(workingDir, "templates"),
    }

    // Check if path is within allowed roots
    for _, root := range allowedRoots {
        if strings.HasPrefix(absPath, root) {
            return nil
        }
    }

    return fmt.Errorf("file path outside allowed directories: %s", absPath)
}
```

**Usage**:
```go
// Validate before storing in database
if err := validateFilePath(taskData.FilePath); err != nil {
    return fmt.Errorf("security: %w", err)
}
```

#### Symlink Handling

**Threat**: Symlink pointing to sensitive file (e.g., `/etc/shadow`).

**Mitigation**:
```go
func validateFileIsRegular(filePath string) error {
    info, err := os.Lstat(filePath)  // Lstat doesn't follow symlinks
    if err != nil {
        return err
    }

    if info.Mode()&os.ModeSymlink != 0 {
        return fmt.Errorf("symlinks are not allowed: %s", filePath)
    }

    if !info.Mode().IsRegular() {
        return fmt.Errorf("not a regular file: %s", filePath)
    }

    return nil
}
```

**Alternative Approach** (follow symlinks but validate target):
```go
func validateFileTarget(filePath string) error {
    // Stat follows symlinks
    info, err := os.Stat(filePath)
    if err != nil {
        return err
    }

    // Get real path after following symlinks
    realPath, err := filepath.EvalSymlinks(filePath)
    if err != nil {
        return err
    }

    // Validate real path is in allowed directories
    return validateFilePath(realPath)
}
```

**Recommendation**: Use Lstat and reject symlinks for MVP. Support symlinks in future version if needed.

---

### 2. Database Security

#### SQL Injection Prevention

**Threat**: Malicious frontmatter data used in SQL queries.

**Mitigation**: Always use parameterized queries (Go database/sql automatically handles this).

**Example**:
```go
// SAFE: Parameterized query
query := "SELECT * FROM tasks WHERE key = ?"
row := db.QueryRow(query, taskKey)

// UNSAFE: String concatenation (NEVER do this)
query := fmt.Sprintf("SELECT * FROM tasks WHERE key = '%s'", taskKey)
```

**Enforcement**: All repository methods use parameterized queries via `database/sql` package.

#### Foreign Key Enforcement

**Requirement**: Prevent orphaned tasks (tasks without valid feature).

**Implementation**: SQLite foreign key constraints (already enabled in db.InitDB).

```sql
CREATE TABLE tasks (
    ...
    feature_id INTEGER NOT NULL,
    FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
);
```

**Verification**:
```go
// Verify foreign keys are enabled
var fkEnabled int
db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
if fkEnabled != 1 {
    return fmt.Errorf("foreign keys not enabled")
}
```

#### Transaction Isolation

**Requirement**: Prevent race conditions during concurrent sync operations.

**Mitigation**: Use WAL mode (Write-Ahead Logging) for better concurrency.

```go
// Already enabled in db.InitDB
db.Exec("PRAGMA journal_mode = WAL")
```

**WAL Mode Benefits**:
- Readers don't block writers
- Writers don't block readers
- Better crash recovery

**Caution**: Only one sync operation should run at a time. Use filesystem lock if needed.

---

### 3. YAML Parsing Security

#### YAML Injection

**Threat**: Malicious YAML that attempts code execution or resource exhaustion.

**Mitigation**: `gopkg.in/yaml.v3` is safe for read-only parsing (no code execution).

**Additional Safeguards**:
```go
// Set maximum YAML size (prevent DoS)
const maxYAMLSize = 1 * 1024 * 1024  // 1 MB

func parseYAMLFrontmatter(data []byte) (*TaskMetadata, error) {
    if len(data) > maxYAMLSize {
        return nil, fmt.Errorf("YAML frontmatter too large: %d bytes", len(data))
    }

    var metadata TaskMetadata
    if err := yaml.Unmarshal(data, &metadata); err != nil {
        return nil, fmt.Errorf("invalid YAML: %w", err)
    }

    return &metadata, nil
}
```

#### Unexpected Fields

**Threat**: Malicious YAML with unexpected fields that exploit parser bugs.

**Mitigation**: Use strict unmarshaling (reject unknown fields).

```go
func parseTaskMetadataStrict(data []byte) (*TaskMetadata, error) {
    dec := yaml.NewDecoder(bytes.NewReader(data))
    dec.KnownFields(true)  // Reject unknown fields

    var metadata TaskMetadata
    if err := dec.Decode(&metadata); err != nil {
        return nil, fmt.Errorf("invalid YAML: %w", err)
    }

    return &metadata, nil
}
```

#### Field Validation

**Threat**: Malicious values in expected fields (e.g., extremely long strings).

**Mitigation**: Validate all fields after parsing.

```go
func validateTaskMetadata(metadata *TaskMetadata) error {
    // Validate key format
    if !regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`).MatchString(metadata.Key) {
        return fmt.Errorf("invalid task key format: %s", metadata.Key)
    }

    // Validate title length
    if len(metadata.Title) > 500 {
        return fmt.Errorf("title too long: %d characters", len(metadata.Title))
    }

    // Validate description length
    if metadata.Description != nil && len(*metadata.Description) > 10000 {
        return fmt.Errorf("description too long: %d characters", len(*metadata.Description))
    }

    // Validate file_path
    if err := validateFilePath(metadata.FilePath); err != nil {
        return err
    }

    return nil
}
```

---

### 4. Atomic Operations

#### Atomic File Writes

**Requirement**: Prevent partial writes or corruption during config file creation.

**Implementation**: Write to temp file, then atomic rename.

```go
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
    // Create temp file in same directory
    dir := filepath.Dir(path)
    tmpFile, err := ioutil.TempFile(dir, ".tmp-*")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    tmpPath := tmpFile.Name()
    defer os.Remove(tmpPath)  // Cleanup on error

    // Write data
    if _, err := tmpFile.Write(data); err != nil {
        tmpFile.Close()
        return fmt.Errorf("failed to write temp file: %w", err)
    }

    // Sync to disk
    if err := tmpFile.Sync(); err != nil {
        tmpFile.Close()
        return fmt.Errorf("failed to sync temp file: %w", err)
    }

    tmpFile.Close()

    // Set permissions
    if err := os.Chmod(tmpPath, perm); err != nil {
        return fmt.Errorf("failed to set permissions: %w", err)
    }

    // Atomic rename (on Unix, Windows has caveats)
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("failed to rename temp file: %w", err)
    }

    return nil
}
```

#### Atomic Database Operations

**Requirement**: All sync operations succeed or rollback (no partial updates).

**Implementation**: Use transaction with deferred rollback.

```go
func (e *SyncEngine) Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error) {
    // Begin transaction
    tx, err := e.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()  // Safety net (no-op if committed)

    // ... perform all database operations ...

    // Commit transaction (all-or-nothing)
    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return report, nil
}
```

---

### 5. Input Validation

#### Task Key Validation

**Requirement**: Prevent malicious task keys.

**Implementation**:
```go
func validateTaskKey(key string) error {
    // Enforce strict format: T-E##-F##-###
    pattern := regexp.MustCompile(`^T-E\d{2}-F\d{2}-\d{3}$`)
    if !pattern.MatchString(key) {
        return fmt.Errorf("invalid task key format: %s", key)
    }

    return nil
}
```

#### Epic/Feature Key Validation

**Requirement**: Prevent malicious epic/feature keys.

**Implementation**:
```go
func validateEpicKey(key string) error {
    pattern := regexp.MustCompile(`^E\d{2}$`)
    if !pattern.MatchString(key) {
        return fmt.Errorf("invalid epic key format: %s", key)
    }
    return nil
}

func validateFeatureKey(key string) error {
    pattern := regexp.MustCompile(`^E\d{2}-F\d{2}$`)
    if !pattern.MatchString(key) {
        return fmt.Errorf("invalid feature key format: %s", key)
    }
    return nil
}
```

#### Folder Path Validation

**Requirement**: Prevent path traversal in --folder flag.

**Implementation**:
```go
func validateFolderPath(folderPath string) error {
    // Convert to absolute path
    absPath, err := filepath.Abs(folderPath)
    if err != nil {
        return fmt.Errorf("invalid folder path: %w", err)
    }

    // Check if path exists and is a directory
    info, err := os.Stat(absPath)
    if err != nil {
        return fmt.Errorf("folder does not exist: %s", folderPath)
    }

    if !info.IsDir() {
        return fmt.Errorf("not a directory: %s", folderPath)
    }

    // Validate within working directory
    wd, _ := os.Getwd()
    if !strings.HasPrefix(absPath, wd) {
        return fmt.Errorf("folder path outside working directory: %s", folderPath)
    }

    return nil
}
```

---

### 6. Backup and Recovery

#### Pre-Sync Backup (--backup flag)

**Requirement**: Create backup before sync to enable rollback.

**Implementation**:
```go
func createDatabaseBackup(dbPath string) (string, error) {
    // Generate backup filename with timestamp
    timestamp := time.Now().Format("2006-01-02T15-04-05")
    backupPath := fmt.Sprintf("%s.backup.%s", dbPath, timestamp)

    // Check if backup already exists
    if _, err := os.Stat(backupPath); err == nil {
        return "", fmt.Errorf("backup already exists: %s", backupPath)
    }

    // Copy database file
    if err := copyFile(dbPath, backupPath); err != nil {
        return "", fmt.Errorf("failed to create backup: %w", err)
    }

    // Set same permissions as original
    info, err := os.Stat(dbPath)
    if err != nil {
        return backupPath, nil  // Return backup path even if chmod fails
    }

    os.Chmod(backupPath, info.Mode())

    return backupPath, nil
}

func copyFile(src, dst string) error {
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()

    destFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destFile.Close()

    if _, err := io.Copy(destFile, sourceFile); err != nil {
        return err
    }

    return destFile.Sync()
}
```

**Usage**:
```bash
# Create backup before sync
shark sync --backup

# If sync fails, restore from backup
cp shark-tasks.db.backup.2025-12-16T10-30-00 shark-tasks.db
```

#### Transaction Rollback

**Automatic Rollback**: If any error occurs during sync, transaction is rolled back automatically.

```go
defer tx.Rollback()  // Safety net: no-op if tx.Commit() succeeds

// If any error occurs before commit, rollback happens automatically
if err := someOperation(); err != nil {
    return err  // Deferred rollback executes
}

tx.Commit()  // Only commit if all operations succeed
```

---

### 7. Denial of Service (DoS) Prevention

#### File Limits

**Threat**: Attacker creates millions of task files to exhaust resources.

**Mitigation**:
```go
const (
    maxFilesPerSync = 10000  // Reasonable limit for single sync
    maxFileSize     = 1024 * 1024  // 1 MB per file
)

func (s *FileScanner) Scan(rootPath string) ([]TaskFileInfo, error) {
    var files []TaskFileInfo

    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        // Check file count limit
        if len(files) >= maxFilesPerSync {
            return fmt.Errorf("too many files: limit is %d", maxFilesPerSync)
        }

        // Check file size
        if info.Size() > maxFileSize {
            log.Warnf("Skipping large file: %s (%d bytes)", path, info.Size())
            return nil
        }

        // ... rest of scanning logic ...
    })

    return files, err
}
```

#### Context Timeout

**Threat**: Sync operation hangs indefinitely.

**Mitigation**: Always use context with timeout.

```go
// In CLI command
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

report, err := engine.Sync(ctx, opts)
```

#### Resource Limits

**Mitigation**: Set reasonable limits on string lengths, array sizes, etc.

```go
const (
    maxTitleLength       = 500
    maxDescriptionLength = 10000
    maxDependenciesCount = 50
)
```

---

### 8. Audit Logging

#### Sync Operations Logging

**Requirement**: Log all sync operations for audit trail.

**Implementation**:
```go
func (e *SyncEngine) Sync(ctx context.Context, opts SyncOptions) (*SyncReport, error) {
    // Log sync start
    log.Infof("Sync started: folder=%s strategy=%s dry_run=%v",
        opts.FolderPath, opts.Strategy, opts.DryRun)

    // ... perform sync ...

    // Log sync completion
    log.Infof("Sync completed: imported=%d updated=%d conflicts=%d",
        report.TasksImported, report.TasksUpdated, report.ConflictsResolved)

    return report, nil
}
```

#### Task History Records

**Requirement**: Create history records for all task changes.

**Implementation** (already in design):
```sql
INSERT INTO task_history (task_id, old_status, new_status, agent, notes)
VALUES (?, ?, ?, 'sync', 'Updated from file during sync: title, file_path');
```

---

### 9. Secure Defaults

#### Default Config

**Secure Defaults**:
```json
{
  "default_epic": null,
  "default_agent": null,
  "color_enabled": true,
  "json_output": false
}
```

**No Sensitive Defaults**: Config file does not contain passwords, tokens, or API keys.

#### Default Permissions

- Database: 0600 (read/write owner only)
- Config: 0644 (read for all, write for owner)
- Folders: 0755 (standard directory permissions)

---

## Security Checklist

### Implementation Phase

- [ ] Set database file permissions to 0600 (Unix)
- [ ] Validate all file paths before use
- [ ] Reject symlinks or validate symlink targets
- [ ] Use parameterized queries for all database operations
- [ ] Verify foreign key enforcement is enabled
- [ ] Implement atomic file writes (temp + rename)
- [ ] Use transactions for all sync operations
- [ ] Validate task keys, epic keys, feature keys
- [ ] Limit YAML frontmatter size
- [ ] Limit maximum files per sync
- [ ] Implement context timeout for all operations
- [ ] Log all sync operations for audit trail
- [ ] Create task history records for all changes

### Testing Phase

- [ ] Test path traversal attack prevention
- [ ] Test symlink handling
- [ ] Test YAML injection attempts
- [ ] Test file size limits
- [ ] Test file count limits
- [ ] Test transaction rollback on error
- [ ] Test atomic file writes (simulate crash)
- [ ] Test concurrent sync operations
- [ ] Verify foreign key constraints work

### Deployment Phase

- [ ] Document security best practices
- [ ] Document backup/restore procedure
- [ ] Configure log retention policy
- [ ] Review file permissions on production systems

---

## Security Monitoring

### Metrics to Track

1. **Failed Sync Operations**: Track count and reasons
2. **Invalid File Paths**: Track attempts to access unauthorized paths
3. **YAML Parse Errors**: Track malformed YAML attempts
4. **Transaction Rollbacks**: Track database errors
5. **File Size Violations**: Track attempts to sync large files

### Alerts

- Alert on multiple failed sync operations (potential attack)
- Alert on path traversal attempts
- Alert on transaction rollback rate > threshold

---

## Incident Response

### Security Incident: Unauthorized File Access

**Detection**: Log shows path traversal attempt
**Response**:
1. Review logs for pattern of attacks
2. Verify file path validation is working
3. Check for compromised files
4. Restore from backup if needed

### Security Incident: Database Corruption

**Detection**: PRAGMA integrity_check fails
**Response**:
1. Stop all sync operations
2. Restore from latest backup
3. Investigate cause (malicious input, disk failure, etc.)
4. Re-run sync after validation

---

**Document Complete**: 2025-12-16
**Next Document**: 08-implementation-phases.md (coordinator creates)
