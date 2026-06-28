# Workflow: /plan-sprint

Mode-aware sprint planning workflow. Reads sprint state and backlog from shark, proposes entity assignments in interactive or auto mode, and reports the readiness-score delta on exit. Never calls `shark sprint start`.

---

## Step 0: Argument Parse

Parse `$ARGUMENTS`:

1. Extract the sprint key (first positional argument).
2. Validate: the key must match the pattern `S` followed by one to three digits (e.g., `S001`, `S42`, `S999`). Case-insensitive (accept `s001` as `S001`).
3. If the key does not match, stop immediately:
   ```
   /plan-sprint only operates on sprints. Got: {KEY}
   ```
   Do not call any shark commands.
4. Parse `--mode` flag: accept `interactive` (default) or `auto`. Any other value is an error — print usage and stop.
5. Parse `--max-add=N` (optional, default: unlimited). If provided, N must be a positive integer.

---

## Step 1: Read Sprint State

```bash
shark get {S###} --json
```

Parse the JSON response:
- `status` — the sprint's current lifecycle status
- `key` — canonical sprint key (use this for all subsequent calls)
- `capacity` — per-agent-type capacity values (if present at this level; otherwise read from plan in Step 2)

**If status is `planning`**: proceed silently.

**If status is NOT in `{planning, active}`**:
- Print an advisory:
  ```
  Note: Sprint {S###} is in status: {status}. Planning a {status} sprint is allowed but unusual.
  Continue? (yes/no)
  ```
- If user says **no**: exit cleanly. Do not call any further shark commands.
- If user says **yes**: continue to Step 2.

**If sprint is `active`**: treat the same as non-planning — show advisory, ask to continue.

---

## Step 2: Read Plan View

```bash
shark sprint plan {S###} --json
```

Store the full JSON response in `PLAN`. Do **not** call `shark sprint plan` again in this workflow body unless explicitly triggered by a user add/remove action in a new turn.

Parse from `PLAN`:
- `backlog` — list of entities eligible for assignment, each with: `key`, `title`, `size`, `agent_type`, `priority`, `feature_key` (if a task)
- `capacity_by_agent` — map of agent type → remaining capacity (in size points)
- `readiness_score` — the score at the start of this session (store as `INITIAL_READINESS`)
- `readiness_factors` — breakdown (unassigned, unsized, blocked, etc.)

If `backlog` is empty:
```
Sprint {S###} backlog is empty — no eligible entities to add.
Readiness score: {INITIAL_READINESS}
To execute: /run-sprint {S###}
```
Exit.

---

## Step 3: Branch on --mode

### 3a. Interactive Mode

1. Display the readiness summary:
   ```
   Sprint {S###} — Readiness: {INITIAL_READINESS}
   Capacity by agent type: {capacity_by_agent summary}
   Backlog: {count} eligible entities
   ```

2. Display any oversize or unsized entities as a warning:
   ```
   Warning: {N} entities are unsized (size=0 or missing) and excluded from capacity calculation.
   ```

3. Group backlog entities by `feature_key` (tasks with the same `E##-F##` parent), then a group for standalones (bugs, change-cards, tech-debt without a feature parent).

4. For each group, display the entities:
   ```
   Feature {feature_key} (or "Standalone entities"):
     • {KEY} — {title} [size={size}, agent={agent_type}, priority={priority}]
     • …
   Add all from this group? (yes/no/pick)
   ```
   - **yes** → confirm all entities in the group
   - **no** → skip all
   - **pick** → for each entity in the group, ask individually: `Add {KEY} — {title}? (yes/no)`

5. After each confirmed entity (or confirmed group), call:
   ```bash
   shark sprint add {S###} {KEY} --json
   ```
   Parse the response to confirm success. If the entity is already assigned, print a notice and skip (do not error).

6. Respect `--max-add=N`: once N entities have been added in this session, stop presenting more and note the limit was reached.

7. After all groups are processed, proceed to Step 4.

### 3b. Auto Mode

1. Display the readiness summary (same as 3a step 1).

2. Greedy fill algorithm:
   - Sort backlog by: `priority` descending, then `size` ascending (smaller entities that fit first).
   - For each agent type, maintain a running allocated total starting at 0.
   - Iterate through the sorted backlog:
     - If `capacity_by_agent[entity.agent_type] - allocated[entity.agent_type] >= entity.size`:
       - Mark entity as SELECTED.
       - Increment `allocated[entity.agent_type]` by `entity.size`.
     - Else: mark entity as SKIPPED (over capacity for that agent type).
   - If `entity.agent_type` is unknown or capacity data absent: mark as SKIPPED with a note.
   - Respect `--max-add=N` limit: stop selecting once N entities are marked SELECTED.

3. Display the proposed plan:
   ```
   Proposed plan for Sprint {S###}:
   SELECTED ({count} entities, {total_size} pts):
     • {KEY} — {title} [size={size}, agent={agent_type}]
     …
   SKIPPED ({count} entities — over capacity or no agent type):
     • {KEY} — {title} [size={size}, agent={agent_type}] — reason: {reason}
     …

   Capacity utilization by agent type:
     {agent_type}: {allocated}/{capacity} pts
     …

   Apply this plan? (yes/no)
   ```

4. **ONE confirmation gate**: wait for user response.
   - **no** → exit without calling any `shark sprint add`. Report: "Plan cancelled. Sprint {S###} unchanged."
   - **yes** → for each SELECTED entity (in sort order):
     ```bash
     shark sprint add {S###} {KEY} --json
     ```
     Parse response. If already assigned: print notice and continue. If error: print error, continue with remaining entities (do not abort entire batch on single failure).

5. Proceed to Step 4.

---

## Step 4: Re-Read Readiness

```bash
shark sprint readiness {S###} --json
```

Parse `readiness_score` as `FINAL_READINESS`.

Report the delta:
```
Readiness update for Sprint {S###}:
  Before: {INITIAL_READINESS}
  After:  {FINAL_READINESS}
  Delta:  {+/−N} points
```

If the delta is 0 (no entities added or readiness unchanged), say so explicitly:
```
  Delta:  0 (no new assignments made)
```

---

## Step 5: Exit

Print the completion message:
```
Sprint {S###} planning complete.
To execute this sprint: /run-sprint {S###}
```

**DO NOT call `shark sprint start`.** Starting a sprint is an explicit user action. The user must run `/run-sprint {S###}` (which will offer to start the sprint if needed).

---

## Error Handling

- `shark sprint plan` returns no backlog: handled in Step 2.
- `shark sprint add` returns an error for one entity: log the error, continue with the rest.
- `shark sprint add` returns "already assigned": print a notice (`{KEY} is already in sprint {S###} — skipping.`), do not count it against `--max-add`.
- Sprint key not found (`shark get` returns not-found): print `Sprint {S###} not found.` and exit.
- User interrupts (Ctrl-C) mid-interactive session: no partial state is committed for unconfirmed items because each `shark sprint add` is called immediately after per-item confirmation. Items confirmed before interrupt are already assigned.

---

## Idempotency

This workflow is safe to re-invoke:
- Already-assigned entities are skipped in both modes (the `shark sprint add` call returns a "already assigned" response or the plan view excludes them from the eligible backlog).
- The readiness delta is always reported relative to the start of the current invocation, not a historical baseline.
- The workflow does not track cross-session state; shark is the authoritative state store.

---

## Constraints

- All shark calls use `--json`.
- `shark sprint plan` is called exactly once at the start (Step 2). It is not re-called within the same workflow body.
- `shark sprint start` is never called.
- `shark sprint add` is only called after explicit user confirmation.
