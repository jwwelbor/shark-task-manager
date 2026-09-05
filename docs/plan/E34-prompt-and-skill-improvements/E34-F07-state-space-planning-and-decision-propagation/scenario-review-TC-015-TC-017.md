---
feature_key: E34-F07-state-space-planning-and-decision-propagation
epic_key: E34
title: TC-015–TC-017 Scenario Review — Reviewed Evidence Record
task: T-E34-F07-006
---

# TC-015–TC-017 Scenario Review

Per test-plan.md's AC-5 section ("AC-5's own wording requires 'a
rendered-output sample,' not only a UAT report narrative... this plan
requires a **committed rendered artifact** proving all three scenarios,
matching E34-F06's precedent exactly"), this document is that committed
artifact. `state-space-coverage.md` contains no `{{include}}` directives, so
its rendered output is its checked-in content verbatim, reproduced in full
below (not excerpted or paraphrased); each of TC-015, TC-016, and TC-017
then walks a concrete fixture through that rendered content to its required
outcome, per test-plan.md's AC Test Matrix Input/Setup and Expected-outcome
cells. Section references use the file's own `##` headers — stable anchors
that do not shift on future edits.

This satisfies task T-E34-F07-006's scope (the AC-5 rendered-output sample)
and feature.md's three acceptance scenarios ("Plan a multi-entity lifecycle",
"Propagate a ratified decision", "Reject naming drift").

