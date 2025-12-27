# Feature PRD: E10-F01 Task Activity & Notes System

**Feature Key**: E10-F01
**Epic**: [E10: Advanced Task Intelligence & Context Management](../epic.md)
**Status**: Draft
**Priority**: Must Have (Phase 1)
**Execution Order**: 1

---

## Goal

### Problem

AI development agents and human developers lose critical implementation context when working on tasks because decisions, blockers, solutions, and external references are not systematically captured. When resuming work after pausing or switching contexts, agents waste time re-analyzing codebases and rediscovering decisions already made. Tech leads reviewing completed work have no visibility into the reasoning behind implementation choices.

**Real Example from E13**: During T-E13-F05-002 (Flash Prevention Script), the agent made a critical decision to use IIFE pattern instead of module to avoid async loading delay, found a Safari-specific flash fix by moving the script before the viewport meta tag, and referenced a similar pattern from shadcn-vue. None of this context was captured in a structured, searchable way - forcing anyone reviewing or continuing the work to reverse-engineer the reasoning from code.

### Solution

Create a rich, typed note system integrated into the task lifecycle that allows:
- **AI agents** to record decisions, blockers, solutions, and references during implementation with structured note types
- **Tech leads** to review implementation reasoning and troubleshooting history without manual code analysis
- **Developers** to search across all task notes to discover related implementations and proven patterns
- **Complete timeline view** combining status changes and notes for full task history context

The system introduces a `task_notes` table with nine typed note categories (decision, blocker, solution, reference, implementation, testing, future, question, comment) and CLI commands for creating, filtering, searching, and viewing notes in timeline format.

### Impact

**For AI Agents**:
- **Zero context loss on resume**: See all decisions, blockers, and solutions in chronological order
- **50% faster resume time**: Read timeline instead of re-analyzing entire codebase
- **Reduced re-work**: Avoid rediscovering decisions already made in previous sessions

**For Tech Leads**:
- **30% faster review time**: Instant visibility into implementation approach via decision notes
- **Quality confidence**: See testing notes and solutions to problems encountered
- **Knowledge transfer**: Understand "why" behind implementations, not just "what"

**For Developers**:
- **Pattern discovery**: Find related implementations through searchable decision and solution notes (5+ uses per week)
- **Learning acceleration**: Learn from proven patterns and avoided pitfalls documented in notes

**For Product Managers**:
- **Blocker visibility**: Track blocker patterns to identify systemic issues requiring intervention
- **Work quality**: Verify completeness through implementation and testing notes

---

## User Personas

### Persona 1: AI Development Agent (Claude Code)

**Profile**:
- **Role/Title**: AI-powered development agent performing backend, frontend, and test implementation
- **Experience Level**: Expert technical knowledge but limited session continuity (must resume work across conversation boundaries)
- **Key Characteristics**:
  - Executes complex multi-step tasks requiring decisions, research, and iteration
  - Works asynchronously with frequent pause/resume cycles as conversations end
  - Benefits from structured context to avoid re-analyzing codebases on resume

**Goals Related to This Feature**:
1. Record implementation decisions and rationale during task execution
2. Document blockers encountered and solutions discovered
3. Resume paused tasks with full context of what was done and what remains
4. Reference external patterns and documentation for future learning

**Pain Points This Feature Addresses**:
- **Context Loss on Resume**: When resuming a paused task, agents must re-read files and rediscover decisions already made
- **Invisible Implementation Reasoning**: No structured way to capture "why we chose X over Y"
- **Lost Troubleshooting Knowledge**: Solutions to problems (e.g., "Safari flash fix") are lost if not documented

**Success Looks Like**:
AI agents can pause work at any point, resume hours or days later with complete context (including decisions made, blockers encountered, and solutions discovered), eliminating wasted time re-analyzing code.

---

### Persona 2: Human Developer (Technical Lead)

**Profile**:
- **Role/Title**: Senior developer or tech lead managing development workflow and code quality
- **Experience Level**: 5+ years development experience, responsible for task review and approval
- **Key Characteristics**:
  - Reviews task completions from AI agents and human developers
  - Needs to quickly understand what was done and verify quality
  - Troubleshoots blockers and investigates related implementations

**Goals Related to This Feature**:
1. Quickly understand implementation approach and reasoning during code review
2. Verify that problems encountered were properly addressed
3. Search for tasks that implemented similar features or patterns
4. Identify common blockers across multiple tasks

