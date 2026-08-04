# Test Plan: E38-F12 - Parallel Team Topology

**Created:** 2026-08-04
**Feature PRD:** `docs/plan/E38-shark-attack-team-orchestration/E38-F12-parallel-team-topology/spec.md`
**Task Spec:** Feature-level prompt/skill specification; task generation has not yet created task specifications.
**Status:** APPROVED

## Scope and drift analysis

E38-F12 is prompt/skill-layer only. It adds no deterministic Go runtime,
scheduler, claim store, schema, or second Question lifecycle. Its tests must
read the real embedded/authored content, validate rendered includes, and use
the existing parity fixture; they must not simulate agent-team execution or
invent workflow-mutation tests.

No task specs exist, so no task-to-feature drift exists. The feature's planned
implementation surface matches the governing proposal: topology adapter,
council/sprint wording, Rider pointers, and authored/embedded mirrors. Task
generation must preserve that scope boundary.

No I-## or X-## row names E38-F12 in the feature spec, E38 interaction map,
E38 cross-epic map, or product map. This plan does not invent identities,
contract pointers, or deferrals. TC-001 and TC-005 instead validate local F09
and E39 reuse by their documented references.

## Traceability and AC test matrix

All implementation cases belong in
`tests/contracts/e38_f12_parallel_team_topology_test.go`.

| AC | Requirements | Technique | Test case | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|
| AC-001 | REQ-F-001/002 | Contract-surface enumeration | TC-001 | Render `parallel-team.md` and `run-agent-team.md`. | `shark plan <root> --json` supplies keys only; each key reaches one teammate's keyed `shark next` before claim. Cover `select_task` and `parallel_candidates`; reject coordinator delivery claim or prompt construction. |
| AC-002 | REQ-F-003 | State-transition enumeration | TC-002 | Read selection/refill/closing procedure. | Claimed child exclusion, in-flight dedup, terminal-list check. Empty selection with a nonterminal paused/blocked/gated child reports it, never success. |
| AC-003 | REQ-F-004 | Decision table | TC-003 | Rows for absent ownership evidence, absent isolation evidence, and isolated producer/consumer. | Both missing-evidence rows use `Sequential`; producer/consumer ordering binds even with isolation evidence. |
| AC-004 | REQ-F-005 | Contract-surface enumeration | TC-004 | Read shared-worktree procedure with/without disjoint ownership evidence. | Parallel craft needs evidence; coordinator serializes commits and `make fmt && make lint && make test`. Reject standing merge referee/concurrent gate use. |
| AC-005 | REQ-F-006 | State-transition enumeration | TC-005 | Render Question/council references for routine and material envelopes. | Teammate mints/configures/links Q; coordinator routes/resolves under Q lease; ready work refills. Material alone uses council; routine cannot bypass E39. |
| AC-006 | REQ-F-007 | Event-state transition | TC-006 | Starvation, session-boundary, all-parked scenarios. | Deterministic longest hold converts; boundary converts all holds; all-parked output names Q keys. Reject fixed Question-wait timeout. |
| AC-007 | REQ-F-008 | Attack-class enumeration | TC-007 | Config key absent/present and lease vocabulary corpus. | Export `SHARK_CLAIM_TTL_SECONDS=1800` only if config key absent; ten-minute heartbeat and expiry/replan appear. Reject TTL zero, config override, normal force-steal, second store. |
| AC-008 | REQ-F-009 | Decision table | TC-008 | Active sprint before/after `shark plan sprint`; planning/retro text. | Backlog-only selection and documented interim ordering; council evidence; owner-only start/close. Reject free selection and automatic lifecycle actions. |
| AC-009 | REQ-F-010 | Negative corpus enumeration | TC-009 | Read all sprint-team alias surfaces. | Thin `run-agent-team --sprint <S###>` alias (or final equivalent), owner close gate. Reject feature grouping/nested bootstrap; solo run-sprint stays unchanged. |
| AC-010 | REQ-F-011 | Contract-surface enumeration | TC-010 | Read isolation-integrator procedure. | One worktree/branch, serial merge/full gate, one traced fix-forward then escalation, reviewed closeout. Reject integrator Shark mutation. |
| AC-011 | REQ-F-012 | Required-field enumeration | TC-011 | Read closing report template. | Entity, teammate, outcome, merge commit, gate, waves, duration, Question counts, fix-forward count, bounded note/council destination. Reject rendered prompt/credential/transcript field. |
| AC-012 | REQ-F-013 | Fault injection plus recovery | TC-012 | Existing `compareParity` MapFS seam and sync command. | Byte drift in either tree fails; `make sync-shark-attack-skill` repairs; parity and full suite pass. Reject one-direction-only comparison. |

## Acceptance-criteria review

Every AC is unambiguous, testable, traceable, complete, and supplies an
expected outcome through the matrix above. No AC is an open-ended robustness
assertion: each has an enumerated content surface, state/event table, corpus,
or parity fault model. Every case includes an explicit negative condition.

## Caller-path contracts

All E38-F12 cases are content-only. The entrypoint is the real renderer or
direct repository-file read, and `content-only` is the justification. Tests
must not mock above a hypothetical runtime entrypoint because none is added.

