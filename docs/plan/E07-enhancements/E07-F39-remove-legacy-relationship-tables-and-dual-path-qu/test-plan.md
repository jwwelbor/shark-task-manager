---
feature_key: E07-F39
epic_key: E07
title: Test Plan — Remove legacy relationship tables and dual-path query code
type: test-plan
---

# E07-F39 Test Plan

## 1. AC Test Matrix

Each acceptance criterion from `spec.md §2.3` is mapped to at least one test case with input, expected outcome, and edge cases.

---

### AC-1: Legacy identifiers confined to `db.go` / `migrate.go`

**Criterion:** `grep -r "task_relationships\|feature_relationships\|epic_relationships" internal/ --include="*.go"` returns only matches inside `internal/db/db.go` and `internal/db/migrate.go`.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-1.1 | Grep assertion passes after Task 001 completion | Run `grep -r "task_relationships\|feature_relationships\|epic_relationships" internal/ --include="*.go"` on the post-implementation tree | Zero matches outside `internal/db/db.go` and `internal/db/migrate.go` | Check `cmd/` and `tools/` trees too — spec §5 flags hidden callers outside `internal/` |
| TC-1.2 | CLI migrate command removed | `grep -r "migrate_relationships" internal/ --include="*.go"` | No matches in `internal/cli/commands/` | File may be replaced; verify no shim was left behind |
| TC-1.3 | Legacy model files deleted | `ls internal/models/task_relationship.go internal/models/feature_relationship.go internal/models/epic_relationship.go` | All three files absent | A file removed from disk but still imported elsewhere would cause compile failure — caught by `make build` |
| TC-1.4 | Legacy repository files deleted | `ls internal/repository/task/relationship.go internal/repository/feature/relationship.go internal/repository/epic/relationship.go` | All three files absent | Same compile-time guarantee as TC-1.3 |
| TC-1.5 | Compile passes after deletions | `make build` | Zero compile errors | Ensures no dangling import of deleted types |

**Test infrastructure:** CI lint/build step (`make fmt && make lint && make test`). TC-1.1 can be added as a shell assertion in a CI script or as a `TestLegacyTableReferencesAbsent` function in `internal/repository/entityrel/repository_test.go` using `os/exec` to run grep.

---

### AC-2: `shark task deps` works end-to-end via `entity_relationships`

**Criterion:** `shark task deps E07-F01-001` resolves dependency chain solely through `entity_relationships`. No dual-path fallback to `task_relationships`.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-2.1 | `dependency.go` no longer queries `task_relationships` | Unit test: seed a `depends_on` relationship in `entity_relationships` for two tasks, call `TaskRepository.IsBlocked` (dependency.go) | Returns correct blocked status; no SQL touching `task_relationships` | Task where `task_relationships` has a row but `entity_relationships` does not — after migration this row is gone, so it should return `false` |
| TC-2.2 | `ValidateDependencies` finds blocking tasks via `entity_relationships` | Existing `TestTaskRepository_ValidateDependencies` (`internal/repository/task/dependency_test.go`) updated to seed `entity_relationships` | Test passes with no changes to assertions; only fixture seeding changes | Task with `depends_on` JSON field (legacy field) alongside an `entity_relationships` row — legacy field removal is out of scope; test should confirm `entity_relationships` row is checked |
| TC-2.3 | Dependency tree CLI round-trip | Integration: create two tasks, run `shark link add T-A T-B --type=depends_on`, run `shark task deps T-A` | Output shows T-B as dependency | Circular dependency: A→B→A should be detected (existing `TestDetectCycleDirect` in entityrel covers this) |
| TC-2.4 | Incoming blocked-by query (`dependency.go:314`) replaced | After removing dual-path at line 314, test that `GetBlockingTasks` returns tasks whose IDs are in `entity_relationships.from_entity_id` with `depends_on` | Correct set returned | Empty `entity_relationships` table: should return empty slice, not error |

