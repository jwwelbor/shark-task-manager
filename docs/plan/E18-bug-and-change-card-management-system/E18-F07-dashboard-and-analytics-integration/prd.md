# E18-F07: Dashboard and Analytics Integration -- Feature PRD

**Feature Key**: E18-F07
**Epic**: [E18 -- Bug and Change-Card Management System](../epic.md)
**Complexity Tier**: STANDARD
**Status**: In Refinement (BA)

---

## Goal

### Problem

Once bugs (F02) and change-cards (F03) exist as first-class entities with full CLI support (F04, F05) and unified key auto-detection (F06), they remain invisible in the two primary project oversight surfaces: the `shark status` dashboard and the `shark analytics` command. Product owners cannot see the balance between planned work (epics/features/tasks) and reactive work (bugs/change-cards). QA engineers cannot view bug severity distributions or resolution trends. Change-card throughput is unmeasured.

### Solution

Extend the existing `StatusService.GetDashboard()` and `shark analytics` command to include bug and change-card data. This is an incremental extension of existing rendering and calculation infrastructure -- not a new command or service.

**Dashboard** (`shark status`):
- Add a "Bugs" section with counts by status and severity breakdown, displayed conditionally (only when bugs exist in the database).
- Add a "Change Cards" section with counts by status, displayed conditionally.
- Extend feature-level status (`shark status E07-F01`) to show linked bug count and severity summary when bugs are linked to that feature.

**Analytics** (`shark analytics`):
- Add bug metrics: total count, count by status, count by severity, average resolution time (reported-to-resolved).
- Add change-card metrics: total count, count by status, approval rate, average time-to-completion (proposed-to-completed).
- Add `--type=bug` and `--type=change` filters to show entity-specific analytics.

### Impact

- Product owners get a complete single-view dashboard covering all 5 entity types.
- Bug resolution time and severity trends become measurable, enabling data-driven quality investment decisions.
- Change-card throughput validates the lightweight enhancement path introduced by E18.
- All 5 epic-level success metrics (see [success-metrics.md](../success-metrics.md)) become measurable through CLI output.

---

## Personas

This feature serves all three personas defined in the [epic personas document](../personas.md):

| Persona | Primary Interaction | Value |
|---------|-------------------|-------|
| **Developer** | Glances at `shark status` to see if bugs are assigned | Awareness of reactive work queue |
| **QA Engineer** | Checks `shark analytics --type=bug` for severity and resolution trends | Quality trend monitoring |
| **Product Owner** | Reviews `shark status` and `shark analytics` for planned vs. reactive work balance | Capacity planning and prioritization |

No new personas are introduced by this feature.

---

## User Stories (MoSCoW)

### Must Have

**US-F07-001**: Dashboard Bug Summary
As a product owner, I want to see bug counts by status on the project dashboard so that I know the current bug backlog size at a glance.

**US-F07-002**: Dashboard Bug Severity Breakdown
As a QA engineer, I want to see bug severity distribution on the project dashboard so that I can identify if critical bugs are accumulating.

**US-F07-003**: Dashboard Change-Card Summary
As a product owner, I want to see change-card counts by status on the project dashboard so that I know how many enhancements are proposed, approved, or in progress.

**US-F07-004**: Analytics Bug Metrics
As a QA engineer, I want to see total bugs, bugs by status, bugs by severity, and average resolution time in analytics output so that I can track quality trends over time.

**US-F07-005**: Analytics Change-Card Metrics
As a product owner, I want to see total change-cards, change-cards by status, approval rate, and average time-to-completion in analytics output so that I can measure the lightweight enhancement pipeline.

**US-F07-006**: Analytics Type Filtering
As a QA engineer, I want to run `shark analytics --type=bug` to see only bug-related analytics so that I can focus on defect metrics without noise from other entity types.

### Should Have

**US-F07-007**: Feature-Level Bug Context
As a product owner, I want `shark status E07-F01` to show linked bug count and severity summary when bugs are linked to that feature so that I can assess per-feature quality.

**US-F07-008**: Conditional Section Display
As a developer, I want the dashboard to omit the Bugs section entirely when no bugs exist so that the output remains clean in projects that have not adopted bug tracking yet.

