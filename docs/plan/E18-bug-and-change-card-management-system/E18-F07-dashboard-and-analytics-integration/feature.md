---
feature_key: E18-F07-dashboard-and-analytics-integration
epic_key: E18
title: Dashboard and Analytics Integration
description: Extend shark status dashboard and shark analytics to include bug and change-card counts, severity breakdowns, resolution metrics, and throughput data.
---

# Dashboard and Analytics Integration

**Feature Key**: E18-F07
**Execution Order**: 7 (depends on F02, F03, F06)

---

## Goal

### Problem

After bugs and change-cards are functional entities with full CLI support, they are invisible in the project dashboard (`shark status`) and analytics (`shark analytics`). Product owners cannot see the balance between planned work and reactive work. QA engineers cannot see bug severity distributions. Change-card throughput is not measured.

### Solution

Extend the dashboard and analytics systems to include bug and change-card data:
- **Dashboard** (`shark status`): Add "Bugs" section with counts by status and severity breakdown. Add "Change Cards" section with counts by status. Sections are conditionally displayed (only shown when entities of that type exist). Feature status (`shark status E07-F01`) shows linked bug count if any bugs are linked.
- **Analytics** (`shark analytics`): Add bug metrics (total, by status, by severity, average resolution time). Add change-card metrics (total, by status, approval rate, average time-to-completion). Support `--type=bug` and `--type=change` filters.

### Impact

- Product owners get a complete view of all work types in a single dashboard
- Bug severity and resolution trends are measurable
- Change-card throughput validates the lightweight enhancement path
- All 5 success metrics from the epic can be measured through CLI

---

## Scope

### In Scope

1. **Dashboard bug section** -- Bug counts by status (reported: N, triaged: N, in_fix: N, etc.). Severity breakdown (critical: N, high: N, medium: N, low: N). Conditionally displayed (no section if zero bugs exist).
2. **Dashboard change-card section** -- Change-card counts by status (proposed: N, approved: N, in_progress: N, completed: N, declined: N). Conditionally displayed.
3. **Feature status bug context** -- `shark status E07-F01` includes linked bug count and severity summary if bugs are linked to that feature.
4. **Analytics bug metrics** -- Total bugs, bugs by status, bugs by severity, average resolution time (reported-to-resolved). `shark analytics --type=bug` shows bug-only analytics.
5. **Analytics change-card metrics** -- Total change-cards, change-cards by status, approval rate (approved / (approved + declined)), average time-to-completion (proposed-to-completed). `shark analytics --type=change` shows change-card-only analytics.
6. **Resolution time calculation** -- Computed from status history timestamps. Average of (resolution_timestamp - creation_timestamp) for all bugs in terminal status.
7. **Throughput calculation** -- Count of change-cards reaching completed status per time period.
8. **Repository query methods** -- CountBugsByStatus, CountBugsBySeverity, GetAverageBugResolutionTime, CountChangeCardsByStatus, GetChangeCardApprovalRate, GetAverageChangeCardCompletionTime.
9. **Tests** -- Dashboard rendering tests, analytics calculation tests with mock data.

### Out of Scope

- Web UI dashboard (explicitly excluded per scope.md)
- SLA tracking and escalation (explicitly excluded per scope.md)
- Historical trend charts (beyond raw metrics)

---

## Requirements Traceability

| Epic Requirement | Coverage |
|-----------------|----------|
| REQ-F-013 (Dashboard Integration) | Dashboard bug and change-card sections |
| REQ-F-014 (Analytics Integration) | Analytics bug and change-card metrics |
| REQ-NF-002 (List Performance) | Efficient aggregate queries for dashboard |

---

## Success Metrics Validation

This feature directly enables measurement of all 5 epic success metrics:

| Success Metric | Analytics Output |
|---------------|-----------------|
| Metric 1: Bug Tracking Adoption Rate | `shark analytics --type=bug --json` provides creation counts |
| Metric 2: Average Bug Resolution Time | `shark analytics --type=bug` includes avg resolution time |
| Metric 3: Change-Card Throughput | `shark analytics --type=change` includes completed count |
| Metric 4: Bug Creation Speed | Not analytics-dependent (measured via shell timing) |
| Metric 5: Dashboard Visibility Score | `shark status` displays bug and change-card sections |

---

## Dependencies

- **F02 (Bug Entity Core)**: Requires BugRepository query methods.
- **F03 (Change-Card Entity Core)**: Requires ChangeCardRepository query methods.
- **F06 (Unified CLI Integration)**: Dashboard and analytics commands must recognize bug/change-card entity types.
- **Existing status package**: Extends `CalculationService` for dashboard data.
- **Existing analytics service**: Extends for bug/change-card metrics.

---

## Sprint Sizing

**Estimate**: 1-2 sprints (M complexity)

- Dashboard rendering: Medium (new sections, conditional display)
- Analytics calculations: Medium (resolution time, approval rate, throughput)
- Repository query methods: Small-Medium (aggregate queries)
- Feature status extension: Small
- Tests: Medium

---

*Last Updated*: 2026-03-02
