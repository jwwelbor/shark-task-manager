{{template "_resume_preamble" .}}RESUME feature-level QA for {{.id}}: "{{.title}}".

Check for an existing QA report at docs/review/{epic-folder}/{feature-folder}/qa/*-{{.id}}-qa.md.
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders.)
- If QA report exists with PASS verdict → {{template "advance" .}}
- If QA report exists with FAIL verdict → verify whether kicked-back tasks have since been fixed; if so, re-run QA; otherwise report to user and STOP
- If NO report exists → perform the full feature QA below. Do NOT advance without a written report.

---

Feature code review passed for {{.id}} ("{{.title}}"). Perform feature-level QA testing.

Load skill: `shark-data/skills/quality/SKILL.md`

READ:
(1) Feature spec at {{.file_path}} for acceptance criteria and scope
(2) Feature test-plan.md for all test cases and expected coverage
(3) Code review report from docs/review/{epic-folder}/{feature-folder}/code_review/*-{{.id}}-review.md for scope of changes
(4) All task specs: `{{template "list_json" .}}` → collect every task's Scope file list
(5) `git diff $(git merge-base HEAD main)..HEAD` to derive all touched paths

SCOPED TEST RUN (single pass across the full feature — do NOT run per-task):

1. Quality gates (always run, fast):
   - `cd backend && make fmt && make lint`

2. Targeted unit tests:
   - Union all changed source files from git diff AND from all task Scope sections
   - Map each to its test counterpart(s) under backend/tests/unit/
     - backend/app/services/X.py → backend/tests/unit/services/test_X.py
     - backend/app/api/X.py → backend/tests/unit/api/test_X.py
     - backend/app/db/models/X.py → backend/tests/unit/db/
   - Add any tests explicitly named in the feature test-plan ACs
   - Run: `cd backend && uv run pytest <all-paths> -x --tb=short`

3. Targeted integration tests (only if changeset crosses an integration seam):
   - DB migration / schema change → backend/tests/integration/db/ + tests using the changed table
   - API endpoint change → backend/tests/integration/api/ for that route
   - Service contract change → tests for the consumer
   - Pure intra-service or doc-only changes → skip

4. Sanity spot-check (only if changes are broad across a shared module):
   - `cd backend && uv run pytest tests/unit -x --tb=short`

DO NOT RUN: make test / full integration suite / codex red-team (UAT owns red-team)

VALIDATE:
- Quality gates pass (fmt, lint)
- All targeted tests pass — zero failures introduced by this feature
- Every AC from the feature test-plan is exercised by a named test
- No regressions in targeted scope

PRODUCE QA report to docs/review/{epic-folder}/{feature-folder}/qa/<timestamp>-{{.id}}-qa.md:
(Derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders.)
- Verdict: PASS or FAIL
- Test scope rationale
- Test results summary
- AC verification: name the test that proves each AC
- Any pre-existing failures encountered

ON PASS → {{template "advance" .}}
ON FAIL:
- For each failing task, kick back: `shark status set <task-id> ready_for_development --reason "<specific failures>"`
- Set feature back: `{{template "status_set" .}} active --reason "QA failed — see report, tasks kicked back"`
