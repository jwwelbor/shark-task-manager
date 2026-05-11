---
epic_key: E02
title: Shark 2.0 — Single-Artifact Consolidation
description: Consolidate the shark workflow stack — Go CLI engine, ~10 shark-coupled skills, and 15 agent definitions — into a single drop-in artifact installed via 'shark init'. Decouple craft (skills) from workflow scaffolding (prompts) using a structural test rather than lexical search. Engine assembles fully-rendered prompts via 'shark next', removing the need for harness-side dispatch logic. Six features F1–F6.
size: XXL
---

# Shark 2.0 — Single-Artifact Consolidation

**Epic Key**: E02

---

## Goal

### Problem

The shark workflow stack ships across three uncoordinated locations:

- `~/projects/shark-task-manager/` — Go CLI engine + 1750-line `shark-templates/.sharkworkflow.json` + per-status `*.tmpl` instruction files.
- `~/.claude/skills/` — 29 skills, ~10 of which are referenced by shark prompts and embed shark's workflow contract directly into their content (status names, `shark --field` storage calls, gate enforcement).
- `~/.claude/commands/` + `~/.claude/agents/` — slash commands and 15 agent definitions, tightly coupled to shark.

Three concrete failures result:

1. **Structural coupling, not lexical.** Shark-referenced skills don't just *mention* shark — they embed the workflow contract: fetching entity context, storing notes via `shark --field`, gating transitions, advancing status. Stripping shark vocabulary leaves the contract intact. An audit of `specification-writing` alone found 401 shark-vocab hits across 4,078 lines, with state mutations woven into the craft.
2. **Discovery, not inlining.** Templates say `LOAD: quality skill` as text and the agent has to find the skill at runtime. Works for Claude Code (which ships `.claude/skills/`); breaks for Codex/Copilot/Gemini because the rendered prompt isn't self-contained.
3. **No single deployable.** A new project today needs the Go binary + a synced copy of `~/.claude/skills/` + a copy of `~/.claude/agents/` + a `shark-templates/` directory. There is no `shark init` that lays everything down.

### Solution

Ship **one drop-in artifact**. A user installs the shark binary and runs `shark init`, which lays down a `shark-data/` directory in the project containing everything needed to run shark workflows. Skills live there in standalone form; the harness only contains a tiny `shark/SKILL.md` trigger.

Two mechanical shifts power this:

- **Structural decoupling.** Extract workflow scaffolding (gates, fetches, mutations, advancements) from craft (how to do the activity). Skills declare `inputs:` frontmatter; prompts promise to provide those inputs. The decision test for any line: *"if you removed this, would the activity still be that activity?"* — yes → skill (craft); no → prompt (scaffolding).
- **Engine-driven dispatch.** Replace the orchestration skill with a `shark next <entity>` command that emits a fully-rendered prompt + dispatch metadata. The harness loop becomes ~5 lines: `loop: response = shark next; spawn agent; shark advance`.

### Impact

- **Single-command bootstrap.** New projects go from ~4 disjoint installs to `shark init` in ~seconds.
- **Cross-harness portable.** Rendered prompts are self-contained; shark workflows run identically under Codex/Copilot/Gemini.
- **Decoupled craft.** Skills become standalone-readable; a stranger with no shark context can execute them. Craft evolves independently of workflow.
- **Override safety.** `shark-data/overrides/` survives `shark upgrade`; canonical defaults update without trampling local edits.

---

## Business Value

**Rating**: High

This is foundational refactoring that unblocks multi-harness support, future-proofs the craft library against host-system churn, and pays down structural debt that has compounded as skills accumulated workflow contract logic. Without it, every new harness integration requires duplicating shark's runtime assumptions inside each skill.

---

## Architecture rule: skills vs prompts

The single decision that drives F1's shape:

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

The test cleanly separates **craft-checks** (intrinsic to the activity) from **gate-checks** (workflow demands). Craft stays in the skill; gates move to the prompt.

