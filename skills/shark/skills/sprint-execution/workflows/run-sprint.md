# Workflow: /run-sprint

Solo sprint pull-loop. Calls `shark sprint next` to get the next entity, delegates to `/run` for per-entity orchestration, loops until the backlog is drained or `--max-iterations` is reached, then prompts the user before closing the sprint. Never auto-closes.

---

## Step 0: Argument Parse

Parse `$ARGUMENTS`:

1. Extract the sprint key (first positional argument).
2. Validate: the key must match the pattern `S` followed by one to three digits (e.g., `S001`, `S42`, `S999`). Case-insensitive (accept `s001` as `S001`).
3. If the key does not match, stop immediately:
   ```
   /run-sprint only operates on sprints. Got: {KEY}
   ```
   Do not call any shark commands.
4. Parse optional flags:
   - `--agent=<type>` (default: not set — pull all agent types)
   - `--max-iterations=N` (default: 50; must be a positive integer if provided)
   - `--carryover=<value>` (default: not set — omit from `shark sprint close` if absent)

Store as `SPRINT_KEY`, `AGENT_FILTER`, `MAX_ITERATIONS`, `CARRYOVER_VALUE`.

---

## Step 1: Sprint Status Check

```bash
shark get {SPRINT_KEY} --json
```

Parse the JSON response. Extract:
- `status` — the sprint's current lifecycle status
- `key` — canonical sprint key (use this for all subsequent calls)

**If status is `planning`**:
```
Sprint {SPRINT_KEY} is in `planning` status and has not been started yet.
Call `shark sprint start {SPRINT_KEY}` to start it now? (yes/no)
```
- **yes** → call `shark sprint start {SPRINT_KEY} --json`. Parse response. If the call fails, print the error and exit.
- **no** → exit cleanly:
  ```
  Exiting. Start the sprint first, then re-run /run-sprint {SPRINT_KEY}.
  ```

**If status is `completed`, `archived`, or `cancelled`**:
```
Sprint {SPRINT_KEY} is {status}. Nothing to run.
```
Exit cleanly. Do NOT call `shark sprint close` or any other mutating command.

**If status is `active`**: proceed to Step 2.

**If status is any other value**: print a warning and ask the user whether to continue:
```
Sprint {SPRINT_KEY} is in an unexpected status: {status}.
Continue with the pull-loop anyway? (yes/no)
```
- **no** → exit.
- **yes** → proceed to Step 2.

---

## Step 2: Pull-Loop

Initialize:
- `ITERATION = 0`
- `DISPATCHED = []` (list of entity keys dispatched this session)

Loop:

```
while ITERATION < MAX_ITERATIONS:
    ITERATION += 1

    NEXT_CMD = "shark sprint next {SPRINT_KEY} --json"
    if AGENT_FILTER is set:
        NEXT_CMD = "shark sprint next {SPRINT_KEY} --agent={AGENT_FILTER} --json"

    Execute NEXT_CMD.
    Parse JSON response as NEXT_ENTITY.

    if NEXT_ENTITY is null or empty (no entity returned):
        break  # backlog drained; exit loop normally

    ENTITY_KEY = NEXT_ENTITY.key  # e.g., "E07-F01-001", "B003"

    Print progress:
    "[Sprint {SPRINT_KEY}] Iteration {ITERATION}/{MAX_ITERATIONS} — dispatching {ENTITY_KEY}"

    Append ENTITY_KEY to DISPATCHED.

    /run {ENTITY_KEY}
    # Delegate entirely to the existing orchestration skill.
    # /run reads orchestrator_action and drives the entity to terminal status.
    # When /run returns, resume the loop.

if ITERATION >= MAX_ITERATIONS and NEXT_ENTITY was not empty:
    Print cap notice:
    "[Sprint {SPRINT_KEY}] Reached --max-iterations={MAX_ITERATIONS}. Loop stopped.
    Dispatched {len(DISPATCHED)} entities this session: {DISPATCHED}
    Sprint is still active. Re-run /run-sprint {SPRINT_KEY} to continue."
    Proceed to Step 3 (still report and prompt; do not skip close gate).
```

