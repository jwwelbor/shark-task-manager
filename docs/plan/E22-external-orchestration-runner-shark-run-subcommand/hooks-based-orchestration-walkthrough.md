# Hooks-Based Orchestration — Alternative to `shark run`

**Purpose**: Explore using Claude Code hooks (specifically the `Stop` hook) to drive workflow advancement automatically, replacing or complementing the Go-based `shark run` controller from E22.

**Date**: 2026-03-23

---

## Core Idea

Instead of a Go binary that owns the dispatch loop, use **Claude Code hooks** to mechanically advance status after each agent invocation. The outer loop becomes a trivial bash script (or even manual invocations), while hooks handle the "advance on completion" enforcement that E22 was designed to provide.

**E22 approach**: Go binary → dispatch agent → wait for exit → advance status → loop
**Hooks approach**: Bash loop → launch `claude -p` → Stop hook fires on exit → hook advances status → loop continues

---

## How Claude Code Hooks Work (Relevant Subset)

### The `Stop` Hook

Fires **every time Claude finishes responding** (completes a turn). This is the key event for orchestration.

```
User prompt → Claude works → [tools, edits, etc.] → Claude stops → Stop hook fires
```

**What the hook receives** (JSON on stdin):
```json
{
  "session_id": "abc123",
  "cwd": "/home/user/projects/shark-task-manager",
  "hook_event_name": "Stop",
  "stop_hook_active": false
}
```

**What the hook can do**:
- Run arbitrary shell commands (e.g., `shark status advance`)
- Return exit code 0 (allow stop) or exit code 2 + JSON (block stop, force Claude to continue)
- Access environment variables set by the parent process

**Infinite loop guard**: When a Stop hook blocks with `decision: "block"`, Claude continues working. The *next* Stop event sets `stop_hook_active: true`, allowing the hook to detect the second attempt and let Claude actually stop.

### Hook Configuration

Hooks are configured in `.claude/settings.json` (project-level, committable) or `.claude/settings.local.json` (local, gitignored):

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/advance-shark-task.sh"
          }
        ]
      }
    ]
  }
}
```

---

## Architecture: Three Components

### 1. The Stop Hook — `advance-shark-task.sh`

Fires after every Claude session. Reads the task key from environment, advances status.

```bash
#!/bin/bash
set -euo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Only act when SHARK_TASK_KEY is set (orchestrated mode)
TASK_KEY="${SHARK_TASK_KEY:-}"
if [ -z "$TASK_KEY" ]; then
  exit 0  # Not in orchestration mode, do nothing
fi

# Don't advance if this is a re-fire after block (safety)
STOP_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active // false')
if [ "$STOP_ACTIVE" = "true" ]; then
  exit 0
fi

# Advance the entity to its next workflow status
cd "$CLAUDE_PROJECT_DIR" 2>/dev/null || cd "$(echo "$INPUT" | jq -r '.cwd')"

RESULT=$(shark status advance "$TASK_KEY" --json 2>&1) || {
  echo "Hook: Failed to advance $TASK_KEY: $RESULT" >&2
  exit 0  # Don't block Claude from stopping on advance failure
}

NEW_STATUS=$(echo "$RESULT" | jq -r '.to_status // .status // "unknown"')
echo "Hook: Advanced $TASK_KEY → $NEW_STATUS" >&2
exit 0
```

### 2. The Orchestration Loop — `shark-orchestrate.sh`

A bash script that replaces the Go RunController. Reads status, determines action, launches the appropriate agent, repeats.

```bash
#!/bin/bash
set -euo pipefail

# Usage: ./shark-orchestrate.sh E10-F03-001 [--dry-run]
KEY="$1"
DRY_RUN="${2:-}"

echo "=== Orchestrating $KEY ==="

