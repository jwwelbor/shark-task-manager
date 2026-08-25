# Test Plan: E32-F06 - Cleanup — remove fallback paths, retire shark-templates/

**Created:** 2026-08-25  
**Feature PRD:** `docs/plan/E32-shark-20-single-artifact-consolidation/epic.md`  
**Feature specification:** `docs/plan/E32-shark-20-single-artifact-consolidation/E32-F06-cleanup-remove-fallback-paths-retire-shark-templat/spec.md`  
**Status:** APPROVED FOR DEVELOPMENT, with release-window gate on AC-008

## Spec Drift Analysis

### Drift Findings

The legacy `feature.md` is stale: it identifies itself as E02-F06/E02, says the
renderer still has a fallback, asks for broad repository-wide text removal, and
does not describe the explicit JSON refusal. The authoritative `spec.md`
corrects all four points. AC-010 repairs the identity; the implementation must
use `spec.md`, preserve historical records, and not recreate renderer work.

The 2026-06-22 assessment is also stale about the renderer; the research report
confirms that `findTemplateDir()` is already canonical-only. It remains useful
only as historic release-window evidence. No release evidence attached to this
feature currently satisfies AC-008, so external command deletion is prohibited
until the specified evidence is attached.

### Traceability Matrix

| Feature requirement | ACs | Coverage | Notes |
|---|---|---|---|
| Reject every root legacy JSON before parsing | AC-001, AC-002 | Yes | TC-001, TC-002, TC-003 |
| Reject explicit JSON; retain explicit migration | AC-003 | Yes | TC-004 |
| Preserve canonical YAML, configured bundle, inline, overrides, embedded defaults | AC-004 | Yes | TC-005 |
| Prompts remain canonical-only | AC-005, AC-006 | Yes | TC-006, TC-007 |
| Retire commands only after compatibility window | AC-008 | Yes | TC-009 |
| Correct current CLI and documentation guidance only | AC-007, AC-009 | Yes | TC-008, TC-010 |
| Repair planning identity | AC-010 | Yes | TC-011 |

No I-## or X-## applies: the specification expressly assigns E32's X-05 and
X-12 to E32-F04 and declares no E32 interaction map. Therefore no shared
contract test pointer is required or invented here.

## Acceptance Criteria Review

All ten ACs are unambiguous and testable. AC-008 is deliberately state-gated:
the negative branch (no qualifying evidence) is a release-blocking test, not a
reason to delete external files. The source `feature.md` identity mismatch is
resolved by AC-010 before task authoring.

## AC Test Matrix

| AC | Test case | Setup/input | Expected outcome | Edge or negative case |
|---|---|---|---|---|
| AC-001 | TC-001 root JSON with omitted field | temp project: `{}` plus valid/malformed root JSON | typed deprecation error; no workflow loaded | malformed content gives same typed error |
| AC-002 | TC-002 root JSON with empty field | `workflow_config: ""` plus root JSON | same typed error | whitespace-only field is treated as empty |
| AC-003 | TC-004 explicit JSON migration | explicit relative and absolute JSON target; override sentinel | loader rejects; install migrates to `workflow/`, sentinel remains | YAML index is not rejected |
| AC-004 | TC-005 supported-source matrix | no JSON, embedded/default, YAML dir/index, custom absolute data root, inline/override | each resolves canonical expected source | missing JSON still permits default |
| AC-005 | TC-006 renderer ignores retired tree | fixture with only retired tree, then canonical prompts | retired tree cannot supply output; canonical renders | absolute configured bundle wins |
| AC-006 | TC-007 production-tree audit | tracked `cmd`, `internal`, shipped directories | no production reference/tree; test fixture exception only | historical docs are excluded |
| AC-007 | TC-008 command help | real Cobra `config validate --help` | current YAML/embedded terms, no JSON-validation claim | deprecated filename absent |
| AC-008 | TC-009 release-gate decision table | evidence absent / fully present | absent: stop, preserve eight files; present: exact eight absent | incomplete or stale evidence blocks |
| AC-009 | TC-010 active-doc inventory | six listed current docs plus historical samples | current docs accurate; historical files byte-identical | historical legacy terms permitted |
| AC-010 | TC-011 identity metadata | `feature.md` front matter | E32-F06 and E32 exactly | stale E02 values fail |

