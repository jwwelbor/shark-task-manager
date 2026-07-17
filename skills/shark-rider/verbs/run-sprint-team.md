# /shark-rider run-sprint-team — Team sprint execution

Team pull-loop: groups sprint entities by feature, dispatches each feature group
via `/shark-rider run-agent-team` (one team at a time), falls back to `/shark-rider run` for standalones.

Usage: `/shark-rider run-sprint-team S### [--size=N] [--features=E##-F##,...] [--carryover=VALUE]`

- `--size=N` — override teammate count per feature team
- `--features=E##-F##,...` — restrict dispatch to specific feature groups
- `--carryover=VALUE` — pass to `shark sprint close` if the user confirms close

## Procedure

1. Read `skills/sprint-execution/SKILL.md` (under this shark skill's directory),
   follow the **team (`/shark-rider run-sprint-team`)** workflow described there, passing any
   remaining arguments through.

## Notes

- Requires `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` in `~/.claude/settings.json`.
- For solo sequential execution, use `/shark-rider run-sprint`.
