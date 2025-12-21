# Feature: Status Dashboard & Reporting

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E05-task-mgmt-cli-capabilities/epic.md)

## Goal

### Problem

Developers and managers need quick, comprehensive visibility into project health across multiple epics without running multiple commands or parsing JSON output. Answering simple questions like "What's the overall project status?" or "Which tasks are currently active?" requires running `shark epic list`, `shark task list --status=active`, `shark task list --status=blocked`, and mentally aggregating results. There's no single view showing epic progress, active work, blockers, and recent activity. Stakeholders need progress reports that currently require manual effort: querying multiple commands, exporting JSON, and formatting in external tools. Without a unified dashboard, users spend 5+ minutes gathering information that should be available instantly, and decision-making is delayed by lack of visibility into bottlenecks and project health.

### Solution

Implement `shark status` command that provides a comprehensive, at-a-glance dashboard of entire project state with optional epic-level filtering. Display project summary (total epics/features/tasks), epic-level breakdown with ASCII progress bars showing completion percentages, active tasks grouped by agent type, blocked tasks with blocking reasons, and recently completed tasks (last 24 hours). Support filtering to specific epics (`shark status --epic=E01`) for focused views. Provide JSON output mode for generating external reports or feeding into other tools. Use Rich library for terminal formatting with tables, progress bars, and color-coded status indicators (green for on-track, yellow for warnings, red for blockers). Build on E04-F01 (Database queries), E04-F02 (CLI Framework), and E04-F04 (Epic/Feature queries) to efficiently aggregate and display data.

### Impact

- **Decision Speed**: Reduce time to answer "what's the project status?" from 5+ minutes to <5 seconds
- **Bottleneck Visibility**: Immediately identify blocked tasks and epics needing attention through color-coded indicators
- **Stakeholder Reporting**: Generate progress reports instantly with `shark status --json | jq` instead of manual aggregation
- **Team Coordination**: See active work distribution across agent types to balance workload
- **Progress Transparency**: Real-time progress bars eliminate stale status reports and manual calculations

## User Personas

### Primary Persona: Product Manager / Technical Lead

**Role**: Human developer managing multi-epic projects
**Environment**: Terminal, managing 2-5 concurrent epics, reporting to stakeholders

**Key Characteristics**:
- Needs quick project health visibility
- Reports progress weekly to stakeholders
- Makes prioritization decisions based on bottlenecks
- Coordinates work across multiple agent types

**Goals**:
- See project status at a glance: `shark status`
- Identify blocked work immediately
- Understand epic completion percentages
- See what each agent is working on
- Generate stakeholder reports

**Pain Points this Feature Addresses**:
- Running 5+ commands to get complete picture
- Manual calculation of progress percentages
- No visibility into blockers across all epics
- Time-consuming report generation

### Secondary Persona: AI Agent (Reporting to Users)

**Role**: Agent providing status updates to users
**Environment**: Claude Code CLI, needs to report project state

**Key Characteristics**:
- Needs to answer "what's the status?" questions
- Reports progress after completing work
- Identifies next priorities
- Provides structured status in messages

**Goals**:
- Get comprehensive status: `shark status --json`
- Parse and present status to users
- Identify incomplete work
- Report blockers and active tasks

**Pain Points this Feature Addresses**:
- No single command for complete status
- Difficult to parse multiple command outputs
- Cannot easily explain project state to users
- No structured format for status reporting

## User Stories

### Must-Have User Stories

**Story 1: View Project Dashboard**
- As a developer, I want to run `shark status`, so that I see a comprehensive dashboard with epic progress, active tasks, blocked tasks, and recent completions.

**Story 2: See Epic Progress Bars**
- As a developer, I want to see visual progress bars for each epic showing completion percentage, so that I can quickly identify epics needing attention.

**Story 3: Identify Blocked Tasks**
- As a developer, I want blocked tasks displayed prominently with blocking reasons, so that I can prioritize resolving blockers.

**Story 4: See Active Work Distribution**
- As a developer, I want active tasks grouped by agent type, so that I understand workload distribution and can balance assignments.

**Story 5: View Recent Completions**
- As a developer, I want to see recently completed tasks (last 24 hours), so that I understand recent progress and momentum.

**Story 6: Filter by Epic**
- As a developer, I want to run `shark status --epic=E01`, so that I can focus on a single epic's status without noise from other epics.

**Story 7: Export JSON for Reports**
- As a developer, I want to run `shark status --json`, so that I can generate stakeholder reports or feed data into external tools.

