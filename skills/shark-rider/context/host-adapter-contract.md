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