**US-F07-009**: JSON Output for Analytics
As a CI/CD pipeline agent, I want `shark analytics --type=bug --json` to produce machine-readable JSON so that I can build automated quality dashboards.

### Could Have

**US-F07-010**: Resolution Time Percentiles
As a QA engineer, I want analytics to show P50 and P90 resolution times (not just average) so that I can identify outlier bugs that take disproportionately long to fix.

---

## Functional Requirements

### Requirement Traceability

All requirements trace to parent epic requirements:

| Feature Requirement | Epic Requirement | Relationship |
|--------------------|-----------------|--------------|
| FR-F07-001 through FR-F07-004 | REQ-F-013 (Dashboard Integration) | Detailed implementation of epic requirement |
| FR-F07-005 through FR-F07-009 | REQ-F-014 (Analytics Integration) | Detailed implementation of epic requirement |
| FR-F07-010 | REQ-NF-002 (List Performance) | Aggregate query performance constraint |

### Dashboard Requirements

**FR-F07-001**: Bug Status Section in Dashboard
- The `shark status` command (no arguments, project-level dashboard) must include a "Bugs" section showing counts by status.
- Statuses displayed: reported, triaged, in_fix, in_verification, resolved, wont_fix, duplicate.
- Section is conditionally displayed: omitted entirely when the database contains zero bugs.
- Acceptance Criteria:
  - Given 3 bugs in the database (1 reported, 1 in_fix, 1 resolved), when the user runs `shark status`, then the output includes a "Bugs" section showing "reported: 1, in_fix: 1, resolved: 1".
  - Given 0 bugs in the database, when the user runs `shark status`, then no "Bugs" section appears in the output.
  - Given `shark status --json` is run, then the JSON output includes a `bugs` object with status counts.

**FR-F07-002**: Bug Severity Breakdown in Dashboard
- The dashboard "Bugs" section must include a severity breakdown showing counts for each severity level.
- Severity levels: critical, high, medium, low.
- Only open (non-terminal) bugs are counted in the severity breakdown.
- Acceptance Criteria:
  - Given 4 open bugs (1 critical, 2 high, 1 medium), when the user runs `shark status`, then the "Bugs" section includes "Severity: critical: 1, high: 2, medium: 1, low: 0".
  - Given all bugs are in terminal status (resolved, wont_fix, duplicate), when the user runs `shark status`, then the severity breakdown shows all zeros or is omitted.

**FR-F07-003**: Change-Card Status Section in Dashboard
- The `shark status` command must include a "Change Cards" section showing counts by status.
- Statuses displayed: proposed, approved, in_progress, completed, declined.
- Section is conditionally displayed: omitted entirely when the database contains zero change-cards.
- Acceptance Criteria:
  - Given 5 change-cards (2 proposed, 1 approved, 1 in_progress, 1 completed), when the user runs `shark status`, then the output includes a "Change Cards" section showing counts for each status.
  - Given 0 change-cards, when the user runs `shark status`, then no "Change Cards" section appears.

**FR-F07-004**: Feature-Level Bug Context
- When viewing a feature's status (`shark status E07-F01`), the output must include linked bug information if any bugs are linked to that feature.
- Information shown: total linked bug count and severity breakdown of open linked bugs.
- Acceptance Criteria:
  - Given feature E07-F01 has 3 linked bugs (2 open, 1 resolved), when the user runs `shark status E07-F01`, then the output includes "Linked Bugs: 3 (2 open -- 1 high, 1 medium)".
  - Given feature E07-F01 has 0 linked bugs, when the user runs `shark status E07-F01`, then no bug information appears.

### Analytics Requirements

**FR-F07-005**: Bug Analytics -- Totals and Status Breakdown
- `shark analytics` (without type filter) must include bug summary metrics: total bug count and count by status.
- `shark analytics --type=bug` must show bug-only analytics.
- Acceptance Criteria:
  - Given 10 bugs in various statuses, when the user runs `shark analytics`, then the output includes a "Bugs" section with total count and per-status breakdown.
  - Given `shark analytics --type=bug` is run, then only bug analytics are shown (no session duration or other metrics).

