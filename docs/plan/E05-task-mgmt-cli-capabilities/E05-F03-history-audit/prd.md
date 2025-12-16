# Feature: History & Audit Trail

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E05-task-mgmt-cli-capabilities/epic.md)

## Goal

### Problem

While E04-F01 creates task_history records for every status change, there's no way to query, analyze, or export this historical data. Developers need to understand how tasks progressed through their lifecycle ("Why was this task reopened?", "Who completed this work?", "How long did this task stay in review?"), but must directly query the database or read markdown files. Teams conducting retrospectives have no visibility into workflow patterns: which tasks get blocked frequently, average time in each status, or agent performance metrics. Without history commands, the audit trail data is write-only—captured but not accessible. Managers cannot answer stakeholder questions about who worked on what and when. Compliance requirements may need audit trails exported for record-keeping, but there's no export functionality.

### Solution

Implement comprehensive history query and export commands built on E04-F01's task_history table. Provide `pm task history <task-key>` to show complete lifecycle of a single task with all status transitions, timestamps, agents, and notes. Provide `pm history` for project-wide activity log showing recent changes across all tasks. Support filtering by agent (`--agent=<name>`), timeframe (`--since=7d`), task/epic/feature (`--epic=E01`), and status transitions (`--status-change=todo->in_progress`). Enable export to CSV and JSON formats for external analysis, stakeholder reporting, and compliance. Display human-readable timeline format in terminal and structured data for programmatic access. Build on E04-F02 (CLI Framework) for consistent output formatting and E04-F01 (Database) for efficient history queries.

### Impact

- **Retrospective Insights**: Enable data-driven retrospectives with concrete metrics on workflow patterns and bottlenecks
- **Accountability**: Clear audit trail of who changed what and when for compliance and transparency
- **Performance Analysis**: Understand task duration, time-in-status, and agent productivity through historical data
- **Compliance**: Export audit trails for record-keeping and regulatory requirements
- **Debugging**: Trace task lifecycle to understand why tasks are in unexpected states or were reopened

## User Personas

### Primary Persona: Product Manager / Technical Lead (Retrospectives)

**Role**: Team lead conducting sprint/project retrospectives
**Environment**: Terminal, analyzing completed work

**Key Characteristics**:
- Runs retrospectives weekly or biweekly
- Needs concrete data on workflow issues
- Identifies bottlenecks and improvement areas
- Reports team velocity and performance

**Goals**:
- See all status changes for completed tasks
- Understand how long tasks spent in each status
- Identify frequently blocked tasks
- Export data for retrospective discussions
- Track agent performance and workload

**Pain Points this Feature Addresses**:
- No visibility into task lifecycle timing
- Cannot identify workflow bottlenecks
- Manual data collection is time-consuming
- No concrete metrics for retrospectives
- Cannot track who worked on what

### Secondary Persona: Developer (Debugging)

**Role**: Developer investigating task issues
**Environment**: Terminal, troubleshooting unexpected task states

**Key Characteristics**:
- Needs to understand task state changes
- Investigates why tasks were reopened
- Traces ownership and timeline
- Debugs workflow issues

**Goals**:
- See complete history of single task
- Understand who made changes and why
- Find when status changed unexpectedly
- See notes from previous status changes

**Pain Points this Feature Addresses**:
- Cannot see why task is in current state
- No way to trace ownership changes
- Missing context on status changes
- Cannot see reopen reasons

### Tertiary Persona: Manager (Compliance & Reporting)

**Role**: Manager providing audit trails and stakeholder reports
**Environment**: Terminal, generating reports for external stakeholders

**Key Characteristics**:
- Needs audit trails for compliance
- Reports to stakeholders regularly
- Requires exportable data formats
- Tracks work allocation and completion

**Goals**:
- Export history to CSV for spreadsheets
- Generate monthly activity reports
- Provide audit trails for compliance
- Track completion rates and velocity

**Pain Points this Feature Addresses**:
- No export functionality
- Cannot generate compliance reports
- Manual activity tracking is tedious
- No shareable data formats

## User Stories

### Must-Have User Stories

**Story 1: View Task History**
- As a developer, I want to run `pm task history T-E01-F02-003`, so that I see complete lifecycle of that task with all status changes, timestamps, and agents.

**Story 2: View Project Activity Log**
- As a manager, I want to run `pm history`, so that I see recent activity across all tasks (last 50 events by default).

