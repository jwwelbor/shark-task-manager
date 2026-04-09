---
feature_key: E07-F38
title: Auto-reopen parent entities on child regression — Combined Spec
status: in_specification
last_updated: 2026-04-06
references:
  - feature.md
  - prd-source.md      # Original epic-level PRD (E26, cancelled and rescoped)
  - architecture.md    # Original epic-level architecture (E26)
  - research.md        # Brownfield analysis of existing helpers and extension points
---

# E07-F38: Auto-reopen Parent Entities on Child Regression — Specification

This is the **combined requirements + architecture** document for feature E07-F38. It is the single source of truth for what to build and how to build it. The original epic-level PRD (`prd-source.md`) and architecture (`architecture.md`) — drafted when this work was scoped as cancelled epic E26 — remain in this directory as reference but are **not** the implementation contract; this document is.

For business context, the canonical UAT scenarios, and the full problem analysis, see `prd-source.md` Sections 1, 5, and 6 (UAT-1 through UAT-8). For the full ADR set with rationale, alternatives, and consequences, see `architecture.md` Section 2 (ADR-001 through ADR-006).

---

## Part 1 — Requirements

### 1.1 Scope (incremental over the parent epic)

E07 is an enhancements epic; E07-F38 is the only feature delivering the auto-reopen cascade. Everything below is **net new behavior** introduced by this feature.

In scope:

- Backward-transition detection in `TaskService.TransitionStatus` and `FeatureService.TransitionStatus` (post-hook, after the existing post-hook block).
- A unified cascade helper that walks `task → feature → epic`, reopening any closed parent atomically inside its own transaction.
- History-based reopen target lookup (last non-terminal status from `entity_history`) with workflow-driven fallback chain.
- Audit row in `entity_history` for each auto-reopen, with a structured `notes` prefix that is greppable and renderable.
- `shark status history` formatter update to label auto-reopen rows distinctly (CLI presentation only).
- Unification of the existing creation-trigger reopens (`maybeReopenParentFeature`, `maybeReopenParentEpic`) onto the same helper — single source of truth.
- Tests covering basic and advanced workflow profiles, idempotency, fallback paths, and the negative case for bugs/change-cards.

Out of scope (explicit):

- Bugs and change-cards — they have no parent rollup and never trigger cascade. (Structural exclusion: no hook in `BugService` or `ChangeCardService`.)
- Forward cascades (auto-completing parents when all children become terminal). Already handled by existing rollup code.
- Retroactive backfill of parents that are stale today. Cascade activates on transitions occurring **after** the feature ships.
- New CLI commands or HTTP endpoints. Behavior is observable only through additional `entity_history` rows and updated parent statuses.
- Schema migrations. The `entity_history.notes` column is sufficient to carry the auto-reopen reason. `CurrentSchemaVersion` does **not** change. `skip_migrations` does **not** need to be flipped. (Conditional exception: see REQ-N-005 below — if the existing `entity_history` index does not cover the lookup query, an index addition is the only schema change permitted, and it follows the standard migration protocol.)
- Disabling cascades for orchestrator-driven transitions. Orchestrator transitions go through the same `TaskService.TransitionStatus` and cascade the same way.

### 1.2 Functional Requirements

Each requirement traces back to a UAT scenario in `prd-source.md` Section 6 and to a success criterion (SC) in `prd-source.md` Section 2.

**REQ-F-001** (UAT-1, SC1, SC4)
When a task transitions backward — i.e., the `from_status` is terminal per `workflow.Service.IsTerminalStatus(LevelTask)` and the `to_status` is non-terminal — the cascade MUST reopen every terminal ancestor in the chain (parent feature, then parent epic) within a single cascade transaction.

**REQ-F-002** (UAT-1, SC1)
When a feature transitions backward (terminal → non-terminal), the cascade MUST reopen the parent epic if and only if the parent epic is currently in a terminal status. The cascade MUST NOT walk further up (no concept above epic).

**REQ-F-003** (SC2, UAT-2)
The existing creation-trigger reopens (`TaskService.maybeReopenParentFeature` called from `CreateTask`, and `FeatureService.maybeReopenParentEpic` called from `CreateFeature`) MUST be refactored to delegate to the same cascade helper used by the backward-transition trigger. After refactoring, no parallel implementation of "reopen parent" logic exists. (Satisfies SC9.)

**REQ-F-004** (SC3, UAT-1, UAT-6)
For each ancestor that the cascade reopens, the target status MUST be resolved using a three-step lookup, in order:
1. The most recent row in `entity_history` for that ancestor whose `to_status` is **not** in the terminal-status set for that entity type's workflow level.
2. If no such row exists, `workflow.Service.ForLevel(level).GetAggregationStatuses()[0]`, when the slice is non-empty.
3. Otherwise, `workflow.Service.ForLevel(level).GetInitialStatusString()`.

When the resolver falls back to step 2 or 3, the auto-reopen audit row MUST include a `[fallback: ...]` suffix in its `notes` field indicating which fallback was used.

**REQ-F-005** (SC5, UAT-4)
The cascade MUST NOT hardcode any status name. Terminal classification is computed via `workflow.Service.IsTerminalStatus`. The terminal-status set used in the SQL `NOT IN (...)` clause for the history lookup is computed per-cascade by enumerating the active workflow profile's statuses for the relevant level and filtering by `IsTerminalStatus`. The behavior is identical on basic, advanced, and custom workflow profiles with no code changes.

