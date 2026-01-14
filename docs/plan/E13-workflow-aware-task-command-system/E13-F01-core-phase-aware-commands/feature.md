---
feature_key: E13-F01-core-phase-aware-commands
epic_key: E13
title: Core Phase-Aware Commands
description: Implement claim, finish, and reject commands that read workflow configuration. Replace start, complete, approve, next-status, and reopen with phase-aware alternatives.
---

# Core Phase-Aware Commands

**Feature Key**: E13-F01-core-phase-aware-commands

---

## Epic

- **Epic PRD**: [E13 Workflow-Aware Task Command System](../../epic.md)
- **Epic Architecture**: [System Architecture](../../architecture/system-architecture.md)
- **Command Specifications**: [Command Specs](../../architecture/command-specifications.md)
- **Workflow Config Reader**: [Workflow Integration](../../architecture/workflow-config-reader.md)

---

## Goal

### Problem

The current shark task commands (`start`, `complete`, `approve`, `reopen`, `next-status`) assume a hardcoded workflow (`todo → in_progress → ready_for_review → completed`) and fail to work properly with custom workflows that have arbitrary phases and agent-specific transitions. This creates several critical issues:

1. **Broken AI Orchestrator**: AI agents cannot assign tasks across custom workflow phases because commands hardcode target statuses
2. **Confusing Semantics**: Multiple overlapping commands (`complete`, `approve`, `next-status`) all do similar things with unclear distinctions
3. **No Workflow Awareness**: Commands don't consult `.sharkconfig.json` to determine valid transitions
4. **Team Adoption Blocker**: Teams with custom workflows (kanban, SAFe, multi-stage review) cannot use shark effectively

**Who Experiences This**:
- **AI Orchestrator Agent (Atlas)**: Cannot reliably assign tasks across custom workflow phases
- **Development Teams**: Forced to use hardcoded workflow or write custom scripts
- **Product Managers**: Cannot configure workflows to match their team's process

### Solution

Replace hardcoded workflow assumptions with **three phase-aware commands** that read workflow configuration dynamically:

1. **`shark task claim`**: Claims task for agent, transitioning `ready_for_X → in_X`
2. **`shark task finish`**: Completes current phase, advancing to next phase based on workflow config
3. **`shark task reject`**: Sends task backward in workflow for rework

These commands will:
- Read `status_flow` from `.sharkconfig.json` to determine valid transitions
- Use pattern matching (`ready_for_*`, `in_*`) to identify phase boundaries
- Work identically across any custom workflow configuration
- Provide clear, intent-based semantics (claim → work → finish)

The existing commands (`start`, `complete`, `approve`, `reopen`, `next-status`) will be deprecated with warnings but remain functional for backward compatibility.

### Impact

**For AI Agents**:
- 100% compatibility with any custom workflow without code changes
- Clear phase-based semantics that map to agent capabilities
- Automatic handoff to next agent type based on workflow metadata

**For Human Users**:
- Reduce from 25 to 18 commands (28% reduction in cognitive load)
- Eliminate confusion between `complete` vs `approve` vs `next-status`
- Better alignment with SDLC terminology used in planning sessions

**For Development Teams**:
- Configure any workflow in `.sharkconfig.json` without shark code changes
- Support kanban, scrum, SAFe, custom review processes
- Clear audit trail of phase transitions and work sessions

**Expected Outcomes** (Measurable):
- Zero workflow-assumption bugs in AI orchestrator after migration
- 90% of users migrate to new commands within 2 releases
- 78% faster command discovery through categorized help output
- Support for 100% of tested workflow configurations (5+ configs in test suite)

---

## User Personas

### Persona 1: Atlas (AI Orchestrator Agent)

**Profile**:
- **Role/Title**: Autonomous AI agent coordinating multi-agent task execution
- **Experience Level**: Fully automated, no human supervision
- **Key Characteristics**:
  - Queries tasks by workflow phase and agent type
  - Assigns tasks to specialized agents (backend, frontend, QA, etc.)
  - Tracks work sessions and detects stale/abandoned work
  - Needs deterministic, reliable command behavior

**Goals Related to This Feature**:
1. Assign next available task to appropriate agent type based on workflow phase
2. Track when agents claim work and when they finish
3. Handle task rejection and reassignment automatically
4. Support any custom workflow without code changes

