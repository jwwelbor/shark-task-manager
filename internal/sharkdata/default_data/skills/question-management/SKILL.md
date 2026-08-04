---
name: question-management
description: Create, route, answer, and resolve durable Shark Questions for material unresolved decisions. Use when design, requirements, planning, feasibility, review, or implementation work finds an open item whose answer changes scope, a contract, acceptance, sequencing, risk acceptance, or a lasting product or architecture decision.
---

# Question Management

Use Shark Questions as the lifecycle record for an unresolved decision. Keep
the eventual answer in its narrowest authoritative product, architecture, or
entity document; use the Question's evidence and resolution pointers to link
the two. Do not treat a chat exchange, a `TBD`, or a council file as a
substitute for the Question lifecycle.

## Decide whether to record a Question

Apply this materiality test before you create or reuse a Question. Record one
only when its answer changes one or more of:

- Scope, acceptance criteria, sequencing, or an approved deferral.
- A product, requirements, architecture, security, data, frontend, or API
  contract.
- A decision with a lasting consequence, including accepted risk.
- Whether a specific Shark entity can safely advance.

For a non-material item such as a routine fact lookup, an already-settled
choice, a low-impact writing preference, or a speculative assumption, record
the rationale in the working document instead of creating a Question.

## Record and route the Question

1. Search for existing coverage before minting a record. Use deduplication
   before creation:

   Search Questions for the decision phrase, then page Question
   lists in `open`, `answering`, and `ready_for_resolution` status with
   `shark question list --status=<status> --limit=100 --offset=<offset>`.

   Reuse or update a Question that asks the same decision. Never create a
   duplicate merely because it appears in another design document.

2. Create one decision-shaped record. State the context, answer needed or
   viable options, consequence, affected entity/document, and evidence paths.

   Use `shark question create "<decision to make>"` with `--summary`,
   `--description`, `--requester`, and `--blocking` only when the Question
   qualifies as a gate.

3. Configure the smallest real responder set and a real resolution owner.
   Configure before adding a block: an earlier `question_blocks` link is inert.

   Use `shark question configure-workflow Q### --resolution-owner="<owner>"
   --responder="<identity>"`, then link the decision source with
   `shark related-docs add "Decision source" <path> --question=Q###`. If a
   gate is justified, use `shark link Q### <entity-key>
   --type=question_blocks` after configuration.

   Add `question_blocks` only when the named entity cannot safely proceed
   without the answer. Do not use it as a general relationship or priority
   marker.

4. When Shark Attack governs the execution, route through its canonical
   material threshold at `skills/shark-attack/workflows/council.md`. Its
   classification and category/default-path table decide council versus routine
   routing; do not restate or replace that policy here. A Question classified
   routine remains in the standard E39 lifecycle and creates no council
   artifact.

5. Respect the active execution protocol. When workers may not mutate Shark,
   return a structured Question proposal or answer to the parent. The parent
   owns Question creation, claims, responses, resolution, and entity gates.

6. Preserve existing consumers. `solution-walkthrough` may present an
   operator-approved Question response, but it must not create or resolve a
   Question automatically.

## Answer and resolve

1. Update the narrowest authoritative decision record first:

   | Decision scope | Authoritative record |
   |---|---|
   | Product direction or cross-epic sequencing | `docs/product/progress.md` Decision Log |
   | Shared architecture or reusable policy | `docs/architecture/adr/` |
   | Epic direction | Epic architecture and integration maps |
   | Feature, task, bug, or change decision | Authoritative local spec/design |

2. Claim and respond only as the configured responder. Use the authoritative
   record's path as evidence. Do not put credentials, rendered prompts, full transcripts,
   or unbounded chat history into Question fields.

   Dispatch `shark next Q### --json`. Continue only when its
   `current_responder` matches the responder, then claim it and capture the
   returned session ID:

   ```sh
   shark claim Q### --by=<current-responder> --json
   shark question respond Q### --session=<session-id> \
     --responder=<current-responder> --summary="<bounded response>" \
     --evidence-pointer=<durable-record-path>
   shark release Q### --session=<session-id>
   ```

3. Resolve only after all configured responses are recorded and the owner has
   an authoritative decision pointer:

   Use one of `local_clarification`, `feature_change`, `product_decision`,
   `architecture_decision`, `follow_up_work`, or `no_lasting_consequence` as
   the resolution kind. Each kind has a distinct pointer contract:

   | Resolution kind | Resolution pointer |
   |---|---|
   | `local_clarification` | Existing `note:<id>` |
   | `feature_change` | Existing authoritative feature specification or document path |
   | `product_decision` | Existing `docs/product/progress.md#<decision-anchor>` |
   | `architecture_decision` | Existing `<adr-path>;<affected-architecture-or-spec-path>` |
   | `follow_up_work` | Existing follow-up Shark entity key |
   | `no_lasting_consequence` | No pointer; bounded responses and Question history are the durable record |

   For every kind except `no_lasting_consequence`, use the matching validated
   pointer:

   ```sh
   shark question resolve Q### --owner=<resolution-owner> \
     --resolution-kind=<resolution-kind> \
     --resolution-pointer=<durable-record-path>
   ```

   For `no_lasting_consequence`, omit `--resolution-pointer`:

   ```sh
   shark question resolve Q### --owner=<resolution-owner> \
     --resolution-kind=no_lasting_consequence
   ```

   Keep the Question open if the decision remains unresolved. Resolving a
   Question releases a qualifying `question_blocks` gate; do not resolve merely
   because a discussion occurred.

## Report the outcome

Report the Question key, route (routine or council), affected entity if
blocked, current lifecycle status, and authoritative decision-record path.
State explicitly when no Question was needed and why.