**REQ-F-006** (SC6, UAT-5)
Bugs and change-cards MUST NOT trigger any cascade reopen. This is enforced structurally: no cascade hook exists in `BugService.TransitionStatus` or `ChangeCardService.TransitionStatus`. A regression on a bug or change-card linked to a `completed` feature leaves the feature `completed` and writes no auto-reopen history rows.

**REQ-F-007** (SC7, UAT-7)
Each auto-reopen MUST write exactly one `entity_history` row per ancestor that was reopened. The row's `notes` column MUST begin with the literal prefix `auto_reopen:` and MUST identify the triggering child's key and the trigger kind. The two canonical reason formats are:

```
auto_reopen: triggered by <triggering-key> regression (<trigger-entity-type>)
auto_reopen: triggered by <triggering-key> creation (<trigger-entity-type>)
```

When the reopen target was resolved via fallback, an additional ` [fallback: aggregation]` or ` [fallback: initial]` suffix MUST be appended.

**REQ-F-008** (SC8, UAT-3)
The cascade MUST be idempotent. If the cascade walks to an ancestor that is already in a non-terminal status, it MUST skip the update for that ancestor and MUST NOT write a history row for it. The cascade continues walking up the chain because a non-terminal feature does not preclude a terminal epic. Idempotency is implemented by re-fetching each ancestor's current status **inside** the cascade transaction immediately before the update.

**REQ-F-009** (UAT-7)
`shark status history <key>` MUST visually distinguish auto-reopen rows from manually-triggered transitions in both human and `--json` output. In human output, rows whose `notes` begin with `auto_reopen:` MUST be rendered with a distinct label or color. In `--json` output, no schema change is required — the existing `notes` field is sufficient because consumers can filter on the `auto_reopen:` prefix.

**REQ-F-010** (UAT-8)
After an auto-reopen fires, all existing dashboard rollup commands (`shark status`, `shark epic status`, `shark feature get`, `shark progress`) MUST reflect the reopened state immediately on the next read. No additional invalidation logic is required because rollups always read live from the database; this requirement is satisfied by REQ-F-001/REQ-F-002 plus the existing rollup code paths and exists here only as a regression-prevention guard for the test suite.

### 1.3 Non-Functional Requirements

**REQ-N-001 (Performance, SC4)**
The cascade overhead MUST be ≤50ms (P95) per child transition on a typical project. The architectural cost analysis (architecture.md Section 6) shows worst-case ~17ms with the existing schema. The 50ms budget includes both legs of the cascade (feature update + history write + epic update + history write) plus the transaction begin/commit.

**REQ-N-002 (Atomicity, partial SC1)**
All parent updates inside a single cascade invocation MUST be atomic relative to each other. If the feature update succeeds but the epic update fails, the cascade transaction MUST roll back, leaving both parents at their pre-cascade status. The cascade transaction is **separate** from the child transition's transaction (see ADR-003 in `architecture.md` for the rationale of the two-phase commit). The atomicity guarantee is therefore scoped to "the parent chain" rather than "the full child + parent chain".

**REQ-N-003 (Failure handling)**
If the cascade transaction fails at any point (BeginTx, query, update, or Commit), the failure MUST NOT bubble up to the caller of `TransitionStatus`. The original child transition succeeds and returns normally. The cascade failure MUST be logged at WARN level via `slog.Warn` with structured fields including the triggering child key, the ancestor that failed (if known), and the underlying error. This matches the existing best-effort idiom of `recalculateFeatureProgress` and `maybeReopenParentFeature`.

**REQ-N-004 (Backward compatibility)**
No existing CLI command, JSON output schema, or exit code changes. New behavior is observable only through (a) updated parent statuses, (b) new `entity_history` rows with the `auto_reopen:` prefix, and (c) the cosmetic label in `shark status history`. Existing automation, tests, and consumers continue to work unchanged.

**REQ-N-005 (Schema)**
No schema change unless verification (during implementation) shows that the existing `entity_history` indexes do not support the lookup query `WHERE entity_type = ? AND entity_id = ? AND to_status NOT IN (...) ORDER BY changed_at DESC LIMIT 1`. If an index addition is required, it MUST follow the project migration protocol in `.claude/rules/database-critical.md`: add the migration in `internal/db/db.go`, bump `CurrentSchemaVersion` from 11 to 12, and notify the developer to flip `skip_migrations: false` in `.sharkconfig.json` for one shark command. No table or column additions are permitted under any circumstances.

**REQ-N-006 (Test coverage)**
Service-layer cascade logic MUST achieve ≥80% line coverage. All error paths (BeginTx failure, history-lookup error, repository update failure, commit failure) MUST have explicit unit tests. Repository-layer additions (`GetLastNonTerminalStatus`, `UpdateStatusTx` on feature/epic, `CreateTx` on entity history) MUST have repository tests against the real test database per `.claude/rules/testing/repository-tests.md`.

**REQ-N-007 (Code quality gate)**
`make fmt && make lint && make test` MUST pass cleanly. No new lint warnings introduced.

### 1.4 Acceptance Criteria

The acceptance criteria below are testable and map directly to requirements above. Each is owned by a specific test or set of tests.

