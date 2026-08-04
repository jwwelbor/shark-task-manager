---
feature_key: E38-F12
title: Parallel Team Topology
status: proposed
---

# E38-F12 — Parallel Team Topology

This specification is incremental over the [epic PRD](../epic.md), especially its goals, scope boundaries, team structure, and workflow-ownership decisions, and over the [epic architecture](../architecture.md), only where its current skill/protocol boundary remains applicable. The historical runtime design in that architecture is not revived. The decision source for this feature is the [Parallel Team Integration proposal](../proposal-parallel-team-integration.md), especially §§2–9. The [research report](research-report.md) Capability map governs the capabilities reused or extended here.

F09 is now present on the live baseline (commit `9eb07d63`), so the former thin-spec condition in `feature.md` no longer applies. This document specifies the proposal's two ordered phases without changing its recorded decisions.

## Requirements

### Functional requirements

| ID | Requirement | Epic traceability |
|---|---|---|
| REQ-F-001 | `/shark-rider run-agent-team` MUST be a topology adapter, not a scheduler: selection comes from `shark plan <root> --json` outside an active sprint and every selected delivery key re-enters through `shark next <key> --json`. The coordinator MUST not reconstruct a worker prompt or claim a delivery entity. | Epic PRD workflow ownership and out-of-scope boundary; proposal §§1–2 |
| REQ-F-002 | Each teammate MUST be the sole ordinary keyed Rider-loop parent for one assigned delivery entity at a time: keyed next, claim, worker dispatch with the exact rendered prompt, heartbeat as needed, semantic outcome advance, and release. A selector supplies only a key; it never supplies a claimable prompt. | Epic PRD ownership integrity; proposal §2 |
| REQ-F-003 | The coordinator MUST use rolling refill: after each teammate completion it re-runs selection and assigns only keys not already in flight. `shark plan <feature>` is relied on as claim-aware, while in-flight-key dedup remains a race defense. A no-candidate selection result is not success; completion requires `shark list <epic> <feature> --json` to show every task terminal, otherwise the closing report identifies paused or blocked work. | Epic PRD execution reporting; proposal §2; research Findings 1 |
| REQ-F-004 | Topology selection MUST reuse F09's independent two-axis model. `Sequential` is the default; shared-worktree parallel execution requires recorded disjoint ownership evidence, and isolated execution requires recorded isolation evidence. Missing matching evidence MUST degrade to `Sequential`; producer/consumer order remains binding under either parallel topology. | Epic PRD safety and scope boundaries; proposal §2; F09 REQ-F-001/002 |
| REQ-F-005 | The Phase 1 shared-worktree procedure MUST serialize commits and quality-gate runs through a coordinator-brokered turn and retain file-scoped staging discipline. It MUST not introduce a standing merge-referee role. | Epic PRD quality and ownership boundaries; proposal §§2, 4 |
| REQ-F-006 | The council wrap MUST preserve F09/E39 authority: the entity-parent teammate mints, configures, and links a scoped `Q###`; the coordinator routes responders and resolves the Question under its Question lease; routine Questions use the existing E39 route; material matters use the existing council route. Unrelated ready work continues. | Epic PRD escalation and durable-handoff requirements; proposal §3 |
| REQ-F-007 | A live Question hold MUST be event-bounded: held teammates heartbeat at least every ten minutes, starvation converts the deterministically longest-held worker to the existing bounded-handoff fallback, and each session boundary converts every hold before cleanup. When all remaining work is Question-gated, the coordinator reports the open Question keys rather than treating selection emptiness as completion. | Epic PRD liveness and escalation requirements; proposal §3 |
| REQ-F-008 | Team bootstrap MUST export `SHARK_CLAIM_TTL_SECONDS=1800` only while `.sharkconfig.json` has no `claim_ttl_seconds` key. Claims remain expiring liveness signals; the procedure MUST not specify persistent leases, force-steal as normal recovery, or a second claim store. | Epic PRD state ownership and recovery boundaries; proposal §§2, 9 |
| REQ-F-009 | Active-sprint execution MUST treat that sprint backlog as its sole selection universe. Until E19-F09 ships `shark plan sprint`, the coordinator MUST enumerate the active backlog in the documented sprint order, filter in-flight and Question-gated keys, and select the top eligible key per idle workflow role. Planning and retro are council ceremonies; only the owner starts or closes a sprint. | Epic PRD sprint/process boundary; proposal §5 |
| REQ-F-010 | `/run-sprint-team` MUST become a thin alias to the persistent parallel-team procedure's sprint mode. It MUST retire the legacy group-by-feature loop and must not nest one team bootstrap inside another. | Epic PRD workflow simplicity boundary; proposal §§5–6 |
| REQ-F-011 | Phase 2 MUST add only a thin integrator for isolated worktrees: one worktree and branch per teammate, serial merge-in on the integration branch, post-merge `make fmt && make lint && make test`, mechanical-conflict handling only, one scoped fix-forward after a red gate, durable feature-note and council-handoff traces, escalation after two consecutive fix-forward failures, and reviewed worktree closeout before removal. The integrator MUST never mutate Shark state. | Epic PRD quality, durable-evidence, and ownership requirements; proposal §4 |
| REQ-F-012 | The closing report MUST list entity, teammate, semantic outcome, merge commit, gate result, wave count, wall-clock duration, raised/resolved Question counts, and fix-forward count; it is persisted as a bounded feature note, with escalations retained in the council ledger. | Epic PRD operator-handoff requirement; proposal §7 |
| REQ-F-013 | The authored Shark Attack tree and its embedded mirror MUST remain byte-identical, and all changed prompt/skill contracts MUST be covered by rendered-content or contract tests. | Epic PRD distributable-skill requirement; F09 REQ-F-016; research Capability map |

### Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| REQ-NF-001 | This feature is prompt/skill-layer only: it MUST add no Go runtime, scheduler, dispatcher, claim store, schema, migration, synthetic graph edge, or provider adapter. | Diff review and `go test ./...` |
| REQ-NF-002 | Ordinary `/shark-rider run` remains the authoritative single-entity parent loop and is behaviorally unchanged. Team procedures reference it rather than copying or weakening it. | Contract tests inspect the thin pointer and keyed-loop language |
| REQ-NF-003 | No council artifact, handoff, team task, or closing report may persist a rendered prompt, credential, token, unrestricted worker transcript, or a second Question lifecycle record. | Negative corpus tests and F09 provenance guards |
| REQ-NF-004 | Parallelism is opt-in evidence-based behavior. Host capability discovery precedes topology selection; an absent team or isolation capability uses the documented sequential fallback without inventing a host command. | Scenario-table contract tests |
| REQ-NF-005 | The documentation must be internally consistent across authored and embedded skill trees, Rider verbs, and the embedded sprint-execution skill. | `make sync-shark-attack-skill` followed by `make test` |

### Acceptance criteria

| ID | Testable criterion |
|---|---|
| AC-001 | A free-selection fixture shows each team candidate originates in `shark plan <root> --json`, is passed to exactly one teammate keyed Rider loop, and reaches `shark next <key> --json` before any claim. The coordinator has no delivery-claim operation. |
| AC-002 | A claimed child is excluded by the keyed plan contract; an in-flight duplicate is rejected by the coordinator's dedup rule; a no-candidate selection with a non-terminal child yields a blocked/paused report rather than success. |
| AC-003 | A scenario table proves independent two-axis classification and both missing-evidence paths degrade to `Sequential`; a producer/consumer pair is serialized even with isolation evidence. |
| AC-004 | The shared-worktree procedure permits parallel teammate craft only with recorded disjoint ownership, and serializes commits plus full quality gates. |
| AC-005 | A routine `question` control envelope follows the existing E39 Question route: the teammate creates/configures/links the Question, the coordinator routes and resolves it, and unrelated ready work can refill. A material question follows the existing council route only. |
| AC-006 | A starvation event converts the oldest held worker to a bounded handoff; a session-boundary event converts every hold; an all-parked run reports the open `Q###` values. The procedure never uses a fixed Question-wait timeout. |
| AC-007 | Bootstrap contains the 1800-second environment-scoped lease policy, ten-minute heartbeat cadence, and normal expiry/replan recovery; it does not prescribe TTL zero, a config-file lease override, or a second claim store. |
| AC-008 | Sprint mode uses active-backlog-only selection and documented interim ordering until `shark plan sprint` exists; planning and retro record council evidence, while sprint start and close remain explicit owner gates. |
| AC-009 | `run-sprint-team` is a thin alias into sprint mode and no rendered procedure retains feature-grouped nested team bootstrapping. |
| AC-010 | Isolation mode provisions one worktree/branch per teammate, serializes integration, runs the full post-merge gate, leaves the required durable traces for fix-forward work, escalates after the second consecutive failed fix-forward, and performs reviewed closeout. Its integrator has no Shark state mutation. |
| AC-011 | The closing-report contract contains every REQ-F-012 field and records the bounded feature note/council escalation destinations without a rendered-prompt field. |
| AC-012 | Deliberate byte drift in either Shark Attack tree fails the existing parity gate; `make sync-shark-attack-skill` repairs it; the full test suite passes after synchronization. |

