# Success Metrics

**Epic**: [Workflow-Aware Task Command System](./epic.md)

---

## Overview

This document defines the Key Performance Indicators (KPIs) that will measure the success of workflow-aware task commands. Metrics focus on compatibility, adoption, and user experience improvements.

**Measurement Timeline**:
- **T+2 weeks**: Initial compatibility and command usage metrics
- **T+6 weeks**: Full migration tracking and adoption rates
- **T+12 weeks**: Long-term success evaluation and user feedback

---

## Primary Success Metrics

### Metric 1: Workflow Compatibility Rate

**Type**: Lagging

**What We're Measuring**:
Percentage of workflow configurations that work correctly with new phase-aware commands without modification

**How We'll Measure**:
- **Data Source**: Automated test suite with diverse workflow configurations
- **Calculation Method**:
  ```
  Compatibility Rate = (Workflows Passing All Tests / Total Workflows Tested) × 100%
  ```
- **Test Workflows**:
  - Default 3-state workflow (todo → in_progress → completed)
  - Simple 5-state custom workflow
  - Complex 10-state enterprise workflow (SAFe-like)
  - Minimal 2-state workflow (draft → completed)
  - Branching workflow with skip paths
- **Measurement Frequency**: Continuous (CI/CD pipeline)

**Success Criteria**:
- **Baseline**: 33% (only default workflow works with current commands)
- **Target**: 100% (all test workflows work with phase-aware commands)
- **Timeline**: Achieved by release 1 (non-negotiable)
- **Minimum Viable**: 95% (allow for edge cases)

**Relates To**:
- **Requirement(s)**: REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-004, REQ-NF-009
- **User Journey**: Journey 5 (Workflow Setup)
- **Business Value**: Core value prop - workflow flexibility

---

### Metric 2: AI Orchestrator Integration Success

**Type**: Lagging

**What We're Measuring**:
Ability of AI orchestrator to successfully assign tasks and manage handoffs using only phase-aware commands (no hardcoded status values)

**How We'll Measure**:
- **Data Source**: Integration test suite simulating full orchestrator workflow
- **Calculation Method**: Binary success/failure per scenario
- **Test Scenarios**:
  1. Task assigned and claimed by first agent
  2. Task finished and handed to next phase agent
  3. Task rejected backward through workflow
  4. Task blocked and unblocked
  5. Multiple concurrent task assignments
  6. Workflow config changed mid-execution
- **Measurement Frequency**: Per release

**Success Criteria**:
- **Baseline**: 0% (current hardcoded commands incompatible with orchestrator)
- **Target**: 100% (all scenarios pass)
- **Timeline**: Achieved by release 1
- **Minimum Viable**: 100% (critical requirement, no negotiation)

**Relates To**:
- **Requirement(s)**: REQ-F-001, REQ-F-002, REQ-F-003, REQ-F-006, REQ-NF-010
- **User Journey**: Journey 1 (AI Orchestrator Autonomous Assignment)
- **Business Value**: Enables core orchestrator functionality

---

### Metric 3: Command Discovery Speed

**Type**: Leading

**What We're Measuring**:
Time for users to find the correct command for their task (via help text, search)

**How We'll Measure**:
- **Data Source**: User testing sessions (5-10 users)
- **Test Method**:
  1. Give user scenario: "You want to start working on a task"
  2. Measure time from `shark task --help` to correct command
  3. Repeat with categorized vs. uncategorized help
- **Calculation Method**:
  ```
  Avg Discovery Time = Sum(discovery_times) / User Count
  ```
- **Measurement Frequency**: Pre-release and post-release user testing

**Success Criteria**:
- **Baseline**: 15 seconds (average with 25 uncategorized commands)
- **Target**: 3 seconds (78% reduction)
- **Timeline**: Measured at T+2 weeks
- **Minimum Viable**: 5 seconds (67% reduction)

**Relates To**:
- **Requirement(s)**: REQ-F-011
- **User Journey**: All journeys (improved UX)
- **Business Value**: Developer experience, reduces learning curve

---

### Metric 4: Migration Adoption Rate

**Type**: Lagging

**What We're Measuring**:
Percentage of users who switch from deprecated commands to new phase-aware commands

**How We'll Measure**:
- **Data Source**: Command telemetry (if enabled) or survey
- **Calculation Method**:
  ```
  Adoption Rate = (Users Using New Commands / Total Active Users) × 100%
  ```
- **Tracking Method**:
  - Release 1: Track usage of both old and new commands
  - Release 2: Count deprecation warnings shown
  - Release 3: Measure failed attempts at removed commands
- **Measurement Frequency**: Weekly for 12 weeks post-release

