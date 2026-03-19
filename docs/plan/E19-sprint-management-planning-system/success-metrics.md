# Success Metrics

**Epic**: [Sprint Management & Planning System](./epic.md)

---

## Overview

This document defines the Key Performance Indicators (KPIs) that will measure the success of the Sprint Management & Planning System.

**Measurement Timeline**: Initial adoption metrics at 2 sprints post-launch; velocity accuracy and planning efficiency metrics at 5+ sprints post-launch (minimum data threshold for meaningful trend analysis).

---

## Primary Success Metrics

### Metric 1: Sprint Feature Adoption

**Type**: Leading

**What We're Measuring**:
Whether teams actually use sprint management commands in their workflow rather than continuing to manage iterations informally.

**How We'll Measure**:
- **Data Source**: Shark command usage (sprint commands executed per sprint cycle)
- **Calculation Method**: Count distinct sprint lifecycle events (create, start, close) per 2-week period
- **Measurement Frequency**: Per sprint cycle (every 1-4 weeks depending on team cadence)

**Success Criteria**:
- **Baseline**: 0 (feature does not exist today)
- **Target**: At least 1 complete sprint lifecycle (create -> plan -> start -> close) per iteration cycle within 2 sprints of launch
- **Timeline**: 2 sprints post-launch
- **Minimum Viable**: Sprint create and close commands used at least once per iteration

**Relates To**:
- **Requirement(s)**: REQ-F-001, REQ-F-002, REQ-F-006
- **User Journey**: [Sprint Creation](./user-journeys.md#journey-1-sprint-creation--configuration), [Sprint Close](./user-journeys.md#journey-4-sprint-close--retrospective)
- **Business Value**: Validates that the feature fills a real workflow gap

---

### Metric 2: Sprint Planning Efficiency

**Type**: Lagging

**What We're Measuring**:
Whether the sprint planning view and capacity tools reduce the time and effort required to scope a sprint compared to the pre-sprint (manual scanning) baseline.

**How We'll Measure**:
- **Data Source**: Time from sprint creation to sprint start (measured from database timestamps)
- **Calculation Method**: `sprint.start_date_actual - sprint.created_at` for each sprint, averaged over trailing 3 sprints
- **Measurement Frequency**: Per sprint cycle

**Success Criteria**:
- **Baseline**: Sprint scoping currently takes an undefined amount of time because there is no structured process; estimate 30-60 minutes of manual backlog scanning and mental bookkeeping per iteration
- **Target**: Sprint transitions from `planning` to `active` within the same planning session (under 30 minutes of wall-clock time between sprint create and sprint start, assuming backlog is pre-prioritized)
- **Timeline**: 3 sprints post-launch
- **Minimum Viable**: Sprint planning does not take longer than the informal process it replaces

**Relates To**:
- **Requirement(s)**: REQ-F-011, REQ-F-012, REQ-F-013, REQ-F-014
- **User Journey**: [Sprint Planning](./user-journeys.md#journey-2-sprint-planning)
- **Business Value**: Reduced planning overhead means more time for execution

---

### Metric 3: Velocity Prediction Accuracy

**Type**: Lagging

**What We're Measuring**:
How accurately historical velocity data predicts actual sprint completion, validating that the velocity calculation is useful for planning.

**How We'll Measure**:
- **Data Source**: `shark sprint velocity --json` output and `shark sprint summary --json` output
- **Calculation Method**: `|predicted_velocity - actual_velocity| / predicted_velocity * 100` where predicted velocity is the trailing 3-sprint average at planning time and actual velocity is completed tasks/points at sprint close
- **Measurement Frequency**: Per sprint close

**Success Criteria**:
- **Baseline**: No velocity data exists (cannot predict)
- **Target**: Prediction variance under 15% after 5 completed sprints (trailing average stabilizes)
- **Timeline**: 5 sprints post-launch
- **Minimum Viable**: Prediction variance under 25% after 5 sprints

**Relates To**:
- **Requirement(s)**: REQ-F-007
- **User Journey**: [Sprint Monitoring](./user-journeys.md#journey-3-sprint-monitoring), Step 4
- **Business Value**: Accurate velocity enables realistic sprint scoping, reducing both overcommitment (burnout, missed deadlines) and underutilization (wasted capacity)

---

### Metric 4: Sprint Completion Rate

**Type**: Lagging

**What We're Measuring**:
The percentage of planned sprint scope that is completed by sprint close, indicating whether sprint planning produces achievable commitments.

**How We'll Measure**:
- **Data Source**: `shark sprint summary --json` output
- **Calculation Method**: `completed_tasks / (completed_tasks + carryover_tasks) * 100` at sprint close (excludes tasks removed mid-sprint from the denominator)
- **Measurement Frequency**: Per sprint close

**Success Criteria**:
- **Baseline**: No sprint-scoped data exists
- **Target**: Average completion rate above 80% across trailing 3 sprints (indicating realistic planning)
- **Timeline**: 5 sprints post-launch
- **Minimum Viable**: Average completion rate above 60% (team is learning to scope realistically)

**Relates To**:
- **Requirement(s)**: REQ-F-006, REQ-F-009
- **User Journey**: [Sprint Close & Retrospective](./user-journeys.md#journey-4-sprint-close--retrospective)
- **Business Value**: High completion rates indicate effective planning; low rates signal planning process issues that need attention

---

## Secondary Success Metrics

### Metric 5: Agent Capacity Utilization

**Type**: Leading

**What We're Measuring**:
Whether agent capacity tracking leads to balanced workload distribution across agent types.

**How We'll Measure**:
- **Data Source**: `shark sprint capacity show --json` output at sprint close
- **Calculation Method**: `allocated_points / capacity_points * 100` per agent type; standard deviation across agent types measures balance
- **Measurement Frequency**: Per sprint close

**Success Criteria**:
- **Target**: No agent type exceeds 110% capacity at sprint start; standard deviation of utilization across agent types is under 20 percentage points
- **Minimum Viable**: Capacity data is populated and visible for at least 2 agent types per sprint

**Relates To**:
- **Requirement(s)**: REQ-F-014, REQ-F-015
- **Business Value**: Balanced workload prevents bottlenecks and improves overall throughput

---

### Metric 6: AI Orchestrator Sprint Automation

**Type**: Leading

**What We're Measuring**:
Whether the AI orchestrator can successfully use sprint commands programmatically to auto-plan sprints.

**How We'll Measure**:
- **Data Source**: Orchestrator logs showing successful `shark sprint` command sequences with `--json` output
- **Calculation Method**: Count of sprints where the orchestrator successfully executed the full plan sequence (create -> add tasks -> readiness check -> report to PM)
- **Measurement Frequency**: Per sprint cycle

**Success Criteria**:
- **Target**: Orchestrator can auto-plan a sprint (select tasks within capacity, produce readiness report) without error in 90% of attempts
- **Minimum Viable**: Orchestrator can execute individual sprint commands (`--json` output works, commands return expected data shapes)

**Relates To**:
- **Requirement(s)**: REQ-NF-005, REQ-F-003, REQ-F-004
- **Business Value**: Autonomous sprint planning reduces PM workload and enables faster iteration cadence

---

## Success Criteria Summary

The epic is considered **successful** if:

1. At least 1 complete sprint lifecycle is executed per iteration cycle within 2 sprints of launch (adoption)
2. Velocity prediction variance is under 15% after 5 completed sprints (accuracy)
3. Average sprint completion rate exceeds 80% across trailing 3 sprints (planning quality)
4. No regressions in existing command performance or functionality (backward compatibility)

---

*See also*: [Requirements](./requirements.md)
