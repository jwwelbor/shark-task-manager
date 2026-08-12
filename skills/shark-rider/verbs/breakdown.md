# /shark-rider breakdown — Source document to epic portfolio

Turn one authoritative project document into a reviewed epic portfolio. Define
the smallest coherent set of charter-ready epics, delivery waves, and
cross-epic map effects. Present the proposal, ask for approval, create the
approved epics in the same interaction, verify them, and stop.

Usage:

```text
/shark-rider breakdown <docs-path> [--output=<docs-path>]
```

Use this Mode-3 recipe for a gap analysis, strategy, roadmap, product brief, or
other source that may contain more than one epic. Use `/shark-rider vision`
when the input already describes one cohesive epic.

## Resolve the document

Accept one positional Markdown path and either no flag or one `--output` flag.
Reject unknown flags, additional positional arguments, and a missing target.

Resolve the target to a regular Markdown file beneath the project `docs/`
directory. Reject traversal, symlinks that resolve outside `docs/`,
non-Markdown files, and unresolved paths. Read the file in full.

Default the output to `<source-stem>-epic-breakdown.md` beside the source.
Require confirmation before overwriting an existing output file. Keep the
output beneath `docs/`. Writing the proposal does not authorize Shark or
authoritative-map changes.

## Collect current product and delivery evidence

Read the source document and the smallest relevant set of current records:

- product intent, success criteria, user needs, personas, and journeys under
  `docs/product/`;
- product roadmaps or delivery plans under `docs/plan/`;
- architecture decisions and constraints under `docs/architecture/`;
- `docs/product/cross-epic-integration-map.md` and `docs/product/progress.md`
  when present;
- existing epic charters and per-epic interaction or cross-epic maps that the
  source names or overlaps.

Read live Shark state before deciding whether work is new, reopened,
superseded, or already covered:

```bash
shark list --all --json
shark get <potentially-overlapping-epic-key> --json
shark links <potentially-overlapping-epic-key> --json
```

Use `docs/product/progress.md` as a product-document decision log, not as live
execution state. Use Shark as the lifecycle and relationship authority. Treat
code, tests, artifacts, and runtime evidence as delivery truth. A completed
workflow status alone does not prove that the product outcome works.

Do not inspect sprint capacity or velocity in this procedure. Sprint planning
starts after Shark decomposes approved epics into features and tasks. Do not
invent dates, sprint counts, or elapsed-time estimates.

## Establish intrinsic scale, then compare when useful

Decide whether each candidate is epic-sized from the source and product
outcomes first. The procedure must work when the project has no existing
epics, when prior epics use a different decomposition style, or when the source
describes a new product area.

An intrinsically epic-sized candidate:

- produces one meaningful product or operational outcome;
- needs several demonstrable increments to reach its acceptance result;
- has one coherent priority, acceptance, and cancellation decision;
- is broader than one bounded implementation, evidence, or learning step;
- is not merely an architecture layer, delivery phase, or enabling artifact;
- has a boundary that reduces governance ambiguity enough to justify any new
  cross-epic interactions.

Do not invent feature candidates to prove that an epic contains several
increments. Use the number of distinct success conditions, affected journeys
or operations, compatibility obligations, and risk gates as breadth evidence.
If those signals do not establish multi-increment breadth, classify the work
below epic level or merge it with the outcome it enables.

Comparable current or completed epics are an optional secondary drift check,
not an input required to determine boundaries. When genuinely comparable
precedents exist, record:

- its primary outcome and breadth;
- feature count, completed and remaining feature count, and known task count;
- the number and type of cross-epic contracts;
- whether it delivered a foundation, user experience, operating capability,
  or research result;
- whether the source proposes a larger product, a replacement, or remediation
  of that precedent.

Use historical feature counts and outcome breadth only to detect a material
change in portfolio granularity. They are not a quota, target, or substitute
for the intrinsic tests. Explain material deviations when the comparison is
valid. If a remediation proposal creates more epics than the product area it
replaces, require specific evidence that the new product scope is larger. Do
not accept “the source has more headings” as evidence. Do not estimate the new
feature count in this procedure.

