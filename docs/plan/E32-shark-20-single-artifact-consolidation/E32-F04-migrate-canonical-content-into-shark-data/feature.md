---
feature_key: E02-F04-migrate-canonical-content-into-shark-data
epic_key: E02
title: Migrate canonical content into shark-data/
description: Move templates, workflows, decoupled skills, and agents into canonical shark-data/ shipped by shark. Convert .sharkworkflow.json to YAML, convert .tmpl prompts to .md, replace LOAD-skill mentions with {{include:}}, move decoupled skills and agents, update embedded FS.
size: L
---

# Migrate canonical content into shark-data/

**Feature Key**: E02-F04

---

## Epic

- **Epic PRD**: [Epic](../epic.md)

---

## Goal

### Problem

After F1, the in-scope skills are decoupled but still live in `~/.claude/skills/`. After F2, the engine can read `shark-data/` but no canonical `shark-data/` content exists yet. After F3, the engine can embed and bootstrap a `shark-data/` tree but the embedded content is empty.

F4 is the **content migration** that fills `shark-data/` with the real artifacts and re-enables dispatch end-to-end.

### Solution

Six concrete moves:

1. **Workflow JSON → YAML.** Convert `.sharkworkflow.json` → `shark-data/workflow/{epic,feature,task,bug,change,tech-debt}.yaml`. Preserve `workflow_short` profiles (e.g., `feature.short.yaml` or `workflow-config.yaml` selector).
2. **Prompts `.tmpl` → `.md`.** Convert `shark-templates/{entity}/*.tmpl` → `shark-data/prompts/{entity}/*.md`. Add optional frontmatter where it adds value (variable manifest, includes list).
3. **Replace `LOAD: <skill>` with `{{include:}}`.** In every prompt file, replace the textual `LOAD: <skill>` mentions with `{{include: skills/<skill>/<workflow>.md}}` so prompts are self-contained.
4. **Move decoupled skills.** Output of F1 → `shark-data/skills/`. Skills become canonical defaults shipped with the binary.
5. **Move agents.** In-scope agents from `~/.claude/agents/` → `shark-data/agents/`.
6. **Update embedded FS.** F3's `embed.FS` (E8) now ships actual canonical content.
7. **Materialize `_partials/`.** F1.c produced `_partials_inventory.md`; F4 turns recurring scaffolding patterns into actual `_partials/` files used by prompts.

### Impact

- A fresh `shark init` lays down a complete tree.
- `/run E01-F02-???` on a real project exercises the new path end-to-end.
- Rendered prompts compared against pre-migration baseline — same agent behavior, same artifacts produced.

---

## Scope

### Workflow YAML conversion

- For each entity (`epic`, `feature`, `task`, `bug`, `change`, `tech-debt`):
  - Convert from `.sharkworkflow.json` to `shark-data/workflow/{entity}.yaml`.
  - Preserve every step, agent assignment, prompt-file pointer, skip_for, and routing key.
  - Preserve `workflow_short` profiles. Choice of layout: `feature.yaml` + `feature.short.yaml` selected via `workflow-config.yaml`.
- Validate round-trip: rendered output equivalent to current JSON for every status.

### Prompt file conversion

- `shark-templates/{entity}/{status}.tmpl` → `shark-data/prompts/{entity}/{status}.md`.
- Optional YAML frontmatter (manifest of variables and includes) where it adds value.
- All `LOAD: <skill>` text replaced with `{{include: skills/<skill>/<workflow>.md}}`.

### Skills and agents migration

- Move output of F1 (decoupled skills) from `~/.claude/skills/` → `shark-data/skills/`.
- Move in-scope agents from `~/.claude/agents/` → `shark-data/agents/`.
- The original `~/.claude/skills/` and `~/.claude/agents/` paths stay untouched in F4 — F5 deletes them.

### Partials materialization

- Take `_partials_inventory.md` from F1.c.
- For each recurring pattern (e.g., `_read_parent_context`, `_codex_gate`, `_advance`), create a partial file under `shark-data/prompts/_partials/`.
- Update the converted prompt files to reference these partials via `{{template "_name" .}}`.

### Embedded FS update

- Update `cmd/shark/`'s `//go:embed` to point at the canonical `shark-data/` (now populated).
- New shark binary release ships full default content.

---

## `shark-templates/` rename impact

Any external script that hardcodes the directory name breaks. **Audit before F4** — Bash hooks under `~/.claude/hooks/` and `~/projects/shark-task-manager/scripts/` are likely candidates.

---

## Agent dispatch routing — DECISION REQUIRED

F4 moves agents to `shark-data/agents/`, but Claude Code's `Agent` tool reads `~/.claude/agents/`. Three options:

- **Inline** the agent definition into the rendered prompt — loses sub-agent isolation benefits.
- **Copy on init** — `shark init` writes `shark-data/agents/` → `.claude/agents/`. Re-creates the duplication F1–F5 was trying to eliminate.
- **New harness↔shark agent contract** — harness asks shark for the agent definition before each spawn. Cleanest, but means `shark/SKILL.md` is no longer "5 lines"; orchestration regrows in disguise.

**This must be decided before F4 ships.** Captured as a follow-up idea.

---

## Acceptance Criteria / Exit gate

1. `shark init` lays down a complete tree on a fresh project.
2. `/run E01-F02-???` on a real project exercises the new path end-to-end.
3. Rendered prompts compared against pre-migration baseline — semantically equivalent (golden-output diff).
4. F1's "branch dispatch is broken by design" is **resolved** — F4 lands together with F1 to re-enable dispatch.
5. All `LOAD: <skill>` references in prompts replaced with `{{include:}}`.
6. `shark-templates/` audit complete: no external scripts hardcoding the directory name (or replacements identified).
7. Agent dispatch routing decision recorded in epic.md.

---

## Out of Scope (for F4)

- Deleting old paths in `~/.claude/skills/` and `~/.claude/agents/` (F5).
- Removing `shark-templates/` fallback (F6).
- Adding deprecation warnings to slash commands (F5).

---

## Dependencies

- **Blocks**: F5.
- **Blocked by**: F1, F3.

---

## Risks

- **Golden-output diff is the real exit gate.** `shark validate` catches link rot, not semantic rot. Manually compare rendered prompts on a representative sample before declaring F4 done.
- **`_partials/` inventory may be wrong.** F1.c produced an inventory based on extracted sidecars; F4 may discover during materialization that some "partials" are too varied to merge. Allow the inventory to be revised.
- **Workflow profile preservation.** `feature_short`, `task_short`, `epic_short` exist for a reason. Conversion must preserve them; validate by running an entity through the short-profile path.

---

*Last Updated*: 2026-05-10
