# User Journeys

**Epic**: [Advanced Task Intelligence & Context Management](./epic.md)

---

## Overview

This document maps the key workflows enabled by advanced task intelligence features, based on real usage patterns from E13 dark mode implementation.

---

## Journey 1: AI Agent Task Execution with Context Capture

**Persona**: AI Development Agent (Claude Code)

**Goal**: Execute a task while capturing decisions, blockers, and solutions for future reference

**Preconditions**:
- Task exists in database (T-E13-F05-002: Implement Flash Prevention Script)
- Agent has started task via `shark task start T-E13-F05-002`

### Happy Path

1. **Begin Implementation**
   - User action: Agent reads task description and acceptance criteria
   - System response: Task status updated to `in_progress`, work session created
   - Expected outcome: Agent has clear requirements and tracking has begun

2. **Record Implementation Decision**
   - User action: `shark task note add T-E13-F05-002 --type decision "Used IIFE pattern instead of module to avoid async loading delay"`
   - System response: Note stored with timestamp, type, and agent ID
   - Expected outcome: Decision is captured for future reference

3. **Document External Reference**
   - User action: `shark task note add T-E13-F05-002 --type reference "Similar pattern used in shadcn-vue: https://github.com/..."`
   - System response: Reference note stored
   - Expected outcome: Useful patterns are linked for learning

4. **Encounter and Solve Problem**
   - User action: `shark task note add T-E13-F05-002 --type solution "Safari flash fix: Moved script BEFORE viewport meta tag"`
   - System response: Solution note stored
   - Expected outcome: Troubleshooting steps are documented

5. **Complete Task with Metadata**
   - User action: `shark task complete T-E13-F05-002 --files-created "index.html" --summary "Implemented IIFE flash prevention script" --verified`
   - System response: Task status → `ready_for_review`, completion metadata stored
   - Expected outcome: Rich completion record exists with all context

**Success Outcome**: Task is complete with full audit trail of decisions, solutions, and completion details

### Alternative Paths

**Alt Path A: Blocked by Dependency**
- **Trigger**: Agent discovers task depends on incomplete task
- **Branch Point**: After Step 1
- **Flow**:
  1. `shark task note add T-E13-F05-004 --type blocker "Blocked by missing useTheme composable - see T-E13-F05-003"`
  2. `shark task link T-E13-F05-004 --depends-on T-E13-F05-003`
  3. `shark task block T-E13-F05-004 --reason "Waiting for useTheme composable"`
- **Outcome**: Task marked blocked with relationship tracked, agent can work on other tasks

**Alt Path B: Pause for External Input**
- **Trigger**: Agent needs design decision from human
- **Branch Point**: After Step 3
- **Flow**:
  1. `shark task note add T-E13-F05-004 --type question "Should theme toggle be in header or settings page?"`
  2. `shark task context set T-E13-F05-004 --current-step "Created base component, awaiting placement decision"`
  3. `shark task session pause T-E13-F05-004 --note "Waiting for design decision"`
- **Outcome**: Task paused with clear question and resume point

---

## Journey 2: Resume Paused Task with Full Context

**Persona**: AI Development Agent (Claude Code)

**Goal**: Resume a previously paused task without re-analyzing entire codebase

**Preconditions**:
- Task T-E13-F05-004 was paused with notes and context
- New conversation/session started (context lost)

### Happy Path

1. **Identify Next Task**
   - User action: `shark task next --agent frontend`
   - System response: Returns T-E13-F05-004 with status `in_progress` (paused)
   - Expected outcome: Agent knows which task to resume

2. **Retrieve Full Context**
   - User action: `shark task resume T-E13-F05-004`
   - System response: Returns task details, all notes (chronologically), context data, acceptance criteria status, work sessions
   - Expected outcome: Agent sees full history including:
     - What was completed: "Created base ThemeToggle.vue component, added icon switching"
     - Current step: "Implementing dropdown menu for 3 theme options"
     - Open questions: "Should theme toggle be in header or settings page?"
     - Related tasks: T-E13-F05-003 (completed), T-E13-F05-001 (completed)

3. **Review Timeline**
   - User action: `shark task timeline T-E13-F05-004`
   - System response: Chronological view of status changes, notes, sessions
   - Expected outcome: Agent sees full activity history including decisions made

4. **Continue Work**
   - User action: Agent implements next step based on context
   - System response: Work session resumed automatically
   - Expected outcome: No time wasted re-reading files or rediscovering context

5. **Complete Task**
   - User action: `shark task complete T-E13-F05-004 --files-created "ThemeToggle.vue" --tests "8/8 passing"`
   - System response: Task marked ready for review with completion metadata
   - Expected outcome: All work sessions and notes preserved in history

**Success Outcome**: Agent resumes work immediately with zero context-switching overhead

---

## Journey 3: Tech Lead Reviews Completed Task

**Persona**: Human Developer (Technical Lead)

**Goal**: Review a completed task and verify quality before approving

**Preconditions**:
- Task T-E13-F05-003 is `ready_for_review`
- Tech lead needs to verify implementation quality

### Happy Path

