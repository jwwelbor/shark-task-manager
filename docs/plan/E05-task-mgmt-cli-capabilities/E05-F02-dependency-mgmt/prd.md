# Feature: Dependency Management

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E05-task-mgmt-cli-capabilities/epic.md)

## Goal

### Problem

Tasks often depend on other tasks being completed first, but the current system provides minimal dependency support. While E04-F01 stores dependencies as JSON arrays and E04-F03 performs basic checks (excluding tasks with incomplete dependencies from `pm task next`), there's no way to visualize dependency chains, detect circular dependencies before they cause deadlocks, understand which tasks block others (downstream impact), or get recommended task orderings. Agents may create circular dependencies accidentally (Task A depends on B, B depends on C, C depends on A), causing deadlocks. Developers cannot answer questions like "What tasks depend on this one?" or "What's blocking this task from starting?" without manually reading task files. When a completed task is reopened, dependent tasks should potentially be blocked, but there's no automatic mechanism for this. The lack of dependency intelligence makes task prioritization difficult and error-prone.

### Solution

Implement comprehensive dependency management commands and validation built on E04-F01 database. Provide `pm task deps <task-key>` to visualize dependency trees showing both upstream (prerequisites) and downstream (dependents) relationships with status indicators. Implement circular dependency detection that validates dependency chains when adding new dependencies, preventing cycles before they're created. Add automatic dependency validation: when task status changes, check if dependent tasks should be blocked (e.g., if prerequisite moves from "completed" back to "in_progress", block all dependents). Provide dependency queries: `pm task list --blocked-by=<task-key>` to find tasks waiting on specific prerequisite. Use ASCII tree visualization for terminal display and JSON output for programmatic access. Integrate validation into E04-F03 task operations and E04-F06 task creation to enforce dependency integrity.

### Impact

- **Deadlock Prevention**: 100% prevention of circular dependencies through validation at creation time
- **Dependency Visibility**: Instant visualization of dependency chains eliminates manual task file reading
- **Prioritization Intelligence**: Understand task ordering and critical paths through dependency trees
- **Automatic Blocking**: Dependent tasks automatically blocked when prerequisites are reopened, maintaining workflow integrity
- **Error Reduction**: Catch invalid dependencies (non-existent tasks, circular refs) before they cause workflow problems

## User Personas

### Primary Persona: Product Manager / Technical Lead

**Role**: Human developer managing complex multi-task features
**Environment**: Terminal, planning task dependencies and sequencing

**Key Characteristics**:
- Plans feature implementation with task dependencies
- Needs to understand task ordering and critical paths
- Resolves blocking issues across tasks
- Ensures dependencies are correct

**Goals**:
- Visualize task dependencies: `pm task deps T-E01-F02-005`
- Find what's blocking a task from starting
- See downstream impact: "What tasks depend on this?"
- Prevent circular dependencies
- Verify all dependencies are valid

**Pain Points this Feature Addresses**:
- No way to see dependency chains
- Circular dependencies cause deadlocks
- Cannot find what's blocking a task
- Must manually verify dependency correctness
- No understanding of downstream impact

### Secondary Persona: AI Agent (Task Planning)

**Role**: Agent planning work and understanding task relationships
**Environment**: Claude Code CLI, selecting tasks to work on

**Key Characteristics**:
- Needs to understand prerequisites before starting work
- Avoids creating circular dependencies
- Wants to know downstream impact of work
- Requires structured dependency information

**Goals**:
- Check dependencies before starting: `pm task deps T-E01-F02-005 --json`
- Verify all prerequisites are complete
- Understand if task is blocking others
- Create tasks with valid dependencies only

**Pain Points this Feature Addresses**:
- Cannot verify prerequisites without reading files
- May create circular dependencies accidentally
- No way to know if task blocks critical work
- Dependency validation is manual

## User Stories

### Must-Have User Stories

**Story 1: Visualize Dependency Tree**
- As a developer, I want to run `pm task deps T-E01-F02-005`, so that I see both upstream (prerequisites) and downstream (dependents) tasks in a tree view.

**Story 2: Detect Circular Dependencies**
- As a developer, I want the system to prevent circular dependencies, so that I cannot create Task A → B → C → A cycles that cause deadlocks.

**Story 3: Show Dependency Status**
- As a developer, I want dependency trees to show each task's status, so that I can see which prerequisites are complete vs incomplete.

