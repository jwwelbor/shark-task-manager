# Route-Based Short Workflow (example / testing)

The compact **`.sharkworkflow-short.json`** converted to the Shark 2.x
route-based `steps:` schema (Epic E35), produced by the converter in
[`../route-based-codex/convert-from-codex.py`](../route-based-codex/convert-from-codex.py).

Five entities (no tech-debt, which the short workflow doesn't define):
`epic, feature, task, bug, change`. Same conventions and caveats as the codex
example — see [`../route-based-codex/README.md`](../route-based-codex/README.md)
and [`docs/guides/route-based-workflow.md`](../../docs/guides/route-based-workflow.md).

Point a project at it (relative or absolute path both work — absolute is the
"shared bundle" path model):

```json
{ "workflow_config": "examples/route-based-short/workflow.yaml" }
```

Having this alongside the codex example lets you run two independent projects on
two different route-based workflows at once — a useful multi-config test.

Regenerate:

```bash
python3 examples/route-based-codex/convert-from-codex.py \
  path/to/source-workflow.json examples/route-based-short
```
