# Route a live consultation through an E39 Question

## Goal

When a dispatched worker's control envelope carries `kind: question` (see
`context/worker-control-schema.yaml`), materialize that consultation as a
durable E39 Question and route it to its responder — without inventing a
second question, handoff, or resolution store (REQ-F-004). Every step below
is parent-executed; a dispatched worker never runs a Shark mutation command
(ADR-005).

This file covers the full loop: mint, configure, gate, route, respond, and
resolve — for a **routine** question only. When a question instead crosses
the material threshold `workflows/council.md` defines, route it through that
file's procedure instead of this one; this file does not restate that
threshold.

## Why the parent, never the worker

The envelope's `question` variant carries no `question_id` (D-005): the
parent mints `Q###` itself, so a worker-authored identity is never round-
tripped by accident. The parent copies `category`, `question`,
`why_blocking`, and `evidence` from the envelope into the durable record; it
adds no fields the envelope does not already carry.

## Mint

The parent — never the worker — creates the Question using Shark's generic
entity-`create` command for the `question` type, exactly as spec.md API §3
documents it: title positional, `--summary`, `--requester`, and `--blocking`
flags. Shark assigns `Q###`; the caller never supplies it.

## Configure

Configure the Question's resolution owner and its ordered responder list
with `question configure-workflow`. This is what moves the Question from
`draft` to `open` and is what makes the routing below possible.

## Gate — link `question_blocks`

Link the Question to the dispatched entity with `link Q### <entity-key>
--type=question_blocks`. **Order is load-bearing (D-006):** the qualification
check only recognizes a Question whose status is `open` or `answering`, so a
`question_blocks` edge created before configure-workflow runs is silently
inert — it exists in the relationship table but blocks nothing. Always
configure before you link.

Once the edge qualifies, the dispatched entity's keyed-next call returns a
paused response carrying a compact `question_block`: `question_key`,
`summary`, `resolution_owner`, and `current_responder` — nothing else. The
blocked worker sees only that; it is not the current responder and is not
the resolution owner, so a `question full` read is correctly denied to it.
The same gate rejects a status-advance attempt on the blocked entity while
the link qualifies; `status set` is documented elsewhere as the human
escape hatch that intentionally does not check this gate, and it has no
place on this loop.

## Route

`next Q###` dispatches a single scoped prompt naming only the current
pending responder — never the other configured responders, and never the
blocked entity's key. Claim `Q###` under the parent's own session before
handing that prompt to a responder worker; a second `next Q###` issued while
the claim is still live collapses to a paused response instead of naming a
competing responder (AC-007). Keep the blocked entity's own lease alive by
heartbeat for the whole consultation — that lease is a separate concern from
the Question's lease and is owned exactly as the rest of this protocol
already documents.

## Respond

The **parent** — never the worker — transcribes the responder's answer.
ADR-005 forbids workers from running Shark mutation commands, so the worker
returns its answer as structured text in its final response; the parent
copies that text into `question respond Q### --session <parent-sid>
--responder <identity> --summary ... --evidence-pointer ...` under the
lease it already holds on `Q###` (REQ-F-005). `RecordResponse` requires
`claim.SessionID == input.SessionID && claim.ClaimedBy == input.Responder` —
a response submitted without the matching parent-held lease is rejected,
never silently ignored. A replayed `question respond` carrying the identical
session, responder, summary, and evidence pointer is idempotent: retrying a
call that failed only in delivery, not in the write, never produces a
duplicate response record.

`--evidence-pointer` is bounded, validated text — the same
`ValidateQuestionBoundedText` check that rejects `system prompt`,
`user prompt`, `bearer `, and comparable markers elsewhere, which is
precisely the structural enforcement of "transcripts and rendered prompts
never become council evidence." F09 adds no second validator for this; it
reuses the one E39 already owns.

Once every configured responder has completed, the Question moves itself to
its ready-for-resolution status on its own — no separate parent action is
needed to notice that everyone has answered.

## Resolve

`question resolve Q### --owner <resolution-owner> --resolution-kind <kind>
--resolution-pointer <pointer>` closes the Question (REQ-F-006) once every
responder has completed. The owner must match the identity
`configure-workflow` set; classification and destination validation happen
before the Question transitions, reusing the same destination checks E39
already runs — F09 adds nothing new here either.

Resolving is what releases the gate: once `Q###` is `resolved`, the
`question_blocks` predicate this file's Gate section relies on no longer
qualifies, and the dispatched entity's next status-advance attempt succeeds.

## Result

One `Q###` exists, one `question_blocks` edge gated the dispatched entity
for exactly as long as the Question stayed open or answering, and the
current responder received exactly one scoped prompt at a time. The parent
transcribed the answer and resolved the Question under its own claim and
authority — the worker never touched a Shark mutation command. No bespoke
question, handoff, or resolution record was created to get here.
