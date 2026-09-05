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