while true; do
  # 1. Read current status
  STATUS=$(shark get "$KEY" --field status 2>/dev/null)
  echo ""
  echo "--- Status: $STATUS ---"

  # 2. Check for terminal status
  if [[ "$STATUS" == "completed" || "$STATUS" == "cancelled" ]]; then
    echo "=== $KEY reached terminal status: $STATUS ==="
    break
  fi

  # 3. Read orchestrator action for this status
  ACTION_JSON=$(shark config get-status-action "$STATUS" --json 2>/dev/null) || {
    echo "ERROR: No orchestrator action defined for status '$STATUS'"
    break
  }

  ACTION=$(echo "$ACTION_JSON" | jq -r '.action')
  AGENT_TYPE=$(echo "$ACTION_JSON" | jq -r '.agent_type // "developer"')
  PROVIDER=$(echo "$ACTION_JSON" | jq -r '.provider // "anthropic"')
  MODEL=$(echo "$ACTION_JSON" | jq -r '.model // empty')
  INSTRUCTION=$(echo "$ACTION_JSON" | jq -r '.instruction // empty')

  echo "  Action: $ACTION | Agent: $AGENT_TYPE | Provider: $PROVIDER"

  case "$ACTION" in

    advance_status)
      # Auto-advance, no agent needed
      if [ -n "$DRY_RUN" ]; then
        echo "  [dry-run] Would advance $KEY"
        # Simulate by reading what the next status would be
        NEXT=$(shark status options "$KEY" --json 2>/dev/null | jq -r '.[0] // "unknown"')
        echo "  [dry-run] Next status would be: $NEXT"
        break  # Can't continue dry-run without actually advancing
      fi
      shark status advance "$KEY" --json >/dev/null
      echo "  Advanced (no agent dispatch)"
      ;;

    spawn_agent)
      if [ -n "$DRY_RUN" ]; then
        echo "  [dry-run] Would dispatch $PROVIDER/$AGENT_TYPE"
        echo "  [dry-run] Instruction: ${INSTRUCTION:0:100}..."
        break
      fi

      # Advance ready_for_* → in_* before dispatch (marks work started)
      if [[ "$STATUS" == ready_for_* ]]; then
        shark status advance "$KEY" --json >/dev/null
        STATUS=$(shark get "$KEY" --field status)
        echo "  Advanced to $STATUS (work started)"
      fi

      echo "  Dispatching $PROVIDER agent ($AGENT_TYPE)..."

      # Dispatch based on provider
      if [ "$PROVIDER" = "anthropic" ] || [ "$PROVIDER" = "claude" ]; then
        # Claude CLI — the Stop hook will advance status on exit
        SHARK_TASK_KEY="$KEY" claude -p "$INSTRUCTION" \
          --disallowedTools "Bash(shark status advance*)" \
          --disallowedTools "Bash(shark task next-status*)" \
          --disallowedTools "Bash(shark status set*)" \
          --disallowedTools "Bash(shark task set-status*)" \
          ${MODEL:+--model "$MODEL"} \
          --output-format json \
          || {
            echo "  ERROR: Agent exited non-zero. Status stays at $STATUS."
            echo "  Re-run to resume from $STATUS."
            exit 1
          }
        # Note: Stop hook already advanced status via SHARK_TASK_KEY.
        # But the hook only fires for Claude. For non-Claude providers,
        # we advance manually below.

      elif [ "$PROVIDER" = "openai" ] || [ "$PROVIDER" = "codex" ]; then
        # Codex CLI — no hook integration, advance manually
        codex exec \
          ${MODEL:+-m "$MODEL"} \
          --full-auto \
          -c "instruction: $INSTRUCTION" \
          || {
            echo "  ERROR: Codex agent exited non-zero. Status stays at $STATUS."
            exit 1
          }
        # Manually advance since Codex doesn't trigger Claude hooks
        shark status advance "$KEY" --json >/dev/null
      fi

      echo "  Agent completed successfully."
      ;;

    cascade)
      echo "  Cascading to child entities..."
      # List children, run orchestration for each
      CHILDREN=$(shark list "$KEY" --json 2>/dev/null | jq -r '.[].key')
      for CHILD in $CHILDREN; do
        CHILD_STATUS=$(shark get "$CHILD" --field status 2>/dev/null)
        if [[ "$CHILD_STATUS" != "completed" && "$CHILD_STATUS" != "cancelled" ]]; then
          echo "  → Recursing into $CHILD ($CHILD_STATUS)"
          "$0" "$CHILD" $DRY_RUN  # Recursive call
        else
          echo "  → Skipping $CHILD ($CHILD_STATUS)"
        fi
      done
      # All children done, advance parent
      shark status advance "$KEY" --json >/dev/null
      echo "  All children complete. Advanced parent."
      ;;

    pause|wait_for_triage|check_or_resume)
      echo "=== PAUSED at $STATUS ==="
      echo "  Awaiting manual intervention."
      echo "  Resume with: $0 $KEY"
      break
      ;;

    archive)
      echo "=== $KEY archived ==="
      break
      ;;

    *)
      echo "ERROR: Unknown action '$ACTION' for status '$STATUS'"
      exit 1
      ;;
  esac
