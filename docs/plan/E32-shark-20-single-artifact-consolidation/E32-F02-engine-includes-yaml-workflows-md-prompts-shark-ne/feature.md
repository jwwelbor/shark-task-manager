---
feature_key: E02-F02-engine-includes-yaml-md-shark-next
epic_key: E02
title: Engine — includes, YAML workflows, .md prompts, shark next
description: Shark engine changes E1-E5 — {{include:}} and {{augment:}} template directives, YAML workflow loader, .md prompt-file support, shark next command, shark-data/ resolution with shark-templates/ fallback.
size: L
---

# Engine — includes, YAML workflows, .md prompts, shark next

**Feature Key**: E02-F02

---

## Epic

- **Epic PRD**: [Epic](../epic.md)

---

## Goal

### Problem

The shark engine today reads `.tmpl` files via Go templates, references workflow definitions in a 1750-line `.sharkworkflow.json`, and relies on the harness to load skills at dispatch time via "LOAD: <skill>" text instructions. This means:

- Rendered prompts are not self-contained — the agent has to discover skill content via filesystem conventions baked into the harness.
- Workflow definitions are JSON-only; cumbersome to edit, harder to profile-variant.
- The harness owns dispatch loop logic.
- There's no override mechanism distinguishing canonical defaults from local edits.

### Solution

Five engine changes (E1–E5) that together let the engine assemble a fully-rendered, self-contained prompt:

- **E1** — `{{include: <path>}}` and `{{augment: <path>}}` template directives, with cycle detection, depth cap, size warning, and override resolution.
- **E2** — YAML workflow loader; reads `shark-data/workflow/{entity}.yaml` and stitches into the in-memory workflow model.
- **E3** — New `shark next <entity>` command. Returns JSON `{ prompt, agent_type, provider, model, action }`. Replaces harness-side dispatch logic.
- **E4** — `.md` prompt-file support. Engine reads any extension; strips optional YAML frontmatter; renders body as Go template.
- **E5** — `shark-data/` directory resolution. Engine looks up `shark-data/` (configurable via `.sharkconfig.json`); falls back to `shark-templates/` for one release.

### Impact

- Rendered prompts are **fully self-contained** — skill content is inlined at render time.
- Harness dispatch becomes ~5 lines: loop on `shark next`, spawn agent, `shark advance`.
- Old `shark-templates/` layout still works (fallback) — no daily-work disruption.
- Override path is wired in: `shark-data/overrides/<path>` wins over default.

---

## Scope

### E1 — `{{include:}}` and `{{augment:}}` directives (M)

- **Where**: `internal/templates/`
- Inline file content at render time.
- Cycle detection (proposed depth cap: 5).
- Size warning above threshold (proposed: 50KB).
- Override resolution: `shark-data/overrides/skills/quality/foo.md` fully replaces `shark-data/skills/quality/foo.md` — never merges.
- `{{augment:}}` differs from `{{include:}}` in (TBD — design note: probably "include but allow downstream override to extend rather than replace"). Confirm in implementation.

### E2 — YAML workflow loader (M)

- **Where**: `internal/config/`
- Read `shark-data/workflow/{entity}.yaml`.
- Stitch into the in-memory workflow model the engine already uses.
- Equivalent semantics to current `.sharkworkflow.json`.
- Start with **`task.yaml`** as proof; round-trip-equivalent to current JSON for one entity before doing the rest.

### E3 — `shark next <entity>` command (M)

- **Where**: `cmd/shark/` + `internal/runner/`
- Returns JSON shape:
  ```json
  {
    "prompt": "<fully assembled, skills inlined>",
    "agent_type": "developer",
    "provider": "claude-code",
    "model": "claude-sonnet-4-6",
    "action": "spawn_agent"
  }
  ```
- `action` values: `spawn_agent`, `pause`, `archive`, etc.
- Plugs into existing `RunController` + `AgentDispatcher` interface; doesn't replace.
- Replaces harness-side dispatch logic.

### E4 — `.md` prompt-file support (S)

- **Where**: `internal/templates/`
- Engine reads any extension (`.tmpl`, `.md`).
- Strips optional YAML frontmatter before rendering body as Go template.
- Frontmatter exposes step metadata (entity_type, includes, variables) for validation in F3.

### E5 — `shark-data/` resolution (S)

- **Where**: `internal/templates/` resolver
- Engine looks up `shark-data/`, configurable via `.sharkconfig.json`.
- Falls back to `shark-templates/` for one release (deliberate back-compat for daily work).

---

## Reuse what already works

- `shark-templates/partials/_*.tmpl` already implements reusable fragments via `{{template "_name" .}}`. **Keep both** mechanisms: in-tree `{{template "_partial" .}}` for partials, cross-tree `{{include: skills/.../foo.md}}` for skill inlining.
- `shark-templates/feature_short/`, `task_short/`, `epic_short/` already implement workflow profile variants. **Preserve the profile concept** — YAML files just live next to each other (`feature.yaml`, `feature.short.yaml`) with `workflow-config.yaml` selecting the active profile per entity.
- `RunController` + `AgentDispatcher` interface in `internal/runner/` already abstracts agent dispatch per provider. `shark next` plugs into this; doesn't replace it.

---

## Acceptance Criteria / Exit gate

1. Manual `shark-data/` for one feature.
2. `shark next E01-F02-001 --json | jq .prompt` returns a rendered prompt with skill content inlined.
3. Diff against the existing `.tmpl` output — semantically equivalent.
4. Old `shark-templates/` path still works (fallback).
5. `{{include:}}` cycle detection fires on a deliberate cycle test fixture.
6. Override resolution: `shark-data/overrides/<path>` wins over `shark-data/<path>` when both exist.

---

## Out of Scope (for F2)

- `shark init` / `shark upgrade` / `shark validate` (F3).
- `embed.FS` of canonical defaults (F3).
- Migrating actual content into `shark-data/` (F4).
- Per-harness prompt variants (`prompts/task/ready_for_development.codex.md`) — deferred post-epic.

---

## Dependencies

- **Blocks**: F3, F4.
- **Blocked by**: none. Can run in parallel with F1.

---

## Risks

- **`{{include:}}` blowup.** A skill including a skill including a skill produces huge prompts. Cycle detection + depth cap + size warning are mandatory, not nice-to-haves.
- **Override merge surprise.** `overrides/` must **fully replace** the default — never merge. Document this in the engine code AND in user-facing docs. A merge surprise will cause silent drift.
- **YAML round-trip equivalence.** First YAML conversion (E2) must produce identical workflow semantics to the current JSON. Validate on `task.yaml` as the proof entity before generalizing.

---

*Last Updated*: 2026-05-10
