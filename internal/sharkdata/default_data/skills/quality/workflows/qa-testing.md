---
inputs:
  - task_id: opaque task identifier (string)
  - task_spec_path: absolute path to the task spec markdown
  - feature_prd_path: absolute path to the feature PRD markdown
  - api_spec_path: absolute path to the API specification markdown (optional)
  - test_plan_path: absolute path to a pre-authored QA test plan, if one exists (optional)
  - interaction_map_path: absolute path to parent `<epic-id>-interaction-map.md` if present
  - design_refs: list of paths to wireframes / mockups / design files (optional, may be empty)
  - impl_paths: list of paths to changed source files
  - test_paths: list of paths to test files for the changes
  - acceptance_criteria: list of acceptance criteria text (extracted from spec)
  - has_frontend: bool — task involves frontend code (components, pages, styles, templates)
  - dev_server_command: string — how to start the project dev server (optional, only if has_frontend=true)
  - codex_command: string — pre-rendered codex exec command for red-team verification
  - qa_report_path: absolute path where the structured QA results report should be written
  - qa_exploratory_path: absolute path where exploratory findings should be written
outputs:
  - verdict: PASS | FAIL
  - qa_report: structured markdown written to qa_report_path
  - exploratory_findings: structured markdown written to qa_exploratory_path
  - bugs: list of {severity, summary, reproduction, expected, actual, fix_location} (empty on PASS)
  - blockers: list of blocker findings from any verification step
  - wiring_coverage: list of {contract_id, producer, consumers, contract_test, test_exists, test_passes}
---

# Workflow: QA Testing (craft)

## Purpose

Perform quality assurance testing for an implemented change. Validate code against acceptance criteria, run automated tests, perform frontend visual verification, run independent codex red-team, and produce a structured report with a clear PASS/FAIL verdict.

## Process

### Step 1: Read Testing Context

Open and read the inputs provided:

- `task_spec_path` — what was supposed to be built
- `feature_prd_path` — feature requirements and intent
- `api_spec_path` (if provided) — single source of truth for API contracts
- `impl_paths` — what was actually built (review the diff or final source)
- `test_paths` — what's being tested
- `interaction_map_path` and any `Cross-feature interactions` section, if
  present

You should leave this step with a clear mental model of: (a) what the change is supposed to do, (b) how it's supposed to be exercised, (c) what would constitute a successful behavior, (d) what edge cases the AC implies.

### Step 2: Load Pre-Existing Test Plan (Shift-Left)

If `test_plan_path` is provided:

- This is your primary validation reference — a QA agent already reviewed the spec against the feature PRD and wrote concrete acceptance test cases.
- Use the test plan's `TC-NNN` cases as your test scenarios.
- Verify each case passes as specified (inputs, expected outputs, edge cases).
- Add exploratory testing beyond the test plan, but the test plan scenarios are mandatory.

If no test plan exists:

- Construct test scenarios from scratch based on the acceptance criteria and feature PRD.
- Note this gap in the QA report — entering development without test planning is a workflow weak signal.

### Step 3: Run Automated Tests

First, consult `docs/architecture/tech-stack.md` (the **Quality Gate** section) or `docs/architecture/coding-standards.md` to determine the project's format, lint, and test commands. If neither document exists, infer from the repo: a `Makefile` → its documented format/lint/test targets; `go.mod` → `gofmt`/`go vet ./...`/`go test ./...`; `package.json` → its format/lint/test scripts; `pyproject.toml` → the configured formatter/linter and `pytest`.

Run the project's test suite scoped to this change. Use the appropriate runner determined above — choose by reading the project's docs, not by guessing.

Check test output for:

- All tests pass.
- No skipped tests that should run.
- Real assertions on positive outcomes, not just "doesn't crash."
- Edge cases covered.

If tests fail, capture the failure details verbatim — they go into the QA report.

### Step 4: Frontend Visual Verification (mandatory if `has_frontend=true`)

**If the task includes ANY frontend code, you CANNOT approve QA without loading the page in a browser and visually confirming it works.** "Tests pass" is not sufficient.

1. **Detect frontend changes** — components, pages, styles, templates count. If none, skip to Step 5.
2. **Start the dev server** — run `dev_server_command`.
3. **Load the page in a browser** — use browser automation (Chrome MCP tools, Playwright MCP, or equivalent):
   - Navigate to ALL pages affected by the change.
   - Verify pages load without console errors.
   - Verify the UI renders correctly (no blank screens, broken layouts, missing elements).
   - Test the golden-path interaction (click buttons, fill forms, navigate flows end-to-end).
   - Check for visual regressions in adjacent UI areas.
4. **Compare against design references** — if `design_refs` is non-empty, load each reference and compare against the rendered page. The implementation MUST match in layout, spacing, colors, typography, and interaction patterns. Deviations are BLOCKERS unless the task spec explicitly marks them as acceptable.
   - If `design_refs` is empty, note "No design references found" in the report. Visual verification of *functionality* still required.
