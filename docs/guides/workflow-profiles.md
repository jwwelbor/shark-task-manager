# Workflow Configuration

> **Note:** The basic/advanced workflow "profiles" subsystem
> (`shark init update --workflow=...`, `shark init merge ...`, the
> `internal/init/profiles/` package) was **removed**. Workflows are now
> defined as JSON files inside `shark-templates/` and selected via the
> `workflow_config` field in `.sharkconfig.json`.
>
> This page explains the current model. If you came here looking for the
> old `--workflow=basic|advanced` flags, see the
> [migration table](#migration-from-the-old-profile-system) below.

## Overview

Shark loads its workflow definition (status list, status flow, agent
routing, orchestrator actions) from a single JSON file. The path to that
file is configured in `.sharkconfig.json`:

```json
{
  "workflow_config": "shark-templates/.sharkworkflow-short.json"
}
```

The two workflow files shipped with `shark` live inside the embedded
`shark-templates/` tree and are copied into your project on
`shark admin init`:

| File | Description |
|---|---|
| `shark-templates/.sharkworkflow-short.json` | **Default.** Compact workflow suitable for solo developers and small teams. |
| `shark-templates/.sharkworkflow.json` | Long-form workflow with the full multi-stage TDD/refinement/QA pipeline and per-status agent routing. |

To switch workflows, edit `workflow_config` in `.sharkconfig.json`.

## Customizing a workflow

1. Copy one of the shipped files to a new path (anywhere in your project).
2. Edit it.
3. Set `workflow_config` in `.sharkconfig.json` to point at your copy.

`shark admin init` will leave your custom file alone — it only re-syncs
files inside `shark-templates/`.

> **Tip:** If you edit a file *inside* `shark-templates/` directly, the
> next `shark admin init` will detect the drift and warn you, but it will
> not overwrite your changes unless you re-run with `--force`. See
> [Templates re-sync behavior](../cli-reference/initialization.md#templates-re-sync-behavior).

## Migration from the old profile system

| Old command | New approach |
|---|---|
| `shark init update --workflow=basic` | Set `"workflow_config": "shark-templates/.sharkworkflow-short.json"` in `.sharkconfig.json` |
| `shark init update --workflow=advanced` | Set `"workflow_config": "shark-templates/.sharkworkflow.json"` in `.sharkconfig.json` |
| `shark init merge --workflow=advanced --force` | Same as above — edit `workflow_config` directly |
| `shark init update --workflow=advanced --dry-run` | Inspect the workflow file with any text editor before changing `workflow_config` |
| `shark init update` (add missing fields) | No replacement — there are no profile-specific fields to merge anymore |

## Choosing a workflow

### Use `.sharkworkflow-short.json` (default) if:
- You're working solo or with a small team (1-2 people)
- Your workflow is simple and flexible
- You don't need formal review processes
- You're prototyping or exploring

### Use `.sharkworkflow.json` (long form) if:
- You have a team with defined roles (3+ people)
- You practice test-driven development with discrete refinement, code
  review, and QA phases
- You need agent routing per status (BA, tech_lead, developer, qa,
  product_owner)

## Schema

The structure of a workflow file is documented in
[Workflow Configuration](../cli-reference/workflow-configuration.md).

## Related Documentation

- [Initialization](../cli-reference/initialization.md) — `shark admin init` and template re-sync behavior
- [Workflow Configuration](../cli-reference/workflow-configuration.md) — full schema reference
- [Configuration](../cli-reference/configuration.md) — `.sharkconfig.json` field reference
