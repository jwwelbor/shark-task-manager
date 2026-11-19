# Workflow: /shark-rider run-sprint

Solo sprint pull-loop. Calls `shark sprint next` to select an eligible entity and delegates to `/shark-rider run` for per-entity orchestration. An empty selection is considered a drained backlog only after the full all-status assignment set is verified terminal under each entity's workflow. A role-filtered empty slice may leave other sprint assignments unfinished. The loop then prompts the user before any close; it never auto-closes.

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
   - `--agent=<type>` (default: not set — select across all agent types; an empty filtered slice is not proof that the sprint is drained)
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
- `LOOP_OUTCOME = "running"`

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
        # Empty means no candidate survived the selector's current filters.
        # Verify the whole sprint; omit AGENT_FILTER and all other selector filters.
        BACKLOG = execute `shark sprint backlog {SPRINT_KEY} --view=ordered --all --json`
        if the command fails or its JSON is malformed:
            LOOP_OUTCOME = "paused"
            print that terminality could not be verified; do not report drained
            break

        UNFINISHED = []
        TERMINALITY_UNVERIFIED = false
        WORKFLOW_BY_TYPE = {}
        For every distinct entity type in BACKLOG.items:
            run `/shark-rider workflow {ENTITY_TYPE} --json` once to resolve
            that type's workflow and explicitly configured terminal statuses
            if the workflow or its terminal statuses cannot be resolved:
                set TERMINALITY_UNVERIFIED = true
            else:
                save the resolved workflow in WORKFLOW_BY_TYPE[ENTITY_TYPE]

        For every item in BACKLOG.items, including terminal-status items:
            reuse that workflow metadata for every matching item
            if its workflow, current status, or terminal marker cannot be resolved:
                set TERMINALITY_UNVERIFIED = true
            else if its current status is not marked terminal by that workflow:
                add its key, entity type, and status to UNFINISHED

        if BACKLOG.total_count does not match the returned item count:
            LOOP_OUTCOME = "paused"
            print that the full assignment inventory is incomplete
            break

        if TERMINALITY_UNVERIFIED is true:
            LOOP_OUTCOME = "paused"
            print that the full assignment set's terminality could not be verified
            break

        if UNFINISHED is not empty:
            LOOP_OUTCOME = "paused"
            if AGENT_FILTER is set:
                report that the {AGENT_FILTER} role-filtered slice returned no
                candidate, but nonterminal assignments remain in the full sprint
            else:
                report that no candidate was returned, but nonterminal assignments
                remain in the full sprint
            list each unfinished key, entity type, and current status; label work
            held, filtered, or blocked only when supported by the available evidence
            break  # paused with unfinished assignments; not normal completion

        # A successful all-status read with no unfinished assignments proves
        # that every assigned entity is terminal (including an empty assignment set).
        LOOP_OUTCOME = "drained"
        break  # normal completion is allowed only after terminality is verified

    ENTITY_KEY = NEXT_ENTITY.key  # e.g., "E07-F01-001", "B003"

    Print progress:
    "[Sprint {SPRINT_KEY}] Iteration {ITERATION}/{MAX_ITERATIONS} — dispatching {ENTITY_KEY}"

    Append ENTITY_KEY to DISPATCHED.

    /shark-rider run {ENTITY_KEY}
    # Delegate entirely to the existing orchestration skill.
    # /shark-rider run reads orchestrator_action and drives the entity to terminal status.
    # When /shark-rider run returns, resume the loop.

if ITERATION >= MAX_ITERATIONS and NEXT_ENTITY was not empty:
    LOOP_OUTCOME = "capped"
    Print cap notice:
    "[Sprint {SPRINT_KEY}] Reached --max-iterations={MAX_ITERATIONS}. Loop stopped.
    Dispatched {len(DISPATCHED)} entities this session: {DISPATCHED}
    Sprint is still active. Re-run /shark-rider run-sprint {SPRINT_KEY} to continue."
    Proceed to Step 3 (still report and prompt; do not skip close gate).
```

**Loop exit conditions:**
1. `shark sprint next` returns empty and the unfiltered all-status assignment check confirms every assigned entity is terminal — normal completion; backlog drained.
2. `shark sprint next` returns empty but unfinished assignments remain, or terminality cannot be verified — paused; report the held/filtered/blocked items and do not call it normal completion.
3. `ITERATION >= MAX_ITERATIONS` — cap reached; cap notice printed above.

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

Print a session summary that matches `LOOP_OUTCOME`:
- `drained`: `[Sprint {SPRINT_KEY}] Pull-loop drained; every assigned entity is terminal.`
- `paused`: `[Sprint {SPRINT_KEY}] Pull-loop paused; an empty selector result did not prove the sprint drained.` Include the unfinished keys/statuses or the reason terminality could not be verified.
- `capped`: report the cap and that the sprint remains active, as above.

In every case, include the number and keys of entities dispatched this session.

---

## Step 4: Close Gate

**THIS STEP REQUIRES EXPLICIT USER CONFIRMATION. Do NOT skip this step under any circumstances.**

If `LOOP_OUTCOME` is `paused` or `capped`, state that the sprint is not
verified drained. List assignment keys and statuses only when they come from a
complete, successful all-status read. If that inventory is unavailable or
incomplete, say so and do not imply the full remaining set is known. A close
with carryover may reassign or remove incomplete assignments according to the
selected carryover mode. The owner must still explicitly confirm, and the
accepted Sprint Goal Review below remains a separate required gate.

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
| `shark sprint next` returns a malformed JSON response | Print the raw response, report `shark sprint next returned unexpected output`, set `LOOP_OUTCOME = "paused"`, and proceed to Step 3 without calling the backlog drained |
| All-status sprint backlog read fails or is malformed after an empty selection | Set `LOOP_OUTCOME = "paused"`; report that terminality could not be verified and do not call the backlog drained |
| An assigned entity's status or configured terminal marker cannot be resolved | Set `LOOP_OUTCOME = "paused"`; report the unresolved key and do not count it as terminal |
| `/shark-rider run {ENTITY_KEY}` fails or the agent errors | Log error; continue loop (do not abort sprint) |
| `shark sprint close` fails | Print full error; sprint remains open; suggest manual retry |
| `shark sprint close` fails with no accepted goal review | Sprint remains active, no completion row created; submit `shark sprint goal-review {SPRINT_KEY} --outcome=accepted` (with goal/before/after/reviewer evidence) before retrying close |
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
- `shark sprint next` is the sole source of entities to dispatch. After an empty result, `shark sprint backlog {SPRINT_KEY} --view=ordered --all --json` is a read-only whole-sprint terminality check only; it is never used to choose an entity. That check omits `AGENT_FILTER` and other selector filters.
- `/shark-rider run {ENTITY_KEY}` is the sole dispatch mechanism. The loop does NOT call `shark status advance` or any other entity-manipulation command directly.
- `shark sprint close` is only called after explicit user confirmation in Step 4, and only after an accepted `shark sprint goal-review` submission for this close attempt.
- The loop honors `--max-iterations` and exits at the cap with a notice.
- `shark sprint start` is only called if status is `planning` AND the user explicitly confirms (Step 1).
