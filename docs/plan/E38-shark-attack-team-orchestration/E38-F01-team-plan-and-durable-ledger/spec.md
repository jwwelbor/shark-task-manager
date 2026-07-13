---
feature_key: E38-F01-team-plan-and-durable-ledger
epic_key: E38
title: Team Plan and Durable Ledger
status: proposed
---

# Team Plan and Durable Ledger

This specification is incremental over the E38 epic. See the epic PRD for
business context, goals, constraints, and shared success criteria:

- Epic PRD §2, goals 1–5 and success criteria: team-plan coverage, duplicate
  work protection, dependency safety, failure containment, resume safety, and
  operator diagnosability.
- Epic PRD §3, in-scope items for preview, durable execution summaries, claim
  coordination, and resumable execution.
- Epic PRD §4, constraints on workflow authority, ownership, concurrency, and
  persistence.
- Epic UAT Plan UAT-01, UAT-02, UAT-03, UAT-04, UAT-06, and UAT-07.

## Requirements

### Functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-F-001 | The planner shall accept an epic or feature root key and produce a read-only snapshot containing every direct child exactly once, including terminal, non-terminal, gated, claimed, capability-excluded, and dependency-ineligible children. Each child shall include its canonical key, entity type, current status, execution order, priority when available, and an explicit exclusion reason when it is not eligible. | Epic PRD §2 success criterion “Team-plan coverage”; UAT-01 |
| REQ-F-002 | The planner shall build a deterministic dependency graph for the root’s direct children, reject cycles and unresolved entity references, preserve dependencies that resolve outside the root as external prerequisites, and assign non-negative topological waves. A child with an unresolved, non-success external prerequisite shall be represented but excluded with a dependency reason. | Epic PRD §2 goals 2 and 3; UAT-03 and UAT-04 |
| REQ-F-003 | The planner shall resolve each child’s workflow dispatch metadata through the canonical Shark routing/prompt seam and snapshot the action, responsibility/agent type, provider, model, effort, pause or human-gate classification, and unresolved-placeholder diagnostics. It shall never persist the rendered prompt or full worker instructions in the plan or ledger. | Epic PRD §2 goals 2 and 4; UAT-01 and UAT-03 |
| REQ-F-004 | The planner shall select and report the actual execution mode (`parallel` or `sequential`) and positive concurrency limit from the requested limit plus host capability facts. It shall report the reason for sequential fallback or exclusion when safe team capability, worktree isolation, or resource ownership is unavailable. | Epic PRD §3 in-scope “bounded concurrency and safe fallback”; UAT-03 and UAT-07 |
| REQ-F-005 | The planner shall compute a stable SHA-256 `plan_hash` from canonical root, child, dependency, workflow-metadata, mode, limit, and capability-exclusion data. Equivalent unchanged inputs shall produce the same hash regardless of database row order; any material input change shall produce a different hash. | Epic PRD §2 success criteria “Team-plan coverage” and “Resume safety”; UAT-01 and UAT-06 |
| REQ-F-006 | The ledger service shall persist a confirmed plan and all of its item rows in one short transaction before any child claim or worker dispatch. A failed transaction shall leave neither the run nor partial item membership. Preview shall not write either ledger table. | Epic PRD §4 constraints; UAT-01 and UAT-06 |
| REQ-F-007 | The ledger shall enforce one membership row per `(team_run_id, child_type, child_key)`, retain item lifecycle state, attempt number, dependency snapshot, planned and actual session links, outcome, bounded evidence, skip reason, and timestamps, and support idempotent terminal-result recording. A completed item shall not be implicitly dispatched again by resume; an explicit retry shall increment `attempt`. | Epic PRD §2 success criteria “Duplicate work protection” and “Resume safety”; UAT-02 and UAT-06 |
| REQ-F-008 | The ledger service shall expose the shared `TeamPlan`, `TeamRun`, `TeamRunItem`, and `TeamRunResult` shapes required by E38-F02, E38-F03, and E38-F05, including root identity, run status, execution mode, limit, plan hash, complete item results, claim/session links, aggregate fields, and next action. F01 shall provide persistence primitives only; scheduling, aggregate routing, and presentation remain in the consuming features. | E38 interaction map I-01; E38 architecture §4.2 |

