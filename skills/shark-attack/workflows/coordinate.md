# Coordinate: select and route by the two-axis model

## Goal

Apply `context/operating-model.md`'s two-axis model to one piece of
requested work, then hand it to the procedure that executes it. This file
routes only — it does not restate the axis definitions, the illustrative
examples, or the degradation rule. Read `context/operating-model.md` first.

## Procedure

1. **Classify coordination level** using Axis 1 of
   `context/operating-model.md` (`Direct`, `Batch`, or `Council`).
2. **Classify execution topology** using Axis 2 and its degradation rule
   in `context/operating-model.md` (`Sequential`, `Parallel with
   ownership`, or `Parallel with isolation`). Classify this axis
   independently of step 1's result — see `context/operating-model.md`'s
   decision table for worked examples of that independence.
3. **Route by the coordination level classified in step 1.** The topology
   axis never changes which procedure file is selected — only how the
   selected procedure dispatches:

   | Coordination level | Procedure |
   |---|---|
   | `Direct` | `direct.md` |
   | `Batch` | `batch.md` |
   | `Council` | `council.md` |

   `batch.md` and `council.md` apply the topology classified in step 2 to
   choose their own dispatch shape, calling into `execute-wave.md`'s wave
   mechanics whenever that topology is a parallel one.

## Result

Coordination level, classified independently of execution topology,
selects exactly one destination procedure. That procedure then applies
the topology already classified in step 2 to decide whether it dispatches
one worker, a wave, or a council round.
