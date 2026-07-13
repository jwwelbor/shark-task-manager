# E38 Brownfield Research Report: Shark Attack Team Orchestration

**Scan date**: 2026-07-13  
**Scope**: Existing implementation and feasibility for coordinating multiple
workers under an epic or feature root.  The E38 PRD is in
[`epic.md`](./epic.md).

## Executive conclusion

E38 should extend the existing Shark 2.x dispatch contract rather than create a
second prompt, workflow, claim, or single-worker runner.  The repository already
has:

- `shark next`, which resolves one dispatchable child through a parent cascade,
  renders the complete prompt, and filters live claims;
- `shark run`, which owns one root lease, heartbeats it, dispatches an external
  provider, and loops through configured workflow actions;
- route-based workflow metadata with semantic outcomes, agent/provider/model
  routing, skills, prompts, pause/terminal boundaries, and cascade actions;
- atomic per-entity claims with TTL reclamation and work-session journaling;
- task dependency data, validation, relationship-backed dependency reads, and
  sprint readiness code that already parses dependency graphs.

What does not exist is a team-level plan and durable execution record.  Current
parent cascade is deliberately first-child/one-step selection, not a complete
plan of all children or a parallel scheduler.  Current `RunController` is a
single-entity status loop, not a multi-child coordinator.  E38 therefore needs a
new team orchestration service/package and CLI/API surface, while reusing the
existing per-child `next`/prompt/dispatcher/claim/transition seams.

The recommendation is an extend-first design with a small durable team-run
ledger.  Do not add a new AI provider runtime or teach workers to mutate Shark
state.  The team parent owns the root lease and root transitions; each child
worker receives the existing Shark-rendered prompt and returns an outcome to the
team coordinator.

## 1. Existing implementations relevant to E38

### 1.1 Epic scope and explicit constraints

