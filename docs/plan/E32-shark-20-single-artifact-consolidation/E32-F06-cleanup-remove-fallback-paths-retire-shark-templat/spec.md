# E32-F06 specification: retire legacy resolution paths

## Scope and traceability

This specification is incremental to the parent epic. See Epic PRD §2, SC7; §3, Cleanup; §4, A5; and §6, scenario 6. See the parent architecture ADR-3 and ADR-6, §5 F06, §6.2, and §6.3. The authoritative feature identity is `E32-F06`; the stale `E02-F06` front matter in `feature.md` must be corrected when that source document is next edited.

The validated capability map in `research-report.md` establishes that this feature reuses canonical `shark-data/prompts` resolution, the embedded bundle, and replace-only overrides. It extends only the explicit JSON migration/refusal boundary and the F05 command retirement. It does not reimplement rendering, embedded-data installation, or override resolution.

## Requirements

### Functional requirements

| ID | Requirement | Epic trace |
|---|---|---|
| REQ-F-001 | Preserve `shark-data/prompts` and embedded defaults as the only prompt source. A repository directory named `shark-templates` must never affect prompt discovery or rendering. | Epic PRD §2 SC7; §3 Cleanup |
| REQ-F-002 | Reject an explicit JSON `workflow_config` target and a discovered root `.sharkworkflow.json` before any JSON workflow is loaded. The error must state that legacy JSON workflow files are unsupported and direct the operator to remove or empty `workflow_config` and remove or rename the root JSON file for embedded defaults, or run `shark admin install-shark-data` for an editable bundle. | Epic PRD §2 SC7; §3 Cleanup; §6 scenario 6 |
| REQ-F-003 | Keep `shark admin install-shark-data` as the sole migration action that may rewrite a deprecated JSON `workflow_config` to the installed YAML workflow directory. It must retain the configured `shark_data_path` and report the prior JSON value. | Epic PRD §4; architecture ADR-3 and §6.3 |
| REQ-F-004 | Remove the eight F05-deprecated host commands: `run`, `feature`, `epic`, `task`, `prd`, `dispatch`, `develop`, and `release`. Perform this deletion only after evidence of the promised release window and normal use is recorded. | Epic PRD §3 Cleanup; feature.md Dependencies and Risks |
| REQ-F-005 | Update active operator documentation and command help to describe only YAML workflow directories or indexes, embedded defaults, `shark-data/`, and the migration path. Preserve historical plans, changelogs, QA evidence, and archival analysis as historical records. | Epic PRD §3 Cleanup; §4 A5 |

### Non-functional requirements

| Area | Requirement |
|---|---|
| Reliability | The cutover must fail closed with the same actionable deprecation guidance on every legacy JSON entry path; it must not silently fall back to a JSON workflow or a legacy template tree. |
| Security | Resolve configuration paths with the existing project-root and home-expansion checks. Do not relax path-traversal validation while removing the JSON branch. |
| Performance | Keep prompt discovery bounded to the existing configured `shark-data/prompts` walk-up and embedded backstop; do not add filesystem scans for legacy locations. |
| Operations | Retain explicit `install-shark-data` migration and replace-only overrides so operators can recover through a supported, observable command. |

### Acceptance criteria

| ID | Criterion |
|---|---|
| AC-001 | A temporary project containing only `shark-templates/task/in_qa.tmpl` resolves the canonical prompt path and renders embedded or installed canonical content, never that template. |
| AC-002 | A project with `workflow_config` set to any JSON file fails before JSON parsing and emits the migration guidance in REQ-F-002. |
| AC-003 | A project with no `workflow_config` but a root `.sharkworkflow.json` fails with the same migration guidance; a project without either uses embedded defaults or its configured YAML bundle. |
| AC-004 | `shark admin install-shark-data` migrates an explicit JSON target to `<shark_data_path>/workflow`, reports `migrated_from`, and remains idempotent for a YAML directory or YAML index. |
| AC-005 | Production source under `cmd/` and `internal/` has no executable `shark-templates` resolution path. The repository contains no `shark-templates/` directory. |
| AC-006 | The eight named files are absent from `~/.claude/commands/` only after the release-window evidence is attached to the implementation record. |
| AC-007 | `CLAUDE.md`, active CLI references, workflow guides, and the active architectural overview do not present `shark-templates/` or JSON workflows as supported configuration. Historical records remain unchanged. |
| AC-008 | Targeted template, configuration, workflow-loader, command, and documentation-reference tests pass; the full required Go quality gate passes after implementation. |

### Out of scope

- New prompt syntax, rendering behavior, workflow statuses, or data models.
- Changes to `shark-data/overrides/` replace-only semantics, embedded-bundle layout, or the YAML loader's supported directory/index inputs.
- Rewriting historical planning, changelog, QA, or archival-analysis records to erase accurate legacy references.
- Replacing third-party scripts or tools outside the named shared-harness command directory; unresolved external consumers are migration findings, not an invitation to preserve a runtime fallback.

## Architecture

### Component changes