**Upstream observation (not this task's to fix):** the canonical file's
"Closed lifecycle tables" section states the table has "these six columns,"
but the table it introduces actually lists seven rows (value, meaning, entry
transitions, exit transitions, terminal/no-exit marker, invalid-transition
list, failure/recovery behavior); test-plan.md's TC-001 repeats the same
"six" count. That prose/table mismatch belongs to T-E34-F07-001's file, not
this task's scope — flagged here for the parent loop rather than corrected
in someone else's artifact.

## Rendered `state-space-coverage.md` output

The canonical source is
`internal/sharkdata/default_data/skills/quality/workflows/state-space-coverage.md`.
Full rendered content, verbatim (no template includes to resolve):

```markdown
---
inputs:
  - field_or_entity: the lifecycle field, entity, or decision under review
  - candidate_axes: interaction-map (I-##/X-##) rows, caller paths, and prior spec.md "Cross-feature interactions" sections that might name this field or entity
  - decision_record: the accepted design/decision this change may diverge from, if any (an ADR, a resolved Question, a tech-debt disposition, or a prior spec.md decision)
outputs:
  - closed_table: the field's value/meaning/transition/terminal/invalid/recovery table (see Closed lifecycle tables)
  - technique_selection: the chosen ISTQB technique for this field's test cases (see Technique selection from state shape)
  - dependency_discovery_log: per-axis inclusion/exclusion rationale (see Dependency discovery by interaction and caller path)
  - shipped_consumer_handoffs: list of {caller_path, owning_feature_key, affected_ac_ids, regression_test_pointer} entries, if any shipped consumer is affected
  - change_impact_set: one I-04 `ChangeImpactSet` record, when a decision changes already-implemented or already-specified behavior (see I-04 propagation)
  - divergence_citation: pointer to prior decision plus new evidence, when rework diverges from it (see Design divergence)
---

# Workflow: State-Space Coverage

## Purpose

This is the canonical source for four related planning-decision disciplines
that specification, test-planning, task-review, and decision-resolution
prompts reference instead of restating: how to write a closed lifecycle
table, how to pick a test-design technique from that table's shape, how to
discover every dependency on a field or entity before changing it, and how to
propagate a ratified decision to every artifact it invalidates. Callers add
one reference line to the relevant section below; they do not restate its
procedure.

## Closed lifecycle tables

A field is **behavior-bearing** — and therefore requires a closed table,
not prose — when a documented transition on it determines dispatch routing,
gate outcome, or persisted disposition. Not every enum qualifies: a purely
descriptive tag with no transition consequence does not need this table.

When a field is behavior-bearing, `specification.md` must declare a closed
table with these six columns, one row per value:

| Column | Meaning |
|---|---|
| value | the field's value |
| meaning | one-line description of what this value represents |
| entry transitions | which prior values/events may transition into this value |
| exit transitions | which values/events this value may transition to |
| terminal/no-exit marker | whether this value has no further exit transition (terminal), explicitly marked, not left implicit |
| invalid-transition list | transitions from this value that are explicitly disallowed (not merely unmentioned) |
| failure/recovery behavior | what happens when a transition fails from this value, and how the field recovers |

**Review rule:** a prose-only progression (a narrative description of what
"usually" happens) or an "other state" catch-all fails specification review.
Every value must have its own row; every row must have all six columns
filled in, not left blank on the assumption a reader will infer them.

## Technique selection from state shape

When `specification.md` declares a closed lifecycle table for a field,
`test_planning.md` must select the **state-transition / decision-table**
technique for that field's test cases — not an ad-hoc equivalence-partitioning
pass that happens to touch the field.

The selected technique's cases must cover:

- every value in the closed table (not a sampled subset);
- every relevant cross-entity axis pulled from the I-##/X-## interaction-map
  rows naming that field (see Dependency discovery, below, for how those axes
  are found);
- explicit invalid-transition cases (asserting the disallowed transition is
  rejected, not merely untested);
- explicit recovery cases (asserting the failure/recovery behavior column is
  exercised); and
- explicit state-addition regression cases (a test that would fail if a new
  value were added to the table without updating the consuming logic).

A test plan that covers only the "happy path" transitions and omits invalid,
recovery, or state-addition cases has not applied this technique — it has
applied ordinary functional testing to a state-shaped field.

## Dependency discovery by interaction and caller path

Before specifying a change to a behavior-bearing field or entity, discover
every dependency on it by searching these sources **in priority order**:

1. **I-##/X-## interaction-map rows** naming the field or entity — the
   authoritative cross-feature/cross-epic contract list.
2. **Production caller paths** — using this project's existing
   Caller-Path Contract concept (already used in test-plan.md's TC format:
   production entrypoint, lowest allowed mock seam, forbidden mocks,
   counter-factual) to trace who actually calls into code touching this
   field, not just who is documented as depending on it.
3. **Persistence readers** — grep for the field name in non-test `.go` files
   to find every persistence reader of the persisted value, including
   readers not named in any interaction-map row.
4. **Deduplication/short-circuit logic** — check whether any discovered
   caller applies deduplication or early-return logic keyed on this field,
   since that logic silently changes which downstream consumers actually see
   a given transition.
5. **Named downstream obligations** recorded in prior spec.md "Cross-feature
   interactions" sections — obligations a previous feature already declared
   against this field or entity.

For every candidate axis considered from these sources, record one line of
**inclusion/exclusion rationale** — why it was pulled in as a dependency, or
why it was excluded. A dependency search that stops at "no direct
entity-tracked foreign-key relationship" without walking the other four
sources has not discovered the full dependency set; a directly tracked
entity relationship is one axis among five, not the only one.

`specification.md`'s READ list references this section instead of the
unstated "grep for related services" heuristic it previously carried.

## Shipped consumer re-verification

When a feature adds or widens a state value that is read by an
**already-completed feature's implemented consumer** (discovered per
Dependency discovery, above), the specification must record, for that
consumer:

- the consumer's **caller path**;
- the consumer's **owning feature key**;
- the **affected AC IDs** on the consumer side; and
- a **regression-test pointer** — existing or newly assigned — that proves
  the consumer still behaves correctly against the widened state space.

If the current feature cannot safely amend the consumer directly, it must
create a linked task or tech-debt record naming the consumer feature — a
staged handoff, never presented as already-verified. A specification that
widens a state space consumed elsewhere and says nothing about the consumer
has not satisfied this section, even if its own local tests pass.

## I-04 propagation

Resolving a Question, tech-debt item, change card, or ADR that changes
already-implemented or already-specified behavior must produce one I-04
`ChangeImpactSet` (shape: architecture.md#i-04-changeimpactset-v1) naming:

- every invalidated spec, test-plan, task, interaction-map, or standards
  artifact; and
- every affected shipped consumer AC.

Each named artifact or AC must be either amended in the same change or
linked to explicit follow-up work — never a completion record that omits an affected artifact without a stated disposition. A resolution that lists
three invalidated artifacts and silently drops a fourth has not satisfied
this section; omission of one item is exactly the failure mode this section
exists to catch, and it is not cured by the other three being handled
correctly.

`prompts/tech_debt/resolved.md` and `skills/question-management/SKILL.md`
reference this section from their own resolution steps rather than
restating its procedure.

## Design divergence

Rework that departs from an accepted fix design must cite:

- the **original decision pointer**;
- the **new evidence** that justifies departing from it;
- the **affected consumers** of the original design; and
- the **resulting artifact/test amendments** the divergence requires.

Absence of a cited divergence means the recorded compatible design remains
controlling — silently doing something different from an accepted design,
with no citation, is not a valid alternative; it is an unexplained
divergence.

This generalizes `defect-class-sweep.md`'s **Backward-looking rework**
section from the point-instance defect-repair context to the broader
planning-decision context described above. This section references that
section directly rather than redefining divergence handling twice — see
`skills/quality/workflows/defect-class-sweep.md`'s "Backward-looking rework"
section for the full search-then-cite procedure (affected code/tests, decision
notes, tech-debt records, prior review-finding notes, spec/standards docs) and
apply it here unchanged.

## Self-verification (before returning)

- [ ] Every behavior-bearing field considered has a closed table with all six
      columns filled in — no prose-only progression, no "other state"
      catch-all.
- [ ] Every closed-table field's test cases use the state-transition/
      decision-table technique and cover every value, every relevant
      cross-entity axis, invalid transitions, recovery, and state-addition
      regression.
- [ ] Dependency discovery walked all five sources in order and recorded one
      inclusion/exclusion rationale line per candidate axis — not stopped at
      directly tracked entity relationships alone.
- [ ] Every shipped consumer affected by a widened state space has a
      recorded caller path, owning feature key, affected AC IDs, and a
      regression-test pointer, or a linked staged-handoff task/tech-debt
      record.
- [ ] Every I-04 `ChangeImpactSet` names every invalidated artifact and
      affected consumer AC with a stated disposition — no silent omission.
- [ ] Any rework diverging from a recorded design cites the original
      decision pointer, new evidence, affected consumers, and resulting
      amendments — absence of a citation means the recorded design still
      controls.
```

## TC-015 — Plan a multi-entity lifecycle

**Fixture:** A new feature's spec.md adds a `dedup_pending` value to the
(hypothetical) `SyncJob.status` field. `dedup_pending` is read by an
already-shipped "dedup filter" service that skips scheduling any job whose
`status` is `dedup_pending`, and its retry path can fail back to
`dedup_pending` from `queued`. The new spec is checked against the rendered
closed-table, dependency-discovery, and shipped-consumer sections above.

**Walkthrough:**

1. *Closed-table check.* `dedup_pending` determines dispatch routing (it
   suppresses scheduling), so "Closed lifecycle tables" requires a
   behavior-bearing row for it, not prose: value (`dedup_pending`), meaning
   ("job held because a duplicate was detected"), entry transitions (from
   `queued` on duplicate detection), exit transitions (to `queued` on
   manual retry), terminal/no-exit marker (explicitly `false` — it has an
   exit), invalid-transition list (e.g. `dedup_pending → completed` directly
   is disallowed), and failure/recovery behavior (a failed retry stays in
   `dedup_pending` and increments a retry counter). A spec that instead
   describes this value only as "the job waits until the duplicate clears"
   — a prose-only progression — fails the "Review rule" verbatim.
2. *Dependency discovery.* The spec must walk all five sources in priority
   order, not stop at the first hit: (1) any I-##/X-## row already naming
   `SyncJob.status` — none yet, since this is the field's first cross-feature
   appearance; (2) production caller paths — tracing who actually calls into
   code touching `SyncJob.status` surfaces the dedup filter as a caller; (3)
   persistence readers — grepping the field name in non-test `.go` files
   confirms the dedup filter is a persistence reader of the persisted value;
   (4) deduplication/short-circuit logic — this *is* the dedup filter's own
   early-return behavior, which "Dependency discovery" requires recording
   explicitly because it "silently changes which downstream consumers
   actually see a given transition"; (5) named downstream obligations from
   prior spec.md sections — checked and found empty here. Each of the five
   gets one inclusion/exclusion rationale line.
3. *Shipped-consumer re-verification.* Because the dedup filter is an
   already-completed feature's implemented consumer of the widened state
   space, the spec must record the filter's caller path, its owning feature
   key, the affected AC IDs on the filter's side, and a regression-test
   pointer proving the filter still behaves correctly against the new value
   — or, if the current feature cannot safely amend the filter directly, a
   linked staged-handoff task naming the filter's feature.

**Expected outcome:** the failure state (`dedup_pending`) and its recovery
transition (`dedup_pending → queued` on retry) appear in the closed table;
the consumer path, existing AC, cross-entity axis, and regression test are
all named before the plan can pass — matching test-plan.md's TC-015 Expected
outcome cell exactly.

**Edge case (direct-dependency-only discovery, REQ-F-003):** A fixture
variant where the spec discovers the dedup filter *only* because
`SyncJob` happens to have a direct entity-tracked foreign key to a table the
filter also reads — and does not walk caller paths, persistence-reader
grep, dedup/short-circuit logic, or prior spec obligations — must be flagged
as insufficient discovery. "Dependency discovery"'s own text names this
exact failure mode: "A dependency search that stops at 'no direct
entity-tracked foreign-key relationship' without walking the other four
sources has not discovered the full dependency set; a directly tracked
entity relationship is one axis among five, not the only one." The
"Self-verification" checklist reinforces this as a pre-return gate:
dependency discovery must have "walked all five sources in order... not
stopped at directly tracked entity relationships alone." A search that never
reaches sources 2–5 fails this checklist item even if it happens to land on
the correct consumer via source 3's territory by luck.

**Verdict: PASS.** The rendered content requires the closed-table row
(including the recovery transition), the five-source dependency walk with
per-axis rationale, and the shipped-consumer handoff fields before the plan
may pass; a discovery pass that stops at a direct entity relationship alone
is explicitly and structurally rejected by the same content, proving
REQ-F-003's "not limited to direct Shark dependencies" requirement is
enforced by the rendered guidance itself, not merely implied.

## TC-016 — Propagate a ratified decision

**Fixture:** A Question resolution changes an accepted design for converting
stored timestamps between UTC and local time. The change invalidates two
specs (spec A: the storage-layer spec that documents the old conversion
rule; spec B: a consumer feature's spec that assumes the old rule when
formatting output), one test plan (test plan C: the storage-layer's test
plan, whose conversion test cases assert the old rule), and one shipped
consumer AC (AC D: an already-shipped report-generation feature's AC that
displays converted timestamps).

**Walkthrough:** "I-04 propagation" requires the resolution to produce one
I-04 `ChangeImpactSet` naming every invalidated artifact (specs A and B,
test plan C) and every affected shipped-consumer AC (AC D). Architecture.md's
I-04 shape table defines `affected_artifacts` (path, kind, invalidated
text/contract, disposition, optional follow-up key) and `affected_consumers`
(entity key, caller path, AC IDs, regression-test pointer) as the fields that
carry this; `status: accounted` is only correct when "each affected artifact
is amended or has an existing linked follow-up key, each shipped consumer
has an assigned regression test, and no shared-name mismatch remains
unexplained." Applying the fixture: a compliant resolution amends spec A and
test plan C in the same change, links spec B to an explicit follow-up task
(a staged handoff, since the consumer feature's team owns that file), and
assigns AC D a regression-test pointer against the shipped report feature —
all four items named, each with a stated disposition (amended or
follow-up-linked). `tech_debt/resolved.md` and
`skills/question-management/SKILL.md` both reference this section from their
own resolution steps (confirmed in the checked-in files — see
`skills/question-management/SKILL.md`'s step referencing
`state-space-coverage.md#i-04-propagation` and `tech_debt/resolved.md`'s
conditional I-04 line), so a Question resolution following either path is
routed to this same rendered obligation.

**Edge case (silent omission of an affected spec, per test-plan.md's exact
cell):** A fixture variant where the resolution names spec A, test plan C,
and consumer AC D plus their dispositions, but says nothing at all about
spec B — no mention, no disposition, not even a deferred one — must be
caught as non-compliant. The rendered text is explicit and does not tolerate
a 3-of-4 pass: "A resolution that lists three invalidated artifacts and
silently drops a fourth has not satisfied this section; omission of one item
is exactly the failure mode this section exists to catch, and it is not
cured by the other three being handled correctly." Per the I-04 shape's own
definition, `status: accounted` requires "each affected artifact is amended
or has an existing linked follow-up key"; with spec B carrying neither, the
omission forces `status: incomplete`, not a passing record with an unlisted
gap — matching test-plan.md TC-016's edge-case wording ("a completion record
omitting any one artifact without a stated disposition is rejected by the
workflow's own text") verbatim.

**Verdict: PASS.** The rendered content requires every invalidated artifact
and every affected shipped-consumer AC to be named with a stated
disposition before `status: accounted` may be reported, and it names the
silent-omission failure mode explicitly rather than leaving it to
inference — a completion record dropping one of the four fixture items is
rejected by the workflow's own text, not merely discouraged.

## TC-I-04-CHANGE-IMPACT-CLOSURE cross-reference

Per test-plan.md's "Cross-feature contract tests (I-##)" section,
`TC-I-04-CHANGE-IMPACT-CLOSURE` (also named `TC-I-04-DECISION-PROPAGATION` in
`E34-interaction-map.md`) is the constituent pair TC-016 (above — the
workflow-content obligation) plus TC-007 (`shark impact record`'s runtime
persistence hook). TC-007 is proven against a real repository, not only
mocks, in `internal/cli/commands/impact_cmd_test.go` (T-E34-F07-005's
delivered test file — confirmed present in the tree at review time).
Together they prove both halves of the I-04 contract this feature owns; this
file supplies TC-016's half.

## TC-017 — Reject naming drift

**Fixture:** A task implementing part of this epic's own interaction-map
plumbing renames a shared interaction field from `class_key` to `defectKey`
in its local Go code, without updating the owning spec.md or
`E34-interaction-map.md`, which still document the field as `class_key`. The
task's own code compiles and its own unit tests pass under the new local
name.

**Walkthrough:** `task_review.md`'s "Shared naming integrity" checklist item
(added per REQ-F-005, referencing this workflow file) states: "Compare every
shared field/state/event/contract name the task touches against the owning
specification and interaction map verbatim; report unexplained drift as a
contract finding (blocking) even when the local name compiles/passes
tests." Applying the fixture: task review compares the task's `defectKey`
usage against the owning spec.md and `E34-interaction-map.md`, both of which
name the field `class_key` verbatim; the names do not match, and the task's
notes/spec citations contain no reference to an approved amendment renaming
the field. Per the checklist item's explicit clause, this drift is reported
as a **blocking contract finding**, and the fact that the task's own code
compiles and its own tests pass is stated as *irrelevant* to that
determination ("even when the local name compiles/passes tests") — task
review does not accept the local implementation name as authoritative over
the owning contract documents.

**Edge case (explained rename, REQ-F-005's exact wording):** A fixture
variant where the same rename is accompanied by a citation to an approved
spec.md amendment that itself updates `class_key` to `defectKey` (i.e., the
owning specification and interaction map have already been amended in the
same or a linked prior change, and the task cites that amendment) must
**not** be flagged. The checklist item's operative word is "unexplained" —
it requires reporting "unexplained drift," not every rename outright; once
the owning specification and interaction map are compared "verbatim" against
the task's usage post-amendment, the names match and no drift exists to
report. This mirrors the workflow's own "Design divergence" section
principle one level up (a cited, evidenced departure from a prior decision
is distinct from a silent one) applied here to naming rather than fix
design.

**Verdict: PASS.** An unexplained rename against the owning spec/interaction
map is rejected as a blocking contract finding regardless of local
compile/test status, per the checklist item's explicit clause; a rename
backed by a cited, approved amendment to the owning documents is correctly
excluded from that same clause because the drift is no longer "unexplained"
once the verbatim comparison is made against the amended documents.
