---
feature_key: E19-F08-sprint-workflow-yaml-first-class-lifecycle-with-em
epic_key: E19
title: Sprint Workflow YAML — first-class lifecycle with embedded agent routing
description: Add sprint.yaml to embedded workflow defaults so sprint lifecycle is a first-class workflow entity with spawn_agent routing, matching the pattern used by tasks, bugs, features, and other entities.
size: S
---

# Sprint Workflow YAML — first-class lifecycle with embedded agent routing

**Feature Key**: E19-F08

---

## Goal

Sprint is the only entity type without an embedded `workflow.yaml`. The three sprint skills (`sprint-planning`, `sprint-execution`, `sprint-analytics`) exist but are disconnected from the workflow engine — there's no moment when `shark status advance` triggers the right agent. This feature adds `internal/sharkdata/default_data/workflow/sprint.yaml`, wiring the sprint lifecycle (`planning → active → closing → archived`) to `spawn_agent` actions at each step.

---

## Scope

- Add `internal/sharkdata/default_data/workflow/sprint.yaml` with steps: `planning`, `active`, `closing`, `archived`, `on_hold`
- Phase mapping must match what `SprintService.assignableSprintStatuses()` expects: `planning` → phase `planning`, `active` → phase `execution`
- Add sprint prompt stubs under `internal/sharkdata/default_data/prompts/sprint/` (planning.md, active.md, closing.md) or reference skill workflow files directly
- The hard constraints in `SprintService` (single-active enforcement, deletion guard, carryover) remain in Go — the YAML is additive agent routing only

## Design Decisions

**Remove single-active sprint constraint** — the guard in `StartSprint()` that prevents a second sprint from going active is removed entirely. No flag, no config. Multiple simultaneous active sprints are valid; they represent parallel workstreams. AI agents coordinate through claiming, not sprint exclusivity.

**`shark sprint next` with multiple active sprints** — when >1 sprint is active, convention-first: prompts instruct the agent to specify a sprint key explicitly. Add an optional CLI parameter only if prompt-driven convention proves insufficient in practice.

## Out of Scope

- Moving sprint business invariants (carryover logic, capacity) from `SprintService` to YAML

## Dependencies

- E19-F06 (Sprint Orchestration Skills) — the skills being wired in must be complete

## Acceptance Criteria

- [ ] `shark status advance <sprint-key>` from `planning` triggers sprint-planning agent
- [ ] `shark status advance <sprint-key>` from `active` triggers sprint-execution agent  
- [ ] `shark status advance <sprint-key>` from `closing` triggers sprint-analytics agent
- [ ] `make test` passes with no regressions
- [ ] Embedded workflow test (`cc040_embedded_workflow_test.go`) passes for sprint level

*Last Updated*: 2026-06-26
