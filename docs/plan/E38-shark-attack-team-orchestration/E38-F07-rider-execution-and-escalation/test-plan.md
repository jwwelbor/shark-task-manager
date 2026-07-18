# Test Plan: E38-F07 — Rider Execution and Escalation Loop

**Created:** 2026-07-15  
**Feature PRD:** `feature.md`  
**Feature specification:** `spec.md`  
**Status:** APPROVED

## Spec drift analysis

### Drift findings

None. The specification narrows the historical runtime-oriented E38 material to
the active procedure boundary in `feature.md`: existing Shark commands and the
Rider parent loop remain authoritative, while Shark Attack supplies bounded
role, handoff, and escalation guidance.

### Traceability matrix

| Feature requirement | Acceptance criteria | Planned coverage |
|---|---|---|
| REQ-F-001 / REQ-NF-001 | AC-001 | TC-001 |
| REQ-F-002 / REQ-NF-002 | AC-003 | TC-003, TC-004 |
| REQ-F-003 | AC-002 | TC-002 |
| REQ-F-004 | AC-004 | TC-005 |
| REQ-F-005 / REQ-F-006 / REQ-NF-003 | AC-005, AC-006 | TC-006, TC-007, shared I-04 TC-001/TC-002 |
| REQ-F-007 / REQ-NF-004 | AC-007 | TC-008 |

## AC test matrix

| AC | Technique | Setup / input | Expected outcome and edge cases |
|---|---|---|---|
| AC-001 | Contract-surface enumeration | Read the host `run.md` and materialized `execute.md`. | Both name `shark next`, claim the returned entity, pass `response.prompt` unchanged, process an outcome, advance by it, and release the session. Reject `shark get` prompt reconstruction or a Shark persona as a host adapter. |
| AC-002 | State-transition / negative-path enumeration | Inspect worker preamble and task-execution guidance. | Parent alone owns lease and transition operations; workers return evidence/outcome. Negative: worker instructions cannot authorize `claim`, `heartbeat`, `release`, `status advance`, or `status set` on the dispatched entity. |
| AC-003 | Decision table | Exercise documented actions `spawn_agent`, `pause`, `archive`, `error`; outcomes pass/fail/blocked/missing; and gate failure with/without kickbacks. | Spawn path releases after transition; pause/archive/error stop; missing outcome creates a blocker record; kickbacks apply before parent advance; partial work never becomes a fabricated success. |
| AC-004 | Equivalence partitioning | Compare a valid workflow-role self-pull example with roster-only/model-tier/legacy-agent alternatives. | Only the workflow-resolved role uses `shark sprint next --agent=<type>` to select a key, then enters `/shark-rider run <selected-key>`. Negative: direct claim/execution of its `BacklogItemView`, roster responsibility, or model tier does not confer authority. |
| AC-005 | Decision table | Escalate a material architecture question with policy and without policy. | Record question, evidence, responsible role, requested decision, route, and next owner. Absent policy uses `council-review`; no fixed human destination is introduced. |
| AC-006 | State-transition / recovery | Start from a refreshed coordinator with claim/history plus decision, handoff, unresolved escalation, and inbox pointers. | Resume reads bounded durable context and ordinary Shark state. Negative: no second run ledger, transcript, or prompt copy is required. |
| AC-007 | Installation-contract enumeration | Materialize Shark-data into a temporary project. | `execute.md` is installed with resolvable references; host Rider remains a local procedure. Negative: bundle content does not add `team` commands, runtime code, or provider configuration. |

## Integration scenarios

| Scenario | Components and boundary | Coverage |
|---|---|---|
| UAT-03 ordinary dispatch ownership | `skills/shark-rider/verbs/run.md` → `shark next` response → parent dispatch → worker contract. | TC-001 and TC-002 validate X-01 prompt/ownership wording; UAT verifies a live configured workflow. |
| UAT-04/UAT-06 stop and recovery | Rider procedure → configured outcomes, claim/history → council pointers. | TC-003, TC-004, and TC-007 cover explicit content boundaries; UAT runs configured stop states. |
| UAT-05 escalation clarity | Shark Attack execution procedure → existing escalation/message schema. | TC-006 plus shared I-04 TC-001/TC-002. |
| UAT-01/UAT-02 role pull and claim safety | Existing role selector/claim service → Rider procedure. | TC-005 limits F07 to consuming existing authority; owner tests remain in F06/E19. |
| UAT-07 concise handoff | Worker outcome → council handoff/escalation → fresh operator. | TC-006 and TC-007 require bounded metadata and exclude prompts/secrets. |

## Test infrastructure

- Follow embedded-content materialization patterns in
  `internal/sharkdata/shark_attack_workflows_test.go`; use `sharkdata.Init` in
  a temporary directory and read the installed file.