**Story 8: Color-Coded Status Indicators**
- As a developer, I want color-coded indicators (green=on-track, yellow=warnings, red=blockers), so that I can quickly identify issues.

### Should-Have User Stories

**Story 9: Show Task Counts by Status**
- As a developer, I want to see task count breakdown (todo: 15, in_progress: 5, completed: 30), so that I understand overall workflow state.

**Story 10: Highlight High-Priority Blockers**
- As a developer, I want high-priority blocked tasks highlighted separately, so that I focus on critical blockers first.

**Story 11: Show Epic Health Indicators**
- As a developer, I want epics with >50% blocked tasks marked as "unhealthy", so that I can identify problematic epics.

**Story 12: Recent Activity Timeframe**
- As a developer, I want to specify `shark status --recent=7d` to see completions from last 7 days, so that I can customize recency window.

### Could-Have User Stories

**Story 13: Velocity Metrics**
- As a developer, I want to see average tasks completed per day/week, so that I can estimate completion dates.

**Story 14: Burndown Visualization**
- As a developer, I want ASCII burndown chart showing remaining tasks over time, so that I can visualize progress trends.

## Requirements

### Functional Requirements

**Status Command (shark status):**

1. The system must provide `shark status` command that displays comprehensive project dashboard

2. The dashboard must include these sections (in order):
   - Project Summary
   - Epic Breakdown
   - Active Tasks
   - Blocked Tasks
   - Recent Completions

3. The command must support `--epic=<epic-key>` flag to filter to single epic

4. The command must support `--json` flag for machine-readable output

5. The command must support `--no-color` flag for plain text output

**Project Summary Section:**

6. The summary must display:
   - Total number of epics
   - Total number of features
   - Total number of tasks
   - Overall completion percentage (weighted average of epic progress)
   - Total blocked tasks count

7. Format:
   ```
   PROJECT SUMMARY
   ===============
   Epics: 5 (3 active, 2 completed)
   Features: 23 (15 active, 8 completed)
   Tasks: 127 (45 todo, 12 in progress, 5 ready for review, 60 completed, 5 blocked)
   Overall Progress: 47.3%
   Blocked: 5 tasks
   ```

**Epic Breakdown Section:**

8. The epic breakdown must display table with columns: Epic | Title | Progress | Tasks | Status

9. Progress must be displayed as both percentage and ASCII progress bar

10. Progress bars must use 20-character width: `[##########----------]` (50%)

11. Progress bar characters: `#` for completed, `-` for remaining

12. Epics must be sorted by priority (high → low), then by key

13. Format:
   ```
   EPIC BREAKDOWN
   ==============
   Epic   Title                        Progress                  Tasks    Status
   ─────────────────────────────────────────────────────────────────────────────
   E01    Identity Platform            [############--------] 60%  30/50    active
   E02    Task Management CLI          [########------------] 40%  20/50    active
   E03    Documentation System         [####################] 100% 10/10    completed
   ```

14. Progress must be color-coded:
    - Green: ≥75% complete
    - Yellow: 25-74% complete
    - Red: <25% complete OR has >3 blocked tasks

**Active Tasks Section:**

15. The active tasks section must display tasks with status="in_progress"

16. Tasks must be grouped by agent_type

17. Each group must show agent type header and list of tasks

18. Format:
   ```
   ACTIVE TASKS (12)
   =================
   Frontend (3):
     • T-E01-F02-005: Build user profile component
     • T-E01-F02-007: Implement responsive navigation
     • T-E02-F01-003: Create task list UI

   Backend (5):
     • T-E01-F01-002: Implement JWT validation
     • T-E01-F03-001: Build API authentication layer
     • T-E02-F02-001: Database schema implementation
     • T-E02-F02-003: Task CRUD operations
     • T-E03-F01-001: Documentation API endpoints

   API (2):
     • T-E01-F01-005: Document authentication endpoints
     • T-E02-F01-002: Task management API specs

   Testing (2):
     • T-E01-F04-001: Auth integration tests
     • T-E02-F03-001: Task lifecycle tests
   ```

19. If no active tasks, display: "No tasks currently in progress"

**Blocked Tasks Section:**

20. The blocked tasks section must display tasks with status="blocked"

21. Each blocked task must show: key, title, and blocking reason

22. Blocked tasks must be color-coded red

