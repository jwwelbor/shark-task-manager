---
epic_key: E38
title: Shark Attack Team Orchestration
description: Provide a reusable role-based team skill and Rider procedure for coordinating specialized workers around existing Shark workflow, claims, prompts, quality gates, and escalation paths.
last_updated: 2026-07-26
---

# Shark Attack Team Orchestration

**Epic Key**: E38

## 1. Problem statement and business justification

Shark can route a workflow step to one worker and can group sprint work through
team-oriented skills, but it does not provide one consistent, Shark-owned way to
coordinate several specialized workers against a complex feature or epic. Teams
currently rely on manual coordination or host-specific team behavior to decide
which workers run, which work can run together, how dependencies are honored, and
how partial results return to the workflow. That creates duplicated context,
avoidable idle time, race conditions around shared files, and incomplete or
ambiguous handoffs.

The existing `shark next` dispatch contract, workflow configuration, prompt
assembly, claims, role filtering, and outcome-based status routing are the
authoritative control surfaces. Shark Attack uses those contracts through a
host-side Rider procedure and a reusable prompt/Markdown skill. It does not add
a team ledger, scheduler, aggregate router, resume store, or autonomous runtime.
Workers perform scoped craft; the parent procedure retains ownership of leases
and workflow transitions.

The team is a hierarchy, not a flat pool of agents. The Tech Director is the
council chair and owns routing, tie-breaks, and filtering of escalations. The
Product Manager keeps work aligned to the product direction and challenges
scope drift. The Architect owns technical direction and development
specifications. The Business Analyst owns requirements, decomposition, and
acceptance criteria. The Scrum Master owns process, sprint order, agent
check-ins, tracking updates, and escalation of stalled work. Developers and QA
workers are short-lived specialists: they pull work appropriate to their role,
claim it, produce the artifacts required by their skill, and their session ends
when the task ends. Quality is judged at the workflow quality gate, not by an
implementation worker declaring its own work approved.

The Tech Director does not become the Shark process owner. Shark remains the
source of workflow state, claims, leases, and history; the Tech Director uses
those surfaces to understand who owns work and where a question belongs. When
an answer is not in the product definition or recorded decisions, the Tech
Director consults the relevant experts, makes the tie-break when appropriate,
records it durably, and escalates only decisions that materially affect product
direction or outcome. Escalations are raised for review; the protocol does not
hard-code a particular human escalation target.

This epic is justified because reliable coordination makes specialized agents a
repeatable delivery capability rather than a collection of ad-hoc prompts. It
should make roles, questions, handoffs, and escalation decisions explicit while
preserving the ordinary Shark workflow as the single source of truth.

## 2. Goals and success criteria (measurable)

### Goals

1. Define the reusable `shark-attack` skill as the distributed recipe for team
   setup, role collaboration, delegation, communication, memory, and
   escalation, with the existing Shark skill referenced for CLI mechanics.
2. Keep council memory at project scope under `docs/council/`, with optional
   project-level gitignore treatment for teams that do not want to commit it.
3. Make role-aware self-pull the normal assignment model: workers pull the next
   priority item for their workflow-defined role, then claim it; the Scrum
   Master plans and monitors the process but does not pre-assign every task.
4. Define a repeatable Rider loop around `shark next`, claim, dispatch, outcome,
   advance, release, and explicit escalation.

### Success criteria

| Measure | Target | Verification |
|---|---:|---|
| Role correctness | 100% of role-aware pulls return only work eligible for the requesting workflow role | Role-filter fixture |
| Claim safety | 0 duplicate active claims; a race produces one winner and one conflict without force-stealing | Concurrent claim test |
| Prompt fidelity | 100% of dispatched workers receive the ordinary `shark next` prompt unchanged | Dispatch-path test |
| Ownership integrity | 0 worker-owned transitions, heartbeats, or releases for the dispatched parent entity | Transition/claim history audit |
| Escalation clarity | 100% of unresolved questions identify evidence, responsible role, decision needed, and next owner | Escalation artifact review |
| Handoff continuity | A refreshed worker can continue from durable council decisions and handoffs without prior chat context | Council-memory fixture |

