---
name: sprint-planning
display_name: Sprint Planning
description: Mode-aware sprint scoping skill that walks an orchestrator or human through planning a sprint by reading shark sprint plan and readiness data, proposing entity assignments, and confirming changes. Never starts the sprint — starting is an explicit user action via /run-sprint.
---

# Sprint Planning

## What This Is

A **mode-aware sprint scoping harness**. It reads `shark sprint plan` once at the start, surfaces backlog entities grouped by feature and agent type, and walks through assignment confirmation — either item-by-item (interactive mode) or with a single greedy-fill proposal (auto mode). When done, it reports the readiness-score delta and suggests `/run-sprint` as the next step. It does NOT call `shark sprint start`.

```
shark sprint plan {S###} --json → surface backlog → confirm assignments → shark sprint readiness {S###} --json → report delta
```

It does NOT decide which sprint strategy to use. It reads capacity and readiness from shark, proposes based on that data, and requires explicit user confirmation before any `shark sprint add` call.

## Command

```
/plan-sprint S###                       # Interactive mode (default)
/plan-sprint S###  --mode=auto          # Greedy-fill with single confirmation
/plan-sprint S###  --mode=interactive   # Explicit interactive (same as default)
/plan-sprint S###  --max-add=N          # Limit total entities added this session
```

See: `workflows/plan-sprint.md` for the full step-by-step workflow.

## Modes

- **interactive** (default) — surfaces each backlog group (by feature, then standalones), asks which items to add, calls `shark sprint add` only after explicit per-item or per-group confirmation.
- **auto** — greedily selects backlog entities up to capacity per agent type, sorted by priority descending then size ascending, proposes the full plan, asks for ONE final confirmation before calling `shark sprint add` for each selected entity.

## JSON Handling

- All shark calls use `--json` or `--field`
- Never truncate output with `head`, `tail`, or `grep`
- Sprint plan JSON includes backlog list, capacity per agent type, and readiness score
