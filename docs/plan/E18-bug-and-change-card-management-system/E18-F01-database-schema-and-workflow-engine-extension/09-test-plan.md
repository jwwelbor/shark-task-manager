# E18-F01 Test Plan: Database Schema and Workflow Engine Extension

**Feature**: E18-F01
**Date**: 2026-03-03
**Author**: QA Agent
**Tier**: STANDARD
**Status**: Complete

---

## Table of Contents

1. [Acceptance Criteria Test Matrix](#1-acceptance-criteria-test-matrix)
2. [Component Test Strategy](#2-component-test-strategy)
3. [Integration Scenarios](#3-integration-scenarios)
4. [Epic UAT Traceability](#4-epic-uat-traceability)

---

## 1. Acceptance Criteria Test Matrix

Each acceptance criterion from every story in the feature PRD is mapped to concrete test cases with inputs, expected outputs, and edge cases.

---

### Story 1: bugs Table Schema

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| bugs table exists with correct columns | TC-S1-01: Verify bugs table creation on fresh DB | `InitDB()` on empty database file | Table `bugs` appears in `sqlite_master` with all 12 columns: id, key, title, status, severity, slug, linked_entity_type, linked_entity_key, context_data, file_path, created_at, updated_at | N/A (schema creation) |
| Indexes exist on key, status, severity, linked_entity_key, slug | TC-S1-02: Verify all 5 bugs indexes | `InitDB()` then query `sqlite_master` for indexes | `idx_bugs_key`, `idx_bugs_status`, `idx_bugs_severity`, `idx_bugs_linked_entity_key`, `idx_bugs_slug` all exist | Running `InitDB()` twice does not create duplicate indexes |
| Table created via auto-migration system | TC-S1-03: Idempotent table creation | Call `InitDB()` twice on same database | No error on second call; table still has correct schema | Database with pre-existing bugs table (from prior `InitDB()`) |
| Severity CHECK constraint rejects invalid values | TC-S1-04: Invalid severity rejected | `INSERT INTO bugs (key, title, severity) VALUES ('B001', 'test', 'invalid')` | CHECK constraint violation error | Values: empty string, NULL (rejected by NOT NULL), 'CRITICAL' (case sensitivity), 'moderate' |
| Severity CHECK constraint accepts valid values | TC-S1-05: Valid severities accepted | Insert rows with severity = 'critical', 'high', 'medium', 'low' | All 4 inserts succeed | N/A |
| Default status is 'reported' | TC-S1-06: Default status applied | `INSERT INTO bugs (key, title) VALUES ('B002', 'test')` | Queried row has status = 'reported' | N/A |
| Default severity is 'medium' | TC-S1-07: Default severity applied | `INSERT INTO bugs (key, title) VALUES ('B003', 'test')` | Queried row has severity = 'medium' | N/A |
| Default timestamps applied | TC-S1-08: Timestamp defaults | Insert row without specifying created_at or updated_at | Both columns are non-null and contain valid timestamps | N/A |
| UNIQUE constraint on key | TC-S1-09: Duplicate key rejected | Insert two bugs with key = 'B001' | Second insert fails with UNIQUE constraint error | N/A |
| updated_at trigger fires | TC-S1-10: Trigger updates timestamp | Insert bug, wait 1s, UPDATE title | updated_at changes after UPDATE | Verify trigger name `bugs_updated_at` exists in sqlite_master |

**Test Type**: Database integration tests (real DB, following `internal/db/ideas_migration_test.go` pattern)

**Test File**: `internal/db/bugs_table_test.go`

---

### Story 2: change_cards Table Schema

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| change_cards table exists with correct columns | TC-S2-01: Verify change_cards table creation | `InitDB()` on empty database | Table `change_cards` in `sqlite_master` with 11 columns (no severity) | N/A |
| Indexes exist on key, status, linked_entity_key, slug | TC-S2-02: Verify all 4 change_cards indexes | Query `sqlite_master` | `idx_change_cards_key`, `idx_change_cards_status`, `idx_change_cards_linked_entity_key`, `idx_change_cards_slug` | Idempotent creation |
| Table created via auto-migration | TC-S2-03: Idempotent creation | `InitDB()` twice | No error, schema intact | N/A |
| Default status is 'proposed' | TC-S2-04: Default status applied | `INSERT INTO change_cards (key, title) VALUES ('C001', 'test')` | status = 'proposed' | N/A |
| UNIQUE constraint on key | TC-S2-05: Duplicate key rejected | Two inserts with key = 'C001' | Second insert fails | N/A |
| updated_at trigger fires | TC-S2-06: Trigger updates timestamp | Insert, then UPDATE | updated_at changes | Verify trigger `change_cards_updated_at` exists |

**Test Type**: Database integration tests

**Test File**: `internal/db/change_cards_table_test.go`

---

### Story 3: Bug Workflow Level

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| MultiLevelWorkflow has Bug field | TC-S3-01: Struct field exists | Create `MultiLevelWorkflow{}`, set `Bug` field | Compiles and field is accessible | Nil Bug field (must not panic) |
| GetWorkflowForLevel("bug") returns config | TC-S3-02: Bug level dispatch with nil | `MultiLevelWorkflow{}.GetWorkflowForLevel("bug")` | Returns non-nil `*WorkflowConfig` (default) | N/A |
| GetWorkflowForLevel("bug") falls back to default | TC-S3-03: Fallback to DefaultBugWorkflow | `MultiLevelWorkflow{Bug: nil}.GetWorkflowForLevel("bug")` | Returns workflow with 7 statuses matching DefaultBugWorkflow() | N/A |
| GetWorkflowForLevel("bug") returns custom when set | TC-S3-04: Custom bug workflow | `MultiLevelWorkflow{Bug: customConfig}.GetWorkflowForLevel("bug")` | Returns customConfig (pointer equality) | N/A |
| DefaultBugWorkflow status flow | TC-S3-05: Bug status transitions | Call `DefaultBugWorkflow()` | `reported` -> `{triaged, wont_fix, duplicate}`, `triaged` -> `{in_fix, wont_fix, duplicate}`, `in_fix` -> `{in_verification, wont_fix}`, `in_verification` -> `{resolved, in_fix, wont_fix}`, terminal statuses have empty arrays | Missing transitions, extra transitions |
| Terminal statuses defined | TC-S3-06: Bug terminal statuses | `DefaultBugWorkflow().SpecialStatuses` | `_complete_` contains: resolved, wont_fix, duplicate | N/A |
| Status metadata defined | TC-S3-07: Bug status metadata | `DefaultBugWorkflow().StatusMetadata` | All 7 statuses have Color, Phase, ProgressWeight, Responsibility | Missing metadata for any status |
| Progress weights match spec | TC-S3-08: Bug progress weights | `DefaultBugWorkflow().StatusMetadata` | reported=0, triaged=0.2, in_fix=0.5, in_verification=0.8, resolved=1.0, wont_fix=1.0, duplicate=1.0 | N/A |
| Start status is reported | TC-S3-09: Bug start status | `DefaultBugWorkflow().SpecialStatuses[StartStatusKey]` | Contains "reported" | N/A |

**Test Type**: Unit tests (no DB, no mocks needed -- pure function tests)

**Test File**: `internal/config/workflow_multilevel_test.go` (extend existing), `internal/config/workflow_default_test.go` (new or extend)

---

### Story 4: Change-Card Workflow Level

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| MultiLevelWorkflow has Change field | TC-S4-01: Struct field exists | Create `MultiLevelWorkflow{}`, set `Change` field | Compiles and accessible | Nil Change field |
| GetWorkflowForLevel("change") returns config | TC-S4-02: Change level dispatch with nil | `MultiLevelWorkflow{}.GetWorkflowForLevel("change")` | Returns non-nil `*WorkflowConfig` (default) | N/A |
| GetWorkflowForLevel("change") returns custom when set | TC-S4-03: Custom change workflow | `MultiLevelWorkflow{Change: customConfig}.GetWorkflowForLevel("change")` | Returns customConfig | N/A |
| DefaultChangeCardWorkflow status flow | TC-S4-04: Change-card transitions | Call `DefaultChangeCardWorkflow()` | `proposed` -> `{approved, declined}`, `approved` -> `{in_progress, proposed}`, `in_progress` -> `{completed, approved}`, terminal: empty | Extra/missing transitions |
| Terminal statuses defined | TC-S4-05: Change-card terminal statuses | `DefaultChangeCardWorkflow().SpecialStatuses` | `_complete_` contains: completed, declined | N/A |
| Status metadata defined | TC-S4-06: Change-card status metadata | `DefaultChangeCardWorkflow().StatusMetadata` | All 5 statuses have Color, Phase, ProgressWeight, Responsibility | N/A |
| Progress weights match spec | TC-S4-07: Change-card progress weights | `DefaultChangeCardWorkflow().StatusMetadata` | proposed=0, approved=0.25, in_progress=0.5, completed=1.0, declined=1.0 | N/A |
| Start status is proposed | TC-S4-08: Change-card start status | `DefaultChangeCardWorkflow().SpecialStatuses[StartStatusKey]` | Contains "proposed" | N/A |
| declined reachable only from proposed | TC-S4-09: Declined transition constraint | Inspect StatusFlow | Only `proposed` lists `declined` as a valid transition | N/A |

**Test Type**: Unit tests

**Test File**: Same as Story 3 test files

---

### Story 5: entity_notes Migration

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| CHECK constraint removed | TC-S5-01: Bug entity_type accepted | After migration: `INSERT INTO entity_notes (entity_type, entity_id, note_type, content) VALUES ('bug', 1, 'comment', 'test')` | Insert succeeds | N/A |
| Change entity_type accepted | TC-S5-02: Change entity_type accepted | Same insert with entity_type = 'change' | Insert succeeds | N/A |
| Existing notes preserved | TC-S5-03: Data preservation | Create DB, insert notes with entity_type epic/feature/task, run migration | All pre-existing notes exist with unchanged content, entity_type, entity_id | Large number of notes (100+) |
| Migration is idempotent | TC-S5-04: Double migration | Run `InitDB()` twice on same database | No error on second run; no duplicate data | N/A |
| Indexes recreated after migration | TC-S5-05: entity_notes indexes exist | After migration, query `sqlite_master` | `idx_entity_notes_type`, `idx_entity_notes_created_at`, `idx_entity_notes_entity_type`, `idx_entity_notes_type_entity` all present | N/A |
| Cascade delete triggers preserved | TC-S5-06: Cascade delete still works | After migration: insert task, insert note for task, delete task | Note is also deleted | Test for all 3 trigger types: task, feature, epic |
| Transaction-protected migration | TC-S5-07: Rollback on failure | Simulate failure mid-migration (difficult to test directly; verify by checking idempotency behavior) | Original table intact if migration cannot complete | N/A |

**Test Type**: Database integration tests

**Test File**: `internal/db/entity_notes_migration_test.go`

---

### Story 6: Workflow Profile Integration

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| Advanced profile includes bug workflow | TC-S6-01: Parse advanced.json | Load `internal/init/profiles/advanced.json` | JSON contains `bug_workflow` key with `status_flow`, `status_metadata`, `special_statuses` | N/A |
| Advanced profile includes change workflow | TC-S6-02: Parse advanced.json | Load `internal/init/profiles/advanced.json` | JSON contains `change_workflow` key | N/A |
| Basic profile includes bug workflow | TC-S6-03: Parse basic.json | Load `internal/init/profiles/basic.json` | JSON contains `bug_workflow` key | N/A |
| Basic profile includes change workflow | TC-S6-04: Parse basic.json | Load `internal/init/profiles/basic.json` | JSON contains `change_workflow` key | N/A |
| Existing workflow sections preserved | TC-S6-05: Non-destructive update | Load profile, verify epic/feature/task sections unchanged | All existing keys still present with same values | N/A |
| Config deserialization round-trip | TC-S6-06: Write then read | Apply advanced profile to `.sharkconfig.json`, reload | `GetWorkflowForLevel("bug")` returns workflow from config, not default | N/A |

**Test Type**: Unit tests (JSON parsing) + Integration tests (profile application)

**Test File**: `internal/config/workflow_profiles_test.go` or `internal/init/profiles_test.go`

---

### Error Story 1: Migration Safety

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| Migration with existing notes | TC-E1-01: Large dataset migration | Insert 100 notes, run migration | All 100 notes preserved with correct data | N/A |
| Migration idempotent | TC-E1-02: Double migration no duplicates | Insert 10 notes, run InitDB() twice | Exactly 10 notes exist (not 20) | N/A |

**Test Type**: Database integration tests (covered by TC-S5-03 and TC-S5-04)

---

### Error Story 2: Severity Constraint

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| Invalid severity rejected | TC-E2-01: CHECK enforcement | Insert with severity = 'invalid', '', 'MEDIUM' (uppercase) | All rejected by CHECK | Case sensitivity verification |
| Valid severities accepted | TC-E2-02: All valid values | Insert with each of: critical, high, medium, low | All succeed | N/A |

**Test Type**: Database integration tests (covered by TC-S1-04 and TC-S1-05)

---

### Non-Functional: Forward Compatibility (F01-REQ-NF-003)

| AC | Test Case | Input | Expected Output | Edge Cases |
|----|-----------|-------|-----------------|------------|
| Old config loads without error | TC-NF-01: Pre-E18 config | Load `.sharkconfig.json` without bug_workflow or change_workflow keys | No parse error | N/A |
| Bug workflow falls back to default | TC-NF-02: Missing bug_workflow | `GetWorkflowForLevel("bug")` with old config | Returns `DefaultBugWorkflow()` result | N/A |
| Change workflow falls back to default | TC-NF-03: Missing change_workflow | `GetWorkflowForLevel("change")` with old config | Returns `DefaultChangeCardWorkflow()` result | N/A |

**Test Type**: Unit tests

---

## 2. Component Test Strategy

F01 has four distinct components. Each component has a clear test strategy aligned with the codebase's existing test patterns.

---

### Component A: Database Schema (`internal/db/db.go`)

**What**: New `bugs` and `change_cards` tables, indexes, triggers, entity_notes migration.

**Test Approach**: Real database integration tests. Follow the pattern established in `internal/db/ideas_migration_test.go`:
- Create temporary database file per test
- `defer os.Remove(tmpFile)` for cleanup
- Call `InitDB()` to create schema
- Verify via `sqlite_master` queries and direct SQL operations

**Key Tests**:
1. **Table existence**: Query `sqlite_master` for table names (pattern: `TestIdeasTableCreation`)
2. **Column verification**: Query `pragma_table_info()` for all expected columns (pattern: `TestIdeasTableSchema`)
3. **Index verification**: Query `sqlite_master` for index names (pattern: `TestIdeasTableIndexes`)
4. **Constraint enforcement**: Insert invalid data, verify CHECK violation (pattern: `TestIdeasTableStatusConstraint`)
5. **Default values**: Insert minimal data, verify defaults applied (pattern: `TestIdeasTableInsertAndQuery`)
6. **entity_notes migration**: Pre-populate notes, run migration, verify data preserved and new entity_types accepted
7. **Idempotency**: Call `InitDB()` twice, verify no errors

**Test File Locations**:
- `internal/db/bugs_table_test.go` (new)
- `internal/db/change_cards_table_test.go` (new)
- `internal/db/entity_notes_migration_test.go` (new)

**Coverage Target**: All table creation, index creation, trigger creation, constraint enforcement, migration logic, and idempotency paths.

---

### Component B: Workflow Engine Extension (`internal/config/workflow_multilevel.go`)

**What**: Two new fields on `MultiLevelWorkflow`, two new switch cases in `GetWorkflowForLevel()`.

**Test Approach**: Unit tests with no database. Follow pattern in `internal/config/workflow_multilevel_test.go`:
- Test each new level with nil field (default fallback)
- Test each new level with custom config (returns custom)
- Regression: existing epic/feature/task behavior unchanged

**Key Tests**:
1. `TestGetWorkflowForLevel_BugWithNil` -- returns DefaultBugWorkflow
2. `TestGetWorkflowForLevel_BugWithCustom` -- returns custom config
3. `TestGetWorkflowForLevel_ChangeWithNil` -- returns DefaultChangeCardWorkflow
4. `TestGetWorkflowForLevel_ChangeWithCustom` -- returns custom config
5. Regression: re-run existing tests for epic/feature/task levels

**Test File Location**: `internal/config/workflow_multilevel_test.go` (extend existing)

**Coverage Target**: All branches in `GetWorkflowForLevel()` switch statement.

---

### Component C: Default Workflow Definitions (`internal/config/workflow_default.go`)

**What**: `DefaultBugWorkflow()` and `DefaultChangeCardWorkflow()` functions.

**Test Approach**: Unit tests. Each function returns a `*WorkflowConfig` struct that can be inspected directly.

**Key Tests**:
1. **Status count**: Bug has 7 statuses, change-card has 5
2. **Status flow correctness**: Each status maps to the correct list of valid transitions (table-driven test)
3. **Terminal statuses**: Statuses with empty transition arrays match SpecialStatuses `_complete_`
4. **Start status**: SpecialStatuses `_start_` matches expected entry status
5. **Progress weights**: Each status metadata has the correct weight (table-driven test)
6. **Metadata completeness**: Every status in StatusFlow has a corresponding StatusMetadata entry
7. **Backward transitions**: Verify `in_verification -> in_fix` (bug) and `approved -> proposed` (change-card) are present

**Test File Location**: `internal/config/workflow_default_test.go` (new or extend existing)

**Coverage Target**: 100% of both functions. Every status and every transition verified.

---

### Component D: Workflow Profile Integration (`internal/init/profiles/`)

**What**: Updated `basic.json` and `advanced.json` with `bug_workflow` and `change_workflow` sections.

**Test Approach**: Unit tests for JSON parsing validity. Integration tests for profile application round-trip.

**Key Tests**:
1. **JSON validity**: Parse both profile files, verify no JSON errors
2. **Key presence**: `bug_workflow` and `change_workflow` keys exist in both profiles
3. **Structure validation**: New sections have `status_flow`, `status_metadata`, `special_statuses` sub-keys
4. **Preservation**: Existing epic/feature/task sections are unchanged (compare before/after)
5. **Round-trip**: Apply profile, read config, verify `GetWorkflowForLevel("bug")` returns profile-defined workflow

**Test File Location**: `internal/init/profiles_test.go` (new or extend)

**Coverage Target**: Profile loading, key presence, and round-trip deserialization.

---

## 3. Integration Scenarios

These scenarios verify that F01 components work correctly together and with existing Shark infrastructure.

---

### INT-01: Fresh Database Full Initialization

**Goal**: Verify that a completely new project gets all F01 schema objects.

**Preconditions**: No database file exists.

**Steps**:
1. Call `InitDB("test_fresh.db")`
2. Verify `bugs` table exists with correct schema
3. Verify `change_cards` table exists with correct schema
4. Verify all 9 indexes exist (5 bugs + 4 change_cards)
5. Verify `entity_notes` table has no CHECK constraint on entity_type
6. Verify triggers exist: `bugs_updated_at`, `change_cards_updated_at`

**Expected**: All tables, indexes, and triggers created. No errors.

**Traces to UAT**: Scenario 1 (Fresh Database Initialization) in feature PRD.

---

### INT-02: Existing Database Upgrade (Pre-E18)

**Goal**: Verify that a database from before E18 upgrades correctly without data loss.

**Preconditions**: Database with existing epics, features, tasks, and entity_notes (with CHECK constraint).

**Steps**:
1. Create database with `InitDB()` using a pre-E18 codebase simulation (create entity_notes with CHECK constraint, insert sample notes)
2. Call `InitDB()` with the E18-F01 migration code
3. Verify new tables (`bugs`, `change_cards`) exist
4. Verify all existing data (epics, features, tasks, notes) is unchanged
5. Insert entity_note with entity_type = 'bug' -- must succeed
6. Insert entity_note with entity_type = 'change' -- must succeed

**Expected**: Seamless upgrade. Zero data loss. New entity_types accepted.

**Traces to UAT**: Scenario 2 (Existing Database Upgrade) in feature PRD.

---

### INT-03: Workflow Engine + Config Forward Compatibility

**Goal**: Verify that a pre-E18 `.sharkconfig.json` (without bug/change workflow sections) works correctly with the extended workflow engine.

**Preconditions**: `.sharkconfig.json` with only epic, feature, task workflow sections.

**Steps**:
1. Load pre-E18 config file
2. Construct `MultiLevelWorkflow` from config (Bug and Change fields will be nil)
3. Call `GetWorkflowForLevel("bug")` -- must return `DefaultBugWorkflow()`
4. Call `GetWorkflowForLevel("change")` -- must return `DefaultChangeCardWorkflow()`
5. Call `GetWorkflowForLevel("epic")` -- must return same result as before E18

**Expected**: Graceful fallback to defaults. No panics. No errors. Existing levels unaffected.

**Traces to UAT**: Scenario 6 (Old Config File Compatibility), CE-5 (Config Persistence).

---

### INT-04: Profile Update + Workflow Resolution

**Goal**: Verify that applying a workflow profile makes the bug/change workflows available through `GetWorkflowForLevel()`.

**Preconditions**: Project initialized with basic profile (no bug/change workflow).

**Steps**:
1. Apply advanced profile (`shark init update --workflow=advanced`)
2. Reload config from `.sharkconfig.json`
3. Call `GetWorkflowForLevel("bug")` -- must return profile-defined workflow (not default)
4. Call `GetWorkflowForLevel("change")` -- must return profile-defined workflow
5. Call `GetWorkflowForLevel("epic")` -- must still return epic workflow (not overwritten)

**Expected**: Profile workflows take precedence over defaults. Existing levels preserved.

**Traces to UAT**: Scenario 5 (Workflow Profile Application), CE-2 (Workflow Engine Extension).

---

### INT-05: Entity Notes Cross-Entity Compatibility

**Goal**: Verify that after migration, entity_notes works for all 5 entity types.

**Preconditions**: Database with migrated entity_notes table.

**Steps**:
1. Insert notes with entity_type = 'epic', 'feature', 'task' (existing types)
2. Insert notes with entity_type = 'bug', 'change' (new types)
3. Query all notes -- all 5 exist
4. Delete a task -- verify cascade deletes only task's notes
5. Verify bug and change notes still exist (no cascade yet -- triggers are in F02/F03)

**Expected**: All entity types coexist. Existing cascade triggers work for epic/feature/task. No cascade for bug/change (expected -- triggers added in F02/F03).

**Traces to UAT**: CE-4 (Pattern Consistency), J1-HP (bug notes), J2-HP (change-card notes).

---

### INT-06: Regression -- Existing Entity Workflows Unchanged

**Goal**: Confirm that adding bug/change workflow levels does not alter existing epic, feature, or task workflow behavior.

**Preconditions**: Default `MultiLevelWorkflow{}`.

**Steps**:
1. Call `GetWorkflowForLevel("epic")` -- verify returns `DefaultEpicWorkflow()` (4 statuses)
2. Call `GetWorkflowForLevel("feature")` -- verify returns default feature workflow (4 statuses)
3. Call `GetWorkflowForLevel("task")` -- verify returns default task workflow (5 statuses)
4. Call `GetWorkflowForLevel("unknown")` -- verify falls back to task default

**Expected**: Identical behavior to pre-E18 code for all existing levels.

**Traces to UAT**: CE-2 (Workflow Engine Extension) -- regression validation.

---

## 4. Epic UAT Traceability

This section maps F01 test cases back to the epic-level UAT scenarios (from `E18-UAT-ACCEPTANCE-PLAN.md`) to show which UAT requirements F01 satisfies.

| Epic UAT Scenario | F01 Test Cases | F01 Coverage | Remaining Coverage |
|-------------------|----------------|--------------|-------------------|
| J1-HP (Bug Happy Path) | TC-S1-01 through TC-S1-10 (table exists), TC-S3-05 (workflow transitions), TC-S5-01 (notes) | Schema and workflow definitions exist | F02 (repository/service), F04 (CLI commands) |
| J1-ERR-1 (Invalid Transition) | TC-S3-05 (terminal statuses have empty arrays) | Workflow transition rules defined | F02 (service enforcement), F06 (unified dispatch) |
| J2-HP (Change-Card Happy Path) | TC-S2-01 through TC-S2-06 (table exists), TC-S4-04 (workflow transitions), TC-S5-02 (notes) | Schema and workflow definitions exist | F03 (repository/service), F05 (CLI commands) |
| J2-ERR-1 (Re-approve) | TC-S4-09 (declined only from proposed) | Workflow rules defined | F03 (service enforcement) |
| CE-2 (Workflow Engine Extension) | TC-S3-02, TC-S3-03, TC-S4-02, TC-S4-03, INT-03, INT-04, INT-06 | ForLevel() dispatch works for bug and change | F02/F03 (runtime enforcement via service) |
| CE-5 (Config Persistence) | TC-S6-01 through TC-S6-06, INT-03, INT-04 | Profile includes bug/change workflows; config round-trip works | Runtime validation (F02/F03) |
| Scenario 1 (Fresh DB) | INT-01 | Fully covered by F01 | N/A |
| Scenario 2 (Existing DB Upgrade) | INT-02 | Fully covered by F01 | N/A |
| Scenario 3 (Bug Workflow Resolution) | TC-S3-02, TC-S3-05, TC-S3-09 | Workflow config returns correct statuses and transitions | F02 (service uses workflow) |
| Scenario 4 (Change-Card Workflow Resolution) | TC-S4-02, TC-S4-04, TC-S4-08 | Workflow config returns correct statuses and transitions | F03 (service uses workflow) |
| Scenario 5 (Workflow Profile Application) | TC-S6-01 through TC-S6-06, INT-04 | Profiles generate correct config sections | N/A |
| Scenario 6 (Old Config Compatibility) | TC-NF-01, TC-NF-02, TC-NF-03, INT-03 | Fully covered by F01 | N/A |

---

## Test Execution Summary

| Category | Test Count | Type | Test File |
|----------|-----------|------|-----------|
| Story 1 (bugs table) | 10 | DB integration | `internal/db/bugs_table_test.go` |
| Story 2 (change_cards table) | 6 | DB integration | `internal/db/change_cards_table_test.go` |
| Story 3 (bug workflow) | 9 | Unit | `internal/config/workflow_multilevel_test.go`, `internal/config/workflow_default_test.go` |
| Story 4 (change-card workflow) | 9 | Unit | Same as Story 3 |
| Story 5 (entity_notes migration) | 7 | DB integration | `internal/db/entity_notes_migration_test.go` |
| Story 6 (workflow profiles) | 6 | Unit + integration | `internal/init/profiles_test.go` |
| Non-functional (forward compat) | 3 | Unit | Inline with workflow tests |
| Integration scenarios | 6 | Integration | Distributed across test files |
| **Total** | **56** | | |

---

## Exit Gate Checklist

- [x] Every acceptance criterion has at least one test case with inputs and expected outputs
- [x] Edge cases identified for constraint enforcement, idempotency, and backward compatibility
- [x] Component test strategy defined for all 4 components (DB schema, workflow engine, default workflows, profiles)
- [x] Integration scenarios cover cross-component interactions
- [x] Test types follow codebase conventions (real DB for `internal/db/`, unit tests for `internal/config/`)
- [x] Test file locations specified using existing naming patterns
- [x] Epic UAT traceability shows which F01 tests satisfy which epic-level scenarios
- [x] Actionable for TDD -- developers can write tests first from this plan

---

*Last Updated*: 2026-03-03