### Locked design decisions

> Direction reset: the runtime-oriented F01–F03 design was superseded on
> 2026-07-13. Their documents remain as historical design records only. F04,
> F06, and F07 delivered the first Shark Attack protocol tranche; the approved
> v2 triage re-scoped F05 and added F08–F11 as the active roadmap. Completing
> the first tranche does not approve the open implementation decisions recorded
> in the v2 plan or satisfy the future v2 qualification.

- **Team artifact**: the primary deliverable is an elaborate, reusable set of
  prompts and Markdown skill procedures, not a new autonomous AI runtime. Any
  Shark changes are limited to the seams needed to make the existing workflow,
  sprint, claim, and session contracts support those recipes.
- **Skill ownership**: `shark-attack` owns the team verb/sub-skill for
  communication and memory. It may reference Shark shorthand and should be
  distributable with the project’s standard `shark-data/overrides/` mechanism;
  it must not duplicate the core Shark skill.
- **Team configuration**: a project may define its council and specialist
  roster in YAML. The YAML records role, responsibility, communication
  behavior, escalation authority, and model/tier preference (for example,
  `tech-director` may prefer Fable, Opus, or GPT-5.6 Sol). It is configuration
  for the skill’s team recipe, not a replacement for workflow YAML or a second
  source of Shark status transitions.
- **Product prerequisite**: the team cannot reliably self-direct without a
  product definition. Setup recommends the full product-design/bootstrap path
  and requires at least the early product-definition gates (D01-D02, with D05
  when the project needs the additional context) before relying on autonomous
  routing; missing context increases escalation rather than being guessed.
- **Communication and memory**: council memory is project-level, while active
  agent communication uses durable inbox-style messages under the council
  area. Members read and remove/acknowledge inbox items after acting on them;
  decisions, handoffs, and unresolved questions remain in the durable record.
- **Assignment and lifecycle**: Shark workflow metadata determines a worker’s
  role. The sanctioned role-aware procedure first selects the next eligible
  item with `shark sprint next --agent=<type>`, then submits that exact item to
  the owning claim path. Selection is read-only; `ClaimService` remains the
  atomic lease and conflict boundary. A future atomic `claim-next` operation,
  if required, belongs to the sprint/claim owner rather than this epic. The
  legacy database `agent` assignment concept is not part of sprint planning and
  should not be revived unless a future requirement adds it.
- **Escalation contract**: existing quality skills may identify a rejection or
  blocker, but the team protocol converts that into an escalation-for-review
  event handled by the Tech Director/council routing. Contracts remain
  single-responsibility: workers report evidence, the council decides or
  escalates, and Shark advances workflow state.

## 3. Scope: in-scope and out-of-scope boundaries

The active implementation boundary is provider-neutral skill and Rider
coordination around existing Shark commands, plus the narrow deterministic data
tooling and validation required by the approved v2 F05 scope. It does not add a
team runtime, scheduler, ledger, aggregate router, or second lifecycle store.
The former planner/scheduler/ledger/aggregate bullets below are retained only as
historical context and are not implementation commitments.

### In scope

- The distributable `shark-attack` skill, role roster, council memory,
  communication, handoff, and escalation procedures.
- A role-aware pull/claim recipe using existing Shark filters, named selectors,
  claim leases, heartbeat, and release behavior.
- A Rider procedure around `shark next`, host-agent dispatch, worker outcomes,
  configured status advancement, and clean stop conditions.
- Deterministic council artifacts and validation, with only the thin Shark
  admin surface needed to create and validate those artifacts.
- Provider-neutral question routing, same-worker resume where supported, and
  bounded replacement-worker fallback where it is not.
- Integrity prerequisites, adapter conformance evidence, and a separately
  approved complicated-lifecycle qualification run for Shark Attack v2.