The planner is read-only with respect to Shark entity state. It may inspect
claims and workflow data, but it shall not claim, heartbeat, release, advance,
set status, create notes, dispatch workers, or mutate entity files.

### Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| REQ-NFR-001 | Planning shall perform work bounded by the root’s direct-child count and shall not hold a database write transaction while reading workflow metadata, calculating waves, or resolving host capabilities. | Planner benchmark and SQL/transaction instrumentation in `internal/team/plan_test.go`. |
| REQ-NFR-002 | Ledger writes shall use the repository’s existing SQLite/Turso connection conventions, short transactions, database constraints, and bounded retry handling for transient busy/locked errors. A retry shall not duplicate a run or item row. | Migration and repository tests in `internal/repository/teamrun/repository_test.go`; busy-lock fixture. |
| REQ-NFR-003 | The plan hash serialization shall be canonical: sort children by wave, execution order, priority, and canonical key; sort dependency keys and metadata lists; encode absent values distinctly from empty values; and hash UTF-8 JSON with no volatile timestamps, prompts, claims, or worker output. | Hash golden tests in `internal/team/plan_test.go`. |
| REQ-NFR-004 | Plan and ledger output shall exclude credentials, provider tokens, rendered prompts, unrestricted stdout/stderr, and full transcripts. The item-result input shall accept only a bounded evidence summary and validated project-relative artifact references; it shall reject prompt text and known credential/token patterns rather than persisting raw worker output. | Security tests and review of `internal/team` serialization paths. |
| REQ-NFR-005 | Root type, child type, entity keys, execution mode, lifecycle statuses, concurrency, attempt, JSON dependency lists, and bounded text fields shall be validated before persistence. Roster membership shall not grant workflow or claim authority. | Model validation tests in `internal/team/models_test.go` and repository constraint tests. |
| REQ-NFR-006 | All new service methods shall accept `context.Context` first, use constructor injection, return domain types, wrap errors with operation and entity context, and keep business rules out of repositories and CLI commands. | Code review against `CLAUDE.md`, `.claude/rules/architecture.md`, and `.claude/rules/services/service-design.md`. |
| REQ-NFR-007 | The additive schema migration shall be idempotent on fresh, current, partially migrated, and upgraded databases and shall not delete or rewrite existing entity, claim, or session data. | Migration tests in `internal/db/team_run_migration_test.go`. |

### Acceptance criteria

