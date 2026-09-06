{{template "_resume_preamble" .}}Assess feature {{.id}}: "{{.title}}".

Check feature metadata: {{template "get_json" .}}. If complexity_tier already assigned, route to research; every tier requires validated research.

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

Canonical per-tier artifact and gate matrix: `skills/quality/context/tier-matrix.md` — do not restate it here.

STORE: include this exact line in your final response so the parent loop can persist it as the decision note:
`COMPLEXITY NOTE: COMPLEXITY: {tier} (score: {score}/27)`

## Step 3: Route

Release the tier as a semantic outcome — never name a target status:

- All valid tiers -> end with `RECOMMENDED OUTCOME: pass` to enter research.
- Research selects the final SIMPLE/STANDARD/COMPLEX outcome after the report is complete.
- Do NOT run Shark status commands yourself; the parent loop will apply the outcome.
