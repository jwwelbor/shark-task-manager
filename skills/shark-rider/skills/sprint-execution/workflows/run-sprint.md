# Workflow: /shark-rider run-sprint

Solo sprint pull-loop. Calls the sprint-scoped `shark plan S###` selector to get the next entity, delegates to `/shark-rider run` for per-entity orchestration, loops until no dispatchable entity is selected or `--max-iterations` is reached, then prompts the user before closing the sprint. Never auto-closes.

---

## Step 0: Argument Parse

Parse `$ARGUMENTS`:

1. Extract the sprint key (first positional argument).
2. Validate: the key must match the pattern `S` followed by one to three digits (e.g., `S001`, `S42`, `S999`). Case-insensitive (accept `s001` as `S001`).
3. If the key does not match, stop immediately:
   ```
   /shark-rider run-sprint only operates on sprints. Got: {KEY}
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
Set `SPRINT_KEY = key` before continuing so later identity checks use the canonical form.

**If status is `planning`**:
Auto-start it without asking — the user has standing authorization for
this (confirmed 2026-09-05: asking here wasted hours waiting on a decision
that should be automatic). Call `shark sprint start {SPRINT_KEY} --json`.
Parse response. If the call fails, print the error and exit. Print a notice
either way:
```
Sprint {SPRINT_KEY} was in `planning` status; started automatically.
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
- `SELECTION_PAUSED = false`
- `SELECTED_ENTITY = null`

Loop:

```
while ITERATION < MAX_ITERATIONS:
    ITERATION += 1
    SELECTED_ENTITY = null

    NEXT_CMD = "shark plan {SPRINT_KEY} --json"
    if AGENT_FILTER is set:
        NEXT_CMD = "shark plan {SPRINT_KEY} --agent={AGENT_FILTER} --json"

    Execute NEXT_CMD.
    Parse JSON response as SPRINT_SELECTION.

    if the response is malformed or SPRINT_SELECTION.sprint_key is missing:
        print the raw response and report that sprint selection returned unexpected output
        stop the pull-loop; do not dispatch an entity

    if SPRINT_SELECTION.sprint_key != SPRINT_KEY:
        report the selected sprint key mismatch and stop the pull-loop without dispatching an entity

    if SPRINT_SELECTION.action == "pause":
        print "No dispatchable entity was selected; sprint drain was not verified."
        SELECTION_PAUSED = true
        break

    if SPRINT_SELECTION.action == "select_item":
        SELECTED_ENTITY = SPRINT_SELECTION.entity
    else if SPRINT_SELECTION.action == "parallel_candidates":
        if SPRINT_SELECTION.entities is empty:
            report unexpected output, include "sprint drain was not verified", and stop the pull-loop; do not dispatch an entity
        SELECTED_ENTITY = SPRINT_SELECTION.entities[0]  # first item in selector order
    else:
        report the unexpected selection action and stop the pull-loop; do not dispatch an entity

    if SELECTED_ENTITY is null:
        print "No dispatchable entity was selected; sprint drain was not verified."
        report null selected entity and stop the pull-loop; do not dispatch an entity

    if SELECTED_ENTITY.key is missing, empty, or whitespace-only:
        print "No dispatchable entity was selected; sprint drain was not verified."
        report the selected entity key is invalid and stop the pull-loop; do not dispatch an entity

    if SELECTED_ENTITY.sprint_key != SPRINT_KEY:
        SELECTED_ENTITY = null
        print "No dispatchable entity was selected; sprint drain was not verified."
        report the selected entity sprint key mismatch and stop the pull-loop without dispatching an entity

    ENTITY_KEY = SELECTED_ENTITY.key  # e.g., "E07-F01-001", "B003"

    Print progress:
    "[Sprint {SPRINT_KEY}] Iteration {ITERATION}/{MAX_ITERATIONS} — dispatching {ENTITY_KEY}"

    Append ENTITY_KEY to DISPATCHED.

    /shark-rider run {ENTITY_KEY}
    # Delegate entirely to the existing orchestration skill.
    # /shark-rider run reads orchestrator_action and drives the entity to terminal status.
    # When /shark-rider run returns, resume the loop.

if ITERATION >= MAX_ITERATIONS and SELECTED_ENTITY is not null:
    Print cap notice:
    "[Sprint {SPRINT_KEY}] Reached --max-iterations={MAX_ITERATIONS}. Loop stopped.
    Dispatched {len(DISPATCHED)} entities this session: {DISPATCHED}
    Sprint is still active. Re-run /shark-rider run-sprint {SPRINT_KEY} to continue."
    Proceed to Step 3 (still report and prompt; do not skip close gate).
```

