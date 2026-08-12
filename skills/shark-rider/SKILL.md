---
name: shark-rider
description: Shark Rider coordinates host-side project setup and workflow actions around the Shark CLI. Use `/shark-rider <action>` for Rider actions and `shark <command>` for the CLI; it covers project bootstrap, product and brownfield coordination, entity dispatch, review, planning, and related host workflows.
---

# Shark Rider — Router and host-side coordinator

## What Shark is

**Shark** is a Go CLI + HTTP API that tracks project work in SQLite/Turso — a
hierarchy of **epics → features → tasks**, plus standalone **bugs**,
**change-cards**, **tech-debt**, and **ideas** — and a **route-based workflow
engine** (Shark 2.x): each entity moves through per-entity workflow steps, each
step declares an `outcomes:` map, and a step can dispatch an AI agent with a
fully-rendered prompt. Status is a pure phase; a **claim** is the work lease.

**Shark Rider** is the host-side coordination layer around that CLI. It turns a
`/shark-rider <action>` request or plain prose into the owning Rider action. A
bare `shark …` line remains a CLI command and is never interpreted as a Rider
action. This file is a thin router: it picks an action and reads
`verbs/<action>.md` for the procedure.
`/run <key>` is an alias for `/shark-rider run <key>`.

## The boundary: CLI owns, Rider drives

Everything downstream depends on getting this split right.

```
┌─────────────────────────── shark CLI (owns) ──────────────────────────────┐
│ Data plane      entities, status, leases, notes, context, docs, search    │
│ Keyed routing + prompt assembly: shark next <key> renders the step prompt │
│                 inlines skill content, and resolves the agent persona      │
│                 from the shark-data bundle                                 │
└───────────────────────────────────────────────────────────────────────────┘
             ▲ CLI calls                         │ response.prompt (verbatim)
             │                                    ▼
┌─────────────────────────── Shark Rider (drives) ─────────────────────────────┐
│ Outer loop      claim → spawn host agent with the prompt → advance → release  │
│ Translation     NL / verbs → shark data commands, then summarize              │
│ Local recipes   multi-agent / craft procedures the CLI does not provide       │
└───────────────────────────────────────────────────────────────────────────────┘
```

Shark separates work **selection** from keyed **dispatch**:

- `shark plan [root|collection]` selects work and never dispatches. Bare
  `shark plan` selects one epic or an epic-only `parallel_candidates` tie;
  `shark plan <epic|feature>` evaluates one hierarchy edge and returns direct
  children as a selection; `shark plan bugs|change-cards|tech-debt` selects the
  next claimable standalone tier. None of these claim, advance, or spawn an
  agent. `/shark-rider plan [root|collection]` reads its response and
  recommends an execution shape.
- `shark next <key> --json` resolves keyed workflow dispatch for one concrete
  entity, cascading internally to the first dispatchable descendant.
  `/shark-rider run <key>` is the explicit handoff from a selected key to
  keyed dispatch.

**The golden invariant applies only to keyed dispatch:**
`shark next <key> --json` is the only keyed dispatch API. Its `response.prompt`
already contains the rendered workflow prompt, inlined `{{include:}}` skill
content, and the Shark specialist persona. Pass `response.prompt` to the host
agent unchanged.

- ❌ Do **not** build prompts from `shark get … orchestrator_action` (that is an
  inspection surface, not the dispatch API).
- ❌ Do **not** load Shark skills or specialist personas from the host
  filesystem, and do **not** set the host `subagent_type` to a Shark persona
  name (e.g. `business-analyst`) — those personas live in `shark-data/agents/`,
  not the host's native registry. The persona is already inside the prompt; use
  a host-safe primitive (Claude Code: `general-purpose`).

> Recorded failure (2026-07-04): a run spawned `orchestrator_action.agent_type`
> directly as a subagent, bypassing keyed `shark next <key> --json`, and failed
> because the persona wasn't in the host registry. Repoint to keyed
> `shark next <key> --json` and dispatch its prompt.
> See `docs/architecture/shark-rider-dispatch-prompt-assembly.md`.

## Three ways Rider uses the CLI

