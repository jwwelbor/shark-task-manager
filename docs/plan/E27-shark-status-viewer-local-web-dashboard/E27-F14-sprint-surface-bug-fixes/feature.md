---
feature_key: E27-F14-sprint-surface-bug-fixes
epic_key: E27
title: Sprint Surface Bug Fixes
description: Fix bugs and wireframe alignment gaps discovered during post-implementation review of E27-F13. Covers P0 report crash, P1 sidebar/plan/overview structural gaps, and P2 polish issues.
size: M
---

# Sprint Surface Bug Fixes

**Feature Key**: E27-F14  
**Epic**: E27 — Shark Status Viewer  
**Source**: Post-implementation review of E27-F13 against wireframes in `sprint-planning-web-ui-wireframes.md`

---

## Goal

### Problem

The E27-F13 sprint mode implementation shipped with several bugs and wireframe alignment gaps discovered during a Playwright-assisted review. The Report tab crashes with a 500 for any non-completed sprint. The sidebar sprint tree shows 0 items in all buckets despite real sprint data existing. The Plan tab is missing its "Assigned This Sprint" column. The Overview is missing its Capacity/Readiness and Blockers/Risks panels. Date ranges show time components. Status badges on Plan tab items show "unknown".

### Solution

Eight targeted fixes across the Go viewer service and the viewer.html SPA, grouped by the priority order established in the UX gap analysis. All fixes are isolated to `internal/viewer/assets/viewer.html` (JS) and `internal/services/viewer_service.go` (Go). No new API endpoints needed.

### Impact

- Report tab becomes usable for all sprint statuses, not just completed sprints
- Sidebar sprint tree correctly reflects live sprint state (bucket counts, item lists)
- Plan tab shows both candidate backlog and currently assigned scope side by side
- Overview surfaces capacity pressure and blockers without navigating to Plan
- Date display is clean and readable across all sprint surface views

---

## Bug Inventory (from UX gap analysis)

| ID | Priority | Layer | Description |
|----|----------|-------|-------------|
| P0-1 | P0 | Go | Report tab 500 for planning/active sprints |
| P1-1 | P1 | JS | Sidebar bucket key mismatch — all buckets show 0 |
| P1-2 | P1 | JS | Plan tab missing "Assigned This Sprint" column |
| P1-3 | P1 | JS | Overview missing Capacity/Readiness + Blockers/Risks panels |
| P1-4 | P1 | JS | Overview missing Assigned Scope Snapshot bar |
| P2-1 | P2 | JS | Date range shows HH:MM:SS — should be date-only |
| P2-2 | P2 | JS | WIP counter excludes active workflow categories |
| P2-3 | P2 | JS | Plan tab status badges show "unknown" |

---

## Acceptance Criteria

- [ ] `GET /api/v1/viewer/sprint/report` returns 200 for planning and active sprints with available data and nil summary
- [ ] Sidebar "Active Sprint" total and bucket counts match actual sprint assignment counts
- [ ] Sprint date ranges display as "May 8 → May 22" with no time component
- [ ] Plan tab renders two sections: Candidate Backlog (unassigned) and Assigned This Sprint (scoped)
- [ ] Plan tab items show correct status badges (not "unknown")
- [ ] WIP metric card counts items in `in_progress`, `ready_for_approval`, and `ready_for_development`
- [ ] Overview renders Capacity by Agent Type and Blockers/Risks panels
- [ ] Overview bottom shows Assigned Scope Snapshot status distribution

---

## Out of Scope

Per wireframe "Deferred interactions" — not part of this feature:
- Drag-and-drop prioritization
- Inline capacity editing
- Multi-sprint comparison
- Plan tab Priority/Size/Dependency filters and search box (P2-4)
- Error state retry button (P2-6)
- Empty planning state screen (P2-7)
- Cycle Time by Phase and Agent Utilization in Report tab

---

## Files Affected

- `internal/services/viewer_service.go` — SprintReport, GetSummary error handling
- `internal/viewer/assets/viewer.html` — all JS rendering functions

*Last Updated*: 2026-05-08
