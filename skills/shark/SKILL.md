---
name: shark
description: Query and manage project state and drive shark workflows — epics, features, tasks, bugs, change-cards, ideas. Use for ALL project state queries, task lifecycle, status transitions (status set / advance --outcome), claim/release/heartbeat leases, blocking/unblocking, notes/context/related-docs, status dashboards, analytics, creating/updating/deleting entities, and any command starting with "shark". Also the router for `/shark <verb>` and triggers the dispatch loop on `/run <key>`.
---

# /shark — Router & CLI driver

## What Shark is

**Shark** is a Go CLI + HTTP API that tracks project work in SQLite/Turso — a
hierarchy of **epics → features → tasks**, plus standalone **bugs**,
**change-cards**, **tech-debt**, and **ideas** — and a **route-based workflow
engine** (Shark 2.x): each entity moves through per-entity workflow steps, each
step declares an `outcomes:` map, and a step can dispatch an AI agent with a
fully-rendered prompt. Status is a pure phase; a **claim** is the work lease.

This skill is the **client** that drives that CLI. It turns a request — a
`/shark <verb>` command, a bare `shark …` line, or plain prose — into `shark`
CLI calls and, where a recipe calls for it, spawned AI agents. This file is a
thin **router**: it picks a verb and reads `verbs/<verb>.md` for the procedure.
`/run <key>` is an alias for `/shark run <key>`.

## The boundary: CLI owns, skill drives

Everything downstream depends on getting this split right.

```
┌─────────────────────────── shark CLI (owns) ───────────────────────────┐
│ Data plane      entities, status, leases, notes, context, docs, search  │
│ Workflow engine routing + prompt assembly: shark next renders the step   │
│                 prompt, inlines skill content, resolves the agent persona │
│                 from the shark-data bundle                               │
└──────────────────────────────────────────────────────────────────────────┘
             ▲ CLI calls                         │ response.prompt (verbatim)
             │                                    ▼
┌─────────────────────────── this skill (drives) ─────────────────────────┐
│ Outer loop      claim → spawn host agent with the prompt → advance → release │
│ Translation     NL / verbs → shark data commands, then summarize          │
│ Local recipes   multi-agent / craft procedures the CLI does not provide   │
└──────────────────────────────────────────────────────────────────────────┘
```

**The golden invariant (for driving workflows):** `shark next <key> --json` is
the **only** dispatch API. Its `response.prompt` already contains the rendered
workflow prompt, inlined `{{include:}}` skill content, and the Shark specialist
persona. Pass it to a host agent **unchanged**.

- ❌ Do **not** build prompts from `shark get … orchestrator_action` (that is an
  inspection surface, not the dispatch API).
- ❌ Do **not** load Shark skills or specialist personas from the host
  filesystem, and do **not** set the host `subagent_type` to a Shark persona
  name (e.g. `business-analyst`) — those personas live in `shark-data/agents/`,
  not the host's native registry. The persona is already inside the prompt; use
  a host-safe primitive (Claude Code: `general-purpose`).

> Recorded failure (2026-07-04): a run spawned `orchestrator_action.agent_type`
> directly as a subagent, bypassing `shark next`, and failed because the persona
> wasn't in the host registry. Repoint to `shark next` and dispatch its prompt.
> See `docs/architecture/shark-dispatch-prompt-assembly.md`.

## Three ways the skill uses the CLI

Every verb is one of three execution modes. Recognizing the mode tells you how
much the skill does vs how much the CLI does:

| Mode | The skill's job | The CLI's job |
|------|-----------------|---------------|
| **1 · Data-plane passthrough** | Translate the request to a `shark` data command, run it, report | Do the read/write |
| **2 · Engine dispatch** | Run a mechanical loop; pass `response.prompt` unchanged | Route the workflow **and** assemble the prompt (`shark next`) |
| **3 · Local AI recipe** | Run a multi-agent / craft procedure (read the verb or sub-skill) | Serve data reads/writes around the recipe |

