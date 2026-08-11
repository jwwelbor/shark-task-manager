---
epic_key: E27
doc_type: uat-plan
status: draft
---

# E27 — UAT Plan: Shark Status Viewer (Local Web Dashboard)

**Epic:** E27 — Shark Status Viewer — Local Web Dashboard
**Author:** Claude (architect agent, in_design phase)
**Date:** 2026-04-11

This plan describes **what to verify** to accept E27 as "done" — not how to implement or automate the tests. It maps directly to the six success criteria in `epic.md` §Success Criteria and adds the cross-feature integration scenarios and non-functional checks that the architecture makes necessary.

Scope of UAT is the full end-to-end experience: `shark web` → browser → visible data → interaction → shutdown. Unit/integration tests at the package level are the decomposer's and developer's concern and are out of scope for this plan (they are covered by the standard per-feature acceptance criteria).

---

## 1. UAT Environment

**Required before any UAT scenario:**

- Shark built from the E27 branch: `make build` produces `./bin/shark` and `./bin/shark-task-manager` cleanly; `make fmt && make lint && make test` all green.
- A non-trivial shark project with seeded data: at least 3 epics, 10 features across them, 50 tasks across the features, 2 bugs, 2 change-cards, and at least one blocked task, one `ready_for_review` task, and one completed feature. The project used to draft E27 itself is an acceptable fixture.
- Two backends tested in turn:
  1. **Local SQLite** (`database.backend = "local"` in `.sharkconfig.json`, or absent).
  2. **Turso cloud** (`database.backend = "turso"` with a valid URL + token file per `docs/TURSO_QUICKSTART.md`).
- A modern Chromium-based browser (Chrome/Edge/Brave/Arc) on Linux or macOS. Firefox is a stretch target for Phase 1.
- Network offline for at least one scenario (to prove ADR-E27-006 — no CDN dependency).

---

## 2. UAT Scenarios

Each scenario is **one user-observable outcome**. Scenarios are numbered by feature area; within each area they follow the natural "happy path → edge case → failure mode" ordering.

### Area A — Startup and Discovery

#### A1. Starts from the project root
**Given** a shark project at `/path/to/proj` with a populated database,
**when** the user runs `shark web` from `/path/to/proj`,
**then** a URL `http://127.0.0.1:7777` is printed within 2 seconds of issuing the command,
**and** the default browser opens to that URL (on a desktop session with a TTY),
**and** the dashboard renders the project's epics in the sidebar without user interaction.

*Success criterion mapping:* "shark web starts server and opens browser within 2 seconds".

#### A2. Starts from a subdirectory
**Given** the same project,
**when** the user runs `shark web` from `/path/to/proj/docs/plan/E01-foo/`,
**then** the viewer opens and displays the same data as in A1 — no "project root not found" error, no empty dashboard.

*Success criterion mapping:* "Works from any subdirectory in a shark project".

#### A3. Explicit port
**Given** an otherwise idle machine,
**when** the user runs `shark web --port 9100`,
**then** the server binds to `127.0.0.1:9100` and prints the corresponding URL,
**and** port 7777 is not touched.

#### A4. Port 7777 in use, auto-increment
**Given** port 7777 is already bound by another process,
**when** the user runs `shark web` without `--port`,
**then** the server falls back to the next free port in the range 7778–7790 and prints the actual URL.

#### A5. Explicit port already in use — strict failure
**Given** port 9100 is already bound,
**when** the user runs `shark web --port 9100`,
**then** the command exits non-zero within ~1 second with a clear error message mentioning the port,
**and** no server is left listening.

#### A6. Headless / SSH mode
**Given** the command is invoked over SSH (no GUI session, no `DISPLAY`),
**or** with `--no-open`,
**when** `shark web --no-open` is run,
**then** the URL is printed,
**and** the server serves requests,
**and** no browser launch is attempted (no error from `xdg-open`).

#### A7. Graceful shutdown
**Given** `shark web` is running and serving requests,
**when** the user presses `Ctrl-C`,
**then** in-flight HTTP requests complete (up to the 30-second shutdown timeout),
**and** the process exits cleanly with exit code 0,
**and** the TCP port is released immediately (no `bind: address already in use` on an immediate re-run).

