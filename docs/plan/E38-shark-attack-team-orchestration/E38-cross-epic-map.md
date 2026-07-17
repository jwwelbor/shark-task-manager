---
type: cross-epic-integration-map
epic: E38
last_updated: 2026-07-13
---
# E38 Cross-Epic Integration Map

Architect dependency review and CX review were completed during E38 design.
The architect identified the existing dispatch, workflow, sprint, telemetry,
and Shark-data boundaries. CX review covered preview-to-execution continuity,
pause and approval handoffs, resume state, role handoffs, escalation copy, and
council-memory continuity.

| ID | Producer epic | Consumer epic(s) | Integration purpose | Contract / shape source | UX / CX handoff notes | Owning feature | Status | Test coverage pointer |
|----|---------------|------------------|---------------------|-------------------------|-----------------------|----------------|--------|-----------------------|
| X-01 | E22 — External Orchestration Runner | E38 — Shark Attack Team Orchestration | Reuse single-worker dispatchers, process results, worktree capability checks, claims, and the fully rendered child prompt | E38 architecture §4.3; existing E22 runner dispatcher contract | Each team member still receives one scoped Shark prompt; provider failures are surfaced as child outcomes rather than hidden host behavior | E38-F02 Scheduler and Claims | assigned | E38 uat-plan.md UAT-03, UAT-07, UAT-10 |
| X-02 | E16/E35 — Multi-Level and Route-Based Workflow | E38 — Shark Attack Team Orchestration | Resolve worker roles, pause/terminal boundaries, semantic child outcomes, and configured root routing | E38 architecture §4.1 and §4.4; docs/guides/route-based-workflow.md | Approval and quality gates remain visible decisions; aggregate routing preserves the project’s configured language instead of inventing statuses | E38-F03 Aggregate Routing and Resume | assigned | E38 uat-plan.md UAT-04, UAT-05, UAT-10 |
| X-03 | E19 — Sprint Management and Planning | E38 — Shark Attack Team Orchestration | Supply priority/dependency order and workflow-role-aware pull/claim behavior for team members | E38 architecture §4.1 and §4.6; sprint pull/claim contract | Scrum Master monitors sequence while specialists self-pull eligible work; no legacy agent assignment is revived | E38-F04 Shark Attack Skill and Role Protocol | assigned | E38 uat-plan.md UAT-12 |
| X-04 | E23 — OpenTelemetry Observability | E38 — Shark Attack Team Orchestration | Carry run, root, child, wave, claim/session, duration, and outcome context into existing structured telemetry | E38 architecture §6; existing E23 observability contract | Operators can follow a paused or failed handoff without exposing prompts or worker secrets | E38-F05 Reporting and Operator Surface | assigned | E38 uat-plan.md UAT-03, UAT-04, UAT-06, UAT-07 |
| X-05 | E32 — Shark 2.0 Single-Artifact Consolidation | E38 — Shark Attack Team Orchestration | Distribute `shark-attack`, roster, and communication procedures through embedded Shark-data and replace-only overrides | E38 architecture §2 ADR-007 and §5 Phase 4; E32 embedded bundle contract | Setup explains where the skill and optional private council memory live; refreshed workers see the same versioned procedure | E38-F04 Shark Attack Skill and Role Protocol | assigned | E38 uat-plan.md UAT-08, UAT-09 |

These five X-## rows are the stable product-level integrations for E38. Any
new producer or consumer must receive a new X-## ID in the global map before it
is referenced by decomposition or feature specifications.
