# Execute a wave: parallel-with-ownership and parallel-with-isolation dispatch

## Goal

Provide the dispatch and integration mechanics for a candidate set whose
execution topology already resolved (via `coordinate.md` step 2 and
`context/operating-model.md`) to `Parallel with ownership` or `Parallel
with isolation`. `batch.md` and `council.md` hand their candidate set
here once that resolution is made; this file does not re-derive the
topology or restate `context/operating-model.md`'s degradation rule.

## Parallel with ownership

Use only when the chair has already recorded exclusive file/component
ownership per candidate, the candidates' write sets are disjoint, and any
shared contract between them is frozen for the wave's duration.

1. Confirm the recorded ownership evidence names a disjoint write scope
   for every candidate in the wave. A candidate with no recorded scope,
   or a scope overlapping another candidate's, is not eligible for this
   wave — dispatch it Sequential instead (`direct.md`).
2. Dispatch each eligible candidate through its own keyed call: claim,
   hand off the returned prompt, apply the outcome, release — exactly as
   a single Direct dispatch does, repeated once per candidate.
   `skills/shark-rider/verbs/run.md`'s fork section documents this
   fan-out mechanism; do not restate it here.
3. Because ownership evidence guarantees disjoint write sets, no
   candidate's changes require conflict resolution against another's
   before integrating. Integrate each candidate's outcome as it
   completes.

## Parallel with isolation

Use when write sets may overlap and candidates instead run in separate
worktrees or equivalent isolated sessions, with the chair controlling
integration explicitly.

1. Confirm the recorded isolation evidence for every candidate in the
   wave: a provisioned isolated working root per the host's provider
   capability reference (`providers/codex.md`'s directed isolation,
   `providers/claude-code.md`'s worktree isolation, or an equivalent). A
   candidate with no recorded isolation evidence is not eligible for
   this wave — dispatch it Sequential instead.
2. Dispatch each eligible candidate the same way ownership waves do
   (step 2 above), but inside its own provisioned isolated root rather
   than the parent's own checkout.
3. The chair controls base revision, integration order, and conflict
   resolution explicitly. Isolation only guarantees the candidates ran
   without stepping on each other's working tree — not that their
   changes merge cleanly or land in the order they finished. Integrate
   one candidate at a time and re-run the post-integration gate before
   integrating the next.

## Producer/consumer order survives isolation

Isolation evidence proves candidates can write without colliding on
disk; it proves nothing about whether their changes are safe to run at
the same time. A candidate that depends on another candidate's
not-yet-integrated contract is not eligible for the same wave as its producer,
even when both have valid isolation evidence — integrate the producer
first, then dispatch the consumer. This holds under `Parallel with
ownership` too: ownership evidence only proves disjoint write scope,
not that the scopes are logically independent. See
`context/operating-model.md`'s isolation/dependency rule for the
underlying statement; this section applies it to wave eligibility, it
does not restate it.

## Result

Every candidate in the wave is dispatched through the same single-key
claim/dispatch/release mechanism a Direct dispatch uses. Ownership or
isolation evidence — never both required for the same wave — governs
whether it may run alongside its peers, and producer/consumer contract
order is preserved regardless of which evidence authorized the wave.
