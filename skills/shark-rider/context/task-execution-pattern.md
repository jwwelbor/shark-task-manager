# Agent Task Execution Pattern (Shark 2.x)

## Overview

How an agent executes a shark-tracked entity. Under Shark 2.x the **dispatch
loop owns the lease and all status transitions** (see `verbs/run.md`). A
Rider-dispatched worker does its craft and **returns a semantic outcome plus
bounded evidence and parent-persistence directives**. It does not mutate lease
or workflow state for the dispatched entity: it does **not** claim, heartbeat,
release, advance, or status-set on its own when driven by `/shark-rider run`.
The worker may still write bounded notes and context for its dispatched entity
as additive handoff evidence. The parent persists those directives after the
worker returns.

## The universal pattern (driven agent)

```bash
# 1. GET: read details, context, and the instruction (parent already claimed)
shark get <key> --json

# 2. WORK: do role-specific work

# 3. RETURN: bounded evidence and directives for the parent loop to persist
#    {
#      "outcome": "pass" | "fail" | "blocked",
#      "summary": "...",
#      "evidence": ["tests: make test", "docs: docs/design.md"],
#      "note": "a concise handoff or decision"
#    }
```

The parent persists those directives, runs
`shark status advance <key> --outcome <outcome>`, and releases the lease. **Do
not call lease or workflow-transition commands yourself** in this mode. For the
dispatched entity, the worker never claims, heartbeats, releases, or
transitions; those mutations remain parent-loop responsibilities. The allowed
worker writes in this mode are bounded notes and context that capture progress,
handoff evidence, or a blocker for the parent to interpret.

> Running an entity **by hand** (no `/shark-rider run`)? Then you own the lease:
> `shark claim` → work → `shark status advance <key> --outcome <…>` → `shark release`.

## Step 1: Get entity & read context

```bash
shark get E01-F02-001 --json
```

Extract: `title`, `description`, `status`, `file_path` (read the spec for
acceptance criteria), `depends_on` / `dependency_status`, and
`orchestrator_action.instruction` — **follow it if present**.

Load supporting context:

```bash
shark related-docs list --task=E01-F02-001
shark context E01-F02-001 --json
shark task notes E01-F02-001
```

## Step 2: Do role-specific work

- **Researcher / BA**: clarify scope, define acceptance criteria
- **Architect**: design solution, write the spec
- **Developer**: tests first (TDD), implement, validate
- **Tech Lead / reviewer**: review code, check coverage
- **QA / UAT**: run tests, check acceptance criteria, red-team

## Step 3: Return bounded evidence and parent-persistence directives

Write only the smallest Shark evidence the parent loop needs for the dispatched
entity. Allowed writes are bounded notes and context for progress, decisions,
handoff evidence, or blockers. Return only the concise information the parent
needs to persist and route the work:

- A summary and bounded evidence, such as a test command and relevant artifact
  paths.
- A `PARENT NOTE: <text>` or `COMPLEXITY NOTE: <text>` directive when the
  parent should record a decision, handoff, or gate result.
- A blocker reason with `RECOMMENDED OUTCOME: blocked` when work cannot
  continue. The parent records the blocker and applies the configured route.
- Any task kickback in the exact `<task-id> -> <status> --reason "<why>"`
  format when the workflow calls for it.

The parent validates and persists these directives through the parent-owned
Rider loop. Keep all evidence and directive text bounded; do not return
credentials, rendered prompts, unrestricted output, or unrelated paths.

## Step 4: Return an outcome

Return — do not print a transition or issue a Shark mutation — a structured
result:

```
{
  "outcome": "pass",
  "summary": "Implemented POST /api/auth with tests",
  "evidence": ["make test", "internal/api/auth.go"],
  "note": "endpoint added"
}
```

- `pass` — the step's goal is met; parent routes via `outcomes[pass]`.
- `fail` — needs rework; parent routes back via `outcomes[fail]`.
- `blocked` — external blocker; parent routes to the blocked/parking step.

## `gate_result_v1` steps (structured gate results)

Everything above is the `legacy` `result_contract` path (the default: free-form
directive lines like `RECOMMENDED OUTCOME: blocked`, parsed leniently by the
parent). A step whose `shark get`/`shark next --json` reports
`result_contract: gate_result_v1` (T-E34-F05-005) instead requires the worker
to return the single canonical worker-control envelope
(`context/worker-control-schema.yaml`) with a nested `gate_result` payload —
no free-form directive lines, no second Rider-only grammar:

```
{
  "kind": "final",
  "recommended_outcome": "pass",
  "evidence": [],
  "gate_result": {
    "schema_version": 1,
    "summary": "..."
  }
}
```

The ENTIRE trimmed response must be this JSON object. The host adapter — not
the worker — then routes it through the shared ingestion CLI surface (see
`context/host-adapter-contract.md`'s "`result_contract`-gated dispatch"
section and `verbs/run.md`); it never parses this envelope as a legacy
directive. A `gate_result_v1` step's worker must not emit the legacy
free-form forms (`RECOMMENDED OUTCOME:`, `PARENT NOTE:`, bare kickback lines)
— the parent fails the step closed rather than silently falling back to
legacy parsing.

## Handling blockers

Return the reason and `blocked` — let the parent persist the blocker and route
it:

```
{
  "outcome": "blocked",
  "summary": "Missing API spec — cannot define contract",
  "note": "Missing API spec — cannot define contract"
}
RECOMMENDED OUTCOME: blocked
```

Do not hardcode a target status (`ready_for_*` / `in_*`); routing is the
workflow's job via the released outcome.

## Common mistakes

| Mistake | Correct approach |
|---------|-----------------|
| Agent calls `shark status advance` itself under `/shark-rider run` | Return an outcome; the parent advances |
| `shark status advance KEY --status X` | `shark status advance KEY --outcome pass\|fail\|blocked` |
| `shark task next-status KEY` / `set-status KEY X` | `shark status advance` / `shark status set` |
| Hardcoding `ready_for_*` target on block | Return `blocked`; the route decides |
| Skipping `shark get` first | Always read details before working |
| `shark status options KEY` | `shark status transitions KEY` |
| Piping JSON through python/jq | Use `--field`: `shark get KEY --field status` |