**Success Criteria**:
- **Baseline**: 0% (new commands don't exist)
- **Target (Release 2)**: 50% adoption
- **Target (Release 3)**: 90% adoption
- **Timeline**: 90% by 12 weeks after Release 2
- **Minimum Viable**: 70% by Release 3

**Relates To**:
- **Requirement(s)**: REQ-NF-003
- **User Journey**: All journeys (migration)
- **Business Value**: Smooth transition, reduced support burden

---

## Secondary Success Metrics

### Metric 5: Command Count Reduction

**Type**: Lagging

**What We're Measuring**:
Reduction in total number of task subcommands after consolidation

**How We'll Measure**:
- **Data Source**: Count of commands in `shark task --help`
- **Calculation Method**: Direct count before and after
- **Measurement Frequency**: Per release

**Success Criteria**:
- **Baseline**: 25 commands
- **Target**: 18 commands (28% reduction)
- **Timeline**: Achieved by Release 3 (after deprecated commands removed)
- **Minimum Viable**: 20 commands (20% reduction)

**Relates To**:
- **Requirement(s)**: REQ-F-008, REQ-F-009, REQ-F-010
- **Impact**: Cognitive load reduction

---

### Metric 6: Error Message Clarity

**Type**: Leading

**What We're Measuring**:
Percentage of users who successfully resolve errors using only error message guidance (no documentation lookup)

**How We'll Measure**:
- **Data Source**: User testing with intentional error scenarios
- **Test Method**:
  1. Give user invalid command (e.g., claim already-claimed task)
  2. Measure if user successfully recovers using error message alone
  3. Track documentation lookups
- **Calculation Method**:
  ```
  Self-Service Resolution = (Resolved Without Docs / Total Error Cases) × 100%
  ```
- **Measurement Frequency**: User testing at T+4 weeks

**Success Criteria**:
- **Baseline**: 30% (current error messages are terse)
- **Target**: 80% (actionable error messages)
- **Timeline**: Measured at T+4 weeks
- **Minimum Viable**: 60%

**Relates To**:
- **Requirement(s)**: REQ-NF-005, REQ-F-012
- **Impact**: Reduced support tickets, faster error recovery

---

### Metric 7: Phase Transition Correctness

**Type**: Lagging

**What We're Measuring**:
Percentage of status transitions that correctly follow workflow configuration (no invalid transitions)

**How We'll Measure**:
- **Data Source**: Database audit log of all status transitions
- **Calculation Method**:
  ```
  Correctness = (Valid Transitions / Total Transitions) × 100%
  ```
- **Measurement Frequency**: Continuous monitoring via audit log
- **Alert Threshold**: < 99% correctness

**Success Criteria**:
- **Baseline**: 85% (current hardcoded transitions sometimes violate custom workflows)
- **Target**: 100% (all transitions validated against workflow config)
- **Timeline**: Achieved by release 1, maintained forever
- **Minimum Viable**: 99.9% (allow for edge cases/bugs)

**Relates To**:
- **Requirement(s)**: REQ-F-004, REQ-F-005
- **Impact**: Workflow integrity, prevents bad state

---

## Success Criteria Summary

The epic is considered **successful** if:

1. **100% workflow compatibility** - All workflow configurations work with phase-aware commands
2. **100% orchestrator integration** - AI orchestrator successfully assigns/manages tasks
3. **90% migration adoption** - Users switch to new commands by Release 3
4. **No regressions** in existing functionality (backward compatibility maintained)

**Go/No-Go Decision Points**:

- **After Release 1**: If compatibility < 95% OR orchestrator integration fails, DO NOT proceed to Release 2
- **After Release 2**: If adoption < 40%, delay Release 3 and increase migration support
- **Before Release 3**: If adoption < 70%, keep deprecated commands for one more release

---

## Anti-Metrics (What We're NOT Optimizing For)

**NOT measuring**:
- Lines of code written (focus on functionality, not code volume)
- Speed of implementation (focus on correctness and UX)
- Number of features (focus on essential commands, not feature creep)

**Why**: These metrics could incentivize wrong behaviors (bloat, rushed code, unnecessary features).

---

## Measurement Dashboard

**Tracking Location**: `docs/metrics/E13-workflow-commands.md`

**Update Frequency**: Weekly during implementation, bi-weekly after release

**Template**:
```markdown
## Week of YYYY-MM-DD

| Metric | Target | Current | Status | Notes |
|--------|--------|---------|--------|-------|
| Workflow Compatibility | 100% | 95% | ⚠️ | 1 edge case failing |
| Orchestrator Integration | 100% | 100% | ✅ | All scenarios pass |
| Command Discovery | 3s | 4.2s | ⚠️ | Needs categorization polish |
| Migration Adoption | 50% | 35% | ⚠️ | Increase docs/examples |
```

---

*See also*: [Requirements](./requirements.md), [Scope](./scope.md)
