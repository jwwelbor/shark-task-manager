# Feature PRD: E10-F02 Task Completion Intelligence

**Feature Key**: E10-F02
**Epic**: [E10: Advanced Task Intelligence & Context Management](../epic.md)
**Status**: Draft
**Priority**: Must Have (Phase 1)
**Execution Order**: 2

---

## Goal

### Problem

When AI agents and developers complete tasks, critical implementation metadata is lost because the system only captures basic timestamps. Reviewers can't see what files were actually modified, what tests were added, whether verification occurred, or which agent performed the work. This creates several painful scenarios:

**Real Example from E13**: When T-E13-F05-003 (useTheme composable) was completed, there was no structured record that it created two files (`useTheme.ts` and `useTheme.spec.ts`), added 16 tests (all passing), modified `index.html`, and was completed by agent `a5ad46d`. A tech lead reviewing the task had to manually inspect the git diff and test output to understand what was actually delivered. When T-E13-F05-004 (ThemeToggle) later needed to find "which tasks modified index.html", there was no efficient way to discover that T-E13-F05-003 and T-E13-F05-002 both touched that file.

**File Change Discovery Problem**: Developers cannot answer "which tasks modified useTheme.ts?" without manually searching git history or reading every task description. This wastes 10-15 minutes per discovery query and often results in incomplete findings.

**Verification Opacity**: Tasks marked "completed" have no indication whether they were verified through automated tests, manual testing, or not verified at all. QA reviewers waste time asking "Did this task include tests?" instead of seeing structured verification metadata.

### Solution

Enhance task completion workflow to automatically capture and store rich completion metadata including:
- **Files created and modified** during implementation (searchable)
- **Test coverage and status** (e.g., "16/16 passing")
- **Verification method and status** (automated, manual, not verified)
- **Agent execution ID** linking to specific agent session
- **Completion summary** describing what was delivered

The system introduces three new database columns:
1. `completion_metadata` (JSON) - Stores files, tests, agent ID, summary
2. `verification_status` (TEXT) - Enum: not_verified, verified, failed, manual_required
3. `verification_notes` (TEXT) - Additional verification details

CLI enhancements:
- `shark task complete` gains flags for `--files-created`, `--files-modified`, `--tests`, `--summary`, `--verified`
- `shark task get <task-key> --completion-details` displays parsed metadata in human-readable format
- `shark search --file <filename>` discovers all tasks that created or modified specific files

### Impact

**For Tech Leads**:
- **60% faster review time**: Instantly see what files changed and test status without manual code inspection
- **Quality confidence**: Verification status immediately visible (verified/not verified/manual required)
- **Scope verification**: Confirm changes are limited to expected files (prevents scope creep)

**For AI Agents**:
- **Automatic metadata capture**: Agent execution IDs automatically recorded for traceability
- **Structured completion**: Clear completion metadata shows exactly what was delivered
- **Discovery efficiency**: Find related tasks by file changed (5+ uses per week)

**For Developers**:
- **File change history**: Answer "which tasks modified X file?" in <10 seconds instead of 10-15 minutes
- **Pattern discovery**: Find all tasks that created similar files (e.g., "all tasks that created .spec.ts files")
- **Impact analysis**: Identify downstream tasks affected by file changes

**For Product Managers**:
- **Verification visibility**: Track percentage of tasks with verified test coverage
- **Quality metrics**: Measure test coverage trends across features
- **Delivery confidence**: Confirm tasks include documentation artifacts referenced in metadata

---

## User Personas

### Persona 1: AI Development Agent (Claude Code)

**Profile**:
- **Role/Title**: AI-powered development agent performing implementation tasks
- **Experience Level**: Expert technical execution but limited cross-session context
- **Key Characteristics**:
  - Completes tasks by creating/modifying files, adding tests, and verifying functionality
  - Tracks agent execution IDs to link work sessions
  - Benefits from structured completion metadata for future reference

**Goals Related to This Feature**:
1. Record what files were created and modified during task implementation
2. Document test coverage and test status (passing/failing counts)
3. Link task completion to specific agent execution ID for traceability
4. Provide clear completion summary for reviewer understanding

**Pain Points This Feature Addresses**:
- **Lost Implementation Details**: No record of which files were touched during task execution
- **Test Status Opacity**: Test results are mentioned in notes but not structured metadata
- **Traceability Gap**: No link between task and agent execution session

**Success Looks Like**:
AI agents complete tasks with rich metadata automatically captured (files, tests, agent ID), enabling reviewers to understand deliverables instantly and enabling future file-based discovery.

---

### Persona 2: Human Developer (Technical Lead)

