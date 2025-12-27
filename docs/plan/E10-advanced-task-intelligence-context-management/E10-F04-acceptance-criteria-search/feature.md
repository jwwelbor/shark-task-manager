# Feature PRD: E10-F04 Acceptance Criteria & Search

**Feature Key**: E10-F04
**Epic**: [E10: Advanced Task Intelligence & Context Management](../epic.md)
**Status**: Draft
**Priority**: Should Have (Phase 2)
**Execution Order**: 4

---

## Goal

### Problem

Development teams lack a systematic way to track acceptance criteria completion at the database level, forcing manual checklist management in markdown files that can't be queried, aggregated, or used for automated progress reporting. Additionally, discovering relevant task information requires navigating multiple files and tables - there's no unified search capability that spans task descriptions, notes, completion metadata, and file modifications.

**Real Example from E13**: During T-E13-F05-002 (Flash Prevention Script), there were 7 acceptance criteria tracked only in markdown. Tech leads reviewing the task had no way to query "How many criteria are met?" or "Which criteria failed?" without manually reading the task file. Similarly, when investigating which tasks modified `index.html`, developers had to manually search through task files or git history - no way to query "show all tasks that touched this file."

### Solution

Implement two complementary capabilities:

1. **Structured Acceptance Criteria Tracking**: Create a `task_criteria` table that stores acceptance criteria as database records with status tracking (pending, in_progress, complete, failed, na). Enable:
   - Importing criteria from markdown task files automatically
   - Checking off criteria as work progresses
   - Querying criteria status and aggregating progress at task/feature/epic levels
   - Verification notes for each criterion

2. **Unified Full-Text Search**: Implement `shark search` command that searches across:
   - Task titles and descriptions
   - Task notes (from E10-F01)
   - Completion metadata (files modified/created)
   - Acceptance criteria text
   - With filters for file, agent, epic, feature, status, and note type

### Impact

**For AI Agents**:
- **Objective progress tracking**: Check off criteria as they're met, verify completion systematically (eliminate guesswork)
- **Quick discovery**: Search "localStorage persistence" and find all related tasks, notes, and implementations (5x faster than manual search)
- **File impact analysis**: Query "which tasks modified useTheme.ts" to understand change history

**For Tech Leads**:
- **Instant verification**: See "7/7 criteria met" at a glance instead of reading entire task files
- **Pattern discovery**: Search decision notes for "singleton pattern" to enforce consistency (70% faster code review)
- **Regression prevention**: Find all tasks that modified a file before making changes

**For Product Managers**:
- **Progress visibility**: Aggregate criteria across features: "28/35 criteria complete for E13-F05 = 80% done"
- **Data-driven reporting**: Query completion rates, not manual estimation
- **Blocker identification**: Search blocker notes to identify patterns requiring intervention

**For Developers**:
- **Learning**: Search solutions to find proven patterns ("How did we solve Safari flash issues?")
- **Impact analysis**: Find all tasks that touched critical files before refactoring
- **Context gathering**: Single search across all task metadata instead of multiple queries

---

## User Personas

See [personas.md](../personas.md) for detailed user personas:
- **Persona 1**: AI Development Agent (Claude Code)
- **Persona 2**: Human Developer (Technical Lead)
- **Persona 3**: Product Manager

---

## User Stories

### Must-Have Stories (Phase 2)

**Story 1**: As an AI development agent, I want to import acceptance criteria from task markdown into the database so that I can track completion systematically without manual transcription.

**Acceptance Criteria**:
- [ ] CLI command `shark task criteria import <task-key>` parses markdown file acceptance criteria sections
- [ ] Criteria extracted from markdown checklist format: `- [ ] criterion text` or `- [x] criterion text`
- [ ] Each criterion inserted as separate record in task_criteria table with status 'pending' or 'complete'
- [ ] Supports standard markdown checkbox syntax and numbered lists
- [ ] Reports number of criteria imported: "Imported 7 acceptance criteria for T-E13-F05-002"

---

**Story 2**: As a tech lead, I want to see criteria completion status (7/7 met) so that I can verify tasks are complete without reading entire task files.

**Acceptance Criteria**:
- [ ] CLI command `shark task criteria <task-key>` displays criteria summary
- [ ] Output shows total, complete, pending, failed, in_progress, and na counts
- [ ] Each criterion displayed with status icon: ✓ (complete), ✗ (failed), ○ (pending), ◐ (in_progress), − (na)
- [ ] Percentage calculation included: "85% complete (6/7 criteria)"
- [ ] Criteria ordered by status (failed first, then pending, in_progress, complete)

