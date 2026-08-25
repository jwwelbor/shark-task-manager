# Test Plan: E32-F06 - Cleanup: retire legacy resolution paths

**Created:** 2026-08-25  
**Feature PRD:** `feature.md` (stale `E02-F06` identity; use `E32-F06`)  
**Task specification:** `spec.md`  
**Parent UAT:** `../uat-plan.md` scenarios A9 and I1  
**Status:** NEEDS_REFINEMENT

## Scope and release gate

This plan covers runtime legacy-resolution removal, explicit JSON refusal, install-time migration, active operator guidance, and conditional host-command retirement. It does not rewrite historical records.

Do not delete the eight host commands until the implementation record contains the promised F05 release tag, one full release window of normal-use evidence, and a command-directory audit. The 2026-06-22 assessment says that evidence was absent; refresh it before implementation.

## Spec drift analysis

### Drift findings

| ID | Severity | Finding | Required resolution |
|---|---|---|---|
| D-01 | RESOLVED | The stale `feature.md` fallback/error language conflicted with canonical-ignore renderer behavior. | Feature brief amended to match `spec.md` and current renderer evidence. |
| D-02 | RESOLVED | The feature had stale E02 identity fields. | Front matter and heading now identify E32-F06 under E32. |
| D-03 | WARNING | The old PRD permits only changelog legacy references; the specification preserves historical plans, QA evidence, and archival analysis. | Use the specification's active-document allowlist; do not rewrite history. |
| D-04 | WARNING | `assessment.md` says a renderer fallback remains, but research and current source show no legacy renderer pass. | Treat assessment renderer evidence as stale; keep a regression test. |
| D-05 | RESOLVED | AC-006 required an explicit release-window evidence location. | `docs/review/E32-shark-20-single-artifact-consolidation/E32-F06-cleanup-remove-fallback-paths-retire-shark-templat/release-window-audit-20260825.md` records tags, interval, and host audit. |

### Traceability matrix

| Feature requirement | Specification AC | Covered? | Notes |
|---|---|---|---|
| Canonical prompt source; no legacy fallback | AC-001, AC-005 | Yes | Resolves D-01 in favor of current architecture. |
| Legacy JSON refusal with migration help | AC-002, AC-003 | Yes | Covers explicit target and root discovery. |
| Install-time migration only | AC-004 | Yes | Covers custom/absolute data path and idempotence. |
| Retire eight commands after release window | AC-006 | Partial | External evidence is mandatory. |
| Active canonical guidance | AC-007 | Yes | Explicitly preserves history. |
| Focused and full quality gates | AC-008 | Yes | Execution gate, not substitute evidence. |

## Acceptance criteria review

| AC | Ambiguous | Testable | Complete | Result |
|---|---|---|---|---|
| AC-001 | No | Yes | Yes | Ready |
| AC-002 | No | Yes | Yes | Ready |
| AC-003 | No | Yes | Yes | Ready |
| AC-004 | No | Yes | Yes | Ready |
| AC-005 | No | Yes | Yes | Ready |
| AC-006 | No | Yes | Yes | Release evidence recorded |
| AC-007 | No | Yes | Yes | Ready |
| AC-008 | No | Yes | Yes | Ready |

## ISTQB technique application

| AC | Technique(s) | Test cases | Rationale |
|---|---|---|---|
| AC-001 | Equivalence partitioning; contract-surface enumeration | TC-001 | Canonical source classes versus populated legacy directory. |
| AC-002 | Equivalence partitioning; decision table | TC-002 | Relative, absolute, and home-expanded JSON targets. |
| AC-003 | State transition; decision table | TC-003, TC-004 | Absent/empty config, root JSON, embedded, YAML directory/index. |
| AC-004 | Decision table; state transition | TC-005 | JSON migration versus supported YAML idempotence. |
| AC-005 | Contract-surface enumeration | TC-006 | Production resolver source and shipped tree. |
| AC-006 | State transition; contract-surface enumeration | TC-007 | Evidence gate before exact eight-file deletion. |
| AC-007 | Equivalence partitioning; contract-surface enumeration | TC-008 | Active versus historical documents. |
| AC-008 | State transition | TC-009 | Focused suites before full required gate. |

## ISO 25010 coverage matrix

