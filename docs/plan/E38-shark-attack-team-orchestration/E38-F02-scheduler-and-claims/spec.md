---
feature_key: E38-F02-scheduler-and-claims
epic_key: E38
title: Scheduler and Claims
status: proposed
---

# Scheduler and Claims

This specification is incremental over the E38 epic. See the epic PRD for
business context and shared goals: §2 goals 2–5 and success criteria for
duplicate-work protection, dependency safety, parallel efficiency, failure
containment, resume safety, and operator diagnosability. See epic PRD §3 for
the execution, claim, lease, cancellation, and compatibility boundaries.
The system-level decisions are in the parent architecture document, notably
§4.2 (team-run domain), §4.3 (scheduler lifecycle), §4.4 (aggregate outcome),
§4.5 (council context), §4.6 (operator result), and ADR-001 through ADR-006.

## Requirements

### Functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-F-001 | The scheduler shall accept a confirmed F01 `TeamRun` and its persisted `TeamRunItem` snapshot, and shall never silently re-plan, add, remove, or reorder membership during execution. A plan-hash mismatch shall stop execution with a refresh-required diagnostic. | Epic PRD §3; parent architecture §3.3; I-01 |
| REQ-F-002 | The scheduler shall evaluate dependencies from the persisted item graph, dispatch only eligible items in the lowest ready wave, and leave an item blocked or skipped with a bounded reason when a prerequisite is failed, blocked, paused, cancelled, skipped, or unsatisfied externally. No dependent item may be marked successful because an unrelated predecessor failed. | Epic PRD §2 goals 2–3; parent architecture §4.3; I-01 |
| REQ-F-003 | The scheduler shall select the persisted execution mode and enforce its positive concurrency limit. Parallel mode shall never exceed the configured limit; sequential mode shall have at most one active child. Unknown or overlapping resource ownership shall use the planned safe fallback or record a capability exclusion; it shall not be inferred safe from distinct entity keys alone. | Epic PRD §2 success criterion “Parallel efficiency”; parent architecture ADR-004 |
| REQ-F-004 | Immediately before dispatch, the coordinator shall claim the child through `ClaimService.Claim` with `Force=false`, persist the returned claim session ID, and transition the item through the ledger’s compare-and-set lifecycle. A live claim conflict shall become an item diagnostic and shall not steal, overwrite, or falsely complete the child. | Epic PRD §4 ownership constraints; parent architecture ADR-005 |
| REQ-F-005 | The coordinator shall resolve each child at dispatch time through the canonical `dispatch.DispatchStepResolver`, pass the rendered instruction and resolved entity metadata to an injected `runner.AgentDispatcher`, and never call a CLI command recursively or reconstruct prompts from workflow YAML. Pause, terminal, unresolved-workflow, and unresolved-placeholder results shall be recorded as item-level non-dispatch outcomes. | Epic PRD §3 workflow/prompt boundary; parent architecture ADR-003 |
| REQ-F-006 | The coordinator shall heartbeat the root lease and every active child lease using the exact returned session IDs, honor the configured claim TTL including explicit zero meaning no expiry, and record heartbeat failures as run/item diagnostics. A worker shall not own claim, heartbeat, release, or workflow-transition operations. | Epic PRD §4; parent architecture ADR-005; `internal/services/claim_service.go` |
| REQ-F-007 | Every claimed child shall be released in cleanup using `ClaimService.Release` with its session ID, including dispatch error, cancellation, panic recovery, and result-persistence failure paths. A failed or stale release shall be reported without force-releasing or deleting a replacement claim. | Epic PRD §2 failure containment; parent architecture §4.3 |
| REQ-F-008 | The scheduler shall map each dispatcher result or error to one durable item result containing item status, semantic/process outcome, bounded evidence, artifact references, attempt, timing, worker session, and skip/failure reason. One worker failure, unavailable provider, or cancellation shall not prevent unrelated ready items from completing. | Epic PRD §2 failure containment and diagnosability; I-02 |
| REQ-F-009 | The scheduler shall persist lifecycle and result updates through F01 `Ledger`/CAS methods, outside external-worker transactions, and shall make terminal recording idempotent. Resume shall not redispatch a completed item; retry requires an explicit new attempt and must not overwrite a terminal result from another attempt. | Epic PRD §2 resume safety; parent architecture §3.3; I-02 |
| REQ-F-010 | On completion, pause, failure, cancellation, or partial progress, the scheduler shall return a complete `TeamRunResult` with root identity, run status, actual mode and limit, every planned item, per-item outcome/reason, claim and worker sessions, evidence references, counts, and a next-action diagnostic. Aggregate root routing remains owned by F03. | Epic PRD §2 operator diagnosability; I-02 and I-05 |
| REQ-F-011 | The scheduler shall emit structured execution events containing run ID, root key, child key, wave, item status, provider, duration, claim/session correlation, and outcome, while excluding rendered prompts, credentials, unrestricted stdout/stderr, and transcripts. | Epic PRD §3 observability boundary; parent architecture §6 |

### Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| REQ-NFR-001 | Scheduling decisions shall be deterministic for the same persisted snapshot: sort ready items by wave, execution order, priority, and canonical child key; bound active workers by the run limit and host capability. | `internal/team/scheduler_test.go` deterministic and concurrency tests |
| REQ-NFR-002 | Worker execution shall never hold a database transaction. Ledger writes shall use the existing short-transaction and SQLite-busy retry conventions, and concurrent writes shall remain bounded/serializable enough to avoid avoidable lock storms. | `internal/team/scheduler_test.go`; `internal/repository/teamrun/repository_test.go` |
| REQ-NFR-003 | All new coordinator dependencies shall be constructor-injected interfaces, context-aware, and testable with mocks. Business policy remains in `internal/team`; SQL remains in `internal/repository/teamrun`; CLI adapters remain thin. | `CLAUDE.md`; `.claude/rules/architecture.md`; `.claude/rules/services/service-design.md` |
| REQ-NFR-004 | Claim safety shall rely on the existing unique entity claim and session-scoped release semantics. The scheduler shall not add a `team_run_id` to `entity_claims`, use force-stealing for normal work, or introduce a scheduler-specific TTL. | `internal/services/claim_service.go`; `internal/repository/claim/claim_repository.go` |
| REQ-NFR-005 | Item evidence and artifact references shall use F01’s bounds and validation. Scheduler logs and persisted outputs shall not contain secrets or rendered/system prompts. | `internal/team/models.go`; `tests/contracts/e38_interactions_test.go#TestTC001_I01TeamRunResultContract` |
| REQ-NFR-006 | Resource safety shall be conservative: use `runner.WorktreeCreator` only as an injected isolation capability, and serialize or exclude unknown/overlapping ownership with an explicit reason. | `internal/runner/worktree.go`; parent architecture ADR-004 |

### Acceptance criteria

| ID | Given / when | Then | Test pointer |
|---|---|---|---|
| AC-F02-001 | A confirmed run has three independent items, limit two, and safe parallel capabilities. | Exactly two workers can be active; the third starts only after a slot is released; all item transitions and results are durable. | `internal/team/scheduler_test.go#TestScheduler_BoundsParallelism` |
| AC-F02-002 | A run contains A and B in wave 0 and C depending on both. | A and B may dispatch according to the limit; C is not claimed or dispatched until both have configured acceptable success outcomes. | `internal/team/scheduler_test.go#TestScheduler_GatesDependentWave` |
| AC-F02-003 | A predecessor fails, pauses, is cancelled, or has an unsatisfied external prerequisite. | The dependent item remains blocked/skipped with the exact dependency reason and is never falsely completed; unrelated ready items continue. | `internal/team/scheduler_test.go#TestScheduler_ContainsDependencyFailure` |
| AC-F02-004 | Two coordinators race to schedule the same child. | At most one claim succeeds; the loser records a claim-conflict diagnostic without force-stealing or dispatching the child. | `internal/team/scheduler_test.go#TestScheduler_ClaimConflictIsNonDestructive` |
| AC-F02-005 | A dispatcher returns success, non-zero exit, provider-not-found, or context cancellation. | The child is released by its exact claim session, the process/semantic result and bounded evidence are recorded, and independent items continue; cancellation leaves explicit cancelled/paused state. | `internal/team/scheduler_test.go#TestScheduler_MapsDispatcherOutcomes` |
| AC-F02-006 | A child dispatch fails before result persistence, or result persistence fails after dispatch. | Cleanup attempts session-scoped release; the failure is visible in the item/run diagnostic; no force release can affect a replacement claim. | `internal/team/scheduler_test.go#TestScheduler_ReleasesOnAllExitPaths` |
| AC-F02-007 | A child runs longer than one heartbeat interval, with TTL enabled or explicitly zero. | Root and active child heartbeats use the correct session IDs; zero TTL does not trigger reclaim; a failed heartbeat is observable and cannot be mistaken for successful completion. | `internal/team/scheduler_test.go#TestScheduler_HeartbeatsRootAndChildren` |
| AC-F02-008 | The resolved step is pause, terminal, unresolved workflow, or contains unresolved placeholders. | No claim/dispatch occurs; the item receives the applicable bounded skip/pause diagnostic and the run result identifies the next action. | `internal/team/scheduler_test.go#TestScheduler_SkipsNonDispatchableSteps` |
| AC-F02-009 | A worker completes and the coordinator is interrupted, then resumes the same run. | Completed items are not dispatched again; expired claimed/running items are reconciled before an explicit retry; unfinished eligible items run once and preserve prior evidence. | `internal/team/scheduler_test.go#TestScheduler_ResumeIsIdempotent` |
| AC-F02-010 | A worker returns unrestricted output containing a prompt or credential pattern. | Persisted evidence is rejected or reduced to a bounded safe summary; logs and `TeamRunResult` contain no prompt, secret, or full transcript. | `internal/team/scheduler_test.go#TestScheduler_RejectsSensitiveEvidence`; `tests/contracts/e38_interactions_test.go#TestTC001_I01TeamRunResultContract` |
| AC-F02-011 | The host reports unknown or overlapping resource ownership. | The scheduler uses the confirmed sequential fallback or excludes the unsafe item with a durable reason; it never runs unsafe parallel work. | `internal/team/scheduler_test.go#TestScheduler_UsesResourceSafetyFallback` |
| AC-F02-012 | All items finish with mixed success, failure, skip, and pause outcomes. | The returned result contains every planned item, mode, limit, sessions, timing, evidence/reasons, counts, and next action; F02 does not directly advance the root workflow. | `internal/team/scheduler_test.go#TestScheduler_ReturnsCompleteTeamRunResult`; `tests/contracts/e38_interactions_test.go#TestTC001_I01TeamRunResultContract` |