**Story 3: Filter History by Agent**
- As a manager, I want to run `pm history --agent=backend-agent`, so that I see all changes made by specific agent.

**Story 4: Filter History by Timeframe**
- As a developer, I want to run `pm history --since=7d`, so that I see activity from last 7 days.

**Story 5: Export History to CSV**
- As a manager, I want to run `pm history --format=csv > activity.csv`, so that I can analyze data in spreadsheets.

**Story 6: Export History to JSON**
- As a developer, I want to run `pm history --format=json`, so that I can process data programmatically.

**Story 7: Show History with Notes**
- As a developer, I want status change notes to be displayed in history (e.g., "Reopened: Add error handling"), so that I understand context of changes.

**Story 8: Timeline Format**
- As a user, I want history displayed as chronological timeline with relative timestamps ("2 hours ago"), so that I can quickly understand recency.

### Should-Have User Stories

**Story 9: Filter by Epic/Feature**
- As a manager, I want to run `pm history --epic=E01`, so that I see activity for specific epic only.

**Story 10: Filter by Status Transition**
- As a developer, I want to run `pm history --status-change=ready_for_review->in_progress`, so that I find all task reopens.

**Story 11: Show Time in Status**
- As a manager, I want task history to calculate time spent in each status, so that I can identify slow stages in workflow.

**Story 12: Pagination**
- As a user, I want `pm history --limit=100 --offset=50` to paginate large result sets, so that I can browse historical data efficiently.

### Could-Have User Stories

**Story 13: Aggregate Statistics**
- As a manager, I want `pm history --stats` to show summary statistics (total changes, most active agents, average time-in-status), so that I get quick insights.

**Story 14: Compare Time Periods**
- As a manager, I want to compare activity between time periods (this week vs last week), so that I can track velocity trends.

## Requirements

### Functional Requirements

**Task History Command (pm task history):**

1. The system must provide `pm task history <task-key>` command that displays complete lifecycle of single task

2. The command must query task_history table for all records with matching task_id

3. History must be displayed in chronological order (oldest first)

4. Each history entry must show:
   - Timestamp (absolute and relative: "2025-12-14 10:30:00 (2 hours ago)")
   - Status transition (e.g., "todo → in_progress")
   - Agent who made change
   - Notes (if any)

5. Format:
   ```
   HISTORY: T-E01-F02-003 - Build authentication middleware
   =========================================================

   2025-12-10 09:15:00 (4 days ago)
   • Created by: project-manager
   • Status: todo
   • Notes: Task created

   2025-12-11 14:30:00 (3 days ago)
   • Changed by: backend-agent
   • Status: todo → in_progress
   • Notes: Starting implementation

   2025-12-12 16:45:00 (2 days ago)
   • Changed by: backend-agent
   • Status: in_progress → ready_for_review
   • Notes: Implemented JWT validation

   2025-12-13 10:00:00 (1 day ago)
   • Changed by: code-reviewer
   • Status: ready_for_review → in_progress
   • Notes: Reopened: Add error handling for expired tokens

   2025-12-14 11:00:00 (1 hour ago)
   • Changed by: backend-agent
   • Status: in_progress → ready_for_review
   • Notes: Added error handling
   ```