## ISTQB Technique Application (per AC)

| AC | Technique(s) | Test cases | Rationale |
|---|---|---|---|
| AC-001 | Equivalence partitioning; attack-class enumeration | TC-001, TC-003 | Omitted configuration plus valid/malformed legacy contents must share refusal. |
| AC-002 | Equivalence partitioning | TC-002 | Empty, whitespace, and omitted selections are distinct input partitions. |
| AC-003 | Decision table; contract-surface enumeration | TC-004 | Target kind and bundle location determine reject/migrate/preserve behavior. |
| AC-004 | Decision table | TC-005 | Mutually exclusive source-selection branches must retain precedence. |
| AC-005 | Equivalence partitioning | TC-006 | Retired-only vs canonical-present prompt trees prove selection. |
| AC-006 | Contract-surface enumeration | TC-007 | Enumerate shipped Go paths and directory artifact boundary. |
| AC-007 | Equivalence partitioning | TC-008 | Help must include supported claims and exclude retired claims. |
| AC-008 | State transition; decision table | TC-009 | Evidence state controls the only permitted destructive transition. |
| AC-009 | Content inventory enumeration | TC-010 | Current and historical documentation have intentionally different rules. |
| AC-010 | Exact-value assertion | TC-011 | Metadata is a two-field traceability contract. |

## ISO 25010 Coverage Matrix

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-001 | N/A internal branch | ✅ TC-001 | ✅ error review | ✅ TC-003 | ✅ TC-003 | ✅ focused test | N/A no OS-specific behavior |
| AC-002 | ✅ TC-002 | N/A internal branch | ✅ TC-002 | ✅ error review | ✅ TC-002 | ✅ TC-002 | ✅ focused test | N/A |
| AC-003 | ✅ TC-004 | N/A | ✅ TC-004 | ✅ migration output | ✅ TC-004 | ✅ path-containment regression | ✅ focused test | ✅ relative/absolute variants |
| AC-004 | ✅ TC-005 | ✅ no new scans review | ✅ TC-005 | N/A no user-facing change | ✅ TC-005 | ✅ existing containment retained | ✅ focused test | ✅ relative/absolute variants |
| AC-005 | ✅ TC-006 | ✅ no probe regression | ✅ TC-006 | N/A internal resolver | ✅ TC-006 | ✅ unreviewed tree ignored | ✅ focused test | ✅ absolute-root variant |
| AC-006 | ✅ TC-007 | N/A static audit | ✅ TC-007 | N/A | ✅ TC-007 | ✅ retired source unavailable | ✅ audit is maintainability guard | N/A |
| AC-007 | ✅ TC-008 | N/A | ✅ TC-008 | ✅ TC-008 | N/A | N/A no input boundary | ✅ command regression | N/A |
| AC-008 | ✅ TC-009 | N/A | ✅ gate protects users | ✅ handoff report | ✅ TC-009 | ✅ read-only audit before deletion | ✅ evidence checklist | N/A |
| AC-009 | ✅ TC-010 | N/A | ✅ TC-010 | ✅ manual wording review | N/A | N/A | ✅ inventory check | N/A |
| AC-010 | ✅ TC-011 | N/A | N/A | N/A | N/A | N/A | ✅ traceability guard | N/A |

### Coverage Gaps

None. Runtime observability is intentionally not added: these configuration
loader outcomes already expose typed errors to callers and `config validate`;
adding metrics/logs would expand the feature without an operational requirement.

## Observability Design

| Behavior | Metric | Log | Trace span | Alert | Test assertion |
|---|---|---|---|---|---|
| Legacy JSON refusal | internal — typed error is caller-visible evidence; no new instrumentation | existing caller error presentation | N/A | N/A | TC-001..TC-003 assert `errors.Is` and guidance |
| Supported canonical source resolution | internal — source tracking is existing evidence | N/A | N/A | N/A | TC-005 asserts loaded source/result |
| Prompt resolution | internal — renderer return/error is direct evidence | N/A | N/A | N/A | TC-006 renders canonical content and rejects retired-only fixture |
| Command retirement | implementation-handoff release evidence | read-only audit report | N/A | release blocker | TC-009 requires attached evidence |
| Current guidance | direct-file content evidence | N/A | N/A | N/A | TC-008, TC-010 |

