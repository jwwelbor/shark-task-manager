---
type: interaction-map
epic: E38
last_updated: 2026-07-25
---
# E38 Cross-Feature Interaction Map

E38 now has two interaction generations:

- I-01, I-02, I-03, and I-05 describe the superseded F01-F03 runtime design.
  Keep these IDs as historical records because existing specifications and
  reviews cite them.
- I-04 and I-06 through I-11 describe the active Shark Attack v2 delivery
  boundary across F04-F11. I-04 remains active because v2 extends the delivered
  council communication contract instead of replacing it.

The I-## IDs are stable within this epic. Reuse them in feature specifications,
task specifications, shared contract tests, review, QA, and UAT.

## Active interactions

| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |
|----|------------------|---------------------|-------|---------|-------|
| I-04 | E38-F04 Shark Attack Skill and Role Protocol | E38-F05 Lightweight Handoff and Operator Guidance; E38-F09 Provider-Neutral Coordination and Live Resume | [E38 architecture §4.5 Council communication contract](architecture.md#45-council-communication-contract) | Roster role; bounded question; handoff; decision; escalation; resolution; entity scope; evidence references | File artifact and skill handoff |
| I-06 | E38-F06 Role-aware Pull and Claim Guidance | E38-F05 Lightweight Handoff and Operator Guidance; E38-F09 Provider-Neutral Coordination and Live Resume | [F06 requirements](E38-F06-role-aware-pull-and-claim/feature.md#requirements) and [v2 authority model](../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#authority-model) | Workflow-resolved role; selected entity key; claim eligibility; existing-claim, workflow-gate, or no-work result | Read-only CLI selection and Rider handoff |
| I-07 | E38-F07 Rider Execution and Escalation Loop | E38-F05 Lightweight Handoff and Operator Guidance; E38-F09 Provider-Neutral Coordination and Live Resume | [Live question-and-resume protocol](../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#live-question-and-resume-protocol) | Keyed dispatch identity; parent-owned claim and transition boundary; workflow outcome; heartbeat state; question or bounded-handoff result | Rider procedure and worker-control contract |
| I-08 | E38-F08 Shark Attack v2 Integrity Prerequisites | E38-F09 Provider-Neutral Coordination and Live Resume; E38-F11 Complicated Lifecycle Release Qualification | [Implementation plan §5, Tranche A](../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#5-file-by-file-implementation-plan) | Read-only plan trace; unchanged status/history evidence; tech-debt related-document support; research prompt/validator rule; single implementation-gate ownership; nested-worktree regression result | Behavior contract and quality-gate evidence |
| I-09 | E38-F05 Lightweight Handoff and Operator Guidance | E38-F09 Provider-Neutral Coordination and Live Resume; E38-F11 Complicated Lifecycle Release Qualification | [Implementation plan §5, Tranche B](../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#5-file-by-file-implementation-plan) | Artifact type and immutable ID; effective roles; entity-or-collection scope; evidence paths; timestamps; `supersedes`; validation and generation result | Typed YAML file artifact and CLI data-tooling contract |
| I-10 | E38-F09 Provider-Neutral Coordination and Live Resume | E38-F10 Cross-Provider Adapter Conformance; E38-F11 Complicated Lifecycle Release Qualification | [Provider-neutral adapter contract](../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#3-provider-neutral-adapter-contract) | Coordination level; execution topology; capability profile; exact prompt hash and byte length; worker identity; control envelope; follow-up, interrupt, isolation, and replacement behavior | Skill protocol and host-adapter contract |
| I-11 | E38-F10 Cross-Provider Adapter Conformance | E38-F11 Complicated Lifecycle Release Qualification | [Cross-host conformance fixture](../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#cross-host-conformance-fixture) | Provider capability result; supported and unsupported operations; preference mapping; captured native evidence; sequential and replacement-worker fallback | Conformance evidence handoff |

## Historical interactions

These rows remain stable for auditability. F01-F03 are cancelled and create no
active implementation dependency.

| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |
|----|------------------|---------------------|-------|---------|-------|
| I-01 | E38-F01 Team Plan and Durable Ledger | E38-F02 Scheduler and Claims; E38-F03 Aggregate Routing and Resume; E38-F05 Reporting and Operator Surface | E38 architecture §4.2 Team-run domain contract | `TeamPlan`, `team_runs`, `team_run_items`, plan hash, wave, item status, claim/session links | Shared data model and service contract |
| I-02 | E38-F02 Scheduler and Claims | E38-F03 Aggregate Routing and Resume; E38-F05 Reporting and Operator Surface | E38 architecture §4.4 Aggregate outcome contract | Per-child semantic outcome, process result, evidence, skip reason, dependency state, timestamps | Service contract and shared data model |
| I-03 | E38-F03 Aggregate Routing and Resume | E38-F05 Reporting and Operator Surface | E38 architecture §4.4 Aggregate outcome contract | Aggregate outcome, configured target/paused boundary, root transition result, next action | Service contract and CLI/skill handoff |
| I-05 | E38-F02 Scheduler and Claims | E38-F05 Reporting and Operator Surface | E38 architecture §4.6 Operator contract | JSON `TeamRunResult`, per-child status, mode, concurrency, counts, evidence, next action | CLI contract |

Historical F02 and F03 specifications also cite I-04. Those references describe
the cancelled runtime consumer paths; the active I-04 consumers are F05 and
F09.

## Dependency order

The active feature graph has this topological order:

1. F04 provides the council and role baseline while F08 repairs the independent
   integrity prerequisites.
2. F06 adds role-aware selection and claim guidance after F04.
3. F07 provides the parent-owned Rider loop after F04 and F06.
4. F05 adds deterministic handoff and council-artifact tooling after F04, F06,
   and F07.
5. F09 integrates F04-F08 into the provider-neutral coordination and live-resume
   contract.
6. F10 applies the F09 adapter contract to additional providers.
7. F11 consumes the integrated contracts and conformance evidence for release
   qualification.

No active edge points to an earlier layer. The graph is acyclic. The interaction
rows define handoffs only; they do not approve the unresolved decisions in the
Shark Attack v2 implementation plan.

No row uses X-## semantics; cross-epic integrations are recorded in
`E38-cross-epic-map.md` and the global product map.