1. **View Completion Summary**
   - User action: `shark task get T-E13-F05-003 --completion-details`
   - System response: Shows completion metadata including:
     - Files created: `useTheme.ts`, `useTheme.spec.ts`
     - Tests: "16/16 passing"
     - Verification status: "verified"
     - Agent ID: "a5ad46d"
   - Expected outcome: Tech lead immediately sees what was done

2. **Review Implementation Notes**
   - User action: `shark task notes T-E13-F05-003`
   - System response: Shows chronological notes filtered by type:
     - DECISION: "Used singleton pattern for theme state"
     - IMPLEMENTATION: "localStorage persistence with media query listeners"
     - TESTING: "16/16 tests passing - covered all edge cases"
   - Expected outcome: Tech lead understands approach and decisions

3. **Check Acceptance Criteria**
   - User action: `shark task criteria T-E13-F05-003`
   - System response: Shows 7/7 criteria complete:
     - ✓ Composable returns reactive theme ref
     - ✓ Supports light/dark/system modes
     - ✓ Persists to localStorage
     - ✓ Detects system preference changes
     - ✓ Provides setTheme function
     - ✓ Includes comprehensive tests
     - ✓ TypeScript types exported
   - Expected outcome: Tech lead confirms all requirements met

4. **Approve Task**
   - User action: `shark task approve T-E13-F05-003 --note "Clean implementation, good test coverage"`
   - System response: Task status → `completed`, approval note added to history
   - Expected outcome: Task officially completed with approval note

**Success Outcome**: Tech lead approves task confidently in <5 minutes with full verification

### Alternative Paths

**Alt Path A: Request Changes**
- **Trigger**: Tech lead finds issue during review
- **Branch Point**: After Step 3
- **Flow**:
  1. `shark task reopen T-E13-F05-003 --note "Need to handle null localStorage case"`
  2. Task status → `in_progress`
  3. Agent sees reopen note in timeline on next resume
- **Outcome**: Clear feedback provided, task returned for revision

---

## Journey 4: Product Manager Tracks Feature Progress

**Persona**: Product Manager

**Goal**: Understand feature completion status and identify blockers

**Preconditions**:
- Feature E13-F05 (Dark Mode Feature) has 7 tasks
- PM needs to report progress to stakeholders

### Happy Path

1. **View Feature Progress**
   - User action: `shark feature get E13-F05`
   - System response: Shows 5/7 tasks completed (71% progress)
   - Expected outcome: PM knows overall completion percentage

2. **Check Acceptance Criteria Progress**
   - User action: `shark feature criteria E13-F05`
   - System response: Aggregates criteria across all tasks:
     - Total criteria: 35
     - Complete: 28
     - In progress: 5
     - Failed: 0
     - Pending: 2
   - Expected outcome: PM sees detailed progress breakdown (80% criteria met)

3. **Identify Blocked Tasks**
   - User action: `shark task list E13-F05 --status blocked`
   - System response: Shows 1 blocked task:
     - T-E13-F05-007: Dark Mode Integration Testing
     - Blocked by: T-E13-F05-004 (ThemeToggle component)
     - Blocker type: "dependency"
   - Expected outcome: PM knows exactly what's blocking and why

4. **Find Related Tasks**
   - User action: `shark task deps T-E13-F05-007`
   - System response: Shows dependency graph:
     - Depends on: T-E13-F05-004 (in_progress)
     - Which depends on: T-E13-F05-003 (completed), T-E13-F05-001 (completed)
   - Expected outcome: PM understands dependency chain

**Success Outcome**: PM has complete visibility into progress, blockers, and dependencies for stakeholder reporting

---

## Journey 5: Developer Discovers Related Implementation

**Persona**: Human Developer (Technical Lead)

**Goal**: Find existing implementations similar to current task

**Preconditions**:
- Developer starting new task involving localStorage persistence
- Previous tasks have implementation notes and completion metadata

### Happy Path

1. **Search by Implementation Pattern**
   - User action: `shark search "localStorage persistence"`
   - System response: Returns tasks with matching notes or descriptions:
     - T-E13-F05-003: useTheme composable (note: "localStorage persistence with media query")
     - T-E13-F05-002: Flash prevention (note: "reads wgm-theme-preference from localStorage")
   - Expected outcome: Developer finds 2 related implementations

2. **Search by File Modified**
   - User action: `shark search --file "useTheme.ts"`
   - System response: Returns all tasks that created or modified useTheme.ts:
     - T-E13-F05-003: Create useTheme() Composable
     - T-E13-F05-004: Create ThemeToggle Component (uses useTheme)
   - Expected outcome: Developer understands file's history

3. **View Implementation Details**
   - User action: `shark task notes T-E13-F05-003 --type decision`
   - System response: Shows decision notes:
     - "Used singleton pattern for theme state"
     - "Chose ref() over reactive() for single value"
   - Expected outcome: Developer learns pattern rationale

**Success Outcome**: Developer finds relevant examples and understands patterns without manual codebase search

---

*See also*: [Requirements](./requirements.md), [Personas](./personas.md)