| ID | Given / when | Then | Test pointer |
|---|---|---|---|
| AC-F01-001 | A feature has four direct children: two independent children, one child depending on both, and one configured approval boundary. An operator requests a plan twice without changing Shark state. | Both results contain each child once; the dependency child is in a later wave; the approval boundary is visible; all resolved role/provider/model metadata and exclusions are present; both hashes match; no claim, status, note, file, or ledger mutation exists. | `internal/team/plan_test.go#TestPlanner_ReadOnlyCompleteSnapshot`; `tests/contracts/e38_interactions_test.go#TC-001` |
| AC-F01-002 | A plan contains a dependency cycle, a missing entity, malformed legacy task dependency JSON, or an unresolvable workflow step. | Planning fails with a typed validation error that identifies the offending root/child and cause; it does not return a dispatchable partial plan or write ledger rows. | `internal/team/plan_test.go#TestPlanner_RejectsInvalidGraph` |
| AC-F01-003 | A task has legacy `tasks.depends_on` data and/or a matching `entity_relationships` `depends_on` row. | The planner-facing dependency adapter de-duplicates the normalized edge by canonical child key; new relationship rows are canonical, legacy JSON is compatibility input, and malformed legacy JSON fails loudly instead of being ignored. | `internal/team/dependency_adapter_test.go#TestDependencySources` |
| AC-F01-004 | Three children are independent, the requested concurrency is two, and host capabilities permit bounded team execution. | The plan reports `parallel`, limit `2`, deterministic waves, and no more than the supplied capability/resource limit; no worker starts during planning. | `internal/team/plan_test.go#TestPlanner_SelectsBoundedParallelMode` |
| AC-F01-005 | The host cannot guarantee safe team execution or worktree/resource isolation. | If the host can execute one ordinary Shark worker and the plan can serialize all eligible work, the plan selects `sequential` and records the exact degraded-mode reason (`parallel_unavailable`, `unknown_resource_ownership`, or `overlapping_resource_ownership`). If no safe single-worker execution capability exists, or a required workflow step cannot be executed by any configured adapter, planning returns a typed capability error before mutation. It never reports parallel mode for an unsafe host. | `internal/team/plan_test.go#TestPlanner_UsesExplicitSequentialFallback` |
| AC-F01-006 | A confirmed plan is persisted while the item insert fails midway. | The transaction rolls back the run and all items. A successful retry creates one complete run and one row per planned child, with a unique membership constraint enforced by SQLite. | `internal/repository/teamrun/repository_test.go#TestRepository_CreateRunWithItems_IsAtomic` |
| AC-F01-007 | The same confirmed plan is persisted twice, then one terminal item result is recorded twice. | A second persistence of the same root and plan hash returns the existing run with `idempotent=true` and creates no rows. A persistence with the same root but a different hash returns typed `ErrPlanDrift` and requires a fresh explicit run. Repeating an identical terminal item result returns the existing result with `idempotent=true`; a different terminal outcome for the same attempt returns typed `ErrConflictingTerminalResult`; attempt remains unchanged. | `internal/team/ledger_service_test.go#TestLedger_Idempotency` |
| AC-F01-008 | A plan changes after persistence by child status, dependency edge, workflow metadata, execution mode, or concurrency limit. | Replanning produces a different hash; resume-facing lookup reports refresh required rather than silently merging the changed snapshot. | `internal/team/plan_test.go#TestPlanHash_DetectsMaterialDrift` |
| AC-F01-009 | A ledger item contains evidence larger than the configured bound, rendered prompt text, or a known credential/token pattern. | The item-result validator rejects the input before persistence; accepted evidence is bounded and contains only validated project-relative artifact references; prompt and secret fields are absent from database rows and JSON. | `internal/team/models_test.go#TestLedgerOutput_RejectsSensitiveContent` |
| AC-F01-010 | Ordinary `shark next` and `shark run` execute against the same fixture before and after the shared dispatch-step extraction. | Their existing response/prompt behavior and status ownership remain unchanged; the team planner can consume the same resolved metadata without invoking Cobra or recursively invoking a CLI command. | `internal/cli/commands/next_test.go#TestNext_RegressionAfterDispatchStepExtraction`; `internal/cli/commands/run_test.go#TestRun_RegressionAfterDispatchStepExtraction` |

### Out of scope for this feature

- Launching workers, scheduling waves, claiming or heartbeating children,
  releasing leases, detecting runtime worker failures, or enforcing resource
  overlap during execution. E38-F02 owns those behaviors.
- Aggregate semantic outcome calculation, root transition, interruption
  reconciliation, and the user-facing resume operation. E38-F03 owns those
  behaviors.
- Human-readable or JSON CLI commands, summary formatting, and telemetry
  presentation. E38-F05 owns those surfaces.
- The `shark-attack` skill, roster YAML, council inbox, memory, escalation
  protocol, and role-aware sprint pull. E38-F04 owns those artifacts.
- Backlog creation, decomposition, dependency rewriting, automatic merge or
  conflict resolution, provider runtime creation, cross-project execution,
  dashboards, cost accounting, or changing ordinary single-worker behavior.

## Architecture

### Component changes

F01 adds the domain and persistence seams needed by the later scheduler,
aggregate, and reporting features. It does not add an attack CLI command; the
operator surface belongs to F05.