### Out of scope for this feature

- Building or confirming plans, discovering children, normalizing dependencies,
  or adding ledger tables; those are F01 responsibilities.
- Aggregate semantic routing, root status transitions, interruption
  reconciliation policy, and the public resume/summary command; those are F03
  and F05 responsibilities.
- The `shark-attack` skill, roster YAML, council inbox, memory, and escalation
  protocol; those are F04 responsibilities.
- New providers, a native agent-team runtime, cross-database execution,
  automatic decomposition, merging, conflict resolution, dashboards, or cost
  accounting.
- Worker-owned claims, heartbeats, releases, notes, workflow transitions, or
  root ownership.

## Architecture

### Component changes

| Path | Change | Responsibility |
|---|---|---|
| `internal/team/scheduler.go` | Create | Coordinator implementing dependency-ready wave selection, bounded slots, claim/dispatch/heartbeat/release lifecycle, cancellation, cleanup, result mapping, and durable run/item updates. |
| `internal/team/interfaces.go` | Modify | Add narrow scheduler-facing interfaces for claims, ledger lifecycle/CAS, step resolution, dispatcher selection, resource capability, clock/ID generation, and optional structured event sink; keep existing planner interfaces stable. |
| `internal/team/models.go` | Modify only if required | Reuse F01 run/item/result types; add only bounded process-result or diagnostic fields that cannot be represented by existing `outcome`, `skip_reason`, `evidence`, timestamps, and session links. |
| `internal/team/scheduler_test.go` | Create | Mock-only scheduler tests for waves, concurrency, claim races, leases, failures, cancellation, resume, resource fallback, and idempotency. |
| `internal/team/ledger_service.go` | Modify | Add explicit claim/running/result CAS helpers only where scheduler ownership/attempt validation cannot use existing `RecordItemResult`, `UpdateRun`, and `ListItems`; preserve validation and idempotency in the service. |
| `internal/repository/teamrun/repository.go` | Modify only if required | Add SQL methods for scheduler CAS transitions or active-item lookup; use existing `CompareAndSetItem`, `ListItems`, and short transactions before adding new queries. |
| `internal/services/claim_service.go` | Reuse; extend diagnostics only if required | Scheduler calls `Claim`, `Heartbeat`, and session-scoped `Release`; no new scheduler TTL or force-steal path. |
| `internal/repository/claim/claim_repository.go` | Reuse unchanged | Existing unique claim, renewal, reclaim, and session-release atomicity remains authoritative. |
| `internal/dispatch/step.go` | Reuse unchanged | `DispatchStepResolver.Resolve(ctx, entityType, key)` is the canonical prompt/metadata seam. |
| `internal/runner/dispatcher.go` | Reuse unchanged | `AgentDispatcher.Dispatch(ctx, DispatchInput)` executes one child process and returns `DispatchResult`/typed errors. |
| `internal/runner/worktree.go` | Reuse through adapter | Optional isolated working directory capability; the scheduler does not assume worktrees prove resource non-overlap. |
| `internal/cli/services_global.go` | Modify | Wire the scheduler’s production dependencies using the existing global accessor pattern; do not put scheduling policy or formatting in Cobra. |
| `internal/cli/commands/attack.go` | Not created by F02 | Public attack start/resume/summary formatting belongs to F05; a future adapter must call the same scheduler service. |
| `tests/contracts/e38_interactions_test.go` | Modify only when the shared result shape changes | Keep one shared contract test for the I-01/I-02/I-05 result shape; do not create twin producer/consumer contract tests. |

