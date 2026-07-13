# Workflow: /run-sprint-team

Team sprint execution. Reads the sprint's backlog, groups assigned entities by feature key, dispatches each feature group sequentially via `/run-agent-team` (one team at a time — the agent-teams primitive constraint), and falls back to plain `/run` for standalone entities (bugs, change-cards, tech-debt without a feature parent). Prints a burndown between each feature group. Prompts the user before closing the sprint. Never auto-closes.

---

## Step 0: Argument Parse

Parse `$ARGUMENTS`:

1. Extract the sprint key (first positional argument).
2. Validate: the key must match the pattern `S` followed by one to three digits (e.g., `S001`, `S42`, `S999`). Case-insensitive (accept `s001` as `S001`).
3. If the key does not match, stop immediately:
   ```
   /run-sprint-team only operates on sprints. Got: {KEY}
   ```
   Do not call any shark commands.
4. Parse optional flags:
   - `--size=N` (default: not set — passed through to each `/run-agent-team` invocation as-is)
   - `--features=E##-F##[,E##-F##,...]` (default: not set — dispatch all feature groups)
   - `--carryover=<value>` (default: not set — omit from `shark sprint close` if absent)

Store as `SPRINT_KEY`, `TEAM_SIZE`, `FEATURE_FILTER` (parsed as a set of feature keys), `CARRYOVER_VALUE`.

---

## Step 1: Preconditions (inherited from `/run-agent-team`)

Run all preconditions from the run-agent-team orchestration workflow (Preconditions section), in order, before doing any sprint or entity work:

1. **Env var enabled.** Verify `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` is set in your Claude Code settings (`env` block) or shell environment. If missing, instruct the user to add it and restart Claude Code. **Abort.**
2. **Version.** `claude --version` must be ≥ `2.1.32`. If older, **abort.**
3. **Branch.** `git branch --show-current` — on `main`/`master` or unrelated branch, **ask the user** before continuing. If user declines, **abort.**
4. **Worktree clean.** `.git/MERGE_HEAD` must not exist. `git status --porcelain` should be empty or only contain expected work-in-progress. If dirty in unexpected ways, **abort.**
5. **No existing team in this session.** If a team is already active, ask the user to clean it up (`Clean up the team`) before continuing. **Abort** if they do not.

If any precondition fails, print the failure reason and stop. **Do not dispatch any entity — not even standalones — if preconditions fail.**

---

## Step 2: Sprint Status Check

```bash
/shark-rider query: get {SPRINT_KEY} --json
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
  Exiting. Start the sprint first, then re-run /run-sprint-team {SPRINT_KEY}.
  ```

**If status is `completed`, `archived`, or `cancelled`**:
```
Sprint {SPRINT_KEY} is {status}. Nothing to run.
```
Exit cleanly. Do NOT call `shark sprint close` or any other mutating command.

**If status is `active`**: proceed to Step 3.

**If status is any other value**: print a warning and ask the user whether to continue:
```
Sprint {SPRINT_KEY} is in an unexpected status: {status}.
Continue with team dispatch anyway? (yes/no)
```
- **no** → exit.
- **yes** → proceed to Step 3.

---

## Step 3: Group Entities by Feature

```bash
shark sprint backlog {SPRINT_KEY} --json
```

Parse the JSON response as `BACKLOG` (a list of entity objects). If the response is empty or null, print:
```
Sprint {SPRINT_KEY} has no assigned entities. Nothing to dispatch.
```
Proceed directly to Step 6 (post-loop reporting).

For each entity in `BACKLOG`, classify it:

**Feature-grouped entities (tasks):**
- Tasks have keys matching `E##-F##-###` (e.g., `E07-F01-003`).
- Extract the feature key by taking the first two segments: `E##-F##`.
- Add to `FEATURE_GROUPS[feature_key]`.

**Standalone entities:**
- Bugs (`B###`), change-cards (`CC-###`), and tech-debt items (`TD-###`) that do NOT have a feature parent form a single `STANDALONE` list.
- Any other entity without an extractable `E##-F##` prefix also goes to `STANDALONE`.

After classification, build:
- `FEATURE_GROUPS`: dict of feature_key → list of entity objects
- `STANDALONE`: list of entity objects

**Apply `--features` filter (if `FEATURE_FILTER` is set):**
- Retain only feature keys that appear in `FEATURE_FILTER`.
- Remove all other feature groups from `FEATURE_GROUPS`.
- If `FEATURE_FILTER` is set, also **exclude** `STANDALONE` entities UNLESS the filter explicitly contains the string `standalone`.
- If the filter reduces `FEATURE_GROUPS` to empty and standalones are excluded, print:
  ```
  --features filter matched no feature groups in sprint {SPRINT_KEY}. Nothing to dispatch.
  ```
  Proceed to Step 6.

**Sort `FEATURE_GROUPS`** by epic number ascending, then by feature number ascending (e.g., E07-F01 before E07-F02 before E08-F01).

Print a dispatch plan:
```
[Sprint {SPRINT_KEY}] Dispatch plan:
  Feature groups ({N}): {feature_keys joined by ", "}
  Standalones ({M}): {standalone_entity_keys joined by ", "}
  Team size: {TEAM_SIZE if set, else "default (3)"}
Proceeding with sequential feature dispatch.
```