**Existing tests to update:** `TestTaskRepository_ValidateDependencies` and `TestBuildDependencyGraph` in `internal/repository/task/dependency_test.go` — update fixtures to seed `entity_relationships` instead of `task_relationships`.

---

### AC-3: `shark task link` / `shark task unlink` round-trip via `entity_relationships`

**Criterion:** Link and unlink operations use only `entity_relationships`; no write to legacy tables.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-3.1 | `task link` creates row in `entity_relationships` | Existing test in `task_relationship_commands_test.go` (mock updated to verify `entityrel` repo call) | Row created in `entity_relationships`, not `task_relationships` | Link already exists: idempotent or conflict error (per `entityrel` existing semantics) |
| TC-3.2 | `task unlink` deletes from `entity_relationships` | Same test file, unlink scenario | Row removed from `entity_relationships`, `task_relationships` untouched (table may not exist post-migration) | Unlink of non-existent relationship: should return graceful not-found or no-op |
| TC-3.3 | `shark link list` shows linked entities | `shark link add E07-F01-001 B001 --type=related_to`, then `shark link list E07-F01-001` | B001 appears in output | List with no relationships: empty output, no error |
| TC-3.4 | No write to `task_relationships` after migration | Integration: after DROP migration runs, attempt `shark task link` | Succeeds; `task_relationships` table absent causes no error | If legacy table happened to exist (pre-migration state): test that the legacy repo files being deleted means there is no code path to write to it |

**Existing tests to update:** `internal/cli/commands/task_relationship_commands_test.go` — update mocks from `MockTaskRelationshipRepository` to `MockEntityRelationshipRepository` (or verify they already use the `entityrel` path post-E21-F11).

---

### AC-4: Three legacy tables absent after migration

**Criterion:** After `migrateDropLegacyRelationshipTables` runs, `SELECT name FROM sqlite_master WHERE type='table' AND name IN ('task_relationships','feature_relationships','epic_relationships')` returns zero rows.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-4.1 | Tables absent after migration | New test `TestMigrateDropLegacyRelationshipTables` in `internal/db/migrate_test.go`: fresh DB, run `runMigrations`, query `sqlite_master` | Zero rows returned for all three table names | DB that never had the legacy tables (fresh install): `DROP TABLE IF EXISTS` must not error |
| TC-4.2 | Migration is idempotent | Same test: run `runMigrations` twice on same DB | Second run does not error, tables still absent | DB where only two of three tables exist (partial legacy state): all three DROPs execute without error |
| TC-4.3 | `CurrentSchemaVersion` incremented to 12 | Unit test: check `db.CurrentSchemaVersion == 12` | Compile-time constant check | Schema version must be exactly 12 after this feature; if 11 is already taken, bump is enforced |
| TC-4.4 | Version guard skips migration on current DB | Test: DB with schema version 12 already set, run `runMigrations` | Migration skipped (no DDL executed) | Must not error on skip; existing `migration_polymorphic_test.go:TestMigratePolymorphicTables_Idempotent` is the reference pattern |

**Test infrastructure:** Follow the pattern in `internal/db/migration_polymorphic_test.go`. Use `internal/test` package for DB setup. New test file: `internal/db/drop_legacy_relationship_tables_migration_test.go`.

---

### AC-5: Viewer hierarchy JSON includes task links to bugs / features / epics