**Story 4: Find Blocking Tasks**
- As a developer, I want to run `pm task list --blocked-by=T-E01-F01-005`, so that I can see all tasks waiting on a specific prerequisite.

**Story 5: Validate Dependencies on Creation**
- As a user, I want task creation to validate dependencies (tasks exist, no cycles), so that I cannot create tasks with invalid dependencies.

**Story 6: Automatic Blocking on Reopening**
- As a system, when a completed task is reopened to "in_progress", I want dependent tasks to be automatically blocked, so that workflow integrity is maintained.

**Story 7: JSON Dependency Output**
- As an AI agent, I want `pm task deps <key> --json` to return structured dependency data, so that I can parse and reason about task relationships.

**Story 8: Show Critical Path**
- As a developer, I want to see which tasks have the longest dependency chains (critical path), so that I can prioritize work on bottleneck tasks.

### Should-Have User Stories

**Story 9: Add Dependencies to Existing Task**
- As a developer, I want to run `pm task add-dep T-E01-F02-005 --depends-on=T-E01-F01-003` to add dependency to existing task, so that I don't edit markdown manually.

**Story 10: Remove Dependencies**
- As a developer, I want to run `pm task remove-dep T-E01-F02-005 --remove=T-E01-F01-003` to remove incorrect dependencies.

**Story 11: Suggest Task Ordering**
- As a developer, I want `pm task list --epic=E01 --order-by-deps` to show tasks in dependency order (prerequisites first), so that I understand ideal work sequence.

**Story 12: Show Dependency Depth**
- As a developer, I want to see dependency chain depth (how many levels deep), so that I can identify overly complex dependency graphs.

### Could-Have User Stories

**Story 13: Validate All Dependencies**
- As a user, I want `pm validate-deps` to check all tasks for circular dependencies and invalid references, so that I can audit entire project.

**Story 14: Dependency Impact Analysis**
- As a developer, I want to see "If I complete this task, N blocked tasks will become available", so that I understand impact of completing work.

## Requirements

### Functional Requirements

**Dependency Tree Visualization (pm task deps):**

1. The system must provide `pm task deps <task-key>` command that displays dependency tree

2. The tree must show both upstream (prerequisites) and downstream (dependents) relationships

3. Tree format must use ASCII art:
   ```
   T-E01-F02-005: Build user authentication

   Prerequisites (upstream):
     ├── T-E01-F01-002: Implement JWT validation [✓ completed]
     │   └── T-E01-F01-001: Database schema setup [✓ completed]
     └── T-E01-F01-005: API authentication docs [⚠ in_progress]

   Dependents (downstream):
     ├── T-E01-F02-007: User profile component [⏸ blocked]
     └── T-E01-F03-001: Protected routes [○ todo]
   ```

4. Each task in tree must show: key, title, and status indicator:
   - ✓ (green): completed
   - → (blue): in_progress
   - ○ (gray): todo
   - ⚠ (yellow): ready_for_review
   - ⏸ (red): blocked

5. Tree must recurse up to 5 levels deep (prevent infinite output on circular refs)

6. If task has no dependencies, show: "No prerequisites"

7. If task has no dependents, show: "No dependents"

**Circular Dependency Detection:**

8. The system must detect circular dependencies using depth-first search (DFS) algorithm

9. Circular dependency check must run when:
   - Creating new task with `--depends-on`
   - Adding dependency with `pm task add-dep`
   - Updating task dependencies via sync

10. If circular dependency detected, operation must fail with error: "Error: Circular dependency detected: T-E01-F01-001 → T-E01-F01-002 → T-E01-F01-003 → T-E01-F01-001"

11. Error must show full cycle path to help debugging

12. Detection algorithm must handle complex graphs (multiple paths, transitive dependencies)

13. Algorithm must complete in <500ms for graphs with 1000 tasks

**Dependency Validation:**

14. The system must validate that all referenced dependency tasks exist in database

15. If non-existent task referenced, operation must fail: "Error: Dependency task T-E99-F99-999 does not exist"

16. Validation must check that dependencies belong to same epic or earlier epics (no forward epic dependencies)

17. The system must allow dependencies across features within same epic

18. The system must allow dependencies to previous epics (E01 → E02 allowed, E02 → E01 allowed)

**Automatic Blocking on Status Change:**

19. When a task with status="completed" is changed to any incomplete status (in_progress, todo, reopened):
    - Query all tasks that depend on this task
    - For each dependent task with status="todo" or "in_progress":
      - Change status to "blocked"
      - Set blocked_reason = "Prerequisite task <key> was reopened"
      - Create task_history record

