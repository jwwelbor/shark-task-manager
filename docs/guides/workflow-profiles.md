# Workflow Configuration

> **Note:** The basic/advanced workflow "profiles" subsystem
> (`shark init update --workflow=...`, `shark init merge ...`, the
> `internal/init/profiles/` package) was **removed**. Workflows are now
> defined as **per-entity YAML files** under `shark-data/workflow/` and
> selected via the `workflow_config` field in `.sharkconfig.json`.
>
> A bare Shark 1.x JSON workflow file (e.g. `.sharkworkflow.json`) is no
> longer a valid `workflow_config` target — the loader rejects it with a
> migration hint. Run `shark init` to materialize the `shark-data/` tree.
>
> This page explains the current model. If you came here looking for the
> old `--workflow=basic|advanced` flags, see the
> [migration table](#migration-from-the-old-profile-system) below.

## Overview

Shark loads its workflow definitions (status list, status flow / outcome
routing, agent routing, orchestrator actions) from per-entity YAML files.
`workflow_config` in `.sharkconfig.json` points at either a **directory**
of per-entity YAML (the default) or a **master index file**:

```json
{
  "workflow_config": "shark-data/workflow"
}
```

```
shark-data/workflow/
├── task.yaml
├── feature.yaml
├── epic.yaml
├── bug.yaml
├── change.yaml
└── tech-debt.yaml
```

`shark admin init` materializes the `shark-data/` tree (workflows, prompts,
skills, agents). The `shark-data/overrides/` subtree layers on top of the
bundled defaults and is never overwritten by `shark admin init`.

See the [Route-Based Workflow Guide](route-based-workflow.md) for the
consolidated `steps:` schema, outcome routing, and master-index resolution.

## Customizing a workflow

1. Edit the per-entity YAML under `shark-data/workflow/` directly, **or**
2. Place an override at `shark-data/overrides/workflow/<entity>.yaml`, **or**
3. Point `workflow_config` at your own directory or master index file.

Overrides and files outside the bundled tree are left untouched by
`shark admin init`.

## Migration from the old profile system

| Old command | New approach |
|---|---|
| `shark init update --workflow=basic` | Edit `shark-data/workflow/*.yaml` (or supply overrides) for a compact flow |
| `shark init update --workflow=advanced` | Edit `shark-data/workflow/*.yaml` (or supply overrides) for the multi-stage flow |
| `shark init merge --workflow=advanced --force` | Edit the per-entity YAML / overrides directly |
| `shark init update --workflow=advanced --dry-run` | Inspect the YAML with any text editor before editing |
| `shark init update` (add missing fields) | No replacement — there are no profile-specific fields to merge anymore |

## Schema

The structure of a workflow file is documented in
[Workflow Configuration](../cli-reference/workflow-configuration.md) and the
route-based `steps:` schema in the
[Route-Based Workflow Guide](route-based-workflow.md).

## Related Documentation

- [Initialization](../cli-reference/initialization.md) — `shark admin init` and template re-sync behavior
- [Workflow Configuration](../cli-reference/workflow-configuration.md) — full schema reference
- [Configuration](../cli-reference/configuration.md) — `.sharkconfig.json` field reference
