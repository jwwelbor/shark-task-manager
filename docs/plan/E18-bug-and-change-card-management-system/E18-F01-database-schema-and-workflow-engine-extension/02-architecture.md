# E18-F01 Technical Architecture: Database Schema and Workflow Engine Extension

**Feature**: E18-F01
**Date**: 2026-03-03
**Status**: Draft
**Complexity**: Standard (S-M)

---

## 1. Architecture Overview

F01 is a **pure infrastructure feature** -- it creates the data layer and workflow definitions that all downstream E18 features (F02-F07) depend on. There are no new services, repositories, models, or CLI commands in F01. The scope is:

1. Two new database tables (`bugs`, `change_cards`)
2. Workflow engine extension (two new levels: `bug`, `change`)
3. `entity_notes` table migration (CHECK constraint removal)
4. Workflow profile updates (basic and advanced profiles)

All changes are additive. No existing behavior is modified.

### Component Diagram

```
+----------------------------+     +------------------------------+
| internal/db/db.go          |     | internal/config/             |
|                            |     |   workflow_multilevel.go      |
| - bugs table (CREATE)      |     |   - MultiLevelWorkflow.Bug   |
| - change_cards table       |     |   - MultiLevelWorkflow.Change|
| - entity_notes migration   |     |   workflow_default.go        |
| - indexes                  |     |   - DefaultBugWorkflow()     |
+----------------------------+     |   - DefaultChangeCardWorkflow|
                                   +------------------------------+
                                              |
                                   +------------------------------+
                                   | internal/workflow/levels.go   |
                                   |   - LevelBug = "bug"         |
                                   |   - LevelChange = "change"   |
                                   +------------------------------+
                                              |
+----------------------------+     +------------------------------+
| internal/init/             |     | .sharkconfig.json            |
|   profiles/basic.json      |     |   - bug_workflow section     |
|   profiles/advanced.json   |     |   - change_workflow section  |
+----------------------------+     +------------------------------+
```

### Data Flow

```
InitDB() call
  |
  v
createSchema()        -- bugs table, change_cards table (CREATE TABLE IF NOT EXISTS)
  |
  v
runMigrations()       -- entity_notes CHECK constraint removal (idempotent)
  |                   -- bugs/change_cards indexes (CREATE INDEX IF NOT EXISTS)
  v
Database ready with new tables + migrated entity_notes
```

```
GetWorkflowForLevel("bug") call
  |
  v
MultiLevelWorkflow.GetWorkflowForLevel()
  |
  +-- m.Bug != nil? --> return m.Bug (from .sharkconfig.json)
  +-- m.Bug == nil? --> return DefaultBugWorkflow() (hardcoded default)
```

---

## 2. Key Architecture Decisions

### ADR-1: Standalone Tables (Not Task Subtypes)

**Decision**: Create separate `bugs` and `change_cards` tables rather than adding a `type` column to the existing `tasks` table.

**Rationale**:
- Bugs and change-cards have fundamentally different schemas (severity on bugs, not on change-cards; different default statuses)
- Independent workflows with separate status flows
- Aligns with industry norms (Jira, Linear, Bugzilla all use distinct entity types)
- Avoids polluting the tasks table with nullable columns
- Enables independent indexes optimized for each entity type

**Consequences**:
- Positive: Clean separation of concerns; each entity type evolves independently
- Negative: More tables to manage; cross-entity queries (e.g., "all work items") require UNION

### ADR-2: Ideas Table Pattern Reuse

**Decision**: Model `bugs` and `change_cards` tables after the existing `ideas` table pattern (standalone entity with auto-increment key, optional linking columns, timestamps).

**Rationale**:
- The `ideas` table (`internal/db/db.go`, lines 337-370) demonstrates a proven pattern for standalone entities
- Auto-increment primary key with formatted TEXT key (B001, C001) follows the idea's key pattern
- Optional linking via `linked_entity_type` + `linked_entity_key` matches the idea's `converted_to_type`/`converted_to_key` pattern
- Reduces implementation risk through pattern reuse

**Consequences**:
- Positive: ~60% code reuse from established patterns
- Negative: None identified

### ADR-3: Remove entity_notes CHECK Constraint Entirely

**Decision**: Remove the `CHECK (entity_type IN ('epic', 'feature', 'task'))` constraint from `entity_notes` entirely rather than updating it to include `'bug'` and `'change'`.

**Rationale**:
- SQLite does not support ALTER TABLE for CHECK constraints; migration requires table recreation regardless
- Removing the constraint eliminates future migration needs when adding entity types
- Application-layer validation in the service layer (`NoteService`) is the primary validation point
- The CHECK constraint was defense-in-depth, not the primary enforcement mechanism
- Per Tech Feasibility Review, Recommendation 3

**Consequences**:
- Positive: Zero future database migrations needed for new entity types; simpler migration logic
- Negative: Slightly weaker database-level validation (mitigated by service-layer validation)

### ADR-4: Workflow Level Naming Convention

**Decision**: Bug workflow level is `"bug"`, change-card workflow level is `"change"` (not `"change_card"`).

