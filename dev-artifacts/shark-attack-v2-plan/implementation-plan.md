# Shark Attack v2 implementation plan

Status: implementation-ready proposal

Date: 2026-07-25

Revised: 2026-07-25 — corrections applied after a code-verification review of every repo-checkable claim.

Scope: planning only; this document does not authorize backlog creation, claims, workflow transitions, or code changes

## Executive decision

Shark Attack v2 should keep the proposed two-axis model:

1. **Coordination level** answers how much judgment and escalation structure the work needs: Direct, Batch, or Council.
2. **Execution topology** answers how changes can safely be made: Sequential, Parallel with ownership, or Parallel with isolation.

The axes must remain independent. The first Shark Attack was a successful **Batch + mostly Sequential** run over seven mild tech-debt items. It proved parallel research, specialist review, parent-owned Shark state, and durable decisions. It did not prove the intended hierarchy on a complicated epic or feature. The chair still acted as prompt courier, lease operator, integrator, and manual escalation router, while most implementation passed serially through one worker.

The defining v2 correction is therefore not “spawn more agents.” It is a provider-neutral, live question-and-resume loop:

1. A developer reports a bounded question without relinquishing the entity or mutating Shark.
2. The chair classifies and routes the question to the appropriate council role.
3. The specialist answers from bounded evidence.
4. The chair returns the answer to the same developer context when the host supports it.
5. The developer resumes; the parent retains claim, heartbeat, transition, kickback, and release authority.
6. Only material decisions, unresolved questions, or context replacement create durable council artifacts.
7. If a host cannot resume the worker, the chair creates a bounded handoff and starts a replacement worker with prompt-hash provenance and the resolved answer.

The release test must be a real epic/feature lifecycle with cross-feature contracts, implementation waves, a deliberately triggered council escalation, rework, integration review, QA, and UAT. The TD-043–TD-049 batch remains a regression fixture, not the v2 acceptance test.

## Evidence and assessment method

This plan distinguishes:

- **Confirmed current**: reproduced in the live implementation or its tests.
- **Repaired**: the first-run incident is no longer present.
- **Design choice**: the incident is corrected, but policy must be made explicit.
- **Conversation evidence**: useful operational evidence whose provider sessions are not yet available through the local `ctx` index.

Primary repository evidence includes:

- the current protocol in `skills/shark-attack/`;
- the embedded distribution in `internal/sharkdata/default_data/skills/shark-attack/`;
- the first-run handoffs in `docs/council/handoffs/`;
- the seven decisions in `docs/council/decisions/d-td04*-first-attack.yaml`;
- the live selection and transition paths in `internal/cli/commands/plan.go` and `internal/cli/commands/next.go`;
- the related-document dispatch in `internal/cli/commands/related_docs.go`;
- the research validator in `internal/research/validator.go`;
- the tech-debt workflow and prompts under `internal/sharkdata/default_data/`;
- the existing Shark Attack bundle and contract tests.

The first-run conversation review found one chair and three depth-one worker lanes. Research was parallel. Architecture and QA were follow-up assignments. The chair manually copied corrections and prompts, maintained the Shark lifecycle, and routed review. There was no demonstrated developer → council question → council answer → same-developer resume cycle. Provider session/event identifiers should be retained in an evidence appendix if the conversations are later imported into `ctx`, but the durable YAML artifacts remain the canonical evidence today.

## 1. Current-state assessment

### What worked

- Parallel research reduced discovery latency across TD-043–TD-049.
- Architect and QA review found ordering, schema, role, evidence, and test problems that a single implementation pass missed.
- The parent retained Shark claim and transition authority.
- Work was reordered by shared-file and dependency pressure rather than by entity number alone.
- Failed validation produced bounded recovery instead of abandoned state.
- Durable decisions and handoffs preserved the reasoning needed for resume.
- The protocol added no runtime, scheduler, claim store, credential store, or model router.

### What did not yet work

- The standing hierarchy was described more strongly than it was exercised. The chair performed most coordination directly.
- Developers did not have a seamless structured way to ask a council question and resume.
- Long `response.prompt` values were manually relayed.
- Shared-worktree implementation forced serialization after parallel research.
- Artifact rules contradicted one another and lacked deterministic enforcement.
- The mild tech-debt batch did not exercise feature decomposition, cross-feature contracts, implementation integration, deep review, QA, or UAT.

### Finding ownership and currentness

| ID | Current status | Correct owner | Planned correction |
|---|---|---|---|
| SA-001 `shark plan` persists simulated advances | Confirmed current; the mutation is deliberate and test-pinned as parity with keyed next, so removing it is a semantic change requiring confirmation (§8) | Shark CLI/service | Add a read-only status overlay/pure resolver. Preserve one-pull selection semantics while preventing status/history writes. Keyed `shark next` remains mutating. |
| SA-002 tech-debt related docs unsupported by CLI | Confirmed current, narrow | Shark CLI | Add tech-debt dispatch to add/list/delete. Reuse existing `TechDebtService` link/list/unlink methods; no database migration. |
| SA-003 prompt transfer is manual | Confirmed as an enforcement/provenance gap; `skills/shark-rider/verbs/run.md` already mandates exact `response.prompt` transport for the Claude Code host | Shark CLI + Rider host adapters | Emit hash provenance from keyed next, capture the actual adapter payload with fixtures, and extend the transport rules to non-Claude adapters. The gap is enforcement and provenance, not missing transport documentation. |
| SA-004 prompt/validator line mismatch | Confirmed current, class-wide | Shark prompt/validator contract | Make the physical-line rule explicit in every affected research prompt or change the validator contract consistently. Add a multiline counterexample and a corpus-wide guard. |
| SA-005 nested worktrees contaminate tests | Repaired | Repository tests | Keep the existing nested-`.git` skip and add a focused temporary nested-worktree regression. Do not schedule a second implementation fix. |
| SA-006 duplicate developer/full gate | Confirmed current | Shark workflow bundle | Make `in_progress` the sole implementation and full-gate state; make `triaged` an agentless normalization advance. |
| SA-007 no artifact generator | Confirmed current | Shark deterministic data tooling | Add schema-backed generation that validates before writing. |
| SA-008 no multi-root scope | Confirmed design gap | Council schema/tooling | Add a tagged scope union supporting one entity or a nonempty unique collection. |
| SA-009 immutability contradicts “update” | Confirmed current | Council schema and skill | Make revisions immutable: a new ID plus `supersedes`; never edit the prior record. |
| SA-010 operational role not in roster | Incident repaired; policy open | Roster/schema/tooling | Default closeout to Technical Director or Scrum Master. Permit project specialists only when present in the effective roster. Do not add a global Maintainer role by accident. |
| SA-011 evidence confinement is too narrow | Confirmed current | Council schema/tooling | Confine artifact writes to `docs/council`; allow validated repository-relative evidence references inside the project root. |
| SA-012 no deterministic council validator | Confirmed current; roster-only validation exists | Shark deterministic data tooling | Extend validation to council artifacts, revisions, effective roles, scope, evidence, timestamps, and paths. |

