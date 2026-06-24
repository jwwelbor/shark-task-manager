# E32 Brownfield Research Report — Shark 2.0 Single-Artifact Consolidation

**Date:** 2026-06-22  
**Researcher:** Researcher Agent  
**Epic:** E32 — Shark 2.0 — Single-Artifact Consolidation

---

## Executive Summary

Most of the **engine work** described in E32's engineering appendix (E1–E9) is already fully implemented in the codebase. The `{{include:}}`/`{{augment:}}` directive system, YAML workflow loader, `shark next` command, `.md` prompt support, `shark-data/` resolution, and `shark init`/`upgrade`/`validate` lifecycle commands are all present and working. The remaining work for E32 is primarily **content migration and harness integration**: populating the embedded FS with the full prompt corpus (F04), repointing the Claude harness (F05), cleaning up legacy fallback paths (F06), adding JSONL dispatch telemetry (F07), and consolidating supplemental skills (F08). F01 (skill extraction) has all tasks in `ready_for_review` or better — the skill extraction corpus work is complete.

---

## 1. Existing Implementations (with File Paths)

### E1 — `{{include:}}` / `{{augment:}}` Directives

**File:** `internal/templates/includes.go`

Fully implemented. Key exports:
- `IncludeResolver` struct with `dataRoot string`, `warnFn func(string, int)`
- `NewIncludeResolver(dataRoot string) *IncludeResolver`
- `Resolve(content string) (string, error)` — preprocesses `{{include:}}` and `{{augment:}}` in any string
- `resolveDepth()` — recursive with `visited map[string]bool` for cycle detection, depth capped at `IncludeDepthCap = 5`
- `resolvePath()` — checks `<dataRoot>/overrides/<path>` first, then `<dataRoot>/<path>`
- Constants: `IncludeDepthCap = 5`, `IncludeSizeWarnBytes = 50 * 1024`
- Regex: `\{\{\s*(include|augment)\s*:\s*([^}\s]+(?:\s+[^}\s]+)*?)\s*\}\}`

Override semantics are correctly implemented: overrides take precedence, never merge.

### E2 — YAML Workflow Loader

**File:** `internal/config/workflow/yaml_loader.go`

Fully implemented. Key exports:
- `LoadMultiLevelWorkflowFromYAML(dataDir string) (*MultiLevelWorkflow, error)`
- `LoadMultiLevelWorkflowFromYAMLDir(workflowDir, overridesDir string)`
- `parseWorkflowYAML()` — routes YAML → JSON → `WorkflowConfig` via existing JSON struct tags (single schema source, no duplication)
- `yamlEntityFiles` maps entity slots to filenames: `epic.yaml`, `feature.yaml`, `task.yaml`, `bug.yaml`, `change.yaml`, `tech-debt.yaml`, `sprint.yaml`
- Override precedence: `<dataDir>/overrides/workflow/<file>.yaml` over `<dataDir>/workflow/<file>.yaml`
- Per-file errors accumulated (not fail-fast) to avoid regression on partial workflow sets

### E3 — `shark next` Command

**File:** `internal/cli/commands/next.go` (658 lines)

Fully implemented. Key structure:
- `NextResponse` struct: `EntityKey, EntityType, Status, Action, AgentType, Provider, Model, Prompt, ResolvedVia, Error`
- `nextCmd` cobra command: `shark next <entity-key>` with `--preview` flag
- `resolveNext()`: recursive cascade resolution; `maxCascadeDepth` guard
- `tryCascade()`: iterates dispatchable children, recurses on first actionable child
- `applyWireAction()`: maps internal YAML verbs to wire vocabulary; handles `advance_and_recurse`
- `attachAgentBody()`: inlines agent persona via `LoadAgentBodyForInline()` using `IncludeResolver`
- `LoadAgentBodyForInline(root, agentType string) (string, bool)`: resolves `agents/<type>.md` with override support
- `normalizeWireAction()`: maps `spawn_agent/check_or_resume/advance_status/pause/archive`
- `nextAdapterCache`: per-invocation cache keyed by entity type

