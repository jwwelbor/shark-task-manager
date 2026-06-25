# Agent Task Execution Pattern (Shark 2.x)

## Overview

How an agent executes a shark-tracked entity. Under Shark 2.x the **dispatch
loop owns the lease and all status transitions** (see `verbs/run.md`). A spawned
agent does its craft and **returns a semantic outcome** — it does **not** claim,
advance, or release on its own when driven by `/shark run`.

## The universal pattern (driven agent)

```bash
# 1. GET: read details, context, and the instruction (parent already claimed)
shark get <key> --json

# 2. WORK: do role-specific work

# 3. SAVE: context and notes for resume/handoff
shark context set <key> --field current_step --value "..."
shark create note <key> "..."

# 4. RETURN: a structured outcome to the parent loop
#    { "outcome": "pass" | "fail" | "blocked", "summary": "...", "note": "..." }
```

The parent then runs `shark status advance <key> --outcome <outcome>` and
releases the lease. **Do not call advance/claim/release yourself** in this mode.

> Running an entity **by hand** (no `/shark run`)? Then you own the lease:
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

## Step 3: Save progress & notes

```bash
shark context set E01-F02-001 --field current_step --value "Completed implementation"
shark context set E01-F02-001 --field completed_steps --value '["Tests","Implementation"]'
shark create note E01-F02-001 "Chose JWT for stateless auth" --type decision
```

## Step 4: Return an outcome

Return — do not print a transition — a structured result:

```
{ "outcome": "pass", "summary": "Implemented POST /api/auth with tests", "note": "endpoint added" }
```

- `pass` — the step's goal is met; parent routes via `outcomes[pass]`.
- `fail` — needs rework; parent routes back via `outcomes[fail]`.
- `blocked` — external blocker; parent routes to the blocked/parking step.

## Handling blockers

Record the reason as a note and return `blocked` — let the parent route it:

```bash
shark create note E01-F02-001 "Missing API spec — cannot define contract" --type blocker
# return { "outcome": "blocked", "summary": "Missing API spec" }
```

Do not hardcode a target status (`ready_for_*` / `in_*`); routing is the
workflow's job via the released outcome.

## Common mistakes

| Mistake | Correct approach |
|---------|-----------------|
| Agent calls `shark status advance` itself under `/shark run` | Return an outcome; the parent advances |
| `shark status advance KEY --status X` | `shark status advance KEY --outcome pass\|fail\|blocked` |
| `shark task next-status KEY` / `set-status KEY X` | `shark status advance` / `shark status set` |
| Hardcoding `ready_for_*` target on block | Return `blocked`; the route decides |
| Skipping `shark get` first | Always read details before working |
| `shark status options KEY` | `shark status transitions KEY` |
| Piping JSON through python/jq | Use `--field`: `shark get KEY --field status` |