### Area B — Dashboard Overview

#### B1. Summary cards reflect the database
**Given** the dashboard view,
**when** it first loads,
**then** summary pill cards show counts for epics, features, tasks, bugs, and change-cards that exactly match the counts `shark epic list --json | jq length`, etc. return from the CLI against the same database.

*Success criterion mapping:* "Single source of truth" — viewer and CLI must never disagree.

#### B2. Status breakdowns use workflow colors
**Given** the summary view,
**then** every status badge is rendered with the color declared in `.sharkconfig.json`'s `status_metadata`, not a hardcoded default,
**and** phases are labeled correctly (planning / development / review / qa / done).

#### B3. Feature progress bars
**Given** a feature with, e.g., 4 tasks of which 2 are `completed`, 1 `in_progress`, 1 `todo`,
**when** the viewer shows that feature in the feature list,
**then** the weighted-progress number matches what `shark feature get E##-F## --json | jq .progress` returns.

#### B4. Recent activity feed
**Given** the dashboard,
**when** it first loads,
**then** a "Recent activity" panel shows up to N (default 50) most recent status changes across all entity types, newest first,
**and** the timestamps are localized or rendered in a format that is unambiguous (e.g., ISO-8601 or "2 minutes ago" with tooltip).

### Area C — Sidebar Navigation (Hierarchy)

#### C1. Tree renders all epics and features
**Given** the viewer opens,
**when** the left sidebar loads,
**then** every epic in the database appears as a top-level node,
**and** expanding an epic reveals every feature belonging to it,
**and** feature nodes show a task-count badge matching `SELECT count(*) FROM tasks WHERE feature_id = ...`.

#### C2. Status dots are colored
**Given** an epic or feature in the sidebar,
**then** a status dot next to its title is colored according to the workflow config (per B2).

#### C3. Collapsible state is preserved across tab switches
**Given** the user expands epics E01 and E03 but leaves E02 collapsed,
**when** they click a tab (Epics → Tasks) and return to Epics,
**then** the same expand/collapse state is preserved.

#### C4. Clicking a node opens the entity view
**Given** the sidebar,
**when** the user clicks a feature node,
**then** the main panel switches to the entity view for that feature (see Area D).

#### C5. Hierarchy performance with 500 tasks
**Given** a project with 500 tasks across ~50 features across ~10 epics,
**when** `shark web` starts and the sidebar first loads,
**then** the hierarchy API call completes in under 500 ms on the developer's workstation,
**and** the tree renders within a perceptible instant after the network request resolves.

*Success criterion mapping:* "Hierarchy loads in < 500ms for projects with 500 tasks".

### Area D — Entity View

#### D1. Spec markdown renders
**Given** the user clicks on any entity that has a `file_path` pointing to an existing markdown file on disk,
**when** the entity view opens,
**then** the markdown content is rendered as **formatted HTML** — headings, lists, code blocks, tables — not raw markdown,
**and** Marked.js is loaded from `/static/vendor/marked.min.js` (same-origin, not a CDN — see Area G).

*Success criterion mapping:* "Spec documents render as formatted markdown".

#### D2. Missing file graceful fallback
**Given** an entity whose DB record references a `file_path` that does not exist on disk (e.g., file was manually deleted),
**when** its entity view opens,
**then** the viewer shows a clear "Spec file not found on disk" message with the expected path,
**and** does **not** 500, crash, or show a stack trace.

#### D3. Properties panel
**Given** any entity view,
**then** a properties panel shows at minimum: key, title, status (with color), phase, created-at, updated-at,
**and** for features: execution order, weighted progress, task count;
**and** for tasks: agent type (if any), priority, execution order, blocked flag;
**and** for bugs: severity.

#### D4. History button
**Given** the entity view,
**when** the user clicks "History",
**then** the transition history view opens for the same entity (Area E).

### Area E — Transition History

#### E1. History for any entity type
**Given** any of: epic (`E07`), feature (`E07-F01`), task (`E07-F01-001`), bug (`B001`), change-card (`CC-001`),
**when** the history view opens for that entity,
**then** a reverse-chronological table shows every recorded status change,
**and** from/to status cells render with the workflow colors,
**and** the agent (if any) and timestamp are shown.