**Pain Points This Feature Addresses**:
- **Opaque Implementation Reasoning**: Can't see "why" decisions were made without reading all code and commit messages
- **Hidden Problem-Solving**: Solutions to bugs/issues aren't documented, leading to repeated debugging
- **Difficult Knowledge Discovery**: No way to search "which tasks used singleton pattern" or "how did we solve Safari flash issues"

**Success Looks Like**:
Tech leads can review a completed task and immediately see all architectural decisions, problems encountered with solutions, and testing verification - approving confidently in <5 minutes instead of 15+ minutes of code analysis.

---

### Persona 3: Product Manager

**Profile**:
- **Role/Title**: Product manager overseeing feature development and delivery
- **Experience Level**: 3+ years product management, focuses on delivery velocity and risk management
- **Key Characteristics**:
  - Tracks feature completion and identifies blockers preventing progress
  - Needs visibility into what's blocking work and why
  - Reports on delivery status and risks to stakeholders

**Goals Related to This Feature**:
1. Identify blocked tasks and understand root causes
2. Track recurring blocker patterns to escalate systemic issues
3. Verify work quality through testing and implementation notes
4. Report on task progress with confidence in completeness

**Pain Points This Feature Addresses**:
- **Hidden Blockers**: Blockers are buried in task history notes, not categorized or searchable
- **No Blocker Analytics**: Can't answer "What's blocking us most frequently?"
- **Quality Uncertainty**: Can't easily verify that tasks were properly tested and verified

**Success Looks Like**:
Product managers can identify all blocked tasks with categorized blocker types, analyze blocker patterns to escalate systemic issues (e.g., "5 tasks blocked by missing dark mode CSS variables"), and verify work quality through structured testing notes.

---

## User Stories

### Must-Have Stories

**Story 1**: As an AI development agent, I want to record implementation decisions with specific types (decision/solution/blocker) so that I can resume work with full context and other developers can understand my reasoning.

**Acceptance Criteria**:
- [ ] CLI command `shark task note add <task-key> --type <type> "<content>"` exists
- [ ] Nine note types supported: comment, decision, blocker, solution, reference, implementation, testing, future, question
- [ ] Notes automatically capture timestamp, creator (agent ID or username), and task association
- [ ] Notes cannot be created for non-existent tasks
- [ ] Note content is required (cannot be empty)

---

**Story 2**: As a tech lead, I want to view only decision notes for a task so that I can quickly understand the implementation approach without reading all comments.

**Acceptance Criteria**:
- [ ] CLI command `shark task notes <task-key>` lists all notes chronologically
- [ ] CLI command `shark task notes <task-key> --type decision` filters to decision notes only
- [ ] CLI command supports multiple types: `--type decision,solution`
- [ ] Output shows note type, timestamp, creator, and full content
- [ ] Notes ordered by created_at ascending (oldest first)

---

**Story 3**: As an AI agent resuming work, I want to see a complete timeline of status changes and notes so that I understand the full task history without manual analysis.

**Acceptance Criteria**:
- [ ] CLI command `shark task timeline <task-key>` exists
- [ ] Output interleaves status changes from task_history and notes from task_notes
- [ ] Each entry shows timestamp, event type (status or note type), content, and actor
- [ ] Timeline ordered chronologically ascending (oldest to newest)
- [ ] Status changes formatted distinctly from notes (e.g., "Status: todo → in_progress")

---

**Story 4**: As a developer, I want to search for "singleton pattern" across all task notes so that I can find related implementations and learn from existing approaches.

**Acceptance Criteria**:
- [ ] CLI command `shark notes search "<query>"` searches all task_notes.content
- [ ] Search is case-insensitive
- [ ] Results show task key, note type, timestamp, and matching content
- [ ] Optional filter `--epic <epic-key>` limits search to tasks in specific epic
- [ ] Optional filter `--type <type>` searches only specific note types
- [ ] Results ordered by relevance then date descending

---

### Should-Have Stories

**Story 5**: As a product manager, I want to find all tasks with blocker notes so that I can identify common blocker patterns and escalate systemic issues.

**Acceptance Criteria**:
- [ ] `shark notes search --type blocker` returns all blocker notes across all tasks
- [ ] Output groups blockers by epic/feature for pattern analysis
- [ ] Can filter by date range: `--after <date> --before <date>`

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I try to add a note to a non-existent task, I want to receive a clear error message so that I can correct my mistake.

