# Planning and decomposition rubric v1

Assess the supplied D01-D05 planning evidence as one set. Do not use implementation outcomes, oracle results, terminal status, or later review outcomes to raise a planning score. Apply this rubric unchanged to human calibration examples and the held-out evaluation set.

Score each of these five dimensions with equal weight (0.20):

1. Problem and target-user clarity: identify the user, concrete problem, relevant context, scope, and exclusions. Higher scores require a clear connection between the problem and the intended users.
2. Measurable success criteria and constraints: define observable outcomes, measurement conditions, and operational or product limits. Higher scores require criteria that can guide acceptance and constraints consistent with scope.
3. Relevance and support of research claims: connect research to the decision, distinguish evidence from assumptions, and retain bounded source identities for factual claims. Unsupported market numbers or competitor claims do not count as established evidence; clear uncertainty is preferable to unsupported certainty.
4. Feasibility and trade-off reasoning: compare plausible approaches against the stated constraints, explain costs and risks, and justify a feasible choice. Listing options alone is insufficient.
5. Synthesis and scope consistency: connect the final synthesis and proposed work to the earlier problem, users, criteria, research, and constraints. Identify contradictions, unresolved decisions, and gaps in decomposition when decomposition is present. Do not assume tasks or decisions that the supplied evidence does not contain.

Use these common anchors for each dimension:

- 0.00: absent, irrelevant, or contradicted by the supplied evidence.
- 0.25: mentioned, with substantial gaps or unsupported assertions.
- 0.50: usable direction, with material uncertainty, weak support, or incomplete reasoning.
- 0.75: clear and mostly supported, with bounded gaps stated explicitly.
- 1.00: clear, specific, supported within the supplied evidence, and consistent across the set.

The overall score is the arithmetic mean of the five dimension scores, in the range 0 through 1. Cite supplied entry IDs in each dimension rationale. Explain how missing evidence or contradictions affected the score. Do not fill missing evidence with outside knowledge. A validation pass means the captured judge evidence satisfies the contract; it does not imply a high planning score.

Human scorers must review separate calibration examples using this exact rubric, record the five scores and their overall mean, and retain their identity and review date. Calibration IDs must be unique and disjoint from evaluation IDs. The judge must not supply its own human labels.
