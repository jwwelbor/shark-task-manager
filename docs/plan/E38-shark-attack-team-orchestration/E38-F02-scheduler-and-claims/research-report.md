# E38-F02 Research Report: Scheduler and Claims

**Scan date:** 2026-07-13
**Repository:** `/tmp/shark-task-manager-E38-20260713`
**Feature:** Scheduler and Claims
**Depth:** Deep tactical codebase and dependency analysis

## Executive recommendation

Implement F02 as a new coordinator/scheduler in `internal/team`, extending the
F01 plan and ledger contracts and reusing the existing dispatch, claim, session,
dependency, and worktree seams. Do not extend `internal/runner.RunController`
into a multi-child loop: it owns a single entity and a single status-stage
result. Do not put scheduling in Cobra commands or in the worker prompt.

The scheduler should:

1. Load the confirmed F01 `TeamRun` and immutable `TeamRunItem` snapshot.
2. Select the next dependency-ready wave deterministically.
3. Claim each child immediately before dispatch with `ClaimService.Claim`.
4. Record claim conflicts as item-level diagnostics, not as force-steals.
5. Dispatch the canonical `dispatch.DispatchStep` prompt through the existing
   `runner.AgentDispatcher` contract.
6. Heartbeat the root and active child claims, release each child by its
   session ID, and persist the semantic process result through the F01 ledger.
7. Bound concurrency, serialize when resource ownership is unknown/overlapping,
   and contain one worker failure so unrelated ready items can finish.

The scheduler is not present in the checkout yet. F01 is already represented by
`internal/team` models/planner/ledger code, `internal/repository/teamrun`, and
the additive `team_runs`/`team_run_items` schema migration. F02 should consume
those existing contracts rather than introduce another run or result model.

## 1. Scope and strategic context

The parent epic research establishes that ordinary `shark next` and `shark run`
are single-entity dispatch paths. The E38 PRD excludes new providers, automatic
merges, cross-database execution, backlog decomposition, and changes to normal
single-entity execution. F02 therefore owns only bounded execution of a
confirmed team plan and safe child lease handling.

Sibling feature contracts constrain F02 as follows:

| Feature | Relationship to F02 |
|---|---|
| F01 Team Plan and Durable Ledger | Producer. Supplies `TeamPlan`, `TeamRun`, `TeamRunItem`, plan hash, wave, dependency, attempt, claim-session, and result fields. F02 must persist execution state through this contract. |
| F04 Shark Attack Skill and Role Protocol | Producer. Supplies role/roster and communication context. F02 may attach role identity/hand-off metadata but must not make roster YAML authoritative for workflow routing. |
| F03 Aggregate Routing and Resume | Consumer. Consumes per-child semantic outcomes, process results, evidence, skip reasons, dependency state, and claim/session links. F02 must leave enough durable evidence to reconcile interrupted runs. |
| F05 Reporting and Operator Surface | Consumer. Consumes `TeamRunResult`, per-item lifecycle, mode, concurrency, timing, evidence, and next-action diagnostics. |
| E22 dispatcher contract | Cross-epic producer. Existing `AgentDispatcher`, `DispatchInput`, and `DispatchResult` are the worker execution boundary. |

Authoritative references: [`epic.md`](../epic.md), [`research.md`](../research.md),
[`architecture.md`](../architecture.md), and [`E38-interaction-map.md`](../E38-interaction-map.md).

## 2. Existing implementations and extension points

### 2.1 F01 team plan and durable ledger — extend directly

F01 has already created the domain boundary needed by F02:

- [`internal/team/models.go:210`](../../../../internal/team/models.go#L210)
  defines `TeamPlanItem`, including wave, dependencies, dispatch metadata,
  claim diagnostics, eligibility, and exclusion reason.
- [`internal/team/models.go:225`](../../../../internal/team/models.go#L225)
  defines `TeamPlan`; execution mode and concurrency limit are already part of
  the immutable plan hash input.
- The same file defines `TeamRun`, `TeamRunItem`, `ItemResultUpdate`, item/run
  statuses, bounded evidence, artifact references, attempt numbers, and
  claim/worker session IDs. These are the durable scheduler hand-off fields.
- [`internal/team/interfaces.go:10`](../../../../internal/team/interfaces.go#L10)
  exposes the read-only planner contract. `PlannerDeps` deliberately has no
  mutation-capable claim or dispatch dependency.
- [`internal/team/plan.go:42`](../../../../internal/team/plan.go#L42)
  builds a complete child snapshot, attaches dependencies, validates cycles and
  eligibility, and finalizes a deterministic plan. F02 should treat the
  confirmed persisted snapshot as its execution input, not re-plan silently.
- [`internal/team/ledger_service.go:30`](../../../../internal/team/ledger_service.go#L30)
  exposes `PersistConfirmedPlan`, `UpdateRun`, `RecordItemResult`, and read
  methods. `RecordItemResult` already validates ownership, attempt, terminal
  idempotency, evidence bounds, and compare-and-set behavior.
- [`internal/repository/teamrun/repository.go:1`](../../../../internal/repository/teamrun/repository.go#L1)
  is the SQL-only repository. `CreateRunWithItemsIfAbsent` confirms one plan
  snapshot atomically; `CompareAndSetItem` is the concurrency-safe persistence
  seam F02 needs for item lifecycle/result updates.

**Extension decision:** extend these interfaces with the smallest scheduler
operations that are missing, likely a scheduler-facing `LoadRunnableItems` or
use of `ListItems` plus in-memory readiness evaluation. Avoid adding scheduler
policy to the SQL repository. If F02 needs a claim-session update before a
worker starts, add a ledger method with explicit expected status/attempt rather
than directly updating rows from the scheduler.

### 2.2 Canonical dispatch-step resolution — extend/extract, never duplicate

[`internal/dispatch/step.go:58`](../../../../internal/dispatch/step.go#L58)
defines `DispatchStep`, and [`internal/dispatch/step.go:82`](../../../../internal/dispatch/step.go#L82)
defines `DispatchStepResolver`. `StepResolver.Resolve` reads current status,
generates placeholders, resolves the configured action, classifies terminal and
pause/human-gate steps, and assembles the fully rendered prompt.

The F01 planner already depends on this resolver to store metadata without
persisting prompts. F02 must resolve or re-resolve the child step through this
same seam at dispatch time, then pass `Prompt`, `EntityKey`, `EntityType`,
`Status`, `AgentType`, and provider/model metadata to the dispatcher. It must
not call `shark next` recursively, reconstruct prompts from YAML, or persist
rendered/system prompts in the ledger.

**Extension decision:** reuse as-is if the scheduler can receive a resolver;
otherwise expose a shared factory/adapter from the CLI wiring. Keep the resolver
read-only. A resolver result with a pause, terminal, unresolved workflow, or
unresolved placeholders becomes an item-level skip/pause diagnostic according to
the F03 aggregate contract, not a claim or dispatch.

### 2.3 Claims and leases — extend `ClaimService`, preserve repository atomicity

[`internal/services/claim_service.go:96`](../../../../internal/services/claim_service.go#L96)
implements reclaim-before-claim, actor/session generation, conflict diagnostics,
and optional work-session opening. [`ClaimService.Release`](../../../../internal/services/claim_service.go#L161)
supports safe session-scoped release and requires `force` for unscoped release.
[`ClaimService.Heartbeat`](../../../../internal/services/claim_service.go#L223)
renews the lease and records progress/note.

The SQL layer at [`internal/repository/claim/claim_repository.go:35`](../../../../internal/repository/claim/claim_repository.go#L35)
uses the unique `(entity_type, entity_key)` constraint for the single-grab
guarantee. `ReleaseSession`, `Renew`, and `ReclaimExpired` are the required
safe-release, heartbeat, and crash-recovery operations.

Current semantics that F02 must retain:

- Claim immediately before dispatch, not when previewing or persisting a plan.
- Never use `Force` for normal scheduling; a live claim is a visible conflict.
- Store the returned `SessionID` in the team item and use that exact ID for
  heartbeat and release.
- Release in a `defer`/cleanup path even when dispatch returns an error.
- Treat a failed release as an operator-visible diagnostic; do not delete or
  overwrite a replacement claim.
- Use the configured TTL, including explicit zero meaning no expiry. Do not add
  a scheduler-specific expiry policy.
- Because work-session writes are best-effort telemetry in `ClaimService`, the
  team ledger remains the authoritative item lifecycle record.

**Extension decision:** use `ClaimService` directly through a narrow interface
owned by `internal/team`. Add team-specific diagnostics only in the team item
ledger. Do not add `team_run_id` to `entity_claims`; the team item already links
the claim session, and the generic claim table is shared by ordinary Shark runs.

### 2.4 Existing single-entity runner — reuse dispatcher only

[`internal/runner/controller.go`](../../../../internal/runner/controller.go)
contains `RunController`, `RunResult`, and `StageLog` for one entity/status loop.
It gates status advancement on process success and stops at pause, archive,
failure, or cancellation. Its injected seams are useful precedents, but its
state model is not suitable for F02: one entity key, one current status, and one
linear stage list.

[`internal/runner/dispatcher.go:45`](../../../../internal/runner/dispatcher.go#L45)
defines `AgentDispatcher`; `DispatchInput` carries the rendered instruction,
working directory, entity metadata, agent type, and model, while
`DispatchResult` carries exit code, stdout/stderr, duration, and command.
Claude and Codex implementations are already behind this interface.

**Extension decision:** inject a provider-keyed dispatcher map or factory into
the new scheduler. Reuse `execAndCapture` behavior through the dispatchers.
Do not call `RunController.Run` for each child: it would allow child-level
status transitions and would conflate a worker process result with a team
semantic outcome. The F02 scheduler should record the process result and leave
root/child workflow transitions to the parent/team orchestration contract.

### 2.5 Dependency graph and readiness — reuse parsing, add generic wave logic

[`internal/repository/task/dependency.go:88`](../../../../internal/repository/task/dependency.go#L88)
provides task-feature graph construction and cycle validation. The same file
also reads prerequisite tasks from both the legacy `tasks.depends_on` JSON and
relationship-backed dependencies. `internal/dependency/detector.go` provides
the reusable graph/cycle primitive. Sprint readiness code in
`internal/repository/sprint/repository.go` and `internal/services/sprint_service.go`
already evaluates dependency satisfaction in memory.

F01's `TeamPlanItem.DependencyKeys` and `Wave` are now the more appropriate
execution input. F02 must not query only dispatchable children, because a
terminal or excluded prerequisite still affects dependent eligibility.

**Extension decision:** reuse task dependency adapters and normalization from
F01; add scheduler-local readiness evaluation over the persisted typed item
graph. Do not force the generic scheduler to depend directly on task SQL. For
external prerequisites, honor F01's `External`/`Satisfied` flags and report an
unsatisfied external prerequisite as blocked or skipped per the configured
aggregate policy.

### 2.6 Worktree and resource safety — extend capability seam, conservative default

[`internal/runner/worktree.go:15`](../../../../internal/runner/worktree.go#L15)
defines `WorktreeCreator`, and `WorktreePaths` creates isolated paths/branches
for an entity. Existing `shark run --worktree` setup is in
[`internal/cli/commands/run.go`](../../../../internal/cli/commands/run.go).

There is no general file/resource ownership index for team children. F02 must
therefore treat unknown or overlapping ownership conservatively: choose
sequential mode, exclude the unsafe item, or require an explicit capability
override before parallel dispatch. Do not infer safety merely from distinct
entity keys. Worktree creation is not by itself proof that shared generated
files, databases, or repository metadata are safe to edit concurrently.

**Extension decision:** consume an injected capability/resource policy. Reuse
`WorktreeCreator` for isolated worker directories where supported. The policy
must make fallback and exclusion reasons durable in the item/run result.

### 2.7 Database schema and transaction behavior — extend additive F01 schema

`internal/db/db.go` invokes `migrateTeamRunTables` during schema application and
creates `team_runs` and `team_run_items` additively. The schema has allow-listed
run/item statuses, JSON dependency arrays, unique typed membership per run,
indexes by wave/status, child, claim session, and worker session.

The repository uses short transactions with retry behavior for SQLite busy
errors. F02 should use the repository/ledger APIs and keep worker execution
outside database transactions. Never hold a transaction while an external
worker runs. This repository's SQLite busy behavior makes serialized, bounded
ledger writes preferable to many independent writes from worker goroutines.

**Extension decision:** no new F02 tables are required. Add only scheduler
fields/methods if the existing item lifecycle cannot represent a transition;
prefer the existing `item_status`, `attempt`, `claim_session_id`,
`worker_session_id`, `outcome`, `skip_reason`, timestamps, and evidence fields.

## 3. Proposed F02 dependency and lifecycle map

```text
confirmed TeamRun / TeamRunItems (F01)
        |
        v
read item snapshot + dependency graph + capability facts
        |
        v
select ready wave / bounded slots / safe mode
        |
        +--> claim child atomically -- conflict --> item skipped/claimed diagnostic
        |
        v
resolve canonical dispatch step + prompt
        |
        v
dispatch through AgentDispatcher
        |
        +--> heartbeat root and active child leases
        |
        v
release child by session ID
        |
        v
CAS ledger item result + process evidence
        |
        v
repeat wave or return TeamRunResult to F03/F05
```

Recommended item lifecycle:

`planned -> claimed -> running -> completed|failed|paused|blocked|cancelled`

Dependency-ineligible items should become `skipped` or `blocked` with an
explicit reason. A claim conflict should not be represented as successful
completion and should not be retried by stealing. A worker failure should
release its claim and record its exit/error while allowing independent workers
to complete. Dependent items must remain unstarted until all required
predecessors have an acceptable semantic result.

The coordinator should own the root lease/session. Child worker prompts retain
the existing worker-ownership preamble: workers craft and report evidence; they
do not claim, heartbeat, release, or advance the dispatched entity. The
coordinator performs those operations and F03 later owns aggregate root
routing.

## 4. Integration point inventory

| Surface | Current path | F02 use | Change classification |
|---|---|---|---|
| Confirmed plan | `internal/team/models.go`, `internal/team/ledger_service.go` | Load immutable items, waves, dependencies, plan hash | Extend consumer only |
| Item persistence | `internal/repository/teamrun/repository.go` | CAS claim/running/result lifecycle | Extend narrow methods if needed |
| Dispatch metadata/prompt | `internal/dispatch/step.go` | Resolve canonical step at dispatch | Reuse/extract wiring |
| Claim policy | `internal/services/claim_service.go` | Claim, heartbeat, session release, TTL reclaim | Reuse through interface |
| Claim SQL | `internal/repository/claim/claim_repository.go` | Atomic unique claim and safe session operations | Reuse unchanged |
| Worker process | `internal/runner/dispatcher.go` and provider dispatchers | Execute one child | Reuse interface |
| Dependency data | `internal/repository/task/dependency.go`, `internal/dependency` | Normalize/read prerequisites and detect cycles | Reuse adapters; new generic readiness |
| Resource isolation | `internal/runner/worktree.go` | Optional per-child worktrees/capability checks | Extend policy adapter |
| Telemetry | `internal/runner/logging.go`, `internal/runner/StageLog`, work sessions | Emit root/child/wave/claim/session/duration/outcome events | Extend event fields at coordinator |
| CLI | `internal/cli/commands` | Future F05 surface; F02 should expose service, not own formatting | New thin adapter later |

## 5. Extension versus new-code decisions

### Extend/reuse

- F01 `TeamRun`/`TeamRunItem`/`TeamPlan` and ledger repository/service.
- `ClaimService` and `entity_claims`; preserve atomic unique claims and TTL.
- `dispatch.StepResolver` and prompt assembly.
- `runner.AgentDispatcher`, `DispatchInput`, and `DispatchResult`.
- Existing dependency normalization, cycle detection, and worktree creator.
- Existing structured logging and mock-based testing conventions.

### New code required

- `internal/team/scheduler.go` or equivalent coordinator package implementing
  dependency-aware bounded execution.
- Scheduler consumer interfaces for ledger, claim service, dispatch resolver,
  dispatcher selection, capability/resource policy, and clock/ID generation.
- Readiness/wave execution policy that is deterministic and independent of SQL.
- Root/child heartbeat orchestration and cleanup handling.
- Item-level process-result mapping and durable claim-conflict/failure reasons.
- Scheduler tests covering parallel slots, dependency gating, claim races,
  heartbeat/release failures, cancellation, provider errors, fallback, and
  idempotent ledger updates.

Do not create a second provider abstraction, second claim table, second prompt
renderer, or a second team result schema.

## 6. Risks and feasibility constraints

1. **Worker output is not automatically a semantic workflow outcome.** The
   dispatcher exposes process exit/output, while F03 owns semantic aggregation.
   F02 needs a small, explicit result adapter and must not infer root status from
   exit code alone.
2. **Claim release races are real.** A TTL reclaim/reissue can occur before a
   late cleanup path. Session-scoped release is mandatory; unscoped release can
   delete another worker's lease.
3. **SQLite serializes writes.** Parallel workers can execute concurrently, but
   ledger updates and claim writes must be short, bounded, and retry-safe.
4. **Prompt/workflow drift can occur after F01 confirmation.** Re-resolve the
   child step at dispatch and detect material metadata drift against the plan;
   pause or require re-plan rather than dispatching a mismatched worker silently.
5. **Resource ownership is incomplete.** The safe default is sequential fallback
   when ownership is unknown or overlapping. Parallelism is an optimization,
   not a correctness prerequisite.
6. **Cancellation must preserve resumability.** Cancel active workers through
   context, release only owned claims, record cancelled/paused item state, and
   leave unfinished items eligible for F03 resume.
7. **Worker prompts can contain sensitive data.** Persist bounded evidence and
   artifact references only; never persist rendered prompts or raw full stdout by
   default. F01 already validates evidence bounds/sensitive patterns.

## 7. Recommended implementation sequence

1. Define scheduler interfaces in `internal/team` at the consumer boundary.
2. Implement deterministic ready-item selection from the F01 snapshot and
   dependency statuses; add sequential execution first.
3. Add claim-before-dispatch, heartbeat, session release, and CAS ledger
   transitions with failure cleanup.
4. Add dispatcher selection and canonical step/prompt resolution, including
   plan metadata drift checks.
5. Add bounded parallel wave execution with a worker-group context and explicit
   per-item error isolation.
6. Add capability/resource policy and sequential fallback diagnostics.
7. Integrate F04 role/communication hooks as metadata/handoff events only.
8. Hand the resulting per-child contract to F03/F05; leave root aggregate
   routing and operator formatting to those features.

## 8. Test and verification surface

Existing relevant tests include:

- `internal/team/*_test.go` for plan, dependency adapter, model, and ledger
  invariants.
- `internal/repository/teamrun/*_test.go` and
  `internal/db/team_run_migration_test.go` for persistence and SQLite behavior.
- `internal/services/claim_service_test.go` and
  `internal/repository/claim/claim_repository_test.go` for TTL, conflict,
  heartbeat, release, and session semantics.
- `internal/runner/controller_test.go`, dispatcher tests, and worktree tests
  for injected process and isolation seams.
- `internal/repository/task/dependency*_test.go` and
  `internal/dependency/detector_test.go` for dependency correctness.

F02-specific tests should use mocked claims, ledger, resolver, dispatcher, and
clock/resource policy. Only repository tests should use the real database. At
minimum, assert:

- no child dispatch occurs before a successful claim;
- a claim conflict never triggers `Force` or worker dispatch;
- every successful claim is released with the same session ID;
- late cleanup cannot release a replacement claim;
- dependent children do not run after failed/blocked predecessors;
- independent children respect the concurrency limit;
- one worker failure does not corrupt unrelated item state;
- cancellation leaves resumable durable item states;
- duplicate result delivery is idempotent and conflicting terminal results fail;
- root/child heartbeats stop cleanly and heartbeat errors are reported;
- sequential fallback is explicit and visible.

## Exit-gate assessment

- **Existing related code identified with file paths:** yes.
- **Extension points documented:** yes; F01 ledger, dispatch resolver,
  ClaimService, dispatcher, dependency, worktree, and telemetry seams are named.
- **Inter-feature dependencies mapped:** yes; F01/F04 inputs and F03/F05 outputs
  are explicit.
- **Extension-vs-new analysis completed:** yes.
- **Actionable for architect/specification:** yes. The recommended design is a
  new `internal/team` scheduler around existing contracts with no second engine.

No Shark workflow-state transition commands were run for E38-F02. The parent
loop should advance the feature to the next research/specification stage after
reviewing this artifact.