**Profile**:
- **Role/Title**: Senior developer or tech lead responsible for code review and quality assurance
- **Experience Level**: 5+ years development experience, reviews 10-20 tasks per week
- **Key Characteristics**:
  - Reviews completed tasks before approving for merge
  - Needs to quickly verify scope, test coverage, and quality
  - Investigates file change history to understand evolution

**Goals Related to This Feature**:
1. Instantly see what files were modified in a completed task
2. Verify test coverage and test status without manual inspection
3. Confirm verification method (automated tests vs manual testing)
4. Find all tasks that modified a specific file for impact analysis

**Pain Points This Feature Addresses**:
- **Manual File Discovery**: Must manually inspect git diffs to see file changes (5-10 minutes per task)
- **Test Verification Overhead**: Must read test output or run tests to verify coverage
- **File History Gaps**: Cannot efficiently answer "which tasks touched file X?"
- **Verification Ambiguity**: No structured indicator of whether task was properly verified

**Success Looks Like**:
Tech leads can review a completed task and see structured metadata showing files changed, test status, verification status, and agent ID - approving in <3 minutes instead of 10+ minutes. Can discover all tasks affecting a file in <10 seconds.

---

### Persona 3: QA Specialist

**Profile**:
- **Role/Title**: Quality assurance specialist responsible for verification before production
- **Experience Level**: 3+ years QA experience, focuses on test coverage and verification
- **Key Characteristics**:
  - Reviews tasks in "ready_for_review" status
  - Needs to verify that testing is adequate before approval
  - Tracks verification status across features

**Goals Related to This Feature**:
1. Quickly identify tasks that lack test coverage
2. Verify that automated tests were run and passed
3. Distinguish between automated vs manual verification
4. Track verification status trends across features

**Pain Points This Feature Addresses**:
- **Test Coverage Uncertainty**: No structured field indicating test status
- **Verification Method Ambiguity**: Can't distinguish automated tests from manual testing
- **Quality Metrics Gap**: Can't easily measure "% of tasks with verified tests"

**Success Looks Like**:
QA specialists can filter tasks by verification status, see structured test metadata (16/16 passing), and approve verified tasks confidently while flagging unverified tasks for additional testing.

---

## User Stories

### Must-Have Stories

**Story 1**: As an AI development agent, I want to record files created and modified during task completion so that reviewers know exactly what was delivered.

**Acceptance Criteria**:
- [ ] CLI command `shark task complete <task-key>` accepts `--files-created <path>` flag (repeatable)
- [ ] CLI command `shark task complete <task-key>` accepts `--files-modified <path>` flag (repeatable)
- [ ] File paths stored in `completion_metadata` JSON under `files_created` and `files_modified` arrays
- [ ] File paths are relative to project root
- [ ] Completion succeeds even if no files specified (optional metadata)

---

**Story 2**: As an AI agent, I want to record test coverage and status when completing a task so that reviewers can verify testing quality.

**Acceptance Criteria**:
- [ ] CLI command `shark task complete` accepts `--tests <summary>` flag (e.g., "16/16 passing")
- [ ] Test summary stored in `completion_metadata` JSON under `test_status` field
- [ ] Test summary is freeform text (allows "16/16 passing", "21/21 tests passing", "no tests required")
- [ ] `--verified` flag sets `verification_status` to "verified"
- [ ] Default `verification_status` is "not_verified" if `--verified` omitted

---

**Story 3**: As a tech lead, I want to see completion metadata when reviewing a task so that I can verify scope and quality without manual code inspection.

**Acceptance Criteria**:
- [ ] CLI command `shark task get <task-key>` accepts `--completion-details` flag
- [ ] Output shows files created (bulleted list)
- [ ] Output shows files modified (bulleted list)
- [ ] Output shows test status (e.g., "16/16 passing")
- [ ] Output shows verification status (verified/not_verified/failed/manual_required)
- [ ] Output shows agent ID (e.g., "a5ad46d")
- [ ] Output shows completion summary (if provided)
- [ ] `--json` flag outputs raw completion_metadata JSON

---

**Story 4**: As a developer, I want to find all tasks that modified a specific file so that I can understand its change history.

**Acceptance Criteria**:
- [ ] CLI command `shark search --file "<filename>"` exists
- [ ] Searches `completion_metadata` JSON for exact filename matches in `files_created` and `files_modified` arrays
- [ ] Supports partial filename match (e.g., "useTheme" matches "useTheme.ts" and "useTheme.spec.ts")
- [ ] Results show task key, title, status, completion date
- [ ] Results ordered by completion date descending (most recent first)
- [ ] Optional filter `--epic <epic-key>` limits search to specific epic
- [ ] Optional filter `--status <status>` filters by task status

