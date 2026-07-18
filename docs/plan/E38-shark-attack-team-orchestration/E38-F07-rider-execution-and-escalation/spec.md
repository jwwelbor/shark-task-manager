---
feature_key: E38-F07-rider-execution-and-escalation-loop
epic_key: E38
title: Rider Execution and Escalation Loop
status: proposed
---

# E38-F07 Rider Execution and Escalation Loop

This specification is incremental over the [epic PRD](../epic.md), especially
§2 (goals and locked decisions), §3 (scope and council contract), and §6
(UAT-03 through UAT-12). It follows the active boundary in the [epic
architecture](../architecture.md#current-architecture-boundary): the Rider
drives the ordinary Shark loop, the Shark Attack skill adds collaboration
guidance, and Shark remains the authority for state, routing, prompts, claims,
leases, and history.

## Requirements

### Functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-F-001 | Define one reusable Rider procedure that repeatedly calls `shark next <root> --json`, claims the returned concrete entity, dispatches `response.prompt` unchanged, records a semantic worker result, advances by that exact outcome, and releases the session before the next iteration. | Epic §2 goal 4; §3 in scope; UAT-03/UAT-04 |
| REQ-F-002 | Treat `spawn_agent`, `pause`, `archive`, and `error` as the complete dispatch action contract. Stop on pause, archive, error, or an explicit human gate; never infer success from partial child progress or create aggregate statuses. | Epic §3 constraints; UAT-04/UAT-05 |
| REQ-F-003 | Keep claim, heartbeat, release, notes, kickbacks, and status transition ownership in the Rider/parent procedure. The dispatched worker performs scoped craft and returns evidence plus a configured semantic outcome; it never mutates the dispatched entity's lease or workflow state. | Epic §2 locked decisions; §3 out of scope; UAT-03 |
| REQ-F-004 | Define role-aware self-pull as workflow-role selection through `shark sprint next --agent=<type>`, followed by `/shark-rider run <selected-key>`. The selection result is never directly claimed or executed: the Rider loop must call `shark next`, then claim `response.entity_key` and dispatch canonical prompt/provider metadata. Roster membership, responsibility prose, and model-tier preferences cannot grant selection, claim, or transition authority. | Feature requirements; Epic §2 assignment; UAT-12 |
| REQ-F-005 | Define escalation routing as worker question to responsible role, then chair, then product or human review only when the decision changes scope, architecture, or a quality gate. Every escalation references its question, evidence, responsible role, requested decision, route, and next owner. | Feature requirements; Epic §3 council contract; UAT-10/UAT-11 |
| REQ-F-006 | Persist a decision, handoff, blocker, or unresolved escalation through the existing bounded `docs/council/` artifact contract before a worker refresh or an escalation. Do not persist rendered prompts, credentials, unrestricted transcripts, or a second resume state. | Feature requirements; Epic §2 communication; UAT-09/UAT-10 |
| REQ-F-007 | Preserve ordinary `/shark-rider run` compatibility. The team procedure may reference Shark Attack collaboration and role-aware pull guidance, but it must not introduce a provider runtime, agent-team engine, new Shark command, or direct status assignment. | Epic §3 out of scope; current architecture boundary |

### Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| REQ-NF-001 | Prompt fidelity is exact: the procedure passes `response.prompt` verbatim and does not rebuild it from `shark get`, local skills, or a Shark persona name. | Dispatch contract content test; UAT-03 |
| REQ-NF-002 | The procedure is interruption-safe: each acquired session is released on normal, failed, or exception paths, and live work is recoverable through ordinary Shark claims, history, and `shark next`. | Procedure review and focused content tests; UAT-04/UAT-05 |
| REQ-NF-003 | Escalation and handoff records contain bounded relative artifact pointers and omit secrets, prompts, and unrestricted output. | Existing council-message contract tests plus execution procedure test |
| REQ-NF-004 | All content remains distributable through the embedded Shark-data bundle and replace-only overrides; host Rider guidance remains a host-local procedure and does not duplicate bundle personas. | `shark admin install-shark-data` fixture test |

### Acceptance criteria

| ID | Testable criterion |
|---|---|
| AC-001 | A documented run follows `next → claim → unchanged prompt dispatch → outcome/note/kickback handling → configured advance → session release` for the concrete entity returned by Shark. |
| AC-002 | A worker-facing ownership preamble prohibits worker claim, heartbeat, release, status advance, and direct status set for its dispatched entity; the parent procedure alone owns those actions. |
| AC-003 | `pause`, `archive`, `error`, missing outcome, worker failure, and a quality-gate fail have explicit stop/release or kickback behavior without a fabricated terminal status. |
| AC-004 | A role-aware self-pull example uses the existing workflow-role selector only to obtain a key, then enters `/shark-rider run <selected-key>`; it never directly claims or executes the returned `BacklogItemView` and does not treat the roster as authorization. |
| AC-005 | An escalation example records the material question, evidence, responsible role, requested decision, route, and next owner; it routes missing policy to `council-review` and does not name a fixed human target. |
| AC-006 | A refreshed coordinator can find the ordinary Shark claim/history state and bounded council decision, handoff, or escalation pointers needed to resume. |
| AC-007 | Materialized Shark-data contains the execution procedure and its references resolve without modifying ordinary single-entity Rider behavior. |

### Out of scope

- `internal/team`, `team_runs`, aggregate routing, a scheduler, a resume store,
  or any new Shark CLI command.
- A provider runtime, autonomous host-native agent-team engine, cross-project
  coordinator, credential store, dashboard, or telemetry product.
- Worker-owned workflow transitions, claim force-stealing, approval bypasses,
  automatic merge/conflict resolution, or a fixed human escalation destination.
- Reimplementing Shark prompt assembly, workflow routing, role filtering,
  claims, notes, or history.

## Architecture

### Component changes

| Path | Change | Boundary |
|---|---|---|
| `skills/shark-rider/verbs/run.md` | Refine the host-side parent loop, outcome directives, kickback ordering, release guarantees, and clean stop semantics. | Rider procedure; no Shark persona loading. |
| `skills/shark-rider/context/task-execution-pattern.md` | Align the spawned-worker result contract with parent-owned workflow mutation and bounded evidence. | Worker craft only. |
| `internal/sharkdata/default_data/skills/shark-attack/SKILL.md` | Link the council protocol to the existing Rider execution procedure and clarify that the chair has escalation authority, not workflow authority. | Collaboration guidance only. |
| `internal/sharkdata/default_data/skills/shark-attack/workflows/execute.md` | Add the role-aware pull, parent-loop, handoff, and escalation recipe for a chair-led team. | Uses existing Shark and Rider commands. |
| `internal/sharkdata/shark_attack_workflows_test.go` | Assert that a materialized bundle includes execution guidance and its required ownership, escalation, and resume boundaries. | Embedded-content contract. |
| `tests/contracts/e38_f07_interactions_test.go` | Add focused contract checks for exact prompt fidelity, parent ownership, role-aware self-pull wording, and bounded escalation/handoff fields. | Feature-level contract coverage. |

No schema, repository, service, HTTP API, Cobra command, workflow-YAML, or
provider adapter change is part of this feature. Existing `shark next`, claim,
heartbeat, release, note, and status-advance commands remain the operational
interfaces.

### Data model changes

There are no database changes. The procedure consumes the existing Shark
dispatch response (`action`, `entity_key`, `status`, `prompt`, `agent_type`,
`provider`, `model`, and `effort`), session-scoped claim identity, configured
outcomes, notes, history, and the council artifact schema from
`shark-attack/context/message-schema.md`.

The execution workflow adds no mutable run record. Durable collaboration state
is a bounded decision, handoff, or escalation under `docs/council/`; workflow
state remains in Shark.

### API/interface contracts

1. **Dispatch response:** `shark next <root> --json` is the only dispatch API.
   The parent consumes the returned concrete `entity_key`; `response.prompt` is
   the complete worker payload and is passed unchanged.
2. **Worker result:** the worker returns one configured semantic outcome and
   bounded evidence or notes. It may emit a complexity/parent note or explicit
   task kickbacks; it does not claim, heartbeat, release, or transition the
   dispatched entity.
3. **Parent loop:** the Rider claims, renews when needed, persists directives,
   applies kickbacks before parent advancement, advances with the exact outcome,
   and releases the same session on every exit path.
4. **Escalation record:** the existing council artifact shape supplies role,
   root/child scope, evidence, requested decision, route, status, and next
   action. `council-review` is the fallback route when project policy does not
   define one.

### Key technical decisions

| Decision | Rationale |
|---|---|
| Reuse the existing Rider `/run` loop instead of adding a team runner. | The active E38 architecture explicitly rejects a second runtime and keeps Shark's dispatch contract authoritative. |
| Treat the Shark Attack execution workflow as guidance around the Rider loop. | It supplies hierarchy, handoffs, and escalation while avoiding duplicate claims, status, prompt, or provider logic. |
| Advance by the worker's configured semantic outcome rather than a hardcoded status. | Route-based workflows own valid transitions and quality/approval boundaries. |
| Escalate material questions through roles and the chair, then review. | This preserves role accountability while avoiding a hard-coded human target or autonomous scope decision. |
| Use ordinary claims/history plus bounded council artifacts for resume. | This gives refreshed workers durable context without an aggregate ledger or a second resume state. |

### Integration with existing code and docs

- Dispatch semantics follow `skills/shark-rider/verbs/run.md`,
  `skills/shark-rider/context/workflow-and-status.md`, and
  `docs/architecture/shark-dispatch-prompt-assembly.md`.
- Council artifacts and escalation rules follow
  `internal/sharkdata/default_data/skills/shark-attack/context/message-schema.md`
  and `workflows/escalate.md`.
- Role-aware selection remains the existing Sprint surface documented by
  `internal/sharkdata/default_data/skills/shark-attack/workflows/pull-by-role.md`;
  it supplies a key to `/shark-rider run`, whose `shark next` response is the
  only claimable dispatch surface.
- Bundle delivery follows `internal/sharkdata/embed.go` and the existing
  `internal/sharkdata/*_test.go` materialization tests.

## Cross-feature interactions

### Consumes

- **I-04 — Council communication contract**; producer: E38-F04 Shark Attack
  Skill and Role Protocol. Shape source: E38 architecture §4.5 Council
  communication contract. Contract test pointer:
  `tests/contracts/e38_f04_interactions_test.go#TC-001`. E38-F07 consumes the
  existing bounded roster role, inbox message, handoff, decision, escalation,
  resolution, and root/child scope shape to record escalation and resume
  context; it does not create a second artifact schema.

## Cross-epic integrations

### Validates

- **X-01 — External orchestration runner dispatch seam**; producer epic: E22;
  consumer: E38. Contract / shape source: E38 architecture §4.3; existing E22
  runner dispatcher contract. UX/CX handoff: each worker receives one scoped
  Shark prompt and provider failures become visible outcomes. Test coverage:
  E38 uat-plan.md UAT-03, UAT-07, and `tests/contracts/e38_f07_interactions_test.go#TC-001`.

- **X-02 — Route-based workflow semantics**; producer epics: E16/E35;
  consumer: E38. Contract / shape source: E38 architecture §4.1 and §4.4;
  docs/guides/route-based-workflow.md. UX/CX handoff: pause and approval remain
  visible configured decisions, and the procedure never creates an aggregate
  status. Test coverage: E38 uat-plan.md UAT-04, UAT-05, UAT-10, and
  `tests/contracts/e38_f07_interactions_test.go#TC-002`.

## Exit-gate checklist

- [x] Every requirement is testable and traces to the active epic boundary.
- [x] The design names every planned source and test path.
- [x] No database/runtime/provider extension is implied.
- [x] I-04 reuses the parent map's exact shape source and contract pointer.
- [x] X-01 and X-02 retain their product-map contract sources, UX/CX handoffs,
  and UAT pointers.