### E4 — `.md` Prompt Support

**File:** `internal/templates/orchestrator_renderer.go`

Fully implemented:
- `NewOrchestratorRenderer()` globs both `.tmpl` and `.md` files in the template dir
- `stripFrontmatter(content string) string` strips `---` YAML frontmatter from `.md` files before template parsing
- `.md` files are loaded as Go templates (same engine, same functions) — no separate path needed

### E5 — `shark-data/` Resolution with `shark-templates/` Fallback

**File:** `internal/templates/orchestrator_renderer.go`

Fully implemented:
- `findTemplateDir()`: Pass 1 checks `shark-data/prompts/` (Shark 2.0 layout); Pass 2 falls back to `shark-templates/` (legacy)
- `detectIncludeRoot()`: Returns parent of `prompts/` dir when Shark 2.0 layout detected — wires the `IncludeResolver` correctly
- `OrchestratorRenderer.IncludeRoot() string`: returns resolved data root for external callers

### E6 / E7 / E8 / E9 — `shark init`, `shark upgrade`, `shark validate`, Embedded FS

**Files:**
- `internal/sharkdata/embed.go` (539 lines) — core logic
- `internal/cli/commands/sharkdata_cmd.go` (348 lines) — CLI adapters

Fully implemented:
- `//go:embed all:default_data` embeds the canonical tree at compile time
- `SharkDataDirName = "shark-data"`, `embedRootDir = "default_data"`
- `Init(projectRoot string) (string, error)`: idempotent; returns `ErrAlreadyInitialized` if `shark-data/` exists; caller is told to run `upgrade` instead
- `Upgrade(projectRoot string, dryRun bool) (*DiffSummary, error)`: refreshes all files except `overrides/`; `DiffSummary` has `Added, Updated, Unchanged, SkippedOverrides` fields; dry-run mode safe
- `Validate(projectRoot string) (*ValidationReport, error)`: walks workflow YAML, prompt includes, skill frontmatter
  - `validateWorkflowYAML()`: checks for expected files (`epic.yaml`, `feature.yaml`, `task.yaml`, `bug.yaml`, `change.yaml`), validates YAML structure
  - `validatePromptIncludes()`: walks `prompts/*.md`, checks each `{{include:}}` path resolves
  - `validateSkillFrontmatter()`: checks `SKILL.md` frontmatter parses
  - `ValidationReport`, `ValidationIssue`, `IssueLevelError/Warning/Info` types
- CLI wrappers: `shark init`, `shark upgrade [--dry-run]`, `shark validate` all wired into cobra tree; `defaultWorkflowConfigDir = "shark-data/workflow/"` set on init

### Observability Subsystem (relevant to F07)

**Files:**
- `internal/observability/` package: `logger.go`, `metrics.go`, `noop.go`, `provider.go`
- `internal/runner/controller.go`: calls `emitStageStart`, `emitStageTransition`, `emitStageDispatch`, `emitStageComplete`, `emitStageError` at critical points
- `internal/cli/observability_global.go`: global accessor

Current exporters in `provider.go`: `"stdout"` and `"otlp"` (gRPC). No `file_jsonl` exporter exists yet — this is the gap F07 fills. The exporter type is a string switch in `buildTracerProvider()` / `buildMeterProvider()` — adding `"file_jsonl"` is a clean extension.

### Agent Dispatcher (relevant to F04/F05)

**Files:**
- `internal/runner/dispatcher.go`: `AgentDispatcher` interface, `DispatchInput`, `DispatchResult`, `DefaultDisallowedTools`
- `internal/runner/controller.go`: `RunOptions`, `RunResult`, `EntityTransitioner` interface

`DefaultDisallowedTools` already blocks `shark status advance*` and `shark task next-status*` — preventing agent self-advancement.

