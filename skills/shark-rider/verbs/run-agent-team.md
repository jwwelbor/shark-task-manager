# /shark-rider run-agent-team — Topology adapter entrypoint

Run a selected root through the canonical Shark Attack team topology.

Usage:

```
/shark-rider run-agent-team <epic-key|feature-key>
/shark-rider run-agent-team --sprint S###
```

## Prerequisites

Before you delegate, confirm that the active host provider supports the team,
follow-up, interrupt, and isolation capabilities required by the selected
topology. Check the current branch and worktree state. On `main`/`master`, an
unrelated branch, an unresolved merge, or unexpected dirty work, ask the owner
before continuing. If the required capability or topology evidence is absent,
use the `Sequential` topology described by the canonical procedure.

## Procedure

1. For the non-sprint form, accept only an epic or feature key. If the key is
   a task, bug, change-card, tech-debt item, Question, or another leaf,
   report `run-agent-team requires an epic or feature root; no team run was
   started.` and stop.
2. Retrieve the canonical procedure through the bundle boundary:

   ```bash
   shark skill get shark-attack workflows/parallel-team.md
   ```

   If retrieval is unavailable, report `The Shark Attack parallel-team
   procedure is unavailable in this bundle; no team run was started.` and
   stop. Do not read, copy, or construct a host-local substitute.
3. For `<epic-key|feature-key>`, follow the retrieved canonical Shark Attack
   `parallel-team.md` procedure with that root key.
4. For `--sprint S###`, follow the retrieved procedure in sprint mode. The
   active sprint backlog is the only selection universe.
5. Let the canonical procedure choose keys and assign each selected key to one
   ordinary keyed `/shark-rider run` parent. Do not construct a worker prompt,
   calculate a DAG, claim a delivery entity, or advance or release its status
   here.
6. Report the closing state. Only the owner may start or close the sprint; do
   not perform either lifecycle action from this entrypoint.

## Result

The host entrypoint supplies topology preconditions and then delegates all
selection, keyed dispatch, Question handling, and integration to the canonical
topology adapter.