**Pain Points This Feature Addresses**:
- Cannot use `start` command with custom workflows (hardcoded `in_progress`)
- No way to query tasks by workflow phase (only hardcoded `todo` status)
- Unclear when to use `complete` vs `approve` vs `next-status`
- Workflow config changes break orchestrator logic

**Success Looks Like**:
Atlas can read any `.sharkconfig.json`, query tasks in `ready_for_development` phase, claim them for a backend agent, and finish them to advance to `ready_for_code_review` without hardcoded status names. All phase transitions are validated against workflow config, and invalid transitions fail gracefully with clear error messages.

### Persona 2: Dev (Software Developer)

**Profile**:
- **Role/Title**: Full-stack developer at mid-size SaaS company
- **Experience Level**: 5+ years development, moderate CLI proficiency
- **Key Characteristics**:
  - Works on tasks assigned by PM or AI orchestrator
  - Uses shark CLI during development workflow
  - Needs fast, intuitive commands
  - Frustrated by overlapping/confusing commands

**Goals Related to This Feature**:
1. Claim a task, work on it, and mark it complete without memorizing status names
2. Reject tasks back to PM when requirements are unclear
3. Understand what happens next after finishing a task
4. Use same commands regardless of team's workflow config

**Pain Points This Feature Addresses**:
- Confusion between `start`, `complete`, `approve` - which one to use when?
- `start` command doesn't work when team uses custom `ready_for_development` status
- No clear way to reject task back to previous phase
- Has to check workflow config manually to know valid next statuses

**Success Looks Like**:
Dev runs `shark task claim T-E07-F20-001`, works on the task, runs `shark task finish T-E07-F20-001` with completion notes, and sees clear output showing the task advanced to code review phase. If requirements are unclear, dev runs `shark task reject T-E07-F20-001 --reason="Missing acceptance criteria"` and the task returns to BA for refinement.

### Persona 3: Sarah (Product Manager / Scrum Master)

**Profile**:
- **Role/Title**: Product Manager leading 8-person development team
- **Experience Level**: 3 years PM, limited technical background
- **Key Characteristics**:
  - Configures team workflow to match sprint process
  - Reviews tasks in approval phase
  - Needs clear visibility into task progress
  - Wants workflow to enforce team process

**Goals Related to This Feature**:
1. Configure custom workflow phases (refinement → development → review → QA → approval)
2. Ensure tasks follow workflow rules (no skipping phases)
3. Approve completed tasks efficiently
4. Reject tasks back to appropriate phase when needed

**Pain Points This Feature Addresses**:
- Team's workflow has 7 phases but shark commands assume 3
- `approve` command hardcoded to set status `completed`, but team needs `in_approval` → `completed` transition
- Cannot enforce workflow rules - developers can bypass review phase
- No way to reject task to specific earlier phase

**Success Looks Like**:
Sarah configures `.sharkconfig.json` with team's 7-phase workflow. Shark commands automatically adapt to the custom workflow. When reviewing tasks in `ready_for_approval`, Sarah uses `shark task finish` to move them to `completed`, or `shark task reject --reason="Missing test coverage" --to=in_qa` to send them back to QA. All transitions are validated against the configured workflow.

---

## User Stories

### Must-Have Stories

**Story 1**: As Atlas (AI orchestrator), I want to claim a task for a specific agent type so that the task status reflects active work and prevents duplicate assignment.

**Acceptance Criteria**:
- [ ] `shark task claim <task-key> --agent=<type>` transitions task from `ready_for_X` to `in_X`
- [ ] Command reads workflow config to determine target `in_X` status
- [ ] Work session is created with start timestamp and agent ID
- [ ] Task history records the transition with agent who claimed it
- [ ] JSON output includes new status and next phase information
- [ ] Error if task not in `ready_for_*` status: "Cannot claim task in 'in_development' status. Use 'shark task finish' or 'shark task reject'."
- [ ] Warning if agent type doesn't match expected types from workflow metadata

**Story 2**: As Dev (developer), I want to mark my work complete so that the task moves to the next phase without me knowing the exact status name.

