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

When dispatching a bug in a selected working directory, pass that actual
directory to `shark next {KEY} --workdir "$WORKDIR" --json` before spawning
the worker. Shark checks that it is the corresponding project root in the
parent repository or a linked worktree. When no separate workdir was selected, use the parent project root as the worker directory.

## Ownership model

The parent loop owns the lease and every status transition. Per spawned step:

1. Parent claims `response.entity_key`.
2. Parent spawns a host-safe worker with `response.prompt`.
3. Worker reports its result in its final response; it may register related
   documents but never changes Shark lifecycle state. The
   result is an outcome plus optional parent-loop directives (see "When the
   worker returns"). The outcome may be **any outcome key defined for the
   current step** (`shark status transitions <key> --json`), not just
   pass/fail/blocked — workflow prompts route through extra outcomes such as
   `simple`, `standard`, or `deep_verify`. Workers signal it with a trailing
   `RECOMMENDED OUTCOME: <key>` line; a worker that instead returns a JSON
   `{ "outcome": ... }` object means the same thing.
4. After every terminal worker result (`final`, `blocked_external`, or
   `failed`), parent verifies the worker's documented completion guarantee,
   then applies any directives (kickbacks, notes) and runs
   `shark status advance {response.entity_key} --outcome <key>`. Use an
   awaited foreground worker when the host has no documented retirement
   operation; never use a background worker whose later idle notifications
   cannot be prevented. Missing completion evidence blocks advancement but
   never prevents release.
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
| `response.model` | Requested model for `response.provider`, regardless of provider |
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

**Exclusivity pre-flight.** Inspect the host inventory for the keyed entity
before claiming it. Use `ListAgents` or the provider's process/session
inventory. If the inventory is empty, continue to route preflight and the
normal non-force claim. A claim conflict ends this attempt before submission.

For each visible worker, obtain its original, non-reusable host handle and
the Shark session bound to that handle at dispatch (`HOST_HANDLE` and
`HOST_WORKER_SID`). Read the entity, session and handle of the retired
dispatch independently from the parent's original launch and confirmed
retirement, such as its successful session-qualified release. Set
`RETIRED_KEY`, `RETIRED_SID` and `RETIRED_HANDLE` only from that record.
An idle notification, a terminal message, an entity match or a PID alone
cannot establish retirement. If the inventory or attribution is inconclusive,
preserve the worker and report unresolved exclusivity before claiming.

Run this guard in the parent project for each visible worker. Bind
`host_stop` and `host_confirm_exit` to the provider's stop and exit check
on the supplied original handle. A live holder, unavailable operation,
failed stop or unconfirmed exit ends the attempt before any new launch.
The Shark CLI may return JSON null for an empty claim list.

```bash
[[ -n ${HOST_HANDLE:-} && -n ${HOST_WORKER_SID:-} &&
   ${RETIRED_KEY:-} == "${ENTITY_KEY:?}" &&
   ${RETIRED_HANDLE:-} == "$HOST_HANDLE" &&
   ${RETIRED_SID:-} == "$HOST_WORKER_SID" ]] || exit 1
current_claims=$(shark claims --json) || exit 1
jq -es --arg key "$ENTITY_KEY" '
  length == 1 and (.[0] | . == null or
    (type == "array" and all(.[]; (.entity_key | type) == "string" and .entity_key != $key)))
' <<<"$current_claims" >/dev/null || exit 1
host_stop "$HOST_HANDLE" && host_confirm_exit "$HOST_HANDLE" || exit 1
```

Continue only after confirming exit for every attributable lingering worker.
A competing parent can still acquire the lease after inspection. Use the
ordinary claim below and preserve the winner on conflict.

> Recorded failure (2026-09-15): three prior resumptions' workers for the
> same task stayed alive well after their idle notifications, and one
> continued unprompted into a later resumption's assigned scope — caught
> only because a subsequent worker happened to run `ps -ef` and noticed.
> No corruption resulted that time, but it was luck, not a guarantee.

### Route attestation pre-flight (B077)

Before claiming the entity or starting a worker, prefer the exact requested
provider/model route. For an Anthropic request, Claude Code is preferred when
available, but its absence alone is not a rejection reason. Use a fallback
only through a documented compatible host selector and explicit compatibility
mapping. The selector must observe the worker-correlated effective
provider/model identity. Record the mapping and its capability and identity
evidence in `evidence_ref`; never infer compatibility from model names. Never
silently substitute the requested route.
The requested route, Shark persona
(`response.agent_type`), host worker class, and reasoning effort
(`response.effort`) are separate values. Do not infer the model from
`response.agent_type`, host worker class, or
`response.effort`, and do not use the parent's model as an implicit fallback.

When `response.model` is empty, preserve that exact requested value. Use a
provider default only when the host reference documents it and the adapter can
capture the worker-correlated concrete effective model. Reject before claim
only when no exact route or documented compatible fallback can be selected and
observed; missing or ambiguous route evidence is not permission to guess.

If pre-flight classifies the route as `unsupported` or `unverified`, the
parent/adapter issues a bounded parent/adapter pre-dispatch rejection that
records the requested provider/model and the capability or identity evidence
gap. This rejection has no worker control envelope or worker result, creates
no claim, and causes no status advance; the parent stops and reports the
rejection. `blocked_external` is reserved for a dispatched worker's control
envelope. Do not turn a pre-dispatch rejection into that worker result.

On a native Codex host, do not infer native host model selection from the
Codex CLI reference. The CLI's `--model` option may be used only when policy
explicitly selects the Codex CLI adapter and a documented compatible mapping
provides the selector; that option's presence proves selector availability,
not support or effective identity for a particular model. Require
worker-correlated observation of the effective provider and model before
attesting the route. Do not switch to the Codex CLI adapter silently.

For a native explicit model or effort, check the actual spawn mode for a
supported selector for each requested field and a child-correlated observation
of each. A mode that inherits from its parent cannot select an explicit
override. Keep omitted effort unforced. For any selected route, pass the requested model or explicitly mapped fallback
model through its documented selector without overwriting the Shark request.
Correlate host evidence through `worker_id` and, when needed, a distinct
`host_session_id`; `session_id` remains only the parent's Shark claim ID. Apply
the phase/status field and evidence rules in `context/host-adapter-contract.md`.
A post-dispatch `unverified` or `mismatch` route rejects the worker result,
stops the worker when supported, and does not advance the entity. A `verified`
route permits its worker result to continue through the existing control-
envelope path. Keep `response.prompt` byte-for-byte unchanged throughout this
check.

### B077 default bug-stage Codex fallback mappings

For the embedded bug workflow's Anthropic `spawn_agent` stages, prefer the
exact requested route through Claude Code when it is available. When Claude
Code is unavailable, the following explicit compatibility mapping selects
the Codex CLI route.
This is a workflow-level compatibility decision for bug-stage work; it does
not claim that any Codex model is equivalent to the requested Anthropic model.
Keep every requested provider/model value unchanged and record the selected
Codex route separately.

| entity_type | status | agent_type | requested_provider | requested_model | host_worker_class | selected_provider | selected_model | mapping_ref |
|---|---|---|---|---|---|---|---|---|
| bug | research | researcher | anthropic | haiku | codex-cli | openai | discovered model from selected CLI config/read | B077-default-bug-anthropic-codex-v1 |
| bug | repair_plan | architect | anthropic | sonnet | codex-cli | openai | discovered model from selected CLI config/read | B077-default-bug-anthropic-codex-v1 |
| bug | plan_review | tech-lead | anthropic | opus | codex-cli | openai | discovered model from selected CLI config/read | B077-default-bug-anthropic-codex-v1 |
| bug | development | developer | anthropic | sonnet | codex-cli | openai | discovered model from selected CLI config/read | B077-default-bug-anthropic-codex-v1 |
| bug | code_review | tech-lead | anthropic | opus | codex-cli | openai | discovered model from selected CLI config/read | B077-default-bug-anthropic-codex-v1 |
| bug | qa | qa | anthropic | sonnet | codex-cli | openai | discovered model from selected CLI config/read | B077-default-bug-anthropic-codex-v1 |

These are the current Anthropic stage tuples in the embedded default bug
workflow. The table applies only to the exact entity type, status, persona,
provider, and model shown; it does not authorize fallback for other entities,
stages, personas, Anthropic models, or an empty model. The mapping fixes the
selected provider and host class, but never pins a Codex model ID. Before claim,
resolve the selected Codex executable and use that same binary, selected
working directory, profile, and environment to perform app-server
initialization followed by `config/read`. Record its concrete configured model
as `selected_model` only when the returned provider is `openai` and the model
is non-empty. This read discovers the model to select; it does not authorize
launching with an implicit or ambient model. Pass the discovered value through
the explicit `--model` selector on every launch and capture the final argv for
that invocation. Verify the captured executable and argv contain the same
selected model and provider overrides recorded by preflight. Do not derive the
model from the parent session, environment, persona, or effort, and do not
replace an absent or conflicting discovery result with a guessed model ID.
If the exact route or table entry does not match, or model discovery, explicit
selection, argv capture, or worker-identity evidence is unavailable, reject
before claim with no worker result or status advance. A failed read,
unsupported explicit effort, or conflicting provider/model result stops this
assignment before claim. Keep only the bounded request, selected route,
executable identity, and argv digest as preflight evidence.
Before selecting this fallback, verify that the installed Codex CLI supports
`codex exec --model`, `codex login status` reports an authenticated session,
and persistent rollout records can be read from the configured Codex home.
Use only a CLI version whose selector and local identity-record shape have
been checked; the current host evidence is `codex-cli 0.170.0`. These are
local capability checks and make no model/API call. If any check fails, reject
before claim.

For each selected Codex CLI route, run this preflight block before claim. Use
the same shell and selected environment through launch. Set `REQUEST_FILE`
to this assignment's keyed response, `RIDER_ROOT` to the loaded Rider skill,
`CODEX_BIN` to the selected absolute executable and `ROUTE_FILE` to a fresh
private file for this attempt. Set `MAPPING_REF` only for a matching table
entry after the exact Claude route is unavailable. For an empty exact request
supply `DEFAULT_MODEL` from the independently documented rule for this CLI
and profile. Set `OBSERVATION_READY=1` only after checking the producing-turn
observation channel. Check support for the requested effort before invoking
the reader. These inputs come from this dispatch's capability check.

```bash
unset SELECTED_MODEL
REQUESTED_MODEL=$(jq -er '.model | strings' "$REQUEST_FILE") || exit 1
REQUESTED_EFFORT=$(jq -er '.effort // "" | strings' "$REQUEST_FILE") || exit 1
reader_args=(--request "$REQUEST_FILE" --cli "${CODEX_BIN:?}" --cwd "${WORKDIR:?}"
  --provider "${SELECTED_PROVIDER:?}" --profile "${SELECTED_PROFILE:-}"
  --sandbox "${SELECTED_SANDBOX:-}" --mapping "${MAPPING_REF:-}" --default-model "${DEFAULT_MODEL:-}"
  --evidence-ref "$ROUTE_FILE")
if [[ ${OBSERVATION_READY:-0} == 1 ]]; then reader_args+=(--observable); fi
timeout -k 2 15 python3 "$RIDER_ROOT/scripts/read-codex-route.py" \
  "${reader_args[@]}" > "$ROUTE_FILE" || exit 1
SELECTED_MODEL=$(jq -er '.selected_model | strings | select(length > 0)' "$ROUTE_FILE") || exit 1
```

A failed reader ends this assignment before any claim, submission or fallback.
Publish its bounded preflight rejection. On a timeout or unreadable receipt,
report preflight/unverified with the original request, class and this attempt's
evidence reference. Include only fields established by this read. A matched
provider with no discovered model carries only the provider. A completed
selection with an unavailable observation channel carries the pair. A named
profile unsupported by the reader cannot authorize a launch under that profile.

Set `ENTITY_KEY` to the concrete key from the keyed response. Set
`SELECTED_HARNESS=codex` for Codex CLI or native Codex and `claude` for Claude
Agent. Use the checked version and the preflighted model. The harness spelling
is what selects the corresponding Shark template branch.

Claim the concrete entity returned by Shark:

```bash
claim_by=$CLAUDE_SID
entity_type=$(jq -er '.entity_type | strings' "$REQUEST_FILE") || exit 1
if [[ $entity_type == question ]]; then
  claim_by=$(jq -er '.current_responder | strings | select(length > 0)' "$REQUEST_FILE") || exit 1
fi
claim_args=(--by "$claim_by" --field session_id)
if [[ -n ${SELECTED_HARNESS:-} ]]; then
  claim_args+=(--harness "$SELECTED_HARNESS" --harness-version "${SELECTED_VERSION:?}"
               --harness-model "${SELECTED_MODEL:?}")
fi
SID=$(shark claim "$ENTITY_KEY" "${claim_args[@]}") || exit 1
# This shell owns one assignment. Only clean up on an unsuccessful exit.
retire_failed_assignment() {
  local rc=$?
  if (( rc != 0 )); then
    if [[ -n ${SUPERVISOR_PID:-} ]]; then
      kill "$SUPERVISOR_PID" 2>/dev/null || true
      wait "$SUPERVISOR_PID" 2>/dev/null || true
    fi
    shark release "$ENTITY_KEY" --session "$SID" >&2
  fi
}
trap retire_failed_assignment EXIT
```

Before spawning the worker, start a parent-owned lease supervisor for this
session. The supervisor must renew with `shark heartbeat` at
`max(TTL/3, 1 second)` using the same `SID`; use the project-configured claim
TTL (the default is 15 minutes), not a hard-coded worker timeout. This is a
lease supervisor, not the liveness event stream and not a worker
responsibility.

- Start the supervisor before spawning the worker and keep it running through
  the worker's entire execution and any consultation. Bind its handle to this
  assignment. For the shell supervisor, keep its original child PID in
  `SUPERVISOR_PID` and clear the variable as soon as the supervisor is reaped.
- Treat any heartbeat failure as lease loss: stop the worker immediately, do
  not deliver its handoff, and do not apply the result. A stale handoff is
  context for a fresh keyed dispatch only; do not re-claim automatically under
  the old result or session.
- After the worker returns a terminal result, issue one final heartbeat. If it
  fails, stop the periodic supervisor and do not apply the result. If it
  succeeds, stop the periodic supervisor and immediately perform the result
  application (`--apply-result` for `gate_result_v1`), which still performs
  its own active-session re-check before writing.
- Release the session-scoped claim on every terminal or failure path when the
  session still matches; a lost or reissued lease is safely left to the next
  keyed dispatch and must never be used to authorize a write.

For every initial or replacement bug, refresh the assignment after starting
the supervisor, immediately before submission. Use fresh private paths for
`CURRENT_FILE` and `PROMPT_FILE`. Run the following commands in the parent
project. On any failed check retire this assignment and its supervisor and
release only its still-matching session. Discard its files. A fresh keyed
selection and preflight are required for another attempt.

```bash
shark heartbeat "$ENTITY_KEY" --session "$SID" >/dev/null || exit 1
shark next "$ENTITY_KEY" --workdir "$WORKDIR" --prompt-out "$PROMPT_FILE" --json > "$CURRENT_FILE" || exit 1
jq -e --slurpfile old "$REQUEST_FILE" --arg family "$SELECTED_HARNESS" \
  --arg version "$SELECTED_VERSION" --arg model "$SELECTED_MODEL" '
    def assignment: with_entries(select(.key | IN("entity_key","entity_type","status","action","agent_type","provider","model","effort")));
    .action == "spawn_agent" and assignment == ($old[0] | assignment)
    and .harness == $family and .harness_version == $version and .harness_model == $model
  ' "$CURRENT_FILE" >/dev/null || exit 1
[[ $(sha256sum < "$PROMPT_FILE" | cut -d" " -f1) == "$(jq -r .prompt_sha256 "$CURRENT_FILE")" ]] || exit 1
[[ $(wc -c < "$PROMPT_FILE") -eq $(jq -r .prompt_bytes "$CURRENT_FILE") ]] || exit 1
```

For a Question, use the pre-submit check in the retrieved
`workflows/route-question.md` for every responder, including a replacement.
Its focused E39 read and session-qualified heartbeat run under the current
Question claim. Submit the scoped prompt from the original keyed response.

Spawn the host worker using a host-safe adapter:

### Claude Code Agent tool adapter

- Select the Agent tool's `subagent_type` by `response.effort` —
  `low`/`medium`/`high`/`xhigh` → `shark-worker-<effort>`; absent,
  unrecognized, or not defined in this project → `general-purpose`
- For model selection, when `response.provider` is `anthropic` and `response.model` is
  non-empty, pass it as the Agent tool's `model` parameter (accepts
  `sonnet`/`opus`/`haiku`/`fable`, or a full model name). `subagent_type`
  only controls reasoning effort — `model` is what actually selects the
  model, so both must be set together; do not skip this and assume effort
  routing covers it. When `response.model` is empty, omit `model` only if
  preflight selected Claude's documented provider default and the adapter can
  capture the worker-correlated effective provider and concrete model; keep
  the requested model empty. If Claude Code is unavailable, continue only
  through an explicitly documented compatible host route. Do not pass a
  non-Claude value (e.g. `codex`) as the Agent tool's `model` — that param
  only accepts Claude models.

