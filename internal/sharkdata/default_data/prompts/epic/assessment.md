{{template "_resume_preamble" .}}Assess epic {{.id}}: "{{.title}}".

Check epic metadata: {{template "get_json" .}}. If already reclassified (status is cancelled or on_hold with a reclassification note), route immediately.

---

EPIC INTAKE TRIAGE — confirm this is genuinely epic-scale work before the heavyweight refinement/research/design/decomposition pipeline runs. A misfiled epic (e.g. docs-only work with no runtime code) can never legitimately decompose into features; catching it here avoids a downstream decomposition step inventing filler features just to satisfy the "at least one feature" gate.

READ:
(1) Epic description at {{.file_path}}
(2) Codebase via quick grep for related files, runtime code, and existing entities that might already cover this work

## Step 1: Classification Validation

Is this genuinely an EPIC?

EPIC = 2+ meaningful features, needs architecture/design decisions, multi-capability, code- or product-bearing.

DISQUALIFIERS (any one is enough to reclassify):
- Docs-only work with no runtime code or product surface (e.g. "write architecture decisions", "canonicalize vocabulary")
- Process, governance, or config change not tied to user/product value
- Single atomic change
- Single multi-step capability that doesn't need architecture decisions or a second feature

If none of the disqualifiers apply, skip to Step 3.

## Step 2: Reclassify If Misfiled

Classify using the same signals as `/triage`, scoped to the EPIC-vs-lighter-entity decision:

| Signal | Classification |
|--------|----------------|
| Process / infra / docs-governance change (or "chore") not tied to product value | **Change Card** |
| Documentation or architecture debt (quality/maintainability, not new capability) | **Tech Debt** |
| Single atomic change | **Task** (under an existing feature) |
| Single multi-step capability, no second feature or architecture decision needed | **Feature** (under a related epic) |
| Speculative / future concept not yet committed | **Idea** |

An epic has no natural existing container, so **Feature** and **Task** targets need a container chosen first:

- **Feature**: choose a clearly related, non-cancelled parent epic for this capability: `shark list`. If no safe parent exists, add a note explaining why, then `shark status set {{.id}} on_hold --reason "Reclassified as feature; needs human parent-epic selection"` and STOP — do NOT create the feature under the epic being cancelled.
- **Task**: choose a parent epic, then find or create an enhancement feature under that parent: `shark list <parent-epic>` | grep -i enhance. If no safe parent exists, add a note explaining why, then `shark status set {{.id}} on_hold --reason "Reclassified as task; needs human parent-epic selection"` and STOP — do NOT create the task under the epic being cancelled.

Once the target type (and container, for Feature/Task) is determined:

{{include: skills/assessment/workflows/reclassify-misfiled-entity.md}}

## Step 3: Route

Genuine epic -> `shark status advance {{.id}} --outcome pass`