done

echo ""
echo "=== Orchestration of $KEY finished ==="
```

### 3. Kickoff

```bash
# Single task through full workflow
./shark-orchestrate.sh E10-F03-001

# Feature (drives tasks via cascade)
./shark-orchestrate.sh E10-F03

# Epic (drives features via cascade, which drive tasks)
./shark-orchestrate.sh E10

# Preview without executing
./shark-orchestrate.sh E10 --dry-run
```

---

## Walkthrough: E10 Epic via Hooks

Using the same E10 example from `run-loop-walkthrough-E10.md`.

### Happy Path

```
$ ./shark-orchestrate.sh E10

=== Orchestrating E10 ===

--- Status: draft ---
  Action: advance_status
  Advanced (no agent dispatch)

--- Status: ready_for_refinement ---
  Action: spawn_agent | Agent: business-analyst | Provider: anthropic
  Advanced to in_refinement (work started)
  Dispatching anthropic agent (business-analyst)...
    → claude -p "Write epic PRD for E10..." (with --disallowedTools)
    → Claude works... creates PRD docs...
    → Claude stops → Stop hook fires → shark status advance E10
    → Hook: Advanced E10 → ready_for_research
  Agent completed successfully.

--- Status: ready_for_research ---
  Action: spawn_agent | Agent: researcher | Provider: anthropic
  Advanced to in_research (work started)
  Dispatching anthropic agent (researcher)...
    → claude -p "Brownfield analysis for E10..." (with --disallowedTools)
    → Claude works... produces research report...
    → Claude stops → Stop hook fires → shark status advance E10
    → Hook: Advanced E10 → ready_for_design
  Agent completed successfully.

--- Status: ready_for_design ---
  Action: spawn_agent | Agent: architect | Provider: anthropic
  Advanced to in_design (work started)
  Dispatching anthropic agent (architect)...
    → claude -p "Architecture + UAT plan for E10..." (with --disallowedTools)
    → Claude stops → Stop hook fires → shark status advance E10
    → Hook: Advanced E10 → ready_for_decomposition
  Agent completed successfully.

--- Status: ready_for_decomposition ---
  Action: spawn_agent | Agent: product-manager | Provider: anthropic
  Advanced to in_decomposition (work started)
  Dispatching anthropic agent (product-manager)...
    → Claude stops → Stop hook fires → shark status advance E10
    → Hook: Advanced E10 → active
  Agent completed successfully.

--- Status: active ---
  Action: cascade
  Cascading to child entities...
  → Skipping E10-F01 (completed)
  → Skipping E10-F02 (completed)
  → Recursing into E10-F03 (active)
    [nested orchestration of E10-F03 runs here]
  → Skipping E10-F04 (completed)
  → Skipping E10-F05 (completed)
  → Recursing into E10-F06 (draft)
    [nested orchestration of E10-F06 runs here]
  All children complete. Advanced parent.

--- Status: completed ---
=== E10 reached terminal status: completed ===

=== Orchestration of E10 finished ===
```

### Failure & Resume

```
$ ./shark-orchestrate.sh E10