5. **Document visual verification results** — screenshots or descriptions of what was observed, design reference comparison (match/mismatch with specifics), console errors observed, interaction test results.

If you cannot load the page (server won't start, browser tools unavailable), QA MUST FAIL. Do not approve frontend code you haven't seen rendered.

### Step 5: General Manual Testing

Even with passing automated tests, verify:

1. **UI/UX quality** — forms render, error messages are user-friendly, loading and success states work.
2. **Browser compatibility** — Chrome, Firefox, Safari (where applicable).
3. **Edge cases not in tests** — long inputs, special characters, network failures mid-flow.
4. **Performance** — page load, form submission lag, SLA compliance.

### Step 6: E2E Reachability Verification (mandatory)

Before verifying acceptance criteria, confirm the feature is actually wired into the runtime. This step prevents the #1 false-positive: all unit tests pass but the feature has no call sites and cannot be reached from the application entry point.

1. **Identify the public surface** — what new functions, classes, services, endpoints, or CLI commands does this change introduce?
2. **Search for call sites** — for each public component, search the codebase (excluding tests) for actual invocations.
3. **Trace to entry point** — from each call site, trace upward to verify it connects to a runtime entry point (`main()`, app factory, router, CLI handler).
4. **Check registration** — if the codebase uses DI, registries, or plugin systems, verify new components are registered.
5. **Verify routes** — if endpoints are introduced, verify they're mounted in the router.

**If the feature has zero call sites outside of tests, QA MUST FAIL.** Unit tests passing with mocks is not evidence of E2E functionality. Add this to the report as a BLOCKER finding.

### Step 7: Production Caller Signature Match (mandatory)

Reachability (Step 6) verifies the code is wired in. **Signature match** verifies the tests actually exercise it the way production calls it. Common false-positive: tests pass kwargs the production caller never passes, so the test exercises a code path production cannot reach.

For each service / function / endpoint introduced or modified:

1. Find the production caller — the file:line where production code invokes the service. Reuse the call-site search from Step 6.
2. Note the exact argument shape production passes — required positional args, optional kwargs left at defaults, kwargs not passed at all.
3. Look at the test(s) for that service. Do they call with the same shape?

**At least one test per service-contract change must drive the production signature exactly.** If you cannot find one, the verdict is FAIL — add it as a BLOCKER finding citing the production caller's `file:line` and the test's `file:line` so the developer can see the gap.

### Step 8: Codex Red-Team Verification (mandatory)

Run the provided `codex_command` to independently verify the implementation against the spec. This catches gaps that the primary QA pass may miss due to shared blind spots.

**Timeout**: 5 minutes. If it times out, retry once. After two failures, log `"Codex QA red-team: FAILED — [error]"` as a non-blocking note (do not skip QA because of codex failure, but document it).

**Include codex findings in the QA report.** If codex finds blocking issues that your own testing missed, those are BLOCKERS.

The codex prompt is constructed by the caller; the methodology codex applies — enumerate every violation per attack/error class, group findings, never iterate one-finding-at-a-time — is preserved by passing a well-structured `codex_command`. Don't let codex degrade into iterative back-and-forth; the report should be enumerative.

### Step 9: Integration Test Assertion Quality

When reviewing integration test results, verify tests assert positive outcomes — not just absence of exceptions:

- If a service has graceful degradation, check that tests distinguish between "succeeded normally" and "failed gracefully." A test that returns `200 OK` with empty/default data because the real pipeline errored silently is a **false positive**.
- Look for assertions on non-empty results (e.g., `assert result.summary != ""`, not just `assert result is not None`).
- If ALL integration tests pass but the service has a known graceful degradation path, explicitly verify that at least one test exercises the **happy path** — real data flowing through the full pipeline, not the fallback.

**If integration tests only exercise the degradation/fallback path, QA MUST FAIL.** Passing via fallback is not evidence that the real pipeline works.

### Step 10: Verify Against Acceptance Criteria

Walk each item in `acceptance_criteria`. For each:

- Mark `✅ PASS` or `❌ FAIL`.
- Where PASS: cite the test or manual verification that confirmed it.
- Where FAIL: capture expected vs actual, severity, and file:line where the fix is needed.

### Step 10.5: Wiring Coverage Matrix

For STANDARD/COMPLEX features, extend the QA report with a wiring coverage
matrix:

| Contract | Producer | Consumer(s) | Contract test | Test exists | Test passes |
|----------|----------|-------------|---------------|-------------|-------------|
| CONTRACT-001 | task | task(s) | test pointer | yes/no | yes/no |
| I-01 | feature/task | feature(s) | shared test pointer | yes/no | yes/no |

Rules:

- Include one row per CONTRACT-### from task specs.
- Include one row per I-## from the epic interaction map that this feature
  produces or consumes.
- Any CONTRACT-### or I-## row with a missing contract test, a test that does
  not assert the documented shape, or a failing test is an automatic FAIL.
- For a missing or broken I-## contract test, reopen the producer task and add a
  blocker note to the consuming feature.

### Step 11: Document Findings

Write a structured QA report to `qa_report_path`. Format:

```markdown
# QA Test Results: <task_id>

**Task:** <title>
**Tested by:** QA Agent
**Date:** <ISO timestamp>
**Result:** PASS | FAIL

## Test Summary
- Automated tests: <X>/<Y> passed
- Manual testing: <summary>
- Acceptance criteria: <X>/<Y> met
- Performance: <observed vs target>

## Acceptance Criteria Verification
- [<x|✅|❌>] <AC text> — <status with citation>

## Wiring Coverage Matrix
<CONTRACT-### and I-## rows with producer/consumer, contract test, test-exists, test-passes>

## Test Results
<detailed results>

## Quality Assessment
- UI/UX quality: <observation>
- Error messages: <observation>
- Browser compatibility: <observation>

## Codex Red-Team Findings
<verbatim codex output, grouped by category>

## Bugs (if any)

### BUG-NNN: <title>
**Severity:** <critical | high | medium | low>
**Impact:** <description>
**Reproduction:**
1. <step>
2. <step>

**Expected:** <expected behavior>
**Actual:** <actual behavior>
**Fix needed in:** <file:line>
```

Write exploratory findings (additional observations beyond the AC pass/fail) to `qa_exploratory_path`, even if none — record explicitly that no further issues were found.

### Step 12: Decide Verdict

#### PROHIBITED: Conditional Pass

**"Conditional pass" is not a valid QA verdict.** If any test fails, errors, or cannot run, the verdict is FAIL. QA must either:

1. Fix the blocking issue (if it's a test-only bug like a fixture defect), re-run ALL tests, and give a clean PASS/FAIL verdict, OR
2. FAIL the task and document the defects with enough detail for the developer to fix.

Never approve on the assumption that someone else will fix a prerequisite. If you cannot verify all tests pass in your session, the verdict is FAIL.

**Re-run mandate**: If you found and fixed a test bug during QA (e.g., fixture defect, missing import), you MUST re-run the entire test suite after the fix. A partial run where some tests errored at setup means you haven't validated the code paths those tests cover.

#### Verdict rules

- All tests pass + all AC met + reachability verified + signature match verified + codex returns PASS → **PASS**.
- Any blocker (failing test, unmet AC, missing call sites, signature mismatch, codex blocker, design mismatch) → **FAIL**.
- Missing or failing wiring coverage for CONTRACT-### or I-## rows → **FAIL**.
- Minor issues only (typos, minor UX improvements, exploratory observations) → **PASS** with notes.

### Step 13: Return Summary

Return a clear, structured summary with:

- Verdict (PASS | FAIL).
- Test counts and AC verification counts.
- Performance observations.
- Path to QA report.
- For FAIL: bullet list of bugs with severity, reproduction summary, fix location.

## Common Issues

### Tests Pass Locally But Fail in CI

Check: env vars in CI, database state cleanup between tests, timing assumptions (CI is usually slower than local). Note the discrepancy in the report.

### No Tests Exist

If no automated tests cover the change:

- Manual testing only is insufficient — record this as a BLOCKER unless the task spec explicitly waives it.
- Recommend tests be added before approving.

### Tests Are Insufficient

If tests cover only the happy path and miss edge cases declared in the AC:

- Note the gap in the report.
- If manual testing covered the missing scenarios successfully, this is a PASS-with-notes; if not, FAIL.

### Missing Test Credentials

If integration tests need credentials and they're not present in the project's documented credential store:

- Stop. Don't guess or skip tests.
- Note as BLOCKER.

## Quality Checklist

- [ ] Pre-existing test plan loaded (if available)
- [ ] All test plan scenarios validated (if test plan exists)
- [ ] Frontend visual verification performed — page loaded in browser, UI confirmed working (if `has_frontend`)
- [ ] Design reference comparison completed — implementation matches proposed designs (if `design_refs` non-empty)
- [ ] E2E reachability verified — all new components have call sites and connect to entry point
- [ ] Production caller signature match — at least one test per service-contract change drives the production argument shape
- [ ] Codex red-team verification run and findings incorporated
- [ ] All acceptance criteria verified
- [ ] Wiring coverage matrix completed for CONTRACT-### and I-## rows
- [ ] Every I-## contract test exists, asserts the documented shape, and passes
- [ ] Automated tests run (or noted if missing)
- [ ] Manual testing completed
- [ ] Edge cases tested
- [ ] Error messages are user-friendly
- [ ] Performance is acceptable
- [ ] Browser compatibility checked (where applicable)
- [ ] Bugs documented with reproduction steps
- [ ] Decision documented (PASS/FAIL with reasoning)

## Remember

Your role is **quality gatekeeper**:

- Don't approve buggy code to be nice.
- Don't skip tests because they're slow.
- Don't assume tests cover everything.
- Do provide clear reproduction steps for bugs.
- Do document why you pass or fail.
