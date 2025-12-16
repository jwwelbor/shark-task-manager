# Security Design: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: security-architect

## Purpose

This document analyzes security considerations for the database layer, including SQL injection prevention, file permissions, data privacy, access control, audit logging, and vulnerability mitigation. While this is a single-developer CLI tool, security best practices ensure data integrity and protect against accidental or malicious corruption.

---

## Threat Model

### Assets to Protect

1. **Project Database** (`project.db`)
   - Contains all project state (epics, features, tasks, history)
   - Single source of truth for project management
   - Critical for agent and developer workflows

2. **Task Markdown Files**
   - Contain detailed task descriptions and context
   - Referenced by database (`tasks.file_path`)
   - May contain sensitive information (API endpoints, architecture details)

3. **Application Integrity**
   - CLI tool must behave predictably
   - Database schema must remain consistent
   - Migrations must not corrupt data

### Threat Actors

| Actor | Profile | Motivation | Capabilities |
|-------|---------|------------|--------------|
| **Malicious Input** | Crafted CLI arguments or file content | Cause crashes, inject SQL, corrupt data | Limited to user context |
| **Accidental User Error** | Developer mistakes | Unintended data loss | Full user permissions |
| **Filesystem Attacks** | Unauthorized file access | Read project data, modify database | OS-level access required |
| **Dependency Vulnerabilities** | Compromised Python packages | Code execution, data exfiltration | Depends on vulnerability |

### Attack Scenarios

1. **SQL Injection**
   - Attacker crafts task title or key to inject SQL
   - Result: Data leakage, corruption, or deletion

2. **Path Traversal**
   - Attacker specifies `file_path` like `../../../../etc/passwd`
   - Result: Access to files outside project directory

3. **Database Corruption**
   - Attacker modifies database file directly
   - Result: Application crashes, data loss

4. **Privilege Escalation**
   - Database file has overly permissive permissions
   - Result: Other users can read/modify project data

5. **Dependency Vulnerabilities**
   - Vulnerability in SQLAlchemy, Alembic, or other dependencies
   - Result: Remote code execution, data exfiltration

---

## Security Controls

### 1. SQL Injection Prevention

**Threat**: Malicious input in task titles, keys, or filters could inject SQL commands.

**Control**: **Parameterized Queries Only**

**Implementation**:
```python
# SAFE: SQLAlchemy ORM automatically parameterizes
stmt = select(Task).where(Task.status == user_input)
session.execute(stmt)

# Generated SQL (parameterized):
# SELECT * FROM tasks WHERE status = ?
# Parameters: ['todo']

# UNSAFE: Never do this
# session.execute(f"SELECT * FROM tasks WHERE status = '{user_input}'")
```

**Enforcement**:
- All queries use SQLAlchemy ORM (mandatory)
- No raw SQL strings with user input concatenation
- Code review checks for `execute(f"...")` patterns
- mypy type checking prevents accidental string concatenation

**Validation Layer**:
```python
# Validate before database operations
def create_task(key: str, title: str, ...):
    validate_task_key(key)  # Reject malformed keys
    validate_task_status(status)  # Reject invalid enums

    task = Task(key=key, title=title, ...)
    session.add(task)  # SQLAlchemy handles parameterization
```

**Risk Level**: **Low** (fully mitigated by ORM)

---

### 2. File Permission Controls

**Threat**: Database file readable/writable by other users on shared systems.

**Control**: **Restrictive File Permissions (600)**

**Implementation**:
```python
import os
import stat
from pathlib import Path

def set_secure_permissions(db_path: Path) -> None:
    """Set database file permissions to owner-only (600)"""
    if os.name != 'nt':  # Unix systems only
        # 600 = owner read/write only
        os.chmod(db_path, stat.S_IRUSR | stat.S_IWUSR)

        # Also set permissions on WAL and shared memory files
        wal_path = db_path.with_suffix(".db-wal")
        shm_path = db_path.with_suffix(".db-shm")

        if wal_path.exists():
            os.chmod(wal_path, stat.S_IRUSR | stat.S_IWUSR)
        if shm_path.exists():
            os.chmod(shm_path, stat.S_IRUSR | stat.S_IWUSR)
```

**Automatic Application**:
- Set permissions immediately after database creation
- Re-apply after migrations
- Verify on application startup

**Windows Considerations**:
- Windows uses different permission model (ACLs)
- Not enforced on Windows (single-user systems typically)
- Document recommendation to use user-specific project directories

**Risk Level**: **Medium** (mitigated for Unix, accepted risk on Windows)

---

### 3. Input Validation

**Threat**: Malformed input causes database constraint violations, crashes, or unexpected behavior.

**Control**: **Multi-Layer Validation**

**Validation Layers**:

1. **Application Layer** (before database):
   ```python
   # Validate key format
   validate_task_key(key)  # Regex pattern check

   # Validate enum values
   validate_task_status(status)  # Check against allowed values

   # Validate ranges
   validate_task_priority(priority)  # 1-10 only

   # Validate JSON
   validate_depends_on(depends_on)  # Valid JSON array
   ```

2. **Database Layer** (constraints):
   ```sql
   -- CHECK constraints enforce rules
   CHECK (status IN ('todo', 'in_progress', 'blocked', ...))
   CHECK (priority >= 1 AND priority <= 10)

   -- Foreign key constraints prevent orphans
   FOREIGN KEY (feature_id) REFERENCES features(id)
   ```

3. **Type Layer** (Python type hints + mypy):
   ```python
   def create_task(status: str, priority: int) -> Task:
       # mypy enforces correct types at compile time
   ```

**Sanitization**:
- No sanitization needed for database (parameterized queries)
- Markdown content NOT sanitized (stored as-is, rendered securely by viewer)

**Risk Level**: **Low** (multi-layer defense in depth)

---

### 4. Path Traversal Prevention

**Threat**: `file_path` field could reference files outside project directory.

**Control**: **Path Validation and Sandboxing**

**Implementation**:
```python
from pathlib import Path

def validate_file_path(file_path: str, project_root: Path) -> Path:
    """
    Validate that file path is within project directory.

    Args:
        file_path: Absolute path to validate
        project_root: Project root directory

    Returns:
        Validated Path object

    Raises:
        ValidationError: If path is outside project or uses traversal
    """
    try:
        path = Path(file_path).resolve()
        root = project_root.resolve()

        # Check if path is within project directory
        if not path.is_relative_to(root):
            raise ValidationError(
                f"File path must be within project directory: {file_path}"
            )

        # Additional check for .. traversal attempts
        if ".." in file_path:
            raise ValidationError(
                f"File path contains invalid traversal: {file_path}"
            )

        return path

    except (ValueError, OSError) as e:
        raise ValidationError(f"Invalid file path: {e}")
```

**Usage in Repository**:
```python
def create_task(..., file_path: str | None = None):
    if file_path:
        # Validate before storing
        validated_path = validate_file_path(file_path, project_root)
        task.file_path = str(validated_path)
```

**Risk Level**: **Low** (mitigated by validation)

---

### 5. Data Privacy

**Threat**: Sensitive information logged or leaked through error messages.

**Control**: **No Sensitive Data Logging**

**Implementation**:

**Logging Policy**:
```python
import logging

logger = logging.getLogger("pm.database")

# GOOD: Log operations without sensitive data
logger.info(f"Created task: {task.key}")

# BAD: Don't log full task content (may contain sensitive descriptions)
# logger.debug(f"Task data: {task.to_dict()}")

# GOOD: Log errors without query parameters
logger.error("Failed to create task", exc_info=True)

# BAD: Don't log SQL with user input
# logger.debug(f"Executing: SELECT * WHERE title = '{user_input}'")
```

**SQL Query Logging**:
```python
# Disable in production
engine = create_engine(
    database_url,
    echo=False,  # No SQL logging in production
    echo_pool=False
)

# Enable only in development
if os.getenv("ENV") == "development":
    logging.getLogger('sqlalchemy.engine').setLevel(logging.INFO)
```

**Error Messages**:
```python
# GOOD: Generic error message for users
raise ValidationError("Invalid task key format")

# BAD: Don't expose internal details
# raise ValidationError(f"Regex {PATTERN} failed for input {user_input}")
```

**Risk Level**: **Low** (controlled through logging configuration)

---

### 6. Database Integrity Protection

**Threat**: Database file modified directly, bypassing application constraints.

**Control**: **Integrity Checks and Backups**

**Implementation**:

**Integrity Check on Startup**:
```python
def check_database_integrity() -> bool:
    """Run SQLite integrity check"""
    with session_factory.get_session() as session:
        result = session.execute(text("PRAGMA integrity_check"))
        status = result.scalar()

        if status != "ok":
            logger.error(f"Database integrity check failed: {status}")
            return False

        return True

# On application startup
if not check_database_integrity():
    print("ERROR: Database is corrupted. Restore from backup.")
    sys.exit(2)
```

**Backup Strategy**:
```python
from datetime import datetime
import shutil

def backup_database(db_path: Path, backup_dir: Path) -> Path:
    """Create timestamped database backup"""
    timestamp = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
    backup_name = f"project_backup_{timestamp}.db"
    backup_path = backup_dir / backup_name

    # Copy database file (SQLite safe for file copy when not in use)
    shutil.copy2(db_path, backup_path)

    # Set restrictive permissions on backup
    if os.name != 'nt':
        os.chmod(backup_path, stat.S_IRUSR | stat.S_IWUSR)

    logger.info(f"Database backed up to: {backup_path}")
    return backup_path
```

