# Feature: Task Lifecycle Operations

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/epic.md)

## Goal

### Problem

AI agents and developers need efficient, reliable commands to manage task lifecycle from creation to completion. Agents must quickly find the next available task matching their specialty without reading multiple markdown files. Developers need to query tasks by multiple criteria (status, epic, feature, agent type, priority), transition tasks through states (todo → in_progress → ready_for_review → completed), handle blocked tasks, and reopen tasks for rework. Without structured commands, agents waste up to 50K tokens and 2-3 minutes per session just determining what to work on next. Manual status transitions are error-prone, leading to inconsistent states where folder location doesn't match database status. There's no validation of state transitions (e.g., preventing tasks from going directly from "todo" to "completed"), and no automatic history tracking of who changed what when.

### Solution

Implement comprehensive CLI commands for task lifecycle management built on E04-F01 (Database) and E04-F02 (CLI Framework). Provide commands for querying tasks (`shark task list`, `shark task get`), discovering work (`shark task next`), transitioning states (`shark task start`, `shark task complete`, `shark task approve`, `shark task block`, `shark task unblock`, `shark task reopen`), and viewing metadata. Each command validates state transitions, updates the database atomically, records history, triggers file operations (via E04-F05), and returns both human-readable and JSON output. The `shark task next` command implements intelligent task selection based on agent type, epic filter, priority, and basic dependency checking (ensures dependencies are not in "todo" or "blocked" status).

### Impact

- **Agent Efficiency**: Reduce task discovery time from 120 seconds to <5 seconds with `shark task next --agent=frontend --json`
- **State Consistency**: 100% database/file consistency through validated state transitions and atomic updates
- **Workflow Automation**: Agents can fully automate task selection, execution, and completion without human intervention
- **Developer Productivity**: Single commands replace manual folder navigation, file reading, and status checking
- **Data Quality**: Automatic history recording enables E05-F03 audit trails and retrospective analysis

## User Personas

### Primary Persona: AI Agent (Claude Code Agents)

**Role**: Autonomous code generation and task execution agent
**Environment**: Claude Code CLI, stateless between sessions, token-constrained

**Key Characteristics**:
- Needs to find next available task in <10 seconds
- Must parse command output programmatically (JSON)
- Cannot manually check dependencies or read task files
- Requires atomic status updates (no partial states)

**Goals**:
- Run `shark task next --agent=backend --json` to get next task instantly
- Update status with single command: `shark task start <key>`
- Mark task ready for review: `shark task complete <key>`
- Block tasks when prerequisites are missing: `shark task block <key> --reason="..."`

**Pain Points this Feature Addresses**:
- Current system requires reading 20+ files to find available tasks
- No way to validate dependencies before starting work
- Manual status updates require editing markdown frontmatter
- No structured way to report blocking issues

### Secondary Persona: Product Manager / Technical Lead

**Role**: Human developer managing multi-epic projects with AI agents
**Environment**: Terminal, managing 2-5 concurrent epics

**Key Characteristics**:
- Needs to review agent-completed work
- Queries tasks by multiple filters
- Handles blocked tasks and reopens work
- Requires visibility into task status

**Goals**:
- List tasks by status: `shark task list --status=ready-for-review`
- Review specific task: `shark task get T-E01-F02-003`
- Approve completed work: `shark task approve <key>`
- Reopen for rework: `shark task reopen <key> --notes="Add error handling"`
- Unblock tasks: `shark task unblock <key>`

**Pain Points this Feature Addresses**:
- No easy way to filter tasks by multiple criteria
- Manual status updates require editing markdown frontmatter
- No tracking of who approved/reopened tasks
- Can't see task details without opening files

## User Stories

### Must-Have User Stories

**Story 1: List Tasks with Filters**
- As a developer, I want to run `shark task list --status=todo --agent=frontend --epic=E01`, so that I can see all available frontend tasks for Epic E01.

**Story 2: Get Task Details**
- As a developer or agent, I want to run `shark task get T-E01-F02-003 --json`, so that I can retrieve full task details including file path, dependencies, and status.

**Story 3: Find Next Available Task**
- As an AI agent, I want to run `shark task next --agent=backend --epic=E01 --json`, so that I can get the highest-priority available task without reading multiple files.

**Story 4: Start a Task**
- As an AI agent, I want to run `shark task start T-E01-F02-003`, so that the task status updates to "in_progress" and I'm assigned as the agent.

**Story 5: Complete a Task**
- As an AI agent, I want to run `shark task complete T-E01-F02-003 --notes="Implemented user authentication"`, so that the task moves to "ready_for_review" status and I can record completion notes.