**Loop exit conditions:**
1. `shark sprint next` returns empty — normal completion; backlog drained.
2. `ITERATION >= MAX_ITERATIONS` — cap reached; cap notice printed above.

**On `/run` failure (agent error or unexpected exit):** Print the error, add a note to the entity if possible, and continue the loop. Do not abort the sprint pull-loop on a single entity failure.

---

## Step 3: Post-Loop Reporting

Print sprint progress reports:

```bash
shark sprint burndown {SPRINT_KEY} --json
```

Display the burndown output (remaining vs. completed entity count and size). If the command is unavailable or returns an error, print a notice and continue.

```bash
shark sprint summary {SPRINT_KEY} --json
```

Display the summary output (completed, in-progress, incomplete entity counts). Basic summary only — do not pass `--detailed`.

Print a session summary:
```
[Sprint {SPRINT_KEY}] Pull-loop complete.
Entities dispatched this session ({len(DISPATCHED)}): {DISPATCHED joined by ", "}
```

---

## Step 4: Close Gate

**THIS STEP REQUIRES EXPLICIT USER CONFIRMATION. Do NOT skip this step under any circumstances.**

Ask the user:
```
Close sprint {SPRINT_KEY} now? (yes/no)
Carryover strategy: {CARRYOVER_VALUE if set, else "default (per sharkconfig)"}
```

**If user says no (or anything other than explicit "yes")**:
```
Sprint {SPRINT_KEY} left in active state. Run /run-sprint {SPRINT_KEY} again to continue, or close manually with:
  shark sprint close {SPRINT_KEY}
```
Exit.

**If user says yes**: construct the close command.

If `CARRYOVER_VALUE` is set:
```bash
shark sprint close {SPRINT_KEY} --carryover={CARRYOVER_VALUE} --json
```

If `CARRYOVER_VALUE` is NOT set:
```bash
shark sprint close {SPRINT_KEY} --json
```

Parse the response. On success:
```
Sprint {SPRINT_KEY} closed successfully.
```

On error: print the full error message. Do not retry silently.

---

## Error Handling

| Situation | Behavior |
|---|---|
| Sprint key not found (`shark get` returns not-found) | Print `Sprint {SPRINT_KEY} not found.` and exit |
| `shark sprint start` fails | Print error; exit without entering the loop |
| `shark sprint next` returns a malformed JSON response | Print the raw response, report `shark sprint next returned unexpected output`, exit loop and proceed to Step 3 |
| `/run {ENTITY_KEY}` fails or the agent errors | Log error; continue loop (do not abort sprint) |
| `shark sprint close` fails | Print full error; sprint remains open; suggest manual retry |
| User provides `--max-iterations=0` or negative | Treat as invalid; print usage and exit |

---

## Idempotency

This workflow is safe to re-invoke:

- If the sprint is already `completed`/`archived`/`cancelled`, Step 1 exits with a notice — no double-close possible (TC-NF02).
- If re-invoked on an `active` sprint mid-execution, the loop picks up where shark left off: `shark sprint next` returns the next un-dispatched entity regardless of what this session previously dispatched. DISPATCHED tracking is session-local and does not affect shark state.
- `shark sprint close` is only reachable through the explicit user-confirmation gate in Step 4 — not callable via re-entry without the user answering "yes" again.

---

## Constraints

- All shark calls use `--json`.
- `shark sprint next` is the sole source of "what to work on next". The loop does NOT read the backlog directly or pick entities itself.
- `/run {ENTITY_KEY}` is the sole dispatch mechanism. The loop does NOT call `shark status advance` or any other entity-manipulation command directly.
- `shark sprint close` is only called after explicit user confirmation in Step 4.
- The loop honors `--max-iterations` and exits at the cap with a notice.
- `shark sprint start` is only called if status is `planning` AND the user explicitly confirms (Step 1).