If no comparable epics exist, state that the optional portfolio cross-check was
unavailable and continue with the intrinsic tests. Do not lower confidence
solely because precedents are absent. Lower confidence only when the source
lacks the outcome, acceptance, interaction, or breadth evidence needed to
justify a boundary.

## Extract outcomes and classify work

Build a source-to-outcome inventory. For each candidate, record:

- the person or product behavior that changes;
- the observable artifact, state transition, answer, or capability produced;
- a measurable success signal and verification evidence;
- prerequisites, consumers, and existing capabilities reused;
- the effect of delay, cancellation, or descoping;
- uncertainty that needs a bounded research or validation gate.

Separate outcomes from activities. “Build a service,” “add a schema,” “write
an ADR,” and “redesign the UI” are implementation activities until they name
who receives value and what observable result changes.

Classify each candidate at the smallest plausible Shark level before grouping
epics:

- **task** for one bounded implementation or evidence step;
- **bug** for behavior that violates an accepted contract;
- **tech-debt** for internal quality work without a new product outcome;
- **change-card** for an approved direction, lifecycle, or compatibility
  change that does not need an epic decomposition;
- **below epic** for work that belongs in later feature or task decomposition;
- **epic** for a multi-feature product outcome with independent governance.

This classification is routing, not decomposition. Do not name, count, size,
order, or write proposed features. The approved epic's Shark workflow owns
feature decomposition as a separate step.

Do not assume that each source outcome, gate, phase, or heading is an epic. An
ADR, oracle, benchmark, migration, test harness, research gate, or storage
cleanup is not automatically an epic. Keep enabling work inside the outcome
epic it unlocks unless it has its own beneficiaries, acceptance, funding or
priority decision, and cancellation path.

## Choose epic boundaries

Treat an epic as one independently governable product outcome that can be
accepted, prioritized, paused, cancelled, or superseded without making its
neighbors incoherent. Apply every boundary test below.

### Outcome and acceptance

- Give the epic one primary outcome and one plain-language reason to exist.
- Require an epic-level acceptance scenario with a real trigger, production
  path, observable result, and durable evidence.
- Ensure the epic produces a meaningful increment even if later epics never
  start.
- Do not accept an epic whose success criterion is owned entirely by a later
  epic. Move acceptance to the activation owner or merge the candidates.

### Cohesion and interactions

- Keep work together when feature interactions are dense, unstable, or jointly
  required to prove one outcome.
- Split work only when a stable, testable handoff also supports independent
  acceptance, prioritization, and cancellation. A stable contract alone does
  not require a new epic.
- Avoid layer-only epics such as “database,” “backend,” or “UI” unless that
  layer creates an independently accepted operational or product result.
- Give every cross-epic handoff one producer, at least one consumer, a concrete
  payload or state, an acceptance owner, and an integration test location.
- Reject circular acceptance. If two candidates can pass only after each other
  completes, merge them or name one activation owner.

### Risk and lifecycle

- Put a high-uncertainty assumption behind the earliest bounded learning gate
  that can change the roadmap.
- Keep research inside the outcome epic unless it produces a durable result
  with an independently funded stop-or-continue decision.
- Separate deterministic foundations from research-grade inference only when
  their acceptance and cancellation decisions are truly independent.
- Check whether cancelling the epic leaves a coherent portfolio and an
  explicit disposition for consumers.
- Prefer an existing epic, feature, change-card, or tech-debt item when it is
  the smallest durable owner. Do not create a duplicate epic.

## Run the merge challenge

Challenge every proposed epic boundary after the first pass. For each adjacent
or highly interacting pair, describe the merged outcome and keep the split only
if all of these statements are true:

1. Each side has a separately meaningful acceptance result.
2. Each side can be prioritized or cancelled independently.
3. Their handoff is stable enough to test without completing both sides.
4. The extra lifecycle and cross-epic contract overhead improves governance.

A later delivery wave is not, by itself, an epic boundary. A different team,
repository, architecture layer, gate number, or implementation phase is also
not enough.

