# Test Plan: E38-F06 - Role-aware Pull and Claim Guidance

**Created:** 2026-07-14
**Feature PRD:** `docs/plan/E38-shark-attack-team-orchestration/E38-F06-role-aware-pull-and-claim/feature.md`
**Task Spec:** `docs/plan/E38-shark-attack-team-orchestration/E38-F06-role-aware-pull-and-claim/spec.md`
**Status:** APPROVED

## Scope and test strategy

F06 is a correction to the documented select-then-claim protocol plus focused
regression coverage. It must not create a claim-next operation, scheduler,
authorization layer, persistence change, or new CLI command. Tests are split
at the owning production seams:

- `SprintService.GetNextTask(ctx, agentType)` proves role filtering and the
  existing deterministic selection order with a mocked sprint repository.
- `runSprintNext` proves the CLI passes the exact `--agent` value and only
  formats the selected item, using `MockSprintService` rather than a database.
- `ClaimService.Claim(ClaimInput)` remains the atomic concurrency seam; its
  mock-repository tests prove live-claim conflicts and the absence of an
  implicit force claim.
- `tests/contracts/e38_f04_interactions_test.go` proves the embedded procedure
  states the selection/claim boundary without asserting a new runtime API.

Repository tests alone may use a real database. The service and CLI tests in
this plan use their existing mock seams; contract tests read the embedded
bundle through `sharkdata.ReadEmbedded`.

## Spec drift analysis

### Drift findings

1. **No implementation drift:** `feature.md` asks for role-filtered pull,
   existing atomic claim behavior, compatibility, focused coverage, and
   `shark-attack` documentation. `spec.md` decomposes those into REQ-F-001
   through REQ-F-006 without adding a scheduler, authorization, or data model.
2. **X-03 coverage disposition:** the E38 and global cross-epic maps now point
   to the real E38 UAT-01 and UAT-02 scenarios and the canonical shared
   `TestTC003_X03RolePullContractUsesWorkflowAndClaimAuthorities` contract
   test. F06 declares no new X-## ownership; it extends that F04-owned test
   instead of creating a duplicate consumer test.

### Traceability matrix

| Feature requirement | Specification AC | Planned evidence | Coverage |
|---|---|---|---|
| Role receives only eligible work | AC-001, AC-002 | TC-F06-001, TC-F06-002 | Yes |
| Claim race yields one winner and one conflict without force-steal | AC-005 | TC-F06-005 | Yes |
| Ordinary `sprint next` and claim/release callers stay compatible | AC-003, AC-004 | TC-F06-003, TC-F06-004 | Yes |
| Role filtering is tested and procedure documented | AC-006, AC-007 | TC-F06-006, TC-F06-007 | Yes |

## Acceptance-criteria review

All seven ACs are unambiguous, testable, traceable to REQ-F-001 through
REQ-F-006, and specify observable outputs. The negative partitions are
explicit: a role may not receive another role's work; no-match may not select
or claim a fallback; selection may not mutate a lease; a conflict may not
force-steal; and procedure text may not describe live claims as filtered out
or roster/model metadata as authority.

### Missing coverage

None. The feature declares no I-## interactions and no new X-## integration.
X-03 is preserved as F04-owned traceability and is covered by the shared
contract test specified below.

## ISTQB technique application

| AC | Technique(s) | Test cases | Rationale |
|---|---|---|---|
| AC-001 | Equivalence partitioning + stable-order comparison | TC-F06-001 | Candidate sets differ by agent type and rank. |
| AC-002 | Equivalence partitioning + negative testing | TC-F06-002 | A requested role with zero eligible candidates is a distinct partition. |
| AC-003 | Regression comparison + decision table | TC-F06-003 | Flag absent versus flag supplied must preserve the legacy output contract. |
| AC-004 | Interface contract enumeration | TC-F06-004 | CLI input/output and prohibited side effects form the adapter surface. |
| AC-005 | State-transition + race-condition enumeration | TC-F06-005 | Two claim attempts move from unclaimed to one lease plus one conflict. |
| AC-006 | Contract-surface enumeration | TC-F06-006 | Required and prohibited procedure statements are finite embedded content. |
| AC-007 | Authority decision table | TC-F06-007 | Workflow metadata versus roster, legacy assignment, and model tier are mutually exclusive authorities. |

