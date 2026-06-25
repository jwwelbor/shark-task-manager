# /shark help — Verbs & state-aware next actions

Usage:
```
/shark help            # state-aware: read project state, suggest next actions
/shark help --fast     # static verb list, zero shark calls
```

## `--fast` (static, no shark calls)

Print the three groups verbatim:

```
Getting started:  /shark project-init | product-design | vision "idea"
Day-to-day:       /shark run <key> | triage "desc" | viewer | status | list <key> | get <key>
Maintenance:      /shark update-docs | amend <key> "change" | revalidate <key> | help [--fast]
```

## Bare `/shark help` (state-aware)

Run read-only queries, then propose prioritized next actions:
```bash
shark status                 # overall dashboard
shark task list --blocked    # what's stuck
shark claims                 # active leases (who/what is in-flight)
shark next <key> --preview   # next dispatch step for a given entity (if a key is in context)
```
From the results, suggest concrete next commands — e.g. "3 tasks blocked → review
B-notes", "feature E01-F02 is at `active` with 2 unclaimed tasks → `/shark run E01-F02`",
"no work in progress → `/shark run <epic>` or `/shark vision \"…\"`".

Keep it short: the current state in one or two lines, then 2–4 suggested actions.