**Recommended Backup Schedule**:
- Before schema migrations (automatic)
- Daily automated backup (optional)
- Manual backup before major operations

**Risk Level**: **Medium** (mitigated by integrity checks + backups)

---

### 7. Foreign Key Enforcement

**Threat**: Orphaned records (tasks without features, features without epics).

**Control**: **SQLite Foreign Keys Enabled**

**Implementation**:
```python
# Enable foreign keys on every connection
@event.listens_for(engine, "connect")
def set_sqlite_pragma(dbapi_conn, connection_record):
    cursor = dbapi_conn.cursor()
    cursor.execute("PRAGMA foreign_keys=ON")  # Critical!
    cursor.close()
```

**Verification**:
```python
def verify_foreign_keys_enabled() -> bool:
    """Verify foreign keys are enabled"""
    with session_factory.get_session() as session:
        result = session.execute(text("PRAGMA foreign_keys"))
        return result.scalar() == 1

# On startup
if not verify_foreign_keys_enabled():
    logger.error("Foreign keys not enabled!")
    raise DatabaseError("Database configuration error: foreign keys disabled")
```

**Risk Level**: **Low** (enforced at connection level)

---

### 8. Schema Version Validation

**Threat**: Application uses outdated schema, causing errors or data corruption.

**Control**: **Schema Version Checks**

**Implementation**:
```python
EXPECTED_SCHEMA_VERSION = "001"

def validate_schema_version() -> None:
    """Validate database schema version matches application"""
    current_version = session_factory.get_schema_version()

    if current_version is None:
        raise SchemaVersionMismatch("uninitialized", EXPECTED_SCHEMA_VERSION)

    if current_version != EXPECTED_SCHEMA_VERSION:
        raise SchemaVersionMismatch(current_version, EXPECTED_SCHEMA_VERSION)

# On startup
try:
    validate_schema_version()
except SchemaVersionMismatch as e:
    print(f"ERROR: {e}")
    print("Run: alembic upgrade head")
    sys.exit(2)
```

**Risk Level**: **Low** (prevented by version checks)

---

### 9. Transaction Isolation

**Threat**: Concurrent operations cause race conditions or partial updates.

**Control**: **ACID Transactions with Rollback**

**Implementation**:
```python
@contextmanager
def get_db_session():
    """Context manager with automatic rollback"""
    session = SessionLocal()
    try:
        yield session
        session.commit()  # Commit on success
    except Exception:
        session.rollback()  # Rollback on any error
        raise
    finally:
        session.close()

# Usage
with get_db_session() as session:
    # Multi-step operation
    task.status = "completed"
    task.completed_at = datetime.utcnow()
    history = TaskHistory(task_id=task.id, ...)
    session.add(history)
    # All or nothing (atomic)
```

**Isolation Level**:
- SQLite default: SERIALIZABLE (strictest)
- No configuration needed (automatic)

**Risk Level**: **Low** (ACID guarantees from SQLite)

---

### 10. Dependency Security

**Threat**: Vulnerabilities in Python dependencies (SQLAlchemy, Alembic, etc.).

**Control**: **Dependency Pinning and Audits**

**Implementation**:

**requirements.txt** (pinned versions):
```
sqlalchemy==2.0.23
alembic==1.13.0
click==8.1.7
rich==13.7.0
```

**Security Audit**:
```bash
# Check for known vulnerabilities
pip-audit

# Update dependencies
pip install --upgrade sqlalchemy alembic

# Re-freeze
pip freeze > requirements.txt
```

**Automated Scanning** (optional):
- GitHub Dependabot alerts
- Snyk integration
- Weekly `pip-audit` runs

**Risk Level**: **Medium** (mitigated by pinning + regular audits)

---

## Security Best Practices

### Development

1. **Never commit database files to Git**
   ```gitignore
   # .gitignore
   project.db
   project.db-wal
   project.db-shm
   *.db
   ```

2. **Use environment-specific databases**
   ```bash
   # Development
   export PM_DB_PATH=./dev_project.db

   # Testing
   export PM_DB_PATH=:memory:
   ```

3. **Enable SQL logging only in development**
   ```python
   if os.getenv("ENV") == "development":
       config.echo = True
   ```

### Production

1. **Set restrictive file permissions**
   ```bash
   chmod 600 project.db
   chmod 600 project.db-wal
   chmod 600 project.db-shm
   ```

2. **Regular backups**
   ```bash
   # Daily backup
   cp project.db backups/project_$(date +%Y%m%d).db
   ```

3. **Integrity checks**
   ```bash
   # Weekly verification
   sqlite3 project.db "PRAGMA integrity_check;"
   ```