20. Automatic blocking must be transactional (all or nothing)

21. The system must log: "Blocked 3 dependent tasks due to prerequisite T-E01-F01-005 reopening"

**Dependency Queries:**

22. The system must support `pm task list --blocked-by=<task-key>` to find tasks waiting on specific prerequisite

23. The system must support `pm task list --depends-on=<task-key>` (alias for --blocked-by)

24. The system must support `pm task list --no-deps` to show only tasks with no dependencies (entry points)

25. The system must support `pm task list --has-dependents` to show tasks that other tasks depend on

**Adding Dependencies (pm task add-dep):**

26. The system must provide `pm task add-dep <task-key> --depends-on=<dep-key>` command

27. The command must validate dependency task exists

28. The command must check for circular dependencies before adding

29. The command must update task's depends_on JSON array in database

30. The command must update task markdown file frontmatter

31. The command must create task_history record: "Added dependency: <dep-key>"

32. Multiple dependencies can be added: `--depends-on=T-E01-F01-001,T-E01-F01-002`

**Removing Dependencies (pm task remove-dep):**

33. The system must provide `pm task remove-dep <task-key> --remove=<dep-key>` command

34. The command must remove specified dependency from depends_on array

35. The command must update database and markdown file

36. The command must create task_history record: "Removed dependency: <dep-key>"

37. If dependency doesn't exist, show warning but don't error

**JSON Output:**

38. With `--json` flag, `pm task deps` must return structured data:
    ```json
    {
      "task": {"key": "T-E01-F02-005", "title": "Build user authentication"},
      "prerequisites": [
        {
          "key": "T-E01-F01-002",
          "title": "Implement JWT validation",
          "status": "completed",
          "depth": 1,
          "prerequisites": [
            {"key": "T-E01-F01-001", "title": "Database schema", "status": "completed", "depth": 2}
          ]
        }
      ],
      "dependents": [
        {"key": "T-E01-F02-007", "title": "User profile component", "status": "blocked", "depth": 1}
      ],
      "has_circular_dependency": false,
      "max_depth": 2
    }
    ```

**Critical Path Identification:**

39. The system must calculate dependency chain depth for each task

40. The system must support `pm task list --order-by-depth` to show tasks ordered by dependency depth (shallowest first)

41. Tasks with no dependencies have depth 0

42. Tasks depending on depth-N tasks have depth N+1

43. Critical path is the longest chain from entry point (depth 0) to any task

**Integration with Task Operations:**

44. `pm task next` must exclude tasks with incomplete dependencies (already in E04-F03, enhanced here)

45. `pm task start` must warn if starting task with incomplete dependencies: "Warning: Task has 2 incomplete prerequisites"

46. `pm task complete` must check if completion unblocks dependent tasks (log message)

47. `pm task create` must validate dependencies and check for cycles

### Non-Functional Requirements

**Performance:**

- Dependency tree generation must complete in <200ms for chains up to 10 levels deep
- Circular dependency detection must complete in <500ms for graphs with 1000 tasks
- Automatic blocking of dependents must complete in <500ms for 100 dependent tasks
- Dependency queries (--blocked-by) must return in <100ms

**Usability:**

- Tree visualization must be readable and fit in 80-column terminal
- Error messages must show full cycle path for debugging
- Status indicators must be color-coded and clear
- JSON output must be parseable and complete

**Reliability:**

- Circular dependency detection must never have false positives
- Automatic blocking must be transactional (all or nothing)
- Dependency validation must catch all invalid references
- Algorithm must handle edge cases (self-dependencies, empty arrays)

**Data Integrity:**

- Dependencies must always reference valid task IDs
- Circular dependencies must be impossible to create
- Dependency arrays must be valid JSON
- Status changes must maintain dependency consistency

## Acceptance Criteria

### Dependency Tree Visualization

**Given** task T-E01-F02-005 depends on T-E01-F01-002 and T-E01-F01-005
**And** T-E01-F01-002 depends on T-E01-F01-001
**And** T-E01-F02-007 depends on T-E01-F02-005
**When** I run `pm task deps T-E01-F02-005`
**Then** tree shows:
- Prerequisites: T-E01-F01-002 (with nested T-E01-F01-001) and T-E01-F01-005
- Dependents: T-E01-F02-007
**And** each task shows status indicator