Every verb is one of three execution modes. Recognizing the mode tells you how
much the skill does vs how much the CLI does:

| Mode | The skill's job | The CLI's job |
|------|-----------------|---------------|
| **1 · Data-plane passthrough** | Translate the request to a `shark` data command, run it, report | Execute data commands; `shark plan` selection remains read-only |
| **2 · Engine dispatch** | Run a mechanical loop; pass `response.prompt` unchanged | Route the workflow **and** assemble the prompt (`shark next <key>`) |
| **3 · Local AI recipe** | Run a host-side craft procedure (read the action or sub-skill) | Serve data reads/writes around the recipe |

Notation: a leading **`/`** (`/shark-rider run`) is a **procedure you read and
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
| Workflow inspection | `/shark-rider workflow [entity-type\|entity-key]` → show compact workflow by default; for concrete keys, read current status and transitions first → `verbs/workflow.md` |
| Web dashboard | `/shark-rider viewer` → `shark web` → `verbs/viewer.md` |

Prefer `--field` for single values; never pipe JSON through `head`/`grep`/`jq`.
NL prose routes here via `verbs/query.md`. If the user wants to *drive* an entity
(not just read/set it), send them to Mode 2.

### Mode 2 — Engine dispatch (`/shark-rider run`, sprint pull-loops)

The CLI owns routing **and** prompt assembly; the skill is the outer loop. Never
reconstruct the prompt (see the golden invariant above).

**`/shark-rider run <key>`** (alias `/run <key>`) — the core loop (`verbs/run.md`):

```
loop:
  shark next <key> --json                     # → response.{action, entity_key, prompt}
  shark claim <response.entity_key> --by "$CLAUDE_SID" --field session_id
  spawn host agent with response.prompt        # general-purpose; prompt verbatim
  # worker returns { outcome: pass|fail|blocked, note }
  shark status advance <response.entity_key> --outcome <outcome> \
    --session "$SID" --from-status <response.status>
  shark release <response.entity_key> --session "$SID"   # always, even on failure
```

`action` may be `pause` / `archive` / `error` → report and stop. The parent owns
the lease and every transition; the child never claims/advances/releases. See
`context/task-execution-pattern.md` for the spawned agent's contract.

**Sprint pull-loops** wrap Mode 2 across many entities:

| Verb | Recipe |
|------|--------|
| `/shark-rider run-sprint S###` | Read `skills/sprint-execution/SKILL.md`; `shark sprint next` → `/shark-rider run` per entity; gate close on user confirmation → `verbs/run-sprint.md` |
| `/shark-rider run-agent-team <epic\|feature>` | Confirm host topology prerequisites, then delegate to the canonical Shark Attack topology adapter → `verbs/run-agent-team.md` |
| `/shark-rider run-sprint-team S###` | Thin alias for `/shark-rider run-agent-team --sprint S###`; the owner retains the sprint close gate → `verbs/run-sprint-team.md` |

### Mode 3 — Local AI recipes (verb / sub-skill + CLI around it)

Procedures the CLI doesn't provide. Read the verb (and any sub-skill it names)
and perform it, using `shark` only for the data reads/writes it calls out.

**Getting started**

| Capability | Recipe |
|-----------|--------|
| Idea → epic → workflow | `/shark-rider vision "idea"` → follow `shark skill get specification-writing workflows/write-epic.md`; fallback `shark create epic`; then offer `/shark-rider run <epic>` → `verbs/vision.md` |
| Source document → epic portfolio | `/shark-rider breakdown <docs-path> [--output=<docs-path>]` → derive intrinsic epic scale from outcomes and acceptance, test merges, optionally compare genuine precedents, then propose the exact charter-ready epic delta; confirm, create, and verify approved epics in the same interaction; leave feature decomposition to each epic's Shark workflow → `verbs/breakdown.md` |
| Progress-driven project setup | `/shark-rider project bootstrap` → prepare Shark, seed progress, coordinate discovery, and integrate architecture → `verbs/project.md` |
| Product design | `/shark-rider project product-design` → owning action retrieves the `product-design` bundle and checkpoints each artifact → `verbs/product-design.md` |
| Brownfield analysis | `/shark-rider project brownfield-analysis` → owning action runs the selected analysis depth and checkpoints outputs → `verbs/brownfield-analysis.md` |

