# Shark Dispatch Prompt Assembly

Shark owns workflow routing and prompt assembly. The host harness owns only the
outer loop and the execution primitive that runs the returned prompt.

The canonical dispatch contract is `shark next <key> --json`. Harnesses should
not build prompts from `shark get ... orchestrator_action`, and they should not
load Shark skills or Shark specialist agents from their own filesystem.

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant Harness as CLI harness
    participant Skill as Shark skill run verb
    participant Next as shark next command
    participant Engine as prompt assembly engine
    participant Bundle as shark-data bundle
    participant Agent as host execution agent
    participant Status as shark status command

    User->>Harness: Start slash run with entity key
    Harness->>Skill: Load Shark run procedure
    Skill->>Next: shark next key --json
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
    Skill->>Next: shark next key --json

    alt Shark returns pause or archive
        Next-->>Skill: JSON with terminal wire action
        Skill-->>Harness: Stop and report state
    else Shark returns error
        Next-->>Skill: JSON or nonzero exit with error detail
        Skill-->>Harness: Stop and surface failure
    end
```

## Contract

- `shark next` returns the harness-facing wire shape:
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
That bypassed the `shark next` assembly path and failed because Shark specialist
personas live in `shark-data/agents/`, not in the host's native agent registry.

The remediation is to repoint the run harness to `shark next`, dispatch the
returned `prompt`, and treat native host agent selection as an adapter concern.
