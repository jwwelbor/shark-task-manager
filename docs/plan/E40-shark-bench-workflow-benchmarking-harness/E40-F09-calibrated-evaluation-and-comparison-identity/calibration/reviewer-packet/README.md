# E40 human review packet

## What the rubric is for

The planning rubric is meant to judge the **generated D01–D05 planning
documents as one set**. The feature request/use case and its supplied reference
facts are context: use them to decide whether the documents address the right
problem and support their claims. Do not score the request itself, or the
implementation and its runtime behavior.

The five scores are not one-to-one ratings of five individual files. For each
dimension, read across the D01–D05 set and consider the evidence most relevant
to that dimension:

| Dimension | Assess across the document set |
|---|---|
| P | Whether the problem, intended users, context, and scope are clear. |
| M | Whether success can be observed and the stated constraints fit the scope. |
| R | Whether research informs the decision and factual claims have support. |
| F | Whether feasible options, costs, risks, and trade-offs are reasoned through. |
| S | Whether the documents agree on users, scope, evidence, criteria, and decisions, and surface gaps or contradictions. |

See [the rubric](../v1/rubric.md) for the full definitions and score anchors.

## Review the current baseline documents

For the recurring-task feature in the current Codex baseline, review the actual
retained D01–D05 outputs below. This is one document set, so use one
[baseline score sheet](scores/BASELINE-RECURRING-TASKS.md) to record P, M, R,
F, and S for the set as a whole.

This sheet is for a human review of the baseline output. Keep its ratings
separate from judge-calibration labels: this recurring-task scenario is held
out for evaluation, so do not enter these scores in the v1 calibration labels
or use them to tune the judge. The CAL sheets below are the separate human
calibration exercise.

Read the [feature request](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/replay-input/input/prompt.md)
and [captured reference answers](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/replay-input/evaluator/replay/reference-bundle.json)
as context. Then review all five generated documents together:

| Generated document | Open it |
|---|---|
| D01 — Vision Statement | [D01-vision-statement.md](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D01-vision-statement.md) |
| D02 — Success Criteria | [D02-success-criteria.md](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D02-success-criteria.md) |
| D03 — Market Research | [D03-market-research.md](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D03-market-research.md) |
| D04 — Feasibility Report | [D04-feasibility-report.md](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D04-feasibility-report.md) |
| D05 — Stakeholder Insights | [D05-stakeholder-insights.md](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D05-stakeholder-insights.md) |

The [retained result record](../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/result.json)
binds these files to scenario `py-feature-recurring-tasks` and records their
digests. The request and reference answers help you assess relevance and
support; they are not what you score. Do not score implementation or runtime
outcomes with this planning rubric.

## What these calibration examples contain

The six sections in [examples.md](../v1/examples.md) are fictional calibration
examples. In each, `REF-01` is the example request and supplied facts;
`D01`–`D05` are the planning evidence to score against that context. They are
short, condensed examples, **not full documents produced by Shark and not the
baseline documents from the bench run**. A score on these sheets therefore
rates only these condensed examples. It cannot establish that the judge is
calibrated on real Shark-generated baseline documents. If your task is to
provide the independent human labels for judge calibration, use these CAL
sheets; if your task is to review the baseline itself, use the separate sheet
linked above.

That gap matters for this baseline review: the generated Shark documents are
the artifacts that should be judged with this rubric. The calibration packet
now links to the current recurring-task baseline outputs above. Do not mistake
the fictional example links below for those baseline documents.

## If you are reviewing the calibration examples

You can use these sheets to practice applying the rubric to a small, fictional
example set. For each example, read its `REF-01` and all five `D01`–`D05`
entries, then score each rubric dimension against the set as a whole. Do not
grade only the entry whose name seems closest to the dimension. Cite supplied
IDs in the rationale; do not fill evidence gaps with outside knowledge.

Enter one whole-number grade from **0 to 4** for each dimension:

| Grade | Use it when the evidence is… |
|---:|---|
| 0 | Absent, irrelevant, or contradicted. |
| 1 | Mentioned, but with major gaps or unsupported claims. |
| 2 | Usable as a direction, with material gaps or uncertainty. |
| 3 | Mostly clear and supported, with bounded gaps. |
| 4 | Clear, specific, supported by the supplied evidence, and consistent. |

The score sheets use the 0–4 form of the rubric's 0–1 anchors. The five
dimensions have equal weight. The overall label is `(P + M + R + F + S) / 20`;
do not assign a separate overall grade. Leave a grade blank if you need
clarification; blank means unscored, not zero.

## Materials

1. [Rubric and score anchors](../v1/rubric.md).
2. [Six fictional calibration examples](../v1/examples.md).
3. [Original calibration instructions](../v1/README.md), including how
   labels are used and kept separate from held-out evaluation.
4. The [original blank worksheet](../v1/human-scores.md) collects all six
   ratings in one place. The individual sheets below are an alternative.
5. The [manifest](../v1/manifest.json) records frozen source identities; it is
   not needed to choose grades.

The broader [E40 prompt and model performance rubric](../../../prompt-model-performance-rubric.md)
covers benchmark comparisons such as quality, time, and cost. It is useful
context but is not used to grade these six planning examples. The [judge
prompt](../v1/judge-prompt.txt), [machine-readable examples](../v1/examples.json),
and [blank machine-label template](../v1/human-labels.template.json) are used
later to run and record calibration; they are not needed to choose grades.

## Calibration example score sheets

These six sheets are for the separate, fictional calibration examples only.
They are not six additional baseline outputs and are not required to score the
recurring-task baseline documents above.

| Example | Topic | Review this fictional material | Record grades here |
|---|---|---|---|
| CAL-001 | Export a personal reading list | [REF-01 and D01–D05](../v1/examples.md#cal-001-export-a-personal-reading-list) | [CAL-001 score sheet](scores/CAL-001.md) |
| CAL-002 | Tag several inventory items | [REF-01 and D01–D05](../v1/examples.md#cal-002-tag-several-inventory-items) | [CAL-002 score sheet](scores/CAL-002.md) |
| CAL-003 | Find notes by title | [REF-01 and D01–D05](../v1/examples.md#cal-003-find-notes-by-title) | [CAL-003 score sheet](scores/CAL-003.md) |
| CAL-004 | Export a pantry list | [REF-01 and D01–D05](../v1/examples.md#cal-004-export-a-pantry-list) | [CAL-004 score sheet](scores/CAL-004.md) |
| CAL-005 | Tag selected research notes | [REF-01 and D01–D05](../v1/examples.md#cal-005-tag-selected-research-notes) | [CAL-005 score sheet](scores/CAL-005.md) |
| CAL-006 | Find saved recipes by title | [REF-01 and D01–D05](../v1/examples.md#cal-006-find-saved-recipes-by-title) | [CAL-006 score sheet](scores/CAL-006.md) |

When finished, return the completed sheets or send the grades and reasons in
chat. Include your name or stable reviewer ID and review date. Ratings are the
human labels and should be transcribed as given, not adjusted to match a
judge's scores.
