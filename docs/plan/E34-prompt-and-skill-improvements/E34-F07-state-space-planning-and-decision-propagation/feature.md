---
feature_key: E34-F07-state-space-planning-and-decision-propagation
epic_key: E34
title: State-Space Planning and Decision Propagation
description: Make planning workflows enumerate lifecycle state spaces, trace producer-consumer interactions beyond direct dependencies, and propagate material decisions into affected specifications, acceptance criteria, and tests.
---

# State-Space Planning and Decision Propagation

**Feature Key**: E34-F07

## Goal

### Problem

Planning gates can accept a prose lifecycle, a single-entity decision table,
or a deferred downstream obligation without proving that every state and
consumer is covered. Later decisions may change the intended behavior while
the affected specification and test plan remain stale. E04's cross-feature
deduplication defect and stale UAT specification findings are concrete results
of these disconnected views.

### Solution

Make state-space and change-impact analysis explicit in specification, test
planning, task review, epic feature review, and decision-resolution workflows.
Behavioral lifecycle fields receive closed transition tables. Multi-entity
axes come from interaction maps and production caller paths, not an arbitrary
foreign-key distance. Material decisions produce one bounded impact set that
updates affected plans now or names linked follow-up work.

### Impact

Planning reviews can prove that lifecycle failures, cross-entity combinations,
existing consumers, names, and ratified decisions remain closed from design to
tests. A state or decision change cannot silently invalidate already shipped
acceptance criteria.

## Research findings

- `feature/specification.md` and `feature/test_planning.md` validate
  requirements and tests but do not require a closed transition table for each
  behavior-bearing lifecycle field.
- The test-design guidance describes state-transition and decision-table
  techniques, but the prompts do not make them conditional obligations when
  the feature introduces a lifecycle or disposition state space.
- E34-F03 already establishes I-## interaction and caller-path concepts. These
  are better dependency-discovery inputs than “one foreign-key hop,” which can
  miss event, service, file, CLI, and cross-epic consumers.
- `tech_debt/resolved.md` records resolution but does not require an impact
  sweep across specs, test plans, tasks, decisions, or shipped consumer ACs.
  E39 Questions provide durable decision provenance that should feed the same
  impact procedure.

## Change-impact contract

