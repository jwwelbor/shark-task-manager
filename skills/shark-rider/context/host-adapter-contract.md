# Host adapter contract (provider-neutral)

## Scope

This file states the provider-neutral request/result field set a host
adapter (Codex, Claude Code, and future providers under I-11) exchanges
with the parent. It has no embedded mirror — `skills/shark-rider/` is never
rendered from the embedded bundle, so this file sits outside the
shark-attack skill's parity gate (AC-016). `verbs/run.md`'s exact-transport
section is the sole normative consumer; it links here instead of restating
these shapes.

## Request fields (parent -> adapter spawn / follow-up)

These are exactly `shark next <key> --json`'s wire fields
(`internal/cli/commands/next.go`'s `NextResponse`) — an adapter receives
them unchanged, never renamed or re-derived:

| Field | Meaning |
|---|---|
| `entity_key` | Concrete entity being dispatched |
| `entity_type` | Concrete entity type |
| `status` | Current workflow status |
| `action` | `spawn_agent` — the only action an adapter ever spawns a worker for |
| `agent_type` | Shark persona metadata for logging/provenance and adapter selection |
| `provider` | e.g. `anthropic`, `openai`, `codex` |
| `model` | Model to dispatch the worker with |
| `effort` | Optional reasoning-effort override |
| `prompt` | The exact prompt payload — pass verbatim to the adapter's native spawn/prompt field, no summarization or hand transcription |
| `prompt_sha256` | Hex-encoded SHA-256 digest of the exact `prompt` bytes (REQ-F-011) |
| `prompt_bytes` | Byte length of `prompt`, computed alongside `prompt_sha256` from the same materialized string |

CLI adapters that need the exact prompt bytes on disk rather than in memory
use `shark next <key> --json --prompt-out <path>`; verify the written file
against `prompt_sha256` by independent recomputation before spawning (Prompt
transport contract, dev-artifacts' `shark-attack-v2-plan/implementation-plan.md`,
`#3-provider-neutral-adapter-contract`). Store only hash, entity key, worker
identity, and timestamps as provenance — never the rendered prompt
(AC-024).

## Result fields (adapter -> parent)

An adapter returns a worker/session identity plus the worker's response
text. The parent reads that text as one control envelope
(`context/worker-control-schema.yaml`,
`kind: final|question|needs_council|blocked_external|failed`) — this file
adds no second envelope shape:

| Field | Meaning |
|---|---|
| `worker_id` | Provider-assigned worker/agent identity, retained for follow-up/resume — never a rendered prompt or transcript |
| `session_id` | The parent's own claim/session identity for the dispatched entity (`shark claim ... --field session_id`) — a Shark concept, not a provider one |
| `kind` | `final`, `question`, `needs_council`, `blocked_external`, or `failed` — see `context/worker-control-schema.yaml` |
| `recommended_outcome` | Present only when `kind: final`; opaque, passed to `shark status advance --outcome` byte-for-byte |
| `evidence` | Bounded evidence references; never the rendered prompt or a transcript |
| `gate_result` | Present on `kind: final` only when the dispatched step's `result_contract` (from `shark next <key> --json`) is `gate_result_v1` — the I-02 GateResult v1 nested payload; see `context/worker-control-schema.yaml`'s `example_final_gate_result` |

## Host route attestation (ephemeral adapter metadata)

The parent/adapter evaluates route attestation before claim and spawn, then
records post-dispatch identity separately when a supported route launches. It
does not change the `shark next` request or add fields to the worker control
envelope. A pre-dispatch rejection is parent/adapter metadata, not a worker
control envelope or worker result; route attestation itself is not a worker
control envelope. Keep the Shark persona (`agent_type`), host
worker class, provider/model, and reasoning effort as distinct values:

| Field | Meaning |
|---|---|
| `route_status` | `verified`, `unsupported`, `unverified`, or `mismatch`, with phase-specific semantics below; `verified` and `mismatch` compare observations with the selected documented route |
| `requested_provider` | Exact `provider` value from `shark next <key> --json` |
| `requested_model` | Exact `model` value from `shark next <key> --json`; retain an empty value when no override was requested |
| `selected_provider` | Provider explicitly selected by the host route policy; absent when no supported route is selected |
| `selected_model` | Model explicitly selected by the host route policy and passed through its selector; never inferred from the parent or copied into effective fields |
| `host_worker_class` | Host execution adapter selected for this worker (native host or CLI); never the Shark persona or model |
| `effort` | Exact requested reasoning-effort setting, kept separate from provider/model selection |
| `effective_provider` | Provider independently observed for this worker; present for `verified` and `mismatch`, absent for `unverified` |
| `effective_model` | Model independently observed for this worker; present for `verified` and `mismatch`, absent for `unverified` |
| `worker_id` | Provider-assigned worker identity used to correlate the launch request with host session metadata |
| `host_session_id` | Optional, distinctly named host session identity when the host exposes it or it is needed for correlation |
| `evidence_ref` | Bounded reference to the captured host capability and identity evidence; never a prompt or transcript |

The adapter prefers the exact requested provider/model route. For an Anthropic
request, Claude Code is preferred when available, but its absence alone is not a
rejection reason. A host may select a fallback only through an explicit,
documented compatibility mapping and selector. `selected_provider` and
`selected_model` record that selection separately from the request and the
worker's later observed identity. `evidence_ref` identifies the mapping and the
evidence for the selected route. A workflow-level compatibility mapping does
not claim semantic equivalence between the requested and selected models.
Never infer compatibility from similar model names, effort, Shark persona,
worker class, parent model, or a provider configuration default. If no exact
route or documented compatible fallback can be selected and its effective
provider/model observed for the worker, reject before claim. Preserve the
requested fields exactly even when the selected route is a fallback.

When `requested_model` is empty, keep it empty. A provider default is a usable
route only when the host reference documents that default and the adapter can
independently observe the worker's concrete effective model. The resulting
`effective_model` records that observed value; it is never inferred from the
empty request. If the provider default or its identity cannot be evidenced,
reject before claim.

`session_id` is exclusively the parent's Shark claim/session ID from
`shark claim ... --field session_id`; it is not a provider or host identity.
Never use `session_id` as host identity or correlation. Correlate host evidence
through `worker_id` and, when needed, the distinct `host_session_id`. Never
copy requested or selected values into effective fields to imply verification.
`effort` is the exact requested value when present and remains separate from
route selection.

Route attestation is ephemeral adapter metadata and does not expand the
durable provenance fields above. The phase/status matrix is normative. Every
row's `evidence_ref` is a bounded reference to host capability evidence or a
specific evidence gap; it never points to a prompt or transcript. For
preflight rows, the optional `effort` field is present only when the dispatch
requested an effort override.

### Phase/status matrix

| Phase | route_status | Required fields | Forbidden fields | Evidence | Launch disposition |
|---|---|---|---|---|---|
| preflight | unsupported | `route_status`, `requested_provider`, `requested_model`, `host_worker_class`, `evidence_ref` | `selected_provider`, `selected_model`, `worker_id`, `host_session_id`, `effective_provider`, `effective_model` | Evidence proves no exact host selector, documented provider default, or documented compatible fallback exists. | Reject before claim; do not spawn. No worker result, no claim, and no status advance; parent stops and reports. |
| preflight | unverified | `route_status`, `requested_provider`, `requested_model`, `selected_provider`, `selected_model`, `host_worker_class`, `evidence_ref` | `worker_id`, `host_session_id`, `effective_provider`, `effective_model` | An explicit selected route exists, but compatibility, selector, default, or identity-capture evidence is missing or inconclusive. | Reject before claim; do not spawn. No worker result, no claim, and no status advance; parent stops and reports. |
| post-dispatch | verified | `route_status`, `requested_provider`, `requested_model`, `selected_provider`, `selected_model`, `host_worker_class`, `worker_id`, `effective_provider`, `effective_model`, `evidence_ref` | none | An explicit selector chooses the requested route or documented compatible fallback, and worker-correlated observed provider and model match the selected provider and model. If `requested_model` is empty, the documented provider default is selected and its concrete effective model is observed. | Launch permitted after preflight; a worker result may enter its existing envelope path. |
| post-dispatch | unverified | `route_status`, `requested_provider`, `requested_model`, `selected_provider`, `selected_model`, `host_worker_class`, `worker_id`, `evidence_ref` | `effective_provider`, `effective_model` | The selector was used but worker-correlated observation is missing or inconclusive; do not infer effective fields. | Reject the worker result, stop the worker when supported, and do not advance. |
| post-dispatch | mismatch | `route_status`, `requested_provider`, `requested_model`, `selected_provider`, `selected_model`, `host_worker_class`, `worker_id`, `effective_provider`, `effective_model`, `evidence_ref` | none | Worker-correlated observed provider or model differs from the selected provider or model; retain both observed fields to show the mismatch. | Reject the worker result, stop the worker when supported, and do not advance. |

`unsupported` means capability evidence establishes there is no usable exact,
default, or documented compatible host route; no selected identity fields are
present. `unverified` in preflight means the adapter selected a route under an
explicit policy, so both selected fields are present, but required selector or
identity-capture evidence is absent or inconclusive. Both preflight states
reject before claim or spawn; neither has a worker control envelope/result,
claim, or status advance. A post-dispatch `unverified` state records the
selected route and launched worker identity but has no independently observed
effective fields. `mismatch` means either the independently observed provider
or model differs from the selected provider or model; retain both effective
fields so the discrepancy is inspectable. A documented fallback can be
`verified` even when its effective values differ from the request, because the
request and selected mapping are both preserved in the route evidence. Only
`verified` permits the worker result to continue through the existing
control-envelope path. The Shark worker envelope schema remains unchanged.

## Lease lifetime during external execution (B075)

The parent owns the lease for the complete external execution lifecycle: from
claim, through worker execution and consultation, until result application or
failure cleanup. The worker never claims, heartbeats, releases, or transitions
the entity. Before spawning a long-running worker, the parent starts a lease
supervisor using the same `session_id` and renews at `max(TTL/3, 1 second)`;
the default TTL is 15 minutes, but project configuration wins.

If renewal fails, the parent stops the worker and must never apply the result
or deliver its handoff under the lost authority. The handoff is context only:
re-claiming under the old result or session is not an authorized recovery.
Recovery is a fresh keyed dispatch and a new successful claim. The parent may
attempt a session-scoped release for cleanup, but a release that finds the
session gone or reissued must not be treated as permission to write.

When the worker returns a terminal result, the parent sends one final
heartbeat. A failed final heartbeat means never apply; a successful one allows
the parent to stop the periodic supervisor and immediately call the shared
`--apply-result` boundary. That boundary performs its own active-session
re-check, so expiry or reclaim between the final heartbeat and persistence
still fails closed with zero writes.

### `result_contract`-gated dispatch (T-E34-F05-004)

`shark next <key> --json` additionally exposes `result_contract`
(`legacy` or `gate_result_v1`) and, when set, `outcome_roles` — the
per-outcome semantic role map the parent's ingestion boundary validates
against. A step whose `result_contract` is absent or `legacy` keeps the
existing free-form directive handling (`verbs/run.md`'s exact-transport
section); the adapter never needs to distinguish it further.

A step whose `result_contract` is `gate_result_v1` requires the adapter to
route the worker's terminal envelope through the shared ingestion CLI
surface instead of interpreting it directly:

- Write the worker's raw terminal response (the whole trimmed `kind: final`
  envelope, `gate_result` included) to a file, then invoke the ingestion
  command with that file, the durable `run_id`, and the authorized
  `session_id` — the same boundary (`internal/runner.IngestGateResult`) the
  core runner calls directly for its own synchronous dispatch. There is no
  second, Rider-only parser for this envelope shape.