| Cases | Renderer/direct-file entrypoint | Lowest allowed seam | Forbidden mocks/simulations | Counter-factual |
|---|---|---|---|---|
| TC-001--TC-011 | `sharkdata.ReadEmbedded` for embedded Shark Attack/sprint content; `os.ReadFile` for authored Rider content | Real repository/embedded file read | Orchestrator, scheduler, agent-team, claim, Question, git, or synthetic workflow-transition runtime simulation | A plausible prose-only check could pass despite missing or contradictory published procedure text. |
| TC-012 | `compareParity(authored, embedded)` MapFS fixture seam plus `make sync-shark-attack-skill` | Existing pure parity comparator | Mutating compiled embed FS or checking only authored-to-embedded direction | A one-way/snapshot parity test misses embedded-only drift and cannot prove repair. |

## ISO 25010 coverage matrix

`N/A` is deliberate: no new runtime performance behavior exists. Other N/A
cells mean the named content-only AC does not exercise that characteristic.

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-001 | N/A | ✅ TC-001 | N/A | ✅ TC-001 | ✅ TC-001 | ✅ TC-001 | ✅ TC-001 |
| AC-002 | ✅ TC-002 | N/A | ✅ TC-002 | N/A | ✅ TC-002 | ✅ TC-002 | ✅ TC-002 | ✅ TC-002 |
| AC-003 | ✅ TC-003 | N/A | ✅ TC-003 | ✅ TC-003 | ✅ TC-003 | N/A | ✅ TC-003 | ✅ TC-003 |
| AC-004 | ✅ TC-004 | N/A | N/A | N/A | ✅ TC-004 | ✅ TC-004 | ✅ TC-004 | N/A |
| AC-005 | ✅ TC-005 | N/A | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 |
| AC-006 | ✅ TC-006 | N/A | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 | ✅ TC-006 |
| AC-007 | ✅ TC-007 | N/A | ✅ TC-007 | N/A | ✅ TC-007 | ✅ TC-007 | ✅ TC-007 | ✅ TC-007 |
| AC-008 | ✅ TC-008 | N/A | ✅ TC-008 | ✅ TC-008 | ✅ TC-008 | ✅ TC-008 | ✅ TC-008 | ✅ TC-008 |
| AC-009 | ✅ TC-009 | N/A | ✅ TC-009 | ✅ TC-009 | ✅ TC-009 | N/A | ✅ TC-009 | ✅ TC-009 |
| AC-010 | ✅ TC-010 | N/A | ✅ TC-010 | N/A | ✅ TC-010 | ✅ TC-010 | ✅ TC-010 | N/A |
| AC-011 | ✅ TC-011 | N/A | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 | ✅ TC-011 |
| AC-012 | ✅ TC-012 | N/A | ✅ TC-012 | N/A | ✅ TC-012 | N/A | ✅ TC-012 | ✅ TC-012 |

## Observability design

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| E38-F12 procedures | Internal -- no new runtime observability, justified by REQ-NF-001. | Existing Shark/council evidence only. | N/A | N/A | TC-001--TC-012 reject new runtime/store contract. |
| Question/escalation handoff | Existing bounded feature note/council ledger. | Existing Question/council records. | N/A | All-parked run surfaces open Q keys. | TC-005, TC-006, TC-011 inspect required destinations and prohibited prompt persistence. |

No new instrumentation is required or permitted.

## Integration scenarios

| Scenario | Boundary | E38 UAT contribution | Test evidence |
|---|---|---|---|
| Keyed team dispatch | Rider team verb -> canonical topology procedure -> keyed `/run` | UAT-03, UAT-06 | TC-001 proves prompt/claim ownership is retained by teammate parent. |
| Claim and paused work | Selection/refill -> existing claim and Question gates | UAT-02, UAT-06, UAT-07 | TC-002/TC-006 require no false completion and open-key reporting. |
| Council and Question route | Teammate -> E39 route; coordinator -> council | UAT-04, UAT-05, UAT-07 | TC-005/TC-011 verify authority and bounded durable evidence. |
| Sprint ceremony/execution | Sprint alias -> active backlog/council planning-retro | UAT-06, UAT-07 | TC-008/TC-009 preserve owner gates and prohibit nesting. |
| Distribution parity | Authored Shark Attack tree <-> embedded mirror <-> sync | E38 acceptance gate | TC-012 proves detect, repair, clean regression. |

## Test infrastructure

- `tests/contracts/e38_f09_interactions_test.go`: use its authored versus
  embedded repository-read convention and content-contract style; do not create
  an agent-team runtime harness.
- `internal/sharkdata/shark_attack_parity_test.go`: reuse `compareParity` MapFS
  fault injection and the real-tree gate for AC-012.
- `internal/sharkdata/shark_attack_sync.go` and
  `cmd/sync-shark-attack-skill`: existing repair path, invoked by
  `make sync-shark-attack-skill`.
- `internal/cli/commands/testdata/rendered-prompts/`: existing renderer golden
  infrastructure; run/update only when changed bundle output requires it.
- No new helper is needed beyond small local read/table/corpus helpers in the
  planned F12 contract test file.

## Codex test-plan red-team

**Verdict:** PASS
**Issues raised:** 2
**Issues addressed before dev:** 2
**Issues deferred:** 0

1. Runtime simulations of the team would violate REQ-NF-001. Addressed by
content-only caller-path contracts using renderer/direct-file evidence.
2. AC-012 must prove repair and both drift directions, not detection alone.
Addressed by TC-012's MapFS comparator cases and sync recovery requirement.

The rendered dispatch prompt asks for a supplied `codex_command`, but it does
not supply one. This red-team applies its required checks directly: technique
fit, enumeration completeness, deliberate ISO coverage, no missing runtime
observability, negative cases, and no caller-path opt-out that hides a runtime
seam.

## Recommendations

- [x] Ready for development: every AC has an explicit content-contract case,
  technique, negative coverage, integration trace, and existing infrastructure.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.