- Clear ownership rules: Rider owns claims and parent transitions; workers own
  craft and evidence; the council owns questions, decisions, and escalation.
- Compatibility with ordinary Shark CLI behavior and existing single-worker
  dispatch.
- The `shark-attack` skill, its communication/memory procedure, role roster YAML,
  project council-memory layout, and explicit setup/bootstrap guidance.
- The smallest Shark change needed to correct role/agent filtering, if a live
  gap is found during F06 verification.

### Out of scope

- Automatic creation, decomposition, reprioritization, or rewriting of epics,
  features, tasks, or dependencies. The team consumes existing Shark entities.
- Replacing or weakening workflow definitions, configured outcomes, quality
  gates, human approval, claim ownership, or the `shark next` prompt contract.
- Allowing workers to claim, heartbeat, release, advance, or directly set the
  dispatched root’s workflow state.
- A new AI model, provider, native agent-team runtime, or credential-management
  system. Provider adapters remain host concerns.
- Cross-project or distributed execution across multiple Shark databases or
  machines.
- Automatic merging of branches, conflict resolution, or approval of code,
  requirements, or releases.
- A web dashboard, notification service, cost accounting system, or long-term
  analytics product. The epic requires execution reporting, not a new UI.
- Changing the existing behavior of ordinary single-entity `/run` execution or
  sprint planning and close operations.
- Making the Tech Director responsible for Shark workflow transitions or
  embedding a fixed human escalation destination in the protocol.
- Treating model/tier preferences in the roster YAML as mandatory provider
  availability or as a replacement for the workflow’s resolved agent metadata.

### Team structure and YAML contract

The skill must make the hierarchy visible before it starts work. The canonical
shape is a chair-led council plus role specialists:

```text
Tech Director (chair: routing, tie-breaks, escalation filter)
├── Product Manager (product direction and scope)
├── Architect (technical direction and specifications)
├── Business Analyst (requirements and decomposition)
├── Scrum Master (process, sprint order, monitoring, tracking)
└── Delivery specialists
    ├── Developer (implementation)
    ├── QA (verification and red-team testing)
    └── other workflow-defined specialists or human gates
```

The project roster YAML is declarative. A representative contract is:

```yaml
team: shark-attack
chair: tech-director
memory_root: docs/council
communication:
  inbox_root: docs/council/inbox
  acknowledge_after_read: true
  retain_decisions: true
escalation:
  triggers_file: docs/product/escalation_triggers.md
  route: council-review
members:
  - id: tech-director
    role: chair
    responsibilities: [route, consult, tie_break, record_decision]
    model_tier: opus
  - id: product-manager
    role: product
    responsibilities: [personas, goals, scope_guard]
  - id: architect
    role: architecture
    responsibilities: [stack, design, dependencies, specifications]
  - id: business-analyst
    role: requirements
    responsibilities: [stories, acceptance_criteria, decomposition]
  - id: scrum-master
    role: process
    responsibilities: [pull, monitor, sprint_order, tracking]
  - id: developer
    role: implementation
    responsibilities: [implement, test, report_artifacts]
  - id: qa
    role: quality
    responsibilities: [verify, red_team, report_findings]
```

The exact model tier is a preference and may be overridden by the host or
workflow metadata. The YAML does not assign Shark statuses, create backlog
items, or authorize a member to bypass claims. Workflow YAML remains the source
of truth for the step’s resolved responsibility, agent, provider, model, and
outcomes; the team roster explains how those roles collaborate around it.

### Project council memory and communication contract

The skill must define this project-local layout:

```text
docs/council/
├── README.md                 # protocol and index
├── decisions/                # durable council and tie-break decisions
├── handoffs/                 # concise worker-to-worker context
├── escalations/              # review requests and resolutions
└── inbox/<member-id>/        # actionable messages; ack/remove after action
```

