# UAT Test Guide - Notes and Context for Epic and Feature

**Feature:** E16-F04 - Notes and Context for Epic and Feature
**Epic:** E16 - Multi-Level Workflow System
**Generated:** 2026-02-10
**Status:** PASSED (2026-02-11)
**Results:** [UAT-E16-F04-20260211-175609-results.md](results/UAT-E16-F04-20260211-175609-results.md)

---

## Epic Context

**Epic Goal:** Extend shark's configurable workflow engine from task-only to epic, feature, and task levels with level-specific status flows, orchestrator actions, and cascading state management.

**This Feature's Role:** E16-F04 extends the note and context system (previously task-only) to epic and feature entities, enabling agents to record decisions, findings, blockers, and context at all entity levels. This is essential for agents performing epic/feature-level planning work (BAs, architects, researchers).

**Related Features:**
- E16-F01 (Core Workflow Engine) - completed - Provides workflow status/position context
- E16-F02 (Orchestrator Actions) - completed - Returns orchestrator actions in transitions
- E16-F03 (Display & Aggregation Threshold) - completed - Planning vs aggregation display
- E16-F05 (Backward Transition & Escalation) - draft - Not yet started
- E16-F06 (Workflow Visualization) - draft - Not yet started

**Integration Points:**
- Resume commands integrate with workflow position from E16-F01
- Notes and context enrich `shark get` output alongside aggregation display from E16-F03
- Entity_notes table is polymorphic - supports epic, feature, and task entities

---

## Design Intent

**From Epic PRD (FR-10):**
> 1. `shark epic note add <key> --type <type> "<content>"` -- add notes to epics
> 2. `shark feature note add <key> --type <type> "<content>"` -- add notes to features
> 3. `shark epic context set <key> --field <field> --value "<value>"` -- set context on epics
> 4. `shark feature context set <key> --field <field> --value "<value>"` -- set context on features
> 5. Same note types as tasks: comment, decision, blocker, solution, reference, implementation, testing, future, question
> 6. `shark epic resume <key>` and `shark feature resume <key>` -- get full context for resuming work

**From Feature PRD:**
> REQ-F-001 through REQ-F-012 covering: note add/list for epic/feature, context set/get for epic/feature, resume for epic/feature, enhanced get commands showing notes/context.

**Key Design Decisions:**
- Polymorphic entity_notes table with entity_type + entity_id pattern
- ContextData stored as JSON in context_data column on epics and features tables
- Service layer pattern: NoteService, ContextService, ResumeService
- Same note types as existing task notes (10 types including rejection)

---

## Cross-Feature Integration Tests

### Integration Scenario 1: Resume with Workflow Position
**Features:** E16-F01 (Core Workflow Engine) + E16-F04 (Notes & Context)
**Scenario:** Resume command shows workflow position from E16-F01 alongside notes/context from E16-F04

### Integration Scenario 2: Enhanced Get with Aggregation Display
**Features:** E16-F03 (Display & Aggregation) + E16-F04 (Notes & Context)
**Scenario:** `shark get E16-F04` shows both aggregation progress and notes/context

---

## Feature Acceptance Validation

| Feature AC | Description | Status |
|------------|-------------|--------|
| AC-1 | `shark epic note add` works with all note types | [ ] |
| AC-2 | `shark feature note add` works with all note types | [ ] |
| AC-3 | `shark epic context set/get` works | [ ] |
| AC-4 | `shark feature context set/get` works | [ ] |
| AC-5 | `shark epic resume` shows notes, context, history, status | [ ] |
| AC-6 | `shark feature resume` shows notes, context, history, status | [ ] |
| AC-7 | `shark get` for epic/feature includes notes in output | [ ] |
| AC-8 | Notes search supports entity-type filter | [ ] |
| AC-9 | Database schema supports notes/context for all entity types | [ ] |

---

## Test Scenarios

### Scenario 1: Database Schema and Migration
**Tasks covered:** T-E16-F04-004

**Steps:**
1. Verify entity_notes table exists with correct schema
2. Verify context_data columns on epics and features tables
3. Verify indexes exist

