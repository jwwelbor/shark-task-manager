---
feature: E27-F03
title: Single-File SPA - IDE-Style Dark Dashboard Interface
reviewer: task-review-agent
date: 2026-04-11
verdict: PASS
---

# Task Review Report — E27-F03

**Feature**: E27-F03 — Single-File SPA - IDE-Style Dark Dashboard Interface
**Date**: 2026-04-11
**Verdict**: PASS

---

## 1. Sources Reviewed

| Source | Path |
|--------|------|
| Feature spec | `docs/plan/E27/.../feature.md` |
| Detailed spec | `docs/plan/E27/.../spec.md` |
| Test plan | `docs/plan/E27/.../test-plan.md` |
| UI reference | `docs/status-viewer-ui.md` |
| T-E27-F03-001 | tasks/T-E27-F03-001.md |
| T-E27-F03-002 | tasks/T-E27-F03-002.md |
| T-E27-F03-003 | tasks/T-E27-F03-003.md |
| T-E27-F03-004 | tasks/T-E27-F03-004.md |
| T-E27-F03-005 | tasks/T-E27-F03-005.md |
| T-E27-F03-006 | tasks/T-E27-F03-006.md |
| T-E27-F03-007 | tasks/T-E27-F03-007.md |

---

## 2. Task Structure and Ordering

| # | Task Key | Title | Order | Verdict |
|---|----------|-------|-------|---------|
| 1 | T-E27-F03-001 | App shell, CSS dark theme, status color mapping, Pick Folder screen | 1 | PASS |
| 2 | T-E27-F03-002 | Pick Folder screen wiring + API client module (7 fetch wrappers) | 2 | PASS |
| 3 | T-E27-F03-003 | Sidebar tree navigator with expand/collapse and selected state | 3 | PASS |
| 4 | T-E27-F03-004 | Dashboard: Status Breakdown cards + Feature Progress bars | 4 | PASS |
| 5 | T-E27-F03-005 | Dashboard: Active Transitions table + Stale Entities filter section | 5 | PASS |
| 6 | T-E27-F03-006 | Entity View: properties panel, Marked.js markdown rendering, Info/Transitions toggle | 6 | PASS |
| 7 | T-E27-F03-007 | History drawer, cross-view navigation (key clicks), Escape key, Doc View, Go embed smoke tests | 7 | PASS |

Task ordering matches spec.md §6 exactly. Dependencies are correctly sequenced (each task depends only on prior tasks). T-004 and T-006 can be developed in parallel after T-003, which is correctly noted in both task files.

---

## 3. 22 Acceptance Criteria Coverage

| # | AC from spec.md §9 | Covered By | Status |
|---|---------------------|-----------|--------|
| AC-01 | `internal/viewer/assets/viewer.html` exists as a single self-contained HTML file | T-001 (creates file), T-007 (smoke test TC-SMOKE-01) | COVERED |
| AC-02 | `internal/viewer/assets.go` embeds `viewer.html` via `go:embed` | T-007 (creates assets.go + TC-SMOKE-01, TC-SMOKE-02) | COVERED |
| AC-03 | SPA renders Pick Folder screen on initial load in dark slate theme | T-001 (renderPickFolder, dark theme CSS, TC-VIS-01) | COVERED |
| AC-04 | Clicking Pick Folder triggers hierarchy + summary + recent-activity + workflow-meta loads | T-002 (load sequence, TC-API-01) | COVERED |
| AC-05 | On successful load, SPA transitions to Dashboard view | T-002 (appState = 'dashboard', TC-NAV-01) | COVERED |
| AC-06 | Dashboard renders all 4 sections: Status Breakdown, Feature Progress, Active Transitions, Stale Entities | T-004 (sections 1+2), T-005 (sections 3+4), TC-VIS-04..07 | COVERED |
| AC-07 | Sidebar renders full Hierarchy tree with correct indentation (16/40/56px), expand/collapse, selected state | T-003 (renderSidebar, TC-TREE-01..04) | COVERED |
| AC-08 | Sidebar renders flat sections (Ideas, Change Cards, Tags, Tech Debt) when present | T-003 (buildFlatSection, TC-TREE-05) | COVERED |
| AC-09 | All status badges and dots use the exact hex colors from the mapping table | T-001 (STATUS_COLORS, getStatusColor), TC-COLOR-01..20 | COVERED |
| AC-10 | Clicking a tree entity node transitions to Entity View | T-003 (node click → appState='entity_view', TC-NAV-02) | COVERED |
| AC-11 | Entity View shows properties panel, Info/Transitions toggle, and markdown content via Marked.js | T-006 (renderEntityView, renderMarkdownPane, TC-VIS-08, TC-VIS-09) | COVERED |
| AC-12 | Clicking "Transitions" shows history drawer with reverse-chronological transitions | T-007 (renderHistoryDrawer, TC-VIS-10, TC-NAV-05) | COVERED |
| AC-13 | Clicking a doc node transitions to Doc View with plain markdown | T-007 (renderDocView, TC-NAV-04) | COVERED |
| AC-14 | Clicking an entity key in dashboard navigates to that entity (expand ancestors, scroll, select) | T-007 (navigateToEntity, TC-NAV-03) | COVERED |
| AC-15 | Escape key returns to Dashboard from Entity or Doc View | T-007 (keydown handler, TC-KB-01) | COVERED |
| AC-16 | Header "Dashboard" click returns to Dashboard | T-007 (header wiring, TC-NAV-06) | COVERED |
| AC-17 | Header "Refresh" button reloads all cached data | T-007 (Refresh handler, TC-NAV-07) | COVERED |
| AC-18 | Header "Pick Folder" button returns to Pick Folder state | T-007 (Pick Folder button, TC-NAV-08) | COVERED |
| AC-19 | Stale Entities filter excludes terminal statuses and only shows items with `updated_at > 7 days ago` | T-005 (renderStaleEntities, TC-STALE-01..03) | COVERED |
| AC-20 | SPA tolerates missing optional fields without crashing | T-004 (optional chaining guards), T-005, T-006 (TC-API-05..07) | COVERED |
| AC-21 | Marked.js CDN failure falls back to preformatted text with warning | T-006 (fallback if typeof marked === 'undefined', TC-API-08) | COVERED |
| AC-22 | `go test ./internal/viewer/...` passes (embedded file smoke tests) | T-007 (assets_test.go, 3 smoke tests, TC-SMOKE-01..03) | COVERED |

