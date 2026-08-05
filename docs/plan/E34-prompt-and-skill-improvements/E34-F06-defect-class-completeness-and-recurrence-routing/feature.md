---
feature_key: E34-F06-defect-class-completeness-and-recurrence-routing
epic_key: E34
title: Defect-Class Completeness and Recurrence Routing
description: Generalize gate findings into reusable defect classes, require whole-surface sweeps and structural guards, and route true recurrence through existing Question and council mechanisms without arbitrary retry counts.
---

# Defect-Class Completeness and Recurrence Routing

**Feature Key**: E34-F06

## Goal

### Problem

Review and rework guidance says to enumerate defects, but the procedure is
distributed across prompts and does not define when a class is actually
closed. Rework can fix one cited instance without consulting prior decisions,
review can rediscover the next sibling in a later round, and recurrence can be
confused with a new class or an already accepted risk. Numeric retry limits do
not solve that classification problem.

### Solution

Add one reusable defect-class workflow to the embedded quality bundle. A gate
names the class, records every in-scope instance, and proposes a guard. Rework
searches code and durable project records before changing the implementation.
Re-verification checks the entire class and its guard. Structured recurrence or
severity conflicts route through the existing Question and Shark Attack
council mechanisms when evidence cannot resolve them directly.

### Impact

A gate reports a defect family once with a complete surface, developers repair
the family rather than one symptom, and later reviewers can distinguish an
incomplete sweep from genuinely new evidence. Escalation is driven by durable
class evidence, not by how many times an agent happened to run.

## Research findings

- The canonical development prompt already requires a touched-module sibling
  sweep, and the UAT rubric requires enumerate-not-iterate and three-part
  re-verification. Neither defines a reusable sweep artifact, prior-record
  lookup, or structural-guard closure rule.
- WWGM's approval and red-team overrides add useful treatment for declared
  staged edges and already-dispositioned recurring findings. These are general
  gate semantics and belong upstream; copying the full overrides has already
  masked newer canonical prompt changes.
- E04's repeated span-invariant, rollback, and bare-assert findings demonstrate
  that repeated review rounds are an effect, not a reliable trigger. A true
  recurrence is evidence that a completed class sweep or guard failed.
- E38 and E39 already provide materiality, council, Question, responder, and
  durable resolution paths. This feature should route conflicts there rather
  than create a retry counter, escalation table, or new persistence type.

## Defect-class contract

