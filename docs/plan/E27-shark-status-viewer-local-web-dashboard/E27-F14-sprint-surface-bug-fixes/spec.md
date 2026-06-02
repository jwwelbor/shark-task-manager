---
feature_key: E27-F14
epic_key: E27
doc_type: spec
status: draft
---

# E27-F14 — Spec: Sprint Surface Bug Fixes

**Feature:** E27-F14 — Sprint Surface Bug Fixes  
**Epic:** E27 — Shark Status Viewer (Local Web Dashboard)  
**Source:** Post-implementation review of E27-F13 against wireframes in `sprint-planning-web-ui-wireframes.md`

---

## 1. Requirements

### 1.1 Functional Requirements

These requirements are incremental fixes to the E27-F13 sprint surface. For system-level business context see the epic PRD (`docs/plan/E27-shark-status-viewer-local-web-dashboard/epic.md`).

**REQ-F-001 (P0): Report tab graceful degradation**  
`GET /api/v1/viewer/sprint/report` must return HTTP 200 for sprints in any status (`planning`, `active`, `completed`, `archived`). For non-completed/non-archived sprints where `GetSummary` returns an error, the response body must include `sprint`, `burndown`, `velocity`, and `summary: null`.

**REQ-F-002 (P1): Sidebar sprint bucket counts are accurate**  
The "Active Sprint" section in the sidebar must display non-zero item counts whenever a sprint has assigned items. Bucket totals (Ready, In Progress, Blocked, Done) must correctly map workflow status categories from the API response to the four display buckets. The header-level total must equal the sum across all buckets.

**REQ-F-003 (P2): Sprint date ranges display as date-only**  
Sprint start and end dates must render as `"May 8 → May 22"` (month and day, no time component). Cross-year dates must include the year: `"Dec 30, 2025 → Jan 13, 2026"`. When either date is absent the display must show `"—"` for that position.

**REQ-F-004 (P2): Plan tab status badges show real status values**  
Every candidate backlog item in the Plan tab must render a meaningful status badge. The badge must reflect the entity's actual workflow status or, if not available in the payload, fall back to `status_category`. Items with no determinable status must show `"unassigned"` in a neutral (non-error) color rather than `"unknown"`.

**REQ-F-005 (P2): WIP metric is renamed "Active" and counts all active-work categories**  
The WIP metric card in the Overview must count items in all active workflow states (`in_development`, `ready_for_approval`, `ready_for_development`, `in_progress`, `in_review`, `in_qa`, `ready_for_refinement_tech`, `ready_for_refinement_ba`, `in_refinement`). The metric card label must read "Active". Blocked and Done items must not be counted.

**REQ-F-006 (P1): Plan tab shows two columns — Candidate Backlog and Assigned This Sprint**  
The Plan tab must render a side-by-side layout: left column "Candidate Backlog" (unassigned items with filters/checkboxes), right column "Assigned This Sprint" (currently scoped items with a count in the column header). On viewports narrower than 900 px the columns must stack vertically.

**REQ-F-007 (P1): Overview shows Capacity/Readiness and Blockers/Risks panels**  
The Overview tab must include a two-column health zone between the metric pills and the Sprint Backlog section: left panel shows per-agent Capacity rows (allocated/total + OK/HIGH/OVER indicator) and failing Readiness factors; right panel shows Blockers enumerated by key/title/days-blocked age and a warning count of unsized items.

**REQ-F-008 (P1): Overview shows Assigned Scope Snapshot bar**  
An "Assigned Scope Snapshot" section must appear at the bottom of the Overview, displaying a one-line status distribution of assigned sprint items (`Ready N | In Progress N | Blocked N | Done N`). Buckets with zero items must be omitted. The section must be absent when the sprint has no assigned items.

**REQ-F-009 (P2): Plan tab UX review and layout improvement**  
The Plan tab layout must be evaluated and improved so candidate backlog items are scannable, clearly communicate status, agent type, and assignment state, and pass a basic readability check in the dark theme. This work is coordinated with the UX designer agent and builds on REQ-F-004 (status badges) and REQ-F-006 (two-column layout).

### 1.2 Non-Functional Requirements

**REQ-NF-001 (Performance):** All fixes are client-side JS changes or a single-line Go change. No new API endpoints, no additional round trips. The existing polling budget (10-second interval for recent activity) is unaffected.

**REQ-NF-002 (No schema changes):** No database migrations. `CurrentSchemaVersion` in `internal/db/db.go` is not bumped. Follows the E27 architecture decision to keep the viewer layer purely at the composition/presentation level (see E27 architecture.md §3.1).

