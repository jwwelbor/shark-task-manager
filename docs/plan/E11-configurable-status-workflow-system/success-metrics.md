# Success Metrics

**Epic**: [Configurable Status Workflow System](./epic.md)

---

## Overview

This document defines the Key Performance Indicators (KPIs) that will measure the success of the configurable status workflow system.

**Measurement Timeline**: Initial metrics at 2 weeks post-launch, comprehensive evaluation at 3 months

---

## Primary Success Metrics

### Metric 1: Adoption Rate

**Type**: Leading

**What We're Measuring**:
Percentage of new Shark projects that define custom workflow configuration within 30 days of project creation

**How We'll Measure**:
- **Data Source**: Shark database (count projects with custom `status_flow` in `.sharkconfig.json`)
- **Calculation Method**: `(projects_with_custom_workflow / total_new_projects) * 100`
- **Measurement Frequency**: Weekly

**Success Criteria**:
- **Baseline**: 0% (new feature)
- **Target**: 40% adoption within 30 days (new projects define custom workflows)
- **Timeline**: Measured at 2 weeks, 4 weeks, 8 weeks post-launch
- **Minimum Viable**: 25% adoption (indicates feature resonates with users)

**Relates To**:
- **Requirement(s)**: REQ-F-001 (Load Workflow from Config), REQ-F-007 (Backward Compatible Default Workflow)
- **User Journey**: Journey 5 (Project Manager Customizes Workflow)
- **Business Value**: Adoption validates product-market fit for configurable workflows

---

### Metric 2: Workflow Validation Success Rate

**Type**: Lagging

**What We're Measuring**:
Percentage of status update commands that succeed on first attempt (not blocked by validation errors)

**How We'll Measure**:
- **Data Source**: CLI command logs (exit codes)
- **Calculation Method**: `(successful_status_updates / total_status_update_attempts) * 100`
- **Measurement Frequency**: Daily aggregation

**Success Criteria**:
- **Baseline**: Unknown (establish in first week)
- **Target**: >95% success rate
- **Timeline**: Measured continuously, evaluated at 30 days
- **Minimum Viable**: >90% (indicates workflow matches reality)

**Relates To**:
- **Requirement(s)**: REQ-F-004 (Enforce Valid Transitions)
- **User Journey**: All agent journeys (transition validation)
- **Business Value**: High success rate means workflows are well-designed; low rate indicates misalignment

---

### Metric 3: Force Flag Usage Rate

**Type**: Lagging

**What We're Measuring**:
Percentage of status updates that use `--force` flag to bypass validation

**How We'll Measure**:
- **Data Source**: task_history table (`forced=true` flag)
- **Calculation Method**: `(forced_transitions / total_transitions) * 100`
- **Measurement Frequency**: Daily aggregation

