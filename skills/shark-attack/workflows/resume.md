# Resume: live-consultation follow-up, bounded replacement, and worker refresh

## Goal

Restore a worker's ability to keep going after either of two events: (1) a
live consultation's answer becomes available while the dispatched worker
may or may not still be running, and (2) any other worker refresh — a fresh
process starting cold after a crash, a lease loss, or the bounded
replacement this file's own procedure below may start. Do this without
depending on prior chat history and without storing sensitive transcripts.

## Prerequisites

- Know the project root, the member ID, and — for a live-consultation
  resume — the entity key of the dispatched worker the consultation is
  blocking.
- Use the council layout under `docs/council/`.
- Know which capabilities the active host declares. `providers/codex.md`
  and `providers/claude-code.md` each record follow-up, interrupt, and
  isolation support for one host, backed by captured evidence. This file
  never invents a capability claim neither provider reference backs.

## Capability discovery

Perform capability discovery **before** selecting an execution topology,
coordination level, or resume path (REQ-F-012). A missing follow-up,
interrupt, or isolation capability is data that drives one of this file's
documented fallbacks below — never license to invent an unverified
provider command. Read the active host's entry in `providers/codex.md` or
`providers/claude-code.md` first; `context/operating-model.md` owns
topology and coordination-level selection and is consulted only after this
step.

## Resume path: same-worker follow-up or bounded replacement

Once a live consultation's answer is available, decide how to deliver it
using the follow-up capability discovered above — never by assumption
(REQ-F-008):

1. **Follow-up supported.** Deliver the answer to the *same* worker
   identity as a native follow-up. This path creates zero new workers and
   needs no context reload — the original worker keeps its own context.
2. **Follow-up unsupported.** Build one bounded immutable handoff and start
   exactly one replacement worker from it. Never start more than one
   replacement, and never dispatch the original worker's identity a second
   time.

### Handoff content

The handoff built in step 2 carries only:

- the dispatched entity's key
- the consultation question
- the recorded answer
- evidence pointers

It excludes rendered prompt content, credentials, access tokens, and
transcripts. The replacement worker loads it through the restore-durable-
context procedure below — the handoff supplies context, never authority
(REQ-F-010): the replacement still needs its own fresh keyed dispatch and
claim before it can act.

## Interrupt and stale consultations

A responder that goes silent triggers a bounded escalation ladder: the
parent pings once, then either interrupts and replaces or waits, then — if
still no answer — stops at a hard deadline. The parent sends this single
ping to the current responder once the consultation's wait window has
elapsed, before doing anything else in this section; a ping is a nudge, not
a fresh dispatch — it never mints a second `Q###`, spawns a worker, or
restarts the consultation's deadline, and the parent sends at most one ping
per responder. A responder that answers before the wait window elapses is
never pinged at all — see the fast path in the responder-outcome ladder
below.

Whether the parent then cancels the stale consultation before routing a
replacement responder, or simply waits to the deadline, depends on the
interrupt capability discovered above (AC-023):

1. **Interrupt supported.**
   1. Cancel the stale consultation before routing the replacement
      responder.
   2. Route exactly one replacement responder.

   Cancelling changes no Shark state: no claim, status, or history record
   is written by the cancel itself.
2. **Interrupt unsupported.** Skip the cancel step and let the stale
   consultation expire at its deadline — a deadline-only fallback. Never
   invoke a provider operation either `providers/codex.md` or
   `providers/claude-code.md` lists under "Unsupported Operations" to
   simulate a cancel; both providers list Interrupt there today.

### Deadline stop

If no answer is recorded before the consultation's deadline — whether
because interrupt was unsupported and the wait simply expired, or because
the routed replacement responder also went silent — the parent performs all
four of the following as one ordered, non-optional sequence:

1. Stop the dispatched entity's write worker(s).
2. Record a bounded unresolved handoff (the same four-field shape `###
   Handoff content` above defines — never the rendered prompt).
3. Record a blocker against the dispatched entity.
4. Release the dispatched entity's lease.

Every one of these four steps runs; none is optional and none may be
skipped or reordered. A deadline that records the blocker but skips the
lease release — or the reverse — leaves the dispatched entity in an
unrecoverable half-state.

