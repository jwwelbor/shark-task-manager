Review test plan for feature {{.id}}: "{{.title}}".

Check for existing test plan review report at docs/review/{epic-folder}/{feature-folder}/test_plan_review/{{.id}}-test-plan-review.md.
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders.)
If report exists with PASS verdict, advance immediately. If FAIL, send back to ready_for_test_planning.

---

TEST PLAN REVIEW — GEMINI PERSPECTIVE

READ:
(1) Feature spec at {{.file_path}} — for the full requirements and AC list
(2) Test plan at the sibling test-plan.md (same directory as spec)

Run /gemini-exec analysis with the following prompt (substitute the actual resolved paths):

"You are reviewing a feature test plan for coverage gaps before task generation begins.
Your job is to ensure the test plan fully covers the spec so that no requirement slips
through to implementation untested.

READ the feature spec at <spec-path> and the test plan at <test-plan-path>.

CHECK:
- Requirement traceability: every REQ-F-xxx in the spec maps to at least one TC-ID in the test plan
- AC coverage: every acceptance criterion has a corresponding test case that directly validates it
- Negative and edge cases: unhappy paths, boundary values, and error conditions have test cases
- Test isolation: no test case depends on side-effects from another test case
- Caller-path contracts: integration points and API contracts are covered by wiring tests
- Observability: test cases specify what to assert, not just what to execute

LABEL each gap:
- CRITICAL: a requirement or AC has zero test coverage — blocks task generation
- HIGH: coverage exists but is insufficient (e.g., happy-path only, missing boundaries)
- LOW: recommendation or additional coverage worth considering

Report:
VERDICT: PASS or FAIL
Traceability matrix: REQ-F-xxx → TC-ID(s) (flag any REQ with no TC)
CRITICAL gaps (if any)
HIGH gaps (if any)
LOW suggestions (if any)
Overall assessment"

PRODUCE test plan review report at docs/review/{epic-folder}/{feature-folder}/test_plan_review/{{.id}}-test-plan-review.md:
- Gemini output verbatim
- Verdict: PASS or FAIL
- Traceability matrix and any CRITICAL/HIGH gaps for easy reference

DECISION:
- PASS (no CRITICAL gaps) → shark status advance {{.id}}
- FAIL (any CRITICAL gap) → shark status set {{.id}} ready_for_test_planning --reason "<list uncovered requirements>"
