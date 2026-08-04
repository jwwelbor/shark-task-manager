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

## Capability declaration

Adapters implement the capability set dev-artifacts'
`shark-attack-v2-plan/implementation-plan.md`
(`#3-provider-neutral-adapter-contract`) defines: Spawn, Send/follow-up,
Progress/final, Wait/poll, Interrupt/list,
Isolate, Resume, Provenance. `providers/codex.md` and
`providers/claude-code.md` record which of these each installed host
actually supports, with captured evidence — this file states the field
shapes those capabilities carry, not which host supports which capability.
Capability detection precedes topology/coordination selection (REQ-F-012);
a missing capability is data that drives a documented fallback, never
license to invent an unverified provider command.

## Resume

Follow-up and resume behavior — same-worker delivery when the host
supports resume, otherwise exactly one bounded replacement worker built
from an immutable handoff — follows the shark-attack skill's
`workflows/resume.md`; this file does not restate that procedure.
