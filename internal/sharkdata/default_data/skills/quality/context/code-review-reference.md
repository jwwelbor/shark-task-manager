# Code Review Reference

Use this file when `workflows/review-code.md` needs the fuller examples for idiom checks, tooling, or report layout.

## Idiomatic language review checklist

| Language | What to check |
|---|---|
| Python | Pythonic patterns, type hints, no string-formatted SQL, prefer `pathlib`, idiomatic exception handling |
| TypeScript | Strict types, no `any` without justification, prefer `unknown` for untrusted input, discriminated unions, proper Promise typing |
| Go | Error handling, goroutine lifecycle, idiomatic struct embedding, `context.Context` propagation |
| All | Avoid nested ternaries, premature abstraction, commented-out code, and debug prints |

## Complexity tooling examples

```bash
# Python
uv run radon cc <files> --min=B

# TypeScript
npx eslint <files> --rule 'complexity: ["error", 10]'

# Go
gocyclo -over 10 <files>
```

## Report skeleton

The report written to `code_review_report_path` should contain:

1. Executive summary
2. Findings table
3. Reuse opportunities
4. Standards crosswalk
5. Tests review
6. Quality rubric
7. Risk hotspots
8. Codex red-team
9. Production caller chains
10. Triage summary
11. Verdict
