---
feature_key: E19-F01-sprint-database-schema-core-entity-foundation
epic_key: E19
title: Test Plan — Sprint Database Schema & Core Entity Foundation
spec_version: 1.0
last_updated: 2026-05-05
complexity: STANDARD
---

# Test Plan — E19-F01

> **Concise plan.** STANDARD-tier feature. The full enumerated test list (≥21 tests) lives in `spec.md` §6 and is **not duplicated here**. This document adds: (1) AC→test traceability, (2) test categories + patterns to follow, (3) explicit edge-case coverage gaps, (4) risk-based prioritization. Treat spec §6 as the canonical test inventory.

---

## 1. AC → Test Traceability Matrix

One row per AC from spec §7, mapped to the test name and target file. All test names match those in spec §6.

| AC | Asserts | Test name | File |
|---|---|---|---|
| AC-1  | `sprints` table + columns exist | `TestMigrateSprintTables_CreatesAllThreeTables` | `internal/db/sprint_tables_migration_test.go` |
| AC-2  | `sprint_assignments` table + columns exist | `TestMigrateSprintTables_CreatesAllThreeTables` | `internal/db/sprint_tables_migration_test.go` |
| AC-3  | No `task_id` FK on `sprint_assignments` (polymorphic) | `TestMigrateSprintTables_CreatesAllThreeTables` (column-list assertion) | `internal/db/sprint_tables_migration_test.go` |
| AC-4  | Partial unique index enforces one-active-sprint-per-entity | `TestMigrateSprintTables_PartialUniqueIndex` | `internal/db/sprint_tables_migration_test.go` |
| AC-5  | `sprint_capacity` table + `UNIQUE(sprint_id, agent_type)` | `TestMigrateSprintTables_CreatesAllThreeTables` (+ unique-constraint assertion) | `internal/db/sprint_tables_migration_test.go` |
| AC-6  | FK `sprint_id ON DELETE CASCADE`; no DB CHECK on entity_type | `TestMigrateSprintTables_NoEntityTypeCheckConstraint` (CHECK absence) + cascade tests cover FK | `internal/db/sprint_tables_migration_test.go` |
| AC-7  | All seven indexes exist | `TestMigrateSprintTables_CreatesAllIndexes` | `internal/db/sprint_tables_migration_test.go` |
| AC-8  | Migration idempotent | `TestMigrateSprintTables_Idempotent` | `internal/db/sprint_tables_migration_test.go` |
| AC-9  | `CurrentSchemaVersion == 18` | `TestSchemaVersionBumpedTo18` | `internal/db/sprint_tables_migration_test.go` |
| AC-10 | Cascade-delete triggers exist for all 4 parents | `TestMigrateSprintTables_CascadeDeleteFrom{Task,Bug,ChangeCard,TechDebt}` (4 tests) | `internal/db/sprint_tables_migration_test.go` |
| AC-11 | `S###` parsed as `EntityTypeSprint` | New cases in `TestKeyService_DetectEntityType`, `TestKeyService_Parse`, `TestKeyService_Normalize`, `TestKeyService_Format` | `internal/keys/service_test.go` |
| AC-12 | `IsSprintKey()` matches case-insensitively | `TestIsSprintKey` | `internal/keys/validation_test.go` |
| AC-13 | `Sprint.Validate()` enforces structural rules | `TestSprint_Validate_*` | `internal/models/sprint_test.go` |
| AC-14 | `SprintAssignment.Validate()` enforces entity_type allowlist | `TestSprintAssignment_Validate_*` + `TestValidateSprintAssignmentEntityType_{Valid,Invalid}` | `internal/models/sprint_test.go` |
| AC-15 | `make fmt && make lint && make test` all pass | CI gate (manual at handoff) | — |
| AC-16 | All §6 tests present and green | Full `go test ./...` | — |
| AC-17 | No existing tests regress | Full `go test ./...` | — |

**Coverage gap check:** every AC except AC-15/16/17 (cross-cutting CI gates) maps to at least one named test. Nothing in spec §6 is orphaned from an AC.

---

## 2. Test Categories & Existing Patterns to Follow

