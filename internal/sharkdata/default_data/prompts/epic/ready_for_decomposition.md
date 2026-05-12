{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Decompose epic {{.id}}: "{{.title}}" into features.

Check for existing features: {{template "list" .}}. If features exist covering all epic requirements with proper ordering, advance immediately.

---

{{include: skills/specification-writing/workflows/write-feature-prd.md}}

READ:
(1) Epic PRD for scope and requirements
(2) Architecture doc for component boundaries
(3) Research report for implementation approach

PRODUCE features via shark CLI:

For each feature:
(1) {{template "create_feature" .}} "<title>" --execution-order=N --size=<1|2|3|5|8|13>
(2) Write a 1-paragraph description to the feature file (NOT a full PRD — that happens at feature level)

{{template "_sizing_feature" .}}

Feature decomposition rules:
- Each feature maps to a cohesive capability boundary
- Features are independently deliverable where possible
- Execution order reflects dependencies
- Description includes: what it does, why it's needed, key integration points

CRITICAL: Feature descriptions are THIN — one paragraph. The feature workflow will handle PRD, architecture, and test planning at feature level. Do NOT front-load detail here.

EXIT GATE:
- All epic requirements covered by at least one feature
- Features have execution order and dependency annotations
- No overlapping scope between features
- Descriptions are specific enough for assessment
- Every feature carries a non-empty --size; features sized 13/XXL are split before exit
