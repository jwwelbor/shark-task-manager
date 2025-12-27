# Success Metrics

**Epic**: [Advanced Task Intelligence & Context Management](./epic.md)

---

## Overview

This document defines the Key Performance Indicators (KPIs) that will measure the success of this epic.

**Measurement Timeline**: Initial metrics at 2 weeks post-Phase 1 launch, full evaluation at 3 months post-Phase 3 completion

---

## Primary Success Metrics

### Metric 1: Task Resume Success Rate

**Type**: Leading

**What We're Measuring**:
Percentage of paused tasks that AI agents successfully resume without human intervention or codebase re-analysis.

**How We'll Measure**:
- **Data Source**: Work sessions table + task_notes table
- **Calculation Method**:
  ```sql
  -- Count tasks with 2+ sessions where second session includes resume command
  -- AND no human intervention notes between sessions
  SELECT COUNT(*) FROM tasks WHERE
    (SELECT COUNT(*) FROM work_sessions WHERE task_id = tasks.id) >= 2
    AND NOT EXISTS (
      SELECT 1 FROM task_notes WHERE task_id = tasks.id
      AND note_type = 'question'
      AND created_at BETWEEN session1.ended_at AND session2.started_at
    )
  ```
- **Measurement Frequency**: Weekly analysis

**Success Criteria**:
- **Baseline**: ~40% (current state - most resumes require codebase re-reading)
- **Target**: 90% of paused tasks resume successfully with context alone
- **Timeline**: 3 months post-Phase 1 completion
- **Minimum Viable**: 75% resume success rate

**Relates To**:
- **Requirement(s)**: REQ-F-009 (Resume Command), REQ-F-008 (Structured Context)
- **User Journey**: Journey 2 (Resume Paused Task)
- **Business Value**: Core value prop - AI agents work more efficiently across sessions

---

### Metric 2: Completion Metadata Coverage

**Type**: Lagging

**What We're Measuring**:
Percentage of completed tasks that have completion metadata captured (files modified, verification status, etc.).

**How We'll Measure**:
- **Data Source**: tasks.completion_metadata column
- **Calculation Method**:
  ```sql
  SELECT
    COUNT(*) FILTER (WHERE completion_metadata IS NOT NULL AND completion_metadata != '{}') * 100.0 /
    COUNT(*)
  FROM tasks
  WHERE status = 'completed'
  AND completed_at > '2025-12-26'  -- Post-E10 launch
  ```
- **Measurement Frequency**: Daily dashboard

