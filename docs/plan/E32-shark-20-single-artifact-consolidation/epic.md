---
epic_key: E32
title: Shark 2.0 — Single-Artifact Consolidation
description: Consolidate the shark workflow stack — Go CLI engine, ~10 shark-coupled skills, and 15 agent definitions — into a single drop-in artifact installed via 'shark init'. Decouple craft (skills) from workflow scaffolding (prompts) using a structural test rather than lexical search. Engine assembles fully-rendered prompts via 'shark next', removing the need for harness-side dispatch logic. Eight features F1–F8.
size: XXL
---

# Shark 2.0 — Single-Artifact Consolidation

**Epic Key**: E32

> This PRD is the **single source of business context** for E32. Features under
> this epic REFERENCE this document for problem, goals, scope, constraints,
> stakeholder impact, and acceptance criteria — they must not restate them.
> Engineering detail (target architecture, engine changes, layer rules,
> per-feature verification) lives in the **Engineering Reference Appendix** at the
> bottom; that appendix is a design aid, not a substitute for any of the six
> business sections above it.

---

## 1. Problem Statement and Business Justification

The shark workflow stack today ships across **three uncoordinated locations**, and
no single command installs it into a project:

- `~/projects/shark-task-manager/` — the Go CLI engine plus a ~1750-line
  `shark-templates/.sharkworkflow.json` and per-status `*.tmpl` instruction files.
- `~/.claude/skills/` — 29 skills, ~10 of which are referenced by shark prompts
  and embed shark's workflow contract directly in their content (status names,
  `shark --field` storage calls, gate enforcement).
- `~/.claude/commands/` + `~/.claude/agents/` — slash commands and 15 agent
  definitions, all tightly coupled to shark.

This produces three concrete, recurring failures:

1. **Structural coupling, not merely lexical.** Shark-referenced skills do not
   just *mention* shark — they embed the workflow contract: fetching entity
   context, storing notes via `shark --field`, gating transitions, advancing
   status. Stripping shark vocabulary from a skill leaves the contract intact, so
   the skill is still not portable. An audit of `specification-writing` alone
   found 401 shark-vocabulary hits across 4,078 lines, with state mutations woven
   into the craft.
2. **Runtime discovery, not inlined content.** Prompt templates say
   `LOAD: quality skill` as plain text, and the executing agent must *find* the
   skill at runtime. This works under Claude Code (which ships `.claude/skills/`)
   but breaks under Codex, Copilot, and Gemini because the rendered prompt is not
   self-contained.
3. **No single deployable.** A new project today requires the Go binary **plus** a
   synced copy of `~/.claude/skills/` **plus** a copy of `~/.claude/agents/`
   **plus** a `shark-templates/` directory. There is no `shark init` that lays the
   full stack down in one step.

**Business justification.** This is foundational refactoring that unblocks
multi-harness support (Codex/Copilot/Gemini), future-proofs the craft library
against host-system churn, and pays down structural debt that compounds every
time a skill accumulates workflow-contract logic. Without it, **every** new
harness integration requires re-deriving shark's runtime assumptions inside each
skill, and **every** new project pays a four-step, error-prone manual install.
Business value rating: **High** — it is a prerequisite for the portability and
distribution work that follows it.

---

## 2. Goals and Success Criteria

### Goals

- **G1 — One drop-in artifact.** A user installs the shark binary, runs
  `shark init`, and the project receives a `shark-data/` directory containing
  everything needed to run shark workflows (skills, agents, prompts, workflow
  definitions).
- **G2 — Structural decoupling of craft from scaffolding.** Reusable
  methodology (skills) is separated from workflow scaffolding (prompts) using the
  structural decision test, with an `inputs:`/`outputs:` frontmatter contract
  between them.
- **G3 — Engine-driven dispatch.** `shark next <entity>` emits a fully-rendered,
  self-contained prompt plus dispatch metadata, so the harness loop reduces to a
  trivial "call `shark next`, spawn agent, call `shark advance`" cycle.
