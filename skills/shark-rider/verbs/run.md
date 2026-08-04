# /shark-rider run — Dispatch Loop

Drive an entity through its workflow. Usage:

```
/shark-rider run <entity-key>     # E01, E01-F02, E01-F02-003, B001, CC-001
/shark-rider run bugs             # collection: every non-terminal bug
/shark-rider run change-cards     # collection: every non-terminal change-card
/shark-rider run tech-debt  # collection: every non-terminal tech-debt item
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
3. Worker reports its result in its final response — no Shark commands. The
   result is an outcome plus optional parent-loop directives (see "When the
   worker returns"). The outcome may be **any outcome key defined for the
   current step** (`shark status transitions <key> --json`), not just
   pass/fail/blocked — workflow prompts route through extra outcomes such as
   `simple`, `standard`, or `deep_verify`. Workers signal it with a trailing
   `RECOMMENDED OUTCOME: <key>` line; a worker that instead returns a JSON
   `{ "outcome": ... }` object means the same thing.
4. Parent applies any directives (kickbacks, notes), then runs
   `shark status advance {response.entity_key} --outcome <key>`.
5. Parent releases the lease on every success, failure, or exception path.

The child never claims, advances, releases, or heartbeats.

## Roots vs collections

- A specific key is worked by repeatedly calling `shark next <that-key> --json`.
- An epic or feature key may resolve to an unclaimed child; execute the returned
  `response.entity_key`, then call `shark next <original-root> --json` again.
- When cascade resolution finds 2+ dispatchable children tied in the top
  tier, `shark next` stops instead of picking one and returns a fork response
  — see "`parallel_candidates` (fork)" under Step 2.
- Collection keywords are not valid `shark next` roots. For `bugs`,
  `change-cards`, and `tech-debt`, enumerate non-terminal items with
  `shark bug list --json`, `shark change list --json`, or
  `shark td list --json`, then run this loop for each concrete key.

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

Only `shark next {KEY} --json` supplies a claimable dispatch response. Do not
claim a selected `BacklogItemView` directly. A selector (including a role-aware
`shark sprint next --agent=<type>` pull) supplies only a key. Reenter this
procedure with that key so `shark next` returns the canonical dispatch response
before any claim or worker execution.

Parse the JSON response:

| Field | Meaning |
|-------|---------|
| `response.action` | Wire action: `spawn_agent`, `pause`, `archive`, `error`, or `parallel_candidates` (fork — see below) |
| `response.entity_key` | Concrete entity to claim, execute, advance, and release |
| `response.entity_type` | Concrete entity type |
| `response.status` | Current workflow status |
| `response.prompt` | Self-contained execution prompt |
| `response.prompt_sha256` | Hex-encoded SHA-256 digest of the exact `response.prompt` bytes (REQ-F-011) |
| `response.prompt_bytes` | Byte length of `response.prompt`, computed alongside `prompt_sha256` |
| `response.agent_type` | Shark persona metadata, not a native host subagent name |
| `response.provider` | Provider metadata, e.g. `anthropic`, `openai`, `codex` |
| `response.model` | Model to dispatch the worker with (e.g. `sonnet`, `opus`, `haiku`, `fable`) when `response.provider` is `anthropic` |
| `response.effort` | Optional reasoning-effort override for the worker (`low`, `medium`, `high`, `xhigh`) |
| `response.resolved_via` | Optional parent keys traversed by cascade resolution |
| `response.error` | Error detail when action is `error` or pause carries a warning |

Report: `Entity {response.entity_key} is at status: {response.status}`.

CLI adapters that need the exact prompt bytes on disk rather than in memory
add `--prompt-out <path>` to the `shark next` call above; verify the written
file's SHA-256 against `response.prompt_sha256` before spawning. See
`context/host-adapter-contract.md` for the full provider-neutral
request/result field set an adapter exchanges with the parent.

## Step 2 — Execute the wire action

### `spawn_agent`

Claim the concrete entity returned by Shark:

```bash
SID=$(shark claim {response.entity_key} --by "$CLAUDE_SID" --field session_id)
```

Spawn the host worker using a host-safe adapter:

- Claude Code Agent tool: select `subagent_type` by `response.effort` —
  `low`/`medium`/`high`/`xhigh` → `shark-worker-<effort>`; absent,
  unrecognized, or not defined in this project → `general-purpose`
- Model: when `response.provider` is `anthropic` and `response.model` is
  non-empty, pass it as the Agent tool's `model` parameter (accepts
  `sonnet`/`opus`/`haiku`/`fable`, or a full model name). `subagent_type`
  only controls reasoning effort — `model` is what actually selects the
  model, so both must be set together; do not skip this and assume effort
  routing covers it. Omit `model` when `response.model` is empty. Do not pass
  a non-Claude value (e.g. `codex`) as the Agent tool's `model` — that param
  only accepts Claude models.
- Prompt: exactly `response.prompt`
- Metadata to record/pass through if the host supports it: `response.agent_type`,
  `response.provider`, `response.model`
- Worker execution mode: single worker by default. Do not infer recursive
  delegation from Shark persona names alone. Only recurse when the workflow
  prompt explicitly invokes a multi-agent skill or recipe (for example the
  sprint-execution skill or UAT's Codex red-team step).

Do not set the host `subagent_type` to Shark names such as `business-analyst`,
`product-manager`, or `tech-director`; those personas are already inside
`response.prompt`.

For long steps, periodically renew the parent lease:

```bash
shark heartbeat {response.entity_key} --session "$SID" --progress <0..1> --note "<step>"
```

### Mid-run consultation (`kind: question` / `kind: needs_council`)

Before the worker returns a final result, it may instead pause with a
control envelope (the shark-attack skill's
`context/worker-control-schema.yaml`,
`kind: final|question|needs_council|blocked_external|failed`) — distinct
from the `RECOMMENDED OUTCOME:` vocabulary below, which only ever applies
to `kind: final`. Keep heartbeating the dispatched entity's own lease for
the whole consultation, bounded by its remaining claim lease — never let it
lapse while a responder is pending.

- **`kind: question`** — route the consultation through the shark-attack
  skill's `workflows/route-question.md` (mint the `Q###`, configure, gate,
  route, respond, resolve); do not restate that procedure here. Deliver the
  answer back to the worker using the resume path
  `context/host-adapter-contract.md` and the shark-attack skill's
  `workflows/resume.md` describe: the same worker identity by native
  follow-up when the host supports resume, otherwise exactly one bounded
  replacement worker built from an immutable handoff. Once delivered, the
  worker keeps running under this step — it may pause again with another
  `kind: question`, or eventually return `kind: final`.
