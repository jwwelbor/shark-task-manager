---
epic_key: E34
title: Prompt and Skill Improvements
size: L
---

# Prompt and Skill Improvements

**Epic Key**: E34

## Problem statement and business justification

Shark uses prompts and skills to carry workflow policy across planning,
implementation, review, and approval. Several essential contracts are still
duplicated or implicit: a cross-feature handoff can lose its producer,
consumer, and verification shape; a gate worker can return findings that the
parent does not persist before a transition; a repeated defect can receive a
point fix without a complete class sweep; and a replace-only project override
can hide later canonical improvements.

Those gaps cause avoidable rework, weaken the audit trail, and leave a
customized project unable to distinguish intentional local policy from stale
canonical content. E34 makes these workflow obligations durable, verifiable,
and reusable across Shark-managed projects. Avoiding one repeated rejection
loop or masked quality gate saves multiple agent sessions; the benefit grows as
later epics reuse the same contracts.

## Business value

**Rating**: High

These workflows drive delivery across every Shark-managed project. Preventing
one repeated rejection loop, stale decision, or masked canonical gate saves
multiple agent sessions and restores confidence in the audit record. The value
compounds because the capabilities are shared by every later epic and project.

## Epic components

- [Requirements catalog](./requirements.md)
- [Scope boundaries](./scope.md)
- [Workflow architecture](./architecture.md)
- [Cross-feature interaction map](./E34-interaction-map.md)
- [Review quality improvement plan](./E34-review-quality-improvement-plan.md)

## Feature portfolio

| Feature | Capability | State at this planning handoff |
|---|---|---|
| E34-F01 | Harness-aware prompt rendering | Existing feature |
| E34-F02 | Evidence-based demo script | Existing feature |
| E34-F03 | Deliverable decomposition and staged integration | Completed |
| E34-F04 | Question adoption for design and decisions | Completed |
| E34-F05 | Structured gate results and parent persistence | Draft, fully planned |
| E34-F06 | Defect-class completeness and recurrence routing | Draft, fully planned |
| E34-F07 | State-space planning and decision propagation | Draft, fully planned |
| E34-F08 | Tier-consistent gates and final integration review | Draft, fully planned |
| E34-F09 | Override drift visibility and WWGM reconciliation | Draft, fully planned |
| E34-F10 | Product critical-path guard for delivery workflows | Draft, fully planned |
| E34-F11 | Layered skill extraction adoption | Draft, fully planned |

Live lifecycle status remains in Shark; this table describes only the planning
packet and must not be used as a status cache.

Decision `D-E34-LEGACY-PROMPTS-001` (resolved 2026-08-31) covered the two
earlier unowned dev-artifact prompt requests, neither of which was delivered
by F05-F09. The "earlier ignored dev-artifact review prompt" was confirmed to
not exist anywhere in the repository and is cancelled — no tracked work is
created for it. The "skill-workflow-extraction prompt" does exist at
`dev-artifacts/planning/skill-workflow-extraction-prompt.md` (dated
2026-06-22) and is tracked as REQ-F-006 reference tooling by `E34-F11`
(`T-E34-F11-001`). See the E34 decision note referencing
`D-E34-LEGACY-PROMPTS-001` and `requirements.md` REQ-F-006 for the full
record.

## Goals and success criteria (measurable)

1. Every E34 cross-feature handoff has one registered I-## ID with a resolvable
   shape source, named producer, consumer set, and shared verification pointer.
2. In both Shark Rider and the core runner, every configured structured gate
   validates and persists bounded evidence, findings, remediation sweeps, and
   kickbacks before the parent makes a lifecycle transition.
3. Every closed blocking defect class has a completed sweep with enumerated
   instances, zero open instances, and a verified structural guard; recurrence
   classification uses that durable evidence rather than review-round count.
4. Every material decision identifies affected specifications, tests,
   consumers, and acceptance criteria, then records either an amendment or a
   linked follow-up item for each.
5. SIMPLE, STANDARD, and COMPLEX fixtures render their exact required artifact
   and gate matrix, and each required gate records the executed command,
   working directory, exit status, runner-native counts, skip comparison, and
   bounded evidence pointer.
6. The canonical epic workflow includes an additive `integration_review` step
   that evaluates the complete accumulated diff and cannot replace a failed
   required feature verdict with its own pass result.
7. Override status deterministically classifies every eligible path as
   `current`, `upstream_changed`, `identical_redundant`, `orphaned`, or
   `baseline_unknown`, without printing or modifying override content.
8. The E34 improvement plan assigns a disposition to every E04 proposal item
   and current WWGM override, with E40 scenarios recorded only as later,
   non-blocking validation work.
9. Before selecting or dispatching delivery work, Shark planning and execution
   prompts consult D01, D02, the delivery roadmap, and a durable critical-path
   artifact, then report the path gate, contribution, executable evidence,
   dependencies, and any side-quest disposition.