**Story 6: Approve a Task**
- As a developer, I want to run `shark task approve T-E01-F02-003`, so that reviewed work moves to "completed" status.

**Story 7: Block a Task**
- As an AI agent, I want to run `shark task block T-E01-F02-003 --reason="Missing API specification"`, so that the task is marked as blocked with a clear reason.

**Story 8: Unblock a Task**
- As a developer, I want to run `shark task unblock T-E01-F02-003`, so that a previously blocked task returns to "todo" status and becomes available again.

**Story 9: Reopen a Task**
- As a developer, I want to run `shark task reopen T-E01-F02-003 --notes="Add error handling"`, so that a task in "ready_for_review" returns to "in_progress" for additional work.

**Story 10: Validate State Transitions**
- As a user, I want invalid state transitions (e.g., "completed" → "in_progress") to be rejected with clear error messages, so that task state remains consistent.

### Should-Have User Stories

**Story 11: Filter by Multiple Statuses**
- As a developer, I want to run `shark task list --status=todo,in_progress`, so that I can see all active work.

**Story 12: Filter by Priority Range**
- As a developer, I want to run `shark task list --priority-max=3`, so that I can see only high-priority tasks.

**Story 13: Sort Task Results**
- As a developer, I want to run `shark task list --sort-by=priority --sort-order=asc`, so that tasks are ordered by priority.

**Story 14: Check Task Dependencies**
- As an AI agent, I want `shark task next` to automatically exclude tasks with incomplete dependencies, so that I don't start work that will be blocked.

### Could-Have User Stories

**Story 15: Bulk Status Updates**
- As a developer, I want to run `shark task bulk-update --status=todo --epic=E01 --new-status=blocked`, so that I can handle epic-wide blockers efficiently.

**Story 16: Task Assignment**
- As a developer, I want to run `shark task assign T-E01-F02-003 --agent=frontend-specialist`, so that I can manually assign tasks to specific agents.

## Requirements

### Functional Requirements

**Task Listing (shark task list):**

1. The system must provide `shark task list` command that returns all tasks

2. The system must support filtering by `--status` (single or multiple values: `--status=todo,in_progress`)

3. The system must support filtering by `--epic` (single value: `--epic=E01`)

4. The system must support filtering by `--feature` (single value: `--feature=E01-F02`)

5. The system must support filtering by `--agent` (single value: `--agent=frontend`)

6. The system must support filtering by `--priority` (single value or range: `--priority=1` or `--priority-min=1 --priority-max=3`)

7. The system must support `--blocked` flag to show only blocked tasks

8. Multiple filters must be combined with AND logic (all conditions must match)

9. Human-readable output must display tasks as a table with columns: Key | Title | Status | Priority | Agent

10. JSON output must return: `{"results": [<task objects>], "count": <integer>}`

**Task Details (shark task get):**

11. The system must provide `shark task get <task-key>` command that returns a single task by key

12. If task does not exist, exit with code 1 and message "Error: Task <key> does not exist"

13. Human-readable output must display all task fields including: key, title, description, status, agent_type, priority, depends_on, file_path, timestamps

14. JSON output must return full task object with all fields

15. The system must include dependency status in output (whether each dependency is complete)

**Task Discovery (shark task next):**

16. The system must provide `shark task next` command that returns the highest-priority available task

17. The system must filter by `--agent` (optional): only return tasks matching agent_type

18. The system must filter by `--epic` (optional): only return tasks in specified epic

19. The system must exclude tasks with status other than "todo"

20. The system must exclude tasks with incomplete dependencies (basic check: dependencies not in "completed" or "archived" status)

21. Priority is determined by `priority` field (1 = highest, 10 = lowest)

22. If no tasks are available, return empty result with message "No available tasks found"

23. JSON output must include: task key, title, file_path, dependencies, dependency_status

**Task Start (shark task start):**

24. The system must provide `shark task start <task-key>` command that transitions task to "in_progress"

25. The command must validate current status is "todo" (reject if status is not "todo")

26. The command must update database: status = "in_progress", started_at = current UTC timestamp, assigned_agent = current user/agent identifier

27. The command must record history: old_status="todo", new_status="in_progress", timestamp, agent

28. File path remains unchanged (status tracked in database only)

29. If task is blocked (has incomplete dependencies), warn but allow start (agent may be working around blocker)

30. The command must update feature progress calculation

**Task Complete (shark task complete):**

31. The system must provide `shark task complete <task-key>` command that transitions task to "ready_for_review"

