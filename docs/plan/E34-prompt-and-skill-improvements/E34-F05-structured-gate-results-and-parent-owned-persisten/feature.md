---
feature_key: E34-F05-structured-gate-results-and-parent-owned-persisten
epic_key: E34
title: Structured Gate Results and Parent-Owned Persistence
description: Define a versioned structured result contract for review, QA, and UAT gates; make Shark Rider and the core runner validate and persist bounded evidence, findings, remediation sweeps, and kickbacks before lifecycle transitions.
---

# Structured Gate Results and Parent-Owned Persistence

**Feature Key**: E34-F05

## Goal

### Problem

Shark gate prompts ask workers to record findings and kick back tasks, but the
Rider contract reserves lifecycle mutations for the parent. The current Rider
loop recognizes a few free-form directive lines, while the core runner reads
only `RECOMMENDED OUTCOME`. This split caused E04 to accumulate dozens of
review findings without one durable `review-finding` note, which made
recurrence analysis and reliable replay impossible.

### Solution

Define one versioned `GateResult` payload for code review, QA, UAT, and other
configured quality gates inside the existing canonical `kind: final`
worker-control envelope. Both `/shark-rider run` and `shark run` validate the
same shape, persist its bounded evidence and directives idempotently under the
durable worker run and parent lease, and only then advance by its opaque
workflow outcome. Preserve a compatibility path for non-gate workflow steps
and existing directive output without adding a second result envelope.

### Impact

Every structured gate result becomes durable before its transition. A failed
gate cannot silently omit findings or kickbacks, malformed output cannot
advance work, and replaying an already-persisted result cannot duplicate notes
or repeat a kickback.

## Research findings

- `skills/shark-rider/verbs/run.md` makes the parent responsible for claims,
  notes, kickbacks, transitions, and release, but parses only outcome,
  complexity note, parent note, ordinary note, and text kickback lines.
- `internal/runner/controller.go` selects a transition from one whole-line
  `RECOMMENDED OUTCOME` marker. It has no generic gate-result persistence
  boundary. Its Question handoff is the closest local pattern for a bounded,
  validated, parent-owned worker result.
- Existing gate prompts define useful finding metadata, and Shark notes already
  accept typed metadata. The missing capability is a trustworthy handoff and
  persistence service, not a new finding table.
- The WWGM E04 inventory found zero `review-finding` notes despite 45 recorded
  review findings. The worker/parent ownership contradiction is therefore an
  observed production-path failure, not a hypothetical formatting concern.

## Public contract

