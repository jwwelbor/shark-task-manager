# UAT Evidence — E32 F01–F05

**Epic:** E32 — Shark 2.0 — Single-Artifact Consolidation
**Features under review:** F01, F02, F03, F04, F05 (claimed completed via "big commit" path, never run through shark workflow)
**Date:** 2026-05-12
**Evidence collector:** Claude (no judgments; raw facts only)

> Note: Features were not driven through shark workflow. There are **no QA reports, no code reviews, no test artifacts** under feature directories. Evidence is direct artifact inspection plus live invocation of the shipped CLI.

---

## Acceptance criteria from the epic verification table

| Feature | Verification clause(s) |
| --- | --- |
| **F01** | (a) Each craft skill workflow standalone-readable by a stranger with no shark context. (b) Every craft file has `inputs:` (and `outputs:` where meaningful) frontmatter. (c) Sidecars present in `_extracted/` with tagged blocks. (d) `_partials_inventory.md` exists. (e) Branch dispatch expected to fail — F1 not independently mergeable. |
| **F02** | Manual `shark-data/` for one feature; `shark next E01-F02-001 --json \| jq .prompt` returns a rendered prompt **with skill content inlined**; old `shark-templates/` path still works. Plus F02 internal AC: cycle detection fires on a deliberate cycle fixture; override resolution wins over default. |
| **F03** | `cd /tmp/fresh && shark init && shark validate && shark create epic "test" && shark next <new-key>` — full bootstrap path runs clean. |
| **F04** | On a real project, `/run E01-F02-???` produces same agent behavior + same artifacts as pre-migration. Diff rendered prompts before/after — semantically equivalent. |
| **F05** | Move `~/.claude/skills/quality/` aside; `/run` workflow that uses quality still works (everything resolves from `shark-data/skills/`). |

Epic-level success criteria (from epic.md "Success Criteria" + "Impact"):
- Rendered prompts **fully self-contained** with skill content inlined.
- Cross-harness portable (Codex/Copilot/Gemini get same prompt content).
- A fresh project can run shark workflows entirely from `shark-data/` plus only the `shark` skill in the harness.
- `~/.claude/skills/{specification-writing,quality,architecture,research,implementation,test-driven-development,assessment,uat,debugging,orchestration}/` deleted.
- In-scope agents deleted from `~/.claude/agents/`.

---

## F01 — Extract craft from scaffolding (per-workflow-file)

| AC | Evidence | Met? |
| --- | --- | --- |
| (a) Standalone-readable craft skills | 9 skill dirs present at `shark-data/skills/`: architecture, assessment, debugging, implementation, quality, research, specification-writing, test-driven-development, uat | Inspection needed by codex |
| (b) `inputs:` frontmatter on every workflow file | Confirmed across all workflow files in architecture/, debugging/, implementation/, quality/, research/. Subset of skills (assessment, test-driven-development, uat) have only a top-level `SKILL.md` with no `workflows/` subdir | Partial — confirm whether AC applies to skills without `workflows/` |
| (c) `_extracted/` sidecars with tagged blocks (`# fetch`, `# gate`, `# mutate`, `# advance`) | `find shark-data/skills -type d -name "_extracted"` → **zero matches**. No `_extracted/` directories exist anywhere under `shark-data/skills/`. | **No** |
| (d) `_partials_inventory.md` exists | Found at `docs/plan/E32-shark-20-single-artifact-consolidation/E32-F01-extract-craft-from-scaffolding-per-workflow-file/_partials_inventory.md` | Yes |
| (e) Branch dispatch expected to fail | Not applicable post-merge — the feature has been merged | n/a |

Evidence files:
- `shark-data/skills/*/` (9 skill directories)
- `docs/plan/E32-shark-20-single-artifact-consolidation/E32-F01-.../_partials_inventory.md`

---

## F02 — Engine: includes, YAML, .md, `shark next`

### Engine support (E1–E5 in code)

