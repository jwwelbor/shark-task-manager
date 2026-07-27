# F05 assessment and v2 triage amendment

> **Historical assessment superseded on 2026-07-26:** The original assessment
> below considered only the lightweight F05 feature file and omitted the
> approved Shark Attack v2 triage decision. The v2 triage amendment beginning
> at [Corrected scope decision](#corrected-scope-decision) is current. It
> assigns Tranche B deterministic council tooling to F05 while retaining the
> implementation plan's unresolved design decisions.

## Goal

Decide whether E38-F05 needs separate feature delivery.

## Evidence

The feature asks for concise operator reporting of pulled, claimed,
dispatched, completed, blocked, and escalated work. Completed E38-F04 already
defines the durable council handoff, decision, and escalation artifacts. Its
embedded `message-schema.md` supplies the bounded scope, evidence, role,
status, and next-action fields. Completed E38-F07 already defines the Rider
parent loop and its operator-facing execution, escalation, stop, and resume
guidance in `workflows/execute.md`.

The remaining F05 wording does not identify an independent user outcome,
runtime surface, or contract that those features do not provide. Adding a
separate status or reporting subsystem would violate its out-of-scope boundary.

## Corrected scope decision

This conclusion is invalid because it omitted the approved v2 re-scope. F05
owns the deterministic council-artifact contract: typed artifacts, validated
generation, immutable revisions, entity-or-collection scope, evidence
confinement, effective-roster role checks, and thin `shark admin council`
commands. Those capabilities are distinct from the prose-only F04/F07
protocol.

## Superseded complexity triage

**Score:** 5/27
**Tier:** SIMPLE

### Technical complexity

1. File impact: 1/3 — one existing Rider or Shark Attack guide would change.
2. Pattern novelty: 0/3 — reuse the existing handoff and escalation pattern.
3. Data model: 0/3 — no persistence change.
4. API surface: 0/3 — no command or service change.
5. Cross-feature dependencies: 1/3 — reuse F04 and F07 artifacts.
6. UI complexity: 0/3 — Markdown guidance only.

### Execution complexity

7. Task estimation: 1/3 — one documentation task.
8. Regression risk: 1/3 — documentation could misstate Rider ownership.
9. Execution effort: 1/3 — a bounded review and contract check are required.

**Technical total:** 2/18

**Execution total:** 3/9
**Overall total:** 5/27

## Corrected complexity triage

**Score:** 23/27
**Tier:** COMPLEX

### Technical complexity

1. **File impact: 3/3** — the plan changes or adds models, repository and
   service layers, admin command wiring, embedded data, authored skill
   schemas/templates, documentation, fixtures, and contract tests.
2. **Pattern novelty: 3/3** — the repository has bounded Markdown council
   guidance and a generic entity writer, but no typed, revisioned council
   artifact contract or council validation service.
3. **Data model: 3/3** — artifact identity, roles, tagged scope, evidence,
   timestamps, and immutable `supersedes` lineage require a new validated
   model and YAML round-trip fixtures.
4. **API surface: 2/3** — new `shark admin council create`, `validate`, and
   `validate-wave` commands must be thin integrations over the service while
   preserving existing CLI contracts.
5. **Cross-feature dependencies: 3/3** — F05 builds on F04/F06/F07 and F08,
   and provides the artifact and ownership contract required by F09 and F11.
6. **UI complexity: 0/3** — there is no graphical UI; CLI output and YAML
   artifacts are the operator surfaces.

### Execution complexity

7. **Task estimation: 3/3** — the implementation plan identifies more than
   ten independently testable changes across model, repository, service, CLI,
   authored/embedded skill data, documentation, and fixtures.
8. **Regression risk: 3/3** — it touches embedded Shark-data loading and
   existing Shark Attack artifacts; an incorrect boundary could permit path
   escape, overwrite, invalid roles, or broken installed-skill behavior.
9. **Execution effort: 3/3** — the contract needs design confirmation,
   staged implementation, adversarial filesystem fixtures, integration tests,
   and full Go gates.

**Technical total:** 14/18
**Execution total:** 9/9
**Overall total:** 23/27

### Tier assignment

F05 is a correctly scoped feature: it delivers multiple cross-cutting,
design-dependent capabilities across substantially more than four files. It
is not a single task and should be delivered as COMPLEX work.

### Autonomous-build feasibility

- Task count: more than 10 (threshold <=10)
- Regression risk: 3/3 (threshold <=1)
- Execution effort: 3/3 (threshold <=1)
- Circular dependencies: no confirmed circular dependency; sequencing still
  requires the F08 integrity prerequisites and coordination with F09/F11.

**Recommendation:** manual, staged execution after research and specification;
do not begin implementation until the plan's unresolved schema, namespace,
closeout-role, and migration decisions have been recorded or approved.

## Correct next action

Continue F05 from assessment using the linked Shark Attack v2 implementation
plan. Do not treat the plan's open schema, namespace, closeout-role, or
migration decisions as approved.
