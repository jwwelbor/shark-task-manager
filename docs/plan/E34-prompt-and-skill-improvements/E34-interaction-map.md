---
type: interaction-map
epic: E34
last_updated: 2026-08-05
---

# E34 Cross-Feature Interaction Map

E34 has nine features. This map is the authoritative registry for stable
cross-feature interaction IDs. Shape definitions live in
[architecture.md](./architecture.md); feature plans name their producer and
consumer obligations.

| ID | Producer feature | Consumer feature(s) | Shape source | Payload | Style |
|---|---|---|---|---|---|
| I-01 | E34-F03 Deliverable Feature Decomposition and Staged Integration Acceptance | E34-F02 Evidence-Based Demo Script Skill | [I-01 ReadinessEvidence v1](./architecture.md#i-01-readinessevidence-v1) | Readiness evidence classifies demo claims without granting acceptance authority | Documentation policy and feature-contract handoff |
| I-02 | E34-F05 Structured Gate Results and Parent-Owned Persistence | E34-F06, E34-F07, E34-F08 | [I-02 GateResult v1](./architecture.md#i-02-gateresult-v1) | Bounded gate outcome, evidence, findings, kickbacks, and sweeps persisted by the parent | JSON worker-to-parent contract |
| I-03 | E34-F06 Defect-Class Completeness and Recurrence Routing | E34-F08 | [I-03 DefectClassSweep v1](./architecture.md#i-03-defectclasssweep-v1) | Enumerated class scope, instances, dispositions, structural guard, and verification | Gate evidence nested in I-02 |
| I-04 | E34-F07 State-Space Planning and Decision Propagation | E34-F08 | [I-04 ChangeImpactSet v1](./architecture.md#i-04-changeimpactset-v1) | Decision/state change, affected artifacts and consumers, amendments, follow-ups, and verification | Planning evidence nested in I-02 or linked decision record |
| I-05 | E34-F08 Tier-Consistent Gates and Final Integration Review | E34-F09 | [I-05 CanonicalAdoptionManifest v1](./architecture.md#i-05-canonicaladoptionmanifest-v1) | Canonical bundle/version changes and explicit override adoption actions | Versioned JSON artifact and related document |

## Producer and consumer obligations

### I-01 readiness evidence shape

E34-F03 produces the documented readiness shape. E34-F02 consumes it
read-only and remains the activation owner for the real demo-script caller
chain. This compatibility heading preserves the existing consumer anchor; the
normative fields remain in
[architecture.md](./architecture.md#i-01-readinessevidence-v1).

The shared structural contract test is **TC-I-01-READINESS-SYMMETRY** at
`internal/cli/commands/interaction_prompts_test.go::TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY`.
Historical F03 **TC-002** covers prompt rendering only and is not an I-01 shape
contract test.

### I-02 GateResult

E34-F05 owns the model, parser, bounds, parent binding, persistence order,
replay behavior, and Rider/core parity. F06 uses it for findings and class
sweeps; F07 uses it for planning-gate findings and change impacts; F08 uses it
for tier gates and epic integration review.

The planned shared contract test is
`E34-F05-structured-gate-results-and-parent-owned-persisten/test-plan.md#TC-I-02-GATERESULT-PARITY`.
F05 test planning must create that exact pointer; all consumers reference it
instead of creating twin schema tests.

### I-03 DefectClassSweep

E34-F06 owns class identity, enumeration scope, count invariants, disposition,
guard closure, recurrence, and re-verification. E34-F08 rejects final
integration closure when a prior blocking class lacks a complete I-03 or its
guard is unverified.

The planned shared contract test is
`E34-F06-defect-class-completeness-and-recurrence-routing/test-plan.md#TC-I-03-DEFECT-CLASS-CLOSURE`.

### I-04 ChangeImpactSet

E34-F07 owns affected-artifact and consumer discovery, amendment/follow-up
accounting, shared-name checks, and shipped-AC regression assignments. E34-F08
verifies each I-04 is `accounted` before epic completion.

The planned shared contract test is
`E34-F07-state-space-planning-and-decision-propagation/test-plan.md#TC-I-04-DECISION-PROPAGATION`.

### I-05 CanonicalAdoptionManifest

E34-F08 produces I-05 only after canonical prompt, skill, workflow, and gate
changes pass their full validation. E34-F09 consumes its exact paths and
digests to plan override inspection; the manifest never authorizes automatic
project edits.

The planned shared contract test is
`E34-F09-override-drift-visibility-and-wwgm-reconciliation/test-plan.md#TC-I-05-OVERRIDE-ADOPTION`.

## Dependency order

1. E34-F05 establishes I-02.
2. E34-F06 and E34-F07 may proceed in parallel after F05; they produce I-03
   and I-04 independently.
3. E34-F08 consumes I-02 through I-04 and produces I-05.
4. E34-F09 consumes I-05 and performs cross-repository reconciliation.

Shark stores the same order through `depends_on` relationships. E40 is a
future validation consumer, not an E34 dependency.

## Registration

Registered with Shark as the E34 **Interaction Map** related document.
