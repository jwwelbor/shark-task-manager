# Make sprint execution reach integration evidence

## Goal and authorization

The owner requested an actionable Shark plan and corresponding WWGM S010 changes on 2026-10-15. This is planning and intake, with no implementation dispatch or workflow completion. The explicit request authorizes this intake without repeating the triage confirmation question.

The consumer is WWGM's existing S010 goal: one clean complete-book Phase 0A and three qualifying generated-character/session journeys. Shark changes are a separate delivery track; WWGM must not wait for them to ship.

## Evidence and limits

The installed `skills/shark-rider/verbs/run.md` returns to the original dispatch key. The solo sprint recipe selects individual items, drives each key, and returns to selection without an explicit parent verification sweep. Live WWGM E10-F02 and E10 are active. Tasks have development → completed, while feature verification/UAT and epic integration review live on parents. Bug review/QA does not establish the whole consumer capability.

This supports a scheduling gap in the recipe. It does not prove that the engine cannot advance parents, that every prior session skipped QA, or that Shark caused incorrect implementation guidance. Feature verification may combine review and QA according to tier; a separate QA status is not universally required.

The recipe also constructs `shark sprint next {SPRINT_KEY}`, but the installed help accepts no positional key. It labels an empty response as a drained backlog and continues after a worker failure, with a default 50-iteration cap. These are distinct issues, with separate dispositions below.

All-status bugs, change cards and tech debt were enumerated, as were E19 and E34 features. Historical B045 and B047 were checked; neither is approval of the current recipe. E34-F03/F06/F07/F10 are completed. Their existing policies should be reused; capture adoption gaps without recreating those features.

## Triage dispositions

| Issue | Disposition and owner | Boundary |
| --- | --- | --- |
| Parent verification is not explicitly scheduled by the sprint recipe | [CC-048](changes/CC-048.md), process change anchored to E19-F06 | Schedule relevant parent gates without launching unrelated descendants or bypassing unfinished children. |
| Empty selection is called a drained backlog | [B080](bugs/B080.md), behavior bug anchored to E19-F06 | Distinguish finished, blocked, claimed, filtered and admission-excluded work. |
| Unsupported sprint positional argument | [B081](bugs/B081.md), behavior bug anchored to E19-F06 | Verify active sprint identity, use supported selection, and reconcile deployed recipe copies. |
| Repeated work can consume the iteration cap without removing the same blocker | [CC-049](changes/CC-049.md), process change anchored to E19-F06 | Escalate causal recurrence while preserving independent useful work and correct lease cleanup. |
| Caller-path tests and review metrics | Existing [CC-044](changes/CC-044.md); add consumer evidence | Reuse delivered E34-F06 policy; do not create a second review-hardening project. |
| Cross-entity lifecycle/state combinations omitted from planning | Existing [CC-046](changes/CC-046.md); add B051 evidence | Include durable completion, artifact state and public retry eligibility as independent axes. Reconcile delivered E34-F07 before new work. |
| Layered decomposition leaves activation for later | Existing E34-F03; add adoption evidence | Require trigger, producer, consumer, activation owner and executable proof. No duplicate feature. |
| Critical-path policy is not sufficient by itself | Existing E34-F10; add adoption evidence | WWGM lacked `docs/plan/product-critical-path.md`; this session supplies that project artifact. No claim that the guard was absent from rendered prompts. |
| Unified change creation drops required size | [B082](bugs/B082.md), CLI behavior bug | `shark create change ... --size=3` failed twice; `shark change create ... --size=3` created CC-048/049. Preserve configuration. |

The issues are not trivial edits to task-owned implementation; they need their owning workflows. Triage captured breadcrumbs only. The delivery expectations below are this separately requested plan.

## Execute the Shark track

1. Fix B081 and B080 under their normal bug research/development/review/QA paths. Verify the installed command shape and active-sprint identity. Exercise an actually finished sprint plus blocked, leased, filtered and admission-excluded cases; no empty result may fabricate goal acceptance. Check canonical, embedded and installed copies before choosing the authoritative edit surface.
2. Refine CC-048 and CC-049 together against the current selector and parent workflows. Specify which parent checkpoint is due, when it can run, how unfinished children are reported, and how sprint scope prevents unrelated descendant dispatch. Use actual outcomes and evidence changes for recurrence handling; a renamed task or a fresh run is not progress by itself.
3. Reconcile CC-044/046 with completed E34-F03/F06/F07/F10. Implement only uncovered obligations. Require a production caller, explicit mock boundary, shared test pointer and fault/recovery state matrix before the relevant development step. Preserve exact-candidate review and finding disposition.
4. Fix B082 independently through the CLI workflow. Confirm unified and entity-specific creation forward the same explicit size without relaxing required-size configuration.
5. Validate content changes using rendering/include/reference checks. Use behavioral runner/selector tests only for executable changes and the project Go gate for any Go edits. Obtain normal review, QA and delivery evidence; a policy string assertion alone does not demonstrate execution behavior.

## Verify the combined behavior

Use an isolated project with leaf tasks and standalones plus their parent feature/epic. Show that leaf completion exposes the due parent gate, blocked children retain an honest blocker, unrelated siblings stay outside the selected scope, and missing evidence prevents completion. Demonstrate recurrence escalation before repeating the same unsuccessful boundary indefinitely. Keep sprint goal review and explicit close authorization as separate final gates.

The WWGM application counterexample is B051: extraction evidence can be committed while typed artifacts are absent and retry excludes a false-complete plan item. Its real service-path reproduction now exists. That defect needs product regression coverage under WWGM B051, not a simulated Shark policy test.

## Consumer plan and checkpoint

WWGM's plan is `../wwgm/docs/plan/sprints/S010-integration-and-delivery-plan.md`, with the canonical path at `../wwgm/docs/plan/product-critical-path.md`. It keeps E10-F02 as acceptance owner, adds a bounded curation/package integration test owner, reuses the existing generated-composition/lifecycle/runner owners, and makes parent verification an explicit coordinator checkpoint.

Do not mark any of these Shark items delivered from this document. No application tests, Shark implementation tests or workflow gates ran during intake.
