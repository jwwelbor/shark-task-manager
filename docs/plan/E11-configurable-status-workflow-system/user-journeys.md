# User Journeys

**Epic**: [Configurable Status Workflow System](./epic.md)

---

## Overview

This document maps the key user workflows enabled by the configurable status workflow system, focusing on multi-agent collaboration and complex development processes.

---

## Journey 1: Business Analyst Refines Draft Task

**Persona**: Business Analyst Agent

**Goal**: Find unrefined tasks, analyze requirements, produce specification, and hand off to developers

**Preconditions**:
- Workflow config defines `ready_for_refinement` and `in_refinement` statuses
- Tasks exist with status `draft` or `ready_for_refinement`

### Happy Path

1. **Query Tasks Needing Refinement**
   - Agent action: Runs `shark task next --agent=business-analyst`
   - System response: Queries tasks with status `ready_for_refinement` where `agent_types` includes "business-analyst"
   - Expected outcome: Returns highest-priority unrefined task (e.g., T-E11-F01-005)

2. **Claim Task Atomically**
   - Agent action: Runs `shark task set-status T-E11-F01-005 in_refinement`
   - System response: Validates transition (`ready_for_refinement → in_refinement` is allowed), updates status, records in task_history
   - Expected outcome: Task status changes, other agents querying `ready_for_refinement` no longer see this task

3. **Analyze Requirements**
   - Agent action: Reads task file, analyzes context, generates specification document
   - System response: N/A (file operations)
   - Expected outcome: Detailed requirements document created in task directory

4. **Hand Off to Development**
   - Agent action: Runs `shark task set-status T-E11-F01-005 ready_for_development --notes="Spec complete, see requirements.md"`
   - System response: Validates transition, updates status, records notes in task_history
   - Expected outcome: Developer agents querying `ready_for_development` now see this task

**Success Outcome**: Task has complete specification and is visible in developer agent queues

### Alternative Paths

**Alt Path A: Requirements Incomplete**
- **Trigger**: Agent discovers missing information during analysis
- **Branch Point**: After Step 3
- **Flow**:
  1. Agent runs `shark task set-status T-E11-F01-005 blocked --reason="Awaiting client input on auth method"`
  2. System validates transition, marks task as blocked
  3. Agent or human unblocks later: `shark task unblock T-E11-F01-005`
  4. Task returns to `ready_for_refinement` status
- **Outcome**: Task paused until dependency resolved

**Alt Path B: Task Already Claimed**
- **Trigger**: Another agent already moved task to `in_refinement`
- **Branch Point**: Between Step 1 and Step 2
- **Flow**:
  1. First query returns task T-E11-F01-005
  2. Before agent can claim it, another agent transitions it
  3. Agent's `set-status` command fails with "Task T-E11-F01-005 already in_refinement"
  4. Agent re-runs query, gets next available task
- **Outcome**: Race condition handled gracefully, agent gets different task

### Critical Decision Points

- **Decision at Step 4**: Agent must decide if specification is complete enough for development. Incomplete specs cause rework later.

---

## Journey 2: Developer Implements Feature

**Persona**: Developer Agent

**Goal**: Find fully-specified tasks, implement code, submit for review

**Preconditions**:
- Workflow config defines `ready_for_development`, `in_development`, `ready_for_review` statuses
- Business analyst has completed specifications

### Happy Path

1. **Query Ready Tasks**
   - Agent action: Runs `shark task list --status=ready_for_development --agent=developer --json`
   - System response: Returns JSON array of tasks with complete specifications
   - Expected outcome: Agent selects highest-priority task from list

2. **Start Implementation**
   - Agent action: Runs `shark task start T-E11-F01-005` (convenience command)
   - System response: Validates `ready_for_development → in_development` transition, updates status
   - Expected outcome: Task marked as "in progress", removed from other developers' queues

3. **Write Code**
   - Agent action: Implements feature following TDD workflow
   - System response: N/A (code operations)
   - Expected outcome: Tests pass, implementation complete

4. **Submit for Review**
   - Agent action: Runs `shark task complete T-E11-F01-005 --notes="Implemented OAuth flow, 95% test coverage"`
   - System response: Validates `in_development → ready_for_review`, updates status
   - Expected outcome: Tech lead agents querying `ready_for_review` see this task

**Success Outcome**: Code is implemented, tested, and awaiting review

### Alternative Paths

**Alt Path A: Specification Incomplete**
- **Trigger**: Developer discovers missing requirements during implementation
- **Branch Point**: After Step 2 (during coding)
- **Flow**:
  1. Developer runs `shark task set-status T-E11-F01-005 ready_for_refinement --notes="Missing: error handling for expired tokens"`
  2. System validates backward transition (`in_development → ready_for_refinement`), updates status
  3. Business analyst sees task back in refinement queue
  4. Analyst completes spec, moves back to `ready_for_development`
  5. Developer (same or different) re-implements