The SA-004 defect class is: **machine-enforced prompt output constraints are not guaranteed to be stated consistently in every rendered prompt that can produce that output**. The implementation sweep must enumerate the complete research-prompt corpus, not patch only tech debt, and add one structural prompt/validator contract test.

### Semantics that must not be “fixed” away

`shark plan` is supposed to show what one pull would select after logically traversing agentless states. The defect is persistence, not traversal. Its replacement must simulate transitions in an in-memory overlay and return the same candidate and explanatory trace that a real pull would reach. It must not merely stop at the first agentless state.

Likewise, `shark next <key> --json` remains the only canonical worker dispatch contract. Advisory selection, a roster role, a model preference, or a host-native task list cannot replace the keyed dispatch.

## 2. Recommended operating model

### Authority model

The hierarchy is a routing and judgment structure, not a second workflow engine:

- **Technical Director / chair**: selects coordination level and topology, integrates evidence, routes questions, and owns the parent loop.
- **Product Manager**: resolves value, priority, scope, acceptance, and product trade-offs.
- **Business Analyst**: resolves requirement meaning, workflow, edge cases, and traceability.
- **Architect**: resolves boundaries, contracts, sequencing, data shape, integration, and technical risk.
- **Scrum Master**: resolves process, ordering, lease/heartbeat, blocked-flow, and ceremony questions.
- **Developer**: implements only the bounded dispatch, reports evidence, and raises questions early.
- **Quality Analyst**: resolves test strategy, acceptance evidence, regression scope, and release-gate questions.

The chair may answer a routine question directly only when the answer is already explicit in authoritative project evidence. Material ambiguity goes to the relevant specialist. Product/technical disagreement goes to Council coordination.

### Coordination levels

#### Direct

Use when one entity is bounded, acceptance is clear, the edit surface is small, and no material cross-entity decision is expected.

Minimum protocol:

1. Parent runs keyed `shark next`.
2. One worker receives the exact prompt.
3. Worker implements and reports evidence.
4. Parent verifies and advances or kicks back.
5. Escalate only if the worker reports a material question.

No standing council, inbox, or decision artifact is required.

#### Batch

Use for related entities that benefit from one scope and conflict analysis but do not need sustained specialist adjudication.

Minimum protocol:

1. Chair defines a collection scope and one dependency/file-overlap map.
2. Research runs in parallel where read-only.
3. One batch decision or handoff records only material shared conclusions.
4. Implementation runs in conflict-aware waves.
5. The parent records per-entity Shark outcomes.
6. Run one full gate per integrated tranche with actual changes.
7. Create one consolidated observation record.

The TD-043–TD-049 run is the reference Batch case.

#### Council

Use when any of these is true:

- architecture or product intent is materially unclear;
- two specialists disagree;
- a cross-feature or cross-epic contract is missing or inconsistent;
- implementation crosses ownership boundaries or has high blast radius;
- security, data migration, cost, or irreversible behavior is at risk;
- acceptance requires independent technical and product judgment;
- a worker cannot proceed safely from project evidence.

Council means a standing, messageable hierarchy for the active tranche. It does not mean one artifact per role or a meeting before every task.

### Execution topologies

#### Sequential

Use for overlapping files, dependent edits, one shared database/environment, producer-before-consumer contracts, or hosts without safe isolation.

#### Parallel with ownership

Use only when the chair records exclusive file/component ownership, workers have disjoint write sets, shared contracts are frozen, and the parent can integrate without speculative conflict resolution. Shared files are owned by one worker.

#### Parallel with isolation

Use separate worktrees or equivalent isolated sessions when write sets may overlap but work can be integrated as explicit commits/tranches. The parent controls base revision, integration order, conflict resolution, and the post-integration gate.

Isolation does not make logically dependent changes parallel. Producer/consumer contract order still applies.

### Selection algorithm

1. Inspect the root entity, current keyed dispatch, children, dependencies, acceptance, and likely edit surfaces.
2. Choose the lowest coordination level that covers ambiguity and risk.
3. Choose topology independently from file/contract/environment safety.
4. Record the choice in the chair’s bounded run state; create a durable artifact only if needed for resume or audit.
5. Re-evaluate both axes after research, a question, a kickback, or integration conflict.
6. Degrade to Sequential whenever ownership or isolation cannot be proven.

Examples:

| Work | Coordination | Topology | Why |
|---|---|---|---|
| One localized task with clear tests | Direct | Sequential | One worker is cheaper and clearer. |
| Seven related tech-debt items | Batch | Parallel research, then Sequential/ownership waves | Shared files and gates limit safe mutation despite easy parallel discovery. |
| Cross-cutting feature with API, storage, CLI, and UAT | Council | Mixed: research parallel; isolated/disjoint implementation waves; controlled integration | Requirements, contracts, review, and acceptance require live specialist routing. |

### Live question-and-resume protocol

A worker returns an adapter control envelope. This envelope is separate from Shark’s workflow-configured outcome:

```yaml
kind: final|question|needs_council|blocked_external|failed
recommended_outcome: <present only for kind=final; copied verbatim>
evidence: []
```

`recommended_outcome` is opaque to Shark Attack. It may be `completed`, `simple`, `standard`, `deep_verify`, or any other key configured by the active workflow. The adapter and chair must never coerce it into the control-envelope vocabulary.

A `question` payload contains only:

```yaml
question_id: q-<stable-run-suffix>
entity_key: <dispatched key>
category: product|requirements|architecture|quality|process
question: <one answerable question>
why_blocking: <bounded impact>
evidence:
  - <repository-relative path or command result reference>
options:
  - <optional bounded alternatives>
recommendation: <optional worker view>
```

The chair:

1. sets a consultation deadline bounded by the current claim lease, keeps the parent claim alive, and does not advance status;
2. routes by category to the standing role;
3. sends only the dispatch identity, question, bounded evidence, and authoritative references;
4. obtains an answer with rationale and any conditions;
5. returns it to the same worker using native follow-up messaging;
6. records a decision only if it changes a contract, scope, acceptance, ordering, risk posture, or future resume state;
7. uses an immutable handoff plus replacement worker if live resume is absent.

If the specialist is silent, fails, or disappears, the chair pings once, interrupts/cancels the stale consultation where supported, and routes to one replacement specialist. If no qualified responder returns before the consultation deadline, the chair pauses: stop write workers, create a bounded unresolved handoff, record the external/process blocker through the parent, and release according to the canonical Rider loop. Never hold a claim indefinitely.

If heartbeat fails or the lease is lost, the parent immediately stops/interrupts active mutation workers. It must not deliver a council answer, integrate changes, or transition the entity under the lost authority. Resume requires a fresh keyed dispatch and successful claim; the handoff supplies context but never restores authority.

Routine answers remain in the host conversation. Transcripts and rendered prompts never become council evidence.

### Complicated epic/feature references from WWGM

The live WWGM Shark state makes E04-F02 the strongest primary hierarchy/resume case:

- E04 is active with five features and explicit I-01–I-04 contracts in `docs/plan/E04-upload-source-ledger/E04-interaction-map.md`.
- E04-F02 is active at 55%; three tasks are back in development after review/UAT.
- Its Shark notes record a worker whose dispatched task-generation reply never arrived and a later correction after the parent self-certified a gate instead of obtaining the designated reviewer verdict.
- `docs/review/E04-upload-source-ledger/E04-F02-durable-docling-source-ledger/uat-20260722T190258Z-E04-F02.md` rejects a behaviorally green implementation because the producer drops I-02 heading metadata needed by the future F03 consumer, the service violates hard architecture caps, database rollback omits filesystem side effects, and routine selectors omit load-bearing tests.
- The UAT routes separate kickbacks to T-E04-F02-002, T-E04-F02-004, and T-E04-F02-005, whose edit surfaces overlap enough to require sequential work or isolated integration.

E04-F02 should prove that a developer can ask whether a missing future-consumer field is a producer defect now, receive an Architect/Business Analyst answer, and resume. It should also prove that a transaction/filesystem question routes to Architect plus QA, that a lost worker reply degrades to bounded handoff rather than parent self-certification, and that reviewer authority remains distinct from chair judgment.

WWGM E18 is the secondary decision-quality replay:

- `docs/plan/E18-pool-paradigm-absorption-blades-in-the-dark-univer/epic.md` defines a falsifiable GO/NO-GO boundary: kernel-additive extension may proceed; cross-cutting engine rework must stop and escalate.
- `gate-verdict.md` distinguishes engine defects from verification-apparatus assumptions, preserves a qualified “GO — proven-with-findings,” records owner-authorized live spend, and shows that a broad review found serious defects after green gates.

E18 should prove that the council can stop work, preserve qualifications, distinguish evidence classes, require owner authorization, and return NO-GO when warranted.

WWGM E14 remains a useful contract-shape precedent: its interaction map has producer/consumer direction that differs from build order, one growing integration test shared across features, a corrected missing consumer declaration, blocking specialist review, and traceable epic UAT. The v2 release test may borrow that shape, but E04-F02 and E18 are the primary behavioral references.

## 3. Provider-neutral adapter contract

The core protocol describes capabilities, not provider commands:

| Capability | Required behavior |
|---|---|
| Spawn | Start a bounded worker with role, task name, exact prompt payload, scope, and optional preference profile. |
| Send/follow up | Deliver a question answer or correction to the identified live worker. |
| Progress/final | Return inspectable state and a bounded evidence result. |
| Wait/poll | Let the chair remain responsive while workers proceed. |
| Interrupt/list | Enumerate and stop bounded workers without changing Shark state. |
| Isolate | Give a write worker a known base revision and independent filesystem when requested. |
| Resume | Continue the same bounded context when supported; otherwise return “unsupported.” |
| Provenance | Hash the exact adapter-boundary prompt and retain worker/session identity without storing the rendered prompt. |

Capability detection occurs before topology selection. Missing messaging, resume, or isolation is data that causes fallback; it is not permission to invent a command.

### Prompt transport contract

“Exact prompt” means exact at the adapter boundary, not a claim about a model’s internal tokenization:

1. Read `response.prompt` directly from `shark next <key> --json`. Shark emits `prompt_sha256` (and prompt byte length) in the same response, so no external tool derives the hash.
2. Pass that string to the host’s native spawn/prompt field without summarization or hand transcription. CLI adapters that need raw bytes on disk use the keyed-next `--prompt-out <path>` flag; `--field prompt` appends a trailing newline and is not byte-exact.
3. Capture the actual spawn payload through a hook, mock, or tool-call fixture.
4. Verify the captured payload against `prompt_sha256` by independent recomputation.
5. Store only hash, entity key, provider worker ID, and timestamps as provenance.

### Provider capability matrix

Capability claims below are based on installed capability inspection where available and current official documentation. Antigravity and Copilot were not locally executable in this checkout, so their adapter acceptance must remain documentation-backed and feature-gated until run in an installed environment. Re-verify the Codex documentation link at implementation time — the cited domain was not re-checked during plan verification.