| AC | Evidence | Met? |
| --- | --- | --- |
| `{{include:}}` directive | `internal/templates/includes.go` implements it. `internal/templates/includes_test.go` tests basic include, nested, override-wins, MD frontmatter strip, depth-cap, cycle-detected, sibling-shared-partial, missing-file-with-locations, reject-absolute-path, reject-parent-traversal, size-warning. All tests pass (`go test ./internal/templates/`). | Yes |
| `{{augment:}}` directive | Implemented; current behaviour identical to `{{include:}}` per `TestIncludeResolver_AugmentSameAsIncludeForNow`. Epic noted "TBD" for the difference. | Yes (with caveat — augment ≡ include) |
| YAML workflow loader | `shark-data/workflow/{epic,feature,task,bug,change}.yaml` present and well-formed. **No `tech-debt.yaml`** in `shark-data/workflow/` even though epic lists `tech-debt.yaml`. | Partial (missing tech-debt.yaml) |
| `.md` prompt-file support | Prompts under `shark-data/prompts/` are `.md`; engine renders them. | Yes |
| `shark-data/` resolution | `shark next` resolves from `shark-data/`; engine has `internal/sharkdata/embed.go` with `//go:embed`. Legacy `shark-templates/` fallback still present in codebase. | Yes |
| `shark next` command | Present (`./bin/shark next --help`). For task entity: returns JSON `{action, agent_type, provider, model, prompt}` (provider field is empty string). | Yes |

### F02 acceptance: rendered prompt with **skill content inlined**

**Live test against an actual entity** (`./bin/shark next T-E32-F07-001 --json`):

- Prompt size: 30,584 chars
- Skill references in prompt:
  - `Load skill: \`shark-data/skills/implementation/SKILL.md\``
  - `Load skill: \`shark-data/skills/test-driven-development/SKILL.md\``
- Inline content checks:
  - Contains `# Implementation Agent` heading from the skill? **No**
  - Contains `# Test-Driven Development` heading? **No**
  - Contains TDD workflow content (`TDD INNER LOOP`)? **No**
  - Contains `implement-backend` workflow content? **No**

**Conclusion:** Prompts use **path-reference** to skills (`Load skill: shark-data/skills/...`), not engine-inlined skill content via `{{include:}}`. The agent must still resolve and load these at runtime.

> This contradicts the epic's stated impact: *"Rendered prompts are **fully self-contained** — skill content is inlined at render time"*, and the F02 AC #2 verbatim: *"`shark next E01-F02-001 --json \| jq .prompt` returns a rendered prompt with skill content inlined."*

Source of the design pivot: commit `6a5427f "F4 step 5 — convert .tmpl prompts to .md with path-ref skills"`.

### F02 acceptance: feature-level prompt rendering

Live test:
```
$ ./bin/shark next E32-F02 --json
[shark next] unrendered placeholder <enhancement-feature> left in prompt for E32-F02
```

`shark next` errors out with an unrendered placeholder when invoked on a feature entity. Affected status: `draft` (the current status of every E32 feature).

Evidence files / locations:
- `internal/templates/includes.go`, `internal/templates/includes_test.go`
- `internal/cli/commands/next.go`
- `shark-data/prompts/task/*.md` (zero `{{include:}}` directives found across all entity prompt dirs)
- `shark-data/workflow/{epic,feature,task,bug,change}.yaml`

---

## F03 — Engine: `shark init`, `upgrade`, `validate`, embedded FS

### Bootstrap path verification (verbatim from epic)

**Test sequence:**
```
$ cd /tmp && rm -rf fresh-shark-uat && mkdir fresh-shark-uat && cd fresh-shark-uat
$ shark init
Initialized shark-data/ at /tmp/fresh-shark-uat/shark-data
Set workflow_config: "shark-data/workflow/" in .sharkconfig.json
$ shark validate
shark-data/ at /tmp/fresh-shark-uat/shark-data — 1 issue(s):
  [warning] skills/uat/SKILL.md: frontmatter is not strict YAML: yaml: line 20: did not find expected comment or line break
$ shark create epic "test"
ERROR: Failed to read epic template: open shark-templates/entity/epic.md: no such file or directory
INFO: Make sure you've run 'shark init' to create templates
```

