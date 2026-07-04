Review features for epic {{.id}}: "{{.title}}".

Check for existing feature review report in feature_reviews/ directory. If report exists with PASS verdict, advance immediately. If FAIL, send back to decomposition.

---

FEATURE DECOMPOSITION REVIEW

This is a quality gate comparing the generated features against the epic requirements. The goal is to catch gaps, overlaps, or misalignments BEFORE task generation begins.

READ:
(1) Epic PRD at {{.file_path}} for goals, scope, requirements, and success criteria
(2) All feature files: {{template "list_json" .}}, then read each feature's file_path
(3) Architecture doc (if exists) in the epic directory for component boundaries
(4) Research report (if exists) in the epic directory for implementation context
(5) {{.id}}-interaction-map.md if present
(6) {{.id}}-cross-epic-map.md and docs/product/cross-epic-integration-map.md if present

VERIFY:

## Requirements Coverage
- [ ] Every epic requirement is addressed by at least one feature
- [ ] No epic requirements are missing or only partially covered
- [ ] Success criteria from the epic PRD map to specific features

## Feature Quality
- [ ] Each feature represents a cohesive, independently deliverable capability
- [ ] Feature descriptions are specific enough for assessment and specification
- [ ] No overlapping scope between features (clear boundaries)
- [ ] Feature titles accurately reflect their content

## Ordering & Dependencies
- [ ] Execution order reflects actual dependencies
- [ ] No circular dependency chains
- [ ] Foundation features (shared infrastructure, data models) come first
- [ ] Features can be worked on in the specified order

## Scope Alignment
- [ ] Features stay within epic scope (no scope creep)
- [ ] No unnecessary features that don't map to epic requirements
- [ ] Feature granularity is appropriate (not too large, not too small)

### Interaction-map closure (multi-feature epics only)

For multi-feature epics, read {{.id}}-interaction-map.md and verify:

- Every I-## row has a producer feature that exists
- Every I-## row has at least one consumer feature that exists
- The producer feature's description or spec names the I-## under "Produces"
- Each consumer feature's description or spec names the I-## under "Consumes"
- Producer and consumer cite the SAME shape source
- No orphans: an I-## with no producer, no consumer, or mismatched shape source
  is FAIL

### Cross-epic integration closure

If {{.id}}-cross-epic-map.md or docs/product/cross-epic-integration-map.md
contains X-## rows relevant to this epic, verify:

- Every relevant X-## row has producer epic and consumer epic(s) named
- Producer and consumer feature ownership is named before this review passes
- Each referenced producer or consumer feature exists in the feature list
- Contract / shape source matches between the per-epic map, the global product
  map, and any feature description that names the X-##
- UX / CX handoff notes are populated or marked not applicable with a reason
- Test coverage pointer exists or is explicitly deferred in
  docs/product/progress.md with a decision-log entry
- No orphans: an X-## with missing producer feature, missing consumer feature,
  mismatched shape source, or missing coverage disposition is FAIL

PRODUCE feature review report at feature_reviews/{{.id}}-feature-review.md:
- Verdict: PASS or FAIL
- Requirements coverage matrix (epic requirement -> feature mapping)
- Interaction-map closure table (multi-feature epics): one row per I-## with
  producer, consumer(s), shape source, and closure status
- Cross-epic integration closure table: one row per relevant X-## with producer
  epic/feature, consumer epic(s)/feature(s), shape source, coverage pointer or
  deferral, and closure status
- Gaps identified (if any)
- Overlaps identified (if any)
- Ordering issues (if any)
- Recommendations

Print interaction-map closure table and cross-epic integration closure table in
the report before the final verdict.

DECISION:
- ALL PASS -> shark status advance {{.id}}
- ANY FAIL -> shark status set {{.id}} decomposition --reason "<specific gaps or issues to fix>"