**Rationale**:
- Matches the key prefix pattern: B for bug, C for change
- Keeps level names short and consistent with existing levels: `"epic"`, `"feature"`, `"task"`
- The `ForLevel()` API uses short strings; `"change"` is clearer than `"change_card"` in this context

**Consequences**:
- Positive: Consistent naming; shorter config keys
- Negative: None

### ADR-5: Default Status Names

**Decision**: Bug default status is `reported` (not `open` or `new`). Change-card default status is `proposed` (not `draft` or `submitted`).

**Rationale**:
- `reported` clearly indicates a bug has been observed and reported, distinct from an in-progress investigation
- `proposed` clearly indicates a change-card is a proposal awaiting approval, distinct from already-approved work
- These names match the workflows defined in the E18 epic PRD
- Distinct from existing entity status names (`todo`, `draft`) to avoid confusion

**Consequences**:
- Positive: Clear, unambiguous status names; no collision with existing entity statuses
- Negative: None

### ADR-6: Severity as Dedicated Column (Not context_data)

**Decision**: The `severity` field is a dedicated database column on the `bugs` table with a CHECK constraint, not a JSON field in `context_data`.

**Rationale**:
- Severity is used for filtering in list queries (`shark bug list --severity=critical`)
- A dedicated column enables indexed queries for fast filtering
- CHECK constraint (`severity IN ('critical', 'high', 'medium', 'low')`) enforces data integrity at the database level
- Other optional metadata (environment, component, reproduction steps) goes in `context_data` JSON

**Consequences**:
- Positive: Fast indexed queries; database-level validation
- Negative: Schema change required if severity levels change (unlikely; industry-standard set)

### ADR-7: Cascade Delete Triggers Deferred to F02/F03

**Decision**: Cascade delete triggers for bugs and change-cards (to clean up entity_notes when a bug/change-card is deleted) are NOT included in F01. They will be added in F02/F03 when the delete operations are implemented.

**Rationale**:
- F01 creates the tables but no repository layer or delete operations exist yet
- Triggers referencing non-existent delete operations would be premature
- The existing pattern (cascade triggers created alongside the table) applies when the delete path exists
- F02/F03 will add: `entity_notes_cascade_delete_bug` and `entity_notes_cascade_delete_change_card` triggers

**Consequences**:
- Positive: Clean separation; triggers tested alongside delete operations
- Negative: Brief window where orphaned notes could exist (until F02/F03 adds triggers; no user-facing delete exists in this window)

---

## 3. Integration Points

### 3.1 `internal/db/db.go` -- Database Schema and Migrations

**Changes**:
1. Add `bugs` table creation SQL to `createSchema()` (using `CREATE TABLE IF NOT EXISTS`)
2. Add `change_cards` table creation SQL to `createSchema()`
3. Add index creation for both tables to `createSchema()` (using `CREATE INDEX IF NOT EXISTS`)
4. Add `migrateEntityNotesRemoveCheckConstraint()` function to `runMigrations()`
5. Add updated_at triggers for both new tables

**Pattern**: Follow the existing `ideas` table creation pattern in `createSchema()` (lines 337-370 of current db.go).

**Migration Strategy for entity_notes**:
```
1. Check if CHECK constraint still exists (query sqlite_master for CREATE TABLE statement)
2. If constraint exists:
   a. BEGIN TRANSACTION
   b. CREATE TABLE entity_notes_new (same schema, no CHECK on entity_type)
   c. INSERT INTO entity_notes_new SELECT * FROM entity_notes
   d. DROP TABLE entity_notes
   e. ALTER TABLE entity_notes_new RENAME TO entity_notes
   f. Recreate all indexes on entity_notes
   g. Recreate all cascade delete triggers on entity_notes
   h. COMMIT
3. If constraint does not exist: no-op (idempotent)
```

### 3.2 `internal/config/workflow_multilevel.go` -- Struct Extension

**Changes**:
1. Add `Bug *WorkflowConfig` field to `MultiLevelWorkflow` struct
2. Add `Change *WorkflowConfig` field to `MultiLevelWorkflow` struct
3. Add `case "bug":` to `GetWorkflowForLevel()` switch
4. Add `case "change":` to `GetWorkflowForLevel()` switch

**Pattern**: Exact replication of existing `case "epic":` / `case "feature":` / `case "task":` branches.

### 3.3 `internal/config/workflow_default.go` -- Default Workflows

**Changes**:
1. Add `DefaultBugWorkflow()` function
2. Add `DefaultChangeCardWorkflow()` function

**Pattern**: Follow `DefaultEpicWorkflow()` structure (lines 84-105 of current workflow_default.go). Each function returns a `*WorkflowConfig` with:
- `StatusFlow` map defining valid transitions
- `StatusMetadata` map with color, description, phase, progress weight, responsibility
- `SpecialStatuses` map with `_start_` and `_complete_` entries

### 3.4 `internal/workflow/levels.go` -- Level Constants

**Changes**:
1. Add `LevelBug = "bug"` constant
2. Add `LevelChange = "change"` constant

**Pattern**: Same file, same const block, two new lines.

