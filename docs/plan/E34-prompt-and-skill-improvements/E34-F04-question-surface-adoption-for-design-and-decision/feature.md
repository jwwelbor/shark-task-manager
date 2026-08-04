---
feature_key: E34-F04-question-surface-adoption-for-design-and-decision
epic_key: E34
title: Question Surface Adoption for Design and Decision Prompts
description: Adopt the E39 Question lifecycle across decision-producing skills and prompts through one reusable Question skill, durable Q### records, and content-only verification.
---

# Question Surface Adoption for Design and Decision Prompts

**Feature Key**: E34-F04

## Goal

### Problem

Shark's E39 platform provides a first-class Question lifecycle, but the
architecture, frontend, database, requirements, planning, and product-design
workflows usually leave consequential unresolved items in prose or surface
them only in chat. That makes the item hard to discover, route, block, and
close. Shark Attack council covers material escalations, but it is not the
shared procedure those decision-producing workflows currently invoke.

### Solution

Ship a bundle-local `question-management` skill that defines when and how to
deduplicate, create, configure, route, answer, and resolve a `Q###` record.
Update decision-producing skills and prompts to reference it at their
open-decision boundary. Use the Question as lifecycle state and the narrowest
product, ADR, epic, or local design/specification document as the authoritative
decision source.

### Impact

Every material unresolved item produced by an in-scope workflow has one
deduplicated, linked Question or an explicit non-material rationale. Only
explicitly linked work is blocked. Resolutions point to an authoritative record
before Questions become terminal.

## Research findings

E39 is complete and already supplies typed Questions, ordered responders,
claim-bound responses, scoped `question_blocks`, safe read surfaces, and
resolution provenance. This feature must consume that platform; it does not
need Go, schema, CLI, API, workflow-YAML, or migration work.

Architecture's system, backend, frontend, database, and security workflows
have repeated interactive-only open-question reviews. The system template has
an "Open Questions & Decisions Required" section. Equivalent gaps exist in
`epic/design`, `epic/refinement`, and `feature/specification` prompts;
specification-writing workflows; product-design D01, D04, D06, D07, D08, D12,
and D14; and frontend aesthetic-direction work.

Shark Attack retains its existing boundary: a routine, bounded, single-role
Question uses E39 without a council artifact. Specialist disagreement, an
inconsistent cross-entity contract, high blast radius, irreversibility, or no
safe evidence-based path follows the council workflow. Solution walkthrough
remains an operator-approved Question consumer and must not auto-create or
auto-resolve Questions.

## User stories

### Must-have

**Story 1:** As a design or workflow agent, I want one Question procedure so
that I can turn a consequential unresolved decision into a durable, correctly
routed Shark record.

- [ ] `question-management/SKILL.md` defines candidate selection,
  deduplication, creation, configuration, blocking, escalation, response, and
  resolution.
- [ ] It identifies the authoritative decision record and supported E39
  resolution kinds.
- [ ] It preserves parent-owned mutations when a workflow forbids worker CRUD.

**Story 2:** As a stakeholder, I want material open decisions linked to their
source document and affected work so I can resolve them without relying on an
earlier chat.

- [ ] In-scope producers reference the shared skill at their decision boundary.
- [ ] Routine Questions remain in E39 without a council artifact; material
  Questions follow the existing council threshold.
- [ ] `question_blocks` is configured only after Question workflow setup and
  only for work that cannot safely proceed.

## Requirements

1. **REQ-F-001 — Reusable Question skill**
   - Ship `internal/sharkdata/default_data/skills/question-management/SKILL.md`.
   - Cover materiality, search/deduplication, source-document links, blocking
     edge order, council routing, parent-owned mutation, and durable resolution.

2. **REQ-F-002 — Architecture and frontend adoption**
   - Replace interactive-only open-question guidance in system, backend,
     frontend, database, and security design with a shared-skill reference plus
     workflow-specific candidate cues.
   - Apply the same procedure to a missing or contested frontend aesthetic
     direction when it meets the materiality test.

3. **REQ-F-003 — Specification, prompt, and product-design adoption**
   - Update `write-epic`, `write-feature-prd`, `refine-task-requirements`, and
     `decompose-epic`; `epic/refinement`, `epic/design`, and
     `feature/specification`; and decision-bearing product-design workflows.
   - Require a linked open Question for material unresolved items instead of
     treating no `TBD` text as sufficient closure.

4. **REQ-F-004 — Preserve existing consumers**
   - Reuse, never duplicate, Shark Attack council materiality and
     solution-walkthrough resolution boundaries.

5. **REQ-NF-001 — Content-only delivery**
   - No E39 platform, Question workflow, or runtime behavior changes.

6. **REQ-NF-002 — Prompt integrity**
   - Render changed prompts, update affected goldens, add focused content tests,
     and use no simulated policy engine.

## Implementation plan

1. Add and validate the `question-management` bundle skill with concise,
   copyable commands and the materiality decision tree.
2. Reference it from the architecture, specification-writing, product-design,
   and frontend-design skill routers and their listed decision producers.
3. Update the Architect agent and architecture template so each unresolved
   design item has a Question link or a documented non-material rationale.
4. Preserve Shark Attack's canonical material threshold and do not add a
   competing council format.
5. Add rendered-prompt and content-integrity tests, then run `make fmt`,
   `make lint`, and `make test`.

## Acceptance scenarios

**Record a material design decision**

- Given a design workflow finds an unresolved API, risk-acceptance, or scope
  choice,
- When it uses `question-management`,
- Then it reuses or creates one linked `Q###` with a real owner and responders,
- And it blocks only explicitly linked work when progress is unsafe.

**Resolve a Question**

- Given all configured responders have answered a Question,
- When the resolution owner accepts a decision,
- Then the authoritative decision document is updated before Question response
  and resolution provenance are recorded,
- And the Question reaches its appropriate terminal state.

## Out of scope

- E39 Question platform changes, including schema, CLI, API, workflow YAML, or
  database migrations.
- A new council policy, artifact type, or escalation threshold.
- Question records for routine fact lookup, low-impact authoring preferences,
  or every research assumption.

## Verification plan

- Assert that every named decision producer references the shared skill and
  durable-record policy.
- Render each changed prompt with the production renderer and update only its
  affected golden fixture.
- Verify the skill's command sequence, lifecycle boundaries, and required
  vocabulary with focused content tests.
- Run the standard quality gate.

*Last Updated*: 2026-08-04
