# User Journeys

**Epic**: [Workflow-Aware Task Command System](./epic.md)

---

## Overview

This document maps the key user workflows enabled by workflow-aware task commands. All journeys demonstrate how commands adapt to custom workflow configurations defined in `.sharkconfig.json`.

**Workflow Example** (used throughout journeys):
```
draft → ready_for_refinement → in_refinement → ready_for_development →
in_development → ready_for_code_review → in_code_review → ready_for_qa →
in_qa → ready_for_approval → in_approval → completed
```

---

## Journey 1: AI Orchestrator Autonomous Task Assignment

**Persona**: Atlas (AI Orchestrator Agent)

**Goal**: Continuously assign and track tasks across all workflow phases without hardcoded status assumptions

**Preconditions**:
- Orchestrator running as systemd service
- Workflow config loaded from `.sharkconfig.json`
- Multiple AI agents available (Claude, GPT-4)
- Tasks exist in various `ready_for_X` states

### Happy Path

1. **Poll for Ready Tasks**
   - System action: Orchestrator wakes every 30 seconds
   - System executes: `shark task list --status=ready_for_development --agent=backend --json`
   - Expected outcome: JSON array of tasks with metadata including workflow phase

2. **Select Highest Priority Task**
   - System action: Apply priority algorithm (explicit priority + dependencies + age)
   - System queries: `shark task next --agent=backend --epic=E07 --json`
   - Expected outcome: Single task object `T-E07-F20-001` returned

3. **Claim Task for Agent**
   - System action: Reserve task to prevent duplicate assignment
   - System executes: `shark task claim T-E07-F20-001 --agent=backend --json`
   - Expected outcome:
     - Status transitions: `ready_for_development` → `in_development`
     - Work session created with start timestamp
     - Agent ID recorded
     - JSON response confirms transition

4. **Get Full Task Context**
   - System action: Retrieve all task details for agent execution
   - System executes: `shark task resume T-E07-F20-001 --json`
   - Expected outcome: JSON with acceptance criteria, related docs, dependencies, notes

5. **Spawn AI Agent**
   - System action: Create Claude agent instance with task context
   - Agent executes: Read files, write code, run tests (outside shark)
   - Expected outcome: Agent works on implementation

6. **Monitor Progress**
   - System action: Parse agent notes for progress indicators
   - Agent periodically executes: `shark task note T-E07-F20-001 "PROGRESS: 50% - API endpoints complete"`
   - Expected outcome: Orchestrator tracks health score, detects stale agents

7. **Agent Finishes Work**
   - System action: Mark phase complete and advance to next
   - System executes: `shark task finish T-E07-F20-001 --notes="Implementation complete" --json`
   - Expected outcome:
     - System reads workflow config `status_flow["in_development"]` → finds `ready_for_code_review`
     - Status transitions: `in_development` → `ready_for_code_review`
     - Work session closed with end timestamp
     - JSON response indicates next phase and responsible agent types

8. **Next Cycle: Assign to Code Reviewer**
   - System action: Orchestrator detects task in new phase
   - System executes: `shark task next --agent=tech-lead --json`
   - System executes: `shark task claim T-E07-F20-001 --agent=tech-lead`
   - Expected outcome: Task moves `ready_for_code_review` → `in_code_review`, assigned to tech-lead agent

**Success Outcome**: Task flows through entire workflow (refinement → development → code review → QA → approval → completed) with orchestrator adapting to each phase based on workflow config. Zero hardcoded status values in orchestrator code.

### Alternative Paths

**Alt Path A: Agent Encounters Blocker**
- **Trigger**: Agent cannot proceed due to missing dependency
- **Branch Point**: After Step 5 (agent working)
- **Flow**:
  1. Agent executes: `shark task block T-E07-F20-001 --reason="Waiting for DB schema migration approval"`
  2. Status transitions: `in_development` → `blocked`
  3. Orchestrator stops assigning this task
  4. When blocker resolved: `shark task unblock T-E07-F20-001`
  5. Status returns: `blocked` → `in_development` (or `ready_for_development` if abandoned)
- **Outcome**: Task paused until external dependency resolved

**Alt Path B: Task Not Ready for Development**
- **Trigger**: Agent discovers acceptance criteria are incomplete
- **Branch Point**: After Step 5 (agent starts work)
- **Flow**:
  1. Agent executes: `shark task reject T-E07-F20-001 --reason="Acceptance criteria missing" --to=in_refinement`
  2. Status transitions: `in_development` → `in_refinement`
  3. Note added explaining rejection reason
  4. BA agent claims and refines task
  5. BA finishes: Status → `ready_for_development`
  6. Developer agent claims again
- **Outcome**: Task returns to earlier phase for rework

### Critical Decision Points

- **Decision at Step 7**: System must read workflow config to determine next phase. If config has multiple valid transitions (e.g., `in_development` → [`ready_for_code_review`, `ready_for_qa`]), system chooses based on priority or user-defined rules.

