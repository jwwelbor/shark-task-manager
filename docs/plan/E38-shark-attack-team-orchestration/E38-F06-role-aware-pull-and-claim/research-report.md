# Prior-Art Audit — E38-F06 Role-aware Pull and Claim Guidance

**Task:** `T-E38-F06-001`
**Scope:** Existing selection, thin CLI adapter, lease owner, embedded procedure,
and their test seams.
**Conclusion:** The Go implementation already has the intended select-then-claim
ownership boundary. The one discrepancy is the shipped procedure's assertion
that sprint selection excludes already-claimed work; it does not. Correct that
wording and add the focused regression/contract coverage described below. No
scheduler, claim-next operation, authorization layer, persistence change, or
production Go correction is indicated by this audit.

## Verified ownership boundaries

| Concern | Owning surface | Verified behavior | Boundary to preserve |
|---|---|---|---|
| Role-filtered selection and ordering | `SprintService.GetNextTask` in `internal/services/sprint_service.go` | It removes terminal items, applies a non-empty `agentType` filter before collecting candidates for the stable four-tier sort, then returns one `BacklogItemView` or `nil`. | Selection only; it has no `ClaimService` dependency and must not become claim-aware. |
| CLI selection adapter | `runSprintNext` in `internal/cli/commands/sprint.go` | It reads `--agent`, passes the string unchanged to `GetNextTask`, and serializes or prints the returned value/no-item result. | Keep the command thin: no claim, heartbeat, release, workflow transition, or role inference. |
| Live-lease race and session policy | `ClaimService.Claim` in `internal/services/claim_service.go` | It reclaims expired leases, creates/honors a session ID, delegates the atomic claim, reports `ErrAlreadyClaimed` for a live lease unless `Force` is explicitly supplied, and owns force-steal policy. | The role-pull procedure uses the ordinary `Force: false` path for exactly the selected entity; it must not select an alternative role/item after conflict. |
| Distributable role-pull protocol | `internal/sharkdata/default_data/skills/shark-attack/workflows/pull-by-role.md` | It already identifies workflow-resolved `agent_type`, the sprint command/service, ClaimService, canonical prompt metadata, and the non-authority of roster/legacy/model data. | It must accurately describe selection as read-only and leave claims/races to ClaimService. |

## Evidence and audit result

### Selection filters before sorting

`GetNextTask(ctx, agentType)` declares the role filter in its contract and
applies `agentType != "" && item.AgentType != agentType` while iterating
non-terminal backlog items, before `sort.SliceStable` runs. Its deterministic
sort remains: sprint order, execution order, priority, then assignment time.
With an empty agent type the condition is skipped, preserving the unfiltered
candidate set and order.

The service does not carry a claim repository or ClaimService dependency
(`SprintService` holds sprint/workflow/capacity dependencies only). Therefore it
cannot and does not exclude current live claims. This is intentional: a claim
can change after read-only selection, so conflict belongs to the atomic claim
step rather than a stale selection predicate.

### CLI remains selection-only

`runSprintNext` calls only `cmd.Flags().GetString("agent")` and
`getSprintAssignmentService().GetNextTask(cmd.Context(), agentType)`, followed
by output formatting. A search of that command and `SprintService` finds no
`Claim` call. The command's no-item result also distinguishes a supplied agent
type in human output without selecting a fallback item.

### Claim conflict remains ClaimService-owned

`ClaimService.Claim` first performs TTL-based expired-lease reclamation, then
attempts the exact entity claim. On `ErrAlreadyClaimed` with `Force: false`, it
returns the conflict (including existing-holder detail when available) and does
not release or retry. `Force` is an explicit administrative capability in the
service, not a behavior for the role-pull procedure. Heartbeat and
session-scoped release likewise stay on ClaimService's owner path.

### Procedure discrepancy

The embedded `pull-by-role.md` currently says the sprint service "excludes
ineligible, blocked, or already-claimed work." The first two categories belong
to selection, but the last is not supported by `SprintService.GetNextTask` and
contradicts the intended select-then-claim protocol. This is the sole audit
discrepancy.