### Embedded Default Data

**Directory:** `internal/sharkdata/default_data/`

Current contents (NOT what's in project-local `shark-data/`):
- `agents/` — 9 agent `.md` files (fully populated)
- `overrides/.gitkeep`
- `prompts/.gitkeep` — **EMPTY** (only a .gitkeep)
- `skills/` — assessment, debugging, implementation, quality, research, specification-writing, test-driven-development, uat (8 skills, no architecture/product-design/brownfield-analysis/frontend-design)
- `workflow/` — bug.yaml, change.yaml, epic.yaml, feature.yaml, task.yaml (5 files)

**Project-local `shark-data/`** (fully populated, NOT yet embedded):
- `agents/` — 9 agents
- `prompts/` — all entity prompts + `_partials/` (16 partial files), fully populated
- `skills/` — 15+ skills including supplemental batch (architecture, brownfield-analysis, frontend-design, product-design, and more)
- `workflow/` — all YAML files including tech-debt, sprint

The **gap between embedded defaults and project-local `shark-data/` is the core content migration work** (F04).

---

## 2. Patterns and Conventions to Follow

### Cobra + Service Layer

CLI commands are thin wrappers: parse → call service → format output. No business logic in commands. New commands (`shark next`, `shark init`, `shark upgrade`, `shark validate`) correctly implement this pattern.

### Constructor Injection

All services use constructor injection. `IncludeResolver` is constructed in `NewOrchestratorRenderer()` and `attachAgentBody()` with `NewIncludeResolver(dataRoot)`. No global singletons for template logic.

### Interface at Point of Use

`EntityTransitioner` in `controller.go`, `AgentDispatcher` in `dispatcher.go` — both defined at point of use, enabling mock injection in tests.

### `//go:embed` Pattern

The `internal/sharkdata/embed.go` uses `//go:embed all:default_data` with the `all:` prefix to include hidden files (`.gitkeep`, future `.gitignore` in `overrides/`). Any new embedded content must follow this pattern — do NOT use `embed.FS` without `all:` or hidden files will be silently skipped.

### Error Handling

Per-file errors accumulated (not fail-fast) in the YAML loader — critical for partial workflow sets. The same pattern should be followed in `validatePromptIncludes()` to avoid aborting on the first broken include.

### Override Semantics (Non-Negotiable)

`overrides/<path>` always takes precedence over `<path>`. `Upgrade()` **never** touches `overrides/`. This is a hard contract that F04 content migration must not violate.

### Template Functions

`orchestratorFuncs()` in `orchestrator_renderer.go` provides: `dict`, `list`, `default` (Sprig-parity), `eq`, `ne`, `isEmpty`, `isSimple/Standard/Complex`. Any new prompt templates must use only these functions or add them to `orchestratorFuncs()`.

---

## 3. Integration Points

| Layer | Component | File | Notes |
|---|---|---|---|
| CLI | `shark next` | `internal/cli/commands/next.go` | E3 complete; add `--preview` to existing flag |
| CLI | `shark init/upgrade/validate` | `internal/cli/commands/sharkdata_cmd.go` | E6-E9 complete; thin wrappers around sharkdata package |
| Templates | `OrchestratorRenderer` | `internal/templates/orchestrator_renderer.go` | E4/E5 complete; resolves shark-data/ → shark-templates/ fallback |
| Templates | `IncludeResolver` | `internal/templates/includes.go` | E1 complete; used by renderer and `next.go` |
| Config | YAML workflow loader | `internal/config/workflow/yaml_loader.go` | E2 complete; per-entity YAML files |
| Embedded FS | Default data | `internal/sharkdata/embed.go` + `default_data/` | E8 complete; prompts/ is currently empty |
| Observability | OTel provider | `internal/observability/provider.go` | Existing stdout/otlp exporters; F07 adds file_jsonl |
| Runner | Dispatch loop | `internal/runner/controller.go` + `dispatcher.go` | Emits stage spans; F07 adds per-call attributes |

---

## 4. Extend vs New Analysis

### F01 — Extract craft from scaffolding (skill extraction)

**Status:** 8/8 tasks in ready_for_review or completed. Content work is done in `shark-data/skills/`. No Go code changes needed.

**Extend:** The `IncludeResolver` (E1) already handles `{{include:}}` directives that the extracted skills use. No new code.

### F02 — Engine (E1-E5)

**Status:** ALL IMPLEMENTED. E1 (`includes.go`), E2 (`yaml_loader.go`), E3 (`next.go`), E4/E5 (`orchestrator_renderer.go`) are production code.

**Extend:** No new code needed. F02 can be considered done at the engine level. The feature's `in_assessment` status should be reviewed — the implementation may pre-date the feature spec.

### F03 — Engine lifecycle (E6-E9)

**Status:** ALL IMPLEMENTED. `sharkdata/embed.go` + `sharkdata_cmd.go` are production code.

**Extend:** No new code needed. F03 `draft` status should be reviewed — this code shipped before the feature was formally planned in E32.

### F04 — Content migration (prompts/, embedded FS)

**Status:** NOT DONE. The embedded `default_data/prompts/` directory contains only `.gitkeep`. Project-local `shark-data/prompts/` is fully populated.

**New work required:**
1. Copy `shark-data/prompts/` content into `internal/sharkdata/default_data/prompts/`
2. Copy remaining skills into `internal/sharkdata/default_data/skills/` (architecture, brownfield-analysis, frontend-design, product-design are missing from embedded tree)
3. Migrate `shark-templates/*.json` workflow files → per-entity YAML in `internal/sharkdata/default_data/workflow/` (already has 5 files; need to verify parity with JSON configs)
4. Convert `.tmpl` files in `shark-templates/` to `.md` prompts in `default_data/prompts/` (or verify they are already superseded by the `.md` files in `shark-data/prompts/`)
5. Replace `LOAD:` directives with `{{include:}}` in any prompts that still use the old syntax
6. Update agent dispatch routing decision (inline vs copy-on-init) per F04 feature spec

**Cannot extend:** The embedded FS is a compile-time snapshot. Content must be physically copied into `default_data/` and the binary rebuilt. No runtime merging is possible.

### F05 — Repoint harness

**Status:** NOT DONE. Skills still exist in `~/.claude/skills/` alongside `shark-data/skills/`.

**New work (outside Go repo):**
- Delete in-scope skills from `~/.claude/skills/` (specification-writing, quality, architecture, research, implementation, test-driven-development, assessment, uat, debugging)
- Delete in-scope agents from `~/.claude/agents/`
- Rewrite `~/.claude/skills/shark/SKILL.md` to tiny dispatch loop
- Add deprecation headers to slash commands (`/run`, `/feature`, `/epic`, `/task`, `/prd`, `/dispatch`, `/develop`, `/release`)

**Extend (in repo):** No new Go code needed. F05 is purely harness-side content changes.

### F06 — Cleanup / remove fallback paths

**Status:** NOT DONE. The fallback in `orchestrator_renderer.go` `findTemplateDir()` Pass 2 (→ `shark-templates/`) is still live. `shark-templates/` directory still exists.

**Extend:**
- Remove the Pass 2 fallback from `findTemplateDir()` in `orchestrator_renderer.go` (small targeted deletion)
- Remove `shark-templates/` directory from the repo
- Remove deprecated slash commands
- Remove `legacy .sharkworkflow.json` reader path (check `internal/config/workflow/` for any JSON-specific loading that doesn't go through the YAML path)

**Risk:** Cannot remove fallback until F04 content migration is complete and the embedded FS has full prompt coverage. F06 must sequence after F04.

### F07 — Dispatch instrumentation / file_jsonl exporter

**Status:** NOT DONE. Observability package exists with `stdout` and `otlp` exporters. `controller.go` already has `emitStage*` calls at all critical points.

**Extend:** Add `"file_jsonl"` case to the exporter switch in `buildTracerProvider()` and `buildMeterProvider()` in `internal/observability/provider.go`. The span emission infrastructure is already wired. This is a pure extension — no architectural changes.

**New work:**
- Implement `fileJSONLSpanExporter` type satisfying `sdktrace.SpanExporter` interface
- Write spans as JSONL to `<project>/shark-data/.stats/events.jsonl`
- Add per-call attributes to `next.go` spans: prompt bytes, unresolved placeholder count, agent type, action
- Detect unresolved `<…>` placeholders in rendered prompts and emit to stderr
- Add `file_jsonl` to `config.ObservabilityConfig.Exporter` validation

**Cannot extend:** No OTel `file_jsonl` exporter exists in the Go OTel SDK — this must be implemented from scratch as a custom exporter implementing `sdktrace.SpanExporter`. The interface is simple: `ExportSpans(ctx, []sdktrace.ReadOnlySpan) error` and `Shutdown(ctx) error`.

### F08 — Supplemental skill library consolidation

**Status:** Partially done (Batch A complete: brownfield-analysis, frontend-design, product-design in `shark-data/skills/`). Batch C audit (`batch-c-audit.md`) found two skills needing cleanup (assessment `assets/` directory, quality `context/` directory, specification-writing has 2 status transition commands that need extraction, uat template has placeholder commands).

**Extend:** All work is in `shark-data/skills/` content files. No Go code changes needed. The purity violations found in Batch C audit are low-severity content fixes.

---

## 5. Technical Risks and Feasibility

### Risk 1: Embedded FS Content Divergence (F04)

**Probability: High | Impact: High**

The embedded `default_data/` is stale relative to `shark-data/`. Any shark binary built from current main will ship empty prompts. This means `shark init` produces a `shark-data/` with no prompt files, and `shark validate` will report `{{include:}}` violations immediately.

**Mitigation:** F04 must be completed before any release that expects `shark init` to produce a working project setup. The binary rebuild is the gate.

### Risk 2: .tmpl → .md Migration Completeness (F04)

**Probability: Medium | Impact: Medium**

Some `.tmpl` files in `shark-templates/` may contain prompt logic that has not yet been ported to `.md` files in `shark-data/prompts/`. If F06 removes the fallback before F04 is complete, those prompts will 404 at render time.

**Mitigation:** Before removing the fallback (F06), audit which entity/status combinations have `.md` prompts in `shark-data/prompts/` and which still fall through to `shark-templates/`. The `findTemplateDir()` Pass 2 currently provides the safety net.

### Risk 3: Agent Dispatch Routing Contract (F04)

**Probability: Medium | Impact: High**

The F04 spec notes a pending decision: agent dispatch routing (inline / copy-on-init / new contract). The `LoadAgentBodyForInline()` function in `next.go` already implements the "inline" model — it reads `agents/<type>.md` and appends it to the prompt at `shark next` time. This is the implemented contract. The F04 decision may need to be resolved against this existing implementation to avoid contradicting it.

**Mitigation:** Confirm that the "inline" model (`attachAgentBody()` in `next.go`) is the canonical dispatch routing strategy before finalizing F04 scope. If a different model is chosen, `next.go` needs updating.

### Risk 4: F07 Custom OTel Exporter Stability

**Probability: Low | Impact: Low**

Custom OTel exporters must implement the `sdktrace.SpanExporter` interface correctly or spans are silently dropped. File I/O in an exporter that's called on the hot path of every `shark next` invocation could add measurable latency.

**Mitigation:** Implement with buffered writes and async flush on `Shutdown()`. Use the existing `sdktrace.NewSimpleSpanProcessor` (synchronous) initially for correctness, then evaluate async if latency is a concern. The exporter is off by default (`Enabled: false` in `ObservabilityConfig`).

### Risk 5: Skill Purity — specification-writing

**Probability: Certain | Impact: Low**

The F08 Batch C audit found specification-writing has 2 active `shark status advance` commands at lines 114-115 of its SKILL.md that are not in `_extracted/`. These must be moved to the `_extracted/SKILL.md` sidecar before F08 is considered complete.

**Mitigation:** Targeted edit to move the two command lines to `_extracted/SKILL.md` and replace with descriptive prose in canonical `SKILL.md`.

---

## 6. Recommended Implementation Approach

### Sequencing

```
F01 (content done) → in_code_review gate
F02 (engine done) → close as complete
F03 (engine done) → close as complete
F04 (content migration) → CRITICAL PATH; triggers embedded FS rebuild
F05 (harness repoint) → after F04 verified working
F06 (cleanup) → after F05; remove fallback only when prompts verified
F07 (telemetry) → parallel with F04/F05; additive, no dependencies
F08 (supplemental skills) → parallel; low-risk content fixes
```

### Priority Actions

1. **Verify F02 and F03 engine completeness** — these features may be in `draft`/`in_assessment` despite the code being fully shipped. Review the feature specs against the implementation and advance their status to reflect reality. This avoids false "work remaining" signals.

2. **Execute F04 content migration** — specifically: copy `shark-data/prompts/` into `internal/sharkdata/default_data/prompts/`, copy missing skills into `default_data/skills/`, rebuild binary, run `shark validate` against a fresh `shark init` to confirm no broken includes.

3. **Extend observability for F07** — add `"file_jsonl"` to `provider.go`'s switch statement and implement `fileJSONLSpanExporter`. Scope is 2–3 new files + small extension to `next.go` for per-call attributes.

4. **F08 skill purity fixes** — move `specification-writing`'s 2 status commands to `_extracted/SKILL.md`. Review `assessment/assets/` and `quality/context/` per the Batch C audit recommendations.

5. **F05 harness repoint** — after F04 is validated. This is outside the Go repo; it's harness file management.

6. **F06 fallback removal** — gated on F05 completion and a passing `shark validate` against embedded defaults.

### Extend vs New Summary

| Feature | Approach | New Code? |
|---|---|---|
| F01 | Content work done; no code | None |
| F02 | Engine already implemented | None |
| F03 | Engine already implemented | None |
| F04 | Copy content into `default_data/`; rebuild | None (content only) |
| F05 | Delete harness files; rewrite SKILL.md | None (harness only) |
| F06 | Delete ~10 lines from `findTemplateDir()`; delete `shark-templates/` | Deletion only |
| F07 | Add `file_jsonl` case to provider switch + 1 custom exporter type | ~200 lines new |
| F08 | Content fixes in `shark-data/skills/` | None (content only) |

The E32 epic is predominantly **content migration and sequencing work**, not new engine development. The engine is built. The risk is ensuring the embedded FS gets populated (F04) before the fallback is removed (F06).

---

## References

- `internal/templates/includes.go` — E1: IncludeResolver
- `internal/config/workflow/yaml_loader.go` — E2: YAML loader
- `internal/cli/commands/next.go` — E3: shark next
- `internal/templates/orchestrator_renderer.go` — E4/E5: .md support, resolution
- `internal/sharkdata/embed.go` — E6/E7/E8/E9: lifecycle commands + embedded FS
- `internal/cli/commands/sharkdata_cmd.go` — CLI wrappers for init/upgrade/validate
- `internal/observability/provider.go` — OTel exporter switch (F07 extension point)
- `internal/runner/controller.go` — stage span emission hooks
- `shark-data/` — project-local working canonical tree (not embedded)
- `internal/sharkdata/default_data/` — embedded canonical tree (prompts/ empty)
- `docs/plan/E32-shark-20-single-artifact-consolidation/E32-F08-.../batch-c-audit.md` — Batch C purity audit results