**REQ-NF-003 (Backward compatibility):** The `GetSummary` error-handling change in Go must not affect existing tests for completed/archived sprints. All other fixes are additive JS changes that degrade gracefully when data fields are absent.

**REQ-NF-004 (No new dependencies):** No new npm packages, CDN references, or Go modules. Fixes use only data already present in existing API responses.

### 1.3 Acceptance Criteria

| ID | Criterion | Verifiable by |
|----|-----------|---------------|
| AC-001 | `GET /api/v1/viewer/sprint/report` returns HTTP 200 for a `planning` sprint | curl / handler test |
| AC-002 | Report response body for planning sprint includes `summary: null` (not absent) | curl / JSON assertion |
| AC-003 | Completed/archived sprints still return full `summary` object | existing tests pass |
| AC-004 | Sidebar "Active Sprint" total count matches actual sprint assignment count | manual + JS unit |
| AC-005 | Ready bucket shows items in `ready_for_development`, `todo`, `draft` statuses | manual |
| AC-006 | In Progress bucket shows items in `in_development`, `ready_for_approval`, `in_progress` statuses | manual |
| AC-007 | Blocked bucket shows only items in `blocked` status | manual |
| AC-008 | Done bucket shows items in `completed`, `resolved` statuses | manual |
| AC-009 | Sprint date range renders as `"May 8 → May 22"` (no time component) | manual |
| AC-010 | Cross-year date range renders as `"Dec 30, 2025 → Jan 13, 2026"` | JS unit |
| AC-011 | Plan tab items show real status badges (e.g., `todo`, `in_development`) not `unknown` | manual |
| AC-012 | Items with no status show `unassigned` in neutral color | manual |
| AC-013 | Overview "Active" metric counts `in_development` + `ready_for_approval` + `ready_for_development` (and other active states); label reads "Active" | manual |
| AC-014 | Plan tab shows two columns side-by-side on desktop (≥900 px) | manual |
| AC-015 | Plan tab columns stack vertically at <900 px | manual |
| AC-016 | "Assigned This Sprint" column shows current sprint scope with item count in header | manual |
| AC-017 | Overview shows Capacity by Agent Type and Blockers/Risks panels | manual |
| AC-018 | Capacity rows show `OK`/`HIGH`/`OVER` indicator | manual |
| AC-019 | Blocked items in Capacity panel are clickable and open the detail drawer | manual |
| AC-020 | Overview bottom shows "Assigned Scope Snapshot" with bucket distribution | manual |
| AC-021 | Snapshot section absent when sprint has no assigned items | manual |
| AC-022 | Plan tab passes readability check in dark theme | manual / UX review |
| AC-023 | All existing `viewer_service_test.go` tests continue to pass | `make test` |

### 1.4 Out of Scope

Per `feature.md` "Out of Scope" and wireframe "Deferred interactions":
- Drag-and-drop prioritization
- Inline capacity editing
- Multi-sprint comparison
- Plan tab Priority/Size/Dependency filters and search box (P2-4)
- Error state retry button (P2-6)
- Empty planning state screen (P2-7)
- Cycle Time by Phase and Agent Utilization in Report tab

---

## 2. Architecture

### 2.1 Component Changes

This feature touches exactly two files and optionally a third (repository):

| File | Change Type | Reason |
|------|-------------|--------|
| `internal/services/viewer_service.go` | Modify | Fix P0-1: make `GetSummary` error non-fatal in `SprintReport()` |
| `internal/viewer/assets/viewer.html` | Modify | Fix all JS bugs (P1-1 through P2-3) and add missing panels |
| `internal/repository/sprint/repository.go` | Modify (conditional) | Fix P2-2 Part B: add `status` to `ListUnassignedBacklog` query only if frontend fallback is insufficient |

No new files. No new API endpoints. No configuration changes. Follows the E27 architecture principle that viewer layer changes are isolated to these two/three files (see E27 `architecture.md` §1.1 "Files Affected" column).

### 2.2 Data Model Changes

**None.** No schema migrations, no new DTO fields, no `CurrentSchemaVersion` bump. This is confirmed by the E27 architecture doc §3.1: "E27 is purely additive at the presentation/composition layer."

The one data-availability issue (Plan tab status badges, REQ-F-004) is resolvable without schema change:

- **Primary fix (frontend fallback):** `item.status || item.status_category || 'unassigned'` in `renderSprintPlan()`. The `status_category` field is already present on `BacklogItem` structs that flow through `sprintBacklogItems()`.