32. The command must validate current status is "in_progress" (reject if not)

33. The command must accept optional `--notes` flag to record completion notes

34. The command must update database: status = "ready_for_review", completed_at = current UTC timestamp

35. The command must record history with notes if provided

36. File path remains unchanged (status tracked in database only)

37. The command must update feature progress calculation

**Task Approve (shark task approve):**

38. The system must provide `shark task approve <task-key>` command that transitions task to "completed"

39. The command must validate current status is "ready_for_review" (reject if not)

40. The command must accept optional `--notes` flag to record approval notes

41. The command must update database: status = "completed"

42. The command must record history with notes if provided

43. File path remains unchanged (status tracked in database only)

44. The command must update feature progress calculation (completed task contributes to 100%)

**Task Block (shark task block):**

45. The system must provide `shark task block <task-key>` command that transitions task to "blocked"

46. The command must accept required `--reason` flag explaining why task is blocked

47. The command must validate current status (can block from "todo" or "in_progress")

48. The command must update database: status = "blocked", blocked_at = current UTC timestamp, blocked_reason = reason

49. The command must record history with blocking reason

50. File path remains unchanged (status tracked in database only)

**Task Unblock (shark task unblock):**

51. The system must provide `shark task unblock <task-key>` command that removes block

52. The command must validate current status is "blocked" (reject if not)

53. The command must return task to "todo" status (reset to initial state)

54. The command must clear blocked_reason and blocked_at fields

55. The command must record history: old_status="blocked", new_status="todo"

56. File path remains unchanged (status tracked in database only)

**Task Reopen (shark task reopen):**

57. The system must provide `shark task reopen <task-key>` command that returns task to "in_progress"

58. The command must validate current status is "ready_for_review" (reject if not)

59. The command must accept optional `--notes` flag explaining why task needs rework

60. The command must update database: status = "in_progress", clear completed_at

61. The command must record history with reopen notes

62. File path remains unchanged (status tracked in database only)

63. The command must update feature progress calculation (no longer counts toward completion)

**State Transition Validation:**

64. The system must define valid state transitions:
    - todo → in_progress (start)
    - todo → blocked (block)
    - in_progress → ready_for_review (complete)
    - in_progress → blocked (block)
    - ready_for_review → completed (approve)
    - ready_for_review → in_progress (reopen)
    - blocked → todo (unblock)

65. All other transitions must be rejected with error: "Error: Invalid state transition from <old_status> to <new_status>"

66. The system must validate transitions before database updates

**History Recording:**

67. Every status change must create a task_history record with: task_id, old_status, new_status, agent, notes, timestamp

68. History must be recorded atomically with status update (same transaction)

69. Agent identifier must be derived from: --agent flag, environment variable USER, or "unknown"

**Integration with Other Features:**

70. Task file paths never change (tasks remain in feature/tasks/ directory regardless of status)

71. All progress updates must call E04-F01 progress calculation functions

72. All commands must use E04-F02 CLI framework (Click, Rich, error handling)

### Non-Functional Requirements

**Performance:**

- `shark task list` must return results in <100ms for 1,000 tasks
- `shark task next` must return result in <50ms
- Status update commands must complete in <200ms (including database + file operations)
- Filtering and sorting must not cause full table scans (use indexes)

**Usability:**

- Error messages must be specific: "Cannot start task T-E01-F02-003 because status is 'in_progress'. Use 'shark task get' to see current status."
- Success messages must confirm action: "Task T-E01-F02-003 started. Status: in_progress"
- JSON output must be parseable by agents without ambiguity
- Table output must fit in 80-column terminal width

**Reliability:**

- Database updates and file operations must be atomic (transaction rollback on failure)
- State transitions must be validated before execution
- Concurrent updates must use database locks to prevent race conditions
- Failed file operations must rollback database changes

**Data Integrity:**

- History records must be immutable (append-only, no updates or deletes)
- Status transitions must follow defined state machine
- Timestamps must always be UTC
- All database updates must use transactions

## Acceptance Criteria

### Task Listing

**Given** the database contains 50 tasks across 3 epics
**When** I run `shark task list --status=todo --epic=E01`
**Then** only tasks with status="todo" and epic="E01" are returned
**And** the query completes in <100ms

**Given** I run `shark task list --json`
**When** the command completes
**Then** output is valid JSON with structure: `{"results": [...], "count": N}`

### Task Discovery

**Given** Epic E01 has 5 frontend tasks with status="todo" and priorities [1, 3, 5, 7, 9]
**When** I run `shark task next --agent=frontend --epic=E01`
**Then** the task with priority=1 is returned (highest priority)