The canonical shape is `GateResult` schema version 1, carried as the optional
`gate_result` member of the existing single worker-control `kind: final`
envelope. Its normative field definitions live in
[Architecture: I-02 GateResult v1](../architecture.md#i-02-gateresult-v1).
The outer `recommended_outcome` and common `evidence` remain authoritative.
The nested object contains only gate-specific summary, findings, kickbacks,
sweeps, impacts, and completeness explanation; a second gate, outcome, or
evidence field is invalid.

The outer `recommended_outcome` value remains workflow-defined. The parser
validates it against the current step's configured outcomes and uses that
outcome's configured
`success|route_rework|kickback_rework|blocked|hold|cancelled` semantic role
only for completeness checks; it never maps the opaque key to a hardcoded
status.
Evidence contains compact command results and artifact pointers, not full logs,
prompts, transcripts, credentials, or unrestricted tool output.

## Requirements

1. **REQ-F-001 — Shared GateResult model and parser**
   - Define one Go model, JSON decoder, validation contract, and error taxonomy
     used by the core runner and mirrored exactly by Rider guidance.
   - Require a non-zero schema version and concise summary, and validate the
     outer configured outcome and evidence with bounded collections and unique
     finding and kickback identities.
   - Reject duplicate envelopes, unknown versions, unknown outcomes, invalid
     top-level JSON shapes, secret-bearing text, and oversized content.

2. **REQ-F-002 — Parent-owned persistence**
   - Persist a gate-summary/evidence `review` note, `review-finding` notes,
     remediation-sweep `reference` notes, I-04 `reference` notes, and task
     kickbacks before transition. Sweep and impact notes use existing note
     types with bounded `record_kind` metadata; no new note enum or database
     migration is implied. An I-04 note identifies its source key and operation
     digest.
   - Store finding metadata for gate, severity, `class_key`, class statement,
     fingerprint, affected acceptance/test identifiers, disposition, and the
     parent session identity.
   - Keep persistence within the entity and note services; do not let a gate
     worker claim, advance, release, or force-set workflow state.

3. **REQ-F-003 — Replay safety**
   - Bind persistence to a stable `run_id`, entity, source status, and operation
     digest. Associate each renewable parent session without making it the
     replay identity.
   - Let a restarted parent with a newly claimed authorized session resume an
     exact partial result without duplicate notes, kickbacks, or transitions.
   - Reject a non-identical replay under the same persistence identity.
   - Make already-applied kickbacks safe to retry and fail closed on a
     conflicting target status or reason.
   - Distinguish `persistence_complete` from `transition_applied`; resume the
     guarded transition or lease release after either crash window.
   - Under a per-run lock, create `result.json` once and atomically replace only
     `operation-state.json` under the existing `.shark/runs/<run-id>/`
     directory. Identical result bytes are idempotent; a different result never
     overwrites the accepted record.
   - Give every target write a deterministic suboperation ID stored in note
     metadata or bounded entity-history reason metadata. On resume, reconcile
     the sidecar from these durable target records before applying missing
     operations, closing the target-commit/sidecar-update crash window.
   - Add `shark run <entity-key> --resume-run=<run_id>
     --session=<authorized-session-id>`. It acquires the run lock and validates
     the recorded entity, source status, route, and digest before resuming.

4. **REQ-F-004 — Gate completeness**
   - Every structured step maps each opaque outcome key to exactly one semantic
     role. Every kickback must target a different entity from the bound main
     entity. `success` and `route_rework` contain no kickbacks;
     `route_rework` uses the configured main-entity transition, while
     `kickback_rework` requires at least one child/cross-entity kickback.
     `blocked`, `hold`, or `cancelled` requires
     `no_kickback_reason` when no kickback exists. Findings may accompany these
     cases but do not replace the routing requirement.
   - A `success` role cannot contain an open blocking finding; keys such as
     `deep_verify` may be role `success` without being renamed.
   - A finding disposition must be one of the schema-defined values and must
     point to a durable decision when it is already dispositioned.

5. **REQ-F-005 — Rider and core-runner parity**
   - Update Rider's run procedure, task execution pattern, host adapter result
     contract, and relevant prompts to emit and consume the same envelope.
   - Extend `shark run` to validate and persist the envelope before calling
     `TransitionStatus`.
   - Add parity tests that run identical fixtures through both documented
     paths and compare accepted data, rejection classes, and persistence
     order.

6. **REQ-F-006 — Compatibility**
   - Route steps select `result_contract: legacy|gate_result_v1`; omission
     defaults to `legacy`, unknown values fail validation, and `shark next`
     exposes the resolved value to both execution paths.
   - Structured steps declare an exact `outcome_roles` map covering every
     configured outcome. Missing, extra, or unknown roles fail validation, and
     `shark next` exposes the map to both paths.
   - Non-gate steps may continue using `legacy` while migrations proceed.
   - A gate explicitly configured for structured results must fail closed when
     the envelope is absent or malformed; it must not fall back silently.
   - Existing note storage remains readable; no database migration is planned.

7. **REQ-NF-001 — Bounded and safe data**
   - Reuse the Question model's bounded-text and forbidden-marker approach.
   - Report validation errors by field and class without echoing rejected
     secrets or entire worker output.
   - Use owner-only run directories/files, reject symlink or non-regular
     targets, and fsync same-directory atomic writes; result creation is
     no-replace under the per-run lock.

## Implementation plan

1. Extend the canonical final worker-control envelope with an optional
   GateResult payload; add the model, validation helpers, and unit tests without
   introducing a second marker parser.
2. Add a locked, immutable-create-once `.shark/runs/<run-id>/result.json` and
   atomically replaced `operation-state.json` for the terminal GateResult
   envelope and replay journal. Add a persistence coordinator behind injected
   interfaces for notes, task status changes, operation-state lookup, and
   worker-retirement evidence.
3. Integrate validation and persistence into the core runner between dispatch
   success and transition.
4. Add authenticated `shark run <entity-key> --apply-result=<result-file>
   --run-id=<run_id> --session=<session-id>` for Rider's initial ingestion and
   `--resume-run=<run_id>` for recovery. Replace Rider's independent directive
   grammar with this shared service and document the compatibility boundary.
5. Add and validate the route-step `result_contract` field, update gate prompt
   partials once, then render all consumers and add parity, malformed-input,
   persistence-order, restart, crash-window, and replay tests.

## Acceptance scenarios

**Persist a rejected gate before routing rework**

- Given a review worker returns a valid v1 result with two findings and one
  task kickback,
- When either Rider or the core runner consumes it,
- Then both findings and bounded evidence are durable before the task kickback
  and feature transition,
- And the transition uses the returned configured outcome.

**Reject incomplete or hostile output**

- Given a structured gate returns malformed JSON, an unknown outcome, a second
  envelope, oversized evidence, or forbidden credential material,
- When the parent validates the result,
- Then no note, kickback, or lifecycle transition is written,
- And the run surfaces a bounded validation error.

**Replay a committed result**

- Given persistence succeeded but the parent crashed before transition, or the
  transition succeeded but the parent crashed before lease release,
- When a restarted parent claims an authorized replacement session and resumes
  the same `run_id` and exact envelope,
- Then the parent completes the transition and release exactly once without
  duplicate notes or kickbacks.

## Dependencies and interactions

- Produces **I-02 GateResult v1** for E34-F06, E34-F07, and E34-F08.
- E34-F06 and E34-F07 depend on this feature in Shark.
- This feature reuses the canonical worker-control envelope, run liveness
  directory, existing notes, task workflow services, lease identity, and the
  Question handoff validation pattern. Result and operation-state records
  themselves are new.

## Out of scope

- A new review-finding table, analytics warehouse, or transcript store.
- A universal output format for non-gate craft workers.
- Retry-count escalation or a new council implementation.
- Any WWGM-specific test command, coding standard, or override cleanup.

## Verification plan

- Unit-test schema validation, top-level shape checks, rejection of nested
  gate/outcome/evidence aliases, secret markers, and every
  text/collection/total bound at limit-1, limit, and limit+1.
- Service-test persistence ordering, partial failure, parent restart with a
  replacement session, exact replay, conflicting replay, and idempotent
  kickbacks.
- Failure-inject after every durable note/history commit and before sidecar
  replacement; reconcile by suboperation ID and prove no duplicate note,
  history entry, or kickback. Race two initial results and require immutable
  first-writer-wins conflict handling.
- Failure-inject after `persistence_complete`, after transition, and before
  release; assert transition and release occur exactly once and only after
  terminal worker-retirement evidence.
- Table-test every semantic role: `success` with and without open or
  severity-conflict blockers and with a forbidden child kickback;
  `route_rework` with no kickback and rejection of any kickback;
  `kickback_rework` with and without a child kickback;
  and `blocked`, `hold`, and `cancelled` with kickbacks or with present/missing
  `no_kickback_reason`. Under every role reject a kickback targeting the bound
  main entity. Include opaque `deep_verify` as `success` and reject obsolete
  semantic role `rework`.
- Workflow-table every named adoption-matrix entry and representative excluded
  steps. Test omitted selector → `legacy`, unknown selector/role, incomplete or
  extra role maps, missing structured payload, Question pause, ADR impact
  ingestion, and whole-file overrides. Assert `shark next --json` exposes the
  same resolved `result_contract` and `outcome_roles` to both callers.
- Filesystem-test absolute/escaping run IDs and paths, symlinked intermediate
  and leaf components, directory/FIFO/socket targets, oversized reads,
  `0700`/`0600` modes, permissive or non-regular existing targets, exclusive
  temporary collisions, fsync/rename cleanup, and crash after create-once
  `result.json` but before operation-state initialization.
- Run shared valid, malformed, conflicting-replay, and partial-resume fixtures
  through both the core ingestion call and Rider's apply-result/resume CLI.
  Compare normalized results, bounded error classes, suboperation order, target
  writes, and selected transition; rendered prompts alone are not parity
  evidence.
- Run `make fmt`, `make lint`, `make test`, and `git diff --check`.

*Last Updated*: 2026-08-05
