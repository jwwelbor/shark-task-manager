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
| **Getting started** | `project-init` | Bootstrap architecture docs for a new/brownfield project |
| | `product-design` | Run the product-design (D01–D14) workflow |
| | `vision` | Turn a one-line idea into a shark epic + kick off its workflow |
| **Day-to-day** | `run` | Drive an entity through its workflow (claim → agent → advance → release) |
| | `triage` | Quick-capture & classify a discovered work item into the right entity |
| | `viewer` | Launch the web dashboard (`shark web`) |
| | `status` / `list` / `get` | Pass through to the shark CLI (handled by `query`) |
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

## Detailed references

- `verbs/*.md` — the per-verb procedures (read on demand)
- `context/workflow-and-status.md` — 2.x status, outcomes, claim/lease
- `context/entity-crud.md` — create / update / delete patterns
- `context/notes-context-docs.md` — notes, context, related docs
- `HOOKS.md` — optional automation hooks