All 22 ACs are addressed across the 7 tasks.

---

## 4. Visual Spec Coverage Verification

### Indentation (status-viewer-ui.md vs. tasks)

| Node Type | Spec (status-viewer-ui.md) | T-003 Requirement |
|-----------|---------------------------|------------------|
| Epic | 16px | 16px |
| Feature | 40px | 40px |
| Task | 56px | 56px |
| Flat section node | 18px | 18px |
| Doc at epic level | 16px italic muted | 16px italic muted |
| Doc at feature level | 40px italic muted | 40px italic muted |

All indentation values match exactly.

### Terminal Status List

| Source | Terminal Statuses |
|--------|------------------|
| status-viewer-ui.md | done, cancelled, closed, wont_fix, verified, rejected, promoted, approved (8) |
| spec.md §2.2 | done, cancelled, closed, wont_fix, verified, rejected, promoted, approved (8) |
| T-E27-F03-001 `isTerminalStatus()` | done, cancelled, closed, wont_fix, verified, rejected, promoted, approved, **completed** (9) |
| T-E27-F03-005 Stale AC | done, cancelled, closed, wont_fix, verified, rejected, promoted, approved, **completed** (9) |

T-001 and T-005 add `completed` to the terminal status list. This is intentional and documented in T-001's Technical Notes: "`completed` is a valid shark workflow terminal status not listed in status-viewer-ui.md but must be included." The addition is conservative (prevents completed shark workflow tasks from appearing in stale list) and does not conflict with any stale entity test cases. The test-plan.md (TC-STALE-02) references "8 listed statuses" from status-viewer-ui.md but T-005's AC explicitly enumerates all 9. This minor discrepancy does not block implementation — the developer should implement the 9-status list per T-001/T-005 ACs.

### STATUS_COLORS Table

T-001 requires:
- All 20 statuses from status-viewer-ui.md/spec.md §2.2 — explicitly listed
- All shark long-form workflow statuses (ready_for_development, in_development, etc.) — explicitly listed
- Fallback `#666666` for unknowns via `getStatusColor()`

Complete coverage confirmed.

### Application States

| State | Source | Covered By |
|-------|--------|-----------|
| #01 Pick Folder | T-001 (static UI), T-002 (button wiring) | COVERED |
| #02 Dashboard | T-004 (sections 1+2), T-005 (sections 3+4) | COVERED |
| #03 Entity View | T-006 (properties + markdown + toggle) | COVERED |
| #04 Doc View | T-007 (renderDocView) | COVERED |

### API Client (7 endpoints)

