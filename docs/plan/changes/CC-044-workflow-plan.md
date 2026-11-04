# CC-044 bug-workflow repair plan

## Goal and scope

Reduce repeated repair/review churn by carrying authoritative context and accepted repair constraints through development, review, and QA. This records the agreed design direction, not an implemented or validated workflow. CC-044 owns this process change. B083 separately tracks the benchmark harness/model routing gap under E40-F13; do not expand this implementation into that repair.

The architecture consultation and review evidence are retained locally as development artifacts. Subsequent owner choices are recorded in [CC-044 integration decisions](CC-044-integration-decisions.md).

## Evidence

- Bug research currently passes directly to development. Code-review and QA failures both return to development: `internal/sharkdata/default_data/workflow/bug.yaml`.
- Existing bug research requires a reproduction and report but no independently reviewed repair plan: `internal/sharkdata/default_data/prompts/bug/research.md`.
- Bug and task development prompts allow early advancement for existing green implementations before context/rework obligations are checked.
- Shared code review uses `git diff HEAD~1`, which can omit unaccepted predecessor changes. Embedded `skills/quality/workflows/review-code.md` mandates a nested Codex invocation with failure handling conflicting with Rider deep-review.
- WWGM B110's frozen review found scanner-boundary, accounting-preservation, and helper-size defects despite an accepted route and a green local quality gate. Source: `/home/jwwel/projects/wwgm/dev-artifacts/2026-11-04-0220-b110-successor-review/overall-review.md`; route: `/home/jwwel/projects/wwgm/docs/plan/decisions/DB-2026-11-03-b099-residual-redraw-route.md`.

## Proposed stages

| Step | Owner | Required evidence | Exit and backward routing |
|---|---|---|---|
| Research | Researcher | Reproduction, causal explanation, affected callers/siblings, linked decisions, prior findings, uncertainties | Supported diagnosis -> repair plan; unresolved causal work stays research; unavailable external evidence -> blocked |
| Repair plan | Architect or developer | Repair approach, preserved behavior, authoritative constraints, implementation scope, assertions, review baseline | Reviewable proposal -> plan review; invalid diagnosis -> research |
| Plan review | Independent tech lead | Exact plan revision checked against research, code, callers, and accepted decisions | Approved -> development; approach/coverage gap -> repair plan; causal gap -> research |
| Development | Developer | Accepted plan, implementation, regression tests, contract coverage, required defect sweeps | Verified candidate -> code review; approach change -> repair plan; diagnosis change -> research |
| Code review | Existing review worker using deep-review procedure | Frozen complete scope, candidate identity, accepted plan and coverage, complete review evidence | Pass -> QA; coding defect -> development; design omission -> repair plan; causal contradiction -> research |
| QA | QA worker | Executed production-caller scenarios, boundary and preservation assertions, reviewed candidate | Pass -> existing completion/acceptance policy; rejection -> earliest invalidated step |

Use existing phase vocabulary such as `planning` and `review` for new named steps so backward-transition detection remains valid. Validate exact status names and outcome contracts before implementation. Unavailable external evidence must not be reported as repair acceptance.

Research and planning may be brief for simple bugs. Recurrent, uncertain, stateful, accounting, and cross-boundary repairs need deeper analysis. Bug research asks what failed, why, where equivalent failures may exist, and what must remain true; it is distinct from feature capability research.

## Shared contract and rework rules

Keep one coverage index: obligation -> authoritative source -> production caller/boundary -> planned change -> required assertions -> current evidence. It references existing decisions and specifications rather than superseding them. Conflicting authority goes to the owner.

Remove early-advance escape paths before required context, caller, rework, and coverage checks. Existing implementations must satisfy the same exit requirements as new work. Reuse E34-F06's canonical defect-class sweep and project-specific retry limits; do not duplicate the procedure.

Route multiple blocking findings to the earliest invalidated stage and retain every finding. Changed diagnosis invalidates planning and downstream evidence; material plan changes invalidate plan approval; candidate changes invalidate review/QA evidence. Reuse unchanged valid evidence. A failed predecessor remains within review scope until accepted.

For in-flight bugs, preserve code and history. Before further advancement, identify missing or invalid evidence and return only to the earliest affected step. Do not mass-reset existing work.

## Review integration and orchestration

Strengthen the existing code-review step; do not stack a duplicate code-review gate. Retain independent plan review because it evaluates a different artifact. Keep executed QA obligations; source review alone cannot satisfy them. Preserve existing combined SIMPLE/STANDARD feature verification and separate COMPLEX QA policy.