## ISO 25010 coverage matrix

`N/A` means the characteristic does not apply to this narrow in-process CLI,
service, or Markdown-contract behavior; no network, UI, portability, or new
runtime is introduced.

| AC | Functional suitability | Performance efficiency | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | ✅ TC-F06-001 | N/A: in-memory selection | ✅ TC-F06-001 | N/A: service surface | ✅ TC-F06-001 | N/A: no auth change | ✅ focused regression | N/A: Go service |
| AC-002 | ✅ TC-F06-002 | N/A: in-memory selection | ✅ TC-F06-002 | ✅ TC-F06-002 bounded procedure result | ✅ TC-F06-002 | N/A: no auth change | ✅ focused regression | N/A: Go service |
| AC-003 | ✅ TC-F06-003 | N/A: no new work | ✅ TC-F06-003 | ✅ TC-F06-003 human and JSON output | ✅ TC-F06-003 | N/A: no secret input | ✅ adapter regression | N/A: existing CLI |
| AC-004 | ✅ TC-F06-004 | N/A: adapter only | ✅ TC-F06-004 | N/A: no new UI | ✅ TC-F06-004 | ✅ TC-F06-004 no hidden lease mutation | ✅ thin-adapter test | N/A: existing CLI |
| AC-005 | ✅ TC-F06-005 | N/A: mock concurrency seam | ✅ TC-F06-005 | ✅ bounded conflict message path | ✅ TC-F06-005 | ✅ TC-F06-005 no force-steal | ✅ existing ClaimService seam | N/A: existing service |
| AC-006 | ✅ TC-F06-006 | N/A: embedded content | ✅ TC-F06-006 | ✅ TC-F06-006 explicit outcomes | ✅ TC-F06-006 | ✅ TC-F06-006 no secret/session leakage claim | ✅ contract test | ✅ embedded bundle |
| AC-007 | ✅ TC-F06-007 | N/A: embedded content | ✅ TC-F06-007 | ✅ TC-F06-007 clear authority | ✅ TC-F06-007 | ✅ TC-F06-007 no false authorization claim | ✅ contract test | ✅ embedded bundle |

### Coverage gaps

None. Existing service and CLI tests are deterministic and mock their
repositories; no new production observability is warranted for a correction
whose behavior is already observable through existing command results and
claim-conflict errors.

## Observability design

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Role-filtered selection | Internal — no new instrumentation; selected `BacklogItemView` is existing command evidence | Existing CLI output only | N/A | N/A | TC-F06-001 / TC-F06-003 assert selected item and serialized shape |
| No eligible role item | Internal — no new instrumentation; nil result is existing bounded outcome | Existing procedure wording | N/A | N/A | TC-F06-002 / TC-F06-006 assert no fallback selection or claim |
| Live-claim race | Existing ClaimService conflict result | Existing claim error path | N/A | N/A | TC-F06-005 asserts exactly one success and `ErrAlreadyClaimed` for the other |
| Authority/procedure guidance | Internal embedded-content contract | N/A | N/A | N/A | TC-F06-006 / TC-F06-007 assert required and prohibited wording |

Implementation hook: no new metrics, logs, or traces are required. A developer
must not add observability solely to satisfy this plan; changing the existing
selection/claim boundary would require an amended plan.

## Integration scenarios

| Scenario | Components and boundary | UAT traceability | Verification |
|---|---|---|---|
| Role pull | Rider procedure → `shark sprint next --agent` → `SprintService.GetNextTask` | UAT-01 | Named workflow `agent_type` filters before the existing sprint order; no-role remains unfiltered. |
| Claim race | Role-selected entity → `ClaimService.Claim` → claim repository | UAT-02 | One active lease and one conflict; no force-steal or alternative-role retry. |
| Parent ownership | Procedure text → Rider parent loop → ordinary claim/release and workflow paths | UAT-03 | The selected item is passed to the owner path; worker does not mutate dispatched parent state. |
| X-03 preserved contract | E19 sprint/claim seams → F04-owned `pull-by-role.md` | X-03; map pointer discrepancy noted above | Shared TC-F06-006/007 remains in `tests/contracts/e38_f04_interactions_test.go`; do not add a duplicate X-03 test. |

### Cross-feature contract tests (I-##)

None. `spec.md` explicitly declares no I-## interactions for F06.

### Cross-epic integration tests (X-##)