- On adapter-side recovery of an interrupted dispatch, the resume command
  (same durable `run_id` and `session_id`, no new result bytes) re-ingests
  whatever envelope was already durably recorded rather than asking the
  worker to resend it — see `verbs/run.md`'s recovery section.
- Either call fails closed (no transition applied) on an absent or
  malformed envelope; the adapter must surface that failure to the parent
  rather than falling back to free-form directive parsing for a
  `gate_result_v1` step.

#### The ingestion command, concretely (TD-230)

The "shared ingestion CLI surface" above is one flag on the existing `run`
verb, not a separate command:

```
shark run <entity-key> --apply-result <path-to-envelope-file> --run-id <run-id> --session <session-id>
```

All three flags are required together. Omitting `--session` fails with
`--apply-result requires --session=<authorized-session-id>` before the file
is even read. `--session` must name the **currently active** claim/lease
session on `<entity-key>` — not merely a non-empty string — or the call fails
closed with zero writes (`apply-result authorization failed: ...`); this is
checked before the envelope file is opened.

`run-id` is a **caller-minted, opaque string** for this initial-ingestion
call — there is nothing to look up, and `shark next --json` returns no
`run_id` field to reuse. It becomes this gate stage's persistence key under
`.shark/runs/<run-id>/`, which is exactly what a later `--resume-run` (below)
needs to find the durable result again. Pick something durable and traceable
to the dispatch, e.g. `<entity-key>-<status>-<unix-timestamp>`; any string
accepted by the identity allowlist (letters, digits, `-`, `_`, `.`) works.