### Host-neutral handoff and lifecycle

- Prompt: exactly `response.prompt`
- Metadata to record/pass through if the host supports it: `response.agent_type`,
  `response.provider`, `response.model`
- Worker execution mode: single worker by default. Do not infer recursive
  delegation from Shark persona names alone. Only recurse when the workflow
  prompt explicitly invokes a multi-agent skill or recipe (for example the
  sprint-execution skill or UAT's Codex red-team step).
- Worker lifecycle: choose parent-owned awaited execution by default. A
  background dispatch is permitted only when the provider reference documents
  how the parent establishes terminal completion for that worker. Neither
  installed provider reference currently documents a native retirement
  operation, so do not use background dispatch for this loop.
- Worker termination: once you have ingested the worker's terminal result
  (Step 2's "Establish terminal completion" below) and released the lease,
  actively terminate the worker's host session (`TaskStop` for an
  in-process/teammate adapter, or the provider's documented equivalent) —
  do not leave it addressable. Do this even for an awaited foreground call
  that already returned; a resumable agent identity left alive after its
  assigned dispatch is exactly what let a prior resumption's worker keep
  acting, unprompted, well past its scope (see the recorded failure above).

Do not set the host `subagent_type` to Shark names such as `business-analyst`,
`product-manager`, or `tech-director`; those personas are already inside
`response.prompt`.

For long steps, periodically renew the parent lease:

```bash
shark heartbeat {response.entity_key} --session "$SID" --progress <0..1> --note "<step>"
```

**Progress check-ins and staleness.** The dispatched prompt (Shark's own
`PARENT LOOP OWNERSHIP CONTRACT` preamble) tells the worker to check in with
you at intervals for any dispatch likely to run longer than ~15 minutes,
rather than going silent until a single final report. Expect those
check-ins for long-running work, and treat their absence as a signal, not
just a curiosity:

- If a worker you expect to be long-running has produced no check-in and no
  terminal result for well beyond your last-known progress point (roughly
  2x the interval it should be checking in at, or ~30 minutes with no
  signal at all for a dispatch with no stated interval), check its actual
  status (`ListAgents`, or the provider's task/process listing) before
  assuming it is simply "still working." Do not silently keep waiting on a
  worker you cannot account for.
- If it is confirmed dead, hung, or unreachable: record what evidence
  exists (partial commits, evidence files, notes it left), release the
  Shark lease if still held, and re-dispatch a fresh worker with a handoff
  built from that evidence — do not resume the same identity blind.
- If it is alive but has simply gone quiet without checking in: nudge it
  once for a status update before deciding it is lost. Two silent nudges
  with no response is lost, not busy.

### Long-running Codex CLI adapter

Keep the canonical Shark prompt unchanged. The parent must pass
`response.prompt` byte-for-byte to the worker; do not append heartbeat
instructions to it. Instead, use Codex's JSONL event stream as an observable
sidecar and emit parent-owned Shark heartbeats from that stream.

For a long planning or review step, allow up to **15 minutes** and retain both
the event stream and the final message. Run the worker in the background so
the parent can poll `$EVENT_FILE` while it is still running, and keep stdout
(the JSONL event stream) separate from stderr:

For a mapped fallback, the preflight selects the documented Codex CLI adapter
and provider, then discovers the available model through that selected CLI's
`config/read` interface. Set `SELECTED_MODEL` only to the discovered model and
record the discovery evidence with the dispatch. Do not infer a model from the
request tuple or a fixed compatibility table. For an exact Codex request, pass
the exact requested model through the same explicit selector.
Make the prompt, result and event paths absolute and unique to this invocation.
Use the absolute `CODEX_BIN` resolved in preflight, its profile, environment
and authorized sandbox. An omitted profile or sandbox uses the checked host
configuration. Set `REQUESTED_MODEL` from the final keyed response, including
its empty value for a documented default, and `REQUESTED_EFFORT` from its
optional effort field.
Never derive `SELECTED_MODEL` from the parent session, a Codex configuration
default, or an environment variable. Pass every explicit selection through
Codex CLI's `--model` selector. For an evidenced default, omit that flag.
Use the producing worker's observation to establish which model ran.
Do not use Codex's provider default for a mapped Anthropic fallback.

```bash
# Use the provider selected at preflight and the model discovered through the
# selected CLI. An exact request keeps its requested model. Never substitute
# a hard-coded mapped model or use an ambient default.
CODEX_ROUTE_ARGS=(--config "model_provider=${SELECTED_PROVIDER:?}")
if [[ -n ${SELECTED_PROFILE:-} ]]; then
  CODEX_ROUTE_ARGS+=(--profile "$SELECTED_PROFILE")
fi
if [[ -n ${SELECTED_SANDBOX:-} ]]; then
  CODEX_ROUTE_ARGS+=(--sandbox "$SELECTED_SANDBOX")
fi
if [[ -n ${MAPPING_REF:-} || -n ${REQUESTED_MODEL:-} ]]; then
  CODEX_ROUTE_ARGS+=(--model "${SELECTED_MODEL:?missing preflight selection}")
fi
if [[ -n ${REQUESTED_EFFORT:-} ]]; then
  CODEX_ROUTE_ARGS+=(--config "model_reasoning_effort=$REQUESTED_EFFORT")
fi
(
  cd -- "${WORKDIR:?}" || exit 1
  printf '%s\0' "${CODEX_BIN:?}" exec -C "$WORKDIR" \
    "${CODEX_ROUTE_ARGS[@]}" --json --output-last-message "$RESULT_FILE" \
    - > "$ARGV_FILE"
  exec timeout --foreground 900 "${CODEX_BIN:?}" exec -C "$WORKDIR" \
    "${CODEX_ROUTE_ARGS[@]}" --json --output-last-message "$RESULT_FILE" \
    - < "$PROMPT_FILE"
) > "$EVENT_FILE" 2> "$EVENT_FILE.stderr" &
CODEX_PID=$!
```

For each launch, persist the exact executable and NUL-delimited argv to the
invocation's private `ARGV_FILE` immediately before `exec`; this capture is
preflight/dispatch evidence, not a substitute for worker observation. Require
the selected provider override on every launch. Require exactly one matching
model selector for a mapped request or an explicit pin. An exact default
requires its documented selection and no model override in the captured argv.

For the B077 Codex CLI route, take `worker_id` from the `thread.started`
event's `thread_id`. The event identifies the worker thread but contains no
effective model. Correlate that exact ID to a persisted Codex rollout under
`$CODEX_HOME/sessions` (or `$HOME/.codex/sessions` when `CODEX_HOME` is unset):
the rollout's `session_meta.payload.id` must equal the event's `thread_id`.
Read `effective_provider` from `session_meta.payload.model_provider` and
`effective_model` from each `turn_context.payload.model` in the turn that produced this
message. Correlate that turn to this invocation. Compare its recorded effort
independently with every explicit request. Missing or conflicting observations
within the producing turn, or an ambiguous turn join, leave it unverified.
Require both values and require them to match `SELECTED_PROVIDER` and
`SELECTED_MODEL`. Missing or ambiguous metadata is `unverified`; an observed
different provider or model is `mismatch`; conflicting records for one ID are
`unverified`. Never select the newest rollout by timestamp or use another
worker's metadata.
This identity path is Codex CLI-specific; it does not attest native Codex host
workers. If the rollout format or its configured storage path is unavailable,
reject before claim or reject the worker result as unverified, according to
the dispatch phase.

While the worker runs, inspect `$EVENT_FILE` at least once per minute. For
each new event, renew the lease and record the latest event type or item type:

```bash
shark heartbeat {response.entity_key} --session "$SID" --progress <0..1> \
  --note "worker alive; latest Codex event: <event-type>"
```

Use elapsed time to estimate progress; do not infer completion from a
heartbeat. The JSONL file is a durable, trackable worker heartbeat. Only the
worker's terminal result in `$RESULT_FILE` authorizes a workflow transition.
Wait for the worker to finish (`wait "$CODEX_PID"`) before reading
`$RESULT_FILE`. On model-access denial or another nonzero exit, ignore any final file, keep
the attempt's attributable receipt, observed identity, usage and cost (missing
fields stay unknown), stop its supervisor and release only the original
session if still held. Make zero alternate submissions or automatic retries.
If the 15-minute limit expires, record the event-log path and
its last event in a blocker note, release the lease, and do not advance the
workflow.

At the Codex completion boundary for a fresh `exec`, wait for this invocation,
then verify its event stream against its persistent rollout before either
result handler reads the file. Set `OBSERVATION_FILE` to a private path unique
to this invocation. The existing failure cleanup stops the supervisor, retires
the original handle and releases only the matching lease. Preserve the event
reference and any known failed-attempt accounting. An error or timeout ends
the dispatch even when a previous or early final message exists. For a resumed
session, independently bind the new turn using the resume adapter's own
invocation boundaries. Refuse ambiguous turn attribution.

```bash
worker_exit=0
wait "$CODEX_PID" || worker_exit=$?
python3 "$RIDER_ROOT/scripts/observe-codex-turn.py" "$EVENT_FILE" \
  "${CODEX_HOME:-$HOME/.codex}" "$ROUTE_FILE" "$ARGV_FILE" "$RESULT_FILE" \
  --worker-exit "$worker_exit" > "$OBSERVATION_FILE" || exit 1
EFFECTIVE_PROVIDER=$(jq -er .effective_provider "$OBSERVATION_FILE") || exit 1
EFFECTIVE_MODEL=$(jq -er .effective_model "$OBSERVATION_FILE") || exit 1
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
  use the session-qualified blocker note in step 3 below with the envelope's bounded evidence, then advance
  with outcome `blocked`.

### `gate_result_v1` steps — shared ingestion CLI, not directive parsing

Check `response.result_contract` (from `shark next {KEY} --json`,
T-E34-F05-005) before parsing the worker's final response. A step whose
`result_contract` is `gate_result_v1` never goes through the directive
markers below (steps 1-6) — parsing `RECOMMENDED OUTCOME:`/`PARENT NOTE:`/
kickback lines out of that worker's response, or advancing directly, would
be a second, Rider-only implementation of what `internal/runner.IngestGateResult`
already validates and persists for the core runner. Instead:

1. Establish the worker's terminal completion exactly as in step 2 below
   (awaited foreground worker, or a documented native retirement operation).
2. Write the worker's raw terminal response (the whole trimmed `kind: final`
   envelope, `gate_result` included — see
   `context/worker-control-schema.yaml`'s `example_final_gate_result`) to a
   result file.
3. Call the shared ingestion CLI surface with that file, the durable
   `run_id` this dispatch is using, and the authorized session id — the
   same call the core runner makes directly for its own synchronous
   dispatch (`context/host-adapter-contract.md`'s "`result_contract`-gated
   dispatch" section has the full field list). It validates the envelope
   and nested `gate_result` against the step's `outcome_roles`, persists
   notes/kickbacks/impacts, and applies the transition — fail closed (no
   transition) on any validation error.
   If a bug worker used a selected working directory, include the same
   `--workdir "$WORKDIR"` in the `shark run --apply-result` call so evidence
   is resolved in the selected checkout.
4. Release the lease with the ingested outcome. For a bug, if the resulting
   status equals the dispatched status, stop this invocation and report the
   unresolved stage. Otherwise return to Step 1 with the original `{KEY}`.

A step whose `result_contract` is absent or `legacy` is unaffected — continue
with the directive parsing below exactly as before.

When the worker returns, parse its final response for the directive markers
the workflow prompts emit, apply them in this order, then advance:

1. **Resolve the outcome.** Take the trailing `RECOMMENDED OUTCOME: <key>`
   line (or the `outcome` field of a JSON return). Any outcome key defined
   for the current step is valid — `shark status advance` rejects unknown
   keys, so pass the worker's key through verbatim rather than coercing it to
   pass/fail/blocked. For a missing outcome, select `blocked` and use the bounded response tail for the session-qualified blocker note in step 3.
2. **Establish terminal completion before advancing.** Apply this to every
   terminal envelope: `final`, `blocked_external`, and `failed`. An awaited
   foreground worker satisfies the check when its invocation exits after
   returning the envelope. Use a native retirement operation only if the
   provider reference documents it and it returns terminal acknowledgement.
   Otherwise, do not dispatch future steps as background agents; use
   parent-owned awaited execution. If a prior background worker has no
   documented completion guarantee, record a blocker note with bounded
   evidence, release the Shark lease, and stop. Do not advance the workflow
   or claim that later idle notifications are suppressed. See
   `context/host-adapter-contract.md` for the provider-neutral boundary.
3. **Persist notes.** Use the session that authorized this worker for each insert.
   If an insert fails, stop handling the result and perform session-scoped failure cleanup.
   - For a failed worker, external blocker or missing outcome, persist its bounded evidence or response tail:
     ```bash
     shark create note {response.entity_key} "<worker evidence>" --type blocker --session "$SID" || exit 1
     ```
   - A `COMPLEXITY NOTE: <text>` line (complexity-triage steps) must be
     stored as a decision so later steps can read the tier:
     ```bash
     shark create note {response.entity_key} "<text>" --type decision --session "$SID" || exit 1
     ```
   - A `PARENT NOTE: <text>` line records the gate result:
     ```bash
     shark create note {response.entity_key} "<text>" --type comment --session "$SID" || exit 1
     ```
   - Any other `note` the worker returned:
     ```bash
     shark create note {response.entity_key} "<note>" --type comment --session "$SID" || exit 1
     ```
4. **Apply structured review kickbacks through the coordinator.** Preserve the
   worker's full `gate_result` and its `recommended_outcome` for the existing
   `--apply-result` path. The parent-owned coordinator resolves each child
   destination from the review outcome key and the target's configured
   `rework_outcomes`; it checks Question holds and the parent lease before
   each child write, then applies the parent outcome transition. If any child
   is blocked or cannot be routed or authorized, do not advance the parent.
   Never translate a kickback into a direct child status
   command or a legacy `<task-id> -> <status>` directive. A `fail` outcome
   whose response names no kickbacks is suspicious — record a blocker note and
   surface it to the user rather than silently advancing.
5. **Advance by the resolved outcome**, attributing the transition to the
   agent that did the work (this populates `entity_history.changed_by`):
   ```bash
   shark status advance "$ENTITY_KEY" --outcome "$OUTCOME" --workdir "$WORKDIR" \
     --session "$SID" --from-status "{response.status}" \
     --agent "$(jq -er .agent_type "$REQUEST_FILE")@${EFFECTIVE_PROVIDER:?}/${EFFECTIVE_MODEL:?}"
   ```
   The outcome is workflow-configured: do not substitute a hard-coded status
   or coerce a valid semantic outcome to `pass` or `fail`. Include
   `--session`/`--from-status` on each claimed legacy advance, including when `advance_guard` is disabled
   (see the route-based workflow guide, §2).
   For bug research dispatched in a linked worktree, include `--workdir`
   with the worker's actual working directory so Shark validates the report
   in that checkout using the parent project's research policy.
6. **Release the lease, always**, stamping the outcome on the work session:
   ```bash
   shark release {response.entity_key} --session "$SID" --outcome <key>
   ```
7. **Terminate the worker's host session** (`TaskStop` or the provider's
   equivalent) now that its terminal result is ingested — see "Worker
   termination" above. Releasing the Shark lease retires the entity's
   workflow state; it does not stop the host process. Skipping this is how
   a worker outlives its assigned dispatch and keeps acting unsupervised.
8. For a bug, if the resulting status equals the dispatched status, stop this
   invocation and report the unresolved stage. Otherwise return to Step 1 with
   the original `{KEY}`.

If the worker fails or throws, still release the lease, terminate its host
session, record a blocker note if possible, and surface the failure before
deciding whether to retry.

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

### Resuming an interrupted `gate_result_v1` dispatch

If a `gate_result_v1` dispatch was interrupted after the worker returned its
terminal envelope but before the parent confirmed the ingestion call
completed, do not re-run the worker and do not resend the envelope — the
durable `run_id` sidecar (T-E34-F05-002) already holds it. Call the resume
CLI surface with that same `run_id` and the authorized session id (no new
result bytes accepted). It reports the durable resume decision and, when the
persisted state is not yet fully applied, re-ingests the stored envelope
through the same boundary the initial dispatch used — one resume
implementation shared by the core runner's own crash recovery and this
loop's recovery, not a second Rider-specific one.