F06 declares no new X-## row. It preserves F04's X-03 contract rather than
becoming its owner. The shared contract coverage is:

| ID | Producer | Consumer | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| X-03 (preserved) | E19 sprint and claim surfaces | E38 F04 procedure, preserved by F06 | `spec.md` API contracts and `pull-by-role.md` | `tests/contracts/e38_f04_interactions_test.go: TestTC003_X03RolePullContractUsesWorkflowAndClaimAuthorities` | TC-F06-006, TC-F06-007 additions to the same test |

## Test infrastructure

| Area | Existing pattern to follow | New helper needed |
|---|---|---|
| Sprint service | `internal/services/sprint_service_test.go` uses `MockSprintRepository` and calls the real `SprintService.GetNextTask` comparator. | No; add table rows or a focused test with mixed-agent `BacklogItem` values. |
| Sprint CLI | `internal/cli/commands/sprint_test.go` uses `MockSprintService.GetNextTaskFunc` and directly calls `runSprintNext`. | No; capture the exact `agentType` argument and assert it does not call claim/status seams. |
| Claim service | `internal/services/claim_service_test.go` uses `mockClaimRepo`; `TestClaimService_Claim_BlockedWhenLive` covers the non-force conflict. | No; add a two-attempt selected-key test only if the existing test cannot express the shared caller path. |
| Embedded procedure | `tests/contracts/e38_f04_interactions_test.go` reads `pull-by-role.md` via `sharkdata.ReadEmbedded`. | No; extend `TestTC003...` rather than create a second contract file. |

## Caller-path contracts

| TC | Production entrypoint and argument shape | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-F06-001 | `SprintService.GetNextTask(ctx, "developer")` | `MockSprintRepository.ListFunc` / `ListBacklogFunc` | Sort comparator, agent filter, or `SelectionReason` helper | A global sort before filtering returns a higher-ranked architect item. |
| TC-F06-002 | `SprintService.GetNextTask(ctx, "qa")` | `MockSprintRepository.ListFunc` / `ListBacklogFunc` | Service result or a fabricated CLI no-item response | A fallback implementation returns another role's item instead of nil. |
| TC-F06-003 | `runSprintNext(cmd, []string{})` with `--agent` absent | `MockSprintService.GetNextTaskFunc` | Real DB, output formatter, command-side selection logic | A compatibility regression changes no-agent selection or serialized fields. |
| TC-F06-004 | `runSprintNext(cmd, []string{})` with `--agent=developer` | `MockSprintService.GetNextTaskFunc` | `ClaimService`, `StatusService`, heartbeat, release, or any command-side mutation | The adapter trims/rewrites the role or performs an unrequested lease mutation. |
| TC-F06-005 | `ClaimService.Claim(ctx, ClaimInput{EntityType: "task", EntityKey: "E38-F06-001", Force: false})` twice | `mockClaimRepo.ClaimFn` / `ReclaimFn` | `ClaimService.Claim`, force release path, or a fake conflict result | A loser steals or silently replaces a live lease rather than receiving `ErrAlreadyClaimed`. |
| TC-F06-006 | `sharkdata.ReadEmbedded("skills/shark-attack/workflows/pull-by-role.md")` | Embedded bundle reader | A copied fixture, source-file-only read, or unembedded override | The shipped procedure incorrectly says selection excludes live claims or omits bounded outcomes. |
| TC-F06-007 | `sharkdata.ReadEmbedded("skills/shark-attack/workflows/pull-by-role.md")` | Embedded bundle reader | A copied fixture or assertion only on a positive authority term | The procedure lets roster role, legacy assignment, or `model_tier` override workflow metadata. |

## Acceptance test cases

### TC-F06-001: Role filter precedes deterministic sprint ordering

**Feature requirement:** REQ-F-001 and REQ-F-002; feature AC “a configured role receives only eligible work.”
**Acceptance criterion:** AC-001.
**Technique applied:** Equivalence partitioning and stable-order comparison.
**ISO 25010 characteristics:** Functional suitability, compatibility, reliability.

**Caller-path contract:** Drive `SprintService.GetNextTask(ctx, "developer")` with the real service comparator and a mocked repository. Do not mock the filter or comparator. A buggy global sort would select architect `E38-F04-001` (sprint order 1) instead of developer `E38-F06-002` (sprint order 2).

