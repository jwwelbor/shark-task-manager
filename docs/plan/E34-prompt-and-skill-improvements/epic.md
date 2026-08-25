---
epic_key: E34
title: Prompt and Skill Improvements
size: L
---

# Prompt and Skill Improvements

**Epic Key**: E34

## Goal

### Problem

Shark's prompts and skills carry workflow policy across planning,
implementation, review, and approval, but several contracts remain duplicated
or implicit. Cross-feature handoffs can lose their producer/consumer shape,
material Questions can remain in prose, gate workers can return findings the
parent never persists, and replace-only project overrides can hide canonical
improvements after an upgrade. These gaps make workflow behavior difficult to
audit and allow the same defect class or stale decision to cycle through
multiple review rounds.

### Solution

Build a layered, reusable workflow-quality system. Preserve the existing
interaction-map, demo-evidence, staged-integration, and Question capabilities;
add structured parent-owned gate results, defect-class completeness, state and
decision propagation, tier-consistent executable gates, final epic integration
review, and visible override drift. Keep canonical policy project-neutral and
track project-specific adoption separately.

### Impact

- Cross-feature and cross-epic obligations remain traceable from planning to
  integrated acceptance.
- Material decisions and review findings become durable before lifecycle
  transitions.
- Rework repairs complete defect classes and leaves executable guards.
- Every complexity tier is reviewed against the artifacts it actually creates
  with tool-produced evidence.
- Epic completion evaluates the full accumulated change without silently
  overriding a failed feature gate.
- Customized projects can see and reconcile override drift deliberately.

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

Live lifecycle status remains in Shark; this table describes only the planning
packet and must not be used as a status cache.

Decision `D-E34-LEGACY-PROMPTS-001` explicitly defers the two earlier unowned
dev-artifact prompt requests. They are not delivered by F05-F09 and no ignored
path is claimed. The epic decomposition owner must assign each to a tracked
Shark feature/task and repository path, or record cancellation, before
decomposition can pass.

## Success criteria

1. Every E34 cross-feature handoff has one stable I-## shape source, producer,
   consumer set, and shared verification obligation.
2. Every configured structured gate persists bounded evidence, findings,
   sweeps, and kickbacks before transition in both Rider and core-runner paths.
3. Recurrence is classified from durable completed-sweep evidence, and every
   closed blocking class has a verified structural guard.
4. Lifecycle changes use closed transition tables and every material decision
   accounts for affected specs, tests, consumers, and ACs.
5. SIMPLE, STANDARD, and COMPLEX fixtures render exactly the canonical artifact
   and gate matrix and require tool-produced command results.
6. Canonical epic workflow contains an additive final integration review over
   the complete accumulated change.
7. Override status deterministically classifies current, upstream-changed,
   redundant, orphaned, and unknown-baseline paths without exposing content or
   modifying overrides.
8. Every E04 proposal item and WWGM override has a disposition in the linked
   improvement plan and E34-F05–F09.

## Constraints and assumptions

- Parent loops retain claim and lifecycle mutation authority.
- Existing notes, workflow outcomes, Questions, councils, and interaction maps
  are reused before adding storage or entity types.
- Canonical content cannot assume one project language, test runner, database,
  model provider, or local rule.
- Project override reconciliation remains an explicit operator action.
- The E40 benchmark operator is shipped but provider-backed comparison is not a
  prerequisite; pinned E34 scenarios remain a later validation consumer.

## High-level acceptance scenarios

**Structured failed gate**

- Given a gate finds several instances of one defect class,
- When the worker returns its bounded result,
- Then the parent persists every finding and sweep, applies validated
  kickbacks, and only then routes the configured failure outcome.

**Integrated epic candidate**

- Given all required feature gates pass and the candidate contains changes
  from several features,
- When epic integration review runs,
- Then it evaluates the full accumulated diff and closes interactions,
  decisions, findings, guards, standards, and predicted debt.

**Customized project upgrade**

- Given a project carries replace-only overrides and the canonical bundle
  changes,
- When an operator runs upgrade dry-run or override status,
- Then Shark reports digest-based drift classifications and leaves every
  override untouched.

## Open questions and assumptions

All epic-level planning decisions for E34-F05 through E34-F09 are resolved.
No open Question is required before implementation. The historical WWGM
E04-F02 lifecycle inconsistency is explicitly assigned to the later WWGM
reconciliation item; it is not treated as a global approval-policy decision.

*Last Updated*: 2026-08-05
