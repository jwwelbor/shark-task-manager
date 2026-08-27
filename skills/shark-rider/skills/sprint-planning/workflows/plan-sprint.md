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
- `goal`, `name` — the sprint's stated scope, used by the Step 1.5 tech-debt cross-check

**If status is `planning`**: proceed silently.

**If status is `active`**:
- Print an advisory:
  ```
  Note: Sprint {S###} is already active. Adding entities to an active sprint is allowed but unusual.
  Continue? (yes/no)
  ```
- If user says **no**: exit cleanly. Do not call any further shark commands.
- If user says **yes**: continue to Step 2.

**If status is NOT in `{planning, active}`**:
- Print an advisory:
  ```
  Note: Sprint {S###} is in status: {status}. Planning a {status} sprint is allowed but unusual.
  Continue? (yes/no)
  ```
- If user says **no**: exit cleanly. Do not call any further shark commands.
- If user says **yes**: continue to Step 2.

---

## Step 1.5: Tech-Debt Landmine Cross-Check

Before proposing any backlog assignments, check whether open tech-debt
contains a **landmine** under this sprint's scope — a defect *in the code
path the sprint is about to build or extend*, as opposed to debt that is
merely topically adjacent. This is a Rider-side judgment call (relevance
requires reading intent, not just matching), not something `shark` itself
computes — do not look for or wait on any shark-internal readiness/gate
field to surface this; shark's `readiness` score and `plan` backlog answer
"is this entity eligible for assignment," not "is this a defect under our
target surface," and severity triage in the tech-debt backlog does not
reliably keep pace with intake.

1. From Step 1's `goal` and `name`, extract the epic/feature keys (e.g.
   `E09`, `E11`, `E04-F02`) and domain nouns (e.g. "freeze", "reload",
   "manifest") the sprint actually names.
2. Pull the full tech-debt backlog: `shark list tech-debt --json`. Use the
   full list, not Step 2's `shark sprint plan` backlog — that only contains
   items already eligible for assignment by shark's own rules, a narrower
   and different question.
3. Filter to items whose `title`/`description` reference the same
   epic/feature keys or domain nouns from step 1, regardless of `severity`
   or `status`.
4. For each match, judge it the way an architect would: is this a defect IN
   the surface the sprint is about to write or extend (a landmine), or just
   topically related (leave it alone)? Only landmines count as findings.
5. If any landmines are found, present them before continuing to Step 2:
   ```
   Tech-debt landmine check for Sprint {S###}:
     • {TD-KEY} — {title}
       Why it hits this sprint: {one sentence}
       Effort: {effort_estimate}   Status: {status}
   Pull {TD-KEY} into this sprint now (recommended: --at 1)? (yes/no)
   ```
   On **yes**: `shark sprint add {S###} {TD-KEY} --at 1 --json`, then record
   the decision as a note on both entities (`shark create note {S###} "..."
   --type=decision`, and the matching note on `{TD-KEY}`) per CLAUDE.md
   Rule 14.
   On **no**: leave it and continue; do not silently drop it — repeat it in
   the Step 5 completion summary as a known, declined risk.
6. If nothing matches, say so explicitly:
   ```
   Tech-debt landmine check for Sprint {S###}: none found.
   ```
   Silence here must mean "checked, clean" — never skip this line.

This step never blocks planning and never assigns anything without the
per-item confirmation above, in both interactive and auto mode.

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
To execute: /shark-rider run-sprint {S###}
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
     - If `entity.size == 0` (unsized): mark as SKIPPED with reason "unsized — excluded from capacity calculation". Do not increment allocated.
     - Else if `capacity_by_agent[entity.agent_type] - allocated[entity.agent_type] >= entity.size`:
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
To execute this sprint: /shark-rider run-sprint {S###}
```

If Step 1.5 found any landmines the user declined to pull in, repeat them
here so they aren't lost:
```
Declined tech-debt risk (still open under this sprint's surface):
  • {TD-KEY} — {title}
```

**DO NOT call `shark sprint start`.** Starting a sprint is an explicit user action. The user must run `/shark-rider run-sprint {S###}` (which will offer to start the sprint if needed).

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

---

## Size and Scope Guidelines (owner-ratified 2026-08-15)

A sprint's job is to produce **one coherent, demoable increment** tied to a
single Sprint Goal — not a bucket of whatever backlog items fit under a point
ceiling. Capacity/velocity is a forecasting tool for *how much* of that goal
fits in the window; it is not license to pull in unrelated backlog just
because points are free.

Apply these checks in every planning session, interactive or auto:

- **One goal, one theme.** Before proposing assignments, look at what
  feature/epic clusters the eligible backlog actually falls into. If the
  candidate set spans more than ~2 unrelated epics/themes, don't propose all
  of it — propose the largest coherent cluster (the epic/feature group with
  the most in-flight or interdependent work) as the sprint, and set the
  `--goal` on `shark sprint create`/`shark sprint update` to name the
  demoable outcome in one sentence.
- **Standalone entities (bugs/change-cards/tech-debt) need a home, not a
  default.** Don't dump all eligible bugs/CCs/TDs into whichever sprint is
  being planned just because they showed up in the backlog. Tech-debt items
  in `identified` status are their normal resting state — being eligible is
  not the same as being sprint-worthy. Pull in only items that are (a) direct
  functional blockers on the sprint's chosen theme, or (b) explicitly
  requested by the user. Route everything else to a future sprint grouped by
  its own coherent theme (see "Route leftovers" below) rather than leaving it
  to accumulate unsorted in the next planning pass.
- **Every sprint-assigned entity needs a size before the sprint is
  considered ready.** `shark sprint readiness` scores "Sizing coverage" and
  "Capacity utilization" — an entity with no `size` silently drops out of the
  capacity math and inflates the readiness score misleadingly. Before
  reporting readiness, check `unsized_entities` in the readiness response and
  size every one of them with `shark update <key> --size N` (fibonacci: 1,
  2, 3, 5, 8, 13) using the same judgment you'd apply to any other estimate —
  read the entity, compare it to already-sized siblings in the same
  feature/theme, don't leave it as a TODO for later. Note: tech-debt entities
  may already carry a t-shirt `effort_estimate` (XS–XXL) in their doc
  frontmatter — that is a *different* field from sprint `size` and does not
  substitute for it; both can coexist.
- **Capacity should reflect the chosen scope, not an arbitrary constant.**
  When no capacity is configured yet (`shark sprint capacity show` returns
  empty), set it based on the actual point total of the coherent cluster
  you're proposing — not a round guess applied before scope is narrowed.
  Capacity utilization near 100% for a single-theme sprint is a good signal;
  utilization over ~150% is a sign the scope needs narrowing, not that
  capacity needs raising.
- **Route leftovers to themed future sprints, not one big backlog dump.**
  When narrowing scope drops entities out of the sprint being planned, group
  the remainder by their own coherent theme (shared epic, shared subsystem,
  shared root cause) and create one sprint per theme
  (`shark sprint create "<theme>" --start=… --end=… --goal="…"`) rather than
  parking everything in a single undifferentiated follow-up sprint.

Origin: S002 was initially planned with 59 entities across 6 unrelated
tracks (E04, E10, E11, E19, generic tooling bugs, cross-cutting change-cards,
and unrelated Blades/E12 tech-debt) at 178% capacity utilization, with 24
entities carrying no size at all. The owner flagged that a sprint needs to
represent a demoable increment, not a capacity-filling grab-bag. It was
re-scoped down to the 21-task E11 readiness cluster (verdict → report →
waiver write path) and the remainder was split into five themed follow-up
sprints (S003–S007).