The normative **I-03 DefectClassSweep v1** shape lives in
[Architecture](../architecture.md#i-03-defectclasssweep-v1). A class contains a
stable `class_key`, one-line class statement, enumerated search scope, counts,
instance evidence, remediation disposition, structural guard, and verification
evidence. It is persisted through E34-F05's GateResult and existing typed
notes.

## Requirements

1. **REQ-F-001 — Reusable sweep workflow**
   - Add one bundle-local quality workflow that review, QA, UAT, and
     development prompts reference rather than duplicate.
   - Define class naming, search scope, enumeration, zero-result reporting,
     instance evidence, guard selection, closure, and re-verification.
   - Require counts for searched sites, matching instances, repaired
     instances, intentionally dispositioned instances, and remaining open
     instances.

2. **REQ-F-002 — Backward-looking rework**
   - Before designing a repair, search the affected code and tests, feature and
     epic decisions, tech-debt records, prior review-finding notes, relevant
     specifications, and project standards for the class and affected symbol.
   - Implement a recorded compatible fix design or cite the durable evidence
     that justifies divergence.
   - Preserve unrelated owner decisions; do not reinterpret an existing
     disposition without new evidence.

3. **REQ-F-003 — Structural guard closure**
   - A class is complete only when every enumerated instance is fixed or has a
     cited disposition and a guard is verified.
   - A guard may be a lint rule, property test, contract test, conformance
     test, deterministic check, or other executable prevention mechanism.
   - If no feasible guard exists, keep the class open through a linked Shark
     work item instead of claiming closure.

4. **REQ-F-004 — Full-class re-verification**
   - Re-verification always checks the named fixes, re-enumerates the whole
     declared class scope, verifies the guard counterfactual, and reruns the
     full gate rubric.
   - A narrow fix request cannot reduce this scope.
   - The result reports scope and counts even when no remaining instance is
     found.

5. **REQ-F-005 — Evidence-based recurrence**
   - Treat the exact fingerprint resurfacing after a recorded repair as
     recurrence.
   - Treat a new fingerprint as recurrence only when it belongs to the same
     `class_key` and lies inside a previously completed sweep scope.
   - Treat findings outside that closed scope, or under a new class, as normal
     findings and route ordinary rework.
   - Do not use round number alone as an escalation or owner-interruption rule.

6. **REQ-F-006 — Disposition and severity conflict**
   - Keep a recurring finding visible but non-blocking when a dated,
     owner-grounded decision covers the same fingerprint and no material new
     evidence changes the risk.
   - When fresh evidence conflicts with the recorded severity or acceptance,
     label the result `severity_conflict` and block normal advancement.
   - Resolve a bounded single-owner conflict through an existing Question;
     route specialist disagreement, inconsistent cross-entity contracts, high
     blast radius, irreversibility, or no safe evidence path through the
     existing Shark Attack council workflow.

7. **REQ-F-007 — Generic bundle content**
   - Do not embed WWGM defect names, Python tools, test database variables, or
     local paths in the canonical workflow.
   - Discover project standards and executable guard commands from the project
     guidance supplied to the worker.

8. **REQ-NF-001 — No new authority or store**
   - Reuse E34-F05 GateResult, existing notes, E39 Questions, and E38 councils.
   - Do not introduce a recurrence table, automatic owner approval, or a new
     lifecycle engine.

## Implementation plan

1. Add the defect-class workflow and its I-03 output contract to the quality
   bundle, manifest, and contributor index.
2. Replace duplicated sweep text in review, QA, UAT, and development content
   with concise references plus gate-specific inputs.
3. Extend rework guidance with durable-record discovery and guard closure.
4. Add recurrence/disposition/severity-conflict routing to gate output policy
   and Rider handling through E34-F05.
5. Add focused content, render, reference, class-scenario, and routing tests.

## Acceptance scenarios

**Close an enumerated class**

- Given a review finds one unsafe error-recovery pattern,
- When the reviewer applies the shared workflow,
- Then it searches the declared surface, lists every matching instance and
  count, and returns one class-level sweep through GateResult,
- And rework cannot call the class closed until all instances and the guard
  are verified.

**Distinguish recurrence from a new finding**

- Given a later gate finds another instance,
- When its fingerprint and class are compared with prior completed sweeps,
- Then an in-scope instance is marked recurrence and a previously out-of-scope
  instance follows normal rework,
- And round count does not change the classification.

**Route a severity conflict**

- Given a new HIGH finding matches a prior accepted LOW-risk decision,
- When the gate records materially different evidence,
- Then the result blocks normal advancement and routes the conflict through an
  existing Question or council according to the canonical threshold.

## Dependencies and interactions

- Depends on E34-F05 for structured finding and sweep persistence.
- Consumes **I-02 GateResult v1**.
- Produces **I-03 DefectClassSweep v1** for E34-F08 final integration review.

## Out of scope

- Arbitrary escalation at review round three or owner hard-stop at round five.
- A global catalog of project-specific defect classes.
- Automatic changes to lint configuration or application code.
- Replacing the Question or council workflows.

## Verification plan

- Validate bundle manifest/index registration and every referenced path.
- Render all changed prompts through the production renderer.
- Test point instance, multiple siblings, zero remaining instances, same
  fingerprint recurrence, new in-scope fingerprint, new class, accepted risk,
  and severity-conflict scenarios.
- Assert no numeric retry threshold and no WWGM-specific term appears in the
  canonical workflow.
- Run `make fmt`, `make lint`, `make test`, and `git diff --check`.

*Last Updated*: 2026-08-05