The smallest owning correction is the embedded procedure and its existing
shared contract test. It should say that `shark sprint next --agent=<type>` is
read-only, role-filtered selection; it **does not exclude live claims**. The
next bounded action is to call `ClaimService.Claim` for the returned entity;
claim conflict is reported to the parent/Rider loop, with no force-claim,
alternate-role selection, or worker-owned transition. No Go source change is
needed unless the focused tests below disprove the verified implementation.

## Existing and planned test seams

| Test-plan case | Production entrypoint and allowed seam | Existing evidence | Required focused evidence / owner |
|---|---|---|---|
| TC-F06-001 | `SprintService.GetNextTask(ctx, "developer")`; `MockSprintRepository` | `TestGetNextTask_TC001`–`TC005` prove the existing unfiltered four-tier comparator, but do not call a non-empty agent filter. | Add a mixed-role test in `internal/services/sprint_service_test.go` proving the developer item at the best developer sprint rank beats a higher-ranked architect item. |
| TC-F06-002 | `SprintService.GetNextTask(ctx, "qa")`; `MockSprintRepository` | No existing no-match role-filter test. | Add a real-service comparator test that returns `(nil, nil)` for QA when only active developer/architect items exist. |
| TC-F06-003 | `runSprintNext(cmd, []string{})` with no `--agent`; `MockSprintService.GetNextTaskFunc` | `TestSprintNext_TC015_JSONIncludesNewFields` and `TestSprintNext_TC016_HumanOutputShowsSprintOrderAndReason` exercise existing selected-result output. | Add/extend adapter coverage to capture the empty argument and verify established JSON/human/no-item shapes remain unchanged. |
| TC-F06-004 | `runSprintNext(cmd, []string{})` with `--agent=developer`; `MockSprintService.GetNextTaskFunc` | The mock seam and direct command tests exist, but none asserts a non-empty value is forwarded exactly. | Add a command test that captures literal `developer`, asserts the selected result is serialized, and introduces no claim/status/heartbeat/release seam. |
| TC-F06-005 | Two `ClaimService.Claim` calls for one entity with `Force: false`; `mockClaimRepo` | `TestClaimService_Claim_ReclaimsBeforeClaiming` proves expiry reclamation and `TestClaimService_Claim_BlockedWhenLive` proves ordinary live-claim rejection. | Reuse the current owner; add the two-attempt same-selected-key case only if needed to make the one-winner/one-conflict caller path explicit. Do not add any sprint or CLI claim logic. |
| TC-F06-006 | `sharkdata.ReadEmbedded("skills/shark-attack/workflows/pull-by-role.md")` | `TestTC003_X03RolePullContractUsesWorkflowAndClaimAuthorities` verifies the existing authority vocabulary from the embedded bundle. | Extend that test to require read-only selection, the fact that selection does not exclude live claims, ClaimService conflict handling, and bounded no-role/no-item/conflict/gate outcomes. |
| TC-F06-007 | Same embedded read in `TestTC003...` | Existing test requires workflow `agent_type`, legacy assignment, model tier, and no roster-granted authority. | Retain and strengthen negative assertions that roster, legacy assignment, and model preference cannot override workflow metadata or turn direct claim into authorization. |

## Implementation guardrails

- Preserve the current `SprintService` pre-sort agent filter and unfiltered
  behavior. Do not add a claim lookup to selection.
- Preserve `runSprintNext` as a formatter/adapter; it must not claim or mutate
  workflow/lease state.
- Preserve `ClaimService` as the only lease/session/concurrency owner. The
  procedure must use the selected entity and normal non-force claim path.
- Extend the F04-owned shared contract test instead of creating a duplicate
  X-03 consumer contract.
- The correction is documentation plus focused regression tests only; adding a
  scheduler, claim-next API, role store, migration, or new CLI command would
  exceed F06's specification.

## Verification performed

This audit was based on direct source and test inspection of the required
files. Focused package/contract tests should be run after downstream F06 test
and documentation changes; this task itself changes only this report.