**Acceptance Criteria**:
- [ ] Error message: "Task T-XYZ not found"
- [ ] Exit code 1 (not found)
- [ ] Suggests checking task key with `shark task list`

**Error Story 2**: As a user, when I try to add a note with an invalid type, I want to see all valid types so that I can choose the correct one.

**Acceptance Criteria**:
- [ ] Error message: "Invalid note type: xyz (must be one of: comment, decision, blocker, solution, reference, implementation, testing, future, question)"
- [ ] Exit code 3 (invalid state)

---

## Requirements

### Functional Requirements

**Category: Note Creation & Storage**

1. **REQ-F-001**: Task Note Creation with Type Classification
   - **Description**: System must allow adding typed notes to tasks with nine distinct classifications: comment, decision, blocker, solution, reference, implementation, testing, future, question
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CLI command `shark task note add <task-key> --type <type> "<content>"` exists
     - [ ] Nine note types supported with CHECK constraint enforcement
     - [ ] Notes include timestamp, creator (agent ID or username), task_id
     - [ ] Notes cannot be created for non-existent tasks (foreign key constraint)
     - [ ] Note content is required (NOT NULL constraint)

---

**Category: Note Retrieval & Filtering**

2. **REQ-F-002**: Task Note Retrieval and Filtering
   - **Description**: System must allow viewing all notes for a task, optionally filtered by type
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CLI command `shark task notes <task-key>` lists all notes chronologically
     - [ ] CLI command `shark task notes <task-key> --type <type>` filters by note type
     - [ ] Multiple types supported: `--type decision,solution`
     - [ ] Output shows note type, timestamp, creator, and content
     - [ ] Notes ordered by created_at ascending (oldest first)
     - [ ] `--json` flag outputs structured JSON

---

**Category: Timeline & History**

3. **REQ-F-003**: Task Timeline View
   - **Description**: System must provide unified chronological view of status changes and notes
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CLI command `shark task timeline <task-key>` exists
     - [ ] Output interleaves status changes from task_history and notes from task_notes
     - [ ] Each entry shows timestamp, event type, content, and actor
     - [ ] Timeline ordered chronologically ascending
     - [ ] Status changes formatted distinctly from notes
     - [ ] `--json` flag outputs structured timeline

---

**Category: Search & Discovery**

4. **REQ-F-004**: Cross-Task Note Search
   - **Description**: System must support searching note content across all tasks
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CLI command `shark notes search "<query>"` exists
     - [ ] Search is case-insensitive
     - [ ] Results show task key, note type, timestamp, and matching content
     - [ ] Optional filter `--epic <epic-key>` limits to specific epic
     - [ ] Optional filter `--type <type>` searches specific note types
     - [ ] Results ordered by relevance then date descending

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-002**: Note Retrieval Performance
   - **Description**: Retrieving all notes for a task must complete in <500ms
   - **Measurement**: Execute `shark task notes <task-key>` and measure execution time
   - **Target**: p99 < 500ms for tasks with up to 100 notes
   - **Justification**: AI agents retrieve notes frequently on resume; slow retrieval breaks flow

2. **REQ-NF-003**: Timeline Generation Performance
   - **Description**: Generating timeline view must complete in <1 second
   - **Measurement**: Execute `shark task timeline <task-key>` and measure execution time
   - **Target**: p95 < 1 second for tasks with up to 500 events (status changes + notes combined)
   - **Justification**: Timeline is information-dense; users will tolerate slight delay for comprehensive view

**Data Integrity**

3. **REQ-NF-004**: Foreign Key Enforcement
   - **Description**: All note records must enforce referential integrity via foreign keys
   - **Implementation**: `task_notes.task_id` → `tasks.id` with `ON DELETE CASCADE`
   - **Compliance**: SQLite `PRAGMA foreign_keys = ON`
   - **Risk Mitigation**: Prevents orphaned notes when tasks are deleted

4. **REQ-NF-005**: Cascade Deletion
   - **Description**: Deleting a task must cascade delete all related notes
   - **Implementation**: FOREIGN KEY ... ON DELETE CASCADE
   - **Compliance**: SQLite foreign key constraints
   - **Risk Mitigation**: Prevents data inconsistency and orphaned records

**Usability**