- **G4 — Single source of truth.** `shark-data/` replaces the three-location
  stack; in-scope skills and agents are deleted from `~/.claude/` after
  migration; local edits survive upgrades via `shark-data/overrides/`.

### Success Criteria (measurable, testable)

| # | Criterion | How it is measured |
|---|-----------|--------------------|
| SC1 | A fresh project runs shark workflows from `shark-data/` plus only the `shark/SKILL.md` trigger in the harness. | In an empty temp dir: `shark init && shark validate && shark create epic "test" && shark next <new-key> --json` completes with exit code 0 and a rendered prompt. |
| SC2 | Rendered prompts are semantically equivalent before and after migration. | For a representative entity, diff the `shark next … --json \| jq .prompt` output against the pre-migration rendered prompt; differences are limited to inlining (skill content present) with no change in instructions, gates, or referenced artifacts. |
| SC3 | `shark next` returns self-contained prompts with skill content inlined. | `shark next <entity> --json \| jq .prompt` contains the body of every skill the prompt references — no unresolved `LOAD:`/`{{include:}}` directives remain. |
| SC4 | In-scope skills and agents are removed from the harness. | `~/.claude/skills/{specification-writing,quality,architecture,research,implementation,test-driven-development,assessment,uat,debugging,orchestration}/` are deleted; the nine in-scope agents no longer exist under `~/.claude/agents/`. |
| SC5 | `shark validate` catches structural defects. | `shark validate` fails when a skill references a shark status name, when any `{{include:}}` path does not resolve, or when a workflow YAML names a missing `agent_type`/`prompt`; passes on the canonical tree. |
| SC6 | Override safety. | An edit placed in `shark-data/overrides/<path>` wins over the canonical default at render time and survives `shark upgrade` unchanged. |
| SC7 | Legacy paths removed in the final phase. | `grep -r "shark-templates" cmd/ internal/` returns nothing; loading a legacy `.sharkworkflow.json` produces a deprecation error. |
| SC8 | No canonical-tree contradictions. | No `_extracted/` scaffolding sidecars remain under `shark-data/skills/` as shipped craft (they are consumed into prompts/workflow or explicitly marked non-shipped before the epic closes). |

---

## 3. Scope

### In Scope

- **Skill decoupling and migration** for the in-scope shark-coupled skills:
  `specification-writing`, `architecture`, `research` (absorbing the stale
  `discovery` references), `quality`, `implementation`,
  `test-driven-development`, `debugging`, `assessment`, `uat`. Each is split into
  craft (skill) and scaffolding (prompt), with an `inputs:`/`outputs:` contract,
  and moved to `shark-data/skills/`.
- **Removal of the `orchestration` skill**, replaced by the engine-emitted
  `shark next` command.
- **Agent migration** to `shark-data/agents/` for the nine shark-coupled agents:
  architect, business-analyst, developer, qa, tech-lead, product-manager,
  tech-director, researcher, uat-agent.
- **Engine changes** in `~/projects/shark-task-manager`: `{{include:}}` /
  `{{augment:}}` template directives with override resolution, a YAML workflow
  loader, `.md` prompt-file support, the `shark next` command, `shark-data/`
  resolution, `shark init` / `shark upgrade` / `shark validate`, and an embedded
  default `shark-data/` (`//go:embed`).
- **Stale-reference cleanup** of `discovery` (collapse into `research`) and
  `build` (drop or replace) in the new tree.
- **Harness cutover**: repoint the harness to resolve from `shark-data/`,
  reduce `shark/SKILL.md` to the tiny trigger, add deprecation notes to slash
  commands.
- **Cleanup**: drop the `shark-templates/` fallback and retire the legacy
  `.sharkworkflow.json`.
