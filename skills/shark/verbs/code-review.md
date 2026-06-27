# /shark code-review — Multi-angle parallel code review

Run a structured code review using six specialist subagents in parallel, then
consolidate into a PASS / PASS-with-triage / FAIL report with Blocker /
Non-blocker / Nit triage.

Usage: `/shark code-review [--fix] [--comment] [<target>]`

- `--fix` — apply safe one-liner fixes automatically
- `--comment` — post findings as inline GitHub PR comments
- `<target>` — file, path, or PR reference (defaults to current diff)

## Procedure

1. Run `shark skill get code-review` and follow the returned skill instructions,
   passing any remaining arguments as that skill's arguments.
2. **If the command fails** because the bundle code-review skill is unavailable,
   print a concise unavailable message and stop:
   > `/shark code-review` is not yet available in this project's content bundle
   > (`shark skill get code-review` failed). For now, use the standalone
   > `/code-review` command if installed.

   Do not improvise a review workflow inline — keep degradation honest.
