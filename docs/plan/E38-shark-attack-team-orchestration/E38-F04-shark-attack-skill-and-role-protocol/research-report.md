# E38-F04 Research Report: Shark Attack Skill and Role Protocol

> Direction reset note (2026-07-13): references to the former F01–F03 runtime
> contracts below are historical dependency analysis. The active F04 boundary
> is the skill/protocol described in `feature.md`, with F06/F07 as consumers.

**Scan date:** 2026-07-13
**Repository:** `/home/jwwel/projects/shark-task-manager`
**Feature:** Shark Attack Skill and Role Protocol
**Depth:** Deep tactical codebase and dependency analysis

## Executive recommendation

Implement F04 as a distributable content/protocol package under the existing
embedded `shark-data` system. Add one canonical `shark-attack` skill directory,
one validated project-roster contract, and setup/protocol documents for council
memory, inbox messages, handoffs, and escalation. Reuse the existing agent
personas, workflow role metadata, sprint role filter, claim behavior, prompt
include/override resolution, and Shark notes/context for durable entity state.

Do not create a second scheduler, claim API, workflow authority, or provider
runtime. The roster describes collaboration and model preferences; workflow
YAML remains authoritative for the resolved responsibility, agent, provider,
model, skills, and semantic outcome. A roster member must never gain status,
claim, or transition authority merely by being listed.

There is no existing F04 research report in the feature directory and no
existing `shark-attack` skill, roster, or `docs/council/` implementation in the
current checkout. The work is therefore new content built on proven extension
points, not a refactor of an existing attack-team implementation.

## 1. Strategic scope and sibling contracts

The feature description is [`feature.md`](./feature.md). The parent epic and
architecture define the required chair-led council, roster YAML, project-local
`docs/council/` layout, workflow-role-aware self-pull, and communication shape
([`epic.md`](../epic.md), especially the team structure/YAML and council-memory
sections; [`architecture.md`](../architecture.md), ADR-007 and §4.5).

Sibling relationships:

| Sibling | Relationship to F04 | Required boundary |
|---|---|---|
| E38-F01 Team Plan and Durable Ledger | Supplies planned role/provider/model metadata and durable run/item identity | F04 consumes the role metadata contract; it does not own ledger persistence |
| E38-F02 Scheduler and Claims | Dispatches children and owns child claim/session lifecycle | F04 supplies role/communication context only; it must not claim or release work |
| E38-F03 Aggregate Routing and Resume | Consumes worker outcomes and routes the root | F04 supplies handoff, escalation, and role context; configured workflow outcomes remain authoritative |
| E38-F05 Reporting and Operator Surface | Reports per-child outcomes, evidence, next action, and council handoffs | F04 defines bounded artifact/message fields and paths; F05 formats them |
| E19 Sprint Management | Provides priority/dependency ordering and role-aware pull/claim behavior | F04 should reuse `sprint next --agent`; any `claim-next` addition belongs to the owning sprint/claim service |
| E32 Single-Artifact Consolidation | Provides embedded bundle and replace-only override distribution | F04 must ship through `internal/sharkdata/default_data` and update bundle validation metadata |

## 2. Existing implementations and reusable patterns

### 2.1 Embedded skill, agent, workflow, and override bundle

The canonical source is [`internal/sharkdata/default_data/`](../../../internal/sharkdata/default_data/):

- [`README.md`](../../../internal/sharkdata/default_data/README.md) defines the
  bundle layout (`skills/`, `agents/`, `workflow/`, `overrides/`) and states
  that overrides fully replace defaults.
- [`manifest.yaml`](../../../internal/sharkdata/default_data/manifest.yaml)
  is the validator’s declarative skill registry. Every new skill needs a
  normalized `name` and ownership entry here.
- [`internal/sharkdata/embed.go`](../../../internal/sharkdata/embed.go) embeds
  the tree, resolves disk content with embedded fallback, and preserves
  project-local overrides during upgrade.
- [`internal/cli/commands/sharkdata_cmd.go`](../../../internal/cli/commands/sharkdata_cmd.go)
  exposes `shark admin install-shark-data`, upgrade, and validation. F04 setup
  guidance should point to this existing command rather than inventing an
  installer.
- [`internal/sharkdata/embed.go`](../../../internal/sharkdata/embed.go) also
  validates workflow references to agents, prompts, and skills; the new skill
  must pass this validation and must not be referenced by workflow YAML until
  its canonical file exists.

**Extension decision:** extend the embedded content tree and manifest. Do not
add a separate package manager, repository-level `skills/` source of truth, or
runtime loader. The expected shipped path is
`internal/sharkdata/default_data/skills/shark-attack/SKILL.md`, materialized as
`shark-data/skills/shark-attack/SKILL.md`; project customizations belong under
`shark-data/overrides/skills/shark-attack/`.