Every message identifies sender, recipient role, root/child key, subject,
question or requested action, evidence/artifact links, and urgency. Inbox files
are actionable queue entries: a member acknowledges or removes one after
acting, while any resulting decision, handoff, or unresolved question is copied
to the durable area. This keeps active communication lightweight without
losing the context needed by a refreshed developer or QA worker.

The escalation trigger document is product/project policy, not a fixed human
destination. A worker escalates when the answer cannot be derived from product
docs or recorded decisions, when a material product or architecture direction
would change, when experts disagree, or when a process/quality blocker cannot
be resolved within the worker’s responsibility. The Tech Director filters the
request, consults the appropriate role, records a tie-break when authorized,
or routes it for review when it must remain unresolved.

## 4. Constraints and assumptions

### Constraints

- Shark remains the source of truth for entity state, workflow routing,
  dependency data, prompt assembly, claims, and outcome history.
- Rider owns the parent procedure’s claim, heartbeat, release, and configured
  status transition. A worker returns craft evidence and a semantic outcome.
- The skill must honor local worktree safety rules and stop when the ordinary
  workflow returns pause, archive, error, or a human gate.
- Role roster preferences never override workflow YAML, claim ownership, or
  configured outcomes.
- Handoffs and escalation records must be bounded and must not contain prompts,
  credentials, or unrestricted transcripts.

### Assumptions

- “Attack team” means a coordinated group of specialized workers focused on one
  existing epic or feature outcome; it does not mean an autonomous planner that
  invents backlog items.
- A team root has a finite set of existing child entities and dependency data
  that Shark can read before dispatch. Missing or ambiguous relationships are a
  plan-time error, not permission to guess.
- The host can expose a stable execution primitive and can report worker start,
  completion, failure, and interruption. Shark does not need to own the worker
  process itself.
- Existing single-worker workflows remain the compatibility baseline. Team mode
  is opt-in or explicitly selected and cannot silently change ordinary runs.
- Operators accept a preview and understand that parallel workers can modify
  different files concurrently only when the plan marks that work safe.
- Product and council memory may be project-local and optionally ignored by the
  project; the skill must define the file contract without requiring every
  project to commit private council deliberation.

## 5. Stakeholder impact

### Developers and operators

Developers gain one repeatable way to coordinate architecture, implementation,
testing, review, and other specialized work under a shared root. They pull or
receive scoped work, use ordinary claims, and escalate questions through the
defined role path. Ordinary single-worker execution remains available.

### Product managers and business analysts

They receive clearer handoffs, decisions, blockers, and escalation requests.
They do not need to manage worker claims or provider details.

### Architects, reviewers, and QA specialists

Their roles become explicit team assignments derived from workflow metadata.
Parallel specialists can increase coverage, but review and approval gates remain
visible workflow boundaries and cannot be silently skipped by a successful
implementation worker.

### AI workers and host execution adapters

Workers receive one scoped, fully assembled prompt and return work evidence plus a
semantic outcome. They lose responsibility for orchestration decisions and direct
status mutation. Host adapters must support the team’s bounded scheduling and
reporting contract, or the system must use the documented sequential fallback.

### Shark maintainers and project administrators

Maintainers gain a single coordination contract to test across host adapters.
Administrators configure the role roster and council layout; no new database,
team runtime, or cross-project service is required by this epic.

## 6. Historical runtime UAT scenarios

> The scenarios in this section describe the rejected planner/scheduler/ledger
> runtime and are retained for traceability only. They are not active E38
> acceptance criteria. The active acceptance plan is `uat-plan.md`, focused on
> F04–F07 and ordinary Shark CLI behavior.

### UAT-1: Preview a complete team plan

**Given** feature `E38-F01` has four non-terminal child tasks, two independent
tasks, one task that depends on both, and one task assigned to a human approval
step

**When** the operator requests an attack-team preview

**Then** Shark lists every eligible child exactly once, shows its configured
worker role and provider, groups independent work into an executable wave,
places the dependent task after its predecessors, identifies the approval task as
a pause boundary, and performs no claim, status, or file mutation.

### UAT-2: Execute independent work with dependency ordering