| Category | Files | Pattern to mirror | Real DB? |
|---|---|---|---|
| **Migration / schema integration** | `internal/db/sprint_tables_migration_test.go` (new) | `internal/db/entity_notes_migration_test.go` and `internal/db/change_cards_table_test.go` (closest analogues). Use `test.GetTestDB()` per `.claude/rules/testing/repository-tests.md`. Clean up sprint rows in setup with `DELETE FROM sprint_assignments`, etc. | **Yes** (real DB — this is the only category that uses one) |
| **Key parser unit** | extend `internal/keys/service_test.go` | Existing table-driven cases for `B###`/`CC-###`/`TD-###` in the same file. No DB. | No |
| **Key helper unit** | extend `internal/keys/validation_test.go` | `TestIsTechDebtKey` is the closest mirror. | No |
| **Model / validation unit** | `internal/models/sprint_test.go` (new) | `internal/models/validation_test.go` patterns for `ValidateEpicKey`/`ValidateTechDebtKey`; pure-logic table-driven. Per `.claude/rules/testing/architecture.md` §4 — no DB, no I/O. | No |

All categories follow the **testing golden rule**: only the migration tests touch the real test DB; everything else uses pure unit-test patterns.

---

## 3. Edge Cases Requiring Explicit Coverage

The cases below are called out explicitly because they are the most likely places a fast implementation would silently fail. Each maps to a test in spec §6, but is highlighted here so reviewers can verify it was not skipped.

| Edge case | Why it matters | Where covered (spec §6) |
|---|---|---|
| **`S001` vs `S1` vs `S0001`** boundary | Spec mandates **strict 3-digit zero-padded** — a regex written as `^S\d+$` would silently accept `S1` and `S0001`. Must reject both. | §6.2 row 3 (`"S0"`, `"S1"`, `"S0001"` → `EntityTypeUnknown`) |
| **Lowercase input `s024`** to `Parse()` vs to `ValidateSprintKey()` | `KeyService.Parse()` normalizes upstream → must accept `s024`. `ValidateSprintKey()` is the post-normalize gate → must reject `s024`. Two different contracts; tests must cover both directions. | §6.2 row 2 (Parse accepts) + §6.3 `TestValidateSprintKey_Invalid` (validator rejects) |
| **Empty string** to all validators | Trivially missed — `""` must fail every validator and be rejected by every `IsXxxKey()` helper. | §6.3 `TestValidateSprintKey_Invalid`, `TestValidateSprintAssignmentEntityType_Invalid` |
| **Empty `entity_type` on `sprint_assignments`** at the DB layer | Per §3.4, the DB has *no* CHECK; insert with `entity_type=''` therefore *succeeds* at the DB layer. App-layer `ValidateSprintAssignmentEntityType("")` must reject it. Both behaviours need tests so the post-B018 convention is verifiable. | §6.1 `TestMigrateSprintTables_NoEntityTypeCheckConstraint` (DB allows arbitrary string) + §6.3 (validator rejects empty) |
| **`removed_at` partial unique index behaviour** | Two rows with same `(entity_type, entity_id)` and both `removed_at IS NULL` → must conflict. The same two rows where one has a non-null `removed_at` → must succeed. This is the exact integrity guarantee for REQ-F-004 AC-5. | §6.1 `TestMigrateSprintTables_PartialUniqueIndex` (both directions) |
| **`start_date == end_date`** (boundary on `CHECK (start_date < end_date)`) | Strict `<` means equal dates must be rejected. Spec §3.3 uses `<` not `<=`, so equal is invalid. | §6.1 `TestMigrateSprintTables_StartEndDateCheck` — extend to also assert equal-dates rejected (verify during implementation) |
| **`SPRINT-1`, `S-001`, `Sprint001`** lookalikes | The `S###` pattern is short — common typos must not parse as sprint. | §6.2 row 3 (`"SPRINT-1"` → `EntityTypeUnknown`) |
| **Cascade delete from each parent type** | Easy to write three triggers and forget the fourth (notably `tech_debts`, the most recently added parent). | §6.1 `TestMigrateSprintTables_CascadeDeleteFrom{Task,Bug,ChangeCard,TechDebt}` (one per parent) |
| **`UNIQUE(sprint_id, agent_type)` on `sprint_capacity`** | Not explicitly listed as its own test in §6.1, but it is part of the schema. **Recommend adding** a sub-assertion to `TestMigrateSprintTables_CreatesAllThreeTables` that inserts duplicate `(sprint_id, agent_type)` and asserts `UNIQUE constraint failed`. | Coverage gap — see §5 below |
| **Idempotency under partial state** | Idempotent test in §6.1 runs migration twice on a clean DB. Should also run once, manually drop the `sprint_capacity` index, and verify a second run does **not** rebuild it (idempotency check looks at table-existence only). Not strictly required for STANDARD tier but worth a note. | Optional — not required to ship |

