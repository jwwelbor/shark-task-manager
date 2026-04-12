---
feature: E27-F01
title: DB Init Extraction - Shared Database Initialization Package
reviewer: task-review-agent
date: 2026-04-11
verdict: PASS
---

# Task Review: E27-F01 — DB Init Extraction

## Verdict: PASS

All four tasks cover the feature requirements with correct sequencing, atomic
scoping, declared dependencies, and sufficient detail for a developer agent to
implement without ambiguity.

---

## Feature Requirements Coverage

| Requirement | Source | Covered By |
|-------------|--------|-----------|
| Create `internal/dbinit` package with `Init`/`MustInit`/`Options` | spec REQ-F-001, REQ-F-002 | T-001 |
| Backend selection (sqlite/local/turso/unsupported) | spec REQ-F-003 | T-001 |
| Turso auth-token loading (`auth_token_file`, `TURSO_AUTH_TOKEN`) | spec REQ-F-004 | T-001 |
| `skip_migrations` fast-path preserved | spec REQ-F-005 | T-001 |
| `internal/cli/db_init.go` reduced to delegate ≤15 lines | spec REQ-F-006, AC-7 | T-002 |
| `db_helper.go` wrappers preserved (Option A) | spec step 9 | T-002 |
| `cmd/server/main.go` swaps `db.InitDB` → `dbinit.Init` | spec REQ-F-007 | T-003 |
| Logging parity (`slog` lines preserved) | spec REQ-NF-003 | T-003 |
| `*repository.DB` SQL accessor if missing | spec step 6 | T-003 |
| Unit tests ≥80% line coverage on `internal/dbinit` | spec REQ-NF-002, AC quality gate | T-004 |
| No circular imports (`internal/dbinit` ≠ import `internal/cli`) | spec REQ-F-008, AC-8 | T-001 (design) + T-004 (TC-U-002) |
| All existing CLI and server tests pass unchanged | spec REQ-NF-002 | T-002, T-003, T-004 |

All 8 spec acceptance criteria (AC-1 through AC-8) are directly represented in
task acceptance criteria.

---

## Task-by-Task Assessment

### T-E27-F01-001 — Create internal/dbinit package (execution_order: 1)

**Scope:** Atomic. Covers the entire `internal/dbinit` package: `init.go`,
`project_root.go`, all helpers (`resolveProjectRoot`, `resolveConfigPath`,
`loadDatabaseConfig`, `initLocal`, `initTurso`), dispatch logic, `MustInit`.

**Detail quality:** Excellent. Public API signatures defined, all helper
signatures defined, dispatch switch specified with exact case strings, error
prefix conventions documented, risk items (R-1 accessor collision, R-4 Turso
type-assert) called out explicitly.

**Dependency declaration:** Correct — depends on nothing; T-002 and T-003
blocked by this task.

**Constraint check:** The constraint "do NOT modify `internal/cli/db_init.go`,
`cmd/server/main.go`, or test files in this task" is explicitly stated, keeping
the task atomic.

**Concerns:** None. The note about a potential `LoadDatabaseConfig` export
needed in T-002 is pre-identified and handled.

### T-E27-F01-002 — Migrate internal/cli/db_init.go to delegate (execution_order: 2)

**Scope:** Atomic. Covers exactly `internal/cli/db_init.go` collapse and
`internal/cli/db_helper.go` thin-wrapper updates. Files NOT to touch are
explicitly listed.

**Detail quality:** Good. The exact replacement code (~15 lines) is provided.
Option A (preserve wrappers as thin forwarders) is selected and justified.
Verification commands are specified with expected outcomes.

**Dependency declaration:** Correct — blocked by T-001; does not block T-003
(parallel execution possible).

**Concerns:** The task notes that exporting `LoadDatabaseConfig` from
`internal/dbinit` may be needed; this is handled inline ("add that export to
`internal/dbinit/init.go` created in T-001") with no ambiguity about where the
change lands. No issue.

### T-E27-F01-003 — Migrate cmd/server/main.go to use dbinit.Init (execution_order: 3)

**Scope:** Atomic. Covers `cmd/server/main.go` only; `services.go` and
`main_test.go` are explicitly unchanged.

**Detail quality:** Good. Current code block identified and replacement block
provided verbatim. Accessor addition on `*repository.DB` is specified with
name-collision guidance. `defer` placement risk with `os.Exit` is explicitly
flagged.

**Dependency declaration:** Correct — blocked by T-001; runs in parallel with
T-002.

**One minor note:** The spec's architecture section uses `repoDB.DB()` for
`db.CheckIntegrity`, while the task's replacement code uses `repoDB.SQL()`. The
task notes this discrepancy ("If the field is named `DB` and `DB()` would
collide, use `SQL()`") and defers the final name to inspection, which is the
correct approach. Not a blocker.

### T-E27-F01-004 — Write unit tests for internal/dbinit package (execution_order: 4)

**Scope:** Atomic. Covers test files only: `init_test.go`,
`project_root_test.go`, `init_integration_test.go`.

**Detail quality:** Excellent. All 23 unit test IDs (TC-U-003 through TC-U-025)
from the test plan are listed with per-case setup and expected results. 6
project-root scenarios (TC-U-012 through TC-U-017) enumerated. Integration tests
gated behind `//go:build integration` tag — correct pattern. Quality gate
commands specified.

**Dependency declaration:** Correct — blocked by T-001; can run in parallel with
T-002 and T-003.

**Concerns:** None. The note about black-box vs white-box package choice
(`dbinit_test` vs `dbinit`) is correctly handled.

---

## Sequencing and Parallelism

```
T-001 (foundation)
  ├── T-002 (CLI delegate)
  ├── T-003 (server) [parallel with T-002]
  └── T-004 (tests)  [parallel with T-002 and T-003]
```

This is correct and matches the spec's implementation plan. T-001 must complete
first; T-002, T-003, T-004 can proceed in any order after T-001.

---

## Gap Analysis

No gaps found. All feature requirements, acceptance criteria, and test plan
items are traceable to at least one task. No requirement is split awkwardly
across tasks in a way that would create integration ambiguity.

---

## Issues

None blocking. One informational note:

- **Note (non-blocking):** T-003 and spec architecture disagree on whether the
  `*repository.DB` sql accessor is named `DB()` or `SQL()`. Both the task and
  the spec acknowledge this is resolved at implementation time by inspecting the
  struct. The developer agent is correctly guided to inspect before deciding.
  No action needed at task review stage.