| AC | Description | Verifying tests |
|----|-------------|-----------------|
| AC-01 | A task transition from terminal to non-terminal reopens its parent feature to that feature's most recent prior non-terminal status. | `TestCascade_TaskBackwardReopensFeature` (new, in `cascade_reopen_test.go`) |
| AC-02 | The same trigger also reopens the grandparent epic when the epic is currently in a terminal status. | `TestCascade_TaskBackwardReopensEpic` (new) |
| AC-03 | A feature transition from terminal to non-terminal reopens its parent epic but does NOT touch any other entity. | `TestCascade_FeatureBackwardReopensEpic` (new) |
| AC-04 | If the parent feature is already non-terminal, the cascade SKIPS it but STILL checks the epic. | `TestCascade_NonTerminalFeatureContinuesToEpic` (new) |
| AC-05 | If both parents are non-terminal, the cascade is a complete no-op (no DB writes other than the BeginTx/Commit envelope). | `TestCascade_AllAncestorsNonTerminalNoOp` (new) |
| AC-06 | Each auto-reopen writes an `entity_history` row whose `notes` begins with `auto_reopen:` and includes the triggering child key. | `TestCascade_HistoryRowFormat` (new) |
| AC-07 | When no prior non-terminal history exists for an ancestor, the resolver falls back to aggregation status, and the history row's `notes` includes `[fallback: aggregation]`. | `TestResolveReopenTarget_FallbackAggregation` (new) |
| AC-08 | When neither history nor aggregation statuses exist, the resolver falls back to the workflow's initial status, and the history row's `notes` includes `[fallback: initial]`. | `TestResolveReopenTarget_FallbackInitial` (new) |
| AC-09 | A second concurrent regression on a different child of the same feature is a no-op (idempotent), confirmed by counting `entity_history` rows after both regressions. | `TestCascade_IdempotentOnSecondRegression` (new) |
| AC-10 | Existing `task_service_test.go` tests for `maybeReopenParentFeature` (lines 2173–2617) continue to pass after the helper is refactored to delegate to the cascade. | Existing test suite (regression check) |
| AC-11 | A bug regressing from `closed` to `open` while linked to a `completed` feature does NOT reopen the feature and writes no `auto_reopen:` history rows. | `TestCascade_BugDoesNotTriggerCascade` (new, integration-level) |
| AC-12 | The cascade fires correctly under both the basic and advanced workflow profiles. The same cascade test is parameterized over both profile configs and asserts the same behavior. | `TestCascade_BasicProfile` and `TestCascade_AdvancedProfile` (new, table-driven) |
| AC-13 | Cascade transaction failure (simulated via mock error on `Commit`) does NOT fail the original child transition; the original returns success and a WARN log is emitted. | `TestCascade_TxFailureIsNonBlocking` (new) |
| AC-14 | `shark status history E07-F01` renders auto-reopen rows with a distinct visual label in human output. JSON output is unchanged but consumers can filter by `notes` prefix. | `TestStatusHistoryFormatter_AutoReopenLabel` (new, in CLI commands package, mocked) |
| AC-15 | Cascade overhead is ≤50ms P95 measured by a benchmark over 1000 runs of the worst-case path (both legs fire). | `BenchmarkCascade_BothLegs` (new) |
| AC-16 | `make fmt && make lint && make test` exits 0 with no warnings introduced. | CI gate |

### 1.5 Out of Scope (explicit, requirements-level)

These were considered and consciously excluded:

- A new `is_terminal: true` flag on `status_metadata`. The existing `_complete_` special-status group is sufficient. Adding the flag is a refinement-time decision deferred per ADR-006 in `architecture.md`.
- A per-profile `auto_reopen_enabled` opt-out toggle. Default-on is the unanimous user preference; opt-out can be added later as a one-line gate at the top of `cascadeParentReopens` if needed.
- A per-workflow `reopen_fallback` configurable status override. The three-step resolver chain in REQ-F-004 is sufficient for v1; a config override can be inserted between steps 1 and 2 later.
- Notifications, webhooks, or external alerting on cascade events. Only `slog.Warn` on cascade failure.
- Cross-epic cascades (reopening sibling or upstream-dependency entities). Cascade walks `task → its own feature → its own epic` only.
- A retroactive migration that scans existing data and reopens parents whose children are non-terminal today. Cascade activates on **new** transitions only.

---

## Part 2 — Architecture

This section is the implementation blueprint. It mirrors the structure of `architecture.md` but is condensed to the parts that bind to E07-F38 requirements. For full ADR rationale, alternatives considered, and the deferred-decisions discussion, read `architecture.md` Sections 2 and 6 in full.

### 2.1 Component Changes

#### 2.1.1 New files

| Path | Purpose | Approx LOC |
|------|---------|------------|
| `internal/services/cascade_reopen.go` | Hosts `cascadeParentReopens`, `resolveReopenTarget`, `cascadeDeps` struct, `cascadeTrigger` struct, and the `ParentReopenHistoryQuerier` interface. Single source of truth for parent-reopen logic. | ~180 |
| `internal/services/cascade_reopen_test.go` | Unit tests for the cascade helper using mocks for `ParentReopenHistoryQuerier`, `CascadeFeatureRepo`, `CascadeEpicRepo`, `EntityHistoryTxRecorder`. Covers AC-01..AC-09, AC-13. | ~400 |

#### 2.1.2 Files modified