**Capture, review & maintain**

| Capability | Recipe |
|-----------|--------|
| Capture & classify an item | `/shark-rider triage "desc"` → read `skills/triage/SKILL.md`; dedup via `shark list <type>` per candidate → confirm → `shark create <type> …` → `verbs/triage.md` |
| Multi-angle code review | `/shark-rider deep-review [effort] [--fix] [--comment] [target]` (aliases `comprehensive-review`, `pr-review`) → read `skills/shark-rider/skills/deep-review/SKILL.md`; 6 subagents + consolidator → PASS/FAIL triage → `verbs/deep-review.md` |
| Spec ↔ tasks ↔ status audit | `/shark-rider revalidate <key>` → inline audit from `shark get`/`shark list`; optional `shark skill get quality workflows/validate-*.md` → READY/WARNINGS/NOT READY → `verbs/revalidate.md` |
| Evidence-based demo preparation | `/shark-rider demo <epic-key|feature-key> [--draft]` → collect documented state and linked guidance, retrieve `demo-script`, and create a traceable demo artifact without becoming a UAT gate → `verbs/demo.md` |
| Solution decision walkthrough | `/shark-rider walkthrough <target> [scope]` (entity key or `docs/` path) → collect authoritative Shark-linked or document-first context, retrieve `solution-walkthrough`, then resolve or ratify decisions one at a time → `verbs/walkthrough.md` |
| Apply spec change & rewind | `/shark-rider amend <key> "change"` → edit spec → `shark create note <key> "Amended: …" --type=requirement` → resolve target from workflow YAML → `shark status set <key> <target> --force` → `verbs/amend.md` |
| Refresh architecture docs | `/shark-rider update-docs` → diff-driven refresh of `docs/architecture/*` → `verbs/update-docs.md` |
| Reconcile filesystem → shark | `/shark-rider sync <epic-key>` → bulk-sync one epic folder's docs into shark entities (filesystem is source of truth). **Explicit user invocation only**; for one authored doc use `shark create … --key --file` instead → `verbs/sync.md` |

**Plan & advise**

| Capability | Recipe |
|-----------|--------|
| Scope a sprint | `/shark-rider plan-sprint S###` → read `skills/sprint-planning/SKILL.md`; reads `shark sprint plan` + readiness, proposes, confirms. Never `shark sprint start` → `verbs/plan-sprint.md` |
| Sprint retrospective | `/shark-rider retro-sprint S###` → read `skills/sprint-analytics/SKILL.md`; `shark sprint summary --detailed` + velocity → five-section report → `verbs/retro-sprint.md` |
| Consult an agent persona | `/shark-rider consult <agent> [referent]` → `shark agent list --json` (resolve) → `shark agent get <agent>` → adopt persona inline, read-only → `verbs/consult.md` |
| Inspect workflow/status flow | `/shark-rider workflow [entity-type\|entity-key] [--all\|--json]` → read status for keys, then render `shark admin workflow list` → `verbs/workflow.md` |
| Select next work | `/shark-rider plan [root\|collection]` → call `shark plan [root\|collection]` once → recommend an execution shape for the returned selection. It does not claim or launch work itself → `verbs/plan.md` |
| State-aware portfolio advice | `/shark-rider help` → call bare `shark plan` once → report the selected epic, evaluate an epic tie through the `/plan` evidence gate, or surface a pause reason. Stop before keyed dispatch. `--fast`/`commands` = static, no CLI → `verbs/help.md` |

## Golden path

The end-to-end lifecycle (run only the steps you need):

```
/shark-rider project bootstrap          # 1. one-time: architecture docs for a new/brownfield repo
/shark-rider project product-design             #    optional: product design D01–D14
/shark-rider vision "one-line idea"     # 2. idea → epic + spec
/shark-rider run <epic-key>             # 3. drive it: refinement → features → tasks → review
/shark-rider triage "found a bug"       #    anytime: capture & classify discovered work
/shark-rider deep-review high           # 4. pre-merge quality gate on the diff
/shark-rider help                       #    anytime: where am I, what's next
```

