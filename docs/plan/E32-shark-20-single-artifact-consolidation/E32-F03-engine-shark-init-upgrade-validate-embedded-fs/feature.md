---
feature_key: E02-F03-engine-shark-init-upgrade-validate-embedded-fs
epic_key: E02
title: Engine — shark init, upgrade, validate, embedded FS
description: Shark engine changes E6-E9 — shark init writes embedded shark-data/ to project root, shark upgrade updates everything except overrides/, embed.FS of canonical shark-data/ shipped with binary, shark validate checks template parse + include resolution + skill purity + workflow refs.
size: M
---

# Engine — shark init, upgrade, validate, embedded FS

**Feature Key**: E02-F03

---

## Epic

- **Epic PRD**: [Epic](../epic.md)

---

## Goal

### Problem

After F2, the engine can render fully-assembled prompts from a `shark-data/` tree — but a user still has to **manually create that tree**. There's no `shark init` to lay it down, no `shark upgrade` to pull in newer canonical defaults, and no `shark validate` to catch link rot before dispatch fails at runtime.

Without F3, "shark 2.0" is just a smarter renderer. With F3, it's a single-binary deployable.

### Solution

Four engine commands (E6–E9) backed by an embedded filesystem of canonical defaults:

- **E6** — `shark init`. Copies embedded default `shark-data/` into project root. Idempotent; refuses to overwrite if already initialized.
- **E7** — `shark upgrade`. Updates everything **except** `shark-data/overrides/`. Diff/dry-run flag.
- **E8** — `embed.FS` of canonical `shark-data/` shipped with the binary via `//go:embed`.
- **E9** — `shark validate`. Checks: skills don't reference shark/status names; all `{{include:}}` paths resolve; all `agent_type` and `prompt` references in workflow YAML exist; no template parse errors.

### Impact

- **One-command bootstrap.** `cd /tmp/fresh && shark init && shark validate && shark create epic "test" && shark next <new-key>` runs clean.
- **Override safety.** `shark upgrade` updates canonical defaults but never touches `shark-data/overrides/`.
- **Link-rot detection at commit time.** `shark validate` catches missing skills, broken includes, missing agent refs before dispatch.

---

## Scope

### E6 — `shark init` command (S)

- **Where**: `cmd/shark/` + `embed.FS`
- Copies embedded default `shark-data/` into project root.
- Idempotent: refuses to overwrite if already initialized (writes `.sharkconfig.json` initialized-with version on first run).
- Reuse `shark sync` machinery for filesystem ↔ DB sync rather than duplicating it.

### E7 — `shark upgrade` command (S)

- **Where**: `cmd/shark/`
- Diff embedded `shark-data/` against on-disk `shark-data/`.
- Apply with `--dry-run` flag for preview.
- **Never touches `shark-data/overrides/`** — overrides are fully off-limits.
- Document the merge story for "I want new defaults but keep my overrides."

### E8 — Embed canonical defaults (S)

- **Where**: `cmd/shark/`
- `//go:embed` the canonical `shark-data/` (skills, agents, prompts, workflows) in the binary.
- Each engine release re-embeds the latest content.

### E9 — `shark validate` (M)

- **Where**: `cmd/shark/`
- Checks:
  - Skills don't reference shark/status names (lexical signal — passing is necessary but not sufficient; structural decoupling validated separately).
  - All `{{include:}}` paths resolve.
  - All `agent_type` and `prompt` references in workflow YAML exist.
  - No template parse errors.
- Add to CI as a pre-commit/CI hook in shark-task-manager.

---

## Acceptance Criteria / Exit gate

1. `cd /tmp/fresh && shark init && shark validate` succeeds on a fresh project.
2. `shark create epic "test" && shark next <new-key>` works on a brand-new project after creating one entity.
3. `shark upgrade --dry-run` shows the canonical-default diff without applying.
4. `shark upgrade` applied: `shark-data/overrides/` is byte-identical before and after.
5. `shark validate` fails on a deliberately-broken `{{include:}}` reference.
6. `shark validate` fails on a workflow YAML referencing a nonexistent agent or prompt.

---

## Out of Scope (for F3)

- Migrating canonical content into `shark-data/` (F4 — F3 only ships the embedding mechanism, not the final content).
- Repointing the harness (F5).
- Removing fallback paths (F6).

---

## Dependencies

- **Blocks**: F4.
- **Blocked by**: F2 (needs `{{include:}}` directives, YAML loader, `.md` support, `shark next` to validate against).

---

## Risks

- **Validation can detect link rot, not semantic rot.** `shark validate` catches unresolved `{{include:}}` paths and missing agent refs. It can't catch a status template including the *wrong* skill workflow path — content resolves, plausibly relevant, behavior silently degraded. *Recommendation*: add a golden-output diff suite to F3's exit gate — render a small set of representative dispatches before/after and require manual review of any change.
- **Inner-loop dev cycle for skill iteration gets worse.** Today: edit `~/.claude/skills/quality/...`, save, run, see effect. Post-consolidation: edit `shark-data/skills/quality/...` in the shark repo → rebuild shark → `shark upgrade` in target project → test. *Mitigation candidate*: a `shark dev` mode that loads `shark-data/` from disk (override path or current working dir) instead of `embed.FS`. Worth folding into F3 if it doesn't slip the schedule.
- **Single-binary global invalidation.** Embedded FS means a shark release ships canonical defaults to every project on `shark upgrade`. A bug in v2.4 default `qa-testing.md` silently changes prompts in every non-overridden project. No per-project version pin yet. *Mitigation candidate*: `.sharkconfig.json` records initialized-with version; `shark next` warns when running on a project initialized far behind current.

---

*Last Updated*: 2026-05-10
