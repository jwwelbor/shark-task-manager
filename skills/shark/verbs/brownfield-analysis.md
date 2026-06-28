# /shark brownfield-analysis — Deep codebase analysis

Analyze an existing (brownfield) codebase and produce enterprise-grade documentation:
architecture, code quality, technical debt, security, migration readiness.

Usage: `/shark brownfield-analysis [path]`

- `[path]` — root of the codebase to analyze (defaults to current directory)

## Procedure

1. Read `skills/brownfield-analysis/SKILL.md` (under this shark skill's directory) and
   follow its procedure, passing any remaining arguments through as that skill's
   arguments.

## Notes

- For a lightweight bootstrap of `docs/architecture/` as part of `project-init`, the
  bundle's `research/workflows/brownfield-analysis.md` is used automatically — you do
  not need to invoke this verb for that flow.
- This verb runs the comprehensive standalone methodology (10 analysis areas, ~10–20
  output documents). Use it for full enterprise handoffs, technical debt audits, or
  migration readiness assessments.