---

## Journey 2: PM Sprint Planning and Work Assignment

**Persona**: Sarah (Product Manager)

**Goal**: Assign work to team members for upcoming sprint based on workflow phase and readiness

**Preconditions**:
- Epic E07 with multiple features created
- Tasks in various workflow states
- Team has defined agent types (backend, frontend, qa)

### Happy Path

1. **Monday Morning: Plan Sprint Work**
   - User action: Sarah opens terminal, wants to see what's ready for development
   - User executes: `shark task list --status=ready_for_development --json`
   - Expected outcome: List of 15 tasks ready for developers to claim

2. **Assign Backend Work**
   - User action: Wants to assign backend tasks for the sprint
   - User executes: `shark feature next E07 --agent=backend`
   - Expected outcome: Returns F20 as highest priority feature with backend work
   - User executes: `shark task next E07 F20 --agent=backend`
   - Expected outcome: Returns T-E07-F20-001 (highest priority task in F20 ready for backend)

3. **Notify Developer**
   - User action: Sarah assigns task to developer
   - User message: "Hey Dev, please work on T-E07-F20-001"
   - Expected outcome: Dev acknowledges

4. **Developer Claims Task**
   - Developer action: Dev starts work
   - Developer executes: `shark task claim T-E07-F20-001 --agent=backend`
   - Expected outcome:
     - Status: `ready_for_development` → `in_development`
     - Session logged with Dev's agent ID and start time

5. **Mid-Sprint: Check Progress**
   - User action: Sarah wants sprint status
   - User executes: `shark status --json`
   - Expected outcome: Dashboard showing tasks by phase (5 in_development, 3 ready_for_qa, etc.)

6. **Developer Finishes Implementation**
   - Developer action: Code complete, ready for review
   - Developer executes: `shark task finish T-E07-F20-001 --notes="API implementation complete"`
   - Expected outcome:
     - Status: `in_development` → `ready_for_code_review`
     - Sarah sees task moved to review phase

7. **Code Review Phase**
   - Tech Lead action: Reviews code
   - Tech Lead executes: `shark task claim T-E07-F20-001 --agent=tech-lead`
   - Status: `ready_for_code_review` → `in_code_review`
   - Tech Lead approves: `shark task finish T-E07-F20-001 --notes="LGTM, approved"`
   - Status: `in_code_review` → `ready_for_qa`

8. **Final Approval**
   - QA action: Tests pass, task ready for Sarah's approval
   - QA executes: `shark task finish T-E07-F20-002 --notes="All tests passed"`
   - Status: `in_qa` → `ready_for_approval`
   - Sarah executes: `shark task claim T-E07-F20-002 --agent=product-manager`
   - Status: `ready_for_approval` → `in_approval`
   - Sarah approves: `shark task finish T-E07-F20-002`
   - Status: `in_approval` → `completed`

**Success Outcome**: Sarah successfully plans sprint, assigns work, tracks progress across all workflow phases, and approves final deliverables using consistent `claim → finish` pattern at each phase.

### Alternative Paths

**Alt Path A: Code Review Rejects Implementation**
- **Trigger**: Tech lead finds issues during code review
- **Branch Point**: After Step 7 (code review)
- **Flow**:
  1. Tech Lead executes: `shark task reject T-E07-F20-001 --reason="Missing error handling for edge cases"`
  2. Status: `in_code_review` → `in_development`
  3. Dev gets notification (via orchestrator or manual check)
  4. Dev re-claims and fixes issues
  5. Dev finishes again: Status → `ready_for_code_review`
- **Outcome**: Task cycles back for rework with clear rejection reason

---

## Journey 3: Multi-Agent Task Handoff Through Workflow

**Persona**: Multiple agents (BA → Developer → Tech Lead → QA → PM)

**Goal**: Task flows through complete SDLC workflow with multiple agents, each using same command pattern

**Preconditions**:
- Custom workflow configured with 5 phases
- Task created in `draft` status
- Different agent types assigned to each phase

### Happy Path

1. **Phase 1: Business Analysis (Refinement)**
   - BA claims: `shark task claim T-E07-F20-003 --agent=business-analyst`
   - Status: `ready_for_refinement` → `in_refinement`
   - BA writes acceptance criteria, adds context
   - BA finishes: `shark task finish T-E07-F20-003 --notes="Acceptance criteria defined"`
   - Status: `in_refinement` → `ready_for_development`

2. **Phase 2: Development**
   - Developer claims: `shark task claim T-E07-F20-003 --agent=developer`
   - Status: `ready_for_development` → `in_development`
   - Developer implements feature
   - Developer finishes: `shark task finish T-E07-F20-003`
   - Status: `in_development` → `ready_for_code_review`

3. **Phase 3: Code Review**
   - Tech Lead claims: `shark task claim T-E07-F20-003 --agent=tech-lead`
   - Status: `ready_for_code_review` → `in_code_review`
   - Tech Lead reviews code
   - Tech Lead finishes: `shark task finish T-E07-F20-003`
   - Status: `in_code_review` → `ready_for_qa`

