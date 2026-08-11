# /shark-rider breakdown — Source document to epic portfolio

Turn one authoritative project document into a reviewed epic proposal, including
epic boundaries, delivery waves, feature-slice candidates, and cross-epic map
effects. The default mode writes a proposal only. It does not create Shark
entities or change lifecycle state.

Usage:

```text
/shark-rider breakdown <docs-path> [--output=<docs-path>]
/shark-rider breakdown <approved-breakdown-path> --create
```

This is a Mode-3 product-planning recipe. Use it for a gap analysis, strategy,
roadmap, product brief, or other document that may contain more than one epic.
Use `/shark-rider vision` when the input is already one cohesive epic idea.

## Resolve the mode and document

Accept one positional Markdown path and either no flag, one `--output` flag, or
the `--create` flag. Reject unknown flags, additional positional arguments, and
a missing target.

Resolve the target to a regular Markdown file beneath the project `docs/`
directory. Reject traversal, symlinks that resolve outside `docs/`, non-Markdown
files, and unresolved paths. Read the file in full.

In proposal mode, default the output to `<source-stem>-epic-breakdown.md` beside
the source. Require confirmation before overwriting an existing output file.
The output must also remain beneath `docs/`.

In create mode, treat the target as an approved breakdown document. Re-run all
current-state checks before proposing mutations. The document is evidence of a
proposal, not proof of approval; show the exact creation and relationship delta
and require explicit owner confirmation before the first write.

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

Read live Shark state before deciding whether work is new, reopened, superseded,
or already covered:

```bash
shark list --all --json
shark get <potentially-overlapping-epic-key> --json
shark links <potentially-overlapping-epic-key> --json
```

Use `docs/product/progress.md` as a product-document decision log, not as live
execution state. Use Shark as the lifecycle and relationship authority. Treat
code, tests, artifacts, and current runtime evidence as delivery truth; a
completed workflow status alone does not prove that the product outcome works.

Inspect sprint evidence only when it exists and can improve a delivery claim:

```bash
shark sprint list --json
shark sprint velocity --json
```

Do not invent velocity, capacity, sprint count, dates, or elapsed-time
estimates. If there is no comparable completed-sprint history, report delivery
forecast confidence as unknown.

## Extract outcomes before drawing boundaries

Build a source-to-outcome inventory. For each candidate outcome, record:

- the user, operator, or product behavior that changes;
- the observable artifact, state transition, answer, or capability produced;
- a measurable success signal and the evidence needed to verify it;
- current prerequisites and existing capabilities that remain in place;
- consumers that need its output;
- the consequence if the outcome is delayed, cancelled, or descoped;
- uncertainty that needs a bounded research or validation gate.

Separate outcomes from activities. “Build a service,” “add a schema,” “write an
ADR,” and “redesign the UI” are implementation activities until they identify
who receives value and what observable result changes.

## Choose epic boundaries

Treat an epic as one independently governable product outcome that can be
accepted, sequenced, paused, cancelled, or superseded without making its
neighbors incoherent. Apply every boundary test below.

### Outcome and acceptance

- Give the epic one primary outcome and one plain-language reason to exist.
- Require an epic-level acceptance scenario with a real trigger, a production
  path, an observable result, and durable evidence.
- Ensure the epic produces a meaningful increment even if later epics never
  start. A contract, benchmark, migration, or operational control may be the
  increment when it is usable by a named consumer.
- Do not accept an epic whose success criterion is owned entirely by a later
  epic. Move acceptance to the activation owner or redraw the boundary.

### Cohesion and interactions

- Keep work together when its feature interactions are dense, unstable, and
  jointly required to prove one outcome.
- Split work when a stable, testable contract lets a producer and consumer
  deliver and evolve independently.
- Avoid layer-only epics such as “database,” “backend,” or “UI” unless that
  layer itself creates an independently accepted operational or product result.
- Give every cross-epic handoff one producer, at least one consumer, a concrete
  payload or state, an acceptance owner, and an integration test location.
- Reject circular acceptance. If two epics can pass only after each other is
  complete, redraw the slices or name a single activation owner.

### Risk and learning

- Put a high-uncertainty assumption behind the earliest bounded learning gate
  that can change the roadmap.
- Keep research inside the outcome epic unless the research produces a durable,
  independently actionable decision or oracle used by several epics.
- Separate deterministic foundations from research-grade inference when they
  have different evidence, cost, failure, or stop conditions.
- Record an explicit stop or descoping trigger for speculative work.

### Lifecycle independence

- Check whether the epic can be prioritized and reported independently.
- Check whether cancelling it leaves a coherent portfolio and an explicit
  disposition for its consumers.
- Check whether the epic changes the meaning of a completed epic. Require an
  ADR or product decision and a live Shark reconciliation plan instead of
  silently rewriting history.
- Prefer the smallest durable home when existing epic scope already owns the
  outcome; propose a feature, change-card, tech-debt item, or reopen decision
  instead of a duplicate epic.

## Validate feature and sprint fit

Scrum does not supply a standard epic size. In Shark, a sprint is a time-boxed
container for tasks, bugs, change-cards, and tech-debt. Epics and features are
not sprint assignments. Sprint capacity, pull order, burndown, and velocity are
derived from the assigned leaf entities.

Use these implications when testing an epic boundary:

- Let an epic span multiple sprints when its outcome remains coherent.
- Identify two or more likely feature slices when the outcome genuinely needs
  them; do not invent filler to force epic classification.
- Make each feature candidate a vertical, independently demonstrable increment
  with its own trigger, production path, observable result, and UAT scenario.