**Acceptance Criteria**:
- [ ] `shark task finish <task-key>` advances task to next workflow phase
- [ ] Command reads workflow config `status_flow[current_status]` to determine valid next statuses
- [ ] Selects first `ready_for_*` status as default forward transition
- [ ] If multiple `ready_for_*` options exist, prompts for selection (interactive mode)
- [ ] In `--json` mode, automatically selects first `ready_for_*` status
- [ ] Work session is closed with end timestamp
- [ ] Optional `--notes="..."` flag adds completion notes to task history
- [ ] Output shows new status and which agent types can claim next phase
- [ ] Works from any `in_*` status (development, code_review, QA, approval, etc.)

**Story 3**: As Dev, I want to reject a task back to refinement when acceptance criteria are incomplete, with a clear reason.

**Acceptance Criteria**:
- [ ] `shark task reject <task-key> --reason="..."` sends task to earlier workflow phase
- [ ] `--reason` flag is required (command fails without it)
- [ ] Optional `--to=<status>` specifies exact target status (validated against workflow)
- [ ] If `--to` not provided, auto-determines appropriate backward status (refinement or previous phase)
- [ ] Command validates target status is "earlier" in workflow (backward transition)
- [ ] Rejection reason is recorded in task history and task notes
- [ ] Work session is closed with `rejected` outcome
- [ ] Error if no valid backward transitions exist
- [ ] Output shows new status and responsible agent types for that phase

**Story 4**: As Atlas, I want all commands to validate transitions against workflow config so that invalid phase transitions are prevented.

**Acceptance Criteria**:
- [ ] All commands (claim, finish, reject) consult workflow service before executing transitions
- [ ] `workflow.Service.ValidateTransition(from, to, commandType)` is called for every transition
- [ ] Invalid transitions return clear errors: "Cannot transition from 'in_development' to 'completed'. Valid next states: ready_for_code_review, ready_for_refinement, blocked"
- [ ] Error messages reference `.sharkconfig.json` for troubleshooting
- [ ] `--force` flag allows admin to override validation (with warning logged)
- [ ] All validation errors include suggestion for correct command or target status

**Story 5**: As Dev, I want commands to work without workflow config so that I can use shark in projects that haven't configured custom workflows.

**Acceptance Criteria**:
- [ ] If `.sharkconfig.json` missing, commands use default workflow
- [ ] Default workflow: `todo → in_progress → ready_for_review → completed`
- [ ] Warning logged when default workflow is used: "Using default workflow (no .sharkconfig.json found)"
- [ ] All commands work identically with default or custom workflows
- [ ] Test suite includes tests for both default and custom workflow scenarios

---

### Should-Have Stories

**Story 6**: As Sarah (PM), when I run a command in wrong context, I want a helpful suggestion instead of just an error.

**Acceptance Criteria**:
- [ ] If `claim` used on task in `in_*` status, suggests: "Task already claimed. Use 'shark task finish' to complete or 'shark task reject' to send back."
- [ ] If `finish` used on task in `ready_for_*` status, suggests: "Use 'shark task claim' first to start work."
- [ ] Suggestions include actual task ID in example command
- [ ] Works in both human-readable and JSON output modes

**Story 7**: As Atlas, I want work sessions tracked automatically so I can calculate phase duration for analytics.

**Acceptance Criteria**:
- [ ] `claim` command creates session record in `task_sessions` table
- [ ] `finish` command closes session with end timestamp and outcome='completed'
- [ ] `reject` command closes session with outcome='rejected'
- [ ] Session includes task ID, agent ID, start time, end time, outcome
- [ ] `shark task sessions <task-key>` displays all session history
- [ ] JSON output includes duration calculations

---

### Edge Case & Error Stories

**Error Story 1**: As Dev, when I try to claim a task that's already claimed, I want to see who claimed it and when so I can coordinate.

**Acceptance Criteria**:
- [ ] Error message: "Cannot claim task T-E07-F20-001. Task is already in 'in_development' status (claimed by backend at 2026-01-11 09:00)"
- [ ] Suggests: "Use 'shark task finish' to complete work or 'shark task reject' to send back"
- [ ] If `--force` flag used, allows re-claiming (admin override)

**Error Story 2**: As Dev, when workflow has no valid forward transition, I want clear guidance on what to do.

**Acceptance Criteria**:
- [ ] If `finish` called on terminal status: "Cannot finish task in 'completed' status. Task is already in terminal state."
- [ ] If workflow config broken (no outgoing transitions): "Workflow error - 'in_qa' has no outgoing transitions. This is a configuration error. Fix .sharkconfig.json or contact admin."
- [ ] If multiple forward paths exist without `ready_for_*`: Interactive prompt to select next status