### Responder-outcome ladder

The escalation ladder above resolves differently depending on when — if
ever — the responder answers:

| # | Responder behavior | Expected result | Sub-case |
|---|---|---|---|
| 1 | Answers before any ping | No ping sent; answer recorded normally (fast path) | TC-010-17 |
| 2 | Silent, then answers after the one ping | No replacement routed; answer recorded from the original responder | TC-010-18 |
| 3 | Silent through the ping, replacement responder answers | Replacement's answer is recorded; the original responder's pending state is closed out, not left dangling | TC-010-19 |
| 4 | Silent through the ping and the replacement, deadline reached | Deadline stop above runs in full: write workers stopped, bounded unresolved handoff recorded, blocker recorded, lease released | TC-010-04 |
| 5 | Answers at exactly the deadline boundary | Counts as answered — the deadline stop above only runs once the deadline instant has fully passed with no answer recorded, so an answer landing at that instant is accepted | TC-010-20 |

Row 4 restates TC-010-01 through TC-010-04 as one ladder rung: exactly one
ping (TC-010-01), the interrupt-or-wait choice above (TC-010-02), at most
one replacement responder (TC-010-03), and the ordered deadline-stop
sequence (TC-010-04).

## Capability-driven resolution across independent flags

Isolation, follow-up, and interrupt are three independent capabilities.
Each one resolves on its own evidence — a host missing one capability never
forces the fallback for a different one:

| # | Isolation detected? | Follow-up detected? | Interrupt detected? | Expected resolved behavior | Sub-case |
|---|---|---|---|---|---|
| 1 | yes | yes | yes | Topology: Parallel-with-isolation eligible; Follow-up: same-worker (zero new workers); Interrupt: cancel-then-replace | TC-010-15 |
| 2 | no | yes | yes | Topology: Sequential (isolation undetected); Follow-up: same-worker (zero new workers); Interrupt: cancel-then-replace | TC-010-08 |
| 3 | yes | no | yes | Topology: Parallel-with-isolation eligible; Follow-up: bounded replacement (exactly one replacement worker); Interrupt: cancel-then-replace | TC-010-09 |
| 4 | yes | yes | no | Topology: Parallel-with-isolation eligible; Follow-up: same-worker (zero new workers); Interrupt: deadline-only (no cancel attempt) | TC-010-10 |
| 5 | no | no | no | Topology: Sequential (isolation undetected); Follow-up: bounded replacement (exactly one replacement worker); Interrupt: deadline-only (no cancel attempt) | TC-010-16 |

Row 1 is the all-supported baseline. Rows 2–4 each undetect exactly one
capability flag relative to row 1, and only that flag's own resolved
segment changes. Row 5 undetects all three at once — the case that would
catch an implementation that only checks one flag and assumes the others
follow from it.

## Restore durable context (worker refresh)

A replacement worker started from the handoff above, and any other worker
resuming after a refresh, restores its bounded context the same way:

1. Read relevant durable records in this order:

   - `docs/council/decisions/`
   - `docs/council/handoffs/`
   - `docs/council/escalations/`, including unresolved items
   - `docs/council/inbox/<member-id>/`

2. Keep only records that match the active root key and, when present,
   child key. Report malformed, stale, secret-bearing, or out-of-root
   references as actionable errors; do not load them into worker context.

3. Build the resume context from bounded paths and metadata: IDs, types,
   roles, root and child scope, statuses, evidence, timestamps, and next
   actions. Do not load rendered prompts, credentials, access tokens, or
   unrestricted worker output.

4. Act on each actionable inbox message. Before you acknowledge it,
   preserve the resulting decision, handoff, unresolved escalation, or
   resolution in the applicable durable directory.

5. Acknowledge or remove the message after its durable result exists.
   Repeating the acknowledgement is a no-op and does not remove the
   durable record.

## Result

A refreshed worker — whether it is the bounded replacement started above or
any other worker resuming after a refresh — has the same durable decision,
handoff, unresolved escalation, and inbox pointers needed to continue
safely. A follow-up-eligible consultation instead resolves with zero new
workers and no context reload. Use the existing workflow and claim
procedures for any work selection or lease operation; resuming never
restores authority on its own (REQ-F-010).