- Target feature sizes `1`, `2`, `3`, or `5`; split likely `8` or `13` features
  before an eventual epic decomposition can pass.
- Expect eventual sprint-ready tasks to target `1`, `2`, or `3`; split a task
  estimated at `5` or larger before sprint planning.
- Describe delivery as dependency waves and earliest usable increments, not
  invented dates. Use live velocity only after comparable tasks have been sized
  and completed.
- Treat sprint order as an execution override within one sprint. It does not
  replace feature `execution_order`, hard dependencies, or cross-epic contracts.

Split a proposed epic again when it has no usable increment before every
feature finishes, contains several unrelated success measures, has a long
serial feature chain with stable handoffs, mixes a learning gate with
committed delivery, or cannot explain how leaf work could become sprint-ready.

Merge proposed epics when they repeat the same acceptance scenario, exchange
an unstable internal contract, cannot be cancelled independently, or create
governance overhead without a separately meaningful outcome.

## Reconcile interaction and roadmap effects

Keep three planning layers separate:

1. Delivery waves describe why-now sequence and safe parallelism.
2. Shark `depends_on` represents a hard completion barrier. Use softer links
   and notes for phase-aware coordination that is not a completion barrier.
3. `docs/product/cross-epic-integration-map.md` records product-level X-##
   contracts. It is not the execution ledger.

For every proposed epic, compare its handoffs with the current global map:

- **reuse** an existing X-## when the producer, consumer intent, and contract
  remain materially the same;
- **amend** an X-## when the contract remains but a new epic consumes, validates,
  replaces, or deprecates part of it;
- **retire or supersede** a row only with an explicit disposition and preserved
  history;
- **propose a new row** when a new cross-epic payload or lifecycle handoff
  exists.

In a proposal document, use candidate IDs such as `C-01`; do not mint X-## or
I-## IDs outside their authoritative maps. Epic design assigns I-## IDs for
cross-feature interactions. An approved global-map update assigns stable X-##
IDs.

For each candidate interaction, record producer, consumer, payload or state,
contract source, user or operator handoff, gate mode, acceptance owner,
dependency type, and test evidence. Report both how the new epic affects
existing rows and how existing contracts constrain the new epic.

## Write the proposal

Write the following sections to the resolved output path:

1. **Decision summary** — recommended epic count, why these boundaries, and the
   most consequential owner decisions.
2. **Source outcome inventory** — requirements and measures traced to source
   sections.
3. **Current-state reconciliation** — overlap with live epics, completed-state
   contradictions, and reuse or reopen recommendations.
4. **Proposed epics** — for each candidate: title, outcome, beneficiaries,
   scope, exclusions, acceptance evidence, first usable increment, likely
   feature slices, dependencies, cancellation effect, risk gate, and source
   trace.
5. **Boundary alternatives** — credible larger and smaller groupings, with the
   reason each was rejected or retained as an owner choice.
6. **Interaction matrix** — candidate cross-epic and major intra-epic handoffs.
7. **Delivery waves** — hard prerequisites, safe parallel work, integration
   points, and the earliest usable result in each wave.
8. **Sprint-fit assessment** — likely sprint-ready slices, sizing unknowns, and
   live velocity evidence if available. Do not assign work to a sprint.
9. **Cross-epic map delta** — reuse, amend, retire, and proposed rows; leave
   authoritative X-## assignment pending approval.
10. **Proposed Shark changes** — entities, hard and soft links, existing-state
    reconciliations, and related-doc links that create mode would apply.
11. **Open decisions and confidence** — what the owner must decide, why, impact,
    recommendation, and evidence gaps.

End with an explicit statement that proposal mode made no Shark entity,
lifecycle, sprint, or authoritative cross-epic-map changes.

## Apply an approved breakdown

Run this section only for an explicit `--create` invocation and after the owner
confirms the exact delta. Re-read the source, proposal, live epic list, relevant
epics, links, and the global map immediately before writing.

For each approved new epic:

```bash
shark create epic "<title>" --description="<one-sentence outcome>" --json
```

Capture the assigned key from each response. Replace each generated placeholder
with a concise charter containing the approved outcome, scope, exclusions,
success measures, first usable increment, dependencies, and source-document
breadcrumb. Do not front-load feature PRDs.

Add only approved hard completion barriers and softer relationships, using the
actual assigned keys. Link the approved breakdown to each created epic:

```bash
shark link <consumer-key> <producer-key> --type=depends_on
shark link <epic-key> <related-key> --type=related_to
shark related-docs add "Epic breakdown source" <approved-breakdown-path> --epic=<epic-key>
```

Update `docs/product/cross-epic-integration-map.md` only for the approved rows,
assign stable X-## IDs there, and update `docs/product/progress.md` with a
decision-log entry. Update the breakdown document with the assigned epic and
X-## keys so the proposal does not remain detached from live state.

For a reopen, supersede, cancel, or completed-state reconciliation, show the
specific lifecycle action and evidence first. Do not use `--force`, do not
change historical state silently, and do not perform the action unless the
owner approved that exact disposition.

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

- Do not turn source headings directly into epics without applying the boundary
  tests.
- Do not treat architecture layers, teams, or repositories as automatic epic
  boundaries.
- Do not use completion status as product acceptance evidence.
- Do not use a new epic to hide an unresolved ownership or integration decision.
- Do not change live Shark state or authoritative maps in proposal mode.
- Do not assign delivery dates or sprint counts without live, comparable
  velocity and refined leaf estimates.
