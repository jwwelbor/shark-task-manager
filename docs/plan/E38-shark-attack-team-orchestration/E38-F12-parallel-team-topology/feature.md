---
feature_key: E38-F12-parallel-team-topology
epic_key: E38
title: Parallel Team Topology
description: Skill-layer follow-on to E38-F09 implementing the parallel team topology recorded in the parallel-team-integration proposal: /run-agent-team becomes a topology adapter under the two-axis model — shark plan (or the active sprint backlog) selects candidates, each teammate runs the ordinary keyed Rider loop as parent of its assigned entity, the coordinator owns integration and the council interface, and the shark-attack council wraps the run (questions via E39, escalations, sprint planning and retro ceremonies). Phase 1: ownership topology + council wrap + sprint contract, run-agent-team revised to the keyed-Rider teammate contract, run-sprint-team superseded as a thin alias. Phase 2: isolation topology with the thin integrator role (merge-in, post-merge gate, reviewed worktree closeout). Prompt/skill-layer only: no runtime, scheduler, claim store, or schema changes. Spec stays thin until F09 merges. Depends on E38-F09 skill restructure; consumes E19-F09 (shark plan sprint) when it ships, with client-side backlog enumeration as the interim.
---

# Parallel Team Topology

**Feature Key**: E38-F12-parallel-team-topology

> **Thin by design.** Per the governing proposal (§8) and the F05/F08
> trunk-integrity caveat, this feature is deliberately not specified in detail
> until E38-F09 merges — F09's skill restructure (REQ-F-015) renames the files
> this feature writes to. The proposal below is the single source of truth for
> the design and all recorded decisions; do not restate or fork them here.

---

## Governing design

**[Parallel Team Integration Proposal](../proposal-parallel-team-integration.md)**
(v4, decision-complete, 2026-08-02) — architect-reviewed; recorded decisions
in its §9 cover lease policy (30-min scoped TTL via `SHARK_CLAIM_TTL_SECONDS`,
config key stays unset), event-bounded persistent question holds, the
integrator role replacing a standing merge-referee, and the sprint contract
(council ceremonies + coordinator selection boundary).

## Goal (summary)

Make `/run-agent-team` a topology adapter under E38's two-axis model:
`shark plan` (or the active sprint backlog) selects candidates; each teammate
runs the ordinary keyed Rider loop as the *parent* of its assigned entity
(fixing the current template's advance-with-no-claim violation); the
**coordinator** owns integration and the council interface and never claims
delivery entities; the shark-attack council wraps the run — E39 Questions,
escalations, sprint planning and retro ceremonies.

## Deliverables (from proposal §8)

**Phase 1 — ownership topology + council wrap + sprint contract**
1. `skills/shark-attack/workflows/parallel-team.md` (coordinator procedure:
   loop, council wrap, sprint mode, closing report; shared-worktree
   serialization rules). Embedded mirror synced — parity gate applies.
2. Host `/run-agent-team` command revised to a thin pointer; teammate task
   body becomes the keyed Rider loop; hand-rolled DAG steps replaced by
   `shark plan`.
3. Sprint planning/retro ceremony additions to the council workflows;
   `/run-sprint-team` rewritten as a thin alias into sprint mode.
4. Resolve V-001 (claim filtering in keyed plan) during spec.

**Phase 2 — isolation topology**
5. The integrator role: worktree-per-teammate creation, serial merge-in +
   post-merge `make fmt && make lint && make test`, fix-forward protocol with
   durable traces, reviewed worktree closeout.

## Dependencies

- **E38-F09** (blocking): skill restructure must land first; spec thickens
  only after F09 merges.
- **E19-F09** (consumed, non-blocking): `shark plan sprint` replaces the
  coordinator's client-side sprint-backlog enumeration when it ships; the
  enumeration is the documented interim.

## Out of scope

No runtime, scheduler, dispatcher, or second claim store; no synthetic edges
written back to shark; no provider adapters beyond F09's Codex/Claude
references; no schema changes; no concurrent active sprints. See proposal
§8 "Not built" and §5 "Not pursued".

*Last Updated*: 2026-08-02
