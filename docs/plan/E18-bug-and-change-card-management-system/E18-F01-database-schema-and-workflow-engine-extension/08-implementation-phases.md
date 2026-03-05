# E18-F01 Implementation Phases

**Feature**: E18-F01
**Date**: 2026-03-03
**Status**: Draft

---

## Phase Overview

F01 is divided into 4 implementation phases, ordered by dependency. Each phase is independently testable.

| Phase | Description | Size | Dependencies |
|-------|-------------|------|-------------|
| 1 | Workflow level constants and engine extension | S | None |
| 2 | Default workflow definitions | S | Phase 1 |
| 3 | Database schema (new tables + entity_notes migration) | M | None (can run in parallel with Phase 1-2) |
| 4 | Workflow profile integration (basic.json + advanced.json) | S | Phase 2 |

**Total estimated size**: S-M (as noted in PRD)

---

## Phase 1: Workflow Level Constants and Engine Extension

**Goal**: Add `LevelBug` and `LevelChange` to the workflow engine so `ForLevel("bug")` and `ForLevel("change")` work.

### Files Modified

1. **`internal/workflow/levels.go`**
   - Add `LevelBug = "bug"` and `LevelChange = "change"` constants to the existing const block

2. **`internal/config/workflow_multilevel.go`**
   - Add `Bug *WorkflowConfig` and `Change *WorkflowConfig` fields to `MultiLevelWorkflow` struct
   - Add `case "bug":` and `case "change":` branches to `GetWorkflowForLevel()` switch statement
   - Bug case: return `m.Bug` if non-nil, else `DefaultBugWorkflow()`
   - Change case: return `m.Change` if non-nil, else `DefaultChangeCardWorkflow()`

### Implementation Notes

The `GetWorkflowForLevel()` default case currently returns `DefaultWorkflow()` (task default). After this change, unknown levels still return the task default. Only `"bug"` and `"change"` are explicitly handled.

### Test Criteria

- `GetWorkflowForLevel("bug")` returns a non-nil `*WorkflowConfig`
- `GetWorkflowForLevel("change")` returns a non-nil `*WorkflowConfig`
- `GetWorkflowForLevel("bug")` with nil `m.Bug` returns `DefaultBugWorkflow()` result
- `GetWorkflowForLevel("change")` with nil `m.Change` returns `DefaultChangeCardWorkflow()` result
- `GetWorkflowForLevel("bug")` with non-nil `m.Bug` returns the custom config
- Existing epic/feature/task behavior unchanged (regression check)
- `workflow.LevelBug == "bug"` and `workflow.LevelChange == "change"`

### Exit Criteria

`workflowSvc.ForLevel(workflow.LevelBug)` and `workflowSvc.ForLevel(workflow.LevelChange)` return valid Service instances.

---

## Phase 2: Default Workflow Definitions

**Goal**: Implement `DefaultBugWorkflow()` and `DefaultChangeCardWorkflow()` with correct status flows, metadata, and special statuses.

### Files Modified

1. **`internal/config/workflow_default.go`**
   - Add `DefaultBugWorkflow()` function (follow `DefaultEpicWorkflow()` pattern)
   - Add `DefaultChangeCardWorkflow()` function

### DefaultBugWorkflow() Specification

```go
func DefaultBugWorkflow() *WorkflowConfig {
    return &WorkflowConfig{
        Version: DefaultWorkflowVersion,
        StatusFlow: map[string][]string{
            "reported":        {"triaged", "wont_fix", "duplicate"},
            "triaged":         {"in_fix", "wont_fix", "duplicate"},
            "in_fix":          {"in_verification", "wont_fix"},
            "in_verification": {"resolved", "in_fix", "wont_fix"},
            "resolved":        {},
            "wont_fix":        {},
            "duplicate":       {},
        },
        StatusMetadata: map[string]StatusMetadata{
            "reported":        {Color: "red", Description: "Bug reported, awaiting triage", Phase: "planning", ProgressWeight: 0.0, Responsibility: "agent"},
            "triaged":         {Color: "orange", Description: "Bug triaged, ready for fix", Phase: "planning", ProgressWeight: 0.2, Responsibility: "agent"},
            "in_fix":          {Color: "blue", Description: "Fix in progress", Phase: "development", ProgressWeight: 0.5, Responsibility: "agent"},
            "in_verification": {Color: "yellow", Description: "Fix complete, verifying resolution", Phase: "review", ProgressWeight: 0.8, Responsibility: "agent"},
            "resolved":        {Color: "green", Description: "Bug resolved and verified", Phase: "done", ProgressWeight: 1.0},
            "wont_fix":        {Color: "gray", Description: "Bug will not be fixed", Phase: "done", ProgressWeight: 1.0},
            "duplicate":       {Color: "gray", Description: "Duplicate of another bug", Phase: "done", ProgressWeight: 1.0},
        },
        SpecialStatuses: map[string][]string{
            StartStatusKey:    {"reported"},
            CompleteStatusKey: {"resolved", "wont_fix", "duplicate"},
        },
        RequireRejectionReason: true,
    }
}
```