--- Status: ready_for_research ---
  Advanced to in_research (work started)
  Dispatching anthropic agent (researcher)...
    → claude -p "Brownfield analysis..."
    → Claude crashes (exit code 1)
    → Stop hook does NOT fire (session failed, not clean stop)
  ERROR: Agent exited non-zero. Status stays at in_research.
  Re-run to resume from in_research.

# Later, user re-runs:
$ ./shark-orchestrate.sh E10

--- Status: in_research ---
  Action: spawn_agent (resume instruction)
  # Already at in_*, no pre-advance needed
  Dispatching anthropic agent (researcher)...
    → claude -p "Resume research for E10, check for existing work..."
    → Claude works... finds partial report, completes it...
    → Claude stops → Stop hook fires → shark status advance E10
    → Hook: Advanced E10 → ready_for_design
  Agent completed successfully.

# Continues normally from ready_for_design...
```

### What Causes the Flow to Stop

| Condition | Behavior | How to Resume |
|-----------|----------|---------------|
| **Terminal status** (`completed`, `cancelled`) | Loop exits naturally | N/A — entity is done |
| **Pause action** (`pause`, `wait_for_triage`) | Loop breaks with message | Perform manual action, re-run `./shark-orchestrate.sh KEY` |
| **Agent failure** (non-zero exit) | Loop exits with error | Fix the issue, re-run (resumes from `in_*` status) |
| **No orchestrator action** for status | Loop exits with error | Add action to `.sharkconfig.json`, re-run |
| **Ctrl+C** (user interrupt) | Process killed, status stays at current | Re-run (resumes from persisted status) |
| **Codex agent failure** | Same as Claude failure | Same as Claude failure |

---

## Key Difference: Where Status Advancement Happens

### E22 `shark run` (Go Controller)

```
Go RunController
  ├── dispatch agent (os/exec)
  ├── wait for exit code
  ├── IF exit == 0: controller calls TransitionStatus()    ← Go code advances
  └── loop
```

The Go binary owns all advancement. The agent has zero control.

### Hooks Approach

```
Bash loop
  ├── launch claude -p (with SHARK_TASK_KEY env var)
  ├── Claude works...
  ├── Claude stops → Stop hook fires
  │     └── hook calls: shark status advance $SHARK_TASK_KEY    ← hook advances
  ├── claude process exits
  └── bash loop reads new status and continues
```

The **Stop hook** owns advancement for Claude agents. The **bash loop** owns advancement for non-Claude agents (Codex) and for `advance_status` actions.

---

## Hook vs Stop-Hook Interaction Detail

### Normal Flow (Agent Succeeds)

```
1. Bash loop sets SHARK_TASK_KEY=E10, launches claude -p "..."
2. Claude receives instruction, does work (edits files, runs tests, etc.)
3. Claude finishes responding → Stop event fires
4. .claude/hooks/advance-shark-task.sh runs:
   - Reads SHARK_TASK_KEY from env → "E10"
   - Reads stop_hook_active from stdin → false
   - Runs: shark status advance E10 --json
   - Prints to stderr: "Hook: Advanced E10 → ready_for_design"
   - Exits 0 (allows Claude to stop)
