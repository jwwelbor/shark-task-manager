# /shark run — Dispatch Loop

Drive an entity through its workflow. Usage:

```
/shark run <entity-key>     # E01, E01-F02, E01-F02-003, B001, CC-001
/shark run bugs             # collection: every non-terminal bug
/shark run change-cards     # collection: every non-terminal change-card
```

`/run <key>` is an alias for this verb.

## Contract

This verb is a mechanical host loop around the canonical Shark dispatch API:

```
loop: shark next {KEY} --json -> response.action -> execute response.prompt -> advance -> loop
```

`shark next` owns workflow routing and prompt assembly. The returned
`response.prompt` is the only execution payload. It already contains the
rendered workflow prompt, resolved `{{include: ...}}` skill content, and the
Shark specialist agent persona from the content bundle.

Do not build prompts from `shark get ... orchestrator_action`. Do not load Shark
skills or Shark specialist agents from the host filesystem. Treat
`response.agent_type`, `response.provider`, and `response.model` as Shark
metadata for logging/provenance and host adapter selection only.

## Ownership model

The parent loop owns the lease and every status transition. Per spawned step:

1. Parent claims `response.entity_key`.
2. Parent spawns a host-safe worker with `response.prompt`.
3. Worker returns only `{ "outcome": "pass" | "fail" | "blocked", "summary": "...", "note": "..." }`.
4. Parent runs `shark status advance {response.entity_key} --outcome <outcome>`.
5. Parent releases the lease on every success, failure, or exception path.

The child never claims, advances, releases, or heartbeats.

## Roots vs collections

- A specific key is worked by repeatedly calling `shark next <that-key> --json`.
- An epic or feature key may resolve to an unclaimed child; execute the returned
  `response.entity_key`, then call `shark next <original-root> --json` again.
- Collection keywords are not valid `shark next` roots. For `bugs` and
  `change-cards`, enumerate non-terminal items with `shark bug list --json` or
  `shark change list --json`, then run this loop for each concrete key.

## Step 0 — Branch check

Shark's own DB is the activity record — `shark status history <key>` shows every
transition with timestamp, agent, and outcome, and the `shark claim --by
"$CLAUDE_SID"` in Step 2 ties the session to the entity. Do not write a separate
run log.

Check `git branch --show-current`:
- On `main`/`master` -> ask the user before proceeding.
- On the matching branch for this entity -> continue.
- On an unrelated branch -> ask the user.

Branch patterns: `E##-F##`, `E##-F##-description`, `feature/E##-F##`,
`fix/B###-*`, `change/CC-###-*`.

## Step 1 — Read next dispatch response

```bash
shark next {KEY} --json
```

Parse the JSON response:

| Field | Meaning |
|-------|---------|
| `response.action` | Wire action: `spawn_agent`, `pause`, `archive`, or `error` |
| `response.entity_key` | Concrete entity to claim, execute, advance, and release |
| `response.entity_type` | Concrete entity type |
| `response.status` | Current workflow status |
| `response.prompt` | Self-contained execution prompt |
| `response.agent_type` | Shark persona metadata, not a native host subagent name |
| `response.provider` | Provider metadata, e.g. `anthropic`, `openai`, `codex` |
| `response.model` | Model metadata or override |
| `response.effort` | Optional reasoning-effort override for the worker (`low`, `medium`, `high`, `xhigh`) |
| `response.resolved_via` | Optional parent keys traversed by cascade resolution |
| `response.error` | Error detail when action is `error` or pause carries a warning |

Report: `Entity {response.entity_key} is at status: {response.status}`.

## Step 2 — Execute the wire action

### `spawn_agent`

Claim the concrete entity returned by Shark:

```bash
SID=$(shark claim {response.entity_key} --by "$CLAUDE_SID" --field session_id)
```

Spawn the host worker using a host-safe adapter:

- Claude Code Agent tool: select `subagent_type` by `response.effort` —
  `low`/`medium`/`high`/`xhigh` → `shark-worker-<effort>`; absent or
  unrecognized effort → `general-purpose`
- Prompt: exactly `response.prompt`
- Metadata to record/pass through if the host supports it: `response.agent_type`,
  `response.provider`, `response.model`

Do not set the host `subagent_type` to Shark names such as `business-analyst`,
`product-manager`, or `tech-director`; those personas are already inside
`response.prompt`.

For long steps, periodically renew the parent lease:

```bash
shark heartbeat {response.entity_key} --session "$SID" --progress <0..1> --note "<step>"
```

When the worker returns:

1. If it returned `note`, record it:
   ```bash
   shark create note {response.entity_key} "<note>" --type comment
   ```
2. Advance by the returned outcome, attributing the transition to the agent
   that did the work (this populates `entity_history.changed_by`):
   ```bash
   shark status advance {response.entity_key} --outcome <pass|fail|blocked> \
     --agent "{response.agent_type}@{response.provider}/{response.model}"
   ```
3. Release the lease, always, stamping the outcome on the work session:
   ```bash
   shark release {response.entity_key} --session "$SID" --outcome <pass|fail|blocked>
   ```
4. Return to Step 1 with the original `{KEY}`.

If the worker fails or throws, still release the lease, record a blocker note if
possible, and surface the failure before deciding whether to retry.

### `pause`

Report `response.prompt` and any `response.error`. Stop.

### `archive`

Report done. Stop.

### `error`

Report `response.error` and stop. Do not retry blindly.

## Resuming

Re-invoke `/shark run {KEY}`. The loop calls `shark next {KEY} --json` and picks
up from the current workflow state. Use `shark claims` to see live work; expired
leases are reclaimed automatically, and an administrative `shark release {KEY}`
can clear a dead parent lease when needed.
