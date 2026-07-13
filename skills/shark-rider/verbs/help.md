# /shark-rider help — Verbs & state-aware next actions

Usage:
```
/shark-rider help            # state-aware: read project state, suggest next actions
/shark-rider help --fast     # static verb list, zero shark calls
/shark-rider help commands   # static Shark command reference, zero shark calls
/shark-rider help <verb>     # static help for a known /shark-rider verb
```

## Static help: `--fast` and `commands`

Treat `commands` as a static alias for `--fast`. Do not call Shark state for
either form. Print the verb groups plus the compact command reference below.

```
Getting started:  /shark-rider project bootstrap | product-design | vision "idea"
Day-to-day:       /shark-rider run <key> | triage "desc" | viewer | status | list <key> | get <key>
Sprint:           /shark-rider plan-sprint <key> | run-sprint <key> | run-sprint-team <key> | retro-sprint <key>
Maintenance:      /shark-rider update-docs | amend <key> "change" | revalidate <key> | help [commands|<verb>]

Read:             shark status [key] | shark list [epic] [feature] | shark get <key> | shark view <key> | shark search "query"
Create:           shark create epic|feature|task|bug|change|tech-debt|idea|note ...
Workflow:         shark status advance <key> --outcome pass|fail|blocked | shark status set <key> <status>
Leases:           shark claim <key> | shark heartbeat <key> | shark release <key> [--session SID | --force] | shark claims
Notes:            shark create note <key> "text" --type=comment | shark notes search "query"
Next preview:     shark next <key> --preview
Bundle content:   shark skill get <name> [path] | shark agent get <name>
```

## Specific command help

If the user asks `/shark-rider help <verb>` for a known verb, print a short static
entry. Do not run Shark state calls.

| Verb | Static help |
|------|-------------|
| `project bootstrap` | Bootstrap architecture docs through `shark skill get research workflows/bootstrap.md`. Afterward suggest product design, vision capture, or `/shark-rider run <key>`. |
| `product-design` | Run the bundled product-design D01-D14 methodology through `shark skill get product-design`. |
| `vision` | Turn an idea into a Shark epic through the bundled epic-writing workflow, then offer `/shark-rider run <epic-key>`. |
| `run` | Drive an epic, feature, task, bug, change-card, or tech-debt item through its workflow. Use `/shark-rider run <key>`. |
| `triage` | Capture, classify, confirm, create, and stop. Use `/shark-rider triage "thing to track"`. |
| `code-review` | Multi-angle review. Flags: `--fix` for safe fixes, `--comment` for GitHub PR comments. |
| `brownfield-analysis` | Deep existing-codebase analysis and documentation. |
| `viewer` | Launch the local web dashboard through `shark web`. |
| `status` | Read project or entity status: `/shark-rider status` or `/shark-rider status <key>`. |
| `list` | List epics, features, tasks, or standalone entity collections: `/shark-rider list`, `/shark-rider list E01`, `/shark-rider list E01 F02`. |
| `get` | Read one entity by key: `/shark-rider get <key>`. |
| `plan-sprint` | Scope a sprint from backlog and readiness data, then ask before adding work. |
| `run-sprint` | Sequential sprint pull-loop using `/shark-rider run` per entity. |
| `run-sprint-team` | Sprint execution grouped by feature, with standalone entities run sequentially. |
| `retro-sprint` | Post-close sprint retrospective from Shark sprint analytics. |
| `update-docs` | Diff-driven refresh of `docs/architecture/*`. |
| `amend` | Apply a spec change to an entity and rewind it to the appropriate phase. |
| `revalidate` | Audit spec, task readiness, and workflow state before more execution. |
| `help` | This help. Use `/shark-rider help commands` for a static reference. |

For unknown verbs, show `/shark-rider help commands` and suggest `/shark-rider <command>`
passthrough for direct CLI queries.

## Bare `/shark-rider help` (state-aware)

Run read-only queries, then propose prioritized next actions:
```bash
shark status                 # overall dashboard
shark task list --blocked    # what's stuck
shark claims                 # active leases (who/what is in-flight)
shark next <key> --preview   # next dispatch step for a given entity (if a key is in context)
```
Only run `shark next <key> --preview` when a concrete key is already in the
conversation or current command context.

From the results, suggest concrete next commands — e.g. "3 tasks blocked → review
B-notes", "feature E01-F02 is at `active` with 2 unclaimed tasks → `/shark-rider run E01-F02`",
"no work in progress → `/shark-rider run <epic>` or `/shark-rider vision \"…\"`".

Convert phase-oriented guidance into Shark terms:
- Missing architecture or product direction: `/shark-rider project bootstrap`,
  `/shark-rider project product-design`, or `/shark-rider vision "idea"`.
- Direction exists but no tracked initiative: `/shark-rider vision "next idea"` or
  `/shark-rider triage "thing to track"`.
- Ready tracked work exists: `/shark-rider run <key>`.
- Blocked work exists: inspect the blocked entity, add a note if needed with
  `shark create note <key> "..." --type=blocker`, or run `/shark-rider revalidate <key>`.
- Work is already claimed: report the claim and suggest waiting, releasing, or
  continuing the claimed key as appropriate.

Keep it short: the current state in one or two lines, then 2–4 suggested actions.
