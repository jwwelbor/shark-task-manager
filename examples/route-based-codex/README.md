# Route-Based Codex Workflow (example / testing)

These are the **codex workflow converted to the Shark 2.x route-based `steps:`
schema** (Epic E35), produced by `convert-from-codex.py` from
`shark-templates/.sharkworkflow-codex.json`.

They are an **example/testing artifact**, not canonical config. Outcome
assignments (`pass`/`fail`/`blocked`) are derived heuristically by phase order
and should be reviewed before any production use.

## Layout

```
workflow.yaml            # master index: entity -> workflow file
workflow/
  epic.yaml feature.yaml task.yaml bug.yaml change.yaml tech-debt.yaml
convert-from-codex.py    # the converter (re-run to regenerate)
```

Each entity file uses the consolidated `steps:` schema: `ready_for_X`/`in_X`
pairs collapsed into one step with `aliases:` for the old names, a per-step
`outcomes:` map, and `parking:`/`terminal:` flags. See
[`docs/guides/route-based-workflow.md`](../../docs/guides/route-based-workflow.md).

## How to test against a database

> Do this on a **local copy** of your data first. The status migration rewrites
> the live `status` column.

1. Point `workflow_config` at the index in `.sharkconfig.json`:
   ```json
   { "workflow_config": "examples/route-based-codex/workflow.yaml" }
   ```
2. Validate it loads:
   ```bash
   shark admin config validate
   ```
3. Preview the status migration (old names -> new step names):
   ```bash
   shark admin migrate statuses          # dry-run
   shark admin migrate statuses --apply  # execute (single transaction)
   ```
4. Use the route-based commands:
   ```bash
   shark status advance <key> --outcome pass|fail|blocked
   shark claim <key> --by <agent>        # prints a session id
   shark heartbeat <key> --session <sid> --progress 0.5
   shark release <key> --session <sid>   # (alias: unclaim)
   ```

## Known limitation: task lifecycle fidelity

The codex **task** workflow is compact (`draft → development → completed`). Real
task data often carries richer statuses (`ready_for_code_review`,
`ready_for_qa`, `ready_for_approval`, …) from the long workflow. The converter
aliases all of those to `development` so migration leaves no orphan — which
**collapses mid-pipeline tasks into `development`**. If you want to preserve the
full dev → code-review → qa → approval lifecycle, convert
`shark-templates/.sharkworkflow.json` (the long workflow) for the task entity
instead, and rely on the master index to mix profiles per entity.

Regenerate after editing the source workflow:

```bash
python3 examples/route-based-codex/convert-from-codex.py \
  shark-templates/.sharkworkflow-codex.json examples/route-based-codex
```
