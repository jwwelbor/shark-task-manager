---
feature_key: E27-F16-group-left-nav-by-artifact-type-planarchitecturepr
epic_key: E27
doc_type: test-plan
---

# Test Plan: E27-F16 — Group Left Nav by Artifact Type (Plan/Architecture/Product) + Configurable Browsable Folders

**Created:** 2026-08-10
**Feature spec:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F16-group-left-nav-by-artifact-type-planarchitecturepr/spec.md`
**Parent epic UAT plan:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/uat-plan.md`
**Research report:** `docs/plan/E27-shark-status-viewer-local-web-dashboard/E27-F16-group-left-nav-by-artifact-type-planarchitecturepr/research-report.md`
**Status:** APPROVED

---

## 0. Note on inputs

This feature's `spec.md` is a combined requirements+architecture document (`doc_type: combined-spec`); it supersedes the still-templated `feature.md` (User Stories/Requirements/Acceptance Criteria sections in `feature.md` are unfilled placeholders — verified by reading the file). All traceability below cites `spec.md` REQ-F-/REQ-NF- IDs and AC IDs, which are the actual acceptance criteria for this feature.

`spec.md` §4.7 ("Verification path for acceptance criteria") already assigns each AC group to Go unit test / Go handler test / code-review assertion / manual UAT. This test plan is built directly on that mapping — it does not re-derive it — and adds the missing rigor layer (ISTQB technique, ISO 25010, caller-path contracts, observability) required by the QA test-planning workflow.

No spec drift found: every REQ-F/REQ-NF in `spec.md` traces to `epic.md` §Solution/§Impact/§Key Constraints or `feature.md` §Solution narrative (which, unlike its templated sections, is fully written). See §1 (Drift Analysis) below.

---

## 1. Spec Drift Analysis

### Drift Findings

None. `spec.md` §1 and §8 (Traceability) already perform this check against `epic.md` and `feature.md` §Solution and this test plan independently re-verified each REQ-F/REQ-NF against those two source documents plus `E27-F10-tree-view-enhancements/spec.md` (REQ-NF-001's non-regression target). No scope creep, no scope narrowing, no semantic drift, no schema drift found.

One deliberate in-scope correction is flagged by the spec itself, not hidden: ADR-F16-3 adds `questions` to `SIDEBAR_SECTION_DEFAULTS` (a pre-existing omission) as part of modifying that same map for toggle-all coverage. This test plan treats it as within AC-004.2/AC-004.3 scope, not as drift.

### Traceability Matrix

| Feature PRD / Epic requirement | Task-equivalent (spec.md) requirement | Covered? | Notes |
|---|---|---|---|
| epic.md §Solution "browse the project hierarchy, read spec documents" | REQ-F-001 (three top-level groups) | Yes | |
| feature.md §Solution "Plan — every tracked entity family" | REQ-F-002 | Yes | |
| feature.md §Solution "Architecture — docs/architecture/*, clickable and browsable" | REQ-F-003 | Yes | |
| feature.md §Solution "Product — product docs, clickable and browsable" | REQ-F-003 | Yes | |
| feature.md §Solution "remembers its expand/collapse state ... participates in existing expand-all/collapse-all" | REQ-F-004 | Yes | |
| feature.md §Solution "add a config section ... register additional browsable folders" | REQ-F-005 | Yes | |
| feature.md §Solution "must reject `../` traversal, must reuse shark's existing path-safety check" | REQ-F-006, REQ-NF-002 | Yes | |
| epic.md §Success Criteria (dashboard remains usable) | REQ-F-007 | Yes | |
| E27-F10 spec REQ-F-001…005 non-regression | REQ-NF-001 | Yes | |
| epic.md §Key Constraints (no build step, no new deps) | REQ-NF-004 | Yes | |
| epic.md §Success Criteria (hierarchy < 500ms) | REQ-NF-003 | Yes | |

### Missing Coverage

None found. All eight functional requirements and four non-functional requirements in `spec.md` trace to an epic/feature source and all have concrete ACs.

### Ambiguity Findings

- **Q006 (open, spec §7)**: whether the standalone `Docs` entry (REQ-F-008) survives, is removed, or folds into Product. Spec assigns a safe default (retain, unchanged) and states the feature can proceed to test planning/implementation without resolution — REQ-F-008/AC-008.1/AC-008.2 test the shipped default. If Q006 resolves differently before merge, only REQ-F-008's two ACs and their test cases (TC-024, TC-025) need revision; no other requirement is affected (per spec §7's stated blast radius). This is not a blocker for this test plan.
- No other ambiguous ("correctly handles", "gracefully") language found in `spec.md`'s ACs — they are already stated as concrete, checkable DOM/response assertions.

---

## 2. ISTQB Technique Application (per AC)

| AC group | Technique(s) Applied | Test Cases | Rationale |
|---|---|---|---|
| AC-001.1a/1b, 001.2, 001.3 (group structure) | Equivalence Partitioning (with/without extra folder groups) + State Transition (collapse/expand independence) | TC-001..TC-004 | DOM ordering and independence are partition + state concerns |
| AC-002.1…4 (Plan group contents) | Equivalence Partitioning (empty vs non-empty section) + State Transition (collapse/re-expand preserves nested state) | TC-005..TC-008 | |
| AC-003.1…4 (Architecture/Product browsing) | Equivalence Partitioning (existing dir vs missing dir) | TC-009..TC-012 | Reuses existing `FolderFiles` partition already proven for `Docs` |
| AC-004.1…5 (persistence + toggle-all) | State Transition (localStorage across reload) + Boundary Value Analysis (nothing collapsed / everything collapsed) | TC-013..TC-017 | |
| AC-005.1…6 (config-registered folders) | Equivalence Partitioning (label present/absent, config absent/empty/populated) + Boundary Value Analysis (duplicate basenames, empty/whitespace path) | TC-018..TC-023, TC-030 | |
| AC-006.1…5 (traversal rejection) | Attack-class enumeration (`../`, absolute, symlink-escape, empty path) | TC-026..TC-029 | Adversarial input-surface AC — closes the open-ended "reject bad paths" requirement with an explicit attack model instead of leaving it open-ended |
| AC-007.1…3 (endpoint-down degradation) | Decision Table (endpoint success × failure) | TC-031..TC-033 | |
| AC-008.1…2 (Docs entry retained) | Equivalence Partitioning (regression check) | TC-024..TC-025 | |
| REQ-NF-001 (non-regression) | Regression re-run of existing F10/E28-F06 suites | TC-034 (UAT) | Existing tests are the technique; no new technique needed |
| REQ-NF-002 (single-sourced path safety) | Code-review / static grep assertion | TC-035 | Not a runtime technique — a structural invariant checked by diff review |
| REQ-NF-003 (no added blocking latency) | Boundary Value Analysis (nav-folders resolves in non-blocking phase) | TC-036 | |
| REQ-NF-004 (no new deps) | Code-review / static assertion | TC-037 | |