## Inspecting a workflow

Shark 2.x workflows are config-driven YAML — the step list per entity is **not**
fixed here; derive it live rather than hardcoding it:

```bash
shark status transitions <key>    # valid next statuses / outcomes from the current step
shark status transitions <key> --json
```

Keyed `shark next <key> --json` is the dispatch API, not a read-only inspection
command. It may auto-advance cascade-complete parents or agentless
`advance_status` placeholders while resolving the next dispatch.

## Dispatch algorithm

1. Take the **first token** of the arguments.
2. If it matches a recognized verb → `Read verbs/<verb>.md` and follow it with the
   remaining tokens.
3. Otherwise (a bare entity key, a CLI subcommand like `status`/`list`/`get`, or
   NL prose) → `Read verbs/query.md` with the **full** argument string.

Recognized verbs: `project`, `project-init`, `product-design`, `vision`, `breakdown`, `run`, `plan`, `triage`, `demo`, `walkthrough`,
`deep-review` (= `comprehensive-review` / `pr-review`), `brownfield-analysis`,
`viewer`, `consult`, `workflow`, `plan-sprint`, `run-sprint`, `run-agent-team`, `run-sprint-team`,
`retro-sprint`, `sync` (explicit user invocation only), `update-docs`, `amend`,
`revalidate`, `help`. Everything else (including `status` / `list` / `get`)
falls through to `query`. `/shark-rider run` and `/run` both route to `verbs/run.md`.

## Content bundle retrieval (used by owning Rider actions)

Rider actions that delegate to the project's bundled **skills** or **agents**
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

## Host-local sub-skills (`shark-rider/skills/`)

Host-local AI-orchestration procedures that Mode-3 verbs read directly.

| Sub-skill | Entry point | Purpose |
|-----------|-------------|---------|
| `brownfield-analysis` | `/brownfield-analysis` or `/shark-rider brownfield-analysis` | Deep analysis and documentation of an existing codebase — architecture, business logic, technical debt, security, migration readiness. Read `skills/brownfield-analysis/SKILL.md`. |
| `deep-review` | `/deep-review` or `/shark-rider deep-review` | Multi-angle parallel code review. Six specialist subagents (bugs, removed behavior, contracts, reuse, tests, standards) then a consolidator produce a PASS/FAIL report with Blocker/Non-blocker/Nit triage. Flags: `--fix`, `--comment`. Aliases: `/comprehensive-review`, `/pr-review`. Read `shark-rider/skills/deep-review/SKILL.md`. |
| `triage` | `/triage` or `/shark-rider triage` | Quick-capture and classify a discovered work item under the right parent. Dedups first, confirms before creating. Read `skills/triage/SKILL.md`. |
| `sprint-planning` | `/shark-rider plan-sprint` | Mode-aware sprint scoping: reads sprint plan + readiness, proposes assignments, confirms. Never calls `shark sprint start`. Read `skills/sprint-planning/SKILL.md`. |
| `sprint-execution` | `/shark-rider run-sprint`, `/shark-rider run-sprint-team` | Sprint pull-loop harnesses. The solo loop delegates per-entity dispatch to `/shark-rider run`; the team alias delegates to `/shark-rider run-agent-team --sprint`. The owner gates sprint closure. Read `skills/sprint-execution/SKILL.md`. |
| `sprint-analytics` | `/shark-rider retro-sprint` | Post-close retrospective from `shark sprint summary --detailed` + velocity → five-section report. Read `skills/sprint-analytics/SKILL.md`. |

## Detailed references

- `verbs/*.md` — the per-verb procedures (read on demand)
- `verbs/workflow.md` — workflow inspection wrapper for entity types and keys
- `context/workflow-and-status.md` — 2.x status, outcomes, claim/lease
- `context/entity-crud.md` — create / update / delete patterns
- `context/notes-context-docs.md` — notes, context, related docs
- `context/task-execution-pattern.md` — how a spawned agent executes one entity
- **Shark Dispatch Prompt Assembly** architecture document — the keyed `shark next <key>` dispatch contract
- `HOOKS.md` — optional automation hooks
