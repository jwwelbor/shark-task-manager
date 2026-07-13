# /shark-rider plan-sprint — Sprint planning

Scope a sprint by surfacing backlog entities, proposing assignments grouped by
feature and agent type, and confirming additions — without starting the sprint.

Usage: `/shark-rider plan-sprint S### [--mode=interactive|auto] [--max-add=N]`

- `--mode=auto` — greedy-fill proposal with one confirmation gate
- `--mode=interactive` — (default) item-by-item confirmation per feature group
- `--max-add=N` — cap total entities added this session

## Procedure

1. Read `skills/sprint-planning/SKILL.md` (under this shark skill's directory) and
   follow its procedure, passing any remaining arguments through as that skill's
   arguments.
