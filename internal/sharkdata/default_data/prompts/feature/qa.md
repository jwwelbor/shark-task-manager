Verification gate (craft review) passed for COMPLEX feature {{.id}} ("{{.title}}"). Launch QA agent for the deep second verification gate. (SIMPLE/STANDARD features skip this gate — their review, test run, wiring matrix, and contract compliance were merged into the verification gate.)

{{include: skills/quality/SKILL.md}}

READ:
(1) Feature spec at {{.file_path}} for acceptance criteria and scope
(2) Feature test-plan.md for all test cases and expected coverage
(3) Code review report from docs/review/{epic-folder}/{feature-folder}/code_review/*-{{.id}}-review.md
  (derive path from {{.file_path}}: replace "docs/plan/" → "docs/review/", keep epic/feature folders) for scope of changes
(4) All task specs: `{{template "list_json" .}}` → collect every task's Scope file list
(5) `git diff $(git merge-base HEAD main)..HEAD` to derive all touched paths
(6) Parent interaction map and spec.md "Cross-feature interactions" section if present
(7) Parent {{.epic_id}}-cross-epic-map.md,
    docs/product/cross-epic-integration-map.md, and spec.md
    "Cross-epic integrations" section if present

SCOPED TEST RUN (single pass across the full feature — do NOT run per-task):

1. **Quality gates (always run, fast):** run the project's format + lint as documented in `docs/architecture/tech-stack.md` (**Quality Gate** section) or `docs/architecture/coding-standards.md`; otherwise infer from the repo (Makefile targets, `go vet ./...`, `package.json` scripts, configured Python tooling).

2. **Targeted unit tests:**
   - Union all changed source files from git diff AND from all task Scope sections
   - Map each to its test counterpart(s) using the project's test layout (see `docs/architecture/file-system.md` / `tech-stack.md`)
   - Add any tests explicitly named in the feature test-plan ACs
   - Run only those with the project's test runner

3. **Targeted integration tests (only if changeset crosses an integration seam):**
   - DB migration / schema change → corresponding integration tests and tests using the changed table
   - API endpoint change → integration tests for that route
   - Service contract change → tests for the consumer
   - Cross-feature I-## contract → run the shared contract test named in the feature spec and test-plan pointer
   - Cross-epic X-## integration → run the contract, journey handoff, or
     integration test named in the feature spec, product map, and test-plan
     pointer unless explicitly deferred
   - Pure intra-service or doc-only changes → skip

4. **Sanity spot-check (only if changes are broad across a shared module):**
   - Run the project's unit suite using the runner documented in `docs/architecture/tech-stack.md` or inferred from the repo

DO NOT RUN: full integration suite / codex red-team (UAT owns red-team)

VALIDATE:
- Quality gates pass (fmt, lint)
- All targeted tests pass — zero failures introduced by this feature
- Every AC from the feature test-plan is exercised by a named test
- Caller-Path Contract compliance: every contracted TC's committed test calls the declared entrypoint with the production argument shape, mocks no higher than the declared seam (violations are BLOCKERs naming the TC-ID)
- No regressions in targeted scope
- Pre-existing failures in scope are explicitly identified (not silently ignored)
- Wiring coverage matrix includes one row per CONTRACT-### and I-## with
  producer/consumer, contract test, test-exists, and test-passes columns
- Wiring coverage matrix includes one row per X-## this feature produces,
  consumes, or validates with producer/consumer feature refs, contract / shape
  source, coverage pointer or deferral, test-exists, and test-passes columns
- Any CONTRACT-###, I-##, or X-## row with a missing contract test, a test that
  does not assert the documented shape, a mismatched shape source, missing
  coverage disposition, or a failing test is an automatic FAIL

PRODUCE QA report to docs/review/{epic-folder}/{feature-folder}/qa/<timestamp>-{{.id}}-qa.md:
- Verdict: PASS or FAIL
- Test scope rationale: touched files from git diff and why each test path was chosen
- Test results summary (counts, durations)
- AC verification: name the test that proves each AC
- Wiring coverage matrix: CONTRACT-### and I-## rows with producer/consumer,
  contract test, test-exists, and test-passes columns; include X-## rows with
  producer/consumer, contract / shape source, contract test or deferral,
  test-exists, and test-passes columns
- Edge cases tested
- Any pre-existing failures encountered

ON PASS → {{template "advance" .}}
ON FAIL:
- For each failing task, kick back: `shark status set <task-id> development --reason "<specific failures>"`
- For a missing or broken I-## contract test: reopen the producer task in this
  feature and add a blocker note to each consuming feature:
  `shark feature note add <consumer_feature_id> --type blocker "Cross-feature I-## contract test failing at producer feature {{.id}}"`
- For a missing or broken X-## contract test or handoff test: reopen the
  producer or validating task in this feature, add blocker notes to consuming
  features, and append the failed X-## coverage status to docs/product/progress.md
- Set feature back: `{{template "status_set" .}} active --reason "QA failed — see report, tasks kicked back"`
