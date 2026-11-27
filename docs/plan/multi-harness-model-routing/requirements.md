# E41 requirements catalog

Use the [epic PRD](./epic.md) for business goals and UAT outcomes. The rows below define the behavior each feature must deliver. P0 establishes compatibility and executable routes. P1 depends on that foundation. Both priorities are required for epic acceptance.

| ID | Priority | Requirement | Journey | Verification |
|---|---|---|---|---|
| REQ-F-01 | P0 | Resolve low, medium and high from a named profile. Honor an applicable explicit pin. Explain the winning source and reject ambiguous or incompatible combinations before dispatch. | [J1](./user-journeys.md#j1-select-and-preview-a-route) | UAT-01, R01–R03 |
| REQ-F-02 | P0 | Deliver the selected model and explicit effort through each supported CLI mode. Preserve legacy result parsing and record the requested, selected and observed identity separately for each attempt. Mark unavailable observations as unknown. | [J2](./user-journeys.md#j2-qualify-and-migrate-a-workflow) | UAT-02, R04–R07 |
| REQ-F-03 | P1 | Bind the reviewer to the candidate and every contributing producer, including producers reached by fallback. Check the fresh session and every required difference at admission. Preserve ordinary negative review outcomes. | [J3](./user-journeys.md#j3-review-and-recover-a-step) | UAT-03, R08 |
| REQ-F-04 | P1 | Try configured alternates in order after eligible failures. Revalidate constraints and remaining budget on each attempt. Stop on a policy denial or valid workflow outcome. Require proof of retirement and replay safety after a worker starts. | [J3](./user-journeys.md#j3-review-and-recover-a-step) | UAT-04, R09–R10 |
| REQ-F-05 | P1 | After external routes are exhausted, allow Rider one eligible inline attempt using the keyed prompt, existing lease and admission checks. Record the execution mode and pause if inline fails. | [J3](./user-journeys.md#j3-review-and-recover-a-step) | UAT-05, R09, R13 |
| REQ-F-06 | P1 | Let operators check a local binding or a frozen catalog, distinguish unsupported combinations from unknown account access, and migrate or roll back one workflow at a run boundary. Provide release guidance for permissions and diagnostics. | [J2](./user-journeys.md#j2-qualify-and-migrate-a-workflow) | UAT-02, UAT-07, R11, R13 |
| REQ-F-07 | P1 | Freeze each E40 profile’s assignments, catalog and workload identity. Verify that workers used those assignments. Retain compatible historical readers and reject mixed-input reruns. | [J4](./user-journeys.md#j4-compare-two-profiles) | UAT-06, R12–R13 |
| REQ-F-08 | P1 | Populate an advisory local inventory of installed harness executable paths and versions during `shark admin init` when absent, or on the first routing lookup if absent. Let operators force a refresh with `shark admin harness refresh`; retain a stale last-known snapshot if refresh fails. Revalidate compatibility, entitlement and availability at launch. | [J2](./user-journeys.md#j2-qualify-and-migrate-a-workflow) | UAT-02, R11, R13 |

| ID | Priority | Quality requirement | Verification |
|---|---|---|---|
| REQ-NF-01 | P0 | Enforce approved credentials, workspace and tool access on each route. Redact prompts and secret values from routing diagnostics. Include a denied-action check in each qualified permission mode. | UAT-02, R07 |
| REQ-NF-02 | P0 | An offline preview reads pinned inputs with zero inference calls. Machine-readable codes and plain-text explanations identify the affected step, reason and corrective action without relying on color. | UAT-01, R02–R03, R11 |
| REQ-NF-03 | P1 | Bound captured output and execution attempts. Keep concurrent sibling attempts distinguishable and reject stale leases and duplicate results after restart. | UAT-02, UAT-04, R06–R07, R09–R10 |

R01–R13 refer to the [test matrix](./09-test-plan.md). During design, produce `E41-interaction-map.md` for the route decision consumed by adapters, the attempt evidence consumed by review and recovery, and the frozen plan consumed by E40 and Rider. Assign stable interaction IDs only after the shape sources resolve to sections in `architecture.md`. [Design handoff](./decisions.md#prd-document-decisions)