- **Supplemental skill-library consolidation track (F08)** for broader reusable
  skills migrated from the jaunty-panda plan — these ship in `shark-data/skills/`
  but are **not blockers** for F01–F06 unless a prompt or workflow explicitly
  depends on them.
- **Dispatch instrumentation (F07)** — a JSONL dispatch-event exporter for
  `shark next` so dispatch decisions are observable.

### Out of Scope

- **Skills that stay untouched** in `~/.claude/skills/`: brainstorming,
  brownfield-analysis, claude-md-improver, code-simplifier, collaboration,
  cx-designer, feature-design, frontend-design, graphify, product-design,
  refactoring, resolve-pr-comments, skill-creator, socratic-method, triage,
  vue-development, xlsx. (Where Batch-A copies already exist under
  `shark-data/skills/`, that is supplemental-track work, not core E32 scope.)
- **Per-harness prompt variants** (e.g. `prompts/task/ready_for_development.codex.md`),
  automatic harness detection, and per-provider prompt/token optimization — all
  deferred to a follow-up epic.
- **Override drift reconciliation / diff tooling**, per-project version pinning
  of embedded defaults, a `shark dev` disk-loading mode, and a golden-output diff
  suite — captured as follow-up backlog, not built here.
- **Rewriting harness-level helper agents** (Explore, Plan, general-purpose) —
  they remain in `~/.claude/agents/`.
- **Cross-repo shared CI** between `~/.claude/` and shark-task-manager — noted as
  a recommendation, not delivered in this epic.

### Scope Boundary Rules

- Backwards compatibility is required through F04; **F05 is the first phase
  permitted to break the legacy paths**, and it does so deliberately.
- `shark-data/overrides/<path>` **fully replaces** the corresponding default —
  it never merges. This is a hard rule to avoid silent drift.

---

## 4. Constraints and Assumptions

### Constraints

- **C1 — Daily work depends on shark.** F01–F04 must each remain
  backwards-compatible with the existing three-location setup; only F05 breaks
  the old paths.
- **C2 — Two repos, no shared CI.** Skill/agent content lives under `~/.claude/`;
  engine changes live in `~/projects/shark-task-manager`; F04 spans both. Work
  must be sequenced so neither repo is left in a broken intermediate state.
- **C3 — Embedded-FS distribution.** Canonical defaults ship inside the binary
  via `//go:embed`; a shark release pushes new defaults to every project on
  `shark upgrade`. There is no per-project version pin in this epic, so the
  upgrade story must preserve `overrides/`.
- **C4 — Render-size and cycle safety.** `{{include:}}` resolution must guard
  against include cycles (cycle detection), runaway nesting (proposed depth cap
  of 5), and oversized prompts (render-size warning above a threshold, proposed
  50KB).
- **C5 — Override semantics are replace-only.** Documented and enforced; a merge
  surprise would cause silent behavioral drift.
- **C6 — Claude Code's `Agent` tool reads `~/.claude/agents/`.** Moving agents to
  `shark-data/agents/` required an explicit dispatch-routing decision. **Resolved:
  copy-on-init.** `shark admin init` syncs `shark-data/agents/` → `.claude/agents/`,
  so the harness keeps reading from its expected location and no new spawn-time
  contract is needed. See *Agent dispatch routing* below.

### Assumptions

