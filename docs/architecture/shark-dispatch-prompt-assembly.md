# Shark Dispatch Prompt Assembly

This document covers keyed workflow dispatch. Shark owns keyed routing and
prompt assembly. The host harness owns only the outer loop and the execution
primitive that runs the returned prompt.

The canonical dispatch contract is `shark next <key> --json`. Harnesses should
not build prompts from `shark get ... orchestrator_action`, and they should not
load Shark skills or Shark specialist agents from their own filesystem.

## Command modes

This wire contract applies only to keyed `shark next <key> --json`. It requires
an entity key — bare `shark next` (no key) is invalid and errors, pointing the
operator at `shark plan`. `shark plan [root|collection]` is the separate,
read-only work-selection surface: bare `shark plan` returns one selected epic
(or an epic-only `parallel_candidates` tie); `shark plan <epic|feature>`
evaluates one hierarchy edge and returns direct children as a
`hierarchy_selection` envelope; `shark plan bugs|change-cards|tech-debt`
selects the next claimable standalone tier. None of `shark plan`'s selection
responses assemble a specialist dispatch prompt or claim, advance, or
normalize workflow state — only a leaf entity, or a parent already at its own
agent step, returns a rendered dispatch prompt from `shark plan`, identical to
what `shark next` would return for that same entity.

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant Harness as CLI harness
    participant Skill as Shark skill run verb
    participant Next as keyed shark next command
    participant Engine as prompt assembly engine
    participant Bundle as shark-data bundle
    participant Agent as host execution agent
    participant Status as shark status command

    User->>Harness: Start slash run with entity key
    Harness->>Skill: Load Shark run procedure
    Skill->>Next: shark next {KEY} --json
    Next->>Engine: Resolve entity status and workflow step
    Engine->>Bundle: Read workflow prompt file
    Engine->>Bundle: Inline skill content through include directives
    Engine->>Bundle: Inline specialist agent persona
    Engine-->>Next: Consolidated prompt plus routing metadata
    Next-->>Skill: JSON with action agent_type provider model prompt
    Skill->>Harness: Request execution with returned prompt
    Harness->>Agent: Spawn host-safe agent with response.prompt
    Agent-->>Harness: Return semantic outcome
    Harness->>Status: shark status advance key --outcome result
    Status-->>Harness: Transition result
    Harness->>Skill: Continue loop
    Skill->>Next: shark next {KEY} --json

    alt Shark returns pause or archive
        Next-->>Skill: JSON with terminal wire action
        Skill-->>Harness: Stop and report state
    else Shark returns error
        Next-->>Skill: JSON or nonzero exit with error detail
        Skill-->>Harness: Stop and surface failure
    end
```

## Contract

- `shark next <key> --json` returns the harness-facing wire shape:
  `action`, `agent_type`, `provider`, `model`, and `prompt`.
- `prompt` is the execution payload. It must already contain the rendered
  workflow prompt, included skill content, and the Shark specialist persona.
- `agent_type` is Shark metadata for routing, logs, and prompt provenance. It is
  not necessarily a native host subagent type.
- The harness may choose the host execution primitive, such as a built-in
  `general-purpose` subagent, but it must pass `response.prompt` unchanged.
- `shark get ... orchestrator_action` remains an inspection surface. It is not
  the dispatch prompt assembly API.

## Failure Mode Captured 2026-07-04

The failed `~/projects/wwgm` session used `shark get E01 --json` and attempted
to spawn `orchestrator_action.agent_type` directly as a Claude Code subagent.
That bypassed the keyed `shark next <key>` assembly path and failed because
Shark specialist personas live in `shark-data/agents/`, not in the host's
native agent registry.

The remediation is to repoint the run harness to `shark next <key> --json`,
dispatch the returned `prompt`, and treat native host agent selection as an
adapter concern.