## Integration Scenarios

| Scenario | Boundary verification | Epic UAT contribution |
|---|---|---|
| Legacy root JSON to config validation | loader's typed error becomes exactly one actionable validation finding, never parsed workflow or duplicate warning | A9 |
| Explicit JSON to install command | rejection remains at loader; explicit `shark admin install-shark-data` writes canonical YAML reference and retains overrides | A2, A8, A9 |
| Canonical defaults to prompt renderer | no legacy JSON permits embedded/canonical resolution; renderer never reads a retired prompt tree | A3, A4, A9, I1 |
| Release evidence to harness cleanup | F04 A2-A4 evidence plus shipped F05 release and one normal-use day are a hard precondition before external deletion | A9 after its prerequisites |

## Cross-feature Contract Tests (I-##)

Not applicable. `spec.md` declares no I-## because no E32 interaction map
exists. No current Shark-status read is performed or recorded, because the
feature has no staged cross-feature edge.

## Cross-epic Integration Tests (X-##)

Not applicable. The global map's X-05 and X-12 producer ownership is E32-F04;
E32-F06 declares neither a producer nor consumer contract. No deferral is
needed.

## Caller-Path Contracts

| TC | Production entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `workflow.LoadMultiLevelWorkflow(configPath)` | filesystem at temp project root | loader, config-byte parser, root JSON existence check | A fallback loader would parse the valid JSON and return a workflow. |
| TC-002 | `workflow.LoadMultiLevelWorkflowFromBytes(configPath, []byte('{"workflow_config":""}'))` with root file on disk | filesystem root-file stat | `loadMultiLevelWorkflowFromBytes`; string-only target helper | Empty field could silently bypass a discovered legacy file. |
| TC-003 | `workflow.ValidateWorkflowFiles(configPath)` | filesystem only | `LoadMultiLevelWorkflow`; validation-finding construction | Validator could emit duplicate/info findings instead of one actionable refusal. |
| TC-004 | `runSharkInstallData(&cobra.Command{}, nil)` from temp project with real config path | embedded bundle filesystem; no repository mock | `ensureWorkflowConfigField`, config manager, install command | A migration could rewrite the wrong target or erase `overrides/`. |
| TC-005 | `workflow.LoadMultiLevelWorkflow(configPath)` and `LoadMultiLevelWorkflowFromBytesWithDefaultWorkflowDir(configPath, data, defaultDir)` | filesystem fixtures | source resolver, YAML loader, inline workflow parser | Removing JSON could also break YAML/default/override precedence. |
| TC-006 | `findTemplateDir()` then `NewOrchestratorRenderer(dir).Render("task/in_qa.md", data)` | filesystem prompt fixtures | `findTemplateDir`, `hasPromptFiles`, renderer | A retired tree could be selected when it is the only tree. |
| TC-007 | repository-file audit (`rg` over tracked `cmd` and `internal`) | direct-file entrypoint | N/A — content-only | A live source reference or shipped retired tree could return unnoticed. |
| TC-008 | `configValidateCmd.Help()` / `shark config validate --help` | Cobra command construction | help text formatter | Help could retain the false JSON-validation promise. |
| TC-009 | release-handoff checklist and direct `~/.claude/commands/<name>.md` file audit | direct-file entrypoint | N/A — external content-only, read-only until gate passes | Deletion could occur with no qualifying release or one-day evidence. |
| TC-010 | direct-file entrypoints for the six named active docs | N/A — content-only | N/A | A current operator guide could still direct users to retired behavior. |
| TC-011 | direct-file entrypoint `feature.md` front matter | N/A — content-only | N/A | Task generation could attach work to E02 instead of E32. |

## Acceptance Test Cases

