---
feature_key: E38-F06-role-aware-pull-and-claim-guidance
epic_key: E38
title: Role-aware Pull and Claim Guidance
status: proposed
---

# E38-F06 Role-aware Pull and Claim Guidance

This specification is incremental over the [E38 epic PRD](../epic.md),
especially its assignment and worker-authority decisions, and over the active
architecture boundary in [architecture.md](../architecture.md). It refines the
role-pull procedure published by E38-F04; it does not revive the superseded
team-runtime design described elsewhere in the historical architecture record.

The implementation audit found that `SprintService.GetNextTask` already applies
the supplied agent-type filter before it orders candidates, and `ClaimService`
already provides the required single-entity atomic-claim and lease behavior.
The required product correction is therefore narrow: make the published
procedure accurately distinguish selection from claiming, and add focused
regression coverage around that boundary. No new claim-next operation, role
store, assignment model, scheduler, or database migration is part of this
feature.

## Requirements

### Functional requirements

| ID | Requirement | Traceability |
|---|---|---|
| REQ-F-001 | The role-pull procedure MUST accept only the workflow-resolved `agent_type` as its eligibility input and pass it unchanged to `shark sprint next --agent=<type>` / `SprintService.GetNextTask(ctx, agentType)`. Roster role, legacy assignment terminology, actor identity, and model preference MUST NOT alter selection. | Epic assignment boundary; F04 REQ-F-007; X-03 |
| REQ-F-002 | `SprintService.GetNextTask` MUST filter non-terminal sprint candidates by the requested agent type before its existing deterministic ordering. With no agent type, it MUST preserve existing unfiltered selection and ordering. | `internal/services/sprint_service.go`; feature acceptance criteria |
| REQ-F-003 | A successful role pull is a read-only selection of one candidate, not a claim. The procedure MUST pass only that returned entity to the owning claim path and MUST NOT describe the selection service as excluding live claims. | `internal/services/sprint_service.go`; `internal/services/claim_service.go` |
| REQ-F-004 | The owning claim path MUST continue to use `ClaimService.Claim` for the selected entity. A live-claim race is an explicit conflict outcome; workers MUST NOT retry by selecting another role, force-claim, or steal a lease. | Feature acceptance criteria; ClaimService contract |
| REQ-F-005 | The procedure MUST return explicit bounded guidance for: no role resolved, no eligible item, claim conflict, and a workflow pause/gate. Each outcome leaves workflow state and the dispatched parent lease with the owning Rider loop. | Epic worker-authority boundary; F04 worker-ownership protocol |
| REQ-F-006 | Documentation and contract tests MUST identify the selection/claim split: selection respects role and sprint order; `ClaimService` owns session generation, expiry reclamation, conflict reporting, heartbeat, and session-scoped release. | X-03; E38 UAT-01, UAT-02 |

### Non-functional requirements

| ID | Requirement | Verification |
|---|---|---|
| REQ-NF-001 | The feature MUST not add a database table, migration, new CLI command, scheduler, or alternate workflow engine. | Diff and architecture review |
| REQ-NF-002 | The role filter MUST remain deterministic, applied before sorting, and must not change the existing unfiltered four-tier sprint ordering. | Focused service regression tests |
| REQ-NF-003 | Role metadata is advisory to the authorized pull procedure, not a credential. The procedure MUST not claim that a direct local `shark claim` call performs role authorization. | Procedure contract test and documentation review |
| REQ-NF-004 | The procedure MUST not expose prompts, credentials, session secrets, or unrestricted worker output. Claim/session identifiers are used only by the existing owner paths. | Procedure contract test and content review |

### Acceptance criteria