| Path | Change | Responsibility |
|---|---|---|
| `internal/team/models.go` | Create | Define `TeamPlan`, `TeamPlanItem`, `TeamRun`, `TeamRunItem`, `TeamRunResult`, `CapabilityFacts`, `DependencyEdge`, `DispatchMetadata`, lifecycle enums, validation, and bounded evidence rules. |
| `internal/team/interfaces.go` | Create | Define consumer-side interfaces for child snapshots, dependency reads, dispatch-step resolution, claim diagnostics, capability facts, and transaction-backed ledger persistence. |
| `internal/team/plan.go` | Create | Implement read-only root validation, complete child snapshotting, dependency normalization, cycle/missing-reference validation, deterministic wave assignment, capability exclusion, canonical serialization, and plan hashing. |
| `internal/team/ledger_service.go` | Create | Own confirmed-plan persistence, item membership validation, idempotent item/run updates, hash-drift checks, and transaction boundaries. It shall not schedule or transition entities. |
| `internal/team/plan_test.go`, `internal/team/dependency_adapter_test.go`, `internal/team/ledger_service_test.go`, `internal/team/models_test.go` | Create | Mock-based planner/service/model tests. These tests shall not use the real database. |
| `internal/repository/teamrun/repository.go` | Create | Implement pure SQL CRUD and transaction-bound inserts/updates for `team_runs` and `team_run_items`, following `internal/repository/claim/claim_repository.go` and `internal/repository/worksession/repository.go`. |
| `internal/repository/teamrun/repository_test.go` | Create | Use the real repository test database, clean up rows per test, and verify constraints, ordering, idempotent reads, and transaction behavior. |
| `internal/db/db.go` | Modify | Add schema version 28 and an idempotent `migrateTeamRunTables` call from `runMigrations`; create tables, indexes, checks, foreign key, and timestamps without altering existing tables. |
| `internal/db/team_run_migration_test.go` | Create | Verify fresh creation, upgrade, rerun idempotency, rollback-safe DDL behavior where supported by the migration pattern, foreign key cascade, checks, and indexes. |
| `internal/services/cascade_service.go` | Modify | Extract a read-only direct-child listing seam, `ListChildren(ctx context.Context, entityType, key string) ([]CascadeChildSnapshot, error)`, from the existing cascade queries. Preserve `DescribeDispatchableChildren` filtering and ordering behavior; F01 uses the broader snapshot seam rather than the first-dispatchable-child behavior. |
| `internal/dispatch/step.go` | Create | Define the reusable service-level `DispatchStepResolver` and transient `DispatchStep` contract used by ordinary `next`/`run` callers and F01. The contract exposes action, status, role/agent, provider, model, effort, gate classification, prompt diagnostics, and transient rendered prompt without coupling callers to Cobra output. |
| `internal/cli/commands/next.go`, `internal/cli/commands/run.go` | Modify | Delegate single-entity metadata and prompt resolution to the shared dispatch-step seam while preserving the existing `NextResponse`, `RunController`, prompt assembly, and worker-ownership behavior. No team-specific mutation is added. |
| `internal/cli/services_global.go` | Modify | Add constructor/accessor wiring for the team planner/ledger service using the existing global service accessor pattern; keep command formatting and DB access out of `internal/team`. |
| `tests/contracts/e38_interactions_test.go` | Create | Prove the I-01 shared shape once for producer and consumers, including plan hash, waves, item status, planned metadata, claim/session links, and complete item result serialization. |

The optional HTTP adapter is not part of F01. If a later API surface is added,
it must call the same service through `cmd/server/services.go` and must not
create a second planner or ledger implementation.

### Data model changes

Add two normalized tables through `internal/db/db.go`. Do not add team columns
to `epics`, `features`, `tasks`, `entity_claims`, or `work_sessions`.

#### `team_runs`