---

**Story 5**: As an AI agent, I want to automatically capture my agent execution ID when completing a task so that work sessions are traceable.

**Acceptance Criteria**:
- [ ] CLI command `shark task complete` accepts `--agent-id <id>` flag
- [ ] Agent ID stored in `completion_metadata` JSON under `agent_execution_id` field
- [ ] Agent ID automatically inferred from system context if flag omitted (future enhancement)
- [ ] Agent ID displayed in `--completion-details` output

---

### Should-Have Stories

**Story 6**: As a QA specialist, I want to filter tasks by verification status so that I can identify unverified tasks requiring additional testing.

**Acceptance Criteria**:
- [ ] CLI command `shark task list` accepts `--verification <status>` filter
- [ ] Supported values: verified, not_verified, failed, manual_required
- [ ] Can combine with other filters (e.g., `--status ready_for_review --verification not_verified`)
- [ ] Output shows verification status column

---

**Story 7**: As a product manager, I want to see aggregate verification metrics for a feature so that I can report on quality.

**Acceptance Criteria**:
- [ ] CLI command `shark feature get <feature-key>` shows verification breakdown
- [ ] Output shows: X/Y tasks verified, Z not verified, W failed
- [ ] Percentage calculation included

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I mark a task verified but it has no tests specified, I want a warning so that I can add test details.

**Acceptance Criteria**:
- [ ] Warning message displayed: "Task marked verified but no tests specified (use --tests to document test coverage)"
- [ ] Completion proceeds (warning only, not error)
- [ ] Exit code 0 (success with warning)

**Error Story 2**: As a user, when I try to complete a task that's not in "in_progress" status, I want a clear error so that I follow proper workflow.

**Acceptance Criteria**:
- [ ] Error message: "Cannot complete task in status <status> (must be in_progress)"
- [ ] Exit code 3 (invalid state)
- [ ] Existing error handling preserved (already implemented in base system)

---

## Requirements

### Functional Requirements

**Category: Completion Metadata Capture**

1. **REQ-F-005**: Completion Metadata Capture
   - **Description**: System must capture detailed metadata when tasks are completed, including files created/modified, tests added, agent ID, and summary
   - **User Story**: Links to Story 1, Story 2, Story 5
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task complete` accepts `--files-created <path>` (repeatable flag)
     - [ ] `shark task complete` accepts `--files-modified <path>` (repeatable flag)
     - [ ] `shark task complete` accepts `--tests <summary>` (e.g., "16/16 passing")
     - [ ] `shark task complete` accepts `--summary <text>` (completion summary)
     - [ ] `shark task complete` accepts `--verified` flag (sets verification_status to "verified")
     - [ ] `shark task complete` accepts `--agent-id <id>` (agent execution ID)
     - [ ] Metadata stored as JSON in `completion_metadata` column
     - [ ] `verification_status` field updated based on `--verified` flag (verified/not_verified)
     - [ ] Agent ID captured automatically from system context if available
     - [ ] All flags are optional (completion succeeds with minimal metadata)

---

**Category: Completion Metadata Retrieval**

2. **REQ-F-006**: Completion Details Retrieval
   - **Description**: System must display completion metadata when viewing tasks in human-readable format
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task get <task-key>` accepts `--completion-details` flag
     - [ ] Output includes: files created (bulleted list), files modified (bulleted list), tests added/status, verification status, agent ID, completion summary
     - [ ] Human-readable formatting (not raw JSON)
     - [ ] `--json` flag outputs raw `completion_metadata` JSON
     - [ ] Works for tasks completed before feature release (null/empty metadata handled gracefully)

---

**Category: File-Based Discovery**

3. **REQ-F-007**: File-Based Task Discovery
   - **Description**: System must support finding tasks that created or modified specific files
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] CLI command `shark search --file "<filename>"` exists
     - [ ] Searches `completion_metadata` JSON for file matches (exact and partial)
     - [ ] Searches both `files_created` and `files_modified` arrays
     - [ ] Returns task key, title, status, and completion date
     - [ ] Results ordered by completion date descending
     - [ ] Supports filtering by `--epic <epic-key>`
     - [ ] Supports filtering by `--feature <feature-key>`
     - [ ] Supports filtering by `--status <status>`
     - [ ] `--json` flag outputs structured JSON results

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Completion Metadata Parsing
   - **Description**: Parsing and displaying completion metadata must complete in <200ms
   - **Measurement**: Execute `shark task get <task-key> --completion-details` and measure execution time
   - **Target**: p99 < 200ms
   - **Justification**: Metadata is frequently viewed during review; must be instant