**Criterion:** After creating a task→bug link via `shark link add`, loading the viewer `Hierarchy()` response for that task includes the bug in the `Relationships` field.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-5.1 | Task→bug link appears in `Hierarchy()` | New unit test in `internal/services/viewer_service_test.go`: mock `entityRelSvc.GetRelationships` to return a `related_to` relationship to a bug, call `Hierarchy()`, inspect `ViewerTask.Relationships` | One `ViewerRelatedEntity` with `entity_type=bug`, `entity_key=B001`, `relationship_type=related_to`, `direction=outgoing` | Viewer returned zero tasks: no crash, empty Relationships list |
| TC-5.2 | Task→feature link appears in `Hierarchy()` | Same setup; mock returns a `related_to` relationship to a feature | `ViewerRelatedEntity` with `entity_type=feature`, `entity_key=E07-F01` | Feature linked as "blocked_by" type: correct direction attribute |
| TC-5.3 | Task→task `depends_on` appears in `Hierarchy()` | Mock returns `depends_on` relationship where both ends are tasks | `ViewerRelatedEntity` with `entity_type=task`, `direction=outgoing`, `relationship_type=depends_on` | Both incoming and outgoing relationships for the same task: both present in `Relationships` slice |
| TC-5.4 | `FeatureTasks()` includes cross-entity links | New unit test in `viewer_service_test.go`: same mocking pattern in `FeatureTasks()` path | Each `ViewerTask` in result has `Relationships` populated from `entityRelSvc` | `GetRelationships` returns error for one task: propagate error or skip that task (decision must be consistent with REQ-N-004) |
| TC-5.5 | `Relationships` field present in JSON output | JSON serialization of `ViewerTask` includes `"relationships"` key | Field is non-null, contains array | Empty array serializes as `[]` not `null` (REQ-N-003 backward compatibility) |
| TC-5.6 | REQ-N-003: `depends_on_keys` / `blocked_by_keys` / `blocks_keys` populated if kept | If backward-compat fields are kept (Q1 resolution), verify they are populated from `entity_relationships` task→task data only | Old fields contain only task keys; `Relationships` contains all entity types | If old fields are removed cleanly (Q1 resolves to clean cut), this test is skipped |

**Test infrastructure:** New tests in `internal/services/viewer_service_test.go`. Mock pattern: add `MockEntityRelationshipService` to the existing mock set following `mockViewerTaskRelRepo` shape. EntityRegistry must be mockable to resolve `(entity_type, id) → key`.

---

### AC-6: Deleted symbols absent from codebase

**Criterion:** `taskRelAdapter`, `ViewerTaskRelationshipRepository`, `ViewerTaskRelationship`, `WithTaskRelRepo` do not appear in any `.go` file.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-6.1 | `taskRelAdapter` absent | `grep -r "taskRelAdapter" internal/ --include="*.go"` | Zero matches | A comment mentioning the old name is acceptable; a type reference is not |
| TC-6.2 | `ViewerTaskRelationshipRepository` absent | Same grep pattern | Zero matches | Grep must cover test files too — old tests must be deleted, not just commented |
| TC-6.3 | `ViewerTaskRelationship` absent | Same grep pattern | Zero matches | Distinguish from `ViewerRelatedEntity` which is the replacement |
| TC-6.4 | `WithTaskRelRepo` absent | Same grep pattern | Zero matches | Wire.go must not have the call; viewer_service_test.go must have the old tests deleted |
| TC-6.5 | Compile passes without deleted symbols | `make build` | Zero compile errors | Most reliable automated check — any missed grep is caught here |

**Test infrastructure:** CI `make build`. Can also add `TestLegacyViewerSymbolsAbsent` as a grep-based test in `internal/services/viewer_service_test.go` using `os/exec`, following the same pattern as TC-1.1.

---

### AC-7: `make fmt && make lint && make test` clean

**Criterion:** Full quality gate passes with no new warnings, no skipped/disabled tests.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-7.1 | `make fmt` produces no diffs | Run `make fmt`, then `git diff` | No unstaged changes | Particularly check new adapter files and viewer_service.go edits |
| TC-7.2 | `make lint` zero warnings | `make lint` | Exit code 0, no new lint issues | Shadow variable warnings in new adapter implementations are common — address proactively |
| TC-7.3 | `make test` zero failures | `make test` | All tests pass, including updated and new tests | Packages with deleted files must compile after deletion; orphan imports detected |
| TC-7.4 | No skipped tests from deleted files | Verify deleted test files have no `//go:build ignore` remnant | Deleted files are absent, not commented out | If a test in a deleted file exercises logic now in a different file, equivalent coverage must exist elsewhere |

**Test infrastructure:** CI/Makefile. This is the master quality gate covering all other ACs.

---

### AC-8: `CurrentSchemaVersion` incremented to 12; migration idempotent on re-run

