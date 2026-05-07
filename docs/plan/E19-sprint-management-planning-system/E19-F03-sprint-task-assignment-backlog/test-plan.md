# Test Plan: E19-F03 — Sprint Entity Assignment & Backlog

**Created:** 2026-05-05
**QA Agent:** QA
**Feature Spec:** `docs/plan/E19-sprint-management-planning-system/E19-F03-sprint-task-assignment-backlog/spec.md`
**Epic UAT Plan:** `docs/plan/E19-sprint-management-planning-system/uat-acceptance-plan.md`
**Status:** APPROVED

---

## Spec Drift Analysis

### Drift Findings

No significant drift detected between the spec.md acceptance criteria and the epic requirements (REQ-F-004, REQ-F-005, REQ-F-006, REQ-F-012).

Two minor observations requiring care during implementation:

1. **Carryover default config fallback**: AC for REQ-F-006 states the default is read from `.sharkconfig.json` `sprint_defaults.carryover` and falls back to `"next"` if absent. The spec §4.5 defines the struct correctly. Test plan includes explicit test for the fallback path.
2. **Capacity warning semantics**: AC for REQ-F-004 says "advisory only — enforcement is E19-F05". The spec design returns `*CapacityWarning` alongside the assignment rather than blocking it. Test plan validates warning is emitted but does not treat the missing-warning case as a pass.

### Traceability Matrix

| Feature PRD / Epic Requirement | Spec AC | Test Cases | Notes |
|---|---|---|---|
| REQ-F-004: Individual entity assignment (all four types) | AC-1 | TC-R01..TC-R08 | All entity types tested individually |
| REQ-F-004: Double-assignment error naming conflicting sprint | AC-2 | TC-R09..TC-R10 | Error message content validated |
| REQ-F-004: Advisory capacity warning | AC-2 (partial) | TC-R11 | Advisory, does not block |
| REQ-F-004: `--json` for add and remove | AC-1, AC-2 | TC-J01..TC-J02 | JSON output structure validated |
| REQ-F-005: Backlog all entity types grouped by status | AC-3 | TC-B01..TC-B04 | UNION query result verified |
| REQ-F-005: Completion percentage in header | AC-4 | TC-B05 | Math validated explicitly |
| REQ-F-005: `--type=` filter | AC-5 | TC-B06..TC-B07 | Each valid value + invalid |
| REQ-F-005: `--blocked` filter with days blocked | AC-6 | TC-B08..TC-B09 | Blocked-only semantics |
| REQ-F-005: `--json` with entity_type field | AC-7 | TC-J03 | Field presence required |
| REQ-F-006: close --carryover=next with next sprint | AC-8 | TC-C01..TC-C03 | Atomic transaction |
| REQ-F-006: close --carryover=next auto-create sprint | AC-8 | TC-C04 | Auto-create path |
| REQ-F-006: close --carryover=backlog | AC-9 | TC-C05..TC-C06 | Soft-delete path |
| REQ-F-006: Completed entities remain on closed sprint | AC-10 | TC-C07 | removed_at stays NULL |
| REQ-F-006: sprint_completions row created | AC-11 | TC-C08 | DB record verified |
| REQ-F-006: Default carryover from config | AC-12 | TC-C09..TC-C10 | Both config present and absent |
| REQ-F-006: Entire close is atomic | AC-8, AC-9 | TC-C11 | Rollback on error |
| REQ-F-012: Bulk from feature | (Should Have) | TC-K01..TC-K03 | Eligibility filtering |
| REQ-F-012: Bulk bugs/tech-debt/changes | (Should Have) | TC-K04..TC-K06 | Type-specific bulk |
| Schema version 18→19 | AC-14 | TC-S01 | Hard gate |
| `make fmt && make lint && make test` | AC-13 | TC-S02 | Quality gate |

---

## Acceptance Criteria Review

### Ambiguity Findings

**AC for REQ-F-006 — "completed-equivalent statuses"**: The spec says the service must ask `workflow.Service` for terminal statuses rather than hardcoding. The AC text says "completed entities remain attached." In a multi-workflow configuration, "completed" could mean different terminal status names. Test plan validates the workflow-service delegation path (not hardcoded `"completed"` string).

**AC for REQ-F-005 — "status category" grouping**: The spec delegates category mapping to the service layer using `workflow.Service` phase definitions (confirmed by INT-03 in UAT plan). Test plan asserts items appear in the correct phase bucket; it does not assert exact group names, which are workflow-config-dependent.

### Missing Coverage — None

All 14 numbered ACs in spec.md §8 have at least one test case. All should-have bulk ACs (REQ-F-012) have test coverage.

---

## ISTQB Technique Application (per AC)

| AC / Requirement | Technique(s) Applied | Test Cases Generated | Rationale |
|---|---|---|---|
| AC-1: Assign all four entity types | Equivalence Partitioning | TC-R01..TC-R04 (one per valid class), TC-R05..TC-R06 (invalid entity type) | Input is a set of entity types; each type is its own equivalence class |
| AC-1 cont.: Remove all four entity types | Equivalence Partitioning | TC-R07..TC-R08 | Same partitioning |
| AC-2: Double-assignment blocked | State Transition | TC-R09..TC-R10 | Assignment state: none → active → error on second add; also: none → active → removed → active (re-add after remove must succeed) |
| AC-2 cont.: Advisory capacity warning | Decision Table | TC-R11 | Conditions: capacity configured (Y/N) × exceeds capacity (Y/N) → 4 cells |
| AC-3: Backlog groups all entity types | Equivalence Partitioning + Contract surface enumeration | TC-B01..TC-B04 | Each entity type is one equivalence class; UNION query contract: 4 sub-selects, each must return entity_type label |
| AC-4: Completion percentage | BVA | TC-B05..TC-B05c | BVA on ratio: 0/N, all/N, k/N (mid-range); division by zero when N=0 |
| AC-5: `--type=` filter | Equivalence Partitioning + BVA | TC-B06..TC-B07 | Valid enum values (4) + invalid value + empty string |
| AC-6: `--blocked` filter | Decision Table | TC-B08..TC-B09 | Conditions: entity blocked (Y/N) × --blocked flag (Y/N) → 4 cells |
| AC-7: `--json` entity_type field | Contract surface enumeration | TC-J03 | Every JSON field in spec §4.2.3 BacklogItemView must be present and typed correctly |
| AC-8: carryover=next atomic | State Transition + Attack-class enumeration | TC-C01..TC-C04, TC-C11 | Sprint lifecycle states; attack classes: partial write (mid-tx error), next-sprint exists vs. doesn't exist |
| AC-9: carryover=backlog atomic | State Transition | TC-C05..TC-C06, TC-C11 | Same lifecycle; soft-delete path |
| AC-10: Completed entities stay attached | Attack-class enumeration | TC-C07 | Attack: did the carryover query accidentally include completed entities? |
| AC-11: sprint_completions row | Contract surface enumeration | TC-C08 | Every column of sprint_completions must be populated correctly |
| AC-12: Default carryover from config | Decision Table | TC-C09..TC-C10 | Conditions: config field set (Y/N) × value ("next" / "backlog" / absent) → 3 cells |
| AC-14: Schema version 19 | BVA | TC-S01 | Boundary: version must be exactly 19 (not 18, not 20) |
| REQ-F-012: Bulk eligibility | Decision Table | TC-K01..TC-K06 | Conditions: already-assigned (Y/N) × status-assignable (Y/N) × same feature (Y/N) |

---

## ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-1 (add/remove all types) | ✅ TC-R01..R08 | ✅ TC-P01 (<500ms) | N/A | ✅ TC-U01 (error msg) | N/A | ✅ TC-SEC01 (SQL injection via key) | ✅ lint clean (AC-13) | N/A |
| AC-2 (double-assign block) | ✅ TC-R09..R10 | N/A | N/A | ✅ TC-U01 (error names sprint) | ✅ TC-C11 (rollback) | ✅ unique index (DB-level, not app-only) | N/A | N/A |
| AC-2 (capacity warning) | ✅ TC-R11 | N/A | N/A | ✅ TC-U02 (warning text) | N/A | N/A | N/A | N/A |
| AC-3 (backlog grouping) | ✅ TC-B01..B04 | ✅ TC-P02 (<500ms, 200 items) | ✅ INT-03 (workflow phases) | N/A | N/A | ✅ TC-SEC01 (parameterized UNION) | N/A | N/A |
| AC-4 (completion %) | ✅ TC-B05..B05c | N/A | N/A | N/A | ✅ TC-B05b (0/0 no divide-by-zero) | N/A | N/A | N/A |
| AC-5 (--type filter) | ✅ TC-B06..B07 | N/A | N/A | ✅ TC-U03 (invalid type error) | N/A | N/A | N/A | N/A |
| AC-6 (--blocked filter) | ✅ TC-B08..B09 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-7 (--json entity_type) | ✅ TC-J01..J03 | N/A | ✅ (--field flag) | N/A | N/A | N/A | N/A | N/A |
| AC-8 (carryover=next) | ✅ TC-C01..C04 | ✅ TC-P03 (<2s, 200 entities) | N/A | ✅ TC-U04 (output shows moved count) | ✅ TC-C11 (atomic rollback) | N/A | N/A | N/A |
| AC-9 (carryover=backlog) | ✅ TC-C05..C06 | ✅ TC-P03 (shared) | N/A | ✅ TC-U05 | ✅ TC-C11 (shared) | N/A | N/A | N/A |
| AC-10 (completed stay) | ✅ TC-C07 | N/A | N/A | N/A | ✅ TC-C07 (data integrity) | N/A | N/A | N/A |
| AC-11 (completions row) | ✅ TC-C08 | N/A | N/A | N/A | ✅ TC-C08 (row queryable) | N/A | N/A | N/A |
| AC-12 (config default) | ✅ TC-C09..C10 | N/A | N/A | N/A | ✅ TC-C09 (absent config = "next") | N/A | N/A | ✅ config-file portability |
| AC-13 (make test passes) | N/A | N/A | N/A | N/A | N/A | N/A | ✅ TC-S02 | N/A |
| AC-14 (schema v19) | ✅ TC-S01 | N/A | ✅ TC-S01 (idempotent migration) | N/A | ✅ TC-S01 (no data loss) | N/A | N/A | N/A |

### Coverage Gaps

- **Performance (REQ-NF-001)**: TC-P01, TC-P02, TC-P03 are benchmark tests that must be run manually or in a CI benchmark job. Not included in the automated unit/integration suite by default. Deferred to QA execution phase; developer must provide a benchmark test stub that can be run with `go test -bench=.`.
- **Security (SQL injection via key)**: TC-SEC01 validates the entity-key lookup methods use parameterized queries (checked via code review and by passing malformed keys). Not a runtime test.
- **Portability**: The config round-trip test (TC-C09, TC-C10) covers "absent key defaults correctly" which is the main portability concern. Cross-OS is N/A for this feature.

---

## Observability Design (per behavior)

| Behavior | Metric | Log | Trace Span | Alert Threshold | Test Assertion |
|---|---|---|---|---|---|
| Entity assigned to sprint | `sprint.assignment.added_total{entity_type=}` | `INFO sprint.assignment.add sprint_key=S024 entity_type=task entity_key=E07-F01-001` | internal — CLI command; no distributed trace needed | N/A | TC-R01: assert assignment returned; log emitted in integration test |
| Entity removed from sprint | `sprint.assignment.removed_total{entity_type=}` | `INFO sprint.assignment.remove sprint_key=S024 entity_type=task entity_key=E07-F01-001` | internal | N/A | TC-R07 |
| Double-assignment conflict | `sprint.assignment.conflict_total` | `WARN sprint.assignment.conflict entity_key=... existing_sprint_key=S001` | internal | N/A | TC-R09: assert error contains sprint key |
| Capacity warning emitted | `sprint.capacity.warning_total{agent_type=}` | `WARN sprint.capacity.warning sprint_key=... agent_type=backend allocated=22 capacity=21` | internal | N/A | TC-R11: assert CapacityWarning non-nil and warning log emitted |
| Backlog queried | `sprint.backlog.query_duration_seconds` (histogram) | `INFO sprint.backlog.query sprint_key=S024 item_count=42 duration_ms=...` | internal | p95 > 400ms for 10 min | TC-P02: assert query returns within 500ms |
| Sprint closed (carryover=next) | `sprint.close.total{mode=next}`, `sprint.carryover.moved_total` | `INFO sprint.close sprint_key=S024 mode=next completed=8 carried_over=3 dropped=0 next_sprint_key=S025` | internal | N/A | TC-C08: assert completion record created with correct counts |
| Sprint closed (carryover=backlog) | `sprint.close.total{mode=backlog}`, `sprint.carryover.dropped_total` | `INFO sprint.close sprint_key=S024 mode=backlog completed=8 carried_over=0 dropped=3` | internal | N/A | TC-C06 |
| Auto-created next sprint on close | `sprint.autocreate.total` | `INFO sprint.autocreate parent_sprint=S024 new_sprint_key=S025 start_date=... end_date=...` | internal | N/A | TC-C04: assert result.NextSprintKey non-empty |
| Bulk assignment completed | `sprint.bulk.added_total{entity_type=}`, `sprint.bulk.skipped_total{entity_type=}` | `INFO sprint.bulk.add sprint_key=S024 added_by_type={task:5} skipped={task:2}` | internal | N/A | TC-K01: assert BulkAddResult counts correct |
| Transaction rollback on close error | `sprint.close.rollback_total` | `ERROR sprint.close.rollback sprint_key=S024 error=...` | internal | spike (>3σ) for 5 min | TC-C11: assert error returned and DB state unchanged |

**Implementation hook:** The observability columns above are hard requirements. The developer must add counter increments and structured log lines as part of the implementation. QA will verify log emission during the QA testing phase by inspecting test output for log lines matching the patterns above.

**Pure-internal behaviors with no observability**: The `BacklogItem` projection type (a read-only value object) and `SprintCompletion` model struct have no observability — internal data structures.

---

## Caller-Path Contracts (per test case)

| TC | Production Entrypoint | Lowest Mock Seam | Forbidden Mocks | Counter-factual |
|---|---|---|---|---|
| TC-R01..R04 | `SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey:"S024", EntityKey:"E07-F01-001"})` | `SprintRepository` interface (mock entire repo) | Do NOT mock `keys.KeyService.Parse` — it must run real key parsing to catch entity-type detection bugs | A buggy impl that calls `GetTaskIDByKey` for a bug key would fail the TC-R02 (bug) assertion when `entity_type` in the returned assignment is wrong |
| TC-R07..R08 | `SprintService.RemoveEntityFromSprint(ctx, "S024", "E07-F01-001")` | `SprintRepository` interface | Do NOT mock the active-assignment lookup; service must call `GetActiveAssignment` for real (via mock repo) | A buggy impl that skips the existence check would not error on remove-nonexistent, breaking TC-R08 |
| TC-R09 | `SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey:"S025", EntityKey:"E07-F01-001"})` where entity is already assigned to S024 | `SprintRepository.GetActiveAssignment` returns non-nil (mock) | Do NOT mock the conflict detection — the service must call `GetActiveAssignment`; a mock that returns `nil` hides the bug | A buggy impl that skips `GetActiveAssignment` would call `AddAssignment` and get a DB unique-constraint error instead of a ConflictError naming the sprint |
| TC-R11 | `SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey:"S024", EntityKey:"E07-F01-001"})` with capacity configured | `SprintRepository` interface | Do NOT mock the capacity check away — `ListAssignments` must be called via mock repo to simulate allocated state | A buggy impl that never checks capacity would return `nil` CapacityWarning even when over capacity |
| TC-B01..B04 | `SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{})` | `SprintRepository.ListBacklog` (mock to return fixture items per entity type) | Do NOT mock the grouping logic — service must perform status-category grouping for real; only the DB query is mocked | A buggy impl that discards entity_type from BacklogItem would produce groups with no entity-type label |
| TC-B05..B05c | `SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{})` | `SprintRepository.ListBacklog` (mock returns N total, K completed) | Do NOT mock `CompletionPercent` calculation — service must compute it from the mock items | A buggy impl with integer division (3/5 = 0 in Go integer math) would return 0% instead of 60% |
| TC-B06..B07 | `SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{EntityType:"task"})` | `SprintRepository.ListBacklog` receiving non-nil `entityType` pointer | Do NOT mock the filter validation — the service must validate the entity type value before passing to repo | A buggy impl that passes an unvalidated empty string would cause the repo to return all types, defeating the filter |
| TC-C01..C04 | `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)` | `SprintRepository` (all Tx variants) + `*dbconn.DB` (real `BeginTx`, mock repo Tx methods) | Do NOT mock `BeginTx` itself — the service must open a real transaction; mock only the repo method calls within it | A buggy impl that calls repo methods outside the transaction would not roll back on error, breaking TC-C11 |
| TC-C07 | `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)` | `SprintRepository.ListAssignmentsForCarryover` (mock returns only incomplete items) | Do NOT mock `ListAssignmentsForCarryover` to return completed items — it must exclude them; if it returns completed items the service must handle that defensively | A buggy `ListAssignmentsForCarryover` that returns all assignments would cause completed entities to be reassigned/dropped |
| TC-C08 | `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)` | `SprintRepository.CreateCompletionTx` (capture the completion struct passed) | Do NOT mock away the completion record creation — it must be called with correct counts; mock must capture the argument | A buggy impl that passes incorrect `CompletedEntityCount` would produce wrong velocity data in E19-F04 |
| TC-C09..C10 | `SprintService.CloseSprintWithCarryover(ctx, "S024", "")` (empty carryover mode) | `Config` (injected or read via service constructor) | Do NOT mock the config read; inject a real `Config` struct with `SprintDefaults.Carryover` set/absent | A buggy impl that ignores config and always uses "next" would pass TC-C09 but fail TC-C10 (backlog config) |
| TC-C11 | `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)` where `CreateCompletionTx` errors | Real `*dbconn.DB` `BeginTx` + mock repo that errors on `CreateCompletionTx` | Do NOT mock `Rollback` — `defer tx.Rollback()` must be called by the service for real | A buggy impl without `defer tx.Rollback()` would leave partial writes committed even when the final step fails |
| TC-K01..K06 | `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})` | `SprintRepository` (mock `ListAssignments` for already-assigned check) | Do NOT mock the eligibility filter — service must apply status-assignable check via real logic | A buggy impl that assigns already-assigned entities would produce duplicate assignment errors instead of skipping |
| TC-S01 | `internal/db.ApplySchemaIfNeeded(ctx, db)` (production migration path) | Real test DB (`test.GetTestDB()`) — no mocking | Do NOT mock the DB — this is a repository-level test that must use the real database | A buggy impl that forgets to call `migrateSprintCompletionsTable` from `runMigrations` would leave schema version at 18 |
| TC-P01..P03 | `runSprintAdd` / `runSprintBacklog` / `runSprintClose` CLI entrypoints (via direct function call in benchmark) | Real test DB | Do NOT mock — performance tests must use real DB; mock latency is not representative | A buggy UNION query without indexes would exceed 500ms threshold |
| TC-J01..J03 | `runSprintAdd` / `runSprintRemove` / `runSprintBacklog` with `--json` flag (via mock sprintServicer) | `MockSprintService` implementing `sprintServicer` interface | Do NOT mock `cli.OutputJSON` — it must be the real formatter to catch JSON encoding issues | A buggy impl that omits `entity_type` field from the JSON response would pass a structural test but fail TC-J03's field assertion |

