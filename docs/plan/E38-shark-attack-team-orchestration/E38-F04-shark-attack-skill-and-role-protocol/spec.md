---
feature_key: E38-F04-shark-attack-skill-and-role-protocol
epic_key: E38
title: Shark Attack Skill and Role Protocol
status: proposed
---

# E38-F04 Shark Attack Skill and Role Protocol

> Direction reset note (2026-07-13): this specification remains useful for the
> skill, roster, council, and escalation content. References to F01–F05 runtime
> consumers are historical; active acceptance is defined by `feature.md` and
> `../../uat-plan.md`.

> Scope correction (2026-07-14): the roster describes a collaboration
> protocol; it is not a team entity or an authorization model. Do not interpret
> responsibility prose or council-path preferences as permissions. Workers and
> coordinators use Shark's existing selection, claim, lease, and workflow APIs;
> F06 owns role-filter enforcement.

This specification is incremental over the [epic PRD](../epic.md), especially §2
(goals and locked decisions), §3 (scope and council contract), and §6
(UAT-8 through UAT-12). System decisions follow the [epic architecture](../architecture.md),
especially ADR-007 and §4.5. Feature research is in [research-report.md](research-report.md).

## Requirements

### Functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-F-001 | Ship a shark-attack skill defining a chair-led council, stable member IDs, role responsibilities, communication behavior, escalation authority, model-tier preference, and workflow-metadata precedence. | Epic §2 locked decisions; §3 Team structure |
| REQ-F-002 | Define a roster contract requiring team, chair, memory_root, communication, escalation, and unique members with IDs, roles, and responsibilities; model_tier is optional preference data. | Epic §3; UAT-8 |
| REQ-F-003 | Define docs/council/README.md, decisions/, handoffs/, escalations/, and inbox/<member-id>/, including setup, retention, privacy/gitignore, and refreshed-worker continuity. | Epic §2 council memory; UAT-9 |
| REQ-F-004 | Every inbox message MUST contain sender role, recipient role, root key, optional child key, subject, requested action or question, urgency, evidence/artifact links, and creation time. Messages are acknowledged/removed after action; resulting decisions, handoffs, unresolved questions, and resolutions are durable. | Architecture §4.5; I-04 |
| REQ-F-005 | Define bounded handoff, decision, escalation, and resolution artifacts preserving scope, roles, evidence, status, and next action; never store rendered prompts, credentials, or unrestricted worker output. | Epic §2 escalation contract; UAT-10/UAT-11 |
| REQ-F-006 | Define escalation triggers for missing evidence, material direction changes, expert disagreement, and unresolved process/quality blockers. If docs/product/escalation_triggers.md is absent, pause and route to council-review; never hard-code a human destination. | Epic §3; research-report §7; UAT-11 |
| REQ-F-007 | Role-aware self-pull MUST use workflow-resolved agent_type/step metadata, existing sprint priority/dependency ordering, and the owning claim service. Roster/model/legacy assignment data cannot override workflow authority. | Epic §2 assignment; X-03; UAT-12 |
| REQ-F-008 | Workers may read state, pull/claim authorized child work, write scoped artifacts, heartbeat/release their own child lease where authorized, and return evidence; they MUST NOT mutate the dispatched root lease or workflow state. | Epic §3 out of scope; Architecture ADR-005 |
| REQ-F-009 | Distribution MUST use the embedded Shark-data bundle and replace-only project overrides, referencing shark admin install-shark-data and existing /shark-rider, sprint, notes, context, and claim procedures. | Epic §2 skill ownership; X-05 |
| REQ-F-010 | Define missing-product-context and unavailable-team-capability behavior: recommend bootstrap/escalation or explicit sequential fallback, never guess product decisions or silently change ordinary /run. | Epic §2 prerequisite; §3 constraints |
| REQ-F-011 | Map stable roster IDs to existing personas (tech-director, product-manager, architect, business-analyst, scrum-master, developer, qa) while allowing project specialists without duplicating persona prompts. | Epic §3; research-report §2.2 |
| REQ-F-012 | Resume guidance MUST read durable decisions, handoffs, unresolved escalations, and inbox state, preserving unresolved context for F03 and bounded pointers for F05. | Epic §2 communication; I-04 |

### Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| REQ-NF-001 | Embedded bundle and manifest validation MUST reject invalid skill identity, roster structure, and persona references with actionable diagnostics. | shark admin validate-data; fixture tests |
| REQ-NF-002 | The published artifact protocol MUST reject credentials, access tokens, rendered prompts, unrestricted stdout, invalid Shark keys, and unsafe artifact paths. Roster prose and roster path preferences do not confer authority. | Content/security tests |
| REQ-NF-003 | Reads, acknowledgements, and writes MUST be idempotent across refreshed workers; conflicting reuse of an artifact ID is an actionable error. | Contract tests |
| REQ-NF-004 | No provider credentials, AI runtime, second workflow engine, or second claim store may be introduced. | Architecture review |
| REQ-NF-005 | Downstream consumers receive structured metadata and paths, not free-form role prose. | I-04 contract test |

### Acceptance criteria

| ID | Testable criterion |
|---|---|
| AC-001 | A valid chair-led roster passes bundle validation, including required fields, unique IDs, persona mappings, and preference-only model tiers. |
| AC-002 | Missing chair, duplicate ID, empty responsibility, unknown persona, or malformed roster structure fails validation with field diagnostics. |
| AC-003 | A message for root E38 and child E38-F04 remains understandable after worker refresh; acknowledgement/removal leaves the resulting durable handoff or decision. |
| AC-004 | Decision, handoff, escalation, and resolution artifacts contain required scope/role/evidence/next-action fields, exclude secrets/prompts, and are idempotent. |
| AC-005 | With no escalation policy file and an unresolved material question, the protocol creates an unresolved escalation, routes council-review, and recommends pause/review. |
| AC-006 | A developer self-pull uses workflow role filtering and existing deterministic sprint order, then the owning claim path; roster model tier and legacy assignment do not affect eligibility. |
| AC-007 | A child worker returns evidence and semantic outcome while the parent coordinator retains root lease and root workflow transition ownership. |
| AC-008 | Private council content can be gitignored or overridden while the protocol and required resume pointers remain available to refreshed workers. |
| AC-009 | A project override replaces only the default shark-attack skill and leaves unrelated embedded skills available. |
| AC-010 | Missing product gates or unavailable team capability produce bootstrap/escalation or explicit sequential fallback guidance without changing ordinary /run. |

### Out of scope

- Team planning/ledger, dependency scheduling, dispatch, claims, aggregate routing, root transitions, and reporting owned by E38-F01 through E38-F05.
- A new internal/team runtime, provider/model/credential system, dashboard, notification service, or cross-project executor.
- Automatic entity creation, decomposition, reprioritization, status mutation, merging, conflict resolution, approval, or fixed human escalation routing.
- Database tables for roster, inbox, memory, handoffs, or escalations.
- Atomic claim-next implementation; if required, it belongs to E19/F02. F04 documents and consumes that contract.

## Architecture

### Component changes