- **A1** — The structural decision test (*"if you removed this line, would the
  activity still be that activity?"*) cleanly separates craft-checks from
  gate-checks for the in-scope skills. The jaunty-panda three-layer model
  (workflow / prompts / skills) is adopted as the governing architecture rule.
- **A2** — Existing engine machinery is reusable rather than replaced:
  `{{template "_partial" .}}` partials, the `feature_short`/`task_short`/`epic_short`
  profile concept, the `RunController` + `AgentDispatcher` interface, and
  `shark sync` for filesystem↔DB sync.
- **A3** — `inputs:`/`outputs:` frontmatter is a sufficient contract between a
  prompt and a skill; prompts promise to supply the inputs, skills promise to
  operate without further host assumptions.
- **A4** — A one-release deprecation window (warnings before deletion) is
  acceptable for the harness cutover (F05) and slash-command removal.
- **A5** — External scripts/hooks that hardcode `shark-templates/` (likely under
  `~/.claude/hooks/` and `scripts/`) can be found and updated before F04.

---

## 5. Stakeholder Impact

| Stakeholder | Impact |
|-------------|--------|
| **Solo developer (primary user)** running shark under Claude Code | New-project bootstrap drops from ~4 disjoint installs to a single `shark init`. Inner-loop skill iteration gets **slower** post-consolidation (edit in shark repo → rebuild → `shark upgrade` → test instead of editing `~/.claude/skills/` live) — an accepted trade-off, with a `shark dev` disk-load mode noted as future mitigation. Local customizations move to `shark-data/overrides/`. |
| **Future operators** running shark under Codex / Copilot / Gemini | Become viable for the first time: rendered prompts are self-contained, so shark workflows run identically across harnesses. This is the principal forward-looking beneficiary. |
| **Craft / skill authors** | Skills become standalone-readable — a stranger with no shark context can execute them. Craft evolves independently of workflow scaffolding, reducing coupling-induced regressions. |
| **Harness maintainers** (Claude Code config owners) | `~/.claude/skills/` and `~/.claude/agents/` shrink to harness-level helpers plus the tiny `shark/SKILL.md` trigger. Slash commands carry deprecation notes for one release. F05 is a **global hard cutover** for the shared harness — must be communicated and timed. |
| **Engine maintainers** (shark-task-manager contributors) | Gain new responsibilities: template include resolution, YAML workflow loading, `shark next`/`init`/`upgrade`/`validate`, and embedded-FS distribution. `shark validate` becomes the structural guardrail; a golden-output diff suite is recommended follow-up. |
| **Authors of downstream consumers** (hooks, scripts, dashboards, CLAUDE.md, muscle memory) | Status vocabulary and the `shark-templates/` path name leak into places shark cannot see; these must be audited and updated, and some leakage persists as known follow-up risk. |

---

## 6. High-Level Acceptance Criteria (UAT Scenarios)

These are the end-to-end scenarios that, when all pass, signal E32 is
acceptable. Each maps to one or more success criteria in Section 2.

- **UAT-1 — Fresh-project bootstrap (SC1, SC5).**
  *Given* an empty directory with the shark binary on PATH, *when* the operator
  runs `shark init`, then `shark validate`, then `shark create epic "test"`, then
  `shark next <new-epic-key> --json`, *then* every command exits 0 and the final
  command returns a rendered prompt with no unresolved include directives.

- **UAT-2 — Self-contained rendered prompt (SC3).**
  *Given* a project initialized with `shark-data/`, *when* the operator runs
  `shark next <entity> --json | jq .prompt`, *then* the output contains the full
  body of every skill the prompt references (skills are inlined, not named).

- **UAT-3 — Behavioral parity with the legacy stack (SC2).**
  *Given* a real project entity, *when* it is run through `/run` post-migration,
  *then* the agent produces the same artifacts and exhibits the same behavior as
  the pre-migration run, and a diff of the rendered prompts shows only inlining
  differences — no changed instructions, gates, or referenced artifacts.

- **UAT-4 — Override survives upgrade (SC6).**
  *Given* a customization placed at `shark-data/overrides/skills/quality/workflows/review-code.md`,
  *when* the operator runs `shark upgrade`, *then* the override file is unchanged
  and continues to win over the canonical default at render time.

- **UAT-5 — Harness resolves from `shark-data/` only (SC4).**
  *Given* the in-scope harness skill directory (e.g. `~/.claude/skills/quality/`)
  is moved aside, *when* a `/run` workflow that uses that skill executes, *then*
  it still works because the skill resolves from `shark-data/skills/`.

- **UAT-6 — Validation rejects structural defects (SC5).**
  *Given* a deliberately broken canonical tree (a skill that references a shark
  status name, a dangling `{{include:}}` path, or a workflow YAML naming a
  missing agent), *when* the operator runs `shark validate`, *then* the command
  fails and names the offending file and defect.

- **UAT-7 — Legacy paths retired (SC7).**
  *Given* the cleanup phase is complete, *when* the operator runs
  `grep -r "shark-templates" cmd/ internal/` and attempts to load a legacy
  `.sharkworkflow.json`, *then* the grep returns nothing and the legacy load is
  refused with a deprecation error.

- **UAT-8 — Canonical tree is clean (SC8).**
  *Given* the epic is ready to close, *when* the operator inspects
  `shark-data/skills/`, *then* no `_extracted/` scaffolding sidecars remain as
  shipped craft — each has been consumed into prompts/workflow or explicitly
  marked non-shipped.

---

---

# Engineering Reference Appendix

> The material below is **engineering design reference**, not business context.
> It exists so feature and architecture work can point at concrete structures.
> It does not override or restate Sections 1–6, which remain the authoritative
> business source of truth.

## Architecture rule: skills vs prompts

> **Skill** = the activity. How to do TDD, write a PRD, run a QA pass.
> **Prompt** = the workflow. Read inputs, prepare files, gate, advance.

Decision test for any line of existing skill content:

> *If you removed this, would the activity still be that activity?*

| Example line | Where it goes |
| ------------ | ------------- |
| "In TDD, verify the test fails before writing the implementation" | **Skill** — intrinsic to TDD |
| "When debugging, isolate variables before forming a hypothesis" | **Skill** — craft |
| "Walk the spec, identify the golden path and edge cases" | **Skill** — craft |
| "Run `shark get $ID` to fetch the spec path" | **Prompt** — scaffolding |
| "Verify codex red-team has run before advancing to UAT" | **Prompt** — gate |
| "Verify all design refs from PRD appear in task spec" | **Prompt** — gate (regardless of how craft-flavored it sounds) |
| "Set status `ready_for_qa_review`" | **Prompt** — state mutation |

The test cleanly separates **craft-checks** (intrinsic to the activity) from
**gate-checks** (workflow demands). Craft stays in the skill; gates move to the
prompt. A skill can be 100% shark-vocab-free and still embed the host contract
(assume an entity ID, fetch parent context, write notes back, advance state) —
the craft/scaffolding split is the actual decoupling, not lexical purity.

**Inputs contract** — every craft skill workflow declares its inputs in
frontmatter:

```markdown
---
inputs:
  - spec_path: absolute path to the feature spec markdown
  - scope_summary: 1-paragraph description of what's being QA'd
outputs:
  - defect_report: structured markdown
---
```

## Target architecture

```
<harness>/.claude/skills/shark/SKILL.md             # tiny harness-side trigger, ~50 lines

<project>/.sharkconfig.json                          # data dir path, profile selector, db path, harness type
<project>/shark-data/
  workflow-config.yaml                               # which workflow yaml per entity (default vs short profile)
  workflow/
    epic.yaml feature.yaml task.yaml bug.yaml change.yaml tech-debt.yaml
  prompts/
    epic/{status}.md feature/{status}.md task/{status}.md bug/{status}.md change/{status}.md
    _partials/                                       # _read_section.md, _exit_gate.md, _tdd_process.md, etc.
  skills/
    specification-writing/ quality/ architecture/ research/ implementation/
    test-driven-development/ assessment/ uat/ debugging/
  agents/
    architect.md business-analyst.md developer.md qa.md tech-lead.md
    product-manager.md tech-director.md researcher.md uat-agent.md
  overrides/                                         # local-only; shark upgrades never touch
    {skills,agents,prompts,workflow}/                # mirror tree; entries here win over defaults
```

### Layer rules

- **Workflow YAML** owns step order, routing keys, agent assignment, prompt-file
  pointer, and skip_for. Status name = step name = entity status.
- **Prompt files** own *workflow scaffolding*: read inputs, prepare files, run
  skills, gate outputs, advance state. Coupled to shark by design. The contract
  between prompt and skill is the skill's `inputs:` frontmatter.
- **Skills** own *craft*: how to do the activity, with no shark context.
- **Engine** owns state, template rendering, override resolution, and prompt
  assembly.

### Simplified dispatch model

**Shark 2.0** (engine-driven dispatch):
```
harness → /run E01-F02-001
       → loops:
           response = shark next E01-F02-001 --json
           # { prompt: "<fully assembled, skills inlined>",
           #   agent_type, provider, model, action: "spawn_agent" | "pause" | "archive" }
           if action == "spawn_agent": spawn agent with response.prompt → shark advance after completion
           if action == "pause" / "archive": stop and report
```

## Engine changes (in `~/projects/shark-task-manager`)

| # | Change | Where | Effort |
| - | ------ | ----- | ------ |
| E1 | `{{include: <path>}}` and `{{augment: <path>}}` directives; cycle detection; depth cap; size warning; override resolution. | `internal/templates/` | M |
| E2 | YAML workflow loader (`shark-data/workflow/{entity}.yaml`) into the existing in-memory model. | `internal/config/` | M |
| E3 | `shark next <entity>` returning JSON `{ prompt, agent_type, provider, model, action }`. | `cmd/shark/` + `internal/runner/` | M |
| E4 | `.md` prompt-file support; strip optional YAML frontmatter; render body as Go template. | `internal/templates/` | S |
| E5 | `shark-data/` resolution (configurable via `.sharkconfig.json`); `shark-templates/` fallback for one release. | `internal/templates/` resolver | S |
| E6 | `shark init` copying embedded default `shark-data/` into the project; idempotent. | `cmd/shark/` + `embed.FS` | S |
| E7 | `shark upgrade` updating everything except `shark-data/overrides/`; diff/dry-run flag. | `cmd/shark/` | S |
| E8 | Embed default `shark-data/` via `//go:embed`. | `cmd/shark/` | S |
| E9 | `shark validate`: skills free of shark/status names; all `{{include:}}` paths resolve; all `agent_type`/`prompt` refs exist; no template parse errors. | `cmd/shark/` | M |

## Phase / feature breakdown

| Feature | Title | Order | Depends on |
| ------- | ----- | ----- | ---------- |
| F01 | Extract craft from scaffolding (per-workflow-file) | 1 | — |
| F02 | Engine — includes, YAML workflows, .md prompts, shark next | 2 | — |
| F03 | Engine — shark init, upgrade, validate, embedded FS | 3 | F02 |
| F04 | Migrate canonical content into `shark-data/` | 4 | F01, F03 |
| F05 | Repoint harness, deprecate, simplify `shark/SKILL.md` | 5 | F04 |
| F06 | Cleanup — drop fallbacks, retire `shark-templates/` | 6 | F05 |
| F07 | Dispatch instrumentation — file JSONL exporter for `shark next` | — | F03 |
| F08 | Supplemental skill-library consolidation (jaunty-panda) | — | — (non-blocking) |

F01 and F02 can run in parallel since both gate F04 via different paths. F07 and
F08 are independent supplemental tracks (instrumentation and broader skill
library) that do not block the F01→F06 critical path.

## Per-feature verification

| Feature | How to verify |
| ------- | -------------- |
| F01 | Each craft workflow standalone-readable; `inputs:`/`outputs:` frontmatter present; scaffolding sidecars in `_extracted/` with tagged blocks; `_partials_inventory.md` exists. Lands with F04 (not independently mergeable). |
| F02 | `shark next <entity> --json \| jq .prompt` returns rendered prompt with skills inlined; old `shark-templates/` path still works. Verify against the rebuilt local binary. |
| F03 | `cd /tmp/fresh && shark init && shark validate && shark create epic "test" && shark next <new-key>` runs clean. |
| F04 | `/run <entity>` produces same agent behavior + artifacts as pre-migration; diff rendered prompts before/after. Workflow coverage includes every shipped prompt entity. |
| F05 | Move `~/.claude/skills/quality/` aside; `/run` that uses quality still works from `shark-data/skills/`. |
| F06 | `grep -r "shark-templates" cmd/ internal/` returns nothing; legacy `.sharkworkflow.json` refused with deprecation error. |
| F07 | Dispatch events for `shark next` appear in the JSONL export with the expected fields. |
| F08 | Supplemental skills present under `shark-data/skills/` and pass the purity gate; no `_extracted/` sidecars remain as shipped craft. |

## Risks + watch-outs

- **`{{include:}}` blowup** — cycle detection, depth cap (5), render-size warning (50KB).
- **Override merge semantics** — overrides fully replace, never merge.
- **Stale workflow refs** — `discovery`, `build` fixed before they enter the new tree.
- **`shark-templates/` rename** — audit external scripts/hooks before F04.
- **Backwards compatibility** — F01–F04 compatible; F05 is the deliberate break.
- **Embedded-FS update flow** — `shark upgrade` must keep overrides; document the merge story.

## Agent dispatch routing

F04 moves agents to `shark-data/agents/`, but Claude Code's `Agent` tool reads
`~/.claude/agents/`. Three options were considered:

- **Inline** — embed the agent definition into the rendered prompt. Rejected: loses
  sub-agent isolation benefits.
- **Copy on init** — `shark admin init` syncs `shark-data/agents/` → `.claude/agents/`.
  **← Chosen.**
- **New harness↔shark agent contract** — the harness asks shark for the agent
  definition before each spawn. Rejected: cleanest in theory, but `shark/SKILL.md`
  would no longer be "5 lines" and orchestration regrows in disguise.

**Decision: Copy on init.** `shark admin init` already syncs `shark-templates/` →
the project directory; extending it to copy `shark-data/agents/` → `.claude/agents/`
follows the exact same template-sync pattern, keeps the harness reading from its
expected location, and avoids the complexity of a new spawn-time contract. The
duplication this introduces is a managed, init-time copy (the same trade-off already
accepted for skills and prompts), not a hand-maintained fork.

## Resolved decisions

| # | Decision | Resolution |
| - | -------- | ---------- |
| 1 | Engine work in or out? | **In.** |
| 2 | Fate of in-scope `~/.claude/skills/`? | **Delete after migration.** |
| 3 | The `orchestration` skill? | **Removed** (replaced by `shark next`). |
| 4 | Prompt-file format? | **`.md`** with optional stripped frontmatter. |
| 5 | Agents to `shark-data/agents/`? | **Yes.** |
| 6 | Slash commands? | **Deprecation notes; remove in cleanup.** |
| 7 | Decoupling rule? | **Structural,** not lexical. |
| 8 | Agent dispatch routing (C6)? | **Copy-on-init** — `shark admin init` syncs `shark-data/agents/` → `.claude/agents/`. |

## Follow-up backlog (NOT solved by F01–F08)

- Override drift = silent fork; no diff/reconcile path.
- Single-binary global invalidation; no per-project version pin.
- Inner-loop skill-iteration cycle worsens (mitigation: `shark dev` disk-load mode).
- Status vocabulary leaks into hooks, scripts, dashboards, CLAUDE.md, muscle memory.
- Validation detects link rot, not semantic rot (mitigation: golden-output diff suite).
- Two repos, no shared CI (mitigation: CI hook running `shark validate` against the harness checkout).

---

*Last Updated*: 2026-06-22
