# Hooks-Based Status Advancement — Preventing Skipped Statuses

**Purpose**: Use Claude Code hooks to prevent Claude from skipping workflow statuses during interactive sessions. Claude does the work; hooks own the advancement.

**Date**: 2026-03-24

---

## The Problem

During interactive sessions, Claude calls `shark status advance` multiple times in rapid succession, blasting through statuses without performing actual work at each stage. Prompt-level guards ("never advance more than once") are fundamentally unenforceable because Claude controls its own enforcement.

**What we want**: Claude does the work for a status, then the *system* advances to the next status — not Claude.

---

## Core Mechanism: Intercept + Defer

Two hooks working together:

1. **PreToolUse hook** — intercepts Claude's attempts to call `shark status advance` (and variants). Captures which entity Claude wants to advance, stores the intent in a session-scoped file, and **blocks the command** with a message telling Claude the system handles advancement.

2. **Stop hook** — fires when Claude finishes responding. Reads the stored intent file, executes the deferred `shark status advance`, and cleans up.

**Result**: Claude's natural behavior (trying to advance when done) creates the signal. The hooks capture that signal and defer execution to when Claude actually stops working. Claude cannot skip statuses because it never executes the advance command directly.

---

## How It Works: Step by Step

### Normal Interactive Session

```
1. User: "Work on E07-F01-001, it's in in_development"

2. Claude reads the task, writes code, runs tests, etc.

3. Claude decides it's done and tries:
   Bash: shark status advance E07-F01-001

4. PreToolUse hook intercepts:
   - Parses command → extracts key "E07-F01-001"
   - Writes to /tmp/shark-deferred-{session_id}.json:
     {"key": "E07-F01-001", "action": "advance"}
   - Returns: BLOCK with reason
     "Status advancement is managed by the system.
      Your work will be reviewed and status advanced when complete."

5. Claude sees the block message, understands, wraps up its response.

6. Claude stops → Stop hook fires:
   - Reads session_id from stdin JSON
   - Reads /tmp/shark-deferred-{session_id}.json → key = "E07-F01-001"
   - Executes: shark status advance E07-F01-001 --json
   - Logs: "Advanced E07-F01-001 → ready_for_code_review"
   - Deletes the deferred file
   - Exits 0

7. User sees Claude's work + hook's log message.
   Status has advanced exactly once.
```

### Claude Tries to Skip (Multiple Advances)

```
1. Claude tries: shark status advance E07-F01-001
   → PreToolUse hook: BLOCKED. Stored intent for E07-F01-001.

2. Claude tries again: shark status advance E07-F01-001
   → PreToolUse hook: BLOCKED again. Overwrites stored intent (same key, idempotent).

3. Claude tries to advance a SECOND time to skip ahead:
   shark status advance E07-F01-001
   → PreToolUse hook: BLOCKED. Still the same key stored.

4. Claude stops → Stop hook fires:
   → Reads deferred file → advances E07-F01-001 ONCE.
   → One status transition. No skipping.
```

**Key property**: No matter how many times Claude attempts to advance within a session turn, only **one** advance executes — when Claude actually stops.

### Session Where Claude Doesn't Try to Advance

```
1. User: "Fix the formatting in display_service.go"

2. Claude edits the file, never tries to advance any status.

3. Claude stops → Stop hook fires:
   → No deferred file exists for this session_id
   → No-op. Exit 0.

4. Normal session, zero impact.
```

### Concurrent Sessions

```
Session A (session_id: "aaa"):
  → Claude works on E07-F01-001
  → PreToolUse stores to /tmp/shark-deferred-aaa.json
  → Stop hook reads /tmp/shark-deferred-aaa.json → advances E07-F01-001

Session B (session_id: "bbb"):
  → Claude works on E10-F03-002
  → PreToolUse stores to /tmp/shark-deferred-bbb.json
  → Stop hook reads /tmp/shark-deferred-bbb.json → advances E10-F03-002

No conflict — each session has its own deferred file keyed by session_id.
```

---

## The Two Hook Scripts

### 1. PreToolUse Hook — `.claude/hooks/intercept-status-advance.sh`

Intercepts any Bash command that would advance, set, or transition status forward.

