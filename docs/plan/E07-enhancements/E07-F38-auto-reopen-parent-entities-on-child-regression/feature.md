---
feature_key: E07-F38-auto-reopen-parent-entities-on-child-regression
epic_key: E07
title: Auto-reopen parent entities on child regression
description: Automatically reopen a closed parent feature/epic when a child task transitions backward (e.g., UAT rejection sends task from in_qa back to in_development).
---

# Auto-reopen parent entities on child regression

**Feature Key**: E07-F38

> Originally scoped as epic E26 with 7 layered sub-features. Cancelled and rescoped as a single enhancement feature. Original planning artifacts preserved alongside this file (`prd-source.md`, `architecture.md`, `research.md`, `uat-plan.md`).

---

## Problem

When a task is sent backward in the workflow (e.g., QA fails it, UAT rejects it), its parent feature and epic remain in their closed/completed state. The dashboard shows the parent as "done" while child work is actively regressing. Developers must manually reopen the parent chain, and orchestrator agents can't tell that the parent needs new attention.

The codebase already has `maybeReopenParentFeature` (in `task_service.go`) and `maybeReopenParentEpic` (in `feature_service.go`), but they only fire on **`CreateTask`** / **`CreateFeature`**, not on backward status transitions.

## Solution

Add a post-hook to `TaskService.TransitionStatus` and `FeatureService.TransitionStatus` that detects backward transitions out of terminal status and reopens the parent chain in a single transaction. Reuse the existing `entity_history` table to determine the restore-target status (no schema migration).

## Scope

**In scope:**
- Backward-transition detection in `TaskService.TransitionStatus` and `FeatureService.TransitionStatus`
- Cascade helper that walks task → feature → epic and reopens any closed parent atomically
- History-based restore target lookup (last non-terminal status from `entity_history`, with sensible fallback)
- Audit row in `entity_history` with `notes` indicating auto-reopen
- `shark status history` output flag for auto-reopen rows
- Unify the existing creation-trigger reopens (`maybeReopenParentFeature`/`maybeReopenParentEpic`) onto the same helper
- Tests covering both basic and advanced workflow profiles

**Out of scope:**
- Bugs and change-cards (no parent-child cascade)
- Forward cascades (already handled by existing code)
- Retroactive backfill of historical regressions
- New CLI commands or HTTP endpoints
- Schema migrations

## Tasks (planned)

1. Add `GetLastNonTerminalStatus` and `CreateTx` to `EntityHistoryRepository`; add `UpdateStatusTx` to `FeatureRepository` and `EpicRepository`. Repository tests against real DB.
2. Create `cascade_reopen.go` helper in `internal/services/` with the unified parent-reopen cascade. Unit tests with mocks.
3. Wire the helper into `TaskService.TransitionStatus` and `FeatureService.TransitionStatus` (backward-transition trigger). Refactor existing `maybeReopenParentFeature`/`maybeReopenParentEpic` to call the same helper.
4. Update `shark status history` formatter to label auto-reopen rows distinctly.
5. End-to-end integration tests covering the UAT-rejection path under both basic and advanced workflow profiles. Verify bugs/change-cards are unaffected.

## Acceptance Criteria

- **AC1**: When a task in a closed feature transitions backward into a non-terminal status, the parent feature is automatically reopened in the same transaction.
- **AC2**: When a feature in a closed epic transitions backward, the parent epic is automatically reopened.
- **AC3**: The reopen target status is the most recent non-terminal status from `entity_history`, with a sensible fallback for entities lacking history.
- **AC4**: An `entity_history` row is written for each auto-reopen with a `notes` field that distinguishes it from manual transitions.
- **AC5**: `shark status history <key>` visually distinguishes auto-reopen rows.
- **AC6**: Existing creation-trigger reopens (`CreateTask`/`CreateFeature`) use the same cascade helper — no parallel implementations remain.
- **AC7**: Bugs and change-cards never trigger cascade reopens.
- **AC8**: Both basic and advanced workflow profiles work; the implementation does not hardcode status names.
- **AC9**: Cascade overhead is ≤50ms per transition (P95).
- **AC10**: All `make fmt && make lint && make test` checks pass.

## Reference Documents

- `prd-source.md` — Original epic-level PRD with full problem analysis and stakeholder impact
- `architecture.md` — System design with 6 ADRs and integration points
- `research.md` — Brownfield analysis identifying existing helpers and extension points
- `uat-plan.md` — 8 UAT scenarios with mappings to acceptance criteria