### 2.2 Existing role/persona definitions

The existing role files are reusable behavior baselines:

- [`agents/tech-director.md`](../../../internal/sharkdata/default_data/agents/tech-director.md)
  already describes chair-like monitoring, routing, escalation, and keeping
  Shark as source of truth.
- [`agents/product-manager.md`](../../../internal/sharkdata/default_data/agents/product-manager.md)
  owns product direction, priority, scope, and stakeholder escalation.
- [`agents/architect.md`](../../../internal/sharkdata/default_data/agents/architect.md)
  owns technical design, feasibility, standards, and decisions.
- [`agents/business-analyst.md`](../../../internal/sharkdata/default_data/agents/business-analyst.md)
  owns requirements, acceptance criteria, decomposition, and edge cases.
- [`agents/developer.md`](../../../internal/sharkdata/default_data/agents/developer.md)
  owns implementation and tests; [`agents/qa.md`](../../../internal/sharkdata/default_data/agents/qa.md)
  owns verification and defects.
- [`agents/researcher.md`](../../../internal/sharkdata/default_data/agents/researcher.md)
  supplies the discovery/context role used by this report and future council
  assignments.

[`docs/research/project-roles.md`](../../../docs/research/project-roles.md) and
[`docs/research/roles-to-skills.md`](../../../docs/research/roles-to-skills.md)
are the existing taxonomy references. They should be linked from the new skill,
not copied into a competing role definition. The roster may use stable role
IDs (`tech-director`, `product-manager`, etc.) and map them to existing agent
persona names, while allowing project-specific specialists.

**Extension decision:** reuse persona files and role taxonomy. Add only the
team collaboration layer (responsibilities, communication behavior,
escalation authority, optional model tier). Do not duplicate persona prompts in
the roster or let a roster model preference override workflow-resolved routing.

### 2.3 Route-based workflow role authority

Role and assignment authority already exists in route-based workflow config:

- [`internal/config/workflow/schema.go`](../../../internal/config/workflow/schema.go)
  defines step responsibility, agent types, action, provider, model, effort,
  skills, prompt, semantic outcomes, pause, and terminal metadata.
- [`internal/config/workflow/steps.go`](../../../internal/config/workflow/steps.go)
  resolves/narrows step metadata and agent types.
- Default entity workflows under
  [`internal/sharkdata/default_data/workflow/`](../../../internal/sharkdata/default_data/workflow/)
  show the established agent/human/none boundaries.

The epic explicitly says the roster explains collaboration around workflow
steps; it does not assign statuses or bypass claims. F04 documentation must
repeat this rule in setup, pull, escalation, and resume procedures.

**Extension decision:** documentation-only integration with workflow metadata,
unless implementation discovers a narrowly required roster validator. Never
make roster membership a second routing source. If a future validator is
needed, place it beside existing shark-data validation and validate keys,
responsibilities, allowed IDs, and safe text—not workflow transitions.

### 2.4 Role-aware pull and claim foundations

E19 already provides the relevant self-pull seam:

- [`internal/cli/commands/sprint.go`](../../../internal/cli/commands/sprint.go)
  exposes `shark sprint next --agent=<type>` and keeps the CLI adapter thin.
- [`internal/services/sprint_service.go`](../../../internal/services/sprint_service.go)
  implements `GetNextTask(ctx, agentType)` and filters by `agent_type` before
  deterministic priority/order selection.
- [`internal/repository/sprint/repository.go`](../../../internal/repository/sprint/repository.go)
  reads assigned entities, dependency data, and agent type; sprint readiness
  also evaluates dependency satisfaction.
- [`internal/services/claim_service.go`](../../../internal/services/claim_service.go)
  owns atomic claims, TTL/reclaim, heartbeats, conflict diagnostics, and safe
  session-scoped release.
- [`skills/shark-rider/skills/sprint-execution/`](../../../skills/shark-rider/skills/sprint-execution/)
  contains the existing solo and team pull-loop recipes and explicitly
  delegates actual per-entity work to `/run`/`/run-agent-team`.

Current `sprint next --agent` is the reusable role-filtered selection behavior,
but it is not yet the complete F04 `claim-next` contract: it returns a candidate
and does not atomically combine role selection with claim in one operation.
That missing mutation belongs to the E19 sprint/claim owner or E38 scheduler,
not to a markdown skill. F04 should document the intended sequence and the
ownership boundary, then call the available commands/services supported by the
consuming workflow.

**Extension decision:** extend the existing role-aware pull contract only if
E19/F02 explicitly requires an atomic `claim-next`; otherwise keep F04 as
portable guidance around `shark sprint next --agent` plus the existing claim
service. Do not implement claim logic in shell/markdown instructions.

### 2.5 Existing execution recipes and communication analogues

