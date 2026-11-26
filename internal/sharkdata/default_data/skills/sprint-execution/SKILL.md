---
name: sprint-execution
display_name: Sprint Execution
description: Solo and team sprint execution skills. The solo pull-loop uses /shark-rider run; the team alias delegates active-backlog selection and dispatch to the canonical /shark-rider run-agent-team topology adapter.
---

# Sprint Execution

## What This Is

A **sprint-shaped harness around the existing orchestration skill**. Solo mode
calls `shark plan S### --json` to select from the requested sprint and hands
the first ordered candidate to `/shark-rider run`. Team mode delegates
active-backlog selection and keyed dispatch to the canonical topology adapter.
Neither mode builds worker prompts.

```
shark plan S### --json → entity key → /shark-rider run {entity_key} → repeat; verify terminality before calling the sprint drained
```

This skill covers **two execution modes**:

- **`/shark-rider run-sprint S###`** (solo) — sequential pull-loop for one sprint. Calls `shark plan S### --json` (and passes `--agent` when requested), validates the returned sprint key, pulls one ordered entity at a time, and drives it to terminal status via `/shark-rider run` before selecting again. Suitable for any sprint regardless of entity mix.
- **`/shark-rider run-sprint-team S###`** (team) — a thin alias for
  `/shark-rider run-agent-team --sprint S###`. The canonical topology adapter
  uses the active backlog as the selection universe only after a read-only
  `shark sprint get S### --json` preflight confirms the sprint's configured
  execution-phase status. It sends each selected key to an ordinary keyed
  Rider parent. It does not group work by feature or create nested teams.

Both modes gate the sprint close operation on explicit user confirmation.

## Commands

```
/shark-rider run-sprint S###                              # Solo sequential execution
/shark-rider run-sprint S### --agent=backend              # Pull only backend-agent slice
/shark-rider run-sprint S### --max-iterations=N           # Cap loop at N (default 50)
/shark-rider run-sprint S### --carryover=backlog          # Carryover strategy if user confirms close

/shark-rider run-sprint-team S###                         # Team topology alias
```

**Read `workflows/run-sprint.md`** to get the solo pull-loop step-by-step procedure.
**Read `workflows/run-sprint-team.md`** to get the team execution step-by-step procedure.

## Key Design Decisions

- **Delegation to `/shark-rider run`**: Solo sprint execution delegates
  per-entity dispatch to the existing orchestration skill rather than
  re-implementing it. This ensures keyed `response.prompt` remains the worker
  payload and workflow source-of-truth stays in Shark.
- **Team topology alias**: Team sprint execution delegates to
  `/shark-rider run-agent-team --sprint S###`; `skills/shark-attack/workflows/parallel-team.md` owns
  active-backlog selection, topology, and integration.
- **`--max-iterations` cap**: Prevents runaway loops when sprint selection returns the same entity repeatedly (e.g., an entity that bounced back to a non-terminal state). Default cap is 50.
- **Explicit close gate**: Closing a sprint with carryover is a planning decision. Both skills require explicit user confirmation before calling `shark sprint close`, and both require a submitted Sprint Goal Review (`shark sprint goal-review` with `--outcome=accepted`) before that close call — a rejected or absent review returns the sprint to active and creates no completion row.
- **JSON-only shark consumption**: All shark calls use `--json` or `--field`. Human-readable output is not a stable contract.

## JSON Handling

- All shark calls use `--json` or `--field`
- Never truncate output with `head`, `tail`, or `grep`
- `shark plan S### --json` returns a sprint-scoped selection envelope. `/run-sprint` validates its `sprint_key`, dispatches `entity` for `select_item`, and dispatches the first ordered item from `entities` for `parallel_candidates`.
- The keyless `shark sprint next --json` command is not sprint-scoped. Use `shark plan S### --json` when the caller supplies a sprint key.
- When the sprint-scoped selector returns `pause`, `/run-sprint` checks `shark sprint backlog S### --view=ordered --all --json` without selector filters. It calls the sprint drained only after a complete inventory and workflow-status check confirms every assigned entity is terminal; incomplete evidence, unresolved statuses, remaining work, or an empty `--agent` slice leaves the run paused.