**Worked example.** Uses the shipped `tech-debt` workflow's `in_progress`
step (`shark-data/workflow/tech-debt.yaml`), which declares
`result_contract: gate_result_v1`, `outcomes: {pass: resolved, ...}`, and
`outcome_roles: {pass: success, ...}` — `resolved` is that workflow's
terminal status. (That step's YAML `action` is `check_or_resume`, not
`spawn_agent` — the CLI normalizes both to the wire `action: spawn_agent`
this file's Request-fields table describes, so `shark next --json` never
surfaces the distinction to an adapter.) The worker's entire trimmed stdout
is one `kind: final` envelope; write it verbatim to a file:

```json
{
  "kind": "final",
  "recommended_outcome": "pass",
  "evidence": [
    {
      "kind": "file",
      "pointer": "internal/gateresult/gateresult.go",
      "summary": "Reviewed bounds and role validation"
    }
  ],
  "gate_result": {
    "schema_version": 1,
    "summary": "No blocking findings; two minor style notes filed as follow-ups.",
    "findings": [
      {
        "severity": "minor",
        "class_key": "unused-import",
        "class_statement": "Leftover import after refactor",
        "fingerprint": "gateresult-role-go-l1-unused-import",
        "disposition": "fixed"
      }
    ],
    "kickbacks": []
  }
}
```