| Path | Change | Rationale |
|------|--------|-----------|
| `internal/repository/entityhistory/repository.go` | Add `GetLastNonTerminalStatus(ctx, entityType, entityID, terminalStatuses []string) (string, bool, error)`. Add `CreateTx(ctx, tx *sql.Tx, history *models.EntityHistory) error`. | REQ-F-004 (history lookup), REQ-N-002 (transactional history write). |
| `internal/repository/entityhistory/repository_test.go` | Add tests for `GetLastNonTerminalStatus` (happy path, empty result, terminal-set filtering) and `CreateTx`. | REQ-N-006. |
| `internal/repository/feature_repository.go` | Add `UpdateStatusTx(ctx, tx *sql.Tx, id int64, status string, agent *string, notes *string) error` (mirrors existing `UpdateStatus` but accepts `*sql.Tx`). | REQ-N-002 (cascade owns transaction). |
| `internal/repository/feature_repository_test.go` | Add test for `UpdateStatusTx` including rollback verification. | REQ-N-006. |
| `internal/repository/epic_repository.go` | Add `UpdateStatusTx(ctx, tx *sql.Tx, id int64, status string, agent *string, notes *string) error`. | REQ-N-002. |
| `internal/repository/epic_repository_test.go` | Add test for `UpdateStatusTx` including rollback verification. | REQ-N-006. |
| `internal/services/task_service.go` | (a) Inject new optional dependencies via constructor (db handle, feature/epic repo for cascade reads, history querier, history Tx recorder). (b) Add cascade post-hook in `TransitionStatus` after `recalculateFeatureProgress` (currently line 618). (c) Refactor `maybeReopenParentFeature` (currently line 1054) to call the unified cascade helper. | REQ-F-001, REQ-F-003. |
| `internal/services/task_service_test.go` | Update existing `maybeReopenParentFeature` tests (lines 2173–2617) to validate they pass against the refactored implementation. Add new tests for the `TransitionStatus` post-hook backward-trigger path. | AC-10, REQ-F-001. |
| `internal/services/feature_service.go` | (a) Inject new optional dependencies. (b) Add cascade post-hook in `TransitionStatus` (currently line 209) after the existing child-task counter call. (c) Refactor `maybeReopenParentEpic` (currently line 788) to call the unified cascade helper, replacing the current `aggStatuses[0]` jump with the history-lookup resolver. | REQ-F-002, REQ-F-003. |
| `internal/services/feature_service_test.go` | Update existing `maybeReopenParentEpic` tests and add new tests for the `TransitionStatus` post-hook path. | REQ-F-002, REQ-F-003. |
| `internal/services/backward_transition_test.go` | Extend the existing backward-transition test file with cascade-aware assertions, particularly for the basic vs advanced profile parameterization (AC-12). | AC-12. |
| `internal/cli/services_global.go` | Wire the new dependencies into `GetTaskService()` and `GetFeatureService()`. New deps are constructed lazily from the existing global `*repository.DB` and `*workflow.Service`. | Required for cascade to fire in CLI commands. |
| `cmd/server/services.go` | Wire the same dependencies into `WireServices()` for the HTTP API. | Parity with CLI. |
| `internal/cli/commands/status.go` (or the formatter that handles `shark status history`) | Detect rows whose `notes` begin with `auto_reopen:` and render with a distinct label in human output. | REQ-F-009, AC-14. |
| `internal/cli/commands/status_test.go` (or wherever the formatter is tested) | Add a formatter test using a mocked history list with one auto-reopen row. | AC-14. |

#### 2.1.3 Files NOT modified (deliberately)

| Path | Why untouched |
|------|---------------|
| `internal/services/entity_service.go` (`EntityService.TransitionStatus`) | Per ADR-001: keeping the cascade out of `EntityService` means Bug and ChangeCard transitions are not polluted with parent-walking logic. The cascade is added as a post-hook **in the calling entity service**, after `EntityService.TransitionStatus` returns. |
| `internal/services/bug_service.go`, `change_card_service.go` | REQ-F-006: bugs and change-cards never cascade. Structural exclusion — no hook added. |
| `internal/services/epic_service.go` | Epic is the top of the hierarchy. No parent to reopen. |
| `internal/workflow/service.go` | All terminal classification, fallback, and level scoping use the existing API (`IsTerminalStatus`, `ForLevel`, `GetAggregationStatuses`, `GetInitialStatusString`). |
| `internal/db/db.go` | No schema migration unless the index check (REQ-N-005) discovers a missing index. If touched, it is for an index addition only and must follow the standard `CurrentSchemaVersion` bump protocol. |
| `entity_history` table schema | The `notes` column is sufficient to carry the auto-reopen reason per ADR-005. |

### 2.2 Data Model Changes

**None.** The `entity_history` table is reused as-is. The `notes` column carries the auto-reopen reason per the structured prefix scheme defined in REQ-F-007. No new tables, no new columns, no `CurrentSchemaVersion` bump.

The conditional exception in REQ-N-005 (potential index addition) is the only schema change permitted, and only if the implementation phase verifies the existing index does not cover the lookup query. If verified to already cover the query, the change is zero.

### 2.3 API and Interface Contracts

#### 2.3.1 Repository-layer additions

```go
// internal/repository/entityhistory/repository.go

// GetLastNonTerminalStatus returns the to_status of the most recent entity_history
// row for (entityType, entityID) where to_status is not in terminalStatuses.
//
// Returns (status, true, nil) on hit, ("", false, nil) on no rows, ("", false, err) on DB error.
//
// SQL:
//   SELECT to_status FROM entity_history
//   WHERE entity_type = ? AND entity_id = ?
//     AND to_status NOT IN (?, ?, ...)
//   ORDER BY changed_at DESC LIMIT 1
func (r *EntityHistoryRepository) GetLastNonTerminalStatus(
    ctx context.Context,
    entityType models.EntityType,
    entityID int64,
    terminalStatuses []string,
) (string, bool, error)

// CreateTx writes a new entity_history row inside an existing transaction.
// Mirrors Create() but accepts *sql.Tx for cascade-owned transaction use.
func (r *EntityHistoryRepository) CreateTx(
    ctx context.Context,
    tx *sql.Tx,
    history *models.EntityHistory,
) error
```

