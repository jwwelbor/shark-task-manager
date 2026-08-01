# Scope Boundaries

**Epic**: [Question and Decision Workflow Management](./epic.md)

## In Scope

E39 delivers the generic Question entity and the platform surfaces necessary to
use it safely: entity registration, structured context, workflow/claim
integration, relationships, keyed dispatch, focused queries, advancement gates,
and durable resolution provenance. See [Requirements](./requirements.md) for
testable requirements.

## Out of Scope

1. **Repairing E38-F09's existing adapter or continuation defects**
   - E39 supplies the generic lifecycle that F09 may consume later; it does not
     repair F09's provider-adapter provenance or live-continuation behavior.

2. **Unrestricted chat or transcript storage**
   - A Question is a coordination record, not a conversation system. Context is
     bounded; material records belong in typed notes or authoritative documents.

3. **Worker-owned actions on linked work**
   - Responding to a Question cannot claim, advance, or resolve a linked entity.
     The linked-work owner retains its workflow authority.

4. **Global or implicit blocking**
   - An open Question cannot block unrelated entities. Blocking requires both
     an explicit blocking designation and an explicit relationship.

5. **Implementation work before refinement and decomposition**
   - The epic does not authorize implementation tasks until the unresolved
     design decisions are answered and the work is decomposed.

6. **Parallel responder collection in the first release**
   - E39 intentionally uses serial responses to preserve existing claim safety.
     Parallel response collection is a future candidate, not a release promise.

## Deferred Follow-On Candidates

| Candidate | Why deferred |
|---|---|
| Parallel response collection | Requires a concurrency and merge model beyond the serial claim invariant. |
| Adoption/time-to-resolution KPI program | Requires a telemetry source, baseline, target, cohort, and accountable owner. |
| Rich viewer interaction design | First-release viewer/API requirements remain a design decision. |

