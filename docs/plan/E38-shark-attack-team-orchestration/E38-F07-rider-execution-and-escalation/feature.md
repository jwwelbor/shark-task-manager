---
feature_key: E38-F07-rider-execution-and-escalation-loop
epic_key: E38
title: Rider Execution and Escalation Loop
description: Define the host-side team procedure that runs shark next, claims work, dispatches prompts, records outcomes, advances state, and routes questions through the role hierarchy and escalation path.
---

# Rider Execution and Escalation Loop

This feature defines the reusable host-side team procedure. The chair or
Scrum Master drives the ordinary Shark loop: retrieve `shark next <key> --json`,
claim the returned entity, dispatch the rendered prompt to the appropriate
worker, record the worker result, advance the entity through the configured
outcome, and release the claim. The team skill adds role coordination,
questions, handoffs, and escalation around that loop; it does not replace the
loop with a new runtime.

Dependencies: E38-F04 and E38-F06; execution order: 3; size: 3 (M).

## Requirements

- Pass the `shark next` prompt to the worker unchanged.
- Keep claim, heartbeat, release, and workflow-transition ownership in the
  Rider/parent procedure; workers perform craft only.
- Support both assigned work and role-aware self-pull without requiring a new
  Shark command.
- Define a clear escalation path: worker question → role owner → chair →
  product or human review when the decision changes scope, architecture, or a
  quality gate.
- Record decisions, handoffs, blockers, and unresolved questions in the
  project council area so a refreshed worker can resume context.
- Stop cleanly on pause, archive, error, or an explicit human gate; do not
  invent aggregate statuses or mark partial work successful.

## Acceptance criteria

- A team run can be followed as a repeatable prompt/procedure using only
  existing Shark commands and host-agent dispatch.
- A worker never advances or releases the entity dispatched to it.
- Every escalation names the question, responsible role, evidence, decision
  needed, and next owner.
- Interruption leaves ordinary Shark claims and statuses recoverable through
  the existing workflow; no second resume state is introduced.
- The procedure has a concise human-readable handoff format and a role-aware
  pull example.

## Out of scope

- `internal/team`, `team_runs`, aggregate routing, resume reconciliation, or
  new `team` CLI commands.
- A provider runtime, autonomous agent-team engine, or cross-project worker
  coordinator.

## Research recovery note

The prior task-generation blocker named the wrong required artifact: it called
for a separate prior-art report. The unified research report now provides the
required Capability map. The historical blocker remains in Shark for audit
purposes. The recorded specification and test-plan notes remain authoritative,
but their files are not present in this checkout and must be restored before
task generation can resume.
