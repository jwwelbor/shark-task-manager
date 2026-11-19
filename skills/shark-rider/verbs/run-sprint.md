# /shark-rider run-sprint — Solo sprint execution

Solo sequential pull-loop: repeatedly calls the sprint-scoped `shark plan S###`
selector, delegates each entity to `/shark-rider run`, and prompts before closing.

Usage: `/shark-rider run-sprint S### [--agent=TYPE] [--max-iterations=N] [--carryover=VALUE]`

- `--agent=TYPE` — pull only entities for a specific agent type
- `--max-iterations=N` — cap the loop at N iterations (default 50)
- `--carryover=VALUE` — pass to `shark sprint close` if the user confirms close

## Procedure

1. Read `skills/sprint-execution/SKILL.md` (under this shark skill's directory),
   follow the **solo (`/shark-rider run-sprint`)** workflow described there, passing any
   remaining arguments through.

## Notes

- For team topology mode, use `/shark-rider run-sprint-team`.
- A sprint in `planning` status is started automatically under the standing
  authorization recorded in the workflow.