Prefer fewer coherent epics when the evidence is ambiguous. Use delivery waves
only for cross-epic sequencing. Leave internal sequencing to later epic
decomposition.

## Check portfolio inflation and decomposition readiness

Compare the proposed epic count, acceptance outcomes, and new cross-epic
contracts with the source-derived intrinsic scale. Use portfolio precedents as
an additional check only when they are genuinely comparable. Then run these
checks:

- Flag a candidate that appears to be one bounded increment, enabling artifact,
  or implementation phase rather than a multi-increment outcome.
- Split an epic that is materially broader than precedent only when it contains
  multiple independently accepted outcomes.
- Flag a proposal that multiplies epic count relative to the product area being
  replaced without a documented increase in product scope.
- Treat each new X-## contract, separate UAT plan, lifecycle, and status report
  as real coordination cost.
- Reclassify enabling or phase-only candidates before accepting portfolio
  inflation.

For each retained epic, verify that the proposal provides:

- one clear product or operational outcome;
- measurable success criteria;
- explicit in-scope and out-of-scope boundaries;
- constraints and assumptions;
- affected people and systems;
- high-level UAT scenarios with observable results;
- dependencies, cross-epic contracts, cancellation effect, and source trace;
- enough breadth to require later decomposition into several demonstrable
  increments, without defining those increments now.

Report the original candidate count, merged candidate count, intrinsic breadth
evidence, any optional comparable portfolio breadth, and the reason any
expansion remains.

## Define the decomposition handoff

`/shark-rider breakdown` stops at charter-ready epics. Epic decomposition is a
separate Shark workflow step after creation, refinement, research, and design.
Do not propose feature titles, feature counts, feature sizes, execution order,
tasks, or sprint assignments here.

For each epic, write a short decomposition handoff that records only what the
later workflow must preserve:

- success criteria and UAT coverage that later features must collectively
  satisfy;
- interaction and contract constraints that need I-## ownership;
- risk or learning gates that affect later sequencing;
- existing components and compatibility obligations;
- unresolved decisions that must close before or during decomposition.

Do not solve those feature-boundary or ordering decisions in this procedure.
After the epic exists, use `/shark-rider run <epic-key>` to drive its normal
workflow through decomposition. Sprint planning follows task generation.

## Reconcile interactions and roadmap effects

Keep three planning layers separate:

1. Delivery waves describe why-now sequence and safe parallelism.
2. Shark `depends_on` represents a hard completion barrier. Use softer links
   and notes for phase-aware coordination that is not a completion barrier.
3. `docs/product/cross-epic-integration-map.md` records product-level X-##
   contracts. It is not the execution ledger.

For every proposed epic, compare its handoffs with the current global map:

- **reuse** an existing X-## when producer, consumer intent, and contract remain
  materially the same;
- **amend** an X-## when a new epic consumes, validates, replaces, or deprecates
  part of the contract;
- **retire or supersede** a row only with an explicit disposition and preserved
  history;
- **propose a new row** only for a genuine cross-epic payload or lifecycle
  handoff.

Use candidate IDs such as `C-01` in proposals. Do not mint X-## or I-## IDs
outside their authoritative maps. Epic design assigns I-## IDs for
cross-feature interactions. An approved global-map update assigns X-## IDs.

For each candidate interaction, record producer, consumer, payload or state,
contract source, user or operator handoff, gate mode, acceptance owner,
dependency type, and test evidence. Treat cross-epic contract overhead as a
boundary cost, not as proof that the split is correct.

## Write the proposal

Write these sections to the resolved output path:

1. **Decision summary** — recommended epic count and consequential decisions.
2. **Epic scale assessment** — intrinsic breadth evidence, original and final
   candidate counts, plus an optional precedent comparison when available.
3. **Source outcome inventory** — requirements and measures traced to source.
4. **Classification results** — which source candidates became epics, belong
   below an epic, or fit a standalone entity and why.
5. **Current-state reconciliation** — overlap, contradictions, and reuse or
   reopen recommendations.
