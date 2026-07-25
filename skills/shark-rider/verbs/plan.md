# /shark-rider plan — Execution-shape recommendation

Usage:

```
/shark-rider plan [entity-key|bugs|change-cards|tech-debt]
```

Recommend how to execute the selection returned by Shark. This verb
recommends only: it never claims, dispatches, advances, releases, or launches
an agent team.

## Procedure

1. Accept zero or one root argument. Reject additional arguments. Invoke
   `shark plan [root]` exactly once and retain the response unchanged,
   especially every dispatch `entities[].prompt`.
   Shark caps tied candidates using `.sharkconfig.json`
   `max_parallel_items` (default `5`).
2. Classify the response:
   - bare `mode=portfolio_selection`, `action=select_epic` — report that epic
     as the next root and recommend `/shark-rider run <entity-key>`.
   - bare `mode=portfolio_selection`, `action=parallel_candidates` — evaluate
     the epic-only candidate list below. These entries are selectors, not
     dispatch prompts.
   - bare `mode=portfolio_selection`, `action=pause` — report its evidence or
     ordering reason; do not guess.
   - `mode=hierarchy_selection`, `action=select_feature|select_task` — report
     the selected direct child and recommend `/shark-rider run <entity-key>`.
   - `mode=hierarchy_selection`, `action=parallel_candidates` — evaluate the
     direct-child candidates below. These entries contain no worker prompts.
   - `action=parallel_dispatch` (standalone collections) — evaluate the entire
     batch below.
   - a plain keyed dispatch response (a leaf entity, or a parent already at
     its own agent step) — recommend the standard sequential
     `/shark-rider run <requested-root>` path.
   - `action=pause|archive` — report the stop state; do not invent work.
3. For a parallel batch, identify the scope from the invocation and response:
   - bare `parallel_candidates`: inspect
     `docs/product/cross-epic-integration-map.md` and only relevant `X-##` rows;
   - epic selection scope: resolve the epic-local interaction map from Shark-linked
     documents or the epic planning directory, then inspect only relevant
     `I-##` rows for returned feature candidates;
   - feature selection scope: inspect stored task dependency evidence only, using the
     returned task keys and their Shark dependency data;
   - standalone collection scope (`bugs`, `change-cards`, `tech-debt`): use the
     returned stored priority/severity tier and any stored entity relationship
     evidence; do not invent an integration map.
4. A map row is usable only when it is present, well formed, and resolved
   enough to support the decision. A missing, malformed, `proposed`, or
   `deferred` relevant row is an integration evidence gap. It cannot justify
   independent parallel launch.
5. Recommend exactly one execution approach:
   - **sequential execution** when a producer-to-consumer handoff must complete,
     only one normal dispatch exists, or evidence cannot support concurrency;
   - **independent parallel agents** when no shared decision remains and all
     relevant dependency and integration evidence establishes independence;
   - **agent team** with one coordinator and one scoped worker per entity when
     work can start together but still needs shared contract, UX/CX, or
     integration decisions.
6. Do not claim or launch anything. Always state that the operator may choose
   any one entity from a parallel batch and run it sequentially instead.

## Recommendation rules

- `parallel_execution=available` proves only that workflow state and stored
  dependencies allow concurrent dispatch. It does not prove product integration
  safety.
- Preserve each returned dispatch prompt exactly. If the operator later approves
  execution, the selected runner must obtain or use that exact Shark response;
  Rider must not rewrite or merge worker prompts.
- A producer-to-consumer dependency always chooses sequential execution.
- Missing integration evidence chooses sequential execution with an evidence
  gap, never independent parallel agents.
- Choose an agent team only when work can begin concurrently and a named shared
  decision genuinely needs a coordinator.

## Required response

Start with exactly one:

```
Recommendation: sequential
Recommendation: independent parallel agents
Recommendation: agent team
Recommendation: evidence gap
```

Then include:

### Shark result

State the requested root, returned mode and action, returned entity keys in
order, and whether Shark marked parallel execution available. Do not print
full prompts.

### Integration evidence

Name the relevant `X-##`, `I-##`, task dependency, standalone relationship, or
evidence gap that controls the recommendation.

### Execution approach

Give one approach and its decisive reason:

- sequential: show `/shark-rider run <root-or-selected-entity>`;
- independent parallel agents: show one concurrent
  `/shark-rider run <entity-key>` command per returned entity;
- agent team: name one coordinator responsibility and one scoped worker per
  returned entity, but do not launch it.

For every parallel batch, end this section by stating:

```
You may instead select any one entity from the batch and run it sequentially.
```

### Not selected

List paused, archived, blocked, claimed, lower-priority, deferred, or
evidence-incomplete work that the Shark response or inspected evidence exposes.