### TC-001: Root legacy JSON is refused before parsing (AC-001)

**Feature requirement:** REQ-F-001. **Technique:** equivalence partitioning + attack-class enumeration. **ISO:** Functional, Compatibility, Reliability, Security.

**Preconditions/Input:** For each of valid JSON containing a recognizable workflow and malformed JSON, create `<temp>/.sharkworkflow.json`; write `<temp>/.sharkconfig.json` as `{}`.

**Expected:** `LoadMultiLevelWorkflow(configPath)` returns `errors.Is(err, ErrDeprecatedWorkflowConfigJSON) == true`; error includes removal and `shark admin install-shark-data` remediation; no `MultiLevelWorkflow` is returned.

**Negative:** must not parse either JSON variant or return the recognizable workflow.

### TC-002: Empty legacy selection is refused (AC-002)

**Feature requirement:** REQ-F-001. **Technique:** equivalence partitioning. **ISO:** Functional, Compatibility, Reliability, Security.

**Input:** `{ "workflow_config": "" }` and `{ "workflow_config": "   " }`, each with a root legacy file.

**Expected:** the same typed error and guidance as TC-001. **Negative:** empty/whitespace must not enable embedded-default fallthrough while root JSON exists.

### TC-003: Config validation reports one actionable legacy finding (AC-001)

**Feature requirement:** REQ-F-001. **Technique:** contract-surface enumeration. **ISO:** Functional, Usability, Reliability.

**Input:** TC-001's valid root JSON fixture. **Expected:** `ValidateWorkflowFiles` has one error-level finding identifying the configuration and deprecation remediation; it contains no loaded source or duplicate-definition warning. **Negative:** it must not claim JSON was validated.

### TC-004: Explicit JSON is rejected while install migrates and preserves overrides (AC-003)

**Feature requirement:** REQ-F-002. **Technique:** decision table + contract-surface enumeration. **ISO:** Functional, Compatibility, Usability, Reliability, Security, Portability.

**Input:** relative and absolute `workflow_config` JSON targets; configured relative/absolute `shark_data_path`; pre-create `overrides/skills/quality/workflows/review-code.md` with sentinel bytes.

**Expected:** loader returns typed error. `runSharkInstallData` creates `<root>/shark-data/workflow/` or `<absolute-bundle>/workflow/`, writes that directory (with trailing slash) to `workflow_config`, and leaves the override sentinel byte-identical. A YAML index remains accepted.

**Negative:** must not delete overrides, retain a JSON target, or reject YAML merely for a `.sharkworkflow`-like basename.

### TC-005: Supported source-selection matrix retains canonical behavior (AC-004)

**Feature requirement:** REQ-F-003. **Technique:** decision table. **ISO:** Functional, Performance, Compatibility, Reliability, Security, Portability.

**Input/Expected partitions:** (1) no JSON/no explicit source → embedded/default result usable; (2) YAML directory → expected YAML marker; (3) YAML index → expected marker; (4) custom absolute `shark_data_path`/default workflow directory → expected marker; (5) inline block plus configured source → documented precedence; (6) override fixture → replace-only source wins. Clear workflow cache between rows.

**Negative:** a missing legacy JSON must not prevent default resolution; no row may add a directory scan beyond canonical resolution.

### TC-006: Retired template tree cannot influence rendering (AC-005)

**Feature requirement:** REQ-F-004. **Technique:** equivalence partitioning. **ISO:** Functional, Performance, Compatibility, Reliability, Security, Portability.

**Input:** temp working root with only `shark-templates/task/in_qa.tmpl` containing `RETIRED`; then add `shark-data/prompts/task/in_qa.md` containing `CANONICAL`; also cover configured absolute prompt root.

**Expected:** retired-only fixture is not selected and cannot render `RETIRED`; canonical fixture renders `CANONICAL`; absolute canonical root resolves directly.

**Negative:** never add/reinstate a `shark-templates` probe.

### TC-007: Production tree has no retired template source or shipped tree (AC-006)

**Feature requirement:** REQ-F-004. **Technique:** contract-surface enumeration. **ISO:** Functional, Compatibility, Reliability, Security, Maintainability.

