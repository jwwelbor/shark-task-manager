---
feature_key: E34-F03-deliverable-feature-decomposition-and-staged-integ
epic_key: E34
title: Deliverable Feature Decomposition and Staged Integration Acceptance
description: Make feature boundaries independently demonstrable, declare intentional staged integration before implementation, preserve strict security and integrity gates, and require downstream owners to close deferred production wiring.
---

# Deliverable Feature Decomposition and Staged Integration Acceptance

**Feature Key**: E34-F03-deliverable-feature-decomposition-and-staged-integ

> Triage brief: this captures the evidence, intended boundary, and downstream
> relationship established during triage. Detailed requirements and task
> decomposition remain work for the normal feature workflow.

## Epic

- **Epic PRD**: [Epic](../epic.md)
- **Related feature**: [E34-F02 Evidence-Based Demo Script Skill](../E34-F02-evidence-based-demo-script-skill/feature.md)

## Source evidence

- WWGM retrospective:
  `/home/jwwel/projects/wwgm/.worktrees/e04-complete-book-intake/dev-artifacts/2026-07-21-1600-halfway-feature-uat-retrospective/retrospective.md`
- Researched remediation plan:
  `/home/jwwel/projects/wwgm/.worktrees/e04-complete-book-intake/dev-artifacts/2026-07-21-1600-halfway-feature-uat-retrospective/remediation-plan.md`
- Triggering work: WWGM E04-F01 UAT, with E15-F02 as the prior recurrence.
- Related tracked WWGM obligations: TD-083 and B019.

The remediation plan is the detailed research artifact for this triage item.
This file records enough context to preserve the classification and scope without
turning triage into feature specification.

## Problem and evidence

Shark currently allows epics to be decomposed along architectural layers even
when an earlier feature cannot be driven through a production entrypoint to an
observable outcome until a later feature exists. Strict UAT then evaluates the
earlier feature as if it were a release-ready vertical slice and rediscovers the
known missing half. The result is a recurring rejection and owner-decision cycle
rather than an early decomposition decision.

The reviewed evidence separates three materially different cases:

| Evidence | Actual defect class | Policy consequence |
|---|---|---|
| E04-F01 `resolve_retry()` has no production caller until E04-F04 | A later feature owns activation of an earlier contract | Move the live-wiring acceptance criterion to the activation owner, or declare a narrow contract-only obligation before development |
| E04-F01 `uploaded_by` and `attested_by` lack an authenticated principal while E20 remains draft | A live path lacks a security prerequisite required by its present contract | Keep blocking, move the prerequisite earlier, hold exposure, or revise the current feature boundary; never treat it as a sequencing waiver |
| E15-F02 replay tests need captures produced with E15-F03 | The chosen review object does not own all fixtures required by the test | Declare accumulated-branch versus isolated-feature review semantics and assign fixture ownership explicitly |

Follow-up review of the E04 UAT artifact found three HIGH findings, including a
required CI-coverage finding in addition to the two sequencing/prerequisite
findings. The full round consumed about 35 minutes. The original retrospective
summarized two HIGH findings and about 20 minutes, so implementation should rely
on the researched remediation plan and underlying UAT artifact for measurements.

Other findings that shaped the proposal:

- E04 already documented the E04-F04 retry caller and the E20 authentication
  dependency. Disclosure in prose did not give UAT a machine- or
  prompt-enforceable acceptance disposition.
- A future Shark key is evidence of ownership, not evidence that a missing
  current security, integrity, or production-path requirement is acceptable.
- The UAT agent, skill, rubric, template, and feature approval prompt do not yet
  express one consistent distinction between assessor verdict, owner override,
  Accept with Conditions, and a truly passing production path.
- E34's existing I-##/X-## guidance follows interactions through QA, but does not
  yet require a complete demonstrable feature boundary, an activation owner, or
  closure of staged integration at UAT and epic completion.
- Throwaway callers created only to satisfy UAT would hide the ownership problem
  and add speculative production code; they are not an acceptable remedy.

## Classification decision

This is a **feature under E34**, not a WWGM-only task or a UAT bug:

- The behavior spans epic decomposition, feature specification and planning,
  review/UAT policy, and downstream closure guidance.
- It produces one cohesive shared capability: a feature acceptance/readiness
  contract used throughout Shark's prompt and skill lifecycle.
- No existing E34 feature owns that contract. E34-F01 owns harness-aware prompt
  rendering; E34-F02 owns demo-script generation and is a consumer of the new
  readiness evidence.
- Duplicate searches across E34 features and Shark's bug, tech-debt, change,
  idea, note, and text-search surfaces found no equivalent tracked entity.

## Intended capability boundary

During the normal feature workflow, refine a policy and embedded prompt/skill
contract with these outcomes:

1. **Deliverable decomposition**
   - Define an epic as an end-to-end outcome, a feature as an independently
     demonstrable state transition, and a task as contributing component work.
   - Require each feature to identify a real trigger, observable result,
     production path, complete UAT scenario, current prerequisites, and outputs
     intended for later consumers.
   - Reject feature acceptance criteria that require a later feature; move them
     to that feature or merge the proposed slices.

2. **Early staged-integration declaration**
   - Default interactions to `live`.
   - Permit `contract-only` only when declared no later than specification and
     confirmed at task review, with named counterpart entities, shared contract
     evidence, an activation owner, closure key, counterpart status, and review
     basis.
   - Treat reverse build-order consumption as a decomposition warning.

3. **Strict, consistent gate semantics**
   - A missing caller on a declared `live` edge remains blocking.
   - A complete, predeclared `contract-only` edge may be eligible for Accept
     with Conditions only after explicit owner approval.
   - Missing authentication, authorization, current integrity guarantees,
     unsafe exposure, or any unmet current-feature criterion always remains
     blocking.
   - Preserve the independent assessor verdict separately from an owner
     `override-accept` decision and its conditions.

4. **Downstream closure**
   - Require the activation owner's UAT to prove the real caller chain, shared
     contract, production-path integration test, and counterfactual failure when
     the wiring is removed or bypassed.
   - Do not allow an epic to complete with an unresolved internal activation
     obligation. External obligations may remain only with a named future owner
     and an explicit roadmap decision.

5. **Regression guards**
   - Cover prompt rendering and bundled UAT/review content so early declaration,
     security non-waiver, activation ownership, closure, and verdict vocabulary
     cannot drift independently.

Likely implementation surfaces are enumerated in the remediation plan. They
include epic design/decomposition/review prompts, the interaction-map templates,
feature specification through approval prompts, the bundled UAT and quality
skills, and rendered-prompt policy tests. Specification should verify the full
surface before generating tasks.

## Relationship to E34-F02

E34-F03 produces the acceptance/readiness model; E34-F02 consumes it to create a
truthful stakeholder demo. The relationship is directional and should become an
I-## row in E34's interaction map during design.

E34-F02 must read the latest independent assessor verdict, separate owner
decision, open conditions, I-##/X-## mode, activation owner, closure key,
counterpart status, and review basis. It must classify claims as:

1. `Demonstrated now`
2. `Not demonstrated / pending integration`
3. `Accepted risks and overrides`

Contract-only behavior, an open activation obligation, or behavior accepted by
overriding a blocking verdict cannot be promoted to verified end-to-end
delivery. If no complete observable scenario exists, E34-F02 should surface the
decomposition/evidence gap as a triage candidate; demo generation must not
become an acceptance gate.

Sequence E34-F03 policy specification before E34-F02 specification so F02 does
not invent a parallel readiness model. The E34-F02 feature contract was updated
during this triage to preserve that producer/consumer boundary.

## Prospective application

- Preserve completed WWGM feature history; do not claim later wiring existed.
- E04-F04 should own the live retry caller and its production-path integration
  test, and should close TD-083.
- E04's authentication gap represented by B019 remains a real roadmap/security
  decision, not a contract-only exception.
- E15 should record accumulated-branch review semantics separately from fixture
  ownership and isolated-commit reproducibility.

## Out of scope

- A new workflow engine, entity type, or runtime claim store.
- Weakening or replacing independent Codex red-team UAT.
- Automatically downgrading HIGH security, integrity, or undeclared wiring
  findings because later work has a Shark key.
- Rewriting historical feature verdicts or manufacturing throwaway callers.
- Requiring every feature to be independently deployable; the requirement is
  independently demonstrable and acceptable within declared release scope.
- Implementing E34-F02's demo recipe in this feature.

## Validation signals for specification

The finished behavior should demonstrate that:

- a feature claiming a later feature's acceptance criterion is rejected during
  decomposition or task review;
- a complete, predeclared contract-only edge receives the intended conditional
  treatment, while an undeclared zero-caller component remains blocking;
- a missing authenticated-principal boundary remains blocking even with a
  future auth epic;
- activation-owner UAT requires live wiring and closure;
- assessor verdict and owner decision remain separate facts; and
- every normal-mode E34-F02 demo claim is supported by acceptance, evidence, and
  activation state.

Track avoidable sequencing-only UAT rejects, early contract-only declarations,
activation closure rate, improper HIGH downgrades (target zero), avoidable UAT
time, and the share of features with a trigger, observable result,
production-path test, and complete UAT scenario.

---

*Triaged*: 2026-07-21

## Amendment — 2026-07-21

This feature remains a prompt and skill-content change. Its automated checks
validate bundle integrity: changed prompts render, includes resolve, and the
documented F03-to-F02 handoff files exist. Human review evaluates policy
wording against the specification. The feature does not require decision-table,
mutation, or simulated-runtime tests for prose-only policy changes.
