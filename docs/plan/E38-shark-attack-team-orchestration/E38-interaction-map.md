---
type: interaction-map
epic: E38
last_updated: 2026-07-13
---
# E38 Cross-Feature Interaction Map

E38 is expected to decompose into five features. The I-## IDs below are stable
within this epic and must be reused by decomposition, feature specifications,
task specifications, review, and QA.

| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |
|----|------------------|---------------------|-------|---------|-------|
| I-01 | E38-F01 Team Plan and Durable Ledger | E38-F02 Scheduler and Claims; E38-F03 Aggregate Routing and Resume; E38-F05 Reporting and Operator Surface | E38 architecture §4.2 Team-run domain contract | `TeamPlan`, `team_runs`, `team_run_items`, plan hash, wave, item status, claim/session links | Shared data model and service contract |
| I-02 | E38-F02 Scheduler and Claims | E38-F03 Aggregate Routing and Resume; E38-F05 Reporting and Operator Surface | E38 architecture §4.4 Aggregate outcome contract | Per-child semantic outcome, process result, evidence, skip reason, dependency state, timestamps | Service contract and shared data model |
| I-03 | E38-F03 Aggregate Routing and Resume | E38-F05 Reporting and Operator Surface | E38 architecture §4.4 Aggregate outcome contract | Aggregate outcome, configured target/paused boundary, root transition result, next action | Service contract and CLI/skill handoff |
| I-04 | E38-F04 Shark Attack Skill and Role Protocol | E38-F02 Scheduler and Claims; E38-F03 Aggregate Routing and Resume; E38-F05 Reporting and Operator Surface | E38 architecture §4.5 Council communication contract | Roster role, inbox message, handoff, decision, escalation, resolution, root/child scope | File artifact and UI/skill handoff |
| I-05 | E38-F02 Scheduler and Claims | E38-F05 Reporting and Operator Surface | E38 architecture §4.6 Operator contract | JSON `TeamRunResult`, per-child status, mode, concurrency, counts, evidence, next action | CLI contract |

No row uses X-## semantics; cross-epic integrations are recorded in
`E38-cross-epic-map.md` and the global product map.