2. **REQ-NF-002**: File Search Performance
   - **Description**: Searching tasks by filename must complete in <2 seconds for databases with <10,000 tasks
   - **Measurement**: Execute `shark search --file "<filename>"` and measure execution time
   - **Target**: p95 < 2 seconds
   - **Justification**: File discovery is critical for developer workflow; slow search breaks productivity

**Data Integrity**

3. **REQ-NF-006**: JSON Validation
   - **Description**: `completion_metadata` field must contain valid JSON
   - **Implementation**: Validate JSON structure before insert/update, return error on invalid JSON
   - **Testing**: Attempt to insert malformed JSON, verify error returned
   - **Risk Mitigation**: Prevents data corruption and parsing errors

4. **REQ-NF-010**: Backward Compatibility
   - **Description**: Existing tasks without completion metadata must display gracefully
   - **Implementation**: Handle null/empty `completion_metadata` fields in display logic
   - **Testing**: View task created before feature release, verify no errors
   - **Risk Mitigation**: Seamless upgrade path for existing databases

**Usability**

5. **REQ-NF-007**: Human-Readable Output
   - **Description**: Completion metadata must display in human-readable format by default
   - **Implementation**: Parse JSON and format as bulleted lists, labeled sections
   - **Testing**: Manual review of `--completion-details` output
   - **Risk Mitigation**: Improves adoption by making metadata accessible to humans

6. **REQ-NF-008**: JSON Output Mode
   - **Description**: All commands must support `--json` flag for machine-readable output
   - **Implementation**: Marshal completion metadata to JSON when `--json` flag present
   - **Testing**: Verify JSON output is valid and parseable
   - **Risk Mitigation**: Enables AI agent automation and scripting

**Security**

7. **REQ-NF-011**: Content Sanitization
   - **Description**: File paths and summaries must be sanitized to prevent injection attacks
   - **Implementation**: Escape special characters, validate paths are relative (not absolute with system paths)
   - **Compliance**: OWASP injection prevention guidelines
   - **Risk Mitigation**: Prevents malicious file paths or summaries

---

## Database Schema

### Schema Changes

```sql
-- Add three new columns to tasks table
ALTER TABLE tasks ADD COLUMN completion_metadata TEXT;  -- JSON blob
ALTER TABLE tasks ADD COLUMN verification_status TEXT
  CHECK (verification_status IN ('not_verified', 'verified', 'failed', 'manual_required'))
  DEFAULT 'not_verified';
ALTER TABLE tasks ADD COLUMN verification_notes TEXT;

-- Index for filtering by verification status
CREATE INDEX idx_tasks_verification_status ON tasks(verification_status);
```

### completion_metadata JSON Structure

```json
{
  "files_created": [
    "frontend/src/composables/useTheme.ts",
    "frontend/src/composables/__tests__/useTheme.spec.ts"
  ],
  "files_modified": [
    "frontend/index.html"
  ],
  "test_status": "16/16 passing",
  "agent_execution_id": "a5ad46d",
  "completion_summary": "Implemented singleton theme composable with localStorage persistence. Supports light/dark/system themes with media query listeners.",
  "documentation_artifacts": [
    "dev-artifacts/2025-12-26-useTheme-composable/verification-report.md",
    "dev-artifacts/2025-12-26-useTheme-composable/task-completion-summary.md"
  ],
  "lines_added": 178,
  "verification_method": "automated_tests"
}
```

**JSON Field Definitions**:
- `files_created` (array): Relative paths to files created during task execution
- `files_modified` (array): Relative paths to files modified during task execution
- `test_status` (string): Freeform summary of test results (e.g., "16/16 passing", "21/21 tests passing")
- `agent_execution_id` (string): Unique identifier linking to agent session that completed task
- `completion_summary` (string): Human-readable description of what was delivered
- `documentation_artifacts` (array): Paths to additional documentation generated during task
- `lines_added` (number): Optional metric for code volume
- `verification_method` (string): How verification was performed (automated_tests, manual_testing, code_review)

**Schema Design Rationale**:
- **JSON for flexibility**: Completion metadata varies by task type; JSON allows extensibility without schema changes
- **Separate verification_status column**: Frequently filtered/queried; dedicated column improves performance vs JSON field
- **verification_notes as TEXT**: Separate from JSON for simplicity; QA reviewers add notes post-completion
- **CHECK constraint on verification_status**: Four fixed states unlikely to change; constraint ensures data integrity

---

## CLI Commands Specification

### Command: `shark task complete` (Enhanced)

**Purpose**: Complete a task with rich completion metadata