```go
// internal/repository/feature_repository.go

// UpdateStatusTx updates a feature's status inside an existing transaction.
// Mirrors UpdateStatus() but accepts *sql.Tx so the cascade can own the transaction
// and roll back atomically across feature + epic updates.
func (r *FeatureRepository) UpdateStatusTx(
    ctx context.Context,
    tx *sql.Tx,
    id int64,
    status string,
    agent *string,
    notes *string,
) error
```

```go
// internal/repository/epic_repository.go

// UpdateStatusTx updates an epic's status inside an existing transaction.
func (r *EpicRepository) UpdateStatusTx(
    ctx context.Context,
    tx *sql.Tx,
    id int64,
    status string,
    agent *string,
    notes *string,
) error
```

#### 2.3.2 Service-layer additions

```go
// internal/services/cascade_reopen.go

// ParentReopenHistoryQuerier is the narrow interface the cascade needs
// for resolving each ancestor's prior non-terminal status. Defined at
// point of use per the project's "interface at consumer side" convention.
type ParentReopenHistoryQuerier interface {
    GetLastNonTerminalStatus(
        ctx context.Context,
        entityType models.EntityType,
        entityID int64,
        terminalStatuses []string,
    ) (string, bool, error)
}

// CascadeFeatureRepo is the narrow read+Tx-update interface the cascade needs
// for the feature ancestor.
type CascadeFeatureRepo interface {
    GetByID(ctx context.Context, id int64) (*models.Feature, error)
    UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error
}

// CascadeEpicRepo is the narrow read+Tx-update interface the cascade needs
// for the epic ancestor.
type CascadeEpicRepo interface {
    GetByID(ctx context.Context, id int64) (*models.Epic, error)
    UpdateStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, agent *string, notes *string) error
}

// EntityHistoryTxRecorder writes history rows inside a cascade-owned transaction.
type EntityHistoryTxRecorder interface {
    CreateTx(ctx context.Context, tx *sql.Tx, history *models.EntityHistory) error
}

// cascadeDeps bundles all dependencies the cascade helper needs. Held by
// TaskService and FeatureService and passed into the helper at call time.
type cascadeDeps struct {
    db             *repository.DB
    featureRepo    CascadeFeatureRepo
    epicRepo       CascadeEpicRepo
    historyQuerier ParentReopenHistoryQuerier
    historyTx      EntityHistoryTxRecorder
    workflowSvc    *workflow.Service
}

// cascadeTrigger describes what fired the cascade so the audit row can
// reference the trigger correctly.
type cascadeTrigger struct {
    triggerKey  string                    // e.g. "E07-F01-003"
    triggerKind string                    // "regression" or "creation"
    triggerType models.EntityType         // "task" or "feature"
    startLeg    cascadeLeg                // "feature" (start at feature leg) or "epic" (skip feature, start at epic)
    featureID   int64                     // feature to start from (or feature whose epic to walk if startLeg=="epic")
}

// cascadeParentReopens walks the parent chain from cascadeTrigger up to the epic,
// reopening any terminal ancestor inside a single owned transaction.
//
// CONTRACT:
//  - Best-effort: errors do NOT propagate to the caller. They are logged via slog.Warn
//    with structured fields including triggerKey, ancestor, and error.
//  - Atomic across the parent chain (feature + epic): on any error, the cascade Tx
//    rolls back and neither parent is updated.
//  - Idempotent: ancestors already non-terminal are skipped (no update, no history row),
//    but the walk continues up the chain.
//  - Profile-agnostic: terminal classification uses workflow.Service.IsTerminalStatus
//    for the appropriate level. No status names are hardcoded.
func cascadeParentReopens(ctx context.Context, deps cascadeDeps, trigger cascadeTrigger)

// resolveReopenTarget implements the three-step fallback chain for choosing
// where to reopen an ancestor. Returns (status, fallbackKind) where fallbackKind
// is "" (history hit), "aggregation", or "initial".
func resolveReopenTarget(
    ctx context.Context,
    historyQuerier ParentReopenHistoryQuerier,
    entityType models.EntityType,
    entityID int64,
    levelWorkflow *workflow.Service,
) (status string, fallbackKind string, err error)
```

#### 2.3.3 Constructor signature changes

`NewTaskService` and `NewFeatureService` gain new parameters. All new parameters are optional (nil-safe) — when any one is nil, the cascade is silently disabled. This preserves backward compatibility for tests and any embedder.

```go
// internal/services/task_service.go

func NewTaskService(
    repo TaskRepository,
    workflowSvc *workflow.Service,
    creatorSvc *taskcreation.Creator,
    noteRepo TaskNoteRepository,
    // NEW (cascade dependencies — all optional, nil-safe)
    db *repository.DB,
    featureRepo CascadeFeatureRepo,
    epicRepo CascadeEpicRepo,
    historyQuerier ParentReopenHistoryQuerier,
    historyTxRecorder EntityHistoryTxRecorder,
) *TaskService
```

```go
// internal/services/feature_service.go

func NewFeatureService(
    repo FeatureRepository,
    workflowSvc *workflow.Service,
    noteRepo FeatureNoteRepository,
    taskCounter FeatureTaskCounter,
    // NEW (cascade dependencies — all optional, nil-safe)
    db *repository.DB,
    epicRepo CascadeEpicRepo,
    historyQuerier ParentReopenHistoryQuerier,
    historyTxRecorder EntityHistoryTxRecorder,
) *FeatureService
```

Note that `FeatureService` does not need a `CascadeFeatureRepo` because the cascade from a feature transition starts directly at the epic leg (the feature being transitioned **is** itself the trigger; the cascade walks upward from it).