5. Claude session ends, process exits with code 0
6. Bash loop reads new status: ready_for_design
7. Loop continues
```

### Agent Crash (Non-Zero Exit)

```
1. Bash loop launches claude -p "..."
2. Claude encounters fatal error, session terminates abnormally
3. Stop hook may or may not fire depending on crash type:
   - Clean API error → StopFailure event (different from Stop, hook doesn't fire)
   - Tool timeout → Stop may fire
4. claude process exits with non-zero code
5. Bash loop catches non-zero exit → breaks with error
6. Status stays at in_* → resumable on next run
```

**Safety**: Even if the Stop hook fires on a partial failure and advances status prematurely, the bash loop detects the non-zero exit code and stops. The advancement is idempotent — the next status is just the next `in_*` or `ready_for_*`, which the resume logic handles.

### Non-Orchestrated Sessions (Normal Dev Work)

```
1. Developer runs: claude (interactive, no SHARK_TASK_KEY set)
2. Claude works on whatever the developer asks
3. Claude stops → Stop hook fires
4. Hook checks SHARK_TASK_KEY → empty
5. Hook exits 0 immediately (no-op)
6. Normal Claude behavior, zero impact
```

The hook is **inert** unless `SHARK_TASK_KEY` is explicitly set by the orchestration script.

---

## Comparison: Hooks vs `shark run`

| Dimension | `shark run` (E22) | Hooks + Bash |
|-----------|-------------------|--------------|
| **Lines of Go code** | ~800+ (controller, dispatchers, worktree, tests) | 0 |
| **Lines of bash** | 0 | ~120 (orchestrator + hook) |
| **Agent isolation** | `--disallowedTools` | `--disallowedTools` (same) |
| **Status advancement** | Go `TransitionStatus()` call | `shark status advance` CLI call from hook |
| **Resume on failure** | Re-run `shark run KEY` | Re-run `./shark-orchestrate.sh KEY` |
| **Parallel cascade** | Go goroutines + semaphore + worktrees | Possible with `&` + `wait`, but less controlled |
| **Multi-provider** | Go dispatcher interface | Bash case statement |
| **Structured logging** | Go logger with JSON output | Bash echo + stderr |
| **Testability** | Go unit tests with mock interfaces | Bash testing is harder |
| **Maintenance** | Go code in shark binary | External script + hook config |
| **Works for interactive sessions** | No (only via `shark run`) | Yes (hook fires for any Claude session with SHARK_TASK_KEY set) |
| **Dependency** | None (built into shark) | Requires `jq`, Claude CLI |

### When to Use Each

**Use hooks approach when:**
- You want minimal implementation effort
- You're primarily using Claude (not multi-provider)
- Sequential execution is sufficient
- You want status advancement to work in interactive sessions too
- You're comfortable with bash orchestration

**Use `shark run` (Go) when:**
- You need parallel cascade with worktree isolation
- You need structured JSON logging and audit trails
- You want the orchestrator built into the shark binary
- You need robust multi-provider dispatch
- You need comprehensive error handling and retry logic
- You want Go-level testability

---

## Implementation Checklist

### To implement hooks-based orchestration:

1. **Create hook script**: `.claude/hooks/advance-shark-task.sh`
   - Read `SHARK_TASK_KEY` from environment
   - Read `stop_hook_active` from stdin to prevent loops
   - Call `shark status advance` on the task key
   - Exit 0 always (never block Claude from stopping)

2. **Configure hook**: `.claude/settings.json`
   ```json
   {
     "hooks": {
       "Stop": [
         {
           "hooks": [
             {
               "type": "command",
               "command": ".claude/hooks/advance-shark-task.sh"
             }
           ]
         }
       ]
     }
   }
   ```

3. **Create orchestration script**: `scripts/shark-orchestrate.sh`
   - Read status → read action → dispatch or advance → loop
   - Set `SHARK_TASK_KEY` env var for Claude dispatches
   - Handle cascade via recursive calls
   - Handle pause/archive as loop exits

4. **Verify `--disallowedTools` works**: Test that Claude cannot self-advance when the flag is set.

5. **Test the flow end-to-end**: Drive a task from `draft` to `completed`.

---

## Open Questions

1. **Does the Stop hook fire reliably on `claude -p` (non-interactive) exits?** Need to verify that programmatic sessions trigger Stop the same as interactive ones.

2. **Does `SessionEnd` fire after `Stop`?** If so, should advancement happen in `SessionEnd` instead (guaranteed to fire even on crash)?

3. **Race condition**: If the Stop hook advances status, but the bash loop also checks status — is there a window where the loop reads the old status? Likely not (hook completes synchronously before `claude` exits), but worth verifying.

4. **Multi-entity hooks**: If two orchestration scripts run concurrently with different `SHARK_TASK_KEY` values, each Claude session inherits its own env. No conflict expected, but untested.

5. **Hook + `shark run` coexistence**: If someone uses `shark run` (Go controller) with the hook also configured, status would advance twice (once by hook, once by Go). The hook should detect `SHARK_TASK_KEY` absence and no-op. Or: `shark run` should unset `SHARK_TASK_KEY` explicitly.
