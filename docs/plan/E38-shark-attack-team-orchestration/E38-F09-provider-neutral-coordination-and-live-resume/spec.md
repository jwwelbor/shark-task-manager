---
feature_key: E38-F09-provider-neutral-coordination-and-live-resume
epic_key: E38
title: Provider-Neutral Coordination and Live Resume
status: proposed
supersedes: pre-rewind specification pass (bespoke QuestionControl design, 2026-07-30)
---

# E38-F09 Provider-Neutral Coordination and Live Resume

This specification is incremental over the [epic PRD](../epic.md) — especially §2
(goals, success criteria, locked design decisions), §3 (scope, in/out boundaries,
team structure) — and the [epic architecture](../architecture.md), especially
ADR-003 (canonical child dispatch), ADR-005 (worker vs. root ownership), ADR-007
(`shark-attack` is a recipe, not a second engine), and §4.5 (council
communication contract). Feature research is in
[research-report.md](research-report.md); its Capability map governs what this
feature reuses, extends, and refuses to re-implement.

> **Supersession notice (2026-08-01).** The pre-rewind specification and task
> pass designed a bespoke `QuestionControl` / `ControlEnvelope` question store
> (see research-report Findings #5). E39 shipped a first-class Question
> lifecycle on 2026-07-31 (PR #145). Per the owner decision recorded in
> `feature.md`'s 2026-08-01 amendment, that bespoke design is **superseded**.
> No prior E38-F09 task constitutes delivered coverage for any requirement
> below, and the unmerged branch `feature/E38-F05-council-artifacts` must not be
> merged or resumed as-is.

> **Degraded upstream notice.** Two upstream producers this feature's
> interaction map names (F08 via I-08, F05 via I-09) report Shark status
> `completed` but have **no implementation reachable from `main`**. This
> specification is written to be buildable on the *actual* trunk, not on the
> declared one. See §"Degraded upstream dependencies" and REQ-F-018.

---

## Requirements

### Functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-F-001 | The skill MUST define coordination-level selection (`Direct`, `Batch`, `Council`) with evidenced entry thresholds, selected **independently** of execution topology. Default is `Direct` unless a Batch/Council threshold is evidenced. | Epic §3 in-scope (provider-neutral question routing); plan §2 Coordination levels; I-10 |
| REQ-F-002 | The skill MUST define execution-topology selection (`Sequential`, `Parallel with ownership`, `Parallel with isolation`) selected independently of coordination level. `Sequential` is the default; a parallel topology requires captured, recorded ownership or isolation evidence, and MUST degrade to `Sequential` whenever that evidence cannot be produced. | GATE-F09-005; plan §2 Execution topologies; epic §3 |
| REQ-F-003 | A dispatched worker MUST return an ephemeral control envelope with `kind` ∈ {`final`, `question`, `needs_council`, `blocked_external`, `failed`}. For `kind: final` the envelope carries `recommended_outcome` **verbatim and opaque** — any outcome key configured by the active workflow (`pass`, `simple`, `standard`, `deep_verify`, …) passes through unchanged and is never coerced into control-envelope vocabulary. | Plan §2 Live question-and-resume protocol; I-07; `skills/shark-rider/verbs/run.md` lines 42, 158 |
| REQ-F-004 | On `kind: question`, the parent MUST materialize the question as an **E39 Question entity**: create `Q###`, configure its resolution owner and responders, and link it to the dispatched entity with a `question_blocks` edge. F09 MUST NOT introduce a parallel question, responder, handoff, or resolution store. | X-06; research-report Decisions; E39 architecture §2–§4 |
| REQ-F-005 | Responder routing MUST use E39's serial dispatch (`shark next Q### --json`). Because workers never execute Shark mutation commands, the **parent** transcribes the responder worker's returned answer into `shark question respond` under the parent-held `Q###` lease. | REQ-F-007; ADR-005; `internal/services/question_workflow_service.go:116` |
| REQ-F-006 | Question closure MUST use E39's authoritative resolution (`shark question resolve` with `--resolution-kind` and `--resolution-pointer`), after which the `question_blocks` predicate no longer qualifies and the dispatched entity becomes advanceable again. `withdraw` and `supersede` are the non-resolution exits. | X-06; `internal/services/question_blocker.go:86` |
| REQ-F-007 | During consultation the parent MUST keep the dispatched entity's claim alive by heartbeat, MUST bound the consultation by the remaining claim lease, and MUST NOT advance or set the dispatched entity's status. | Epic §2 ownership integrity; ADR-005; I-07 |
| REQ-F-008 | Where the host natively supports it, the answer MUST be delivered to the **same still-live worker** as a follow-up. Where it is not supported, the parent MUST construct a bounded immutable handoff and start exactly **one** replacement worker. Capability, not assumption, decides which path runs. | Plan §2; plan §3 Resume capability; I-10 |
| REQ-F-009 | If a responder is silent, fails, or disappears, the parent MUST ping once, interrupt/cancel the stale consultation where supported, and route to one replacement responder. If no qualified responder returns before the consultation deadline, the parent MUST stop write workers, record a bounded unresolved handoff, record the blocker, and release the lease. A claim MUST NOT be held indefinitely. | Plan §2; epic §2 escalation clarity |
| REQ-F-010 | On heartbeat failure or lease loss, the parent MUST immediately stop active mutation workers and MUST NOT deliver a council answer, integrate changes, or transition the entity under the lost authority. Resume requires a fresh keyed dispatch and a successful claim; the handoff supplies context but never restores authority. | Plan §2; epic §2 claim safety; ADR-005 |
| REQ-F-011 | Keyed `shark next <key> --json` MUST emit `prompt_sha256` (hex SHA-256 of the exact prompt bytes) and `prompt_bytes` (byte length), and MUST accept `--prompt-out <path>` writing the exact UTF-8 prompt bytes with **no trailing newline**. `--field prompt` remains documented as not byte-exact. | Plan §5 Tranche C; I-10; research-report Capability map |
| REQ-F-012 | Adapters MUST perform capability discovery **before** topology selection. A missing spawn, follow-up, interrupt, isolation, or resume capability is data that causes a documented fallback; it MUST NOT authorize inventing an unverified provider command. | Plan §3; epic §3 out-of-scope (provider adapters remain host concerns) |
| REQ-F-013 | F09 MUST ship provider references for **Codex and Claude Code only**, each backed by captured installed-host evidence. Any capability not verified on an installed host is recorded as unsupported. Copilot and Antigravity are F10's scope. | GATE-F09-004; F10 `feature.md`; I-10 → I-11 |
| REQ-F-014 | The roster schema MUST accept an additive, provider-neutral `capability_profile` ∈ {`fast`, `balanced`, `deep`} with optional `requirements` (tools, context, messaging, isolation). Existing `model_tier` MUST remain accepted, emit a deprecation warning, and carry **no provider mapping**. Absent preference maps silently to the host default. Neither field grants selection, claim, or status authority. | GATE-F09-001; epic §3 out-of-scope (tier preferences are not authority) |
| REQ-F-015 | `skills/shark-attack/SKILL.md` MUST become a minimal router (invariants, two axes, selection, parent loop, question loop, links), with operating model, authority, and executable per-mode workflows in dedicated files. No rule may be stated in two places. | Plan §5 Tranche C; epic §2 skill ownership |
| REQ-F-016 | A deterministic parity gate MUST make byte-drift between the authored `skills/shark-attack/` tree and the embedded `internal/sharkdata/default_data/skills/shark-attack/` tree impossible, and MUST refuse unexpected embedded-only files. The gate ships in Go so it runs inside `make test`. | Plan §5 Tranche C; plan §8 #10; research-report Findings #4 |
| REQ-F-017 | Worker-owned child pulling MUST be excluded from the v2 normal path. `skills/shark-attack/workflows/pull-by-role.md` is retired into a compatibility reference; the Rider re-entry rule (selector supplies a key; `shark next <key>` supplies the dispatch) remains the only sanctioned path. | GATE-F09-002; plan §8 #6; research-report Capability map |
| REQ-F-018 | F09 MUST NOT assume any I-08 or I-09 payload element is present on trunk. For each element F09 depends on, the specification MUST state the dependency, its verified trunk state, and F09's own coverage. | Research-report Findings #1; §"Degraded upstream dependencies" below |

### Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| REQ-NF-001 | No new AI runtime, scheduler, claim store, credential store, second workflow engine, or second lifecycle store may be introduced. | Architecture review; epic §3 out of scope |
| REQ-NF-002 | The behavior of ordinary single-entity `shark run` and `internal/runner/*` MUST be unchanged. F09 adds no Go interface, field, or branch under `internal/runner/`. | `go test ./internal/runner/...` unchanged; diff review |
| REQ-NF-003 | Provenance MUST store only prompt hash, byte length, entity key, provider worker/session identity, and timestamps. The rendered prompt MUST NOT be persisted as council evidence, artifact content, or telemetry payload. | AC-024; `tests/contracts/e38_f09_interactions_test.go#TC-002` and `#TC-010`; existing `prompt_bytes` OTel attribute remains a span attribute only |
| REQ-NF-004 | Question and evidence text MUST reject credentials, tokens, rendered prompts, and transcripts. F09 relies on E39's existing `ValidateQuestionBoundedText` denylist (`api_key`, `password=`, `authorization:`, `bearer `, `system prompt`, `user prompt`, `assistant:`) rather than adding a second validator. | `internal/models/question.go:190`; contract test |
| REQ-NF-005 | `--prompt-out` bytes MUST hash to the emitted `prompt_sha256` under an adversarial fixture containing newlines, code fences, quotes, Unicode, and shell metacharacters. | Adversarial byte fixture test |
| REQ-NF-006 | The skill restructure and its replacement contract tests MUST land in a **single** change. Five test files pin exact paths and verbatim prose of files this feature renames or retires: `tests/contracts/e38_f04_interactions_test.go`, `tests/contracts/e38_f07_interactions_test.go`, `internal/sharkdata/shark_attack_workflows_test.go`, `internal/sharkdata/shark_attack_pull_test.go`, and `internal/sharkdata/shark_attack_test.go`; any intermediate state fails the gate. | Plan §5 Tranche C sequencing constraint; CI |
| REQ-NF-007 | Prompt hashing MUST add no measurable latency to keyed dispatch (single SHA-256 pass over an already-materialized string, computed once per response). | Benchmark or reasoned review |

### Acceptance criteria

| ID | Testable criterion |
|---|---|
| AC-001 | A scenario table classifies coordination level and execution topology **independently**: at least one case yields `Direct`+`Sequential`, one `Batch`+parallel-research-then-sequential-writes, and one `Council`+mixed. Changing one axis in a fixture does not change the other. |
| AC-002 | With no ownership or isolation evidence recorded, a parallel-topology request resolves to `Sequential`. |
| AC-003 | A control envelope carrying `recommended_outcome: deep_verify` (a key absent from the control vocabulary) round-trips to the parent's `shark status advance --outcome deep_verify` unchanged. |
| AC-004 | A `kind: question` envelope produces exactly one `Q###` with a `question_blocks` edge to the dispatched entity, and **zero** new bespoke question/handoff/resolution records. A `question_blocks` edge created while the Question is still `draft` does **not** block the target, proving the configure-before-link ordering is enforced by the fixture rather than by prose. |
| AC-005 | While that Question is `open` or `answering`, `shark next <dispatched-key> --json` returns `action: pause` with a populated `question_block`, and `shark status advance <dispatched-key>` is rejected as question-blocked. |
| AC-006 | The Rider loop uses `shark status advance` (gate-enforcing) on the dispatched entity. `shark status set` — which by design does **not** check the question gate — is documented as a human escape hatch and appears on no Rider path. |
| AC-007 | `shark next Q### --json` returns a `spawn_agent` dispatch naming only the **current** pending responder; a competing keyed-next against a live claim collapses to `pause`. |
| AC-008 | The parent records the responder worker's answer via `shark question respond --session <parent-sid> --responder <identity>`; a response submitted without the matching parent lease is rejected. |
| AC-009 | After `shark question resolve`, the `question_blocks` predicate no longer qualifies and the dispatched entity advances normally. |
| AC-010 | During consultation, heartbeats renew the dispatched entity's lease and the entity's status and history are byte-unchanged. |
| AC-011 | Simulated lease loss mid-consultation stops mutation workers and blocks answer delivery, integration, and transition; recovery requires fresh keyed dispatch plus claim. |
| AC-012 | A silent responder is pinged once, replaced once, then — if still unanswered before the deadline — produces a bounded unresolved handoff, a recorded blocker, and a released lease. No lease outlives its TTL. |
| AC-013 | For an adversarial prompt fixture, `shark next <key> --prompt-out <path>` writes bytes whose SHA-256 equals the response's `prompt_sha256`, and `prompt_bytes` equals the file size. `--field prompt` differs by exactly one trailing newline. |
| AC-014 | A roster using `capability_profile: deep` validates. A roster using legacy `model_tier: opus` validates **with a deprecation warning** and produces no provider mapping and no authority change. A roster omitting both validates silently. |
| AC-015 | Provider references for Codex and Claude Code each declare supported operations, unsupported operations, and the sequential fallback, and each cites captured installed-host evidence. A capability with no captured evidence is recorded as unsupported. |
| AC-016 | Byte-drift introduced into either `skills/shark-attack/` or the embedded mirror fails the parity gate; running the sync helper then the check is clean. An embedded-only file not present in the authored tree fails the gate. |
| AC-017 | No rule in the restructured skill appears in two files; a fresh agent given only `SKILL.md` reaches the correct workflow for a Direct, a Batch, and a Council scenario. |
| AC-018 | `pull-by-role.md` no longer describes a sanctioned normal-path claim; the Rider re-entry rule is the only claim path in the rendered corpus. |
| AC-019 | The skill restructure PR leaves `tests/contracts/e38_f04_interactions_test.go` and `e38_f07_interactions_test.go` green in the same commit — no intermediate red state. |
| AC-020 | `go test ./internal/runner/...` passes unchanged and the diff touches no file under `internal/runner/`. |
| AC-021 | Against one shared fixture: a host declaring **resume supported** delivers the answer to the *same* worker identity and creates **zero** new workers; a host declaring **resume unsupported** creates **exactly one** replacement worker from a bounded immutable handoff. The handoff carries entity key, question, answer, and evidence pointers — and **no rendered prompt**. |
| AC-022 | Capability discovery precedes topology selection: a fixture in which isolation is undetected resolves to `Sequential` **without attempting an isolation command**. The same holds for undetected follow-up (forces replacement) and undetected interrupt (forces deadline-only expiry). |
| AC-023 | Where interrupt is supported, a stale consultation is cancelled before the replacement responder is routed, and cancelling changes no Shark state. Where interrupt is unsupported, the documented fallback runs and no unverified provider command is issued. |
| AC-024 | A provenance record contains only prompt hash, byte length, entity key, provider worker/session identity, and timestamps. No council artifact, handoff, decision, note, or telemetry payload in the run contains the rendered prompt text. |

### Out of scope

- **Any change under `internal/runner/`.** See Key technical decision D-002.
- Copilot and Antigravity adapters (F10 via I-11); the complicated-lifecycle release qualification run (F11).
- Removal of the embedded `shark-attack` bundle. GATE-F09-003 records an owner
  preference for client-skill-only distribution, but that removal retires the
  X-05 contract owned by completed F04. See the **Consumes X-05** row.
- The council artifact model, repository, service, and `shark admin council`
  tooling (I-09 / F05 scope) — F09 consumes them if present and degrades if not.
- Repairing the remaining I-08 integrity items F09 does not depend on
  (`shark plan` read-only semantics, tech-debt related-docs routing, research
  validator rule statement, tech-debt gate ownership, nested-worktree
  regression). See §"Degraded upstream dependencies".
- Reconciling the F05/F08 Shark-status-versus-trunk mismatch. F09 surfaces it;
  the owner adjudicates it.
- Any new database table, column, or migration.

---

## Architecture

### Component changes

**Go production code — exactly one file changes.**

| File | Change |
|---|---|
| `internal/cli/commands/next.go` | Add `PromptSHA256 string \`json:"prompt_sha256,omitempty"\`` and `PromptBytes int \`json:"prompt_bytes,omitempty"\`` to `NextResponse` (currently next.go:140-183). Populate both immediately after `assembleDispatchPrompt` at next.go:619-623, so the hash covers the final assembled payload including the ownership preamble and agent body. Register a `--prompt-out <path>` flag alongside the existing `--sequential` flag (next.go:257-264) writing `resp.Prompt` bytes with no trailing newline. Update `nextCmd.Long` to include `question` in the documented `entity_type` set (currently omitted despite working). |

**Go test code (new).**

| File | Purpose |
|---|---|
| `tests/contracts/e38_f09_interactions_test.go` (new) | TC-001…TC-009. The I-10 producer contract test and the X-06 consumer-activation test. |
| `internal/cli/commands/next_provenance_test.go` (new) | Adversarial byte fixture for REQ-NF-005 / AC-013. |
| `internal/sharkdata/shark_attack_parity_test.go` (new) | REQ-F-016 parity gate: walks both trees, byte-compares, refuses embedded-only files. |

**Skill content — authored tree, mirrored byte-for-byte into the embedded tree.**

| File | Change |
|---|---|
| `skills/shark-attack/SKILL.md` | Reduce to a router: invariants, two axes, selection, parent loop, question loop, links. |
| `skills/shark-attack/context/operating-model.md` (new) | Full coordination-level and topology rules, thresholds, degradation. |
| `skills/shark-attack/context/authority.md` (new) | Parent/worker/council authority and category→role routing. |
| `skills/shark-attack/context/roster-schema.yaml` | Add `capability_profile` + `requirements`; retain `model_tier` as deprecated preference. |
| `skills/shark-attack/context/worker-control-schema.yaml` (new) | The ephemeral control envelope (REQ-F-003). Explicitly not a durable file. |
| `skills/shark-attack/workflows/{coordinate,direct,batch,council,route-question,execute-wave,resume}.md` (new/replacing `execute.md`, `communicate.md`, `escalate.md`, `resume.md`) | Executable procedures with bounded inputs/outputs and fallbacks. |
| `skills/shark-attack/workflows/pull-by-role.md` | Retire to a compatibility reference (REQ-F-017). |
| `skills/shark-attack/providers/{codex,claude-code}.md` (new) | Capability mapping, detection, unsupported behavior, sequential fallback, citations. |
| `skills/shark-rider/verbs/run.md` | Extend the existing exact-transport section with hash provenance, the `question` control-envelope path, heartbeat during consultation, same-worker follow-up, and bounded replacement. |
| `skills/shark-rider/context/host-adapter-contract.md` (new) | Provider-neutral request/result fields and prompt-hash provenance. |
| `internal/sharkdata/default_data/skills/shark-attack/**` | Byte-for-byte mirror of every authored change above. |

Note: `skills/shark-rider/` is authored-only — it has no embedded counterpart and
no manifest entry — so Rider changes are single-tree and outside the parity gate.

### Data model changes

**None.** No new table, column, index, trigger, or migration; `CurrentSchemaVersion`
stays at 32. This is deliberate and is the core consequence of the E39 pivot:
the Question lifecycle F09 needs already exists (schema versions 29–32,
`internal/db/db.go:479-482`). F09 stores no durable state of its own beyond
council artifacts, which are files.

Two additive **wire** fields are added to the keyed-next JSON response
(`prompt_sha256`, `prompt_bytes`). Both are `omitempty` and additive, so existing
consumers that ignore unknown fields are unaffected.

### API / interface contracts

**1. Keyed-next provenance (F09 produces; I-10).**

```
shark next <key> --json
  → { ..., "prompt": "<exact>", "prompt_sha256": "<hex>", "prompt_bytes": <int> }
shark next <key> --prompt-out <path>     # exact UTF-8 bytes, no trailing newline
```

Byte-exactness is defined **at the adapter boundary**, not as a claim about model
tokenization. Verification is independent recomputation of the SHA-256 over the
captured spawn payload.

**2. Worker control envelope (ephemeral, F09 produces; I-10).**

```yaml
kind: final|question|needs_council|blocked_external|failed
recommended_outcome: <verbatim; present only for kind=final>
evidence: []
```

`recommended_outcome` is opaque. The parent passes it to
`shark status advance --outcome <key>` untouched.

The `question` variant carries: `entity_key`, `category`
(`product|requirements|architecture|quality|process`), `question`,
`why_blocking`, `evidence[]`, optional `options[]`, optional `recommendation`.
It carries **no** `question_id` — the identity is E39's `Q###`, minted by the
parent, not by the worker.

**3. Live question loop (F09 consumes E39; X-06).** All calls are parent-executed:

| Step | Command | Notes |
|---|---|---|
| Mint | `shark create question "<title>" --summary … --requester … --blocking` | `Q###` assigned by Shark; never specified by the caller |
| Configure | `shark question configure-workflow Q### --resolution-owner <id> --responder <id>…` | Responders ordered; drives serial routing |
| Gate | `shark link Q### <entity-key> --type=question_blocks` | Source must be a Question; target must be a non-Question eligible entity. **Order is load-bearing**: `QualifyQuestionBlock` only qualifies a Question whose status is `open` or `answering`, so a `question_blocks` edge created while the Question is still `draft` is silently inert. Configure the workflow (which moves `draft` → `open`) **before** linking. |
| Route | `shark next Q### --json` | Returns `spawn_agent` naming only the current pending responder |
| Claim | `shark claim Q### --by <responder>` | Parent holds the Q### lease |
| Record | `shark question respond Q### --session <sid> --responder <id> --summary … --evidence-pointer …` | Requires exact `SessionID` + `ClaimedBy` match; exact replays are idempotent |
| Close | `shark question resolve Q### --owner <id> --resolution-kind <kind> --resolution-pointer <ptr>` | Requires `ready_for_resolution` and all responders completed |
| Read | `question blocking-for <key>` / `open-by-responder <id>` / `full Q### --actor <id>` | `full` is authorized only for the current responder or the resolution owner |

**Transcription seam (REQ-F-005).** `RecordResponse` requires
`claim.SessionID == input.SessionID && claim.ClaimedBy == input.Responder`
(`internal/services/question_workflow_service.go:116`), but ADR-005 forbids
workers from running Shark mutation commands. The responder worker therefore
returns its answer as structured text in its final response, and the **parent**
transcribes it into `question respond` under the parent-held lease. The
`--evidence-pointer` must resolve on disk or in the database per `Resolve`'s
`validateResolutionDestination`; `ValidateQuestionBoundedText` rejects
`system prompt`, `user prompt`, `bearer `, and friends — which is precisely the
structural enforcement of the plan's "transcripts and rendered prompts never
become council evidence" rule. F09 adds no second validator (REQ-NF-004).

**4. Capability profile (roster, additive).**

```yaml
capability_profile: fast|balanced|deep     # new, non-authoritative
requirements:                              # new, optional
  tools: read|write|test
  context: bounded|extended
  messaging: optional|required
  isolation: optional|required
model_tier: <any>                          # retained, deprecated, warns, no mapping
```

### Key technical decisions

| ID | Decision | Rationale |
|---|---|---|
| D-001 | **Build the live-question loop entirely on E39's public Question API.** | X-06 names F09 the activation owner and E39-F04 explicitly declined to build the consumer side. The alternative (a bespoke `QuestionControl` store) was built on the abandoned branch and rejected by owner decision. Research-report Decisions. |
| D-002 | **Add no code under `internal/runner/`.** The provider-neutral adapter contract ships as skill/documentation files plus a conformance fixture test, not a Go interface. | The Rider loop spawns **host-native** workers with `response.prompt`; it never calls `runner.Dispatch` (verified: `skills/shark-rider/verbs/run.md` contains no `shark run` or `internal/runner` reference). `internal/runner` powers `shark run`, which epic §3 places out of scope ("provider adapters remain host concerns"; "changing the existing behavior of ordinary single-entity `/run` execution"). Plan §5 Tranche C lists zero `internal/runner/` files. **This deviates from the research report's Capability-map `EXTEND` verdict on `AgentDispatcher`**; per Rule 7 the epic scope and implementation plan are the more authoritative and more recent constraint, and extending `AgentDispatcher` would change `/run` to serve a path that does not use it. |
| D-003 | **Prompt provenance is F09's own scope, not an F08 backfill.** | Plan §5 lists provenance under **Tranche C** (F09), not Tranche A (F08). The abandoned commit `e079efd0` bundled F08+F09 changes together, which is why it reads as F08 work. F09 owns it outright. |
| D-004 | **Hash the fully assembled prompt**, after `assembleDispatchPrompt` — including the ownership preamble and agent body. | That string is what the adapter actually transports. Hashing the pre-assembly instruction would verify something no worker ever receives. |
| D-005 | **The parent mints `Q###`, not the worker.** The control envelope carries no `question_id`. | Workers hold no Shark authority (ADR-005). A worker-generated identity would either be ignored or create a second identity space. |
| D-006 | **`question_blocks` is the structural hold**; the parent does not rely on prose discipline to avoid advancing during consultation. | The gate is enforced at `shark status advance` (`status_group.go:545`), keyed `next` (`next.go:498-508`), `shark run` preflight, and the runner. Note the deliberate hole: **`shark status set` bypasses it.** Hence AC-006 — `status set` is a human escape hatch and appears on no Rider path. |
| D-007 | **Sequential is the default topology**; parallel requires captured evidence. | GATE-F09-005; plan §8 recommended defaults. Isolation does not make logically dependent changes parallel — producer/consumer contract order still applies. |
| D-008 | **`capability_profile` is additive; `model_tier` is retained with a warning and no provider mapping.** | GATE-F09-001. A hard replacement would break every existing roster and, worse, a provider mapping would imply the roster grants provider authority — which epic §3 explicitly forbids. |
| D-009 | **Skill changes land in both trees plus a parity gate; the embedded bundle is not removed.** | The parity gate is required under either canonical direction (plan §8 #10). Removal would retire X-05, a contract owned by completed F04 — not a feature spec's call. See the Consumes X-05 row. |
| D-010 | **The skill restructure and its replacement contract tests are one atomic change.** | `e38_f04_interactions_test.go` and `e38_f07_interactions_test.go` pin verbatim prose and exact paths of `message-schema.md`, `pull-by-role.md`, `worker-ownership.md`, `execute.md`, and `run.md`. `internal/sharkdata/shark_attack_workflows_test.go`, `shark_attack_pull_test.go`, and `shark_attack_test.go` independently pin the same and adjacent files via the installed bundle (`Init(root)` + `os.ReadFile`), not just via `tests/contracts`. Any split — across any of these five files — leaves an intermediate red commit (REQ-NF-006). |
| D-011 | **Contract tests follow the house `e38_f0N_interactions_test.go` convention**, not the plan's proposed `shark_attack_v2_test.go`. | F10 and F11 must cite the I-10 pointer verbatim; naming it conventionally now avoids a later rename that would break their citations. |

### Integration with existing code

| Seam | Exact location | F09's interaction |
|---|---|---|
| `NextResponse` | `internal/cli/commands/next.go:140-183` | Add two `omitempty` fields. No existing field changes. |
| Prompt assembly | `assembleDispatchPrompt`, `internal/cli/commands/next.go:996-1006`, called at :619-623 | Hash its return value. |
| `next` flag registration | `internal/cli/commands/next.go:257-264` (`--sequential` only today) | Add `--prompt-out`. |
| JSON emission | `outputNextJSON`, `internal/cli/commands/next.go:1070-1077` | Unchanged — new fields marshal automatically. |
| OTel span | `internal/cli/commands/next.go:352` already sets `attribute.Int("prompt_bytes", len(resp.Prompt))` | Reuse the same value for the wire field. Keeps span and wire consistent by construction. |
| `services.QuestionBlock` | `internal/services/question_blocker.go:32-38` | Read-only consumer of the compact handoff already surfaced at `next.go:173-175`. |
| Question gate | `QuestionBlocker.Check`, `internal/services/question_blocker.go:129`; qualification at `:82-103` | Relied upon, not modified. |
| Advance guard | `guardQuestionBlockedStatusAdvance`, `internal/cli/commands/status_group.go:545` | Relied upon (AC-005, AC-006). |
| Question CLI | `internal/cli/commands/question.go:452-484` | Consumed as-is. No new subcommand. |
| Question workflow | `internal/sharkdata/default_data/workflow/question.yaml` | Consumed as-is. Not modified. |
| Rider loop | `skills/shark-rider/verbs/run.md` (`response.action` ∈ `spawn_agent`, `pause`, `archive`, `error`, `parallel_candidates`) | Extended with the question path. Today the file contains **zero** occurrences of "question" — this is the gap F09 closes. |
| Roster validator | `internal/sharkdata/embed.go:987` (`ModelTier`), `:1133` (blank check), `:1169` (allowed-key map) | Add `capability_profile` and `requirements` to the allowed-key map; add the deprecation warning for `model_tier`. |
| Skill CLI-string gate | `TestEmbedded_SkillsContainNoBareSharkCLIRefs`, `internal/sharkdata/embed_test.go:852` | New skill prose must respect it. Note its verb list omits `shark next`/`claim`/`release`, so it will not catch dispatch-verb prose — the parity gate and contract tests carry that weight instead. |

### Degraded upstream dependencies

Per REQ-F-018, each I-08 / I-09 payload element, its verified trunk state, and
F09's coverage:

| Upstream element | Declared source | Verified state on `main` | F09 depends on it? | F09's coverage |
|---|---|---|---|---|
| Keyed-next prompt provenance | I-08 (F08) | **Absent** (`grep prompt_sha256` → no match) | **Yes** | F09 delivers it directly (REQ-F-011). Per D-003 this was always Tranche C scope. |
| Read-only `shark plan` (no status/history write) | I-08 (F08) | **Absent** — `plan.go:631` still calls `autoAdvanceCascadeParent`; pinned by `plan_dispatch_test.go:254` ("exactly one auto-advance transition") | **No** | F09's selection is chair-side and uses keyed `shark next`. The skill MUST NOT instruct a chair to call `shark plan` for read-only inspection, because on current trunk that call mutates. Recorded as a constraint, not repaired here. |
| Tech-debt related-docs routing | I-08 (F08) | **Absent** (`related_docs.go` has no tech-debt route) | No | Out of scope. |
| Research validator single-rule statement | I-08 (F08) | Unverified | No | Out of scope. |
| Tech-debt single implementation/gate state | I-08 (F08) | **Absent** | No | Out of scope. |
| Nested-worktree audit regression | I-08 (F08) | Unverified | No | Out of scope. |
| Council artifact model / repository / service / `shark admin council` tooling | I-09 (F05) | **Absent** — `internal/models/council_artifact.go`, `internal/repository/council_artifact_repository.go`, `internal/services/council_artifact_service.go`, `internal/cli/commands/admin_council.go` all missing; no `admin council` subcommand registered. F05's directory holds only `feature.md` and `assessment.md`. | Partially | F09's durable decisions/handoffs (REQ-F-008, REQ-F-009) are **files under `docs/council/`** written per the existing `message-schema.md` conventions. F09 MUST NOT call `shark admin council validate|create|validate-wave`, and MUST NOT import a council artifact Go type. If F05's tooling later lands, F09's artifacts are validated by it without change. |

Both F08 and F05 report Shark status `completed` while their implementations are
unreachable from `main` (`git merge-base --is-ancestor e079efd0 HEAD` → not an
ancestor). This is a project-integrity matter for the owner, recorded here as
evidence rather than adjudicated.

---

## Cross-feature interactions

### Produces

- **I-10** — Provider-neutral adapter contract. Consumers: E38-F10 Cross-Provider
  Adapter Conformance; E38-F11 Complicated Lifecycle Release Qualification.
  Payload: coordination level; execution topology; capability profile; exact
  prompt hash and byte length; worker identity; control envelope; follow-up,
  interrupt, isolation, and replacement behavior.
  **Shape source**: [Provider-neutral adapter contract](../../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#3-provider-neutral-adapter-contract).
  **Contract tests**: `tests/contracts/e38_f09_interactions_test.go#TC-002`
  (provenance byte-exactness and worker identity), `#TC-003` (control envelope,
  capability profile, supported/unsupported declaration), `#TC-005` (two-axis
  independence), `#TC-010` (follow-up, interrupt, isolation, and replacement
  behavior — the resume half of the payload).

### Consumes

- **I-04** — Council communication contract. Producer: E38-F04 Shark Attack Skill
  and Role Protocol. Payload consumed: roster role; bounded question; handoff;
  decision; escalation; resolution; entity scope; evidence references. F09 routes
  material questions through this existing chair/role escalation shape; routine
  question/answer pairs live in the E39 Question record, not a second council
  transcript.
  **Shape source**: [E38 architecture §4.5 Council communication contract](../architecture.md#45-council-communication-contract).
  **Contract tests**: `tests/contracts/e38_f09_interactions_test.go#TC-006`.

- **I-06** — Role-aware pull and claim guidance. Producer: E38-F06. Payload
  consumed: workflow-resolved role; selected entity key; claim eligibility. F09
  does not touch selection or claim eligibility; capability profile and roster
  membership remain non-authoritative.
  **Shape source**: [F06 requirements](../E38-F06-role-aware-pull-and-claim/feature.md#requirements) and [v2 authority model](../../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#authority-model).
  **Contract tests**: `tests/contracts/e38_f09_interactions_test.go#TC-007`.

- **I-07** — Rider execution and escalation loop. Producer: E38-F07. Payload
  consumed: keyed dispatch identity; parent-owned claim and transition boundary;
  workflow outcome; heartbeat state; question or bounded-handoff result. F09 adds
  question routing **around** this loop, not inside a new one.
  **Shape source**: [Live question-and-resume protocol](../../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#live-question-and-resume-protocol).
  **Contract tests**: `tests/contracts/e38_f09_interactions_test.go#TC-001`.

- **I-08** — Shark Attack v2 integrity prerequisites. Producer: E38-F08. Payload
  consumed: read-only plan trace; unchanged status/history evidence; tech-debt
  related-document support; research prompt/validator rule; single
  implementation-gate ownership; nested-worktree regression result.
  **Producer implementation is unverified on `main`** — see §"Degraded upstream
  dependencies" for the per-element dependency table.
  **Shape source**: [Implementation plan §5, Tranche A](../../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#5-file-by-file-implementation-plan).
  **Contract tests**: `tests/contracts/e38_f09_interactions_test.go#TC-009`
  (asserts F09's behavior under the degraded state: F09 delivers its own
  provenance and never invokes `shark plan` as a read-only probe).

- **I-09** — Deterministic council artifacts and operator handoffs. Producer:
  E38-F05. Payload consumed: artifact type and immutable ID; effective roles;
  entity-or-collection scope; evidence paths; timestamps; `supersedes`;
  validation and generation result. **Producer implementation is unverified on
  `main`** — F09 writes `docs/council/` files directly and calls no council
  tooling.
  **Shape source**: [Implementation plan §5, Tranche B](../../../../dev-artifacts/shark-attack-v2-plan/implementation-plan.md#5-file-by-file-implementation-plan).
  **Contract tests**: `tests/contracts/e38_f09_interactions_test.go#TC-009`.

---

## Cross-epic integrations

### Consumes / Validates

- **X-06** — Supply a durable serial Question lifecycle, scoped blocking
  visibility, and authoritative resolution for provider-neutral live-question
  handling.
  **Producer**: E39 — Question and Decision Workflow Management (E39-F04 Focused
  Question Read Surfaces and Consumer Handoff). **Consumer / activation owner**:
  E38-F09 (this feature).
  **Contract / shape source**: E39 architecture §2–§4; E38-F09 feature.md.
  **UX / CX handoff notes**: one scoped responder prompt; compact blocked-work
  handoff; Question state rather than chat or council copies supports resume.
  Concretely: the dispatched worker receives only the compact
  `question_block` (`question_key`, `summary`, `resolution_owner`,
  `current_responder`) — it is neither the current responder nor the resolution
  owner, so `question full` is correctly denied to it. The responder receives one
  scoped prompt naming only itself.
  **Test coverage**: the shared producer-shape proof remains
  `tests/contracts/e39_interactions_test.go#TC-004`
  (`TestTC004_X06ProducerPublicQuestionHandoffIsReadOnly`) — F09 cites it
  verbatim and does **not** create a twin. F09 adds distinct **consumer
  activation** coverage at
  `tests/contracts/e38_f09_interactions_test.go#TC-004`, which exercises the
  end-to-end mint → gate → route → respond → resolve → unblock loop that the
  producer test deliberately does not drive. E39's cross-epic map already
  records this obligation ("E38-F09 … must add consumer coverage when
  resumed"); this row discharges it.

- **X-05** — Distribute `shark-attack`, roster, and communication procedures
  through embedded Shark-data and replace-only overrides.
  **Producer**: E32 — Shark 2.0 Single-Artifact Consolidation. **Owning
  feature**: E38-F04 (completed). **Consumer**: E38-F09 — every skill change in
  this feature ships through that embedded/replace-only mechanism.
  **Contract / shape source**: E38 architecture §2 ADR-007 and §5 Phase 4; E32
  embedded bundle contract.
  **UX / CX handoff notes**: refreshed workers must see the same versioned
  procedure, so the restructured v2 skill must remain installable and
  override-compatible; the replace-only override boundary is unchanged.
  **Test coverage**: `tests/contracts/e38_f09_interactions_test.go#TC-008`
  (authored/embedded parity plus replace-only override behavior), alongside the
  existing `tests/contracts/e38_f04_interactions_test.go#TC-004`
  (`TestTC004_X05EmbeddedSkillOverrideIsReplaceOnly`), which F09 leaves intact.
  **Unresolved tension — owner decision required.** The live Shark record for
  E38-F09 carries `implementation_decisions.GATE-F09-003`:
  *"client-skill-only; remove embedded Shark Attack bundle."* Removing the
  embedded bundle would retire X-05, a contract owned by a completed feature and
  recorded in `docs/product/cross-epic-integration-map.md`. A feature
  specification may not alter map-assigned values, so **F09 does not remove the
  embedded bundle**; it ships changes to both trees and adds the parity gate,
  which is required under either canonical direction (plan §8 #10). Acting on
  GATE-F09-003 requires an explicit X-05 amendment by the map owner.
  (Note: the referenced artifact
  `docs/council/decisions/d-e38-f09-v2-owner-gates-001.yaml` is **not present on
  `main`** — it exists only in the unmerged worktree. The authority for
  GATE-F09-001…005 cited throughout this spec is the live Shark
  `context_data.implementation_decisions` on E38-F09.)

---

## Contract test index

| TC | Covers | Interaction |
|---|---|---|
| TC-001 | Parent-owned loop invariants: worker returns envelope only; parent claims, heartbeats, advances, releases; zero worker-owned transitions | I-07 |
| TC-002 | Prompt provenance byte-exactness under adversarial fixture; `--prompt-out` vs `--field prompt` | I-10 |
| TC-003 | Control envelope round-trip incl. opaque `recommended_outcome`; capability profile; supported/unsupported declaration for Codex and Claude Code | I-10 |
| TC-004 | **X-06 consumer activation**: mint → `question_blocks` → `next Q###` → respond → resolve → unblock | X-06 |
| TC-005 | Two-axis independence: coordination level and topology vary independently; sequential degradation | I-10 |
| TC-006 | Council routing threshold; routine answers create no artifact; material questions create one immutable record | I-04 |
| TC-007 | Role-aware selection and claim eligibility unchanged; roster/profile grant no authority; `pull-by-role` retired from the normal path | I-06 |
| TC-008 | Authored/embedded parity gate; replace-only override preserved | X-05 |
| TC-009 | Degraded-upstream behavior: F09 delivers its own provenance, invokes no `shark plan` read-only probe, and imports no council artifact type | I-08, I-09 |
| TC-010 | Resume lifecycle: same-worker follow-up vs. bounded replacement; interrupt of a stale consultation; isolation detection driving topology; capability-discovery-before-selection ordering; no rendered prompt in any handoff | I-10 |

---

## Exit-gate checklist

- [x] Every requirement is testable — each REQ maps to at least one AC or contract TC.
- [x] Every architecture decision references an existing pattern or explains the deviation (D-002 explicitly explains the deviation from the research report's `EXTEND` verdict).
- [x] File paths listed for all changes — one Go production file, three new Go test files, and an enumerated skill-content table.
- [x] No TBDs in critical sections. The two genuinely unresolved items (GATE-F09-003 vs. X-05; the F05/F08 status-versus-trunk mismatch) are stated as bounded owner decisions with a recorded default behavior, not deferred blanks.
- [x] Multi-feature epic: every I-## produced (I-10) or consumed (I-04, I-06, I-07, I-08, I-09) is declared with shape source and contract test pointer.
- [x] Every X-## consumed or validated (X-06, X-05) is declared with producer/consumer ownership, matching contract/shape source, UX/CX handoff notes, and test coverage pointers.
- [x] No twin tests: X-06's shared producer shape is proved once at `e39_interactions_test.go#TC-004`, cited verbatim; F09 adds only distinct consumer-activation coverage.
