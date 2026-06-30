# /shark deep-review — Multi-angle parallel code review

Run a structured code review using six specialist subagents in parallel, then
consolidate into a PASS / PASS-with-triage / FAIL report with Blocker /
Non-blocker / Nit triage.

Usage: `/shark deep-review [<effort>] [--fix] [--comment] [<target>]`

Aliases: `/shark comprehensive-review`, `/shark pr-review`

- `<effort>` — `low` | `medium` | `high` | `xhigh` | `max` (default: inherits session effort)
  - `low` / `medium` — precision-biased: fewer findings, higher confidence only
  - `high` — recall-biased: broader coverage, surfaces lower-confidence issues; **recommended for pre-merge gates**
  - `xhigh` / `max` — maximum depth; use for large or high-risk diffs
- `--fix` — apply safe one-liner fixes automatically
- `--comment` — post findings as inline GitHub PR comments
- `<target>` — file, path, or PR reference (defaults to current diff)

## Procedure

1. Read `skills/shark/skills/deep-review/SKILL.md` and follow its procedure,
   passing any remaining arguments through as that skill's arguments.