| Host | Native capability | Adapter requirement | Safe fallback | Evidence source |
|---|---|---|---|---|
| Codex | Native subagent spawn, follow-up, wait, list, interrupt, nested delegation, per-spawn model/reasoning preference. Subagents inherit sandbox/permissions; automatic worktree isolation is not documented. | Pass `response.prompt` directly as the spawn message, retain returned agent ID, use follow-up for council answers, and have the parent provision explicit worktrees before write waves. | Sequential workers in the parent checkout; durable handoff when the session cannot retain a worker. | Installed collaboration tools plus [Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents). |
| Claude Code | Named subagents support foreground/background execution, results, resumable IDs for supported agents, model/effort preferences, and optional worktree isolation. Agent teams add direct teammate messaging but are experimental, non-nested, and share files. | Use a team only for the standing council when enabled. Use named worktree-isolated subagents for write waves. Rehydrate council roles from durable artifacts after session loss. | Lead relays among named subagents; sequential execution when teams/worktrees are unavailable. | Installed Claude Code plus [subagents](https://code.claude.com/docs/en/sub-agents), [agent teams](https://code.claude.com/docs/en/agent-teams), and [worktrees](https://code.claude.com/docs/en/worktrees). |
| GitHub Copilot CLI | Custom agents and `/fleet` support parallel/nested work, status, and messaging; top-level sessions and experimental worktree sessions are documented. `/fleet` child isolation and child restoration are not documented. | Use custom agents for role identity. Use `/fleet` only for read-only/disjoint work. Use separate worktree sessions for overlapping writes and parent-controlled integration. | Sequential custom-agent tasks; pause if the required command is unavailable rather than assuming it. | Official [CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference), [fleet](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/fleet), and [agent sessions](https://docs.github.com/en/copilot/how-tos/github-copilot-app/agent-sessions); not installed locally. |
| Google Antigravity | Documented subagent spawn, messaging, list/kill, retained idle context, nested delegation, and branch-workspace isolation. Exact model/effort control is more limited than the other hosts. | Map standing roles to messageable idle agents and implementation workers to branch workspaces. Detect capability availability; do not require paid preview teamwork features. | One-shot/sequential execution and immutable handoff when live subagents or resume are unavailable. | Official [subagents](https://antigravity.google/docs/subagents), [CLI subagents](https://antigravity.google/docs/cli/subagents), and [hooks](https://antigravity.google/docs/hooks); not installed locally. |

Official sources:

- [OpenAI Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents)
- [Claude Code subagents](https://code.claude.com/docs/en/sub-agents)
- [Claude Code agent teams](https://code.claude.com/docs/en/agent-teams)
- [Claude Code worktrees](https://code.claude.com/docs/en/worktrees)
- [GitHub Copilot CLI command reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference)
- [GitHub Copilot fleet](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/fleet)
- [GitHub Copilot agent sessions](https://docs.github.com/en/copilot/how-tos/github-copilot-app/agent-sessions)
- [Google Antigravity subagents](https://antigravity.google/docs/subagents)
- [Google Antigravity CLI subagents](https://antigravity.google/docs/cli/subagents)
- [Google Antigravity hooks](https://antigravity.google/docs/hooks)

### Provider-neutral model preferences

Replace values such as `opus` and `high` with:

```yaml
capability_profile: fast|balanced|deep
requirements:
  tools: read|write|test
  context: bounded|extended
  messaging: optional|required
  isolation: optional|required
```

Adapters map the profile best-effort to installed models and effort controls. Unsupported preferences fall back to the host default with a warning. They never select work or grant Shark authority.

## 4. Target skill and tooling structure

Use `skills/shark-attack/` as the human-authored source and generate/check the embedded mirror under `internal/sharkdata/default_data/skills/shark-attack/`. A parity test must make drift impossible.

Note: this **reverses the currently recorded convention** (edit `internal/sharkdata/default_data/` as canonical; the repo copy is kept in lockstep manually). Today the two trees are byte-identical, no sync script exists, and no test compares the trees — tree-level parity is unenforced. The direction of canonicalness is therefore a decision requiring confirmation (§8); the parity gate is required under either direction.

```text
skills/shark-attack/
├── SKILL.md
├── context/
│   ├── roster-schema.yaml
│   ├── operating-model.md
│   ├── authority.md
│   ├── worker-control-schema.yaml
│   ├── council-artifact-schema.yaml
│   └── execution-scope-schema.yaml
├── workflows/
│   ├── coordinate.md
│   ├── direct.md
│   ├── batch.md
│   ├── council.md
│   ├── route-question.md
│   ├── execute-wave.md
│   └── resume.md
├── providers/
│   ├── codex.md
│   ├── claude-code.md
│   ├── github-copilot.md
│   └── antigravity.md
└── templates/
    ├── decision.yaml
    ├── handoff.yaml
    ├── escalation.yaml
    ├── resolution.yaml
    └── batch-observation.yaml
```

`SKILL.md` should contain only triggers, invariants, the two-axis selection step, the parent loop, the question loop, and direct links to the needed workflow/provider resource. It must not duplicate provider commands or complete schemas.

Consolidation:

- Replace `workflows/execute.md` with level-specific workflows plus `execute-wave.md`.
- Fold `workflows/communicate.md` and `workflows/escalate.md` into `authority.md` and `route-question.md`.
- Replace the conflicting `context/message-schema.md` with the deterministic artifact and worker-control schemas.
- Fold `context/worker-ownership.md` into `authority.md` and `execute-wave.md`.
- Retire `workflows/pull-by-role.md` from the normal Rider path. Preserve its worker-owned-child behavior in a compatibility reference only if a live consumer is found.
- Keep `workflows/setup.md` (its actual location; there is no top-level `setup.md`) only if installation/override instructions cannot live in Shark-data documentation; otherwise remove it from the runtime skill.

Deterministic behavior belongs outside prose:

- council parsing, validation, and generation: Shark admin data tooling;
- prompt extraction/hashing: Shark keyed-next output (`prompt_sha256`, `--prompt-out`) plus host adapters and conformance fixtures;
- execution-wave ownership checks: compiled Shark admin tooling behind a service;
- status simulation: Shark selection service;
- worktree isolation: host adapter plus parent integration workflow;
- quality gates: project workflow/tests;
- provider capability mapping: provider references plus detection fixtures.

## 5. File-by-file implementation plan

### Tranche A — repair confirmed Shark integrity defects

| Exact file | Change and owner | Dependencies | Acceptance evidence |
|---|---|---|---|
| `internal/cli/commands/plan.go` | Shark CLI: resolve against a plan-only virtual status overlay and emit simulated actions without calling transition persistence. | Define the resolver seam shared with `next`. | Plan selects the same eventual entity but leaves entity status and history byte-equivalent. |
| `internal/cli/commands/next.go` | Shark CLI/service: separate pure resolution from the mutating keyed-next transition executor. Keep keyed-next behavior unchanged. | Resolver seam. | Existing keyed-next cascade/agentless tests remain green. |
| `internal/cli/commands/plan_dispatch_test.go` | Replace mutation-pinning plan tests with selection-equivalence and no-history-write regressions. | Plan resolver. | Before/after database status-history comparison. |
| `internal/cli/commands/related_docs.go` | Add `--tech-debt` to add/list/delete routing. | Existing `TechDebtService` APIs. | Command tests cover all three operations and invalid mixed selectors. |
| `internal/services/entity_document_service_test.go` | Include tech debt in the all-entity-types document contract. | CLI route not required. | Polymorphic contract includes tech debt. |
| `internal/sharkdata/default_data/prompts/*/research.md` | State the evidence layout exactly for every prompt using the universal research validator (six today: epic, feature, bug, change, sprint, tech_debt; task has no research step — the guard test must enumerate dynamically). | Choose validator contract first. | Rendered corpus contains one unambiguous rule. |
| `internal/research/validator.go` | Keep or deliberately revise the physical-line rule, but expose one contract consumed by prompts/tests. Recommended first release: keep it and document it to avoid parser ambiguity. | Prompt decision. | Multiline evidence fails with an actionable error; same-line evidence passes. |
| `internal/research/validator_test.go` and prompt golden/contract tests | Sweep the entire prompt surface and add structural guard. | Prompt + validator changes. | Test enumerates every applicable prompt, not a hard-coded sample. |
| `internal/sharkdata/default_data/workflow/tech-debt.yaml` | Make `triaged` agentless and `in_progress` the sole developer implementation/full-gate state. | Confirm transition shape. | One developer dispatch and one full gate per unchanged implementation cycle. |
| `internal/sharkdata/default_data/prompts/tech_debt/triaged.md` | Remove implementation/gate instructions or retire the prompt when the state becomes agentless. (Today `triaged.md` holds the full implementation + mandatory-gate block; `in_progress.md` duplicates one gate-check sentence verbatim and defers to `triaged.md` by prose pointer.) | Workflow change. | No duplicate instructions in rendered dispatch corpus, including prose pointers that re-import them. |
| `internal/sharkdata/default_data/prompts/tech_debt/in_progress.md` | Own implementation, resumability, and the one full exit gate; inline the full gate block rather than referencing `triaged.md` prose. | Workflow change. | Resume after interruption does not dispatch a second independent implementation pass. |
| `internal/config/legacy_reference_audit_test.go` | Verification only: add a direct temporary nested-worktree regression around the existing `.git` skip. | None. | Nested checkout content cannot contaminate the audit; ordinary repo content still can fail it. |

### Tranche B — deterministic council data contract

| Exact file | Change and owner | Dependencies | Acceptance evidence |
|---|---|---|---|
| `internal/models/council_artifact.go` (new) | Typed data model for artifact type, immutable ID, role references, scope union, evidence, timestamps, and `supersedes`. No orchestration or validation policy. | Approve schema defaults below. | YAML round-trip fixtures. |
| `internal/repository/council_artifact_repository.go` (new) | Confined filesystem access for listing, reading, and create-without-overwrite under `docs/council`; resolve paths/symlinks inside the project root. Reuse `internal/fileops.EntityFileWriter` for atomic create-without-overwrite rather than new machinery. | Model + project root. | Repository tests cover traversal, symlink escape, collision, and atomic create. |
| `internal/services/council_artifact_service.go` (new) | Own artifact validation, generation, revision rules, effective-roster checks, evidence policy, project traversal, and execution-wave ownership validation. | Model, repository, effective roster, templates. | Table-driven malformed fixtures and wave fixtures. |
| `internal/cli/commands/admin_council.go` (new) | Thin commands: `shark admin council validate [path]`, `create --type ...`, and `validate-wave`. This is data tooling, not a team runtime. Resolve the service through the CLI accessor. | Council service. | CLI integration tests in a temp project. |
| `internal/cli/commands/admin.go` and `internal/cli/services_global.go` | Register the council subcommands/help and the council service accessor. | Command + service. | Help/invalid-argument tests and accessor wiring test. |
| `internal/sharkdata/embed.go` | Validate only the embedded/effective Shark Attack schemas, templates, and roster. Do not traverse project `docs/council` here. | Final bundle. | Default bundle and malformed override tests. |
| `skills/shark-attack/context/council-artifact-schema.yaml` (new) | Human-readable schema source matching the Go model. | Approved policy. | Cross-check test between schema constants and parser behavior. |
| `skills/shark-attack/context/worker-control-schema.yaml` (new) | Ephemeral control envelope with `final`, question/escalation, blocker, and failure variants. The `final` variant carries the untouched workflow `recommended_outcome`. It is explicitly not a required durable file. | Question loop and Rider outcome contract. | Arbitrary outcome keys round-trip unchanged; the question fixture is accepted by all adapters. |
| `skills/shark-attack/context/execution-scope-schema.yaml` (new) | Tagged union: `entity` XOR `collection`; collection has unique canonical root keys and optional children. | Scope decision. | Ambiguous/empty/duplicate scopes rejected. |
| `skills/shark-attack/templates/*.yaml` (new) | Minimal decision, handoff, escalation, resolution, and batch-observation templates. | Final schemas. | Generator golden tests. |
| `docs/council/README.md` | Split write confinement from reference confinement and document immutable revisions. (Today the README states neither rule — its path language covers only private/gitignored material. The immutability and confinement rules being corrected live in `context/message-schema.md`, and the contradictions are recorded in the h-td043/h-td045 handoff observations.) | Validator behavior. | README examples pass live validator. |
| `internal/services/testdata/council_artifacts/**` (new) | Valid and malformed YAML for every rule. | Validator. | Each malformed fixture fails for one named reason. |

Recommended schema defaults:

- Scope is a tagged union: exactly one of `entity` or `collection`.
- Revisions are new immutable records with `supersedes`; prior records remain byte-unchanged.
- Artifact writers may write only under `docs/council`.
- Evidence may reference an existing repository-relative path that resolves inside the project root; symlink escape, secrets, transcripts, and rendered prompts are rejected.
- Operational sender/recipient roles must exist in the effective roster.
- Technical Director owns closeout unless the effective roster explicitly assigns another canonical role.

### Tranche C — hierarchy, adapters, and skill redesign

| Exact file | Change and owner | Dependencies | Acceptance evidence |
|---|---|---|---|
| `skills/shark-attack/SKILL.md` | Minimal router: invariants, two axes, selection, parent loop, question loop, links. | Stable council tooling and adapter contract. | Fresh-agent comprehension test; word-count/duplicate-rule check. |
| `skills/shark-attack/context/roster-schema.yaml` | Replace provider-specific `model_tier` with provider-neutral capability profiles; retain deprecated input compatibility. (Current values mix `opus` and `high`; two of seven default members omit the field entirely.) | Profile names approved. | Roster validator accepts v2; old roster emits warning, not authority change; absent `model_tier` maps silently to host default with no warning. |
| `skills/shark-attack/context/operating-model.md` (new) | Full level/topology rules and escalation thresholds. | Operating model. | Scenario table classifies Direct, Batch, and Council independently from topology. |
| `skills/shark-attack/context/authority.md` (new) | Parent/worker/council authority and role routing. | None. | Contract tests assert parent-only Shark mutation. |
| `skills/shark-attack/workflows/{coordinate,direct,batch,council,route-question,execute-wave,resume}.md` (new/replaced) | Executable procedures with bounded inputs/outputs and fallback. | Data/adapter contracts. | Rendered workflow tests and fresh-agent forward tests. |
| `skills/shark-attack/providers/{codex,claude-code,github-copilot,antigravity}.md` (new) | Current native mapping, detection, unsupported behavior, sequential fallback, and citations. | Provider conformance contract. | Documentation link check plus installed-host smoke where available. |
| `skills/shark-rider/verbs/run.md` | Extend the existing exact-transport section (it already mandates passing `response.prompt` verbatim with host-safe adapter selection) with hash provenance, question outcome handling, heartbeat during consultation, same-worker follow-up, and bounded replacement fallback. | Adapter contract. | End-to-end Rider consumer test from keyed next through worker result. |
| `skills/shark-rider/context/host-adapter-contract.md` (new) | Provider-neutral request/result fields and prompt-hash provenance. | None. | Shared conformance fixture. |
| Keyed-next prompt provenance in `internal/cli/commands/next.go` + service (replaces the previously proposed `skills/shark-rider/scripts/prompt-envelope.py`) | Shark CLI: add `prompt_sha256` and prompt byte length to the keyed-next JSON response, plus a `--prompt-out <path>` flag writing the exact UTF-8 prompt bytes (no trailing newline) for CLI adapters. Compiled, provider-independent, no interpreter detection, no metadata sidecar. Native tool adapters read `prompt` from the JSON directly. | Adapter contract. | Adversarial byte fixture (newlines, fences, quotes, Unicode, shell metacharacters): `--prompt-out` bytes hash to the emitted `prompt_sha256`; `--field prompt` documented as not byte-exact (trailing newline). |
| `tests/contracts/shark_attack_v2_test.go` (new) | Behavior contracts for two axes, authority, artifact threshold, question routing, and fallbacks. | Skill rewrite. | Fails if prose loses any invariant or duplicates provider instructions into core. |
| `tests/contracts/shark_attack_adapter_test.go` (new) | One fixture applied to all four provider mappings. | Provider references. | Each adapter declares supported/unsupported operations and sequential fallback. |
| Tree-parity gate: Go test + small Go sync helper or `go:generate` (replaces the previously proposed `scripts/sync-shark-attack-skill.py`) | Deterministic mirror sync/check between the authored skill and embedded Shark data; refuse unexpected embedded-only files. Implemented in Go so the gate runs inside `make test` with no Python dependency. | Final authored tree; canonical-direction decision (§8). | Parity test fails on any byte drift; sync followed by check is clean. |
| `tests/contracts/e38_f04_interactions_test.go` and `e38_f07_interactions_test.go` | Replace obsolete substring assertions with v2 behavior contracts; retain override and keyed-dispatch guarantees. | Skill rewrite. | Existing E38 guarantees remain observable without pinning old filenames. |
| `internal/sharkdata/shark_attack_*_test.go` (three today: roster, workflows-prose, pull) | Validate installed schemas/templates/workflows rather than prose fragments. `TestTC103` currently reads the live repo `docs/council/` layout (structure-only, test-scoped) — keep that boundary: no production-loader traversal of `docs/council`. | Bundle update. | Embedded install is complete and usable. |
| `internal/sharkdata/default_data/skills/shark-attack/**` | Generated byte-for-byte mirror of the authored skill. | All skill edits. | Parity test and installed-bundle tests. |
| `internal/sharkdata/default_data/README.md` | Document v2 install/override/migration behavior. | Final structure. | Examples resolve from embedded and replace-only override paths. |

Add a deterministic parity command or test rather than relying on manual copying. The implementation PR must fail if the authored and embedded trees differ.

Sequencing constraint: `tests/contracts/e38_f04_interactions_test.go` and `e38_f07_interactions_test.go` pin exact paths and verbatim prose of nearly every file this tranche renames, folds, or deletes (`message-schema.md`, `pull-by-role.md`, `worker-ownership.md`, `execute.md`, `run.md`). Their replacement contracts must land in the same PR as the skill restructure — any intermediate state fails the gate.

## 6. Validation plan

### Unit and component tests

- Pure plan resolver: simulated agentless transitions, cascade traversal, no repository writes.
- Keyed next: existing real transition behavior unchanged.
- Related docs: tech-debt add/list/delete and mixed-selector errors.
- Research contract: enumerate all applicable prompts; same-line evidence positive, multiline negative.
- Tech-debt workflow: exactly one developer state and one full-gate owner.
- Worker control envelope: arbitrary workflow outcomes pass through unchanged; control kinds never become transition keys.
- Council parser/generator: valid round trips and atomic no-write failure.
- Roster migration: v1 `model_tier` warning and v2 profile validation.
- Embedded/authored parity and replace-only project overrides.

### Malformed council fixtures

Cover:

- invalid YAML/type/required field;
- artifact ID not matching path/type directory;
- unknown role;
- both/neither entity and collection scope;
- empty, duplicate, or malformed keys;
- timestamp order and invalid RFC3339;
- missing/nonexistent/out-of-root/symlink-escaped evidence;
- likely secret, transcript, or rendered-prompt evidence;
- overwrite attempt;
- missing/cyclic/duplicate `supersedes`;
- duplicate ID with non-byte-equivalent content.

### State-history regression

Create a temporary database with a cascade parent whose children are terminal and whose next logical state is agentless:

1. snapshot entity rows and status history;
2. run `shark plan <key> --json`;
3. assert the planned candidate and simulated trace;
4. byte-compare rows/history to the snapshot;
5. run keyed `shark next <key> --json`;
6. assert the real transitions and history entries.

### Prompt-transport integrity

Use a prompt containing newlines, Markdown fences, quotes, Unicode, and shell metacharacters. For each installed adapter:

1. capture the raw `response.prompt` and the shark-emitted `prompt_sha256`;
2. independently recompute the hash and assert it matches the emitted value;
3. intercept the actual spawn/tool payload;
4. assert byte equality and the same hash;
5. assert no council artifact contains the prompt.

Documentation-only providers stay explicitly pending until exercised on an installed host.

### Shared-worktree conflict test

Give two workers a shared file plus otherwise disjoint files:

- Parallel with ownership must reject the plan because the shared file has two writers.
- Sequential must accept it.
- Parallel with isolation may accept it only with base revisions, integration order, and a named integrator.
- After integration, run one full gate for the tranche, not one redundant gate per unchanged entity.

### Cross-host conformance fixture

Each adapter must report:

- supported capabilities;
- exact spawn field;
- worker identifier;
- progress/final retrieval;
- follow-up/resume behavior;
- interrupt/list behavior;
- isolation behavior;
- preference mapping;
- explicit unsupported values;
- sequential and replacement-worker fallback.

No adapter passes on prose claims alone; installed hosts need a smoke fixture or captured native tool invocation.

### Fresh-agent forward tests

Do not tell agents the expected topology, defect, or verdict. Give only project evidence and acceptance criteria, then score behavior from durable state and adapter events.

#### Mild regression: TD-043–TD-049

Repeat the collection as Batch:

- parallel research;
- one dependency/overlap analysis;
- safe implementation waves;
- one integrated gate per changed tranche;
- per-entity outcomes;
- one consolidated observation.

This proves backward compatibility, not v2 completion.

#### Required release test: complicated epic/feature

Select a user-approved, genuinely complicated epic/feature and execute it with fresh native agents in an isolated real project worktree. It must require nontrivial production-code changes, real commits and integration, project quality gates, independent review, and UAT. A fixture may seed initial Shark state or inject a transport failure, but it may not synthesize worker questions, adapter events, defects, verdicts, commits, or outcomes. Do not mutate WWGM E04/E18 themselves without separate authorization; use their evidence to select or shape the real candidate.

The candidate should carry the pressures of WWGM E04-F02, E18’s material GO/NO-GO fork, and E14’s explicit interaction-map discipline:

1. At least two features and multiple tasks per feature.
2. Research, architecture/specification, feature review, task generation/review, implementation, code review/deep verification, QA, approval, and UAT.
3. At least one cross-feature producer/consumer contract (`I-01`) and one cross-cutting concern.
4. A read-only parallel research wave.
5. A disjoint implementation wave with explicit ownership.
6. A shared-contract/shared-file wave that must become sequential or isolated plus controlled integration.
7. One unseeded, discoverable contract or integration defect; fresh agents are not told its location or expected conclusion.
8. A developer must encounter and raise a genuine structured architecture or acceptance question during the production change.
9. The chair must route it to the correct standing role, keep the lease alive, return the answer, and resume the same developer where supported.
10. Force one review or QA kickback and a fresh-worker resume from an immutable decision/handoff.
11. Inject one transport-level lost worker response without fabricating the worker’s result. The parent may verify transport and preserve evidence, but must not self-certify a specialist gate; it must resume, replace, or pause.
12. Include a database-plus-filesystem or equivalent multi-resource atomicity question so behavioral green tests alone are insufficient.
13. Include an owner-authorized action and a real stop/NO-GO branch that the chair cannot infer from roster or model preference.
14. Interrupt the first council responder after dispatch. The chair must cancel/replace within the consultation deadline or pause and release; it may not hold the claim indefinitely.
15. Assert every worker re-enters through keyed `shark next` and receives the exact prompt hash.
16. Assert the parent alone claims, advances, kicks back, notes, and releases.
17. Assert `shark plan` never changes status/history.
18. Assert real worker commits are integrated in declared order and one full quality gate runs per integrated changed tranche.
19. Assert final multi-root/batch artifacts contain bounded evidence and no transcript/prompt leakage.
20. Require literal approval before approval → completed.

The test passes only when the feature’s integrated behavior and UAT pass. A collection of completed tasks is insufficient.

## 7. Migration and compatibility

### Existing council artifacts

- Grandfather current v1 artifacts as read-only historical records.
- Do not rewrite the staged first-run decisions/handoffs in place.
- Validator reports v1 compatibility warnings separately from v2 errors.
- Any new correction creates a v2 record with `supersedes`.
- An optional migration report may propose normalized replacements, but it must not write without explicit user action.

### Existing rosters

- Accept `model_tier` for one compatibility window.
- Map known values best-effort to `fast`, `balanced`, or `deep` and warn.
- An absent `model_tier` is valid today (two of seven default roster members omit it) and maps silently to the host default — no warning.
- Unknown values fall back to host default.
- Authority continues to come from Shark state and the parent loop, never the mapped profile.

### Embedded distribution and overrides

- Author once in `skills/shark-attack/`.
- Generate/check the embedded mirror.
- Preserve replace-only overrides under `shark-data/overrides/skills/shark-attack/`.
- Validate the effective merged view, including project-specific specialist roles.
- Missing optional provider files do not block Direct Sequential operation.

### Older or limited hosts

- No parallelism: run the same wave sequentially.
- No messaging: chair returns a bounded handoff and starts a replacement worker.
- No resume: never claim restoration; use prompt hash + handoff + current keyed dispatch.
- No isolation: reject overlapping parallel writes.
- No list/interrupt: use bounded foreground work or pause before dispatching concurrent workers.
- No model selection: use host default without changing authority.

## 8. Risks and open decisions

### Confirmed delivery risks

- Separating plan simulation from keyed-next mutation can regress cascade semantics if implemented as two drifting resolvers.
- A prose-only adapter can falsely imply exact transport; fixtures must observe the actual boundary.
- Provider features may change or be gated by version/account; adapters need detection and documented fallback.
- Parallel isolated changes can still violate logical contract order.
- Council artifacts can become a shadow workflow if routine messages are persisted.
- Embedded/authored skill drift can silently ship stale behavior.
- A “successful” simple batch can mask hierarchy and resume failures.

### Recommended defaults

- Use Direct unless a Batch/Council threshold is evidenced.
- Use Sequential unless disjoint ownership or isolation is proven.
- Make `in_progress` the single tech-debt implementation/gate state.
- Use immutable `supersedes` revisions.
- Keep the seven canonical roles; project specialists must be declared in the effective roster.
- Use `fast`, `balanced`, and `deep` capability profiles.
- Ship Copilot and Antigravity as feature-detected adapters until locally exercised.
- Make the complicated epic/feature forward test a release gate, not a later demonstration.

### Decisions requiring confirmation

1. Approve `shark admin council create`, `validate`, and `validate-wave` as the deterministic data-tooling namespace.
2. Approve the tagged `entity` XOR `collection` scope model.
3. Approve immutable `supersedes` revisions and grandfathering rather than rewriting v1 artifacts.
4. Approve Technical Director as default closeout owner and no new global Maintainer role.
5. Approve `fast|balanced|deep` as adapter preferences.
6. Decide whether worker-owned child pulling has a live consumer; otherwise remove it from the v2 core. (Repo-wide search found no Go runtime consumer — only the skill's own tests, `execute.md` cross-references, and the E38-F06 research report — which supports removal with a compatibility reference.)
7. Approve documentation-backed Copilot/Antigravity adapters as experimental until installed-host conformance runs.
8. Select and approve a real complicated epic/feature for release qualification. Run it in an isolated worktree and do not reuse or mutate WWGM E04/E18 without explicit authorization.
9. Approve the `shark plan` read-only semantic change. The current persistence is deliberate and test-pinned as parity with keyed next (`plan_dispatch_test.go` asserts exactly one auto-advance transition, "same as next.go's cascade completion"); confirm no existing consumer depends on plan's side effects before those tests are replaced.
10. Approve the canonical authoring direction for the skill: `skills/shark-attack/` authored source generating the embedded mirror (as this plan proposes) versus the currently recorded embedded-canonical convention. The tree-parity gate ships under either direction.

## 9. Phased delivery

### Phase 0 — integrity repairs

Deliver SA-001, SA-002, SA-004, SA-006, and the SA-005 regression.

Exit criteria:

- plan status/history is unchanged;
- keyed next is behavior-compatible;
- tech-debt related docs work;
- the research defect class is guarded across the prompt corpus;
- tech debt has one implementation/full-gate owner;
- focused tests and the repository’s full Go quality gate pass.

### Phase 1 — smallest valuable v2

Deliver:

- Direct/Batch/Council and topology selection;
- live developer-question routing and same-worker resume/fallback;
- authority rules and heartbeat behavior;
- deterministic artifact schema, validator, generator, revisions, multi-root scope, evidence rules;
- capability-profile roster;
- Codex and Claude Code adapters;
- authored/embedded parity;
- the Direct and seven-item Batch regressions.

Exit criteria:

- native prompt transport is hash-verified on both installed hosts;
- routine questions create no artifact;
- a material question creates one valid immutable decision/handoff;
- a silent council responder is replaced or causes a bounded pause/release before lease loss;
- arbitrary workflow outcomes pass through the control envelope unchanged;
- fallback runs sequentially without authority drift;
- all bundle/override/contract tests and the full quality gate pass.

### Phase 2 — provider breadth

Deliver feature-detected Copilot and Antigravity adapters and installed-host conformance where environments are available.

Exit criteria:

- each installed adapter passes the common fixture;
- unavailable capabilities degrade explicitly;
- overlapping writes never run in an unisolated shared checkout;
- provider docs and detection results are current.

### Phase 3 — release qualification on a complicated epic/feature

Run the required fresh-agent forward test across the full epic/feature lifecycle. Use E04-F02 as the hierarchy/transport/kickback reference, E18 as the decision-quality/authorization reference, and E14 as the interaction-contract shape reference.

Exit criteria:

- hierarchy routes and resolves at least one real blocking question;
- native agents make, commit, and integrate nontrivial production changes; no canned fixture supplies their outcomes;
- same-worker resume or bounded replacement is evidenced;
- a silent first council responder follows the bounded replacement/pause path;
- contract omission is caught and corrected;
- implementation topology changes based on ownership/isolation evidence;
- integration, deep review, QA, UAT, and literal approval all execute;
- parent-only Shark authority and prompt-hash provenance hold;
- no prompt/transcript leakage occurs;
- the integrated epic/feature outcome, not merely its tasks, passes.

Shark Attack v2 is not complete before Phase 3 passes.

## Final recommendation

### Recommended architecture

Use a small provider-neutral skill protocol over Shark’s existing durable authority, with thin native host adapters, deterministic Shark admin tooling for council artifacts, and Rider-owned prompt transport. Maintain a live, messageable council only when judgment warrants it. Do not build a runtime, scheduler, ledger, claim store, model router, or credential store.

### Recommended coordination levels and execution topologies

- Coordination: Direct, Batch, Council.
- Topology: Sequential, Parallel with ownership, Parallel with isolation.
- Select them independently and re-evaluate after research, escalation, or conflict.

### First implementation tranche

Start with the integrity defects that would invalidate later evidence: non-mutating `shark plan`, tech-debt related-doc routing, the class-wide research prompt/validator contract, and the single tech-debt implementation/gate owner. Keep the already-fixed nested-worktree behavior as a regression.

### Decisions requiring confirmation

Confirm the decisions in [§8: Risks and open decisions](#8-risks-and-open-decisions) before
implementation.

### Explicit non-goals

- No new agent runtime or workflow engine.
- No second claim, lease, status, or task store.
- No model router or credential store.
- No provider/model name in the authority model.
- No assumption that agent count implies safe parallel writes.
- No durable artifact for every routine message.
- No transcript or rendered-prompt storage in council artifacts.
- No replacement of keyed `shark next`.
- No release claim based only on the mild seven-item tech-debt batch.