**Syntax**:
```bash
shark task complete <task-key> \
  [--files-created <path>]... \
  [--files-modified <path>]... \
  [--tests <summary>] \
  [--summary <text>] \
  [--verified] \
  [--agent-id <id>] \
  [--notes <text>] \
  [--json]
```

**Arguments**:
- `<task-key>`: Required. Task key (e.g., T-E13-F05-003)
- `--files-created <path>`: Optional. Repeatable. File path created during task (relative to project root)
- `--files-modified <path>`: Optional. Repeatable. File path modified during task
- `--tests <summary>`: Optional. Test status summary (e.g., "16/16 passing")
- `--summary <text>`: Optional. Completion summary describing what was delivered
- `--verified`: Optional. Flag indicating task was verified (sets verification_status to "verified")
- `--agent-id <id>`: Optional. Agent execution ID for traceability
- `--notes <text>`: Optional. Additional notes (existing flag, preserved)
- `--json`: Optional. Output JSON

**Examples**:
```bash
# Minimal completion (no metadata)
shark task complete T-E13-F05-003

# Complete with full metadata
shark task complete T-E13-F05-003 \
  --files-created "frontend/src/composables/useTheme.ts" \
  --files-created "frontend/src/composables/__tests__/useTheme.spec.ts" \
  --files-modified "frontend/index.html" \
  --tests "16/16 passing" \
  --summary "Implemented singleton theme composable with localStorage persistence" \
  --verified \
  --agent-id "a5ad46d"

# Complete with verification failure
shark task complete T-E13-F05-007 \
  --tests "3/5 failing" \
  --verification-status failed \
  --notes "Integration tests failing due to missing dark mode CSS"
```

**Output**:
```
Task T-E13-F05-003 completed

Status: in_progress → ready_for_review

Completion Metadata:
  Files Created:
    - frontend/src/composables/useTheme.ts
    - frontend/src/composables/__tests__/useTheme.spec.ts

  Files Modified:
    - frontend/index.html

  Tests: 16/16 passing
  Verification: verified
  Agent ID: a5ad46d

  Summary: Implemented singleton theme composable with localStorage persistence
```

**Warning Cases**:
- `--verified` specified but no `--tests` → Warning: "Task marked verified but no tests specified"
- No metadata flags specified → No warning, completion proceeds normally

**Error Cases**:
- Task not in "in_progress" status → Error: "Cannot complete task in status <status> (must be in_progress)"
- Task not found → Error: "Task T-XYZ not found"

---

### Command: `shark task get` (Enhanced)

**Purpose**: View task details with optional completion metadata display

**Syntax**:
```bash
shark task get <task-key> [--completion-details] [--json]
```

**New Flag**:
- `--completion-details`: Display parsed completion metadata in human-readable format

**Examples**:
```bash
# Standard task view (unchanged)
shark task get T-E13-F05-003

# View with completion details
shark task get T-E13-F05-003 --completion-details

# JSON output with raw metadata
shark task get T-E13-F05-003 --json
```

**Output (with --completion-details)**:
```
Task: T-E13-F05-003
Title: Create useTheme() Composable
Status: completed
Epic: E13 - Dark Mode Support
Feature: E13-F05 - Dark Mode Feature

Completion Details:
  Completed: 2025-12-26 15:45
  Agent: a5ad46d
  Verification: verified

  Files Created:
    - frontend/src/composables/useTheme.ts
    - frontend/src/composables/__tests__/useTheme.spec.ts

  Files Modified:
    - frontend/index.html

  Tests: 16/16 passing

  Summary:
  Implemented singleton theme composable with localStorage persistence.
  Supports light/dark/system themes with media query listeners.

  Documentation Artifacts:
    - dev-artifacts/2025-12-26-useTheme-composable/verification-report.md
    - dev-artifacts/2025-12-26-useTheme-composable/task-completion-summary.md
```

---

### Command: `shark search --file` (New)

**Purpose**: Find all tasks that created or modified a specific file

**Syntax**:
```bash
shark search --file "<filename>" [--epic <epic>] [--feature <feature>] [--status <status>] [--json]
```

**Arguments**:
- `--file <filename>`: Required. Filename or partial path to search for
- `--epic <epic-key>`: Optional. Filter to specific epic
- `--feature <feature-key>`: Optional. Filter to specific feature
- `--status <status>`: Optional. Filter by task status
- `--json`: Optional. Output JSON

**Examples**:
```bash
# Find all tasks that modified useTheme.ts
shark search --file "useTheme.ts"

# Partial filename search
shark search --file "useTheme"  # Matches useTheme.ts and useTheme.spec.ts

# Filter by epic
shark search --file "index.html" --epic E13

# Filter by status
shark search --file "useTheme" --status completed
```

