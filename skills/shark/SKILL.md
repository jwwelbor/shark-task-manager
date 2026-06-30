---
name: shark
description: Query and manage project state and drive shark workflows — epics, features, tasks, bugs, change-cards, ideas. Use for ALL project state queries, task lifecycle, status transitions (status set / advance --outcome), claim/release/heartbeat leases, blocking/unblocking, notes/context/related-docs, status dashboards, analytics, creating/updating/deleting entities, and any command starting with "shark". Also the router for `/shark <verb>` and triggers the dispatch loop on `/run <key>`.
---

# /shark — Router

`/shark <verb> [args]` is the single entry point for all shark operations. This
file is a thin **router**: it picks a verb and reads the matching `verbs/<verb>.md`,
which holds the actual procedure. `/run <key>` is an alias for `/shark run <key>`.

## Dispatch algorithm

1. Take the **first token** of the arguments.
2. If it matches a verb in the allowlist below → `Read verbs/<verb>.md` and follow it,
   passing the remaining tokens as that verb's arguments.
3. Otherwise (a bare entity key, a CLI subcommand like `status`/`list`/`get`, or
   natural-language prose) → `Read verbs/query.md` and follow it with the **full**
   argument string.

The `/run` command and a bare `/shark run` both route to `verbs/run.md`.

## Verb allowlist

| Group | Verb | Purpose |
|-------|------|---------|
| **Getting started** | `project` | Pre-epic setup namespace: bootstrap, brownfield-analysis, product-design |
| | `product-design` | Run the product-design (D01–D14) workflow |
| | `vision` | Turn a one-line idea into a shark epic + kick off its workflow |
| **Day-to-day** | `run` | Drive an entity through its workflow (claim → agent → advance → release) |
| | `triage` | Quick-capture & classify a discovered work item into the right entity |
| | `deep-review` | Multi-angle parallel code review (6 subagents + consolidator); flags: --fix, --comment. Aliases: comprehensive-review, pr-review |
| | `brownfield-analysis` | Deep analysis and documentation of an existing codebase |
| | `viewer` | Launch the web dashboard (`shark web`) |
| | `consult` | Load an agent persona as an advisor and converse inline (read-only by default) |
| | `status` / `list` / `get` | Pass through to the shark CLI (handled by `query`) |
| **Sprint lifecycle** | `plan-sprint` | Scope a sprint: surface backlog, propose assignments, confirm additions |
| | `run-sprint` | Solo sequential pull-loop: drive sprint to completion entity-by-entity |
| | `run-sprint-team` | Team pull-loop: one `/run-agent-team` per feature group, then standalones |
| | `retro-sprint` | Post-close retrospective: five-section report with data-driven recommendations |
| **Maintenance** | `update-docs` | Diff-driven refresh of `docs/architecture/*` |
| | `amend` | Apply a spec change to an entity and rewind it to the right phase |
| | `revalidate` | Audit spec ↔ tasks ↔ status readiness |
| | `help` | State-aware next-actions (`--fast` for a static list) |
| **(default)** | `query` | NL questions + direct CLI passthrough |

## Content bundle retrieval (used by content-referencing verbs)

Several verbs delegate to the project's bundled **skills** or **agents**. Retrieve
that content through shark, not by reading `shark-data/` directly:

```bash
shark skill get <name> [relative-path]
shark agent get <name>
shark skill list --json
shark agent list --json
```

`shark skill get` and `shark agent get` resolve `.sharkconfig.json`'s
`shark_data_path`, layer `overrides/` over disk defaults, and fall back to the
embedded canonical bundle when no `shark-data/` tree exists. Default output
resolves `{{include: ...}}` / `{{augment: ...}}` and strips Markdown
frontmatter. Use `--raw` only when you need exact stored bytes.

`workflow_config` selects the active **workflow graph / status routing** only —
it is NOT the content bundle root. (In this repo `workflow_config` points at an
example workflow tree while bundle content still lives under `shark-data/`.)
When `workflow_config` is absent/empty, the default workflow dir is
`<bundle>/workflow/`; an explicit `workflow_config` always wins.

If a referenced `shark skill get ...` or `shark agent get ...` command fails
because the content is missing, **degrade gracefully** — print a clear
"unavailable / coming soon" message; never hard-fail.

**`shark skill get` and `shark agent get` serve bundle content only** (workflow-injected
craft skills + agents). The deployed sub-skills under `skills/shark/skills/` are read
**in place** — never fetched via `shark skill get`; a verb that delegates to a deployed
sub-skill must `Read skills/<name>/SKILL.md` directly.

## Sub-skills (`skills/shark/skills/`)

AI-orchestration workflows that extend the shark skill. These are not shark CLI
commands; they are multi-agent or structured-analysis procedures invoked via
their own slash commands or from within a verb procedure.

| Sub-skill | Entry point | Purpose |
|-----------|-------------|---------|
| `brownfield-analysis` | `/brownfield-analysis` or `/shark brownfield-analysis` | Deep analysis and documentation of an existing (brownfield) codebase — architecture, business logic, technical debt, security, migration readiness. Read `skills/brownfield-analysis/SKILL.md`. |
| `deep-review` | `/deep-review` or `/shark deep-review` | Multi-angle parallel code review. Six specialist subagents (bugs, removed behavior, contracts, reuse, tests, standards) then a consolidator produces a PASS/FAIL report with Blocker/Non-blocker/Nit triage. Flags: `--fix`, `--comment`. Aliases: `/comprehensive-review`, `/pr-review`. Read `skills/shark/skills/deep-review/SKILL.md`. |
| `triage` | `/triage` or `/shark triage` | Quick-capture and classify a discovered work item (task, feature, bug, tech-debt, change-card, idea, or note) under the right parent. Searches for duplicates first, confirms before creating. Read `skills/triage/SKILL.md`. |
| `sprint-planning` | `/shark plan-sprint` | Mode-aware sprint scoping: reads shark sprint plan + readiness, proposes backlog assignments, confirms with user. Never calls `shark sprint start`. Read `skills/sprint-planning/SKILL.md`. |
| `sprint-execution` | `/shark run-sprint`, `/shark run-sprint-team` | Sprint pull-loop harnesses (solo and team). Delegates per-entity dispatch to `/run` or `/run-agent-team`; gates sprint close on explicit user confirmation. Read `skills/sprint-execution/SKILL.md`. |
| `sprint-analytics` | `/shark retro-sprint` | Post-close retrospective synthesis: reads `shark sprint summary --detailed` + velocity, produces five-section markdown report with quantitative recommendations. Read `skills/sprint-analytics/SKILL.md`. |

**Boundary rule:** Sub-skills live here because **no workflow YAML references them** — `skills:` in
workflow step definitions only cite bundle craft skills (`quality`, `research`, `architecture`, …).
Standalone on-demand procedures (analysis, review, triage) belong here; workflow-injected craft
belongs in the bundle. When adding a new AI-driven workflow, default to a sub-skill if no workflow
YAML will inject it.

## Detailed references

- `verbs/*.md` — the per-verb procedures (read on demand)
- `context/workflow-and-status.md` — 2.x status, outcomes, claim/lease
- `context/entity-crud.md` — create / update / delete patterns
- `context/notes-context-docs.md` — notes, context, related docs
- `HOOKS.md` — optional automation hooks