- **Conditional backend fix:** The existing `ListUnassignedBacklog` query in `internal/repository/sprint/repository.go` (~line 1010) does NOT include the `status` column in its SELECT. If the frontend fallback proves insufficient (i.e., `status_category` is also absent on plan items), the task query must be amended to include `t.status`. All four entity sub-selects (tasks, bugs, change_cards, tech_debts) should be updated for consistency. The `BacklogItem.Status` field already exists in the struct (`json:"status,omitempty"`), so no model change is needed.

### 2.3 API/Interface Contracts

No new endpoints. The existing `GET /api/v1/viewer/sprint/report` response schema is unchanged — `summary` remains a nullable `*SprintSummaryResult`. The fix changes behavior: for non-completed sprints, `summary` is now explicitly `null` in the JSON response instead of the endpoint returning HTTP 500. This is backward-compatible: callers that already handle `summary: null` (including `renderSprintReport` in `viewer.html`) are unaffected.

### 2.4 Key Technical Decisions

**Decision 1: Make `GetSummary` error non-fatal in `SprintReport()`, not in `SprintAnalyticsService`.**

Rationale: The status guard in `SprintAnalyticsService.GetSummary` (`sprint summary is available for completed or archived sprints only`) is a correct business rule for that service's contract. The viewer's `SprintReport` method should tolerate a missing summary and degrade gracefully rather than changing the analytics service's behavior. This is consistent with ADR-E27-003 (read-only viewer service, thin composition) and service-design.md §4 (error wrapping with business context).

Fix location: `internal/services/viewer_service.go`, `SprintReport()` method (~line 817):
```go
// Before (propagates error as HTTP 500):
summary, err := s.sprintAnalyticsSvc.GetSummary(ctx, sprintEntity.Key, false)
if err != nil {
    return nil, fmt.Errorf("viewer sprint report: failed to load summary: %w", err)
}

// After (non-fatal, summary is nil for planning/active sprints):
summary, err := s.sprintAnalyticsSvc.GetSummary(ctx, sprintEntity.Key, false)
if err != nil {
    summary = nil // summary only available for completed/archived sprints
}
```

**Decision 2: Derive `ACTIVE_SPRINT_CATEGORIES` from `SPRINT_BUCKET_MAP` rather than a second hardcoded list.**

Rationale: The WIP counter bug (REQ-F-005) and the sidebar bucket bug (REQ-F-002) both stem from hardcoded status-name lists that diverge from the actual workflow. Introducing `SPRINT_BUCKET_MAP` as the single source of truth for status-category → display-bucket mapping (T-E27-F14-002), then deriving the WIP/Active set from it (T-E27-F14-006), prevents the two lists from drifting again. This follows CLAUDE.md's `feedback_no_hardcoded_statuses.md` memory: never hardcode status names.

**Decision 3: Two-column Plan layout uses data already in `sprintOverviewData`, no new fetch.**

Rationale: The "Assigned This Sprint" column (REQ-F-006) needs the currently assigned sprint items. These are already fetched by `renderSprintModePanels()` and available in `sprintOverviewData.backlog.groups`. Passing `overviewPayload` as a third argument to `renderSprintPlan()` reuses existing data, avoids a new API call, and keeps the fix to a single file.

**Decision 4: `formatDateOnly()` parses the ISO date string directly rather than constructing a `Date` object.**

Rationale: Sprint start/end dates are midnight UTC timestamps. Constructing `new Date(isoString)` and then calling `.getMonth()` etc. applies a local timezone offset, which can shift the displayed day by one for users east or west of UTC. Parsing `isoString.split('T')[0]` avoids this without requiring timezone-aware date arithmetic.

**Decision 5: `ListUnassignedBacklog` status column added conditionally (Part B only if needed).**

Rationale: The query is a multi-entity UNION ALL across four tables. Adding the `status` column requires amending all four sub-selects consistently. This is straightforward but is a backend change that should only happen if the frontend `status_category` fallback is insufficient. The task file for T-E27-F14-004 directs: "Start with Part A (frontend only). Only do Part B if the query confirms status is not returned." Conforming to `CLAUDE.md` patterns: do not pre-emptively modify repository code.

### 2.5 Integration with Existing Code

**`internal/services/viewer_service.go`**

- Method: `SprintReport()` at ~line 794
- Change: Replace the fatal `if err != nil { return nil, fmt.Errorf(...) }` after `GetSummary` call with `if err != nil { summary = nil }`. Three lines changed, no signature change, no new imports.
- Test impact: Existing `viewer_service_test.go` tests for completed sprints must continue to pass. A new test case must verify that a planning-sprint report call returns a non-nil `*SprintReportResponse` with `Summary == nil`.