**Error Story 3**: As Atlas, when I try to reject a task forward, I want the command to fail with explanation.

**Acceptance Criteria**:
- [ ] Error: "Cannot reject from 'in_development' to 'ready_for_qa'. Rejection must move to earlier workflow phase. Current phase: development. Target phase: qa (later than current)."
- [ ] Lists valid backward transitions: "Valid backward transitions: in_refinement, ready_for_refinement"

---

## Requirements

### Functional Requirements

**Category: Core Phase-Aware Commands**

1. **REQ-F-001**: Implement `shark task claim` Command
   - **Description**: New command that claims a task for an agent, transitioning from `ready_for_X` to `in_X` state
   - **User Story**: Links to Story 1, Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Command accepts task ID and optional `--agent=<type>` flag
     - [ ] Reads current task status (must be `ready_for_*` pattern)
     - [ ] Consults workflow config to find corresponding `in_*` status
     - [ ] Transitions status atomically in database transaction
     - [ ] Creates work session record with start timestamp and agent ID
     - [ ] Returns JSON with new status, next possible phases, assigned agent types
     - [ ] Fails gracefully if task not in `ready_for_*` state with clear error message
   - **Reference**: [Command Specification](../../architecture/command-specifications.md#command-shark-task-claim)

2. **REQ-F-002**: Implement `shark task finish` Command
   - **Description**: New command that completes current phase and advances task to next workflow stage
   - **User Story**: Links to Story 2, Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Command accepts task ID and optional `--notes="..."` flag
     - [ ] Reads current task status (must be `in_*` pattern)
     - [ ] Consults workflow config `status_flow[current_status]` to determine valid next statuses
     - [ ] If multiple next states possible, selects `ready_for_*` pattern (standard forward flow)
     - [ ] Closes work session with end timestamp
     - [ ] Adds completion note if provided
     - [ ] Returns JSON with new status and responsible agent types for next phase
     - [ ] Works identically across all workflow phases
   - **Reference**: [Command Specification](../../architecture/command-specifications.md#command-shark-task-finish)

3. **REQ-F-003**: Implement `shark task reject` Command
   - **Description**: New command that sends task backward in workflow for rework
   - **User Story**: Links to Story 3, Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Command accepts task ID, `--reason="..."` (required), and optional `--to=<status>`
     - [ ] If `--to` provided, validates it exists in workflow config
     - [ ] If `--to` not provided, determines appropriate previous phase from workflow
     - [ ] Transitions status to earlier phase
     - [ ] Creates note with rejection reason and who rejected
     - [ ] Closes current work session
     - [ ] Returns JSON with new status and responsible agent types
     - [ ] Validates target status is "earlier" in workflow to prevent forward rejection
   - **Reference**: [Command Specification](../../architecture/command-specifications.md#command-shark-task-reject)

4. **REQ-F-004**: Workflow Configuration Integration
   - **Description**: All commands must read and validate against `.sharkconfig.json` workflow configuration
   - **User Story**: Links to Story 4, Story 5
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Uses `workflow.Service` singleton with file modification time caching
     - [ ] Reads `status_flow` section for valid transitions
     - [ ] Reads `status_metadata` section for phase and agent_types
     - [ ] Caches config in memory to avoid repeated file reads
     - [ ] Handles missing config gracefully (falls back to default workflow)
     - [ ] Validates workflow on load (no orphaned states, all transitions valid)
   - **Reference**: [Workflow Config Reader](../../architecture/workflow-config-reader.md)

5. **REQ-F-005**: Status Transition Validation
   - **Description**: Validate all status transitions against workflow configuration before executing
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Before any status change, check `status_flow[current_status]` contains target status
     - [ ] Return clear error if transition invalid: "Cannot transition from X to Y. Valid next states: [A, B, C]"
     - [ ] Log all transition attempts (successful and failed) for audit
     - [ ] Provide override flag `--force` for admin emergency transitions (with warning)
     - [ ] Work consistently across claim, finish, reject commands
   - **Reference**: [Transition Validation](../../architecture/transition-validation.md)

**Category: Command Deprecation**

6. **REQ-F-006**: Deprecate Old Commands with Warnings
   - **Description**: Add deprecation warnings to old commands, maintain functionality
   - **User Story**: N/A (technical requirement)
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `task start` shows warning: "DEPRECATED: Use 'shark task claim' instead. This command will be removed in v2.0."
     - [ ] `task complete` shows warning: "DEPRECATED: Use 'shark task finish' instead."
     - [ ] `task approve` shows warning: "DEPRECATED: Use 'shark task finish' instead."
     - [ ] `task next-status` shows warning: "DEPRECATED: Use 'shark task finish' instead."
     - [ ] `task reopen` shows warning: "DEPRECATED: Use 'shark task reject --reason=...' instead."
     - [ ] All old commands remain functional (no breaking changes)
     - [ ] Warnings shown in stderr, not stdout (don't break scripts)

**Category: Work Session Tracking**

7. **REQ-F-007**: Work Session Management
   - **Description**: Track when tasks are claimed and finished to calculate work duration
   - **User Story**: Links to Story 7
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `claim` command creates session record with start timestamp and agent ID
     - [ ] `finish` command closes session with end timestamp and outcome='completed'
     - [ ] `reject` command closes session and marks as outcome='rejected'
     - [ ] Sessions stored in `task_sessions` table (or `work_sessions` if already exists)
     - [ ] JSON output includes duration calculations
     - [ ] `shark task sessions <task-id>` shows all session history

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Command Execution Time
   - **Description**: Phase-aware commands must execute within acceptable latency bounds
   - **Measurement**: Measure time from command invocation to output display
   - **Target**:
     - Claim/finish/reject: < 500ms for 90th percentile
     - Workflow config read: < 50ms (cached after first read)
     - Workflow config cache hit: < 1ms
   - **Justification**: AI orchestrator polls every 30 seconds; slow commands block assignment loop

2. **REQ-NF-002**: Workflow Config Caching
   - **Description**: Cache workflow configuration in memory to avoid repeated file reads
   - **Measurement**: Count file system reads during command execution
   - **Target**: Config read once per process, cached for lifetime (with file mod time check)
   - **Implementation**: Singleton pattern with file modification time check

**Backward Compatibility**

3. **REQ-NF-003**: Deprecated Command Support
   - **Description**: Old commands must continue working with deprecation warnings
   - **Implementation**:
     - Phase 1 (Release 1): Add new commands, keep old commands functional
     - Phase 2 (Release 2): Add deprecation warnings to old commands
     - Phase 3 (Release 3+): Remove old commands, return error with migration guide
   - **Deprecation Targets**: start, complete, approve, next-status, reopen

4. **REQ-NF-004**: Workflow Config Fallback
   - **Description**: Commands must work without workflow config (default workflow)
   - **Implementation**:
     - If `.sharkconfig.json` missing or `status_flow` not defined, use hardcoded default
     - Default workflow: `todo → in_progress → ready_for_review → completed`
     - Log warning when using default workflow
     - All tests run against both custom and default workflows

**Error Handling**

5. **REQ-NF-005**: Clear Error Messages
   - **Description**: All errors must provide actionable guidance
   - **Examples**:
     - "Cannot claim task T-E07-F20-001: Task is already claimed by agent 'backend'. Use 'shark task finish' to complete or 'shark task reject' to send back."
     - "Invalid transition from 'in_development' to 'completed'. Valid next states: ready_for_code_review, ready_for_refinement, blocked. Check workflow config: .sharkconfig.json"
   - **Acceptance Criteria**:
     - [ ] Every error includes what went wrong
     - [ ] Every error suggests how to fix it
     - [ ] Errors reference relevant config files or commands
     - [ ] JSON errors include error codes for programmatic handling

6. **REQ-NF-006**: Transaction Safety
   - **Description**: Status transitions must be atomic (all-or-nothing)
   - **Implementation**:
     - Wrap status update + session management + note creation in database transaction
     - Rollback on any failure
     - Return error if transaction fails
     - Log all transaction attempts for debugging
   - **Testing**: Simulate database failures, verify rollback

**Documentation**

7. **REQ-NF-007**: Updated Command Reference
   - **Description**: All documentation must reflect new command model
   - **Deliverables**:
     - [ ] CLI_REFERENCE.md updated with new commands
     - [ ] CLAUDE.md updated with workflow-aware examples
     - [ ] Migration guide: docs/migration/OLD_TO_NEW_COMMANDS.md
     - [ ] Workflow configuration guide: docs/WORKFLOW_CONFIG.md
   - **Acceptance**: Documentation reviewed by all personas

8. **REQ-NF-008**: Inline Help Text
   - **Description**: Every command must have comprehensive `--help` output
   - **Acceptance Criteria**:
     - [ ] Description of command purpose
     - [ ] All flags documented with examples
     - [ ] Examples showing common usage
     - [ ] Related commands listed
     - [ ] Workflow-aware commands explain phase transitions

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Standard Workflow - Task Flows Through All Phases**
- **Given** a project with custom 7-phase workflow configured in `.sharkconfig.json`
- **And** a task in `ready_for_development` status
- **When** developer runs `shark task claim T-E07-F20-001 --agent=backend`
- **Then** task status changes to `in_development`
- **And** work session is created with start timestamp
- **When** developer runs `shark task finish T-E07-F20-001 --notes="API implemented"`
- **Then** task status changes to `ready_for_code_review` (next phase from workflow)
- **And** work session is closed with completion timestamp
- **And** completion notes are recorded in task history
- **When** tech lead runs `shark task claim T-E07-F20-001 --agent=tech-lead`
- **Then** task status changes to `in_code_review`
- **And** new work session is created for tech lead

**Scenario 2: Rejection - Task Sent Back for Rework**
- **Given** a task in `in_development` status claimed by backend agent
- **When** developer runs `shark task reject T-E07-F20-001 --reason="Acceptance criteria incomplete"`
- **Then** task status changes to `in_refinement` (earlier phase)
- **And** rejection reason is recorded in task notes
- **And** work session is closed with outcome='rejected'
- **And** output shows which agent types (business-analyst) can claim the task

**Scenario 3: Default Workflow - No Config File**
- **Given** a project with no `.sharkconfig.json` file
- **When** developer runs `shark task claim T-E01-F01-001 --agent=dev`
- **Then** warning is logged: "Using default workflow (no .sharkconfig.json found)"
- **And** task status changes to `in_progress` (default workflow)
- **When** developer runs `shark task finish T-E01-F01-001`
- **Then** task status changes to `ready_for_review` (default workflow next phase)

**Scenario 4: Error Handling - Invalid Transition Blocked**
- **Given** a task in `in_development` status
- **When** developer runs `shark task finish` and workflow has no `ready_for_*` transitions (config error)
- **Then** error is returned: "Workflow error - 'in_development' has no outgoing transitions. This is a configuration error. Fix .sharkconfig.json or contact admin."
- **And** task status is unchanged
- **And** transaction is rolled back

**Scenario 5: Force Override - Admin Emergency Transition**
- **Given** a task in `completed` status (terminal)
- **When** admin runs `shark task claim T-E07-F20-001 --agent=backend --force`
- **Then** warning is logged: "WARNING: Force flag used to override workflow validation"
- **And** task status changes to `in_development` despite being invalid transition
- **And** transition is logged in task history with force=true flag

---

## Out of Scope

### Explicitly Excluded

1. **Analytics Commands (`shark-analytics` CLI)**
   - **Why**: Separate CLI binary for analytics/reporting, not part of task lifecycle commands
   - **Future**: Analytics commands will be moved to dedicated `shark-analytics` CLI in E13-F04
   - **Workaround**: Use existing `shark task history` and `shark task sessions` for basic analytics

2. **Workflow Designer UI**
   - **Why**: Complexity too high, teams can edit `.sharkconfig.json` directly
   - **Future**: May add interactive workflow editor in future epic
   - **Workaround**: Copy example workflow configs from `docs/examples/workflows/`

3. **Automatic Command Migration**
   - **Why**: Risk of breaking existing scripts, better to provide clear deprecation warnings
   - **Future**: Will remove old commands in v2.0 after deprecation period
   - **Workaround**: Users manually update scripts using migration guide

4. **Bulk Task Operations**
   - **Why**: Would require confirmation prompts and complex error handling
   - **Future**: May add `--all` flag to claim/finish/reject in E13-F05
   - **Workaround**: Use shell loops: `for task in $(shark task list --json | jq -r '.[].key'); do shark task finish $task; done`

5. **Interactive Phase Selection in JSON Mode**
   - **Why**: JSON mode is for automation, must be deterministic
   - **Future**: N/A - by design
   - **Workaround**: Use `--to=<status>` flag to explicitly specify target status

---

## Success Metrics

### Primary Metrics

1. **Workflow Compatibility Rate**
   - **What**: Percentage of tested workflow configurations that work without modification
   - **Target**: 100% of 5+ test workflow configs
   - **Timeline**: Before F01 completion
   - **Measurement**: Automated test suite with varied workflow configs

2. **Command Migration Adoption**
   - **What**: Percentage of users using new commands vs deprecated commands
   - **Target**: 90% of command invocations use new commands within 2 releases
   - **Timeline**: 3 months after deprecation warnings added
   - **Measurement**: Command usage telemetry (if added) or survey

3. **AI Orchestrator Success Rate**
   - **What**: Percentage of task assignments that succeed without workflow errors
   - **Target**: 100% success rate (zero workflow-assumption bugs)
   - **Timeline**: After integration testing
   - **Measurement**: Orchestrator logs, error rate monitoring

---

### Secondary Metrics

- **Command Execution Time**: 90th percentile < 500ms (claim, finish, reject)
- **Workflow Config Cache Hit Rate**: > 99% (only reload when config modified)
- **Help Command Discoverability**: 78% faster command discovery through categorization (user testing)

---

## Dependencies & Integrations

### Dependencies

- **Workflow Service (`internal/workflow/service.go`)**: Must enhance with phase-aware methods (REQ-F-004)
- **Repository Layer (`internal/repository/task_repository.go`)**: Uses existing `UpdateStatusForced` method
- **Work Session Tracking (`internal/repository/work_session_repository.go`)**: Existing session tracking (create/close sessions)
- **Workflow Configuration (`.sharkconfig.json`)**: Must have valid `status_flow` and `status_metadata` sections

### Integration Requirements

- **Task History**: All transitions logged to `task_history` table with agent, timestamp, notes
- **Work Sessions**: Sessions tracked in `work_sessions` or `task_sessions` table
- **Cascade Triggers**: Feature/epic status updates triggered by task status changes (existing functionality)

---

## Test Plan

### Unit Tests

**Test Files to Create**:
- `internal/cli/commands/task_claim_test.go`
- `internal/cli/commands/task_finish_test.go`
- `internal/cli/commands/task_reject_test.go`
- `internal/workflow/service_phase_aware_test.go`

**Test Coverage**:
1. **Command Argument Parsing**: Valid/invalid task keys, flag combinations
2. **Workflow Service Methods**: `GetNextPhaseStatus`, `GetPreviousPhaseStatus`, `GetClaimStatus`
3. **Status Pattern Matching**: `ready_for_*` → `in_*` transformations
4. **Transition Validation**: Valid/invalid transitions against workflow config
5. **Error Messages**: All error paths return clear, actionable messages
6. **JSON Output**: Correct structure, includes all required fields

**Mocking Strategy** (per CLAUDE.md):
- CLI tests use `MockTaskRepository`, `MockWorkflowService`, `MockSessionRepository`
- Repository tests use real database with cleanup before each test
- No CLI tests use real database

### Integration Tests

**Test Workflow Configurations**:
- `testdata/workflow-default.json` - Default 3-state workflow
- `testdata/workflow-simple.json` - Simple 5-state workflow
- `testdata/workflow-complex.json` - Enterprise 10-state workflow with branches
- `testdata/workflow-kanban.json` - Kanban board workflow
- `testdata/workflow-safe.json` - SAFe-style multi-stage workflow

**Integration Test Scenarios**:
1. **Complete Task Lifecycle**: Create → claim → finish → claim (next phase) → finish → approve
2. **Rejection Flow**: Claim → reject back to refinement → claim → finish
3. **Multiple Phases**: Task flows through 7+ workflow phases without errors
4. **Workflow Change**: Modify `.sharkconfig.json`, verify commands reload config
5. **Default Fallback**: Remove config file, verify default workflow works

**Database Tests**:
- Transaction rollback on errors (simulate DB failures)
- Concurrent claims (two agents claim same task)
- Session tracking accuracy (start/end timestamps, durations)

### Workflow Compatibility Tests

**Test Coverage**:
- [ ] Default workflow (3 states)
- [ ] Simple custom workflow (5 states)
- [ ] Complex workflow (10+ states with branches)
- [ ] Minimal workflow (draft → completed)
- [ ] Workflow with multiple forward paths (e.g., skip code review)
- [ ] Workflow with bidirectional transitions (dev ↔ refinement)

**Acceptance**: 100% of test workflows pass all phase-aware commands without modification

---

## Files to Modify

### New Files to Create

1. **`internal/cli/commands/task_claim.go`** (~150 lines)
   - Implement claim command with workflow integration
   - Agent auto-detection logic
   - Work session creation

2. **`internal/cli/commands/task_finish.go`** (~200 lines)
   - Implement finish command (reuse logic from task_next_status.go)
   - Interactive transition selection for multiple valid next states
   - Work session closure

3. **`internal/cli/commands/task_reject.go`** (~180 lines)
   - Implement reject command with reason validation
   - Backward transition logic
   - Rejection note creation

4. **`internal/workflow/phase_helpers.go`** (~150 lines)
   - `GetNextPhaseStatus`, `GetPreviousPhaseStatus`, `GetClaimStatus`
   - Phase pattern matching (`ready_for_*`, `in_*`)
   - Phase order comparison

### Files to Modify

1. **`internal/cli/commands/task.go`**
   - Add command registrations for claim, finish, reject
   - Extract shared helpers (workflow loading, session management)

2. **`internal/cli/commands/task_start.go`** (deprecation)
   - Add deprecation warning in command description
   - Add warning to stderr on execution

3. **`internal/cli/commands/task_complete.go`** (deprecation)
   - Add deprecation warning

4. **`internal/cli/commands/task_approve.go`** (deprecation)
   - Add deprecation warning

5. **`internal/cli/commands/task_reopen.go`** (deprecation)
   - Add deprecation warning

6. **`internal/cli/commands/task_next_status.go`** (deprecation)
   - Add deprecation warning
   - Extract interactive transition selection logic for reuse in finish

7. **`internal/workflow/service.go`**
   - Add phase-aware helper methods
   - Enhance caching with file modification time check

8. **`docs/CLI_REFERENCE.md`**
   - Document new commands with examples
   - Add migration examples

9. **`CLAUDE.md`**
   - Update workflow-aware command examples
   - Update task creation/management patterns

### Commands to Delete (Future - Phase 3)

- `task start` → removed after deprecation period
- `task complete` → removed after deprecation period
- `task approve` → removed after deprecation period
- `task next-status` → removed after deprecation period
- `task reopen` → removed after deprecation period

---

## Implementation Notes

### Code Reuse Opportunities

1. **Workflow Config Loading** (duplicated in 5 commands):
   - Extract to `loadWorkflowForCommand()` helper
   - Eliminates ~50 lines of duplicate code

2. **Work Session Management**:
   - Extract to `createWorkSession()` and `endWorkSession()` helpers
   - Consistent session handling across claim/finish/reject

3. **Interactive Transition Selection** (from task_next_status.go):
   - Reuse `promptForSelection()` and `printTransitions()` in finish command
   - Already implements workflow-driven suggestions

### Transaction Pattern

All commands use same atomic transaction pattern:
```go
tx, err := db.BeginTx(ctx, nil)
defer tx.Rollback()

// 1. Validate preconditions (status, workflow)
// 2. Update task status
// 3. Record history
// 4. Update/create session

tx.Commit()
```

### Workflow Integration Pattern

All commands follow same workflow pattern:
```go
workflow := workflow.NewService(projectRoot)
task := taskRepo.GetByKey(ctx, taskKey)
targetStatus := workflow.GetNextPhaseStatus(task.Status) // or GetClaimStatus, GetPreviousPhaseStatus
workflow.ValidateTransition(task.Status, targetStatus, "finish")
taskRepo.UpdateStatusForced(ctx, task.ID, targetStatus, ...)
```

---

## Related Documentation

**Epic Level**:
- [Epic PRD](../../epic.md)
- [Requirements](../../requirements.md) - REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-004, REQ-F-005
- [User Journeys](../../user-journeys.md) - Journey 1 (AI Orchestrator), Journey 4 (Developer)

**Architecture**:
- [System Architecture](../../architecture/system-architecture.md)
- [Command Specifications](../../architecture/command-specifications.md)
- [Workflow Config Reader](../../architecture/workflow-config-reader.md)
- [Transition Validation](../../architecture/transition-validation.md)

**Research**:
- [Current Implementation Analysis](../../research/current-implementation-analysis.md)

---

*Last Updated*: 2026-01-11
