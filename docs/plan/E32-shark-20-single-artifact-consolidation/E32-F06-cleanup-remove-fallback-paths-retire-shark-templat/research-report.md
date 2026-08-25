---
research_schema: 2
entity_key: E32-F06
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - workflow_operations
  - documentation
related_work: true
---

# E32-F06 research report: retire legacy resolution paths

## Scope

E32-F06 completes the final compatibility removal for the E32 single-artifact
bundle. It covers the engine's legacy template and JSON-workflow paths, the
eight F05-deprecated harness slash commands, and current operator-facing
documentation. It does not alter the `shark-data/overrides/` replace-only
contract, introduce new workflow behavior, or rewrite immutable historical
planning records.

Terms in this report: **canonical bundle** means `shark-data/` or its embedded
defaults; **legacy template path** means `shark-templates/`; **legacy JSON
workflow** means a `.sharkworkflow.json` target; and **release window** means
the promised intervening release in which F05's deprecated commands remained
available.

The feature file uses the obsolete identity `E02-F06` despite residing in E32
and being dispatched as E32-F06. Treat `E32-F06` as the authoritative entity
identity; correct the stale feature front matter before relying on that file as
an implementation specification.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md`, `assessment.md`, and
  `../epic.md`.
- [x] `affected_implementation_or_contract` — Evidence:
  `internal/templates/orchestrator_renderer.go`,
  `internal/config/aliases.go`, `internal/cli/commands/sharkdata_cmd.go`, and
  `internal/cli/commands/config.go`.
- [x] `related_work` — Evidence: `../research.md`, `../architecture.md`,
  `../uat-plan.md`,
  `../E32-F04-migrate-canonical-content-into-shark-data/feature.md`, and
  `../E32-F05-repoint-harness-deprecate-slash-commands-simplify/{feature.md,research.md,test-plan.md}`.
- [x] `pattern_contract` — Evidence:
  `internal/templates/orchestrator_renderer.go` and
  `internal/sharkdata/embed.go`; current documentation at `CLAUDE.md` and
  `docs/cli-reference/{initialization.md,configuration.md}`.
- [x] `dependency_impact` — Evidence:
  `internal/config/aliases.go`, `internal/config/workflow/`,
  `internal/cli/commands/config.go`,
  `internal/templates/shark_data_renderer_test.go`, and
  `internal/config/workflow/workflow_file_loading_test.go`.

## Capability map

| Capability | Evidence | Decision | E32-F06 responsibility |
|---|---|---|---|
| Canonical `shark-data/prompts` resolution | `internal/templates/orchestrator_renderer.go`; parent `research.md` | REUSE | Preserve the single canonical prompt resolution path already present; do not recreate a fallback. |
| Embedded canonical bundle and override safety | `internal/sharkdata/embed.go`; parent `architecture.md` ADR-3 | REUSE | Keep embedded-default and replace-only override behavior unchanged while removing obsolete callers. |
| Explicit migration from JSON workflow targets | `internal/config/aliases.go`; `internal/cli/commands/sharkdata_cmd.go`; `CLAUDE.md` | EXTEND | Make every legacy JSON entry point reject with the documented migration guidance, then remove obsolete JSON loading branches and tests. |
| F04 embedded corpus parity | `E32-F04.../feature.md`; parent `uat-plan.md` A2-A4 | REUSE (prerequisite) | Require its clean init, validation, and render evidence before destructive cleanup. |
| F05 slash-command deprecation | `E32-F05.../feature.md`, `research.md`, and `test-plan.md` | EXTEND (retirement) | Remove only the eight named commands after the promised release window and preserve a recoverable rollback reference until the gate is met. |
| Feature identity in its local specification | `feature.md` front matter versus dispatch key and directory | CONTRADICTS | Repair the stale E02 identity before task generation; it conflicts with the E32 parent and related-document registration. |
| Continued `.sharkworkflow.json` loading or validation as supported input | `internal/config/aliases.go`; `internal/cli/commands/config.go`; workflow-file tests | CONTRADICTS | F06's acceptance contract requires a clear deprecation error, not a fallback or a CLI description that says the file is validated. |

## Findings

1. The template-resolution part of the cleanup is already complete in the live
   renderer. `findTemplateDir()` resolves an absolute configured bundle, walks
   upward for `shark-data/prompts`, then returns the canonical prompt path; it
   contains no `shark-templates/` pass. The feature assessment is therefore
   stale on this point and must not drive a deletion that no longer exists.

2. Legacy JSON treatment is only partly retired. Operator documentation and
   the `workflow_config` path describe a migration hint, and
   `install-shark-data` rewrites deprecated JSON targets. However,
   `resolveWorkflowDir()` still labels a file as a legacy fallback, the config
   command says it validates `.sharkworkflow.json`, and the workflow package
   has broad JSON-loader test coverage. The remaining work is a coordinated
   removal/refusal change across the resolver, caller behavior, command help,
   and tests—not a string-only cleanup.

3. F04 and F05 are hard dependencies, not just historical context. The
   architecture and UAT plan require successful embedded-corpus coverage,
   fresh-project validation, and a rendered-prompt check before fallback
   retirement. F05's test plan promises that the eight named commands remain
   functional for one release; the F06 assessment records that no qualifying
   release had shipped as of 2026-06-22. Verify the release tag and normal-use
   evidence afresh before deleting commands.

4. Documentation is mostly migrated for active JSON-workflow guidance:
   `CLAUDE.md` and the initialization/configuration references tell operators
   how to move to embedded defaults or `shark admin install-shark-data`.
   A scoped documentation audit must distinguish these current migration
   instructions from historical E20/E32 records, which should remain factual
   history rather than be rewritten.

5. External leakage remains a deployment risk. The parent architecture names
   `~/.claude/hooks/`, scripts, dashboards, and operator muscle memory as
   consumers of the retired path. Search and migration verification must cover
   those live consumers before removal; a repository-only grep cannot prove
   the shared-harness cutover safe.

## Decisions

1. **Use standard rigor and proceed only when its gates are evidenced.** The
   work has bounded code changes, but affects runtime resolution, configuration
   migration, shared-harness commands, and user documentation.

2. **Reuse the completed canonical renderer path.** Do not add another
   resolver or reinstate `shark-templates/` to make tests pass. Replace any
   remaining fallback-oriented tests with canonical-path and explicit-error
   coverage.

3. **Implement explicit refusal, not silent fallback, for legacy JSON.** Every
   former load path must produce the documented migration error or migrate only
   through the explicit install command. Update the config command description
   and remove JSON-loader behavior/tests that assert successful loading.

4. **Treat F04 validation and the F05 release window as release gates.** Before
   deletion, record fresh evidence for `shark init`, `shark admin validate`, a
   rendered prompt, the qualifying release, normal use, and the eight-command
   retirement audit. If any is absent, keep the feature blocked rather than
   weakening the promised compatibility window.

5. **Preserve historical documents and correct only active material.** Update
   current guides, command help, and onboarding. Do not erase historical plan,
   QA, changelog, or assessment references merely to satisfy a broad text grep.

## Sources

- `docs/plan/E32-shark-20-single-artifact-consolidation/epic.md`
- `docs/plan/E32-shark-20-single-artifact-consolidation/research.md`
- `docs/plan/E32-shark-20-single-artifact-consolidation/architecture.md`
- `docs/plan/E32-shark-20-single-artifact-consolidation/uat-plan.md`
- `docs/plan/E32-shark-20-single-artifact-consolidation/E32-F06-cleanup-remove-fallback-paths-retire-shark-templat/{feature.md,assessment.md}`
- `docs/plan/E32-shark-20-single-artifact-consolidation/E32-F04-migrate-canonical-content-into-shark-data/feature.md`
- `docs/plan/E32-shark-20-single-artifact-consolidation/E32-F05-repoint-harness-deprecate-slash-commands-simplify/{feature.md,research.md,test-plan.md}`
- `internal/templates/orchestrator_renderer.go` and `internal/templates/shark_data_renderer_test.go`
- `internal/config/aliases.go`, `internal/config/workflow/`, and `internal/config/workflow/workflow_file_loading_test.go`
- `internal/cli/commands/{config.go,sharkdata_cmd.go}` and `internal/sharkdata/embed.go`
- `CLAUDE.md`, `docs/cli-reference/initialization.md`, and `docs/cli-reference/configuration.md`

RECOMMENDED OUTCOME: standard
