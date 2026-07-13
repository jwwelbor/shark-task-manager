# Project bootstrap execution contract

This is the execution map for `/shark-rider project bootstrap`. It is a
host-side procedure contract, not a second Shark workflow.

## Parent/child handoff

| Phase | Owner | Observable evidence | Return to parent |
|---|---|---|---|
| Prepare Shark | bootstrap | `shark admin init` or `shark admin validate` succeeds | local store is ready |
| Seed progress | bootstrap | `docs/product/progress.md` exists before child work | selected setup parameters are visible |
| Product design | product-design adapter | bundle retrieved; each selected D artifact is written; progress checkpoint follows each artifact | `CHILD ACTION RESULT` with artifacts and next step |
| Brownfield analysis | brownfield adapter | each selected analysis area produces durable output and checkpoint | `CHILD ACTION RESULT` with areas and next step |
| Architecture gate | bootstrap | D01/D04/D07 evidence is reconciled; provisional state is cleared or explicitly deferred | approval, deferral, or blocker |
| Handoff | bootstrap | selected artifacts and unresolved decisions are visible | first epic-planning action |

The parent loads and follows the child Rider procedure in the same host run. A
child must not emit another `/shark-rider ...` dispatch or create a recursive
host loop. Bundle retrieval remains owned by the child adapter.

## Bounded smoke run

Use a temporary fixture project and a deliberately bounded scope (D01-D04 or
D01-D07). Monitor these checkpoints in order:

1. Existing Shark data and configuration are preserved.
2. `progress.md` is created before product or analysis work starts.
3. The child procedure is loaded, the bundle is retrieved, and the first child
   artifact is produced.
4. `progress.md` changes after the durable artifact, with the active item
   `[~]` and completed items `[x]`.
5. Interrupt after one artifact, re-enter bootstrap, and confirm it resumes at
   the next evidence gap without regenerating the completed artifact.
6. Confirm the child returns a structured result and bootstrap consumes it.
7. For D04 feedback, confirm `feasible as described` still reaches the
   architecture integration gate and that other feedback returns to bootstrap
   reconciliation rather than dispatching recursively.

This smoke run is the real LLM/host validation. The normal test suite should
cover the deterministic contract and fixture state transitions without invoking
an LLM for every test.
