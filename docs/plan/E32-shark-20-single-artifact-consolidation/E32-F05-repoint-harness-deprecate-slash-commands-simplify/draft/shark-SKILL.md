---
name: shark
description: Dispatch loop trigger for shark workflows. Pulls fully-rendered prompts from 'shark next' and spawns the assigned agent until shark says stop.
when_to_use: when the user invokes /run on any shark entity (epic, feature, task, bug, change, tech-debt) — or any other shark dispatch entry point
allowed-tools: Bash(shark:*)
version: 2.0.0
---

# Shark — dispatch loop trigger

This skill is the **only shark-specific thing** the harness needs. It pulls fully-assembled prompts from `shark next` and spawns the assigned agent. All workflow logic lives inside the shark binary; the harness just dispatches.

## The dispatch loop

```
while true:
  response = shark next <key> --json

  case response.action:
    spawn_agent:
      spawn agent of type response.agent_type with prompt response.prompt
      on completion: shark advance <key>

    pause:
      stop, report status to user

    archive:
      stop, report archive to user

    error:
      stop, surface response.error to user
```

That's the entire skill. The prompt comes fully assembled — skill content inlined, partials resolved, variables substituted. The harness does not load skills, does not parse status names, does not know the workflow.

## Per-harness adapter notes

Each harness implements `spawn agent of type X with prompt P` differently:

- **Claude Code**: `Agent` tool with `subagent_type=X` and `prompt=P`. On completion, the parent run resumes and calls `shark advance`.
- **Codex / Copilot / Gemini**: harness-specific — the harness translates the agent type to its native subagent contract.

Pre-flight checks, retry/error UX, and "pause" rendering may stay harness-side because each harness handles them differently. Keep this minimal — if it grows, it's a sign that orchestration is leaking back into the harness.

## How to use

The harness invokes this skill on any shark dispatch entry point (e.g., `/run E01-F02-001`). The user does not invoke this skill directly.

For **shark CLI usage** (querying state, creating entities, manual status changes, etc.), see the canonical shark reference at `<shark-data>/skills/shark-cli-reference.md` — that's the user-facing documentation that travels with the binary.

## Reference / debugging

- `shark next <key> --preview` — show what `shark next` would return without dispatching.
- `shark next <key> --json | jq .prompt` — inspect the rendered prompt for a debug session.
- `shark next <key> --json | jq .agent_type` — check which agent will be spawned.
- `shark validate` — check that all `{{include:}}` paths resolve and all agent / prompt references in workflow YAML exist.

## Pause / resume

If `response.action == "pause"`, stop. The user sees the pause reason and chooses what to do (retry, manually advance, abandon). On resume, dispatch the entity again — `shark next` will return whatever's appropriate for the current state.

## Hooks

Pre-tool / post-tool hooks defined in `HOOKS.md` (this skill's sibling file) fire around the dispatch loop's `shark next` and `shark advance` calls. Hooks are harness-side, not shark-side.

## What this skill is NOT

- It does NOT contain workflow logic.
- It does NOT load other skills (the engine inlines them into the prompt).
- It does NOT parse status names or know which agents handle which steps.
- It does NOT hold project-specific configuration (that lives in `<project>/.sharkconfig.json` and `<project>/shark-data/`).

If you find yourself adding any of these to this skill, stop — the right home is shark-side (workflow YAML, prompt files, partials, or the engine binary itself).
