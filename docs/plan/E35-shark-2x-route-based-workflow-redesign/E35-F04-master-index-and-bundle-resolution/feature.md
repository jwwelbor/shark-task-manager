---
feature_key: E35-F04-master-index-and-bundle-resolution
epic_key: E35
title: Master index and bundle resolution
description: .sharkconfig.json points at one master index file (absolute path allowed) mapping entity -> workflow yaml (mix profiles per entity). The shark-data bundle root is the resolution base: prompts/skills/agents resolve relative to it; local overrides/ layer on top. Absolute paths are the entire 'remote' story (shared mount / monorepo / submodule) — no URL fetch, cache, or pinning. Decision D6. Can run in parallel with F2.
size: M
---

# Master index and bundle resolution

**Feature Key**: E35-F04

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Design (single source of truth)**: [route-based-workflow-redesign.md](../../route-based-workflow-redesign.md) — D6, §5, §8

---

## Goal

### Problem

`.sharkconfig.json` points at a single workflow file. The redesign wants per-entity
workflows that can mix profiles (e.g. `task.short.yaml` alongside `feature.yaml`),
and it wants prompts/skills/agents to resolve from a known root so a project can
consume a *shared* bundle. There is no master index and no single resolution base
today.

### Solution

Introduce a **master index** and **bundle-rooted resolution** (D6, design §5):

```jsonc
// .sharkconfig.json
{ "workflow_config": "shark-data/workflow.yaml" }   // absolute path also allowed
```

```yaml
# shark-data/workflow.yaml  (the index)
entities:
  task:      workflow/task.short.yaml   # mix profiles per entity
  feature:   workflow/feature.yaml
  epic:      workflow/epic.yaml
  bug:       workflow/bug.yaml
  change:    workflow/change.yaml
  tech-debt: workflow/tech-debt.yaml
```

- The **shark-data bundle root is the resolution base.** Relative paths resolve
  against the index file's location; absolute paths point anywhere on the
  filesystem.
- Absolute paths are the *entire* "remote" story — a shared mount, monorepo, or git
  submodule. **No URL fetch, no cache, no pinning, no trust machinery.**
- The unit of sharing is the **whole bundle** (workflow + prompts + skills +
  agents). A project consumes a shared bundle by pointing at it and customizes only
  through its local `overrides/`. Everything the workflow references is guaranteed to
  resolve because it resolves from the same root.

### Impact

- Per-entity workflows, profiles mixable per entity.
- One resolution base for every artifact a workflow references.
- Shared bundles (mount/monorepo/submodule) work with zero fetch infrastructure.

---

## Scope

- Loader: accept the master index file; parse `entities:` → per-entity workflow
  paths; load each referenced workflow YAML.
- Resolution: relative paths resolve against the index file's directory (the bundle
  root); absolute paths resolve as-is.
- Layer local `overrides/` on top of bundle defaults at resolution time (override
  wins).
- `.sharkconfig.json` `workflow_config` accepts both relative and absolute index
  paths.
- Resolve prompts/skills/agents referenced by steps (from F01's `Step`) against the
  same bundle root.

### Out of Scope

- The `{{include:}}`/`{{augment:}}` rendering of resolved files — that is E32 engine
  work; this feature delivers *path resolution*, not template expansion.
- Outcome routing — F02.
- `shark validate` bundle-root checks are authored in F06 (this feature provides the
  resolver they call).

---

## Acceptance Criteria

1. `.sharkconfig.json → index file → per-entity workflow YAML` loads for all entity
   types; a missing entity mapping or missing target file is a clear error.
2. Relative artifact paths resolve against the index file's directory; absolute
   paths resolve as-is.
3. An artifact present under `overrides/<path>` wins over the bundle default.
4. An index placed outside the project (absolute path / shared mount) loads and its
   relative references still resolve from that external root.
5. Profiles are mixable per entity (e.g. `task.short.yaml` + `feature.yaml`).

---

## Verification

- Loader tests: index parsing, per-entity load, relative vs absolute resolution,
  override precedence, external-root resolution.
- A fixture bundle outside the repo tree loads end-to-end.
- `make fmt && make lint && make test` pass.

---

## Dependencies

- **Blocked by**: F01 (steps reference prompts/skills/agents the resolver locates) —
  loosely; the resolver can land in parallel with F02.
- **Blocks**: F06 (validate bundle-root checks build on this resolver).
- Independent of F02/F03 routing/lease work — schedulable in parallel.

---

*Last Updated*: 2026-06-23