| Column | SQLite type and constraint | Meaning |
|---|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` | Stable run identifier. |
| `root_key` | `TEXT NOT NULL` | Canonical epic or feature key. |
| `root_type` | `TEXT NOT NULL CHECK (root_type IN ('epic','feature'))` | Typed root routing. |
| `status` | `TEXT NOT NULL CHECK (status IN ('planned','running','paused','failed','completed','cancelled'))` | Run lifecycle owned by the coordinator/aggregate features. |
| `execution_mode` | `TEXT NOT NULL CHECK (execution_mode IN ('parallel','sequential'))` | Actual selected mode, including explicit fallback. |
| `concurrency_limit` | `INTEGER NOT NULL CHECK (concurrency_limit > 0)` | Maximum permitted active children. |
| `plan_hash` | `TEXT NOT NULL` | Lowercase 64-character SHA-256 digest. |
| `aggregate_outcome` | `TEXT` | Semantic result supplied by F03; nullable before aggregation. |
| `next_action` | `TEXT` | Bounded operator guidance; nullable before aggregation. |
| `root_session_id` | `TEXT` | Root claim/session correlation; never a worker identity. |
| `started_at`, `completed_at` | `TIMESTAMP` | Lifecycle timestamps; nullable as appropriate. |
| `created_at`, `updated_at` | `TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP` | Audit timestamps. |

Add indexes for `(root_type, root_key, status)`, `status`, and `plan_hash`.

#### `team_run_items`

| Column | SQLite type and constraint | Meaning |
|---|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` | Stable item identifier. |
| `team_run_id` | `INTEGER NOT NULL REFERENCES team_runs(id) ON DELETE CASCADE` | Owning run. |
| `child_key` | `TEXT NOT NULL` | Canonical child key. |
| `child_type` | `TEXT NOT NULL` | Typed child identity. |
| `wave` | `INTEGER NOT NULL CHECK (wave >= 0)` | Topological dependency wave. |
| `execution_order` | `INTEGER NOT NULL` | Deterministic sibling order captured at planning time. |
| `dependency_keys` | `TEXT NOT NULL` | Canonical sorted JSON array of prerequisite keys. |
| `planned_role` | `TEXT` | Workflow-resolved responsibility/agent role. |
| `planned_action` | `TEXT` | Workflow action classification. |
| `planned_agent_type`, `planned_provider`, `planned_model`, `planned_effort` | `TEXT` | Resolved worker metadata; no prompt. |
| `item_status` | `TEXT NOT NULL CHECK (item_status IN ('planned','claimed','running','completed','failed','blocked','paused','skipped','cancelled'))` | Item lifecycle. |
| `claim_session_id`, `worker_session_id` | `TEXT` | Correlation to claim and host work session records. |
| `outcome` | `TEXT` | Semantic child outcome; nullable before completion. |
| `skip_reason` | `TEXT` | Bounded reason for exclusion or non-dispatch. |
| `evidence` | `TEXT` | Bounded summary and project-relative artifact references only. |
| `attempt` | `INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0)` | Explicit retry accounting. |
| `started_at`, `completed_at` | `TIMESTAMP` | Per-item lifecycle timestamps. |
| `created_at`, `updated_at` | `TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP` | Audit timestamps. |

Enforce `UNIQUE(team_run_id, child_type, child_key)` and add indexes for
`(team_run_id, wave, item_status)`, `(child_type, child_key)`,
`claim_session_id`, and `worker_session_id`. The ledger links claim/session
identifiers but does not replace `entity_claims` or `work_sessions`.

The migration shall be additive and idempotent. Increment
`CurrentSchemaVersion` from 27 to 28, create both tables and indexes from
`migrateTeamRunTables`, and call it after the existing work-session migration
in `runMigrations`. Existing schema initialization and `ApplySchemaIfNeeded`
must continue to apply the migration on fresh and upgraded databases.

### API and interface contracts

The following are service-level contracts. Names are Go identifiers; these are
interfaces and data shapes, not implementation code.

#### Planner

The `Planner` interface has one context-first operation:
`Plan(ctx context.Context, input PlanInput) (*TeamPlan, error)`. `PlanInput`
contains `RootType models.EntityType`, `RootKey string`,
`RequestedConcurrency int`, and `Capabilities CapabilityFacts`.

`Plan` accepts only `epic` or `feature` roots. It returns a complete snapshot
or a validation error. It does not reserve, claim, transition, note, dispatch,
or write.

#### Dispatch-step resolver

The `DispatchStepResolver` interface has one context-first operation:
`Resolve(ctx context.Context, entityType models.EntityType, key string)
(DispatchStep, error)`.

`DispatchStep` contains `EntityKey`, `EntityType`, `Status`, `Action`,
`AgentType`, `Provider`, `Model`, `Effort`, gate/pause classification, and
unresolved-placeholder diagnostics. It may contain a rendered `Prompt` only in
memory for the eventual dispatcher; `TeamPlan` and ledger conversion must
discard it.

#### Ledger service

