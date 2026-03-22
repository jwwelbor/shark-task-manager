# Migration Strategy: E21-F08 Polymorphic Data Model Unification

**Feature**: E21-F08
**Author**: Architect
**Date**: 2026-03-20
**Status**: Draft

---

## 1. Migration Overview

This migration consolidates 6 tables (5 per-entity document join tables + 1 task history table) into 2 polymorphic tables (`entity_documents`, `entity_history`).

**Schema Version**: 6 -> 7

**Migration Type**: Destructive (old tables dropped after data copy and verification)

**Per database-critical.md**: Developer MUST set `skip_migrations: false` in `.sharkconfig.json` before running the next shark command after this change is deployed, then set it back to `true`.

---

## 2. Migration Function

**Location**: `internal/db/migrate.go`

**Function name**: `migrateToPolymorphicTables(db *sql.DB) error`

**Called from**: `runMigrations()` in `internal/db/db.go`

### 2.1 Step-by-Step Algorithm

```
Step 1: Create entity_documents table (IF NOT EXISTS)
Step 2: Create entity_history table (IF NOT EXISTS)
Step 3: Create indexes (IF NOT EXISTS)
Step 4: Copy data from epic_documents -> entity_documents (entity_type='epic')
Step 5: Copy data from feature_documents -> entity_documents (entity_type='feature')
Step 6: Copy data from task_documents -> entity_documents (entity_type='task', preserve link_type)
Step 7: Copy data from bug_documents -> entity_documents (entity_type='bug')
Step 8: Copy data from change_card_documents -> entity_documents (entity_type='change')
Step 9: Copy data from task_history -> entity_history (entity_type='task')
Step 10: Verify row counts (old vs new)
Step 11: Drop old tables (only if verification passes)
Step 12: Drop old indexes
```

### 2.2 SQL Statements

#### Document Migration

```sql
-- Step 4: Epic documents
INSERT OR IGNORE INTO entity_documents (entity_type, entity_id, document_id, link_type, created_at)
SELECT 'epic', epic_id, document_id, 'general', created_at
FROM epic_documents;

-- Step 5: Feature documents
INSERT OR IGNORE INTO entity_documents (entity_type, entity_id, document_id, link_type, created_at)
SELECT 'feature', feature_id, document_id, 'general', created_at
FROM feature_documents;

-- Step 6: Task documents (preserves link_type)
INSERT OR IGNORE INTO entity_documents (entity_type, entity_id, document_id, link_type, created_at)
SELECT 'task', task_id, document_id, COALESCE(link_type, 'general'), created_at
FROM task_documents;

-- Step 7: Bug documents
INSERT OR IGNORE INTO entity_documents (entity_type, entity_id, document_id, link_type, created_at)
SELECT 'bug', bug_id, document_id, 'general', created_at
FROM bug_documents;

-- Step 8: Change card documents
INSERT OR IGNORE INTO entity_documents (entity_type, entity_id, document_id, link_type, created_at)
SELECT 'change', change_card_id, document_id, 'general', created_at
FROM change_card_documents;
```

#### History Migration

```sql
-- Step 9: Task history
INSERT INTO entity_history (entity_type, entity_id, from_status, to_status, changed_by, notes, forced, rejection_reason, changed_at)
SELECT 'task', task_id, old_status, new_status, agent, notes,
       COALESCE(forced, 0), rejection_reason, timestamp
FROM task_history;
```

**Notes on task_history migration**:
- `old_status` maps to `from_status`
- `new_status` maps to `to_status`
- `agent` maps to `changed_by`
- `timestamp` maps to `changed_at`
- `forced` may be NULL in existing data; `COALESCE(forced, 0)` normalizes to 0

#### Verification Queries

```sql
-- Step 10: Verify document migration
SELECT
    (SELECT COUNT(*) FROM epic_documents) AS epic_old,
    (SELECT COUNT(*) FROM entity_documents WHERE entity_type = 'epic') AS epic_new,
    (SELECT COUNT(*) FROM feature_documents) AS feature_old,
    (SELECT COUNT(*) FROM entity_documents WHERE entity_type = 'feature') AS feature_new,
    (SELECT COUNT(*) FROM task_documents) AS task_old,
    (SELECT COUNT(*) FROM entity_documents WHERE entity_type = 'task') AS task_new,
    (SELECT COUNT(*) FROM bug_documents) AS bug_old,
    (SELECT COUNT(*) FROM entity_documents WHERE entity_type = 'bug') AS bug_new,
    (SELECT COUNT(*) FROM change_card_documents) AS change_old,
    (SELECT COUNT(*) FROM entity_documents WHERE entity_type = 'change') AS change_new;

-- Verify history migration
SELECT
    (SELECT COUNT(*) FROM task_history) AS history_old,
    (SELECT COUNT(*) FROM entity_history WHERE entity_type = 'task') AS history_new;
```

#### Drop Old Tables (Only After Verification)

```sql
-- Step 11: Drop old tables
DROP TABLE IF EXISTS epic_documents;
DROP TABLE IF EXISTS feature_documents;
DROP TABLE IF EXISTS task_documents;
DROP TABLE IF EXISTS bug_documents;
DROP TABLE IF EXISTS change_card_documents;
DROP TABLE IF EXISTS task_history;
```

---

## 3. Go Implementation Pattern

