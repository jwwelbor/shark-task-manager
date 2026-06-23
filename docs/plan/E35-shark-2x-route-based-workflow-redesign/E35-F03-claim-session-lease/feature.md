---
feature_key: E35-F03-claim-session-lease
epic_key: E35
title: Claim / session lease
description: Status becomes a pure phase; the claim is the lease. An agent claims an entity (session = who + when); shark next hands out only unclaimed entities. Claim fields (claimed_by/session/claimed_at or reuse sessions table) + schema version bump. Crash recovery: sync-release on every exit, detached heartbeats renew lease + carry progress, TTL = K missed updates as universal backstop. Updates do triple duty (lease + progress + telemetry) routed through the E32-F07 events.jsonl/span pipeline. Decision D3, design section 6.
size: L
---

# Claim / session lease

**Feature Key**: E35-F03

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Design (single source of truth)**: [route-based-workflow-redesign.md](../../route-based-workflow-redesign.md) — D3, §6, §8

---

## Goal

### Problem

The `in_X` markers exist only so a dispatch loop can tell "waiting" from "being
worked." That conflates *phase* with *who-holds-it-now*, doubling every workflow
stage. Removing `in_X` (F02 collapses the routing) leaves a gap: with status as a
pure phase, two agents could both grab the same `ready`-phase entity, and a crashed
agent would wedge an entity forever.

### Solution

Make **status a pure phase and the claim the lease** (D3, design §6). An agent
*claims* an entity — a session of *who* + *when* — orthogonal to status.
`shark next` hands out only **unclaimed** entities, so `in_X` is no longer needed
to hide in-flight work.

One lease primitive serves both dispatch modes:

- **Sync dispatch** (loop spawns agent → blocks → releases): the orchestrator
  releases on *every* exit — success → `outcome`, crash/timeout →
  `release(blocked, reason="agent died")`. The TTL almost never fires.
- **Parallel / detached dispatch**: the agent (or its runner) emits periodic
  **updates** that renew the lease *and* carry progress. Miss too many →
  `shark next` reclaims and requeues.
- **TTL is the universal backstop**, expressed as **K missed updates** (one global
  cadence, no per-step timeout guessing) — nothing wedges permanently even if the
  orchestrator itself dies.

**Updates do triple duty** — lease renewal + progress reporting + telemetry. They
route through the same `events.jsonl`/span pipeline that **E32-F07** adds; no
separate heartbeat channel.

### Impact

- Concurrency-safe dispatch without `in_X` statuses.
- Crash/timeout recovery is automatic via the TTL backstop.
- "Is it alive / how far along" and F07's "measure efficiency" become one data stream.

---

## Scope

- Persistence: claim fields (`claimed_by` / `session` / `claimed_at`) — or reuse
  the existing sessions table if it fits — plus a `CurrentSchemaVersion` bump and
  migration function (see database-critical rules; bump is the key step).
- `shark next`: filter to **unclaimed** entities; claim atomically on hand-out.
- Release path (from F02): clear the claim on release; on abnormal exit, release as
  `blocked` with a reason.
- Lease updates: renew `claimed_at` + carry progress; reclaim when K updates are
  missed.
- Fold lease updates into the E32-F07 `events.jsonl`/span pipeline (no new channel).
- Config: one global lease cadence / K-missed-updates value (config-driven, not
  per-step).

### Out of Scope

- The `outcomes:` resolution and `release(outcome)` mechanics — F02 (this feature
  consumes them).
- Parking-resume target computation — F05.
- Harness loop rewrite that *calls* claim/release — F06.

---

## Acceptance Criteria

1. `shark next` returns only unclaimed entities and claims the one it hands out
   atomically (no double-grab under concurrent callers).
2. A normal release clears the claim; the entity is reclaimable at its new phase.
3. An abnormal exit (no release) is reclaimed after K missed updates and re-dispatched
   as an unclaimed phase row.
4. Lease updates renew the claim and emit progress through the E32-F07
   `events.jsonl`/span pipeline — no separate heartbeat path exists.
5. `CurrentSchemaVersion` is bumped and the claim-field migration runs on existing
   databases (verified against the skip_migrations path).
6. The K-missed-updates backstop is a single global config value, not per-step.

---

## Verification

- Repository tests (real DB, with cleanup) cover claim/release/reclaim and the
  unclaimed-only filter.
- Concurrency test: two `shark next` callers cannot claim the same entity.
- Migration test: a pre-migration DB gains the claim fields after a version bump.
- `make fmt && make lint && make test` pass.

---

## Dependencies

- **Blocked by**: F02 (release is the other half of claim/release).
- **Blocks**: F05 (migration coordinates the schema bump), F06 (harness calls claim/release).
- **Coordinates with**: E32-F07 (telemetry pipeline) — reuse, do not duplicate.

---

*Last Updated*: 2026-06-23
