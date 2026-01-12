# Requirements

**Epic**: [Workflow-Aware Task Command System](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for the workflow-aware task command system.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

**Priority Framework**: MoSCoW (Must Have, Should Have, Could Have, Won't Have)

---

## Functional Requirements

### Must Have Requirements

#### Category 1: Core Phase-Aware Commands

**REQ-F-001**: Implement `shark task claim` Command
- **Description**: New command that claims a task for an agent, transitioning from `ready_for_X` to `in_X` state
- **User Story**: As Atlas (AI orchestrator), I want to claim a task for a specific agent so that the task status reflects active work and prevents duplicate assignment
- **Acceptance Criteria**:
  - [ ] Command accepts task ID and optional `--agent=<type>` flag
  - [ ] Reads current task status (must be `ready_for_*` pattern)
  - [ ] Consults workflow config to find corresponding `in_*` status
  - [ ] Transitions status (e.g., `ready_for_development` → `in_development`)
  - [ ] Creates work session record with start timestamp and agent ID
  - [ ] Returns JSON with new status, next possible phases, assigned agent types
  - [ ] Fails gracefully if task not in `ready_for_*` state with clear error message
- **Related Journey**: Journey 1 (AI Orchestrator), Step 3; Journey 4 (Developer), Step 3

**REQ-F-002**: Implement `shark task finish` Command
- **Description**: New command that completes current phase and advances task to next workflow stage
- **User Story**: As Dev (developer), I want to mark my work complete so that the task moves to the next phase (code review) without me knowing the exact status name
- **Acceptance Criteria**:
  - [ ] Command accepts task ID and optional `--notes="..."` flag
  - [ ] Reads current task status (must be `in_*` pattern)
  - [ ] Consults workflow config `status_flow[current_status]` to determine valid next statuses
  - [ ] If multiple next states possible, selects `ready_for_*` pattern (standard forward flow)
  - [ ] Transitions status (e.g., `in_development` → `ready_for_code_review`)
  - [ ] Closes work session with end timestamp
  - [ ] Adds completion note if provided
  - [ ] Returns JSON with new status and responsible agent types for next phase
  - [ ] Works identically across all workflow phases (in_development, in_qa, in_approval, etc.)
- **Related Journey**: Journey 1, Step 7; Journey 3, Steps 1-5; Journey 4, Step 5

**REQ-F-003**: Implement `shark task reject` Command
- **Description**: New command that sends task backward in workflow for rework
- **User Story**: As Dev, I want to reject a task back to refinement when acceptance criteria are incomplete, with a clear reason
- **Acceptance Criteria**:
  - [ ] Command accepts task ID, `--reason="..."` (required), and optional `--to=<status>`
  - [ ] If `--to` provided, validates it exists in workflow config
  - [ ] If `--to` not provided, determines appropriate previous phase from workflow
  - [ ] Transitions status (e.g., `in_development` → `in_refinement`)
  - [ ] Creates note with rejection reason and who rejected
  - [ ] Closes current work session
  - [ ] Returns JSON with new status and responsible agent types
  - [ ] Validates target status is "earlier" in workflow to prevent forward rejection
- **Related Journey**: Journey 1, Alt Path B; Journey 2, Alt Path A; Journey 4, Alt Path A

**REQ-F-004**: Workflow Configuration Reader
- **Description**: Centralized module that reads and validates workflow configuration from `.sharkconfig.json`
- **User Story**: As Atlas, I need commands to adapt to any workflow configuration so I don't need code changes when workflow phases change
- **Acceptance Criteria**:
  - [ ] Reads `status_flow` section from `.sharkconfig.json`
  - [ ] Reads `status_metadata` section with phase and agent_types
  - [ ] Caches config in memory to avoid repeated file reads
  - [ ] Validates workflow on load (no orphaned states, all transitions valid)
  - [ ] Provides API: `GetValidTransitions(from_status string) []string`
  - [ ] Provides API: `GetPhase(status string) string`
  - [ ] Provides API: `GetAgentTypes(status string) []string`
  - [ ] Handles missing config gracefully (falls back to default workflow)
- **Related Journey**: All journeys (foundational)

**REQ-F-005**: Status Transition Validation
- **Description**: Validate all status transitions against workflow configuration before executing
- **User Story**: As a workflow administrator, I want invalid transitions to be rejected so my custom workflow is enforced
- **Acceptance Criteria**:
  - [ ] Before any status change, check `status_flow[current_status]` contains target status
  - [ ] Return clear error if transition invalid: "Cannot transition from X to Y. Valid next states: [A, B, C]"
  - [ ] Log all transition attempts (successful and failed) for audit
  - [ ] Provide override flag `--force` for admin emergency transitions (with warning)
  - [ ] Work consistently across claim, finish, reject, block, unblock commands
- **Related Journey**: Journey 5 (Workflow Setup), Step 4

#### Category 2: Enhanced Work Assignment

**REQ-F-006**: Workflow-Aware `shark task next` Command
- **Description**: Enhance existing `next` command to filter by workflow phase and agent type
- **User Story**: As Atlas, I want to query for tasks in a specific workflow phase for a specific agent type
- **Acceptance Criteria**:
  - [ ] Accepts `--agent=<type>` flag to filter by agent assignment
  - [ ] Queries tasks with status matching `ready_for_*` pattern
  - [ ] Filters results where `status_metadata[status].agent_types` contains requested agent type
  - [ ] Returns highest priority task matching criteria
  - [ ] Returns empty/null if no tasks available
  - [ ] JSON output includes task details and workflow phase info
  - [ ] Backward compatible (works without workflow config using current logic)
- **Related Journey**: Journey 1, Step 2; Journey 2, Step 2

**REQ-F-007**: Implement `shark feature next` Command
- **Description**: New command to get next feature with available work for an agent type
- **User Story**: As Sarah (PM), I want to find the next feature that has work ready for backend developers
- **Acceptance Criteria**:
  - [ ] Command accepts epic key and optional `--agent=<type>` flag
  - [ ] Queries features in epic ordered by execution_order and priority
  - [ ] For each feature, checks if any tasks match agent type and are in `ready_for_*` state
  - [ ] Returns first feature with available work
  - [ ] JSON output includes feature details and count of available tasks
  - [ ] If no features have work, returns clear message
- **Related Journey**: Journey 2, Step 2

#### Category 3: Command Consolidation

**REQ-F-008**: Consolidate Dependency Commands
- **Description**: Merge `blocks`, `blocked-by` into `deps --type=<type>` subcommand
- **User Story**: As a developer, I want one command to view all dependency relationships instead of remembering multiple commands
- **Acceptance Criteria**:
  - [ ] `shark task deps <task-id>` shows all relationships (default)
  - [ ] `shark task deps <task-id> --type=depends-on` shows dependencies
  - [ ] `shark task deps <task-id> --type=blocks` shows outgoing blockers
  - [ ] `shark task deps <task-id> --type=blocked-by` shows incoming dependencies
  - [ ] Output format matches existing dependency display
  - [ ] JSON output includes relationship types
  - [ ] Deprecated commands (`blocks`, `blocked-by`) show warning and redirect to `deps`
- **Related Journey**: N/A (UX improvement)

**REQ-F-009**: Consolidate Note Commands
- **Description**: Merge `note` (add) and `notes` (list) into single `notes` command with subcommands
- **User Story**: As a user, I want consistent command structure where listing and adding use the same base command
- **Acceptance Criteria**:
  - [ ] `shark task notes <task-id>` lists all notes (default behavior)
  - [ ] `shark task notes add <task-id> "message"` adds new note
  - [ ] `shark task notes list <task-id>` explicitly lists notes
  - [ ] Supports `--type=<type>` flag for filtering/categorizing
  - [ ] Deprecated standalone `note` command redirects to `notes add`
  - [ ] JSON output for both list and add operations
- **Related Journey**: Journey 4, Step 4

**REQ-F-010**: Merge Timeline into History
- **Description**: Remove separate `timeline` command, add as `--format=timeline` flag to `history`
- **User Story**: As a user, I want one command for viewing task history in different formats
- **Acceptance Criteria**:
  - [ ] `shark task history <task-id>` shows default history view
  - [ ] `shark task history <task-id> --format=timeline` shows timeline visualization
  - [ ] `shark task history <task-id> --format=json` shows JSON output
  - [ ] Timeline format matches existing `timeline` command output
  - [ ] Deprecated `timeline` command shows warning and redirects
- **Related Journey**: N/A (UX improvement)

#### Category 4: Command Categorization

**REQ-F-011**: Categorize Task Commands in Help Output
- **Description**: Organize `shark task --help` output into logical categories like main `shark --help`
- **User Story**: As a user, I want categorized commands so I can quickly find the command I need without scanning 25 alphabetical entries
- **Acceptance Criteria**:
  - [ ] Help output groups commands into categories:
    - Task Lifecycle: create, get, list, update, delete
    - Phase Management: claim, finish, reject, block/unblock
    - Work Assignment: next
    - Context & Documentation: resume, context, criteria, notes
    - Relationships & Dependencies: deps, link/unlink
    - Analytics & History: history, sessions
  - [ ] Each category has brief description
  - [ ] Categories visually separated (spacing or headers)
  - [ ] Alphabetical listing still available with `--all` flag
  - [ ] Matches main `shark --help` categorization style
- **Related Journey**: All journeys (discoverability)

### Should Have Requirements

**REQ-F-012**: Context-Aware Command Suggestions
- **Description**: When a command is used in wrong context, suggest correct command based on task status
- **User Story**: As a user, when I run `shark task claim` on a task already claimed, I want a helpful suggestion instead of just an error
- **Acceptance Criteria**:
  - [ ] If `claim` used on `in_*` status, suggest `finish` or `reject`
  - [ ] If `finish` used on `ready_for_*` status, suggest `claim` first
  - [ ] If `reject` used on `ready_for_*` status, suggest appropriate backward status
  - [ ] Suggestions include example commands with actual task ID
  - [ ] Suggestions reference workflow config to be accurate
  - [ ] Works in both human-readable and JSON output modes
- **Related Journey**: Improves all journeys (error recovery)

**REQ-F-013**: Work Session Tracking
- **Description**: Track when tasks are claimed and finished to calculate work duration
- **User Story**: As Sarah (PM), I want to see how long tasks spend in each phase for sprint planning
- **Acceptance Criteria**:
  - [ ] `claim` command creates session record with start timestamp and agent ID
  - [ ] `finish` command closes session with end timestamp
  - [ ] `reject` command closes session and marks as incomplete
  - [ ] `block` command pauses session (not closed)
  - [ ] `unblock` command resumes session
  - [ ] `shark task sessions <task-id>` shows all session history
  - [ ] Sessions stored in `task_sessions` table
  - [ ] JSON output includes duration calculations
- **Related Journey**: Journey 2, Steps 4-6

**REQ-F-014**: Workflow Visualization
- **Description**: Command to display current workflow configuration as diagram
- **User Story**: As Alex (workflow admin), I want to visualize my workflow configuration to verify transitions
- **Acceptance Criteria**:
  - [ ] `shark workflow list` shows workflow as text diagram
  - [ ] `shark workflow list --json` shows full config structure
  - [ ] Diagram shows all statuses and transitions
  - [ ] Highlights special statuses (_start_, _complete_)
  - [ ] Colors/markers for different phases
  - [ ] Shows agent types assigned to each status
- **Related Journey**: Journey 5, Step 2

### Could Have Requirements

**REQ-F-015**: Bulk Status Transitions
- **Description**: Apply claim/finish/reject to multiple tasks at once
- **User Story**: As Sarah, I want to approve multiple tasks at once after sprint review
- **Acceptance Criteria**:
  - [ ] Commands accept `--all` flag to operate on query results
  - [ ] `shark task finish --status=ready_for_approval --all` finishes all tasks in approval phase
  - [ ] Requires confirmation prompt (unless `--yes` flag)
  - [ ] Reports success/failure for each task
  - [ ] Stops on first error unless `--continue-on-error`
  - [ ] JSON output lists all results
- **Related Journey**: Journey 2, Step 8 (scaled up)

**REQ-F-016**: Agent Type Auto-Detection
- **Description**: Infer agent type from environment or user configuration
- **User Story**: As Dev, I don't want to specify `--agent=backend` every time if I'm always a backend developer
- **Acceptance Criteria**:
  - [ ] Check environment variable `SHARK_AGENT_TYPE`
  - [ ] Check user config file `~/.sharkconfig` for `default_agent_type`
  - [ ] Use detected agent type if not explicitly provided
  - [ ] Show which agent type is being used in verbose mode
  - [ ] Allow override with explicit `--agent` flag
- **Related Journey**: Journey 4 (reduces typing)

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: Command Execution Time
- **Description**: Phase-aware commands must execute within acceptable latency bounds
- **Measurement**: Measure time from command invocation to output display
- **Target**:
  - Claim/finish/reject: < 500ms for 90th percentile
  - Next query: < 1s for database with 10,000 tasks
  - Workflow config read: < 50ms (cached after first read)
- **Justification**: AI orchestrator polls every 30 seconds; slow commands block assignment loop

**REQ-NF-002**: Workflow Config Caching
- **Description**: Cache workflow configuration in memory to avoid repeated file reads
- **Measurement**: Count file system reads during command execution
- **Target**: Config read once per process, cached for lifetime
- **Implementation**: Singleton pattern with file modification time check

### Backward Compatibility

**REQ-NF-003**: Deprecated Command Support
- **Description**: Old commands must continue working with deprecation warnings
- **Implementation**:
  - Phase 1 (Release 1): Add new commands, keep old commands functional
  - Phase 2 (Release 2): Add deprecation warnings to old commands
  - Phase 3 (Release 3): Remove old commands, return error with migration guide
- **Deprecation Targets**:
  - `start` → `claim`
  - `complete` → `finish`
  - `approve` → `finish`
  - `next-status` → `finish`
  - `reopen` → `reject`
  - `blocks` → `deps --type=blocks`
  - `blocked-by` → `deps --type=blocked-by`
  - `timeline` → `history --format=timeline`

**REQ-NF-004**: Workflow Config Fallback
- **Description**: Commands must work without workflow config (default workflow)
- **Implementation**:
  - If `.sharkconfig.json` missing or `status_flow` not defined, use hardcoded default
  - Default workflow: `todo → in_progress → ready_for_review → completed`
  - Log warning when using default workflow
  - All tests run against both custom and default workflows

### Error Handling

**REQ-NF-005**: Clear Error Messages
- **Description**: All errors must provide actionable guidance
- **Examples**:
  - "Cannot claim task T-E07-F20-001: Task is already claimed by agent 'backend'. Use 'shark task finish' to complete or 'shark task reject' to send back."
  - "Invalid transition from 'in_development' to 'completed'. Valid next states: ready_for_code_review, ready_for_refinement, blocked. Check workflow config: .sharkconfig.json"
  - "Workflow config validation failed: Status 'in_qa' has no outgoing transitions. Every non-terminal status must have at least one valid next state."
- **Acceptance Criteria**:
  - [ ] Every error includes what went wrong
  - [ ] Every error suggests how to fix it
  - [ ] Errors reference relevant config files or commands
  - [ ] JSON errors include error codes for programmatic handling

**REQ-NF-006**: Transaction Safety
- **Description**: Status transitions must be atomic (all-or-nothing)
- **Implementation**:
  - Wrap status update + session management + note creation in database transaction
  - Rollback on any failure
  - Return error if transaction fails
  - Log all transaction attempts for debugging
- **Testing**: Simulate database failures, verify rollback

### Documentation

**REQ-NF-007**: Updated Command Reference
- **Description**: All documentation must reflect new command model
- **Deliverables**:
  - [ ] CLI_REFERENCE.md updated with new commands
  - [ ] CLAUDE.md updated with workflow-aware examples
  - [ ] Migration guide: OLD_TO_NEW_COMMANDS.md
  - [ ] Workflow configuration guide: WORKFLOW_CONFIG.md
  - [ ] Example workflows (kanban, scrum, SAFe)
- **Acceptance**: Documentation reviewed by all personas

**REQ-NF-008**: Inline Help Text
- **Description**: Every command must have comprehensive `--help` output
- **Acceptance Criteria**:
  - [ ] Description of command purpose
  - [ ] All flags documented with examples
  - [ ] Examples showing common usage
  - [ ] Related commands listed
  - [ ] Workflow-aware commands explain phase transitions
  - [ ] Help text generated from code (not duplicated)

### Testing

**REQ-NF-009**: Workflow Compatibility Test Suite
- **Description**: Automated tests verify commands work with various workflow configs
- **Coverage**:
  - [ ] Default workflow (3 states)
  - [ ] Simple custom workflow (5 states)
  - [ ] Complex workflow (10+ states with branches)
  - [ ] Minimal workflow (draft → completed)
  - [ ] Workflow with multiple paths (e.g., skip code review)
- **Acceptance**: 100% of test workflows pass all phase-aware commands

**REQ-NF-010**: Integration Tests for AI Orchestrator
- **Description**: Simulate full orchestrator workflow with task assignment and handoff
- **Scenarios**:
  - [ ] Task flows through complete workflow with different agents
  - [ ] Task rejected backward and re-processed
  - [ ] Task blocked and unblocked mid-workflow
  - [ ] Multiple tasks assigned concurrently
  - [ ] Workflow config changed mid-execution
- **Acceptance**: All scenarios complete successfully with workflow-aware commands

---

## Out of Scope (Won't Have)

See [Scope Boundaries](./scope.md) for detailed exclusions.

---

*See also*: [Success Metrics](./success-metrics.md), [User Journeys](./user-journeys.md)