The normative **I-04 ChangeImpactSet v1** shape lives in
[Architecture](../architecture.md#i-04-changeimpactset-v1). It identifies the
decision or state change, affected producer and consumer artifacts, shipped
acceptance criteria, required amendments or linked follow-up keys, naming
checks, and verification evidence. Planning gates return it through E34-F05's
GateResult when a change invalidates existing material. The F05 parent
coordinator persists each returned set as an idempotent typed `reference` note
before kickbacks or transition. Human Question resolution emits the same note
through the validated resolution service rather than a worker envelope.

## Requirements

1. **REQ-F-001 — Closed lifecycle tables**
   - A specification that introduces or changes a lifecycle, status,
     disposition, approval, retry, or failure field must list every defined
     value, meaning, allowed entry transitions, allowed exit transitions,
     terminal behavior, invalid transitions, and failure/recovery behavior.
   - A prose-only progression or an unbounded “other state” is a specification
     review failure.
   - A downstream transition obligation must name its owning feature and
     interaction; the owning specification must close it before passing.

2. **REQ-F-002 — Technique selection from state shape**
   - Test planning must select state-transition testing whenever an in-scope
     behavior depends on a lifecycle or disposition field.
   - Decision tables must cover each value and relevant cross-entity axis,
     including failure and terminal states.
   - The plan must include invalid-transition, recovery, and state-addition
     regression cases, not only the happy progression.

3. **REQ-F-003 — Dependency discovery by interaction and caller path**
   - Discover other state axes from I-## and X-## maps, API/event/data
     contracts, production callers, persistence readers, deduplication and
     short-circuit logic, and named downstream obligations.
   - Do not limit discovery to direct Shark dependencies, foreign keys, or one
     repository hop.
   - Record why each candidate axis is included or excluded.

4. **REQ-F-004 — Shipped consumer re-verification**
   - When a feature adds or changes a state read by an implemented consumer,
     list the consumer path, owning feature, affected ACs, and regression test.
   - Reopen the consumer's acceptance surface through an assigned task or
     linked follow-up when the current feature cannot safely update it.
   - A staged handoff must use the existing E34-F03 declaration contract and
     cannot be presented as live verification.

5. **REQ-F-005 — Shared naming integrity**
   - Task review must compare shared field, state, event, and contract names
     with the owning specification and interaction map verbatim.
   - Unexplained drift is a contract finding, even when local types compile.

6. **REQ-F-006 — Decision propagation**
   - Resolving a Question, tech debt item, change card, or ADR that changes
     implemented or specified behavior must produce I-04.
   - Amend invalidated specs, test plans, task requirements, interaction maps,
     standards, and ACs in the same change, or create and link explicit
     follow-up work for each deferred amendment.
   - A completion record must not claim consistency while any affected
     artifact is omitted without disposition.

7. **REQ-F-007 — Design divergence**
   - Rework that departs from an accepted fix design must cite the original
     decision, new evidence, affected consumers, and resulting artifact/test
     amendments.
   - Absence of a cited divergence means the recorded compatible design remains
     controlling.

8. **REQ-NF-001 — Content-level enforcement**
   - Implement these obligations in shared prompts, workflows, templates, and
     tests. Do not add lifecycle columns, relationship types, or a new
     dependency graph to the Shark database.

## Implementation plan

1. Add reusable lifecycle-table and change-impact guidance with templates for
   I-04 and state-space coverage.
2. Update feature specification, test planning, task review, and epic feature
   review prompts to invoke the guidance at their existing gates.
3. Update Question/tech-debt/decision resolution content to produce or verify
   the affected-artifact set and persist its typed `reference` note through the
   parent-owned resolution service.
4. Add rendered-prompt, reference, naming-drift, deferred-obligation,
   multi-entity-axis, and decision-propagation tests.

## Acceptance scenarios

**Plan a multi-entity lifecycle**

- Given a feature adds a failure state read by a deduplication service owned by
  a shipped feature,
- When specification and test planning run,
- Then the closed table includes the failure state and recovery transitions,
- And the consumer path, existing AC, cross-entity axis, and regression test
  are assigned before the plan passes.

**Propagate a ratified decision**

- Given a Question or tech-debt resolution changes an accepted conversion
  design,
- When the owner records the resolution,
- Then I-04 names every invalidated spec, test plan, task, and consumer AC,
- And each item is amended or linked to explicit follow-up work in the same
  session.

**Reject naming drift**

- Given a task renames an interaction field without updating its owning
  contract,
- When task review compares the shared names,
- Then it returns a structured contract finding rather than accepting the
  local implementation name.

## Dependencies and interactions

- Depends on E34-F05 for structured planning-gate findings and impact evidence.
- Consumes **I-02 GateResult v1**.
- Produces **I-04 ChangeImpactSet v1** for E34-F08 final integration review.
- Reuses E34-F03 I-## and staged-integration semantics.

## Out of scope

- A runtime state-machine engine or generated application code.
- A fixed foreign-key-hop rule for dependency discovery.
- Automatic rewriting of specs or tests from a decision record.
- New Shark entity or relationship types.

## Verification plan

- Render each changed prompt and update only affected goldens.
- Test closed and incomplete lifecycle tables, deferred obligations, non-FK
  consumers, shipped-consumer changes, naming drift, and decision propagation.
- Use decision-table fixtures for allowed entry, allowed exit, forbidden edge,
  terminal-state exit, recovery edge, and a new state whose consumer and
  regression coverage were not updated; every incomplete fixture must fail.
- Verify existing solo-feature and non-stateful workflows remain valid.
- Run `make fmt`, `make lint`, `make test`, and `git diff --check`.

*Last Updated*: 2026-08-05
