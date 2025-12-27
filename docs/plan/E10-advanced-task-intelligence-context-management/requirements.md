# Requirements

**Epic**: [Advanced Task Intelligence & Context Management](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for this epic.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization** aligned with the phased implementation plan:
- **Must Have** (Phase 1): Critical for AI agent workflow; epic fails without these
- **Should Have** (Phase 2): Important for full functionality; high value but deferrable
- **Could Have** (Phase 3): Valuable enhancements; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

## Must Have Requirements (Phase 1)

### Category 1: Task Notes & Activity Log

**REQ-F-001**: Task Note Creation with Type Classification
- **Description**: System must allow adding typed notes to tasks with classifications: comment, decision, blocker, solution, reference, implementation, testing, future, question
- **User Story**: As an AI agent, I want to record implementation decisions with specific types so that I can categorize and retrieve them later
- **Acceptance Criteria**:
  - [ ] CLI command `shark task note add <task-key> --type <type> "<content>"` exists
  - [ ] Nine note types supported: comment, decision, blocker, solution, reference, implementation, testing, future, question
  - [ ] Notes include timestamp, creator (agent ID or username), task_id
  - [ ] Notes cannot be created for non-existent tasks
- **Related Journey**: Journey 1 (AI Agent Task Execution), Steps 2-4
- **Schema**: `task_notes` table with columns: id, task_id, note_type, content, created_by, created_at

**REQ-F-002**: Task Note Retrieval and Filtering
- **Description**: System must allow viewing all notes for a task, optionally filtered by type
- **User Story**: As a tech lead, I want to view only decision notes for a task so that I can understand the implementation approach
- **Acceptance Criteria**:
  - [ ] CLI command `shark task notes <task-key>` lists all notes chronologically
  - [ ] CLI command `shark task notes <task-key> --type <type>` filters by note type
  - [ ] Output shows note type, timestamp, creator, and content
  - [ ] Notes are ordered by created_at ascending (oldest first)
- **Related Journey**: Journey 3 (Tech Lead Review), Step 2

**REQ-F-003**: Task Timeline View
- **Description**: System must provide unified chronological view of status changes and notes
- **User Story**: As an AI agent resuming work, I want to see the complete timeline of what happened so that I understand the task history
- **Acceptance Criteria**:
  - [ ] CLI command `shark task timeline <task-key>` exists
  - [ ] Output interleaves status changes from task_history and notes from task_notes
  - [ ] Each entry shows timestamp, event type (status change or note type), content, and actor
  - [ ] Timeline ordered chronologically (ascending)
- **Related Journey**: Journey 2 (Resume Paused Task), Step 3

**REQ-F-004**: Cross-Task Note Search
- **Description**: System must support searching note content across all tasks
- **User Story**: As a developer, I want to search for "singleton pattern" across all task notes so that I can find related implementations
- **Acceptance Criteria**:
  - [ ] CLI command `shark notes search "<query>"` exists
  - [ ] Search is case-insensitive
  - [ ] Results show task key, note type, timestamp, and matching content
  - [ ] Optional filter by epic: `shark notes search "<query>" --epic E13`
- **Related Journey**: Journey 5 (Discover Related Implementation), Step 1

---

### Category 2: Completion Metadata

**REQ-F-005**: Completion Metadata Capture
- **Description**: System must capture detailed metadata when tasks are completed
- **User Story**: As an AI agent, I want to record files modified and verification status so that reviewers know what was done
- **Acceptance Criteria**:
  - [ ] `shark task complete` accepts flags: `--files-created`, `--files-modified`, `--tests`, `--summary`, `--verified`
  - [ ] Metadata stored as JSON in `completion_metadata` column
  - [ ] `verification_status` field updated based on `--verified` flag (verified/not_verified)
  - [ ] Agent ID captured automatically from system context
- **Related Journey**: Journey 1 (AI Agent Execution), Step 5
- **Schema**: ALTER TABLE tasks ADD COLUMN completion_metadata TEXT, verification_status TEXT, verification_notes TEXT

**REQ-F-006**: Completion Details Retrieval
- **Description**: System must display completion metadata when viewing tasks
- **User Story**: As a tech lead, I want to see what files were modified when reviewing a task so that I can verify the scope
- **Acceptance Criteria**:
  - [ ] `shark task get <task-key> --completion-details` shows parsed metadata
  - [ ] Output includes: files created, files modified, tests added/status, verification status, agent ID
  - [ ] Human-readable formatting (not raw JSON)
- **Related Journey**: Journey 3 (Tech Lead Review), Step 1

**REQ-F-007**: File-Based Task Discovery
- **Description**: System must support finding tasks that modified specific files
- **User Story**: As a developer, I want to find all tasks that modified "useTheme.ts" so that I understand its change history
- **Acceptance Criteria**:
  - [ ] CLI command `shark search --file "<filename>"` exists
  - [ ] Searches completion_metadata JSON for file matches (exact and partial)
  - [ ] Returns task key, title, status, and completion date
  - [ ] Supports filtering by epic or feature
- **Related Journey**: Journey 5 (Discover Related Implementation), Step 2

---

### Category 3: Task Context & Resume Data

**REQ-F-008**: Structured Context Storage
- **Description**: System must allow storing structured JSON context data for tasks
- **User Story**: As an AI agent, I want to record my current step and open questions so that I can resume efficiently
- **Acceptance Criteria**:
  - [ ] ALTER TABLE tasks ADD COLUMN context_data TEXT (JSON)
  - [ ] `shark task context set <task-key> --field <field> "<value>"` updates specific field
  - [ ] Supported fields: current_step, completed_steps, remaining_steps, implementation_decisions, open_questions
  - [ ] Context data is valid JSON
- **Related Journey**: Journey 1 (AI Agent Execution), Alt Path B
- **Schema**: JSON structure with fields: progress, implementation_decisions, open_questions, blockers, acceptance_criteria_status, related_tasks

**REQ-F-009**: Resume Command with Full Context
- **Description**: System must provide comprehensive context retrieval for resuming tasks
- **User Story**: As an AI agent, I want one command that gives me everything I need to resume work so that I don't waste time gathering context
- **Acceptance Criteria**:
  - [ ] CLI command `shark task resume <task-key>` exists
  - [ ] Output includes: task details, all notes (chronological), context_data, acceptance criteria status, work sessions, related tasks
  - [ ] Human-readable formatting with sections for each context type
  - [ ] Highlights open questions and blockers prominently
- **Related Journey**: Journey 2 (Resume Paused Task), Step 2

---

## Should Have Requirements (Phase 2)

### Category 4: Task Relationships

**REQ-F-010**: Bidirectional Task Relationships
- **Description**: System must support typed relationships between tasks
- **User Story**: As a PM, I want to see which tasks depend on T-E13-F05-003 so that I can understand downstream impacts
- **Acceptance Criteria**:
  - [ ] CREATE TABLE task_relationships with columns: id, from_task_id, to_task_id, relationship_type, created_at
  - [ ] Relationship types: depends_on, blocks, related_to, follows, spawned_from, duplicates, references
  - [ ] Foreign keys enforce task existence
  - [ ] Cascade delete when tasks are deleted
- **Related Journey**: Journey 1 (AI Agent Execution), Alt Path A; Journey 4 (PM Tracks Progress), Step 4
- **Schema**: `task_relationships` table with CHECK constraint on relationship_type

**REQ-F-011**: Relationship Management Commands
- **Description**: System must provide CLI commands for creating and viewing relationships
- **User Story**: As an AI agent, I want to link T-E13-F05-004 as depending on T-E13-F05-003 so that dependencies are tracked
- **Acceptance Criteria**:
  - [ ] `shark task link <task-key> --depends-on <other-task>` creates depends_on relationship
  - [ ] Similar flags for other relationship types: --blocks, --related-to, --follows, --spawned-from
  - [ ] `shark task deps <task-key>` shows all relationships (incoming and outgoing)
  - [ ] `shark task graph <task-key>` generates dependency graph visualization
- **Related Journey**: Journey 4 (PM Tracks Progress), Step 4

**REQ-F-012**: Relationship Querying
- **Description**: System must support queries based on relationships
- **User Story**: As a developer, I want to find what tasks are blocked by T-E13-F05-003 so that I know what will unblock
- **Acceptance Criteria**:
  - [ ] `shark task blocked-by <task-key>` shows incoming blocks relationships
  - [ ] `shark task blocks <task-key>` shows outgoing blocks relationships
  - [ ] `shark task related <task-key>` shows all related tasks regardless of type
  - [ ] Output includes relationship type and direction
- **Related Journey**: Journey 4 (PM Tracks Progress), Step 3

---

### Category 5: Acceptance Criteria Tracking

**REQ-F-013**: Criteria Storage and Management
- **Description**: System must store acceptance criteria as structured database records
- **User Story**: As an AI agent, I want to check off acceptance criteria as I complete them so that I track progress
- **Acceptance Criteria**:
  - [ ] CREATE TABLE task_criteria with columns: id, task_id, criterion, status, verified_at, verification_notes
  - [ ] Status values: pending, in_progress, complete, failed, na
  - [ ] `shark task criteria import <task-key>` parses markdown acceptance criteria into DB
  - [ ] `shark task criteria add <task-key> "<criterion>"` manually adds criterion
- **Related Journey**: Journey 3 (Tech Lead Review), Step 3
- **Schema**: `task_criteria` table with CHECK constraint on status

**REQ-F-014**: Criteria Progress Tracking
- **Description**: System must allow checking off and viewing criteria progress
- **User Story**: As a tech lead, I want to see 7/7 criteria met so that I know the task is complete
- **Acceptance Criteria**:
  - [ ] `shark task criteria check <task-key> <criterion-id>` marks criterion complete
  - [ ] `shark task criteria fail <task-key> <criterion-id> --note "<reason>"` marks criterion failed
  - [ ] `shark task criteria <task-key>` shows progress summary (X/Y complete)
  - [ ] Output uses checkmarks ✓ for complete, ✗ for failed, ○ for pending
- **Related Journey**: Journey 3 (Tech Lead Review), Step 3

**REQ-F-015**: Feature-Level Criteria Aggregation
- **Description**: System must aggregate criteria across all tasks in a feature
- **User Story**: As a PM, I want to see 28/35 criteria complete for feature E13-F05 so that I can report progress
- **Acceptance Criteria**:
  - [ ] `shark feature criteria <feature-key>` aggregates across all feature tasks
  - [ ] Output shows total, complete, in_progress, failed, pending counts
  - [ ] Percentage calculation included
  - [ ] Option to show breakdown by task
- **Related Journey**: Journey 4 (PM Tracks Progress), Step 2

---

### Category 6: Enhanced Search

**REQ-F-016**: Full-Text Search Across Task Data
- **Description**: System must support full-text search across titles, descriptions, notes, and metadata
- **User Story**: As a developer, I want to search "singleton pattern" and find all related tasks so that I can learn from existing implementations
- **Acceptance Criteria**:
  - [ ] `shark search "<query>"` searches tasks.title, tasks.description, task_notes.content, completion_metadata
  - [ ] Case-insensitive search
  - [ ] Results ranked by relevance (title match > note match > metadata match)
  - [ ] Optional filters: --status, --epic, --feature, --agent
- **Related Journey**: Journey 5 (Discover Related Implementation), Step 1

**REQ-F-017**: Search by Note Type
- **Description**: System must allow searching within specific note types
- **User Story**: As a tech lead, I want to search decision notes for "composable" so that I find architectural decisions
- **Acceptance Criteria**:
  - [ ] `shark search "<query>" --note-type decision` limits to specific note types
  - [ ] Multiple types supported: `--note-type decision,solution`
  - [ ] Results show note type and timestamp
- **Related Journey**: Journey 5 (Discover Related Implementation), Step 1

---

## Could Have Requirements (Phase 3)

### Category 7: Work Sessions

**REQ-F-018**: Work Session Tracking
- **Description**: System must track individual work sessions for tasks
- **User Story**: As a PM, I want to see that task X had 3 work sessions totaling 4 hours so that I can improve estimates
- **Acceptance Criteria**:
  - [ ] CREATE TABLE work_sessions with columns: id, task_id, agent_id, started_at, ended_at, outcome
  - [ ] `shark task start` automatically creates new work session
  - [ ] `shark task session pause <task-key> --note "<reason>"` ends session with outcome 'paused'
  - [ ] `shark task complete` ends session with outcome 'completed'
- **Related Journey**: Journey 1 (AI Agent Execution), Steps 1 and 5
- **Schema**: `work_sessions` table with outcome CHECK constraint

**REQ-F-019**: Session History and Analytics
- **Description**: System must provide visibility into work session patterns
- **User Story**: As a PM, I want to see average session duration so that I can understand interruption patterns
- **Acceptance Criteria**:
  - [ ] `shark task sessions <task-key>` shows all sessions with duration
  - [ ] `shark analytics --session-duration --epic E13` calculates average duration
  - [ ] `shark analytics --pause-frequency` shows how often tasks are paused
  - [ ] Output includes total time, session count, average duration
- **Related Journey**: Journey 4 (PM Tracks Progress), implied analytics need

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: Search Performance
- **Description**: Full-text search must return results within 2 seconds for databases with <10,000 tasks
- **Measurement**: Execute `shark search` with various queries and measure response time
- **Target**: p95 < 2 seconds
- **Justification**: Developers need quick discovery; slow search breaks flow

**REQ-NF-002**: Note Retrieval Performance
- **Description**: Retrieving all notes for a task must complete in <500ms
- **Measurement**: `shark task notes <task-key>` execution time
- **Target**: p99 < 500ms
- **Justification**: AI agents retrieve notes frequently on resume; must be fast

**REQ-NF-003**: Timeline Generation Performance
- **Description**: Generating timeline view must complete in <1 second for tasks with <500 events
- **Measurement**: `shark task timeline <task-key>` execution time
- **Target**: p95 < 1 second
- **Justification**: Timeline is information-dense; users will tolerate slight delay

---

### Data Integrity

**REQ-NF-004**: Foreign Key Enforcement
- **Description**: All relationships must enforce referential integrity via foreign keys
- **Implementation**: task_notes.task_id → tasks.id, task_relationships.from_task_id → tasks.id, etc.
- **Compliance**: SQLite PRAGMA foreign_keys = ON
- **Risk Mitigation**: Prevents orphaned notes, criteria, or relationships

**REQ-NF-005**: Cascade Deletion
- **Description**: Deleting a task must cascade delete all related notes, criteria, relationships, and sessions
- **Implementation**: FOREIGN KEY ... ON DELETE CASCADE
- **Compliance**: SQLite foreign key constraints
- **Risk Mitigation**: Prevents data inconsistency and orphaned records

**REQ-NF-006**: JSON Validation
- **Description**: context_data and completion_metadata fields must contain valid JSON
- **Implementation**: Validate JSON before insert/update, return error on invalid JSON
- **Testing**: Attempt to insert malformed JSON, verify error returned
- **Risk Mitigation**: Prevents data corruption and parsing errors

---

### Usability

**REQ-NF-007**: Human-Readable Output
- **Description**: All CLI commands must provide human-readable output by default (not JSON)
- **Implementation**: Format tables, use unicode symbols (✓✗○), colorize status
- **Testing**: Manual review of CLI output for readability
- **Risk Mitigation**: Improves developer experience and adoption

**REQ-NF-008**: JSON Output Mode
- **Description**: All CLI commands must support --json flag for machine-readable output
- **Implementation**: Marshal results to JSON when --json flag present
- **Testing**: Verify all commands produce valid, parseable JSON with --json
- **Risk Mitigation**: Enables AI agent automation and scripting

---

### Backward Compatibility

**REQ-NF-009**: Database Migration
- **Description**: All schema changes must be applied via automatic migrations
- **Implementation**: Use migration system in internal/db/migrations.go
- **Testing**: Test migration on existing database, verify no data loss
- **Risk Mitigation**: Existing Shark installations upgrade seamlessly

**REQ-NF-010**: CLI Compatibility
- **Description**: Existing CLI commands must continue to work unchanged
- **Implementation**: New features are additive (new flags, new commands), not breaking changes
- **Testing**: Run existing CLI test suite, verify no regressions
- **Risk Mitigation**: Prevents breaking user workflows

---

### Security

**REQ-NF-011**: Note Content Sanitization
- **Description**: Task notes must sanitize HTML/SQL to prevent injection attacks
- **Implementation**: Escape special characters before display, use parameterized queries
- **Compliance**: OWASP injection prevention guidelines
- **Risk Mitigation**: Protects against malicious content in notes

**REQ-NF-012**: Agent ID Validation
- **Description**: Agent IDs must be validated before storage
- **Implementation**: Regex validation, max length enforcement
- **Testing**: Attempt to store excessively long or malformed agent IDs
- **Risk Mitigation**: Prevents data corruption and potential exploits

---

*See also*: [Success Metrics](./success-metrics.md), [Scope](./scope.md)
