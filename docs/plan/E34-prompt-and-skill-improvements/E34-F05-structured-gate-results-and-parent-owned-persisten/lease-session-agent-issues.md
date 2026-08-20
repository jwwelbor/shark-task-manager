# E34-F05 Lease, Session, and Agent Issues

## Scope and evidence boundary

This note records the lease/session/agent problems relevant to E34-F05 as of
2026-08-17. It distinguishes repository evidence from hypotheses. It is an
investigation record, not evidence that the proposed durable protocol has been
implemented.

Evidence inspected:

- `shark get E34-F05 --json`: the feature is still `draft`, at `0%`, with one
  pending task and no status-history entries.
- `skills/shark-rider/verbs/run.md` and
  `skills/shark-rider/context/host-adapter-contract.md`.
- `internal/sharkdata/default_data/skills/shark-attack/context/authority.md`
  and `context/worker-ownership.md`.
- `docs/plan/E35-shark-2x-route-based-workflow-redesign/E35-F03-claim-session-lease/feature.md`.
- Current `.shark/runs/`: sampled run directories contain `run.log` entries
  such as `run_end ... outcome=paused ... stages=0`; no durable
  `started.json`, `heartbeat.json`, or `result.json` sidecars were present in
  the inspected run tree.

## Observed issues

### 1. A terminal worker response is not a durable completion record

The host adapter contract carries the worker's response text, `worker_id`, the
parent's `session_id`, and bounded evidence. The Rider instructions require an
awaited foreground process because the installed provider references do not
document a native retirement operation. This means a lost notification,
orchestrator restart, or provider disconnect can leave the parent with an
artifact but without a verified terminal result.

The existing run log is not sufficient recovery evidence: it records a compact
end marker but does not establish the atomic relationship among start,
heartbeat, terminal result, worker retirement, and result location.

### 2. Shark session identity and provider worker identity are different

`session_id` is the parent's Shark claim/lease identity. `worker_id` is the
provider-assigned execution identity used for follow-up or resume. They have
different owners and lifetimes, but the current workflow requires both to be
carried through the adapter boundary. If either is dropped, recovery cannot
reliably answer all of these questions:

- Which lease authorizes the next mutation?
- Which provider process produced the artifact?
- Can the same worker be resumed, or must a bounded replacement be started?
- Has the original worker retired, or is it still capable of writing?

This is the strongest session-related failure mode; it is a contract gap even
when the underlying claim store is healthy.

### 3. Lease loss is correctly fail-closed but operationally under-instrumented

The authority contract says that a heartbeat failure or expired claim must stop
mutation workers, prevent answers/integration/transitions, treat the handoff as
context only, and require a fresh keyed dispatch and claim. That protects state
integrity, but the operator needs durable evidence of *why* the lease was lost,
which worker was stopped, what artifact was left behind, and whether a
replacement is safe.

Without a bounded heartbeat/progress record and retirement marker, a stalled
worker, a dead provider process, and a lost parent notification can look the
same from the outside.

### 4. Consultations create two leases that must not be conflated

During a Question or council consultation, the parent must keep the dispatched
entity's lease alive while a separate Question lease may be opened. The parent
owns both relevant workflow authorities, but they have different entity keys,
sessions, deadlines, and release paths. A responder disappearing must not cause
the root worker to continue mutating, and a Question release must not be
mistaken for root-worker completion.

This is a likely nested-agent/session failure surface, especially after a
parent restart or replacement responder.

### 5. Resume capability is provider-specific and currently asymmetric

The Codex provider reference documents post-exit resume, but no live follow-up
to a still-running `codex exec` worker and no live-worker listing. The Claude
reference documents background-session listing but no provider-native
retirement operation. Therefore a generic adapter cannot safely promise
same-worker continuation, active-worker discovery, or cancellation across
providers.

The safe current rule is synchronous/awaited execution. Any background worker
without an independently documented retirement acknowledgement must release
the lease and stop the loop rather than advance on an idle or late notification.

## Relationship to E34-F05 and E35-F03

E35-F03 defines the lower-level lease model: claim is the lease, updates renew
the lease and carry progress, and missed updates provide a global crash
backstop. E34-F05 needs to add the execution-layer records around that model:

```text
claim/session lease
        |
        +-- started record (run + worker identity)
        +-- bounded heartbeat/progress records
        +-- one atomic terminal result
        +-- retirement acknowledgement
        +-- parent verification before advance/release
```

E35-F03 is design context here, not proof that the E34-F05 result protocol is
already implemented.

## Required contract and test cases

The implementation should make these records independently readable after a
parent restart and bind every record to the same run identity:

1. Lost final notification: artifact and terminal result remain recoverable;
   parent can verify them without chat history.
2. Heartbeat timeout: parent records lease loss, stops mutation, releases the
   claim, and does not advance.
3. Duplicate terminal result: exactly one accepted result; later writes are
   rejected or recorded as bounded conflicts.
4. Worker artifact without retirement: parent distinguishes incomplete work
   from a safe terminal result.
5. Root worker plus Question consultation: root and Question leases remain
   distinct, and neither release path can satisfy the other.
6. Parent restart: a fresh parent can correlate run, Shark session, provider
   worker, artifact, and result location.
7. Provider asymmetry: Codex post-exit resume and Claude background listing are
   represented as capabilities, not assumed operations.

## Recommended implementation boundary

Keep Shark claim, heartbeat, status transition, and release authority in the
parent. Add a bounded, atomic sidecar protocol under `.shark/runs/<run-id>/`
with explicit `run_id`, `entity_key`, `shark_session_id`, `worker_id`, phase,
nested-operation state, timestamps, retirement state, and result path. Parent
polling should consume those records independently of chat notifications and
must verify terminal result plus retirement before advancing. A replacement
worker receives an immutable handoff and a newly claimed Shark session; it must
never inherit authority from the old session merely because it can read the old
artifact.