### Out of scope

- Any change to Shark runtime, scheduler, dispatcher, claim/work-session store, workflow engine, schema, migration, or HTTP API.
- A second claim, question, handoff, telemetry, or council-state store.
- New provider adapters or native agent-team implementations; F12 consumes the capability references F09 already delivered.
- Concurrent active sprints, automatic sprint start/close, automatic release approval, automatic conflict resolution, or an unreviewed worktree removal.
- E19-F09's `shark plan sprint` implementation. The documented client-side active-backlog selection is the compatible interim.

## Architecture

### Component changes

The design follows F09's split between the minimal Shark Attack router, single-purpose workflow files, canonical keyed Rider parent loop, and authored/embedded parity. It extends the research Capability map rather than re-implementing planning, prompt assembly, leases, Question lifecycle, provider capability detection, or council persistence.

| File | Change |
|---|---|
| `skills/shark-attack/workflows/parallel-team.md` | Create the canonical coordinator procedure. It owns selection/refill, topology evidence checks, coordinator/teammate/worker authority, 1800-second bootstrap, shared-worktree serialization, Question-hold events, sprint mode, closing report, and Phase 2 integrator handoff. It links rather than duplicates F09, Rider, E39, and council procedures. |
| `internal/sharkdata/default_data/skills/shark-attack/workflows/parallel-team.md` | Add the byte-identical embedded mirror of the authored workflow. |
| `skills/shark-attack/workflows/council.md` | Extend the existing material-council procedure with the planning and retro ceremony contracts, their evidence artifact requirements, chair responsibilities, and owner-only sprint lifecycle gate. Routine E39 Questions remain a link to `route-question.md`. |
| `internal/sharkdata/default_data/skills/shark-attack/workflows/council.md` | Apply the byte-identical mirror change. |
| `skills/shark-rider/verbs/run-agent-team.md` | Create the host-facing thin pointer. It retains host capability/branch/worktree preconditions, then delegates to the Shark Attack parallel-team procedure; it contains no hand-rolled DAG, no delivery transition, and no independently assembled worker prompt. |
| `skills/shark-rider/SKILL.md` | Register `/shark-rider run-agent-team` and revise the team-sprint description so it points to the persistent topology adapter instead of group-by-feature bootstrapping. |
| `skills/shark-rider/verbs/run-sprint-team.md` | Replace the legacy feature-group wrapper with a thin alias to `/shark-rider run-agent-team --sprint <S###>` (or the final equivalent documented by `parallel-team.md`), preserving explicit owner close gating. |
| `skills/shark-rider/skills/sprint-execution/SKILL.md` | Update the team-mode overview, command examples, and contract to the thin sprint-mode alias; retain solo `/run-sprint` behavior unchanged. |
| `skills/shark-rider/skills/sprint-execution/workflows/run-sprint-team.md` | Replace the old backlog grouping/nested-team workflow with the thin alias procedure and its explicit close-gate reference. |
| `internal/sharkdata/default_data/skills/sprint-execution/SKILL.md` | Mirror the revised embedded sprint-execution contract. |
| `internal/sharkdata/default_data/skills/sprint-execution/workflows/run-sprint-team.md` | Mirror the revised embedded team-sprint alias procedure. |
| `tests/contracts/e38_f12_parallel_team_topology_test.go` | Create content/contract tests for REQ-F-001 through REQ-F-013 and AC-001 through AC-012. Reuse F09 embedded-tree helpers/patterns where appropriate; do not add a runtime test harness for prose-only behavior. |

No production Go file changes, database changes, or public API/interface changes are permitted by this architecture.

### Data model changes

None. Shark's existing entity state, claim leases, work sessions, Question entities, dependency/Question gates, notes, and council files remain the sources of truth. Agent-team tasks are an ephemeral coordination overlay; they do not persist plan membership, waves, claims, Question state, or worker prompts. The closing report is a bounded feature note plus existing council artifacts, not a new record type.

### API and interface contracts