`EvidenceRef` (outer envelope, per-item — `internal/workercontrol/envelope.go`)
takes `kind`/`pointer` required, `summary`/`command`/`working_directory`/
`exit_code` optional; a bare string array (`"evidence": ["path/to/file.md"]`)
is rejected as a shape violation. `gate_result.findings[].severity` and
`.disposition` are bounded free text/closed-enum respectively
(`disposition` must be one of `open`, `fixed`, `already_dispositioned`,
`severity_conflict`, `not_reproducible`); a `success`-role outcome (like
`pass` above) must carry zero `kickbacks` and no `open`/`severity_conflict`
finding left blocking.

Then invoke:

```
shark run TD-231 --apply-result /tmp/gate-result.json --run-id TD-231-in_progress-1758210000 --session a1b2c3d4
```

Expected output shape (`applyResultOutput`):

```json
{
  "run_id": "TD-231-in_progress-1758210000",
  "entity_key": "TD-231",
  "entity_type": "tech_debt",
  "outcome_key": "pass",
  "role": "success",
  "to_status": "resolved",
  "transitioned": true,
  "status": {
    "run_id": "TD-231-in_progress-1758210000",
    "entity_key": "TD-231",
    "persistence_state": "transition_applied",
    "retirement_state": "retired",
    "elapsed_seconds": 0.42,
    "result_location": ".shark/runs/TD-231-in_progress-1758210000/result.json",
    "completed_suboperation_count": 6
  }
}
```

`transitioned: false` is not a failure — it is the idempotent case, seen when
the entity is already at the resolved target status (a retried call after a
prior success).

**`gate_result.summary` is capped at 1000 UTF-8 bytes** (`gateresult.SummaryMaxBytes`),
and every free-text field in the payload (`summary`, `class_statement`,
`no_kickback_reason`, `change_summary`, …) shares that same bound. Exceeding
it fails the *entire* ingestion — nothing is persisted — with an error of the
form `<field> (bounds): must contain 1 through 1000 UTF-8 bytes`. There is no
truncate-and-warn path and no way for a worker to self-check byte count
before submitting; a dispatch prompt asking for a "compact 2-4 sentence
verdict" has still been observed going ~2x and, separately, 27 bytes over the
cap. Budget for this explicitly rather than trusting a sentence-count
instruction alone.

**Lease release is deterministic, not "sometimes."** `--apply-result`
re-ingests once more with `RetirementConfirmed`/`RunConcluded` set **only
when the resolved `to_status` is terminal** for that entity's workflow
level. Concretely:

- Terminal `to_status` (e.g. `resolved`, `completed`) → this call releases
  the lease itself. A subsequent `shark release` on the same session
  correctly reports "No matching claim to release" — the lease is already
  gone, not an error.
- Non-terminal `to_status` (e.g. a `code_review` step routing back into
  `specification`) → the lease is **not** released by `--apply-result`. The
  adapter must call `shark release <entity-key> --session <session-id>
  --outcome <key>` itself afterward.

A belt-and-braces `shark release` call after every `--apply-result` is
therefore safe either way: `ClaimService.Release` is idempotent (a second
release against an already-closed session returns `released: false`, not an
error).

#### Resuming an interrupted ingestion (`--resume-run`)

If the dispatch was interrupted after the worker returned its terminal
envelope but before the adapter confirmed `--apply-result` completed, do not
re-run the worker and do not resend the envelope bytes — the durable
`run_id` sidecar already holds them:

```
shark run <entity-key> --resume-run <run-id> --session <session-id>
```

This never accepts new result bytes. It reports the durable resume decision
(`resume_action`: `already_transitioned` | `resume_transition` |
`resume_next_operation`) and, when the persisted state is not yet fully
applied, re-ingests the exact same durably-stored envelope through the same
`internal/runner.IngestGateResult` boundary — never a second implementation.
It also applies the same terminal-vs-non-terminal lease-release rule above,
for every resume action including `already_transitioned` (which performs no
repeat write, but still verifies the entity's live status against the
recorded target and releases the lease if not already released).

**Which `run_id` to pass depends on who dispatched the stage:**

- **Adapter-driven dispatch** (this file's own `--apply-result` call, above):
  pass back the exact same caller-minted `run_id` you invented for that call.
- **Core-runner-driven dispatch** (`shark run` itself ran the
  `gate_result_v1` stage synchronously, e.g. inside a cascade): the runner
  mints its own per-stage id as `<invocation-run-id>-<entity-key>-g<stage-iteration>`
  (`internal/runner/controller.go`'s `gateStageRunID`). Do not reconstruct
  this string by hand — entity-key interpolation was added later specifically
  to avoid collisions between cascade siblings dispatching the same stage
  number, so an adapter that reassembles the format from memory can compute
  a stale shape. Read the authoritative value back from that stage's own
  status output instead — `shark run`'s own per-stage JSON result carries it
  at `stages[].gate_status.run_id` (`RunResult.Stages[].GateStatus`,
  `internal/runner/controller.go`) — or, for the adapter-driven case above,
  `applyResultOutput.run_id` (the JSON shown above). Neither is the bare
  invocation-level run_id `shark run` logs at start.

## Capability declaration

Adapters implement the capability set dev-artifacts'
`shark-attack-v2-plan/implementation-plan.md`
(`#3-provider-neutral-adapter-contract`) defines: Spawn, Send/follow-up,
Progress/final, Wait/poll, Interrupt/list, Isolate, Resume, Provenance.
`providers/codex.md` and
`providers/claude-code.md` record which of these each installed host
actually supports, with captured evidence — this file states the field
shapes those capabilities carry, not which host supports which capability.
Capability detection precedes topology/coordination selection (REQ-F-012);
a missing capability is data that drives a documented fallback, never
license to invent an unverified provider command.

## Terminal worker policy

`final`, `blocked_external`, and `failed` are terminal control envelopes.
Before the parent advances the corresponding Shark step, it needs one
documented completion guarantee for the native worker:

- An awaited foreground invocation exits after producing the envelope. Its
  process exit is the terminal acknowledgement.
- A provider may supply a documented worker-retirement operation and terminal
  acknowledgement. Use that operation only when the provider reference
  captures it.

Neither installed provider reference currently documents the second option.
When an adapter cannot establish it, it must not dispatch the step as a
background agent. Use parent-owned, synchronous or otherwise awaited execution
instead. If a previously backgrounded worker produces a terminal envelope
without a documented completion guarantee, record bounded evidence, release the
lease, and stop the Rider loop without advancing it. Do not claim that later
idle notifications have been suppressed.

## Resume

Follow-up and resume behavior — same-worker delivery when the host
supports resume, otherwise exactly one bounded replacement worker built
from an immutable handoff — follows the shark-attack skill's
`workflows/resume.md`; this file does not restate that procedure.