### Data model changes

No new tables or columns are required. F02 consumes the additive F01
`team_runs` and `team_run_items` schema in `internal/db/db.go`. The scheduler
uses these existing fields:

- `team_runs`: `status`, `execution_mode`, `concurrency_limit`, `plan_hash`,
  `root_session_id`, lifecycle timestamps, aggregate placeholder, and
  `next_action`.
- `team_run_items`: `wave`, `execution_order`, `dependency_keys`, planned
  worker metadata, `item_status`, `attempt`, `claim_session_id`,
  `worker_session_id`, `outcome`, `skip_reason`, bounded `evidence`, artifact
  references, and lifecycle timestamps.

The scheduler must not add team ownership columns to `entity_claims` or
`work_sessions`. The claim session ID is the safe release key; the team ledger
is the authoritative execution record and work-session telemetry remains
best-effort as defined by `ClaimService`.

### API and interface contracts

The scheduler service is an internal Go service, not a new provider or CLI
protocol. Its public service boundary shall be context-first and return the
existing domain result:

- `Start(ctx context.Context, runID int64, rootSessionID string) (*TeamRunResult, error)`
- `Resume(ctx context.Context, runID int64, rootSessionID string) (*TeamRunResult, error)`
- `Run(ctx context.Context, runID int64, rootSessionID string, mode RunMode) (*TeamRunResult, error)`

The implementation may choose one start/resume method if the caller supplies
an explicit run mode, but it must preserve these semantics:

1. Load and validate the confirmed run and complete item snapshot.
2. Refuse plan drift and invalid run/item transitions before child mutation.
3. Claim immediately before dispatch, never at preview/plan time.
4. Resolve a fresh canonical step, dispatch through `AgentDispatcher`, and
   translate process output into bounded ledger evidence.
5. Release by exact session ID in cleanup, then record the terminal item result
   through the ledger CAS seam.
6. Return a complete `TeamRunResult`; do not transition the root.

The injected seams are:

| Interface | Required operations | Existing source |
|---|---|---|
| `TeamLedger` | load run/items, claim/running CAS, record item result, update run | `internal/team/ledger_service.go` |
| `TeamClaims` | `Claim`, `Heartbeat`, session-scoped `Release` | `internal/services/claim_service.go` |
| `DispatchStepResolver` | `Resolve(ctx, entityType, key)` | `internal/dispatch/step.go` |
| `AgentDispatcher` | `Dispatch(ctx, runner.DispatchInput)` | `internal/runner/dispatcher.go` |
| `ResourcePolicy` | determine safe mode/slot and per-item working directory | `internal/runner/worktree.go` plus F01 capability facts |
| `Clock/ID` | injectable time and session correlation where needed | existing Go time/session generation patterns |

The coordinator owns root heartbeats because the parent loop owns the root
lease. Child worker prompts retain the existing ownership preamble from
`internal/cli/commands/next.go`; no prompt grants the worker Shark mutation
authority.

### Key technical decisions

1. **Put policy in `internal/team`, not Cobra.** This follows parent ADR-001
   and the repository’s thin-command/fat-service rule. It keeps scheduler
   behavior mockable and leaves F05 presentation separate.
2. **Use the F01 ledger as the execution source of truth.** Claims alone cannot
   prove membership, dependency blocking, attempt identity, or resume state.
   Existing F01 CAS and idempotency are extended only at their narrow seams.
3. **Reuse canonical dispatch and generic claims.** This follows parent
   ADR-003 and ADR-005: one prompt renderer, one provider boundary, one claim
   table, configured TTL, and session-safe release.
4. **Schedule waves in memory, persist transitions briefly.** Dependency
   readiness and slot selection are policy; SQL stores durable state. No worker
   process runs inside a transaction, which follows the SQLite busy/retry
   conventions in `internal/repository/teamrun/repository.go`.
