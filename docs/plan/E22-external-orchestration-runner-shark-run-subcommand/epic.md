---
epic_key: E22
title: External Orchestration Runner - shark run subcommand
description: Replace the LLM-controlled /run orchestration loop with a Go subcommand inside shark that owns the entire dispatch lifecycle. The shark run command reads entity state via internal service calls, invokes Claude CLI or Codex CLI for each workflow stage via os/exec, and only advances status after the external process exits successfully. This removes the LLM from the control loop entirely — it can no longer skip stages, blast through gates, or advance its own status. Claude is the primary development agent, Codex acts as red-team, and the workflow templates specify which agent to use at each stage.
---

# External Orchestration Runner - shark run subcommand

**Epic Key**: E22

---

## Goal

### Problem

The current `/run` orchestration loop is controlled by the LLM itself — Claude reads shark state, decides what to do, spawns subagents, and advances statuses. In practice, Claude routinely skips workflow stages entirely, blasting through all 13 status transitions without performing any actual work. This was observed directly: a single bash loop calling `shark status advance` 13 times took a task from `draft` to `completed` in seconds with zero work done. The root cause is that prompt-level guards are fundamentally unenforceable — no amount of instruction prevents the LLM from taking shortcuts, fabricating completion notes, or simply advancing past gates.

### Solution

Move the orchestration loop out of the LLM and into shark itself as a `shark run` Go subcommand. Shark reads entity state via its internal service layer (no JSON round-tripping), invokes Claude CLI or Codex CLI via `os/exec` for each workflow stage, and only advances status after the external process exits successfully. The LLM becomes a scoped worker — it gets invoked with a prompt for one stage, does the work, and exits. It never has access to `shark status advance`.

### Impact

- Workflow stages are mechanically enforced — impossible for the LLM to skip
- Every stage produces actual work before advancement occurs
- Quality gates (code review, QA, UAT) are guaranteed to execute
- Development velocity improves because rework from skipped stages is eliminated

---

## Business Value

**Rating**: High

Workflow integrity is the foundation of the entire PDLC system. When the orchestrator skips stages, it undermines every quality gate downstream — code review, QA, and UAT become meaningless because the work they're supposed to validate was never done. Fixing this makes the entire shark workflow system trustworthy.

---

## Background & Discovery

### What we observed

In another project, the `/run` command (which is a Claude Code skill defined in `skills/orchestration/workflows/run.md`) was supposed to drive a task through its workflow by reading `orchestrator_action` from shark, spawning agents at each stage, and looping. Instead, it effectively did:

```bash
for i in $(seq 1 13); do shark status advance T-E21-F03-007; done
```

Result: `draft -> ready_for_refinement_ba -> in_refinement_ba -> ... -> completed` with no agents spawned and no work done.

### Why prompt-level fixes won't work

The `/run` workflow instructions are already explicit: "Never advance more than once without re-reading orchestrator_action," "spawn the agent every time, unconditionally." The LLM ignored all of it. Adding more instructions, stronger wording, or additional verification prompts won't change the fundamental problem: the LLM controls its own enforcement. It can always find a shortcut.

### Why shark-level enforcement works

A Go process that calls `claude -p "do development for task X"` and waits for exit code 0 before calling `taskService.Advance(key)` is something the LLM literally cannot bypass. It doesn't have access to the advance function. It does its scoped work and exits.

---

## Approach

### Architecture: `shark run` Go subcommand

The orchestration loop lives inside shark as a new command. It uses shark's internal service layer directly and shells out to Claude/Codex via `os/exec`.

```
shark run E21-F03-007

loop:
  task := taskService.Get(key)        // direct Go call
  action := task.OrchestratorAction

  if action == "spawn_agent":
    if agent_type requires claude:
      exec("claude", "-p", instruction, "--disallowedTools", "Bash(shark status advance*)")
    if agent_type requires codex:
      exec("codex", "exec", "-m", model, "-s", sandbox, instruction)

    if exit_code == 0:
      taskService.Advance(key)        // only shark advances
    else:
      log failure, stop or retry

  if action == "pause": stop
  if action == "archive": stop
```

### Agent roles

- **Claude CLI** (`claude -p`): Primary agent for all development stages — BA refinement, tech refinement, development, code review, QA
- **Codex CLI** (`codex exec`): Red-team verification at QA and UAT stages
- **Which agent to use is specified by the workflow templates** — the `orchestrator_action.agent_type` field already carries this, and templates can specify `claude`, `codex`, or both

### Key CLI patterns

**Claude (headless, scoped):**
```bash
claude -p "instruction from template" \
  --allowedTools "Bash(git:*)" "Read" "Edit" \
  --disallowedTools "Bash(shark status advance*)" \
  --max-turns N \
  --output-format json
```

**Codex (red-team, read-only):**
```bash
codex exec -m gpt-5.2-codex -s read-only \
  -c model_reasoning_effort=high \
  --skip-git-repo-check \
  "red-team instruction from template"
```

### What this replaces

The Claude Code `/run` skill (`skills/orchestration/workflows/run.md` and `commands/run.md`) becomes unnecessary for task execution. It may still serve as a lightweight wrapper that calls `shark run` under the hood, or it can be retired entirely.

---

## Key Design Decisions

1. **Go over Python/Bash** — shark already exists as a Go binary with full access to the service layer. No JSON round-tripping, single binary to deploy, natural place for enforcement.

2. **Advance AFTER, not before** — The current `/run` workflow advances status before spawning the agent (moving from `ready_for_*` to `in_*`). The new design can still do this for the `ready → in` transition, but the `in → ready_for_next` transition only happens after the CLI exits 0.

3. **LLM cannot advance forward** — Claude is invoked with `--disallowedTools` blocking shark advance commands. It can still call `shark status set` to send a task backward (e.g., code review rejecting to `ready_for_development`), preserving quality gate rejection.

4. **Templates drive agent selection** — No hardcoded mapping of stages to agents. The `orchestrator_action` from shark's workflow templates already specifies `agent_type`, `instruction`, and `skills`. The `shark run` command just reads and dispatches.

---

*Last Updated*: 2026-03-20