| ID | Testable criterion |
|---|---|
| AC-001 | Given active-sprint candidates for at least two agent types, `GetNextTask(ctx, "developer")` returns the highest-ranked non-terminal developer item even when a different role has a higher global rank. |
| AC-002 | Given no non-terminal candidate for the requested agent type, `GetNextTask` returns no item; the procedure reports no eligible work and does not claim another item. |
| AC-003 | Given no `--agent` flag, `shark sprint next` preserves its current candidate set, four-tier order, JSON shape, and human output. |
| AC-004 | The CLI adapter passes the exact `--agent` value to the sprint service and does not claim, heartbeat, release, or transition the returned entity. |
| AC-005 | Two callers that attempt to claim the same role-selected entity produce one claim and one existing-claim conflict through `ClaimService`; neither procedure path uses `--force`. |
| AC-006 | The embedded `pull-by-role.md` procedure says that selection does not exclude live claims and names claim conflict as the bounded follow-up outcome. |
| AC-007 | The role-pull contract continues to state that workflow-resolved `agent_type`, sprint priority/dependency order, canonical prompt metadata, and the existing claim service are the authorities; roster role and `model_tier` are not. |

### Out of scope

- A new atomic claim-next endpoint or CLI command. If a future atomic
  select-and-claim operation is required, it belongs to the sprint/claim owning
  surface rather than this guidance feature.
- Server-side authorization for arbitrary direct local `shark claim` calls.
- A team ledger, scheduler, aggregate outcome router, resume store, or new
  `internal/team` package.
- Reinstating legacy task agent assignment as planning authority, or allowing a
  roster/model preference to override workflow metadata.
- Worker-owned transitions, force-steals, or mutation of a dispatched parent
  entity's claim/lease.

## Architecture

### Component changes

| Path | Change | Ownership boundary |
|---|---|---|
| `internal/sharkdata/default_data/skills/shark-attack/workflows/pull-by-role.md` | Correct the procedure to describe `sprint next --agent` as role-filtered selection only; remove the inaccurate assertion that it excludes already-claimed work; add explicit no-item, claim-conflict, and workflow-gate outcomes. | E38-F06 owns the portable guidance; it does not implement selection or leases. |
| `tests/contracts/e38_f04_interactions_test.go` | Extend TC-003 (or add its E38-F06 successor in the same contract file) to require the corrected selection/claim wording and the prohibited force-claim behavior. | Shared X-03 procedure contract. |
| `internal/services/sprint_service_test.go` | Add a focused `GetNextTask` regression test proving agent filtering happens before sorting and that no-match returns nil without weakening unfiltered ordering tests. | Sprint service owns selection. |
| `internal/cli/commands/sprint_test.go` | Add a CLI-adapter test proving `--agent` is passed through exactly and that the command only serializes/displays the selected result. | CLI remains a thin adapter. |
| `internal/services/claim_service_test.go` | Reuse or extend the existing live-claim conflict coverage only if the selected-item caller path needs an explicit assertion; retain the existing no-force default. | Claim service owns atomic lease behavior. |

No production Go source change is planned by this feature: the audited
`internal/services/sprint_service.go`, `internal/cli/commands/sprint.go`, and
`internal/services/claim_service.go` already implement the intended boundary.
If the focused tests disprove that audit, the smallest correction is limited to
the owning service or thin CLI adapter that fails the test; it must not add a
new cross-service runtime or persistence surface.

### Data model changes

There are no database schema, migration, model, repository, or configuration
changes. Existing data remains authoritative:

- `tasks.agent_type` is the persisted task metadata exposed in sprint backlog
  projections; the role-pull procedure supplies the corresponding
  workflow-resolved `agent_type` filter.
- `sprint_assignments` supplies sprint membership and ordering; it does not
  store a worker lease or a role authorization grant.
- `entity_claims` and work sessions remain the lease/session records owned by
  `ClaimService`. They are deliberately consulted only by the claim path, not
  asserted to be part of sprint selection.

### API and interface contracts

The public procedure uses existing interfaces only:

| Surface | Contract |
|---|---|
| `shark sprint next --agent=<type>` | Read-only selection. It delegates the exact flag value to `SprintService.GetNextTask(ctx, agentType)` and returns the highest-ranked non-terminal sprint item whose persisted agent type equals the requested workflow role. No flag retains the existing unfiltered behavior. A no-match returns no item. |
| `SprintService.GetNextTask(ctx, agentType)` | Filters before its stable four-tier order: sprint order, execution order, priority, then assignment time. It does not acquire or release a claim. |
| `ClaimService.Claim(ClaimInput)` | Acquires a lease for exactly the selected entity, generates or honors a session ID, reclaims expired leases according to the configured TTL, and returns a live-claim conflict without force. It is the concurrency boundary for races. |
| `ClaimService.Heartbeat` and session-scoped `Release` | Remain the sole existing mechanisms for lease renewal and safe release after successful claim. The procedure does not duplicate their policy. |
| Shark Rider parent loop | Resolves workflow role/prompt metadata, owns the dispatched parent lease and transitions, and records semantic outcomes. A role-pull worker returns an outcome; it does not alter parent workflow state. |

### Key technical decisions

| Decision | Rationale |
|---|---|
| Preserve the two-step select then claim flow. | The code already separates deterministic selection from atomic lease acquisition. Pretending selection is claim-aware would conceal an expected race and require an out-of-scope atomic claim-next design. |
| Correct the procedure instead of adding a new CLI surface. | `sprint next --agent` already passes the filter to the service and `ClaimService` already reports live-claim conflicts. The observed defect is inaccurate protocol guidance, not a missing runtime primitive. |
| Treat workflow-resolved `agent_type` as the procedure input. | This preserves the workflow as assignment authority and prevents roster/model metadata from becoming a competing selector. |
| Keep direct-CLI authorization out of scope. | Local CLI claims are lease operations, not an identity/authorization system. The safe guarantee here is that the sanctioned Rider procedure only submits the role-selected entity to the claim service. |
| Test the service and adapter separately. | A service test proves filter-before-sort behavior; a CLI test proves flag forwarding and no hidden mutation. Existing claim-service tests remain the concurrency proof. |

### Integration with existing code

- The selection adapter is `runSprintNext` in
  `internal/cli/commands/sprint.go`; it reads `--agent`, calls the sprint
  service, and formats the returned `BacklogItemView`.
- The selection implementation is `SprintService.GetNextTask` in
  `internal/services/sprint_service.go`. Its non-terminal filtering and stable
  ordering remain unchanged; the new regression coverage protects the existing
  pre-sort `agentType` comparison.
- Sprint backlog projections come from `internal/repository/sprint/repository.go`.
  Task rows project `tasks.agent_type`; entities without that field remain
  eligible for unfiltered pulls but do not satisfy a named task-role filter.
- Lease behavior follows `internal/services/claim_service.go` and the
  `internal/cli/commands/claim.go` adapter. Existing `Claim`, `Heartbeat`, and
  session-scoped `Release` semantics remain intact.
- The distributable procedure remains under the embedded Shark-data bundle,
  extending F04's `pull-by-role.md` rather than creating a second Rider or
  Shark skill.

## Cross-feature interactions

None. The parent interaction map does not assign an I-## contract to E38-F06.
This feature corrects a local procedure owned by E38-F04 and adds regression
coverage; it neither produces nor changes the mapped I-04 council-message
payload.

## Cross-epic integrations

No additional X-## declaration is required. E38-cross-epic-map.md assigns
X-03 to E38-F04 as the E19-to-E38 owner. F06 preserves that exact contract
without changing its producer, consumer, shape source, or ownership: E19
sprint and claim surfaces produce it; the E38-F04 procedure consumes it; the
UX/CX handoff is that the Scrum Master monitors sequence while specialists
self-pull eligible work and legacy assignment is not revived. Its coverage is
E38 UAT-01, UAT-02, and the canonical shared
`tests/contracts/e38_f04_interactions_test.go#TC-003`; F06 extends that one
contract test rather than creating a competing X-03 test.

## Exit-gate checklist

- [x] Every requirement is testable and traces to the epic, F04 protocol, or X-03.
- [x] The design names the current implementation seams and preserves their authority boundaries.
- [x] Every planned documentation and test change has a concrete path; no production source change is required by the completed audit.
- [x] No critical TBD remains: live-claim races are explicit outcomes, not hidden selection behavior.
- [x] No I-## or X-## record is misattributed: F06 preserves the F04-owned X-03 contract without changing the parent maps.