**Success Criteria**:
- **Baseline**: 0% (new feature)
- **Target**: <5% force flag usage
- **Timeline**: Measured at 2 weeks, 4 weeks, 8 weeks
- **Minimum Viable**: <10% (>10% indicates workflow doesn't match actual process)

**Relates To**:
- **Requirement(s)**: REQ-F-005 (Support Force Flag Override), REQ-NF-012 (Force Flag Abuse Detection)
- **User Journey**: Journey 6 (Emergency Hotfix with Force Flag)
- **Business Value**: Low force usage validates that workflows are realistic; high usage indicates design problems

---

### Metric 4: Workflow Validation Performance

**Type**: Lagging

**What We're Measuring**:
Latency overhead added by workflow validation to status update operations (95th percentile)

**How We'll Measure**:
- **Data Source**: Benchmark tests (automated performance suite)
- **Calculation Method**: Measure `UpdateStatus()` latency with vs without validation, calculate delta
- **Measurement Frequency**: Per-build (CI pipeline)

**Success Criteria**:
- **Baseline**: ~15ms P95 (current status update latency without validation)
- **Target**: <20ms added overhead (total P95 <35ms)
- **Timeline**: Measured in CI on every commit
- **Minimum Viable**: <100ms total latency (usability threshold)

**Relates To**:
- **Requirement(s)**: REQ-F-004 (Enforce Valid Transitions), REQ-NF-001 (Low Latency Status Validation)
- **User Journey**: All agent journeys (performance affects all status changes)
- **Business Value**: High latency degrades agent efficiency; must be imperceptible to users

---

### Metric 5: Agent Query Efficiency

**Type**: Leading

**What We're Measuring**:
Ratio of `task next` calls to `task complete` calls (measures how many queries per completed task)

**How We'll Measure**:
- **Data Source**: CLI command logs
- **Calculation Method**: `queries_per_completion = task_next_count / task_complete_count`
- **Measurement Frequency**: Weekly aggregation

**Success Criteria**:
- **Baseline**: Unknown (establish in first 2 weeks)
- **Target**: <3 queries per completion (agents find relevant tasks quickly)
- **Timeline**: Measured at 2 weeks, 4 weeks, 8 weeks
- **Minimum Viable**: <5 queries per completion

**Relates To**:
- **Requirement(s)**: REQ-F-015 (Filter by Agent Type), REQ-F-016 (Filter by Workflow Phase)
- **User Journey**: Journey 1 (Business Analyst), Step 1 (Query by Agent)
- **Business Value**: Low ratio means agents spend more time implementing, less time searching

---

## Secondary Success Metrics

### Metric 6: Configuration Error Rate

**Type**: Lagging

**What We're Measuring**:
Percentage of workflow configs that fail validation on first attempt

**How We'll Measure**:
- **Data Source**: `shark workflow validate` command logs (exit code 2)
- **Calculation Method**: `(failed_validations / total_validations) * 100`
- **Measurement Frequency**: Weekly aggregation

**Success Criteria**:
- **Baseline**: Unknown (establish in first 2 weeks)
- **Target**: <20% failure rate (most configs validate on first try)
- **Timeline**: 4 weeks post-launch
- **Minimum Viable**: <40%

**Improvement Lever**: High error rate indicates need for better documentation or schema design

---

### Metric 7: Support Ticket Volume

**Type**: Lagging

**What We're Measuring**:
Number of GitHub issues/discussions tagged "workflow" or "status" per week

**How We'll Measure**:
- **Data Source**: GitHub API (count issues with labels)
- **Calculation Method**: Count issues created per week
- **Measurement Frequency**: Weekly

**Success Criteria**:
- **Baseline**: 0 (pre-feature)
- **Target**: <2 support questions per week (after first 2 weeks)
- **Timeline**: Weeks 3-8 (exclude week 1-2 for launch spike)
- **Minimum Viable**: <5 support questions per week

**Quality Indicator**: Low support volume indicates good documentation and intuitive UX

---

## Success Criteria Summary

The epic is considered **successful** if:

1. **Adoption**: 40% of new projects define custom workflows within 30 days
2. **Usability**: <5% of transitions require `--force` flag
3. **Performance**: Validation adds <20ms overhead (P95)
4. **Backward Compatibility**: Existing projects work unchanged with default workflow
5. **No critical regressions**: Existing Shark features continue to work

---

## Risk Metrics

These metrics indicate problems if thresholds are exceeded:

| Metric | Warning Threshold | Critical Threshold | Action |
|--------|------------------|-------------------|--------|
| Force flag usage | >5% | >10% | Review workflows, gather user feedback on why forced transitions are needed |
| Config error rate | >30% | >50% | Improve documentation, add schema validation examples |
| Support ticket volume | >5/week | >10/week | Update docs, create FAQ, record video tutorial |
| Validation performance | >50ms P95 | >100ms P95 | Optimize config loading, add caching |

---

## Long-Term Metrics (3-6 Months)

### Workflow Complexity Growth
**Measure**: Average number of statuses per workflow over time
**Target**: Steady increase indicates teams customizing to match their processes
**Insight**: If workflows remain simple (4-5 statuses), feature may be underutilized

### Multi-Agent Workflow Adoption
**Measure**: Percentage of workflows that use agent_types metadata
**Target**: >60% of custom workflows
**Insight**: Validates AI-agent use case; low adoption may indicate metadata is confusing

### Backward Transition Usage
**Measure**: Percentage of transitions that move backward in workflow (e.g., QA → Dev)
**Target**: 10-20% of transitions (indicates realistic rework loops)
**Insight**: Too low suggests workflows don't support iteration; too high suggests quality issues

---

*See also*: [Requirements](./requirements.md), [User Journeys](./user-journeys.md)