| Endpoint | Wrapper Function | Covered By |
|----------|-----------------|-----------|
| GET /api/v1/viewer/workflow-meta | apiGetWorkflowMeta() | T-002 |
| GET /api/v1/viewer/hierarchy | apiGetHierarchy() | T-002 |
| GET /api/v1/viewer/summary | apiGetSummary() | T-002 |
| GET /api/v1/viewer/recent-activity?limit=25 | apiGetRecentActivity(limit=25) | T-002 |
| GET /api/v1/viewer/file/{key} | apiGetFile(key) | T-002 |
| GET /api/v1/viewer/history/{key} | apiGetHistory(key) | T-002 |
| GET /api/v1/viewer/features/{key}/tasks | apiGetFeatureTasks(key) | T-002 |

All 7 endpoints covered with correct function signatures and error handling.

---

## 5. Component Architecture Coverage

| # | Component (spec.md §4) | Covered By |
|---|------------------------|-----------|
| 1 | App Shell | T-001 |
| 2 | CSS Theme | T-001 |
| 3 | State Machine | T-001 (scaffold), T-007 (complete) |
| 4 | API Client | T-002 |
| 5 | Sidebar Tree | T-003 |
| 6 | Dashboard (sections 1+2) | T-004 |
| 7 | Dashboard Activity/Stale | T-005 |
| 8 | Entity View | T-006 |
| 9 | History Drawer | T-007 |
| 10 | Navigation Logic | T-007 |
| 11 | Pick Folder Screen | T-001 (static), T-002 (wired) |
| 12 | Status Color Mapping + Utilities | T-001 |

All 12 components from the spec §4 architecture table are mapped to tasks.

---

## 6. Key Behavior Coverage

| Behavior | Source | Covered By |
|----------|--------|-----------|
| Click key in dashboard → navigate to entity (expand, scroll, select) | status-viewer-ui.md §Key Behaviors Navigation | T-007 (navigateToEntity) |
| Escape closes current view | status-viewer-ui.md §Key Behaviors Keyboard | T-007 (keydown handler) |
| Epics start expanded, features start collapsed | spec.md §2.6.4 | T-003 |
| Expand/collapse preserves state on refresh | spec.md §2.6.4 | T-003 |
| scrollSidebarToKey helper | spec.md §2.6.4 | T-003 (scrollSidebarToKey), T-007 (navigateToEntity calls it) |
| Header "Dashboard" click | spec.md §2.4 | T-007 |
| Header "Refresh" button | spec.md §2.4 | T-007 |
| Marked.js CDN fallback | spec.md §2.7.3 | T-006 |
| go:embed assets.go | spec.md §4 | T-007 |

---

## 7. Dependencies Check

| Task | Declared Dependencies | Correct? |
|------|-----------------------|---------|
| T-001 | None | Correct (foundation task) |
| T-002 | T-001 | Correct |
| T-003 | T-002 | Correct |
| T-004 | T-003 | Correct |
| T-005 | T-004, T-001 (isTerminalStatus), T-002 (recentActivity) | Correct |
| T-006 | T-003 | Correct (can run parallel to T-004, T-005) |
| T-007 | T-006, T-004, T-005 | Correct (final integration task) |

Dependency graph is acyclic and complete.

---

## 8. Issues and Observations

### Minor (Non-Blocking)

1. **Terminal status list has 9 entries vs. 8 in UI spec**: T-001 and T-005 add `completed` to `isTerminalStatus()`. This is an intentional improvement documented in T-001's Technical Notes. Developer should implement the 9-status version. Test-plan TC-STALE-02 references "8 listed statuses" — the developer should verify against T-005's AC list (9 statuses) as the authoritative source.

2. **T-001 title mentions "Pick Folder screen" which might imply overlap with T-002**: The task split is clean — T-001 delivers the static Pick Folder rendering (no API), T-002 wires the button click to the API load sequence. This is correct per spec.md §6 Task 1 and Task 2 scopes.

3. **`entityViewTab` state variable**: T-006 describes this as "state variable (or local var)". For consistency with the spec §5.3 state variables, the developer should confirm whether this is a top-level state variable or a local render variable. Either approach works but should be consistent.

4. **Header wiring timing**: T-007 notes that header button wiring "may have been stubbed in earlier tasks." This creates a slight ambiguity about who owns the header wiring. The developer should be aware that T-007 is the definitive owner of header button wiring.

### No Blocking Issues Found

---

## 9. Verdict

**PASS**

All 7 tasks are present in correct execution order (1-7). All 22 acceptance criteria from spec.md §9 are addressed across the task set. All 4 application states are covered. All 7 API endpoints are covered. All visual spec requirements (indentation, color table, terminal status list, component architecture) are correctly reflected in task requirements. Dependencies are acyclic and correctly sequenced. The test plan is referenced in each task via test case IDs. Minor observations above are non-blocking and within developer discretion.

The task set is ready for implementation.
