---
name: test-driven-development
display_name: Test-Driven Development (TDD)
description: Write the test first, watch it fail, write minimal code to pass — when implementing any feature, bugfix, or behavior change, before writing production code
inputs:
  - change_description: string — what feature, bugfix, refactor, or behavior change is being implemented
  - target_paths: list of paths — source files where production code will be added or modified
  - test_paths: list of paths — test files where new tests will be added (may be created)
  - test_runner_command: string — project's test runner command scoped to a single test file or target
  - test_plan_path: absolute path to a pre-authored QA test plan with TC-NNN cases (optional — when present, test scenarios come from this plan)
  - feature_requirements_path: absolute path to a feature-level requirements/PRD document (optional — provides intent behind test plan cases)
  - acceptance_criteria: list of acceptance criteria text (optional — used as test scenario source when no test plan exists)
  - has_repository_layer: bool — whether the codebase has a repository/data-access layer documented in its test architecture
outputs:
  - tests_added: list of {path, test_name, behavior_under_test} — every new test written test-first
  - red_evidence: list of {test_name, observed_failure_message} — proof each test was watched failing before implementation
  - production_code_changes: list of paths — source files modified to make tests pass
  - green_evidence: list of {test_name, status} — proof every test passes after implementation, with no regressions
  - refactor_notes: list of strings — any post-green cleanup performed (deduplication, renames, helper extraction)
  - skipped_with_reason: list of {item, reason} — anything intentionally not test-driven, citing an authorized disposition
---

# Test-Driven Development (TDD)

## Overview & When to Use

Write the test first, watch it fail for the right reason, then write minimal code to pass. If you did not watch the test fail, you do not know whether it tests the right behavior.

- **Always apply to:** new features, bug fixes, behavior changes, and refactors.
- **Exceptions (only with an explicit authorized disposition):** throwaway prototypes, generated code, or non-executable configuration/documentation.
- **Unclear requirements or testability:** do not guess or skip TDD. Return `blocked` with the specific question about the missing requirement, contract, or testability constraint.

## The Iron Law & Scope

```
NO NEW PRODUCTION CODE WITHOUT A FAILING TEST FIRST
```

- **New code written in this session:** If you wrote new production code before a failing test existed, delete that new code and start over. Do not keep it as "reference" or adapt it while writing tests.
- **Scope — new code only:** Existing or resumed code already in the working tree or branch is **not** deleted. Preserve it and prove it against tests that fail without it (revert-and-run, or a mutation).

## Red-Green-Refactor Cycle

### 1. RED — Write Failing Test
- Write one minimal test for one behavior with a clear name that states the desired outcome.
- Exercise real code paths; introduce mocks only where required by the test plan or documented project boundaries.

### 2. Verify RED (Mandatory)
- Run the project's documented `test_runner_command` scoped to the test file or target.
- Confirm the test **fails** (not errors on syntax, imports, or setup) because the expected behavior is missing.
- If the test passes immediately, fix the test—it is testing existing behavior or the wrong code path.

### 3. GREEN — Minimal Code
- Write the simplest production code that makes the failing test pass.
- Do not add unrequested features, speculative abstractions, or unrelated refactors.

### 4. Verify GREEN (Mandatory)
- Re-run the targeted test and the relevant suite using the project's documented test runner.
- Confirm the new test passes, existing tests remain green, and output is free of errors or warnings.
- If any test fails, fix the production code rather than weakening the test.

### 5. REFACTOR — Clean Up
- Only while tests are green: remove duplication, clarify names, and extract helpers without adding new behavior.
- Re-run tests to confirm they stay green, then repeat from RED for the next behavior.

## Test Plan-Driven TDD & Caller-Path Contract Compliance

When a QA-authored test plan (`test_plan_path` or `docs/plan/$EPIC_ID/$FEATURE_ID/test_plans/`) exists, scenarios come from the test plan and parent feature PRD rather than your own interpretation:

1. **Read the test plan and parent PRD first** to ground concrete inputs, edge cases, negative cases, and expected outputs.
2. **Trace every `TC-NNN` case** to automated tests and include the `TC-ID` in the test name or a comment for mechanical traceability.
3. **Honor Caller-Path Contracts** (see `skills/quality/context/caller-path-contracts.md`) unless marked `internal — function under test is the production entrypoint`:
   - **Call the declared `Entrypoint`**, not a deeper internal helper or kernel function.
   - **Pass the declared `Production argument shape`** (same positional arguments and omitted optional parameters/kwargs).
   - **Mock no higher than the declared `Mock seam`**; mocking higher short-circuits the production wiring under test.
   - **Never mock `Forbidden seams`**; a test that mocks a forbidden seam is invalid even if it passes.
4. **When a contract or test plan is absent:** derive scenarios from acceptance criteria and test through the real caller entrypoint rather than bypassing wiring layers.

## Project-Neutral Test Boundaries & Runner Discovery

Follow the target project's documented test architecture, mocking boundaries, and test runner commands in `CLAUDE.md`, `AGENTS.md`, `docs/architecture/coding-standards.md`, `docs/architecture/test-gate.md`, or `docs/architecture/tech-stack.md` (or the repository's documented build/task targets).

- Do **not** invent ad-hoc mocking boundaries or assume database vs. mock layer rules not documented by the project.
- Do **not** guess test runner flags or fallback commands; read the project docs or build targets first.
- If testability is blocked or documented rules conflict, return `blocked` with the specific question rather than bypassing TDD.

## Red Flags — Stop and Correct

- Writing production code before a failing test, or keeping pre-written session code "as reference"
- Tests written after implementation that pass immediately without observed RED failure
- Skipping RED verification or proceeding when a test errors instead of failing on behavior
- Bypassing a declared `Entrypoint`, altering `Production argument shape`, mocking above `Mock seam`, or mocking `Forbidden seams`
- Inventing undocumented DB/mock layer rules or rationalizing skipping TDD "just this once"

For new code written in this session, any red flag means delete the unverified production code and restart from RED. For existing or resumed code, preserve it and prove coverage with tests that fail without it.

## Verification Checklist & Debugging Integration

Before marking work complete, confirm:
- [ ] Every new behavior or bug fix has a test written before production code (or existing/resumed code was proved via revert-and-run or mutation)
- [ ] Each test failed for the expected missing-behavior reason before implementation
- [ ] Minimal code was written to pass, all targeted and suite tests pass, and output is clean
- [ ] Every `TC-NNN` test (when a test plan exists) names its `TC-ID` and complies with its Caller-Path Contract
- [ ] Mocking boundaries and test runner commands match the project's documented conventions

**Debugging integration:** Never fix a bug without a failing regression test first. Follow the capture → reproduce → verify RED → minimal fix → verify GREEN → expand workflow in [references/debugging-workflow.md](references/debugging-workflow.md).

**Final rule:** No new production code without a failing test first—and no exceptions without an authorized disposition or returning `blocked`.