```bash
#!/bin/bash
# .claude/hooks/intercept-status-advance.sh
#
# Intercepts Claude's attempts to advance shark status.
# Captures the intent and blocks the command.
# The Stop hook will execute the deferred advance.

INPUT=$(cat)

# Extract tool info
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // ""')
TOOL_INPUT=$(echo "$INPUT" | jq -r '.tool_input // {}')

# Only intercept Bash tool calls
if [ "$TOOL_NAME" != "Bash" ]; then
  exit 0  # Allow non-Bash tools
fi

# Extract the command being run
COMMAND=$(echo "$TOOL_INPUT" | jq -r '.command // ""')

# Check if this is a forward status-advance command
# Only intercept advance/next-status (always forward). Let "set" through —
# it's used for legitimate backward transitions (rejections) and the
# workflow service validates transitions anyway.
if ! echo "$COMMAND" | grep -qE '(shark\s+(status\s+advance|task\s+next-status|feature\s+next-status|epic\s+next-status))'; then
  exit 0  # Not a forward status command, allow it
fi

# Extract the entity key from the command
# Handles patterns like:
#   shark status advance E07-F01-001
#   shark status set E07-F01-001 in_development
#   shark task next-status E07-F01-001
ENTITY_KEY=$(echo "$COMMAND" | grep -oE '(E[0-9]{2}(-F[0-9]{2}(-[0-9]{3})?)?|B[0-9]{3}|CC-[0-9]{3})' | head -1)

if [ -z "$ENTITY_KEY" ]; then
  exit 0  # Can't parse key, allow (shouldn't happen)
fi

# Store the deferred advance intent, keyed by session_id
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')
DEFERRED_DIR="/tmp/shark-deferred"
mkdir -p "$DEFERRED_DIR"

# Write the intent
cat > "${DEFERRED_DIR}/${SESSION_ID}.json" <<INTENT_EOF
{
  "key": "$ENTITY_KEY",
  "original_command": $(echo "$COMMAND" | jq -Rs .),
  "timestamp": "$(date -Iseconds)"
}
INTENT_EOF

# Block the command — output JSON to stdout
cat <<'BLOCK_EOF'
{
  "decision": "block",
  "reason": "Status advancement is managed by the system. Your work will be validated and status advanced automatically when you finish. Continue with your current task or let me know you're done."
}
BLOCK_EOF

exit 2  # Exit code 2 = block the tool call
```

### 2. Stop Hook — `.claude/hooks/advance-on-stop.sh`

Fires when Claude finishes. Executes any deferred status advance.

```bash
#!/bin/bash
# .claude/hooks/advance-on-stop.sh
#
# Executes deferred status advancement when Claude finishes working.
# Reads the intent stored by intercept-status-advance.sh.

INPUT=$(cat)

SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')
DEFERRED_FILE="/tmp/shark-deferred/${SESSION_ID}.json"

# No deferred advance for this session → no-op
if [ ! -f "$DEFERRED_FILE" ]; then
  exit 0
fi

# Read the deferred intent
ENTITY_KEY=$(jq -r '.key' "$DEFERRED_FILE")

if [ -z "$ENTITY_KEY" ] || [ "$ENTITY_KEY" = "null" ]; then
  rm -f "$DEFERRED_FILE"
  exit 0
fi

# Determine working directory
CWD=$(echo "$INPUT" | jq -r '.cwd // "."')
cd "$CWD" 2>/dev/null || true

# Execute the deferred advance
RESULT=$(shark status advance "$ENTITY_KEY" --json 2>&1) && {
  NEW_STATUS=$(echo "$RESULT" | jq -r '.to_status // .status // "unknown"')
  echo "Hook: Advanced $ENTITY_KEY → $NEW_STATUS" >&2
} || {
  echo "Hook: Failed to advance $ENTITY_KEY: $RESULT" >&2
  # Don't block Claude from stopping — just log the failure
}

# Clean up
rm -f "$DEFERRED_FILE"

exit 0  # Always allow Claude to stop
```

---

## Hook Configuration