| Path | Change |
|---|---|
| `internal/templates/orchestrator_renderer.go` | Preserve the current single canonical `findTemplateDir()` flow. Do not add a legacy branch; update only comments if they inaccurately imply legacy template support. |
| `internal/templates/shark_data_renderer_test.go` | Retain and extend the canonical-only assertions for `findTemplateDir()` so a populated legacy tree cannot become a renderer input. |
| `internal/config/workflow/parser.go` | Remove implicit root `.sharkworkflow.json` selection/loading. Detect both explicit JSON targets and a legacy root JSON file before loader selection, and return the shared deprecation error. Keep YAML directory and YAML-index resolution unchanged. |
| `internal/config/aliases.go` | Remove the legacy-file fallback signal from `resolveWorkflowDir()` and its callers. Return canonical workflow and overrides locations for absent default disk content so embedded YAML remains the backstop. Keep project-root validation and `shark_data_path` resolution. |
| `internal/config/workflow/workflow_file_loading_test.go` | Delete successful legacy JSON-loading cases and add table-driven explicit-target, root-file, empty-config, and YAML-directory/index acceptance coverage. |
| `internal/config/workflow_config_resolve_test.go` | Replace legacy-fallback expectations with canonical-directory/embedded-backstop expectations and the explicit-refusal boundary. |
| `internal/config/workflow/workflow_validation_dx_test.go` | Remove assertions that validate a legacy JSON workflow as supported input; assert diagnostic migration guidance instead. |
| `internal/cli/commands/config.go` | Change `config validate` help so it does not claim to validate `.sharkworkflow.json`; it must describe supported configuration and surface the loader diagnostic. |
| `internal/cli/commands/config_test.go` | Add command-help and legacy-JSON diagnostic coverage for the revised command contract. |
| `internal/cli/commands/sharkdata_cmd.go` | Preserve the explicit install migration behavior; change only wording if necessary so it is clearly a migration command rather than runtime compatibility. |
| `internal/cli/commands/sharkdata_cmd_test.go` | Preserve JSON-to-YAML migration tests and add or retain assertions for custom and absolute `shark_data_path`, `migrated_from`, and YAML idempotence. |
| `CLAUDE.md` | Keep the current migration guidance, but ensure it describes no implicit JSON loading or legacy template path. |
| `docs/cli-reference/configuration.md`, `docs/cli-reference/initialization.md`, `docs/guides/route-based-workflow.md`, `docs/guides/workflow-profiles.md`, `docs/architectural-overview.md` | Update active documentation and diagrams to use the canonical Shark 2.0 bundle and YAML workflow terminology. |
| `~/.claude/commands/{run,feature,epic,task,prd,dispatch,develop,release}.md` | Delete the eight retired F05 commands after the release-window gate; these are shared-harness operations, not repository source files. |

No schema, database migration, HTTP API, or new persisted data is required.

### Interfaces and error contract

The existing workflow entry points continue to accept a configured YAML workflow directory or YAML index and an omitted or empty `workflow_config` for embedded defaults. An explicit JSON `workflow_config`, or a root `.sharkworkflow.json` that would otherwise be discovered, returns the shared `DeprecatedWorkflowConfigJSONError()` diagnostic before `loadWorkflowFile()` or any JSON parser is called. The diagnostic must retain the migration actions in REQ-F-002. `shark admin install-shark-data` remains the explicit exception: it may convert JSON configuration to the installed YAML directory and must not make JSON a runtime input again.

### Technical decisions

| Decision | Rationale |
|---|---|
| Treat removal as a fail-closed migration boundary. | The parent architecture ADR-6 makes F06 the deliberate hard cutover. A missing bundle uses embedded canonical content; a present legacy JSON file is an operator-actionable configuration problem, not a fallback candidate. |
| Retain the current renderer rather than deleting code that no longer exists. | The research report verifies that `findTemplateDir()` already ignores `shark-templates`. A regression test is the correct F06 deliverable for that completed cleanup. |
| Preserve install-time migration but remove runtime loading. | This follows ADR-3's embedded distribution and replace-only override contract while providing a single recovery path. |
| Scope documentation cleanup to active guidance. | It satisfies the operator contract without corrupting historical evidence, consistent with the research report's Decision 5. |
| Gate host-command deletion on release evidence. | F05 promised one functional release window. The requirement protects users of the shared harness and cannot be inferred from repository tests. |

### Integration with existing code

The renderer continues to use `findTemplateDir()`, `NewOrchestratorRenderer()`, and `newOrchestratorRendererFromEmbed()` in `internal/templates/orchestrator_renderer.go`. Configuration continues through `resolveWorkflowFilePath()`, `hasExplicitDeprecatedJSONWorkflowConfig()`, `IsDeprecatedWorkflowConfigTarget()`, and `DeprecatedWorkflowConfigJSONError()` in `internal/config/workflow/parser.go`, with directory selection through `resolveWorkflowDir()` in `internal/config/aliases.go`. CLI behavior remains thin: `config validate` uses `config.ValidateWorkflowFiles()`, while the explicit migration command continues through the existing Shark-data command helpers.

Follow the existing constructor-free configuration and Cobra-command patterns; do not introduce a service, repository, model, or global singleton for this deletion-only feature. Preserve `pathutil.ExpandHome`, project-root containment, and the embedded-FS fallback already used by the renderer and workflow loader.

### Verification plan

Run the focused Go tests for `internal/templates`, `internal/config`, `internal/config/workflow`, and `internal/cli/commands`; run the repository searches backing AC-005 and AC-007; exercise a fresh temporary project for AC-002 through AC-004; and run `make fmt && make lint && make test` after Go changes. Capture the release tag, elapsed release window, normal-use evidence, and absence of all eight host commands before recording AC-006 as passed.

### Unresolved decisions

There are no material unresolved architecture decisions. The renderer cutover, JSON refusal wording, and explicit-install exception are resolved by the validated research report and parent ADRs. The release-window evidence is an implementation gate, not a design question; if it is absent, implementation must stop without deleting the shared-harness commands.

## Cross-feature interactions

Not applicable. The parent epic has no `E32-interaction-map.md`; therefore no map-owned `I-##` contract exists for E32-F06 to produce or consume.

## Cross-epic integrations

Not applicable. The global map assigns E32's relevant rows to E32-F04 (`X-05` and `X-12`), not to E32-F06. This cleanup feature neither produces, consumes, nor validates a map-owned `X-##` contract.