### 2.4 Key Technical Decisions

The full ADR set with rationale and alternatives is in `architecture.md` Section 2. Summary of the binding decisions for implementation:

| ADR | Decision | Why it matters for implementation |
|-----|----------|-----------------------------------|
| **ADR-001** | Cascade lives in `internal/services/cascade_reopen.go`, not in `EntityService`. | Bug and ChangeCard services have no cascade hook, satisfying REQ-F-006 structurally. There is exactly one implementation of "reopen parent", satisfying REQ-F-003. |
| **ADR-002** | Reopen target = "last non-terminal status from history", with a workflow-driven fallback chain (aggregation → initial). | Honors REQ-F-004 and UAT-1's expectation that a feature in `in_qa` returns to `in_qa`, not to a generic entry status. |
| **ADR-003** | Two-phase commit: child transition commits first via `EntityService.TransitionStatus`, then the cascade opens its own transaction for the parent updates. | Avoids a multi-epic refactor of `EntityService` to thread `*sql.Tx` through every repository. The atomicity guarantee is scoped to the parent chain (REQ-N-002), and cascade failure is non-blocking (REQ-N-003). |
| **ADR-004** | Idempotency via re-fetching each ancestor's status **inside the cascade transaction** before update. | Closes the race window for concurrent regressions (REQ-F-008). No locks, no uniqueness constraints, no advisory locking needed. |
| **ADR-005** | Audit trail uses `entity_history.notes` with `auto_reopen:` prefix. No schema change. | Satisfies REQ-F-007 with zero migration cost. Greppable, filterable, and easy to render. |
| **ADR-006** | Three deferred refinement decisions (`is_terminal` flag, `auto_reopen_enabled` opt-out, `reopen_fallback` override) are NOT pre-resolved. The architecture is built so each decision slots into a well-defined extension point without restructuring. | Implementation does not need to resolve them. Future work can add them in one place each. |

### 2.5 Integration with Existing Code

#### 2.5.1 Trigger A: Task transition post-hook

In `internal/services/task_service.go`, inside `TransitionStatus`, after the existing post-hook block at line 618:

```go
// Existing post-hook (already present):
s.recalculateFeatureProgress(ctx, task.FeatureID)

// NEW: cascade post-hook
if s.cascadeEnabled() {
    taskWf := s.workflowSvc.ForLevel(workflow.LevelTask)
    if taskWf.IsTerminalStatus(result.FromStatus) && !taskWf.IsTerminalStatus(result.ToStatus) {
        cascadeParentReopens(ctx, s.cascadeDeps(), cascadeTrigger{
            triggerKey:  key,
            triggerKind: "regression",
            triggerType: models.EntityTypeTask,
            startLeg:    cascadeLegFeature,
            featureID:   task.FeatureID,
        })
    }
}
```

`s.cascadeEnabled()` returns true iff all five new dependencies are non-nil (degrades gracefully). `s.cascadeDeps()` packages the dependencies into the `cascadeDeps` struct.

#### 2.5.2 Trigger B: Feature transition post-hook

In `internal/services/feature_service.go`, inside `TransitionStatus`, after the existing child task counter post-hook at line ~228:

```go
// NEW: cascade post-hook
if s.cascadeEnabled() {
    featureWf := s.workflowSvc.ForLevel(workflow.LevelFeature)
    if featureWf.IsTerminalStatus(result.FromStatus) && !featureWf.IsTerminalStatus(result.ToStatus) {
        cascadeParentReopens(ctx, s.cascadeDeps(), cascadeTrigger{
            triggerKey:  featureKey,
            triggerKind: "regression",
            triggerType: models.EntityTypeFeature,
            startLeg:    cascadeLegEpic,        // skip feature leg, start at epic
            featureID:   result.EntityID,        // used to look up the parent epic via featureRepo.GetByID
        })
    }
}
```

When `startLeg == cascadeLegEpic`, the cascade helper still calls `featureRepo.GetByID(ctx, trigger.featureID)` to get the feature record (it needs `feature.EpicID`), but it does NOT update the feature itself.

#### 2.5.3 Trigger C: Refactored CreateTask reopen

In `internal/services/task_service.go`, the existing `maybeReopenParentFeature` helper at line 1054 is refactored. The current implementation does an `aggStatuses[0]` jump and only walks one level. The refactored version:

```go
func (s *TaskService) maybeReopenParentFeature(ctx context.Context, featureKey, taskKey string) {
    if !s.cascadeEnabled() {
        // Preserve legacy behavior when cascade dependencies are not wired (e.g., in tests).
        s.legacyMaybeReopenParentFeature(ctx, featureKey, taskKey)
        return
    }

    // Look up the feature by key to get its FeatureID.
    feature, err := s.lookupFeatureByKey(ctx, featureKey)
    if err != nil || feature == nil {
        return
    }

    cascadeParentReopens(ctx, s.cascadeDeps(), cascadeTrigger{
        triggerKey:  taskKey,
        triggerKind: "creation",
        triggerType: models.EntityTypeTask,
        startLeg:    cascadeLegFeature,
        featureID:   feature.ID,
    })
}
```

The legacy fallback path (`legacyMaybeReopenParentFeature`) preserves the current behavior verbatim and is used only when cascade deps are nil. This means the existing test suite (`task_service_test.go` lines 2173–2617) — which constructs `TaskService` without cascade deps — continues to pass against the legacy fallback. Once those existing tests are updated to wire cascade deps, they exercise the new path.

The same refactoring pattern applies to `FeatureService.maybeReopenParentEpic` at line 788.

