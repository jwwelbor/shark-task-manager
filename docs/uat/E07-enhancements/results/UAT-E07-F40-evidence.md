# UAT Evidence Compilation — T-E07-F40-001

**Feature:** E07-F40 — File logging destination for observability
**Task under UAT:** T-E07-F40-001 — Add `log_file` field to `ObservabilityConfig` and init scaffold
**Compiled:** 2026-04-18
**Compiler:** Claude (evidence only — NOT an assessor)

This document is a neutral inventory. It does NOT judge whether any criterion is met.
Codex is the sole assessor in Phase 3.

---

## 1. Scope

Task T-E07-F40-001 is a SIMPLE-tier, data-struct-only scaffolding task inside feature
E07-F40. It adds the `log_file` field to the observability config struct and ensures
`shark admin init` emits `"log_file": ""` in the scaffolded config so downstream tasks
(T-002 runtime wiring, T-003 docs) can discover and wire it.

Downstream tasks (NOT in scope for this UAT):
- T-E07-F40-002: runtime wiring, file handle lifecycle, path resolution, MkdirAll, fallback
- T-E07-F40-003: documentation, release notes
- T-E07-F40-004: (additional)

---

## 2. Acceptance Criteria (from task + feature PRDs)

### Task-level ACs (T-E07-F40-001.md)

- **AC-T1** — `ObservabilityConfig` round-trips a non-empty `log_file` value through JSON
  marshal/unmarshal unchanged.
- **AC-T2** — `shark admin init` produces a `.sharkconfig.json` whose
  `observability` object contains an explicit `"log_file": ""` key.
- **AC-T3** — All pre-existing observability fields remain unaffected (no renames,
  no removals, no behavior change).

### Feature-level requirements that intersect this task

- **REQ-F-001** — `ObservabilityConfig` supports a `log_file` string field that
  round-trips through JSON.
- **REQ-F-009** — Scaffolded default config includes `"log_file": ""` so operators
  can discover the option.
- Feature Test Plan cases **9** and **10** are in this task's scope
  (cases 1–8 belong to T-002 / T-003).

---

## 3. Evidence Sources

| Artifact | Path |
|---|---|
| Epic PRD | `docs/plan/E07-enhancements/epic.md` |
| Feature PRD | `docs/plan/E07-enhancements/E07-F40-file-logging-destination-for-observability/feature.md` |
| Task spec | `docs/plan/E07-enhancements/E07-F40-file-logging-destination-for-observability/tasks/T-E07-F40-001.md` |
| Code Review v1 (PASS) | `docs/plan/E07-enhancements/E07-F40-file-logging-destination-for-observability/code_review/20260418-181825-T-E07-F40-001-code-review.md` |
| Code Review v2 (PASS post-fix) | `docs/plan/E07-enhancements/E07-F40-file-logging-destination-for-observability/code_review/20260418-182806-T-E07-F40-001-review-v2.md` |
| QA Report v1 (FAIL — BUG-001) | `docs/plan/E07-enhancements/E07-F40-file-logging-destination-for-observability/qa_reports/20260418-182206-T-E07-F40-001-qa.md` |
| QA Report v2 (PASS) | `docs/plan/E07-enhancements/E07-F40-file-logging-destination-for-observability/qa_reports/20260418-183157-T-E07-F40-001-qa-v2.md` |
| Task decomposition review | `docs/plan/E07-enhancements/E07-F40-file-logging-destination-for-observability/task_reviews/E07-F40-task-review.md` |

### Implementation / test files (for codex to read directly)

| File | Purpose |
|---|---|
| `internal/config/config.go` | Defines `ObservabilityConfig` struct |
| `internal/config/manager.go` | `Load()` parses raw map into `ObservabilityConfig` — BUG-001 fix location |
| `internal/config/manager_test.go` | Contains `TestLoadConfig_ObservabilityLogFile` regression test |
| `internal/init/types.go` | `ObservabilityConfigDefault` mirror struct for init scaffold |
| `internal/init/config.go` | `createConfig` emits scaffolded JSON |
| `internal/init/config_test.go` | `TestCreateConfig` and `TestCreateConfigShape` verify scaffold |
| `internal/cli/root.go` | Wiring entry point — `loadedCfg.GetObservability()` |

