---
research_schema: 2
entity_key: E38-F12
entity_type: feature
recipe: universal
rigor: complex
categories:
  - workflow_operations
  - documentation
related_work: true
---

# Research report: E38-F12 — Parallel Team Topology

## Scope

E38-F12 is a COMPLEX, prompt/skill-layer follow-on to E38-F09. It makes
`/run-agent-team` a topology adapter: `shark plan` (or, until E19-F09 ships,
the documented client-side active-sprint backlog enumeration) selects work;
each teammate is the ordinary keyed Rider parent for exactly one assigned
entity; the coordinator owns integration and council routing. Phase 1 is the
shared-worktree ownership topology and council/sprint contract; Phase 2 adds
isolated worktrees and the thin integrator. It does not add a Shark runtime,
scheduler, claim store, schema change, synthetic graph, or provider adapter.

Vocabulary: **coordinator** (team/council and integration owner, never a
delivery-entity claimant), **teammate** (keyed Rider-loop parent), **worker**
(the craft-only child receiving `response.prompt`), **ownership topology**,
**isolation topology**, **integrator**, **rolling refill**, and **event-bounded
question hold**. The governing proposal is the decision source; this report
records evidence and implementation boundaries rather than restating it.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md` and
  `../proposal-parallel-team-integration.md` §§2, 4, 5, and 8 define the two
  axes, roles, phases, exclusions, and terminology.
- [x] `affected_implementation_or_contract` — Evidence: `skills/shark-rider/SKILL.md` (keyed `next`/Rider contract),
  `skills/shark-rider/skills/sprint-execution/workflows/run-sprint-team.md`,
  and `skills/shark-attack/{SKILL.md,context/operating-model.md}` establish
  the procedures F12 must replace or extend.
- [x] `related_work` — Evidence: parent `research.md` and `requirements.md`;
  sibling feature briefs; upstream
  `E38-F09-provider-neutral-coordination-and-live-resume/research-report.md`;
  `E39-question-and-decision-workflow-management/research-report.md`; and
  `E19-F09-sprint-selection-root-on-shark-plan/feature.md` were read.
- [x] `pattern_contract` — Evidence: `skills/shark-attack/context/operating-model.md`,
  `workflows/batch.md`, `workflows/resume.md`, and `workflows/route-question.md`
  establish independent coordination/topology selection, Sequential fallback,
  bounded replacement, and the E39 Question authority boundary.
- [x] `dependency_impact` — Evidence: `internal/services/plan_hierarchy_service.go:30-35,195-197` and
  `internal/cli/commands/plan.go:580-618` prove keyed hierarchy planning
  returns only currently claimable direct children and remains selection-only;
  `E19-F09` documents the distinct active-sprint interim and future plan root.
- [x] `cross_boundary_risks` — Evidence: the governing proposal §§2–6;
  `skills/shark-rider/SKILL.md`; and
  `skills/shark-attack/workflows/route-question.md` identify the selection /
  keyed-dispatch / lease / Question / git-integration boundaries.
- [x] `alternatives` — Evidence: governing proposal §§2, 4, 5, 6, and 9
  evaluates and rejects a second scheduler/claim store, persistent TTL=0,
  a standing merge referee, hand-rolled DAG selection, and nested
  `/run-sprint-team` team bootstraps.

## Capability map

| Capability | Upstream evidence | Decision for E38-F12 |
| --- | --- | --- |
| Parent E38 authority: Shark owns workflow/leases; host drives keyed dispatch | `research.md`; `requirements.md`; E38-F07 research report | REUSE — retain parent-owned authority; F12 changes team procedure only. |
| Council roster, durable decisions, escalation, and resume | E38-F04 research report; `skills/shark-attack/SKILL.md` | EXTEND — add parallel-team and sprint-ceremony procedure links, not another council store. |
| Role-aware selection and atomic ordinary claims | E38-F06 research report | REUSE — selection is advisory/read-only and the selected entity is claimed by its keyed Rider parent. |
| Ordinary keyed Rider loop and rendered prompt | E38-F07 research report; `skills/shark-rider/SKILL.md` | REUSE — each teammate runs this loop; no team-level prompt construction or worker state mutation. |
| Two-axis topology, provider capability, live resume, and client/embedded parity | E38-F09 research report; `skills/shark-attack/context/operating-model.md` | EXTEND — introduce the topology-adapter procedure under the existing axes and preserve the authored/embedded mirror gate. |
| First-class scoped Question lifecycle | E39 research report; `skills/shark-attack/workflows/route-question.md` | REUSE (consumer) — teammate mints/links its entity's Question; coordinator routes/responds/resolves under the Q lease. No bespoke question queue. |
| Sprint selection root | E19-F09 feature brief | EXTEND later — consume `shark plan sprint` when shipped; until then use only the proposal's documented client-side active-sprint enumeration. |
| Existing group-by-feature sprint-team and no-claim teammate template | governing proposal §6; current `run-sprint-team.md` | CONTRADICTS legacy contract — replace with a thin alias into the persistent team workflow and keyed teammate parent loop. The governing proposal resolves this; no open reconciliation remains. |

This feature **extends** the established Shark Attack/Rider capability; it
creates only the new prompt-layer `parallel-team` and integrator procedures.
It will not re-implement Shark planning, keyed prompt assembly, claims,
Question storage, provider runtimes, or sprint schema/services.

## Findings

1. V-001 is resolved: keyed `shark plan <feature>` is claim-aware. The plan
   hierarchy contract calls its children "non-terminal, unclaimed" and
   `DescribeChildren` returns the currently claimable subset. Rolling refill
   may rely on this but should retain proposal step-2 in-flight-key dedup as
   race defense; `shark plan` remains non-mutating and does not reserve work.
2. The current sprint-team workflow still groups by feature and serializes one
   nested team bootstrap at a time. It must become a thin alias into the new
   workflow's sprint mode; otherwise it duplicates selection and violates the
   one-coordinator model.
3. Parallelism is not a default. The existing two-axis contract requires
   recorded ownership evidence for shared-worktree parallelism or isolation
   evidence for worktrees; missing evidence degrades to Sequential. Logical
   producer/consumer ordering survives either parallel topology.
4. The main cross-boundary risk is authority leakage: a teammate without the
   ordinary Rider claim/advance/release loop can advance unleased work; a
   coordinator claiming delivery work creates a second parent; and a team
   prompt that rebuilds `response.prompt` would fork the dispatch source.
5. Question holds require the E39 route, not a team-side pause state.
   Persistent live holds are bounded by starvation conversion and the session
   boundary, with heartbeats while held. The coordinator owns responder
   routing and resolution; unrelated work continues.
6. Isolation adds only an integrator for serial merge-in, post-merge
   `make fmt && make lint && make test`, fix-forward traces, and reviewed
   worktree closeout. It does not add a standing merge-referee or let the
   integrator mutate Shark state.

## Decisions

1. Proceed with the proposal's two phases, Phase 1 before isolation, and keep
   the proposal as the single decision source.
2. Make `skills/shark-attack/workflows/parallel-team.md` the authored
   procedure home, synchronize its embedded mirror, and test parity. Host
   `/run-agent-team` and `/run-sprint-team` become thin pointers/aliases.
3. Use `shark plan` for free selection and its claim-aware hierarchy behavior;
   use the documented active-sprint interim until E19-F09 supplies the sprint
   plan root. Each selected delivery key re-enters through `shark next <key>`.
4. Preserve the 30-minute environment-scoped TTL decision
   (`SHARK_CLAIM_TTL_SECONDS=1800` with no config key), event-bounded Question
   holds, and sequential degradation when topology evidence is absent.
5. Treat all legacy procedure conflicts as replacement work, not as fallback
   branches. No unresolved design contradiction or external blocker prevents
   specification.

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F12-parallel-team-topology/{feature.md,assessment.md}`
- `docs/plan/E38-shark-attack-team-orchestration/{research.md,requirements.md,proposal-parallel-team-integration.md}`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F{04,F06,F07,F09}-*/research-report.md` and sibling feature briefs
- `docs/plan/E39-question-and-decision-workflow-management/research-report.md`
- `docs/plan/E19-sprint-management-planning-system/E19-F09-sprint-selection-root-on-shark-plan/feature.md`
- `skills/shark-rider/SKILL.md`, `skills/shark-rider/skills/sprint-execution/workflows/run-sprint-team.md`
- `skills/shark-attack/{SKILL.md,context/operating-model.md,workflows/batch.md,workflows/resume.md,workflows/route-question.md}`
- `internal/services/plan_hierarchy_service.go`, `internal/cli/commands/plan.go`

RECOMMENDED OUTCOME: pass