### Circular Dependency Detection

**Given** task T-E01-F01-001 depends on T-E01-F01-002
**And** task T-E01-F01-002 depends on T-E01-F01-003
**When** I try to create T-E01-F01-003 with dependency on T-E01-F01-001
**Then** error is displayed: "Circular dependency detected: T-E01-F01-001 → T-E01-F01-002 → T-E01-F01-003 → T-E01-F01-001"
**And** task is not created
**And** exit code is 3 (validation error)

### Automatic Blocking on Reopening

**Given** task T-E01-F01-005 has status="completed"
**And** tasks T-E01-F02-007 and T-E01-F02-009 depend on T-E01-F01-005
**And** both dependents have status="todo"
**When** I run `pm task reopen T-E01-F01-005`
**Then** T-E01-F01-005 status changes to "in_progress"
**And** T-E01-F02-007 status changes to "blocked"
**And** T-E01-F02-009 status changes to "blocked"
**And** blocking reason is "Prerequisite task T-E01-F01-005 was reopened"
**And** message shows: "Blocked 2 dependent tasks"

### Blocked-By Query

**Given** 5 tasks depend on T-E01-F01-005
**When** I run `pm task list --blocked-by=T-E01-F01-005`
**Then** all 5 dependent tasks are returned
**And** only tasks with T-E01-F01-005 in depends_on array are shown

### Adding Dependencies

**Given** task T-E01-F02-005 exists with no dependencies
**When** I run `pm task add-dep T-E01-F02-005 --depends-on=T-E01-F01-002`
**Then** task's depends_on array includes "T-E01-F01-002"
**And** database record is updated
**And** markdown file frontmatter is updated
**And** task_history record is created

**Given** adding dependency would create circular dependency
**When** I run `pm task add-dep <key> --depends-on=<circular-dep>`
**Then** error is displayed with cycle path
**And** dependency is not added

### Removing Dependencies

**Given** task T-E01-F02-005 depends on [T-E01-F01-002, T-E01-F01-005]
**When** I run `pm task remove-dep T-E01-F02-005 --remove=T-E01-F01-002`
**Then** depends_on array becomes ["T-E01-F01-005"]
**And** database and file are updated
**And** task_history record is created

### JSON Output

**Given** I run `pm task deps T-E01-F02-005 --json`
**When** the command completes
**Then** output is valid JSON
**And** JSON includes prerequisites array with nested structure
**And** JSON includes dependents array
**And** JSON includes has_circular_dependency=false
**And** I can parse with `jq '.prerequisites[0].key'`

### Validation on Task Creation

**Given** I try to create task with non-existent dependency
**When** I run `pm task create --depends-on=T-E99-F99-999 ...`
**Then** error is displayed: "Dependency task T-E99-F99-999 does not exist"
**And** task is not created

### Critical Path Calculation

**Given** task T-E01-F01-001 has depth 0 (no dependencies)
**And** task T-E01-F01-002 depends on T-E01-F01-001 (depth 1)
**And** task T-E01-F02-005 depends on T-E01-F01-002 (depth 2)
**When** I run `pm task list --order-by-depth`
**Then** tasks are ordered: T-E01-F01-001, T-E01-F01-002, T-E01-F02-005
**And** depth values are correct

### Performance

**Given** a dependency graph with 1000 tasks
**When** I run circular dependency detection
**Then** detection completes in <500ms

**Given** a task with 100 dependents
**When** I reopen the prerequisite task
**Then** all 100 dependents are blocked in <500ms

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Visual Dependency Graphs** - Graphical visualization (PNG, SVG) is out of scope. Only ASCII tree visualization.

2. **Dependency Templates** - Saved dependency patterns or templates for common task sequences.

3. **Soft Dependencies** - "Nice to have" vs "required" dependency types.

4. **Time-Based Dependencies** - "Task B can start 2 days after Task A completes."

5. **Resource Dependencies** - Dependencies based on shared resources or team members.

6. **Automatic Task Ordering** - Automatically reordering task list by dependencies (users can use --order-by-depth manually).

7. **Dependency Import/Export** - Bulk dependency management via CSV.

8. **Cross-Epic Dependency Validation** - Enforcing rules like "Epic E02 cannot depend on Epic E03."

9. **Dependency Change History** - Detailed audit trail of dependency additions/removals (basic history in task_history only).

10. **Predictive Blocking** - "If you start this task, these other tasks will be blocked" warnings.