**Input:** tracked `cmd/` and `internal/` production files, plus repository top-level shipped directories.

**Expected:** `rg -n 'shark-templates' cmd internal` is empty; no shipped `shark-templates/` exists. **Negative:** fixture-only mentions under `*_test.go` are permitted; historical docs are not mutated or counted.

### TC-008: Config validation help states only supported behavior (AC-007)

**Feature requirement:** REQ-F-006. **Technique:** equivalence partitioning. **ISO:** Functional, Usability, Maintainability.

**Input:** `shark config validate --help` and Cobra long description.

**Expected:** mentions `.sharkconfig.json` and supported YAML/embedded sources; neither output claims it validates `.sharkworkflow.json`. **Negative:** no deprecated JSON-validation sentence remains.

### TC-009: Command deletion obeys the release-window state machine (AC-008)

**Feature requirement:** REQ-F-005. **Technique:** state transition + decision table. **ISO:** Functional, Compatibility, Usability, Reliability, Security, Maintainability.

**Input:** evidence matrix: (a) no qualifying release, (b) release but less than one normal-use day, (c) release plus dated one-day normal-use evidence and fresh A2/A3/A4 prerequisite evidence. Audit the exact eight command paths and `~/.claude/hooks/` plus `scripts/` read-only.

**Expected:** a/b produce an implementation-handoff blocker and preserve all eight files. Only c authorizes their removal, then each exact file is absent and no unrelated command/hook is touched.

**Negative:** do not use a calendar assumption, old assessment, or an unverified tag as gate proof.

### TC-010: Current guidance is corrected while history remains intact (AC-009)

**Feature requirement:** REQ-F-006. **Technique:** content inventory enumeration. **ISO:** Functional, Compatibility, Usability, Maintainability.

**Input:** `CLAUDE.md`, five named CLI/guide docs, and a selected E20/E32 historical record with recorded pre-change hash.

**Expected:** active docs state canonical `shark-data`, embedded defaults, and explicit JSON refusal/remediation; historical file hashes remain unchanged. **Negative:** do not blanket-delete legacy vocabulary from historical plans/changelogs.

### TC-011: Feature metadata identifies E32-F06 (AC-010)

**Feature requirement:** REQ-F-007. **Technique:** exact-value assertion. **ISO:** Functional, Maintainability.

**Input:** YAML front matter in this feature's `feature.md`. **Expected:** `feature_key: E32-F06` and `epic_key: E32`. **Negative:** E02-derived values fail before implementation task creation.

## Test Infrastructure

- Follow temp-project helpers in `internal/config/workflow/workflow_file_loading_test.go` (`writeJSON`, `writeWorkflowYAML`, `ClearWorkflowCache`) for TC-001..005; these are unit/package tests with no database.
- Extend `internal/config/workflow/workflow_validation_dx_test.go` for TC-003, driving `ValidateWorkflowFiles` rather than stubbing the loader.
- Extend `internal/templates/shark_data_renderer_test.go` for TC-006; it already has the canonical-only regression fixture and real renderer.
- Extend `internal/cli/commands/sharkdata_cmd_test.go` for TC-004 and `internal/cli/commands/config_test.go` for TC-008. Command/service tests use mocks or temp filesystem as applicable; do not use a real database.
- TC-007 and TC-010 can be small Go file-content assertions or documented repository audits. No new helper is needed. TC-009 is an implementation-handoff checklist, not an automated external-deletion helper.
- The repository testing rule applies: only repository tests use the real database; this feature needs none.

## Codex Test-Plan Red-Team

**Verdict:** PENDING  
**Issues raised:** PENDING  
**Issues addressed before dev:** PENDING  
**Issues deferred:** PENDING

The required independent review is run against this drafted plan before this
plan is finalized. Its verbatim result and responses are appended below.

## Recommendations

- [x] Ready for implementation planning: every AC has a technique, ISO row, concrete case, and caller/content-path contract.
- [x] Release blocker: do not delete external harness commands until TC-009 evidence is complete.
- [ ] Needs BA refinement.
- [ ] Needs technical refinement.
