# Baseline score sheet: recurring tasks

This sheet scores the **generated D01–D05 documents together**. The request
and reference answers below are evidence for your judgment; do not score the
use case itself, the implementation, or runtime results.

These are human baseline-review ratings, not judge-calibration labels. Keep
them separate from the [v1 human-label template](../../v1/human-labels.template.json):
this recurring-task scenario is held out for evaluation and must not be used
to calibrate the judge.

Scenario: `py-feature-recurring-tasks`  
Retained replay attempt: `fe3676df462e312c`

## Read these materials

Start with the [feature request](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/replay-input/input/prompt.md)
and [captured reference answers](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/replay-input/evaluator/replay/reference-bundle.json).
Then read the full generated set:

1. [D01 — Vision Statement](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D01-vision-statement.md)
2. [D02 — Success Criteria](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D02-success-criteria.md)
3. [D03 — Market Research](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D03-market-research.md)
4. [D04 — Feasibility Report](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D04-feasibility-report.md)
5. [D05 — Stakeholder Insights](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/artifact-root/docs/product/D05-stakeholder-insights.md)

The [retained result record](../../../../../../../../shark-task-manager-operator-work/e40-codex-baseline-20261027b/replay/py-feature-recurring-tasks/attempt-fe3676df462e312c/result.json)
records the artifact digests for this set.

Reviewer name or stable ID: ____________________  Review date: ____________________

## Score the full set

For every row, consider evidence across D01–D05. The “look first” column points
to likely evidence, but no dimension belongs to only one document. Use the
[full planning rubric](../../v1/rubric.md) if you need its detailed definitions.

| Dimension | Assess across the full set | Grade (0–4) | Brief reason with document or reference evidence |
|---|---|---:|---|
| P — Problem and target users | Are the users, concrete problem, context, scope, and exclusions clear and consistent? Look first at D01, then check the rest. | | |
| M — Success criteria and constraints | Are outcomes observable, and are measurement conditions and constraints clear and consistent? Look first at D02, then check the rest. | | |
| R — Research relevance and support | Does research inform decisions, distinguish evidence from assumptions, and identify support for factual claims? Look first at D03 and the supplied research answers; check claims elsewhere too. | | |
| F — Feasibility and trade-offs | Are plausible approaches compared against constraints, with costs, risks, and a reasoned choice? Look first at D04, then check its fit with the rest. | | |
| S — Synthesis and scope consistency | Do the documents agree on users, scope, evidence, criteria, and decisions? Are contradictions, unresolved choices, or decomposition gaps surfaced? Read across all five, including D05. | | |

Use these common anchors for each dimension:

| Grade | Use it when the evidence is… |
|---:|---|
| 0 | Absent, irrelevant, or contradicted. |
| 1 | Mentioned, but with major gaps or unsupported claims. |
| 2 | Usable as a direction, with material gaps or uncertainty. |
| 3 | Mostly clear and supported, with bounded gaps. |
| 4 | Clear, specific, supported by the supplied evidence, and consistent. |

The five dimensions have equal weight. If all five grades are present, calculate
the overall label as `(P + M + R + F + S) / 20`; do not enter a separate
overall grade. Leave any uncertain grade blank and explain what needs
clarification. A blank is unscored, not zero.

[Back to the reviewer packet](../README.md)