### DefaultChangeCardWorkflow() Specification

```go
func DefaultChangeCardWorkflow() *WorkflowConfig {
    return &WorkflowConfig{
        Version: DefaultWorkflowVersion,
        StatusFlow: map[string][]string{
            "proposed":    {"approved", "declined"},
            "approved":    {"in_progress", "proposed"},
            "in_progress": {"completed", "approved"},
            "completed":   {},
            "declined":    {},
        },
        StatusMetadata: map[string]StatusMetadata{
            "proposed":    {Color: "gray", Description: "Change proposed, awaiting approval", Phase: "planning", ProgressWeight: 0.0, Responsibility: "human"},
            "approved":    {Color: "blue", Description: "Change approved, ready for implementation", Phase: "planning", ProgressWeight: 0.25, Responsibility: "agent"},
            "in_progress": {Color: "yellow", Description: "Change being implemented", Phase: "development", ProgressWeight: 0.5, Responsibility: "agent"},
            "completed":   {Color: "green", Description: "Change completed", Phase: "done", ProgressWeight: 1.0},
            "declined":    {Color: "red", Description: "Change request declined", Phase: "done", ProgressWeight: 1.0},
        },
        SpecialStatuses: map[string][]string{
            StartStatusKey:    {"proposed"},
            CompleteStatusKey: {"completed", "declined"},
        },
        RequireRejectionReason: true,
    }
}
```

### Test Criteria

- `DefaultBugWorkflow()` returns non-nil config with 7 statuses
- Bug status flow: `reported` has transitions to `triaged`, `wont_fix`, `duplicate`
- Bug terminal statuses: `resolved`, `wont_fix`, `duplicate` have empty transition arrays
- Bug start status: `reported`
- Bug progress weights match specification
- `DefaultChangeCardWorkflow()` returns non-nil config with 5 statuses
- Change-card status flow: `proposed` has transitions to `approved`, `declined`
- Change-card terminal statuses: `completed`, `declined` have empty transition arrays
- Change-card start status: `proposed`
- Backward transitions: `in_verification -> in_fix` (bug), `approved -> proposed` (change) are correctly identified

### Exit Criteria

Both functions return valid `*WorkflowConfig` instances that pass `ValidateConfig()`.

---

## Phase 3: Database Schema

**Goal**: Create `bugs` and `change_cards` tables with indexes and triggers. Migrate `entity_notes` to remove CHECK constraint.

### Files Modified

1. **`internal/db/db.go`**
   - Add bugs table creation + indexes + trigger to `createSchema()`
   - Add change_cards table creation + indexes + trigger to `createSchema()`
   - Add `migrateEntityNotesRemoveCheckConstraint()` function
   - Call the new migration function from `runMigrations()`

### Implementation Order Within Phase

1. Add `bugs` table SQL to `createSchema()` -- follows ideas table pattern
2. Add `change_cards` table SQL to `createSchema()` -- follows bugs table pattern
3. Add index creation SQL for both tables
4. Add updated_at triggers for both tables
5. Implement `migrateEntityNotesRemoveCheckConstraint()` -- table recreation migration
6. Add migration call to `runMigrations()`

### entity_notes Migration Implementation

```go
func migrateEntityNotesRemoveCheckConstraint(db *sql.DB) error {
    // Step 1: Check if entity_notes table exists
    // If not, nothing to migrate (fresh DB will create it without CHECK)

    // Step 2: Check if CHECK constraint still exists
    // Query: SELECT sql FROM sqlite_master WHERE type='table' AND name='entity_notes'
    // If sql does NOT contain "CHECK (entity_type IN", return nil (already migrated)

    // Step 3: Run transaction-protected table recreation
    // See 03-data-design.md section 2.3 for full migration strategy
}
```

### Test Criteria

- Fresh database: `bugs` and `change_cards` tables exist after `InitDB()`
- Fresh database: All indexes exist
- Fresh database: `entity_notes` table has NO CHECK constraint on `entity_type`
- Existing database: Tables created alongside existing tables
- Existing database: entity_notes data preserved after migration
- Existing database: entity_notes indexes recreated after migration
- Existing database: entity_notes cascade delete triggers work after migration
- Idempotency: Running `InitDB()` twice does not error
- Constraint: `INSERT INTO bugs (..., severity, ...) VALUES (..., 'invalid', ...)` fails with CHECK violation
- Constraint: `INSERT INTO bugs (..., severity, ...) VALUES (..., 'critical', ...)` succeeds
- Default: Bug with no explicit status gets `reported`
- Default: Bug with no explicit severity gets `medium`
- Default: Change-card with no explicit status gets `proposed`
- After migration: `INSERT INTO entity_notes (..., entity_type, ...) VALUES (..., 'bug', ...)` succeeds
- After migration: `INSERT INTO entity_notes (..., entity_type, ...) VALUES (..., 'change', ...)` succeeds
- After migration: Existing notes with entity_type 'epic', 'feature', 'task' are preserved