6. If task has no history (shouldn't happen), show: "No history found for task"

7. The command must calculate time spent in each status and display summary:
   ```
   Time in Status:
   • todo: 1 day, 5 hours
   • in_progress: 1 day, 8 hours
   • ready_for_review: Currently (1 hour so far)
   ```

**Project Activity Log (pm history):**

8. The system must provide `pm history` command that displays recent activity across all tasks

9. The command must default to showing last 50 history records (configurable with --limit)

10. History must be displayed in reverse chronological order (most recent first)

11. Each entry must show: timestamp, task key, status transition, agent

12. Format:
    ```
    PROJECT ACTIVITY LOG (Last 50 events)
    ======================================

    2 hours ago | T-E01-F02-003 | in_progress → ready_for_review | backend-agent
    5 hours ago | T-E01-F01-005 | todo → in_progress | api-agent
    8 hours ago | T-E02-F01-001 | ready_for_review → completed | code-reviewer
    1 day ago   | T-E01-F02-001 | in_progress → blocked | frontend-agent
                  Reason: Waiting for API specification
    ```

13. Entries with notes must display notes indented below main line

**Filtering:**

14. The system must support `--agent=<agent-name>` to filter history by agent who made changes

15. The system must support `--since=<timeframe>` to filter by time (formats: `7d`, `48h`, `2w`, `YYYY-MM-DD`)

16. The system must support `--until=<timeframe>` to filter by end date

17. The system must support `--epic=<epic-key>` to filter to tasks in specific epic

18. The system must support `--feature=<feature-key>` to filter to tasks in specific feature

19. The system must support `--task=<task-key>` to filter to specific task (alias for `pm task history`)

20. The system must support `--status-change=<old>-><new>` to filter by specific transitions (e.g., `--status-change=ready_for_review->in_progress` for reopens)

21. Multiple filters must combine with AND logic

22. Example: `pm history --agent=backend-agent --since=7d --epic=E01` shows backend-agent changes in Epic E01 from last 7 days

**Pagination:**

23. The system must support `--limit=<N>` to limit number of results (default: 50)

24. The system must support `--offset=<N>` to skip first N results (for pagination)

25. Output must show pagination info: "Showing 51-100 of 347 total events"

**Export Formats:**

26. The system must support `--format=csv` to export history as CSV

27. CSV format must include columns: timestamp, task_key, task_title, old_status, new_status, agent, notes

28. CSV must follow RFC 4180 standard (quoted strings, escaped commas)

29. CSV header row must be included

30. Example CSV:
    ```csv
    timestamp,task_key,task_title,old_status,new_status,agent,notes
    2025-12-14T10:30:00Z,T-E01-F02-003,Build auth middleware,in_progress,ready_for_review,backend-agent,Added error handling
    2025-12-13T14:00:00Z,T-E01-F01-005,API docs,todo,in_progress,api-agent,Starting documentation
    ```

31. The system must support `--format=json` to export history as JSON array

32. JSON format must include full history records:
    ```json
    [
      {
        "timestamp": "2025-12-14T10:30:00Z",
        "task_key": "T-E01-F02-003",
        "task_title": "Build authentication middleware",
        "old_status": "in_progress",
        "new_status": "ready_for_review",
        "agent": "backend-agent",
        "notes": "Added error handling",
        "time_ago": "2 hours ago"
      },
      ...
    ]
    ```

33. Default format (no --format flag) is human-readable terminal output

**Time Calculations:**

34. The system must calculate time spent in each status for task history

35. Calculation: time_in_status = (next_transition_timestamp - this_transition_timestamp)

36. For current status (no next transition), calculate from last transition to now

37. Display format: "X days, Y hours" or "X hours, Y minutes" or "X minutes" depending on duration

38. Total time from creation to current state must be calculated and displayed

**Relative Timestamps:**

39. All timestamps must show both absolute and relative time

40. Absolute format: "2025-12-14 10:30:00" (local timezone)

41. Relative format: "2 hours ago", "3 days ago", "just now" (<1 minute)

42. Use humanize library or similar for relative time formatting

**Query Performance:**

43. History queries must use database indexes on task_id, timestamp, agent

44. Pagination must use LIMIT and OFFSET for efficiency

45. Filtering must use WHERE clauses (not in-memory filtering)

46. Queries must complete in <200ms for datasets with 10,000 history records

**Integration:**

47. All status-changing operations in E04-F03 must create history records (already required in E04-F01)

48. History records must be created in same transaction as status changes (atomicity)

49. History table must be append-only (no updates or deletes)

### Non-Functional Requirements

**Performance:**

- `pm task history` must complete in <100ms for tasks with 50 history records
- `pm history` must complete in <200ms for 50 most recent events
- CSV export of 1,000 records must complete in <2 seconds
- JSON export of 1,000 records must complete in <1 second

**Usability:**

- Timeline format must be easy to scan and read
- Relative timestamps must be intuitive ("2 hours ago" not "7200 seconds ago")
- Color coding must highlight important transitions (green=completed, red=blocked, yellow=reopened)
- CSV must be importable into Excel, Google Sheets without issues

**Data Integrity:**

- History records must be immutable (never modified after creation)
- Timestamps must be in UTC in database, local timezone in display
- All history records must have valid task_id (foreign key enforced)
- Export formats must be valid (parseable CSV/JSON)

**Reliability:**

- Missing history data must not crash (show "No history found")
- Invalid date formats must show clear error
- Large result sets must not cause memory issues (use pagination)
- Export failures must show clear error messages

## Acceptance Criteria

### Task History Display

**Given** task T-E01-F02-003 has 5 history records
**When** I run `pm task history T-E01-F02-003`
**Then** all 5 records are displayed in chronological order
**And** each record shows timestamp, status transition, agent, and notes
**And** time spent in each status is calculated and displayed
**And** output is formatted as readable timeline

### Project Activity Log

**Given** project has 100 recent history records
**When** I run `pm history`
**Then** last 50 records are displayed (default limit)
**And** records are in reverse chronological order (most recent first)
**And** pagination info shows "Showing 1-50 of 100 total events"

**Given** I run `pm history --limit=10`
**When** the command completes
**Then** only 10 most recent records are shown

### Agent Filtering

**Given** project history includes changes by multiple agents
**When** I run `pm history --agent=backend-agent`
**Then** only records with agent="backend-agent" are shown
**And** other agents' changes are excluded

### Timeframe Filtering

**Given** project has history records spanning 30 days
**When** I run `pm history --since=7d`
**Then** only records from last 7 days are shown

**Given** I run `pm history --since=2025-12-01 --until=2025-12-07`
**When** the command completes
**Then** only records between Dec 1-7 are shown

### Status Transition Filtering

**Given** project history includes various status transitions
**When** I run `pm history --status-change=ready_for_review->in_progress`
**Then** only records showing reopens (ready_for_review → in_progress) are displayed

### CSV Export

**Given** I run `pm history --format=csv`
**When** the command completes
**Then** output is valid CSV with header row
**And** each history record is a CSV row
**And** I can import CSV into Excel without errors
**And** special characters in notes are properly escaped

### JSON Export

**Given** I run `pm history --format=json`
**When** the command completes
**Then** output is valid JSON array
**And** each element is a history record object
**And** I can parse with `jq '.[0].task_key'`

### Time Calculation

**Given** task was in status "in_progress" from Dec 10 10:00 to Dec 12 14:00
**When** I view task history
**Then** time in status shows "2 days, 4 hours"

**Given** task is currently in status "ready_for_review" for 3 hours
**When** I view task history
**Then** time in current status shows "Currently (3 hours so far)"

### Relative Timestamps

**Given** history record from 2 hours ago
**When** displayed
**Then** timestamp shows "2 hours ago"

**Given** history record from 45 seconds ago
**When** displayed
**Then** timestamp shows "just now"

### Epic/Feature Filtering

**Given** I run `pm history --epic=E01`
**When** the command completes
**Then** only history for tasks in Epic E01 is shown

### Pagination

**Given** I run `pm history --limit=50 --offset=50`
**When** the command completes
**Then** records 51-100 are displayed
**And** pagination info shows "Showing 51-100 of <total>"

### Color Coding

**Given** history includes completed and blocked transitions
**When** displayed in terminal with color enabled
**Then** completed transitions are green
**And** blocked transitions are red
**And** reopened transitions are yellow

**Given** I run `pm history --no-color`
**When** displayed
**Then** no ANSI color codes are present

### Performance

**Given** task has 50 history records
**When** I run `pm task history <key>`
**Then** output is displayed in <100ms

**Given** I export 1,000 history records to CSV
**When** I run `pm history --format=csv --limit=1000`
**Then** export completes in <2 seconds

### Error Handling

**Given** I run `pm task history T-E99-F99-999` (non-existent task)
**When** the command executes
**Then** error is displayed: "Error: Task T-E99-F99-999 does not exist"
**And** exit code is 1

**Given** I run `pm history --since=invalid-date`
**When** the command executes
**Then** error is displayed: "Error: Invalid date format. Use: YYYY-MM-DD, 7d, 48h, or 2w"
**And** exit code is 1

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Real-Time Activity Feed** - Live-updating history stream is out of scope. History is snapshot at query time.

2. **History Editing** - No ability to modify or delete history records. Audit trail is immutable.

3. **Advanced Analytics** - Velocity charts, burndown graphs, statistical analysis are out of scope (basic time calculations only).

4. **History Comparison** - Comparing history between tasks or time periods.

5. **Notification on Changes** - No alerts or notifications when history records are created.

6. **History Retention Policies** - No automatic archival or deletion of old history records.

7. **Detailed Agent Profiles** - No agent performance dashboards or detailed agent analytics.

8. **History Rollback** - No ability to revert tasks to previous states based on history.

9. **External System Integration** - No automatic sending of history to Slack, email, or external logging systems.

10. **Compliance Reporting Templates** - Pre-built compliance report formats (SOC2, HIPAA) are out of scope (users export raw data and format externally).
