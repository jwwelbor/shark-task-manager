Feature code review passed for {{.id}} ("{{.title}}"). Launch QA agent for feature-level testing.

{{include: skills/quality/SKILL.md}}

READ:
(1) Feature spec at {{.file_path}} for acceptance criteria and scope
(2) Feature test-plan.md for all test cases and expected coverage
(3) Code review report from docs/review/{epic-folder}/{feature-folder}/code_review/*-{{.id}}-review.md
  (derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders) for scope of changes
(4) All task specs: `{{template "list_json" .}}` → collect every task's Scope file list
(5) `git diff $(git merge-base HEAD main)..HEAD` to derive all touched paths
(6) Parent interaction map and spec.md "Cross-feature interactions" section if present

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
   - Cross-feature I-## contract → run the shared contract test named in the
     feature spec and test-plan pointer
   - Pure intra-service or doc-only changes → skip

4. Sanity spot-check (only if changes are broad across a shared module):
   - `cd backend && uv run pytest tests/unit -x --tb=short`

DO NOT RUN: make test / full integration suite / codex red-team (UAT owns red-team)

VALIDATE:
- Quality gates pass (fmt, lint)
- All targeted tests pass — zero failures introduced by this feature
- Every AC from the feature test-plan is exercised by a named test
- No regressions in targeted scope
- Pre-existing failures in scope are explicitly identified (not silently ignored)
- Wiring coverage matrix includes one row per CONTRACT-### and I-## with
  producer/consumer, contract test, test-exists, and test-passes columns
- Any CONTRACT-### or I-## row with a missing contract test, a test that does
  not assert the documented shape, or a failing test is an automatic FAIL

PRODUCE QA report to docs/review/{epic-folder}/{feature-folder}/qa/<timestamp>-{{.id}}-qa.md:
- Verdict: PASS or FAIL
- Test scope rationale: touched files from git diff and why each test path was chosen
- Test results summary (counts, durations)
- AC verification: name the test that proves each AC
- Wiring coverage matrix: CONTRACT-### and I-## rows with producer/consumer,
  contract test, test-exists, test-passes columns
- Edge cases tested
- Any pre-existing failures encountered

ON PASS → {{template "advance" .}}
ON FAIL:
- For each failing task, kick back: `shark status set <task-id> development --reason "<specific failures>"`
- For a missing or broken I-## contract test: reopen the producer task in this
  feature and add a blocker note to each consuming feature:
  `shark feature note add <consumer_feature_id> --type blocker "Cross-feature I-## contract test failing at producer feature {{.id}}"`
- Set feature back: `{{template "status_set" .}} active --reason "QA failed — see report, tasks kicked back"`