**Why this matters more than lexical purity**: a skill can be 100% shark-vocab-free and still embed the host contract — assume an entity ID, fetch parent context, write notes back, advance state. The original "grep for `shark` and move it" framing would pass a clean skill that still fails as a portable artifact. The craft/scaffolding split is the actual decoupling.

**Inputs contract** — every craft skill workflow declares its inputs in frontmatter:

```markdown
---
inputs:
  - spec_path: absolute path to the feature spec markdown
  - scope_summary: 1-paragraph description of what's being QA'd
outputs:
  - defect_report: structured markdown
---
```

This is the host contract, materialized as data. Prompts promise to provide these inputs; skills promise to operate on them without further assumptions.

---

## Target architecture

```
<harness>/.claude/skills/shark/SKILL.md             # tiny harness-side trigger, ~50 lines
                                                     # contains the "loop on shark next" instruction

<project>/.sharkconfig.json                          # data dir path, profile selector, db path, harness type
<project>/shark-data/
  workflow-config.yaml                               # which workflow yaml per entity (default vs short profile)
  workflow/
    epic.yaml
    feature.yaml
    task.yaml
    bug.yaml
    change.yaml
    tech-debt.yaml
  prompts/
    epic/{status}.md                                 # adapter layer: routing + skill includes
    feature/{status}.md
    task/{status}.md
    bug/{status}.md
    change/{status}.md
    _partials/                                       # _read_section.md, _exit_gate.md, _tdd_process.md, etc.
  skills/
    specification-writing/  (SKILL.md + workflows/ + context/)
    quality/                (SKILL.md + workflows/ + context/)
    architecture/           (SKILL.md + workflows/ + context/)
    research/               (SKILL.md + workflows/ + context/)
    implementation/         (SKILL.md + workflows/ + context/)
    test-driven-development/(SKILL.md)
    assessment/             (SKILL.md)
    uat/                    (SKILL.md)
    debugging/              (SKILL.md + workflows/)
  agents/
    architect.md, business-analyst.md, developer.md, qa.md, tech-lead.md,
    product-manager.md, tech-director.md, researcher.md, uat-agent.md
  overrides/                                         # local-only; shark upgrades never touch
    {skills,agents,prompts,workflow}/                # mirror tree; entries here win over defaults
```

### Layer rules

- **Workflow YAML** owns step order, routing keys, agent assignment, prompt-file pointer, and skip_for. Status name = step name = entity status.
- **Prompt files** own *workflow scaffolding*: read inputs, prepare files, run skills, gate outputs, advance state. Tightly coupled to shark by design — allowed to be ugly. The contract between prompt and skill is the skill's `inputs:` frontmatter.
- **Skills** own *craft*: how to do the activity. Read with no shark context, the skill must still execute. Skill content makes **no implicit assumption** about who called it, what state preceded the call, or what comes next.
- **Engine** owns state, template rendering, override resolution, and prompt assembly.

### Simplified dispatch model

The orchestration skill goes away because the loop logic moves into the engine + a tiny harness trigger:

**Today** (skill-driven dispatch):
```
harness → /run E01-F02-001
       → loads orchestration/workflows/run.md
       → runs Bash to call shark get, parse JSON, extract orchestrator_action
       → spawns agent with the instruction string from shark
       → on completion, calls shark status advance
       → loops
```

**Shark 2.0** (engine-driven dispatch):
```
harness → /run E01-F02-001 (still a trigger, but trivial now)
       → loops:
           response = shark next E01-F02-001 --json
           # response = { prompt: "<fully assembled, skills inlined>",
           #              agent_type, provider, model, action: "spawn_agent" | "pause" | "archive" | ... }
           if action == "spawn_agent": spawn agent with response.prompt → call `shark advance E01-F02-001` after completion
           if action == "pause" / "archive": stop and report
       → that's the entire loop
```

This means:
- The harness doesn't know workflow logic.
- The prompt is fully assembled — all `{{include:}}` resolution, variable substitution, partial expansion happens in the engine before the prompt leaves shark.
- Multi-harness support is trivial later.