**Loop exit conditions:**
1. `shark plan {SPRINT_KEY}` returns `action == "pause"` — no dispatchable entity was selected; sprint drain was not verified.
2. `ITERATION >= MAX_ITERATIONS` — cap reached; cap notice printed above.
3. The sprint selection response is malformed or names a different sprint — stop without dispatching an entity.

**On `/shark-rider run` failure (agent error or unexpected exit):** Print the error, add a note to the entity if possible, and continue the loop. Do not abort the sprint pull-loop on a single entity failure.

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
[Sprint {SPRINT_KEY}] Pull-loop stopped.
Entities dispatched this session ({len(DISPATCHED)}): {DISPATCHED joined by ", "}
```

If `SELECTION_PAUSED` is true, include this exact status in the summary:
```
No dispatchable entity was selected; sprint drain was not verified.
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
Sprint {SPRINT_KEY} left in active state. Run /shark-rider run-sprint {SPRINT_KEY} again to continue, or close manually with:
  shark sprint close {SPRINT_KEY}
```
Exit.

**If user says yes**: closing requires a submitted Sprint Goal Review first — a
close attempt with no accepted review returns the sprint to active and
creates no completion row. Collect the goal-review evidence from the user
before calling close:

```
Before closing, record the Sprint Goal Review:
  Declared executable goal:
  Observed result before the demonstration:
  Observed result after the demonstration:
  Reviewer identity:
  Outcome (accepted/rejected):
```

Submit the review:
```bash
shark sprint goal-review {SPRINT_KEY} --goal={GOAL} --before={BEFORE_RESULT} --after={AFTER_RESULT} --reviewer={REVIEWER} --outcome={OUTCOME} --json
```

If the outcome is `rejected` (or the user has no evidence to submit), do not
proceed to close — report that the sprint remains active pending an accepted
goal review, and exit.

Only after a submitted review with `--outcome=accepted`, construct the close
command.

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

On error: print the full error message, including the case where close
reports no accepted goal review — in that case, the sprint remains active;
suggest re-running `shark sprint goal-review` with `--outcome=accepted`
before retrying close. Do not retry silently.

---

## Error Handling

| Situation | Behavior |
|---|---|
| Sprint key not found (`shark get` returns not-found) | Print `Sprint {SPRINT_KEY} not found.` and exit |
| `shark sprint start` fails | Print error; exit without entering the loop |
| `shark plan {SPRINT_KEY}` returns malformed JSON, an unknown action, or an invalid selection shape | Print the raw response when available, report `sprint selection returned unexpected output`, stop the loop, and proceed to Step 3 |
| Sprint selection or selected entity has a different `sprint_key` than `SPRINT_KEY` | Report the mismatch, do not dispatch the entity, stop the loop, and proceed to Step 3 |
| `/shark-rider run {ENTITY_KEY}` fails or the agent errors | Log error; continue loop (do not abort sprint) |
| `shark sprint close` fails | Print full error; sprint remains open; suggest manual retry |
| `shark sprint close` fails with no accepted goal review | Sprint remains active, no completion row created; submit `shark sprint goal-review {SPRINT_KEY} --outcome=accepted` (with goal/before/after/reviewer evidence) before retrying close |
| User provides `--max-iterations=0` or negative | Treat as invalid; print usage and exit |

---

## Idempotency

This workflow is safe to re-invoke:

- If the sprint is already `completed`/`archived`/`cancelled`, Step 1 exits with a notice — no double-close possible (TC-NF02).
- If re-invoked on an `active` sprint mid-execution, the loop asks `shark plan {SPRINT_KEY}` again and uses its first ordered candidate. DISPATCHED tracking is session-local and does not affect shark state.
- `shark sprint close` is only reachable through the explicit user-confirmation gate in Step 4 — not callable via re-entry without the user answering "yes" again.

---

## Constraints

- All shark calls use `--json`.
- `shark plan {SPRINT_KEY} --json` is the sprint-scoped source of "what to work on next". The loop does NOT read the backlog directly or pick entities itself; when it receives `parallel_candidates`, it consumes the first candidate in selector order for this solo iteration.
- `/shark-rider run {ENTITY_KEY}` is the sole dispatch mechanism. The loop does NOT call `shark status advance` or any other entity-manipulation command directly.
- `shark sprint close` is only called after explicit user confirmation in Step 4, and only after an accepted `shark sprint goal-review` submission for this close attempt.
- The loop honors `--max-iterations` and exits at the cap with a notice.
- `shark sprint start` is only called if status is `planning` AND the user explicitly confirms (Step 1).
