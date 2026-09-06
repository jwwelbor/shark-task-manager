{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Decompose epic {{.id}}: "{{.title}}" into features.

Check for existing features: {{template "list" .}}. If features exist covering all epic requirements with proper ordering, advance immediately.

---

{{template "_product_critical_path_guard" .}}

{{include: skills/specification-writing/workflows/write-feature-prd.md}}

READ:
(1) Epic PRD for scope and requirements
(2) Architecture doc for component boundaries
(3) Research report for implementation approach
(4) {{.id}}-interaction-map.md if present. For a multi-feature epic, every I-##
    must have a producer feature and at least one consumer feature in the
    feature list you create.
    Rule anchor: every I-## must have a producer feature and at least one consumer feature.
(5) {{.id}}-cross-epic-map.md and docs/product/cross-epic-integration-map.md if
    present. Every relevant X-## must be assigned to producer and consumer
    features before decomposition exits.
    Rule anchor: every X-## must name producer and consumer feature ownership.

PRODUCE features via shark CLI:

For each feature:
(1) {{template "create_feature" .}} "<title>" --order=N --size=<1|2|3|5|8|13>
(2) Write a 1-paragraph description to the feature file (NOT a full PRD — that happens at feature level)

{{template "_sizing_feature" .}}

Feature decomposition rules:
- Each feature maps to a cohesive capability boundary
- Features are independently deliverable where possible
- Before accepting a boundary, require a real trigger, observable result,
  production path, complete UAT scenario, current prerequisites, and outputs
  for later consumers. If an acceptance depends on a later feature, reassign it
  to the named activation owner with a complete declared staged disposition or
  redesign the slices; "a future feature will wire it" alone is not acceptable.
- Execution order reflects dependencies
- Description includes: what it does, why it's needed, key integration points
- If an interaction map exists, each feature description names the I-## IDs the
  feature will produce or consume, using explicit "Produces: I-##" and
  "Consumes: I-##" phrasing so reviewers can trace the map at a glance.
- Every I-## from the interaction map has a producer feature AND at least one
  consumer feature in the resulting list; no orphan wires.
- `live` is the default gate mode. A `contract-only` I-## must already name
  counterpart identities, a current status read live from Shark, shared-contract evidence, activation
  owner, closure key, and review basis; reject incomplete declarations and
  report reverse build-order consumption as a decomposition warning.
- If a cross-epic map exists, each feature description names the X-## IDs the
  feature will produce, consume, or validate, using explicit "Produces: X-##",
  "Consumes: X-##", or "Validates: X-##" phrasing.
- Update {{.id}}-cross-epic-map.md and docs/product/cross-epic-integration-map.md
  with the assigned owning feature(s) for this epic's role in each relevant
  X-## row. Keep X-## separate from I-##; do not renumber either ID space.
- Every X-## relevant to this epic has producer and consumer feature ownership
  named before exit, or an explicit deferred decision in docs/product/progress.md.

CRITICAL: Feature descriptions are THIN — one paragraph. The feature workflow will handle PRD, architecture, and test planning at feature level. Do NOT front-load detail here.

ANTI-FILLER RULE: if decomposition would yield zero features, or only a single filler feature invented to satisfy the "at least one feature" gate, do NOT create it. This epic was misclassified — determine the target type using the same signals as `epic/assessment.md`'s reclassification table (change-card/tech-debt/task/feature/idea as appropriate), then perform the reclassify-and-cancel procedure in `skills/assessment/workflows/reclassify-misfiled-entity.md`. If the reclassification target is unclear, stop, explain why in your final response, and end with `RECOMMENDED OUTCOME: fail` so the parent loop can send the epic back to design for reassessment.

EXIT GATE:
- All epic requirements covered by at least one feature
- Features have execution order and dependency annotations
- No overlapping scope between features
- Descriptions are specific enough for assessment
- Every feature carries a non-empty --size; features sized 13/XXL are split before exit
- Every relevant X-## in {{.id}}-cross-epic-map.md has producer and consumer
  feature ownership named, and the global product map is updated with the same
  owner information or a progress decision-log deferral
- Multi-feature epic: every I-## in the interaction map has a producer feature
  AND at least one consumer feature; no orphan wires; no feature boundary has
  an acceptance dependency on a later feature without reassignment or redesign
