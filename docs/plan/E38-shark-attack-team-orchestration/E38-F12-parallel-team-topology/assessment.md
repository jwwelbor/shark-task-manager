# Complexity Triage Report — Parallel Team Topology

**Feature:** E38-F12 — Parallel Team Topology
**Assessment date:** 2026-08-04
**Score:** 16/27
**Tier:** COMPLEX

## Scope validation

E38-F12 is correctly scoped as a feature. It delivers two phased, user-facing
coordination capabilities: an ownership-based team execution topology with a
council and sprint contract, and an isolation topology with integration and
closeout. It requires design decisions about authority, lease/recovery,
question holds, selection, merge gating, and compatibility; it changes multiple
skill and command contracts across more than four files. It is not a single
atomic change that can safely be carried as a task.

The governing proposal remains the design source of truth. Its formerly
blocking E38-F09 skill restructure is now present on `main` (commit
`9eb07d63`, PR #149), so this assessment does not identify an external
sequencing blocker. E19-F09 remains a non-blocking consumption dependency: the
coordinator must retain documented client-side backlog enumeration until
`shark plan sprint` is available.

## Evidence reviewed

- `feature.md` and the governing
  `../proposal-parallel-team-integration.md`, especially its deliverables,
  sequencing, and recorded decisions.
- `skills/shark-attack/context/operating-model.md` and
  `workflows/coordinate.md`: existing two-axis classification and degradation
  contract that F12 must extend without duplicating.
- `skills/shark-rider/SKILL.md`, `verbs/run.md`,
  `verbs/run-sprint-team.md`, and
  `skills/sprint-execution/workflows/run-sprint-team.md`: the existing Rider
  parent-loop and group-by-feature team contracts.
- Host `/run-agent-team` and `/run-sprint-team` command sources: the existing
  team command still uses a missing `orchestration` workflow and its teammate
  template advances Shark state without the keyed Rider parent-loop contract.
- `internal/sharkdata/default_data/skills/shark-attack/`: authored changes
  require an embedded mirror and F09's deterministic parity coverage.

## Dimension scores

### Technical complexity (6 dimensions)

1. **File impact: 3/3** — Phase 1 changes/adds the authored and embedded
   Shark Attack workflow, Shark Rider and sprint-team procedures, host command
   pointers, and rendered/contract fixtures. Phase 2 adds the integrator and
   isolation/closeout procedure and its checks. The expected surface exceeds
   ten authored, mirror, command, and verification files.
2. **Pattern novelty: 2/3** — F09 already supplies the two-axis model,
   provider references, canonical keyed-dispatch contract, and parity pattern.
   F12 nevertheless introduces a new topology-adapter procedure that must
   compose those contracts with rolling refill, council handling, and a thin
   integrator without recreating a scheduler or claim store.
3. **Data model: 0/3** — The proposal explicitly excludes schema, migration,
   second claim-store, runtime, and scheduler work.
4. **API surface: 2/3** — Existing host-facing `/run-agent-team` and
   `/run-sprint-team` command contracts change materially, but no new Shark
   CLI or HTTP endpoint is introduced.
5. **Cross-feature dependencies: 2/3** — F12 consumes E38-F09's delivered
   routing/restructure, E39's Question lifecycle, E19-F09's future sprint
   plan root, and completed E38 council/Rider contracts. No circular
   dependency is evidenced.
6. **UI complexity: 0/3** — No graphical or interactive UI is in scope;
   procedural prompts and concise terminal/operator output are the surfaces.

**Technical total:** 9/18

### Execution complexity (3 dimensions)

7. **Task estimation: 3/3** — At least eight independently testable slices
   are required: topology procedure; parity mirror and drift coverage; host
   command pointer; keyed teammate loop; selection/refill and V-001 evidence;
   council question/escalation and sprint ceremonies; sprint-team alias;
   isolation integrator/merge/closeout and regression scenarios.
8. **Regression risk: 2/3** — This is behavior-preserving replacement work on
   existing execution guidance. A mistake can bypass claims, duplicate
   scheduling, assign parent keys to teammates, strand question-blocked work,
   or remove the required integration gate.
9. **Execution effort: 2/3** — The two phases span roughly three to four
   weeks of staged specification, implementation, parity synchronization,
   rendered-prompt/contract tests, and exercised integration scenarios.

**Execution total:** 7/9
**Overall total:** 16/27

## Tier assignment

**Assigned tier:** COMPLEX

F12 crosses the COMPLEX threshold because the replacement is broad,
behavior-preserving workflow work with eight or more independently verifiable
slices and multiple cross-feature contracts. The absence of Go/schema work
reduces implementation-layer complexity, but does not make the coordination,
authority, and safety contracts safe to implement as one autonomous change.

## Autonomous-build feasibility

- Task count: at least 8 (threshold <= 7; the prompt's broader threshold of
  <= 10 is not sufficient to overcome the risk and effort disqualifiers)
- Regression risk: 2/3 (threshold <= 1)
- Execution effort: 2/3 (threshold <= 1)
- Circular dependencies: no confirmed circular dependency

**Recommendation:** manual, staged execution. Retain the proposal as the
single decision source; write a full specification after validating V-001
(keyed-plan handling of actively claimed children), split Phase 1 from Phase
2 in task generation, and preserve F09 authored/embedded parity and canonical
keyed Rider-loop tests throughout.