**Given** the operator confirms the preview and the host supports a concurrency
limit of two

**When** the team starts

**Then** the two independent workers may run at the same time, the dependent
worker does not start before both predecessors return their configured success
outcome, every worker receives the Shark-rendered prompt for its own child, and
the team summary records start, completion, and outcome data for all dispatched
children.

### UAT-3: Protect an already-claimed child

**Given** one planned child is actively claimed by another session

**When** the attack team builds or refreshes its dispatch plan

**Then** it does not steal or dispatch that child, reports the claim conflict with
the child key and current availability, and leaves unrelated eligible children
available for execution.

### UAT-4: Contain a worker failure

**Given** one independent worker returns a failure and a second independent
worker succeeds, while a third child depends on the failed worker

**When** the team aggregates the results

**Then** the successful child remains recorded as successful, the dependent child
is not dispatched or marked complete, the root is routed through its configured
failure or partial outcome, and the operator receives the failed child and a
specific next action.

### UAT-5: Stop at a human or quality gate

**Given** a child reaches a configured human approval, pause, or quality-gate
step

**When** the team reaches that step

**Then** the team records the gate, does not auto-approve or bypass it, preserves
all completed child outcomes, and reports the exact decision required before
resume.

### UAT-6: Resume after interruption without duplicate work

**Given** a team process stops after two of five children complete and persists
their outcomes

**When** the operator invokes the same team root again

**Then** the resumed plan recognizes the two completed children, does not create
duplicate completions or notes, reconciles stale claims according to the lease
rules, dispatches only unfinished eligible children, and produces one coherent
aggregate result.

### UAT-7: Handle unavailable team capability

**Given** the selected host cannot provide safe bounded team execution

**When** the operator requests team execution

**Then** Shark either uses the documented sequential fallback or stops before
mutation with an actionable capability error, clearly identifies which mode was
used, and never reports parallel execution that did not occur.

### UAT-8: Preserve ordinary single-worker behavior

**Given** an operator invokes the existing single-entity run path for a child that
does not qualify for attack-team execution

**When** the workflow dispatches that child

**Then** the child follows the existing `shark next` prompt, lease, outcome, and
status-transition contract with no team-only side effects.

### UAT-9: Load a chair-led team from YAML

**Given** a project has the `shark-attack` roster YAML with a Tech Director
chair, Product Manager, Architect, Business Analyst, Scrum Master, Developer,
and QA members, including model/tier preferences

**When** the team is initialized

**Then** the skill presents the hierarchy and responsibilities, resolves model
preferences as non-binding host hints, keeps workflow YAML authoritative for
step routing, and refuses to treat roster membership as permission to mutate
Shark status or bypass a claim.

### UAT-10: Preserve council memory across refreshed workers

**Given** the council has a decision, a worker handoff, and an actionable inbox
message under `docs/council/`

**When** a Developer or QA worker is refreshed for a later task

**Then** it can read the relevant durable decision and handoff, the acted-on
inbox item is acknowledged or removed, and the durable decision/history remains
available without requiring the prior worker conversation.

### UAT-11: Filter and route an escalation

**Given** a worker raises a question that is not answered by product docs or
recorded decisions and could materially change the outcome

**When** the team processes the escalation

**Then** it records the evidence and trigger, routes the question to the
appropriate council role through the communication protocol, lets the Tech
Director consult or make a documented tie-break, and raises it for review when
it cannot be safely resolved without naming a fixed human destination.

### Historical UAT-12: superseded pull-and-claim wording

**Given** a sprint contains priority-ordered work for architecture,
implementation, and QA

**When** a role worker requests the next item

**Then** the worker uses `shark sprint next --agent=<type>` to select only work
eligible for that workflow-defined role, and the owning claim path separately
claims that selected item. The claim path, not selection, identifies the
session owner and reports a live-claim conflict; the legacy database `agent`
assignment field does not override the workflow role. The active UAT plan
supersedes this historical scenario with UAT-01 and UAT-02.