`shark create epic` fails in a freshly-initialized project. The bootstrap path described in the F03 verification clause does not complete. **`shark next <new-key>` cannot be tested because the prior step failed.**

Root cause (source inspection):
- `internal/cli/commands/epic_helpers.go:712-713`:
  ```go
  templateDir := templates.GetTemplateDirName()
  templatePath := filepath.Join(templateDir, "entity", "epic.md")
  ```
  `templates.GetTemplateDirName()` returns `shark-templates`, not `shark-data/prompts` or similar. `shark init` lays down `shark-data/` but `shark create epic` still expects the legacy `shark-templates/entity/epic.md`.

Other observations:
- `shark init`, `shark upgrade`, `shark validate` commands all present in CLI help.
- `shark validate` runs on the newly-init'd directory but emits 1 warning: `skills/uat/SKILL.md: frontmatter is not strict YAML: yaml: line 20: did not find expected comment or line break`. Same warning fires on the source project too.
- `internal/sharkdata/embed.go` carries `//go:embed` with embedded canonical `shark-data/`.
- The init message says `Set workflow_config: "shark-data/workflow/" in .sharkconfig.json` — directory path with trailing slash, not a file path. Need to verify this is intentional vs a bug (the epic envisions `workflow-config.yaml`).

Evidence files / locations:
- `internal/cli/commands/init.go`, `upgrade.go`, `validate.go`, `sharkdata_cmd.go`
- `internal/sharkdata/embed.go`
- `internal/cli/commands/epic_helpers.go:700-720` (the legacy template read)
- Reproduction in `/tmp/fresh-shark-uat/` (transient)

---

## F04 — Migrate canonical content into `shark-data/`

| Item | Evidence | Status |
| --- | --- | --- |
| `shark-data/workflow/{entity}.yaml` per entity | epic, feature, task, bug, change all present | tech-debt entity missing |
| Per-entity prompts under `shark-data/prompts/{entity}/` | task: 13 .md files; feature: 24; epic: 16; bug: 7; change: 8; tech_debt: 6; _partials: 16 | Present |
| Decoupled skills under `shark-data/skills/` | 9 skill dirs (per F01) | Present |
| Agents under `shark-data/agents/` | 9 agent .md files (architect, business-analyst, developer, product-manager, qa, researcher, tech-director, tech-lead, uat-agent) | Present |
| `LOAD-skill` mentions replaced with `{{include:}}` | **0** include directives across all of `shark-data/prompts/` (counted via `grep -rh "{{include:" shark-data/prompts/`). Prompts instead say `Load skill: shark-data/skills/...` as path references. | **Not done** as specified; replaced by a different mechanism |
| `embed.FS` of canonical defaults | Present at `internal/sharkdata/embed.go` | Present |
| Old `.sharkworkflow.json` removed | `shark-templates/.sharkworkflow.json` and `shark-templates/.sharkworkflow-short.json` still present | Still present (back-compat) |
| Diff rendered prompts before/after — semantically equivalent | Not verified — no captured baseline | Unverified |

Notes:
- The F4 plan explicitly said "Replace LOAD-skill mentions with `{{include: skills/<skill>/<workflow>.md}}`". The actual implementation kept path references, so the rendered prompt is not fully assembled — it tells the agent to go load files itself.
- Old `shark-templates/` tree still drives entity creation (see F03 finding) — meaning F4's content migration is functionally partial.

Evidence files / locations:
- `shark-data/workflow/`, `shark-data/prompts/`, `shark-data/skills/`, `shark-data/agents/`
- `internal/sharkdata/embed.go`
- `shark-templates/` (still present)