Notation: a leading **`/`** (`/shark run`) is a **procedure you read and
perform**; a bare **`shark …`** in a code fence is a **command you execute**.
There is no `shark run-sprint` binary — those are skill verbs.

---

### Mode 1 — Data-plane passthrough (`query` default + direct CLI)

Translate to a read-only or mutating `shark` command and report. No agents. Full
patterns in the `context/*.md` references.

| Capability | Recipe |
|-----------|--------|
| Read state | `shark status [key]` · `shark list [epic] [feature]` · `shark get <key> [--field f]` · `shark view <key>` · `shark search "q"` · `shark claims` |
| Status & leases by hand | `shark status advance <key> --outcome pass\|fail\|blocked` · `shark status set <key> <status> [--force]` · `shark status transitions\|history <key>` · `shark claim\|heartbeat\|release <key>` → `context/workflow-and-status.md` |
| Entity CRUD | `shark create epic\|feature\|task\|bug\|change\|idea …` · doc already on disk? add `--key=<KEY> --file=<path>` to link it (never a tree sync for one entity) · after create, fill any shark-generated placeholder file with available context · `shark update <key> …` (no `--status`) · `shark delete <key>` · `shark link <a> <b> --type=…` → `context/entity-crud.md` |
| Notes · context · docs | `shark create note <key> "…" --type=…` · `shark context set <key> --field … --value …` · `shark related-docs add\|list …` → `context/notes-context-docs.md` |
| Workflow inspection | `/shark workflow [entity-type\|entity-key]` → show compact workflow by default; for concrete keys, read current status and transitions first → `verbs/workflow.md` |
| Web dashboard | `/shark viewer` → `shark web` → `verbs/viewer.md` |

Prefer `--field` for single values; never pipe JSON through `head`/`grep`/`jq`.
NL prose routes here via `verbs/query.md`. If the user wants to *drive* an entity
(not just read/set it), send them to Mode 2.

### Mode 2 — Engine dispatch (`/shark run`, sprint pull-loops)

The CLI owns routing **and** prompt assembly; the skill is the outer loop. Never
reconstruct the prompt (see the golden invariant above).

**`/shark run <key>`** (alias `/run`) — the core loop (`verbs/run.md`):

```
loop:
  shark next <key> --json                     # → response.{action, entity_key, prompt}
  shark claim <response.entity_key> --by "$CLAUDE_SID" --field session_id
  spawn host agent with response.prompt        # general-purpose; prompt verbatim
  # worker returns { outcome: pass|fail|blocked, note }
  shark status advance <response.entity_key> --outcome <outcome>
  shark release <response.entity_key> --session "$SID"   # always, even on failure
```

`action` may be `pause` / `archive` / `error` → report and stop. The parent owns
the lease and every transition; the child never claims/advances/releases. See
`context/task-execution-pattern.md` for the spawned agent's contract.

**Sprint pull-loops** wrap Mode 2 across many entities:

| Verb | Recipe |
|------|--------|
| `/shark run-sprint S###` | Read `skills/sprint-execution/SKILL.md`; `shark sprint next` → `/run` per entity; gate close on user confirmation → `verbs/run-sprint.md` |
| `/shark run-sprint-team S###` | Same sub-skill; group by feature → `/run-agent-team` per group, standalones via `/run` → `verbs/run-sprint-team.md` |

### Mode 3 — Local AI recipes (verb / sub-skill + CLI around it)

Procedures the CLI doesn't provide. Read the verb (and any sub-skill it names)
and perform it, using `shark` only for the data reads/writes it calls out.

**Getting started**