The E38 PRD establishes that the authoritative surfaces are `shark next`,
workflow configuration, prompt assembly, claims, and outcome routing
([`epic.md:18-31`](./epic.md#L18-L31)).  It also explicitly excludes backlog
creation/decomposition, new providers, cross-database execution, automatic
merges, and weakening workflow or claim ownership ([`epic.md:83-119`](./epic.md#L83-L119)).

The acceptance criteria require all children to appear exactly once in a preview,
dependency waves, claim conflict reporting, worker-failure containment, human
pause boundaries, resumability, and sequential fallback
([`epic.md:169-255`](./epic.md#L169-L255)).  These requirements are not satisfied
by the existing single-child cascade alone.

### 1.2 Canonical single-step dispatch: `shark next`

[`internal/cli/commands/next.go:104-149`](../../../internal/cli/commands/next.go#L104-L149)
defines the stable JSON contract: entity key/type/status, action, agent type,
provider, model, effort, and a fully rendered prompt.  It also exposes
`resolved_via` for audit of parent cascade traversal.

The resolver reads status and terminal state, generates placeholders, populates
the configured action, handles legacy/unknown statuses as a pause, and assembles
the agent body into the prompt
([`next.go:272-411`](../../../internal/cli/commands/next.go#L272-L411)).
This is the prompt and action contract E38 should call or factor behind a
service-level adapter; it must not reconstruct prompts from raw workflow YAML.

The existing cascade implementation is important but limited: it iterates
children, skips live claims, recursively finds the first `spawn_agent`, and
returns one wire response; when all children are terminal it auto-advances the
parent ([`next.go:418-527`](../../../internal/cli/commands/next.go#L418-L527)).
It does not return every child, compute dependency waves, reserve a concurrency
budget, or persist a team plan.

### 1.3 Existing single-entity execution: `shark run`

The CLI command claims the normalized root before running, starts a heartbeat,
builds entity-specific transition and placeholder adapters, selects Claude/Codex
dispatchers, and releases the root lease on every return path
([`internal/cli/commands/run.go:75-223`](../../../internal/cli/commands/run.go#L75-L223)).
The lease heartbeat and session-scoped release are implemented at
[`run.go:260-337`](../../../internal/cli/commands/run.go#L260-L337).

`RunController` is intentionally single-entity.  Its loop reads current status,
gets a populated action, handles pause/archive/advance/spawn, dispatches one
agent, gates advancement on success, and stops on terminal/pause/failure
([`internal/runner/controller.go:175-245`](../../../internal/runner/controller.go#L175-L245)).
Its injected seams are reusable: `EntityTransitioner`,
`PlaceholderGenerator`, `ActionService`, `PromptAssembler`, and a provider-keyed
`AgentDispatcher` map ([`controller.go:98-173`](../../../internal/runner/controller.go#L98-L173)).

The dispatcher contract already captures instruction, work directory, entity
metadata, provider model/effort-related input, and process results.  The shared
executor captures stdout/stderr, duration, exit code, and returns structured
failure errors ([`internal/runner/dispatcher.go:29-118`](../../../internal/runner/dispatcher.go#L29-L118),
[`dispatcher.go:159-228`](../../../internal/runner/dispatcher.go#L159-L228)).
Claude and Codex implementations are isolated behind this interface
([`claude_dispatcher.go:21-91`](../../../internal/runner/claude_dispatcher.go#L21-L91),
[`codex_dispatcher.go:20-86`](../../../internal/runner/codex_dispatcher.go#L20-L86)).

### 1.4 Workflow, prompt, and outcome configuration

Legacy `OrchestratorAction` already carries action, agent type, provider, model,
effort, skills, and an instruction template, with validation for action-specific
requirements ([`internal/config/action/orchestrator.go:13-105`](../../../internal/config/action/orchestrator.go#L13-L105)).
Template population supports both `.tmpl`/`.md` files and legacy inline
replacement ([`orchestrator.go:207-232`](../../../internal/config/action/orchestrator.go#L207-L232)).

The route-based `Step` schema is the more important E38 source of truth.  It
contains responsibility, agent types, action, provider, model, effort, skills,
prompt, semantic outcomes, pause/terminal flags, and aggregation metadata
([`internal/config/workflow/schema.go:106-212`](../../../internal/config/workflow/schema.go#L106-L212)).
The team planner should read these fields to classify worker steps versus human,
quality-gate, pause, archive, and auto-advance boundaries.  It must route the
root using configured outcomes, never infer “success means completed.”

### 1.5 Claims, leases, and sessions

`EntityClaim` is a unique per-entity lease with claimed-by/session identity,
heartbeat, progress, and note; TTL expiry is the crash-recovery mechanism
([`internal/models/entity_claim.go:8-25`](../../../internal/models/entity_claim.go#L8-L25)).
The claim repository enforces the unique single-grab with a database constraint
and provides session-scoped release, renewal, expiry reclamation, and listing
([`internal/repository/claim/claim_repository.go:31-64`](../../../internal/repository/claim/claim_repository.go#L31-L64),
[`claim_repository.go:81-138`](../../../internal/repository/claim/claim_repository.go#L81-L138)).

`ClaimService` owns TTL policy, reclaim-before-claim, conflict diagnostics,
heartbeat, and claimability checks ([`internal/services/claim_service.go:18-33`](../../../internal/services/claim_service.go#L18-L33),
[`claim_service.go:92-153`](../../../internal/services/claim_service.go#L92-L153),
[`claim_service.go:220-250`](../../../internal/services/claim_service.go#L220-L250)).
Claim-created sessions are opened and closed best-effort for telemetry
([`claim_service.go:181-217`](../../../internal/services/claim_service.go#L181-L217)).

The database history confirms `entity_claims` was added for route-based dispatch
and `work_sessions` was made entity-generic for all entity types
([`internal/db/db.go:436-459`](../../../internal/db/db.go#L436-L459)).
These are the correct primitives for child claim protection, but neither stores a
team run ID, plan membership, wave, skip reason, worker outcome, or aggregate
resume state.

### 1.6 Children, dependencies, and existing scheduling-adjacent code

Tasks carry `agent_type`, `depends_on`, assigned agent, blocked reason, execution
order, and completion metadata
([`internal/models/task.go:11-40`](../../../internal/models/task.go#L11-L40)).
Task repositories can build a dependency graph, validate cycles, get dependents,
and retrieve prerequisite tasks from both `depends_on` JSON and
`entity_relationships`
([`internal/repository/task/dependency.go:21-92`](../../../internal/repository/task/dependency.go#L21-L92),
[`dependency.go:104-165`](../../../internal/repository/task/dependency.go#L104-L165)).
This is useful for a feature-root team plan, but it is task/feature-oriented and
does not provide a generic epic/feature child-plan DTO or topological waves.

Sprint readiness already joins assignment data with `agent_type` and
`depends_on`, then computes dependency satisfaction in memory
([`internal/repository/sprint/repository.go:883-896`](../../../internal/repository/sprint/repository.go#L883-L896),
[`repository.go:1159-1225`](../../../internal/repository/sprint/repository.go#L1159-L1225),
[`internal/services/sprint_service.go:2562-2611`](../../../internal/services/sprint_service.go#L2562-L2611)).
This is a reusable parsing/query pattern, not a team executor: it scores sprint
readiness and does not dispatch, claim, or persist per-child execution outcomes.

### 1.7 Existing host-side team workflow

`/shark-rider run-sprint-team` groups sprint tasks by feature and invokes
`/run-agent-team` one feature at a time, with `/run` fallback for standalone
entities ([`skills/shark-rider/verbs/run-sprint-team.md:1-21`](../../../skills/shark-rider/verbs/run-sprint-team.md#L1-L21)).
The bundled workflow requires host capability/version/branch/worktree checks and
explicitly serializes feature-team invocations
([`internal/sharkdata/default_data/skills/sprint-execution/workflows/run-sprint-team.md:27-37`](../../../internal/sharkdata/default_data/skills/sprint-execution/workflows/run-sprint-team.md#L27-L37),
[`run-sprint-team.md:130-167`](../../../internal/sharkdata/default_data/skills/sprint-execution/workflows/run-sprint-team.md#L130-L167)).
This is valuable operational precedent for preconditions and fallback, but it is
host skill logic, not Shark-owned durable orchestration and it has no generic
epic/feature plan/result contract.

## 2. Patterns and conventions to follow

1. Keep CLI commands thin.  Commands parse keys/options, call a service, and
   format JSON/table output; business logic belongs in services.  This is the
   explicit command rule in [`.claude/rules/cli/commands.md`](../../../.claude/rules/cli/commands.md)
   and the service rule in
   [`.claude/rules/services/service-design.md`](../../../.claude/rules/services/service-design.md).
2. Define interfaces at the consumer boundary and inject repositories/services.
   Use `context.Context` first, DTOs for plan/execution inputs, domain results,
   and wrapped errors.  `RunControllerDeps` demonstrates the established
   constructor-injection style ([`controller.go:98-220`](../../../internal/runner/controller.go#L98-L220)).
3. Treat route-based workflow steps and semantic outcomes as authoritative.
   Do not hardcode status lists or assume `pass` maps to a particular status.
4. Reuse Shark’s rendered prompt, including the agent body and placeholder
   aliases.  The worker receives one scoped prompt and returns an outcome; it
   does not claim, heartbeat, release, or transition its dispatched entity.  The
   current worker ownership preamble and `next` prompt assembly encode this
   contract ([`next.go:632-649`](../../../internal/cli/commands/next.go#L632-L649)).
5. Preserve repository boundaries: repositories perform SQL, services own
   orchestration and transaction decisions, and tests use mocked repositories or
   services except repository tests.  See
   [`.claude/rules/testing/architecture.md`](../../../.claude/rules/testing/architecture.md).
6. Use structured JSON for machine consumers and concise human output.  Existing
   `NextResponse`, `RunResult`, and `StageLog` are good shape precedents, but E38
   needs a team-level schema rather than overloading stage logs.

## 3. Integration map

| Concern | Existing integration point | E38 use | Gap |
|---|---|---|---|
| Root/child discovery | Epic/feature repositories and `nextDescribeDispatchableChildren` used by `next.go` | Build a complete child snapshot | Existing cascade exposes one dispatch candidate, not a complete stable plan |
| Workflow routing | `config/workflow.Step`, `ActionService`, per-entity workflow narrowing | Derive worker/gate metadata and route root outcomes | Need team classification and aggregate outcome policy |
| Prompt | placeholder generators, action service, `assembleDispatchPrompt` | Render each child exactly as ordinary dispatch | Must expose/reuse this at a service seam without duplicating rendering |
| External worker | `runner.AgentDispatcher`, Claude/Codex dispatchers | Run bounded child workers | Need scheduler adapter and capability detection |
| Claims | `ClaimService`, `entity_claims`, `shark next` claimability | Claim child immediately before dispatch; release session-scoped | Need team-aware conflict/skipped reporting and parent/root lease policy |
| Dependencies | task `depends_on`, relationship queries, `dependency.Detector` | Topological waves and predecessor gating | Need generic plan graph/topological scheduler; cycle/missing-edge plan errors |
| Root transitions | `EntityTransitioner`, `RunController`, configured outcome routing | Parent-only root transition after aggregate result | Existing controller transitions one entity at a time and has no aggregate result |
| Execution telemetry | `work_sessions`, run slog events, `RunResult`/`StageLog` | Record worker timing/outcome and operator diagnostics | No durable team-run/member ledger or resume cursor |
| Worktree safety | `runner.WorktreeCreator`, `--worktree`, sprint preconditions | Validate worktree/shared-file safety | Need a team-level conflict policy; do not parallelize overlapping worktrees/files |
| Operator surface | `shark next`, `shark run`, host sprint team skill | Add preview/start/resume/team summary | New command/API/schema needed; preserve ordinary paths |

## 4. Extension versus new code

### Extend/reuse

- **Prompt/action resolution: extend** the existing `next` resolver or extract a
  service-level dispatch-step builder.  New code must call the same placeholder,
  workflow narrowing, action population, and agent-body assembly path.  A second
  renderer would create drift and violate E38’s “existing prompt contract” rule.
- **Worker process invocation: extend** `runner.AgentDispatcher`.  Add no team
  provider implementation; use the existing provider map and process result
  contract.  A team scheduler should receive a dispatcher factory/map by
  dependency injection.
- **Claims and heartbeats: extend** `ClaimService`.  Use the existing atomic
  claim and session-scoped release.  Add only team-specific reporting/ownership
  metadata if required; do not weaken the unique claim constraint or use `Force`
  for normal team dispatch.
- **Workflow routing: extend** `ActionService`, `workflow.Service`, and typed
  transition adapters.  Team aggregation must resolve a configured outcome on
  the root and use existing transition validation/history.
- **Child reads/dependency parsing: extend** repository/service interfaces with a
  narrow “list children for root”/dependency snapshot method, preferably backed by
  existing typed repositories and relationship queries.  Reuse JSON parsing and
  cycle validation from `internal/repository/task/dependency.go`.
- **Worktree/capability checks: extend** `runner.WorktreeCreator` and host
  precondition helpers.  The team plan should mark degraded sequential mode when
  safe concurrency is unavailable.
- **Testing/observability: extend** existing runner and claim mocks, table-driven
  tests, stage event conventions, and JSON output assertions.

### New code required

- **Team plan domain model and planner**: new package (likely under
  `internal/team` or `internal/runner/team`) with root, child records, worker
  metadata, dependency edges, waves, concurrency limit, claim state, exclusion
  reason, and capability mode.  Existing `NextResponse` is one step and cannot
  represent the required full plan.
- **Dependency-aware bounded scheduler**: new service that executes independent
  children concurrently, waits for successful configured predecessor outcomes,
  suppresses dependents after failure/block/pause, cancels safely, and falls back
  sequentially.  `RunController` cannot be extended into this by adding a loop:
  its state/result model is explicitly one entity and its transitioner assumes a
  single key.
- **Durable team-run ledger**: new tables or an equivalent durable repository
  record are required by the resume and “exactly once” criteria.  Recommended
  `team_runs` (root, mode, limit, status, timestamps, aggregate outcome) and
  `team_run_items` (child key, wave, planned worker, claim/session ID, started/
  ended, outcome, skipped reason, evidence, attempt).  Existing claims are
  leases, and work sessions are telemetry; neither is a plan membership or
  aggregate execution record.  A filesystem JSON artifact alone would not be
  sufficient as the Shark source of truth.
- **Team aggregate result and root router**: new logic that maps child outcomes
  to semantic root outcomes, preserves partial/failed/paused distinctions, and
  calls the configured root transition.  It must not mark the root successful
  merely because some children passed.
- **Preview/start/resume CLI surface**: new thin commands (exact naming to be
  decided by architecture) for plan preview and execution/reporting.  They must
  guarantee preview is read-only and expose machine-readable per-child data.
- **Shared-resource conflict detector**: new plan-time policy/adapter for file
  ownership and worktree conflicts.  No existing repository field provides a
  complete file-conflict graph; task `file_path` and worktree support can be
  inputs, with unknown overlap treated conservatively.

## 5. Technical risks and feasibility

**Feasibility: confirmed with a medium implementation risk.**  The worker,
workflow, prompt, lease, and transition primitives are already implemented and
tested.  The risk is correctness of the new durable scheduler and its boundaries,
not external provider integration.

| Risk | Impact | Mitigation |
|---|---|---|
| Duplicate dispatch after interruption | High | Persist plan item state before dispatch; claim immediately before worker start; treat completed item ledger state as authoritative on resume; release by session ID only |
| Root and child ownership confusion | High | Root coordinator owns root lease/transitions; worker prompts retain ownership preamble; scheduler owns child claim lifecycle |
| Dependency ambiguity/cycles | High | Validate all edges during preview; fail plan on missing/ambiguous relationship or cycle; never guess ordering |
| Partial result misrouted as success | High | Explicit aggregate state machine; require all eligible children and required gates to satisfy configured success criteria before root pass |
| Live claim conflicts | Medium | Use `IsClaimable`/atomic `Claim`, record conflict and continue unrelated wave members; never force-steal by default |
| Provider/host lacks team capability | Medium | Capability probe, explicit sequential fallback, and mode in plan/result; never claim parallelism that did not happen |
| Concurrent file edits | High | Require non-overlapping declared paths or isolated worktrees; serialize unknown/overlapping work; surface excluded reason |
| SQLite write contention | Medium | Keep writes small/serialized where needed, retry bounded transient busy errors, and avoid long transactions around worker processes |
| Existing `work_sessions` outcome vocabulary/semantics | Medium | Do not overload it for team item state; keep team ledger outcome schema separate and link claim/session IDs |
| Cross-entity workflow differences | Medium | Narrow action/workflow services per entity type as `run.go` does; resolve every child independently |

## 6. Recommended implementation approach

1. Define the team-run/item domain schema and state machine first, including
   preview-only, planned, claimed, running, completed, failed, blocked, paused,
   skipped, cancelled, and not-yet-eligible states.  Specify idempotency keys and
   resume rules before adding concurrency.
2. Extract or expose a reusable single-step dispatch builder from `next.go` so a
   planner can obtain a fully rendered child dispatch step without invoking the
   CLI recursively or duplicating prompt logic.  Preserve `NextResponse` for
   compatibility.
3. Add a read-only planner that snapshots root children, workflow metadata,
   dependencies, claims, and conflict reasons; topologically sorts eligible work
   into waves; and records the plan durably.  Preview must perform no claims,
   status changes, notes, or file mutations.
4. Add a coordinator that claims a child atomically at launch, dispatches via the
   existing `AgentDispatcher`, records the worker semantic outcome/evidence,
   releases the child session-scoped, and advances only eligible dependent waves.
   Use bounded concurrency and a sequential fallback.
5. Add aggregate routing that computes the root semantic outcome and invokes the
   existing typed transition service.  Stop at configured human/quality/pause
   actions and make the exact next decision visible.
6. Add preview, execute, resume, and summary output with JSON as the stable
   contract.  Include root, mode, concurrency, each child’s planned/actual
   worker, dependency state, claim conflict, outcome, skipped reason, evidence,
   and next action.
7. Verify with mocked scheduler/dispatcher/claim services, repository tests for
   the ledger, and integration fixtures for three independent children, a
   dependency wave, a live claim, failure containment, pause gates, interruption
   and resume, sequential fallback, and ordinary `shark next`/`shark run`
   compatibility.

## 7. Exit-gate assessment

- **All existing related code identified**: yes — single-step dispatch, runner,
  workflow/action/prompt system, claims/sessions, dependencies, sprint team
  precedent, worktree support, and tests are cited above.
- **Extension-vs-new analysis for every component**: yes — see Section 4.
- **Feasibility confirmed or risks flagged**: yes — feasible; durable idempotent
  scheduling and shared-file safety are the primary risks.
- **Actionable for architect**: yes — use the existing per-child dispatch
  contract and add only the team planner, scheduler, ledger, aggregate router,
  and thin operator surface.

**Recommended outcome for parent loop**: `pass` — research is complete and ready
for architecture/design, subject to preserving the root/worker ownership and
durable-ledger boundaries above.
