---
name: sprint-execution
display_name: Sprint Execution
description: Solo and team sprint execution skills. The solo pull-loop uses /shark-rider run; the team alias delegates active-backlog selection and dispatch to the canonical /shark-rider run-agent-team topology adapter.
---

# Sprint Execution

## What This Is

A **sprint-shaped harness around the existing orchestration skill**. Solo mode
calls `shark sprint next --json` and hands each returned entity to
`/shark-rider run`. Team mode delegates active-backlog selection and keyed
dispatch to the canonical topology adapter. Neither mode builds worker prompts.

```
shark sprint next --json → entity key → /shark-rider run {entity_key} → loop until empty
```

This skill covers **two execution modes**:

- **`/shark-rider run-sprint S###`** (solo) — sequential pull-loop for a single Claude session. Pulls one entity at a time, drives it to terminal status via `/shark-rider run`, then pulls the next. Suitable for any sprint regardless of entity mix.
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
  `/shark-rider run-agent-team --sprint S###`; `parallel-team.md` owns
  active-backlog selection, topology, and integration.
- **`--max-iterations` cap**: Prevents runaway loops when `shark sprint next` returns the same entity repeatedly (e.g., an entity that bounced back to a non-terminal state). Default cap is 50.
- **Explicit close gate**: Closing a sprint with carryover is a planning decision. Both skills require explicit user confirmation before calling `shark sprint close`, and both require a submitted Sprint Goal Review (`shark sprint goal-review` with `--outcome=accepted`) before that close call — a rejected or absent review returns the sprint to active and creates no completion row.
- **JSON-only shark consumption**: All shark calls use `--json` or `--field`. Human-readable output is not a stable contract.

## JSON Handling

- All shark calls use `--json` or `--field`
- Never truncate output with `head`, `tail`, or `grep`
- `shark sprint next --json` returns the next entity object or an empty result; both cases are handled