---

## Confirmed scope

In-scope skills (decouple, move to `shark-data/skills/`):

| Skill                     | Status refs | Decision                                                                     |
| ------------------------- | ----------- | ---------------------------------------------------------------------------- |
| `specification-writing`   | 24          | In scope                                                                     |
| `architecture`            | 9           | In scope                                                                     |
| `research`                | 9           | In scope; absorb stale `discovery` references                                |
| `quality`                 | 7           | In scope                                                                     |
| `implementation`          | 5           | In scope                                                                     |
| `test-driven-development` | 1+          | In scope                                                                     |
| `debugging`               | 2           | In scope                                                                     |
| `assessment`              | 1           | In scope                                                                     |
| `uat`                     | 1           | In scope                                                                     |
| `orchestration`           | 2           | **Removed entirely** (replaced by `shark next` engine command)               |
| `shark`                   | 38          | Stays as harness-side trigger skill at `<harness>/.claude/skills/shark/`     |

In-scope agents (move to `shark-data/agents/`): architect, business-analyst, developer, qa, tech-lead, product-manager, tech-director, researcher, uat-agent.

**Stale references** found in workflow JSON: `discovery` (4 refs, no skill exists — collapse into `research`), `build` (1 ref — likely meant `devops`, drop the ref or replace).

**Out of scope** (stay untouched in `~/.claude/skills/`): brainstorming, brownfield-analysis, claude-md-improver, code-simplifier, collaboration, cx-designer, feature-design, frontend-design, graphify, product-design, refactoring, resolve-pr-comments, skill-creator, socratic-method, triage, vue-development, xlsx.

---

## Resolved decisions