6. **Proposed epics** — problem and justification, outcome, goals, measurable
   success criteria, scope, exclusions, constraints, stakeholder impact,
   high-level UAT, dependencies, cancellation effect, and source trace.
7. **Merge challenge and alternatives** — pairs tested, retained splits, and
   credible larger or smaller groupings.
8. **Interaction matrix** — candidate cross-epic handoffs.
9. **Delivery waves** — prerequisites, parallel work, and integration points.
10. **Decomposition handoff** — constraints and evidence the later epic
    workflow must preserve, without proposed features.
11. **Cross-epic map delta** — reuse, amend, retire, and proposed rows.
12. **Proposed Shark changes** — entities, links, reconciliations, and docs.
13. **Open decisions and confidence** — decision, reason, impact,
    recommendation, and evidence gaps.

Before asking for approval, end the draft with an explicit statement that no
Shark entity, lifecycle, sprint, or authoritative cross-epic-map changes have
been made.

## Present the proposal and ask for approval

After writing the proposal, show a concise exact delta:

- epic titles and one-sentence outcomes to create;
- existing entities to reuse, annotate, reopen, supersede, or cancel;
- hard dependencies and softer links to add;
- candidate cross-epic map rows to add, amend, or retire;
- product documents to update;
- confirmation that this action will not create features, run workflows, or
  assign sprint work.

Ask the user whether to create and apply that exact proposal. Wait for explicit
confirmation before changing Shark state or an authoritative map. If the user
requests changes, revise the proposal, show the new exact delta, and ask again.
If the user declines, stop with the proposal artifact and state that no live
changes were made.

Do not require a second invocation or a special apply flag. Approval continues
the current `/shark-rider breakdown` interaction, like Rider triage.

## Create the approved epics

After explicit approval, re-read the source, proposal, live epic list, relevant
epics, links, and global map immediately before writing. Re-run duplicate and
current-state checks. If the refresh changes the approved delta, show the
changed delta and obtain approval again. Otherwise, continue without another
confirmation.

For each approved new epic:

```bash
shark create epic "<title>" --description="<one-sentence outcome>" --json
```

Capture each assigned key. Replace generated placeholders with concise charters
containing the approved problem and justification, outcome, goals, scope,
exclusions, constraints, affected stakeholders, measures, UAT, decomposition
constraints, dependencies, and source breadcrumb. Do not create feature PRDs.

Add only approved hard completion barriers and softer relationships, using the
actual keys. Link the approved breakdown to each epic:

```bash
shark link <consumer-key> <producer-key> --type=depends_on
shark link <epic-key> <related-key> --type=related_to
shark related-docs add "Epic breakdown source" <breakdown-path> --epic=<epic-key>
```

Update `docs/product/cross-epic-integration-map.md` only for approved rows,
assign X-## IDs there, and update `docs/product/progress.md` with a decision-log
entry. Add assigned epic and X-## keys to the breakdown document.

For a reopen, supersede, cancel, or completed-state reconciliation, show the
specific lifecycle action and evidence first. Do not use `--force`, change
historical state silently, or apply a disposition that the owner did not
approve.

Verify the applied result:

```bash
shark get <created-epic-key> --json
shark links <created-epic-key> --json
shark admin validate
git diff --check
```

Stop after creation and verification. Do not run the new epics, advance their
workflow status, create features, create or start a sprint, or assign leaf work
to a sprint unless the user separately requests it.

## Boundaries

- Do not turn source headings, phases, or gates directly into epics.
- Do not treat architecture layers, teams, repositories, or delivery waves as
  automatic epic boundaries.
- Do not promote enabling work to an epic only because it is measurable.
- Do not accept a materially inflated portfolio without a scope-based reason.
- Do not name, count, size, order, or write features in this procedure.
- Do not use completion status as product acceptance evidence.
- Do not change live Shark state or authoritative maps before explicit approval.
- Do not interpret approval of one delta as approval of a changed delta.
- Do not assign dates, sprint counts, or sprint work.