*Success criterion mapping:* "Transition history shows all entries with status badges".

#### E2. Parity with CLI
**Given** any entity with a history,
**then** the rows shown match — entry-for-entry, in the same order — what `shark status history <key>` prints.

#### E3. Empty history
**Given** a freshly-created entity with no status changes yet,
**then** the history view shows an empty-state message, not a blank panel or an error.

### Area F — Database Backend Support

#### F1. Local SQLite
**Given** `.sharkconfig.json` has `database.backend = "local"` (or is absent),
**when** `shark web` runs,
**then** every scenario in Areas A–E passes against the local database.

#### F2. Turso cloud
**Given** `.sharkconfig.json` has `database.backend = "turso"` with a valid URL and auth-token file,
**when** `shark web` runs,
**then** every scenario in Areas A–E passes against the Turso database,
**and** the viewer shows exactly the same data a cold-start CLI command (`shark epic list`) returns against the same Turso URL.

*Success criterion mapping:* "Works with both local SQLite and Turso cloud".

#### F3. Misconfigured backend
**Given** a `.sharkconfig.json` with `database.backend = "turso"` but an unreachable URL,
**when** `shark web` runs,
**then** the command exits non-zero with a clear error message naming the backend and the underlying cause (e.g., "failed to connect to libsql://..."),
**and** the server does not leave a listener behind.

### Area G — Offline / No-CDN

#### G1. Fully offline
**Given** the machine has no network connectivity (Wi-Fi off, `/etc/hosts` blocking external DNS),
**when** `shark web` runs and the user opens the viewer in a browser,
**then** the SPA loads, the dashboard renders, markdown renders correctly, and no request in the browser's network panel targets any third-party host,
**and** the only requests the browser makes are to `127.0.0.1:<port>/...`.

*ADR mapping:* ADR-E27-006 (Marked.js served same-origin).

#### G2. CDN block sanity check
**Given** a browser DevTools network panel open on first viewer load,
**then** the requests list contains only paths starting with `/` (no `https://cdn.*`, no `https://unpkg.*`, no third-party domain whatsoever).

### Area H — Security Boundaries

#### H1. Bind address is localhost only
**Given** `shark web` is running,
**when** a second machine on the same LAN attempts `curl http://<dev-ip>:7777/`,
**then** the connection is refused (server did not bind `0.0.0.0`).

#### H2. CORS rejects non-local origins
**Given** the server is running,
**when** a request is sent with `Origin: https://evil.example.com`,
**then** the response does not include an `Access-Control-Allow-Origin` header matching that origin,
**and** a browser reading that response would block cross-origin access.

*ADR mapping:* ADR-E27-007.

#### H3. CORS accepts localhost origins on any port
**Given** the server is running,
**when** a request comes in with `Origin: http://localhost:5173`,
**then** the response `Access-Control-Allow-Origin` echoes `http://localhost:5173`.

#### H4. File endpoint resolves from DB only (path-traversal probe)
**Given** a malicious key attempt like `GET /api/v1/viewer/file/..%2F..%2F..%2Fetc%2Fpasswd`,
**when** the request is made,
**then** the response is a 400 or 404 with no file content,
**and** the response body contains no system-path strings.

*ADR mapping:* ADR-E27-008.

#### H5. File endpoint refuses paths outside project root
**Given** a hand-crafted DB row where `file_path` points outside the project root (e.g., via symlink),
**when** `GET /api/v1/viewer/file/<that_key>` is hit,
**then** the response is HTTP 403 and the server log contains a security warning,
**and** no file content is returned.

#### H6. No write endpoints under `/api/v1/viewer/`
**Given** the server is running,
**when** any `POST`, `PUT`, `PATCH`, or `DELETE` is attempted against any `/api/v1/viewer/*` path,
**then** the response is 405 Method Not Allowed (or 404),
**and** nothing in the database is modified.

*ADR mapping:* ADR-E27-003 (read-only).

### Area I — Cross-Feature Integration

These scenarios exercise more than one of the four features at once and are the most important ones for the overall epic sign-off.

#### I1. Full loop against a live project
**Given** the user cold-starts `shark web` in a project they have not previously opened,
**when** they click through: Dashboard → sidebar expand → feature click → entity view → History button → back → task click,
**then** every transition completes without an error,
**and** the displayed data at every step matches what the equivalent CLI commands return (spot-check at least three entities).

