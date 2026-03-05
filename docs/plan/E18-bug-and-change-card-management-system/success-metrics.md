# Success Metrics

**Epic**: [Bug and Change-Card Management System](./epic.md)

---

## Overview

This document defines the Key Performance Indicators (KPIs) that will measure the success of the Bug and Change-Card Management System.

**Measurement Timeline**: Initial metrics at 2 weeks post-launch (adoption and basic usage), full evaluation at 8 weeks (workflow efficiency and quality impact).

---

## Primary Success Metrics

### Metric 1: Bug Tracking Adoption Rate

**Type**: Leading

**What We're Measuring**:
Percentage of bugs tracked in Shark versus external tools or informal channels. This measures whether the system is actually being used as the single source of truth for defects.

**How We'll Measure**:
- **Data Source**: Shark database (`shark analytics --type=bug`)
- **Calculation Method**: Count of bugs created per week via `shark bug create`; compare with team's historical bug volume from prior tools
- **Measurement Frequency**: Weekly

**Success Criteria**:
- **Baseline**: 0 bugs tracked in Shark (no bug entity type exists today)
- **Target**: 90% of all bugs reported through `shark bug create` within 8 weeks of launch
- **Timeline**: 8 weeks post-launch
- **Minimum Viable**: 50% of bugs tracked in Shark within 4 weeks

**Relates To**:
- **Requirement(s)**: REQ-F-001, REQ-F-006
- **User Journey**: Journey 1 (Bug Lifecycle)
- **Business Value**: Eliminates external tool dependency; establishes single source of truth

---

### Metric 2: Average Bug Resolution Time

**Type**: Lagging

**What We're Measuring**:
The elapsed time from bug creation (`reported` status) to resolution (`resolved`, `wont_fix`, or `duplicate` status). This measures whether the dedicated workflow accelerates defect resolution.

**How We'll Measure**:
- **Data Source**: Shark database, status history timestamps
- **Calculation Method**: Average of (resolution_timestamp - creation_timestamp) for all bugs resolved in the measurement period
- **Measurement Frequency**: Weekly aggregate

**Success Criteria**:
- **Baseline**: No measurable baseline (bugs not tracked in Shark today). Use first 2 weeks of data as baseline.
- **Target**: 20% reduction in average resolution time by week 8 compared to week 2 baseline
- **Timeline**: 8 weeks post-launch (comparing week 8 to week 2)
- **Minimum Viable**: Resolution time does not increase compared to baseline (no regression)

**Relates To**:
- **Requirement(s)**: REQ-F-004, REQ-F-005, REQ-F-014
- **User Journey**: Journey 1 (Bug Lifecycle, Steps 1-5)
- **Business Value**: Faster defect resolution improves software quality and user satisfaction

---

### Metric 3: Change-Card Throughput

**Type**: Lagging

**What We're Measuring**:
The number of change-cards that move from `proposed` to `completed` per month. This measures whether the lightweight enhancement path is being used and delivering value.

**How We'll Measure**:
- **Data Source**: Shark database (`shark analytics --type=change`)
- **Calculation Method**: Count of change-cards reaching `completed` status per calendar month
- **Measurement Frequency**: Monthly

**Success Criteria**:
- **Baseline**: 0 change-cards (entity type does not exist today)
- **Target**: 5 or more change-cards completed per month by month 2
- **Timeline**: 8 weeks post-launch
- **Minimum Viable**: 2 or more change-cards completed per month

**Relates To**:
- **Requirement(s)**: REQ-F-007, REQ-F-009, REQ-F-011
- **User Journey**: Journey 2 (Change-Card Lifecycle)
- **Business Value**: Lightweight path for improvements that would otherwise go untracked

---

## Secondary Success Metrics

### Metric 4: Bug Creation Speed

**Type**: Leading

**What We're Measuring**:
The time it takes a developer to create a bug report using `shark bug create`. This measures whether the CLI experience is fast enough to not interrupt workflow.

**How We'll Measure**:
- **Data Source**: Manual testing, shell timing (`time shark bug create "..." --severity=high --link=E07-F01`)
- **Calculation Method**: Wall clock time from command invocation to completion
- **Measurement Frequency**: At launch, then quarterly

**Success Criteria**:
- **Baseline**: N/A (new capability)
- **Target**: Under 500ms on local SQLite, under 2 seconds on Turso cloud
- **Timeline**: At launch
- **Minimum Viable**: Under 1 second local, under 3 seconds cloud

**Relates To**:
- **Requirement(s)**: REQ-NF-001
- **User Journey**: Journey 1, Step 1
- **Business Value**: Fast creation encourages bug reporting; slow creation discourages it

---

### Metric 5: Dashboard Visibility Score

**Type**: Leading

**What We're Measuring**:
Whether bugs and change-cards appear in the standard `shark status` dashboard without additional commands. Binary: either the information is present or it is not.

**How We'll Measure**:
- **Data Source**: Manual verification of `shark status` output
- **Calculation Method**: Checklist verification: (1) bug counts by status visible, (2) bug severity breakdown visible, (3) change-card counts by status visible
- **Measurement Frequency**: At launch

**Success Criteria**:
- **Baseline**: No bug or change-card information in dashboard (zero visibility)
- **Target**: All three checklist items pass
- **Timeline**: At launch
- **Minimum Viable**: Bug counts visible (items 1 and 2)

**Relates To**:
- **Requirement(s)**: REQ-F-013
- **User Journey**: Journey 3, Step 2
- **Business Value**: Product owners need a single view of all work types

---

## Success Criteria Summary

The epic is considered **successful** if:

1. 90% or more of bugs are tracked in Shark within 8 weeks of launch (Metric 1)
2. Average bug resolution time does not increase versus the 2-week baseline, with a target of 20% reduction by week 8 (Metric 2)
3. 5 or more change-cards are completed per month by month 2 (Metric 3)
4. Bug creation completes in under 500ms on local SQLite (Metric 4)
5. The `shark status` dashboard displays bug and change-card information without additional commands (Metric 5)

---

*See also*: [Requirements](./requirements.md)