23. Format:
   ```
   BLOCKED TASKS (5)
   =================
   • T-E01-F02-003: User authentication flow
     Reason: Waiting for API specification from backend team

   • T-E02-F01-007: Task dependency validation
     Reason: Missing dependency graph algorithm implementation

   • T-E03-F01-002: API documentation generation
     Reason: OpenAPI schema not finalized
   ```

24. If no blocked tasks, display: "No blocked tasks" (in green)

**Recent Completions Section:**

25. The recent completions section must display tasks with status="completed" updated in last 24 hours

26. Tasks must be sorted by completion time (most recent first)

27. Each task must show: key, title, completion timestamp (relative: "2 hours ago")

28. Format:
   ```
   RECENT COMPLETIONS (Last 24 hours)
   ===================================
   • T-E01-F01-003: JWT token generation - 2 hours ago
   • T-E01-F02-001: Login form component - 5 hours ago
   • T-E02-F01-001: Database connection setup - 18 hours ago
   • T-E02-F02-002: Task listing endpoint - 23 hours ago
   ```

29. If no recent completions, display: "No tasks completed in last 24 hours"

30. The system must support `--recent=<timeframe>` flag to customize window: `--recent=7d`, `--recent=48h`

**Epic-Filtered View:**

31. With `--epic=<epic-key>` flag, the dashboard must show only data for specified epic

32. Project summary must show epic-specific counts (features/tasks in this epic only)

33. Epic breakdown must show only the specified epic (single row)

34. Active/blocked/completed tasks must be filtered to specified epic

35. Dashboard title must indicate filter: "PROJECT STATUS - Epic E01"

**JSON Output:**

36. With `--json` flag, output must be valid JSON with structure:
    ```json
    {
      "summary": {
        "epics": {"total": 5, "active": 3, "completed": 2},
        "features": {"total": 23, "active": 15, "completed": 8},
        "tasks": {"total": 127, "todo": 45, "in_progress": 12, "ready_for_review": 5, "completed": 60, "blocked": 5},
        "overall_progress": 47.3
      },
      "epics": [
        {"key": "E01", "title": "Identity Platform", "progress": 60.0, "tasks": {"total": 50, "completed": 30}, "status": "active"},
        ...
      ],
      "active_tasks": {
        "frontend": [{"key": "T-E01-F02-005", "title": "Build user profile component"}, ...],
        "backend": [...]
      },
      "blocked_tasks": [
        {"key": "T-E01-F02-003", "title": "User authentication flow", "reason": "Waiting for API spec"},
        ...
      ],
      "recent_completions": [
        {"key": "T-E01-F01-003", "title": "JWT token generation", "completed_at": "2025-12-14T10:30:00Z", "completed_ago": "2 hours ago"},
        ...
      ]
    }
    ```

**Query Performance:**

37. The system must use efficient database queries with JOINs to minimize round trips

38. Epic progress must be calculated in single query (not N+1)

39. Active tasks must be fetched with single query grouped by agent_type

40. The command must complete in <500ms for projects with 100 epics

**Color Coding:**

41. The system must use Rich library for color and formatting

42. Color scheme:
    - Green: completed tasks, healthy epics (≥75% progress)
    - Yellow: in-progress tasks, warning epics (25-74% progress)
    - Red: blocked tasks, unhealthy epics (<25% progress or >3 blockers)
    - Blue: ready-for-review tasks
    - Gray: todo tasks

43. With `--no-color` flag, all color codes must be stripped

### Non-Functional Requirements

**Performance:**

- Dashboard rendering must complete in <500ms for 100 epics, 500 features, 1000 tasks
- Database queries must use efficient JOINs and aggregations
- Progress bar rendering must be instant (<10ms)
- JSON serialization must complete in <100ms

**Usability:**

- Dashboard must fit in 80-column terminal width (with wrapping for long titles)
- Progress bars must be visually consistent and aligned
- Color coding must be meaningful and accessible
- Empty sections must show helpful "no data" messages

**Accessibility:**

- Color must not be the only indicator (use symbols too: ✓, !, ⚠)
- --no-color mode must be fully functional
- Output must be readable in screen readers
- JSON output must be valid and parseable

**Reliability:**

- Missing data must not crash (show 0% progress, empty lists)
- Database errors must show user-friendly error message
- Large datasets must not cause performance degradation
- Terminal width detection must handle edge cases

## Acceptance Criteria

### Full Dashboard Display