| Boundary | Contract |
|---|---|
| Selection → dispatch | `shark plan` is read-only selection. Every assigned entity re-enters with keyed `shark next`, whose rendered `response.prompt` is the sole worker payload. |
| Coordinator → teammate | The coordinator gives one concrete delivery key and topology constraints. The teammate is the keyed Rider parent and is the only delivery-entity claimant/advancer/releaser. |
| Worker → teammate | F09's existing ephemeral control-envelope schema remains authoritative. Semantic outcome keys are opaque and pass unchanged through the teammate parent. |
| Question handling | The entity-parent teammate creates/configures/links the E39 Question; the coordinator routes/responds/resolves under the Question lease. Existing `route-question.md` and `council.md` decide routine versus material paths. |
| Integrator → teammate | The integrator reports entity key, merge commit, and green/red gate result; only the teammate advances/releases after a green integration gate. On red, the fix-forward/council route applies with no integrator Shark mutation. |
| Sprint mode | The active backlog bounds selection; `shark plan sprint` replaces the documented client-side enumeration only after E19-F09 delivers it. Sprint start/close remain owner actions. |

### Key technical decisions

1. The coordinator is a topology and integration owner, never a second delivery-entity parent. This preserves the parent-loop and lease boundary in `skills/shark-rider/verbs/run.md` and F09.
2. F12 uses `shark plan` rather than retaining legacy order/link/DAG computation. Live code confirms `PlanHierarchyService.DescribeChildren` returns non-terminal, unclaimed, dependency-satisfied direct children; the procedure retains in-flight dedup because selection is not a reservation.
3. F12 creates one new canonical workflow rather than changing the two-axis classifier or copying its rules. This keeps F09's one-rule/one-file design and makes `parallel-team.md` a topology adapter beneath it.
4. The 30-minute environment-scoped TTL remains a liveness mechanism. A configuration key would suppress the environment override, while TTL zero would strand dead teammates; neither is acceptable.
5. Persistent holds are bounded by starvation and session events rather than a clock, because the intended same-worker resume path remains useful while a live worker exists but cannot survive team cleanup.
6. Isolation adds an integrator rather than a merge-referee or scheduler. The integrator's git-only scope fills the uncovered merge-and-post-merge-gate gap while canonical worker prompts retain craft/review discipline.

### Integration with existing code

This feature changes no Go implementation. Its live-code dependency evidence is `internal/services/plan_hierarchy_service.go`: `PlanHierarchyService.DescribeChildren(context.Context, parentType, parentKey)` returns only currently claimable children, and `internal/cli/commands/plan.go` exposes that read-only selection contract. The full post-merge quality gate is the existing `make fmt && make lint && make test` sequence. Prompt/skill integration follows `skills/shark-rider/verbs/run.md` for keyed dispatch and `skills/shark-attack/workflows/{route-question,council,resume}.md` for consultation, material routing, and resume boundaries.

## Cross-feature interactions

No current `I-##` row in [E38-interaction-map.md](../E38-interaction-map.md) names E38-F12 as a producer or consumer. This specification therefore does not invent an interaction identity or alter an existing row's producer/consumer ownership. F12 reuses F09's established contracts locally: the two-axis and provider-neutral coordination contract, the keyed Rider parent-loop contract, and the E39 Question consumer activation. If a later map revision assigns an I-## row to F12, its producer/consumer identities, shape source, gate mode, and shared contract-test pointer must be copied verbatim into this section before the associated implementation task is accepted.

## Cross-epic integrations

No current `X-##` row in [E38-cross-epic-map.md](../E38-cross-epic-map.md) or [the product integration map](../../../product/cross-epic-integration-map.md) names E38-F12 as an owning producer, consumer, or validator. The feature consumes already-activated E39/F09 behavior and the future E19-F09 plan-root through the local contracts above, without claiming or redefining X-06 or creating a new global integration ID. Any future cross-epic-map assignment must be copied verbatim here, including its shape source, UX/CX handoff note, and test-coverage pointer or product-progress deferral.

## Verification plan

1. Add the focused F12 contract suite and run it with the existing F09 interaction suite to prove the authored/embedded and Rider/embedded-sprint surfaces agree.
2. Run `make sync-shark-attack-skill`, introduce controlled mirror drift in a temporary test fixture, prove the parity failure, then re-run the sync path and confirm it is clean.
3. Run `make fmt && make lint && make test`. Although F12 has no production Go behavior, its contract suite is Go code and the repository quality gate applies.