`N/A` means the characteristic does not materially apply. No latency SLO, platform-specific behavior, or user interface is introduced.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-001 | ✅ TC-001 | ✅ TC-001 | N/A | ✅ TC-001 | N/A | ✅ TC-009 | N/A |
| AC-002 | ✅ TC-002 | N/A | ✅ TC-002 | ✅ TC-002 | ✅ TC-002 | ✅ TC-002 | ✅ TC-009 | N/A |
| AC-003 | ✅ TC-003, TC-004 | N/A | ✅ TC-004 | ✅ TC-003 | ✅ TC-003, TC-004 | ✅ TC-004 | ✅ TC-009 | ✅ TC-004 |
| AC-004 | ✅ TC-005 | N/A | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-009 | ✅ TC-005 |
| AC-005 | ✅ TC-006 | ✅ TC-006 | N/A | N/A | ✅ TC-006 | ✅ TC-006 | ✅ TC-009 | N/A |
| AC-006 | ✅ TC-007 | N/A | ✅ TC-007 | ✅ TC-007 | ✅ TC-007 | N/A | ✅ TC-009 | N/A |
| AC-007 | ✅ TC-008 | N/A | ✅ TC-008 | ✅ TC-008 | N/A | N/A | ✅ TC-009 | N/A |
| AC-008 | ✅ TC-009 | N/A | N/A | N/A | ✅ TC-009 | N/A | ✅ TC-009 | N/A |

### Coverage gaps

- No test-design coverage gap remains; the external release evidence is recorded in the feature review directory.

## Observability design

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Canonical renderer ignores legacy tree | Internal - deterministic behavior | Internal - no log needed | Internal - no trace needed | N/A | TC-001 asserts canonical marker. |
| Runtime JSON refusal | Error return is operator evidence | Error contains all migration actions | Internal - no trace needed | N/A | TC-002 and TC-003 assert typed error and wording. |
| Install migration | Existing JSON `migrated_from` output | Existing migration message | Internal - no trace needed | N/A | TC-005 asserts output and persisted config. |
| Command retirement | Release record is evidence | Audit record lists gate evidence and files | N/A | Gate absent means no deletion | TC-007 asserts evidence before absence. |

No new instrumentation is required. Existing diagnostics and install output are the required production evidence.

## Integration scenarios

| Scenario | Components and boundary | Verification | UAT trace |
|---|---|---|---|
| Legacy refusal | CLI/config loader -> workflow parser -> YAML/embedded resolver | JSON target/root fails before `loadWorkflowFile()`; JSON never yields workflow. | A9, I1 |
| Supported recovery | `install-shark-data` -> config -> YAML loader | JSON config becomes `<shark_data_path>/workflow`; normal loading is YAML/embedded only. | A2, A9, I1 |
| Canonical rendering | Renderer -> `shark-data/prompts` or embedded bundle | Legacy sibling tree cannot affect output. | A3, A4, A9, I1 |
| Operator contract | Help/docs -> migration action | Guidance names YAML, embedded defaults, Shark-data, and installation. | A9, I1 |

## Cross-feature contract tests (I-##)

None. `spec.md` declares no E32 interaction map or map-owned `I-##` edge. Epic UAT I1 is a journey, not an invented feature-contract pointer.

## Cross-epic integration tests (X-##)

None. `spec.md` and `docs/product/cross-epic-integration-map.md` assign E32 X-05 and X-12 to E32-F04, not E32-F06. No F06 X-row is deferred.

## Test infrastructure

| Need | Existing pattern | Plan |
|---|---|---|
| Renderer resolution | `internal/templates/shark_data_renderer_test.go`, `orchestrator_renderer_test.go` | Temp project, canonical `.md` marker, legacy `.tmpl` marker, controlled cwd. |
| Workflow resolution | `internal/config/workflow/workflow_file_loading_test.go`, `internal/config/workflow_config_resolve_test.go` | Reuse `writeJSON`, YAML fixtures, `ClearWorkflowCache`, table subtests. |
| CLI diagnostics/install | `internal/cli/commands/config_test.go`, `sharkdata_cmd_test.go` | Reuse Cobra output capture and temporary projects. |
| Static active-document audit | Narrow Go/rg test beside config/command tests | Explicit active-source allowlist; historical dirs excluded. |
| Host command audit | Release evidence plus `~/.claude/commands/` | Exact paths only; never broad deletion. |

These are renderer/config/workflow/CLI/content tests. They must not create or use the real database; repository-only tests are the exception.

## Caller-path contracts