- **`kind: needs_council`** — route through the shark-attack skill's
  `workflows/council.md`; do not duplicate its threshold or procedure here.
- **`kind: blocked_external` or `kind: failed`** — no `RECOMMENDED OUTCOME:`
  line follows. Treat it exactly like the "No outcome at all" case below:
  record a blocker note quoting the envelope's `evidence`, then advance
  with outcome `blocked`.

When the worker returns, parse its final response for the directive markers
the workflow prompts emit, apply them in this order, then advance:

1. **Resolve the outcome.** Take the trailing `RECOMMENDED OUTCOME: <key>`
   line (or the `outcome` field of a JSON return). Any outcome key defined
   for the current step is valid — `shark status advance` rejects unknown
   keys, so pass the worker's key through verbatim rather than coercing it to
   pass/fail/blocked. No outcome at all → treat as `blocked` and record a
   blocker note quoting the tail of the response.
2. **Persist notes.**
   - A `COMPLEXITY NOTE: <text>` line (complexity-triage steps) must be
     stored as a decision so later steps can read the tier:
     ```bash
     shark create note {response.entity_key} "<text>" --type decision
     ```
   - A `PARENT NOTE: <text>` line records the gate result:
     ```bash
     shark create note {response.entity_key} "<text>" --type comment
     ```
   - Any other `note` the worker returned:
     ```bash
     shark create note {response.entity_key} "<note>" --type comment
     ```
3. **Apply task kickbacks.** Gate steps (code review, QA, UAT) that fail list
   kickback lines in the form
   `<task-id> -> <status> --reason "<why>"`.
   Apply each one BEFORE advancing the parent, so the reopened tasks are
   already in place when the feature drops back:
   ```bash
   shark status set <task-id> <status> --reason "<why>" --force
   ```
   A `fail` outcome whose response names no kickbacks is suspicious — record a
   blocker note and surface it to the user rather than silently advancing.