5. **Treat process result and semantic outcome as separate data.** A zero exit
   code is evidence of process completion, not permission to infer a root
   workflow transition. F03 owns aggregate interpretation and root routing.
6. **Fail closed on resource uncertainty.** Worktree isolation is a capability,
   not proof of non-overlapping generated files, databases, or metadata. The
   configured sequential fallback or explicit exclusion is safer and remains
   diagnosable.

### Integration with existing code

- `internal/team/plan.go` supplies the immutable `TeamPlan` and its waves;
  F02 consumes the persisted snapshot rather than invoking `Plan` again.
- `internal/team/ledger_service.go` supplies `GetRun`, `ListItems`,
  `UpdateRun`, and `RecordItemResult`; scheduler-specific CAS methods, if
  needed, must validate expected status, attempt, and claim session there.
- `internal/repository/teamrun/repository.go` remains SQL-only and is the
  only place for new team-run queries. Use `CompareAndSetItem` for stale-writer
  protection and preserve the existing SQLite retry behavior.
- `internal/services/claim_service.go` is called with child entity type/key;
  `ClaimInput.Force` is always false for normal scheduling. The returned
  `models.EntityClaim.SessionID` is persisted and reused for heartbeat and
  `Release(..., sessionID, outcome, false)`.
- `internal/dispatch/step.go` resolves current workflow status and renders the
  child prompt at dispatch time. F02 passes `DispatchInput.Instruction`,
  `EntityKey`, `EntityType`, `Status`, `AgentType`, and `Model` to the selected
  dispatcher without persisting the instruction.
- `internal/runner/dispatcher.go` provides `DispatchResult.ExitCode`,
  `Duration`, and command/error classification. stdout/stderr are summarized
  and sanitized before ledger persistence or structured logging.
- `internal/cli/services_global.go` wires the production implementation;
  future `attack` commands must remain argument parsing, service invocation,
  and output formatting only.

## Cross-feature interactions

| Direction | ID | Contract |
|---|---|---|
| Consumes | I-01 | Producer: E38-F01 Team Plan and Durable Ledger. Shape source: **E38 architecture §4.2 Team-run domain contract**. F02 consumes the confirmed `TeamPlan`/`TeamRun`/`TeamRunItem` snapshot, plan hash, waves, item status, and claim/session fields. Contract test: `tests/contracts/e38_interactions_test.go#TC-001` (implemented as `TestTC001_I01TeamRunResultContract`). |
| Produces | I-02 | Consumers: E38-F03 Aggregate Routing and Resume and E38-F05 Reporting and Operator Surface. Shape source: **E38 architecture §4.4 Aggregate outcome contract**. F02 supplies per-child semantic/process outcome, evidence, skip reason, dependency state, timestamps, and session links. Contract test: `tests/contracts/e38_interactions_test.go#TC-001` (shared result serialization; aggregate-specific assertions belong to F03). |
| Consumes | I-04 | Producer: E38-F04 Shark Attack Skill and Role Protocol. Shape source: **E38 architecture §4.5 Council communication contract**. F02 consumes role/roster and communication context only as execution metadata; it does not make roster YAML authoritative for workflow routing or mutation authority. Contract test pointer: `tests/contracts/e38_interactions_test.go#TC-004` (to be added with the shared I-04 contract before F04 integration). |
| Produces | I-05 | Consumer: E38-F05 Reporting and Operator Surface. Shape source: **E38 architecture §4.6 Operator contract**. F02 supplies the machine-readable `TeamRunResult`, per-child status, actual mode, concurrency, counts, evidence, and next-action diagnostics; F05 owns CLI formatting. Contract test: `tests/contracts/e38_interactions_test.go#TC-001` for the shared JSON shape, with F05 presentation tests covering operator fields. |

The I-## identifiers and shape-source wording above mirror
`docs/plan/E38-shark-attack-team-orchestration/E38-interaction-map.md`.

## Cross-epic integrations

| Direction | ID | Contract |
|---|---|---|
| Consumes | X-01 | Producer: E22 External Orchestration Runner; consumer: E38-F02. Shape source: **E38 architecture §4.3 and existing E22 runner dispatcher contract**. F02 consumes the single-worker dispatcher, process result, worktree capability, claim, and fully rendered child-prompt seams. Coverage: `docs/plan/E38-shark-attack-team-orchestration/uat-plan.md` UAT-03, UAT-07, UAT-10. |

The X-01 ownership mirrors both `E38-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md`; it is a reuse contract, not a
new provider or alternate dispatch path.