**Criterion:** Schema version is exactly 12; running `runMigrations` twice does not error.

| # | Test Case | Input / Setup | Expected Outcome | Edge Cases |
|---|---|---|---|---|
| TC-8.1 | `CurrentSchemaVersion == 12` | Compile-time: `const CurrentSchemaVersion = 12` in `internal/db/db.go` | Code compiles with value 12 | If another migration was added concurrently and version 12 is taken, bump to 13; this test plan uses 12 per spec |
| TC-8.2 | Second `runMigrations` call does not error | Test: run `runMigrations` on fresh DB (version recorded as 12), run again | Second call returns nil error, no DDL side effects | DB where schema_version row is absent: migration must set it correctly on first run |
| TC-8.3 | Migration skips DROP if tables already absent | Test: DB where the three tables never existed, run migration | Succeeds silently; `DROP TABLE IF EXISTS` semantics guarantee this | Contrast with TC-4.1: the IF EXISTS makes it safe |

**Test infrastructure:** `internal/db/drop_legacy_relationship_tables_migration_test.go`. Follow `TestMigratePolymorphicTables_Idempotent` in `internal/db/migration_polymorphic_test.go` as the exact reference.

---

## 2. Integration Scenarios

### Scenario IS-1: Full CLI round-trip after migration

**Components involved:** `shark link add/list`, `EntityRelationshipService`, `entityrel` repository, `task/dependency.go`

**Verification:**
1. Run `shark link add E07-F01-001 E07-F01-002 --type=depends_on`
2. Run `shark task deps E07-F01-001`
3. Assert E07-F01-002 appears as a dependency
4. Run `shark link add E07-F01-001 B001 --type=related_to`
5. Run `shark link list E07-F01-001`
6. Assert B001 appears; confirm it shows entity type `bug`

**Boundary checks:** `entity_relationships` table is the only table queried; `task_relationships` table is absent (post-migration).

**Epic UAT contribution:** Validates that relationship data created before migration (now in `entity_relationships` post-E21-F11) is correctly served by the refactored code paths. Directly validates the "data integrity across refactor" story of E07.

---

### Scenario IS-2: Viewer hierarchy cross-entity relationship display

**Components involved:** `ViewerService.Hierarchy()`, `EntityRelationshipService`, `EntityRegistry`, viewer HTTP endpoint

**Verification:**
1. Create a task T-A and a bug B001 in the test DB
2. Insert a `related_to` relationship in `entity_relationships` (from_entity_type=task, to_entity_type=bug)
3. Call `Hierarchy()`
4. Assert T-A's `ViewerTask.Relationships` contains a `ViewerRelatedEntity` with entity_type=bug and entity_key=B001
5. Assert `taskRelAdapter` code path is not exercised (it is deleted)

**Boundary checks:** `EntityRegistry` must resolve the bug ID to B001 key. Mock or real registry can be used; unit test uses mock.

**Epic UAT contribution:** This is the core viewer fix — the bug where task→bug links were silently dropped. Satisfies the viewer cross-entity display capability added to scope per the notes in T-E07-F39-001.

---

### Scenario IS-3: Template helper `related_tasks` / `related_features` / `related_epics` via new adapters

**Components involved:** `internal/config/template/helpers.go`, `EntityRelTaskKeyAdapter`, new `EntityRelFeatureKeyAdapter`, new `EntityRelEpicKeyAdapter`, `entityrel` repository

**Verification:**
1. Seed `entity_relationships` with: task→task, task→feature, task→epic entries
2. Execute template rendering with `{{related_tasks}}`, `{{related_features}}`, `{{related_epics}}`
3. Assert each template variable returns the correct comma-separated keys

**Boundary checks:**
- `EntityRelFeatureKeyAdapter` and `EntityRelEpicKeyAdapter` must be new (not yet existing); test file is `internal/repository/entityrel/repository_test.go` with new test functions
- Template call with no related entities: empty string result, no panic

**Epic UAT contribution:** Ensures template output for documentation generation remains correct after the legacy table references in `helpers.go` are replaced.