#### I2. Edit with CLI, refresh in browser
**Given** the viewer is showing feature E07-F01,
**when** the user runs `shark task next-status E07-F01-001` in a separate terminal,
**then** clicking Refresh in the viewer displays the updated task status within 1 second,
**and** the history view for E07-F01-001 shows the new transition at the top.

*This is the single most important cross-feature scenario — it proves the dbinit extraction, the API, the SPA, and the CLI all agree on the same source of truth.*

#### I3. Two viewer tabs, one database
**Given** two browser tabs opened to the same `http://127.0.0.1:<port>`,
**when** the user clicks through different entities in each tab independently,
**then** neither tab corrupts the other's state (no shared state leakage),
**and** both tabs reflect the same underlying DB rows.

#### I4. Server stays up through repeated SPA reloads
**Given** a user browser-reloads the viewer 20 times in succession,
**then** the server does not leak goroutines or FDs (check via `ss -tnp` and `ps` for a simple sanity threshold),
**and** every load renders correctly.

### Area J — Accessibility and Usability (baseline)

These are not strict pass/fail gates — they are the "nothing is obviously broken" smoke checks a reviewer should do once.

#### J1. Keyboard navigation
**When** the user tabs through the sidebar tree and the top-nav,
**then** focus indicators are visible,
**and** arrow keys move between sibling tree nodes,
**and** `Escape` closes any open panel.

#### J2. No console errors
**Given** the viewer is loaded and the user clicks through 5 entities,
**then** the browser DevTools Console shows zero JavaScript errors and zero failed network requests (except intentional 404s for missing spec files, which are expected and handled).

#### J3. Readable at 1280×800
**Given** a browser window of 1280×800 (common laptop resolution),
**when** the viewer is open,
**then** no text is clipped, no panels overlap, and the sidebar, main panel, and top nav are all usable without horizontal scrolling.

### Area K — Grouped Nav & Browsable Folders (E27-F16)

#### K1. Fixed group order and styling
**Given** the dashboard is loaded with zero and then two configured browsable folders,
**when** the sidebar renders,
**then** Plan, Architecture, and Product are the first three groups in that order,
**and** configured groups follow them with the distinct group header styling.

#### K2. Group collapse preserves nested state
**Given** Plan is expanded and a nested section has its own collapse state,
**when** Plan is collapsed and re-expanded,
**then** the other groups are unaffected and the nested section retains its state.

#### K3. Plan contents remain complete
**Given** a project with every tracked entity family and an active sprint,
**when** sprint mode is entered,
**then** the sprint tree and all six tracked-entity sections are inside Plan,
**and** an empty entity section remains omitted.

#### K4. Built-in folder browsing
**Given** Architecture exists and Product is absent in a fixture,
**when** each built-in folder entry is opened,
**then** the existing folder view lists Architecture and renders Product's empty result without an error toast.

#### K5. Group persistence and toggle-all
**Given** one group, one nested section, and one configured folder group are collapsed,
**when** the page reloads and then the sidebar toggle-all control is used,
**then** persisted state is restored and toggle-all expands and collapses every group and section,
**and** the same interactions remain usable when localStorage throws.

#### K6. Configured folder rendering
**Given** configured folders with an omitted label, a missing path, and duplicate basenames on distinct paths,
**when** the sidebar renders,
**then** labels default from the basename, missing folders are marked unavailable but remain browseable, and distinct paths have distinct groups.

#### K7. Nav-folder endpoint degradation
**Given** the nav-folders endpoint returns 500 before one load and succeeds before the next,
**when** the dashboard loads,
**then** the failed load still shows exactly the built-in Architecture and Product groups without an error toast,
**and** the successful load does not duplicate them.

#### K8. Standalone Docs entry retained (provisional)
**Given** an existing stored collapse preference for Docs,
**when** the grouped sidebar loads,
**then** Docs remains outside every group with its `folder:docs` entry and honors that stored preference.