**Success Criteria:**
- [ ] entity_notes table with entity_type, entity_id, note_type, content, created_by, metadata, created_at
- [ ] epics table has context_data column
- [ ] features table has context_data column

### Scenario 2: EntityNote Model and Repository
**Tasks covered:** T-E16-F04-005

**Steps:**
1. Verify EntityNote model validation
2. Verify CRUD operations via repository
3. Verify search functionality

**Success Criteria:**
- [ ] Model validates entity type, entity ID, note type, and content
- [ ] Repository CRUD (Create, GetByID, GetByEntity, Delete) works
- [ ] Search across entities works

### Scenario 3: Context Data for Epic and Feature
**Tasks covered:** T-E16-F04-006

**Steps:**
1. Verify ContextData model (JSON serialization/deserialization)
2. Verify epic repository context methods
3. Verify feature repository context methods

**Success Criteria:**
- [ ] ContextData model serializes/deserializes correctly
- [ ] Epic context get/update works
- [ ] Feature context get/update works

### Scenario 4: NoteService and ContextService
**Tasks covered:** T-E16-F04-007

**Steps:**
1. Verify NoteService resolves entity keys correctly
2. Verify ContextService field updates with merge semantics
3. Verify service-level validation

**Success Criteria:**
- [ ] NoteService adds notes via entity key (not raw ID)
- [ ] ContextService set/get/clear work
- [ ] Invalid fields rejected

### Scenario 5: Epic Note and Context CLI Commands
**Tasks covered:** T-E16-F04-008

**Steps:**
1. Add note to epic via CLI
2. List notes for epic via CLI
3. Set context field on epic via CLI
4. Get context for epic via CLI

**Success Criteria:**
- [ ] `shark epic note add` persists note
- [ ] `shark epic notes` lists notes
- [ ] `shark epic context set` persists context field
- [ ] `shark epic context get` shows context
- [ ] JSON output works for all commands

### Scenario 6: Feature Note and Context CLI Commands
**Tasks covered:** T-E16-F04-009

**Steps:**
1. Add note to feature via CLI
2. List notes for feature via CLI
3. Set context field on feature via CLI
4. Get context for feature via CLI

**Success Criteria:**
- [ ] `shark feature note add` persists note
- [ ] `shark feature notes` lists notes
- [ ] `shark feature context set` persists context field
- [ ] `shark feature context get` shows context
- [ ] JSON output works for all commands

### Scenario 7: Resume CLI Commands
**Tasks covered:** T-E16-F04-010

**Steps:**
1. Test epic resume command
2. Test feature resume command
3. Verify resume includes notes, context, status, and child entity summaries

**Success Criteria:**
- [ ] `shark epic resume` shows epic overview, notes, context, feature summaries
- [ ] `shark feature resume` shows feature overview, notes, context, task summaries
- [ ] JSON output works

### Scenario 8: Enhanced Get Commands
**Tasks covered:** T-E16-F04-011

**Steps:**
1. Test `shark get <epic-key>` includes notes
2. Test `shark get <feature-key>` includes notes
3. Verify JSON output includes notes array

**Success Criteria:**
- [ ] Epic get output includes notes section
- [ ] Feature get output includes notes section
- [ ] JSON output has notes array

### Scenario 9: Notes Search with Entity-Type Filter
**Tasks covered:** T-E16-F04-003

**Steps:**
1. Search notes with --entity-type filter
2. Verify filtering narrows results correctly

**Success Criteria:**
- [ ] `shark notes search <query> --entity-type epic` only returns epic notes
- [ ] `shark notes search <query> --entity-type feature` only returns feature notes

### Scenario 10: Integration Tests and Edge Cases
**Tasks covered:** T-E16-F04-012

**Steps:**
1. Run full test suite
2. Verify edge cases (empty content, invalid types, missing entities)

**Success Criteria:**
- [ ] All tests pass (`make test`)
- [ ] Edge cases handled gracefully

### Scenario 11: Work Sessions Investigation
**Tasks covered:** T-E16-F04-002

**Steps:**
1. Review investigation findings about work_sessions removal

**Success Criteria:**
- [ ] Investigation documented with clear recommendation

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | (none yet) |
| Result | - |
| Results File | - |

**Previous Sessions:** None
