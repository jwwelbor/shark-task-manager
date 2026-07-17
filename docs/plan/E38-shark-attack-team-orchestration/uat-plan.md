# E38 UAT Plan: Shark Attack Skill and Rider Protocol

**Epic:** E38 — Shark Attack Team Orchestration  
**Date:** 2026-07-13  
**Scope:** F04, F05, F06, and F07

This plan verifies a reusable role-based skill and host-side Rider procedure
over ordinary Shark CLI primitives. It deliberately does not require a team
ledger, scheduler, aggregate router, resume store, or new team command.

## Success-criteria coverage

| Criterion | Verification |
|---|---|
| Role correctness | UAT-01 |
| Claim safety | UAT-02 |
| Prompt fidelity and workflow ownership | UAT-03 |
| Council memory and handoffs | UAT-04 |
| Escalation clarity | UAT-05 |
| Repeatable Rider loop | UAT-06 |
| Lightweight operator handoff | UAT-07 |

## UAT scenarios

### UAT-01 — Pull work by workflow role

Given priority-ordered work exists for architecture, implementation, and QA,
a role worker requests the next eligible item. Verify that selection is
restricted to the workflow-defined role, uses the existing named selector and
agent filtering behavior, and does not revive the legacy database `agent`
assignment as authority.

Acceptance: each role receives only eligible work; no-role callers retain
existing behavior.

### UAT-02 — Claim race remains safe

Given two workers request the same eligible item, verify that the existing
claim operation produces one winner and one conflict, with no force-steal. The
winner can heartbeat and release its session-scoped claim.

Acceptance: at most one active claim exists for the item.

### UAT-03 — Ordinary dispatch ownership is preserved

Given a role worker is dispatched through the team procedure, verify that the
procedure obtains `shark next <key> --json`, passes `response.prompt` unchanged,
and owns claim, heartbeat, release, and status advancement. The worker performs
craft only and does not mutate the dispatched parent entity.

Acceptance: dispatch uses the ordinary Shark prompt and history contains zero
worker-owned parent transitions.

### UAT-04 — Council context survives worker refresh

Given `docs/council/` contains a decision, handoff, and actionable inbox item,
verify that a refreshed worker can find the relevant context, acknowledges the
inbox item after acting, and leaves durable decisions available.

Acceptance: continuation does not depend on the prior worker conversation.

### UAT-05 — Escalation has a clear route

Given a worker raises an unanswered question that may change scope,
architecture, or a quality gate, verify that the escalation records the
question, evidence, affected Shark key, responsible role, decision needed, and
next owner. Route it through the role hierarchy and pause for review when the
council cannot safely decide.

Acceptance: no unresolved question is silently guessed or routed to an
invented fixed human destination.

### UAT-06 — Rider loop handles normal stop conditions

Given an entity reaches pass, fail, blocked, pause, archive, or error, verify
that the Rider procedure records the worker result, advances only through a
configured Shark outcome, releases the claim, and stops at the appropriate
boundary.

Acceptance: partial work is not reported as successful and ordinary Shark
status/history remain authoritative.

### UAT-07 — Handoff is concise and actionable

Given a team pass ends with completed, blocked, escalated, and remaining work,
verify that the operator handoff identifies each key, current owner/status,
evidence or artifact, open question, next role, and next command. It must not
include prompts, credentials, or unrestricted transcripts.

Acceptance: a fresh operator can continue the work from the handoff alone.

## Out of scope checks

The following are explicit non-requirements for E38: `internal/team`,
`team_runs` tables, scheduler waves, aggregate outcome routing, resume
reconciliation, parallel-runtime benchmarks, and new `team` CLI commands.

## Acceptance gate

All seven scenarios pass against the configured workflow fixtures. The reviewer
also confirms that the role-filtering correction remains covered and that
ordinary `shark next`, `shark claim`, `shark heartbeat`, `shark release`, and
status-advance behavior is unchanged.