### `.claude/settings.json`

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/intercept-status-advance.sh"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": ".claude/hooks/advance-on-stop.sh"
          }
        ]
      }
    ]
  }
}
```

**Note**: The PreToolUse matcher is set to `"Bash"` — it only fires for Bash tool calls, not for Read, Edit, Write, etc. The hook script itself does the fine-grained filtering (only status-advance commands).

---

## What Causes Status to Advance

| Scenario | What Happens |
|----------|-------------|
| Claude tries `shark status advance` | PreToolUse blocks it. Stop hook advances once when Claude finishes. |
| Claude tries `shark status set X in_development` | Same — intercepted and deferred. |
| Claude tries multiple advances in one turn | All blocked. Only one advance on Stop. Last key wins. |
| Claude never tries to advance | Stop hook finds no deferred file. No-op. |
| User runs `! shark status advance` | Runs directly in shell, bypasses hooks entirely. Works as normal. |
| User manually types advance in chat | Claude executes it → intercepted by hook. Same deferred behavior. |

## What Causes the Flow to Stop

| Condition | Behavior |
|-----------|----------|
| Claude finishes its response naturally | Stop hook fires, advances once if deferred intent exists |
| Claude encounters an error and gives up | Stop hook still fires (clean stop), advances if intent exists |
| Session crashes / API error | StopFailure event — Stop hook does NOT fire, no advance, status stays put |
| User hits Ctrl+C | Session killed, deferred file remains (stale), no advance |
| Terminal status reached | `shark status advance` in Stop hook returns error (already terminal), logged and ignored |

---

## Interaction with `/run` and `shark run`

### With `/run` skill (current LLM orchestration)

The hooks make `/run` safer immediately:
- `/run` dispatches agents that try to advance status → hooks intercept
- Each agent turn advances at most once
- The LLM orchestrator can still read status, dispatch agents, etc. — it just can't skip statuses

This is a **drop-in safety net** that doesn't require changing how `/run` works.

### With `shark run` (Go controller from E22)

If `shark run` is also implemented:
- `shark run` advances status via Go service calls (not CLI), so hooks don't interfere
- Claude agents dispatched by `shark run` have `--disallowedTools` AND hooks as defense in depth
- The hooks provide an extra safety layer even if `--disallowedTools` is misconfigured

### Standalone (no orchestration)

For plain interactive sessions ("work on task E07-F01-001"):
- Claude does the work, tries to advance when done
- Hooks capture and defer — one clean advance per turn
- No orchestration script needed for single-status work

---

## Limitations and Edge Cases

### 1. Claude might not try to advance at all

If Claude finishes work but doesn't attempt `shark status advance`, the Stop hook has no deferred intent and does nothing. The status stays put.

**Mitigation**: Instruct Claude (in CLAUDE.md or task instructions) to always call `shark status advance` when done with a status. The hook handles the rest.

### 2. Wrong entity key

If Claude tries to advance a different task than the one it's supposed to be working on, the hook defers that wrong key.

**Mitigation**: The PreToolUse hook could validate the key against a "currently assigned" task (if tracked elsewhere). For now, Claude's own context about which task it's working on is the source of truth.

### 3. Backward transitions (rejections)

The hook should NOT block backward transitions like `shark status set E07-F01-001 changes_requested --reason "..."`. These are valid rejection flows.

**Current behavior**: The regex in `intercept-status-advance.sh` only intercepts `advance` and `next-status` commands (which are always forward). `shark status set` is allowed through — it's used for legitimate backward transitions (rejections) and the workflow service validates the transition anyway.

### 4. Stale deferred files

If a session crashes and `/tmp/shark-deferred/{session_id}.json` isn't cleaned up, it persists. A new session with a different ID won't be affected, but the file lingers.

**Mitigation**: Add a `SessionEnd` hook that cleans up, or use a cron job to delete files older than 24 hours:
```bash
find /tmp/shark-deferred -name "*.json" -mtime +1 -delete
```

### 5. The Stop hook fires per response, not per session

If the user has a multi-turn conversation and Claude tries to advance in turn 3, the Stop hook fires at the end of turn 3 and advances. If Claude tries to advance again in turn 5, it advances again at the end of turn 5.

**This is actually correct behavior** — each turn of work should produce at most one advancement. Multi-turn sessions that span multiple statuses work naturally.

---

## Comparison to E22 `shark run`

| Concern | Hooks Approach | `shark run` (Go) |
|---------|---------------|------------------|
| **Prevents status skipping** | Yes — PreToolUse blocks, Stop defers | Yes — Go binary owns all advancement |
| **Works in interactive sessions** | Yes — the whole point | No — only in automated runs |
| **Implementation effort** | ~80 lines of bash, hook config | ~800+ lines of Go |
| **Concurrent sessions** | Session-scoped files, no conflicts | N/A (one entity per run) |
| **Multi-provider dispatch** | Not applicable (interactive) | Yes (Claude + Codex) |
| **Automated full-workflow runs** | Not the goal (use with `/run` or manual) | Yes — primary purpose |
| **Defense in depth** | Can layer WITH `shark run` | Can layer WITH hooks |

**These are complementary, not competing.** Hooks solve the interactive-session problem. `shark run` solves the automated-orchestration problem. Both can coexist.

---

## Implementation Checklist

- [ ] Create `.claude/hooks/intercept-status-advance.sh` — PreToolUse hook
- [ ] Create `.claude/hooks/advance-on-stop.sh` — Stop hook
- [ ] Add hook configuration to `.claude/settings.json`
- [ ] Make both scripts executable (`chmod +x`)
- [ ] Refine regex to only intercept `advance`/`next-status` (not `set` for backward transitions)
- [ ] Test: Claude tries to advance → blocked, deferred, executed on stop
- [ ] Test: Claude tries multiple advances → only one executes
- [ ] Test: Concurrent sessions with different tasks → no cross-contamination
- [ ] Test: Normal session without advance → no impact
- [ ] Add `SessionEnd` cleanup hook for stale deferred files
- [ ] Update CLAUDE.md to instruct Claude to call `shark status advance` when done (hooks handle the rest)

---

## Open Questions

1. **Does PreToolUse receive the full Bash command string?** Need to verify the exact shape of `tool_input` for Bash calls — is it `{"command": "shark status advance E07-F01-001"}` or something else?

2. **Does the Stop hook fire after a blocked PreToolUse?** If Claude's only action was the blocked advance and it has nothing else to do, does it stop (triggering the Stop hook)? Or does it try to find other work?

3. **Should the Stop hook block (force continue) if Claude tried to advance but hasn't done enough work?** This would be a quality gate — e.g., check if Claude made any file edits before allowing the advance. Adds complexity but prevents empty advances.

4. **PreToolUse hook latency**: The hook runs on every Bash tool call. If it adds noticeable latency, it could slow down normal development. The grep + jq pipeline should be fast (<50ms), but worth measuring.
