---
feature_key: E38-F06-role-aware-pull-and-claim-guidance
epic_key: E38
title: Role-aware Pull and Claim Guidance
description: Use existing Shark role/agent filtering and claim semantics to let workers pull eligible work for their configured workflow role, with only the smallest CLI corrections required.
---

# Role-aware Pull and Claim Guidance

This feature makes role-aware self-pull the normal assignment model for the
team. A worker asks Shark for the next eligible item for its workflow-defined
role, claims that item through the existing lease, and works only within the
claimed scope. The feature preserves the existing primary-designation and
agent/role selector corrections; it does not revive legacy database assignment
or create a team scheduler.

Dependencies: E38-F04; execution order: 2; size: 2 (S).

## Requirements

- Use the existing Shark role/agent filtering and named selector behavior.
- Filter before claiming so a worker cannot claim work outside its role.
- Preserve atomic claim conflict behavior, lease ownership, heartbeat, and
  release semantics.
- Treat no eligible work, an existing claim, and a workflow gate as explicit
  outcomes for the Rider procedure.
- Make only the smallest CLI correction needed if a role filter is missing or
  inconsistent; do not add a team-run data model or scheduler.

## Acceptance criteria

- A developer, QA, architect, or other configured role receives only eligible
  work for that role.
- Two workers racing for the same eligible item still produce one successful
  claim and one conflict; no force-steal is introduced.
- Existing `shark next`, `shark sprint next`, `shark claim`, and release behavior
  remains compatible for callers that do not specify a role.
- The role-filtering behavior is covered by focused CLI/service tests and is
  documented in the `shark-attack` procedure.

## Out of scope

- A new scheduler, team ledger, aggregate router, or resume command.
- Reintroducing the legacy task `agent` assignment as a planning authority.
- Worker-owned workflow transitions for the dispatched parent entity.