### 3.5 `internal/init/profiles/basic.json` -- Basic Profile

**Changes**: Add `bug_workflow` and `change_workflow` sections with simplified status flows.

### 3.6 `internal/init/profiles/advanced.json` -- Advanced Profile

**Changes**: Add `bug_workflow` and `change_workflow` sections with full status flows, agent assignments, and metadata.

### 3.7 Config Deserialization -- Forward Compatibility

**No code changes required**. The config system uses `map[string]interface{}` for profile data (`GetProfileMap()`), which naturally handles new JSON keys. The `MultiLevelWorkflow` struct uses pointer fields (`*WorkflowConfig`), so missing sections in old configs result in `nil` fields that fall back to defaults via `GetWorkflowForLevel()`. This is the existing forward-compatibility mechanism.

---

## 4. Workflow Definitions

### 4.1 Bug Workflow

```
Status Flow:
  reported --> triaged --> in_fix --> in_verification --> resolved
                                                     --> wont_fix (from reported, triaged, in_fix, in_verification)
              duplicate (from reported, triaged)

Entry status: reported
Terminal statuses: resolved, wont_fix, duplicate
```

| Status | Color | Phase | Progress Weight | Responsibility |
|--------|-------|-------|-----------------|----------------|
| reported | red | planning | 0 | agent |
| triaged | orange | planning | 20 | agent |
| in_fix | blue | development | 50 | developer |
| in_verification | yellow | review | 80 | qa_team |
| resolved | green | done | 100 | none |
| wont_fix | gray | done | 100 | none |
| duplicate | gray | done | 100 | none |

### 4.2 Change-Card Workflow

```
Status Flow:
  proposed --> approved --> in_progress --> completed
          --> declined (from proposed only)

Entry status: proposed
Terminal statuses: completed, declined
```

| Status | Color | Phase | Progress Weight | Responsibility |
|--------|-------|-------|-----------------|----------------|
| proposed | gray | planning | 0 | human |
| approved | blue | planning | 25 | agent |
| in_progress | yellow | development | 50 | developer |
| completed | green | done | 100 | none |
| declined | red | done | 100 | none |

---

## 5. Testing Strategy

### 5.1 Database Tests (Repository Layer -- Real DB)

- **Fresh database initialization**: Verify `bugs` and `change_cards` tables are created with correct schemas
- **Existing database upgrade**: Verify migration adds tables without affecting existing data
- **entity_notes migration**: Verify CHECK constraint removal, data preservation, index recreation, trigger recreation
- **entity_notes idempotency**: Verify running migration twice does not fail or duplicate data
- **Constraint enforcement**: Verify CHECK constraint on `bugs.severity` rejects invalid values
- **Default values**: Verify default status and timestamp values are applied

### 5.2 Workflow Engine Tests (Unit Tests -- No DB)

- **GetWorkflowForLevel("bug")**: Returns bug workflow (custom or default)
- **GetWorkflowForLevel("change")**: Returns change-card workflow (custom or default)
- **Existing levels unchanged**: Epic, feature, task workflows unaffected
- **DefaultBugWorkflow()**: Correct status flow, metadata, terminal statuses
- **DefaultChangeCardWorkflow()**: Correct status flow, metadata, terminal statuses
- **Forward compatibility**: Old config (without bug/change sections) falls back to defaults

### 5.3 Profile Tests (Integration Tests)

- **Advanced profile**: Contains `bug_workflow` and `change_workflow` sections
- **Basic profile**: Contains simplified `bug_workflow` and `change_workflow` sections
- **Profile preservation**: Existing epic/feature/task sections unchanged after update

---

## 6. Risk Mitigations

| Risk | Mitigation |
|------|------------|
| entity_notes migration data loss | Transaction-protected; rollback on failure; idempotent check prevents re-execution |
| Old config files break | Forward compatibility via nil pointer + default fallback; no parsing changes needed |
| Schema version conflicts | New tables use `CREATE TABLE IF NOT EXISTS`; indexes use `CREATE INDEX IF NOT EXISTS` |
| Workflow engine regression | Existing level tests remain; new tests added for bug/change levels only |

---

## 7. Files Modified

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/db/db.go` | Modified | Add bugs/change_cards tables, indexes, updated_at triggers; add entity_notes migration |
| `internal/config/workflow_multilevel.go` | Modified | Add Bug/Change fields to struct; extend switch |
| `internal/config/workflow_default.go` | Modified | Add DefaultBugWorkflow(), DefaultChangeCardWorkflow() |
| `internal/workflow/levels.go` | Modified | Add LevelBug, LevelChange constants |
| `internal/init/profiles/basic.json` | Modified | Add bug_workflow, change_workflow sections |
| `internal/init/profiles/advanced.json` | Modified | Add bug_workflow, change_workflow sections |
| `internal/config/workflow_multilevel_test.go` | Modified | Add tests for bug/change level dispatch |
| `internal/db/db_test.go` (or new migration test) | Modified | Add migration and schema tests |

**No new files** -- all changes are extensions to existing files.

---

*Last Updated*: 2026-03-03