**Output**:
```
Found 3 tasks matching file "useTheme":

T-E13-F05-003: Create useTheme() Composable (completed)
  Completed: 2025-12-26 15:45
  Files Created: useTheme.ts, useTheme.spec.ts

T-E13-F05-004: Create ThemeToggle Component (completed)
  Completed: 2025-12-26 18:30
  Files Modified: useTheme.ts (imported in ThemeToggle.vue)

T-E13-F05-007: Dark Mode Integration Testing (in_progress)
  Files Modified: useTheme.spec.ts (added integration tests)
```

---

### Command: `shark task list` (Enhanced - Should-Have)

**Purpose**: List tasks with optional verification status filter

**New Flag**:
- `--verification <status>`: Filter by verification status (verified, not_verified, failed, manual_required)

**Example**:
```bash
# Find unverified tasks ready for review
shark task list --status ready_for_review --verification not_verified
```

---

## User Journeys

### Journey 1: AI Agent Completes Task with Metadata

**Persona**: AI Development Agent

**Scenario**: Agent completes T-E13-F05-003 (useTheme composable) and captures full completion metadata

**Steps**:
1. Agent finishes implementation, runs tests, sees 16/16 passing
2. Agent completes task with metadata:
   ```bash
   shark task complete T-E13-F05-003 \
     --files-created "frontend/src/composables/useTheme.ts" \
     --files-created "frontend/src/composables/__tests__/useTheme.spec.ts" \
     --files-modified "frontend/index.html" \
     --tests "16/16 passing" \
     --summary "Implemented singleton theme composable with localStorage persistence" \
     --verified \
     --agent-id "a5ad46d"
   ```
3. System stores metadata in `completion_metadata` JSON, sets `verification_status` to "verified"
4. Agent sees confirmation with parsed metadata display

**Outcome**: Full completion context captured for review and future discovery (zero data loss)

---

### Journey 2: Tech Lead Reviews Completion Metadata

**Persona**: Human Developer (Technical Lead)

**Scenario**: Tech lead reviews completed T-E13-F05-003 to verify scope and quality

**Steps**:
1. Views completion details: `shark task get T-E13-F05-003 --completion-details`
2. Sees structured output:
   - Files Created: useTheme.ts, useTheme.spec.ts
   - Files Modified: index.html
   - Tests: 16/16 passing
   - Verification: verified
   - Agent: a5ad46d
3. Confirms scope (only expected files modified), test coverage adequate, verification complete
4. Approves task: `shark task approve T-E13-F05-003 --notes "Excellent test coverage"`

**Outcome**: Tech lead approves confidently in <3 minutes without manual code inspection (60% time reduction)

---

### Journey 3: Developer Discovers File Change History

**Persona**: Human Developer (Technical Lead)

**Scenario**: Developer needs to find all tasks that modified `index.html` to understand its evolution

**Steps**:
1. Searches by file: `shark search --file "index.html"`
2. System queries `completion_metadata` JSON fields
3. Results show:
   - T-E13-F05-002: Flash Prevention Script (modified index.html)
   - T-E13-F05-003: useTheme Composable (modified index.html)
   - T-E13-F01-001: Initial HTML structure (created index.html)
4. Developer reviews each task to understand changes
5. Discovers pattern: Multiple tasks modify index.html for theme initialization

**Outcome**: Developer discovers complete file history in <10 seconds instead of 10-15 minutes of git log analysis

---

### Journey 4: QA Specialist Identifies Unverified Tasks

**Persona**: QA Specialist

**Scenario**: QA needs to find all ready_for_review tasks lacking automated tests

**Steps**:
1. Filters by verification status: `shark task list --status ready_for_review --verification not_verified`
2. System returns 3 tasks without verified tests
3. QA reviews each task, adds manual testing notes
4. Updates verification status: `shark task update T-E13-F05-006 --verification-status manual_required --verification-notes "Requires UAT for accessibility"`

**Outcome**: QA identifies unverified tasks systematically, ensuring quality gates are met before approval

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Metadata Capture on Completion**
- **Given** task T-E13-F05-003 is in "in_progress" status
- **When** user executes `shark task complete T-E13-F05-003 --files-created "useTheme.ts" --tests "16/16 passing" --verified`
- **Then** task status changes to "ready_for_review"
- **And** `completion_metadata` JSON includes `{"files_created": ["useTheme.ts"], "test_status": "16/16 passing"}`
- **And** `verification_status` is "verified"
- **And** completion timestamp is recorded