- Follow contract-content patterns in `tests/contracts/e38_f04_interactions_test.go`;
  use `sharkdata.ReadEmbedded` and string assertions against the published
  protocol rather than a real database or a host subprocess.
- Follow dispatch assembly coverage in `internal/cli/commands/next_test.go` for
  the production `shark next` response shape. This feature must not duplicate
  that command's test fixture.
- No new test helper is required. These are pure file/bundle contract tests;
  they must not use the real Shark database.

## Cross-feature contract tests

| I-## | Producer | Consumer | Shape source | Shared pointer | Planned use |
|---|---|---|---|---|---|
| I-04 | E38-F04 | E38-F07 | E38 architecture §4.5 Council communication contract | `tests/contracts/e38_f04_interactions_test.go#TestTC001_I04InboxMessageProtocol` and `#TestTC002_I04ArtifactProtocol` | TC-006 and TC-007 consume the exact existing bounded message/artifact contract; no twin I-04 test is created. |

## Cross-epic integration tests

| X-## | Contract / shape source | Planned test | Product-map coverage |
|---|---|---|---|
| X-01 | E38 architecture §4.3; existing E22 runner dispatcher contract | TC-001 checks procedure-level exact prompt and host adapter rules; command-level prompt construction remains covered by `internal/cli/commands/next_test.go`. | UAT-03 and UAT-07; `tests/contracts/e38_f07_interactions_test.go#TestTC001_X01DispatchPromptFidelity` |
| X-02 | E38 architecture §4.1 and §4.4; `docs/guides/route-based-workflow.md` | TC-003/TC-004 check configured semantic-outcome and stop/release guidance. | UAT-04, UAT-05, UAT-10; `tests/contracts/e38_f07_interactions_test.go#TestTC002_X02ConfiguredOutcomeAndStopBoundaries` |

## Caller-path contracts

| TC | Production entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `/shark-rider run <root>` procedure using `shark next <root> --json` and `response.prompt` | Internal-only documentation contract; `sharkdata.ReadEmbedded` for bundle content | Do not mock or rebuild `response.prompt`; do not test `shark get` as a dispatch source | A duplicated prompt builder omits the ownership preamble or silently changes provider/persona content. |
| TC-002 | Same Rider procedure, worker preamble, and `task-execution-pattern.md` | Internal-only content contract | Do not replace the worker contract with a helper-only permission list | A worker is allowed to advance/release its dispatched entity and races the parent. |
| TC-003 | `/shark-rider run <root>` action branch | Internal-only content contract | Do not assume terminal status names or mock a transition above the procedure boundary | Pause/archive/error are treated as pass or a partial run becomes success. |
| TC-004 | `/shark-rider run <root>` outcome/kickback branch | Internal-only content contract | Do not omit release or kickback ordering from the asserted procedure | A failed gate advances its parent before reopening the affected task, or leaves a lease live. |
| TC-005 | Existing role selector followed by `/shark-rider run <selected-key>` | Internal-only content contract; F06/E19 own service mocks | Do not mock a roster into an authorization grant or treat the selected `BacklogItemView` as claimable | A model-tier or legacy assignment selects work outside the workflow role, or a direct claim bypasses canonical `shark next` prompt/provider metadata. |
| TC-006 | `shark-attack/workflows/execute.md` → existing `escalate.md` artifact contract | `sharkdata.ReadEmbedded` | Do not use free-form transcript fixtures or a fixed human target | A material question lacks evidence/owner/decision route or is silently guessed. |
| TC-007 | `shark-attack/workflows/execute.md` → existing `resume.md`/message schema | `sharkdata.ReadEmbedded` | Do not introduce a team-run fixture or database state | A refreshed coordinator cannot resume without previous chat or leaks a prompt/transcript. |
| TC-008 | `shark admin install-shark-data` materialization path | `sharkdata.Init` temporary project | Do not use a real project database or rely on an uninstalled source path | The embedded bundle omits execution guidance, while a source-tree-only test falsely passes. |

## Acceptance test cases

### TC-001: X-01 Rider dispatch preserves the canonical prompt

**Requirements:** REQ-F-001, REQ-NF-001; **AC:** AC-001; **Technique:**
contract-surface enumeration; **ISO 25010:** Functional suitability,
compatibility, maintainability.

Read the host Rider procedure and materialized execution workflow. Assert the
canonical `shark next <root> --json` source, returned concrete entity claim,
verbatim `response.prompt`, semantic outcome, configured advance, and
session-scoped release all appear. Assert negative wording rejects prompt
reconstruction and Shark persona names as host adapters.

### TC-002: Parent alone owns dispatched-entity workflow mutation