**Given** all tasks have incomplete dependencies
**When** I run `shark task next --agent=frontend`
**Then** result is empty with message "No available tasks found"

**Given** no tasks match the filters
**When** I run `shark task next --agent=nonexistent`
**Then** exit code is 0 (not an error)
**And** message is "No available tasks found"

### Task Start

**Given** task T-E01-F02-003 has status="todo"
**When** I run `shark task start T-E01-F02-003`
**Then** task status is updated to "in_progress"
**And** started_at is set to current UTC time
**And** assigned_agent is set
**And** a history record is created
**And** file path remains unchanged
**And** success message is displayed

**Given** task T-E01-F02-003 has status="in_progress"
**When** I run `shark task start T-E01-F02-003`
**Then** error is displayed: "Invalid state transition from in_progress to in_progress"
**And** exit code is 3 (validation error)
**And** no database changes occur

### Task Complete

**Given** task T-E01-F02-003 has status="in_progress"
**When** I run `shark task complete T-E01-F02-003 --notes="Implemented user auth"`
**Then** task status is updated to "ready_for_review"
**And** completed_at is set to current UTC time
**And** history record includes notes "Implemented user auth"
**And** file path remains unchanged

**Given** task T-E01-F02-003 has status="todo"
**When** I run `shark task complete T-E01-F02-003`
**Then** error is displayed: "Cannot complete task with status 'todo'. Task must be 'in_progress'."
**And** exit code is 3

### Task Approve

**Given** task T-E01-F02-003 has status="ready_for_review"
**When** I run `shark task approve T-E01-F02-003`
**Then** task status is updated to "completed"
**And** history record is created
**And** file path remains unchanged
**And** feature progress is recalculated

### Task Block

**Given** task T-E01-F02-003 has status="in_progress"
**When** I run `shark task block T-E01-F02-003 --reason="Missing API docs"`
**Then** task status is updated to "blocked"
**And** blocked_reason is "Missing API docs"
**And** blocked_at is set to current UTC time
**And** file path remains unchanged

**Given** I run `shark task block T-E01-F02-003` without --reason flag
**When** the command executes
**Then** error is displayed: "Missing required option '--reason'"
**And** exit code is 1 (user error)

### Task Unblock

**Given** task T-E01-F02-003 has status="blocked"
**When** I run `shark task unblock T-E01-F02-003`
**Then** task status is updated to "todo"
**And** blocked_reason is cleared (NULL)
**And** blocked_at is cleared (NULL)
**And** file path remains unchanged

### Task Reopen

**Given** task T-E01-F02-003 has status="ready_for_review"
**When** I run `shark task reopen T-E01-F02-003 --notes="Add error handling"`
**Then** task status is updated to "in_progress"
**And** completed_at is cleared (NULL)
**And** history record includes notes
**And** file path remains unchanged

### State Transition Validation

**Given** task has status="completed"
**When** I attempt any status change (start, complete, block)
**Then** error is displayed: "Invalid state transition from completed to <new_status>"
**And** no database changes occur

### History Recording

**Given** I run `shark task start T-E01-F02-003`
**When** the command completes successfully
**Then** a task_history record exists with:
- task_id = <id of T-E01-F02-003>
- old_status = "todo"
- new_status = "in_progress"
- timestamp = current UTC time
- agent = current user/agent identifier

### Atomic Transactions

**Given** database update succeeds but file operation fails
**When** I run `shark task start T-E01-F02-003`
**Then** the database transaction is rolled back
**And** task status remains "todo"
**And** no history record is created
**And** error is displayed with exit code 2

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Task Creation** - Creating new tasks is in E04-F06 (Task Creation & Templating).

2. **Task Deletion** - Deleting tasks is not part of core lifecycle. Archive functionality may be added later.

3. **Dependency Graph Visualization** - Showing dependency trees is in E05-F02 (Dependency Management).

4. **Advanced Dependency Validation** - Circular dependency detection and deep dependency checking are in E05-F02.

5. **Status Dashboard** - The `shark status` command showing progress bars and metrics is in E05-F01 (Status Dashboard).

6. **Batch Operations** - Bulk status updates are in optional E05 features.

7. **Task Assignment UI** - Manual assignment beyond automatic agent setting is out of scope.

8. **Task Estimation** - Story points and velocity tracking are in optional E05 features.

9. **File Content Operations** - Reading or editing task markdown files is not included (users/agents do this manually).

10. **Advanced Filtering** - Full-text search, regex filters, and saved filter profiles are in optional E05 features.