```go
func migrateToPolymorphicTables(db *sql.DB) error {
    // Check if migration already completed (entity_documents exists AND old tables don't)
    if tableExists(db, "entity_documents") && !tableExists(db, "epic_documents") {
        return nil // Already migrated
    }

    // Step 1-3: Create new tables and indexes
    if err := createPolymorphicTables(db); err != nil {
        return fmt.Errorf("failed to create polymorphic tables: %w", err)
    }

    // Steps 4-9: Migrate data (only if old tables still exist)
    if tableExists(db, "epic_documents") {
        if err := migrateDocumentData(db); err != nil {
            return fmt.Errorf("failed to migrate document data: %w", err)
        }
    }

    if tableExists(db, "task_history") {
        if err := migrateHistoryData(db); err != nil {
            return fmt.Errorf("failed to migrate history data: %w", err)
        }
    }

    // Step 10: Verify row counts
    if err := verifyMigration(db); err != nil {
        return fmt.Errorf("migration verification failed (old tables retained): %w", err)
    }

    // Step 11-12: Drop old tables and indexes
    if err := dropOldTables(db); err != nil {
        return fmt.Errorf("failed to drop old tables: %w", err)
    }

    return nil
}

func verifyMigration(db *sql.DB) error {
    checks := []struct {
        oldTable   string
        entityType string
    }{
        {"epic_documents", "epic"},
        {"feature_documents", "feature"},
        {"task_documents", "task"},
        {"bug_documents", "bug"},
        {"change_card_documents", "change"},
    }

    for _, c := range checks {
        if !tableExists(db, c.oldTable) {
            continue // Old table already dropped (re-run safety)
        }
        var oldCount, newCount int
        db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", c.oldTable)).Scan(&oldCount)
        db.QueryRow("SELECT COUNT(*) FROM entity_documents WHERE entity_type = ?", c.entityType).Scan(&newCount)
        if newCount < oldCount {
            return fmt.Errorf("%s: expected >= %d rows in entity_documents, got %d", c.oldTable, oldCount, newCount)
        }
    }

    // Verify task_history
    if tableExists(db, "task_history") {
        var oldCount, newCount int
        db.QueryRow("SELECT COUNT(*) FROM task_history").Scan(&oldCount)
        db.QueryRow("SELECT COUNT(*) FROM entity_history WHERE entity_type = 'task'").Scan(&newCount)
        if newCount < oldCount {
            return fmt.Errorf("task_history: expected >= %d rows in entity_history, got %d", oldCount, newCount)
        }
    }

    return nil
}
```

---

## 4. Idempotency

The migration is fully idempotent:

| Scenario | Behavior |
|----------|----------|
| First run (old tables exist, new don't) | Creates new tables, copies data, verifies, drops old |
| Second run (new tables exist, old don't) | Early return -- `tableExists` check |
| Partial run (new tables exist, old still exist) | Copies data again (INSERT OR IGNORE prevents duplicates), verifies, drops old |
| New database (no old tables, no new tables) | Creates new tables (empty), skips copy (old tables don't exist), skips drop |

The `INSERT OR IGNORE` on document migration handles re-runs safely due to the UNIQUE constraint on `(entity_type, entity_id, document_id)`.

The `INSERT INTO` on history migration does not use `INSERT OR IGNORE` because history has no unique constraint (multiple transitions are valid). To handle re-runs, the history migration checks if the old table exists before copying.

---

## 5. Rollback Strategy

**Before migration**: The developer should back up the database:
```bash
cp shark-tasks.db shark-tasks.db.backup-pre-f08
```

**If verification fails**: The migration function returns an error without dropping old tables. Both old and new tables coexist. The developer can:
1. Investigate the verification failure
2. Fix data issues
3. Re-run the migration

**If issues discovered after migration**: Restore from backup:
```bash
cp shark-tasks.db.backup-pre-f08 shark-tasks.db
```

**Note**: Once old tables are dropped and the application runs successfully, rollback requires a code revert (to restore old table references) plus a database restore.

---

## 6. Schema DDL Changes in `db.go`

The core schema in `internal/db/db.go` needs the following updates:

1. **Add `entity_documents` DDL** alongside (not replacing) the old table DDL, since the migration handles the transition
2. **Add `entity_history` DDL**
3. **Bump `CurrentSchemaVersion`** from 6 to 7
4. **Remove old table DDL** from the schema initialization (for new databases, only create the new tables)

For new databases (no existing data), the old per-entity tables are never created. Only `entity_documents` and `entity_history` are created in the schema.

For existing databases, the migration function handles the transition.

---

## 7. Impact on `skip_migrations` Flag

Per `database-critical.md`:

> This change adds a migration. Set `skip_migrations: false` in `.sharkconfig.json` before running the next shark command, then set it back to `true`.

The migration:
1. Checks `CurrentSchemaVersion` (6 vs 7)
2. If behind, runs `migrateToPolymorphicTables`
3. Updates stored version to 7

If `skip_migrations: true` and version is already 7, the migration is skipped. If version is 6, the migration runs regardless of the flag (version check triggers it).

---

## 8. Timeline and Ordering

```
1. Merge schema changes (new table DDL + migration function)
2. Developer sets skip_migrations: false
3. Developer runs any shark command (triggers migration)
4. Migration creates new tables, copies data, verifies, drops old tables
5. Developer sets skip_migrations: true
6. Merge repository changes (EntityDocumentRepository, EntityHistoryRepository)
7. Merge service changes (simplified EntityDocumentService, history recording)
8. Merge CLI changes (--bug/--change flags, entity history command)
```

Steps 6-8 can be merged together or separately. The critical ordering is that step 1 (schema) must be deployed and migrated before steps 6-8.

---

*Last Updated*: 2026-03-20
