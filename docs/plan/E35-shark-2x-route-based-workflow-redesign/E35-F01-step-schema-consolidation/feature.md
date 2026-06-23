---
feature_key: E35-F01-step-schema-consolidation
epic_key: E35
title: Step-schema consolidation
description: Merge status_flow + status_metadata into one per-step block. New Step struct with phase/color/progress_weight/responsibility/action/agent/prompt/outcomes/aliases/parking/terminal. instruction_template -> prompt; start: at top. Drop the two-map split in the loader/schema/parser/multilevel. Decision D5.
size: M
---

# Step-schema consolidation

**Feature Key**: E35-F01

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Design (single source of truth)**: [route-based-workflow-redesign.md](../../route-based-workflow-redesign.md) — D5, §4, §8

---

## Goal

### Problem

Each entity workflow YAML keeps two parallel maps keyed by status name:
`status_flow` (the transition graph) and `status_metadata` (color, phase,
`progress_weight`, `responsibility`, `orchestrator_action`). A reader must
cross-reference both to understand one step, and metadata lives apart from the
thing it describes. This split is also what the rest of the redesign needs gone
before `outcomes:` routing (F02) can replace `status_flow`.

### Solution

Collapse both maps into **one block per step** (design §4, D5). Introduce a
`Step` struct that carries everything for a status in one place:

```yaml
version: "1.0"
start: draft
steps:
  qa:
    phase: qa
    color: cyan
    progress_weight: 0.85
    responsibility: agent
    action: spawn_agent
    agent: qa
    provider: anthropic
    model: sonnet
    skills: [quality]
    prompt: feature/qa.md          # was instruction_template
    aliases: [ready_for_qa, in_qa] # placeholder field; consumed in F02/F05
    outcomes: { pass: approval }   # placeholder field; routing logic lands in F02
  completed:
    phase: done
    progress_weight: 1.0
    terminal: true                 # was special_statuses._complete_
```

This feature delivers the **schema and loader** changes only: the merged `Step`
struct, `start:` at the top, `instruction_template` → `prompt`, and the
`terminal`/`parking` flags. The `outcomes:` and `aliases:` fields are parsed and
preserved here but their *behavior* (routing, migration) is owned by F02 and F05.

### Impact

- One block per step; no `status_flow`/`status_metadata` cross-referencing.
- Workflow YAML is the foundation the rest of E35 builds on.
- Metadata co-located with its step — easier to read, review, and diff.

---

## Scope

- New `Step` struct in `internal/config/workflow/schema.go` with fields:
  `phase`, `color`, `progress_weight`, `responsibility`, `action`, `agent`,
  `provider`, `model`, `skills`, `prompt`, `outcomes`, `aliases`, `parking`,
  `terminal`.
- Top-level `start:` key naming the entry step.
- `yaml_loader.go` / `parser.go` / `multilevel.go`: parse the merged form; drop
  the `status_flow` + `status_metadata` two-map parsing path.
- Rename `instruction_template` → `prompt` (loader-level).
- Map the old `special_statuses._complete_` concept onto `terminal: true`.
- Update canonical workflow YAMLs under `internal/sharkdata/default_data/workflow/`
  to the merged form (one block per step) for whatever workflows already exist in
  YAML, so the loader has real fixtures.

### Out of Scope

- Routing behavior / `release(outcome)` — F02.
- Claim/lease fields and persistence — F03.
- Index-file loading and bundle resolution — F04.
- Status migration via `aliases:` — F05 (this feature only *parses* the field).

---

## Acceptance Criteria

1. The loader parses the merged per-step block; `status_flow` and
   `status_metadata` are no longer recognized keys (loading the old two-map form
   is a clear error, not silently ignored).
2. `start:` resolves to a defined step; an undefined `start` target is a load error.
3. `terminal: true` steps carry no `outcomes` and are accepted; non-terminal steps
   need not yet validate `outcomes` content (F02 adds that).
4. `prompt:` replaces `instruction_template:`; both the struct field and canonical
   YAMLs use `prompt`.
5. Canonical workflow YAMLs load cleanly in the merged form.
6. No hardcoded status names introduced — phases/steps come from YAML only.

---

## Verification

- `go test ./internal/config/workflow/...` covers merged-form parsing, `start:`
  resolution, `terminal` handling, and rejection of the two-map form.
- `make fmt && make lint && make test` pass.
- A canonical YAML round-trips: load → in-memory `Step` map matches the file.

---

## Dependencies

- **Blocks**: F02 (outcomes routing builds on the merged `Step`), F05 (aliases).
- **Blocked by**: none. Foundation for the epic.

---

*Last Updated*: 2026-06-23