---

### Scenario IS-4: Schema migration on existing production-like DB

**Components involved:** `internal/db/db.go:migrateDropLegacyRelationshipTables`, `runMigrations`, `CurrentSchemaVersion`

**Verification:**
1. Open a DB at schema version 11 that has the three legacy tables present (seeded with rows)
2. Run `runMigrations`
3. Assert version is now 12
4. Assert the three tables are absent
5. Assert all existing `entity_relationships` rows are intact (data is not affected)
6. Run `runMigrations` again — no error

**Boundary checks:** DB with `skip_migrations: true` on Turso (simulated): the version-guard (`version >= CurrentSchemaVersion`) correctly skips; tables remain until the developer flips the flag.

**Epic UAT contribution:** Confirms safe deployment path. The developer instruction in spec §3.2.1 is the process gate; this test validates the mechanism.

---

### Scenario IS-5: Viewer `FeatureTasks()` O(N) query count

**Components involved:** `ViewerService.FeatureTasks()`, `EntityRelationshipService.GetRelationships` per task

**Verification:**
1. Seed a feature with 5 tasks, each with 2 outgoing relationships
2. Call `FeatureTasks()`
3. Count `GetRelationships` calls via a counting mock
4. Assert exactly 5 calls (one per task), not 1 (bulk) or 10 (per-relationship)

**Boundary checks:** Feature with zero tasks: zero `GetRelationships` calls. Task with zero relationships: `GetRelationships` called once, returns empty slice, `ViewerTask.Relationships` is empty non-nil slice.

**Epic UAT contribution:** Validates REQ-N-004 — O(N) per-task query cost is acceptable and correctly implemented, not O(N²).

---

## 3. Test Infrastructure

### 3.1 Existing tests to delete (files being removed alongside production code)

| File | Reason |
|---|---|
| `internal/repository/task/relationship_test.go` | Tests `internal/repository/task/relationship.go` which is deleted |
| `internal/repository/task_relationship_repository_test.go` | Tests legacy task relationship repository |
| `internal/repository/relationship_repositories_test.go` | Tests `feature_relationships` and `epic_relationships` repos |

These deletions are safe because the underlying production code is deleted; no coverage gap exists since the equivalent behaviour is already covered by `internal/repository/entityrel/repository_test.go`.

### 3.2 Existing tests to update (fixture changes only)

| File | Change |
|---|---|
| `internal/repository/task/dependency_test.go` | `TestTaskRepository_ValidateDependencies`, `TestBuildDependencyGraph`: seed `entity_relationships` instead of `task_relationships`. No assertion changes. |
| `internal/cli/commands/task_relationship_commands_test.go` | Update mock from `MockTaskRelationshipRepository` to `MockEntityRelationshipRepository`. Verify mock interface matches `entityrel` path. |
| `internal/cli/commands/task_get_blocking_test.go`, `task_get_blocking_integration_test.go` | Re-run; update fixtures only if they directly seed `task_relationships`. If they use the CLI, no change needed. |

### 3.3 Existing tests to update (logic changes)

| File | Change |
|---|---|
| `internal/services/viewer_service_test.go` — tests covering `taskRelRepo` injection | Delete: `TestViewerService_Hierarchy_GracefulWhenRelRepoNil`, `TestViewerService_Hierarchy_DependsOnNilToEmpty`, `TestViewerService_FeatureTasks_DependsOnNilWhenRelRepoNil`, `TestViewerTask_DependsOnFieldExists` (if it tests the old string-slice fields), and all other tests that call `svc.WithTaskRelRepo(...)` | The "graceful nil" tests are no longer valid — `entityRelSvc` is required, not optional |
| `internal/services/viewer_service_test.go` — `DependsOn`/`BlockedBy`/`Blocks` assertions | Update or delete assertions on `depends_on_keys`, `blocked_by_keys`, `blocks_keys` fields; replace with assertions on `relationships []ViewerRelatedEntity` |

### 3.4 New tests to create