| Path | Change | Ownership boundary |
|---|---|---|
| internal/sharkdata/default_data/skills/shark-attack/SKILL.md | Create the distributable recipe covering setup, roster, roles, pull, communication, handoff, escalation, resume, security, and ownership. | Protocol only; no runtime. |
| internal/sharkdata/default_data/skills/shark-attack/context/roster-schema.yaml | Create canonical roster schema/template and role-to-persona mapping. | Model tier is non-authoritative. |
| internal/sharkdata/default_data/skills/shark-attack/context/message-schema.md | Define message, handoff, decision, escalation, and resolution fields/examples. | I-04 shape. |
| internal/sharkdata/default_data/skills/shark-attack/workflows/setup.md | Define installation, bootstrap, council initialization, private-memory option, and validation. | Existing admin commands. |
| internal/sharkdata/default_data/skills/shark-attack/workflows/pull-by-role.md | Define role-aware self-pull and worker ownership guardrails. | Consumes E19/F02. |
| internal/sharkdata/default_data/skills/shark-attack/workflows/communicate.md | Define inbox acknowledgement and durable-copy lifecycle. | File protocol. |
| internal/sharkdata/default_data/skills/shark-attack/workflows/escalate.md | Define triggers, council-review routing, and resolution. | No fixed human destination. |
| internal/sharkdata/default_data/skills/shark-attack/workflows/resume.md | Define refreshed-worker context loading and unresolved escalation handoff. | Supplies F03/F05 context. |
| internal/sharkdata/default_data/manifest.yaml | Add canonical shark-attack skill identity. | Existing validator. |
| internal/sharkdata/default_data/README.md | Document the skill and replace-only override location if required by bundle conventions. | E32 distribution. |
| docs/council/README.md | Create project protocol index, layout, retention, privacy, and refresh instructions. | Project artifact. |
| docs/council/decisions/.gitkeep, handoffs/.gitkeep, escalations/.gitkeep, inbox/.gitkeep | Create directory markers; member inboxes derive from roster IDs. | Project file contract. |
| tests/contracts/e38_f04_interactions_test.go | Create I-04 contract fixtures: TC-001 shared message shape, TC-002 durable lifecycle, TC-003 role pull, TC-004 bundle override. | Shared pointers for consumers. |
| internal/sharkdata/embed_test.go or focused internal/sharkdata/shark_attack_test.go | Extend validation tests for skill identity, roster structure, personas, and overrides. | Existing test harness. |

No database migration or change to internal/models, internal/repository,
internal/services/claim_service.go, internal/services/sprint_service.go, or
internal/cli/commands/sprint.go is owned by F04. If E19/F02 changes claim-next,
F04 consumes the finalized interface.

### Data model changes

There are no Shark database schema changes. The file contracts are:

1. **Roster YAML**: team MUST equal shark-attack; chair MUST reference a member;
   communication requires acknowledge_after_read and retain_decisions;
   escalation has optional triggers_file and non-empty route; member IDs are
   unique and each member has non-empty role and responsibilities, optional
   existing persona, and optional model_tier. The values describe local
   collaboration and never grant Shark status or claim authority.
2. **Inbox message**: structured metadata with message_id, sender_role,
   recipient_role, root_key, optional child_key, subject, requested_action or
   question, urgency, evidence, and created_at. Body content is bounded context,
   not a prompt transcript.
3. **Durable artifact**: decisions, handoffs, escalations, and resolutions retain
   artifact_id, type, status, roles, root/child scope, evidence, created/updated
   timestamps, and next_action; escalations add trigger and resolution/reference
   fields. Artifact IDs make resume writes idempotent.

Apply the artifact protocol's bounded-path and key rules when writing a message
or durable artifact. The roster validator does not turn roster text or local
path preferences into authorization policy.

### API/interface contracts

F04 adds no HTTP API. Its portable interfaces are:

- **Roster loader**: project/roster path in; structurally validated roster plus
  explicit role-to-persona mapping or field-specific error out. It cannot grant
  status or claim authority.
- **Council file protocol**: the published Markdown/YAML schema defines typed
  records under the configured council root. Workers follow its bounded-path,
  duplicate-ID, and acknowledgement guidance; F04 adds no runtime writer.
- **Role pull handoff**: consumes SprintService.GetNextTask(ctx, agentType) /
  shark sprint next --agent=<type> and ClaimService.Claim with ClaimInput. The
  scheduler owns atomicity, lease, release, and retry behavior.
- **I-04 handoff**: exact Architecture §4.5 fields are sender role, recipient
  role, root/child key, subject, requested action or question, urgency, evidence
  links, and created time. Durable type/status/next-action are local extensions.

### Key technical decisions