**`internal/viewer/assets/viewer.html`**

All JS changes are within this single file. Specific functions and approximate line numbers (all in the SPA's `<script>` section):

| Task | Function(s) Modified/Added | Approximate Location |
|------|---------------------------|----------------------|
| T-E27-F14-002 | `SPRINT_BUCKET_MAP` (new const), `SPRINT_BUCKET_LABELS` (replace), `sprintBucketGroups()` | ~line 5782 |
| T-E27-F14-003 | `formatDateOnly()` (new fn), `sprintFormatRange()` | ~line 5753 |
| T-E27-F14-004 | `renderSprintPlan()` — status badge line | ~line 5957 |
| T-E27-F14-005 | `renderSprintPlan()` — restructure to two-column layout; `renderSprintModePanels()` — pass `overviewPayload`; CSS additions | ~line 5944 |
| T-E27-F14-006 | `ACTIVE_SPRINT_CATEGORIES` (new const derived from `SPRINT_BUCKET_MAP`), `renderSprintOverview()` — WIP counter and label | ~line 6181 |
| T-E27-F14-007 | `renderSprintHealthColumns()` (new fn), `renderSprintOverview()` — insert call after metrics | ~line 6181 |
| T-E27-F14-008 | `renderScopeSnapshot()` (new fn), `renderSprintOverview()` — append at end | ~line 6181 |
| T-E27-F14-009 | `renderSprintPlan()` layout/CSS — UX redesign layer on top of T-005 | ~line 5944 |

The dependency order for JS changes (dictated by `SPRINT_BUCKET_MAP` being a shared constant):
1. T-002 (`SPRINT_BUCKET_MAP`) must land first
2. T-003 (date formatter) is independent — can land in any order
3. T-004 (status badge fallback) is independent — can land in any order  
4. T-005 (two-column plan) depends on T-002, T-003, T-004
5. T-006 (Active counter) depends on T-002
6. T-007 (Capacity/Blockers panels) depends on T-005, T-006
7. T-008 (Scope Snapshot) depends on T-005, T-006
8. T-009 (UX redesign) depends on T-004, T-005

**`internal/repository/sprint/repository.go`** (conditional, Part B of T-E27-F14-004 only)

- Method: `ListUnassignedBacklog()` at ~line 995
- Change if needed: Add `t.status` to the task sub-select in the UNION ALL query (~line 1010), and add `NULL AS status` (or relevant column) to the bug, change_card, and tech_debt sub-selects for consistent column count.
- The `BacklogItem.Status string` field (~line 310) already exists and is `json:"status,omitempty"` — no struct change needed.

### 2.6 Task Execution Order

Tasks are organized in two execution waves, reflecting the `depends_on` relationships in shark:

**Wave 1 (parallel, order=1):**
- T-E27-F14-001: Go service fix (`viewer_service.go`)
- T-E27-F14-002: JS bucket map fix (`viewer.html`)
- T-E27-F14-003: JS date formatter (`viewer.html`)
- T-E27-F14-004: JS status badge fallback (`viewer.html`, optionally `repository.go`)

**Wave 2 (parallel, order=2, depends on T-002, T-003, T-004):**
- T-E27-F14-005: Plan tab two-column layout
- T-E27-F14-006: Active counter fix

**Wave 3 (parallel, order=3, depends on T-005, T-006):**
- T-E27-F14-007: Capacity/Blockers panels
- T-E27-F14-008: Scope Snapshot bar
- T-E27-F14-009: Plan tab UX redesign

---

## 3. Testing Notes

- **Go change (T-001):** Add one new test case in `viewer_service_test.go` for a `SprintReport` call where `MockSprintAnalyticsService.GetSummary` returns an error; assert `response.Summary == nil` and `err == nil`. Run `make test` after the change.
- **JS changes (T-002 through T-009):** No Go test coverage required. Manual verification against a sprint with assigned items (e.g., `S010` in the wormwoodgm project per T-E27-F14-009 notes). Acceptance criteria in §1.3 serve as the manual test checklist.
- **Conditional repository change (T-004 Part B):** If added, verify with a `ListUnassignedBacklog` call that `BacklogItem.Status` is populated for tasks. Existing repository test file at `internal/repository/sprint/repository_test.go` should cover this without a new file.
- Quality gate applies: `make fmt && make lint && make test` must pass before declaring any Go change complete.