**Scenario 2: Completion Details Display**
- **Given** task T-E13-F05-003 has completion metadata stored
- **When** user executes `shark task get T-E13-F05-003 --completion-details`
- **Then** output shows files created (bulleted list)
- **And** output shows files modified (bulleted list)
- **And** output shows test status ("16/16 passing")
- **And** output shows verification status ("verified")
- **And** output shows agent ID
- **And** output is human-readable (not raw JSON)

**Scenario 3: File-Based Discovery**
- **Given** 3 tasks have created or modified "useTheme.ts"
- **When** user executes `shark search --file "useTheme.ts"`
- **Then** all 3 tasks are returned
- **And** results show task key, title, status, completion date
- **And** results ordered by completion date descending (most recent first)

**Scenario 4: Partial Filename Matching**
- **Given** task T-E13-F05-003 created "useTheme.ts" and "useTheme.spec.ts"
- **When** user executes `shark search --file "useTheme"`
- **Then** task T-E13-F05-003 is returned (matches both files)
- **And** matched files are indicated in output

**Scenario 5: Backward Compatibility**
- **Given** task T-E13-F02-001 was completed before feature release (no completion metadata)
- **When** user executes `shark task get T-E13-F02-001 --completion-details`
- **Then** output shows "No completion metadata available"
- **And** no errors occur
- **And** other task details display normally

**Scenario 6: Verification Warning**
- **Given** task T-E13-F05-004 is in "in_progress" status
- **When** user executes `shark task complete T-E13-F05-004 --verified` (no --tests flag)
- **Then** task completes successfully
- **And** warning message displayed: "Task marked verified but no tests specified"
- **And** `verification_status` is "verified"

---

## Out of Scope

### Explicitly Excluded

1. **Automatic File Detection**
   - **Why**: Requires git integration or file system monitoring; high complexity for Phase 1
   - **Future**: Phase 3 enhancement - auto-detect files in git diff and suggest for metadata
   - **Workaround**: Manual specification via `--files-created` and `--files-modified` flags

2. **Test Result Parsing**
   - **Why**: Varies by test framework (Jest, pytest, Go testing); complex parsing logic
   - **Future**: Phase 2 enhancement - parse test output from common frameworks
   - **Workaround**: Freeform `--tests` flag allows any summary format

3. **Diff Display in Completion Details**
   - **Why**: Requires git integration; out of scope for task management system
   - **Future**: Not planned; users can view diffs via `git diff`
   - **Workaround**: Use `shark task get --completion-details` for file list, then `git diff <file>`

4. **Metadata Editing After Completion**
   - **Why**: Completion metadata is immutable audit trail; editing would compromise integrity
   - **Future**: Not planned; re-complete task if metadata incorrect
   - **Workaround**: Add clarification in `verification_notes` field

5. **Line Count Statistics**
   - **Why**: Requires git integration or code analysis; not critical for Phase 1
   - **Future**: Phase 3 enhancement if metrics prove valuable
   - **Workaround**: Manually add `lines_added` field to completion_metadata JSON if needed

---

## Success Metrics

### Primary Metrics

1. **Metadata Adoption Rate**
   - **What**: Percentage of completed tasks with completion_metadata populated
   - **Target**: 70% of completed tasks (at least files_created or files_modified populated)
   - **Timeline**: 2 weeks after Phase 1 release
   - **Measurement**: SQL query: `SELECT COUNT(*) FROM tasks WHERE status='completed' AND completion_metadata IS NOT NULL AND completion_metadata != '{}'`

2. **Review Speed Improvement**
   - **What**: Average time for tech leads to review completed tasks
   - **Target**: 60% reduction in review time (from 10 minutes to 4 minutes)
   - **Timeline**: 1 month after Phase 1 release
   - **Measurement**: Manual timing logs + tech lead survey

3. **File Discovery Usage**
   - **What**: Frequency of `shark search --file` usage
   - **Target**: 5+ searches per developer per week
   - **Timeline**: 1 month after Phase 1 release
   - **Measurement**: CLI usage analytics

---

### Secondary Metrics

- **Verification Rate**: 60% of completed tasks have verification_status = "verified" (indicates quality emphasis)
- **Test Documentation**: 50% of completed tasks include `--tests` metadata (indicates testing culture)
- **Discovery Success Rate**: 80% of file searches return at least 1 result (indicates good metadata completeness)

---

## Dependencies & Integrations

### Dependencies

- **Existing Tables**: Requires `tasks` table (already exists)
- **Existing Commands**: Extends `shark task complete` command (already exists in `internal/cli/commands/task_complete.go`)
- **Existing Commands**: Extends `shark task get` command (already exists in `internal/cli/commands/task_get.go`)
- **CLI Framework**: Uses Cobra command structure in `internal/cli/commands/`
- **Repository Pattern**: Follows existing repository pattern from `internal/repository/task_repository.go`