---

**Story 3**: As an AI agent, I want to mark individual acceptance criteria as complete so that I track progress as I implement each requirement.

**Acceptance Criteria**:
- [ ] CLI command `shark task criteria check <task-key> <criterion-id>` marks criterion complete
- [ ] Status updates to 'complete', verified_at set to current timestamp
- [ ] Optional `--note` parameter adds verification notes
- [ ] Displays updated criteria summary after check
- [ ] Cannot check already-completed criteria (idempotent operation)

---

**Story 4**: As a tech lead, I want to mark criteria as failed with a reason so that I can track what needs remediation.

**Acceptance Criteria**:
- [ ] CLI command `shark task criteria fail <task-key> <criterion-id> --note "<reason>"` marks criterion failed
- [ ] Status updates to 'failed', verification_notes stores reason
- [ ] `--note` parameter is required for failed criteria
- [ ] Failed criteria highlighted in red in output
- [ ] Can list all tasks with failed criteria: `shark task list --failed-criteria`

---

**Story 5**: As a product manager, I want to aggregate acceptance criteria across all tasks in a feature so that I can report objective completion percentages to stakeholders.

**Acceptance Criteria**:
- [ ] CLI command `shark feature criteria <feature-key>` aggregates across all feature tasks
- [ ] Output shows total criteria, complete count, percentage
- [ ] Breakdown by status: pending, in_progress, complete, failed, na
- [ ] Optional `--by-task` flag shows per-task breakdown
- [ ] Example output: "Feature E13-F05: 28/35 criteria complete (80%)"

---

**Story 6**: As a developer, I want to search for "localStorage persistence" across all task data (titles, notes, criteria, metadata) so that I can find all related implementations.

**Acceptance Criteria**:
- [ ] CLI command `shark search "<query>"` searches across tasks.title, tasks.description, task_notes.content, task_criteria.criterion, completion_metadata
- [ ] Case-insensitive search
- [ ] Results ranked by relevance: title match > note match > criterion match > metadata match
- [ ] Results show task key, title, match type (title/note/criterion/metadata), and snippet
- [ ] Optional filters: `--status`, `--epic`, `--feature`, `--agent`

---

**Story 7**: As a developer, I want to find all tasks that modified a specific file so that I can understand its change history before making modifications.

**Acceptance Criteria**:
- [ ] CLI command `shark search --file "<filename>"` searches completion_metadata JSON for file matches
- [ ] Supports exact filename match and partial path match
- [ ] Searches both files_created and files_modified in completion metadata
- [ ] Results show task key, title, status, completion date, and file operation (created/modified)
- [ ] Optional filters: `--epic`, `--feature`

---

**Story 8**: As a tech lead, I want to search decision notes only for "composable pattern" so that I can find architectural decisions without unrelated notes.

**Acceptance Criteria**:
- [ ] `shark search "<query>" --note-type decision` limits search to decision notes
- [ ] Multiple types supported: `--note-type decision,solution`
- [ ] Results show task key, note type, timestamp, and matching content
- [ ] Ordered by relevance then date descending

---

## Requirements

See [requirements.md](../requirements.md) for complete functional and non-functional requirements:
- **REQ-F-013**: Criteria Storage and Management
- **REQ-F-014**: Criteria Import from Markdown
- **REQ-F-015**: Full-Text Search Across Task Data
- **REQ-F-016**: Search by File Modified and Note Type
- **REQ-F-017**: Feature-Level Criteria Aggregation

---

## Database Schema

### New Table: task_criteria

```sql
CREATE TABLE task_criteria (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    criterion TEXT NOT NULL,
    status TEXT CHECK (status IN (
        'pending',      -- Not yet started
        'in_progress',  -- Being worked on
        'complete',     -- Met and verified
        'failed',       -- Did not meet requirement
        'na'            -- Not applicable (requirement changed)
    )) DEFAULT 'pending',
    verified_at TIMESTAMP,
    verification_notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_criteria_task_id ON task_criteria(task_id);
CREATE INDEX idx_task_criteria_status ON task_criteria(status);
CREATE INDEX idx_task_criteria_task_status ON task_criteria(task_id, status);
```

### Full-Text Search Virtual Table