The `Ledger` interface exposes these context-first operations:

- `PersistConfirmedPlan(ctx context.Context, plan *TeamPlan,
  rootSessionID string) (*TeamRun, error)`;
- `GetRun(ctx context.Context, runID int64) (*TeamRun, error)`;
- `ListItems(ctx context.Context, runID int64) ([]*TeamRunItem, error)`;
- `UpdateRun(ctx context.Context, update RunUpdate) (*TeamRun, error)`; and
- `RecordItemResult(ctx context.Context, update ItemResultUpdate)
  (*TeamRunItem, error)`.

`PersistConfirmedPlan` owns the transaction boundary and inserts the run plus
all planned items before returning. `UpdateRun` and `RecordItemResult` must
validate allowed state changes and use idempotency keys derived from run/item
identity and attempt. Later features may call these methods, but only F02/F03
may supply runtime claim, worker, or aggregate values.

Idempotency is explicit: the same root plus canonical plan hash is the same
confirmed run and returns the existing run without inserting duplicate
membership. The same root with a different hash is plan drift and returns
`ErrPlanDrift`; callers must create a new explicit run rather than silently
merge snapshots. Repeating the same terminal item result for the same attempt
returns the stored result, while a different terminal result for that attempt
returns `ErrConflictingTerminalResult`. Neither case increments the attempt.

Capability selection is also deterministic. A host that can execute one
ordinary worker but cannot safely run a team selects sequential mode and records
the reason. A host that cannot execute even one required workflow step returns a
typed capability error before any ledger mutation. Unknown or overlapping file
ownership is safe in sequential mode but is never treated as safe parallel
execution.

#### Shared I-01 result shape

`TeamRunResult` is the stable cross-feature shape from E38 architecture §4.2:

- root key and root type;
- run ID, run status, execution mode, concurrency limit, and plan hash;
- aggregate outcome and next action, nullable before F03 aggregation;
- complete item results with child key/type, wave, planned and actual worker
  metadata, item status, claim/session IDs, outcome, skip reason, evidence
  links, timestamps, and attempt.

### Key technical decisions

#### Use `internal/team` instead of extending `RunController`

F01 follows the service-layer rule in `internal/services` and the E38 ADR-001
decision. Team planning spans multiple children and persistence; it is not a
single-entity run loop. `internal/runner/controller.go` remains the ordinary
single-entity execution contract, and F02 can consume the team interfaces
without adding team state to `RunController`.

#### Use a normalized ledger instead of claims, sessions, or files

F01 follows E38 ADR-002 and the existing SQLite repository/migration pattern in
`internal/db/db.go`, `internal/repository/claim/claim_repository.go`, and
`internal/repository/worksession/repository.go`. Claims are leases and work
sessions are timing telemetry; neither records complete membership, attempts,
skip reasons, plan drift, or durable aggregate state. A filesystem ledger could
diverge from SQLite and is therefore not authoritative.

#### Reuse the canonical dispatch resolution

F01 follows E38 ADR-003. The current `resolveNext` path in
`internal/cli/commands/next.go` and prompt assembly used by `run.go` are the
behavioral source of truth. Extract a shared service-level resolver and keep
`NextResponse` as the CLI wire contract. Team code must not invoke Cobra,
rebuild prompts, or persist prompt content.

#### Make dependency compatibility explicit

New dependency writes use `entity_relationships` and its typed relationship
service. For task prerequisites, the planner adapter calls the existing
`TaskRepository.GetTaskDependencies`, which deliberately reads both legacy
`tasks.depends_on` JSON and polymorphic `entity_relationships` and de-duplicates
by task key. The adapter treats malformed JSON or missing targets as errors.
This preserves existing data while making the polymorphic table the canonical
source for new relationships; F01 does not silently ignore one representation.

#### Keep preview side-effect free and persistence atomic

F01 follows E38 ADR-002 and the existing service transaction rule in
`.claude/rules/services/service-design.md`: planning reads through injected
interfaces; `PersistConfirmedPlan` begins a short transaction, inserts the run
and every item, commits, and only then permits F02 to claim a child. The
repository owns SQL and constraints; the service owns validation,
idempotency, and transaction semantics.

