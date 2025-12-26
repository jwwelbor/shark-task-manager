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

# Related documents
shark related-docs add "<title>" <path> --task=<key>
shark related-docs add "<title>" <path> --feature=<key>
shark related-docs add "<title>" <path> --epic=<key>
shark related-docs list --task=<key>
shark related-docs delete "<title>" --task=<key>
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