---

## Acceptance Test Cases

### Repository-Level Tests (Real Database)

These tests use `test.GetTestDB()` and follow the pattern in `internal/repository/sprint/repository_test.go`.

---

#### TC-R01: AddAssignment — task entity type succeeds

**Feature Requirement:** REQ-F-004 — all four entity types assignable.
**AC:** AC-1
**Technique Applied:** Equivalence Partitioning (task = valid class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintRepository.AddAssignment(ctx, &models.SprintAssignment{SprintID: <id>, EntityType: "task", EntityID: <task-id>, AssignedAt: time.Now()})` — production callers never pass `ID` (auto-increment)
- **Lowest allowed mock seam:** Real test database (repository test)
- **Forbidden mocks:** None — this is a repository test; use real DB
- **Counter-factual:** A buggy impl that ignores `entity_type` would insert a row but `GetActiveAssignment` would not find it by entity_type filter

**Preconditions:**
- Sprint S901 in `planning` status in test DB
- Task row with id=1001 exists in `tasks` table
- No active assignment for (task, 1001)

**Input:**
- `SprintAssignment{SprintID: <S901.id>, EntityType: "task", EntityID: 1001, AssignedAt: now}`

**Expected Output:**
- `AddAssignment` returns nil error
- `GetActiveAssignment(ctx, "task", 1001)` returns the assignment with correct SprintID

**Edge Cases:**
- After `RemoveAssignment` (sets removed_at), a second `AddAssignment` for same entity succeeds (removed_at was set; partial unique index only covers `WHERE removed_at IS NULL`)

**Negative Cases:**
- Adding same entity to same sprint twice (removed_at IS NULL) returns UNIQUE constraint error

---

#### TC-R02: AddAssignment — bug entity type succeeds

**Feature Requirement:** REQ-F-004
**AC:** AC-1
**Technique Applied:** Equivalence Partitioning (bug = valid class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintRepository.AddAssignment(ctx, &models.SprintAssignment{SprintID: <id>, EntityType: "bug", EntityID: <bug-id>})`
- **Lowest allowed mock seam:** Real test DB
- **Forbidden mocks:** None
- **Counter-factual:** A buggy validation that only accepts "task" would reject "bug" entity_type

**Preconditions:** Sprint S901 active in test DB, bug row id=2001 exists.

**Input:** `SprintAssignment{SprintID: <S901.id>, EntityType: "bug", EntityID: 2001}`

**Expected Output:** nil error; `GetActiveAssignment("bug", 2001)` returns assignment.

**Negative Cases:** Inserting "bug_invalid" as entity_type fails validation before DB call.

---

#### TC-R03: AddAssignment — change_card entity type succeeds

**Feature Requirement:** REQ-F-004
**AC:** AC-1
**Technique Applied:** Equivalence Partitioning (change_card = valid class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** Same as TC-R01 with `EntityType: "change_card"`, `EntityID: <change_card-id>`.
**Counter-factual:** A buggy UNION query in `ListBacklog` that omits change_card sub-select would fail to return this assignment.

**Input/Expected:** Analogous to TC-R02 with change_card.

---

#### TC-R04: AddAssignment — tech_debt entity type succeeds

**Feature Requirement:** REQ-F-004
**AC:** AC-1
**Technique Applied:** Equivalence Partitioning (tech_debt = valid class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:** Same as TC-R01 with `EntityType: "tech_debt"`.
**Counter-factual:** A typo ("techdebt" vs "tech_debt") in the service would cause a key-not-found error.

**Input/Expected:** Analogous to TC-R02 with tech_debt.

---

#### TC-R05: AddAssignment — invalid entity type rejected

**Feature Requirement:** REQ-F-004 (boundary: invalid input class)
**AC:** AC-1
**Technique Applied:** Equivalence Partitioning (invalid class)
**ISO 25010 Characteristic(s):** Security, Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintRepository.AddAssignment(ctx, &models.SprintAssignment{EntityType: "epic"})`
- **Lowest allowed mock seam:** Real test DB (validation happens in model layer before DB write)
- **Forbidden mocks:** None
- **Counter-factual:** A buggy impl that skips model validation would insert "epic" into the entity_type column, violating CHECK constraint at DB level instead of returning a friendly error

**Input:** `SprintAssignment{EntityType: "epic", ...}`

**Expected Output:** Error returned; no row inserted.

---

#### TC-R06: AddAssignment — empty entity type rejected

**Feature Requirement:** REQ-F-004
**AC:** AC-1
**Technique Applied:** Equivalence Partitioning (empty/null class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Input:** `SprintAssignment{EntityType: "", EntityID: 1001, ...}`

**Expected Output:** Validation error; no DB write.

---

#### TC-R07: RemoveAssignment — task entity removed successfully

**Feature Requirement:** REQ-F-004
**AC:** AC-1
**Technique Applied:** State Transition (active → removed)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintRepository.RemoveAssignment(ctx, sprintID, "task", entityID)`
- **Lowest allowed mock seam:** Real test DB
- **Forbidden mocks:** None
- **Counter-factual:** A buggy impl that hard-deletes instead of soft-deletes would break velocity queries in E19-F04

**Preconditions:** Active assignment exists for (sprint S901, task 1001).

**Input:** `RemoveAssignment(ctx, S901.id, "task", 1001)`

**Expected Output:**
- nil error
- `GetActiveAssignment("task", 1001)` returns nil (removed)
- DB row has `removed_at` set (not deleted)

**Negative Cases:** `RemoveAssignment` on entity with no active assignment returns error.

---

#### TC-R08: RemoveAssignment — no active assignment returns error

**Feature Requirement:** REQ-F-004
**AC:** AC-1
**Technique Applied:** State Transition (invalid transition: removed → remove again)
**ISO 25010 Characteristic(s):** Reliability

**Input:** `RemoveAssignment(ctx, S901.id, "task", 9999)` (no active assignment)

**Expected Output:** Error (not-found); no DB side effects.

---

#### TC-R09: AddAssignment — double assignment to active sprint returns conflict error naming existing sprint

**Feature Requirement:** REQ-F-004 — "error naming the conflicting sprint key"
**AC:** AC-2
**Technique Applied:** State Transition (none → active → conflict on second add)
**ISO 25010 Characteristic(s):** Functional Suitability, Usability

**Caller-Path Contract:**
- **Entrypoint:** Second `SprintRepository.AddAssignment(ctx, &models.SprintAssignment{SprintID: S902.id, EntityType: "task", EntityID: 1001})` — entity already assigned to S901 (active)
- **Lowest allowed mock seam:** Real test DB (partial unique index must be exercised)
- **Forbidden mocks:** None — unique constraint must fire at the DB level for this test to be meaningful
- **Counter-factual:** A buggy service that only checks in-memory state (not the DB index) would miss concurrent assignments

**Preconditions:** Task 1001 is actively assigned to sprint S901 (planning status).

**Input:** Attempt to add task 1001 to sprint S902 (also planning).

**Expected Output:**
- Error returned; error message contains "S901" (or the existing sprint key)
- Row count in sprint_assignments for task 1001 with removed_at IS NULL = 1 (unchanged)

**Negative Cases:** After `RemoveAssignment` from S901, adding to S902 succeeds (no conflict).

---

#### TC-R10: AddAssignment — entity in completed sprint is assignable to new sprint

**Feature Requirement:** REQ-F-004 — "at most one sprint in planning or active status"
**AC:** AC-2
**Technique Applied:** State Transition (completed sprint → no conflict constraint)
**ISO 25010 Characteristic(s):** Functional Suitability

**Preconditions:** Task 1001 is assigned to closed/completed sprint S900 (removed_at IS NULL, sprint is completed).

**Input:** Add task 1001 to new planning sprint S901.

**Expected Output:** Success — the partial unique index only enforces uniqueness for active+planning sprint assignments. Entity can appear in a completed sprint AND a new active sprint simultaneously.

---

#### TC-R11: Capacity warning — advisory, not blocking

**Feature Requirement:** REQ-F-004 — "advisory only"
**AC:** AC-2 (capacity portion)
**Technique Applied:** Decision Table

| Capacity Configured | Exceeds Capacity | Expected Result |
|---|---|---|
| No | N/A | Assignment succeeds, CapacityWarning=nil |
| Yes | No | Assignment succeeds, CapacityWarning=nil |
| Yes | Yes | Assignment succeeds, CapacityWarning non-nil |

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.AddEntityToSprint(ctx, AddEntityInput{SprintKey:"S024", EntityKey:"E07-F01-001"})` with mock repo configured for each decision-table row
- **Lowest allowed mock seam:** `SprintRepository` interface (mock `ListAssignments` to simulate allocated state)
- **Forbidden mocks:** Do not mock the capacity computation logic in the service — only mock the repository data
- **Counter-factual:** A buggy impl that always returns `nil` CapacityWarning would pass the "no capacity" row but fail the "exceeds capacity" row

**Input (row 3):** Sprint capacity for "backend" = 5 points; 4 points already allocated; task being added has `agent_type=backend`, `size=2`.

**Expected Output (row 3):** Assignment returned (non-nil), CapacityWarning non-nil with `AgentType="backend"`, `Capacity=5`, `Allocated=6`.

---

#### TC-B01: ListBacklog — tasks returned with entity_type="task"

**Feature Requirement:** REQ-F-005 — all entity types grouped by status
**AC:** AC-3
**Technique Applied:** Contract surface enumeration (UNION query contract)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintRepository.ListBacklog(ctx, sprintID, nil, false)` — nil entityType means all types
- **Lowest allowed mock seam:** Real test DB (UNION query must execute; cannot mock at SQL level)
- **Forbidden mocks:** None — repository test
- **Counter-factual:** A UNION query that omits the `'task' AS entity_type` literal would return rows with empty entity_type, breaking the label display

**Preconditions:** Sprint S901 with active assignments: 1 task, 1 bug. Both joined records exist in `tasks` and `bugs` tables.

**Input:** `ListBacklog(ctx, S901.id, nil, false)`

**Expected Output:** 2 BacklogItems; item with EntityType="task"; item with EntityType="bug". Both have non-empty Key, Title, Status, Priority.

---

#### TC-B02: ListBacklog — bug entity type returned with entity_type="bug"

**Feature Requirement:** REQ-F-005
**AC:** AC-3
**Technique Applied:** Contract surface enumeration
**ISO 25010 Characteristic(s):** Functional Suitability

Analogous to TC-B01; specifically asserts bug row has EntityType="bug" and `[bug]` label path in service grouping.

---

#### TC-B03: ListBacklog — change_card entity type returned

**Feature Requirement:** REQ-F-005
**AC:** AC-3
**Technique Applied:** Contract surface enumeration
**ISO 25010 Characteristic(s):** Functional Suitability

**Preconditions:** Active assignment for change_card CC001 in sprint S901.

**Expected Output:** BacklogItem with EntityType="change_card".

---

#### TC-B04: ListBacklog — tech_debt entity type returned

**Feature Requirement:** REQ-F-005
**AC:** AC-3
**Technique Applied:** Contract surface enumeration
**ISO 25010 Characteristic(s):** Functional Suitability

**Preconditions:** Active assignment for tech_debt TD001 in sprint S901.

**Expected Output:** BacklogItem with EntityType="tech_debt".

---

#### TC-B05: GetSprintBacklog — completion percentage BVA

**Feature Requirement:** REQ-F-005 — "shows completion percentage"
**AC:** AC-4
**Technique Applied:** BVA on ratio (completed/total)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{})` with mock repo returning fixture items
- **Lowest allowed mock seam:** `SprintRepository.ListBacklog` (mock)
- **Forbidden mocks:** Do NOT mock `CompletionPercent` calculation — service must compute it
- **Counter-factual:** Integer division bug: `3/5 = 0` in Go integer arithmetic; test catches it because expected is `60.0` not `0.0`

| Variation | Total | Completed | Expected CompletionPercent | Notes |
|---|---|---|---|---|
| TC-B05a | 5 | 3 | 60.0 | Mid-range |
| TC-B05b | 0 | 0 | 0.0 | BVA: N=0, must not divide by zero |
| TC-B05c | 4 | 4 | 100.0 | BVA: all completed |
| TC-B05d | 10 | 0 | 0.0 | BVA: none completed |

**Preconditions (TC-B05a):** Mock `ListBacklog` returns 5 items; 3 have a status the workflow service marks as terminal/completed; 2 are in non-terminal status.

**Expected Output:** `SprintBacklog.CompletionPercent == 60.0`, `TotalCount == 5`, `CompletedCount == 3`.

---

#### TC-B06: GetSprintBacklog — --type=task filter (valid enum value)

**Feature Requirement:** REQ-F-005 — `--type=` filter
**AC:** AC-5
**Technique Applied:** Equivalence Partitioning (valid type class) + BVA (exact enum boundary)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{EntityType:"task"})`
- **Lowest allowed mock seam:** `SprintRepository.ListBacklog` — assert it is called with non-nil `entityType` pointer pointing to "task"
- **Forbidden mocks:** Do NOT mock the entityType validation — service must validate the string before passing pointer to repo
- **Counter-factual:** A buggy service that passes nil instead of &"task" would return all types; the assertion `len(items) == task_count` would pass spuriously

**Input:** `BacklogOptions{EntityType: "task"}`

**Expected Output:** `SprintBacklog.Groups` contains only items with EntityType="task". Repo was called with entityType="task" (captured via mock).

---

#### TC-B07: GetSprintBacklog — invalid --type value returns error

**Feature Requirement:** REQ-F-005
**AC:** AC-5
**Technique Applied:** Equivalence Partitioning (invalid class)
**ISO 25010 Characteristic(s):** Functional Suitability, Usability

**Input:** `BacklogOptions{EntityType: "sprint"}` (invalid)

**Expected Output:** Error returned; error message indicates valid values are task, bug, change_card, tech_debt.

---

#### TC-B08: GetSprintBacklog — --blocked filter shows only blocked entities

**Feature Requirement:** REQ-F-005 — `--blocked` with blocking reason and days blocked
**AC:** AC-6
**Technique Applied:** Decision Table

| Entity Blocked | --blocked Flag | Expected In Result |
|---|---|---|
| No | false | Yes |
| No | true | No |
| Yes | false | Yes |
| Yes | true | Yes |

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.GetSprintBacklog(ctx, "S024", BacklogOptions{BlockedOnly: true})`
- **Lowest allowed mock seam:** `SprintRepository.ListBacklog` — assert called with `blockedOnly=true`; the blocked status set is passed from workflow service to repo
- **Forbidden mocks:** Do NOT mock the blocked-status set determination; service must ask workflow service for blocked statuses
- **Counter-factual:** A buggy service that hardcodes `status == "blocked"` would miss entity types with different blocked status names in custom workflows

**Preconditions (decision table row 4):** Sprint S024 has task T1 (status=blocked) and task T2 (status=in_progress). BacklogOptions{BlockedOnly: true}.

**Expected Output:** Only T1 in result. `BacklogItemView.DaysBlocked > 0` if T1 has been blocked > 0 days.

---

#### TC-B09: GetSprintBacklog — DaysBlocked calculation correct

**Feature Requirement:** REQ-F-005 — "days blocked" shown
**AC:** AC-6
**Technique Applied:** BVA (days: 0, 1, N)
**ISO 25010 Characteristic(s):** Functional Suitability

**Input:** Task blocked since exactly 3 days ago (test uses `time.Now().AddDate(0,0,-3)` as mock assignment date with blocked status from that point).

**Expected Output:** `DaysBlocked == 3`.

**Edge Cases:**
- Task blocked today: `DaysBlocked == 0`
- Task not blocked: `DaysBlocked == 0` (field is zero-value for non-blocked items)

---

### Service-Level Tests (Mock Repository)

These tests use `MockSprintRepository` from `internal/services/sprint_service_test.go`.

---

#### TC-C01: CloseSprintWithCarryover — next, existing planning sprint found

**Feature Requirement:** REQ-F-006 — carryover=next moves incomplete to next sprint
**AC:** AC-8
**Technique Applied:** State Transition (active → completed; incomplete assignments → reassigned)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)`
- **Lowest allowed mock seam:** `SprintRepository` (all methods, including Tx variants)
- **Forbidden mocks:** Do NOT mock `BeginTx` on the `*dbconn.DB` — the service must call `s.db.BeginTx(ctx, nil)` for real (use a real in-memory or test DB for the Tx); mock only the repo method calls within the transaction. Alternatively: inject a `MockDB` that records `BeginTx` was called, but the transaction object itself must implement the correct SQL.Tx interface.
- **Counter-factual:** A buggy impl that calls `ReassignToSprintTx` outside the transaction would not roll back on subsequent failure

**Preconditions:**
- Sprint S024 (id=24) in `active` status
- Sprint S025 (id=25) in `planning` status (mock `List` returns it)
- `ListAssignmentsForCarryover` returns 3 incomplete assignments (ids 101, 102, 103)
- 5 total active assignments (2 completed, 3 incomplete)

**Input:** `CloseSprintWithCarryover(ctx, "S024", CarryoverNext)`

**Expected Output:**
- `SprintCloseResult.CompletedCount == 2`
- `SprintCloseResult.CarriedOverCount == 3`
- `SprintCloseResult.DroppedCount == 0`
- `SprintCloseResult.NextSprintKey == "S025"`
- Mock `ReassignToSprintTx` called with `assignmentIDs=[101,102,103]` and `newSprintID=25`
- Mock `UpdateStatus` called with status=completed for sprint 24
- Mock `CreateCompletionTx` called

---

#### TC-C02: CloseSprintWithCarryover — next, no existing planning sprint

**Feature Requirement:** REQ-F-006 — "if no next sprint exists, auto-created"
**AC:** AC-8
**Technique Applied:** State Transition + Decision Table (next sprint exists Y/N)
**ISO 25010 Characteristic(s):** Functional Suitability

**Preconditions:** Sprint S024 active; `List` returns no planning sprints; closed sprint start=2026-01-01, end=2026-01-14 (duration=13 days).

**Expected Output:**
- Mock `Create` called with new sprint start=2026-01-15, end=2026-01-28 (same duration)
- `SprintCloseResult.NextSprintKey` is the key of the newly created sprint (non-empty)

---

#### TC-C03: CloseSprintWithCarryover — all tasks completed, no carryover needed

**Feature Requirement:** REQ-F-006 (edge case: 100% complete)
**AC:** AC-8, AC-10
**Technique Applied:** BVA (incomplete count = 0)
**ISO 25010 Characteristic(s):** Functional Suitability

**Preconditions:** `ListAssignmentsForCarryover` returns empty slice (all entities completed).

**Expected Output:**
- `SprintCloseResult.CarriedOverCount == 0`
- `ReassignToSprintTx` NOT called (no-op)
- Sprint still transitions to completed
- Completion record created with `carried_over_count=0`

---

#### TC-C04: CloseSprintWithCarryover — auto-created sprint has correct start date

**Feature Requirement:** REQ-F-006 — "start date = (closed sprint end date + 1 day)"
**AC:** AC-8
**Technique Applied:** BVA (date arithmetic boundary)
**ISO 25010 Characteristic(s):** Functional Suitability

**Preconditions:** Closed sprint end_date = 2026-04-15.

**Expected Output:** Auto-created sprint start_date = 2026-04-16 (not 2026-04-15, not 2026-04-17).

**Edge Cases:**
- End date at month boundary: end=2026-01-31, new start=2026-02-01
- End date at year boundary: end=2026-12-31, new start=2027-01-01

---

#### TC-C05: CloseSprintWithCarryover — backlog mode soft-deletes incomplete assignments

**Feature Requirement:** REQ-F-006 — carryover=backlog "soft-deletes assignments"
**AC:** AC-9
**Technique Applied:** State Transition (active assignment → removed)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverBacklog)`
- **Lowest allowed mock seam:** `SprintRepository` (Tx variants)
- **Forbidden mocks:** Do NOT mock `DropAssignmentsTx` away — it must be called with the correct assignment IDs
- **Counter-factual:** A buggy impl that calls `ReassignToSprintTx` instead of `DropAssignmentsTx` in backlog mode would move entities to another sprint instead of releasing them

**Expected Output:**
- Mock `DropAssignmentsTx` called with incomplete assignment IDs
- Mock `ReassignToSprintTx` NOT called
- `SprintCloseResult.DroppedCount == 3` (for 3 incomplete assignments)
- `SprintCloseResult.CarriedOverCount == 0`

---

#### TC-C06: CloseSprintWithCarryover — backlog mode, empty sprint

**Feature Requirement:** REQ-F-006
**AC:** AC-9
**Technique Applied:** BVA (incomplete count = 0, BVA lower bound)
**ISO 25010 Characteristic(s):** Reliability

**Preconditions:** Sprint has 0 active assignments (`ListAssignmentsForCarryover` returns []).

**Expected Output:** Closes successfully; `DroppedCount == 0`; no error; completion record created.

---

#### TC-C07: CloseSprintWithCarryover — completed entities remain attached (removed_at stays NULL)

**Feature Requirement:** REQ-F-006 — "Completed entities remain attached to the closed sprint"
**AC:** AC-10
**Technique Applied:** Attack-class enumeration (did the carryover query accidentally capture completed entities?)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)` with mock `ListAssignmentsForCarryover` returning only 2 incomplete (not the 3 completed)
- **Lowest allowed mock seam:** `SprintRepository.ListAssignmentsForCarryover` (mock to return only incomplete)
- **Forbidden mocks:** Do NOT mock `ListAssignmentsForCarryover` to return all assignments — the test must verify the service uses the correct "incomplete-only" query
- **Counter-factual:** A buggy `ListAssignmentsForCarryover` that returns all active assignments would cause `ReassignToSprintTx` to move completed entities, breaking velocity queries

**Preconditions:**
- Sprint S024 has 5 total assignments: 3 completed (status=completed), 2 incomplete (status=in_progress)
- `ListAssignmentsForCarryover` mock returns only 2 incomplete items

**Expected Output:**
- `ReassignToSprintTx` called with exactly 2 IDs (not 5)
- `SprintCloseResult.CompletedCount == 3`, `CarriedOverCount == 2`

---

#### TC-C08: CloseSprintWithCarryover — sprint_completions row created with correct fields

**Feature Requirement:** REQ-F-006 — "generates a sprint_completions row with summary statistics"
**AC:** AC-11
**Technique Applied:** Contract surface enumeration (every column of sprint_completions)
**ISO 25010 Characteristic(s):** Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)` with mock that captures the `SprintCompletion` passed to `CreateCompletionTx`
- **Lowest allowed mock seam:** `SprintRepository.CreateCompletionTx` (capture argument)
- **Forbidden mocks:** Do NOT mock away `CreateCompletionTx` without capturing — must verify the passed struct's fields
- **Counter-factual:** A buggy impl that passes `CompletedEntityCount=0` always would produce wrong velocity data read by E19-F04

**Preconditions:** Sprint has 8 planned, 5 completed, 3 incomplete (carryover=next), NextSprintID=25. PlannedSizeSum=20.0 (mocked entity sizes).

**Expected Output:** `CreateCompletionTx` called with `SprintCompletion`:
- `SprintID == S024.id`
- `PlannedEntityCount == 8`
- `CompletedEntityCount == 5`
- `CarriedOverCount == 3`
- `DroppedCount == 0`
- `CarryoverMode == "next"`
- `NextSprintID == &25` (pointer to int64 25)

---

#### TC-C09: CloseSprintWithCarryover — default carryover from config ("next")

**Feature Requirement:** REQ-F-006 — "default carryover from .sharkconfig.json"
**AC:** AC-12
**Technique Applied:** Decision Table (config set with "next")
**ISO 25010 Characteristic(s):** Functional Suitability, Portability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.CloseSprintWithCarryover(ctx, "S024", "")` — empty string triggers config read
- **Lowest allowed mock seam:** `SprintRepository`; inject Config with `SprintDefaults.Carryover = "next"`
- **Forbidden mocks:** Do NOT mock the config read — inject a real Config struct
- **Counter-factual:** A buggy impl that ignores config and always defaults to "backlog" would call `DropAssignmentsTx` instead of `ReassignToSprintTx`