4. **Phase 4: QA Testing**
   - QA claims: `shark task claim T-E07-F20-003 --agent=qa`
   - Status: `ready_for_qa` → `in_qa`
   - QA runs test suite
   - QA finishes: `shark task finish T-E07-F20-003 --notes="All tests passed"`
   - Status: `in_qa` → `ready_for_approval`

5. **Phase 5: Final Approval**
   - PM claims: `shark task claim T-E07-F20-003 --agent=product-manager`
   - Status: `ready_for_approval` → `in_approval`
   - PM verifies deliverable
   - PM finishes: `shark task finish T-E07-F20-003`
   - Status: `in_approval` → `completed`

**Success Outcome**: Task passes through 5 workflow phases with 5 different agents, each using identical `claim → finish` pattern. Commands adapt to workflow config at each step. No agent needs to know full workflow, only their phase.

---

## Journey 4: Developer Task Execution (Claim to Complete)

**Persona**: Dev (Software Developer)

**Goal**: Claim task, implement feature, hand off to next phase

**Preconditions**:
- Task T-E07-F20-004 in `ready_for_development` status
- Dev has backend agent role
- Acceptance criteria defined in task

### Happy Path

1. **Find Next Task**
   - User action: Dev wants to work on next backend task
   - User executes: `shark task next --agent=backend`
   - Expected outcome: Returns T-E07-F20-004

2. **Get Task Details**
   - User action: Dev reviews requirements
   - User executes: `shark task get T-E07-F20-004`
   - Expected outcome: Shows acceptance criteria, description, related docs

3. **Claim Task**
   - User action: Dev starts work session
   - User executes: `shark task claim T-E07-F20-004 --agent=backend`
   - Expected outcome:
     - Status: `ready_for_development` → `in_development`
     - Session start time logged
     - Output confirms transition and shows next phase will be `code_review`

4. **Work on Implementation**
   - User action: Dev writes code, runs tests
   - User periodically: `shark task note T-E07-F20-004 "Implemented authentication endpoint"`
   - Expected outcome: Progress tracked in notes

5. **Finish Implementation**
   - User action: Dev completes coding work
   - User executes: `shark task finish T-E07-F20-004 --notes="Auth API complete, ready for review"`
   - Expected outcome:
     - Status: `in_development` → `ready_for_code_review`
     - Session end time logged
     - Output shows task moved to review phase
     - Indicates tech-lead agent type can claim next

**Success Outcome**: Dev claims task, completes work, hands off to next phase with clear status transition. Session tracked from claim to finish.

### Alternative Paths

**Alt Path A: Task Not Ready**
- **Trigger**: Dev starts work and realizes specs are incomplete
- **Branch Point**: After Step 3 (claimed task)
- **Flow**:
  1. Dev executes: `shark task reject T-E07-F20-004 --reason="Missing DB schema definition" --to=in_refinement`
  2. Status: `in_development` → `in_refinement`
  3. BA agent gets notified (via orchestrator or manual check)
  4. BA adds missing schema definition
  5. BA finishes: Status → `ready_for_development`
  6. Dev (or another dev) claims again
- **Outcome**: Task sent back for refinement with clear reason

---

## Journey 5: Custom Workflow Setup by Administrator

**Persona**: Alex (Workflow Administrator)

**Goal**: Configure custom workflow and verify commands work with new phases

**Preconditions**:
- Clean shark installation
- Admin has workflow design documented

### Happy Path

1. **Define Workflow in Config**
   - User action: Edit `.sharkconfig.json`
   - User adds custom `status_flow` and `status_metadata` sections
   - Expected outcome: Workflow saved with 7 phases

2. **Validate Workflow**
   - User action: Check workflow is valid
   - User executes: `shark workflow validate`
   - Expected outcome: Confirmation that all transitions are valid, no orphaned states

3. **Create Test Task**
   - User action: Create task to test workflow
   - User executes: `shark task create E07 F20 "Test workflow task"`
   - Expected outcome: Task created in `draft` status (workflow start state)

4. **Test Phase Transitions**
   - User action: Manually walk through workflow
   - User executes: `shark task claim TEST-001`
   - Expected outcome: Transitions from `draft` → first `in_` state based on workflow
   - User executes: `shark task finish TEST-001`
   - Expected outcome: Transitions to next `ready_for_` state

5. **Verify Agent Type Filtering**
   - User action: Query tasks by agent type
   - User executes: `shark task list --status=ready_for_development --agent=backend --json`
   - Expected outcome: Returns only tasks in phases assigned to backend agents per `status_metadata.agent_types`

**Success Outcome**: Admin creates custom 7-phase workflow, commands adapt automatically without code changes. Phase-aware commands work correctly with new workflow configuration.

---

*See also*: [Requirements](./requirements.md), [Personas](./personas.md)