**FR-F07-006**: Bug Analytics -- Severity Distribution
- Bug analytics must include severity distribution: count of bugs at each severity level.
- Acceptance Criteria:
  - Given 10 bugs with mixed severities, when the user runs `shark analytics --type=bug`, then the output includes severity distribution (e.g., "critical: 2, high: 3, medium: 4, low: 1").

**FR-F07-007**: Bug Analytics -- Average Resolution Time
- Bug analytics must include average resolution time for resolved bugs.
- Resolution time is calculated as the difference between the bug's `created_at` timestamp and the timestamp when the bug reached a terminal status (resolved, wont_fix, or duplicate).
- Terminal status timestamps come from status history records.
- Acceptance Criteria:
  - Given 3 resolved bugs with resolution times of 2h, 4h, and 6h, when the user runs `shark analytics --type=bug`, then the output includes "Avg Resolution Time: 4h 0m".
  - Given 0 resolved bugs, when the user runs `shark analytics --type=bug`, then the resolution time field shows "N/A" or "No resolved bugs".

**FR-F07-008**: Change-Card Analytics -- Totals and Status Breakdown
- `shark analytics` must include change-card summary metrics: total count and count by status.
- `shark analytics --type=change` must show change-card-only analytics.
- Acceptance Criteria:
  - Given 8 change-cards in various statuses, when the user runs `shark analytics --type=change`, then the output includes total count and per-status breakdown.

**FR-F07-009**: Change-Card Analytics -- Approval Rate and Throughput
- Change-card analytics must include:
  - Approval rate: approved / (approved + declined), expressed as a percentage.
  - Average time-to-completion: average elapsed time from proposed to completed for change-cards that reached completed status.
- Acceptance Criteria:
  - Given 6 change-cards decided (4 approved, 2 declined), when the user runs `shark analytics --type=change`, then the output includes "Approval Rate: 66.7%".
  - Given 3 completed change-cards with completion times of 1d, 3d, and 5d, then the output includes "Avg Completion Time: 3d 0h".
  - Given 0 completed change-cards, then the completion time field shows "N/A".

**FR-F07-010**: Analytics JSON Output
- All analytics outputs must support `--json` for machine consumption.
- JSON structure must be consistent with existing analytics JSON patterns.
- Acceptance Criteria:
  - Given `shark analytics --type=bug --json` is run, then the output is valid JSON containing keys: `total_bugs`, `bugs_by_status`, `bugs_by_severity`, `avg_resolution_time_seconds`.
  - Given `shark analytics --type=change --json` is run, then the output is valid JSON containing keys: `total_change_cards`, `change_cards_by_status`, `approval_rate`, `avg_completion_time_seconds`.

---

## Technical Integration Points

This section describes where the feature integrates with existing code. It references existing patterns without duplicating architecture documentation.

### Repository Layer (New Query Methods)

The following aggregate query methods are needed on the bug and change-card repositories (created by F02 and F03):

| Method | Repository | Returns |
|--------|-----------|---------|
| `CountByStatus(ctx)` | BugRepository | `map[string]int` |
| `CountBySeverity(ctx, openOnly bool)` | BugRepository | `map[string]int` |
| `GetAverageResolutionTime(ctx)` | BugRepository | `*time.Duration, error` |
| `CountLinkedToFeature(ctx, featureKey)` | BugRepository | `int, error` |
| `CountLinkedToFeatureBySeverity(ctx, featureKey, openOnly)` | BugRepository | `map[string]int` |
| `CountByStatus(ctx)` | ChangeCardRepository | `map[string]int` |
| `GetApprovalRate(ctx)` | ChangeCardRepository | `float64, error` |
| `GetAverageCompletionTime(ctx)` | ChangeCardRepository | `*time.Duration, error` |

These methods perform aggregate SQL queries (COUNT, AVG, GROUP BY) and return summary data, not entity lists. They follow the existing repository pattern of returning domain-level types.

### StatusService Extension (Dashboard)

The `StatusService.GetDashboard()` method in `internal/status/status.go` must be extended to:
1. Query `BugRepository.CountByStatus()` and `BugRepository.CountBySeverity()`.
2. Query `ChangeCardRepository.CountByStatus()`.
3. Add `BugSummary` and `ChangeCardSummary` fields to the `StatusDashboard` struct.
4. Conditionally populate these fields (nil/omitted when counts are zero).