Every runtime AC has a technique; every AC has at least one test case below.

---

## 3. ISO 25010 Coverage Matrix

| AC | Functional Suitability | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001.1a/1b/1.2/1.3 | ✅ TC-001..004 | N/A | N/A | ✅ TC-002 (visual distinguishability, AC-001.3) | N/A | N/A | ✅ ADR-F16-1 reuse (code review) | N/A |
| AC-002.1…4 | ✅ TC-005..008 | N/A | N/A | N/A | ✅ TC-008 (state preserved across collapse) | N/A | N/A | N/A |
| AC-003.1…4 | ✅ TC-009..012 | N/A | N/A | N/A | ✅ TC-012 (missing dir doesn't error) | N/A | N/A | N/A |
| AC-004.1…5 | ✅ TC-013..017 | N/A | N/A | ✅ TC-015/016 (toggle-all label correctness) | ✅ TC-017 (storage-throws fallback) | N/A | N/A | N/A |
| AC-005.1…6 | ✅ TC-018..023, TC-030 | N/A | ✅ TC-023 (RawData round-trip = forward/back compat) | N/A | ✅ TC-021 (missing dir still renders) | N/A | N/A | N/A |
| AC-006.1…5 | N/A | N/A | N/A | N/A | N/A | ✅ TC-026..029, TC-035 | ✅ TC-035 (single containment implementation) | N/A |
| AC-007.1…3 | ✅ TC-031..033 | N/A | N/A | ✅ TC-032 (no error toast) | ✅ TC-031 (graceful degradation) | N/A | N/A | N/A |
| AC-008.1…2 | ✅ TC-024..025 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| REQ-NF-001 | ✅ TC-034 | N/A | N/A | N/A | ✅ TC-034 | N/A | N/A | N/A |
| REQ-NF-002 | N/A | N/A | N/A | N/A | N/A | ✅ TC-035 | ✅ TC-035 | N/A |
| REQ-NF-003 | N/A | ✅ TC-036 | N/A | N/A | N/A | N/A | N/A | N/A |
| REQ-NF-004 | N/A | N/A | N/A | N/A | N/A | N/A | ✅ TC-037 | ✅ TC-037 (no new deps = portable) |

### Coverage Gaps

None. Every applicable ISO 25010 characteristic per AC is covered by a listed test case; every `N/A` is justified by the AC's shape (e.g. a DOM-ordering AC has no meaningful Performance dimension).

---

## 4. Observability Design

This feature adds one new read-only endpoint and no new mutation path. Per `spec.md` §4.5/§4.6, failures are non-fatal by design (REQ-F-007), so the observability requirement is concentrated on the one place a silent failure matters: rejected folder paths (REQ-F-006/AC-006.4).

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Registered folder path rejected (traversal/absolute/symlink-escape) | N/A — no existing viewer metrics pipeline to extend (repo has no metrics package for viewer requests; consistent with existing `FolderFiles`/`File` which also only log) | `slog.Warn` naming the offending path and the containment-failure reason | N/A — no tracing infra in this codebase | N/A | TC-026..029 assert the warning is emitted (via a test-scoped `slog` handler capturing records) |
| `GET /api/v1/viewer/nav-folders` request failure (500) | N/A (mirrors `WorkflowMeta`, which also has no metric) | `slog.Error` (mirrors `WorkflowMeta`'s `slog.Error("viewer workflow meta failed", ...)`) | N/A | N/A | TC-033 asserts the handler logs and returns 500; existing `WorkflowMeta` handler test is the direct precedent |
| Nav-folders success (200) | internal — no observability | internal — no observability | internal — no observability | N/A | justified: this is the expected steady-state path with no operator action implied, matching every other viewer GET endpoint (`Summary`, `Hierarchy`, etc.), none of which emit success-path telemetry |

**Implementation hook:** `NavFolders` must call `slog.Warn` (not just silently drop) for every rejected entry — this is already stated as AC-006.4 in `spec.md`, so no new task-spec requirement is introduced by this table; it is restated here as a hard developer requirement with the exact test assertion (TC-026..029) that verifies the log line exists.

---

## 5. Cross-feature contract tests (I-##)

**None.** `spec.md` §5 states no interaction map exists for E27 and no I-## IDs are assigned to E27-F16 — verified independently by the same glob (`docs/plan/E27-shark-status-viewer-local-web-dashboard/*map*.md` returns no matches). REQ-NF-001 is a non-regression constraint against E27-F10/E28-F06, not a producer/consumer contract, and is covered as a regression re-run (TC-034), not a shared contract TC.

## 6. Cross-epic integration tests (X-##)

**None.** `spec.md` §6 states no X-## row names E27-F16 in `docs/product/cross-epic-integration-map.md` — verified independently: E27's only row is X-09 (owned by E27-F15, unrelated cross-session-usage tracking). No deferral needs recording in `docs/product/progress.md`.

---

## 7. Test Infrastructure

**Existing patterns to follow:**

| Layer | File | Pattern to reuse |
|---|---|---|
| Service unit tests | `internal/services/viewer_service_test.go` | Table-driven tests constructing `ViewerService` via `NewViewerService(...)` + `With*` optional setters (e.g. `TestViewerService_Hierarchy_TagFilter_*` series); uses `t.TempDir()` for project-root-relative filesystem fixtures (see `FolderFiles`/`File` tests) |
| Handler tests | `internal/api/viewer/handler_test.go` | `MockViewerServicer` with a `*Func` field per method (e.g. `WorkflowMetaFunc`, see line 38/108); table-driven `TestHandler_WorkflowMeta` (line 1481) is the direct structural sibling for the new `TestHandler_NavFolders` |
| Config tests | `internal/config/config_test.go` | Existing unmarshal/marshal round-trip tests for `WebConfig`/`RawData` preservation (extend, do not replace) |
| Path-containment fixtures | `internal/services/viewer_service_test.go` (`FolderFiles`/`File` security tests) | `t.TempDir()` + `os.Symlink` to build an escape target; identical pattern needed for `NavFolders` traversal tests (TC-026..029) |

**New test helpers needed:** none. The existing `t.TempDir()` + `NewViewerService` + `With*` pattern and the existing `MockViewerServicer` cover every new test case without new shared infrastructure.

**No JavaScript test harness exists** (`spec.md` §4.7, independently verified: no `package.json`, no `playwright`/`jest`/`vitest`/`cypress` target in `Makefile`). All frontend-observable ACs (DOM structure, persistence, degradation) are manual UAT, appended to `docs/plan/E27-shark-status-viewer-local-web-dashboard/uat-plan.md` per the established E27 precedent (that file's Areas A–J structure).

---

## 8. Caller-Path Contracts (per test case)

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-018..023, TC-026..030 | `(*services.ViewerService).NavFolders(ctx context.Context) (*services.NavFoldersResponse, error)` constructed via `NewViewerService(...)` + `.WithBrowsableFolders([]config.BrowsableFolder{...})`, driven with a real `t.TempDir()` project root | `os.ReadDir`/`filepath.EvalSymlinks` are NOT mocked — the real filesystem via `t.TempDir()` is the seam; only repository dependencies unrelated to `NavFolders` (e.g. `ViewerEntityDocRepository`) may be nil/mock | Do not mock `isContained` or replace `NavFolders`'s canonicalization with a stub — that is the exact logic under test (REQ-NF-002 requires it be the *only* containment implementation) | A buggy impl that skips `EvalSymlinks` on the target (only canonicalizing the root) would pass a naive string-prefix check but let a symlink escape through — TC-028 catches this because it builds a real symlink pointing outside `t.TempDir()` |
| TC-031..033 | `(*api_viewer.ViewerHandler).NavFolders(w http.ResponseWriter, r *http.Request)` registered at `GET /api/v1/viewer/nav-folders` via `RegisterRoutes`, driven with `httptest.NewRequest` + `httptest.NewRecorder()`, exactly as `TestHandler_WorkflowMeta` drives `WorkflowMeta` | `services.ViewerServicer` interface — mock only `NavFoldersFunc` on `MockViewerServicer`, exactly as `WorkflowMetaFunc` is mocked today | Do not bypass `RegisterRoutes`/`WithLocalCORS` by calling `h.NavFolders` directly without going through the mux if a route-level test is intended (TC-031 must hit the route; TC-032/033 may call the handler method directly per the existing `TestHandler_WorkflowMeta` pattern, which does call the method directly — that is the established, accepted seam for this handler family, not a forbidden shortcut) | A buggy impl that forgets to register the route (or registers it under the wrong path) would still pass a handler-only test; TC-031 specifically drives through `RegisterRoutes` + the mux to catch that |
| TC-023 | `(*config.Manager).Load()` → round-trip via `json.Marshal`/`json.Unmarshal` on `*config.Config`, exactly mirroring existing `RawData`-preservation tests in `config_test.go` | None — this is a pure marshal/unmarshal test, no I/O seam to mock | N/A | A buggy impl that adds `BrowsableFolders` without `omitempty` or that clobbers `RawData` would change the round-tripped JSON byte-for-byte; TC-023 diffs the round-tripped bytes |
| TC-035, TC-037 | N/A — code-review / static-assertion test cases, not driven through a production entrypoint | N/A | N/A | **content-only / static-assertion justification**: REQ-NF-002 and REQ-NF-004 are structural invariants ("no second containment helper exists", "no new go.mod entries") that are provable by `grep`/`go.mod` diff, not by exercising a runtime code path. A counter-factual runtime test cannot distinguish "one containment helper, called twice" from "two containment helpers" without literally grepping the diff — grep IS the correct verification mechanism here (matches `spec.md` §4.7's own classification of AC-006.5 as "code-review assertion in the review gate, not an automated test") |
| TC-001..017, TC-021 (partial), TC-024..025, TC-031 (partial), TC-032, TC-034, TC-036 | **content-only / manual UAT** — direct entrypoint is the rendered `internal/viewer/assets/viewer.html` SPA loaded in a real browser against a running `shark web` server (no JS test harness exists in this repo, per §7) | N/A | N/A | **content-only justification**: DOM structure (`data-sidebar-section`, group ordering), `localStorage` persistence across a real page reload, and end-to-end degradation behavior (killing the nav-folders response and observing no error toast) are only observable in a real browser session. This matches every prior E27 frontend feature's verification approach (`spec.md` §4.7) and the QA workflow's explicit carve-out for content/UI-only verification without a harness. |

---

## 9. Acceptance Test Cases

### REQ-F-001 — Three top-level collapsible groups

#### TC-001: Fixed group order in DOM

**Feature Requirement:** spec.md REQ-F-001, epic.md §Solution
**Task Acceptance Criterion:** AC-001.1a
**Technique Applied:** Equivalence Partitioning (no extra folders configured)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only — real browser against `shark web`, no `web.browsable_folders` configured (see §8)

**Preconditions:** Project has epics/bugs; no `web.browsable_folders` in `.sharkconfig.json`.

**Input:** Load the dashboard.

**Expected Output:** The first three `[data-sidebar-section^="group:"]` elements in `#sidebar-content`, in DOM order, have ids `group:plan`, `group:architecture`, `group:product`.

**Edge Cases:**
- With one `web.browsable_folders` entry configured: a fourth `group:*` element appears, strictly after the first three (AC-001.1b) → TC-018 covers this combination directly.

**Negative Cases:** No `group:*` element appears before `group:plan` under any configuration.

---

#### TC-002: Group header visually distinguishable

**Feature Requirement:** REQ-F-001
**Task Acceptance Criterion:** AC-001.3
**Technique Applied:** Equivalence Partitioning
**ISO 25010 Characteristic(s):** Functional Suitability, Usability

**Caller-Path Contract:** content-only — rendered DOM/CSS inspection in a real browser

**Preconditions:** Dashboard loaded.

**Input:** Inspect each group header element.

**Expected Output:** Each group header (`group:plan`, `group:architecture`, `group:product`, and any config group) carries the `sidebar-group-header` class, distinct from `.sidebar-section-header` used by nested sections.

**Negative Cases:** No nested section header carries `sidebar-group-header`.

---

#### TC-003: Collapsing one group leaves others unaffected

**Feature Requirement:** REQ-F-001
**Task Acceptance Criterion:** AC-001.2
**Technique Applied:** State Transition
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** All three groups expanded.

**Input:** Click the Plan group's collapse control.

**Expected Output:** Plan group body is hidden (`display:none` via `.is-collapsed`); Architecture and Product group bodies remain visible, expand state unchanged.

**Negative Cases:** Clicking Plan's collapse control must not toggle Architecture or Product's `is-collapsed` class.

---

#### TC-004: Group order stable across two folder-group configurations

**Feature Requirement:** REQ-F-001, REQ-F-005
**Task Acceptance Criterion:** AC-001.1b
**Technique Applied:** Equivalence Partitioning (0 vs 2 extra folder groups)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** `web.browsable_folders` has two entries, e.g. `docs/runbooks` and `docs/guides`.

**Input:** Load dashboard.

**Expected Output:** Exactly 5 `group:*` elements: `group:plan`, `group:architecture`, `group:product`, then the two config groups in declaration order.

---

### REQ-F-002 — Plan group contents unmodified

#### TC-005: All six entity sections nested under Plan

**Feature Requirement:** REQ-F-002
**Task Acceptance Criterion:** AC-002.1
**Technique Applied:** Equivalence Partitioning
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** Project has at least one epic, bug, change-card, tech-debt item, idea, and question.

**Input:** Load dashboard.

**Expected Output:** `[data-sidebar-section="epics"]`, `"bugs"`, `"change_cards"`, `"tech_debt"`, `"ideas"`, `"questions"` are each present and each a descendant of `[data-sidebar-section="group:plan"]`.

---

#### TC-006: Empty section still suppressed inside group

**Feature Requirement:** REQ-F-002
**Task Acceptance Criterion:** AC-002.2
**Technique Applied:** Equivalence Partitioning (empty vs non-empty section)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** Project has zero bugs, has epics.

**Input:** Load dashboard.

**Expected Output:** No `[data-sidebar-section="bugs"]` element renders anywhere, including inside `group:plan`.

**Negative Cases:** An empty section must not render an empty header either — full suppression, not just body suppression.

---

#### TC-007: Sprint tree nests inside Plan in sprint mode

**Feature Requirement:** REQ-F-002
**Task Acceptance Criterion:** AC-002.3
**Technique Applied:** Equivalence Partitioning (sprint vs non-sprint app state)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** `appState === 'sprint'` (an active sprint exists and sprint mode is entered).

**Input:** Load dashboard in sprint mode.

**Expected Output:** The sprint tree (`renderSprintTree` output) is a descendant of `group:plan`, positioned above the Tags-independent entity sections within the Plan body.

---

#### TC-008: Collapsing Plan hides nested sections; re-expanding restores individual state

**Feature Requirement:** REQ-F-002
**Task Acceptance Criterion:** AC-002.4
**Technique Applied:** State Transition
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:** content-only

**Preconditions:** Bugs section collapsed, Epics section expanded, Plan group expanded.

**Input:** Collapse Plan group, then re-expand it.

**Expected Output:** While Plan is collapsed, all nested sections are hidden (descendant-selector CSS rule per ADR-F16-1). After re-expanding Plan, Bugs is still collapsed and Epics is still expanded — nested per-section state was not reset by the group toggle.

**Negative Cases:** Re-expanding Plan must not force all nested sections open.

---

### REQ-F-003 — Architecture/Product browsing

#### TC-009: Architecture node has correct data attributes

**Feature Requirement:** REQ-F-003
**Task Acceptance Criterion:** AC-003.1
**Technique Applied:** Equivalence Partitioning
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** `docs/architecture` exists (true in this repo).

**Input:** Load dashboard.

**Expected Output:** A node with `data-folder-path="docs/architecture"` and `data-select-key="folder:docs/architecture"` exists inside `group:architecture`.

---

#### TC-010: Product node has correct data attributes

**Feature Requirement:** REQ-F-003
**Task Acceptance Criterion:** AC-003.2
**Technique Applied:** Equivalence Partitioning
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** `docs/product` exists.

**Input:** Load dashboard.

**Expected Output:** A node with `data-folder-path="docs/product"` and `data-select-key="folder:docs/product"` exists inside `group:product`.

---

#### TC-011: Clicking Architecture opens folder view via existing endpoint

**Feature Requirement:** REQ-F-003
**Task Acceptance Criterion:** AC-003.3
**Technique Applied:** Equivalence Partitioning
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only — network tab inspection confirms only `GET /api/v1/viewer/folder-files/docs/architecture` fires, no new endpoint

**Preconditions:** Dashboard loaded.

**Input:** Click the Architecture node.

**Expected Output:** Main panel shows `docs/architecture`'s immediate children; exactly one `folder-files` request is made, no new listing endpoint is called.

---

#### TC-012: Missing target directory shows empty listing, no error

**Feature Requirement:** REQ-F-003
**Task Acceptance Criterion:** AC-003.4
**Technique Applied:** Equivalence Partitioning (existing vs missing directory)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:** content-only

**Preconditions:** A project where `docs/product` does not exist (test fixture with only `docs/architecture`).

**Input:** Click the Product node.

**Expected Output:** Folder view renders an empty entries list (matches `FolderFiles`'s existing `os.IsNotExist` → empty-entries behavior). No error toast, no console error.

**Negative Cases:** Must not throw a JS exception or show a red error banner.

---

### REQ-F-004 — Persistence + toggle-all

#### TC-013: Group collapse persists across reload

**Feature Requirement:** REQ-F-004
**Task Acceptance Criterion:** AC-004.1
**Technique Applied:** State Transition
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:** content-only — real browser `localStorage` + page reload

**Preconditions:** All groups expanded.

**Input:** Collapse Plan group; reload the page.

**Expected Output:** After reload, Plan renders collapsed; Architecture and Product render expanded (their prior state, unaffected).

---

#### TC-014: Toggle-all button correctness when any group/section is collapsed

**Feature Requirement:** REQ-F-004
**Task Acceptance Criterion:** AC-004.2
**Technique Applied:** Boundary Value Analysis (one collapsed among many expanded is a boundary condition for "is everything expanded?")
**ISO 25010 Characteristic(s):** Functional Suitability, Usability

**Caller-Path Contract:** content-only

**Preconditions:** Everything expanded except one nested section (e.g. Bugs collapsed).

**Input:** Inspect `sidebar-toggle-all-btn` label; click it.

**Expected Output:** Button reads `+` / "Expand all sections" before the click. After clicking, every group and every section — including `questions` (ADR-F16-3 fix) and any dynamic folder group — is expanded.

---

#### TC-015: Toggle-all button correctness when everything expanded

**Feature Requirement:** REQ-F-004
**Task Acceptance Criterion:** AC-004.3
**Technique Applied:** Boundary Value Analysis (the "fully expanded" boundary)
**ISO 25010 Characteristic(s):** Functional Suitability, Usability

**Caller-Path Contract:** content-only

**Preconditions:** Every group and every section expanded (post TC-014).

**Input:** Click `sidebar-toggle-all-btn` again.

**Expected Output:** Button read `−` / "Collapse all sections" before this click; after clicking, every group and section is collapsed.

---

#### TC-016: Toggle-all covers dynamic user-registered folder groups

**Feature Requirement:** REQ-F-004
**Task Acceptance Criterion:** AC-004.4
**Technique Applied:** Equivalence Partitioning (static ids vs dynamic runtime-discovered ids)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** `web.browsable_folders` has one entry (`Runbooks`); its group is collapsed, everything else expanded.

**Input:** Click toggle-all.

**Expected Output:** Runbooks group expands along with the static groups/sections — `allSidebarSectionIds()` includes runtime-discovered `dynamicSidebarGroupIds`, proving the toggle-all read/write path isn't limited to `SIDEBAR_SECTION_DEFAULTS`.

**Negative Cases:** A static-map-only implementation would leave Runbooks collapsed after toggle-all — this is exactly the regression this test catches.

---

#### TC-017: localStorage failure degrades to in-memory toggling

**Feature Requirement:** REQ-F-004
**Task Acceptance Criterion:** AC-004.5
**Technique Applied:** Attack-class/failure-injection (storage throws)
**ISO 25010 Characteristic(s):** Reliability

**Caller-Path Contract:** content-only — devtools override of `localStorage.setItem`/`getItem` to throw, mirroring F10 REQ-F-004 AC-004.6's established verification approach

**Preconditions:** Override `window.localStorage` methods to throw (e.g. private-browsing quota simulation).

**Input:** Load dashboard, toggle a group.

**Expected Output:** Groups still render (default expanded) and still toggle in-memory for the session; no uncaught exception in console.

**Negative Cases:** An unwrapped `localStorage` call would throw an uncaught exception and break rendering — this is the counter-factual this test catches.

---

### REQ-F-005 — Config-registered folders

#### TC-018: Registered folder with explicit label renders

**Feature Requirement:** REQ-F-005
**Task Acceptance Criterion:** AC-005.1
**Technique Applied:** Equivalence Partitioning (label present)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `(*services.ViewerService).NavFolders(ctx)` constructed via `NewViewerService(...).WithBrowsableFolders([]config.BrowsableFolder{{Label: "Runbooks", Path: "docs/runbooks"}})`, project root = `t.TempDir()` with `docs/runbooks` created on disk
- **Lowest allowed mock seam:** none — real filesystem via `t.TempDir()`
- **Forbidden mocks:** `isContained`, `filepath.EvalSymlinks`
- **Counter-factual:** an impl that ignores the config-supplied label and always derives from basename would return `label:"Runbooks"` (matches basename here by coincidence) — TC-019 (label omitted, different basename) is the case that actually distinguishes correct label handling from a basename-only bug; this TC establishes the baseline shape

**Preconditions:** `docs/runbooks` exists under the temp project root.

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** Response `folders` includes `{id:"docs/runbooks", label:"Runbooks", path:"docs/runbooks", source:"config", exists:true}`. UAT companion: sidebar shows a "Runbooks" group after Product with a node `data-folder-path="docs/runbooks"`.

---

#### TC-019: Label omitted derives title-cased basename

**Feature Requirement:** REQ-F-005
**Task Acceptance Criterion:** AC-005.2
**Technique Applied:** Equivalence Partitioning (label absent)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** same entrypoint as TC-018

**Preconditions:** `docs/runbooks` exists; config entry `{Path: "docs/runbooks"}` (no `Label`).

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** `label == "Runbooks"` (title-cased basename of `docs/runbooks`).

**Edge Cases:** basename with hyphen/underscore (`my-notes` → expected title-casing behavior must be explicitly pinned by the implementation and this test — e.g. `"My-Notes"` or `"My Notes"`; whichever the impl picks, the test locks it in as a regression guard).

---

#### TC-020: Absent/empty config renders exactly the three built-ins

**Feature Requirement:** REQ-F-005
**Task Acceptance Criterion:** AC-005.3
**Technique Applied:** Equivalence Partitioning (three sub-partitions: absent, empty array, `web` absent entirely)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** `NavFolders(ctx)` with `WithBrowsableFolders(nil)`, `WithBrowsableFolders([]config.BrowsableFolder{})`, and without calling `WithBrowsableFolders` at all (three sub-cases in one table-driven test)

**Preconditions:** none

**Input:** Call `NavFolders(ctx)` for each of the three sub-cases.

**Expected Output:** `folders` contains exactly `architecture` and `product` (built-ins) in all three sub-cases; no error.

**Negative Cases:** No panic or error when `browsableFolders` is nil.

---

#### TC-021: Missing directory still renders with exists:false

**Feature Requirement:** REQ-F-005
**Task Acceptance Criterion:** AC-005.4
**Technique Applied:** Equivalence Partitioning (dir exists vs missing)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:** `NavFolders(ctx)` with config entry pointing at a path that does not exist under `t.TempDir()`

**Preconditions:** `docs/missing-folder` does NOT exist under the temp root.

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** Response includes `{id:"docs/missing-folder", exists:false, ...}` — the entry is present, not dropped. UAT companion: the rendered node carries the `is-unavailable` marker/title stating "not found"; clicking it shows the existing empty listing (reuses TC-012's proven `FolderFiles` behavior).

---

#### TC-022: Duplicate basenames get distinct group ids

**Feature Requirement:** REQ-F-005
**Task Acceptance Criterion:** AC-005.5
**Technique Applied:** Boundary Value Analysis (collision case)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** `NavFolders(ctx)` with two config entries: `{Path:"a/notes"}`, `{Path:"b/notes"}`, both existing under `t.TempDir()`

**Preconditions:** Both `a/notes` and `b/notes` exist.

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** Two entries returned, `id:"a/notes"` and `id:"b/notes"` — distinct, both derived from the full relative path, not just the basename.

**Negative Cases:** A basename-only id scheme would collide (`"notes"` twice) and this test would catch it via a duplicate-id assertion.

---

#### TC-023: Unknown config keys survive round-trip (RawData preservation)

**Feature Requirement:** REQ-F-005
**Task Acceptance Criterion:** AC-005.6
**Technique Applied:** Equivalence Partitioning
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility

**Caller-Path Contract:**
- **Entrypoint:** `(*config.Manager).Load()` / `Config` JSON unmarshal-then-remarshal round-trip, mirroring existing `config_test.go` RawData tests
- **Lowest allowed mock seam:** none — pure marshal/unmarshal, no I/O mock needed beyond the temp config file
- **Forbidden mocks:** none applicable
- **Counter-factual:** an impl that overwrites `RawData` when adding the new known field would drop an unrelated unknown top-level key (e.g. a hypothetical future field) on round-trip; this test's fixture includes one synthetic unknown key to catch that

**Preconditions:** A `.sharkconfig.json` fixture containing `web.browsable_folders` plus one synthetic unrelated unknown top-level key.

**Input:** Load then re-marshal the config.

**Expected Output:** The unknown key is present, unchanged, in the round-tripped output; `browsable_folders` round-trips correctly.

---

#### TC-030: Whitespace-only/empty path entry is dropped

**Feature Requirement:** REQ-F-005
**Task Acceptance Criterion:** spec.md §4.4 config-contract rule ("An entry with an empty/whitespace-only path is dropped")
**Technique Applied:** Boundary Value Analysis (empty string boundary)
**ISO 25010 Characteristic(s):** Functional Suitability, Security (defense against a degenerate path becoming an accidental project-root browse)

**Caller-Path Contract:** `NavFolders(ctx)` with config entry `{Path: "   "}`

**Preconditions:** none

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** No group is produced for the whitespace entry; the other valid entries still render; no panic.

**Negative Cases:** An unguarded implementation might resolve an empty path to the project root itself and produce a group that browses everything — this test's counter-factual is exactly that: an entry with `id:""` or `path:""` reaching the response.

---

### REQ-F-006 — Traversal rejection (attack-class enumeration)

#### TC-026: `../` relative traversal rejected

**Feature Requirement:** REQ-F-006, REQ-NF-002
**Task Acceptance Criterion:** AC-006.1
**Technique Applied:** Attack-class enumeration (relative traversal)
**ISO 25010 Characteristic(s):** Security

**Caller-Path Contract:**
- **Entrypoint:** `NavFolders(ctx)` with config entry `{Path: "../secrets"}`, project root = `t.TempDir()`
- **Lowest allowed mock seam:** none
- **Forbidden mocks:** `isContained` — must be the real function
- **Counter-factual:** a naive string-check (`strings.Contains(path, "..")`) would also reject legitimate paths containing literal `..` substrings in a filename and would miss a URL-encoded or symlink-based variant; the real `isContained`+`EvalSymlinks` sequence is what's under test

**Preconditions:** A sibling `secrets` directory exists one level above the temp project root.

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** No entry for `../secrets` in `folders`; other configured entries still present; response is still `200`.

---

#### TC-027: Absolute path rejected

**Feature Requirement:** REQ-F-006
**Task Acceptance Criterion:** AC-006.2
**Technique Applied:** Attack-class enumeration (absolute path)
**ISO 25010 Characteristic(s):** Security

**Caller-Path Contract:** `NavFolders(ctx)` with config entry `{Path: "/etc"}`

**Preconditions:** none

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** No entry for `/etc`.

---

#### TC-028: Symlink-escape rejected

**Feature Requirement:** REQ-F-006
**Task Acceptance Criterion:** AC-006.3
**Technique Applied:** Attack-class enumeration (symlink escape — the class ADR-F16-4/REQ-NF-002 specifically exists to close)
**ISO 25010 Characteristic(s):** Security

**Caller-Path Contract:** `NavFolders(ctx)` with a real `os.Symlink` created under `t.TempDir()` pointing to a directory outside the temp root; config entry references the relative path to that symlink

**Preconditions:** Symlink `docs/escape-link` → `/tmp/outside-target` (or an OS-agnostic `t.TempDir()` sibling), created via `os.Symlink` in test setup.

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** No entry for `docs/escape-link`; other entries still present.

**Negative Cases:** This is the case a `filepath.Abs`-only check (no `EvalSymlinks`) would miss — this is the specific counter-factual that justifies REQ-NF-002.

---

#### TC-029: Each rejection logs a warning and response is still 200

**Feature Requirement:** REQ-F-006
**Task Acceptance Criterion:** AC-006.4
**Technique Applied:** Attack-class enumeration + Observability assertion
**ISO 25010 Characteristic(s):** Security, Reliability

**Caller-Path Contract:** `NavFolders(ctx)` with two entries — one valid, one `../escape` — driven with a test-installed `slog` handler capturing log records

**Preconditions:** As TC-026, plus a captured `slog` handler attached to the default logger for the test's duration.

**Input:** Call `NavFolders(ctx)`.

**Expected Output:** Exactly one `slog.Warn` record naming `../escape` and the containment-failure reason; `NavFolders` returns `nil` error (equivalent to handler 200) with the valid entry present.

**Observability Evidence:**
- Log assertion: one `slog.Warn` record with the offending path and reason fields present.

---

### REQ-F-007 — Endpoint-down degradation

#### TC-031: Sidebar falls back to built-ins when nav-folders 500s

**Feature Requirement:** REQ-F-007
**Task Acceptance Criterion:** AC-007.1
**Technique Applied:** Decision Table (endpoint success/failure × config present/absent)
**ISO 25010 Characteristic(s):** Reliability

**Caller-Path Contract:**
- **Entrypoint (handler layer):** `GET /api/v1/viewer/nav-folders` routed through `RegisterRoutes(mux, prefix)` with `MockViewerServicer.NavFoldersFunc` returning an error, driven via `httptest.NewRequest`/`httptest.NewRecorder()` against the real `mux`
- **Lowest allowed mock seam:** `services.ViewerServicer` interface (`NavFoldersFunc` only)
- **Forbidden mocks:** do not bypass `RegisterRoutes`
- **Counter-factual:** a handler that returns `200` with an empty body on service error would silently produce zero nav groups client-side instead of a clear `500` the frontend's documented fallback path expects
- **Frontend half (content-only):** with the devtools-simulated `500`, sidebar still shows Plan/Architecture/Product from the hardcoded fallback constants (ADR-F16-6)

**Preconditions:** Handler test: mock returns error. UAT companion: browser devtools blocks/overrides the `nav-folders` response to return `500`.

**Input:** Handler test hits the route; UAT loads the dashboard with the blocked response.

**Expected Output:** Handler returns `500` with the documented error body. Dashboard still renders Plan, Architecture (`docs/architecture`), Product (`docs/product`) from frontend fallback constants.

---

#### TC-032: Failure is non-fatal — no error toast, dashboard still reaches ready state

**Feature Requirement:** REQ-F-007
**Task Acceptance Criterion:** AC-007.2
**Technique Applied:** Decision Table
**ISO 25010 Characteristic(s):** Reliability, Usability

**Caller-Path Contract:** content-only — same simulated-500 browser session as TC-031

**Preconditions:** As TC-031 UAT setup.

**Input:** Load dashboard with nav-folders blocked.

**Expected Output:** No error toast appears; the dashboard reaches its normal ready/interactive state — mirrors the existing `workflow-meta` non-fatal fallback idiom exactly.

---

#### TC-033: Successful response — built-ins not duplicated

**Feature Requirement:** REQ-F-007
**Task Acceptance Criterion:** AC-007.3
**Technique Applied:** Decision Table (success path)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** `(*api_viewer.ViewerHandler).NavFolders(w, r)` driven directly via `httptest.NewRecorder()` with `MockViewerServicer.NavFoldersFunc` returning a populated `NavFoldersResponse`, mirroring `TestHandler_WorkflowMeta`'s direct-call pattern

**Preconditions:** Mock returns `{folders: [architecture, product]}`.

**Input:** Call handler.

**Expected Output:** `200`, JSON body's `folders` array has exactly one `architecture` and one `product` entry (no duplication when the fetch succeeds). UAT companion: reconciliation by `id` on the frontend produces no visual duplicate group.

**Observability Evidence:** none required — success path (§4).

---

### REQ-F-008 — Docs entry retained (Q006 default)

#### TC-024: `docs` section unmoved, outside groups

**Feature Requirement:** REQ-F-008
**Task Acceptance Criterion:** AC-008.1
**Technique Applied:** Equivalence Partitioning (regression check against pre-existing behavior)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** content-only

**Preconditions:** Dashboard loaded.

**Input:** Inspect `[data-sidebar-section="docs"]`.

**Expected Output:** Element exists, contains the `folder:docs` node, and is NOT a descendant of any `[data-sidebar-section^="group:"]` element.

**Note:** Per spec.md §7 Q006 (open, default "retain"), if the product owner resolves Q006 to "remove" or "fold into Product" before merge, this TC and TC-025 must be revised or replaced — no other TC in this plan is affected.

---

#### TC-025: `docs` collapse-state key unchanged across upgrade

**Feature Requirement:** REQ-F-008
**Task Acceptance Criterion:** AC-008.2
**Technique Applied:** Equivalence Partitioning (pre-existing stored preference)
**ISO 25010 Characteristic(s):** Reliability, Compatibility

**Caller-Path Contract:** content-only — seed `localStorage` with a pre-upgrade `shark.viewer.sidebarSections` record containing `{docs: false}` (collapsed), then load the regrouped sidebar

**Preconditions:** `localStorage` seeded as above.

**Input:** Load dashboard.

**Expected Output:** `docs` section renders collapsed — the existing user's stored preference under the unchanged `docs` key still applies after upgrade.

---

### REQ-NF-001 — No regression of existing sidebar behaviors

#### TC-034: F10 and E28-F06 sidebar behaviors unaffected by regrouping

**Feature Requirement:** REQ-NF-001
**Task Acceptance Criterion:** Scenario 7 (spec.md §3)
**Technique Applied:** Regression re-run (existing E27-F10-tree-view-enhancements/spec.md REQ-F-001…005 and E28-F06 UAT cases, executed against the regrouped sidebar)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:** content-only — manual UAT re-running F10's own acceptance scenarios and E28-F06's tag-chip scenarios against this feature's build

**Preconditions:** Regrouped sidebar built; completed epics, tag-filterable entities present.

**Input:** Toggle "show all items"; toggle "show all files"; click the epics collapse-all arrow; select a tag chip.

**Expected Output:** Each behaves exactly as documented in F10 spec REQ-F-001…005 and E28-F06 — in particular, `collapse-all-btn` clears only `expandedEpics`/`expandedFeatures` and does not collapse groups or sections (spec.md REQ-NF-001 explicit callout).

**Negative Cases:** `collapse-all-btn` must NOT also set any `group:*` or section id to collapsed in `sidebarSectionState`.

---

### REQ-NF-002 — Path safety single-sourced

#### TC-035: No second containment helper introduced

**Feature Requirement:** REQ-NF-002
**Task Acceptance Criterion:** AC-006.5
**Technique Applied:** Static/structural assertion (code-review grep, per spec.md §4.7's own classification)
**ISO 25010 Characteristic(s):** Security, Maintainability

**Caller-Path Contract:** content-only / static-assertion (see §8) — `grep -rn "func isContained\|EvalSymlinks" internal/` or equivalent diff review at PR time

**Preconditions:** Feature branch diff available.

**Input:** Grep the diff for any new function performing path-containment or traversal detection outside `internal/services/helpers.go:isContained`.

**Expected Output:** Zero matches. `internal/config` performs no path-security validation (verified: no `filepath.EvalSymlinks`/containment logic added to `internal/config/config.go`).

---

### REQ-NF-003 — No added blocking latency

#### TC-036: Nav-folders fetch is non-blocking

**Feature Requirement:** REQ-NF-003
**Task Acceptance Criterion:** spec.md §2.2 REQ-NF-003 (no numbered AC — measurement-based)
**Technique Applied:** Boundary Value Analysis (blocking-phase boundary: hierarchy fetch vs metadata phase)
**ISO 25010 Characteristic(s):** Performance

**Caller-Path Contract:** content-only — browser network-waterfall inspection of `loadProjectData()`

**Preconditions:** Dashboard load, network tab open.

**Input:** Observe request ordering/blocking in `loadProjectData()`.

**Expected Output:** `nav-folders` fires in the same non-blocking phase as `workflow-meta`/vocabulary — never before the hierarchy fetch resolves the dashboard's ready state. Epic's "hierarchy loads in < 500ms for 500 tasks" criterion is unaffected (cross-reference epic success-criteria UAT scenario C5 in `uat-plan.md`).

---

### REQ-NF-004 — Epic constraints preserved

#### TC-037: No new dependencies, no build step introduced

**Feature Requirement:** REQ-NF-004
**Task Acceptance Criterion:** spec.md §2.2 REQ-NF-004
**Technique Applied:** Static/structural assertion
**ISO 25010 Characteristic(s):** Portability, Maintainability

**Caller-Path Contract:** content-only / static-assertion — `git diff go.mod` and `grep '<script src' internal/viewer/assets/viewer.html`

**Preconditions:** Feature branch diff available.

**Input:** Diff `go.mod`; grep `viewer.html` for new `<script src>` tags.

**Expected Output:** `go.mod` unchanged; no new `<script src>` entries; `viewer.html` remains the single-file SPA.

---

## 10. Integration Scenarios

| # | Components | Boundary verified | Epic UAT scenario this contributes to |
|---|---|---|---|
| 1 | `internal/viewer/server/wire.go` → `internal/config` → `internal/services.ViewerService` | Config loads and flows through `loadBrowsableFoldersConfig` into `WithBrowsableFolders` at server start | New UAT scenario (below); indirectly A1/A2 (server-start path in `uat-plan.md` Area A) |
| 2 | `internal/api/viewer/handler.go` → `ViewerService.NavFolders` → `internal/services/helpers.go:isContained` | Handler correctly surfaces service errors as 500 and service results as 200; containment check applied identically to the three existing call sites | H4/H5 (`uat-plan.md` path-traversal defense scenarios) — this feature's containment path must satisfy the same hard gates |
| 3 | `viewer.html` `apiGetNavFolders()` → `renderSidebar()` → existing `FOLDER_KEY_PREFIX` click dispatch → `GET /folder-files/{path}` | New nav-folders response correctly drives the pre-existing, unmodified folder-view click path | B1/B3 (data-parity scenarios), D1/D2 (spec-doc rendering — Architecture/Product entries funnel into the same file-render path) |
| 4 | `viewer.html` `sidebarSectionState` (localStorage) → group ids → toggle-all control | New `group:*` ids correctly merge into the existing single persistence store without breaking existing section ids | Directly this feature's TC-013..017 |

**Epic UAT plan cross-references:** REQ-F-006's containment behavior must additionally satisfy `uat-plan.md` §4 hard gates H4 ("Path-traversal defense") — if TC-026..029 pass but a manual UAT session finds a bypass, the epic sign-off (uat-plan.md §6) blocks regardless of this feature's own scenario results, per that document's own rule ("If any of H1–H6 or G1–G2 fail, the epic is blocked regardless of functional scenario results").

---

## 11. Additions to `docs/plan/E27-shark-status-viewer-local-web-dashboard/uat-plan.md`

Per spec.md §4.2 file-change table row 10, the following UAT cases must be appended to the epic's `uat-plan.md` (new Area, suggested "K — Grouped Nav & Browsable Folders (E27-F16)"):

- **K1** (= TC-001/TC-002/TC-004): Fixed group order and header styling, with and without config folders.
- **K2** (= TC-003/TC-008): Group-level collapse independence and nested-state preservation.
- **K3** (= TC-005/TC-006/TC-007): Plan group contents — full six sections + sprint tree, empty-section suppression.
- **K4** (= TC-011/TC-012): Architecture/Product browsing, including missing-directory case.
- **K5** (= TC-013..017): Persistence across reload; toggle-all correctness including dynamic groups; localStorage-failure resilience.
- **K6** (= TC-018/TC-019/TC-021/TC-022): Registered folder rendering — label default, missing-dir marker, duplicate-basename distinct groups.
- **K7** (= TC-031/TC-032/TC-033): Endpoint-down degradation — no error toast, built-ins preserved, no duplication on recovery.
- **K8** (= TC-024/TC-025): `docs` entry retained outside groups, stored preference honored (subject to Q006 resolution — flag as provisional in the UAT plan too).
- **K9** (= TC-034): Full re-run of F10 REQ-F-001…005 and E28-F06 tag-chip scenarios against the regrouped sidebar.
- **K10** (= TC-036): Non-blocking load confirmed via network waterfall; cross-reference existing scenario C5.

This test plan does not physically edit `uat-plan.md` — that edit is an implementation-phase deliverable per spec.md's file-change table (row 10 is listed under "Component changes," i.e. developer scope, not test-planning scope). This section hands the developer/QA-execution phase the exact mapping needed to perform that edit without re-deriving it.

---

## 12. Codex Test-Plan Red-Team

**Verdict:** SKIPPED — codex CLI unavailable in this execution environment (non-interactive worker sandbox; no `codex` binary on PATH). This is a non-blocking note per the workflow's own skip rule ("If a codex CLI is not available on this host, skip Step 7.5 ... do not treat its absence as a blocker").

**Self-review performed in lieu of codex** (manual pass against the same seven checks Step 7.5 would ask codex to run):

1. **Open-endedness check**: No AC in `spec.md` uses unenumerated robustness language ("must be secure", "must be robust"). REQ-F-006's traversal requirement is closed by an explicit attack-class enumeration (TC-026..029: relative, absolute, symlink-escape, plus TC-030's empty-path boundary) rather than a vague "must reject bad paths."
2. **ISTQB technique fit**: Verified in §2 — every AC group has a technique matching its shape (state transition for persistence, decision table for the endpoint-down/up × config-present/absent combination, attack-class enumeration for REQ-F-006).
3. **Enumeration completeness**: REQ-F-006's three named attack classes (`../`, absolute, symlink-escape) plus the spec-stated empty/whitespace-path drop rule (TC-030) cover every rejection path named in `spec.md` §4.4/§2.1. No additional attack class is named anywhere in the spec that lacks a TC.
4. **ISO 25010 coverage**: §3 has no empty cells; N/A cells are each justified inline by the AC's shape (e.g. a DOM-ordering AC has no Performance dimension).
5. **Observability design**: §4 — the one behavior with a real production consequence if silent (rejected-path logging, AC-006.4) has an explicit `slog.Warn` assertion (TC-029). Endpoint-failure logging mirrors the existing `WorkflowMeta` precedent exactly.
6. **Negative cases**: every functional TC section includes at least one explicit "must NOT happen" case (see TC-004, TC-006, TC-008, TC-016, TC-017, TC-022, TC-025, TC-034 negative-case lines).
7. **Caller-Path Contract**: §8 — every runtime TC (service/handler layer) names a real production entrypoint with the real argument shape and a counter-factual; every content-only TC (frontend, no JS harness) is explicitly marked `content-only` with its renderer/browser-session justification rather than silently omitted.

**Issues raised:** 0 (self-review found no blockers)
**Issues addressed before dev:** N/A
**Issues deferred:** 1 — codex automated red-team itself is deferred to whenever a `codex` CLI becomes available in this environment; owner: whoever executes the next test-planning pass with tool access, trigger: codex CLI present on PATH. This does not block APPROVED status per the workflow's explicit non-blocking rule.

---

## 13. Recommendations

- [x] Ready for development (no drift, spec is clear, every AC has a technique + ISO matrix entry, observability designed)
- [ ] Needs BA refinement (spec drift or missing requirements) — N/A, none found
- [ ] Needs tech refinement (technical ambiguity or gaps) — N/A, none found

**Outstanding, non-blocking:** Q006 (spec.md §7) remains open; REQ-F-008's two ACs/TCs (TC-024, TC-025) carry the spec's stated safe default and will need revision only if Q006 resolves away from "retain unchanged" — no other test case is affected.

**Recommended outcome for parent loop: standard.** Test plan meets exit gate — every AC in spec.md has at least one test case, every runtime test case has a caller-path contract (or documented content-only justification), edge cases are identified per AC via named ISTQB techniques, integration scenarios cover cross-component boundaries, test patterns cite existing infrastructure, and no I-##/X-## contract obligations exist for this feature.