### Exit Criteria

`InitDB()` succeeds on both fresh and existing databases. All tables, indexes, triggers, and constraints are in place.

---

## Phase 4: Workflow Profile Integration

**Goal**: Update basic.json and advanced.json to include bug and change-card workflow sections.

### Files Modified

1. **`internal/init/profiles/basic.json`**
   - Add `bug_workflow` section with simplified workflow (matches DefaultBugWorkflow minus agent types)
   - Add `change_workflow` section with simplified workflow (matches DefaultChangeCardWorkflow minus agent types)

2. **`internal/init/profiles/advanced.json`**
   - Add `bug_workflow` section with full workflow including agent type assignments
   - Add `change_workflow` section with full workflow including agent type assignments

### Profile JSON Structure

The new sections follow the same structure as existing `epic_workflow` and `feature_workflow` sections in `advanced.json`:

```json
{
  "bug_workflow": {
    "status_flow_version": "1.0",
    "status_flow": { ... },
    "status_metadata": { ... },
    "special_statuses": { ... },
    "require_rejection_reason": true
  },
  "change_workflow": {
    "status_flow_version": "1.0",
    "status_flow": { ... },
    "status_metadata": { ... },
    "special_statuses": { ... },
    "require_rejection_reason": true
  }
}
```

### Config Deserialization Path

When `shark init update --workflow=advanced` runs:
1. `GetProfileMap("advanced")` loads the JSON file into `map[string]interface{}`
2. The config merger copies all keys (including new `bug_workflow`, `change_workflow`) to `.sharkconfig.json`
3. When the config is later loaded, `MultiLevelWorkflow` deserialization picks up `Bug` and `Change` fields from the `bug_workflow` and `change_workflow` JSON keys

**Verification needed**: Confirm that the config parser maps `bug_workflow` JSON key to `MultiLevelWorkflow.Bug` field and `change_workflow` to `MultiLevelWorkflow.Change`. If the config parser uses a different mapping mechanism (e.g., reading from top-level `status_flow` rather than per-level sections), the parser may need a small update. Check `internal/config/workflow_parser.go` during implementation.

### Test Criteria

- `basic.json` contains `bug_workflow` and `change_workflow` keys
- `advanced.json` contains `bug_workflow` and `change_workflow` keys
- Both profiles parse without JSON errors
- Existing epic/feature/task sections unchanged after adding new sections
- After `shark init update --workflow=advanced`, `.sharkconfig.json` contains bug and change workflow sections
- After `shark init update --workflow=basic`, `.sharkconfig.json` contains simplified bug and change workflow sections
- A pre-E18 `.sharkconfig.json` (without bug/change sections) loads without error; `GetWorkflowForLevel("bug")` falls back to `DefaultBugWorkflow()`

### Exit Criteria

Both profiles include complete bug and change-card workflow sections. Config round-trip (write then read) preserves all workflow definitions.

---

## Dependency Graph

```
Phase 1 (Workflow Constants + Engine)
    |
    v
Phase 2 (Default Workflows) -----> Phase 4 (Profile Integration)

Phase 3 (Database Schema) --- independent, can run in parallel with 1-2
```

Phase 3 has no dependency on Phases 1-2 because the database tables do not reference workflow definitions. The workflow engine is used by the service layer (F02/F03), not by the database layer.

---

## Verification Checklist (All Phases Complete)

- [ ] `InitDB()` succeeds on fresh database
- [ ] `InitDB()` succeeds on existing pre-E18 database
- [ ] `bugs` table has correct schema with CHECK constraint on severity
- [ ] `change_cards` table has correct schema
- [ ] All 9 indexes created (5 for bugs, 4 for change_cards)
- [ ] entity_notes CHECK constraint removed; existing data preserved
- [ ] `GetWorkflowForLevel("bug")` returns correct workflow
- [ ] `GetWorkflowForLevel("change")` returns correct workflow
- [ ] Existing epic/feature/task workflows unchanged
- [ ] `basic.json` and `advanced.json` contain bug/change workflow sections
- [ ] Old config files (without bug/change sections) load without error
- [ ] `make fmt && make lint && make test` passes

---

*Last Updated*: 2026-03-03