---

## F05 — Repoint harness, deprecate slash commands, simplify `shark/SKILL.md`

| Item | Evidence | Status |
| --- | --- | --- |
| Delete in-scope skills from `~/.claude/skills/` (specification-writing, quality, architecture, research, implementation, test-driven-development, assessment, uat, debugging) | All 9 removed (checked filesystem) | Done |
| Delete `orchestration` skill | Removed | Done |
| Delete in-scope agents from `~/.claude/agents/` (architect, business-analyst, developer, qa, tech-lead, product-manager, tech-director, researcher, uat-agent) | All 9 removed | Done |
| Rewrite `~/.claude/skills/shark/SKILL.md` to include `shark next` loop | 63-line SKILL.md present; references `workflows/run.md` which "Calls `shark next <key>` in a loop, spawns agents per the engine's instructions, advances on completion" | Done |
| Add deprecation headers to `/run`, `/feature`, `/epic`, `/task`, `/prd`, `/dispatch`, `/develop`, `/release` | `~/.claude/commands/run.md` exists but has **no deprecation header**. `feature.md`, `epic.md`, `task.md`, `prd.md`, `dispatch.md`, `develop.md`, `release.md` files are **absent** from `~/.claude/commands/`. | Missing deprecation header on `run.md`; the other listed slash commands appear to have been removed outright rather than deprecated |
| Verification: move `~/.claude/skills/quality/` aside, `/run` still works | Not executed — `~/.claude/skills/quality/` already absent | Cannot reproduce as specified |

Evidence files / locations:
- `~/.claude/skills/` (listing)
- `~/.claude/agents/` (listing)
- `~/.claude/skills/shark/SKILL.md`
- `~/.claude/commands/run.md`

---

## Cross-feature observations

1. **No `tasks/` subdirectory under F01, F02, F03, F04, F05.** Only F07 has tasks. Features were merged as "big commits" without going through shark's workflow. There are no QA reports, code reviews, or test artifacts attached to these features.
2. **Epic and all features still show `status: draft`** in the shark database; none were advanced. Progress is 0% on all.
3. **`provider` field in `shark next` JSON is empty string** for a task entity (epic schema specified `provider` as a populated field, e.g. `"claude-code"`).
4. **Single live validate warning** persists in both source and freshly-init'd projects: `skills/uat/SKILL.md` frontmatter parses as not-strict YAML at line 20. Engine still strips frontmatter at render time, so it's flagged "informational only".
5. **Embedded shark-data/ versions of the workflow YAML** — verified loadable at init time; round-trip equivalence to old `.sharkworkflow.json` not independently verified in this evidence collection.

---

## File paths for codex cross-check

| Area | Paths |
| --- | --- |
| Epic & feature PRDs | `docs/plan/E32-shark-20-single-artifact-consolidation/epic.md`<br>`docs/plan/E32-shark-20-single-artifact-consolidation/E32-F01-.../feature.md` … F02, F03, F04, F05, F06, F07 |
| Engine — include resolver | `internal/templates/includes.go`, `internal/templates/includes_test.go` |
| Engine — agent body / placeholders | `internal/templates/agent_body.go`, `internal/templates/agent_body_test.go` |
| Engine — embedded FS | `internal/sharkdata/embed.go`, `internal/sharkdata/embed_test.go` |
| CLI commands | `internal/cli/commands/{next,init,upgrade,validate,sharkdata_cmd,epic_helpers,create}.go` |
| Shipped canonical content | `shark-data/{workflow,prompts,skills,agents,overrides}/` |
| Legacy templates (still present) | `shark-templates/` |
| Harness state | `~/.claude/skills/shark/SKILL.md`, `~/.claude/skills/`, `~/.claude/agents/`, `~/.claude/commands/` |
| Bootstrap repro (transient) | `/tmp/fresh-shark-uat/` |

---

*This file is evidence only — no assessment. Codex (Phase 3) is the sole assessor.*