### Integration Requirements

- **Task Completion Flow**: Modify existing `shark task complete` to accept new metadata flags and update `completion_metadata`, `verification_status` columns
- **Task Display**: Modify existing `shark task get` to parse and display completion metadata when `--completion-details` flag present
- **Search Integration**: Create new search functionality for file-based discovery (extends existing search infrastructure)

### Downstream Dependencies

Features that depend on E10-F02:
- **E10-F05**: Work Sessions & Resume Context (resume command includes completion metadata)
- **E10-F04**: Acceptance Criteria & Search (extends search to include completion metadata + criteria)

---

## Implementation Plan

### Database Migration

**Migration File**: `internal/db/migrations/011_add_completion_metadata.sql`

```sql
-- Add completion metadata columns to tasks table
ALTER TABLE tasks ADD COLUMN completion_metadata TEXT;  -- JSON blob
ALTER TABLE tasks ADD COLUMN verification_status TEXT
  CHECK (verification_status IN ('not_verified', 'verified', 'failed', 'manual_required'))
  DEFAULT 'not_verified';
ALTER TABLE tasks ADD COLUMN verification_notes TEXT;

-- Index for filtering by verification status
CREATE INDEX idx_tasks_verification_status ON tasks(verification_status);
```

**Backward Compatibility**:
- Existing tasks will have `completion_metadata = NULL`, `verification_status = 'not_verified'`
- Display logic handles null metadata gracefully ("No completion metadata available")
- No breaking changes to existing commands

---

### Code Organization

- **Repository**: Extend `internal/repository/task_repository.go` (existing file)
  - Add method: `UpdateCompletionMetadata(taskID int, metadata CompletionMetadata, verificationStatus string) error`
  - Add method: `SearchByFile(filename string, filters TaskFilters) ([]Task, error)`
- **Models**: `internal/models/completion_metadata.go` (new file)
  - Define `CompletionMetadata` struct matching JSON structure
  - Add JSON marshal/unmarshal methods
  - Add validation methods
- **Commands**:
  - Modify: `internal/cli/commands/task_complete.go` (add metadata flags)
  - Modify: `internal/cli/commands/task_get.go` (add --completion-details flag)
  - Create: `internal/cli/commands/search_file.go` (new file for `search --file`)
- **Tests**:
  - Extend: `internal/repository/task_repository_test.go` (test completion metadata methods)
  - Extend: `internal/cli/commands/task_complete_test.go` (test new flags)
  - Create: `internal/cli/commands/search_file_test.go` (test file search)

---

### Testing Strategy

1. **Unit Tests**:
   - Repository method `UpdateCompletionMetadata()` stores and retrieves JSON correctly
   - Repository method `SearchByFile()` queries JSON fields accurately
   - JSON marshal/unmarshal for `CompletionMetadata` struct
   - Validation logic for verification_status enum

2. **Integration Tests**:
   - Full CLI command `shark task complete` with all metadata flags
   - Full CLI command `shark task get --completion-details` displays parsed metadata
   - Full CLI command `shark search --file` returns correct results
   - Backward compatibility: viewing tasks without completion metadata

3. **Performance Tests**:
   - Load 10,000 tasks with completion metadata, measure file search time (<2 seconds)
   - Measure `--completion-details` parsing time (<200ms)

4. **Edge Cases**:
   - Task with no completion metadata (null handling)
   - Task with empty completion metadata (`{}` JSON)
   - File search with no matches
   - File search with 100+ matches
   - Partial filename matching (e.g., "useTheme" matches "useTheme.ts" and "useTheme.spec.ts")
   - Special characters in file paths (spaces, dashes, underscores)
   - Malformed JSON in completion_metadata (should fail gracefully)

---

## Open Questions

- **Q1**: Should file paths be validated to ensure they exist in the project?
  - **Recommendation**: No validation for Phase 1 (allows documenting deleted/moved files). Add validation as Phase 2 enhancement if needed.
- **Q2**: Should `shark search --file` support regex patterns?
  - **Recommendation**: No - partial string matching sufficient for Phase 1. Add regex in Phase 2 if users request it.
- **Q3**: Should completion metadata be editable after task completion?
  - **Recommendation**: No - immutable audit trail. Users must re-complete task if metadata incorrect. Use `verification_notes` for corrections.
- **Q4**: Should agent_execution_id be automatically inferred from system context?
  - **Recommendation**: Phase 2 enhancement - requires tracking active agent session. Manual `--agent-id` flag sufficient for Phase 1.

---

*Last Updated*: 2025-12-26
*Status*: Ready for Review
*Author*: BusinessAnalyst Agent