#### 2.5.4 The cascade helper body

```go
// internal/services/cascade_reopen.go

func cascadeParentReopens(ctx context.Context, deps cascadeDeps, trigger cascadeTrigger) {
    tx, err := deps.db.BeginTx(ctx, nil)
    if err != nil {
        slog.Warn("cascade: failed to begin transaction",
            "trigger_key", trigger.triggerKey, "error", err)
        return
    }
    defer func() { _ = tx.Rollback() }()

    var feature *models.Feature

    // === Feature leg (skipped if trigger started at feature itself) ===
    if trigger.startLeg == cascadeLegFeature {
        feature, err = deps.featureRepo.GetByID(ctx, trigger.featureID)
        if err != nil || feature == nil {
            slog.Warn("cascade: feature lookup failed",
                "trigger_key", trigger.triggerKey, "feature_id", trigger.featureID, "error", err)
            return
        }
        featureWf := deps.workflowSvc.ForLevel(workflow.LevelFeature)
        if featureWf.IsTerminalStatus(string(feature.Status)) {
            target, fallbackKind, terr := resolveReopenTarget(ctx, deps.historyQuerier,
                models.EntityTypeFeature, feature.ID, featureWf)
            if terr != nil {
                slog.Warn("cascade: resolve target failed (feature)",
                    "trigger_key", trigger.triggerKey, "error", terr)
                return
            }
            notes := buildAutoReopenNotes(trigger, fallbackKind)
            if uerr := deps.featureRepo.UpdateStatusTx(ctx, tx, feature.ID, target, nil, &notes); uerr != nil {
                slog.Warn("cascade: feature update failed",
                    "trigger_key", trigger.triggerKey, "feature_key", feature.Key, "error", uerr)
                return
            }
            if herr := deps.historyTx.CreateTx(ctx, tx, &models.EntityHistory{
                EntityType: models.EntityTypeFeature,
                EntityID:   feature.ID,
                FromStatus: string(feature.Status),
                ToStatus:   target,
                ChangedBy:  ptrString("system"),
                Notes:      &notes,
            }); herr != nil {
                slog.Warn("cascade: feature history write failed",
                    "trigger_key", trigger.triggerKey, "error", herr)
                return
            }
        }
        // Else: idempotent skip (REQ-F-008). Continue to epic leg.
    } else {
        // startLeg == cascadeLegEpic: still need feature.EpicID
        feature, err = deps.featureRepo.GetByID(ctx, trigger.featureID)
        if err != nil || feature == nil {
            slog.Warn("cascade: feature lookup failed (epic-only path)",
                "trigger_key", trigger.triggerKey, "error", err)
            return
        }
    }

    // === Epic leg ===
    epic, err := deps.epicRepo.GetByID(ctx, feature.EpicID)
    if err != nil || epic == nil {
        slog.Warn("cascade: epic lookup failed",
            "trigger_key", trigger.triggerKey, "epic_id", feature.EpicID, "error", err)
        return
    }
    epicWf := deps.workflowSvc.ForLevel(workflow.LevelEpic)
    if epicWf.IsTerminalStatus(string(epic.Status)) {
        target, fallbackKind, terr := resolveReopenTarget(ctx, deps.historyQuerier,
            models.EntityTypeEpic, epic.ID, epicWf)
        if terr != nil {
            slog.Warn("cascade: resolve target failed (epic)",
                "trigger_key", trigger.triggerKey, "error", terr)
            return
        }
        notes := buildAutoReopenNotes(trigger, fallbackKind)
        if uerr := deps.epicRepo.UpdateStatusTx(ctx, tx, epic.ID, target, nil, &notes); uerr != nil {
            slog.Warn("cascade: epic update failed",
                "trigger_key", trigger.triggerKey, "epic_key", epic.Key, "error", uerr)
            return
        }
        if herr := deps.historyTx.CreateTx(ctx, tx, &models.EntityHistory{
            EntityType: models.EntityTypeEpic,
            EntityID:   epic.ID,
            FromStatus: string(epic.Status),
            ToStatus:   target,
            ChangedBy:  ptrString("system"),
            Notes:      &notes,
        }); herr != nil {
            slog.Warn("cascade: epic history write failed",
                "trigger_key", trigger.triggerKey, "error", herr)
            return
        }
    }
    // Else: idempotent skip.

    if cerr := tx.Commit(); cerr != nil {
        slog.Warn("cascade: commit failed",
            "trigger_key", trigger.triggerKey, "error", cerr)
        return
    }
}
```

`buildAutoReopenNotes(trigger, fallbackKind)` produces the structured prefix string per REQ-F-007.

#### 2.5.5 Resolver body

```go
func resolveReopenTarget(
    ctx context.Context,
    historyQuerier ParentReopenHistoryQuerier,
    entityType models.EntityType,
    entityID int64,
    levelWf *workflow.Service,
) (string, string, error) {
    // Step 1: enumerate terminal statuses for this level from the workflow.
    terminalSet := levelWf.TerminalStatuses() // helper: returns []string of all terminal status names
    // (If TerminalStatuses() does not exist on workflow.Service, it can be implemented in
    // ~6 lines by enumerating all statuses and filtering by IsTerminalStatus. This is the
    // ONE small addition to workflow.Service permitted by this spec, and only if needed.)

    target, found, err := historyQuerier.GetLastNonTerminalStatus(ctx, entityType, entityID, terminalSet)
    if err != nil {
        return "", "", err
    }
    if found {
        return target, "", nil
    }

    // Step 2: aggregation status fallback.
    aggStatuses := levelWf.GetAggregationStatuses()
    if len(aggStatuses) > 0 {
        return aggStatuses[0], "aggregation", nil
    }

    // Step 3: initial status fallback.
    return levelWf.GetInitialStatusString(), "initial", nil
}
```