[`skills/shark-rider/skills/sprint-execution/SKILL.md`](../../../skills/shark-rider/skills/sprint-execution/SKILL.md)
and its `run-sprint*.md` workflows provide proven content patterns: front
matter, explicit preconditions, JSON-only Shark calls, loop exit conditions,
failure handling, and close gates. F04 should follow those conventions for
setup and role pull procedures, while replacing sprint-specific behavior with
root/child council behavior.

Shark already provides durable entity context and typed notes:
[`docs/cli-reference/context-commands.md`](../../../docs/cli-reference/context-commands.md),
[`docs/cli-reference/discovery-commands.md`](../../../docs/cli-reference/discovery-commands.md),
and entity note references in the epic/feature CLI docs. Use these for
entity-scoped decisions, blockers, and evidence links. They are not a
replacement for project-level council files, because F04 requires continuity
across refreshed workers and multiple roots.

There is no existing `docs/council/`, inbox implementation, handoff schema, or
escalation trigger file in this checkout. `docs/product/` currently contains
product integration/progress artifacts but not the required
`escalation_triggers.md`; setup must create or clearly mark that project-local
policy file as optional/configured input rather than silently assuming it.

**Extension decision:** reuse the existing content/procedure style and Shark
notes/context for entity links. Create the council directory/file contract as
new project-local protocol artifacts; do not store active inbox messages in
the database or claim tables for this feature.

## 3. Integration points

| Integration point | Existing path | F04 use | Classification |
|---|---|---|---|
| Embedded skill distribution | `internal/sharkdata/default_data/skills/`, `manifest.yaml`, `embed.go` | Ship `shark-attack` and validate identity/structure | Extend |
| Project customization | `shark-data/overrides/skills/` and install/upgrade commands | Allow replace-only project policy/role overrides | Reuse |
| Agent personas | `internal/sharkdata/default_data/agents/*.md` | Map roster IDs to existing personas | Reuse |
| Workflow role authority | `internal/config/workflow/schema.go`, `steps.go`, `workflow/*.yaml` | Explain precedence and derive role context | Reuse/document |
| Role-aware selection | `sprint.go`, `SprintService.GetNextTask`, sprint repository | Define self-pull behavior and `--agent` filtering | Extend only if atomic claim-next is required |
| Claim safety | `ClaimService`, claim repository, `entity_claims` | Reference safe claim/heartbeat/release ownership | Reuse; no new claim store |
| Team plan metadata | E38-F01 planned `TeamPlanItem`/ledger contract in sibling docs | Attach planned role and handoff references without changing plan authority | Consumer |
| Scheduler lifecycle | E38-F02 planned scheduler/claims contract | Emit role-specific instructions and bounded evidence/handoff paths | Consumer |
| Aggregate/resume | E38-F03 feature contract and architecture §4.5 | Preserve escalation/pause/handoff context for resume | Consumer |
| Reporting | E38-F05 feature contract | Supply bounded message/evidence pointers and next-action context | Producer |
| Project memory | `docs/council/` required by E38 but absent today | Add README, decisions, handoffs, escalations, inbox contract | New content |
| Escalation policy | `docs/product/` exists, trigger file absent | Define configurable trigger path and council-review route | New project artifact/template |

## 4. Inter-feature dependency map

```text
E38-F01 Team Plan / Ledger
        │ planned role + root/child identity
        ├──────────────► E38-F02 Scheduler / Claims
        │                        │ worker result + claim/session evidence
        │                        └────────► E38-F03 Aggregate / Resume
        │                                      │ outcome + next action
        └──────────────► E38-F04 Skill / Role Protocol ─────► E38-F05 Reporting
                              │ role, message, handoff, escalation contract
                              ├──────────────► F02 scheduler diagnostics
                              └──────────────► F03 resume/aggregate context

E19 sprint next/claim ───────► F04 role-aware self-pull guidance
E32 shark-data bundle ───────► F04 distribution and replace-only overrides
E16/E35 workflow metadata ───► role/step authority used by all consumers
```

The durable communication contract from architecture §4.5 should be stable and
small: sender role, recipient role, root/child key, subject, requested action
or question, urgency, evidence/artifact links, and created time. Inbox entries
are actionable and acknowledged/removed after action; decisions, handoffs,
unresolved questions, and escalation resolutions are copied to durable
directories. F02/F03/F05 should consume paths and metadata, not parse free-form
role prose.

## 5. Extension versus new analysis

