{{template "_resume_preamble" .}}Assess feature {{.id}}: "{{.title}}".

Check feature metadata: {{template "get_json" .}}. If complexity_tier already assigned, route immediately.

---

COMBINED SCOPE VALIDATION + COMPLEXITY TRIAGE

{{include: skills/assessment/workflows/complexity-triage.md}}

(Use complexity_triage mode.)

READ:
(1) Feature description at {{.file_path}}
(2) Parent epic PRD for context ({{template "get_json_epic" .}} for path)
(3) Codebase via quick grep for related files and patterns

## Step 1: Scope Validation

Is this properly scoped as a FEATURE or is it actually a TASK?

FEATURE = multi-capability (3+ changes), requires design decisions, 4+ files, cross-cutting concerns.
TASK = single atomic change, applies existing patterns, 1-3 files.

IF MISCLASSIFIED AS FEATURE (actually a task):

This feature already has a natural container — find or create an enhancement feature under its own parent epic: {{template "list_epic" .}} | grep -i enhance. Target type is Task.

{{include: skills/assessment/workflows/reclassify-misfiled-entity.md}}

STOP once reclassified.

## Step 2: Complexity Triage

SCORE using 9 dimensions (max 27):
- Technical (6): File Impact, Pattern Novelty, Data Model, API Surface, Cross-Feature Deps, UI Complexity
- Execution (3): Task Estimation, Regression Risk, Execution Effort

TIER: 0-6=SIMPLE, 7-15=STANDARD, 16+=COMPLEX

STORE: {{template "create_note" .}} --content="COMPLEXITY: {tier} (score: {score}/27)" --type=decision

## Step 3: Route

Release the tier as a semantic outcome — never name a target status:

- SIMPLE -> shark status advance {{.id}} --outcome simple
- STANDARD -> shark status advance {{.id}} --outcome standard
- COMPLEX -> shark status advance {{.id}} --outcome pass
