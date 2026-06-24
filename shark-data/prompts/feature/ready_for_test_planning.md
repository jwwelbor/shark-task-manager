{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Create test plan for feature {{.id}}: "{{.title}}".

Check for existing test-plan.md in feature directory. If test plan exists meeting exit gate criteria, advance immediately.

---

{{include: skills/quality/workflows/test-planning.md}}

READ:
(1) Feature spec.md for requirements and acceptance criteria
(2) Parent epic uat-plan.md for UAT scenarios this feature must satisfy
(3) Feature research report (if exists) for existing test infrastructure
(4) CLAUDE.md testing architecture rules (repo tests use real DB, everything else uses mocks)

PRODUCE test-plan.md:

(1) **AC Test Matrix**: For each acceptance criterion in spec.md:
    - Test case description
    - Input/setup
    - Expected outcome
    - Edge cases

(2) **Caller-Path Contract** (per test case, mandatory if a production caller exists above the entrypoint):
    - Production entrypoint (function + argument shape production callers actually use)
    - Lowest allowed mock seam
    - Forbidden mocks (seams that hid prior bugs — e.g. helper-test signatures production never passes)
    - Counter-factual (one sentence: what a buggy impl would do that this test would catch)
    See ~/.claude/skills/quality/workflows/test-planning.md Step 5.8 for guidance.

(3) **Integration Scenarios**: Cross-component interactions:
    - Which components interact
    - What to verify at boundaries
    - Reference to epic UAT scenarios this feature contributes to

(4) **Test Infrastructure**: What test utilities exist vs need creation:
    - Existing test patterns to follow (with file paths)
    - New test helpers needed (if any)

CRITICAL: Tests trace to FEATURE acceptance criteria (in spec.md), which trace to epic requirements. No orphaned tests. Tests drive the production caller signature, not a convenient helper signature — caller-path contracts close that gap at design time.

EXIT GATE:
- Every AC in spec.md has at least one test case
- Every test case has a caller-path contract (or a documented internal-only justification)
- Edge cases identified for each AC
- Integration scenarios cover cross-component boundaries
- Test patterns reference existing infrastructure
