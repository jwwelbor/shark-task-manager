---
epic_key: E38
title: Shark Attack Team Orchestration
description: Coordinate multiple specialized workers against an epic or feature while preserving Shark workflow routing, leases, dependency order, quality gates, and resumable execution.
last_updated: 2026-07-13
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
assembly, claims, and outcome-based status routing are the authoritative control
surfaces. Attack-team orchestration must use those contracts while adding a
team-level plan, bounded execution, and an aggregate result. Workers perform
scoped craft; the parent orchestrator retains ownership of leases and workflow
transitions.

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
should reduce elapsed time for independent work, improve coverage from parallel
specialists, and preserve trust in Shark’s workflow history when a member fails,
pauses, or needs to resume.

## 2. Goals and success criteria (measurable)

### Goals

1. Give operators one explicit attack-team entry point for an epic or feature
   whose existing child work can benefit from multiple specialized workers.
2. Build every team plan from Shark’s existing child entities, dependencies,
   workflow steps, agent metadata, and prompt-assembly contract.
3. Run independent work concurrently when the host supports bounded concurrency;
   serialize dependent work and protect shared files and claims.
4. Produce a team result that accounts for every dispatched child, records each
   worker outcome, and routes the root through the configured workflow rather than
   guessing a status.
5. Make interruption, worker failure, provider unavailability, and re-entry safe
   and diagnosable.
6. Define the reusable `shark-attack` skill as the distributed recipe for team
   setup, role collaboration, delegation, communication, memory, and
   escalation, with the existing Shark skill referenced for CLI mechanics.
7. Keep council memory at project scope under `docs/council/`, with optional
   project-level gitignore treatment for teams that do not want to commit it.
8. Make role-aware self-pull the normal assignment model: workers pull the next
   priority item for their workflow-defined role, then claim it; the Scrum
   Master plans and monitors the process but does not pre-assign every task.

### Success criteria

| Measure | Target | Verification |
|---|---:|---|
| Team-plan coverage | 100% of eligible non-terminal children appear exactly once in the plan, with dependency and assigned-worker metadata | Fixture plan comparison |
| Duplicate work protection | 0 duplicate active claims for a child during a team run; already-claimed children are reported and not dispatched again | Concurrent-claim test |
| Workflow integrity | 0 worker-owned status transitions; 100% of root transitions are accepted through configured outcome routing | Transition-history audit |
| Dependency safety | 100% of dependent children wait until required predecessors reach their configured success outcome; blocked predecessors dispatch no dependent child | Dependency fixture with a blocked predecessor |
| Parallel efficiency | For a fixture with at least three independent children, median wall-clock execution is at least 25% lower than the sequential baseline when concurrency is available | Repeated timed integration run |
| Failure containment | In 100% of injected worker-failure runs, unaffected independent children retain their recorded outcomes, dependent children are not falsely marked complete, and the root does not reach a success terminal state | Failure-injection test |
| Resume safety | Two consecutive runs after an interruption create 0 duplicate child completions or notes and dispatch only unfinished eligible work | Stop-and-resume fixture |
| Operator diagnosability | 100% of completed, paused, and failed runs report the root, child count, per-child outcome, skipped reason, and next recommended action | Output/schema assertions |

### Locked design decisions

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
  role. `claim-next` is the role-aware combination of selecting the next item
  and claiming it, and `shark sprint next` must be corrected to provide that
  behavior. The legacy database `agent` assignment concept is not part of
  sprint planning and should not be revived unless a future requirement adds it.
- **Escalation contract**: existing quality skills may identify a rejection or
  blocker, but the team protocol converts that into an escalation-for-review
  event handled by the Tech Director/council routing. Contracts remain
  single-responsibility: workers report evidence, the council decides or
  escalates, and Shark advances workflow state.

## 3. Scope: in-scope and out-of-scope boundaries

### In scope

- An explicit attack-team orchestration surface for an epic or feature root.
- A previewable team plan that lists eligible children, worker roles and
  providers from workflow metadata, dependency waves, concurrency limits, and
  children excluded from dispatch with a reason.
- Dispatch of existing Shark-rendered prompts through the host execution adapter;
  the team layer may coordinate workers but must not rebuild their prompts.
- Dependency-aware scheduling, bounded concurrency, cancellation, and safe
  fallback to sequential execution when team capability is unavailable.
- Claim and lease coordination so one child has at most one active worker, with
  clear handling for expired, conflicting, or unavailable claims.
- Aggregation of child outcomes into a semantic team result that follows the
  root’s configured workflow outcomes.
- Explicit handling for partial completion, worker failure, provider failure,
  human approval or pause gates, cancellation, and resumed execution.
- A durable, machine-readable execution summary and concise operator-facing
  reporting sufficient to diagnose what ran and what remains.
- Compatibility with the current single-worker dispatch path; a root or child
  that does not qualify for team execution continues to use the normal path.
- The `shark-attack` skill, its communication/memory procedure, role roster YAML,
  project council-memory layout, and explicit setup/bootstrap guidance.
- The smallest Shark changes needed for role-aware `sprint next`, `claim-next`,
  current-session/owner visibility, and compatibility with the skill’s team
  communication protocol.

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
- The parent orchestration loop owns the root lease and all root transitions.
  A worker returns a semantic outcome and evidence; it does not transition the
  dispatched entity.
- The implementation must honor the repository’s local-worktree safety rules
  and must not dispatch work when branch, worktree, or required host capability
  checks fail.
- Concurrency is bounded by an explicit team limit and by detected dependency or
  shared-resource conflicts. A host without safe team execution must use the
  defined sequential fallback and identify that degraded mode to the operator.
- Team execution is resumable from persisted Shark state. In-memory scheduler
  state is not sufficient to declare work complete.
- Existing workflow configuration may contain pause, archive, human, and
  provider-specific steps. The team must stop or route at those boundaries
  instead of treating every child as parallel agent work.
- Output must distinguish completed, failed, blocked, skipped, cancelled, and
  not-yet-eligible children. A partial result must never be presented as a
  successful team completion.

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
testing, review, and other specialized work under a shared root. They must review
the team plan, choose or accept the concurrency limit, and respond when the run
pauses or returns a partial result. Ordinary single-worker execution remains
available for small or tightly coupled work.

### Product managers and business analysts

They receive clearer progress and failure summaries for large initiatives. They
can see which child outcomes are complete, blocked, or awaiting a human decision
without inferring progress from an optimistic root status. They do not need to
manage worker claims or provider details.

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

Maintainers gain a single coordination contract to test across providers and host
adapters. Administrators may need to configure team limits and ensure the chosen
host execution capability is available. No new database or cross-project service
is required by this epic.

## 6. High-level acceptance criteria (UAT scenarios)

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

### UAT-12: Pull and claim by workflow role

**Given** a sprint contains priority-ordered work for architecture,
implementation, and QA

**When** a role worker requests the next item

**Then** `shark sprint next`/`claim-next` selects only work eligible for that
workflow-defined role, atomically identifies the current session owner, and
returns the item with its Shark-rendered step prompt; the legacy database
`agent` assignment field does not override the workflow role.