| Component | Extend existing? | Recommendation and rationale |
|---|---:|---|
| Skill distribution | Yes | Add to embedded bundle/manifest and use existing install/override resolver |
| Role personas | Yes | Link existing agent files; add only roster collaboration metadata |
| Workflow assignment | Yes/reuse | Read workflow role metadata; roster never overrides it |
| Sprint role selection | Conditional | Reuse `sprint next --agent`; implement atomic claim-next only in E19/F02 owner if required |
| Claim/heartbeat/release | No new implementation in F04 | Delegate to `ClaimService`/scheduler; skill only states the contract |
| Team ledger | No | Consume F01 fields; do not add council columns/tables |
| Worker dispatch | No | Consume F02/F03 orchestration; no provider or process runtime |
| Council roster YAML | New | Add a project-local, validated declarative contract with role responsibilities and model preference only |
| Council memory layout | New | Add `docs/council/README.md`, `decisions/`, `handoffs/`, `escalations/`, and `inbox/<member-id>/` guidance/templates |
| Message/handoff schema | New content contract | Keep bounded, structured front matter and artifact links; preserve root/child scope |
| Escalation procedure | New content contract | Use configurable trigger document and council review routing; do not hard-code a human destination |
| CLI mechanics | Reuse/document | Link to existing `shark`, `/shark-rider`, sprint, notes, context, and claim commands; do not duplicate command behavior |
| Runtime roster validation | Probably new narrow validator | Add only if acceptance tests require machine validation; place it in shark-data validation and keep it schema-focused |

## 6. Recommended implementation approach

1. Add `shark-attack/SKILL.md` under the embedded default skill tree. Keep it a
   recipe: prerequisites/setup, roster loading, chair-led collaboration,
   role-aware self-pull, message lifecycle, handoffs, escalation, resume
   context, and security/ownership rules. Reference existing Shark Rider and
   sprint skills for command mechanics.
2. Add a canonical roster template/schema under the skill (or a clearly
   documented project template path). Validate required fields: team name,
   chair, memory root, inbox root, communication flags, escalation route, and
   member IDs/roles/responsibilities. Treat model tier as a preference.
3. Add `manifest.yaml` skill identity metadata and content-validation coverage
   for front matter, referenced paths, roster examples, and no missing agent
   references. Verify with `shark admin validate-data` after materializing the
   bundle in a fixture.
4. Define the project-local council contract and setup guidance. The minimum
   layout is:

   ```text
   docs/council/
   ├── README.md
   ├── decisions/
   ├── handoffs/
   ├── escalations/
   └── inbox/<member-id>/
   ```

   Make private council content optionally gitignored, but keep durable
   decisions and resolutions available to refreshed workers.
5. Keep messages structured and bounded. Require root/child identity and
   artifact references; reject credentials, rendered prompts, unrestricted
   stdout, and arbitrary secret-bearing content. Let F05 display pointers and
   let F03 use unresolved escalations as pause/next-action context.
6. Specify role-aware pull as a contract: derive the worker role from workflow
   metadata, select by existing priority/dependency order, then claim through
   the owning Shark service. Explicitly state that an `agent` field or roster
   entry cannot override workflow-defined responsibility.
7. Add cross-feature contract tests or content fixtures for I-04 and UAT-08
   through UAT-12. Verify distribution/override behavior, refreshed-worker
   memory continuity, message acknowledgment/retention, escalation routing,
   and role-filtered selection with claim ownership.

## 7. Risks, constraints, and knowledge gaps

- **Bundle path mismatch:** user instructions mention `shark-data/skills/`, but
  this source checkout ships content from `internal/sharkdata/default_data/`.
  Implementation must update the embedded source and validate materialized
  output; editing only a generated `shark-data/` directory would not ship.
- **Role vocabulary mismatch:** existing persona IDs are names such as
  `tech-director` and `business-analyst`, while the epic roster example uses
  abstract roles such as `chair` and `requirements`. Define explicit mapping;
  do not rely on string similarity.
- **Atomic claim-next ownership is unresolved:** current `sprint next --agent`
  filters/selects but does not itself claim. F04 can document the desired
  behavior, but implementation authority must be assigned to E19/F02 before
  adding a command.
- **Escalation policy input is absent:** `docs/product/escalation_triggers.md`
  does not exist today. The skill must define fallback behavior (pause and
  route to council review) and a setup instruction for projects to supply it.
- **F01/F02 code availability:** the sibling reports describe team contracts,
  but the current main checkout has no `internal/team` package. F04 should
  consume their finalized public contract, not import provisional paths or
  duplicate it.
- **Private memory handling:** decide whether setup creates empty directories,
  `.gitkeep` files, or only documents the contract. Do not require private
  deliberation to be committed, but do not make required resume information
  inaccessible to refreshed workers.

## 8. Exit-gate assessment

- Existing related code identified with file paths: **yes**.
- Extension points documented: **yes** (bundle, personas, workflow roles,
  sprint pull, claims, notes/context, and validation).
- Inter-feature dependencies mapped: **yes** (F01/F02/F03/F05 plus E19/E32).
- New-vs-extension decisions made: **yes**.
- Actionable for architect/specification: **yes**.

**Recommended outcome: pass — ready for architecture/specification.**