---

## Step 4: Sequential Per-Feature Dispatch

**CONSTRAINT: Only one `/run-agent-team` invocation may be active at any time. Always wait for a `/run-agent-team` invocation to return fully before starting the next one.**

For each `FEATURE_KEY` in the sorted feature groups:

1. Print:
   ```
   [Sprint {SPRINT_KEY}] Dispatching feature group {FEATURE_KEY} via /run-agent-team...
   ```

2. Invoke `/run-agent-team`:
   - If `TEAM_SIZE` is set:
     ```
     /run-agent-team {FEATURE_KEY} --size={TEAM_SIZE}
     ```
   - If `TEAM_SIZE` is not set:
     ```
     /run-agent-team {FEATURE_KEY}
     ```

3. **Wait for `/run-agent-team` to return.** Do not proceed until it has completed (all feature tasks terminal, team cleaned up).

4. On return, print a burndown (progress check between groups):
   ```bash
   shark sprint burndown {SPRINT_KEY} --json
   ```
   Display the burndown output. If the command fails or is unavailable, print a notice and continue.

5. Move to the next feature group.

**On `/run-agent-team` failure:** Print the error and ask the user whether to continue to the next feature group or abort:
```
/run-agent-team {FEATURE_KEY} encountered an error.
Continue to next feature group? (yes/no)
```
- **yes** → continue to next feature group.
- **no** → proceed to Step 6 (post-loop reporting and close gate).

---

## Step 5: Standalone Dispatch

If `STANDALONE` is non-empty:

Print:
```
[Sprint {SPRINT_KEY}] Dispatching {len(STANDALONE)} standalone entities via /run...
```

For each `ENTITY_KEY` in `STANDALONE`, sequentially:

1. Print:
   ```
   [Sprint {SPRINT_KEY}] Dispatching standalone {ENTITY_KEY} via /run...
   ```
2. Invoke `/run {ENTITY_KEY}`.
3. Wait for `/run` to return before dispatching the next standalone.

**On `/run {ENTITY_KEY}` failure:** Log the error. Continue to the next standalone entity. Do not abort the entire sprint dispatch on a single standalone failure.

---

## Step 6: Post-Loop Reporting

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
[Sprint {SPRINT_KEY}] Team dispatch complete.
Feature groups dispatched: {DISPATCHED_FEATURES joined by ", "}
Standalones dispatched: {DISPATCHED_STANDALONES joined by ", "}
```

---

## Step 7: Close Gate

**THIS STEP REQUIRES EXPLICIT USER CONFIRMATION. Do NOT skip this step under any circumstances.**

Ask the user:
```
Close sprint {SPRINT_KEY} now? (yes/no)
Carryover strategy: {CARRYOVER_VALUE if set, else "default (per sharkconfig)"}
```

**If user says no (or anything other than explicit "yes")**:
```
Sprint {SPRINT_KEY} left in active state. Run /run-sprint-team {SPRINT_KEY} again to continue, or close manually with:
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
| Sprint key not found (the state lookup returns not-found) | Print `Sprint {SPRINT_KEY} not found.` and exit |
| Precondition failure (Step 1) | Print specific precondition failure; abort before any dispatch |
| `shark sprint start` fails | Print error; exit without dispatching any entities |
| `shark sprint backlog` returns malformed JSON | Print raw response; report `shark sprint backlog returned unexpected output`; exit |
| `--features` filter matches no feature groups | Print notice; proceed to Step 6 |
| `/run-agent-team {FEATURE_KEY}` fails | Print error; ask user to continue or abort |
| `/run {ENTITY_KEY}` fails (standalone) | Log error; continue to next standalone |
| `shark sprint burndown` unavailable | Print notice; continue (burndown is informational only) |
| `shark sprint close` fails | Print full error; sprint remains open; suggest manual retry |

---

## Idempotency

This workflow is safe to re-invoke:

- If the sprint is already `completed`/`archived`/`cancelled`, Step 2 exits with a notice — no double-close possible.
- If re-invoked mid-execution, already-completed feature groups will show their tasks as terminal; `/run-agent-team` will pick up from current shark state for any remaining non-terminal tasks.
- `shark sprint close` is only reachable through the explicit user-confirmation gate in Step 7.

---

## Constraints

- All shark calls use `--json`.
- **Only one `/run-agent-team` invocation active at a time** — never concurrent.
- `/run-agent-team` is only used for feature-key groups (`E##-F##`). It is never invoked with bug, change-card, or tech-debt keys.
- Standalones are dispatched via `/run {ENTITY_KEY}`, not `/run-agent-team`.
- `shark sprint close` is only called after explicit user confirmation in Step 7.
- `shark sprint start` is only called if status is `planning` AND the user explicitly confirms (Step 2).
- The `--features` filter restricts dispatch; it never adds entities beyond the sprint's own backlog.
- Preconditions from `/run-agent-team` must all pass before any dispatch begins (Step 1). A failure aborts the entire workflow, including standalone dispatch.
