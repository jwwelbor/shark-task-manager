{{define "_qa_process"}}{{include: skills/quality/SKILL.md}}

READ:
(1) Task spec at {{.file_path}} for acceptance criteria and **Scope file list**
(2) Feature test-plan.md for test cases mapped to this task
(3) Feature spec.md for feature-level intent
(4) Code review report from {{.review_base}}code-review-*-{{.id}}.md
(5) Implementation code and existing test results
(6) `git diff` of the task's changes to derive the actual touched paths

SCOPED TEST RUN (single pass; do NOT run the full suite):

The full integration suite may have pre-existing failures unrelated to any single task. Running it adds significant wall time and produces noise that obscures real regressions. Instead:

1. **Quality gates (always run, fast):** run the project's format + lint as documented in `docs/architecture/tech-stack.md` (**Quality Gate** section) or `docs/architecture/coding-standards.md`; otherwise infer from the repo (Makefile targets, `go vet ./...`, `package.json` scripts, configured Python tooling).

2. **Targeted unit tests (always run):**
   - Identify changed source files from `git diff` AND from the task's Scope section
   - Map each changed source file to its test counterparts using the project's test layout (see `docs/architecture/file-system.md` / `tech-stack.md`)
   - Add any tests explicitly named in the task's Test Cases / Acceptance Criteria
   - Run only those with the project's test runner

3. **Targeted integration tests (run only if the change crosses an integration seam):**
   - DB migration / schema change → corresponding integration tests and any test using the changed table
   - API endpoint change → integration tests for that route
   - Service contract change consumed by another service → tests for the consumer
   - If the change is purely intra-service or doc/comment-only, **skip integration tests entirely**

4. **Sanity unit-suite spot-check (only if changes are broad):**
   - For changes touching a shared module / interface used by many consumers, run the project's full unit suite once (using the runner documented in `docs/architecture/tech-stack.md` or inferred from the repo)
   - For localized changes (single service / single API route / doc-only), skip this — the targeted run above is sufficient.

VALIDATE:
- Quality gates pass (fmt, lint)
- All targeted tests pass (zero failures introduced by this task)
- Acceptance criteria from the feature test plan are exercised by the tests you ran (cite test names)
- Edge cases identified in the test plan are covered
- No regressions in the targeted scope

DO NOT RUN:
- Full integration suite (too slow for per-task QA; may have pre-existing failures)
- Codex red-team (UAT owns red-team; running it here is redundant)

PRODUCE QA report at {{.review_base}}qa-<timestamp>-{{.id}}.md:
- Verdict: PASS or FAIL
- Test scope rationale: which paths you ran and why (cite touched files from `git diff`)
- Test results summary (counts, durations)
- AC verification status — name the test that proves each AC
- Edge cases tested
- Any pre-existing failures encountered in the targeted scope (explicitly identified, not silently ignored)

DECISION:
- ALL PASS → {{template "advance" .}}
- ANY FAIL → shark status set {{.id}} development --reason "<specific failures to fix>"
{{end}}