| TC | Production entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `NewOrchestratorRenderer(findTemplateDir())`, then `Render("task/in_qa.md", data)` | No mock; real filesystem/renderer | Do not pass canonical dir directly or stub directory selection, embedded reader, or cwd. | A fallback could render `LEGACY` rather than `CANONICAL`. |
| TC-002 | `workflow.LoadMultiLevelWorkflow(configPath)` with explicit JSON config | No mock; real config/filesystem | Do not call only classifier/error helpers or mock JSON parsing. | Loader could parse JSON before refusal or omit migration steps. |
| TC-003 | `workflow.LoadMultiLevelWorkflow(configPath)` with absent/empty config plus root JSON | No mock; real root sentinel | Do not call `resolveWorkflowFilePath` alone or mock root stat. | Root JSON could load rather than fail. |
| TC-004 | `workflow.LoadMultiLevelWorkflow(configPath)` for embedded/YAML cases | No mock; real YAML fixtures | Do not stub embedded defaults, directory/index resolver, or containment check. | Cutover could reject absence/YAML or weaken containment. |
| TC-005 | Cobra `shark admin install-shark-data --json` in temp project | Embedded bundle filesystem only | Do not call `ensureWorkflowConfigField` alone or mock persistence/output. | Migration could alter data path, omit `migrated_from`, or rewrite YAML. |
| TC-006 | Direct source/tree audit of `cmd/`, `internal/`, repository root | Content-only: direct filesystem | Do not grep only changed files or hide source through ignores. | Hidden resolver branch or shipped legacy dir remains. |
| TC-007 | Direct release record and eight exact command paths | Content-only: direct filesystem | Do not infer gate from status or use wildcard deletion. | Premature deletion could pass absence-only test. |
| TC-008 | Direct named active docs and `shark config validate --help` | Content-only: direct files/command help | Do not search all docs or accept support claims. | Active guidance could mislead while history hides it. |
| TC-009 | `make fmt && make lint && make test` after focused suites | No mock | Do not replace full gate with selected tests. | Formatting, lint, or unrelated regression escapes. |

## Acceptance test cases

### TC-001: Canonical prompt rendering ignores a populated legacy tree

**Feature requirement:** REQ-F-001.  
**Acceptance criterion:** AC-001.  
**Technique:** Equivalence partitioning; contract-surface enumeration.  
**ISO 25010:** Functional suitability, performance efficiency, compatibility, reliability.  
**Preconditions:** Temp project has `shark-data/prompts/task/in_qa.md` containing `CANONICAL {{.task_id}}` and `shark-templates/task/in_qa.tmpl` containing `LEGACY {{.task_id}}`; cwd is inside the project.  
**Input:** Render `task/in_qa.md` with `task_id=E32-F06-001` through TC-001's caller path.  
**Expected output:** `CANONICAL E32-F06-001`; no output contains `LEGACY`.  
**Edge cases:** Canonical disk bundle absent plus legacy tree present renders embedded canonical content; configured absolute Shark-data path renders canonical content.  
**Negative case:** Legacy-only content is never selected.  
**Observability:** Assert selected directory and canonical output marker.

### TC-002: Every explicit JSON target fails before parser selection

**Feature requirement:** REQ-F-002 and path safety.  
**Acceptance criterion:** AC-002.  
**Technique:** Equivalence partitioning; decision table.  
**ISO 25010:** Functional suitability, compatibility, usability, reliability, security.  
**Input:** `workflow_config` is `legacy/workflow.json`, an in-root absolute JSON path, and a home-expanded JSON path; each holds valid JSON.  
**Expected output:** Every case returns `ErrDeprecatedWorkflowConfigJSON` before `loadWorkflowFile()` parses JSON. Diagnostic says JSON is unsupported, remove/empty `workflow_config`, remove/rename root JSON, or run `shark admin install-shark-data`.  
**Edge cases:** Relative, absolute, and home-expanded targets.  
**Negative case:** No JSON workflow result, silent fallback, or containment bypass.  
**Observability:** Assert typed error and all migration phrases.

### TC-003: Root legacy JSON is refused with absent or empty configuration

**Feature requirement:** REQ-F-002.  
**Acceptance criterion:** AC-003 root-file clause.  
**Technique:** State transition; decision table.  
**ISO 25010:** Functional suitability, compatibility, usability, reliability.  
**Preconditions:** Temp root has a valid `.sharkworkflow.json` with a distinguishing `legacy-root-only` status.  
**Input:** Load with `{}` and with `{"workflow_config":""}`.  
**Expected output:** Both return TC-002's typed diagnostic; neither result contains `legacy-root-only`.  
**Edge cases:** Missing versus empty field.  
**Negative case:** Root JSON is never embedded defaults, YAML, or supported input.  
**Observability:** Assert error identity and phrases.

### TC-004: Embedded, YAML directory, and YAML index paths remain supported

**Feature requirement:** REQ-F-002 and REQ-F-003.  
**Acceptance criterion:** AC-003 canonical-path clause.  
**Technique:** Decision table; state transition.  
**ISO 25010:** Functional suitability, compatibility, reliability, security, portability.  
**Input:** Load (1) no config/no root JSON, (2) explicit `shark-data/workflow` directory, (3) explicit `.sharkworkflow.yaml` index, and (4) `../escape/workflow`.  
**Expected output:** First three load embedded/YAML workflows; traversal returns containment error.  
**Edge cases:** Missing disk bundle uses embedded defaults; YAML index is not JSON.  
**Negative case:** No path relaxation or JSON selection.  
**Observability:** Assert workflow source/results and validation error.

