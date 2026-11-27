{{define "_tdd_process"}}TDD PROCESS:
Choose verification by artifact type. This table decides which verification applies and overrides any red-first wording in the included skills; those skills supply the red-green procedure for behavior-changing code. A change that mixes types applies each row to its own part.

| Change type | Verification |
|---|---|
| Behavior-changing code | Red first: the failing test is observed before implementation |
| Refactor with no behavior change | Existing tests are green before and after; no new red step is required |
| Prompt, documentation or configuration | The test plan's planned check for the TC (render, include resolution, validator); a source-text search only when the literal is an explicit contract |

When the work item has a test plan, read the feature test plan and then:
- Write one test per TC, with variants as separate cases. Each test name contains the TC-ID.
- Assert the TC's **Expected Output** exactly.
- Keep the TC's Caller-Path Contract rules: drive the declared entrypoint with the production argument shape, mock no higher than the lowest mock seam, never touch a forbidden mock, and satisfy the counter-factual (the check fails against an empty or buggy implementation).
- Never edit a TC's Preconditions, Input or Expected Output; return `blocked` with the specific conflict instead.

Do not write a test that searches source text unless the searched literal is an explicit contract.

Follow the project's documented test architecture and mocking boundaries, and run the focused test with the project's test command:
1. For behavior-changing code, confirm the test fails for the right reason (the missing behavior, not an import error or fixture bug).
2. Implement the minimum code to pass (green). Do NOT add features beyond what the work item requires.
3. Refactor while keeping tests green.

Tests that mock above the declared entrypoint, or use argument shapes production never passes, will be rejected at code review.{{end}}