| # | Decision | Resolution |
| - | -------- | ---------- |
| 1 | Engine work in or out? | **In.** Defined here, implemented as part of the epic. |
| 2 | Fate of in-scope skills in `~/.claude/skills/`? | **Delete after migration.** `shark-data/` becomes single source of truth. |
| 3 | What about the `orchestration` skill? | **Removed.** Shark CLI itself emits fully-rendered prompts; harness loop becomes ~5 lines in `shark/SKILL.md`. |
| 4 | Prompt-file format? | **`.md`** (Go templates don't require `.tmpl`). Optional YAML frontmatter for metadata; engine strips it before render. |
| 5 | Agents — move to `shark-data/agents/`? | **Yes.** They're shark-coupled (status vocabulary). `~/.claude/agents/` keeps only harness-level helpers (Explore, Plan, general-purpose). |
| 6 | Slash commands fate? | **Deprecation notes in place.** Leave for one release; remove in cleanup phase. |
| 7 | Decoupling rule — lexical or structural? | **Structural.** See "Architecture rule" above. |

---

## Engine changes (in `~/projects/shark-task-manager`)

| # | Change | Where | Effort |
| - | ------ | ----- | ------ |
| E1 | `{{include: <path>}}` and `{{augment: <path>}}` template directives. Inline file content; cycle detection; depth cap; size warning. Resolves overrides (`shark-data/overrides/<path>` wins over default). | `internal/templates/` | M |
| E2 | YAML workflow loader. Read `shark-data/workflow/{entity}.yaml`; stitch into in-memory workflow model the engine already uses. Equivalent semantics to current JSON. | `internal/config/` | M |
| E3 | New `shark next <entity>` command. Returns JSON: `{ prompt, agent_type, provider, model, action }`. Replaces the harness-side dispatch logic. | `cmd/shark/` + `internal/runner/` | M |
| E4 | `.md` prompt-file support. Engine reads any extension; strips optional YAML frontmatter; renders body as Go template. Frontmatter exposes step metadata for validation. | `internal/templates/` | S |
| E5 | `shark-data/` resolution. Engine looks up `shark-data/` (configurable via `.sharkconfig.json`); falls back to `shark-templates/` for one release. | `internal/templates/` resolver | S |
| E6 | `shark init` command. Copies embedded default `shark-data/` into project root. Idempotent; refuses to overwrite if already initialized. | `cmd/shark/` + `embed.FS` | S |
| E7 | `shark upgrade` command. Updates everything **except** `shark-data/overrides/`. Diff/dry-run flag. | `cmd/shark/` | S |
| E8 | Embed default `shark-data/` (skills, agents, prompts, workflows) in the binary via `//go:embed`. | `cmd/shark/` | S |
| E9 | Validation tooling — `shark validate` checks: skills don't reference shark/status names; all `{{include:}}` paths resolve; all `agent_type` and `prompt` references in workflow YAML exist; no template parse errors. | `cmd/shark/` | M |

**Deferred (post-epic)**: per-harness prompt variants (`prompts/task/ready_for_development.codex.md`), automatic harness detection, per-provider prompt token optimization. Mention in the epic but ship in a follow-up epic.

---

## Phase / feature breakdown

Six features, each independently shippable. Phases run mostly sequentially with one dependency arrow.

| Feature | Title | Order | Depends on |
| ------- | ----- | ----- | ---------- |
| F01 | Extract craft from scaffolding (per-workflow-file) | 1 | — |
| F02 | Engine — includes, YAML workflows, .md prompts, shark next | 2 | — |
| F03 | Engine — shark init, upgrade, validate, embedded FS | 3 | F02 |
| F04 | Migrate canonical content into `shark-data/` | 4 | F01, F03 |
| F05 | Repoint harness, deprecate, simplify `shark/SKILL.md` | 5 | F04 |
| F06 | Cleanup — drop fallbacks, retire `shark-templates/` | 6 | F05 |

F1 and F2 can run in parallel since both gate F4 via different paths. See per-feature `feature.md` for details.

---

## Reuse what already works

- `shark-templates/partials/_*.tmpl` already implements reusable fragments via `{{template "_name" .}}`. Plan **keeps both** mechanisms: in-tree `{{template "_partial" .}}` for partials, cross-tree `{{include: skills/.../foo.md}}` for skill inlining.
- `shark-templates/feature_short/`, `task_short/`, `epic_short/` already implement workflow profile variants. Plan **preserves the profile concept** — YAML files just live next to each other (`feature.yaml`, `feature.short.yaml`) with `workflow-config.yaml` selecting the active profile per entity.
- `RunController` + `AgentDispatcher` interface in `internal/runner/` already abstracts agent dispatch per provider. `shark next` plugs into this; doesn't replace it.
- `shark sync` for filesystem ↔ DB sync — `shark init` should reuse this machinery rather than duplicate it.

---

## Verification

| Feature | How to verify |
| ------- | -------------- |
| F1      | (a) Each craft skill workflow standalone-readable by a stranger with no shark context. (b) Every craft file has `inputs:` (and `outputs:` where meaningful) frontmatter. (c) Sidecars present in `_extracted/` with tagged blocks (`# fetch`, `# gate`, `# mutate`, `# advance`). (d) `_partials_inventory.md` exists. (e) Branch dispatch is **expected to fail** — F1 is not independently mergeable; lands with F4. |
| F2      | Manual `shark-data/` for one feature; `shark next E01-F02-001 --json \| jq .prompt` returns rendered prompt with skill content inlined; old `shark-templates/` path still works. |
| F3      | `cd /tmp/fresh && shark init && shark validate && shark create epic "test" && shark next <new-key>` — full bootstrap path runs clean. |
| F4      | On a real project, `/run E01-F02-???` produces same agent behavior + same artifacts as pre-migration. Diff rendered prompts before/after — semantically equivalent. |
| F5      | Move `~/.claude/skills/quality/` aside; `/run` workflow that uses quality still works (because everything resolves from `shark-data/skills/`). |
| F6      | `grep -r "shark-templates" cmd/ internal/` returns nothing; old `.sharkworkflow.json` files refused with deprecation error. |

---

## Risks + watch-outs

- **`{{include:}}` blowup.** A skill including a skill including a skill produces huge prompts. Need cycle detection, depth cap (proposed: 5), and a render-size warning above some threshold (proposed: 50KB).
- **Override merge semantics.** `shark-data/overrides/skills/quality/workflows/review-code.md` must **fully replace** the default — never merge. Document this clearly. A merge surprise will cause silent drift.
- **Two stale workflow refs** (`discovery`, `build`) — fix in F1 before they get carried into the new tree.
- **`shark-templates/` rename.** Any external script that hardcodes the directory name breaks. Audit before F4. Bash hooks under `~/.claude/hooks/` and `~/projects/shark-task-manager/scripts/` are the likely candidates.
- **Daily work depends on shark.** F1, F2, F3, F4 must each be backwards-compatible with the existing setup; F5 is the first phase that breaks the old paths, and that's deliberate.
- **Embedded FS update flow.** Each engine release re-embeds the latest `shark-data/`. Need a story for "I want to pull in the new defaults but keep my overrides" — `shark upgrade` covers this but document the merge story.

---

## What consolidation does NOT solve (follow-up backlog)

These problems persist after F1–F6 land. Captured as ideas linked to E02 so they're not lost:

- Override drift = silent fork. No diff/reconcile path; long-term every active project forks.
- Single-binary global invalidation. Embedded FS means a shark release ships canonical defaults to every project on `shark upgrade`. No per-project version pin.
- Inner-loop dev cycle for skill iteration gets worse. Today: edit `~/.claude/skills/quality/...`, save, run, see effect. Post-consolidation: edit in shark repo → rebuild shark → `shark upgrade` → test. *Mitigation candidate: `shark dev` mode loading from disk instead of embed.FS.*
- Status vocabulary leaks into consumers shark can't see — hooks, scripts, dashboards, CLAUDE.md content, slash commands, even muscle memory.
- Agent dispatch routing is under-specified. F4 moves agents to `shark-data/agents/`, but Claude Code's `Agent` tool reads `~/.claude/agents/`. Three options (inline / copy on init / new harness↔shark agent contract) — decide before F4.
- Validation can detect link rot, not semantic rot. Recommendation: add a golden-output diff suite to F3's exit gate.
- F5 is a global hard cutover for the shared harness. Mitigation: ship harness skills with deprecation warnings for one release before deletion.
- The "tiny `shark/SKILL.md`" claim won't survive harness diversity. Plan accordingly: don't budget F5 as if `shark/SKILL.md` is trivial.
- Two repos, no shared CI. F1 lives in `~/.claude/`; F2/F3 in shark-task-manager. F4 spans both. Recommendation: CI hook in shark-task-manager that runs `shark validate` against a checkout of the harness skills.

---

## Quick Reference

**Primary Users**: Solo developer (the user) running shark workflows under Claude Code; future operators running shark under Codex / Copilot / Gemini harnesses.

**Key Features**:
- Structural decoupling of craft (skills) from workflow scaffolding (prompts) with `inputs:` frontmatter contract.
- `shark next <entity>` engine command emitting fully-rendered prompts with skills inlined.
- `shark init` / `shark upgrade` / `shark validate` lifecycle commands; embedded FS shipping canonical defaults.
- Single `shark-data/` deployable replacing the three-location stack; `overrides/` for local-only changes.

**Success Criteria**:
- A fresh project can run shark workflows entirely from `shark-data/` plus only the `shark` skill in the harness.
- Rendered prompts before/after are semantically equivalent (same agent behavior, same artifacts produced).
- `~/.claude/skills/{specification-writing,quality,architecture,research,implementation,test-driven-development,assessment,uat,debugging,orchestration}/` are deleted; in-scope agents moved out of `~/.claude/agents/`.

**Timeline**: Multi-sprint epic; 6 features sequenced with one parallel branch (F1 || F2 → F3 → F4 → F5 → F6).

---

*Last Updated*: 2026-05-10
