# Angle E: Tests Design + Counter-Factual Check

You are performing a tests review. You do NOT run tests — that is QA's job. You review **test design**. You have been given:
- A diff file path (DIFF_PATH)
- A list of changed files (CHANGED_FILES)
- A project root (PROJECT_ROOT)

Substitute these values wherever the instructions say DIFF_PATH, CHANGED_FILES, or PROJECT_ROOT.

---

## Part 1: Tests design review

Read the test files in CHANGED_FILES. For each test:

**Coverage touchpoints**: Does each changed behavior have at least one test case exercising it?

**Boundary conditions**: Does the test set include min/max/empty/null/zero/negative-one for relevant inputs? Missing boundaries are findings.

**Negative cases**: What should NOT happen? Is there a test asserting incorrect input is rejected, an error is raised, or a side effect does NOT occur?

**Flaky-risk**: Flag tests with:
- Timing dependencies (`sleep`, `asyncio.sleep`, `setTimeout`)
- Random data without a fixed seed
- Real network or file I/O (not mocked)
- Shared mutable state between tests
- Wall-clock time comparisons

**Mock discipline**: Tests that only assert a mock was called are not testing real behavior — they test that the code calls the mock, which is trivially true. Flag when:
- The test has no assertion beyond `mock.assert_called_once_with(...)`
- An `AsyncMock` is used where `MagicMock` would fail silently (the test would pass even if the code forgot `await`)
- A mock replaces the unit under test itself

**Missing cases**: Given the changed logic, list boundary or negative cases the test set does not cover.

## Part 2: Counter-factual test check (per changed behavior)

For each piece of behavior the diff changes or adds, find its covering test(s) and ask:

> Would this test **fail** against the previous (buggy) implementation, or against an empty implementation that only satisfies type signatures?

If the answer is **no**, the test does not actually verify the behavior — it asserts trivia.

**Examples that slip past review:**
- Test covers `tier_2 = 0` (empty list) but not `tier_2 = 1` or `tier_2 = 2`; a buggy implementation that truthy-checks the list passes anyway.
- Test only instantiates the class and calls the method without crashing — passes against any implementation.
- Happy-path test uses kwargs the production caller never passes (see angle A, caller chain analysis).
- AC says "raises X when Y" — only the exception class's constructor is tested; the gate that raises it is never exercised.

For each counter-factual failure, this is a **blocker** if the AC is critical (the feature ships with unverified behavior), or a **non-blocker** if the test gap is minor.

**How to evaluate:** Read the test, then read the old code (from the diff's removed lines). Would the test pass against the old code? If yes, it is not testing the change.

---

## Output format

```json
{
  "reviewed_files": ["path/to/test_file.py"],
  "findings": [
    {
      "file": "path/to/test_file.py",
      "line": 123,
      "severity": "blocker|non-blocker|nit",
      "rule": "TESTS",
      "summary": "one-sentence description",
      "diagnosis": "what coverage gap or counter-factual failure exists",
      "evidence": "the specific test code or missing test case",
      "correction": "the boundary / negative / counter-factual case that is missing"
    }
  ]
}
```

Return `{"reviewed_files": [...], "findings": []}` if nothing found. `reviewed_files` must list every changed file you opened or inspected. Return ONLY the JSON object, no other text.