**Success Criteria**:
- **Baseline**: 0% (feature doesn't exist yet)
- **Target**: 100% of completed tasks have completion metadata
- **Timeline**: 1 month post-Phase 1 completion
- **Minimum Viable**: 85% coverage

**Relates To**:
- **Requirement(s)**: REQ-F-005 (Completion Metadata Capture)
- **User Journey**: Journey 3 (Tech Lead Reviews Task)
- **Business Value**: Quality assurance and knowledge capture

---

### Metric 3: Dependency-Related Blocker Reduction

**Type**: Lagging

**What We're Measuring**:
Reduction in task blockers caused by unknown dependencies (measured by blocker notes mentioning "didn't know" or "discovered dependency").

**How We'll Measure**:
- **Data Source**: task_notes table (note_type = 'blocker')
- **Calculation Method**:
  ```sql
  -- Compare blocker frequency before/after relationship features
  SELECT
    COUNT(*) FILTER (WHERE content LIKE '%didn''t know%' OR content LIKE '%discovered dependency%')
  FROM task_notes
  WHERE note_type = 'blocker'
  AND created_at BETWEEN date1 AND date2
  ```
- **Measurement Frequency**: Monthly comparison (pre-E10 vs. post-E10)

**Success Criteria**:
- **Baseline**: ~20% of blockers are dependency-discovery related (pre-E10)
- **Target**: <10% of blockers (50% reduction)
- **Timeline**: 3 months post-Phase 2 completion
- **Minimum Viable**: <15% of blockers (25% reduction)

**Relates To**:
- **Requirement(s)**: REQ-F-010 (Bidirectional Relationships), REQ-F-011 (Relationship Commands)
- **User Journey**: Journey 1 Alt Path A (Blocked by Dependency)
- **Business Value**: Reduce wasted time and context switching

---

### Metric 4: Tech Lead Review Time

**Type**: Leading

**What We're Measuring**:
Average time from task entering `ready_for_review` status to being approved or reopened.

**How We'll Measure**:
- **Data Source**: task_history table (status transitions)
- **Calculation Method**:
  ```sql
  SELECT AVG(
    EXTRACT(EPOCH FROM (approved_at - ready_for_review_at)) / 3600
  ) AS avg_hours
  FROM (
    SELECT
      task_id,
      MIN(changed_at) FILTER (WHERE new_status = 'ready_for_review') AS ready_for_review_at,
      MIN(changed_at) FILTER (WHERE new_status = 'completed') AS approved_at
    FROM task_history
    GROUP BY task_id
  )
  ```
- **Measurement Frequency**: Weekly rolling average

**Success Criteria**:
- **Baseline**: ~30 minutes average (current state)
- **Target**: <10 minutes average (67% reduction)
- **Timeline**: 2 months post-Phase 1 completion
- **Minimum Viable**: <15 minutes average (50% reduction)

**Relates To**:
- **Requirement(s)**: REQ-F-006 (Completion Details), REQ-F-014 (Criteria Progress)
- **User Journey**: Journey 3 (Tech Lead Reviews Task)
- **Business Value**: Faster review cycles, higher throughput

---

### Metric 5: Search Utilization Rate

**Type**: Leading

**What We're Measuring**:
Number of `shark search` and `shark task find` commands executed per week as indicator of knowledge discovery adoption.

**How We'll Measure**:
- **Data Source**: CLI usage logs (if implemented) or manual survey
- **Calculation Method**: Count of search command executions / active users / week
- **Measurement Frequency**: Weekly

**Success Criteria**:
- **Baseline**: 0 (feature doesn't exist)
- **Target**: 5+ searches per active user per week
- **Timeline**: 3 months post-Phase 2 completion
- **Minimum Viable**: 2 searches per active user per week

**Relates To**:
- **Requirement(s)**: REQ-F-016 (Full-Text Search), REQ-F-007 (File-Based Discovery)
- **User Journey**: Journey 5 (Discover Related Implementation)
- **Business Value**: Knowledge reuse and learning from past implementations

---

## Secondary Metrics

### Metric 6: Note Adoption Rate

**What We're Measuring**: Average notes per task

**Target**: 3+ notes per task (indicates agents are recording decisions/solutions)

**Measurement**: `SELECT AVG(note_count) FROM (SELECT COUNT(*) as note_count FROM task_notes GROUP BY task_id)`

---

### Metric 7: Acceptance Criteria Completion Rate

**What We're Measuring**: Percentage of tasks with all criteria marked complete before approval

**Target**: 95% of approved tasks have 100% criteria complete

**Measurement**: Join task_criteria with tasks where status = 'completed'

---

### Metric 8: Relationship Graph Depth

**What We're Measuring**: Average number of related tasks per task

**Target**: 2+ relationships per task (indicates dependency awareness)

**Measurement**: `SELECT AVG(relationship_count) FROM (SELECT COUNT(*) as relationship_count FROM task_relationships GROUP BY from_task_id)`

---

## Success Criteria Summary

The epic is considered **successful** if:

1. **90%+ task resume success rate** (Metric 1 meets target)
2. **100% completion metadata coverage** (Metric 2 meets target)
3. **50% reduction in dependency-related blockers** (Metric 3 meets target)
4. **<10 minute average review time** (Metric 4 meets target)
5. **No performance regressions** (all search/retrieval operations within SLA per REQ-NF-001/002/003)

The epic is considered **minimally viable** if metrics 1, 2, and 4 meet their minimum viable thresholds.

---

## Measurement Infrastructure

### Required Instrumentation

To measure these metrics, we need:

1. **Work Session Tracking**: Implemented in Phase 3 (REQ-F-018)
2. **CLI Usage Logging**: Optional enhancement to track command usage (out of scope for E10, future epic)
3. **Database Query Scripts**: SQL queries for metrics 1-8 (to be created during implementation)
4. **Dashboard/Report**: Weekly automated report via `shark analytics` command (stretch goal)

### Baseline Data Collection

Before launching Phase 1, collect baseline data:
- Current task completion patterns (tasks.created_at, tasks.completed_at)
- Current blocker frequency (task_history.notes containing blocker mentions)
- Current review time (time between status changes)

---

*See also*: [Requirements](./requirements.md), [User Journeys](./user-journeys.md)