5. **REQ-NF-007**: Human-Readable Output
   - **Description**: All CLI commands must provide human-readable output by default (not JSON)
   - **Implementation**: Format tables, use labels `[TYPE]`, colorize timestamps
   - **Testing**: Manual review of CLI output for readability
   - **Risk Mitigation**: Improves developer experience and adoption

6. **REQ-NF-008**: JSON Output Mode
   - **Description**: All CLI commands must support `--json` flag for machine-readable output
   - **Implementation**: Marshal results to JSON when `--json` flag present
   - **Testing**: Verify all commands produce valid, parseable JSON with `--json`
   - **Risk Mitigation**: Enables AI agent automation and scripting

**Security**

7. **REQ-NF-011**: Note Content Sanitization
   - **Description**: Task notes must sanitize HTML/SQL to prevent injection attacks
   - **Implementation**: Escape special characters before display, use parameterized queries
   - **Compliance**: OWASP injection prevention guidelines
   - **Risk Mitigation**: Protects against malicious content in notes

---

## Database Schema

### New Table: task_notes

```sql
CREATE TABLE task_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    note_type TEXT CHECK (note_type IN (
        'comment',         -- General observation
        'decision',        -- Why we chose X over Y
        'blocker',         -- What's blocking progress
        'solution',        -- How we solved a problem
        'reference',       -- External links, documentation
        'implementation',  -- What we actually built
        'testing',         -- Test results, coverage
        'future',          -- Future improvements / TODO
        'question'         -- Unanswered questions
    )) NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT,  -- 'claude' or username
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Index for fast retrieval of all notes for a task
CREATE INDEX idx_task_notes_task_id ON task_notes(task_id);

-- Index for filtering by note type
CREATE INDEX idx_task_notes_type ON task_notes(note_type);

-- Index for searching note content (full-text search)
CREATE INDEX idx_task_notes_content ON task_notes(content);

-- Composite index for filtered queries
CREATE INDEX idx_task_notes_task_type ON task_notes(task_id, note_type);
```

**Schema Design Rationale**:
- **TEXT for created_by**: No users table in single-user CLI; TEXT allows flexible agent IDs and usernames
- **CHECK constraint for note_type**: Nine fixed types unlikely to change; simpler and faster than separate types table
- **ON DELETE CASCADE**: Notes meaningless without parent task; cascade prevents orphaned records
- **Index on content**: Enables fast cross-task search for REQ-F-004

---

## CLI Commands Specification

### Command: `shark task note add`

**Purpose**: Add a typed note to a task

**Syntax**:
```bash
shark task note add <task-key> --type <type> "<content>" [--created-by <creator>]
```

**Arguments**:
- `<task-key>`: Required. Task key (e.g., T-E13-F05-002)
- `--type <type>`: Required. Note type (comment|decision|blocker|solution|reference|implementation|testing|future|question)
- `"<content>"`: Required. Note content (quoted string)
- `--created-by <creator>`: Optional. Override creator (default: inferred from system context)

**Examples**:
```bash
# Record implementation decision
shark task note add T-E13-F05-002 --type decision "Used IIFE pattern instead of module to avoid async loading delay"

# Document external reference
shark task note add T-E13-F05-002 --type reference "Similar pattern: https://github.com/shadcn-ui/ui/blob/main/apps/www/public/themes.js"

# Capture solution to problem
shark task note add T-E13-F05-002 --type solution "Safari flash fix: Moved script BEFORE viewport meta tag"

# Track blocker
shark task note add T-E13-F05-004 --type blocker "Blocked by missing useTheme composable - see T-E13-F05-003"
```

**Output**:
```
Note added to T-E13-F05-002

[DECISION] 2025-12-26 14:15 (claude)
Used IIFE pattern instead of module to avoid async loading delay
```

**Error Cases**:
- Task not found → Exit code 1, message "Task T-XYZ not found"
- Invalid note type → Exit code 3, message "Invalid note type: xyz (must be one of: ...)"
- Empty content → Exit code 3, message "Note content cannot be empty"

---

### Command: `shark task notes`

**Purpose**: View all notes for a task

**Syntax**:
```bash
shark task notes <task-key> [--type <type>] [--json]
```

**Arguments**:
- `<task-key>`: Required. Task key
- `--type <type>`: Optional. Filter by note type (comma-separated for multiple)
- `--json`: Optional. Output JSON

