Review spec for feature {{.id}}: "{{.title}}".

Check for existing spec review report at docs/review/{epic-folder}/{feature-folder}/spec_review/{{.id}}-spec-review.md.
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders.)
If report exists with PASS verdict, advance immediately. If FAIL, send back to ready_for_specification.

---

SPEC REVIEW — CODEX PERSPECTIVE

READ:
(1) Feature spec at {{.file_path}}
(2) Parent epic PRD for scope context

Run /codex-exec read-only with the following prompt (substitute the actual resolved spec path):

"You are reviewing a feature specification before test planning begins. Your job is to catch gaps,
contradictions, and untestable requirements that would cause rework downstream.

READ the spec at <spec-path>.

CHECK:
- Requirements completeness: every user-facing behavior has a REQ-F-xxx ID; no behavior is implied but unspecified
- Acceptance criteria: each AC is testable, unambiguous, and states an observable outcome
- Scope boundaries: no silent scope creep beyond what the epic authorizes; no missing capability needed to satisfy the ACs
- Internal contradictions: no two requirements conflict or produce ambiguous behavior
- Architecture coherence: the proposed design can satisfy all stated requirements
- Error handling and edge cases: unhappy paths are addressed, not left to implementer discretion

LABEL each issue:
- CRITICAL: spec cannot be acted on without resolution — blocks test planning
- HIGH: significant implementation risk or ambiguity
- LOW: recommendation or nit

Report:
VERDICT: PASS or FAIL
CRITICAL issues (if any)
HIGH issues (if any)
LOW issues (if any)
Overall assessment"

PRODUCE spec review report at docs/review/{epic-folder}/{feature-folder}/spec_review/{{.id}}-spec-review.md:
- Codex output verbatim
- Verdict: PASS or FAIL
- CRITICAL and HIGH items listed for easy reference

DECISION:
- PASS (no CRITICAL issues) → shark status advance {{.id}}
- FAIL (any CRITICAL issue) → shark status set {{.id}} ready_for_specification --reason "<list critical issues>"