4. **Update dependencies**
   ```bash
   # Monthly security updates
   pip install --upgrade sqlalchemy alembic
   pip-audit
   ```

---

## Compliance and Audit

### Data Retention

**Task History**:
- All status changes recorded in `task_history` table
- Permanent audit trail (not deleted with tasks due to CASCADE)
- Timestamp precision: second-level UTC

**Access Logging**:
- Application logs all database operations (INFO level)
- Log format: `{timestamp} {operation} {entity_key} {user}`
- Logs stored: `~/.pm/logs/database.log`

**Data Export**:
- Tasks exportable to JSON/CSV (E05-F03)
- Audit trail exportable for compliance

### Privacy Considerations

**No Personal Data**:
- Database contains project metadata only
- No personally identifiable information (PII)
- No authentication credentials

**User Information**:
- `assigned_agent` field stores agent names (not PII)
- `task_history.agent` stores who made changes (not PII)
- No email addresses, phone numbers, or sensitive data

---

## Incident Response

### Database Corruption

**Detection**:
```bash
# Run integrity check
sqlite3 project.db "PRAGMA integrity_check;"
```

**Response**:
1. Stop application immediately
2. Restore from most recent backup
3. Verify backup integrity
4. Investigate cause (disk failure, power loss, etc.)

### Unauthorized Access

**Detection**:
- File permission changes
- Unexpected database modifications
- Unknown processes accessing database file

**Response**:
1. Revoke file access immediately
2. Review audit logs for unauthorized changes
3. Restore from backup if data compromised
4. Reset file permissions

### Data Loss

**Prevention**:
- Regular automated backups
- Transaction rollback on errors
- Schema migrations with downgrade support

**Recovery**:
1. Restore from most recent backup
2. Replay operations from audit logs (manual)
3. Verify data consistency

---

## Security Testing

### Test Cases

1. **SQL Injection Test**:
   ```python
   # Attempt SQL injection in task title
   malicious_title = "Test'; DROP TABLE tasks; --"
   task = create_task(title=malicious_title)
   # Verify: No SQL execution, title stored as literal string
   ```

2. **Path Traversal Test**:
   ```python
   # Attempt directory traversal in file path
   malicious_path = "../../../../../../etc/passwd"
   # Verify: ValidationError raised
   ```

3. **Permission Test**:
   ```python
   # Verify file permissions after creation
   db_path = Path("project.db")
   assert oct(db_path.stat().st_mode)[-3:] == "600"
   ```

4. **Foreign Key Test**:
   ```python
   # Attempt to create task with invalid feature_id
   task = Task(feature_id=9999, key="T-E04-F01-001", ...)
   # Verify: IntegrityError raised
   ```

5. **Transaction Rollback Test**:
   ```python
   # Create task, force error, verify rollback
   with get_db_session() as session:
       task = Task(...)
       session.add(task)
       raise Exception("Forced error")
   # Verify: Task not in database
   ```

---

## Security Checklist

Before deployment, verify:

- [ ] SQL injection protection: All queries use ORM
- [ ] File permissions: Database file is 600 (Unix)
- [ ] Input validation: All user input validated
- [ ] Path traversal: File paths validated
- [ ] Logging: No sensitive data logged
- [ ] Integrity checks: Enabled on startup
- [ ] Backups: Automated backup strategy in place
- [ ] Foreign keys: Enabled (verified on startup)
- [ ] Schema version: Validated on startup
- [ ] Dependencies: Pinned and audited
- [ ] .gitignore: Database files excluded
- [ ] Error handling: Graceful error messages
- [ ] Transaction rollback: Automatic on failures

---

## Summary

This security design provides:

1. **SQL Injection Prevention**: Parameterized queries via ORM (100% coverage)
2. **File Permission Controls**: 600 permissions on Unix systems
3. **Input Validation**: Multi-layer validation (application + database + types)
4. **Path Traversal Prevention**: File path validation and sandboxing
5. **Data Privacy**: No sensitive data logging
6. **Database Integrity**: Integrity checks + automated backups
7. **Foreign Key Enforcement**: Enabled at connection level
8. **Schema Version Validation**: Startup checks prevent version mismatches
9. **Transaction Isolation**: ACID guarantees with automatic rollback
10. **Dependency Security**: Pinned versions + regular audits

**Overall Security Posture**: **Strong** for a single-developer CLI tool

**Key Mitigations**:
- All high-risk threats mitigated (SQL injection, path traversal)
- Medium-risk threats have compensating controls (backups, audits)
- Low-risk threats accepted (single-user context, local-only access)

**Recommendations**:
1. Run `pip-audit` monthly
2. Enable daily automated backups
3. Review audit logs weekly
4. Test disaster recovery procedures quarterly

---

**Security Design Complete**: 2025-12-14
**Next Document**: 07-performance-design.md (coordinator creates performance specifications)