**Examples**:
```bash
# View all notes
shark task notes T-E13-F05-002

# View only decision notes
shark task notes T-E13-F05-002 --type decision

# View decision and solution notes
shark task notes T-E13-F05-002 --type decision,solution
```

**Output (Human-Readable)**:
```
Task T-E13-F05-002: Implement Flash Prevention Script (5 notes)

[DECISION] 2025-12-26 14:15 (claude)
Used IIFE pattern instead of module to avoid async loading delay

[REFERENCE] 2025-12-26 14:18 (claude)
Similar pattern used in shadcn-vue: https://github.com/...

[IMPLEMENTATION] 2025-12-26 14:20 (claude)
Added to index.html:9-35

[TESTING] 2025-12-26 14:25 (claude)
Verified in Chrome, Firefox

[SOLUTION] 2025-12-26 14:30 (claude)
Safari flash fix: Moved script BEFORE viewport meta tag
```

---

### Command: `shark task timeline`

**Purpose**: View unified timeline of status changes and notes

**Syntax**:
```bash
shark task timeline <task-key> [--json]
```

**Example**:
```bash
shark task timeline T-E13-F05-002
```

**Output**:
```
Task T-E13-F05-002: Implement Flash Prevention Script

Timeline:
  2025-12-26 06:22  Created                               (jwwelbor)
  2025-12-26 14:15  [DECISION] Used IIFE pattern          (claude)
  2025-12-26 14:20  [IMPLEMENTATION] Added to index.html  (claude)
  2025-12-26 14:25  [TESTING] Verified in Chrome, Firefox (claude)
  2025-12-26 14:30  [SOLUTION] Safari flash fix           (claude)
  2025-12-26 17:45  Status: todo → ready_for_review       (jwwelbor)
  2025-12-26 17:45  Status: ready_for_review → completed  (jwwelbor)
```

---

### Command: `shark notes search`

**Purpose**: Search note content across all tasks

**Syntax**:
```bash
shark notes search "<query>" [--epic <epic>] [--feature <feature>] [--type <type>] [--json]
```

**Arguments**:
- `"<query>"`: Required. Search query (case-insensitive)
- `--epic <epic-key>`: Optional. Filter to epic
- `--feature <feature-key>`: Optional. Filter to feature
- `--type <type>`: Optional. Filter to note types (comma-separated)
- `--json`: Optional. Output JSON

**Examples**:
```bash
# Search all notes
shark notes search "singleton pattern"

# Search decision notes only
shark notes search "singleton pattern" --type decision

# Search within epic
shark notes search "dark mode" --epic E13
```

**Output**:
```
Found 2 results for "singleton pattern":

T-E13-F05-003: Create useTheme() Composable
  [DECISION] 2025-12-26 14:15 (claude)
  Used singleton pattern for theme state to ensure single source of truth

T-E13-F02-004: Create useContentColor Composable
  [DECISION] 2025-12-25 16:30 (claude)
  Considered singleton pattern but chose factory pattern for flexibility
```

---

## User Journeys

### Journey 1: AI Agent Captures Implementation Context

**Persona**: AI Development Agent

**Scenario**: Agent implements T-E13-F05-002 (Flash Prevention Script) and captures decisions, solutions, and references

**Steps**:
1. Agent starts task: `shark task start T-E13-F05-002`
2. Makes architectural decision, captures it:
   ```bash
   shark task note add T-E13-F05-002 --type decision \
     "Used IIFE pattern instead of module to avoid async loading delay"
   ```
3. Finds useful reference, documents it:
   ```bash
   shark task note add T-E13-F05-002 --type reference \
     "Similar pattern: https://github.com/shadcn-ui/ui/blob/main/apps/www/public/themes.js"
   ```
4. Encounters Safari bug, documents solution:
   ```bash
   shark task note add T-E13-F05-002 --type solution \
     "Safari flash fix: Moved script BEFORE viewport meta tag"
   ```

**Outcome**: Full implementation context captured for future reference and review (zero context loss)

---

### Journey 2: AI Agent Resumes Paused Task

**Persona**: AI Development Agent

**Scenario**: Agent paused T-E13-F05-004 yesterday, resumes today with no prior context

**Steps**:
1. Agent identifies next task: `shark task next --agent frontend` → Returns T-E13-F05-004
2. Retrieves full timeline: `shark task timeline T-E13-F05-004`
3. Timeline shows:
   - [DECISION] Component location: frontend/src/components/ThemeToggle.vue
   - [IMPLEMENTATION] Created base component with icon switching
   - [QUESTION] Should theme toggle be in header or settings page?
   - [BLOCKER] Waiting for design decision on placement
