# /shark-rider retro-sprint — Sprint retrospective

Reads a closed sprint's summary and velocity data from shark, synthesizes a
five-section markdown retrospective report with data-driven recommendations.

Usage: `/shark-rider retro-sprint S### [--no-write]`

- `--no-write` — print report to stdout instead of writing to `docs/sprints/`

## Procedure

1. Read `skills/sprint-analytics/SKILL.md` (under this shark skill's directory) and
   follow its procedure, passing any remaining arguments through as that skill's
   arguments.

## Notes

- Sprint must be `completed` or `archived`; the skill exits with a clear message
  if the sprint is still active.
- The five fixed sections are: Outcome, Velocity Context, Carryover Analysis,
  Cycle-Time Highlights, Recommendations.
- Recommendations always cite quantitative thresholds from the sprint data —
  no generic placeholder advice.