#### K9. Existing sidebar controls remain unchanged
**Given** the grouped sidebar,
**when** the user toggles show-all items, show-all files, the tree-only collapse-all control, and a tag chip,
**then** each behavior remains as specified by E27-F10 and E28-F06,
**and** tree-only collapse-all does not change group or section state.

#### K10. Navigation metadata is non-blocking
**Given** browser network tools are open during dashboard startup,
**when** the dashboard loads,
**then** nav-folders runs outside the hierarchy critical path and does not affect the Area C5 hierarchy performance measurement.

---

## 3. Performance Considerations

| Scenario | Target | Measurement |
|---|---|---|
| `shark web` → URL printed | ≤ 2 s | wall clock from keystroke to first printed line |
| Hierarchy endpoint at 500 tasks | ≤ 500 ms server-side | handler log timing, or devtools Network panel `GET /api/v1/viewer/hierarchy` duration |
| Summary endpoint | ≤ 200 ms | devtools |
| History endpoint (any single entity) | ≤ 150 ms | devtools |
| File endpoint (10 KB markdown) | ≤ 100 ms | devtools |
| SPA initial paint after all endpoints resolve | ≤ 150 ms | devtools Performance panel |
| Memory growth over 20 reloads | < 5 MB RSS delta | `ps -o rss= -p $PID` before/after |

These are the developer-workstation targets called out in the epic PRD. If a scenario reliably exceeds a target on a modest machine (e.g., an 8 GB ultrabook), it's a bug.

---

## 4. Security Considerations Recap

The architecture puts these guarantees in place; UAT Areas H1–H6 are the verification points:

1. **Localhost-only bind.** Verified in H1.
2. **Safe CORS.** Verified in H2, H3.
3. **Read-only surface.** Verified in H6.
4. **Path-traversal defense.** Verified in H4, H5.
5. **No third-party runtime.** Verified in G1, G2.
6. **No new mutation paths.** Architecture ADR-E27-003; no test needed beyond H6 confirming 405s.

If any of H1–H6 or G1–G2 fail, the epic is blocked regardless of functional scenario results. These are hard gates.

---

## 5. Success Criteria → Scenario Traceability

Every success criterion in `epic.md` maps to at least one UAT scenario. Nothing should be "covered by vibes".

| Epic success criterion | Scenarios |
|---|---|
| `shark web` starts server and opens browser within 2 seconds | A1, A2 |
| Hierarchy loads in < 500ms for projects with 500 tasks | C5 + Performance table row |
| Spec documents render as formatted markdown | D1, D2 |
| Transition history shows all entries with status badges | E1, E2, E3 |
| Works from any subdirectory in a shark project | A2 |
| Works with both local SQLite and Turso cloud | F1, F2, F3 |

And the architecture-introduced guarantees that are not in the original success-criteria list but must also be verified:

| Architecture guarantee | Scenarios |
|---|---|
| Read-only viewer surface (ADR-003) | H6 |
| Offline-capable SPA (ADR-006) | G1, G2 |
| Safe CORS (ADR-007) | H2, H3 |
| File-endpoint containment (ADR-008) | H4, H5 |
| In-process CLI runner (ADR-009) | A1, A7 |
| Bounded port auto-increment (ADR-010) | A3, A4, A5 |
| Graceful headless fallback (ADR-011) | A6 |
| Data parity with CLI | B1, B3, E2, I1, I2 |

---

## 6. Sign-Off Checklist

The epic is ready to move from `in_feature_review` to `active` when, on **both** local SQLite and Turso:

- [ ] Every scenario in Areas A–H passes (A1–A7, B1–B4, C1–C5, D1–D4, E1–E3, F1–F3, G1–G2, H1–H6).
- [ ] Every cross-feature scenario I1–I4 passes.
- [ ] All Section 3 performance targets met on a developer workstation.
- [ ] `make fmt && make lint && make test` are green on the epic branch.
- [ ] No new dependency was added to `go.mod` (architecture guarantee).
- [ ] No database migration was introduced (architecture guarantee; `CurrentSchemaVersion` unchanged).
- [ ] Section 5 traceability table has no orphaned success criteria.

Scenarios in Area J (accessibility/usability) are "strongly recommended but non-blocking" for Phase 1 — failures there produce follow-up tasks, not an epic rejection.

---

*UAT plan sign-off pending feature review gate.*
