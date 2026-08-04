# Batch: parallel research, conflict-aware write waves

## Goal

Execute work that `coordinate.md` classified as `Batch` coordination
level: several related entities that benefit from one scope and conflict
analysis but do not need sustained specialist adjudication. Read-only
research may run concurrently; writes run under the execution topology
`coordinate.md` classified in its own step 2 — `Sequential`, `Parallel
with ownership`, or `Parallel with isolation` — never assumed parallel
just because the coordination level is `Batch`. See
`context/operating-model.md` for the topology definitions and its
degradation rule; this file does not restate them.

## Procedure

1. **Enumerate the related entities.** Use the existing selection
   surface (a hierarchy tier, a sprint pull, or a fork response) to
   gather the candidate set — do not invent a second selection
   mechanism.
2. **Research in parallel where read-only.** Reading code, docs, or
   Shark state for multiple candidates concurrently needs no ownership
   or isolation evidence, because nothing is written. Keep research
   bounded to the candidate set; do not fan out beyond it.
3. **Resolve the write topology per `coordinate.md` step 2.** If the
   resolved topology is `Sequential`, dispatch each candidate one at a
   time through the ordinary Rider loop (as `direct.md` does, repeated
   per candidate). If it resolved to `Parallel with ownership` or
   `Parallel with isolation`, hand the candidate set to
   `execute-wave.md` for the wave shape and integration mechanics.
4. **Never let isolation evidence reorder dependent writes.** A
   candidate that consumes a contract another candidate in the same
   batch produces runs only after that contract is integrated —
   never in the same wave as its producer — regardless of which
   topology the batch resolved to. Isolated write sets do not imply
   independent write *order*; see `context/operating-model.md`'s
   isolation/dependency rule for the full statement.
5. **Record one consolidated observation.** When the batch completes (or
   partially completes), write a single durable record under
   `docs/council/` describing the batch's scope, resolved topology, and
   outcome per candidate, following `context/message-schema.md`'s
   artifact contract. Do not write one record per candidate — the batch
   is the observation's scope.

## Result

Research for the batch's candidate set may overlap in time. Writes
follow the topology `coordinate.md` already resolved, and any
producer/consumer ordering between candidates is preserved regardless of
that topology. One consolidated record captures the batch's outcome.