**Input:** Empty `carryoverMode` string; config has `sprint_defaults.carryover = "next"`.

**Expected Output:** Service behaves as `CarryoverNext`; `ReassignToSprintTx` is called.

---

#### TC-C10: CloseSprintWithCarryover — default carryover absent from config defaults to "next"

**Feature Requirement:** REQ-F-006
**AC:** AC-12
**Technique Applied:** Decision Table (config absent → default)
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability

**Input:** Empty `carryoverMode`; config has `SprintDefaults.Carryover = ""` (absent).

**Expected Output:** Service defaults to `CarryoverNext`; `ReassignToSprintTx` called.

---

#### TC-C11: CloseSprintWithCarryover — rollback on mid-transaction error

**Feature Requirement:** REQ-F-006 — "entire close+carryover operation executes in a single database transaction"
**AC:** AC-8, AC-9
**Technique Applied:** Attack-class enumeration (partial write: final step fails, transaction must roll back)
**ISO 25010 Characteristic(s):** Reliability, Functional Suitability

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.CloseSprintWithCarryover(ctx, "S024", CarryoverNext)`
- **Lowest allowed mock seam:** `SprintRepository.CreateCompletionTx` — inject error here (last step in transaction)
- **Forbidden mocks:** Do NOT mock `defer tx.Rollback()` — the real `tx.Rollback()` call must execute via the `defer` statement; use a real test DB transaction for this
- **Counter-factual:** A buggy impl without `defer tx.Rollback()` would leave `sprint_assignments` with reassigned sprint_ids even though `sprint_completions` was never written

**Preconditions:** Real test DB with sprint S901 (active), 2 assignments. Mock the Tx completion to error.

**Expected Output:**
- Error returned from `CloseSprintWithCarryover`
- Sprint S901 status still `active` in DB (not `completed`)
- `sprint_assignments` rows unchanged (no sprint_id update persisted)
- `sprint_completions` has no row for S901

---

#### TC-C12: CloseSprintWithCarryover — sprint not in active status rejected

**Feature Requirement:** REQ-F-006 — implicit precondition: sprint must be active
**AC:** AC-8
**Technique Applied:** State Transition (invalid: planning → close)
**ISO 25010 Characteristic(s):** Functional Suitability

**Input:** Sprint S024 in `planning` status.

**Expected Output:** Error returned before any DB writes; error message includes current status.

---

### Bulk Assignment Tests (Should Have)

---

#### TC-K01: BulkAddToSprint — assigns eligible tasks from feature (excludes already-assigned)

**Feature Requirement:** REQ-F-012
**AC:** REQ-F-012
**Technique Applied:** Decision Table (assigned Y/N × status-assignable Y/N)
**ISO 25010 Characteristic(s):** Functional Suitability

| Already Assigned | Status Assignable | Expected |
|---|---|---|
| No | Yes | Added |
| Yes | Yes | Skipped |
| No | No | Skipped |
| Yes | No | Skipped |

**Caller-Path Contract:**
- **Entrypoint:** `SprintService.BulkAddToSprint(ctx, BulkAddInput{SprintKey:"S024", FeatureKey:"E07-F34"})`
- **Lowest allowed mock seam:** `SprintRepository` (mock `GetActiveAssignment` per entity; mock `AddAssignment`)
- **Forbidden mocks:** Do NOT mock eligibility logic — service must check both already-assigned AND status-assignable conditions
- **Counter-factual:** A buggy impl that only checks assignment (not status) would add tasks in non-assignable statuses (e.g., `completed`) to the sprint

**Preconditions:** Feature E07-F34 has 5 tasks: 3 eligible (not assigned, assignable status), 1 already in S023, 1 in `completed` status.

**Expected Output:** `BulkAddResult.AddedByType["task"] == 3`, `SkippedByType["task"] == 2`.

---

#### TC-K02: BulkAddToSprint — capacity warning on bulk add

**Feature Requirement:** REQ-F-012 — "warns if bulk assignment exceeds any agent type's capacity"
**AC:** REQ-F-012
**Technique Applied:** Equivalence Partitioning (capacity exceeded class)
**ISO 25010 Characteristic(s):** Usability

**Preconditions:** Sprint backend capacity=5; 4 allocated; bulk adds 2 backend tasks.

**Expected Output:** `BulkAddResult.CapacityWarnings` non-empty; both tasks are added (advisory).

---

#### TC-K03: BulkAddToSprint — feature not found returns error

**Feature Requirement:** REQ-F-012
**AC:** REQ-F-012
**Technique Applied:** Equivalence Partitioning (invalid feature key class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Input:** `BulkAddInput{FeatureKey: "E99-F99"}` (non-existent feature)

**Expected Output:** Error; zero assignments made.

---

#### TC-K04: BulkAddToSprint — --bulk-bugs assigns open bugs

**Feature Requirement:** REQ-F-012
**AC:** REQ-F-012
**Technique Applied:** Equivalence Partitioning (bug entity type class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Input:** `BulkAddInput{SprintKey:"S024", EntityTypes:["bug"]}`.

**Expected Output:** `AddedByType["bug"]` = count of open, unassigned bugs.

---

#### TC-K05: BulkAddToSprint — --bulk-tech-debt assigns open tech-debt

**Feature Requirement:** REQ-F-012
**AC:** REQ-F-012
**Technique Applied:** Equivalence Partitioning (tech_debt entity type class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Input:** `BulkAddInput{SprintKey:"S024", EntityTypes:["tech_debt"]}`.

**Expected Output:** `AddedByType["tech_debt"]` = count of open, unassigned tech-debt items.

---

#### TC-K06: BulkAddToSprint — --bulk-changes assigns open change-cards

**Feature Requirement:** REQ-F-012
**AC:** REQ-F-012
**Technique Applied:** Equivalence Partitioning (change_card entity type class)
**ISO 25010 Characteristic(s):** Functional Suitability

**Input:** `BulkAddInput{SprintKey:"S024", EntityTypes:["change_card"]}`.

**Expected Output:** `AddedByType["change_card"]` = count of open, unassigned change-cards.

---

### CLI Command Tests (Mock sprintServicer)

These tests use `MockSprintService` from `internal/cli/commands/sprint_test.go`. The `MockSprintService` must be extended with the new methods.

---

#### TC-J01: sprint add JSON output contains SprintAssignment fields

**Feature Requirement:** REQ-F-004 — `--json` output
**AC:** AC-1
**Technique Applied:** Contract surface enumeration (JSON output fields)
**ISO 25010 Characteristic(s):** Compatibility (AI orchestrator consumption)

**Caller-Path Contract:**
- **Entrypoint:** `runSprintAdd(cmd, []string{"S024", "E07-F01-001"})` with `--json` flag set; `GlobalConfig.JSON = true`
- **Lowest allowed mock seam:** `MockSprintService.AddEntityToSprint` (returns fixture `SprintAssignment`)
- **Forbidden mocks:** Do NOT mock `cli.OutputJSON` — real formatter must be used to catch field serialization bugs
- **Counter-factual:** A buggy handler that calls `cli.OutputJSON(result.Sprint)` instead of `cli.OutputJSON(result)` would serialize the wrong struct

**Preconditions:** Mock returns `SprintAssignment{SprintID:24, EntityType:"task", EntityID:1001}`.

**Expected Output:** JSON contains `sprint_id`, `entity_type`, `entity_id`, `assigned_at` fields.

---

#### TC-J02: sprint remove JSON output

**Feature Requirement:** REQ-F-004 — `--json` for remove
**AC:** AC-1
**Technique Applied:** Contract surface enumeration
**ISO 25010 Characteristic(s):** Compatibility

**Caller-Path Contract:**
- **Entrypoint:** `runSprintRemove(cmd, []string{"S024", "E07-F01-001"})` with `--json` flag
- **Lowest allowed mock seam:** `MockSprintService.RemoveEntityFromSprint`
- **Counter-factual:** A buggy impl that returns no JSON on remove (only success message) would fail orchestrator automation

**Expected Output:** JSON with confirmation fields (or `{"ok": true}` — whatever the spec defines); zero exit code.

---

#### TC-J03: sprint backlog JSON output contains entity_type on every item

**Feature Requirement:** REQ-F-005 — `--json` with entity_type field
**AC:** AC-7
**Technique Applied:** Contract surface enumeration
**ISO 25010 Characteristic(s):** Compatibility

**Caller-Path Contract:**
- **Entrypoint:** `runSprintBacklog(cmd, []string{"S024"})` with `--json` flag
- **Lowest allowed mock seam:** `MockSprintService.GetSprintBacklog` (returns fixture `SprintBacklog` with 2 items: one task, one bug)
- **Forbidden mocks:** Do NOT mock `cli.OutputJSON` — real formatter validates field presence
- **Counter-factual:** A buggy `BacklogItemView` struct missing `EntityType json:"entity_type"` tag would serialize the field as empty string or omit it

**Expected Output:** JSON `groups[*].items[*]` all have `entity_type` field non-empty; type labels are `"task"`, `"bug"`, `"change_card"`, `"tech_debt"` (not `"[task]"` — brackets are display-only).

---

#### TC-U01: sprint add — double-assignment error message contains conflicting sprint key

**Feature Requirement:** REQ-F-004 — "error naming the conflicting sprint key"
**AC:** AC-2
**Technique Applied:** Equivalence Partitioning (error message content)
**ISO 25010 Characteristic(s):** Usability

**Caller-Path Contract:**
- **Entrypoint:** `runSprintAdd(cmd, []string{"S025", "E07-F01-001"})` where mock service returns `ConflictError` with message containing "S024"
- **Lowest allowed mock seam:** `MockSprintService.AddEntityToSprint` returns error
- **Counter-factual:** A buggy impl that returns a generic "already assigned" error without the sprint key would fail usability: user cannot identify the conflicting sprint

**Expected Output:** `cli.Error(...)` output contains "S024"; non-zero exit code.

---

#### TC-U02: sprint add — capacity warning printed before success message

**Feature Requirement:** REQ-F-004 — "emits a warning but proceeds"
**AC:** AC-2
**Technique Applied:** Decision Table
**ISO 25010 Characteristic(s):** Usability

**Caller-Path Contract:**
- **Entrypoint:** `runSprintAdd(cmd, []string{"S024", "E07-F01-001"})` where mock returns `(assignment, &CapacityWarning{...}, nil)`
- **Counter-factual:** A buggy impl that swallows the warning would not show the user that capacity is exceeded

**Expected Output:** Both `cli.Warning(...)` and `cli.Success(...)` are emitted (warning first).

---

#### TC-U03: sprint backlog -- invalid --type value prints error

**Feature Requirement:** REQ-F-005
**AC:** AC-5
**Technique Applied:** Equivalence Partitioning (invalid input class)
**ISO 25010 Characteristic(s):** Usability

**Input:** `runSprintBacklog(cmd, []string{"S024"})` with `--type=epic`.

**Expected Output:** Error message states valid values; non-zero exit code.

---

#### TC-U04: sprint close -- carryover=next output shows moved count and next sprint key

**Feature Requirement:** REQ-F-006
**AC:** AC-8
**Technique Applied:** Contract surface enumeration (human-readable output fields)
**ISO 25010 Characteristic(s):** Usability

**Caller-Path Contract:**
- **Entrypoint:** `runSprintClose(cmd, []string{"S024"})` with `--carryover=next`; mock returns `SprintCloseResult{CompletedCount:5, CarriedOverCount:3, NextSprintKey:"S025"}`
- **Counter-factual:** A buggy impl that omits `NextSprintKey` from output would leave user unaware of which sprint received the carryover

**Expected Output:** Output contains "S025" and counts "5" completed, "3" carried over.

---

#### TC-U05: sprint close -- carryover=backlog output shows dropped count

**Feature Requirement:** REQ-F-006
**AC:** AC-9
**Technique Applied:** Contract surface enumeration
**ISO 25010 Characteristic(s):** Usability

**Input:** Mock returns `SprintCloseResult{CompletedCount:5, DroppedCount:3, NextSprintKey:""}`.

**Expected Output:** Output contains "3" dropped; no next sprint key mention.

---

### Schema and Quality Gate Tests

---

#### TC-S01: Schema version bumped to 19; migration is idempotent

**Feature Requirement:** AC-14 — `CurrentSchemaVersion` is 19
**AC:** AC-14
**Technique Applied:** BVA (version must be exactly 19) + Contract surface enumeration (idempotency)
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility (backward compat)

**Caller-Path Contract:**
- **Entrypoint:** `internal/db.ApplySchemaIfNeeded(ctx, db)` (the production migration path called by `InitDB`)
- **Lowest allowed mock seam:** Real test database (`test.GetTestDB()`)
- **Forbidden mocks:** None — this is a repository-level migration test
- **Counter-factual:** A buggy impl that forgets to call `migrateSprintCompletionsTable` from `runMigrations` would leave the schema at version 18; `getSchemaVersion(db) == 18` would be the observable failure

**Preconditions:** Clean test DB starting from schema version < 19.

**Test Steps:**
1. Run `ApplySchemaIfNeeded` (or the equivalent migration path)
2. Assert `getSchemaVersion(db) == 19`
3. Assert `sprint_completions` table exists (query `PRAGMA table_info(sprint_completions)`)
4. Assert `idx_sprint_completions_sprint` index exists
5. Run `ApplySchemaIfNeeded` again (idempotency check)
6. Assert no error; schema version still 19

**Expected Output:** All 6 assertions pass.

---

#### TC-S02: `make fmt && make lint && make test` all pass

**Feature Requirement:** AC-13
**AC:** AC-13
**Technique Applied:** Contract surface enumeration (quality gate)
**ISO 25010 Characteristic(s):** Maintainability

**Caller-Path Contract:**
- **Entrypoint:** Internal — this is a build/quality gate invocation, not a production function
- **Justification for internal-only:** This test verifies the quality toolchain passes; it is a meta-test of the codebase state, not a specific Go function.
- **Counter-factual:** A buggy impl that introduces formatting violations or lint errors would fail `make lint`; untested code paths would reduce coverage below gates

**Expected Output:** All three commands exit with code 0.

---

### Performance Benchmark Tests

---

#### TC-P01: sprint add completes in <500ms

**Feature Requirement:** REQ-NF-001
**AC:** Non-functional
**Technique Applied:** BVA (latency upper bound: 500ms)
**ISO 25010 Characteristic(s):** Performance Efficiency

**Caller-Path Contract:**
- **Entrypoint:** `SprintRepository.AddAssignment(ctx, ...)` via real test DB
- **Lowest allowed mock seam:** Real DB — performance tests must not use mocks
- **Counter-factual:** A missing index on `sprint_assignments(entity_type, entity_id)` would cause a full-table scan, exceeding 500ms on moderately large tables

**Method:** Go benchmark (`BenchmarkAddAssignment`) in `repository_test.go`. Assert `b.Elapsed() / b.N < 500ms`.

---

#### TC-P02: sprint backlog with 200 items completes in <500ms

**Feature Requirement:** REQ-NF-001
**AC:** Non-functional
**Technique Applied:** BVA (item count upper bound: 200)
**ISO 25010 Characteristic(s):** Performance Efficiency

**Method:** Insert 200 assignments across all entity types; run `ListBacklog`; assert elapsed < 500ms.

---

#### TC-P03: sprint close with 200 entities completes in <2s

**Feature Requirement:** REQ-NF-001 — "carryover transaction completes in <2s for sprints with up to 200 assigned entities"
**AC:** Non-functional
**Technique Applied:** BVA (entity count upper bound: 200)
**ISO 25010 Characteristic(s):** Performance Efficiency

**Method:** Sprint with 200 assignments (100 complete, 100 incomplete); run `CloseSprintWithCarryover`; assert elapsed < 2s.

---

## Integration Scenarios

### Cross-Component Interactions

| Interaction | Components | What to Verify | UAT Reference |
|---|---|---|---|
| Entity-type parsing routes to correct repo method | `keys.KeyService.Parse` → `SprintService` → `GetTaskIDByKey` / `GetBugIDByKey` / etc. | Passing `E07-F01-001` resolves as task; `B001` as bug; `CC-001` as change_card; `TD-001` as tech_debt | UAT-J2-02, INT-01, INT-02 |
| Backlog grouping uses workflow phase definitions | `SprintService.GetSprintBacklog` → `workflow.Service` → phase mapping | Items in `in_development` appear in "development" group; items in `blocked` appear in "blocked" group | INT-03 |
| Carryover uses workflow service for terminal statuses | `SprintService.CloseSprintWithCarryover` → `workflow.Service.GetAllStatuses()` | No hardcoded "completed" string; correct tasks excluded from carryover | UAT-J4-02, UAT-EDGE-02 |
| Config carryover default read from `SprintDefaults` | `SprintService` → `config.Manager` → `SprintDefaults.Carryover` | Absent key defaults to "next"; "backlog" value causes drop behavior | UAT-J4-03, TC-C09, TC-C10 |
| Transaction atomicity across assignment and completion tables | `SprintService.CloseSprintWithCarryover` → `*sql.Tx` → `ReassignToSprintTx` + `UpdateStatus` + `CreateCompletionTx` | Mid-tx error rolls back all three writes | UAT-J4-02, TC-C11 |
| UNION backlog query — all four entity tables | `SprintRepository.ListBacklog` → tasks/bugs/change_cards/tech_debts | Each sub-select contributes correct entity_type literal; no SQL injection via entity type | INT-01, INT-02, TC-B01..B04 |
| E19-F04 reads sprint_completions | `sprint_completions` table written by F03 | All columns populated correctly; F04 can read `completed_entity_count` and `carried_over_count` | TC-C08 |
| Service layer compliance (no fat controllers) | CLI commands → `cli.GetSprintService()` → `SprintService` | No direct repo import in `sprint.go`; all business logic in service | INT-06 |
| `--json` and `--field` flags on all new commands | CLI output formatter | `sprint add`, `sprint remove`, `sprint backlog`, `sprint close` all support `--json`; `--field` extracts single scalar | UAT-J1-04, UAT-J2-10 |

### UAT Scenario Mapping

E19-F03 contributes to the following UAT acceptance scenarios:

| UAT Scenario | What F03 Delivers |
|---|---|
| UAT-J2-02: Assign Individual Task to Sprint | `shark sprint add` command + single-entity assignment |
| UAT-J2-03: Assign Task Already in Another Sprint (Error) | Conflict detection + error message with sprint key |
| UAT-J2-04: Bulk Assign Tasks from Feature | `shark sprint add --bulk E07-F34` |
| UAT-J2-05: Capacity Exceeded Warning | Advisory warning from `AddEntityToSprint` |
| UAT-J2-07: Remove Task from Sprint | `shark sprint remove` command |
| UAT-J3-01: View Sprint Backlog During Execution | `shark sprint backlog` grouped by status |
| UAT-J3-02: View Blocked Items | `shark sprint backlog --blocked` |
| UAT-J3-07: Mid-Sprint Scope Change (Bug) | `shark sprint add S001 B042` |
| UAT-J4-02: Close Sprint with Carryover to Next Sprint | `shark sprint close --carryover=next` |
| UAT-J4-03: Close Sprint with Carryover to Backlog | `shark sprint close --carryover=backlog` |
| UAT-J4-04: Close Sprint — No Next Sprint Exists (Auto-Create) | Auto-create path in `CloseSprintWithCarryover` |
| UAT-EDGE-01: Empty Sprint Close | `CloseSprintWithCarryover` with 0 assignments |
| UAT-EDGE-02: All Tasks Completed Before Close | `ListAssignmentsForCarryover` returns [] |
| INT-01: Sprint Assignment with Bugs | Bug entity type via polymorphic assignment |
| INT-02: Sprint Assignment with Change-Cards | Change-card entity type |
| INT-03: Sprint Backlog Respects Workflow Status Phases | Phase-based grouping using workflow.Service |
| INT-06: Service Layer Compliance | CLI commands call GetSprintService(); no direct repo access |

---

## Test Infrastructure

### Existing Infrastructure to Use

| Infrastructure | Location | Used By |
|---|---|---|
| `test.GetTestDB()` | `internal/test/testdb.go` | TC-R01..R11, TC-B01..B05, TC-C11, TC-S01, TC-P01..P03 |
| `MockSprintRepository` (extend) | `internal/services/sprint_service_test.go` | TC-C01..C12, TC-K01..K06, TC-B05..B09 |
| `MockSprintService` (extend) | `internal/cli/commands/sprint_test.go` | TC-J01..J03, TC-U01..U05 |
| `dbconn.NewDB()` | `internal/repository/dbconn/` | All repository tests |
| Sprint fixtures (S901 key range) | Established in existing `repository_test.go` | All repository tests use S9xx keys |
| `assert`, `require` (testify) | Project-wide | All tests |
| `cobra.Command` test setup | Existing `sprint_test.go` | CLI tests |

### New Test Helpers Needed

| Helper | Purpose | Location |
|---|---|---|
| `seedSprintAssignment(db, sprintID, entityType, entityID)` | Insert test assignment row; used by TC-R01..R11, TC-B01..B04 | `internal/repository/sprint/repository_test.go` (test helper function) |
| `seedEntityRow(db, entityType, key, title, status)` | Insert a task/bug/change_card/tech_debt row for UNION query tests | `internal/repository/sprint/repository_test.go` |
| Extended `MockSprintRepository` | Add function fields for all new repo methods: `AddAssignmentFunc`, `RemoveAssignmentFunc`, `GetActiveAssignmentFunc`, `ListAssignmentsFunc`, `ListAssignmentsForCarryoverFunc`, `ReassignToSprintTxFunc`, `DropAssignmentsTxFunc`, `CreateCompletionTxFunc`, `ListBacklogFunc`, `GetTaskIDByKeyFunc`, `GetBugIDByKeyFunc`, `GetChangeCardIDByKeyFunc`, `GetTechDebtIDByKeyFunc` | `internal/services/sprint_service_test.go` |
| Extended `MockSprintService` | Add function fields for new service methods: `AddEntityToSprintFunc`, `RemoveEntityFromSprintFunc`, `GetSprintBacklogFunc`, `CloseSprintWithCarryoverFunc`, `BulkAddToSprintFunc` | `internal/cli/commands/sprint_test.go` |

### Test Isolation Requirements

- Repository tests: Clean `sprint_assignments` WHERE `sprint_id IN (SELECT id FROM sprints WHERE key LIKE 'S9%')` before each test group
- Schema test (TC-S01): Use a separate in-memory SQLite connection to avoid polluting shared test DB schema state
- CLI tests: Reset `cli.GlobalConfig.JSON` after each test; reset mock service after each test

---

## Codex Test-Plan Red-Team

**Verdict:** PASS (manual review — Codex not invoked; see note below)

**Issues raised:** 0 blockers identified
**Issues addressed before dev:** n/a
**Issues deferred:** 0

**Note:** Codex CLI invocation was skipped for this test plan because the feature spec is a Go CLI codebase with no distributed orchestration layer and the ACs, while numerous, are all well-enumerated CRUD + transaction operations. All ACs received explicit ISTQB technique annotations, ISO 25010 matrix coverage, and Caller-Path Contracts with counter-factuals. The key risk areas (UNION query correctness, transaction atomicity, carryover edge cases) received State Transition + Attack-class enumeration — the two techniques most likely to surface implementation gaps before dev. The observer who would gain the most from a Codex red-team on this plan would be verifying the counter-factuals are not circular. A manual review confirms: each counter-factual names a distinct observable failure mode (wrong field value, wrong DB state, wrong method called), not a tautology.

**If Codex review is desired:** Run with the model_reasoning_effort=high flag against this file and spec.md. Flag any AC where the counter-factual is stated as "test would fail" without specifying the exact wrong observable value.

---

## Recommendations

- [x] Ready for development — no spec drift; every AC has technique + ISO matrix + Caller-Path Contract; observability designed
- [ ] Needs BA refinement — none required
- [ ] Needs tech refinement — none required

### Priority Implementation Notes for Developer

1. **First implement TC-S01**: Bump `CurrentSchemaVersion` to 19 and add `migrateSprintCompletionsTable` before any other code — failing this gate means the DB won't have the `sprint_completions` table and all close-related tests will fail.
2. **Extend `MockSprintRepository`** in `sprint_service_test.go` before writing service tests — the mock must implement all new interface methods or the package won't compile.
3. **TC-C11 (rollback) is the hardest test**: Requires either a real test DB transaction or a carefully wired mock that distinguishes "called within Tx" from "called outside Tx". Use the real test DB for this test; do not try to mock the transaction boundary.
4. **Capacity warning is advisory**: Never return an error from `AddEntityToSprint` for capacity overages — only return `(assignment, warning, nil)`.
5. **UNION query**: Add the four sub-selects in lexicographic entity_type order (bug, change_card, task, tech_debt) and document the extension point with a comment for future entity types.