4. Agent sees open question, pings human for decision
5. Human responds: "Header"
6. Agent adds resolution: `shark task note add T-E13-F05-004 --type solution "Placement decision: ThemeToggle goes in AppHeader.vue top-right"`
7. Agent continues implementation with full context

**Outcome**: Zero time wasted re-analyzing code; agent resumes immediately with complete understanding

---

### Journey 3: Tech Lead Reviews Implementation

**Persona**: Human Developer (Technical Lead)

**Scenario**: Tech lead reviews completed T-E13-F05-003 before approving

**Steps**:
1. Views task notes to understand approach: `shark task notes T-E13-F05-003 --type decision,implementation`
2. Output shows:
   - [DECISION] Used singleton pattern for theme state
   - [DECISION] Chose ref() over reactive() for single value
   - [IMPLEMENTATION] localStorage persistence with media query listeners
3. Reviews testing: `shark task notes T-E13-F05-003 --type testing`
4. Output shows: "16/16 tests passing - covered all edge cases"
5. Approves confidently: `shark task approve T-E13-F05-003`

**Outcome**: Tech lead approves in <5 minutes with full understanding of implementation quality

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Note Creation**
- **Given** a task T-E13-F05-002 exists in the database
- **When** user executes `shark task note add T-E13-F05-002 --type decision "Used IIFE pattern"`
- **Then** note is created with type="decision", content="Used IIFE pattern", timestamp=now
- **And** note is associated with task_id for T-E13-F05-002
- **And** success message is displayed with note details

**Scenario 2: Note Filtering**
- **Given** task T-E13-F05-002 has 5 notes (2 decision, 1 solution, 2 implementation)
- **When** user executes `shark task notes T-E13-F05-002 --type decision`
- **Then** only 2 decision notes are displayed
- **And** notes are ordered chronologically (oldest first)

**Scenario 3: Timeline View**
- **Given** task T-E13-F05-002 has 3 status changes and 5 notes
- **When** user executes `shark task timeline T-E13-F05-002`
- **Then** all 8 events are displayed in chronological order
- **And** status changes are formatted as "Status: old → new"
- **And** notes are formatted as "[TYPE] content"

**Scenario 4: Cross-Task Search**
- **Given** 10 tasks exist with various notes containing "singleton pattern"
- **When** user executes `shark notes search "singleton pattern" --type decision`
- **Then** all decision notes containing "singleton pattern" are returned
- **And** results show task key, note type, timestamp, and content
- **And** search is case-insensitive

**Scenario 5: Error Handling - Non-Existent Task**
- **Given** task T-XYZ does not exist
- **When** user executes `shark task note add T-XYZ --type decision "Test"`
- **Then** error message "Task T-XYZ not found" is displayed
- **And** exit code is 1 (not found)

---

## Out of Scope

### Explicitly Excluded

1. **Note Editing After Creation**
   - **Why**: Notes are immutable audit trail; editing would compromise historical accuracy
   - **Future**: Not planned; users can add clarification notes instead
   - **Workaround**: Add new note with type "comment" to clarify/correct previous note

2. **Note Deletion**
   - **Why**: Maintains complete audit trail; deletion would create gaps in history
   - **Future**: Not planned unless compliance requires it
   - **Workaround**: Notes are searchable and filterable; irrelevant notes can be ignored

3. **Markdown Formatting in Notes**
   - **Why**: Complexity; plain text sufficient for Phase 1
   - **Future**: Phase 2 enhancement if users request rich formatting
   - **Workaround**: Use external links for rich documentation

4. **Note Reactions/Upvotes**
   - **Why**: Single-user CLI tool; collaborative features not applicable
   - **Future**: Multi-user team version might include reactions
   - **Workaround**: N/A

5. **Automatic Note Extraction from Commits**
   - **Why**: Complex AI/NLP feature; high implementation cost
   - **Future**: Phase 3 enhancement if AI-generated summaries prove valuable
   - **Workaround**: Manual note creation by agents during implementation

---

## Success Metrics

### Primary Metrics