**Given** a project with 3 epics, 15 features, 127 tasks
**When** I run `shark status`
**Then** dashboard displays all sections:
- Project Summary with correct counts
- Epic Breakdown with 3 epics and progress bars
- Active Tasks grouped by agent type
- Blocked Tasks with reasons
- Recent Completions from last 24 hours
**And** the output fits in terminal width

### Epic Progress Bars

**Given** Epic E01 has 60% progress
**When** dashboard is displayed
**Then** progress bar shows: `[############--------] 60%`
**And** progress bar is 20 characters wide
**And** 12 characters are filled (`#`), 8 are empty (`-`)

**Given** Epic E02 has 100% progress
**When** dashboard is displayed
**Then** progress bar shows: `[####################] 100%`

### Color Coding

**Given** Epic E01 has 80% progress (healthy)
**When** dashboard is displayed with color enabled
**Then** progress bar is green

**Given** Epic E02 has 15% progress (unhealthy)
**When** dashboard is displayed with color enabled
**Then** progress bar is red

**Given** Epic E03 has 5 blocked tasks
**When** dashboard is displayed
**Then** epic is marked red (unhealthy due to blockers)

**Given** I run `shark status --no-color`
**When** dashboard is displayed
**Then** no ANSI color codes are present
**And** output is plain text

### Active Tasks Grouping

**Given** 12 tasks are in_progress: 3 frontend, 5 backend, 2 api, 2 testing
**When** dashboard displays active tasks section
**Then** tasks are grouped by agent type
**And** each group shows agent type header
**And** group headers show: "Frontend (3):", "Backend (5):", etc.

### Blocked Tasks Display

**Given** 3 tasks are blocked with different reasons
**When** dashboard displays blocked tasks section
**Then** each task shows key, title, and blocking reason
**And** blocked tasks are color-coded red
**And** blocking reasons are clearly visible

**Given** no tasks are blocked
**When** dashboard displays blocked tasks section
**Then** message shows: "No blocked tasks" in green

### Recent Completions

**Given** 4 tasks were completed in last 24 hours
**When** dashboard displays recent completions section
**Then** all 4 tasks are shown
**And** completion times are relative ("2 hours ago", "18 hours ago")
**And** tasks are sorted by completion time (most recent first)

**Given** no tasks completed in last 24 hours
**When** dashboard displays recent completions section
**Then** message shows: "No tasks completed in last 24 hours"

### Epic-Filtered View

**Given** I run `shark status --epic=E01`
**When** dashboard is displayed
**Then** only Epic E01 data is shown
**And** project summary shows E01-specific counts
**And** epic breakdown shows only E01
**And** active/blocked/completed tasks are filtered to E01
**And** dashboard title shows: "PROJECT STATUS - Epic E01"

### JSON Output

**Given** I run `shark status --json`
**When** the command completes
**Then** output is valid JSON
**And** JSON structure matches documented schema
**And** I can parse with `jq '.summary.overall_progress'` to get 47.3
**And** all sections (summary, epics, active_tasks, blocked_tasks, recent_completions) are present

### Performance

**Given** project has 100 epics, 500 features, 2000 tasks
**When** I run `shark status`
**Then** dashboard renders in <500ms
**And** no database N+1 query problems occur

### Error Handling

**Given** database connection fails
**When** I run `shark status`
**Then** error message is displayed: "Error: Database connection failed"
**And** exit code is 2 (system error)

**Given** project has no epics
**When** I run `shark status`
**Then** dashboard shows: "No epics found. Create epics to get started."
**And** exit code is 0 (not an error)

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Historical Progress Tracking** - Showing progress over time (trend charts) is out of scope. Only current state is displayed.

2. **Gantt Charts** - Timeline visualization of tasks is too complex for CLI.

3. **Dependency Graphs** - Visualizing task dependencies is in E05-F02 (Dependency Management).

4. **Detailed Task History** - Full audit trail is in E05-F03 (History & Audit Trail).

5. **Custom Dashboard Layouts** - User-defined dashboard sections and ordering are out of scope.

6. **Real-Time Updates** - Dashboard is snapshot at run time, not live-updating.

7. **Email/Slack Reports** - Automated report distribution is out of scope (users can pipe JSON to external tools).

8. **Comparison Views** - Side-by-side comparison of multiple epics is out of scope.

9. **Interactive Dashboard** - No TUI (text user interface) with keyboard navigation. Output is static.

10. **Predictive Metrics** - Estimated completion dates based on velocity are in optional E05 features.