**Requirements:** REQ-F-003; **AC:** AC-002; **Technique:** state-transition
and negative-path enumeration; **ISO 25010:** Functional suitability,
reliability, security.

Assert the worker preamble names every forbidden dispatched-entity mutation and
the parent procedure owns claim, heartbeat, transition, and release. Edge case:
the workflow may explicitly orchestrate *other* children, but that exception
cannot authorize mutation of the dispatched entity.

### TC-003: X-02 stop actions preserve configured workflow boundaries

**Requirements:** REQ-F-002, REQ-NF-002; **AC:** AC-003; **Technique:**
decision table; **ISO 25010:** Functional suitability, reliability.

Assert separate documented handling for `spawn_agent`, `pause`, `archive`, and
`error`; the latter three stop. Assert status advance receives a semantic
outcome and not a hardcoded terminal status. Negative case: an `error` action
must not be retried blindly.

### TC-004: Failure, missing outcome, and kickback release safely

**Requirements:** REQ-F-002, REQ-F-003, REQ-NF-002; **AC:** AC-003;
**Technique:** decision table; **ISO 25010:** Reliability, maintainability.

Assert missing outcome produces a blocker note, task kickbacks precede parent
advance, a fail with no kickback is surfaced, and release is required on every
success/failure/exception path. Negative case: the procedure does not report a
partial or failed worker as a successful root.

### TC-005: Role-aware self-pull consumes existing authority

**Requirements:** REQ-F-004; **AC:** AC-004; **Technique:** equivalence
partitioning; **ISO 25010:** Functional suitability, security, compatibility.

Assert the valid partition uses the workflow-resolved role with `shark sprint
next --agent=<type>` only to select a key, then invokes `/shark-rider run
<selected-key>`. Assert that the ordinary Rider loop calls `shark next` before
claiming `response.entity_key` and dispatching exact `response.prompt` with
provider metadata. Assert invalid partitions—direct claim or execution of the
returned `BacklogItemView`, roster-only membership, model tier, responsibility
prose, and legacy agent assignment—do not grant claim or transition authority.

### TC-006: Material escalation carries bounded decision metadata

**Requirements:** REQ-F-005, REQ-F-006, REQ-NF-003; **AC:** AC-005;
**Technique:** decision table; **ISO 25010:** Functional suitability,
reliability, security.

For material scope/architecture/quality questions, assert the execution recipe
requires question, evidence, responsible role, requested decision, route, and
next owner, and routes absent policy to `council-review`. Negative cases:
non-material questions do not automatically escalate; no fixed human target,
prompt, credential, or transcript is stored.

### TC-007: Refresh resumes from Shark and bounded council pointers

**Requirements:** REQ-F-006, REQ-NF-002, REQ-NF-003; **AC:** AC-006;
**Technique:** state-transition/recovery; **ISO 25010:** Reliability, security,
usability.

Assert the procedure directs a refreshed coordinator to ordinary claims/history
and scoped decisions, handoffs, unresolved escalations, and inbox records.
Negative case: it must not require prior chat or create an aggregate resume
store.

### TC-008: Embedded distribution retains execution guidance

**Requirements:** REQ-F-007, REQ-NF-004; **AC:** AC-007; **Technique:**
contract-surface enumeration; **ISO 25010:** Portability, compatibility,
maintainability.

Materialize the bundle into a temporary project and assert `execute.md` exists
and references the ordinary Rider procedure plus existing council workflows.
Negative case: installed content contains no new `team` CLI command, provider
runtime, or workflow-state authority for the chair.

## Observability design

| Behavior | Metric / log / trace | Justification |
|---|---|---|
| Published procedure wording and bundled workflow content | Internal — no runtime observability | This feature creates no runtime behavior; source and materialization contract tests are the executable evidence. |
| Live claims, workflow transition, and outcome history | Existing Shark claim/history and configured workflow records | The feature deliberately reuses existing operational evidence and adds no telemetry surface. |
| Council handoff/escalation | Bounded artifact path and metadata | Existing council protocol provides durable, privacy-bounded evidence. |

## Codex test-plan red-team

**Verdict:** PASS  
**Issues raised:** 1  
**Issues addressed before dev:** 1  
**Issues deferred:** 0

The first review found that an open-ended “interruption safe” claim could invite
unbounded test expansion. TC-003 and TC-004 now enumerate the exact action,
outcome, kickback, missing-outcome, and release classes. Each AC has a named
test technique, a negative case, ISO 25010 coverage, and a caller-path contract.
Runtime observability is intentionally N/A because the scope adds no runtime
behavior; existing Shark state/history and bounded council artifacts remain the
specified evidence surfaces.

## Recommendation

- [x] Ready for task generation: no unresolved spec drift, every AC has a
  concrete test case, and cross-feature/cross-epic contract pointers are
  explicit.