| Decision | Rationale |
|---|---|
| Ship Markdown/YAML under internal/sharkdata/default_data, registered in manifest.yaml. | Follows Architecture ADR-007 and E32 embedded/replace-only distribution; avoids a second package manager/runtime. |
| Keep communication file-based under docs/council/. | Matches project-scope memory, survives refresh, and avoids coupling transient collaboration to Shark tables. |
| Reuse existing personas and workflow metadata. | agents/*.md, internal/config/workflow/schema.go, and steps.go are existing authority seams; duplication creates conflict. |
| Keep selection/claims in existing services/owning features. | SprintService.GetNextTask gives deterministic role filtering/order and ClaimService owns lease safety. |
| Use bounded structured artifact guidance with stable IDs. | Gives F06/F07 stable pointers, supports repeatable manual resume, and limits secret leakage without adding a persistence runtime. |
| Default absent escalation policy to pause plus council-review. | Research confirms the policy file is absent; pausing is safer than guessing and preserves approval boundaries. |

### Integration with existing code

- Bundle resolution/validation follows internal/sharkdata/embed.go and
  internal/cli/commands/sharkdata_cmd.go; setup uses shark admin install-shark-data
  and shark admin validate-data.
- Persona mapping uses internal/sharkdata/default_data/agents/,
  docs/research/project-roles.md, and docs/research/roles-to-skills.md.
- Workflow authority follows internal/config/workflow/schema.go,
  internal/config/workflow/steps.go, and
  internal/sharkdata/default_data/workflow/*.yaml.
- Pull guidance wraps internal/cli/commands/sprint.go’s
  shark sprint next --agent and internal/services/sprint_service.go’s
  GetNextTask(ctx context.Context, agentType string).
- Claim guidance delegates to internal/services/claim_service.go’s ClaimInput,
  Claim, Heartbeat, and session-scoped Release.
- Entity-scoped evidence may link Shark notes/context; project decisions remain
  files. F02 consumes role/message paths while owning scheduling/claims, F03
  consumes escalation/resume context while owning aggregate routing, and F05
  consumes bounded pointers while owning presentation.

## Cross-feature interactions

### Produces

- **I-04 — Council communication contract**; consumers: E38-F02 Scheduler and
  Claims, E38-F03 Aggregate Routing and Resume, and E38-F05 Reporting and
  Operator Surface. Shape source: E38 architecture §4.5 Council communication
  contract. Contract tests:
  `tests/contracts/e38_f04_interactions_test.go#TC-001` (canonical shared
  contract pointer). The same contract suite's TC-002 provides supplementary
  durable artifact lifecycle coverage. Payload is roster role, inbox
  message, handoff, decision, escalation, resolution, and root/child scope;
  consumers receive paths and bounded metadata, not free-form role prose.

## Cross-epic integrations

### Consumes

- **X-03 — Workflow-role-aware pull/claim behavior**; producer epic: E19 —
  Sprint Management and Planning; consumer: E38-F04. Contract / shape source:
  E38 architecture §4.1 and §4.6; sprint pull/claim contract, exactly as
  recorded in E38-cross-epic-map.md. UX/CX handoff: Scrum Master monitors
  sequence while specialists self-pull eligible work; legacy agent assignment
  is not revived. Test coverage: E38 uat-plan.md UAT-12 and
  tests/contracts/e38_f04_interactions_test.go#TC-003; atomic claim-next,
  if needed, remains E19/F02-owned.

- **X-05 — Embedded Shark-data and replace-only override distribution**;
  producer epic: E32 — Shark 2.0 Single-Artifact Consolidation; consumer:
  E38-F04. Contract /
  shape source: E38 architecture §2 ADR-007 and §5 Phase 4; E32 embedded
  bundle contract, exactly as recorded in E38-cross-epic-map.md. UX/CX
  handoff: setup explains where the skill and optional private council memory
  live so refreshed workers see the same versioned procedure. Test coverage:
  E38 uat-plan.md UAT-08 and UAT-09 and
  tests/contracts/e38_f04_interactions_test.go#TC-004.

## Exit-gate checklist

- [x] Every requirement is testable and traces to the epic.
- [x] Architecture decisions reference existing patterns or explain deviation.
- [x] All planned source/test paths are listed.
- [x] No critical TBD remains; absent escalation policy has an explicit fallback.
- [x] I-04 uses the exact parent shape source and shared contract pointers.
- [x] X-03 and X-05 preserve the map's producer/consumer directions (E19 -> E38-F04 and E32 -> E38-F04), exact shape sources, UX/CX handoffs, and coverage.
