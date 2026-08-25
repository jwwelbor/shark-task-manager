# Test Plan — E32-F06 legacy-resolution cleanup

## AC test matrix

| TC | AC | Technique | Setup and production entrypoint | Expected result / counterfactual |
|---|---|---|---|---|
| TC-001 | AC-001 | State transition | Temp project with root `.sharkworkflow.json`; call `workflow.LoadMultiLevelWorkflow(configPath)` and `config.ValidateWorkflowFiles`. Lowest seam: filesystem only. | `errors.Is(err, workflow.ErrDeprecatedWorkflowConfigJSON)`; a buggy fallback would deserialize the JSON workflow. |
| TC-002 | AC-002 | Equivalence partition | As TC-001 with `workflow_config: ""`. | Same typed error; empty config must not evade root-file refusal. |
| TC-003 | AC-003 | Decision table | Explicit JSON target through loader and `shark admin install-shark-data` with relative/custom/absolute bundle roots. | Runtime loader rejects; install command rewrites to YAML and reports `migrated_from`; a buggy implementation restores runtime JSON loading. |
| TC-004 | AC-004 | State transition | No JSON file; run loader with omitted config, YAML directory, YAML index, and custom `shark_data_path`. | Canonical/embedded defaults or named YAML source load; legacy refusal must not break supported paths. |
| TC-005 | AC-005 | Attack-class enumeration | Renderer fixture containing only `shark-templates/` prompt tree; exercise `findTemplateDir` and real renderer. | Canonical/embedded content wins; a buggy probe renders retired content. |
| TC-006 | AC-006 | Static inspection | `rg -n 'shark-templates' cmd internal` and `test ! -d shark-templates`; allow test fixture labels only. | No executable production reference or shipped tree. |
| TC-007 | AC-007 | Content contract | Invoke `shark config validate --help` through existing Cobra command test seam. | Help names `.sharkconfig.json` plus YAML/embedded sources, not JSON validation. |
| TC-008 | AC-008 | Checklist / operational audit | Read-only audit of eight exact `~/.claude/commands/*.md` paths, hooks and `scripts/`; attach v2.0.0/v2.0.1 release evidence. | Delete only after evidence; absent evidence is a release blocker, not a test workaround. |
| TC-009 | AC-009 | Content partition | Search named active docs and inspect diagrams/examples; exclude archived plans/reviews/changelog. | No active instruction presents retired paths as supported. |
| TC-010 | AC-010 | Traceability | Read `feature.md` front matter before task generation. | Keys are `E32-F06` / `E32`; a stale identity fails planning gate. |

## Caller-path contracts

TC-001 through TC-004 call the public workflow loader using the same config-file
shape production commands use. They may mock no loader/parser helper; temporary
filesystem setup is the lowest allowed seam. TC-005 uses the real renderer and
may not stub directory selection. TC-007 invokes the Cobra command/help layer.
TC-006 and TC-009 are content-only static checks; TC-008 is a release-gated
external operational check, with no simulated host deletion.

## Integration scenarios

1. Legacy root JSON reaches both workflow loading and configuration validation
   with one typed, actionable refusal.
2. Explicit migration through `shark admin install-shark-data` produces the
   supported YAML source which the normal loader subsequently accepts.
3. A project without legacy inputs continues from installed canonical data or
   embedded defaults, preserving E32's single-artifact distribution path.

No I-## or X-## contract is declared by the specification.

## Test infrastructure

- `internal/config/workflow/workflow_file_loading_test.go` supplies temporary
  project, config-file, YAML directory, index, and loader patterns.
- `internal/config/workflow_config_resolve_test.go` covers resolution contracts.
- `internal/config/workflow/workflow_validation_dx_test.go` covers diagnostics.
- `internal/templates/shark_data_renderer_test.go` provides canonical-only
  renderer coverage.
- `internal/cli/commands/config_test.go` and `sharkdata_cmd_test.go` supply
  Cobra/help and install-migration patterns.

Run focused package tests, then `make fmt && make lint && make test`, and
`git diff --check`. UAT must attach the external command audit and release
window evidence before accepting TC-008.