## Scope: in-scope and out-of-scope boundaries

### In scope

- Reusable prompt and skill policy for interaction tracking, evidence-based
  demonstration, material Questions, defect-class sweeps, state and decision
  closure, tier-consistent gates, and final integration review.
- The versioned, bounded `GateResult` contract and parent-owned persistence in
  Shark Rider and the Go core runner.
- Canonical workflow, prompt-rendering, validation, parity-test, manifest, and
  index changes required to support a final epic integration review.
- Read-only override-drift classification, explicit baseline provenance,
  upgrade-summary visibility, and acknowledgement metadata.
- One planned WWGM reconciliation item that promotes reusable behavior,
  retains intentional local policy, and accounts for existing change records.
- A reusable product critical-path guard across epic, feature, task, and
  sprint selection and dispatch workflows. This is E34-F10's distinct
  pre-dispatch product-alignment boundary; E34-F07 owns lifecycle/decision
  propagation and E34-F08 owns quality gates after work is selected.

### Out of scope

- Project-specific commands, environments, coding standards, model selection,
  test databases, and workflow order in Shark's canonical policy.
- Automatic override merging, patching, deletion, disabling, or rewriting.
- New storage types for review findings, recurrence, decisions, or interactions
  when typed notes, existing workflows, Questions, councils, and I/X maps can
  carry the record.
- A generated runtime state-machine engine, escalation based only on retry
  count, a global owner-approval setting, or a new standalone QA artifact for
  every STANDARD feature.
- Making the unfinished E40 benchmark corpus, harness, or measured delta a
  prerequisite for E34 delivery.

## Constraints and assumptions

- Parent loops retain claim, persistence, release, and lifecycle-transition
  authority; dispatched workers return bounded evidence only.
- E34 reuses existing notes, workflow outcomes, Questions, councils, and
  interaction maps before adding a new contract or storage mechanism.
- Canonical Shark content remains project-neutral: it cannot require a language,
  test runner, database, model provider, or project-local command.
- Operators reconcile project overrides explicitly. Shark may report digests
  and recommended actions but must not expose or alter override bytes.
- The historical WWGM E04-F02 lifecycle inconsistency is a bounded WWGM
  reconciliation concern, not a reason to change Shark's global approval
  policy.
- The E40 benchmark operator is shipped but provider-backed comparison is not a
  prerequisite; pinned E34 scenarios remain a later validation consumer.

## Stakeholder impact

| Stakeholder | Impact |
|---|---|
| Shark project operators | Gain deterministic, read-only visibility into override drift and an explicit acknowledgement path before local reconciliation. |
| Parent-loop owners | Retain lifecycle authority and gain idempotent, persisted gate evidence before any configured route executes. |
| Review, QA, and UAT workers | Use one tier-appropriate evidence contract and report findings without claiming state-mutation authority. |
| Feature planners and implementers | Receive stable interaction, decision-impact, and defect-sweep obligations that expose affected consumers and required regression coverage. |
| WWGM maintainers | Can promote reusable quality policy upstream while preserving intentional workflow policy and tracking local safeguards separately. |

## High-level acceptance criteria (UAT scenarios)

### Persist a structured failed gate before routing rework

- Given a configured gate finds several instances of one defect class,
- When its worker returns a bounded `GateResult`,
- Then the parent validates and persists the gate evidence, findings, sweep,
  and valid kickbacks before it routes the configured failure outcome,
- And malformed output, conflicting replay, or incomplete persistence cannot
  advance the entity.

### Close an integrated epic without superseding feature gates

- Given all required feature gates pass and the candidate contains changes from
  several features and rework rounds,
- When `integration_review` evaluates the resolved base-to-head diff,
- Then it verifies interactions, decision impacts, findings, defect guards,
  standards, and predicted debt across the accumulated candidate,
- And its pass result cannot supersede a failed required feature verdict.

### Verify each complexity tier with executable evidence

- Given equivalent SIMPLE, STANDARD, and COMPLEX fixtures,
- When their review and UAT prompts are rendered and their required gates run,
- Then each fixture requires only its matrix-defined artifacts and gates,
- And every required gate provides the exact command, working directory, exit
  status, runner counts, skip comparison, and bounded evidence pointer.

### Inspect a customized project safely

- Given a project has replace-only overrides and the canonical bundle changes,
- When an operator runs override status or upgrade dry-run,
- Then Shark reports stable digest-based classifications and summary counts,
- And the command neither changes nor exposes any override content.

## Open questions and assumptions

All epic-level planning decisions for E34-F05 through E34-F10 are resolved.
No open Question is required before implementation. The historical WWGM
E04-F02 lifecycle inconsistency is explicitly assigned to the later WWGM
reconciliation item; it is not treated as a global approval-policy decision.

*Last Updated*: 2026-08-30