1. **Note Adoption Rate**
   - **What**: Percentage of completed tasks with at least 3 notes (decision, implementation, or solution type)
   - **Target**: 80% of completed tasks
   - **Timeline**: 2 weeks after Phase 1 release
   - **Measurement**: SQL query: `SELECT COUNT(*) FROM tasks WHERE status='completed' AND id IN (SELECT task_id FROM task_notes GROUP BY task_id HAVING COUNT(*) >= 3)`

2. **Resume Efficiency**
   - **What**: Time for AI agent to resume paused task (time to first productive action)
   - **Target**: 50% reduction compared to pre-notes baseline
   - **Timeline**: 1 month after Phase 1 release
   - **Measurement**: AI agent self-report + timing logs

3. **Discovery Usage**
   - **What**: Frequency of `shark notes search` usage
   - **Target**: 5+ searches per developer per week
   - **Timeline**: 1 month after Phase 1 release
   - **Measurement**: CLI usage analytics

---

### Secondary Metrics

- **Review Speed**: Tech lead task review time reduced by 30% (measured via timestamps)
- **Note Type Distribution**: Decision, implementation, and testing notes account for 60% of all notes (indicates quality usage)
- **Search Success Rate**: 80% of searches return at least 1 relevant result (indicates good discoverability)

---

## Dependencies & Integrations

### Dependencies

- **Existing Tables**: Requires `tasks` table (already exists)
- **Existing Tables**: Requires `task_history` table (already exists) for timeline integration
- **CLI Framework**: Uses Cobra command structure in `internal/cli/commands/`
- **Repository Pattern**: Follows existing repository pattern from `internal/repository/`

### Integration Requirements

- **Timeline Integration**: `shark task timeline` must query both `task_history` and `task_notes` tables, merge results chronologically
- **Search Integration**: Note search foundation for E10-F04 (Acceptance Criteria & Search) which extends search to include criteria

### Downstream Dependencies

Features that depend on E10-F01:
- **E10-F03**: Task Relationships & Dependencies (uses blocker notes for dependency reasoning)
- **E10-F04**: Acceptance Criteria & Search (extends search to include notes + criteria)
- **E10-F05**: Work Sessions & Resume Context (resume command includes timeline view)

---

## Implementation Plan

### Database Migration

**Migration File**: `internal/db/migrations/010_add_task_notes.sql`

```sql
CREATE TABLE IF NOT EXISTS task_notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    note_type TEXT CHECK (note_type IN (
        'comment', 'decision', 'blocker', 'solution',
        'reference', 'implementation', 'testing', 'future', 'question'
    )) NOT NULL,
    content TEXT NOT NULL,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_notes_task_id ON task_notes(task_id);
CREATE INDEX idx_task_notes_type ON task_notes(note_type);
CREATE INDEX idx_task_notes_content ON task_notes(content);
CREATE INDEX idx_task_notes_task_type ON task_notes(task_id, note_type);
```

**Backward Compatibility**: No breaking changes to existing commands or schema

---

### Code Organization

- **Repository**: `internal/repository/task_note_repository.go` (new file)
- **Models**: `internal/models/task_note.go` (new file)
- **Commands**:
  - `internal/cli/commands/task_note.go` (new file for `task note add`)
  - `internal/cli/commands/task_notes.go` (new file for `task notes` and `task timeline`)
  - `internal/cli/commands/notes_search.go` (new file for `notes search`)
- **Tests**:
  - `internal/repository/task_note_repository_test.go`
  - `internal/cli/commands/task_note_test.go`

---

### Testing Strategy

1. **Unit Tests**: Repository methods (Create, GetByTask, Search, GetTimeline)
2. **Integration Tests**: Full CLI commands with real database
3. **Performance Tests**: Load 100 notes, measure retrieval and search times
4. **Edge Cases**:
   - Task with zero notes
   - Task with 500+ notes
   - Search with no results
   - Timeline with interleaved events
   - Special characters in note content (HTML, quotes, SQL)

---

## Open Questions

- **Q1**: Should notes support markdown formatting, or plain text only?
  - **Recommendation**: Plain text for Phase 1, markdown in Phase 2 if users request it
- **Q2**: Should there be a character limit on note content?
  - **Recommendation**: No hard limit, but warn if >1000 characters (likely better suited for documentation)
- **Q3**: Should notes be editable after creation?
  - **Recommendation**: No - immutable audit trail. Users can add clarification notes if needed.

---

*Last Updated*: 2025-12-26
*Status*: Ready for Review
*Author*: ProductManager Agent