Reuse Rider deep-review's native Workflow -> host-dispatched six angles and consolidator -> validated comparable CLI capability ladder. Preserve complete scope, runner metadata, candidate binding, fallback reasons, and incomplete-result handling. A failed six-angle run cannot silently become a passing single-CLI review. Replace the nested `codex_command` path in integrated quality review rather than running both.

Place portable review criteria in the existing quality craft content and retain existing Rider host execution. Establish one canonical rubric source; exact packaging requires validation. `shark next <key> --json` remains the dispatch API and its prompt is passed unchanged. The existing parent owns leases, heartbeats, transitions, and result ingestion. Angle reviewers only produce evidence. No new runner, result parser, claim owner, or model router.

Adopt the existing `gate_result_v1` contract for the relevant bug gates. Configured outcomes such as `needs_research`, `replan`, and implementation rework can use `route_rework`; exact names remain an implementation choice. Verify actual dispatch and ingestion, not only schema acceptance.

## Approved model recommendations

| Work | Suggested model | Effort |
|---|---|---|
| Bounded inventory/history/caller scans | gpt-6-luna | medium |
| Routine research and repair planning | gpt-6.1-terra | high |
| Difficult causal research and repair planning | gpt-6.1-sol | high |
| Independent plan review | gpt-6.1-sol; separate Terra reviewer for simple repairs | high |
| Development | gpt-6-luna | high |
| Review A bugs/callers, D reuse/complexity, F standards/security | gpt-6.1-terra | high |
| Complex review B behavior, C contracts/siblings, E test adequacy | gpt-6.1-sol | high |
| Consolidation | gpt-6.1-sol | high |
| QA | gpt-6.1-terra | medium execution; high interpretation |
| Final acceptance/dispute | gpt-6.1-sol | high |

Reserve Astra for exceptional consultation on unresolved architectural disagreement or contradictory causal evidence after focused Sol assessment. Preserve all review/evidence obligations when reducing model cost. These are recommendations to validate, not measured quality equivalence or current workflow settings.

Current workflow steps use explicit provider/model/effort. Existing roster `capability_profile: fast|balanced|deep` is advisory, not a step-to-model mapper. B083 tracks benchmark harness switching and compatibility separately; do not introduce unsupported workflow syntax or silent parent-model inheritance. Record requested and verified effective execution identity separately, with unknown identity explicit.

## Implementation sequence

1. Create an isolated CC-044 worktree from the verified base; preserve unrelated branches and worktrees.
2. Finalize stage artifacts, freshness rules, named outcomes, and brief/deeper investigation requirements against live workflow contracts.
3. Add bug planning/review steps and entity-specific prompts; close development shortcuts and carry context/coverage into existing review and QA.
4. Integrate deep-review through existing content and host boundaries; remove duplicate review invocation and narrow-diff instructions.
5. Audit existing GateResult ingestion and review-finding telemetry. Add only demonstrated missing deterministic checks or query/report coverage; avoid a new data store.
6. Validate routing, artifact consumption, review completeness, candidate freshness, and rollout. Resolve actual effective bundle paths and local overrides before claiming WWGM receives changes.

## Verification and acceptance

- Simple repair uses brief evidence with independent plan review.
- Wrong diagnosis returns to research; inadequate plan returns to planning; coding defect returns to development.
- Omitted cost/model preservation fails plan review or verification even if disposition/call-count tests pass.
- Existing green code cannot bypass accepted context and coverage obligations.
- A material unapproved plan change or stale candidate evidence cannot satisfy its gate.
- Defect-class sweeps cover siblings, preserve valid dispositions, and classify recurrence using existing policy.
- Review includes all unaccepted predecessor code and reports incomplete angle coverage truthfully.
- Configured new outcome names survive real dispatch and result ingestion; leases remain parent-owned.
- Prompt rendering/includes and local references resolve. Review policy wording substantively; avoid tests that merely mirror prose.
- Run applicable bundle/runner tests. After Go changes, run `make fmt && make lint && make test` as required.
- Separate completed implementation, candidate review, QA, shipping, and product acceptance in reporting.

## Boundaries and open checks

Reviewers judge causal sufficiency, design quality, and meaningful assertions. Deterministic checks can validate evidence references, revisions, candidate identity, outcomes, and structured results where existing contracts support them; presence alone does not establish quality. The precise admission checks and review packaging still need implementation-level design.

The bench issue is captured as B083 and is deferred from CC-044. No workflow or application implementation is included in this artifact.