| File | Test Functions | AC Coverage |
|---|---|---|
| `internal/db/drop_legacy_relationship_tables_migration_test.go` | `TestMigrateDropLegacyRelationshipTables_FreshDB` | AC-4, AC-8 |
| | `TestMigrateDropLegacyRelationshipTables_Idempotent` | AC-8 |
| | `TestMigrateDropLegacyRelationshipTables_TablesAlreadyAbsent` | AC-4 edge case |
| | `TestMigrateDropLegacyRelationshipTables_SchemaVersionBumped` | AC-8 |
| `internal/repository/entityrel/repository_test.go` (additions) | `TestEntityRelFeatureKeyAdapter_ListRelatedFeatureKeys` | AC-3, IS-3 |
| | `TestEntityRelEpicKeyAdapter_ListRelatedEpicKeys` | AC-3, IS-3 |
| | `TestEntityRelFeatureKeyAdapter_EmptyResult` | AC-3 edge case |
| `internal/services/viewer_service_test.go` (new functions) | `TestViewerService_Hierarchy_IncludesCrossEntityRelationships` | AC-5 |
| | `TestViewerService_Hierarchy_TaskBugLinkVisible` | AC-5, IS-2 |
| | `TestViewerService_Hierarchy_TaskFeatureLinkVisible` | AC-5 |
| | `TestViewerService_FeatureTasks_IncludesRelationships` | AC-5 |
| | `TestViewerService_FeatureTasks_RelationshipsPerTaskCallCount` | IS-5, REQ-N-004 |
| | `TestViewerTask_RelationshipsFieldPresent` | AC-5, AC-6 |

### 3.5 Existing infrastructure patterns to follow

| Pattern | Location |
|---|---|
| DB migration idempotency test | `internal/db/migration_polymorphic_test.go:TestMigratePolymorphicTables_Idempotent` |
| DB migration fresh-DB test | `internal/db/migration_polymorphic_test.go:TestMigratePolymorphicTables_FreshDB` |
| `entityrel` repository tests (real DB) | `internal/repository/entityrel/repository_test.go` |
| Viewer service mock pattern | `internal/services/viewer_service_test.go` — `mockViewerTaskRelRepo` shape (being deleted, but the pattern is the model) |
| Per-task call count mock | Create `countingEntityRelSvc` in `viewer_service_test.go` following the existing mock-with-function-field pattern |
| Dependency test fixture seeding | `internal/repository/task/dependency_test.go` — update to seed `entity_relationships` following `entityrel/repository_test.go` fixture pattern |

---

## 4. Quality Gates

- Every AC in spec.md §2.3 has at least one test case in this plan (confirmed above)
- AC-7 (`make fmt && make lint && make test`) is the master gate — passing it implies AC-1, AC-6, and AC-7 itself
- Deleted test files do not leave coverage gaps: equivalent coverage exists in `entityrel/repository_test.go`
- New `entityRelSvc` dependency in `ViewerService` is required (not optional) — tests must not construct `ViewerService` without it post-refactor
- REQ-N-003 backward-compatibility decision (Q1) must be resolved before TC-5.6 is written as pass/skip

---

## 5. Traceability Summary

| AC | Test Cases | Integration Scenarios |
|---|---|---|
| AC-1 (legacy identifiers confined) | TC-1.1 – TC-1.5 | IS-1 |
| AC-2 (task deps via entity_relationships) | TC-2.1 – TC-2.4 | IS-1 |
| AC-3 (link/unlink round-trip) | TC-3.1 – TC-3.4 | IS-1, IS-3 |
| AC-4 (tables absent after migration) | TC-4.1 – TC-4.4 | IS-4 |
| AC-5 (viewer cross-entity links) | TC-5.1 – TC-5.6 | IS-2, IS-5 |
| AC-6 (deleted symbols absent) | TC-6.1 – TC-6.5 | — |
| AC-7 (make test clean) | TC-7.1 – TC-7.4 | All |
| AC-8 (schema version + idempotency) | TC-8.1 – TC-8.3 | IS-4 |