- **Outcome**: Spec improved, task re-queued for development

**Alt Path B: Task Blocked by Dependency**
- **Trigger**: Developer cannot proceed due to external dependency
- **Branch Point**: After Step 2
- **Flow**:
  1. Developer runs `shark task block T-E11-F01-005 --reason="Waiting for OAuth library v2.0 release"`
  2. System validates transition, marks blocked
  3. When dependency resolves, `shark task unblock T-E11-F01-005` returns task to `ready_for_development`
- **Outcome**: Task paused without losing implementation progress

### Critical Decision Points

- **Decision at Step 2**: If spec is incomplete, send back to refinement immediately (don't waste time on partial implementation)

---

## Journey 3: QA Finds Bugs and Rejects Implementation

**Persona**: QA Agent

**Goal**: Test implemented features, approve or reject based on quality

**Preconditions**:
- Workflow config defines `ready_for_qa`, `in_qa`, `ready_for_approval` statuses
- Code review has passed (task in `ready_for_qa`)

### Happy Path

1. **Query QA Queue**
   - Agent action: Runs `shark task next --agent=qa`
   - System response: Returns highest-priority task with status `ready_for_qa`
   - Expected outcome: Agent receives task T-E11-F01-005 for testing

2. **Start QA**
   - Agent action: Runs `shark task set-status T-E11-F01-005 in_qa`
   - System response: Validates transition, updates status
   - Expected outcome: Task marked as "being tested", removed from other QA agents' queues

3. **Execute Test Suite**
   - Agent action: Runs automated tests, manual test scenarios
   - System response: N/A (testing operations)
   - Expected outcome: Test results show 3 critical bugs

4. **Reject Implementation**
   - Agent action: Runs `shark task set-status T-E11-F01-005 in_development --notes="CRITICAL: Login fails on Safari, OAuth token refresh broken, mobile UI broken"`
   - System response: Validates backward transition (`in_qa → in_development`), updates status, records notes
   - Expected outcome: Task back in developer queue with detailed bug report

**Success Outcome**: Buggy implementation rejected, clear feedback provided, developers notified automatically

### Alternative Paths

**Alt Path A: Tests Pass (Happy Path)**
- **Trigger**: All tests pass
- **Branch Point**: After Step 3
- **Flow**:
  1. QA agent runs `shark task set-status T-E11-F01-005 ready_for_approval --notes="All tests passed, ready for product approval"`
  2. System validates transition, updates status
  3. Product manager agents see task in approval queue
- **Outcome**: Task advances to final approval

**Alt Path B: Architecture Issue Found**
- **Trigger**: QA discovers fundamental design flaw (not just bugs)
- **Branch Point**: After Step 3
- **Flow**:
  1. QA runs `shark task set-status T-E11-F01-005 ready_for_refinement --notes="OAuth implementation violates OWASP security guidelines, needs architectural rework"`
  2. System validates multi-hop backward transition (`in_qa → ready_for_refinement`)
  3. Business analyst re-analyzes, revises spec
  4. Task flows through dev → review → QA again
- **Outcome**: Architectural issue caught before production, task reworked properly

### Critical Decision Points

- **Decision at Step 4**: Determine severity (bugs → development, architecture issues → refinement)

---

## Journey 4: Tech Lead Reviews Code

**Persona**: Tech Lead Agent

**Goal**: Review code quality, approve or request changes

**Preconditions**:
- Workflow config defines `ready_for_review`, `in_review`, `ready_for_qa` statuses
- Developer has completed implementation

### Happy Path

1. **Query Review Queue**
   - Agent action: Runs `shark task list --status=ready_for_review --agent=tech-lead`
   - System response: Returns all tasks awaiting code review
   - Expected outcome: Agent prioritizes and selects task T-E11-F01-005

2. **Start Review**
   - Agent action: Runs `shark task set-status T-E11-F01-005 in_review`
   - System response: Validates transition, updates status
   - Expected outcome: Task marked as "under review"

3. **Analyze Code**
   - Agent action: Reviews code structure, test coverage, adherence to standards
   - System response: N/A (review operations)
   - Expected outcome: Review complete with findings

4. **Approve and Advance to QA**
   - Agent action: Runs `shark task set-status T-E11-F01-005 ready_for_qa --notes="Code review passed, 95% coverage, follows standards"`
   - System response: Validates transition, updates status
   - Expected outcome: Task visible in QA agent queues

**Success Outcome**: Code approved, task advances to testing phase

### Alternative Paths

**Alt Path A: Code Quality Issues**
- **Trigger**: Review finds bugs or style violations
- **Branch Point**: After Step 3
- **Flow**:
  1. Tech lead runs `shark task set-status T-E11-F01-005 in_development --notes="Fix: missing error handling, add validation for null inputs, improve variable naming"`
  2. Task returns to developer queue
  3. Developer fixes issues, resubmits
- **Outcome**: Code improved before QA

**Alt Path B: Architectural Concerns**
- **Trigger**: Review finds design pattern violations
- **Branch Point**: After Step 3
- **Flow**:
  1. Tech lead runs `shark task set-status T-E11-F01-005 ready_for_refinement --notes="OAuth implementation should use repository pattern, not direct DB calls"`
  2. Business analyst updates spec
  3. Developer re-implements with correct pattern
- **Outcome**: Architecture corrected early

### Critical Decision Points

- **Decision at Step 4**: Severity of issues determines rejection path (minor bugs → dev, architecture → refinement)

---

## Journey 5: Project Manager Customizes Workflow

**Persona**: Human Project Manager

**Goal**: Configure Shark workflow to match team's development process

**Preconditions**:
- Project initialized with Shark
- Team has existing workflow they want to replicate

### Happy Path

1. **Open Workflow Config**
   - User action: Opens `.sharkconfig.json` in editor
   - System response: N/A (file editing)
   - Expected outcome: Sees existing workflow config

2. **Define Custom Workflow**
   - User action: Edits `status_flow` and `status_metadata` sections to match team process
   - System response: N/A (file editing)
   - Expected outcome: Custom workflow defined (e.g., simple Kanban: backlog → in_progress → done)

3. **Validate Configuration**
   - User action: Runs `shark workflow validate`
   - System response: Parses config, checks for errors (unreachable statuses, circular refs, undefined references)
   - Expected outcome: Validation passes or shows specific errors to fix

4. **Start Using New Workflow**
   - User action: Creates new tasks with `shark task create`
   - System response: New tasks use configured workflow statuses
   - Expected outcome: New tasks follow team's custom workflow, existing tasks (if any) remain unchanged with default workflow

**Success Outcome**: Shark workflow now matches team's process for all new tasks

### Alternative Paths

**Alt Path A: Validation Fails**
- **Trigger**: Config has errors
- **Branch Point**: After Step 3
- **Flow**:
  1. Validation shows: "Status 'stuck' has no path to terminal statuses"
  2. User edits config to fix
  3. Re-runs validation
  4. Passes, proceeds to use workflow
- **Outcome**: Invalid config caught before tasks are created

**Alt Path B: Want to Change Workflow Later**
- **Trigger**: Team decides to modify workflow after tasks exist
- **Branch Point**: After Step 4 (weeks later)
- **Flow**:
  1. User edits `.sharkconfig.json` with new statuses/transitions
  2. Runs `shark workflow validate` to verify
  3. New tasks use updated workflow
  4. Existing tasks continue with their current statuses (backward compatible)
- **Outcome**: Workflow evolution supported without breaking existing tasks

### Critical Decision Points

- **Decision at Step 2**: Choose workflow carefully - while changeable, existing tasks won't auto-migrate to new workflow

---

## Journey 6: Emergency Hotfix with Force Flag

**Persona**: Human Developer (Tech Lead)

**Goal**: Deploy urgent fix bypassing normal workflow validation

**Preconditions**:
- Production issue requires immediate fix
- Normal workflow would delay deployment (e.g., draft → dev → review → QA → approval)

### Happy Path

1. **Create Hotfix Task**
   - User action: Runs `shark task create --epic=E11 --feature=F03 "Fix critical login bug" --status=draft`
   - System response: Creates task in draft status
   - Expected outcome: Task T-E11-F03-012 created

2. **Implement Fix**
   - User action: Writes minimal fix, tests locally
   - System response: N/A (coding)
   - Expected outcome: Fix ready to deploy

3. **Force to Completed**
   - User action: Runs `shark task set-status T-E11-F03-012 completed --force --notes="HOTFIX: Emergency bypass for production login issue"`
   - System response: Bypasses workflow validation, updates status, logs forced transition with warning
   - Expected outcome: Task marked completed, deployable immediately

**Success Outcome**: Emergency fix deployed without workflow delays, audit trail shows forced transition

### Alternative Paths

**Alt Path A: Forget Force Flag**
- **Trigger**: User tries to skip workflow without `--force`
- **Branch Point**: After Step 3
- **Flow**:
  1. User runs `shark task set-status T-E11-F03-012 completed`
  2. System rejects: "Invalid transition from draft to completed. Valid next: [ready_for_refinement, cancelled, on_hold]. Use --force to override."
  3. User adds `--force` flag
  4. Succeeds
- **Outcome**: Workflow enforcement prevents accidental invalid transitions

### Critical Decision Points

- **Decision at Step 3**: Use --force only for genuine emergencies (overuse indicates workflow doesn't match reality)

---

*See also*: [Requirements](./requirements.md)