#### 2.5.6 CLI presentation update

In whichever file currently formats `shark status history` rows, detect rows whose `notes` begin with the literal `auto_reopen:` prefix and render them with a distinct visual treatment. Suggested approach: prepend an `[auto]` label and use the existing color scheme's "info" or "system" color. JSON output is unchanged — consumers filter by the prefix themselves.

### 2.6 Migration and Rollout

**No data migration.** Behavioral change activates on transitions occurring after this feature ships. No retroactive cleanup.

**No schema migration** unless the index check (REQ-N-005) discovers a missing index, in which case a single index addition follows the standard migration protocol with a `CurrentSchemaVersion` bump from 11 to 12.

**Rollout sequence (mirrors `architecture.md` Section 5):**

1. Land repository additions (`GetLastNonTerminalStatus`, `UpdateStatusTx` on feature/epic, `CreateTx` on entity history). Pure additions; no behavior change.
2. Land `cascade_reopen.go` and unit tests with mocks. No production wiring yet.
3. Update `TaskService` and `FeatureService` constructors with the new optional dependencies (nil-safe).
4. Wire CLI (`internal/cli/services_global.go`) and HTTP (`cmd/server/services.go`) to inject the new dependencies. Cascade now fires in production paths.
5. Refactor `maybeReopenParentFeature` and `maybeReopenParentEpic` to delegate to the unified helper. Existing tests stay green via the legacy-fallback path until they are updated to wire cascade deps.
6. Update `shark status history` formatter to render auto-reopen rows distinctly.
7. Add integration tests covering the basic + advanced profile parameterization (AC-12) and the bug/change-card negative test (AC-11).

Each phase is independently shippable. Rollback at any phase is `git revert`.

### 2.7 Performance Validation

Per REQ-N-001, cascade overhead must be ≤50ms P95. The architecture's worst-case cost analysis (architecture.md Section 6) is ~17ms with significant headroom. AC-15 introduces a benchmark (`BenchmarkCascade_BothLegs`) that measures the worst-case path over 1000 runs and asserts the P95 budget.

### 2.8 Risks and Mitigations

The full risk register is in `architecture.md` Section 7. The implementation-relevant risks are:

| Risk | Mitigation |
|------|------------|
| `entity_history` index missing for the lookup query | REQ-N-005 — verify during implementation; if missing, add via standard migration with `CurrentSchemaVersion` bump from 11 to 12. |
| Concurrent regressions race on parent status check | ADR-004 — re-fetch inside the cascade transaction. SQLite WAL mode serializes writers. |
| Refactoring `maybeReopen*` breaks existing tests | The legacy-fallback path preserves existing test behavior until tests are migrated to wire cascade deps. AC-10 explicitly requires existing tests to stay green. |
| Bug or change-card cascade leakage | Structural exclusion — no hook in those services. AC-11 verifies. |
| Cascade failure leaves stale parents | REQ-N-003 — non-blocking with `slog.Warn`. Re-trigger is idempotent and self-heals. Documented in inline comments on `cascadeParentReopens`. |

---

## Part 3 — Exit Gate Checklist

This spec is complete when every box below is checked:

- [x] Every requirement (REQ-F-001..010, REQ-N-001..007) is testable and has at least one verifying test in the AC table.
- [x] Every architecture decision references an existing pattern (ADR or `architecture.md` section) or explains the deviation.
- [x] File paths are listed for all changes (Section 2.1).
- [x] Function signatures are listed for all new repository and service methods (Section 2.3).
- [x] Trigger sites are identified by file and line number (Section 2.5).
- [x] No TBDs in critical sections.
- [x] Out-of-scope items are explicit at both requirement-level (Section 1.5) and architecture-level (Section 2.1.3).
- [x] Migration strategy is explicit (Section 2.6) and consistent with `.claude/rules/database-critical.md`.
- [x] Performance budget is named and has a verifying benchmark (REQ-N-001, AC-15).
- [x] Backward compatibility is preserved (REQ-N-004); no CLI/API/exit-code changes.

---

## References

- `feature.md` — Feature description, problem statement, planned tasks, acceptance criteria
- `prd-source.md` — Original epic-level PRD (E26, cancelled and rescoped) — Sections 1, 5, 6 for business context and UAT scenarios
- `architecture.md` — Original epic-level architecture (E26) — Section 2 for full ADRs, Section 6 for performance, Section 7 for risks
- `research.md` — Brownfield analysis with file paths, line numbers, and integration points for the existing helpers being unified
- `uat-plan.md` — 8 UAT scenarios with mappings to acceptance criteria
- `.claude/rules/architecture.md` — Service-layer / repository-layer separation rules
- `.claude/rules/services/service-design.md` — Constructor injection, narrow interface, transaction ownership patterns
- `.claude/rules/database-critical.md` — Migration protocol if index addition is needed
- `internal/services/task_service.go` lines 320, 382, 614–622, 1046–1092 — Existing reopen helper and `TransitionStatus` post-hook block
- `internal/services/feature_service.go` lines 209–237, 776, 780–814 — Existing reopen helper and `TransitionStatus` post-hook block
- `internal/repository/entityhistory/repository.go` — Where `GetLastNonTerminalStatus` and `CreateTx` are added
- `internal/db/db.go` — `CurrentSchemaVersion = 11` (line 438), unchanged unless index addition is required
