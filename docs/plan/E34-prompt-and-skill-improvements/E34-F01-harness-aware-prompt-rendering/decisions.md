# E34-F01 Decision Record

## Status

Accepted for feature refinement review.

## Summary

We are choosing a **hybrid routing model** with a **clean client-facing response contract**.

Shark remains responsible for:

- workflow routing
- prompt assembly
- internal agent/persona selection for prompt inclusion

The client remains responsible for:

- final model selection
- final harness execution strategy
- subagent/execution-profile routing based on Shark-provided model metadata

## Decision 1: Use Hybrid Routing

We are **not** building a fully Shark-side harness matrix that picks the exact runtime model for every host.

We are also **not** using a pure client-side free-for-all where Shark returns only vague generic intent.

We are using a hybrid model:

- Shark produces the fully assembled orchestration prompt
- Shark returns structured model-routing metadata
- the client maps that metadata onto its own harness-specific execution choices

### Why

- keeps Shark as the workflow authority
- keeps harness-specific execution logic near the harness
- avoids hardcoding every host quirk into Shark
- still gives enough structure to avoid uncontrolled client drift

## Decision 2: Keep `agent` Internal To Shark

The workflow `agent` field remains authoritative inside Shark.

It is used for:

- selecting the Shark agent/persona body
- inlining that agent content into the assembled prompt

The client does **not** need `agent_type` in the response.

### Why

- `agent` is primarily a Shark prompt-assembly concept
- exposing `agent_type` would leak an internal routing key
- clients could misread it as a native harness subagent type
- that confusion is exactly what we want to avoid

## Decision 3: Clean Up Client-Facing Metadata

The client-facing response should be minimized.

The important response payload is:

- `prompt`
- `model.class`
- `model.effort`
- optionally `model.prompt_profile`

### Why

- `prompt` is the real orchestration payload
- the client only needs enough metadata to choose execution strategy
- extra Shark-internal metadata increases confusion without adding value

## Decision 4: Use A `model` Object, Not `hints`

We are **not** calling the routing metadata `hints`.

We are using a structured `model` object instead.

### Proposed shape

```json
{
  "prompt": "...fully assembled orchestration prompt...",
  "model": {
    "class": "strategic",
    "effort": "high",
    "prompt_profile": "researcher"
  }
}
```

### Why

- this data is part of the execution contract, not soft advice
- `model.*` is easier to understand than generic `hints.*`
- it gives the client a clear routing surface

## Decision 5: `model.class` And `model.effort` Are In Scope

We are treating these as the primary client-routing fields:

- `model.class`
- `model.effort`

### Meaning

- `model.class` describes the kind of model the client should choose
- `model.effort` describes the target reasoning effort level

The client is responsible for translating those into actual harness-specific runtime choices.

### Why

- we want portable routing intent, not Shark-locked model IDs
- the same Shark work may map to different concrete models per harness

## Decision 6: `model.prompt_profile` Is Allowed, But Secondary

We think `model.prompt_profile` may be useful for client-side execution routing, especially for choosing different subagent/execution profiles such as:

- `general`
- `researcher`
- `code-review`
- `qa`
- `uat`

But this field is secondary to `model.class` and `model.effort`.

### Why

- it may help clients choose the right local execution primitive
- it is a better client-facing routing key than `agent_type`

### Caution

This field should represent a client execution profile, not a disguised copy of Shark's internal `agent` field.

## Decision 7: Prompt Selection Stays In Shark

Prompt/persona selection remains Shark-owned.

That means:

- workflow config still declares the internal `agent`
- Shark still assembles the prompt from workflow prompt + agent/persona content
- the client receives the final assembled prompt and executes it

### Why

- preserves the `shark next` contract as the prompt assembly surface
- keeps Shark personas out of client-specific agent registries
- avoids repeating prompt-assembly logic in every client

## Decision 8: We Are Optimizing For A Narrower Client Contract

The client should not need to understand Shark workflow internals.

The client should only need:

- the orchestration prompt
- model-routing metadata

### Why

- smaller boundary
- less coupling
- lower drift risk
- easier future support for multiple harnesses

## Agreed Boundary

### Shark owns

- workflow state and routing
- prompt assembly
- internal `agent` selection
- returned `prompt`
- returned `model` metadata

### Client owns

- runtime model selection
- effort translation into harness-specific knobs
- subagent/execution-profile routing
- actual execution primitive

## Response Contract Direction

The cleaned-up response direction is:

```json
{
  "prompt": "...",
  "model": {
    "class": "strategic",
    "effort": "high",
    "prompt_profile": "researcher"
  }
}
```

Possible simplification if `prompt_profile` proves unnecessary:

```json
{
  "prompt": "...",
  "model": {
    "class": "strategic",
    "effort": "high"
  }
}
```

## Implications For Implementation

If we build this feature, Shark likely needs:

1. internal model-routing metadata resolution
2. response-contract changes on `shark next`
3. internal retention of `agent` for prompt assembly only
4. removal or demotion of client-facing `agent_type`
5. client-supplied harness identity/capability handling where needed for audit or future routing expansion

## Non-Decisions

These are still not locked:

- exact schema and naming for the config file that resolves `model.class` and `model.effort`
- whether `prompt_profile` is required in v1 or optional
- whether backward compatibility requires temporarily returning legacy fields alongside the new contract

## Working Recommendation

For E34-F01, the design should proceed assuming:

- hybrid routing
- internal Shark `agent`
- client-facing `prompt`
- client-facing `model.class`
- client-facing `model.effort`
- optional `model.prompt_profile`

This is the current agreed direction for review before build.