**Preconditions:** Active sprint backlog contains non-terminal architect `E38-F04-001` at sprint order 1, developer `E38-F06-002` at sprint order 2, and developer `E38-F06-003` at sprint order 3; all have distinct `AgentType` values as named.
**Input:** `agentType="developer"`.
**Expected output:** Return `E38-F06-002`; never return `E38-F04-001`; selection reason remains the existing sprint-order result among developer candidates.
**Edge cases:** A terminal developer item is excluded; a nil task `AgentType` is eligible only for the unfiltered call.
**Negative case:** No architect, QA, or nil-agent candidate is returned for the developer filter.

### TC-F06-002: No matching role returns no item and no fallback

**Feature requirement:** REQ-F-001 and REQ-F-005.
**Acceptance criterion:** AC-002.
**Technique applied:** Equivalence partitioning and negative testing.
**ISO 25010 characteristics:** Functional suitability, reliability, usability.

**Caller-path contract:** Drive `SprintService.GetNextTask(ctx, "qa")` with a mocked repository containing only active developer and architect items. Do not mock a precomputed nil result. A buggy fallback would return the globally highest item.
**Preconditions:** The active sprint has `E38-F06-002` (developer) and `E38-F04-001` (architect), and no non-terminal QA item.
**Input:** `agentType="qa"`.
**Expected output:** `(nil, nil)` (or the service's established no-item representation); procedure branch reports “no eligible work” and makes no claim call.
**Edge cases:** Terminal QA item remains ignored; an empty agent string is tested separately as unfiltered compatibility.
**Negative case:** Do not select an item for another role and do not claim anything.

### TC-F06-003: Unfiltered CLI compatibility remains byte-shape compatible

**Feature requirement:** REQ-F-002 and REQ-NF-002.
**Acceptance criterion:** AC-003.
**Technique applied:** Regression comparison and decision table.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability, reliability.

**Caller-path contract:** Drive `runSprintNext(cmd, []string{})` with no `--agent` flag and `MockSprintService.GetNextTaskFunc`; do not use a real database or bypass the formatter. A buggy adapter could introduce a default role or change JSON/human fields.
**Preconditions:** Mock service records received agent value and returns a `BacklogItemView` with all existing public fields, including sprint order and selection reason.
**Input:** No `--agent`; execute once in JSON mode and once in human-output mode.
**Expected output:** Service receives `""`; command writes the established selected-item shape and does not add a role-dependent field or alter ordering semantics.
**Edge cases:** The selection result is nil and produces the established no-item output.
**Negative case:** The adapter must not infer a role from environment, roster, or model metadata.

### TC-F06-004: CLI forwards the exact role and performs selection only

**Feature requirement:** REQ-F-001, REQ-F-003, and REQ-F-005.
**Acceptance criterion:** AC-004.
**Technique applied:** Interface contract enumeration.
**ISO 25010 characteristics:** Functional suitability, compatibility, reliability, security.

**Caller-path contract:** Drive `runSprintNext(cmd, []string{})` with `--agent=developer`; allow only `MockSprintService.GetNextTaskFunc` below the command. Do not introduce a claim, status, heartbeat, or release mock because their invocation is forbidden. A buggy command could trim/rewrite the flag or claim the selected entity.
**Preconditions:** Mock captures `agentType` and returns `E38-F06-002`.
**Input:** Literal CLI flag `--agent=developer`.
**Expected output:** Captured service argument is exactly `developer`; output serializes `E38-F06-002`; no lease or workflow method is reached.
**Edge cases:** An agent value with a valid non-default name is forwarded unchanged.
**Negative case:** No `ClaimService`, heartbeat, release, or transition is invoked by `sprint next`.

### TC-F06-005: Same selected entity has one ordinary claim winner and one conflict

**Feature requirement:** REQ-F-004 and REQ-F-006.
**Acceptance criterion:** AC-005.
**Technique applied:** State-transition and race-condition enumeration.
**ISO 25010 characteristics:** Functional suitability, reliability, security, usability.

**Caller-path contract:** Drive two sequentially controlled calls through `ClaimService.Claim` for the same selected task with `Force: false`; mock only the claim repository's reclaim and claim calls. Do not mock the service, return a fabricated conflict, or call the force-release path. A buggy implementation could steal the first lease or retry another role.
**Preconditions:** Repository accepts the first `E38-F06-001` claim and returns `claimrepo.ErrAlreadyClaimed` with the existing live claim for the second.
**Input:** Two `ClaimInput`s for the identical entity, distinct caller/session identities, both `Force: false`.
**Expected output:** Exactly one returned lease; one error preserving `ErrAlreadyClaimed`; existing live claim remains owned by the first caller.
**Edge cases:** Expired claims are reclaimed by existing TTL behavior before the first attempt.
**Negative case:** Neither caller uses `--force`, force release, an alternate-role selection, or a lease steal.

### TC-F06-006: Embedded role-pull procedure states selection/claim split and bounded outcomes

**Feature requirement:** REQ-F-003, REQ-F-005, REQ-F-006, REQ-NF-003, and REQ-NF-004.
**Acceptance criterion:** AC-006.
**Technique applied:** Contract-surface enumeration.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability, reliability, security, portability.

**Caller-path contract:** Drive the shipped artifact through `sharkdata.ReadEmbedded("skills/shark-attack/workflows/pull-by-role.md")`, extending `TestTC003_X03RolePullContractUsesWorkflowAndClaimAuthorities`. Do not read only a source fixture. A buggy bundle could ship stale wording even if the checked-out Markdown is correct.
**Preconditions:** The embedded default Shark-data bundle is available.
**Input:** Read the embedded procedure and enumerate required phrases/rules.
**Expected output:** It states that `shark sprint next --agent=<type>` is read-only role-filtered selection, selection does not exclude live claims, `ClaimService.Claim` handles conflict, and no-role/no-item/conflict/workflow-gate outcomes are bounded and parent-owned.
**Edge cases:** The procedure remains valid after embedded-bundle generation and does not expose prompts, credentials, session secrets, or unrestricted worker output.
**Negative case:** It must not claim a selection is already claimed, claim-authorized, force-claimable, or worker-transitionable.

### TC-F06-007: Procedure preserves workflow authority over roster and model metadata

**Feature requirement:** REQ-F-001, REQ-F-006, and REQ-NF-003.
**Acceptance criterion:** AC-007.
**Technique applied:** Authority decision table.
**ISO 25010 characteristics:** Functional suitability, compatibility, usability, reliability, security, portability.

**Caller-path contract:** Drive the embedded `pull-by-role.md` via `sharkdata.ReadEmbedded` in the same shared X-03 test. Do not assert only that the desired phrase exists; assert the disallowed authority sources are rejected. A buggy procedure could mention workflow role while still allowing roster or model preferences to alter selection.
**Preconditions:** Embedded procedure includes the role-pull authority section.
**Input:** Enumerate workflow-resolved `agent_type`, sprint priority/dependency order, canonical prompt metadata, `ClaimService`, roster role, legacy `agent` assignment, and `model_tier`.
**Expected output:** Workflow `agent_type`, sprint order, canonical prompt metadata, and ClaimService are named as authorities; roster role, legacy assignment, and `model_tier` are explicitly non-authoritative and do not grant claim or status authority.
**Edge cases:** A direct local `shark claim` remains a lease operation, not role authorization.
**Negative case:** No text permits roster/model data to override workflow metadata or direct local claims to be represented as authorization.

## Codex test-plan red-team

**Verdict:** UNAVAILABLE (non-blocking)
**Issues raised:** 0
**Issues addressed before dev:** 0
**Issues deferred:** 1 — external Codex review unavailable in this worker sandbox.

Attempted command (read-only, ephemeral):

```text
codex exec --sandbox read-only --ephemeral -C /home/jwwel/projects/shark-task-manager ...
```

It failed before review with: `failed to initialize in-process app-server
client: Read-only file system (os error 30)`. Per the workflow's unavailable
review rule, this is documented as a non-blocking review gap rather than
retried. The plan independently enumerates every AC's test technique, negative
partition, ISO decision, observability decision, and caller-path contract.

## Recommendations

- [x] Ready for development: every AC has concrete test coverage, a fitting
  ISTQB technique, ISO 25010 decisions, observability decision, and caller-path
  contract.
- [ ] Needs BA refinement.
- [ ] Needs technical refinement.

X-03 is resolved in the epic and global maps with UAT-01, UAT-02, and the
canonical shared contract test; no separate F06 X-03 coverage record is needed.