```sql
CREATE VIRTUAL TABLE task_search_fts USING fts5(
    task_key UNINDEXED,
    title,
    description,
    note_content,
    criterion_text,
    metadata_text,
    tokenize='porter unicode61'
);
```

See implementation plan section for complete migration script with triggers.

---

## CLI Commands Summary

### Criteria Management
- `shark task criteria import <task-key>` - Import criteria from markdown
- `shark task criteria <task-key>` - View criteria status and progress
- `shark task criteria check <task-key> <id>` - Mark criterion complete
- `shark task criteria fail <task-key> <id> --note "<reason>"` - Mark criterion failed
- `shark task criteria add <task-key> "<text>"` - Add criterion manually

### Criteria Aggregation
- `shark feature criteria <feature-key>` - Aggregate across feature tasks
- `shark epic criteria <epic-key>` - Aggregate across epic tasks

### Search
- `shark search "<query>"` - Search all task data
- `shark search --file "<filename>"` - Find tasks that modified file
- `shark search "<query>" --note-type <type>` - Search specific note types
- `shark search "<query>" --epic <epic> --status <status>` - Filtered search

---

## User Journeys

See [user-journeys.md](../user-journeys.md) for detailed user journeys:
- **Journey 1**: AI Agent Tracks Criteria During Implementation
- **Journey 2**: Tech Lead Verifies Criteria Before Approval
- **Journey 3**: Product Manager Reports Feature Progress
- **Journey 4**: Developer Discovers Related Implementations
- **Journey 5**: Developer Investigates File Change History

---

## Success Metrics

### Primary Metrics

1. **Criteria Adoption Rate**: 70% of completed tasks have imported acceptance criteria
2. **Search Usage Frequency**: 5+ searches per developer per week
3. **Criteria Verification Rate**: 90% of criteria marked complete before task completion

### Secondary Metrics

- **Feature Progress Reporting**: 80% of features use `shark feature criteria` for tracking
- **File Search Usage**: 3+ file searches per developer per week
- **Search Result Relevance**: 85% of searches return at least 1 relevant result
- **Criteria Completion Before Approval**: 95% of approved tasks have 100% criteria complete

---

## Dependencies & Integrations

### Dependencies
- **E10-F01: Task Activity & Notes System** (search includes notes)
- **E10-F02: Completion Metadata** (search includes files_created/files_modified)
- Existing `tasks` table

### Downstream Dependencies
- **E10-F05: Work Sessions & Resume Context** (may include criteria progress)
- Future analytics features (criteria completion rates)

---

## Implementation Plan

### Database Migration

**File**: `internal/db/migrations/014_add_task_criteria_search.sql`

Complete migration includes:
1. `task_criteria` table creation
2. `task_search_fts` FTS5 virtual table
3. Triggers to maintain FTS index from tasks, task_notes, task_criteria

### Code Organization

**New Files**:
- `internal/repository/task_criteria_repository.go`
- `internal/repository/search_repository.go`
- `internal/models/task_criteria.go`
- `internal/models/search_result.go`
- `internal/cli/commands/task_criteria.go`
- `internal/cli/commands/feature_criteria.go`
- `internal/cli/commands/search.go`
- `internal/taskfile/criteria_parser.go`

**Test Files**:
- `internal/repository/task_criteria_repository_test.go`
- `internal/repository/search_repository_test.go`
- `internal/cli/commands/task_criteria_test.go`
- `internal/taskfile/criteria_parser_test.go`

### Testing Strategy

1. **Unit Tests**: Repository methods, criteria parser, search ranking
2. **Integration Tests**: Full CLI commands with real database, FTS triggers
3. **Performance Tests**: Search with 10K tasks, aggregation with 20 tasks
4. **Edge Cases**: Zero criteria, 100+ criteria, malformed markdown, special characters

---

## Out of Scope

1. **Criteria Editing After Import** - Re-import or use `criteria add` instead
2. **Automatic Criteria Detection from Code** - Manual markdown authoring required
3. **Criteria Dependencies/Ordering** - Use markdown numbering to indicate order
4. **Search Result Ranking Configuration** - Fixed relevance ranking
5. **Search Highlighting in Snippets** - Match type indicated but not highlighted
6. **Criteria Templates/Presets** - Use external markdown templates

---

*Last Updated*: 2025-12-26
*Status*: Ready for Review
*Author*: BusinessAnalyst Agent