---

## 4. Evidence Map (criterion → artifacts, no judgment)

| # | Criterion | Evidence available | Evidence references |
|---|---|---|---|
| AC-T1 | JSON round-trip for `log_file` | Yes | Struct+tag in `internal/config/config.go`; Load-path extraction at `internal/config/manager.go` (3-line block added per BUG-001 fix); regression test `TestLoadConfig_ObservabilityLogFile` in `internal/config/manager_test.go` (4 sub-cases); QA v2 confirms live end-to-end round-trip |
| AC-T2 | Scaffold emits `"log_file": ""` | Yes | `ObservabilityConfigDefault.LogFile` with `json:"log_file"` (no omitempty) in `internal/init/types.go`; `LogFile: ""` default assignment in `internal/init/config.go`; `TestCreateConfig` (value check) and `TestCreateConfigShape` (key-presence check) in `internal/init/config_test.go` |
| AC-T3 | Pre-existing fields unaffected | Yes | Code review v1+v2 both confirm purely additive diff; QA v2 reports 55/55 packages pass `make test`; no renames/removals in any diff |
| REQ-F-001 | Feature-level round-trip requirement | Yes | Same as AC-T1 |
| REQ-F-009 | Feature-level scaffold requirement | Yes | Same as AC-T2 |
| Test case 9 | Scaffold emits `log_file` empty-string | Yes | `TestCreateConfig` in `internal/init/config_test.go` |
| Test case 10 | Scaffold JSON contains `log_file` key | Yes | `TestCreateConfigShape` in `internal/init/config_test.go` |

### Cross-cutting / red-team items for codex to verify

- **Wiring integrity:** `internal/cli/root.go` should call `cfg.GetObservability()` so
  the loaded `LogFile` value is actually available to the observability subsystem.
  (Filesystem I/O / file-handle open is explicitly deferred to T-002.)
- **Path traversal / injection:** this task is string-only. No filesystem I/O is
  performed against `log_file` at this layer. Security validation is explicitly
  deferred to T-002. Flag if you believe any validation is required NOW.
- **JSON tag semantics asymmetry:** `ObservabilityConfig.LogFile` uses
  `json:"log_file,omitempty"` (user configs stay clean); `ObservabilityConfigDefault.LogFile`
  uses `json:"log_file"` (scaffold must always emit the key). Flag if this is
  incorrect.
- **BUG-001:** v1 QA found that `manager.Load()` extracted every observability field
  EXCEPT `log_file`, so runtime parse silently dropped user values. v2 adds a 3-line
  comma-ok block at `internal/config/manager.go` to extract it. Verify the fix is
  correctly placed and present.
- **Call-site verification:** confirm `LogFile` has at least one non-test reference
  so it's not dead code at the data-struct level. (Downstream consumers wire it in
  T-002, but at minimum the extraction block in `manager.go` must reference it.)

---

## 5. Known Issues from Project Tracking

- **BUG-001** (found v1 QA, fixed in commit `cc8f560`): `internal/config/manager.go`
  `Load()` originally extracted 10 observability fields via comma-ok type assertions
  but omitted `log_file`. Without the fix the struct field round-trips via direct
  marshal/unmarshal but runtime load-path silently drops the value. v2 QA confirmed
  the 3-line fix works.

No other known issues.

---

## 6. Notes for the Assessor (Codex)

This is a deliberately narrow data-scaffold task. Filesystem I/O, path handling,
MkdirAll, open-failure fallback, and logger rebinding all belong to **T-E07-F40-002**,
which is NOT under UAT here. Do not reject this task for missing functionality that
the feature PRD explicitly defers to downstream tasks.

Do reject if:
- The struct field, scaffold default, or runtime extraction is missing or wrong.
- Any pre-existing observability field was broken.
- The BUG-001 fix is incomplete or missing.
- The new field is truly dead code (no non-test references anywhere).
- There is an actual security exposure at the string-field layer (not deferred).
- `make fmt && make lint && make test` would fail.
