# Shark Task Manager: Complete Workflow Guide

This guide walks you through the complete lifecycle of managing work in Shark, from creating epics down to completing individual tasks.

## Table of Contents

- [Overview](#overview)
- [Step 1: Create an Epic](#step-1-create-an-epic)
- [Step 2: Create Features](#step-2-create-features)
- [Step 3: Create Tasks](#step-3-create-tasks)
- [Step 4: Task Status Lifecycle](#step-4-task-status-lifecycle)
- [Step 5: Tracking Progress](#step-5-tracking-progress)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)

## Overview

Shark uses a hierarchical structure to organize work:

```
Epic (E01)
├── Feature (E01-F01)
│   ├── Task (T-E01-F01-001)
│   ├── Task (T-E01-F01-002)
│   └── Task (T-E01-F01-003)
└── Feature (E01-F02)
    ├── Task (T-E01-F02-001)
    └── Task (T-E01-F02-002)
```

**Key Concepts:**
- **Epic**: Large body of work (e.g., "User Authentication System")
- **Feature**: Discrete functionality within an epic (e.g., "Login Flow", "Password Reset")
- **Task**: Atomic work unit that can be assigned and tracked (e.g., "Implement login API endpoint")

## Step 1: Create an Epic

An epic represents a major initiative or project goal.

### Basic Epic Creation

```bash
# Create a basic epic
shark epic create "User Authentication System" \
  --priority=high \
  --business-value=high

# Output:
# Epic created: E01
```

### View Epic Details

```bash
shark epic get E01

# Output:
# Epic: E01
#
# Title    | User Authentication System
# Status   | active
# Priority | high
# Progress | 0.0%
# Path     | docs/plan/E01/
# Filename | epic.md
```

### Epic Creation with Custom Path

For custom organization (e.g., organizing by quarter):

```bash
shark epic create "Q1 2025 Features" \
  --path="docs/roadmap/2025-q1" \
  --priority=medium

# This creates the epic at: docs/roadmap/2025-q1/E02/epic.md
```

### List All Epics

```bash
shark epic list

# Output (table):
# Key | Title                      | Status | Progress | Priority
# E01 | User Authentication System | active | 0.0%     | high
# E02 | Q1 2025 Features          | active | 0.0%     | medium
```

## Step 2: Create Features

Features break down epics into manageable pieces of functionality.

### Basic Feature Creation

```bash
# Create a feature under an epic
shark feature create --epic=E01 "Login Flow" \
  --execution-order=1

# Output:
# Feature created: E01-F01
```

### Create Multiple Features

```bash
# Create additional features for the authentication epic
shark feature create --epic=E01 "User Registration" --execution-order=2
shark feature create --epic=E01 "Password Reset" --execution-order=3
shark feature create --epic=E01 "Two-Factor Authentication" --execution-order=4

# Output:
# Feature created: E01-F02
# Feature created: E01-F03
# Feature created: E01-F04
```

### Feature with Custom Path

```bash
# Organize feature in a custom location
shark feature create --epic=E01 "OAuth Integration" \
  --path="docs/integrations/oauth" \
  --execution-order=5

# This creates: docs/integrations/oauth/E01-F05/feature.md
```

### View Feature Details

```bash
shark feature get E01-F01

# Output:
# Feature: E01-F01
#
# Title    | Login Flow
# Epic ID  | 1
# Status   | active
# Progress | 0.0%
# Path     | docs/plan/E01/E01-F01/
# Filename | feature.md
#
# Task Status Breakdown
#
# todo            | 0
# in_progress     | 0
# ready_for_review| 0
# completed       | 0
```

### List Features

```bash
# List all features in an epic
shark feature list E01

# Output (table):
# Key     | Title                     | Epic ID | Status | Progress | Tasks | Order
# E01-F01 | Login Flow                | 1       | active | 0.0%     | 0     | 1
# E01-F02 | User Registration         | 1       | active | 0.0%     | 0     | 2
# E01-F03 | Password Reset            | 1       | active | 0.0%     | 0     | 3
# E01-F04 | Two-Factor Authentication | 1       | active | 0.0%     | 0     | 4
```

## Step 3: Create Tasks

Tasks are the atomic units of work that can be assigned and tracked through completion.

### Basic Task Creation

```bash
# Create a task in a feature
shark task create --epic=E01 --feature=F01 \
  "Implement login API endpoint" \
  --priority=8 \
  --agent=backend

# Output:
# Task created: T-E01-F01-001
# File: docs/plan/E01/E01-F01/tasks/T-E01-F01-001.md
```

### Create Multiple Tasks

```bash
# Create a series of tasks for the Login Flow feature
shark task create --epic=E01 --feature=F01 \
  "Design login UI components" \
  --priority=7 \
  --agent=frontend

shark task create --epic=E01 --feature=F01 \
  "Add JWT token generation" \
  --priority=9 \
  --agent=backend \
  --depends-on='["T-E01-F01-001"]'

shark task create --epic=E01 --feature=F01 \
  "Write integration tests for login" \
  --priority=6 \
  --agent=testing \
  --depends-on='["T-E01-F01-001", "T-E01-F01-003"]'

# Output:
# Task created: T-E01-F01-002
# Task created: T-E01-F01-003
# Task created: T-E01-F01-004
```

### Task with Custom Filename

For specialized task documentation (e.g., PRP - Product Requirements Plan):

```bash
shark task create --epic=E01 --feature=F01 \
  "Login Flow Implementation Guide" \
  --filename="docs/plan/E01/E01-F01/PRP/implementation-guide.md" \
  --priority=5

# This creates a task with custom file location
```

### View Task Details

```bash
shark task get T-E01-F01-001

# Output:
# Task: T-E01-F01-001
# Title: Implement login API endpoint
# Status: todo
# Priority: 8
# Path: docs/plan/E01/E01-F01/tasks/
# Filename: T-E01-F01-001.md
# Agent Type: backend
# Created: 2025-12-24 20:45:10
```

### List Tasks

```bash
# List all tasks in a feature
shark task list E01 F01

# Output (table):
# Key           | Title                              | Status | Priority | Agent Type | Order
# T-E01-F01-001 | Implement login API endpoint       | todo   | 8        | backend    | -
# T-E01-F01-002 | Design login UI components         | todo   | 7        | frontend   | -
# T-E01-F01-003 | Add JWT token generation           | todo   | 9        | backend    | -
# T-E01-F01-004 | Write integration tests for login  | todo   | 6        | testing    | -

# Filter by status
shark task list E01 F01 --status=todo

# Filter by agent type
shark task list --agent=backend
```

## Step 4: Task Status Lifecycle

Tasks flow through a series of states from creation to completion. Understanding this lifecycle is crucial for effective project management.

### Task Status States

```
┌──────┐
│ todo │ ──────────────────────┐
└──────┘                       │
   │                           │
   │ shark task start          │
   ▼                           │
┌─────────────┐                │
│ in_progress │ ◄──────────────┤
└─────────────┘                │
   │                           │
   │ shark task complete       │
   ▼                           │
┌──────────────────┐           │
│ ready_for_review │           │
└──────────────────┘           │
   │           │               │
   │           │ shark task    │
   │           │ reopen        │
   │           └───────────────┘
   │
   │ shark task approve
   ▼
┌───────────┐
│ completed │
└───────────┘

          ┌─────────┐
          │ blocked │ ◄── Can transition from any state
          └─────────┘     (shark task block)
```

### 1. Starting Work (todo → in_progress)

```bash
# Start working on a task
shark task start T-E01-F01-001

# Output:
# Task T-E01-F01-001 status updated: todo → in_progress
# Started at: 2025-12-24 20:50:15

# Optionally specify the agent working on it
shark task start T-E01-F01-002 --agent="frontend-agent-1"
```

**What happens:**
- Status changes to `in_progress`
- `started_at` timestamp is recorded
- Optional agent assignment is recorded

### 2. Completing Work (in_progress → ready_for_review)

```bash
# Mark task as complete (ready for review)
shark task complete T-E01-F01-001 \
  --notes="Implemented login endpoint with email/password validation"

# Output:
# Task T-E01-F01-001 status updated: in_progress → ready_for_review
# Completed at: 2025-12-24 21:15:30
```

**What happens:**
- Status changes to `ready_for_review`
- `completed_at` timestamp is recorded
- Optional notes are stored in task history

### 3. Approving Work (ready_for_review → completed)

After code review, testing, or quality gates:

```bash
# Approve the completed work
shark task approve T-E01-F01-001 \
  --notes="Code reviewed, tests passing, merged to main"

# Output:
# Task T-E01-F01-001 status updated: ready_for_review → completed
# Approved at: 2025-12-24 21:30:45
```

**What happens:**
- Status changes to `completed`
- Task is considered done
- Feature and Epic progress updates automatically

### 4. Reopening Work (ready_for_review → in_progress)

If issues are found during review:

```bash
# Reopen a task for additional work
shark task reopen T-E01-F01-002 \
  --notes="Needs additional validation for email format"

# Output:
# Task T-E01-F01-002 status updated: ready_for_review → in_progress
```

### 5. Blocking a Task

When a task is blocked by external dependencies:

```bash
# Block a task with a reason
shark task block T-E01-F01-003 \
  --reason="Waiting for API specification from product team"

# Output:
# Task T-E01-F01-003 status updated: in_progress → blocked
# Blocked at: 2025-12-24 22:00:00
# Reason: Waiting for API specification from product team
```

### 6. Unblocking a Task

```bash
# Unblock a task when dependency is resolved
shark task unblock T-E01-F01-003

# Output:
# Task T-E01-F01-003 status updated: blocked → todo
```

### Force Status Transitions

For administrative overrides (use with caution):

```bash
# Force a task to any status
shark task start T-E01-F01-004 --force

# This bypasses normal state transition validation
# Useful for recovering from incorrect states
```

## Step 5: Tracking Progress

Shark automatically calculates progress at all levels based on task completion.

### View Task Progress

```bash
# Get next available task
shark task next --epic=E01

# Output:
# Next available task:
# Key: T-E01-F01-001
# Title: Implement login API endpoint
# Priority: 8
# Status: todo
# Dependencies: none
```

### View Feature Progress

```bash
shark feature get E01-F01

# Output:
# Feature: E01-F01
#
# Title    | Login Flow
# Progress | 50.0%    # (2 of 4 tasks completed)
#
# Task Status Breakdown
#
# todo             | 0
# in_progress      | 1
# ready_for_review | 1
# completed        | 2
```

### View Epic Progress

```bash
shark epic get E01

# Output:
# Epic: E01
#
# Title    | User Authentication System
# Progress | 25.0%    # Average of all feature progress
#
# Features
#
# Key     | Title                     | Status | Progress | Tasks
# E01-F01 | Login Flow                | active | 50.0%    | 4
# E01-F02 | User Registration         | active | 0.0%     | 0
# E01-F03 | Password Reset            | active | 0.0%     | 0
# E01-F04 | Two-Factor Authentication | active | 0.0%     | 0
```

### JSON Output for Automation

All commands support `--json` for machine-readable output:

```bash
# Get task details in JSON
shark task get T-E01-F01-001 --json

# Output:
{
  "task": {
    "id": 1,
    "feature_id": 1,
    "key": "T-E01-F01-001",
    "title": "Implement login API endpoint",
    "status": "completed",
    "priority": 8,
    "agent_type": "backend",
    "created_at": "2025-12-24T20:45:10Z",
    "started_at": "2025-12-24T20:50:15Z",
    "completed_at": "2025-12-24T21:15:30Z"
  },
  "path": "docs/plan/E01/E01-F01/tasks/",
  "filename": "T-E01-F01-001.md",
  "dependency_status": {},
  "related_documents": []
}
```

## Advanced Features

### Related Documents

Link design docs, specifications, or other documentation to tasks:

```bash
# Add related documents
shark related-docs add "API Specification" docs/api-spec.md --task=T-E01-F01-001
shark related-docs add "UI Mockups" docs/mockups.md --task=T-E01-F01-002

# View task with related docs
shark task get T-E01-F01-001

# Output includes:
# Related Documents:
#   - API Specification (docs/api-spec.md)
```

### Task Dependencies

Create dependency chains between tasks:

```bash
# Create dependent tasks
shark task create --epic=E01 --feature=F01 \
  "Deploy login service" \
  --depends-on='["T-E01-F01-001", "T-E01-F01-003"]' \
  --priority=5

# When viewing the task, dependencies are shown
shark task get T-E01-F01-005

# Output includes:
# Dependencies:
#   - T-E01-F01-001: completed
#   - T-E01-F01-003: in_progress
```

### Custom Organization Paths

Organize your work outside the default structure:

```bash
# Epic organized by time period
shark epic create "2025 Q1 Roadmap" \
  --path="roadmap/2025-q1"

# Feature organized by team
shark feature create --epic=E02 "Mobile App Rewrite" \
  --path="teams/mobile"

# Task with custom documentation structure
shark task create --epic=E02 --feature=F01 \
  "Architecture Design" \
  --filename="teams/mobile/architecture/design-doc.md"
```

## Best Practices

### 1. Epic Planning

**DO:**
- Create epics for major initiatives (2+ weeks of work)
- Use clear, outcome-focused titles
- Set business value to help prioritize
- Organize by time period for roadmap tracking

**DON'T:**
- Create epics for small features (use features instead)
- Leave epic descriptions empty
- Create overlapping epics

### 2. Feature Breakdown

**DO:**
- Break epics into 3-8 features
- Use `execution-order` to sequence work
- Ensure each feature delivers value independently
- Keep feature scope to 1-2 weeks max

**DON'T:**
- Create features that can't be completed independently
- Mix unrelated functionality in one feature
- Create features smaller than tasks

### 3. Task Creation

**DO:**
- Create tasks that can be completed in 1-3 days
- Use clear, action-oriented titles ("Implement...", "Add...", "Fix...")
- Set appropriate priority (1-10)
- Assign agent type for routing
- Document dependencies

**DON'T:**
- Create tasks larger than a few days of work
- Leave tasks unassigned to features
- Create circular dependencies

### 4. Status Management

**DO:**
- Start tasks when actually beginning work
- Complete tasks when implementation is done (not when merged)
- Approve tasks after code review and testing
- Block tasks immediately when dependencies arise
- Use notes to document status changes

**DON'T:**
- Skip status transitions (use the workflow)
- Leave tasks in `in_progress` for weeks
- Approve without review
- Use `--force` except for corrections

### 5. Progress Tracking

**DO:**
- Check `shark task next` to find available work
- Review feature progress regularly
- Use JSON output for dashboard integrations
- Track blocked tasks proactively

**DON'T:**
- Manually edit status in database
- Skip the status workflow
- Ignore dependency warnings

## Example: Complete Workflow

Here's a complete example from epic creation to task completion:

```bash
# 1. Create Epic
shark epic create "Payment Processing" --priority=high --business-value=high
# Created: E03

# 2. Create Features
shark feature create --epic=E03 "Stripe Integration" --execution-order=1
# Created: E03-F01

shark feature create --epic=E03 "Payment UI" --execution-order=2
# Created: E03-F02

# 3. Create Tasks for Stripe Integration
shark task create --epic=E03 --feature=F01 \
  "Set up Stripe API credentials" \
  --priority=9 --agent=backend
# Created: T-E03-F01-001

shark task create --epic=E03 --feature=F01 \
  "Implement payment intent creation" \
  --priority=8 --agent=backend \
  --depends-on='["T-E03-F01-001"]'
# Created: T-E03-F01-002

# 4. Work the tasks
shark task start T-E03-F01-001
# ... do the work ...
shark task complete T-E03-F01-001 --notes="Stripe API configured in staging"
shark task approve T-E03-F01-001 --notes="Verified in staging environment"

# 5. Check progress
shark feature get E03-F01
# Progress: 50.0% (1 of 2 tasks completed)

shark epic get E03
# Progress: 25.0% (1 of 4 total tasks completed)
```

## AI Agent Workflow: Long-Running Tasks Across Sessions

This section demonstrates how AI agents use Shark to manage complex, multi-session tasks with full context preservation.

### Scenario: Complex Feature Implementation

An AI agent is implementing a user authentication system that will span multiple chat sessions due to complexity.

#### Session 1: Starting Work and Capturing Initial Context

```bash
# Agent starts a new chat session and picks up next task
shark task next --agent=backend --json
# Returns: T-E01-F01-001 "Implement login API endpoint"

# Start the task (automatically creates work session)
shark task start T-E01-F01-001 --agent="claude-sonnet-001"
#  SUCCESS  Task started
# Work session #1 created

# Agent begins implementation...
# After 2 hours of work, agent has made progress but isn't done

# Save progress notes as work progresses
shark task note add T-E01-F01-001 \
  "Created login endpoint skeleton at internal/api/login.go" \
  --category=progress

shark task note add T-E01-F01-001 \
  "Decided to use JWT with 24h expiry based on security requirements" \
  --category=decision

shark task note add T-E01-F01-001 \
  "How many failed login attempts before lockout?" \
  --category=question

# Agent encounters a blocker
shark task note add T-E01-F01-001 \
  "Need database schema for user_sessions table from DBA" \
  --category=blocker

# Before ending session, save structured context for easy resume
shark task context set T-E01-F01-001 \
  --progress="Created API endpoint structure, JWT generation implemented, password validation pending" \
  --decisions='["JWT with 24h expiry", "Bcrypt cost factor 12", "Rate limiting at middleware level"]' \
  --questions='["Failed login lockout threshold?", "Should we log failed attempts?"]' \
  --blockers='["Need user_sessions table schema from DBA"]' \
  --acceptance-criteria="2 of 5 criteria passing"

# Mark task as blocked (automatically ends work session)
shark task block T-E01-F01-001 \
  --reason="Waiting for user_sessions table schema"
#  SUCCESS  Task blocked
# Work session #1 ended (Duration: 2h 15m, Outcome: blocked)

# Session 1 ends - context preserved ✓
```

#### Session 2: Resuming After Blocker Resolved (Days Later)

```bash
# New chat session starts, different agent instance
# Agent needs to quickly understand where work was left off

# Get comprehensive resume context
shark task resume T-E01-F01-001

# Output shows:
# ═══════════════════════════════════════════════════════════
# Resume Context: T-E01-F01-001
# Implement login API endpoint
# ═══════════════════════════════════════════════════════════
#
# [1] TASK OVERVIEW
# Status:       blocked
# Priority:     8 (high)
# Agent:        backend
#
# [2] CURRENT PROGRESS
# Created API endpoint structure, JWT generation implemented,
# password validation pending
#
# [3] KEY DECISIONS MADE
#   • JWT with 24h expiry
#   • Bcrypt cost factor 12
#   • Rate limiting at middleware level
#
# [4] OPEN QUESTIONS ⚠️
#   ? Failed login lockout threshold?
#   ? Should we log failed attempts?
#
# [5] BLOCKERS ⚠️
#   ! Need user_sessions table schema from DBA
#
# [6] LAST WORK SESSION
# Started: 2025-12-24 14:00:00
# Ended:   2025-12-24 16:15:00
# Duration: 2h 15m
# Outcome: blocked
#
# [7] RECENT NOTES (last 5)
# 2025-12-24 16:10 | BLOCKER | Need user_sessions schema from DBA
# 2025-12-24 15:30 | QUESTION| How many failed login attempts before lockout?
# 2025-12-24 15:00 | DECISION| Decided to use JWT with 24h expiry
# 2025-12-24 14:30 | PROGRESS| Created login endpoint skeleton
#
# [8] ACCEPTANCE CRITERIA (2/5 passing)
# ✓ JWT token generation works
# ✓ Password bcrypt validation
# ○ Rate limiting implemented
# ○ Failed login tracking
# ○ Session management
#
# [9] SUGGESTED NEXT ACTIONS
#   • Resolve blocker: user_sessions table schema
#   • Answer questions about lockout policy
#   • Complete remaining 3 acceptance criteria

# Agent confirms blocker is resolved
shark task unblock T-E01-F01-001
#  SUCCESS  Task unblocked (blocked → todo)

# Resume work (creates new work session)
shark task start T-E01-F01-001 --agent="claude-sonnet-002"
#  SUCCESS  Task started
# Work session #2 created

# Agent implements remaining functionality...

# Add progress notes
shark task note add T-E01-F01-001 \
  "Implemented rate limiting: 5 attempts per 15 minutes" \
  --category=progress

shark task note add T-E01-F01-001 \
  "Confirmed with product: lockout after 5 failed attempts for 1 hour" \
  --category=decision

# Update context as work progresses
shark task context set T-E01-F01-001 \
  --progress="All functionality complete, tests passing" \
  --acceptance-criteria="5 of 5 criteria passing"

# Mark complete with metadata
shark task complete T-E01-F01-001 \
  --files-created="internal/api/login.go,internal/api/login_test.go,internal/middleware/rate_limit.go" \
  --files-modified="internal/router/routes.go,docs/api/auth.md" \
  --tests \
  --verified \
  --notes="Login API complete with JWT auth, rate limiting, and session management"
#  SUCCESS  Task ready for review
# Work session #2 ended (Duration: 1h 45m, Outcome: completed)
```

#### Session 3: Code Review and Approval (Another Agent)

```bash
# Reviewer agent in new session picks up task for review
shark task list --status=ready_for_review --json
# Returns: T-E01-F01-001

# Get full context for review
shark task resume T-E01-F01-001 --json | jq '{
  title: .title,
  progress: .context.progress,
  decisions: .context.decisions,
  files_created: .completion_metadata.files_created,
  files_modified: .completion_metadata.files_modified,
  tests_written: .completion_metadata.tests_written
}'

# Review passes, approve task
shark task approve T-E01-F01-001 \
  --agent="reviewer-001" \
  --notes="Code review passed. Tests comprehensive, security reviewed."
#  SUCCESS  Task approved and completed
```

### Multi-Task Workflow: Managing Dependencies

```bash
# Agent working on feature with dependent tasks

# View all tasks in feature
shark task list E01 F01 --json

# Create dependency relationships
shark task link T-E01-F01-003 T-E01-F01-001 --type=depends-on
shark task link T-E01-F01-004 T-E01-F01-002 --type=depends-on
shark task link T-E01-F01-005 T-E01-F01-003 --type=depends-on

# Get next available task (respects dependencies)
shark task next --epic=E01 --json
# Returns only tasks with all dependencies completed

# Start task
shark task start T-E01-F01-003 --agent="backend-agent"

# Check what dependencies were required
shark task deps T-E01-F01-003
# Shows: T-E01-F01-001 (completed)

# Check what tasks are blocked by current work
shark task blocks T-E01-F01-003
# Shows: T-E01-F01-005 (todo) - waiting for this task
```

### Pattern: Systematic Progress Tracking

```bash
# Session 1: Start complex multi-file refactoring
shark task start T-E04-F02-001 --agent="refactor-agent"

# Add progress notes after each file
shark task note add T-E04-F02-001 "Refactored user.go - extracted validation" --category=progress
shark task note add T-E04-F02-001 "Refactored auth.go - simplified token logic" --category=progress
shark task note add T-E04-F02-001 "Updated 15 test files to match new structure" --category=progress

# Document architectural decision
shark task note add T-E04-F02-001 \
  "Moved validation to separate package for reusability" \
  --category=decision

# Save context before pausing
shark task context set T-E04-F02-001 \
  --progress="Refactored 12 of 20 files, all tests passing" \
  --decisions='["Validation in separate package", "Kept backward compat wrappers"]'

# End session (task stays in_progress)
# Work session auto-ended when agent stops

# ─────────────────────────────────────────────────────────

# Session 2: Resume next day
shark task resume T-E04-F02-001
# Shows exactly where we left off: 12 of 20 files done

# Continue work...
# Complete remaining 8 files...

shark task complete T-E04-F02-001 \
  --files-modified="internal/user/*.go,internal/auth/*.go,internal/validation/*.go" \
  --tests \
  --verified
```

### Pattern: Handling Interruptions and Context Switches

```bash
# Working on Task A
shark task start T-E01-F01-006 --agent="agent-001"

# 30 minutes in, urgent Task B needs attention
# Save context for Task A first
shark task context set T-E01-F01-006 \
  --progress="Database migration written, needs testing" \
  --questions='["Should we add rollback procedure?"]'

# Switch to urgent task (Task A work session auto-paused)
shark task start T-E01-F02-001 --agent="agent-001"
# ... complete urgent work ...
shark task complete T-E01-F02-001

# Return to Task A
shark task resume T-E01-F01-006
# Shows: "Database migration written, needs testing"
# Question: "Should we add rollback procedure?"

# Continue Task A (new work session created)
shark task start T-E01-F01-006 --agent="agent-001"
```

### Pattern: Team Handoff

```bash
# Agent A starts task, does initial research
shark task start T-E03-F01-001 --agent="research-agent"

shark task note add T-E03-F01-001 \
  "Researched 3 payment providers: Stripe (best fit), Square, PayPal" \
  --category=progress

shark task note add T-E03-F01-001 \
  "Stripe chosen: better API, lower fees, better docs" \
  --category=decision

shark task context set T-E03-F01-001 \
  --progress="Research complete, ready for implementation" \
  --decisions='["Use Stripe", "Store payment methods in vault"]' \
  --questions='["PCI compliance requirements?"]'

# Agent A blocks task for compliance review
shark task block T-E03-F01-001 --reason="Awaiting PCI compliance review"

# ─────────────────────────────────────────────────────────

# Agent B picks up after compliance approved
shark task unblock T-E03-F01-001

# Agent B gets full context from Agent A's work
shark task resume T-E03-F01-001
# Shows all research, decisions, and questions

# Agent B starts implementation with full context
shark task start T-E03-F01-001 --agent="backend-agent"
```

### Best Practices for AI Agents

#### 1. Always Use `task resume` at Session Start

```bash
# ❌ BAD: Starting without context
shark task get T-E01-F01-001 --json  # Missing context, notes, sessions
shark task start T-E01-F01-001

# ✅ GOOD: Get full resume context first
shark task resume T-E01-F01-001 --json
# Review context, then start
shark task start T-E01-F01-001 --agent="agent-id"
```

#### 2. Capture Context Continuously

```bash
# ✅ GOOD: Add notes as decisions are made
shark task note add T-E01-F01-001 "Using Redis for session store" --category=decision
shark task note add T-E01-F01-001 "Completed user service integration" --category=progress
shark task note add T-E01-F01-001 "Need Redis connection string" --category=blocker

# ✅ GOOD: Save structured context before pausing
shark task context set T-E01-F01-001 \
  --progress="3 of 5 services integrated" \
  --decisions='["Redis for sessions", "JWT in Authorization header"]' \
  --blockers='["Need Redis connection string"]'
```

#### 3. Use Complete Command with Metadata

```bash
# ❌ BAD: Missing valuable tracking data
shark task complete T-E01-F01-001

# ✅ GOOD: Full metadata for future reference
shark task complete T-E01-F01-001 \
  --files-created="internal/auth/service.go,internal/auth/service_test.go" \
  --files-modified="cmd/server/main.go,go.mod" \
  --tests \
  --verified \
  --notes="Auth service complete with Redis sessions"
```

#### 4. Leverage Analytics for Planning

```bash
# Check historical session duration for similar tasks
shark analytics sessions --session-duration --agent=backend --json

# Use insights to estimate remaining work
# If average session is 2h and we're 50% done, expect 2h more
```

#### 5. Use Search to Find Related Work

```bash
# Before starting auth work, check what touched auth files
shark search --file="internal/auth/*.go" --json

# Learn from previous tasks' notes
shark notes search "authentication" --json

# Check dependencies
shark task deps T-E01-F01-005 --json
```

### Complete Example: Week-Long Feature Implementation

```bash
# ═══════════════════════════════════════════════════════════
# MONDAY: Session 1 (2 hours)
# ═══════════════════════════════════════════════════════════

shark task next --epic=E01 --json
# Returns: T-E01-F01-001 "Implement OAuth integration"

shark task start T-E01-F01-001 --agent="agent-monday-am"

# Import acceptance criteria from task file
shark task criteria import T-E01-F01-001
#  SUCCESS  Imported 8 acceptance criteria

# Work for 2 hours, make progress on research
shark task note add T-E01-F01-001 "Evaluated OAuth providers" --category=progress
shark task note add T-E01-F01-001 "Auth0 selected for managed solution" --category=decision

shark task context set T-E01-F01-001 \
  --progress="Provider research complete, starting integration" \
  --decisions='["Auth0 for OAuth", "PKCE flow for mobile"]'

# Block due to external dependency
shark task block T-E01-F01-001 --reason="Waiting for Auth0 account setup"
# Work session ends: 2h 0m, outcome: blocked

# ═══════════════════════════════════════════════════════════
# WEDNESDAY: Session 2 (3 hours)
# ═══════════════════════════════════════════════════════════

shark task unblock T-E01-F01-001
shark task resume T-E01-F01-001 --json  # Review context from Monday

shark task start T-E01-F01-001 --agent="agent-wed-pm"

# Implement OAuth callback
shark task note add T-E01-F01-001 "OAuth callback endpoint implemented" --category=progress
shark task criteria check T-E01-F01-001 "OAuth callback handles authorization code"

shark task context set T-E01-F01-001 \
  --progress="OAuth callback working, token exchange implemented" \
  --acceptance-criteria="3 of 8 criteria passing"

# Work session continues...

shark task complete T-E01-F01-001 \
  --files-created="internal/oauth/auth0.go,internal/oauth/auth0_test.go" \
  --files-modified="internal/router/routes.go" \
  --tests \
  --verified
# Work session ends: 3h 15m, outcome: completed

# ═══════════════════════════════════════════════════════════
# THURSDAY: Session 3 - Code Review
# ═══════════════════════════════════════════════════════════

shark task resume T-E01-F01-001 --json
# Reviewer sees all context, notes, decisions, files

shark task approve T-E01-F01-001 --notes="OAuth integration approved"

# ═══════════════════════════════════════════════════════════
# ANALYTICS: Review time spent
# ═══════════════════════════════════════════════════════════

shark task sessions T-E01-F01-001
# Session 1: 2h 0m (blocked)
# Session 2: 3h 15m (completed)
# Total: 5h 15m across 2 work sessions

shark analytics sessions --session-duration --epic=E01
# Use for future task estimation
```

## Quick Reference

### Common Commands

```bash
# Epic management
shark epic create "<title>" [--priority=<level>] [--path=<path>]
shark epic list [--json]
shark epic get <epic-key> [--json]

# Feature management
shark feature create --epic=<key> "<title>" [--execution-order=<n>] [--path=<path>]
shark feature list [<epic-key>] [--json]
shark feature get <feature-key> [--json]

# Task management
shark task create --epic=<key> --feature=<key> "<title>" [options]
shark task list [<epic>] [<feature>] [--status=<status>] [--agent=<type>]
shark task get <task-key> [--json]
shark task next [--epic=<key>] [--agent=<type>]

# Task lifecycle
shark task start <task-key> [--agent=<id>]
shark task complete <task-key> [--notes="<notes>"]
shark task approve <task-key> [--notes="<notes>"]
shark task reopen <task-key> [--notes="<notes>"]
shark task block <task-key> --reason="<reason>"
shark task unblock <task-key>

# Task intelligence (E10 features)
shark task note add <task-key> "<note>" [--category=progress|blocker|question|decision|context]
shark task notes <task-key> [--json]
shark task timeline <task-key> [--json]
shark notes search "<query>" [--category=<type>] [--epic=<key>]

shark task link <source> <target> [--type=depends-on|blocks|relates-to|duplicates]
shark task unlink <source> <target>
shark task deps <task-key> [--json]
shark task blocked-by <task-key> [--json]
shark task blocks <task-key> [--json]

shark task criteria import <task-key>
shark task criteria check <task-key> "<criterion>"
shark task criteria fail <task-key> "<criterion>"
shark feature criteria <feature-key> [--json]

shark task context set <task-key> [--progress="..."] [--decisions="..."] [--questions="..."] [--blockers="..."]
shark task context get <task-key> [--json]
shark task context clear <task-key>
shark task resume <task-key> [--json]
shark task sessions <task-key> [--json]

shark analytics sessions --session-duration [--epic=<key>] [--agent=<type>] [--json]
shark analytics sessions --pause-frequency [--epic=<key>] [--json]

shark search --file="<path>" [--json]
```

### Status Flow

```
todo → in_progress → ready_for_review → completed
  ↕
blocked ←───────────────────────────────────┘
```

### Task Priorities

- **9-10**: Critical/Urgent
- **7-8**: High priority
- **5-6**: Medium priority
- **3-4**: Low priority
- **1-2**: Nice to have

---

For more information, see:
- [CLI Reference](CLI_REFERENCE.md) - Complete command documentation
- [CLAUDE.md](../CLAUDE.md) - Development guidelines
- [README.md](../README.md) - Project overview