The `StatusDashboard` struct needs new optional fields:

```go
type StatusDashboard struct {
    // ... existing fields ...
    BugSummary        *BugDashboardSummary        `json:"bugs,omitempty"`
    ChangeCardSummary *ChangeCardDashboardSummary  `json:"change_cards,omitempty"`
}
```

### Analytics Extension

The `shark analytics` command currently supports session-duration and pause-frequency analysis. Bug and change-card analytics are a new analysis type. The command needs:
1. New `--type` flag accepting values: `bug`, `change` (in addition to existing analysis flags).
2. A new service or extension to `EpicAnalyticsService` that queries bug and change-card repositories.
3. Terminal and JSON formatters for the new metrics.

### FormatDashboard Extension

The `FormatDashboard()` function in `internal/status/formatter.go` must be extended with new rendering sections for bugs and change-cards, following the existing section rendering pattern.

---

## Scope Boundaries

### In Scope

1. Dashboard bug section (status counts, severity breakdown, conditional display).
2. Dashboard change-card section (status counts, conditional display).
3. Feature-level linked bug context.
4. Analytics bug metrics (total, by status, by severity, avg resolution time).
5. Analytics change-card metrics (total, by status, approval rate, avg completion time).
6. `--type=bug` and `--type=change` analytics filters.
7. JSON output for all new metrics.
8. Repository aggregate query methods for dashboard and analytics.
9. Unit tests for all calculation and rendering logic.

### Out of Scope

- **Web UI dashboard**: Explicitly excluded per [epic scope.md](../scope.md). All output is terminal-based or JSON.
- **Historical trend charts**: This feature provides current-state metrics and averages, not time-series charts. Time-series analysis is a future enhancement.
- **SLA tracking and escalation**: Excluded per epic scope. No time-based alerts or rules.
- **Resolution time percentiles (P50/P90)**: Could-Have (US-F07-010). Include if implementation is simple; defer otherwise.
- **Epic-level linked bug/change-card rollup**: Only feature-level linking is in scope for the dashboard. Epic-level rollups (aggregate bugs across all features in an epic) are a future enhancement.
- **New CLI commands**: This feature extends existing commands (`shark status`, `shark analytics`). No new top-level commands are introduced.

---

## Dependencies

| Dependency | Type | Detail |
|-----------|------|--------|
| E18-F01 (Database Schema) | Hard | Bug and change-card tables must exist |
| E18-F02 (Bug Entity Core) | Hard | BugRepository with aggregate query methods |
| E18-F03 (Change-Card Entity Core) | Hard | ChangeCardRepository with aggregate query methods |
| E18-F06 (Unified CLI Integration) | Soft | Dashboard rendering must recognize bug/change-card entity types; can be developed in parallel |
| Existing `StatusService` | Extension | Dashboard code extends existing service |
| Existing `shark analytics` | Extension | Analytics command extended with new flags and output |

---

## Acceptance Criteria Summary

The feature is complete when:

1. `shark status` displays bug counts by status and severity breakdown when bugs exist.
2. `shark status` displays change-card counts by status when change-cards exist.
3. Both sections are omitted when the respective entity type has zero records.
4. `shark status E07-F01` shows linked bug count and severity when bugs are linked to the feature.
5. `shark analytics --type=bug` shows total, by-status, by-severity, and average resolution time.
6. `shark analytics --type=change` shows total, by-status, approval rate, and average completion time.
7. `shark analytics` (no type filter) includes both bug and change-card summaries alongside existing metrics.
8. All outputs support `--json` with consistent structure.
9. Dashboard and analytics tests pass with mock data covering zero-entity, single-entity, and multi-entity scenarios.
10. Performance: aggregate queries complete in under 100ms for databases with up to 1000 bugs and 1000 change-cards.

---

*Traces to*: [REQ-F-013](../requirements.md), [REQ-F-014](../requirements.md), [REQ-NF-002](../requirements.md)
*Enables*: [Success Metrics 1-5](../success-metrics.md)
*Last Updated*: 2026-03-03
