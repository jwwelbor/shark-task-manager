# /shark run — Dispatch Loop (Shark 2.x-native)

Drive an entity through its workflow. Usage:

```
/shark run <entity-key>     # E01, E01-F02, E01-F02-003, B001, CC-001
/shark run bugs             # collection: every non-terminal bug
/shark run change-cards     # collection: every non-terminal change-card
```

`/run <key>` is an alias for this verb.

## What this is

A **mechanical dispatch loop**. It reads state, runs the action the workflow
returns, and loops. It does not know what the workflow steps are or whether any
step is "needed" — it runs whatever shark hands back.

```
loop:  shark get {KEY} --json → orchestrator_action → execute → goto loop
```

## Ownership model — single owner

The **parent loop owns the lease and every status transition.** Per iteration:

1. Parent **claims** the entity (acquires the session lease).
2. Parent **spawns the child agent** for the current step.
3. Child does its craft and **returns a structured outcome only**:
   `{ "outcome": "pass" | "fail" | "blocked", "summary": "...", "note": "..." }`.
   The child **never** claims, advances, releases, or heartbeats.
4. Parent runs `shark status advance {KEY} --outcome <outcome>`.
5. Parent **releases the lease in every terminal/error path** — success, agent
   failure, or exception.

The claim TTL is the backstop if the *parent* itself crashes mid-iteration.

## Entity-type detection

| Pattern | Type | State read |
|---------|------|-----------|
| `E##` | Epic | `shark get {KEY} --json` |
| `E##-F##` | Feature | `shark get {KEY} --json` |
| `E##-F##-###` | Task | `shark get {KEY} --json` |
| `B###` | Bug | `shark get {KEY} --json` |
| `CC-###` | Change-card | `shark get {KEY} --json` |
| `bugs` | Bug collection | see **Roots vs collections** |
| `change-cards` | Change-card collection | see **Roots vs collections** |

## Roots vs collections

- A **specific key** (epic/feature/task/bug/change) is worked directly.
- `shark next <root>` accepts a single **entity root key** (epic/feature) and
  returns the next *unclaimed* child. Use it ONLY with a valid key.
- **Collection keywords** (`bugs`, `change-cards`) are NOT valid `next` roots.
  Handle them by enumeration instead:
  ```bash
  shark bug list --json        # or: shark change list --json
  ```
  Filter to non-terminal items, then claim/work each one individually through
  the loop below.

## Step 0 — Log + branch check

```bash
mkdir -p docs/workflow
echo '{"ts":"'$(date -Iseconds)'","sid":"'$CLAUDE_SID'","event":"run_started","entity":"{KEY}","detail":{"command":"/shark run {KEY}","branch":"'$(git branch --show-current)'"}}' >> docs/workflow/activity.jsonl
```

Check `git branch --show-current`:
- On `main`/`master` → **ASK USER** before proceeding.
- On the matching branch for this entity → continue.
- On an unrelated branch → **ASK USER**.

Branch patterns: `E##-F##`, `E##-F##-description`, `feature/E##-F##`,
`fix/B###-*`, `change/CC-###-*`.

## Step 1 — Build the phase maps (once per run)

Resolve the active workflow YAML (`.sharkconfig.json` → `workflow_config`; if
absent, `<content-bundle>/workflow/`). Parse the entity's workflow file and build:

- `status → phase` — from each step's `phase:`, **plus every name in that step's
  `aliases:`** (so legacy historical statuses like `ready_for_qa` resolve to the
  current step's phase).
- `phase → order` — the linear order phases appear in the workflow.

These maps drive the escape-hatch below. **Do not hardcode any `ready_for_*` /
`in_*` names** — everything comes from the YAML.

## Step 2 — Read state

```bash
shark get {KEY} --json
```

Read `status` and `orchestrator_action`. The `orchestrator_action` is the **sole
instruction**. Report: `"Entity {KEY} is at status: {status}"`.

Single-field reads (never pipe JSON through `head`/`grep`/`python`/`jq`):
```bash
shark get {KEY} --json --field status
shark get {KEY} --json --field orchestrator_action
```

| Field | Meaning |
|-------|---------|
| `orchestrator_action.action` | `spawn_agent`, `advance_status`, `cascade`, `pause`, `archive` |
| `orchestrator_action.agent_type` | Which agent to spawn |
| `orchestrator_action.instruction` | The fully-rendered prompt (engine already inlined bundle prompts/skills/agents) |
| `orchestrator_action.skills` | Skills the agent should use |

## Step 3 — Escape hatch (rework detector)

Before spawning into a step, count how many times this entity has been **routed
backward** (a `fail`/rework hop):

```bash
shark status history {KEY} --json
```

For each transition, map `old_status` and `new_status` to phases via the Step-1
maps. Count the transition as **rework** when `phase→order[new] < phase→order[old]`.
Historical statuses that resolve to **no** phase are **skipped, not counted**.

If rework count **≥ 2**:
```bash
shark create note {KEY} "Loop guard: {N} backward routes detected — halting for human review." --type=blocker
shark status set {KEY} blocked --force --reason "Loop guard: {N} rework routes — human review required"
```
Report the halt and **STOP**. (No hardcoded status names anywhere in this check.)

## Step 4 — Execute the action

### `spawn_agent`

```bash
SID=$(shark claim {KEY} --by "$CLAUDE_SID" --field session_id)   # acquire lease
```
1. Spawn the child agent (Agent tool): `subagent_type` =
   `orchestrator_action.agent_type`, prompt = `orchestrator_action.instruction`.
   For long steps, periodically renew the lease:
   `shark heartbeat {KEY} --session "$SID" --progress <0..1> --note "<step>"`.
2. The child returns `{ outcome, summary, note? }`. If it returned a `note`, record it:
   `shark create note {KEY} "<note>" --type comment`.
3. Advance by the released outcome (never a bare advance):
   ```bash
   shark status advance {KEY} --outcome <pass|fail|blocked>
   ```
4. **Release the lease** — always, including on agent failure/exception:
   ```bash
   shark release {KEY} --session "$SID"
   ```
5. Go to **Step 2**.

If the agent fails or throws, still run step 4 (release), record a `blocker`
note, and surface the failure to the user before deciding whether to retry.

### `advance_status`

Auto step, no agent. `shark status advance {KEY} --outcome pass` → go to Step 2.

### `cascade`

The entity is a parent that drives children.

- **Epic** → spawn the agent from `orchestrator_action` (it iterates features),
  or drive features directly via the loop, one per child.
- **Feature** → enter the **Task dispatch loop** below.

### `pause`

Report `orchestrator_action.instruction` to the user. **STOP.**

### `archive`

Report done. **STOP.**

---

## Task dispatch loop (feature cascading into tasks)

```
loop:
  shark list {EPIC} {FEATURE} --json            # all tasks
  for each task not at a terminal status (completed, cancelled):
    run Steps 2–4 above for that task key (claim → agent → advance --outcome → release)
  if all tasks completed → STOP
  if any task still active → continue loop
```

Rules:
- For EVERY task, read `orchestrator_action` (Step 2) before acting.
- Never advance a task without reading its `orchestrator_action` first.
- The loop terminates only when all tasks reach `completed` (or are
  blocked/cancelled).

Stop conditions: all tasks completed; an unresolvable blocker; or a spec
contradiction needing a human decision.

---

## The rule

```
read orchestrator_action → execute → read orchestrator_action → execute → …
```

It does NOT: know workflow steps, decide whether a step is needed, skip ahead,
advance more than once without re-reading, or substitute its own judgment for
shark's instructions.

## Resuming

Re-invoke `/shark run {KEY}`. Step 2 reads the current status; the loop picks up
wherever the entity is. Expired leases are reclaimed automatically; a live lease
from a dead parent clears on TTL (or `shark release {KEY}` administratively).