---

## 4. Risk-Based Prioritization

The schema-migration risk is the dominant threat (per `database-critical.md`: "Bumping `CurrentSchemaVersion` is the only required step to make a new migration run on existing databases" — forgetting it ships a broken release). The tests below are **non-negotiable for merge**:

### P0 — Must pass to merge (schema-migration safety)

1. `TestSchemaVersionBumpedTo18` — directly catches the highest-frequency migration bug.
2. `TestMigrateSprintTables_Idempotent` — verifies the migration is safe on databases that already have it.
3. `TestMigrateSprintTables_CreatesAllThreeTables` + `TestMigrateSprintTables_CreatesAllIndexes` — verifies the migration actually ran end-to-end.
4. `TestMigrateSprintTables_PartialUniqueIndex` — the integrity guarantee that downstream sprint-assignment logic (E19-F03) will rely on. If broken, every dependent feature is broken.
5. `TestMigrateSprintTables_NoEntityTypeCheckConstraint` — verifies the post-B018 convention is honoured; regression here re-creates exactly the problem B018 fixed.

### P1 — Must pass to merge (correctness)

6. All four cascade-delete tests (`TestMigrateSprintTables_CascadeDeleteFrom*`) — easy to omit one trigger.
7. `TestMigrateSprintTables_StartEndDateCheck` — only DB-level CHECK on `sprints`; if missing, the model validator becomes the sole gate.
8. `TestKeyService_Parse` / `TestKeyService_DetectEntityType` (sprint cases) — every CLI command in E19-F02+ depends on this.
9. `TestSprint_Validate_*`, `TestSprintAssignment_Validate_*` — replaces the absent DB CHECK with app-layer enforcement.

### P2 — Should pass, lower blast radius

10. `TestKeyService_Normalize`, `TestKeyService_Format` (sprint cases) — symmetry/round-trip; useful but not directly safety-critical.
11. `TestIsSprintKey` — convenience helper, less load-bearing than `KeyService.Parse()`.

### Risk summary

| Risk (from spec §8) | Caught by |
|---|---|
| Forgot `CurrentSchemaVersion` bump | P0 #1 |
| Re-introduced `entity_type` CHECK | P0 #5 |
| Partial unique index missing or wrong | P0 #4 |
| Cascade trigger missing for one parent | P1 #6 |
| `S###` regex too permissive (accepts `S1`/`S0001`) | P1 #8 (`TestKeyService_DetectEntityType`) |
| Slug collisions across sprints | Out of scope this feature; deferred to E19-F02 lookup tests |

---

## 5. Recommended Additions Beyond Spec §6

Two small additions that close gaps identified in §3 above:

1. **`sprint_capacity` UNIQUE constraint test** — Add as a sub-assertion in `TestMigrateSprintTables_CreatesAllThreeTables`, or as a stand-alone `TestMigrateSprintTables_SprintCapacityUniqueConstraint`. Inserts two rows with the same `(sprint_id, agent_type)` and asserts `UNIQUE constraint failed`. Cheap, closes AC-5.
2. **`start_date == end_date` boundary** — Extend `TestMigrateSprintTables_StartEndDateCheck` to assert equal-date insert is also rejected, not just end-before-start. One extra assertion, no extra test scaffold.

Both are low-effort, increase confidence, and do not require expanding scope.

---

## 6. Acceptance for "Test Planning Done"

This plan is complete when:

- [x] Every AC in spec §7 maps to at least one named test (§1 above).
- [x] Test categories + reference patterns are identified (§2).
- [x] Edge cases that are easy to silently miss are called out (§3).
- [x] Risk-based prioritization classifies tests P0/P1/P2 (§4).
- [x] Coverage gaps in spec §6 are flagged with concrete additions (§5).

Implementation proceeds against spec §6 as the canonical test list. This plan governs which tests are merge-blockers and how to interpret the spec's table.

*End of test plan.*
