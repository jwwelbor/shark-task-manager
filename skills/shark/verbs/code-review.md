# /shark code-review — Multi-angle parallel code review

Run a structured code review using six specialist subagents in parallel, then
consolidate into a PASS / PASS-with-triage / FAIL report with Blocker /
Non-blocker / Nit triage.

Usage: `/shark code-review [--fix] [--comment] [<target>]`

- `--fix` — apply safe one-liner fixes automatically
- `--comment` — post findings as inline GitHub PR comments
- `<target>` — file, path, or PR reference (defaults to current diff)

## Procedure

1. Read `skills/code-review/SKILL.md` (under this shark skill's directory) and
   follow its procedure, passing any remaining arguments through as that skill's
   arguments.