4. **Advance by the resolved outcome**, attributing the transition to the
   agent that did the work (this populates `entity_history.changed_by`):
   ```bash
   shark status advance {response.entity_key} --outcome <key> \
     --session "$SID" --from-status "{response.status}" \
     --agent "{response.agent_type}@{response.provider}/{response.model}"
   ```
   The outcome is workflow-configured: do not substitute a hard-coded status
   or coerce a valid semantic outcome to `pass` or `fail`. Include
   `--session`/`--from-status` when `.sharkconfig.json` enables `advance_guard`
   (see the route-based workflow guide, §2).
5. **Release the lease, always**, stamping the outcome on the work session:
   ```bash
   shark release {response.entity_key} --session "$SID" --outcome <key>
   ```
6. Return to Step 1 with the original `{KEY}`.

If the worker fails or throws, still release the lease, record a blocker note if
possible, and surface the failure before deciding whether to retry.

### `parallel_candidates` (fork)

Shark stops at a cascade fork instead of silently picking one child: 2+
dispatchable children tie in the top tier. The response reuses the
`HierarchyPlanSelectionResponse` envelope the `plan` verb already consumes —
`response.mode` is `hierarchy_selection`, `response.action` is
`parallel_candidates`, `response.parallel_execution` is `available`,
`response.selection_reason` is `parallel_tie`. The tell is what's absent:
there is no `response.prompt` — a fork is a selection, not a dispatch.

| Field | Meaning |
|-------|---------|
| `response.root_key` / `response.root_type` | The parent whose children forked |
| `response.resolved_via` | Keys walked to reach the fork, e.g. `["E02","F03"]` |
| `response.entities` | Candidate children: `entity_key`, `entity_type`, `title`, `status`, `execution_order`, `priority`, optional `depends_on` / `blocks` / `links` arrays of `{key, status, type}`, and optional `warnings` |

1. **Evaluate integration safety.** `response.parallel_execution: "available"`
   proves only that workflow state and stored dependencies allow concurrent
   dispatch — it does not prove product integration safety; the Rider
   decides. Apply the same evidence procedure `/shark-rider plan` already
   defines for its own `hierarchy_selection` / `parallel_candidates`
   response (its Procedure steps 3–5 and its Recommendation rules): scope by
   `response.root_type` (epic root -> epic-local interaction map `I-##`
   rows; feature root -> stored task dependency evidence, using
   `response.entities[].depends_on` / `blocks` / `links`), and treat a
   missing, malformed, `proposed`, or `deferred` relevant row as an
   integration evidence gap that rules out that candidate for independent
   parallel launch. Do not restate or reimplement that procedure here — read
   it. A candidate warning with
   `code: "DANGLING_RELATIONSHIP"` means a stored relationship endpoint could
   not be resolved and the reported edge set is incomplete. Report the
   warning and do not independently parallel-launch that candidate until the
   row is repaired; following one candidate sequentially remains available.
2. **Choose a subset.** Keep every entity in `response.entities` that clears
   the check above. This may be all of them, several, or exactly one —
   nothing requires fanning out just because Shark offered a tie.
3. **Dispatch each chosen candidate the same way.** Fan-out and
   follow-one-candidate are the same mechanism: call
   `shark next <child-key> --json` for each chosen `entity_key`. Following a
   single candidate is just the one-child case of this same call — there is
   no separate command for it. Return to Step 1 with each candidate's key;
   each reenters this procedure independently (claim, spawn, advance,
   release) exactly like any other keyed dispatch.

An operator who wants Shark's pre-fork single-track behavior back can pass
`--sequential` to `shark next`, or set `sequential_dispatch: true` in
`.sharkconfig.json` (the flag overrides the config). That choice is made
before Step 1 runs; this loop does not implement it.

### `pause`

Report `response.prompt` and any `response.error`. Stop.

### `archive`

Report done. Stop.

### `error`

Report `response.error` and stop. Do not retry blindly.

## Resuming

Re-invoke `/shark-rider run {KEY}`. The loop calls `shark next {KEY} --json` and picks
up from the current workflow state. Use `shark claims` to see live work; expired
leases are reclaimed automatically, and an administrative `shark release {KEY}`
can clear a dead parent lease when needed.
