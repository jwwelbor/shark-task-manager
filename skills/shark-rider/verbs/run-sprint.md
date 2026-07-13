# /shark-rider run-sprint — Solo sprint execution

Solo sequential pull-loop: repeatedly calls `shark sprint next`, delegates each
entity to `/run`, and loops until the backlog is drained. Prompts before closing.

Usage: `/shark-rider run-sprint S### [--agent=TYPE] [--max-iterations=N] [--carryover=VALUE]`

- `--agent=TYPE` — pull only entities for a specific agent type
- `--max-iterations=N` — cap the loop at N iterations (default 50)
- `--carryover=VALUE` — pass to `shark sprint close` if the user confirms close

## Procedure

1. Read `skills/sprint-execution/SKILL.md` (under this shark skill's directory),
   follow the **solo (`/shark-rider run-sprint`)** workflow described there, passing any
   remaining arguments through.

## Notes

- For team mode (parallel feature groups via agent teams), use `/shark-rider run-sprint-team`.
- The sprint must be in `active` status; the skill will offer to start it if it is
  still in `planning`.
