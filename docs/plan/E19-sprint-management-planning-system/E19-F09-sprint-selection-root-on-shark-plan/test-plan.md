# Test plan: E19-F09 — Sprint selection root on `shark plan`

**Status:** APPROVED

## Spec drift analysis

The feature brief and `spec.md` agree: one read-only selector ranks by sprint
order, excludes terminal/claimed/Question-gated work, supports a planning
preview, and retains `sprint next` only as a compatibility projection. No
scope drift, schema work, or new dispatch path is introduced.

## AC test matrix

| AC | Technique | Cases | Expected evidence |
| --- | --- | --- | --- |
| Active selection and re-plan | Decision table and BVA | TC-001, TC-002 | claim, Question, terminal, and cap behavior is exact. |
| Planning preview | State transition | TC-003 | planning is read-only; unsupported states stop. |
| Role eligibility | Equivalence partitioning | TC-004 | current workflow role wins over display metadata. |
| Direct versus expansion | Contract-surface enumeration | TC-005 | all six assignment types retain their dispatch boundary. |
| Compatibility alias | Differential contract | TC-006 | legacy winner equals sequential plan winner. |

## Caller-path contracts

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
| --- | --- | --- | --- | --- |
| TC-001 | shared sprint-selection service with active selector, no role, cap 5 | sprint repository, claim reader, Question reader, workflow service | selector and prefiltered fixtures | claimed or gated work returns again. |
| TC-002 | `runPlan(cmd, ["sprint"])` | plan-selection service | `runPlan` itself | command bypasses the shared selector or writes. |
| TC-003 | `runPlan(cmd, ["S001"])` | plan-selection service | command and response renderer | planning is treated as executable dispatch. |
| TC-004 | shared selector with `--agent developer` | workflow metadata provider | persisted `BacklogItem.AgentType` | stale display metadata authorizes the wrong role. |
| TC-005 | shared selector over task, bug, change, tech-debt, feature, epic | assignment query seam | fixture that already drops hierarchy types | feature/epic assignments disappear or auto-expand. |
| TC-006 | `runSprintNext(cmd, args)` and `runPlan(cmd, ["sprint"])` | shared selection service | independent per-command winner mocks | aliases drift to different keys. |

## Acceptance test cases

### TC-001: Exclude unavailable assignments and respect cap

**Requirements:** REQ-F09-001, REQ-F09-003, REQ-F09-004, REQ-NF09-001.
**Technique:** Decision table over status/claim/Question state plus cap BVA.
**Setup:** six ordered assignments: direct task A rank 1, claimed bug B rank 2,
Question-gated change C rank 3, terminal tech-debt D rank 4, direct task E
rank 5, direct bug F rank 6.
**Expected:** A, E, F only, in that order; cap 1 returns A only; no claim,
status, or session record is written.
**Negative:** B, C, and D never appear.

### TC-002: Render the active-sprint plan response

**Requirements:** REQ-F09-001, REQ-NF09-001.
**Technique:** Contract-surface enumeration.
**Expected:** `shark plan sprint --json` emits the existing selection shape,
contains only service-eligible candidates, obeys the cap, and makes no write.

### TC-003: Render planning sprint as a preview

**Requirements:** REQ-F09-002, REQ-NF09-001.
**Technique:** State transition.
**Expected:** planning `S001` reports preview mode without a prompt or lease;
archived `S001` returns the documented unavailable/pause result.

### TC-004: Filter by current workflow role

**Requirements:** REQ-F09-004.
**Technique:** Equivalence partitioning.
**Expected:** a current `developer` step is present for developer and absent
for QA, even when persisted display agent metadata says QA. Cover each direct
assignment type.

### TC-005: Preserve direct and expansion candidates

**Requirements:** REQ-F09-005.
**Technique:** Contract-surface enumeration.
**Expected:** task, bug, change-card, and tech-debt are direct; feature and
epic return the explicit expansion marker and never trigger traversal.

### TC-006: Preserve legacy compatibility

**Requirements:** REQ-F09-006, REQ-NF09-004.
**Technique:** Differential contract test.
**Expected:** equivalent fixtures make `sprint next --agent developer` return
the first sequential plan key, preserve existing JSON fields, include the
deprecation notice in help only, and omit it from JSON.

## Integration scenarios

| Scenario | Boundary | Verification |
| --- | --- | --- |
| Claim turnover | plan selection → keyed `next` | Claim A without a status change, then verify re-plan surfaces E. |
| Question hold | Question lifecycle → selection | Open Question excludes its candidate; resolve restores eligibility. |
| X-03 | E19 selector → E38 team coordinator | Ordered role-aware candidates remain read-only; each selected key re-enters `shark next`. Consumer coverage is `tests/contracts/e38_f04_interactions_test.go#TC-003`. |

## Test infrastructure

- Reuse command override patterns from `internal/cli/commands/plan_parallel_test.go`
  and sprint command tests.
- Reuse four-tier fixtures and mock workflow/repository patterns from
  `internal/services/sprint_service_test.go` and
  `internal/services/sprint_decoupling_test.go`.
- Use an isolated real database only for persisted claim and Question-gate
  integration scenarios. CLI and service unit tests use mocks.

## ISO 25010 coverage

| AC group | Functional | Reliability | Compatibility | Maintainability |
| --- | --- | --- | --- | --- |
| Selection | TC-001, TC-002 | re-plan in TC-001 | N/A: local command contract | shared seam in TC-002 |
| Preview | TC-003 | invalid-state branch | N/A | command/service split |
| Roles and types | TC-004, TC-005 | N/A | all entity classes | workflow-derived rules |
| Compatibility | TC-006 | differential fixture | TC-006 | one shared selector |

## Observability design

Keep the existing `shark.plan` trace as the production evidence boundary. If
implementation adds selection attributes, assert sprint mode, preview state,
and candidate count in TC-002; otherwise no new instrumentation is required
for this read-only local operation.

## Codex test-plan red-team

**Verdict:** PASS. The plan enumerates all eligibility dimensions and does not
use open-ended robustness claims. Every AC has a named technique, a negative
or invalid-state case, a caller path, and a counter-factual.

## Recommendation

Ready for task generation.

RECOMMENDED OUTCOME: pass