### TC-005: Explicit install migrates JSON once and preserves YAML configuration

**Feature requirement:** REQ-F-003.  
**Acceptance criterion:** AC-004.  
**Technique:** Decision table; state transition.  
**ISO 25010:** Functional suitability, compatibility, usability, reliability, security, portability.  
**Input:** Execute with JSON plus `custom-data`; JSON plus absolute in-project bundle; YAML directory; and YAML index. Run each twice.  
**Expected output:** JSON becomes `<shark_data_path>/workflow`, output has original `migrated_from`, data path stays unchanged. YAML values stay unchanged with empty `migrated_from`; second run succeeds.  
**Negative case:** No runtime JSON loading and no YAML/data-path overwrite.  
**Observability:** Assert JSON output, persisted config, and human migration message.

### TC-006: Production source and shipped tree have no legacy template resolver

**Feature requirement:** REQ-F-001.  
**Acceptance criterion:** AC-005.  
**Technique:** Contract-surface enumeration.  
**ISO 25010:** Functional suitability, performance efficiency, reliability, security, maintainability.  
**Input:** Walk executable Go under `cmd/` and `internal/`; inspect root directories named exactly `shark-templates`, excluding named generated/dev-artifact fixtures from shipped-tree assertion.  
**Expected output:** No executable source resolves `shark-templates`; no shipped repository legacy directory exists.  
**Edge cases:** Comments/test names are not runtime resolution.  
**Negative case:** Any `os.Stat`, `filepath.Join`, glob, or fallback branch resolving legacy dir fails.  
**Observability:** Keep exact matches in failure output.

### TC-007: Retire only the eight commands after evidenced release window

**Feature requirement:** REQ-F-004.  
**Acceptance criterion:** AC-006.  
**Technique:** State transition; contract-surface enumeration.  
**ISO 25010:** Functional suitability, compatibility, usability, reliability.  
**Preconditions:** Implementation record contains F05 release tag, elapsed-window dates, normal-use evidence, and live-consumer audit.  
**Input:** Audit `~/.claude/commands/{run,feature,epic,task,prd,dispatch,develop,release}.md`.  
**Expected output:** All evidence exists; all eight exact paths are absent; no other command changes.  
**Negative case:** If evidence is absent, stop before deletion and record AC-006 blocked.  
**Observability:** Preserve release evidence and exact audit list.

### TC-008: Active operator guidance names supported configuration only

**Feature requirement:** REQ-F-005.  
**Acceptance criterion:** AC-007.  
**Technique:** Equivalence partitioning; contract-surface enumeration.  
**ISO 25010:** Functional suitability, compatibility, usability, maintainability.  
**Input:** Audit `CLAUDE.md`, configuration/initialization CLI docs, route-based/workflow-profile guides, architectural overview, and `shark config validate --help`.  
**Expected output:** Active guidance describes YAML dirs/indexes, embedded defaults, `shark-data/`, and installation; it does not present legacy templates/JSON as supported. Historical records stay unchanged.  
**Negative case:** Migration warnings may name JSON only as unsupported; help does not say it validates `.sharkworkflow.json`.  
**Observability:** Preserve direct-file/help audit output.

### TC-009: Focused suites and required Go quality gate pass

**Feature requirement:** Verification plan.  
**Acceptance criterion:** AC-008.  
**Technique:** State transition.  
**ISO 25010:** Functional suitability, reliability, maintainability.  
**Input:** Run `go test ./internal/templates`, `go test ./internal/config`, `go test ./internal/config/workflow`, `go test ./internal/cli/commands`, then `make fmt && make lint && make test`.  
**Expected output:** Every command exits 0. If formatting changes source, rerun lint and full test.  
**Negative case:** Focused green results do not pass AC-008 if full gate fails.  
**Observability:** Preserve output bound to reviewed commit SHA.

## Codex test-plan red-team

**Verdict:** PENDING  
**Issues raised:** PENDING  
**Issues addressed before dev:** PENDING  
**Issues deferred:** PENDING

The red-team result will be appended after review. Any unresolved blocker keeps the status at `NEEDS_REFINEMENT`.

## Recommendations

- [x] Ready for development
- [x] Feature-owner refinement complete: D-01 and D-02 resolved.
- [x] Release-owner refinement complete: D-05 evidence attached.
- [ ] Needs technical refinement