| Capability | Recipe |
|-----------|--------|
| Idea → epic → workflow | `/shark vision "idea"` → follow `shark skill get specification-writing workflows/write-epic.md`; fallback `shark create epic`; then offer `/shark run <epic>` → `verbs/vision.md` |
| Pre-epic setup menu | `/shark project [bootstrap\|brownfield-analysis\|product-design]` — independent activities, not a sequence → `verbs/project.md` |
| Bootstrap architecture docs | `/shark project bootstrap` → follow `shark skill get research workflows/bootstrap.md` |
| Analyze existing codebase | `/shark brownfield-analysis [path]` → read `skills/brownfield-analysis/SKILL.md` (multi-agent) → `verbs/brownfield-analysis.md` |
| Product design D01–D14 | `/shark product-design` → `shark skill get product-design` → `verbs/product-design.md` |

**Capture, review & maintain**

| Capability | Recipe |
|-----------|--------|
| Capture & classify an item | `/shark triage "desc"` → read `skills/triage/SKILL.md`; dedup via `shark list <type>` per candidate → confirm → `shark create <type> …` → `verbs/triage.md` |
| Multi-angle code review | `/shark deep-review [effort] [--fix] [--comment] [target]` (aliases `comprehensive-review`, `pr-review`) → read `skills/shark/skills/deep-review/SKILL.md`; 6 subagents + consolidator → PASS/FAIL triage → `verbs/deep-review.md` |
| Spec ↔ tasks ↔ status audit | `/shark revalidate <key>` → inline audit from `shark get`/`shark list`; optional `shark skill get quality workflows/validate-*.md` → READY/WARNINGS/NOT READY → `verbs/revalidate.md` |
| Apply spec change & rewind | `/shark amend <key> "change"` → edit spec → `shark create note <key> "Amended: …" --type=requirement` → resolve target from workflow YAML → `shark status set <key> <target> --force` → `verbs/amend.md` |
| Refresh architecture docs | `/shark update-docs` → diff-driven refresh of `docs/architecture/*` → `verbs/update-docs.md` |
| Reconcile filesystem → shark | `/shark sync <epic-key>` → bulk-sync one epic folder's docs into shark entities (filesystem is source of truth). **Explicit user invocation only**; for one authored doc use `shark create … --key --file` instead → `verbs/sync.md` |

**Plan & advise**

| Capability | Recipe |
|-----------|--------|
| Scope a sprint | `/shark plan-sprint S###` → read `skills/sprint-planning/SKILL.md`; reads `shark sprint plan` + readiness, proposes, confirms. Never `shark sprint start` → `verbs/plan-sprint.md` |
| Sprint retrospective | `/shark retro-sprint S###` → read `skills/sprint-analytics/SKILL.md`; `shark sprint summary --detailed` + velocity → five-section report → `verbs/retro-sprint.md` |
| Consult an agent persona | `/shark consult <agent> [referent]` → `shark agent list --json` (resolve) → `shark agent get <agent>` → adopt persona inline, read-only → `verbs/consult.md` |
| Inspect workflow/status flow | `/shark workflow [entity-type\|entity-key] [--all\|--json]` → read status for keys, then render `shark admin workflow list` → `verbs/workflow.md` |
| State-aware next actions | `/shark help` → `shark status` + `shark task list --blocked` + `shark claims` → propose 2–4 next commands. `--fast`/`commands` = static, no CLI → `verbs/help.md` |

## Golden path

The end-to-end lifecycle (run only the steps you need):

```
/shark project bootstrap          # 1. one-time: architecture docs for a new/brownfield repo
/shark product-design             #    optional: product design D01–D14
/shark vision "one-line idea"     # 2. idea → epic + spec
/shark run <epic-key>             # 3. drive it: refinement → features → tasks → review
/shark triage "found a bug"       #    anytime: capture & classify discovered work
/shark deep-review high           # 4. pre-merge quality gate on the diff
/shark help                       #    anytime: where am I, what's next
```

## Inspecting a workflow

Shark 2.x workflows are config-driven YAML — the step list per entity is **not**
fixed here; derive it live rather than hardcoding it:

```bash
shark status transitions <key>    # valid next statuses / outcomes from the current step
shark status transitions <key> --json
shark next <key> --preview        # what the engine would dispatch next (no claim/spawn)
```