#### Preserve root/worker ownership boundaries

F01 follows E38 ADR-005 and the parent-loop ownership contract. F01 stores a
root session correlation and child claim/session fields but does not acquire or
release them. Workers never gain status-transition authority from the ledger;
F02/F03 remain responsible for runtime claims and configured routing.

### Integration with existing code

| Existing seam | F01 integration |
|---|---|
| `internal/services/cascade_service.go` — `CascadeService.DescribeDispatchableChildren(ctx, entityType, key)` | Extract and reuse direct-child enumeration for complete snapshots. Keep terminal filtering, dependency filtering, and first-child cascade behavior unchanged for ordinary `shark next`. |
| `internal/repository/epic/repository.go`, `internal/repository/feature/repository.go`, `internal/repository/task/repository.go` | Supply canonical entity lookup, child listing, status, execution order, priority, and task dependency reads through interfaces owned by `internal/team`. |
| `internal/services/entity_relationship_service.go`, `internal/repository/entityrel/repository.go`, `internal/dependency/detector.go` | Normalize typed `depends_on` edges, detect cycles, resolve external prerequisites, and produce deterministic graph input. |
| `internal/config/workflow/schema.go`, `internal/config/action/orchestrator.go`, `internal/cli/commands/next.go` | Resolve workflow responsibility, action, provider/model/effort, terminal/gate behavior, and placeholder diagnostics through the shared dispatch-step resolver. Never hardcode terminal or success statuses. |
| `internal/runner/controller.go` and `internal/runner/dispatcher.go` | F01 exposes metadata compatible with `runner.DispatchInput` and the existing `AgentDispatcher`; it does not call or modify the worker dispatch loop. |
| `internal/services/claim_service.go`, `internal/repository/claim/claim_repository.go` | F01 reads claim diagnostics through an injected interface for plan exclusions only. Claim mutation remains F02-owned. A preview claim check is point-in-time and is not a reservation. |
| `internal/models/work_session.go`, `internal/repository/worksession/repository.go` | Store string session correlation fields only; retain work-session ownership and timing semantics. F01 never treats `work_sessions` as the plan ledger. |
| `internal/db/db.go`, `internal/repository/dbconn/db.go` | Follow additive migrations, `ApplySchemaIfNeeded`, canonical UTC timestamp formatting, WAL/busy-timeout settings, and transaction helpers. |
| `internal/cli/services_global.go`, `cmd/server/services.go` | Add CLI service construction in F01. Do not add an HTTP endpoint or second orchestration implementation; a future server adapter must inject the same service. |

## Cross-feature interactions

### Produces: I-01 — Team-run domain contract

- **Consumers:** E38-F02 Scheduler and Claims; E38-F03 Aggregate Routing and
  Resume; E38-F05 Reporting and Operator Surface.
- **Shape source:** E38 architecture §4.2 Team-run domain contract.
- **Contract:** `TeamPlan`, `team_runs`, `team_run_items`, stable plan hash,
  dependency wave, item status, planned/actual worker metadata, claim/session
  links, attempt, outcome, skip reason, evidence, and lifecycle timestamps.
- **Contract tests:** `tests/contracts/e38_interactions_test.go#TC-001`.
  This single test proves the producer/consumer shape; consuming features must
  reference the same pointer rather than create twin contract tests.

### Consumes

None. F01 is execution order 1 and has no upstream I-## dependency.

## Cross-epic integrations

None. The parent E38 cross-epic map assigns its rows to later E38 features;
F01’s use of existing repository, workflow, claim, session, and
runner seams is internal implementation reuse, not an X-## product boundary.

## Exit-gate checklist

- [x] Every requirement has a testable statement or named verification.
- [x] Requirements are incremental and trace to the E38 epic PRD/UAT plan.
- [x] All planned code and test paths are listed.
- [x] The migration, constraints, indexes, and idempotency rules are explicit.
- [x] Architecture decisions reference current repository patterns or explain
      the deviation.
- [x] I-01 is declared with the exact parent shape source and one contract-test
      pointer for all consumers.
- [x] No X-## integration is invented or assigned to F01.
- [x] No critical section has an unresolved decision.