## Dispatch algorithm

1. Take the **first token** of the arguments.
2. If it matches a recognized verb → `Read verbs/<verb>.md` and follow it with the
   remaining tokens.
3. Otherwise (a bare entity key, a CLI subcommand like `status`/`list`/`get`, or
   NL prose) → `Read verbs/query.md` with the **full** argument string.

Recognized verbs: `project`, `product-design`, `vision`, `run`, `triage`,
`deep-review` (= `comprehensive-review` / `pr-review`), `brownfield-analysis`,
`viewer`, `consult`, `workflow`, `plan-sprint`, `run-sprint`, `run-sprint-team`,
`retro-sprint`, `sync` (explicit user invocation only), `update-docs`, `amend`,
`revalidate`, `help`. Everything else (including `status` / `list` / `get`)
falls through to `query`. `/run` and `/shark run` both route to `verbs/run.md`.

## Content bundle retrieval (used by Mode 3 verbs)

Mode-3 verbs that delegate to the project's bundled **skills** or **agents**
retrieve that content through shark, not by reading `shark-data/` directly:

```bash
shark skill get <name> [relative-path]     # a bundle skill / workflow file
shark agent get <name>                     # a bundle agent persona
shark skill list --json                    # what's available
shark agent list --json
```

If a `shark skill get` / `shark agent get` fails because content is missing,
**degrade gracefully** — print a clear "unavailable" message; never hard-fail.
(Exception: the shark **sub-skills** below are host-local — `Read` them directly,
never via `shark skill get`.)

## Sub-skills (`shark/skills/`)

Host-local AI-orchestration procedures that Mode-3 verbs read directly.

| Sub-skill | Entry point | Purpose |
|-----------|-------------|---------|
| `brownfield-analysis` | `/brownfield-analysis` or `/shark brownfield-analysis` | Deep analysis and documentation of an existing codebase — architecture, business logic, technical debt, security, migration readiness. Read `skills/brownfield-analysis/SKILL.md`. |
| `deep-review` | `/deep-review` or `/shark deep-review` | Multi-angle parallel code review. Six specialist subagents (bugs, removed behavior, contracts, reuse, tests, standards) then a consolidator produce a PASS/FAIL report with Blocker/Non-blocker/Nit triage. Flags: `--fix`, `--comment`. Aliases: `/comprehensive-review`, `/pr-review`. Read `shark/skills/deep-review/SKILL.md`. |
| `triage` | `/triage` or `/shark triage` | Quick-capture and classify a discovered work item under the right parent. Dedups first, confirms before creating. Read `skills/triage/SKILL.md`. |
| `sprint-planning` | `/shark plan-sprint` | Mode-aware sprint scoping: reads sprint plan + readiness, proposes assignments, confirms. Never calls `shark sprint start`. Read `skills/sprint-planning/SKILL.md`. |
| `sprint-execution` | `/shark run-sprint`, `/shark run-sprint-team` | Sprint pull-loop harnesses (solo and team). Delegate per-entity dispatch to `/run` or `/run-agent-team`; gate close on confirmation. Read `skills/sprint-execution/SKILL.md`. |
| `sprint-analytics` | `/shark retro-sprint` | Post-close retrospective from `shark sprint summary --detailed` + velocity → five-section report. Read `skills/sprint-analytics/SKILL.md`. |

## Detailed references

- `verbs/*.md` — the per-verb procedures (read on demand)
- `verbs/workflow.md` — workflow inspection wrapper for entity types and keys
- `context/workflow-and-status.md` — 2.x status, outcomes, claim/lease
- `context/entity-crud.md` — create / update / delete patterns
- `context/notes-context-docs.md` — notes, context, related docs
- `context/task-execution-pattern.md` — how a spawned agent executes one entity
- `docs/architecture/shark-dispatch-prompt-assembly.md` — the `shark next` dispatch contract
- `HOOKS.md` — optional automation hooks
